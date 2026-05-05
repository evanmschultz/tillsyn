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
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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

// =============================================================================
// Stream-JSON Monitor tests (F.7-CORE F.7.4)
// =============================================================================
//
// These tests exercise the cross-CLI Monitor against two adapter
// implementations (MockAdapter from mock_adapter_test.go + claudeAdapter from
// the cli_claude package) to prove the seam is multi-adapter ready. They also
// pin the CLI-agnosticism property by reading monitor.go's source bytes and
// asserting no claude-specific event-type literals leaked into the routing.

// captureLogger collects every Monitor log line into an in-memory slice for
// assertion. Goroutine-safe so the Monitor's internal goroutines (none today,
// but defensive against future refactors) cannot race against a test reader.
type captureLogger struct {
	mu    sync.Mutex
	lines []string
}

func (c *captureLogger) Printf(format string, args ...any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lines = append(c.lines, fmt.Sprintf(format, args...))
}

func (c *captureLogger) snapshot() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, len(c.lines))
	copy(out, c.lines)
	return out
}

// TestMonitor_Run_MockAdapterIntegration feeds the recorded
// testdata/mock_stream_minimal.jsonl fixture through the Monitor wired to a
// MockAdapter. Asserts every event reaches the sink in order and the terminal
// report carries the fixture's cost / denial / reason / errors.
func TestMonitor_Run_MockAdapterIntegration(t *testing.T) {
	t.Parallel()

	fixturePath := filepath.Join("testdata", "mock_stream_minimal.jsonl")
	f, err := os.Open(fixturePath)
	if err != nil {
		t.Fatalf("open fixture %q: %v", fixturePath, err)
	}
	t.Cleanup(func() { _ = f.Close() })

	adapter := newMockAdapter()
	sink := make(chan StreamEvent, 16)
	logger := &captureLogger{}

	monitor := NewMonitor(adapter, f, sink, logger)
	report, err := monitor.Run(context.Background())
	if err != nil {
		t.Fatalf("Run returned err = %v; want nil", err)
	}
	close(sink)

	// Drain sink; expect 3 events.
	var events []StreamEvent
	for ev := range sink {
		events = append(events, ev)
	}
	if len(events) != 3 {
		t.Fatalf("sink received %d events; want 3", len(events))
	}
	if events[0].Type != "mock_chunk" || events[0].Text != "hello" {
		t.Errorf("event[0] = %+v; want mock_chunk text=hello", events[0])
	}
	if events[1].Type != "mock_chunk" || events[1].Text != "world" {
		t.Errorf("event[1] = %+v; want mock_chunk text=world", events[1])
	}
	if !events[2].IsTerminal {
		t.Errorf("event[2].IsTerminal = false; want true")
	}

	// Terminal report from fixture line 3.
	if report.Cost == nil {
		t.Fatalf("report.Cost = nil; want non-nil")
	}
	if *report.Cost != 0.5 {
		t.Errorf("*report.Cost = %v; want 0.5", *report.Cost)
	}
	if report.Reason != "ok" {
		t.Errorf("report.Reason = %q; want %q", report.Reason, "ok")
	}
	if len(report.Denials) != 1 {
		t.Fatalf("len(report.Denials) = %d; want 1", len(report.Denials))
	}
	if report.Denials[0].ToolName != "Bash" {
		t.Errorf("Denials[0].ToolName = %q; want Bash", report.Denials[0].ToolName)
	}

	// No malformed-line warnings expected from a clean fixture.
	if len(logger.snapshot()) != 0 {
		t.Errorf("logger captured warnings on clean fixture: %v", logger.snapshot())
	}
}

