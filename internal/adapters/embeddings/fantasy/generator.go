package fantasyembed

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"os"
	"strings"
	"unicode"

	"charm.land/fantasy"
	"charm.land/fantasy/providers/openai"
	"charm.land/fantasy/providers/openaicompat"
)

const (
	defaultProviderName            = "ollama"
	openAIProviderName             = "openai"
	ollamaProviderName             = "ollama"
	deterministicProviderName      = "deterministic"
	defaultOpenAIModelName         = "text-embedding-3-small"
	defaultOllamaModelName         = "qwen3-embedding:8b"
	defaultDeterministicModelName  = "hash-bow-v1"
	defaultOllamaBaseURL           = "http://127.0.0.1:11434/v1"
	defaultDeterministicDimensions = int64(256)
	defaultAPIKeyEnv               = "OPENAI_API_KEY"
)

// Config defines embedding provider settings for the fantasy adapter.
type Config struct {
	Provider   string
	Model      string
	APIKeyEnv  string
	BaseURL    string
	Dimensions int64
}

// Generator wraps either a fantasy embedding model or the local deterministic provider.
type Generator struct {
	model                   fantasy.EmbeddingModel
	dimensions              *int64
	deterministicDimensions int
}

// New creates one configured embedding generator.
func New(ctx context.Context, cfg Config) (*Generator, error) {
	providerName := strings.TrimSpace(strings.ToLower(cfg.Provider))
	if providerName == "" {
		providerName = defaultProviderName
	}
	modelName := strings.TrimSpace(cfg.Model)
	if modelName == "" {
		switch providerName {
		case deterministicProviderName:
			modelName = defaultDeterministicModelName
		case openAIProviderName:
			modelName = defaultOpenAIModelName
		default:
			modelName = defaultOllamaModelName
		}
	}

	if providerName == deterministicProviderName {
		dimensions := cfg.Dimensions
		if dimensions <= 0 {
			dimensions = defaultDeterministicDimensions
		}
		return &Generator{deterministicDimensions: int(dimensions)}, nil
	}

	var provider fantasy.Provider
	switch providerName {
	case openAIProviderName:
		apiKeyEnv := strings.TrimSpace(cfg.APIKeyEnv)
		if apiKeyEnv == "" {
			apiKeyEnv = defaultAPIKeyEnv
		}
		apiKey := strings.TrimSpace(os.Getenv(apiKeyEnv))
		if apiKey == "" {
			return nil, fmt.Errorf("embedding api key env %q is not set", apiKeyEnv)
		}
		opts := []openai.Option{openai.WithAPIKey(apiKey)}
		if baseURL := strings.TrimSpace(cfg.BaseURL); baseURL != "" {
			opts = append(opts, openai.WithBaseURL(baseURL))
		}
		p, err := openai.New(opts...)
		if err != nil {
			return nil, fmt.Errorf("create fantasy openai provider: %w", err)
		}
		provider = p
	case ollamaProviderName:
		baseURL := strings.TrimSpace(cfg.BaseURL)
		if baseURL == "" {
			baseURL = defaultOllamaBaseURL
		}
		opts := []openaicompat.Option{openaicompat.WithBaseURL(baseURL)}
		if apiKeyEnv := strings.TrimSpace(cfg.APIKeyEnv); apiKeyEnv != "" {
			if configured := strings.TrimSpace(os.Getenv(apiKeyEnv)); configured != "" {
				opts = append(opts, openaicompat.WithAPIKey(configured))
			}
		}
		p, err := openaicompat.New(opts...)
		if err != nil {
			return nil, fmt.Errorf("create fantasy ollama-compatible provider: %w", err)
		}
		provider = p
	default:
		return nil, fmt.Errorf("unsupported embeddings.provider %q", providerName)
	}

	embeddingProvider, ok := provider.(fantasy.EmbeddingProvider)
	if !ok {
		return nil, errors.New("configured fantasy provider does not support embeddings")
	}
	model, err := embeddingProvider.EmbeddingModel(ctx, modelName)
	if err != nil {
		return nil, fmt.Errorf("create embedding model %q: %w", modelName, err)
	}

	var dimensions *int64
	if cfg.Dimensions > 0 {
		dim := cfg.Dimensions
		dimensions = &dim
	}
	return &Generator{
		model:      model,
		dimensions: dimensions,
	}, nil
}

// Embed returns embedding vectors for the given input strings.
func (g *Generator) Embed(ctx context.Context, inputs []string) ([][]float32, error) {
	if len(inputs) == 0 {
		return nil, errors.New("embedding inputs are required")
	}
	callInputs := make([]string, len(inputs))
	for idx, input := range inputs {
		trimmed := strings.TrimSpace(input)
		if trimmed == "" {
			return nil, fmt.Errorf("embedding input at index %d is empty", idx)
		}
		callInputs[idx] = trimmed
	}
	if g != nil && g.deterministicDimensions > 0 {
		out := make([][]float32, 0, len(callInputs))
		for _, input := range callInputs {
			out = append(out, deterministicEmbeddingVector(input, g.deterministicDimensions))
		}
		return out, nil
	}
	if g == nil || g.model == nil {
		return nil, errors.New("embedding generator is not initialized")
	}
	resp, err := g.model.Embed(ctx, fantasy.EmbeddingCall{
		Inputs:     callInputs,
		Dimensions: g.dimensions,
	})
	if err != nil {
		return nil, fmt.Errorf("embed inputs: %w", err)
	}
	out := make([][]float32, len(callInputs))
	for _, row := range resp.Embeddings {
		if row.Index < 0 || row.Index >= len(out) {
			continue
		}
		out[row.Index] = append([]float32(nil), row.Vector...)
	}
	for idx, row := range out {
		if len(row) == 0 {
			return nil, fmt.Errorf("missing embedding for input index %d", idx)
		}
	}
	return out, nil
}

func deterministicEmbeddingVector(input string, dimensions int) []float32 {
	if dimensions <= 0 {
		dimensions = int(defaultDeterministicDimensions)
	}
	vector := make([]float32, dimensions)
	tokens := deterministicEmbeddingTokens(input)
	if len(tokens) == 0 {
		tokens = []string{strings.TrimSpace(strings.ToLower(input))}
	}
	for _, token := range tokens {
		sum := deterministicTokenHash(token)
		if sum == 0 {
			continue
		}
		primary := int(sum % uint64(dimensions))
		sign := float32(1)
		if (sum>>7)&1 == 1 {
			sign = -1
		}
		vector[primary] += sign
		secondary := int((sum >> 17) % uint64(dimensions))
		if secondary != primary {
			vector[secondary] += 0.5 * sign
		}
	}
	return normalizeDeterministicVector(vector)
}

func deterministicEmbeddingTokens(input string) []string {
	lower := strings.ToLower(strings.TrimSpace(input))
	if lower == "" {
		return nil
	}
	fields := strings.FieldsFunc(lower, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	capacity := 0
	if len(fields) > 0 {
		capacity = (len(fields) * 2) - 1
	}
	out := make([]string, 0, capacity)
	for idx, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		out = append(out, field)
		if idx > 0 {
			out = append(out, fields[idx-1]+"_"+field)
		}
	}
	return out
}

func deterministicTokenHash(token string) uint64 {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(token))
	return hasher.Sum64()
}

func normalizeDeterministicVector(vector []float32) []float32 {
	var magnitude float64
	for _, value := range vector {
		magnitude += float64(value * value)
	}
	if magnitude == 0 {
		return vector
	}
	scale := float32(1 / math.Sqrt(magnitude))
	for idx := range vector {
		vector[idx] *= scale
	}
	return vector
}
