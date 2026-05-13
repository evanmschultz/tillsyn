# W5.D3 — BUILD-QA-FALSIFICATION Verdict

**Date:** 2026-05-13
**Reviewer:** go-qa-falsification-agent (opus)
**Overall verdict:** FAIL

## Attack Hypotheses Tested

| # | Hypothesis | Result |
| - | --- | --- |
| 1 | `tea.KeyPressMsg` (not deprecated `tea.KeyMsg`) used in Update type-switch | REFUTED — line 54 uses `msg.(tea.KeyPressMsg)`; no `tea.KeyMsg` anywhere |
| 2 | Zero `tea.Quit` references in `textinput.go` | REFUTED — file scan: 0 hits |
| 3 | No `var _ tea.Model = (...)` interface assertion | REFUTED — full file read (98 lines): no such line |
| 4 | `Update` return type is concrete `(TextInputModel, tea.Cmd)` | REFUTED — line 53 signature matches |
| 5 | `Init` returns a real `tea.Cmd`, not a copy of a value | REFUTED — `go doc charm.land/bubbles/v2/textinput.Blink` confirms `Blink` is `func() tea.Msg` (a `tea.Cmd`); returning it as a function value is correct |
| 5b | Spec says `Init() — delegates to inner.Init()`, code returns `textinput.Blink` | REFUTED — `go doc textinput.Model` shows no `Init()` method exists on the inner model; spec text is imprecise but code's choice is the only valid implementation and the doc-comment on line 41 explicitly notes the rationale |
| 6 | Constructor discards `Focus()` cmd — might break first-render cursor blink | REFUTED — `textinput.Model.Focus()` has pointer receiver, mutates the model's focus state field (which is copied into the returned struct via assignment); Init returns `textinput.Blink` separately to drive the blink animation. Tests pass and the blink cmd flows from Init, not Focus |
| 7 | Spec line 443 mandates `validate(inner.Value())` on **all other (non-Enter) messages** to "keep error state current" | **CONFIRMED — counterexample** (see below) |
| 8 | Nil validate must not panic | REFUTED — line 61 guards `if m.validate != nil`; `TestTextInputModel_Validation` nil-validator row passes |
| 9 | View appends error string below input when `m.err != nil` | REFUTED — lines 80-86 do exactly this; `TestTextInputModel_View` "error present" sub-test passes |
| 10 | Submitted=true after invalid-then-valid sequence — does state reset? | REFUTED — lines 67-68 always set `m.submitted = true; m.err = nil` on validate-passes branch; if user types valid input after a rejection AND presses Enter again, validator re-runs and resets. Note: the spec mandates auto-reset of `m.err` on every keystroke (hypothesis 7) which this code does not do, but `submitted` does reset correctly on next successful Enter |
| 11 | Migration marker exact format and placement | REFUTED — both files line 1: `// MIGRATION TARGET: github.com/hylla-org/lykta`; immediately before `package components` |
| 12 | YAGNI — anything beyond spec | REFUTED — no extra accessors, no extra fields, no unused helpers |
| 13 | Stale style helper import (pre-W5.D5 dep) | REFUTED — only imports are `fmt`, `tea`, and `textinput`; no `internal/tui/style` reference |

## Unmitigated Counterexamples

### CE-1 — Validation NOT run on non-Enter keystrokes (spec line 443 violated)

**Severity:** SPEC-COMPLIANCE FAIL (counterexample to acceptance-criterion bullet 4 "validation + submit matrix covered")

**Spec quote (PLAN.md line 443):**
> "On all other messages: calls `validate(inner.Value())` to keep error state current."

**Code (textinput.go lines 71-76):**
```go
// For non-Enter messages, delegate to inner and propagate its command.
var cmd tea.Cmd
m.inner, cmd = m.inner.Update(msg)
return m, cmd
```

The non-Enter branch delegates to `inner.Update` and returns — it does NOT call `m.validate(m.inner.Value())`. Consequence: if the user types into a field with a failing validator, presses Enter (sets `m.err`), then types more characters that would now satisfy the validator, `m.err` stays stale until the user presses Enter again. The View renders the stale error indefinitely. This directly contradicts "to keep error state current."

**Reproduction (pseudo-Go, sub-component scope):**
```
m := NewTextInput("ph", func(s string) error {
    if len(s) < 3 { return fmt.Errorf("too short") }
    return nil
})
m, _ = m.Update(typeKey('a'))                          // value="a", err=nil (validate never ran)
m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})   // value="a", err="too short", submitted=false
m, _ = m.Update(typeKey('b'))                          // value="ab", err STILL "too short" (BUG — should re-run validate)
m, _ = m.Update(typeKey('c'))                          // value="abc", err STILL "too short" (BUG — value now valid, error stale)
m.View()                                                // renders "too short" beneath the input
```

The existing test suite does NOT cover this case — `TestTextInputModel_Submit`'s "non-Enter key does not submit" sub-test only checks `Submitted()`, not `Err()` currency.

**Fix sketch:** in the non-Enter branch, after delegating to `inner.Update`, call `m.validate(m.inner.Value())` (if non-nil) and update `m.err` accordingly. Add a regression test case "Err() clears when keystroke makes value valid after prior rejection."

---

## NITs

### NIT-1 — TextInputModel docstring says "Update never returns tea.Quit" but non-Enter branch returns `cmd` from inner

**File:** `textinput.go` lines 17-19 vs line 75.

