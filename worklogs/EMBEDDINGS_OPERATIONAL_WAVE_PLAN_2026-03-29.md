# Embeddings Operational Wave Plan

Created: 2026-03-29
Status: in progress
Lane: `agent/embeddings`

## Objective

Finish the embeddings/search wave so semantic search is operationally usable, observable, and maintainable instead of remaining provider-wired but lifecycle-incomplete.

This wave is only complete when:
- normal work-item mutations enqueue indexing work without blocking on provider calls,
- embedding readiness is durable across retries and restarts,
- search degrades cleanly and explains why,
- operators can inspect pending, running, ready, failed, and stale states from CLI, MCP, and TUI,
- explicit reindex/backfill flows exist for existing data,
- runtime logs emit a stable lifecycle contract,
- touched packages remain above the repo coverage floor and `just ci` passes.

## Locked Scope

Index now:
- work items only, using one shared indexed subject type for tasks, branches, phases, and subtasks

Do not index in this wave:
- auth requests
- auth sessions
- capability leases
- handoffs
- attention items
- change-event coordination/audit records

Later candidates:
1. comments and task threads
2. project-level descriptive docs and standards
3. selected project notes and imported markdown resources

## Current Gaps

1. Work-item mutations synchronously call embedding refresh logic after writes.
2. Only final vector rows are stored; there is no durable indexing lifecycle state.
3. There is no background worker, no startup resume, and no stuck-job recovery.
4. There is no retry ledger or operator-visible failure inventory.
5. Existing tasks cannot be explicitly reindexed or backfilled.
6. Search silently falls back to keyword and does not expose semantic readiness or fallback reasons.
7. CLI, MCP, and TUI do not expose embedding health or indexing progress.

## Architecture Direction

Keep the separation of concerns strict:

1. Domain/app mutation paths own content-change detection and enqueue intent only.
2. Storage owns durable lifecycle state plus persisted vector documents.
3. A background worker owns provider calls, retries, progress, and recovery.
4. Search owns lexical/semantic composition and fallback reporting.
5. CLI, MCP, and TUI own operator visibility; they do not own indexing decisions.

The wave must not embed raw vectors on project/task/work-item rows or domain structs.

Instead:
- keep one lifecycle/state table keyed by `subject_type + subject_id`
- keep one vector/document table keyed by the same subject identity
- treat the lifecycle table as the source of truth for indexing state

## Indexed Content Contract

### Embedded fields now

For each work item, embed:
- `title`
- `description`
- `labels`
- `metadata.objective`
- `metadata.acceptance_criteria`
- `metadata.validation_plan`
- `metadata.blocked_reason`
- `metadata.risk_notes`

### Structured filters that remain filters

Do not replace these with embeddings:
- project scope
- archived inclusion
- lifecycle state
- level/scope
- kind
- `labels_any`
- `labels_all`

Notes:
- labels remain both structured filters and indexed text in this wave
- auth or coordination entities remain explicit operational inventory only

## Lifecycle State Model

Add a durable per-subject lifecycle record with at least:
- `subject_type`
- `subject_id`
- `project_id`
- `content_hash_desired`
- `content_hash_indexed`
- `model_provider`
- `model_name`
- `model_dimensions`
- `model_signature`
- `status` (`pending|running|ready|failed|stale`)
- `attempt_count`
- `retry_count`
- `max_attempts`
- `next_attempt_at`
- `last_enqueued_at`
- `last_started_at`
- `last_heartbeat_at`
- `last_succeeded_at`
- `last_failed_at`
- `last_error_code`
- `last_error_message`
- `last_error_summary`
- `stale_reason`
- `claimed_by`
- `claim_expires_at`
- `created_at`
- `updated_at`

Keep the vector/document row separate with:
- `subject_type`
- `subject_id`
- `project_id`
- `content`
- `content_hash`
- `embedding`
- `updated_at`

State rules:
- `pending`: work is queued or eligible to run
- `running`: worker has an active claim and heartbeat
- `ready`: indexed row matches desired hash and current model signature
- `failed`: latest attempt exhausted retry budget or hit a non-retryable error
- `stale`: desired hash or model signature no longer matches the ready row

Idempotence rules:
- duplicate enqueue of same desired hash/model signature must not create duplicate work
- retries must be safe after process crash
- worker restart must be able to reclaim expired `running` rows
- explicit reindex must be safe to rerun

## Worker Model

Runtime worker behavior:

1. Startup
- resume any expired `running` rows into retryable work
- mark rows stale when model signature has changed
- optionally enqueue an initial project-wide catch-up scan when embeddings are enabled and state is missing

2. Normal loop
- claim eligible `pending|stale|retryable failed` rows transactionally
- transition row to `running`
- emit `start`
- fetch canonical content from current source row
- skip and mark success-delete when source is gone or content is empty
- request embeddings from provider
- upsert vector row
- transition row to `ready`
- emit `success`

3. Failure path
- classify retryable vs non-retryable failures
- increment counters and set exponential backoff
- emit `fail`
- when retryable, schedule `next_attempt_at` and emit `retry`
- when not retryable or attempts exhausted, leave row `failed`

4. Stuck-job recovery
- any `running` row whose heartbeat/claim has expired is considered abandoned
- startup recovery and periodic sweeps must return it to retryable state

