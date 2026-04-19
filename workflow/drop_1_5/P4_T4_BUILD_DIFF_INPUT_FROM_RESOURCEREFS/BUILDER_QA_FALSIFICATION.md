# P4-T4 Builder QA — Falsification

Append `## Round K` per falsification-QA pass. Each round: verdict (PASS/FAIL), attack vector results (mitigated / accepted / broke-through), Hylla feedback if any.

## Round 1

**Verdict: PASS** — no unmitigated counterexample constructed.

### Attack Vectors Attempted

| # | Vector | Outcome |
|---|--------|---------|
| 1 | `domain.PlanItem` vs `domain.Task` drift | REFUTED — `domain.PlanItem` does not exist as a Go type anywhere in `internal/`; builder's `*domain.Task` usage is correct. |
| 2 | Dedup "package wins" ordering bugs (`[package, path]`, `[path, package, path]`, etc.) | REFUTED — manual trace of all permutations: all converge. `path/file` branch checks `HasSuffix("/")` on existing entry and skips correctly. No ordering-sensitive bug. |
| 3 | `nil` vs empty `Tags` | REFUTED — `len(ref.Tags) == 0` handles both nil-slice and empty-length-slice identically in Go. |
| 4 | ResourceRef mutation invariant | REFUTED — `resolveDiffPaths` only reads `ref.Tags[0]` and `ref.Location`; writes go only to local `out []string` and `seenIdx map[string]int`. `item.Metadata.ResourceRefs` is untouched. |
| 5 | `TestDiffMode_RecomputesOnItemChange` is a weak proof of "no caching" | Accepted as sound — the test reloads the model with updated service state, which is the realistic path given Bubble Tea's value-type semantics. "No caching" is correctly implemented because `enterDiffMode` always calls `resolveDiffPaths(activeTask)` with the fresh result from `selectedTaskInCurrentColumn()`. |
| 6 | `enterDiffMode` nil-task dispatch (no selected task) | REFUTED — `activeTask` stays `nil`, `SetItem(nil)` guards on `d == nil`, `resolveDiffPaths(nil)` returns nil immediately, `diffModeCmd` passes nil paths → whole-repo fallback. No panic. |
| 7 | `package=""` / `package="."` edge cases | REFUTED in production — `normalizeTaskMetadata` (called by `domain.NewTask`) skips any ResourceRef with empty Location at workitem.go:273-275. Empty-Location refs cannot reach `resolveDiffPaths` via a real task. `"."+"/"="./"`  is not a production path. |
| 8 | Service interface method count ≠ 44 | REFUTED — counted lines 35-78 of `model.go` inclusive = 44 methods. P4-T4 diff touches only `diff_mode.go` and `diff_mode_test.go`; no `model.go` changes. |
| 9 | New top-level `Model` fields added | REFUTED — P4-T4 git diff (`0e22cdf`+`60b6fc5`) shows zero changes to `model.go`. `diff *diffMode` and `diffBackMode inputMode` are P4-T3 fields. |
| 10 | Hylla staleness misled builder decisions | REFUTED — P4-T3/T4 commits are on `drop/1.5`, not merged to `@main`. Hylla was used for pre-existing domain types (correctly); LSP for live symbol resolution. No stale-Hylla misread found. |
| 11 | `diffModeCmd` signature change breaks upstream callers | REFUTED — `diffModeCmd` has exactly one call site (`enterDiffMode` in same file). No external callers. No test breakage from adding `paths []string` parameter. |
| 12 | Concurrency race: `SetItem` + async `Differ.Diff` | REFUTED — `paths` is captured by value into the `diffModeCmd` closure at call time. `d.activeItem` is never accessed inside the async closure. No race possible. |
| 13 | `activeItem` field on `diffMode` is dead / `SetItem` is a no-op on path computation | Structural concern, NOT a counterexample. `d.activeItem` is written at `diff_mode.go:69` and never read anywhere — confirmed by grepping `activeItem` across all of `internal/tui/` (3 hits, all in `diff_mode.go`). `enterDiffMode` calls `resolveDiffPaths(activeTask)` directly with the local variable, not via `d.activeItem`. `SetItem` satisfies the acceptance criterion ("called from Model before entering modeDiff") but the stored value has no downstream effect. The dangling pointer also retains the task value in memory until the next `SetItem` or `diffMode` disposal. Routed as a cleanup item for orchestrator. |
| 14 | Missing test for `[package="foo", path="foo"]` order (package-first dedup variant) | Test coverage gap, NOT a counterexample. `TestResolveDiffPaths_PackageWinsOverPath` only tests path-first, package-second. The reverse order is correctly handled by the `path/file` branch's `HasSuffix("/")` check but has no dedicated test. Code is correct; gap is coverage-only. Routed as a note to orchestrator. |

### mage Verification

- `mage test-pkg ./internal/tui/...` → 379 tests, 0 failures (run directly against `drop/1.5` worktree).
- `mage ci` → 1323 tests, 0 failures, `internal/tui` at 70.8% (above 70% floor), build OK, formatting OK.

### Conclusions

All 14 attack vectors attempted. Zero CONFIRMED counterexamples. Two non-falsifying items routed to orchestrator:
- **Cleanup**: `diffMode.activeItem` is a dead field (written by `SetItem`, never read). Minor memory retention and future-maintainer confusion. No acceptance-criterion violation.
- **Test coverage note**: Missing test case for `[package="foo", path="foo"]` (package-first, then same-location path ref). Code is correct; only the test coverage is missing.

### Verdict

