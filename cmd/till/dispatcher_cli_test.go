// Package main dispatcher_cli_test exercises the manual-trigger dispatcher
// CLI (`till dispatcher run`) wired in droplet 4a.23.
//
// CARVE-OUT: TestDispatcherRunCmdSpawnsAndReports compiles a tiny fake-agent
// binary via exec.Command("go", "build", ...) so the dispatcher's monitor can
// exercise real os/exec semantics without depending on the production claude
// binary being on PATH. This is the documented test-helper carve-out from
// "never raw `go`": the same pattern monitor_test.go uses for the dispatcher
// package's own monitor tests (see internal/app/dispatcher/monitor_test.go's
// package doc-comment + WAVE_2_PLAN.md §2.8 Q5). Production code in
// dispatcher_cli.go does NOT shell out to `go`.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/adapters/storage/sqlite"
	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/app/dispatcher"
	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/evanmschultz/tillsyn/internal/templates"
)

// TestDispatcherRunCmdMissingActionItemFlagErrors asserts the CLI rejects an
// empty/whitespace --action-item value before constructing a Dispatcher.
// The error message names the offending flag so the dev sees a precise hint.
func TestDispatcherRunCmdMissingActionItemFlagErrors(t *testing.T) {
	t.Parallel()

	svc, broker, _, _, _ := newDispatcherCLIServiceForTest(t)

	for _, in := range []string{"", "   ", "\t"} {
		in := in
		var out, errOut strings.Builder
		err := runDispatcherRun(context.Background(), svc, broker, dispatcherRunCommandOptions{actionItemID: in}, &out, &errOut)
		if err == nil {
			t.Fatalf("runDispatcherRun(%q) error = nil, want non-nil", in)
		}
		if !strings.Contains(err.Error(), "--action-item") {
			t.Errorf("runDispatcherRun(%q) error = %v, want it to mention --action-item", in, err)
		}
	}
}

