package app

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// gitStatusFixture builds a throwaway git repo at t.TempDir() with a tiny
// commit timeline driven by gitStatusFixture.commit. Mirrors the env-isolation
// pattern from internal/tui/gitdiff/exec_differ_test.go (round 3) — see that
// file's gitFixture doc-comment for the full motivation. Stripping every
// GIT_*= entry from os.Environ() before appending isolation overrides keeps
// the fixture safe under a `git push` pre-push hook environment that leaks
// GIT_DIR / GIT_INDEX_FILE / etc. pointing at the bare-root repo.
type gitStatusFixture struct {
	t    *testing.T
	root string
	home string
}

// newGitStatusFixture initializes a fresh git repository in a tempdir,
// configures a deterministic identity, and disables GPG signing so the
// fixture runs identically on dev machines with global signing policies.
func newGitStatusFixture(t *testing.T) *gitStatusFixture {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git not on PATH: %v", err)
	}
	root := t.TempDir()
	home := t.TempDir()
	fx := &gitStatusFixture{t: t, root: root, home: home}
	fx.git("init", "--initial-branch=main")
	fx.git("config", "user.email", "git-status-test@example.com")
	fx.git("config", "user.name", "git-status-test")
	fx.git("config", "commit.gpgsign", "false")
	fx.git("config", "tag.gpgsign", "false")
	return fx
}

// git runs a git subcommand inside the fixture, failing the test on any
// non-zero exit. Env handling mirrors filteredGitEnv (strip GIT_*) and adds
// isolation overrides so the per-test invocation never touches the dev's
// global config or an enclosing bare repo.
func (f *gitStatusFixture) git(args ...string) string {
	f.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = f.root
	cmd.Env = append(filteredGitEnv(),
		"GIT_AUTHOR_DATE=2026-01-01T00:00:00Z",
		"GIT_COMMITTER_DATE=2026-01-01T00:00:00Z",
		"GIT_PAGER=cat",
		"GIT_TERMINAL_PROMPT=0",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=/dev/null",
		"HOME="+f.home,
		"XDG_CONFIG_HOME="+f.home,
		"GIT_CEILING_DIRECTORIES="+f.root,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		f.t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, string(out))
	}
	return strings.TrimSpace(string(out))
}

