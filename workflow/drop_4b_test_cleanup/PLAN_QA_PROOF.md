# Plan QA Proof — Round 1 — Drop 4b Test Cleanup

**Verdict:** `pass-with-findings` (5 findings; none block plan acceptance, but D1.5 needs description tightening before build).

## 1. Acceptance-Criteria Coverage (Brief §5 A1–A6 → Droplets)

Trace from each acceptance criterion to a droplet, verified by reading the plan + cited symbols.

- **A1** (`mage ci` green) → implicit per WORKFLOW.md Phase 6 + restated in D1.6 acceptance ("mage ci passes (drop-end verification)"). Covered.
- **A2** (R5 — `target_type=actionItem` succeeds + regression tests on each schema-enum value) → D1.1 (domain alias + table-driven tests on `actionItem` / `ActionItem` / `ACTIONITEM`) + D1.2 (MCP schema enum shrunk to `project`, `action_item`, `actionItem`). Covered.
- **A3** (R6 — rename 6.1, goleak 6.2, state-transition 6.3) → D1.3. Covered modulo R6.2 deviation (see §4 below).
- **A4** (R7 — full-chain gate-fail 7.1, applyCleanExit chain 7.2, stub-param 7.3, file split 7.4) → D1.3 (7.4) + D1.4 (7.1/7.2/7.3). Covered modulo R7.1/7.2 deviation (see §4).
- **A5** (R8 — orch MCP supersede with role-gating + subtree-gating + happy path tests) → D1.5 (interface + adapter) + D1.6 (tool registration + tests). Covered but mis-layered — see Finding 1.1.
- **A6** (no regression on 3379 baseline) → implicit per each droplet's `mage test-pkg` acceptance. Covered.

Sub-refinement coverage: R5, R6.1, R6.2 (deferred), R6.3, R7.1 (deviated), R7.2 (deviated), R7.3, R7.4, R8 — every sub-refinement mapped to a droplet. No gaps.

## 2. Symbol-Existence Verification

Spot-checked every concrete symbol the plan names:

- 2.1 `NormalizeCommentTargetType` — `internal/domain/comment.go:143`. Confirmed. Current body matches plan's description (case-insensitive range against `validCommentTargetTypes`, no alias step).
- 2.2 `validCommentTargetTypes` — `internal/domain/comment.go:22`. Confirmed — contains only `CommentTargetTypeProject` (`"project"`) and `CommentTargetTypeActionItem` (`"action_item"`). The plan's claim that `strings.ToLower("actionItem")` = `"actionitem"` fails to match `strings.ToLower("action_item")` = `"action_item"` is correct — the alias step is genuinely needed.
- 2.3 `registerCommentTools` schema — `internal/adapters/mcp_rpc/extended_tools.go:2232`; the `mcp.Enum(...)` call lives at line 2243 with the exact stale tokens the plan names. Confirmed.
- 2.4 `TestAutoDispatchE2EGatePassViaNewDispatcher` — `internal/app/dispatcher/subscriber_test.go:543`. Confirmed exactly.
- 2.5 `TestAutoDispatchE2EGateFailViaNewDispatcher` — `subscriber_test.go:625` (declaration; doc-comment starts at 616). Plan says "lines 616-666" which is the full block including doc. Acceptable.
- 2.6 `stubE2ETemplateResolver` — `subscriber_test.go:513` (type decl) through line 522 (`GetProjectTemplate` method end). Plan says "lines 509-522" which includes the leading doc-comment at 509. Acceptable.
- 2.7 `SupersedeActionItemRequest` — `internal/adapters/mcp_common/mcp_surface.go:354`. Confirmed exactly. Doc comment notes "no MCP tool registration exposes supersede so agent-driven flows cannot reach it" — confirms gap D1.6 fills.
- 2.8 `app.Service.SupersedeActionItem` — `internal/app/service.go:1815`. Confirmed exactly.
- 2.9 `till.action_item` operation enum — `internal/adapters/mcp_rpc/extended_tools.go:1440`. Confirmed exact contents `("get", "list", "search", "create", "update", "move", "move_state", "delete", "restore", "reparent")`.
- 2.10 `ActionItemService` interface — `internal/adapters/mcp_common/mcp_surface.go:848`. `ReparentActionItem` is the last actionItem-mutation method at line 857. Confirmed — `SupersedeActionItem` is NOT currently on the interface, so D1.5 will add it.

All symbols verified. No drift between plan claims and tree.

## 3. File-Disjointness + `blocked_by` Integrity

