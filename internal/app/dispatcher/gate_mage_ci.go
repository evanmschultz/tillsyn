package dispatcher

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/evanmschultz/tillsyn/internal/templates"
)

// commandRunner is the package-private test seam between the gate
// implementations (4b.3 mage_ci, 4b.4 mage_test_pkg) and the underlying
// os/exec invocation. Production code wires the defaultCommandRunner which
// shells out via *exec.Cmd; tests inject a fake that returns canned
// stdout/stderr/exitCode/err tuples.
//
// The four return values map to the four observable outcomes a gate
// implementation must distinguish:
//
//   - stdout / stderr: captured byte streams; the gate concatenates them
//     (stderr after stdout) and feeds the joined buffer through tailOutput.
//   - exitCode: the child process exit code. Zero = pass; non-zero = fail.
//     Meaningful only when err is nil OR err is a process-exit error
//     (commandRunner implementations normalize this).
//   - err: non-nil signals that the runner could NOT report a clean
//     exit. Three distinguishable causes:
//     1. Process-start failure (binary not on PATH, dir missing, fork-EXEC
//     errno) — err is the raw os/exec error, exitCode == 0, streams empty.
//     2. Context cancellation — err wraps ctx.Err() (context.Canceled or
//     context.DeadlineExceeded). Caller distinguishes via ctx.Err() check
//     before reading err.
//     3. Other Wait() error not exposing an exit code (signal, OS error) —
//     err is the raw os/exec error, exitCode == 0.
//
// Gate implementations check ctx.Err() first to disambiguate (2) from (1)
// and (3), then route start-vs-exit on err being nil.
//
// The seam is the single extension point 4b.4's mage_test_pkg gate reuses;
// adding new mage gates does not require a new seam.
type commandRunner interface {
	Run(ctx context.Context, dir string, name string, args ...string) (stdout []byte, stderr []byte, exitCode int, err error)
}

// execCommandRunner is the production commandRunner that wraps
// exec.CommandContext. cmd.Dir is set to the supplied dir; stdout and stderr
// are captured into separate bytes.Buffer values so callers can render them
// in stdout-then-stderr order (the convention mage's structured logger emits
// failures on stderr, so trailing-stderr in the joined output keeps the
// failure tail bytes-bounded under the 8KB cap).
//
// The runner distinguishes start-failure from exit-failure by splitting
// cmd.Start() and cmd.Wait():
//
//   - cmd.Start() error → returned as raw err with exitCode == 0.
//   - cmd.Wait() *exec.ExitError → exitCode populated, err == nil (the
//     non-zero exit IS the failure signal; the ExitError adds no information
//     the gate needs).
//   - cmd.Wait() other error → err returned raw with exitCode == 0
//     (ctx-cancel falls into this branch because exec.CommandContext kills
//     the child when ctx fires; gates check ctx.Err() to disambiguate).
type execCommandRunner struct{}

// Run is execCommandRunner's commandRunner implementation. See the type-level
// doc-comment for start-vs-exit error distinction semantics.
func (execCommandRunner) Run(ctx context.Context, dir string, name string, args ...string) ([]byte, []byte, int, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		return nil, nil, 0, err
	}

	if err := cmd.Wait(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return stdoutBuf.Bytes(), stderrBuf.Bytes(), exitErr.ExitCode(), nil
		}
		return stdoutBuf.Bytes(), stderrBuf.Bytes(), 0, err
	}

	return stdoutBuf.Bytes(), stderrBuf.Bytes(), 0, nil
}

// defaultCommandRunner is the production runner instance gateMageCI invokes.
// Tests swap this var via t.Cleanup-restored assignment to inject a fake
// commandRunner; production code never reassigns it.
var defaultCommandRunner commandRunner = execCommandRunner{}

