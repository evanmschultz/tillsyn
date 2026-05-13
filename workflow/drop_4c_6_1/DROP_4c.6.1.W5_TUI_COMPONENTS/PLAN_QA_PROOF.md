# PLAN_QA_PROOF — Drop 4c.6.1.W5 (TUI Components + Style + Vim Keybinding Dispatcher)

**Round:** 1
**Reviewer:** L2 plan-QA proof agent
**Mode:** Filesystem-MD-only; Hylla off
**Target:** `workflow/drop_4c_6_1/DROP_4c.6.1.W5_TUI_COMPONENTS/PLAN.md` + `_BLOCKERS.toml`

---

## Verdict

**PASS WITH FINDINGS** — the 6-droplet decomposition is structurally sound, evidence-grounded, and the `blocked_by` graph is acyclic with all shared-package pairs (D2→D3→D4→D5 on `internal/tui/components`) correctly serialized. D1 (style) and D6 (keybindings) live in independent new packages and correctly carry no blockers. Acceptance criteria are testable; `mage test-pkg` exists at `magefile.go:31,47` with `coverageThreshold = 70.0` at line 38. The migration-marker hard rule (AC2, Context-Block `constraint (critical)`) is consistently applied as a per-droplet acceptance bullet. Narrative count (6) matches D1–D6 enumeration.

Findings below are **non-blocking clarifications + one substantive ambiguity** in D6's baseline-JSON shape contract. None invalidates the plan; round 2 (or builder per-judgment) resolves.

---

## 1. Findings (FF — substantive, must address)

- 1.1 [Axis: spec-conformance] [severity: medium] **D6 baseline-JSON shape contract is ambiguous** → PLAN.md lines 524-536 specify the embedded `stilBaselineTillsynCommands` as a **flat 4-element JSON array** of `Command` entries, but the stil baseline at `/Users/evanschultz/Documents/Code/hylla/stil/main/src/bindings/baseline.json:100-108` ships the 4 Tillsyn commands **nested** under `product_extensions.tillsyn.commands`. REVISION_BRIEF §2.19 (lines 416-428) specifies the **Tillsyn-local** file uses the same **nested** shape. PLAN.md line 587 says "Parses `baselineJSON` as `{"commands": [...]}` or equivalent baseline shape" — but the embedded literal in R2 is a flat array (no `{"commands": ...}` wrapper). The loader must handle both shapes consistently, AND the embedded `DefaultBaselineJSON()` bytes need a definite shape. → **fix_hint**: pin the embedded bytes to ONE shape (recommend the same nested `{"product_extensions":{"tillsyn":{"commands":[...]}}}` shape that the stil source uses + matches the local file shape), and update L2 plan's `LoadBindings` semantics to say "extracts `product_extensions.tillsyn.commands` from both baseline and local payloads, then ID-merges." This keeps the loader symmetric across baseline and local inputs and makes the future stil-solid migration mechanical (KEYBIND-R3).

- 1.2 [Axis: parallelization-graph] [severity: low] **D6 has zero `blocked_by` and shares no package with D1-D5 — confirmed correct, but PLAN.md's `Wave-Boundary Concerns §2` (line 727) warns the embedded baseline bytes will go stale if stil updates** → PLAN.md acknowledges this as an "accepted known risk" via KEYBIND-R3. → **fix_hint** (optional): add a `// TODO(KEYBIND-R3)` doc-comment requirement in D6's acceptance so the staleness link is grep-discoverable in code.

## 2. Missing Evidence (NIT — nice-to-have, not blocking)

- 2.1 [Axis: acceptance-criteria-coverage] [severity: low] **D1 coverage-gate mitigation phrased as builder-choice not hard requirement** → PLAN.md lines 181-185 say "If coverage < 70% because there are no testable functions, add one exported accessor function with a test, e.g. `AllColors() []lipgloss.Color`." This is conditional language ("if coverage < 70%"). Since `internal/tui/style` IS pure constants, the gate **WILL** fail unless a testable function is shipped. → **fix_hint**: tighten D1 acceptance bullet 2 from conditional to imperative — "Add at least one testable function (e.g. `AllColors() []lipgloss.Color`) with a co-located test that exercises every exported declaration; otherwise `mage test-pkg ./internal/tui/style` fails coverage gate." Optional `<style>_test.go` file path should be added to D1's Paths section if mandated.

