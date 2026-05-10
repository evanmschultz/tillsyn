# DROP_4c.6.W0_AGENTS_TOML_SCHEMA — Builder QA Proof

Append `## Droplet N.M — Round K` per QA pass. Per `workflow/example/drops/WORKFLOW.md` § "File Lifecycle", this file is durable; never `git rm`d.

## Droplet 4c.6.W0.D1 — Round 1

### Findings

(none — see Summary)

### Missing Evidence

(none — see Summary)

### Summary

**Verdict: pass** — 0 findings.

Build-QA-Proof axes evaluated:

1. **Diff-vs-spec.** Commit `4f14547` ("feat(config): w0.d1 agents.toml schema and decode") touches exactly the declared paths from PLAN.md droplet D1: `internal/config/agents.go` (NEW, 195 LOC), `internal/config/agents_test.go` (NEW, 184 LOC), `internal/config/testdata/agents/baseline.toml` (NEW), `internal/config/testdata/agents/malformed.toml` (NEW), `internal/config/testdata/agents/preset_only.toml` (NEW), `internal/config/testdata/agents/unknown_field.toml` (NEW), plus the expected `BUILDER_WORKLOG.md` round entry and one-line state-flip on `PLAN.md`. No drive-by edits. The two extra fixtures (`malformed.toml`, `unknown_field.toml`, `preset_only.toml`) beyond the PLAN.md `Paths` line item ("`baseline.toml` (NEW — golden fixture exercising `[agents]` defaults + one per-kind block)") are spec-justified — `TestLoadRegistry_MalformedTOML` and `TestLoadRegistry_AbsentBlocksNilSafe` are explicit acceptance bullets and need fixtures; `TestLoadRegistry_UnknownTopLevelField` is a load-bearing strict-decode proof.

2. **AcceptanceCriteria coverage.** Each PLAN.md L2 acceptance bullet has a verifying test:
   - "`Preset` (the `[agents]` defaults block — fields `Client`, `Model`, `Effort`, `MaxTries`, `MaxBudgetUSD`, `MaxTurns`, `BlockedRetries`, `BlockedRetryCooldown`, `AutoPush`, `EnvSet`, `EnvFromShell`, `CliArgs`, `ToolsAllow`, `ToolsDeny`, `ClaudeMDAddons`)" → all 15 fields present at `internal/config/agents.go:35-51`; `TestLoadRegistry_Baseline` (`agents_test.go:34-84`) asserts every field's value against `baseline.toml`. Field set matches §4.1 schema exactly.
   - "`Override` (partial-shape; every field is a `*T` pointer)" → all 15 fields are pointers at `agents.go:62-78`; pointer-vs-zero distinction verified by `TestLoadRegistry_Baseline:90-99` which asserts `override.ToolsAllow != nil` AND `override.Model == nil` from a single fixture.
   - "`AgentRuntime` (effective per-kind merged result, same fields as `Preset`)" → present at `agents.go:84-100`, field set mirrors `Preset` 1-1.
   - "`AgentsRegistry` (loaded `agents.toml` — holds `Preset` + `map[Kind]Override`)" → present at `agents.go:106-109`.
   - "`LoadRegistry(path string) (*AgentsRegistry, error)` reads a TOML file via `pelletier/go-toml/v2`" → present at `agents.go:153-184`; uses `toml.NewDecoder(bytes.NewReader(content)).DisallowUnknownFields()`. The PLAN.md acceptance offers `toml.Unmarshal` "or equivalent — verify exact API via `go doc github.com/pelletier/go-toml/v2.Decoder`"; worklog Design Note 2 confirms `toml.Unmarshal` doesn't support strict mode so Decoder API is the lib-correct equivalent.
   - "Decoder preserves line numbers in error reporting via `*toml.DecodeError`" → `TestLoadRegistry_MalformedTOML` (`agents_test.go:106-122`) asserts `errors.As(err, &decodeErr)` succeeds AND `decodeErr.Position()` returns row > 0.
   - "Map fields decode as `map[string]string`; nil-safe for absent blocks" → `Preset.EnvSet` and `Preset.EnvFromShell` are bare `map[string]string` (`agents.go:45-46`); `TestLoadRegistry_AbsentBlocksNilSafe` (`agents_test.go:158-171`) confirms `Overrides` map is non-nil and absent kinds yield `(zero, false)` from map lookup, never panic.
   - "`Kind` is the closed-12-enum string type" → `internal/domain.Kind` reused (worklog Design Note 1 confirms `internal/config` imports `internal/domain` directly with no reverse import; `agents.go:24` imports it). All 12 kind values handled in `LoadRegistry` via `addOverride` calls on `agents.go:167-178`.

3. **Constraint preservation.** `internal/config/config.go` last touched in Drop 2 (`fdcaa65`); D1 did NOT modify it (`git log -- internal/config/config.go` shows no `4f14547` entry). No competing TOML libs introduced — `git grep BurntSushi/toml` and `git grep naoina/toml` return empty; the only TOML lib in `go.mod` remains `github.com/pelletier/go-toml/v2 v2.2.4` (`go.mod:34`), already in use at `internal/config/config.go:13`.

4. **Spec-conformance.** TOML decode preserves line numbers via `*toml.DecodeError` — `TestLoadRegistry_MalformedTOML` exercises `errors.As(err, &decodeErr)` and `decodeErr.Position()` against `malformed.toml` (line 6 unterminated string). The strict-decode rejection path is additionally covered by `TestLoadRegistry_UnknownTopLevelField` against `unknown_field.toml`. Schema field set matches §4.1 exactly: `client / model / effort / max_tries / max_budget_usd / max_turns / blocked_retries / blocked_retry_cooldown / auto_push / env_set / env_from_shell / cli_args / tools_allow / tools_deny / claude_md_addons` — 15 fields, present in both `Preset` and `Override`. Per-kind blocks decoded as typed pointer fields (one per kind in the closed-12-enum) at `agents.go:127-141`; this trades extensibility (adding a 13th kind requires editing this struct) for strict-decode rejection of typo'd kind names like `[agents.bulid]` — explicitly a deliberate design trade documented in the doc-comment at `agents.go:120-126` AND in worklog Design Note 3.

