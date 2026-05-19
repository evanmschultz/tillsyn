# Plan QA Proof — Round 2 — Drop 4b Test Cleanup

**Verdict:** `pass` (0 findings).

All round-1 PROOF findings dev-approved for closure are addressed by concrete, evidence-grounded round-2 plan changes. All four round-1 CONFIRMED falsification findings are addressed. The planner's unprompted `Reason *string` args-struct discovery is sound — verified against the current `handleActionItemOperation` body and `bindArgumentsStrict` strict-decode behavior. `_BLOCKERS.toml` unchanged; locked-decision conformance holds; action-string convention claims line up with the cited file:lines.

## 1. Round-1 Finding Closure (PROOF side)

Each round-1 PROOF finding mapped to round-2 plan change + evidence pointer.

- **1.1 (medium) → CLOSED.** D1.5 Scope section explicitly says "The adapter method `AppServiceAdapter.SupersedeActionItem` ALREADY EXISTS at `internal/adapters/mcp_common/app_service_adapter_mcp.go:1051-1075` (Drop 4c.5 droplet B.1). The `app_service_adapter_mcp.go` body is NOT modified by this droplet — any attempt to re-implement the method will introduce a compile error (duplicate function definition)." (PLAN.md:135). Acceptance bullet 3 reinforces: "`app_service_adapter_mcp.go` is NOT modified (no re-implementation of existing method)." (PLAN.md:152). Paths list at PLAN.md:129-131 contains `mcp_surface.go` + `app_service_adapter_lifecycle_test.go` only — `app_service_adapter_mcp.go` is NOT in paths. Verified: file body at lines 1051-1075 is the existing method, matching the round-2 description exactly.

- **1.2 (medium) → CLOSED.** D1.5 step 5 (adapter-layer orchestrator-role check) is DROPPED. PLAN.md:141 ("Role-gating note") explicitly states: "It does NOT include an orchestrator-role check at the adapter layer, which is correct: role-gating for this operation belongs at the MCP-RPC layer via `authorizeMCPMutation(\"supersede_task\", ...)` in D1.6. Builder must NOT add a role check to the adapter body." D1.6 acceptance bullet 6 (PLAN.md:189) confirms the role-gate moves to MCP layer: "non-orch → `auth_denied`".

- **1.3 (low) → CLOSED.** D1.6 step 4 (PLAN.md:173) uses action string `"supersede_task"` with cited convention: `"restore_task"` (line 1357), `"reparent_task"` (line 1401), `"delete_task"` (line 1309), `"create_task"` (line 931). Acceptance bullet 5 (PLAN.md:188) explicitly pins: "`authorizeMCPMutation` action string is `\"supersede_task\"` (not `\"supersede_action_item\"`)." Cited line numbers verified directly against `extended_tools.go` — see §6 below.

- **1.4 (low) → CLOSED.** D1.2 acceptance bullet 4 (PLAN.md:65) now includes: "Regression: `IsValidCommentTargetType(\"branch\")`, `IsValidCommentTargetType(\"phase\")`, `IsValidCommentTargetType(\"subtask\")`, `IsValidCommentTargetType(\"decision\")`, `IsValidCommentTargetType(\"note\")` all return `false` (domain layer enforces the shrunk enum). Builder adds these as table-driven cases in `internal/domain/comment_test.go` (or confirms they are implicitly covered by existing negative-case coverage) — the test confirming legacy-token rejection may live in either domain or mcp_rpc; builder picks the natural location."

- **1.5 (low) → CLOSED.** D1.6 test-case naming explicitly corrected (PLAN.md:180): "Non-subtree `action_item_id` (item belongs to a different auth scope) → `auth_denied`. Note: `authorizeMCPMutation` returns `auth_denied` for scope mismatches; this is NOT a `not_found`. The test case is named \"non-subtree action_item_id → `auth_denied`\", not \"subtree-gating via not-found\" (correcting round-1 wording per 1.5-proof)."

All 5 round-1 PROOF findings closed. None left dangling.

## 2. Round-1 Falsification Closure (CONFIRMED tier)

The dev approved closure of the 4 CONFIRMED findings (F1/F2/F3/F4). Round-2 changes:

- **F1 (HIGH — D1.5 misrepresented existing code) → CLOSED.** Same evidence as 1.1 above (PLAN.md:135 + acceptance bullet 3 at PLAN.md:152). Adapter method at lines 1051-1075 confirmed extant via direct Read. D1.5 paths list excludes `app_service_adapter_mcp.go`.

