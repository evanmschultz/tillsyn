# VEC Audit 01 - App Search + Embeddings
Timestamp: 2026-03-04 00:34:48 UTC

## Commands Run and Outcomes
1. `nl -ba VECTOR_SEARCH_EXECUTION_PLAN.md | sed -n '1,260p'`
   - Outcome: reviewed plan sections 3, 4, 5 and execution notes.
2. `nl -ba internal/app/service.go | sed -n '940,1415p'`
   - Outcome: reviewed search modes/filters/ranking/pagination implementation.
3. `nl -ba internal/app/search_embeddings.go | sed -n '1,320p'`
   - Outcome: reviewed embedding content construction and embedding-index update behavior.
4. `nl -ba internal/app/ports.go | sed -n '1,280p'`
   - Outcome: reviewed app-layer repository/search-related interfaces.
5. `rg -n ... internal/app/service_test.go`
   - Outcome: located relevant test coverage for search behavior and fallbacks.
6. `just test-pkg ./internal/app`
   - Outcome: `ok   github.com/hylla/tillsyn/internal/app (cached)`.
7. Context7
   - `resolve-library-id("sqlite-vec")` -> `/asg017/sqlite-vec`.
   - `query-docs("/asg017/sqlite-vec", hybrid/FTS+vector composition and fallback)` -> confirms hybrid behavior is query-composition level (FTS + vector combination in SQL examples), with fallback policy implemented by caller/query logic.

## Findings by Severity

### High
1. Silent embedding/index update failures can leave semantic/hybrid quality degraded without observability.
   - Evidence:
     - `internal/app/search_embeddings.go:59-62` returns on embed failure/empty vectors with no surfaced signal.
     - `internal/app/search_embeddings.go:71` ignores upsert errors.
     - `internal/app/search_embeddings.go:79` ignores delete errors.
   - Risk:
     - Index drift can accumulate and semantic/hybrid behavior can regress silently.
     - Execution plan expected retry/logging characteristics in Wave C (`VECTOR_SEARCH_EXECUTION_PLAN.md:148-154`).

### Medium
1. Plan mismatch: labels are embedded into vector content though plan classifies labels as keyword-centric.
   - Evidence:
     - `internal/app/search_embeddings.go:94-96` includes joined `task.Labels` in embedding content.
     - Plan marks labels as keyword-centric (`VECTOR_SEARCH_EXECUTION_PLAN.md:80-84`).
   - Risk:
     - Label taxonomy can dominate semantic similarity and reduce precision/relevance of vector ranking.

2. Missing regression test for semantic-mode fallback-to-keyword path.
   - Evidence:
     - Fallback logic applies to both semantic and hybrid: `internal/app/service.go:1100-1102`.
     - Tests cover semantic success path: `internal/app/service_test.go:1531-1587`.
     - Tests cover hybrid fallback only: `internal/app/service_test.go:1591-1637`.
   - Risk:
     - Semantic fallback behavior could regress without failing tests.

### Low
1. `levels` filter values are not explicitly validated against the allowed level set.
   - Evidence:
     - `internal/app/service.go:1230-1248` lowercases/matches directly; no enum validation.
   - Risk:
     - Invalid values fail closed to empty results, which is safe but can be hard to diagnose.

2. Dedup behavior is implicit; duplicate semantic rows are not explicitly normalized to best-score semantics.
   - Evidence:
     - Semantic map assignment overwrites by `task_id`: `internal/app/service.go:1089-1095`.
   - Risk:
     - If adapter/query ever returns duplicate `task_id` rows with unstable ordering, rank reproducibility may degrade.

## Completeness Checklist vs Plan (Sections 3, 4, 5)

### Section 3 - Search Contract (V1)
- 3.1 Query modes (`keyword|semantic|hybrid`, default hybrid): **PASS**
  - `internal/app/service.go:972-973`, `internal/app/service.go:1185-1194`
- 3.2 Filters (`project_id`, `states`, `include_archived`, `levels`, `kinds`, `labels_any`, `labels_all`): **PASS**
  - `internal/app/service.go:993-996`, `internal/app/service.go:1062-1064`, `internal/app/service.go:1243-1285`
- 3.3 Sorting (`rank_desc` default + optional sorts): **PASS**
  - `internal/app/service.go:976-979`, `internal/app/service.go:1146-1172`, `internal/app/service.go:1199-1210`
- 3.4 Pagination/limits defaults and guardrails, deterministic ordering: **PASS**
  - `internal/app/service.go:1213-1227`, `internal/app/service.go:1146-1173`, `internal/app/service.go:1287-1302`, `internal/app/service.go:1175-1182`

### Section 4 - Indexed Content Plan
- 4.1 Vector+keyword fields include title/description + required metadata fields: **PASS**
  - `internal/app/search_embeddings.go:92-101`
  - `internal/app/service.go:1344-1353`
- 4.2 Labels keyword-centric classification: **FAIL (plan drift)**
  - Labels currently included in embedding content: `internal/app/search_embeddings.go:94-96`
  - Plan intent: `VECTOR_SEARCH_EXECUTION_PLAN.md:80-84`

### Section 5 - Ranking and Dedup Plan
- 5.1 Candidate windows (lexical + vector): **PASS**
  - Lexical candidates from filtered tasks: `internal/app/service.go:1040-1072`
  - Semantic candidate window via `max(limit*4, searchSemanticK)`: `internal/app/service.go:1082-1087`
- 5.2 Dedup key task id: **PASS (implicit)**
  - Task-level matching/score maps keyed by task id: `internal/app/service.go:1027`, `internal/app/service.go:1075`, `internal/app/service.go:1129`
- 5.3 Score composition (normalized lexical + semantic with weights): **PASS**
  - `internal/app/service.go:1133-1141`
- 5.4 Default weights (0.55/0.45): **PASS**
  - `internal/app/service.go:45-47`, `internal/app/search_embeddings.go:111-127`
- 5.5 Stable tie-breakers (project/state/column/position/id): **PASS**
  - `internal/app/service.go:1287-1302`

## Residual Risks
1. Embedding pipeline has no surfaced failure signal at app layer, so search quality regressions may present as relevance drift rather than explicit errors.
2. Label inclusion in embedding text may produce semantic ranking noise inconsistent with current plan intent.
3. Semantic fallback behavior is implemented but not fully regression-protected by dedicated tests.

## Final Verdict
App-layer vector/hybrid search is **functionally near-complete** for plan sections 3 and 5, and mostly complete for section 4. However, **not fully plan-conformant** due to section 4.2 drift (labels in embedding payload) and there is one **high QA risk** (silent embedding/index failures) plus medium test-completeness risk (semantic fallback regression coverage gap).
