# PLAN_QA_PROOF — Drop 4c.6.1.W5 (TUI Components + Style + Vim Keybinding Dispatcher) — Round 2

**Round:** 2
**Reviewer:** L2 plan-QA proof agent (round-2 absorption verification)
**Mode:** Filesystem-MD-only; Hylla off
**Target:** `workflow/drop_4c_6_1/DROP_4c.6.1.W5_TUI_COMPONENTS/PLAN.md` + `_BLOCKERS.toml`
**Round-1 inputs:** `PLAN_QA_PROOF.md` (PASS WITH FINDINGS: 2 FFs + 8 NITs) + `PLAN_QA_FALSIFICATION.md` (FAIL: 3 FFs + 8 NITs)

---

## Verdict

**PASS** — all 5 round-1 substantive FFs (Fals FF1.1 CRITICAL, Fals FF1.2 HIGH, Fals FF1.3 + Proof FF1.1 HIGH/MEDIUM, Proof FF1.2 LOW) are FULLY ABSORBED; 13 of 16 NITs are absorbed and 3 are DEFERRED-with-explicit-rationale (KEYBIND-R4 leader-key state machine; MIGRATE-MARKER-R1 mage target; NIT 2.6 advisory-only — non-actionable). The decomposition is structurally sound, the `blocked_by` graph is acyclic, narrative count (6) matches enumeration (D1-D6), `_BLOCKERS.toml` mirrors PLAN.md, and per-droplet acceptance bullets reflect every absorbed finding.

One non-blocking observation routed as NIT to L1-hygiene (not a W5 defect): L1 `PLAN.md` lines 506 + 510-511 still carry the pre-R10 "implements `tea.Model`" / "pure Bubble Tea v2 models" framing. The R10 locked decision at L1 line 928 supersedes this in spirit (and W5 R2 PLAN.md correctly absorbs the locked decision), but the L1 prose was not actually edited. This is L1-hygiene to clean up at W5 closeout — not a W5 plan defect.

Ready for L2 droplet dispatch.

---

## 1. Round-2 Absorption Verification (per round-1 finding)

### 1.1 Fals FF1.1 CRITICAL — `tea.Model` contract mismatch — **ABSORBED**

- W5 R2 PLAN.md lines 18, 56-63: components reframed as "Bubble Tea **sub-components**" — they do NOT implement `tea.Model`.
- AC3 (line 78-82): "`ConfirmModel` as a Bubble Tea **sub-component** (NOT `tea.Model`)" — has `Update(tea.Msg) (ConfirmModel, tea.Cmd)`, `View() string`, accessor methods.
- AC4 (lines 83-86): same reframe for `PickerMultiModel`.
- D3 AC (lines 460-461): "`TextInputModel` is a Bubble Tea sub-component ... NOT `tea.Model`. No `var _ tea.Model = (...)` assertion."
- D4 AC (lines 559-563): both pickers as sub-components, NOT `tea.Model`.
- Context Block `constraint (critical)` (lines 157-161): "Components are Bubble Tea **sub-components**, NOT standalone `tea.Model` implementations. `View() string` (not `View() tea.View`). No `var _ tea.Model = (*ConfirmModel)(nil)` assertions."
- Reference (lines 203-206): "`tea.Model` in `charm.land/bubbletea/v2@v2.0.0-rc.2/tea.go:52-63` has `View() View` (struct return), NOT `View() string`. W5 components intentionally do NOT implement `tea.Model`."
- **Verified evidence** — `internal/tui/model.go:2027` is `func (m Model) View() tea.View { ... }`; PLAN.md correctly cites the bubbletea v2 rc.2 interface as `View() View`.
- L1 R10 locked decision (L1 PLAN.md line 928) explicitly authorizes this reframe.

### 1.2 Fals FF1.2 HIGH — `tea.Quit` kills parent TUI — **ABSORBED**

