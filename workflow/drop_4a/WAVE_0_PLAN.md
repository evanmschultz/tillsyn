# DROP_4A — WAVE 0 — DEV HYGIENE INFRASTRUCTURE

**State:** planning
**Wave position:** Wave 0 (lands FIRST in Drop 4a sequence; Wave 1 → Wave 2 → Wave 3 → Wave 4 follow).
**Paths (expected):** `magefile.go`, `.githooks/pre-commit` (NEW), `.githooks/pre-push` (NEW), `CONTRIBUTING.md`
**Packages (expected):** `main` (the magefile package — `//go:build mage`)
**REVISION_BRIEF ref:** `workflow/drop_4a/REVISION_BRIEF.md` § "Wave 0 — Dev hygiene infrastructure"
**Started:** 2026-05-03
**Closed:** —

## Wave Purpose

Wave 0 ships the dev-hygiene tooling that subsequent Drop 4a waves consume. Every builder spawned in Waves 1-4 will commit through `.githooks/pre-commit` (running `mage format-check`) and push through `.githooks/pre-push` (running `mage ci`), mirroring exactly what GitHub Actions runs. This catches format drift + CI breakage locally before commits land, before push, before CI burn — the cheapest possible feedback loop for the dispatcher-build work that follows.

The wave also fixes a pre-existing ergonomic bug: `func Format(path string) error` (`magefile.go:199`) requires a positional argument, so plain `mage format` errors with "not enough arguments." Since `Format` is mage's entire formatter surface and `trackedGoFiles()` already exists for whole-tree mode, splitting into `Format()` (no-arg, whole tree) and `FormatPath(path string)` (scoped) restores the obvious ergonomic without breaking the path-scoped use-case.

## Wave Decomposition

Four droplets in a strict linear chain. The chain is mandatory because (i) W0.1 and W0.3 both edit `magefile.go` (same-file lock), and (ii) every downstream droplet consumes the deliverable of the prior one — pre-commit script needs `mage format-check`, install-hooks needs the hook files, docs reference the install target.

| ID    | Title                                            | Blocked by  |
| ----- | ------------------------------------------------ | ----------- |
| W0.1  | `MAGE FORMAT` ERGONOMICS + `MAGE FORMAT-CHECK`   | —           |
| W0.2  | `.GITHOOKS/` PRE-COMMIT + PRE-PUSH SCRIPTS       | W0.1        |
| W0.3  | `MAGE INSTALL-HOOKS` TARGET                      | W0.2        |
| W0.4  | CONTRIBUTING.MD HOOKS DOCS                       | W0.3        |

Total: 4 droplets.

## Droplet Decomposition

### W0.1 — `MAGE FORMAT` ERGONOMICS + `MAGE FORMAT-CHECK`

