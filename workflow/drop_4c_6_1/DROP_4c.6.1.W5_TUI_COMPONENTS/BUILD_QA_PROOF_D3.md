# W5.D3 — BUILD-QA-PROOF Verdict

**Date:** 2026-05-13
**Reviewer:** go-qa-proof-agent (opus)
**Overall verdict:** PASS WITH NITS

## Acceptance Bullet Coverage

### Bullet 1 — `textinput.go` compiles alongside `confirm.go` in `internal/tui/components`

- **Evidence:** `mage test-pkg ./internal/tui/components` → PKG PASS (0.00s); 28 tests, 28 passed, 0 failed (full scoped run captured this session). Package contains `confirm.go`, `confirm_test.go`, `progress.go`, `textinput.go`, `textinput_test.go`.
- **Verdict:** PASS

### Bullet 2 — `TextInputModel` is a Bubble Tea sub-component: `Update(tea.Msg) (TextInputModel, tea.Cmd)`, `View() string`, `Init() tea.Cmd` — NOT `tea.Model`. No `var _ tea.Model = (...)` assertion

- **Evidence:**
  - `internal/tui/components/textinput.go:42` — `func (m TextInputModel) Init() tea.Cmd` — concrete receiver, correct signature.
  - `internal/tui/components/textinput.go:53` — `func (m TextInputModel) Update(msg tea.Msg) (TextInputModel, tea.Cmd)` — concrete return tuple, NOT `tea.Model`.
  - `internal/tui/components/textinput.go:80` — `func (m TextInputModel) View() string` — concrete signature.
  - No `var _ tea.Model = ...` assertion present anywhere in `textinput.go` (visual scan of full file; only declarations are the struct + 6 methods).
- **Verdict:** PASS

### Bullet 3 — `Value()`, `Err()`, `Submitted()` accessors exist

- **Evidence:**
  - `internal/tui/components/textinput.go:89` — `func (m TextInputModel) Value() string { return m.inner.Value() }`
  - `internal/tui/components/textinput.go:93` — `func (m TextInputModel) Err() error { return m.err }`
  - `internal/tui/components/textinput.go:97` — `func (m TextInputModel) Submitted() bool { return m.submitted }`
- **Verdict:** PASS

### Bullet 4 — `textinput_test.go` passes: validation + submit matrix covered; no non-nil cmd returned

- **Evidence:**
  - `internal/tui/components/textinput_test.go:26` `TestTextInputModel_Validation` — table-driven, asserts `if cmd != nil { t.Errorf(...) }` at line 62-64.
  - `internal/tui/components/textinput_test.go:82` `TestTextInputModel_Submit` — three subtests: nil validator, failing validator, non-Enter key. Lines 89-90, 105-106 assert non-nil cmd is rejected for Enter paths.
  - `internal/tui/components/textinput_test.go:119` "non-Enter key does not submit" — line 123 explicitly tolerates inner-driven cmd for character keys (`_ = cmd`), which is correct: the no-non-nil-cmd contract applies to the model's own Enter path, not to delegated inner cmds for other keys.
  - `mage test-pkg ./internal/tui/components`: 28/28 PASS.
- **Verdict:** PASS

### Bullet 5 — Migration marker present in both files as file-level comment before `package components`

- **Evidence:**
  - `internal/tui/components/textinput.go:1` — `// MIGRATION TARGET: github.com/hylla-org/lykta` (line 1, before `package components` at line 7).
  - `internal/tui/components/textinput_test.go:1` — `// MIGRATION TARGET: github.com/hylla-org/lykta` (line 1, immediately before `package components` at line 2).
  - In `textinput.go` the migration marker is separated from `package components` by a blank line and the package doc comment block (lines 3-6). The marker is still the topmost file-level comment, before the package clause — matches the spec wording "file-level comment before `package components`."
- **Verdict:** PASS

### Bullet 6 — `mage test-pkg ./internal/tui/components` passes (full package)

- **Evidence:** `[PKG PASS] github.com/evanmschultz/tillsyn/internal/tui/components (0.00s)` — 28 tests, 28 passed, 0 failed, 0 skipped (captured this session).
- **Verdict:** PASS

## Special-Focus Verification

