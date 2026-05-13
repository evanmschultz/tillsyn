# W2.D4 — BUILD-QA-FALSIFICATION Verdict

**Date:** 2026-05-13
**Reviewer:** go-qa-falsification-agent (opus)
**Overall verdict:** PASS

## Attack Hypotheses Tested

### H1 — Esc-vs-n disambiguation REFUTED

Outer dispatch at `cmd/till/init_cmd.go:255-273` intercepts `tea.KeyEsc` BEFORE forwarding to `m.mcpConfirm.Update(msg)`:

- Lines 259-262: `if kp, ok := msg.(tea.KeyPressMsg); ok && kp.Code == tea.KeyEsc { m.step = initTUIStepCancelled; return m, tea.Quit }`.
- `n` / `N` flows through to `m.mcpConfirm.Update(msg)` at line 263 because its `key.Code` is `'n'` / `'N'`, not `tea.KeyEsc`. Inside confirm.go line 53, `'n', 'N'` sets `m.cancelled=true; m.done=true`.
- Back at outer dispatch line 264-272: `m.mcpConfirm.Done()` is true, so `mcpYes := m.mcpConfirm.Confirmed()` reads false (confirm.go `Confirmed()` returns the `confirmed` field only, NOT `!cancelled`). `m.finalPayload.MCP = &false`, step → `initTUIStepDone`.

Confirmed by `TestInitTUIModel_MCPStep/n_no` (init_cmd_test.go:1618-1631) which asserts `Done()==true`, `Cancelled()==false`, `MCPRegistration()==false` after `n`. And `TestInitTUIModel_MCPStep/esc_cancel` (lines 1633-1643) asserts `Cancelled()==true`, `Done()==false` after Esc. 4/4 sub-tests PASS via `mage test-func ./cmd/till TestInitTUIModel_MCPStep`.

### H2 — NewConfirm API signature REFUTED

`internal/tui/components/confirm.go:28` declares `func NewConfirm(prompt string, defaultYes bool) ConfirmModel`. Call site `init_cmd.go:185` uses `components.NewConfirm("Register MCP server in .mcp.json?", true)` — matches signature, second arg `true` is defaultYes. confirm.go:57-58 enforces default-YES on Enter: `if m.defaultYes { m.confirmed = true }`. Spec match.

### H3 — Confirmed() vs Cancelled() accessors REFUTED

confirm.go:82-85 exposes both accessors independently:
- `Confirmed() bool { return m.confirmed }`
- `Cancelled() bool { return m.cancelled }`

After Enter with defaultYes=true: confirm.go:56-58 sets `confirmed=true`; `cancelled` stays false. So `Confirmed()==true && Cancelled()==false`. After `n`: `confirmed=false && cancelled=true`. Distinct boolean fields, not derived from each other. Verified by reading the struct (confirm.go:18-24) and Update switch (44-67).

Note: confirm.go's `Cancelled()` reports the SUB-COMPONENT'S notion of cancel (merges n/N/Esc), but the OUTER initTUIModel's `Cancelled()` (init_cmd.go:319-321) checks `m.step == initTUIStepCancelled` — set ONLY by Esc-intercept (lines 211, 244, 260), NEVER from confirm.Cancelled(). The asymmetric mapping is intentional and correct.

### H4 — payload.MCPRegistration() usage in consumers REFUTED

`git grep -n "payload\.MCP" -- cmd/till/init_cmd.go`:
```
cmd/till/init_cmd.go:509: registerMCPJSON(destDir, payload.MCPRegistration())
cmd/till/init_cmd.go:524: if payload.MCPRegistration() {
```

Both consumer read-sites use the accessor. The only bare `payload.MCP` writes are:
- Line 39 — struct field declaration on `initJSONPayload`.
- Line 269 — `m.finalPayload.MCP = &mcpYes` — the PRODUCER write inside Update where TUI just resolved user input. Per spec this is the write site, not a consumer read.

No bare-pointer dereferences in consumer paths. Spec satisfied.

### H5 — Hardwire removal REFUTED

