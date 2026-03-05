# FR003_QA_PASS1_DARWIN

Timestamp (UTC): 2026-03-05T05:41:16Z  
Lane: QA pass 1 (FR-003 modal parity), checkpoint `QA1-FR003-DARWIN`

## Scope
- Inspected (read-only):
  - `internal/tui/model.go`
  - `internal/tui/*_test.go`
  - `PLAN.md`
  - `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md`
- Executed tests:
  - `just test-pkg ./internal/tui`

## Findings (Severity Ordered)

### Low
1. Automated coverage does not explicitly assert task-info vs task-edit section-order parity; current tests validate presence tokens and mode titles, but not full ordered section parity.
- Evidence:
  - Task/add/edit overlay assertions focus on title/field presence: `internal/tui/model_test.go:4720`, `internal/tui/model_test.go:4733`
  - Task info + edit label-section token checks exist, but not explicit ordered parity checks: `internal/tui/model_test.go:7512`, `internal/tui/model_test.go:7527`
  - Implementation now contains explicit section-structured builders for both task info and task form:
    - task form section builder: `internal/tui/model.go:11973`
    - task info section builder: `internal/tui/model.go:12276`

## Validation Notes (Functional)
- Shared full-screen frame/render path is in place for info + task/project add/edit modals:
  - shared frame + viewport renderer helpers: `internal/tui/model.go:12874`, `internal/tui/model.go:12887`, `internal/tui/model.go:12904`
  - `modeTaskInfo` routes through shared renderer: `internal/tui/model.go:12993`, `internal/tui/model.go:13005`
  - task/project add/edit routes through same shared renderer path: `internal/tui/model.go:13612`, `internal/tui/model.go:13639`, `internal/tui/model.go:13657`
- Shared full-screen sizing path is used:
  - width policy: `internal/tui/model.go:11866`, `internal/tui/model.go:11868`
  - node modal frame width: `internal/tui/model.go:12875`, `internal/tui/model.go:12881`
  - body viewport height policy: `internal/tui/model.go:11913`, `internal/tui/model.go:11918`
- Task edit section order/style tracks task-info section layout (read vs edit state):
  - task-form order: summary -> description -> subtasks -> labels -> dependencies -> comments -> resources -> metadata
    - starts at: `internal/tui/model.go:11973`
    - dependencies section: `internal/tui/model.go:12090`
    - comments section: `internal/tui/model.go:12127`
    - resources section: `internal/tui/model.go:12162`
    - metadata inputs: `internal/tui/model.go:12181`
  - task-info order with matching section sequence:
    - starts at: `internal/tui/model.go:12276`
    - dependencies section: `internal/tui/model.go:12343`
    - comments section: `internal/tui/model.go:12355`
    - resources section: `internal/tui/model.go:12386`
    - metadata section: `internal/tui/model.go:12407`

## Commands Run + Outcomes
1. `rg -n "taskInfoBodyLines|renderNodeModalViewport|buildAutoScrollViewport|..." internal/tui/model.go` -> PASS (located shared renderer, helper, and mode routing points).
2. `nl -ba internal/tui/model.go | sed -n '11856,12140p'` -> PASS (captured task-form section helper + shared sizing helpers).
3. `nl -ba internal/tui/model.go | sed -n '12120,12340p'` -> PASS (captured task-form remaining sections + task-info section builder start).
4. `nl -ba internal/tui/model.go | sed -n '12276,12428p'` -> PASS (captured task-info section order/details for parity comparison).
5. `nl -ba internal/tui/model.go | sed -n '13608,13676p'` -> PASS (captured add/edit node modal routing through shared renderer).
6. `rg -n "..." internal/tui/model_test.go` + `nl -ba` targeted slices -> PASS (captured relevant coverage lines).
7. `rg -n "FR-003|modal parity|..." PLAN.md COLLAB_VECTOR_MCP_E2E_WORKSHEET.md` -> PASS (captured tracking context lines).
8. `just test-pkg ./internal/tui` -> PASS (`ok github.com/hylla/tillsyn/internal/tui (cached)`).

## Acceptance Criteria Checklist
1. Inspect `internal/tui/model.go` for shared renderer usage and sectioned helpers: **PASS**
2. Run `just test-pkg ./internal/tui`: **PASS**
3. If tests fail/runtime error occurs, run Context7 before remediation notes: **PASS (N/A, no failures/runtime errors)**
4. Produce findings with severity and exact `file:line` references: **PASS**
5. Include command evidence and pass/fail summary: **PASS**
6. Objective verification:
   - task/project edit modals use same full-screen shared modal frame/render path as info modal: **PASS**
   - task edit content follows same section order/style as task info (read vs edit state): **PASS**
   - no obvious TUI regression from this change in package tests: **PASS**

## Architecture-Boundary Compliance Note
- Read-only QA lane respected. No production code edits were made.
- Only allowed write was performed to this worklog file.

## Context7 Compliance Note
- Failure-triggered Context7 requirement was not activated because no tests/runtime commands failed.
- No remediation proposal requiring additional API research was needed in this pass.

## Risks / Open Questions
1. Should we add one explicit parity test that compares ordered section headers/tokens between task-info and edit-task overlays to prevent future drift?
2. Tracking naming appears to use FR-002/FX-002 for this parity wave in worksheet entries (`COLLAB_VECTOR_MCP_E2E_WORKSHEET.md:124`, `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md:131`, `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md:138`); confirm whether FR-003 should be recorded as a distinct row.
