package dispatcher

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/evanmschultz/tillsyn/internal/templates"
)

// stubProjectLister is a deterministic projectLister test stub. Tests assign
// projects + err directly; calls is incremented on every ListProjects so
// idempotency assertions can pin "Start enumerated projects exactly once".
type stubProjectLister struct {
	projects []domain.Project
	err      error
	calls    atomic.Int32
}

// ListProjects records the call and returns the configured fixture.
func (s *stubProjectLister) ListProjects(_ context.Context, _ bool) ([]domain.Project, error) {
	s.calls.Add(1)
	if s.err != nil {
		return nil, s.err
	}
	return append([]domain.Project(nil), s.projects...), nil
}

// subscriberWalkerStub is a minimal walkerService stub used by the subscriber
// tests. It returns one configured item from ListActionItems so the walker's
// EligibleForPromotion path produces a single eligible candidate. Distinct
// from walker_test.go's stubWalkerService to avoid in-package name collision.
type subscriberWalkerStub struct {
	item domain.ActionItem
}

func (s *subscriberWalkerStub) ListColumns(_ context.Context, _ string, _ bool) ([]domain.Column, error) {
	return nil, nil
}

func (s *subscriberWalkerStub) ListActionItems(_ context.Context, _ string, _ bool) ([]domain.ActionItem, error) {
	if s.item.ID == "" {
		return nil, nil
	}
	return []domain.ActionItem{s.item}, nil
}

func (s *subscriberWalkerStub) MoveActionItem(_ context.Context, _, _ string, _ int) (domain.ActionItem, error) {
	return s.item, nil
}

// countingProjectReader counts GetProject calls. Subscriber tests use the
// call count as a proxy for "RunOnce fired" — RunOnce hits Stage 1 (project
// resolution) immediately after the non-todo gate, so a GetProject call is
// observable evidence that the subscriber pipeline reached the dispatcher.
// The default ErrNotFound short-circuits the downstream pipeline so the test
// does not need to wire walker / monitor.
type countingProjectReader struct {
	count atomic.Int32
}

func (c *countingProjectReader) GetProject(_ context.Context, _ string) (domain.Project, error) {
	c.count.Add(1)
	return domain.Project{}, app.ErrNotFound
}

// waitForSubscriberDelivery publishes the supplied events on a tight retry
// loop until check returns true or deadline elapses. Replaces the brittle
// "time.Sleep then Publish once" pattern that races the subscriber goroutine
// reaching Wait — the in-process broker advances Sequence on every Publish
// (see live_wait.go nextSequence), so every iteration is a fresh wakeup from
// the subscriber's cursor perspective, guaranteeing at least one publish
// lands AFTER the subscriber's Wait registers within the deadline. Stops as
// soon as check fires to keep over-publishing bounded. Fails the test if the
// deadline expires before check returns true.
func waitForSubscriberDelivery(t *testing.T, broker app.LiveWaitBroker, events []app.LiveWaitEvent, check func() bool, deadline time.Duration) {
	t.Helper()
	end := time.Now().Add(deadline)
	for {
		if check() {
			return
		}
		for _, ev := range events {
			broker.Publish(ev)
		}
		if check() {
			return
		}
		if time.Now().After(end) {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}
	if !check() {
		t.Fatalf("waitForSubscriberDelivery: check never returned true within %s", deadline)
	}
}

// newSubscriberDispatcherForTest builds a dispatcher with stubs sufficient
// to exercise Start/Stop + the subscriber goroutine. Walker is wired against
// subscriberWalkerStub so EligibleForPromotion returns a deterministic item;
// RunOnce hits the counting project reader and short-circuits to
// ResultSkipped (project not found). Tests assert on the project-reader
// call count to confirm RunOnce fired.
func newSubscriberDispatcherForTest(broker app.LiveWaitBroker, lister projectLister, item domain.ActionItem) (*dispatcher, *countingProjectReader) {
	walkerStub := &subscriberWalkerStub{item: item}
	counter := &countingProjectReader{}
	d := &dispatcher{
		svc: &stubActionItemReader{
			item: item,
		},
		projects:       counter,
		listing:        walkerStub,
		broker:         broker,
		walker:         newTreeWalker(walkerStub),
		projectsLister: lister,
		clock:          time.Now,
	}
	return d, counter
}

// TestDispatcherStartSpawnsPerProjectSubscribers asserts that Start spins one
// subscriber goroutine per project returned by ListProjects. Each goroutine
// independently subscribes to its project's broker channel; publishing one
// event per project triggers RunOnce on each project's subscriber.
func TestDispatcherStartSpawnsPerProjectSubscribers(t *testing.T) {
	t.Parallel()

	broker := app.NewInProcessLiveWaitBroker()
	lister := &stubProjectLister{
		projects: []domain.Project{
			{ID: "proj-a"},
			{ID: "proj-b"},
			{ID: "proj-c"},
		},
	}
	item := domain.ActionItem{
		ID:             "ai-todo",
		ProjectID:      "proj-a",
		LifecycleState: domain.StateTodo,
	}
	d, counter := newSubscriberDispatcherForTest(broker, lister, item)

	if err := d.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v, want nil", err)
	}
	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		_ = d.Stop(stopCtx)
	})

	if got := lister.calls.Load(); got != 1 {
		t.Fatalf("ListProjects calls = %d, want 1 (Start enumerates once)", got)
	}

	// Publish-retry guarantees at least one publish per key lands after
	// each subscriber's Wait registers, regardless of goroutine scheduling.
	// Each subscriber's RunOnce hits the counting project reader once on
	// its event; we wait until counter >= 3 (one per project).
	events := []app.LiveWaitEvent{
		{Type: app.LiveWaitEventActionItemChanged, Key: "proj-a", Value: "proj-a"},
		{Type: app.LiveWaitEventActionItemChanged, Key: "proj-b", Value: "proj-b"},
		{Type: app.LiveWaitEventActionItemChanged, Key: "proj-c", Value: "proj-c"},
	}
	waitForSubscriberDelivery(t, broker, events, func() bool {
		return counter.count.Load() >= 3
	}, 500*time.Millisecond)
}

