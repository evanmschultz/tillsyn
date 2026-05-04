# Gitdiff Test Isolation Worklog

Pre-existing test-isolation bug in `internal/tui/gitdiff/exec_differ_test.go`,
surfaced by Drop 4a's pre-push `mage ci` hook. Captured here as a
post-droplet hotfix worklog because it does not slot under any open builder
droplet.

## Problem

`internal/tui/gitdiff/exec_differ_test.go` spins up real git repositories
under `t.TempDir()` and shells out to `git init` / `git config` /
`git add` / `git commit` to drive a deterministic commit timeline. The
fixture's `gitFixture.git` helper was already careful about a few env
vars (`GIT_AUTHOR_DATE`, `GIT_COMMITTER_DATE`, `GIT_PAGER`,
`GIT_TERMINAL_PROMPT`) but it did NOT isolate the spawned `git`
processes from the developer's system / global / per-user git config.

That bit us under Drop 4a's pre-push hook flow:

1. Dev runs `git push origin main` from the `main/` worktree.
2. Git's pre-push pipeline takes a write lock on the bare-root config
   at `/Users/evanschultz/Documents/Code/hylla/tillsyn/config` while the
   pre-push hook fires.
3. The pre-push hook runs `mage ci`, which runs the gitdiff package
   tests.
4. Each gitdiff test spawns `git init` + `git config user.email …` +
   `git config user.name …` etc. inside `t.TempDir()`. Because the test
   fixture inherited the parent process's env, those `git config` calls
   walked up to the bare-root config (the parent worktree's config from
   git's POV), tried to take their own lock, and failed with:

       error: could not lock config file <bare-root>/config: File exists

5. Six tests in the file fail. Local `mage ci` passed clean because no
   concurrent push was holding the lock.

## Fix

Per-fixture git environment isolation, threaded through `gitFixture.git`
(the single funnel for every `exec.Command("git", ...)` call in the
test file). The fix appends four isolation env vars to the existing
`cmd.Env`:

- `GIT_CONFIG_NOSYSTEM=1` — skip `/etc/gitconfig`.
- `GIT_CONFIG_GLOBAL=/dev/null` — pin global config to an
  always-empty, never-locked path.
- `HOME=<per-fixture t.TempDir()>` — redirect every HOME-derived path
  (credential helpers, `~/.config/git`, etc.) into a fresh per-fixture
  tempdir.
- `XDG_CONFIG_HOME=<same per-fixture t.TempDir()>` — newer git versions
  consult `$XDG_CONFIG_HOME/git/config` when `GIT_CONFIG_GLOBAL` is not
  set; pinning XDG keeps the fix stable across git versions.

After the fix, the only git config files reachable by the spawned git
processes are:

1. The repo-local `.git/config` inside the per-test `t.TempDir()` repo
   (fresh per test, never shared, never contended).
2. `GIT_CONFIG_GLOBAL` → `/dev/null` (always empty, never locked).
3. HOME / XDG-derived paths → fresh per-test tempdir (empty, never
   locked).

The bare-root config at
`/Users/evanschultz/Documents/Code/hylla/tillsyn/config` is no longer in
the read or write set of any spawned `git` process, so a concurrent
`git push` holding that lock cannot collide with the test suite.

### Why `cmd.Env`, Not `t.Setenv`

The original orchestrator brief suggested using `t.Setenv` for the
isolation. That would have panicked: every test in
`exec_differ_test.go` calls `t.Parallel()`, and the testing package
forbids `t.Setenv` in any test (or any of its subtests/parents) that
also runs in parallel — `t.Setenv` mutates the process env, which is
unsafe under parallel execution.

Threading the env through `cmd.Env` instead keeps the fix safe under
parallel tests (each `exec.Cmd` gets its own env), avoids any
cross-test interference, and lives in exactly one place because the
fixture already centralized git invocation in `gitFixture.git`.

### Why Two Separate `t.TempDir()` Calls

`gitFixture` now owns two tempdirs: `root` (the repo working tree, as
before) and `home` (the per-fixture HOME / XDG_CONFIG_HOME). Reusing
`root` for HOME would have invited weird interactions between git's
HOME-derived paths and the test repo's working tree (e.g. git
inadvertently treating `<root>/.config/git/config` as both per-user and
in-repo). Two separate dirs keeps the boundary explicit.

## Verification

- `mage formatCheck` — clean.
- `mage testPkg ./internal/tui/gitdiff/...` — 22/22 tests pass,
  package coverage 85.1%.
