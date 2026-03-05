# VEC_FIX_PLAN_P1_INFO_MODAL_AUDIT

Timestamp (UTC): 2026-03-05T03:25:02Z
Lane: P1 (read-only audit)
Scope audited: `internal/tui/model.go`, `internal/tui/model_test.go`

## Commands Run and Outcomes
- `pwd && ls` -> verified repo root and expected files present.
- `date -u +"%Y-%m-%dT%H:%M:%SZ"` -> captured audit timestamp.
- `rg -n "modeTaskInfo|modeThread|renderModeOverlay|openTaskInfo|..." internal/tui/model.go` -> located modal routing, task-info lifecycle, comment loading/rendering points.
- `rg -n "TaskInfo|comments|thread|modeTaskInfo|modeThread" internal/tui/model_test.go` -> located behavioral coverage.
- `nl -ba internal/tui/model.go | sed -n '1390,1495p'` and `sed -n '1690,1815p'` -> confirmed `View()` routing and overlay composition.
- `nl -ba internal/tui/model.go | sed -n '11740,12570p'` -> captured task-info lifecycle + full render block + comments section.
- `nl -ba internal/tui/model.go | sed -n '6180,6505p'` -> captured mode key handling for thread/task-info transitions.
- `nl -ba internal/tui/model.go | sed -n '11390,11670p'` -> captured help/interaction contract for task-info and thread.
- `nl -ba internal/tui/model.go | sed -n '10380,10470p'` -> captured scope-level comment target mapping.
- `nl -ba internal/tui/model_test.go | sed -n '1035,1105p'` and `sed -n '1238,1312p'` and `sed -n '1360,1445p'` -> captured tests for phase mapping, project details flow, and task-info comment preview.

## Findings by Severity

### High
1. Split read surfaces create divergence risk between task-info and thread-details UX.
- Evidence:
  - `View()` hard-routes `modeThread` and `modeDescriptionEditor` outside overlay flow: `internal/tui/model.go:1417-1422`.
  - `modeTaskInfo` is rendered inside `renderModeOverlay` with a separate, long rendering path: `internal/tui/model.go:12375-12555`.
  - Closing description editor has branching restore logic for both `modeThread` and `modeTaskInfo`: `internal/tui/model.go:2884-2918`.
- Why this matters: two parallel “details/read” paths increase drift risk for keyboard help, markdown behavior, and future node-type expansion.
- Recommendation: introduce a shared details surface model (target + markdown + comment policy + actions), then keep task/thread as thin wrappers.

### Medium
2. No dedicated info overlay exists for project-level nodes; project details rely on thread-details mode.
- Evidence:
  - Overlay switch has `modeTaskInfo` but no project/branch/phase/subphase-specific info modes: `internal/tui/model.go:12301-13360`.
  - Scope types are mapped for comments (`project/branch/phase/subphase/task/subtask`), but this is target mapping, not per-node info overlay rendering: `internal/tui/model.go:10397-10414`.
  - Project details UX is verified in thread mode tests (`thread-project`, `e` to open details): `internal/tui/model_test.go:1262-1284`.
- Why this matters: users see task-style info modal for work items but project details are in a different interaction model.
- Recommendation: define one node-details contract and bind both project/work-item details to it.

3. Task-info comments are preview-only and intentionally truncated.
- Evidence:
  - Preview cap constant = 5: `internal/tui/model.go:160-161`.
  - Task-info renders recent preview and hides older count: `internal/tui/model.go:12481-12503`.
  - Full comments path is via thread shortcut from task-info (`c`): `internal/tui/model.go:6462-6463`, help text `internal/tui/model.go:11507`.
- Why this matters: long conversations are not inspectable in the info modal itself.
- Recommendation: keep preview in modal, but add explicit “open full thread” affordance line near comment section (already partially implied by footer/help).

### Low
4. Coverage is good for preview presence and thread return path, but missing targeted assertions for preview truncation/error branch.
- Evidence:
  - Positive preview coverage exists: `internal/tui/model_test.go:1374-1425`.
  - Task-info -> thread phase target mapping and return coverage exists: `internal/tui/model_test.go:1057-1089`.
  - No scoped test found for `taskInfoCommentsError` rendering branch at `internal/tui/model.go:12476-12478` or hidden-count branch at `internal/tui/model.go:12501-12503`.