// TestDispatcherStartTriggersRunOnceOnEvent asserts that publishing a
// synthetic ActionItemChanged event on a project's broker channel causes the
// subscriber to call RunOnce within 100ms.
func TestDispatcherStartTriggersRunOnceOnEvent(t *testing.T) {
	t.Parallel()

	broker := app.NewInProcessLiveWaitBroker()
	lister := &stubProjectLister{
		projects: []domain.Project{{ID: "proj-1"}},
	}
	item := domain.ActionItem{
		ID:             "ai-todo",
		ProjectID:      "proj-1",
		LifecycleState: domain.StateTodo,
	}
	d, counter := newSubscriberDispatcherForTest(broker, lister, item)

	if err := d.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v, want nil", err)
	}
	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		_ = d.Stop(stopCtx)
	})

	// Publish-retry guarantees at least one publish lands after the
	// subscriber's Wait registers, regardless of goroutine scheduling.
	events := []app.LiveWaitEvent{
		{Type: app.LiveWaitEventActionItemChanged, Key: "proj-1", Value: "proj-1"},
	}
	waitForSubscriberDelivery(t, broker, events, func() bool {
		return counter.count.Load() >= 1
	}, 200*time.Millisecond)
	if got := counter.count.Load(); got < 1 {
		t.Fatalf("RunOnce calls = %d, want >= 1", got)
	}
}

// TestDispatcherStopCancelsAllSubscribers asserts that Stop cancels every
// subscriber goroutine spawned by Start and waits for them to drain. Drain
// is observed via WaitGroup completion within Stop's ctx-deadline.
func TestDispatcherStopCancelsAllSubscribers(t *testing.T) {
	t.Parallel()

	broker := app.NewInProcessLiveWaitBroker()
	lister := &stubProjectLister{
		projects: []domain.Project{
			{ID: "proj-a"},
			{ID: "proj-b"},
		},
	}
	d, _ := newSubscriberDispatcherForTest(broker, lister, domain.ActionItem{})

	if err := d.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v, want nil", err)
	}
	// subWG.Add(N) runs synchronously inside Start before Start returns, so
	// Stop's cancel + subWG.Wait drains regardless of whether the spawned
	// goroutines have reached their Wait call yet — ctx.Err and ctx.Done in
	// runBrokerSubscriber both respond to cancellation.

	stopCtx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if err := d.Stop(stopCtx); err != nil {
		t.Fatalf("Stop() error = %v, want nil (clean drain)", err)
	}

	// Confirm Stop fully drained: WaitGroup counter is 0. A fresh short
	// Wait verifies draining is a no-op now.
	drained := make(chan struct{})
	go func() {
		d.subWG.Wait()
		close(drained)
	}()
	select {
	case <-drained:
	case <-time.After(50 * time.Millisecond):
		t.Fatalf("subWG.Wait() did not return after Stop; subscriber goroutines leaked")
	}
}

