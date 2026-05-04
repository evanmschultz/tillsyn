package dispatcher

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// stubActionItemReader is a deterministic test stub for the dispatcher's
// action-item lookup dependency. The test suite injects one of these into a
// dispatcher constructed via the package-internal struct literal so RunOnce
// scenarios can be exercised without a full Service + Repository graph.
type stubActionItemReader struct {
	wantID string
	item   domain.ActionItem
	err    error
	calls  int
}

// GetActionItem records the call and returns the configured fixture.
func (s *stubActionItemReader) GetActionItem(_ context.Context, actionItemID string) (domain.ActionItem, error) {
	s.calls++
	s.wantID = actionItemID
	if s.err != nil {
		return domain.ActionItem{}, s.err
	}
	return s.item, nil
}

// newServiceForConstructorTest returns one *app.Service that is shape-valid
// for NewDispatcher's nil-check. The service is never invoked through the
// dispatcher in constructor tests (those exercise validation only); RunOnce
// tests use the dispatcher{} struct-literal path with a stub reader.
func newServiceForConstructorTest() *app.Service {
	return app.NewService(nil, nil, nil, app.ServiceConfig{})
}

// newBrokerForTest returns one in-process broker for tests.
func newBrokerForTest() app.LiveWaitBroker {
	return app.NewInProcessLiveWaitBroker()
}

// TestNewDispatcherRejectsNilService asserts the constructor wraps
// ErrInvalidDispatcherConfig when svc is nil.
func TestNewDispatcherRejectsNilService(t *testing.T) {
	t.Parallel()

	d, err := NewDispatcher(nil, newBrokerForTest(), Options{})
	if err == nil {
		t.Fatalf("NewDispatcher(nil svc) error = nil, want non-nil")
	}
	if !errors.Is(err, ErrInvalidDispatcherConfig) {
		t.Fatalf("NewDispatcher(nil svc) error = %v, want errors.Is(ErrInvalidDispatcherConfig)", err)
	}
	if d != nil {
		t.Fatalf("NewDispatcher(nil svc) dispatcher = %v, want nil", d)
	}
}

// TestNewDispatcherRejectsNilBroker asserts the constructor wraps
// ErrInvalidDispatcherConfig when broker is nil.
func TestNewDispatcherRejectsNilBroker(t *testing.T) {
	t.Parallel()

	d, err := NewDispatcher(newServiceForConstructorTest(), nil, Options{})
	if err == nil {
		t.Fatalf("NewDispatcher(nil broker) error = nil, want non-nil")
	}
	if !errors.Is(err, ErrInvalidDispatcherConfig) {
		t.Fatalf("NewDispatcher(nil broker) error = %v, want errors.Is(ErrInvalidDispatcherConfig)", err)
	}
	if d != nil {
		t.Fatalf("NewDispatcher(nil broker) dispatcher = %v, want nil", d)
	}
}

// TestNewDispatcherSucceedsWithValidArgs asserts the constructor returns a
// non-nil dispatcher when both dependencies are non-nil.
func TestNewDispatcherSucceedsWithValidArgs(t *testing.T) {
	t.Parallel()

	d, err := NewDispatcher(newServiceForConstructorTest(), newBrokerForTest(), Options{})
	if err != nil {
		t.Fatalf("NewDispatcher() error = %v, want nil", err)
	}
	if d == nil {
		t.Fatalf("NewDispatcher() dispatcher = nil, want non-nil")
	}
	// Compile-time assertion below in dispatcher.go also catches this; the
	// runtime check guards against future refactors that drop the
	// interface-satisfaction guarantee.
	var _ Dispatcher = d
}

// TestRunOnceSkipsEmptyActionItemID asserts that an empty/whitespace ID
// returns ResultSkipped without consulting the service.
func TestRunOnceSkipsEmptyActionItemID(t *testing.T) {
	t.Parallel()

	stub := &stubActionItemReader{}
	d := newDispatcherForTest(stub)

	outcome, err := d.RunOnce(context.Background(), "   ")
	if err != nil {
		t.Fatalf("RunOnce() error = %v, want nil", err)
	}
	if outcome.Result != ResultSkipped {
		t.Fatalf("RunOnce() Result = %q, want %q", outcome.Result, ResultSkipped)
	}
	if outcome.ActionItemID != "" {
		t.Fatalf("RunOnce() ActionItemID = %q, want empty", outcome.ActionItemID)
	}
	if stub.calls != 0 {
		t.Fatalf("stub.calls = %d, want 0 (empty ID short-circuits)", stub.calls)
	}
}

