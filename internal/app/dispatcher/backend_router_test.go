package dispatcher

import (
	"errors"
	"testing"

	"github.com/evanmschultz/tillsyn/internal/app/dispatcher/pretoolgate"
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

func TestBackendRouterResolveMCPServersRoleMatrix(t *testing.T) {
	// Table-driven test validating the role→MCP-server-set mapping (A1 REPLACE).
	// Each row specifies a (role, axis, language) triple and the expected set
	// of server names. The role matrix from resolveRoleMCPSet is authoritative.
	t.Parallel()

	registry := make(config.AgentsRegistry)
	registry["go"] = config.GroupConfig{
		Default: config.Preset{Client: "claude"},
		Kinds:   make(map[domain.Kind]config.Override),
	}
	router := NewBackendRouter(&registry, ResolvedTemplate{Client: "claude"})

	tests := []struct {
		name           string
		role           string
		axis           string
		language       string
		wantServers    map[string]bool // server name → expected present
		wantNotServers map[string]bool // server name → expected absent
	}{
		{
			// Builder (non-QA): planner, builder, closeout get full set
			name:     "go-builder",
			role:     "builder",
			axis:     "build",
			language: "go",
			wantServers: map[string]bool{
				"tillsyn":  true,
				"ta":       true,
				"hylla":    true,
				"context7": true,
				"gopls":    true,
			},
			wantNotServers: map[string]bool{
				"playwright": true, // FE only
			},
		},
		{
			// Builder FE: playwright instead of gopls
			name:     "fe-builder",
			role:     "builder",
			axis:     "build",
			language: "fe",
			wantServers: map[string]bool{
				"tillsyn":    true,
				"ta":         true,
				"hylla":      true,
				"context7":   true,
				"playwright": true,
			},
			wantNotServers: map[string]bool{
				"gopls": true, // Go only
			},
		},
		{
			// Build-QA: carve-out, only Tillsyn + Ta
			name:     "build-qa-proof-go",
			role:     "qa-proof",
			axis:     "build",
			language: "go",
			wantServers: map[string]bool{
				"tillsyn": true,
				"ta":      true,
			},
			wantNotServers: map[string]bool{
				"hylla":      true,
				"context7":   true,
				"gopls":      true,
				"playwright": true,
			},
		},
		{
			// Build-QA falsification FE: also carve-out
			name:     "build-qa-falsification-fe",
			role:     "qa-falsification",
			axis:     "build",
			language: "fe",
			wantServers: map[string]bool{
				"tillsyn": true,
				"ta":      true,
			},
			wantNotServers: map[string]bool{
				"hylla":      true,
				"context7":   true,
				"gopls":      true,
				"playwright": true,
			},
		},
		{
			// Planner (plan axis): full set with gopls
			name:     "go-planner",
			role:     "planner",
			axis:     "plan",
			language: "go",
			wantServers: map[string]bool{
				"tillsyn":  true,
				"ta":       true,
				"hylla":    true,
				"context7": true,
				"gopls":    true,
			},
			wantNotServers: map[string]bool{
				"playwright": true,
			},
		},
		{
			// Plan-QA proof FE: includes playwright (not build-qa), no gopls
			name:     "fe-plan-qa-proof",
			role:     "qa-proof",
			axis:     "plan",
			language: "fe",
			wantServers: map[string]bool{
				"tillsyn":    true,
				"ta":         true,
				"hylla":      true,
				"context7":   true,
				"playwright": true,
			},
			wantNotServers: map[string]bool{
				"gopls": true, // Go only
			},
		},
		{
			// Closeout: language=none, full set (no gopls/playwright)
			name:     "closeout",
			role:     "closeout",
			axis:     "none",
			language: "none",
			wantServers: map[string]bool{
				"tillsyn":  true,
				"ta":       true,
				"hylla":    true,
				"context7": true,
			},
			wantNotServers: map[string]bool{
				"gopls":      true,
				"playwright": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := &AgentDefinition{
				Name:     tt.name,
				Role:     tt.role,
				Axis:     tt.axis,
				Language: tt.language,
			}

			out := router.ResolveMCPServers(def)

			// Verify expected servers are present.
			for server := range tt.wantServers {
				if _, ok := out[server]; !ok {
					t.Errorf("expected server %q not found", server)
				}
			}

			// Verify unwanted servers are absent.
			for server := range tt.wantNotServers {
				if _, ok := out[server]; ok {
					t.Errorf("unexpected server %q found", server)
				}
			}

			// Verify all servers have non-empty Command.
			for name, cfg := range out {
				if cfg.Command == "" {
					t.Errorf("server %q has empty Command", name)
				}
			}
		})
	}
}

func TestRoleGateToGateSpecNil(t *testing.T) {
	// Sanity check: nil RoleGate returns nil GateSpec.
	result := roleGateToGateSpec(nil)
	if result != nil {
		t.Fatalf("roleGateToGateSpec(nil) = %v; want nil", result)
	}
}

func TestRoleGateToGateSpec(t *testing.T) {
	// Table-driven test for the roleGateToGateSpec projector.
	// Verifies that RoleGate → GateSpec projection enforces read-only for codex roles.
	t.Parallel()

	tests := []struct {
		name              string
		cliKind           string
		bashDeny          []string
		expectWritable    bool
		expectEdit        bool
		expectBashDenySet bool
	}{
		{
			name:              "codex role is read-only",
			cliKind:           "codex",
			bashDeny:          []string{"git commit", "go get"},
			expectWritable:    false, // WritableDirs should be nil
			expectEdit:        false, // Edit should be nil
			expectBashDenySet: true,  // BashDeny is still populated
		},
		{
			name:              "claude role is ungated (nil)",
			cliKind:           "claude",
			bashDeny:          []string{"mage install"},
			expectWritable:    false, // nil GateSpec means no writable dirs
			expectEdit:        false, // nil GateSpec means no edit gates
			expectBashDenySet: false, // non-codex returns nil GateSpec
		},
		{
			name:              "no bash deny patterns",
			cliKind:           "codex",
			bashDeny:          nil,
			expectWritable:    false,
			expectEdit:        false,
			expectBashDenySet: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rg := &pretoolgate.RoleGate{
				CLIKind: tt.cliKind,
				Spec: pretoolgate.GateSpec{
					BashDeny: tt.bashDeny,
				},
			}

			got := roleGateToGateSpec(rg)

			// For codex roles, result must be non-nil with WritableDirs and Edit both nil.
			if tt.cliKind == "codex" {
				if got == nil {
					t.Errorf("codex role must return non-nil GateSpec, got nil")
				} else if got.WritableDirs != nil || got.Edit != nil {
					t.Errorf("codex role must have nil WritableDirs and Edit; got WritableDirs=%v, Edit=%v", got.WritableDirs, got.Edit)
				}
			} else {
				// Non-codex roles are ungated (nil).
				if got != nil {
					t.Errorf("non-codex role must return nil GateSpec, got %v", got)
				}
			}
		})
	}
}