- W5 R2 PLAN.md lines 19-20: every `return tea.Quit` in D2/D3/D4 replaced with `return nil`.
- D2 confirm.go Update spec (lines 343-352): each key handler returns `m, nil` with `done=true`; explicit `// **NEVER `return m, tea.Quit`** — this kills the parent TUI.` line 351.
- D3 textinput.go (line 444): `**NEVER `return m, tea.Quit`**`.
- D4 picker_single.go (line 513): `**NEVER `return m, tea.Quit`**`.
- D4 picker_multi.go (line 541): `**NEVER `return m, tea.Quit`**`.
- Test assertions (lines 363, 456, 523, 553): "Assert NO case returns non-nil cmd" — table-driven coverage that `tea.Cmd == nil`.
- Risk Notes R5 (lines 136-139): "Components are SUB-MODELS — they must NOT return `tea.Quit`. The parent TUI polls `Done()` / `Confirmed()` / `Cancelled()` / `Submitted()` accessors..."
- Accessor surface (lines 82, 86): `Done()`, `Confirmed()`, `Cancelled()`, `Submitted()`, `Selected()` are the signal path.

### 1.3 Fals FF1.3 + Proof FF1.1 — loader JSON shape ambiguity — **ABSORBED**

- W5 R2 PLAN.md lines 21-22: ONE pinned shape — nested `{"product_extensions":{"tillsyn":{"commands":[...]}}}` — for BOTH baseline embed AND local file.
- Embedded literal (lines 718-732): `stilBaselineTillsynJSON` in nested-schema form.
- Internal decoder struct (lines 748-757): `bindingsFile` parses `product_extensions.tillsyn.commands` from BOTH inputs.
- `LoadBindings` semantics (lines 758-766): "Parses `baselineJSON` via `bindingsFile`; extracts `product_extensions.tillsyn.commands` (4 baseline entries). If `localPath != ""`: ... ID-deep-merge ... Local wins on collision."
- AC6 (lines 91-97): explicit shape pinned in acceptance; same nested schema for both.
- Context Block `constraint (high)` (lines 162-165): "Baseline Tillsyn commands are embedded as package-level `[]byte` in the nested schema form `{"product_extensions":{"tillsyn":{"commands":[...]}}}`."
- **Verified evidence** — `/Users/evanschultz/Documents/Code/hylla/stil/main/src/bindings/baseline.json:100-108` confirms 4 entries under `product_extensions.tillsyn.commands` — exact shape matches embedded literal at PLAN.md lines 720-731.

### 1.4 Proof FF1.2 LOW — KEYBIND-R3 staleness TODO — **ABSORBED**

- W5 R2 PLAN.md lines 27, 716-717: `// TODO(KEYBIND-R3): refresh embedded stil baseline bytes when stil-solid v<X> publishes` comment required in `loader.go` near `stilBaselineTillsynJSON` declaration.
- AC6 (line 826-827): grep-discoverable comment requirement carried into AcceptanceCriteria.

### 1.5 NITs — 13 absorbed + 3 deferred-with-reason

