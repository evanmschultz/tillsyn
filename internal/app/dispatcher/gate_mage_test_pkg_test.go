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

// scriptedCall is one entry in scriptedCommandRunner's per-package response
// table: the (stdout, stderr, exitCode, err) tuple returned for one Run
// invocation. The runner consumes entries in order — entry [0] is the
// response for the first call, entry [1] for the second, and so on.
//
// gate_mage_test_pkg's per-package iteration needs different responses across
// calls (e.g. pkg1 passes, pkg2 fails); fakeCommandRunner from gate_mage_ci_test.go
// returns the same tuple every call, which doesn't fit. scriptedCommandRunner
// is a sibling test double rather than a generalization so the 4b.3 gate's
// existing fake stays unchanged.
type scriptedCall struct {
	stdout   []byte
	stderr   []byte
	exitCode int
	err      error
}

// scriptedCommandRunner is the test double that returns scripted per-call
// responses and records the (dir, name, args) tuple of every invocation so
// tests can assert (a) the right packages were forwarded, (b) iteration
// halted on the right failure (no extra calls past the failing package), and
// (c) the gate did not fall through to the runner on guard-rejection paths
// (zero calls).
//
// The runner panics on overflow rather than wrapping around: a test that
// expects N calls but the gate makes N+1 gets a clear failure instead of
// silently re-using the last response.
type scriptedCommandRunner struct {
	script []scriptedCall
	calls  []scriptedInvocation
}

// scriptedInvocation captures one Run invocation's inputs for assertion.
type scriptedInvocation struct {
	dir  string
	name string
	args []string
}

// Run implements commandRunner. Returns the next scripted response and
// records the invocation. Panics on overflow (script exhausted).
func (s *scriptedCommandRunner) Run(_ context.Context, dir string, name string, args ...string) ([]byte, []byte, int, error) {
	idx := len(s.calls)
	// Copy args so a later mutation of the gate's args slice doesn't poison
	// the recorded snapshot.
	argsCopy := make([]string, len(args))
	copy(argsCopy, args)
	s.calls = append(s.calls, scriptedInvocation{
		dir:  dir,
		name: name,
		args: argsCopy,
	})
	if idx >= len(s.script) {
		panic(fmt.Sprintf(
			"scriptedCommandRunner: call %d exceeds scripted responses (%d); "+
				"test expected gate to halt earlier",
			idx+1, len(s.script),
		))
	}
	r := s.script[idx]
	return r.stdout, r.stderr, r.exitCode, r.err
}

// gateMageTestPkgFixtureProject returns a domain.Project with a populated
// worktree so the empty-worktree guard passes by default.
func gateMageTestPkgFixtureProject() domain.Project {
	return domain.Project{
		ID:                  "proj-mage-test-pkg-1",
		RepoPrimaryWorktree: "/tmp/proj-mage-test-pkg-1",
	}
}

// gateMageTestPkgFixtureItem returns a build-kind action item with the
// supplied packages — the gate reads only Kind (opaquely) and Packages from
// the action item, so the fixture stays minimal.
func gateMageTestPkgFixtureItem(pkgs []string) domain.ActionItem {
	return domain.ActionItem{
		ID:       "ai-mage-test-pkg-1",
		Kind:     domain.KindBuild,
		Packages: pkgs,
	}
}

