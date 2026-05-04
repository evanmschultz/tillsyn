package gitdiff

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// gitFixture constructs a throwaway git repo under t.TempDir() with a tiny
// commit timeline driven by gitFixture.commit. The struct captures the repo
// root so tests can build an execDiffer scoped to it without mutating the
// process working directory.
//
// home is a separate per-fixture tempdir reused as HOME / XDG_CONFIG_HOME for
// every spawned git process. Together with GIT_CONFIG_NOSYSTEM=1,
// GIT_CONFIG_GLOBAL=/dev/null, and GIT_CEILING_DIRECTORIES=<root> this fully
// isolates the fixture from the developer's system / global / per-user git
// config — including any bare-root or parent-repo config file that a
// concurrent git operation might be holding a lock on. Without this
// isolation the tests are flaky under concurrent git activity (e.g. a
// `git push` invoking the pre-push `mage ci` hook while the bare-root config
// is locked) and fail with:
//
//	error: could not lock config file <bare-root>/config: File exists
//
// Round 1 (config isolation alone) wasn't enough on its own: even with the
// config search path pinned, `git init` still performs repository discovery
// that walks UP from cwd looking for an existing repo (`.git/` or a bare
// layout: HEAD + config + refs/). On dev machines where the test tempdir
// sits beneath a bare repo (here, the bare root one directory above main/),
// that walk finds the bare repo and tries to lock its config. Round 2 adds
// GIT_CEILING_DIRECTORIES=<root> so discovery halts at the fixture's own
// repo dir and never reaches the bare root.
//
// The fix lives on cmd.Env rather than t.Setenv because every test in this
// file uses t.Parallel(), and t.Setenv panics when called from a parallel
// test. Threading isolation through cmd.Env keeps the fixture safe under
// parallel execution.
type gitFixture struct {
	t    *testing.T
	root string
	home string
}

// newGitFixture initializes a fresh git repository in a tempdir, configures a
// deterministic author/committer identity, and disables GPG signing so the
// fixture runs identically on dev machines with global signing policies.
func newGitFixture(t *testing.T) *gitFixture {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git not on PATH: %v", err)
	}

	root := t.TempDir()
	home := t.TempDir()
	fx := &gitFixture{t: t, root: root, home: home}

	fx.git("init", "--initial-branch=main")
	fx.git("config", "user.email", "gitdiff-test@example.com")
	fx.git("config", "user.name", "gitdiff-test")
	fx.git("config", "commit.gpgsign", "false")
	fx.git("config", "tag.gpgsign", "false")

	return fx
}

// git runs a git subcommand inside the fixture, failing the test immediately
// on any non-zero exit so setup errors surface where they happen.
//
// The command's environment is built from filteredEnv() — os.Environ() with
// every GIT_*=... entry stripped — and then isolation overrides are appended.
// Filtering is critical: when this test binary runs inside a `git push`
// pre-push hook, git itself sets GIT_DIR / GIT_INDEX_FILE / GIT_WORK_TREE /
// GIT_PREFIX / GIT_REFLOG_ACTION / etc. on the hook's environment, all
// pointing at the bare-root repo that invoked the hook. If we let those leak
// in via os.Environ(), git honors GIT_DIR over any discovery-walk logic,
// completely bypassing GIT_CEILING_DIRECTORIES — and `git init` writes to
// the bare-root config (which is locked by the in-flight push) instead of
// the per-test tempdir. Round 1 (config isolation) and Round 2 (ceiling dir)
// both appended to os.Environ() and so failed under the hook context with:
//
//	error: could not lock config file <bare-root>/config: File exists
//	fatal: Unable to create '<bare-root>/worktrees/main/index.lock': File exists.
//
// Round 3 fixes that by filtering ALL GIT_* keys before appending isolation,
// so no inherited GIT_DIR/GIT_INDEX_FILE/etc. can reach the per-test git
// invocation. Later entries win in exec.Cmd's env handling, so the appended
// isolation values are authoritative.
func (f *gitFixture) git(args ...string) string {
	f.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = f.root
	cmd.Env = append(filteredEnv(),
		"GIT_AUTHOR_DATE=2026-01-01T00:00:00Z",
		"GIT_COMMITTER_DATE=2026-01-01T00:00:00Z",
		"GIT_PAGER=cat",
		"GIT_TERMINAL_PROMPT=0",
		// Isolation: skip system config (/etc/gitconfig) entirely.
		"GIT_CONFIG_NOSYSTEM=1",
		// Isolation: pin the global config to /dev/null so git never
		// reads or attempts to write the developer's ~/.gitconfig (or
		// any bare-root config a concurrent git op may have locked).
		"GIT_CONFIG_GLOBAL=/dev/null",
		// Isolation: redirect HOME to a per-fixture tempdir so any
		// HOME-derived path (credential helpers, ~/.config/git, etc.)
		// resolves under the fixture rather than the dev's real home.
		"HOME="+f.home,
		// Isolation: same idea for XDG — newer git versions consult
		// $XDG_CONFIG_HOME/git/config when GIT_CONFIG_GLOBAL isn't set,
		// and we want a consistent answer regardless of git version.
		"XDG_CONFIG_HOME="+f.home,
		// Isolation: stop git's repository-discovery walk at f.root so
		// it never finds an enclosing repo (e.g. the bare root that
		// contains main/). Without this, `git init` walks UP from cwd
		// looking for a `.git/` or a bare layout, finds the bare repo
		// at <bare-root>/, and tries to lock <bare-root>/config — which
		// collides with a concurrent `git push` holding that lock and
		// fails with:
		//   error: could not lock config file <bare-root>/config: File exists
		// GIT_CEILING_DIRECTORIES is a colon-separated list of dirs git's
		// discovery walk will not cross; pinning it to the fixture root
		// guarantees the walk never escapes the per-test tempdir.
		"GIT_CEILING_DIRECTORIES="+f.root,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		f.t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, string(out))
	}
	return strings.TrimSpace(string(out))
}

