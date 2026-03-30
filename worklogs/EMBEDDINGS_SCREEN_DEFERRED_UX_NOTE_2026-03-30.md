## Deferred UX Note

Date: 2026-03-30
Lane: `agent/embeddings`

This lane intentionally leaves the embeddings inventory screen as an operational
health/status surface, not a fully designed search/browse workflow.

Current accepted purpose:
- show whether embeddings are `pending`, `running`, `ready`, `failed`, or `stale`
- show operator-visible scope, runtime/model signature, and row-level lifecycle state
- support explicit reindex and direct open-to-node actions for investigation

Deferred for later design work:
- whether embeddings inventory should become a richer search surface
- whether inventory and search should be merged into one operator workflow
- what the ideal filtering/navigation model should be once comments and other
  indexed subject types expand beyond the current MVP

No design decision is being made on those deferred UX questions in this lane.
This note records the issue so follow-up work can address it deliberately.
