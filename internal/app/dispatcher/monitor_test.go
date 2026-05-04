// Package dispatcher monitor_test exercises the Wave 2.8 process monitor
// against a fake agent binary compiled from testdata/fakeagent.go.
//
// CARVE-OUT (documented loudly): the helper buildFakeAgent invokes the
// `go` toolchain via exec.Command("go", "build", ...) to produce a tmpfile
// agent binary. This is the one explicit exception to the project's
// "never raw `go`" rule, blessed by WAVE_2_PLAN.md §2.8 Q5 and PLAN.md
// §7 line 300. The exception is scoped to test setup ONLY — production
// code in monitor.go does NOT shell out to `go`. Mage tests already shell
// out to `go test` underneath, so the spirit of the rule (no agent / no
// production code path bypasses mage) is preserved.
package dispatcher

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// stubMonitorService is the deterministic test fixture for the process
// monitor. Tests configure the per-action-item state the monitor will
// observe on its refetch + record every Service mutation so assertions can
// pin the dispatcher → service contract directly.
//
// Concurrency: the monitor goroutine drives stub mutations from a separate
// goroutine; mu serializes the test goroutine's reads against the
// goroutine's writes.
type stubMonitorService struct {
	mu sync.Mutex

	// items is the current state map keyed by action-item ID. The monitor
	// refetches via GetActionItem before applying the failed transition;
	// tests pre-seed the map and may mutate entries between Track and Wait
	// to simulate the "agent already moved to complete" race.
	items map[string]domain.ActionItem

	// columns is returned by ListColumns. Tests using crash paths must
	// include a "Failed" column so the monitor can resolve the column ID.
	columns []domain.Column

	// Recorded mutations.
	moveCalls   atomic.Int32
	updateCalls atomic.Int32
	lastMoveID  atomic.Pointer[string]
	lastMoveCol atomic.Pointer[string]
	lastUpdate  atomic.Pointer[domain.ActionItemMetadata]

	// Optional injected errors keyed by method name.
	getErr    error
	moveErr   error
	updateErr error
}

func newStubMonitorService() *stubMonitorService {
	return &stubMonitorService{
		items:   make(map[string]domain.ActionItem),
		columns: canonicalColumns(),
	}
}

func (s *stubMonitorService) seed(item domain.ActionItem) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[item.ID] = item
}

