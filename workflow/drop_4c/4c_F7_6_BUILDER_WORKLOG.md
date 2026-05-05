# Drop 4c F.7.6 — Builder Worklog

**Droplet:** F.7.6 — Required system-plugin pre-flight check.
**Status:** complete (NOT committed; orchestrator commits post-QA-green per F.7-CORE REV-13).
**Verifier:** `mage check` + `mage ci` green (23 pkgs, 2577 passed, 1 pre-existing skip, internal/templates 97.0% coverage, internal/app/dispatcher 74.0% coverage).

## Goal

Add system-plugin pre-flight check that runs at per-dispatch (not bootstrap — there is no `till bootstrap` command on this codebase today) by:

1. Extending the `templates.Tillsyn` struct (declared in F.7.18.2; previously extended by F.7.1 with `SpawnTempRoot`) with a new `RequiresPlugins []string` field per master PLAN.md §5 Tillsyn-struct extension policy + REV-7.
2. Validating each entry at template Load time (non-empty, no whitespace, single `@` separator with both segments non-empty when scoped, within-list duplicates rejected).
3. Shipping `dispatcher.CheckRequiredPlugins(ctx, lister, required)` + `dispatcher.ClaudePluginLister` interface + `execClaudePluginLister` production shell-out impl over `claude plugin list --json`.
4. Wiring per-dispatch via a new `dispatcher.RequiredPluginsForProject` package-level hook BuildSpawnCommand consults before adapter resolution. A nil hook (the package default) short-circuits the pre-flight to a no-op so adopters who do not opt in pay no exec cost per spawn.

## Deliverables

### `internal/templates/schema.go`

