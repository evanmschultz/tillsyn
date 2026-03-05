# QA1-A App Search + Embeddings Remediation Pass 1
Timestamp: 2026-03-04 01:03:50 UTC

## Commands/Tests Executed
1. `mcp__context7-mcp__resolve-library-id` for `github.com/charmbracelet/log`
   - Outcome: resolved `/charmbracelet/log`.
2. `mcp__context7-mcp__query-docs` for package-level `log.Warn`/`log.Error` structured-field usage
   - Outcome: confirmed supported package-level leveled logging with key-value fields.
3. `nl -ba VECTOR_SEARCH_EXECUTION_PLAN.md | sed -n '60,110p'`
   - Outcome: confirmed section-4 labels classification and section-5 dedup expectations.
4. `rg --files .tmp/vec-wavef-evidence/20260303_175936`
   - Outcome: verified evidence bundle presence.
5. `nl -ba .tmp/vec-wavef-evidence/20260303_175936/test_pkg_internal_app.txt | sed -n '1,120p'`
   - Outcome: prior wave evidence shows `ok   github.com/hylla/tillsyn/internal/app (cached)`.
6. `just test-pkg ./internal/app`
   - Outcome: `ok   github.com/hylla/tillsyn/internal/app (cached)`.

## Findings by Severity

### High
- None.

### Medium
1. Section-4 labels classification still drifts between code and plan.
   - Plan classifies labels as keyword-centric: `VECTOR_SEARCH_EXECUTION_PLAN.md:80-83`.
   - Embedding content still includes labels: `internal/app/search_embeddings.go:129-131`.
   - Impact: semantic scoring can continue to be influenced by taxonomy labels unless plan/docs are updated (or code behavior is changed in a future lane).

### Low
1. Level filter validation is warn-and-fail-closed rather than hard validation error.
   - Supported set and warning: `internal/app/service.go:51-59`, `internal/app/service.go:1008-1010`, `internal/app/service.go:1261-1271`.
   - Filtering remains deterministic/fail-closed by exact normalized membership check: `internal/app/service.go:1275-1283`.
   - Assessment: acceptable and deterministic for this wave; caller diagnostics rely on logs.

2. Remediation for embedding observability uses warning logs but there is no direct assertion of log emission in tests.
   - Warning log paths added: `internal/app/search_embeddings.go:57-64`, `internal/app/search_embeddings.go:70-76`, `internal/app/search_embeddings.go:80-85`, `internal/app/search_embeddings.go:97-103`, `internal/app/search_embeddings.go:112-114`.
   - Assessment: behavior is implemented; explicit log-capture tests are optional hardening.

## Required Checkpoint Assessment
1. Silent embedding refresh/drop failures now emit explicit warnings: **PASS**
   - Evidence: warning calls in `refreshTaskEmbedding` and `dropTaskEmbedding` at `internal/app/search_embeddings.go:57-64`, `:70-76`, `:80-85`, `:97-103`, `:112-114`.
2. Semantic-mode fallback to keyword has regression test coverage: **PASS**
   - Evidence: `TestSearchTaskMatchesSemanticFallsBackToKeyword` at `internal/app/service_test.go:1591-1639`.
3. Semantic dedup keeps strongest score for duplicate task_id rows: **PASS**
   - Evidence: max-similarity keep logic `internal/app/service.go:1108-1112`.
   - Regression coverage: `TestSearchTaskMatchesSemanticModeDuplicateRowsKeepMaxSimilarity` at `internal/app/service_test.go:1641-1697`.
4. Level-filter validation/warning behavior is acceptable and deterministic: **PASS (warn-and-fail-closed)**
   - Evidence: `internal/app/service.go:1008-1010`, `internal/app/service.go:1275-1283`.
5. Section-4 labels classification drift status and docs-adjustment recommendation: **PASS (status confirmed, docs adjustment needed)**
   - Evidence: plan `VECTOR_SEARCH_EXECUTION_PLAN.md:80-83` vs code `internal/app/search_embeddings.go:129-131`.

## Verdict
- **QA1-A: PASS** for remediation completeness of app-layer hardening goals.
- One non-blocking documentation/product-alignment follow-up remains for labels classification drift.

## Unresolved Risks
1. Labels-in-embeddings behavior remains inconsistent with section-4 classification text.
2. Log-emission presence is not directly asserted by tests (implementation exists and package tests pass).

## Exact Next Step
1. Decide and execute one alignment path in the next lane:
   - Option A: update plan/docs to explicitly allow labels in embedding content, or
   - Option B: remove labels from embedding content and update/regression-test search ranking expectations.
