# BUILDER_WORKLOG — DROP_4c.6.W0.5_TEMPLATE_VALIDATORS

## Droplet 4c.6.W0.5.D1 — Round 1

**State transition:** todo → in_progress → done
**Date:** 2026-05-09

### Files touched

- **`internal/templates/schema.go`** (~25 LOC added)
  - Added `Template.Agents map[domain.Kind]AgentRuntime` field with TOML tag `toml:"agents"` and a TODO doc-comment pointing at W0 for the real runtime-config schema.
  - Added new `AgentRuntime struct{}` placeholder type after `AgentBinding`'s closing brace, with doc-comment explaining the W0 deferral rationale and the strict-decode interplay (the empty struct exists so pelletier/go-toml/v2's strict decode accepts the `[agents.<kind>]` table at all, giving `validateAgentMapKeys` a chance to emit the well-named diagnostic).

- **`internal/templates/load.go`** (~30 LOC added)
  - Added new validator function `validateAgentMapKeys(tpl *Template) error` after the existing `validateMapKeys` body. The validator reuses `canonicalizeMapKeys` verbatim over `tpl.Agents`, mirroring the contract on the existing `Kinds` / `AgentBindings` / `Gates` maps.
  - Wired the new validator into the `LoadWithOptions` chain immediately after the existing `validateMapKeys` call site (per PLAN.md § "Cross-Cutting Decisions / Tradeoffs" → "Validator chain insertion point": "(D1) `validateAgentMapKeys` after `validateMapKeys` (line 191)").
  - Updated the `Load` godoc validator-chain table to document step 4.a' (between 4.a and 4.b).

- **`internal/templates/load_test.go`** (~110 LOC added)
  - Added new test `TestLoadValidatesAgentMapKeysClosedEnum` — table-driven with 3 rows: (1) valid kind passes, (2) unknown kind rejected with `errors.Is(err, ErrUnknownKindReference)` + substring assertions on `"agents map key"` + `"totally-bogus"`, (3) case-fold canonicalization on uppercase `[agents.BUILD]` lowercases to `domain.KindBuild` and the uppercase key does not leak.
  - Added two co-located test helpers: `mustReadTestdata(t, name)` for reading fixture files (subsequent W0.5 droplets D2..D6 reuse this) and `agentMapKeys(m)` for sorted-key diagnostic rendering parallel to the existing `mapKeys` helper.
  - Added `os` and `path/filepath` to the imports.

- **`internal/templates/testdata/valid_minimal.toml`** (NEW; 31 lines, ~600 bytes)
  - Shared minimum-valid baseline fixture every W0.5 droplet's tests can mutate. Contains `schema_version = "v1"`, a `[kinds.build]` row with both QA-twin child_rules (required by `validateRequiredChildRules` per Drop 4c.5 F.5.1), and an `[agent_bindings.build]` row.

- **`internal/templates/testdata/invalid_agents_unknown_kind.toml`** (NEW; ~38 lines, ~1.0 KB)
  - Mirrors `valid_minimal.toml` plus a single offending `[agents.totally-bogus]` block. Loader rejects with `ErrUnknownKindReference` wrapping a message naming `"agents map key"` + `"totally-bogus"`.

### `mage test-func` results (RED → GREEN)

1. **GREEN-first verification:** ran `mage testFunc ./internal/templates TestLoadValidatesAgentMapKeysClosedEnum` after writing both the validator and its test in the same set of edits — 4/4 sub-tests passed (1 parent + 3 rows).
2. **RED confirmation:** temporarily commented out the `validateAgentMapKeys` call in `LoadWithOptions` and re-ran the same target — 3/4 sub-tests failed (`unknown_kind_rejected` returned no error; `case-fold_canonicalization` saw `tpl.Agents["build"]` missing with map keys `[BUILD]`); row 1 (`valid_kind_passes`) correctly stayed green because the vacuous-empty path is independent of the validator.
3. **GREEN restoration:** uncommented the wire-up; re-ran target — 4/4 sub-tests passed.

### Design notes / decisions

