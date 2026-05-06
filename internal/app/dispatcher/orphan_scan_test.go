package dispatcher_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/app/dispatcher"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// orphan_scan_test.go covers Drop 4c F.7-CORE F.7.8's OrphanScanner.Scan
// algorithm per spawn architecture memory §8 and the per-item branch table
// in OrphanScanner.Scan's doc-comment. The seven scenarios below exhaust
// every documented branch the scanner walks.
//
// Test scaffolding is rolled inline (stubActionItemReader, stubProcessChecker,
// orphanCaptureLogger) — there is no existing reusable fake for
// ActionItemReader / ProcessChecker / MonitorLogger in the dispatcher
// package, so the test file owns its own minimal stubs. The captureLogger
// shape mirrors monitor_test.go's existing helper pattern (sync.Mutex +
// snapshot()).

// stubActionItemReader is a deterministic ActionItemReader for orphan-scan
// tests. The recorded items slice is returned verbatim; the recorded err is
// returned (when non-nil) instead of items so error-path tests stay
// compact. ListInProgress is goroutine-safe defensively even though the
// scanner is single-threaded today.
type stubActionItemReader struct {
	items []domain.ActionItem
	err   error
}

// ListInProgress satisfies dispatcher.ActionItemReader.
func (s *stubActionItemReader) ListInProgress(_ context.Context) ([]domain.ActionItem, error) {
	if s.err != nil {
		return nil, s.err
	}
	// Return a copy so the scanner's per-iteration mutations cannot leak
	// across tests. (The scanner does not mutate the slice today, but
	// stub-side defensive copies keep tests independent.)
	out := make([]domain.ActionItem, len(s.items))
	copy(out, s.items)
	return out, nil
}

// stubProcessChecker is a programmable PID-liveness oracle. alive maps PID →
// liveness; missing PIDs default to dead (false). The default is "dead" so
// tests assert "alive" branches by explicitly populating the map.
//
// Round-2: stubProcessChecker captures the expectedCmdlineSubstring it was
// queried with (lastCmdline keyed by PID) so tests can assert the scanner
// passes the correct substring per manifest.CLIKind. The bare alive map is
// indexed by PID alone — the cmdline arg is captured but not used to
// modulate the boolean answer at this stub layer (DefaultProcessChecker
// owns the substring-match semantics; this stub is for scenario plumbing).
type stubProcessChecker struct {
	mu          sync.Mutex
	alive       map[int]bool
	lastCmdline map[int]string
}

// IsAlive satisfies dispatcher.ProcessChecker.
func (s *stubProcessChecker) IsAlive(pid int, expectedCmdlineSubstring string) bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	if s.lastCmdline == nil {
		s.lastCmdline = map[int]string{}
	}
	s.lastCmdline[pid] = expectedCmdlineSubstring
	alive := false
	if s.alive != nil {
		alive = s.alive[pid]
	}
	s.mu.Unlock()
	return alive
}

// snapshotLastCmdline returns a copy of the per-PID expectedCmdlineSubstring
// map captured by IsAlive. Tests use this to assert the scanner derives
// the correct substring from manifest.CLIKind.
func (s *stubProcessChecker) snapshotLastCmdline() map[int]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[int]string, len(s.lastCmdline))
	for k, v := range s.lastCmdline {
		out[k] = v
	}
	return out
}

// orphanCaptureLogger is a goroutine-safe MonitorLogger fake that records
// every Printf line. The structure mirrors monitor_test.go's captureLogger
// to keep the in-package test convention uniform.
type orphanCaptureLogger struct {
	mu    sync.Mutex
	lines []string
}

// Printf satisfies dispatcher.MonitorLogger.
func (c *orphanCaptureLogger) Printf(format string, args ...any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lines = append(c.lines, fmt.Sprintf(format, args...))
}

// snapshot returns a copy of the recorded log lines.
func (c *orphanCaptureLogger) snapshot() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, len(c.lines))
	copy(out, c.lines)
	return out
}

