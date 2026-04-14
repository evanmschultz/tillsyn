# Tillsyn Cascade Ledger

Per-slice snapshot of project state, cost, and code-quality deltas. Populated by the orchestrator at two moments:

- Once, at project start: **Starting Baseline** below.
- Per-slice, at slice-end: a new `## Slice <N> — <Title>` section appended by the `SLICE <N> END — LEDGER UPDATE` task.

**Hylla ingest invariants (every slice-end ingest):**

- Full enrichment (`enrichment_mode=full_enrichment`) — never `structural_only`.
- From the GitHub remote.
- After `git push` **and** `gh run watch --exit-status` green.
- Orchestrator-run only. Subagents never call `hylla_ingest`.

**Node count format:** `total: N, code: A, tests: B, packages: C`. The identity `A + B + C = N - 1` holds because Hylla also emits one project-level snapshot node that is not code, test, or package.

---

## Starting Baseline — 2026-04-13 (pre-Slice-0)

- **Artifact ref**: `github.com/evanmschultz/tillsyn@main`
- **Ingest snapshot**: 3
- **Commit**: `0af254066bf6be0758ba83f4f166ce19fe1a14ad` (git describe: `0af2540`)
- **Enrichment mode**: `full_enrichment`
- **Enrichment models**: openai `gpt-5-mini` (summary) + `text-embedding-3-small` (embeddings)
- **Latest ingest run cost**: $0.3908 (snapshot 2 → 3 delta, 129s enrichment)
  - Summary calls: 133 (4 failures, 129 successful) · embedding calls: 129 · reused: 3
  - Tokens: 1,112,635 input / 68,630 output / 47,296 reasoning
- **Cumulative cost-to-date (lineage)**: $8.2342 across all snapshots to date
  - Summary calls: 4,780 · embedding calls: 4,626 · reused: 114
  - Tokens: 14,210,218 input / 2,553,216 output / 1,898,816 reasoning
- **Node count**: TBD — first populated count is captured at the `SLICE 0 END — LEDGER UPDATE` run. Format at that point: `total: N, code: A, tests: B, packages: C` with `A + B + C = N - 1`.
- **Orphan count (baseline)**: TBD — same as node count, first captured at Slice 0 end.
- **Ingest run IDs (lineage)**:
  - `inspect-update-1776113037713825000-github-com-evanmschultz-tillsyn-main` (snapshot 3, 2026-04-13)
  - Prior snapshots: 1 (commit `b411b48`), 2 (commit `870de3e`).

---

## Slice 0 — Project Reset + Docs Cleanup

_To be populated by the `SLICE 0 END — LEDGER UPDATE` task once Slice 0 closes._

Template for every slice entry (append to this file; do not rewrite prior entries):

```
## Slice <N> — <Title>

- **Closed**: YYYY-MM-DD
- **Slice plan-item ID**: <uuid>
- **Ingest snapshot**: <snapshot_int>
- **Commit**: `<sha>` (git describe: `<short>`)
- **Ingest cost (this run)**: $X.XXXX
- **Cost-to-date (lineage)**: $Y.YYYY
- **Node count**: total: N, code: A, tests: B, packages: C (Δ±K vs prior)
- **Orphan count**: prev → now (found P, cleaned Q, residual R)
- **Refactors / code-quality deltas**:
  - bullet
- **Description**: 2–3 sentence summary of what shipped in this slice.
- **Commit SHAs**: sha1, sha2, …
- **Notable plan-item IDs**: uuid1, uuid2, …
- **Unknowns forwarded**: bullet, or "none".
```