| # | NIT | Disposition | Evidence in W5 R2 PLAN.md |
|---|---|---|---|
| Proof 2.1 | D1 coverage-gate conditional→imperative | ABSORBED | `palette_test.go` mandatory in D1 Paths (line 256); AC1 lists it (line 71); AC7 says "guaranteed by `palette_test.go`" (line 99-100); D1 Specify (lines 277-281) requires `TestAllColors_NonEmpty` calling `AllColors()`. |
| Proof 2.2 | D5 header/footer coverage fallback | ABSORBED | D5 note (lines 625-630): explicit builder authority to add `header_test.go` / `footer_test.go` if coverage drops below 70%, NOT in L1 paths but within D5's authority. |
| Proof 2.3 | D2 progress.go optional style-import dep | ABSORBED | `progress.go` MUST use inline lipgloss style — importing `internal/tui/style` would add undeclared `blocked_by D1` (Context Block constraint high, lines 167-168; D2 spec lines 376-378). |
| Proof 2.4 | D3 textinput.go marker self-contradiction | ABSORBED | "NOT required again" clause deleted; only hard per-file rule remains (lines 423-424). |
| Proof 2.5 | D6 test marker implicit | CONFIRMED PASS | AC2 (line 73-77) + D6 AC (line 835) explicitly require migration markers in all 4 files including `dispatcher_test.go`. |
| Proof 2.6 | R3/R4 advisory risk notes | NO ACTION | Risk Notes R3 + R4 (lines 127-135) are advisory-only — flagged for awareness, no plan defect. |
| Proof 2.7 | D6 modes.go marker placement | ABSORBED | Two separate `//` comment lines explicitly specified (lines 691-693); AC line 822-824 carries forward. |
| Proof 2.8 | AC8 `mage ci` gate placement | ABSORBED | AC8 (lines 101-103): drop-end gate after BOTH D5 and D6 done; D6 Mage verification updated to include `mage ci` (line 838); D5 note (lines 642-645): explicit not-per-D5-gate. |
| Fals NIT 1.4 | Multi-key sequences under-specified | DEFERRED-AS-NIT | KEYBIND-R4 tracked; `NewDispatcher` skips multi-key registration in `bindings[ModeNav]`; `// TODO(KEYBIND-R4)` comment required (lines 39, 144-145, 180-183, 786-787); `TestDispatcher_MultiKey_Returns_NoOp` added (lines 812-814). |
| Fals NIT 1.5 | Local 5-command file shape unspecified | ABSORBED | Canonical local file shape pinned per REVISION_BRIEF §2.19 (lines 207-211, 767-770); test fixture uses exact shape (lines 800-803). |
| Fals NIT 1.6 | Coverage gate trap (pure-var package) | ABSORBED inline via NIT 2.1 | Same finding; `palette_test.go` mandated in D1 Paths. |
| Fals NIT 1.7 | Migration-marker enforceability — no automated gate | DEFERRED-AS-NIT | Option (b) chosen — per-droplet build-QA-proof checks marker on every file in Paths (AC2 line 73-77, ValidationPlan line 110-111, every droplet AC). Option (a) `mage check-migration-markers` deferred as MIGRATE-MARKER-R1 (lines 42, 184-186). |
| Fals NIT 1.8 | `tea.KeyMsg` vs `tea.KeyPressMsg` codebase inconsistency | ABSORBED | All `case tea.KeyMsg:` in D2/D3/D4 type-switches changed to `case tea.KeyPressMsg:` (lines 169-174, 343, 440, 510, 538); `Dispatch(msg tea.KeyMsg, mode Mode)` parameter stays `tea.KeyMsg` (interface — correct for public API). |
| Fals NIT 1.9 | Dispatch multi-key can't be expressed | DEFERRED-AS-NIT | Same disposition as NIT 1.4 (KEYBIND-R4). `Dispatch` resolves on single `tea.KeyMsg.String()`; multi-key return `NoOp`. |
| Fals NIT 1.10 | progress.go L1/L2 contradiction | ABSORBED | progress.go is passive render-only struct (not sub-component); no `Init()` / `Update()` methods (lines 45, 166-168, 220-223, 366-381). L2 AC reflects this; L1 staleness routed below. |
| Fals NIT 1.11 | Test-file marker placement awkward | ABSORBED | Explicit Context Block note (lines 149-154): on `_test.go` files the marker is a file-level `//` comment immediately before `package <name>`, NOT promoted to package doc. |

**Deferred-with-reason count check:** 3 deferred (NIT 1.4 / 1.7 / 1.9, all tracked as KEYBIND-R4 or MIGRATE-MARKER-R1). Round-2 spawn-prompt said "13 ABSORBED + 2 DEFERRED-with-reason" — actual count is 13 absorbed + 3 deferred + 0 confirmed-pass-no-action + 1 advisory-only = matches the `feedback_nits_are_first_class.md` discipline (every NIT enumerated with explicit disposition).