// hasLineContaining reports whether any captured log line contains substr.
func (c *orphanCaptureLogger) hasLineContaining(substr string) bool {
	for _, line := range c.snapshot() {
		if strings.Contains(line, substr) {
			return true
		}
	}
	return false
}

// writeManifestForTest materializes a per-item bundle directory under root,
// writes a manifest.json with the supplied PID and CLIKind="claude", and
// returns the bundle root path. Thin wrapper over
// writeManifestForTestWithCLIKind for legacy call sites.
func writeManifestForTest(t *testing.T, item domain.ActionItem, projectRoot string, pid int) string {
	t.Helper()
	return writeManifestForTestWithCLIKind(t, item, projectRoot, pid, "claude")
}

// writeManifestForTestWithCLIKind materializes a per-item bundle directory
// under root, writes a manifest.json with the supplied PID and CLIKind, and
// returns the bundle root path. Uses dispatcher.NewBundle / WriteManifest
// so the on-disk shape stays faithful to the production writer (no
// hand-rolled JSON drift). Round-2 introduces this variant so cmdline-match
// tests can vary the manifest's CLIKind between "claude" / "codex" / "".
func writeManifestForTestWithCLIKind(t *testing.T, item domain.ActionItem, projectRoot string, pid int, cliKind string) string {
	t.Helper()
	bundle, err := dispatcher.NewBundle(item, dispatcher.SpawnTempRootProject, projectRoot)
	if err != nil {
		t.Fatalf("NewBundle: %v", err)
	}
	t.Cleanup(func() { _ = bundle.Cleanup() })
	payload := dispatcher.ManifestMetadata{
		SpawnID:      bundle.SpawnID,
		ActionItemID: item.ID,
		Kind:         item.Kind,
		CLIKind:      cliKind,
		ClaudePID:    pid,
		StartedAt:    bundle.StartedAt,
		Paths:        item.Paths,
	}
	if err := bundle.WriteManifest(payload); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}
	return bundle.Paths.Root
}

// makeItem builds an in_progress action item with the supplied ID and
// SpawnBundlePath. Other fields default to zero values.
func makeItem(id string, bundlePath string) domain.ActionItem {
	return domain.ActionItem{
		ID:             id,
		Kind:           domain.KindBuild,
		LifecycleState: domain.StateInProgress,
		Metadata: domain.ActionItemMetadata{
			SpawnBundlePath: bundlePath,
		},
	}
}

// TestOrphanScannerScan_NoInProgressItems verifies the empty-input branch:
// when ListInProgress returns zero items the scanner returns a nil slice
// and nil error and never calls OnOrphanFound.
func TestOrphanScannerScan_NoInProgressItems(t *testing.T) {
	t.Parallel()

	var callbackCount int
	scanner := &dispatcher.OrphanScanner{
		ActionItems:    &stubActionItemReader{items: nil},
		ProcessChecker: &stubProcessChecker{},
		OnOrphanFound: func(_ context.Context, _ domain.ActionItem, _ string) error {
			callbackCount++
			return nil
		},
	}

	orphans, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan: unexpected error %v", err)
	}
	if len(orphans) != 0 {
		t.Fatalf("orphans = %v, want empty", orphans)
	}
	if callbackCount != 0 {
		t.Fatalf("OnOrphanFound called %d times, want 0", callbackCount)
	}
}

// TestOrphanScannerScan_AllAlive verifies the all-healthy branch: every
// item's recorded PID is alive, so the scanner reaps nothing and never
// calls OnOrphanFound.
func TestOrphanScannerScan_AllAlive(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	itemA := makeItem("ai-alive-A", "")
	itemB := makeItem("ai-alive-B", "")
	itemA.Metadata.SpawnBundlePath = writeManifestForTest(t, itemA, projectRoot, 11111)
	itemB.Metadata.SpawnBundlePath = writeManifestForTest(t, itemB, projectRoot, 22222)

	checker := &stubProcessChecker{alive: map[int]bool{11111: true, 22222: true}}
	var callbackCount int
	scanner := &dispatcher.OrphanScanner{
		ActionItems:    &stubActionItemReader{items: []domain.ActionItem{itemA, itemB}},
		ProcessChecker: checker,
		OnOrphanFound: func(_ context.Context, _ domain.ActionItem, _ string) error {
			callbackCount++
			return nil
		},
	}

	orphans, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan: unexpected error %v", err)
	}
	if len(orphans) != 0 {
		t.Fatalf("orphans = %v, want empty (all PIDs alive)", orphans)
	}
	if callbackCount != 0 {
		t.Fatalf("OnOrphanFound called %d times, want 0", callbackCount)
	}
}