// TestGateMageTestPkgPassesAllPackages asserts a clean run across multiple
// packages yields GateStatusPassed with empty Output and nil Err. Verifies
// the gate forwarded every package in declared order and made exactly N
// runner calls.
func TestGateMageTestPkgPassesAllPackages(t *testing.T) {
	pkgs := []string{
		"./internal/app/dispatcher",
		"./internal/domain",
		"./internal/templates",
	}
	runner := &scriptedCommandRunner{
		script: []scriptedCall{
			{stdout: []byte("pkg1 ok\n"), exitCode: 0},
			{stdout: []byte("pkg2 ok\n"), exitCode: 0},
			{stdout: []byte("pkg3 ok\n"), exitCode: 0},
		},
	}
	withFakeCommandRunner(t, runner)

	result := gateMageTestPkg(context.Background(), gateMageTestPkgFixtureItem(pkgs), gateMageTestPkgFixtureProject())

	if result.Status != GateStatusPassed {
		t.Fatalf("Status = %q, want %q", result.Status, GateStatusPassed)
	}
	if result.Err != nil {
		t.Fatalf("Err = %v, want nil", result.Err)
	}
	if result.Output != "" {
		t.Fatalf("Output = %q, want empty on pass", result.Output)
	}
	if result.GateName != templates.GateKindMageTestPkg {
		t.Fatalf("GateName = %q, want %q", result.GateName, templates.GateKindMageTestPkg)
	}
	if result.Duration <= 0 {
		t.Fatalf("Duration = %v, want > 0", result.Duration)
	}
	if len(runner.calls) != len(pkgs) {
		t.Fatalf("runner.calls = %d, want %d", len(runner.calls), len(pkgs))
	}
	for i, call := range runner.calls {
		if call.dir != "/tmp/proj-mage-test-pkg-1" {
			t.Fatalf("call[%d].dir = %q, want %q", i, call.dir, "/tmp/proj-mage-test-pkg-1")
		}
		if call.name != "mage" {
			t.Fatalf("call[%d].name = %q, want %q", i, call.name, "mage")
		}
		if len(call.args) != 2 {
			t.Fatalf("call[%d].args = %v, want 2 args", i, call.args)
		}
		if call.args[0] != "test-pkg" {
			t.Fatalf("call[%d].args[0] = %q, want %q", i, call.args[0], "test-pkg")
		}
		if call.args[1] != pkgs[i] {
			t.Fatalf("call[%d].args[1] = %q, want %q (declared order)", i, call.args[1], pkgs[i])
		}
	}
}

// TestGateMageTestPkgFailsOnFirstPackageNonZero asserts a non-zero exit on
// the first package halts iteration: result is Failed, Err names the failed
// package and exit code, and the runner records exactly one call (pkg2 and
// pkg3 NOT invoked).
func TestGateMageTestPkgFailsOnFirstPackageNonZero(t *testing.T) {
	pkgs := []string{"pkg1", "pkg2", "pkg3"}
	runner := &scriptedCommandRunner{
		script: []scriptedCall{
			{stdout: []byte("pkg1 fail\n"), stderr: []byte("FAIL pkg1\n"), exitCode: 1},
			// pkg2 / pkg3 entries deliberately absent — overflow panics if reached.
		},
	}
	withFakeCommandRunner(t, runner)

	result := gateMageTestPkg(context.Background(), gateMageTestPkgFixtureItem(pkgs), gateMageTestPkgFixtureProject())

	if result.Status != GateStatusFailed {
		t.Fatalf("Status = %q, want %q", result.Status, GateStatusFailed)
	}
	if result.Err == nil {
		t.Fatal("Err = nil, want non-nil naming failed package")
	}
	if !strings.Contains(result.Err.Error(), "mage test-pkg pkg1") {
		t.Fatalf("Err = %v, want substring %q", result.Err, "mage test-pkg pkg1")
	}
	if !strings.Contains(result.Err.Error(), "exit code 1") {
		t.Fatalf("Err = %v, want substring %q", result.Err, "exit code 1")
	}
	if len(runner.calls) != 1 {
		t.Fatalf("runner.calls = %d, want 1 (halt-on-first-failure violated)", len(runner.calls))
	}
	if result.Output == "" {
		t.Fatal("Output = empty, want failed-package's tail captured")
	}
	if !strings.Contains(result.Output, "FAIL pkg1") {
		t.Fatalf("Output = %q, want substring %q", result.Output, "FAIL pkg1")
	}
}