DAG walk:
- D1.1 → no deps. Package `internal/domain`, sole droplet in package.
- D1.2 → no deps. Package `internal/adapters/mcp_rpc`, file `extended_tools.go`.
- D1.3 → no deps. Package `internal/app/dispatcher`, files `subscriber_test.go` + new `dispatcher_e2e_test.go`.
- D1.4 → `blocked_by: [1.3]`. Package `internal/app/dispatcher`, file `dispatcher_e2e_test.go`. Shares package + file with D1.3. Correctly serialized.
- D1.5 → no deps. Package `internal/adapters/mcp_common`. Disjoint from all others.
- D1.6 → `blocked_by: [1.2, 1.5]`. Package `internal/adapters/mcp_rpc`, file `extended_tools.go` (shared with D1.2 — correctly serialized) + depends on D1.5's interface change (correctly serialized).

`_BLOCKERS.toml` mirrors PLAN.md inline `Blocked by:` bullets exactly (D1.4 ← D1.3; D1.6 ← D1.2, D1.5). No drift, no cycles, no missing edges.

Cross-droplet file-sharing audit: every (file, package) overlap is covered by a `blocked_by` edge. No silent parallel-edit conflicts.

## 4. Planner Deviations from Brief — Justification Review

The planner flagged two deviations in the Notes section. Both are reasonable.

