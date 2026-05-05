# F.7.5c Builder QA Falsification — Round 1

## Verdict

**PASS-WITH-NITS** — No CONFIRMED counterexample against the F.7.5c claim
as currently shipped (spawn.go passes literal `nil`; production wiring is
deferred). One latent-panic risk (typed-nil pointer satisfying
`PermissionGrantsLister`) is REFUTED today because no caller passes a
typed-nil; it is recorded as an explicit NIT against the deferred
plumbing seam so the follow-up droplet does not silently re-introduce
it. Two cosmetic NITs in the failure-injection retrofit + V3 branch
table.

## Scope

Round 1 adversarial review of F.7.5c. Read-only. No source edits. Sister
artifact: `4c_F7_5c_BUILDER_QA_PROOF.md` (PASS, dated round 1).

## Files reviewed

- `internal/app/dispatcher/cli_claude/render/render.go` (525 lines)
- `internal/app/dispatcher/cli_claude/render/init.go` (62 lines)
- `internal/app/dispatcher/cli_claude/render/render_test.go` (774 lines)
- `internal/app/dispatcher/spawn.go` (lines 99–166, 437–467)
- `internal/app/dispatcher/spawn_test.go` (lines 677–751)
- `internal/app/dispatcher/binding_resolved.go` (L116–153 — ResolveBinding)
- `internal/app/dispatcher/cli_adapter.go` (L29–49 — CLIKind enum)
- `internal/app/permission_grants_store.go` (interface contract)
- `internal/adapters/storage/sqlite/permission_grants_repo.go`
  (storage ordering / case folding)

## Attack-by-attack verdicts

### A1. Typed-nil pointer satisfying interface (latent panic)

**REFUTED today, NIT against deferred plumbing.**

The interface-with-typed-nil trap exists in two places:

1. `init.go` L52–59 `adaptRender`. Path:
   - `grantsLister any` arrives carrying `(*FakeLister)(nil)`.
   - L53 `if grantsLister != nil` evaluates **TRUE** (interface has a
     non-nil dynamic type even with a nil dynamic value).
   - L54 `typed, ok := grantsLister.(PermissionGrantsLister)` succeeds
     when the typed-nil's pointer-type implements the interface — `ok`
     is true; `typed` is a typed-nil interface value.
   - L58 `lister = typed` → propagates downstream.

2. `render.go` L525 `mergeAllowList`. Path:
   - `if grantsLister == nil || …` evaluates **FALSE** for a typed-nil.
   - L529 `grantsLister.ListGrantsForKind(...)` invokes a method on a nil
     pointer receiver. Production `*sqlite.Repository.ListGrantsForKind`
     dereferences `r.db` (`permission_grants_repo.go` L68), causing a
     `runtime.Error` nil-pointer-dereference panic.

**Why REFUTED today:** spawn.go L463 passes literal `nil` (untyped). The
`any` parameter receives a true nil interface, not a typed-nil. Both
guards work correctly for that call. No production code path constructs
a typed-nil today.

**Why this matters:** the worklog at L134–151 describes deferred plumbing
that adds an `authBundle.PermissionGrantsLister any` field. If a future
builder writes `authBundle.PermissionGrantsLister = (*sqlite.Repository)(nil)`
(plausible during partial-bootstrap initialization where the repo handle
isn't yet populated), the panic will fire on first spawn. No test today
covers a typed-nil lister. The classic remedy is a `reflect.ValueOf(x).IsNil()`
guard inside `adaptRender` for pointer / interface / map / slice / chan /
func kinds, OR a discipline rule on the deferred plumbing site that the
field is set with a true nil interface until populated.

NIT — recorded so the deferred-plumbing follow-up droplet adds either a
typed-nil-defense test OR a reflect-based guard at `adaptRender`.

### A2. Empty CLIKind short-circuit violating L11/L15 default-to-claude

**REFUTED.**

`ResolveBinding` (`binding_resolved.go` L129–131) ALWAYS substitutes
`CLIKindClaude` when `rawBinding.CLIKind == ""`. So in production, a
`BindingResolved` reaching `mergeAllowList` cannot have an empty CLIKind
unless a unit test constructs the struct directly. The
`TestRenderSettingsEmptyCLIKindSkipsLookup` test at L737–758 is exactly
that — manual construction with a corrupted `CLIKind = ""` to exercise
the defensive short-circuit. The contract is preserved; the
short-circuit guards a path that would otherwise hit the storage layer's
`domain.ErrInvalidID` rejection (`permission_grants_repo.go` L64–66).

PASS.

### A3. Grant ordering non-determinism

**REFUTED.**

Storage query in `permission_grants_repo.go` L68–73 uses
`ORDER BY granted_at ASC, id ASC` — fully deterministic on both keys.
`mergeAllowList` (L508–541) preserves the lister's slice order (range
loop, append-only, no sort). The
`TestRenderSettingsListerThreeGrantsAppendedAfterBinding` test pins the
exact 5-element order (binding ["Read", "Grep"] + 3 grants in lister
order). PASS.

### A4. Cycle-break soundness vs F.7.4 monitor

**REFUTED.**

The cycle-break shape: `dispatcher → cli_claude → render → dispatcher`
is NOT a cycle because `dispatcher` does NOT import `cli_claude` or
`render` directly. The only flow is:
- `dispatcher/cli_claude/init.go` blank-imports `dispatcher/cli_claude/render`
  (`init.go` L11).
- `dispatcher/cli_claude/render/init.go` (this droplet) calls
  `dispatcher.RegisterBundleRenderFunc(adaptRender)` at L32.
