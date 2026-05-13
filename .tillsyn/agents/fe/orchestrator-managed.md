---
name: orchestrator-managed
description: Documents orchestrator responsibilities for the four orchestrator-managed kinds in Tillsyn FE drops — closeout, refinement, discussion, human-verify. FE-specific notes on Playwright, visual audit, stil consistency. ORCH-MANAGED-R1 placeholder.
model: orchestrator-managed
tools: Read, Edit, Write, Grep, Glob
---

<!-- Tillsyn-project-local; lifted from ~/.claude/agents/ and adapted for Tillsyn's workflow. Future projects use embedded defaults shipped in Drop 4c.8. -->

## Role

This file documents the orchestrator's behavior for the four **orchestrator-managed action-item kinds** in Tillsyn FE drops: `closeout`, `refinement`, `discussion`, and `human-verify`. These kinds are handled directly by the orchestrator, not by a dedicated subagent.

**ORCH-MANAGED-R1:** Drop 4c.8 will split these into dedicated agents. This file is the interim documentation for FE drop orchestration.

**Relationship to go/orchestrator-managed.md:** FE orchestration follows the same 4-kind structure as Go orchestration, with additional FE-specific notes. Read both files if you are handling a drop with mixed Go + FE work.

## Core Constraint: Orchestrator Never Edits FE Source

The orchestrator handles these four kinds. The orchestrator **NEVER** calls `Edit` or `Write` on FE source files (`.tsx`, `.astro`, `.css`, `.ts` in `fe/`) — only the FE builder subagent does that. This rule applies even during `discussion` or `refinement` items that surface code-adjacent FE decisions.

**Exception for trivial mid-flight stabilization:** Per `CLAUDE.md` §"Orchestrator-as-Hub Architecture", the orchestrator MAY directly edit for trivial typo fixes or single-constant updates when the cascade adds overhead without value. For FE: non-behavioral CSS comment fixes, migration marker corrections on a single file. When in doubt, prefer the FE builder.

## `kind: closeout` (FE Extensions)

Runs at drop end (WORKFLOW.md Phase 7 + FE extensions documented in `fe/closeout-agent.md`). Key FE-specific additions:

**Playwright coverage summary:** After aggregating subagent closing comments, compile which components have Playwright screenshot coverage at 375/768/1280px. Identify gaps. Document in CLOSEOUT.md "FE Playwright Coverage Summary."

**Visual regression audit:** For drops introducing new visible UI: inventory which new components have screenshot baselines. Any new component without a screenshot baseline is a debt item. Add to refinements if not addressed during the drop.

**A11y coverage notes:** Compile all a11y findings from QA reports — both addressed and deferred. Note which interactive elements received `browser_snapshot` ARIA inspection.

**Stil token consistency:** Note any hardcoded-value findings from QA. Confirm all new components use `var(--*)` tokens from `src/styles/tokens.css`.

After these aggregations: write CLOSEOUT.md, verify CI green, push, `gh run watch --exit-status`, Hylla reingest (drop-end only, full enrichment, from GitHub remote), flip L1 to complete, merge PR, cleanup worktree.

## `kind: refinement` (FE Extensions)

Same as Go: perpetual rollup, accumulates across drops. FE-specific refinement categories to track:

- **Visual regression debt:** components shipped without Playwright screenshot baselines.
- **A11y debt:** known accessibility gaps deferred from a build QA round.
- **Stil token drift:** hardcoded values that should be tokens but were deferred.
- **Migration marker debt:** components that didn't get `// MIGRATION TARGET: @hylla/stil-solid` markers.
- **Intermediate-viewport gaps:** responsive issues at non-standard viewports noted by QA falsification but deferred.

Route each FE refinement to the appropriate future drop. Do NOT silently discard FE refinements — they represent visual and accessibility debt.

## `kind: discussion` (FE Context)

Same as Go: description = converged shape, comments = audit trail. FE-specific discussion topics:

- Component boundary decisions (when to extract a new island vs keep inline).
- A11y approach decisions (ARIA pattern choices, keyboard navigation contract).
- Stil token naming decisions (new token names that affect multiple components).
- Wails IPC contract decisions (FE ↔ Go boundary shape changes).
- Vim engine extension decisions (new key bindings, context-switch behavior).

After FE decisions converge: update the discussion item's description. Mirror to PLAN.md or REVISION_BRIEF if the decision affects future planning.

## `kind: human-verify` (FE Context)

Same as Go: dev sign-off hold point via `till.attention_item`. FE-specific human-verify triggers:

- Visual design approval before shipping a new visible UI component.
- A11y audit by a human reviewer when `browser_snapshot` alone is insufficient.
- Viewport coverage approval when intermediate-size gaps were identified by QA.
- Migration marker review when a new component is flagged for future extraction.

The drop does NOT proceed past this point until the dev acknowledges. After sign-off: move `human-verify` to `complete` and unblock downstream work.

## MD-Doc Ownership Split

Same as Go:
- **Drop-orch (drop branch):** owns `workflow/drop_N/` artifacts (PLAN.md, CLOSEOUT.md, QA files, BUILDER_WORKLOG.md).
- **STEWARD (`main` post-merge):** splices CLOSEOUT.md content into top-level MDs, removes the drop worktree.

Do NOT use Claude Code's built-in `TaskCreate`/`TaskUpdate`/`TaskList` — they evaporate on compaction. Use Tillsyn exclusively.

## Coordination Surfaces

- `till.comment` — shared append-only thread lane.
- `till.handoff` — structured next-action routing.
- `till.attention_item` — durable inbox for human approval (visual design sign-off, a11y audit).
- Open handoffs are the primary Action Required rows for the addressed viewer.
