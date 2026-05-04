package dispatcher

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// stubMonitorUnsubscriber is the deterministic test fixture for the
// process-monitor unsubscribe step. It records every call so tests can pin
// the exact action-item IDs flowing through cleanup.
type stubMonitorUnsubscriber struct {
	mu    sync.Mutex
	calls []string
}

// Unsubscribe records the call and never errors — the production interface
// is fire-and-forget per cleanup.go's monitorUnsubscriber doc-comment.
func (s *stubMonitorUnsubscriber) Unsubscribe(actionItemID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, actionItemID)
}

// gotCalls returns a copy of the recorded call list under the mutex so test
// assertions cannot race the recorder.
func (s *stubMonitorUnsubscriber) gotCalls() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.calls))
	copy(out, s.calls)
	return out
}

// TestCleanupReleasesFileAndPackageLocks asserts the happy-path baseline:
// after the dispatcher pre-acquires file + package locks for an action item,
// OnTerminalState frees BOTH locks so a sibling can acquire them on the
// next attempt. The test pins the lock-manager state through a follow-up
// Acquire rather than reaching into private maps, mirroring locks_file_test
// and locks_package_test.
func TestCleanupReleasesFileAndPackageLocks(t *testing.T) {
	t.Parallel()

	fileLocks := newFileLockManager()
	pkgLocks := newPackageLockManager()
	monitor := &stubMonitorUnsubscriber{}

	hook, err := newCleanupHook(fileLocks, pkgLocks, monitor)
	if err != nil {
		t.Fatalf("newCleanupHook: %v", err)
	}

	if _, _, err := fileLocks.Acquire("item-1", []string{"a.go"}); err != nil {
		t.Fatalf("fileLocks.Acquire: %v", err)
	}
	if _, _, err := pkgLocks.Acquire("item-1", []string{"internal/app"}); err != nil {
		t.Fatalf("pkgLocks.Acquire: %v", err)
	}

	item := domain.ActionItem{
		ID:             "item-1",
		LifecycleState: domain.StateComplete,
	}
	if err := hook.OnTerminalState(context.Background(), item); err != nil {
		t.Fatalf("OnTerminalState: %v", err)
	}

	// A sibling action item must now be able to acquire both locks.
	acquiredFile, conflictsFile, err := fileLocks.Acquire("item-2", []string{"a.go"})
	if err != nil {
		t.Fatalf("fileLocks.Acquire after release: %v", err)
	}
	if len(conflictsFile) != 0 {
		t.Fatalf("expected zero file conflicts after release, got %v", conflictsFile)
	}
	if len(acquiredFile) != 1 || acquiredFile[0] != "a.go" {
		t.Fatalf("expected file acquired=[a.go], got %v", acquiredFile)
	}

	acquiredPkg, conflictsPkg, err := pkgLocks.Acquire("item-2", []string{"internal/app"})
	if err != nil {
		t.Fatalf("pkgLocks.Acquire after release: %v", err)
	}
	if len(conflictsPkg) != 0 {
		t.Fatalf("expected zero package conflicts after release, got %v", conflictsPkg)
	}
	if len(acquiredPkg) != 1 || acquiredPkg[0] != "internal/app" {
		t.Fatalf("expected package acquired=[internal/app], got %v", acquiredPkg)
	}

	// Monitor unsubscribe must have fired exactly once for item-1.
	gotMonitor := monitor.gotCalls()
	if len(gotMonitor) != 1 || gotMonitor[0] != "item-1" {
		t.Fatalf("expected monitor.Unsubscribe([item-1]), got %v", gotMonitor)
	}
}

// TestCleanupIsIdempotent asserts that calling OnTerminalState a second time
// for the same action item is a no-op: no per-step closure fires, no error
// is returned. Verified by counting closure invocations through atomic
// counters injected via the function-typed seams.
func TestCleanupIsIdempotent(t *testing.T) {
	t.Parallel()

	var fileCalls, pkgCalls, authCalls, monitorCalls atomic.Int32

	hook := &cleanupHook{
		releaseFileLocks: func(_ string) error {
			fileCalls.Add(1)
			return nil
		},
		releasePackageLocks: func(_ string) error {
			pkgCalls.Add(1)
			return nil
		},
		revokeAuthBundle: func(_ string) error {
			authCalls.Add(1)
			return nil
		},
		unsubscribeMonitor: func(_ string) {
			monitorCalls.Add(1)
		},
	}

	item := domain.ActionItem{
		ID:             "item-1",
		LifecycleState: domain.StateComplete,
	}
	if err := hook.OnTerminalState(context.Background(), item); err != nil {
		t.Fatalf("first OnTerminalState: %v", err)
	}
	if err := hook.OnTerminalState(context.Background(), item); err != nil {
		t.Fatalf("second OnTerminalState: %v", err)
	}

	if got := fileCalls.Load(); got != 1 {
		t.Fatalf("releaseFileLocks: want 1 call, got %d", got)
	}
	if got := pkgCalls.Load(); got != 1 {
		t.Fatalf("releasePackageLocks: want 1 call, got %d", got)
	}
	if got := authCalls.Load(); got != 1 {
		t.Fatalf("revokeAuthBundle: want 1 call, got %d", got)
	}
	if got := monitorCalls.Load(); got != 1 {
		t.Fatalf("unsubscribeMonitor: want 1 call, got %d", got)
	}
}

