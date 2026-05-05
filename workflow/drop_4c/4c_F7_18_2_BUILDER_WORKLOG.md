# Drop 4c F.7.18.2 — Builder Worklog

**Droplet:** F.7.18.2 — Schema-3: `Tillsyn` top-level struct
**Status:** complete
**Verifier:** `mage ci` green (21 pkgs, 2319 passed, 1 pre-existing skip, templates 96.0% coverage)

## Goal

Land NEW top-level `Tillsyn` struct in `internal/templates/schema.go` with EXACTLY two fields (`MaxContextBundleChars int` + `MaxAggregatorDuration Duration`) plus a `Tillsyn Tillsyn` field on `Template`. Wire `validateTillsyn` into the `templates.Load` validator chain so negative values fail load while zero values pass through to engine-time default substitution. Add a strict-decode regression test so REV-3's "extenders inherit unknown-key rejection automatically" contract is provably wired.

## Deliverables

### `internal/templates/schema.go`

- Added `Tillsyn Tillsyn` field to `Template` struct with TOML tag `tillsyn`.
- Added new `Tillsyn` struct with exactly two fields:
  - `MaxContextBundleChars int` (TOML tag `max_context_bundle_chars`).
  - `MaxAggregatorDuration Duration` (TOML tag `max_aggregator_duration`).
- Doc comments cite master PLAN §5 extension policy + REV-3 + master PLAN L14/L15 default-substitution semantics. No additional fields per REV-3 contract — F.7-CORE F.7.1 + F.7.6 extend later.

### `internal/templates/load.go`

- New `ErrInvalidTillsynGlobals` sentinel (top-level Load error, not nested under `ErrInvalidAgentBinding` because the [tillsyn] table is global, not per-binding).
- New `validateTillsyn` function: rejects negative `MaxContextBundleChars` or `MaxAggregatorDuration`; zero-values pass cleanly (engine-time default substitution per master PLAN L14/L15).
- Wired into `Load` validator chain after `validateAgentBindingContext` (step 4h).
- Updated package doc-comment validator-chain enumeration to include step `h.`.

### `internal/templates/schema_test.go`

- Extended `TestTemplateTOMLRoundTrip` to populate the new `Tillsyn` field with `MaxContextBundleChars = 200000` + `MaxAggregatorDuration = 2 * time.Second`. Round-trip equality (TOML → struct → TOML → struct) confirms TOML tags symmetric.

### `internal/templates/load_test.go`

Added 7 new tests covering all 8 prompt-mandated scenarios (the strict-decode regression doubles as scenario 8):

| # | Test | Asserts |
|---|------|---------|
| 1 | `TestLoadTillsynHappyPath` | `[tillsyn] max_context_bundle_chars = 200000, max_aggregator_duration = "2s"` loads, fields land verbatim. |
| 2 | `TestLoadTillsynEmptyTableDecodes` | `[tillsyn]` with all fields omitted loads, struct zero-valued. |
| 3 | `TestLoadTillsynOmittedTableZeroValue` | No `[tillsyn]` table at all loads, struct zero-valued. |
| 4 | `TestLoadTillsynZeroValuesAllowed` | Explicit `max_context_bundle_chars = 0` + `max_aggregator_duration = "0s"` loads (not a `> 0` validator). |
| 5 | `TestLoadTillsynRejectsNegativeMaxContextBundleChars` | `-1` fails with `ErrInvalidTillsynGlobals`; error names `max_context_bundle_chars` + `-1`. |
| 6 | `TestLoadTillsynRejectsNegativeMaxAggregatorDuration` | `"-1s"` fails with `ErrInvalidTillsynGlobals`; error names `max_aggregator_duration` + `-1s`. |
| 7 | `TestLoadTillsynStrictDecodeUnknownFieldRejected` | **REV-3 contract**: `[tillsyn] bogus_field = true` (alongside a valid known field) fails with `ErrUnknownTemplateKey`; error names `bogus_field`. |

Imports widened with `time` for duration assertions.

## Acceptance Criteria — Closed

- [x] `Tillsyn` struct added to `schema.go` with EXACTLY 2 fields.
- [x] `Tillsyn Tillsyn` field added to `Template` with TOML tag `tillsyn`.
- [x] `validateTillsyn` wired into `Load` chain (after `validateAgentBindingContext`).
- [x] All 8 test scenarios pass.
- [x] Strict-decode unknown-key test asserts `[tillsyn] bogus_field = true` fails with `ErrUnknownTemplateKey`.
- [x] `mage check` + `mage ci` green.
- [x] Worklog written.
- [x] NO additional fields on `Tillsyn` beyond the two specified.

## REV-3 Contract Receipt

Strict-decode unknown-key rejection on the new `Tillsyn` struct is inherited automatically from `load.go` step 3 (`DisallowUnknownFields`). The regression test in scenario 7 proves it fires today; F.7-CORE F.7.1 + F.7.6 may extend the struct with `SpawnTempRoot` + `RequiresPlugins` and a future bogus-key test will continue to pass without re-shaping `validateTillsyn`. The validator's body deliberately limits itself to field-level negative checks, so adding a new field there is purely additive.

## Out-of-Scope (deferred per prompt)

- Cross-cap warning (`binding.Context.MaxChars > tpl.Tillsyn.MaxContextBundleChars` → warn-only structured log) — listed in `F7_18_CONTEXT_AGG_PLAN.md` line 159 but explicitly absent from this droplet's prompt acceptance criteria. Templates package has no logger today; introducing one is YAGNI. If F.7.18.4's planner wants the warning, it can land it then.
- Extra fields on `Tillsyn` (e.g. `SpawnTempRoot`, `RequiresPlugins`) — REV-3 forbids; F.7-CORE F.7.1 + F.7.6 own those additions.

## Verification

```
mage ci
  tests: 2320
  passed: 2319
  failed: 0
  skipped: 1   # pre-existing TestStewardIntegrationDropOrchSupersedeRejected
  packages: 21
  pkg passed: 21
  internal/templates coverage: 96.0%
```

All packages at or above the 70% coverage gate.
