# F.7-CORE F.7.14 — Push Gate (Builder Worklog)

## What landed

Completion-builder pass: prior dispatch hit API stream idle timeout AFTER writing `internal/app/dispatcher/gate_push.go` (the gate file compiles clean). This pass closed the remaining work.

Edited / created files (all NEW except as noted):

- **NEW**: `internal/app/dispatcher/gate_push_test.go` — full table-aligned unit-test suite for `PushGateRunner.Run`. 11 tests total (the 6 prompt-mandated scenarios plus 5 defense-in-depth adjacents matching `gate_commit_test.go`'s pattern).
- **EDITED**: `internal/templates/schema.go` — added `GateKindPush GateKind = "push"` constant + appended `GateKindPush` to the `validGateKinds` slice. Doc-comment cross-references the new gate implementation.
- **EDITED**: `internal/templates/schema_test.go` — moved `GateKind("push")` from invalid-cases to valid-cases in `TestGateKindClosedEnum`. Updated the test's leading doc-comment to reflect F.7.14's enum-promotion rationale.

NO edits to `gate_push.go` itself — read-only review confirmed the file's exported surface, sentinels, and documented contract are coherent and align with the F.7.14 spec / `gate_commit.go` symmetry. Treating any rewrite as out of scope per orchestrator hard-constraints.

## Test-file scenarios (`gate_push_test.go`)

The 6 prompt-mandated scenarios + 5 defense-in-depth adjacents:

1. **`TestPushGateRunHappyPath`** — toggle on, branch resolves to `"drop/4c"`, `GitPush` succeeds → `Run` returns nil and the `(repoPath, branch)` tuple flows verbatim from `project.RepoPrimaryWorktree` + `GitCurrentBranch`'s return into `GitPush`.
2. **`TestPushGateRunToggleOff`** — `IsDispatcherPushEnabled() == false` (nil pointer, default state) short-circuits to `nil`. Neither git seam fires. Asserts the symmetry-only `ErrPushGateDisabled` sentinel does NOT slip onto the no-op path (per `gate_push.go`'s docstring contract).
3. **`TestPushGateRunToggleExplicitFalse`** — `*bool=false` (vs nil) is treated identically to nil per the three-state pointer-bool design. No git seams fire.
4. **`TestPushGateRunPushFails`** — toggle on, branch resolves, `GitPush` returns a synthetic error → wrapped via `ErrPushGatePushFailed`; both `errors.Is(err, ErrPushGatePushFailed)` and `errors.Is(err, pushErr)` reachable per the `fmt.Errorf("%w: %w")` shape in `gate_push.go:221`.
5. **`TestPushGateRunBranchMissingEmpty`** — `GitCurrentBranch` returns `""` → `ErrPushGateBranchMissing`. `GitPush` MUST NOT fire.
6. **`TestPushGateRunBranchMissingError`** — `GitCurrentBranch` returns a non-nil error → `ErrPushGateBranchMissing` AND the underlying error reachable via `errors.Is`. Two-shape collapse to one sentinel exercised.
7. **`TestPushGateRunNilReceiver`** — `var runner *PushGateRunner; runner.Run(...)` → loud error, no panic.
8. **`TestPushGateRunNilItem`** — nil `*domain.ActionItem` → loud error, no git seams fired.
9. **`TestPushGateRunNilGitCurrentBranchField`** — runner with `GitCurrentBranch: nil` → loud error after the toggle guard. `GitPush` does not fire.
10. **`TestPushGateRunNilGitPushField`** — runner with `GitPush: nil` after `GitCurrentBranch` resolves → loud error. Branch resolution counted exactly once.
11. **`TestGateKindPushRegistered`** — cross-package belt-and-suspenders: `templates.IsValidGateKind(templates.GateKindPush)` is true and `string(templates.GateKindPush) == "push"`.

## Acceptance checklist

- [x] Tests align with actual `gate_push.go` API (read-confirmed). Sentinels asserted by exact name: `ErrPushGatePushFailed`, `ErrPushGateBranchMissing`, `ErrPushGateDisabled`.
- [x] All 6 prompt-mandated scenarios pass (verified via `mage ci` + targeted `mage testFunc TestPushGateRunHappyPath`). 5 adjacents pass alongside.
- [x] `GateKindPush` registered in closed enum (`schema.go`). Constant + slice entry added.
- [x] `schema_test.go` has `"push"` in valid cases (moved from invalid). Doc-comment updated.
- [x] `mage check` green: 2683 passed / 1 skipped (pre-existing) / 0 failed across 24 packages. Coverage 70%+ on every package; templates 97.0%, dispatcher 72.7%.
- [x] `mage ci` green: same shape (Sources → Formatting → Coverage → Build → Build).
- [x] Worklog written (this file).
- [x] **NO commit by builder** — per F.7-CORE REV-13 + orchestrator hard-constraint.

## Hard-constraints honored

- DO NOT commit — confirmed: no `git commit` invoked.
- Edited only `gate_push_test.go` (NEW), `schema.go`, `schema_test.go`, and the worklog. No rewrite of `gate_push.go`.
- `mage check` + `mage ci` (NEVER `mage install`) — confirmed both ran green.
- NO Hylla calls — confirmed.

## Suggested commit message (orchestrator runs `git commit`, not builder)

Single-line conventional, ≤72 chars:

```
test(dispatcher): add F.7.14 push gate tests + register GateKindPush
```

Character count: 67. Within the ≤72-char gate.

## mage ci status

```
Sources    [SUCCESS]
Formatting [SUCCESS]
Coverage   [SUCCESS] (70.0% min met; 24/24 packages above gate)
Build      [SUCCESS]
```

2683 passed / 1 skipped (pre-existing `TestStewardIntegrationDropOrchSupersedeRejected`) / 0 failed.

## Notes for QA pairs

- **Sentinel polarity**: `ErrPushGateDisabled` exists per `gate_push.go:46-55` as a future-safe label only. The toggle-off path returns `nil`, NOT this sentinel. `TestPushGateRunToggleOff` asserts the sentinel is NOT reachable on the no-op path.
- **Two-shape branch-missing collapse**: per `gate_push.go:206-212`, both empty-string return AND non-nil error from `GitCurrentBranch` collapse to `ErrPushGateBranchMissing`. Both shapes are pinned by separate tests (5, 6 above).
- **Idempotency / mutation**: gate does NOT mutate `item` (no per-action-item field updated by the push gate; this is symmetric-but-different from `CommitGateRunner` which sets `item.EndCommit`). Tests do not assert mutation absence beyond the toggle-off path because `gate_push.go` simply has no field to mutate.
- **catalog + auth params**: inert for this gate (per `gate_push.go:175-178`). Tests pass `templates.KindCatalog{}` + `AuthBundle{}` zero values — the gate's algorithm never reads either.

## Hylla Feedback

N/A — task touched non-Go files only via worklog; the Go work was scoped via `Read` + `rg` against the active checkout. No Hylla queries were issued for this completion-builder pass per the orchestrator's "NO Hylla calls" hard-constraint.
