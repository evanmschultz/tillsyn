# P2 Edit Modal Audit (Read-Only)

Timestamp (UTC): 2026-03-04T01:03:57Z

## Scope + Method

- Read scope used:
  - `internal/tui/model.go`
  - `internal/tui/model_test.go`
- Write scope used:
  - `worklogs/VEC_FIX_PLAN_P2_EDIT_MODAL_AUDIT.md` only
- No code changes were made.

## Commands Run

- `rg -n "create|edit|modal|form|..." internal/tui/model.go`
- `rg -n "create|edit|modal|form|..." internal/tui/model_test.go`
- `rg -n "mode(Add|Edit|Project|Task)|..." internal/tui/model.go internal/tui/model_test.go`
- `nl -ba internal/tui/model.go | sed -n '<target ranges>'`
  - key ranges: `50-92`, `96-154`, `484-548`, `2168-2235`, `2388-3370`, `5920-6065`, `6188-6395`, `7260-7610`, `7740-8115`, `8120-8248`, `11390-11620`, `12220-13680`
- `nl -ba internal/tui/model_test.go | sed -n '<target ranges>'`
  - key ranges: `820-940`, `1200-1315`, `1500-1760`, `2120-2185`, `2460-2575`, `8610-8795`

---

## 1) Create/Edit Flow Map

### 1.1 Modal modes and state model

- Input modes include dedicated create/edit modes for task/project plus related modal states (`modeAddTask`, `modeEditTask`, `modeAddProject`, `modeEditProject`, `modeDescriptionEditor`, etc.): `internal/tui/model.go:57-83`.
- Model carries separate task/project form state buckets (`formInputs` + `taskForm*` vs `projectFormInputs` + `projectForm*`): `internal/tui/model.go:497-536`.

### 1.2 Task + project create/edit entrypoints

- Board-key entrypoints:
  - New task -> `startTaskForm(nil)`: `internal/tui/model.go:5962-5964`.
  - Edit task -> `startTaskForm(&task)`: `internal/tui/model.go:5994-6001`.
  - New project -> `startProjectForm(nil)`: `internal/tui/model.go:5973-5975`.
  - Edit project -> `startProjectForm(&project)`: `internal/tui/model.go:6002-6009`.
- Command-palette mirrors these:
  - Task/project create/edit commands route to same starters: `internal/tui/model.go:8129-8131`, `internal/tui/model.go:8198-8205`, `internal/tui/model.go:8207-8215`.
- Quick actions also reuse task edit starter: `internal/tui/model.go:8470-8477`.

### 1.3 Node-type create/edit reuse (branch/phase/subphase/subtask)

- Non-project node forms are task-form presets, not separate form implementations:
  - Subtask preset: `internal/tui/model.go:2562-2570`.
  - Branch preset: `internal/tui/model.go:2573-2587`.
  - Phase/subphase preset: `internal/tui/model.go:2590-2609`.
- Command palette node actions feed those presets (`new-branch`, `new-phase`, `new-subphase`) and branch edit reuses task edit form: `internal/tui/model.go:8138-8178`.

### 1.4 Form initialization and description editor pattern

- Project form init + edit prefill in one function (`startProjectForm`): `internal/tui/model.go:2418-2455`.
- Task form init + edit prefill in one function (`startTaskForm`): `internal/tui/model.go:2457-2537`.
- Both forms treat description as markdown backed by separate `*FormDescription` string + compact display summary:
  - task: `internal/tui/model.go:3104-3111`
  - project: `internal/tui/model.go:3113-3120`
- Both enter shared fullscreen markdown editor via parallel handlers:
  - task: `internal/tui/model.go:2673-2693`
  - project: `internal/tui/model.go:2695-2715`
  - save target routing: `internal/tui/model.go:2860-2876`

### 1.5 Submit paths

- Create task + edit task submit branches in `submitInputMode`:
  - create path: `internal/tui/model.go:7779-7839`
  - edit path: `internal/tui/model.go:7863-7958`
- Add/edit project submit branch in same function:
  - `internal/tui/model.go:8038-8115`

### 1.6 Existing tests covering these flows

- Add/edit project happy path: `internal/tui/model_test.go:831-884`.
- Task metadata prefill + submit in edit form: `internal/tui/model_test.go:1505-1567`.
- Branch node create/edit routed via task form mode/kind/scope: `internal/tui/model_test.go:2125-2163`.
- Description editor routing and save/cancel for task/project forms:
  - task: `internal/tui/model_test.go:2481-2516`
  - project: `internal/tui/model_test.go:2518-2552`
- Thread details -> edit shortcut behavior (display-to-edit bridge) for task/project:
  - task: `internal/tui/model_test.go:1216-1260`
  - project: `internal/tui/model_test.go:1262-1296`

---

## 2) Duplicated Rendering/Stying Logic Blocking Shared Component Behavior

### 2.1 Duplicate modal event loops for task vs project forms

- Two separate mode handlers implement near-identical control flow (`esc`, `tab`, `shift+tab`, `enter`, clipboard, field update), with only field-specific branches different:
  - task: `internal/tui/model.go:7421-7521`
  - project: `internal/tui/model.go:7524-7564`
- Impact: behavior parity fixes require touching both branches, increasing drift risk.

### 2.2 Duplicate modal renderer branches for task vs project forms

- `renderModeOverlay` builds task and project form lines in separate branches with similar width/label/focus logic:
  - task render branch: `internal/tui/model.go:13271-13320`
  - project render branch: `internal/tui/model.go:13321-13343`
