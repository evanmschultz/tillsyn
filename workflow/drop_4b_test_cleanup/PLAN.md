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

**State:** done
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

**Notes (F6 — alias persistence round-trip):** `internal/adapters/storage/sqlite/repo.go:3095` re-normalizes on read using `NormalizeCommentTargetType`. If any existing row was stored as `'actionItem'` (the pre-D1.1 schema-declared form), reading it back after D1.1 will produce `'action_item'` (different from the stored bytes). Anything doing an exact-string match on the stored form (e.g. a UNIQUE constraint, a future migration script) will see the drift. Mitigation: the pre-MVP no-migration-logic rule (`feedback_no_migration_logic_pre_mvp.md`) means the dev wipes `~/.tillsyn/tillsyn.db` on any schema/state-vocab change — so no live stored rows with the camelCase form are expected in the dev environment. This drift is accepted under that rule. Builder does NOT need to add a persistence-path regression test; the Notes here are the acceptance record.

---

### Droplet 1.2 — R5: MCP schema enum update for `till.comment`

**State:** done
**Paths:**
- `internal/adapters/mcp_rpc/extended_tools.go`
**Packages:** `internal/adapters/mcp_rpc`
**Blocked by:** —

**Scope:** Update the `target_type` schema enum in `registerCommentTools` (`extended_tools.go` line 2243) to reflect post-Drop-1.75 vocabulary. The current enum `("project", "branch", "phase", "actionItem", "subtask", "decision", "note")` includes stale pre-1.75 tokens. Replace with `("project", "action_item", "actionItem")` where `"action_item"` is the canonical form and `"actionItem"` is kept as an accepted alias (so callers that already pass `"actionItem"` continue to work after D1.1's domain fix). Update the `target_type` description string to replace the stale `"project|branch|phase|actionItem|subtask|decision|note"` literal with `"project|action_item|actionItem"` — the description string is what MCP callers read as the contract; leaving it stale recreates the original R5 schema-vs-runtime drift.

This droplet is adapter-only (schema declaration). Domain validation is fixed in droplet 1.1. No new test file needed — tested via `mage test-pkg internal/adapters/mcp_rpc` (existing handler tests).

**Changes:**
- `internal/adapters/mcp_rpc/extended_tools.go`: change `mcp.Enum("project", "branch", "phase", "actionItem", "subtask", "decision", "note")` to `mcp.Enum("project", "action_item", "actionItem")` at line 2243. Update the `target_type` description string from `"project|branch|phase|actionItem|subtask|decision|note"` to `"project|action_item|actionItem"`.

**Acceptance:**
- `mage test-pkg internal/adapters/mcp_rpc` passes.
- Schema enum for `till.comment target_type` contains `"project"`, `"action_item"`, `"actionItem"` and excludes stale pre-1.75 tokens (`"branch"`, `"phase"`, `"subtask"`, `"decision"`, `"note"`).
- The `target_type` description string lists exactly `"project|action_item|actionItem"` and contains no stale pre-1.75 tokens.
- Regression: `IsValidCommentTargetType("branch")`, `IsValidCommentTargetType("phase")`, `IsValidCommentTargetType("subtask")`, `IsValidCommentTargetType("decision")`, `IsValidCommentTargetType("note")` all return `false` (domain layer enforces the shrunk enum). Builder adds these as table-driven cases in `internal/domain/comment_test.go` (or confirms they are implicitly covered by existing negative-case coverage) — the test confirming legacy-token rejection may live in either domain or mcp_rpc; builder picks the natural location.

---

### Droplet 1.3 — R6 fixups + R7.4 file split + goleak

**State:** done
**Paths:**
- `internal/app/dispatcher/subscriber_test.go`
- `internal/app/dispatcher/dispatcher_e2e_test.go` (new — not yet in tree)
**Packages:** `internal/app/dispatcher`
**Blocked by:** —

**DEV-ACTION PREREQ:** Before this droplet builds, the dev must run `go get go.uber.org/goleak` in the `main/` worktree shell. The goleak library is NOT in `go.mod` (confirmed). Builder blocks until the dep is available.

**Scope:** Three R6 fixups + the R7.4 file split in one atomic step (merged because D3 renames the test being moved by D4; merging avoids a two-step rename-then-move). Also wires goleak goroutine-leak detection per dev approval (F4 resolution).

**R6.1:** Rename `TestAutoDispatchE2EGatePassViaNewDispatcher` (subscriber_test.go line 543) to `TestAutoDispatch_NewDispatcherGateWiring`. The name overstates scope; the test's inline comment (lines 533-542) already documents the honest scope.

**R6.2:** Wire `go.uber.org/goleak` goroutine-leak detection. `TestAutoDispatchE2EGatePassViaNewDispatcher` (renamed to `TestAutoDispatch_NewDispatcherGateWiring`) does NOT call `t.Parallel()` (per its inline comment at line 541 — swaps the package-level `defaultCommandRunner` var). `TestAutoDispatchE2EGateFailViaNewDispatcher` DOES call `t.Parallel()` (line 626 in the current file; will be moved to `dispatcher_e2e_test.go` via R7.4 below). Since the two e2e tests have mixed parallel/non-parallel execution, use `goleak.VerifyTestMain(m)` in a `TestMain` function in `dispatcher_e2e_test.go` (the new file created by R7.4). Place `goleak.VerifyTestMain(m)` at the end of the `TestMain` body BEFORE calling `os.Exit`. Builder first checks whether any existing `TestMain` exists in the `dispatcher` package (no `TestMain` was found in the current file survey); if one is found, ADD `goleak.VerifyTestMain(m)` to the existing one rather than creating a new one. If `goleak.VerifyTestMain(m)` causes sibling-test goroutine-inflation false-positives during `mage test-pkg internal/app/dispatcher`, builder documents in worklog and falls back to `goleak.VerifyNone(t)` at the END of each e2e test body (not in `t.Cleanup` — goleak docs flag that as fragile).

**R6.2 scope-creep guard (round-2 falsification finding 1.4):** `goleak.VerifyTestMain(m)` runs package-wide and may surface goroutine leaks in tests UNRELATED to this droplet's scope (the dispatcher package has ~25 test files with subscriber loops, `dispatcher.Start` in tests, etc.). **If any such leak surfaces:**
1. Document the leak source (test name + likely goroutine source) in `BUILDER_WORKLOG.md` under a `## Out-of-Scope Leak Findings` subsection — these become refinement candidates for a future drop.
2. Do NOT fix the unrelated leak in this drop. Scope-creep would silently expand R6.2 into "audit and fix all goroutine leaks in `internal/app/dispatcher`" which is a separate drop's work.
3. Fall back to per-test `goleak.VerifyNone(t)` at the end of THIS droplet's two e2e test bodies — that scopes the leak detection to only the R6.2-target tests without disturbing unrelated tests.

**R6.3:** Add a clarifying comment to the `lister.calls == 1` assertion in `TestDispatcherStartTriggersRunOnceOnEvent` (or whichever test pins subscriber lifecycle state): `// "state transitions" in D5 spec means dispatcher lifecycle (Start/Stop), not action-item state. This lister-calls pin is the lifecycle-transition signal.`

**R7.4:** Create `internal/app/dispatcher/dispatcher_e2e_test.go` (new file, package `dispatcher`). Move `stubE2ETemplateResolver` (subscriber_test.go lines 509-522), the renamed `TestAutoDispatch_NewDispatcherGateWiring` (previously `TestAutoDispatchE2EGatePassViaNewDispatcher`), and `TestAutoDispatchE2EGateFailViaNewDispatcher` (subscriber_test.go lines 616-666) into the new file. Remove those symbols from `subscriber_test.go`. The new file gets the package declaration, required imports, goleak `TestMain`, and the moved symbols.

**Acceptance:**
- `mage test-pkg internal/app/dispatcher` passes with same test count.
- `dispatcher_e2e_test.go` exists and contains the e2e tests and goleak `TestMain`.
- `subscriber_test.go` no longer contains `stubE2ETemplateResolver` or the two e2e test functions.
- `TestAutoDispatch_NewDispatcherGateWiring` exists (renamed from `TestAutoDispatchE2EGatePassViaNewDispatcher`); old name no longer exists.
- Goleak is wired — either via `TestMain` + `goleak.VerifyTestMain(m)` or via `goleak.VerifyNone(t)` at end of each test body if `TestMain` causes sibling inflation. Builder worklog documents the approach taken.

---

### Droplet 1.4 — R7.1 + R7.2 + R7.3: e2e test enrichment

**State:** in_progress
**Paths:**
- `internal/app/dispatcher/dispatcher_e2e_test.go`
**Packages:** `internal/app/dispatcher`
**Blocked by:** 1.3

**Scope:** Enrich the e2e tests in `dispatcher_e2e_test.go` per R7.1, R7.2, R7.3.

**R7.1:** Add `TestAutoDispatchE2E_GateFailFullChain` (new, not yet in tree). The broker chain IS reachable from in-package tests: `TestDispatcherStartTriggersRunOnceOnEvent` (`subscriber_test.go:178`) and `TestHandleSubscriberEventInvokesRunOnceForEachEligibleItem` (`subscriber_test.go:445`) already drive `handleSubscriberEvent` end-to-end with stub walkers. For the gate-fail path (`ErrGateNotRegistered`), no subprocess spawn occurs because `applyCleanExitTransition` is called after the gate runner returns `GateStatusFailed` synchronously. Add a test using the same stub-walker pattern as those two existing tests: wire a walker stub returning a `KindBuild` action item, configure a `GateKindMageTestPkg` template (not registered by `NewDispatcher`), drive via `handleSubscriberEvent` (or the broker chain), assert `metadata.outcome=failure` on the action item after the monitor's `applyCleanExitTransition` fires. This is the end-to-end broker-chain test R7.1 requires. Builder documents the exact chain reached in the worklog. If the subscriber chain cannot be fully reached without a subprocess (builder discovers this at implementation time), builder MUST report in worklog and get dev sign-off before falling back to a direct-`processMonitor`-construction test — that fallback is NOT pre-authorized.

**R7.2:** Add `TestAutoDispatchE2E_ApplyCleanExitTransitionCoverage` (new, not yet in tree). Wire the subscriber chain as in R7.1, covering at least: (C1) skip path when item is already `StateComplete` before gate run, and (C2) ctx-cancel pre-loop. These paths were added inline at commit `d949f6f` and are unit-tested in `monitor_test.go` but NOT yet reached via the integration chain. The goal of R7.2 is the integration-chain coverage of those paths. Builder notes honest scope in worklog; same fallback rule as R7.1 applies (no pre-authorized direct-monitor fallback).

**R7.3:** Parameterize `stubE2ETemplateResolver`. Add field `tplByProject map[string]templates.Template` to `stubE2ETemplateResolver`. In `GetProjectTemplate`, if `tplByProject[projectID]` is set, return it; else return `tpl`. Add one table-driven test `TestStubE2ETemplateResolverRoutesPerProject` (new, not yet in tree) asserting per-project routing returns the configured per-project template.

**Notes (F5 — real-resolver limitation):** `TestStubE2ETemplateResolverRoutesPerProject` only proves the test stub routes per-project correctly; it does NOT pin production-resolver behavior. The real `dispatcherTemplateResolver` at `cmd/till/main.go:2704` lives in `package main` and is not importable from `internal/app/dispatcher`. Coverage of the real resolver's per-project routing logic is not testable from this package. This limitation is accepted for this drop. A future refinement adds a test in `cmd/till/main_test.go` (where `dispatcherTemplateResolver` lives) asserting per-project routing via the real resolver.

**Acceptance:**
- `mage test-pkg internal/app/dispatcher` passes.
- `dispatcher_e2e_test.go` contains the three new tests plus the parameterized `stubE2ETemplateResolver`.
- `TestAutoDispatchE2E_GateFailFullChain` reaches `applyCleanExitTransition` via the broker chain (default path); fallback to direct-monitor construction requires dev sign-off documented in the worklog.
- `TestAutoDispatchE2E_ApplyCleanExitTransitionCoverage` covers at least the C1 (already-complete skip) and C2 (ctx-cancel pre-loop) paths.
- Builder worklog documents the exact chain reached for R7.1 and R7.2.

---

### Droplet 1.5 — R8: `SupersedeActionItem` interface addition + mock-implementer compile gate

**State:** done
**Paths:**
- `internal/adapters/mcp_common/mcp_surface.go`
- `internal/adapters/mcp_common/app_service_adapter_lifecycle_test.go`
- `internal/adapters/mcp_rpc/extended_tools_test.go` (add `SupersedeActionItem` stub method to `stubExpandedService` — see acceptance below; required atomic with interface widening to keep `internal/adapters/mcp_rpc` package compile-green)
**Packages:** `internal/adapters/mcp_common`, `internal/adapters/mcp_rpc`
**Blocked by:** 1.2 (D1.2 also touches `internal/adapters/mcp_rpc/extended_tools.go` — same package, serialize per package-level locking rule from Drop 4a)

**Scope:** Add `SupersedeActionItem` to the `ActionItemService` interface. The adapter method `AppServiceAdapter.SupersedeActionItem` ALREADY EXISTS at `internal/adapters/mcp_common/app_service_adapter_mcp.go:1051-1075` (Drop 4c.5 droplet B.1). The `app_service_adapter_mcp.go` body is NOT modified by this droplet — any attempt to re-implement the method will introduce a compile error (duplicate function definition).

**`mcp_surface.go`:** Add `SupersedeActionItem(context.Context, SupersedeActionItemRequest) (domain.ActionItem, error)` to the `ActionItemService` interface at line 857 (after `ReparentActionItem`). `SupersedeActionItemRequest` type already exists at `mcp_surface.go:354`. This is the only change to `mcp_surface.go`.

**After the interface addition:** `AppServiceAdapter` already implements `SupersedeActionItem` at `app_service_adapter_mcp.go:1051-1075`. The compile-time interface-satisfaction check (via any existing test that assigns `*AppServiceAdapter` to `ActionItemService`) will confirm the method signature matches. Builder verifies via `mage test-pkg internal/adapters/mcp_common`.

**Role-gating note:** The existing `SupersedeActionItem` adapter method (line 1051-1075) uses `withMutationGuardContext` + `assertOwnerStateGate` — the same STEWARD owner-state-gate pattern as `MoveActionItemState` and `ReparentActionItem`. It does NOT include an orchestrator-role check at the adapter layer, which is correct: role-gating for this operation belongs at the MCP-RPC layer via `authorizeMCPMutation("supersede_task", ...)` in D1.6. Builder must NOT add a role check to the adapter body.

**Tests in `app_service_adapter_lifecycle_test.go`:**
- Happy path: any authenticated session + failed item with correct STEWARD ownership → returns superseded item. (Use a non-STEWARD-owned item to bypass the `assertOwnerStateGate`, matching test patterns elsewhere in the file.)
- STEWARD-owned item + non-steward session → `ErrAuthorizationDenied` (from `assertOwnerStateGate`). This mirrors the existing `TestStewardIntegrationDropOrchSupersedeRejected` at `handler_steward_integration_test.go:466` which exercises exactly this path via the full adapter.
- Missing `ActionItemID` → `ErrInvalidCaptureStateRequest`.
- `TestStewardIntegrationDropOrchSupersedeRejected` (`handler_steward_integration_test.go:466`) must still pass after this droplet lands — it calls `fixture.adapter.SupersedeActionItem` directly and tests that drop-orch-on-STEWARD-owned rejects. Builder verifies: `mage test-pkg internal/adapters/mcp_rpc` still green after D1.5 lands.

**Acceptance:**
- `mage test-pkg internal/adapters/mcp_common` passes.
- `ActionItemService` interface at `mcp_surface.go:848` includes `SupersedeActionItem(context.Context, SupersedeActionItemRequest) (domain.ActionItem, error)` after `ReparentActionItem`.
- `app_service_adapter_mcp.go` is NOT modified (no re-implementation of existing method).
- `TestStewardIntegrationDropOrchSupersedeRejected` at `handler_steward_integration_test.go:466` still passes (regression guard).
- Adapter-layer test cases in `app_service_adapter_lifecycle_test.go` pass: happy path + STEWARD-owner-gate rejection + missing-ID validation.
- **Mock-implementer compile gate (round-2 falsification finding 1.5):** `mage test-pkg internal/adapters/mcp_rpc` still compiles after the interface widening. The `stubExpandedService` test fake at `internal/adapters/mcp_rpc/extended_tools_test.go` implements `ActionItemService` — when this droplet adds `SupersedeActionItem` to the interface, the stub must gain a matching method. D1.5 (this droplet) adds a minimal stub: `func (s *stubExpandedService) SupersedeActionItem(ctx context.Context, req mcpcommon.SupersedeActionItemRequest) (domain.ActionItem, error) { return s.supersedeResult, s.supersedeErr }` (plus the two fields on the stub struct). D1.6's table-driven tests configure these fields per-case. Without this stub addition, D1.6 cannot start (cross-package compile failure).

---

### Droplet 1.6 — R8: `till.action_item operation=supersede` MCP tool registration

**State:** done
**Paths:**
- `internal/adapters/mcp_rpc/extended_tools.go`
- `internal/adapters/mcp_rpc/extended_tools_test.go`
**Packages:** `internal/adapters/mcp_rpc`
**Blocked by:** 1.2, 1.5

**Scope:** Expose `operation=supersede` on the `till.action_item` MCP tool.

**`extended_tools.go`:**
1. Add `"supersede"` to the `till.action_item` operation enum at line 1440: `mcp.Enum("get", "list", "search", "create", "update", "move", "move_state", "delete", "restore", "reparent", "supersede")`.
2. Add `mcp.WithString("reason", mcp.Description("Required for operation=supersede. Human-readable reason why this failed item is being superseded."))` to the `till.action_item` tool schema parameters.
3. Add `Reason *string `json:"reason"`` field to the `args` struct inside `handleActionItemOperation` (pointer-sentinel consistent with the `Owner`, `Title`, `Description` etc. pointer pattern established in Drop 4c.5-A.1 and visible at line 750 of the current struct). The `bindArgumentsStrict` decoder rejects unknown JSON keys — adding `Reason` to the struct is REQUIRED before the `supersede` case can receive the argument.
4. Add `case "supersede":` in `handleActionItemOperation` switch: validate `action_item_id` non-empty (also call `rejectMutationDottedActionItemID` as other mutation cases do), validate `args.Reason` is non-nil and `strings.TrimSpace(*args.Reason)` non-empty (return `invalid_request` if missing or blank), call `authorizeMCPMutation` with action string `"supersede_task"` (matching the `_task` suffix convention used by `"restore_task"` (line 1357), `"reparent_task"` (line 1401), `"delete_task"` (line 1309), `"create_task"` (line 931)), build actor tuple via `buildAuthenticatedMutationActor`, call `tasks.SupersedeActionItem(ctx, mcpcommon.SupersedeActionItemRequest{ActionItemID: actionItemID, Reason: *args.Reason, Actor: actor})`, return JSON result.
5. Update `till.action_item` description string (line 1439) to include `operation=supersede` in the operation list.
6. **Stale doc-comment cleanup (round-2 falsification finding 1.3):** update the two stale doc-comments that currently claim "no MCP exposes supersede":
   - `cmd/till/main.go:850` — the comment around the human-only supersede CLI must be updated to reflect that an MCP path now exists. Replace the existing "no MCP exposes supersede" phrasing with something like "Human-only CLI path; an MCP path also exists at `till.action_item operation=supersede` (gated by `authorizeMCPMutation`)."
   - `internal/adapters/mcp_common/mcp_surface.go:351` — same correction; comment must no longer claim MCP does not expose supersede.
   Builder reads each line for the exact stale wording before rewriting.

**`extended_tools_test.go`:** Add table-driven tests for supersede:
- Valid orchestrator session + `"failed"` item + non-empty reason → success, item returned with `LifecycleState=complete`.
- Missing `reason` (nil pointer) → `invalid_request`.
- Non-orchestrator session → `auth_denied`.
- Non-subtree `action_item_id` (item belongs to a different auth scope) → `auth_denied`. Note: `authorizeMCPMutation` returns `auth_denied` for scope mismatches; this is NOT a `not_found`. The test case is named "non-subtree action_item_id → `auth_denied`", not "subtree-gating via not-found" (correcting round-1 wording per 1.5-proof).
- Supersede on `todo`-state item (or `in_progress`, or `complete`) → typed error wrapping `ErrTransitionBlocked` with the `"supersede only applies to failed items"` message. This test pins the service-layer failed-only invariant (`service.go:1843-1845`) at the MCP boundary so a future refactor cannot silently weaken it for MCP callers.

**Acceptance:**
- `mage test-pkg internal/adapters/mcp_rpc` passes.
- `till.action_item operation=supersede` is in the registered tool enum.
- The `args` struct inside `handleActionItemOperation` contains `Reason *string`.
- The `till.action_item` schema includes `mcp.WithString("reason", ...)`.
- `authorizeMCPMutation` action string is `"supersede_task"` (not `"supersede_action_item"`).
- Tests cover: orchestrator role-gating (non-orch → `auth_denied`), non-subtree scope (`auth_denied`), happy path (failed item → complete), failed-only invariant (`todo`/`in_progress`/`complete` → `ErrTransitionBlocked`), and missing-reason (`invalid_request`).
- Stale doc-comments at `cmd/till/main.go:850` and `internal/adapters/mcp_common/mcp_surface.go:351` no longer contain phrasing claiming "no MCP exposes supersede" (or equivalent). Verified via `grep -L 'no MCP exposes supersede' cmd/till/main.go internal/adapters/mcp_common/mcp_surface.go` returning both paths (no match).
- `mage ci` passes (drop-end verification).

## Notes

**Planner answers to REVISION_BRIEF §7 questions:**
- **Q1 (R5 direction)**: option (a) confirmed. Domain alias normalization + schema enum update.
- **Q2 (R6.1)**: rename (option a). Enrichment would duplicate R7 work.
- **Q3 (R8 shape)**: `operation=supersede` on `till.action_item`. Not a new field on `update`.
- **Q4 (R7.4)**: split into `dispatcher_e2e_test.go`. Clean extraction point at line ~509 in current `subscriber_test.go`.

**Round-2 deviations from round-1 plan (all per dev-approved brief):**
- **D1.5 scope corrected**: adapter method EXISTS at `app_service_adapter_mcp.go:1051-1075` (Drop 4c.5 B.1). D1.5 now scopes to interface-addition only; no adapter re-implementation.
- **D1.5 role-gate removed**: orchestrator-role check belongs at D1.6's `authorizeMCPMutation` call, not at the adapter layer.
- **D1.4 escape clause removed**: "direct monitor unit coverage acceptable if broker chain impractical" clause removed. Broker chain end-to-end is the REQUIRED default. Direct-monitor fallback requires dev sign-off in the worklog.
- **D1.3 goleak wired**: dev approved `go get go.uber.org/goleak`. Builder wires `goleak.VerifyTestMain(m)` in new `dispatcher_e2e_test.go`; falls back to `VerifyNone(t)` at test-body end if `TestMain` causes sibling-test goroutine inflation.
- **D1.6 action string corrected**: `"supersede_task"` not `"supersede_action_item"`.
- **D1.6 args struct addition**: `Reason *string` field and `mcp.WithString("reason", ...)` schema declaration added — required for `bindArgumentsStrict` to accept the argument.
- **D1.6 failed-only invariant test added**: 5th test case pinning `ErrTransitionBlocked` for non-failed items at the MCP boundary.
- **D1.2 description-string acceptance tightened**: explicit bullet pins the new description string (not just enum values).
- **D1.2 legacy-token rejection acceptance added**: confirms `"branch"`, `"phase"`, `"subtask"`, `"decision"`, `"note"` are rejected at domain layer.
- **F5 accepted as stub-only**: real `dispatcherTemplateResolver` lives in `package main` and is not importable from `internal/app/dispatcher`. Stub-only coverage accepted; production-resolver test deferred to a future refinement in `cmd/till/main_test.go`.