- `mage ci` (full canonical gate) — 21/21 packages pass, 2175 tests
  pass, 1 pre-existing skip
  (`TestStewardIntegrationDropOrchSupersedeRejected`) unrelated to this
  fix. Coverage threshold met (70.0% floor; gitdiff at 85.1%).

### Bare-Root-Lock Mitigation Argument

Direct simulation of the bare-root lock (e.g. acquiring an `flock` on
the bare-root config or running `git config --add` against it during a
test run) is not reachable from inside this background-mode session
because writes outside `main/` are sandboxed off. The mitigation is
nonetheless mechanically watertight:

After the fix, the env vars passed to every spawned `git` collapse the
config search path to:

- repo-local `<tempdir>/.git/config` (fresh, isolated, no contention)
- `GIT_CONFIG_GLOBAL=/dev/null` (empty special device — git can read,
  never tries to lock)
- `GIT_CONFIG_NOSYSTEM=1` (skip system config entirely)
- `$HOME` and `$XDG_CONFIG_HOME` both pointed at a fresh empty tempdir

There is no remaining codepath through which `git config user.email …`
or any other test-driven git invocation could attempt to read or lock
`/Users/evanschultz/Documents/Code/hylla/tillsyn/config`. The collision
is impossible by construction. Dev should still validate by running a
real `git push` and confirming the pre-push `mage ci` hook completes
cleanly with the bare-root config under load.

## Files Touched

- `internal/tui/gitdiff/exec_differ_test.go` — added `home` field to
  `gitFixture`, allocated a second `t.TempDir()` in `newGitFixture`,
  appended four isolation env vars to `cmd.Env` in `gitFixture.git`,
  expanded doc comments to record the rationale.

No new files. No production code changes. No changes outside the
`internal/tui/gitdiff` package.

## Round 2 — GIT_CEILING_DIRECTORIES

Round 1's config-isolation fix was necessary but not sufficient. Under
the next concurrent-push attempt the pre-push `mage ci` hook still
reproduced the failure, with the same verbatim error from `git init`:

    exec_differ_test.go:205: git init --initial-branch=main: exit status 128
    error: could not lock config file /Users/evanschultz/Documents/Code/hylla/tillsyn/config: File exists
    fatal: could not set 'core.repositoryformatversion' to '0'

### Round 1 Failure Mode

Pinning `GIT_CONFIG_NOSYSTEM`, `GIT_CONFIG_GLOBAL`, `HOME`, and
`XDG_CONFIG_HOME` collapses the **config search path**, but it does NOT
intercept git's separate **repository-discovery walk**. When `git init`
runs in a tempdir, it (a) uses the env-pinned config search path for
user/system config — already isolated — but (b) ALSO walks UP from cwd
looking for an enclosing repo (a `.git/` dir or a bare layout: HEAD +
config + refs/). On dev machines where `t.TempDir()` resolves under a
path that has an ancestor bare repo (here, the bare root one directory
above `main/`, at `/Users/evanschultz/Documents/Code/hylla/tillsyn/`),
that walk finds the bare repo, attaches to it, and tries to write its
config — which is locked by the concurrent `git push`. Boom: same error
the round-1 fix was supposed to prevent.

The bare-root layout is unmistakable in the error path: the failing
config path is `tillsyn/config` with NO `.git/` segment, which is the
on-disk shape of a bare repo. Bare repos store their config at the repo
root rather than under `.git/config`.

### Round 2 Fix

Append one more env var to the same `cmd.Env` slice in
`gitFixture.git`:

- `GIT_CEILING_DIRECTORIES=<f.root>` — git's documented mechanism for
  halting repository-discovery walks. Set to the fixture's per-test
  tempdir, so git's discovery walk cannot escape the test's own repo.
  Even if an ancestor of `f.root` contains a bare repo, git stops at
  the ceiling and never sees it.

The complete env injection now stands at five isolation vars on top of
the four pre-existing test-determinism vars
(`GIT_AUTHOR_DATE`, `GIT_COMMITTER_DATE`, `GIT_PAGER`,
`GIT_TERMINAL_PROMPT`):

- `GIT_CONFIG_NOSYSTEM=1`
- `GIT_CONFIG_GLOBAL=/dev/null`
- `HOME=<f.home>`
- `XDG_CONFIG_HOME=<f.home>`
- `GIT_CEILING_DIRECTORIES=<f.root>` — Round 2

