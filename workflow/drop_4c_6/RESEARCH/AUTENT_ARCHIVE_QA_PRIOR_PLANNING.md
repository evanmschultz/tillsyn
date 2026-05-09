# Autent / Archive / QA Prior-Planning — Code-State Truth (2026-05-08)

Read-only investigation into the dev's verbatim concerns ahead of W11 (runtime fail-loud) + team-aware architecture scoping. All citations are `path:line` against the working tree (some files dirty per `git status`; the file paths themselves are stable).

---

## A. Autent Role-Gating Coverage

### A.1 What is "autent" in this repo

"autent" is the dev's shorthand for the auth/role-gating system. **The package `internal/app/autent/` does not exist.** Auth is split across two layers:

1. **External `github.com/evanmschultz/autent` library** — imported as `autent`, `autentdomain`, `autentsqlite`, `autenttoken` at `internal/adapters/auth/autentauth/service.go:13-16`. Provides session-issuance, principal/client registry, rule-based ABAC policy evaluation.
2. **`autentauth` adapter** at `internal/adapters/auth/autentauth/service.go` — the tillsyn-side wrapper that drives the external library and bolts on tillsyn-specific concerns (steward principal type, approved_path scoping, auth_requests table).

A second tillsyn-internal layer enforces role-action coverage independently of autent:

3. **Capability-lease gate** at `internal/app/kind_capability.go:469-548` (`enforceMutationGuardAcrossScopes`) keyed off `domain.CapabilityRole` × `domain.CapabilityAction` matrix at `internal/domain/capability.go:232-292` (`DefaultCapabilityActions`).

### A.2 What dimensions does autent gate on today

Three independent dimensions, evaluated in order:

1. **Session validity** (autent layer). `Service.Authorize` at `internal/adapters/auth/autentauth/service.go:730-785` calls `s.service.Authorize(...)` against the external autent library. The library returns one of `allow | session_required | session_expired | grant_required | deny | invalid` based on rule evaluation. **Today's policy** is a single permissive rule installed by `EnsureDogfoodPolicy` at `service.go:135-169`: `effect=allow, actions=[*], resources={*,*,*}, priority=100`. Constant `DogfoodPolicyRuleID = "tillsyn-dogfood-allow-all"` at `service.go:26`. **No role gating happens at the autent layer in the dogfood configuration.** Any authenticated session is permitted any action against any resource.

2. **`approved_path` scope check** at `service.go:1276-1293` (`authorizeApprovedPath`) + `service.go:1295-1335` (`authContextPath`). When the issued session carries an `approved_path` metadata key (set during request approval, `service.go:468-498`), the gate parses it via `domain.ParseAuthRequestPath` and rejects with `approved_path_denied` if the mutation context's `(project_id, branch_id, scope_type, scope_id, phase_path)` does not lie *within* the approved subtree. This is project-rooted hierarchy scoping; not role gating.

3. **Capability-lease role-action match** at `internal/app/kind_capability.go:469-548`. Gated on `s.requireAgentLease` (default `true` at `service.go:166`). For an agent-principal mutation:
   - Looks up the lease by `AgentInstanceID` (line 487).
   - Validates lease identity, project match, action validity, role-action match (`lease.Role.CanPerform(action)` at line 508), revocation, expiry, and lease-scope coverage.
   - The role-action matrix at `domain/capability.go:232-292` is the canonical role gate.

### A.3 MCP-tool-level role gating today

The mutation-side helper `authorizeMCPMutation` at `internal/adapters/server/mcpapi/extended_tools.go:41-68` accepts a free-form `action` string ("create_action_item", "update_task", "create_comment", etc.) and forwards it to `MutationAuthorizer.AuthorizeMutation` → autent's rule engine. **Because the dogfood rule allows `*` for actions, the action string is never checked against caller role at the autent layer in the current configuration.** The capability-lease gate at the app/service layer is the only role-action enforcement live today.

Coverage of MCP tools enumerated in the prompt (sourced from `mcpapi/extended_tools.go` + `mcpapi/handler.go` + `mcpapi/handoff_tools.go` registrations):

