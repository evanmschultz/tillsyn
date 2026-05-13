# DROP_4c.6.1.W2_TILL_INIT — TILL_INIT MULTI-GROUP OVERHAUL

**State:** planning
**Wave:** C (blocked by W1 + W4.D1 + W5; blocks W3, W7.D3 — cmd/till package compile lock)
**Blocked by:** 4c.6.1.W1 (HOME-tier path convention), 4c.6.1.W4.D1 (agent subdir layout + canonical group names), 4c.6.1.W5 (TUI components: picker_multi.go + confirm.go)
**Blocks:** 4c.6.1.W3 (cmd/till package), 4c.6.1.W7.D3 (cmd/till package compile lock)
**Paths:** `cmd/till/init_cmd.go`, `cmd/till/init_cmd_test.go`
**Packages:** `cmd/till`
**Source-of-truth:** REVISION_BRIEF §2.3–2.6; SKETCH §2 + §4.1; L1 PLAN.md lines 219–253

## Round 2 Changes (L2 re-plan absorbing round-1 plan-QA findings + R10 decisions — 2026-05-12)

All findings from round-1 proof (PASS-WITH-CONDITIONS: 1 FF + 7 NITs) and round-1 falsification (FAIL: 3 FFs + 7 NITs) absorbed. R10 cross-cutting decisions applied. Summary:

- **Proof FF1 = Fals FF2 (D7 KindPayload JSON stopgap)**: ABSORBED per R10-D2. D7 acceptance + RiskNotes + ContextBlocks + KindPayload now specify `Metadata.Groups = payload.Groups` (typed field shipped by W1.D2). `KindPayload` stopgap removed. W2-GROUPS-R1 refinement RESOLVED inline — W1.D2 ships the typed field before D7 dispatches.
- **Fals FF1 (till-prefix drift in W4.D1)**: ABSORBED per R10-D1. W4.D1 performs `git mv till-go → go` + `git mv till-gen → gen`. Canonical group names are `go`, `fe`, `gen` (no `till-` prefix). D1's `allowedInitGroups` uses these names. D5's embed path uses `builtin/agents/<group>/` (unprefixed post-W4.D1). All references to `till-gen`/`till-go` removed from W2 spec.
- **Fals FF3 = Proof NIT1 (_BLOCKERS.toml W2.D1 entry missing)**: RESOLVED UNILATERALLY by orchestrator before round-2 dispatch. `_BLOCKERS.toml` already has the W2.D1 entry. Marked RESOLVED here.
- **Proof NIT2 (D1 missing CONSUMER-TIE bullet)**: ABSORBED. D1 acceptance now includes explicit `run(ctx, args, ...)` CONSUMER-TIE bullet.
- **Proof NIT3 (D3/D4 missing JSON-mode run() mirror)**: ABSORBED. D3 + D4 acceptance now include explicit JSON-mode `run()` supplement bullets.
- **Proof NIT4 (D7 wrong cite for CreateProjectInput)**: ABSORBED. D7 reference corrected: `internal/app/service.go:286` (`CreateProjectInput`), `internal/domain/project.go:212` (`ProjectInput`).
- **Proof NIT5 (D5 missing explicit blocked_by W4.D1)**: ABSORBED. D5 `Blocked by` now includes explicit `4c.6.1.W4.D1` (consistent with D3/D4's explicit W5 cross-wave pattern).
- **Proof NIT6 (agents.toml multi-group aggregation gap)**: DEFERRED. Gap is faithful to L1 contract (L1 acceptance bullets don't include `agents.toml` multi-group aggregation). Surfaced to orchestrator in closing summary. See Notes §agents.toml Gap.
- **Proof NIT7 = Fals NIT3 (MCP bool vs *bool)**: ABSORBED. `initJSONPayload.MCP` changes from `bool` to `*bool` in **D1** (first droplet touching `initJSONPayload`). `MCPRegistration() bool` accessor added (nil→true default). D4 reads via `payload.MCPRegistration()`. D1 KindPayload updated.
- **Fals NIT1 (coarse blocked_by W5 for D3/D4)**: DOCUMENTED. D3 only needs `picker_multi.go` (W5.D4); D4 only needs `confirm.go` (W5.D2). Tightening to droplet-level external blockers is a dispatch-optimization; pre-Drop-4b dispatcher enforces at wave level. Kept at wave level for L2 simplicity. Tracked as dispatch-optimization note.
- **Fals NIT2 (D7 KindPayload field carries stopgap)**: ABSORBED with FF2 — D7 KindPayload updated.
- **Fals NIT4 (D6 partial-state idempotency)**: ABSORBED. D6 acceptance: blanket file-level skip + fail-loud warning if file exists AND one or more selected groups are absent from the existing file. Simpler than per-group block parsing; consistent with no-migration philosophy.
- **Fals NIT5 (D7 Language mapping selection-order)**: ABSORBED. D7 Language mapping uses `payload.Groups[0]` (selection-order wins). Tradeoff documented in RiskNotes.
- **Fals NIT6 (reservedInitGroups ambiguous fate)**: ABSORBED. D1 acceptance explicitly pins deletion of `reservedInitGroups` map and its doc-comment.
- **Fals NIT7 (Disabled bool dead field fate)**: ABSORBED. D1 acceptance: keep `Disabled bool` on `initTUIGroupRow` for D1→D3 interim (all rows enabled, field is inert). D3 deletes it with the full picker_multi.go replacement.

---

## Planner

### Decomposition Shape

Seven atomic droplets in a strict serial chain. All share `cmd/till/init_cmd.go` +
`cmd/till/init_cmd_test.go` (same package compile unit → full serialization required).
D1→D7 is the only valid dispatch order. No parallelism within this wave.

| Droplet | Title | Blocked by |
|---------|-------|------------|
| W2.D1 | Payload Group→Groups + group names + validation + MCP *bool | 4c.6.1.W4.D1 |
| W2.D2 | FLAT layout detection + old-schema agents.toml detection | W2.D1 |
| W2.D3 | Multi-select TUI picker for group selection | W2.D2, 4c.6.1.W5 |
| W2.D4 | TUI MCP confirm step | W2.D3, 4c.6.1.W5 |
| W2.D5 | copyAgentFiles refactor to subdir-per-group | W2.D4, 4c.6.1.W4.D1 |
| W2.D6 | template.toml write (aggregate from HOME or embedded per group) | W2.D5 |
| W2.D7 | createProjectDBRecord upgrade to CreateProjectWithMetadata | W2.D6 |

`_BLOCKERS.toml` is at this dir level (7 immediate children — already reflects W2.D1 entry added in round-2 pre-dispatch fix).

---

### W2.D1 — PAYLOAD GROUP→GROUPS + GROUP NAMES + VALIDATION + MCP *BOOL

- **State:** todo
- **Kind:** `build` (atomic droplet; `Irreducible: true`)
- **Paths:**
  - `cmd/till/init_cmd.go` (MODIFY)
  - `cmd/till/init_cmd_test.go` (MODIFY — update existing group-validation + MCP tests)
- **Packages:** `cmd/till`
- **Blocked by:** 4c.6.1.W4.D1 (canonical group names `go`, `fe`, `gen` confirmed by W4.D1's `git mv till-go→go` + `git mv till-gen→gen`; D1 must not use stale `till-gen`/`till-go` names)
- **Acceptance:**
  - `initJSONPayload.Group string` field renamed to `Groups []string`. JSON tag changes from `"group"` to `"groups"`. Old `"group"` key in `--json` payloads no longer accepted.
  - `initJSONPayload.MCP bool` changes to `MCP *bool` with JSON tag `"mcp,omitempty"`. A helper `func (p initJSONPayload) MCPRegistration() bool { if p.MCP == nil { return true }; return *p.MCP }` is added, mirroring `OrchSelfApprovalIsEnabled()` pattern (`internal/domain/project.go:157`). Omitting `"mcp"` from a `--json` payload defaults to YES (MCP registration enabled).
  - `allowedInitGroups` changes from `["till-gen", "till-go"]` to `["gen", "go", "fe"]`.
  - `reservedInitGroups` map AND its doc-comment are DELETED entirely. The reserved-group validation branch in `validateInitPayload` is also removed. If a future group needs reservation, the validation can re-introduce it.
  - `validateInitPayload` updated: checks `len(p.Groups) > 0`, each group in `Groups` must be in `allowedInitGroups`. Returns a clear error listing invalid groups.
  - `initTUIGroupRows` updated to three rows (gen, go, fe), all enabled. The `Disabled bool` field on `initTUIGroupRow` is KEPT for the D1→D3 interim (all rows have `Disabled: false` — the field is inert but the struct shape is preserved; D3 removes it when `picker_multi.go` takes over).
  - `nextEnabledGroupRow` and `prevEnabledGroupRow` helpers remain (unchanged) — their skip-disabled logic is a no-op since all rows are enabled. D3 removes them.
  - `initTUIModel.finalPayload.Group = row.Name` (line ~235) becomes `finalPayload.Groups = []string{row.Name}` (single-value slice bridge until D3 ships multi-select).
  - All doc-comments referencing `Group string`, `till-gen`, `till-go`, `"till-gdd"`, `reservedInitGroups` are updated or removed.
  - CONSUMER-TIE: validation behavior tested via `run(ctx, args, &out, io.Discard)` end-to-end — at minimum: (a) valid single-group `--json '{"name":"x","groups":["go"]}'` (no `mcp` key — verifies nil→true default); (b) valid multi-element `--json '{"name":"x","groups":["go","fe"],"mcp":false}'`; (c) invalid group `--json '{"name":"x","groups":["bogus"]}'` expects non-zero exit + error substring "invalid". Unit assertions on `validateInitPayload` are acceptable as supplement.
  - `mage test-pkg ./cmd/till` passes with all existing and new tests green. `mage ci` green.
- **Specify:**
  - **Objective:** Change `initJSONPayload` from single `Group string` to `Groups []string`, change `MCP bool` to `MCP *bool` with nil→true default accessor, and update `allowedInitGroups` to the new three canonical group names (`gen`, `go`, `fe`) without the `till-` prefix per W4.D1's `git mv`. Delete `reservedInitGroups`. This is the foundational change that all downstream D2–D7 droplets depend on — all code in `init_cmd.go` that reads `payload.Group` must become `payload.Groups`, and `payload.MCP` becomes `payload.MCPRegistration()`.
  - **AcceptanceCriteria:**
    - `initJSONPayload.Groups []string` replaces `initJSONPayload.Group string`. JSON tag `"groups"`.
    - `initJSONPayload.MCP *bool` with JSON tag `"mcp,omitempty"`. `MCPRegistration() bool` accessor: nil→true.
    - `allowedInitGroups = []string{"gen", "go", "fe"}`.
    - `reservedInitGroups` map + doc-comment DELETED. Reserved-group branch in `validateInitPayload` removed.
    - `validateInitPayload` validates `len(Groups) >= 1` and each element in allowed list.
    - `initTUIGroupRows` has three rows (gen/go/fe), all enabled. `Disabled bool` field KEPT on struct (inert, D3 removes it).
    - `initTUIModel.finalPayload.Groups = []string{row.Name}` (single-value bridge until D3).
    - All references to old names (`till-gen`, `till-go`, `till-gdd`) removed from production code and tests.
    - CONSUMER-TIE: three `run()` end-to-end tests (valid single-group no-mcp-key, valid multi-group mcp-false, invalid group error).
    - `mage test-pkg ./cmd/till` green; `mage ci` green.
  - **ValidationPlan:** `mage test-pkg ./cmd/till`; `mage ci`.
  - **RiskNotes:**
    - `initJSONPayload.Group` is read in at least `runInitPipeline` (passes to `copyAgentFiles`) and the Laslig summary (line ~452). Builder must search ALL references to `payload.Group` in `init_cmd.go` via LSP `findReferences` before renaming.
    - `initTUIModel.finalPayload.Group = row.Name` (line ~235) becomes `finalPayload.Groups = []string{row.Name}` in D1's TUI stub. Full multi-select lands in D3 — D1 uses single-value `Groups` slice as the bridge, consistent with the D3 blocker.
    - `reservedInitGroups` deletion: confirm via LSP that `reservedInitGroups` is only referenced in `validateInitPayload` before deleting. If referenced elsewhere, delete all references.
    - After D1, `runInitPipeline` still calls `copyAgentFiles(destDir, payload.Groups[0])` (single-group stub) — D5 upgrades this to the full multi-group loop. D1 must NOT break the compile in the interim.
    - MCP `*bool` change: `payload.MCPRegistration()` accessor is the ONLY correct call site for reading MCP intent. Do NOT read `payload.MCP` directly anywhere in D1's edits. D4 and later droplets also use `MCPRegistration()`.
    - W4.D1 must be `complete` before D1 dispatches — confirms canonical group names `go`/`fe`/`gen` (no `till-` prefix).
  - **ContextBlocks:**
    - `constraint` (critical): D1 must not break `mage test-pkg ./cmd/till`. Every change is in `init_cmd.go` + `init_cmd_test.go` only. No other files touched.
    - `constraint` (critical): `reservedInitGroups` — DELETE entirely. Do NOT keep empty. Rationale: the rationale for its existence (till-gdd reserved-but-not-shipped) evaporates with the new naming scheme. Empty map is dead code.
    - `decision` (normal): TUI single-select stays single-value in D1 (`Groups = []string{row.Name}`). Full multi-select UI (D3) is blocked by W5 and arrives later in the chain.
    - `decision` (normal): `Disabled bool` field on `initTUIGroupRow` is kept for D1→D3 interim (all rows `Disabled: false` — inert). D3 removes it wholesale when `picker_multi.go` replaces the inline picker.
    - `decision` (normal): `MCP *bool` nil→true mirrors the `OrchSelfApprovalEnabled *bool` pattern at `internal/domain/project.go:128`. `MCPRegistration() bool` is the read accessor. This is the only spec-consistent answer for "default true if absent" in JSON mode.
    - `reference` (normal): REVISION_BRIEF §2.3 + §2.6; L1 PLAN.md lines 249–256; locked-decision "Group names: go, fe, gen (no till- prefix)"; `internal/domain/project.go:128` (`OrchSelfApprovalEnabled *bool` pattern to mirror).
    - `warning` (high): The `till-gdd` reserved group row in `initTUIGroupRows` is removed in D1 — tests asserting "till-gdd is disabled" will break and must be updated.
    - `warning` (high): W4.D1's `git mv` is required. If W4.D1 has not shipped when D1 dispatches, the canonical group names are unconfirmed. Dispatch gate: W4.D1 must be `complete`.
  - **KindPayload:** `{"changes":[{"file":"cmd/till/init_cmd.go","symbol":"initJSONPayload","action":"modify","shape_hint":"Group string → Groups []string (json:groups); MCP bool → MCP *bool (json:mcp,omitempty); add MCPRegistration() bool accessor (nil→true)"},{"file":"cmd/till/init_cmd.go","symbol":"allowedInitGroups","action":"modify","shape_hint":"[\"gen\",\"go\",\"fe\"]"},{"file":"cmd/till/init_cmd.go","symbol":"reservedInitGroups","action":"delete","shape_hint":"delete var + doc-comment + reserved-group branch in validateInitPayload"},{"file":"cmd/till/init_cmd.go","symbol":"initTUIGroupRows","action":"modify","shape_hint":"3 rows all enabled: gen/go/fe; keep Disabled bool field on struct (D3 removes it)"},{"file":"cmd/till/init_cmd.go","symbol":"validateInitPayload","action":"modify","shape_hint":"validate len(Groups)>=1; each in allowedInitGroups; remove reserved-group branch"},{"file":"cmd/till/init_cmd_test.go","symbol":"TestValidateInitPayload + TestInitTUIGroupRows + TestMCPRegistration","action":"modify","shape_hint":"update expected groups; remove till-gdd disabled assertions; add MCPRegistration nil→true test; add 3 CONSUMER-TIE run() tests"}]}`

---

### W2.D2 — FLAT LAYOUT DETECTION + OLD-SCHEMA agents.toml DETECTION

- **State:** todo
- **Kind:** `build` (atomic droplet; `Irreducible: true`)
- **Paths:**
  - `cmd/till/init_cmd.go` (MODIFY)
  - `cmd/till/init_cmd_test.go` (MODIFY — add FLAT-detection and old-schema-detection tests)
- **Packages:** `cmd/till`
- **Blocked by:** W2.D1
- **Acceptance:**
  - `runInitPipeline` calls a new `detectFLATLayout(destDir string) error` function BEFORE calling `copyAgentFiles`. If `<destDir>/.tillsyn/agents/` contains `.md` files directly at root (FLAT layout), `runInitPipeline` returns a non-zero error: `"FLAT agent layout detected at <destDir>/.tillsyn/agents/. Remove it and re-run: rm -rf <destDir>/.tillsyn/agents && till init --group <group>"`.
  - `runInitPipeline` calls a new `detectOldSchemaAgentsTOML(destDir string) error` function BEFORE calling `copyAgentFiles`. If `<destDir>/agents.toml` exists and any of its first 20 lines (trimmed) starts with `[agents.`, returns a non-zero error: `"agents.toml uses the old [agents.kind] schema. Remove it and re-run: rm <destDir>/agents.toml && till init --group <group>"`.
  - Both checks run before any file-copy side effects. A project with neither condition passes through unaffected.
  - Re-run on clean-state (new schema subdir layout): both checks pass (no error).
  - CONSUMER-TIE: tests via `run(ctx, args, &out, io.Discard)` end-to-end — one test for FLAT layout present (expects non-zero + error substring), one test for old-schema `agents.toml` present (same), one test for clean state (both pass, exits zero).
  - `mage test-pkg ./cmd/till` passes; `mage ci` green.
- **Specify:**
  - **Objective:** Add fail-loud pre-flight checks to `runInitPipeline` for two known bad states: (1) FLAT agent layout from Drop 4c.6 or earlier sessions, (2) old `[agents.kind]` schema in `agents.toml`. Both fail with a clear error and remediation instruction — no migration code, no silent skip.
  - **AcceptanceCriteria:**
    - FLAT detection: reads `<destDir>/.tillsyn/agents/`; if any direct child is a `.md` regular file (not a subdirectory), returns error with exact message above.
    - Old-schema detection: reads first 20 lines of `<destDir>/agents.toml`; if any line stripped of leading whitespace starts with `[agents.`, returns error with exact message above. If file absent, check is a no-op.
    - Both checks run before `copyAgentFiles` call — no partial writes on failure.
    - CONSUMER-TIE: three `run()` end-to-end tests (FLAT layout, old-schema, clean state).
    - `mage test-pkg ./cmd/till` green; `mage ci` green.
  - **ValidationPlan:** `mage test-pkg ./cmd/till`; `mage ci`.
  - **RiskNotes:**
    - FLAT detection: `<destDir>/.tillsyn/agents/` may not exist on first run — the check is a no-op when the directory is absent (only trigger when it exists AND contains `.md` files at root).
    - Old-schema prefix is `[agents.` (with trailing dot) — matches `[agents.build]`, `[agents.plan]`, etc. Does NOT match `[agents]` (no dot) or `[go.build]` (new schema). Builder verifies the prefix string exactly.
    - The error messages contain `<destDir>` as a placeholder — actual implementation interpolates the real path.
    - Both detection functions are new, not yet in tree.
    - D5 refactors `copyAgentFiles` — the FLAT check is placed in `runInitPipeline` (not inside `copyAgentFiles`) so it survives the D5 rewrite independently.
  - **ContextBlocks:**
    - `constraint` (critical): NO migration code. Fail loud, give clear remediation, stop.
    - `decision` (normal): detection placed in `runInitPipeline` (pre-check before `copyAgentFiles`), NOT inside `copyAgentFiles`. Reason: D5 rewrites `copyAgentFiles`; placing detection in `runInitPipeline` makes D2's check D5-independent.
    - `reference` (normal): REVISION_BRIEF §2.3 FF2 disposition + NIT3; L1 PLAN.md lines 250–251.
    - `warning` (high): First 20 lines heuristic for old-schema detection — if a user has a long comment block before the first `[agents.X]` section, the check may miss. 20 lines is a reasonable pragmatic bound; doc-comment explains the heuristic.
  - **KindPayload:** `{"changes":[{"file":"cmd/till/init_cmd.go","symbol":"detectFLATLayout","action":"add","shape_hint":"func(destDir string) error — new, not yet in tree; reads .tillsyn/agents/; fails if *.md present at root"},{"file":"cmd/till/init_cmd.go","symbol":"detectOldSchemaAgentsTOML","action":"add","shape_hint":"func(destDir string) error — new, not yet in tree; reads first 20 lines of agents.toml; fails on [agents. prefix"},{"file":"cmd/till/init_cmd.go","symbol":"runInitPipeline","action":"modify","shape_hint":"add detectFLATLayout + detectOldSchemaAgentsTOML calls before copyAgentFiles"},{"file":"cmd/till/init_cmd_test.go","symbol":"TestRunInitPipeline_FLATDetection + TestRunInitPipeline_OldSchemaDetection","action":"add","shape_hint":"end-to-end via run(); new, not yet in tree; 3 cases"}]}`

---

### W2.D3 — MULTI-SELECT TUI PICKER FOR GROUP SELECTION

- **State:** todo
- **Kind:** `build` (atomic droplet; `Irreducible: true`)
- **Paths:**
  - `cmd/till/init_cmd.go` (MODIFY)
  - `cmd/till/init_cmd_test.go` (MODIFY — update TUI model tests for multi-select)
- **Packages:** `cmd/till`
- **Blocked by:** W2.D2, 4c.6.1.W5 (provides `internal/tui/components/picker_multi.go` — new, not yet in tree until W5 ships; note: D3 specifically needs W5.D4 which ships `picker_multi.go`; the wave-level blocker is sufficient pre-Drop-4b dispatcher)
- **Acceptance:**
  - `initTUIModel` replaces its single-select group-cursor step with the `picker_multi.go` component from `internal/tui/components`. All of `["gen", "go", "fe"]` are selectable (no disabled rows in the new model).
  - Space-bar (or equivalent per `picker_multi.go` API) toggles individual group selection. Enter confirms the selection.
  - `finalPayload.Groups` is set to the slice of all selected group names (minimum one required — model rejects empty selection with an inline hint and refuses to advance).
  - The default selection is `["gen"]` — first row pre-selected so one Enter accepts the default immediately.
  - The TUI model's View renders a multi-select group list (checked/unchecked rows visible).
  - Dead code removed: `initTUIGroupRows []initTUIGroupRow`, `initTUIGroupRow` struct (including `Disabled bool` field), `nextEnabledGroupRow`, `prevEnabledGroupRow`, `groupCursor int` — all replaced by the `picker_multi.go` component.
  - Tests: `teatest_v2` pattern (per existing `init_cmd_test.go` conventions) drives model directly to verify Done/Cancelled/Payload state.
  - CONSUMER-TIE supplement: `run(..., '--json', '{"name":"x","groups":["go","fe"],"mcp":false}')` exercises the multi-group payload path without entering the TUI; this is the JSON-mode mirror of D3's TUI multi-select.
  - `mage test-pkg ./cmd/till` passes; `mage ci` green.
- **Specify:**
  - **Objective:** Replace the single-select group picker in `initTUIModel` with the `picker_multi.go` component shipped by W5. This enables `till init` to collect multiple groups at once (e.g. `go` + `fe`). The component API (constructor signature, key-event contract) is defined by W5 and must be verified via LSP `documentSymbol` on `internal/tui/components/picker_multi.go` after W5 ships before authoring D3's imports.
  - **AcceptanceCriteria:**
    - `initTUIModel` contains a field of type `components.PickerMulti` (or equivalent exported type from `internal/tui/components` — verify name via LSP after W5 ships).
    - Multi-select: space toggles, Enter confirms, Esc cancels.
    - Minimum 1 group required — empty selection refuses to advance with visible hint.
    - Default selection: `["gen"]` pre-selected.
    - `Payload().Groups` is the slice of selected group names.
    - Dead code removed: `initTUIGroupRows`, `initTUIGroupRow` struct (and its `Disabled bool` field), `nextEnabledGroupRow`, `prevEnabledGroupRow`, `groupCursor int`.
    - TUI model tests: verify multi-select toggle behavior; verify default selection; verify minimum-1 enforcement.
    - CONSUMER-TIE supplement: `run(--json '{"name":"x","groups":["go","fe"],"mcp":false}')` passes.
    - `mage test-pkg ./cmd/till` green; `mage ci` green.
  - **ValidationPlan:** `mage test-pkg ./cmd/till`; `mage ci`.
  - **RiskNotes:**
    - `picker_multi.go` is new, not yet in tree (W5 ships it). Builder MUST run LSP `documentSymbol` on `internal/tui/components/picker_multi.go` after W5 completes to get the exact exported type name, constructor signature, and key-event contract before writing D3's import.
    - W5 components are Bubble Tea sub-models, NOT direct `tea.Model` implementors (per R10 locked decision — `View()` returns `tea.View` struct, not string; sub-models don't satisfy `tea.Model` directly). Builder does NOT write `var _ tea.Model = (*components.PickerMulti)(nil)`.
    - W5 components must NOT return `tea.Quit` (kills parent TUI). Use `return nil` + `Done()`/`Cancelled()` accessors instead (per R10 locked decision — W5 fals FF2 disposition).
    - The `initTUIStepGroup` step shape changes — it wraps `picker_multi.go`'s model rather than the inline cursor. The `Update` method dispatches to the component's `Update`.
  - **ContextBlocks:**
    - `constraint` (critical): D3 is `blocked_by 4c.6.1.W5`. Builder must NOT dispatch D3 until W5 is `complete`.
    - `constraint` (high): minimum-1 selection enforcement is a runtime invariant — `validateInitPayload` already checks `len(Groups) >= 1`; TUI must prevent submitting empty selection before that validation runs.
    - `constraint` (high): picker_multi component must NOT call `tea.Quit`. Use `Done()`/`Cancelled()` accessors + `return nil` cmd per R10 W5-fals-FF2 decision.
    - `decision` (normal): default selection is `["gen"]` — language-agnostic group is the safe default for any project.
    - `reference` (normal): REVISION_BRIEF §2.3; W5 L1 container for `picker_multi.go` API; `internal/tui/components/` (new package, not yet in tree).
    - `warning` (high): `picker_multi.go` type and constructor name are new, not yet in tree. Do NOT guess or hard-code them — verify via LSP after W5 ships.
  - **KindPayload:** `{"changes":[{"file":"cmd/till/init_cmd.go","symbol":"initTUIModel","action":"modify","shape_hint":"replace groupCursor int + initTUIGroupRows + initTUIGroupRow struct (including Disabled bool) with components.PickerMulti field; Update dispatches to component"},{"file":"cmd/till/init_cmd.go","symbol":"nextEnabledGroupRow + prevEnabledGroupRow","action":"delete","shape_hint":"dead code after picker_multi.go manages its own navigation"},{"file":"cmd/till/init_cmd_test.go","symbol":"TestInitTUIModel_GroupMultiSelect","action":"add","shape_hint":"new, not yet in tree; teatest_v2 drive; verify toggle + default + min-1"},{"file":"cmd/till/init_cmd_test.go","symbol":"TestRunInit_JSONMode_MultiGroup","action":"add","shape_hint":"CONSUMER-TIE supplement; run() with groups:[go,fe]; new, not yet in tree"}]}`

---

### W2.D4 — TUI MCP CONFIRM STEP

- **State:** todo
- **Kind:** `build` (atomic droplet; `Irreducible: true`)
- **Paths:**
  - `cmd/till/init_cmd.go` (MODIFY)
  - `cmd/till/init_cmd_test.go` (MODIFY — add MCP confirm step tests)
- **Packages:** `cmd/till`
- **Blocked by:** W2.D3, 4c.6.1.W5 (provides `internal/tui/components/confirm.go` — new, not yet in tree until W5 ships; note: D4 specifically needs W5.D2 which ships `confirm.go`; wave-level blocker is sufficient pre-Drop-4b dispatcher)
- **Acceptance:**
  - `initTUIModel` gains a new step (e.g. `initTUIStepMCP`) after the group-selection step. The step renders a y/n prompt using `confirm.go` from `internal/tui/components`.
  - Default answer is YES (`.mcp.json` registration default = true per REVISION_BRIEF §2.6).
  - Pressing Enter accepts the default (YES). Pressing y/Y explicitly sets YES. Pressing n/N sets NO. Pressing Esc cancels the walk.
  - `initJSONPayload.MCP` (the `*bool` field from D1) is set via the confirm response. In JSON mode (`runInitJSON`), the field is consumed via `payload.MCPRegistration()` — which returns true for nil (omitted field).
  - `initTUIModel.finalPayload.MCP = false` hard-wiring at line ~236 is REMOVED. Replaced by confirm component result: `finalPayload.MCP = &mcpYes` where `mcpYes bool` is set from the confirm step.
  - `initTUIStepDone` terminal state assignment now occurs after the MCP confirm step (not after the group step).
  - Tests: TUI model tests via `teatest_v2` verify MCP step transitions (Enter=YES, n=NO, Esc=cancel).
  - CONSUMER-TIE supplement: `run(..., '--json', '{"name":"x","groups":["go"],"mcp":true}')` (MCP=true path) + `run(..., '--json', '{"name":"x","groups":["go"],"mcp":false}')` (MCP=false path) + `run(..., '--json', '{"name":"x","groups":["go"]}')` (no `mcp` key — verifies nil→true default from D1's MCPRegistration) are all exercised and pass.
  - `mage test-pkg ./cmd/till` passes; `mage ci` green.
- **Specify:**
  - **Objective:** Add the MCP registration confirm step to the `till init` TUI walk. Currently `runInitTUI` hardwires `MCP = false`; this closes REVISION_BRIEF §2.6 by prompting the user with default YES. Uses `confirm.go` from W5. D4 reads `payload.MCPRegistration()` (the accessor from D1) rather than `payload.MCP` directly.
  - **AcceptanceCriteria:**
    - New TUI step added after group selection.
    - Default = YES (pressing Enter at the prompt accepts `.mcp.json` registration).
    - `initJSONPayload.MCP` (`*bool` from D1) reflects user's choice via pointer set in the confirm step.
    - `MCP = false` hard-wiring removed.
    - TUI step tests via `teatest_v2`.
    - CONSUMER-TIE supplement: three `run(--json)` paths (mcp:true, mcp:false, no-mcp-key) all pass.
    - `mage test-pkg ./cmd/till` green; `mage ci` green.
  - **ValidationPlan:** `mage test-pkg ./cmd/till`; `mage ci`.
  - **RiskNotes:**
    - `confirm.go` is new, not yet in tree (W5 ships it). Builder MUST run LSP `documentSymbol` on `internal/tui/components/confirm.go` after W5 completes to get the exact exported type name and API before writing D4's import.
    - W5 components must NOT return `tea.Quit`. Use `return nil` + `Done()`/`Cancelled()` accessors (per R10 W5-fals-FF2 decision).
    - `payload.MCPRegistration()` is the correct call site for reading MCP intent (defined in D1). Do NOT read `payload.MCP` directly — the `*bool` requires the accessor.
    - The `initTUIStepDone` terminal state assignment must now occur after the MCP confirm step (not after the group step).
  - **ContextBlocks:**
    - `constraint` (critical): D4 is `blocked_by 4c.6.1.W5`. Builder must NOT dispatch D4 until W5 is `complete`.
    - `constraint` (high): confirm component must NOT call `tea.Quit`. Use `Done()`/`Cancelled()` accessors + `return nil` per R10 W5-fals-FF2 decision.
    - `decision` (normal): default MCP = YES in TUI mode per dev directive (REVISION_BRIEF §2.6: "Default = YES per dev directive").
    - `decision` (normal): D4 reads `payload.MCPRegistration()` for all MCP-reading call sites — NOT `payload.MCP` directly.
    - `reference` (normal): REVISION_BRIEF §2.6; `cmd/till/init_cmd.go:runInitTUI` line ~236 (`MCP = false` hard-wire to remove); D1's `MCPRegistration() bool` accessor.
    - `warning` (high): `confirm.go` type name is new, not yet in tree. Verify via LSP after W5 ships.
  - **KindPayload:** `{"changes":[{"file":"cmd/till/init_cmd.go","symbol":"initTUIStep","action":"modify","shape_hint":"add initTUIStepMCP constant after group step"},{"file":"cmd/till/init_cmd.go","symbol":"initTUIModel","action":"modify","shape_hint":"add confirm.go component field; Update handles MCP step; remove MCP=false hardwire; set finalPayload.MCP = &mcpYes via confirm result"},{"file":"cmd/till/init_cmd_test.go","symbol":"TestInitTUIModel_MCPStep","action":"add","shape_hint":"new, not yet in tree; teatest_v2 drive; verify Enter=YES, n=NO, Esc=cancel"},{"file":"cmd/till/init_cmd_test.go","symbol":"TestRunInit_JSONMode_MCPPaths","action":"add","shape_hint":"CONSUMER-TIE supplement; run() with mcp:true, mcp:false, no-mcp-key (nil→true); new, not yet in tree"}]}`

---

### W2.D5 — copyAgentFiles REFACTOR TO SUBDIR-PER-GROUP

- **State:** todo
- **Kind:** `build` (atomic droplet; `Irreducible: true`)
- **Paths:**
  - `cmd/till/init_cmd.go` (MODIFY)
  - `cmd/till/init_cmd_test.go` (MODIFY — update copyAgentFiles tests for subdir-per-group)
- **Packages:** `cmd/till`
- **Blocked by:** W2.D4, 4c.6.1.W4.D1 (W4.D1's `git mv` makes the canonical embedded paths `builtin/agents/go/`, `builtin/agents/fe/`, `builtin/agents/gen/` — D5 reads from these unprefixed paths; without W4.D1 the paths don't exist and `fs.ReadDir` returns ENOENT)
- **Acceptance:**
  - `copyAgentFiles` signature changes from `(destDir, group string) (int, int, error)` to `(destDir string, groups []string) (int, int, error)` (or equivalent multi-group signature).
  - For each group in `groups`: copies embedded `agents/<group>/*.md` to `<destDir>/.tillsyn/agents/<group>/*.md` (subdir-per-group, NOT flat). Embed path: `builtin/agents/<group>/` (unprefixed — W4.D1's canonical names `go`/`fe`/`gen`, NOT `till-go`/`till-gen`).
  - Creates `<destDir>/.tillsyn/agents/<group>/` directory for each group.
  - Idempotent: existing files at `<destDir>/.tillsyn/agents/<group>/<name>.md` are SKIPPED (not overwritten).
  - FLAT detection guard (from D2) is preserved — it lives in `runInitPipeline`, NOT in `copyAgentFiles`. D5 must NOT add FLAT detection into `copyAgentFiles`.
  - `runInitPipeline` updated: calls `copyAgentFiles(destDir, payload.Groups)`.
  - Laslig summary row updated: `"groups"` key (comma-joined list) replaces `"group"` key.
  - CONSUMER-TIE: `run(ctx, args, &out, io.Discard)` end-to-end — single-group test: `--json '{"name":"x","groups":["go"],"mcp":false}'` verifies `<destDir>/.tillsyn/agents/go/<name>.md` created. Multi-group test: `--json '{"name":"x","groups":["go","fe"],"mcp":false}'` verifies both `agents/go/` and `agents/fe/` subdirs created.
  - `mage test-pkg ./cmd/till` passes; `mage ci` green.
- **Specify:**
  - **Objective:** Refactor `copyAgentFiles` from single-group flat copy (`<destDir>/.tillsyn/agents/<name>.md`) to multi-group subdir copy (`<destDir>/.tillsyn/agents/<group>/<name>.md`). This is the structural core of the W2 overhaul — after D5, the agent files land in the correct subdir-per-group layout. The embed source paths use W4.D1's canonical unprefixed names (`go`/`fe`/`gen`).
  - **AcceptanceCriteria:**
    - Subdir-per-group: `<destDir>/.tillsyn/agents/<group>/<name>.md` for each group.
    - Multi-group: both groups processed when `groups = ["go","fe"]`.
    - Idempotent skip for existing files.
    - Embed source: `builtin/agents/<group>/` (unprefixed — `go`/`fe`/`gen` per W4.D1).
    - FLAT detection guard from D2 is NOT in `copyAgentFiles` — it remains in `runInitPipeline`.
    - `added` count = total files created across all groups; `skipped` count = total skipped.
    - CONSUMER-TIE: single-group and multi-group `run()` end-to-end tests.
    - `mage test-pkg ./cmd/till` green; `mage ci` green.
  - **ValidationPlan:** `mage test-pkg ./cmd/till`; `mage ci`.
  - **RiskNotes:**
    - The embedded template FS path is `path.Join("builtin", "agents", group)` — group is the UNPREFIXED canonical name (`go`, `fe`, `gen`). W4.D1's `git mv` makes these paths exist. Builder verifies via `fs.ReadDir(templates.DefaultTemplateFS, "builtin/agents")` after W4.D1 ships — confirm `go/`, `fe/`, `gen/` subdirs exist (NOT `till-go/`, `till-gen/`).
    - W4.D1 is an explicit `blocked_by` for D5 — do NOT dispatch D5 until W4.D1 is `complete`.
    - The Laslig summary output changes — `"group"` key in `writeCLIKV` becomes `"groups"` (a comma-joined list). Builder updates the summary row in `runInitPipeline`.
  - **ContextBlocks:**
    - `constraint` (critical): FLAT detection is in `runInitPipeline`, NOT in `copyAgentFiles`. D5 must NOT add FLAT detection into `copyAgentFiles` — that would create a double-check redundancy and could conflict with D2's error message format.
    - `constraint` (high): idempotent skip applies per-file. A partially-initialized project (some groups present, some not) is a valid state — D5 must process all groups in the slice and skip files that exist.
    - `constraint` (critical): embed path uses UNPREFIXED group names (`go`/`fe`/`gen`). Builder must NOT use `till-go`/`till-gen` — those paths will return ENOENT after W4.D1's `git mv`.
    - `decision` (normal): `added` + `skipped` counts are aggregated across all groups (total, not per-group). Simpler Laslig summary.
    - `reference` (normal): REVISION_BRIEF §2.3 subdir-per-group shape; SKETCH §2.1; L1 PLAN.md line 259.
  - **KindPayload:** `{"changes":[{"file":"cmd/till/init_cmd.go","symbol":"copyAgentFiles","action":"modify","shape_hint":"signature: (destDir string, groups []string) (int, int, error); inner loop per group; embed path = builtin/agents/<group>/ (unprefixed); dest = .tillsyn/agents/<group>/<name>.md"},{"file":"cmd/till/init_cmd.go","symbol":"runInitPipeline","action":"modify","shape_hint":"call copyAgentFiles(destDir, payload.Groups); update Laslig summary groups key"},{"file":"cmd/till/init_cmd_test.go","symbol":"TestCopyAgentFiles_SubdirPerGroup + TestRunInitPipeline_MultiGroup","action":"modify","shape_hint":"update for subdir layout + unprefixed embed paths; add multi-group CONSUMER-TIE test"}]}`

---

### W2.D6 — template.toml WRITE (AGGREGATE FROM HOME OR EMBEDDED PER GROUP)

- **State:** todo
- **Kind:** `build` (atomic droplet; `Irreducible: true`)
- **Paths:**
  - `cmd/till/init_cmd.go` (MODIFY)
  - `cmd/till/init_cmd_test.go` (MODIFY — add template.toml write tests)
- **Packages:** `cmd/till`
- **Blocked by:** W2.D5
- **Acceptance:**
  - `runInitPipeline` calls a new `writeTemplateTOML(destDir string, groups []string, homeDir string) (int, int, error)` function (or equivalent) after `copyAgentFiles` succeeds.
  - For each group in `groups`: source is `filepath.Join(homeDir, ".tillsyn", "templates", group+".toml")` (HOME tier) if exists; else embedded `builtin/till-<group>.toml`.
  - Aggregated TOML content is written to `<destDir>/.tillsyn/template.toml`.
  - **Idempotency with partial-state warning:** If `<destDir>/.tillsyn/template.toml` already exists, the file is NOT overwritten (blanket skip). However: if the existing file is absent one or more `[<group>]` sections for the current selected groups, `runInitPipeline` prints a warning: `"WARN: <destDir>/.tillsyn/template.toml already exists but is missing sections for group(s): [<missing-list>]. Remove it and re-run to regenerate."` (non-fatal — exits zero, warning only). This addresses the partial-state scenario without migration code.
  - `homeDir` is derived from `os.UserHomeDir()` unless `rootOpts.homeDir` is non-empty (overrides for test isolation).
  - Laslig summary row added: `"template.toml"` → `"added"` or `"skipped (already exists)"`.
  - CONSUMER-TIE: `run()` end-to-end tests — (a) HOME tier present (mock homeDir), (b) HOME tier absent (falls back to embedded), (c) idempotent re-run (file exists, skipped, no error), (d) partial-state re-run (file exists, missing group section — verifies warning in output but zero exit).
  - `mage test-pkg ./cmd/till` passes; `mage ci` green.
- **Specify:**
  - **Objective:** After group files are copied, write `<destDir>/.tillsyn/template.toml` aggregated from the selected groups' templates. Source is HOME tier (`~/.tillsyn/templates/<group>.toml`) if present, else embedded binary default. Blanket skip if file exists; warn on partial-state (missing group sections). Closes REVISION_BRIEF §2.4.
  - **AcceptanceCriteria:**
    - `<destDir>/.tillsyn/template.toml` written after `copyAgentFiles` completes.
    - HOME tier sourced from `filepath.Join(homeDir, ".tillsyn", "templates", group+".toml")`.
    - Embedded fallback from `builtin/till-<group>.toml` (verify exact embed path via `fs.ReadDir` on `templates.DefaultTemplateFS` after W4.D2 ships).
    - Blanket skip if file exists. Warning if file exists AND missing group sections. Non-fatal.
    - `homeDir` override via `rootOpts.homeDir` for test isolation.
    - CONSUMER-TIE: four `run()` tests (HOME present, HOME absent, idempotent, partial-state warning).
    - `mage test-pkg ./cmd/till` green; `mage ci` green.
  - **ValidationPlan:** `mage test-pkg ./cmd/till`; `mage ci`.
  - **RiskNotes:**
    - `platform.Paths` has NO `TemplatesDir` field (confirmed by reading `internal/platform/paths.go`). Builder constructs the HOME templates path manually: `filepath.Join(homeDir, ".tillsyn", "templates")`. This is NOT routed through `platform.DefaultPathsWithOptions`. Refinement PLATFORM-TEMPLATES-R1 tracks adding `TemplatesDir` to `platform.Paths` in a later drop.
    - Partial-state warning detection: check if the existing `template.toml` content contains `"[<group>]"` or `"[<group>."` for each selected group. Simple string check — not full TOML parse. If present, group is considered covered; if absent, group is missing. Builder documents the heuristic in a doc-comment.
    - The embedded `builtin/till-<group>.toml` paths are confirmed by W4.D2 (which updates the TOML files). Builder verifies the exact embed path via `fs.ReadDir(templates.DefaultTemplateFS, "builtin")` after W4.D2 ships. Do NOT hard-code embed paths before W4.D2 completes.
    - `writeTemplateTOML` is new, not yet in tree.
  - **ContextBlocks:**
    - `constraint` (critical): blanket skip — do NOT overwrite existing `template.toml`. If users have customized it, `till init --re-run` must not destroy their work. Only warn on missing groups (non-fatal).
    - `decision` (normal): partial-state warning uses a simple string presence check (not full TOML parse) for the `[<group>]` section header. Simpler and consistent with "no migration, fail loud" philosophy — the warning gives users an explicit instruction to `rm` and re-run.
    - `decision` (normal): `platform.Paths.TemplatesDir` does not exist; construct HOME templates path directly from `homeDir`. Refinement PLATFORM-TEMPLATES-R1 tracks adding it.
    - `reference` (normal): REVISION_BRIEF §2.4; SKETCH §1.2; W1 container (HOME-tier path convention for TEMPLATE resolution); `internal/platform/paths.go` (confirmed no TemplatesDir field).
    - `warning` (high): embedded `till-<group>.toml` paths must be confirmed after W4.D2 ships (the schema changes in W4.D2 may change embed paths). Do not hard-code paths before W4.D2 completes.
  - **KindPayload:** `{"changes":[{"file":"cmd/till/init_cmd.go","symbol":"writeTemplateTOML","action":"add","shape_hint":"func(destDir string, groups []string, homeDir string) (int, int, error) — new, not yet in tree; HOME-then-embedded fallback per group; blanket skip if exists; warn on missing group sections"},{"file":"cmd/till/init_cmd.go","symbol":"runInitPipeline","action":"modify","shape_hint":"call writeTemplateTOML after copyAgentFiles; add template.toml row to Laslig summary"},{"file":"cmd/till/init_cmd_test.go","symbol":"TestWriteTemplateTOML + TestRunInitPipeline_TemplateTOML","action":"add","shape_hint":"new, not yet in tree; HOME-tier present + absent cases; idempotent re-run; partial-state warning case"}]}`

---

### W2.D7 — createProjectDBRecord UPGRADE TO CreateProjectWithMetadata

- **State:** todo
- **Kind:** `build` (atomic droplet; `Irreducible: true`)
- **Paths:**
  - `cmd/till/init_cmd.go` (MODIFY)
  - `cmd/till/init_cmd_test.go` (MODIFY — update project-DB-record tests)
- **Packages:** `cmd/till`
- **Blocked by:** W2.D6
- **Acceptance:**
  - `createProjectDBRecord` calls `svc.CreateProjectWithMetadata(ctx, app.CreateProjectInput{...})` instead of `svc.CreateProject(ctx, name, "")`.
  - Fields populated in `CreateProjectInput` (`internal/app/service.go:286`):
    - `Name = payload.Name`
    - `RepoPrimaryWorktree = cwd` (absolute path from `os.Getwd()`)
    - `RepoBareRoot` = result of `git rev-parse --git-common-dir` executed in `cwd`; if the command fails (not a git repo or bare-root not found), `RepoBareRoot = ""` (empty — not a fatal error).
    - `Language` = `payload.Groups[0]` mapped through language closed enum: `"go"` if first group is `"go"`, `"fe"` if first group is `"fe"`, `""` if first group is `"gen"` (or any unmapped value). Selection-order wins: user's first group pick determines primary language. This respects user intent over fixed priority.
    - `Metadata.Groups = payload.Groups` — typed field from W1.D2 (`internal/domain/project.go:ProjectMetadata.Groups []string`). Write directly. NO `KindPayload` JSON stopgap.
  - Bare-root detection: `exec.CommandContext(ctx, "git", "rev-parse", "--git-common-dir")` run in `cwd`; output is trimmed and resolved to absolute path via `filepath.Abs`. If relative (e.g. `.git`), resolve relative to `cwd`. If the command exits non-zero, `RepoBareRoot = ""`.
  - Idempotency unchanged: if a project with the same name already exists, skip creation and return `"already exists — skipped"`.
  - Laslig summary row for `"project DB"` unchanged format.
  - CONSUMER-TIE: `run(ctx, args, &out, io.Discard)` end-to-end — (a) new project in a git repo (verifies `RepoPrimaryWorktree` non-empty), (b) new project NOT in a git repo (verifies graceful empty `RepoBareRoot`), (c) re-run idempotent.
  - `mage test-pkg ./cmd/till` passes; `mage ci` green.
- **Specify:**
  - **Objective:** Upgrade `createProjectDBRecord` to populate the `RepoPrimaryWorktree`, `RepoBareRoot`, `Language`, and `Metadata.Groups` first-class project fields. Metadata.Groups uses the typed field shipped by W1.D2 — NOT a KindPayload JSON stopgap. This closes the dispatcher gap: `ErrInvalidSpawnInput` errors because `RepoPrimaryWorktree` is empty on projects created by old `till init`.
  - **AcceptanceCriteria:**
    - `RepoPrimaryWorktree` = `os.Getwd()` (absolute path, non-empty).
    - `RepoBareRoot` = git bare-root path if detectable; `""` otherwise. Not a fatal error.
    - `Language` = `payload.Groups[0]`-mapped value (go→"go", fe→"fe", gen→""). Selection-order wins.
    - `Metadata.Groups = payload.Groups` (typed `[]string` field from W1.D2 — verify via LSP `documentSymbol` on `internal/domain/project.go:ProjectMetadata` after W1.D2 ships; field name is `Groups`, JSON tag `groups,omitempty`).
    - NO `Metadata.KindPayload` JSON stopgap for groups. KindPayload left at its zero value.
    - Idempotent: re-run skips if name exists.
    - CONSUMER-TIE: three `run()` end-to-end tests (git-repo case, non-git case, idempotent re-run).
    - `mage test-pkg ./cmd/till` green; `mage ci` green.
  - **ValidationPlan:** `mage test-pkg ./cmd/till`; `mage ci`.
  - **RiskNotes:**
    - `exec.CommandContext(ctx, "git", "rev-parse", "--git-common-dir")` may not be available in CI (bare PATH). Builder uses `exec.LookPath("git")` first and sets `RepoBareRoot = ""` gracefully if git is absent.
    - `git rev-parse --git-common-dir` returns a RELATIVE path from the working directory (e.g. `.git` for a regular repo, or `../..` for a worktree). Builder resolves with `filepath.Abs(filepath.Join(cwd, output))`.
    - `Metadata.Groups` typed field: verify via LSP `documentSymbol` on `internal/domain/project.go:ProjectMetadata` after W1.D2 ships. The field was added by W1.D2 (`Groups []string` with JSON tag `groups,omitempty`). If W1.D2 has not shipped when D7 dispatches, this field will not exist — D7 MUST wait for W1 to be `complete` (already enforced by wave-level `W2 blocked_by W1`).
    - `Language` mapping: `payload.Groups[0]` is the source. For a `["gen"]` project, Language = `""` (correct — gen has no language bias). For `["gen", "go"]`, Language = `""` because gen is first. User intent: if user selected gen before go, gen is primary. The fixed go-priority heuristic was explicitly rejected per NIT5 absorption — selection-order is the policy. Builder documents this in a doc-comment.
    - Tests for `createProjectDBRecord` must mock the DB (following existing test patterns in `init_cmd_test.go`) — no real DB in unit tests.
  - **ContextBlocks:**
    - `constraint` (critical): Use `Metadata.Groups = payload.Groups` (typed field from W1.D2). Do NOT use `Metadata.KindPayload = {"groups":[...]}` stopgap. The typed field exists post-W1.D2 and is the correct consumer surface.
    - `constraint` (critical): `Metadata.Groups` typed field requires W1.D2 to have shipped. D7 is Wave C (W2 blocked_by W1). By D7 dispatch time, W1.D2 is complete. Verify via LSP before writing.
    - `decision` (normal): `Language` = `payload.Groups[0]` mapped value (selection-order wins). Documented as explicit policy: "user's first group selection determines primary language; fixed-priority heuristic rejected per plan-QA NIT5."
    - `decision` (normal): `RepoBareRoot` detection via `git rev-parse --git-common-dir`. Failure is non-fatal — empty string is the meaningful zero value per `domain/project.go:29`.
    - `reference` (normal): REVISION_BRIEF §2.5; `internal/app/service.go:286` (`CreateProjectInput` — the shape D7 constructs) wraps `internal/domain/project.go:212` (`ProjectInput` — internal validation shape called by `NewProjectFromInput`); `internal/domain/project.go:119` (`ProjectMetadata` — confirmed no Groups field TODAY; W1.D2 adds it).
    - `warning` (high): `Metadata.Groups` typed field does NOT exist in the current tree (confirmed by reading `internal/domain/project.go:119-155` — no Groups field today). W1.D2 ships it. D7 builder MUST verify field exists via LSP after W1.D2 is `complete` before writing `Metadata.Groups = payload.Groups`.
  - **KindPayload:** `{"changes":[{"file":"cmd/till/init_cmd.go","symbol":"createProjectDBRecord","action":"modify","shape_hint":"call CreateProjectWithMetadata with RepoPrimaryWorktree=cwd, RepoBareRoot=git-rev-parse, Language=Groups[0]-mapped, Metadata.Groups=payload.Groups (typed field from W1.D2 — NOT KindPayload)"},{"file":"cmd/till/init_cmd_test.go","symbol":"TestCreateProjectDBRecord_FullFields","action":"modify","shape_hint":"assert RepoPrimaryWorktree non-empty; assert Metadata.Groups = payload.Groups; assert Language mapping from Groups[0]; 3 CONSUMER-TIE run() cases"}]}`

---

## Blockers Reference

Mirrors the `blocked_by` declarations above. `PLAN.md` is truth.

| Droplet | Blocked by |
|---------|-----------|
| W2.D1 | 4c.6.1.W4.D1 |
| W2.D2 | W2.D1 |
| W2.D3 | W2.D2, 4c.6.1.W5 |
| W2.D4 | W2.D3, 4c.6.1.W5 |
| W2.D5 | W2.D4, 4c.6.1.W4.D1 |
| W2.D6 | W2.D5 |
| W2.D7 | W2.D6 |

---

## Raised Refinements (W2 scope)

| ID | Description | Status |
|----|-------------|--------|
| PLATFORM-TEMPLATES-R1 | Add `TemplatesDir string` to `internal/platform/paths.go:Paths` so D6's HOME-tier template path goes through `platform.DefaultPathsWithOptions` instead of direct construction. Future drop owns `internal/platform`. | OPEN |
| W2-GROUPS-R1 | Add typed `Groups []string` to `internal/domain/project.go:ProjectMetadata` and migrate from `KindPayload` JSON stopgap. | **RESOLVED** — R10-D2: W1.D2 ships `ProjectMetadata.Groups []string` typed field. D7 uses it directly. No stopgap. |

---

## Notes

### CONSUMER-TIE TEST CONTRACT

All tests in `init_cmd_test.go` invoke `run(ctx, args, &out, io.Discard)` end-to-end per the CONSUMER-TIE TEST CONTRACT from REVISION_BRIEF §2 + L1 PLAN.md NIT3. Unit assertions directly on internal helpers (`copyAgentFiles`, `validateInitPayload`, etc.) are acceptable as SUPPLEMENTS but not as the PRIMARY acceptance gate. The end-to-end `run()` call is the primary gate for each droplet's acceptance criteria.

### Build Discipline

- `mage test-pkg ./cmd/till` after each droplet — must be green before committing.
- `mage ci` as the final gate before D7 closes.
- Never `mage install`. Never `go test` directly.
- Single-line conventional commits ≤72 chars. No body.

### W5 Dependency Note

D3 and D4 are blocked by 4c.6.1.W5. The orchestrator MUST NOT dispatch D3 until W5's `complete` state is confirmed. After W5 ships, builder verifies `internal/tui/components/picker_multi.go` and `confirm.go` via LSP `documentSymbol` before authoring D3 and D4's imports.

**Dispatch-optimization note (Fals NIT1 disposition):** D3 only needs `picker_multi.go` (W5.D4) and D4 only needs `confirm.go` (W5.D2). Tightening to droplet-level external blockers (`W2.D3 blocked_by W5.D4`; `W2.D4 blocked_by W5.D2`) would save ~2-3 hours of unnecessary serialization. Pre-Drop-4b the dispatcher enforces at wave level, making this documentation-only. Kept at wave level for L2 simplicity. Future drop may tighten when the dispatcher supports droplet-level external-blocker granularity.

### W5 Component Contract (R10 Locked Decisions)

Per R10 (W5 fals FF1 + FF2 dispositions):
- W5 components are Bubble Tea sub-models composed by an outer `tea.Model`. They do NOT satisfy `tea.Model` directly (`View()` returns `tea.View`, not `string`). Builder must NOT write `var _ tea.Model = (*PickerMulti)(nil)` or similar.
- W5 components must NOT return `tea.Quit` (kills parent TUI). Use `return nil` + `Done()`/`Cancelled()` accessors instead.

### agents.toml Multi-Group Aggregation Gap (Proof NIT6 — DEFERRED)

REVISION_BRIEF §2.3 line 60: "Aggregate the group's bindings into `<project>/agents.toml` under `[<group>]` and `[<group>.<kind>]` sections." L2 D6 covers `template.toml` aggregation but NOT `agents.toml` multi-group aggregation. Today's `copyAgentsTOML` (in `runInitPipeline`) copies a single static embedded fixture `builtin/agents.example.toml` — single-group only.

This gap is faithful to the L1 contract (L1 acceptance bullets don't include `agents.toml` multi-group aggregation). Two options:
- **(a)** W4.D2 ships an embedded `agents.example.toml` containing `[go]` + `[fe]` + `[gen]` sections; `copyAgentsTOML` stays single-fixture. Add an acceptance bullet to W4.D2 confirming all three group sections exist in the fixture.
- **(b)** Add a D6b droplet to W2 for `agents.toml` multi-group aggregation from per-group sources.

**Routing:** This decision requires orchestrator adjudication. The planner recommends Option (a) as the simpler path — a well-structured single fixture with all groups is effectively equivalent to per-group aggregation at init time. If dev chooses Option (b), a new droplet must be added between D5 and D6 in the serial chain.
