# DROP_N — Closeout

Written once at drop close. See `drops/WORKFLOW.md` § "Phase 7 — Closeout" for the full step list.

- **Closed:** YYYY-MM-DD
- **Final commit:** <sha>
- **CI run:** <gh run url>

## Single-Phase vs Two-Phase Splice — Which Variant Your Project Uses

**Single-orch projects** (one orchestrator owns the whole drop end-to-end): the drop-orch writes directly to the project-root aggregation files (`HYLLA_FEEDBACK.md`, `REFINEMENTS.md`, `LEDGER.md`, `WIKI_CHANGELOG.md`, `WIKI.md`) at closeout. Follow each section below as-is.

**Two-phase projects with a persistent integrating orchestrator** (e.g. a steward orch that lives across drops, as in the Tillsyn repo): the drop-orch stages the content **inside this file + the drop dir** pre-merge on the drop branch; the integrating orch reads `drop_N/` post-merge on `main/` and splices into the project-root aggregation files. The drop-orch does **not** write the project-root MDs directly in this variant. Wherever a section below says "append to project-root `X.md`", two-phase projects should read that as "stage here for the integrating orch to append to project-root `X.md` post-merge."

**Failures retention.** Every branched level of `drop_N/` carries a `failures/` subdir. When a plan, QA, or build round fails, its artifact content moves into `failures/` at that level so the next iteration can read + count the prior failure. Never delete QA / plan / build artifacts.

## Code-Understanding Index Feedback Aggregation

<Roll up every `## Hylla Feedback` (or equivalent) subsection from `BUILDER_WORKLOG.md`. Deduplicate. Append the consolidated entry to the project-root `HYLLA_FEEDBACK.md` (or equivalent) so feedback survives the drop dir. Two-phase projects: stage the consolidated entry here; integrating orch appends post-merge.>

## Refinements

<Roll up usage findings that surfaced during the drop — things worth revisiting but out of scope for this drop. Append to `REFINEMENTS.md` (or an index-specific refinements file). One bullet per refinement with enough context for future-you. Two-phase projects: stage here; integrating orch appends post-merge.>

## Ledger Entry

<Per-drop cost, node / file / package count deltas, refactor notes, drop description. Append to project-root `LEDGER.md`. Two-phase projects: stage here; integrating orch appends post-merge.>

## Wiki Changelog

<One-liner describing what changed for this drop's readers. Append to `WIKI_CHANGELOG.md`. If any best practice shifted, also update `WIKI.md` in place. Two-phase projects: stage the changelog line + any `WIKI.md` delta here; integrating orch splices both post-merge.>

## Code-Understanding Index Ingest

- **Triggered:** YYYY-MM-DD HH:MM (after CI green — never before)
- **Mode:** full_enrichment (Go projects using Hylla)
- **Source:** `github.com/<org>/<project>@main` (always from the remote, never from a local working copy)
- **Result:** <ingest run id + outcome>

<For non-Go projects without a code-understanding index, delete this section. For FE projects or other languages with their own indexes, substitute the equivalent reingest command.>

## WIKI.md Updates

<If any section of `WIKI.md` was updated in place during closeout, list the headings touched here. The actual changes are in the `WIKI.md` diff; this is a pointer.>