// TestDispatcherStartIdempotent asserts that a second Start call without an
// intervening Stop returns ErrAlreadyStarted and does not spawn duplicate
// subscribers (ListProjects called exactly once).
func TestDispatcherStartIdempotent(t *testing.T) {
	t.Parallel()

	broker := app.NewInProcessLiveWaitBroker()
	lister := &stubProjectLister{
		projects: []domain.Project{{ID: "proj-1"}},
	}
	d, _ := newSubscriberDispatcherForTest(broker, lister, domain.ActionItem{})

	if err := d.Start(context.Background()); err != nil {
		t.Fatalf("first Start() error = %v, want nil", err)
	}
	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		_ = d.Stop(stopCtx)
	})

	err := d.Start(context.Background())
	if err == nil {
		t.Fatalf("second Start() error = nil, want ErrAlreadyStarted")
	}
	if !errors.Is(err, ErrAlreadyStarted) {
		t.Fatalf("second Start() error = %v, want errors.Is(ErrAlreadyStarted)", err)
	}
	if got := lister.calls.Load(); got != 1 {
		t.Fatalf("ListProjects calls = %d, want 1 (second Start must not enumerate)", got)
	}
}

// TestDispatcherStopIdempotent asserts that a second Stop call after the
// first returns nil immediately and does not block.
func TestDispatcherStopIdempotent(t *testing.T) {
	t.Parallel()

	broker := app.NewInProcessLiveWaitBroker()
	lister := &stubProjectLister{
		projects: []domain.Project{{ID: "proj-1"}},
	}
	d, _ := newSubscriberDispatcherForTest(broker, lister, domain.ActionItem{})

	if err := d.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v, want nil", err)
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if err := d.Stop(stopCtx); err != nil {
		t.Fatalf("first Stop() error = %v, want nil", err)
	}

	// Second Stop must return immediately. Bound the wait to detect a
	// regression that re-enters Wait/cancel/wg.
	done := make(chan error, 1)
	go func() {
		done <- d.Stop(context.Background())
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("second Stop() error = %v, want nil", err)
		}
	case <-time.After(50 * time.Millisecond):
		t.Fatalf("second Stop() did not return within 50ms")
	}

	// Stop on a never-started dispatcher: also nil.
	freshLister := &stubProjectLister{projects: []domain.Project{{ID: "proj-1"}}}
	fresh, _ := newSubscriberDispatcherForTest(broker, freshLister, domain.ActionItem{})
	if err := fresh.Stop(context.Background()); err != nil {
		t.Fatalf("Stop(no prior Start) error = %v, want nil", err)
	}
}