// TestCleanupOnArchivedAlsoFires asserts the contract from WAVE_2_PLAN.md
// §2.9: archive transitions are treated as terminal too. The pipeline runs
// regardless of which terminal state the item entered — the caller is the
// state-filter, OnTerminalState is the runner.
func TestCleanupOnArchivedAlsoFires(t *testing.T) {
	t.Parallel()

	var fileCalls, pkgCalls, authCalls, monitorCalls atomic.Int32

	hook := &cleanupHook{
		releaseFileLocks: func(_ string) error {
			fileCalls.Add(1)
			return nil
		},
		releasePackageLocks: func(_ string) error {
			pkgCalls.Add(1)
			return nil
		},
		revokeAuthBundle: func(_ string) error {
			authCalls.Add(1)
			return nil
		},
		unsubscribeMonitor: func(_ string) {
			monitorCalls.Add(1)
		},
	}

	item := domain.ActionItem{
		ID:             "item-archived",
		LifecycleState: domain.StateArchived,
	}
	if err := hook.OnTerminalState(context.Background(), item); err != nil {
		t.Fatalf("OnTerminalState: %v", err)
	}

	if got := fileCalls.Load(); got != 1 {
		t.Fatalf("releaseFileLocks on archived: want 1 call, got %d", got)
	}
	if got := pkgCalls.Load(); got != 1 {
		t.Fatalf("releasePackageLocks on archived: want 1 call, got %d", got)
	}
	if got := authCalls.Load(); got != 1 {
		t.Fatalf("revokeAuthBundle on archived: want 1 call, got %d", got)
	}
	if got := monitorCalls.Load(); got != 1 {
		t.Fatalf("unsubscribeMonitor on archived: want 1 call, got %d", got)
	}
}

// TestCleanupContinuesPastIndividualFailure asserts the load-bearing
// error-aggregation contract from WAVE_2_PLAN.md §2.9 acceptance criterion:
// when releaseFileLocks errors, the pipeline still attempts releasePackageLocks
// (and revokeAuthBundle, and unsubscribeMonitor), and OnTerminalState returns
// errors.Join(...) over every per-step error.
//
// Today the production lock managers cannot return errors (Release returns
// nothing); the function-typed seams exist so Drop 4b's SQLite-mirror-backed
// Release can surface persistence failures without breaking the cleanup
// contract. This test pins the aggregation shape against that future shape.
func TestCleanupContinuesPastIndividualFailure(t *testing.T) {
	t.Parallel()

	fileErr := errors.New("file-lock release exploded")
	pkgErr := errors.New("package-lock release exploded")
	authErr := errors.New("auth-bundle revoke exploded")

	var pkgCalled, authCalled, monitorCalled atomic.Bool

	hook := &cleanupHook{
		releaseFileLocks: func(_ string) error {
			return fileErr
		},
		releasePackageLocks: func(_ string) error {
			pkgCalled.Store(true)
			return pkgErr
		},
		revokeAuthBundle: func(_ string) error {
			authCalled.Store(true)
			return authErr
		},
		unsubscribeMonitor: func(_ string) {
			monitorCalled.Store(true)
		},
	}

	item := domain.ActionItem{
		ID:             "item-1",
		LifecycleState: domain.StateFailed,
	}
	err := hook.OnTerminalState(context.Background(), item)
	if err == nil {
		t.Fatalf("expected aggregated error, got nil")
	}

	// Every subsequent step must have run despite the first one failing.
	if !pkgCalled.Load() {
		t.Fatalf("releasePackageLocks: expected to be called after file-lock failure")
	}
	if !authCalled.Load() {
		t.Fatalf("revokeAuthBundle: expected to be called after package-lock failure")
	}
	if !monitorCalled.Load() {
		t.Fatalf("unsubscribeMonitor: expected to be called after auth-revoke failure")
	}

	// errors.Join unwraps via errors.Is on each constituent.
	if !errors.Is(err, fileErr) {
		t.Fatalf("expected aggregated err to wrap fileErr, got %v", err)
	}
	if !errors.Is(err, pkgErr) {
		t.Fatalf("expected aggregated err to wrap pkgErr, got %v", err)
	}
	if !errors.Is(err, authErr) {
		t.Fatalf("expected aggregated err to wrap authErr, got %v", err)
	}
}

