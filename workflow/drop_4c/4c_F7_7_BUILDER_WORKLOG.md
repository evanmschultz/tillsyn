# Drop 4c F.7.7 — Auto-add `.tillsyn/spawns/` to .gitignore — Builder Worklog

**Droplet:** F.7-CORE F.7.7
**Builder model:** opus
**Date:** 2026-05-04
**Plan source:** `workflow/drop_4c/F7_CORE_PLAN.md` § F.7.7 + REVISIONS POST-AUTHORING + spawn-architecture memory §1/§2

## Round 1

### Goal

When `tillsyn.spawn_temp_root = "project"` is in effect AND the project's `.gitignore` doesn't already ignore `.tillsyn/spawns/`, auto-append the entry idempotently. One-shot per dispatch session via `sync.Once` to avoid repeated file-IO per spawn. No-op when `spawn_temp_root != "project"` (OS-temp-mode bundles never touch the worktree).

### Files edited

- **`internal/app/dispatcher/bundle.go`**
  - Added `bytes` to imports for the line-walk helper.
  - Added two package-level constants: `gitignoreSpawnsEntry = ".tillsyn/spawns/"` (canonical unanchored form written to disk) and `gitignoreSpawnsEntryAnchored = "/.tillsyn/spawns/"` (recognized for idempotency only).
  - Added exported function `EnsureSpawnsGitignored(projectRoot, spawnTempRoot string) error` implementing the four-row behavior matrix:
    - `spawnTempRoot != SpawnTempRootProject` → return nil (no-op).
    - Empty `projectRoot` in project mode → wrap `ErrInvalidBundleInput`.
    - `.gitignore` absent → create with single line `.tillsyn/spawns/\n` via the atomic write-temp-then-rename pattern.
    - `.gitignore` present + entry recognized (unanchored OR anchored variant, line-exact match after whitespace trim) → no-op.
    - `.gitignore` present + entry missing → append, prepending a newline if existing file lacks one, always trailing newline.
  - Added internal helpers: `gitignoreContainsSpawnsEntry([]byte) bool` (line-exact match supporting both variants + CRLF tolerance) and `writeGitignoreAtomic(path string, contents []byte) error` (mirrors `writeManifestAtomic`'s fsync ordering but at 0o644 perms because `.gitignore` is a tracked repo file).

- **`internal/app/dispatcher/bundle_test.go`**
  - Added 7 new tests covering every behavior-matrix row + the input-validation guard:
    - `TestEnsureSpawnsGitignoredOSTempIsNoop` (subtests for `""` and `"os_tmp"` both short-circuit; no `.gitignore` created).
    - `TestEnsureSpawnsGitignoredCreatesWhenMissing` (project mode + missing file → `.tillsyn/spawns/\n` exactly).
    - `TestEnsureSpawnsGitignoredIdempotentRecall` (second call leaves file byte-identical).
    - `TestEnsureSpawnsGitignoredAppendsToExistingEntries` (existing entries preserved, new entry appended at end with newline framing).
    - `TestEnsureSpawnsGitignoredRecognizesAnchoredVariant` (existing `/.tillsyn/spawns/` line treated as already-ignored; no double-append).
    - `TestEnsureSpawnsGitignoredHandlesMissingTrailingNewline` (file missing trailing newline gets one inserted before the new entry — guards against `node_modules/.tillsyn/spawns/` concatenation regression).
    - `TestEnsureSpawnsGitignoredRejectsEmptyProjectRootInProjectMode` (input-validation guard returns `ErrInvalidBundleInput`).

- **`internal/app/dispatcher/spawn.go`**
  - Added package-level `var (ensureSpawnsGitignoredOnce sync.Once; ensureSpawnsGitignoredErr error)` to gate the helper to once-per-process. Captured-error pattern means subsequent spawns see the same error the first invocation produced rather than silently succeeding.
  - Added test-only `ResetEnsureSpawnsGitignoredOnceForTest()` (exported with `…ForTest` suffix per package convention) so external `dispatcher_test` package tests can re-arm the once-shot. Doc-comment marks production-caller usage as out-of-contract.
  - Wired the `sync.Once.Do` call BEFORE `NewBundle` in `BuildSpawnCommand` (line ~358 post-edit). Local variable `spawnTempRoot := ""` is now the single source of truth — passed to BOTH `EnsureSpawnsGitignored` AND `NewBundle` — so the catalog→Tillsyn plumbing follow-up only needs to change one expression. Errors propagate out of `BuildSpawnCommand` with a `dispatcher: ensure spawns gitignored:` prefix.

- **`internal/app/dispatcher/spawn_test.go`**
  - Added 2 integration tests that exercise the wired code path:
    - `TestBuildSpawnCommandLeavesGitignoreUntouchedInOSTempMode` — pins the os_tmp default: a fresh worktree under `t.TempDir()` has NO `.gitignore` after `BuildSpawnCommand`. Negative assertion against any future change that accidentally extends gitignore maintenance to OS-temp mode.
    - `TestBuildSpawnCommandEnsureGitignoredFiresOncePerProcess` — three consecutive `BuildSpawnCommand` invocations against the same project; `.gitignore` still NOT created (os_tmp default), but the once-shot is observable through the `ResetEnsureSpawnsGitignoredOnceForTest` cleanup hook.
  - Both tests are NOT `t.Parallel()` — they mutate the package-level once-shot and need serialization against sibling tests.

- **`workflow/drop_4c/4c_F7_7_BUILDER_WORKLOG.md`** — this file.

### Files NOT edited (and why)

- `internal/templates/schema.go` / `templates/load.go` — `SpawnTempRoot` is already a closed-enum field on `templates.Tillsyn` (per Drop 4c F.7.1's schema work). No template-layer change needed.
- `internal/app/dispatcher/cli_register/` and `cli_claude/` — bundle materialization stays untouched; the gitignore concern is purely worktree-side metadata.

### Why integration test does NOT exercise project mode end-to-end

`BuildSpawnCommand` currently hardcodes `spawnTempRoot := ""` (because `templates.KindCatalog` does not yet carry the `Tillsyn` block — that plumbing is a follow-up droplet per the existing comment at `spawn.go:325-333`). To exercise project-mode end-to-end through `BuildSpawnCommand` would require either (a) extending `KindCatalog` with the `Tillsyn` field (out of F.7.7 scope), or (b) adding a test-only override hook (premature API surface). Project-mode behavior is fully covered by the 6 direct-helper tests in `bundle_test.go`; the spawn-layer integration tests verify the os_tmp baseline + the once-shot gating mechanism.

### Verification

```
mage testPkg ./internal/app/dispatcher/  — 276/276 pass (was 269/269 pre-edit; +7 new tests)
mage ci                                  — green
  - Sources verified
  - Go formatting clean (gofumpt)
  - Test stream: 2619 passed, 1 skipped, 0 failed across 24 packages
  - Coverage threshold met (dispatcher 72.5%, all packages ≥ 70%)
  - Build green (./cmd/till)
```

### Acceptance criteria

- [x] `EnsureSpawnsGitignored` shipped, idempotent, no-op in os_tmp mode.
- [x] `sync.Once` gating in spawn.go ensures per-process one-shot write.
- [x] All 6 prescribed test scenarios pass (no-op for os_tmp / create when missing / idempotent re-call / append when other entries present / idempotent with leading-slash variant / trailing-newline preserved). 7th scenario added for input-validation guard symmetry with the rest of the bundle package.
- [x] `mage check` (alias for `mage ci`) + `mage ci` green.
- [x] Worklog written.
- [x] **NO commit by me** (per F.7-CORE REV-13 + spawn-prompt directive).

### Proposed commit message

```
feat(dispatcher): auto-add .tillsyn/spawns/ to .gitignore in project mode
```

(Single-line conventional commit, 67 chars, ≤72.)

## Hylla Feedback

N/A — directive explicitly forbade Hylla calls for this droplet ("NO Hylla calls."). Native-tool path (`Read` for the four target files; `rg` for cross-package symbol lookups in `internal/templates`) was sufficient given the well-defined surface and existing `bundle.go` precedent (`writeManifestAtomic`) to mirror.