- **State:** todo
- **Paths:** `magefile.go`
- **Packages:** `main` (magefile build tag)
- **Acceptance:**
  - Existing exported `Format(path string) error` at `magefile.go:199` is split into TWO public targets:
    - `Format() error` — no-arg, formats every tracked Go file by calling `trackedGoFiles()` (already at `magefile.go:307`) then `runGofumptWrite(files)`. Returns early when `trackedGoFiles()` returns empty (mirrors existing line 206-208 behaviour).
    - `FormatPath(path string) error` — preserves the prior single-path / single-directory semantics: trim, reject empty, `os.Stat` check, `runGofumptWrite([]string{path})`.
  - **No variadic form.** Mage's variadic-arg discovery is fragile across versions; a clean two-function split is unambiguous.
  - New public `FormatCheck() error` — single-line wrapper that calls the existing private `formatCheck()` at `magefile.go:218`. Doc comment cites: "Public wrapper for the private `formatCheck` gate so `.githooks/pre-commit` can invoke `mage format-check` without depending on internal helpers."
  - `Aliases` map at `magefile.go:26-34` updated:
    - Existing `"fmt": Format` survives (now points at the new no-arg `Format`; both invocations `mage fmt` and `mage format` work).
    - NEW entry `"format-check": FormatCheck` so the kebab-case CLI name (`mage format-check`) maps to the camel-case Go identifier per existing alias precedent (`"test-golden": TestGolden`, etc.).
  - Private `formatCheck()` (lines 218-235) is NOT renamed or deleted — `CI()` at `magefile.go:155` continues to call it directly through its `formatCheck` reference. Only `FormatCheck` is added; `formatCheck` stays.
  - Doc comments on every new top-level function per project rule.
  - Existing call sites in this file: `Aliases["fmt"]` (line 33), `CI()` stage table (line 155 references `formatCheck`). No other in-tree call sites to migrate.
  - **Verification:**
    - `mage` (Default = CI) green.
    - `mage format` (no arg) succeeds and reformats every tracked Go file (or no-ops if everything is already gofumpt-clean).
    - `mage format ./magefile.go` succeeds (path-scoped via the new `FormatPath` if the alias dispatch supports it; otherwise document `mage format-path ./magefile.go` as the explicit CLI form — builder picks based on mage v1.17 behaviour).
    - `mage format-check` succeeds clean and emits non-zero exit + filename list when a tracked file is dirty (mirror an intentional dirty-test in the worklog, then revert).
    - `mage ci` green (private `formatCheck` still wired into CI stage table).
- **Blocked by:** —
- **Notes:** Single-file edit, ~25-line LOC delta (split + new wrapper + alias). Builder must verify mage's CLI-name resolution for `FormatPath` — if `mage format ./path` does NOT route to `FormatPath` automatically, expose it as `mage format-path ./path` via the alias map (`"format-path": FormatPath`) and document. Either form is acceptable; the no-arg `mage format` ergonomic fix is the load-bearing acceptance.

### W0.2 — `.GITHOOKS/` PRE-COMMIT + PRE-PUSH SCRIPTS

- **State:** todo
- **Paths:** `.githooks/pre-commit` (NEW), `.githooks/pre-push` (NEW)
- **Packages:** — (shell scripts; no Go package)
- **Acceptance:**
  - Both scripts written in **POSIX `sh`** (not Bash) for portability across macOS dev + Linux CI testers. Shebang `#!/bin/sh`. `set -eu` (no `-o pipefail` — POSIX `sh` lacks pipefail).
  - `.githooks/pre-commit`:
    - Single `mage format-check` invocation, exit non-zero on failure.
    - On failure, prints a one-line hint: `format-check failed — run 'mage format' to auto-fix, or '--no-verify' to bypass with explicit reason.` (Bypass guidance is informational; honoring `--no-verify` requires no script logic — git handles the bypass at invocation time.)
  - `.githooks/pre-push`:
    - Single `mage ci` invocation, exit non-zero on failure.
    - On failure, prints: `mage ci failed — fix locally before pushing, or '--no-verify' to bypass with explicit reason.`
  - Both files committed with mode `0755` (executable). The build that adds them MUST verify executability via `git ls-files -s .githooks/` (mode column shows `100755`).
  - Both files contain a top-of-file comment naming the install command: `# Activated by 'mage install-hooks' (sets core.hooksPath = .githooks).`
  - **No** repo-name / branch-name / abs-path hardcoding — scripts must work in any clone of the repo (including future worktrees).
  - **No** Go-toolchain version pinning, `GOPROXY` overrides, or `GOMODCACHE` overrides — scripts must respect the dev's configured environment per `feedback_mage_precommit_ci_parity.md`.
  - **Verification:**
    - `git ls-files -s .githooks/pre-commit .githooks/pre-push` shows both as `100755`.
    - With `core.hooksPath = .githooks` set manually (`git config core.hooksPath .githooks`), make a trivial dirty commit on a temp branch in the worktree and verify the pre-commit hook runs `mage format-check` (worklog the run + the `git config --unset core.hooksPath` rollback so dev state is preserved — formal install lands in W0.3).
