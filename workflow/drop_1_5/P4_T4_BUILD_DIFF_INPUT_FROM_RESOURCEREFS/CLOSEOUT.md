# P4-T4 Closeout

## Verdict

**DONE** — P4-T4 BUILD DIFF INPUT FROM RESOURCEREFS closed 2026-04-18.

## Rounds

- **Round 1** — build PASS, QA Proof PASS, QA Falsification PASS with two non-falsifying notes raised (dead `activeItem` field, missing package-first dedup test). Orchestrator requested fixes before commit.
- **Round 2** — fixup build PASS, QA Proof PASS, QA Falsification PASS with all Round 1 notes resolved.

## Fixups applied (Round 2)

- Added `func (d *diffMode) resolvePaths() []string` method reading `d.activeItem`. `enterDiffMode` now calls `m.diff.resolvePaths()` instead of `resolveDiffPaths(activeTask)` on a local variable. `SetItem` is genuinely load-bearing — the stored field is the sole source for path resolution.
- Added `TestResolveDiffPaths_PackageFirstThenPath` covering the package-first, path-second dedup ordering (complement of Round 1's path-first test).

## Commit

- SHA: `e8914fc`
- Branch: `drop/1.5`
- Message: `feat(tui): resolve diff paths from active task resourcerefs`
- Files: `internal/tui/diff_mode.go`, `internal/tui/diff_mode_test.go` (+469 / -6 lines).

## CI

- Run ID: `24613681204`
- Both jobs green: `ci (macos-latest)` in 54s, `release snapshot check` in 1m1s.

## Gates

- `mage test-pkg ./internal/tui/...` — 380 tests, 0 failures.
- `mage ci` — 1324 tests, 0 failures, `internal/tui` at 70.9% coverage (above 70% floor), build OK, formatting OK.

## Hylla Feedback

None. P4-T4 changes live on `drop/1.5`, never in Hylla's `@main` artifact during the build — subagents correctly used `Read`/`Grep`/`LSP` for uncommitted files and Hylla only for pre-existing `main`-baseline domain types. Ingest happens at drop end.

## Notes for drop-end ledger

- `ResourceRef.Tags[0]` convention extended to `"package"` in production code. P3-A and P3-B established `"path"` and `"file"`; `"package"` is tested via fixture data only (no picker writes `"package"` refs yet — that comes in a future drop).
- Option D replan holds: when Drop 1's `PlanItem.Paths` + `PlanItem.Packages` domain fields ship, migration is a mechanical swap of the read source inside `resolveDiffPaths` — partition logic stays identical.
- `d.activeItem *domain.Task` is the new field on `diffMode`. Not a `Model` field — `Model` field count unchanged (0 net).
