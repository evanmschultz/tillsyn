# DROP_N — Closeout

Written once at drop close. See `drops/WORKFLOW.md` § "Phase 7 — Closeout" for the full step list.

- **Closed:** YYYY-MM-DD
- **Final commit:** <sha>
- **CI run:** <gh run url>

## Code-Understanding Index Feedback Aggregation

<Roll up every `## Hylla Feedback` (or equivalent) subsection from `BUILDER_WORKLOG.md`. Deduplicate. Append the consolidated entry to the project-root `HYLLA_FEEDBACK.md` (or equivalent) so feedback survives the drop dir.>

## Refinements

<Roll up usage findings that surfaced during the drop — things worth revisiting but out of scope for this drop. Append to `REFINEMENTS.md` (or an index-specific refinements file). One bullet per refinement with enough context for future-you.>

## Ledger Entry

<Per-drop cost, node / file / package count deltas, refactor notes, drop description. Append to project-root `LEDGER.md`.>

## Wiki Changelog

<One-liner describing what changed for this drop's readers. Append to `WIKI_CHANGELOG.md`. If any best practice shifted, also update `WIKI.md` in place.>

## Code-Understanding Index Ingest

- **Triggered:** YYYY-MM-DD HH:MM (after CI green — never before)
- **Mode:** full_enrichment (Go projects using Hylla)
- **Source:** `github.com/<org>/<project>@main` (always from the remote, never from a local working copy)
- **Result:** <ingest run id + outcome>

<For non-Go projects without a code-understanding index, delete this section. For FE projects or other languages with their own indexes, substitute the equivalent reingest command.>

## WIKI.md Updates

<If any section of `WIKI.md` was updated in place during closeout, list the headings touched here. The actual changes are in the `WIKI.md` diff; this is a pointer.>