- **Blocked by:** W0.1 (pre-commit script invokes `mage format-check`; that target must exist before this droplet's verification step can pass).
- **Notes:** `.githooks/` is a NEW directory. No existing path conflict. Pre-MVP rule: builder does NOT also run `git config core.hooksPath` from a mage step in this droplet — that's W0.3's scope.

### W0.3 — `MAGE INSTALL-HOOKS` TARGET

- **State:** todo
- **Paths:** `magefile.go`
- **Packages:** `main` (magefile build tag)
- **Acceptance:**
  - New public `InstallHooks() error` mage target. Doc comment: "Sets `core.hooksPath = .githooks` for this clone so the tracked `.githooks/pre-commit` and `.githooks/pre-push` scripts run on every commit/push. One-time-per-clone — re-running is idempotent (git overwrites the existing config value with the same value)."
  - Implementation: single `runCommand("git", "config", "core.hooksPath", ".githooks")` invocation, returning its error. Use `runCommand` (`magefile.go:367`) — NOT `captureCommand` — so any git-config error chatter surfaces directly to the dev.
  - **Pre-flight assertion:** before the `git config` call, `os.Stat(".githooks/pre-commit")` and `os.Stat(".githooks/pre-push")`. If either is missing, return `fmt.Errorf("install-hooks: %s missing — run from the worktree root and ensure .githooks/ files are tracked", path)`. This guards against running in a stale clone where W0.2's files haven't been pulled.
  - **NOT idempotent state-side beyond what git provides** — `git config core.hooksPath .githooks` always overwrites the prior value with the same value, no error. Acceptance asserts: re-running `mage install-hooks` twice in a row succeeds both times with no diagnostic output.
  - `Aliases` map at `magefile.go:26-34` updated: NEW entry `"install-hooks": InstallHooks`.
  - **Does NOT auto-run during `mage ci`, `mage build`, or any other mage target.** Activation is a deliberate dev gesture, not implicit.
  - **Verification:**
    - `mage install-hooks` succeeds in a fresh clone (or after `git config --unset core.hooksPath`); `git config --get core.hooksPath` returns `.githooks`.
    - `mage install-hooks` succeeds again immediately after — idempotent.
    - With `.githooks/pre-commit` deleted in a temp branch, `mage install-hooks` fails with the pre-flight error message before touching `git config`.
    - `mage ci` green (no regression on the existing gate).
- **Blocked by:** W0.2 (the pre-flight `os.Stat` assertion + the meaning of `core.hooksPath = .githooks` both depend on the hook files existing).
- **Notes:** Single-file edit on `magefile.go` (~30-line LOC delta). **File-level conflict with W0.1** — both edit `magefile.go`. The W0.1 → W0.2 → W0.3 blocker chain serializes them transitively, satisfying the same-file-lock rule (`CLAUDE.md` § Cascade Tree Structure → "File- and package-level blocking").

### W0.4 — CONTRIBUTING.MD HOOKS DOCS

- **State:** todo
- **Paths:** `CONTRIBUTING.md`
- **Packages:** — (markdown only)
- **Acceptance:**
  - **Replace** the existing `## Recommended Pre-Push Hook` section (`CONTRIBUTING.md:42-57`) with a new `## Local Git Hooks` section. The current text instructs the dev to write a hook to `.git/hooks/pre-push` directly — that's untracked, divergent across machines, and now superseded by the tracked `.githooks/` story. Do NOT leave both documented; the old section MUST be deleted, not appended-to.
  - New `## Local Git Hooks` section documents:
    - Two tracked hook files at `.githooks/pre-commit` (runs `mage format-check`) and `.githooks/pre-push` (runs `mage ci`).
    - One-time-per-clone activation: `mage install-hooks` (sets `core.hooksPath = .githooks`).
    - Bypass policy: `git commit --no-verify` / `git push --no-verify` are honored by git natively. Per dev discipline, never bypass without an explicit reason captured in the commit message or PR description.
    - Sanity check: `git config --get core.hooksPath` returns `.githooks` after install.
    - One-line note that pre-push runs the same `mage ci` GitHub Actions runs, so a green pre-push run is a strong predictor of green CI.
  - Update the mage target list at `CONTRIBUTING.md:19-26`: add bullets for `mage format-check` (CI-parity format gate) and `mage install-hooks` (one-time-per-clone hook activation). Keep existing entries unchanged.
  - The existing `## Local Workflow` section (lines 5-27), the Windows Note (lines 28-40), the GitHub Actions Model section (lines 64-72), the Branch Protection Recommendation (lines 74-80), and the Dev MCP Server Setup section (lines 86-122) are NOT touched.
  - **Verification:**
    - `Read CONTRIBUTING.md` shows the new `## Local Git Hooks` section in place of the old `## Recommended Pre-Push Hook` section.
    - The new mage target list includes `format-check` and `install-hooks`.
    - Markdown links/anchors elsewhere in the repo are not broken — quick scan via `Grep` for `Recommended Pre-Push Hook` returns zero hits anywhere in `main/` (legacy section name fully retired).
- **Blocked by:** W0.3 (docs reference `mage install-hooks` which must exist; docs reference the hook scripts which must exist).
- **Notes:** Pure markdown — no Hylla, no `mage ci` impact. Builder verification is `Read` + `Grep`-based. Per `feedback_md_update_qa.md`, builder self-QAs the edit for cross-reference consistency before handoff to QA twins.

## Sequencing And Cross-Wave Concerns

- 0.1 Linear chain `W0.1 → W0.2 → W0.3 → W0.4` is mandatory; no parallelism within Wave 0.
- 0.2 Wave 0 must close (all 4 droplets in `complete`, both QA twins green per droplet) before Wave 1 dispatches. This is the locked decision **L6** in `REVISION_BRIEF.md`.
- 0.3 Once Wave 0 lands, Wave 1+ builders run with `.githooks/pre-commit` active, so any builder that ships gofumpt-dirty code gets caught at commit time before it ever reaches QA. This is the load-bearing reason for Wave 0's sequencing position.
- 0.4 No interaction with Tillsyn-runtime today (filesystem-MD only). Drop 4a's dispatcher-spawn primitives land in Wave 2 and consume nothing from Wave 0 directly — the dependency is purely operational (catch format/CI drift cheap, before Wave 1+ build work commits).

## Risks

- 0.1 **Mage CLI variadic / kebab-case behaviour** — W0.1's split assumes `mage format` (no arg) routes to `Format()`. If mage v1.17.0's CLI dispatch surprises the builder (e.g. requires an explicit alias for the no-arg case), the builder updates the `Aliases` map accordingly and documents in W0.4. Acceptance is "no-arg `mage format` works"; the implementation path stays builder-decided.
- 0.2 **POSIX `sh` portability** — POSIX `sh` lacks `pipefail`; `set -eu` is the strictest portable mode. If a future hook needs piped commands, the script-author falls back to detecting `bash` and `set -o pipefail` conditionally. Out of scope for Wave 0.
- 0.3 **`core.hooksPath` shadowing local hooks** — devs with existing `.git/hooks/pre-commit` lose them on `mage install-hooks`. CONTRIBUTING.md (W0.4) flags this with a one-line note: "If you have local hooks in `.git/hooks/`, copy them into `.githooks/` before running `mage install-hooks`, since `core.hooksPath` overrides the default lookup."
- 0.4 **`--no-verify` bypass discipline** — system-honored, but per dev discipline (`feedback_qa_before_commit.md`), bypass requires an explicit reason captured in the commit message or PR. W0.4 documents this; no code enforces it.

## Hylla Feedback

None — Hylla answered everything needed. (Wave 0 scope is markdown + magefile + shell — no Go-symbol queries issued; Hylla is Go-only today per `feedback_hylla_go_only_today.md`. Code understanding came from `Read` on `magefile.go`, `CONTRIBUTING.md`, and `workflow/drop_3/PLAN.md` for format reference.)
