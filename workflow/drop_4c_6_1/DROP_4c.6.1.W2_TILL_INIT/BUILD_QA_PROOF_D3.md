# W2.D3 — BUILD-QA-PROOF Verdict

**Date:** 2026-05-13
**Reviewer:** go-qa-proof-agent (opus)
**Overall verdict:** PASS

## Acceptance Bullet Coverage

### Bullet 1 — replaces single-select with picker_multi.go component from internal/tui/components; all three groups selectable, no disabled rows

- **Quote:** "`initTUIModel` replaces its single-select group-cursor step with the `picker_multi.go` component from `internal/tui/components`. All of `["gen", "go", "fe"]` are selectable (no disabled rows in the new model)."
- **Evidence:**
  - `cmd/till/init_cmd.go:27` adds the import `github.com/evanmschultz/tillsyn/internal/tui/components`.
  - `cmd/till/init_cmd.go:142` declares `groupPicker components.PickerMultiModel` (concrete exported type, not `tea.Model`).
  - `cmd/till/init_cmd.go:170` constructs the picker via `components.NewPickerMulti(allowedInitGroups)`.
  - `cmd/till/init_cmd.go:62` `allowedInitGroups = []string{"gen", "go", "fe"}` — exactly the three groups.
  - `internal/tui/components/picker_multi.go` carries no Disabled / Enabled concept — every row is selectable.
- **Verdict:** PASS

### Bullet 2 — Space toggles, Enter confirms, Esc cancels

- **Quote:** "Space-bar (or equivalent per `picker_multi.go` API) toggles individual group selection. Enter confirms the selection."
  - Sub-criterion (Specify): "Multi-select: space toggles, Enter confirms, Esc cancels."
- **Evidence:**
  - `internal/tui/components/picker_multi.go:58-77` implements Space-toggle, Enter-confirm, Escape-cancel semantics inside the component's Update switch.
  - `cmd/till/init_cmd.go:232-233` dispatches `msg` through `m.groupPicker.Update(msg)` on the group step, so Space / Enter / Esc keypresses reach the component.
  - `cmd/till/init_cmd.go:234-238` reads `groupPicker.Done()` + `groupPicker.Cancelled()` and routes Esc-cancel to `initTUIStepCancelled` + `tea.Quit`.
- **Verdict:** PASS

### Bullet 3 — finalPayload.Groups = selected slice; minimum 1 enforced with inline hint that refuses to advance

- **Quote:** "`finalPayload.Groups` is set to the slice of all selected group names (minimum one required — model rejects empty selection with an inline hint and refuses to advance)."
- **Evidence:**
  - `cmd/till/init_cmd.go:225-231` intercepts Enter on the group step BEFORE dispatching to the picker: if `len(m.groupPicker.Selected()) == 0`, sets `m.emptyHint = "select at least one group (space to toggle)"` and returns `(m, nil)` — step stays on `initTUIStepGroup`.
  - `cmd/till/init_cmd.go:239` assigns `m.finalPayload.Groups = m.groupPicker.Selected()` once the picker reports `Done()` without cancellation.
  - `cmd/till/init_cmd.go:269-273` renders `m.emptyHint` below the picker view on the group step when non-empty.
  - Test `cmd/till/init_cmd_test.go:1468-1492` (`min1_empty_selection_refuses_advance`) verifies: deselect gen, press Enter — step remains `initTUIStepGroup`, `emptyHint` is non-empty, re-selecting gen + Enter then advances to `initTUIStepDone` and `emptyHint` clears.
- **Verdict:** PASS

### Bullet 4 — default selection is ["gen"] — first row pre-selected, one Enter accepts it

- **Quote:** "The default selection is `["gen"]` — first row pre-selected so one Enter accepts the default immediately."
- **Evidence:**
  - `cmd/till/init_cmd.go:170-171` constructor sends a single synthetic `tea.KeyPressMsg{Code: tea.KeySpace}` to the picker after construction; since `PickerMultiModel` starts with `cursor == 0` and `allowedInitGroups[0] == "gen"`, this toggles gen to selected.
  - Test `cmd/till/init_cmd_test.go:1435-1445` (`default_gen_preselected`) verifies: advance to group step, press Enter once — step is `initTUIStepDone` and `Groups == ["gen"]`.
- **Verdict:** PASS

### Bullet 5 — View renders multi-select list with checked/unchecked rows visible

- **Quote:** "The TUI model's View renders a multi-select group list (checked/unchecked rows visible)."
- **Evidence:**
  - `cmd/till/init_cmd.go:267` View on `initTUIStepGroup` prints the header `"Agent groups (j/k to move, space to toggle, enter to confirm, esc to cancel)\n\n"`.
  - `cmd/till/init_cmd.go:268` calls `m.groupPicker.View()`, which (per `internal/tui/components/picker_multi.go:83-99`) renders each row as `"> [x] item"` or `"  [ ] item"`.
- **Verdict:** PASS

### Bullet 6 — dead code removed (initTUIGroupRows, initTUIGroupRow, Disabled field, nextEnabledGroupRow, prevEnabledGroupRow, groupCursor)