| MCP tool | Code location | Role gate today |
|---|---|---|
| `till.action_item operation=create` | `extended_tools.go:896-1069` | `authorizeMCPMutation` (session+approved_path) → app `Service.CreateActionItem` → `enforceMutationGuardAcrossScopes(...CapabilityActionCreateChild)` at `service.go:1061`. **Builder/Research/QA roles ALL have `create-child` per `capability.go:249-291`** — no role narrowing today. |
| `till.action_item operation=update` | `extended_tools.go:1070-1179` | session+approved_path → `enforceMutationGuard(...CapabilityActionEditNode)` at `service.go:1530, 1566`. **Builder/Research/QA all have `edit-node`.** |
| `till.action_item operation=move` | `extended_tools.go:1180-1247` | session+approved_path → guard at `service.go:1285` with action resolved via `moveAction`. |
| `till.action_item operation=move_state` | `extended_tools.go:1248-1296` | same path; resolves to `MarkInProgress`/`MarkComplete`/`MarkFailed`. |
| `till.action_item operation=delete` (mode=archive\|hard) | `extended_tools.go:1297-1344`; impl `service.go:1766-1825` | session+approved_path → `enforceMutationGuardAcrossScopes(...CapabilityActionArchiveOrCleanup)` at `service.go:1782, 1801`. **`archive-or-cleanup` is granted only to `orchestrator` and `system` per `capability.go:235-291`** — builder/QA/research are correctly blocked. **STEWARD owner-state-lock** at `app_service_adapter_mcp.go:1164-1194` (`assertOwnerStateGate`) hardcodes `Owner == "STEWARD"` against `caller.AuthRequestPrincipalType == "steward"`. |
| `till.action_item operation=restore` | `extended_tools.go:1345-1388` | session+approved_path → app `Service.RestoreActionItem` (full chain unverified — no `enforceMutationGuard` call sighted in `RestoreActionItem`; this is a **gap finding**, see §B.4). |
| `till.action_item operation=reparent` | `extended_tools.go:1389-1433` | session+approved_path → app `Service.ReparentActionItem`. |
| `till.comment operation=create` | `extended_tools.go:2298-2345` | `authorizeMCPMutation` action="create_comment" → autent (allow-all). **NO role-vs-target check.** Any authenticated agent can comment on any action item. The capability-lease layer permits `comment` for all roles per `capability.go:235-291`. |
| `till.attention_item operation=raise/list/resolve` | `mcpapi/handler.go` (large surface; scattered) | session+approved_path; capability-lease for state mutations. |
| `till.handoff operation=*` | `handoff_tools.go:259, 321` | session+approved_path; standard mutation guard. |
| `till.auth_request operation=*` | `handler.go` | request/approve/claim/heartbeat/renew/revoke/revoke_all (`extended_tools.go:190-410`); session-issuance flow. |
| `till.capability_lease operation=*` | `extended_tools.go` (lease lifecycle around line 800-1000) | session-issued + dispatcher-coordinated. |
| `till.embeddings.*`, `till.kind.*`, `till.project.*`, `till.get_instructions`, `till.capture_state`, `till.get_bootstrap_guide` | `extended_tools.go` various; `instructions_tool.go`; `instructions_explainer.go` | read-mostly; mutation paths gate via `authorizeMCPMutation` + lease where applicable. |

### A.4 Is there a `delete` MCP tool? — YES

**Two delete surfaces exist**, both calling the same `tasks.DeleteActionItem` underneath:

1. **`till.action_item operation=delete`** with `mode: "archive" | "hard"`. Schema enum at `extended_tools.go:1443` (`mcp.Enum("get", "list", ..., "delete", ...)`). Mode parameter at `extended_tools.go:1482` (`mcp.Enum("archive", "hard")`). Handler dispatches at `extended_tools.go:1297-1344` to `tasks.DeleteActionItem(...)`.
2. **`till.delete_task`** legacy alias registered at `extended_tools.go:1578-1593`. Description: *"Delete one actionItem/work-item (archive or hard; legacy alias for till.action_item operation=delete)."* Same `mode` enum. Same handler via `handleActionItemOperation(ctx, req, "delete_task", "delete")`.