// gateMageCI executes `mage ci` in project.RepoPrimaryWorktree and returns
// the gate verdict. Matches the gateFunc signature registered against
// templates.GateKindMageCI; production wiring lives in 4b.7's subscriber
// (gateRunner.Register call), NOT in this file.
//
// Behavior summary:
//
//   - Empty project.RepoPrimaryWorktree → GateStatusFailed naming the empty
//     field. Mirrors dispatcher.go:392's existing guard so the failure mode
//     is consistent across the dispatcher's worktree-rooted code paths.
//   - mage exits with code 0 → GateStatusPassed. Output is empty; the gate
//     does not retain stdout for passing runs.
//   - mage exits with non-zero code → GateStatusFailed. Err names the exit
//     code; Output carries the bounded tail of combined stdout+stderr per
//     tailOutput's MaxGateOutputLines / MaxGateOutputBytes rule (last 100
//     lines OR last 8KB, whichever is shorter, UTF-8 sanitized).
//   - Process-start failure (mage not on PATH, worktree dir missing, fork
//     errno) → GateStatusFailed. Err wraps the underlying os/exec error
//     with the "mage ci start failed:" prefix so callers can grep the
//     failure mode. Output is empty (no streams to capture).
//   - ctx cancellation (caller timeout, parent ctx teardown) → GateStatusFailed.
//     Err wraps ctx.Err(); the gate distinguishes this from start-failure
//     by checking ctx.Err() before reading the runner's err. The child
//     process is killed by exec.CommandContext when ctx fires.
//
// Pre-conditions enforced by the caller (gateRunner / 4b.7 subscriber):
//
//   - `mage` must be on PATH. The gate does not auto-install or fall back
//     to `go run github.com/magefile/mage`. Missing-binary surfaces as a
//     start-failure verdict so the dev's failure-loop instinct ("did mage
//     even run?") gets a distinct error string.
//   - The action item must be in_progress (the runner only fires gates on
//     post-build verification; this gate trusts the caller).
//
// Output capture is best-effort: binary or non-UTF-8 bytes in mage's
// stdout/stderr are sanitized via tailOutput's strings.ToValidUTF8 path
// before populating GateResult.Output. Callers do not need additional
// sanitization downstream.
//
// Duration is wall-clock from gateMageCI entry to return — includes the
// empty-worktree guard branch as well as the runner invocation. Populated
// on every result regardless of status so dashboards can distinguish a
// fast empty-worktree reject from a slow CI run.
func gateMageCI(ctx context.Context, _ domain.ActionItem, project domain.Project) GateResult {
	start := time.Now()

	if strings.TrimSpace(project.RepoPrimaryWorktree) == "" {
		return GateResult{
			GateName: templates.GateKindMageCI,
			Status:   GateStatusFailed,
			Err:      errors.New("project: RepoPrimaryWorktree is empty"),
			Duration: time.Since(start),
		}
	}

	stdout, stderr, exitCode, runErr := defaultCommandRunner.Run(
		ctx,
		project.RepoPrimaryWorktree,
		"mage",
		"ci",
	)

	// ctx-cancel takes precedence over runErr — exec.CommandContext kills
	// the child when ctx fires and the resulting Wait error is opaque about
	// the cause. ctx.Err() is the authoritative signal.
	if ctxErr := ctx.Err(); ctxErr != nil {
		return GateResult{
			GateName: templates.GateKindMageCI,
			Status:   GateStatusFailed,
			Err:      fmt.Errorf("mage ci context cancelled: %w", ctxErr),
			Duration: time.Since(start),
		}
	}

	if runErr != nil {
		return GateResult{
			GateName: templates.GateKindMageCI,
			Status:   GateStatusFailed,
			Err:      fmt.Errorf("mage ci start failed: %w", runErr),
			Duration: time.Since(start),
		}
	}

	if exitCode == 0 {
		return GateResult{
			GateName: templates.GateKindMageCI,
			Status:   GateStatusPassed,
			Duration: time.Since(start),
		}
	}

	// Non-zero exit: combine stdout+stderr (stderr last so the failure tail
	// stays in the bounded window) and route through tailOutput.
	combined := combineGateOutput(stdout, stderr)
	return GateResult{
		GateName: templates.GateKindMageCI,
		Status:   GateStatusFailed,
		Err:      fmt.Errorf("mage ci failed: exit code %d", exitCode),
		Output:   tailOutput(combined, MaxGateOutputLines, MaxGateOutputBytes),
		Duration: time.Since(start),
	}
}

// combineGateOutput joins stdout and stderr into a single byte slice with
// stderr appended after stdout. A trailing newline is inserted between the
// two streams when stdout does not already end with one so the boundary
// stays line-aligned for the line-tail bounding in tailOutput.
//
// The stderr-after-stdout convention matches mage's structured logger which
// emits failure summaries on stderr; trailing-stderr keeps the failure tail
// bytes-bounded under the 8KB cap when the gate fails. Shared with 4b.4's
// gate_mage_test_pkg.go.
func combineGateOutput(stdout, stderr []byte) []byte {
	if len(stdout) == 0 {
		return stderr
	}
	if len(stderr) == 0 {
		return stdout
	}
	combined := make([]byte, 0, len(stdout)+1+len(stderr))
	combined = append(combined, stdout...)
	if stdout[len(stdout)-1] != '\n' {
		combined = append(combined, '\n')
	}
	combined = append(combined, stderr...)
	return combined
}
