# Vector Search Execution Plan

Created: 2026-03-03  
Owner: Codex orchestrator (single writer for this file)

## 1) Objective

Ship production-grade hybrid search for `tillsyn` using:
1. embeddings generation via `charm.land/fantasy` (replaced to `evanmschultz/fantasy` fork commit),
2. vector retrieval via `sqlite-vec`,
3. keyword retrieval and filters with deterministic ranking, sorting, pagination, and limits,
4. final TUI support for editing/viewing all indexed metadata fields before collaborative end-to-end testing.

## 2) Locked Decisions

1. Driver/storage direction:
   - migrate SQLite adapter from `modernc` to `ncruces/go-sqlite3`,
   - use `github.com/asg017/sqlite-vec-go-bindings/ncruces` for vec-enabled SQLite runtime.
2. Embeddings provider direction:
   - use `charm.land/fantasy` API,
   - pin `go.mod` replace to `github.com/evanmschultz/fantasy v0.0.0-20260219222711-d1be5103494b`.
3. Schema evolution policy:
   - implement forward-only, idempotent schema ensures now,
   - do not implement generic legacy backfill framework in this wave.
4. Search mode policy:
   - default mode is `hybrid`,
   - fallback to keyword-only if embeddings/vector path is unavailable.
5. TUI metadata requirement:
   - fields selected for embedding/search must be viewable/editable in TUI before collaborative final verification,
   - this TUI accessibility phase lands last, immediately before collaborative test closeout.

## 3) Search Contract (V1)

### 3.1 Query Modes

1. `keyword`: lexical only.
2. `semantic`: vector only.
3. `hybrid` (default): lexical + vector combined score.

### 3.2 Filters

1. `project_id` (optional in cross-project mode).
2. `states` (existing semantics retained).
3. `include_archived` (existing semantics retained).
4. `levels` (`project|branch|phase|subphase|task|subtask`) for scope filtering.
5. `kinds` (optional).
6. `labels_any` and `labels_all` (optional).

### 3.3 Sorting

1. default: `rank_desc` (hybrid/semantic/keyword score).
2. optional:
   - `updated_at_desc`
   - `created_at_desc`
   - `title_asc`

### 3.4 Pagination and Limits

1. Add optional `limit` and `offset` parameters to search surfaces.
2. Defaults:
   - `limit=50`
   - `offset=0`
3. Guardrails:
   - `limit` max `200`
   - negative `offset` rejected.
4. Deterministic ordering required for stable pagination.

## 4) Indexed Content Plan

### 4.1 Vector + Keyword (Hybrid)

1. `task.title`
2. `task.description`
3. `task.labels` (serialized text alongside semantic metadata for hybrid relevance).
4. `task.metadata.objective`
5. `task.metadata.acceptance_criteria`
6. `task.metadata.validation_plan`
7. `task.metadata.blocked_reason`
8. `task.metadata.risk_notes`

### 4.2 Keyword + Filter-Centric

1. IDs and structural filters (`project_id`, `task_id`, `kind`, `scope`, `state`).
2. `labels_any` and `labels_all` filter semantics for taxonomy matching.

### 4.3 Phase-2 Expansion (same schema family)

1. comments (`summary`, `body_markdown`) mapped to target task/scope.
2. attention items (`summary`, `body_markdown`) mapped to scope/task.

## 5) Ranking and Dedup Plan

1. Candidate sets:
   - lexical candidate window,
   - vector candidate window.
2. Dedup key: target task/work-item id.
3. Score composition:
   - normalized lexical score,
   - normalized vector score,
   - configurable weights (`lexical_weight`, `semantic_weight`).
4. Default weights:
   - lexical `0.55`
   - semantic `0.45`
5. Tie-breakers:
   - project id, state, column, position, item id (stable deterministic fallback).

## 6) Schema Plan

1. Add forward-only idempotent ensures for:
   - `task_embeddings` table (`task_id`, `project_id`, `content_hash`, `content`, `embedding`, `updated_at`),
   - `task_embeddings` indexes (`project_id`, `updated_at`),
   - vec scalar-function-backed embedding writes/search reads (`vec_f32`, `vec_distance_cosine`) through adapter methods.
2. Keep existing migrations intact.
3. Keep lexical keyword retrieval in app-layer scoring for this wave; FTS/search-docs/queue tables are deferred until explicitly scoped.
4. If vec is unavailable:
   - schema ensure does not break repository open,
   - vector methods return a stable sentinel error,
   - keyword search remains fully operational.

## 7) TUI Accessibility Plan (Last Implementation Phase)

Before collaborative end-to-end validation, ensure TUI can view/edit all V1 indexed metadata fields:
1. `objective`
2. `acceptance_criteria`
3. `validation_plan`
4. `blocked_reason`
5. `risk_notes`

UI implementation rules:
1. reuse existing modal/form rendering primitives (DRY where reasonable),
2. follow existing styling/layout patterns,
3. avoid one-off duplicated UI logic; compose reusable field components/helpers.