// commit writes contents to name (creating parent dirs), stages the file,
// and commits with the supplied message. Returns nothing — callers don't
// need the SHA in the pre-check tests.
func (f *gitStatusFixture) commit(name, contents, message string) {
	f.t.Helper()
	full := filepath.Join(f.root, name)
	if dir := filepath.Dir(full); dir != f.root {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			f.t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(full, []byte(contents), 0o644); err != nil {
		f.t.Fatalf("write %s: %v", full, err)
	}
	f.git("add", name)
	f.git("commit", "-m", message)
}

// dirty rewrites name with new contents WITHOUT staging or committing —
// the file is now dirty in working-tree terms, which is what the pre-check
// detects via `git status --porcelain`.
func (f *gitStatusFixture) dirty(name, contents string) {
	f.t.Helper()
	full := filepath.Join(f.root, name)
	if err := os.WriteFile(full, []byte(contents), 0o644); err != nil {
		f.t.Fatalf("dirty write %s: %v", full, err)
	}
}

// TestDefaultGitStatusCheckerReturnsNilOnCleanPath asserts the happy path:
// every declared path is committed and unmodified, so the checker reports
// no dirty paths.
func TestDefaultGitStatusCheckerReturnsNilOnCleanPath(t *testing.T) {
	t.Parallel()
	fx := newGitStatusFixture(t)
	fx.commit("internal/foo/bar.go", "package foo\n", "add bar")

	dirty, err := defaultGitStatusChecker(context.Background(), fx.root, []string{"internal/foo/bar.go"})
	if err != nil {
		t.Fatalf("defaultGitStatusChecker err = %v, want nil", err)
	}
	if len(dirty) != 0 {
		t.Fatalf("dirty = %v, want empty", dirty)
	}
}

// TestDefaultGitStatusCheckerDetectsDirtyTrackedFile asserts that a tracked
// file with uncommitted modifications is flagged dirty.
func TestDefaultGitStatusCheckerDetectsDirtyTrackedFile(t *testing.T) {
	t.Parallel()
	fx := newGitStatusFixture(t)
	fx.commit("internal/foo/bar.go", "package foo\n", "add bar")
	fx.dirty("internal/foo/bar.go", "package foo\n\n// modified\n")

	dirty, err := defaultGitStatusChecker(context.Background(), fx.root, []string{"internal/foo/bar.go"})
	if err != nil {
		t.Fatalf("defaultGitStatusChecker err = %v, want nil", err)
	}
	if len(dirty) != 1 || dirty[0] != "internal/foo/bar.go" {
		t.Fatalf("dirty = %v, want [internal/foo/bar.go]", dirty)
	}
}

// TestDefaultGitStatusCheckerWalksMultiplePaths asserts that mixing one
// dirty + one clean path yields a dirty list naming ONLY the dirty entry.
func TestDefaultGitStatusCheckerWalksMultiplePaths(t *testing.T) {
	t.Parallel()
	fx := newGitStatusFixture(t)
	fx.commit("alpha.go", "package alpha\n", "add alpha")
	fx.commit("beta.go", "package beta\n", "add beta")
	fx.dirty("beta.go", "package beta\n\n// modified\n")

	dirty, err := defaultGitStatusChecker(context.Background(), fx.root, []string{"alpha.go", "beta.go"})
	if err != nil {
		t.Fatalf("defaultGitStatusChecker err = %v, want nil", err)
	}
	if len(dirty) != 1 || dirty[0] != "beta.go" {
		t.Fatalf("dirty = %v, want [beta.go]", dirty)
	}
}

// TestDefaultGitStatusCheckerEmptyPathsReturnsNil asserts the degenerate-
// input fast path: no paths means no work and definitionally no dirty.
func TestDefaultGitStatusCheckerEmptyPathsReturnsNil(t *testing.T) {
	t.Parallel()
	fx := newGitStatusFixture(t)
	fx.commit("alpha.go", "package alpha\n", "add alpha")

	dirty, err := defaultGitStatusChecker(context.Background(), fx.root, nil)
	if err != nil {
		t.Fatalf("defaultGitStatusChecker err = %v, want nil", err)
	}
	if len(dirty) != 0 {
		t.Fatalf("dirty = %v, want empty", dirty)
	}
}

// TestDefaultGitStatusCheckerIgnoresInheritedGitDir asserts the round-3
// env-isolation fix: setting GIT_DIR to a bogus path on the parent process
// must NOT redirect git's repository discovery — the checker must still
// operate against repoRoot. This is the load-bearing safety for running
// inside a `git push` pre-push hook (which leaks GIT_DIR pointing at the
// bare repo).
func TestDefaultGitStatusCheckerIgnoresInheritedGitDir(t *testing.T) {
	// Cannot t.Parallel() — this test mutates process env via t.Setenv.
	fx := newGitStatusFixture(t)
	fx.commit("internal/foo/bar.go", "package foo\n", "add bar")
	fx.dirty("internal/foo/bar.go", "package foo\n\n// modified\n")

	// Simulate hook context: GIT_DIR / GIT_INDEX_FILE leak from parent.
	t.Setenv("GIT_DIR", filepath.Join(t.TempDir(), "bogus.git"))
	t.Setenv("GIT_INDEX_FILE", filepath.Join(t.TempDir(), "bogus.index"))

	dirty, err := defaultGitStatusChecker(context.Background(), fx.root, []string{"internal/foo/bar.go"})
	if err != nil {
		t.Fatalf("defaultGitStatusChecker err = %v, want nil; pollution leaked through", err)
	}
	if len(dirty) != 1 || dirty[0] != "internal/foo/bar.go" {
		t.Fatalf("dirty = %v, want [internal/foo/bar.go]; env-isolation failed", dirty)
	}
}

// TestDefaultGitStatusCheckerEmptyRepoRootRejects asserts that an empty
// repoRoot is rejected with a descriptive error rather than silently
// accepted (which would mask a mis-configured project).
func TestDefaultGitStatusCheckerEmptyRepoRootRejects(t *testing.T) {
	t.Parallel()
	_, err := defaultGitStatusChecker(context.Background(), "   ", []string{"foo.go"})
	if err == nil {
		t.Fatal("defaultGitStatusChecker(emptyRoot) err = nil, want error")
	}
	if !strings.Contains(err.Error(), "empty repoRoot") {
		t.Fatalf("err = %v, want 'empty repoRoot' substring", err)
	}
}

// TestDefaultGitStatusCheckerWrapsPathspecErrors asserts that an out-of-
// worktree path produces a non-nil error (not an ErrPathsDirty) so callers
// can surface the misconfiguration verbatim.
func TestDefaultGitStatusCheckerWrapsPathspecErrors(t *testing.T) {
	t.Parallel()
	fx := newGitStatusFixture(t)
	fx.commit("alpha.go", "package alpha\n", "add alpha")

	// Path traversal escapes the worktree; git rejects with non-zero exit.
	_, err := defaultGitStatusChecker(context.Background(), fx.root, []string{"../../../etc/passwd"})
	if err == nil {
		t.Fatal("defaultGitStatusChecker(pathTraversal) err = nil, want error")
	}
	if errors.Is(err, ErrPathsDirty) {
		t.Fatalf("err = %v, must NOT be ErrPathsDirty (env error, not dirty error)", err)
	}
}

// TestDefaultGitStatusCheckerHandlesGitBinaryMissing asserts that the
// production checker returns a wrapped ErrGitNotFound (not a generic exec
// error) when `git` is absent from PATH. The PATH override points at a
// fresh empty tmpdir so exec.LookPath cannot resolve `git` at all,
// simulating a stripped CI image without git installed. Callers rely on
// errors.Is(err, ErrGitNotFound) to distinguish "fix your env" from "fix
// your tree".
//
// Sequential test (no t.Parallel) because t.Setenv requires it.
func TestOsGitStatusCheckerHandlesGitBinaryMissing(t *testing.T) {
	emptyDir := t.TempDir()
	t.Setenv("PATH", emptyDir)

	_, err := defaultGitStatusChecker(context.Background(), t.TempDir(), []string{"foo.go"})
	if err == nil {
		t.Fatal("defaultGitStatusChecker(no git on PATH) err = nil, want ErrGitNotFound")
	}
	if !errors.Is(err, ErrGitNotFound) {
		t.Fatalf("err = %v, want wrapped ErrGitNotFound", err)
	}
	if errors.Is(err, ErrPathsDirty) {
		t.Fatalf("err = %v, must NOT be ErrPathsDirty (env error, not dirty error)", err)
	}
}

// TestOsGitStatusCheckerHandlesNonexistentPath pins git's behavior when a
// declared path does not exist on disk inside the worktree. Empirically
// observed: `git status --porcelain -- <missing>` returns exit code 0
// with empty stdout when the path is unknown to git AND missing on disk
// (it's neither tracked nor untracked-with-content), so the checker
// reports the path as clean. This is acceptable "best-effort" semantics
// for the pre-check — the downstream domain layer already validates path
// existence at the ref-attachment / package-coverage layer; the pre-
// check's job is "don't start work on a dirty file," not "validate every
// path exists." If git's behavior changes in a future version (some
// versions warn about pathspec-not-matched), the test pins whichever
// behavior is observed and surfaces the change as a refinement.
func TestOsGitStatusCheckerHandlesNonexistentPath(t *testing.T) {
	t.Parallel()
	fx := newGitStatusFixture(t)
	fx.commit("alpha.go", "package alpha\n", "add alpha")

	dirty, err := defaultGitStatusChecker(context.Background(), fx.root, []string{"does/not/exist.go"})
	if err != nil {
		t.Fatalf("defaultGitStatusChecker(missing path) err = %v, want nil (non-existent path is not dirty)", err)
	}
	if len(dirty) != 0 {
		t.Fatalf("dirty = %v, want empty (non-existent path must not be flagged dirty)", dirty)
	}
}
