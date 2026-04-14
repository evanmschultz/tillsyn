# Hylla Feedback

Running log of Hylla ergonomics, tool-shape, search-quality, and definition-quality feedback from subagents and the orchestrator.

At slice-end, the orchestrator aggregates feedback from every subagent closing comment produced during the slice (search each plan-item's comments for a `## Hylla Feedback` section) into a new `## Slice <N>` heading below.

## What Goes In

- **Failed-search anecdotes** — "I searched for X with `hylla_search_keyword`; Hylla returned nothing; I fell back to `LSP.workspaceSymbol` which found it."
- **Tool-shape gripes** — awkward parameters, confusing response shapes, missing fields, inconsistent naming.
- **Def-quality gripes** — summaries that don't summarize, docstrings that mislead, weird ID formats.
- **Missing capabilities** — "I wanted to X; Hylla doesn't seem to support it."
- **Ergonomics** — stuff that slows us down: friction, ceremony, repeated boilerplate.

## What Doesn't Go In

- Genuine bugs — file an issue instead; this file is for ergonomics + shape feedback.
- Praise — the point is improvement; keep the file focused.

## Entry Format

Every entry is terse and concrete.

```
- **Query**: <tool name + key inputs>
- **Missed because**: <hypothesis: wrong search mode, schema gap, missing summary, stale ingest, etc.>
- **Worked via**: <fallback tool + inputs that found the thing>
- **Suggestion**: <one-liner for what Hylla could do better>
```

Subagents who had no Hylla misses on their task still emit a `## Hylla Feedback` section in their closing comment with `None — Hylla answered everything needed.`. Explicit "no miss" is useful signal.

---

## Slice 0 — Project Reset + Docs Cleanup

_To be populated by the `SLICE 0 END — LEDGER UPDATE` task once Slice 0 closes._
