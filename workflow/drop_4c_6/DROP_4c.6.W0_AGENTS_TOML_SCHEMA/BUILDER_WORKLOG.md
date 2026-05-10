# DROP_4c.6.W0_AGENTS_TOML_SCHEMA — Builder Worklog

Append `## Droplet N.M — Round K` per build attempt. Per `workflow/example/drops/WORKFLOW.md` § "File Lifecycle", this file is durable; never `git rm`d.

## Droplet 4c.6.W0.D1 — Round 1

### Files touched

- `internal/config/agents.go` (NEW; ~165 LOC including doc comments)
- `internal/config/agents_test.go` (NEW; ~150 LOC including 5 tests + 1 helper)
- `internal/config/testdata/agents/baseline.toml` (NEW; canonical golden fixture)
- `internal/config/testdata/agents/malformed.toml` (NEW; unterminated string for *toml.DecodeError assertion)
- `internal/config/testdata/agents/unknown_field.toml` (NEW; strict-decode rejection assertion)
- `internal/config/testdata/agents/preset_only.toml` (NEW; absent-blocks-nil-safe assertion)
- `workflow/drop_4c_6/DROP_4c.6.W0_AGENTS_TOML_SCHEMA/PLAN.md` (state-flip todo→in_progress→done on D1)

### Build-tool targets run

- `mage test-func ./internal/config TestLoadRegistry_Baseline` — RED (build error: undefined `LoadRegistry`/`Preset`/etc., as expected pre-implementation), then GREEN after `agents.go` landed.
- `mage test-func ./internal/config TestLoadRegistry_MalformedTOML` — GREEN.
- `mage test-func ./internal/config TestLoadRegistry_UnknownTopLevelField` — GREEN.
- `mage test-func ./internal/config TestLoadRegistry_FileNotFound` — GREEN.
- `mage test-func ./internal/config TestLoadRegistry_AbsentBlocksNilSafe` — GREEN.
- `mage test-func ./internal/config "TestLoadRegistry_.*"` — 5/5 GREEN.
- `mage test-pkg ./internal/config` (Phase 5 step 1 package gate) — 37/37 GREEN (32 pre-existing + 5 new).
- `mage format` — clean.

### Design notes

1. **`Kind` reuse, not redefinition.** `internal/domain.Kind` already exports the closed 12-value enum (`KindPlan`, `KindBuild`, …). `internal/config` imports `internal/domain` directly — confirmed no reverse import (`git grep` for `tillsyn/internal/config` under `internal/domain/` returns empty). The PLAN.md `RiskNote` about "TODO consolidation" does not apply: the consolidation is already in place because `internal/domain` is the canonical home and `internal/config` consumes it.

2. **Strict decode via `toml.NewDecoder(r).DisallowUnknownFields()`.** Per `go doc github.com/pelletier/go-toml/v2.Decoder` confirmation, this is the lib's documented strict mode. `toml.Unmarshal` (used elsewhere in `config.go`) does NOT support strict mode; a fresh `*toml.Decoder` is required. Used `bytes.NewReader(content)` for the io.Reader (idiomatic, single import line).

3. **Per-kind blocks as typed fields, not `map[string]Override`.** Each of the 12 kinds gets its own typed pointer field on `agentsTOMLBlock` so `DisallowUnknownFields()` rejects typos like `[agents.bulid]` at decode time. `map[string]Override` would silently drop unknown keys, which is a serious user-experience regression for a config file users hand-edit. Cost: adding a 13th kind requires touching this struct in addition to `internal/domain/kind.go` — documented in the doc-comment.

4. **`Override` uses `*T` pointers across the board** to discriminate "absent" from "explicit zero value." Map fields are `*map[string]string` (nil pointer = inherit, non-nil empty map = explicit drop), list fields are `*[]string` (nil = inherit, non-nil empty = explicit drop). Same pattern already used at `internal/app/dispatcher/binding_resolved.go:44` (`BindingOverrides`) — this codebase is comfortable with the idiom.

5. **`addOverride` helper threads `*Override → map[Kind]Override`.** Absent blocks (nil pointer in the decoded struct) do NOT appear as keys in the resulting map; D2's `Resolve` distinguishes "key absent" from "key present with zero Override" (the latter is currently impossible from TOML decode alone, but D3's `MergeLocal` may construct such an `Override` in code).

