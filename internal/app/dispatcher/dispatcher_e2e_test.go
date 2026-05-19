// dispatcher_e2e_test.go holds the D5 end-to-end integration tests that
// exercise the production NewDispatcher constructor path. Separated from
// subscriber_test.go (Drop 4b R7.4) so the file split makes the e2e scope
// immediately visible and goleak wiring stays co-located with its targets.
package dispatcher

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"go.uber.org/goleak"

	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/evanmschultz/tillsyn/internal/templates"
)

// TestMain wires goleak goroutine-leak detection for the e2e tests in this
// file. goleak.VerifyTestMain runs all tests via m.Run(), checks for leaked
// goroutines after they complete, and handles its own os.Exit call — callers
// must NOT wrap VerifyTestMain in os.Exit (it returns void, not int).
//
// Scope: this TestMain covers the full dispatcher package test binary (all
// *_test.go files in internal/app/dispatcher), not just the e2e tests — a
// single package may have only one TestMain. If VerifyTestMain surfaces leaks
// from tests outside this file, those are documented in the builder worklog
// under "Out-of-Scope Leak Findings" and deferred to a future drop per the
// R6.2 scope-creep guard in PLAN.md (lines 86-90 region).
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

// stubE2ETemplateResolver is a deterministic TemplateResolver stub for the D5
// end-to-end tests. It returns the configured template without consulting any
// storage; tests assign tpl directly to control what gate sequence the gate
// runner sees.
//
// R7.3 parameterization: tplByProject allows per-project template overrides.
// GetProjectTemplate returns tplByProject[projectID] when an entry exists for
// the supplied ID; otherwise it falls back to the shared tpl field. Tests that
// only need a single template leave tplByProject nil or empty.
type stubE2ETemplateResolver struct {
	tpl          templates.Template
	tplByProject map[string]templates.Template
}

// GetProjectTemplate implements TemplateResolver. When tplByProject contains
// an entry for projectID that entry is returned; otherwise tpl is returned so
// single-template tests continue to work without change.
func (s *stubE2ETemplateResolver) GetProjectTemplate(_ context.Context, projectID string) (templates.Template, error) {
	if s.tplByProject != nil {
		if tpl, ok := s.tplByProject[projectID]; ok {
			return tpl, nil
		}
	}
	return s.tpl, nil
}

