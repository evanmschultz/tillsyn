---
task: DROP 1.5 END — LEDGER UPDATE
tillsyn_id: 5bc5bb8d (per earlier Tillsyn list — verify at execute time)
role: orchestrator (not builder — orch runs ingest; dev runs git ops)
state: in_progress_awaiting_merge
blocked_by: none (P4-T4 + P2-A both done; ingest waits for post-merge per 2026-04-19 dev correction)
worktree: /Users/evanschultz/Documents/Code/hylla/tillsyn/drop/1.5/
---

# DROP 1.5 END — LEDGER UPDATE

## Purpose

Drop-end closeout task. Runs once all sibling build-drops are `done` and their CI runs are green. Owned by drop-orch. Responsible for:

1. Running `hylla_ingest` (full enrichment, from GitHub remote `github.com/evanmschultz/tillsyn@main`, after final push + CI green).
2. Populating the five STEWARD-owned level_2 findings drops under persistent level_1 parents with Drop 1.5 content (descriptions only — STEWARD writes MD on main post-merge):
   - `DROP_1_5_DISCUSSIONS`
   - `DROP_1_5_HYLLA_FINDINGS`
   - `DROP_1_5_LEDGER_ENTRY`
   - `DROP_1_5_WIKI_CHANGELOG`
   - `DROP_1_5_REFINEMENTS_RAISED` — transcribe all 15 items from `/Users/evanschultz/.claude/projects/-Users-evanschultz-Documents-Code-hylla-tillsyn/memory/project_drop_1_5_tillsyn_refinements_raised.md` verbatim.
3. Verifying all pre-merge checks pass (see STEWARD_ORCH_PROMPT.md §10 12-step checklist).
4. Requesting dev to merge `drop/1.5` → `main`.
5. After merge, STEWARD writes the five MD files on `main` post-merge (outside this drop's scope).

## Hylla Ingest Invariants

- Always `enrichment_mode=full_enrichment`. Never `structural_only`.
- Always source from the GitHub remote (`github.com/evanmschultz/tillsyn@main`). Never from local working copy.
- Never before `git push` + `gh run watch --exit-status` green.
- Only the drop-orch calls `hylla_ingest`. Subagents never do. STEWARD never does.

## Pre-Merge Checklist

See `main/STEWARD_ORCH_PROMPT.md` §10 (12-step drop-end checklist). Do NOT duplicate here — read that section at execute time.

## Cleanup

- `workflow/drop_1_5/` at bare-repo level can stay as post-mortem or be deleted after merge.
- The Tillsyn `DROP 1.5 — TUI REFACTOR` level_1 drop stays in its current `in_progress` state until refinement 12 is fixed and a manual TUI flip can finalize it. The audit trail lives in:
  - This `workflow/drop_1_5/` directory (pre-cleanup).
  - Git commit history on `drop/1.5` branch.
  - `~/.claude/projects/-Users-evanschultz-Documents-Code-hylla-tillsyn/memory/project_drop_1_5_tillsyn_refinements_raised.md` (items 1-15).
