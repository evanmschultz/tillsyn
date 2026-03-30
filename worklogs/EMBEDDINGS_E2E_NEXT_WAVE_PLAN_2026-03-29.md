# Embeddings Next Wave Plan

Created: 2026-03-29
Status: in progress
Lane: `agent/embeddings`

## Objective

Extend the operational embeddings wave from work-item-only indexing into:
- thread/comment context indexing,
- project-document indexing,
- higher-fidelity integration and end-to-end verification,
- a local deterministic runtime path that can drive a real SQLite database without external secrets.

This wave is only complete when:
- comment and thread content materially affect semantic task search where that content maps to a work-item target,
- project descriptive content is indexed as its own subject type rather than being shoved into project rows,
- lifecycle state and operator surfaces still stay aligned across CLI, MCP, and TUI,
- real-DB tests cover the new subject families,
- TUI golden and tea-driven coverage prove the user-visible operator/search states,
- a manual collaborative run can be performed locally against a real SQLite database.

## Scope

### New indexed subject families

1. `thread_context`
- one lifecycle/document row per comment target
- content is rebuilt from the canonical target plus all comments on that target
- eligible targets in this wave:
  - `project`
  - `branch`
  - `phase`
  - `task`
  - `subtask`
  - `decision`
  - `note`

2. `project_document`
- one lifecycle/document row per project
- content is rebuilt from the canonical project description and standards surfaces

### Still excluded

- auth requests/sessions
- capability leases
- handoffs
- attention items
- change-event audit/control-plane records

## Architecture Direction

The first wave made lifecycle state generic but left the vector store task-shaped.
This wave removes that mismatch.

### Document model

Replace task-only vector rows with generic embedding documents carrying:
- `subject_type`
- `subject_id`
- `project_id`
- `search_target_type`
- `search_target_id`
- `content`
- `content_hash`
- `embedding`
- `updated_at`

Rules:
- `work_item` documents target the same work item
- `thread_context` documents target the underlying thread target
- `project_document` documents target the project itself

### Search behavior

Task search should aggregate semantic hits from:
- `work_item` documents targeting work items
- `thread_context` documents whose `search_target_type` resolves to a work item

Project-document rows are indexed in this wave for lifecycle and future search expansion, but do not need to widen task search semantics beyond the current task-oriented contract.

### Mutation ownership

- task create/update/move/archive/restore/delete:
  - enqueue `work_item`
  - enqueue dependent `thread_context` for that target
- project update/archive/restore:
  - enqueue `project_document`
  - enqueue dependent `thread_context` for the project target
- comment create:
  - enqueue `thread_context` for the comment target
- explicit reindex:
  - backfill all three subject families for the requested scope

## Indexed Content Contracts

### Work item

Continue embedding:
- `title`
- `description`
- `labels`
- `metadata.objective`
- `metadata.acceptance_criteria`
- `metadata.validation_plan`
- `metadata.blocked_reason`
- `metadata.risk_notes`

### Thread context

Embed:
- target display/title context
- target description/details context where present
- each comment `summary`
- each comment `body_markdown`
- lightweight actor attribution per comment when useful for context

Do not embed:
- auth/session metadata
- lease/coordination data

### Project document

Embed:
- `project.name`
- `project.description`
- `project.metadata.tags`
- `project.metadata.standards_markdown`

Keep structured:
- project id/scope
- archived state
- project kind

## Lifecycle / Worker Requirements

The existing lifecycle contract stays authoritative:
- `pending`
- `running`
- `ready`
- `failed`
- `stale`

Additional requirements for this wave:
- worker must rebuild all three subject families idempotently
- recovery and retry behavior must apply equally to all subject types
- startup reconciliation and steady-state recovery must continue emitting per-subject `retry` and `stale` events
- operator inventory must show mixed subject families without ambiguity

## Testing Strategy

### Real-DB integration

Add or extend tests that use actual SQLite files or in-memory SQLite:
- multi-subject lifecycle storage
- thread-context backfill from persisted comments
- project-document backfill from persisted projects
- semantic search aggregation from work-item + thread-context documents
- CLI status/reindex output against a real repository + service stack
- MCP status/reindex/search integration where appropriate

### TUI coverage

Add or extend:
- tea-driven tests for embeddings status mode with mixed subject rows
- tea-driven search-state tests showing fallback and semantic-ready behavior
- golden output for at least one embeddings operator surface if visible layout changes land

### End-to-end runtime

Add a deterministic local embeddings provider so:
- the background worker can run against a real SQLite DB in tests and local manual runs
- no network or external API key is required
- query/document vectors remain stable across runs

### Manual collaborative validation

Prepare one real SQLite database with:
- a project
- work items
- thread comments on at least one work item and one project
- project description + standards text

Then validate:
1. `till embeddings status`
2. `till embeddings reindex --wait`
3. semantic task search using comment-derived language
4. TUI embeddings status mode
5. TUI search/fallback labels

## Acceptance Checklist

- generic document store exists and replaces task-only semantic storage on the active path
- `thread_context` lifecycle rows are enqueued and processed
- `project_document` lifecycle rows are enqueued and processed
- task search can be satisfied by thread/comment content through semantic ranking
- CLI/MCP/TUI status surfaces remain aligned and operator-readable
- real-DB integration tests pass
- TUI golden checks pass
- `just check` passes
- `just ci` passes
- one real local DB is prepared for collaborative manual verification
