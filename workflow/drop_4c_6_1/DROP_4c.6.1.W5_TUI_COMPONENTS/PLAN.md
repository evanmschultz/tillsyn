# DROP 4c.6.1.W5 — TUI Components + Style System + Vim Keybinding Dispatcher

**State:** planning
**Kind:** plan (L2 sub-plan — atomic droplets at leaves)
**Wave:** Wave A (no blockers; dispatches in parallel with W0, W4.D1, W6, W7.D1, W8)
**Blocks:** 4c.6.1.W2 (TUI uses `confirm.go` + `picker_multi.go` from this wave)
**Directory:** `workflow/drop_4c_6_1/DROP_4c.6.1.W5_TUI_COMPONENTS/`

---

## Round 2 Changes (L2 round-2 planner absorption — 2026-05-12)

Absorbed round-1 plan-QA proof (PASS-WITH-FINDINGS: 2 FFs + 8 NITs) and falsification (FAIL: 3 FFs + 8 NITs) plus R10 inline absorptions from `workflow/drop_4c_6_1/PLAN.md` locked decisions (lines 926).

### Falsification FFs (were FAIL — now absorbed)

- **Fals FF1.1 CRITICAL** (`tea.Model` contract mismatch — `View()` returns `tea.View`, not `string`): ABSORBED.
  W5 components are **Bubble Tea sub-components** composed by the outer `tea.Model` at `internal/tui/model.go`. They do NOT implement `tea.Model`. All "implements `tea.Model`" claims removed from AC3, AC4, D3 AC, D4 AC, and the L1 scope note. Reframed: every component has `Update(tea.Msg) (X, tea.Cmd)` + `View() string` + accessor methods. No `var _ tea.Model = (*ConfirmModel)(nil)` static check anywhere.