- 2.2 [Axis: acceptance-criteria-coverage] [severity: low] **D5 header/footer coverage carry-over depends on test inheritance, not explicit per-file coverage** → PLAN.md lines 474-479 say "header.go and footer.go have no co-located test files... Builder should verify coverage stays ≥70% with the full package's test suite." This relies on the package-wide weighted coverage from D2/D3/D4 tests carrying header/footer's untested LOC under 30% of the package. With 11 files in `internal/tui/components` (confirm, progress, textinput, picker_single, picker_multi, header, footer + 4 test files), the math probably works — but it isn't enforced. → **fix_hint**: keep the "if needed, add `TestHeader_View` / `TestFooter_View` smoke tests" clause from PLAN.md line 478-479, and add an explicit acceptance bullet "If `mage test-pkg ./internal/tui/components` coverage drops below 70% after D5, ship minimal `header_test.go` + `footer_test.go` smoke tests — these are NOT in the L1 paths list but are within D5's authority to create."

- 2.3 [Axis: spec-conformance] [severity: low] **D2 progress.go's optional dep on `internal/tui/style`** → PLAN.md lines 252-255 say `progress.go`'s `View()` "renders the message with a `Label` style from `internal/tui/style` (or inline lipgloss style if the builder prefers to avoid a dep on `internal/tui/style` at this stage — either is fine)." D2 has no `blocked_by D1`, but if the builder chooses to import `internal/tui/style`, D2 needs D1 to compile. → **fix_hint**: either (a) make D2's `progress.go` import-style-package conditional explicit — "MUST use inline lipgloss style; depending on `internal/tui/style` would add a `blocked_by D1` not currently in the plan"; OR (b) add `D2 blocked_by D1` (and accept the parallelism loss). Option (a) is preferable — it preserves D1/D2 parallelism and keeps the L1 wave graph intact. Same concern applies to D5's `header.go`/`footer.go` (PLAN.md lines 458-459, 471).

- 2.4 [Axis: spec-conformance] [severity: low] **D3 textinput.go migration-marker self-contradiction** → PLAN.md lines 295-299 say "Package-doc migration marker NOT required again (already on `confirm.go`'s `package` declaration). File-level... doc-comment is sufficient... however per the hard rule 'every file carries `// MIGRATION TARGET: ...`', apply the migration marker as a file-level comment immediately before the package declaration in EVERY file. Builder must comply." The text contradicts itself: first says "not required again" then mandates per-file marker. → **fix_hint**: delete the "not required again" clause; keep only the hard rule ("every file carries the marker as a file-level comment immediately above the package declaration"). The hard rule is restated in Context Blocks `constraint (critical)` (PLAN.md line 83-85) and matches AC2 (line 29-31).

- 2.5 [Axis: acceptance-criteria-coverage] [severity: low] **D6 `dispatcher_test.go` migration marker requirement implicit only via AC2** → PLAN.md line 637 includes "Migration markers present in all 4 files" under D6 AcceptanceCriteria — good. Test-file marker is on file lines 614 + 320 + 381 etc. Confirm L2 plan-QA falsification sibling attacks any file lacking the marker; treat consistent with proof side. (No action needed; flagged for completeness.)

- 2.6 [Axis: parallelization-graph] [severity: low] **R3 (header/footer coverage) and R4 (W2 critical path D2→D3→D4) are non-actionable risk notes** — PLAN.md lines 71-79 — these are advisory only (orchestrator dispatch optimization for R4; coverage-watch for R3). Spawn-prompt's "cross-planner consistency angles for W5" already flags the W2-via-D4 unblock point separately. → **No action needed**; flagged for awareness.

