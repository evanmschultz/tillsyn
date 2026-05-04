package dispatcher

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// Result is a closed enum classifying one RunOnce outcome.
type Result string

// Result constants enumerate every RunOnce verdict the dispatcher can emit.
// The set is intentionally small; later Wave 2 droplets refine which path
// each one takes (e.g. ResultBlocked is emitted by the conflict detector in
// 2.7) but never add new values.
const (
	// ResultSpawned reports that the dispatcher launched one agent subprocess
	// for the action item. Wave 2.6 wires the actual spawn; the skeleton
	// droplet (2.1) never returns this value.
	ResultSpawned Result = "spawned"
	// ResultSkipped reports that the action item is ineligible for spawn at
	// this moment (not in todo, missing, or otherwise filtered). Skipping is
	// not a failure: the dispatcher will reconsider the item on the next
	// state-change event.
	ResultSkipped Result = "skipped"
	// ResultBlocked reports that a runtime blocker was inserted (sibling
	// overlap on paths or packages, per Wave 2.7) and the action item is
	// waiting on a sibling to complete. The orchestrator receives an
	// attention item alongside this result.
	ResultBlocked Result = "blocked"
	// ResultFailed reports that the dispatcher tried to spawn the action
	// item but the spawn or its monitored process produced an error. The
	// action item is moved to the failed lifecycle state by the monitor
	// (Wave 2.8).
	ResultFailed Result = "failed"
)

// DispatchOutcome captures one RunOnce result for logging, the manual-trigger
// CLI (Wave 2.10), and downstream dashboards.
type DispatchOutcome struct {
	// ActionItemID is the input ID, trimmed and validated.
	ActionItemID string
	// AgentName is the resolved agent variant (e.g. go-builder-agent). Empty
	// when Result is Skipped or the action item has no kind binding.
	AgentName string
	// SpawnedAt is the wall-clock time the dispatcher recorded the outcome.
	// For ResultSkipped this is the moment the skip decision was reached;
	// for ResultSpawned it is the moment immediately before the subprocess
	// is launched.
	SpawnedAt time.Time
	// Result is the closed-enum verdict. See Result constants.
	Result Result
}

// Options carries dispatcher configuration. The struct is intentionally open
// for extension: Wave 2.6, 2.8, and 2.10 each add fields, and Drop 4b adds
// gate-runner / commit-agent fields. Add new fields rather than introduce a
// new constructor variant. Zero-value Options is valid in Wave 2.1.
type Options struct {
	// _ is a placeholder so gofumpt-formatted struct literals using positional
	// composition produce a clear compile error if a caller relies on the
	// field shape. Later droplets replace this with concrete fields.
	_ struct{}
}

// Dispatcher is the cascade dispatcher contract consumed by the manual-trigger
// CLI (Wave 2.10) and Drop 4b's daemon mode. RunOnce is the only method
// fully wired in Wave 2; Start/Stop are stubs that return ErrNotImplemented
// until Drop 4b lands the continuous-mode loop.
type Dispatcher interface {
	// RunOnce evaluates one action item and either skips it, inserts a
	// blocker, or spawns an agent. It is the manual-trigger entry point.
	// RunOnce returns a non-nil error only for unexpected failures
	// (database errors, malformed configuration). Eligibility decisions
	// (todo-only, blocker-cleared) surface as Result values, not errors.
	RunOnce(ctx context.Context, actionItemID string) (DispatchOutcome, error)
	// Start begins the continuous-mode dispatcher loop that subscribes to
	// LiveWaitBroker action-item-changed events and walks the tree on each
	// wakeup. Drop 4b implements this; Wave 2 returns ErrNotImplemented.
	Start(ctx context.Context) error
	// Stop tears down the continuous-mode loop started by Start. Drop 4b
	// implements this; Wave 2 returns ErrNotImplemented.
	Stop(ctx context.Context) error
}

// Sentinel errors exposed by the dispatcher package.
var (
	// ErrNotImplemented is returned by Start and Stop until Drop 4b wires
	// the continuous-mode loop. Callers detect this with errors.Is to
	// distinguish "not yet wired" from real failures.
	ErrNotImplemented = errors.New("dispatcher: continuous-mode not implemented in wave 2")
	// ErrInvalidDispatcherConfig is returned by NewDispatcher when a required
	// dependency is nil. Callers detect this with errors.Is to give the
	// dev a precise misconfiguration message.
	ErrInvalidDispatcherConfig = errors.New("dispatcher: invalid configuration")
)

// actionItemReader is the narrow consumer-side view the dispatcher uses to
// fetch action items. *app.Service satisfies this interface; the indirection
// lets the test suite inject deterministic stubs without standing up a full
// service + repository graph.
type actionItemReader interface {
	GetActionItem(ctx context.Context, actionItemID string) (domain.ActionItem, error)
}