// TestGateMageTestPkgFailsOnSecondPackageNonZero asserts iteration halts on
// the FIRST failing package even when earlier packages passed: pkg1 exits 0,
// pkg2 exits 1, pkg3 is NOT invoked. Err names pkg2, runner.calls == 2.
func TestGateMageTestPkgFailsOnSecondPackageNonZero(t *testing.T) {
	pkgs := []string{"pkg1", "pkg2", "pkg3"}
	runner := &scriptedCommandRunner{
		script: []scriptedCall{
			{stdout: []byte("pkg1 ok\n"), exitCode: 0},
			{stdout: []byte("pkg2 fail\n"), stderr: []byte("FAIL pkg2\n"), exitCode: 1},
			// pkg3 entry absent — overflow panics if reached.
		},
	}
	withFakeCommandRunner(t, runner)

	result := gateMageTestPkg(context.Background(), gateMageTestPkgFixtureItem(pkgs), gateMageTestPkgFixtureProject())

	if result.Status != GateStatusFailed {
		t.Fatalf("Status = %q, want %q", result.Status, GateStatusFailed)
	}
	if !strings.Contains(result.Err.Error(), "mage test-pkg pkg2") {
		t.Fatalf("Err = %v, want substring %q", result.Err, "mage test-pkg pkg2")
	}
	if !strings.Contains(result.Err.Error(), "exit code 1") {
		t.Fatalf("Err = %v, want substring %q", result.Err, "exit code 1")
	}
	if len(runner.calls) != 2 {
		t.Fatalf("runner.calls = %d, want 2 (pkg3 should NOT be invoked)", len(runner.calls))
	}
	// Last call must be for pkg2, not pkg3.
	if runner.calls[1].args[1] != "pkg2" {
		t.Fatalf("runner.calls[1].args[1] = %q, want %q (pkg3 must not be invoked)",
			runner.calls[1].args[1], "pkg2")
	}
}

// TestGateMageTestPkgRejectsEmptyPackages asserts the empty-Packages guard
// fires before the runner is invoked, returning a fail-loud verdict (per
// WAVE_A_PLAN.md plan-revision WA-A5). The fake is configured to pass to
// prove the guard short-circuits — silent-pass would otherwise mask the
// QA gap the guard exists to surface.
func TestGateMageTestPkgRejectsEmptyPackages(t *testing.T) {
	runner := &scriptedCommandRunner{
		script: []scriptedCall{
			{exitCode: 0}, // would pass; guard must short-circuit
		},
	}
	withFakeCommandRunner(t, runner)

	result := gateMageTestPkg(context.Background(), gateMageTestPkgFixtureItem(nil), gateMageTestPkgFixtureProject())

	if result.Status != GateStatusFailed {
		t.Fatalf("Status = %q, want %q (fail-loud per WA-A5)", result.Status, GateStatusFailed)
	}
	if result.Err == nil {
		t.Fatal("Err = nil, want non-nil empty-packages error")
	}
	if !strings.Contains(result.Err.Error(), "no packages") {
		t.Fatalf("Err = %v, want substring %q", result.Err, "no packages")
	}
	if len(runner.calls) != 0 {
		t.Fatalf("runner.calls = %d, want 0 (guard must short-circuit)", len(runner.calls))
	}
	if result.GateName != templates.GateKindMageTestPkg {
		t.Fatalf("GateName = %q, want %q", result.GateName, templates.GateKindMageTestPkg)
	}
}

// TestGateMageTestPkgRejectsEmptyWorktree asserts the empty-worktree guard
// fires before the runner is invoked. Mirrors the gateMageCI counterpart.
func TestGateMageTestPkgRejectsEmptyWorktree(t *testing.T) {
	runner := &scriptedCommandRunner{
		script: []scriptedCall{
			{exitCode: 0},
		},
	}
	withFakeCommandRunner(t, runner)

	project := domain.Project{
		ID:                  "proj-empty-worktree",
		RepoPrimaryWorktree: "",
	}
	result := gateMageTestPkg(context.Background(), gateMageTestPkgFixtureItem([]string{"pkg1"}), project)

	if result.Status != GateStatusFailed {
		t.Fatalf("Status = %q, want %q", result.Status, GateStatusFailed)
	}
	if !strings.Contains(result.Err.Error(), "RepoPrimaryWorktree is empty") {
		t.Fatalf("Err = %v, want substring %q", result.Err, "RepoPrimaryWorktree is empty")
	}
	if len(runner.calls) != 0 {
		t.Fatalf("runner.calls = %d, want 0 (guard must short-circuit)", len(runner.calls))
	}
}