// TestAutoDispatch_NewDispatcherGateWiring pins two D5 invariants on the
// gate-pass branch:
//
//  1. The production NewDispatcher constructor wires gates such that
//     `d.gates.Run` with a mage_ci template returns GateStatusPassed when
//     the underlying command exits 0.
//  2. Start launches the per-project subscriber goroutine without panic and
//     ListProjects is invoked exactly once.
//
// What this test does NOT verify (intentionally, to keep the assertion set
// honest after Drop 4b R6/R7 falsification): the publish -> subscriber ->
// handleSubscriberEvent chain firing as a side-effect. The walker stub
// returns no eligible items, so even if a publish landed it would observe
// nothing. The broker chain is exercised in
// TestDispatcherStartTriggersRunOnceOnEvent + TestHandleSubscriberEvent... .
//
// The test does NOT call t.Parallel() because withFakeCommandRunner swaps a
// package-level var (defaultCommandRunner); parallel execution would race on
// that swap.
func TestAutoDispatch_NewDispatcherGateWiring(t *testing.T) {
	// Wire a fake command runner that simulates a successful mage ci run
	// (exit code 0, representative stdout discarded on pass path).
	fake := &fakeCommandRunner{
		stdout:   []byte("mage: target ci ok\n"),
		stderr:   nil,
		exitCode: 0,
	}
	withFakeCommandRunner(t, fake)

	broker := app.NewInProcessLiveWaitBroker()
	resolver := &stubE2ETemplateResolver{
		tpl: templates.Template{
			SchemaVersion: templates.SchemaVersionV1,
			Gates: map[domain.Kind][]templates.GateKind{
				domain.KindBuild: {templates.GateKindMageCI},
			},
		},
	}

	// NewDispatcher is the production constructor path mandated by D5.
	svc := app.NewService(nil, nil, nil, app.ServiceConfig{})
	d, err := NewDispatcher(svc, broker, Options{TemplateResolver: resolver})
	if err != nil {
		t.Fatalf("NewDispatcher() error = %v, want nil", err)
	}

	// Override storage-touching fields after construction so the subscriber
	// goroutine can exercise the broker→event chain without hitting nil repo
	// pointers inside the real *app.Service.
	lister := &stubProjectLister{
		projects: []domain.Project{{ID: "proj-e2e-pass"}},
	}
	d.projectsLister = lister
	d.listing = &subscriberWalkerStub{} // empty eligible set: no RunOnce calls, no nil-deref
	d.walker = newTreeWalker(&subscriberWalkerStub{})

	// Start the dispatcher — this is the broker→subscriber half of the e2e chain.
	if err := d.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v, want nil", err)
	}
	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		_ = d.Stop(stopCtx)
	})

	// Confirm Start enumerated projects exactly once (subscriber-startup pin).
	// "state transitions" in the D5 spec means dispatcher lifecycle (Start/Stop),
	// not action-item state. This lister-calls pin is the lifecycle-transition signal.
	if got := lister.calls.Load(); got != 1 {
		t.Fatalf("ListProjects calls = %d, want 1 (Start enumerates once)", got)
	}

	// Gate-pass branch: run the gate runner directly against a build item +
	// populated project. NewDispatcher wired mage_ci into d.gates; the fake
	// command runner returns exit 0 -> GateStatusPassed.
	item := domain.ActionItem{ID: "ai-e2e-pass", Kind: domain.KindBuild}
	project := domain.Project{
		ID:                  "proj-e2e-pass",
		RepoPrimaryWorktree: "/tmp/proj-e2e-pass",
	}
	results := d.gates.Run(context.Background(), item, project, &resolver.tpl)
	if len(results) != 1 {
		t.Fatalf("gate results len = %d, want 1", len(results))
	}
	if results[0].Status != GateStatusPassed {
		t.Fatalf("gate results[0].Status = %q, want %q; Err = %v",
			results[0].Status, GateStatusPassed, results[0].Err)
	}
	if results[0].Err != nil {
		t.Fatalf("gate results[0].Err = %v, want nil (gate passed)", results[0].Err)
	}
}

// TestAutoDispatchE2EGateFailViaNewDispatcher is the D5 end-to-end integration
// test for the gate-fail branch. It exercises the same production wiring as
// the gate-pass test but targets a gate kind (mage_test_pkg) that NewDispatcher
// does NOT register, triggering ErrGateNotRegistered → GateStatusFailed.
//
// ErrGateNotRegistered is the canonical fail-loud signal that the template
// references a gate the runner cannot resolve — distinguishable from a gate
// that ran and failed (which carries a different Err value). The two failure
// modes being distinguishable via errors.Is is the load-bearing assertion.
func TestAutoDispatchE2EGateFailViaNewDispatcher(t *testing.T) {
	t.Parallel()

	broker := app.NewInProcessLiveWaitBroker()
	// mage_test_pkg is in the closed GateKind enum (4b.1) but is NOT registered
	// by NewDispatcher (only mage_ci is registered per dispatcher.go:327-329).
	// Requesting it via Run causes ErrGateNotRegistered → GateStatusFailed.
	resolver := &stubE2ETemplateResolver{
		tpl: templates.Template{
			SchemaVersion: templates.SchemaVersionV1,
			Gates: map[domain.Kind][]templates.GateKind{
				domain.KindBuild: {templates.GateKindMageTestPkg},
			},
		},
	}

	svc := app.NewService(nil, nil, nil, app.ServiceConfig{})
	d, err := NewDispatcher(svc, broker, Options{TemplateResolver: resolver})
	if err != nil {
		t.Fatalf("NewDispatcher() error = %v, want nil", err)
	}

	// Gate-fail branch: the gate runner cannot resolve mage_test_pkg → fails.
	item := domain.ActionItem{ID: "ai-e2e-fail", Kind: domain.KindBuild}
	project := domain.Project{
		ID:                  "proj-e2e-fail",
		RepoPrimaryWorktree: "/tmp/proj-e2e-fail",
	}
	results := d.gates.Run(context.Background(), item, project, &resolver.tpl)
	if len(results) != 1 {
		t.Fatalf("gate results len = %d, want 1 (unregistered gate halts after first failure)", len(results))
	}
	if results[0].Status != GateStatusFailed {
		t.Fatalf("gate results[0].Status = %q, want %q", results[0].Status, GateStatusFailed)
	}
	if !errors.Is(results[0].Err, ErrGateNotRegistered) {
		t.Fatalf("gate results[0].Err = %v, want errors.Is(ErrGateNotRegistered)", results[0].Err)
	}
	if results[0].GateName != templates.GateKindMageTestPkg {
		t.Fatalf("gate results[0].GateName = %q, want %q", results[0].GateName, templates.GateKindMageTestPkg)
	}
}