Both flow through `app.Service.DeleteActionItem(...)` at `internal/app/service.go:1766-1825`:

- `mode=archive` (default per `service.go:1768-1770`, default value `DeleteModeArchive` from `service.go:160-162`): calls `actionItem.Archive(now)` (sets `ArchivedAt` + `LifecycleState=StateArchived`) → `repo.UpdateActionItem` → `publishActionItemChanged`. **Soft delete.**
- `mode=hard`: calls `repo.DeleteActionItem` (row removal) + embedding-index cleanup + lifecycle cleanup. **Hard delete.**

**Both modes are MCP-callable today, gated only by `CapabilityActionArchiveOrCleanup` (orchestrator+system roles).** A builder/QA/research lease cannot call them — but a session without a lease (user-principal) can, and the dogfood autent rule does not narrow it further.

### A.5 Is `archive` MCP-callable, or domain-only?

**Both. `domain.ActionItem.Archive` exists at `internal/domain/action_item.go:618-624`** as an in-memory mutation method. **`till.action_item operation=delete mode=archive` is the MCP path that calls it indirectly** (via `Service.DeleteActionItem` mode-archive branch at `service.go:1772-1791`). **There is no separate `till.action_item operation=archive` tool** — archive is folded into `operation=delete`'s `mode` parameter.

**Counter to the dev's expectation** (per concern #2 quote, "I as a user cant do that and that is a problem"): a user CAN archive via `till.action_item operation=delete mode=archive` or via `till.delete_task mode=archive` today — both are exposed MCP surfaces. What is currently missing is a dedicated TUI/CLI archive control; the MCP plumbing is already in place but capability-locked to orchestrator+system roles, plus STEWARD owner-state-lock for STEWARD-owned items.

The `SKETCH.md` W11 spec at `workflow/drop_4c_6/SKETCH.md:949, 955` proposes flipping these to **MCP-rejected for all actors**, on the rationale that archive (and any hard-delete) should be human-only via UI/CLI per `project_team_aware_architecture.md`. That is the planned future state, not today's state.

### A.6 Team-axis gap

There is **no team / team-membership / contributor-private-DB axis** today. Evidence:

- Autent `Principal` carries `ID`, `DisplayName`, `Type` (one of `user|agent|service|steward`) — see `internal/adapters/auth/autentauth/service.go:194-218` (IssueSession). No `Team` field, no membership table, no team-rule predicate.
- The capability matrix is single-axis (role) at `domain/capability.go:232-292` — no team membership, no per-team rules.
- The closest existing "team-like" gate is the **STEWARD owner-state-lock** at `internal/adapters/server/common/app_service_adapter_mcp.go:1158-1194`. It hardcodes `stewardOwner = "STEWARD"` (line 1162) and `stewardPrincipalType = "steward"` (line 1159). This is a single-owner check on action items where `item.Owner == "STEWARD"`; it generalizes only by string equality and is not parameterized by team membership.
- Action-item `Owner` is a free-form string at `internal/domain/action_item.go:46-55` ("free-form string — trim-only, no closed-enum membership check"). Today's only concrete value is `"STEWARD"`.

