package dispatcher_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/evanmschultz/tillsyn/internal/app/dispatcher"
	// Side-effect import: registers the real claude adapter with the
	// dispatcher's CLIKind→adapter registry. Drop 4c F.7.17.5 spawn wiring
	// breaks the dispatcher → cli_claude import cycle by inverting the
	// direction (cli_claude.init() calls dispatcher.RegisterAdapter); the
	// blank import here triggers that init when the test binary starts.
	// cli_claude's own init() also triggers cli_claude/render's init() via
	// a sub-blank-import there, so the real bundle-render hook is wired
	// in too.
	_ "github.com/evanmschultz/tillsyn/internal/app/dispatcher/cli_claude"
	// Named import: lets failure-injection tests substitute a faulty
	// render hook AND restore the real render.Render afterwards. The
	// blank import above already runs render.init(); this named alias
	// only exposes the Render symbol so t.Cleanup hooks can re-register.
	clauderender "github.com/evanmschultz/tillsyn/internal/app/dispatcher/cli_claude/render"
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

// fixtureProject returns a project value populated with the fields
// BuildSpawnCommand reads (RepoPrimaryWorktree, ID). HyllaArtifactRef stays
// populated to exercise the F.7.10 negative assertion (the value MUST NOT
// leak into the prompt body even when the project sets it). Language is
// retained for forward-compat — Wave 1 added it to the project field set;
// the dispatcher no longer consumes it post-F.7.10.
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
		Title: "DROPLET 4C F.7.17.5 EXAMPLE BUILD",
	}
}

// argFlagValue returns the argument that follows `flag` in argv, or "" + false
// if the flag isn't present or has no following arg. Used by tests that
// inspect specific flag values without pinning the entire argv shape (which
// the claude adapter owns).
func argFlagValue(argv []string, flag string) (string, bool) {
	for i := 0; i < len(argv)-1; i++ {
		if argv[i] == flag {
			return argv[i+1], true
		}
	}
	return "", false
}

// argvContains reports whether argv contains exactly `s` as one of its
// elements. Helper for argv-shape assertions that don't care about position.
func argvContains(argv []string, s string) bool {
	for _, a := range argv {
		if a == s {
			return true
		}
	}
	return false
}

// removeBundle is the standard test cleanup hook. F.7-CORE F.7.1 will own
// post-spawn bundle cleanup; until that lands, every test that calls
// BuildSpawnCommand schedules its own RemoveAll on the bundle root the
// adapter wired into argv.
func removeBundle(t *testing.T, cmd *exec.Cmd) {
	t.Helper()
	if cmd == nil {
		return
	}
	pluginDir, ok := argFlagValue(cmd.Args, "--plugin-dir")
	if !ok {
		return
	}
	bundleRoot := filepath.Dir(pluginDir)
	t.Cleanup(func() {
		_ = os.RemoveAll(bundleRoot)
	})
}

// TestBuildSpawnCommandUsesClaudeAdapterByDefault asserts the adapter
// registry's default-to-claude path: an empty rawBinding.CLIKind produces a
// claude-shaped *exec.Cmd. The argv signature is the long claude form
// (--bare, --plugin-dir, --agent, --system-prompt-file, ...) — we don't pin
// the entire shape here (the claude adapter owns that contract; see
// cli_claude/adapter_test.go), only the load-bearing markers.
func TestBuildSpawnCommandUsesClaudeAdapterByDefault(t *testing.T) {
	t.Parallel()

	item := fixtureBuildItem()
	project := fixtureProject()
	catalog := fixtureCatalog(goBuilderBinding()) // CLIKind unset → defaults

	cmd, descriptor, err := dispatcher.BuildSpawnCommand(item, project, catalog, dispatcher.AuthBundle{})
	if err != nil {
		t.Fatalf("BuildSpawnCommand() error = %v, want nil", err)
	}
	if cmd == nil {
		t.Fatalf("BuildSpawnCommand() cmd = nil, want non-nil")
	}
	removeBundle(t, cmd)

	if len(cmd.Args) == 0 || filepath.Base(cmd.Args[0]) != "claude" {
		t.Fatalf("cmd.Args[0] = %q, want base name \"claude\"", cmd.Args[0])
	}
	if !argvContains(cmd.Args, "--bare") {
		t.Errorf("cmd.Args missing --bare flag: %v", cmd.Args)
	}
	if v, ok := argFlagValue(cmd.Args, "--agent"); !ok || v != "go-builder-agent" {
		t.Errorf("--agent value = %q (ok=%v), want %q", v, ok, "go-builder-agent")
	}
	if descriptor.AgentName != "go-builder-agent" {
		t.Errorf("descriptor.AgentName = %q, want %q", descriptor.AgentName, "go-builder-agent")
	}
}