### 1.6 D1 `palette_test.go` mandate — **VERIFIED**

- D1 Paths (line 256): `internal/tui/style/palette_test.go` (NEW — MANDATORY; see coverage note)
- D1 Specify (lines 277-281): `TestAllColors_NonEmpty` calls `AllColors()` and asserts len > 0, each element non-zero.
- D1 AC (lines 296-299): coverage gate mandated; "`palette_test.go` MUST exist with `TestAllColors_NonEmpty`."
- KindPayload preview (line 850): D1 paths list includes `palette_test.go`.

### 1.7 `case tea.KeyMsg:` → `case tea.KeyPressMsg:` propagation — **VERIFIED**

- D2 confirm.go Update (line 343): "type-switch on `tea.KeyPressMsg` (NOT `tea.KeyMsg`)".
- D3 textinput.go Update (line 440): "type-switch on `tea.KeyPressMsg`".
- D4 picker_single.go Update (line 510): "type-switch on `tea.KeyPressMsg` (NOT `tea.KeyMsg`)".
- D4 picker_multi.go Update (line 538): "type-switch on `tea.KeyPressMsg` (NOT `tea.KeyMsg`)".
- Context Block `constraint (high)` (lines 169-174): explicit codebase-consistency rule + double-fire risk acknowledged; `Dispatch(msg tea.KeyMsg, mode Mode)` parameter correctly remains interface type.
- LSP-verified (cited by R1 reviewer + reverified by R2 reviewer): `internal/tui/model.go:2004` uses `case tea.KeyPressMsg:`.

---

## 2. Other Round-2 Proof Checks

### 2.1 `_BLOCKERS.toml` mirrors PLAN.md — **PASS**

- `_BLOCKERS.toml` lists D3→D2, D4→D3, D5→D4 (3 entries).
- PLAN.md `Blocked by:` bullets: D1 line 262 (none), D2 line 319 (none), D3 line 411 (D2), D4 line 487 (D3), D5 line 586 (D4), D6 line 664 (none).
- D1 + D6 + D2 have no blockers and are correctly NOT in TOML (leading comment at TOML line 4 says so for D1 + D6; D2 also not in TOML, also correct).
- TOML reasons match PLAN.md narrative (all 3 cite "package compile" / "internal/tui/components package" shared-package serialization).

### 2.2 PLAN-QA-DISCIPLINE-R1 — **PASS**

Every NEW-behavior acceptance bullet ships with a test in the same droplet OR a same-package test from a prior droplet:

