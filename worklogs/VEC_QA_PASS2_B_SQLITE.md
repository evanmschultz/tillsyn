# QA2-B: SQLite + ncruces + sqlite-vec Remediation Pass 2
Timestamp (UTC): 2026-03-04T01:14:13Z

## Context7 Compliance
- Consulted Context7 before behavior claims:
- `/ncruces/go-sqlite3` for extension/runtime behavior context (auto-extension/load behavior and runtime setup semantics).
- `/asg017/sqlite-vec` for vec capability/function behavior (`vec_version()`, vec-backed SQL function usage expectations).
- Context7 was available; no fallback source was required.

## Commands/Tests Executed And Outcomes
- `rg --files internal/adapters/storage/sqlite` -> scoped file inventory complete.
- `rg -n "vec|sqlite-vec|ncruces|capab|probe|guard|sentinel|vec_version|vec_f32|vec_distance_cosine" internal/adapters/storage/sqlite/repo.go internal/adapters/storage/sqlite/repo_test.go go.mod` -> located all relevant implementation/test paths.
- `nl -ba go.mod | sed -n '1,220p'` -> confirmed dependency pins (`github.com/asg017/sqlite-vec-go-bindings`, `github.com/ncruces/go-sqlite3`).
- `nl -ba internal/adapters/storage/sqlite/repo.go | sed -n '1,260p'` + `...260,620p` + `...1340,1495p` + `...2570,2645p` -> verified open/migrate/probe/guard/vector-method behavior.
- `nl -ba internal/adapters/storage/sqlite/repo_test.go | sed -n '1,90p'` + `...90,290p` -> verified vec round-trip and vec-unavailable guard tests.
- `nl -ba VECTOR_SEARCH_EXECUTION_PLAN.md | sed -n '96,126p'` -> reviewed section-6 schema/fallback wording.
- `nl -ba worklogs/VEC_QA_PASS1_B_SQLITE.md | sed -n '1,260p'` and `nl -ba worklogs/VEC_AUDIT_02_SQLITE_VEC.md | sed -n '1,260p'` -> compared prior QA posture.
- `sed -n '1,240p' .tmp/vec-wavef-evidence/20260303_180827/test_pkg_internal_adapters_storage_sqlite.txt` -> historical package evidence confirms pass.
- `just test-pkg ./internal/adapters/storage/sqlite` -> `ok github.com/hylla/tillsyn/internal/adapters/storage/sqlite (cached)`.

## Findings By Severity

### High
- None.

### Medium
- None.

### Low
1. Probe missing-function branch is validated by code path and guard tests, but there is no deterministic test seam that simulates a real `vec_version()` missing-function DB response at probe time.
- Evidence:
- `internal/adapters/storage/sqlite/repo.go:2599`
- `internal/adapters/storage/sqlite/repo.go:2600`
- `internal/adapters/storage/sqlite/repo.go:353`
- `internal/adapters/storage/sqlite/repo.go:355`
- `internal/adapters/storage/sqlite/repo_test.go:196`
- `internal/adapters/storage/sqlite/repo_test.go:230`
- Impact: low confidence gap around exact DB error-text shape matching in `isMissingFunctionErr`.

## Required Check Results

1. Verify vec capability probe/open behavior and guard sentinel paths: **PASS**
- Probe on open/migrate path: `internal/adapters/storage/sqlite/repo.go:353`, `internal/adapters/storage/sqlite/repo.go:2596`.
- Sentinel handling and non-fatal open when vec unavailable: `internal/adapters/storage/sqlite/repo.go:354`, `internal/adapters/storage/sqlite/repo.go:355`, `internal/adapters/storage/sqlite/repo.go:2602`.
- Guard sentinel path for vector methods: `internal/adapters/storage/sqlite/repo.go:1380`, `internal/adapters/storage/sqlite/repo.go:1434`, `internal/adapters/storage/sqlite/repo.go:2613`.

2. Verify vec guard tests exist and still pass: **PASS**
- Guard tests exist: `internal/adapters/storage/sqlite/repo_test.go:196`.
- Capability-aware round-trip test exists: `internal/adapters/storage/sqlite/repo_test.go:110`, `internal/adapters/storage/sqlite/repo_test.go:120`.
- Scoped test command pass: `just test-pkg ./internal/adapters/storage/sqlite` -> `ok ... (cached)`.

3. Verify section-6 docs align with implemented storage shape: **PASS**
- Section-6 now specifies `task_embeddings` + indexes + vec scalar-function-backed adapter calls:
  - `VECTOR_SEARCH_EXECUTION_PLAN.md:109`
  - `VECTOR_SEARCH_EXECUTION_PLAN.md:112`
- Implementation matches:
  - `internal/adapters/storage/sqlite/repo.go:173`
  - `internal/adapters/storage/sqlite/repo.go:183`
  - `internal/adapters/storage/sqlite/repo.go:1389`
  - `internal/adapters/storage/sqlite/repo.go:1450`
- Section-6 vec-unavailable fallback language aligns with code:
  - `VECTOR_SEARCH_EXECUTION_PLAN.md:115`
  - `VECTOR_SEARCH_EXECUTION_PLAN.md:117`
  - `internal/adapters/storage/sqlite/repo.go:355`
  - `internal/adapters/storage/sqlite/repo.go:2618`

## Unresolved Risks
- `isMissingFunctionErr` depends on message-text matching (`"no such function"`), which may vary by driver/runtime error wording.
- Test pass was cached in this run; no fresh non-cached runtime variation was exercised.

## Exact Next Step
1. Add one deterministic test seam for probe-time missing-function behavior (`SELECT vec_version()` failure shape) and assert migrate/open remains non-fatal; rerun `just test-pkg ./internal/adapters/storage/sqlite`.

## Verdict
**PASS** for QA2-B.