- **R6.2 deferral (goleak → comment-only)** — justified. Adding `runtime.NumGoroutine()` brackets in `t.Parallel()` suites is genuinely unreliable; the brief explicitly listed `goleak` as an option but noted "may need adding to `go.mod`." Planner correctly chose comment-only and surfaced the dev question. **Accept.**
- **R7.1/7.2 scope shrink (direct `processMonitor` construction allowed)** — justified. The real chain spawns an external `claude` subprocess; reaching `applyCleanExitTransition` through it in-test requires either a fake processMonitor wired into the dispatcher (which the current `Options` shape doesn't expose) or a much heavier integration harness. The planner correctly identified this and surfaced the dev question. Builder worklog requirement to honestly describe scope is the right discipline. **Accept.**

Both deviations are surfaced as Open Questions in the plan's Notes section. Dev should confirm during Phase 3 discussion before build starts.

## 5. Acceptance-Criterion Verifiability

Every droplet's acceptance is yes/no-checkable by `mage test-pkg <pkg>` + grep on a named symbol. Spot check:

- D1.1 — `NormalizeCommentTargetType("actionItem") == "action_item"` (table-driven test assertion). Verifiable.
- D1.2 — schema enum contains `"action_item"` + excludes `"branch"` etc. Verifiable by grep on the call site.
- D1.3 — rename existence (`TestAutoDispatch_NewDispatcherGateWiring` present, old name absent); file existence; symbol-move check. All verifiable.
- D1.4 — three test names present; `tplByProject` field on `stubE2ETemplateResolver`. Verifiable.
- D1.5 — `ActionItemService` interface includes `SupersedeActionItem`; compile-check via `mage test-pkg`. Verifiable. (Caveat: see Finding 1.1.)
- D1.6 — `"supersede"` in operation enum at line 1440; `Reason *string` in args struct; table-driven tests. Verifiable.

No fuzzy acceptance criteria.

## 6. Out-of-Scope Discipline

Scanned the full plan for forbidden content:
- R1/R2/R3/R4 — not referenced. ✓
- D5 implementation redesign — not referenced; the plan touches D5's tests only. ✓
- New gate kinds — not introduced. ✓
- New MCP surfaces beyond R5 (`till.comment` schema) + R8 (`till.action_item supersede`) — none. ✓

Out-of-scope discipline holds.

## 1. Findings

- **1.1 [Axis: spec-conformance] [severity: medium]** D1.5 description claims `AppServiceAdapter.SupersedeActionItem` needs to be implemented, but the method **already exists** at `internal/adapters/mcp_common/app_service_adapter_mcp.go:1051` (Drop 4c.5 droplet B.1 wired it). The real structural gap is on the `ActionItemService` interface at `mcp_surface.go:848-861` — `SupersedeActionItem` is not listed there. → Tighten D1.5's "Changes" to: (a) **add** `SupersedeActionItem(context.Context, SupersedeActionItemRequest) (domain.ActionItem, error)` to the `ActionItemService` interface after `ReparentActionItem`; (b) **verify** `AppServiceAdapter` already satisfies the new interface method (it does — already implemented); (c) add the lifecycle_test cases the plan describes against the existing method. Builder should NOT re-implement the adapter body.
- **1.2 [Axis: spec-conformance] [severity: medium]** D1.5 step 5 says "if `principal_role != 'orchestrator'`, return `ErrAuthorizationDenied`" — this introduces a role-gating check at the adapter layer that is inconsistent with the surrounding pattern. The existing `MoveActionItemState`, `ReparentActionItem`, and `SupersedeActionItem` adapter methods (lines 1000, 1115, 1051 respectively) use only `withMutationGuardContext` + `assertOwnerStateGate` (STEWARD-owner check) — they delegate role-gating to the MCP-layer `authorizeMCPMutation` call. The brief's "orch-self-approval gating" lives at the MCP-RPC layer via `authorizeMCPMutation` action strings, not at the adapter layer. → Remove step 5 from D1.5; the role gate lives in D1.6 via `authorizeMCPMutation(ctx, ..., "supersede_action_item", ...)`. Update the D1.5 "Non-orchestrator session → `ErrAuthorizationDenied`" test case to instead assert STEWARD-owner-gate behavior (matching the existing `assertOwnerStateGate` pattern in `MoveActionItemState`), and move the orchestrator-role gating tests to D1.6 where the MCP authorize call lives.
- **1.3 [Axis: spec-conformance] [severity: low]** D1.6 step 3 uses authorize-action string `"supersede_action_item"`, but every existing `till.action_item` mutation case in `extended_tools.go` uses the `_task` suffix (`reparent_task`, `restore_task`, `move_state_task` etc.). → Pick `"supersede_task"` for consistency with the surrounding `*_task` naming, OR document the deviation if the planner chose `_action_item` intentionally (post-Drop-1.75 vocab-modernization signal). Either way, surface the choice explicitly in the droplet description so the builder doesn't guess.
- **1.4 [Axis: acceptance-criteria-coverage] [severity: low]** D1.2's acceptance does not include the converse-coverage check: confirm that domain-layer validation (`NormalizeCommentTargetType`) still rejects values the schema enum excludes (e.g. `"branch"` from a legacy caller). After D1.1 + D1.2 land, a regression test that `IsValidCommentTargetType("branch") == false` would pin the schema/runtime alignment claim end-to-end. → Add one bullet to D1.2 acceptance: "regression test in `internal/adapters/mcp_rpc` (or domain, builder picks) confirming legacy tokens (`branch`, `phase`, `subtask`, `decision`, `note`) are rejected by the comment-create path at the MCP boundary."
- **1.5 [Axis: acceptance-criteria-coverage] [severity: low]** D1.6 acceptance includes "subtree-gating (via not-found)" but no droplet directly tests the lease-subtree scope path. `authorizeMCPMutation` does call into the auth-context store, which scopes by approved-path — so non-subtree calls do fall through as `auth_denied` rather than `not_found`. → Either rename the test case `"Out-of-subtree action item → auth_denied"` (and reproduce the lease-subtree-mismatch fixture pattern used elsewhere in `extended_tools_test.go`), or explicitly note that subtree-gating is exercised transitively via `authorizeMCPMutation`'s existing test surface and this droplet only adds the role-gating + happy-path tests.

## 2. Missing Evidence

- **2.1** None. Every symbol the plan cites is verified at the named file:line.

## 3. Summary

Verdict: **pass-with-findings** (3 medium + 2 low — none block plan acceptance).

The plan correctly decomposes R5/R6/R7/R8 into 6 atomic droplets with clean file-disjointness and a sound `blocked_by` DAG. Acceptance-criterion coverage is complete; the two planner-flagged deviations (R6.2 goleak, R7.1/7.2 scope) are justified and surfaced for dev confirmation. The five findings are description-quality issues, not structural decomposition flaws:

- 1.1 and 1.2 (medium) need D1.5's "Changes" rewritten to reflect that the adapter method already exists and to remove the spurious adapter-layer role-gate. Without this fix the builder will either (a) be confused about whether to rewrite the existing method, or (b) introduce a redundant role check that diverges from the surrounding pattern.
- 1.3 (low) is a naming-convention nit on the authorize-action string.
- 1.4 and 1.5 (low) tighten test coverage on the edges of the R5 + R8 acceptances.

Recommend round-2 plan revision to address 1.1 + 1.2 explicitly; 1.3/1.4/1.5 can be absorbed inline by the builder per the NITs-first-class rule but are easier to land in the plan.

## TL;DR

- **T1** A1–A6 fully covered across D1.1–D1.6; every sub-refinement (R5, R6.1/6.2/6.3, R7.1/7.2/7.3/7.4, R8) mapped to a droplet.
- **T2** All 10 cited symbols verified at the named file:line. No description drift.
- **T3** File-disjointness clean; `blocked_by` DAG acyclic (D1.4→D1.3; D1.6→D1.2,D1.5); `_BLOCKERS.toml` mirrors PLAN.md exactly.
- **T4** Two planner deviations (R6.2 deferred to comment, R7.1/7.2 allows direct-monitor construction) are justified and routed to dev via Open Questions.
- **T5** Every droplet acceptance is `mage test-pkg` + grep verifiable; no fuzzy criteria.
- **T6** Out-of-scope discipline holds — no R1/R2/R3/R4, no D5 redesign, no new gate kinds, no MCP surfaces beyond R5/R8.
- **T1-findings** Verdict pass-with-findings (3 medium + 2 low). 1.1 + 1.2 (D1.5 adapter method already exists; role-gate mis-layered) are the load-bearing fixes for round-2 revision.