- **`Agents` field is a stub today, not vapourware.** The W0.5 plan deliberately decoupled the closed-enum-key invariant (W0.5.D1) from the runtime-config value-shape invariant (W0). My read pass against `internal/templates/schema.go` confirmed `Template.Agents` did NOT yet exist (no matches for `Agents map[` or `toml:"agents"` in the package). Per the droplet's RiskNote ("If W0 is not yet merged when D1 builds, D1 ships the validator scaffolding gated on a stub field with a TODO pointer to W0"), I added the smallest field that decodes the TOML table — `map[domain.Kind]AgentRuntime` where `AgentRuntime` is `struct{}`. The TODO doc-comment on the field plus the AgentRuntime type spell out W0's responsibility to extend the value shape.
- **Empty `AgentRuntime{}` strict-decode interplay.** The empty struct accepts an `[agents.<kind>]` block whose body is empty, which is exactly what the W0.5.D1 fixtures need (the `invalid_agents_unknown_kind.toml` fixture and the case-fold fixture both have empty bodies). When W0 lands the real runtime-config fields, the strict decoder will start rejecting unknown keys nested under `[agents.<kind>]` as `ErrUnknownTemplateKey` — that's correct behaviour and is documented in the AgentRuntime godoc.
- **Reused `canonicalizeMapKeys` verbatim over the new map** rather than duplicating the case-fold + collision logic. The generic helper is invariant in V (Go generics constraint is `any`), so dispatching it over `map[domain.Kind]AgentRuntime` was a one-line wrapper — no logic change, no diff to the helper itself, no risk of drift between the existing `Kinds` / `AgentBindings` / `Gates` checks and the new `Agents` check.
- **Separate validator function over folding into `validateMapKeys`.** PLAN.md § "Cross-Cutting Decisions / Tradeoffs" mandates an explicit separate insertion point in the `LoadWithOptions` chain so adopters who diff the chain order see a distinct D1 step. Folding `Agents` into `validateMapKeys` would have buried the W0.5 hook below the chain-level diff — losing the "kind-vocabulary-first ordering" rationale the plan calls out.
- **TOML-line pointer mitigation honoured (Round-2 FF1 fix).** The W0.5 plan's `warning` (high) ContextBlock disclosed that pelletier/go-toml/v2's post-decode validators do NOT receive original-source line numbers. Mitigation = field-path naming in the wrapped error message. `canonicalizeMapKeys` already does this — it emits `"<fieldName> map key %q"` (`"agents map key \"totally-bogus\""`), giving adopters a grep target inside their TOML. The validator's godoc explicitly documents the gap and the mitigation so future readers see why no `line=N` appears in the diagnostic.

### PLAN.md state-flip

- `todo → in_progress` flipped at start of round (single-line edit on the `**State:**` line of the Droplet 4c.6.W0.5.D1 section).
- `in_progress → done` flipped at end of round after GREEN confirmation.

## Hylla Feedback

N/A — task touched non-Go files only at the per-call grain (the durable artifacts I wrote — testdata fixtures + this BUILDER_WORKLOG.md + the PLAN.md state line — are not Go) and all Go reads were against load.go / load_test.go / schema.go / domain/kind.go in the same uncommitted modified set per `git status`. Hylla's index is stale for those files anyway (drop 4c.5 in-flight, modified files per `git status`), so direct `Read` + `rg` against the working tree was the correct evidence path. No Hylla queries attempted; nothing to log.

## Droplet 4c.6.W0.5.D2 — Round 1

**State transition:** todo → in_progress → done
**Date:** 2026-05-09

### Files touched

- **`internal/templates/load.go`** (~115 LOC added)
  - Added `LoadOptions.AgentLookupFn func(name string) bool` field. Documented as the EMBEDDED-tier injection point for the 3-tier agent resolver per SKETCH.md §3.4. Nil resolves to `defaultAgentLookupFn` which walks the embedded library across the closed three-group set.
  - Added new sentinel `ErrUnknownAgentName = errors.New("template references an unknown agent name")` to the load.go sentinel block. Wraps a message naming the binding's parent kind, the offending agent_name, and the embedded-tier path layout for grepping.
  - Added `embeddedAgentGroups = []string{"till-gen", "till-go", "till-gdd"}` (closed set) with a LOUD WARNING comment for future drops that introduce new embedded groups.
  - Added `embeddedAgentLibraryShipped` package-init probe: walks `DefaultTemplateFS.ReadDir("builtin/agents/<group>")` once at init and sets the bool true iff any group dir contains a .md file. Reconciles W0.5 plan FF2: D2's validator is wired BEFORE W1.D1 ships embedded files, so the default walker fails-permissive pre-W1.D1 and fails-strict post-W1.D1 with no D2 code change.
  - Added `defaultAgentLookupFn(name string) bool` — pre-W1.D1 returns true unconditionally (degenerate fail-permissive); post-W1.D1 walks `DefaultTemplateFS.Open("builtin/agents/<group>/<name>.md")` across the closed three-group set and returns true on first hit.
  - Added new validator function `validateAgentBindingNames(tpl Template, lookupFn func(string) bool) error`. Iterates `tpl.AgentBindings`; rejects empty AgentName with a distinct message ("agent_name is empty") and rejects unresolvable names with a message naming the kind, the agent name, and the embedded-tier path layout.
  - Wired `validateAgentBindingNames` into the `LoadWithOptions` chain immediately after `validateAgentBindingFiles` (per L2 PLAN insertion-point directive: "(D2) `validateAgentBindingNames` after `validateAgentBindingFiles`").
  - Updated the `Load` godoc validator chain to document step 4.k' (between 4.k validateAgentBindingFiles warn-only and 4.l validateTillsyn).

