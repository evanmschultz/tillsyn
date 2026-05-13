# W5.D3 R2 — BUILD-QA-PROOF Verdict

**Date:** 2026-05-13
**Reviewer:** go-qa-proof-agent (opus)
**Overall verdict:** PASS

## Acceptance Bullet Coverage

### Bullet 1 — `textinput.go` compiles alongside `confirm.go` in `internal/tui/components`

- **Evidence:** `mage test-pkg ./internal/tui/components` → `[PKG PASS] github.com/evanmschultz/tillsyn/internal/tui/components (0.00s)`; 46 tests / 46 passed / 0 failed / 0 skipped (this session). Package now contains `confirm.go`, `confirm_test.go`, `progress.go`, `textinput.go`, `textinput_test.go`, plus D4 picker files and D5 header/footer files compiling together (all untracked).
- **Verdict:** PASS

### Bullet 2 — `TextInputModel` is a Bubble Tea sub-component: `Update(tea.Msg) (TextInputModel, tea.Cmd)`, `View() string`, `Init() tea.Cmd` — NOT `tea.Model`. No `var _ tea.Model = (...)` assertion

- **Evidence:**
  - `internal/tui/components/textinput.go:46` — `func (m TextInputModel) Init() tea.Cmd`.
  - `internal/tui/components/textinput.go:60` — `func (m TextInputModel) Update(msg tea.Msg) (TextInputModel, tea.Cmd)` — concrete return tuple, not `tea.Model`.
  - `internal/tui/components/textinput.go:96` — `func (m TextInputModel) View() string`.
  - Visual scan of all 113 lines of `textinput.go`: no `var _ tea.Model = ...` assertion.
- **Verdict:** PASS

### Bullet 3 — `Value()`, `Err()`, `Submitted()` accessors exist

- **Evidence:**
  - `internal/tui/components/textinput.go:105` — `func (m TextInputModel) Value() string { return m.inner.Value() }`
  - `internal/tui/components/textinput.go:109` — `func (m TextInputModel) Err() error { return m.err }`
  - `internal/tui/components/textinput.go:113` — `func (m TextInputModel) Submitted() bool { return m.submitted }`
- **Verdict:** PASS

### Bullet 4 — `textinput_test.go` passes: validation + submit matrix covered; no non-nil cmd returned

- **Evidence:**
  - `internal/tui/components/textinput_test.go:27` `TestTextInputModel_Validation` — table-driven (2 rows: nil validator + failing validator); rejects non-nil cmd at line 57-59.
  - `internal/tui/components/textinput_test.go:77` `TestTextInputModel_Submit` — three subtests covering Enter+nil-validator, Enter+failing-validator, non-Enter key. Lines 84-86 and 100-102 reject non-nil cmd on Enter paths; line 118 explicitly tolerates inner-driven cmd on character keys with `_ = cmd` and inline rationale.
  - `internal/tui/components/textinput_test.go:127` `TestTextInputModel_View` — covers no-error and error-present render paths.
  - `internal/tui/components/textinput_test.go:152` `TestTextInputModel_Accessors` — fresh-model defaults.
  - `internal/tui/components/textinput_test.go:169` `TestTextInputModel_ValidationOnKeystroke` — NEW regression test for CE-1: types 'a' (length 1, err non-nil), then 'b'/'c'/'d' (length 4, err nil), asserts `Submitted()=false` throughout.
  - `mage test-pkg ./internal/tui/components`: 46/46 PASS.
- **Verdict:** PASS

### Bullet 5 — Migration marker present in both files as file-level comment before `package components`

- **Evidence:**
  - `internal/tui/components/textinput.go:1` — `// MIGRATION TARGET: github.com/hylla-org/lykta` (line 1, before `package components` at line 7; package doc comment intervenes — see Round-1 NIT-3 / R2 absorption note below).
  - `internal/tui/components/textinput_test.go:1` — `// MIGRATION TARGET: github.com/hylla-org/lykta` (line 1, immediately before `package components` at line 2).
- **Verdict:** PASS

### Bullet 6 — `mage test-pkg ./internal/tui/components` passes (full package)

- **Evidence:** `[PKG PASS] github.com/evanmschultz/tillsyn/internal/tui/components (0.00s)` — 46/46 PASS (this session).
- **Verdict:** PASS

## Round-1 NIT Absorption Status

### PROOF-NIT-1 / FALS-NIT-3 — Validation row 2 self-overriding semantics

- **Status:** ABSORBED
- **Evidence:** `textinput_test.go:45` — row name now `"failing validator rejects empty"`; line 47 `wantSubmit: false` at declaration; line 48 `wantErrNil: false` at declaration; no `tests[1].wantSubmit = false` override anywhere (grep confirms absence). Inline comment at lines 43-44 explains why empty value exercises the failing path. Test passes.

### PROOF-NIT-2 / FALS-NIT-5 — Enter-path cmd discard doc-comment

- **Status:** ABSORBED
- **Evidence:** `textinput.go:66-71` — multi-line doc-comment now states: "Discard inner cmd on Enter: Enter is a terminal event for this wrapper and we always return nil, suppressing any side-effect command the inner model might produce. This is a deliberate coupling point to bubbles internals — if a future version of bubbles/textinput gains an Enter-side-effect command, it will be silently dropped here by design." This satisfies both R1 PROOF-NIT-2's ask (note the discard is by design AND a coupling point) and R1 FALS-NIT-5's ask (annotate the suppression rationale on Enter terminal events).

