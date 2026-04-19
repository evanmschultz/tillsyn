# DROP_1_5_REFINEMENTS_RAISED — drop-in-ready content for STEWARD

**Target**: `main/REFINEMENTS.md` append after merge. STEWARD writes.
**Tillsyn drop**: `318cd690-b88d-4a5a-8a84-7c9486286305` (under REFINEMENTS parent `eea91708-...`).
**Source of record**: `~/.claude/projects/-Users-evanschultz-Documents-Code-hylla-tillsyn/memory/project_drop_1_5_tillsyn_refinements_raised.md` — this file mirrors that memory verbatim at drop-end, in numerical order.

---

## Drop 1.5 Tillsyn / Claude Code refinements raised (18 items)

Full details (Evidence / Summary / Severity / Proposed fix / Workaround) live in the memory file above. Brief index here; STEWARD splices the full content into `main/REFINEMENTS.md` per the memory.

### 1. Auth layer drop-collapse parity (carryover)
Session path grammar still uses `project/branch/phase`; lease `scope_type` enum rejects `branch`/`phase`/`task`/`subtask`. Only `project` passes validation. **Severity: high.**

### 2. Auth cache hook failed across compaction
`PreCompact`/`SessionStart` hooks at `~/.claude/hooks/save_tillsyn_auth.sh` + `restore_tillsyn_auth.sh` did not survive compaction during Drop 1.5; cache dir empty post-compact; session secret lost; dev approved fresh orch auth to unblock. **Severity: high.**

### 3. Subagent spawn-prompt credentials lost on mid-run compaction
go-planning-agent compacted ~13min into a 46-tool-use session; spawn-prompt bundle dropped; subsequent `till.plan_item update` returned `session_required`; ~13 min of analysis lost. **Severity: high.**

### 4. Cascade workflow discipline — drop-orch spin-up checklist + planner-contract enforcement
Five cascade rules violated in one DROP_1.5_ORCH session (drop root left todo, planners scoped as analysis-producers with `completion_notes`, no `blocked_by` wiring, STEWARD level_2 drops never staged, LEDGER_UPDATE never created). **Severity: high.** Proposed fix: Drop 3 template `child_rules` auto-generation + `/drop-orch-spinup` slash command + first-class `research` vs `planner` role split.

### 5. Subagent spawn-prompt credentials lost on INITIAL tool invocation (not compaction)
Different from item 3: agent never threads spawn-prompt auth bundle onto its first tool call. `till.plan_item update` returns `session_required` on first mutation. **Severity: high.** Proposed fix: agent-definition credential-threading section + Tillsyn-side short-lived "spawn-scoped continuation claim" handle.

### 6. `appendResourceRefIfMissing` dedup ignores `Tags` field
`internal/tui/model.go:9711-9722` compares on `ResourceType + PathMode + Location` only; `Tags []string` not in equality check. Surfaced by P3-B QA falsification. **Severity: medium today, high once P4-T4 package picker ships as production writer.**

### 7. `go.mod` indirect marker drift after new direct imports
`go.mod:117` still reads `chroma/v2 // indirect` after P4-T2 added direct import. `mage ci` has zero `go mod tidy` enforcement gate. **Severity: low today, medium for downstream tooling.**

### 8. `mage test-func` hardcodes `-count=1`, preventing stress iteration
P4-T2 QA falsification needed `-race -count=50` for `TestHighlighter_Concurrent`; no mage target supports it. **Severity: medium.** Proposed fix: `mage test-stress <pkg> <func> <count>` target.

### 9. `mage test-golden-update` hard-scoped to `./internal/tui`, misses sibling packages
P4-T2 golden lives in `./internal/tui/gitdiff/testdata/golden/` — invisible to the existing target. Env-var substitute used. **Severity: medium.** Proposed fix: accept optional package-path arg.

### 10. Equal-scope orchestrator lease overlap requires unreachable `override_token`
`AllowEqualScopeDelegation: true` still blocks equal-scope orch overlap; no documented way to mint override token. Breaks parallel drop-orch arrangement (Drop 1 / Drop 1.5). **Severity: high.** Proposed fix: auto-mint + auto-consume override when both sides consent.