// TestRunOnceSkipsNonExistentActionItem asserts that ErrNotFound from the
// service surfaces as ResultSkipped.
func TestRunOnceSkipsNonExistentActionItem(t *testing.T) {
	t.Parallel()

	stub := &stubActionItemReader{err: app.ErrNotFound}
	d := newDispatcherForTest(stub)

	outcome, err := d.RunOnce(context.Background(), "missing-id")
	if err != nil {
		t.Fatalf("RunOnce() error = %v, want nil", err)
	}
	if outcome.Result != ResultSkipped {
		t.Fatalf("RunOnce() Result = %q, want %q", outcome.Result, ResultSkipped)
	}
	if outcome.ActionItemID != "missing-id" {
		t.Fatalf("RunOnce() ActionItemID = %q, want %q", outcome.ActionItemID, "missing-id")
	}
	if stub.calls != 1 {
		t.Fatalf("stub.calls = %d, want 1", stub.calls)
	}
	if stub.wantID != "missing-id" {
		t.Fatalf("stub forwarded ID = %q, want %q", stub.wantID, "missing-id")
	}
}

// TestRunOnceSkipsNonTodoActionItem asserts that an action item in any
// non-todo lifecycle state surfaces as ResultSkipped.
func TestRunOnceSkipsNonTodoActionItem(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		state domain.LifecycleState
	}{
		{name: "in_progress", state: domain.StateInProgress},
		{name: "complete", state: domain.StateComplete},
		{name: "failed", state: domain.StateFailed},
		{name: "archived", state: domain.StateArchived},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			stub := &stubActionItemReader{
				item: domain.ActionItem{
					ID:             "ai-1",
					LifecycleState: tc.state,
				},
			}
			d := newDispatcherForTest(stub)

			outcome, err := d.RunOnce(context.Background(), "ai-1")
			if err != nil {
				t.Fatalf("RunOnce() error = %v, want nil", err)
			}
			if outcome.Result != ResultSkipped {
				t.Fatalf("RunOnce() Result = %q, want %q (state=%q)", outcome.Result, ResultSkipped, tc.state)
			}
			if outcome.ActionItemID != "ai-1" {
				t.Fatalf("RunOnce() ActionItemID = %q, want %q", outcome.ActionItemID, "ai-1")
			}
		})
	}
}

// TestRunOnceTodoActionItemSkipsForNow asserts that the Wave 2.1 skeleton
// returns Skipped even for eligible todo items — the walker / locks / spawn
// machinery lands in later droplets. This pins the skeleton's contract so a
// future droplet's "now Spawned" change is a deliberate test edit, not a
// silent regression.
func TestRunOnceTodoActionItemSkipsForNow(t *testing.T) {
	t.Parallel()

	stub := &stubActionItemReader{
		item: domain.ActionItem{
			ID:             "ai-todo",
			LifecycleState: domain.StateTodo,
		},
	}
	d := newDispatcherForTest(stub)

	outcome, err := d.RunOnce(context.Background(), "ai-todo")
	if err != nil {
		t.Fatalf("RunOnce() error = %v, want nil", err)
	}
	if outcome.Result != ResultSkipped {
		t.Fatalf("RunOnce() Result = %q, want %q (skeleton always skips in 2.1)", outcome.Result, ResultSkipped)
	}
}

// TestRunOncePropagatesUnexpectedServiceError asserts that errors other than
// app.ErrNotFound bubble up to the caller wrapped with context.
func TestRunOncePropagatesUnexpectedServiceError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("database closed")
	stub := &stubActionItemReader{err: wantErr}
	d := newDispatcherForTest(stub)

	_, err := d.RunOnce(context.Background(), "ai-x")
	if err == nil {
		t.Fatalf("RunOnce() error = nil, want %v", wantErr)
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("RunOnce() error = %v, want errors.Is(%v)", err, wantErr)
	}
}

// TestRunOnceTrimsActionItemIDWhitespace asserts the dispatcher trims
// whitespace before the lookup, so " ai-1 " and "ai-1" share one path.
func TestRunOnceTrimsActionItemIDWhitespace(t *testing.T) {
	t.Parallel()

	stub := &stubActionItemReader{
		item: domain.ActionItem{
			ID:             "ai-trim",
			LifecycleState: domain.StateInProgress,
		},
	}
	d := newDispatcherForTest(stub)

	outcome, err := d.RunOnce(context.Background(), "  ai-trim  ")
	if err != nil {
		t.Fatalf("RunOnce() error = %v, want nil", err)
	}
	if outcome.ActionItemID != "ai-trim" {
		t.Fatalf("RunOnce() ActionItemID = %q, want %q", outcome.ActionItemID, "ai-trim")
	}
	if stub.wantID != "ai-trim" {
		t.Fatalf("stub forwarded ID = %q, want %q (whitespace must be trimmed)", stub.wantID, "ai-trim")
	}
}

