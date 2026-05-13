# PLAN QA Falsification — W5 (TUI Components + Style + Vim Keybinding Dispatcher)

**Round:** 2
**Mode:** Filesystem-MD-only; Hylla OFF
**Target:** `workflow/drop_4c_6_1/DROP_4c.6.1.W5_TUI_COMPONENTS/PLAN.md` round-2 absorption + `_BLOCKERS.toml`

---

## Verdict

**PASS-WITH-FINDINGS** — 1 FF (MEDIUM, downgraded from would-be CRITICAL because L1 locked-decisions already absorbed the underlying contract) + 6 NITs. All three round-1 FF dispositions are correctly applied at the L2 layer. The remaining FF is a **cross-doc inconsistency**: L1 PLAN.md's W5 scope block (lines 506, 510-511) was not updated when L1 Round-10 locked-decisions absorbed the same fix. Round-2 W5 PLAN's claim "L1 line 470 note updated (in L2 Scope note)" is misleading — L1 was NOT actually edited; only the L2 W5 Scope note was. Builders crossing L1 wave-W5 scope text will read the stale "implements `tea.Model`" contract. Downgraded from CRITICAL because L1 line 928 (Locked architectural decisions, Round-10 absorption W5 fals FF1+FF2) is the precedence-take-all ledger and IS correct — but the contradiction is real and orchestrator-level fix is needed.

The plan is otherwise architecturally sound. The 6-droplet decomposition is correct, the `blocked_by` graph is acyclic, the embedded baseline-JSON nested-schema commitment is consistent across baseline embed + local file + decoder, and the multi-key KEYBIND-R4 deferral is now testable (`TestDispatcher_MultiKey_Returns_NoOp`).

---

## 1. Findings

### 1.1 FF — Severity: MEDIUM — L1 wave-W5 scope block has stale "implements `tea.Model`" contract; round-2 absorption note misstates the fix

**Trace (counterexample):**

- Round-2 W5 PLAN.md line 17-18 states: "All 'implements `tea.Model`' claims removed from AC3, AC4, D3 AC, D4 AC, **and the L1 scope note**."
- Round-2 W5 PLAN.md line 45 states: "Fals NIT1.10 (`progress.go` L1/L2 contradiction): ABSORBED. L1 line 470 note updated (in L2 Scope note); D2 progress.go is explicitly a passive render-only struct…"
- However, the actual L1 PLAN.md was NOT edited. L1 PLAN.md lines 506, 510, 511 (current) still read verbatim:
  - line 506: "All component implementations are pure Bubble Tea v2 models (Init/Update/View)."
  - line 510: "`confirm.go` implements `tea.Model`; renders y/n prompt; `Confirmed()` / `Cancelled()` accessors work."
  - line 511: "`picker_multi.go` implements `tea.Model`; returns `[]string` of selected items…"
- L1 PLAN.md line 928 (Locked architectural decisions — Round-10 absorption block) DOES contain the corrected contract: "W5 components are Bubble Tea sub-models composed by an outer `tea.Model`; they don't satisfy `tea.Model` directly. Spec drops the `var _ tea.Model = (*ConfirmModel)(nil)` claim and reframes as sub-component interfaces."

**Why this is load-bearing:**

L1 PLAN.md has two layers that now contradict each other:
- L1 **wave-W5 acceptance block** (lines 506-513) says components implement `tea.Model`.
- L1 **locked-decisions ledger** (line 928) says they're sub-models that don't.

The locked-decisions ledger is the precedence-take-all section ("Locked architectural decisions" + explicit Round-10 absorption marker), so L1 effectively absorbed the fix. But a reader (or build-QA-falsification on a downstream W5 droplet) who reads only the L1 wave-W5 scope block will surface a contradiction with what the L2 plan + builds. The round-2 absorption note's claim "L1 line 470 note updated (in L2 Scope note)" is misleading — the L2 Scope note is in the L2 W5 PLAN.md (this file), not in L1.

