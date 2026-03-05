# QA2-A App Search + Embeddings Remediation Pass 2 (Independent Auditor)
Timestamp: 2026-03-04 01:14:27 UTC

## Scope
- Read: `internal/app/**`, `VECTOR_SEARCH_EXECUTION_PLAN.md`, `.tmp/vec-wavef-evidence/20260303_180827/**`, QA1 reference report.
- Write: `worklogs/VEC_QA_PASS2_A_APP.md` only.
- No code/doc/plan edits performed.

## Context7 Compliance
- Ran Context7 for external logger API confirmation before making any external-library behavior claim:
  - `mcp__context7-mcp__resolve-library-id` for `github.com/charmbracelet/log` -> `/charmbracelet/log`.
  - `mcp__context7-mcp__query-docs` -> confirmed package-level structured leveled logging usage (`Warn`, `Error`, key-value fields).

## Commands/Tests Executed and Outcomes
1. `sed -n '1,260p' internal/app/search_embeddings.go`
   - Outcome: confirmed embedding refresh/drop warning paths and embedding-content composition.
2. `nl -ba internal/app/search_embeddings.go | sed -n '45,180p'`
   - Outcome: captured line evidence for warning emissions and label inclusion in embedding content.
3. `nl -ba internal/app/service.go | sed -n '492,770p'`
   - Outcome: confirmed refresh/drop call sites after task mutations.
4. `nl -ba internal/app/service.go | sed -n '1040,1145p'`
   - Outcome: confirmed semantic dedup logic and semantic→keyword fallback path.
5. `nl -ba internal/app/service_test.go | sed -n '1578,1760p'`
   - Outcome: confirmed semantic fallback/dedup regression tests remain present.
6. `nl -ba internal/app/service_test.go | sed -n '1448,1548p'`
   - Outcome: confirmed embedding-refresh content assertions and coverage shape.
7. `nl -ba VECTOR_SEARCH_EXECUTION_PLAN.md | sed -n '68,90p'`
   - Outcome: confirmed plan text now includes `task.labels` in vector+keyword indexed content.
8. `nl -ba VECTOR_SEARCH_EXECUTION_PLAN.md | sed -n '206,232p'`
   - Outcome: confirmed remediation checklist status and labels-alignment item marked remediated in docs pending QA2.
9. `nl -ba .tmp/vec-wavef-evidence/20260303_180827/test_pkg_internal_app.txt | sed -n '1,40p'`
   - Outcome: prior evidence bundle shows app package passing.
10. `just test-pkg ./internal/app`
   - Outcome: `ok   github.com/hylla/tillsyn/internal/app (cached)`.

## Findings by Severity

### High
- None.

### Medium
- None.

### Low
1. Observability warning behavior is implemented in production code, but there is no direct log-capture assertion in `internal/app` tests.
   - Warning paths: `internal/app/search_embeddings.go:57`, `internal/app/search_embeddings.go:70`, `internal/app/search_embeddings.go:80`, `internal/app/search_embeddings.go:97`, `internal/app/search_embeddings.go:113`.
   - Mutation call sites that trigger refresh/drop flows: `internal/app/service.go:513`, `internal/app/service.go:586`, `internal/app/service.go:624`, `internal/app/service.go:648`, `internal/app/service.go:703`, `internal/app/service.go:744`.

2. Labels are included in embedding content and now align with plan text, but there is no explicit assertion that serialized labels are present in the upsert-content expectation list.
   - Inclusion in embedding content: `internal/app/search_embeddings.go:129`.
   - Plan alignment (labels in vector+keyword set): `VECTOR_SEARCH_EXECUTION_PLAN.md:74`.
   - Related test sets labels but expected-content list does not explicitly include label tokens: `internal/app/service_test.go:1480`, `internal/app/service_test.go:1497`.

## Required Check Assessment
1. Verify embedding refresh/drop observability warnings are present (no silent failures): **PASS**
   - Error/failure branches emit warnings in refresh/drop paths (`internal/app/search_embeddings.go:57-64`, `:70-76`, `:80-85`, `:97-103`, `:112-114`).

2. Verify semantic-mode fallback and duplicate-row dedup coverage remains in tests: **PASS**
   - Fallback logic: `internal/app/service.go:1118-1119`.
   - Dedup keep-max-similarity logic: `internal/app/service.go:1109-1111`.
   - Regression coverage:
     - Semantic fallback test: `internal/app/service_test.go:1591-1639`.
     - Hybrid fallback test: `internal/app/service_test.go:1699-1747`.
     - Duplicate-row dedup test: `internal/app/service_test.go:1641-1697`.

3. Verify labels-in-embedding behavior now matches plan text: **PASS**
   - Plan includes labels in embedding/indexed content: `VECTOR_SEARCH_EXECUTION_PLAN.md:74`.
   - Implementation includes labels in `buildTaskEmbeddingContent`: `internal/app/search_embeddings.go:129-131`.
   - Remediation tracker notes docs alignment item: `VECTOR_SEARCH_EXECUTION_PLAN.md:225-227`.

## Verdict
- **QA2-A: PASS**
- Rationale: no High findings, no unresolved Medium findings, required checks satisfied, and package-scoped app tests pass.

## Unresolved Risks
1. Lack of direct log-capture assertions for warning emission leaves observability regressions detectable mainly via manual/runtime inspection.
2. Label-in-embedding alignment is code+plan consistent, but explicit assertion for serialized labels in embedding-content tests is absent.

## Exact Next Step
1. Add one focused app test for log-emission capture on embedding refresh/drop error paths and one explicit assertion that upsert embedding content contains serialized labels (without changing behavior).