- Extended `Tillsyn` struct with `RequiresPlugins []string` field (TOML tag `requires_plugins,omitempty`). Doc-comment cites F.7.6 contract: bare-name vs `<name>@<marketplace>` shapes accepted; matcher rules; empty/nil short-circuit. The type-level Tillsyn doc-comment already mentioned F.7.6 (added speculatively by F.7.18.2's worklog) so no further header edit was needed.

### `internal/templates/load.go`

- Extended `validateTillsyn` to call new `validateTillsynRequiresPlugins` helper.
- New `validateTillsynRequiresPlugins(entries []string)` enforces the entry contract: non-empty, no whitespace (space/tab/CR/LF), at-most-one `@`, both segments non-empty when scoped, within-list duplicates rejected (case-sensitive). Returns `ErrInvalidTillsynGlobals`-wrapped errors naming the offending entry.
- Updated `validateTillsyn` doc-comment + `Per REV-3` paragraph to cite F.7.6 alongside F.7.1.

### `internal/templates/load_test.go`

Added 5 new test functions:

| # | Test | Asserts |
|---|------|---------|
| 1 | `TestLoadTillsynRequiresPluginsHappyPath` | Non-empty `requires_plugins` with bare + scoped shapes loads cleanly; entries lands verbatim. |
| 2 | `TestLoadTillsynRequiresPluginsOmittedZeroValue` | Omitted `[tillsyn]` table OR omitted `requires_plugins` key → nil slice (zero value). |
| 3 | `TestLoadTillsynRequiresPluginsEmptySliceAllowed` | Explicit empty slice `[]` loads cleanly. |
| 4 | `TestLoadTillsynRequiresPluginsRejectionTable` | 8-row table covering: empty entry, whitespace (space + tab), >1 `@`, empty name before `@`, empty marketplace after `@`, bare duplicate, scoped duplicate. Each rejection wraps `ErrInvalidTillsynGlobals` and the error message names the offending construct. |
| 5 | `TestLoadTillsynRequiresPluginsCaseSensitiveDistinct` | `Context7` and `context7` accepted as distinct entries (no fold-matching) since plugin IDs are case-sensitive. |

### `internal/app/dispatcher/plugin_preflight.go` (NEW)

- `RequiredPluginsForProject func(domain.Project) []string` — package-level injection hook. Default nil; adopters wire at boot. Doc-comment explains the seam rationale (KindCatalog does not yet carry [tillsyn] globals; future droplet plumbs it through).
- `pluginPreflightTimeout = 5 * time.Second` constant per F.7.6 acceptance.
- `ClaudePluginListEntry` struct with JSON tags `id`, `marketplace`, `version`, `installPath` per spawn-architecture memory §1 Path B.
- `ClaudePluginLister` interface — single-method `List(ctx) ([]ClaudePluginListEntry, error)`. Tests inject fakes; production wires `execClaudePluginLister`.
- `ErrMissingRequiredPlugins`, `ErrClaudeBinaryMissing`, `ErrPluginListUnparseable` sentinels — distinguishable via `errors.Is` so callers can branch on remediation path (install plugin vs install claude vs report bug).
- `CheckRequiredPlugins(ctx, lister, required) error`:
  - Empty/nil `required` → nil short-circuit before invoking lister.
  - Nil lister + non-empty required → `ErrInvalidSpawnInput`-wrapped error (programming guard).
  - Aggregates ALL missing entries into one error message with order preservation. Each missing entry rendered as `<entry> (run: claude plugin install <entry>)` so the dev sees the full install list in one shot.
- `findMissingPlugins`, `pluginIsInstalled`, `splitPluginEntry`, `formatMissingPlugins` helpers (package-private).
- `execClaudePluginLister` production type with `List(ctx)` method:
  - Wraps caller ctx in `context.WithTimeout(ctx, pluginPreflightTimeout)`.
  - `exec.CommandContext(bounded, "claude", "plugin", "list", "--json")`.
  - Captures stdout + stderr separately; routes `exec.ErrNotFound` via `ErrClaudeBinaryMissing`; routes ctx cancel via `bounded.Err()`; routes non-zero exit via formatted error naming exit code + stderr tail.
  - Calls `parseClaudePluginList` on success.
- `parseClaudePluginList(stdout)` — empty/whitespace-only stdout returns `(nil, nil)`; valid JSON array decodes; any other shape returns `ErrPluginListUnparseable`-wrapped error.
- `defaultClaudePluginLister ClaudePluginLister = execClaudePluginLister{}` — production singleton.

### `internal/app/dispatcher/plugin_preflight_test.go` (NEW)

Added 14 test functions covering happy/missing/empty/error/forward-compat paths via mock lister:

| Category | Test |
|----------|------|
| Short-circuit | `TestCheckRequiredPluginsNilRequiredReturnsNil` — nil `required` returns nil; lister.Calls = 0. |
| Short-circuit | `TestCheckRequiredPluginsEmptyRequiredReturnsNil` — explicit `[]` returns nil; lister.Calls = 0. |
| Happy | `TestCheckRequiredPluginsAllInstalledReturnsNil` — bare + scoped both satisfied; lister.Calls = 1. |
| Failure | `TestCheckRequiredPluginsOneMissing` — one missing; error names entry + install instruction; installed entry NOT in error. |
| Failure | `TestCheckRequiredPluginsMultipleMissingAggregates` — three missing aggregated; order preserved alpha < beta < gamma. |
| Matcher | `TestCheckRequiredPluginsScopedRequirementMatchesScopedInstalled` — 3 sub-cases (match, marketplace mismatch, bare-marketplace installed). |
| Matcher | `TestCheckRequiredPluginsBareRequirementIgnoresMarketplace` — 4 sub-cases (any marketplace matches; missing ID misses). |
| Propagation | `TestCheckRequiredPluginsListerErrorPropagates` — sentinel from fake lister wraps through; does NOT wrap `ErrMissingRequiredPlugins`. |
| Guard | `TestCheckRequiredPluginsNilListerRejected` — nil lister + non-empty required returns `ErrInvalidSpawnInput`. |
| Parser | `TestParseClaudePluginListEmpty` — empty + whitespace-only stdout returns `(nil, nil)`. |
| Parser | `TestParseClaudePluginListHappy` — canonical 2-row JSON decodes correctly. |
| Parser | `TestParseClaudePluginListMalformed` — garbage / object-not-array / trailing-garbage all return `ErrPluginListUnparseable`. |
| Parser | `TestParseClaudePluginListForwardCompatUnknownFields` — future claude versions adding fields decode cleanly (encoding/json non-strict default). |
| Helpers | `TestSplitPluginEntry` — bare vs scoped split. |
| Wiring | `TestExecClaudePluginListerProductionWiring` — `defaultClaudePluginLister` is `execClaudePluginLister`. |
| Hook | `TestRequiredPluginsForProjectHookDefaultIsNil` — package default is nil. |
| Hook | `TestRequiredPluginsForProjectHookReceivesProject` — hook receives `domain.Project` verbatim. |

### `internal/app/dispatcher/spawn.go`

- Added per-dispatch pre-flight invocation between binding lookup/validate and binding resolution. The pre-flight reads the package-level `RequiredPluginsForProject` hook (nil-tolerant short-circuit), invokes `CheckRequiredPlugins` against `defaultClaudePluginLister`. Failures wrap with `dispatcher: plugin pre-flight for kind %q: %w` so `errors.Is` routes through `ErrMissingRequiredPlugins` or `ErrClaudeBinaryMissing` cleanly.
- Doc-comment explains the seam rationale (KindCatalog deferred plumbing) so future maintainers understand why the hook exists rather than direct catalog field access.

## Acceptance Criteria — Closed

- [x] `Tillsyn.RequiresPlugins []string` extension added with TOML tag `requires_plugins,omitempty`.
- [x] Validator rejects empty / whitespace / multi-`@` / empty-segment / duplicate entries with `ErrInvalidTillsynGlobals`.
- [x] `CheckRequiredPlugins(ctx, lister, required)` function shipped.
- [x] `ClaudePluginLister` interface + `execClaudePluginLister` production shell-out impl over `claude plugin list --json` with 5s timeout.
- [x] Mock-based tests cover happy / missing-one / missing-multiple / empty-required / nil-required / scoped-match / bare-match / lister-error-propagation paths.
- [x] Wired into appropriate dispatch hook (per-dispatch via `RequiredPluginsForProject` seam in `BuildSpawnCommand`).
- [x] `mage check` + `mage ci` green (all 23 packages, 2577 tests, 1 pre-existing skip, all coverages above 70%).
- [x] Worklog written.
- [x] **NO commit by builder** per F.7-CORE REV-13.

## Wire-In Decision

The spawn prompt sanctioned wiring "into `till bootstrap` OR per-dispatch pre-flight (whichever fits the codebase boot pattern — likely `cmd/till/main.go` or a service init hook)." On inspection:

- There is no `till bootstrap` subcommand. `cmd/till/main.go` dispatches across many subcommands without a single entry-time hook suitable for a one-shot pre-flight.
- The `dispatcher.RunOnce` hot path (and its `BuildSpawnCommand` substep) is the cleanest per-dispatch site — every spawned action item passes through it.
- `templates.KindCatalog` (the per-project baked snapshot fed to `BuildSpawnCommand`) does NOT carry the [tillsyn] globals today. Extending KindCatalog with `RequiresPlugins []string` is OUT OF SCOPE per the spawn prompt's hard constraint `Edit ONLY listed files + worklog` (catalog.go is not in the list).

The chosen wire-in is a package-level hook `RequiredPluginsForProject func(domain.Project) []string` consumed by `BuildSpawnCommand`. The hook is nil by default — adopters opt in by assigning a function at process boot. This:

1. Honors the hard scope constraint (no catalog.go edits).
2. Provides a real per-dispatch wire (the call site exists, runs every spawn, and propagates failures via `errors.Is`).
3. Makes the data-feed integration a one-line assignment in a future droplet (extend KindCatalog with RequiresPlugins, then `RequiredPluginsForProject = func(p) { ... }` in cmd/till boot).
4. Keeps the no-op short-circuit fast — adopters with no required plugins pay zero exec cost per spawn.

The worklog flags this seam explicitly so the next builder can wire the data feed without re-deriving the rationale.

## Production Lister Coverage Note

The production `execClaudePluginLister.List` is NOT exercised by an integration test today — invoking real `claude plugin list --json` from CI is fragile (CI runners may not carry the claude binary, and the binary's output format is owned by Anthropic, not Tillsyn). The helper functions (`parseClaudePluginList`, `splitPluginEntry`, `findMissingPlugins`, `formatMissingPlugins`) and the matcher (`CheckRequiredPlugins`) are fully covered by mock-driven tests.

When dogfood begins (Drop 5+) and a real claude install is part of the dev environment, an opt-in integration smoke test using `t.Skip` when `claude` is missing on PATH can land then. Until then the wiring assertion (`TestExecClaudePluginListerProductionWiring`) guards against the singleton drifting away from the production type, which is the most important property to pin pre-dogfood.

## Out-of-Scope (deferred)

- KindCatalog extension to carry `RequiresPlugins` — listed by future droplet (catalog.go not in this droplet's edit list).
- Auto-install of missing plugins — explicit non-goal per F.7.6 acceptance criteria.
- Plugin version constraint enforcement — out of scope.
- Codex plugin-list semantics — Drop 4d.
- Bootstrap subcommand (`till bootstrap`) — not present in codebase today; per-dispatch wire-in is the chosen path.

## Verification

```
mage ci
  tests: 2578
  passed: 2577
  failed: 0
  skipped: 1   # pre-existing TestStewardIntegrationDropOrchSupersedeRejected
  packages: 23
  pkg passed: 23
  pkg failed: 0
  internal/templates coverage: 97.0%
  internal/app/dispatcher coverage: 74.0%
```

All packages at or above the 70% coverage gate.

## Hylla Feedback

`N/A — task touched non-Go reference docs (workflow MDs, plan MDs) only for evidence; Go edits were directed by spawn prompt with explicit file paths.`

The spawn prompt enumerated every Go file path verbatim. No symbol search was needed beyond verifying the existing `validateTillsyn` shape and the `BuildSpawnCommand` flow. Direct `Read` on the named files satisfied every evidence need; no Hylla queries issued; no miss to record.

## REV-7 + REV-13 Receipts

- **REV-7**: F.7.18.2 (initial Tillsyn struct, two fields) + F.7.1 (SpawnTempRoot extension) landed before this droplet. F.7.6 adds exactly one field (`RequiresPlugins []string`) and one validator helper. The strict-decode unknown-key contract on the Tillsyn struct continues to fire (verified by the existing `TestLoadTillsynStrictDecodeUnknownFieldRejected` test).
- **REV-13**: builder did NOT run `git commit`. Files are staged for the orchestrator to commit after QA pair (proof + falsification) returns green. `git status --short` reflects the four modified files (load.go, load_test.go, schema.go, spawn.go) plus the two new files (plugin_preflight.go, plugin_preflight_test.go) plus this worklog.