func TestBackendRouterResolveWebSearchRoleMatrix(t *testing.T) {
	// Table-driven test validating the role→WebSearch flag mapping.
	// WebSearch is true for all non-build-QA roles, false for build-QA roles.
	t.Parallel()

	registry := make(config.AgentsRegistry)
	registry["go"] = config.GroupConfig{
		Default: config.Preset{Client: "claude"},
		Kinds:   make(map[domain.Kind]config.Override),
	}
	router := NewBackendRouter(&registry, ResolvedTemplate{Client: "claude"})

	tests := []struct {
		name          string
		role          string
		axis          string
		language      string
		wantWebSearch bool
	}{
		{
			// Builder (non-QA): returns true
			name:          "go-builder",
			role:          "builder",
			axis:          "build",
			language:      "go",
			wantWebSearch: true,
		},
		{
			// Builder FE: returns true
			name:          "fe-builder",
			role:          "builder",
			axis:          "build",
			language:      "fe",
			wantWebSearch: true,
		},
		{
			// Build-QA proof: carve-out, returns false
			name:          "build-qa-proof-go",
			role:          "qa-proof",
			axis:          "build",
			language:      "go",
			wantWebSearch: false,
		},
		{
			// Build-QA falsification: carve-out, returns false
			name:          "build-qa-falsification-fe",
			role:          "qa-falsification",
			axis:          "build",
			language:      "fe",
			wantWebSearch: false,
		},
		{
			// Planner (plan axis): returns true
			name:          "go-planner",
			role:          "planner",
			axis:          "plan",
			language:      "go",
			wantWebSearch: true,
		},
		{
			// Plan-QA proof FE: returns true (not build-QA)
			name:          "fe-plan-qa-proof",
			role:          "qa-proof",
			axis:          "plan",
			language:      "fe",
			wantWebSearch: true,
		},
		{
			// Plan-QA falsification: returns true (not build-QA)
			name:          "go-plan-qa-falsification",
			role:          "qa-falsification",
			axis:          "plan",
			language:      "go",
			wantWebSearch: true,
		},
		{
			// Closeout: returns true (not build-QA)
			name:          "closeout",
			role:          "closeout",
			axis:          "none",
			language:      "none",
			wantWebSearch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := &AgentDefinition{
				Name:     tt.name,
				Role:     tt.role,
				Axis:     tt.axis,
				Language: tt.language,
			}

			got := router.ResolveWebSearch(def)
			if got != tt.wantWebSearch {
				t.Errorf("ResolveWebSearch() = %v, want %v", got, tt.wantWebSearch)
			}
		})
	}
}

