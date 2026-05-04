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
