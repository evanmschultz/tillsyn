# P4-T4 Builder Worklog

Append `## Round K` per build attempt. Each round: files touched, mage targets run, design notes, `## Hylla Feedback` subsection.

## Round 1

### Files Touched

- `internal/tui/diff_mode.go` — added `domain` import; added `activeItem *domain.Task` field to `diffMode`; added `SetItem(*domain.Task)` method; added `resolveDiffPaths(*domain.Task) []string` pure function; updated `enterDiffMode()` to call `SetItem` + `resolveDiffPaths` instead of passing `nil` paths.
- `internal/tui/diff_mode_test.go` — extended `fakeDiffer` with `lastPaths []string` field and updated `Diff` to capture paths; added `newDiffTestTask` helper; added `refWith` helper; added 11 new tests covering all PLAN.md TDD test list items.

### Mage Targets

```
mage test-pkg ./internal/tui/...
  tests: 379, passed: 379, failed: 0 (was 368 pre-P4-T4; +11 new tests)

mage ci
  tests: 1323, passed: 1323, failed: 0
  internal/tui coverage: 70.8% (above 70% floor)
  Build: OK
  Formatting: OK (gofumpt applied to diff_mode_test.go and diff_mode.go)
```

### Design Notes

- `resolveDiffPaths` is unexported, pure, no receiver. Takes `*domain.Task` (not `*domain.PlanItem` — the PLAN.md description uses "PlanItem" as a semantic term but the domain type is `domain.Task`).
- Dedup logic: `seenIdx` map keyed on bare location (no trailing slash). Package refs upgrade path/file entries in-place at the same slice index, so ordering is preserved and package wins.
- `SetItem` on `*diffMode` stores the active item pointer. `enterDiffMode()` calls `selectedTaskInCurrentColumn()`, stores the pointer via `SetItem`, and passes `resolveDiffPaths(activeTask)` to `diffModeCmd`. No new `Model` fields added (accepted: 0 new top-level Model fields).
- `TestDiffMode_RecomputesOnItemChange` reloads the model with updated fake service state to simulate resource refs changing between sessions — avoids internal state mutation that would couple the test to implementation details.

### Known Domain Type Discrepancy

PLAN.md acceptance criteria reference `*domain.PlanItem` — this type does not exist in the codebase. The actual type is `*domain.Task`. Implementation uses `*domain.Task` throughout. This is not a drift; PLAN.md used "PlanItem" as a semantic alias. No action needed.

## Hylla Feedback

None — Hylla answered everything needed. All Go-code reads used Hylla (`hylla_search`, `hylla_node_full`) for the domain types (ResourceRef struct, TaskMetadata.ResourceRefs field, domain.Task struct). LSP was used for live symbol resolution on model.go helpers (selectedTaskInCurrentColumn, enterDiffMode call sites). The one Hylla-adjacent note: Hylla does not index non-Go files (markdown, TOML, magefile) — Read/Glob used directly for those as expected.

## Round 2

### Files Touched

- `internal/tui/diff_mode.go` — added `resolvePaths()` method on `*diffMode` (Fix 1, Option B); changed `enterDiffMode` line 385 from `resolveDiffPaths(activeTask)` to `m.diff.resolvePaths()`, wiring `d.activeItem` into the compute path and making `SetItem` load-bearing.
- `internal/tui/diff_mode_test.go` — added `TestResolveDiffPaths_PackageFirstThenPath` covering the package-first, path-second dedup ordering variant (Fix 2).

### What Changed vs Round 1

Round 1 stored the active item via `SetItem` but computed paths from the local `activeTask` variable, leaving `d.activeItem` dead. Round 2 adds a `resolvePaths()` method that reads `d.activeItem` and routes `enterDiffMode` through it. The behavior is identical at runtime (same pointer, same call site, sequential assignment then read) but `d.activeItem` is now live and `SetItem` is genuinely load-bearing. The new test covers the one untested permutation of the package-wins dedup contract.

### Mage Targets

```
mage test-pkg ./internal/tui/...
  tests: 380, passed: 380, failed: 0 (was 379; +1 new test TestResolveDiffPaths_PackageFirstThenPath)

mage ci
  tests: 1324, passed: 1324, failed: 0 (was 1323)
  internal/tui coverage: 70.9% (above 70% floor)
  Build: OK
  Formatting: OK
```

### Hylla Feedback

N/A — task touched only `drop/1.5` files not yet merged to `main`. Hylla `github.com/evanmschultz/tillsyn@main` is stale for all P4-T3/P4-T4 content. All Go-code reads used `Read` and `LSP` directly per the expected pattern (Hylla ingest is drop-end only).
