package dispatcher

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// Wave 2.9 terminal-state cleanup contract overview.
//
// cleanupHook owns the deterministic teardown that runs whenever an action
// item enters a terminal lifecycle state (StateComplete, StateFailed, or
// StateArchived). Its job is to release every in-process resource the
// dispatcher attached to the item over the course of one spawn:
//
//  1. File-level locks acquired via fileLockManager (Wave 2.3 / droplet 4a.16).
//  2. Package-level locks acquired via packageLockManager (Wave 2.4 / droplet 4a.17).
//  3. Auth-bundle revoke — wired in Drop 4b.5 to Service.RevokeSessionForActionItem
//     (internal/app/auth_requests.go). The closure passed by NewDispatcher
//     iterates every active session whose ApprovedPath resolves to the
//     terminal action item's ID, calls authBackend.RevokeAuthSession on
//     each, AND cascades to repo.RevokeCapabilityLeasesByScope for each
//     matching scope tuple. Multi-session retries (fix-builder cycles)
//     are revoked exhaustively per WC-A2; lease cascade is explicit
//     because RevokeAuthSession does NOT touch capability leases per
//     WC-A1. Drop 4c Theme F.7 (spawn pipeline redesign) replaces the
//     temp-bundle architecture but leaves this seam intact.
//  4. Process-monitor unsubscribe — removes the item's entry from the
//     monitor's tracked-PID map. By the time cleanup runs the process has
//     already exited (the monitor's runHandle goroutine is what drives the
//     terminal-state transition in the crash path, and the agent self-updates
//     in the clean-exit path AFTER the process exits), so this step does NOT
//     kill anything. It is a defensive scrub of the dashboard-facing tracked
//     map only.
//
// All four steps run regardless of individual failures (errors aggregated via
// errors.Join). The lock managers' Release methods cannot fail today (they
// return no error), so failure surfaces only when a future
// SQLite-mirror-backed Release introduced in Drop 4b returns an error from
// the persistence layer; tests inject erroring releasers via the function-
// typed fields on cleanupHook to exercise the aggregation path.
//
// Idempotency: cleanupHook tracks every action-item ID it has cleaned up in a
// mutex-guarded set. The second OnTerminalState call for the same item is a
// no-op — no releases re-fire, no auth-revoke retries, no error returned.
// The set grows unbounded over the dispatcher's lifetime, which is acceptable
// in Wave 2 (manual-trigger CLI process exits between runs); Drop 4b will
// either bound the set with an LRU or wire cleanup off a per-spawn arena
// that drops the entry when the daemon evicts the action item from its live
// graph.
//
// Wiring deferred to droplet 4a.23: this droplet only ships cleanupHook +
// the function-typed seams for testing. The CLI bootstrap droplet (4a.23)
// constructs the production cleanupHook from the dispatcher's
// fileLockManager, packageLockManager, and processMonitor instances via
// newCleanupHook below. Editing dispatcher.go is out of scope here.

// monitorUnsubscriber is the consumer-side narrow view the cleanup hook uses
// to scrub a finished action item from the process monitor's tracked-PID
// map. *processMonitor will satisfy this interface once droplet 4a.23 adds
// the Unsubscribe method during constructor wiring; today the test suite
// injects a deterministic stub via the function-typed monitorUnsubscribe
// field on cleanupHook.
//
// Method signature: Unsubscribe is fire-and-forget. The cleanup hook does
// NOT propagate an error from this step because the tracked-PID map is a
// dashboard-facing convenience structure — losing a delete on a transient
// failure is recoverable and never blocks an action-item state transition.
type monitorUnsubscriber interface {
	Unsubscribe(actionItemID string)
}