// dispatcher is the concrete implementation. The struct is intentionally
// unexported: callers depend on the Dispatcher interface, and NewDispatcher
// returns *dispatcher so Wave 2 droplets can add methods on the concrete
// type without breaking interface conformance.
//
// Future droplets fill in the zero-value fields documented below:
//   - Wave 2.2 adds the broker subscriber goroutine + event channel.
//   - Wave 2.3 adds fileLockManager.
//   - Wave 2.4 adds packageLockManager.
//   - Wave 2.5 adds treeWalker.
//   - Wave 2.6 adds spawner.
//   - Wave 2.7 adds conflictDetector.
//   - Wave 2.8 adds processMonitor.
//   - Wave 2.9 adds cleanupHook.
type dispatcher struct {
	// svc is the application service the dispatcher reads action items
	// through. Stored as actionItemReader for test-injection symmetry; the
	// production constructor only accepts *app.Service.
	svc actionItemReader
	// broker is the live-wait broker the continuous-mode loop will subscribe
	// to in Wave 2.2. RunOnce does not consume it directly today, but the
	// constructor validates non-nil so misconfiguration surfaces at
	// startup rather than on first state-change event.
	broker app.LiveWaitBroker
	// opts carries forward-compatible configuration. Wave 2.1 leaves it at
	// zero value; later droplets read concrete fields.
	opts Options
	// clock returns the current time for outcome timestamps. Tests inject a
	// fake clock through a non-exported helper (see dispatcher_test.go);
	// production callers get time.Now via the constructor default.
	clock func() time.Time
}

// NewDispatcher constructs a dispatcher. svc and broker MUST be non-nil; opts
// is forward-compatible (zero value is valid in Wave 2.1).
//
// Returns ErrInvalidDispatcherConfig wrapped with the offending dependency
// name when validation fails. The exported function returns the unexported
// *dispatcher type so callers either store it as Dispatcher (interface) or
// receive it via type inference; the package never widens the API surface
// to expose the struct fields directly.
func NewDispatcher(svc *app.Service, broker app.LiveWaitBroker, opts Options) (*dispatcher, error) {
	if svc == nil {
		return nil, fmt.Errorf("%w: svc is nil", ErrInvalidDispatcherConfig)
	}
	if broker == nil {
		return nil, fmt.Errorf("%w: broker is nil", ErrInvalidDispatcherConfig)
	}
	return &dispatcher{
		svc:    svc,
		broker: broker,
		opts:   opts,
		clock:  time.Now,
	}, nil
}

// RunOnce evaluates one action item.
//
// Wave 2.1 implements only the skip paths: empty/missing IDs and items not in
// the todo lifecycle state. Later Wave 2 droplets add the eligibility walk,
// conflict detection, lock acquisition, and spawn.
func (d *dispatcher) RunOnce(ctx context.Context, actionItemID string) (DispatchOutcome, error) {
	if d == nil {
		return DispatchOutcome{}, fmt.Errorf("%w: dispatcher is nil", ErrInvalidDispatcherConfig)
	}
	trimmed := strings.TrimSpace(actionItemID)
	now := d.now()
	outcome := DispatchOutcome{
		ActionItemID: trimmed,
		SpawnedAt:    now,
		Result:       ResultSkipped,
	}

	if trimmed == "" {
		return outcome, nil
	}

	item, err := d.svc.GetActionItem(ctx, trimmed)
	if err != nil {
		// Not-found is the documented skip-trigger from the spec: the
		// dispatcher reconsiders the item on the next state-change event.
		// Any other error (database failure, transport error) bubbles up
		// to the caller — RunOnce is best-effort but does not swallow
		// genuine infrastructure errors.
		if errors.Is(err, app.ErrNotFound) {
			return outcome, nil
		}
		return DispatchOutcome{}, fmt.Errorf("dispatcher: get action item %q: %w", trimmed, err)
	}

	if item.LifecycleState != domain.StateTodo {
		// Action item is in a non-todo state (already in_progress, complete,
		// failed, or archived). Wave 2 dispatcher only spawns from todo;
		// auto-promotion of stuck items is the walker's job (2.5) and
		// happens via a separate code path.
		return outcome, nil
	}

	// Wave 2.1 stops here: the eligibility walk + lock acquisition + spawn
	// land in 2.5 / 2.3 / 2.4 / 2.6. Returning Skipped is correct for the
	// skeleton — the action item is in todo but the dispatcher has no
	// machinery yet to decide whether spawn is safe.
	return outcome, nil
}

// Start begins the continuous-mode dispatcher loop. Drop 4b implements this.
func (d *dispatcher) Start(_ context.Context) error {
	return ErrNotImplemented
}

// Stop tears down the continuous-mode dispatcher loop. Drop 4b implements
// this.
func (d *dispatcher) Stop(_ context.Context) error {
	return ErrNotImplemented
}

// now returns the dispatcher's current time, defaulting to time.Now when no
// clock has been injected.
func (d *dispatcher) now() time.Time {
	if d == nil || d.clock == nil {
		return time.Now()
	}
	return d.clock()
}

// Compile-time assertion that *dispatcher satisfies the Dispatcher interface.
var _ Dispatcher = (*dispatcher)(nil)
