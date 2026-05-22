package dispatcher

import (
	"errors"
	"testing"

	"github.com/evanmschultz/tillsyn/internal/config"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

func TestBackendRouterResolveBackend(t *testing.T) {
	tests := []struct {
		name           string
		templateClient string
		presetClient   string
		wantClient     string
		wantErr        bool
		errSentinel    error
	}{
		{
			name:           "template only",
			templateClient: "claude",
			presetClient:   "",
			wantClient:     "claude",
			wantErr:        false,
		},
		{
			name:           "preset only",
			templateClient: "",
			presetClient:   "codex",
			wantClient:     "codex",
			wantErr:        false,
		},
		{
			name:           "both empty",
			templateClient: "",
			presetClient:   "",
			wantClient:     "",
			wantErr:        true,
			errSentinel:    ErrUnroutablePersona,
		},
		{
			name:           "both equal",
			templateClient: "claude",
			presetClient:   "claude",
			wantClient:     "claude",
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Construct a minimal registry with the test preset.
			registry := make(config.AgentsRegistry)
			registry["go"] = config.GroupConfig{
				Default: config.Preset{
					Client: tt.presetClient,
				},
				Kinds: make(map[domain.Kind]config.Override),
			}

			template := ResolvedTemplate{
				Client: tt.templateClient,
			}

			router := NewBackendRouter(&registry, template)
			gotClient, err := router.ResolveBackend("ta-go-builder", "go", "build")

			if tt.wantErr {
				if err == nil {
					t.Fatalf("ResolveBackend() wanted error, got nil")
				}
				if !errors.Is(err, tt.errSentinel) {
					t.Fatalf("ResolveBackend() error = %v, want %v", err, tt.errSentinel)
				}
				return
			}

			if err != nil {
				t.Fatalf("ResolveBackend() unexpected error: %v", err)
			}

			if gotClient != tt.wantClient {
				t.Fatalf("ResolveBackend() = %q, want %q", gotClient, tt.wantClient)
			}
		})
	}
}

func TestBackendRouterResolveBackendConflictingNonEmpty(t *testing.T) {
	// Test the defense-in-depth check: both non-empty but DIFFER.
	// The boot-time validator should prevent this, but we check anyway.
	registry := make(config.AgentsRegistry)
	registry["go"] = config.GroupConfig{
		Default: config.Preset{
			Client: "codex",
		},
		Kinds: make(map[domain.Kind]config.Override),
	}

	template := ResolvedTemplate{
		Client: "claude",
	}

	router := NewBackendRouter(&registry, template)
	_, err := router.ResolveBackend("ta-go-builder", "go", "build")

	if err == nil {
		t.Fatalf("ResolveBackend() wanted error for conflicting non-empty clients, got nil")
	}
	if !errors.Is(err, ErrUnroutablePersona) {
		t.Fatalf("ResolveBackend() error = %v, want ErrUnroutablePersona", err)
	}
}