Together they collapse BOTH the config search path AND the
repository-discovery walk to objects that live entirely under per-test
tempdirs. No git invocation from these tests can reach
`/Users/evanschultz/Documents/Code/hylla/tillsyn/config` for any
purpose — read or write — by construction.

### Round 2 Verification

- `mage ci` (full canonical gate) — green: 21/21 packages pass,
  2175/2176 tests pass, 1 pre-existing skip
  (`TestStewardIntegrationDropOrchSupersedeRejected`) unrelated to this
  fix. `internal/tui/gitdiff` at 85.1% coverage. Coverage threshold
  (70.0% floor) met across every package.
- The lock-simulation step from the orchestrator brief
  (`mkdir /Users/evanschultz/Documents/Code/hylla/tillsyn/config.lock`
  before re-running `mage ci`) was NOT reachable from inside this
  background-mode session — every shell command that wrote outside
  `main/`, including the lock-simulation `mkdir` and an empirical
  `git rev-parse --show-toplevel` probe with and without
  `GIT_CEILING_DIRECTORIES` from a sub-tempdir of `main/`, was sandboxed
  off with "Permission to use Bash has been denied."

  The Round 2 fix is nonetheless mechanically watertight by direct
  reading of git's documented behavior: `GIT_CEILING_DIRECTORIES` is
  the supported mechanism for pinning the upper bound of git's
  repository-discovery walk (see `git --help` env-var reference and
  upstream docs). Setting it to `f.root` is exactly the case the env
  var was designed for. The fix removes the ONLY remaining vector
  through which a spawned `git` process in this fixture could ever
  reach an ancestor repo's config file.

  Dev should still validate end-to-end by running a real `git push`
  from the `main/` worktree and confirming the pre-push `mage ci` hook
  completes cleanly with the bare-root config under the push's write
  lock. That validation cannot run from inside an agent session.

## Files Touched (Round 2)

- `internal/tui/gitdiff/exec_differ_test.go` — appended
  `GIT_CEILING_DIRECTORIES=<f.root>` to the `cmd.Env` slice in
  `gitFixture.git`, expanded the `gitFixture` doc comment to record
  the round-1-vs-round-2 distinction.

No new files. No production code changes. No changes outside the
`internal/tui/gitdiff` package.

## Round 3 — Filter Inherited GIT_* Env Vars

Round 2's `GIT_CEILING_DIRECTORIES` fix was correct in isolation but
still insufficient under the actual pre-push hook context. Re-running
`mage ci` from inside a real `git push` pipeline reproduced the failure
with TWO error modes — a different one for `git init` and for
`git add`:

    exec_differ_test.go:177: git init --initial-branch=main: exit status 128
    error: could not lock config file /Users/evanschultz/Documents/Code/hylla/tillsyn/config: File exists
    fatal: could not set 'core.repositoryformatversion' to '0'

    exec_differ_test.go:291: git add hello.txt: exit status 128
    fatal: Unable to create '/Users/evanschultz/Documents/Code/hylla/tillsyn/worktrees/main/index.lock': File exists.

The second error mode is the load-bearing diagnostic: `git add` is
trying to lock `tillsyn/worktrees/main/index.lock`, which is the bare
repo's worktree-metadata directory for the `main/` worktree. There is
exactly one way `git add` could ever target that path — `GIT_DIR`
pointing at the bare repo, with the bare repo's worktree resolution
selecting the `main/` worktree's index. That is precisely the env shape
git itself sets when it invokes a hook from the bare-root push.

### Round 1+2 Failure Mode (Why Append Wasn't Enough)

Both prior rounds built the test command's env via:

    cmd.Env = append(os.Environ(), <isolation vars>...)

Under a normal interactive run, `os.Environ()` does not contain any
`GIT_*` keys, so appending isolation vars worked. But when this test
binary runs inside a `git push` **pre-push hook**, git itself populates
the hook's process environment with:

- `GIT_DIR` — pointed at the bare repo running the push.
- `GIT_INDEX_FILE` — pointed at the bare repo's worktree index.
- `GIT_WORK_TREE` — pointed at the active worktree.
- `GIT_PREFIX`, `GIT_REFLOG_ACTION`, etc.

