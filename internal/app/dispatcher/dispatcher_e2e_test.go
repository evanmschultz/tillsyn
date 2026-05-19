// dispatcher_e2e_test.go holds the D5 end-to-end integration tests that
// exercise the production NewDispatcher constructor path. Separated from
// subscriber_test.go (Drop 4b R7.4) so the file split makes the e2e scope
// immediately visible and goleak wiring stays co-located with its targets.
package dispatcher

import (
	"context"
	"errors"
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
type stubE2ETemplateResolver struct {
	tpl templates.Template
}

// GetProjectTemplate implements TemplateResolver. The projectID argument is
// ignored so the same template fixture is returned for any project the e2e
// test drives.
func (s *stubE2ETemplateResolver) GetProjectTemplate(_ context.Context, _ string) (templates.Template, error) {
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