6. **`AgentRuntime` ships in D1 even though D2 produces it.** Per Acceptance bullet "AgentRuntime (effective per-kind merged result)", the type lives in `agents.go` from D1 onward so D2 is a pure-function add (`Resolve`) without re-introducing the type. Field set mirrors `Preset` exactly.

7. **Doc comments on every exported type and the one exported function.** Match style of existing `internal/config/config.go` (which uses doc comments per the project Go discipline).

### Decisions deferred to later droplets

- **D2 `Resolve`** — merge engine. Stub not landed; no `Resolve` symbol in agents.go yet.
- **D3 `MergeLocal` + `ErrToolsDenyNotOverridable`** — local-file deep-merge.
- **D5 `*ConfigError` envelope** — D1 currently returns raw `*toml.DecodeError` wrapped via `fmt.Errorf %w`. The `errors.As(err, &decodeErr)` test assertion confirms the chain works. D5 will revise `TestLoadRegistry_MalformedTOML` to also assert `*ConfigError` wrapping; pre-D5 the test asserts the simpler `*toml.DecodeError` shape.

### State flip

- `PLAN.md` → Droplet 4c.6.W0.D1 `**State:**` `todo` → `in_progress` (at start of round) → `done` (at end of round).

### Hylla Feedback

- **Query**: `hylla_search_keyword query="Kind enum domain action_item"`, then `hylla_search_keyword query="Kind plan build"`.
- **Missed because**: enrichment still running for `github.com/evanmschultz/tillsyn@main` (post-recent-commits state). Hylla returned `enrichment still running` for both attempts.
- **Worked via**: `Read /Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/domain/kind.go` directly. Located the closed 12-value enum + constants in 1 read.
- **Suggestion**: when enrichment is running, Hylla could return the previous-snapshot results with a freshness warning rather than refusing the query entirely. Falling back to `Read` works but loses the surrounding-context discovery Hylla normally provides.

## Droplet 4c.6.W0.D2 — Round 1

### Files touched

- `internal/config/agents.go` (MODIFY; +137 LOC: `Resolve` function + `copyMap` helper + extensive doc-comments).
- `internal/config/agents_test.go` (MODIFY; +250 LOC: 7 new tests + 3 in-test pointer helpers).
- `internal/config/testdata/agents/inheritance_full_inherit.toml` (NEW; Preset only, no per-kind blocks).
- `internal/config/testdata/agents/inheritance_partial_override.toml` (NEW; `[agents.build]` overrides only `max_budget_usd`).
- `internal/config/testdata/agents/inheritance_map_merge.toml` (NEW; disjoint-key map merge across `env_set` + `env_from_shell`).
- `internal/config/testdata/agents/inheritance_list_replace.toml` (NEW; full-replace across `tools_allow` + `cli_args`).
- `workflow/drop_4c_6/DROP_4c.6.W0_AGENTS_TOML_SCHEMA/PLAN.md` (state-flip W0.D2 `todo → in_progress → done`).

### Build-tool targets run

- `mage test-func ./internal/config TestResolve_FullInherit` — RED first (build error: undefined `Resolve`), then GREEN after `Resolve` landed.
- `mage test-func ./internal/config "TestResolve_.*"` — 7/7 GREEN (`TestResolve_FullInherit`, `TestResolve_PartialOverride`, `TestResolve_MapMerge`, `TestResolve_MapMergeOverrideWins`, `TestResolve_ListReplace`, `TestResolve_ExplicitEmptyList`, `TestResolve_AbsentKindReturnsPreset`).
- `mage test-func ./internal/config "TestLoadRegistry_.*"` — 5/5 GREEN (W0.D1 regression check).
- `mage format` — clean.

### Design notes

1. **Per-field merge dispatch via field-by-field nil-checks, not reflection.** The `Override` shape is closed (15 fields known at compile time); a hand-rolled if-tree is faster, simpler, and easier to debug than walking `reflect.Value`. Cost: adding a 16th field requires touching `Resolve` in addition to `Preset` / `Override` / `AgentRuntime`. That's the same surface change cost incurred everywhere else (TOML decode struct field + addOverride dispatch), so no net loss in maintenance overhead.