- Recommendation: add narrow tests for comment-load error and >5 comment truncation behavior when implementation lane starts.

## 1) File:Line Map: How Task Info Modal Is Rendered
- Open action from board keymap (`i`): `internal/tui/model.go:5976-5983`.
- Task-info lifecycle setup/teardown:
  - open/init: `internal/tui/model.go:11795-11818`
  - close/reset: `internal/tui/model.go:11820-11833`
  - history backtrack: `internal/tui/model.go:11906-11931`
- View routing:
  - `modeTaskInfo` uses overlay path, not full-screen replacement: `internal/tui/model.go:1417-1422`, `1715-1718`, `1782-1788`.
- Overlay render block:
  - `modeTaskInfo` body: `internal/tui/model.go:12375-12555`
  - details viewport construction/sync: `internal/tui/model.go:11861-11881`, render use `12413-12418`
- Interaction controls in task-info mode:
  - scrolling/nav/open subtask/back/shortcuts: `internal/tui/model.go:6380-6480`

## 2) Comments List in Task Info: Present vs Omitted
- Present now in task-info modal:
  - load comments: `internal/tui/model.go:11768-11793`
  - render section + summary/body preview: `internal/tui/model.go:12475-12500`
  - tested explicitly: `internal/tui/model_test.go:1374-1421`
- Omitted/limited behavior:
  - only recent `5` previewed; older comments collapsed to `+N older comments`: `internal/tui/model.go:160-161`, `12481-12503`
  - no evidence in-scope that comments were removed; current behavior is preview + thread handoff.

## 3) File:Line Map: Node-Type-Specific Info Overlays
- Task/work-item info overlay (`branch/phase/subphase/task/subtask` as work items):
  - single modal path, kind shown inline: `internal/tui/model.go:12375-12407` (includes `kind:` at `12404`).
- Scope-to-comment target mapping across node levels:
  - `project/branch/phase/subphase/task/subtask`: `internal/tui/model.go:10397-10414`.
- Project-specific info surface in this scope:
  - no `modeProjectInfo` overlay found in `renderModeOverlay` cases: `internal/tui/model.go:12301-13360`.
  - project details/read path exercised via thread mode: `internal/tui/model_test.go:1262-1284`.

## 4) Concrete Refactor Seams for Shared Display/Edit Component
1. View routing seam
- Current split: full-screen route for thread/editor vs overlay route for task-info (`internal/tui/model.go:1417-1422`, `1715-1718`).
- Refactor seam: route all details read surfaces through one renderer adapter (`renderDetailsSurface(...)`) with mode-specific shells.

2. Details content seam
- Task-info builds details, metadata, comments, resources in one large block (`internal/tui/model.go:12375-12555`).
- Refactor seam: extract pure section builders (`buildDetailsSections(target)`), then render in task-info/thread/project contexts.

3. Markdown/editor seam
- Description editor close flow already handles both thread/task-info (`internal/tui/model.go:2884-2918`).
- Refactor seam: keep this as shared edit gateway and normalize status/return behavior for all node targets.

4. Comment-source seam
- Task-info comments loaded via dedicated function (`internal/tui/model.go:11768-11793`) and refreshed on back-from-thread (`6282-6285`).
- Refactor seam: centralize comment retrieval/preview policy so preview/full-thread views consume same source with different limits.

## 5) Risks and Minimal Viable Path
### Risks
- Behavioral drift between task-info overlay and thread details mode (keymaps/help/status text).
- Regressions in back-navigation semantics (`esc` path stack vs thread back-mode).
- UX confusion if node types continue to split across modal vs thread details without explicit affordances.

### Minimal Viable Path
1. Extract a shared `details target` resolver and section builder from `modeTaskInfo` render block.
2. Keep existing modes but have both task-info and thread-details consume shared sections.
3. Add focused tests for comment truncation/error branch and node-type parity assertions.
4. Defer full modal unification until behavior parity is proven via tests.

## Recommendation Summary
- Keep current behavior stable for now.
- Implement shared section/render seams first (low-risk extraction).
- Then decide whether to unify task-info and thread-details into one primary details surface.