### 11. Cross-orchestrator lease revoke permitted at project scope — guardrail too loose
DROP_1.5_ORCH successfully revoked DROP_1_ORCH's active project-scope orchestrator lease. Peer orchs at same scope can kill each other's leases. **Severity: high.** Proposed fix: same-level-peer-orch cross-revoke must be rejected.

### 12. Server-side authorize layer rejects stored `Scope: "task"` as "invalid scope type" after MCP reconnect
Every guarded mutation (update / move_state / comment create / capability_lease issue scope_type=actionItem) on a Kind=task, Scope=task action item returns `invalid scope type` after MCP reconnect. Fresh sessions do NOT recover. Dev had to flip states manually in TUI. **Severity: high — BLOCKING for cascade autonomy.** Proposed fix: remap stored `Scope` value into authorize-path scope-type enum before internal capture_state call; audit every authorize-pipeline scope validator for grammar alignment.

### 13. Parent task moved to terminal `done` while child QA tasks still incomplete
P4-T3 build-drop moved to `done` while both QA twins were in `todo` (30s window). Server accepted the terminal transition without engaging parent-blocks-on-incomplete-children guardrail. **Severity: high.** Proposed fix: server-side refusal of `done` on any parent with non-terminal direct children + per-kind `require_children_done` default at template layer.

### 14. Local `mage ci` green but GH Actions `mage ci` failed on gofumpt — parity gap instance
P4-T3 push `0e22cdf`: local `mage ci` green, CI failed immediately on `Checking Go formatting`. Required gofumpt fixup commit `60b6fc5`. **Severity: medium.** Proposed fix: align local + CI gofumpt invocations to identical toolchain resolution + file enumeration + exit-nonzero semantics.

### 15. `client_type` required on MCP-originated `till_auth_request create` is bad ergonomics
MCP create allows empty `client_type`; CLI approve rejects it with opaque "invalid client type". **Severity: medium.** Proposed fix: server auto-populates `client_type="mcp"` on MCP-transport creates, OR approve-side validator names the missing field.

### 16. Small-parallel decomposition validated as system-as-designed cascade operating mode
Dev 2026-04-18: "I love your decomp approach. That is how I meant it to work when designing this system." Pattern: ≤N small parallel planners (≤15-min wall-clock, one surface/package each) + orch-side synthesis + narrow build-drops with explicit `blocked_by`. **Severity: informational.** Proposed capture: WIKI.md §Cascade Operating Mode + Drop 2 `metadata.role` lint for analysis-only planner misclassification.

### 17. Extract frontend-agnostic rendering + diff core out of `internal/tui/` namespace
Dev 2026-04-18 (Path B intent): "we are going to have web and/or electron apps not just the tui... better separation of concerns." `internal/tui/gitdiff/` and `internal/tui/file_viewer_renderer.go` are logically frontend-agnostic but path-bound to TUI. **Severity: medium-high.** Proposed fix: follow-up architecture refactor drop — move `internal/tui/gitdiff/` → `internal/view/gitdiff/`, extract pure renderer logic from `internal/tui/file_viewer_renderer.go` → `internal/view/viewer/`. Leave Bubble Tea-specific pieces (mode lifecycle, viewport, keymap) in TUI. Execute as dedicated post-drop refactor drop; NOT mid-drop.

### 18. STEWARD_ORCH_PROMPT.md §10.1.1 step order contradicts post-merge-ingest rule
`main/STEWARD_ORCH_PROMPT.md:319-334` places `hylla_ingest` at step 5, before "Signal dev to merge" at step 12 — so ingest runs PRE-merge. Dev correction 2026-04-19: "hylla ingest happens after successful merge to main." `feedback_orchestrator_runs_ingest.md` memory updated. **Severity: medium.** Proposed fix: STEWARD edits §10.1.1 to re-sequence ingest as post-merge (after dev merges drop branch, orch confirms merge, THEN calls `hylla_ingest @main`). Trigger STEWARD self-refinement under refinements-gate §10.4 post-merge.
