# DROP_4A — WAVE 2 — DISPATCHER LOOP

**State:** planning
**Wave:** 2 of 5 (Dev Hygiene → Domain Fields → **Dispatcher Loop** → Auth Integration → Closeout)
**Wave depends on:** Wave 1 (`paths`/`packages` first-class on `ActionItem`; project-node `language` / `repo_primary_worktree` / `hylla_artifact_ref` / `dev_mcp_server_name` first-class on `Project`).
**Wave feeds:** Wave 3 (auth bundles consumed by spawn step), Wave 4 (closeout doc updates referencing dispatcher behavior).
**Brief ref:** `workflow/drop_4a/REVISION_BRIEF.md` § Wave 2 (lines 42–54).
**PLAN ref:** `main/PLAN.md` § 19.4 (Drop 4 — Dispatcher Core).
**Started:** 2026-05-03
**Closed:** —

## Wave Purpose

Replace the orchestrator-as-dispatcher loop with a programmatic dispatcher in a new `internal/app/dispatcher/` package. Wave 2 delivers the **manual-trigger dispatcher** milestone — the dispatcher takes over agent spawn, lock management, auto-promotion, conflict detection, process monitoring, and cleanup-on-terminal-state, but is fired from a CLI command rather than auto-running in the server process. Auto-spawn-on-state-change wiring + post-build gates land in Drop 4b.

## Wave Architecture

The dispatcher is a single new package `internal/app/dispatcher/` split into focused files to minimize same-file edit pressure during the build phase. Consumer-side: `Service` exposes a `Dispatcher` interface (constructor + `RunOnce(ctx, actionItemID)` + `Start`/`Stop` for the future continuous mode). Implementation: composes a `LiveWaitBroker` subscriber, two `lockManager`s (file + package), a tree walker, a spawner, a conflict detector, a process monitor, and a cleanup hook.

File layout decision (per atomicity rule + same-package contention attack):

| File                                       | Responsibility                                    | Lands in droplet |
| ------------------------------------------ | ------------------------------------------------- | ---------------- |
| `internal/app/dispatcher/dispatcher.go`    | Type + interface + constructor + `RunOnce` shell  | 2.1              |
| `internal/app/dispatcher/broker_sub.go`    | LiveWaitBroker subscription + event filter        | 2.2              |
| `internal/app/dispatcher/locks_file.go`    | File-level `paths` lock manager                   | 2.3              |
| `internal/app/dispatcher/locks_package.go` | Package-level `packages` lock manager             | 2.4              |
| `internal/app/dispatcher/walker.go`        | Tree walker + auto-promotion eligibility          | 2.5              |
| `internal/app/dispatcher/spawn.go`         | Template-binding lookup + agent spawn invocation  | 2.6              |
| `internal/app/dispatcher/conflict.go`      | Sibling overlap → runtime `blocked_by` insertion  | 2.7              |
| `internal/app/dispatcher/monitor.go`       | Process tracking + crash detection                | 2.8              |
| `internal/app/dispatcher/cleanup.go`       | Terminal-state cleanup (locks, leases, subs)      | 2.9              |
| `cmd/till/dispatcher_cli.go`               | `till dispatcher run` CLI wiring                  | 2.10             |

All ten files share the `internal/app/dispatcher` package (or `cmd/till` for the CLI), so **package-level lock contention forces serial build**: every droplet after 2.1 carries `blocked_by 2.1` plus the specific upstream droplet whose APIs it consumes.

## Wave-Internal Sequencing

Linear chain enforced by package-lock + file-lock blockers:

```
2.1 (package skeleton)
 └─→ 2.2 (broker sub)
      └─→ 2.5 (walker)  ←─┐
 └─→ 2.3 (file locks)     │
      └─→ 2.4 (pkg locks) │
           └─→ 2.7 (conflict detector — needs walker + both lock managers)
 └─→ 2.6 (spawn) ──────────────→ 2.8 (monitor — needs spawn handle)
 └─→ 2.9 (cleanup) — blocked on 2.3 + 2.4 + 2.6 (touches all release paths)
 └─→ 2.10 (CLI) — orchestrates RunOnce; blocked on 2.5 + 2.6 + 2.7 + 2.8 + 2.9
```

Three groups can run in parallel after 2.1: `{2.2 → 2.5}`, `{2.3 → 2.4}`, `{2.6 → 2.8}`. They reconverge at 2.7 (needs walker + locks), 2.9 (needs locks + spawn), and 2.10 (needs everything).

## Cross-Wave Blockers

Every Wave 2 droplet is `blocked_by` Wave 1's domain-field landings. Concrete cross-wave edges:

- **`paths`/`packages` on `ActionItem`** (Wave 1) blocks Wave 2 droplets 2.3, 2.4, 2.5, 2.7 — the lock managers, walker, and conflict detector all read these fields.
- **Project-node fields** (`repo_primary_worktree`, `language`, `hylla_artifact_ref`, `dev_mcp_server_name` on Wave 1) block Wave 2 droplet 2.6 — spawn step `cd`s to `repo_primary_worktree`, picks `{language}-builder-agent` variant, injects `hylla_artifact_ref` into prompt, registers under `dev_mcp_server_name`.
- **`state` on MCP create+move** (Wave 1) blocks Wave 2 droplet 2.5 — walker promotes to `in_progress` via `state="in_progress"`.
- **Always-on parent-blocks-on-failed-child** (Wave 1) blocks Wave 2 droplet 2.5 — walker eligibility check relies on the unconditional invariant.

These cross-wave edges are surfaced per droplet below in the **Blocked by** field.

## Verification Targets

- Per droplet: `mage test-pkg internal/app/dispatcher` (or `cmd/till` for 2.10) and `mage test-func internal/app/dispatcher <TestName>` for the specific scenarios listed.
- Wave-end: `mage ci` clean. Coverage on `internal/app/dispatcher` ≥ 70 %.
- **Never** `mage install`. **Never** raw `go test`/`go build`/`go vet`/`go run`.

---

## Droplet Decomposition

### Wave 2.1 — DISPATCHER PACKAGE SKELETON + INTERFACE

- **State:** todo
- **Paths:** `internal/app/dispatcher/dispatcher.go` (NEW), `internal/app/dispatcher/dispatcher_test.go` (NEW), `internal/app/dispatcher/doc.go` (NEW)
- **Packages:** `internal/app/dispatcher` (NEW)
- **Acceptance:**
  - New package compiles with `mage test-pkg internal/app/dispatcher` clean.
  - `Dispatcher` interface declared with three methods: `RunOnce(ctx context.Context, actionItemID string) (DispatchOutcome, error)`, `Start(ctx context.Context) error`, `Stop(ctx context.Context) error`. `Start`/`Stop` may return `ErrNotImplemented` in this droplet — Drop 4b wires them.
  - Concrete impl `dispatcher` struct holds a `*app.Service` reference, a `LiveWaitBroker`, and zero-value lock-manager / walker / spawner / monitor fields (filled in by later droplets).
  - `NewDispatcher(svc *app.Service, broker app.LiveWaitBroker, opts Options) (*dispatcher, error)` constructor validates `svc != nil` and `broker != nil`; returns wrapped `ErrInvalidDispatcherConfig` otherwise.
  - `DispatchOutcome` is a struct with at minimum: `ActionItemID string`, `AgentName string`, `SpawnedAt time.Time`, `Result Result` (closed enum `Spawned | Skipped | Blocked | Failed`).
  - Test file covers: constructor nil-arg rejection, `RunOnce` returns `Skipped` when given a non-existent action-item ID (via `Service.GetActionItem` returning the standard not-found error), `RunOnce` returns `Skipped` when action item is not in `todo`.
- **Blocked by:** none (entry droplet for Wave 2). Cross-wave: none required for the skeleton itself; later droplets add the cross-wave cuts.
- **Notes:**
  - This droplet establishes the package boundary. Every other Wave 2 droplet edits a different file *within* this package; the package-lock rule means each must `blocked_by 2.1` to avoid concurrent first-edits creating import-graph collisions. This is the canonical case the package-lock rule exists for.
  - The `Options` struct is forward-compatible: drops 2.6, 2.8, 2.10 each add fields. Document it as an open struct in 2.1's doc-comments so QA-falsification doesn't flag the "Options has only one field" attack.
  - YAGNI watch: do NOT add gate-runner, commit-agent, push, or reingest fields here — those are Drop 4b. If QA flags Drop 4b coupling, surface and reject.

### Wave 2.2 — LIVEWAITBROKER SUBSCRIPTION

