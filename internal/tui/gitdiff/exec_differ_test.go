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
// every spawned git process. Together with GIT_CONFIG_NOSYSTEM=1 and
// GIT_CONFIG_GLOBAL=/dev/null this fully isolates the fixture from the
// developer's system / global / per-user git config — including any bare-root
// or parent-repo config file that a concurrent git operation might be holding
// a lock on. Without this isolation the tests are flaky under concurrent git
// activity (e.g. a `git push` invoking the pre-push `mage ci` hook while the
// bare-root config is locked) and fail with:
//
//	error: could not lock config file <bare-root>/config: File exists
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
// The command's environment is built fresh from os.Environ() with isolation
// overrides appended. Later entries win in exec.Cmd's env handling, so the
// HOME / XDG_CONFIG_HOME / GIT_CONFIG_NOSYSTEM / GIT_CONFIG_GLOBAL values
// here override whatever the parent process exported. See the gitFixture doc
// comment above for the full rationale.
func (f *gitFixture) git(args ...string) string {
	f.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = f.root
	cmd.Env = append(os.Environ(),
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
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		f.t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, string(out))
	}
	return strings.TrimSpace(string(out))
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