Diff confirms removal of the old `mcpFalse := false; m.finalPayload.MCP = &mcpFalse` block at the group→done transition. Replacement is the new MCP-step branch.

`git grep -n "mcpFalse" -- cmd/till/init_cmd.go`: no hits. Only `mcpFalse` left in the repo for this file is at `init_cmd_test.go:51` inside the `TestInit_BareInvocation_ReturnsTUIStubError` scripted-program stub — that's a test-only synthetic payload (the test bypasses the real Update by injecting a pre-completed model), NOT a production hardwire. Acceptable.

### H6 — Step transition order REFUTED

State machine in init_cmd.go Update (lines 202-279):

- `initTUIStepName`: Enter → `initTUIStepGroup` (line 220); Esc → `initTUIStepCancelled` (line 212).
- `initTUIStepGroup`: picker.Done & !Cancelled → `initTUIStepMCP` (line 250); picker.Cancelled → `initTUIStepCancelled` (line 245).
- `initTUIStepMCP`: Esc → `initTUIStepCancelled` (line 260); confirm.Done → `initTUIStepDone` (line 270).
- `initTUIStepDone` / `initTUIStepCancelled`: `default` branch ignores further input (line 275-277).

Order matches spec: name → group → MCP → done. Esc at MCP goes to `Cancelled`, NOT back to group (the spec asks "what about Esc at MCP step — does it go to done-as-cancelled OR back to group?" → answer: cancelled-terminal, correct).

### H7 — JSON-mode skipping TUI REFUTED

`runInitJSON` (line 376-393) parses payload, validates, then calls `runInitPipeline(stdout, opts, parsed)`. No TUI invocation. The pipeline reads `payload.MCPRegistration()` at line 509 and 524 — all three paths (true, false, nil-default) route through the accessor. `TestRunInit_JSONMode_MCPPaths` (lines 1661-1720) exercises all 3 cases via run() end-to-end. 4/4 PASS confirmed.

### H8 — Default-YES persistence REFUTED

Trace through Enter×3:
1. Enter on name step: `value = strings.TrimSpace(m.nameInput.Value())` (line 215) — textinput pre-populated with `def = filepath.Base(cwd)` (line 165, 170). `m.finalPayload.Name = def`. Advance.
2. Enter on group step with gen pre-selected (newInitTUIModel line 178-179 sends Space at cursor=0): `m.groupPicker.Selected()` returns `["gen"]`. `m.finalPayload.Groups = ["gen"]`. Advance.
3. Enter on MCP step: confirm.go line 56-58 with defaultYes=true sets `confirmed=true`. Outer `m.mcpConfirm.Confirmed()` returns true. `m.finalPayload.MCP = &true`. Advance to `initTUIStepDone`.

Final payload: `{Name: <cwd-base>, Groups: ["gen"], MCP: &true}`, `MCPRegistration() == true`. Asserted by `TestRunInitTUI_AcceptsDefaultNameAndSelectsTillGo` (lines 255-263). PASS.

### H9 — tea.Quit absence (component vs parent) REFUTED

Production `tea.Quit` calls in init_cmd.go (4 sites): line 213 (name-Esc), 246 (group-cancel), 261 (MCP-Esc), 271 (MCP-Done). All inside the OUTER `initTUIModel.Update` parent. confirm.go's Update (lines 44-67) NEVER returns `tea.Quit` — its comment block at line 17 explicitly states "Update never returns tea.Quit." Production satisfies the per-R10 invariant.

### H10 — CONSUMER-TIE coverage REFUTED

`TestRunInit_JSONMode_MCPPaths` lines 1661-1720 covers:
- `mcp_true`: payload `{"mcp":true}` → `MCPRegistration()=true` → registerMCPJSON path runs.
- `mcp_false`: payload `{"mcp":false}` → `MCPRegistration()=false` → .mcp.json NOT created (asserted with `os.Stat` line 1693-1696).
- `no_mcp_key`: payload omits mcp → nil pointer → defaults true → .mcp.json IS created (asserted with `os.Stat` line 1715-1718).

