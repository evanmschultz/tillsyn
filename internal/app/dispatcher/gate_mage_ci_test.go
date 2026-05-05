package dispatcher

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/evanmschultz/tillsyn/internal/templates"
)

// fakeCommandRunner is the test double swapped into defaultCommandRunner via
// withFakeCommandRunner. It returns the configured stdout/stderr/exitCode/err
// tuple verbatim and records the (dir, name, args) it was called with so
// tests can assert the gate forwarded the expected invocation.
type fakeCommandRunner struct {
	stdout   []byte
	stderr   []byte
	exitCode int
	err      error

	gotDir  string
	gotName string
	gotArgs []string
	calls   int
}

// Run implements commandRunner. The recorded gotDir/gotName/gotArgs reflect
// the most recent invocation; call count lets tests assert the gate did or
// did not fall through to the runner on guard-rejection paths.
func (f *fakeCommandRunner) Run(ctx context.Context, dir string, name string, args ...string) ([]byte, []byte, int, error) {
	f.gotDir = dir
	f.gotName = name
	f.gotArgs = args
	f.calls++
	return f.stdout, f.stderr, f.exitCode, f.err
}

// withFakeCommandRunner swaps defaultCommandRunner for the supplied fake for
// the duration of the test, restoring the original via t.Cleanup. Tests do
// NOT call t.Parallel() because the swap mutates package state.
func withFakeCommandRunner(t *testing.T, fake commandRunner) {
	t.Helper()
	original := defaultCommandRunner
	defaultCommandRunner = fake
	t.Cleanup(func() {
		defaultCommandRunner = original
	})
}

// gateMageCIFixtureProject returns a domain.Project with RepoPrimaryWorktree
// populated to a non-empty absolute path so the gate's empty-worktree guard
// passes by default. Tests that exercise the guard pass the zero value.
func gateMageCIFixtureProject() domain.Project {
	return domain.Project{
		ID:                  "proj-mage-ci-1",
		RepoPrimaryWorktree: "/tmp/proj-mage-ci-1",
	}
}

// gateMageCIFixtureItem returns a build-kind action item — gateMageCI does
// not read item fields today, so the value is opaque, but the fixture keeps
// the call site shape consistent with the production runner's invocation.
func gateMageCIFixtureItem() domain.ActionItem {
	return domain.ActionItem{
		ID:   "ai-mage-ci-1",
		Kind: domain.KindBuild,
	}
}

// TestGateMageCIPassesOnZeroExit asserts a clean `mage ci` run yields
// GateStatusPassed with empty Output and nil Err. The fake runner returns
// representative stdout (mage success summary) which the gate must DISCARD
// on the passing branch.
func TestGateMageCIPassesOnZeroExit(t *testing.T) {
	fake := &fakeCommandRunner{
		stdout:   []byte("mage: target ci ok\n"),
		stderr:   nil,
		exitCode: 0,
		err:      nil,
	}
	withFakeCommandRunner(t, fake)

	result := gateMageCI(context.Background(), gateMageCIFixtureItem(), gateMageCIFixtureProject())

	if result.Status != GateStatusPassed {
		t.Fatalf("Status = %q, want %q", result.Status, GateStatusPassed)
	}
	if result.Err != nil {
		t.Fatalf("Err = %v, want nil", result.Err)
	}
	if result.Output != "" {
		t.Fatalf("Output = %q, want empty on pass", result.Output)
	}
	if result.GateName != templates.GateKindMageCI {
		t.Fatalf("GateName = %q, want %q", result.GateName, templates.GateKindMageCI)
	}
	if result.Duration <= 0 {
		t.Fatalf("Duration = %v, want > 0", result.Duration)
	}
	if fake.calls != 1 {
		t.Fatalf("runner.calls = %d, want 1", fake.calls)
	}
	if fake.gotDir != "/tmp/proj-mage-ci-1" {
		t.Fatalf("runner.gotDir = %q, want %q", fake.gotDir, "/tmp/proj-mage-ci-1")
	}
	if fake.gotName != "mage" {
		t.Fatalf("runner.gotName = %q, want %q", fake.gotName, "mage")
	}
	if len(fake.gotArgs) != 1 || fake.gotArgs[0] != "ci" {
		t.Fatalf("runner.gotArgs = %v, want [ci]", fake.gotArgs)
	}
}

