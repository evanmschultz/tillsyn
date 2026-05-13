---
name: closeout-agent
description: Orchestrator-managed closeout for Tillsyn FE drops. Aggregates Playwright coverage notes, visual regression audit, Hylla feedback, refinements. ORCH-MANAGED-R1 placeholder until Drop 4c.8.
model: orchestrator-managed
tools: Read, Edit, Write, Grep, Glob
---

<!-- Tillsyn-project-local; lifted from ~/.claude/agents/ and adapted for Tillsyn's workflow. Future projects use embedded defaults shipped in Drop 4c.8. -->

## Role

You document the orchestrator's closeout responsibilities for a Tillsyn FE drop. The `closeout` kind is **orchestrator-managed** in Drop 4c — there is no separate closeout subagent spawned today. This file documents what the orchestrator does during Phase 7 (Drop Closeout) for FE work.

**ORCH-MANAGED-R1:** Drop 4c.8 will split orchestrator-managed kinds into dedicated agents. This file is the interim placeholder with full FE closeout discipline documented.

**Relationship to go/closeout-agent.md:** FE closeout follows the same structure as Go closeout (WORKFLOW.md Phase 7 steps) but adds FE-specific aggregation: Playwright coverage summary, visual regression audit, stil token consistency notes, a11y coverage notes.

## FE Closeout Phase Steps (WORKFLOW.md Phase 7 + FE Extensions)

**Step 1 — Aggregate Hylla feedback:**

Read every FE subagent's closing comment for `## Hylla Feedback` sections. FE agents all write `N/A — FE project, Hylla indexes Go only today.` — collect these for the DROP_N_HYLLA_FINDINGS description to confirm consistent N/A reporting.

**Step 2 — Aggregate Playwright coverage notes:**

Read every FE builder's closing comment and QA reports for:
- Viewports tested (375/768/1280px confirmed?).
- Any intermediate-viewport gaps noted.
- Visual regression screenshot locations.
- `browser_snapshot` ARIA inspection summaries.

Compile into the CLOSEOUT.md section "FE Playwright Coverage Summary."

**Step 3 — Aggregate visual regression audit:**

For drops that introduced new components or layout changes: inventory which components have screenshot baselines and which don't. Note any components shipped without visual regression coverage. This is a load-bearing audit — future drops depend on knowing which surfaces have coverage.

**Step 4 — Aggregate a11y coverage notes:**

Read QA reports for a11y findings, both addressed (resolved during the drop) and deferred (routed to refinement). Document in CLOSEOUT.md "FE A11y Coverage Notes."

**Step 5 — Aggregate stil token consistency notes:**

Note any hardcoded values caught by QA during the drop, whether resolved or deferred. Document which new components were added with correct token usage.

**Step 6 — Aggregate refinements raised:**

Collect R-numbered refinements from FE subagent closing comments and QA reports. Aggregate into DROP_N_REFINEMENTS_RAISED.

**Step 7 — Write CLOSEOUT.md:**

Author `workflow/drop_N/CLOSEOUT.md` with:
- Drop title and state.
- Hylla feedback (Step 1) — will be all N/A for FE-only drops.
- Playwright coverage summary (Step 2).
- Visual regression audit (Step 3).
- A11y coverage notes (Step 4).
- Stil token consistency notes (Step 5).
- Refinements (Step 6).
- Ledger entry (cost, scope summary).
- Wiki changelog entries (any WIKI.md sections updated this drop).

**Step 8 — Verify CI green before proceeding:**

FE drops don't have `mage ci` — verify the FE build pipeline passes: `astro check`, `tsc --noEmit`, `eslint .`, `vitest run`, and Playwright MCP final visual sweep. Then `gh run watch --exit-status` after push until CI pipeline lands green.

**Step 9 — Hylla reingest (drop-end only):**

After CI is green: call `hylla_ingest enrichment_mode=full_enrichment` from the GitHub remote. **FE changes don't add to Hylla's Go index, but any accompanying Go changes (IPC handlers, etc.) will be indexed.** Never skip reingest even for FE-only drops — CI green is the gate. Note: Hylla ingest is currently disabled per `feedback_hylla_disabled_for_now.md` — when re-enabled, this discipline applies.

**Step 10 — Flip drop state, merge PR, cleanup:**

Same as Go closeout: move L1 action item to `complete`, merge PR, fast-forward `main/`, remove drop worktree, delete local branch ref.

## What FE Closeout Does NOT Do

- Does NOT trigger Hylla reingest per-droplet.
- Does NOT commit on behalf of builders.
- Does NOT write LEDGER.md, WIKI.md, REFINEMENTS.md directly (STEWARD's post-merge territory).
- Does NOT skip CI green before reingest.
- Does NOT skip the visual regression audit — even if QA passed, the audit documents coverage for future drops.

## STEWARD Boundary

Same as Go: STEWARD runs post-merge on `main`, splices CLOSEOUT.md content into top-level MDs, removes the drop worktree. FE closeout's CLOSEOUT.md is the handoff document STEWARD reads.

## Coordination Surfaces

- `till.comment` — append audit-trail comments on FE action items being closed.
- `till.attention_item` — dev sign-off if visual regression audit surfaces unresolved issues.
- `till.handoff` — structured routing if a FE finding requires a new plan item before merge.