// TestOrphanScannerScan_PassesExpectedCmdlineFromCLIKind verifies the
// PID-reuse guard wiring: OrphanScanner.Scan must derive the expected
// cmdline substring from manifest.CLIKind and pass it to
// ProcessChecker.IsAlive. Today's manifests record CLIKind="claude" so
// the scanner must pass "claude". Future codex bundles must pass
// "codex". Bundles with empty / unknown CLIKind fall back to "" (signal-0
// only — preserves round-1 behaviour for legacy bundles per the
// expectedCmdlineForCLIKind doc-comment).
func TestOrphanScannerScan_PassesExpectedCmdlineFromCLIKind(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()

	// Build three items, one per CLIKind branch. All PIDs are flagged
	// alive in the stub so the scanner takes the alive-skip path; we
	// assert on the captured cmdline argument the stub recorded.
	itemClaude := makeItem("ai-claude", "")
	itemCodex := makeItem("ai-codex", "")
	itemUnknown := makeItem("ai-unknown", "")

	itemClaude.Metadata.SpawnBundlePath = writeManifestForTestWithCLIKind(t, itemClaude, projectRoot, 7001, "claude")
	itemCodex.Metadata.SpawnBundlePath = writeManifestForTestWithCLIKind(t, itemCodex, projectRoot, 7002, "codex")
	itemUnknown.Metadata.SpawnBundlePath = writeManifestForTestWithCLIKind(t, itemUnknown, projectRoot, 7003, "")

	checker := &stubProcessChecker{alive: map[int]bool{7001: true, 7002: true, 7003: true}}
	scanner := &dispatcher.OrphanScanner{
		ActionItems:    &stubActionItemReader{items: []domain.ActionItem{itemClaude, itemCodex, itemUnknown}},
		ProcessChecker: checker,
	}

	orphans, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan: unexpected error %v", err)
	}
	if len(orphans) != 0 {
		t.Fatalf("orphans = %v, want empty (all PIDs flagged alive)", orphans)
	}

	captured := checker.snapshotLastCmdline()
	if got, want := captured[7001], "claude"; got != want {
		t.Fatalf("expected cmdline for PID 7001 (cli_kind=claude) = %q, want %q", got, want)
	}
	if got, want := captured[7002], "codex"; got != want {
		t.Fatalf("expected cmdline for PID 7002 (cli_kind=codex) = %q, want %q", got, want)
	}
	if got, want := captured[7003], ""; got != want {
		t.Fatalf("expected cmdline for PID 7003 (cli_kind=\"\") = %q, want %q (unknown CLIKind opts out of cmdline match)", got, want)
	}
}