// cleanupHook is the in-process terminal-state teardown coordinator. It is
// constructed once per dispatcher (droplet 4a.23) and shared across all
// terminal transitions; concurrent OnTerminalState calls for distinct action
// items are serialized only on the small idempotency-set critical section,
// not on the per-step releases (which serialize internally per their own
// mutexes).
//
// The struct's seams are function-typed rather than interface-typed so that
// tests can inject erroring releasers directly without writing wrapper
// stubs for every scenario. Production wiring (newCleanupHook) closes over
// the lock managers + monitor concretely; the function-typed shape is the
// test-injection layer.
type cleanupHook struct {
	// releaseFileLocks is invoked first in the cleanup pipeline. Production
	// wiring binds this to fileLockManager.Release (which returns no error
	// today; the func signature lifts to error so Drop 4b's SQLite-mirror
	// can surface persistence failures without a breaking API change). Tests
	// inject erroring closures to exercise the aggregation path.
	releaseFileLocks func(actionItemID string) error
	// releasePackageLocks runs second. Same lift-to-error rationale as
	// releaseFileLocks.
	releasePackageLocks func(actionItemID string) error
	// revokeAuthBundle runs third. Drop 4b.5 wires this to
	// Service.RevokeSessionForActionItem via the authRevoker seam threaded
	// through newCleanupHook. The closure receives the same ctx the
	// terminal-state observer passed into OnTerminalState so backend calls
	// (autent's RevokeAuthSession + sqlite's RevokeCapabilityLeasesByScope)
	// honor the cleanup deadline. The function-typed shape lets the wiring
	// change without touching cleanupHook (Drop 4c Theme F.7's per-spawn
	// temp-bundle architecture rebinds this seam, not the field shape).
	revokeAuthBundle func(ctx context.Context, actionItemID string) error
	// unsubscribeMonitor runs fourth. Production wiring binds this to
	// processMonitor.Unsubscribe (added in droplet 4a.23). Returns no error
	// — see monitorUnsubscriber doc for the rationale.
	unsubscribeMonitor func(actionItemID string)

	// mu guards cleaned. The critical section is bounded by a single map
	// lookup + write per OnTerminalState call.
	mu sync.Mutex
	// cleaned records every action-item ID that has been processed by
	// OnTerminalState. Presence is the signal; values are always struct{}.
	// The set grows unbounded over the dispatcher's lifetime; see the
	// package doc-comment for the Wave-2 acceptability rationale and the
	// Drop-4b bounding plan.
	cleaned map[string]struct{}
}

// actionItemAuthRevoker is the consumer-side narrow view the cleanup hook
// uses to revoke every auth session and capability lease bound to one
// action item on terminal-state cleanup. *app.Service satisfies this
// interface via its RevokeSessionForActionItem method (added in Drop 4b.5);
// the test suite injects a deterministic stub via the function-typed
// revokeAuthBundle field on cleanupHook for direct in-package construction.
//
// Method contract: RevokeSessionForActionItem MUST be idempotent (zero
// matching sessions returns nil) and MUST iterate over EVERY matching
// session — retries / fix-builder cycles can leave multiple sessions tied
// to the same action item. See internal/app/auth_requests.go's
// RevokeSessionForActionItem doc-comment for the full multi-session +
// lease-cascade contract.
type actionItemAuthRevoker interface {
	RevokeSessionForActionItem(ctx context.Context, actionItemID string) error
}

// newCleanupHook constructs a cleanupHook bound to the dispatcher's lock
// managers, process monitor, and auth revoker. All four dependencies MUST
// be non-nil; callers wire the production instances via the dispatcher
// constructor in droplet 4a.23. The auth-bundle revoke seam is wired to
// the supplied authRevoker; Drop 4b.5 lands *app.Service as the production
// implementation. Tests can inject a stub authRevoker directly through
// this constructor or build a cleanupHook struct literal in-package for
// finer control over individual seams.
//
// The constructor lifts the lock managers' no-error Release methods into
// error-returning closures so the cleanupHook's pipeline shape stays
// uniform. Production never produces a non-nil error from these closures
// today; tests inject erroring closures directly (constructing cleanupHook
// as a struct literal in-package) to exercise the aggregation path.
//
// Returns ErrInvalidDispatcherConfig wrapped with the offending dependency
// name when validation fails — the same wrap shape the rest of the package
// uses (NewDispatcher, processMonitor.Track) so the dev sees a consistent
// misconfiguration message.
func newCleanupHook(fileLocks *fileLockManager, pkgLocks *packageLockManager, monitor monitorUnsubscriber, authRevoker actionItemAuthRevoker) (*cleanupHook, error) {
	if fileLocks == nil {
		return nil, errInvalidCleanupDep("fileLocks")
	}
	if pkgLocks == nil {
		return nil, errInvalidCleanupDep("pkgLocks")
	}
	if monitor == nil {
		return nil, errInvalidCleanupDep("monitor")
	}
	if authRevoker == nil {
		return nil, errInvalidCleanupDep("authRevoker")
	}
	return &cleanupHook{
		releaseFileLocks: func(actionItemID string) error {
			fileLocks.Release(actionItemID)
			return nil
		},
		releasePackageLocks: func(actionItemID string) error {
			pkgLocks.Release(actionItemID)
			return nil
		},
		revokeAuthBundle: func(ctx context.Context, actionItemID string) error {
			return authRevoker.RevokeSessionForActionItem(ctx, actionItemID)
		},
		unsubscribeMonitor: func(actionItemID string) {
			monitor.Unsubscribe(actionItemID)
		},
		cleaned: make(map[string]struct{}),
	}, nil
}