## Search Contract

Search must report both requested and effective behavior.

Top-level search response contract should include:
- `requested_mode`
- `effective_mode`
- `fallback_reason`
- `semantic_available`
- `semantic_candidate_count`
- `indexed_ready_count`
- `indexed_pending_count`
- `indexed_failed_count`
- `indexed_stale_count`

Per-match contract should include:
- current embedding `status`
- `indexed_at`
- `stale_reason`
- `last_error_summary`
- whether semantic score contributed to ranking

Fallback rules:
- semantic/hybrid never hard-fail only because embeddings are unavailable
- semantic/hybrid may fall back to keyword with an explicit reason
- results for pending/stale/failed subjects still remain eligible through lexical search

## Logging/Event Contract

Use `github.com/charmbracelet/log` structured events with one stable contract.

Required event names:
- `enqueue`
- `start`
- `success`
- `fail`
- `retry`
- `skip`
- `stale`

Required fields on every event where applicable:
- `subject_type`
- `subject_id`
- `project_id`
- `status`
- `reason`
- `content_hash`
- `model_signature`
- `attempt`
- `retry_count`
- `worker_id`
- `duration_ms`
- `next_attempt_at`
- `err`

Event usage:
- `enqueue`: work became pending because of mutation, startup catch-up, or explicit reindex
- `start`: worker claimed a row
- `success`: vector row now matches desired content/model
- `fail`: attempt failed
- `retry`: failure scheduled for another attempt
- `skip`: no-op case such as missing source, empty content, or already-current state
- `stale`: current indexed row invalidated by content hash or model signature change

## Operator Surfaces

### CLI

Add dedicated embeddings commands:
- `till embeddings status`
- `till embeddings reindex`

Minimum CLI behaviors:
- project-scoped and cross-project summary counts
- list/filter by `pending|running|ready|failed|stale`
- show last success/failure timestamps and summaries
- explicit reindex for all or scoped subjects
- optional wait/progress mode for reindex

### MCP

Add dedicated tools:
- `till.get_embeddings_status`
- `till.reindex_embeddings`

Minimum MCP behaviors:
- machine-readable summary counts
- per-subject status rows
- search results include embedding execution metadata and per-match status
- reindex can return immediate accepted state or streamed/progress-like wait results

### TUI

Add operator-visible surfaces without overloading search-only UI:
- project-level embeddings summary in the coordination/notices area
- task/work-item detail indicator for ready/pending/failed/stale
- visible degraded-search status when semantic fallback happened
- scoped reindex action

Attention surface:
- failed or long-stale indexing should be raisable as attention items for human visibility
- non-actionable transient pending/running state should not create noisy user-action alerts

## Done Criteria

This wave is done only if all of the following are true:

1. Work-item mutations no longer block on provider embedding calls.
2. Lifecycle state is persisted and survives restarts.
3. Startup recovery resumes abandoned work.
4. Reindex/backfill exists for existing rows.
5. Search reports fallback behavior and per-result indexing state.
6. CLI, MCP, and TUI all expose embeddings health.
7. Lifecycle logs emit the locked event contract.
8. Touched packages have meaningful regression tests.
9. No dead/orphaned lifecycle code paths remain unreferenced.
10. `just ci` passes at the end of the lane.

## Builder And QA Split

### Builder A: core lifecycle

Owned files/modules:
- `internal/app/**` for enqueue/state/search contracts
- `internal/adapters/storage/sqlite/**` for schema/state persistence
- `internal/adapters/embeddings/fantasy/**` only if needed for classification or plumbing
- `cmd/till/main.go` runtime worker startup/plumbing

QA for Builder A:
1. QA-A1 reviews lifecycle state model, retry/recovery behavior, and schema correctness
2. QA-A2 reviews search fallback/status semantics and regression coverage

### Builder B: operator surfaces

Owned files/modules:
- `cmd/till/**` embeddings CLI surface
- `internal/adapters/server/common/**` search/status transport contracts
- `internal/adapters/server/mcpapi/**` embeddings MCP tools and search output
- `internal/tui/**` project/task embeddings visibility
- `README.md` and `config.example.toml` docs/operator guidance

QA for Builder B:
1. QA-B1 reviews CLI and MCP contract completeness
2. QA-B2 reviews TUI/operator visibility and docs alignment

### Final follow-up QA

After integration:
1. QA-F1 performs end-to-end architecture/regression review across all touched files
2. QA-F2 performs docs/test/operational completeness review to catch missed gaps

## Validation Plan

During implementation:
- use Context7 before code changes and again after any failed test/runtime error
- use `just test-pkg` on touched packages during iteration
- run `just check` after meaningful integrated increments

Before handoff:
- run package-focused tests for every touched package
- run `just ci`

Expected touched-package loop:
- `just test-pkg ./internal/app`
- `just test-pkg ./internal/adapters/storage/sqlite`
- `just test-pkg ./internal/adapters/server/common`
- `just test-pkg ./internal/adapters/server/mcpapi`
- `just test-pkg ./internal/tui`
- `just test-pkg ./cmd/till`

## Handoff Requirement

Do not update `PLAN.md` from this lane.

Record completion evidence in the final handoff note instead:
- implementation summary
- files changed
- commands run
- remaining gaps
- recommended follow-up
