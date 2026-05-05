package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// GitStatusChecker probes whether each declared write-scope path in a project
// worktree is "clean" — i.e. has no uncommitted (working-tree or index)
// changes per `git status --porcelain <path>`. The pre-check on
// Service.CreateActionItem (droplet 4b.6) calls this seam before persisting
// any action item that declares non-empty Paths so a builder cannot start
// work on top of an already-dirty tree.
//
// Contract:
//   - repoRoot is the absolute filesystem path of the project's primary
//     worktree (Project.RepoPrimaryWorktree). The implementation runs
//     `git status` with cmd.Dir = repoRoot.
//   - paths is the worktree-relative path list declared by the caller.
//   - Return (nil, nil) when every path is clean.
//   - Return a non-empty `dirty []string` slice (in input order, deduped via
//     caller-side semantics) when one or more paths are dirty; err is nil in
//     this case so callers can format the rejection themselves.
//   - Return (_, err) for environmental failures: git missing on PATH,
//     non-zero git exit unrelated to dirty status, context cancellation.
//     Callers MUST treat any non-nil err as a hard failure.
//
// Per-path invocation, not batched: typical droplet path counts are <10 and
// the per-path cost is dominated by exec startup, not git work. Batched
// `git status --porcelain --pathspec-from-file -` is a future refinement
// when profiling shows it on a hot path.
//
// Always-on per droplet 4b.6 acceptance criterion 4 — there is no project
// metadata flag to bypass the check today. The post-MVP supersede CLI is
// the documented escape hatch.
type GitStatusChecker func(ctx context.Context, repoRoot string, paths []string) (dirty []string, err error)

// defaultGitStatusChecker is the production implementation of
// GitStatusChecker. It executes `git status --porcelain <path>` per path
// against repoRoot and considers a path dirty when porcelain stdout is
// non-empty (any of: staged, modified, untracked, renamed, deleted).
//
// Environment isolation mirrors internal/tui/gitdiff/exec_differ_test.go's
// gitFixture — see the doc-comment there for the full motivation. In
// summary:
//
//   - Filter os.Environ() to strip every GIT_*=... key. Without this filter
//     a parent process running inside a `git push` pre-push hook leaks
//     GIT_DIR / GIT_INDEX_FILE / GIT_WORK_TREE / GIT_PREFIX into our git
//     subprocess, where GIT_DIR overrides repository-discovery entirely
//     (GIT_CEILING_DIRECTORIES does NOT undo it). The pre-check then runs
//     against the env-pointed bare repo instead of repoRoot.
//   - Append GIT_CEILING_DIRECTORIES=<repoRoot> so git's discovery walk
//     halts at repoRoot and never escapes upward into a parent bare repo.
//   - Append GIT_CONFIG_NOSYSTEM=1, GIT_CONFIG_GLOBAL=/dev/null,
//     HOME=<repoRoot>, XDG_CONFIG_HOME=<repoRoot> so the per-test git
//     invocation never reads a user / global / system config that might be
//     locked by a concurrent push.
//
// HOME asymmetry vs gitdiff round-3 fixture: production points HOME at
// repoRoot itself rather than at a fresh tmpdir (the fixture's pattern).
// Functionally equivalent — GIT_CONFIG_GLOBAL=/dev/null pins the global
// config read to /dev/null regardless of where HOME points, so neither
// path can leak a real ~/.gitconfig. Reusing repoRoot avoids a per-call
// os.MkdirTemp + os.RemoveAll cycle on the production hot path. The
// fixture uses a separate tmpdir because it composes more isolation
// helpers (write-tracking, multi-commit timeline) that benefit from a
// single dedicated home directory; production has no such composition.
//
// Returns ErrGitNotFound (wrapping exec.ErrNotFound) when `git` is missing
// on PATH so callers can distinguish "fix your tree" from "fix your env".
func defaultGitStatusChecker(ctx context.Context, repoRoot string, paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	if strings.TrimSpace(repoRoot) == "" {
		return nil, fmt.Errorf("git status pre-check: empty repoRoot")
	}
	if _, err := exec.LookPath("git"); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGitNotFound, err)
	}

	dirty := make([]string, 0, len(paths))
	for _, path := range paths {
		out, err := runGitStatusPath(ctx, repoRoot, path)
		if err != nil {
			return nil, fmt.Errorf("git status pre-check %q: %w", path, err)
		}
		if len(strings.TrimSpace(out)) > 0 {
			dirty = append(dirty, path)
		}
	}
	if len(dirty) == 0 {
		return nil, nil
	}
	return dirty, nil
}

// runGitStatusPath executes one `git status --porcelain <path>` invocation
// and returns its stdout. Stderr is folded into the error message on
// non-zero exit so a pathspec-out-of-worktree rejection surfaces verbatim
// to the caller. Env isolation matches gitdiff.gitFixture (round-3 fix —
// strip ALL GIT_*= keys before appending isolation overrides).
func runGitStatusPath(ctx context.Context, repoRoot, path string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain", "--", path)
	cmd.Dir = repoRoot
	cmd.Env = append(filteredGitEnv(),
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=/dev/null",
		"HOME="+repoRoot,
		"XDG_CONFIG_HOME="+repoRoot,
		"GIT_CEILING_DIRECTORIES="+repoRoot,
		"GIT_TERMINAL_PROMPT=0",
		"GIT_PAGER=cat",
	)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// Treat ctx cancellation as the canonical sentinel rather than
		// the wrapped exec error so callers can errors.Is(ctx.Err()).
		// Checked first so a ctx-cancel that surfaces stderr ("signal:
		// killed", etc.) still routes to the cancellation path.
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		// Surface any pathspec / out-of-worktree / unborn-branch failure
		// with stderr included so the dev can act on the message.
		stderrText := strings.TrimSpace(stderr.String())
		if stderrText != "" {
			return "", fmt.Errorf("%w: %s", err, stderrText)
		}
		return "", err
	}
	return stdout.String(), nil
}

// filteredGitEnv returns os.Environ() with every GIT_*=... entry removed.
// See defaultGitStatusChecker's doc-comment for the motivation — the same
// fix as internal/tui/gitdiff/exec_differ_test.go round 3 (strip all GIT_*
// keys before appending explicit isolation values, otherwise an inherited
// GIT_DIR completely overrides GIT_CEILING_DIRECTORIES).
func filteredGitEnv() []string {
	src := os.Environ()
	out := make([]string, 0, len(src))
	for _, e := range src {
		if strings.HasPrefix(e, "GIT_") {
			continue
		}
		out = append(out, e)
	}
	return out
}
