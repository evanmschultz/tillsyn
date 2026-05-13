# W5.D3 R2 — BUILD-QA-FALSIFICATION Verdict

**Date:** 2026-05-13
**Reviewer:** go-qa-falsification-agent (opus)
**Overall verdict:** PASS WITH NITS

## CE-1 Resolution Verification

**Status: RESOLVED.**

`internal/tui/components/textinput.go` lines 84-91 — the non-Enter branch of `Update`:

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

Direct assignment to `m.err` means validate's return value (nil or error) is reflected
verbatim — fail→pass transitions clear err, pass→fail transitions set it. Nil-validate
is correctly guarded at line 88. Spec line 443 ("On all other messages: calls
`validate(inner.Value())` to keep error state current") is satisfied literally.

Round-1 CE-1 repro (type then back-down to invalid → stale `m.err`) now succeeds:
each keystroke re-validates and overwrites `m.err`, so View() at line 96-102 renders
the up-to-date error string. `TestTextInputModel_ValidationOnKeystroke`
(textinput_test.go lines 169-203) exercises a fresh-model → "a" (invalid, err non-nil)
→ "abcd" (valid, err nil) sequence; passes under `-race`.

Component-package gate: `mage test-pkg ./internal/tui/components` reports 46/46 PASS.
Regression-specific gate: `mage test-func ./internal/tui/components
TestTextInputModel_ValidationOnKeystroke` PASS.

## New Attack Hypotheses

**H1 — Premature err on first keystroke (REFUTED as bug; INTENDED).**
After typing one char into a min-len-3 validator, Err() is immediately non-nil.
This is what the fix is for — spec line 443 mandates that err reflect current value
state on every msg. Not a bug; it is the contracted behavior.

**H2 — Nil-validate non-Enter no-op (REFUTED).**
Line 88 guards `if m.validate != nil` before invoking. When validate is nil, err
is never written, stays at the zero-value nil from the constructor.

**H3 — m.err clearing fail→pass (REFUTED).**
Single direct assignment `m.err = m.validate(...)` at line 89 makes the clear path
identical to the set path. New test lines 191-197 verify experimentally.

**H4 — m.err setting pass→fail (REFUTED).**
Same single-line mechanism handles both directions. No special-casing.
**NIT-1: test does NOT explicitly exercise pass→fail (typing valid then deleting).**
Code path is symmetric so the gap is low-risk, but a one-line test addition would
make coverage symmetric.

**H5 — Validate fires on non-keystroke messages (cursor.BlinkMsg, focus, paste, mouse, resize).**
The non-Enter branch fires on EVERY non-Enter message, not just keystrokes. Spec
line 443 says "all other messages" — literal match. **NIT-2: validate function is
called extremely frequently (once per blink tick at minimum). Doc-comment lines 51-58
don't warn users that validate must be cheap/pure.** If a downstream caller passes
a validate that does I/O, performance and side-effect behavior will degrade.
A one-line addition to the `NewTextInput` or `TextInputModel` doc-comment stating
"validate must be cheap and side-effect-free" would close this. Low-priority.

**H6 — Test coverage of non-keystroke non-Enter messages (REFUTED as counterexample, NIT).**
`TestTextInputModel_ValidationOnKeystroke` only exercises KeyPressMsg. Other message
types (focus, blur, paste, mouse) traverse the same code path; behavior is identical
because validate sees only the inner value. **NIT-3: no test asserts the same
behavior for non-keystroke non-Enter msgs.** Risk: low — same single code path.

**H7 — Discarded inner cmd on Enter (REFUTED).**
Lines 65-71 explicitly discard inner cmd on Enter with a deliberate-coupling comment.
Spec line 442 mandates `return m, nil` on Enter regardless, so this is correct by
contract.

**H8 — Migration marker present in both files (REFUTED).**
Line 1 of textinput.go and line 1 of textinput_test.go both carry
`// MIGRATION TARGET: github.com/hylla-org/lykta` before `package components`.

**H9 — tea.Quit / tea.Batch / tea.Sequence injection (REFUTED).**
Read of both files shows no occurrence. Update returns either `m, nil` (Enter path)
or `m, cmd` where cmd is inner's cmd (non-Enter path). No quit-class commands.

**H10 — Update return type (REFUTED).**
Line 60: `Update(msg tea.Msg) (TextInputModel, tea.Cmd)` — concrete, not `tea.Model`.

**H11 — Init signature deviation from spec (REFUTED).**
Spec line 439 says "delegates to inner.Init()", but bubbles v2 `textinput.Model` has
no Init method. Implementation uses package-level `textinput.Blink` and the
doc-comment lines 43-45 explicitly documents this deviation. Behavior-equivalent to
"start the blink loop". Acceptable.

**H12 — KeyPressMsg usage (REFUTED).**
Line 61 uses `tea.KeyPressMsg`, the bubbletea v2 idiom (not the v1 `tea.KeyMsg`).

**H13 — KeyEnter detection (REFUTED).**
Line 62: `kp.Code == tea.KeyEnter` — correct.

**H14 — Submitted state machine (REFUTED).**
On Enter with passing validator: `m.submitted = true; m.err = nil` (lines 79-81).
On Enter with failing validator: `m.err = err; return m, nil` with submitted untouched
at its prior value (lines 74-77). On non-Enter: submitted untouched. State machine
is correct.

**H15 — Race conditions (REFUTED).**
`mage test-func` includes `-race`; the new regression test plus full package pass
clean. No shared mutable state across goroutines — Update is value-receiver and
returns a copy.

**H16 — PROOF-NIT-1 table-row rename absorption (REFUTED).**
Test file lines 37-49: row names are "nil validator submits without error" and
"failing validator rejects empty". Names match the assertions in the row's
`wantSubmit`/`wantErrNil` columns. Names are accurate.

**H17 — Fresh-model Err() (REFUTED).**
Test line 181-183 asserts fresh-model Err() is nil. Constructor leaves err at the
zero value (nil).

**H18 — Cmd from inner.Update on non-Enter (REFUTED).**
Line 87 captures cmd, line 91 returns it. Test on lines 186-194 ignores cmd, which
is consistent with the test's purpose (err-state verification, not cmd-state).

## Unmitigated Counterexamples

None.

## NITs

**NIT-1: Test does not exercise pass→fail transition.**
`TestTextInputModel_ValidationOnKeystroke` covers fail (len 1) → pass (len 4) but
not pass → fail (e.g., type "abcd" valid, then backspace to "abc" valid, then
backspace to "ab" invalid — assert Err() flips from nil to non-nil).
Code path is symmetric (single `m.err = m.validate(...)` line), so risk is low,
but explicit coverage would close the asymmetry.
Suggested patch: append a backspace sequence to the existing test or add a second
test case.

**NIT-2: validate cost expectations undocumented.**
With the fix, validate runs on every non-Enter msg including cursor blinks (~every
500ms while focused). Doc-comment lines 31-32 (`NewTextInput`) and lines 50-58
(`Update`) do not state that validate must be cheap and side-effect-free. A future
caller could plug a DB-query validator and degrade TUI responsiveness.
Suggested patch: add "validate must be cheap and side-effect-free; it is invoked on
every Update, including cursor-blink messages" to the `NewTextInput` doc-comment.

**NIT-3: No test for validate firing on non-keystroke messages.**
The non-Enter branch fires on every non-Enter msg type (BlinkMsg, focus, paste,
mouse). Test only verifies KeyPressMsg. Risk: low (same code path); coverage
asymmetry only.
Suggested patch: optional — feed a synthetic non-KeyPressMsg through Update and
assert validate fires.

## Verdict rationale

CE-1 from round-1 is fully resolved: the non-Enter branch invokes
`m.validate(m.inner.Value())` and writes the result to `m.err`, with correct
nil-validate guarding. The new regression test exercises the load-bearing path
(typing through invalid into valid). All 46 components tests pass clean under
`-race`. Round-1's other 13 hypotheses re-checked and still REFUTED; the fix did
not regress anything.

Three NITs surfaced, none load-bearing: pass→fail test asymmetry, validate-cost
documentation gap, and non-keystroke-msg test coverage gap. All are
documentation/coverage refinements, not correctness defects.

**Verdict: PASS WITH NITS.** Recommend absorbing NIT-1 (one-line test addition for
symmetry) and NIT-2 (one-line doc-comment about validate cost) before commit;
NIT-3 is optional polish.