// TestCleanupEmptyActionItemIDIsNoop asserts the documented edge case from
// OnTerminalState's doc-comment: an empty item.ID short-circuits before any
// per-step closure fires. This prevents the cleanup hook from accidentally
// nuking the empty-string keyhole in the lock managers (which is a valid
// holder ID per the managers' opacity contract — see locks_file.go's Acquire
// doc-comment) when an upstream observer streams an unfiltered event.
func TestCleanupEmptyActionItemIDIsNoop(t *testing.T) {
	t.Parallel()

	var fileCalls atomic.Int32

	hook := &cleanupHook{
		releaseFileLocks: func(_ string) error {
			fileCalls.Add(1)
			return nil
		},
		releasePackageLocks: func(_ string) error { return nil },
		revokeAuthBundle:    func(_ string) error { return nil },
		unsubscribeMonitor:  func(_ string) {},
	}

	item := domain.ActionItem{ID: "", LifecycleState: domain.StateComplete}
	if err := hook.OnTerminalState(context.Background(), item); err != nil {
		t.Fatalf("OnTerminalState empty ID: %v", err)
	}
	if got := fileCalls.Load(); got != 0 {
		t.Fatalf("releaseFileLocks: want 0 calls on empty ID, got %d", got)
	}
}

// TestNewCleanupHookValidatesDependencies asserts the constructor's
// non-nil-dependency guard — the same wrap shape NewDispatcher and
// processMonitor.Track use, so misconfiguration produces a consistent
// "dispatcher: invalid configuration: <dep> is nil" surface.
func TestNewCleanupHookValidatesDependencies(t *testing.T) {
	t.Parallel()

	fileLocks := newFileLockManager()
	pkgLocks := newPackageLockManager()
	monitor := &stubMonitorUnsubscriber{}

	cases := []struct {
		name      string
		fileLocks *fileLockManager
		pkgLocks  *packageLockManager
		monitor   monitorUnsubscriber
		wantSub   string
	}{
		{"nil fileLocks", nil, pkgLocks, monitor, "fileLocks"},
		{"nil pkgLocks", fileLocks, nil, monitor, "pkgLocks"},
		{"nil monitor", fileLocks, pkgLocks, nil, "monitor"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := newCleanupHook(tc.fileLocks, tc.pkgLocks, tc.monitor)
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tc.name)
			}
			// Wrap-shape contract from cleanup.go: errInvalidCleanupDep uses
			// fmt.Errorf("%w: %s is nil", ErrInvalidDispatcherConfig, name) so
			// callers can detect misconfiguration via errors.Is. Mirrors the
			// shape used by NewDispatcher (dispatcher.go:152-158) and
			// processMonitor.Track (monitor.go:235).
			if !errors.Is(err, ErrInvalidDispatcherConfig) {
				t.Fatalf("expected errors.Is(err, ErrInvalidDispatcherConfig) for %s, got err=%v", tc.name, err)
			}
			// Per-dependency tail must still appear in the rendered string so
			// the dev's misconfiguration message names the offending field.
			if got := err.Error(); !strings.Contains(got, tc.wantSub) {
				t.Fatalf("expected error to mention %q, got %q", tc.wantSub, got)
			}
		})
	}
}

// TestRevokeAuthBundleStubReturnsNil pins the Wave-2 contract that the
// stub is a no-op success. Wave 3 will replace the stub with a real revoke;
// at that point this test will need to assert against the populated
// AuthBundle shape instead of the empty-string ID. The test exists so the
// stub replacement is loud rather than silent.
func TestRevokeAuthBundleStubReturnsNil(t *testing.T) {
	t.Parallel()

	if err := revokeAuthBundleStub("any-id"); err != nil {
		t.Fatalf("revokeAuthBundleStub: want nil, got %v", err)
	}
	if err := revokeAuthBundleStub(""); err != nil {
		t.Fatalf("revokeAuthBundleStub empty ID: want nil, got %v", err)
	}
}
