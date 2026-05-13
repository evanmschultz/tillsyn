# PLAN QA Falsification — W5 (TUI Components + Style + Vim Keybinding Dispatcher)

**Round:** 1
**Verdict:** **FAIL** — 3 CONFIRMED FFs (one critical contract mismatch, two material under-specs) + 8 NITs. Plan needs absorption pass before L2 droplet dispatch.

## 1. Findings

### 1.1 FF — Severity: CRITICAL — `tea.Model` contract mismatch: `View()` returns the wrong type

**Trace (counterexample):**
- The codebase pins `charm.land/bubbletea/v2 v2.0.0-rc.2` (go.mod line 21).
- In that package (`/Users/evanschultz/go/pkg/mod/charm.land/bubbletea/v2@v2.0.0-rc.2/tea.go:52-63`), the `Model` interface is:
  ```go
  type Model interface {
      Init() Cmd
      Update(Msg) (Model, Cmd)
      View() View          // <-- returns the View struct, NOT string
  }
  ```
  where `tea.View` is a struct (line 120) with `Content Layer`, `Cursor *Cursor`, etc.
- The existing in-tree TUI confirms this: `internal/tui/model.go:2027` is `func (m Model) View() tea.View { ... }` (not `string`).
- PLAN.md repeatedly specifies `View() string` for the new component models:
  - D2 confirm.go (lines 233, 251): `View() string` — renders the prompt with `[y/N]` or `[Y/n]`.
  - D2 progress.go (line 251): `View() string` — renders the message.
  - D3 textinput.go (lines 313-316): `Delegates Init(), Update(), View() to inner` — `inner.View()` returns `string` (bubbles v2 textinput, verified at `bubbles/v2@v2.0.0-rc.1/textinput/textinput.go:684` `func (m Model) View() string`).
  - D4 picker_single.go (line 378): `View() string`.
  - D4 picker_multi.go (line 400): `View() string`.
  - D5 header/footer (lines 459, 471): `View() string`.
- Yet ACs claim these "implement `tea.Model`":
  - AC3: "`confirm.go` implements `tea.Model` (Init/Update/View)".
  - AC4: "`picker_multi.go` implements `tea.Model`".
  - D3 AC: "`TextInputModel` implements `tea.Model`."
  - D4 AC: "`PickerSingleModel` implements `tea.Model`; … `PickerMultiModel` implements `tea.Model`."
  - L1 PLAN.md (line 470): "All component implementations are pure Bubble Tea v2 models (Init/Update/View)."

**Why this is load-bearing:**
A method `View() string` does NOT satisfy `tea.Model` (which requires `View() View`). Builders following PLAN.md literally will produce types whose acceptance criterion "implements `tea.Model`" is false. A trivial `var _ tea.Model = (*ConfirmModel)(nil)` static check would fail at compile time. The L2 plan would only compile if the builder silently deviates from the spec.

**Two valid fixes (planner picks one):**
1. **Sub-model pattern (likely intent):** Components are not standalone `tea.Program`-runnable models; they are sub-views called from inside the outer `Model.View() tea.View`. They legitimately return `string`. **Remove the "implements `tea.Model`" claim from AC3/AC4 and from D3/D4 ACs and from L1 line 470.** Spec their interface as "Has `Update(tea.Msg) (X, tea.Cmd)`, `View() string`, and accessors" — i.e. a Bubble Tea **sub-component** matching the bubbles v2 pattern (where `textinput.Model.View()` also returns `string`).
2. **Top-level program pattern:** If these are intended to be runnable via `tea.NewProgram(component).Run()`, change every `View() string` to `View() tea.View` and add `tea.NewView(...)` wrappers. AC stays; spec text changes.

**Disposition (planner picks):** Option 1 is the correct intent per the codebase (`internal/tui/model.go` is the singular top-level program; everything inside renders to `string` and gets composed). Recommended absorption: drop the `tea.Model` claim and replace with "Bubble Tea sub-component pattern: `Update(tea.Msg) (X, tea.Cmd)` + `View() string` + accessors." Apply at AC3, AC4, D3 AC, D4 AC, and L1 line 470.

---

### 1.2 FF — Severity: HIGH — `tea.Quit` inside reusable sub-components kills the entire TUI