// filteredEnv returns os.Environ() with every GIT_*=... entry removed.
//
// Stripping all GIT_* keys is what makes the fixture safe under a `git push`
// pre-push hook: git sets GIT_DIR, GIT_INDEX_FILE, GIT_WORK_TREE, GIT_PREFIX,
// GIT_REFLOG_ACTION, etc. on the hook's environment, all pointing at the bare
// repo running the push. GIT_DIR in particular overrides repository discovery
// entirely — GIT_CEILING_DIRECTORIES does not undo it. Only by removing every
// inherited GIT_* key and re-adding ONLY the isolation values explicitly can
// we guarantee the per-test git invocation operates inside the fixture.
//
// Re-added GIT_* keys (GIT_AUTHOR_DATE, GIT_COMMITTER_DATE, GIT_PAGER,
// GIT_TERMINAL_PROMPT, GIT_CONFIG_NOSYSTEM, GIT_CONFIG_GLOBAL,
// GIT_CEILING_DIRECTORIES) are appended by the gitFixture.git env block and
// take effect because exec.Cmd resolves duplicates as last-wins.
func filteredEnv() []string {
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

// writeCommit writes contents to name (creating parent dirs), stages it, and
// commits with the provided message. It returns the resulting commit SHA so
// tests can drive the Differ with real revisions.
func (f *gitFixture) writeCommit(name, contents, message string) string {
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
	return f.git("rev-parse", "HEAD")
}

// TestExecDiffer_Ancestor asserts the happy path: linear history, start is an
// ancestor of HEAD, Diff returns a populated Patch and DivergenceAncestor.
func TestExecDiffer_Ancestor(t *testing.T) {
	t.Parallel()

	fx := newGitFixture(t)
	start := fx.writeCommit("hello.txt", "hello\n", "first")
	end := fx.writeCommit("hello.txt", "hello\nworld\n", "second")

	d := newExecDifferIn(fx.root)
	got, err := d.Diff(context.Background(), start, end, nil)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}

	if got.Divergence != DivergenceAncestor {
		t.Errorf("Divergence = %s, want ancestor", got.Divergence)
	}
	if !strings.Contains(got.Patch, "+world") {
		t.Errorf("Patch missing added line; got:\n%s", got.Patch)
	}
	if got.StartSHA != start {
		t.Errorf("StartSHA = %q, want %q", got.StartSHA, start)
	}
	if got.EndSHA != end {
		t.Errorf("EndSHA = %q, want %q", got.EndSHA, end)
	}
}

// TestExecDiffer_Diverged sets up a fork so start is NOT an ancestor of HEAD.
// The patch still computes — the status just flags the divergence.
func TestExecDiffer_Diverged(t *testing.T) {
	t.Parallel()

	fx := newGitFixture(t)
	base := fx.writeCommit("base.txt", "base\n", "base")

	// feature branch advances HEAD.
	fx.git("checkout", "-b", "feature")
	headFeature := fx.writeCommit("base.txt", "base\nfeature\n", "feature-1")

	// forked branch from base, never merged — this is our "start" commit.
	fx.git("checkout", base)
	fx.git("checkout", "-b", "forked")
	start := fx.writeCommit("base.txt", "base\nforked\n", "forked-1")

	// flip HEAD back to the feature branch so start is not an ancestor.
	fx.git("checkout", "feature")

	d := newExecDifferIn(fx.root)
	got, err := d.Diff(context.Background(), start, headFeature, nil)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}

	if got.Divergence != DivergenceDiverged {
		t.Errorf("Divergence = %s, want diverged", got.Divergence)
	}
	if got.Patch == "" {
		t.Error("expected non-empty Patch between forked and feature branches")
	}
}