- **F2 (HIGH — D1.4 R7.1/R7.2 scope deflation) → CLOSED.** D1.4 R7.1 (PLAN.md:109) now reads: "Add a test using the same stub-walker pattern as those two existing tests: wire a walker stub returning a `KindBuild` action item, configure a `GateKindMageTestPkg` template (not registered by `NewDispatcher`), drive via `handleSubscriberEvent` (or the broker chain), assert `metadata.outcome=failure` on the action item after the monitor's `applyCleanExitTransition` fires. This is the end-to-end broker-chain test R7.1 requires. Builder documents the exact chain reached in the worklog. If the subscriber chain cannot be fully reached without a subprocess (builder discovers this at implementation time), builder MUST report in worklog and get dev sign-off before falling back to a direct-`processMonitor`-construction test — that fallback is NOT pre-authorized." The pre-authorized "direct monitor unit coverage acceptable" escape clause is GONE. Acceptance bullet 3 (PLAN.md:120) restates: "fallback to direct-monitor construction requires dev sign-off documented in the worklog." End-to-end broker-chain reach is REQUIRED.

- **F3 (MEDIUM — supersede-non-failed regression missing) → CLOSED.** D1.6 acceptance now lists 5 test cases (PLAN.md:177-181), including the new 5th case (PLAN.md:181): "Supersede on `todo`-state item (or `in_progress`, or `complete`) → typed error wrapping `ErrTransitionBlocked` with the `\"supersede only applies to failed items\"` message. This test pins the service-layer failed-only invariant (`service.go:1843-1845`) at the MCP boundary so a future refactor cannot silently weaken it for MCP callers." Acceptance bullet 6 (PLAN.md:189) restates the failed-only invariant requirement.

- **F4 (LOW — goleak deferred-comment) → CLOSED.** D1.3 R6.2 (PLAN.md:84) wires goleak. PLAN.md:78 records the dev-action prerequisite: "Before this droplet builds, the dev must run `go get go.uber.org/goleak` in the `main/` worktree shell. The goleak library is NOT in `go.mod` (confirmed). Builder blocks until the dep is available." R6.2 specifies `goleak.VerifyTestMain(m)` in a `TestMain` function in `dispatcher_e2e_test.go` (the new file created by R7.4), with documented fallback to `goleak.VerifyNone(t)` at test-body end if `TestMain` causes sibling inflation. Acceptance bullet 5 (PLAN.md:95) confirms: "Goleak is wired — either via `TestMain` + `goleak.VerifyTestMain(m)` or via `goleak.VerifyNone(t)` at end of each test body if `TestMain` causes sibling inflation. Builder worklog documents the approach taken."

All 4 CONFIRMED falsification findings closed.

## 3. Round-1 Falsification (POSSIBLE tier) — Disposition

POSSIBLE findings were optional. The planner addressed a subset; the rest are Notes-acknowledged or accepted under existing project rules.

- **F5 (MEDIUM — real-resolver limitation) → Notes-accepted.** D1.4 has a new "Notes (F5 — real-resolver limitation)" section (PLAN.md:115) explicitly: "`TestStubE2ETemplateResolverRoutesPerProject` only proves the test stub routes per-project correctly; it does NOT pin production-resolver behavior. The real `dispatcherTemplateResolver` at `cmd/till/main.go:2704` lives in `package main` and is not importable from `internal/app/dispatcher`. Coverage of the real resolver's per-project routing logic is not testable from this package. This limitation is accepted for this drop. A future refinement adds a test in `cmd/till/main_test.go` (where `dispatcherTemplateResolver` lives) asserting per-project routing via the real resolver." Round-2 deviations note at PLAN.md:210 reinforces. ACCEPTABLE — limitation surfaced explicitly with documented deferral.

- **F6 (MEDIUM — alias persistence round-trip drift) → Notes-accepted.** D1.1 has a new "Notes (F6 — alias persistence round-trip)" section (PLAN.md:42) explicitly addressing the round-trip risk with the no-migration-logic rule justification. ACCEPTABLE.

- **F7 (LOW — D1.2 description string drift) → CLOSED.** D1.2 description-string acceptance tightened. PLAN.md:64 now includes: "The `target_type` description string lists exactly `\"project|action_item|actionItem\"` and contains no stale pre-1.75 tokens."

- **F8 (LOW — hand-rolled role check) → MOOTED.** Since D1.5 step 5 was DROPPED entirely (per finding 1.2 closure), there is no hand-rolled role check to refactor. The MCP-layer `authorizeMCPMutation("supersede_task", ...)` call in D1.6 is the existing helper. ACCEPTABLE.

## 4. NEW: Unprompted Discovery — `args.Reason *string` Field

The round-2 planner discovered (not in REVISION_BRIEF or round-1 findings) that the `args` struct in `handleActionItemOperation` does not have a `Reason` field, and `bindArgumentsStrict` uses `DisallowUnknownFields` — so the `supersede` case CANNOT receive a `reason` argument without the args struct addition.

**Verification:**