**Trace (counterexample):**
- L1 PLAN.md line 467: "All TUI components route key events through this dispatcher (the W2 `runInitTUI` refactor uses vim-style keys consistent with the rest of the TUI)." This makes clear the components are sub-models inside a larger Bubble Tea program (`runInitTUI`), NOT standalone programs.
- L2 D2 confirm.go (lines 230-233): `y/Y → confirmed=true, done=true, return tea.Quit`; `n/N → cancelled=true, return tea.Quit`; `Enter → return tea.Quit`; `Escape → return tea.Quit`.
- L2 D3 textinput.go (line 319): `Enter key in Update → calls validate; if passes: submitted=true, returns tea.Quit`.
- L2 D4 picker_single.go (line 377): `Enter confirms ... done=true, returns tea.Quit; Escape → done=true ... returns tea.Quit`.
- L2 D4 picker_multi.go (lines 397-399): same `tea.Quit` pattern on Enter/Escape.

**Why this is broken:**
`tea.Quit` is documented as the program-termination signal — it tells the **entire** `tea.Program` to exit. When `runInitTUI` embeds `ConfirmModel` as a step in a multi-step flow (`.mcp.json` confirm prompt → next step), the user pressing `y` would terminate the whole `till init` TUI, not advance to the next step. The correct sub-component pattern is to set internal `done`/`confirmed` state and return `nil` (no Cmd), and let the parent's `Update` poll `m.confirm.Done()` and route to the next step.

**Disposition:** Replace every `return tea.Quit` in D2/D3/D4 component `Update` methods with `return nil` (or a small `func() tea.Msg` that emits a custom `ConfirmDoneMsg`/`PickerDoneMsg`). Add a sentence to the D2/D3/D4 risk notes: "Components are SUB-MODELS — do not call `tea.Quit`. Set internal state; parent unwinds via accessors or custom done-msg." This is consistent with the planner's likely intent (per ConfirmModel having `Done()` accessors) but the literal spec disagrees.

---

### 1.3 FF — Severity: HIGH — Loader spec is shape-ambiguous; baseline-embed JSON ≠ local-file JSON

**Trace (counterexample):**
- D6 loader spec (lines 583-592): "`LoadBindings(baselineJSON []byte, localPath string) (Bindings, error)`. Parses `baselineJSON` as `{"commands": [...]}` or equivalent baseline shape; extracts the 4 Tillsyn commands."
- D6 embedded baseline literal (lines 525-531) is a **bare array** of 4 entries (no top-level `{"commands": [...]}` wrapper, and no `{"product_extensions": {"tillsyn": {"commands": [...]}}}` wrapper):
  ```json
  [
    {"id":"new-drop", ...},
    {"id":"complete-drop", ...},
    {"id":"handoff", ...},
    {"id":"comment", ...}
  ]
  ```