func (s *stubMonitorService) GetActionItem(_ context.Context, actionItemID string) (domain.ActionItem, error) {
	if s.getErr != nil {
		return domain.ActionItem{}, s.getErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[actionItemID]
	if !ok {
		return domain.ActionItem{}, errors.New("stubMonitorService: action item not found")
	}
	return item, nil
}

func (s *stubMonitorService) ListColumns(_ context.Context, _ string, _ bool) ([]domain.Column, error) {
	return s.columns, nil
}

func (s *stubMonitorService) MoveActionItem(_ context.Context, actionItemID, toColumnID string, _ int) (domain.ActionItem, error) {
	s.moveCalls.Add(1)
	idCopy := actionItemID
	colCopy := toColumnID
	s.lastMoveID.Store(&idCopy)
	s.lastMoveCol.Store(&colCopy)
	if s.moveErr != nil {
		return domain.ActionItem{}, s.moveErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[actionItemID]
	if !ok {
		return domain.ActionItem{}, errors.New("stubMonitorService: action item not found")
	}
	item.ColumnID = toColumnID
	item.LifecycleState = domain.StateFailed
	s.items[actionItemID] = item
	return item, nil
}

func (s *stubMonitorService) UpdateActionItem(_ context.Context, in updateActionItemInput) (domain.ActionItem, error) {
	s.updateCalls.Add(1)
	if in.Metadata != nil {
		copy := *in.Metadata
		s.lastUpdate.Store(&copy)
	}
	if s.updateErr != nil {
		return domain.ActionItem{}, s.updateErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[in.ActionItemID]
	if !ok {
		return domain.ActionItem{}, errors.New("stubMonitorService: action item not found")
	}
	if in.Metadata != nil {
		item.Metadata = *in.Metadata
	}
	s.items[in.ActionItemID] = item
	return item, nil
}

// buildFakeAgent compiles testdata/fakeagent.go into a tmpfile binary and
// returns its absolute path. The compile is the documented test-helper
// carve-out from "never raw `go`": see this file's package doc-comment.
//
// The binary is registered with t.Cleanup so each test gets a fresh
// compile (cheap; the source is ~50 lines) and the tmpfile is removed at
// test end. Building once-per-test rather than once-per-package keeps the
// fixture goroutine-safe across t.Parallel callers without a sync.Once
// dance.
func buildFakeAgent(t *testing.T) string {
	t.Helper()
	src := filepath.Join("testdata", "fakeagent.go")
	dir := t.TempDir()
	binName := "fakeagent"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	binPath := filepath.Join(dir, binName)
	cmd := exec.Command("go", "build", "-o", binPath, src)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("buildFakeAgent: go build %s -> %s failed: %v\noutput:\n%s", src, binPath, err, string(out))
	}
	return binPath
}

// seedTodoActionItem returns a domain.ActionItem in StateInProgress (the
// state agents typically sit in while a monitor watches them) with
// canonical column wiring. Tests override fields as needed.
func seedTodoActionItem(id string) domain.ActionItem {
	return domain.ActionItem{
		ID:             id,
		ProjectID:      "proj-monitor",
		LifecycleState: domain.StateInProgress,
		ColumnID:       "col-inprogress",
		Position:       1,
	}
}

// TestMonitorCleanExitMarksNoFailure exercises acceptance §4 (clean-exit
// path): a fake agent that exits 0 produces an outcome with Crashed=false
// and the monitor takes NO action on the action item — the agent owns its
// own terminal-state transition.
func TestMonitorCleanExitMarksNoFailure(t *testing.T) {
	t.Parallel()

	bin := buildFakeAgent(t)
	svc := newStubMonitorService()
	item := seedTodoActionItem("ai-clean")
	svc.seed(item)

	monitor := newProcessMonitor(svc, nil)
	cmd := exec.Command(bin, "exit0")

	h, err := monitor.Track(context.Background(), item.ID, cmd)
	if err != nil {
		t.Fatalf("Track() error = %v, want nil", err)
	}
	defer h.Close()

	outcome, err := h.Wait()
	if err != nil {
		t.Fatalf("Wait() error = %v, want nil", err)
	}
	if outcome.Crashed {
		t.Fatalf("outcome.Crashed = true on clean exit; outcome=%+v", outcome)
	}
	if outcome.ExitCode != 0 {
		t.Fatalf("outcome.ExitCode = %d on clean exit, want 0", outcome.ExitCode)
	}
	if outcome.Signal != "" {
		t.Fatalf("outcome.Signal = %q on clean exit, want empty", outcome.Signal)
	}
	if got := svc.moveCalls.Load(); got != 0 {
		t.Fatalf("svc.moveCalls = %d on clean exit, want 0 (monitor must not touch state)", got)
	}
	if got := svc.updateCalls.Load(); got != 0 {
		t.Fatalf("svc.updateCalls = %d on clean exit, want 0 (monitor must not touch state)", got)
	}
}

// TestMonitorNonZeroExitMarksFailed exercises acceptance §4 (crash path,
// non-zero exit branch): exit 1 → MoveActionItem to failed +
// UpdateActionItem with Outcome=failure and BlockedReason carrying the
// "agent process crashed:" prefix and the exit code.
func TestMonitorNonZeroExitMarksFailed(t *testing.T) {
	t.Parallel()

	bin := buildFakeAgent(t)
	svc := newStubMonitorService()
	item := seedTodoActionItem("ai-exit1")
	svc.seed(item)

	monitor := newProcessMonitor(svc, nil)
	cmd := exec.Command(bin, "exit1")

	h, err := monitor.Track(context.Background(), item.ID, cmd)
	if err != nil {
		t.Fatalf("Track() error = %v, want nil", err)
	}
	defer h.Close()

	outcome, err := h.Wait()
	if err != nil {
		t.Fatalf("Wait() error = %v, want nil (process crashed but service mutations succeeded)", err)
	}
	if !outcome.Crashed {
		t.Fatalf("outcome.Crashed = false on exit-1; outcome=%+v", outcome)
	}
	if outcome.ExitCode != 1 {
		t.Fatalf("outcome.ExitCode = %d on exit-1, want 1", outcome.ExitCode)
	}
	if outcome.Signal != "" {
		t.Fatalf("outcome.Signal = %q on exit-1 (no signal involved), want empty", outcome.Signal)
	}
	if got := svc.moveCalls.Load(); got != 1 {
		t.Fatalf("svc.moveCalls = %d on crash, want 1", got)
	}
	if got := svc.lastMoveCol.Load(); got == nil || *got != "col-failed" {
		gotStr := "<nil>"
		if got != nil {
			gotStr = *got
		}
		t.Fatalf("MoveActionItem column = %q, want col-failed", gotStr)
	}
	if got := svc.updateCalls.Load(); got != 1 {
		t.Fatalf("svc.updateCalls = %d on crash, want 1", got)
	}
	last := svc.lastUpdate.Load()
	if last == nil {
		t.Fatalf("UpdateActionItem metadata not recorded; last update = nil")
	}
	if last.Outcome != "failure" {
		t.Fatalf("metadata.Outcome = %q, want failure", last.Outcome)
	}
	wantPrefix := "agent process crashed:"
	if got := last.BlockedReason; len(got) < len(wantPrefix) || got[:len(wantPrefix)] != wantPrefix {
		t.Fatalf("metadata.BlockedReason = %q, want prefix %q", got, wantPrefix)
	}
	wantContains := "exit code 1"
	if !contains(last.BlockedReason, wantContains) {
		t.Fatalf("metadata.BlockedReason = %q, want it to contain %q", last.BlockedReason, wantContains)
	}
}

// TestMonitorSignalKilledMarksFailed exercises acceptance §4 (crash path,
// signal-kill branch): a hang-mode fake agent killed via Handle.Close
// produces a TerminationOutcome with Crashed=true, ExitCode=-1, and a
// non-empty Signal field (Unix). The action item moves to failed and the
// metadata reason carries the "agent process crashed: signal:" prefix.
//
// Skipped on Windows because the signal-kill semantic is Unix-shaped;
// Drop 4a does not target Windows for the cascade dispatcher.
func TestMonitorSignalKilledMarksFailed(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("signal-kill semantics are Unix-shaped; Drop 4a dispatcher is Unix-only")
	}

	bin := buildFakeAgent(t)
	svc := newStubMonitorService()
	item := seedTodoActionItem("ai-killed")
	svc.seed(item)

	monitor := newProcessMonitor(svc, nil)
	cmd := exec.Command(bin, "hang")

	h, err := monitor.Track(context.Background(), item.ID, cmd)
	if err != nil {
		t.Fatalf("Track() error = %v, want nil", err)
	}

	// Give the fake agent a moment to actually start sleeping before we
	// kill it; otherwise the test races against the binary's startup
	// path and the kill can land before cmd.Start has produced a Process
	// (it can't — cmd.Start is synchronous — but the kernel still needs a
	// scheduling tick to put the process into the sleep syscall).
	time.Sleep(50 * time.Millisecond)
	h.Close() // sends SIGKILL via cmd.Process.Kill

	outcome, err := h.Wait()
	if err != nil {
		t.Fatalf("Wait() error = %v, want nil (signal-kill but service mutations succeeded)", err)
	}
	if !outcome.Crashed {
		t.Fatalf("outcome.Crashed = false on signal-kill; outcome=%+v", outcome)
	}
	if outcome.ExitCode != -1 {
		t.Fatalf("outcome.ExitCode = %d on signal-kill, want -1", outcome.ExitCode)
	}
	if outcome.Signal == "" {
		t.Fatalf("outcome.Signal is empty on signal-kill; want a signal name like %q", "killed")
	}
	if got := svc.moveCalls.Load(); got != 1 {
		t.Fatalf("svc.moveCalls = %d on signal-kill, want 1", got)
	}
	last := svc.lastUpdate.Load()
	if last == nil {
		t.Fatalf("UpdateActionItem metadata not recorded; last update = nil")
	}
	if last.Outcome != "failure" {
		t.Fatalf("metadata.Outcome = %q, want failure", last.Outcome)
	}
	wantPrefix := "agent process crashed: signal:"
	if got := last.BlockedReason; len(got) < len(wantPrefix) || got[:len(wantPrefix)] != wantPrefix {
		t.Fatalf("metadata.BlockedReason = %q, want prefix %q", got, wantPrefix)
	}
}

// TestMonitorTracksDurationAccurately exercises acceptance §4 (Duration
// tracking): a fake agent that sleeps 100ms produces an outcome whose
// Duration is at least 100ms. Upper bound is loose because process startup
// + scheduling can add tens of milliseconds on a busy CI runner.
func TestMonitorTracksDurationAccurately(t *testing.T) {
	t.Parallel()

	bin := buildFakeAgent(t)
	svc := newStubMonitorService()
	item := seedTodoActionItem("ai-duration")
	svc.seed(item)

	monitor := newProcessMonitor(svc, nil)
	cmd := exec.Command(bin, "sleep", "100")

	h, err := monitor.Track(context.Background(), item.ID, cmd)
	if err != nil {
		t.Fatalf("Track() error = %v, want nil", err)
	}
	defer h.Close()

	outcome, err := h.Wait()
	if err != nil {
		t.Fatalf("Wait() error = %v, want nil", err)
	}
	if outcome.Duration < 100*time.Millisecond {
		t.Fatalf("outcome.Duration = %s, want >= 100ms (sleep mode pinned at 100ms)", outcome.Duration)
	}
	if outcome.Crashed {
		t.Fatalf("outcome.Crashed = true on sleep+exit-0; outcome=%+v", outcome)
	}
}

// TestMonitorStateConflictGuardSkipsCompleteItem exercises acceptance §5
// (state-conflict guard): if the action item is already in StateComplete
// when the monitor refetches (the agent self-updated before the process
// exit was observed), the monitor logs but does NOT call MoveActionItem
// or UpdateActionItem.
func TestMonitorStateConflictGuardSkipsCompleteItem(t *testing.T) {
	t.Parallel()

	bin := buildFakeAgent(t)
	svc := newStubMonitorService()
	item := seedTodoActionItem("ai-already-complete")
	// Pre-seed the action item AS IF the agent already moved it to
	// complete before its process exited. The fake-agent crash (exit 1)
	// hits the monitor's refetch — the refetch returns this state — and
	// the guard short-circuits.
	item.LifecycleState = domain.StateComplete
	item.ColumnID = "col-complete"
	svc.seed(item)

	monitor := newProcessMonitor(svc, nil)
	cmd := exec.Command(bin, "exit1")

	h, err := monitor.Track(context.Background(), item.ID, cmd)
	if err != nil {
		t.Fatalf("Track() error = %v, want nil", err)
	}
	defer h.Close()

	outcome, err := h.Wait()
	if err != nil {
		t.Fatalf("Wait() error = %v, want nil (state-conflict-guard short-circuit)", err)
	}
	if !outcome.Crashed {
		t.Fatalf("outcome.Crashed = false on exit-1; outcome=%+v (process state recording is independent of the action-item guard)", outcome)
	}
	if got := svc.moveCalls.Load(); got != 0 {
		t.Fatalf("svc.moveCalls = %d on already-complete guard, want 0 (monitor must NOT downgrade)", got)
	}
	if got := svc.updateCalls.Load(); got != 0 {
		t.Fatalf("svc.updateCalls = %d on already-complete guard, want 0", got)
	}
}

// TestMonitorTrackRejectsInvalidInput pins the input-guard surface so
// future regressions on actionItemID-empty / cmd-nil fail loudly.
func TestMonitorTrackRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	monitor := newProcessMonitor(newStubMonitorService(), nil)

	if _, err := monitor.Track(context.Background(), "", exec.Command("true")); !errors.Is(err, ErrMonitorInvalidInput) {
		t.Errorf("Track(empty id) error = %v, want errors.Is(ErrMonitorInvalidInput)", err)
	}
	if _, err := monitor.Track(context.Background(), "ai-1", nil); !errors.Is(err, ErrMonitorInvalidInput) {
		t.Errorf("Track(nil cmd) error = %v, want errors.Is(ErrMonitorInvalidInput)", err)
	}
}