// errInvalidCleanupDep wraps ErrInvalidDispatcherConfig with a per-dependency
// reason. Mirrors the wrap shape used by NewDispatcher (dispatcher.go:152-158)
// and processMonitor.Track (monitor.go:235) so the dev sees a consistent
// "dispatcher: invalid configuration: <dep> is nil" string regardless of
// which constructor tripped, AND so callers can detect misconfiguration via
// errors.Is(err, ErrInvalidDispatcherConfig) on every constructor in the
// package.
func errInvalidCleanupDep(name string) error {
	return fmt.Errorf("%w: %s is nil", ErrInvalidDispatcherConfig, name)
}

// OnTerminalState runs the four-step cleanup pipeline for item. The method is
// invoked when an action item transitions to StateComplete, StateFailed, or
// StateArchived; the caller (the dispatcher's transition observer in droplet
// 4a.23) is responsible for filtering on lifecycle state — OnTerminalState
// itself does NOT inspect item.LifecycleState because the contract is
// "treat archive as terminal too" and the pipeline is identical across all
// three terminal states.
//
// Empty item.ID is a no-op: the cleanup pipeline has nothing to scrub for an
// item the dispatcher never tracked. No error is returned in that case so
// callers can drive cleanup off a stream of state-change events without
// pre-filtering.
//
// Idempotency: a second call with the same item.ID is a no-op. The check
// runs under mu and short-circuits before any per-step closure fires, so
// re-entry from a buggy upstream observer cannot accidentally double-release
// a path the next acquirer just claimed.
//
// Error aggregation: every step is attempted regardless of individual
// failures. errors.Join folds non-nil per-step errors into one return value.
// The unsubscribeMonitor step does NOT contribute to the error chain (its
// signature is fire-and-forget; see monitorUnsubscriber doc). The pipeline
// order is fixed: file-lock → package-lock → auth-revoke → monitor-unsub.
// Reordering would change which downstream component sees freed locks first;
// today the order is documented but not load-bearing because all four steps
// always run before OnTerminalState returns.
//
// The ctx is forwarded into revokeAuthBundle so the auth-revoke seam (Drop
// 4b.5: Service.RevokeSessionForActionItem) honors the cleanup deadline.
// Lock release + monitor unsubscribe do not consume ctx because their
// implementations are in-process and synchronous.
func (c *cleanupHook) OnTerminalState(ctx context.Context, item domain.ActionItem) error {
	if c == nil {
		return nil
	}
	if item.ID == "" {
		return nil
	}

	c.mu.Lock()
	if c.cleaned == nil {
		c.cleaned = make(map[string]struct{})
	}
	if _, already := c.cleaned[item.ID]; already {
		c.mu.Unlock()
		return nil
	}
	c.cleaned[item.ID] = struct{}{}
	c.mu.Unlock()

	var errs []error
	if c.releaseFileLocks != nil {
		if err := c.releaseFileLocks(item.ID); err != nil {
			errs = append(errs, err)
		}
	}
	if c.releasePackageLocks != nil {
		if err := c.releasePackageLocks(item.ID); err != nil {
			errs = append(errs, err)
		}
	}
	if c.revokeAuthBundle != nil {
		if err := c.revokeAuthBundle(ctx, item.ID); err != nil {
			errs = append(errs, err)
		}
	}
	if c.unsubscribeMonitor != nil {
		c.unsubscribeMonitor(item.ID)
	}
	return errors.Join(errs...)
}