// TestGateMageCIFailsOnNonZeroExit asserts a non-zero exit produces
// GateStatusFailed, an Err naming the exit code, and a populated Output
// bounded by tailOutput's MaxGateOutputLines / MaxGateOutputBytes. The fake
// emits 200 lines on stdout and 50 lines on stderr; the resulting Output
// must (a) be non-empty, (b) NOT contain the earliest stdout line, (c)
// retain trailing stderr.
func TestGateMageCIFailsOnNonZeroExit(t *testing.T) {
	var stdout strings.Builder
	for i := 0; i < 200; i++ {
		fmt.Fprintf(&stdout, "stdout line %03d: build step output\n", i)
	}
	var stderr strings.Builder
	for i := 0; i < 50; i++ {
		fmt.Fprintf(&stderr, "stderr line %03d: error frame\n", i)
	}

	fake := &fakeCommandRunner{
		stdout:   []byte(stdout.String()),
		stderr:   []byte(stderr.String()),
		exitCode: 1,
		err:      nil,
	}
	withFakeCommandRunner(t, fake)

	result := gateMageCI(context.Background(), gateMageCIFixtureItem(), gateMageCIFixtureProject())

	if result.Status != GateStatusFailed {
		t.Fatalf("Status = %q, want %q", result.Status, GateStatusFailed)
	}
	if result.Err == nil {
		t.Fatal("Err = nil, want non-nil naming exit code")
	}
	if !strings.Contains(result.Err.Error(), "exit code 1") {
		t.Fatalf("Err = %v, want substring %q", result.Err, "exit code 1")
	}
	if result.Output == "" {
		t.Fatal("Output = empty, want non-empty failure tail")
	}
	// First stdout line MUST have been dropped by line-bounding (200 stdout
	// + 50 stderr = 250 lines, capped at 100).
	if strings.Contains(result.Output, "stdout line 000:") {
		t.Fatal("Output retained earliest stdout line; tailOutput line bound failed")
	}
	// Last stderr line MUST be present (failure tail kept).
	if !strings.Contains(result.Output, "stderr line 049:") {
		t.Fatal("Output dropped last stderr line; failure tail truncated")
	}
	if result.Duration <= 0 {
		t.Fatalf("Duration = %v, want > 0", result.Duration)
	}
}

// TestGateMageCIFailsOnStartError asserts a runner-reported start failure
// (mage not on PATH, worktree dir missing, etc.) produces GateStatusFailed
// with Err wrapping the underlying error under the "mage ci start failed:"
// prefix so callers can grep the failure mode.
func TestGateMageCIFailsOnStartError(t *testing.T) {
	startErr := errors.New("exec: \"mage\": executable file not found in $PATH")
	fake := &fakeCommandRunner{
		err: startErr,
	}
	withFakeCommandRunner(t, fake)

	result := gateMageCI(context.Background(), gateMageCIFixtureItem(), gateMageCIFixtureProject())

	if result.Status != GateStatusFailed {
		t.Fatalf("Status = %q, want %q", result.Status, GateStatusFailed)
	}
	if result.Err == nil {
		t.Fatal("Err = nil, want non-nil start-failure error")
	}
	if !errors.Is(result.Err, startErr) {
		t.Fatalf("Err = %v, want errors.Is %v", result.Err, startErr)
	}
	if !strings.Contains(result.Err.Error(), "mage ci start failed") {
		t.Fatalf("Err = %v, want substring %q", result.Err, "mage ci start failed")
	}
	if result.Output != "" {
		t.Fatalf("Output = %q, want empty on start failure", result.Output)
	}
}

