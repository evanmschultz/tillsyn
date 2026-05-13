# W2.D3 — BUILD-QA-FALSIFICATION Verdict

**Date:** 2026-05-13
**Reviewer:** go-qa-falsification-agent (opus)
**Overall verdict:** PASS

## Attack Hypotheses Tested

### H1 — Pre-select via Space press hits the wrong index — REFUTED
- `internal/tui/components/picker_multi.go:28-33` — `NewPickerMulti` returns a zero-valued `cursor` field; Go's zero-value semantics put it at `0`.
- `cmd/till/init_cmd.go:170-171` — `NewPickerMulti(allowedInitGroups)` then immediately `picker.Update(tea.KeyPressMsg{Code: tea.KeySpace})`. Since `allowedInitGroups = ["gen", "go", "fe"]`, the cursor is at index 0 = "gen". One Space toggles selected[0]=true.
- Verified end-to-end by `TestRunInitTUI_AcceptsDefaultNameAndSelectsTillGo` (teatest_v2 driving the full program) at `cmd/till/init_cmd_test.go:201-254`, which asserts `Groups == ["gen"]` after a single Enter on the group step.

### H2 — Min-1 enforcement bypass — REFUTED
- `cmd/till/init_cmd.go:225-231` intercepts Enter BEFORE dispatching to the picker. If `m.groupPicker.Selected()` is empty, it sets `m.emptyHint` and returns without touching the picker (so the picker never marks itself Done).
- The interception happens unconditionally on every Enter — there is no exit-path that lets an empty-selection Enter reach the picker's Update.
- `TestInitTUIModel_GroupMultiSelect/min1_empty_selection_refuses_advance` at `cmd/till/init_cmd_test.go:1468-1492` verifies this directly: toggle gen off, press Enter, step stays `initTUIStepGroup`, `emptyHint` is non-empty.

### H3 — Esc cancels through picker — REFUTED
- `tea.KeyEsc == tea.KeyEscape` confirmed via `github.com/charmbracelet/ultraviolet`'s consts: `KeyEsc = KeyEscape = rune(ansi.ESC)`. So a `tea.KeyPressMsg{Code: tea.KeyEsc}` matches the picker's `case tea.KeyEscape:` branch (`picker_multi.go:73-75`).
- Picker sets `cancelled=true; done=true`. In `init_cmd.go:234-237`, `m.groupPicker.Done()` is true and `m.groupPicker.Cancelled()` is true, so `m.step = initTUIStepCancelled; return m, tea.Quit`.
- `runInitTUI` (line 330-332) then surfaces the cancellation as an error.
- Picker-level coverage: `TestPickerMultiModel_Cancel` at `internal/tui/components/picker_multi_test.go:165-189`.

### H4 — Dead code grep — REFUTED (zero hits)
- `git grep "initTUIGroupRow\|initTUIGroupRows\|nextEnabledGroupRow\|prevEnabledGroupRow\|groupCursor" cmd/till/` returns exit code 1 (no matches). All dead-code identifiers fully removed.

### H5 — Default `["gen"]` — REFUTED
- `TestRunInitTUI_AcceptsDefaultNameAndSelectsTillGo` verifies `Payload().Groups == ["gen"]` via teatest_v2 after Enter-Enter (name accept + group confirm). Also covered by `TestInitTUIModel_GroupMultiSelect/default_gen_preselected`.

### H6 — Multi-select ordering — REFUTED
- `picker_multi.go:104-115` `Selected()` iterates `m.items` (source list) and emits selected entries in source order, NOT toggle order. So toggling fe before go would STILL yield `["go", "fe"]` per source order — the contract is "list order, regardless of toggle order".
- `TestInitTUIModel_GroupMultiSelect/multi_select_go_and_fe` asserts the order: `got[0] != "go" || got[1] != "fe"` (line 1463). Test passes.

### H7 — `tea.Quit` absence in picker_multi — REFUTED (and orchestrator hypothesis mis-stated)
- The spec actually requires that `picker_multi.go` (the COMPONENT) MUST NOT return `tea.Quit` — verified by inspection: `picker_multi.go:50-78` returns only `(m, nil)` from every branch.
- `init_cmd.go` legitimately uses `tea.Quit` at lines 204, 237, 245 — these are PARENT model `tea.Quit` calls to terminate the entire `tea.Program` after the walk completes/cancels. This is correct per R10 locked decision (parent issues tea.Quit, component returns accessors only). The orchestrator's "grep init_cmd.go for tea.Quit. Zero hits." is mis-stated — the rule is about the component, not the parent.

### H8 — `var _ tea.Model` on picker usage — REFUTED (zero hits)
- `git grep "var _ tea.Model" cmd/till/init_cmd.go` returns exit code 1 (no matches). Builder correctly did NOT write the interface assertion (per RiskNotes line 184).