- **Fals FF1.2 HIGH** (`tea.Quit` inside reusable sub-components kills parent TUI): ABSORBED.
  Every `return tea.Quit` in D2/D3/D4 Update specs replaced with `return nil`. Components signal completion via `Done()`, `Confirmed()`, `Cancelled()`, `Submitted()`, `Selected()` accessors. The parent TUI (outer `tea.Model` at `internal/tui/model.go` or W2's `runInitTUI`) polls these accessors on each Update cycle to advance its own state machine.
- **Fals FF1.3 HIGH** (loader JSON shape ambiguous — baseline embed ≠ local file nested schema): ABSORBED.
  Both baseline embed and local file now use ONE pinned shape: the nested stil schema `{"product_extensions":{"tillsyn":{"commands":[...]}}}`. The embedded baseline literal in `loader.go` is updated to the nested form. `LoadBindings` extracts `product_extensions.tillsyn.commands` from BOTH inputs via the same code path. See updated D6 Specify.

### Proof FFs (were substantive findings — now absorbed)

- **Proof FF1.1 MEDIUM** (D6 baseline-JSON shape ambiguity — same root as fals FF1.3): ABSORBED inline above.
- **Proof FF1.2 LOW** (KEYBIND-R3 staleness TODO): ABSORBED. D6 `loader.go` Specify now requires a `// TODO(KEYBIND-R3): refresh embedded stil baseline bytes when stil-solid v<X> publishes` comment in `loader.go` for grep-discoverability.

### NITs absorbed (all 16 — 8 proof + 8 fals)

- **Proof NIT2.1** (D1 coverage-gate conditional→imperative): ABSORBED. `palette_test.go` added to D1 Paths as a mandatory NEW file. Acceptance bullet is now imperative.
- **Proof NIT2.2** (D5 header/footer coverage fallback): ABSORBED. Explicit authority added: builder MAY create `header_test.go` + `footer_test.go` if coverage drops below 70%; these are NOT in L1 paths but within D5's authority.
- **Proof NIT2.3** (D2 `progress.go` optional style-import dep): ABSORBED. `progress.go` MUST use inline lipgloss style; importing `internal/tui/style` in D2 would add an undeclared `blocked_by D1`. Constraint added.
- **Proof NIT2.4** (D3 `textinput.go` marker self-contradiction): ABSORBED. "NOT required again" clause deleted. Only the hard per-file rule remains.
- **Proof NIT2.5** (D6 test marker implicit): confirmed pass; no change needed.
- **Proof NIT2.6** (R3/R4 advisory risk notes): no action needed; flagged for awareness only.
- **Proof NIT2.7** (D6 `modes.go` marker placement): ABSORBED. Note added: migration marker and package doc-comment are TWO separate `//` lines immediately above `package keybindings`.
- **Proof NIT2.8** (AC8 `mage ci` gate placement): ABSORBED. Clarified: `mage ci` runs after BOTH D5 AND D6 are `done`. It is a drop-end gate (AC8), not a per-D5 gate. D6's Mage verification updated to add `mage ci`.
- **Fals NIT1.4** (multi-key sequences `["Space","n"]` under-specified): DEFERRED-AS-NIT. Resolution: `NewDispatcher` skips registering commands with multi-key `Keys` arrays in `bindings[ModeNav]` (a single `Dispatch(tea.KeyMsg, Mode)` call cannot express a two-key sequence). Multi-key commands (`new-drop`, `complete-drop`) land only in `Bindings.Commands` (accessible via `DispatchCommand`). A code comment `// TODO(KEYBIND-R4): implement leader-key state machine for multi-key nav bindings` goes in `dispatcher.go`. `TestDispatcher_Dispatch` adds a case asserting multi-key commands return `NoOp` from `Dispatch` (correct behavior, not a bug). Tracked as KEYBIND-R4.
- **Fals NIT1.5** (local 5-command file shape unspecified): ABSORBED. D6 Specify now pins the canonical local file shape per REVISION_BRIEF §2.19: `{"schema_version":1, ..., "product_extensions":{"tillsyn":{"commands":[...5 entries...]}}}`. Test fixture `TestLoadBindings_WithLocal` uses this exact shape.
- **Fals NIT1.6** (coverage gate trap — pure-var package): ABSORBED inline via NIT2.1 (same finding). `palette_test.go` is now mandated in D1 Paths.
- **Fals NIT1.7** (migration-marker enforceability no automated gate): DEFERRED-AS-NIT with disposition. Option (b) chosen: each build-QA-proof agent explicitly checks all files in the droplet's `Paths` list for the `// MIGRATION TARGET: github.com/hylla-org/lykta` marker as a named acceptance bullet. Option (a) (new `mage check-migration-markers` target) deferred as MIGRATE-MARKER-R1 — durable automated gate is better but requires a new droplet and is not in W5's scope.
- **Fals NIT1.8** (`tea.KeyMsg` vs `tea.KeyPressMsg` codebase inconsistency): ABSORBED. All `case tea.KeyMsg:` in D2/D3/D4 Update type-switch specs changed to `case tea.KeyPressMsg:` (consistent with `internal/tui/model.go:2004`; avoids double-fire on key-release). `Dispatch(msg tea.KeyMsg, mode Mode)` parameter remains `tea.KeyMsg` (interface type — correct for the public API to accept either press or release from the caller).
- **Fals NIT1.9** (Dispatch multi-key can't be expressed): DEFERRED-AS-NIT. Same disposition as NIT1.4 + NIT1.6 (KEYBIND-R4). `Dispatch` resolves on single `tea.KeyMsg.String()` only.
- **Fals NIT1.10** (`progress.go` L1/L2 contradiction): ABSORBED. L1 line 470 note updated (in L2 Scope note); D2 progress.go is explicitly a passive render-only struct — NOT a sub-component with `Update()`/`Init()`. L2 AC reflects this.
- **Fals NIT1.11** (test-file migration marker placement awkward): ABSORBED. Explicit note added in Context Blocks: on `_test.go` files, the marker is a file-level `//` comment immediately before `package <name>` — it is NOT promoted to package doc. This works but looks different from production files; accepted as the consistent audit-trail form.

---

## Objective

Build the inline TUI component library at `internal/tui/components/`, the style system at
`internal/tui/style/`, and the vim keybinding dispatcher at `internal/tui/keybindings/`. All
three are NEW Go packages with zero pre-existing files. Every file carries the
`// MIGRATION TARGET: github.com/hylla-org/lykta` doc-comment at package-doc level
(EXTRACT-R1 + KEYBIND-R1). Components are **Bubble Tea sub-components** (NOT standalone
`tea.Model` implementations) — they expose `Update(tea.Msg) (X, tea.Cmd)`, `View() string`,
and typed accessor methods; they compose into the outer `tea.Model` at
`internal/tui/model.go`. The keybinding dispatcher loads the stil baseline 4 Tillsyn
commands (embedded as nested-schema JSON bytes) and optionally ID-merges the project-local
`.tillsyn/bindings.json` (5 additional commands) for a 9-command palette; absent local file =
graceful baseline-only fallback, NOT fail-loud. All component Update methods return `nil`
(never `tea.Quit`); done/cancel state is signaled via accessor methods.

## Acceptance Criteria

- **AC1** — All files listed in Paths sections below exist and compile: 10 component source
  files (`confirm.go`, `confirm_test.go`, `textinput.go`, `textinput_test.go`,
  `picker_single.go`, `picker_single_test.go`, `picker_multi.go`, `picker_multi_test.go`,
  `header.go`, `footer.go`, `progress.go`), 4 style files (`palette.go`, `palette_test.go`,
  `spacing.go`, `typography.go`), 4 keybinding files (`dispatcher.go`, `loader.go`,
  `modes.go`, `dispatcher_test.go`).
- **AC2** — Every file (production AND test) carries `// MIGRATION TARGET:
  github.com/hylla-org/lykta` as a file-level comment immediately before the `package`
  declaration. On production files this becomes part of the package doc-comment; on test
  files it is a standalone `//` line (not promoted to package doc). No file in the three
  new packages is missing this marker.
- **AC3** — `confirm.go` defines `ConfirmModel` as a Bubble Tea **sub-component** (NOT
  `tea.Model`): has `Init() tea.Cmd`, `Update(tea.Msg) (ConfirmModel, tea.Cmd)`,
  `View() string` methods; `Confirmed()`, `Cancelled()`, `Done()` bool accessors exist;
  renders a y/n prompt; handles `y`/`Y`, `n`/`N`, `Enter` (when default set), `Escape` keys;
  all returning `return nil` (never `return tea.Quit`).
- **AC4** — `picker_multi.go` defines `PickerMultiModel` as a Bubble Tea **sub-component**:
  has `Update(tea.Msg) (PickerMultiModel, tea.Cmd)`, `View() string`; returns `[]string` of
  selected items via `Selected()` accessor; `Done()`, `Cancelled()` accessors exist; handles
  `j`/`k` navigation, `Space` toggle, `Enter` confirm, `Escape` cancel; all returning `nil`.
- **AC5** — `dispatcher.go` `Dispatch(keyMsg tea.KeyMsg, mode Mode) HandlerFunc` returns the
  registered handler for the binding or `NoOp` if unregistered; multi-key commands
  (e.g. `new-drop` with `keys: ["Space","n"]`) return `NoOp` from `Dispatch` (pending
  KEYBIND-R4 leader-key state machine).
- **AC6** — `loader.go` `LoadBindings(baselineJSON []byte, localPath string)` parses both
  `baselineJSON` and the local file using the SAME nested schema:
  `{"product_extensions":{"tillsyn":{"commands":[...]}}}`. Loads baseline's 4 Tillsyn
  commands; if `localPath` is non-empty and the file exists, ID-merges local
  `product_extensions.tillsyn.commands` (local wins on collision); returns a `Bindings` value
  with 9 commands when local present, 4 when absent; absent local file returns baseline-only,
  nil error (NOT an error condition).
- **AC7** — `mage test-pkg ./internal/tui/components` passes (≥70% coverage); `mage test-pkg
  ./internal/tui/style` passes (≥70% coverage — guaranteed by `palette_test.go`); `mage
  test-pkg ./internal/tui/keybindings` passes (≥70% coverage).
- **AC8** — `mage ci` green after BOTH D5 AND D6 complete. This is a drop-end gate run by the
  orchestrator once both the final component droplet (D5) and the keybinding droplet (D6) are
  in `done` state. It is NOT a per-D5 gate.

## Validation Plan

- Per-droplet: builder runs `mage test-pkg <pkg>` for the touched package after each droplet.
- Drop-end: `mage ci` from `main/` once all 6 droplets are complete and committed.
- AC3 / AC4 / AC5 / AC6 verified by the per-droplet build-QA agents reading test output.
- AC2 verified per-droplet: build-QA-proof explicitly checks all files in the droplet's
  `Paths` list for the `// MIGRATION TARGET: github.com/hylla-org/lykta` marker.

## Risk Notes

- **R1 — D6 three production files**: `dispatcher.go` + `loader.go` + `modes.go` = 3
  production files in one droplet, which is the CLAUDE.md "smell" threshold. Mitigated: the
  L1 directive explicitly groups all four files as one droplet; each file is small (~30-60
  LOC); they form one coherent package concern (loader feeds dispatcher; modes feeds
  dispatcher); splitting them would leave partially-initialized package state across builds.
  Exception documented here per CLAUDE.md RiskNotes rule.
- **R2 — stil baseline loading mechanism**: The baseline lives at a sibling-repo filesystem
  path which is NOT portable. Builder decision: embed the 4 baseline Tillsyn commands as a
  package-level `[]byte` in the nested schema form. The `LoadBindings` signature accepts
  `baselineJSON []byte` so the caller controls the source. The actual embedded bytes use the
  same nested `product_extensions.tillsyn.commands` schema as the local file — one decoder
  for both.
- **R3 — header/footer/progress test coverage**: `header.go`, `footer.go`, `progress.go` have
  no co-located test files in the L1 paths list. The `internal/tui/components` package still
  needs ≥70% coverage; confirm+textinput+pickers must carry the package. Builder should ensure
  the package coverage stays green; if it fails, add minimal smoke tests for
  header/footer/progress render output (within D5's authority — see D5 note).
- **R4 — W2 critical path through W5**: W2 cannot dispatch until W5 closes, but W5's critical
  path for W2 is D2 → D3 → D4 (three serialized droplets). The parallel droplets D1 and D6
  do not affect W2's readiness. Orch should be aware that W2 dispatch does not unblock the
  moment W5 starts — it unblocks when D4 reaches `done`.
- **R5 — Sub-component pattern (absorbed from fals FF1.2)**: Components are SUB-MODELS — they
  must NOT return `tea.Quit`. The parent TUI polls `Done()` / `Confirmed()` / `Cancelled()` /
  `Submitted()` accessors after each `Update` call and advances its own state machine.
  Builders must not conflate "done with this step" with "exit the program."
- **R6 — Multi-key bindings (KEYBIND-R4)**: Two of the four stil baseline Tillsyn commands
  use multi-key `keys` arrays (`new-drop: ["Space","n"]`, `complete-drop: ["Space","c"]`).
  The current `Dispatch(tea.KeyMsg, Mode)` API resolves on a single key. `NewDispatcher`
  intentionally skips registering multi-key commands in `bindings[ModeNav]` (they remain in
  `Bindings.Commands` for command-mode use). KEYBIND-R4 tracks the leader-key state machine
  to enable nav-mode multi-key dispatch.

## Context Blocks

- **constraint (critical)** — Every file in all three new packages (production AND test) MUST
  carry `// MIGRATION TARGET: github.com/hylla-org/lykta` as a file-level comment immediately
  before the `package` declaration. On `_test.go` files this is a standalone `//` comment,
  not part of the package doc (two separate `//` comment lines in the production package file
  are fine; the test file just prepends the marker before `package <name>`). Build-QA-proof
  agents: check this explicitly for every file in your droplet's Paths list.
- **constraint (critical)** — Never use raw `go test` / `go build` / `go vet`. Always
  `mage <target>`. Never `mage install`.
- **constraint (critical)** — Components are Bubble Tea **sub-components**, NOT standalone
  `tea.Model` implementations. `View() string` (not `View() tea.View`). No
  `var _ tea.Model = (*ConfirmModel)(nil)` assertions. No `return tea.Quit` anywhere in any
  component's `Update` method. Signal completion via accessor methods (`Done()`, etc.); let
  the parent TUI advance its own state machine.
- **constraint (high)** — `loader.go` MUST NOT depend on the sibling-repo filesystem path for
  the stil baseline. Baseline Tillsyn commands are embedded as package-level `[]byte` in the
  nested schema form `{"product_extensions":{"tillsyn":{"commands":[...]}}}`. Only the
  Tillsyn-local `.tillsyn/bindings.json` is read from the live filesystem.
- **constraint (high)** — `progress.go` is a passive render-only struct. It does NOT have
  `Init()` or `Update()` methods. Only `View() string` and `WithMessage(string) Progress`.
  Builder must not add state-machine behavior to `progress.go`.
- **constraint (high)** — All `Update` type-switches in D2/D3/D4 components MUST use
  `case tea.KeyPressMsg:` (NOT `case tea.KeyMsg:`). `tea.KeyMsg` is an interface matching
  both press AND release events; using it would double-fire handlers. The public
  `Dispatch(msg tea.KeyMsg, mode Mode)` API parameter stays as the interface type (callers
  may pass either — correct), but the internal type-switch in Update narrows to
  `tea.KeyPressMsg`.
- **decision** — D4 combines `picker_single.go` + `picker_multi.go` in one droplet. Both
  use the list-selection pattern; combined shipping keeps W2's `picker_multi.go` available
  sooner and reduces package-lock serialization overhead.
- **decision** — `header.go` + `footer.go` are combined in D5 (simple styled chrome, no
  state machine, fits 1-2 code blocks each).
- **decision** — KEYBIND-R4 DEFERRED: multi-key nav-mode bindings (`Space n`, `Space c`)
  require a leader-key pending state machine not in scope for D6. `NewDispatcher` skips
  registering multi-key commands in `bindings[ModeNav]` and logs a code-comment with
  `// TODO(KEYBIND-R4)`. These commands are accessible only via `DispatchCommand(id)`.
- **decision** — Migration-marker automated gate (Option b chosen for now): each build-QA
  proof pass checks the marker per-droplet. Option (a) (`mage check-migration-markers` target)
  deferred as MIGRATE-MARKER-R1.
- **reference** — stil baseline.json path for local reading during development:
  `/Users/evanschultz/Documents/Code/hylla/stil/main/src/bindings/baseline.json`.
  `product_extensions.tillsyn.commands` block contains exactly 4 entries:
  `{"id":"new-drop","keys":["Space","n"],"description":"New drop in current project."}`,
  `{"id":"complete-drop","keys":["Space","c"],"description":"Mark drop complete."}`,
  `{"id":"handoff","command":"handoff","description":"Open handoff dialog for current drop."}`,
  `{"id":"comment","command":"comment","description":"Add a comment thread to current drop."}`.
  This is the verified canonical shape. The embedded `stilBaselineTillsynJSON` bytes in
  `loader.go` wrap these in `{"product_extensions":{"tillsyn":{"commands":[...]}}}`.
- **reference** — Import paths (verified from go.mod + internal/tui/model.go):
  `tea "charm.land/bubbletea/v2"`, `"charm.land/bubbles/v2/textinput"`,
  `"charm.land/bubbles/v2/key"`, `"charm.land/lipgloss/v2"`.
  teatest: `"github.com/charmbracelet/x/exp/teatest/v2"` (replaced by `./third_party/teatest_v2`).
  `tea.KeyPressMsg` (NOT `tea.KeyMsg`) is the concrete type for key-press events in
  `charm.land/bubbletea/v2 v2.0.0-rc.2` (verified: `internal/tui/model.go:2004` uses
  `case tea.KeyPressMsg:`).
- **reference** — `tea.Model` in `charm.land/bubbletea/v2@v2.0.0-rc.2/tea.go:52-63` has
  `View() View` (struct return), NOT `View() string`. W5 components intentionally do NOT
  implement `tea.Model`; they are sub-components returning `View() string` composed by the
  outer `tea.Model` at `internal/tui/model.go`.
- **reference** — Tillsyn-local bindings.json canonical shape (REVISION_BRIEF §2.19):
  `{"schema_version":1,"name":"tillsyn-bindings","description":"...","extends":"stil-baseline","product_extensions":{"tillsyn":{"commands":[...5 entries...]}}}`.
  The 5 local-only commands: `dispatch`, `plan`, `archive`, `settings`, `help` — all have
  `command` field only (no `keys` — command-mode only). No ID collision with the 4 baseline
  commands. Union = 9 commands.
- **warning (high)** — `internal/tui/components` is a NEW package. D2 creates it; D3, D4,
  D5 extend it. If D3/D4/D5 attempt to build before D2 is `done`, Go will fail to compile
  (`package internal/tui/components: no Go files`). The `blocked_by` graph enforces ordering;
  do not dispatch D3 until D2 is `done`.
- **warning (high)** — `progress.go` is in D2 (creates the package) but is a passive struct.
  D2's `mage test-pkg ./internal/tui/components` must pass coverage ≥70% from
  `confirm_test.go` alone (progress.go is untested until D5's package-wide check). If the
  package shows < 70% after D2, builder must add minimal tests for `Progress.View()`.
- **note** — `progress.go` does NOT implement `tea.Model` (no `Init()`, no `Update()`). A
  simple struct with `View() string` and `WithMessage(string) Progress` is the correct design.
  Sub-component pattern does not require `tea.Model`; passive render helpers are valid
  participants in a Bubble Tea view hierarchy.

---

## Droplet Decomposition

### Parallel dispatch graph

```
D1 (internal/tui/style)               ─── no blockers
D2 (components: confirm+progress)     ─── no blockers (creates internal/tui/components)
D6 (internal/tui/keybindings)         ─── no blockers

D3 → D2   (adds textinput to components)
D4 → D3   (adds pickers to components)
D5 → D4   (adds header+footer to components)
```

Topo-sort: `{D1, D2, D6}` → `{D3}` → `{D4}` → `{D5}`.
D1 and D6 can dispatch and complete entirely in parallel with D2-D5.
W2 critical path through W5: D2 → D3 → D4 must complete before W2 can fully use W5.

---

### D1 — Style System

- **Kind:** `build`
- **Irreducible:** `true`
- **State:** `todo`
- **Role:** builder

**Paths:**
- `internal/tui/style/palette.go` (NEW)
- `internal/tui/style/palette_test.go` (NEW — MANDATORY; see coverage note)
- `internal/tui/style/spacing.go` (NEW)
- `internal/tui/style/typography.go` (NEW)

**Packages:** `internal/tui/style` (NEW package — D1 creates it)

**Blocked by:** — (no blockers; Wave A head; separate package from components and keybindings)

**Specify:**

Objective: Create the `internal/tui/style` package with semantic color tokens, spacing
constants, and typography helpers. No dependencies on `internal/tui/components` or
`internal/tui/keybindings`. Three production files plus one mandatory test file.

KindPayload changes:
- `palette.go`: file-level `// MIGRATION TARGET: github.com/hylla-org/lykta` before
  `package style`. `lipgloss.AdaptiveColor` or `lipgloss.Color` constants/vars for semantic
  names: `Primary`, `Accent`, `Success`, `Warning`, `Error`, `Muted`, `Background`, `Surface`,
  `OnSurface`. At minimum exports: `Primary`, `Accent`, `Success`, `Warning`, `Error`, `Muted`.
  Add one exported accessor function: `AllColors() []lipgloss.Color` returning all semantic
  colors in a slice. This function is the test anchor for the coverage gate.
- `palette_test.go`: file-level `// MIGRATION TARGET: github.com/hylla-org/lykta` before
  `package style`. `TestAllColors_NonEmpty`: calls `AllColors()` and asserts len > 0 and
  each element is non-zero. This test MUST be present — pure var/const packages with no test
  functions cause the magefile coverage runner to error with "no coverage rows parsed."
- `spacing.go`: file-level migration marker. `lipgloss.Border`-compatible padding/margin
  integer constants: `SpaceXS = 0`, `SpaceSM = 1`, `SpaceMD = 2`, `SpaceLG = 4`,
  `SpaceXL = 6`. All exported.
- `typography.go`: file-level migration marker. Exported `lipgloss.Style` vars for text
  roles: `Heading`, `Body`, `Label`, `Code`, `Muted`. Constructed from `palette.go` colors.
  Use `lipgloss.NewStyle()` — NOT package-level `var` with init-time dependency loops.

AcceptanceCriteria:
- All 4 files exist and every file carries `// MIGRATION TARGET: github.com/hylla-org/lykta`
  as a file-level comment immediately before the `package style` declaration (build-QA-proof
  checks each file in Paths explicitly).
- `palette.go` exports at minimum: `Primary`, `Accent`, `Success`, `Warning`, `Error`, `Muted`,
  `AllColors() []lipgloss.Color`.
- `spacing.go` exports at minimum: `SpaceSM`, `SpaceMD`, `SpaceLG`.
- `typography.go` exports at minimum: `Heading`, `Body`, `Label`, `Code`.
- `mage test-pkg ./internal/tui/style` passes with ≥70% coverage. `palette_test.go`
  MUST exist with `TestAllColors_NonEmpty` calling `AllColors()` — this is NOT conditional;
  a pure-var/const package without test functions causes the magefile coverage runner to error.

Mage verification: `mage test-pkg ./internal/tui/style`

---

### D2 — Components: Confirm + Progress

- **Kind:** `build`
- **Irreducible:** `true`
- **State:** `todo`
- **Role:** builder

**Paths:**
- `internal/tui/components/confirm.go` (NEW)
- `internal/tui/components/confirm_test.go` (NEW)
- `internal/tui/components/progress.go` (NEW)

**Packages:** `internal/tui/components` (NEW package — D2 creates it)

**Blocked by:** — (no blockers; creates the package)

**Specify:**

Objective: Create the `internal/tui/components` package with the `ConfirmModel` and
`Progress` types. These are the highest-priority components for W2. `confirm.go` must be
complete because W2's `runInitTUI` needs it for the `.mcp.json` write confirmation prompt.
`ConfirmModel` is a Bubble Tea sub-component — NOT a `tea.Model` — with `View() string`
and accessor methods. `progress.go` is a passive render-only struct.

KindPayload changes:
- `confirm.go`:
  - File-level `// MIGRATION TARGET: github.com/hylla-org/lykta` before `package components`.
  - `// ConfirmModel is a y/n prompt Bubble Tea sub-component.`
    ```go
    type ConfirmModel struct {
        prompt     string
        defaultYes bool
        confirmed  bool
        cancelled  bool
        done       bool
    }
    ```
  - Constructor: `func NewConfirm(prompt string, defaultYes bool) ConfirmModel`
  - `Init() tea.Cmd` — returns `nil`.
  - `Update(tea.Msg) (ConfirmModel, tea.Cmd)` — type-switch on `tea.KeyPressMsg` (NOT
    `tea.KeyMsg`):
    - `y`, `Y` → `confirmed=true, done=true`, return `m, nil`.
    - `n`, `N` → `cancelled=true, done=true`, return `m, nil`.
    - `Enter` → if `defaultYes`: `confirmed=true`; else: `cancelled=true`; `done=true`,
      return `m, nil`.
    - `Escape` → `cancelled=true, done=true`, return `m, nil`.
    - All other messages: return `m, nil`.
    - **NEVER `return m, tea.Quit`** — this kills the parent TUI.
  - `View() string` — renders the prompt with `[y/N]` or `[Y/n]` indicator per `defaultYes`.
  - `Confirmed() bool`, `Cancelled() bool`, `Done() bool` accessors.
  - Note: return type is `(ConfirmModel, tea.Cmd)` (concrete type), NOT `(tea.Model, tea.Cmd)`.
    Parent calls: `m.confirm, cmd = m.confirm.Update(msg)`.
- `confirm_test.go`:
  - File-level `// MIGRATION TARGET: github.com/hylla-org/lykta` before `package components`.
  - Table-driven `TestConfirmModel_Update` covering: `y`, `Y` → `Confirmed()=true`;
    `n`, `N` → `Cancelled()=true`; `Enter` with `defaultYes=true` → `Confirmed()=true`;
    `Enter` with `defaultYes=false` → `Cancelled()=true`; `Escape` → `Cancelled()=true`.
    Each row: send `tea.KeyPressMsg{...}` directly to `model.Update(msg)`, assert accessor
    state. `Done()=true` in all cases.
  - Assert NO case returns `tea.Quit` cmd — the returned `tea.Cmd` must be `nil` in all rows.
  - Use direct `model.Update(tea.KeyPressMsg{...})` for state assertions (simpler and
    sufficient; no teatest needed here).
- `progress.go`:
  - File-level `// MIGRATION TARGET: github.com/hylla-org/lykta` before `package components`.
  - `// Progress renders a single-line status message. It is a passive render-only struct,`
  - `// not a Bubble Tea sub-component (no Init/Update).`
    ```go
    type Progress struct {
        message string
    }
    ```
  - Constructor: `func NewProgress(message string) Progress`
  - `View() string` — renders the message with an inline lipgloss style (MUST NOT import
    `internal/tui/style` — D2 has no `blocked_by D1`; use `lipgloss.NewStyle().Foreground(...)` 
    inline). Label-equivalent styling inline is fine.
  - `WithMessage(message string) Progress` — returns a copy with the updated message.
  - **No `Init()` or `Update()` methods.** This is intentional — `progress.go` is display-only.

AcceptanceCriteria:
- `confirm.go` compiles; `ConfirmModel` has `Init`, `Update`, `View` methods (sub-component
  pattern — NOT `tea.Model`); `Confirmed()`, `Cancelled()`, `Done()` accessors exist.
- `Update` return type is `(ConfirmModel, tea.Cmd)` — concrete type, NOT `tea.Model`.
- `confirm_test.go` passes: all table rows cover the key-handling matrix; no row returns a
  non-nil `tea.Cmd`.
- `progress.go` compiles; `Progress` has `View() string` and `WithMessage(string) Progress`;
  does NOT have `Init()` or `Update()` methods; does NOT import `internal/tui/style`.
- Migration marker present in all 3 files as file-level comment before `package components`
  (build-QA-proof checks each file in Paths explicitly).
- `mage test-pkg ./internal/tui/components` passes.

Mage verification: `mage test-pkg ./internal/tui/components`

---

### D3 — Components: TextInput

- **Kind:** `build`
- **Irreducible:** `true`
- **State:** `todo`
- **Role:** builder

**Paths:**
- `internal/tui/components/textinput.go` (NEW)
- `internal/tui/components/textinput_test.go` (NEW)

**Packages:** `internal/tui/components` (existing after D2; D3 adds to it)

**Blocked by:** D2 (same package compile — `internal/tui/components` must exist before D3 builds)

**Specify:**

Objective: Add `TextInputModel` to `internal/tui/components` — a Bubble Tea sub-component
wrapper over `charm.land/bubbles/v2/textinput` with Tillsyn styling and an optional
validation hook. W2 uses this for the project-name input field in `till init`.
`TextInputModel` is a sub-component (NOT `tea.Model`): `View() string`, concrete return types.

KindPayload changes:
- `textinput.go`:
  - File-level `// MIGRATION TARGET: github.com/hylla-org/lykta` before `package components`.
  - Every file in the package carries the migration marker as a file-level comment; this file
    is no exception. The per-file rule is: marker comment immediately before `package components`.
  - `// TextInputModel wraps bubbles/textinput with Tillsyn styling and validation.`
    ```go
    type TextInputModel struct {
        inner     textinput.Model
        validate  func(string) error
        err       error
        submitted bool
    }
    ```
  - Constructor:
    ```go
    func NewTextInput(placeholder string, validate func(string) error) TextInputModel
    ```
    where `validate` may be `nil` (no-op).
  - `Init() tea.Cmd` — delegates to `inner.Init()`.
  - `Update(tea.Msg) (TextInputModel, tea.Cmd)` — type-switch on `tea.KeyPressMsg`; delegates
    to `inner.Update(msg)`; on `Enter` key: calls `validate(inner.Value())` if non-nil;
    if passes: `submitted=true`, return `m, nil`; if fails: sets `m.err`, return `m, nil`.
    On all other messages: calls `validate(inner.Value())` to keep error state current.
    **NEVER `return m, tea.Quit`**.
  - `View() string` — delegates to `inner.View()`; if `m.err != nil`, appends error string
    below the input rendered line.
  - `Value() string`, `Err() error`, `Submitted() bool` accessors.
  - Note: `Update` return type is `(TextInputModel, tea.Cmd)` — concrete type, NOT `tea.Model`.

- `textinput_test.go`:
  - File-level `// MIGRATION TARGET: github.com/hylla-org/lykta` before `package components`.
  - Table-driven `TestTextInputModel_Validation` — cases: nil validator (always valid),
    passing validator, failing validator; assert `Err()` state after keystroke simulation.
  - `TestTextInputModel_Submit` — Enter with valid input (`Submitted()=true`, `Err()=nil`);
    Enter with invalid input (`Submitted()=false`, `Err()!=nil`).
  - Assert NO case returns non-nil `tea.Cmd`.

AcceptanceCriteria:
- `textinput.go` compiles alongside `confirm.go` in `internal/tui/components`.
- `TextInputModel` is a Bubble Tea sub-component: `Update(tea.Msg) (TextInputModel, tea.Cmd)`,
  `View() string`, `Init() tea.Cmd` — NOT `tea.Model`. No `var _ tea.Model = (...)` assertion.
- `Value()`, `Err()`, `Submitted()` accessors exist.
- `textinput_test.go` passes: validation + submit matrix covered; no non-nil cmd returned.
- Migration marker present in both files as file-level comment before `package components`
  (build-QA-proof checks each file in Paths explicitly).
- `mage test-pkg ./internal/tui/components` passes (full package).

Mage verification: `mage test-pkg ./internal/tui/components`

---

### D4 — Components: Pickers (Single + Multi)

- **Kind:** `build`
- **Irreducible:** `true`
- **State:** `todo`
- **Role:** builder

**Paths:**
- `internal/tui/components/picker_single.go` (NEW)
- `internal/tui/components/picker_single_test.go` (NEW)
- `internal/tui/components/picker_multi.go` (NEW)
- `internal/tui/components/picker_multi_test.go` (NEW)

**Packages:** `internal/tui/components` (existing after D3; D4 adds to it)

**Blocked by:** D3 (same package compile)

**Specify:**

Objective: Add `PickerSingleModel` and `PickerMultiModel` to `internal/tui/components`.
Both are Bubble Tea sub-components (NOT `tea.Model`). Both use a list-selection pattern.
`picker_multi.go` is the higher-priority target (W2 uses it for multi-group selection in
`till init`). Combined in one droplet because they share a base list-navigation pattern.
Two production files — within the CLAUDE.md "acceptable" bound.

KindPayload changes:
- `picker_single.go`:
  - File-level `// MIGRATION TARGET: github.com/hylla-org/lykta` before `package components`.
  - ```go
    type PickerSingleModel struct {
        items    []string
        cursor   int
        selected string
        done     bool
    }
    ```
  - Constructor: `func NewPickerSingle(items []string) PickerSingleModel`
  - `Init() tea.Cmd` returns `nil`.
  - `Update(tea.Msg) (PickerSingleModel, tea.Cmd)`: type-switch on `tea.KeyPressMsg` (NOT
    `tea.KeyMsg`): `j`/`k` navigate cursor; `Enter` confirms `selected=items[cursor]`,
    `done=true`, return `m, nil`; `Escape` → `done=true` without selecting, return `m, nil`.
    **NEVER `return m, tea.Quit`**.
  - `View() string`: renders list with `>` cursor indicator; selected item highlighted.
  - `Selected() string`, `Done() bool` accessors.
  - Return type: `(PickerSingleModel, tea.Cmd)` — concrete, NOT `tea.Model`.

- `picker_single_test.go`:
  - File-level migration marker.
  - `TestPickerSingleModel_Navigation`: j/k moves cursor, wraps at boundaries.
  - `TestPickerSingleModel_Select`: Enter confirms selection (`Selected()` = correct item,
    `Done()=true`); Escape cancels (`Selected()=""`, `Done()=true`).
  - Assert no case returns non-nil cmd.

- `picker_multi.go`:
  - File-level `// MIGRATION TARGET: github.com/hylla-org/lykta` before `package components`.
  - ```go
    type PickerMultiModel struct {
        items     []string
        cursor    int
        selected  map[int]bool
        done      bool
        cancelled bool
    }
    ```
  - Constructor: `func NewPickerMulti(items []string) PickerMultiModel`
  - `Init() tea.Cmd` returns `nil`.
  - `Update(tea.Msg) (PickerMultiModel, tea.Cmd)`: type-switch on `tea.KeyPressMsg` (NOT
    `tea.KeyMsg`): `j`/`k` navigate; `Space` toggles `selected[cursor]`; `Enter` →
    `done=true`, return `m, nil`; `Escape` → `cancelled=true, done=true`, return `m, nil`.
    **NEVER `return m, tea.Quit`**.
  - `View() string`: renders list with `[ ]`/`[x]` checkbox per item + `>` cursor.
  - `Selected() []string` returns selected items in original order;
    `Done() bool`, `Cancelled() bool` accessors.
  - Return type: `(PickerMultiModel, tea.Cmd)` — concrete, NOT `tea.Model`.

- `picker_multi_test.go`:
  - File-level migration marker.
  - `TestPickerMultiModel_Toggle`: Space toggles selection on/off.
  - `TestPickerMultiModel_Navigation`: j/k cursor moves.
  - `TestPickerMultiModel_Confirm`: Enter returns correct selected set (`Done()=true`,
    `Cancelled()=false`).
  - `TestPickerMultiModel_Cancel`: Escape sets `Cancelled()=true`, `Selected()` returns empty.
  - Assert no case returns non-nil cmd.

AcceptanceCriteria:
- Both `picker_single.go` and `picker_multi.go` compile alongside all prior components in
  `internal/tui/components`.
- `PickerSingleModel` is a Bubble Tea sub-component: `Update(tea.Msg) (PickerSingleModel, tea.Cmd)`,
  `View() string`; `Selected()` returns chosen item; `Done()` accessor exists. NOT `tea.Model`.
- `PickerMultiModel` is a Bubble Tea sub-component: `Update(tea.Msg) (PickerMultiModel, tea.Cmd)`,
  `View() string`; `Selected() []string` returns selected items; `Done()` and `Cancelled()`
  accessors exist. NOT `tea.Model`.
- All 4 test files pass; no test row returns non-nil cmd.
- Migration markers present in all 4 files as file-level comments before `package components`
  (build-QA-proof checks each file in Paths explicitly).
- `mage test-pkg ./internal/tui/components` passes (full package, ≥70% coverage).

Mage verification: `mage test-pkg ./internal/tui/components`

---

### D5 — Components: Header + Footer

- **Kind:** `build`
- **Irreducible:** `true`
- **State:** `todo`
- **Role:** builder

**Paths:**
- `internal/tui/components/header.go` (NEW)
- `internal/tui/components/footer.go` (NEW)

**Packages:** `internal/tui/components` (existing after D4; D5 adds to it)

**Blocked by:** D4 (same package compile)

**Specify:**

Objective: Add `Header` and `Footer` styled chrome types to `internal/tui/components`.
These are simple passive render structs — no state machine, no `Update()`, no `Init()`.
Combined in one droplet (2 files, each 1-2 code blocks).

KindPayload changes:
- `header.go`:
  - File-level `// MIGRATION TARGET: github.com/hylla-org/lykta` before `package components`.
  - ```go
    // Header renders a styled top chrome bar.
    type Header struct {
        title    string
        subtitle string
        width    int
    }
    ```
  - `func NewHeader(title, subtitle string, width int) Header`
  - `View() string` — renders a full-width bar with `title` left-aligned, `subtitle`
    right-aligned. Use inline lipgloss styles (builder may optionally import
    `internal/tui/style` for `style.Heading`/`style.Body` — D5 has no explicit `blocked_by D1`
    but style package will be complete by Wave A close; if builder imports it, confirm D1 is
    `done` before dispatching D5; otherwise inline styles are always safe).
  - `WithWidth(w int) Header` — returns copy with updated width.
- `footer.go`:
  - File-level `// MIGRATION TARGET: github.com/hylla-org/lykta` before `package components`.
  - ```go
    // Footer renders a styled bottom chrome bar with key hints.
    type Footer struct {
        hints []string
        width int
    }
    ```
  - `func NewFooter(hints []string, width int) Footer`
  - `View() string` — renders hints as a horizontal list, muted style.
  - `WithWidth(w int) Footer` — returns copy.

Note on coverage: `header.go` and `footer.go` have no co-located test files in the L1 paths
list. Builder MUST verify `mage test-pkg ./internal/tui/components` still passes ≥70% coverage
with the full package's test suite (D2 + D3 + D4 tests carry the package). If coverage drops
below 70%, builder has EXPLICIT AUTHORITY to create `header_test.go` + `footer_test.go` smoke
tests in the same package — these are NOT in the L1 paths list but within D5's authority per
the accepted planner note. Builder records any such additions in the worklog.

AcceptanceCriteria:
- Both `header.go` and `footer.go` compile with the full `internal/tui/components` package.
- `Header` has `View() string` and `WithWidth(int) Header`. No `Init()` or `Update()`.
- `Footer` has `View() string` and `WithWidth(int) Footer`. No `Init()` or `Update()`.
- Migration markers present in both files as file-level comments before `package components`
  (build-QA-proof checks each file in Paths explicitly).
- `mage test-pkg ./internal/tui/components` passes (full package, ≥70% coverage).

Mage verification: `mage test-pkg ./internal/tui/components`

Note on AC8 / `mage ci`: AC8 (`mage ci` green) is a DROP-END gate, not a per-D5 gate. The
orchestrator runs `mage ci` after BOTH D5 AND D6 are in `done` state. D5 runs only
`mage test-pkg ./internal/tui/components`; it does not run `mage ci` independently (D6 may
still be in-flight when D5 closes, since D6 is a separate unblocked package).

---

### D6 — Vim Keybinding Dispatcher

- **Kind:** `build`
- **Irreducible:** `true`
- **State:** `todo`
- **Role:** builder

**Paths:**
- `internal/tui/keybindings/dispatcher.go` (NEW)
- `internal/tui/keybindings/loader.go` (NEW)
- `internal/tui/keybindings/modes.go` (NEW)
- `internal/tui/keybindings/dispatcher_test.go` (NEW)

**Packages:** `internal/tui/keybindings` (NEW package — D6 creates it)

**Blocked by:** — (no blockers; new separate package; independent of components and style)

**Specify:**

Objective: Create `internal/tui/keybindings` — the Go-side vim keybinding dispatcher.
The package loads the stil baseline's 4 Tillsyn commands (embedded as nested-schema JSON
bytes, NOT from the sibling-repo filesystem) plus optionally ID-merges the project-local
`.tillsyn/bindings.json` for 5 additional commands (9 total). The `Dispatcher` routes
`tea.KeyMsg` events to registered handler functions per mode. Multi-key nav-mode bindings
are intentionally deferred (KEYBIND-R4).

Critical design decisions:
- Both baseline embed and local file use the SAME nested schema:
  `{"product_extensions":{"tillsyn":{"commands":[...]}}}`. One decoder, one code path.
- The embedded `stilBaselineTillsynJSON` bytes in `loader.go` wrap the 4 verified commands
  in this nested schema form.
- Multi-key commands (`new-drop` keys `["Space","n"]`, `complete-drop` keys `["Space","c"]`)
  are NOT registered in `bindings[ModeNav]` by `NewDispatcher`. They remain in
  `Bindings.Commands` for future KEYBIND-R4 leader-key dispatch. A `// TODO(KEYBIND-R4): ...`
  comment documents this in `dispatcher.go`.
- `Dispatch(msg tea.KeyMsg, mode Mode) HandlerFunc` resolves on `msg.String()` for single-key
  bindings only. Multi-key commands return `NoOp` from `Dispatch`.
- `DispatchCommand(id string) HandlerFunc` allows command-mode lookup by ID for all commands
  (including multi-key ones).

KindPayload changes:
- `modes.go`:
  - Two separate `//` comment lines above `package keybindings`: first line is the migration
    marker `// MIGRATION TARGET: github.com/hylla-org/lykta`, second line is the package doc
    comment `// Package keybindings provides a vim-style keybinding dispatcher for Tillsyn's TUI.`
  - ```go
    // Mode represents a vim-style input mode.
    type Mode int

    const (
        ModeNav Mode = iota  // default; letter and arrow keys navigate
        ModeInsert           // text input active; single-key bindings disabled
        ModeVisual           // item selection
        ModeVisualBlock      // block selection
        ModeCommand          // after `:` — command palette active
        ModeHint             // after `f`/`F` — overlay codes on clickable elements
    )
    ```
  - `func (m Mode) String() string` — returns lowercase mode name.
  - `HandlerFunc` type alias: `type HandlerFunc func() tea.Cmd`
  - `NoOp HandlerFunc = func() tea.Cmd { return nil }` — the no-op handler returned
    when no binding is registered.
  - Note: Mode constants only; mode-transition validation is the CALLER's responsibility,
    not this package's. `Dispatcher` is stateless mode-routing only.

- `loader.go`:
  - File-level `// MIGRATION TARGET: github.com/hylla-org/lykta` before `package keybindings`.
  - `// TODO(KEYBIND-R3): refresh embedded stil baseline bytes when stil-solid v<X> publishes`
    comment near the `stilBaselineTillsynJSON` declaration.
  - Embedded baseline bytes (nested schema form — see reference above for exact shape):
    ```go
    var stilBaselineTillsynJSON = []byte(`{
      "product_extensions": {
        "tillsyn": {
          "commands": [
            {"id":"new-drop",      "keys":["Space","n"], "description":"New drop in current project."},
            {"id":"complete-drop", "keys":["Space","c"], "description":"Mark drop complete."},
            {"id":"handoff",       "command":"handoff",  "description":"Open handoff dialog for current drop."},
            {"id":"comment",       "command":"comment",  "description":"Add a comment thread to current drop."}
          ]
        }
      }
    }`)
    ```
  - `// Command is a parsed command-palette entry from stil's product_extensions.`
    ```go
    type Command struct {
        ID          string   `json:"id"`
        Keys        []string `json:"keys,omitempty"`
        CommandName string   `json:"command,omitempty"`
        Description string   `json:"description"`
    }
    ```
  - `// Bindings holds the merged command set.`
    ```go
    type Bindings struct {
        Commands []Command
    }
    ```
  - Internal struct to parse both baseline and local JSON (same shape):
    ```go
    type bindingsFile struct {
        ProductExtensions struct {
            Tillsyn struct {
                Commands []Command `json:"commands"`
            } `json:"tillsyn"`
        } `json:"product_extensions"`
    }
    ```
  - `func LoadBindings(baselineJSON []byte, localPath string) (Bindings, error)`:
    - Parses `baselineJSON` via `bindingsFile`; extracts `product_extensions.tillsyn.commands`
      (4 baseline entries).
    - If `localPath != ""`: attempts `os.Open(localPath)`; if `os.IsNotExist(err)`: returns
      baseline-only, nil error (graceful fallback, NOT fail-loud). If file exists but is
      malformed JSON: returns error.
    - ID-deep-merge: for each local command, if its ID matches a baseline command, local
      replaces the baseline entry; otherwise append. Local wins on collision.
    - Returns `Bindings{Commands: merged}` — 9 commands when local present, 4 when absent.
    - Local file uses the SAME `bindingsFile` schema. The canonical 5-entry local file shape
      is: `{"schema_version":1,"product_extensions":{"tillsyn":{"commands":[...5 entries...]}}}`.
      Extra top-level fields (`schema_version`, `name`, `description`, `extends`) are ignored
      by the decoder — `bindingsFile` only maps `product_extensions`.
  - `func DefaultBaselineJSON() []byte` — returns `stilBaselineTillsynJSON` bytes.

- `dispatcher.go`:
  - File-level `// MIGRATION TARGET: github.com/hylla-org/lykta` before `package keybindings`.
  - ```go
    // Dispatcher routes tea.KeyMsg events to registered handlers per mode.
    type Dispatcher struct {
        bindings map[Mode]map[string]HandlerFunc // mode → key_string → handler
        commands map[string]HandlerFunc          // command-id → handler
    }
    ```
  - `func NewDispatcher(b Bindings) *Dispatcher` — builds the routing table from `b.Commands`:
    - For commands with `Keys` array of length == 1: registers `keys[0]` in
      `bindings[ModeNav]`.
    - For commands with `Keys` array of length > 1 (multi-key, e.g. `["Space","n"]`): SKIP
      registration in `bindings[ModeNav]`. Add a `// TODO(KEYBIND-R4): implement leader-key
      state machine for multi-key nav bindings (e.g., ["Space","n"])` comment inline.
    - For commands with `CommandName != ""`: registers in `commands[cmd.ID]`.
  - `func (d *Dispatcher) Register(mode Mode, key string, h HandlerFunc)` — explicit override.
  - `func (d *Dispatcher) Dispatch(msg tea.KeyMsg, mode Mode) HandlerFunc` — looks up
    `msg.String()` in `bindings[mode]`; if found, returns handler; else returns `NoOp`.
    Multi-key commands not in `bindings` return `NoOp` (correct — KEYBIND-R4 pending).
  - `func (d *Dispatcher) DispatchCommand(id string) HandlerFunc` — looks up `id` in
    `commands`; if found, returns handler; else returns `NoOp`.

- `dispatcher_test.go`:
  - File-level `// MIGRATION TARGET: github.com/hylla-org/lykta` before `package keybindings`.
  - `TestLoadBindings_BaselineOnly`: `LoadBindings(DefaultBaselineJSON(), "")` returns
    4 commands; no error.
  - `TestLoadBindings_WithLocal`: provide a temp file with 5-command JSON (using the canonical
    local shape from REVISION_BRIEF §2.19: `dispatch`, `plan`, `archive`, `settings`, `help`
    as command-mode-only entries with `CommandName` field); result has 9 commands; no error.
    Test fixture JSON: `{"product_extensions":{"tillsyn":{"commands":[{"id":"dispatch","command":"dispatch","description":"..."},...]}}}`.
  - `TestLoadBindings_LocalWins`: provide a temp file with a command whose ID collides with
    `handoff` (a baseline command); local `description` wins.
  - `TestLoadBindings_MissingLocalFile`: provide a non-existent path; returns baseline-only
    (4 commands); no error.
  - `TestDispatcher_Dispatch`: NewDispatcher with 4-command baseline bindings; `handoff` and
    `comment` are command-mode only (no `keys`) — `Dispatch` for any nav-mode key returns
    `NoOp`; explicit `Register` of a single-key binding (`"j"`, ModeNav) then `Dispatch` for
    that key returns the registered handler.
  - `TestDispatcher_MultiKey_Returns_NoOp`: `new-drop` (keys `["Space","n"]`) and
    `complete-drop` (keys `["Space","c"]`) are multi-key; `Dispatch(tea.KeyPressMsg{...Space...}, ModeNav)`
    returns `NoOp` (not registered in nav-mode bindings — KEYBIND-R4 deferred).
  - `TestDispatcher_Register`: explicit `Register` overrides default; `Dispatch` returns the
    overridden handler.
  - `TestDispatcher_DispatchCommand`: `DispatchCommand("handoff")` on a dispatcher built from
    4-command baseline returns a non-NoOp handler (after explicit `Register` for that id).
  - All tests table-driven where input variants exist.

AcceptanceCriteria:
- `modes.go` exports: `Mode` type, 6 `Mode*` constants, `HandlerFunc`, `NoOp`.
  Package-doc line and migration marker are TWO separate `//` comment lines above
  `package keybindings`.
- `loader.go` exports: `Command`, `Bindings`, `LoadBindings`, `DefaultBaselineJSON`.
  Contains `// TODO(KEYBIND-R3): refresh embedded stil baseline bytes when stil-solid v<X>
  publishes` grep-discoverable comment.
- `dispatcher.go` exports: `Dispatcher`, `NewDispatcher`, `(*Dispatcher).Register`,
  `(*Dispatcher).Dispatch`, `(*Dispatcher).DispatchCommand`. Contains
  `// TODO(KEYBIND-R4): implement leader-key state machine...` comment.
- `dispatcher_test.go` passes: all 7 test cases above.
- `LoadBindings(DefaultBaselineJSON(), "")` returns exactly 4 commands.
- `LoadBindings(DefaultBaselineJSON(), nonExistentPath)` returns 4 commands, nil error.
- Multi-key commands (`new-drop`, `complete-drop`) return `NoOp` from `Dispatch` in nav mode.
- Migration markers present in all 4 files as file-level comments before `package keybindings`
  (build-QA-proof checks each file in Paths explicitly).
- `mage test-pkg ./internal/tui/keybindings` passes (≥70% coverage).
- `mage ci` green (run by orchestrator after BOTH D5 and D6 are `done` — this is AC8).

Mage verification: `mage test-pkg ./internal/tui/keybindings` (per-droplet); `mage ci`
(drop-end gate after D5 + D6 both done — orchestrator runs this, not the builder)

---

## KindPayload — Plan Preview

```json
{
  "children": [
    {"id": "D1", "kind": "build", "title": "STYLE SYSTEM", "paths": ["internal/tui/style/palette.go", "internal/tui/style/palette_test.go", "internal/tui/style/spacing.go", "internal/tui/style/typography.go"], "packages": ["internal/tui/style"], "blocked_by": []},
    {"id": "D2", "kind": "build", "title": "COMPONENTS: CONFIRM + PROGRESS", "paths": ["internal/tui/components/confirm.go", "internal/tui/components/confirm_test.go", "internal/tui/components/progress.go"], "packages": ["internal/tui/components"], "blocked_by": []},
    {"id": "D3", "kind": "build", "title": "COMPONENTS: TEXTINPUT", "paths": ["internal/tui/components/textinput.go", "internal/tui/components/textinput_test.go"], "packages": ["internal/tui/components"], "blocked_by": ["D2"]},
    {"id": "D4", "kind": "build", "title": "COMPONENTS: PICKERS (SINGLE + MULTI)", "paths": ["internal/tui/components/picker_single.go", "internal/tui/components/picker_single_test.go", "internal/tui/components/picker_multi.go", "internal/tui/components/picker_multi_test.go"], "packages": ["internal/tui/components"], "blocked_by": ["D3"]},
    {"id": "D5", "kind": "build", "title": "COMPONENTS: HEADER + FOOTER", "paths": ["internal/tui/components/header.go", "internal/tui/components/footer.go"], "packages": ["internal/tui/components"], "blocked_by": ["D4"]},
    {"id": "D6", "kind": "build", "title": "VIM KEYBINDING DISPATCHER", "paths": ["internal/tui/keybindings/dispatcher.go", "internal/tui/keybindings/loader.go", "internal/tui/keybindings/modes.go", "internal/tui/keybindings/dispatcher_test.go"], "packages": ["internal/tui/keybindings"], "blocked_by": []}
  ]
}
```

## CompletionContract

**StartCriteria:**
- W5 sub-plan container in `planning` state.
- No prior files exist at `internal/tui/components/`, `internal/tui/style/`,
  `internal/tui/keybindings/` — builder must not find pre-existing files (new packages).

**CompletionCriteria:**
- All 6 droplets in `done` state.
- `mage ci` green on the full repo after BOTH D5 and D6 complete.
- All build-QA proof + falsification rounds passed for each droplet.

**CompletionChecklist:**
- [ ] D1 (style) build-QA proof passed
- [ ] D1 (style) build-QA falsification passed
- [ ] D2 (confirm+progress) build-QA proof passed
- [ ] D2 (confirm+progress) build-QA falsification passed
- [ ] D3 (textinput) build-QA proof passed
- [ ] D3 (textinput) build-QA falsification passed
- [ ] D4 (pickers) build-QA proof passed
- [ ] D4 (pickers) build-QA falsification passed
- [ ] D5 (header+footer) build-QA proof passed
- [ ] D5 (header+footer) build-QA falsification passed
- [ ] D6 (keybindings) build-QA proof passed
- [ ] D6 (keybindings) build-QA falsification passed
- [ ] `mage ci` green (orchestrator runs after BOTH D5 + D6 done)

---

## Supporting Files

### `_BLOCKERS.toml`

See `_BLOCKERS.toml` in this directory. D1 and D6 have no blockers and are not listed there.
`_BLOCKERS.toml` mirrors the `Blocked by:` bullets above; PLAN.md is truth on conflicts.

---

## Wave-Boundary Concerns

1. **W2 dispatch readiness**: W2 is blocked by W5 at the L1 level. Within W5, W2's actual
   critical path is D2 → D3 → D4 (three serialized droplets). W2 cannot dispatch before D4
   reaches `done`. Orch should note that W5 closing is necessary but the D4 completion
   milestone is the meaningful unblock point for W2.

2. **D6 baseline embedding**: the `stilBaselineTillsynJSON` bytes hardcoded in `loader.go`
   must exactly match the 4 Tillsyn commands in `baseline.json` at the time of build. If stil
   updates its baseline, this embedded copy becomes stale. `// TODO(KEYBIND-R3)` comment in
   `loader.go` flags this for grep-discoverability. KEYBIND-R3 (tracked) captures the move to
   stil-solid's package artifact when stable. Until then, this is an accepted known risk.

3. **teatest import path**: `"github.com/charmbracelet/x/exp/teatest/v2"` — the go.mod
   `replace` directive maps this to `./third_party/teatest_v2`. Builder imports this path;
   the replace directive handles the redirect. No additional setup needed.

4. **Coverage gates for pure-constant packages**: `internal/tui/style` has no behavioral
   logic by default. `palette_test.go` with `TestAllColors_NonEmpty` is MANDATORY in D1 Paths
   to satisfy the coverage threshold. The magefile errors on "no coverage rows parsed" if
   there are zero test functions.

5. **AC8 gate ownership**: `mage ci` (AC8) is a drop-end gate owned by the orchestrator,
   not by D5 or D6 individually. D5 runs `mage test-pkg ./internal/tui/components`; D6 runs
   `mage test-pkg ./internal/tui/keybindings`. After BOTH are done, orch runs `mage ci`.

6. **Multi-key nav bindings deferred**: KEYBIND-R4. The two Space-leader commands
   (`new-drop`, `complete-drop`) are NOT routable via single-call `Dispatch`. They are
   available via `DispatchCommand` by ID (command-mode routing). This is the correct behavior
   until KEYBIND-R4's leader-key state machine lands.
