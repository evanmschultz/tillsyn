# F.7.5c Builder Worklog — settings.json grant injection (RETRY)

## Goal

Per F.7-CORE F.7.5c spawn prompt: when rendering the per-spawn
`settings.json`, inject permission grants previously stored via F.7.5b's
TUI handshake. Per spawn, read grants for `(project_id, kind, cli_kind)`
from F.7.17.7's `permission_grants` storage and merge them into
`permissions.allow`.

## Conventional commit message (orchestrator commits — DO NOT COMMIT here)

```
feat(render): merge stored permission grants into spawn settings.json
```

## What shipped

### Render package (`internal/app/dispatcher/cli_claude/render/`)

- **`render.go`**:
  - Added `context` import.
  - Added `PermissionGrantsLister` narrower interface — read-only view
    of `app.PermissionGrantsStore.ListGrantsForKind`. Structural typing
    means production wiring passes the full `app.PermissionGrantsStore`
    and Go satisfies this interface implicitly. nil-tolerant by
    contract.
  - Added `ErrInvalidGrantsLister` sentinel for the dispatcher-seam
    type-assertion failure path.
  - **`Render` signature change**: now
    `Render(ctx, bundle, item, project, binding, grantsLister) (string, error)`.
    `ctx` forwarded to `grantsLister.ListGrantsForKind` for cancellation;
    `grantsLister` may be nil (graceful skip → binding.ToolsAllowed
    only). Path-separator + empty-input validation unchanged.
  - **`renderSettings` signature change**: now accepts ctx + item +
    project + lister; merges grants into `permissions.allow` via the
    new `mergeAllowList` helper.
  - **`mergeAllowList` (new pure helper)**: combines
    `binding.ToolsAllowed` first (preserve order, dedup via map) then
    appends grants from the lister (dedup against running set). Skip
    conditions: nil lister OR empty `binding.CLIKind` (storage UNIQUE
    composite forbids "" cli_kind). Lister errors propagate wrapped
    with the kind + cliKind context.

- **`init.go`**:
  - **`adaptRender` (new)**: bridges the dispatcher-side
    `BundleRenderFunc` (which uses `any` for the lister, since
    dispatcher must not import render to name the interface) to the
    render-side typed `PermissionGrantsLister`. nil → nil; non-nil
    that doesn't satisfy the interface → `ErrInvalidGrantsLister`
    (clean error, not panic). Registers `adaptRender` rather than
    `Render` directly.

### Dispatcher package (`internal/app/dispatcher/`)

- **`spawn.go`**:
  - `BundleRenderFunc` type signature extended with
    `ctx context.Context` (first param) and `grantsLister any` (last
    param). Doc comment expanded to explain the `any` choice and
    nil-graceful-skip contract.
  - `BuildSpawnCommand` callsite updated:
    `render(context.Background(), bundle, item, project, resolved, nil)`.
    nil lister documents the deferred plumbing — production wiring of
    `app.PermissionGrantsStore` lands in a follow-up droplet that adds
    a service-locator field to `dispatcher.AuthBundle` (or the Wave-3
    successor). New `TODO(F.7-CORE)` block in the call site captures
    this.

- **`spawn_test.go`**:
  - Faulty hook in `TestBuildSpawnCommandRenderHookFailureCleansUpBundle`
    updated to the new 6-arg signature (`ctx`, bundle, item, project,
    binding, grantsLister-as-any). t.Cleanup restoration uses an
    inline adapter wrapping `clauderender.Render` so the production
    wiring is restored correctly after the failure-injection test.

### Render tests (`internal/app/dispatcher/cli_claude/render/render_test.go`)

- All 14 existing tests updated to the new
  `Render(context.Background(), bundle, item, project, binding, nil)`
  signature.
- Added `stubGrantsLister` fixture — minimal in-memory
  `PermissionGrantsLister` stub that records the lookup tuple
  (projectID, kind, cliKind) and returns canned grants/error.
- Added `readSettingsAllow` helper + `grantFixture` helper for the
  new tests.
- **6 new tests** for the F.7.5c contract:
  1. `TestRenderSettingsNilListerSkipsGrantsLookup` — nil lister →
     binding.ToolsAllowed only, no error.
  2. `TestRenderSettingsListerZeroGrantsLeavesBindingOnly` — non-nil
     lister returning empty slice → binding only, but lister IS
     called (dispatch tuple verified).
  3. `TestRenderSettingsListerThreeGrantsAppendedAfterBinding` —
     happy path: 3 grants merged in lister-supplied order after
     binding entries.
  4. `TestRenderSettingsListerDuplicateRuleDeduped` — grants whose
     `Rule` matches a binding entry are dropped (binding position
     preserved).
  5. `TestRenderSettingsListerErrorWrapsAndRollsBack` — lister
     `ListGrantsForKind` error wrapped via `errors.Is` AND triggers
     full rollback of system-prompt.md + plugin/ subtree.
  6. `TestRenderSettingsEmptyCLIKindSkipsLookup` — empty
     `binding.CLIKind` short-circuits before the lister call (no
     lister invocation).

## Verification

- `mage check` — green. 2650 tests pass, 1 skipped (pre-existing
  `TestStewardIntegrationDropOrchSupersedeRejected`), 0 failed.
- `mage ci` — green. Coverage threshold met across all 24 packages.
  Render package coverage: **84.7%** (above the 70% floor).
  Dispatcher package coverage: 75.1%.
- All file edits via `Edit`/`Write`. No raw `go test` / `go build` /
  `gofmt` shells. Hylla not consulted (per spawn-prompt directive).

## Acceptance criteria (per spawn prompt)

- [x] `PermissionGrantsLister` narrower interface defined in render
      package.
- [x] `Render` signature accepts the lister (nil-tolerant).
- [x] `renderSettings` merges grants with deduplication; binding
      entries first then grants.
- [x] All 4-5 test scenarios pass (delivered 6: zero-grants,
      three-grants, duplicate-dedup, lister-error,
      empty-CLIKind-skip, plus the nil-lister case).
- [x] `BundleRenderFunc` type updated to match new `Render`
      signature.
- [x] spawn.go passes nil lister to hook (graceful skip), documented
      as deferred wiring (worklog + inline `TODO(F.7-CORE)` comment
      block).
- [x] `mage check` + `mage ci` green.
- [x] **NO commit by builder.** Orchestrator commits after QA pair
      returns green.

## Deferred plumbing (handoff to follow-up droplet)

`BuildSpawnCommand` passes `nil` for the grants lister today. The
production wiring path needs:

1. A handle on `app.PermissionGrantsStore` (or a Repository façade
   exposing it) reaching `BuildSpawnCommand`.
2. A new field on `dispatcher.AuthBundle` (or the Wave-3 successor
   struct) carrying that handle as `any` so the dispatcher stays
   cycle-free of `app`.
3. Substitute `nil` → `authBundle.PermissionGrantsLister` at the
   `render(...)` callsite.

The render seam is ready; only the dispatcher-level dependency
plumbing is outstanding. This work is explicitly out of F.7.5c
scope per the spawn prompt ("the spawn pipeline can pass a nil
lister (graceful) since dispatcher hasn't yet plumbed the storage
handle through. Document this as deferred wiring in the worklog.").

## Hylla Feedback

N/A — task touched non-Go-search-relevant areas only and the
spawn-prompt explicitly forbade Hylla calls ("NO Hylla calls.").
All evidence-gathering used Read / Grep over local files within
the four named target paths. Existing interface signatures verified
via `Read` of `internal/app/permission_grants_store.go` and
`internal/domain/permission_grant.go`.