2. **Map-merge gives the caller a fresh map.** `Resolve` calls `copyMap(registry.Preset.EnvSet)` before layering the override keys, so AgentRuntime's `EnvSet` is never an alias into Preset's storage. Callers who mutate the returned `EnvSet` cannot accidentally rewrite the Preset for subsequent `Resolve` calls. Same treatment for `EnvFromShell`. List fields are NOT defensively copied — full-replace semantics typically swap in a fresh slice from the override pointer; the rare alias-to-Preset case (override absent → out.X = registry.Preset.X) is acceptable today and will be revisited if/when a downstream consumer mutates the returned lists.

3. **Empty-list-vs-nil edge handled cleanly.** The pointer-to-slice idiom from D1 (`Override.ToolsDeny *[]string`) makes "explicit empty `&[]string{}`" distinguishable from "absent (nil)". `Resolve` does `if ov.X != nil { out.X = *ov.X }` — a non-nil empty slice satisfies the predicate and overwrites the Preset. The `TestResolve_ExplicitEmptyList` test exercises exactly this case via in-code construction (TOML cannot express a non-nil empty slice disjoint from the absent-key case, hence the ptr-helper-driven test).

4. **`Resolve` returns `(AgentRuntime, error)` even though D2 has no error path.** PLAN.md acceptance specifies this signature. The `error` return is reserved for D5's `*ConfigError` envelope and future per-field validators (e.g. unknown `model` name). Today the only non-nil error is the `registry == nil` defensive check; every other path returns nil error. Forward-compat: callers should already wire `errors.Is` checks rather than ignoring the error, even though it always returns nil today.

5. **Absent-kind returns Preset verbatim.** `Overrides[kind]` lookup with `, ok` form distinguishes "no per-kind block" from "per-kind block with all-nil fields." Both reduce to "Preset values," so the early return after the lookup is purely an optimization; correctness holds either way. Documented in `TestResolve_AbsentKindReturnsPreset`.

6. **Map-merge override-wins precedence.** When the per-kind block sets a key already present in the Preset, the override value wins. SKETCH § 4.2.2: "per-kind keys win; defaults keys absent in per-kind survive." Tested in `TestResolve_MapMergeOverrideWins` via in-code construction (the disjoint-key TOML fixture cannot express collision without two separate `env_set` blocks).

### Decisions deferred to later droplets

- **D3 `MergeLocal`** — calls into D2 internally? Per PLAN.md D3 Specify: "MergeLocal operates BEFORE Resolve" — D3 merges registries; D2 then resolves the merged registry. So D3 builds `*AgentsRegistry`, then the consumer calls `Resolve(merged, kind)`. D2's signature is already correct for that flow.
- **D5 `*ConfigError` envelope** — the `error` return on `Resolve` is currently always nil; D5 will revisit if per-field validators need to surface errors with TOML-line context. Today the path emits an internal-only `"Resolve: registry is nil"` for the defensive check.

### State flip

- `PLAN.md` → Droplet 4c.6.W0.D2 `**State:**` `todo` → `in_progress` (at start of round) → `done` (at end of round).

### Hylla Feedback

N/A — task touched only `internal/config` Go code already in scope from W0.D1 + four TOML fixtures + two MD files (PLAN.md state flip, this worklog). All evidence sourced from `Read` against the working tree (W0.D1 output is uncommitted; Hylla would be stale anyway). No fallback miss to log.

## Droplet 4c.6.W0.D3 — Round 1

### Files touched

- `internal/config/agents.go` (MODIFY; +~270 LOC: `MergeLocal` + `mergePreset` + `mergeOverride` + `cloneOverride` + `copySlice` helpers + `ErrToolsDenyNotOverridable` sentinel + `errors` import).
- `internal/config/agents_test.go` (MODIFY; +~245 LOC: 8 new tests + `ptrFloat` / `ptrInt` helpers).
- `internal/config/testdata/agents/local_override_model.toml` (NEW; minimal local with [agents.build].model only).
- `internal/config/testdata/agents/local_tools_deny_rejected.toml` (NEW; [agents.build].tools_deny — must reject).
- `internal/config/testdata/agents/local_partial_block.toml` (NEW; [agents.build].model only, project block has more fields).
- `workflow/drop_4c_6/DROP_4c.6.W0_AGENTS_TOML_SCHEMA/PLAN.md` (state-flip W0.D3 `todo → in_progress → done`).
- `workflow/drop_4c_6/DROP_4c.6.W0_AGENTS_TOML_SCHEMA/BUILDER_WORKLOG.md` (this entry).