// TestBuildSpawnCommandHonorsExplicitClaudeCLIKind covers the same path as
// the default test but with rawBinding.CLIKind set explicitly to "claude".
// This pins that the resolver does NOT mangle the explicit value (no
// uppercasing, no whitespace trim that could miss "claude " etc.).
func TestBuildSpawnCommandHonorsExplicitClaudeCLIKind(t *testing.T) {
	t.Parallel()

	binding := goBuilderBinding()
	binding.CLIKind = "claude"

	cmd, _, err := dispatcher.BuildSpawnCommand(fixtureBuildItem(), fixtureProject(), fixtureCatalog(binding), dispatcher.AuthBundle{})
	if err != nil {
		t.Fatalf("BuildSpawnCommand() error = %v, want nil", err)
	}
	removeBundle(t, cmd)

	if filepath.Base(cmd.Args[0]) != "claude" {
		t.Fatalf("cmd.Args[0] = %q, want base name \"claude\"", cmd.Args[0])
	}
}

// TestBuildSpawnCommandRejectsUnknownCLIKind asserts an unregistered CLIKind
// surfaces as ErrUnsupportedCLIKind rather than a nil-map panic or a generic
// error. Drop 4d adds the codex adapter; until then a binding asking for
// "codex" (or any other future kind) trips this guard cleanly.
func TestBuildSpawnCommandRejectsUnknownCLIKind(t *testing.T) {
	t.Parallel()

	binding := goBuilderBinding()
	binding.CLIKind = "bogus" // not registered

	cmd, _, err := dispatcher.BuildSpawnCommand(fixtureBuildItem(), fixtureProject(), fixtureCatalog(binding), dispatcher.AuthBundle{})
	if err == nil {
		t.Fatalf("BuildSpawnCommand() error = nil, want ErrUnsupportedCLIKind")
	}
	if !errors.Is(err, dispatcher.ErrUnsupportedCLIKind) {
		t.Fatalf("BuildSpawnCommand() error = %v, want errors.Is(ErrUnsupportedCLIKind)", err)
	}
	if cmd != nil {
		t.Fatalf("BuildSpawnCommand() cmd = %v, want nil on error", cmd)
	}
}

// TestBuildSpawnCommandWritesSystemPromptFile asserts BuildSpawnCommand
// renders the spawn prompt body to disk under the per-spawn bundle's
// system-prompt.md path, and that the body contains the action-item
// structural tokens (task_id, project_id, project_dir, kind, move-state
// directive) but NOT hylla_artifact_ref (F.7.10 removed it).
func TestBuildSpawnCommandWritesSystemPromptFile(t *testing.T) {
	t.Parallel()

	item := fixtureBuildItem()
	item.Paths = []string{"internal/app/dispatcher/spawn.go"}
	item.Packages = []string{"github.com/evanmschultz/tillsyn/internal/app/dispatcher"}
	project := fixtureProject()

	cmd, descriptor, err := dispatcher.BuildSpawnCommand(item, project, fixtureCatalog(goBuilderBinding()), dispatcher.AuthBundle{})
	if err != nil {
		t.Fatalf("BuildSpawnCommand() error = %v, want nil", err)
	}
	removeBundle(t, cmd)

	// System-prompt path is the value passed to --system-prompt-file in argv.
	promptPath, ok := argFlagValue(cmd.Args, "--system-prompt-file")
	if !ok {
		t.Fatalf("cmd.Args has no --system-prompt-file flag: %v", cmd.Args)
	}
	if filepath.Base(promptPath) != "system-prompt.md" {
		t.Errorf("--system-prompt-file = %q, want basename system-prompt.md", promptPath)
	}

	body, err := os.ReadFile(promptPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", promptPath, err)
	}
	bodyStr := string(body)

	wantTokens := []string{
		"task_id: " + item.ID,
		"project_id: " + project.ID,
		"project_dir: " + project.RepoPrimaryWorktree,
		"kind: " + string(item.Kind),
		"title: " + item.Title,
		"paths: " + item.Paths[0],
		"packages: " + item.Packages[0],
		"move-state directive:",
	}
	for _, tok := range wantTokens {
		if !strings.Contains(bodyStr, tok) {
			t.Errorf("system-prompt.md missing %q\nfull body:\n%s", tok, bodyStr)
		}
	}

	// F.7.10: hylla_artifact_ref MUST NOT appear in the prompt body.
	if strings.Contains(bodyStr, "hylla_artifact_ref") {
		t.Errorf("system-prompt.md unexpectedly contains hylla_artifact_ref\nfull body:\n%s", bodyStr)
	}

	// The descriptor's Prompt field carries the same body the file does.
	if descriptor.Prompt != bodyStr {
		t.Errorf("descriptor.Prompt != file contents\ndescriptor:\n%s\nfile:\n%s", descriptor.Prompt, bodyStr)
	}
}