// TestMonitor_Run_ClaudeAdapterIntegration feeds the cli_claude package's
// recorded testdata fixture through the Monitor wired to the real
// claudeAdapter. Proves no regression vs droplet 4c.F.7.17.3 stream parsing
// AND that the CLI-agnostic Monitor handles claude-shaped events without
// special-casing.
func TestMonitor_Run_ClaudeAdapterIntegration(t *testing.T) {
	t.Parallel()

	fixturePath := filepath.Join("cli_claude", "testdata", "claude_stream_minimal.jsonl")
	f, err := os.Open(fixturePath)
	if err != nil {
		t.Fatalf("open fixture %q: %v", fixturePath, err)
	}
	t.Cleanup(func() { _ = f.Close() })

	adapter, ok := lookupAdapter(CLIKindClaude)
	if !ok {
		t.Fatalf("lookupAdapter(CLIKindClaude) = (_, false); claude adapter must be registered via cli_claude side-effect import")
	}
	sink := make(chan StreamEvent, 16)
	logger := &captureLogger{}

	monitor := NewMonitor(adapter, f, sink, logger)
	report, err := monitor.Run(context.Background())
	if err != nil {
		t.Fatalf("Run returned err = %v; want nil", err)
	}
	close(sink)

	var events []StreamEvent
	for ev := range sink {
		events = append(events, ev)
	}
	// claude_stream_minimal.jsonl has 4 lines: system+init, assistant,
	// user, result. The Monitor canonicalizes via the adapter; we count
	// one terminal among them.
	if len(events) != 4 {
		t.Fatalf("sink received %d events; want 4", len(events))
	}
	terminals := 0
	for _, ev := range events {
		if ev.IsTerminal {
			terminals++
		}
	}
	if terminals != 1 {
		t.Errorf("terminal-event count = %d; want 1", terminals)
	}

	if report.Cost == nil {
		t.Fatalf("report.Cost = nil; want non-nil")
	}
	if *report.Cost != 0.0123 {
		t.Errorf("*report.Cost = %v; want 0.0123", *report.Cost)
	}
	if report.Reason != "completed" {
		t.Errorf("report.Reason = %q; want %q", report.Reason, "completed")
	}
	if len(report.Denials) != 1 {
		t.Fatalf("len(report.Denials) = %d; want 1", len(report.Denials))
	}
	if report.Denials[0].ToolName != "Bash" {
		t.Errorf("Denials[0].ToolName = %q; want Bash", report.Denials[0].ToolName)
	}
}

// TestMonitor_Source_NoCLISpecificEventLiterals is the load-bearing
// CLI-agnosticism guard. Reads monitor.go's source bytes from disk and
// asserts NONE of the forbidden claude-specific wire-format literals appear.
// This regression-pins the F.7-CORE F.7.4 + master PLAN.md L11 invariant
// across future refactors.
func TestMonitor_Source_NoCLISpecificEventLiterals(t *testing.T) {
	t.Parallel()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed — cannot locate monitor.go")
	}
	monitorPath := filepath.Join(filepath.Dir(thisFile), "monitor.go")

	bytesRead, err := os.ReadFile(monitorPath)
	if err != nil {
		t.Fatalf("read %q: %v", monitorPath, err)
	}
	src := string(bytesRead)

	forbidden := []string{
		// The exact strings the F.7.4 spec calls out as banned literals
		// in the monitor source. Each represents a claude-specific
		// wire-format token that, if present in monitor.go, would mean
		// the routing is leaking adapter-internal knowledge.
		`system/init`,
		`"assistant"`,
		`"result"`,
	}
	for _, lit := range forbidden {
		if strings.Contains(src, lit) {
			t.Errorf("monitor.go contains forbidden CLI-specific literal %q (Monitor must route via StreamEvent.Type + IsTerminal only)", lit)
		}
	}
}