- **State:** todo
- **Paths:** `internal/app/dispatcher/broker_sub.go` (NEW), `internal/app/dispatcher/broker_sub_test.go` (NEW), `internal/app/live_wait.go` (extend with new event constant), `internal/app/coordination_live_wait.go` (extend with new publish helper), `internal/app/service.go` (call new publisher from `MoveActionItem` + `CreateActionItem` + `UpdateActionItem`).
- **Packages:** `internal/app/dispatcher`, `internal/app`.
- **Acceptance:**
  - New `LiveWaitEventActionItemChanged` constant added to `internal/app/live_wait.go` alongside the four existing event types. Key format: `<projectID>` (project-scoped, mirrors `LiveWaitEventAttentionChanged`'s pattern).
  - New `Service.publishActionItemChanged(projectID string)` helper added to `internal/app/coordination_live_wait.go` (same shape as `publishAttentionChanged`).
  - `Service.MoveActionItem`, `Service.CreateActionItem`, `Service.UpdateActionItem` each call `s.publishActionItemChanged(actionItem.ProjectID)` after a successful repo write. Existing `MoveActionItem` test must still pass.
  - `dispatcher.subscribeBroker(ctx context.Context, projectID string) <-chan app.LiveWaitEvent` wires `Wait` polling into a goroutine that emits events on a channel; cancellation drains.
  - Test scenario `TestDispatcherSubscribesToActionItemChanges`: create dispatcher, publish a synthetic event, assert the subscriber channel receives it within 100 ms.
  - Test scenario `TestDispatcherStopsOnContextCancel`: cancel context, assert goroutine exits within 100 ms (no leak).
- **Blocked by:** 2.1 (package + interface). Cross-wave: none required — `LiveWaitEvent` already exists, this droplet just adds a new event type.
- **Notes:**
  - The existing broker contract (`Wait` blocks until a matching event) is single-shot — the dispatcher wraps it in a re-subscribe loop. Document the `afterSequence` cursor handling so the walker (2.5) reads consistent ordering.
  - Same-file edits in `internal/app/coordination_live_wait.go` and `internal/app/live_wait.go` are isolated additions (new const, new method) — no contention with concurrent same-package edits today, but Wave 1's `state` MCP work *might* touch `coordination_live_wait.go`. Surface to plan-QA as a soft cross-wave coordination concern.
  - Plan-QA falsification attack to address: "broker re-subscribe loop spins on Sequence=0 forever after broker close." Mitigation: document broker-close-yields-error contract, test exercises it.

### Wave 2.3 — FILE-LEVEL LOCK MANAGER

- **State:** todo
- **Paths:** `internal/app/dispatcher/locks_file.go` (NEW), `internal/app/dispatcher/locks_file_test.go` (NEW)
- **Packages:** `internal/app/dispatcher`
- **Acceptance:**
  - `fileLockManager` struct with `Acquire(actionItemID string, paths []string) (acquired []string, conflicts map[string]string, err error)` — returns the subset of paths now held + a map of conflicting paths → holding action-item ID.
  - `Release(actionItemID string)` releases all locks held by that action item.
  - In-process implementation: `sync.Mutex` + `map[string]string` (path → holding action-item ID). No SQLite persistence in this droplet — Drop 4b decides whether to persist.
  - Test scenarios:
    - `TestFileLockAcquireSinglePathSucceeds`
    - `TestFileLockAcquireSamePathTwiceByDifferentItemsConflicts`
    - `TestFileLockReleaseFreesAllPathsHeldByItem`
    - `TestFileLockAcquirePartialConflictReturnsConflicts` — paths `[a, b]` against existing holder of `b` returns `acquired=[a]`, `conflicts={b: <holder>}`.
    - `TestFileLockConcurrentAcquireRaceFree` (use `mage test-pkg` with `-race` via mage default).
  - 100% coverage on `locks_file.go` (small surface, fully testable).
- **Blocked by:** 2.1. Cross-wave: Wave 1 `paths []string` field on `ActionItem` (the lock manager doesn't read `ActionItem` directly — it takes a `[]string` argument — but its only consumer, the walker in 2.5, will read `actionItem.Paths`).
- **Notes:**
  - Plan-QA falsification attack: "what if `paths` contains relative paths and one item normalizes differently than another?" Mitigation: document that the manager treats paths as opaque strings; normalization is the *caller's* responsibility. Add a test asserting `[./a]` and `[a]` are treated as distinct keys.
  - Plan-QA falsification attack: "deadlock if Acquire is called from inside a Release callback." Mitigation: no callbacks in this droplet — flat synchronous API. Document.
  - Same-file lock between 2.3 and 2.4: NO — they're separate files. But same-package: yes; 2.4 must `blocked_by 2.3` (package compile lock).

### Wave 2.4 — PACKAGE-LEVEL LOCK MANAGER

- **State:** todo
- **Paths:** `internal/app/dispatcher/locks_package.go` (NEW), `internal/app/dispatcher/locks_package_test.go` (NEW)
- **Packages:** `internal/app/dispatcher`
- **Acceptance:**
  - `packageLockManager` struct mirroring `fileLockManager`'s shape: `Acquire(actionItemID string, packages []string) (acquired []string, conflicts map[string]string, err error)` + `Release(actionItemID string)`.
  - Same in-process `sync.Mutex` + `map[string]string` (package → holding action-item ID).
  - Test scenarios mirror 2.3:
    - `TestPackageLockAcquireSinglePackageSucceeds`
    - `TestPackageLockAcquireSamePackageTwiceConflicts`
    - `TestPackageLockReleaseFreesAllPackagesHeldByItem`
    - `TestPackageLockAcquirePartialConflictReturnsConflicts`
    - `TestPackageLockConcurrentAcquireRaceFree`
  - 100% coverage on `locks_package.go`.
- **Blocked by:** 2.1, 2.3 (package-lock — same `internal/app/dispatcher` compile unit).
- **Notes:**
  - The two managers are intentionally NOT a single generic `lockManager[K]` — package-lock and file-lock have different planned evolutions in Drop 4b (package-lock will become per-Go-package via `go list -json`; file-lock stays opaque). Premature generic lands as YAGNI.
  - Plan-QA falsification attack: "if file `internal/app/foo.go` is locked by item A and item B locks package `internal/app`, do they conflict?" Mitigation: NO — file-lock and package-lock are **two independent maps**. Cross-locking semantics live in the walker (2.5) and the conflict detector (2.7), where they are explicit. Document this in 2.4's doc-comment.
  - Plan-QA falsification attack: "the planner can declare paths covering a package without including the package — partial coverage." Mitigation: Wave 1's domain validation enforces "every file in `paths` maps to a package in `packages`" (per REVISION_BRIEF Wave 1 line 33). This droplet relies on that invariant; surface as a cross-wave dependency note.

### Wave 2.5 — TREE WALKER + AUTO-PROMOTION

- **State:** todo
- **Paths:** `internal/app/dispatcher/walker.go` (NEW), `internal/app/dispatcher/walker_test.go` (NEW)
- **Packages:** `internal/app/dispatcher`
- **Acceptance:**
  - `treeWalker` struct with `EligibleForPromotion(ctx context.Context, projectID string) ([]domain.ActionItem, error)` returning items in `todo` whose blockers are all clear (no incomplete children, no unmet `blocked_by`, parent in `in_progress` or root).
  - Eligibility predicate documented and tested case-by-case:
    1. Item is in `LifecycleState=todo`.
    2. Every entry in `actionItem.BlockedBy` resolves to an action item in `LifecycleState=complete`.
    3. Every child of the item is in `complete` — wait, **NOT** required for promotion *to* `in_progress`. (Children-complete is a precondition for promotion *to* `complete`, enforced by `ensureActionItemCompletionBlockersClear` already.) Document the asymmetry.
    4. Parent (if any) is in `in_progress` or has `Persistent=true`.
  - `Promote(ctx context.Context, item domain.ActionItem) error` resolves the `in_progress` column ID via the existing `lifecycleStateForColumnID` helper (per `Service.MoveActionItem` source), then calls `Service.MoveActionItem`. Handles `ErrTransitionBlocked` by returning a typed `ErrPromotionBlocked` wrapping the inner error.
  - Test scenarios:
    - `TestWalkerFindsTodoItemWithClearedBlockers`
    - `TestWalkerSkipsTodoItemWithUnmetBlockedBy`
    - `TestWalkerSkipsTodoItemWhoseParentIsTodo`
    - `TestWalkerPromotesEligibleItem` — uses an in-memory `Service` fake; asserts `MoveActionItem` was called with the correct column ID.
    - `TestWalkerPropagatesTransitionBlocked` — promotion against an item whose start-criteria are unmet returns `ErrPromotionBlocked`.
- **Blocked by:** 2.1, 2.2 (package compile + needs subscriber for event-driven walks; in this droplet the walker is invoked imperatively, but its method signature must align with the broker channel from 2.2). Cross-wave: Wave 1 `paths`/`packages` on `ActionItem` (walker reads these to call lock managers in the next droplet — but pure walker logic here doesn't yet acquire locks), Wave 1 `state` MCP-create+move (walker uses lifecycle state names).
- **Notes:**
  - Walker is intentionally **read-only on the tree** — it returns `[]ActionItem`, doesn't promote inline. `Promote` is a separate method so 2.7 (conflict detector) can intercede between "eligible" and "promoted."
  - Plan-QA falsification attack: "walker promotes a parent before all children-blockers verified — race with a concurrent `MoveActionItem` from CLI." Mitigation: `Service.MoveActionItem` already enforces start-criteria + completion-criteria invariants atomically. Walker is best-effort eligibility; final gate is in `Service`.
  - Plan-QA falsification attack: "walker reads stale `BlockedBy` after a same-tick `UpdateActionItem`." Mitigation: walker is invoked from broker events that fire AFTER the repo write (per 2.2 acceptance). Document the ordering in walker.go's package doc-comment.

### Wave 2.6 — AGENT SPAWN

- **State:** todo
- **Paths:** `internal/app/dispatcher/spawn.go` (NEW), `internal/app/dispatcher/spawn_test.go` (NEW)
- **Packages:** `internal/app/dispatcher`
- **Acceptance:**
  - `spawner` struct wrapping `*os/exec.Cmd` construction (not execution — execution is monitored by 2.8).
  - `BuildSpawnCommand(item domain.ActionItem, project domain.Project, catalog templates.KindCatalog, authBundle AuthBundle) (*exec.Cmd, SpawnDescriptor, error)` returns:
    - The constructed `*exec.Cmd` with `Dir = project.RepoPrimaryWorktree`, full argv per REVISION_BRIEF line 49 (`claude --agent <agentName> --bare -p "<prompt>" --mcp-config <perRunPath> --strict-mcp-config --permission-mode acceptEdits --max-budget-usd <N> --max-turns <N>`).
    - `SpawnDescriptor` capturing `AgentName`, `Model`, `MaxBudgetUSD`, `MaxTurns`, `MCPConfigPath`, `Prompt`, `WorkingDir` for logging + the monitor in 2.8.
  - Agent variant resolution: looks up `catalog.LookupAgentBinding(item.Kind)` (existing `KindCatalog` API per `internal/templates/catalog.go:92`). If present, uses `binding.AgentName`. If absent, returns `ErrNoAgentBinding`. If `binding.AgentName` matches the `{lang}-{role}` variant pattern (`go-builder-agent`, `fe-qa-proof-agent`, etc.) and `project.Language` is set, **the planner-defined `AgentName` is taken verbatim**; we do not synthesize variants from `project.Language` in this droplet — Wave 1 gives the planner the project field, the template gives the binding. Document this division of labor.
  - `AuthBundle` is a stub struct with comment "filled in by Wave 3"; constructor accepts a zero-value bundle and emits the `--mcp-config` flag pointing at a placeholder path (`<project_root>/.tillsyn/dispatcher-spawn-XXXX.json`). Wave 3 will wire the real auth-bundle assembly.
  - Test scenarios:
    - `TestBuildSpawnCommandAssemblesArgvForGoBuilder` — fixture: action item with kind=`build`, project with `Language=go`, catalog with `agent_bindings.build = {agent_name = "go-builder-agent", model = "opus", max_turns = 20, max_budget_usd = 5}`. Assert argv matches the expected slice.
    - `TestBuildSpawnCommandSetsCwd` — assert `cmd.Dir == project.RepoPrimaryWorktree`.
    - `TestBuildSpawnCommandReturnsErrNoAgentBindingForUnboundKind` — kind without an entry in `agent_bindings`.
    - `TestBuildSpawnCommandPropagatesAuthBundleStubPath` — placeholder auth path is non-empty.
  - **No actual subprocess execution in this droplet's tests** — spawn is `*exec.Cmd` construction only. Execution lives in 2.8 (monitor) and is exercised via a fake-binary `mage test-pkg` scenario there.
- **Blocked by:** 2.1. Cross-wave: Wave 1 project-node fields (`RepoPrimaryWorktree`, `Language`, `HyllaArtifactRef`, `DevMcpServerName`); Drop 3's `templates.KindCatalog.LookupAgentBinding` (already landed at `internal/templates/catalog.go:92`).
- **Notes:**
  - Plan-QA falsification attack: "what if `binding.AgentName` is empty but the binding exists?" Mitigation: `AgentBinding.Validate` (already lands in Drop 3.13 per `internal/templates/schema.go:264-269`) rejects empty `AgentName` at template-load time. Test asserts the validate-rejected case with a manually-corrupted catalog.
  - Plan-QA falsification attack: "the prompt is hand-rolled here — drift from agent file requirements." Mitigation: prompt assembly is deferred to a `promptAssembler` in this droplet that takes the action item + project + bundle and returns a string; Wave 4's CLAUDE.md updates document the contract. The prompt body itself is opaque to this droplet; tests assert structural fields (`task_id`, `project_dir`, `hylla_artifact_ref`, `move-state directive`) are present, not the full prose.
  - Plan-QA falsification attack: "spawning `claude` requires `claude` on PATH — what if it isn't?" Mitigation: this droplet ONLY constructs `*exec.Cmd` — execution-not-found is the monitor's (2.8) concern. Document.
  - Coupling to Wave 3: **the auth-bundle stub here is the Wave 3 hand-off point.** Wave 3 droplet 3.X (auth flow) will land the `AuthBundle` populated form and replace this droplet's stub. Surface as an explicit cross-wave seam in plan-QA review.

### Wave 2.7 — CONFLICT DETECTOR — SIBLING OVERLAP

- **State:** todo
- **Paths:** `internal/app/dispatcher/conflict.go` (NEW), `internal/app/dispatcher/conflict_test.go` (NEW)
- **Packages:** `internal/app/dispatcher`
- **Acceptance:**
  - `conflictDetector` struct with `DetectSiblingOverlap(ctx context.Context, item domain.ActionItem, siblings []domain.ActionItem) (overlaps []SiblingOverlap, err error)`.
  - `SiblingOverlap` struct: `SiblingID string`, `OverlapKind enum {file, package}`, `OverlapValue string` (the conflicting path or package), `HasExplicitBlockedBy bool`.
  - Detection rule (per CLAUDE.md "File and package level blocking" rule + REVISION_BRIEF L3): two siblings with the same parent and overlapping `paths` OR `packages` MUST have an explicit `blocked_by` between them. The detector flags every overlap and reports whether the existing `BlockedBy` covers it.
  - `InsertRuntimeBlockedBy(ctx context.Context, item domain.ActionItem, siblingID string, reason string) error` — calls `Service.UpdateActionItem` adding `siblingID` to `BlockedBy`. Idempotent: if already present, no-op.
  - Test scenarios:
    - `TestDetectorFindsFileOverlapBetweenSiblings`
    - `TestDetectorFindsPackageOverlapBetweenSiblings`
    - `TestDetectorIgnoresNonSiblings` (different parents)
    - `TestDetectorReportsExplicitBlockedByCovered` — overlap exists but `item.BlockedBy` already includes the sibling.
    - `TestInsertRuntimeBlockedByIsIdempotent`
    - `TestInsertRuntimeBlockedByPostsAttentionItem` — when a runtime blocker is inserted, an attention item fires for the orchestrator (per REVISION_BRIEF Wave 2 line 50 "insert a runtime blocker rather than racing"). Use existing `Service.RaiseAttention` API.
- **Blocked by:** 2.1, 2.5 (walker — conflict detector runs after walker identifies eligible items but BEFORE promote). Cross-wave: Wave 1 `paths`/`packages` on `ActionItem`.
- **Notes:**
  - Plan-QA falsification attack: "siblings here means same `ParentID` — what about cross-subtree overlap (cousins)?" Mitigation: Drop 4a treats sibling-only because cross-subtree blocking explodes the search space. Drop 4b discusses cousins. Document in conflict.go's doc-comment as a known limitation.
  - Plan-QA falsification attack: "what if both siblings are equally eligible — which gets the lock?" Mitigation: deterministic tie-break by `actionItem.Position` (already a field), then by `ID` lex-order. Tested.
  - Plan-QA falsification attack: "an inserted runtime `blocked_by` is permanent — it doesn't get cleaned up after the holder completes." Mitigation: that's the *correct* behavior — the runtime blocker IS the dependency edge. Drop 4b's gate-aware closeout may revisit cleanup of inserted blockers; this droplet documents the rationale and defers.

### Wave 2.8 — PROCESS MONITORING

- **State:** todo
- **Paths:** `internal/app/dispatcher/monitor.go` (NEW), `internal/app/dispatcher/monitor_test.go` (NEW), `internal/app/dispatcher/testdata/fakeagent.go` (NEW — testing helper that compiles to a fake agent binary)
- **Packages:** `internal/app/dispatcher`
- **Acceptance:**
  - `processMonitor` struct with `Track(actionItemID string, cmd *exec.Cmd) (Handle, error)` that starts the process and returns a `Handle` with `Wait() (TerminationOutcome, error)`.
  - `TerminationOutcome` is a struct: `ExitCode int`, `Signal string` (empty if clean exit), `Crashed bool`, `Duration time.Duration`.
  - On crash detection (non-zero exit OR signal): the monitor calls `Service.MoveActionItem` to transition the action item to `failed`, sets `metadata.outcome = "failure"` and `metadata.failure_reason = "agent process crashed: <signal/exit code>"` via `Service.UpdateActionItem`.
  - Concurrent-safe: multiple `Track` calls share the same monitor; each returns its own `Handle`.
  - Test scenarios (using a tiny `testdata/fakeagent.go` compiled via `go build` into a temp binary by the test setup — exec'd through `exec.Command`, NOT raw `go run`):
    - `TestMonitorCleanExitMarksNoFailure` — fake agent exits 0; action item state untouched by monitor (the agent's own state-update is what completes it).
    - `TestMonitorNonZeroExitMarksFailed` — fake agent exits 1; action item moves to `failed` with `metadata.failure_reason` populated.
    - `TestMonitorSignalKilledMarksFailed` — fake agent killed via `cmd.Process.Kill()`; action item moves to `failed`.
    - `TestMonitorTracksDurationAccurately` — fake agent sleeps 100ms; `outcome.Duration >= 100ms`.
- **Blocked by:** 2.1, 2.6 (spawn — monitor consumes `*exec.Cmd` from spawn). Cross-wave: none.
- **Notes:**
  - Plan-QA falsification attack: "what if the action item already moved to `complete` (agent succeeded and updated state) before monitor sees the exit?" Mitigation: monitor checks current `LifecycleState` before applying `failed` transition. If `complete` already, monitor logs but does NOT downgrade. Tested.
  - Plan-QA falsification attack: "compiling a fake agent binary in tests is fragile across CI." Mitigation: `mage test-pkg` already runs `go build` for test compilation; `testdata/fakeagent.go` is excluded from main package via build tag `//go:build ignore`. Test setup invokes `go build -o <tmpfile> testdata/fakeagent.go` via `exec.Command("go", ...)` — note that **this is test-helper invocation of `go build`, not source-line `go build` in production code.** Surface to plan-QA: this is the one carve-out from "never raw go" because mage tests always shell out to `go` underneath.
  - Plan-QA falsification attack: "monitor goroutine leaks if `Wait` is never called." Mitigation: `Track` returns a `Handle` whose `Close()` cancels and reaps; documented contract. Test exercises leak-free path.

### Wave 2.9 — TERMINAL-STATE CLEANUP

- **State:** todo
- **Paths:** `internal/app/dispatcher/cleanup.go` (NEW), `internal/app/dispatcher/cleanup_test.go` (NEW)
- **Packages:** `internal/app/dispatcher`
- **Acceptance:**
  - `cleanupHook` struct with `OnTerminalState(ctx context.Context, item domain.ActionItem) error` invoked when an action item transitions to `complete`, `failed`, or `archived`.
  - Cleanup actions executed in order:
    1. `fileLockManager.Release(item.ID)` (no-op if no locks held).
    2. `packageLockManager.Release(item.ID)`.
    3. **Auth-bundle revoke stub** — calls a `revokeAuthBundle(actionItemID string) error` method whose body is `// Wave 3 fills this in` returning nil. Surface as the explicit Wave 3 seam.
    4. Process-monitor unsubscribe (only the monitor's tracked-PID map; the process itself has already exited).
  - Idempotent: calling `OnTerminalState` twice for the same item is a no-op on the second call.
  - Test scenarios:
    - `TestCleanupReleasesFileAndPackageLocks` — pre-acquire both, call `OnTerminalState`, assert both released.
    - `TestCleanupIsIdempotent`
    - `TestCleanupOnArchivedAlsoFires` — archive transition is treated as terminal for lock-release purposes.
    - `TestCleanupContinuesPastIndividualFailure` — if file-lock release errors, package-lock release still attempted, error aggregated via `errors.Join`.
- **Blocked by:** 2.1, 2.3, 2.4, 2.6 (touches both lock managers + the auth-bundle stub from spawn). Cross-wave: none.
- **Notes:**
  - Plan-QA falsification attack: "what if cleanup runs while a sibling is mid-acquire on the same path?" Mitigation: `fileLockManager.Release` is mutex-guarded internally per 2.3; the sibling's `Acquire` will see the released state on its next check. Tested via concurrent acquire-while-releasing scenario.
  - Plan-QA falsification attack: "auth-bundle revoke is a stub; cleanup will silently leak credentials in Drop 4a." Mitigation: ACCEPTED for 4a — the manual-trigger CLI runs in dev shell, dev cleans up via `till auth_request revoke` if needed. Wave 3 lands the real revoke. Document in cleanup.go's package doc-comment as a Drop 4b deliverable cross-reference.

### Wave 2.10 — MANUAL-TRIGGER CLI

- **State:** todo
- **Paths:** `cmd/till/dispatcher_cli.go` (NEW), `cmd/till/dispatcher_cli_test.go` (NEW), `cmd/till/main.go` (extend root command tree to register `dispatcher` subcommand)
- **Packages:** `cmd/till`
- **Acceptance:**
  - New `till dispatcher` cobra subcommand with one child: `till dispatcher run --action-item <id>`.
  - Flags: `--action-item <UUID>` (required), `--project <UUID>` (optional; resolved from action-item if omitted), `--dry-run` (boolean; if set, emits the planned spawn descriptor as JSON without executing).
  - `RunE` body: instantiates a `dispatcher.Dispatcher` via `NewDispatcher(svc, broker, opts)`, calls `RunOnce(ctx, actionItemID)`, prints the resulting `DispatchOutcome` as a human-readable line (`spawned go-builder-agent for <id> in 12s` / `skipped: <reason>`).
  - On `--dry-run`: walks the eligibility check + builds the spawn descriptor + prints it; does NOT call the spawner. Exits 0.
  - On a `Result=Failed` outcome: exits non-zero with the failure reason on stderr.
  - Test scenarios (using the existing `cmd/till/main_test.go` test harness pattern):
    - `TestDispatcherRunCmdMissingActionItemFlagErrors`
    - `TestDispatcherRunCmdSkipsWhenItemNotInTodo` — fixture: action item in `complete`; CLI exits 0 with `skipped: not in todo`.
    - `TestDispatcherRunCmdDryRunPrintsDescriptor` — fixture: eligible item with binding; CLI exits 0, stdout contains `agent_name`, `model`, `working_dir`.
    - `TestDispatcherRunCmdSpawnsAndReports` — fixture: eligible item with binding pointing at a fake-agent binary. Asserts spawn happens, action item ends up in `in_progress`, monitor handle returned.
  - Help text registered with the existing `cmd/till/help.go` system; `mage test-pkg cmd/till` covers help-rendering golden test if one exists in the repo.
- **Blocked by:** 2.1, 2.5, 2.6, 2.7, 2.8, 2.9 (CLI is the orchestration entry point — needs walker, spawn, conflict detector, monitor, cleanup all wired). Cross-wave: Wave 1 project-node fields (CLI must construct `Service` against a project that has `RepoPrimaryWorktree` populated).
- **Notes:**
  - This droplet is the manual-trigger milestone deliverable per REVISION_BRIEF L7. Auto-promotion-on-state-change wiring (continuous mode) is Drop 4b.
  - Plan-QA falsification attack: "the CLI runs in the dev shell — it doesn't have the dispatcher-singleton broker." Mitigation: CLI bootstraps its own broker + service from the standard `cmd/till/main.go` initialization path (same pattern as `till serve`). The broker isn't shared with a long-running server in this milestone — each `till dispatcher run` is a one-shot. Documented limitation; Drop 4b lands the daemon variant.
  - Plan-QA falsification attack: "if the spawn succeeds and the agent runs for 30 minutes, the CLI hangs." Mitigation: `RunOnce` returns AS SOON AS the spawn descriptor is constructed and the agent is launched in a tracked subprocess; the CLI does NOT wait for agent completion. Documented + tested via `TestDispatcherRunCmdSpawnsAndReports` asserting CLI exits within 1s of spawn.
  - Plan-QA falsification attack: "two parallel `till dispatcher run` invocations targeting overlapping items will race on the lock managers." Mitigation: they DO race, and they DO see each other's locks because both bootstrap from the same SQLite DB (locks are in-process per-CLI today). This is an explicit Drop 4b gap — surface to plan-QA + dev. Mitigation in 4a: document the limitation; recommend dev runs CLI invocations serially during the manual-trigger milestone.

---

## Wave Acceptance Summary

Wave 2 closes when:

- All ten droplets reach `complete` with both build-QA-proof and build-QA-falsification green.
- `mage ci` passes locally with the new `internal/app/dispatcher` package included in coverage.
- `till dispatcher run --action-item <id>` (manual-trigger CLI) successfully spawns at least one fake-agent fixture in test against an in-tree project fixture with Wave-1 fields populated.
- All cross-wave seams to Wave 3 (auth bundle, revoke stub) are documented with explicit `// Wave 3` comment markers and a passing test that asserts the stub returns nil.
- Conflict-detector + walker round-trip exercise: planted overlap between two siblings causes runtime `blocked_by` insertion + attention-item fire, integration-tested.

## Open Questions for Plan-QA Review

1. **Same-package contention vs parallelism.** Ten droplets in one new package serialize via package-lock. Is the resulting linear chain (with three short parallel branches at 2.2/2.3/2.6) acceptable, or does plan-QA prefer further splitting (e.g., move locks into a sub-package `internal/app/dispatcher/locks/`)? Author's stance: KEEP the flat package — sub-packages add navigation cost for a 10-file module.

2. **`exec.Cmd` construction vs execution split (2.6 vs 2.8).** Splitting cmd-construction (2.6) from cmd-execution (2.8) is necessary for testability without a real `claude` binary on PATH. But this means 2.8 owns *both* execution AND monitoring, a wider responsibility than the brief implies. Plan-QA falsification: is this the right split?

3. **Auth-bundle stub seam.** Wave 3 will replace the placeholder `<project_root>/.tillsyn/dispatcher-spawn-XXXX.json` with a real auth-bundle materialization. Is the stub interface (`AuthBundle` zero value, placeholder path) sufficient for Wave 3 to plug into, or does Wave 2 need to commit to a more concrete contract now?

4. **`metadata.failure_reason` field shape.** Droplet 2.8 writes `metadata.failure_reason = "agent process crashed: <code>"`. Is this a free-form string today, or does Drop 4b need a structured `failure` type per PLAN.md L76 ("`failure` concrete type with `failure_kind`/`diagnostic`/`fix_directive`" — deferred)? Author's stance: free-form string in 4a; Drop 4b refactors to structured type.

5. **Test-harness `go build` carve-out (2.8 fakeagent).** Mage rule says "never raw `go build`" — but compiling a `testdata/fakeagent.go` to a temp binary inside a test setup function IS a `go build` invocation under the hood. Is this acceptable as a test-helper carve-out? Precedent in repo: `cmd/till/main_test.go` already compiles fixture binaries this way — verify and reference.

6. **CLI bootstrap symmetry with `till serve`.** Droplet 2.10 bootstraps a Service + Broker per CLI invocation. Drop 4b will land the daemon variant where the dispatcher attaches to the long-running `till serve` process. Should 2.10 ALREADY structure the bootstrap so the daemon variant is a trivial extension, or is that forward-engineering YAGNI? Author's stance: minimal CLI today; 4b refactors.

## Hylla Feedback

N/A — planning touched non-Go files only (PLAN fragment authored against MD brief + Read of Go source for symbol verification; no Hylla queries issued because all symbols verified via direct Read of files identified in the brief). Hylla today indexes only Go and the brief explicitly directed to Read/Grep/Glob/LSP for code understanding.