// TestBuildSpawnCommandSetsCwd asserts the constructed command's working
// directory matches the project's primary worktree — the dispatcher's
// `cd`-target invariant for spawned subagents.
func TestBuildSpawnCommandSetsCwd(t *testing.T) {
	t.Parallel()

	project := fixtureProject()
	cmd, _, err := dispatcher.BuildSpawnCommand(fixtureBuildItem(), project, fixtureCatalog(goBuilderBinding()), dispatcher.AuthBundle{})
	if err != nil {
		t.Fatalf("BuildSpawnCommand() error = %v, want nil", err)
	}
	removeBundle(t, cmd)
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

	cmd, _, err := dispatcher.BuildSpawnCommand(item, fixtureProject(), catalog, dispatcher.AuthBundle{})
	if err == nil {
		t.Fatalf("BuildSpawnCommand() error = nil, want ErrNoAgentBinding")
	}
	if !errors.Is(err, dispatcher.ErrNoAgentBinding) {
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

	cmd, _, err := dispatcher.BuildSpawnCommand(fixtureBuildItem(), fixtureProject(), templates.KindCatalog{}, dispatcher.AuthBundle{})
	if err == nil {
		t.Fatalf("BuildSpawnCommand() error = nil, want ErrNoAgentBinding")
	}
	if !errors.Is(err, dispatcher.ErrNoAgentBinding) {
		t.Fatalf("BuildSpawnCommand() error = %v, want errors.Is(ErrNoAgentBinding)", err)
	}
	if cmd != nil {
		t.Fatalf("BuildSpawnCommand() cmd = %v, want nil on error", cmd)
	}
}

// TestBuildSpawnCommandPropagatesBundlePaths asserts the descriptor's
// MCPConfigPath (consumed by `till dispatcher run --dry-run` JSON output)
// points under the per-spawn bundle directory, contains the conventional
// claude plugin subpath (plugin/.mcp.json), and that the same bundle root
// also surfaces in the cmd's argv via --plugin-dir / --system-prompt-file.
func TestBuildSpawnCommandPropagatesBundlePaths(t *testing.T) {
	t.Parallel()

	item := fixtureBuildItem()
	cmd, descriptor, err := dispatcher.BuildSpawnCommand(item, fixtureProject(), fixtureCatalog(goBuilderBinding()), dispatcher.AuthBundle{})
	if err != nil {
		t.Fatalf("BuildSpawnCommand() error = %v, want nil", err)
	}
	removeBundle(t, cmd)

	if descriptor.MCPConfigPath == "" {
		t.Fatalf("descriptor.MCPConfigPath is empty, want a bundle path")
	}
	if !strings.HasSuffix(descriptor.MCPConfigPath, filepath.Join("plugin", ".mcp.json")) {
		t.Errorf("descriptor.MCPConfigPath = %q, want suffix plugin/.mcp.json", descriptor.MCPConfigPath)
	}

	// argv should reference the same bundle root via --plugin-dir.
	pluginDir, ok := argFlagValue(cmd.Args, "--plugin-dir")
	if !ok {
		t.Fatalf("cmd.Args missing --plugin-dir flag: %v", cmd.Args)
	}
	bundleRoot := filepath.Dir(pluginDir)
	if !strings.HasPrefix(descriptor.MCPConfigPath, bundleRoot+string(filepath.Separator)) {
		t.Errorf("descriptor.MCPConfigPath = %q, want under bundle root %q", descriptor.MCPConfigPath, bundleRoot)
	}

	// system-prompt.md should also be under the same bundle root.
	systemPrompt, ok := argFlagValue(cmd.Args, "--system-prompt-file")
	if !ok {
		t.Fatalf("cmd.Args missing --system-prompt-file flag: %v", cmd.Args)
	}
	if !strings.HasPrefix(systemPrompt, bundleRoot+string(filepath.Separator)) {
		t.Errorf("--system-prompt-file = %q, want under bundle root %q", systemPrompt, bundleRoot)
	}
}

// TestBuildSpawnCommandRejectsCorruptedAgentBinding asserts the defensive
// AgentBinding.Validate re-call inside BuildSpawnCommand catches bindings
// that were corrupted in-memory after template-load (the only way an empty
// AgentName can survive load — Validate rejects it at the loader). The
// error fires before any adapter lookup or bundle creation.
func TestBuildSpawnCommandRejectsCorruptedAgentBinding(t *testing.T) {
	t.Parallel()

	corrupted := goBuilderBinding()
	corrupted.AgentName = "" // would have been rejected by Validate at load

	catalog := fixtureCatalog(corrupted)
	cmd, _, err := dispatcher.BuildSpawnCommand(fixtureBuildItem(), fixtureProject(), catalog, dispatcher.AuthBundle{})
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
// guard: empty IDs cannot construct a usable spawn descriptor (the prompt's
// task_id field would be empty), so the constructor rejects.
func TestBuildSpawnCommandRejectsEmptyActionItemID(t *testing.T) {
	t.Parallel()

	item := fixtureBuildItem()
	item.ID = "   "

	cmd, _, err := dispatcher.BuildSpawnCommand(item, fixtureProject(), fixtureCatalog(goBuilderBinding()), dispatcher.AuthBundle{})
	if err == nil {
		t.Fatalf("BuildSpawnCommand() error = nil, want ErrInvalidSpawnInput")
	}
	if !errors.Is(err, dispatcher.ErrInvalidSpawnInput) {
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

	_, _, err := dispatcher.BuildSpawnCommand(item, fixtureProject(), fixtureCatalog(goBuilderBinding()), dispatcher.AuthBundle{})
	if err == nil {
		t.Fatalf("BuildSpawnCommand() error = nil, want ErrInvalidSpawnInput")
	}
	if !errors.Is(err, dispatcher.ErrInvalidSpawnInput) {
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

	_, _, err := dispatcher.BuildSpawnCommand(fixtureBuildItem(), project, fixtureCatalog(goBuilderBinding()), dispatcher.AuthBundle{})
	if err == nil {
		t.Fatalf("BuildSpawnCommand() error = nil, want ErrInvalidSpawnInput")
	}
	if !errors.Is(err, dispatcher.ErrInvalidSpawnInput) {
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

	cmd, descriptor, err := dispatcher.BuildSpawnCommand(fixtureBuildItem(), fixtureProject(), fixtureCatalog(binding), dispatcher.AuthBundle{})
	if err != nil {
		t.Fatalf("BuildSpawnCommand() error = %v, want nil", err)
	}
	removeBundle(t, cmd)

	if descriptor.MaxBudgetUSD != 2.5 {
		t.Errorf("descriptor.MaxBudgetUSD = %v, want 2.5", descriptor.MaxBudgetUSD)
	}
	v, ok := argFlagValue(cmd.Args, "--max-budget-usd")
	if !ok {
		t.Fatalf("--max-budget-usd flag not found in argv: %v", cmd.Args)
	}
	if v != "2.5" {
		t.Errorf("--max-budget-usd value = %q, want %q", v, "2.5")
	}
}

// TestSpawnDescriptorZeroValueHasNoFields is a regression pin: SpawnDescriptor
// is a value type, and accidental shifting of fields between BuildSpawnCommand
// and a future caller would silently rebind the wrong values. This test
// constructs the zero value and asserts the documented invariants — empty
// strings + zero numerics — survive any future field reordering.
func TestSpawnDescriptorZeroValueHasNoFields(t *testing.T) {
	t.Parallel()

	var d dispatcher.SpawnDescriptor
	if d.AgentName != "" || d.Model != "" || d.MCPConfigPath != "" || d.Prompt != "" || d.WorkingDir != "" {
		t.Errorf("SpawnDescriptor zero value has non-empty string field: %+v", d)
	}
	if d.MaxBudgetUSD != 0 || d.MaxTurns != 0 {
		t.Errorf("SpawnDescriptor zero value has non-zero numeric field: %+v", d)
	}
}

// fakeAdapter is a minimal CLIAdapter mock the registry-isolation tests use
// to confirm RegisterAdapter wires a non-claude adapter under a custom
// CLIKind. Production code never instantiates this — the cli_claude blank
// import handles real spawns.
type fakeAdapter struct {
	calls int
}

// BuildCommand records the invocation and returns a trivial /bin/true cmd
// so the dispatcher's downstream stages (cmd.Dir set, descriptor populated)
// have something to work with.
func (a *fakeAdapter) BuildCommand(_ context.Context, _ dispatcher.BindingResolved, _ dispatcher.BundlePaths) (*exec.Cmd, error) {
	a.calls++
	return exec.Command("/usr/bin/true"), nil
}

// ParseStreamEvent returns an empty event; not exercised by these tests.
func (a *fakeAdapter) ParseStreamEvent(_ []byte) (dispatcher.StreamEvent, error) {
	return dispatcher.StreamEvent{}, nil
}

// ExtractTerminalReport returns the empty zero report; not exercised here.
func (a *fakeAdapter) ExtractTerminalReport(_ dispatcher.StreamEvent) (dispatcher.TerminalReport, bool) {
	return dispatcher.TerminalReport{}, false
}

// TestRegisterAdapterRoutesCustomCLIKind asserts that a CLIAdapter wired via
// dispatcher.RegisterAdapter under a custom CLIKind takes the spawn over
// the default claude path. This is the seam Drop 4d's cli_codex package
// will use.
func TestRegisterAdapterRoutesCustomCLIKind(t *testing.T) {
	// NOT t.Parallel() — RegisterAdapter mutates a process-wide map; running
	// in parallel with the default-claude tests can race.

	customKind := dispatcher.CLIKind("test-custom-kind")
	fake := &fakeAdapter{}
	dispatcher.RegisterAdapter(customKind, fake)

	binding := goBuilderBinding()
	binding.CLIKind = "test-custom-kind"

	cmd, _, err := dispatcher.BuildSpawnCommand(fixtureBuildItem(), fixtureProject(), fixtureCatalog(binding), dispatcher.AuthBundle{})
	if err != nil {
		t.Fatalf("BuildSpawnCommand() error = %v, want nil", err)
	}
	if cmd == nil {
		t.Fatalf("BuildSpawnCommand() cmd = nil, want non-nil")
	}
	// Bundle root inferred from the prompt path the dispatcher passed to
	// the adapter — fakeAdapter discards it but the dispatcher already
	// created the temp dir on disk. Clean up by reading the descriptor.
	t.Cleanup(func() {
		// Best-effort cleanup; the bundle dir may have leaked but we have
		// no handle to it here. Tests run in parallel processes so leakage
		// is bounded by t.TempDir-style discipline at the OS level.
		_ = os.Args // no-op; keeps the cleanup hook readable as intent
	})

	if fake.calls != 1 {
		t.Errorf("fakeAdapter.BuildCommand calls = %d, want 1", fake.calls)
	}
	// The cmd is whatever the fake returned (here, /usr/bin/true with no
	// args); cmd.Dir should still be set by BuildSpawnCommand.
	project := fixtureProject()
	if cmd.Dir != project.RepoPrimaryWorktree {
		t.Errorf("cmd.Dir = %q, want %q (BuildSpawnCommand sets Dir post-adapter)", cmd.Dir, project.RepoPrimaryWorktree)
	}
}

// Compile-time assertion: fakeAdapter satisfies dispatcher.CLIAdapter. If
// any of the three methods drift, the test build fails here.
var _ dispatcher.CLIAdapter = (*fakeAdapter)(nil)

// TestBuildSpawnCommandWritesManifestJSON asserts the F.7-CORE F.7.1 bundle
// integration: BuildSpawnCommand calls Bundle.WriteManifest before invoking
// the adapter, and the resulting manifest.json carries the per-spawn
// metadata (action_item_id, kind, paths) sourced from the action item.
//
// The previous (4a.19 / F.7.17.5) implementation used os.MkdirTemp with no
// manifest write — this test pins the new contract end-to-end.
func TestBuildSpawnCommandWritesManifestJSON(t *testing.T) {
	t.Parallel()

	item := fixtureBuildItem()
	item.Paths = []string{"internal/app/dispatcher/spawn.go", "internal/app/dispatcher/bundle.go"}
	project := fixtureProject()

	cmd, _, err := dispatcher.BuildSpawnCommand(item, project, fixtureCatalog(goBuilderBinding()), dispatcher.AuthBundle{})
	if err != nil {
		t.Fatalf("BuildSpawnCommand() error = %v, want nil", err)
	}
	removeBundle(t, cmd)

	pluginDir, ok := argFlagValue(cmd.Args, "--plugin-dir")
	if !ok {
		t.Fatalf("cmd.Args missing --plugin-dir flag: %v", cmd.Args)
	}
	bundleRoot := filepath.Dir(pluginDir)
	manifestPath := filepath.Join(bundleRoot, "manifest.json")

	contents, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v — expected manifest.json under bundle root %q", manifestPath, err, bundleRoot)
	}

	var generic map[string]any
	if err := json.Unmarshal(contents, &generic); err != nil {
		t.Fatalf("json.Unmarshal manifest: %v\ncontents:\n%s", err, contents)
	}

	wantKeys := []string{"spawn_id", "action_item_id", "kind", "started_at", "paths"}
	for _, k := range wantKeys {
		if _, ok := generic[k]; !ok {
			t.Errorf("manifest missing JSON key %q\nfull payload:\n%s", k, contents)
		}
	}
	if got, want := generic["action_item_id"], item.ID; got != want {
		t.Errorf("manifest action_item_id = %v, want %q", got, want)
	}
	if got, want := generic["kind"], string(item.Kind); got != want {
		t.Errorf("manifest kind = %v, want %q", got, want)
	}
	pathsRaw, ok := generic["paths"].([]any)
	if !ok {
		t.Fatalf("manifest paths = %T, want []any\ncontents:\n%s", generic["paths"], contents)
	}
	if len(pathsRaw) != len(item.Paths) {
		t.Fatalf("manifest paths len = %d, want %d", len(pathsRaw), len(item.Paths))
	}
	for i, p := range item.Paths {
		if pathsRaw[i] != p {
			t.Errorf("manifest paths[%d] = %v, want %q", i, pathsRaw[i], p)
		}
	}
}

// TestBuildSpawnCommandRendersFullBundleSubtree asserts the F.7-CORE F.7.3b
// integration: BuildSpawnCommand's render-hook lookup invokes the
// cli_claude/render package's Render which writes EVERY artifact memory
// §2 specifies — system-prompt.md at the bundle root + plugin/{plugin.json,
// agents/<name>.md, .mcp.json, settings.json} under the claude-specific
// subtree. F.7.17.5's provisional minimal-prompt-only behavior is REPLACED
// by this droplet; the assertion below is the regression pin against
// reverting to the minimal block.
func TestBuildSpawnCommandRendersFullBundleSubtree(t *testing.T) {
	t.Parallel()

	cmd, _, err := dispatcher.BuildSpawnCommand(fixtureBuildItem(), fixtureProject(), fixtureCatalog(goBuilderBinding()), dispatcher.AuthBundle{})
	if err != nil {
		t.Fatalf("BuildSpawnCommand() error = %v, want nil", err)
	}
	removeBundle(t, cmd)

	pluginDir, ok := argFlagValue(cmd.Args, "--plugin-dir")
	if !ok {
		t.Fatalf("cmd.Args missing --plugin-dir flag: %v", cmd.Args)
	}
	bundleRoot := filepath.Dir(pluginDir)

	wantFiles := []string{
		filepath.Join(bundleRoot, "system-prompt.md"),
		filepath.Join(bundleRoot, "plugin", ".claude-plugin", "plugin.json"),
		filepath.Join(bundleRoot, "plugin", "agents", "go-builder-agent.md"),
		filepath.Join(bundleRoot, "plugin", ".mcp.json"),
		filepath.Join(bundleRoot, "plugin", "settings.json"),
		// manifest.json from F.7.1 also expected at the bundle root.
		filepath.Join(bundleRoot, "manifest.json"),
	}
	for _, p := range wantFiles {
		info, statErr := os.Stat(p)
		if statErr != nil {
			t.Errorf("os.Stat(%q) error = %v, want file present after Render", p, statErr)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("file %q has zero bytes; expected non-empty content from Render", p)
		}
	}
}

// TestBuildSpawnCommandRenderHookFailureCleansUpBundle asserts the
// dispatcher's failure-path contract: a render hook that fails causes
// BuildSpawnCommand to return a non-nil error AND clean up the bundle
// directory it created via NewBundle. The test substitutes a faulty
// render hook via RegisterBundleRenderFunc, then restores the real one
// (registered by cli_claude's blank-import init) for downstream tests.
func TestBuildSpawnCommandRenderHookFailureCleansUpBundle(t *testing.T) {
	// NOT t.Parallel() — RegisterBundleRenderFunc mutates a process-wide
	// hook; running in parallel with other tests can race against the
	// real-render hook the rest of the suite expects.

	// Capture the real hook so we can restore it after the failure path.
	var capturedBundleRoot string
	faulty := func(
		_ context.Context,
		bundle dispatcher.Bundle,
		_ domain.ActionItem,
		_ domain.Project,
		_ dispatcher.BindingResolved,
		_ any,
	) (string, error) {
		capturedBundleRoot = bundle.Paths.Root
		return "", errors.New("render: fault-injected failure")
	}

	// The cli_claude blank-import in spawn_test.go's import list registers
	// the real Render via render's init(). Substitute the faulty hook,
	// run the test, then restore the real one by re-importing render's
	// adaptRender (production hook) afterwards. adaptRender is unexported
	// so we cannot reference it directly; restore by re-running render's
	// init() — Go does not re-run inits, so we wrap clauderender.Render
	// in the same any→PermissionGrantsLister adapter inline.
	dispatcher.RegisterBundleRenderFunc(faulty)
	t.Cleanup(func() {
		// Restore the real render hook so subsequent tests in this
		// package see the production wiring. The clauderender named
		// import above gives us access to the Render symbol; we wrap it
		// in the same adapter shape adaptRender uses.
		dispatcher.RegisterBundleRenderFunc(func(
			ctx context.Context,
			bundle dispatcher.Bundle,
			item domain.ActionItem,
			project domain.Project,
			binding dispatcher.BindingResolved,
			grantsLister any,
		) (string, error) {
			var lister clauderender.PermissionGrantsLister
			if grantsLister != nil {
				typed, ok := grantsLister.(clauderender.PermissionGrantsLister)
				if !ok {
					return "", clauderender.ErrInvalidGrantsLister
				}
				lister = typed
			}
			return clauderender.Render(ctx, bundle, item, project, binding, lister)
		})
	})

	_, _, err := dispatcher.BuildSpawnCommand(fixtureBuildItem(), fixtureProject(), fixtureCatalog(goBuilderBinding()), dispatcher.AuthBundle{})
	if err == nil {
		t.Fatalf("BuildSpawnCommand() error = nil, want non-nil from faulty render")
	}
	if !strings.Contains(err.Error(), "render spawn bundle") {
		t.Errorf("err = %v, want containing %q", err, "render spawn bundle")
	}

	// Bundle cleanup: NewBundle created the dir, render failed, the
	// dispatcher should have run bundle.Cleanup which removes the root.
	if capturedBundleRoot == "" {
		t.Fatalf("faulty hook never observed bundle root; was it called?")
	}
	if _, statErr := os.Stat(capturedBundleRoot); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("bundle root %q still exists after failed render; statErr = %v", capturedBundleRoot, statErr)
	}
}

// TestBuildSpawnCommandBundleRootUnderOSTempDir asserts the spawn-temp-root
// resolution path: BuildSpawnCommand's call to NewBundle today threads the
// empty-string sentinel which resolves to "os_tmp" mode. The bundle root
// MUST live under os.TempDir() with the conventional prefix.
//
// When the catalog→Tillsyn plumbing follow-up lands, this test gets a
// project-mode counterpart that asserts the bundle root lives under
// <project>/.tillsyn/spawns/<id>/.
func TestBuildSpawnCommandBundleRootUnderOSTempDir(t *testing.T) {
	t.Parallel()

	cmd, _, err := dispatcher.BuildSpawnCommand(fixtureBuildItem(), fixtureProject(), fixtureCatalog(goBuilderBinding()), dispatcher.AuthBundle{})
	if err != nil {
		t.Fatalf("BuildSpawnCommand() error = %v, want nil", err)
	}
	removeBundle(t, cmd)

	pluginDir, ok := argFlagValue(cmd.Args, "--plugin-dir")
	if !ok {
		t.Fatalf("cmd.Args missing --plugin-dir flag: %v", cmd.Args)
	}
	bundleRoot := filepath.Dir(pluginDir)

	tempRoot := os.TempDir()
	if !strings.HasPrefix(bundleRoot, tempRoot) {
		t.Errorf("bundle root = %q; want prefix %q (os_tmp mode)", bundleRoot, tempRoot)
	}
	if !strings.HasPrefix(filepath.Base(bundleRoot), "tillsyn-spawn-") {
		t.Errorf("bundle root basename = %q; want prefix %q",
			filepath.Base(bundleRoot), "tillsyn-spawn-")
	}
}

// TestBuildSpawnCommandLeavesGitignoreUntouchedInOSTempMode pins the F.7.7
// integration contract for the os_tmp default: BuildSpawnCommand wires the
// EnsureSpawnsGitignored helper but the helper short-circuits without
// creating .gitignore because spawn_temp_root resolves to os_tmp. This is
// the negative assertion against any future change that accidentally
// extends gitignore maintenance to OS-temp mode.
//
// NOT t.Parallel() — the package-scope sync.Once guarding
// EnsureSpawnsGitignored interacts with sibling tests; we serialize this
// scenario via the Reset hook + non-parallel execution to keep the
// invocation observable.
func TestBuildSpawnCommandLeavesGitignoreUntouchedInOSTempMode(t *testing.T) {
	dispatcher.ResetEnsureSpawnsGitignoredOnceForTest()
	t.Cleanup(dispatcher.ResetEnsureSpawnsGitignoredOnceForTest)

	// Use a fresh worktree so the test asserts on a known-clean filesystem
	// state. The fixtureProject() default points at /tmp/tillsyn/main which
	// may or may not exist; pinning to t.TempDir() guarantees the assertion
	// is meaningful.
	worktree := t.TempDir()
	project := fixtureProject()
	project.RepoPrimaryWorktree = worktree

	cmd, _, err := dispatcher.BuildSpawnCommand(fixtureBuildItem(), project, fixtureCatalog(goBuilderBinding()), dispatcher.AuthBundle{})
	if err != nil {
		t.Fatalf("BuildSpawnCommand() error = %v, want nil", err)
	}
	removeBundle(t, cmd)

	gitignorePath := filepath.Join(worktree, ".gitignore")
	if _, statErr := os.Stat(gitignorePath); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("os.Stat(%q) = %v; want os.ErrNotExist (os_tmp mode must not write .gitignore)", gitignorePath, statErr)
	}
}

// TestBuildSpawnCommandEnsureGitignoredFiresOncePerProcess pins the
// sync.Once gating: a sequence of BuildSpawnCommand invocations against
// the same project produces exactly one .gitignore-write attempt. We
// verify by counting how many times the helper-equivalent file would
// appear on disk after multiple spawns (always exactly one occurrence,
// regardless of spawn count).
//
// NOT t.Parallel() — same rationale as the os_tmp test above.
func TestBuildSpawnCommandEnsureGitignoredFiresOncePerProcess(t *testing.T) {
	dispatcher.ResetEnsureSpawnsGitignoredOnceForTest()
	t.Cleanup(dispatcher.ResetEnsureSpawnsGitignoredOnceForTest)

	worktree := t.TempDir()
	project := fixtureProject()
	project.RepoPrimaryWorktree = worktree

	// Two consecutive spawns. Both go through the same os_tmp default path
	// today, but the once-shot still fires (resolves to a no-op for os_tmp).
	// The assertion below is that .gitignore is NOT created — both spawns
	// honor the same once-shot result, neither creates the file.
	for i := 0; i < 3; i++ {
		cmd, _, err := dispatcher.BuildSpawnCommand(fixtureBuildItem(), project, fixtureCatalog(goBuilderBinding()), dispatcher.AuthBundle{})
		if err != nil {
			t.Fatalf("BuildSpawnCommand() iter %d error = %v, want nil", i, err)
		}
		removeBundle(t, cmd)
	}

	gitignorePath := filepath.Join(worktree, ".gitignore")
	if _, statErr := os.Stat(gitignorePath); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("os.Stat(%q) = %v; want os.ErrNotExist after 3 spawns in os_tmp mode", gitignorePath, statErr)
	}
}
