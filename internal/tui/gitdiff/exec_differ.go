package gitdiff

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// execDiffer is the default Differ implementation. It shells out to the `git`
// binary available on PATH via os/exec, honoring the caller-supplied
// context.Context for cancellation.
//
// The type is unexported by design: consumers depend on the Differ interface,
// which keeps the diff pane decoupled from the exec mechanism. Tests in this
// package still drive the concrete type directly to assert exec-specific
// behavior (ancestor codes, stderr wrapping, paths filter).
type execDiffer struct {
	// gitBin names the git executable. Default "git" is resolved via PATH.
	gitBin string
	// workDir, when non-empty, is passed to exec.Cmd.Dir so tests (and any
	// future caller that needs to pin the repo) can run against a fixture
	// without chdir'ing the whole process.
	workDir string
}

// NewExecDiffer constructs the default Differ backed by `exec.CommandContext`.
//
// The returned value is an interface, never the concrete struct — callers
// must depend on Differ so the implementation can evolve (in-process go-git,
// cached differs, etc.) without a breaking change.
func NewExecDiffer() Differ {
	return &execDiffer{gitBin: "git"}
}

// newExecDifferIn constructs an execDiffer pinned to workDir. It is used only
// by tests in this package to target a tempdir fixture. The exported
// constructor intentionally omits the directory knob so production callers
// inherit the invoking process's working directory, matching `git`'s default.
func newExecDifferIn(workDir string) *execDiffer {
	return &execDiffer{gitBin: "git", workDir: workDir}
}

// Diff implements Differ. It runs `git diff <start>..<end>` with pager
// disabled and color forced off, streaming stdout into memory and capturing
// stderr for error messages. Ancestry between start and HEAD is computed via
// `git rev-parse --is-ancestor <start> HEAD`, and start/end are resolved to
// full SHAs so DiffResult carries stable identifiers.
func (e *execDiffer) Diff(ctx context.Context, start, end string, paths []string) (DiffResult, error) {
	if strings.TrimSpace(start) == "" || strings.TrimSpace(end) == "" {
		return DiffResult{}, ErrEmptyRevision
	}

	startSHA, err := e.resolve(ctx, start)
	if err != nil {
		return DiffResult{}, err
	}
	endSHA, err := e.resolve(ctx, end)
	if err != nil {
		return DiffResult{}, err
	}

	patch, err := e.diff(ctx, startSHA, endSHA, paths)
	if err != nil {
		return DiffResult{}, err
	}

	divergence := e.ancestry(ctx, startSHA)

	return DiffResult{
		Patch:      patch,
		Divergence: divergence,
		StartSHA:   startSHA,
		EndSHA:     endSHA,
	}, nil
}

// resolve expands a revision string into a full commit SHA via
// `git rev-parse --verify <rev>^{commit}`. Unknown revisions are reported via
// ErrUnknownCommit wrapped around the underlying exec error so callers can
// match with errors.Is.
func (e *execDiffer) resolve(ctx context.Context, rev string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := e.command(ctx, "rev-parse", "--verify", rev+"^{commit}")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if runErr := cmd.Run(); runErr != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", fmt.Errorf("gitdiff: resolve %q canceled: %w", rev, ctxErr)
		}
		return "", fmt.Errorf("%w: resolve %q: %s: %w", ErrUnknownCommit, rev, strings.TrimSpace(stderr.String()), runErr)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// diff runs `git diff <start> <end> [-- paths...]` with machine-friendly
// settings. Non-zero exit is treated as an error; empty stdout is a legitimate
// outcome when the trees match.
func (e *execDiffer) diff(ctx context.Context, start, end string, paths []string) (string, error) {
	args := []string{"diff", "--no-color", start, end}
	if len(paths) > 0 {
		args = append(args, "--")
		args = append(args, paths...)
	}

	var stdout, stderr bytes.Buffer
	cmd := e.command(ctx, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if runErr := cmd.Run(); runErr != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", fmt.Errorf("gitdiff: diff canceled: %w", ctxErr)
		}
		return "", fmt.Errorf("gitdiff: diff %s..%s failed: %s: %w", start, end, strings.TrimSpace(stderr.String()), runErr)
	}
	return stdout.String(), nil
}

// ancestry returns DivergenceAncestor when start is an ancestor of HEAD,
// DivergenceDiverged when it is not, and DivergenceUnknown when the check
// cannot produce a definitive answer (for example, HEAD is unreadable).
//
// Per git-merge-base(1), `--is-ancestor` exits 0 for ancestor, 1 for
// non-ancestor, and any other code for a usage or internal error. Context
// cancellation collapses into DivergenceUnknown because the caller already
// knows via ctx.Err() and the diff output it drives is a no-op in that case.
func (e *execDiffer) ancestry(ctx context.Context, start string) DivergenceStatus {
	cmd := e.command(ctx, "merge-base", "--is-ancestor", start, "HEAD")

	runErr := cmd.Run()
	if runErr == nil {
		return DivergenceAncestor
	}

	var exitErr *exec.ExitError
	if errors.As(runErr, &exitErr) {
		if exitErr.ExitCode() == 1 {
			return DivergenceDiverged
		}
	}
	return DivergenceUnknown
}

// command builds an exec.Cmd with machine-consumption environment settings:
// pager disabled (GIT_PAGER=cat) and terminal prompts off so a misconfigured
// credential helper can never hang the TUI.
func (e *execDiffer) command(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, e.gitBin, args...)
	if e.workDir != "" {
		cmd.Dir = e.workDir
	}
	cmd.Env = append(cmd.Environ(),
		"GIT_PAGER=cat",
		"GIT_TERMINAL_PROMPT=0",
	)
	return cmd
}

// compile-time assertion that execDiffer satisfies Differ.
var _ Differ = (*execDiffer)(nil)