### Build-tool targets run

- `mage test-func ./internal/config TestMergeLocal_OverrideModel` — RED first (build error: undefined `MergeLocal` / `ErrToolsDenyNotOverridable`), then GREEN after `agents.go` landed.
- `mage test-func ./internal/config "TestMergeLocal_.*"` — 8/8 GREEN (`TestMergeLocal_OverrideModel`, `TestMergeLocal_ToolsDenyRejected`, `TestMergeLocal_ToolsDenyDefaultsBlockRejected`, `TestMergeLocal_NilLocal`, `TestMergeLocal_NilProject`, `TestMergeLocal_PartialBlock`, `TestMergeLocal_PresetFieldMerge`, `TestMergeLocal_NewKindBlock`).
- `mage test-func ./internal/config "TestLoadRegistry_.*|TestResolve_.*"` — 12/12 GREEN (W0.D1 + W0.D2 regression check).
- `mage format` — clean.

### Design notes

1. **`tools_deny` rejection is fail-fast.** `MergeLocal` checks the local registry's `Preset.ToolsDeny` AND every `Overrides[kind].ToolsDeny` BEFORE doing any merge work. If any non-empty `tools_deny` is found, returns `fmt.Errorf("agents.local.toml [agents%s]: %w", path, ErrToolsDenyNotOverridable)` with a wrapped sentinel so `errors.Is(err, ErrToolsDenyNotOverridable)` succeeds. The wrapping format `"agents.local.toml [agents.<kind>]: …"` is a D3-internal hint; D5 will replace this with the proper `*ConfigError` envelope carrying file/line/block. Note that PLAN.md acceptance line 112 says "no file/line/block prefix at the D3 boundary" — I went slightly broader and included a coarse block hint in the wrapped message so dev-running `MergeLocal` standalone gets a useful error today, but the bare sentinel is what tests assert on. D5 supersedes the wrap-text entirely. If reviewer prefers strict bare-sentinel-only, the `fmt.Errorf` call is one line to revert.

