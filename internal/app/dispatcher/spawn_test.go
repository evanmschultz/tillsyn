package dispatcher

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/evanmschultz/tillsyn/internal/templates"
)

// goBuilderBinding returns one canonical AgentBinding fixture used by the
// happy-path spawn tests. The values mirror the planner fixture from
// WAVE_2_PLAN.md §2.6: agent_name = "go-builder-agent", model = "opus",
// max_turns = 20, max_budget_usd = 5.
func goBuilderBinding() templates.AgentBinding {
	return templates.AgentBinding{
		AgentName:    "go-builder-agent",
		Model:        "opus",
		MaxTries:     1,
		MaxBudgetUSD: 5,
		MaxTurns:     20,
	}
}

// fixtureCatalog returns one templates.KindCatalog with a single binding for
// domain.KindBuild keyed at the supplied AgentBinding. Used by the happy-path
// scenarios that exercise the full argv assembly.
func fixtureCatalog(binding templates.AgentBinding) templates.KindCatalog {
	return templates.KindCatalog{
		SchemaVersion: templates.SchemaVersionV1,
		AgentBindings: map[domain.Kind]templates.AgentBinding{
			domain.KindBuild: binding,
		},
	}
}

// fixtureProject returns a project value populated with the four fields
// BuildSpawnCommand reads (RepoPrimaryWorktree, ID, HyllaArtifactRef,
// Language). Other fields stay zero-value: they are not consumed in 4a.19.
func fixtureProject() domain.Project {
	return domain.Project{
		ID:                  "proj-1",
		RepoPrimaryWorktree: "/tmp/tillsyn/main",
		HyllaArtifactRef:    "github.com/evanmschultz/tillsyn@main",
		Language:            "go",
	}
}

// fixtureBuildItem returns a build action item with the minimum fields
// BuildSpawnCommand consumes (ID, Kind, Title).
func fixtureBuildItem() domain.ActionItem {
	return domain.ActionItem{
		ID:    "ai-build-1",
		Kind:  domain.KindBuild,
		Title: "DROPLET 4A.19 EXAMPLE BUILD",
	}
}

// TestBuildSpawnCommandAssemblesArgvForGoBuilder asserts the argv slice the
// constructed *exec.Cmd carries matches the REVISION_BRIEF Wave-2 spec when
// the catalog binds kind=build to a go-builder-agent fixture.
func TestBuildSpawnCommandAssemblesArgvForGoBuilder(t *testing.T) {
	t.Parallel()

	item := fixtureBuildItem()
	project := fixtureProject()
	catalog := fixtureCatalog(goBuilderBinding())

	cmd, descriptor, err := BuildSpawnCommand(item, project, catalog, AuthBundle{})
	if err != nil {
		t.Fatalf("BuildSpawnCommand() error = %v, want nil", err)
	}
	if cmd == nil {
		t.Fatalf("BuildSpawnCommand() cmd = nil, want non-nil")
	}

	wantMCP := filepath.Join(project.RepoPrimaryWorktree, ".tillsyn", "dispatcher-spawn-"+item.ID+".json")
	wantArgv := []string{
		"claude",
		"--agent", "go-builder-agent",
		"--bare",
		"-p", descriptor.Prompt,
		"--mcp-config", wantMCP,
		"--strict-mcp-config",
		"--permission-mode", "acceptEdits",
		"--max-budget-usd", "5",
		"--max-turns", "20",
	}

	if len(cmd.Args) != len(wantArgv) {
		t.Fatalf("argv length = %d, want %d (got %v)", len(cmd.Args), len(wantArgv), cmd.Args)
	}
	for i := range wantArgv {
		if cmd.Args[i] != wantArgv[i] {
			t.Errorf("argv[%d] = %q, want %q", i, cmd.Args[i], wantArgv[i])
		}
	}

	// Descriptor mirrors the binding + resolved paths.
	if descriptor.AgentName != "go-builder-agent" {
		t.Errorf("descriptor.AgentName = %q, want %q", descriptor.AgentName, "go-builder-agent")
	}
	if descriptor.Model != "opus" {
		t.Errorf("descriptor.Model = %q, want %q", descriptor.Model, "opus")
	}
	if descriptor.MaxBudgetUSD != 5 {
		t.Errorf("descriptor.MaxBudgetUSD = %v, want 5", descriptor.MaxBudgetUSD)
	}
	if descriptor.MaxTurns != 20 {
		t.Errorf("descriptor.MaxTurns = %d, want 20", descriptor.MaxTurns)
	}
	if descriptor.MCPConfigPath != wantMCP {
		t.Errorf("descriptor.MCPConfigPath = %q, want %q", descriptor.MCPConfigPath, wantMCP)
	}
	if descriptor.WorkingDir != project.RepoPrimaryWorktree {
		t.Errorf("descriptor.WorkingDir = %q, want %q", descriptor.WorkingDir, project.RepoPrimaryWorktree)
	}

	// Prompt structural fields — body opaque, but key tokens MUST be present.
	wantTokens := []string{
		"task_id: " + item.ID,
		"project_id: " + project.ID,
		"project_dir: " + project.RepoPrimaryWorktree,
		"hylla_artifact_ref: " + project.HyllaArtifactRef,
		"kind: " + string(item.Kind),
		"move-state directive:",
	}
	for _, tok := range wantTokens {
		if !strings.Contains(descriptor.Prompt, tok) {
			t.Errorf("descriptor.Prompt missing %q\nfull prompt:\n%s", tok, descriptor.Prompt)
		}
	}
}

