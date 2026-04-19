# DROP_1_5_DISCUSSIONS — drop-in-ready content for STEWARD

**Target**: STEWARD decides whether this needs a `main/DISCUSSIONS.md` section or if each converged topic routes into its own `DISCUSSIONS/<topic>.md`. Current `STEWARD_ORCH_PROMPT.md §10.3` lists DISCUSSIONS parent but no dedicated MD convention. Flag to dev at post-merge discussion.
**Tillsyn drop**: *not staged in pre-compact enumeration* (only 5 findings-drops in §1 of the pasted orch message: HYLLA_FINDINGS, LEDGER_ENTRY, WIKI_CHANGELOG, REFINEMENTS_RAISED, HYLLA_REFINEMENTS_RAISED). DISCUSSIONS parent = `DISCUSSIONS <id-not-captured-in-paste>`; DROP_1_5_DISCUSSIONS level_2 drop apparently NOT staged. STEWARD to confirm.

---

## Drop 1.5 converged discussion topics

### DD-1 — `v` keybinding ownership

- **Converged**: `v` → file viewer mode; `ctrl+d` → diff pane (P4).
- **Landed**: encoded in `workflow/drop_1_5/P2_A_BUILD_FILE_VIEWER_MODE/PLAN.md:14`; shipped in commits `0e22cdf` (ctrl+d) + `af2a69c` (v).
- **Status**: closed, no further action.

### DD-2 — Path B (web/electron separation of concerns)

- **Converged**: pre-compact Path A/B framing (7 leaf drops vs 3 drops with internal blockers) did NOT match dev's actual intent (frontend-agnostic code layout). Drop 1.5 shipped as Path A (7 leaf drops) with code under `internal/tui/gitdiff/` — which satisfies the PLANNING axis but NOT the ARCHITECTURE axis. Dev's real concern captured as refinement item 17 (extract `internal/tui/gitdiff/` + file-viewer renderer to frontend-agnostic `internal/view/...` package in a follow-up refactor drop).
- **Status**: closed via refinement 17; execution deferred to a dedicated architecture refactor drop after Drop 1.5 merges.

### Decomposition approach — small parallel planners

- **Converged**: ≤N small parallel planner subagents (one per surface/package, ≤15-min wall-clock) + orch-side synthesis + narrow build-drops with explicit `blocked_by` — dev-validated 2026-04-18 as system-as-designed cascade operating mode.
- **Status**: captured in `feedback_decomp_small_parallel_plans.md` + refinement item 16 + this DISCUSSIONS entry. Memory applies across all future Tillsyn / Hylla / FE projects.

### Coord substrate pivot — Rak MD vs Tillsyn MCP

- **Converged**: Tillsyn MCP mutations blocked by refinement 12 (invalid scope type on stored Scope=task) after MCP reconnect mid-Drop-1.5. Pivoted remainder of drop (P4-T4 + P2-A + DROP_END_LEDGER_UPDATE) to Rak-style MD coordination at `workflow/drop_1_5/<task>/` per `workflow/example/drops/drops/WORKFLOW.md`.
- **Status**: closed. Rak pattern documented in `workflow/drop_1_5/PLAN.md`; refinement 12 tracks the MCP fix. Until refinement 12 lands, Tillsyn MCP cannot be used for drop state transitions — Rak MD is the operational substrate.

### Ingest post-merge correction

- **Converged**: dev 2026-04-19 during DROP_END start: "hylla ingest happens after successful merge to main. go ahead." Overturns pre-compact planning that had ingest scheduled pre-merge. Memory `feedback_orchestrator_runs_ingest.md` updated. `STEWARD_ORCH_PROMPT.md §10.1.1` 12-step checklist is now inconsistent with this rule — refinement item 18 tracks the STEWARD self-refinement.
- **Status**: rule applied in Drop 1.5 DROP_END execution. STEWARD follow-up required (refinements-gate §10.4).