- The local file `<project>/.tillsyn/bindings.json` per SKETCH §10 and REVISION_BRIEF §2.19 carries `product_extensions.tillsyn.commands` (verified shape — stil's actual `baseline.json` at `/Users/evanschultz/Documents/Code/hylla/stil/main/src/bindings/baseline.json` lines 100-108 has the nested-object shape `{"description": "...", "extends": "stil-baseline", "commands": [...]}`).
- L2 plan says the loader "ID-deep-merges the local `product_extensions.tillsyn.commands`" — implying the loader navigates the nested path inside the local file.
- So: baseline is parsed as a bare-array (or `{"commands":[...]}` wrapper — the spec offers BOTH "or equivalent" without picking), local is parsed via the nested `product_extensions.tillsyn.commands` path. The two shapes are different; the spec doesn't commit to which.

**Why this is broken:**
- A builder following the embedded literal (bare array) will write a decoder for `[]Command`. A builder following the spec sentence ("`{"commands": [...]}`") will write a decoder for an object wrapper. A builder reading "local has `product_extensions.tillsyn.commands`" will write a third decoder. Three different builders produce three different (incompatible) loaders. The `TestLoadBindings_BaselineOnly` test fixture isn't pinned to a shape, so any of the three passes its own test.
- This is the exact class of drift PLAN-QA-DISCIPLINE-R1 is designed to catch: "for every acceptance bullet that asserts NEW behavior, is the shape of the input pinned?" AC6 says "loads baseline 4 Tillsyn commands from `baselineJSON`" without pinning baseline-JSON shape.

**Disposition (planner picks ONE shape and propagates):**
- **Option A (preferred):** Both baseline and local file use the same SHAPE — namely the stil baseline's full schema `{"product_extensions":{"tillsyn":{"commands":[...]}}}` (or the `[product_extensions.tillsyn]` subset `{"description":"...","extends":"stil-baseline","commands":[...]}`). The embedded baseline literal becomes the nested-object form. One decoder, one shape. **Bonus:** the embedded copy is trivially refresh-able by copy-pasting from stil's `baseline.json`.
- **Option B:** Both are bare arrays of `Command`. The local file becomes `<project>/.tillsyn/bindings.json` = `[{"id":"dispatch",...}, ...]`. This drops the `product_extensions.tillsyn` framing and decouples Tillsyn's local file from stil's schema. Documented departure from stil shape; KEYBIND-R3 (future move to stil-side `product_extensions.tillsyn`) becomes a non-trivial rename.

Whichever the planner picks, pin it in AC6 and the embedded literal: write "baselineJSON is `{...exact shape with three example entries...}`" and "local file is `{...exact shape with three example entries...}`" — same shape if Option A, different if Option B.

---

### 1.4 NIT — Multi-key sequences (`["Space", "n"]`) are 50% of the baseline and entirely under-specified in `Dispatch`

**Trace:** The verified stil baseline `product_extensions.tillsyn` has 4 commands: `new-drop` (`["Space", "n"]`), `complete-drop` (`["Space", "c"]`), `handoff` (`command:"handoff"`, no keys), `comment` (`command:"comment"`, no keys). Two of four are multi-key. The D6 dispatcher spec (line 605) says: "for commands with `Keys`, registers `keys[0]` (or multi-key prefix if needed) in `bindings[ModeNav]`" — "or multi-key prefix if needed" is a hand-wave. `Dispatch(msg tea.KeyMsg, mode Mode) HandlerFunc` (line 608) only consumes a single `tea.KeyMsg`. There's no state for a pending leader-prefix (e.g., after `Space`, wait for the next key). The test `TestDispatcher_Dispatch` only covers single-key registration, never `Space n`.

**Disposition:** Either (a) pin a concrete pending-prefix state machine in `dispatcher.go` and add a test case `TestDispatcher_MultiKey_LeaderSpace_n` to `dispatcher_test.go`, or (b) explicitly defer multi-key to a refinement (e.g., add `KEYBIND-R4`) and have `NewDispatcher` register multi-key commands as `commands[id]` (command-mode only, no nav-mode key) until then. Pick one in absorption.

---

### 1.5 NIT — Local 5-command file shape is unspecified (and the test fixture is hand-waved)

**Trace:** D6 spec (line 540): "The 5 Tillsyn-local commands ... `dispatch`, `plan`, `archive`, `settings`, `help`. These are loaded from the local file if present." But neither L2 PLAN.md nor SKETCH §10 nor REVISION_BRIEF §2.14 commits to the per-entry shape of these 5 commands — do they have `keys` (which would conflict with stil's `complete-drop`/`new-drop` Space-leader namespace)? Do they have only `command:` and live in command-mode only? The test `TestLoadBindings_WithLocal` says only "provide a temp file with 5-command JSON; result has 9 commands" — without saying what the JSON looks like.

**Disposition:** Add to D6 spec the canonical local-file contents (e.g., all 5 are `command:"<id>"` palette-only, no keys conflict). Pin to the test fixture in `dispatcher_test.go`. Cross-reference with §2.19 (which presumably owns the project-local bindings.json schema — confirm).

---

### 1.6 NIT — Coverage gate trap on `internal/tui/style`: planner mitigation is too vague

**Trace:** Magefile (line 38) enforces `coverageThreshold = 70.0`; line 366-367 errors out if NO coverage rows are parsed (`"no coverage rows were parsed from go test output"`). Pure package-level `var Primary = lipgloss.NewStyle()...` doesn't generate `coverage:` lines — `go test -cover` reports `[no test files]` (no regex match → no row → magefile errors).

The plan's D1 mitigation (line 173): "if coverage < 70% because there are no testable functions, add one exported accessor function with a test, e.g. `AllColors() []lipgloss.Color`" — but a function with body `return []lipgloss.Color{Primary, Accent, ...}` is one **statement**. A test that calls it exercises that one statement = 100% of 1 statement = passes 70%. So the mitigation works **only if** the builder reads "add a function with at least one statement AND a test that calls it." The plan currently says "add ... a function ... with a test." The 1-statement-floor is implicit; a builder might add a function and forget the test (because the magefile coverage stage runs at `mage ci` time, not at `mage test-pkg` time, and a builder running `mage test-pkg ./internal/tui/style` might see `[no test files]` and shrug).

**Disposition:** Make the mitigation concrete and binding: D1 must include `palette_test.go` (or equivalent) with `TestAllColors_NonEmpty` (or equivalent) calling the accessor. Add to D1 paths: `internal/tui/style/palette_test.go` (NEW). Re-check D1 AC1/AC2 path enumeration to include this file.

---

### 1.7 NIT — Migration-marker enforceability has no automated gate

**Trace:** AC2 says "Every file carries `// MIGRATION TARGET: github.com/hylla-org/lykta`" but there's no `mage` target or grep gate that enforces this. The planner has noted (line 84-86): "Plan-QA falsification will attack any file missing this marker" — meaning enforcement falls to human/LLM review. With 18 new files across W5 (3 style + 11 components incl tests + 4 keybindings), one missed marker is a real risk; build-QA agents on different droplets (D1-D6) won't see each other's files to enforce consistency.

**Disposition:** Two options:
- (a) **Add a one-line grep gate** in a new `mage check-migration-markers` target or as part of `mage ci`. E.g. `grep -L "MIGRATION TARGET: github.com/hylla-org/lykta" internal/tui/components/*.go internal/tui/style/*.go internal/tui/keybindings/*.go` → empty output ⇒ pass. Hook in `mage ci` so it gates with the rest. Effort: ~10 LOC in magefile.go, justifies one new droplet OR fits in D5 (last components droplet) which is already the `mage ci` gate.
- (b) **Add the gate as a build-QA-proof bullet** explicitly enumerated per droplet: "Verify every file in your droplet's `Paths` list contains `// MIGRATION TARGET: github.com/hylla-org/lykta` before package declaration." Less automated but cheaper.

Option (a) is the durable answer (extends to W6 FE migration markers). Option (b) is the cheap answer. Planner picks.

---

### 1.8 NIT — `case tea.KeyMsg:` is interface-type; existing codebase uses `tea.KeyPressMsg`

**Trace:** Bubbletea v2 (`tea.go:259`) defines `KeyMsg` as an **interface** (`fmt.Stringer + Key() Key`); concrete event types are `KeyPressMsg` (line 191) and `KeyReleaseMsg` (line 224). The existing codebase exclusively uses `case tea.KeyPressMsg:` (e.g. `internal/tui/model.go:2004`). The L2 PLAN.md `Update` switches type-switch on `case tea.KeyMsg:` (line 228 D2, line 318 D3, line 376 D4) — this compiles (a type-switch CAN match interface types) but matches BOTH key-press AND key-release, which is almost certainly not the intent (a y/n confirm reacting to BOTH press AND release would double-fire).

**Disposition:** Change every `case tea.KeyMsg:` in D2/D3/D4 specs to `case tea.KeyPressMsg:` for consistency with the codebase AND to avoid double-fire on key-release. Note that `dispatcher.go`'s parameter type `tea.KeyMsg` (line 608) is the **right** choice (interface allows callers to pass either key-press or key-release) — only the type-switch should narrow.

---

### 1.9 NIT — `Dispatcher.Dispatch` resolves on `msg.String()` but the multi-key case can't be expressed

**Trace:** Already partly covered in 1.4 but worth a separate disposition note. `Dispatch(msg tea.KeyMsg, mode Mode)` (line 608-609): "looks up `msg.String()` in `bindings[mode]`." But `KeyPressMsg.String()` returns the single-key textual form (e.g. `"j"`, `"space"`). After a `Space` press, the dispatcher has NO state to remember "we're now in a Space-leader pending state." There's no `Pending() bool` accessor, no second-key dispatch path. The plan's `TestDispatcher_Dispatch` tests single-key only. So the dispatcher as specified is functionally broken for the 2 of 4 baseline commands that use leader-`Space`.

**Disposition:** Same as 1.4; whichever option the planner picks, document the multi-key case explicitly. If deferred, write that `LoadBindings` registers Space-leader commands in `Bindings.Commands` but the `NewDispatcher(b)` SKIPS registering them in `bindings[ModeNav]` (with a code comment + a refinement ID). The 4-of-4 returned count is preserved; the routable count is 2 (handoff, comment via command-mode).

---

### 1.10 NIT — `progress.go` doesn't implement `tea.Model` per plan; AC silent on this

**Trace:** D2 progress.go (lines 256-258): "Note: `progress.go` does NOT need to implement `tea.Model` unless spinner integration requires it. A simple struct with a `View()` method is sufficient." But the AC for D2 (line 263) says "`progress.go` compiles; `Progress` has at least `View() string`" — fine, but the L1 PLAN.md acceptance row (line 449-455) bundles progress with the other components under "All component implementations are pure Bubble Tea v2 models (Init/Update/View)" (line 470). L2 contradicts L1.

**Disposition:** Easy. Update L1 line 470 to acknowledge "all interactive components are pure Bubble Tea v2 sub-components (Update/View); `progress.go` is a passive render-only struct." OR add `Init`/`Update` to progress so it lines up. Recommend the former (passive `Progress` matches the "single-step status line" framing).

---

### 1.11 NIT — Migration-marker rule "every file" applies to `_test.go` too but the file-doc-comment placement is awkward

**Trace:** AC2 and L1 line 473 both say "Every file ... at the package doc-comment level (the comment immediately preceding the `package` declaration or as a standalone doc-comment on the package)." For a test file (`confirm_test.go`), the standard Go pattern is no package doc-comment (only the production file owns it). Putting `// MIGRATION TARGET: ...` immediately before `package components` in `confirm_test.go` creates two consecutive package-level doc comments (one in `confirm.go`, one in `confirm_test.go`) which `gofmt -d` may not flag but golint / `go doc` will treat oddly (only one wins as canonical).

L2 PLAN.md acknowledges this awkwardness at D3 (lines 295-299): "Package-doc migration marker NOT required again ... however per the hard rule 'every file carries' ... apply the migration marker as a file-level comment immediately before the package declaration in EVERY file." This is a workaround, not a clean spec.

**Disposition:** Either accept the awkwardness (it works, just looks weird) and add a single-sentence note in the AC: "On test files, the marker is a file-level comment preceding `package components` — it is NOT promoted to package doc." OR loosen the rule: "Every PRODUCTION file. Test files inherit via the package." The first option keeps the audit trail visible; the second is cleaner Go. Planner picks.

---

## 2. Counterexamples

The three CONFIRMED FFs above each constitute a counterexample to "the L2 plan can be built as written":

- **2.1** (FF 1.1): a builder implementing `View() string` per spec produces types that fail the AC "implements `tea.Model`."
- **2.2** (FF 1.2): a builder calling `tea.Quit` per spec produces components that exit the whole TUI when used by W2's `runInitTUI`.
- **2.3** (FF 1.3): three builders reading the three different shape hints in the loader spec produce three incompatible loaders, all of which "pass" the under-specified test.

NITs 1.4-1.11 are not show-stoppers but each leaves a hole that build-QA will likely catch (and re-spawn the builder for). Prefer absorbing them upfront.

---

## 3. PLAN-QA-DISCIPLINE checks

### 3.1 R1 (every NEW-behavior acceptance bullet → test-runner blocked_by)

- AC3 (confirm `tea.Model` + accessors) → exercised by D2's `confirm_test.go` (same droplet). ✓
- AC4 (picker_multi `tea.Model` + Selected/Cancelled) → D4's `picker_multi_test.go` (same droplet). ✓
- AC5 (dispatcher.go Dispatch) → D6's `dispatcher_test.go` (same droplet). ✓ — but the test is under-specified per 1.4/1.5/1.9.
- AC6 (loader.go LoadBindings 4↔9 commands) → D6's `dispatcher_test.go` (same droplet). ✓ — but the input shape is under-specified per 1.3.
- AC7 (per-package `mage test-pkg`) → ridden by each droplet's Mage Verification. ✓
- AC8 (`mage ci` green) → D5 closes the components package; D6 closes keybindings; both are final-stage droplets. ✓ (assuming D6 also runs `mage ci` — currently D6 mage verification line 640 only says `mage test-pkg ./internal/tui/keybindings`, not `mage ci`. NIT: add `mage ci` to D6's verification per the "either D5 or D6 is last" framing.)

**R1 verdict:** PASS modulo the under-spec FFs being addressed.

### 3.2 R2 (narrative count vs L2 spawn directive D-list)

- L1 PLAN.md spawn directive (line 480) enumerates D1-D6: "D1 style system ... D2 confirm.go + progress.go ... D3 textinput.go ... D4 picker_single.go + picker_multi.go ... D5 header.go + footer.go ... D6 vim keybinding dispatcher." Six droplets.
- L2 PLAN.md droplets: D1, D2, D3, D4, D5, D6. Six droplets.
- L2 PLAN.md narrative claims (line 1, line 7 directory, line 622 KindPayload, line 671-684 checklist): "6 droplets" consistent throughout.
- `_BLOCKERS.toml` mentions D3/D4/D5; D1 and D6 explicitly excluded with reason. Consistent.

**R2 verdict:** PASS.

---

## 4. Attack-vector sweep results

Mapping the spawn-prompt attack vectors to disposition:

1. **Missing `blocked_by` between droplets sharing packages.** Sweep result: D3/D4/D5 all share `internal/tui/components`; chained correctly (D2→D3→D4→D5). D1 (`internal/tui/style`) and D6 (`internal/tui/keybindings`) are disjoint packages, correctly unblocked. **PASS.**
2. **Cycles.** Topo-sort `{D1, D2, D6} → {D3} → {D4} → {D5}` is acyclic. **PASS.**
3. **Hidden file-share lock.** All 18 files are NEW; no overlap with existing files (`internal/tui/` has `model.go`, `diff_mode.go`, etc., but none of the new paths collide). **PASS.**
4. **PLAN.md ↔ `_BLOCKERS.toml` drift.** Both say D3 blocked by D2, D4 blocked by D3, D5 blocked by D4. D1/D6 not in `_BLOCKERS.toml` (correct — no blockers). **PASS.**
5. **YAGNI: 6 droplets justified? D2 split (confirm + progress)?** D2 combines `confirm.go` + `progress.go` — both small, both in same NEW package, both Wave-A-head with no inter-dependency. Combining justifies one droplet (saves a serialization step). progress.go is ~10 LOC; not worth its own droplet. **PASS** (justified).
6. **PLAN-QA-DISCIPLINE-R1.** See §3.1 above. **PASS modulo FF absorption.**
7. **PLAN-QA-DISCIPLINE-R2 numeric.** See §3.2 above. **PASS.**
8. **Migration-marker enforceability.** See NIT 1.7 above. **WEAK** (no automated gate); recommend Option (a) magefile target.
9. **Coverage gate trap on D1.** See NIT 1.6 above. **WEAK** (mitigation under-specified); recommend pinning the test file path.
10. **D6 binding merge semantic (local wins on collision).** Plan spec at line 591 says "ID-deep-merge: for each local command, if its ID matches a baseline command, local replaces baseline entry." Test `TestLoadBindings_LocalWins` (line 619) covers it. Local wins on the description field. **PASS.** (Caveat: the test fixture's shape is under-specified per 1.3/1.5.)
11. **Embedded baseline bytes — stale-against-stil-upstream risk.** Plan covers this at "Wave-Boundary Concerns" line 727: "Refinement KEYBIND-R3 (tracked in SKETCH §8) captures the move to stil-solid's package artifact when stable. Until then, this is an accepted known risk." Documented and accepted. **PASS.** (No counterexample to ship — it's an accepted refinement.)
12. **Mode state machine.** modes.go declares 6 Mode constants. No transition table in the plan — the dispatcher's mode parameter is treated as opaque input. The plan never specifies "illegal transition" (e.g., `ModeHint → ModeVisualBlock` direct). Dispatching is per-mode; the state-machine of WHICH mode is active is held by the **caller** (the outer TUI), not by `Dispatcher`. The current plan is consistent with this — `Dispatcher` is stateless mode-routing only. The state machine itself is NOT shipped in W5; it ships when the outer TUI consumes the dispatcher (W2 / future). **PASS** (no scope creep into state-machine validation; that's a future surface). Minor NIT: the plan should say so explicitly — recommend a one-line note in D6 modes.go: "Mode constants only; mode-transition validation is the caller's responsibility, not this package's."
13. **3-package parallel dispatch (D1 || D2 || D6) — Go toolchain global locks.** D1, D2, D6 all touch DIFFERENT new packages. Go module cache + build cache are content-addressed (no race on read); concurrent compilation of disjoint packages is well-supported by `go build` and `mage`. `go.mod` is unchanged (all needed deps already declared). No `go get` involved. **PASS.**
14. **bubbles v2 dependency.** Verified: `charm.land/bubbles/v2 v2.0.0-rc.1` in go.mod (line 20), `textinput` subpackage exists at `/Users/evanschultz/go/pkg/mod/charm.land/bubbles/v2@v2.0.0-rc.1/textinput/` with `textinput.go` + `styles.go` + `textinput_test.go`. `textinput.Model` exposes `Init`, `Update(tea.Msg) (Model, tea.Cmd)`, `View() string`. Import path `"charm.land/bubbles/v2/textinput"` in PLAN.md (line 104) matches. **PASS.**

---

## 5. Summary

**FAIL — 3 CONFIRMED FFs + 8 NITs.**

The W5 L2 plan has good shape (correct decomposition, correct `blocked_by` graph, correct R1/R2 discipline), but three concrete contract issues will cause builders to either fail acceptance (FF 1.1), break W2's consumer (FF 1.2), or produce mutually-incompatible loaders (FF 1.3). The 8 NITs cover under-specs that build-QA will likely re-route via fix-and-respawn cycles.

**Recommended absorption pass:**
- Fix FF 1.1: drop the "implements `tea.Model`" claim from AC3/AC4/D3/D4 and from L1 line 470; reframe components as Bubble Tea sub-components (`Update(tea.Msg) (X, tea.Cmd)` + `View() string` + accessors).
- Fix FF 1.2: replace every `return tea.Quit` in component `Update` specs with `return nil`; add a one-line risk note about sub-component pattern.
- Fix FF 1.3: pin baselineJSON shape and local-file shape to the SAME schema (recommend full stil baseline shape `{"product_extensions":{"tillsyn":{"commands":[...]}}}` — Option A); update the embedded baseline literal accordingly.
- Absorb NITs 1.4-1.11 with explicit per-NIT dispositions (preferred) or DEFER-with-reason per the `feedback_nits_are_first_class.md` discipline.

After absorption, re-run plan-QA proof + falsification. The plan is otherwise architecturally sound — these are spec-precision issues, not structural ones.

---

## TL;DR

- **T1**: 3 CONFIRMED counterexamples — (a) `View() string` doesn't satisfy `tea.Model` (which is `View() View` in bubbletea v2 rc.2; verified in tree at `tea.go:52-63`); (b) `tea.Quit` inside reusable components kills the parent TUI; (c) loader JSON shape is ambiguous across baseline-embed vs local-file paths.
- **T2**: 3 confirmed counterexamples reproduce: (2.1) `var _ tea.Model = (*ConfirmModel)(nil)` fails to compile under literal spec; (2.2) W2's `runInitTUI` exits when the `.mcp.json` confirm fires `y`; (2.3) three independent builders write three incompatible loaders, each "passes" its own under-specified test.
- **T3**: PLAN-QA-DISCIPLINE R1 (every NEW-behavior bullet → test-runner blocked_by) PASS modulo absorption; R2 (D-list narrative count) PASS — 6 droplets enumerated consistently.
- **T4**: Attack-vector sweep clean on structural axes (blocked_by graph, cycles, file-share, multi-package parallel, bubbles dep) — failures concentrate on contract-precision (V1.1, V1.2, V1.3) and under-specification (NITs 1.4, 1.5, 1.6, 1.7, 1.9, 1.10, 1.11) plus codebase-consistency (NIT 1.8).
- **T5**: Verdict FAIL with absorption pass needed; plan is architecturally sound — fixes are spec-precision edits, not structural rework.

---

## Hylla Feedback

N/A — Hylla is OFF per spawn prompt. All evidence collected via `Read`, `Grep` (system grep), filesystem inspection of `/Users/evanschultz/go/pkg/mod/charm.land/bubbletea/v2@v2.0.0-rc.2/`, and Context7 (which returned bubbletea v1 docs — verified against in-tree v2 source instead).
