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

func TestBackendRouterResolveEnvSetClonesPresetEnvSet(t *testing.T) {
	t.Parallel()
	registry := make(config.AgentsRegistry)
	registry["go"] = config.GroupConfig{
		Default: config.Preset{
			Client: "claude",
			EnvSet: map[string]string{"FOO": "bar", "BAZ": "qux"},
		},
		Kinds: make(map[domain.Kind]config.Override),
	}
	router := NewBackendRouter(&registry, ResolvedTemplate{Client: "claude"})

	item := domain.ActionItem{}
	envSet, envFromShell, err := router.ResolveEnvSet(item, "go", "build")
	if err != nil {
		t.Fatalf("ResolveEnvSet() unexpected error: %v", err)
	}

	// Verify envSet matches preset.
	if len(envSet) != 2 || envSet["FOO"] != "bar" || envSet["BAZ"] != "qux" {
		t.Errorf("envSet = %v, want {FOO:bar BAZ:qux}", envSet)
	}

	// EnvFromShell should be nil since preset has none.
	if envFromShell != nil {
		t.Errorf("envFromShell = %v, want nil", envFromShell)
	}

	// Verify defensive copy: mutate returned map and check original is unaffected.
	envSet["FOO"] = "modified"
	newEnvSet, _, _ := router.ResolveEnvSet(item, "go", "build")
	if newEnvSet["FOO"] != "bar" {
		t.Errorf("defensive copy failed: original EnvSet mutated")
	}
}

func TestBackendRouterResolveEnvSetResolvesEnvFromShellAsymmetric(t *testing.T) {
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "secret-value-123")
	t.Setenv("OLLAMA_ENDPOINT", "http://localhost:11434")

	registry := make(config.AgentsRegistry)
	registry["go"] = config.GroupConfig{
		Default: config.Preset{
			Client: "claude",
			EnvSet: map[string]string{"FOO": "bar"},
			// Asymmetric mapping: SPAWN_NAME (key) = orchestrator shell var (value)
			EnvFromShell: map[string]string{
				"OLLAMA_AUTH":     "ANTHROPIC_AUTH_TOKEN",
				"OLLAMA_BASE_URL": "OLLAMA_ENDPOINT",
			},
		},
		Kinds: make(map[domain.Kind]config.Override),
	}
	router := NewBackendRouter(&registry, ResolvedTemplate{Client: "claude"})

	item := domain.ActionItem{}
	envSet, envFromShell, err := router.ResolveEnvSet(item, "go", "build")
	if err != nil {
		t.Fatalf("ResolveEnvSet() unexpected error: %v", err)
	}

	// Verify envSet still present.
	if envSet["FOO"] != "bar" {
		t.Errorf("envSet[FOO] = %q, want bar", envSet["FOO"])
	}

	// Verify envFromShell is sorted and resolved.
	if len(envFromShell) != 2 {
		t.Fatalf("envFromShell length = %d, want 2", len(envFromShell))
	}

	// Entries should be sorted alphabetically by spawn name.
	expectedEntries := map[string]string{
		"OLLAMA_AUTH":     "secret-value-123",
		"OLLAMA_BASE_URL": "http://localhost:11434",
	}
	for _, entry := range envFromShell {
		parts := splitEnvEntry(entry)
		if len(parts) != 2 {
			t.Fatalf("invalid env entry: %q", entry)
		}
		spawnName, value := parts[0], parts[1]
		want, ok := expectedEntries[spawnName]
		if !ok {
			t.Errorf("unexpected spawn name: %q", spawnName)
			continue
		}
		if value != want {
			t.Errorf("envFromShell[%q] = %q, want %q", spawnName, value, want)
		}
	}

	// Verify deterministic ordering (sorted).
	if envFromShell[0] != "OLLAMA_AUTH=secret-value-123" &&
		envFromShell[0] != "OLLAMA_BASE_URL=http://localhost:11434" {
		t.Errorf("envFromShell[0] = %q, want alphabetically first entry", envFromShell[0])
	}
}

func TestBackendRouterResolveEnvSetMissingShellVarFailsLoud(t *testing.T) {
	// Ensure ANTHROPIC_AUTH_TOKEN is definitely unset.
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")
	t.Setenv("OLLAMA_ENDPOINT", "http://localhost:11434")

	registry := make(config.AgentsRegistry)
	registry["go"] = config.GroupConfig{
		Default: config.Preset{
			Client: "claude",
			EnvFromShell: map[string]string{
				"OLLAMA_AUTH": "UNSET_SHELL_VAR",
			},
		},
		Kinds: make(map[domain.Kind]config.Override),
	}
	router := NewBackendRouter(&registry, ResolvedTemplate{Client: "claude"})

	item := domain.ActionItem{}
	_, _, err := router.ResolveEnvSet(item, "go", "build")

	if err == nil {
		t.Fatalf("ResolveEnvSet() wanted error for missing shell var, got nil")
	}

	// Verify error wraps the sentinel.
	if !errors.Is(err, ErrEnvFromShellMissingShellVar) {
		t.Fatalf("error = %v, want to wrap ErrEnvFromShellMissingShellVar", err)
	}

	// Verify error message includes diagnostic context.
	errStr := err.Error()
	if !containsStr(errStr, "OLLAMA_AUTH") || !containsStr(errStr, "UNSET_SHELL_VAR") {
		t.Errorf("error message missing diagnostic: %v", err)
	}
}

func TestBackendRouterResolveEnvSetEmptyPresetYieldsNil(t *testing.T) {
	t.Parallel()
	registry := make(config.AgentsRegistry)
	registry["go"] = config.GroupConfig{
		Default: config.Preset{
			Client: "claude",
			// No EnvSet, no EnvFromShell.
		},
		Kinds: make(map[domain.Kind]config.Override),
	}
	router := NewBackendRouter(&registry, ResolvedTemplate{Client: "claude"})

	item := domain.ActionItem{}
	envSet, envFromShell, err := router.ResolveEnvSet(item, "go", "build")
	if err != nil {
		t.Fatalf("ResolveEnvSet() unexpected error: %v", err)
	}

	if envSet != nil {
		t.Errorf("envSet = %v, want nil for empty preset", envSet)
	}

	if envFromShell != nil {
		t.Errorf("envFromShell = %v, want nil for empty preset", envFromShell)
	}
}

// Helper functions for tests.

func splitEnvEntry(entry string) []string {
	for i := 0; i < len(entry); i++ {
		if entry[i] == '=' {
			return []string{entry[:i], entry[i+1:]}
		}
	}
	return []string{entry}
}

func containsStr(haystack, needle string) bool {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
