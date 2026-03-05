package fantasyembed

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"charm.land/fantasy"
	"charm.land/fantasy/providers/openai"
)

const (
	defaultProviderName = "openai"
	defaultModelName    = "text-embedding-3-small"
	defaultAPIKeyEnv    = "OPENAI_API_KEY"
)

// Config defines embedding provider settings for the fantasy adapter.
type Config struct {
	Provider   string
	Model      string
	APIKeyEnv  string
	BaseURL    string
	Dimensions int64
}

// Generator wraps a fantasy embedding model to satisfy app.EmbeddingGenerator.
type Generator struct {
	model      fantasy.EmbeddingModel
	dimensions *int64
}

// New creates one configured fantasy embedding generator.
func New(ctx context.Context, cfg Config) (*Generator, error) {
	providerName := strings.TrimSpace(strings.ToLower(cfg.Provider))
	if providerName == "" {
		providerName = defaultProviderName
	}
	modelName := strings.TrimSpace(cfg.Model)
	if modelName == "" {
		modelName = defaultModelName
	}
	apiKeyEnv := strings.TrimSpace(cfg.APIKeyEnv)
	if apiKeyEnv == "" {
		apiKeyEnv = defaultAPIKeyEnv
	}
	apiKey := strings.TrimSpace(os.Getenv(apiKeyEnv))
	if apiKey == "" {
		return nil, fmt.Errorf("embedding api key env %q is not set", apiKeyEnv)
	}

	var provider fantasy.Provider
	switch providerName {
	case defaultProviderName:
		opts := []openai.Option{openai.WithAPIKey(apiKey)}
		if baseURL := strings.TrimSpace(cfg.BaseURL); baseURL != "" {
			opts = append(opts, openai.WithBaseURL(baseURL))
		}
		p, err := openai.New(opts...)
		if err != nil {
			return nil, fmt.Errorf("create fantasy openai provider: %w", err)
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
	if g == nil || g.model == nil {
		return nil, errors.New("embedding generator is not initialized")
	}
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
