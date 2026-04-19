# DROP_N — Builder Worklog

Append a `## Droplet N.M — Round K` section per build attempt. See `drops/WORKFLOW.md` § "Phase 4 — Build (per droplet)" for what each section should contain.

## Droplet N.1 — Round 1

- **Builder:** <builder-agent-type> (e.g. `go-builder-agent` / `fe-builder-agent`)
- **Started:** YYYY-MM-DD HH:MM
- **Files touched:** <list>
- **Build-tool targets run:** <e.g. `mage build` (pass), `mage test-pkg <pkg>` (pass), `npm run test -- <path>` (pass), …>
- **Notes:** <design choices, surprises, library quirks, references to Context7 / language doc / LSP queries that mattered>

### Hylla Feedback

<For Go projects using Hylla. For other languages, rename to the code-understanding index feedback you're collecting, or delete this subsection if you don't use one. Record any case where the index missed and a fallback (LSP / Read / Grep) was needed. Format: Query → Missed because → Worked via → Suggestion. Aggregated into `HYLLA_FEEDBACK.md` (or equivalent) at closeout.>