// TestMonitorTrackRejectsUnstartableCommand asserts that a *exec.Cmd
// whose path does not resolve to a binary is reported as
// ErrMonitorNotStarted, not silently dropped.
func TestMonitorTrackRejectsUnstartableCommand(t *testing.T) {
	t.Parallel()

	monitor := newProcessMonitor(newStubMonitorService(), nil)
	// A path with a NUL byte is guaranteed-unstartable on Unix and Windows.
	cmd := exec.Command("/this/path/does/not/exist/fakeagent-xyzzy-monitor-test")
	_, err := monitor.Track(context.Background(), "ai-unstartable", cmd)
	if err == nil {
		t.Fatalf("Track(unstartable cmd) error = nil, want non-nil")
	}
	if !errors.Is(err, ErrMonitorNotStarted) {
		t.Fatalf("Track(unstartable cmd) error = %v, want errors.Is(ErrMonitorNotStarted)", err)
	}
}

// TestMonitorConcurrentTrackHandlesAreIndependent asserts acceptance §6
// (concurrent-safe): multiple Track calls share the same monitor; each
// returns its own Handle; goroutine leaks are absent (each Handle's
// Wait/Close pair fully reaps the goroutine).
func TestMonitorConcurrentTrackHandlesAreIndependent(t *testing.T) {
	t.Parallel()

	bin := buildFakeAgent(t)
	svc := newStubMonitorService()
	const n = 5
	for i := 0; i < n; i++ {
		svc.seed(seedTodoActionItem(idForIndex(i)))
	}

	monitor := newProcessMonitor(svc, nil)
	handles := make([]*Handle, 0, n)
	for i := 0; i < n; i++ {
		// Mix exit0 and exit1 so the test exercises both branches
		// concurrently. Items i%2==0 exit cleanly; odd-indexed crash.
		mode := "exit0"
		if i%2 == 1 {
			mode = "exit1"
		}
		cmd := exec.Command(bin, mode)
		h, err := monitor.Track(context.Background(), idForIndex(i), cmd)
		if err != nil {
			t.Fatalf("Track(%q) error = %v, want nil", idForIndex(i), err)
		}
		handles = append(handles, h)
	}
	for i, h := range handles {
		outcome, err := h.Wait()
		if err != nil {
			t.Fatalf("Wait(%d) error = %v, want nil", i, err)
		}
		wantCrash := i%2 == 1
		if outcome.Crashed != wantCrash {
			t.Errorf("Wait(%d) outcome.Crashed = %v, want %v; outcome=%+v", i, outcome.Crashed, wantCrash, outcome)
		}
	}
	// Two crashes (indices 1 and 3) → two MoveActionItem calls. Three
	// clean exits → zero. Total moves = 2, total updates = 2.
	if got := svc.moveCalls.Load(); got != 2 {
		t.Fatalf("svc.moveCalls = %d, want 2 (two odd-indexed crashes)", got)
	}
	if got := svc.updateCalls.Load(); got != 2 {
		t.Fatalf("svc.updateCalls = %d, want 2", got)
	}
}

