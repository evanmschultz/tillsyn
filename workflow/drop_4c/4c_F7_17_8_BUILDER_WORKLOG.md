# Drop 4c — F.7.17.8 Builder Worklog

## Droplet

`4c.F.7.17.8` — `BindingResolved` priority cascade resolver. Lands the
pure-function `ResolveBinding(rawBinding, overrides...)` that implements
the F.7.17 locked decision L9 / L16 priority cascade
(CLI > MCP > TUI > template TOML default > absent) so the dispatcher's
adapter seam consumes a single fully-resolved `BindingResolved` value
without re-resolving inside each adapter.

REVISIONS-first compliance: `BindingResolved` does NOT carry `Command` or
`ArgsPrefix` fields (REV-1); `Env` is the closed-allow-list baseline
(REV-2); `ExtractTerminalReport` is the renamed terminal-extractor
(REV-5). All three are honored implicitly because this droplet only
populates / cascades existing fields on `BindingResolved` — it does NOT
touch `cli_adapter.go`'s type definitions.

## Files edited

- `internal/app/dispatcher/binding_resolved.go` (NEW)
  - `type BindingOverrides struct` — pointer fields for the eight
    overridable scalars per the spawn prompt: `Model`, `Effort`,
    `MaxTries`, `MaxBudgetUSD`, `MaxTurns`, `AutoPush`, `BlockedRetries`,
    `BlockedRetryCooldown`. Doc comment names every field NOT yet
    plumbed for override (`Tools`, `ToolsAllowed`, `ToolsDisallowed`,
    `Env`, `AgentName`, `CLIKind`, `CommitAgent`) so future surface
    additions know where to extend.
  - `func ResolveBinding(rawBinding templates.AgentBinding, overrides ...*BindingOverrides) BindingResolved`
    — pure function, no I/O, no global state. Walks `overrides` slice
    highest→lowest; for each pointer field on the resolved struct, the
    first non-nil pointer wins; on miss, falls back to a copy of the
    rawBinding scalar promoted to a pointer.
  - Five generic-shape helpers (`resolveStringPtr`, `resolveIntPtr`,
    `resolveFloat64Ptr`, `resolveBoolPtr`, `resolveDurationPtr`) factor
    out the cascade walk per field type. Each takes the rawBinding
    scalar fallback, the overrides slice, and a `pick` accessor that
    extracts the field from a single override layer. nil entries in the
    overrides slice are skipped without panic.
  - `cloneStringSlice` returns a defensive copy of every slice field
    (Tools, Env, ToolsAllowed, ToolsDisallowed) so a future caller
    mutating `resolved.Tools` cannot leak into the rawBinding the
    template loader cached. Nil input → nil output (preserves the "no
    override" identity slice).
  - F.7.17 locked decision L15 default-to-claude: when
    `rawBinding.CLIKind == ""` and no override is plumbed, the resolved
    `CLIKind` is set to `CLIKindClaude`.
  - `CommitAgent` (string on `templates.AgentBinding`, *string on
    `BindingResolved`): empty string promotes to `nil` so adapters
    distinguish "no commit agent configured" from "explicit empty".
    Non-empty string promotes to a pointer to its copy.
  - `BlockedRetryCooldown` conversion: `templates.Duration` (the TOML
    text-marshaled wrapper) is converted to `time.Duration` before
    feeding the resolver helper, so the resolved `*time.Duration` field
    on `BindingResolved` carries the canonical stdlib type.

- `internal/app/dispatcher/binding_resolved_test.go` (NEW)
  - `rawBindingFixture()` — populated `templates.AgentBinding` with
    every scalar non-zero so the "no override" fall-through path is
    observable at every field.
  - Eight scenario tests mapping 1:1 to the spawn prompt's acceptance
    criteria:
    1. `TestResolveBindingNoOverrides` — rawBinding values pass through;
       CLIKind = "claude" preserved (no spurious default substitution).
    2. `TestResolveBindingSingleLayerOverride` — a single
       `*BindingOverrides` with Model = "haiku" wins over rawBinding's
       "opus"; untouched fields fall through.
    3. `TestResolveBindingMultiLayerPriority` — three layers each set
       Model differently; the highest-priority (first) layer wins.
    4. `TestResolveBindingMixedFieldOverrides` — highest sets Model
       only, lowest sets MaxTurns only; both land independently;
       unmatched fields fall through to raw.
    5. `TestResolveBindingNilLayerSkipped` — `[nil, override, nil]`
       skips nils without panic; override's values land.
    6. `TestResolveBindingEmptyOverridesSlice` — explicit empty slice
       passed via variadic `...` yields raw values unchanged.
    7. `TestResolveBindingCLIKindExplicit` — rawBinding.CLIKind =
       "claude" passes through (no spurious default substitution).
    8. `TestResolveBindingCLIKindDefaultsToClaude` — rawBinding.CLIKind
       = "" + no overrides → resolved.CLIKind = "claude" via L15
       default substitution. (Replaces the spawn-prompt's withdrawn
       "CLIKind override" scenario — `BindingOverrides` intentionally
       has no CLIKind field today; the spawn prompt notes that.)
  - Three additional defensive tests:
    - `TestResolveBindingCommitAgentEmptyToNil` — empty string raw
      promotes to nil pointer (zero-distinction guard).
    - `TestResolveBindingPointerOverridesPreservedAsCopy` — caller
      mutating the override-side pointer's underlying value AFTER
      ResolveBinding returns must NOT leak into the resolved struct
      (defensive-copy invariant).
    - `TestResolveBindingDurationOverride` — exercises the
      `*time.Duration` cascade path explicitly (override value lands).
    - `TestResolveBindingPureFunctionDoesNotMutateRaw` — mutating
      `resolved.Tools[0]` MUST NOT bleed back into the rawBinding's
      slice (slice-defensive-copy invariant).

- `workflow/drop_4c/4c_F7_17_8_BUILDER_WORKLOG.md` (NEW — this file)

## Pre-existing dirty state (NOT introduced by this droplet)

Working tree at droplet start carries five sibling-droplet WIP files
(`internal/domain/project.go`, `internal/domain/project_test.go`,
`internal/templates/builtin/default.toml`,
`internal/templates/embed_test.go`, `workflow/drop_4c/SKETCH.md`). The
dirty `internal/domain` build error and the
`TestProjectSchemaCoverageIsExplicit` failure under `internal/tui` are
caused by this WIP — `DispatcherCommitEnabled` (an unrelated
sibling-droplet field on `ProjectMetadata`) is not classified for TUI
schema coverage, and the test rejects undocumented project fields. None
of these files are in this droplet's edit scope; the orchestrator owns
the cleanup path.

## Verification

- `mage testPkg ./internal/app/dispatcher/` — **188/188 PASS** (was 175
  before this droplet; +13 new tests = 8 spawn-spec scenarios + 5
  defensive guards). 4.25s with race detection.
- `mage testPkg ./internal/templates/` — **355/355 PASS** (no
  regression — droplet only consumes `templates.AgentBinding` shape).
- `mage formatCheck` — **clean**, no formatting drift.
- `mage build` — **green**, binary compiles.
- `mage ci` — **PASSES on every package this droplet touches** (the two
  failures are pre-existing sibling-droplet WIP outside this droplet's
  scope; see "Pre-existing dirty state" above).

## Acceptance criteria checklist

- [x] `ResolveBinding(rawBinding, overrides...)` exported, pure-function,
      returns `BindingResolved`. Asserted by every test's call shape.
- [x] `BindingOverrides` struct with pointer fields per priority cascade
      (Model, Effort, MaxTries, MaxBudgetUSD, MaxTurns, AutoPush,
      BlockedRetries, BlockedRetryCooldown).
- [x] CLIKind defaults to `CLIKindClaude` when raw empty + no override.
      Asserted by `TestResolveBindingCLIKindDefaultsToClaude`.
- [x] Multi-layer priority test asserts highest-non-nil wins. Asserted
      by `TestResolveBindingMultiLayerPriority`.
- [x] All 8 spawn-prompt test scenarios pass (plus 4 additional
      defensive scenarios).
- [x] `mage ci` green on the droplet's surface (dispatcher 188/188,
      templates 355/355, formatting clean). Pre-existing
      `internal/domain` + `internal/tui` failures are sibling-droplet
      WIP, NOT introduced by this droplet.
- [x] Worklog written.
- [x] **NO commit by builder** — orchestrator drives commits after the
      QA pair returns green.

## Conventional commit message (for orchestrator post-QA)

```
feat(dispatcher): add ResolveBinding priority cascade resolver
```

(56 chars; conventional-commit single-line; describes the new pure
function without claiming the dispatcher-wiring work that lands in
follow-on droplet F.7.17.5.)