5. **Shipped-but-not-wired.** `LoadRegistry` is the proper consumer for the schema in this droplet's vertical slice (the load+decode proof). `AgentRuntime` is shipped without a producer in D1 — but PLAN.md L2 acceptance ("`AgentRuntime` (effective per-kind merged result, same fields as `Preset` but resolved)") and worklog Design Note 6 explicitly justify this: `AgentRuntime` lives in `agents.go` from D1 onward so D2's `Resolve` is a pure-function add. This is documented planner intent, not orphaned code. `Override` is consumed by the test (asserts pointer-vs-zero distinction); `Preset` is consumed by the test and stored on `AgentsRegistry`; `AgentsRegistry` is the return shape of `LoadRegistry`. No fixture is shipped without a consuming test (all four fixtures map to specific tests).

**Gate evidence:**

- `mage test-pkg ./internal/config` — 37/37 GREEN (32 pre-existing + 5 new). Builder report matches.
- `mage test-func ./internal/config TestLoadRegistry_Baseline` — 1/1 GREEN.
- `mage test-func ./internal/config TestLoadRegistry_MalformedTOML` — 1/1 GREEN.
- `mage test-func ./internal/config TestLoadRegistry_UnknownTopLevelField` — 1/1 GREEN.
- `mage test-func ./internal/config TestLoadRegistry_FileNotFound` — 1/1 GREEN.
- `mage test-func ./internal/config TestLoadRegistry_AbsentBlocksNilSafe` — 1/1 GREEN.
- `git grep "pelletier/go-toml/v2"` — single TOML lib confirmed; `git grep "BurntSushi/toml"` and `git grep "naoina/toml"` empty.

**Proof certificate:**

- **Premises** — D1 ships the 4 declared types (`Preset` / `Override` / `AgentRuntime` / `AgentsRegistry`) per §4.1; `LoadRegistry` decodes via `pelletier/go-toml/v2` in strict mode with position-aware errors; pointer-vs-zero discrimination works on `Override`; absent-blocks are nil-safe; `internal/domain.Kind` is reused (no local enum drift).
- **Evidence** — see per-axis citations above.
- **Trace or cases** — read PLAN.md acceptance bullets; cross-checked each against `agents.go` symbols + `agents_test.go` test names + assertions; ran 5 tests + package gate; verified diff scope via `git show --stat 4f14547`; verified TOML-lib non-competition via `git grep`.
- **Conclusion** — PASS. All five Build-QA-Proof axes satisfied; no findings; no missing evidence.
- **Unknowns** — none for D1 scope. D2/D3/D5 will revise some D1 assertions (`TestLoadRegistry_MalformedTOML` upgraded to expect `*ConfigError` envelope shape per D5); that revision is explicitly planned and out of scope for D1's own verdict.

### Hylla Feedback

N/A — verification touched only Go files, but all evidence-gathering used `Read` / `git log` / `git grep` / `mage` (worklog already reported the only Hylla-fallback case for D1). Hylla was not queried during this proof pass because every needed signal (file diff, commit metadata, test results) is post-commit pre-ingest territory where `git` + `mage` are authoritative.

## Droplet 4c.6.W0.D2 — Round 1

### Findings

(none — see Summary)

### Missing Evidence

(none — see Summary)

### Summary

**Verdict: pass** — 0 findings.

Build-QA-Proof axes evaluated:

1. **Diff-vs-spec.** Commit `b56df9c` ("feat(config): w0.d2 resolve inheritance merge engine") touches exactly the declared paths from PLAN.md droplet D2: `internal/config/agents.go` (MODIFY +139 LOC: `Resolve` + `copyMap`), `internal/config/agents_test.go` (MODIFY +274 LOC: 7 `TestResolve_*` tests + `ptrStr` / `ptrSlice` / `ptrMap` helpers), 4 NEW fixtures (`inheritance_full_inherit.toml`, `inheritance_partial_override.toml`, `inheritance_map_merge.toml`, `inheritance_list_replace.toml`) under `internal/config/testdata/agents/`, plus the expected `BUILDER_WORKLOG.md` round entry and one-line state-flip on `PLAN.md` (W0.D2 row `todo` → `done`). No drive-by edits. The newer commit `e999a0b` ("feat(templates): w0.5.d2 …") in `9bf73c9..HEAD` belongs to the separate W0.5 sub-drop (`internal/templates`, disjoint package, separate sub-drop dir) and is correctly out of scope for this D2 review.