- **`handleActionItemOperation` exists** at `extended_tools.go:740` (verified by direct Read). Args struct begins at line 742, ends at line 802. Args fields enumerated lines 743-801: `Operation`, `ProjectID`, `ParentID`, `Kind`, `Scope`, `Role`, `StructuralType`, `Owner`, `DropNumber`, `Persistent`, `DevGated`, `Paths`, `Packages`, `Files`, `StartCommit`, `EndCommit`, `ColumnID`, `Title`, `Description`, `Priority`, `DueAt`, `Labels`, `Metadata`, `ActionItemID`, `ToColumnID`, `Position`, `State`, `IncludeArchived`, `Query`, `CrossProject`, `States`, `Levels`, `Kinds`, `LabelsAny`, `LabelsAll`, `SearchMode`, `Sort`, `Limit`, `Offset`, `Mode`, `SessionID`, `SessionSecret`, `AuthContextID`, `AgentInstanceID`, `LeaseToken`, `OverrideToken`. **NO `Reason` field exists today.** Confirmed.

- **`bindArgumentsStrict` enforces `DisallowUnknownFields`.** Source at `internal/adapters/mcp_rpc/strict_decode.go:64-94`. Body at line 85-86: `dec := json.NewDecoder(bytes.NewReader(data))` then `dec.DisallowUnknownFields()`. Doc comment at lines 37-63 explicitly states: "any JSON key in the arguments object that does not map to an exported field on target produces a wrapped errUnknownField." Confirmed.

- **Implication is correct.** If D1.6 adds `mcp.WithString("reason", ...)` to the tool schema and the `supersede` case attempts to read it without adding `Reason *string` to the args struct, `bindArgumentsStrict` will reject the JSON with `invalid_request: unknown field "reason"` BEFORE the switch dispatches to the `supersede` case. The discovery is load-bearing.

- **Round-2 plan addresses it.** D1.6 step 3 (PLAN.md:172): "Add `Reason *string `json:\"reason\"`` field to the `args` struct inside `handleActionItemOperation` (pointer-sentinel consistent with the `Owner`, `Title`, `Description` etc. pointer pattern established in Drop 4c.5-A.1 and visible at line 750 of the current struct). The `bindArgumentsStrict` decoder rejects unknown JSON keys — adding `Reason` to the struct is REQUIRED before the `supersede` case can receive the argument." Acceptance bullet 3 (PLAN.md:186) restates: "The `args` struct inside `handleActionItemOperation` contains `Reason *string`."

The discovery is sound and the plan correctly incorporates the required structural change.

## 5. `_BLOCKERS.toml` Integrity

Read end-to-end. Two blocker entries (D1.4 → D1.3; D1.6 → D1.2, D1.5). Identical to round-1 baseline (no edges added, removed, or changed). PLAN.md inline `Blocked by:` bullets at lines 28 (D1.1: none), 52 (D1.2: none), 76 (D1.3: none), 105 (D1.4: 1.3), 133 (D1.5: none), 165 (D1.6: 1.2, 1.5) mirror `_BLOCKERS.toml` exactly. No drift.

## 6. Action-String Convention Verification

Planner cites `restore_task` at line 1357, `reparent_task` at line 1401, `delete_task` at line 1309, `create_task` at line 931 of `extended_tools.go`. Direct Read confirms each:

- `"delete_task"` at `extended_tools.go:1309` — confirmed (verified inside `case "delete":` block calling `authorizeMCPMutation`).
- `"restore_task"` at `extended_tools.go:1357` — confirmed (verified inside `case "restore":` block).
- `"reparent_task"` at `extended_tools.go:1401` — confirmed (verified inside `case "reparent":` block).
- `"create_task"` at `extended_tools.go:931` — confirmed (verified inside the create handler).

The `_task` suffix convention is consistent across all four cited mutation surfaces. D1.6's choice of `"supersede_task"` aligns with the convention.

## 7. Locked-Decision Conformance

Scanned PLAN.md for forbidden content:

- **R1/R2/R3/R4** — not referenced; locked-decisions section at PLAN.md:15 explicitly enumerates them as out of scope ("R1/R2/R3/R4 stay parked for a separate template-validation hardening drop later"). No droplet references them.
- **D5 implementation redesign** — the plan touches D5's tests only (D1.3 + D1.4). No `internal/app/dispatcher/` non-test files in paths.
- **New gate kinds** — none introduced.
- **New MCP surfaces beyond R5 + R8** — `till.comment` schema (R5) and `till.action_item operation=supersede` (R8) are the only MCP surface changes. No other MCP tool registrations modified.

Locked-decision conformance holds.

## 8. Symbol-Existence Spot Checks (Round-2 Additions)

Spot-checked the symbols newly added in round-2 plan text:

- 8.1 `handleActionItemOperation` — `extended_tools.go:740`. Confirmed.
- 8.2 `bindArgumentsStrict` — `internal/adapters/mcp_rpc/strict_decode.go:64`. Confirmed; uses `dec.DisallowUnknownFields()` at line 86.
- 8.3 Args struct line 750 (cited for pointer-pattern example: `Owner *string`) — confirmed at `extended_tools.go:750`.
- 8.4 `app_service_adapter_mcp.go:1051-1075` (existing adapter method) — confirmed at lines 1051-1075; signature `SupersedeActionItem(ctx context.Context, in SupersedeActionItemRequest) (domain.ActionItem, error)` matches D1.5's proposed interface addition exactly.
- 8.5 `ActionItemService` interface at `mcp_surface.go:848-861` — confirmed; `ReparentActionItem` is the last actionItem-mutation method at line 857, so insertion of `SupersedeActionItem` after it at line 858 is the correct placement.
- 8.6 `SupersedeActionItemRequest` at `mcp_surface.go:354` — confirmed.
- 8.7 `till.action_item` operation enum at `extended_tools.go:1440` — confirmed (current contents `("get", "list", "search", "create", "update", "move", "move_state", "delete", "restore", "reparent")`, no `supersede` yet — D1.6 adds it).

All cited symbols verified at named file:lines. No drift.

## 1. Findings

(none — round-2 plan is acceptance-ready)

## 2. Missing Evidence

- **2.1** None. Every claim in the round-2 plan is grounded in a verified file:line.

## 3. Summary

Verdict: **pass** (0 findings).

Round-2 PLAN.md cleanly addresses all 5 round-1 PROOF findings, all 4 CONFIRMED round-1 FALSIFICATION findings, and one POSSIBLE finding (F7). Two POSSIBLE findings (F5, F6) are explicitly Notes-acknowledged with documented deferral rationale; F8 is mooted by F1's closure.

The planner's unprompted discovery — that `handleActionItemOperation`'s args struct lacks a `Reason` field and `bindArgumentsStrict` enforces `DisallowUnknownFields` — is sound. Without this discovery, the `supersede` case would have failed at the JSON decode boundary before the switch could dispatch. D1.6 step 3 + acceptance bullet 3 correctly require adding `Reason *string` to the struct.

`_BLOCKERS.toml` is unchanged. Action-string convention claims (`restore_task` 1357, `reparent_task` 1401, `delete_task` 1309, `create_task` 931) verified at exact cited lines. Locked-decisions (out-of-scope R1/R2/R3/R4, no D5 redesign, no new gate kinds) hold.

Recommend Phase 4 build dispatch.

## TL;DR

- **T1** All 5 round-1 PROOF findings closed by concrete round-2 plan changes; evidence pointers verified at named file:lines.
- **T2** All 4 CONFIRMED round-1 FALSIFICATION findings closed; F7 (POSSIBLE) closed; F5/F6 (POSSIBLE) Notes-accepted; F8 mooted by F1 closure.
- **T3** NEW `Reason *string` args-struct discovery is sound — `handleActionItemOperation` at line 740 confirmed lacking the field; `bindArgumentsStrict` at strict_decode.go:86 confirmed enforcing `DisallowUnknownFields`; D1.6 step 3 + acceptance bullet 3 correctly add the field.
- **T4** `_BLOCKERS.toml` unchanged; PLAN.md inline `Blocked by:` bullets mirror exactly.
- **T5** Action-string convention cites verified: `delete_task` line 1309, `restore_task` line 1357, `reparent_task` line 1401, `create_task` line 931 — all confirmed in extended_tools.go.
- **T6** Locked-decision conformance holds — no R1/R2/R3/R4, no D5 redesign, no new gate kinds, MCP surface changes scoped to R5 (`till.comment`) + R8 (`till.action_item operation=supersede`) only.
- **T7** Symbol-existence spot checks (8 symbols verified at exact cited lines) — no description drift.
- **T8** Verdict pass with 0 findings. Recommend Phase 4 build dispatch.

## Hylla Feedback

N/A — Hylla MCP tools (`hylla_search_keyword`, `hylla_node_full`, `hylla_refs_find`, `hylla_graph_nav`) were not surfaced in my agent tool environment for this spawn (only `Read`, `Edit`, `Write`, `Bash`, `Context7` available). Fell back to `Read` against the planner's cited file:lines throughout. The orchestrator prompt explicitly directed Hylla-first usage but the tools weren't available — surfacing this here so the orchestrator can confirm Hylla MCP routing for plan-QA-proof agent spawns.

The fallback Read-with-precise-line-offset approach worked cleanly because the round-2 planner produced exact line cites for every claim (1051-1075, 740-802, 854-861, 1309, 1357, 1401, 931, etc.). This is itself a signal that the planner did good evidence-grounding work in this round.