// TestStubE2ETemplateResolverRoutesPerProject asserts the tplByProject
// parameterization added in R7.3: when tplByProject contains an entry for a
// project ID, GetProjectTemplate returns that entry; when the project ID is
// absent, the fallback tpl field is returned.
//
// This test only verifies the stub's routing logic — it does NOT assert
// production-resolver behavior (the real dispatcherTemplateResolver lives in
// package main and is not importable from internal/app/dispatcher).
func TestStubE2ETemplateResolverRoutesPerProject(t *testing.T) {
	t.Parallel()

	tplA := templates.Template{
		SchemaVersion: templates.SchemaVersionV1,
		Gates:         map[domain.Kind][]templates.GateKind{domain.KindBuild: {templates.GateKindMageCI}},
	}
	tplB := templates.Template{
		SchemaVersion: templates.SchemaVersionV1,
		Gates:         map[domain.Kind][]templates.GateKind{domain.KindBuild: {templates.GateKindMageTestPkg}},
	}
	tplDefault := templates.Template{SchemaVersion: "default"}

	cases := []struct {
		name       string
		projectID  string
		wantSchema string
	}{
		{name: "proj-a returns tplA", projectID: "proj-a", wantSchema: templates.SchemaVersionV1},
		{name: "proj-b returns tplB", projectID: "proj-b", wantSchema: templates.SchemaVersionV1},
		{name: "unknown returns default", projectID: "proj-unknown", wantSchema: "default"},
	}

	resolver := &stubE2ETemplateResolver{
		tpl: tplDefault,
		tplByProject: map[string]templates.Template{
			"proj-a": tplA,
			"proj-b": tplB,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := resolver.GetProjectTemplate(context.Background(), tc.projectID)
			if err != nil {
				t.Fatalf("GetProjectTemplate(%q) error = %v, want nil", tc.projectID, err)
			}
			if got.SchemaVersion != tc.wantSchema {
				t.Fatalf("GetProjectTemplate(%q) SchemaVersion = %q, want %q",
					tc.projectID, got.SchemaVersion, tc.wantSchema)
			}
		})
	}

	// Verify tplA and tplB routing returns the specifically-configured gate sequence.
	gotA, _ := resolver.GetProjectTemplate(context.Background(), "proj-a")
	if gates := gotA.Gates[domain.KindBuild]; len(gates) != 1 || gates[0] != templates.GateKindMageCI {
		t.Fatalf("proj-a gates = %v, want [GateKindMageCI]", gates)
	}
	gotB, _ := resolver.GetProjectTemplate(context.Background(), "proj-b")
	if gates := gotB.Gates[domain.KindBuild]; len(gates) != 1 || gates[0] != templates.GateKindMageTestPkg {
		t.Fatalf("proj-b gates = %v, want [GateKindMageTestPkg]", gates)
	}
}

// =============================================================================
// R7.1 + R7.2 helpers
// =============================================================================

type e2eBrokerChainService struct {
	mu sync.Mutex

	item    domain.ActionItem
	project domain.Project
	columns []domain.Column

	moveCalls   int
	updateCalls int
	lastMoveCol string
	lastMeta    *domain.ActionItemMetadata
}