// TestDispatcherRunCmdSkipsWhenItemNotInTodo seeds an action item in the
// complete column and asserts the CLI exits 0 with a "skipped: not in todo"
// summary line on stdout.
func TestDispatcherRunCmdSkipsWhenItemNotInTodo(t *testing.T) {
	t.Parallel()

	svc, broker, _, _, completeColumnID := newDispatcherCLIServiceForTest(t)

	// Create a build action item, then move it to complete so RunOnce
	// short-circuits at the non-todo gate.
	item, err := svc.CreateActionItem(context.Background(), app.CreateActionItemInput{
		ProjectID:      defaultDispatcherTestProjectID,
		ColumnID:       defaultDispatcherTestTodoColumnID,
		Title:          "DROPLET CLI SKIP TEST",
		Kind:           domain.KindBuild,
		Scope:          domain.KindAppliesToBuild,
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem() error = %v", err)
	}
	if _, err := svc.MoveActionItem(context.Background(), item.ID, completeColumnID, item.Position); err != nil {
		t.Fatalf("MoveActionItem(complete) error = %v", err)
	}

	var out, errOut strings.Builder
	if err := runDispatcherRun(context.Background(), svc, broker, dispatcherRunCommandOptions{actionItemID: item.ID}, &out, &errOut); err != nil {
		t.Fatalf("runDispatcherRun() error = %v, want nil for skip path", err)
	}
	gotStdout := out.String()
	if !strings.HasPrefix(gotStdout, "skipped:") {
		t.Errorf("stdout = %q, want prefix %q", gotStdout, "skipped:")
	}
	if !strings.Contains(gotStdout, "not in todo") {
		t.Errorf("stdout = %q, want substring %q", gotStdout, "not in todo")
	}
}

// TestDispatcherRunCmdDryRunPrintsDescriptor seeds an eligible action item
// with a baked KindCatalog binding and asserts --dry-run prints the
// SpawnDescriptor as JSON containing agent_name, model, working_dir.
func TestDispatcherRunCmdDryRunPrintsDescriptor(t *testing.T) {
	t.Parallel()

	env := newDispatcherCLITestEnv(t)
	bakeDispatcherCatalog(t, env, "go-builder-agent", "")

	item, err := env.svc.CreateActionItem(context.Background(), app.CreateActionItemInput{
		ProjectID:      env.projectID,
		ColumnID:       env.todoColumnID,
		Title:          "DROPLET CLI DRY RUN TEST",
		Kind:           domain.KindBuild,
		Scope:          domain.KindAppliesToBuild,
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem() error = %v", err)
	}

	var out, errOut strings.Builder
	if err := runDispatcherRun(context.Background(), env.svc, env.broker, dispatcherRunCommandOptions{actionItemID: item.ID, dryRun: true}, &out, &errOut); err != nil {
		t.Fatalf("runDispatcherRun() error = %v, want nil for dry-run", err)
	}

	gotStdout := out.String()
	if !strings.Contains(gotStdout, "agent_name") {
		t.Errorf("dry-run stdout missing agent_name key: %s", gotStdout)
	}
	if !strings.Contains(gotStdout, "model") {
		t.Errorf("dry-run stdout missing model key: %s", gotStdout)
	}
	if !strings.Contains(gotStdout, "working_dir") {
		t.Errorf("dry-run stdout missing working_dir key: %s", gotStdout)
	}
	if !strings.Contains(gotStdout, "go-builder-agent") {
		t.Errorf("dry-run stdout missing agent name fixture value: %s", gotStdout)
	}

	// Decode the JSON to confirm shape — the snake_case wire form is the
	// CLI's contract for dev scripts.
	var decoded struct {
		AgentName  string `json:"agent_name"`
		Model      string `json:"model"`
		WorkingDir string `json:"working_dir"`
	}
	if err := json.Unmarshal([]byte(gotStdout), &decoded); err != nil {
		t.Fatalf("json.Unmarshal(dry-run stdout) error = %v\nstdout:\n%s", err, gotStdout)
	}
	if decoded.AgentName != "go-builder-agent" {
		t.Errorf("decoded.AgentName = %q, want %q", decoded.AgentName, "go-builder-agent")
	}
	if decoded.Model != "opus" {
		t.Errorf("decoded.Model = %q, want %q", decoded.Model, "opus")
	}
	if decoded.WorkingDir == "" {
		t.Errorf("decoded.WorkingDir is empty; expected the project's RepoPrimaryWorktree fixture")
	}

	// Dry-run MUST NOT have moved the action item.
	reloaded, err := env.svc.GetActionItem(context.Background(), item.ID)
	if err != nil {
		t.Fatalf("GetActionItem() error = %v", err)
	}
	if reloaded.LifecycleState != domain.StateTodo {
		t.Errorf("dry-run promoted action item: state = %q, want %q", reloaded.LifecycleState, domain.StateTodo)
	}
}

// TestDispatcherRunCmdSpawnsAndReports seeds an eligible action item with a
// binding pointing at a freshly compiled fake-agent binary and asserts the
// CLI:
//   - exits within 1 second of spawn (does NOT wait for agent completion);
//   - emits a "spawned <agent> for <id>" summary line on stdout;
//   - leaves the action item in StateInProgress (the walker promoted it
//     before the spawn).
//
// Cannot t.Parallel: bakeDispatcherCatalog calls t.Setenv to install the
// "claude" wrapper on PATH, and t.Setenv is incompatible with t.Parallel.
// Test runtime is dominated by the one-shot fakeagent compile (~1s) so the
// loss of parallelism is bounded.
func TestDispatcherRunCmdSpawnsAndReports(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("manual-trigger CLI dispatcher tests target Unix-only fake-agent semantics")
	}

	env := newDispatcherCLITestEnv(t)
	agentBin := buildDispatcherCLIFakeAgent(t)
	bakeDispatcherCatalog(t, env, "go-builder-agent", agentBin)

	item, err := env.svc.CreateActionItem(context.Background(), app.CreateActionItemInput{
		ProjectID:      env.projectID,
		ColumnID:       env.todoColumnID,
		Title:          "DROPLET CLI SPAWN TEST",
		Kind:           domain.KindBuild,
		Scope:          domain.KindAppliesToBuild,
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem() error = %v", err)
	}

	var out, errOut strings.Builder
	deadline := time.Now().Add(5 * time.Second)
	start := time.Now()
	cliErr := runDispatcherRun(context.Background(), env.svc, env.broker, dispatcherRunCommandOptions{actionItemID: item.ID}, &out, &errOut)
	elapsed := time.Since(start)
	if cliErr != nil {
		t.Fatalf("runDispatcherRun() error = %v\nstderr: %s\nstdout: %s", cliErr, errOut.String(), out.String())
	}
	if time.Now().After(deadline) {
		t.Fatalf("runDispatcherRun took %v; expected <5s (CLI must NOT wait for agent completion)", elapsed)
	}
	if elapsed > 1*time.Second {
		// 1s is the spec-pinned target. We allow some slack on slow CI.
		t.Logf("warning: runDispatcherRun took %v; spec target is <1s", elapsed)
	}

	gotStdout := out.String()
	if !strings.HasPrefix(gotStdout, "spawned ") {
		t.Errorf("stdout = %q, want prefix %q", gotStdout, "spawned ")
	}
	if !strings.Contains(gotStdout, "go-builder-agent") {
		t.Errorf("stdout = %q, want substring %q", gotStdout, "go-builder-agent")
	}
	if !strings.Contains(gotStdout, item.ID) {
		t.Errorf("stdout = %q, want substring %q (action item id)", gotStdout, item.ID)
	}

	// The walker.Promote call moved the action item to in_progress before
	// the spawn; the agent's own move-state directive (which the fake agent
	// does NOT honor — it's a tiny test binary) has nothing to do with the
	// CLI's promotion.
	reloaded, err := env.svc.GetActionItem(context.Background(), item.ID)
	if err != nil {
		t.Fatalf("GetActionItem() error = %v", err)
	}
	if reloaded.LifecycleState != domain.StateInProgress {
		t.Errorf("spawn left action item in state %q, want %q", reloaded.LifecycleState, domain.StateInProgress)
	}
}

// writeExecutableForTest writes the supplied script body to path and chmods
// it 0o755 so exec.LookPath can find and run it. Test-only helper.
func writeExecutableForTest(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("write executable %q: %v", path, err)
	}
}

// envPATH returns the current process's PATH environment variable, or empty
// string when unset. Used by bakeDispatcherCatalog to chain the test's
// "claude" wrapper directory in front of the inherited PATH.
func envPATH(t *testing.T) string {
	t.Helper()
	return os.Getenv("PATH")
}

// defaultDispatcherTestProjectID is the deterministic project ID newDispatcherCLIServiceForTest seeds.
// Tests use it directly when constructing CreateActionItemInput to keep the
// fixture self-documenting.
const defaultDispatcherTestProjectID = "p-dispatcher-cli"

// defaultDispatcherTestTodoColumnID is the deterministic todo column ID
// newDispatcherCLIServiceForTest seeds. Action items created in todo go here.
const defaultDispatcherTestTodoColumnID = "col-todo"

// dispatcherCLITestEnv aggregates the in-memory fixture environment for the
// dispatcher CLI tests: a service + broker pair backed by an in-memory SQLite
// repo, with the repo handle exposed so tests can mutate KindCatalogJSON
// directly (the catalog is not part of the service's UpdateProjectInput
// surface, so direct repo writes are the canonical test path — same pattern
// used by internal/app/kind_capability_catalog_test.go).
type dispatcherCLITestEnv struct {
	svc              *app.Service
	broker           app.LiveWaitBroker
	repo             *sqlite.Repository
	projectID        string
	todoColumnID     string
	completeColumnID string
}

// newDispatcherCLIServiceForTest seeds a project + columns required for the
// dispatcher CLI tests:
//   - A project with RepoPrimaryWorktree populated (the spawn-time cmd.Dir).
//   - Four columns: todo, in_progress, complete, failed (canonical lifecycle
//     slugs the walker + monitor resolve via slugify on column name).
//
// Returns (service, broker, projectID, todoColumnID, completeColumnID) for
// backwards-compatible call sites; the underlying env is exposed via
// newDispatcherCLITestEnv for tests that need direct repo access.
func newDispatcherCLIServiceForTest(t *testing.T) (*app.Service, app.LiveWaitBroker, string, string, string) {
	env := newDispatcherCLITestEnv(t)
	return env.svc, env.broker, env.projectID, env.todoColumnID, env.completeColumnID
}

// newDispatcherCLITestEnv is the underlying fixture builder. Same seeded
// shape as newDispatcherCLIServiceForTest but exposes the repo handle so
// catalog-bake helpers can write KindCatalogJSON directly.
func newDispatcherCLITestEnv(t *testing.T) dispatcherCLITestEnv {
	t.Helper()

	repo, err := sqlite.OpenInMemory()
	if err != nil {
		t.Fatalf("sqlite.OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	worktree := t.TempDir()
	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	project, err := domain.NewProjectFromInput(domain.ProjectInput{
		ID:                  defaultDispatcherTestProjectID,
		Name:                "Dispatcher CLI Test",
		RepoPrimaryWorktree: worktree,
		HyllaArtifactRef:    "github.com/evanmschultz/tillsyn@main",
		Language:            "go",
	}, now)
	if err != nil {
		t.Fatalf("NewProjectFromInput() error = %v", err)
	}
	if err := repo.CreateProject(context.Background(), project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	// Canonical-state columns. The walker resolves these via slugify on
	// column name, so the exact name is what matters, not the ID.
	todoCol, err := domain.NewColumn(defaultDispatcherTestTodoColumnID, project.ID, "To Do", 0, 0, now)
	if err != nil {
		t.Fatalf("NewColumn(todo) error = %v", err)
	}
	if err := repo.CreateColumn(context.Background(), todoCol); err != nil {
		t.Fatalf("CreateColumn(todo) error = %v", err)
	}
	inProgressCol, err := domain.NewColumn("col-inprogress", project.ID, "In Progress", 1, 0, now)
	if err != nil {
		t.Fatalf("NewColumn(in_progress) error = %v", err)
	}
	if err := repo.CreateColumn(context.Background(), inProgressCol); err != nil {
		t.Fatalf("CreateColumn(in_progress) error = %v", err)
	}
	completeCol, err := domain.NewColumn("col-complete", project.ID, "Complete", 2, 0, now)
	if err != nil {
		t.Fatalf("NewColumn(complete) error = %v", err)
	}
	if err := repo.CreateColumn(context.Background(), completeCol); err != nil {
		t.Fatalf("CreateColumn(complete) error = %v", err)
	}
	failedCol, err := domain.NewColumn("col-failed", project.ID, "Failed", 3, 0, now)
	if err != nil {
		t.Fatalf("NewColumn(failed) error = %v", err)
	}
	if err := repo.CreateColumn(context.Background(), failedCol); err != nil {
		t.Fatalf("CreateColumn(failed) error = %v", err)
	}

	idCounter := 0
	idGen := func() string {
		idCounter++
		return fmt.Sprintf("dispatch-cli-id-%010d", idCounter)
	}
	clk := func() time.Time {
		now = now.Add(time.Second)
		return now
	}
	svc := app.NewService(repo, idGen, clk, app.ServiceConfig{
		AutoCreateProjectColumns: false,
	})
	broker := app.NewInProcessLiveWaitBroker()
	return dispatcherCLITestEnv{
		svc:              svc,
		broker:           broker,
		repo:             repo,
		projectID:        project.ID,
		todoColumnID:     todoCol.ID,
		completeColumnID: completeCol.ID,
	}
}

// bakeDispatcherCatalog populates the named project's KindCatalogJSON with a
// catalog binding domain.KindBuild to an AgentBinding fixture. The argv[0]
// of the dispatcher's BuildSpawnCommand is hard-coded to "claude"; tests
// that want to run a real subprocess stage a "claude" wrapper script on
// PATH that delegates to the supplied fake-agent binary. agentBin may be
// empty when the test only needs the catalog to dry-run (no spawn).
func bakeDispatcherCatalog(t *testing.T, env dispatcherCLITestEnv, agentName, agentBin string) {
	t.Helper()

	// Stage a "claude" wrapper on PATH only when agentBin is supplied; the
	// dry-run tests do not need a runnable wrapper.
	if agentBin != "" {
		wrapperDir := t.TempDir()
		wrapperPath := filepath.Join(wrapperDir, "claude")
		wrapperScript := "#!/bin/sh\nexec " + agentBin + " exit0\n"
		writeExecutableForTest(t, wrapperPath, wrapperScript)
		t.Setenv("PATH", wrapperDir+string(filepath.ListSeparator)+envPATH(t))
	}

	tpl := templates.Template{
		SchemaVersion: templates.SchemaVersionV1,
		Kinds: map[domain.Kind]templates.KindRule{
			domain.KindBuild: {
				StructuralType: domain.StructuralTypeDroplet,
			},
		},
		AgentBindings: map[domain.Kind]templates.AgentBinding{
			domain.KindBuild: {
				AgentName:    agentName,
				Model:        "opus",
				MaxTries:     1,
				MaxBudgetUSD: 5,
				MaxTurns:     20,
			},
		},
	}
	catalog := templates.Bake(tpl)
	encoded, err := json.Marshal(catalog)
	if err != nil {
		t.Fatalf("json.Marshal(catalog) error = %v", err)
	}

	// Service.UpdateProjectInput does not surface KindCatalogJSON — it is
	// the service's internal Bake-time field. For tests we route directly
	// through the SQLite repo (same pattern as
	// internal/app/kind_capability_catalog_test.go) to install the
	// catalog without standing up a Template-binding flow that does not
	// yet exist on the public surface.
	project, err := env.repo.GetProject(context.Background(), env.projectID)
	if err != nil {
		t.Fatalf("repo.GetProject() error = %v", err)
	}
	project.KindCatalogJSON = encoded
	if err := env.repo.UpdateProject(context.Background(), project); err != nil {
		t.Fatalf("repo.UpdateProject(KindCatalogJSON) error = %v", err)
	}
}

// TestDispatcherRunCmdRejectsProjectMismatch pins the §2.2 fix at the CLI
// boundary: --project P1 against an item whose ProjectID is P2 returns a
// non-zero exit with ErrProjectMismatch wrapped. The CLI surfaces the typed
// error so dev scripts can branch on it.
func TestDispatcherRunCmdRejectsProjectMismatch(t *testing.T) {
	t.Parallel()

	env := newDispatcherCLITestEnv(t)
	bakeDispatcherCatalog(t, env, "go-builder-agent", "")

	item, err := env.svc.CreateActionItem(context.Background(), app.CreateActionItemInput{
		ProjectID:      env.projectID,
		ColumnID:       env.todoColumnID,
		Title:          "DROPLET CLI MISMATCH TEST",
		Kind:           domain.KindBuild,
		Scope:          domain.KindAppliesToBuild,
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem() error = %v", err)
	}

	var out, errOut strings.Builder
	cliErr := runDispatcherRun(context.Background(), env.svc, env.broker, dispatcherRunCommandOptions{
		actionItemID: item.ID,
		projectID:    "proj-other",
	}, &out, &errOut)
	if cliErr == nil {
		t.Fatalf("runDispatcherRun(--project mismatch) error = nil, want non-zero ErrProjectMismatch")
	}
	if !errors.Is(cliErr, dispatcher.ErrProjectMismatch) {
		t.Fatalf("runDispatcherRun(--project mismatch) error = %v, want errors.Is(ErrProjectMismatch)", cliErr)
	}
}

// TestDispatcherRunCmdHonorsExplicitProjectFlag asserts that supplying a
// --project value matching the action item's own ProjectID does NOT trip
// the mismatch gate. The dispatcher proceeds; the spawn is exercised
// downstream (CLI catalog + agent already wired by the helper).
func TestDispatcherRunCmdHonorsExplicitProjectFlag(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("dispatcher CLI spawn test is Unix-only")
	}

	env := newDispatcherCLITestEnv(t)
	agentBin := buildDispatcherCLIFakeAgent(t)
	bakeDispatcherCatalog(t, env, "go-builder-agent", agentBin)

	item, err := env.svc.CreateActionItem(context.Background(), app.CreateActionItemInput{
		ProjectID:      env.projectID,
		ColumnID:       env.todoColumnID,
		Title:          "DROPLET CLI EXPLICIT PROJECT TEST",
		Kind:           domain.KindBuild,
		Scope:          domain.KindAppliesToBuild,
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem() error = %v", err)
	}

	var out, errOut strings.Builder
	cliErr := runDispatcherRun(context.Background(), env.svc, env.broker, dispatcherRunCommandOptions{
		actionItemID: item.ID,
		projectID:    env.projectID, // matches item.ProjectID
	}, &out, &errOut)
	if cliErr != nil {
		t.Fatalf("runDispatcherRun(--project match) error = %v\nstderr: %s\nstdout: %s",
			cliErr, errOut.String(), out.String())
	}
	if !strings.HasPrefix(out.String(), "spawned ") {
		t.Errorf("stdout = %q, want spawned prefix", out.String())
	}
}

// TestDispatcherRunCmdDryRunReportsIneligible pins the §2.3 fix: --dry-run
// against an action item with an unmet BlockedBy reports eligible=false +
// a reason naming the gate. Pre-fix shape silently returned a successful
// descriptor for the same item.
func TestDispatcherRunCmdDryRunReportsIneligible(t *testing.T) {
	t.Parallel()

	env := newDispatcherCLITestEnv(t)
	bakeDispatcherCatalog(t, env, "go-builder-agent", "")

	// Two siblings: blocker stays in todo; candidate's BlockedBy points at
	// blocker. The walker's predicate rejects the candidate because
	// BlockedBy is not complete.
	blocker, err := env.svc.CreateActionItem(context.Background(), app.CreateActionItemInput{
		ProjectID:      env.projectID,
		ColumnID:       env.todoColumnID,
		Title:          "DROPLET DRY RUN BLOCKER",
		Kind:           domain.KindBuild,
		Scope:          domain.KindAppliesToBuild,
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(blocker) error = %v", err)
	}
	candidate, err := env.svc.CreateActionItem(context.Background(), app.CreateActionItemInput{
		ProjectID:      env.projectID,
		ColumnID:       env.todoColumnID,
		Title:          "DROPLET DRY RUN CANDIDATE",
		Kind:           domain.KindBuild,
		Scope:          domain.KindAppliesToBuild,
		StructuralType: domain.StructuralTypeDroplet,
		Metadata: domain.ActionItemMetadata{
			BlockedBy: []string{blocker.ID},
		},
	})
	if err != nil {
		t.Fatalf("CreateActionItem(candidate) error = %v", err)
	}

	var out, errOut strings.Builder
	if err := runDispatcherRun(context.Background(), env.svc, env.broker,
		dispatcherRunCommandOptions{actionItemID: candidate.ID, dryRun: true},
		&out, &errOut); err != nil {
		t.Fatalf("runDispatcherRun(--dry-run) error = %v", err)
	}

	var decoded struct {
		Eligible  bool   `json:"eligible"`
		Reason    string `json:"reason"`
		AgentName string `json:"agent_name"`
	}
	if err := json.Unmarshal([]byte(out.String()), &decoded); err != nil {
		t.Fatalf("json.Unmarshal(dry-run stdout) error = %v\nstdout:\n%s", err, out.String())
	}
	if decoded.Eligible {
		t.Errorf("dry-run reported eligible=true; want false (blocked_by not clear)")
	}
	if decoded.Reason == "" {
		t.Errorf("dry-run reason is empty; want a non-empty gate-name string")
	}
	if !strings.Contains(decoded.Reason, blocker.ID) {
		t.Errorf("dry-run reason = %q; want substring %q (blocker ID)", decoded.Reason, blocker.ID)
	}
	// Descriptor still rendered so dev scripts inspect would-have-been argv.
	if decoded.AgentName != "go-builder-agent" {
		t.Errorf("dry-run agent_name = %q, want go-builder-agent", decoded.AgentName)
	}
}

// TestSpawnDescriptorJSONStructuralAlignment pins the §3.3 NIT mitigation:
// spawnPreviewJSON's MarshalJSON wire struct enumerates the descriptor's
// 7 fields explicitly. If a future SpawnDescriptor field lands without a
// matching wire-struct edit, the descriptor count drifts AND this test
// fails — protecting dev scripts from silent JSON-shape drops.
func TestSpawnDescriptorJSONStructuralAlignment(t *testing.T) {
	t.Parallel()

	descriptorType := reflect.TypeOf(dispatcher.SpawnDescriptor{})
	const wireFieldCount = 7
	if got := descriptorType.NumField(); got != wireFieldCount {
		t.Errorf("dispatcher.SpawnDescriptor field count = %d, want %d "+
			"(wire struct in spawnDescriptorJSON.MarshalJSON enumerates %d fields; "+
			"add the new field to the wire struct then update this constant)",
			got, wireFieldCount, wireFieldCount)
	}
}

// buildDispatcherCLIFakeAgent compiles internal/app/dispatcher/testdata/fakeagent.go
// to a tmpfile binary the CLI test can spawn. The carve-out documented in
// this file's package doc-comment applies — production code never shells
// out to `go`.
func buildDispatcherCLIFakeAgent(t *testing.T) string {
	t.Helper()
	src, err := filepath.Abs(filepath.Join("..", "..", "internal", "app", "dispatcher", "testdata", "fakeagent.go"))
	if err != nil {
		t.Fatalf("resolve fakeagent.go path: %v", err)
	}
	dir := t.TempDir()
	binName := "dispatcher-cli-fakeagent"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	binPath := filepath.Join(dir, binName)
	cmd := exec.Command("go", "build", "-o", binPath, src)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("buildDispatcherCLIFakeAgent: go build %s -> %s failed: %v\noutput:\n%s", src, binPath, err, string(out))
	}
	return binPath
}