## 8) Implementation Waves

## Wave A: Driver + Vec Runtime Foundation

Acceptance:
1. `ncruces` SQLite driver integrated.
2. `sqlite-vec` runtime import path integrated.
3. repository open/migrate remains stable with fallback behavior.

## Wave B: Search Schema + Storage Adapters

Acceptance:
1. forward-only idempotent `task_embeddings` schema ensures.
2. embedding upsert/read/delete paths in SQLite adapter.
3. lexical path still passes unchanged behavior tests.

## Wave C: Embedding Provider Adapter

Acceptance:
1. `fantasy` fork wired via `go.mod replace` pinned commit.
2. app-facing embedding port(s) and adapter implementation.
3. batch embedding pipeline with retries and logging.

## Wave D: Hybrid Query Engine + API Surface

Acceptance:
1. keyword/semantic/hybrid modes implemented.
2. filters, sorting, pagination, and `limit`/`offset` implemented.
3. MCP + TUI search calls support new options while preserving existing defaults.

## Wave E: TUI Metadata Accessibility (Last Before Collaborative Verification)

Acceptance:
1. all V1 indexed metadata fields are editable/viewable in TUI.
2. UI reuse/styling consistency maintained.
3. no regression in existing task/edit/thread workflows.

## Wave F: QA + Collaborative Verification

Acceptance:
1. worker lanes complete scoped package tests.
2. independent QA subagent signs off code + markdown tracker updates.
3. orchestrator runs `just check` and `just ci`.
4. collaborative testing run is executed with user and recorded evidence.

## 9) Subagent Orchestration Contract (This Wave)

1. Orchestrator:
   - single writer for this file and `PLAN.md` checkpoint updates.
2. Worker lanes:
   - scoped file locks only,
   - Context7 before edits and after failures,
   - package tests via `just test-pkg <pkg>`.
3. QA lanes:
   - review integrated code + updated markdown trackers after worker tests pass,
   - no lane marked complete without QA sign-off.

## 10) Execution Tracker

## Current Status

1. Wave A: completed
2. Wave B: completed
3. Wave C: completed
4. Wave D: completed
5. Wave E: completed
6. Wave F: in progress (automated gates complete; QA pass 1 + QA pass 2 complete; collaborative user verification pending)

## 11) Audit Findings Intake (2026-03-04)

Source audits:
1. `worklogs/VEC_AUDIT_01_APP_SEARCH.md`
2. `worklogs/VEC_AUDIT_02_SQLITE_VEC.md`
3. `worklogs/VEC_AUDIT_03_MCP_SURFACE.md`
4. `worklogs/VEC_AUDIT_04_TUI.md`
5. `worklogs/VEC_AUDIT_05_DOCS_COMPLETENESS.md`

Remediation checklist status (must be closed before marking this section complete):
1. App embeddings observability (`status: complete`):
   - remove silent failure handling for embedding index refresh/drop by adding explicit warning/error telemetry,
   - add semantic-mode fallback regression coverage.
2. SQLite vec runtime/storage hardening (`status: complete`):
   - add explicit vec capability guard path for vector SQL function usage,
   - add tests for vec-capability guard behavior.
3. MCP schema contract hardening (`status: complete`):
   - enforce pagination guardrails in tool schema (not description-only),
   - keep forwarding/tests aligned.
4. TUI coverage and request-shaping parity (`status: complete`):
   - add test coverage for `objective|acceptance_criteria|validation_plan|risk_notes` edit/view behavior,
   - align dependency-inspector search request shaping with explicit mode/sort/offset/default levels,
   - align TUI explicit default `limit` with plan default (`50`).
5. Plan/docs alignment (`status: complete`):
   - resolve section-4 labels classification drift against implemented embedding content,
   - add reproducible evidence artifact paths for `just check`/`just ci` claims,
   - make collaborative evidence destination explicit for Wave F closeout.

Completion rule for this intake:
1. No item above is marked complete until two independent QA passes report no High findings and no unresolved Medium findings for that item.
2. QA pass summaries and command evidence must be recorded in this file and `PLAN.md`.

### 11.1 QA Pass Summaries

QA pass 1 reports:
1. `worklogs/VEC_QA_PASS1_A_APP.md` -> `PASS` (no High; one Medium labels-plan drift resolved in docs).
2. `worklogs/VEC_QA_PASS1_B_SQLITE.md` -> `PASS` (no High/Medium).
3. `worklogs/VEC_QA_PASS1_C_MCP.md` -> `PASS` (no High/Medium).
4. `worklogs/VEC_QA_PASS1_D_TUI.md` -> `FAIL` (Medium default-limit drift; remediated by setting TUI default limit to `50`).
5. `worklogs/VEC_QA_PASS1_E_DOCS.md` -> `FAIL` (High/Medium docs/evidence drift; remediated in plan/docs updates).