- 2.7 [Axis: spec-conformance] [severity: low] **D6 `modes.go` migration marker placement specification** → PLAN.md line 546 says "Package-doc: `// Package keybindings provides a vim-style keybinding dispatcher for Tillsyn's TUI.`" followed by "MIGRATION TARGET marker immediately above package declaration." Two distinct doc-comment lines above `package keybindings`. Builder needs to know if they should be separate comments (// Package … \n// MIGRATION TARGET: …) or one block. → **fix_hint** (optional): one block is conventional Go (separate comments above `package` get attached to package doc; the migration marker becomes part of package doc). Either is fine; consistency-spec recommended.

- 2.8 [Axis: acceptance-criteria-coverage] [severity: low] **AC8 `mage ci` gate placement** → PLAN.md line 47 says "AC8 — `mage ci` green after all droplets complete." D5's "Mage verification" (line 489) ties it to the final component droplet ("`mage test-pkg ./internal/tui/components` then `mage ci`"). D6 (line 640) only specifies `mage test-pkg ./internal/tui/keybindings`. → **fix_hint**: clarify whether `mage ci` is run after D5 alone (since D6 may complete before D5) OR after both D5 AND D6 in `done` state. Recommend explicit "after both D5 and D6 are `done`, the orchestrator runs `mage ci`; AC8 is a drop-end gate, not a per-droplet gate." (Already implicit in CompletionChecklist line 684.)

## 3. Specific Verifications (every spawn-prompt check)

- 3.1 **Migration-marker discipline (spawn-prompt check 8)** — PASS. AC2 (PLAN.md lines 29-31), Context Blocks `constraint (critical)` (lines 83-85), every droplet's per-file specify block carries the marker as an explicit acceptance bullet. D2 confirm.go line 264; D3 textinput.go line 330 (modulo finding 2.4 self-contradiction); D4 all 4 picker files lines 364, 381, 401, 404 (per spec); D5 header/footer lines 446, 462; D6 all 4 files lines 566, 596, 614, plus modes.go line 547. Marker text exact: `// MIGRATION TARGET: github.com/hylla-org/lykta`.

- 3.2 **Coverage-gate mitigation for D1 (spawn-prompt check 9)** — PARTIAL. PLAN.md lines 181-185 mention the mitigation (add `AllColors() []lipgloss.Color` accessor + test) but as conditional language not hard requirement. See finding 2.1.

- 3.3 **Bindings merge — ID-based deep merge baseline 4 + local 5 = 9 (spawn-prompt check 10)** — PASS structurally (merge semantic documented + 4 dedicated tests: `TestLoadBindings_BaselineOnly` 4 cmds, `TestLoadBindings_WithLocal` 9 cmds, `TestLoadBindings_LocalWins` collision wins, `TestLoadBindings_MissingLocalFile` 4 cmds + nil err — PLAN.md lines 616-624). Underlying JSON-shape ambiguity in finding 1.1.

- 3.4 **PLAN-QA-DISCIPLINE-R1 (every NEW-behavior acceptance bullet has test-runner blocked_by shipping it)** — PASS. Every droplet that ships behavior carries its own co-located test file in `Paths`:
  - D1: pure-constants package; mitigation = add test function (finding 2.1 tightens).
  - D2: `confirm_test.go` co-located; tests table-cover key matrix.
  - D3: `textinput_test.go` co-located; tests cover validation + submit.
  - D4: `picker_single_test.go` + `picker_multi_test.go` co-located; tests cover toggle/navigation/confirm/cancel.
  - D5: relies on package coverage carry-over (finding 2.2 tightens).
  - D6: `dispatcher_test.go` co-located; 6 named tests cover baseline-only, with-local, local-wins, missing-local, dispatch, register.
  No bullet describes new behavior that isn't exercised by a test in the same droplet OR a same-package test from a prior droplet.

- 3.5 **PLAN-QA-DISCIPLINE-R2 (narrative count matches enumeration)** — PASS. PLAN.md narrative claims 6 droplets in Objective (line 9 "6 droplets"), Validation Plan (line 52 "6 droplets"), CompletionChecklist (lines 673-684 lists D1-D6), KindPayload preview (lines 648-655 lists D1-D6). Enumeration sections `### D1 — D6` produce exactly 6 sections (lines 137, 191, 271, 337, 424, 493). Count = 6 = 6. ✓

- 3.6 **`_BLOCKERS.toml` mirrors PLAN.md (spawn-prompt check 7)** — PASS. `_BLOCKERS.toml` lists D3→D2, D4→D3, D5→D4 (3 entries). PLAN.md `Blocked by:` lines 207 (D2: none), 285 (D3: D2), 354 (D4: D3), 439 (D5: D4), 510 (D6: none), 152 (D1: none). D1 + D6 have no blockers and are correctly NOT in the TOML (per the leading comment at TOML line 4). D2 also no blockers and not in TOML.

- 3.7 **Acceptance testability + `mage test-pkg` existence** — PASS. `magefile.go:31` aliases `test-pkg` to `TestPkg` function at `magefile.go:47`. `coverageThreshold = 70.0` enforced at line 38. All three new packages (`internal/tui/style`, `internal/tui/components`, `internal/tui/keybindings`) are addressable via `mage test-pkg <pkg-path>`.

- 3.8 **Trace coverage** — PASS. Wave graph (PLAN.md lines 121-131) topo-sorts: `{D1, D2, D6}` → `{D3}` → `{D4}` → `{D5}`. No cycle. `_BLOCKERS.toml` mirrors exactly. Spawn-prompt "cross-planner consistency angle" — D2 (confirm.go) + D4 (picker_multi.go) as W2's effective unblock-point — is correctly an orchestrator-dispatch optimization concern, not a planning defect (L1 wave-level block at `4c.6.1.W2 → 4c.6.1.W5` confirmed PLAN.md L1 line 821).

- 3.9 **Embedded baseline (not sibling-repo filesystem path) at runtime** — PASS. R2 (PLAN.md lines 64-70) + Context Blocks `constraint (high)` (lines 90-92) explicitly forbid sibling-repo filesystem dependency at runtime; baseline bytes embedded as package-level `[]byte`; only `<project>/.tillsyn/bindings.json` is read from the live filesystem. The reference path at PLAN.md line 98 is explicitly labeled "for local reading during development."

- 3.10 **Bubble Tea v2 / Bubbles v2 / Lip Gloss v2 / teatest_v2 deps exist in go.mod** — PASS:
  - `charm.land/bubbles/v2 v2.0.0-rc.1` (go.mod:20)
  - `charm.land/bubbletea/v2 v2.0.0-rc.2` (go.mod:21)
  - `charm.land/lipgloss/v2 v2.0.2` (go.mod:23)
  - `github.com/charmbracelet/x/exp/teatest/v2` (go.mod:29) with `replace ... => ./third_party/teatest_v2` (go.mod:11)
  - `tea.Model` interface signature `Init() Cmd; Update(Msg) (Model, Cmd); View() string` matches Bubble Tea v2 docs (Context7 verified) and PLAN.md AC3/AC4 (lines 32-37) + per-droplet Update signatures.

- 3.11 **Stil baseline 4-command shape verified** — PASS. `/Users/evanschultz/Documents/Code/hylla/stil/main/src/bindings/baseline.json:100-108` confirms 4 entries under `product_extensions.tillsyn.commands`:
  - `new-drop` (keys `["Space","n"]`)
  - `complete-drop` (keys `["Space","c"]`)
  - `handoff` (command `handoff`)
  - `comment` (command `comment`)
  These match PLAN.md lines 524-531 baseline-bytes content (modulo finding 1.1 shape-wrapping ambiguity).

---

## 4. Cross-planner consistency angles

- W2 depends on W5's `confirm.go` (D2) + `picker_multi.go` (D4). W5's effective unblock-point for W2 is D4 close (covers D2 + D3 + D4 serial). L1 plan blocks at wave level (line 821 `4c.6.1.W2 → 4c.6.1.W5`). Not a planning defect; flagged for orchestrator dispatch awareness (PLAN.md R4 line 76-79 already covers).
- D6 keybinding dispatcher consumes stil baseline. L2 plan correctly uses embedded bytes (per R2 + Context Block `constraint (high)`), not sibling-repo filesystem path at runtime.

## Section 0 — SEMI-FORMAL REASONING (rendered in chat response only per orchestrator-facing directive; this MD captures verdict + findings)

## TL;DR
- T1: Verdict — PASS WITH FINDINGS. 2 substantive (1.1 D6 JSON shape; 1.2 staleness TODO) + 8 NITs.
- T2: Decomposition + wave graph + `_BLOCKERS.toml` are correct; narrative count = 6 = enumeration.
- T3: Migration-marker discipline applied per-droplet; one self-contradiction in D3 narrative (finding 2.4).
- T4: D1 coverage-gate mitigation is conditional; tighten to imperative (finding 2.1).
- T5: All PLAN-QA-DISCIPLINE-R1 (test-runner-ships-with-behavior) + R2 (count = 6) PASS.