// TestRunOnceUsesInjectedClock asserts that a dispatcher with a fixed clock
// emits SpawnedAt aligned with that clock — important so the future
// continuous-mode loop can deterministically order outcomes.
func TestRunOnceUsesInjectedClock(t *testing.T) {
	t.Parallel()

	fixed := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	stub := &stubActionItemReader{
		item: domain.ActionItem{
			ID:             "ai-clock",
			LifecycleState: domain.StateComplete,
		},
	}
	d := &dispatcher{
		svc:    stub,
		broker: newBrokerForTest(),
		clock:  func() time.Time { return fixed },
	}

	outcome, err := d.RunOnce(context.Background(), "ai-clock")
	if err != nil {
		t.Fatalf("RunOnce() error = %v, want nil", err)
	}
	if !outcome.SpawnedAt.Equal(fixed) {
		t.Fatalf("RunOnce() SpawnedAt = %v, want %v", outcome.SpawnedAt, fixed)
	}
}

// TestRunOnceNilDispatcherReturnsConfigError asserts a nil-receiver call
// surfaces as ErrInvalidDispatcherConfig rather than panicking. This is
// defense-in-depth: NewDispatcher never returns nil with nil error, but
// future refactors might construct dispatcher values through other paths.
func TestRunOnceNilDispatcherReturnsConfigError(t *testing.T) {
	t.Parallel()

	var d *dispatcher
	_, err := d.RunOnce(context.Background(), "ai-1")
	if err == nil {
		t.Fatalf("nil dispatcher RunOnce() error = nil, want non-nil")
	}
	if !errors.Is(err, ErrInvalidDispatcherConfig) {
		t.Fatalf("nil dispatcher RunOnce() error = %v, want errors.Is(ErrInvalidDispatcherConfig)", err)
	}
}

// TestStartReturnsNotImplemented asserts the Start stub returns
// ErrNotImplemented. Drop 4b replaces this with the continuous-mode loop.
func TestStartReturnsNotImplemented(t *testing.T) {
	t.Parallel()

	d := newDispatcherForTest(&stubActionItemReader{})
	err := d.Start(context.Background())
	if !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("Start() error = %v, want errors.Is(ErrNotImplemented)", err)
	}
}

// TestStopReturnsNotImplemented asserts the Stop stub returns
// ErrNotImplemented. Drop 4b replaces this with the continuous-mode loop.
func TestStopReturnsNotImplemented(t *testing.T) {
	t.Parallel()

	d := newDispatcherForTest(&stubActionItemReader{})
	err := d.Stop(context.Background())
	if !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("Stop() error = %v, want errors.Is(ErrNotImplemented)", err)
	}
}

// TestResultEnumValuesAreStable pins the closed Result enum's wire values so
// downstream consumers (CLI output, logs, dashboards) detect any future
// rename as a deliberate edit rather than a silent regression.
func TestResultEnumValuesAreStable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		got  Result
		want string
	}{
		{ResultSpawned, "spawned"},
		{ResultSkipped, "skipped"},
		{ResultBlocked, "blocked"},
		{ResultFailed, "failed"},
	}
	for _, tc := range cases {
		if string(tc.got) != tc.want {
			t.Errorf("Result %q = %q, want %q", tc.want, string(tc.got), tc.want)
		}
	}
}

// TestErrInvalidDispatcherConfigWraps confirms the sentinel formats with
// fmt.Errorf-style %w wrapping for callers that pattern-match on the wrapped
// reason string.
func TestErrInvalidDispatcherConfigWraps(t *testing.T) {
	t.Parallel()

	wrapped := fmt.Errorf("%w: extra context", ErrInvalidDispatcherConfig)
	if !errors.Is(wrapped, ErrInvalidDispatcherConfig) {
		t.Fatalf("errors.Is(wrapped, ErrInvalidDispatcherConfig) = false, want true")
	}
}

// newDispatcherForTest constructs one dispatcher via the unexported struct
// literal so tests can swap in a stub actionItemReader. Production code
// always goes through NewDispatcher.
func newDispatcherForTest(reader actionItemReader) *dispatcher {
	return &dispatcher{
		svc:    reader,
		broker: newBrokerForTest(),
		clock:  time.Now,
	}
}