- D1 (style): `palette_test.go` co-located (NIT 2.1 absorbed — now mandatory).
- D2 (confirm + progress): `confirm_test.go` covers key matrix; progress.go is passive render-only (no behavior to test beyond `View()` smoke; if coverage drops, D5's authority to add smoke tests covers cross-droplet residue).
- D3 (textinput): `textinput_test.go` covers validation + submit.
- D4 (pickers): `picker_single_test.go` + `picker_multi_test.go` co-located; tests cover toggle/navigation/confirm/cancel.
- D5 (header + footer): relies on package-level coverage carry-over from D2/D3/D4 tests; D5 builder has explicit authority to add `header_test.go` / `footer_test.go` smoke tests if coverage drops (NIT 2.2 absorbed).
- D6 (keybindings): `dispatcher_test.go` covers `TestLoadBindings_BaselineOnly` (4 commands), `TestLoadBindings_WithLocal` (9 commands), `TestLoadBindings_LocalWins` (collision), `TestLoadBindings_MissingLocalFile` (graceful fallback), `TestDispatcher_Dispatch`, `TestDispatcher_MultiKey_Returns_NoOp`, `TestDispatcher_Register`, `TestDispatcher_DispatchCommand`.

No bullet describes behavior unsupported by a test.

### 2.3 PLAN-QA-DISCIPLINE-R2 — **PASS**

Narrative count (6) matches enumeration count (D1, D2, D3, D4, D5, D6):

- Objective + AcceptanceCriteria reference 6 droplets implicitly via AC1's path list (line 68-72).
- Validation Plan (line 108): "once all 6 droplets are complete".
- Parallel dispatch graph (lines 230-238): D1, D2, D3, D4, D5, D6 — 6 nodes.
- Droplet section headings (lines 247, 304, 398, 472, 573, 649): exactly 6 `### Dn — …` blocks.
- KindPayload preview (lines 849-855): array of 6 child entries.
- CompletionChecklist (lines 873-884): 12 bullets = 6 droplets × 2 QA passes; final 13th bullet is `mage ci`.

Count = 6 = 6 across all surfaces.

### 2.4 Migration marker discipline — **PASS**

`// MIGRATION TARGET: github.com/hylla-org/lykta` required on every file (D1-D6, production + test):

- D1 (4 files): palette.go line 272, palette_test.go line 277-278, spacing.go line 282, typography.go line 284. AC bullet line 289-291.
- D2 (3 files): confirm.go line 330, confirm_test.go line 357, progress.go line 367. AC bullet line 390.
- D3 (2 files): textinput.go line 422, textinput_test.go line 451. AC bullet line 464-465.
- D4 (4 files): picker_single.go line 499, picker_single_test.go line 519, picker_multi.go line 526, picker_multi_test.go line 548. AC bullet line 565-566.
- D5 (2 files): header.go line 596, footer.go line 613. AC bullet line 636-637.
- D6 (4 files): modes.go line 692, loader.go line 715, dispatcher.go line 774, dispatcher_test.go line 797. AC bullet line 835-836.

Total: 4+3+2+4+2+4 = 19 files (matches AC1 enumeration line 68-72 — wait, AC1 says 10 + 4 + 4 = 18; D1 has 4 files including the mandatory `palette_test.go`, so 11 + 4 + 4 = 19). Verified at AC1 line 68-72: "10 component source files ... 4 style files ... 4 keybinding files" = 18 in AC1 text, but actual enumeration shows 11 components (confirm/confirm_test/textinput/textinput_test/picker_single/picker_single_test/picker_multi/picker_multi_test/header/footer/progress = 11) + 4 style + 4 keybinding = 19.

**Discrepancy raised as NIT below (NIT 2.1).**

AC2 (lines 73-77) + Context Block `constraint (critical)` (lines 149-154) + ValidationPlan AC2-verification (lines 110-111) all enforce per-droplet build-QA-proof check of every file's marker. Per-droplet AC bullets each explicitly say "build-QA-proof checks each file in Paths explicitly" (lines 291, 391, 465, 566, 637, 836).

### 2.5 Acceptance criteria testability — **PASS**

- AC1: file-existence check (build-QA-proof per droplet).
- AC2: grep for marker string in each file in droplet's Paths.
- AC3/AC4: interface inspection — `Init`, `Update`, `View` method signatures + accessor methods.
- AC5: `Dispatch` returns registered handler or `NoOp`; multi-key returns `NoOp`. Test `TestDispatcher_Dispatch` + `TestDispatcher_MultiKey_Returns_NoOp` cover.
- AC6: `LoadBindings` baseline-only + local-merge + collision + missing-file. 4 named tests in dispatcher_test.go cover (line 798-807).
- AC7: `mage test-pkg <pkg>` exit code + ≥70% coverage. Builder runs per-droplet.
- AC8: `mage ci` green. Orch-run drop-end gate.

All acceptance bullets map to a verifiable check.

### 2.6 Decomposition: 6 droplets across 3 new packages — **PASS**

- 3 new packages: `internal/tui/style` (D1), `internal/tui/components` (D2 creates, D3/D4/D5 extend), `internal/tui/keybindings` (D6).
- D1 + D6 are independent packages — no `blocked_by`; can dispatch in parallel with D2.
- D3 → D2 → D4 → D5 serialized via shared `internal/tui/components` package compile.

Decomposition justified per CLAUDE.md atomic-droplet sizing (1-4 code blocks per droplet, ≤120 LOC each). D6 has 3 production files = the "smell" threshold, mitigated per Risk Note R1 (lines 115-120): coherent package concern (loader feeds dispatcher; modes feeds dispatcher), splitting would leave partially-initialized package state across builds.

### 2.7 Wave-Boundary Concerns — **PASS**

Six wave-boundary concerns enumerated (lines 900-927): W2 dispatch readiness (D4 is meaningful unblock); D6 baseline embedding staleness (KEYBIND-R3 TODO); teatest import path; coverage gates for pure-constant packages; AC8 gate ownership; multi-key nav bindings deferred (KEYBIND-R4). Each is informational; none invalidates the plan.

---

## 3. Findings (FF — substantive)

**None.** All 5 round-1 substantive FFs are fully absorbed. Round-2 verification finds no new substantive issues with the plan.

## 4. Missing Evidence (NIT — non-blocking)

- **2.1 [Axis: spec-conformance] [severity: low]** — AC1 file-count drift. AC1 (PLAN.md lines 68-72) says "10 component source files ... 4 style files ... 4 keybinding files" but the enumerated paths list 11 component files (confirm/confirm_test/textinput/textinput_test/picker_single/picker_single_test/picker_multi/picker_multi_test/header/footer/progress = 11), 4 style files (palette/palette_test/spacing/typography), 4 keybinding files = 19 total, not 18 → `evidence pointer:` AC1 text counts 10+4+4=18, enumeration counts 11+4+4=19 → `fix_hint:` change "10 component source files" to "11 component source files" in AC1 line 69. Trivial textual fix; no semantic implication.

- **2.2 [Axis: parallelization-graph] [severity: low]** — L1 PLAN.md staleness. L1 `PLAN.md` line 506 still says "All component implementations are pure Bubble Tea v2 models (Init/Update/View)"; lines 510-511 still say "`confirm.go` implements `tea.Model`" / "`picker_multi.go` implements `tea.Model`". The R10 locked decision at L1 line 928 supersedes this, and W5 R2 PLAN.md correctly absorbs the locked decision, but the L1 prose at 506/510/511 was not edited. W5 R2 PLAN.md's claim "removed from ... the L1 scope note" (PLAN.md line 18) is aspirational — L1 was not actually edited → `evidence pointer:` L1 PLAN.md lines 506, 510-511 vs locked decision line 928 → `fix_hint:` L1-hygiene edit at W5 closeout: replace "implements `tea.Model`" with "is a Bubble Tea sub-component" in L1 lines 506/510/511; not a W5-defect, but if the orch surfaces this in the closeout MD it prevents future planners reading stale L1 prose. Not a blocker for L2 dispatch.

- **2.3 [Axis: spec-conformance] [severity: low]** — `Note` block convention. PLAN.md uses three distinct prose-tag conventions in Context Blocks: `constraint (critical)` (lines 149-161), `constraint (high)` (lines 162-174), `decision` (lines 175-186), `reference` (lines 187-211), `warning (high)` (lines 212-219), `note` (line 220-223). The L1 cascade-design rubric doesn't strictly enforce a closed tag enum, but the mix of severity-tagged constraints + plain "note" / "decision" / "reference" looks ad-hoc → `evidence pointer:` PLAN.md Context Blocks section lines 147-223 → `fix_hint:` no action — flagged for awareness. Consistent with other L2 sub-plans in this drop.

---

## 5. Specific Verifications (round-2 spawn-prompt checks)

| Spawn-prompt check | Verdict | Evidence |
|---|---|---|
| Fals FF1.1 CRITICAL (`tea.Model` reframe) | RESOLVED | §1.1 above |
| Fals FF1.2 HIGH (`tea.Quit` → `nil`) | RESOLVED | §1.2 above |
| Fals FF1.3 + Proof FF1.1 (nested JSON shape for both) | RESOLVED | §1.3 above |
| Proof FF1.2 (KEYBIND-R3 TODO) | RESOLVED | §1.4 above |
| All 16 round-1 NITs absorbed or deferred-with-reason | 13 ABSORBED + 3 DEFERRED-with-reason | §1.5 table |
| D1 `palette_test.go` added (coverage gate) | RESOLVED | §1.6 |
| `case tea.KeyPressMsg:` propagated | RESOLVED | §1.7 |
| `_BLOCKERS.toml` mirrors PLAN.md | PASS | §2.1 |
| PLAN-QA-DISCIPLINE-R1 + R2 (6 droplets, 3 packages, narrative = enumeration) | PASS | §2.2 + §2.3 + §2.6 |
| Migration marker on every file (D1-D6) | PASS (19 files marker'd) | §2.4 |

---

## 6. Cross-planner consistency angles

- W2 depends on W5's `confirm.go` (D2) + `picker_multi.go` (D4). Effective unblock for W2 is D4 close (D2 → D3 → D4 serialized). PLAN.md Risk Note R4 (lines 132-135) covers; W5 closing is necessary but D4 completion is the meaningful unblock milestone for W2 — orch should sequence accordingly.
- D6 keybinding dispatcher embeds stil baseline bytes; no runtime dependency on sibling-repo filesystem (Context Block `constraint (high)` line 162-165). `// TODO(KEYBIND-R3)` flags the future move to stil-solid as package artifact.
- W6 (FE-side vim engine) will independently implement the same merge semantic (per L1 PLAN.md line 935 R10 locked decision) — separate codebase, same source-of-truth schema (`product_extensions.tillsyn.commands`). No W5/W6 coupling beyond shared schema.

---

## TL;DR

- **T1**: PASS — all 5 round-1 substantive FFs fully absorbed; 13 NITs absorbed + 3 deferred-with-reason; no new substantive findings.
- **T2**: Sub-component reframe (`View() string`, NOT `tea.Model`) is correct per `charm.land/bubbletea/v2@v2.0.0-rc.2/tea.go:52-63` + L1 R10 locked decision (line 928). `return nil` (not `tea.Quit`) propagated everywhere. ONE nested JSON schema (`product_extensions.tillsyn.commands`) for both baseline embed and local file, verified against stil baseline.json:100-108.
- **T3**: 3 deferred NITs all tracked as named refinements: KEYBIND-R4 (leader-key state machine for multi-key nav bindings) + MIGRATE-MARKER-R1 (`mage check-migration-markers` target).
- **T4**: `_BLOCKERS.toml` mirrors PLAN.md (D3→D2, D4→D3, D5→D4); D1+D2+D6 correctly unblocked. Topo: `{D1,D2,D6} → {D3} → {D4} → {D5}` — acyclic.
- **T5**: Three small NITs raised (file-count drift at AC1; L1 staleness re: `tea.Model` claims at L1 lines 506/510/511; convention-mix note) — none blocking; L1 staleness routed to L1-hygiene at W5 closeout.

---

## Hylla Feedback

N/A — Hylla is OFF per spawn prompt. All evidence collected via filesystem `Read` of PLAN.md, `_BLOCKERS.toml`, round-1 verdicts, L1 PLAN.md round-10 locked-decisions block, stil `baseline.json`, and `internal/tui/model.go:1995-2034`. Bubbletea v2 rc.2 `tea.Model` interface evidence was carried forward from R1 reviewers' direct citations (sandbox blocked direct re-read of `/Users/evanschultz/go/pkg/mod/charm.land/bubbletea/v2@v2.0.0-rc.2/tea.go:52-63`) and cross-validated against the in-tree `func (m Model) View() tea.View` at `internal/tui/model.go:2027`, which exercises the interface and confirms the `View() View` return type. The cross-validation hop is intentional — same conclusion, two independent evidence sources.