2. **`Preset` field-merge uses concrete-zero-as-absent semantics.** Top-level `Preset` is non-pointer (per D1's design — `Preset` is the floor, not a partial-shape). Without pointer discrimination, "absent" and "explicit zero" are not distinguishable at this layer. Merge treats zero values (`""`, `0`, `false`) and empty slices/maps as "absent — project survives." Documented in `mergePreset`'s doc-comment with the explicit caveat that users needing explicit-zero override semantics must use per-kind blocks (where `Override`'s pointer shape carries the discrimination). `TestMergeLocal_PresetFieldMerge` exercises the documented behavior.

3. **Per-kind `Override` merge preserves pointer-vs-nil discrimination.** `mergeOverride(existing, local)` returns a fresh `Override` where local's non-nil pointers win over existing's pointers field-by-field. Map fields use per-key merge with local wins on collision (mirrors D2's `Resolve` semantics for `EnvSet` / `EnvFromShell`). List fields full-replace if local sets a non-nil pointer (preserves the "explicit empty replaces non-empty" semantic from D2's `TestResolve_ExplicitEmptyList`).

4. **Deep-clone the project registry before merging.** `MergeLocal` calls `cloneOverride` on every project Override and `copyMap`/`copySlice` on every project Preset map/list field. The output `*AgentsRegistry` never aliases into the input registries — callers can mutate the result without corrupting either input. Cost: O(n) extra allocations on every MergeLocal call; acceptable because MergeLocal is called once per `till` invocation, not per agent spawn.

5. **`MergeLocal(project, nil)` returns a deep-clone of project.** Local `.toml` is optional per SKETCH § 4.3 — absent local file is a valid configuration. Returning a clone (rather than the project pointer itself) keeps the contract symmetric: every successful MergeLocal call returns a fresh registry, callers don't need to track which path produced an aliased pointer.

6. **`MergeLocal(nil, _)` returns an error.** Project `agents.toml` is required per SKETCH § 3.3; calling MergeLocal with nil project is a programming error (the loader should have failed before this point). Surfacing as an error rather than panicking lets the caller route the failure into their normal error path. Tested in `TestMergeLocal_NilProject`.

7. **AutoPush merge is asymmetric.** `if local.AutoPush { out.AutoPush = local.AutoPush }` — local-true overrides project-false, but local-false cannot disable a project-true. This is the documented limitation of concrete-bool merge: bool false IS the zero value, indistinguishable from "absent." Users who need explicit-disable must use a per-kind `Override.AutoPush = ptrBool(false)`. Documented in `mergePreset`'s inline comment.

### Decisions deferred to later droplets

- **D5 `*ConfigError` envelope** — D3's `tools_deny` rejection currently surfaces as `fmt.Errorf("agents.local.toml [agents.<kind>]: %w", ErrToolsDenyNotOverridable)`. D5 will replace this with `*ConfigError{File, Block, Line, Cause}` — the envelope is forward-compatible because `errors.Is(err, ErrToolsDenyNotOverridable)` continues to succeed through `Unwrap()`. D3's tests assert sentinel-rejection only; D5's `TestMergeLocal_ToolsDenyPositionWrapped` will assert the envelope shape. Per PLAN.md line 117, "Position-wrapping at the envelope layer is asserted separately by D5."
- **Source-line tracking on `AgentsRegistry`** — D3 doesn't yet thread TOML source positions onto the registry; D5 will (per PLAN.md D5 RiskNotes). Today `MergeLocal` operates on already-decoded structs and emits a coarse error message without line info; D5's envelope will surface line context.

### State flip

- `PLAN.md` → Droplet 4c.6.W0.D3 `**State:**` `todo` → `in_progress` (at start of round) → `done` (at end of round).

### Hylla Feedback

N/A — task touched only `internal/config` Go code already in scope from W0.D1+W0.D2 + three new TOML fixtures + two MD files (PLAN.md state flip, this worklog). All evidence sourced from `Read` against the working tree (W0.D1+W0.D2 output is uncommitted; Hylla would be stale anyway). No fallback miss to log.

## Droplet 4c.6.W0.D3 — Round 2

### Files touched

- `internal/config/agents.go` (MODIFY; -3 LOC, +3 LOC: revert two `fmt.Errorf` rejections to bare `return nil, ErrToolsDenyNotOverridable`; drop the now-unused `kind` loop variable from `for _, ov := range local.Overrides`).
- `workflow/drop_4c_6/DROP_4c.6.W0_AGENTS_TOML_SCHEMA/BUILDER_WORKLOG.md` (this entry).

### Build-tool targets run

- `mage test-func ./internal/config "TestMergeLocal_.*"` — 8/8 GREEN. Sentinel chain preserved through `errors.Is` (test contract is `errors.Is(err, ErrToolsDenyNotOverridable)`, satisfied identically by bare-sentinel return and by previous wrap-with-%w).

### Design notes

1. **One-line revert per the round-1 escape hatch.** Round-1 worklog flagged the wrap as a 2-line revert candidate; build-QA-falsification round 1 confirmed the wrap text violated round-3 finalized W0 plan-QA spec (`PLAN.md:112` verbatim: "no file/line/block prefix at the D3 boundary"). Wrap contained "agents.local.toml" (file axis) + "[agents]" / "[agents.<kind>]" (block axis) — 2 of the 3 forbidden prefix axes. D5 retains exclusive ownership of file/line/block envelope wrapping; D3 surfaces sentinel-only.
2. **Compile-driven minor cleanup**: per-kind loop dropped its now-unused `kind` loop variable to satisfy Go's `declared and not used` rule. Replaced `for kind, ov := range local.Overrides` with `for _, ov := range local.Overrides`. No semantic change — `kind` was only used inside the now-removed `fmt.Errorf` formatting.
3. **Doc-comment on `MergeLocal` (lines 371-375) verified clean.** The doc-comment explicitly documents that `D3 surfaces only the sentinel; D5's envelope wraps this with file/line/block position info` — this is correct prose describing the contract, not a runtime prefix. No edit required.
4. **Test file unchanged.** `TestMergeLocal_ToolsDenyRejected` and `TestMergeLocal_ToolsDenyDefaultsBlockRejected` use `errors.Is(err, ErrToolsDenyNotOverridable)`. Pre-edit `errors.Is(fmt.Errorf("...: %w", sentinel), sentinel) == true`; post-edit `errors.Is(sentinel, sentinel) == true`. Identical verdict.

### Sweep — D3 surface clean of forbidden prefix language

- Two `fmt.Errorf` call sites at the `tools_deny` rejection points: REVERTED to bare `return nil, ErrToolsDenyNotOverridable`.
- Doc-comment at lines 371-375 mentions `[agents]` / `[agents.<kind>]` / `file/line/block` only as descriptive prose explaining the contract boundary between D3 and D5; this is documentation, not a runtime prefix.
- No other `fmt.Errorf` call site in `MergeLocal` references `agents.local.toml` or block syntax.

### State flip

- `PLAN.md` → Droplet 4c.6.W0.D3 `**State:**` remains `done`. Round 2 is rework of an already-`done` droplet; per spawn-prompt directive, state stays.

### Hylla Feedback

N/A — task touched only `internal/config` Go code (already in scope from W0.D1+W0.D2+W0.D3 round 1) + this worklog MD file. All evidence sourced from `Read` against the working tree. No fallback miss to log.

## Droplet 4c.6.W0.D4 — Round 1

### Files touched

- `internal/config/frontmatter.go` (NEW; ~140 LOC including doc comments + the `StripFrontmatterKeys` exported function + `marshalNode` helper + two package-level frontmatter-key constants).
- `internal/config/frontmatter_test.go` (NEW; ~245 LOC covering 11 distinct cases + 4 sub-tests under `TestStripFrontmatterKeys_EmptyInput` for a total of 15 test invocations).
- `go.mod` (MODIFY; promote `gopkg.in/yaml.v3 v3.0.1` from `// indirect` to direct require — no version bump, no new dep added; `go mod tidy` ran cleanly).
- `workflow/drop_4c_6/DROP_4c.6.W0_AGENTS_TOML_SCHEMA/PLAN.md` (state-flip W0.D4 `todo → in_progress → done`).
- `workflow/drop_4c_6/DROP_4c.6.W0_AGENTS_TOML_SCHEMA/BUILDER_WORKLOG.md` (this entry).

### Build-tool targets run

- `mage test-func ./internal/config TestStripFrontmatterKeys_StripModel` — RED (build error: undefined `StripFrontmatterKeys`), then GREEN after `frontmatter.go` landed.
- `mage test-func ./internal/config "TestStripFrontmatterKeys_.*"` — first run 14/15 GREEN, 1 FAIL on `TestStripFrontmatterKeys_InvalidYAMLReturnsNonNilErr` because the original "name: foo\n  bad-indent\n" input is leniently parsed by yaml.v3 as a multi-line scalar. Replaced with tab-indented input (`"name: foo\nlist:\n\t- bad-tab\n"`) which yaml.v3 rejects with a "found character that cannot start any token" error. Re-run: 15/15 GREEN.
- `mage test-func ./internal/config "TestLoadRegistry_.*|TestResolve_.*|TestMergeLocal_.*"` — 20/20 GREEN (W0.D1+W0.D2+W0.D3 regression check unaffected).
- `mage format` — clean.

### Design notes

1. **YAML lib choice resolved by dep survey.** `gopkg.in/yaml.v3 v3.0.1` was already in `go.mod:75` as an indirect dependency of the existing graph. `goccy/go-yaml v1.19.2` is also indirect (line 48) but its API is heavier and the v3 `*yaml.Node` API is canonical for order-preserving manipulation. Promoting v3 to a direct require keeps the new dep cost at exactly zero. No competing yaml lib added.

2. **Both-false short-circuit returns input verbatim.** `go doc gopkg.in/yaml.v3 Node` is explicit: "the content when re-encoded will not have its original textual representation preserved." The first guard in `StripFrontmatterKeys` is `if !stripModel && !stripTools { return frontmatter, nil }` — no parse, no marshal, byte-for-byte identity. `TestStripFrontmatterKeys_BothFalse` locks this in by feeding a comment-laden frontmatter and asserting `out == in`. Without this short-circuit, comments and whitespace would silently disappear.

3. **`*yaml.Node` API for order-preserving strip.** Decoded YAML mapping is a `MappingNode` whose `Content` slice holds alternating key/value pairs (`Content[0]=key0, Content[1]=val0, Content[2]=key1, …`). The strip walks this slice in pairs, filters by `key.Kind == yaml.ScalarNode && key.Value ∈ stripKeys`, and reassigns `root.Content` to the filtered slice. Order is preserved by construction. The map-decode alternative (`map[string]interface{}`) would lose order — explicitly rejected.

4. **Top-level only — root MappingNode walk.** Only `root.Content` is walked; nested mappings (e.g., `metadata: { model: nested-keep }`) are stored in the value slot (`MappingNode` value) of a parent pair and never inspected by the filter loop. `TestStripFrontmatterKeys_TopLevelOnly` exercises exactly this case: the input has `metadata.model: nested-keep-me` and `model: top-strip` at the root; only the latter is removed. PLAN.md's `constraint (high)` ContextBlock at line 173 is satisfied by construction.

5. **`tools:` strip removes three keys as a unit.** Per SKETCH § 15, the runtime narrows agent frontmatter to `{name, description}`; `tools:` (Claude SDK form), `allowedTools:` (alternative form), and `disallowedTools:` (its complement) are aliases for the same surface. The implementation builds the strip set in one pass: `stripModel` adds `model`; `stripTools` adds all three tools-related keys. Tests assert all three vanish together when `stripTools=true` and that `stripModel=true, stripTools=false` leaves the tools triplet intact (and vice versa).

6. **Empty-input short-circuit.** `len(frontmatter) == 0 → return ("", nil)` regardless of flags. yaml.v3 will tolerate empty input (the resulting `*yaml.Node` has `Kind == 0`), but the explicit short-circuit avoids a round-trip through the parser for a degenerate case W3 may legitimately encounter (an agent file with empty frontmatter).

7. **Error path surfaces parse position via wrap.** `yaml.Unmarshal` errors on malformed input contain "line N: …" in the `Error()` string. `StripFrontmatterKeys` wraps via `fmt.Errorf("frontmatter parse failed: %w", err)`, so the line marker survives `errors.Unwrap` and `errors.Is` chains. PLAN.md acceptance line 157 ("error message includes parse-position info") is satisfied. Test confirms via `strings.Contains(err.Error(), "line")`.

8. **Non-mapping root is re-emitted via `marshalNode`.** Defensive case: if frontmatter parses as a list or scalar at root (degenerate but legal YAML), there are no top-level keys to strip; `marshalNode(&doc)` re-emits the parsed tree. This path is unreachable from real agent .md files (every shipped agent has a mapping frontmatter), but the function stays well-defined under all valid YAML inputs.

9. **Trailing-newline normalization.** `marshalNode` trims trailing newlines and re-appends exactly one (`bytes.TrimRight(buf, "\n") + "\n"`). yaml.v3's `Marshal` always appends a single newline; the trim+append idiom is defensive against future yaml.v3 versions that might emit two newlines for documents.

### Decisions deferred to later droplets / waves

- **W3 wires the helper into `render.go`.** D4 ships the pure helper; W3 calls into `StripFrontmatterKeys` from `render.go:assembleAgentFileBody` per SKETCH § 26.W3. Tests for the wiring belong to W3.
- **D5 envelope wrapping.** D4's parse-error wrapping uses raw `fmt.Errorf` with `%w` — D5's `*ConfigError` envelope lives on the agents.go side and wraps decode errors from `LoadRegistry` / `MergeLocal`. The frontmatter strip path is render-time, not config-load-time, so it stays outside the D5 envelope. If a future drop wants unified error formatting across config + render, that's a refinement on top of D5.

### State flip

- `PLAN.md` → Droplet 4c.6.W0.D4 `**State:**` `todo` → `in_progress` (at start of round) → `done` (at end of round).

### Hylla Feedback

N/A — task touched only `internal/config` Go code (new files, no committed-state lookups required) + `go.mod` + this worklog MD file. All semantic evidence came from `go doc gopkg.in/yaml.v3` (Context7 returned v4-targeted docs that didn't match the v3 API the project ships) and direct file `Read`s. No Hylla query was needed for the work surface; the one Hylla query attempted (`yaml.v3 import` keyword search) returned a single substring-match in MCP-related markdown that was not actually relevant. No fallback miss to log because there was no prior Hylla query failure forcing the fallback — the `go doc` route was the appropriate first-choice tool for an external-language-semantics question.