func (s *e2eBrokerChainService) getMoveCallCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.moveCalls
}

func (s *e2eBrokerChainService) getUpdateCallCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.updateCalls
}

func (s *e2eBrokerChainService) getLastMoveCol() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastMoveCol
}

func (s *e2eBrokerChainService) getLastMeta() *domain.ActionItemMetadata {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastMeta
}

func (s *e2eBrokerChainService) GetActionItem(_ context.Context, _ string) (domain.ActionItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.item, nil
}

func (s *e2eBrokerChainService) GetProject(_ context.Context, _ string) (domain.Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.project, nil
}

func (s *e2eBrokerChainService) ListColumns(_ context.Context, _ string, _ bool) ([]domain.Column, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]domain.Column(nil), s.columns...), nil
}

func (s *e2eBrokerChainService) ListActionItems(_ context.Context, _ string, _ bool) ([]domain.ActionItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return []domain.ActionItem{s.item}, nil
}

func (s *e2eBrokerChainService) MoveActionItem(_ context.Context, _ string, toColumnID string, _ int) (domain.ActionItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.moveCalls++
	s.lastMoveCol = toColumnID
	switch toColumnID {
	case "col-inprogress":
		s.item.LifecycleState = domain.StateInProgress
		s.item.ColumnID = toColumnID
	case "col-failed":
		s.item.LifecycleState = domain.StateFailed
		s.item.ColumnID = toColumnID
	case "col-complete":
		s.item.LifecycleState = domain.StateComplete
		s.item.ColumnID = toColumnID
	}
	return s.item, nil
}

func (s *e2eBrokerChainService) UpdateActionItem(_ context.Context, in updateActionItemInput) (domain.ActionItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updateCalls++
	if in.Metadata != nil {
		cp := *in.Metadata
		s.lastMeta = &cp
		s.item.Metadata = *in.Metadata
	}
	return s.item, nil
}

func e2eCanonicalColumns(projectID string) []domain.Column {
	return []domain.Column{
		{ID: "col-todo", ProjectID: projectID, Name: "To Do", Position: 0},
		{ID: "col-inprogress", ProjectID: projectID, Name: "In Progress", Position: 1},
		{ID: "col-complete", ProjectID: projectID, Name: "Complete", Position: 2},
		{ID: "col-failed", ProjectID: projectID, Name: "Failed", Position: 3},
	}
}

func buildE2EBrokerChainCatalog(t *testing.T) json.RawMessage {
	t.Helper()
	tpl := templates.Template{
		SchemaVersion: templates.SchemaVersionV1,
		Kinds: map[domain.Kind]templates.KindRule{
			domain.KindBuild: {StructuralType: domain.StructuralTypeDroplet},
		},
		AgentBindings: map[domain.Kind]templates.AgentBinding{
			domain.KindBuild: {
				AgentName:    "builder-agent",
				Model:        "opus",
				MaxTries:     1,
				MaxBudgetUSD: 5,
				MaxTurns:     20,
			},
		},
	}
	encoded, err := json.Marshal(templates.Bake(tpl))
	if err != nil {
		t.Fatalf("buildE2EBrokerChainCatalog: json.Marshal error = %v", err)
	}
	return json.RawMessage(encoded)
}

// installFakeClaudeBinary installs a stub "claude" binary on PATH that exits 0
// regardless of its arguments. This simulates a clean-exiting agent for the
// broker-chain tests. The stub is a POSIX shell script rather than a copy of
// fakeagent because BuildSpawnCommand passes --bare/--agent/--system-prompt-file
// etc. as argv[1]+, and fakeagent.go uses argv[1] as the mode selector
// (unknown modes → exit 2). A shell script that ignores all arguments and
// exits 0 avoids that mismatch.
//
// Not t.Parallel: t.Setenv modifies PATH (shared process state).
func installFakeClaudeBinary(t *testing.T) {
	t.Helper()
	claudeDir := t.TempDir()
	claudePath := filepath.Join(claudeDir, "claude")

	// A minimal POSIX shell script that exits 0 regardless of arguments.
	const scriptBody = "#!/bin/sh\nexit 0\n"
	if err := os.WriteFile(claudePath, []byte(scriptBody), 0o755); err != nil {
		t.Fatalf("installFakeClaudeBinary: write claude script: %v", err)
	}

	t.Setenv("PATH", claudeDir)
	t.Cleanup(ResetEnsureSpawnsGitignoredOnceForTest)
}