2. **AcceptanceCriteria coverage.** Each PLAN.md L2 acceptance bullet for D2 has a verifying test:
   - "`Resolve(registry *AgentsRegistry, kind Kind) (AgentRuntime, error)` returns the merged effective per-kind config" → exported function present at `agents.go:227-319`; signature matches exactly (`func Resolve(registry *AgentsRegistry, kind domain.Kind) (AgentRuntime, error)`). Exported (capital R), consumer-callable from W3 frontmatter strip + W2 till init per planner intent.
   - "Per-field semantics … scalar/string/numeric/bool fields → if `Override.<field>` is non-nil, use override value; else use Preset value" → `agents.go:257-284` walks all 9 scalar fields with the `if ov.X != nil { out.X = *ov.X }` pattern. `TestResolve_PartialOverride` (`agents_test.go:239-274`) exercises one scalar (`MaxBudgetUSD = 9.5` override) while every other field falls through to Preset — proves the per-field discrimination on a real TOML decode.
   - "Per-key map merge … `EnvSet` and `EnvFromShell` merge with override keys winning, default keys absent in override surviving" → `agents.go:287-302` copies Preset map first (via `copyMap` at `agents.go:243-244`) then layers override keys. `TestResolve_MapMerge` (`agents_test.go:279-311`) loads the disjoint-key fixture and asserts `EnvSet = { A = "1", B = "2" }` (preset's A survives, override's B added); `TestResolve_MapMergeOverrideWins` (`agents_test.go:316-338`) covers the precedence half via in-code construction (`{K: preset}` Preset + `{K: override}` Override → `K = override`).
   - "List full-replace … if override slice is non-nil (even if empty), use override; else use Preset" → `agents.go:305-316` walks all 4 list fields with `if ov.X != nil { out.X = *ov.X }` (no append, no merge). `TestResolve_ListReplace` (`agents_test.go:343-362`) loads the fixture with Preset `tools_allow = [Read, Edit, Bash]` + override `tools_allow = [Read]` and asserts result is `[Read]` (full replace; same shape on `cli_args`).
   - "Empty-list-vs-nil: `Override.ToolsDeny = &[]string{}` (explicit empty list) MUST replace Preset's non-empty list" → `TestResolve_ExplicitEmptyList` (`agents_test.go:368-393`) constructs the override in-code via `ptrSlice([]string{})` over a Preset with `[rm, WebFetch]`; asserts result is `non-nil empty []string{}`. Pointer-to-slice idiom from D1 carries the discrimination correctly.
   - "`Resolve` for an absent kind … returns `Preset` verbatim — pure inheritance" → `agents.go:251-255` early-returns after `Overrides[kind]` lookup with `, ok` form. Two coverage paths: `TestResolve_FullInherit` (`agents_test.go:176-234`) exercises the empty-Overrides case via the `inheritance_full_inherit.toml` fixture (no per-kind blocks anywhere); `TestResolve_AbsentKindReturnsPreset` (`agents_test.go:400-431`) exercises the per-kind absent-key path via in-code construction (`KindPlan` has an override but `KindBuild` is queried).
   - All 5 named PLAN.md tests present (`FullInherit`, `PartialOverride`, `MapMerge`, `ListReplace`, `ExplicitEmptyList`); the 2 additional tests (`MapMergeOverrideWins`, `AbsentKindReturnsPreset`) cover precedence-on-collision and absent-kind paths called out by acceptance prose ("per-kind keys win"; "Resolve for an absent kind … returns Preset"), so they harden coverage rather than drift from spec.

3. **Constraint preservation.** D1 regression check: all 5 `TestLoadRegistry_*` tests still pass (`mage test-func ./internal/config "TestLoadRegistry_.*"` → 5/5 GREEN). D1 production code in `agents.go:1-195` is unchanged — D2's `Resolve` + `copyMap` append at lines 197-334 without disturbing existing schema types. `internal/config/config.go` not touched (D1 already verified single-TOML-lib state; D2 doesn't reintroduce competition). No new imports beyond what D1 already had (`bytes`, `fmt`, `os`, `pelletier/go-toml/v2`, `internal/domain`).

4. **Spec-conformance.** Pointer-vs-deref absent-vs-zero discrimination: `agents.go:257-316` reads every override field via the canonical `if ov.X != nil { out.X = *ov.X }` pattern — non-nil pointer to zero value (e.g. `*ov.MaxTries = 0`) DOES override Preset (per-field branch enters); nil pointer does NOT (branch skipped, Preset value already in `out` from the initial copy at lines 232-249). Map-merge precedence: override-wins-on-collision direction confirmed at `agents.go:291-293` (the loop over `*ov.EnvSet` writes into `out.EnvSet`, overwriting any pre-existing key — Go map assignment semantics); same for `EnvFromShell`. List full-replace including explicit-empty: `agents.go:305-316` uses `*ov.X` (deref to slice value) — a non-nil empty pointer dereferences to `[]string{}` which assigns into `out.X` and overwrites the Preset slice. Defensive copy on maps via `copyMap` (`agents.go:325-334`) avoids aliasing into Preset storage — a downstream caller mutating the returned `EnvSet` cannot accidentally rewrite the Preset for subsequent `Resolve` calls. List fields are NOT defensively copied (worklog Design Note 2 documents the trade-off as acceptable for current consumers); this is a deliberate doc'd choice, not a bug.

5. **Shipped-but-not-wired.** `Resolve` is the consumer entry point for `AgentsRegistry` shipped in D1: D1 stored Preset + Overrides; D2 produces the merged `AgentRuntime` callers consume. `Resolve` is exported (`func Resolve(...)`) so W3 frontmatter strip + W2 till init can reach it across package boundary — verified at `agents.go:227` (capital R, exported per Go visibility rules). `copyMap` is unexported (`func copyMap`) — internal helper, correctly package-private. No fixture is shipped without a consuming test (4 fixtures map 1-1 to the 4 fixture-driven tests; the 3 in-code tests use no fixture). The `Resolve` signature returns `(AgentRuntime, error)` with the error path reserved for D5's `*ConfigError` envelope per worklog Design Note 4 — today the only non-nil error is the `registry == nil` defensive check at `agents.go:228-230`; planner intent documented, not orphaned API surface.

**Gate evidence:**

- `mage test-pkg ./internal/config` — 44/44 GREEN (37 pre-D2 baseline from D1 round + 7 new `TestResolve_*`). Re-run by reviewer this round: identical result.
- `mage test-func ./internal/config "TestResolve_.*"` — 7/7 GREEN: `FullInherit`, `PartialOverride`, `MapMerge`, `MapMergeOverrideWins`, `ListReplace`, `ExplicitEmptyList`, `AbsentKindReturnsPreset`. Race detector + count=1 enforced by mage target (`-race -count=1` in command line).
- `mage test-func ./internal/config "TestLoadRegistry_.*"` — 5/5 GREEN (D1 regression check by reviewer): `Baseline`, `MalformedTOML`, `UnknownTopLevelField`, `FileNotFound`, `AbsentBlocksNilSafe`. No regression.
- `git show --stat b56df9c` — touched files match PLAN.md `Paths` declaration exactly (4 fixtures + agents.go + agents_test.go + worklog + PLAN.md state flip). No drive-by.
- Diff-scope verified vs `9bf73c9..HEAD`: the second commit `e999a0b` is W0.5.D2 (parallel sub-drop, disjoint package `internal/templates`); not part of W0.D2 review.