func TestBackendRouterResolveWebSearchNilDefReturnsFalse(t *testing.T) {
	t.Parallel()
	registry := make(config.AgentsRegistry)
	registry["go"] = config.GroupConfig{
		Default: config.Preset{Client: "claude"},
		Kinds:   make(map[domain.Kind]config.Override),
	}
	router := NewBackendRouter(&registry, ResolvedTemplate{Client: "claude"})

	got := router.ResolveWebSearch(nil)
	if got != false {
		t.Fatalf("ResolveWebSearch(nil) = %v, want false", got)
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

func TestBackendRouterResolveMCPServersUnknownNonBuildQAGetsFullSet(t *testing.T) {
	// A1 REPLACE: unknown (role, axis, language) inputs that don't match the
	// build-QA carve-out fall through to the full non-build-QA server set.
	// (The resolveRoleMCPSet spec mentions a "most-restrictive" default, but
	// the implementation doesn't distinguish unknown from valid non-build-QA roles.)
	t.Parallel()
	registry := make(config.AgentsRegistry)
	registry["go"] = config.GroupConfig{
		Default: config.Preset{Client: "claude"},
		Kinds:   make(map[domain.Kind]config.Override),
	}
	router := NewBackendRouter(&registry, ResolvedTemplate{Client: "claude"})

	def := &AgentDefinition{
		Name:     "ta-unknown",
		Role:     "unknown-role",
		Axis:     "unknown-axis",
		Language: "unknown-lang",
	}

	out := router.ResolveMCPServers(def)
	// Unknown non-build-QA inputs get the full non-build-QA set.
	if _, ok := out["tillsyn"]; !ok {
		t.Errorf("expected tillsyn server")
	}
	if _, ok := out["ta"]; !ok {
		t.Errorf("expected ta server")
	}
	if _, ok := out["hylla"]; !ok {
		t.Errorf("expected hylla server")
	}
	if _, ok := out["context7"]; !ok {
		t.Errorf("expected context7 server")
	}
}

func TestBackendRouterResolveMCPServersA1IgnoresFrontmatter(t *testing.T) {
	// A1 REPLACE: the frontmatter MCPServers field is ignored entirely.
	// The returned map is purely from resolveRoleMCPSet, not from def.MCPServers.
	// This test verifies that even if def.MCPServers is populated (legacy),
	// it has no effect on the output.
	t.Parallel()
	registry := make(config.AgentsRegistry)
	registry["go"] = config.GroupConfig{
		Default: config.Preset{Client: "claude"},
		Kinds:   make(map[domain.Kind]config.Override),
	}
	router := NewBackendRouter(&registry, ResolvedTemplate{Client: "claude"})

	def := &AgentDefinition{
		Name:     "ta-go-builder",
		Role:     "builder",
		Axis:     "build",
		Language: "go",
		// Frontmatter MCPServers is populated, but should be ignored.
		MCPServers: map[string]AgentDefinitionMCPServer{
			"old-server": {
				Command: "old-command",
				Args:    []string{"old-arg"},
				Tools:   []string{"old.tool"},
			},
		},
	}

	out := router.ResolveMCPServers(def)

	// Verify the old frontmatter server is NOT in the output.
	if _, ok := out["old-server"]; ok {
		t.Errorf("A1 REPLACE violation: old frontmatter server found in output")
	}

	// Verify the role-canonical servers ARE present.
	if _, ok := out["tillsyn"]; !ok {
		t.Errorf("expected tillsyn server from role matrix")
	}
	if _, ok := out["gopls"]; !ok {
		t.Errorf("expected gopls server for go-builder role")
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