**PASS** — P4-T4 claim stands. `resolveDiffPaths` correctly implements the partition/dedup/package-wins contract. `SetItem` is called correctly. Paths are freshly computed on each `enterDiffMode` entry. Service interface at 44 methods. Zero new Model fields. `mage ci` green.

## Hylla Feedback

N/A — P4-T4 changes are on `drop/1.5`, not yet merged to `main`. Hylla `github.com/evanmschultz/tillsyn@main` is entirely stale for all P4-T3/P4-T4 content. All Go code reads used `Read`, `Grep`, and `LSP` directly. No Hylla queries were attempted for P4-T4 code because the ingest precondition (push to main + CI green) has not been met. This is the expected pattern per project CLAUDE.md §"Hylla Baseline" — Hylla ingest happens once per drop at the DROP N END — LEDGER UPDATE task.

## Round 2

**Verdict: PASS** — no unmitigated counterexample constructed.

### Attack Vectors Attempted

| # | Vector | Outcome |
|---|--------|---------|
| 1 | `resolvePaths()` nil-receiver safety | REFUTED — `diff_mode.go:77` guards `if d == nil { return nil }` before reading `d.activeItem`. |
| 2 | `SetItem` → `resolvePaths()` indirection, re-assignment window | REFUTED — lines 395-396 are sequential with no intervening reassignment of `m.diff` or `d.activeItem`. Same-pointer identity. |
| 3 | Old local-var pattern (`resolveDiffPaths(activeTask)`) surviving in production | REFUTED — grep of all non-test `*.go` files under `internal/tui/` shows zero occurrences. Only production call is `resolveDiffPaths(d.activeItem)` at `diff_mode.go:80` inside `resolvePaths()`. |
| 4 | Rapid-succession `SetItem` calls | REFUTED — Bubble Tea's Update loop is single-goroutine; no concurrent dispatch path. |
| 5 | `TestResolveDiffPaths_PackageFirstThenPath` test correctness | REFUTED — test constructs exactly one tag per `refWith`, expects `len(got) == 1` AND `got[0] == "internal/tui/"` via exact string comparison. Assertion is tight and correct. |
| 6 | Tags construction in new test matches `Tags[0]` single-tag contract | REFUTED — each `refWith` in the new test passes exactly one tag string; `Tags[0]` access is safe. |
| 7 | Regression — `TestDiffMode_SetItem_PassesResolvedPaths` | REFUTED — test now exercises full `SetItem → resolvePaths → resolveDiffPaths(d.activeItem)` chain; 380/380 passing. |
| 8 | Regression — `TestDiffMode_RecomputesOnItemChange` | REFUTED — still passing; behavior unchanged (fresh call to `selectedTaskInCurrentColumn()` on each `enterDiffMode` entry). |
| 9 | Nil-chain: `selectedTaskInCurrentColumn()` nil → `SetItem(nil)` → `resolvePaths()` → `resolveDiffPaths(nil)` | REFUTED — each hop has its own nil guard; entire chain terminates safely with nil/empty paths → whole-repo fallback. |
| 10 | Concurrency: `resolvePaths` reads `d.activeItem`; async `diffModeCmd` goroutine | REFUTED — `diffModeCmd` goroutine captures `paths []string` by value; `d.activeItem` is never accessed inside the goroutine. No race. |
| 11 | Service interface method count ≠ 44 | REFUTED — `model.go` unchanged in Round 2 (empty `git diff HEAD`); still 44 methods. |
| 12 | New top-level Model fields | REFUTED — `model.go` not in Round 2 diff; zero new fields. |
| 13 | PLAN.md criterion 6 strict-reading spec-drift | Accepted as non-counterexample — TUI architecture requires async `tea.Cmd`; loose reading ("SetItem's stored item drives Diff") is correct and consistent with Round 1 verdict. Round 2 actually tightens the contract by making `d.activeItem` the actual source. |
| 14 | Memory retention of `d.activeItem` pointer | Accepted as benign — task already retained by service layer; second reference in bounded TUI context is not a leak pattern. |

### mage Verification

- `mage test-pkg ./internal/tui/...` → 380 tests, 0 failures (+1 new test `TestResolveDiffPaths_PackageFirstThenPath`).
- `mage ci` → 1324 tests, 0 failures, `internal/tui` at 70.9% (above 70% floor), build OK, formatting OK.

### Conclusions

All 14 attack vectors attempted. Zero CONFIRMED counterexamples. Round 1's two non-falsifying items are resolved:

- **Fix 1 resolved**: `d.activeItem` is now load-bearing. `SetItem` writes it; `resolvePaths()` reads it as the sole source for path computation in `enterDiffMode`. The old direct local-variable call pattern is fully removed from production code.
- **Fix 2 resolved**: `TestResolveDiffPaths_PackageFirstThenPath` covers the package-first ordering with exact-value, exact-count assertions. No test-quality issues.

No regressions on any Round 1 pass vectors.

### Verdict

**PASS** — P4-T4 Round 2 claim stands. `SetItem` is genuinely load-bearing via `resolvePaths()`. All partition/dedup/package-wins contract cases are covered with tight assertions. Service interface at 44 methods. Zero new Model fields. `mage ci` green.

## Hylla Feedback

N/A — P4-T4 changes are on `drop/1.5`, not yet merged to `main`. Hylla `github.com/evanmschultz/tillsyn@main` is stale for all P4-T3/P4-T4 content. All Go-code reads used `Read`, `Grep`, and `LSP` directly per the expected pattern. Hylla ingest is drop-end only.