// TestGateMageCIRejectsEmptyWorktree asserts the empty-worktree guard fires
// before the runner is invoked: result is Failed, Err names the empty field,
// and the fake runner records zero calls.
func TestGateMageCIRejectsEmptyWorktree(t *testing.T) {
	fake := &fakeCommandRunner{
		exitCode: 0, // would otherwise pass; guard must short-circuit
	}
	withFakeCommandRunner(t, fake)

	project := domain.Project{
		ID:                  "proj-empty-worktree",
		RepoPrimaryWorktree: "",
	}
	result := gateMageCI(context.Background(), gateMageCIFixtureItem(), project)

	if result.Status != GateStatusFailed {
		t.Fatalf("Status = %q, want %q", result.Status, GateStatusFailed)
	}
	if result.Err == nil {
		t.Fatal("Err = nil, want non-nil guard error")
	}
	if !strings.Contains(result.Err.Error(), "RepoPrimaryWorktree is empty") {
		t.Fatalf("Err = %v, want substring %q", result.Err, "RepoPrimaryWorktree is empty")
	}
	if fake.calls != 0 {
		t.Fatalf("runner.calls = %d, want 0 (guard must short-circuit)", fake.calls)
	}
	if result.GateName != templates.GateKindMageCI {
		t.Fatalf("GateName = %q, want %q", result.GateName, templates.GateKindMageCI)
	}
}

// TestGateMageCIRejectsWhitespaceWorktree asserts the empty-worktree guard
// trims whitespace before checking, mirroring dispatcher.go:392 which uses
// strings.TrimSpace on the same field. A worktree that is "   " or "\t\n"
// is functionally empty.
func TestGateMageCIRejectsWhitespaceWorktree(t *testing.T) {
	fake := &fakeCommandRunner{exitCode: 0}
	withFakeCommandRunner(t, fake)

	project := domain.Project{
		ID:                  "proj-ws-worktree",
		RepoPrimaryWorktree: "   \t\n  ",
	}
	result := gateMageCI(context.Background(), gateMageCIFixtureItem(), project)

	if result.Status != GateStatusFailed {
		t.Fatalf("Status = %q, want %q", result.Status, GateStatusFailed)
	}
	if !strings.Contains(result.Err.Error(), "RepoPrimaryWorktree is empty") {
		t.Fatalf("Err = %v, want substring %q", result.Err, "RepoPrimaryWorktree is empty")
	}
	if fake.calls != 0 {
		t.Fatalf("runner.calls = %d, want 0", fake.calls)
	}
}

// TestGateMageCIHonorsContextCancel asserts a pre-cancelled context produces
// GateStatusFailed with Err wrapping ctx.Err(), and the wrap message names
// the ctx-cancel path so dashboards can distinguish ctx-cancel from
// start-failure (both surface as Failed but with different Err prefixes).
//
// The fake runner returns ctx.Err() directly to simulate exec.CommandContext's
// behavior under cancellation. The gate must observe ctx.Err() != nil and
// route to the ctx branch BEFORE the start-failure branch — proven here by
// the "context cancelled" prefix vs the "start failed" prefix.
func TestGateMageCIHonorsContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fake := &fakeCommandRunner{
		err: ctx.Err(), // simulate exec.CommandContext returning ctx.Err()
	}
	withFakeCommandRunner(t, fake)

	result := gateMageCI(ctx, gateMageCIFixtureItem(), gateMageCIFixtureProject())

	if result.Status != GateStatusFailed {
		t.Fatalf("Status = %q, want %q", result.Status, GateStatusFailed)
	}
	if result.Err == nil {
		t.Fatal("Err = nil, want non-nil ctx-cancel error")
	}
	if !errors.Is(result.Err, context.Canceled) {
		t.Fatalf("Err = %v, want errors.Is context.Canceled", result.Err)
	}
	if !strings.Contains(result.Err.Error(), "context cancelled") {
		t.Fatalf("Err = %v, want substring %q (distinguishes ctx-cancel from start-fail)",
			result.Err, "context cancelled")
	}
	if strings.Contains(result.Err.Error(), "start failed") {
		t.Fatalf("Err = %v, must NOT name start-failure on ctx-cancel path", result.Err)
	}
	if result.Output != "" {
		t.Fatalf("Output = %q, want empty on ctx-cancel", result.Output)
	}
}

