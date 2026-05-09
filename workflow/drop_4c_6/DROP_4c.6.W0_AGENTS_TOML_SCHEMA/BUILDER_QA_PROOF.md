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