func waitForE2ECondition(t *testing.T, check func() bool, deadline time.Duration, msg string) {
	t.Helper()
	end := time.Now().Add(deadline)
	for {
		if check() {
			return
		}
		if time.Now().After(end) {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !check() {
		t.Fatalf("waitForE2ECondition: timed out after %s: %s", deadline, msg)
	}
}

func buildE2EBrokerChainDispatcher(
	t *testing.T,
	svc *e2eBrokerChainService,
	gates *gateRunner,
	resolver TemplateResolver,
) *dispatcher {
	t.Helper()
	broker := app.NewInProcessLiveWaitBroker()
	walker := newTreeWalker(svc)
	conflict := newConflictDetector(&stubConflictService{})
	fileLocks := newFileLockManager()
	pkgLocks := newPackageLockManager()
	mon := newProcessMonitor(svc, nil)
	mon.WireGates(gates, resolver)
	cleanup, err := newCleanupHook(fileLocks, pkgLocks, mon, noopAuthRevoker{})
	if err != nil {
		t.Fatalf("buildE2EBrokerChainDispatcher: newCleanupHook error = %v", err)
	}
	lister := &stubProjectLister{
		projects: []domain.Project{{ID: svc.project.ID}},
	}
	return &dispatcher{
		svc:            svc,
		projects:       svc,
		listing:        svc,
		mutator:        svc,
		broker:         broker,
		walker:         walker,
		conflict:       conflict,
		fileLocks:      fileLocks,
		pkgLocks:       pkgLocks,
		monitor:        mon,
		cleanup:        cleanup,
		projectsLister: lister,
		clock:          time.Now,
	}
}

// TestAutoDispatchE2E_GateFailFullChain is the R7.1 broker-chain test. It
// drives handleSubscriberEvent → RunOnce → monitor.Track (real subprocess
// exits 0) → runHandle → applyCleanExitTransition → gate runner (unregistered
// gate → ErrGateNotRegistered → GateStatusFailed) → transitionToFailed.
//
// Chain reached (per PLAN.md R7.1 acceptance):
//
//	handleSubscriberEvent(ctx, "proj-e2e-chain")
//	  → RunOnce stages 1-8
//	  → monitor.Track → subprocess exits 0
//	  → applyCleanExitTransition
//	  → gates.Run → GateStatusFailed (GateKindMageTestPkg unregistered)
//	  → transitionToFailed → MoveActionItem(col-failed) + UpdateActionItem(outcome=failure)
//
// Not t.Parallel: t.Setenv(PATH) injects the fake "claude" binary.
func TestAutoDispatchE2E_GateFailFullChain(t *testing.T) {
	installFakeClaudeBinary(t)

	const projectID = "proj-e2e-chain"
	const itemID = "ai-e2e-chain"

	svc := &e2eBrokerChainService{
		item: domain.ActionItem{
			ID:             itemID,
			ProjectID:      projectID,
			Kind:           domain.KindBuild,
			LifecycleState: domain.StateTodo,
			ColumnID:       "col-todo",
			Position:       1,
		},
		project: domain.Project{
			ID:                  projectID,
			RepoPrimaryWorktree: t.TempDir(),
			KindCatalogJSON:     buildE2EBrokerChainCatalog(t),
		},
		columns: e2eCanonicalColumns(projectID),
	}

	// Wire a gate runner with NO GateKindMageTestPkg registered. The template
	// declares GateKindMageTestPkg for KindBuild, so gates.Run returns
	// GateStatusFailed (ErrGateNotRegistered) → applyCleanExitTransition calls
	// transitionToFailed.
	gates := newGateRunner()
	// Intentionally do NOT register GateKindMageTestPkg.
	resolver := &stubE2ETemplateResolver{
		tpl: templates.Template{
			SchemaVersion: templates.SchemaVersionV1,
			Kinds:         map[domain.Kind]templates.KindRule{domain.KindBuild: {}},
			Gates: map[domain.Kind][]templates.GateKind{
				domain.KindBuild: {templates.GateKindMageTestPkg},
			},
		},
	}

	d := buildE2EBrokerChainDispatcher(t, svc, gates, resolver)
	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		_ = d.Stop(stopCtx)
	})

	// Drive the chain via handleSubscriberEvent — documented in-package entry
	// point (subscriber.go:168).
	d.handleSubscriberEvent(context.Background(), projectID)

	waitForE2ECondition(t, func() bool {
		return svc.getUpdateCallCount() >= 1
	}, 3*time.Second, "applyCleanExitTransition never fired (updateCalls stayed at 0)")

	meta := svc.getLastMeta()
	if meta == nil {
		t.Fatalf("lastMeta is nil; UpdateActionItem was never called")
	}
	if meta.Outcome != "failure" {
		t.Fatalf("metadata.Outcome = %q, want %q (gate fail → transitionToFailed)", meta.Outcome, "failure")
	}
	if meta.BlockedReason == "" {
		t.Errorf("metadata.BlockedReason is empty; want gate-failure context (ErrGateNotRegistered)")
	}

	// The last recorded column must be col-failed (walker.Promote to
	// col-inprogress is first, transitionToFailed to col-failed is last).
	lastCol := svc.getLastMoveCol()
	if lastCol == "" {
		t.Fatalf("lastMoveCol is empty; MoveActionItem was never called")
	}
	if lastCol != "col-failed" {
		t.Fatalf("lastMoveCol = %q, want %q (transitionToFailed)", lastCol, "col-failed")
	}
}