// TestMonitor_Run_MalformedLineLoggedAndSkipped feeds a stream with one
// invalid-JSON line interleaved between valid events. The Monitor must log a
// warning and continue past the malformed line; the terminal report from the
// last valid event must still be returned.
func TestMonitor_Run_MalformedLineLoggedAndSkipped(t *testing.T) {
	t.Parallel()

	stream := `{"type":"mock_chunk","text":"first"}
{not valid json
{"type":"mock_terminal","cost":0.25,"reason":"done"}
`
	adapter := newMockAdapter()
	logger := &captureLogger{}
	sink := make(chan StreamEvent, 16)

	monitor := NewMonitor(adapter, strings.NewReader(stream), sink, logger)
	report, err := monitor.Run(context.Background())
	if err != nil {
		t.Fatalf("Run returned err = %v; want nil (malformed lines are non-fatal)", err)
	}
	close(sink)

	var events []StreamEvent
	for ev := range sink {
		events = append(events, ev)
	}
	if len(events) != 2 {
		t.Fatalf("forwarded events = %d; want 2 (malformed line skipped)", len(events))
	}

	if report.Cost == nil || *report.Cost != 0.25 {
		t.Errorf("report.Cost = %v; want 0.25", report.Cost)
	}

	logs := logger.snapshot()
	foundWarning := false
	for _, line := range logs {
		if strings.Contains(line, "skip malformed") {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Errorf("logger did NOT capture malformed-line warning; got logs = %v", logs)
	}
}

// TestMonitor_Run_EmptyLinesSkippedSilently asserts blank lines / whitespace
// padding around real events flow through without producing parse warnings or
// sink entries.
func TestMonitor_Run_EmptyLinesSkippedSilently(t *testing.T) {
	t.Parallel()

	stream := "\n   \n{\"type\":\"mock_chunk\",\"text\":\"only\"}\n\n{\"type\":\"mock_terminal\",\"cost\":0.1,\"reason\":\"done\"}\n\n"
	adapter := newMockAdapter()
	logger := &captureLogger{}
	sink := make(chan StreamEvent, 16)

	monitor := NewMonitor(adapter, strings.NewReader(stream), sink, logger)
	report, err := monitor.Run(context.Background())
	if err != nil {
		t.Fatalf("Run returned err = %v; want nil", err)
	}
	close(sink)

	var events []StreamEvent
	for ev := range sink {
		events = append(events, ev)
	}
	if len(events) != 2 {
		t.Fatalf("forwarded events = %d; want 2 (empty lines skipped)", len(events))
	}
	if report.Cost == nil || *report.Cost != 0.1 {
		t.Errorf("report.Cost = %v; want 0.1", report.Cost)
	}
	if len(logger.snapshot()) != 0 {
		t.Errorf("empty lines emitted warnings: %v", logger.snapshot())
	}
}

// TestMonitor_Run_ContextCancellation cancels the context after the first
// event and asserts Run returns ctx.Err(). Uses a slow reader so the
// cancellation arrives mid-stream.
func TestMonitor_Run_ContextCancellation(t *testing.T) {
	t.Parallel()

	// The reader emits the first line, then blocks indefinitely on the
	// next Read call. Cancellation between iterations short-circuits the
	// loop.
	stream := []byte("{\"type\":\"mock_chunk\",\"text\":\"a\"}\n")
	reader := newBlockingReader(stream)

	ctx, cancel := context.WithCancel(context.Background())

	adapter := newMockAdapter()
	sink := make(chan StreamEvent, 1)
	logger := &captureLogger{}

	monitor := NewMonitor(adapter, reader, sink, logger)

	done := make(chan struct{})
	var (
		report TerminalReport
		err    error
	)
	go func() {
		report, err = monitor.Run(ctx)
		close(done)
	}()

	// Wait for the first event so we know Run is past at least one
	// iteration before we cancel.
	select {
	case <-sink:
	case <-time.After(2 * time.Second):
		cancel()
		<-done
		t.Fatalf("Run never produced first event before cancellation")
	}

	cancel()
	reader.unblock()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("Run did not return within 2s of cancellation")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Run err = %v; want context.Canceled", err)
	}
	if report.Cost != nil {
		t.Errorf("report.Cost = %v; want nil on cancellation", report.Cost)
	}
}

// TestMonitor_Run_MultipleTerminalEventsReturnsLast simulates a (defensive)
// stream containing two terminal events and asserts the Monitor returns the
// LAST one — per memory §6 the canonical claude stream emits exactly one
// terminal event, but the Monitor must not silently swallow trailing
// terminal noise.
func TestMonitor_Run_MultipleTerminalEventsReturnsLast(t *testing.T) {
	t.Parallel()

	stream := `{"type":"mock_chunk","text":"warmup"}
{"type":"mock_terminal","cost":0.1,"reason":"first"}
{"type":"mock_terminal","cost":0.5,"reason":"second"}
`
	adapter := newMockAdapter()
	sink := make(chan StreamEvent, 16)

	monitor := NewMonitor(adapter, strings.NewReader(stream), sink, nil)
	report, err := monitor.Run(context.Background())
	if err != nil {
		t.Fatalf("Run returned err = %v; want nil", err)
	}
	close(sink)

	if report.Cost == nil || *report.Cost != 0.5 {
		t.Errorf("report.Cost = %v; want 0.5 (last terminal wins)", report.Cost)
	}
	if report.Reason != "second" {
		t.Errorf("report.Reason = %q; want %q", report.Reason, "second")
	}

	terminalCount := 0
	for ev := range sink {
		if ev.IsTerminal {
			terminalCount++
		}
	}
	if terminalCount != 2 {
		t.Errorf("forwarded terminal events = %d; want 2 (both events visible to sink)", terminalCount)
	}
}

