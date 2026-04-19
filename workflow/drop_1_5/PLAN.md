---
drop: DROP 1.5 — TUI REFACTOR
worktree: drop/1.5/
branch: drop/1.5
state: building
coord_substrate: md-files
substrate_reason: Tillsyn MCP `till_action_item move_state` persistently rejects stored `Scope: "task"` as "invalid scope type" (refinement 12). Pivoting remainder of drop to Rak-style md-file coordination per workflow/example/drops/drops/WORKFLOW.md.
---

# Drop 1.5 — Remaining Work Coordination

## Completed (in Tillsyn — frozen state)

- P3-A: BUILD PATH-PICKER CORE — `done`
- P3-B: BUILD FILE-PICKER SPECIALIZATION — `done`
- P4-T1: BUILD GITDIFF DIFFER + EXEC WRAPPER — `done`
- P4-T2: BUILD GITDIFF CHROMA HIGHLIGHTER — `done`
- P4-T3: BUILD DIFF MODE WIRE-UP (CTRL+D) — `done` (commits `0e22cdf` + `60b6fc5`, CI run `24611225218` green)

## Remaining chain

1. **P4-T4** — BUILD DIFF INPUT FROM RESOURCEREFS → `P4_T4_BUILD_DIFF_INPUT_FROM_RESOURCEREFS/`
2. **P2-A** — BUILD FILE VIEWER MODE (V KEYBINDING) → `P2_A_BUILD_FILE_VIEWER_MODE/`
3. **DROP-END** — LEDGER UPDATE → `DROP_END_LEDGER_UPDATE/`

Each subdirectory follows the Rak per-drop layout:

- `PLAN.md` — durable spec (acceptance criteria, TDD list, paths, packages, blocked_by).
- `BUILDER_WORKLOG.md` — append `## Round K` per build attempt (files touched, mage targets, Hylla feedback).
- `BUILDER_QA_PROOF.md` — append `## Round K` per QA proof pass (verdict + citations).
- `BUILDER_QA_FALSIFICATION.md` — append `## Round K` per QA falsification pass (verdict + attack vectors).
- `CLOSEOUT.md` — written once at task end (commit SHA, CI run ID, Hylla findings rollup).

## Orchestrator flow per task

1. Orch edits subtask's `PLAN.md` header `state: in_progress`.
2. Orch spawns `go-builder-agent` with Rak preamble (see below) + per-role appendix pointing at that task's dir.
3. Builder implements + appends `## Round 1` to `BUILDER_WORKLOG.md` + sets its `state: done_build`.
4. Orch spawns `go-qa-proof-agent` + `go-qa-falsification-agent` in parallel, each appends `## Round 1` to its own durable md.
5. Pass+pass → orch commits (`feat(tui): <scope>` single-line) → push → `gh run watch --exit-status` → write `CLOSEOUT.md` → flip subtask `state: done`.
6. Fail → respawn builder for `## Round 2`, QA respawns into `## Round 2`, loop until green.

## Rak spawn preamble (paste verbatim into every subagent prompt)

```
Paradigm override: this Drop 1.5 remainder is coordinated via MD files under
/Users/evanschultz/Documents/Code/hylla/tillsyn/workflow/drop_1_5/, NOT via
Tillsyn. Tillsyn MCP mutations are blocked by a persistent server-side
"invalid scope type" bug on stored Scope="task" action items. Ignore any
instructions in your agent definition that refer to till_*, capture_state,
attention_item, handoff, capability_lease, or auth_request. Read the task's
PLAN.md and the sibling BUILDER_WORKLOG.md / BUILDER_QA_*.md before acting.
Edit only the files your phase owns:

- Builder: the task's PLAN.md header `state` field + append to BUILDER_WORKLOG.md.
- QA Proof: append to BUILDER_QA_PROOF.md only.
- QA Falsification: append to BUILDER_QA_FALSIFICATION.md only.

The Tillsyn Scope="task" discrepancy is refinement item 12 in
/Users/evanschultz/.claude/projects/-Users-evanschultz-Documents-Code-hylla-tillsyn/memory/project_drop_1_5_tillsyn_refinements_raised.md
— do not attempt to work around it at the MCP layer.

Render your response beginning with a `# Section 0 — SEMI-FORMAL REASONING`
block containing `## Proposal`, `## QA Proof`, `## QA Falsification`, and
`## Convergence` passes (4-pass subagent variant per ~/.claude/CLAUDE.md §
"Semi-Formal Reasoning") before your final output. Each pass uses the
5-field certificate (Premises / Evidence / Trace or cases / Conclusion /
Unknowns) where applicable. Convergence must declare (a) QA Falsification
found no unmitigated counterexample, (b) QA Proof confirmed evidence
completeness, (c) remaining Unknowns are routed back to the orchestrator.
If any fail, loop back before Convergence.

Section 0 reasoning stays in the orchestrator-facing response only — do NOT
write Section 0 into PLAN.md, BUILDER_WORKLOG.md, BUILDER_QA_*.md, or
CLOSEOUT.md.
```

## Verification

- Commits land on branch `drop/1.5` in worktree `/Users/evanschultz/Documents/Code/hylla/tillsyn/drop/1.5/`.
- `mage ci` must pass locally **and** on CI before each task closes.
- Final push of last task's commits triggers drop-end Hylla ingest (full enrichment, from remote) — DROP_END_LEDGER_UPDATE task.

## Cleanup at drop end

`workflow/drop_1_5/` is under the bare-repo `workflow/` directory, which is untracked by any branch. No `git rm` needed. Either delete the directory after drop merges to `main`, or keep as post-mortem audit.