// TestGateMageTestPkgFailsOnStartError asserts a runner-reported start
// failure mid-iteration produces GateStatusFailed with Err wrapping the
// start error AND naming the package whose Run did not start. pkg1 starts
// fine and exits 0; pkg2's Run reports a start error; pkg3 is not invoked.
func TestGateMageTestPkgFailsOnStartError(t *testing.T) {
	pkgs := []string{"pkg1", "pkg2", "pkg3"}
	startErr := errors.New("exec: \"mage\": executable file not found in $PATH")
	runner := &scriptedCommandRunner{
		script: []scriptedCall{
			{stdout: []byte("pkg1 ok\n"), exitCode: 0},
			{err: startErr},
			// pkg3 absent — overflow panics if reached.
		},
	}
	withFakeCommandRunner(t, runner)

	result := gateMageTestPkg(context.Background(), gateMageTestPkgFixtureItem(pkgs), gateMageTestPkgFixtureProject())

	if result.Status != GateStatusFailed {
		t.Fatalf("Status = %q, want %q", result.Status, GateStatusFailed)
	}
	if !errors.Is(result.Err, startErr) {
		t.Fatalf("Err = %v, want errors.Is %v", result.Err, startErr)
	}
	if !strings.Contains(result.Err.Error(), "mage test-pkg pkg2") {
		t.Fatalf("Err = %v, want substring %q (failed package named)", result.Err, "mage test-pkg pkg2")
	}
	if !strings.Contains(result.Err.Error(), "start failed") {
		t.Fatalf("Err = %v, want substring %q", result.Err, "start failed")
	}
	if len(runner.calls) != 2 {
		t.Fatalf("runner.calls = %d, want 2 (pkg3 should NOT be invoked)", len(runner.calls))
	}
}