// TestGateMageCITailsLongOutput asserts the gate's Output on failure is
// bounded by tailOutput's MaxGateOutputLines (100) OR MaxGateOutputBytes
// (8KB), whichever is shorter. The fake emits 1000 lines on stdout —
// each line's content is short enough that line-bounding (last 100 lines)
// produces a shorter slice than byte-bounding (last 8KB), so the bound is
// the line cap.
func TestGateMageCITailsLongOutput(t *testing.T) {
	var stdout strings.Builder
	for i := 0; i < 1000; i++ {
		fmt.Fprintf(&stdout, "line %04d\n", i)
	}

	fake := &fakeCommandRunner{
		stdout:   []byte(stdout.String()),
		exitCode: 2,
	}
	withFakeCommandRunner(t, fake)

	result := gateMageCI(context.Background(), gateMageCIFixtureItem(), gateMageCIFixtureProject())

	if result.Status != GateStatusFailed {
		t.Fatalf("Status = %q, want %q", result.Status, GateStatusFailed)
	}

	// Line bound: last 100 lines = lines 900..999. The earliest retained
	// line MUST be line 900; line 899 MUST be dropped.
	if !strings.Contains(result.Output, "line 0900\n") {
		t.Fatal("Output missing earliest retained line (900); line tail wrong")
	}
	if !strings.Contains(result.Output, "line 0999\n") {
		t.Fatal("Output missing last line (999); failure tail truncated")
	}
	if strings.Contains(result.Output, "line 0899\n") {
		t.Fatal("Output retained line 899; line bound exceeded MaxGateOutputLines")
	}

	// Byte bound: regardless of line count, output must not exceed
	// MaxGateOutputBytes. Lines 900..999 with the format above total
	// ~1000 bytes — comfortably under 8KB — so the line bound dominates,
	// but the byte bound still applies as a safety net.
	if len(result.Output) > MaxGateOutputBytes {
		t.Fatalf("len(Output) = %d, want <= MaxGateOutputBytes (%d)",
			len(result.Output), MaxGateOutputBytes)
	}
	// Line bound: at most MaxGateOutputLines newlines.
	if strings.Count(result.Output, "\n") > MaxGateOutputLines {
		t.Fatalf("newlines in Output = %d, want <= MaxGateOutputLines (%d)",
			strings.Count(result.Output, "\n"), MaxGateOutputLines)
	}
}

// TestGateMageCICombinesStdoutAndStderrInOrder asserts the gate's Output on
// failure preserves stdout-then-stderr ordering — stderr trails stdout so the
// failure-summary tail (which mage emits on stderr) stays in the bounded
// window. The fake emits unique markers in each stream; the assertion is on
// their relative position in the output.
func TestGateMageCICombinesStdoutAndStderrInOrder(t *testing.T) {
	fake := &fakeCommandRunner{
		stdout:   []byte("STDOUT_MARKER\n"),
		stderr:   []byte("STDERR_MARKER\n"),
		exitCode: 1,
	}
	withFakeCommandRunner(t, fake)

	result := gateMageCI(context.Background(), gateMageCIFixtureItem(), gateMageCIFixtureProject())

	if result.Status != GateStatusFailed {
		t.Fatalf("Status = %q, want %q", result.Status, GateStatusFailed)
	}
	stdoutIdx := strings.Index(result.Output, "STDOUT_MARKER")
	stderrIdx := strings.Index(result.Output, "STDERR_MARKER")
	if stdoutIdx < 0 || stderrIdx < 0 {
		t.Fatalf("Output missing markers: stdoutIdx=%d stderrIdx=%d, Output=%q",
			stdoutIdx, stderrIdx, result.Output)
	}
	if stdoutIdx >= stderrIdx {
		t.Fatalf("stdout marker at %d, stderr marker at %d; stdout must precede stderr",
			stdoutIdx, stderrIdx)
	}
}