// TestOrphanScannerScan_PIDReuseRejectedByCmdlineMismatch is the end-to-end
// Attack-1 acceptance test: a manifest records ClaudePID=N and
// CLIKind="claude", but the live process at PID N is actually an
// unrelated binary (vim, ssh, …). DefaultProcessChecker.IsAlive must
// report not-alive (the live PID's comm doesn't contain "claude") and the
// scanner must classify the action item as orphaned.
//
// We model the "live PID, wrong binary" condition with a custom
// ProcessChecker that returns false whenever expectedCmdlineSubstring is
// "claude" — equivalent to the production cmdline-mismatch verdict from
// `ps -p <pid> -o comm=` reporting a non-claude binary. The test does not
// shell out to ps directly because that would be a unit test of
// DefaultProcessChecker (covered in TestDefaultProcessChecker_LiveProcess
// via the deliberately-wrong substring assertion); this test pins the
// scanner's classification logic.
func TestOrphanScannerScan_PIDReuseRejectedByCmdlineMismatch(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	item := makeItem("ai-pid-reuse", "")
	item.Metadata.SpawnBundlePath = writeManifestForTestWithCLIKind(t, item, projectRoot, 8001, "claude")

	// Custom checker: treats PID 8001 as physically alive (signal-0 would
	// succeed) but the cmdline does NOT match "claude" — exactly the
	// PID-reuse scenario.
	checker := &cmdlineMismatchChecker{livePID: 8001, refusedSubstring: "claude"}

	var (
		callbackItems []domain.ActionItem
		callbackCount int
	)
	scanner := &dispatcher.OrphanScanner{
		ActionItems:    &stubActionItemReader{items: []domain.ActionItem{item}},
		ProcessChecker: checker,
		OnOrphanFound: func(_ context.Context, it domain.ActionItem, _ string) error {
			callbackCount++
			callbackItems = append(callbackItems, it)
			return nil
		},
	}

	orphans, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan: unexpected error %v", err)
	}
	if len(orphans) != 1 || orphans[0] != "ai-pid-reuse" {
		t.Fatalf("orphans = %v, want [ai-pid-reuse] (PID alive but wrong binary → must reap)", orphans)
	}
	if callbackCount != 1 {
		t.Fatalf("OnOrphanFound called %d times, want 1", callbackCount)
	}
	if callbackItems[0].ID != "ai-pid-reuse" {
		t.Fatalf("callback item ID = %q, want %q", callbackItems[0].ID, "ai-pid-reuse")
	}
}

// TestOrphanScannerScan_OneDead verifies the dead-PID branch: among three
// items the middle one's PID is dead. The scanner returns exactly that ID
// and invokes OnOrphanFound exactly once with the matching action item +
// the manifest path computed as <bundlePath>/manifest.json.
func TestOrphanScannerScan_OneDead(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	live1 := makeItem("ai-live-1", "")
	dead := makeItem("ai-dead", "")
	live2 := makeItem("ai-live-2", "")
	live1.Metadata.SpawnBundlePath = writeManifestForTest(t, live1, projectRoot, 1001)
	dead.Metadata.SpawnBundlePath = writeManifestForTest(t, dead, projectRoot, 2002)
	live2.Metadata.SpawnBundlePath = writeManifestForTest(t, live2, projectRoot, 3003)

	checker := &stubProcessChecker{alive: map[int]bool{1001: true, 3003: true}}

	var (
		callbackItems     []domain.ActionItem
		callbackPaths     []string
		callbackCount     int
		callbackCountLock sync.Mutex
	)
	scanner := &dispatcher.OrphanScanner{
		ActionItems:    &stubActionItemReader{items: []domain.ActionItem{live1, dead, live2}},
		ProcessChecker: checker,
		OnOrphanFound: func(_ context.Context, item domain.ActionItem, manifestPath string) error {
			callbackCountLock.Lock()
			defer callbackCountLock.Unlock()
			callbackCount++
			callbackItems = append(callbackItems, item)
			callbackPaths = append(callbackPaths, manifestPath)
			return nil
		},
	}

	orphans, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan: unexpected error %v", err)
	}
	if len(orphans) != 1 {
		t.Fatalf("orphans = %v, want exactly one ID", orphans)
	}
	if orphans[0] != "ai-dead" {
		t.Fatalf("orphans[0] = %q, want %q", orphans[0], "ai-dead")
	}
	if callbackCount != 1 {
		t.Fatalf("OnOrphanFound called %d times, want 1", callbackCount)
	}
	if callbackItems[0].ID != "ai-dead" {
		t.Fatalf("callback item ID = %q, want %q", callbackItems[0].ID, "ai-dead")
	}
	wantManifestPath := filepath.Join(dead.Metadata.SpawnBundlePath, "manifest.json")
	if callbackPaths[0] != wantManifestPath {
		t.Fatalf("callback manifest path = %q, want %q", callbackPaths[0], wantManifestPath)
	}
}