- **Migration marker IS file-level comment before `package components`:** PASS — both files line 1, before package clause.
- **Return type concrete `(TextInputModel, tea.Cmd)`:** PASS — `textinput.go:53`.
- **`tea.KeyPressMsg` (NOT `tea.KeyMsg`):** PASS — `textinput.go:54` `kp, ok := msg.(tea.KeyPressMsg)`; test file line 27, 83, 121, 144 all use `tea.KeyPressMsg`. No `tea.KeyMsg` anywhere.
- **No `return m, tea.Quit` anywhere:** PASS — visual scan of full `textinput.go`; only returns are `return m, nil` (lines 64, 69) and `return m, cmd` (line 75) where `cmd` is the inner textinput cmd, never `tea.Quit`. Test file contains no `tea.Quit` reference.
- **No `var _ tea.Model = (*TextInputModel)(nil)` assertion:** PASS — absent from both files.
- **Tests assert non-nil cmd is NOT returned by Enter path:** PASS — `textinput_test.go:62-64`, `89-90`, `105-106` all reject non-nil cmd on Enter.

## NITs

### NIT-1 — Validation table-test row 2 has self-overriding semantics

- **File:** `internal/tui/components/textinput_test.go:42-55`
- **Observation:** The second table row is initially declared as `"passing validator submits without error"` with `wantSubmit: true`, then the inline comment block (lines 46-51) explains the situation makes that semantically wrong, and line 55 mutates the table entry (`tests[1].wantSubmit = false`). The result is correct (the test passes), but the table is self-contradictory at the declaration site and the mutation pattern is non-idiomatic for table-driven tests in Go.
- **Severity:** low (cosmetic; correctness unaffected; explains itself in-comment).
- **Suggested fix (deferred — not blocking):** rename the row to `"failing validator rejects empty"`, set `wantSubmit: false` and `wantErrNil: false` at declaration, and drop the `tests[1].wantSubmit = false` line. Alternative: add a real "passing validator submits" path by extending the test to inject a non-empty value via simulated keystrokes before sending Enter.

### NIT-2 — `cmd` discard on Enter path uses sentinel comment

- **File:** `internal/tui/components/textinput.go:57-59`
- **Observation:** On the Enter branch the inner update is invoked and its returned cmd is discarded via `_ = cmd // discard — we always return nil to avoid side-effects on the parent TUI`. This is intentional per the spec ("Update NEVER returns tea.Quit; ... always return nil"), but the inner cmd from `textinput.Model.Update` on an Enter keypress is typically nil anyway — the discard is defensive. If a future bubbles version starts returning a non-nil cmd on Enter (e.g., an animation hook), the discard silently drops it. Worth a doc comment noting that the discard is by design AND a coupling point to bubbles internals.
- **Severity:** low (defensive code; doc-only).

### NIT-3 — Package doc comment block sits between the migration marker and `package`

- **File:** `internal/tui/components/textinput.go:1-7`
- **Observation:** Layout is `// MIGRATION TARGET:` (line 1) → blank (line 2) → 4-line package doc comment (lines 3-6) → `package components` (line 7). The spec wording is "marker comment immediately before `package components`." This file has the marker as the top-of-file comment but NOT immediately before the package clause — the doc comment intervenes. `textinput_test.go` matches the spec exactly (marker on line 1, package on line 2, no intervening comment). The placement in `textinput.go` is defensible (the package doc only lives once and it's attached to a real package source file) and other files in the package may have the same pattern, but it is a literal divergence from the spec wording.
- **Severity:** low (spec ambiguity; functional behavior identical; consistent with stdlib package-doc convention which requires the doc comment to be the immediate-pre-package comment).
- **Suggested resolution:** treat the spec line as "file-level comment before package clause" (which it is) rather than "physically adjacent to package clause." A spec clarification in PLAN.md for future droplets would prevent re-litigation.

## Verdict rationale

All six AcceptanceCriteria pass with on-disk evidence. All special-focus items pass: migration marker file-level, return type concrete, `tea.KeyPressMsg` only, no `tea.Quit`, no `tea.Model` assertion, tests reject non-nil cmd on Enter path. Three NITs raised — all cosmetic / doc-grade; none block the droplet. Overall: **PASS WITH NITS**.
