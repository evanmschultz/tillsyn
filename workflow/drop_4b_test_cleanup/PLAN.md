# DROP_4B_TEST_CLEANUP — Drop 4b Deferred Refinements (R5/R6/R7/R8)

**State:** planning
**Blocked by:** —
**Paths (expected):** test files under `internal/app/dispatcher/` (R6/R7 D5 e2e tests), `internal/app/comments.go` + adjacent (R5 `till.comment` target_type fix), `internal/app/action_items.go` + MCP adapter layer (R8 supersede MCP operation). Planner narrows per droplet.
**Packages (expected):** `internal/app`, `internal/app/dispatcher`, `internal/adapters/server/mcpapi`. Planner sets per-droplet packages.
**PLAN.md ref:** Drop 4b deferred-refinement absorption — itemized in `REVISION_BRIEF.md`
**Workflow:** `workflow/example/drops/WORKFLOW.md`
**Cascade concept:** `AGENT_CASCADE_DESIGN.md`
**Started:** 2026-05-18
**Closed:** —

## Scope

Land Drop 4b's deferred refinements R5/R6/R7/R8 from `project_drop_4b_refinements_raised.md` — name shorthand "test cleanup" because R6/R7 dominate the LOC count, but the drop also covers two coordination-surface MCP fixes (R5, R8). R1/R2/R3/R4 stay parked for a separate template-validation hardening drop later.

See `REVISION_BRIEF.md` for the four refinement breakdowns with file/line pointers, fix paths, and acceptance criteria.

## Planner

### Droplet 1.1 — R5: domain `CommentTargetType` alias normalization

**State:** todo
**Paths:**
- `internal/domain/comment.go`
- `internal/domain/comment_test.go`
**Packages:** `internal/domain`
**Blocked by:** —

**Scope:** Fix the R5 bug at the domain level. `NormalizeCommentTargetType` currently matches `validCommentTargetTypes = ["project", "action_item"]` case-insensitively, but `"actionItem"` (camelCase, schema-declared) does not normalize to `"action_item"` (snake_case). Add a pre-normalization alias step in `NormalizeCommentTargetType` that maps the lowercased form of `"actionitem"` to `"action_item"` before the `validCommentTargetTypes` lookup, so MCP callers passing `"actionItem"` resolve to the correct domain token.

**Changes (new, not yet in tree):**
- `internal/domain/comment.go`: in `NormalizeCommentTargetType`, before the `validCommentTargetTypes` range loop, add a switch/map step that converts `"actionitem"` → `"action_item"`. Keep the existing range loop for the canonical-form pass-through.
- `internal/domain/comment_test.go`: extend existing `NormalizeCommentTargetType` and `IsValidCommentTargetType` table-driven tests with alias cases: `"actionItem"`, `"ActionItem"`, `"ACTIONITEM"` all normalize to `"action_item"` and pass `IsValidCommentTargetType`.

**Acceptance:**
- `mage test-pkg internal/domain` passes.
- `NormalizeCommentTargetType("actionItem")` returns `"action_item"`.
- `IsValidCommentTargetType("actionItem")` returns `true`.
- Existing tests unchanged (no regressions on `"project"` and `"action_item"` paths).

---

### Droplet 1.2 — R5: MCP schema enum update for `till.comment`

**State:** todo
**Paths:**
- `internal/adapters/mcp_rpc/extended_tools.go`
**Packages:** `internal/adapters/mcp_rpc`
**Blocked by:** —