A future builder cross-referencing L1 line 510 ("`confirm.go` implements `tea.Model`") would either (a) reject D2's implementation as failing L1 acceptance, or (b) silently re-introduce `var _ tea.Model = (*ConfirmModel)(nil)` static check, which **fails to compile** because `View() string` doesn't satisfy `tea.Model.View() View`.

**Why MEDIUM and not CRITICAL:**

L1 line 928 (Round-10 locked-decisions ledger) IS authoritative and IS correct. Round-10's "Locked architectural decisions" framing means the locked-decisions wins on conflict. So the contract is recoverable. But the in-section L1 wave-W5 text is stale and any cross-doc consistency check will flag it. The Round-2 absorption note is FALSE about L1 being updated. Honest absorption would say "L2 Scope note + L1 locked-decisions ledger both encode the fix; L1 wave-W5 in-section text is stale and tracked as cross-doc cleanup."

**Disposition (orchestrator picks):**

Two options:
- **Option A (preferred):** Orchestrator edits L1 PLAN.md (out of L2 W5 planner's authority) lines 506, 510, 511 to match line 928's locked-decisions framing. After edit, the round-2 W5 absorption note ("L1 line 470 note updated") becomes true.
- **Option B:** Leave L1 wave-W5 scope text alone (it's a historical record showing what changed); add an explicit cross-reference in L1 wave-W5 scope ("see line 928 locked-decisions for round-10 absorption — supersedes the in-section spec"). Less surgical but preserves audit trail.

W5 round-2 PLAN.md should update its absorption note line 18 + line 45 to honestly describe the current state: "L2 W5 Scope note updated (this file). L1 Locked-decisions ledger (line 928) already updated in Round-10. L1 wave-W5 in-section scope (lines 506, 510, 511) NOT yet updated — flagged for orchestrator." This is a 2-line edit to round-2 W5 PLAN.md.

---

### 1.2 NIT — `bindingsFile` JSON decoder unknown-field handling is unspecified

**Trace:**

Round-2 W5 PLAN.md line 750-757 introduces an internal `bindingsFile` struct that only maps `product_extensions.tillsyn.commands`. Line 770 says: "Extra top-level fields (`schema_version`, `name`, `description`, `extends`) are ignored by the decoder — `bindingsFile` only maps `product_extensions`."

This is true if the builder uses default `json.Unmarshal(data, &bf)` — Go's encoding/json silently ignores unknown fields. **BUT**: a builder reading "fail loudly on malformed JSON" elsewhere in the spec might reach for `json.NewDecoder(r).DisallowUnknownFields()` for safety, which then REJECTS the local file because `schema_version`, `name`, `description`, `extends`, `extends_path` are all unknown. The spec doesn't pin which decoder pattern to use.

**Disposition:** Add a one-line constraint to D6 loader.go Specify: "`LoadBindings` uses default `json.Unmarshal` (which silently ignores unknown fields) — do NOT use `json.NewDecoder.DisallowUnknownFields()`, as the local file deliberately contains additional top-level metadata fields (`schema_version`, `name`, `description`, `extends`, `extends_path`) that aren't in `bindingsFile`."

---

### 1.3 NIT — AC1 file-count drift: "10 component source files" listed but 11 enumerated

**Trace:**

Round-2 W5 PLAN.md line 69-72:

> "AC1 — All files listed in Paths sections below exist and compile: 10 component source files (`confirm.go`, `confirm_test.go`, `textinput.go`, `textinput_test.go`, `picker_single.go`, `picker_single_test.go`, `picker_multi.go`, `picker_multi_test.go`, `header.go`, `footer.go`, `progress.go`)…"

Count: confirm.go, confirm_test.go, textinput.go, textinput_test.go, picker_single.go, picker_single_test.go, picker_multi.go, picker_multi_test.go, header.go, footer.go, progress.go = **eleven** files, called "10".

Cross-check actual paths from D2-D5 (confirm.go, confirm_test.go, progress.go, textinput.go, textinput_test.go, picker_single.go, picker_single_test.go, picker_multi.go, picker_multi_test.go, header.go, footer.go) = 11 files. The literal D-list enumeration is correct (11 files); the number "10" in AC1 is wrong.

**Disposition:** Tighten AC1 to "11 component source files" or rewrite as "all component files listed in D2-D5 Paths sections." Same for style ("4 style files" — palette.go, palette_test.go, spacing.go, typography.go = 4 ✓) and keybinding ("4 keybinding files" — dispatcher.go, loader.go, modes.go, dispatcher_test.go = 4 ✓).

---

### 1.4 NIT — Done()/Confirmed()/Cancelled() polling contract under-specified

**Trace:**

Round-2 PLAN.md line 20: "The parent TUI… polls these accessors on each Update cycle to advance its own state machine."

The pattern is: parent's `Update` calls `m.confirm, cmd = m.confirm.Update(msg)`, then checks `m.confirm.Done()`. But the spec doesn't pin:
- WHERE in the parent's `Update` the accessor poll happens (after sub-component Update? at top of next tick?).
- WHAT cmd the sub-component returns when done (round-2 says `nil` — fine).
- WHETHER a custom `func() tea.Msg` cmd that emits e.g. `ConfirmDoneMsg{Confirmed: true}` would be a valid alternative (more idiomatic Bubble Tea).

W2's D3/D4 are blocked on this contract. W2 round-2 PLAN line 184-185 already says "use `return nil` + `Done()`/`Cancelled()` accessors" — consistent with W5. But the polling cadence is not formalized; left to builder judgment.

**Disposition:** Either accept builder-judgment (likely fine — "after sub-component Update, check `m.confirm.Done()` and route accordingly" is a standard Bubble Tea sub-model pattern) or pin in D2 Specify with a 3-line example showing parent's `Update` polling shape. Recommend leaving as-is + adding a one-line note: "parent integration pattern: after `m.confirm, cmd = m.confirm.Update(msg)`, parent inspects `m.confirm.Done()` and advances its own state machine on the next iteration. No custom `tea.Msg` cmd required from sub-component."

---

### 1.5 NIT — `progress.go` framing in L1 vs round-2 W5 — minor cross-doc consistency drift (same root as 1.1)

**Trace:**

- L1 PLAN.md line 506 (already covered by 1.1): "All component implementations are pure Bubble Tea v2 models (Init/Update/View)."
- L1 PLAN.md line 491: "`progress.go` — single-step status line." (No `tea.Model` claim, just a description.)
- Round-2 W5 PLAN.md line 168-170: "`progress.go` is a passive render-only struct. It does NOT have `Init()` or `Update()` methods."

Both round-2 W5's framing (passive struct) AND L1 line 491 (status line, no `tea.Model` claim) are consistent. The contradiction lives ONLY at L1 line 506 (covered by 1.1).

This is the same root finding as 1.1 but worth a separate row because round-2 W5 PLAN.md line 45 explicitly claims "L1 line 470 note updated" — which would affect line 506's framing for progress.go too. Not updated; same FF.

**Disposition:** Folded into 1.1's fix (orchestrator edits L1 wave-W5 scope, OR adds locked-decisions cross-reference).

---

### 1.6 NIT — `palette_test.go` coverage gate test asserts presence, not behavior

**Trace:**

Round-2 W5 PLAN.md lines 278-281: "`palette_test.go`: file-level `// MIGRATION TARGET: github.com/hylla-org/lykta` before `package style`. `TestAllColors_NonEmpty`: calls `AllColors()` and asserts len > 0 and each element is non-zero."

`AllColors()` returns `[]lipgloss.Color`. `lipgloss.Color` is a `string`-based type — `Color("")` is a valid zero. The test asserts "each element is non-zero" which presumably means `string != ""`. That's correct as a presence assertion but doesn't exercise actual color rendering (e.g. whether the color is parseable by lipgloss). For pure coverage-gate satisfaction this is sufficient (one statement called → 100% coverage of `AllColors`). NIT only because the test is minimal — pure tripwire, not a behavioral validation.

**Disposition:** Accept as-is for D1 (the test's role is the coverage tripwire, nothing more). Consider adding to D1 a TODO/REFINEMENT for richer palette-test coverage post-MVP. Builder may add additional assertions per their judgment.

---

### 1.7 NIT — D5 header.go + footer.go have no concrete W2/W3/W6 consumer in scope; L1-level YAGNI (L2 round-2 faithfully ships per L1 contract)

**Trace:**

Round-2 W5 PLAN.md ships `header.go` + `footer.go` in D5 per L1 paths (line 472). But:
- W2 round-2 PLAN.md uses only `confirm.go` (D4) + `picker_multi.go` (D3 of W2). No reference to Header/Footer.
- W3 (CLI surface) is package `cmd/till`, not TUI.
- W6 is FE/Wails, not Go TUI.
- L1 PLAN.md line 490 says "`header.go` / `footer.go` — styled chrome bars" with no concrete consumer.

This is L1-level YAGNI: L1 mandates these two files but no in-drop consumer needs them. The L2 round-2 plan can't drop them (L1 paths are contract). Could flag to orchestrator for L1 reconsideration.

**Disposition:** Defer — not in W5 L2 planner's authority to remove. Surface to orchestrator as a possible L1 scope-trim (header.go + footer.go → DEFER-to-Drop-4c.7 or later when a concrete consumer surfaces). If orchestrator confirms YAGNI accept, then drop D5 entirely and pull the package-coverage check forward to D4's exit. Otherwise ship as-is — they're cheap (each 1-2 code blocks).

---

## 2. Counterexamples

The single FF (1.1) is the only CONFIRMED counterexample to "round-2 absorption is internally consistent." It does NOT counterexample the executability of W5 — the L2 PLAN.md ACs are correct and a builder following only W5 round-2 will produce correct sub-component code. The counterexample is purely about the absorption note's truthfulness ("L1 line 470 note updated") vs. the actual L1 state.

The 6 NITs are under-specs / drifts that build-QA will catch or builder will resolve via judgment. None are show-stoppers.

---

## 3. PLAN-QA-DISCIPLINE checks

### 3.1 R1 — every NEW-behavior acceptance bullet has a test-runner shipping it

- AC1 (file existence) → per-droplet `mage test-pkg` in each Mage verification. ✓
- AC2 (migration marker) → per-droplet build-QA-proof bullet (Option b from round-1; MIGRATE-MARKER-R1 tracks Option a for later). ✓
- AC3 (ConfirmModel sub-component + accessors) → D2 `confirm_test.go` table-driven. ✓
- AC4 (PickerMultiModel sub-component + accessors) → D4 `picker_multi_test.go`. ✓
- AC5 (Dispatch returns handler or NoOp; multi-key returns NoOp pending KEYBIND-R4) → D6 `dispatcher_test.go` `TestDispatcher_Dispatch` + `TestDispatcher_MultiKey_Returns_NoOp` + `TestDispatcher_Register` + `TestDispatcher_DispatchCommand`. ✓
- AC6 (LoadBindings nested schema, baseline+local merge, 9/4 counts) → D6 `dispatcher_test.go` `TestLoadBindings_BaselineOnly` + `TestLoadBindings_WithLocal` + `TestLoadBindings_LocalWins` + `TestLoadBindings_MissingLocalFile`. ✓
- AC7 (per-package `mage test-pkg` ≥70%) → each droplet Mage verification. ✓ (D1 coverage tripwire is `palette_test.go`/`TestAllColors_NonEmpty`.)
- AC8 (`mage ci` drop-end gate after both D5 + D6) → orchestrator gate, not per-droplet. ✓

**R1 verdict:** PASS.

### 3.2 R2 — narrative count = D-list count

- Narrative (implicit): D1-D6 = 6 enumerated. ✓
- D-list: D1 + D2 + D3 + D4 + D5 + D6 = 6 sections. ✓
- KindPayload preview (lines 850-855): 6 child entries. ✓
- CompletionChecklist (lines 873-884): 12 build-QA entries (proof+falsification × 6 droplets) + 1 mage ci entry. ✓
- `_BLOCKERS.toml`: 3 entries (D3→D2, D4→D3, D5→D4); D1 + D6 correctly absent. ✓

**Sub-finding (NIT 1.3):** AC1 numeric drift (10 vs 11 component files). Separate from droplet count. Droplet count is correct = 6.

**R2 verdict:** PASS for droplet count; NIT 1.3 for AC1 file-count.

---

## 4. Attack-vector sweep results

Per spawn-prompt attack list, with disposition:

1. **Sub-component composition contract** — Round-2 says components are composed by outer `tea.Model` (model.go or W2's `runInitTUI`). Verified: `internal/tui/model.go:2027` returns `tea.View`, model.go has NO current import of `internal/tui/components` (W5 ships NEW package). W2's `runInitTUI` (in `cmd/till/init_cmd.go`) is the actual immediate consumer (W2 D3 + D4 import `internal/tui/components`). W2 round-2 PLAN.md lines 184-185 confirms it expects sub-component pattern. The wording "outer `tea.Model` at `internal/tui/model.go` or W2's `runInitTUI`" is slightly imprecise (these are two different programs, not interchangeable), but the contract holds. **PASS.**

2. **Done()/Confirmed()/Cancelled() polling pattern** — `return nil` + parent polls accessor on next Update cycle is standard Bubble Tea sub-model practice. NIT 1.4 (polling cadence under-specified). **PASS with NIT.**

3. **Embedded baseline literal staleness** — KEYBIND-R3 grep-discoverable TODO comment is the staleness signal. No automated structural comparison against upstream stil baseline.json. Proof FF1.2 deferred this; round-2 absorbed via TODO only. Documented accepted risk. **PASS.**

4. **Loader JSON `bindingsFile` shape** — Same shape baseline + local (nested `product_extensions.tillsyn.commands`). Decoder is `bindingsFile` struct. NIT 1.2 (unspecified whether `DisallowUnknownFields` is used — could break local file decode). **PASS with NIT.**

5. **KEYBIND-R4 multi-key skip test** — `TestDispatcher_MultiKey_Returns_NoOp` (PLAN line 812-814) explicitly asserts multi-key commands return `NoOp` from `Dispatch` in nav-mode. Named test with named assertion. **PASS.**

6. **`tea.KeyPressMsg` propagation** — All D2/D3/D4 Update type-switches narrowed to `case tea.KeyPressMsg:` (PLAN constraint line 170-174). Consistent with `internal/tui/model.go:2004`. No KeyReleaseMsg handling required for y/n confirm, multi-select picker, text input — these are press-driven. **PASS.**

7. **Per-droplet migration-marker check triggerable** — Each droplet's AC + Context Blocks `constraint (critical)` (line 149-154) explicitly mandates the check. Build-QA-proof agents read action-item description (which inherits droplet AC) and verify per-file. Standard L2-to-droplet propagation handles this. **PASS.**

8. **Numeric consistency (6 droplets across 3 packages)** — Droplet count = 6 consistent. Package count = 3 (`internal/tui/style`, `internal/tui/components`, `internal/tui/keybindings`). File count: 4 style + 11 components + 4 keybinding = 19. AC1 says "10 component source files" — should be 11 (NIT 1.3). **PASS with NIT 1.3.**

9. **`palette_test.go` exercises executable function** — `AllColors() []lipgloss.Color` is a real exported function with non-trivial body (returns slice); `TestAllColors_NonEmpty` calls it. Tripwire-grade, satisfies coverage gate. **PASS.**

10. **`progress.go` framing L1 vs L2** — L1 line 491 "single-step status line" is consistent with L2 round-2 passive struct. L1 line 506 ("all component implementations are pure Bubble Tea v2 models") contradicts L2 round-2 passive framing. Same FF 1.1 / NIT 1.5 root: L1 wave-W5 scope text stale, locked-decisions ledger correct. **WEAK (FF 1.1).**

11. **YAGNI on D5 header/footer** — No concrete consumer in W2/W3/W6 round-2 PLANs. L1 mandates them; L2 ships per L1. NIT 1.7 flagged for orchestrator L1-trim consideration. **PASS with NIT.**

---

## 5. Hylla Feedback

N/A — Hylla is OFF per spawn prompt. All evidence collected via `Read` of: round-2 W5 PLAN.md + _BLOCKERS.toml; round-1 PLAN_QA_PROOF.md + PLAN_QA_FALSIFICATION.md (this dir); L1 PLAN.md (drop-4c_6_1 root); W2 round-2 PLAN.md (W2_TILL_INIT/); REVISION_BRIEF.md §2.14 + §2.19; stil baseline.json. Bubble Tea v2 source at `/Users/evanschultz/go/pkg/mod/charm.land/bubbletea/v2@v2.0.0-rc.2/tea.go` was sandbox-denied — relied on round-1 falsification's verified citations (lines 52-63 for `Model`, 120 for `View`, 191 for `KeyPressMsg`, 224 for `KeyReleaseMsg`) and on `internal/tui/model.go:2004` + `:2027` (accessible) as cross-validation that `tea.KeyPressMsg` + `View() tea.View` are correct for bubbletea v2 rc.2.

---

## 6. Summary

**PASS-WITH-FINDINGS** — round-2 absorption is structurally correct and faithfully addresses all three round-1 FFs (sub-component pattern + no `tea.Quit` + nested-schema loader) and all 16 round-1 NITs (8 proof + 8 falsification). The L2 plan is internally consistent and executable as-written.

**One real finding (FF 1.1):** Round-2 absorption note misstates that L1 PLAN.md was updated. L1 wave-W5 scope block (lines 506, 510-511) still carries the OLD "implements `tea.Model`" contract; only L1 locked-decisions ledger (line 928) carries the corrected absorption. Round-2 W5 PLAN.md should either (a) honestly describe this state in its absorption note (2-line edit), or (b) escalate to orchestrator for an L1-level edit (out of L2 W5 planner's authority).

**Six NITs:**
- 1.2 `bindingsFile` JSON decoder unknown-field handling (one-line constraint addition).
- 1.3 AC1 file-count drift "10" vs 11 (one-character fix).
- 1.4 Done() polling cadence under-specified (one-line example, or accept builder judgment).
- 1.5 progress.go L1/L2 — folded into 1.1.
- 1.6 palette_test.go is minimal tripwire (accept; consider future REFINEMENT).
- 1.7 D5 header/footer have no concrete consumer — L1 YAGNI flagged for orchestrator.

After 1.1 + 1.2 + 1.3 + 1.4 absorption, plan is ready for L2 droplet dispatch. NITs 1.5/1.6/1.7 are accept-or-defer per orchestrator/dev call.

---

## TL;DR

- **T1**: Verdict — PASS-WITH-FINDINGS. 1 FF (MEDIUM — L1 cross-doc inconsistency) + 6 NITs. Round-2 absorption is structurally correct; the only real issue is that L1 wave-W5 scope block wasn't updated to match L1 locked-decisions ledger (line 928 IS correct, but lines 506/510/511 are stale).
- **T2**: All three round-1 FFs (tea.Model contract, tea.Quit kills parent, loader JSON shape) correctly absorbed into L2 round-2. Sub-component pattern with `return nil` + accessor polling is consistent across W5+W2 round-2 plans.
- **T3**: All 16 round-1 NITs (8 proof + 8 falsification) absorbed; per-NIT disposition documented in round-2 Changes section. KEYBIND-R4 (multi-key skip) now testable via `TestDispatcher_MultiKey_Returns_NoOp`.
- **T4**: AC1 numeric drift "10 component source files" lists 11 (NIT 1.3). Droplet count = 6 consistent everywhere. Embedded baseline + local file share nested schema; one decoder.
- **T5**: PLAN-QA-DISCIPLINE R1 (test-runner-ships-with-behavior) + R2 (D-list count) both PASS at the droplet level. R2 sub-finding (NIT 1.3) on AC1 file-count separate from droplet count.

---

## Hylla Feedback

N/A — Hylla OFF per spawn prompt; evidence via Read + cross-file consistency checks only.