- The dispatcher discovers the hook via `lookupBundleRenderFunc`
  (`spawn.go` L169–176).

Pattern mirrors `RegisterAdapter` / `lookupAdapter` (`spawn.go` L263,
L272). F.7.4 monitor lives inside the dispatcher package itself, so it
needed no cycle-break — comparison is irrelevant. PASS.

### A5. ErrInvalidGrantsLister + lister-error propagation

**REFUTED.**

Wrapping chain on lister error:
- `mergeAllowList` L530–532:
  `fmt.Errorf("list grants for kind %q cli %q: %w", ...)`.
- `renderSettings` returns it directly (L477).
- `Render` wraps: `fmt.Errorf("render: settings: %w", err)` (L175).
- `BuildSpawnCommand` wraps:
  `fmt.Errorf("dispatcher: render spawn bundle: %w", err)` (L466).

`errors.Is(err, listerErr)` traverses all four wraps unchanged. The
`TestRenderSettingsListerErrorWrapsAndRollsBack` test (L707–732) pins
`errors.Is(err, listerErr)` AND verifies the bundle rollback wiped both
`system-prompt.md` and the `plugin/` subtree. `ErrInvalidGrantsLister`
returned from `adaptRender` likewise propagates through the same chain.
PASS.

### A6. Memory-rule conflicts (no Hylla / no commit / no migration)

**REFUTED.**

- Worklog L155–161 confirms NO Hylla calls.
- Worklog L11–15 explicitly forbids commit by the builder; the
  conventional commit message is staged for orchestrator-side commit.
- No migration logic introduced — the storage layer's
  `permission_grants` table was landed in F.7.17.7; F.7.5c is a
  read-side consumer.

PASS.

### A7. Description drift / silent re-interpretation

**REFUTED.**

Spawn-prompt acceptance criteria match worklog L115–132 1:1:
`PermissionGrantsLister`, `Render` widening, `renderSettings` merge,
4–5 test scenarios (delivered 6), `BundleRenderFunc` signature update,
spawn.go nil-passing, `mage check` + `mage ci` green. No symbol
drift. PASS.

### A8. File / package gating

**REFUTED.**

Edits confined to:
- `internal/app/dispatcher/cli_claude/render/{init,render,render_test}.go`
- `internal/app/dispatcher/{spawn,spawn_test}.go`

These match the spawn-prompt's declared paths exactly. No edits to
`internal/app/permission_grants_store.go`, `internal/domain/permission_grant.go`,
`internal/adapters/storage/sqlite/permission_grants_repo.go` (read for
contract verification only). PASS.

### A9. Failure-injection retrofit hygiene

**NIT — REFUTED as bug, accepted as cosmetic debt.**

`spawn_test.go` L715–732 inlines a duplicate of `adaptRender`'s
`any → PermissionGrantsLister` adapter to restore the production hook
because `adaptRender` is unexported. The duplication is functionally
correct (matches `init.go` L52–59 line-for-line) but is a maintenance
hazard: a future signature change to `adaptRender` will require a
mirror edit in this `t.Cleanup`.

NIT options:
- Export `adaptRender` (or a `RestoreProductionHook` helper) from the
  render package.
- Add a renderer registry in the test fixture itself.

Not blocking — failure-injection-restore is a narrow seam.

### A10. V3 branch-table completeness in QA Proof

**NIT — meta finding.**

The sibling QA Proof's V3 table at L61–66 enumerates three branches:
untyped nil, non-nil-no-interface, non-nil-with-interface. It does NOT
enumerate typed-nil-with-interface (attack A1). The QA Proof's PASS
verdict on V3 is correct against the SHIPPED behavior; the omission is
a documentation gap, not a functional bug.

NIT — sibling QA Proof should add a fourth row noting typed-nil yields
"typed-nil propagation; latent panic deferred to follow-up plumbing."

### A11. Race on RegisterBundleRenderFunc + run order

**REFUTED.**

`renderMu sync.RWMutex` (`spawn.go` L144) guards both the register and
lookup paths. The failure-injection test
`TestBuildSpawnCommandRenderHookFailureCleansUpBundle` is correctly NOT
`t.Parallel()` (comment at L684–686 explains the gating). Other tests
that call `BuildSpawnCommand` rely on the production hook being
registered by the package-import init order. Standard Go init ordering
guarantees `cli_claude/render.init() → cli_claude.init() →
spawn_test.go's blank import` resolves before any `Test*` runs. PASS.

## Counterexamples

None CONFIRMED.

## Summary

PASS-WITH-NITS. The F.7.5c implementation correctly widens `Render`
with ctx + grants-lister, breaks the dispatcher-render cycle via a
last-writer-wins hook registry mirroring `RegisterAdapter`, and ships
6 new tests + 14 retrofits with deterministic ordering, dedup, error
wrapping, and rollback. The latent typed-nil panic (A1) is not
reachable today because production passes literal nil; recording it as
a NIT against the deferred plumbing seam is sufficient. The
failure-injection cleanup adapter (A9) is a maintenance NIT, not a
bug. The QA Proof's V3 branch table (A10) is incomplete but its
verdict is correct.

Recommend the orchestrator commit the build, route NITs A1+A9+A10 to
the follow-up droplet that plumbs the production
`app.PermissionGrantsStore` handle through `BuildSpawnCommand`.

## Hylla Feedback

N/A — task touched non-Go-search-relevant areas only and the
spawn-prompt explicitly forbade Hylla calls. All evidence-gathering
used `Read` over local files within the four named target paths plus
the contract files (`permission_grants_store.go`, `permission_grants_repo.go`,
`binding_resolved.go`, `cli_adapter.go`).
