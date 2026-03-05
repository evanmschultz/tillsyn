# QA1-B SQLite Vec Remediation Audit

Timestamp (UTC): 2026-03-04T01:03:57Z

## Scope

- Read-only audit scope:
  - `internal/adapters/storage/sqlite/repo.go`
  - `internal/adapters/storage/sqlite/repo_test.go`
  - `go.mod` / `go.sum`
  - `VECTOR_SEARCH_EXECUTION_PLAN.md` (sections 2 and 6, plus active remediation intake)
  - `.tmp/vec-wavef-evidence/20260303_175936/*`
- Write scope:
  - `worklogs/VEC_QA_PASS1_B_SQLITE.md` only

## Context7 Compliance

Context7 was consulted before behavior claims:
- `/ncruces/go-sqlite3`: runtime config defaults (memory limits + core features baseline).
- `/asg017/sqlite-vec`: runtime probe function `vec_version()` and vec scalar function references.

No command failures occurred during this pass, so no failure-triggered re-consult was required.

## Commands And Outcomes

1. `rg/nl` inspection commands across scoped files: pass.
2. `just test-pkg ./internal/adapters/storage/sqlite`: pass.
   - Output: `ok github.com/hylla/tillsyn/internal/adapters/storage/sqlite (cached)`
3. Evidence artifact read:
   - `.tmp/vec-wavef-evidence/20260303_175936/test_pkg_internal_adapters_storage_sqlite.txt` confirms package pass.

## Findings By Severity

### High

- None.

### Medium

- None.

### Low

1. `probeVecCapability` unavailable branch is covered by direct logic review, but not by a deterministic test that simulates a real `vec_version()` missing-function DB response.
   - Evidence: probe path and non-fatal swallow in migrate at `internal/adapters/storage/sqlite/repo.go:353-358`, probe implementation at `internal/adapters/storage/sqlite/repo.go:2596-2610`.
   - Existing tests validate method guard behavior, not DB probe error-shape behavior: `internal/adapters/storage/sqlite/repo_test.go:196-230`.

## Required Checkpoints

1. Verify sqlite-vec capability probing exists and is non-fatal during open/migrate: `PASS`.
- Probe exists: `internal/adapters/storage/sqlite/repo.go:2596-2610` (`SELECT vec_version()`).
- Non-fatal open/migrate behavior exists: `internal/adapters/storage/sqlite/repo.go:353-356` (sentinel recognized and migrate returns nil).

2. Verify vector methods guard when vec unavailable and return stable sentinel: `PASS`.
- Sentinel declaration: `internal/adapters/storage/sqlite/repo.go:30-31`.
- Guard in upsert: `internal/adapters/storage/sqlite/repo.go:1380-1382`.
- Guard in search: `internal/adapters/storage/sqlite/repo.go:1434-1436`.
- Stable guard helper: `internal/adapters/storage/sqlite/repo.go:2613-2619`.

3. Verify adapter tests cover vec guard behavior: `PASS`.
- Guard-path test: `internal/adapters/storage/sqlite/repo_test.go:196-230`.
- Existing vec roundtrip test now capability-aware (`skip` when unavailable): `internal/adapters/storage/sqlite/repo_test.go:110-122`.

4. Verify schema/behavior supports current plan requirements for this wave: `PASS`.
- Plan requires ncruces + sqlite-vec runtime direction: `VECTOR_SEARCH_EXECUTION_PLAN.md:16-19`; reflected in pins `go.mod:14` and `go.mod:21` (with checksums in `go.sum:15-16` and `go.sum:134-135`).
- Plan section 6 fallback requirement says vec-unavailable must not break open: `VECTOR_SEARCH_EXECUTION_PLAN.md:114-116`; implemented via non-fatal probe handling at `internal/adapters/storage/sqlite/repo.go:353-356`.
- Current vec storage ensure remains present (`task_embeddings` + indexes): `internal/adapters/storage/sqlite/repo.go:173-184`.

## Completeness Checklist vs Plan

- Plan §2 driver/runtime direction in code pins: `PASS`.
- Plan §6 vec-unavailable non-fatal open behavior: `PASS`.
- Plan intake remediation item (explicit vec guard + tests) `VECTOR_SEARCH_EXECUTION_PLAN.md:213-215`: `PASS`.

## Residual Risks

1. Real missing-function error-shape compatibility for `isMissingFunctionErr` (`"no such function"`) remains inferred from SQLite behavior and code review, not explicitly simulated in package tests.
2. Test command result was cached on this run; prior evidence file also shows pass but does not add new runtime variance.

## Exact Next Step

1. Add one deterministic unit/integration test seam to simulate `vec_version()` missing-function probe failure and assert open/migrate remains non-fatal, then rerun `just test-pkg ./internal/adapters/storage/sqlite`.

## Verdict

`PASS` for QA1-B (no High/Medium findings; remediation objective verified with one Low residual risk).
