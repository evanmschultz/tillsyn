// Drop 4d_5 D0 — tests for loadAgentsRegistryAndValidate, the boot-time
// agents.toml loader that wires config.LoadMultiGroupRegistry into production
// and runs the parallel-peer conflict detector against the embedded default
// templates. Co-located with the production helper in main.go.
package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/evanmschultz/tillsyn/internal/templates"
)

// writeAgentsTOML writes a minimal multi-group agents.toml fixture for tests.
// The shape mirrors the canonical agents.example.toml: one [<group>] block
// with default Client + Model declarations, suitable for driving the
// parallel-peer conflict detector. Returns the absolute file path so tests
// can grep error messages for it.
func writeAgentsTOML(t *testing.T, dir, group, client string) string {
	t.Helper()
	path := filepath.Join(dir, "agents.toml")
	body := `[` + group + `]
client = "` + client + `"
model = "sonnet"
max_tries = 3
max_turns = 20
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile(agents.toml): %v", err)
	}
	return path
}

// TestLoadAgentsRegistryAndValidate_AbsentReturnsNil pins the absence-is-fine
// contract: a fresh machine with no agents.toml in the search directory
// returns (nil, nil) so boot proceeds with the legacy default-to-claude
// resolution path per F.7.17 L15.
func TestLoadAgentsRegistryAndValidate_AbsentReturnsNil(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	registry, err := loadAgentsRegistryAndValidate(dir)
	if err != nil {
		t.Fatalf("absent agents.toml should return nil error; got %v", err)
	}
	if registry != nil {
		t.Fatalf("absent agents.toml should return nil registry; got %v", registry)
	}
}

// TestLoadAgentsRegistryAndValidate_EmptySearchDirReturnsNil covers the
// os.Getwd failure fallback: when the search dir is empty (e.g. a
// containerised CI runner with a torn-down CWD) the loader treats every
// candidate file as absent and returns (nil, nil). Boot proceeds with the
// legacy resolution path rather than failing on a transient filesystem
// quirk.
func TestLoadAgentsRegistryAndValidate_EmptySearchDirReturnsNil(t *testing.T) {
	t.Parallel()

	registry, err := loadAgentsRegistryAndValidate("")
	if err != nil {
		t.Fatalf("empty search dir should return nil error; got %v", err)
	}
	if registry != nil {
		t.Fatalf("empty search dir should return nil registry; got %v", registry)
	}

	registry, err = loadAgentsRegistryAndValidate("   ")
	if err != nil {
		t.Fatalf("whitespace-only search dir should return nil error; got %v", err)
	}
	if registry != nil {
		t.Fatalf("whitespace-only search dir should return nil registry; got %v", registry)
	}
}

// TestLoadAgentsRegistryAndValidate_PresentAndAgreementReturnsRegistry verifies
// the happy path: a well-formed agents.toml whose Preset.Client agrees with
// the embedded builtin template's CLIKind passes validation and returns the
// merged registry. The embedded templates ship with claude-side bindings, so
// an agents.toml that declares client="claude" should always validate.
func TestLoadAgentsRegistryAndValidate_PresentAndAgreementReturnsRegistry(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeAgentsTOML(t, dir, "go", "claude")

	registry, err := loadAgentsRegistryAndValidate(dir)
	if err != nil {
		t.Fatalf("present + agreement should return nil error; got %v", err)
	}
	if registry == nil {
		t.Fatal("present + agreement should return non-nil registry")
	}
	if _, ok := (*registry)["go"]; !ok {
		t.Fatalf("merged registry missing [go] group; got %v", *registry)
	}
}

// TestLoadAgentsRegistryAndValidate_LocalWithoutProjectRejects covers the
// authoring-footgun guard: agents.local.toml exists but agents.toml does not.
// The local override has nothing to merge over, so the loader fails loud
// rather than silently treating local as the base. Pre-condition: helps
// adopters who accidentally only authored the override file see the issue
// at boot.
func TestLoadAgentsRegistryAndValidate_LocalWithoutProjectRejects(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "agents.local.toml"), []byte("[go]\nclient = \"claude\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(agents.local.toml): %v", err)
	}

	_, err := loadAgentsRegistryAndValidate(dir)
	if err == nil {
		t.Fatal("local without project should reject; got nil error")
	}
	if !strings.Contains(err.Error(), "agents.local.toml") {
		t.Fatalf("error message missing 'agents.local.toml' reference: %v", err)
	}
}

// TestLoadAgentsRegistryAndValidate_ConflictAgainstProjectLocalTemplate
// is the core D0 acceptance scenario applied to a project-local template at
// `<projectRoot>/.tillsyn/template.toml`. The adopter has authored a template
// pinning kind=build to cli_kind="claude" but their agents.toml declares
// client = "codex". The loader MUST return an error wrapping
// ErrConflictingCLIKind at boot, NOT at first spawn (round-2 Open Q 6.3
// resolution).
func TestLoadAgentsRegistryAndValidate_ConflictAgainstProjectLocalTemplate(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeAgentsTOML(t, dir, "go", "codex")

	// Write a minimal .tillsyn/template.toml that pins kind=build to
	// cli_kind="claude" — the conflict surface every adopter actually hits.
	tplDir := filepath.Join(dir, ".tillsyn")
	if err := os.MkdirAll(tplDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(.tillsyn): %v", err)
	}
	tplBody := `schema_version = "v1"

[kinds.build]
allowed_parent_kinds = ["plan"]
allowed_child_kinds = ["build-qa-proof", "build-qa-falsification"]
structural_type = "droplet"

[[child_rules]]
when_parent_kind = "build"
create_child_kind = "build-qa-proof"
title = "BUILD-QA-PROOF"
blocked_by_parent = true

[[child_rules]]
when_parent_kind = "build"
create_child_kind = "build-qa-falsification"
title = "BUILD-QA-FALSIFICATION"
blocked_by_parent = true

[agent_bindings.build]
agent_name = "builder-agent"
model = "sonnet"
cli_kind = "claude"
`
	if err := os.WriteFile(filepath.Join(tplDir, "template.toml"), []byte(tplBody), 0o644); err != nil {
		t.Fatalf("WriteFile(template.toml): %v", err)
	}

	_, err := loadAgentsRegistryAndValidate(dir)
	if err == nil {
		t.Fatal("conflict (agents=codex, template=claude) should fail loud; got nil error")
	}
	if !errors.Is(err, templates.ErrConflictingCLIKind) {
		t.Fatalf("error does not wrap ErrConflictingCLIKind: %v", err)
	}
	if !strings.Contains(err.Error(), "claude") || !strings.Contains(err.Error(), "codex") {
		t.Fatalf("error message missing client identifiers: %v", err)
	}
}

// TestLoadAgentsRegistryAndValidate_ProjectLocalTemplateAgreementPasses
// is the inverse: project-local template pins kind=build to cli_kind="codex"
// and agents.toml also declares client = "codex". Agreement → no conflict.
// This locks the symmetric-normalisation contract end-to-end (round-2 HIGH-3
// mitigation): agents.toml "Codex" vs template.toml "codex" must NOT raise
// a false conflict.
func TestLoadAgentsRegistryAndValidate_ProjectLocalTemplateAgreementPasses(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Use case-different forms to exercise normalizeClient symmetrically.
	writeAgentsTOML(t, dir, "go", "Codex")

	tplDir := filepath.Join(dir, ".tillsyn")
	if err := os.MkdirAll(tplDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(.tillsyn): %v", err)
	}
	tplBody := `schema_version = "v1"

[kinds.build]
allowed_parent_kinds = ["plan"]
allowed_child_kinds = ["build-qa-proof", "build-qa-falsification"]
structural_type = "droplet"

[[child_rules]]
when_parent_kind = "build"
create_child_kind = "build-qa-proof"
title = "BUILD-QA-PROOF"
blocked_by_parent = true

[[child_rules]]
when_parent_kind = "build"
create_child_kind = "build-qa-falsification"
title = "BUILD-QA-FALSIFICATION"
blocked_by_parent = true

[agent_bindings.build]
agent_name = "builder-agent"
model = "sonnet"
cli_kind = "codex"
`
	if err := os.WriteFile(filepath.Join(tplDir, "template.toml"), []byte(tplBody), 0o644); err != nil {
		t.Fatalf("WriteFile(template.toml): %v", err)
	}

	registry, err := loadAgentsRegistryAndValidate(dir)
	if err != nil {
		t.Fatalf("agreement (agents=Codex, template=codex) should pass; got %v", err)
	}
	if registry == nil {
		t.Fatal("agreement should return non-nil registry")
	}
}

// TestLoadAgentsRegistryAndValidate_ConflictAgainstBuiltinFailsLoud is the
// secondary acceptance scenario: an agents.toml whose Preset.Client="codex"
// disagrees with one of the embedded builtin templates. This test exercises
// the embedded-floor coverage. Currently the embedded templates do not
// declare cli_kind explicitly — the test skips when no embedded template
// carries a non-empty cli_kind binding.
//
// The embedded till-go template binds at least one of the build/plan/qa
// kinds to cli_kind="claude" via the canonical agents.example.toml floor;
// a registry that overrides the group default to "codex" forces the (group,
// kind) pair to disagree.
//
// This locks the round-2 Open Q 6.3 resolution: conflict surfaces at
// adopter time, not deferred to first spawn.
func TestLoadAgentsRegistryAndValidate_ConflictAgainstBuiltinFailsLoud(t *testing.T) {
	t.Parallel()
	// Pre-check: the embedded till-go template carries at least one
	// AgentBinding with cli_kind=="claude". Without that floor the conflict
	// scenario is vacuous. If a future drop changes the embedded shape,
	// this guard surfaces the divergence cleanly.
	tpl, err := templates.LoadBuiltinTemplate("till-go")
	if err != nil {
		t.Fatalf("LoadBuiltinTemplate(till-go): %v", err)
	}
	catalog := templates.Bake(tpl)
	hasClaudeBinding := false
	for _, binding := range catalog.AgentBindings {
		if strings.EqualFold(strings.TrimSpace(binding.CLIKind), "claude") {
			hasClaudeBinding = true
			break
		}
	}
	if !hasClaudeBinding {
		t.Skip("till-go template has no claude-side cli_kind binding; conflict scenario inapplicable")
	}

	dir := t.TempDir()
	// Use group name "go" so config.Resolve matches the till-go template's
	// kind axis. Client = codex disagrees with claude-side bindings.
	writeAgentsTOML(t, dir, "go", "codex")

	_, err = loadAgentsRegistryAndValidate(dir)
	if err == nil {
		t.Fatal("conflict should fail loud at boot; got nil error")
	}
	if !errors.Is(err, templates.ErrConflictingCLIKind) {
		t.Fatalf("error does not wrap ErrConflictingCLIKind: %v", err)
	}
}
