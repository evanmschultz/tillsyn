package fantasyembed

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"

	"charm.land/fantasy"
)

// TestNewDeterministicGeneratorUsesDefaults verifies the local deterministic provider initializes without network credentials.
func TestNewDeterministicGeneratorUsesDefaults(t *testing.T) {
	generator, err := New(context.Background(), Config{
		Provider: deterministicProviderName,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if generator == nil {
		t.Fatal("expected generator, got nil")
	}
	if generator.model != nil {
		t.Fatal("expected deterministic generator to avoid fantasy provider initialization")
	}
	if generator.deterministicDimensions != int(defaultDeterministicDimensions) {
		t.Fatalf("deterministic dimensions = %d, want %d", generator.deterministicDimensions, defaultDeterministicDimensions)
	}
}

// TestGeneratorEmbedDeterministicStableAndNormalized verifies the deterministic provider is repeatable and length-normalized.
func TestGeneratorEmbedDeterministicStableAndNormalized(t *testing.T) {
	generator, err := New(context.Background(), Config{
		Provider:   deterministicProviderName,
		Dimensions: 16,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	first, err := generator.Embed(context.Background(), []string{"Latency budget belongs in the thread context."})
	if err != nil {
		t.Fatalf("Embed(first) error = %v", err)
	}
	second, err := generator.Embed(context.Background(), []string{"Latency budget belongs in the thread context."})
	if err != nil {
		t.Fatalf("Embed(second) error = %v", err)
	}
	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("expected 1 vector per call, got %d and %d", len(first), len(second))
	}
	if len(first[0]) != 16 {
		t.Fatalf("vector length = %d, want 16", len(first[0]))
	}
	for idx := range first[0] {
		if first[0][idx] != second[0][idx] {
			t.Fatalf("vector component %d differs across calls: %f vs %f", idx, first[0][idx], second[0][idx])
		}
	}
	var magnitude float64
	for _, value := range first[0] {
		magnitude += float64(value * value)
	}
	if math.Abs(math.Sqrt(magnitude)-1) > 0.0001 {
		t.Fatalf("vector magnitude = %f, want ~1", math.Sqrt(magnitude))
	}

	other, err := generator.Embed(context.Background(), []string{"Completely different document terms live here."})
	if err != nil {
		t.Fatalf("Embed(other) error = %v", err)
	}
	if len(other) != 1 {
		t.Fatalf("expected 1 vector for other input, got %d", len(other))
	}
	if vectorsEqual(first[0], other[0]) {
		t.Fatal("expected different inputs to yield different deterministic vectors")
	}
}

// TestGeneratorEmbedRejectsEmptyInputs verifies provider calls remain fail-fast on empty request payloads.
func TestGeneratorEmbedRejectsEmptyInputs(t *testing.T) {
	generator, err := New(context.Background(), Config{
		Provider:   deterministicProviderName,
		Dimensions: 8,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	_, err = generator.Embed(context.Background(), []string{"valid", "   "})
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("Embed() error = %v, want empty-input guidance", err)
	}
}

// TestNewOpenAIGeneratorRequiresAPIKey verifies the OpenAI-backed provider still refuses to start without credentials.
func TestNewOpenAIGeneratorRequiresAPIKey(t *testing.T) {
	t.Setenv(defaultAPIKeyEnv, "")

	_, err := New(context.Background(), Config{
		Provider: openAIProviderName,
		Model:    defaultOpenAIModelName,
	})
	if err == nil || !strings.Contains(err.Error(), defaultAPIKeyEnv) {
		t.Fatalf("New() error = %v, want missing API key guidance", err)
	}
}

// TestNewOpenAIGeneratorConfiguresModel verifies the OpenAI-backed path builds a model wrapper without a network call.
func TestNewOpenAIGeneratorConfiguresModel(t *testing.T) {
	t.Setenv(defaultAPIKeyEnv, "test-key")

	generator, err := New(context.Background(), Config{
		Provider:   openAIProviderName,
		Model:      defaultOpenAIModelName,
		BaseURL:    "https://example.invalid/v1",
		Dimensions: 384,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if generator == nil || generator.model == nil {
		t.Fatalf("expected configured openai model, got %#v", generator)
	}
	if generator.deterministicDimensions != 0 {
		t.Fatalf("deterministic dimensions = %d, want 0", generator.deterministicDimensions)
	}
	if generator.dimensions == nil || *generator.dimensions != 384 {
		t.Fatalf("dimensions = %#v, want 384", generator.dimensions)
	}
}

// TestNewOllamaGeneratorConfiguresModel verifies the default Ollama path builds an embeddings model without requiring API credentials.
func TestNewOllamaGeneratorConfiguresModel(t *testing.T) {
	generator, err := New(context.Background(), Config{
		Provider: ollamaProviderName,
		Model:    defaultOllamaModelName,
		BaseURL:  "http://127.0.0.1:11434/v1",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if generator == nil || generator.model == nil {
		t.Fatalf("expected configured ollama model, got %#v", generator)
	}
	if generator.deterministicDimensions != 0 {
		t.Fatalf("deterministic dimensions = %d, want 0", generator.deterministicDimensions)
	}
	if generator.dimensions != nil {
		t.Fatalf("dimensions = %#v, want nil provider-default dimensions", generator.dimensions)
	}
}

// TestNewRejectsUnsupportedProvider verifies invalid provider names fail fast during config validation.
func TestNewRejectsUnsupportedProvider(t *testing.T) {
	_, err := New(context.Background(), Config{Provider: "bogus"})
	if err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("New() error = %v, want unsupported provider guidance", err)
	}
}

// TestGeneratorEmbedRejectsZeroInputs verifies the adapter refuses empty call batches before touching a provider.
func TestGeneratorEmbedRejectsZeroInputs(t *testing.T) {
	generator, err := New(context.Background(), Config{
		Provider: deterministicProviderName,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = generator.Embed(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatalf("Embed() error = %v, want missing-input guidance", err)
	}
}

// TestGeneratorEmbedRejectsUninitializedGenerator verifies the provider-backed path guards nil generators.
func TestGeneratorEmbedRejectsUninitializedGenerator(t *testing.T) {
	var generator *Generator

	_, err := generator.Embed(context.Background(), []string{"hello"})
	if err == nil || !strings.Contains(err.Error(), "not initialized") {
		t.Fatalf("Embed() error = %v, want initialization guidance", err)
	}
}

// TestGeneratorEmbedTrimsInputsAndPreservesOrder verifies provider-backed calls normalize inputs while keeping response ordering stable.
func TestGeneratorEmbedTrimsInputsAndPreservesOrder(t *testing.T) {
	model := &fakeEmbeddingModel{
		response: &fantasy.EmbeddingResponse{
			Embeddings: []fantasy.Embedding{
				{Index: 1, Vector: []float32{2, 0}},
				{Index: 0, Vector: []float32{1, 0}},
			},
		},
	}
	dimensions := int64(64)
	generator := &Generator{
		model:      model,
		dimensions: &dimensions,
	}

	got, err := generator.Embed(context.Background(), []string{" first  ", "second "})
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if len(model.lastCall.Inputs) != 2 || model.lastCall.Inputs[0] != "first" || model.lastCall.Inputs[1] != "second" {
		t.Fatalf("normalized inputs = %#v, want trimmed inputs", model.lastCall.Inputs)
	}
	if model.lastCall.Dimensions == nil || *model.lastCall.Dimensions != 64 {
		t.Fatalf("call dimensions = %#v, want 64", model.lastCall.Dimensions)
	}
	if len(got) != 2 || !vectorsEqual(got[0], []float32{1, 0}) || !vectorsEqual(got[1], []float32{2, 0}) {
		t.Fatalf("ordered vectors = %#v, want provider order restored by index", got)
	}
}

// TestGeneratorEmbedPropagatesProviderFailures verifies provider errors stay wrapped with embedding context.
func TestGeneratorEmbedPropagatesProviderFailures(t *testing.T) {
	generator := &Generator{
		model: &fakeEmbeddingModel{err: errors.New("provider boom")},
	}

	_, err := generator.Embed(context.Background(), []string{"hello"})
	if err == nil || !strings.Contains(err.Error(), "embed inputs") || !strings.Contains(err.Error(), "provider boom") {
		t.Fatalf("Embed() error = %v, want wrapped provider failure", err)
	}
}

// TestGeneratorEmbedRejectsMissingResponseRows verifies sparse provider responses fail fast instead of returning partial vectors.
func TestGeneratorEmbedRejectsMissingResponseRows(t *testing.T) {
	generator := &Generator{
		model: &fakeEmbeddingModel{
			response: &fantasy.EmbeddingResponse{
				Embeddings: []fantasy.Embedding{
					{Index: 1, Vector: []float32{2, 0}},
					{Index: 99, Vector: []float32{9, 9}},
				},
			},
		},
	}

	_, err := generator.Embed(context.Background(), []string{"first", "second"})
	if err == nil || !strings.Contains(err.Error(), "missing embedding for input index 0") {
		t.Fatalf("Embed() error = %v, want missing-row guidance", err)
	}
}

// TestDeterministicEmbeddingTokensBuildsBigrams verifies the tokenizer keeps stable token and bigram ordering.
func TestDeterministicEmbeddingTokensBuildsBigrams(t *testing.T) {
	got := deterministicEmbeddingTokens("Alpha, Beta! 42")
	want := []string{"alpha", "beta", "alpha_beta", "42", "beta_42"}
	if !stringSlicesEqual(got, want) {
		t.Fatalf("tokens = %#v, want %#v", got, want)
	}
}

// TestDeterministicEmbeddingVectorHandlesSymbolOnlyInput verifies fallback hashing works even when the tokenizer produces no fields.
func TestDeterministicEmbeddingVectorHandlesSymbolOnlyInput(t *testing.T) {
	vector := deterministicEmbeddingVector("!!!", 0)
	if len(vector) != int(defaultDeterministicDimensions) {
		t.Fatalf("vector length = %d, want %d", len(vector), defaultDeterministicDimensions)
	}
	var magnitude float64
	for _, value := range vector {
		magnitude += float64(value * value)
	}
	if math.Abs(math.Sqrt(magnitude)-1) > 0.0001 {
		t.Fatalf("vector magnitude = %f, want ~1", math.Sqrt(magnitude))
	}
}

// TestNormalizeDeterministicVectorLeavesZeroMagnitude verifies zero vectors stay stable instead of dividing by zero.
func TestNormalizeDeterministicVectorLeavesZeroMagnitude(t *testing.T) {
	vector := []float32{0, 0, 0}
	got := normalizeDeterministicVector(vector)
	if !vectorsEqual(got, []float32{0, 0, 0}) {
		t.Fatalf("normalized zero vector = %#v, want unchanged zero vector", got)
	}
}

// vectorsEqual reports whether two vectors contain identical components.
func vectorsEqual(left, right []float32) bool {
	if len(left) != len(right) {
		return false
	}
	for idx := range left {
		if left[idx] != right[idx] {
			return false
		}
	}
	return true
}

// stringSlicesEqual reports whether two string slices contain the same elements in order.
func stringSlicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for idx := range left {
		if left[idx] != right[idx] {
			return false
		}
	}
	return true
}

// fakeEmbeddingModel implements fantasy.EmbeddingModel for adapter tests.
type fakeEmbeddingModel struct {
	response *fantasy.EmbeddingResponse
	err      error
	lastCall fantasy.EmbeddingCall
}

// Embed records the last request and returns the configured response.
func (m *fakeEmbeddingModel) Embed(_ context.Context, call fantasy.EmbeddingCall) (*fantasy.EmbeddingResponse, error) {
	m.lastCall = call
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

// Provider returns one stable fake provider name.
func (m *fakeEmbeddingModel) Provider() string {
	return "fake"
}

// Model returns one stable fake model name.
func (m *fakeEmbeddingModel) Model() string {
	return "fake-model"
}