// TestDispatcherStartStopConcurrent asserts that concurrent Start + Stop
// calls do not deadlock. The final state is "stopped" — a subsequent Start
// must return ErrAlreadyStarted because re-start of a stopped dispatcher is
// rejected.
func TestDispatcherStartStopConcurrent(t *testing.T) {
	t.Parallel()

	broker := app.NewInProcessLiveWaitBroker()
	lister := &stubProjectLister{
		projects: []domain.Project{{ID: "proj-1"}, {ID: "proj-2"}},
	}
	d, _ := newSubscriberDispatcherForTest(broker, lister, domain.ActionItem{})

	startErr := make(chan error, 1)
	stopErr := make(chan error, 1)
	go func() {
		startErr <- d.Start(context.Background())
	}()
	go func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		// Tiny delay so Start has a real chance to win the race; the
		// alternative race (Stop wins, Start no-ops on stopped flag) is
		// also acceptable per the contract.
		time.Sleep(2 * time.Millisecond)
		stopErr <- d.Stop(stopCtx)
	}()

	select {
	case err := <-startErr:
		// Start may succeed or be rejected if Stop ran first. Both are
		// acceptable; the test pins "no deadlock" via the timeout.
		if err != nil && !errors.Is(err, ErrAlreadyStarted) {
			t.Fatalf("Start() error = %v, want nil or ErrAlreadyStarted", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("Start() did not return within 500ms — concurrent Start+Stop deadlocked")
	}
	select {
	case err := <-stopErr:
		if err != nil {
			t.Fatalf("Stop() error = %v, want nil", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("Stop() did not return within 500ms — concurrent Start+Stop deadlocked")
	}

	// Always-on final state: a re-start MUST be rejected because the
	// dispatcher is stopped (or was never successfully started — both
	// surface ErrAlreadyStarted).
	err := d.Start(context.Background())
	if err == nil {
		t.Fatalf("re-Start after Stop error = nil, want ErrAlreadyStarted")
	}
	if !errors.Is(err, ErrAlreadyStarted) {
		t.Fatalf("re-Start after Stop error = %v, want errors.Is(ErrAlreadyStarted)", err)
	}
}

// TestDispatcherStartRejectsListProjectsError asserts that a ListProjects
// failure surfaces as a wrapped error from Start (no goroutines spawned,
// dispatcher not marked started).
func TestDispatcherStartRejectsListProjectsError(t *testing.T) {
	t.Parallel()

	broker := app.NewInProcessLiveWaitBroker()
	wantErr := errors.New("database closed")
	lister := &stubProjectLister{err: wantErr}
	d, _ := newSubscriberDispatcherForTest(broker, lister, domain.ActionItem{})

	err := d.Start(context.Background())
	if err == nil {
		t.Fatalf("Start() error = nil, want wrapped %v", wantErr)
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("Start() error = %v, want errors.Is(%v)", err, wantErr)
	}
	d.subMu.Lock()
	defer d.subMu.Unlock()
	if d.started {
		t.Fatalf("dispatcher.started = true after failed Start, want false")
	}
}

// TestDispatcherStartRejectsNilProjectsLister asserts a nil projectsLister
// surfaces ErrInvalidDispatcherConfig before any goroutine spawn or
// ListProjects call.
func TestDispatcherStartRejectsNilProjectsLister(t *testing.T) {
	t.Parallel()

	d := &dispatcher{
		svc:      &stubActionItemReader{},
		projects: &stubProjectReader{err: app.ErrNotFound},
		listing:  stubListingService{},
		broker:   app.NewInProcessLiveWaitBroker(),
		walker:   newTreeWalker(&subscriberWalkerStub{}),
		clock:    time.Now,
	}
	err := d.Start(context.Background())
	if err == nil {
		t.Fatalf("Start() error = nil, want ErrInvalidDispatcherConfig")
	}
	if !errors.Is(err, ErrInvalidDispatcherConfig) {
		t.Fatalf("Start() error = %v, want errors.Is(ErrInvalidDispatcherConfig)", err)
	}
}

// TestHandleSubscriberEventInvokesRunOnceForEachEligibleItem pins the
// per-event handler in isolation: given two eligible items, RunOnce is
// invoked twice. Protects the invariant that the subscriber drains the
// eligible-set per event rather than just the head.
func TestHandleSubscriberEventInvokesRunOnceForEachEligibleItem(t *testing.T) {
	t.Parallel()

	walker := &multiItemWalkerStub{
		items: []domain.ActionItem{
			{ID: "ai-1", ProjectID: "proj-1", LifecycleState: domain.StateTodo},
			{ID: "ai-2", ProjectID: "proj-1", LifecycleState: domain.StateTodo},
		},
	}
	counter := &countingProjectReader{}
	d := &dispatcher{
		svc: &multiItemActionItemReader{
			items: map[string]domain.ActionItem{
				"ai-1": {ID: "ai-1", ProjectID: "proj-1", LifecycleState: domain.StateTodo},
				"ai-2": {ID: "ai-2", ProjectID: "proj-1", LifecycleState: domain.StateTodo},
			},
		},
		projects: counter,
		listing:  walker,
		broker:   app.NewInProcessLiveWaitBroker(),
		walker:   newTreeWalker(walker),
		clock:    time.Now,
	}

	d.handleSubscriberEvent(context.Background(), "proj-1")

	if got := counter.count.Load(); got != 2 {
		t.Fatalf("counter calls = %d, want 2 (one RunOnce per eligible item)", got)
	}
}

// multiItemWalkerStub returns multiple eligible items so the subscriber's
// per-event drain can be exercised. Distinct from walker_test.go's
// stubWalkerService to avoid in-package name collision.
type multiItemWalkerStub struct {
	items []domain.ActionItem
}

func (m *multiItemWalkerStub) ListColumns(_ context.Context, _ string, _ bool) ([]domain.Column, error) {
	return nil, nil
}

func (m *multiItemWalkerStub) ListActionItems(_ context.Context, _ string, _ bool) ([]domain.ActionItem, error) {
	return append([]domain.ActionItem(nil), m.items...), nil
}

func (m *multiItemWalkerStub) MoveActionItem(_ context.Context, _, _ string, _ int) (domain.ActionItem, error) {
	return domain.ActionItem{}, nil
}

// multiItemActionItemReader returns items from a map so RunOnce's
// GetActionItem lookup resolves whichever item the eligible-set walk asks
// for.
type multiItemActionItemReader struct {
	items map[string]domain.ActionItem
}

func (r *multiItemActionItemReader) GetActionItem(_ context.Context, id string) (domain.ActionItem, error) {
	if item, ok := r.items[id]; ok {
		return item, nil
	}
	return domain.ActionItem{}, app.ErrNotFound
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

// TestAutoDispatchE2EGatePassViaNewDispatcher pins two D5 invariants on the
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
func TestAutoDispatchE2EGatePassViaNewDispatcher(t *testing.T) {
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