// TestGateMageTestPkgHonorsContextCancel asserts a pre-cancelled context
// produces GateStatusFailed with Err wrapping ctx.Err() and the wrap message
// names the ctx-cancel path so dashboards can distinguish ctx-cancel from
// start-failure.
//
// The scripted runner returns ctx.Err() directly to simulate
// exec.CommandContext's behavior under cancellation. The gate must observe
// ctx.Err() != nil and route to the ctx branch BEFORE the start-failure
// branch — proven here by the "context cancelled" prefix vs the "start
// failed" prefix.
func TestGateMageTestPkgHonorsContextCancel(t *testing.T) {
	pkgs := []string{"pkg1", "pkg2"}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	runner := &scriptedCommandRunner{
		script: []scriptedCall{
			{err: ctx.Err()}, // simulate exec.CommandContext returning ctx.Err()
		},
	}
	withFakeCommandRunner(t, runner)

	result := gateMageTestPkg(ctx, gateMageTestPkgFixtureItem(pkgs), gateMageTestPkgFixtureProject())

	if result.Status != GateStatusFailed {
		t.Fatalf("Status = %q, want %q", result.Status, GateStatusFailed)
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
	if !strings.Contains(result.Err.Error(), "pkg1") {
		t.Fatalf("Err = %v, want substring %q (failed package named)", result.Err, "pkg1")
	}
}

// TestGateMageTestPkgAggregatesOutputAcrossPackages asserts the gate's
// Output on failure preserves stdout from every package run so far in
// declared order — pkg1's lines precede pkg2's lines — and is bounded by
// tailOutput's MaxGateOutputLines / MaxGateOutputBytes rule. The fake emits
// 50 lines per package with unique markers; pkg2 fails. The combined
// 100-line output stays under MaxGateOutputLines so every line should be
// retained and pkg1's marker must precede pkg2's marker.
func TestGateMageTestPkgAggregatesOutputAcrossPackages(t *testing.T) {
	pkgs := []string{"pkg1", "pkg2"}
	var pkg1Out strings.Builder
	for i := 0; i < 50; i++ {
		fmt.Fprintf(&pkg1Out, "PKG1_LINE_%02d\n", i)
	}
	var pkg2Out strings.Builder
	for i := 0; i < 50; i++ {
		fmt.Fprintf(&pkg2Out, "PKG2_LINE_%02d\n", i)
	}
	runner := &scriptedCommandRunner{
		script: []scriptedCall{
			{stdout: []byte(pkg1Out.String()), exitCode: 0},
			{stdout: []byte(pkg2Out.String()), stderr: []byte("PKG2_STDERR_FAILURE\n"), exitCode: 1},
		},
	}
	withFakeCommandRunner(t, runner)

	result := gateMageTestPkg(context.Background(), gateMageTestPkgFixtureItem(pkgs), gateMageTestPkgFixtureProject())

	if result.Status != GateStatusFailed {
		t.Fatalf("Status = %q, want %q", result.Status, GateStatusFailed)
	}
	if result.Output == "" {
		t.Fatal("Output = empty, want aggregated tail across packages")
	}

	pkg1Idx := strings.Index(result.Output, "PKG1_LINE_")
	pkg2Idx := strings.Index(result.Output, "PKG2_LINE_")
	stderrIdx := strings.Index(result.Output, "PKG2_STDERR_FAILURE")
	if pkg1Idx < 0 {
		t.Fatalf("Output missing pkg1 marker; aggregation dropped pkg1 stdout")
	}
	if pkg2Idx < 0 {
		t.Fatalf("Output missing pkg2 marker; aggregation dropped pkg2 stdout")
	}
	if stderrIdx < 0 {
		t.Fatalf("Output missing pkg2 stderr marker; failure-stream tail truncated")
	}
	if pkg1Idx >= pkg2Idx {
		t.Fatalf("pkg1 idx %d >= pkg2 idx %d; declared-order aggregation violated", pkg1Idx, pkg2Idx)
	}
	if pkg2Idx >= stderrIdx {
		t.Fatalf("pkg2 stdout idx %d >= stderr idx %d; stdout-then-stderr ordering violated",
			pkg2Idx, stderrIdx)
	}

	// Bounded by tailOutput rule: pkg1 (50) + pkg2 (50) + stderr (1) = 101
	// lines; line bound (100) drops the earliest line — so PKG1_LINE_00
	// MUST be absent but PKG1_LINE_01 must be present. Line bound dominates
	// here because each line is short (~16 bytes); 101 lines is roughly
	// 1.6KB, well under the 8KB byte cap.
	if strings.Contains(result.Output, "PKG1_LINE_00\n") {
		t.Fatalf("Output retained PKG1_LINE_00; line bound exceeded MaxGateOutputLines (%d)",
			MaxGateOutputLines)
	}
	if !strings.Contains(result.Output, "PKG1_LINE_01\n") {
		t.Fatalf("Output missing PKG1_LINE_01; tail-bound trimmed too aggressively")
	}
	if !strings.Contains(result.Output, "PKG2_LINE_49\n") {
		t.Fatalf("Output missing PKG2_LINE_49; failure tail truncated")
	}
	if len(result.Output) > MaxGateOutputBytes {
		t.Fatalf("len(Output) = %d, want <= MaxGateOutputBytes (%d)",
			len(result.Output), MaxGateOutputBytes)
	}
	if strings.Count(result.Output, "\n") > MaxGateOutputLines {
		t.Fatalf("newlines in Output = %d, want <= MaxGateOutputLines (%d)",
			strings.Count(result.Output, "\n"), MaxGateOutputLines)
	}
}