### PROOF-NIT-3 — Package doc comment block between marker and `package`

- **Status:** ACCEPTED (no code change required — functional behavior identical; matches stdlib package-doc convention). The migration marker remains the topmost file-level comment in `textinput.go` (line 1, before any other content); package doc comment (lines 3-7) precedes `package components` (line 7) per stdlib convention. Matches Round-1 reviewer's suggested resolution "treat the spec line as 'file-level comment before package clause.'"

### FALS-NIT-1 — TextInputModel docstring imprecision on `tea.Quit` guarantee

- **Status:** ABSORBED
- **Evidence:** `textinput.go:16-21` — struct doc-comment rewritten: "Update never injects `tea.Quit` into the command stream; pass-through of inner-model commands on non-Enter messages is unchanged and depends on the upstream bubbles library." This precisely matches FALS-NIT-1's suggested narrowing ("Update never injects tea.Quit; pass-through of inner cmds is unchanged"). The guarantee is now scoped to the wrapper's own injections, not to the inner model's pass-through.

### FALS-NIT-2 — Spec line 416 "Tillsyn styling" not applied in D3

- **Status:** DEFERRED TO W5.D5 (per fix-up spec; consistent with R1 falsification's own classification "arguably descoped to W5.D5"). No D3 code change. Acceptable — D3's `Specify.KindPayload` does not enumerate concrete styling requirements; the objective text is the only mention and W5.D5 is the integration droplet. Confirmed by inspection: `textinput.go` has zero `lipgloss` or `internal/tui/style` imports.

### FALS-NIT-4 — `ti.Focus()` cmd discard

- **Status:** ABSORBED
- **Evidence:** `textinput.go:34-36` — constructor now: `_ = ti.Focus() // Focus has pointer-receiver side-effects (sets focused state); the / returned blink cmd is intentionally discarded here — Init() returns textinput.Blink / to start the blink loop via the Bubble Tea runtime instead.` Marks the discard explicitly with `_ =` and gives the multi-line rationale FALS-NIT-4 requested.

## CE-1 Resolution Verification (spec line 443)

- **Spec quote (PLAN.md line 443):** "On all other messages: calls `validate(inner.Value())` to keep error state current."
- **Fixed code (textinput.go:84-90):**
  ```go
  // For non-Enter messages, delegate to inner and propagate its command, then
  // re-run validate so err reflects the current value after every keystroke.
  var cmd tea.Cmd
  m.inner, cmd = m.inner.Update(msg)
  if m.validate != nil {
      m.err = m.validate(m.inner.Value())
  }
  return m, cmd
  ```
  - Validator now runs on the non-Enter branch.
  - Assignment is direct: `m.err = m.validate(...)` — nil result clears prior error (the value-recovery path); non-nil sets the current error.
  - Nil-validator guard preserved (`if m.validate != nil`), so nil-validator path remains a no-op.
  - Inner cmd is propagated (`return m, cmd`), as required by spec for non-Enter messages.
- **Regression test (`TestTextInputModel_ValidationOnKeystroke`, lines 169-203):**
  - Constructs validator rejecting strings shorter than 3 chars.
  - Fresh-model precondition: `Err() == nil` (validate never ran). PASS.
  - Type 'a' → value="a" (length 1), assert `Err() != nil` (rejection populated on keystroke). PASS — value-rejection path verified.
  - Type 'b' / 'c' / 'd' → value="abcd" (length 4), assert `Err() == nil` (value-recovery path: prior error CLEARED because validator now returns nil and assignment overwrites). PASS — value-recovery path verified.
  - Assert `Submitted() == false` throughout. PASS — non-Enter keystrokes never submit.
- **Verdict:** CE-1 RESOLVED. The value-recovery path that R1 falsification flagged ("Err() stays stale until user presses Enter again") is now actively tested and proven correct.

## NITs (new in R2)

None.

The fix-up surgically addresses CE-1 + all 7 NITs from R1 (3 PROOF + 5 FALS, of which 2 pairs overlapped). The new regression test directly exercises both the rejection AND value-recovery sides of spec line 443. No new acceptance regressions observed: all 6 AcceptanceCriteria continue to pass; the 4 pre-existing test functions still pass without modification beyond the table-row rename in `TestTextInputModel_Validation` (which is itself a NIT absorption). Test count grew from 28 (R1 scope) to 46 in the live package run — gain attributable to D4/D5 untracked files compiled into the same package, NOT D3 regressions.

## Verdict rationale

All six AcceptanceCriteria pass with current on-disk evidence. CE-1 is resolved by the validator call in the non-Enter branch (textinput.go:88-90) plus a dedicated regression test exercising the value-recovery path (`TestTextInputModel_ValidationOnKeystroke`). All 7 R1 NITs are either absorbed via code/doc edits (PROOF-NIT-1, PROOF-NIT-2, FALS-NIT-1, FALS-NIT-3, FALS-NIT-4, FALS-NIT-5) or explicitly accepted as cosmetic/deferred (PROOF-NIT-3, FALS-NIT-2). No new NITs surfaced in R2. Mage test-pkg PASS (46/46). Overall: **PASS**.