**Scope:** Update the `target_type` schema enum in `registerCommentTools` (`extended_tools.go` line 2243) to reflect post-Drop-1.75 vocabulary. The current enum `("project", "branch", "phase", "actionItem", "subtask", "decision", "note")` includes stale pre-1.75 tokens. Replace with `("project", "action_item", "actionItem")` where `"action_item"` is the canonical form and `"actionItem"` is kept as an accepted alias (so callers that already pass `"actionItem"` continue to work after D1's domain fix). Update the `target_type` description string to note that `"action_item"` is the canonical post-Drop-1.75 form.

This droplet is adapter-only (schema declaration). Domain validation is fixed in droplet 1.1. No new test file needed — tested via `mage test-pkg internal/adapters/mcp_rpc` (existing handler tests).

**Changes:**
- `internal/adapters/mcp_rpc/extended_tools.go`: change `mcp.Enum("project", "branch", "phase", "actionItem", "subtask", "decision", "note")` to `mcp.Enum("project", "action_item", "actionItem")` and update the description string for `target_type`.

**Acceptance:**
- `mage test-pkg internal/adapters/mcp_rpc` passes.
- Schema enum for `till.comment target_type` contains `"project"`, `"action_item"`, `"actionItem"` and excludes stale pre-1.75 tokens (`"branch"`, `"phase"`, `"subtask"`, `"decision"`, `"note"`).

---

### Droplet 1.3 — R6 fixups + R7.4 file split

**State:** todo
**Paths:**
- `internal/app/dispatcher/subscriber_test.go`
- `internal/app/dispatcher/dispatcher_e2e_test.go` (new — not yet in tree)
**Packages:** `internal/app/dispatcher`
**Blocked by:** —

**Scope:** Three R6 fixups + the R7.4 file split in one atomic step (merged because D3 renames the test being moved by D4; merging avoids a two-step rename-then-move).

**R6.1:** Rename `TestAutoDispatchE2EGatePassViaNewDispatcher` (subscriber_test.go line 543) to `TestAutoDispatch_NewDispatcherGateWiring`. The name overstates scope; the test's inline comment (lines 533-542) already documents the honest scope.

**R6.2 (deviation from brief):** Do NOT add a `runtime.NumGoroutine()` bracket — it is unreliable in `t.Parallel()` suites (sibling tests inflate the count). Instead, add a comment near the `t.Cleanup` call in the renamed test: `// R6.2: goroutine-leak detection deferred; requires go.uber.org/goleak (not in go.mod). t.Cleanup(d.Stop) is the structural drain guard.`

**R6.3:** Add a clarifying comment to the `lister.calls == 1` assertion in `TestDispatcherStartSpawnsPerProjectSubscribers`: `// "state transitions" in D5 spec means dispatcher lifecycle (Start/Stop), not action-item state. This lister-calls pin is the lifecycle-transition signal.`

**R7.4:** Create `internal/app/dispatcher/dispatcher_e2e_test.go` (new file, package `dispatcher`). Move `stubE2ETemplateResolver` (subscriber_test.go lines 509-522), the renamed `TestAutoDispatch_NewDispatcherGateWiring` (previously `TestAutoDispatchE2EGatePassViaNewDispatcher`), and `TestAutoDispatchE2EGateFailViaNewDispatcher` (subscriber_test.go lines 616-666) into the new file. Remove those symbols from `subscriber_test.go`. The new file gets the package declaration, required imports, and the moved symbols.

**Acceptance:**
- `mage test-pkg internal/app/dispatcher` passes with same test count.
- `dispatcher_e2e_test.go` exists and contains the e2e tests.
- `subscriber_test.go` no longer contains `stubE2ETemplateResolver` or the two e2e test functions.
- `TestAutoDispatch_NewDispatcherGateWiring` exists (renamed from `TestAutoDispatchE2EGatePassViaNewDispatcher`); old name no longer exists.

---

### Droplet 1.4 — R7.1 + R7.2 + R7.3: e2e test enrichment

**State:** todo
**Paths:**
- `internal/app/dispatcher/dispatcher_e2e_test.go`
**Packages:** `internal/app/dispatcher`
**Blocked by:** 1.3

**Scope:** Enrich the e2e tests in `dispatcher_e2e_test.go` per R7.1, R7.2, R7.3.

**R7.1:** Add `TestAutoDispatchE2E_GateFailFullChain` (new, not yet in tree). Wire a walker stub returning a `KindBuild` action item with a project that resolves. Drive `d.gates.Run` via the gate runner against the `GateKindMageTestPkg` template (not registered by `NewDispatcher`) — assert `GateStatusFailed` + `errors.Is(results[0].Err, ErrGateNotRegistered)`. If reaching `applyCleanExitTransition` via the broker chain requires injecting a fake processMonitor (the real chain tries to spawn an agent process), builder must document the actual scope in the worklog — direct `applyCleanExitTransition` unit coverage via a constructed monitor is acceptable if the full chain is impractical in an in-package test.

**R7.2:** Add `TestAutoDispatchE2E_ApplyCleanExitTransitionCoverage` (new, not yet in tree). Use the `stubMonitorService` pattern from `monitor_test.go` to construct a `processMonitor` directly and call `applyCleanExitTransition` with scenarios covering: (C1) skip path when item is already `StateComplete` before gate run, and (C2) ctx-cancel pre-loop. These tests exercise the discriminated paths added inline at commit `d949f6f`. Builder notes honest scope in worklog.

**R7.3:** Parameterize `stubE2ETemplateResolver`. Add field `tplByProject map[string]templates.Template` to `stubE2ETemplateResolver`. In `GetProjectTemplate`, if `tplByProject[projectID]` is set, return it; else return `tpl`. Add one table-driven test `TestStubE2ETemplateResolverRoutesPerProject` (new, not yet in tree) asserting per-project routing returns the configured per-project template.

**Acceptance:**
- `mage test-pkg internal/app/dispatcher` passes.
- `dispatcher_e2e_test.go` contains the three new tests plus the parameterized stub.
- Builder worklog honestly describes scope of R7.1 and R7.2 (full chain vs direct monitor construction, whichever is achieved).

---

### Droplet 1.5 — R8: `SupersedeActionItem` interface + adapter

**State:** todo
**Paths:**
- `internal/adapters/mcp_common/mcp_surface.go`
- `internal/adapters/mcp_common/app_service_adapter_mcp.go`
- `internal/adapters/mcp_common/app_service_adapter_lifecycle_test.go`
**Packages:** `internal/adapters/mcp_common`
**Blocked by:** —

**Scope:** Wire the supersede path into the MCP adapter layer.

**`mcp_surface.go`:** Add `SupersedeActionItem(context.Context, SupersedeActionItemRequest) (domain.ActionItem, error)` to the `ActionItemService` interface (after `ReparentActionItem`). `SupersedeActionItemRequest` type already exists at line 354 (`internal/adapters/mcp_common/mcp_surface.go`).

**`app_service_adapter_mcp.go`:** Implement `AppServiceAdapter.SupersedeActionItem`. Pattern:
1. Nil-guard on `a.service`.
2. Trim/validate `in.ActionItemID` non-empty.
3. Call `withMutationGuardContext` for guard-context setup.
4. Fetch the existing item via `a.service.GetActionItem` to run `assertOwnerStateGate`.
5. Check orchestrator-role: if the authenticated caller (from context) has `principal_role != "orchestrator"`, return `ErrAuthorizationDenied` with a clear message ("supersede requires orchestrator-role session").
6. Call `a.service.SupersedeActionItem(ctx, in.ActionItemID, in.Reason)`.
7. Map errors via `mapAppError`.
8. Return `domain.ActionItem`.

**Tests in `app_service_adapter_lifecycle_test.go`:**
- Happy path: orchestrator session + failed item → returns superseded item.
- Non-orchestrator session → `ErrAuthorizationDenied`.
- STEWARD-owned item + non-steward session → `ErrAuthorizationDenied` (from `assertOwnerStateGate`).
- Missing `ActionItemID` → `ErrInvalidCaptureStateRequest`.

**Acceptance:**
- `mage test-pkg internal/adapters/mcp_common` passes.
- `AppServiceAdapter` satisfies the updated `ActionItemService` interface (compile check via test).
- Supersede with orchestrator session succeeds; without orchestrator session fails with auth error.

---

### Droplet 1.6 — R8: `till.action_item operation=supersede` MCP tool registration

**State:** todo
**Paths:**
- `internal/adapters/mcp_rpc/extended_tools.go`
- `internal/adapters/mcp_rpc/extended_tools_test.go`
**Packages:** `internal/adapters/mcp_rpc`
**Blocked by:** 1.2, 1.5

**Scope:** Expose `operation=supersede` on the `till.action_item` MCP tool.

**`extended_tools.go`:**
1. Add `"supersede"` to the `till.action_item` operation enum at line 1440: `mcp.Enum("get", "list", "search", "create", "update", "move", "move_state", "delete", "restore", "reparent", "supersede")`.
2. Add `Reason *string `json:"reason"`` field to the `args` struct inside `handleActionItemOperation` (pointer-sentinel so it distinguishes absent from empty-string, consistent with the pointer pattern used for `Title`, `Description`, etc. post-Drop-4c.5-A.1).
3. Add `case "supersede":` in `handleActionItemOperation` switch: validate `action_item_id` non-empty, validate `reason` pointer is non-nil and non-empty (return `invalid_request` if missing), call `authorizeMCPMutation` with action `"supersede_action_item"`, build actor tuple, call `tasks.SupersedeActionItem(ctx, mcpcommon.SupersedeActionItemRequest{ActionItemID: actionItemID, Reason: *args.Reason, Actor: actor})`, return JSON result.
4. Update `till.action_item` description string to include `operation=supersede` in the operation list.

**`extended_tools_test.go`:** Add table-driven tests for supersede:
- Valid orchestrator session + `"failed"` item + non-empty reason → success, item returned with `LifecycleState=complete`.
- Missing `reason` → `invalid_request`.
- Non-orchestrator session → `auth_denied`.
- Unknown `action_item_id` → `not_found`.

**Acceptance:**
- `mage test-pkg internal/adapters/mcp_rpc` passes.
- `till.action_item operation=supersede` is in the registered tool enum.
- Tests cover role-gating, subtree-gating (via not-found), and the happy path.
- `mage ci` passes (drop-end verification).

## Notes

**Open questions for orchestrator / dev:**

- **R6.2 deviation**: goroutine-leak bracket not implemented (unreliable in parallel test suite without `goleak`). Confirm: (a) accept deferred-comment approach, OR (b) dev runs `go get go.uber.org/goleak` and builder adds goleak assertions.
- **R7.1/7.2 scope**: "integration chain" reaching `applyCleanExitTransition` via the full subscriber path requires injecting a fake processMonitor into the dispatcher (the real chain tries to spawn a claude agent process). Builder may fall back to direct `processMonitor.applyCleanExitTransition` unit-test construction. Confirm: (a) accept direct monitor unit test as sufficient, OR (b) require full subscriber chain (higher complexity, may need additional stub types).

**Planner answers to REVISION_BRIEF §7 questions:**
- **Q1 (R5 direction)**: option (a) confirmed. Domain alias normalization + schema enum update.
- **Q2 (R6.1)**: rename (option a). Enrichment would duplicate R7 work.
- **Q3 (R8 shape)**: `operation=supersede` on `till.action_item`. Not a new field on `update`.
- **Q4 (R7.4)**: split into `dispatcher_e2e_test.go`. Clean extraction point at line ~509 in current `subscriber_test.go`.