// TestOrphanScannerScan_ManifestMissing verifies the os.ErrNotExist branch:
// the action item declares a SpawnBundlePath but no manifest.json exists
// inside it. The scanner skips silently (no error, no callback) and logs a
// debug line referencing the missing manifest.
func TestOrphanScannerScan_ManifestMissing(t *testing.T) {
	t.Parallel()

	bundlePath := t.TempDir() // exists but contains no manifest.json
	item := makeItem("ai-no-manifest", bundlePath)

	logger := &orphanCaptureLogger{}
	var callbackCount int
	scanner := &dispatcher.OrphanScanner{
		ActionItems:    &stubActionItemReader{items: []domain.ActionItem{item}},
		ProcessChecker: &stubProcessChecker{},
		Logger:         logger,
		OnOrphanFound: func(_ context.Context, _ domain.ActionItem, _ string) error {
			callbackCount++
			return nil
		},
	}

	orphans, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan: unexpected error %v", err)
	}
	if len(orphans) != 0 {
		t.Fatalf("orphans = %v, want empty", orphans)
	}
	if callbackCount != 0 {
		t.Fatalf("OnOrphanFound called %d times, want 0", callbackCount)
	}
	if !logger.hasLineContaining("manifest missing") {
		t.Fatalf("expected log line containing %q, got %v", "manifest missing", logger.snapshot())
	}
	// Sanity: the missing-manifest path is genuinely absent on disk —
	// guards against a test-setup bug that accidentally pre-populates
	// manifest.json and trips the wrong scanner branch.
	if _, statErr := os.Stat(filepath.Join(bundlePath, "manifest.json")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected manifest absent, stat err = %v", statErr)
	}
}

// TestOrphanScannerScan_EmptyBundlePath verifies the empty-SpawnBundlePath
// branch: the action item never had a bundle dispatched (or it was already
// cleaned up). The scanner skips with a "no spawn_bundle_path" warning,
// returns no orphans, and never calls OnOrphanFound.
func TestOrphanScannerScan_EmptyBundlePath(t *testing.T) {
	t.Parallel()

	item := makeItem("ai-no-path", "   ") // whitespace-only also resolves to empty per TrimSpace

	logger := &orphanCaptureLogger{}
	var callbackCount int
	scanner := &dispatcher.OrphanScanner{
		ActionItems:    &stubActionItemReader{items: []domain.ActionItem{item}},
		ProcessChecker: &stubProcessChecker{},
		Logger:         logger,
		OnOrphanFound: func(_ context.Context, _ domain.ActionItem, _ string) error {
			callbackCount++
			return nil
		},
	}

	orphans, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan: unexpected error %v", err)
	}
	if len(orphans) != 0 {
		t.Fatalf("orphans = %v, want empty", orphans)
	}
	if callbackCount != 0 {
		t.Fatalf("OnOrphanFound called %d times, want 0", callbackCount)
	}
	if !logger.hasLineContaining("no spawn_bundle_path") {
		t.Fatalf("expected log line containing %q, got %v", "no spawn_bundle_path", logger.snapshot())
	}
}