// TestExecDiffer_UnknownCommit confirms ErrUnknownCommit is wrapped when a
// caller supplies a SHA that does not exist. The exec error remains reachable
// via errors.Unwrap for debug logging.
func TestExecDiffer_UnknownCommit(t *testing.T) {
	t.Parallel()

	fx := newGitFixture(t)
	fx.writeCommit("hello.txt", "hello\n", "first")

	d := newExecDifferIn(fx.root)
	_, err := d.Diff(context.Background(), "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef", "HEAD", nil)
	if !errors.Is(err, ErrUnknownCommit) {
		t.Fatalf("got err=%v, want ErrUnknownCommit wrap", err)
	}
}

// TestExecDiffer_PathsFilter asserts that the paths argument is passed
// through to `git diff -- <paths>` and narrows the output.
func TestExecDiffer_PathsFilter(t *testing.T) {
	t.Parallel()

	fx := newGitFixture(t)
	start := fx.writeCommit("alpha.txt", "alpha\n", "alpha-1")
	// second commit touches BOTH files so the default diff includes both.
	if err := os.WriteFile(filepath.Join(fx.root, "alpha.txt"), []byte("alpha\nalpha-2\n"), 0o644); err != nil {
		t.Fatalf("write alpha.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fx.root, "beta.txt"), []byte("beta\n"), 0o644); err != nil {
		t.Fatalf("write beta.txt: %v", err)
	}
	fx.git("add", "alpha.txt", "beta.txt")
	fx.git("commit", "-m", "both")
	end := fx.git("rev-parse", "HEAD")

	d := newExecDifferIn(fx.root)

	full, err := d.Diff(context.Background(), start, end, nil)
	if err != nil {
		t.Fatalf("unfiltered Diff: %v", err)
	}
	if !strings.Contains(full.Patch, "alpha.txt") || !strings.Contains(full.Patch, "beta.txt") {
		t.Fatalf("sanity check failed; expected both files in unfiltered diff:\n%s", full.Patch)
	}

	filtered, err := d.Diff(context.Background(), start, end, []string{"alpha.txt"})
	if err != nil {
		t.Fatalf("filtered Diff: %v", err)
	}
	if !strings.Contains(filtered.Patch, "alpha.txt") {
		t.Errorf("filtered patch missing alpha.txt:\n%s", filtered.Patch)
	}
	if strings.Contains(filtered.Patch, "beta.txt") {
		t.Errorf("filtered patch unexpectedly contains beta.txt:\n%s", filtered.Patch)
	}
}

// TestExecDiffer_ContextCancellation cancels the context before exec can
// complete and asserts the cancellation propagates as a wrapped error.
func TestExecDiffer_ContextCancellation(t *testing.T) {
	t.Parallel()

	fx := newGitFixture(t)
	start := fx.writeCommit("hello.txt", "hello\n", "first")
	end := fx.writeCommit("hello.txt", "hello\nworld\n", "second")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-canceled: the first exec has no chance to finish.

	d := newExecDifferIn(fx.root)
	_, err := d.Diff(ctx, start, end, nil)
	if err == nil {
		t.Fatal("expected cancellation error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got err=%v, want wrapped context.Canceled", err)
	}
}

// TestExecDiffer_EmptyDiff asserts that diffing a commit against itself
// yields an empty Patch with DivergenceAncestor (a commit is always its own
// ancestor per git semantics).
func TestExecDiffer_EmptyDiff(t *testing.T) {
	t.Parallel()

	fx := newGitFixture(t)
	sha := fx.writeCommit("hello.txt", "hello\n", "only")

	d := newExecDifferIn(fx.root)
	got, err := d.Diff(context.Background(), sha, sha, nil)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}

	if got.Patch != "" {
		t.Errorf("Patch = %q, want empty", got.Patch)
	}
	if got.Divergence != DivergenceAncestor {
		t.Errorf("Divergence = %s, want ancestor", got.Divergence)
	}
}

// TestExecDiffer_DeadlineExceeded pairs with the cancellation test by
// exercising the context.DeadlineExceeded branch explicitly — the branch
// sits on the same error path in Diff, but it is worth proving both exits
// so a future refactor cannot quietly swallow one.
func TestExecDiffer_DeadlineExceeded(t *testing.T) {
	t.Parallel()

	fx := newGitFixture(t)
	start := fx.writeCommit("hello.txt", "hello\n", "first")
	end := fx.writeCommit("hello.txt", "hello\nworld\n", "second")

	ctx, cancel := context.WithDeadline(context.Background(), time.Unix(0, 0))
	defer cancel()

	d := newExecDifferIn(fx.root)
	_, err := d.Diff(ctx, start, end, nil)
	if err == nil {
		t.Fatal("expected deadline error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got err=%v, want wrapped context.DeadlineExceeded", err)
	}
}