// TestBuildSpawnCommandSetsCwd asserts the constructed command's working
// directory matches the project's primary worktree — the dispatcher's
// `cd`-target invariant for spawned subagents.
func TestBuildSpawnCommandSetsCwd(t *testing.T) {
	t.Parallel()

	project := fixtureProject()
	cmd, _, err := BuildSpawnCommand(fixtureBuildItem(), project, fixtureCatalog(goBuilderBinding()), AuthBundle{})
	if err != nil {
		t.Fatalf("BuildSpawnCommand() error = %v, want nil", err)
	}
	if cmd.Dir != project.RepoPrimaryWorktree {
		t.Fatalf("cmd.Dir = %q, want %q", cmd.Dir, project.RepoPrimaryWorktree)
	}
}

// TestBuildSpawnCommandReturnsErrNoAgentBindingForUnboundKind asserts that a
// kind missing from the catalog's AgentBindings map surfaces as
// ErrNoAgentBinding rather than a panic or generic error.
func TestBuildSpawnCommandReturnsErrNoAgentBindingForUnboundKind(t *testing.T) {
	t.Parallel()

	// Catalog has bindings — but only for plan-qa-proof, not build.
	catalog := templates.KindCatalog{
		SchemaVersion: templates.SchemaVersionV1,
		AgentBindings: map[domain.Kind]templates.AgentBinding{
			domain.KindPlanQAProof: goBuilderBinding(), // arbitrary; key matters
		},
	}
	item := fixtureBuildItem() // kind=build, not in the map

	cmd, _, err := BuildSpawnCommand(item, fixtureProject(), catalog, AuthBundle{})
	if err == nil {
		t.Fatalf("BuildSpawnCommand() error = nil, want ErrNoAgentBinding")
	}
	if !errors.Is(err, ErrNoAgentBinding) {
		t.Fatalf("BuildSpawnCommand() error = %v, want errors.Is(ErrNoAgentBinding)", err)
	}
	if cmd != nil {
		t.Fatalf("BuildSpawnCommand() cmd = %v, want nil on error", cmd)
	}
}

// TestBuildSpawnCommandReturnsErrNoAgentBindingForEmptyCatalog covers the
// zero-value catalog path (no template bound at project creation). Confirms
// the LookupAgentBinding nil-map guard in templates.KindCatalog surfaces as
// ErrNoAgentBinding here, NOT as a nil-map panic.
func TestBuildSpawnCommandReturnsErrNoAgentBindingForEmptyCatalog(t *testing.T) {
	t.Parallel()

	cmd, _, err := BuildSpawnCommand(fixtureBuildItem(), fixtureProject(), templates.KindCatalog{}, AuthBundle{})
	if err == nil {
		t.Fatalf("BuildSpawnCommand() error = nil, want ErrNoAgentBinding")
	}
	if !errors.Is(err, ErrNoAgentBinding) {
		t.Fatalf("BuildSpawnCommand() error = %v, want errors.Is(ErrNoAgentBinding)", err)
	}
	if cmd != nil {
		t.Fatalf("BuildSpawnCommand() cmd = %v, want nil on error", cmd)
	}
}