// TestHandleWaitIsIdempotent asserts that calling Wait twice on the same
// Handle returns the same outcome both times — the cmd.Wait result is
// cached and the second call does not block forever.
func TestHandleWaitIsIdempotent(t *testing.T) {
	t.Parallel()

	bin := buildFakeAgent(t)
	svc := newStubMonitorService()
	item := seedTodoActionItem("ai-idempotent-wait")
	svc.seed(item)

	monitor := newProcessMonitor(svc, nil)
	cmd := exec.Command(bin, "exit0")
	h, err := monitor.Track(context.Background(), item.ID, cmd)
	if err != nil {
		t.Fatalf("Track() error = %v, want nil", err)
	}
	defer h.Close()

	first, err := h.Wait()
	if err != nil {
		t.Fatalf("Wait() #1 error = %v, want nil", err)
	}
	second, err := h.Wait()
	if err != nil {
		t.Fatalf("Wait() #2 error = %v, want nil", err)
	}
	if first.ExitCode != second.ExitCode || first.Crashed != second.Crashed {
		t.Fatalf("Wait() returned divergent outcomes: first=%+v second=%+v", first, second)
	}
}

// TestHandleCloseIsIdempotent asserts that multiple Close calls are safe
// and do not panic. The first call kills the process; subsequent calls
// are no-ops via sync.Once.
func TestHandleCloseIsIdempotent(t *testing.T) {
	t.Parallel()

	bin := buildFakeAgent(t)
	svc := newStubMonitorService()
	item := seedTodoActionItem("ai-idempotent-close")
	svc.seed(item)

	monitor := newProcessMonitor(svc, nil)
	cmd := exec.Command(bin, "hang")
	h, err := monitor.Track(context.Background(), item.ID, cmd)
	if err != nil {
		t.Fatalf("Track() error = %v, want nil", err)
	}
	h.Close()
	h.Close() // must not panic
	h.Close() // must not panic
	if _, err := h.Wait(); err != nil {
		t.Fatalf("Wait() after multiple Close error = %v, want nil", err)
	}
}

// idForIndex produces a deterministic action-item ID for the concurrent
// test. Pre-seeded so the stub's GetActionItem can resolve it.
func idForIndex(i int) string {
	return "ai-concurrent-" + string(rune('0'+i))
}

// contains is a tiny strings.Contains stand-in to keep the test imports
// minimal. Behavior identical to strings.Contains for the substrings the
// test uses.
func contains(haystack, needle string) bool {
	if len(needle) == 0 {
		return true
	}
	if len(needle) > len(haystack) {
		return false
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
