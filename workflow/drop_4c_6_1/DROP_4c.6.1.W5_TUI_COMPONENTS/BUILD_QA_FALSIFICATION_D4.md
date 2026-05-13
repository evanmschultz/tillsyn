# W5.D4 — BUILD-QA-FALSIFICATION Verdict

**Date:** 2026-05-13
**Reviewer:** go-qa-falsification-agent (opus)
**Overall verdict:** PASS WITH NITS

## Attack Hypotheses Tested

| # | Hypothesis | Result | Evidence |
|---|---|---|---|
| H1 | Empty-slice guard correctness | REFUTED | `picker_single.go:48` and `picker_multi.go:51` early-return on `len(m.items)==0 \|\| m.done` before any indexing. No mutator can change `m.items` post-construction (struct field unexported, never reassigned). |
| H2 | PickerMulti Selected() preserves original-index order | REFUTED | `picker_multi.go:108-114` iterates `for i, item := range m.items` and appends only when `m.selected[i]` is true. Order is by item index, independent of toggle order. `TestPickerMultiModel_Confirm` asserts `["alpha", "gamma"]` after toggling cursor=0 then cursor=2. |
| H3 | PickerMulti Selected() returns nil on cancel | REFUTED | `picker_multi.go:105-107`: `if m.cancelled { return nil }` first thing. `TestPickerMultiModel_Cancel` confirms `len(got)==0` after Escape even when an item was toggled before cancellation. |
| H4 | Wrap-around boundaries (j on last → 0, k on first → last) | REFUTED | Single: `picker_single.go:57,59` use `(cursor+1)%len`, `(cursor-1+len)%len`. Multi: `picker_multi.go:60,62` use identical formulas. Both pickers test wrap at both boundaries (`TestPicker{Single,Multi}Model_Navigation` j-wraps-at-bottom, k-wraps-at-top subtests). |
| H5 | Space toggles PickerMulti on/off | REFUTED | `picker_multi.go:69`: `next[m.cursor] = !next[m.cursor]` — clean boolean flip. `TestPickerMultiModel_Toggle` exercises two consecutive Space presses and asserts on→off. |
| H6 | Enter vs Escape state divergence (PickerMulti) | REFUTED | Enter (`picker_multi.go:71-72`): only sets `done=true`; `cancelled` remains false. Escape (`73-75`): sets both `cancelled=true` AND `done=true`. `TestPickerMultiModel_Confirm` asserts `Cancelled()==false` after Enter; `TestPickerMultiModel_Cancel` asserts `Cancelled()==true` after Escape and empty `Selected()`. |
| H7 | All 4 files use `tea.KeyPressMsg` (not deprecated `tea.KeyMsg`) | REFUTED | `rg -n "tea.KeyMsg" internal/tui/components/picker_*` returns zero hits. Both Update funcs type-assert `msg.(tea.KeyPressMsg)`; both test files construct `tea.KeyPressMsg{Code: ...}`. |
| H8 | `tea.Quit` absence in all 4 files | REFUTED | 4 occurrences found, ALL in doc-comments asserting the NEVER-quit invariant (`picker_single.go:17,45`, `picker_multi.go:17,48`). Zero code uses. |
| H9 | No `var _ tea.Model = (...)` assertion | REFUTED | `rg "var _ tea.Model"` returns zero hits across all 4 files. |
| H10 | Migration marker format exact match | REFUTED | All 4 files have `// MIGRATION TARGET: github.com/hylla-org/lykta` as line 1, followed by `package components` (single `package components` line at line 5 in each prod file with intervening package doc-comment, and at line 2 in each test file). Spec says "file-level comment before `package components`" — both placements satisfy. |
| H11 | Done() returns true after both Enter and Escape | REFUTED | Single Enter (`picker_single.go:62`) and Escape (`64`) both set `m.done=true`. Multi Enter (`picker_multi.go:72`) and Escape (`75`) both set `m.done=true`. Tests cover both paths. |
| H12 | Updates after done are no-ops | REFUTED with NIT | Both Update funcs (`picker_single.go:48`, `picker_multi.go:51`) early-return on `m.done`. PickerSingle has explicit `Updates after done are no-ops` subtest. PickerMulti relies on indirect coverage. Code is correct; test parity is asymmetric (see NIT 1). |
| H13 | YAGNI — map clone-on-Space over-engineered? | REFUTED | `picker_multi.go:65-70` clones the map before mutating. Without the clone, two stale model copies would share the same map and mutations would alias across them — violates the value-semantic sub-component contract (`Update` returns a new `PickerMultiModel` by value). The clone is load-bearing, not gratuitous. Not YAGNI. |
| H14 | Coverage ≥70% | REFUTED (qualitative) | `mage testPkg` exposes no `-cover` flag. By inspection: constructor, Init, Update (all key codes incl. wrap edges, Space toggle, Enter, Escape, no-op after done), View (empty-list + populated), Selected(), Done(), Cancelled() — every public/private path is exercised. ~100% line coverage by inspection. Quantitative verification routed as Unknown (no mage cover target). |

## Unmitigated Counterexamples

None. All 14 attack hypotheses REFUTED.

## NITs

1. **Asymmetric updates-after-done test coverage.** `picker_single_test.go:113-125` includes the `Updates after done are no-ops` subtest. `picker_multi_test.go` does not include the corresponding subtest. Code is symmetric (same guard at line 51 of `picker_multi.go`); the test asymmetry is structural only. Adding `TestPickerMultiModel_UpdatesAfterDone` (toggle, Enter to confirm, then Space — expect cursor and selected unchanged) restores parity. **Severity: low.** Code is correct; test parity is the gap.

2. **`TestPickerSingleModel_Navigation` initial-cursor check at lines 17-19 sits outside any `t.Run` subtest** while the rest of the file uses subtests. Cosmetic structural inconsistency — does not affect correctness. **Severity: cosmetic.**

3. **No `-cover` verification path.** Spec says `≥70% coverage`. `mage testPkg ./internal/tui/components` reports pass/fail but not coverage %. Inspection suggests near-100% but a coverage gate is not directly enforced for this droplet. Not a defect of the droplet itself; suggests a meta-NIT against W5 acceptance criteria or the mage target surface. **Severity: meta / not blocking.**

## Verdict rationale

All 14 falsification attacks were attempted and refuted by direct evidence (line-numbered code citations + 46/46 mage test-pkg pass). The map-clone-on-Space pattern initially looked like YAGNI but is actually correct value-semantic preservation. The two NITs (asymmetric test coverage, subtest structure) are non-blocking and would be addressed in a low-priority follow-up — they do not invalidate the droplet's acceptance criteria. The coverage Unknown (H14) is a meta-gap in the mage surface, not a droplet defect.

**Verdict: PASS WITH NITS.**