### H9 — CONSUMER-TIE supplement uses run() end-to-end — REFUTED
- `cmd/till/init_cmd_test.go:1501-1517` `TestRunInit_JSONMode_MultiGroup` invokes `run(context.Background(), []string{"--app", "tillsyn-init", "init", "--json", '{"name":"x","groups":["go","fe"],"mcp":false}'}, &out, io.Discard)`. NOT a direct `runInitJSON` call — it routes through cobra dispatch like a real CLI invocation.

### H10 — JSON unmarshal multi-group + validation — REFUTED
- `TestRunInit_JSONMode_MultiGroup` passes; the payload `{"groups":["go","fe"]}` unmarshals into `Groups = ["go","fe"]` and `validateInitPayload` accepts both entries (both are in `allowedInitGroups`).
- `mage test-func ./cmd/till TestRunInit_JSONMode_MultiGroup` PASS confirmed.

### H11 — Cross-droplet bleed — REFUTED
- `git diff --stat` shows ONLY `cmd/till/init_cmd.go` (+59/-101) and `cmd/till/init_cmd_test.go` (+158/-27). No edits outside the declared `paths`.

### H12 — `teatest_v2` pattern adoption — EXHAUSTED, no counterexample found (with a NIT)
- Spec line 166 says "teatest_v2 pattern (per existing init_cmd_test.go conventions)".
- `TestInitTUIModel_GroupMultiSelect` (the new multi-select test) bypasses teatest by calling `m.Update(msg)` directly. The doc-comment at lines 1407-1408 justifies this: "These tests drive the model directly (no teatest program) so they can inspect mid-walk state (emptyHint, step) without relying on WaitFinished."
- Counterpoint: `TestRunInitTUI_AcceptsDefaultNameAndSelectsTillGo` and `TestRunInitTUI_SelectsFeRow` DO use teatest_v2 and exercise multi-select indirectly (default + single-fe-select). So the teatest_v2 coverage requirement is satisfied at the integration level; the direct-Update tests add per-subtest mid-walk-state coverage that teatest_v2 cannot inspect (emptyHint mid-walk requires reading mid-state, not WaitFinished's terminal state).
- Net: NOT a counterexample — the spec word "pattern" is interpreted as "consistent with the test-file's conventions"; teatest_v2 coverage exists via the two RunInitTUI_* tests. See NIT-1 below for the consistency concern.

### H13 — State after Esc on group step — REFUTED
- Picker's `Cancelled() == true; Done() == true; Selected() returns nil` (`picker_multi.go:104-107` short-circuits to nil when cancelled).
- The parent (`init_cmd.go:234-237`) reads `Cancelled()` and transitions to `initTUIStepCancelled`. The parent's own `Cancelled()` accessor (line 290-292) then returns true.
- `runInitTUI` (line 330-332): if `final.Cancelled()` then return `errors.New("till init: cancelled by user")`. The cancel path is correctly surfaced to the caller.

## Unmitigated Counterexamples

None.

## NITs

### NIT-1 — Group-step Esc not directly covered via teatest_v2
The `TestRunInitTUI_EscCancelsWalk` test (line 327-357) sends Esc DURING THE NAME STEP (before advancing to the group step). The group-step Esc path — which depends on the picker's Cancelled()/Done() accessor flow being correctly surfaced through `init_cmd.go:234-237` — has no dedicated end-to-end test. Picker-level Esc is covered by `TestPickerMultiModel_Cancel`, and the parent's wiring is straightforward, so this is a low-risk NIT, but a `TestRunInitTUI_EscCancelsGroupStep` would close the gap. (Suggest: Enter name → Esc group → expect Cancelled() true.)

### NIT-2 — Multi-select test bypasses teatest_v2 contract
The spec line 166 says "teatest_v2 pattern". `TestInitTUIModel_GroupMultiSelect` deliberately drives Update directly to inspect `emptyHint` mid-walk. The doc-comment justifies it, but a hybrid form (teatest_v2 for terminal-state assertions + direct Update for mid-state probes) would be more faithful to spec language and would also exercise the cobra → tea.Program → model dispatch chain on the multi-select path. The current shape is functionally correct; this is purely a methodology adherence NIT.

## Verdict rationale

All 13 attack hypotheses either REFUTED or EXHAUSTED with no counterexample. The two NITs are low-severity test-coverage observations, not functional defects.

Key positive findings:
- Builder correctly avoided returning `tea.Quit` from the component (per R10 W5-fals-FF2 disposition).
- Min-1 enforcement is placed BEFORE picker dispatch (line 225-231), preventing the deadlock case the comment explicitly calls out ("once the picker marks itself Done it ignores further input, which would deadlock the walk").
- The pre-selection via post-construction Space press is a reasonable workaround given the picker API does not expose a "pre-select" constructor parameter. The cursor's zero-value init at 0 makes the post-construction Space deterministic.
- All declared dead code is fully removed (zero hits on the grep).
- `mage ci` passes; `mage test-pkg ./cmd/till` shows 313/313 passing.
- Diff scope is strictly within declared `paths`: `cmd/till/init_cmd.go` + `cmd/till/init_cmd_test.go` only.

Recommend: PASS. Address NIT-1 and NIT-2 if/when a follow-up build touches this file; not blocking.