// TestAutoDispatchE2E_ApplyCleanExitTransitionCoverage is the R7.2 broker-chain
// test. Covers two integration-chain paths through applyCleanExitTransition:
//
// C1 — empty-template fast-path: the resolver returns a zero-valued template
// (SchemaVersion == "") → applyCleanExitTransition skips gate execution and
// calls transitionToComplete → metadata.Outcome = "success". Covers the
// in-loop GateStatusSkipped branch added by commit d949f6f at monitor.go:500.
//
// C2 — skipped-gate no-transition: the gate runner returns GateStatusSkipped
// → applyCleanExitTransition returns nil without any state transition.
//
// PLAN.md R7.2 originally named C2 as "ctx-cancel pre-loop" (the sibling
// branch added by the same commit d949f6f at monitor.go:492 — guards
// `len(tpl.Gates[item.Kind]) > 0 && len(results) == 0`). That pre-loop branch
// requires a deterministic in-monitor cancellation seam that is not currently
// exposed; substituting via GateStatusSkipped covers the equivalent
// behavioral invariant (no state transition) through the broker chain. The
// pre-loop branch remains uncovered at integration scope and is filed as a
// refinement (see REFINEMENTS.md 2026-05-18 entry) for a follow-up drop that
// can extend processMonitor with a cancellation-injection seam.
//
// Not t.Parallel: t.Setenv(PATH) injects the fake "claude" binary.
func TestAutoDispatchE2E_ApplyCleanExitTransitionCoverage(t *testing.T) {
	installFakeClaudeBinary(t)

	t.Run("C1_empty_template_transitions_to_complete", func(t *testing.T) {
		const projectID = "proj-e2e-c1"
		const itemID = "ai-e2e-c1"

		svc := &e2eBrokerChainService{
			item: domain.ActionItem{
				ID:             itemID,
				ProjectID:      projectID,
				Kind:           domain.KindBuild,
				LifecycleState: domain.StateTodo,
				ColumnID:       "col-todo",
				Position:       1,
			},
			project: domain.Project{
				ID:                  projectID,
				RepoPrimaryWorktree: t.TempDir(),
				KindCatalogJSON:     buildE2EBrokerChainCatalog(t),
			},
			columns: e2eCanonicalColumns(projectID),
		}

		// Resolver returns an empty template (SchemaVersion == "") →
		// applyCleanExitTransition skips gate execution and calls
		// transitionToComplete directly.
		emptyResolver := &stubE2ETemplateResolver{tpl: templates.Template{}}
		gates := newGateRunner()
		d := buildE2EBrokerChainDispatcher(t, svc, gates, emptyResolver)
		t.Cleanup(func() {
			stopCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()
			_ = d.Stop(stopCtx)
		})

		d.handleSubscriberEvent(context.Background(), projectID)

		waitForE2ECondition(t, func() bool {
			return svc.getUpdateCallCount() >= 1
		}, 3*time.Second, "C1: applyCleanExitTransition never fired (updateCalls stayed at 0)")

		meta := svc.getLastMeta()
		if meta == nil {
			t.Fatalf("C1: lastMeta is nil; UpdateActionItem was never called")
		}
		if meta.Outcome != "success" {
			t.Fatalf("C1: metadata.Outcome = %q, want %q", meta.Outcome, "success")
		}
	})

	t.Run("C2_skipped_gate_leaves_item_in_progress", func(t *testing.T) {
		const projectID = "proj-e2e-c2"
		const itemID = "ai-e2e-c2"

		svc := &e2eBrokerChainService{
			item: domain.ActionItem{
				ID:             itemID,
				ProjectID:      projectID,
				Kind:           domain.KindBuild,
				LifecycleState: domain.StateTodo,
				ColumnID:       "col-todo",
				Position:       1,
			},
			project: domain.Project{
				ID:                  projectID,
				RepoPrimaryWorktree: t.TempDir(),
				KindCatalogJSON:     buildE2EBrokerChainCatalog(t),
			},
			columns: e2eCanonicalColumns(projectID),
		}

		// GateStatusSkipped means external cancellation (not a verdict) →
		// applyCleanExitTransition returns nil without transitioning state.
		const e2eSkipGateKind templates.GateKind = "e2e-skip-gate"
		skippedGates := newGateRunner()
		if err := skippedGates.Register(e2eSkipGateKind, func(_ context.Context, _ domain.ActionItem, _ domain.Project) GateResult {
			return GateResult{GateName: e2eSkipGateKind, Status: GateStatusSkipped, Err: context.Canceled}
		}); err != nil {
			t.Fatalf("C2: Register skipped gate: %v", err)
		}
		skippedResolver := &stubE2ETemplateResolver{
			tpl: templates.Template{
				SchemaVersion: templates.SchemaVersionV1,
				Kinds:         map[domain.Kind]templates.KindRule{domain.KindBuild: {}},
				Gates:         map[domain.Kind][]templates.GateKind{domain.KindBuild: {e2eSkipGateKind}},
			},
		}

		d := buildE2EBrokerChainDispatcher(t, svc, skippedGates, skippedResolver)
		t.Cleanup(func() {
			stopCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()
			_ = d.Stop(stopCtx)
		})

		d.handleSubscriberEvent(context.Background(), projectID)

		// The skipped-gate path does NOT call UpdateActionItem. Poll for
		// walker.Promote (moveCalls >= 1 from Stage 7) as evidence RunOnce
		// reached Stage 8, then assert no UpdateActionItem after settling.
		waitForE2ECondition(t, func() bool {
			return svc.getMoveCallCount() >= 1
		}, 3*time.Second, "C2: walker.Promote never fired (moveCalls stayed at 0)")

		time.Sleep(500 * time.Millisecond)

		if got := svc.getUpdateCallCount(); got != 0 {
			t.Fatalf("C2: updateCalls = %d, want 0 (skipped gate must not trigger state transition)", got)
		}
		if lastCol := svc.getLastMoveCol(); lastCol != "" && lastCol != "col-inprogress" {
			if lastCol == "col-complete" || lastCol == "col-failed" {
				t.Fatalf("C2: lastMoveCol = %q, want col-inprogress", lastCol)
			}
		}
	})
}