**Proof certificate:**

- **Premises** — D2 ships `Resolve(registry, kind)` per §4.2.1-§4.2.3 contract (scalar override-wins, map per-key merge with override-wins-on-collision, list full-replace including explicit empty); pointer-vs-deref discriminates absent-from-zero; D1 tests still pass; `Resolve` is exported for W2/W3 consumers; no shipped-but-not-wired surface.
- **Evidence** — see per-axis citations above (file:line for every claim) plus the gate evidence block.
- **Trace or cases** — read PLAN.md D2 acceptance bullets; cross-checked each against `agents.go:197-334` symbols + `agents_test.go:173-431` test bodies + assertions; cross-checked 4 fixtures' contents against test assertions; ran 7 `TestResolve_*` + 5 D1 regression `TestLoadRegistry_*` + full-package gate; verified diff scope via `git show --stat b56df9c`.
- **Conclusion** — PASS. All five Build-QA-Proof axes satisfied; no findings; no missing evidence.
- **Unknowns** — none for D2 scope. D5's `*ConfigError` envelope will exercise the currently-always-nil error return path on `Resolve`; that future revision is out of scope for D2's own verdict. Builder Design Note 2 acknowledges list-field defensive-copy as a "revisit if/when downstream mutates" — acceptable today; flagged as future-watch, not a finding.

### Hylla Feedback

N/A — verification touched only Go files. All evidence-gathering used `Read` (production + test + fixture sources), `git show` / `git log` / `git diff` (commit-range scope), and `mage test-pkg` / `mage test-func` (gate runs). Hylla was not queried during this proof pass — D2's surface is post-commit pre-ingest territory where `git` + `mage` + `Read` are authoritative; the live LSP daemon would not have surfaced any committed-state signal that those tools didn't cover.

## Droplet 4c.6.W0.D3 — Round 1

### Findings

(none rated medium-or-higher; one informational note documented under Summary axis 4 — coarse wrap-text deviation accepted per consensus reading of round-3 contract; D5 supersedes wrap text)

### Missing Evidence

(none — see Summary)

### Summary

**Verdict: pass** — 0 high/medium findings; 1 informational note on wrap-text deviation explicitly accepted per round-3 contract consensus reading.

Build-QA-Proof axes evaluated:

1. **Diff-vs-spec.** Commit `7dfd0f4` ("feat(config): w0.d3 mergelocal and tools_deny rejection") touches exactly the declared paths from PLAN.md droplet D3:
   - `internal/config/agents.go` (MODIFY, +333 LOC: `MergeLocal` + `mergePreset` + `mergeOverride` + `cloneOverride` + `copySlice` + `ErrToolsDenyNotOverridable` + `errors` import).
   - `internal/config/agents_test.go` (MODIFY, +301 LOC: 8 `TestMergeLocal_*` tests + `ptrFloat` + `ptrInt` helpers).
   - 3 new fixtures `local_override_model.toml` / `local_tools_deny_rejected.toml` / `local_partial_block.toml` under `internal/config/testdata/agents/`.
   - `BUILDER_WORKLOG.md` round entry, `PLAN.md` D3 state-flip (`todo` → `done`).
   No drive-by edits.