**Concrete gaps for team scenarios** (per the dev's concern #1):

1. **No team registry or membership model.** Adding teams requires a new principal-membership table on the autent side OR a tillsyn-internal table that maps principals to teams.
2. **No team-axis rule predicate.** Autent rules today match on action + resource patterns; they do not match on principal-team. Adding team gating means either custom autent rules referencing principal metadata or a tillsyn-side gate that runs after `autent.Authorize` returns `allow`.
3. **No team-scope lease subtree.** Capability leases bind to `(project, scope)`; they do not carry team identity.
4. **STEWARD gate is single-owner.** Generalizing to "owner is one of these N principals" requires parameterizing `assertOwnerStateGate` away from the hardcoded constant. Today's owner-state-lock is a specialization, not a substrate.
5. **No archive-by-owner check.** Archive is gated on capability role only; it does not check whether the caller is in the owner's team. The team-aware architecture memo proposes archive be human-only via UI/CLI; that decision sidesteps the team-membership gap rather than closing it.

---

## B. Action-Item Delete + Archive Surface

### B.1 MCP-exposed delete surface

Per §A.4 above:
- `till.action_item operation=delete` (`extended_tools.go:1297-1344, 1443, 1482`) — modes `archive | hard`.
- `till.delete_task` (`extended_tools.go:1578-1593`) — legacy alias, same modes.

Both are MCP-callable today, gated by capability-lease role membership + autent session validity + approved_path scope.

### B.2 MCP-exposed archive surface

There is **no dedicated `archive` operation**. Archive is reachable only as `delete` with `mode=archive`. Restore is the inverse:
- `till.action_item operation=restore` (`extended_tools.go:1345-1388`).
- `till.restore_task` (`extended_tools.go:1595-1609`) — legacy alias.

### B.3 Domain `Archive` method

`internal/domain/action_item.go:618-624`:
```
func (t *ActionItem) Archive(now time.Time) {
    ts := now.UTC()
    t.ArchivedAt = &ts
    t.LifecycleState = StateArchived
    t.UpdatedAt = ts
}
```
Called from one MCP path (via `Service.DeleteActionItem mode=archive` at `service.go:1785`) and from internal tests/fixtures. Direct callers from non-app/Go-code paths: **none surfaced in this investigation** — the method is not invoked outside `service.go:1785` in the production source tree.

### B.4 Restore path role-gate gap (NEW finding)

`till.action_item operation=restore` at `extended_tools.go:1345-1388` flows through `tasks.RestoreActionItem(...)` → `app.Service.RestoreActionItem`. Investigation did not surface a `enforceMutationGuardAcrossScopes` call in `RestoreActionItem`'s body (the `rg` results listed only the seven enforce-call sites at `service.go:1061, 1285, 1425, 1494, 1530, 1566, 1782`; line 1782 covers the `delete` archive branch and 1801 the hard-delete branch, but nothing in the restore branch surfaced). **This may mean restore has weaker role-gating than archive** — verifying this is a future-drop concern, not in scope for this investigation. Routing as an open question for orchestrator review (could land as a refinement entry).

### B.5 Verification of dev's expectation

Dev's expectation per concern #2: *"delete should NOT exist in MCP; archive should be human-only (UI/CLI not MCP)."*

**Refuted by current code** — both delete and archive ARE MCP-callable today. **Confirmed by the SKETCH.md W11 plan** — `workflow/drop_4c_6/SKETCH.md:949` and `:955` explicitly propose rejecting `till.action_item.delete` and MCP archive at the boundary, matching the dev's expectation. So today's reality and the planned-W11 reality are mismatched; the dev's concern is correctly diagnosing the mismatch.

The dev's note *"I as a user cant do that and that is a problem"* likely refers to a missing TUI/CLI archive control, NOT a missing MCP path — the MCP path is present and functional, it is the human-facing surface that is missing.

---

## C. Prior QA-Refusal + Auto-Archive Planning Recovery

### C.1 Prior planning location

The prior planning the dev recalls is **on disk in `workflow/drop_4c_6/SKETCH.md`** — it has not been lost. Specifically:

- §11.2 "Failed-QA handling — system-managed wipe-and-restart with synthesized failure context (MVP)" at `SKETCH.md:376-424`.
- W10 work item summary at `SKETCH.md:675` (table row).
- W11 work item ("RUNTIME FAIL-LOUD ON TEMPLATE-ENFORCEMENT VIOLATIONS") at `SKETCH.md:676` and the v2.8.1 detailed spec at `SKETCH.md:944-974`.
- Parent-summary update markers at `SKETCH.md:14, 17, 22` documenting the §11.2 rework.

### C.2 What the spec says

**Trigger:** when `plan-qa-proof` or `plan-qa-falsification` returns a failure verdict, the parent plan transitions to `failed`. Same pattern applies to `build-qa-*` failures on a `build` parent (`SKETCH.md:405`).

**System-managed flow** (`SKETCH.md:386-396, 416-425`):

1. New service method `Service.WipeChildrenAndRePlan(parent_id)`:
   - Step 1: collect QA failure findings from the failed QA-twins' closing comments.
   - Step 2: synthesize a `failure_context` summary string from the findings.
   - Step 3: archive ALL non-archived children of the parent atomically (single transaction). `LifecycleState = archived`. **NEVER hard-delete.** Audit trail preserved.
   - Step 4: append `failure_context` to `parent.metadata.failure_history` — a NEW typed field on `ActionItemMetadata` supporting 1-N entries (multiple wipe cycles).
   - Step 5: transition parent back to `in_progress`.

2. Render layer (W3 + W8 hooks): when assembling the fresh planner-spawn's system-prompt.md, include a "Prior Attempt Failed" section synthesized from `metadata.failure_history[<latest>]`. MVP framing is hardcoded prose; future drops add a template `[failed_plan_prompt_template]` field for adopter customization.

3. Fresh planner spawn: planner reads system-prompt + own action-item content. **Planner is BLIND to archived children** — they are NOT in its preloaded context, and the planner's prompt explicitly forbids reading them via MCP (`SKETCH.md:381-383, 392, 411`).

4. Fresh planner authors fresh build / sub-plan / research children with corrected decomposition. **Planner does NOT create or touch QA-twins.**

5. Template `[[child_rules]]` auto-fire on each new child creation: fresh `plan-qa-twins` on any new sub-plan, fresh `build-qa-twins` on any new build. System-managed; planner uninvolved.

6. Cascade dispatcher fires the new children + their fresh QA-twins. Cycle continues.

**Critical principles** (`SKETCH.md:378-383`):

- Planner agents NEVER write or affect QA action items in default templates. QA-twin lifecycle is system-managed via template `[[child_rules]]`.
- The new planner does NOT see archived children at all. No reading. No reference. No partial revival. Cleanest possible reset.
- The system synthesizes a failure-context prompt section; the planner authors fresh decomposition informed ONLY by (a) the parent plan's current state + (b) the system-supplied failure context.

**Why "blind to archived children"** (`SKETCH.md:398-403`): simplest reset; saves tokens net (synthesis ~200-500 tokens vs full archived-child dump ~2000-5000); prevents missing-things via cherry-pick cognitive load; template-customizable post-MVP for adopters who want surgical revival.

**N-failure escalation:** the SKETCH does NOT spec an N-failure-then-escalate-to-human rule. The implicit model is "wipe-and-replan loops indefinitely until a planner produces a passing decomposition." If the dev wants an N-failure escalation, that is a new item to add. Investigation did not surface any prior planning at all for an N-attempt cap or an attention-item escalation on repeated QA failure.

### C.3 Has any of it been built? — NO

`Service.WipeChildrenAndRePlan` does not exist in the source tree. Evidence:

- `rg "WipeChildrenAndRePlan" internal/` returns zero hits in production code (one match in `SKETCH.md:417` is the spec itself).
- `metadata.failure_history` is not a typed field on `ActionItemMetadata` today — `internal/domain/action_item.go` does not declare it (the metadata struct lives around line 200-240 in the `ActionItemMetadata` definition; no `FailureHistory` field surfaced).
- `failure_context` synthesis logic does not exist anywhere in `internal/app/`.
- The render-layer hooks (W3 + W8) for fresh-spawn system-prompt.md do not yet include a "Prior Attempt Failed" section — those waves are W11-adjacent unplanned work.

### C.4 Comment-restriction prior planning — NONE

Investigation found **no prior planning** in any `workflow/drop_*/PLAN.md`, `CLOSEOUT.md`, `REFINEMENTS.md`, or `SKETCH.md` for the dev's concern #4 ("qa should ONLY comment on their own action items"). The W11 SKETCH spec at `SKETCH.md:944-974` lists structural rejects (planner cannot create QA, builder cannot decompose, no actor deletes, no MCP archive) but **does not include a "QA can only comment on its own action item" rule.** This is a gap the dev's concern surfaces fresh; it has not been previously specced.

---

## D. Comment Semantics

### D.1 Domain shape

`internal/domain/comment.go:35-47`:
```
type Comment struct {
    ID, ProjectID, TargetType, TargetID, Summary, BodyMarkdown string
    ActorID, ActorName string
    ActorType ActorType   // user | agent | system
    CreatedAt, UpdatedAt time.Time
}
```

Comments target either `project` or `action_item` (target-type enum at `comment.go:13-25`; scope-level distinctions removed alongside the 12-kind action-item collapse). Comments carry actor identity (`ActorID`, `ActorName`, `ActorType`) but **NO role field, NO target-author-role link, NO restriction by which target a given role can post on.**

### D.2 MCP create path

`till.comment operation=create` registered at `internal/adapters/server/mcpapi/extended_tools.go:2233-2363`. The handler:

1. Validates required args (`project_id`, `target_type`, `target_id`, `summary`).
2. Calls `authorizeMCPMutation(ctx, ..., "create_comment", "project:"+projectID, "comment", targetID, ...)` at `extended_tools.go:2303-2318`. Action passed to autent: `"create_comment"`. Resource: `(namespace="project:"+projectID, type="comment", id=targetID)`.
3. Calls `comments.CreateComment(...)` (no role narrowing in the request).

**There is NO check** that the caller's role matches the target action-item's role (e.g., "QA-role caller can only post on a target where target.role is also QA's-own-action-item"). The autent dogfood rule allows any action; the capability-lease layer permits `comment` for all roles per `domain/capability.go:235-291`.

### D.3 Capability-action coverage

`domain/capability.go:232-292` `DefaultCapabilityActions`:
- Orchestrator: `comment`, `read`, `create-child`, `edit-node`, `request-auth`, `approve-auth-within-bounds`, `mark-{in_progress, complete, failed}`, `resolve-attention`, `archive-or-cleanup`.
- Builder: `read`, `comment`, `create-child`, `edit-node`, `request-auth`, `mark-{in_progress, complete, failed}`, `attach-evidence`.
- QA: `read`, `comment`, `edit-node`, `request-auth`, `mark-{in_progress, complete, failed}`, `reopen`, `attach-evidence`, `signoff`, `resolve-attention`.
- Research: `read`, `comment`, `create-child`, `edit-node`, `request-auth`, `mark-{in_progress, complete, failed}`, `attach-evidence`.

**`comment` is granted to every role.** No role currently lacks comment authorship privileges. There is no per-target narrowing — a QA agent with a project-scoped lease can post a comment on ANY target in that project, including parent plans, sibling builds, and unrelated action items.

### D.4 Caller-identity audit-trail capability

The `Comment` row stores `ActorID` + `ActorType`, so post-hoc the system can identify who posted what — the audit trail is intact. But there is no **enforcement** that QA-role callers only post on their own action item; that is a future structural-reject rule that would need to be added to W11's hardcoded set OR plumbed through capability-lease scope (e.g., a `comment-on-own-action-item-only` action variant restricted by role).

### D.5 Verification of dev's expectation

Dev's expectation per concern #4: *"qa should ONLY comment on their own action items."* — **Not enforced today.** The plumbing for the rule does not exist; W11's current SKETCH spec does not include this rule (per §C.4 above); no prior drop has planned it. This is a new structural-rule candidate the dev's concern surfaces.

**Implementation cost estimate** (rough, not in scope but useful for planner sizing): a new `CapabilityActionCommentOnOwnActionItem` variant + a target-vs-lease-scope check in `authorizeMCPMutation`'s comment path that confirms `targetID == lease.ScopeID` (or a parent-chain lookup if the lease is action-item-scoped while the target is the same item). Probably ~50-80 LOC + tests.

---

## E. MCP Tool Inventory Verification

### E.1 Cross-check summary

Per the prompt's three claims:

| Claim | Verdict | Evidence |
|---|---|---|
| `till.action_item.delete` does NOT exist | **REFUTED.** Tool exists. | `extended_tools.go:1443` (`mcp.Enum(..., "delete", ...)`) + `:1297-1344` (handler) + `till.delete_task` legacy alias at `:1578-1593`. |
| `till.action_item.archive` exists (or doesn't); if exists, is it role-gated? | **DOES NOT EXIST as a separate tool.** Archive is folded into `delete` via `mode=archive`. The folded path IS role-gated via `CapabilityActionArchiveOrCleanup` (orchestrator+system only). | `extended_tools.go:1482` (mode enum), `service.go:1782` (guard call), `domain/capability.go:235-291` (matrix). |
| `till.action_item.restore` exists; is it role-gated? | **EXISTS** as both `till.action_item operation=restore` and legacy `till.restore_task`. **Role-gating is unverified** — investigation did not surface an `enforceMutationGuardAcrossScopes` call inside `Service.RestoreActionItem` (gap finding §B.4). | `extended_tools.go:1345-1388, 1595-1609`; service-layer enforcement: not surfaced. |

### E.2 Full MCP tool list (mcpapi/ exposure)

From the registration calls in `internal/adapters/server/mcpapi/`:

**Action-item surface** (`extended_tools.go`):
- `till.action_item` (unified, ops: `get|list|search|create|update|move|move_state|delete|restore|reparent`).
- Legacy aliases: `till.list_tasks`, `till.create_task`, `till.update_task`, `till.move_task`, `till.delete_task`, `till.restore_task`, `till.reparent_task`, `till.list_child_tasks`, `till.search_tasks`.

**Comment surface** (`extended_tools.go:2233`):
- `till.comment` (ops: `create|list`).

**Project surface** (`extended_tools.go` ~line 460-700):
- `till.project` (ops: `list|create|update|set_allowed_kinds|list_allowed_kinds|list_change_events|get_dependency_rollup`).
- (`get` operation is included — see `extended_tools.go:815`.)

**Auth + lease + handoff + attention + capture_state + instructions + embeddings + kind**:
- `till.auth_request` — ops: `request|claim|heartbeat|renew|revoke|revoke_all|list|issue` (`extended_tools.go:190-410`).
- `till.capability_lease` — ops scattered around `extended_tools.go:800-1000`.
- `till.handoff` — `handoff_tools.go:22-359`.
- `till.attention_item` — `handler.go` (raise/list/resolve/escalate).
- `till.capture_state`.
- `till.get_instructions` (`instructions_tool.go`).
- `till.get_bootstrap_guide`.
- `till.embeddings` (ops: `status|reindex|search`).
- `till.kind` (ops: `get|list|list_builtin|validate|set`).

### E.3 No `delete` exists OUTSIDE the `delete` operation on action_item

Cross-checked: searches for `Delete` in `mcpapi/extended_tools.go` returned only the `till.delete_task` registration + the `operation=delete` handler. No separate `till.delete_*` tools exist for projects, comments, attention items, handoffs, leases, etc. Project deletion in particular has NO MCP path today — projects are append-only via MCP.

### E.4 Comment-domain has no delete

`till.comment` exposes `create` and `list` only (`extended_tools.go:2243`, `mcp.Enum("create", "list")`). No `delete`, no `update`, no `restore`. Comments are append-only via MCP.

---

## TL;DR — Most Load-Bearing Findings

1. **Autent role-gating is bifurcated.** The autent layer is currently a `allow-all` dogfood rule (`autentauth/service.go:135-169`); real role gating happens at the capability-lease layer (`kind_capability.go:469-548`) keyed off `domain.CapabilityRole × CapabilityAction` (`domain/capability.go:232-292`). MCP tools call both. The action-string passed to autent (`"create_comment"`, `"delete_task"`, etc.) is descriptive metadata in the dogfood configuration — not enforcement.

2. **`till.action_item.delete` exists today** as both a unified-tool operation (`mode: archive|hard`) and a legacy `till.delete_task` alias. Mode=archive folds in archive. Both are role-gated to orchestrator+system via `CapabilityActionArchiveOrCleanup`. **W11's SKETCH spec proposes rejecting them at the MCP boundary** — that is the planned future state, not today's.

3. **Archive has no dedicated MCP tool** — it is `delete mode=archive`. Restore exists but its role-gating chain has a verification gap (§B.4) — a `enforceMutationGuardAcrossScopes` call inside `Service.RestoreActionItem` was not surfaced and may be a real gap.

4. **The QA refusal / auto-archive / failure-context flow IS specced** in `workflow/drop_4c_6/SKETCH.md:376-424` (§11.2) with concrete hooks (`Service.WipeChildrenAndRePlan`, `metadata.failure_history` typed field, archive-not-delete invariant, planner-blind-to-archived-children policy, `[[child_rules]]` auto-create on fresh decomposition). **None of it is built yet.** The render-layer (W3 + W8) and the W10 service method are unimplemented. **N-failure escalation is NOT specced** anywhere — it would be a fresh addition.

5. **Comment role-gating does not exist today.** `till.comment.create` permits any authenticated session with `comment` capability (every role has it) to post on any target — there is no "QA can only comment on its own action item" enforcement, and **no prior drop has specced one.** The dev's concern #4 is a fresh structural-rule candidate.

6. **No team / membership / contributor-private-DB axis exists** in autent or in tillsyn-internal layers. The closest analog is the STEWARD owner-state-lock at `app_service_adapter_mcp.go:1158-1194` — a hardcoded single-owner check, not a substrate for team gating. Adding teams requires (a) a membership model, (b) a team-axis rule predicate in autent, (c) team-scope leases, and (d) generalizing `assertOwnerStateGate` away from the `STEWARD` constant.

7. **Dev's mental model of "code already exists, just needs wipe-and-redo or DRY reuse"** (concern #2) is **partially correct**: the delete + archive + restore plumbing IS in place at the MCP and service layers. What's missing is (a) a dedicated TUI/CLI archive surface for users (the dev's "I as a user cant do that"), (b) the W11 fail-loud rejection of MCP delete/archive that would invert the current allow-with-role-gate posture, and (c) the W10 wipe-and-replan logic that would consume the existing archive primitive cyclically.

---

## Hylla Feedback

- **Query**: `hylla_search_keyword(query="autent", artifact_ref="github.com/evanmschultz/tillsyn@main")` and `hylla_search` with same scope.
- **Missed because**: artifact ingest was mid-enrichment at the time of the query — `enrichment still running for github.com/evanmschultz/tillsyn@main`. Hylla returned no results because it was not yet ready.
- **Worked via**: ripgrep (`rg -n "..."`) over the working tree + `Read` over specific files. Bash `grep`/`find` invocations were rejected by sandbox; ripgrep + Read did the job.
- **Suggestion**: when Hylla is mid-enrichment, the response could surface the most-recent-completed-snapshot timestamp + a one-line hint ("query against snapshot N which is from <timestamp>; latest enrichment in progress, retry in ~M minutes") so callers can decide to fall back deliberately rather than guessing.

- **Query**: `hylla_search_vector` with various field combinations.
- **Missed because**: parameter validation — `field must be summary, content, or docstring`. The `field` parameter was either missing or malformed in the call.
- **Worked via**: ripgrep + Read fallback.
- **Suggestion**: the error message could include the actual rejected value or the closest-match suggestion. Today's "field must be summary, content, or docstring" leaves the caller guessing whether they sent a typo or omitted the field entirely.

- **Ergonomic note**: Bash-tool sandboxing in the spawn context rejected `find` and several `grep -rn` invocations on `internal/` paths but allowed `rg -n` on the same paths. The asymmetry is a workflow friction. Not a Hylla concern per se, but flagging since it shaped which evidence-gathering tools were viable.