All three drive run() end-to-end (cobra → runInitJSON → runInitPipeline → registerMCPJSON). 3/3 sub-tests PASS in mage run above.

### H11 — Pre-existing test updates REFUTED

Four pre-existing tests updated to step through MCP (diff confirmed):
- `TestRunInitTUI_AcceptsDefaultNameAndSelectsTillGo` (init_cmd_test.go:203-264): now sends Enter×3 (name, group, MCP), asserts `MCPRegistration()==true` line 261.
- `TestRunInitTUI_SelectsFeRow` (282-343): also sends an extra Enter for MCP confirm (line 326).
- `TestInitTUIModel_GroupMultiSelect/default_gen_preselected` (1456-1472): after group Enter, asserts step is `initTUIStepMCP` (line 1461-1463), then extra Enter to reach `initTUIStepDone`.
- `TestInitTUIModel_GroupMultiSelect/multi_select_go_and_fe` (1474-1498): same pattern, extra Enter for MCP.
- `TestInitTUIModel_GroupMultiSelect/min1_empty_selection_refuses_advance` (1500-1529): same pattern.

None accidentally skip the MCP step — every test that completes the walk now explicitly sends an Enter at MCP step. Full pkg PASS at 321/321 includes all of these.

### H12 — YAGNI REFUTED

D4 diff additions are minimal and spec-bounded:
- One new `initTUIStep` enum value (`initTUIStepMCP`).
- One new struct field (`mcpConfirm components.ConfirmModel`).
- One NewConfirm call in `newInitTUIModel`.
- One new Update branch (~19 lines).
- One new View branch (3 lines).
- Doc-comment refresh on `initTUIStep` and the model struct.

No abstractions added, no helper types, no premature interfaces. Spec-bounded.

### H13 — Cross-droplet bleed REFUTED

`git status --porcelain`:
```
M cmd/till/init_cmd.go
M cmd/till/init_cmd_test.go
?? workflow/drop_4c_6_1/DROP_4c.6.1.W2_TILL_INIT/BUILDER_WORKLOG.md
```

Confined to declared `paths` for D4. No bleed into other packages or unrelated files.

### H14 — Hermeticity (W1.D1 lesson) REFUTED

D4 diff (`git diff -- cmd/till/init_cmd.go`) introduces zero `os.UserHomeDir()` and zero `os.Getenv("HOME")` calls. The pre-existing `os.UserHomeDir()` at line 854 (inside `registerMCPJSON`'s till-binary fallback) was already present pre-D4 — not introduced by this droplet. Verified by absence of any UserHomeDir lines in the diff hunks.

## Unmitigated Counterexamples

None. All 14 attacks REFUTED with concrete evidence.

## NITs

None. The implementation is minimal, spec-aligned, well-commented, and the comment block at init_cmd.go:256-258 explicitly explains the Esc-vs-n disambiguation choice (the most subtle aspect of the implementation). Test coverage covers Enter-YES, n-NO, Esc-cancel as distinct sub-tests.

## Verdict rationale

The implementation correctly:
1. Adds the `initTUIStepMCP` step between group and done with the closed enum extended properly.
2. Intercepts Esc at outer dispatch BEFORE forwarding to confirm.go, cleanly separating walk-cancel (Esc) from per-step NO (n/N).
3. Uses `components.NewConfirm("...", true)` for default-YES semantic verified against confirm.go:56-58.
4. Removes the old `MCP=false` hardwire entirely (diff confirms, only test-stub synthetic payload retains `mcpFalse := false`).
5. All consumer reads of `payload.MCP` go through `MCPRegistration()` accessor (only 2 consumer sites, both correct).
6. Drives full state-machine Enter×3 happy-path to `{Name: cwd-base, Groups: [gen], MCP: &true}`.
7. Three pre-existing TUI tests updated to step through MCP; three new MCP-step sub-tests added; three CONSUMER-TIE JSON-mode MCP paths covered.
8. No tea.Quit in confirm component (parent owns Quit per R10).
9. No new hermeticity violations.
10. No cross-droplet bleed.
11. 321/321 cmd/till tests PASS under `-race`.

PASS.