// TestOrphanScannerScan_PIDZero verifies the PID==0 branch (spawn architecture
// memory §8 "spawn not yet started, leave alone"): a manifest exists with
// ClaudePID=0. The scanner skips without calling OnOrphanFound or invoking
// ProcessChecker.IsAlive.
func TestOrphanScannerScan_PIDZero(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	item := makeItem("ai-pid-zero", "")
	item.Metadata.SpawnBundlePath = writeManifestForTest(t, item, projectRoot, 0)

	// Sentinel: if the scanner calls IsAlive on a PID==0 manifest the
	// stub records the call and the test asserts it never happens.
	checkerCalls := make(map[int]int)
	checker := &recordingProcessChecker{calls: checkerCalls}

	logger := &orphanCaptureLogger{}
	var callbackCount int
	scanner := &dispatcher.OrphanScanner{
		ActionItems:    &stubActionItemReader{items: []domain.ActionItem{item}},
		ProcessChecker: checker,
		Logger:         logger,
		OnOrphanFound: func(_ context.Context, _ domain.ActionItem, _ string) error {
			callbackCount++
			return nil
		},
	}

	orphans, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan: unexpected error %v", err)
	}
	if len(orphans) != 0 {
		t.Fatalf("orphans = %v, want empty (PID==0 short-circuits)", orphans)
	}
	if callbackCount != 0 {
		t.Fatalf("OnOrphanFound called %d times, want 0", callbackCount)
	}
	if len(checker.calls) != 0 {
		t.Fatalf("ProcessChecker.IsAlive invoked %d times, want 0 (PID==0 short-circuits)", len(checker.calls))
	}
	if !logger.hasLineContaining("zero PID") {
		t.Fatalf("expected log line containing %q, got %v", "zero PID", logger.snapshot())
	}
}

// TestOrphanScannerScan_OnOrphanFoundErrorAggregation verifies that when
// OnOrphanFound errors on one orphan but succeeds on others the scan
// continues to completion, every dead PID still appears in the orphans
// slice, and the returned error wraps the failing-callback error
// (errors.Is matches the sentinel). When two orphans both fail the
// returned error joins both via errors.Join.
func TestOrphanScannerScan_OnOrphanFoundErrorAggregation(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()

	t.Run("single failing callback wraps the sentinel", func(t *testing.T) {
		t.Parallel()
		dead1 := makeItem("ai-dead-1", "")
		dead2 := makeItem("ai-dead-2", "")
		dead3 := makeItem("ai-dead-3", "")
		dead1.Metadata.SpawnBundlePath = writeManifestForTest(t, dead1, projectRoot, 5001)
		dead2.Metadata.SpawnBundlePath = writeManifestForTest(t, dead2, projectRoot, 5002)
		dead3.Metadata.SpawnBundlePath = writeManifestForTest(t, dead3, projectRoot, 5003)

		// All three PIDs missing from the alive map → checker returns false
		// for each → all three flagged orphan.
		checker := &stubProcessChecker{alive: map[int]bool{}}

		sentinel := errors.New("synthetic-callback-error")
		var callbackCount int
		scanner := &dispatcher.OrphanScanner{
			ActionItems:    &stubActionItemReader{items: []domain.ActionItem{dead1, dead2, dead3}},
			ProcessChecker: checker,
			OnOrphanFound: func(_ context.Context, item domain.ActionItem, _ string) error {
				callbackCount++
				if item.ID == "ai-dead-2" {
					return sentinel
				}
				return nil
			},
		}

		orphans, err := scanner.Scan(context.Background())
		if err == nil {
			t.Fatalf("Scan: expected non-nil error from failing callback")
		}
		if !errors.Is(err, sentinel) {
			t.Fatalf("Scan: errors.Is(err, sentinel) = false, err = %v", err)
		}
		if callbackCount != 3 {
			t.Fatalf("OnOrphanFound called %d times, want 3 (scan must continue past failure)", callbackCount)
		}
		if len(orphans) != 3 {
			t.Fatalf("orphans = %v, want 3 entries (failure does NOT remove from result)", orphans)
		}
		// Order MUST match the input order — the scanner walks items
		// linearly and appends post-callback in the doc-comment-described
		// order. Pin the order so future refactors that batch / reorder
		// the loop fail the test loudly.
		wantIDs := []string{"ai-dead-1", "ai-dead-2", "ai-dead-3"}
		for i, want := range wantIDs {
			if orphans[i] != want {
				t.Fatalf("orphans[%d] = %q, want %q", i, orphans[i], want)
			}
		}
	})

	t.Run("two failing callbacks join via errors.Join", func(t *testing.T) {
		t.Parallel()
		deadA := makeItem("ai-dead-A", "")
		deadB := makeItem("ai-dead-B", "")
		deadA.Metadata.SpawnBundlePath = writeManifestForTest(t, deadA, projectRoot, 6001)
		deadB.Metadata.SpawnBundlePath = writeManifestForTest(t, deadB, projectRoot, 6002)

		errA := errors.New("err-a")
		errB := errors.New("err-b")
		scanner := &dispatcher.OrphanScanner{
			ActionItems:    &stubActionItemReader{items: []domain.ActionItem{deadA, deadB}},
			ProcessChecker: &stubProcessChecker{},
			OnOrphanFound: func(_ context.Context, item domain.ActionItem, _ string) error {
				switch item.ID {
				case "ai-dead-A":
					return errA
				case "ai-dead-B":
					return errB
				default:
					return nil
				}
			},
		}

		orphans, err := scanner.Scan(context.Background())
		if err == nil {
			t.Fatalf("Scan: expected non-nil error from two failing callbacks")
		}
		if !errors.Is(err, errA) {
			t.Fatalf("Scan: errors.Is(err, errA) = false, err = %v", err)
		}
		if !errors.Is(err, errB) {
			t.Fatalf("Scan: errors.Is(err, errB) = false, err = %v", err)
		}
		if len(orphans) != 2 {
			t.Fatalf("orphans = %v, want 2 entries", orphans)
		}
	})
}