- **Quote:** "Dead code removed: `initTUIGroupRows []initTUIGroupRow`, `initTUIGroupRow` struct (including `Disabled bool` field), `nextEnabledGroupRow`, `prevEnabledGroupRow`, `groupCursor int` — all replaced by the `picker_multi.go` component."
- **Evidence:** `rg -n "initTUIGroupRow|nextEnabledGroupRow|prevEnabledGroupRow|groupCursor|initTUIGroupRows" cmd/till/` returns ZERO matches in the package. Every symbol is removed from both `init_cmd.go` and `init_cmd_test.go`.
- **Verdict:** PASS

### Bullet 7 — TUI model tests via teatest_v2 pattern verify Done/Cancelled/Payload

- **Quote:** "Tests: `teatest_v2` pattern (per existing `init_cmd_test.go` conventions) drives model directly to verify Done/Cancelled/Payload state."
- **Evidence:**
  - `cmd/till/init_cmd_test.go:1409-1493` adds `TestInitTUIModel_GroupMultiSelect` with three subtests (`default_gen_preselected`, `multi_select_go_and_fe`, `min1_empty_selection_refuses_advance`).
  - The tests drive `Update(...)` directly with synthetic `tea.KeyPressMsg` values and inspect `m.step`, `m.finalPayload.Groups`, and `m.emptyHint`. This is the same direct-drive idiom used elsewhere in the file; mid-walk state inspection is exactly why the Specify text says "drive the model directly" — the doc-comment at line 1407 calls this out explicitly.
- **Verdict:** PASS

### Bullet 8 — CONSUMER-TIE: run --json multi-group payload exercises the path without entering the TUI

- **Quote:** "CONSUMER-TIE supplement: `run(..., '--json', '{"name":"x","groups":["go","fe"],"mcp":false}')` exercises the multi-group payload path without entering the TUI; this is the JSON-mode mirror of D3's TUI multi-select."
- **Evidence:**
  - `cmd/till/init_cmd_test.go:1501-1517` adds `TestRunInit_JSONMode_MultiGroup`, which invokes `run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", `{"name":"x","groups":["go","fe"],"mcp":false}`}, ...)` and asserts no error + "Init" appears in stdout.
- **Verdict:** PASS

### Bullet 9 — mage test-pkg ./cmd/till and mage ci green

- **Quote:** "`mage test-pkg ./cmd/till` passes; `mage ci` green."
- **Evidence:** Locally re-ran `mage test-pkg ./cmd/till` — `tests: 313, passed: 313, failed: 0`. Builder also reported mage ci green in spawn brief.
- **Verdict:** PASS

### Specify-Side Sub-Criterion — field is `components.PickerMultiModel` (concrete, NOT tea.Model)

- **Quote:** "`initTUIModel` contains a field of type `components.PickerMulti` (or equivalent exported type from `internal/tui/components` — verify name via LSP after W5 ships)."
- **Evidence:** `cmd/till/init_cmd.go:142` — `groupPicker components.PickerMultiModel`. The component type at `internal/tui/components/picker_multi.go:18` is `PickerMultiModel`, a value type whose Update returns `(PickerMultiModel, tea.Cmd)` — deliberately NOT satisfying `tea.Model` per R10 locked decision (RiskNote line 184).
- **Verdict:** PASS

### RiskNote — no `var _ tea.Model = (*components.PickerMultiModel)(nil)` written

- **Quote:** "Builder does NOT write `var _ tea.Model = (*components.PickerMulti)(nil)`."
- **Evidence:** `rg "tea.Model.*PickerMulti"` returns no matches in cmd/till; no such interface assertion exists.
- **Verdict:** PASS

### RiskNote — picker component does NOT return tea.Quit

- **Quote:** "W5 components must NOT return `tea.Quit` (kills parent TUI)."
- **Evidence:** `internal/tui/components/picker_multi.go:48` doc-comment "Update NEVER returns tea.Quit" and lines 50-78 confirm Update returns either `(m, nil)` or `(m, nil)` for every branch — no `tea.Quit` path. The parent (`cmd/till/init_cmd.go:236-237, 245`) is the ONLY caller that returns `tea.Quit`, after reading `Done()` + `Cancelled()`.
- **Verdict:** PASS

## NITs

None.

## Verdict rationale

Every acceptance criterion has direct file:line evidence in the diff and tests. The dead-code removal is total (zero `rg` hits across `cmd/till/`). The picker integration uses the concrete `PickerMultiModel` value type — neither a `tea.Model` interface nor a `tea.Quit` path — matching the R10 locked decision and the W5 component contract. The Enter-intercept ordering is correct: empty-selection check happens BEFORE dispatching to the picker (line 225), so the picker never sees an Enter that would mark it Done with empty Selected(). The constructor's one-Space-press seed is the load-bearing trick for default `["gen"]` and is covered by `default_gen_preselected`. Local `mage test-pkg ./cmd/till` re-run confirms 313/313 PASS. No NITs, no missing evidence — verdict is PASS.
