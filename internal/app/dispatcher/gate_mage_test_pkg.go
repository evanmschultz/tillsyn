package dispatcher

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/evanmschultz/tillsyn/internal/templates"
)

// gateMageTestPkg executes `mage test-pkg <pkg>` once per entry in
// item.Packages against project.RepoPrimaryWorktree and returns one aggregated
// gate verdict. Matches the gateFunc signature registered against
// templates.GateKindMageTestPkg; production wiring lives in 4b.7's subscriber
// (gateRunner.Register call), NOT in this file.
//
// Behavior summary:
//
//   - Empty item.Packages → GateStatusFailed with an Err naming "no packages
//     declared". Per WAVE_A_PLAN.md plan-revision WA-A5 this is fail-loud,
//     NOT silent-pass: a planner that bound mage_test_pkg to a kind without
//     populating Packages on the action item has produced a QA gap, and
//     silently passing the gate would let unverified packages slip through.
//     Surfacing the misconfiguration as a gate failure forces the planner /
//     template author to either populate item.Packages or remove the gate
//     binding for that kind.
//   - Empty project.RepoPrimaryWorktree → GateStatusFailed naming the empty
//     field. Mirrors gateMageCI's guard (gate_mage_ci.go) so the failure mode
//     is consistent across the dispatcher's worktree-rooted code paths.
//   - Halt-on-first-failure across packages: packages are iterated in
//     item.Packages's declared order. The first non-zero exit, start-error,
//     or ctx-cancel halts iteration and remaining packages are NOT invoked.
//     This mirrors the gateRunner's halt-on-first-failure semantic at the
//     gate level: a single failed package is enough to fail the gate; running
//     the rest would slow the failure-loop without adding signal.
//   - mage exits 0 for every package → GateStatusPassed. Output is empty;
//     the gate does not retain stdout for fully-passing runs.
//   - mage exits non-zero for some package → GateStatusFailed. Err names the
//     failed package and exit code (e.g. "mage test-pkg ./internal/foo
//     failed: exit code 1"). Output carries the bounded tail of combined
//     stdout+stderr from EVERY package run so far (including the failed one),
//     ordered pkg1 stdout+stderr, pkg2 stdout+stderr, …. The aggregation
//     matters when the failed package's output is short — surrounding
//     successful-package output (e.g. compilation summaries) often carries
//     the context the dev needs for the failure-loop. Bounded by tailOutput's
//     MaxGateOutputLines / MaxGateOutputBytes (last 100 lines OR last 8KB,
//     whichever is shorter, UTF-8 sanitized).
//   - Process-start failure mid-iteration (mage not on PATH, worktree dir
//     missing, fork errno) → GateStatusFailed. Err wraps the underlying
//     os/exec error and names the package whose Run did not start.
//   - ctx cancellation → GateStatusFailed. Err wraps ctx.Err(); the gate
//     distinguishes this from start-failure by checking ctx.Err() before
//     reading the runner's err. The child process is killed by
//     exec.CommandContext when ctx fires.
//
// Pre-conditions enforced by the caller (gateRunner / 4b.7 subscriber):
//
//   - `mage` must be on PATH. Same as gateMageCI; missing-binary surfaces
//     as a start-failure verdict so the dev's failure-loop instinct ("did
//     mage even run?") gets a distinct error string.
//   - The action item must declare item.Packages with at least one entry.
//     Empty Packages is fail-loud per the doc above.
//
// Output capture is best-effort: binary or non-UTF-8 bytes from any
// package's mage stdout/stderr are sanitized via tailOutput's
// strings.ToValidUTF8 path before populating GateResult.Output. Callers do
// not need additional sanitization downstream.
//
// Duration is wall-clock from gateMageTestPkg entry to return — includes
// every per-package runner invocation. Populated on every result regardless
// of status so dashboards can distinguish a fast guard reject from a slow
// per-package run.
//
// commandRunner reuse: this gate calls defaultCommandRunner.Run (defined in
// gate_mage_ci.go) directly. No new commandRunner type is introduced; the
// 4b.3 seam is the single extension point for mage-shelling gates.
func gateMageTestPkg(ctx context.Context, item domain.ActionItem, project domain.Project) GateResult {
	start := time.Now()

	if len(item.Packages) == 0 {
		return GateResult{
			GateName: templates.GateKindMageTestPkg,
			Status:   GateStatusFailed,
			Err: errors.New(
				"mage_test_pkg: action item declares no packages — " +
					"planner must populate item.Packages or remove this gate from kind",
			),
			Duration: time.Since(start),
		}
	}

	if strings.TrimSpace(project.RepoPrimaryWorktree) == "" {
		return GateResult{
			GateName: templates.GateKindMageTestPkg,
			Status:   GateStatusFailed,
			Err:      errors.New("project: RepoPrimaryWorktree is empty"),
			Duration: time.Since(start),
		}
	}

	// aggregated accumulates per-package stdout+stderr in declared order so
	// a failure tail covers the run-so-far, not just the failed package.
	var aggregated []byte

	for _, pkg := range item.Packages {
		stdout, stderr, exitCode, runErr := defaultCommandRunner.Run(
			ctx,
			project.RepoPrimaryWorktree,
			"mage",
			"test-pkg",
			pkg,
		)

		// Always fold this package's output into the running aggregate before
		// inspecting status — even on start-error or ctx-cancel the streams
		// may carry partial bytes worth surfacing in the failure tail.
		aggregated = combineGateOutput(aggregated, combineGateOutput(stdout, stderr))

		// ctx-cancel takes precedence over runErr — exec.CommandContext kills
		// the child when ctx fires and the resulting Wait error is opaque
		// about the cause. ctx.Err() is the authoritative signal. Mirrors
		// gateMageCI's ordering.
		if ctxErr := ctx.Err(); ctxErr != nil {
			return GateResult{
				GateName: templates.GateKindMageTestPkg,
				Status:   GateStatusFailed,
				Err: fmt.Errorf(
					"mage test-pkg %s context cancelled: %w", pkg, ctxErr,
				),
				Output:   tailOutput(aggregated, MaxGateOutputLines, MaxGateOutputBytes),
				Duration: time.Since(start),
			}
		}

		if runErr != nil {
			return GateResult{
				GateName: templates.GateKindMageTestPkg,
				Status:   GateStatusFailed,
				Err: fmt.Errorf(
					"mage test-pkg %s start failed: %w", pkg, runErr,
				),
				Output:   tailOutput(aggregated, MaxGateOutputLines, MaxGateOutputBytes),
				Duration: time.Since(start),
			}
		}

		if exitCode != 0 {
			return GateResult{
				GateName: templates.GateKindMageTestPkg,
				Status:   GateStatusFailed,
				Err: fmt.Errorf(
					"mage test-pkg %s failed: exit code %d", pkg, exitCode,
				),
				Output:   tailOutput(aggregated, MaxGateOutputLines, MaxGateOutputBytes),
				Duration: time.Since(start),
			}
		}
		// exitCode == 0: continue to next package. Successful-package output
		// is retained in `aggregated` so a downstream failure's tail surfaces
		// the surrounding context (compilation summaries, prior package's
		// last-line markers).
	}

	// Every package passed.
	return GateResult{
		GateName: templates.GateKindMageTestPkg,
		Status:   GateStatusPassed,
		Duration: time.Since(start),
	}
}
