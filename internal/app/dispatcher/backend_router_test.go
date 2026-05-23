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

func TestBackendRouterResolveMCPServersFromAgentDefinition(t *testing.T) {
	t.Parallel()
	registry := make(config.AgentsRegistry)
	registry["go"] = config.GroupConfig{
		Default: config.Preset{Client: "claude"},
		Kinds:   make(map[domain.Kind]config.Override),
	}
	router := NewBackendRouter(&registry, ResolvedTemplate{Client: "claude"})

	def := &AgentDefinition{
		Name: "ta-go-builder",
		MCPServers: map[string]AgentDefinitionMCPServer{
			"tillsyn-dev": {
				Command: "till",
				Args:    []string{"mcp"},
				Tools:   []string{"till.action_item", "till.comment"},
			},
		},
	}

	out := router.ResolveMCPServers(def)
	if len(out) != 1 {
		t.Fatalf("want 1 server, got %d", len(out))
	}

	srv, ok := out["tillsyn-dev"]
	if !ok {
		t.Fatal("missing tillsyn-dev server")
	}
	if srv.Command != "till" {
		t.Errorf("Command = %q; want %q", srv.Command, "till")
	}
	if len(srv.Args) != 1 || srv.Args[0] != "mcp" {
		t.Errorf("Args = %v; want [mcp]", srv.Args)
	}
	if len(srv.Tools) != 2 {
		t.Errorf("Tools = %v; want 2 tools", srv.Tools)
	}
}

func TestBackendRouterResolveMCPServersNilDefYieldsNil(t *testing.T) {
	t.Parallel()
	registry := make(config.AgentsRegistry)
	registry["go"] = config.GroupConfig{
		Default: config.Preset{Client: "claude"},
		Kinds:   make(map[domain.Kind]config.Override),
	}
	router := NewBackendRouter(&registry, ResolvedTemplate{Client: "claude"})

	out := router.ResolveMCPServers(nil)
	if out != nil {
		t.Fatalf("want nil, got %v", out)
	}
}

func TestBackendRouterResolveMCPServersEmptyMCPServersYieldsNil(t *testing.T) {
	t.Parallel()
	registry := make(config.AgentsRegistry)
	registry["go"] = config.GroupConfig{
		Default: config.Preset{Client: "claude"},
		Kinds:   make(map[domain.Kind]config.Override),
	}
	router := NewBackendRouter(&registry, ResolvedTemplate{Client: "claude"})

	def := &AgentDefinition{
		Name:       "ta-go-builder",
		MCPServers: make(map[string]AgentDefinitionMCPServer),
	}

	out := router.ResolveMCPServers(def)
	if out != nil {
		t.Fatalf("want nil for empty MCPServers, got %v", out)
	}
}

func TestBackendRouterResolveMCPServersDefensiveCopy(t *testing.T) {
	t.Parallel()
	registry := make(config.AgentsRegistry)
	registry["go"] = config.GroupConfig{
		Default: config.Preset{Client: "claude"},
		Kinds:   make(map[domain.Kind]config.Override),
	}
	router := NewBackendRouter(&registry, ResolvedTemplate{Client: "claude"})

	originalArgs := []string{"mcp"}
	originalTools := []string{"till.action_item"}
	def := &AgentDefinition{
		Name: "ta-go-builder",
		MCPServers: map[string]AgentDefinitionMCPServer{
			"tillsyn-dev": {
				Command: "till",
				Args:    originalArgs,
				Tools:   originalTools,
			},
		},
	}

	out := router.ResolveMCPServers(def)
	srv := out["tillsyn-dev"]

	// Mutate the returned slices and verify the original is unchanged.
	srv.Args[0] = "modified"
	srv.Tools[0] = "modified"

	if def.MCPServers["tillsyn-dev"].Args[0] != "mcp" {
		t.Errorf("defensive copy failed: original Args mutated")
	}
	if def.MCPServers["tillsyn-dev"].Tools[0] != "till.action_item" {
		t.Errorf("defensive copy failed: original Tools mutated")
	}
}