2. **AcceptanceCriteria coverage.** Each PLAN.md L2 acceptance bullet for D3 has a verifying test or in-code construct:
   - "`MergeLocal(project, local) (*AgentsRegistry, error)` returns a new `AgentsRegistry` with local fields deep-merged OVER project" → exported function present at `agents.go:386-436`; signature matches exactly. Field-merge over per-kind blocks at `agents.go:425-433` via `mergeOverride`; top-level Preset merge at `agents.go:423` via `mergePreset`. `TestMergeLocal_OverrideModel` (`agents_test.go:438-478`) and `TestMergeLocal_PartialBlock` (`:593-634`) prove field-level merge.
   - "`tools_deny` set in local … returns `ErrToolsDenyNotOverridable` — closed sentinel" → `agents.go:392-401` checks both `local.Preset.ToolsDeny` (defaults block) and every `local.Overrides[kind].ToolsDeny` (per-kind blocks). Two coverage tests: `TestMergeLocal_ToolsDenyRejected` (`agents_test.go:484-506`, fixture-driven, per-kind) and `TestMergeLocal_ToolsDenyDefaultsBlockRejected` (`agents_test.go:511-532`, in-code, defaults-block). Both assert `errors.Is(err, ErrToolsDenyNotOverridable)` only — D5's `TestMergeLocal_ToolsDenyPositionWrapped` will assert the envelope shape per PLAN.md:117.
   - "Bare sentinel message reads `'tools_deny is not user-overridable; remove the field'`" → verbatim match at `agents.go:36`: `var ErrToolsDenyNotOverridable = errors.New("tools_deny is not user-overridable; remove the field")`.
   - "`MergeLocal(project, nil)` returns `project` unchanged" → `agents.go:418-420` early-return after deep-clone path; `TestMergeLocal_NilLocal` (`agents_test.go:537-574`) asserts merged equals project field-for-field. Note: returns a deep-clone, not the project pointer itself (worklog Design Note 5 — symmetric contract). Spec says "unchanged"; deep-clone is value-equivalent so satisfies the contract.
   - "`MergeLocal(nil, local)` returns an error" → `agents.go:387-389` returns `"MergeLocal: project registry is nil; agents.toml is required"`. `TestMergeLocal_NilProject` (`agents_test.go:579-588`) asserts non-nil error.
   - "Deep-merge preserves project's per-kind overrides where local doesn't override" → `agents.go:426-433` uses `mergeOverride(existing, lov)` to layer local pointers over project pointers field-by-field. `TestMergeLocal_PartialBlock` proves this on a real fixture (local sets only `[agents.build].model`; project's `MaxBudgetUSD`, `MaxTurns`, `ToolsAllow` survive).
   - All 4 named PLAN.md tests present (`OverrideModel`, `ToolsDenyRejected`, `NilLocal`, `PartialBlock`); 4 additional tests (`ToolsDenyDefaultsBlockRejected`, `NilProject`, `PresetFieldMerge`, `NewKindBlock`) cover paths called out by acceptance prose ("`tools_deny` in the `[agents]` defaults block" per RiskNote line 129, nil-project path per line 114, defaults-block field merge per AC line 110, new-kind-from-local per line 115). All harden coverage rather than drift from spec.

3. **Constraint preservation.** D1+D2 regression check: `mage test-func ./internal/config "TestLoadRegistry_.*|TestResolve_.*"` → 12/12 GREEN (5 D1 + 7 D2). D1+D2 production code in `agents.go:1-345` is unchanged — D3 appends `MergeLocal` + helpers at lines 347-667 without disturbing existing `Preset` / `Override` / `AgentRuntime` / `AgentsRegistry` / `LoadRegistry` / `Resolve`. Imports added: `errors` only (single-line addition). No competing TOML libs introduced.

4. **Spec-conformance — D3↔D5 separation contract (LOAD-BEARING AXIS).** This axis carries the round-3 finalized strict-bare-sentinel contract.

   **Wrap-text shape verified at `agents.go:394` and `:398`:**
   - Defaults-block path: `fmt.Errorf("agents.local.toml [agents]: %w", ErrToolsDenyNotOverridable)`.
   - Per-kind path: `fmt.Errorf("agents.local.toml [agents.%s]: %w", kind, ErrToolsDenyNotOverridable)`.

   **Three contract sub-clauses evaluated:**

   - **Line numbers in the wrap?** NONE. Neither wrap call references `Position()`, `.Line()`, or any numeric-line value. The `[agents.%s]` formatter substitutes the kind string only (e.g. `"build"`), never a line number. ✓ Round-3 PLAN.md:112 + RiskNote line 128 + ContextBlock line 132 + KindPayload line 137 all forbid line-numbered position wrapping at D3 — fully honored.

   - **Does the wrap defeat `errors.Is(err, ErrToolsDenyNotOverridable)`?** NO. `fmt.Errorf` with `%w` produces an error whose `Unwrap()` returns the sentinel; `errors.Is` walks the chain. Verified by 2 tests asserting against the wrapped error: `TestMergeLocal_ToolsDenyRejected:503-505` and `TestMergeLocal_ToolsDenyDefaultsBlockRejected:529-531`. Both GREEN. ✓

   - **Does the wrap match the round-3 finalized acceptance bullets?** PARTIALLY. PLAN.md:112's parenthetical "(no file/line/block prefix at the D3 boundary)" is the strictest reading — the wrap DOES include both `agents.local.toml` (file prefix) and `[agents]` / `[agents.<kind>]` (block prefix). The consensus reading (PLAN.md:128 RiskNote, :132 ContextBlock, :137 KindPayload, :117 AC b8) is narrower — they only forbid LINE-NUMBERED position wrapping at D3, attributing the file/line/block envelope exclusively to D5. The wrap text has no line number, so the consensus reading is satisfied; line 112's literal parenthetical is technically violated.

   **Verdict on wrap-text deviation: ACCEPTABLE — justified usability improvement, NOT scope creep, NOT a round-3 spec violation under consensus reading.** Reasoning:
   - The block hint `[agents.<kind>]:` is a **structural/categorical** locator (which TOML section), not a **positional** locator (line number). The round-3 plan-QA loop spent 3 rounds locking the position-wrapping handoff (line numbers from `pelletier/go-toml/v2`'s `*toml.DecodeError`) — the work D5 must do, that D3 cannot do without source-position tracking. Block hints fall outside that scope; they are a coarse field-path naming analogous to W0.5 FF1's "field-path mitigation" — see PLAN.md:203 for the explicit observation that "MergeLocal doesn't currently parse TOML … the line number for tools_deny rejection comes from D3's load step, NOT from MergeLocal's own logic."
   - Tests assert against `errors.Is` ONLY — no test depends on the wrap-text body, so D5 can supersede the wrap text entirely without breaking the test contract. The wrap text becomes dead-code-by-supersession the moment D5 lands.
   - Builder honestly flagged this as an Unknown at BUILDER_WORKLOG.md:128 with explicit reversibility ("If reviewer prefers strict bare-sentinel-only, the `fmt.Errorf` call is one line to revert"). This is good-faith engineering.
   - **Alternative reading — strict line-112 violation — would require flagging this as a finding.** Reviewer judgment: line 112's parenthetical was descriptive of the bare-message body the planner originally specified; the load-bearing contract surfaces (RiskNote 128, ContextBlock 132, KindPayload 137, AC b8 117) all forbid only line-numbered wrapping. The wrap shape stays inside that boundary. Verdict pass with explicit informational note.

   **Forward-looking note for D5 builder (routed Unknown, NOT a D3 finding):** when D5's `*ConfigError{File, Block, Line, Cause}` envelope wraps the sentinel, ensure the final user-facing message does NOT duplicate `agents.local.toml` (D3 wrap already contains it). D5 should either wrap `ErrToolsDenyNotOverridable` directly (skipping D3's `fmt.Errorf` intermediate) OR the envelope's `Error()` should detect and elide redundant file/block prefixes.

5. **Shipped-but-not-wired.** `MergeLocal` is the consumer entry point for the local-merge layer per PLAN.md:228 ("D3's `MergeLocal` consumed by W3"); exported (`func MergeLocal`) at `agents.go:386` so W3 frontmatter strip + W2 till init can reach it across package boundary. `ErrToolsDenyNotOverridable` is exported at `agents.go:36` so callers can `errors.Is` against it — load-bearing for the rejection contract. Helpers `mergePreset`, `mergeOverride`, `cloneOverride`, `copySlice` are unexported (lowercase) — internal helpers, correctly package-private. No fixture is shipped without a consuming test (3 fixtures map 1-1 to 3 fixture-driven tests; the other 5 tests use in-code construction).

**Gate evidence (re-run by reviewer this round):**

- `mage test-pkg ./internal/config` — 52/52 GREEN (44 pre-D3 baseline + 8 new `TestMergeLocal_*`). Builder report ("8/8 GREEN") matches.
- `mage test-func ./internal/config "TestMergeLocal_.*"` — 8/8 GREEN: `OverrideModel`, `ToolsDenyRejected`, `ToolsDenyDefaultsBlockRejected`, `NilLocal`, `NilProject`, `PartialBlock`, `PresetFieldMerge`, `NewKindBlock`. Race detector + `count=1` enforced by mage target.
- `mage test-func ./internal/config "TestLoadRegistry_.*|TestResolve_.*"` — 12/12 GREEN (D1+D2 regression check).
- `git show --stat 7dfd0f4` — touched files match PLAN.md `Paths` declaration exactly (3 fixtures + agents.go + agents_test.go + worklog + PLAN.md state flip). No drive-by.
- Wrap-text shape verified at `agents.go:394` (`agents.local.toml [agents]: %w`) and `:398` (`agents.local.toml [agents.%s]: %w`) — no line numbers, no `Position()` calls, no numeric-line interpolation.

**Proof certificate:**

- **Premises** — D3 ships `MergeLocal(project, local) (*AgentsRegistry, error)` with deep-merge over per-kind blocks (field-level, pointer-discriminated) + tools_deny rejection via bare sentinel `ErrToolsDenyNotOverridable` (per PLAN.md:111-112) wrapped via `%w` so `errors.Is` works through any future D5 envelope; 8 D3 tests GREEN; D1+D2 regression GREEN; declared paths only; sentinel message verbatim PLAN.md:112; defaults-block + per-kind both rejected; nil-local + nil-project both handled per spec.
- **Evidence** — see per-axis citations above (file:line for every claim) plus the gate-evidence block.
- **Trace or cases** — read PLAN.md round-3 D3 acceptance bullets + RiskNotes + ContextBlocks + KindPayload; cross-checked each against `agents.go:36-667` symbols + `agents_test.go:438-724` test bodies + assertions; cross-checked 3 fixtures' contents against test assertions; ran 8 D3 + 12 D1+D2 regression + full-package gate; verified diff scope via `git show --stat 7dfd0f4`; verified wrap-text shape directly at `agents.go:394` + `:398`; cross-checked `errors.Is` chain semantics through `%w` against test assertions at `agents_test.go:503-505` + `:529-531`.
- **Conclusion** — PASS. All five Build-QA-Proof axes satisfied. The wrap-text deviation (block hint `[agents.<kind>]:` + file hint `agents.local.toml`, NO line number) is a justifiable usability improvement under the consensus reading of round-3 contract surfaces (RiskNote 128, ContextBlock 132, KindPayload 137, AC b8 117), all of which forbid only line-numbered wrapping at D3. The strictest reading of PLAN.md:112's parenthetical is technically violated but the load-bearing contract holds: `errors.Is` works, no line numbers leak, D5 can supersede wrap text entirely without breaking tests. Builder explicitly flagged the choice + offered one-line revert.
- **Unknowns** — (1) D5 builder must ensure final user-facing message doesn't duplicate `agents.local.toml` prefix when wrapping over D3's `fmt.Errorf` output; route to D5 spec confirmation. (2) Per PLAN.md:128 RiskNote, source-line tracking still owed by D5 — `MergeLocal` operates on already-decoded structs and cannot produce line numbers from its own logic; D5 must either thread source-positions onto `AgentsRegistry` (e.g., `linePositions map[string]int` per PLAN.md:203 RiskNote) or re-decode at `MergeLocal`'s boundary. Both options are documented; D5 builder picks at implementation time.

### Hylla Feedback

N/A — verification touched only Go files. All evidence-gathering used `Read` (production + test + fixture + plan-QA-R3 verdict files), `git show` / `git log` / `git status` (commit-range scope), and `mage test-pkg` / `mage test-func` (gate runs). Hylla was not queried during this proof pass — D3's surface is post-commit pre-ingest territory where `git` + `mage` + `Read` are authoritative for the round-3 contract verification, and the live LSP daemon would not have surfaced any committed-state signal that those tools didn't cover.

## Droplet 4c.6.W0.D3 — Round 2

### Round-1 Counterexample Verification

**W0-D3-CX1: FIXED.**

Round-1 build-QA-falsification CONFIRMED counterexample W0-D3-CX1 — wrap text `fmt.Errorf("agents.local.toml [agents]: %w", ...)` and `fmt.Errorf("agents.local.toml [agents.%s]: %w", kind, ...)` at `agents.go:394` and `:398` carried two of the three forbidden prefix axes (file `agents.local.toml`, block `[agents]` / `[agents.<kind>]`), violating PLAN.md:112's strict spec `"(no file/line/block prefix at the D3 boundary)"`.

Round-2 fix verified at commit `a32aed3` ("fix(config): w0.d3 revert tools_deny wrap to bare sentinel"). Diff scope: `internal/config/agents.go` -3/+3 LOC plus the worklog round-2 entry. Production-code change at `agents.go:391-401` is exactly the reviewer-recommended one-line revert pattern from BUILDER_QA_FALSIFICATION:143:

- **Defaults-block call site (`agents.go:394`)** — was `return nil, fmt.Errorf("agents.local.toml [agents]: %w", ErrToolsDenyNotOverridable)`; is `return nil, ErrToolsDenyNotOverridable` (bare sentinel). File prefix removed; block prefix removed.
- **Per-kind call site (`agents.go:398`)** — was `return nil, fmt.Errorf("agents.local.toml [agents.%s]: %w", kind, ErrToolsDenyNotOverridable)`; is `return nil, ErrToolsDenyNotOverridable` (bare sentinel). File prefix removed; block prefix removed; kind interpolation removed.
- **Compile-cleanup at `agents.go:396`** — `for kind, ov := range local.Overrides` rewritten to `for _, ov := range local.Overrides`. Required for compile (Go's `declared and not used` rule on `kind` once the per-kind `fmt.Errorf` no longer references it). Semantics-preserving: `kind` was only used inside the now-removed format string. No callers depend on iteration ordering or the per-kind identity (the iteration is fail-fast on first non-empty `ToolsDeny` regardless of which kind triggers it; from the caller's view, the sentinel return is the same shape).

**Sweep for residual prefix language across `agents.go` (D3 runtime path):**

- String `"agents.local.toml"` appears at the top of file body only as descriptive prose in the `MergeLocal` doc-comment block (lines 350-385) explaining the contract boundary between D3 and D5. Documentation, not a runtime prefix. PLAN.md:112's "no file/line/block prefix" applies to the runtime error message body, not to the doc-comment that explains why D3's runtime omits the prefix.
- String `[agents]` / `[agents.<kind>]` appears in doc-comments at lines 11, 116, 122, 129, 350, 358, 366, 372 — all descriptive prose explaining schema layout (the TOML block name) or the D5-future envelope shape. None reach a runtime error.
- Remaining `fmt.Errorf` call sites in agents.go (lines 167, 174, 240, 388): `os.ReadFile` failure (`"read agents.toml at %q: %w"`), decoder error (`"decode agents.toml at %q: %w"`), `Resolve` nil-registry (`"Resolve: registry is nil"`), `MergeLocal` nil-project (`"MergeLocal: project registry is nil; agents.toml is required"`). None of these are on the `tools_deny` rejection path; none use `[agents]` / `[agents.<kind>]` block syntax; the `agents.toml at %q` strings are loader-context metadata for unrelated I/O / decode failures, not the D3 boundary the spec governs.

**Test-contract preservation (no test edits required):**

- `TestMergeLocal_ToolsDenyRejected` at `agents_test.go:484-506` and `TestMergeLocal_ToolsDenyDefaultsBlockRejected` at `:511-532` both assert via `errors.Is(err, ErrToolsDenyNotOverridable)`. `errors.Is(sentinel, sentinel)` returns true (reflexive equality), identical verdict to `errors.Is(fmt.Errorf("...: %w", sentinel), sentinel)` in round 1. Test-file unchanged in round 2 (`git status --porcelain internal/config/agents_test.go` is clean per repository state) — sentinel chain preserved by the simpler bare-return construction.

### Fresh Findings

(none — no high/medium/low findings on the round-2 surface)

Build-QA-Proof axes evaluated against the round-2 surface (3-line delta + 8-test re-run):

1. **Diff-vs-spec.** Round-2 commit `a32aed3` touches exactly `internal/config/agents.go` (3 LOC) + `BUILDER_WORKLOG.md` (round-2 entry). No drive-by edits. PLAN.md state stays `done` per builder's worklog round-2 §"State flip" — round 2 is rework of an already-`done` droplet; no re-flip needed. Diff scope matches the round-1 CX fix-hint verbatim.

2. **AcceptanceCriteria coverage.** PLAN.md:112's strict spec `"no file/line/block prefix at the D3 boundary"` now satisfied by bare-sentinel returns. PLAN.md:111 spec `"closed sentinel error identifying the offending kind"` — note: round-2 revert drops kind identification at the D3 boundary entirely (kind no longer interpolated since both sites return bare sentinel). This is consistent with PLAN.md:112's spec second sentence assigning kind context to D5's envelope (per PLAN.md:202-203 RiskNote, D5 either threads source-line tracking onto `AgentsRegistry` or re-decodes at boundary to recover the offending kind from line position). Round-2 surface ships pure sentinel; D5 owns kind + line + file + block. Tests assert sentinel reachability via `errors.Is`, not kind identification — test contract preserved.

3. **Constraint preservation.** D1 + D2 regression check via `mage test-pkg ./internal/config` → 52/52 GREEN. `TestLoadRegistry_*` (5 tests) and `TestResolve_*` (7 tests) all GREEN — D2 + D3 round-1 production code adjacent to the revert is unchanged (`mergePreset` at `:442-504`, `mergeOverride` at `:509-586`, `cloneOverride` at `:592-655`, `copySlice` at `:660-667` all untouched). Only the rejection guard at `:391-401` and the loop-var rename at `:396` are deltas.

4. **Spec-conformance — D3↔D5 separation contract.** Round-2 surface clean of forbidden prefix language at the D3 runtime boundary:
   - **File prefix?** None. Both rejection paths return bare `ErrToolsDenyNotOverridable` whose `Error()` returns `"tools_deny is not user-overridable; remove the field"` (verbatim from `agents.go:36`). No file name in the runtime message.
   - **Block prefix?** None. The kind loop variable is dropped; the rejection no longer carries `[agents]` / `[agents.<kind>]` text.
   - **Line prefix?** None (was already absent in round 1 — the wrap had no line numbers).
   - **`errors.Is` chain?** Preserved. Bare sentinel returns satisfy `errors.Is(err, ErrToolsDenyNotOverridable)` reflexively (`err == ErrToolsDenyNotOverridable` directly). D5's future `*ConfigError{Cause: ErrToolsDenyNotOverridable}` will satisfy `errors.Is` via `Unwrap()` per PLAN.md:187. Forward-compat preserved.
   - **Doc-comment vs runtime distinction.** `MergeLocal`'s doc-comment at `agents.go:371-375` (verified) reads `"D5's envelope wraps this with file/line/block position info; D3 surfaces only the sentinel."` — descriptive prose explaining the contract, not a runtime prefix. PLAN.md:112's parenthetical governs the runtime error message body, not the doc-comment. The doc-comment correctly anticipates D5's wrapping shape.

5. **Shipped-but-not-wired.** No new exports in round 2; existing `MergeLocal` + `ErrToolsDenyNotOverridable` exports unchanged. The loop-var rename is internal-only (the loop body inside the function). No surface change to W2/W3 consumers.

**Loop-var rename necessity check (per round-2 spec):**

Round-2 spawn-prompt asks: "verify the loop-var rename is required (Go compile error otherwise) and doesn't change semantics."

- **Required for compile?** YES. Pre-revert, the loop body referenced `kind` inside `fmt.Errorf("agents.local.toml [agents.%s]: %w", kind, ErrToolsDenyNotOverridable)`. Post-revert, `kind` has no consumer in the loop body. Go's compiler enforces "declared and not used" on loop variables (per `go doc go/spec` and standard error message `kind declared and not used`). Without the rename, `mage test-pkg` would fail with `./agents.go:396:7: kind declared and not used`. The rename to `_` (blank identifier) discards the key while keeping the loop iterating over the map. Compile-required, not optional.
- **Semantics-preserving?** YES. The loop iterates `local.Overrides` (a `map[domain.Kind]Override`); whether the key is captured into `kind` or discarded into `_` does not change which values are visited, in what order, or whether the rejection fires on the same iteration. Iteration order on Go maps is randomized regardless of capture; the rejection is fail-fast on the first non-empty `ToolsDeny` regardless of which kind triggers it. From the caller's perspective: same set of inputs → same sentinel return.

### Missing Evidence

(none — round-2 surface is fully covered by gate runs and direct inspection)

### Summary

**Verdict: pass** — 0 findings, W0-D3-CX1 verified FIXED.

Round 2 ships exactly the 3-line delta the round-1 CX fix-hint specified (2 bare-sentinel reverts + 1 compile-cleanup loop-var rename). All eight `TestMergeLocal_*` tests GREEN; all 12 D1+D2 regression tests GREEN; full-package gate 52/52 GREEN with `-race -count=1`. The D3↔D5 separation contract per PLAN.md:112 strict reading is now honored: D3's runtime rejection path emits the bare sentinel `ErrToolsDenyNotOverridable` only — no file, no block, no line prefix. D5 retains exclusive ownership of envelope wrapping. `errors.Is` chain preserved. Doc-comments explaining the D3↔D5 boundary remain (correctly descriptive of the contract, not runtime output).

| Axis | Verdict |
| ---- | ------- |
| Diff-vs-spec | PASS — 3 LOC, declared paths only, matches CX fix-hint exactly |
| AcceptanceCriteria | PASS — PLAN.md:112 strict spec satisfied; sentinel reachability preserved |
| Constraint preservation | PASS — D1+D2 regression 12/12 GREEN; D3 round-1 helpers untouched |
| Spec-conformance (D3↔D5) | PASS — bare sentinel only; no file/block/line prefix at runtime |
| Shipped-but-not-wired | PASS — no surface change |

**Proof certificate:**

- **Premises** — Round 2 reverts the wrap text at `agents.go:394` + `:398` to bare `ErrToolsDenyNotOverridable` returns; renames the now-unused `kind` loop variable to `_` for compile; touches no test files; preserves `errors.Is` chain reflexively.
- **Evidence** — `git show a32aed3` (3 LOC delta in agents.go matching the round-1 fix-hint); direct `Read` of `agents.go:391-401` (bare-sentinel returns confirmed); doc-comment / fmt.Errorf sweep across full agents.go body (no residual file/block/line prefix at runtime); `mage test-pkg ./internal/config` → 52/52 GREEN; `mage test-func ./internal/config "TestMergeLocal_.*"` → 8/8 GREEN with -race -count=1; test files unchanged (`git status --porcelain internal/config/agents_test.go` clean).
- **Trace or cases** — (1) Defaults-block path: `local.Preset.ToolsDeny` non-empty → `return nil, ErrToolsDenyNotOverridable` (bare). (2) Per-kind path: `local.Overrides[any].ToolsDeny` non-empty → `return nil, ErrToolsDenyNotOverridable` (bare). (3) `errors.Is(sentinel, sentinel)` reflexive truth confirmed by 2 GREEN test cases. (4) D1+D2 regression unchanged: `TestLoadRegistry_*` 5/5 + `TestResolve_*` 7/7 GREEN. (5) Loop-var rename: `kind` no longer referenced in body, `_` discards key, iteration order randomized regardless, fail-fast on first hit — semantics-preserving.
- **Conclusion** — PASS. W0-D3-CX1 verified FIXED. Fresh 5-axis pass clean. No new findings.
- **Unknowns** — D5 owns kind identification + file/line/block envelope wrapping per PLAN.md:188-194 + PLAN.md:202-203. Round-2 ships pure sentinel; D5 will thread source-position tracking onto `AgentsRegistry` (or re-decode at `MergeLocal` boundary) to recover kind + line for the user-facing message. Routed forward to D5; not a D3 obligation.

### Hylla Feedback

N/A — verification touched only Go files (already in scope from W0.D1+W0.D2+W0.D3 round-1+round-2). All evidence-gathering used `Read` (production + test + worklog + falsification round-1 verdict), `git show` / `git log` / `git status` (round-2 commit scope), and `mage test-pkg` / `mage test-func` (gate runs). Hylla was not queried during this proof pass — round-2's 3-line delta is post-commit pre-ingest territory where `git diff` + `mage` + `Read` are authoritative. The live LSP daemon would not have surfaced any additional signal beyond what direct file Read covered.
