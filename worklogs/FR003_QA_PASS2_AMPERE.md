# FR003_QA_PASS2_AMPERE

Timestamp (UTC): 2026-03-05T03:25:02Z  
Lane: QA pass 2 (FR-003 modal parity), checkpoint `QA2-FR003-AMPERE`

## Scope
- Inspected (read-only):
  - `internal/tui/model.go`
  - `internal/tui/*_test.go`
  - `PLAN.md`
  - `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md`
- Executed tests:
  - `just test-pkg ./internal/tui`

## Findings (Severity Ordered)

### Medium
1. Tracker/doc completeness for FR-003 is not recorded yet in active ledgers.
- Evidence:
  - Findings ledger only has `FR-001` and `FR-002`, no `FR-003`: `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md:123`, `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md:124`
  - Fix/validation rows only include `FX-001` and `FX-002`: `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md:130`, `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md:131`, `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md:137`, `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md:138`
  - PLAN checkpointing currently documents FR-001/FR-002 only: `PLAN.md:2941`, `PLAN.md:3006`
- Impact:
  - FR-003 status/evidence is not yet traceable in canonical planning/worksheet trackers.

### Low
2. Automated tests strongly cover task-info comments/scroll and edit metadata behavior, but do not explicitly assert edit-modal comments metadata rendering tokens.
- Evidence (what is covered):
  - Full task-info comments list with metadata assertions: `internal/tui/model_test.go:1429`
  - Task-info scroll wiring (keyboard + mouse): `internal/tui/model_test.go:1627`
  - Edit-task metadata prefill/submit behavior: `internal/tui/model_test.go:1563`
- Evidence (implementation exists in edit layout):
  - Edit task sectioned body includes comments block and owner/actor/id/summary/body rendering with markdown: `internal/tui/model.go:12127`, `internal/tui/model.go:12138`, `internal/tui/model.go:12145`
- Impact:
  - Possible UI-token drift in edit-comments section could pass tests unnoticed.

## Validation Notes (Functional)
- Shared full-screen renderer path is implemented for all requested modes:
  - Shared renderer helpers: `internal/tui/model.go:12874`, `internal/tui/model.go:12904`
  - `modeTaskInfo` uses shared renderer: `internal/tui/model.go:12993`, `internal/tui/model.go:13005`
  - `modeAddTask/modeEditTask/modeAddProject/modeEditProject` route through same node-modal viewport path: `internal/tui/model.go:13612`, `internal/tui/model.go:13639`, `internal/tui/model.go:13657`
- Full-screen-style sizing path is present:
  - Width policy: `internal/tui/model.go:11863`, `internal/tui/model.go:11866`, `internal/tui/model.go:12875`, `internal/tui/model.go:12881`
  - Body height policy: `internal/tui/model.go:11910`, `internal/tui/model.go:11915`
- Task-info scroll/edit rendering shows no obvious regressions:
  - Keyboard + mouse scroll wiring in update path: `internal/tui/model.go:6394`, `internal/tui/model.go:6411`, `internal/tui/model.go:9083`, `internal/tui/model.go:9090`
  - Task-info description editor open/return path: `internal/tui/model.go:2748`, `internal/tui/model.go:2895`, `internal/tui/model.go:2917`

## Commands Run + Outcomes
1. `rg -n "renderNodeModalViewport|nodeModalBoxStyle|modeTaskInfo|..." internal/tui/model.go` -> PASS (located renderer/sizing/mode wiring paths).
2. `rg -n "FR-003|FR003|modal parity|..." PLAN.md COLLAB_VECTOR_MCP_E2E_WORKSHEET.md` -> PASS (confirmed FR-001/FR-002 records; no FR-003 row).
3. `rg -n "TaskInfo|EditTask|renderModeOverlay|..." internal/tui/model_test.go` -> PASS (located relevant coverage).
4. `nl -ba ...` focused slices across the files -> PASS (captured line-level evidence).
5. `just test-pkg ./internal/tui` -> PASS (`ok github.com/hylla/tillsyn/internal/tui (cached)`).

## Acceptance Criteria Checklist
1. Verify shared full-screen renderer path for `modeTaskInfo` + `modeAddTask`/`modeEditTask`/`modeAddProject`/`modeEditProject`: **PASS**
2. Verify task edit body sections include comments metadata + markdown behavior in sectioned layout: **PASS**
3. Run `just test-pkg ./internal/tui`: **PASS**
4. Run Context7 if tests/runtime fail: **PASS (N/A, no failures occurred)**
5. Provide findings with severity + file:line refs: **PASS**

## Architecture-Boundary Compliance Note
- Read-only QA lane executed as requested; no production code or architecture-layer changes were made.

## Context7 Compliance Note
- Conditional requirement was failure-triggered for this lane; no test/runtime failures occurred, so no Context7 re-consult was required.

## Risks / Open Questions
1. Should FR-003 be added as explicit ledger/tracker entries in both `PLAN.md` and `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md` before closure?
2. Do we want one targeted test to lock edit-modal comments metadata tokens (actor/owner/id/summary markdown body) to prevent UI drift?
