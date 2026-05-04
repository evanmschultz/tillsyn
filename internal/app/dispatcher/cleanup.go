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
//  3. Auth-bundle revoke — STUB in Wave 2. The real revoke is filled in by
//     Drop 4c Theme F.7 (spawn pipeline redesign), which replaces the 4a.19
//     spawn stub with a per-spawn temp-bundle architecture and wires real
//     auth-bundle revoke through the till.auth_request(revoke) MCP path. See
//     workflow/drop_4c/SKETCH.md § Theme F.7. The seam is documented loudly
//     so the dev cleaning up after a manual-trigger CLI run today knows
//     credentials are NOT yet auto revoked; see also spawn.go's AuthBundle
//     stub for the matching seam on the spawn side.
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
	// revokeAuthBundle runs third. Wave 2 wires this to a no-op stub (see
	// the package-level method below); Drop 4c Theme F.7 (spawn pipeline
	// redesign) replaces the stub with a till.auth_request(revoke) call
	// against the action item's session/lease. The function-typed shape lets
	// the wiring change without touching cleanupHook.
	revokeAuthBundle func(actionItemID string) error
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

// newCleanupHook constructs a cleanupHook bound to the dispatcher's lock
// managers and process monitor. All three dependencies MUST be non-nil;
// callers wire the production instances via the dispatcher constructor in
// droplet 4a.23. The auth-bundle revoke seam is wired to the package-level
// stub (revokeAuthBundleStub) here; Drop 4c Theme F.7 swaps it for the real
// revoke without touching the call site.
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
func newCleanupHook(fileLocks *fileLockManager, pkgLocks *packageLockManager, monitor monitorUnsubscriber) (*cleanupHook, error) {
	if fileLocks == nil {
		return nil, errInvalidCleanupDep("fileLocks")
	}
	if pkgLocks == nil {
		return nil, errInvalidCleanupDep("pkgLocks")
	}
	if monitor == nil {
		return nil, errInvalidCleanupDep("monitor")
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
		revokeAuthBundle: revokeAuthBundleStub,
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
func (c *cleanupHook) OnTerminalState(_ context.Context, item domain.ActionItem) error {
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
		if err := c.revokeAuthBundle(item.ID); err != nil {
			errs = append(errs, err)
		}
	}
	if c.unsubscribeMonitor != nil {
		c.unsubscribeMonitor(item.ID)
	}
	return errors.Join(errs...)
}

// revokeAuthBundleStub is the Wave-2 placeholder for the auth-bundle revoke
// step. The body is intentionally empty because Wave 2 has no real auth
// surface to revoke — droplet 4a.19 ships the AuthBundle{} stub on the spawn
// side, and the symmetric stub here closes the loop on the cleanup side.
//
// Drop 4c Theme F.7 (spawn pipeline redesign) fills this in: it replaces the
// 4a.19 spawn stub with a per-spawn temp-bundle architecture and wires this
// stub to a till.auth_request(operation=revoke) call against the action
// item's session and lease, surfacing the revoke error through the
// cleanupHook's errors.Join aggregation. Until then the dev cleaning up
// after a manual-trigger CLI run revokes manually via `till auth_request
// revoke` — see WAVE_2_PLAN.md §2.9 mitigation paragraph and
// workflow/drop_4c/SKETCH.md § Theme F.7.
func revokeAuthBundleStub(_ string) error {
	// Drop 4c Theme F.7 fills this in.
	return nil
}