QA pass 2 reports:
1. `worklogs/VEC_QA_PASS2_A_APP.md` -> `PASS` (no High/Medium).
2. `worklogs/VEC_QA_PASS2_B_SQLITE.md` -> `PASS` (no High/Medium).
3. `worklogs/VEC_QA_PASS2_C_MCP.md` -> `PASS` (no High/Medium).
4. `worklogs/VEC_QA_PASS2_D_TUI.md` -> `PASS` (no High/Medium).
5. `worklogs/VEC_QA_PASS2_E_DOCS.md` -> `PASS` (no High/Medium).

Intake close condition result:
1. All five remediation items satisfy the two-pass rule with no unresolved High/Medium findings in pass 2.
2. Intake section is now closed; only Wave F collaborative evidence capture remains.

### 11.2 Automated Evidence Artifacts

Evidence bundle root:
1. `.tmp/vec-wavef-evidence/20260303_175936/`
2. `.tmp/vec-wavef-evidence/20260303_180827/` (post-remediation + QA pass 2 validation bundle)

Automated gate artifacts:
1. `.tmp/vec-wavef-evidence/20260303_175936/just_check.txt`
2. `.tmp/vec-wavef-evidence/20260303_175936/just_ci.txt`
3. `.tmp/vec-wavef-evidence/20260303_180827/just_check.txt`
4. `.tmp/vec-wavef-evidence/20260303_180827/just_ci.txt`

Scoped package artifacts:
1. `.tmp/vec-wavef-evidence/20260303_175936/test_pkg_internal_app.txt`
2. `.tmp/vec-wavef-evidence/20260303_175936/test_pkg_internal_adapters_storage_sqlite.txt`
3. `.tmp/vec-wavef-evidence/20260303_175936/test_pkg_internal_adapters_server_mcpapi.txt`
4. `.tmp/vec-wavef-evidence/20260303_175936/test_pkg_internal_tui.txt`
5. `.tmp/vec-wavef-evidence/20260303_180827/test_pkg_internal_app.txt`
6. `.tmp/vec-wavef-evidence/20260303_180827/test_pkg_internal_adapters_storage_sqlite.txt`
7. `.tmp/vec-wavef-evidence/20260303_180827/test_pkg_internal_adapters_server_mcpapi.txt`
8. `.tmp/vec-wavef-evidence/20260303_180827/test_pkg_internal_tui.txt`

### 11.3 Wave F Collaborative Evidence Destination

Wave F collaborative user+agent verification evidence must be recorded in:
1. `COLLAB_TEST_2026-03-02_DOGFOOD.md` (primary vector-search collaborative worksheet entry),
2. `MCP_DOGFOODING_WORKSHEET.md` (transport-level corroboration where applicable).

## Change Log

### 2026-03-03

1. Initial execution plan created and decisions locked based on user consensus and subagent research.
2. Completed Wave A foundation:
   - migrated runtime to `github.com/ncruces/go-sqlite3` + `sqlite-vec` ncruces bindings,
   - pinned compatible sqlite runtime version and stabilized runtime config for thread features + bounded memory limits.
3. Completed Wave B storage layer:
   - added `task_embeddings` schema and indexes,
   - implemented adapter methods for embedding upsert/delete/vector search.
4. Completed Wave C embeddings integration:
   - added fantasy embedding adapter and startup wiring in `cmd/till`,
   - pinned `go.mod` replace to `github.com/evanmschultz/fantasy` fork commit.
5. Completed Wave D query/API surface:
   - implemented `keyword|semantic|hybrid` app search with semantic fallback to keyword,
   - implemented `mode/sort/limit/offset` behavior and guardrails,
   - implemented `levels/kinds/labels_any/labels_all` filters in app + MCP transport/tool schema.
6. Completed Wave E TUI metadata accessibility:
   - task form/edit flow now supports `objective`, `acceptance_criteria`, `validation_plan`, `risk_notes`,
   - task-info view renders those metadata sections,
   - TUI search/dependency flows now send explicit limits to avoid implicit truncation.
7. Wave F progress:
   - independent QA subagent passes completed after remediation,
   - scoped package tests passed for all touched packages,
   - full gates passed: `just check`, `just ci`.
8. Remaining closeout for Wave F:
   - run collaborative user+agent verification and record evidence before final close.

### 2026-03-04

1. Added remediation status tracking for audit-intake items (implemented/QA-pass state per item).
2. Resolved plan/code drift:
   - section 4 labels classification now matches implemented embedding content,
   - section 6 schema scope now matches implemented `task_embeddings` + vec-function adapter model.
3. Added explicit evidence artifact paths for `just check` / `just ci` and package-scoped test outputs.
4. Bound Wave F collaborative evidence destination to:
   - `COLLAB_TEST_2026-03-02_DOGFOOD.md`,
   - `MCP_DOGFOODING_WORKSHEET.md`.
5. Recorded QA pass 1 blockers and remediations:
   - TUI default limit aligned to `50`,
   - docs/status/evidence linkage aligned with implemented behavior.
6. Completed QA pass 2 with independent auditors and no unresolved High/Medium findings across intake items; marked remediation checklist complete.