// TestBuildSpawnCommandPropagatesAuthBundleStubPath asserts the placeholder
// `--mcp-config` path is non-empty, lives under the worktree's `.tillsyn/`
// dir, and contains the action-item ID for cross-spawn disambiguation. This
// pins the Wave-3 seam: when Wave 3 lands, the path shape (under .tillsyn,
// keyed by action-item ID) MUST stay compatible.
func TestBuildSpawnCommandPropagatesAuthBundleStubPath(t *testing.T) {
	t.Parallel()

	item := fixtureBuildItem()
	project := fixtureProject()

	_, descriptor, err := BuildSpawnCommand(item, project, fixtureCatalog(goBuilderBinding()), AuthBundle{})
	if err != nil {
		t.Fatalf("BuildSpawnCommand() error = %v, want nil", err)
	}
	if descriptor.MCPConfigPath == "" {
		t.Fatalf("descriptor.MCPConfigPath is empty, want a placeholder path")
	}

	wantPrefix := filepath.Join(project.RepoPrimaryWorktree, ".tillsyn") + string(filepath.Separator)
	if !strings.HasPrefix(descriptor.MCPConfigPath, wantPrefix) {
		t.Errorf("descriptor.MCPConfigPath = %q, want prefix %q", descriptor.MCPConfigPath, wantPrefix)
	}
	if !strings.Contains(descriptor.MCPConfigPath, item.ID) {
		t.Errorf("descriptor.MCPConfigPath = %q, want contains action-item ID %q", descriptor.MCPConfigPath, item.ID)
	}
}

// TestBuildSpawnCommandRejectsCorruptedAgentBinding asserts the defensive
// AgentBinding.Validate re-call inside BuildSpawnCommand catches bindings
// that were corrupted in-memory after template-load (the only way an empty
// AgentName can survive load — Validate rejects it at the loader). This
// pins the plan-QA falsification mitigation from WAVE_2_PLAN.md §2.6.
func TestBuildSpawnCommandRejectsCorruptedAgentBinding(t *testing.T) {
	t.Parallel()

	corrupted := goBuilderBinding()
	corrupted.AgentName = "" // would have been rejected by Validate at load

	catalog := fixtureCatalog(corrupted)
	cmd, _, err := BuildSpawnCommand(fixtureBuildItem(), fixtureProject(), catalog, AuthBundle{})
	if err == nil {
		t.Fatalf("BuildSpawnCommand() error = nil, want ErrInvalidAgentBinding wrap")
	}
	if !errors.Is(err, templates.ErrInvalidAgentBinding) {
		t.Fatalf("BuildSpawnCommand() error = %v, want errors.Is(templates.ErrInvalidAgentBinding)", err)
	}
	if cmd != nil {
		t.Fatalf("BuildSpawnCommand() cmd = %v, want nil on error", cmd)
	}
}

// TestBuildSpawnCommandRejectsEmptyActionItemID covers the input-validation
// guard: empty IDs cannot construct a deterministic MCPConfigPath, so the
// constructor rejects rather than emit a path with an empty segment.
func TestBuildSpawnCommandRejectsEmptyActionItemID(t *testing.T) {
	t.Parallel()

	item := fixtureBuildItem()
	item.ID = "   "

	cmd, _, err := BuildSpawnCommand(item, fixtureProject(), fixtureCatalog(goBuilderBinding()), AuthBundle{})
	if err == nil {
		t.Fatalf("BuildSpawnCommand() error = nil, want ErrInvalidSpawnInput")
	}
	if !errors.Is(err, ErrInvalidSpawnInput) {
		t.Fatalf("BuildSpawnCommand() error = %v, want errors.Is(ErrInvalidSpawnInput)", err)
	}
	if cmd != nil {
		t.Fatalf("BuildSpawnCommand() cmd = %v, want nil on error", cmd)
	}
}

