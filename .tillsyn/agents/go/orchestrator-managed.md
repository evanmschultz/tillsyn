---
name: orchestrator-managed
description: Documents orchestrator responsibilities for the four orchestrator-managed kinds in Tillsyn Go drops — closeout, refinement, discussion, human-verify. ORCH-MANAGED-R1 placeholder until Drop 4c.8 dedicated-agent split.
model: orchestrator-managed
tools: Read, Edit, Write, Grep, Glob
---

<!-- Tillsyn-project-local; lifted from ~/.claude/agents/ and adapted for Tillsyn's workflow. Future projects use embedded defaults shipped in Drop 4c.8. -->

## Role

This file documents the orchestrator's behavior for the four **orchestrator-managed action-item kinds** in Tillsyn's 12-kind closed enum: `closeout`, `refinement`, `discussion`, and `human-verify`. These kinds are handled directly by the orchestrator (the human-launched Claude Code session), not by a dedicated subagent.

**ORCH-MANAGED-R1:** Drop 4c.8 will split these into dedicated agents (`closeout-agent`, `refinement-agent`, `discussion-agent`, `human-verify-agent`). This file is the interim documentation of what the orchestrator does today.

## Core Constraint: Orchestrator Never Edits Go Source

The orchestrator handles these four kinds. The orchestrator MAY edit MD documents on the drop branch. The orchestrator **NEVER** calls `Edit` or `Write` on Go source files (`.go`) — only the builder subagent does that. This rule applies even during `discussion` or `refinement` items that surface code-adjacent decisions.

**Exception for trivial mid-flight stabilization:** Per `CLAUDE.md` §"Orchestrator-as-Hub Architecture", the orchestrator MAY directly edit Go for trivial typo fixes, single-constant updates, mid-flight build-green stabilization, or NIT-class absorptions surfaced by build-QA — when the cascade adds overhead without value. The judgment call: does the change benefit from a builder's TDD loop? When in doubt, prefer the builder.

## `kind: closeout`

Runs at drop end (WORKFLOW.md Phase 7). Orchestrator responsibilities:

1. Aggregate Hylla feedback from every subagent closing comment in the drop.
2. Aggregate refinements raised during the drop.
3. Write `workflow/drop_N/CLOSEOUT.md` with: Hylla feedback, refinements, ledger entry, wiki changelog.
4. Run `mage ci` locally (must pass).
5. Push + `gh run watch --exit-status` (wait for CI green).
6. Call `hylla_ingest enrichment_mode=full_enrichment` from the GitHub remote — drop-end only, after CI green.
7. Flip drop's L1 action item to `complete`.
8. Merge PR, fast-forward `main/`, remove drop worktree, delete local branch ref.

**Never call `hylla_ingest` before CI green. Never per-droplet. Never from local working copy.**

## `kind: refinement`

Perpetual / long-lived tracking umbrella. Drop-end findings roll up here. Orchestrator responsibilities:

- Read all subagent closing comments for refinement bullets raised during the drop.
- Create or update a `kind: refinement` action item with the aggregated findings.
- Refinement items accumulate across drops — they are NOT per-drop disposable items.
- Route each refinement to the appropriate future drop (add to PLAN.md backlog or create a new L1 plan item as appropriate).
- Do NOT silently discard refinements — they represent identified technical debt, ergonomic issues, or missed design opportunities that need future attention.

## `kind: discussion`

Cross-cutting decision park. Orchestrator responsibilities:

- Create a `kind: discussion` action item when a cross-cutting decision needs to be tracked.
- **Description = converged shape** — the final agreed-upon decision after the discussion concludes.
- **Comments = audit trail** — direct quotes from the dev, option analysis, iteration history.
- Actual dev ↔ orchestrator back-and-forth happens in chat (Discussion Mode per CLAUDE.md §"Coordination Model"). Mirror converged points back to the description.
- After convergence, update the description to reflect the decision. Comments preserve the full audit trail.

**No `@`-mention coordination.** Use `till.handoff` for structured routing, `till.comment` for append-only thread discussion. Never `@builder`, `@qa`, etc.

## `kind: human-verify`

Dev sign-off hold point. Orchestrator responsibilities:

- Create a `kind: human-verify` action item when dev sign-off is required before proceeding.
- Use `till.attention_item` to create an inbox entry pointing the dev to the specific decision or artifact.
- Include a checklist of what the dev needs to verify.
- The drop does NOT proceed past this point until the dev acknowledges.
- After dev sign-off: move `human-verify` to `complete` and unblock downstream work.

## MD-Doc Ownership Split

- **Drop-orch (drop branch):** owns per-drop artifact content in `main/workflow/drop_N/` and architecture MD edits when the drop's scope touches process — all on the drop branch, flowing to `main` via PR merge.
- **STEWARD (`main` post-merge):** runs post-merge on `main`, reads `main/workflow/drop_N/` content, collates into the six top-level MDs (`LEDGER.md`, `REFINEMENTS.md`, `HYLLA_FEEDBACK.md`, `WIKI_CHANGELOG.md`, `HYLLA_REFINEMENTS.md`, `WIKI.md`), then removes the drop worktree.

Do NOT use Claude Code's built-in `TaskCreate`/`TaskUpdate`/`TaskList` — they evaporate on compaction/restart. Use Tillsyn exclusively. Do NOT use markdown files for work tracking outside the designated `workflow/drop_N/` artifacts.

## Coordination Surfaces

- `till.comment` — shared append-only thread lane.
- `till.handoff` — structured next-action routing (may target a principal ID like `STEWARD` as a routing address).
- `till.attention_item` — durable inbox for human approval.
- Open handoffs are the primary Action Required rows for the addressed viewer.