// TestMonitor_Run_SlowSinkDoesNotBlock pins the non-blocking-send invariant:
// when the sink buffer is full the Monitor drops the event and continues
// (with a debug log), instead of blocking the reader on a stuck consumer.
func TestMonitor_Run_SlowSinkDoesNotBlock(t *testing.T) {
	t.Parallel()

	stream := strings.Repeat(`{"type":"mock_chunk","text":"x"}`+"\n", 16) +
		`{"type":"mock_terminal","cost":0.9,"reason":"slow"}` + "\n"

	// Buffered to 1 slot — ALL subsequent events should drop rather than
	// block the Monitor.
	sink := make(chan StreamEvent, 1)
	adapter := newMockAdapter()
	logger := &captureLogger{}

	monitor := NewMonitor(adapter, strings.NewReader(stream), sink, logger)

	done := make(chan struct{})
	var (
		report TerminalReport
		err    error
	)
	go func() {
		report, err = monitor.Run(context.Background())
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("Run blocked on full sink instead of dropping events")
	}

	if err != nil {
		t.Fatalf("Run returned err = %v; want nil", err)
	}
	if report.Cost == nil || *report.Cost != 0.9 {
		t.Errorf("report.Cost = %v; want 0.9 (terminal report still extracted on slow sink)", report.Cost)
	}

	// The logger should have at least one drop notice.
	logs := logger.snapshot()
	dropFound := false
	for _, line := range logs {
		if strings.Contains(line, "sink full") {
			dropFound = true
			break
		}
	}
	if !dropFound {
		t.Errorf("logger did NOT capture sink-full warning; logs = %v", logs)
	}
}

// TestMonitor_Run_NilConfigRejected asserts the Monitor rejects nil adapter
// or nil reader at Run time with a wrapped ErrInvalidMonitorConfig sentinel
// callers can errors.Is against.
func TestMonitor_Run_NilConfigRejected(t *testing.T) {
	t.Parallel()

	t.Run("nil adapter", func(t *testing.T) {
		t.Parallel()
		monitor := NewMonitor(nil, strings.NewReader(""), nil, nil)
		_, err := monitor.Run(context.Background())
		if !errors.Is(err, ErrInvalidMonitorConfig) {
			t.Errorf("Run err = %v; want wraps ErrInvalidMonitorConfig", err)
		}
	})

	t.Run("nil reader", func(t *testing.T) {
		t.Parallel()
		monitor := NewMonitor(newMockAdapter(), nil, nil, nil)
		_, err := monitor.Run(context.Background())
		if !errors.Is(err, ErrInvalidMonitorConfig) {
			t.Errorf("Run err = %v; want wraps ErrInvalidMonitorConfig", err)
		}
	})

	t.Run("nil monitor", func(t *testing.T) {
		t.Parallel()
		var monitor *Monitor
		_, err := monitor.Run(context.Background())
		if !errors.Is(err, ErrInvalidMonitorConfig) {
			t.Errorf("nil-monitor Run err = %v; want wraps ErrInvalidMonitorConfig", err)
		}
	})
}

// TestMonitor_Run_NilSinkDoesNotPanic ensures the optional sink contract:
// passing nil for the sink argument flows events through extraction without a
// nil-channel send (which would panic).
func TestMonitor_Run_NilSinkDoesNotPanic(t *testing.T) {
	t.Parallel()

	stream := `{"type":"mock_chunk","text":"x"}
{"type":"mock_terminal","cost":0.42,"reason":"ok"}
`
	adapter := newMockAdapter()
	monitor := NewMonitor(adapter, strings.NewReader(stream), nil, nil)
	report, err := monitor.Run(context.Background())
	if err != nil {
		t.Fatalf("Run with nil sink returned err = %v; want nil", err)
	}
	if report.Cost == nil || *report.Cost != 0.42 {
		t.Errorf("report.Cost = %v; want 0.42", report.Cost)
	}
}

// blockingReader is a test double that returns the supplied buffered bytes
// then blocks indefinitely on subsequent Read calls until unblock is called.
// The cancellation test uses it so Monitor.Run is guaranteed to be in the
// scanner loop when the context is cancelled.
type blockingReader struct {
	mu      sync.Mutex
	pending []byte
	gate    chan struct{}
}

func newBlockingReader(initial []byte) *blockingReader {
	return &blockingReader{
		pending: append([]byte(nil), initial...),
		gate:    make(chan struct{}),
	}
}

func (b *blockingReader) Read(p []byte) (int, error) {
	b.mu.Lock()
	if len(b.pending) > 0 {
		n := copy(p, b.pending)
		b.pending = b.pending[n:]
		b.mu.Unlock()
		return n, nil
	}
	b.mu.Unlock()
	<-b.gate
	return 0, io.EOF
}

func (b *blockingReader) unblock() {
	select {
	case <-b.gate:
		// already closed
	default:
		close(b.gate)
	}
}
