package dispatcher

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/domain"
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

	// Settle: each subscriber needs to register a Wait before we publish.
	time.Sleep(15 * time.Millisecond)
	for _, projectID := range []string{"proj-a", "proj-b", "proj-c"} {
		broker.Publish(app.LiveWaitEvent{
			Type:  app.LiveWaitEventActionItemChanged,
			Key:   projectID,
			Value: projectID,
		})
	}

	// Each subscriber's RunOnce hits the counting project reader once on
	// its event. Expect 3 total calls within the deadline.
	deadline := time.After(500 * time.Millisecond)
	for {
		if counter.count.Load() >= 3 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("counter calls = %d after 500ms, want >= 3 (one per project)", counter.count.Load())
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
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

	// Settle so the subscriber registers a Wait before the publish.
	time.Sleep(10 * time.Millisecond)
	broker.Publish(app.LiveWaitEvent{
		Type:  app.LiveWaitEventActionItemChanged,
		Key:   "proj-1",
		Value: "proj-1",
	})

	deadline := time.After(150 * time.Millisecond)
	for counter.count.Load() == 0 {
		select {
		case <-deadline:
			t.Fatalf("RunOnce never fired within 150ms after event publish")
		default:
			time.Sleep(2 * time.Millisecond)
		}
	}
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
	// Settle so subscriber goroutines reach Wait.
	time.Sleep(10 * time.Millisecond)

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
	time.Sleep(10 * time.Millisecond)

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