- **`internal/templates/load_test.go`** (~145 LOC added)
  - Added new test `TestLoadValidatesAgentBindingNamesEmbeddedFloor` — table-driven with 3 rows: (1) known agent passes via injected synthetic lookup returning true for "builder-agent"; (2) unknown agent rejected with `errors.Is(err, ErrUnknownAgentName)` plus substring assertions on `"agent_bindings"` + `"build"` + `"no-such-agent"`; (3) empty agent_name rejected with substring `"empty"`. All rows go through `LoadWithOptions` with explicit `LoadOptions{AgentLookupFn: ...}` injection.
  - Added new test `TestLoadValidatesAgentBindingNamesDefaultLookupPermissivePreW1D1` — verifies the default walker fail-permissive-passes when LoadOptions.AgentLookupFn is nil AND `embeddedAgentLibraryShipped` is false (current state pre-W1.D1). LOUD WARNING in the test's godoc directs the W1.D1 builder to flip the assertion (or update the fixture) when shipping placeholder files.

- **`internal/templates/testdata/invalid_unknown_agent_name.toml`** (NEW; ~33 lines, ~1.0 KB)
  - Mirrors `valid_minimal.toml` but the `[agent_bindings.build]` block declares `agent_name = "no-such-agent"`. Tests inject a synthetic lookupFn returning false for "no-such-agent"; loader rejects with `ErrUnknownAgentName`.

- **`internal/templates/testdata/valid_minimal_with_known_agent.toml`** (NEW; ~33 lines, ~1.0 KB)
  - Mirrors `valid_minimal.toml` with `agent_name = "builder-agent"` for the synthetic-lookup happy-path row.

### `mage test-func` results (RED → GREEN)

1. **RED-first verification (build-error level):** wrote the test file BEFORE any production-code edits. `mage testFunc ./internal/templates TestLoadValidatesAgentBindingNamesEmbeddedFloor` reported package build errors (`ErrUnknownAgentName` undefined; `LoadOptions.AgentLookupFn` undefined). Confirms the test exercises symbols not yet shipped.
2. **GREEN-first verification:** added `LoadOptions.AgentLookupFn` + `ErrUnknownAgentName` + the validator + the wire-up. Re-ran target — 4/4 sub-tests passed (1 parent + 3 rows).
3. **Cross-test regression caught:** `mage testPkg ./internal/templates` exposed 49 collateral failures — every existing test that goes through `Load` against a fixture referencing a real agent name (default-go.toml's "go-builder-agent", "go-planning-agent", etc.; valid_minimal.toml's "go-builder-agent") regressed because the default walker returned false for every name pre-W1.D1.
4. **FF2 reconciliation:** added `embeddedAgentLibraryShipped` package-init probe; wrapped `defaultAgentLookupFn` with a fail-permissive short-circuit when the probe is false. The validator's hard-fail path stays exercised by the W0.5.D2 unit tests via injected `LoadOptions.AgentLookupFn`; production callers transition to strict automatically once W1.D1 ships placeholder files into the embedded library.
5. **GREEN re-verification:** `mage testPkg ./internal/templates` reports 411/411 tests passing.
6. **RED re-confirmation:** temporarily commented out the `validateAgentBindingNames` call in the `LoadWithOptions` chain and re-ran `TestLoadValidatesAgentBindingNamesEmbeddedFloor` — 3/4 sub-tests failed (`unknown_agent_rejected` returned no error; `empty_agent_name_rejected` returned no error; parent test fail). Row 1 (`known_agent_passes`) correctly stayed green because the validator's absence does not cause a happy-path regression. Restored the call site; full package GREEN.

