# Search And Embeddings Follow-Up Notes

## Deferred UX Note

`PLAN.md` was not edited from this lane per branch instructions. This note captures the deferred design items instead.

## Embeddings Inventory Screen Purpose Today

The TUI `Embeddings` screen is currently an operator inventory and health surface, not the primary human search workflow.

It is useful today for:

- checking whether indexed subjects are `pending`, `running`, `ready`, `failed`, or `stale`
- seeing scope-level readiness counts
- confirming the active provider/model signature on indexed subjects
- triggering `reindex` / `force reindex`
- opening the backing node from one lifecycle row when a human-readable title/path is available

## Explicitly Deferred Questions

These are intentionally deferred from this lane and should be resolved in a later design pass rather than during the current operational embeddings wave:

- whether the embeddings inventory should remain a dedicated health/inventory screen or merge with search
- whether inventory filtering should evolve into a full semantic search/browse surface
- how much search capability belongs inside the inventory modal versus the main `/` search flow
- whether CLI and MCP inventory output should move from lifecycle IDs to richer path/title labels everywhere
- whether search and embeddings inventory should share one unified results surface for operators

## Search Regression Finding

The TUI `/` search regression came from the search modal closing before async results arrived, and zero-result queries falling back to the board instead of staying in an explicit empty-results overlay. That behavior has been corrected in this lane and covered with new regression tests.