// TestOrphanScannerScan_NilConfigInputs verifies the explicit nil-guard
// paths: nil ActionItems reader, nil ProcessChecker, and a nil receiver
// all return ErrInvalidOrphanScannerConfig (no panic, no silent succeed).
// Documents the public contract on the OrphanScanner struct doc-comment.
func TestOrphanScannerScan_NilConfigInputs(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		scanner *dispatcher.OrphanScanner
	}{
		{
			name: "nil ActionItems",
			scanner: &dispatcher.OrphanScanner{
				ProcessChecker: &stubProcessChecker{},
			},
		},
		{
			name: "nil ProcessChecker",
			scanner: &dispatcher.OrphanScanner{
				ActionItems: &stubActionItemReader{},
			},
		},
		{
			name:    "nil receiver",
			scanner: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := tc.scanner.Scan(context.Background())
			if err == nil {
				t.Fatalf("Scan: expected ErrInvalidOrphanScannerConfig, got nil")
			}
			if !errors.Is(err, dispatcher.ErrInvalidOrphanScannerConfig) {
				t.Fatalf("errors.Is(err, ErrInvalidOrphanScannerConfig) = false, err = %v", err)
			}
		})
	}
}

// TestDefaultProcessChecker_LiveProcess sanity-checks that
// DefaultProcessChecker.IsAlive's signal-0 path returns true for the
// current test process (always alive) and false for PID==0 / negative
// inputs (defensive short-circuit per the doc-comment). A real-process
// probe over the test's own PID avoids the platform fragility of "spawn a
// child and kill it" while still exercising the os.FindProcess + signal-0
// path.
//
// Round-2 (Attack 1 from F.7.8 QA-Falsification): the cmdline-match guard
// is exercised via the CommLookup injection seam instead of shelling out
// to real `ps` — under -race + macOS sandboxing the real shell-out has
// produced multi-minute hangs that exhaust the test budget. The injected
// stub lets us assert the production algorithm (lookup → trim → contains)
// without paying the platform cost. A separate
// TestDefaultProcessChecker_PsCommLookupReal test could be added later
// when a stable subprocess fixture lands.
func TestDefaultProcessChecker_LiveProcess(t *testing.T) {
	t.Parallel()
	pid := os.Getpid()

	// Empty substring path: signal-0 only, no CommLookup invoked.
	plainChecker := dispatcher.DefaultProcessChecker{
		CommLookup: func(int) (string, error) {
			t.Fatalf("CommLookup must NOT be invoked when expectedCmdlineSubstring is empty")
			return "", nil
		},
	}
	if !plainChecker.IsAlive(pid, "") {
		t.Fatalf("IsAlive(pid, \"\") = false, want true (empty substring opts out of cmdline match)")
	}

	// Cmdline match path: stub CommLookup returns "claude" → matches
	// "claude" substring.
	matchChecker := dispatcher.DefaultProcessChecker{
		CommLookup: func(p int) (string, error) {
			if p != pid {
				t.Fatalf("CommLookup pid = %d, want %d", p, pid)
			}
			return "claude", nil
		},
	}
	if !matchChecker.IsAlive(pid, "claude") {
		t.Fatalf("IsAlive(pid, \"claude\") = false, want true (stub returned %q)", "claude")
	}

	// PID-reuse defense: stub CommLookup returns "vim" (a recycled-PID
	// scenario) — Contains("vim", "claude") is false → IsAlive false.
	// This is the Attack-1 acceptance check.
	mismatchChecker := dispatcher.DefaultProcessChecker{
		CommLookup: func(int) (string, error) { return "vim", nil },
	}
	if mismatchChecker.IsAlive(pid, "claude") {
		t.Fatalf("IsAlive(pid, \"claude\") = true with comm=\"vim\", want false (PID-reuse guard must reject mismatched comm)")
	}

	// CommLookup error → treat as dead.
	errChecker := dispatcher.DefaultProcessChecker{
		CommLookup: func(int) (string, error) { return "", errors.New("synthetic-lookup-error") },
	}
	if errChecker.IsAlive(pid, "claude") {
		t.Fatalf("IsAlive(pid, \"claude\") = true when CommLookup errors, want false")
	}

	// CommLookup returns empty string (race / unobservable PID) → treat
	// as dead. Guards against Contains("", "claude") false-positive.
	emptyChecker := dispatcher.DefaultProcessChecker{
		CommLookup: func(int) (string, error) { return "", nil },
	}
	if emptyChecker.IsAlive(pid, "claude") {
		t.Fatalf("IsAlive(pid, \"claude\") = true when CommLookup returns empty, want false")
	}

	// pid <= 0 short-circuits regardless of substring or CommLookup.
	guardChecker := dispatcher.DefaultProcessChecker{
		CommLookup: func(int) (string, error) {
			t.Fatalf("CommLookup must NOT be invoked when pid <= 0")
			return "", nil
		},
	}
	if guardChecker.IsAlive(0, "claude") {
		t.Fatalf("IsAlive(0, \"claude\") = true, want false")
	}
	if guardChecker.IsAlive(-1, "claude") {
		t.Fatalf("IsAlive(-1, \"claude\") = true, want false")
	}
}