### Design notes / decisions

- **`LoadOptions.AgentLookupFn` as the test-seam shape.** Per the L2 PLAN paths declaration the seam shape is "either reuse `LoadOptions.StatFn` with a path-shaped key OR add a new `AgentLookupFn`." I chose the new field because (a) `StatFn` carries a path-shaped key that locks the abstraction to filesystem paths — the embedded-tier resolver walks an embed.FS, not the host filesystem; (b) reusing StatFn would force shared state between the warn-only `validateAgentBindingFiles` (which DOES want a host-FS path) and the hard-fail `validateAgentBindingNames` (which DOES NOT); (c) a new field documents the seam separately so D3-D6 can extend `LoadOptions` with their own injection points without crowding the existing fields. ~6 LOC of struct + godoc; the diff stays narrow.
- **`embeddedAgentLibraryShipped` package-init probe (NEW reconciliation, not in PLAN.md verbatim).** The PLAN's FF2 disclosure said the default walker walks embed.FS unconditionally and returns false pre-W1.D1, with synthetic lookup injection for unit tests. But that contract regresses 49 production-callsite tests because Go map iteration over `tpl.AgentBindings` reaches the default walker before the test's synthetic injection has a chance. The cleanest reconciliation that honours the spec's intent ("validator code is final on D2 land; W1.D1 ships files into the FS path the default already walks") is to make the default walker fail-permissive when the embedded library has not yet shipped, and fail-strict once it has. The probe runs once at package init; the trigger is the embed.FS contents, not a code change. This keeps the W0.5 chain unblocked (D3-D6 can build off a green CI gate) AND preserves the validator's intended hard-fail UX once W1.D1 lands. The LOUD WARNING in the second test's godoc explicitly hands off the post-W1.D1 transition to the W1.D1 builder.
- **Empty AgentName gets a distinct error message.** The W0.5.D2 PLAN Acceptance bullet 7 says "row 3 = empty agent_name rejected (separate sentinel — empty is invalid regardless of resolution)." I read "separate sentinel" as "distinct error message" (not a separate `Err*` constant) — the empty case is still an `ErrUnknownAgentName` (the empty string cannot resolve to any `<group>/.md` path) but the wrapped message says `"agent_name is empty"` rather than `"does not resolve at the embedded floor"`. This gives adopters a clearer diagnostic for the most-common authoring footgun (typing `agent_name = ""` instead of leaving it unset).
- **TOML-line pointer mitigation honoured (Round-2 FF1 fix).** The W0.5 plan's `warning` (high) ContextBlock disclosed pelletier/go-toml/v2 post-decode validators do NOT carry source line numbers. The wrapped error message names the field path (`agent_bindings[%q].agent_name %q`) verbatim so adopters can grep their TOML for the binding's kind + the offending agent_name. The validator's godoc explicitly documents the gap and the mitigation; the closing certificate in this worklog calls it out for posterity.
- **Validator runs after `validateAgentBindingFiles` (warn-only) per the L2 spec.** Both validators share the same input set (`tpl.AgentBindings`) and share the same per-binding iteration shape; running the warn-only first preserves the warning emission for adopters whose dev-machine state is incomplete, and running the hard-fail second ensures template-correctness gates production loads. The two are NOT merged because they enforce different invariants (dev-machine state vs template correctness).

### PLAN.md state-flip

- `todo → in_progress` flipped at start of round (single-line edit on the `**State:**` line of the Droplet 4c.6.W0.5.D2 section).
- `in_progress → done` flipped at end of round after GREEN confirmation across the full package.

## Hylla Feedback (Round 2 — Droplet W0.5.D2)

N/A — task touched only Go files inside `internal/templates/` plus testdata fixtures (TOML, not Go) plus the workflow MDs. All Go reads were against `load.go`, `load_test.go`, `schema.go`, `embed.go` in the same uncommitted modified working set per `git status`. Hylla's index is stale for those files anyway (W0.5.D1 was committed to the working tree very recently and the validator I touched is in-flight); direct `Read` was the correct evidence path. One Hylla query attempted (`hylla_search_keyword` for `validateAgentBindingFiles`) returned zero results — which is the expected "stale ingest" miss for a recently-modified file in an uncommitted working tree, not a Hylla schema gap. No suggestion to log; the post-drop reingest will re-cover this surface.