The doc-comment (line 19) says "Update never returns tea.Quit." The non-Enter branch (line 75) propagates whatever cmd `inner.Update` returns. If a future bubbles/textinput release ever returned `tea.Quit` from its inner Update (unlikely but not impossible — e.g., on ctrl+C handling in some bindings), this wrapper would silently forward it. Today the inner model never returns Quit, so the doc-comment is accurate in practice — but the guarantee depends on the upstream library, not on this wrapper. Either narrow the doc-comment ("Update never injects tea.Quit; pass-through of inner cmds is unchanged") or actively filter Quit from `cmd` in the non-Enter branch. Low priority — defensive nit, no current breakage.

### NIT-2 — Spec line 416 mentions "Tillsyn styling" but no styling is applied

**File:** `textinput.go` line 80-86 (`View`).

`Objective` (PLAN.md line 416) says "wraps `bubbles/textinput` with Tillsyn styling and an optional validation hook." The View method just returns `m.inner.View()` with an appended raw error string — no lipgloss styling on the error line, no theme application on the input. The KindPayload section doesn't enumerate specific style requirements, so this is arguably descoped to W5.D5 (style integration), but the objective text is left unfulfilled in D3. Recommend the planner either (a) drop "Tillsyn styling" from the D3 objective and explicitly defer to D5, or (b) add a follow-up build (likely W5.D5) that wires styled rendering into the View. Decision NIT, not a code-correctness issue.

### NIT-3 — Test TestTextInputModel_Validation has dead/contradictory row metadata

**File:** `textinput_test.go` lines 41-55.

The second table row is named `"passing validator submits without error"` (line 42), with field `wantSubmit: true` (line 43), `wantErrNil: false` (line 44), and is then overridden by `tests[1].wantSubmit = false` on line 55. The expectations no longer match the name. The intent is clear from the inline comment ("Adjust: ... this case is 'passing validator with a valid value' — we cannot pre-set inner value without a dedicated helper") but the resulting code is confusing. Either rewrite the row's `name` to reflect the actual assertion (e.g. `"failing validator rejects empty value"`), or refactor to construct the model with a non-empty inner value via a test helper so the "passing" path can be exercised. Test passes — pure clarity NIT.

### NIT-4 — Constructor calls `ti.Focus()` and discards the returned cmd

**File:** `textinput.go` line 32.

`textinput.Model.Focus()` has signature `func (m *Model) Focus() tea.Cmd`. The cmd is the blink-trigger. The constructor discards it; `Init()` returns `textinput.Blink` separately. Both call sites produce equivalent blink behavior in practice (the constructor's Focus mutates internal focused-state on `ti`, and `Init` is what the Bubble Tea runtime calls to start the blink loop), so there is no behavioral defect. But the discarded return is a sleeper — if a future bubbles version changes Focus's cmd semantic, this wrapper silently drops it. Either capture and route it from Init (cleaner) or annotate the discard with a `_ =` to mark intentional. Low priority.

### NIT-5 — Inner.Update return is shadowed but cmd discarded on Enter path

**File:** `textinput.go` lines 56-59.

```go
var cmd tea.Cmd
m.inner, cmd = m.inner.Update(msg)
_ = cmd // discard — we always return nil to avoid side-effects on the parent TUI
```

The doc-comment rationalizes the discard. The Enter path's hard guarantee of `return m, nil` (lines 64, 67-69) means any cmd from the inner model on Enter is silently dropped. For the current bubbles textinput this is fine (Enter doesn't trigger commands), but it's a wide invariant to assert. Consider a comment line stating "if the inner model gains an Enter-side-effect command in a future version, that's intentionally suppressed here because Enter is a terminal event for this wrapper." Documentation-only NIT.

---

## Verdict rationale

**FAIL** because CE-1 is a direct, reproducible violation of an explicit spec mandate (PLAN.md line 443: "On all other messages: calls `validate(inner.Value())` to keep error state current"). The implementation only runs `validate()` on Enter. The user-visible consequence — stale error rendering after value-corrective keystrokes — is a real defect that the existing test suite did not exercise.

All other attack hypotheses (1-6, 8-13) were refuted by direct evidence:

- Bubble Tea v2 API surface used correctly (`KeyPressMsg`, no `KeyMsg`).
- No `tea.Quit` references, no `tea.Model` assertion, concrete return type.
- `textinput.Blink` is genuinely a `tea.Cmd` (verified via `go doc`); `Init` correctly returns it; the spec phrase "delegates to inner.Init()" is doc-imprecision (no such method exists on `textinput.Model`), and the implementation's note on line 41 is accurate.
- Nil validate guarded; migration markers present; no YAGNI; no stale imports.
- 28/28 tests pass, including race build.

Recommended next-round build action:

1. Add the missing validate call to the non-Enter branch of `Update`:
   ```go
   m.inner, cmd = m.inner.Update(msg)
   if m.validate != nil {
       m.err = m.validate(m.inner.Value())  // may set to nil if now valid
   }
   return m, cmd
   ```
2. Add a regression test: `TestTextInputModel_ErrClearsOnValidKeystroke` that types invalid → Enter (errors) → valid keystroke → assert `Err() == nil` BEFORE pressing Enter.
3. Address NITs 1-5 inline per "NITs are first-class" rule (memory feedback_nits_are_first_class).

Pre-W5.D4-parallel note: the fix is package-internal and does not touch any other file in `internal/tui/components`. W5.D4 builder (running in parallel) is in a different file scope; no merge contention expected for this round-2 fix.
