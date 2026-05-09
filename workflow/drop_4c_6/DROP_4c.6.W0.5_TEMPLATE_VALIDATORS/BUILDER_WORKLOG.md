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