// recordingProcessChecker records every PID it was queried for. Used by
// the PID==0 test to assert the scanner short-circuits before consulting
// the checker.
type recordingProcessChecker struct {
	mu    sync.Mutex
	calls map[int]int
	alive map[int]bool
}

// IsAlive satisfies dispatcher.ProcessChecker and records the query.
func (r *recordingProcessChecker) IsAlive(pid int, _ string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.calls == nil {
		r.calls = map[int]int{}
	}
	r.calls[pid]++
	return r.alive[pid]
}

// cmdlineMismatchChecker models the PID-reuse scenario: signal-0 would
// succeed for livePID (the OS has reused the PID for an unrelated process),
// but the cmdline match against refusedSubstring fails. IsAlive returns
// true only when the queried PID matches livePID AND the requested
// expectedCmdlineSubstring is NOT refusedSubstring. Used by
// TestOrphanScannerScan_PIDReuseRejectedByCmdlineMismatch.
type cmdlineMismatchChecker struct {
	livePID          int
	refusedSubstring string
}

// IsAlive satisfies dispatcher.ProcessChecker.
func (c *cmdlineMismatchChecker) IsAlive(pid int, expectedCmdlineSubstring string) bool {
	if c == nil || pid != c.livePID {
		return false
	}
	if expectedCmdlineSubstring == c.refusedSubstring {
		// PID is physically alive but bound to a different binary —
		// production DefaultProcessChecker would observe `ps -p <pid>
		// -o comm=` returning a non-matching command name.
		return false
	}
	return true
}

// _ keeps time import alive across edits even when unused — guards against
// a future refactor that strips a time-dependent assertion accidentally.
var _ = time.Now