- Impact: styling changes (spacing/labels/hints/focus visuals) are duplicated.

### 2.3 Duplicate task/project description-editor bootstrap

- `startTaskDescriptionEditor` and `startProjectDescriptionEditor` are structurally identical except field source/target/path selectors:
  - task: `internal/tui/model.go:2673-2693`
  - project: `internal/tui/model.go:2695-2715`
- Impact: editor behavior updates must be duplicated.

### 2.4 Duplicate task/project form extraction and display sync helpers

- Value extraction duplicated:
  - task: `internal/tui/model.go:3033-3043`
  - project: `internal/tui/model.go:3074-3084`
- Description summary sync duplicated:
  - task: `internal/tui/model.go:3104-3111`
  - project: `internal/tui/model.go:3113-3120`
- Impact: normalization/sanitization/display changes fan out to multiple helpers.

### 2.5 Help text duplication for add/edit pairs

- Add/edit task help lines are duplicated with minimal wording difference: `internal/tui/model.go:11454-11473`.
- Add/edit project help lines are duplicated similarly: `internal/tui/model.go:11511-11524`.
- Impact: doc UX drifts easily and adds maintenance burden.

---

## 3) Proposed Reusable Data Model + Renderer Interface (Unified Display + Edit)

### 3.1 Proposed data model

```go
// NodeModalKind identifies modal families that share display/edit behavior.
type NodeModalKind string

const (
    NodeModalTask    NodeModalKind = "task"
    NodeModalProject NodeModalKind = "project"
)

// FieldSpec defines one editable row in a generic modal form.
type FieldSpec struct {
    Key           string
    Label         string
    Placeholder   string
    CharLimit     int
    UsesMarkdown  bool
    PickerAction  string // optional: "labels", "dependencies", "root_path", etc.
}

// FormSpec defines one create/edit form contract.
type FormSpec struct {
    Kind       NodeModalKind
    Fields     []FieldSpec
    AddTitle   string
    EditTitle  string
    AddHint    string
    EditHint   string
}

// FormState holds mutable state independent from task/project-specific storage fields.
type FormState struct {
    Mode       inputMode
    Focus      int
    Inputs     []textinput.Model
    MarkdownByKey map[string]string
    EditingID  string
}
```

### 3.2 Proposed interfaces

```go
// NodeFormAdapter maps domain data <-> generic form state and submit commands.
type NodeFormAdapter interface {
    Spec() FormSpec
    LoadCreate(m Model) FormState
    LoadEdit(m Model) (FormState, error)
    Submit(m Model, s FormState) (tea.Model, tea.Cmd)
    Cancel(m *Model, s FormState)
}

// NodeDetailsAdapter maps thread/task-info details display to a stable edit action.
type NodeDetailsAdapter interface {
    DetailsTitle(m Model) string
    DetailsLines(m Model, width int) []string
    StartEdit(m Model) (tea.Model, tea.Cmd)
}
```

### 3.3 Why this fits current architecture

- Current code already uses a shared skeleton in practice (task/project both use `newModalInput`, markdown editor bridge, `submitInputMode`, and `renderModeOverlay`): `internal/tui/model.go:2194-2205`, `internal/tui/model.go:2673-2715`, `internal/tui/model.go:7777-8115`, `internal/tui/model.go:13155-13343`.
- Node variants (branch/phase/subphase/subtask) already piggyback on task form presets, so an adapter/preset model aligns with existing behavior: `internal/tui/model.go:2562-2609`, `internal/tui/model.go:8138-8171`.

---

## 4) Minimal File-Level Change Plan + Acceptance Checks

### 4.1 Minimal change plan (within `internal/tui/model.go` + tests)

1. Add generic form spec/state scaffolding and adapter hooks in `model.go`.
2. Refactor task/project input event handling into one shared modal-form update function with adapter-specific callbacks.
3. Refactor task/project overlay rendering into one shared form renderer fed by `FormSpec` and current `FormState`.
4. Keep node-specific task presets (`branch/phase/subphase/subtask`) by mutating task-form adapter defaults only (no separate node form implementation).
5. Keep existing thread/task-info display behavior, but route details->edit trigger through one adapter-style entrypoint so display-to-edit is consistent for task/project.

### 4.2 Acceptance checks

- Functional parity checks (must continue passing):
  - `TestModelAddAndEditProject` (`internal/tui/model_test.go:831-884`)
  - `TestModelEditTaskMetadataFieldsPrefillAndSubmit` (`internal/tui/model_test.go:1505-1567`)
  - `TestModelCommandPaletteBranchLifecycleActions` (`internal/tui/model_test.go:2125-2163`)
  - `TestModelTaskDescriptionEditorFlow` (`internal/tui/model_test.go:2481-2516`)
  - `TestModelProjectDescriptionEditorSeedAndCancel` (`internal/tui/model_test.go:2518-2552`)
  - Thread details edit bridge tests (`internal/tui/model_test.go:1216-1296`)
- Add/adjust tests to assert shared modal renderer/event-handler parity between add/edit task and add/edit project paths.
- Run scoped test gate: `just test-pkg ./internal/tui`.

---

## Notes / Constraints Observed

- In this read-only scope, call sites for `startThreadEditFlow`/`startThread`/`renderThreadModeView` are visible, but their implementations are not present in `model.go`; behavior evidence was therefore taken from call sites + tests (`internal/tui/model.go:6244`, `internal/tui/model.go:6368`, `internal/tui/model.go:1421`, and `internal/tui/model_test.go:1216-1296`).