// TestBuildSpawnCommandRejectsEmptyKind covers the input-validation guard
// for empty Kind — the catalog lookup would silently miss without this
// gate, returning the less-precise ErrNoAgentBinding.
func TestBuildSpawnCommandRejectsEmptyKind(t *testing.T) {
	t.Parallel()

	item := fixtureBuildItem()
	item.Kind = ""

	_, _, err := BuildSpawnCommand(item, fixtureProject(), fixtureCatalog(goBuilderBinding()), AuthBundle{})
	if err == nil {
		t.Fatalf("BuildSpawnCommand() error = nil, want ErrInvalidSpawnInput")
	}
	if !errors.Is(err, ErrInvalidSpawnInput) {
		t.Fatalf("BuildSpawnCommand() error = %v, want errors.Is(ErrInvalidSpawnInput)", err)
	}
}

// TestBuildSpawnCommandRejectsEmptyWorktree covers the input-validation
// guard for an empty RepoPrimaryWorktree — without it, cmd.Dir would default
// to the caller's cwd at execution time, which the dispatcher contract
// explicitly forbids.
func TestBuildSpawnCommandRejectsEmptyWorktree(t *testing.T) {
	t.Parallel()

	project := fixtureProject()
	project.RepoPrimaryWorktree = ""

	_, _, err := BuildSpawnCommand(fixtureBuildItem(), project, fixtureCatalog(goBuilderBinding()), AuthBundle{})
	if err == nil {
		t.Fatalf("BuildSpawnCommand() error = nil, want ErrInvalidSpawnInput")
	}
	if !errors.Is(err, ErrInvalidSpawnInput) {
		t.Fatalf("BuildSpawnCommand() error = %v, want errors.Is(ErrInvalidSpawnInput)", err)
	}
}

// TestBuildSpawnCommandFormatsFractionalBudget asserts a non-integer
// MaxBudgetUSD (e.g. 2.5) round-trips into the CLI flag without spurious
// trailing zeros. The format helper renders 2.5 as "2.5", not "2.50".
func TestBuildSpawnCommandFormatsFractionalBudget(t *testing.T) {
	t.Parallel()

	binding := goBuilderBinding()
	binding.MaxBudgetUSD = 2.5

	cmd, descriptor, err := BuildSpawnCommand(fixtureBuildItem(), fixtureProject(), fixtureCatalog(binding), AuthBundle{})
	if err != nil {
		t.Fatalf("BuildSpawnCommand() error = %v, want nil", err)
	}
	if descriptor.MaxBudgetUSD != 2.5 {
		t.Errorf("descriptor.MaxBudgetUSD = %v, want 2.5", descriptor.MaxBudgetUSD)
	}
	// Locate the --max-budget-usd flag value in argv and assert formatting.
	found := false
	for i := 0; i < len(cmd.Args)-1; i++ {
		if cmd.Args[i] == "--max-budget-usd" {
			if cmd.Args[i+1] != "2.5" {
				t.Errorf("--max-budget-usd value = %q, want %q", cmd.Args[i+1], "2.5")
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("--max-budget-usd flag not found in argv: %v", cmd.Args)
	}
}

// TestSpawnDescriptorZeroValueHasNoFields is a regression pin: SpawnDescriptor
// is a value type, and accidental shifting of fields between BuildSpawnCommand
// and a future caller would silently rebind the wrong values. This test
// constructs the zero value and asserts the documented invariants — empty
// strings + zero numerics — survive any future field reordering.
func TestSpawnDescriptorZeroValueHasNoFields(t *testing.T) {
	t.Parallel()

	var d SpawnDescriptor
	if d.AgentName != "" || d.Model != "" || d.MCPConfigPath != "" || d.Prompt != "" || d.WorkingDir != "" {
		t.Errorf("SpawnDescriptor zero value has non-empty string field: %+v", d)
	}
	if d.MaxBudgetUSD != 0 || d.MaxTurns != 0 {
		t.Errorf("SpawnDescriptor zero value has non-zero numeric field: %+v", d)
	}
}