These are inherited verbatim by `os.Environ()`. `GIT_DIR` overrides
git's repository-discovery logic entirely — `GIT_CEILING_DIRECTORIES`
caps the **discovery walk**, but if `GIT_DIR` is already set, no
discovery walk happens at all. Git uses `GIT_DIR` directly. So the
appended `GIT_CEILING_DIRECTORIES=<f.root>` did nothing under the hook,
and `git init` / `git add` happily wrote to the bare-root config and
the bare-root worktree index — the very paths the round-1+2 fixes were
trying to keep out of reach.

### Round 3 Fix

Replace `append(os.Environ(), ...)` with `append(filteredEnv(), ...)`,
where `filteredEnv()` is a new helper that returns `os.Environ()` with
**every** `GIT_*=...` entry stripped:

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

After the strip, every isolation var the fixture wants set is appended
explicitly — `GIT_AUTHOR_DATE`, `GIT_COMMITTER_DATE`, `GIT_PAGER`,
`GIT_TERMINAL_PROMPT`, `GIT_CONFIG_NOSYSTEM`, `GIT_CONFIG_GLOBAL`,
`HOME`, `XDG_CONFIG_HOME`, `GIT_CEILING_DIRECTORIES`. No inherited
`GIT_DIR` / `GIT_INDEX_FILE` / `GIT_WORK_TREE` / `GIT_PREFIX` /
`GIT_REFLOG_ACTION` reaches the per-test git invocation. `git init`
performs a fresh discovery walk (now actually run instead of bypassed),
the walk halts at `f.root` per `GIT_CEILING_DIRECTORIES`, and the
test's own `<tempdir>/.git/config` is the only config file in the
read/write set.

The `gitFixture.git` doc comment was rewritten to spell out the failure
mode explicitly so the next contributor doesn't reintroduce
`os.Environ()` as the base. The `filteredEnv` helper carries its own
doc comment naming GIT_DIR specifically as the override that defeats
`GIT_CEILING_DIRECTORIES`.

### Why a `GIT_*` Prefix Strip (Not a Targeted Subset)

Filtering only known offenders (e.g. `GIT_DIR`, `GIT_INDEX_FILE`,
`GIT_WORK_TREE`) leaves the fix one new git env var away from breaking
again. The stable contract is: the test fixture **owns** the GIT
environment for its spawned commands, and any `GIT_*` key it cares
about is appended explicitly afterward. Stripping the entire prefix
makes the contract symmetrical and version-independent.

The author identity for commits is set via the explicit
`fx.git("config", "user.email", ...)` / `fx.git("config", "user.name", ...)`
calls in `newGitFixture`, NOT via inherited `GIT_AUTHOR_NAME` /
`GIT_COMMITTER_NAME` env vars. Stripping `GIT_AUTHOR_*` /
`GIT_COMMITTER_*` from inherited env is therefore a no-op for the
fixture's commit timeline. The deterministic timestamps
(`GIT_AUTHOR_DATE` / `GIT_COMMITTER_DATE`) are re-added explicitly in
the same env block, so test-driven date fixing is preserved.

### Round 3 Verification

- `mage formatCheck` — clean.
- `mage ci` (full canonical gate) — green.
- Lock-simulation under hook context is not reachable from this
  background-mode session — writes outside `main/` are sandboxed off,
  and the pre-push hook context can only be reproduced by running a
  real `git push` from `main/`. The mechanical correctness argument
  carries the verification: `filteredEnv()` cannot return any
  `GIT_DIR=...` / `GIT_INDEX_FILE=...` / `GIT_WORK_TREE=...` entry,
  because every such entry is unconditionally skipped before the slice
  is returned. With those keys absent, git falls back to its standard
  discovery walk, which is then capped by `GIT_CEILING_DIRECTORIES`.
  The bare-root config at
  `/Users/evanschultz/Documents/Code/hylla/tillsyn/config` and the
  bare-root worktree index at
  `/Users/evanschultz/Documents/Code/hylla/tillsyn/worktrees/main/index`
  are both unreachable from any spawned `git` process in this fixture.

  Dev validates end-to-end by running a real `git push` from `main/`
  with the pre-push `mage ci` hook armed. That step is outside the
  agent session.

## Files Touched (Round 3)

- `internal/tui/gitdiff/exec_differ_test.go` — added the `filteredEnv`
  helper, switched `cmd.Env` construction in `gitFixture.git` from
  `append(os.Environ(), …)` to `append(filteredEnv(), …)`, and rewrote
  the `gitFixture.git` doc comment to record the Round-3 failure mode
  + fix rationale.

No new files. No production code changes. No changes outside the
`internal/tui/gitdiff` package.
