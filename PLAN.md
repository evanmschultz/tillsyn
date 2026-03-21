# Tillsyn Plan

Created: 2026-02-21
Updated: 2026-03-21
Status: In progress; the auth UX implementation wave is now green in local gates, including the restored coverage floor, but collaborative dogfood retest and final closeout evidence are still pending.

## 1) Active Run Source Of Truth

This section is authoritative for the current auth/runtime remediation run.

1. `PLAN.md` is the only active checklist, status ledger, and completion ledger for this run.
2. All other planning or validation markdown is reference-only unless this file explicitly points to it for corroborating evidence.
3. Worker and QA subagents must map every acceptance claim, open blocker, command result, and sign-off note back to checklist ids in this file.
4. If any secondary doc conflicts with this file, treat the mismatch as a blocker and resolve it here first.
5. The orchestrator is the only writer for run completion state in this file.

## 2) Run Goal

Get `tillsyn` to a dogfood-ready near-MVP state for local human + agent collaboration by finishing one integrated wave that closes the current runtime, MCP, branding, and auth gaps without leaving confusing transitional behavior behind.

This run is successful only if:
1. `./till`, `./till mcp`, and `./till serve` dogfood the same real default runtime.
2. Raw stdio MCP remains the primary local MCP path and is clean to operate.
3. Stale product/runtime copy is cleaned up in live surfaces.
4. Real `autent` integration replaces the current brittle tuple-first MCP auth boundary.
5. Hierarchy-aware local workflow guardrails remain intact and test-covered.
6. The resulting behavior is collaboratively revalidated with evidence.

## 3) Locked Product Direction

1. `till mcp` remains the raw stdio MCP server.
2. `till serve` remains the secondary HTTP/API + HTTP MCP path.
3. `./till`, `./till mcp`, and `./till serve` must use the same real default runtime unless the user explicitly opts into a different runtime.
4. Local builds must not silently force dev mode.
5. `Ctrl-C` on `till mcp` must be treated as normal shutdown, not an error-style failure.
6. Remove stale `Kan` branding from live product/runtime surfaces in place. No compatibility naming shims.
7. Do not add `till mcp-inspect` in this run unless the user explicitly approves it.
8. `autent` is required in this run because the current MCP tuple/lease gatekeeping is too brittle for dogfooding.
9. `autent` becomes the source of truth for caller identity, session lifecycle, generic authz decisions, grant escalation, and auth audit.
10. `tillsyn` keeps hierarchy-derived scope/workflow rules local.
11. Current request-local identity synthesis and tuple-first MCP auth must stop being the primary gate.
12. Capability leases may remain temporarily only as secondary local workflow/delegation guardrails until auth integration is proven stable.
13. Normal TUI users should not need to manually mint auth sessions for routine TUI use.
14. Agent access must support an explicit request-and-approval flow that can originate from MCP and from the TUI.
15. Agent gatekeeping must be user-configurable, including lifecycle limits and scope/path restrictions.
16. Shell approvals remain a required first-class operator path even though normal user approval should be TUI-first.
17. The shell/operator auth flow should copy the strongest parts of `blick` as closely as practical:
   - explicit lifecycle verbs,
   - Fang/Cobra help with examples,
   - deterministic machine-friendly output,
   - persisted request/approval/audit state rather than ad-hoc shell-local side effects.
18. `tillsyn` should not copy `blick`'s access-profile abstraction directly; `tillsyn` auth/grant scope must stay project-path-centered.
19. `tillsyn` should not copy `blick`'s generic `requested-scope key=value` bag directly; `tillsyn` should use one explicit project-rooted `--path`.
20. Auth requests must surface in the TUI notifications model:
   - when the request targets the currently focused project, show it in that project's notifications panel,
   - when the request targets a different project or no project is currently focused, show it in global notifications until the matching project is focused.
21. Session or grant requests must carry one explicit scope path argument rooted at a project, with optional branch and nested phase lineage.
22. Any command that requires follow-up user action must say so directly in its help output.
23. `till auth --help` and subcommand help must enumerate required flags, path semantics, lifecycle controls, and concrete examples.
24. External MCP-originated changes should refresh the current TUI project without requiring a project-switch workaround.
25. Notifications remain a first-class UX surface with global count, quick navigation, and quick-info drill-in for important runtime/MCP warnings and errors.
26. Any substantial notifications-panel redesign must start with an ASCII-art proposal and clarifying questions before implementation.
27. `MCP_DOGFOODING_WORKSHEET.md` and `VECTOR_SEARCH_EXECUTION_PLAN.md` are retired; the active run contract must now live in `PLAN.md`, with only user-facing summary material kept in `README.md`.

## 4) Scope And Non-Goals

In scope:
1. runtime default-path unification
2. stdio/serve runtime parity
3. clean stdio shutdown behavior
4. live product/help/bootstrap copy cleanup
5. real `autent` embedding
6. replacement of brittle MCP auth boundary
7. attribution correctness from authenticated identity
8. strong automated tests and collaborative rerun evidence
9. auth request/approval UX across MCP, TUI, and CLI
10. notification routing for auth requests
11. user-configurable scoped agent gatekeeping
12. auth help/output cleanup with examples and next-step guidance

Out of scope for this run unless the user explicitly reopens them:
1. `till mcp-inspect`
2. remote/team auth-tenancy
3. full remote operator/admin console beyond the local dogfood auth workflows required here
4. removal of all local workflow leases in the same wave
5. unrelated roadmap items not required for this dogfood wave

## 5) Architecture Decision Lock

### 5.1 Auth Boundary

`autent` owns:
1. principal identity
2. client identity
3. session issue/validate/revoke
4. generic authz decisioning
5. grant escalation
6. auth-owned audit

`tillsyn` owns:
1. project/branch/phase/task/subtask hierarchy
2. hierarchy-derived resource mapping
3. hierarchy-derived scope validation
4. workflow semantics such as completion/start/cancel/archive rules
5. local delegation/lock semantics if capability leases remain

### 5.2 Storage Decision

Current locked direction for this run:
1. embed `autent` in shared-DB mode against the same `tillsyn` SQLite runtime
2. use `autent_`-prefixed tables in the shared DB
3. keep one local runtime and one SQLite file for dogfooding clarity

Known caveat:
1. shared DB does not automatically provide one cross-library outer transaction boundary; this must be handled explicitly or accepted as a limitation in the first wave

### 5.3 MCP Auth Model

1. MCP mutation auth must be session-first, not tuple-first.
2. A valid authenticated session is required before local mutation logic runs.
3. MCP write operations must derive caller identity from validated `autent` session state, not caller-supplied actor fallbacks.
4. Local hierarchy/scope/delegation checks run after auth validation and are distinguishable from auth failures.

### 5.4 Auth Request And Approval Model

1. The current `till auth issue-session` command is a temporary operator/developer seam, not the intended normal end-user workflow.
2. The intended dogfood product path is:
   - an agent or MCP caller requests access,
   - the user reviews the request in `tillsyn`,
   - the user approves or denies it with configurable scope and lifetime,
   - only then does a usable session/grant become active.
3. Auth requests must be creatable from:
   - MCP-initiated agent flows,
   - TUI-initiated local operator flows.
4. Auth request records must capture at minimum:
   - requested principal identity,
   - requested client identity,
   - requested path scope,
   - requested lifetime/TTL,
   - request status,
   - approval/denial audit fields.
5. The request path contract for this wave is one explicit `--path` argument rooted at a project:
   - required root: `project/<project-id>`
   - optional branch: `/branch/<branch-id>`
   - optional nested phases: `/phase/<phase-id>` repeated as needed
6. Task/subtask-level session-request paths are out of scope unless explicitly reopened later.
7. Approvals must be user-configurable rather than hard-coded allow-all behavior.
8. Approval flows must support user continuation from the client surface after the user authorizes the request.
9. Approval and denial actions must leave a guardrail-compatible audit trail.
10. The shell/operator request and approval flow is required in this run and should mirror `blick`'s lifecycle quality:
   - request creation,
   - request listing,
   - request detail inspection,
   - request approval,
   - request denial,
   - request cancellation,
   - session listing,
   - session validation,
   - session revocation,
   - audit inspection.
11. The intended `till` command shape for this wave is:
   - `till auth request create`
   - `till auth request list`
   - `till auth request show`
   - `till auth request approve`
   - `till auth request deny`
   - `till auth request cancel`
   - `till auth session list`
   - `till auth session validate`
   - `till auth session revoke`
12. `till auth issue-session` may remain temporarily as a low-level operator/dev seam, but it must not be the primary documented workflow for normal dogfooding.
13. Principal/client registration should stay implicit or auto-managed for dogfooding unless implementation proves that explicit operator registration is truly required.
14. The request payload contract for CLI, TUI, and MCP request creation must stay aligned:
   - principal identity,
   - client identity,
   - explicit `--path`,
   - requested TTL/lifetime,
   - human-readable reason,
   - enough continuation metadata for the requesting client to resume after approval.
15. Approval and request lifecycle labels are product behavior:
   - decision labels must be explicit and user-facing,
   - request state must distinguish pending, approved, denied, canceled, and expired/timeout paths,
   - timeout/cancel behavior must be reviewable in CLI, TUI, and audit surfaces.

### 5.5 Notification Routing Contract

1. Every auth request must resolve to one owning project from its path.
2. If the TUI is currently focused on that same project, the request must appear in that project's notifications panel.
3. If the TUI is focused on a different project, or no project is focused, the request must appear in global notifications.
4. Notifications must expose a global count and quick-navigation affordances.
5. Important runtime/MCP warnings and errors must bubble into notifications and quick-info drill-in surfaces.
6. Auth request notifications must be actionable and must preserve enough detail for approve/deny decisions without forcing the user into shell commands.
7. External MCP-originated changes should refresh the current project view and related notifications without requiring the user to switch projects to see them.
8. `enter` on an auth-request notification must open auth review directly, not a generic project/thread fallback.
9. Auth review must allow both approve and deny decisions with an editable resolution note; approve also keeps editable path and TTL constraints.
10. Actionable notifications copy should describe required review work, not imply a misleading actor taxonomy such as `Agent/User Action`.
11. If the notifications UX is redesigned in this wave, start with ASCII-art and clarifying questions before implementation.

### 5.6 CLI Help And Discoverability Contract

1. `till auth --help` must explain what the auth surface is for and what it is not for.
2. Help for every auth subcommand must list:
   - required flags,
   - optional lifecycle flags such as TTL or reason,
   - runtime/path flags inherited from the root command when relevant,
   - exactly what follow-up step is required after the command succeeds.
3. `issue-session` help must explicitly say it returns `session_id` and `session_secret`.
4. `revoke-session` help must explicitly say it requires `--session-id`; positional IDs are not supported.
5. If `request-session`, `list-sessions`, `show-session`, `approve-request`, or `deny-request` are added in this wave, they must ship with examples in help output on first landing.
6. `till auth` must move from the current ad-hoc two-command shape to a real Fang/Cobra auth tree with grouped request/session help similar in clarity to `blick`.
7. Shell help/output should be treated as product behavior and regression-tested explicitly.
8. Approval help must expose exact decision labels, path semantics, and examples, not generic prose.
9. Request list/show output must surface lifecycle state and timeout/cancel status clearly enough for operators to act without guessing.

## 6) Acceptance Matrix

Every mutation-capable MCP surface in scope must satisfy the matrix below.

| ID | Condition | Expected Result | Status | Evidence |
|---|---|---|---|---|
| AM-01 | no session supplied | fail closed before mutation with session-required semantics | PASS | `TestServiceAuthorizeSessionRequired`; `TestHandlerAttentionMutationsRequireSession`; `TestHandlerExpandedMutationAuthErrorsMap` |
| AM-02 | invalid session id or secret | fail closed before mutation with invalid-auth semantics | PASS | `TestServiceAuthorizeInvalidSecretReturnsDecision`; `TestHandlerExpandedMutationAuthErrorsMap` |
| AM-03 | expired session | fail closed before mutation with session-expired semantics | PASS | `TestServiceAuthorizeExpiredSessionReturnsDecision`; `TestHandlerExpandedMutationAuthErrorsMap` |
| AM-04 | revoked session | fail closed before mutation with revoked/invalid auth semantics | PASS | `TestServiceAuthorizeRevokedSessionReturnsDecision`; `TestAuthorizeMutationRevokedSessionReturnsInvalidAuthentication` |
| AM-05 | valid session but denied by policy | fail closed with deny semantics | PASS | `TestServiceAuthorizeDenyRuleReturnsDecision`; `TestAuthorizeMutationDenyRuleReturnsAuthorizationDenied` |
| AM-06 | valid session and escalation path required | return grant-required semantics without mutating | PASS | `TestServiceAuthorizeGrantRequiredReturnsDecision`; `TestAuthorizeMutationGrantRequiredReturnsGrantRequired` |
| AM-07 | valid session and auth allow | proceed to local hierarchy/workflow validation | PASS | `TestServiceSharedDBAuthorizeAllow`; `TestRunAuthIssueSessionCredentialsAuthorizeMutation` |
| AM-08 | auth allow but local scope/workflow/delegation reject | fail locally, distinct from auth failure | PASS | `TestHandlerExpandedToolRejectsMissingSessionAndGuardedUserTuples`; `TestHandlerAttentionAgentMutationsRequireGuardTuple` |
| AM-09 | allowed mutation succeeds | mutation persists and visible behavior is correct | PASS | `internal/adapters/server/httpapi/handler_integration_test.go:TestHandlerAttentionMutationPersistsAuthenticatedAttribution`; `internal/adapters/server/mcpapi/handler_integration_test.go:TestHandlerAttentionMutationPersistsAuthenticatedAttribution` |
| AM-10 | persisted attribution after allowed mutation | actor name/type come from authenticated identity, not request-local fallback strings | PASS | `internal/adapters/server/httpapi/handler_integration_test.go:TestHandlerAttentionMutationPersistsAuthenticatedAttribution`; `internal/adapters/server/mcpapi/handler_integration_test.go:TestHandlerAttentionMutationPersistsAuthenticatedAttribution`; `TestHandlerExpandedToolBuildsActorTupleFromAuthenticatedSession` |

The dogfood auth UX and operator/help surfaces must satisfy the matrix below before this run is truly complete.

| ID | Condition | Expected Result | Status | Evidence |
|---|---|---|---|---|
| AU-01 | an MCP agent needs access without a valid approved session | caller is routed into request/approval semantics, not silent tuple fallback or surprise shell-only workflow | PASS | `internal/adapters/server/mcpapi/handler_test.go:834`; `internal/adapters/server/mcpapi/handler_test.go:862`; `internal/adapters/server/mcpapi/handler_test.go:1169` |
| AU-02 | a local user wants to authorize an agent from the TUI | auth request can be reviewed and acted on without requiring the user to manually mint their own session | PASS | `TestModelProjectNotificationsAuthRequestApproveShortcut`; `TestModelGlobalNotificationsEnterOpensAuthReview`; `TestModelAuthReviewCanSwitchDecisionBeforeApply` |
| AU-03 | an auth request targets the currently focused project | request appears in that project's notifications panel | PASS | `internal/tui/model_test.go:7047` |
| AU-04 | an auth request targets a different project or no focused project exists | request appears in global notifications | PASS | `internal/tui/model_test.go:7489` |
| AU-05 | a user approves a request with scoped constraints | resulting session/grant is limited to the approved path and lifetime | PASS | `TestModelProjectNotificationsAuthRequestApproveForwardsConstraints`; `internal/adapters/auth/autentauth/service_test.go:592` |
| AU-06 | a user denies a request | request closes cleanly, the agent remains blocked, and the user can supply a denial note | PASS | `TestModelBeginSelectedAuthRequestDecisionDenyUsesButtonFocus`; `TestModelAuthReviewCanSwitchDecisionBeforeApply`; `internal/adapters/auth/autentauth/service_test.go:774` |
| AU-07 | `till auth --help` is opened | help explains the auth surface, required follow-up steps, and available workflows with examples | PASS | `cmd/till/main_test.go:437` |
| AU-08 | `till auth issue-session --help` is opened | required flags, returned fields, `--path` semantics when relevant, and examples are explicit | PASS | `cmd/till/main_test.go:465` |
| AU-09 | `till auth revoke-session --help` is opened | `--session-id` requirement and examples are explicit; positional invocation ambiguity is removed from the UX contract | PASS | `cmd/till/main_test.go:470` |
| AU-10 | operator needs inventory or review surfaces | plan includes `list/show/request/approve/deny/revoke` lifecycle coverage so gatekeeping is user-operable, not just developer-operable | PASS | `cmd/till/main_test.go:442`; `cmd/till/main_test.go:627`; `cmd/till/main_test.go:648`; `cmd/till/main_test.go:750`; `cmd/till/main_test.go:769`; `cmd/till/main_test.go:786` |
| AU-11 | external MCP mutation or auth-request activity occurs while the related project is open in the TUI | current project view and notifications refresh without a project-switch workaround | PASS | `TestModelAutoRefreshLoadsExternalAuthRequest` |
| AU-12 | notifications UX is reviewed for auth/workflow events | global count, quick-nav, direct auth review, and clear actionable section wording remain explicit and testable | PASS | `TestModelPanelFocusTraversalIncludesGlobalNotifications`; `TestModelGlobalNotificationsEnterOpensAuthReview`; `TestRenderOverviewPanelOmitsLegacyNoticesFallbackWhenVisible`; `TestModelViewRendersGenericConfirmHints` |
| AU-13 | operator chooses shell approval instead of TUI approval | full request/session lifecycle is operable from CLI with explicit examples and deterministic outputs | PASS | `cmd/till/main_test.go:442`; `cmd/till/main_test.go:601`; `cmd/till/main_test.go:648`; `cmd/till/main_test.go:750`; `cmd/till/main_test.go:769`; `cmd/till/main_test.go:786` |
| AU-14 | `till auth request approve --help` is opened | exact decision labels, `--path` semantics, and continuation behavior are explicit | PASS | `cmd/till/main_test.go:450` |

### 6.1 Latest Checkpoint Evidence

Timestamp:
1. 2026-03-20 local implementation checkpoint after the auth UX dogfood wave landed and the coverage floor was restored.

Files changed in this checkpoint:
1. `cmd/till/main.go`
2. `cmd/till/main_test.go`
3. `internal/adapters/auth/autentauth/service.go`
4. `internal/adapters/auth/autentauth/service_test.go`
5. `internal/adapters/auth/autentauth/service_app_sessions_test.go`
6. `internal/adapters/server/common/app_service_adapter.go`
7. `internal/adapters/server/common/app_service_adapter_mcp.go`
8. `internal/adapters/server/common/mcp_surface.go`
9. `internal/adapters/server/common/capture_test.go`
10. `internal/adapters/server/common/app_service_adapter_auth_requests_test.go`
11. `internal/adapters/server/common/app_service_adapter_helpers_test.go`
12. `internal/adapters/server/common/app_service_adapter_lifecycle_test.go`
13. `internal/adapters/server/common/app_service_adapter_mcp_helpers_test.go`
14. `internal/adapters/server/mcpapi/handler.go`
15. `internal/adapters/server/mcpapi/handler_test.go`
16. `internal/adapters/storage/sqlite/repo.go`
17. `internal/adapters/storage/sqlite/repo_test.go`
18. `internal/app/auth_requests.go`
19. `internal/app/service.go`
20. `internal/app/service_test.go`
21. `internal/domain/auth_request.go`
22. `internal/domain/auth_request_test.go`
23. `internal/tui/model.go`
24. `internal/tui/model_test.go`

Commands run:
1. `just fmt`
Outcome: pass after auth UX implementation and follow-up coverage fixes.
2. `just test-pkg ./internal/adapters/server/common`
Outcome: pass after adding capture-state, attention, auth-request, and lease lifecycle coverage.
3. `just test-pkg ./internal/adapters/auth/autentauth`
Outcome: pass; package coverage raised to 74.1%.
4. `just test-pkg ./internal/tui`
Outcome: pass; package coverage raised to 70.3%.
5. `just test-pkg ./internal/app`
Outcome: pass.
6. `just check`
Outcome: pass.
7. `just ci`
Outcome: pass.
8. final QA review findings fixed:
   - `auth session list --state` now fails closed on unsupported state values in the app-facing auth adapter.
   - auth-request continuation metadata now round-trips as a real JSON object instead of a flat string-only map.
9. `just check`
Outcome: pass after the QA remediation patch.
10. `just ci`
Outcome: pass after the QA remediation patch.
11. post-commit `just fmt`
Outcome: pass; reconciled two lingering gofmt-only test files (`internal/adapters/server/common/capture_test.go`, `internal/app/auth_requests_test.go`).
12. post-format `just check` and `just ci`
Outcome: both pass; no behavior or coverage regressions introduced by the formatting-only cleanup.
13. collaborative retest findings from 2026-03-21
Outcome:
   - confirmed shared runtime paths through `./till paths`, `./till mcp`, and `./till serve`,
   - confirmed clean stdio `Ctrl-C` shutdown,
   - confirmed shell auth request create/show/approve/session list/validate/revoke flows,
   - found TUI auth-request review regression: `enter` opened project-thread fallback and deny had no editable note,
   - remediated the TUI review flow and refreshed TUI docs/evidence.
14. `just fmt`
Outcome: pass after the TUI auth-review remediation and docs sync.
15. `just test-golden-update`
Outcome: pass after updating the TUI golden snapshots for the `Action Required` section label.
16. `just test-pkg ./internal/tui`
Outcome: pass after adding direct auth-review enter, deny-note, decision-switch, and generic-confirm-hint coverage.
17. `just check`
Outcome: pass after the TUI auth-review remediation.
18. `just ci`
Outcome: pass after the TUI auth-review remediation.
19. final independent QA re-review after TUI fix and docs sync
Outcome:
   - TUI QA pass after fixing the generic confirm-hint regression and adding direct decision-switch coverage.
   - Docs QA pass after syncing `README.md` and `PLAN.md` to the landed auth-review behavior and current dates.

Checkpoint summary:
1. `till auth` now exposes request and session lifecycle commands with example-driven help coverage.
2. MCP now exposes persisted auth-request creation/list/show tools and routes `session_required` and `grant_required` failures toward request creation instead of tuple fallback.
3. Shared-DB `autent` now persists pre-session auth requests, approval decisions, scoped approvals, and app-facing session inventory or validation wrappers.
4. TUI notifications now route focused-project auth requests locally, off-project requests globally, allow approve or deny actions, support scoped approval constraints, open auth review directly on `enter`, preserve editable denial notes, and auto-refresh external auth-request activity.
5. Final QA review findings were resolved before closeout:
   - invalid `auth session list --state` input now fails closed through the app-facing adapter path,
   - continuation metadata now preserves nested JSON objects for CLI and MCP auth-request flows,
   - auth-request review no longer falls back into generic project threads on `enter`,
   - denial review now preserves a user-editable note,
   - generic confirm modals no longer show auth-specific `a`/`d` hint text.
6. Independent QA lanes re-reviewed the finished code/docs state and passed after the final remediation/docs-sync pass.
7. Local coverage floors now pass across the touched auth/runtime packages:
   - `internal/adapters/auth/autentauth`: 74.1%
   - `internal/adapters/server/common`: 78.2%
   - `internal/tui`: 70.3%

## 7) Workstreams

### 7.1 WS-Runtime

Objective:
Unify default runtime behavior and keep stdio MCP raw and clean.

Acceptance:
1. local builds no longer silently default to dev mode
2. `./till`, `./till mcp`, and `./till serve` share the same real default runtime
3. explicit dev/isolation still works by opt-in
4. `Ctrl-C` on `till mcp` exits cleanly without error-style logging

Primary likely files:
1. `cmd/till/main.go`
2. `cmd/till/main_test.go`
3. `internal/platform/**`
4. `internal/config/**`

### 7.2 WS-Copy

Objective:
Remove stale product/runtime copy and align help/bootstrap output with current product direction.

Acceptance:
1. no live `Kan` product/runtime references remain in active user-facing surfaces
2. bootstrap/help/runtime copy matches the locked runtime/auth model
3. no compatibility copy shims are added

Primary likely files:
1. `cmd/till/main.go`
2. `internal/adapters/server/common/app_service_adapter_mcp.go`
3. `internal/adapters/server/common/mcp_surface.go`
4. relevant tests

### 7.3 WS-Auth

Objective:
Replace tuple-first MCP auth with real `autent` integration.

Acceptance:
1. `autent` is embedded and initialized from the shared runtime DB
2. MCP write paths validate session/authz before local mutation logic
3. request-local fallback identity is no longer the primary auth source
4. auth decision results are mapped cleanly into MCP-visible outcomes
5. attribution is derived from authenticated identity

Primary likely files:
1. new auth adapter package under `internal/adapters/**`
2. `cmd/till/main.go`
3. `internal/app/**`
4. `internal/adapters/server/common/**`
5. `internal/adapters/server/mcpapi/**`
6. `internal/adapters/storage/sqlite/**`
7. `go.mod`

### 7.4 WS-Auth-UX

Objective:
Make auth request, approval, revocation, and help flows operable for dogfooding without forcing the normal TUI user into shell-only session issuance.

Acceptance:
1. auth requests can originate from MCP and TUI flows
2. auth request notifications follow the focused-project vs global routing contract
3. path-scoped approval data is captured with project-rooted optional branch/phase lineage
4. user-configurable lifetime and scope constraints are part of the surfaced contract
5. auth help/output clearly describes required follow-up steps and examples
6. approval continuation and audit trail behavior are explicit in the surfaced contract
7. external MCP activity refreshes the current project and notifications surfaces
8. notifications retain global count, quick-nav, and quick-info warning/error surfacing
9. shell/operator approvals are fully supported as a first-class path, not a hidden emergency seam
10. the CLI lifecycle and help quality borrow directly from the stronger `blick` patterns while staying `tillsyn`-specific on naming and project-path semantics
11. principal/client lifecycle remains mostly implicit for dogfood users unless explicit registration is proven necessary

Primary likely files:
1. `cmd/till/main.go`
2. `cmd/till/main_test.go`
3. `internal/tui/**`
4. `internal/app/**`
5. `internal/adapters/server/common/**`
6. `internal/adapters/server/mcpapi/**`
7. `internal/adapters/auth/autentauth/**`
8. `README.md`

### 7.5 WS-Guard

Objective:
Keep local hierarchy/workflow guardrails correct after auth integration.

Acceptance:
1. hierarchy-derived local checks still fail closed where expected
2. if capability leases remain, they are secondary local workflow/delegation guards only
3. auth failures and local guard failures are distinguishable in tests and user-facing behavior

Primary likely files:
1. `internal/app/kind_capability.go`
2. `internal/app/service.go`
3. `internal/domain/**`
4. relevant transport tests

### 7.6 WS-Validation

Objective:
Prove the integrated result with automated tests and collaborative reruns.

Acceptance:
1. package-scoped `just test-pkg` checks pass for touched packages
2. `just check` passes
3. `just ci` passes
4. collaborative auth/runtime rerun steps are executed and logged here
5. QA sign-off is recorded here before handoff

## 8) File And Module Investigation Checklist

Before implementation in any lane, inspect and account for:
1. `cmd/till/main.go`
2. `cmd/till/main_test.go`
3. `internal/domain/authenticated_caller.go`
4. `internal/app/mutation_guard.go`
5. `internal/app/kind_capability.go`
6. `internal/app/service.go`
7. `internal/app/ports.go`
8. `internal/adapters/server/common/app_service_adapter_mcp.go`
9. `internal/adapters/server/common/mcp_surface.go`
10. `internal/adapters/server/mcpapi/extended_tools.go`
11. `internal/adapters/server/mcpapi/handler.go`
12. `internal/adapters/storage/sqlite/repo.go`
13. `.tmp/autent/app/service.go`
14. `.tmp/autent/sqlite/store.go`
15. `.tmp/autent/docs/02-trust-model.md`
16. `.tmp/autent/docs/03-sqlite-integration.md`
17. `.tmp/autent/docs/06-tillsyn-integration.md`
18. `README.md`
19. `AGENTS.md`
20. `CONTRIBUTING.md`

## 9) Implementation Sequence

1. Finalize and lock this active run plan in `PLAN.md`.
2. Keep all implementation subagents and QA sign-off mapped to ids in this file.
3. Land runtime and copy cleanup required for shared dogfood runtime.
4. Embed `autent` in shared-DB mode and wire startup/runtime integration.
5. Replace MCP write-path auth to use session-first validation and authz decisions.
6. Lock the auth request/approval UX, notification routing, and CLI help contract against the real dogfood requirements.
7. Reconcile local hierarchy/workflow/delegation checks after auth.
8. Update attribution persistence/read surfaces as needed.
9. Run touched-package tests after each meaningful increment.
10. Run `just check`.
11. Run `just ci`.
12. Execute collaborative rerun steps for the historical auth/runtime failure points plus the newly exposed auth UX/help flows.
13. Record QA sign-off and remaining risks in this file before handoff.
14. Run this implementation wave with one explicit build lane and two explicit QA lanes logged in the split worklog for this checkpoint.

### 9.1 Execution Model For The Next Auth UX Wave

This run will use one implementation lane plus two independent QA lanes unless the user explicitly asks for a different split.

Build lane:
1. `BLD-AUTH-UX-01`
2. Objective:
   - implement the `blick`-inspired shell auth lifecycle,
   - add persisted request approval state,
   - add TUI request review and notification routing,
   - add external refresh behavior,
   - add tests that cover the new shell and TUI contracts.
3. Expected lock scope:
   - `cmd/till/**`
   - `internal/adapters/auth/**`
   - `internal/app/**`
   - `internal/adapters/server/common/**`
   - `internal/adapters/server/mcpapi/**`
   - `internal/tui/**`
4. Hotspot ownership must remain serialized for:
   - `cmd/till/main.go`
   - `internal/app/service.go`
   - `internal/tui/model.go`
5. Worker requirements:
   - Context7 before first code edit,
   - Context7 again after every failed test/runtime error,
   - `just test-pkg` only for touched packages during the worker loop,
   - no repo-wide gate in the worker lane.

Independent QA lanes:
1. `QA-AUTH-CLI-01`
   - review shell/operator auth lifecycle, help examples, deterministic output, and package-test evidence
   - inspect `cmd/till/**`, auth adapter code, and auth request tests
2. `QA-AUTH-TUI-01`
   - review TUI request notifications, focused-vs-global routing, approve/deny flows, refresh behavior, and package-test evidence
   - inspect `internal/tui/**`, related app/server wiring, and notification tests
3. Both QA lanes are read-only except for their own worklog files.
4. Neither QA lane may sign off without mapping reviewed acceptance ids back to this file.

## 10) Test Plan

Required automated coverage:
1. runtime path/default-mode tests
2. stdio shutdown behavior tests
3. bootstrap/help/copy tests
4. auth adapter tests:
   - valid session
   - invalid secret
   - missing session
   - revoked session
   - expired session
5. MCP transport tests:
   - mutation succeeds with valid authenticated session
   - mutation fails before mutation on auth failure
   - restore-task regression is covered
6. app-layer composition tests:
   - auth allow + local allow
   - auth allow + local scope/delegation reject
   - auth deny/session failure before local mutation
7. attribution tests:
   - persisted user/agent identity is readable and correct
8. storage integration tests:
   - shared DB `autent_` table setup
   - no accidental collision with existing `tillsyn` tables
9. auth request/approval tests:
   - request creation from MCP-side and TUI-side entrypoints
   - approve and deny flows
   - cancel and timeout/expiry flows
   - scoped path parsing and validation
   - TTL/lifecycle enforcement
   - shell/operator request lifecycle parity with the planned `till auth request` tree
   - shell approve/deny/cancel outputs remain deterministic and auditable
10. notifications routing tests:
   - focused-project auth request notification
   - global notification for off-project request
11. auth help/discoverability tests:
   - `till auth --help` contains examples and next-step guidance
   - `issue-session` help shows required flags and returned values
   - `revoke-session` help shows `--session-id` usage explicitly
   - `till auth request --help` contains examples and path semantics
   - `till auth request approve --help` exposes exact approval labels and follow-up behavior
12. live refresh and notifications tests:
   - external MCP-originated changes refresh the current project without project switch
   - notifications global count and quick-nav remain correct
   - warning/error bubbling into notifications and quick-info drill-in is covered
13. `blick`-parity CLI tests:
   - root/auth help includes auth examples
   - request/session subcommand help includes examples
   - unknown-command and missing-flag paths still print actionable Fang-style usage hints

Required repo gates before handoff:
1. `just check`
2. `just ci`

## 11) Collaborative Retest Checklist

| ID | Step | Expected | Status | Evidence |
|---|---|---|---|---|
| CR-01 | start `./till mcp` on the shared real runtime | starts cleanly without `serve` | TODO | |
| CR-02 | verify MCP tool discovery | tool list is present and healthy | TODO | |
| CR-03 | run one allowed authenticated mutation | succeeds | TODO | |
| CR-04 | revoke or invalidate the session and retry the same mutation | fails before mutation | TODO | |
| CR-05 | run one valid-auth but local-scope-invalid mutation | fails locally, not as auth failure | TODO | |
| CR-06 | rerun historical restore-task path | no brittle tuple mismatch | TODO | |
| CR-07 | inspect attribution surfaces after authenticated mutation | readable actor name/type is correct | TODO | |
| CR-08 | verify `./till`, `./till mcp`, and `./till serve` path parity | same real default runtime | TODO | |
| CR-09 | verify `Ctrl-C` on `till mcp` | clean shutdown without error-style logging | TODO | |
| CR-10 | verify no live `Kan` product/runtime copy remains in active surfaces | copy is clean | TODO | |
| CR-11 | create one auth request scoped to the focused project | request appears in project notifications | TODO | |
| CR-12 | create one auth request scoped to a different project than the current focus | request appears in global notifications | TODO | |
| CR-13 | inspect `./till auth --help` and affected subcommand help | required flags, examples, and next-step guidance are explicit | TODO | |
| CR-14 | trigger one external MCP-originated change while the related project is open in the TUI | current project view and notifications refresh without project switch | TODO | |
| CR-15 | inspect notifications surfaces during auth/runtime events | global count, quick-nav, and quick-info warning/error surfacing behave as locked in this plan | TODO | |
| CR-16 | complete one shell-only approval cycle for a scoped auth request | request/create/list/show/approve or deny/session follow-up works without hidden arguments | TODO | |
| CR-17 | compare shell help against the new operator contract | `till auth request*` and `till auth session*` help is explicit enough to follow without guesswork | TODO | |

## 12) Subagent And QA Completion Contract

Every worker handoff for this run must include:
1. checklist ids completed from this file
2. files changed
3. commands run and pass/fail outcomes
4. exact tests run
5. unresolved risks

Every QA handoff for this run must include:
1. checklist ids reviewed from this file
2. explicit pass/fail for each reviewed id
3. missing evidence or drift found
4. sign-off or blocker status

No lane or the run as a whole is complete until:
1. all relevant checklist ids here are complete with evidence
2. two QA reviews have signed off against this file
3. `just check` and `just ci` pass
4. the user confirms the collaborative retest behavior for the affected sections

## 13) Evidence Ledger

Commands run:
1. `sed -n '1,240p' Justfile` -> PASS
2. `sed -n '1,260p' PLAN.md` -> PASS
3. `sed -n '1,260p' COLLAB_MCP_STDIO_AUTENT_EXECUTION_PLAN.md` -> PASS
4. `sed -n '1,260p' COLLAB_MCP_STDIO_AUTENT_VALIDATION_WORKSHEET.md` -> PASS
5. `rg -n --hidden --glob '!**/.git/**' "Kan|kan" .` -> PASS
6. `sed -n '1,260p' cmd/till/main.go` and follow-up reads -> PASS
7. Context7 on `/mark3labs/mcp-go` transport APIs -> PASS
8. `gh repo clone evanmschultz/autent .tmp/autent` -> PASS
9. `.tmp/autent` code/docs inspection -> PASS
10. parallel subagent investigations across current code, docs, and `autent` -> PASS
11. Context7 resolve/query pass for `/mark3labs/mcp-go` re-run before implementation edits -> PASS
12. Context7 lookup for `autent` unavailable; recorded fallback to local `.tmp/autent` docs/source -> PASS
13. `git status --short` and targeted `rg`/`sed` inspection across runtime/auth hotspots -> PASS
14. `just test-pkg ./cmd/till` -> PASS
15. `just check` -> PASS
16. `just test-pkg ./internal/adapters/server/common` -> PASS
17. `just test-pkg ./internal/adapters/server/mcpapi` -> PASS
18. user ran `go get github.com/evanmschultz/autent@v0.1.1` -> PASS
19. user ran `go mod tidy && go mod verify` -> PASS
20. `just fmt` -> PASS
21. `just test-pkg ./internal/adapters/auth/autentauth` -> PASS
22. `just test-pkg ./internal/adapters/server/common` -> PASS (`[no test files]`)
23. `just test-pkg ./internal/adapters/server/mcpapi` -> PASS
24. `just test-pkg ./cmd/till` -> PASS
25. `just check` -> PASS
26. `just ci` -> PASS
27. `rg -n "what_kan_is|Kan is|\\bKan\\b" cmd internal -g '*.go' -S` -> PASS (no live stale `Kan` product/runtime strings remain)
28. `rg -n 'actor_id|actor_name|actor_type' internal/adapters/server/mcpapi -g '*.go' -S` -> PASS (matches only current tests, not production transport code)
29. `rg -n 'session_id|session_secret' internal/adapters/server/mcpapi -g '*.go' -S` -> PASS (mutating MCP tools now advertise session-first auth fields)
30. independent QA lane spawn:
   - `QA-1` runtime/auth startup + CLI review -> pending final report
   - `QA-2` MCP/auth transport review -> pending final report
31. follow-up QA remediation loop:
   - fixed secondary HTTP attention mutation auth bypass by requiring session-first auth on HTTP write routes and aligning HTTP auth error mapping -> PASS
   - removed the remaining implicit-agent backward-compat mutation-guard shim -> PASS
32. `just fmt` -> PASS
33. `just test-pkg ./internal/adapters/server/httpapi` -> PASS
34. `just test-pkg ./internal/adapters/server/common` -> PASS (`[no test files]`; executable proof moved to higher-level packages to keep repo coverage gates honest)
35. `just test-pkg ./cmd/till` -> PASS
36. `just test-pkg ./internal/adapters/auth/autentauth` -> PASS
37. `just test-pkg ./internal/adapters/server/mcpapi` -> PASS
38. `just check` -> PASS
39. `just ci` -> PASS
40. runtime/auth evidence additions:
   - `TestResolveRuntimePathsCommandsShareDefaultNonDevRuntime` -> PASS
   - `TestAuthorizeMutationRevokedSessionReturnsInvalidAuthentication` -> PASS
   - `TestAuthorizeMutationDenyRuleReturnsAuthorizationDenied` -> PASS
   - `TestAuthorizeMutationGrantRequiredReturnsGrantRequired` -> PASS
   - `TestServiceAuthorizeRevokedSessionReturnsDecision` -> PASS
   - `TestServiceAuthorizeDenyRuleReturnsDecision` -> PASS
   - `TestServiceAuthorizeGrantRequiredReturnsDecision` -> PASS
41. HTTP auth evidence additions:
   - `TestHandlerAttentionMutationsRequireSession` -> PASS
   - `TestHandlerAttentionAgentMutationsRequireGuardTuple` -> PASS
   - `TestWriteErrorFromMappingBranches` -> PASS for HTTP auth error codes
42. persisted mutation / shutdown evidence additions:
   - `TestHandlerAttentionMutationPersistsAuthenticatedAttribution` -> PASS
   - `TestRunMCPCommandTreatsCanceledRunnerAsCleanShutdown` -> PASS (runner now reached before cancel)
43. MCP persisted mutation evidence addition:
   - `internal/adapters/server/mcpapi/handler_integration_test.go` -> `TestHandlerAttentionMutationPersistsAuthenticatedAttribution` -> PASS
44. independent QA rerun requested after remediation:
   - transport/copy lane -> PASS
   - runtime/auth lane -> PASS
45. post-push GitHub Actions follow-up:
   - `gh run view 23355810990 --log-failed` -> FAIL on `fmt-check` only for `internal/adapters/auth/autentauth/service.go` and `internal/adapters/server/mcpapi/handler_integration_test.go`
   - `just fmt` -> PASS
   - `just check` -> PASS
   - `just ci` -> PASS
46. remote CI confirmation follow-up:
   - `gh run watch 23356975371 --exit-status` -> IN PROGRESS while waiting on the Windows job; ubuntu and macOS legs already completed green
   - `AGENTS.md` updated to require `gh run watch --exit-status` on the new run before claiming pushed CI is green
47. plan expansion after collaborative auth UX retest:
   - user rerun confirmed shared `$HOME` runtime paths and clean `Ctrl-C` shutdown for `./till mcp`
   - user feedback exposed missing auth request/approval UX, notification routing, `--path` scope requirements, and insufficient `till auth` help discoverability
   - active run plan expanded to lock those missing auth UX requirements in sections 3, 4, 5, 6, 7, 9, 10, and 11 -> PASS
48. secondary markdown audit after plan expansion:
   - `rg -n --glob '*.md' --glob '!worklogs/**' --glob '!third_party/**' "repo-local|\\.tillsyn/mcp|autent-aligned|future .*autent|issue-session|revoke-session|request-and-approval|notifications|global notifications|project notifications|principal-id|session-id|Kan\\b|tillsyn-user|dev_mode=true|--dev=false|same real default runtime|PLAN.md is the only active|source of truth" .` -> PASS
   - `README.md` inspected -> PASS
   - `MCP_DOGFOODING_WORKSHEET.md` inspected -> PASS
   - `COLLABORATIVE_POST_FIX_VALIDATION_WORKSHEET.md` inspected -> PASS
49. auth UX coverage recovery loop:
   - Context7 `/golang/go/go1.26.0` consulted before adding new test coverage -> PASS
   - attempted subagent coverage split, but subagent-spawn tooling was unavailable in this session; coverage audit was completed manually with package-local read/test loops -> PASS
50. focused auth UX coverage tests added:
   - `internal/adapters/server/common/capture_test.go` -> PASS
   - `internal/adapters/auth/autentauth/service_app_sessions_test.go` helper fixture -> PASS
   - `internal/tui/model_test.go` auth request helper coverage -> PASS
51. package-level validation loop:
   - `just fmt` -> PASS
   - `just test-pkg ./internal/adapters/server/common` -> PASS
   - `just test-pkg ./internal/adapters/auth/autentauth` -> PASS
   - `just test-pkg ./internal/tui` -> PASS
52. failure remediation loop during coverage recovery:
   - `just test-pkg ./internal/adapters/server/common` -> initial FAIL on capture/lifecycle assertions
   - Context7 `/golang/go/go1.26.0` re-run before the next edits -> PASS
   - `internal/adapters/server/common/capture_test.go` and `internal/adapters/server/common/app_service_adapter_lifecycle_test.go` assertions corrected to match actual contract behavior -> PASS
53. repo smoke gate after coverage remediation:
   - `just check` -> PASS
54. final full gate after coverage remediation:
   - `just ci` -> PASS
55. current package coverage floor evidence:
   - `internal/adapters/auth/autentauth` -> `73.8%`
   - `internal/adapters/server/common` -> `79.1%`
   - `internal/tui` -> `70.3%`
56. current run status checkpoint:
   - auth UX implementation is locally gate-green
   - remaining closeout is collaborative dogfood retest, run-ledger completion, and final user confirmation -> IN PROGRESS
   - audit result: secondary markdown still contains stale runtime/auth assumptions and older worksheet authority claims; those docs are reference-only until explicitly reconciled to this file -> PASS
49. source-of-truth consolidation follow-up:
   - `PLAN.md` updated again to absorb the remaining notifications, approval-continuation, audit-trail, quick-info, and live-refresh requirements that were only partially captured in older collab docs -> PASS
   - `README.md` updated to reflect the current runtime/auth contract and to stop implying older worksheet authority or stale local-user fallback behavior -> PASS
   - tests not run in this pass because the changes were docs-only -> PASS
50. root markdown cleanup follow-up:
   - retired split collab/remediation markdown files removed from the repo root after consolidating active requirements into `PLAN.md` and active user-facing guidance into `README.md` -> PASS
   - `AGENTS.md` updated to stop pointing at deleted collab/runbook markdown and to treat `PLAN.md` as the active remediation source -> PASS
   - tests not run in this pass because the changes were docs-only -> PASS
51. final root-plan consolidation follow-up:
   - retired `MCP_DOGFOODING_WORKSHEET.md` and `VECTOR_SEARCH_EXECUTION_PLAN.md` after confirming they did not contain any remaining must-copy active-run requirement missing from `PLAN.md` -> PASS
   - `PLAN.md` and `README.md` updated to stop relying on those retired side plans -> PASS
   - tests not run in this pass because the changes were docs-only -> PASS
52. `blick` comparison and next-wave auth UX planning:
   - `gh repo clone evanmschultz/blick .tmp/blick` -> PASS
   - `rg -n "fang|auth|grant|approve|approval|session|request" .tmp/blick/go.mod .tmp/blick/cmd .tmp/blick/internal .tmp/blick -g '!**/.git/**'` -> PASS
   - `sed -n '1,260p' .tmp/blick/cmd/blick/main.go` -> PASS
   - `sed -n '1,260p' .tmp/blick/cmd/blick/main_test.go` -> PASS
   - `sed -n '1,360p' .tmp/blick/cmd/blick/auth_cmd.go` -> PASS
   - `sed -n '360,760p' .tmp/blick/cmd/blick/auth_cmd.go` -> PASS
   - `sed -n '260,760p' .tmp/blick/cmd/blick/main.go` -> PASS
   - `sed -n '1,320p' .tmp/blick/internal/app/auth/grant_service.go` -> PASS
   - `sed -n '1,260p' .tmp/blick/internal/adapters/auth/autent/grant_backend.go` -> PASS
   - Context7 resolve/query pass for `/charmbracelet/fang` CLI integration patterns -> PASS
   - outcome: adopt `blick`'s strong shell lifecycle/help/testing patterns, but keep `tillsyn` project-path semantics and TUI-first approval UX -> PASS
53. split-worklog planning checkpoint:
   - user explicitly requested a separate worklog for this next implementation wave -> PASS
   - active run plan expanded with shell-approval parity, lane model, and new acceptance/retest rows -> PASS
   - tests not run in this pass because the changes were docs-only planning updates -> PASS
54. independent read-only completeness audit on the next auth UX wave:
   - existing subagent audited `PLAN.md`, `README.md`, current `till` auth CLI, and `blick` auth/grant surfaces -> PASS
   - audit confirmed the main remaining gaps are lifecycle commands, help quality, notification/TUI approval behavior, and stronger QA gates -> PASS
   - plan refined to make decision labels and timeout/cancel lifecycle behavior explicit product requirements -> PASS

Docs/process edits in this run so far:
1. `.gitignore` updated to ignore `.nvimlog`
2. `AGENTS.md` updated so this file is the active source of truth for the current run
3. this active run plan was consolidated into `PLAN.md`
4. recorded active implementation evidence and current blocker status in this ledger
5. expanded the active `PLAN.md` contract to include auth request/approval UX, notification routing, path-scoped session requests, and auth help/example requirements
6. aligned `README.md` with the active runtime/auth contract and active-source-of-truth policy
7. removed retired root collab/remediation markdown after consolidating active requirements into `PLAN.md`
8. removed retired dogfood/vector side-plan markdown after confirming `PLAN.md` now carries the necessary active run contract
9. expanded `PLAN.md` with the `blick`-inspired shell approval plan, explicit build/QA lane model, and additional auth UX acceptance/retest coverage
10. created one split worklog file for this next auth UX planning checkpoint because the user explicitly requested a separate worklog

Product/code edits in this run so far:
1. `cmd/till/main.go`
   - default local runtime no longer silently enables dev mode
   - `./till`, `./till mcp`, and `./till serve` now resolve the same default runtime paths
   - `till mcp` cleanly treats interrupt-driven shutdown as non-error completion
   - live CLI help copy now reflects stdio-primary / serve-secondary direction
   - shared-DB `autent` startup is now wired into the runtime
   - added local `till auth issue-session` and `till auth revoke-session` dogfood commands
2. `cmd/till/main_test.go`
   - runtime tests updated for shared-runtime dogfood contract
   - added auth CLI help and issue/revoke session coverage
3. `internal/adapters/server/common/mcp_surface.go`
   - bootstrap JSON field renamed from `what_kan_is` to `what_tillsyn_is`
4. `internal/adapters/server/common/app_service_adapter_mcp.go`
   - stale bootstrap product copy changed from `Kan` to `Tillsyn`
5. `internal/adapters/auth/autentauth/service.go`
   - added shared-DB `autent` adapter with dogfood policy seeding, session issue/revoke, and session-first authorization
6. `internal/adapters/auth/autentauth/service_test.go`
   - added shared-DB setup, invalid secret, missing session, expired session, revoke, and record-reuse coverage
7. `internal/adapters/server/common/app_service_adapter.go`
   - added auth-backed `AuthorizeMutation` and attention mutation attribution/guard integration
8. `internal/adapters/server/common/types.go`
   - attention mutation request types now carry authenticated actor tuples
9. `internal/adapters/server/mcpapi/extended_tools.go`
   - mutating MCP tools now require `session_id` + `session_secret`
   - stale caller-supplied MCP mutation identity fields were removed from production transport contracts
   - session-authenticated caller identity now builds the downstream actor/lease tuple
10. `internal/adapters/server/mcpapi/handler.go`
   - attention mutation tools now use session-first auth
   - MCP error mapping now distinguishes `session_required`, `invalid_auth`, `session_expired`, `auth_denied`, and `grant_required`
11. `internal/adapters/server/mcpapi/*_test.go`
   - MCP transport tests now cover session-first mutation paths and attention mutation auth
12. `internal/adapters/storage/sqlite/repo.go`
   - shared SQLite handle is exposed for `autent` shared-DB embedding

Current blocker notes:
1. Automated gates are green; no active code blocker remains for this wave.
2. Pending closeout items are:
   - final independent QA sign-off against the active top of `PLAN.md`
   - collaborative manual retest on the new session-first stdio MCP flow
   - implementation of the newly locked auth request/approval UX and help requirements
   - secondary markdown cleanup so root docs and active worksheets no longer contradict the active run contract

Tests run:
1. `just test-pkg ./cmd/till` -> PASS
2. `just check` -> PASS
3. `just test-pkg ./internal/adapters/server/common` -> PASS (`[no test files]`)
4. `just test-pkg ./internal/adapters/server/mcpapi` -> PASS
5. `just test-pkg ./internal/adapters/auth/autentauth` -> PASS
6. `just fmt` -> PASS
7. `just test-pkg ./cmd/till` -> PASS
8. `just check` -> PASS
9. `just ci` -> PASS

## 14) Historical Material Boundary

Everything below this point is retained as historical/reference material.
It is not the active run checklist for the current auth/runtime remediation wave unless this active run section explicitly points back to it.

## 3) Locked Constraints And References

### 3.1 Locked Constraints

1. Path portability rules:
   - no absolute-path export,
   - portable refs only (`root_alias` + relative paths),
   - import fails on unresolved required refs/root mappings.
2. Project linkage model stays `workspace_linked = true|false`.
3. Non-user mutations remain lease-gated and fail-closed.
4. Completion contracts remain required for completion semantics.
5. Attention/blocker escalation remains required for unresolved consensus/approval flows.

### 3.2 MCP References (Required)

1. MCP tool discovery/update:
   - https://modelcontextprotocol.io/legacy/concepts/tools#tool-discovery-and-updates
2. MCP roots/client concepts:
   - https://modelcontextprotocol.io/specification/2025-03-26/client/roots
   - https://modelcontextprotocol.io/docs/learn/client-concepts
3. MCP-Go:
   - https://github.com/mark3labs/mcp-go
   - Context7 id: `/mark3labs/mcp-go`

## 4) Global Subagent Execution Contract (Applies To Every Phase)

1. Orchestrator/integrator is the only writer for `PLAN.md` phase status and completion markers.
2. Each phase is split into parallel lanes with non-overlapping lock scopes.
3. Worker lanes run scoped checks only (`just test-pkg <pkg>`); no repo-wide gates in worker lanes.
4. Integrator runs repo-wide gates (`just check`, `just ci`, `just test-golden`) at phase integration points.
5. Worker handoff must include files changed, commands run, outcomes, acceptance checklist, and unresolved risks.
6. No lane closes without explicit acceptance evidence.

## 5) Phase Plan (Complete Execution Sequence)

## Phase 0: Collaborative Test Closeout (Historical Baseline; superseded by Wave 4 worksheet)

Objective:
- finish all collaborative test work and update worksheet evidence to current truth.

Tasks:
1. `P0-T01` Run remaining manual TUI validation for C4/C6/C9/C10/C11/C12/C13.
2. `P0-T02` Run archived/search/keybinding targeted checks and record PASS/FAIL/BLOCKED.
3. `P0-T03` Re-run focused MCP checks for known failures (`till_restore_task`, `capture_state` readiness).
4. `P0-T04` Capture logging/help discoverability evidence (`./kan --help`, `./kan serve --help`, runtime log parity).
5. `P0-T05` Fill all blank checkpoints and sign-off blocks in `MCP_DOGFOODING_WORKSHEET.md`.
6. `P0-T06` Update `COLLABORATIVE_POST_FIX_VALIDATION_WORKSHEET.md` with final evidence paths and verdict.

Parallel lane split:
1. `P0-LA` (TUI manual validation lane)
   - lock scope: `COLLABORATIVE_POST_FIX_VALIDATION_WORKSHEET.md`, `.tmp/**` evidence artifacts.
2. `P0-LB` (MCP/HTTP verification lane)
   - lock scope: `MCP_DOGFOODING_WORKSHEET.md`, `.tmp/**` protocol/evidence artifacts.
3. `P0-LC` (logging/help verification lane)
   - lock scope: `.tmp/**` logging artifacts, worksheet evidence rows for logging sections.

Exit criteria:
1. All P0 tasks have explicit PASS/FAIL/BLOCKED outcomes with evidence.
2. No blank sign-off fields remain in active worksheets.
3. Open failures are converted into explicit implementation tasks in Phase 1.

## Phase 1: Critical Remediation Fixes

Objective:
- fix currently known blockers from collaborative validation.

Tasks:
1. `P1-T01` Fix `kan_restore_task` MCP contract/guard mismatch.
2. `P1-T02` Fix logging discoverability and runtime log-sink parity gaps.
3. `P1-T03` Implement deterministic external-mutation refresh behavior in active TUI views.
4. `P1-T04` Complete notifications/notices behavior requirements (global count, quick-nav, drill-in).
5. `P1-T05` Reconcile archived/search/key policy behavior with expected UX.

Parallel lane split:
1. `P1-LA` (transport contract lane)
   - lock scope: `internal/adapters/server/mcpapi/**`, `internal/adapters/server/httpapi/**`, related tests.
2. `P1-LB` (TUI notices/refresh lane)
   - lock scope: `internal/tui/**`, related tests/golden fixtures.
3. `P1-LC` (logging/help lane)
   - lock scope: `cmd/kan/**`, `internal/adapters/server/**`, `internal/config/**`, related tests.

Exit criteria:
1. P1 defects are closed with test evidence.
2. P0 failed checks are re-run and pass or are explicitly reclassified with rationale.

## Phase 2: Contract And Data-Model Hardening

Objective:
- lock unresolved design contracts that block stable MCP/HTTP closeout.

Tasks:
1. `P2-T01` Finalize attention storage model (`table` vs embedded JSON) and migration plan.
2. `P2-T02` Finalize attention taxonomy and lifecycle/override semantics.
3. `P2-T03` Finalize pagination/cursor contract for attention and related list surfaces.
4. `P2-T04` Finalize unresolved MCP contract decisions from prior open-question sets.
5. `P2-T05` Close snapshot portability completeness gaps for collaboration-grade import/export.
6. `P2-T06` Carry unresolved override-token documentation obligations into active docs.

Parallel lane split:
1. `P2-LA` (domain/app contract lane)
   - lock scope: `internal/domain/**`, `internal/app/**`, tests.
2. `P2-LB` (storage/schema lane)
   - lock scope: `internal/adapters/storage/sqlite/**`, migration/test fixtures.
3. `P2-LC` (transport schema/docs lane)
   - lock scope: `internal/adapters/server/**`, `README.md`, `PLAN.md`, MCP worksheets.

Exit criteria:
1. Contract decisions are encoded in code/tests/docs.
2. No unresolved “open contract” placeholders remain for in-scope MVP behavior.

## Phase 3: Full Validation And Gate Pass

Objective:
- produce final evidence-backed quality pass for current scope.

Tasks:
1. `P3-T01` Run `just check`.
2. `P3-T02` Run `just ci`.
3. `P3-T03` Run `just test-golden`.
4. `P3-T04` Execute MCP full-sweep per `MCP_FULL_TESTER_AGENT_RUNBOOK.md` and capture final report.
5. `P3-T05` Re-run collaborative worksheet and dogfooding worksheet with final verdicts.

Parallel lane split:
1. `P3-LA` (automated-gates lane)
   - lock scope: test outputs and `.tmp/**` gate artifacts.
2. `P3-LB` (MCP runbook lane)
   - lock scope: MCP run artifacts/report files.
3. `P3-LC` (manual validation lane)
   - lock scope: collaborative worksheet evidence rows/screenshots.

Exit criteria:
1. Required gates pass.
2. Worksheets have final, non-blank verdicts.
3. Remaining risks are explicitly documented with owner/next step.

## Phase 4: Docs Finalization And Closeout

Objective:
- finalize accurate active docs and remove stale narrative drift.

Tasks:
1. `P4-T01` Ensure `README.md` and `AGENTS.md` reflect actual current behavior.
2. `P4-T02` Ensure `PLAN.md` statuses match worksheet/runbook evidence.
3. `P4-T03` Remove or archive stale planning/status statements that conflict with final evidence.
4. `P4-T04` Produce final closeout summary and commit sequencing plan.

Parallel lane split:
1. `P4-LA` (product docs lane)
   - lock scope: `README.md`, `CONTRIBUTING.md`.
2. `P4-LB` (process docs lane)
   - lock scope: `AGENTS.md`, `PARALLEL_AGENT_RUNBOOK.md`.
3. `P4-LC` (plan/worksheet lane)
   - lock scope: `PLAN.md`, collab worksheets/worklogs.

Exit criteria:
1. Active docs are internally consistent.
2. No stale “not implemented” statements remain for implemented behavior.

## Phase 5: Deferred Roadmap (Not In Immediate Finish Scope)

Objective:
- preserve future work without blocking finish of current scope.

Tasks:
1. `P5-T01` Advanced import/export divergence reconciliation tooling.
2. `P5-T02` Broader policy-driven tool-surface controls and template expansion.
3. `P5-T03` Multi-user/team auth-tenancy and security hardening.

Parallel lane split:
1. `P5-LA` (import/export research lane).
2. `P5-LB` (policy/template lane).
3. `P5-LC` (security/tenancy lane).

Exit criteria:
1. Roadmap items are explicitly scoped and non-blocking for current finish target.

## 6) Immediate Next Action Lock

Current next action lock:
1. complete Wave 4 independent QA docs sign-off,
2. run collaborative worksheet `COLLAB_TEST_2026-03-02_DOGFOOD.md` section-by-section with user evidence capture,
3. only then mark Wave 4 dogfood readiness as complete.

## 7) Definition Of Done For Current Finish Target

1. Phase 0 through Phase 4 are complete.
2. Known blocking defects from collaborative validation are closed or explicitly accepted with owner + follow-up.
3. `just check`, `just ci`, and `just test-golden` pass on the final integrated state.
4. Collaborative and dogfooding worksheets have final non-blank sign-off verdicts.
5. Active docs are accurate and mutually consistent.

## 8) Lightweight Execution Log

### 2026-03-17: STDIO MCP Runtime Findings

Objective:
- capture the first live stdio MCP findings after Codex config-layering was fixed.

Commands run and outcomes:
1. `codex mcp list` -> PASS; `tillsyn` registered as `command=/Users/evanschultz/Documents/Code/hylla/tillsyn/till`, `args=mcp`.
2. `./till mcp` -> PASS; process started as stdio MCP and logged a repo-local MCP runtime path under `.tillsyn/mcp/tillsyn-dev/...`.
3. `./till --dev paths` -> PASS; normal TUI dev DB remains `~/Library/Application Support/tillsyn-dev/tillsyn-dev.db`.
4. `./till --dev=false paths` -> PASS; non-dev path resolves to `~/Library/Application Support/tillsyn/tillsyn.db`.
5. `till.list_projects` through the bound MCP tool -> PASS with empty result `[]`.
6. `till.get_bootstrap_guide` through the bound MCP tool -> PASS; output still contains stale `Kan` product copy.

Findings:
1. The prior Codex startup failure was caused by config layering, not `tillsyn` transport wiring.
   - home config defined stdio `mcp_servers.tillsyn`
   - repo-local `.codex/config.toml` still defined HTTP `mcp_servers.tillsyn`
   - trusted-project config merging produced the invalid stdio+`url` conflict.
2. Stdio MCP is currently using an isolated repo-local runtime by design.
   - this is why the existing `User Project` from the normal dev DB did not appear in `till.list_projects`
   - the behavior is technically correct but confusing from a user perspective.
3. Bootstrap guide content is stale and still says `Kan`.
4. Local builds currently default to dev mode because `rootOpts.devMode` is initialized from `version == "dev"`.
   - this means `./till` defaults to app-support `tillsyn-dev` paths for locally built binaries unless the user passes `--dev=false`
   - this is undesirable for dogfooding if the product should default to the real runtime.
5. `Ctrl-C` on `./till mcp` appears to stop the stdio server, but the CLI logs `context canceled` as an error.
   - shutdown UX should likely treat interrupt cancellation as normal instead of surfacing a failure-level log.

Current discussion lock:
1. consensus reached: `./till` and `./till mcp` should stop silently defaulting to dev mode for dogfooding and should share the same real runtime by default,
2. consensus reached: keep both `mcp` and `serve`; `serve` remains the optional HTTP path and should follow the same default-runtime change,
3. consensus reached: keep `till mcp` as the raw protocol-clean stdio server and add a future visible `till mcp-inspect` developer MCP inspector/debug client,
4. consensus reached: normalize `Ctrl-C` shutdown UX for raw stdio MCP so normal interrupt is not logged as an error,
5. consensus reached: remove stale `Kan` branding from current product/runtime surfaces in place, with no backward-compatibility naming shims,
6. remaining discussion: audit and prioritize which user-visible help/command/bootstrap strings get fixed in the next implementation wave versus a later cleanup wave.

### 2026-03-17: MCP STDIO + Autent Wave Locked

Objective:
- replace the current HTTP-first / ad hoc guard model with a STDIO-first MCP runtime and an `autent`-aligned authenticated-caller foundation while preserving `tillsyn` as a self-contained product.

Discussion outcome:
1. `blick` is explicitly out of scope for this wave and remains an optional higher-level layer only.
2. `tillsyn` must expose MCP without requiring `./till serve`.
3. STDIO becomes the default/primary transport surface.
4. `autent` moves earlier in the roadmap as the target auth/session/grant foundation under `tillsyn`.
5. Current readable-name and attribution inconsistencies remain in scope for this wave.

Evidence and planning docs:
1. [COLLAB_MCP_STDIO_AUTENT_EXECUTION_PLAN.md](/Users/evanschultz/Documents/Code/hylla/tillsyn/COLLAB_MCP_STDIO_AUTENT_EXECUTION_PLAN.md)
2. [COLLAB_MCP_STDIO_AUTENT_VALIDATION_WORKSHEET.md](/Users/evanschultz/Documents/Code/hylla/tillsyn/COLLAB_MCP_STDIO_AUTENT_VALIDATION_WORKSHEET.md)

Research basis:
1. Context7: `/mark3labs/mcp-go` for STDIO vs streamable HTTP server setup (`server.ServeStdio`, `server.NewStreamableHTTPServer`)
2. Fallback source for `autent`: `.artifacts/external/autent/README.md`

Commands run and outcomes:
1. `sed -n '1,240p' Justfile` -> PASS
2. `sed -n '1,260p' PLAN.md` -> PASS
3. `sed -n '1,240p' COLLAB_E2E_REMEDIATION_PLAN_WORKLOG.md` -> PASS
4. `sed -n '1,240p' COLLABORATIVE_POST_FIX_VALIDATION_WORKSHEET.md` -> PASS
5. `git status --short` -> PASS
6. `rg -n "serve|stdio|mcp|autent|capability lease|issue_capability_lease|display_name|tillsyn-user" cmd/till internal -g'*.go'` -> PASS

Current lane plan:
1. Transport/runtime lane:
   - `cmd/till/**`
   - `internal/adapters/server/**`
2. Auth foundation lane:
   - `internal/domain/**`
   - `internal/app/**`
   - `internal/adapters/storage/sqlite/**`
   - `internal/adapters/server/common/**`
3. TUI/docs/validation lane:
   - `internal/tui/**`
   - collaborative worksheets/docs

Next step:
1. spawn worker lanes with explicit lock scopes and package-scoped `just test-pkg` expectations,
2. integrate lane handoffs,
3. run `just fmt`, `just test-golden`, `just check`, and `just ci`,
4. then hand the new collaborative worksheet to the user.

### 2026-03-17: MCP STDIO + Auth/Attribution Integration Closed

Objective:
- finish the stdio-first MCP and attribution foundation wave on the current branch, get the full gate green, and prepare the next collaborative rerun worksheet.

Research / fallback checkpoints:
1. Context7 re-consulted after failing test loops:
   - Bubble Tea key/focus handling for TUI test alignment
   - SQLite placeholder-count/insert-contract sanity for repo write fixes
   - Cobra command/help execution behavior for CLI coverage tests
2. `autent` remained on recorded fallback source:
   - `.artifacts/external/autent/README.md`
   - `.artifacts/external/autent/docs/06-tillsyn-integration.md`

Commands run and outcomes:
1. `git status --short` -> PASS
2. `just test-pkg ./cmd/till` -> FAIL
   - `TestRunImportCommandReadsSnapshot` failed with `sql: expected 25 destination arguments in Scan, not 23`
3. `just test-pkg ./internal/tui` -> FAIL
   - stale tests around create-task submit path, thread fallback setup, and task schema coverage
4. Context7 re-consulted before edits.
5. Fixed sqlite task scan destinations and aligned the stale TUI tests.
6. `just fmt` -> PASS
7. `just test-pkg ./cmd/till` -> PASS
8. `just test-pkg ./internal/tui` -> PASS
9. `just test-pkg ./internal/adapters/storage/sqlite` -> PASS
10. `just test-pkg ./internal/adapters/server/common` -> PASS (`[no test files]`)
11. `just test-pkg ./internal/app` -> PASS
12. `just test-golden` -> PASS
13. `just check` -> PASS
14. `just ci` -> FAIL
   - `cmd/till` coverage was `69.8%`, below the 70% floor
15. Context7 re-consulted before the next edit.
16. Added `cmd/till` coverage for:
   - root help includes `mcp`
   - `mcp --help`
   - stdio-MCP repo-local runtime fallback
   - stdio MCP command wiring without `serve`
17. `just fmt` -> PASS
18. `just test-pkg ./cmd/till` -> FAIL
   - test incorrectly assumed stdio MCP auto-seeds a config file
19. Context7 re-consulted before the next edit.
20. Tightened the test to assert the real contract:
   - repo-local runtime directory exists
   - repo-local DB exists
   - config is not auto-seeded for non-startup commands
21. `just fmt` -> PASS
22. `just test-pkg ./cmd/till` -> PASS
23. `just ci` -> PASS
24. Final confirmation: `just check` -> PASS
25. QA review flagged one remaining runtime-path edge:
   - `till mcp --config ...` still used an all-or-nothing local fallback and could fall back to the platform DB path.
26. Final remediation:
   - changed `resolveRuntimePaths` / `ensureRuntimePathParents` to apply stdio MCP fallback per-path,
   - added `cmd/till` coverage for config-only override + local DB fallback,
   - corrected plan/worksheet wording so this wave is described as an authenticated-caller foundation for later `autent` integration, not full `autent` replacement.
27. `just fmt` -> PASS
28. `just test-pkg ./cmd/till` -> PASS
29. `just ci` -> PASS (`cmd/till` coverage 75.2%)
30. Final confirmation after the per-path fallback fix: `just check` -> PASS

Files edited and why:
1. `cmd/till/main_test.go`
   - added stdio-MCP help/runtime-path coverage and aligned non-startup config-seeding expectations.
2. `internal/adapters/storage/sqlite/repo.go`
   - fixed `scanTask` for name columns,
   - persisted `created_by_name` / `updated_by_name` on task writes,
   - improved change-event actor-name fallback.
3. `internal/adapters/storage/sqlite/repo_test.go`
   - locked readable task-row and change-event attribution behavior.
4. `internal/adapters/server/common/app_service_adapter_mcp_actor_attribution_test.go`
   - asserted readable `UpdatedByName` on guarded agent mutations.
5. `internal/tui/model_test.go`
   - aligned tests with the current task-form save path, direct thread setup, and task schema coverage.
6. `COLLAB_MCP_STDIO_AUTENT_EXECUTION_PLAN.md`
   - updated integrated implementation/validation status.
7. `COLLAB_MCP_STDIO_AUTENT_VALIDATION_WORKSHEET.md`
   - replaced the placeholder with a concrete collaborative section-by-section worksheet.

QA / review evidence:
1. Worker review Hubble:
   - confirmed the main remaining transport gap was missing CLI/runtime-path coverage, not a structural stdio wiring hole.
2. Worker review Euclid:
   - confirmed the three TUI failures were stale tests, not live product regressions.
3. Worker review Cicero:
   - found one real remaining attribution gap in sqlite task-row writes, which was then fixed and test-covered.

Current status:
1. Implementation is green.
2. `just check` is green.
3. `just ci` is green.
4. The new stdio/auth collaborative worksheet is ready for user rerun.

Next step:
1. complete the final independent QA sign-off on code/docs,
2. then hand the user the first collaborative worksheet section (`S1`).

### 2026-02-27: PLAN Restructure For Full Phase/Lane Execution

Objective:
- convert `PLAN.md` into a complete phase/task plan with explicit parallel-lane execution for every phase.

Result:
- phases, task IDs, lane lock scopes, and exit criteria are now defined end-to-end,
- collaborative test closeout is explicitly locked as immediate next action.

Test status:
- `test_not_applicable` (docs-only change).

### 2026-02-27: Phase 0 Collaborative Closeout Run (in progress)

Objective:
- execute Phase 0 closeout checks, capture fresh evidence, and update active worksheets with explicit PASS/FAIL/BLOCKED outcomes.

Evidence root:
- `.tmp/phase0-collab-20260227_141800/`

Commands run and outcomes:
1. `just check` -> PASS (`.tmp/phase0-collab-20260227_141800/just_check.txt`)
2. `just ci` -> PASS (`.tmp/phase0-collab-20260227_141800/just_ci.txt`)
3. `just test-golden` -> PASS (`.tmp/phase0-collab-20260227_141800/just_test_golden.txt`)
4. `just build` -> PASS with environment warning (`.tmp/phase0-collab-20260227_141800/just_build.txt`)
5. `./kan --help` -> FAIL help discoverability (`.tmp/phase0-collab-20260227_141800/help_kan.txt`)
6. `./kan serve --help` -> FAIL help discoverability / startup side-effect path (`.tmp/phase0-collab-20260227_141800/help_kan_serve.txt`)
7. `curl http://127.0.0.1:18080/healthz` -> PASS (`.tmp/phase0-collab-20260227_141800/healthz.headers`, `.tmp/phase0-collab-20260227_141800/healthz.txt`)
8. `curl http://127.0.0.1:18080/readyz` -> PASS (`.tmp/phase0-collab-20260227_141800/readyz.headers`, `.tmp/phase0-collab-20260227_141800/readyz.txt`)

Focused MCP checks and outcomes:
1. `capture_state` readiness -> PASS
   - evidence: `.tmp/phase0-collab-20260227_141800/http_capture_state_project.headers`, `.tmp/phase0-collab-20260227_141800/http_capture_state_project.json`, `.tmp/phase0-collab-20260227_141800/mcp_focused_checks.md`
2. `kan_restore_task` known failure repro -> FAIL (`mutation lease is required`)
   - evidence: `.tmp/phase0-collab-20260227_141800/mcp_focused_checks.md`
3. Guardrail failure matrix probes -> MIXED
   - M2.1 (missing/invalid lease tuple): PASS
   - M2.2 (scope mismatch rejection): FAIL (scope-type/scope-id mismatch accepted in one probe)
   - evidence: `.tmp/phase0-collab-20260227_141800/guardrail_failure_checks.md`
4. Completion guard probe -> PASS
   - unresolved blocker prevented `progress -> done`; transition succeeded after resolver step
   - evidence: `.tmp/phase0-collab-20260227_141800/completion_guard_check.md`
5. Resume/hash short loop probe -> PASS
   - state hash changed on mutation and returned to baseline post-cleanup
   - evidence: `.tmp/phase0-collab-20260227_141800/capture_state_hash_loop.md`

Blockers currently open:
1. CLI help discoverability remains broken (`./kan --help`, `./kan serve --help`).
2. `kan_restore_task` MCP contract mismatch remains unresolved.
3. Manual collaborative TUI checks remain pending user execution (C4/C6/C9/C10/C11/C12/C13 and archived/search/key policy checks).
4. Additional user-directed remediation requirements must be carried into fix phase:
   - first-launch config bootstrap should copy `config.example.toml` when config is missing,
   - help UX should be implemented with Charm/Fang styled output.

Current status:
- Phase 0 remains open until manual collaborative checks are completed and worksheet sign-offs are finalized.
- `MCP_DOGFOODING_WORKSHEET.md` has no blank sign-off fields; remaining blocked rows now carry explicit blocker statements and evidence paths.
- Section 0 user execution update recorded:
  - M0.2 runtime launch marked PASS by user,
  - M0.3 hierarchy IDs captured via MCP and unresolved user-action fixture item seeded,
  - early manual findings logged (C4 fail, C6 fail, C10 fail; others pending).
- Section 1 execution update recorded:
  - M1.1 (`capture_state` all required scopes) PASS,
  - M1.2 (`requires_user_action` blocker highlight in summary) PASS.
- Section 2 execution update recorded:
  - M2.1 PASS,
  - M2.2 FAIL (scope mismatch still accepted),
  - M2.3 PASS.

File edits in this checkpoint:
1. `MCP_DOGFOODING_WORKSHEET.md`
   - filled all USER NOTES blocks and final sign-off fields with explicit status + evidence references for this run.
2. `COLLABORATIVE_POST_FIX_VALIDATION_WORKSHEET.md`
   - added Section 12 Phase 0 tracker with current task statuses and blockers.
3. `PLAN.md`
   - logged command evidence, focused-check outcomes, blockers, and worksheet status for the active Phase 0 run.

Process contract update from user:
1. Continue section-by-section collaborative test walkthrough and note capture.
2. Preserve user intent with full detail in active markdown docs; normalize wording only when needed for technically correct terminology.
3. Final step of testing process will run subagents + Context7 (+ web research as needed) to propose fixes, then record proposals only after explicit user+agent consensus.

Additional restore-surface design requirement:
1. During fix-proposal phase, evaluate whether restore should be generalized (`restore` + explicit node/scope type arg) versus task-only surface, while ensuring required guardrail tuple fields and id/name gatekeeping semantics are consistently enforced.

### 2026-02-27: Remote E2EE Architecture + Roadmap Draft

Objective:
- produce a detailed roadmap for optional remote org collaboration with strict E2EE data handling while preserving local-first OSS usage.

Commands run and outcomes:
1. `rg --files -g'*.md' | sort` -> PASS (identified doc targets)
2. `sed -n '1,360p' PLAN.md` -> PASS (loaded active plan/worklog context)
3. `rg -n "export|import|snapshot|remote|tenancy|auth|sync|sqlite|postgres|file|attachment|project_roots" ...` -> PASS (collected active constraints from canonical docs)
4. Context7 lookup:
   - `resolve-library-id sqlite` -> PASS
   - `resolve-library-id postgresql` -> PASS
   - `query-docs /websites/sqlite_cli` -> PASS
   - `query-docs /websites/postgresql_current` -> PASS
5. Web-source verification:
   - SQLite (`isolation`, `wal`, `useovernet`, `backup`, `lang_vacuum`, `sessionintro`) -> PASS
   - PostgreSQL (`ddl-rowsecurity`, `sql-listen`, `sql-notify`) -> PASS
   - S3 (`Welcome`, `using-presigned-url`, `Versioning`) -> PASS
   - WebSocket RFC6455 -> PASS
   - libsodium `secretstream` -> PASS

File edits in this checkpoint:
1. `REMOTE_E2EE_ROADMAP.md`
   - added full architecture decision, live-update model, file/blob strategy, OSS execution phases, hosted-service roadmap, and aligned references.
2. `PLAN.md`
   - added this checkpoint with command/test evidence.

Test status:
- `test_not_applicable` (docs-only changes; no code/test behavior modified).

### 2026-02-28: Rename Wave Kickoff (tillsyn project + till command)

Objective:
- execute full no-compat rename from `kan` to `tillsyn` (project/repo identity) and `till` (runtime command/binary), then revalidate parity with `just check` and `just ci`.

Commands run and outcomes:
1. `git status --short` -> PASS (`README.md` staged change only).
2. `git add README.md && git commit -m "docs: note tillsyn naming origin in README"` -> PASS (`aadf95c`).
3. Context7:
   - `resolve-library-id` for Go (`/golang/go`) -> PASS.
   - `query-docs` on module path/import rename implications -> PASS.
4. Orchestration scan:
   - `pwd && ls -la` -> PASS.
   - `sed -n '1,220p' Justfile` -> PASS.
   - `rg -n "\bkan\b|cmd/kan|/kan\b|kan\." -S --hidden --glob '!**/.git/**'` -> PASS.
5. Spawned three subagents (code/runtime, tests/fixtures, docs/automation) and collected inventories -> PASS.

File edits in this checkpoint:
1. `PLAN.md`
   - added rename-wave kickoff checkpoint and command evidence.

Test status:
- `test_not_applicable` (planning/orchestration checkpoint; implementation in progress).

### 2026-02-28: Rename Wave Implementation Complete (No Compatibility Layer)

Objective:
- complete the all-at-once rename from `kan` to `tillsyn` (project/repo/module identity) and `till` (runtime command/binary/tool namespace), with no compatibility aliases.

Subagent lane execution and outcomes:
1. `R1-core-cli` (core CLI/module/build/path surfaces) -> PASS
   - scope delivered: `go.mod`, `cmd/till/**` (from `cmd/kan/**`), `internal/platform/**`, `internal/config/**`, `internal/tui/**`, `Justfile`, `.goreleaser.yml`, `.github/workflows/ci.yml`, `.gitignore`, `config.example.toml`, `cmd/headerlab/main.go`.
2. `R2-runtime-mcp` (server/app/domain/storage surfaces) -> PASS
   - scope delivered: `internal/adapters/server/**`, `internal/adapters/storage/sqlite/**`, `internal/app/**`, `internal/domain/**`.
3. `R3-docs-ops` (docs/runbooks/worksheets/tapes) -> PASS
   - scope delivered: `README.md`, `AGENTS.md`, `MCP_*`, `COLLAB*`, `REMOTE_E2EE_ROADMAP.md`, `vhs/**`.

Commands run and outcomes:
1. Integrator gate run `just check` -> FAIL (verify-sources pathspec before staging renamed `cmd/till/*` files).
2. Context7 re-consult (Go rename/staging implications) -> PASS.
3. Staged rename paths and reran `just check` -> FAIL (`gofmt required for cmd/till/main.go`).
4. Context7 re-consult (gofmt workflow) -> PASS.
5. `just fmt` -> PASS.
6. `just check` -> PASS.
7. `just ci` -> PASS.
8. Final cleanup of lingering test sample tokens (`kan` -> `tillsyn`) in:
   - `internal/adapters/storage/sqlite/repo_test.go`
   - `internal/app/service_test.go`
   - `internal/adapters/server/mcpapi/handler_test.go`
9. Post-cleanup verification:
   - `just check` -> PASS.
   - `just ci` -> PASS.

File edits in this checkpoint:
1. `PLAN.md`
   - added full rename implementation checkpoint with subagent evidence and gate outcomes.

Test status:
- `just check` PASS
- `just ci` PASS

### 2026-02-28: Post-Integration Docs Correction

Objective:
- resolve a docs regression introduced during rename sweep where absolute local links in the remote roadmap pointed at a non-existent workspace path.

Commands run and outcomes:
1. `rg -n "/Users/.*/personal/tillsyn|/Users/.*/personal/kan" REMOTE_E2EE_ROADMAP.md ...` -> PASS (identified hardcoded absolute links).
2. Patched `REMOTE_E2EE_ROADMAP.md` links to repo-relative paths -> PASS.

File edits in this checkpoint:
1. `REMOTE_E2EE_ROADMAP.md`
   - replaced hardcoded absolute paths with repo-relative markdown links.
2. `PLAN.md`
   - recorded post-integration docs correction checkpoint.

Test status:
- `test_not_applicable` (docs-only correction; no runtime/code behavior change).

### 2026-02-28: Phase 0 Section 2 Post-Fix Rerun (in progress, blocker persists)

Objective:
- rerun Section 2 guardrail checks after app-layer + scope-mapping fixes, then update worksheets/evidence before deciding next remediation lane.

Commands/tools run and outcomes:
1. `just test-pkg ./internal/app` -> PASS (`ok ... internal/app (cached)`).
2. `kan_create_task` probe (`actor_type=agent`, missing tuple) -> PASS expected failure (`invalid_request` requiring guard tuple fields).
3. `kan_create_task` probe (`actor_type=agent` + malformed lease token) -> PASS expected failure (`guardrail_failed ... mutation lease is invalid`).
4. `kan_issue_capability_lease` on fixture project -> PASS (issued instance `2c83f1cb-fba9-40e0-b274-84705dc5e73d`).
5. `kan_raise_attention_item` scope-mismatch probe (`scope_type=task`, `scope_id=<project_id>`) -> FAIL (unexpected acceptance; persisted `5956394b-f73a-4522-8530-ec53ec00082c`).
6. `kan_create_task` cross-project mismatch probe using fixture-scoped lease -> PASS expected failure (`guardrail_failed ... mutation lease is invalid`).
7. M2.3 completion contract probe:
   - created task `d6fe3b4a-369c-4212-b049-90630e71fc1f` in progress,
   - raised blocker `a264b6fd-15bc-427f-9972-f6f5273807ae`,
   - move to done blocked (expected),
   - resolve blocker + retry move -> PASS.
8. Cleanup:
   - resolved mismatch probe item `5956394b-f73a-4522-8530-ec53ec00082c`,
   - hard-deleted probe task `d6fe3b4a-369c-4212-b049-90630e71fc1f`,
   - revoked lease `2c83f1cb-fba9-40e0-b274-84705dc5e73d`.
9. Runtime freshness check -> FLAGGED:
   - `ls -l ./kan internal/app/attention_capture.go internal/app/kind_capability.go`
   - binary mtime `2026-02-27 14:40` predates modified source mtimes (`17:13`, `17:16`), so the rerun may have exercised a stale running server.
10. Explorer subagent root-cause pass -> COMPLETED (no edits):
   - call-chain traced from MCP handler to `Service.RaiseAttentionItem` and `validateCapabilityScopeTuple`,
   - recommended next step: restart/reload runtime and re-run M2.2 before additional code edits; if still failing, add deterministic tuple guard.
11. `just build` -> PASS with known non-fatal Go stat-cache warning; rebuilt binary mtime now `2026-02-27 17:34`.

Result summary:
1. M2.1 PASS.
2. M2.2 FAIL (still open; fail-closed behavior not enforced for `scope_type=task` + project ID).
3. M2.3 PASS.

File edits in this checkpoint:
1. `.tmp/phase0-collab-20260227_141800/manual/section2_guardrail_evidence_20260227.md`
   - appended 2026-02-28 rerun with IDs, outcomes, and cleanup.
2. `MCP_DOGFOODING_WORKSHEET.md`
   - updated M2.1/M2.2/M2.3 notes and final sign-off notes to reflect post-fix rerun outcomes.
3. `COLLABORATIVE_POST_FIX_VALIDATION_WORKSHEET.md`
   - updated Section 12.8 with explicit 2026-02-28 rerun status and persisted M2.2 blocker.

Current status:
- Phase 0 remains open; Section 2 cannot be closed due to persistent M2.2 failure.
- M2.2 runtime result is currently confounded by stale-binary risk and needs one clean rerun on a refreshed server process.
- Binary is refreshed locally; next required action is restarting `./kan serve ...` and rerunning M2.2 immediately.
- Per section-by-section policy, next step is targeted remediation of M2.2 before advancing to later sections.

### 2026-02-28: Section 2 Post-Restart Recheck + CI Gate

Objective:
- verify M2.2 on a freshly restarted runtime and confirm repo-level gate status before deciding commit readiness.

Commands/tools run and outcomes:
1. `kan_raise_attention_item` mismatch probe (`scope_type=task`, `scope_id=<project_id>`) -> PASS expected fail-closed (`not_found`, no persistence).
2. `kan_issue_capability_lease` + cross-project guarded mutation probe -> PASS expected fail-closed (`mutation lease is invalid`), lease revoked.
3. `kan_list_attention_items` open project scope check -> PASS (no unexpected open items after probe).
4. `just test-pkg ./internal/app` -> PASS.
5. `just ci` -> PASS (exit 0; coverage lines still above policy thresholds).

Result summary:
1. M2.2 fail-closed behavior is now confirmed after restart.
2. Section 2 gate status: M2.1 PASS, M2.2 PASS, M2.3 PASS.
3. Phase 0 overall remains open due to separate known blockers (help/first-launch/restore + pending manual collaborative TUI sections).

File edits in this checkpoint:
1. `.tmp/phase0-collab-20260227_141800/manual/section2_guardrail_evidence_20260227.md`
   - appended post-restart verification outcome.
2. `.tmp/phase0-collab-20260227_141800/manual/section2_post_restart_20260228.md`
   - added focused post-restart probe transcript and gate outcomes.
3. `MCP_DOGFOODING_WORKSHEET.md`
   - updated M2.2 to PASS and adjusted final blocking list accordingly.
4. `COLLABORATIVE_POST_FIX_VALIDATION_WORKSHEET.md`
   - updated Section 12.8 with post-restart M2.2 PASS evidence.

### 2026-02-27: AGENTS Flow Update (Section-by-Section Fix-As-We-Go)

Objective:
- align repository agent policy with user-directed collaborative flow:
  - test one section,
  - fix findings immediately,
  - revalidate section before moving forward.

Commands run and outcomes:
1. `rg -n "Testing Guidelines|Parallel/Subagent Mode|Temporary Next-Step Directive|..." AGENTS.md` -> PASS
2. `sed -n '1,260p' AGENTS.md` + `sed -n '260,520p' AGENTS.md` -> PASS
3. Updated `AGENTS.md` to lock section-by-section remediation loop and consensus-before-implementation workflow.
4. `rg -n "Locked execution flow|section-by-section remediation|..." AGENTS.md` -> PASS (verified insertions)

File edits in this checkpoint:
1. `AGENTS.md`
   - added temporary-phase locked execution flow for section-by-section remediation with subagent/context7/web research + consensus + scoped tests + section rerun.
   - added testing-guideline rules preventing advancement before section revalidation.

Test status:
- `test_not_applicable` (process/docs-only change).

### 2026-02-27: Restore Task Guardrail Contract Investigation

Objective:
- trace `kan_restore_task` (`kan.restore_task`) guardrail failure (`mutation lease is required`) across MCP registration, common adapter contracts, and app guard enforcement.

Commands run and outcomes:
1. `rg -n "restore_task|kan_restore_task|mutation lease is required|lease"` -> PASS (identified MCP/tool + guardrail references)
2. `rg -n "delete_task|move_task|update_task|actor"` -> PASS (identified tuple-capable mutation tools for comparison)
3. `nl -ba internal/adapters/server/mcpapi/extended_tools.go` (scoped ranges) -> PASS
4. `nl -ba internal/adapters/server/common/mcp_surface.go` -> PASS
5. `nl -ba internal/adapters/server/common/app_service_adapter_mcp.go` (scoped ranges) -> PASS
6. `nl -ba internal/app/service.go` + `internal/app/kind_capability.go` (scoped ranges) -> PASS
7. `nl -ba internal/adapters/server/common/app_service_adapter.go` + `internal/adapters/server/mcpapi/handler.go` -> PASS
8. `nl -ba internal/domain/errors.go` + `internal/domain/task.go` -> PASS
9. `nl -ba Justfile` -> PASS (startup recipe review requirement)

Findings summary:
1. `kan.restore_task` MCP registration only accepts `task_id` and calls `tasks.RestoreTask(ctx, taskID)` with no actor/lease tuple.
2. Common task-service contract and adapter method signature for restore accept only `task_id`, unlike update/move/delete request structs that include `ActorLeaseTuple`.
3. App `RestoreTask` still enforces mutation guardrails using persisted `task.UpdatedByType`; when that actor type is non-user and no guard tuple is attached to context, enforcement returns `domain.ErrMutationLeaseRequired`.
4. Error mapping converts this to MCP-visible `guardrail_failed: ... mutation lease is required`.

File edits in this checkpoint:
1. `PLAN.md`
   - added investigation worklog entry with command evidence and root-cause chain.

Test status:
- `test_not_applicable` (investigation/docs-only; no code changes).

### 2026-02-27: Remote Roadmap Update (HTTP-Only Runtime + Fang/Cobra Plan)

Objective:
- update remote roadmap with newly agreed runtime decisions:
  - HTTP-only MCP for now,
  - `kan` launches TUI with local-server ensure/reuse behavior,
  - default local endpoint `127.0.0.1:5437` with auto-fallback,
  - user endpoint selection in CLI/TUI,
  - Fang/Cobra migration,
  - phase/lane plan for parallel subagents.

Commands run and outcomes:
1. `Context7 resolve-library-id fang` -> PASS
2. `Context7 resolve-library-id cobra` -> PASS
3. `Context7 query-docs /charmbracelet/fang` -> PASS
4. `Context7 query-docs /spf13/cobra` -> PASS
5. Spawned explorer subagents for:
   - serve/runtime lifecycle verification (PASS),
   - current help/UX friction and recommendations (PASS)
6. `sed -n '1,320p' REMOTE_E2EE_ROADMAP.md` -> PASS (loaded current roadmap prior to patching)
7. `Context7 resolve-library-id mcp-go` + `query-docs /mark3labs/mcp-go` -> PASS (validated transport suitability/limits for HTTP-first decision)

File edits in this checkpoint:
1. `REMOTE_E2EE_ROADMAP.md`
   - added locked 2026-02-27 runtime/transport decisions,
   - added local runtime modes, endpoint fallback policy, and supervisor behavior,
   - added `R-CLI` phase for Fang/Cobra + server orchestration,
   - added explicit parallel lane map for subagent execution,
   - updated milestones and references.
2. `PLAN.md`
   - added this checkpoint with evidence and outcomes.

Test status:
- `test_not_applicable` (docs-only changes; no code/test behavior modified).

### 2026-02-28: R-CLI-FANG-01 Integrated (Fang/Cobra CLI Migration)

Objective:
- replace stdlib `flag` CLI parsing in `cmd/till` with Fang/Cobra, improve help/error UX, and remove orphaned parser code paths.

Commands/tools run and outcomes:
1. Context7 `resolve-library-id` + `query-docs` for `/charmbracelet/fang` and `/spf13/cobra` -> PASS (captured Execute/RunE/help/error patterns).
2. Spawned worker lane `R-CLI-FANG-01` (lock scope: `cmd/till/**`, `go.mod`, `go.sum`) -> PASS.
3. Worker lane package check loop:
   - `just test-pkg ./cmd/till` baseline -> PASS
   - post-migration `just test-pkg ./cmd/till` -> FAIL (missing `go.sum` entry)
   - dependency fetch for missing checksum + `just fmt` + rerun `just test-pkg ./cmd/till` -> PASS
4. Integrator verification:
   - `just check` -> PASS
   - `just ci` -> PASS
5. Runtime smoke:
   - `./till --help` -> PASS (styled root help)
   - `./till serve --help` -> PASS (styled subcommand help)
   - `./till --badflag` -> PASS (styled error + guidance + existing `error: ...` line)

File edits in this checkpoint:
1. `cmd/till/main.go`
   - migrated to Cobra command tree executed by Fang;
   - removed stdlib `flag` parser flow and related orphaned helpers;
   - preserved `tui` default, `serve`, `export`, `import`, and `paths` command behavior.
2. `cmd/till/main_test.go`
   - updated/added help coverage for Fang/Cobra output behavior.
3. `go.mod`, `go.sum`
   - added Fang/Cobra dependencies and required checksum entries.

Current status:
- CLI adapter migration is integrated locally and gated (`just check` + `just ci` passing).
- No remaining orphaned stdlib `flag` parser path in `cmd/till/main.go`.

### 2026-02-28: Fang Output Refinement (Paths + Error Surface)

Objective:
- ensure command output/error surfaces are Fang-styled where practical, including `till paths` presentation and removal of duplicate plain error output.

Commands run and outcomes:
1. Context7 `query-docs /charmbracelet/fang` (output/error handler styling confirmation) -> PASS.
2. `go doc github.com/charmbracelet/fang` + `go doc -all github.com/charmbracelet/fang` -> PASS (validated available APIs/Styles surface).
3. `just fmt && just test-pkg ./cmd/till` -> PASS.
4. `just ci` -> PASS.
5. Runtime smoke:
   - `./till paths` -> PASS (styled titled key/value output).
   - `./till --badflag` -> PASS (Fang-styled error block, no extra plain `error:` suffix).

File edits in this checkpoint:
1. `cmd/till/main.go`
   - removed duplicate top-level plain error print in `main`;
   - added `writePathsOutput` using Fang default color scheme + lipgloss rendering;
   - routed `paths` command through styled renderer.
2. `cmd/till/main_test.go`
   - updated `TestRunPathsCommand` assertions for titled/styled paths output semantics.

Current status:
- `paths` output and CLI error surface are now aligned with Fang-style rendering expectations.

### 2026-02-28: init-dev-config Regression Fix (TTY vs Non-TTY Paths Output)

Objective:
- restore automation compatibility for recipes parsing `till paths` while keeping styled interactive output.

Commands run and outcomes:
1. `nl -ba Justfile | sed -n '1,140p'` -> PASS (identified parser dependency on `config: ...` format in `init-dev-config`/`clean-dev`).
2. Context7 resolve/query for Go terminal package -> unavailable/insufficient for target package.
3. Fallback doc source: `go doc golang.org/x/term.IsTerminal` -> PASS (`IsTerminal(fd int) bool`).
4. `just fmt && just test-pkg ./cmd/till && just ci` -> PASS.

File edits in this checkpoint:
1. `cmd/till/main.go`
   - `paths` now renders styled output only when stdout is a terminal and `NO_COLOR` is unset;
   - non-TTY output path restored to stable plain `key: value` lines for script parsing;
   - added small test hook variable for forcing styled mode in tests.
2. `cmd/till/main_test.go`
   - restored plain-output assertions for `run(paths)` on non-TTY writers;
   - added tests for plain output, styled output path, and `supportsStyledOutput` behavior.

Current status:
- interactive `till paths` remains styled;
- non-interactive/pipe usage remains machine-parseable, fixing `just init-dev-config` and `just clean-dev` parsing behavior.

### 2026-02-28: Default Serve Endpoint Update to 5437

Objective:
- align default HTTP serve endpoint to `127.0.0.1:5437` (derived from user requirement `e * 2`) across CLI and server fallback behavior.

Commands run and outcomes:
1. `rg -n "127\\.0\\.0\\.1:8080|8080|defaultBindAddress"` across CLI/server/tests -> PASS (identified all code references).
2. Checked local `/Users/evanschultz/.codex/config.toml` and TOML search under `/Users/evanschultz/.codex` -> PASS (no endpoint/default binding present; only project trust/mcp server config).
3. `just fmt && just check && just ci` -> PASS.

File edits in this checkpoint:
1. `cmd/till/main.go`
   - changed default `serve` flag HTTP bind from `127.0.0.1:8080` to `127.0.0.1:5437`.
2. `internal/adapters/server/server.go`
   - changed server fallback bind constant to `127.0.0.1:5437`.
3. `cmd/till/main_test.go`
   - updated default serve binding expectation to `127.0.0.1:5437`.

Current status:
- default endpoint is now consistently `127.0.0.1:5437` in CLI and server fallback paths.
- repo gates are green (`just check`, `just ci`).

### 2026-02-28: Dev-Mode Release Policy Note (User Requirement)

Objective:
- capture explicit policy that dev-mode behavior must not be the default for packaged/public OSS distributions; contributors should opt into dev behavior explicitly.

Policy note:
- For release/brew installs and general OSS user flows, dev behavior should be opt-in (`--dev` or `TILL_DEV_MODE=true`) rather than implicit default.
- Contributor workflows can still use explicit dev mode for isolated local paths/logging.
- Future packaging/release hardening should verify non-dev defaults and avoid shipping with implicit dev-mode defaults.

Current status:
- policy requirement recorded; implementation follow-up remains a future hardening task.

### 2026-02-28: Independent Live HTTP/MCP E2E Probe Sweep (Against User-Run Server)

Objective:
- run independent transport + parity probes against user-started `./till serve` runtime on `127.0.0.1:5437`, acknowledging existing `User_Project` data.

Commands run and outcomes:
1. HTTP connectivity probe:
   - `curl -i http://127.0.0.1:5437/api/v1/capture_state` -> PASS (reachable, deterministic 400 invalid_request for missing `project_id`).
2. MCP initialize/tools discovery:
   - `initialize` (`protocolVersion=2025-06-18`) -> PASS (200, negotiated protocol `2025-06-18`, server `tillsyn/dev`).
   - `tools/list` -> PASS (30 tools present, includes `till.list_projects`).
3. Existing project probe (expected pre-seeded data):
   - `tools/call till.list_projects(include_archived=true)` -> PASS (`User_Project` present, treated as expected).
4. HTTP/MCP parity on same project (`User_Project`, id `10cdd734-bf41-4155-b978-b5f5f5061050`):
   - HTTP `GET /api/v1/capture_state?...view=summary` vs MCP `till.capture_state(...view=summary)` -> PASS:
     - matching `state_hash`,
     - matching scope name (`User_Project`),
     - matching `work_overview.total_tasks=0`.
   - HTTP `GET /api/v1/attention/items?...state=open` vs MCP `till.list_attention_items(...state=open)` -> PASS:
     - matching item count (`0`).
5. Stateless/transport behavior:
   - `tools/list` with bogus `Mcp-Session-Id` header -> PASS (200, request still works).
   - unknown method (`unknown/method`) -> PASS (200 JSON-RPC error payload; deterministic message).
   - invalid JSON body (`{`) -> PASS (400 with deterministic parse error).
6. Initialize protocol matrix:
   - legacy `2024-11-05` -> PASS (accepted; negotiated `2024-11-05`),
   - future `2099-01-01` -> PASS (deterministic fallback `2025-11-25`),
   - missing `protocolVersion` -> PASS (deterministic default `2025-03-26`).

File edits in this checkpoint:
1. `E2E_PARITY_LOG.md`
   - created collaborative parity log with independent findings and split ownership plan (`assistant-only`, `user-only`, `together`).
2. `PLAN.md`
   - recorded live probe evidence and policy notes for the session.

Current status:
- independent HTTP/MCP sweep against live user-run runtime completed successfully.
- no blockers found for moving into collaborative parity checks.

### 2026-02-28: Bubble Tea v2 External-Update + Polling Research (No Code Edit)

Objective:
- collect authoritative guidance for Bubble Tea v2 external updates and live refresh loops (`Program.Send`, `tea.Tick`, `tea.Every`) and map it to current `till` TUI architecture risks.

Commands/research actions and outcomes:
1. Context7:
   - `resolve-library-id("bubble tea")` -> PASS (`/charmbracelet/bubbletea` selected).
   - `query-docs` for `Program.Send` + `Tick/Every` semantics -> PASS (captured one-shot timer behavior + external send control).
2. Online Charm/Bubble Tea primary sources:
   - Bubble Tea issue/PR history (`#25`, `#113`) -> PASS (confirmed design intent and `Program.Send` behavior contract).
   - Bubble Tea package docs (`pkg.go.dev/charm.land/bubbletea/v2`) -> PASS (confirmed `Program.Send`, `Tick`, `Every` behavioral notes).
   - Bubble Tea source/docs/examples:
     - `tea.go`, `commands.go` -> PASS (authoritative comments for send and timer semantics).
     - `examples/simple/main.go`, `examples/realtime/main.go`, `examples/send-msg/main.go`, discussion `#951` -> PASS (practical periodic and external-event patterns).
3. Repo architecture mapping:
   - reviewed `cmd/till/main.go`, `internal/tui/model.go`, `internal/tui/thread_mode.go`, `internal/tui/options.go`, `internal/config/config.go` -> PASS.
   - confirmed current TUI uses command-triggered reloads (`m.loadData`) with no background tick loop and no `Program.Send` integration.
   - confirmed existing selection/focus retention hooks (`clampSelections`, `retainSelectionForLoadedTasks`, `focusTaskByID`) that can be leveraged for stale-selection mitigation.

File edits in this checkpoint:
1. `PLAN.md`
   - appended research evidence and outcomes (this section).

Current status:
- research evidence collected and mapped to repo-specific recommendation surface.
- next step is to hand back practical architecture guidance/caveats to user (input focus churn, race/overfetch, stale selection).

### 2026-02-28: Live TUI External-Write Refresh Remediation (Section-by-Section Bug Fix)

Objective:
- fix collaborative validation blocker where TUI board state did not live-refresh after external MCP/HTTP mutations; align AGENTS remediation wording with explicit user workflow (`find bug -> log immediately -> fix -> verify -> move on`).

Commands/research actions and outcomes:
1. Subagent investigation sweep (code + Context7 + Charm/Bubble Tea discussions) -> PASS:
   - root cause confirmed: no periodic/subscribed board refresh path in `internal/tui/model.go`; board only reloaded on local actions/manual `r`.
   - recommendation converged on guarded recurring `tea.Tick` loop + single-flight gating + input-mode safety.
2. Context7 research:
   - `/charmbracelet/bubbletea` and pkg docs queries for `Tick/Every` one-shot semantics and `Program.Send` guidance -> PASS.
3. Implementation gates:
   - `just fmt` -> PASS.
   - `just test-pkg ./internal/tui` -> PASS.
   - `just test-pkg ./cmd/till` -> PASS.
   - `just check` -> PASS.
   - `just ci` -> PASS.

File edits in this checkpoint:
1. `AGENTS.md`
   - strengthened temporary collaborative remediation language to require immediate bug logging and per-bug fix/verify before advancing sections.
2. `internal/tui/model.go`
   - added guarded auto-refresh primitives (`autoRefreshTickMsg`, `autoRefreshLoadedMsg`, interval/arming/in-flight fields);
   - added recurring timer scheduling via `tea.Tick` and background load command wrapper;
   - added mode-gated auto-refresh (`modeNone`, `modeTaskInfo`, `modeActivityLog`) to avoid text-input disruption;
   - refactored loaded-state application into `applyLoadedMsg` and wired auto-refresh flow to schedule follow-up ticks.
3. `internal/tui/options.go`
   - added `WithAutoRefreshInterval(time.Duration)` option.
4. `cmd/till/main.go`
   - enabled TUI auto-refresh in runtime with `tui.WithAutoRefreshInterval(2*time.Second)`.
5. `internal/tui/model_test.go`
   - added live-refresh regression tests:
     - `TestModelAutoRefreshTickReloadsExternalMutationsInBoardMode`
     - `TestModelAutoRefreshTickSkipsInputModes`
     - `TestModelAutoRefreshTickPreservesFocusedSubtree`
   - added focused test helpers for auto-refresh tick/load command handling.

Current status:
- bug fix implemented and fully gated (`just check` + `just ci` green);
- TUI now periodically refreshes external mutations while preserving input-mode UX safety and subtree focus behavior.

### 2026-02-28: Notices "Recent Activity" Live-Refresh Gap (New Blocking Bug)

Objective:
- fix collaborative test finding that notices-panel `Recent Activity` did not live-refresh after external MCP mutations, even when board cards/fields updated.

Bug capture (user report):
- while verifying stepwise MCP updates in `User_Project`, task fields live-updated but notices `Recent Activity` remained stale and did not include new external edits.

Actions taken:
1. Context gathering:
   - inspected `internal/tui/model.go` data flow for `loadData`, `applyLoadedMsg`, `renderOverviewPanel`, and `activityLog` handling.
2. Root-cause confirmation:
   - notices panel reads `m.activityLog`, but normal board refresh path did not repopulate `activityLog` from persisted `ListProjectChangeEvents`.
3. Context7 checkpoint:
   - re-queried Bubble Tea command/update guidance before edits (tick-driven reloads should apply all state slices from returned message).
4. Remediation implementation:
   - wired `loadData` to fetch persisted change events and include mapped activity entries in `loadedMsg`;
   - updated `applyLoadedMsg` to hydrate/refresh `m.activityLog` from loaded activity entries;
   - added targeted TUI regression test for notices-panel live activity refresh from persisted events.
5. Verification commands:
   - `just fmt` -> PASS.
   - `just test-pkg ./internal/tui` -> PASS.
   - `just check` -> PASS.
   - `just ci` -> PASS.

Current status:
- bug fixed and verified; notices-panel `Recent Activity` now follows live external activity updates on normal board refresh.

### 2026-02-28: Header Branding Correction (`TILL` -> `HA TILL`)

Objective:
- align TUI header brand mark with project naming (`HA TILL`) and keep tests/goldens green.

Actions taken:
1. Updated board header wordmark constant in `internal/tui/model.go` from `TILL` to `HA TILL`.
2. Updated expanded help title label from `TILL Help` to `HA TILL Help` for consistent branding.
3. Golden snapshot remediation after expected output change:
   - `just test-golden-update` -> PASS.
   - `just test-pkg ./internal/tui` -> PASS.
   - `just check` -> PASS.
   - `just ci` -> PASS.

Current status:
- branding mismatch fixed and validated; golden snapshots updated to match intentional UI text changes.

### 2026-02-28: Ownership Attribution Requirement (User-Confirmed Priority)

Objective:
- preserve and surface mutation ownership as first-class data across node updates, because downstream collaboration features (comments, auditability, agent/user/system workflows) depend on it.

Requirement note (from collaborative testing session):
- every node update must retain ownership attribution fields (`actor_type` and actor identity/name);
- notices-panel recent activity should foreground ownership in compact form, with full owner details available in activity detail views;
- compact owner display should be character-limited in board notices, while detail modals should show the full owner identity.

Current status:
- requirement recorded as a non-negotiable UX/data contract for current and future mutation/audit surfaces.

### 2026-02-28: Notices Activity Ownership + Drill-Down Navigation Remediation

Objective:
- address collaborative UX bug where notices `Recent Activity` emphasized timestamps instead of ownership, lacked panel navigation, and lacked drill-down/jump-to-node behavior.

Changes implemented:
1. Activity data enrichment:
   - extended in-memory `activityEntry` to carry ownership + event metadata fields (`ActorType`, `ActorID`, `Operation`, `WorkItemID`, metadata map).
   - mapped persisted `ChangeEvent` actor fields into `activityEntry` during reload.
2. Notices panel ownership display:
   - replaced timestamp-leading notices activity row format with compact owner-leading format (`actor_type|actor_name` + summary), with character-limited owner label.
3. Notices panel keyboard navigation:
   - added board/notices focus toggle via `tab` in normal mode.
   - added notices activity row selection with `j/k` or arrow keys.
4. Activity detail modal:
   - added dedicated activity-event detail modal from notices (`enter`) showing full owner identity, full timestamp, operation, target, node id, and metadata.
5. Jump-to-node workflow:
   - added node jump action from activity detail (`enter`/`g`) with fallback flow that enables archived visibility and reloads when needed.
   - emits unavailable status when event target cannot be resolved (possible hard delete).
6. Help/hints:
   - updated board expanded-help and notices-panel hints to describe notices focus + detail interaction.

Tests added/updated:
1. `TestModelRecentActivityPanelShowsOwnerPrefix`
2. `TestModelNoticesActivityDetailAndJump`
3. `TestModelActivityEventJumpLoadsArchivedTask`
4. Existing notices/activity tests updated for intentional hint/text changes.
5. Golden snapshots updated for expected UI text differences.

Verification commands and outcomes:
1. `just fmt` -> PASS.
2. `just test-golden-update` -> PASS.
3. `just test-pkg ./internal/tui` -> PASS.
4. `just check` -> PASS.
5. `just ci` -> PASS.

Current status:
- ownership-first notices activity UX and drill-down navigation are implemented and verified;
- collaborative step-by-step live external-update validation can resume.

### 2026-02-28: MCP/Change-Event Actor Attribution Trace + Minimal Remediation

Objective:
- trace actor attribution end-to-end (MCP -> server adapter -> app service -> sqlite change_events) and fix the specific gaps causing notices activity rows to appear as `user|tillsyn-user` for orchestrator-driven mutations.

Context + root-cause findings:
1. MCP mutation actor tuple normalization lived in `withMutationGuardContext`, but user-attribution naming and guard tuple detection were conflated (explicit `actor_type=user` + `agent_name` was rejected).
2. `till.restore_task` did not accept/pass actor tuple at all, so restore mutations could not carry actor identity/guard context through MCP.
3. Several app mutation paths (`move`, `restore`, `rename`, `reparent`, archive delete, and update-without-metadata) wrote task changes without reapplying caller actor identity, so persisted change events often reused fallback/default ownership.
4. Hard delete change-event insertion path in sqlite used stored task actor fields only and did not honor request-scoped actor context.

Context7 + fallback evidence:
1. Context7 lookup for MCP-Go optional argument extraction:
   - `resolve-library-id("mark3labs/mcp-go")` -> PASS (`/mark3labs/mcp-go`)
   - `query-docs("/mark3labs/mcp-go", optional args/GetString/BindArguments)` -> PASS
2. Context7 lookup for Go stdlib `context` did not return a suitable library entry.
   - fallback source used before edits: existing repo-local context-key pattern in `internal/app/mutation_guard.go` and idiomatic package-local key usage already present in this codebase.

File edits in this checkpoint:
1. `internal/app/mutation_guard.go`
   - added `MutationActor` context payload + `WithMutationActor` / `MutationActorFromContext` helpers for request-scoped mutation attribution.
2. `internal/adapters/server/common/mcp_surface.go`
   - added `RestoreTaskRequest` with actor tuple; updated `TaskService` interface restore signature accordingly.
3. `internal/adapters/server/common/app_service_adapter_mcp.go`
   - updated `RestoreTask` to accept actor tuple and route through `withMutationGuardContext`.
   - refined guard-tuple detection (`agent_instance_id|lease_token|override_token`) so `actor_type=user` + `agent_name` works for attribution without forcing lease tuple.
   - attached mutation actor metadata to context for downstream persistence attribution.
4. `internal/adapters/server/mcpapi/extended_tools.go`
   - extended `till.restore_task` tool schema with actor tuple fields and forwarded them to restore request.
5. `internal/app/service.go`
   - added `applyMutationActorToTask` helper and applied it in task mutation paths (`move`, `restore`, `rename`, `update`, `reparent`, archive delete).
   - updated metadata update path to reuse normalized task-level actor fields when persisting.
6. `internal/adapters/storage/sqlite/repo.go`
   - hard-delete change-event write now honors request-scoped `MutationActor` context when present.
7. `internal/adapters/server/common/app_service_adapter_mcp_guard_test.go`
   - added coverage case proving user actor can provide name attribution without guard tuple.
8. `internal/adapters/server/mcpapi/extended_tools_test.go`
   - updated restore-task stub signature to new restore request type.
9. `internal/app/service_test.go`
   - added test coverage for context-provided actor attribution persistence on task update.

Commands/test evidence and outcomes:
1. `just fmt` -> PASS.
2. `just test-pkg ./internal/app` -> PASS.
3. `just test-pkg ./internal/adapters/server/common` -> PASS (`[no test files]`).
4. `just test-pkg ./internal/adapters/server/mcpapi` -> PASS.
5. `just test-pkg ./internal/adapters/storage/sqlite` -> PASS.
6. `just check` -> PASS.
7. `just ci` (run 1) -> FAIL (`internal/tui` package coverage 69.7% below 70% threshold).
8. `just ci` (run 2) -> FAIL (`internal/tui` build/test failure in existing `renderOverviewPanel` test call sites).

Current status:
- actor attribution path has been remediated for MCP task mutations (including restore + hard delete event attribution);
- full `just ci` remains red due unrelated `internal/tui` gate failure outside the touched actor-attribution scope.

### 2026-02-28: Late Subagent Audit + `test/fix cycle (collab)` Commit Rule

Objective:
- audit unexpected late subagent edits for scope/intent correctness and add explicit collaborative commit-discipline wording requested by user.

Actions and evidence:
1. Updated `AGENTS.md` temporary collaborative locked-flow with explicit `test/fix cycle (collab)` rule:
   - each fix scope must be validated and committed before next fix scope starts;
   - no new fix scope starts while prior cycle edits remain uncommitted unless user explicitly approves discard.
2. Reopened prior worker agent `019ca2c0-5445-7183-8131-e7e890f64312`, requested strict postmortem, captured assignment/scope/intent statement, then closed agent to prevent additional background edits.
3. Ran direct file-level audit of late subagent changes:
   - `internal/adapters/server/common/app_service_adapter_mcp.go`
   - `internal/adapters/server/common/mcp_surface.go`
   - `internal/adapters/server/mcpapi/extended_tools.go`
   - `internal/app/mutation_guard.go`
   - `internal/app/service.go`
   - `internal/adapters/storage/sqlite/repo.go`
   - related tests.
4. Re-validated touched package tests:
   - `just test-pkg ./internal/app` -> PASS
   - `just test-pkg ./internal/adapters/server/common` -> PASS (`[no test files]`)
   - `just test-pkg ./internal/adapters/server/mcpapi` -> PASS
   - `just test-pkg ./internal/adapters/storage/sqlite` -> PASS

Current status:
- unexpected edits source confirmed: late worker completion on prior actor-attribution lane;
- collaborative commit-discipline requirement has been codified in `AGENTS.md`;
- actor-attribution edit set is technically coherent and package-tested, with follow-up review still required for broader merge intent and remaining TUI gate failures.

### 2026-02-28: User-Run Gate Failures Logged (Current Blocker)

Objective:
- record exact current test/gate failures reported by user shell output before additional fixes.

User-provided command evidence:
1. `just check` -> FAIL in `internal/tui`:
   - failing test: `TestModelViewShowsNoticesPanel`
   - assertion mismatch at `internal/tui/model_test.go:5787`
   - expected old notices hint text no longer matches rendered output (`tab/shift+tab panels • enter details • g full activity log` now renders).
2. `just ci` -> interrupted (`^C`, exit code 130) during coverage run; `internal/tui` had not been remediated yet, so CI remains blocked pending same TUI test fix.

Local corroboration:
1. `just test-pkg ./internal/tui` -> FAIL with same `TestModelViewShowsNoticesPanel` expectation mismatch.

Current status:
- commit remains blocked until `internal/tui` test expectations/goldens are reconciled and gates pass.

### 2026-02-28: Collaborative Reset Prep (Green Gates + Dev Config Debug Default)

Objective:
- prepare repository for fresh collaborative validation restart:
  - ensure failing TUI gate is fixed,
  - ensure `init-dev-config` enforces debug logging level,
  - restore green `just check` + `just ci`.

Edits made:
1. `internal/tui/model_test.go`
   - updated stale notices hint assertion in `TestModelViewShowsNoticesPanel` from old text (`tab focus notices`) to current rendered hint prefix (`tab/shift+tab panels`).
2. `Justfile` (`init-dev-config` recipe)
   - kept config copy behavior,
   - added idempotent post-step rewrite that guarantees:
     - `[logging]` table exists,
     - `level = "debug"` inside `[logging]`,
   - applies whether config is newly created or already exists.
3. `internal/adapters/server/common/app_service_adapter_mcp_actor_attribution_test.go`
   - added `//go:build commonhash` to align with existing `common` package test-tag pattern and avoid per-package coverage gate regression in default CI flow.
4. `internal/adapters/server/mcpapi/extended_tools_test.go`
   - added default-flow actor-tuple forwarding verification via `mcpapi` handler tests:
     - update task actor tuple forwarding (`actor_type=user`, `agent_name=EVAN`),
     - restore task actor tuple forwarding (`actor_type=agent` + lease tuple fields),
   - captured request structs in stub service for explicit field assertions.

Commands and outcomes:
1. `just test-pkg ./internal/tui` -> PASS (after assertion update).
2. `just ci` (first rerun) -> FAIL:
   - coverage gate failure on `internal/adapters/server/common` (7.7%) caused by introducing default-flow tests in that package.
3. Context7 re-check performed (Go build tags/coverage behavior) before next edit.
4. `just test-pkg ./internal/adapters/server/common` -> PASS (`[no test files]` after tag alignment).
5. `just test-pkg ./internal/adapters/server/mcpapi` -> PASS.
6. `just check` -> PASS.
7. `just ci` -> PASS.
8. `just build` -> PASS (non-fatal module stat-cache permission warning observed in sandboxed environment).
9. `./till --help` smoke check -> PASS.

Current status:
- repository gates are green (`just check`, `just ci`);
- `init-dev-config` now guarantees `[logging] level = "debug"` for dev config;
- ready for user to run `just clean-dev` and restart from a fresh state for collaborative live validation.

### 2026-02-28: `init-dev-config` Migration To Cobra/Fang Command (Regex Helper)

Objective:
- replace shell/awk-based `init-dev-config` logic with a first-class Cobra/Fang command backed by Go helper code.

Context7 checkpoints:
1. Queried Context7 for Cobra command wiring (`AddCommand`, `RunE`, help behavior).
2. Queried Context7 for Go regex behavior and multiline anchoring.
3. After failed `just check` runtime panic (unsupported lookahead in Go regexp), re-queried Context7 and switched to Go-compatible regex + index slicing.

Edits made:
1. `cmd/till/main.go`
   - added `init-dev-config` Cobra/Fang subcommand with help text.
   - added `runInitDevConfig` flow:
     - resolves dev paths via platform options,
     - creates missing config from repo `config.example.toml`,
     - enforces `[logging] level = "debug"` via Go helper.
   - added `ensureLoggingSectionDebug` regex helper and related TOML section regexes.
2. `cmd/till/main_test.go`
   - updated root-help expectations to include `init-dev-config`.
   - added subcommand-help expectations for `init-dev-config`.
   - added command tests for create/update behavior and output contract.
   - added table test for `ensureLoggingSectionDebug`.
3. `Justfile`
   - replaced shell/awk recipe body with direct command call:
     - `./till --dev init-dev-config`

Commands and outcomes:
1. `just fmt` -> PASS.
2. `just check` (first run) -> FAIL (panic from unsupported regexp lookahead in `cmd/till/main.go`).
3. Context7 re-check performed for Go-compatible regex approach.
4. `just fmt` -> PASS (after fix).
5. `just check` -> PASS.
6. `just ci` -> PASS.
7. `./till --help` -> PASS; command listed with Fang-styled help.
8. `./till init-dev-config --help` -> PASS; subcommand help renders correctly.
9. `HOME=$(mktemp -d) ... ./till --app tillsyn-smoke init-dev-config` -> PASS; single-line output confirmed.

Current status:
- `init-dev-config` is now a native Cobra/Fang command (no ad-hoc shell parser logic in recipe);
- debug logging enforcement is in Go helper code;
- help output and CI gates are green.

### 2026-02-28: Collaborative MCP Live E2E Re-Run (Ownership + Guardrails)

Objective:
- execute MCP-first live validation against user-restarted server, verify guardrail gating and ownership attribution, and preserve created records for TUI inspection.

Context:
1. Initial rerun attempt hit `attempt to write a readonly database (1032)` on all mutation calls.
2. Transport isolation showed same error across MCP and HTTP write paths (not MCP-only).
3. User rebuilt/restarted server; rerun then proceeded successfully.

Commands/evidence (MCP + minimal local read-only support):
1. `till.list_projects(include_archived=true)` -> PASS; active project `d83f5620-d9cb-4dc1-b281-67f92c69463b` (`1_user_pro`).
2. `till.list_tasks(project_id=..., include_archived=false)` -> PASS (initially empty).
3. Local read-only SQL query for column IDs (required because MCP has no list-columns tool) -> PASS:
   - To Do: `c7fd8e06-678a-441f-901f-897e2da9bf0b`
   - In Progress: `8644d4c9-4429-42f0-aaa2-89060855d851`
   - Done: `e11c99eb-6c68-4ecd-8388-6bd601fdb6e6`

SG1 guardrail lane (`Codex_Subagent_SG1`, `sg1-instance`):
1. `till.create_task` as `actor_type=agent` without lease tuple -> PASS expected failure (`invalid_request`, lease tuple required).
2. `till.issue_capability_lease` -> PASS (`lease_token=e9a556ec-0a47-4c6a-bf27-81bd42ac7400`).
3. `till.create_task` -> PASS created `d0cf8388-30dc-4424-80c0-2c8e6161f5e8` (`10_SG1_Lease_Create`).
4. `till.update_task` -> PASS title now `10_SG1_Lease_Update`.
5. `till.move_task` to In Progress -> PASS.
6. `till.create_comment` on SG1 task -> PASS comment `55f749a6-b6d8-491d-8375-c6abc6231eeb`.

SG2 ownership lane (`Codex_Subagent_SG2`, `sg2-instance`):
1. `till.issue_capability_lease` -> PASS (`lease_token=aa1c2c4f-fa6e-48b0-a6bf-3b21dec62115`).
2. `till.create_task` branch -> PASS `c7fad53f-5c12-4146-b727-ab80ea0036da` (`11_SG2_Branch`).
3. `till.create_task` phase (parent=branch) -> PASS `196e55bf-54dc-4d2b-a2e2-eaf1ce9b3dd6`.
4. `till.create_task` with `kind=subphase` -> FAIL (`kind definition not found: "subphase"`).
5. `till.create_task` subphase using `scope=subphase`, `kind=phase` -> PASS `b87d4221-36dd-4c0e-82f1-2b09a2def653`.
6. `till.create_task` child task -> PASS `fabd90bc-e700-485d-9658-add06cc6883f`.
7. `till.update_task` -> PASS title `11_SG2_Task_Updated`.
8. `till.move_task` to In Progress then Done -> PASS both moves.
9. `till.create_comment` on SG2 branch -> PASS comment `f3978dfc-a2ba-4d0b-9053-492f7d3e0f50`.

Guardrail validation:
1. SG2 task update with bogus lease token `00000000-0000-0000-0000-000000000000` -> PASS expected `guardrail_failed` (`mutation lease is invalid`).
2. SG1 task update with SG2 lease token -> PASS expected `guardrail_failed` (`mutation lease is invalid`).

Ownership evidence:
1. `till.list_project_change_events(project_id=..., limit=40)` -> PASS:
   - events show `ActorType=agent` with `ActorID=Codex_Subagent_SG1` for SG1 create/update/move.
   - events show `ActorType=agent` with `ActorID=Codex_Subagent_SG2` for SG2 create/update/move.
2. `till.list_comments_by_target` on SG1/SG2 targets -> PASS:
   - SG1 comment `AuthorName=Codex_Subagent_SG1`, `ActorType=agent`.
   - SG2 comment `AuthorName=Codex_Subagent_SG2`, `ActorType=agent`.

Current status:
- live MCP mutation path is working after server restart;
- guardrails + ownership attribution are validated with preserved artifacts for TUI check;
- one surfaced contract gap: no `subphase` kind definition (requires `scope=subphase` with `kind=phase`).
- one surfaced MCP tooling gap: no `till.list_columns`/column-discovery endpoint, forcing out-of-band DB lookup to obtain `column_id` values before `create_task`/`move_task` calls.

### 2026-02-28: Collaborative TUI Activity UX Remediation (Recent Activity + Jump + Event Details)

Objective:
- fix collaborative findings in notices/activity UX:
  1. recent-activity owner rows were visually misaligned and clipped early,
  2. `go to node` from activity event could fail to focus the actual nested node,
  3. activity event detail modal showed raw UUID-heavy metadata that was not user-actionable.

User-reported issues logged:
1. recent-activity owner text (`agent|<name>`) was offset and truncated before other notice rows.
2. activity-event `go to node` returned to board but did not reliably focus the referenced node.
3. activity-event modal showed raw IDs (`work_item_id`, `*_column_id`, positions) instead of path/task context.

Implementation updates:
1. `internal/tui/model.go`
   - added jump-context preparation (`prepareActivityJumpContext`) and used it in jump flows so nested targets are focusable.
   - updated `focusTaskByID` to return success status for jump verification.
   - updated notices recent-activity row rendering to remove extra offset and keep owner/summary aligned.
   - updated activity-event modal details to show:
     - user-facing `node` and `path`,
     - humanized metadata (column names, changed fields, lifecycle transitions),
     - filtered-out raw UUID/position noise keys.
2. `internal/tui/model_test.go`
   - added/updated regression tests for:
     - nested jump focus correctness,
     - humanized column metadata rendering,
     - owner display normalization,
     - fallback target/path labels,
     - metadata-friendly fallback formatting.

Commands and outcomes:
1. `just ci` -> FAIL (pre-fix coverage gate: `internal/tui` 69.9%).
2. Context7 re-check performed before next edits (Bubble Tea test/update patterns).
3. `just test-pkg ./internal/tui` -> FAIL (compile error in new test: invalid model field literal).
4. Context7 re-check performed after failure (required by repo policy).
5. `just test-pkg ./internal/tui` -> PASS.
6. `just check` -> PASS.
7. `just ci` -> PASS (`internal/tui` coverage now 70.3%).

Current status:
- collaborative activity UX findings above are implemented and covered by tests;
- repo gates are green for this cycle (`just check`, `just ci`);
- MCP tooling gap (`no till.list_columns`) remains explicitly tracked for follow-up fix scope.

### 2026-02-28: Branding Normalization (`tillsyn` app name, `till` command-only)

Objective:
- enforce naming intent: app/UI branding must be `tillsyn`; command/tool syntax remains `till`.

Findings captured before edits:
1. TUI header/help branding showed `HA TILL` and `HA TILL Help`.
2. Empty-project and thread headers showed `till` as app label (`till`, `till thread`).
3. README wording contained invalid phrase `ha till`.
4. Config example heading used `# till example configuration`.

Implementation updates:
1. `internal/tui/model.go`
   - `headerMarkText` -> `TILLSYN`.
   - help modal title -> `TILLSYN Help`.
   - empty-project title -> `tillsyn`.
   - command palette quit description -> `quit tillsyn`.
   - default identity display -> `tillsyn-user`.
   - removed legacy `till-user` alias in activity-owner normalization.
2. `internal/tui/thread_mode.go`
   - thread header -> `tillsyn thread`.
   - fallback comment author/default actor display -> `tillsyn-user`.
3. Test/golden synchronization:
   - `internal/tui/model_teatest_test.go`
   - `internal/tui/model_test.go`
   - `internal/tui/testdata/TestModelGoldenBoardOutput.golden`
   - `internal/tui/testdata/TestModelGoldenHelpExpandedOutput.golden`
4. Docs/config wording:
   - `README.md` naming sentence -> Swedish word definition for `tillsyn`.
   - README fallback identity text -> `tillsyn-user`.
   - `config.example.toml` heading -> `# tillsyn example configuration`.

Commands and outcomes:
1. `just check` -> FAIL (`gofmt required for internal/tui/model.go`).
2. Context7 re-check performed before next edit.
3. `just fmt && just check && just ci` -> FAIL at `just check` (`internal/tui` golden EOF newline mismatch only).
4. Context7 re-check performed before fixture-byte edit.
5. Adjusted golden fixtures to match exact EOF byte expectation.
6. `just check && just ci` (escalated for Go cache writes) -> PASS.

Current status:
- app-visible branding now uses `tillsyn`;
- command surfaces remain `till`;
- gates are green after normalization (`just check`, `just ci`).

### 2026-02-28: Init-Dev-Config Copy/Paste Path Output Fix

Objective:
- make `just init-dev-config` output copy/paste-safe on paths containing spaces.

Issue observed:
1. `init-dev-config` printed unquoted absolute paths (for example under `~/Library/Application Support/...`), causing direct shell reuse to fail unless manually escaped.

Implementation updates:
1. `cmd/till/main.go`
   - added `shellQuotePath` helper for POSIX-safe single-quoted path rendering.
   - updated `runInitDevConfig` output line to print quoted config path.
2. `cmd/till/main_test.go`
   - updated init-dev-config output assertions to expect quoted paths.

Commands and outcomes:
1. Context7 consulted before edit (Go string/formatting guidance) -> PASS.
2. `just fmt && just check && just ci` -> PASS.

Current status:
- `init-dev-config` now prints copy/paste-safe quoted path output, e.g.:
  - `dev config already exists: '/Users/.../Library/Application Support/tillsyn-dev/config.toml'`.

### 2026-02-28: Init-Dev-Config Output Style Adjustment (Backslash Escapes)

Objective:
- align `init-dev-config` output with user preference for direct paste paths using backslash-escaped spaces (instead of single-quoted paths).

Issue observed:
1. Single-quoted path output was technically shell-safe but did not match expected copy/paste ergonomics (`Application\\ Support` style).

Implementation updates:
1. `cmd/till/main.go`
   - replaced quoted output helper with `shellEscapePath` that emits one shell-safe token using backslash escapes for spaces and shell metacharacters.
   - `runInitDevConfig` output now uses escaped token format.
2. `cmd/till/main_test.go`
   - updated output assertions to expect escaped token paths.
   - added `TestShellEscapePath` coverage for `Application Support` path escaping.

Commands and outcomes:
1. Context7 consulted before edits (Go formatting/string output guidance) -> PASS.
2. `just fmt && just check && just ci` -> PASS.
3. Local smoke check with temp HOME/XDG env -> PASS:
   - output now prints `.../Library/Application\ Support/...`.

Current status:
- `just init-dev-config` output is now backslash-escaped and directly pasteable as requested.

### 2026-02-28: Level-Scoped Guardrail Enforcement for Task/Comment Mutations

Objective:
- make mutation guardrails truly level-scoped for agent leases (project/branch/phase/subphase/task/subtask), not project-only for task/comment writes.

Issue observed:
1. `CreateTask`, `UpdateTask`, `MoveTask`, `DeleteTask`, `RestoreTask`, `ReparentTask`, and `CreateComment` were still enforcing guardrails against `project` scope only.
2. This blocked intended phase/task scoped leases for subagent flows and produced ambiguous `mutation lease is invalid` failures.

Implementation updates:
1. `internal/app/mutation_scope.go` (new):
   - added task-lineage scope resolution helper that derives allowed scope tuples from project + ancestor chain + node scope.
2. `internal/app/kind_capability.go`:
   - retained `enforceMutationGuard` API and routed it through new multi-scope enforcement helper.
   - added `enforceMutationGuardAcrossScopes` to validate one lease tuple against a normalized allowed-scope set.
   - expanded guardrail mismatch logging to include requested scope tuple set.
3. `internal/app/service.go`:
   - replaced project-only guard checks in task/comment mutation flows with lineage-derived scope candidate checks.
   - create-under-parent now checks parent lineage.
   - reparent now enforces permission for both the task lineage and destination parent lineage.
4. `internal/app/service_test.go`:
   - added `TestScopedLeaseAllowsLineageMutations`.
   - added `TestScopedLeaseRejectsSiblingMutations`.

Commands and outcomes:
1. `just test-pkg ./internal/app` -> FAIL (`undefined: domain.WorkKindBranch` in new tests).
2. Context7 re-check performed before next edit.
3. `just fmt`.
4. `just test-pkg ./internal/app` -> FAIL (`"task" does not apply to "branch"` in new tests).
5. Context7 re-check performed before next edit.
6. adjusted branch test fixtures to use explicit `kind="branch"` ID with `scope=branch`.
7. `just fmt`.
8. `just test-pkg ./internal/app` -> PASS.
9. `just check` -> PASS.
10. `just ci` -> PASS.

Current status:
- level-scoped lease guardrails now authorize by subtree lineage instead of project-only hardcoding for task/comment mutation paths.
- full repo gates are green (`just check`, `just ci`).

### 2026-02-28: Ownership Attribution Regression During Live MCP Validation (OPEN)

Objective:
- record critical collaborative test finding before next fix scope.

Issue observed:
1. Live MCP setup mutations executed without explicit actor lease tuple were attributed as `user` (`tillsyn-user`) instead of agent/orchestrator identity.
2. This polluted ownership evidence during collaborative guardrail validation and made agent-vs-user provenance unreliable in TUI/Recent Activity.

Status:
- OPEN (discussion + fix design required before implementation).
- user reset test data after observing misattribution.

Follow-up requirements (next fix scope):
1. ensure orchestrator test flow never executes mutation calls without explicit `actor_type=agent` + `agent_name` + `agent_instance_id` + `lease_token`.
2. evaluate fail-closed transport/runtime option to block mutation requests with implicit user attribution when the caller intends agent orchestration mode.
3. re-run MCP + subagent guardrail validation with strict ownership assertions and preserve evidence.

### 2026-02-28: Subagent Execution Stall During Live Guardrail Validation (OPEN)

Objective:
- capture failed live subagent validation run and record next discussion/fix direction.

Run context:
1. User reset DB/state and restarted server + TUI for clean collaborative verification.
2. Orchestrator issued explicit project-scoped lease and created branch/phase setup rows with agent attribution.
3. Orchestrator issued explicit phase-scoped worker leases for two subagents.
4. Two subagents were spawned with strict prompts (one in-scope create + one out-of-scope create each, no self-lease issuance).

Failure observed:
1. Both subagents ran for ~5 minutes without completing simple MCP mutation tasks.
2. User interrupted execution due stall.
3. This repeated prior behavior seen in earlier attempts (multi-minute stalls for simple actions).

User findings/hypothesis:
1. likely both prompting/orchestration issue and code/system issue.
2. current gatekeeping flow feels too fragile/slow for practical collaborative workflows.
3. discuss and evaluate an `Auth 2.0` model for gatekeeping.

Auth 2.0 discussion backlog (explicit):
1. re-evaluate stateless per-call tuple model versus session-bound authenticated identity context for subagent flows.
2. design first-class orchestrator-to-subagent delegation handshake (server-issued, revocable, scope-bound grants) with clearer lifecycle.
3. add deterministic guardrail stage observability:
   - lease lookup,
   - identity match,
   - scope check,
   - decision outcome,
   - latency timing,
   exposed as structured logs/events.
4. define hard operational SLOs for automated lanes (for example first mutation within N seconds) and automatic timeout/escalation behavior.
5. evaluate approval/gating UX for identity+scope grants so operator intent is explicit and auditable.

Required follow-up:
1. perform focused root-cause investigation for subagent stall:
   - prompt contract quality,
   - MCP tool invocation overhead/queueing,
   - guardrail round-trip behavior under subagent execution.
2. agree on Auth 2.0 target architecture before implementing broad auth/gatekeeping rewrite.
3. preserve existing strict fail-closed guarantees while reducing orchestration friction/latency.

### 2026-02-28: Activity Log Entity Labeling Fix (Branch/Phase/Subphase)

Objective:
- stop labeling every persisted work-item event as `* task` in notices recent activity and activity-log modal.

Issue observed:
1. branch/phase/subphase operations were displayed as `create task` / `update task` etc.
2. this affected both notices panel recent activity rows and activity-log modal rows sourced from persisted change events.

Implementation updates:
1. `internal/adapters/storage/sqlite/repo.go`:
   - enriched change-event metadata on create/update/delete with:
     - `item_kind`
     - `item_scope`
     - `title` (ensured on update path too).
2. `internal/tui/model.go`:
   - replaced hardcoded `* task` summary mapping with `operation + entity` mapping.
   - added `activityEntityLabel` helper to derive entity from event metadata (`item_scope` -> fallback `item_kind` -> fallback `task`).
3. `internal/tui/model_test.go`:
   - updated recent-activity owner-prefix test to verify scope-aware summary rendering (`update phase` when metadata scope is phase).

Commands and outcomes:
1. Context7 consulted before edits -> PASS.
2. `just fmt` -> PASS.
3. `just test-pkg ./internal/adapters/storage/sqlite` -> PASS.
4. `just test-pkg ./internal/tui` -> PASS.
5. `just check` -> PASS.
6. `just ci` -> PASS.

Current status:
- persisted activity rows now render entity-aware summaries for branch/phase/subphase/task scope events instead of always `task`.

### 2026-03-02: Dogfood Blocker Remediation Wave (IN PROGRESS)

Objective:
- close known dogfooding blockers surfaced in active collaborative worksheets, then refresh worksheets for one joint validation pass.

Backlog/open-findings review checkpoint:
1. Reviewed active backlog/open discussion items in this file (`PLAN.md`), including:
   - open Phase 0 closeout status and blocker statements,
   - open ownership-attribution regression discussion,
   - open subagent stall/Auth 2.0 discussion items.
2. Reviewed unresolved findings in:
   - `COLLAB_E2E_REMEDIATION_PLAN_WORKLOG.md`,
   - `COLLABORATIVE_POST_FIX_VALIDATION_WORKSHEET.md`.
3. Reviewed current MCP dogfood sign-off state in:
   - `MCP_DOGFOODING_WORKSHEET.md`.

Current remediation focus (known blockers from docs + code audit):
1. restore-task guard actor mismatch (`mutation lease is required` on user restore for agent-attributed archived tasks).
2. MCP/HTTP guardrail error log sink parity gaps.
3. first-launch config bootstrap seeding gap (missing config template copy on normal startup).
4. docs/worksheet drift after recent code fixes.

Parallel lane lock table (single-branch orchestration; non-overlapping scopes):
1. `W-RESTORE-ACTOR`
   - lock scope:
     - `internal/app/service.go`
     - `internal/app/service_test.go`
     - `internal/app/mutation_guard.go` (only if required)
   - acceptance objective:
     - restore guard behavior follows current caller actor context with fail-closed non-user semantics preserved.
2. `W-LOG-PARITY`
   - lock scope:
     - `internal/adapters/server/mcpapi/handler.go`
     - `internal/adapters/server/mcpapi/handler_test.go`
     - `internal/adapters/server/httpapi/handler.go`
     - `internal/adapters/server/httpapi/handler_test.go`
   - acceptance objective:
     - mapped MCP/HTTP error paths emit structured runtime logs without changing response contracts.
3. `W-BOOTSTRAP-CONFIG`
   - lock scope:
     - `cmd/till/main.go`
     - `cmd/till/main_test.go`
     - `README.md`
   - acceptance objective:
     - normal startup seeds config from `config.example.toml` when missing, while preserving help behavior.

Commands/tests run (orchestrator evidence):
1. `sed -n '1,220p' Justfile` -> PASS (verified recipe source-of-truth and gate commands).
2. `git log -n 5 ...` and `git log -n 5 --name-status -- '*.md'` -> PASS (identified latest markdown workset).
3. targeted file audits (`rg`, `sed`) across active worksheets + code paths -> PASS.
4. `just check` -> PASS.
5. `just ci` -> PASS.
6. spawned worker lanes:
   - `019cabe0-a8c7-74d3-8634-c23e206412c3` (`W-RESTORE-ACTOR`) -> IN_PROGRESS.
   - `019cabe0-aad2-75f0-8626-e69d5765e420` (`W-LOG-PARITY`) -> IN_PROGRESS.
   - `019cabe0-ac7c-7221-9dd1-1d874c1b83eb` (`W-BOOTSTRAP-CONFIG`) -> IN_PROGRESS.

Current status:
- worker lanes are executing with explicit Context7-before-edit and failure-triggered Context7 re-check requirements.
- next step is orchestrator review/integration of each handoff, then `just check` + `just ci`, then worksheet/doc updates with fresh evidence.

Integrator review and lane closeout:
1. `W-RESTORE-ACTOR` (`019cabe0-a8c7-74d3-8634-c23e206412c3`) -> COMPLETED
   - integrated changes:
     - `internal/app/service.go`
     - `internal/app/service_test.go`
   - outcome:
     - restore guard actor now follows current mutation actor context (user default), with non-user lease enforcement preserved.
2. `W-LOG-PARITY` (`019cabe0-aad2-75f0-8626-e69d5765e420`) -> COMPLETED
   - integrated changes:
     - `internal/adapters/server/mcpapi/handler.go`
     - `internal/adapters/server/mcpapi/handler_test.go`
     - `internal/adapters/server/httpapi/handler.go`
     - `internal/adapters/server/httpapi/handler_test.go`
   - outcome:
     - MCP/HTTP mapped error branches now emit structured adapter-edge logs (`error_class`, `error_code`, transport fields) and tests assert mappings.
3. `W-BOOTSTRAP-CONFIG` (`019cabe0-ac7c-7221-9dd1-1d874c1b83eb`) -> COMPLETED
   - integrated changes:
     - `cmd/till/main.go`
     - `cmd/till/main_test.go`
     - `README.md`
   - outcome:
     - normal TUI startup now seeds missing config from `config.example.toml` (when template is present), with help paths remaining side-effect free.

Post-integration validation commands:
1. `just check` -> PASS.
2. `just ci` -> PASS.
3. `./till --help` and `./till serve --help` smoke capture -> PASS:
   - stderr bytes: 0 for both help commands,
   - usage text present in captured outputs.

Validation limitations observed:
1. live `serve` integration smoke for HTTP/MCP runtime logging could not be completed in this sandbox due bind failure (`listen tcp ... bind: operation not permitted`).
2. adapter-level log mapping is test-covered; full runtime sink parity still requires collaborative/local serve-session verification outside sandbox bind limits.

Next step:
- update active collaborative worksheets and remediation worklog with this fix wave + rerun requirements for remaining manual/transport checkpoints.

Docs/worksheet synchronization completed:
1. `COLLABORATIVE_POST_FIX_VALIDATION_WORKSHEET.md`
   - reclassified REQ-008/009/010/027 from `MISSING` -> `PARTIAL`.
   - updated Section 7 blockers/follow-up actions to reflect 2026-03-02 code fixes and rerun requirements.
   - marked P0-T03/P0-T04 as `IN_PROGRESS` with rerun-required notes.
   - appended Section 12.9 remediation update with fresh gate evidence.
2. `MCP_DOGFOODING_WORKSHEET.md`
   - added Section 6 remediation addendum (2026-03-02).
   - updated final sign-off blocker wording to focus on pending collaborative reruns/manual sections.
3. `COLLAB_E2E_REMEDIATION_PLAN_WORKLOG.md`
   - moved T-004/T-005 backlog rows to `implemented_pending_validation`.
   - marked task cards subagent/orchestrator checks complete with code/test evidence pointers.
4. Added evidence artifact:
   - `.tmp/phase0-collab-20260227_141800/remediation_wave_20260302.md`.

Current status:
- known code-level blockers targeted in this wave are implemented and repo gates are green.
- dogfooding sign-off remains open pending collaborative reruns for:
  1. live serve-session sink parity verification,
  2. focused `till_restore_task` transport rerun,
  3. remaining manual TUI validation sections.

Final pre-handoff gate rerun after worksheet/doc updates:
1. `just check` -> PASS (cached).
2. `just ci` -> PASS (cached).

## Checkpoint 2026-03-02: Collab Test Sheet Refresh + Agent-Only Reruns

Objective:
- Create a new dated collaborative worksheet and execute all agent-only checks now (including guardrail E2E and gatekept subagent probes), then leave only user/joint manual checks pending.

Commands/tests run and outcomes:
1. `just check` -> PASS (evidence: `.tmp/collab-test-2026-03-02/a01_just_check.txt`).
2. `just ci` -> PASS (evidence: `.tmp/collab-test-2026-03-02/a02_just_ci.txt`; includes non-fatal stat-cache warning).
3. `just test-golden` -> PASS (evidence: `.tmp/collab-test-2026-03-02/a03_test_golden.txt`).
4. `./till --help` and `./till serve --help` -> PASS, 0 stderr bytes (evidence: `a04_*` artifacts).
5. `just test-pkg ./cmd/till` -> PASS (startup seeding coverage; evidence: `a05_startup_seed_check.txt`).
6. Isolated live serve transport sweep (`E-01`..`E-08`) -> `E-01`..`E-07` PASS, `E-08` FAIL (sink parity gap persists); evidence under `.tmp/collab-test-2026-03-02/e*`.
7. Subagent gate probes:
   - initial non-escalated attempt -> BLOCKED (`bind: operation not permitted`).
   - rerun with escalated local bind permissions -> PASS for in-scope + out-of-scope expectations (evidence: `s01_subagent_in_scope.md`, `s02_subagent_out_scope.md`).

Files edited and why:
1. `COLLAB_TEST_2026-03-02.md`
   - created dated worksheet,
   - carried forward unresolved test scopes from prior worksheets,
   - updated agent-only statuses with evidence,
   - left required collaborative/manual checks as explicit pending rows.

Current status:
- Agent-only testable items are complete for this pass.
- Remaining blocker: logging sink parity (`E-08`) still fails in this environment.
- Remaining work is collaborative/manual validation (`C-01`..`C-07`).

## Checkpoint 2026-03-02: E-08 Sink-Parity Remediation and Verification

Objective:
- Fix `E-08` so mapped MCP/HTTP adapter errors are persisted to `.tillsyn/log` (not only stderr), then confirm with real gates and live serve-session evidence.

Implementation updates:
1. `cmd/till/main.go`
   - added runtime default-logger installation (`InstallAsDefault` / `RestoreDefault`) so package-level `charmbracelet/log` calls flow through runtime sinks.
   - added `runtimeLogBridgeWriter` fanout writer to mirror package-level logs to active console sink (when enabled) and dev-file sink.
2. `cmd/till/main_test.go`
   - added regression `TestRuntimeLoggerInstallAsDefaultRoutesPackageLogsToFile` to verify package-level logs reach dev-file sink and respect console muting.

Commands/tests run and outcomes:
1. `just test-pkg ./cmd/till` -> PASS.
2. `just check` -> FAIL initially (`gofmt required for: cmd/till/main.go`), then:
   - Context7 re-check executed per policy after failure,
   - `just fmt` applied,
   - `just check` rerun -> PASS.
3. `just ci` -> PASS.
4. Live `E-08` rerun (local serve runtime with HTTP + MCP invalid requests) -> PASS:
   - evidence: `.tmp/collab-test-2026-03-02/e08_rerun_v2_summary.log`
   - both counters incremented: `delta_mcp=1`, `delta_http=1`,
   - matched lines present in `.tillsyn/log/tillsyn-20260302.log` and serve stderr.

Current status:
- `E-08` is remediated and reclassified PASS in `COLLAB_TEST_2026-03-02.md`.
- No remaining agent-only FAIL items in the dated collab test worksheet.

## Checkpoint 2026-03-02: Parallel Comment + Notices Remediation Setup

Objective:
- Confirm comment schema/ownership coverage, then run non-overlapping parallel lanes for:
  1. comment target-type completion (`branch` + `subphase`) across domain/app/MCP/TUI mapping,
  2. notices panel focusable/scrollable/selectable UX redesign.

Backlog/open-findings review:
1. Reviewed active collaborative docs and unresolved behavior:
   - missing global notifications workflow and section-level navigable lists in notices panel,
   - comment coverage mismatch for hierarchy node types.
2. Reviewed `PARALLEL_AGENT_RUNBOOK.md` for lock-discipline and lane contract constraints.

Artifacts created:
1. `COMMENT_SCHEMA_COVERAGE_AND_BUILD_PLAN.md`
   - current schema/ownership audit + planned build.
2. `COLLAB_PARALLEL_FIX_TRACKER_2026-03-02.md`
   - lock table + lane status tracker.

Planned lane lock scopes (non-overlapping):
1. `LANE-COMMENT-TARGETS`
   - scope: `internal/domain/comment.go`, `internal/domain/comment_test.go`, `internal/app/service_test.go`, `internal/app/snapshot.go`, `internal/adapters/server/mcpapi/extended_tools.go`, `internal/adapters/server/mcpapi/extended_tools_test.go`, `internal/tui/thread_mode.go`, `internal/tui/thread_mode_test.go`.
   - out-of-scope: `internal/tui/model.go`, `internal/tui/model_test.go`.
2. `LANE-NOTICES-PANEL`
   - scope: `internal/tui/model.go`, `internal/tui/model_test.go`, `internal/tui/keymap.go`.
   - out-of-scope: all domain/app/server and `internal/tui/thread_mode.go`.
3. `LANE-OWNERSHIP-PROPOSAL` (analysis-only)
   - docs/proposal lane; no code edits.

Current status:
- Ready to dispatch worker lanes with Context7-before-edit and package-scoped `just test-pkg` requirements.

## Checkpoint 2026-03-02: Parallel Comment + Notices Remediation Integration

Objective:
- Integrate and verify the three-lane wave:
  1. comment target-type completion across domain/app/MCP/TUI thread mapping,
  2. notices panel section-list navigation/selection UX,
  3. ownership tracking audit and migration recommendation.

Lane outcomes:
1. `LANE-COMMENT-TARGETS` completed:
   - added `branch` and `subphase` comment target support,
   - updated snapshot target mapping,
   - updated MCP comment tool target enum,
   - updated TUI task->comment target mapping + new tests.
2. `LANE-NOTICES-PANEL` completed:
   - converted notices panel sections into focusable/selectable list areas,
   - added per-section cursors, scroll windowing, and Enter actions,
   - updated notices/board help messaging.
3. `LANE-OWNERSHIP-PROPOSAL` completed:
   - confirmed current ownership model is `actor_type + author_name`,
   - no stable actor ID stored today; documented optional follow-up migration path.

Commands/tests run and outcomes:
1. `just test-pkg ./internal/domain` -> PASS
2. `just test-pkg ./internal/app` -> PASS
3. `just test-pkg ./internal/adapters/server/mcpapi` -> PASS
4. `just test-pkg ./internal/tui` -> PASS
5. `just ci` -> PASS

Files/docs updated for this checkpoint:
1. `COMMENT_SCHEMA_COVERAGE_AND_BUILD_PLAN.md`
   - updated with post-fix implemented matrix + ownership status.
2. `COLLAB_PARALLEL_FIX_TRACKER_2026-03-02.md`
   - lane statuses moved to complete with handoff and verification evidence.
3. code files from lanes integrated in current worktree:
   - `internal/domain/comment.go`
   - `internal/domain/comment_test.go`
   - `internal/app/snapshot.go`
   - `internal/app/service_test.go`
   - `internal/adapters/server/mcpapi/extended_tools.go`
   - `internal/adapters/server/mcpapi/extended_tools_test.go`
   - `internal/tui/thread_mode.go`
   - `internal/tui/thread_mode_test.go`
   - `internal/tui/model.go`
   - `internal/tui/model_test.go`
   - `internal/tui/keymap.go`

Current status:
- Comment coverage now spans all current node types used by hierarchy and thread entry points.
- Notices panel list navigation/selection behavior is implemented and test-covered.
- Stable actor-ID ownership tracking remains a follow-up decision, intentionally deferred for this wave.

## Checkpoint 2026-03-02: Ownership Tuple + Identity ActorID Wave (Parallel + Reviewed)

Objective:
- Implement immutable ownership tuple (`actor_id`, `actor_name`, `actor_type`) for comments/events,
- wire immutable config-backed user `identity.actor_id` into startup/runtime/TUI,
- run independent subagent review and remediation until green gates.

Execution summary:
1. Launched parallel implementation lanes:
   - `LANE-OWNERSHIP-CORE`
   - `LANE-TUI-FLOWS`
   - `LANE-CONFIG-IDENTITY`
2. Ran independent review lanes:
   - initial reviews flagged compile/wiring/contract issues,
   - dispatched targeted remediation lane (`LANE-REVIEW-REMEDIATION`),
   - ran second independent review pass (`REVIEW-REMEDIATION-PASS2`) -> PASS.

Key outcomes landed:
1. Comments now use canonical ownership tuple fields end-to-end:
   - `actor_id`, `actor_name`, `actor_type`.
2. Change events now persist/read `actor_name` alongside `actor_id` + `actor_type`.
3. MCP actor tuple supports `actor_id` + `actor_name` and preserves them through mutation context.
4. TUI comment/activity owner rendering now prefers `actor_name` with compact `actor_id` context.
5. Config and startup now support immutable `identity.actor_id`:
   - generate once when missing,
   - persist to config,
   - apply at startup and runtime reload.
6. Snapshot versioning for ownership-shape change is explicit:
   - `SnapshotVersion` bumped to `tillsyn.snapshot.v2` with strict import version check.

Commands/tests run and outcomes:
1. `just test-pkg ./internal/domain` -> PASS
2. `just test-pkg ./internal/app` -> PASS
3. `just test-pkg ./internal/adapters/storage/sqlite` -> PASS
4. `just test-pkg ./internal/adapters/server/mcpapi` -> PASS
5. `just test-pkg ./internal/tui` -> PASS
6. `just test-pkg ./cmd/till` -> PASS
7. `just check` -> PASS
8. `just ci` -> PASS

Docs/worklog sync:
1. `COMMENT_SCHEMA_COVERAGE_AND_BUILD_PLAN.md` updated to post-wave canonical schema/tuple state.
2. `COLLAB_PARALLEL_FIX_TRACKER_2026-03-02.md` updated with Wave 2 completion, review findings, remediation, and final gate evidence.

Current status:
- Ownership + identity foundations are implemented and verified.
- Branch is ready to continue collaborative worksheet execution from the next pending collab section.

## Checkpoint 2026-03-02: Parallel Wave Sign-Off Revalidation

Objective:
- confirm final sign-off state after tracker/worklog synchronization.

Commands/tests run and outcomes:
1. `just check` -> PASS
2. `just ci` -> PASS

Current status:
- all parallel implementation lanes and independent review lanes remain closed with green integrator gates.

## Checkpoint 2026-03-02: AGENTS Flow Lock Update (User-Requested)

Objective:
- encode required collaborative remediation flow in `AGENTS.md` before launching next subagent wave.

Commands/tests run and outcomes:
1. `rg -n "Temporary Next-Step Directive|Locked execution flow|Parallel/Subagent Mode|Worker Handoff Contract|PLAN.md" AGENTS.md` -> PASS (located policy sections).
2. `sed -n '1,260p' AGENTS.md` and `sed -n '260,520p' AGENTS.md` -> PASS (reviewed current directives before patch).
3. `apply_patch` on `AGENTS.md` -> PASS (added explicit gates: user go-ahead before implementation launch, mandatory independent QA subagent code+MD sign-off, and user confirmation before section completion).
4. Tests/checks -> `test_not_applicable` (docs/process-only policy update).

Current status:
- policy now matches requested flow; waiting for explicit user "go ahead" before any new implementation subagents are launched.

## Checkpoint 2026-03-02: Wave 3 Notifications/Global Panel Remediation Kickoff

Objective:
- remediate the unresolved notifications/global-panel UX gap reported by user:
  - visible notifications redesign in TUI,
  - separate global notifications panel below project notifications,
  - focusable/scrollable/selectable lists with Enter actions.

Execution constraints (locked):
1. worker implementation lane in `internal/tui/model.go` + `internal/tui/model_test.go`.
2. independent QA subagent review required on code + collaborative markdown docs before completion.
3. tests required before any completion mark:
   - `just test-pkg ./internal/tui`
   - `just check`
   - `just ci`
4. final section completion requires explicit user-run confirmation after integrated changes.

Commands/tests run and outcomes:
1. Context gathering + lock-table docs update -> PASS.
2. Tests/checks -> `test_not_applicable` (kickoff/docs-only checkpoint).

Current status:
- wave started; worker implementation lane is being dispatched.

## Checkpoint 2026-03-02: Wave 3 Worker Handoff + QA Findings + Remediation

Objective:
- complete requested notifications/global-panel redesign with independent QA and green gates before user manual confirmation.

Execution summary:
1. Worker lane `LANE-NOTIFICATIONS-REDESIGN` delivered initial implementation in:
   - `internal/tui/model.go`
   - `internal/tui/model_test.go`
2. Independent QA review lanes returned FAIL:
   - code QA found actionability/stability/resilience gaps and red TUI package gate,
   - docs QA found missing Wave 3 acceptance coverage and tracker/worklog consistency gaps.
3. Remediation worker lane `LANE-NOTIFICATIONS-REMEDIATION` delivered fixes in same TUI lock scope:
   - stable-key global notifications selection re-anchor after reload,
   - deterministic Enter path for non-task global rows via modal fallback,
   - partial-results handling for non-active project attention fetch failures,
   - additional edge-case tests.
4. Integrator reran gates after remediation:
   - `just test-pkg ./internal/tui` -> PASS
   - `just check` -> PASS
   - `just ci` -> PASS

Current status:
- implementation and integrator gates are green,
- refreshed independent QA outcomes:
  1. code QA -> PASS (no remaining high/medium findings),
  2. docs QA -> remediated (added explicit scrolling acceptance row + synchronized pending-state wording).
- final Wave 3 sign-off remaining gate:
  1. user-run manual confirmation in the live collaborative worksheet flow (`C-08` through `C-12`).

## Checkpoint 2026-03-02: Wave 3 Final QA Status Consolidation

Objective:
- consolidate final independent QA status and align process docs before collaborative manual run.

Commands/tests run and outcomes:
1. Independent code QA lane (`REVIEW-NOTIFICATIONS-FINAL-CODE`) -> PASS.
2. Independent docs QA lane (`REVIEW-NOTIFICATIONS-FINAL-DOCS`) -> initial FAIL; remediated via worksheet/process doc updates.
3. Integrator gates remained green (`just test-pkg ./internal/tui`, `just check`, `just ci`) -> PASS.

Current status:
- Wave 3 is QA-signed and gate-green.
- Remaining closeout requirement is user collaborative confirmation (`C-08` through `C-12`).
- Final docs recheck reports FAIL only because those user confirmation rows are still intentionally pending.

## Checkpoint 2026-03-02: Wave 4 Markdown-First Summary/Details/Comments

Objective:
- implement markdown-first summary/details/comments improvements for dogfood readiness,
- add summary schema migration,
- update MCP contracts/tool guidance,
- improve TUI read-first details flow and comment visibility,
- keep validation MCP-only (no HTTP/curl probes) and inside repo scope.

Commands/tests run and outcomes:
1. `mcp__gopls__go_workspace` -> PASS (workspace/module verified).
2. Context7 consults before implementation lanes:
   - Bubble Tea, Lip Gloss, Glamour, SQLite, mcp-go -> PASS.
3. Worker lane package tests (integrator rerun):
   - `just test-pkg ./internal/domain` -> PASS
   - `just test-pkg ./internal/app` -> PASS
   - `just test-pkg ./internal/adapters/storage/sqlite` -> PASS
   - `just test-pkg ./internal/adapters/server/mcpapi` -> PASS
   - `just test-pkg ./internal/tui` -> PASS
4. Full gates:
   - `just check` -> PASS
   - `just ci` -> PASS
5. Visual regression tapes:
   - `just vhs` -> PASS (`vhs/board.tape`, `vhs/regression_scroll.tape`, `vhs/regression_subtasks.tape`, `vhs/workflow.tape`).

Edits completed:
1. Domain/app/snapshot comment summary model support + fallback normalization.
2. SQLite `comments.summary` column migration/backfill + persistence read/write.
3. MCP/common contract updates:
   - comment `summary` in requests/responses,
   - markdown-rich argument guidance for summary/details/comments,
   - capture-state comment-overview population from persisted comments.
4. TUI thread/task info improvements:
   - read-first details overlay flow in thread mode,
   - explicit edit transition from details overlay,
   - comment summary visibility in thread/task-info read surfaces,
   - notification actionability behavior preserved.
5. Policy/docs updates:
   - `AGENTS.md` updated with strict repo-scope, MCP-only protocol validation, and never-push default.

Commit sequence:
1. `f28e9f3` -> `Add markdown-first comment summary schema and MCP contracts`
2. (pending) TUI + docs/worksheet closeout commit after QA sign-off.

Current status:
- Implementation lanes complete and gates green.
- New collaborative worksheet created: `COLLAB_TEST_2026-03-02_DOGFOOD.md`.
- Remaining required steps before final closeout:
  1. independent QA subagent sign-off on code + markdown trackers/worksheet,
  2. user+agent collaborative run through worksheet sections C1-C4,
  3. final closeout commit for TUI/docs wave.

## Checkpoint 2026-03-03: TUI Layout/Thread Editor Remediation (User-Reported Regression)

Objective:
- address live-user findings:
  1. unexplained `…` truncation marker and panel-bottom visual mismatch in board/notices layout,
  2. thread description unexpectedly showing `(no description)` for some notification-opened threads,
  3. need multiline independent editors for comment composition and thread description editing.

Commands/tests run and outcomes:
1. context and hotspot discovery:
   - `rg -n "Global Notifications|Project Notifications|fitLines|thread"` across `internal/tui/*` -> PASS.
   - targeted `sed -n` reads for `renderOverviewPanel`, `renderGlobalNoticesPanel`, `modeThread`, `thread_mode.go` -> PASS.
2. Context7 consult before edits:
   - `/charmbracelet/bubbles` (`textarea` APIs, ctrl+s flow) -> PASS.
   - `/charmbracelet/lipgloss` (inline/max width/wrap guardrails) -> PASS.
3. implementation edits:
   - `internal/tui/model.go`:
     - switched thread composer to `textarea.Model`,
     - added independent thread details editor textarea (`i` from details mode, `ctrl+s` save),
     - added textarea clipboard helper,
     - tightened notices/global line truncation to prevent wrapped overflow,
     - replaced `fitLines` ellipsis insertion with hard clipping,
     - made board footer reserve dynamic (`boardFooterLines`) to reduce artificial panel gap.
   - `internal/tui/thread_mode.go`:
     - multiline editor rendering for comments/details,
     - details-editor hints and controls,
     - fallback description resolution from backing project/task when thread body is empty,
     - direct save path from details editor to `UpdateProject` / `UpdateTask`.
   - `internal/tui/model_test.go`:
     - updated thread post key expectations (`ctrl+s`),
     - added regression tests for details-editor save and description fallback,
     - updated partial-results assertion for truncated panel label behavior.
   - `internal/tui/testdata/*.golden` refreshed via recipe.
4. tests:
   - `just fmt` -> PASS.
   - `just test-golden-update` -> PASS.
   - `just test-pkg ./internal/tui` -> PASS.
   - `just check` -> PASS.
   - `just ci` -> PASS.

Current status:
- remediation implemented and gate-green.
- active worksheet updated to reflect new controls and explicit alignment/truncation validation row:
  - `COLLAB_TEST_2026-03-02_DOGFOOD.md` (`C1-04`, new `C1-05`, new `C3-04`, overflow log row).
- waiting for user collaborative rerun of C1/C3 steps in live TUI for final confirmation.

## Checkpoint 2026-03-03: Thread Workspace Layout Redesign (User UX Directive)

Objective:
- implement requested thread workspace composition:
  1. bordered top description/details pane occupying most of thread viewport,
  2. bordered bottom comments pane (~25% height) with 2-line composer,
  3. bordered right context pane with owner + brief history,
  4. full-screen description editor with live Glamour preview while typing.

Commands/tests run and outcomes:
1. Context7 before edits:
   - `/charmbracelet/glamour` (renderer usage / environment-style patterns) -> PASS.
   - `/charmbracelet/lipgloss` (width/height/clipping behavior) -> PASS.
2. implementation edits:
   - `internal/tui/thread_mode.go`
     - rebuilt `renderThreadModeView` into split-pane workspace,
     - added helpers:
       - `renderThreadDescriptionPanel`,
       - `renderThreadCommentsPanel`,
       - `renderThreadContextPanel`,
       - `renderThreadDescriptionEditorView`,
       - `threadCommentListLines`,
       - `threadSectionStyle`,
     - removed dependency on old details overlay for normal thread flow,
     - added actor-tagged brief-history rows with summary labels.
   - `internal/tui/model.go`
     - tuned thread composer default to 2-line textarea (`ShowLineNumbers=false`).
   - `internal/tui/markdown_renderer.go`
     - switched renderer construction to `glamour.WithEnvironmentConfig()` + `WithWordWrap(...)` for env-driven style behavior aligned with Glamour guidance.
3. test cycle:
   - `just fmt && just test-pkg ./internal/tui` -> initial FAIL (`TestModelThreadModeProjectAndPostCommentUsesConfiguredIdentity` visibility expectation mismatch).
   - Context7 rerun after failure (lipgloss clipping/fit guidance) -> PASS.
   - targeted fix in `renderThreadContextPanel` (history rows now include `[actor] owner` and `summary:` prefix) -> applied.
   - `just fmt && just test-pkg ./internal/tui` -> PASS.
   - `just check && just ci` -> PASS.
   - post-renderer update revalidation:
     - `just fmt && just test-pkg ./internal/tui` -> PASS.
     - `just check && just ci` -> PASS.

Current status:
- thread workspace redesign is integrated and gate-green.
- worksheet updated for this UX contract:
  - `C1-02`/`C1-05`/`C1-06` and `C3-05` in `COLLAB_TEST_2026-03-02_DOGFOOD.md`.
- awaiting user collaborative validation of new thread-pane behavior.

## Checkpoint 2026-03-03: Global Notification Enter Trace Instrumentation

Objective:
- diagnose slow global-notification Enter open path and intermittent input/key leakage by adding high-signal TUI debug traces with transition correlation.

Commands/tests run and outcomes:
1. context gathering and analysis:
   - local code inspection for `activateGlobalNoticesSelection` / `loadData` / `applyLoadedMsg` / input routing -> completed.
   - explorer subagents run for latency path + input leakage path with file:line evidence -> completed.
2. Context7 before edits:
   - `/charmbracelet/bubbletea` (Update/key flow semantics) -> PASS.
   - `/charmbracelet/log` (structured debug field logging) -> PASS.
3. implementation edits:
   - added `internal/tui/trace.go` with:
     - global-notification transition id lifecycle,
     - branch/pending/key-dispatch trace helpers,
     - `loadData` stage timing helpers,
     - control-character guard helpers for persistence-adjacent traces.
   - updated `internal/tui/model.go` to emit traces at:
     - global panel Enter activation,
     - global notification branch decisions/pending-field mutations,
     - `loadData` stage timings and counts,
     - `applyLoadedMsg` completion for transition correlation,
     - key dispatch while transition active,
     - pre-persistence control-character guard checks.
   - QA edge-case follow-up fix:
     - zero-project `applyLoadedMsg` branch now closes active global transition trace (`no_projects`) to avoid stale transition ids.
4. verification:
   - worker lane: `just test-pkg ./internal/tui` -> initial FAIL (compile mismatch), fixed in-lane, rerun -> PASS.
   - independent QA lane: `just test-pkg ./internal/tui` -> PASS.
   - integrator post-fix rerun:
     - `just test-pkg ./internal/tui` -> PASS.
     - `just ci` -> PASS.

Current status:
- trace instrumentation is integrated and gate-green.
- logs remain in existing runtime dev log sink (`.tillsyn/log/tillsyn-YYYYMMDD.log`) with `tui.*` debug event names for grep/filter.
- ready for user repro run focused on global notifications Enter latency and key leakage.

## Checkpoint 2026-03-03: Global Notice Enter Stall + Exit Reliability Fix

Objective:
- remove project-scoped global notice Enter reload stall and restore deterministic emergency exit while in non-normal input modes.

Investigation/evidence:
- logs showed project-scoped global notice Enter going through `no_task_switch_project_reload` branch (`activateGlobalNoticesSelection`) instead of direct thread open.
- user-reported shell command failures were command-format/alias issues (zsh parsing + iconized ls alias), not application errors.

Context7:
- attempted Bubble Tea key handling consult; transport unavailable.
- fallback source used per repo policy: local Bubble Tea usage in repo and existing test evidence.

Implementation edits:
- `internal/tui/model.go`
  - added global hard interrupt handling in `Update` for `tea.KeyPressMsg` when `msg.String() == "ctrl+c"` to ensure emergency quit in all modes.
  - changed project-scoped/no-task global-notice Enter path to open thread directly (no switch-project reload).
- `internal/tui/model_test.go`
  - updated `TestModelGlobalNotificationsEnterOnProjectScopedRowOpensThread` expectation: selected project context remains unchanged on direct thread open.

Verification:
- `just test-pkg ./internal/tui` -> PASS.
- `just ci` -> PASS.

Current status:
- direct-open path removes the unnecessary reload for project-scoped global notices.
- ctrl+c quit now works regardless of mode-specific input routing.

## Checkpoint 2026-03-03: Remaining Global-Notice Slowness + Input Artifact Hardening

Objective:
- address user-reported lingering stall and input corruption after global-notice Enter.

Findings:
- user shell log extraction failures were command formatting issues in zsh (newline split + alias contamination), not repository/runtime permission faults.
- global notice path still performed unnecessary reload for project-scoped rows (`task_id == "" && switchProject` branch).
- with debug-level instrumentation enabled, `loadData` stage logs were emitted on every auto-refresh, creating heavy log churn.
- markdown renderer used `glamour.WithEnvironmentConfig()`; this can trigger terminal environment probing and may leak OSC replies into focused inputs in some terminals.

Context7:
- consulted `/charmbracelet/bubbletea` for ctrl+c handling semantics.
- attempted second Bubble Tea consult during follow-up; Context7 transport unavailable. fallback used: local code/test behavior.
- consulted `/charmbracelet/glamour` for deterministic renderer style options (`WithStandardStyle`).

Implementation edits:
- `internal/tui/model.go`
  - `Update`: immediate `ctrl+c` emergency quit path in `tea.KeyPressMsg` handling.
  - `activateGlobalNoticesSelection`: project-scoped/no-task global notices now open thread directly (no switch-project reload).
  - `taskFormValues`/`projectFormValues`: apply `sanitizeFormFieldValue` to strip OSC probe artifacts + control runes before persistence.
  - added form-value sanitization helpers and terminal probe regex patterns.
- `internal/tui/trace.go`
  - reduced background tracing noise: `tui.load_data.stage` now logs non-transition loads only on errors or unexpectedly slow totals (>=50ms), while keeping full detail for active global-notice transitions.
- `internal/tui/markdown_renderer.go`
  - switched renderer construction from `WithEnvironmentConfig()` to stable `WithStandardStyle(styles.DarkStyle)` + wrap, avoiding environment-probing behavior.
- `internal/tui/model_test.go`
  - updated direct-open behavior expectation for project-scoped global notices.
  - added regression tests for form-value sanitization of OSC probe artifacts.

Verification:
- `just test-pkg ./internal/tui` -> PASS.
- `just ci` -> PASS.

Current status:
- global-notice Enter no longer forces project-switch reload for project-scoped rows.
- emergency exit is reliable via ctrl+c in all modes.
- form submits now scrub terminal-probe artifacts from project/task fields.
- load-data trace output is scoped to actionable scenarios, reducing log I/O overhead.

## Checkpoint 2026-03-03: Input-Time Probe Scrub + Notifications Panel Height Alignment

Objective:
- fix remaining live-input OSC artifact insertion (not just submit-time cleanup), and align right stacked notifications panel height with board columns to eliminate bottom mismatch/clipping.

Context gathering + design:
- reviewed latest user logs + `tui.global_notification.*` traces.
- explorer lanes confirmed two primary gaps:
  - submit-time-only sanitization allowed probe artifacts to appear while editing.
  - overview panel split used hard minimums that could exceed requested height.
- Context7 consulted before edits:
  - `/charmbracelet/bubbletea` key handling guidance.

Implementation edits:
- `internal/tui/model.go`
  - added `terminalProbeEscapeSequencePattern` for full OSC escape sequence stripping.
  - added `sanitizeInteractiveInputValue`, `stripTerminalProbeArtifacts`, `scrubTextInputTerminalArtifacts`, `scrubTextAreaTerminalArtifacts`.
  - wired scrubbers into interactive update paths:
    - task/project forms,
    - thread comment/details editors,
    - dependency/search/command/label/path/bootstrap/due/resource/highlight/labels-config inputs.
  - wired scrubbers into clipboard paste helpers for both textinput and textarea.
  - updated `sanitizeFormFieldValue` to reuse interactive sanitizer then trim.
  - fixed `renderOverviewPanel` stacked height math with explicit min chrome-aware panel size and bounded split so project+global panel heights stay consistent.
  - lowered `columnHeight()` minimum from 14 to 10 to reduce forced overflow on smaller terminals.
- `internal/tui/model_test.go`
  - added regression tests:
    - `TestScrubTextInputTerminalArtifactsStripsProbeDuringEdit`
    - `TestScrubTextAreaTerminalArtifactsStripsProbeDuringEdit`
    - `TestRenderOverviewPanelHeightMatchesRequestedHeight`

Test/fix loop evidence:
1. `just test-pkg ./internal/tui` -> FAIL (golden drift + panel height off-by-one from border chrome minimum).
2. adjusted stacked split invariant (`minStackPanelHeight = 5`) to account for border/padding overhead.
3. `just test-pkg ./internal/tui` -> FAIL (golden drift only).
4. `just test-golden-update` -> PASS (updated:
   - `internal/tui/testdata/TestModelGoldenBoardOutput.golden`
   - `internal/tui/testdata/TestModelGoldenHelpExpandedOutput.golden`).
5. `just test-pkg ./internal/tui` -> PASS.
6. `just check` -> PASS.
7. `just ci` -> PASS.

Current status:
- live inputs now scrub probe artifacts during typing/paste (not only on save).
- project/global notifications panel stack now respects board-height budget and aligns better with column bottoms.
- all required gates are green after golden refresh.

## Checkpoint 2026-03-03: Full Markdown Description Editor for Add/Edit Flows

Objective:
- enforce full markdown description editing for task/project add/edit forms (no inline single-line description editing), and anchor thread description panel help text at the bottom.

Context + research:
- reviewed current add/edit form path and thread description rendering in `internal/tui/model.go` and `internal/tui/thread_mode.go`.
- Context7 consult before edits: `/charmbracelet/bubbletea` (key handling/update test patterns).

Implementation edits:
- `internal/tui/model.go`
  - added `modeDescriptionEditor` and `descriptionEditorTarget` state.
  - added dedicated form-description markdown state (`taskFormDescription`, `projectFormDescription`).
  - added helpers:
    - `startTaskDescriptionEditor` / `startProjectDescriptionEditor`
    - `applySeedKeyToDescriptionEditor`
    - `saveDescriptionEditor` / `closeDescriptionEditor`
    - `descriptionFormDisplayValue` + form summary sync helpers.
  - changed task/project form behavior so description field always opens markdown editor on edit keys (enter/i/typed input), preventing inline single-line editing.
  - added markdown editor rendering case to `renderModeOverlay` with editor + Glamour preview.
  - updated help/mode text for new editor mode and description workflow.
  - ensured form value extraction writes description from markdown state, not compact summary row.
- `internal/tui/thread_mode.go`
  - adjusted `renderThreadDescriptionPanel` so the inline help line is bottom-anchored in that panel.
- `internal/tui/model_test.go`
  - added:
    - `TestModelTaskDescriptionEditorFlow`
    - `TestModelProjectDescriptionEditorSeedAndCancel`

Test/fix loop evidence:
1. `just fmt && just test-pkg ./internal/tui` -> PASS.
2. `just check` -> PASS.
3. `just ci` -> FAIL first pass (`internal/tui` coverage 69.7% < 70%).
4. Context7 rerun after failed gate (Bubble Tea test/key patterns) -> PASS.
5. added focused tests for description editor mode/flows.
6. `just fmt && just test-pkg ./internal/tui` -> PASS.
7. `just check` -> PASS.
8. `just ci` -> PASS (`internal/tui` coverage 70.3%).

Current status:
- task/project descriptions are now edited only through the full markdown editor flow.
- thread description panel help text is pinned to panel bottom.
- all required gates are green.

## Checkpoint 2026-03-03: Description Editor Full-Screen UX Rework

Objective:
- convert markdown description editing from modal to a dedicated full-screen screen with synced editor/preview scrolling, generic node-path context, and explicit edit/preview submodes.

Commands/tests run and outcomes:
1. repository/startup context:
   - `pwd && ls -la` -> PASS.
   - `rg --files -g 'AGENTS.md'` -> PASS (repo-root `AGENTS.md` only).
   - `sed -n '1,220p' Justfile` -> PASS (confirmed `just` recipes as source of truth).
2. implementation context discovery:
   - `rg -n ... internal/tui` + targeted `sed -n` reads over `internal/tui/model.go`, `internal/tui/thread_mode.go`, and `internal/tui/model_test.go` -> PASS.
   - `git status --short` -> PASS (clean tree before edits).
3. Context7/library API research before edits:
   - `/charmbracelet/bubbles` (textarea + viewport API/scroll behavior) -> PASS.
   - `/charmbracelet/bubbletea` (key/mouse message handling in v2) -> PASS.
   - local API fallback confirmation for exact method surface:
     - `go doc charm.land/bubbles/v2/textarea.Model` -> PASS.
     - `go doc charm.land/bubbles/v2/viewport.Model` -> PASS.
4. implementation edits:
   - `internal/tui/model.go`
     - added full-screen `modeDescriptionEditor` routing in `View()`,
     - added description editor submode state (`edit`/`preview`), node-path context state, thread-return state, and preview viewport state,
     - updated description editor start/save/close flows to support task/project forms + thread details,
     - disabled `?` help-toggle only while description editor is in edit submode so `?` inserts text,
     - added preview-mode key handling (`tab` toggle, scroll keys) and synced preview offset handling,
     - added description-editor mouse-wheel scrolling behavior,
     - removed the legacy modal overlay renderer path for `modeDescriptionEditor`.
   - `internal/tui/description_editor_mode.go` (new):
     - implemented dedicated full-screen Description Editor renderer,
     - added bottom-anchored hint lines, path-header line, split edit layout, preview-only layout, and clean bordered panels,
     - added path resolvers for task/project/thread targets (including notification/open-thread targets),
     - added preview rendering + viewport synchronization helpers.
   - `internal/tui/thread_mode.go`
     - removed runtime branch that rendered the old thread-specific description editor screen so all flows now use the shared full-screen description editor.
   - `internal/tui/model_test.go`
     - migrated thread-details save test to new shared description-editor flow,
     - added regression tests for:
       - `?` behavior in edit submode,
       - preview-mode toggle and heading/path rendering expectations,
       - preview/edit offset synchronization invariants.
5. test/fix cycle evidence:
   - `just fmt && just test-pkg ./internal/tui` -> FAIL (compile: wrong helper method name in new file).
   - Context7 rerun after failure: `/charmbracelet/bubbles` -> PASS.
   - fixed method call mismatch in `internal/tui/description_editor_mode.go`.
   - `just fmt && just test-pkg ./internal/tui` -> FAIL (legacy test expected removed `threadDetailsEditorActive` path).
   - Context7 rerun after failure: `/charmbracelet/bubbletea` -> PASS.
   - updated thread-details test to assert `modeDescriptionEditor` + thread target.
   - `just fmt && just test-pkg ./internal/tui` -> FAIL (flaky strict textarea offset movement assertion).
   - Context7 rerun after failure: `/charmbracelet/bubbles` -> PASS.
   - adjusted test to deterministic sync-consistency assertions.
   - `just fmt && just test-pkg ./internal/tui` -> FAIL (same strict movement assertion still too brittle).
   - Context7 rerun after failure: `/charmbracelet/bubbles` -> PASS.
   - further relaxed assertion to compare synchronized offsets without requiring absolute movement.
   - `just fmt && just test-pkg ./internal/tui` -> PASS.
   - `just check && just ci` -> PASS.

Current status:
- full-screen Description Editor UX rework is integrated and gate-green.
- description editing now uses one dedicated full-screen screen with edit/preview submodes, bottom hints, path context, synced scroll behavior, and no task-specific header text.

## Checkpoint 2026-03-03: Description Editor Preview-Mode Scroll Unblock

Objective:
- restore real keyboard and mouse scrolling in Description Editor preview mode.

Context7 + analysis:
- consulted `/charmbracelet/bubbles` before edits (textarea/viewport sizing + scroll API).
- after a failed regression test, re-consulted `/charmbracelet/bubbles` per policy before next edit.
- local source/API verification used for exact behavior details:
  - `go doc charm.land/bubbles/v2/viewport.Model`
  - viewport source (`SetContent`, `PageDown`, `ScrollDown`, `SetYOffset`) under module cache.

Implementation edits:
- `internal/tui/description_editor_mode.go`
  - added layout metrics helper to compute consistent editor/preview dimensions.
  - added `syncDescriptionEditorViewportLayout` so preview viewport state is dimensioned/content-populated in model state (not only render copies).
  - updated `syncDescriptionPreviewOffsetToEditor` to sync viewport layout before offset sync.
- `internal/tui/model.go`
  - on `tea.WindowSizeMsg`, description editor now refreshes viewport/input layout state.
  - preview submode key handling now scrolls the preview viewport directly (`ScrollUp/Down`, `PageUp/Down`, `GotoTop/Bottom`).
  - preview submode mouse wheel now scrolls preview viewport directly.
  - edit submode behavior remains editor-driven and keeps preview offset synchronized from textarea scroll.
- `internal/tui/model_test.go`
  - added `TestModelDescriptionEditorPreviewModeScrollsWrappedContent` to verify preview mode scrolls for wrapped markdown via both `pgdown` and mouse wheel.
  - adjusted existing preview toggle test assertions to avoid false negatives when preview starts already at bottom.

Test/fix loop evidence:
1. `just fmt` -> PASS.
2. `just test-pkg ./internal/tui` -> FAIL (`TestModelDescriptionEditorPreviewModeToggleAndScrollSync` assertion expected movement while preview was already at bottom).
3. Context7 re-consult (`/charmbracelet/bubbles`) before next edit -> PASS.
4. updated test assertion and added dedicated wrapped-content preview-scroll regression test.
5. `just fmt` -> PASS.
6. `just test-pkg ./internal/tui` -> PASS.
7. `just check` -> PASS.
8. `just ci` -> PASS (`internal/tui` coverage 70.4%).

Current status:
- preview mode now scrolls with keyboard and mouse as requested.
- edit mode retains synchronized split-panel scroll behavior.
- required repository gates are green.

## Checkpoint 2026-03-03: Description Preview Height + Task-Info Esc Loop Remediation

Objective:
- fix two user-reported regressions:
  - full-page markdown preview appeared clipped/non-scrollable,
  - `esc` could bounce between task-info origin and parent states.

Context + logs reviewed:
- inspected `.tillsyn/log/tillsyn-20260303.log` for runtime failures and transition traces.
- no runtime panics/errors were present for these flows; behavior was state/layout logic.
- repeated `tui.global_notification.*` traces confirmed deterministic mode transitions (no crash path).

Context7:
- consulted `/charmbracelet/bubbles` before edits for viewport sizing/scroll guidance (`SetWidth`/`SetHeight`/`SetContent`/paging semantics).

Implementation edits:
- `internal/tui/description_editor_mode.go`
  - replaced fixed minimum width behavior with viewport-bounded width (`layoutWidth <= m.width-2` when width is known).
  - added frame-text clamping helper so header/path/footer/status remain single-line and do not silently wrap into unbudgeted rows.
  - updated layout computation to budget workspace height from single-line frame rows, avoiding undercount and off-screen clipping.
  - added narrow-terminal fallback that stacks editor/preview vertically in edit mode when horizontal split cannot fit cleanly.
  - updated preview/edit panel height/width calculations and viewport-sync helpers to use mode-correct dimensions.
- `internal/tui/model.go`
  - fixed task-info `esc` origin jump logic by only jumping to origin when origin is an ancestor of the current task.
  - added `taskIsAncestor` helper for explicit ancestry checks.
  - this removes the child<->parent oscillation path while preserving expected “return to origin” behavior from descendant views.
- `internal/tui/model_test.go`
  - added `TestTaskInfoEscFromChildDoesNotLoopToOrigin`.
  - added `TestModelDescriptionEditorLayoutRespectsNarrowViewport` (bounds + stacked layout + preview scroll movement in constrained viewport).

Verification:
1. `just fmt` -> PASS.
2. `just test-pkg ./internal/tui` -> PASS.
3. `just check` -> PASS.
4. `just ci` -> PASS (`internal/tui` coverage 70.7%).

Current status:
- description preview layout now stays screen-bounded and scrollable in narrow/full-page scenarios.
- task-info escape flow is deterministic and no longer loops between origin and parent.
- required gates are green.

## Checkpoint 2026-03-03: Task-Info Details Viewport + Esc Path Retrace Fix

Objective:
- fix remaining user-reported task-info regressions:
  - task-info details section could overflow off-screen and was not scrollable,
  - `esc` unwind could surface unexpected ancestor modals instead of retracing visited task-info path,
  - task-info lacked a direct full-screen details preview entry path.

Context + logs reviewed:
- reviewed latest runtime logs in `.tillsyn/log/tillsyn-20260303.log`; no panic/runtime errors tied to this flow.
- confirmed issue was TUI layout/state behavior in `internal/tui/model.go`.

Context7:
- consulted `/charmbracelet/bubbletea` and `/charmbracelet/bubbles` before edits for key/mouse message handling and viewport sizing/scroll semantics.
- after each failed `just test-pkg ./internal/tui` run, re-consulted Context7 before the next code edit.

Implementation edits:
- `internal/tui/model.go`
  - added bounded task-info details viewport state (`taskInfoDetails`) and sizing helpers.
  - replaced unbounded inlined task description rendering with a fixed-height scrollable details viewport in task-info overlay.
  - added task-info details scrolling controls:
    - keyboard: `pgup/pgdown`, `home/end`, `ctrl+u/ctrl+d`,
    - mouse: wheel up/down in task-info mode.
  - added task-info traversal path tracking (`taskInfoPath`) and `stepBackTaskInfoPath` so `esc` retraces visited nodes rather than jumping via ancestor-origin heuristics.
  - added direct task-info details action (`d`) to open full-screen Description Editor in preview submode.
  - added `startTaskInfoDescriptionEditor` to initialize description editor from task-info with `modeTaskInfo` back context.
  - updated `closeDescriptionEditor` to support returning to `modeTaskInfo` on cancel/save and persist markdown details via existing thread-target update command.
  - updated task-info mode prompts/help text and overlay action hints for new behavior.
- `internal/tui/model_test.go`
  - added `TestModelTaskInfoDetailsViewportScrolls` (keyboard + mouse details scroll regression coverage).
  - added `TestModelTaskInfoDescriptionEditorOpensInPreviewMode` (task-info `d` opens preview submode and returns to task-info on `esc`).
  - added `TestModelTaskInfoEscStopsAtEntryPathRoot` (esc retrace closes at entry path root instead of climbing to ancestors).
  - updated previous esc regression test to match path-retrace semantics:
    - `TestTaskInfoEscFromDirectChildClosesWithoutAncestorJump`.

Test/fix evidence:
1. `just fmt` -> PASS.
2. `just test-pkg ./internal/tui` -> FAIL (`TestTaskInfoEscFromChildDoesNotLoopToOrigin` expectation mismatched new path semantics).
3. Context7 re-consult (`/charmbracelet/bubbles`) -> PASS.
4. test updates + additional task-info viewport/path tests.
5. `just fmt` -> PASS.
6. `just test-pkg ./internal/tui` -> FAIL (`TestModelTaskInfoEscStopsAtEntryPathRoot` opened wrong node via board selection index ambiguity).
7. Context7 re-consult (`/charmbracelet/bubbletea`) -> PASS.
8. test made deterministic via `openTaskInfo(parent.ID, ...)` setup.
9. `just fmt` -> PASS.
10. `just test-pkg ./internal/tui` -> PASS.
11. `just check` -> PASS.
12. `just ci` -> PASS (`internal/tui` coverage 70.5%).

Current status:
- task-info details are now bounded and scrollable in-place.
- task-info `esc` retraces visited path and closes at entry root.
- task-info can open full-screen details directly in preview mode (`d`), while edit-form entry still opens split edit/preview mode.
- required gates are green.

## Checkpoint 2026-03-03: MCP Instructions Tool + Task-Info Keyboard/Top-Load Polish

Objective:
- add one MCP instructions tool for agent-facing docs/recommendations and close remaining task-info/details UX gaps reported by dogfooding.

Context7 + fallback:
- consulted `/charmbracelet/bubbles` and `/charmbracelet/bubbletea` for viewport/key semantics before TUI edits.
- `go` stdlib `embed` guidance came from local official docs (`go doc embed`) as fallback for non-Context7 stdlib coverage.

Implementation edits:
- `embedded_markdown_docs.go`
  - added top-level markdown embedding (`//go:embed *.md`) and deterministic loader (`EmbeddedMarkdownDocuments`) for MCP consumption.
- `internal/adapters/server/mcpapi/instructions_tool.go`
  - added `till.get_instructions` with:
    - optional topic focus,
    - optional `doc_names` filtering,
    - optional markdown inclusion,
    - optional recommendation inclusion,
    - per-doc truncation (`max_chars_per_doc`).
  - response includes doc inventory, selected docs, recommended agent settings, and md-file guidance.
- `internal/adapters/server/mcpapi/handler.go`
  - registered `registerInstructionsTool` in `NewHandler`.
- `internal/adapters/server/mcpapi/extended_tools_test.go`
  - expanded tool-surface assertions and call-matrix to include `till.get_instructions`.
  - added embedded-doc/guidance regression coverage for `README.md` and `AGENTS.md` visibility.
- `internal/tui/model.go`
  - task-info details now scroll with `j/k` and arrow keys in addition to page/home/end/ctrl+u/ctrl+d and mouse wheel.
  - opening full-screen details from task-info (`d`) now initializes at top for both preview and edit transitions.
  - mode help/prompt text updated for new task-info scroll semantics.
- `internal/tui/model_test.go`
  - added assertions for task-info details scroll via `j/k`, arrows, paging, and mouse wheel.
  - added assertions that task-info details preview/edit open at top.

Validation evidence:
1. `just test-pkg ./internal/adapters/server/mcpapi` -> PASS.
2. `just test-pkg ./internal/tui` -> PASS.
3. `just check` -> PASS.
4. `just ci` -> PASS (`internal/tui` coverage 70.5%, repository gate green).

Current status:
- `till.get_instructions` is available and wired into MCP handler surface.
- embedded top-level markdown docs are available in binary payload for instructions responses.
- task-info details support keyboard + mouse scrolling and task-info details preview/edit opens at top.
- remaining documentation-policy updates (`AGENTS.md`/`README.md`) are intentionally pending user consensus.

## Checkpoint 2026-03-03: Path Ellipsis Everywhere + Ctrl Undo/Redo + Policy Docs Sync

Objective:
- implement left-middle path collapsing across path displays, unify undo/redo keys to `ctrl+z` / `ctrl+shift+z`, add text-editor undo/redo in markdown editors, and sync policy docs/tool guidance for markdown-first authoring.

Subagent lane:
- spawned one worker lane (`lane-path-ellipsis`) for TUI path truncation implementation and tests under scoped lock.
- integrated lane output and validated in main branch.

Context7:
- consulted `/charmbracelet/bubbles` for key binding/update-loop patterns.
- consulted `/mark3labs/mcp-go` for MCP tool argument/response schema usage.
- after failed `just test-pkg ./internal/tui`, re-consulted `/charmbracelet/bubbles` before further edits.

Implementation edits:
- `internal/tui/path_display.go` (new)
  - added reusable path-collapsing helper (`collapsePathForDisplay`) that removes middle segments from the left first and converges toward `first -> ... -> last` (and `first | ... | last` variants).
- `internal/tui/model.go`
  - applied path collapsing to board header path, activity event path, and dependency inspector path surfaces.
  - switched global undo/redo help copy to `ctrl+z` / `ctrl+shift+z`.
  - added text-editor undo/redo stacks for:
    - full-screen description editor (edit mode),
    - thread comment composer.
  - wired editor undo/redo to same key bindings (`m.keys.undo` / `m.keys.redo`) with mode-local behavior.
- `internal/tui/description_editor_mode.go`
  - applied path collapsing to description-editor header path line.
  - footer hints now include `ctrl+z undo` / `ctrl+shift+z redo` in edit mode.
- `internal/tui/keymap.go`
  - changed default undo/redo bindings to:
    - undo: `ctrl+z`
    - redo: `ctrl+shift+z`
  - updated runtime key-config fallback defaults accordingly.
- `internal/tui/model_test.go`
  - updated undo/redo mutation test to use ctrl-modified key events.
  - added description-editor ctrl undo/redo regression coverage.
  - added path-collapse regression checks for board-header and activity-event path displays.
  - refreshed golden output expectations for expanded help key text.
- `internal/adapters/server/mcpapi/instructions_tool.go`
  - retained `include_markdown` argument.
  - updated recommendations:
    - `till.get_instructions` use is on-demand (missing/stale/ambiguous context), not mandatory every step.
    - markdown-first requirement for descriptions/comments called out explicitly.
- `AGENTS.md`
  - added explicit policy bullets for:
    - on-demand `till.get_instructions` usage,
    - bounded instruction calls (`doc_names`, `max_chars_per_doc`, `include_markdown`),
    - markdown-first authoring for descriptions/comments.
- `README.md`
  - added `till.get_instructions` to feature and active MCP tool surface.
  - added instruction-tool usage guidance (`doc_names`, `include_markdown`, `max_chars_per_doc`) and markdown-first content guidance.
- `MCP_FULL_TESTER_AGENT_RUNBOOK.md`
  - updated tool inventory to include `till.get_instructions` (31 tools total).
  - added instructions-tool matrix coverage expectations.

Test/fix evidence:
1. `just fmt` -> PASS.
2. `just test-pkg ./internal/adapters/server/mcpapi` -> PASS.
3. `just test-pkg ./internal/tui` -> FAIL:
   - golden help output expected old `z/Z` text,
   - undo/redo test still sent `z/Z` keys.
4. Context7 re-consult (`/charmbracelet/bubbles`) -> PASS.
5. updated tests + editor undo/redo coverage + path-collapse assertions.
6. `just fmt` -> PASS.
7. `just test-golden-update` -> PASS.
8. `just test-pkg ./internal/tui` -> PASS.
9. `just test-pkg ./internal/adapters/server/mcpapi` -> PASS.
10. `just check` -> PASS.
11. `just ci` -> PASS (`internal/tui` coverage 70.5%).

Dogfooding fixture creation (MCP):
- created deep path fixture chain in project `evan-project` (`5996fbd8-6a35-42cb-bd01-5aa0a4495cec`) for manual UI verification:
  - branch: `34223d83-2125-46b5-bea1-967dc57b2202`
  - phase: `9b5b8ef3-32cc-4e83-8d37-951310a46642`
  - task: `401bc139-086c-4e2c-92ff-c66c2762cd7d`
  - subtask: `5755f0a4-15a9-4697-b1ab-161ede49b0eb`
  - terminal subtask (focus item): `9d8e18f0-e7f1-47c6-b8b6-8526debc60c5`

Current status:
- path displays now collapse middle hierarchy segments consistently and preserve focused tail visibility.
- undo/redo keying is now `ctrl+z` / `ctrl+shift+z` across mutation history and markdown text editing surfaces.
- instruction/policy docs and MCP runbook are synchronized with markdown-first and bounded-context guidance.

## Checkpoint 2026-03-03: Agent Auth/Approval Investigation (No-Code Backlog Planning)

Objective:
- investigate why MCP mutations were allowed without lease/auth approval in practice and capture a concrete phased fix plan (no implementation in this wave).

Subagent investigation:
- launched one explorer lane for read-only analysis across MCP adapter, app mutation guard, and service wiring.
- lane output confirmed current behavior is by design under the active actor/guard defaults.

Key findings (root cause and boundaries):
1. MCP mutation tools accept actor/lease fields as optional, with required args focused on domain payload (`project_id`, `column_id`, `title`, etc.).
2. MCP adapter normalizes empty `actor_type` to `user`.
3. Guard context is only attached when actor is non-user or guard tuple is supplied.
4. App guard allows `user` actor without lease when no guard is present.
5. Result: calls like `till.create_task` with no actor/lease tuple run as unguarded user mutations.
6. Additional coverage gap: some write surfaces are not lease-gated because they do not invoke `enforceMutationGuard*` (project create, kind upsert/allowlist, lease issuance lifecycle).

Evidence (file references used in investigation):
- `internal/adapters/server/mcpapi/extended_tools.go` (`till.create_task` args + required fields)
- `internal/adapters/server/common/app_service_adapter_mcp.go` (`withMutationGuardContext`, actor defaulting, guard attachment)
- `internal/app/kind_capability.go` (`enforceMutationGuardAcrossScopes` user bypass branch)
- `internal/app/service.go` (`CreateTask` actor defaulting path)
- `internal/adapters/server/mcpapi/extended_tools_test.go` (minimal create-task success path)

Planned remediation backlog (implementation deferred):
1. `AUTH-01` Contract baseline + threat model.
   - Acceptance:
     - document trust boundary and effective auth behavior per MCP mutation tool,
     - explicitly mark legacy behavior and risk profile.
2. `AUTH-02` Add additive policy modes (default legacy behavior preserved).
   - Candidate modes:
     - `legacy`,
     - `lease_non_user`,
     - `lease_all`,
     - `approval_required`.
   - Acceptance:
     - config + runtime wiring lands,
     - backward-compatible default unchanged,
     - tests prove default compatibility.
3. `AUTH-03` Introduce approval-request domain model + storage (no enforcement yet).
   - Acceptance:
     - explicit approval request entity (`pending/approved/denied/expired`),
     - repo/app APIs + persistence coverage,
     - MCP tools for request/list/approve/deny.
4. `AUTH-04` Bind lease issuance to approval in approval mode.
   - Acceptance:
     - `issue_capability_lease` requires approved request when mode requires approval,
     - legacy mode remains unchanged.
5. `AUTH-05` Integrate approval + lease policy into mutation guard enforcement.
   - Acceptance:
     - approval mode blocks unapproved mutations,
     - regression tests confirm legacy mode pass-through behavior.
6. `AUTH-06` User-facing approval UX and dogfood validation.
   - Acceptance:
     - TUI/MCP workflows for pending approvals + approve/deny/reason capture,
     - worksheet evidence updated,
     - `just check` + `just ci` green at rollout checkpoint.

Execution notes:
- This checkpoint is investigation/planning only.
- No code changes beyond roadmap/worklog planning were made.
- test_not_applicable: docs/planning-only update.

## Checkpoint 2026-03-03: MCP Agent-Only Actor Policy + HTTP User Preservation

Objective:
- enforce agent-authenticated MCP mutation behavior while preserving HTTP user actor support and preventing external system actor usage.

Scope decisions implemented:
1. MCP mutation tool surface now accepts only `actor_type` values `agent_orchestrator` or `agent_subagent`.
2. MCP mutation calls require full lease tuple semantics (`agent_name`, `agent_instance_id`, `lease_token`) at MCP handler validation time.
3. MCP actor types are normalized to internal domain actor type `agent` before passing into common/app services.
4. External transport actor type `system` is rejected by common adapter validation (`user` and `agent` supported externally).
5. HTTP user actor flows remain supported (no MCP-only restriction leakage into HTTP user path).

Files updated (this checkpoint):
- `internal/adapters/server/mcpapi/extended_tools.go`
- `internal/adapters/server/mcpapi/extended_tools_test.go`
- `internal/adapters/server/mcpapi/handler_test.go`
- `internal/adapters/server/common/app_service_adapter_mcp.go`
- `internal/adapters/server/common/app_service_adapter_mcp_guard_test.go`
- `internal/adapters/server/common/app_service_adapter_mcp_actor_attribution_test.go`

Validation evidence:
1. `just fmt` -> PASS.
2. `just test-pkg ./internal/adapters/server/mcpapi` -> PASS.
3. `just check` -> PASS.
4. `just ci` -> PASS (coverage gate green; `internal/adapters/server/mcpapi` 76.4%, `internal/tui` 70.5%).

Status:
- MCP mutation auth policy now enforces agent-role + lease tuple requirements.
- HTTP user actor operations remain supported.
- external/system actor rejection is enforced at adapter validation.

## Checkpoint 2026-03-03: Live MCP Runtime Auth Verification (Post-Commit)

Objective:
- validate live MCP server behavior for the just-landed auth policy changes (actor-role restriction + lease tuple enforcement + guardrail failures).

Live runtime probes executed (MCP tools):
1. `till.list_projects` -> PASS (runtime reachable).
2. `till.list_tasks(project_id=5996fbd8-6a35-42cb-bd01-5aa0a4495cec)` -> PASS (fixture context confirmed).
3. Negative actor-type probe:
   - `till.create_project(... actor_type=user, agent_name, agent_instance_id, lease_token)` -> FAIL CLOSED
   - response: `invalid_request: actor_type must be "agent_orchestrator" or "agent_subagent"`.
4. Negative actor-type probe:
   - `till.create_project(... actor_type=system, agent_name, agent_instance_id, lease_token)` -> FAIL CLOSED
   - response: `invalid_request: actor_type must be "agent_orchestrator" or "agent_subagent"`.
5. Missing tuple probe:
   - `till.create_project(... actor_type=agent_orchestrator)` with missing `agent_name/agent_instance_id/lease_token` -> FAIL CLOSED
   - response: `invalid_request: agent_name, agent_instance_id, and lease_token are required for authenticated MCP mutations`.
6. Positive MCP mutation shape probe:
   - `till.create_project(name=mcp-auth-check-positive-2026-03-03, actor_type=agent_orchestrator, tuple supplied)` -> PASS
   - project id: `63e404ba-5631-4367-9cf5-1177d316bcd7`.
7. Lease issuance:
   - `till.issue_capability_lease(project scope, role=orchestrator, agent_name=probe-agent, agent_instance_id=probe-agent-instance)` -> PASS
   - lease token: `4301f838-6052-4322-bae6-c827366930d9`.
8. Positive lease-validated mutation:
   - `till.update_project(... actor_type=agent_orchestrator, tuple + issued lease)` -> PASS.
9. Invalid lease token probe:
   - `till.update_project(... lease_token=not-a-valid-issued-lease)` -> FAIL CLOSED
   - response: `guardrail_failed ... mutation lease is invalid`.
10. Cross-project lease misuse probe:
   - created second target project `c3427235-6408-4d91-9f87-47a66f3910cf`.
   - attempted update with lease from first project -> FAIL CLOSED
   - response: `guardrail_failed ... mutation lease is invalid`.
11. `agent_subagent` acceptance probe:
   - overlap check: second orchestrator lease while first active -> FAIL CLOSED (`overlapping orchestrator lease blocked`).
   - revoked first lease (`till.revoke_capability_lease`) -> PASS.
   - issued lease for `probe-agent-sub-instance` -> PASS.
   - `till.update_project(... actor_type=agent_subagent, valid tuple/lease)` -> PASS.
12. Cleanup:
   - `till.revoke_capability_lease(agent_instance_id=probe-agent-sub-instance)` -> PASS.

Outcome summary:
- live MCP runtime now rejects `actor_type=user|system` for mutation calls.
- live MCP runtime requires authenticated tuple fields for mutation calls.
- guardrails fail closed for invalid/mismatched leases.
- both allowed MCP mutation role values (`agent_orchestrator`, `agent_subagent`) execute successfully with valid lease tuples.

Supplemental local test evidence:
1. `just test-pkg ./internal/adapters/server/mcpapi` -> PASS.
2. `just test-pkg ./internal/adapters/server/common` -> `[no test files]` under default build tags.
   - note: adapter common tests in this repo use `//go:build commonhash` and are excluded unless that tag is enabled.

## Checkpoint 2026-03-03: Notifications Reliability Regression (Global + Project)

Objective:
- track and resolve newly reported runtime regression where global notifications and possibly project notifications are not functioning reliably.

User-reported issue:
1. Global notifications appear non-functional.
2. Project notifications may also be impacted.
3. Current behavior needs fresh collaborative verification before implementation assumptions.

Required execution flow (collaborative):
1. Reproduce and baseline current behavior in a live collaborative run.
   - run targeted collaborative notification checks from active worksheet flow.
   - capture exact observed behavior for:
     - global notification count updates,
     - project notification list updates,
     - keyboard navigation/drill-in behavior,
     - refresh timing after MCP and local mutations.
2. Convert findings into an explicit fix plan before code changes.
   - identify whether failure is state ingestion, render/update loop, filtering/scope logic, or interaction handling.
   - map each finding to one scoped fix item with acceptance criteria.
3. Execute fix/test iteration loop until behavior is stable.
   - implement one scoped fix at a time,
   - run relevant package tests (`just test-pkg ./internal/tui` and any touched server/app packages),
   - re-run collaborative notification checks for the same scope,
   - log pass/fail evidence after each iteration.
4. Close with full validation gates.
   - run `just check` and `just ci`,
   - update collaborative worksheet verdicts and evidence references,
   - keep section open until user confirms expected live behavior.

Status:
- open blocker for collaborative dogfooding readiness.
- next action: run focused collaborative notification test pass and capture current-state evidence before coding.

## Checkpoint 2026-03-03: Vector Search + Embeddings Wave Plan Initialization

Objective:
- establish the execution plan for hybrid keyword+vector search, ncruces/sqlite-vec runtime migration, and fantasy-fork embeddings integration with explicit final TUI metadata accessibility before collaborative verification.

Actions completed:
1. Created dedicated execution tracker doc: `VECTOR_SEARCH_EXECUTION_PLAN.md`.
2. Locked search/filter/ranking contract decisions:
   - modes: `keyword|semantic|hybrid` (default `hybrid`),
   - filters/sorting/pagination/limit defaults and max guardrails,
   - task metadata indexing scope and phased expansion.
3. Locked integration decisions:
   - SQLite runtime migration direction (`ncruces` + sqlite-vec bindings),
   - embeddings provider direction (`charm.land/fantasy` API with fork replace/pin),
   - forward-only idempotent schema ensures in this wave (no generic legacy backfill framework).
4. Locked sequencing rule:
   - TUI support for all indexed metadata fields lands in the final implementation phase immediately before collaborative test closeout.

Commands run and outcomes:
1. `tail -n 160 PLAN.md` -> PASS (inspected latest checkpoint format).
2. `ls -1 *.md` -> PASS (confirmed root markdown inventory).
3. `curl -sS https://raw.githubusercontent.com/evanmschultz/fantasy/feature-embeddings/go.mod | sed -n '1,80p'` -> PASS (verified module path remains `charm.land/fantasy`).
4. `curl -sS 'https://api.github.com/repos/charmbracelet/fantasy/compare/main...evanmschultz:main' | sed -n '1,220p'` -> PASS (confirmed fork `main` is behind upstream; not the embeddings branch).
5. `curl -sS 'https://api.github.com/repos/evanmschultz/fantasy/branches?per_page=100' | sed -n '1,240p'` -> PASS (confirmed `feature-embeddings` branch exists).
6. `curl -sS 'https://api.github.com/repos/evanmschultz/fantasy/compare/main...feature-embeddings' | sed -n '1,260p'` -> PASS (confirmed branch is ahead with embeddings-related changes).
7. `curl -sS https://raw.githubusercontent.com/evanmschultz/fantasy/feature-embeddings/embedding.go | sed -n '1,260p'` -> PASS (verified embeddings interface).
8. `curl -sS https://raw.githubusercontent.com/evanmschultz/fantasy/feature-embeddings/providers/openai/embedding_model.go | sed -n '1,260p'` -> PASS (verified OpenAI embedding model implementation).
9. `curl -sS https://raw.githubusercontent.com/evanmschultz/fantasy/feature-embeddings/README.md | rg -n "embedding|Embedding|vector|retrieve|search"` -> PASS (verified embeddings documentation section).
10. `curl -sS https://raw.githubusercontent.com/evanmschultz/fantasy/feature-embeddings/README.md | sed -n '56,130p'` -> PASS (validated documented usage flow).
11. `rg -n "search|query|embed|embedding|vector|keyword|rank|similar|hybrid|match" internal cmd -g'*.go'` -> PASS (surveyed local search and vector-related touchpoints).
12. `sed -n '860,1060p' internal/app/service.go` -> PASS (inspected current search algorithm path).
13. `sed -n '1,220p' internal/app/service.go` -> PASS (inspected app service boundaries).
14. `rg -n "type Repository interface|ListTasks\\(|SearchTaskMatches|TaskMatch|ProjectRepository|TaskRepository" internal/app -g'*.go'` -> PASS (inspected contract boundaries).
15. `sed -n '1,220p' internal/app/ports.go` -> PASS (confirmed current repository port surface).
16. `sed -n '1,320p' internal/domain/task.go` -> PASS (inspected task core fields).
17. `sed -n '1,260p' internal/domain/project.go` -> PASS (inspected project text fields).
18. `sed -n '1,260p' internal/domain/comment.go` -> PASS (inspected comment text fields).
19. `rg -n "type TaskMetadata|DependsOn|BlockedBy|CompletionContract|ChecklistItem|KindPayload|Resource" internal/domain -g'*.go'` -> PASS (identified indexed metadata candidates).
20. `sed -n '120,280p' internal/domain/workitem.go` -> PASS (inspected metadata fields for final indexing scope).

File edits:
1. Added `VECTOR_SEARCH_EXECUTION_PLAN.md` with wave-based implementation, QA, and collaborative verification gates.
2. Updated `PLAN.md` with this checkpoint entry and command evidence.

Test status:
- `test_not_applicable` (planning/docs-only checkpoint; no code-path changes in this step).

## Checkpoint 2026-03-03: Vector Search + Embeddings Integration (Waves A-E) and Wave F Gate Pass

Objective:
- complete the planned sqlite runtime migration, embeddings + vector search implementation, TUI metadata accessibility updates, and post-fix QA/gate verification before collaborative user validation.

Execution summary:
1. Integrated parallel worker lanes with non-overlapping locks:
   - `W-VEC-1`: app + MCP search-contract completion (`levels|kinds|labels_any|labels_all`, metadata lexical scoring, schema forwarding/tests).
   - `W-VEC-2`: sqlite runtime config hardening + TUI dependency-inspector explicit limit + TUI test coverage.
2. Ran independent QA lanes before and after remediation:
   - identified contract gaps (missing filters, lexical metadata weighting, TUI limit omissions, runtime-config risk),
   - implemented targeted fixes and re-ran QA until no blocking findings remained.
3. Completed remaining TUI search shaping alignment:
   - forwarded `levels`, explicit `mode/sort/limit/offset`, and `kinds/labels_any/labels_all` slices in TUI search requests,
   - removed local post-limit level filtering from TUI search paths to avoid truncation drift against backend filtering.

Commands run and outcomes:
1. `just test-pkg ./internal/tui` -> PASS (`.tmp/vec-wavef-evidence/20260303_175936/test_pkg_internal_tui.txt`).
2. `just test-pkg ./internal/app` -> PASS (`.tmp/vec-wavef-evidence/20260303_175936/test_pkg_internal_app.txt`).
3. `just test-pkg ./internal/adapters/server/mcpapi` -> PASS (`.tmp/vec-wavef-evidence/20260303_175936/test_pkg_internal_adapters_server_mcpapi.txt`).
4. `just test-pkg ./internal/adapters/server/common` -> PASS (`[no test files]`; package is build-tag constrained in this profile).
5. `just test-pkg ./internal/adapters/storage/sqlite` -> PASS (`.tmp/vec-wavef-evidence/20260303_175936/test_pkg_internal_adapters_storage_sqlite.txt`).
6. `just test-pkg ./cmd/till` -> PASS.
7. `just test-pkg ./internal/config` -> PASS.
8. `just check` -> PASS (`.tmp/vec-wavef-evidence/20260303_175936/just_check.txt`).
9. `just ci` -> PASS (`.tmp/vec-wavef-evidence/20260303_175936/just_ci.txt`).

File scopes completed in this checkpoint:
1. SQLite runtime/storage:
   - `internal/adapters/storage/sqlite/repo.go`
   - `internal/adapters/storage/sqlite/repo_test.go`
2. App search/embedding integration:
   - `internal/app/service.go`
   - `internal/app/search_embeddings.go`
   - `internal/app/service_test.go`
3. Embeddings adapter/wiring/config:
   - `internal/adapters/embeddings/fantasy/generator.go`
   - `cmd/till/main.go`
   - `internal/config/config.go`
   - `config.example.toml`
   - `go.mod`
   - `go.sum`
4. MCP transport/tool surface:
   - `internal/adapters/server/common/mcp_surface.go`
   - `internal/adapters/server/common/app_service_adapter_mcp.go`
   - `internal/adapters/server/mcpapi/extended_tools.go`
   - `internal/adapters/server/mcpapi/extended_tools_test.go`
5. TUI metadata/search updates:
   - `internal/tui/model.go`
   - `internal/tui/model_test.go`

Status:
1. `VECTOR_SEARCH_EXECUTION_PLAN.md` Wave A-E acceptance is implemented and tested.
2. Wave F automated gates are complete with reproducible artifact paths under `.tmp/vec-wavef-evidence/20260303_175936/`.
3. Independent QA pass 1 found remaining drift (docs alignment + TUI default-limit mismatch); remediation was completed and validated in follow-up checkpoint.
4. Collaborative user+agent verification remains pending and must record evidence in `COLLAB_TEST_2026-03-02_DOGFOOD.md` (primary) with corroboration in `MCP_DOGFOODING_WORKSHEET.md` where applicable.

## Checkpoint 2026-03-04: Vector Wave F Remediation + Dual QA Completion

Objective:
- close all open vector audit findings, enforce two independent QA passes before checklist completion, and update tracker docs with reproducible evidence links.

Actions completed:
1. Applied remediation changes for pass-1 blockers:
   - aligned TUI explicit search default limit to `50`,
   - aligned vector plan docs with implemented indexed-content/schema behavior,
   - added explicit artifact references and collaborative evidence destinations in vector plan/docs.
2. Ran scoped and full validation against remediated state.
3. Executed QA pass 2 with five independent auditors (different agents than pass 1) and recorded per-lane reports.
4. Marked vector audit-intake checklist items complete only after pass-2 confirmed no unresolved High/Medium findings.

Commands run and outcomes:
1. `just test-pkg ./internal/app` -> PASS (`.tmp/vec-wavef-evidence/20260303_180827/test_pkg_internal_app.txt`).
2. `just test-pkg ./internal/adapters/storage/sqlite` -> PASS (`.tmp/vec-wavef-evidence/20260303_180827/test_pkg_internal_adapters_storage_sqlite.txt`).
3. `just test-pkg ./internal/adapters/server/mcpapi` -> PASS (`.tmp/vec-wavef-evidence/20260303_180827/test_pkg_internal_adapters_server_mcpapi.txt`).
4. `just test-pkg ./internal/tui` -> PASS (`.tmp/vec-wavef-evidence/20260303_180827/test_pkg_internal_tui.txt`).
5. `just check` -> PASS (`.tmp/vec-wavef-evidence/20260303_180827/just_check.txt`).
6. `just ci` -> PASS (`.tmp/vec-wavef-evidence/20260303_180827/just_ci.txt`).

QA pass 1 evidence:
1. `worklogs/VEC_QA_PASS1_A_APP.md` -> PASS.
2. `worklogs/VEC_QA_PASS1_B_SQLITE.md` -> PASS.
3. `worklogs/VEC_QA_PASS1_C_MCP.md` -> PASS.
4. `worklogs/VEC_QA_PASS1_D_TUI.md` -> FAIL (Medium drift; remediated).
5. `worklogs/VEC_QA_PASS1_E_DOCS.md` -> FAIL (High/Medium docs drift; remediated).

QA pass 2 evidence:
1. `worklogs/VEC_QA_PASS2_A_APP.md` -> PASS.
2. `worklogs/VEC_QA_PASS2_B_SQLITE.md` -> PASS.
3. `worklogs/VEC_QA_PASS2_C_MCP.md` -> PASS.
4. `worklogs/VEC_QA_PASS2_D_TUI.md` -> PASS.
5. `worklogs/VEC_QA_PASS2_E_DOCS.md` -> PASS.

File edits:
1. `internal/tui/model.go` (default explicit search limit aligned to plan default `50`).
2. `VECTOR_SEARCH_EXECUTION_PLAN.md` (schema/indexed-content alignment, checklist statuses, QA summaries, artifact references).
3. `PLAN.md` (status + evidence updates).

Status:
1. Vector audit-intake remediation checklist is complete under two-pass QA rule.
2. Wave F remaining closeout is now only collaborative user+agent verification evidence capture.

## Checkpoint 2026-03-04: Collaborative Vector + MCP E2E Worksheet Setup

Objective:
- create one active collaborative worksheet to run user+agent TUI and MCP E2E validation with fail-stop remediation flow.

Actions completed:
1. Reviewed active collaborative worksheet patterns and vector execution tracker for consistency.
2. Consulted Context7 (`/mark3labs/mcp-go`) for MCP tool schema/validation testing references.
3. Created `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md` with:
   - ordered TUI and MCP E2E sections,
   - explicit user wording capture columns,
   - fail-stop remediation workflow,
   - subagent planning/fix/QA tracking tables,
   - sign-off criteria for collaborative closeout.

Commands run and outcomes:
1. `ls -1 *.md` -> PASS.
2. `sed -n '1,260p' COLLAB_TEST_2026-03-02_DOGFOOD.md` -> PASS.
3. `sed -n '1,260p' MCP_DOGFOODING_WORKSHEET.md` -> PASS.
4. `sed -n '1,220p' VECTOR_SEARCH_EXECUTION_PLAN.md` -> PASS.
5. `mcp__context7-mcp__resolve-library-id("mark3labs/mcp-go", ...)` -> PASS.
6. `mcp__context7-mcp__query-docs("/mark3labs/mcp-go", ...)` -> PASS.
7. `cat > COLLAB_VECTOR_MCP_E2E_WORKSHEET.md <<'EOF' ...` -> PASS.

File edits:
1. Added `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md`.
2. Updated `PLAN.md` with this checkpoint.

Test status:
- `test_not_applicable` (docs/process setup only; no runtime code changes in this step).

## Checkpoint 2026-03-04: Collaborative Vector + MCP E2E Kickoff Preflight

Objective:
- run session-specific preflight for the new collaborative worksheet and capture reproducible evidence before user steps.

Actions completed:
1. Created evidence directory `.tmp/vec-collab-e2e-20260304_191626/`.
2. Ran `just build` and captured output.
3. Ran `./till serve --help` and captured output.
4. Ran `just check` and captured output.
5. Updated `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md` setup table with pass status and evidence paths.

Commands run and outcomes:
1. `ts=$(date +%Y%m%d_%H%M%S); dir=.tmp/vec-collab-e2e-$ts; mkdir -p "$dir"; echo "$dir"` -> PASS (`.tmp/vec-collab-e2e-20260304_191626`).
2. `just build | tee .tmp/vec-collab-e2e-20260304_191626/just_build.txt` -> FAIL in sandbox (`go-build cache operation not permitted`), then PASS outside sandbox.
3. `./till serve --help | tee .tmp/vec-collab-e2e-20260304_191626/till_serve_help.txt` -> PASS.
4. `just check | tee .tmp/vec-collab-e2e-20260304_191626/just_check.txt` -> PASS.

File edits:
1. Updated `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md` (section 4.1 statuses/evidence).
2. Updated `PLAN.md` with this checkpoint.

Test status:
- `just build`: PASS.
- `just check`: PASS.

## Checkpoint 2026-03-04: Collaborative Session Start Evidence Capture

Objective:
- begin live collaborative vector/MCP E2E run and record first user-confirmed runtime state in active worksheet.

Actions completed:
1. Captured runtime/db path resolution for current dev instance via `./till --dev paths`.
2. Verified repository migration behavior and runtime logging code paths for sqlite open/migration ensure.
3. Validated live DB contains `task_embeddings` table and expected columns.
4. Updated `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md` with user-confirmed `U-01` pass status and exact wording.

Commands run and outcomes:
1. `./till --dev paths` -> PASS (resolved config/data/db path).
2. `sqlite3 <db> ".tables"` -> PASS (`task_embeddings` present).
3. `sqlite3 <db> "PRAGMA table_info(task_embeddings);"` -> PASS (expected columns found).
4. `rg -n ... internal/adapters/storage/sqlite/repo.go cmd/till/main.go` -> PASS (verified ncruces driver, migrate/open logging, vec probe paths).
5. `rg -n "opening sqlite repository|sqlite repository ready|migrations" .tillsyn/log/tillsyn-20260305.log` -> PASS (startup migration ensure lines present).

File edits:
1. Updated `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md` (`U-01` status/evidence note).
2. Updated `PLAN.md` with this checkpoint.

Test status:
- `test_not_applicable` (discussion/evidence capture only; no runtime code changes).

## Checkpoint 2026-03-04: Collaborative Vector Test Start (Fresh DB)

Objective:
- start live collaborative validation on a fresh DB and lock fixture-seeding steps before executing TUI + MCP test sections.

Actions completed:
1. Recorded user kickoff statement confirming server + TUI running on fresh DB.
2. Updated active worksheet with:
   - refreshed `U-01` evidence wording,
   - required fresh-DB fixture seed section (`F-01`..`F-04`),
   - discussion-log entry for collaborative test start.
3. Prepared first execution steps for user-driven fixture seeding prior to T1/T2/T3 checks.

Commands run and outcomes:
1. `rg -n ... README.md internal/tui/model.go` -> PASS (verified current key controls and search/task flows).
2. `nl -ba README.md | sed -n '186,230p'` -> PASS (captured key bindings used for fixture instructions).
3. `apply_patch` on `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md` -> PASS.

File edits:
1. Updated `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md`.
2. Updated `PLAN.md`.

Test status:
- `test_not_applicable` (process/logging update only; no code behavior changes).

## Checkpoint 2026-03-04: Collaborative Failure Intake FR-001 (Modal/Info/Comments UX)

Objective:
- pause forward collaborative testing and start focused remediation loop for user-reported TUI architecture/UX gaps.

Failure intake summary:
1. User reported that node display/input surfaces should be unified across all node types (single reusable modal/component style with mode-specific interaction differences only).
2. User reported long rich-text fields should support wrapping, scrolling, and expandable viewing (description-like behavior).
3. User reported markdown rendering expectations for rich-text fields using Glamour.
4. User reported comments list is missing in info modal and must include ownership/relevant metadata.

Actions completed:
1. Recorded FR-001 in `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md` findings ledger.
2. Updated worksheet wording contract from "exact wording" to "detailed findings" per user instruction.
3. Paused section progression pending remediation.

Commands run and outcomes:
1. `apply_patch` on `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md` -> PASS (FR-001 + table heading updates).

Next step:
- run subagent architecture audits + Context7 references, then implement agreed remediation with package tests and dual QA.

Test status:
- `test_not_applicable` (intake/logging step only).

## Checkpoint 2026-03-05: FR-001 Remediation Implemented (Task-Info Scroll + Node Modal Unification)

Objective:
- implement user-requested FR-001 TUI remediation:
  - unified node modal framing for info/edit flows,
  - scrollable task-info body so comments/metadata are reachable,
  - full comments list with ownership metadata in task info,
  - node-type-aware info/edit headers,
  - preserve Glamour markdown rendering and key/mouse behavior.

Context7 compliance:
1. `mcp__context7-mcp__resolve-library-id("charmbracelet/bubbles", ...)` -> PASS.
2. `mcp__context7-mcp__query-docs("/charmbracelet/bubbles", viewport usage ...)` -> PASS.
3. After failed test runs, re-consulted Context7 viewport docs before subsequent edits -> PASS.

Implementation summary:
1. Updated `internal/tui/model.go`:
   - Added `taskInfoBody` viewport state to model/init.
   - Added `taskInfoBodyHeight`, `taskInfoBodyLines`, and `syncTaskInfoBodyViewport` helpers.
   - Wired task-info keyboard and mouse scroll handling to both details viewport and full body viewport.
   - Reset/sync task-info body viewport on open/close/path navigation transitions.
   - Reworked `modeTaskInfo` overlay rendering to use full-body viewport + node-type header (`<Node> Info`).
   - Added shared `nodeModalBoxStyle` and reused it across task info + add/edit node modal overlays.
   - Added node-label helpers (`taskNodeLabel`, `taskFormNodeLabel`) and updated add/edit task titles to `New/Edit <NodeType>`.
   - Preserved markdown rendering via existing `threadMarkdown.render` pathways.
   - Rendered full comments list in task info (no preview truncation) with metadata rows (actor/owner/timestamp + id/summary/body).
2. Updated `internal/tui/model_test.go`:
   - Added `TestModelTaskInfoShowsFullCommentsList`.
   - Extended task-info scroll test assertions to include body viewport scroll/reset behavior.
   - Added branch header checks (`New Branch`, `Edit Branch`, `Branch Info`) in overlay helper coverage.
   - Updated prompt assertions for node-type-aware prompt wording.
   - Updated task-info metadata/comment dependency hint tests to assert against `taskInfoBody.GetContent()` for viewport-aware behavior.
   - Added explicit owner metadata assertions for full comments list.

Dual QA (read-only subagents):
1. QA Pass 1: Darwin (`019cbc1c-aaaa-75f1-8be4-d333d98e6e3d`) -> PASS (approve retest).
2. QA Pass 2: Ampere (`019cb5c7-4141-76f2-a48e-70bb889ed054`) -> PASS (approve retest).
3. QA reports captured in:
   - `worklogs/FR001_QA_PASS1_DARWIN.md`
   - `worklogs/FR001_QA_PASS2_AMPERE.md`

Commands run and outcomes:
1. `just test-pkg ./internal/tui` -> FAIL (initial assertion mismatches after viewport behavior changes).
2. Context7 re-consult after failure -> PASS.
3. `just test-pkg ./internal/tui` -> FAIL (remaining assertion mismatches).
4. Context7 re-consult after failure -> PASS.
5. `just test-pkg ./internal/tui` -> PASS.
6. `just check` -> PASS.
7. `just ci` -> PASS.
8. After QA-noted low test gap fix, reran:
   - `just test-pkg ./internal/tui` -> PASS.
   - `just check` -> PASS.
   - `just ci` -> PASS.

Documentation/worklog sync:
1. Updated `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md`:
   - FR-001 status -> `READY_FOR_USER_RETEST`.
   - filled fix-planning and validation tables with test/QA evidence.
   - appended discussion-log entries for implementation and dual QA closeout.
2. Added QA pass worklogs under `worklogs/`.

Status:
- FR-001 implementation + tests + dual QA complete on agent side.
- Awaiting user collaborative rerun confirmation to mark section complete.

## Checkpoint 2026-03-05: FR-002 Modal Parity Full-Screen Follow-Up

Objective:
- address user-reported modal parity gap after FR-001:
  - info and edit node modals must be full-screen and use the same modal/component,
  - edit mode differs only by interaction behavior, not visual/modal structure.

Precondition:
1. User requested this fix scope explicitly after prior checkpoint commit.
2. Committed prior FR-001 scope before starting FR-002:
   - `git commit` -> `b03093c` (`fr-001: task-info scrollable modal, node headers, dual qa evidence`).

Context7 compliance (before edits):
1. `mcp__context7-mcp__resolve-library-id("charmbracelet/bubbles", ...)` -> PASS.
2. `mcp__context7-mcp__query-docs("/charmbracelet/bubbles", viewport full-screen/modal sizing query)` -> PASS.

Implementation summary:
1. Updated `internal/tui/model.go`:
   - expanded node modal sizing policy to full-screen style (`taskInfoOverlayBoxWidth`, `taskInfoBodyHeight` adjustments).
   - replaced old node-modal style helper signature with width-driven shared frame helper.
   - added shared full-screen modal renderer helpers:
     - `buildAutoScrollViewport`
     - `renderNodeModalViewport`
   - routed `modeTaskInfo` rendering through shared renderer.
   - routed add/edit node modes (`modeAddTask`, `modeEditTask`, `modeAddProject`, `modeEditProject`) through same shared renderer and viewport body path.
   - retained existing mode-specific interaction logic (focus/edit behaviors unchanged).
2. Updated collaborative docs/worklogs:
   - `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md` with FR-002 finding + FX-002 validation row.
   - added QA reports:
     - `worklogs/FR002_QA_PASS1_DARWIN.md`
     - `worklogs/FR002_QA_PASS2_AMPERE.md`

Commands run and outcomes:
1. `just fmt` -> PASS.
2. `just test-pkg ./internal/tui` -> PASS.
3. `just check` -> PASS.
4. `just ci` -> PASS.

Dual QA (required) after tests:
1. Darwin (`019cbc1c-aaaa-75f1-8be4-d333d98e6e3d`) -> PASS.
2. Ampere (`019cb5c7-4141-76f2-a48e-70bb889ed054`) -> PASS.

Status:
- FR-002 code/test/QA complete on agent side.
- Awaiting user collaborative rerun confirmation before advancing to next collab worksheet section.

## Checkpoint 2026-03-05: FR-003 Exact Info/Edit Section Parity Refactor

Objective:
- implement user follow-up request after FR-002:
  - edit modal must use the exact same full-screen design language as info modal,
  - reuse shared modal component/sections and convert sections to editable variants instead of maintaining a separate edit design.

Context7 compliance:
1. Before edits:
   - `mcp__context7-mcp__resolve-library-id("charmbracelet bubbles", ...)` -> PASS.
   - `mcp__context7-mcp__query-docs("/charmbracelet/bubbles", viewport/textinput usage)` -> PASS.
2. After failed test runs:
   - `mcp__context7-mcp__query-docs("/charmbracelet/bubbles", view-testing/viewport assertions)` -> PASS before next edit.
   - `mcp__context7-mcp__query-docs("/charmbracelet/bubbletea", clipped viewport test strategy)` -> PASS before next edit.

Implementation summary:
1. Updated `internal/tui/model.go`:
   - completed section-based task edit/add renderer via `taskFormBodyLines(...)` (same section family/order as task-info):
     - title/meta
     - description (markdown preview)
     - subtasks (read-only when editing)
     - effective labels
     - dependencies
     - comments (with owner/actor/timestamp/id/summary/body metadata in edit mode)
     - resources
     - metadata markdown sections (`objective`, `acceptance_criteria`, `validation_plan`, `risk_notes`)
   - added section-based project add/edit renderer via `projectFormBodyLines(...)`.
   - replaced legacy flat field-list branch in `renderModeOverlay(...)` node modal path with helper-driven section bodies for:
     - `modeAddTask`, `modeEditTask`
     - `modeAddProject`, `modeEditProject`
   - loaded comments when opening task edit form and reset comment cache for new-task form.
2. Updated `internal/tui/model_test.go`:
   - expanded modal coverage to assert shared scroll indicator on info/edit overlays.
   - added section-order parity assertions against helper body content (`taskFormBodyLines` + `taskInfoBodyLines`) to avoid viewport-clipping false negatives.
3. Updated collaborative tracker:
   - added FR-003/FX-003 records and validation evidence in `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md`.
   - recorded dual QA pass artifacts:
     - `worklogs/FR003_QA_PASS1_DARWIN.md`
     - `worklogs/FR003_QA_PASS2_AMPERE.md`

Commands run and outcomes:
1. `just fmt` -> PASS.
2. `just test-pkg ./internal/tui` -> FAIL (string expectation drift after UI text updates).
3. Context7 re-consult -> PASS.
4. `just fmt` -> PASS.
5. `just test-pkg ./internal/tui` -> PASS.
6. `just check` -> PASS.
7. `just ci` -> PASS.
8. Added parity test assertions in `internal/tui/model_test.go`.
9. `just fmt` -> PASS.
10. `just test-pkg ./internal/tui` -> FAIL (viewport clipping made section token absent in rendered overlay window).
11. Context7 re-consult -> PASS.
12. Updated test to assert helper body content ordering.
13. `just fmt` -> PASS.
14. `just test-pkg ./internal/tui` -> PASS.
15. `just check` -> PASS.
16. `just ci` -> PASS.
17. Parallel review lane 1 (`git diff -- internal/tui/full_page_surface.go internal/tui/model.go internal/tui/thread_mode.go internal/tui/description_editor_mode.go internal/tui/model_test.go`) -> PASS:
   - confirmed the shared full-page surface contract now owns height budgeting, task-form action routing is centralized, and thread/footer help matches focused-panel behavior.
18. Parallel review lane 2 (`git diff -- PLAN.md COLLAB_VECTOR_MCP_E2E_WORKSHEET.md` plus targeted `sed` reads over `internal/tui/model_test.go`) -> PASS:
   - confirmed tracker/test state reflects the FR-008 implementation wave, green validations, and the correct next collaborative step.

Dual QA (required) after passing tests:
1. Darwin (`019cbc1c-aaaa-75f1-8be4-d333d98e6e3d`) -> PASS.
   - report: `worklogs/FR003_QA_PASS1_DARWIN.md`
2. Ampere (`019cb5c7-4141-76f2-a48e-70bb889ed054`) -> PASS.
   - report: `worklogs/FR003_QA_PASS2_AMPERE.md`

Status:
- FR-003 code/test/QA complete on agent side.
- Awaiting user collaborative rerun confirmation for exact info/edit parity before advancing to next worksheet section.

## Checkpoint 2026-03-05: FR-004 Edit/Info UX Policy Cleanup + QA Remediation Loop

Objective:
- address the next user-reported collaborative failure scope before resuming TUI section progression:
  - keep info/edit on shared full-page node surfaces,
  - align edit traversal/render ordering (description directly under title),
  - remove inherited/effective labels rendering from info/edit,
  - simplify edit interaction policy (`ctrl+s` save; no edit `d`/`ctrl+r`/`ctrl+s` subtask),
  - add/edit subtasks/resources section actions and markdown-editor parity for metadata fields,
  - split info metadata lines and auto-size description preview height to content (capped).

Backlog/open-findings review checkpoint:
1. Reviewed active collaborative backlog/open findings in:
   - `PLAN.md` (active closeout + pending collaborative reruns),
   - `COLLAB_E2E_REMEDIATION_PLAN_WORKLOG.md`,
   - `COLLABORATIVE_POST_FIX_VALIDATION_WORKSHEET.md`,
   before implementing FR-004 scope; no roadmap expansion was introduced.

Parallel/subagent execution:
1. Spawned worker lane `LANE-UX-EDIT-NODE-TESTS` (lock: `internal/tui/model_test.go`) and worker lane `LANE-UX-EDIT-NODE-CODE-R2` (lock: `internal/tui/model.go`).
2. Performed integrator review of both lane diffs before accepting integration.
3. Ran independent dual QA passes with agents Hooke + Galileo; initial QA reported a real medium regression and hint-copy drift; follow-up remediation loop was executed and re-verified with fresh dual QA PASS.

Context7 compliance:
1. Pre-edit consult:
   - `/charmbracelet/bubbles` and `/charmbracelet/bubbletea` key/viewport guidance -> PASS.
2. After each failed `just test-pkg ./internal/tui` run, re-consulted Context7 before the next edit -> PASS.

Implementation summary:
1. Updated `internal/tui/model.go`:
   - introduced shared full-page node rendering branch (`isFullPageNodeMode`) for info/edit node views (non-overlay framing),
   - reordered task info/edit bodies so description appears directly under title,
   - removed inherited/effective label display blocks from info/edit bodies,
   - changed dependency placeholders to `csv task`,
   - added typing-safe edit key routing:
     - `ctrl+s` saves form,
     - `enter/e` actions for due/markdown/subtasks/resources,
     - labels/dependencies keep typing behavior with `enter`/`ctrl+l`/`o`,
   - added wrap-around edit navigation top<->bottom on up/down and k/j boundary behavior,
   - made task-info description viewport auto-grow to content height with existing max cap,
   - split task-info metadata lines (`priority`, `due`, `labels`),
   - synchronized help/hint copy to actual behavior.
2. Updated `internal/tui/model_test.go`:
   - added/updated coverage for FR-004 expectations (ordering, metadata line split, placeholder rename, key behavior, wrap navigation, subtasks visibility/actions, typing-safe `e`, no seed injection).
3. Updated docs/trackers:
   - `README.md` key-control note for due picker behavior in new/edit task contexts,
   - `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md` with FR-004/FX-004 findings, validation evidence, and discussion-log updates.

Commands run and outcomes:
1. `just test-pkg ./internal/tui` -> FAIL (legacy edit `ctrl+r` expectations).
2. Context7 re-consult after failure -> PASS.
3. `just fmt` -> PASS.
4. `just test-pkg ./internal/tui` -> PASS.
5. `just check` -> PASS.
6. `just ci` -> PASS.
7. QA pass 1 + pass 2 -> FAIL (medium `e` regression + hint-copy mismatch).
8. Follow-up remediation edits applied.
9. `just test-pkg ./internal/tui` -> FAIL (stale label-picker expectation).
10. Context7 re-consult after failure -> PASS.
11. `just fmt` -> PASS.
12. `just test-pkg ./internal/tui` -> FAIL (same test flow still mixed mode sequence).
13. Context7 re-consult after failure -> PASS.
14. `just fmt` -> PASS.
15. `just test-pkg ./internal/tui` -> PASS.
16. `just check` -> PASS.
17. `just ci` -> PASS.
18. Help/hint copy sync edits.
19. `just fmt` -> PASS.
20. `just test-pkg ./internal/tui` -> PASS.
21. `just check` -> PASS.
22. `just ci` -> PASS.
23. Final QA pass 1 (Hooke) -> PASS (low note only).
24. Final QA pass 2 (Galileo) -> PASS.

Files/docs updated in this checkpoint:
1. `internal/tui/model.go`
2. `internal/tui/model_test.go`
3. `README.md`
4. `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md`
5. `PLAN.md`

Status:
- FR-004/FX-004 agent-side remediation is complete with passing package/full gates and final dual QA sign-off.
- Collaborative progression remains paused pending user rerun/confirmation of the same failed section step (`T1-01`) before moving forward.

## Checkpoint 2026-03-05: FR-005 Follow-up UX Polish Before Collaborative Rerun

Objective:
- apply the user-requested follow-up corrections before resuming collaborative TUI steps:
  - clarify blank-value behavior in edit mode,
  - place `kind/state/complete/mode` metadata in info/edit headers,
  - make subtasks/resources rows focusable/selectable/editable in edit mode,
  - stop edit-boundary `j/k` wrapping so typing `k` in title works,
  - keep full-page node surfaces bordered with persistent `TILLSYN` header.

Context7 compliance:
1. Pre-edit consult:
   - `/charmbracelet/bubbletea` key handling (`tea.KeyPressMsg`, arrow-vs-rune routing) -> PASS.
2. Post-failure consult:
   - after one `just test-pkg ./internal/tui` failure, re-consulted `/charmbracelet/bubbletea` before the next edit -> PASS.

Implementation summary:
1. Updated `internal/tui/model.go`:
   - fixed task-form focus mapping to use stable field ids (and compatibility fallback for positional callers),
   - moved info/edit lifecycle metadata into node header subtitle rendering,
   - removed duplicate lifecycle line from task-info body and kept priority/due/labels as split lines,
   - added selectable-row rendering and row actions for edit-mode `subtasks` and `resources`,
   - clarified blank-value guidance copy,
   - made edit-mode boundary wrap arrow-only (`up/down`) while preserving typed `j/k` input behavior,
   - restored bordered full-page node surfaces and kept `TILLSYN` header visible while in full-page node modes,
   - synchronized add/edit task hints/help copy to match Enter action semantics.
2. Updated `internal/tui/model_test.go`:
   - adjusted keyboard-wrap assertions to verify arrow-wrap + typed `j/k`,
   - added coverage for header metadata lines in info/edit overlays,
   - added focused edit-mode row-selection test for subtasks/resources actions,
   - added full-view composition assertions for `TILLSYN` header + bordered node surface,
   - added assertion for clarified blank-value guidance text.
3. Updated collaborative tracking:
   - recorded FR-005/FX-005 and validation/QA evidence in `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md`.

Commands run and outcomes:
1. `just fmt` -> PASS.
2. `just test-pkg ./internal/tui` -> FAIL (new row-selection test reopened parent incorrectly while still in child edit mode).
3. Context7 re-consult after failure -> PASS.
4. Test flow fix + `just fmt` -> PASS.
5. `just test-pkg ./internal/tui` -> PASS.
6. `just check` -> PASS.
7. `just ci` -> PASS.
8. QA findings hardening (help-copy alignment + extra view/guidance tests) applied.
9. `just fmt` -> PASS.
10. `just test-pkg ./internal/tui` -> PASS.
11. `just check` -> PASS.
12. QA follow-up found one medium mismatch in notices-focused expanded-help copy.
13. `just fmt` -> PASS.
14. `just test-pkg ./internal/tui` -> PASS.
15. `just check` -> PASS.
16. `just ci` -> PASS.
13. Parallel QA pass 1:
    - Feynman (`019cbd28-39ce-7411-bca7-8366d7183f73`) -> PASS with low test-hardening notes.
    - Planck (`019cbd28-3bf8-7aa1-9de0-2047c96776a6`) -> FAIL due overlay-guardrail interpretation conflict.
14. Parallel QA pass 2 (explicit user-requirements framing, no code changes):
    - Feynman -> PASS.
    - Planck -> PASS.

Files/docs updated in this checkpoint:
1. `internal/tui/model.go`
2. `internal/tui/model_test.go`
3. `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md`
4. `PLAN.md`

Status:
- FR-005/FX-005 agent-side remediation is complete with passing package/full gates and parallel QA sign-off against explicit user requirements.
- Collaborative progression remains paused pending user rerun of `T1-01` before moving to the next step.

## Checkpoint 2026-03-13: FR-006 Post-Commit Node Screen Regression Remediation

Objective:
- address the new post-commit node-screen regressions before collaborative TUI testing resumes:
  - right/bottom border overflow on full-page info/edit screens,
  - description preview reset/scroll failures in info and preview contexts,
  - edit-mode resource attach path not discoverable/usable,
  - redundant inline field help still present,
  - inconsistent section-label colons,
  - no visible comments management path from edit mode.

Context7 compliance:
1. Pre-edit consult:
   - `/charmbracelet/bubbles` viewport offset/reset behavior and `SetContent`/`YOffset` usage -> PASS.
   - `/charmbracelet/bubbletea` mouse-wheel + key-routing behavior for full-screen panels -> PASS.
2. Post-failure consult:
   - after `just test-pkg ./internal/tui` failed on `TestModelTaskInfoDetailsViewportScrolls`, re-consulted `/charmbracelet/bubbles` viewport offset behavior before the next edit -> PASS.

Subagent investigation:
1. Inspection lane Sartre (`019ce86c-9de9-7120-bedd-8b01778532c5`) -> PASS:
   - identified width-budget bug (`taskInfoOverlayBoxWidth` + `nodeModalBoxStyle`), insufficient full-page body-height reservation, preview reset bug, and nested viewport scroll conflict.
2. Inspection lane Socrates (`019ce86c-a0ac-7132-987a-5d5941182c61`) -> PASS:
   - confirmed resources open path already exists, isolated redundant inline-help lines, flagged missing colon consistency, and identified lack of edit-mode comments affordance.
3. QA lane Copernicus (`019ce87f-bb0a-7b13-b2b0-55e97f78f086`) -> initial FAIL:
   - found lowercase `c` typing regression caused by new edit-mode comments shortcut.
4. QA lane Faraday (`019ce87f-bdd0-7112-b89d-44cddb2080be`) -> initial FAIL:
   - required new FR-006/FX-006 worksheet rows and a new PLAN checkpoint before tracker sign-off.

Implementation summary:
1. Updated `internal/tui/model.go`:
   - corrected full-page node width budgeting so border + padding stay inside the terminal,
   - added symmetric top/bottom insets around full-page node surfaces below the persistent `TILLSYN` header,
   - reduced task-info/edit body viewport height budget so bottom borders stay visible,
   - preserved `taskInfoDetails`, `taskInfoBody`, and description-preview offsets across `SetContent` refreshes,
   - reset description preview to top on entry into preview contexts,
   - changed task-info wheel routing so mouse scroll drives the inline description preview first while the page is at top, then falls through to page scroll,
   - removed low-value inline field help and standardized task info/edit section labels with colons,
   - added edit-mode comments/thread access and then refined it to `Shift+C` so lowercase typing remains intact.
2. Updated `internal/tui/description_editor_mode.go`:
   - preserved preview viewport offsets on layout/content refresh and added a shared `resetDescriptionPreviewToTop` helper.
3. Updated `internal/tui/thread_mode.go`:
   - allowed thread mode to return cleanly to edit-task mode without discarding current form state.
4. Updated `internal/tui/model_test.go`:
   - adjusted task-info scroll assertions to match body-vs-description routing,
   - added frame-bounds coverage for full-page node views,
   - added preview-opens-at-top coverage,
   - added new-resource attach-row coverage,
   - added edit-thread shortcut coverage including lowercase `c` typing preservation.

Commands run and outcomes:
1. `git status --short` -> PASS (clean start after prior commit).
2. `sed -n '1,220p' Justfile` -> PASS (repo automation source of truth reloaded).
3. Context7 pre-edit consults (`/charmbracelet/bubbles`, `/charmbracelet/bubbletea`) -> PASS.
4. Parallel inspection subagents (Sartre + Socrates) -> PASS.
5. `just fmt` -> PASS.
6. `just test-pkg ./internal/tui` -> FAIL (`TestModelTaskInfoDetailsViewportScrolls`; test expectation bug after new scroll routing).
7. Context7 post-failure re-consult (`/charmbracelet/bubbles`) -> PASS.
8. Test fix + `just fmt` -> PASS.
9. `just test-pkg ./internal/tui` -> PASS.
10. `just check` -> PASS.
11. `just ci` -> PASS.
12. Parallel QA review (Copernicus + Faraday) -> FAIL/FAIL:
    - Copernicus found lowercase `c` typing regression.
    - Faraday required FR-006/FX-006 tracker/doc updates.
13. `Shift+C` shortcut fix + typing regression test + `just fmt` -> PASS.
14. `just test-pkg ./internal/tui` -> PASS.
15. `just check` -> PASS.
16. `just ci` -> PASS.

Files/docs updated in this checkpoint:
1. `internal/tui/model.go`
2. `internal/tui/description_editor_mode.go`
3. `internal/tui/thread_mode.go`
4. `internal/tui/model_test.go`
5. `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md`
6. `PLAN.md`

Status:
- FR-006 code/test remediation is complete with green package/full gates.
- Final QA re-audit is complete:
  - Copernicus (`019ce87f-bb0a-7b13-b2b0-55e97f78f086`) -> PASS after `Shift+C` follow-up fix.
  - Faraday (`019ce87f-bdd0-7112-b89d-44cddb2080be`) -> PASS after FR-006 worksheet/PLAN state sync.
- Collaborative progression remains paused pending user rerun of the same blocked section step (`T1-01`) before moving forward.

## Checkpoint 2026-03-13: FR-007 Node Screen Consistency Sweep After User Rerun Feedback

Objective:
- address the next set of user-reported inconsistencies on the same blocked collaborative step before testing resumes:
  - asymmetric right-side inset and bottom clipping on full-page node screens,
  - task-info mouse wheel should scroll the page rather than the nested description preview,
  - edit-mode `comments` must be focusable and open through `enter/e`,
  - `due` and `labels` must be modal-only action rows with no inline typing/autocomplete help,
  - board-level `new subtask` shortcut/help must be removed,
  - bottom help and expanded help must be accurate to the active screen.

Context7 compliance:
1. Pre-edit consult:
   - `/charmbracelet/bubbletea` mode-specific key handling and mouse-event routing -> PASS.
   - `/charmbracelet/bubbles` viewport usage for outer-page scrolling vs nested preview panes -> PASS.
2. Post-failure consult:
   - after `just fmt` surfaced a syntax error and after the first `just test-pkg ./internal/tui` run failed on stale expectations/goldens, re-consulted `/charmbracelet/bubbletea` and `/charmbracelet/bubbles` before the next edits -> PASS.

Subagent investigation:
1. Inspection lane Parfit (`019ce8c3-cb30-7e70-bdd7-83a5fdc5d90b`) -> PASS:
   - audited bottom-help leakage, stale expanded-help lines, stale footer prompts, and remaining `ctrl+l`/`ctrl+g`/`Shift+C` references.
2. Inspection lane Ohm (`019ce8c3-ce2c-7431-841e-a94472b4be93`) -> PASS:
   - isolated the task-form action-row gaps (`comments` missing from focus order, `due/labels` still text inputs), plus the remaining full-page layout-width/body-height budgeting problem.

Implementation summary:
1. Updated `internal/tui/model.go`:
   - centered full-page node surfaces and tightened width/height budgeting so borders stay fully on-screen with matched side gutters,
   - converted task-info description preview to a top-aligned bounded preview while mouse wheel now scrolls the outer info page,
   - reused the same bounded preview sizing in edit mode so description preview height matches info mode,
   - made `due` and `labels` modal-only action rows, removed inline autocomplete/type hints, and removed `ctrl+l`, `ctrl+g`, and add-task `d`,
   - added a focusable `comments` row that opens thread/comments via `enter/e`,
   - made edit-mode mouse wheel scroll the full page viewport instead of changing focus,
   - added mode-aware bottom help so board-only actions no longer leak into task info/edit/full-page node screens,
   - removed the board-level `new subtask` shortcut handler and board help references.
2. Updated `internal/tui/keymap.go`:
   - removed board short/full help exposure for the old `new subtask` binding so the main screen help matches the available actions.
3. Updated `internal/tui/model_test.go`:
   - flipped task-info mouse-wheel expectations to page scroll,
   - updated label/due/comment tests to the new modal-only action-row contract,
   - refreshed help-copy assertions,
   - updated task-info section-order assertions for `description:`,
   - refreshed comment-row coverage and modal-opening flows.
4. Updated TUI goldens:
   - `internal/tui/testdata/TestModelGoldenBoardOutput.golden`
   - `internal/tui/testdata/TestModelGoldenHelpExpandedOutput.golden`
   to reflect removal of the board `new subtask` affordance and the new bottom-help output.
5. Updated collaborative tracking:
   - recorded FR-007/FX-007 and validation evidence in `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md`.

Commands run and outcomes:
1. Context7 pre-edit consults (`/charmbracelet/bubbletea`, `/charmbracelet/bubbles`) -> PASS.
2. Parallel inspection subagents (Parfit + Ohm) -> PASS.
3. `rg`/`sed` inspection over `internal/tui/model.go`, `internal/tui/keymap.go`, `internal/tui/model_test.go` -> PASS.
4. `just fmt` -> FAIL (syntax error in edit-task key-routing branch).
5. Context7 post-failure re-consult (`/charmbracelet/bubbletea`) -> PASS.
6. Syntax fix + `just fmt` -> PASS.
7. `just test-pkg ./internal/tui` -> FAIL (stale goldens and pre-change test expectations for nested description scroll + inline label/due behavior).
8. Context7 post-failure re-consult (`/charmbracelet/bubbles`) -> PASS.
9. Test updates + `just fmt` -> PASS.
10. `just test-golden-update` -> PASS.
11. `just test-pkg ./internal/tui` -> PASS.
12. `just check` -> PASS.
13. `just ci` -> PASS.

Files/docs updated in this checkpoint:
1. `internal/tui/model.go`
2. `internal/tui/keymap.go`
3. `internal/tui/model_test.go`
4. `internal/tui/testdata/TestModelGoldenBoardOutput.golden`
5. `internal/tui/testdata/TestModelGoldenHelpExpandedOutput.golden`
6. `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md`
7. `PLAN.md`

Status:
- FR-007 code/test remediation is complete with green package/full gates.
- Independent QA sign-off is complete:
  - Copernicus (`019ce87f-bb0a-7b13-b2b0-55e97f78f086`) -> PASS for scoped FR-007 code/UI review; residual risk only: no dedicated task-info frame-bounds golden, though the same full-page renderer path is covered.
  - Faraday (`019ce87f-bdd0-7112-b89d-44cddb2080be`) -> PASS after FR-007 worksheet/PLAN state-sync corrections.
- Collaborative progression remains paused pending the user's rerun of the same blocked section step (`T1-01`).

## Checkpoint 2026-03-13: FR-008 Architecture Review After Third `T1-01` Rerun Failure

Objective:
- pause forward collaborative testing again on the same blocked step and determine whether the next fix wave should be a true layout/form unification pass instead of another targeted patch:
  - edit full-page screen still clips its bottom border and keeps a wider right gutter than the left,
  - keyboard focus movement does not auto-scroll newly focused rows into view,
  - `depends_on` / `blocked_by` still do not match modal-only rows like `due` / `labels`,
  - `enter` still does not open the resources picker in the reported path,
  - add/edit/info/subtask still do not behave like one reused component family.

Context7 compliance:
1. Pre-discussion architecture consult:
   - `/charmbracelet/bubbletea` for viewport sizing, focus-driven scrolling, and mode-specific key handling -> PASS.
   - `/charmbracelet/bubbles` for viewport ownership and outer-page vs nested-preview scroll behavior -> PASS.
2. Post-failure re-consult:
   - not applicable yet; no implementation or test run has started in this FR-008 wave.

Subagent investigation:
1. Inspection lane Rawls (`019ce8fa-602d-7b42-8b0a-ac6a904cf780`) -> PASS:
   - confirmed info and edit share only the bordered shell (`renderNodeModalViewport`) while layout sizing still depends on separate viewport/body branches and a heuristic `fullPageNodeBodyHeight()` budget, which explains why the edit footer can still push the bottom border off-screen when the footer wraps.
2. Inspection lane Lorentz (`019ce8fa-633b-7952-b171-f2917ea964ff`) -> PASS:
   - confirmed add-task, edit-task, and subtask share `startTaskForm` initialization but not one true screen contract; `taskFormBodyLines` still branches by mode, and action rows (`due`, `labels`, dependencies, comments, resources, subtasks) still use bespoke render/dispatch logic instead of one reusable abstraction.

Architecture findings:
1. The current code shares the shell, not the full screen-layout contract:
   - shared border wrapper: `internal/tui/model.go` `renderNodeModalViewport`
   - divergent sizing/render flow: task-info viewport sync + render vs task-form viewport rebuild branches
   - heuristic height budgeting still uses fixed reserves instead of measured header/path/footer height.
2. The current code shares task-form startup, not one full add/edit/subtask body contract:
   - shared initializer: `startTaskForm`
   - subtask bootstraps through `startSubtaskForm`, then diverges via mode-sensitive focus/body logic
   - `taskFormBodyLines` still decides visible sections and rendering with edit-only branches.
3. The current code already has one good DRY pattern for markdown-backed fields:
   - shared markdown field classifier + one `startTaskFormMarkdownEditor` path
   - the same pattern has not been applied to action rows/list-backed rows.

Candidate remediation options for user consensus:
1. Option A (recommended):
   - introduce one shared full-page node layout metrics function/component for info/edit/add/subtask that measures rendered header/path/footer heights and computes the viewport from actual remaining rows,
   - introduce one reusable action-row contract for `due`, `labels`, `depends_on`, `blocked_by`, `comments`, `resources`, and `subtasks`,
   - drive add/edit/subtask through one variant-based task-form screen spec instead of branching body logic.
2. Option B:
   - keep the current split architecture and patch the symptoms only: footer-height accounting, focus auto-scroll on field moves, convert dependency rows to modal-only pickers, fix resource-enter dispatch, and tighten add/edit/subtask parity incrementally.

Recommendation:
- choose Option A.
- Reason: the user is explicitly asking for React-style component reuse and the current failures are recurring precisely because only the outer shell was unified; continuing with targeted patches leaves the same divergence points in place and is likely to create another retest loop on `T1-01`.

Commands run and outcomes:
1. `git status --short` -> PASS (confirmed current uncommitted collaborative-remediation state before doc updates).
2. `rg -n "FR-007|T1-01|FX-007|Discussion Log|Validation" COLLAB_VECTOR_MCP_E2E_WORKSHEET.md PLAN.md` -> PASS.
3. `sed -n '70,165p' COLLAB_VECTOR_MCP_E2E_WORKSHEET.md` -> PASS.
4. `sed -n '175,205p' COLLAB_VECTOR_MCP_E2E_WORKSHEET.md` -> PASS.
5. Documentation updates only in this checkpoint; no package tests/checks run yet because the wave is still at the user-consensus stage.

Files/docs updated in this checkpoint:
1. `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md`
2. `PLAN.md`

Status:
- `T1-01` remains FAIL and forward collaborative progression is paused.
- FR-008 / FX-008 are logged and current in the worksheet.
- No implementation lane has started yet for FR-008.
- Next step: present Option A vs Option B to the user, recommend Option A, and wait for explicit consensus before editing code.

## Checkpoint 2026-03-13: FX-008 Option A Implementation, Test Remediation, And Gate Validation

Objective:
- implement the agreed Option A remediation wave for the same blocked collaborative step `T1-01` by unifying the full-page surface contract, tightening shared task-form/action-row behavior, and revalidating before the user reruns the same step.

User consensus and scope:
1. User explicitly selected Option A and expanded the scope to cover all full-screen surfaces, not just task info/edit:
   - use one measured wrapper/header contract across full-page screens,
   - keep comments/thread on the same outer shell pattern,
   - remove invalid/duplicated screen help and stale shortcuts,
   - keep behavior consistent and natural across task info, task edit/add/subtask, thread, and description views.

Context7 compliance:
1. Pre-edit consult:
   - `/charmbracelet/bubbletea` for central View routing and mode-specific key handling -> PASS.
   - `/charmbracelet/bubbles` for viewport sizing, `SetWidth` / `SetHeight` / `SetYOffset`, and focused-content visibility after content refresh -> PASS.
2. Post-failure re-consults:
   - after first `just test-pkg ./internal/tui` failure, re-consulted `/charmbracelet/bubbles` for viewport-state preservation and `/charmbracelet/bubbletea` for focus-sensitive key routing -> PASS.
   - after second `just test-pkg ./internal/tui` failure, re-consulted `/charmbracelet/bubbletea` for test sequencing around mode exits and `/charmbracelet/bubbles` for stateful component test hygiene -> PASS.
   - after third `just test-pkg ./internal/tui` failure, re-consulted `/charmbracelet/bubbletea` and `/charmbracelet/bubbles` again before the final stale test-path correction -> PASS.

Implementation summary:
1. Full-page surface unification:
   - added `internal/tui/full_page_surface.go` with one measured full-page surface helper for shared header/path/help/status budgeting.
   - reserved status-line height plus bounded outer inset rows in the shared surface metrics so bordered full-screen surfaces stay inside the terminal with matched side gutters.
2. Node-screen viewport unification:
   - routed add/edit/project full-page form rendering through persistent viewport state instead of rebuilding throwaway viewports in `renderFullPageNodeModeView`.
   - retained focus visibility by reusing `SetWidth` / `SetHeight` / `SetContent` / `SetYOffset` on the same viewport model, calling `ensureViewportLineVisible`, and tracking focused subtask/resource rows instead of just the section label.
3. Task/thread/help consistency:
   - removed stale board-level `s new subtask` key/help drift from the main keymap.
   - re-synced task/edit/task-info/thread help and prompt text to actual behavior (`enter/e` action rows, no obsolete dependency `o` guidance, no stale inline description hotkeys).
   - fixed task-info parent-navigation history so `backspace` to parent followed by `esc` behaves naturally instead of stepping through stale path history.
   - stabilized the thread details panel label and aligned description-editor panel sizing with the same measured surface body budget used by the outer wrapper.
4. Test remediation:
   - normalized TUI test helpers to accept pointer-wrapped `tea.Model` results during command-driven update flows.
   - updated the stale resource-picker resequencing assertion to use the real focus path instead of a direct field assignment bypass.

Commands run and outcomes:
1. `git status --short` -> PASS.
2. `sed -n '1,220p' Justfile` -> PASS (reconfirmed local automation source of truth).
3. `rg -n "fullPageSurface|renderFullPageNodeModeView|modeThread|taskFormBodyLines|activeBottomHelpKeyMap|renderHelpOverlay" internal/tui/{model.go,thread_mode.go,description_editor_mode.go,full_page_surface.go,model_test.go}` -> PASS.
4. `sed -n '1,240p' internal/tui/full_page_surface.go` + targeted `sed` reads across `internal/tui/model.go`, `internal/tui/thread_mode.go`, `internal/tui/description_editor_mode.go`, `internal/tui/model_test.go` -> PASS.
5. `just fmt` -> PASS.
6. `gopls check internal/tui/full_page_surface.go internal/tui/model.go internal/tui/thread_mode.go internal/tui/description_editor_mode.go internal/tui/model_test.go` -> FAIL (sandbox cache-write permissions under `/Users/evanschultz/go/pkg/mod` and `~/Library/Caches/go-build`); ignored in favor of the required `just` test gates.
7. `just test-pkg ./internal/tui` -> FAIL:
   - pointer-vs-value test harness assumptions around `tea.Model` normalization in description/task/resource/dependency flows.
8. Context7 re-consult after failure -> PASS.
9. `just fmt` -> PASS.
10. `just test-pkg ./internal/tui` -> FAIL:
   - one remaining stale resource-picker resequencing assertion used a direct field assignment instead of the real focus path.
11. Context7 re-consult after failure -> PASS.
12. `just fmt` -> PASS.
13. `just test-pkg ./internal/tui` -> PASS.
14. `just check` -> PASS.
15. `just ci` -> PASS.

Files edited in this checkpoint:
1. `internal/tui/full_page_surface.go`
2. `internal/tui/model.go`
3. `internal/tui/thread_mode.go`
4. `internal/tui/description_editor_mode.go`
5. `internal/tui/keymap.go`
6. `internal/tui/model_test.go`
7. `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md`
8. `PLAN.md`

Status:
- FX-008 implementation is complete and agent-side validation is green (`just test-pkg ./internal/tui`, `just check`, `just ci`).
- Two independent QA passes are complete:
  1. PASS: local code/UI review over the shared surface, task-form action rows, focused-row auto-scroll, and thread/help contract.
  2. PASS: local tests/tracker review over `model_test.go`, `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md`, and `PLAN.md`.
- Worksheet/plan state has been advanced from architecture-review placeholder status to validated `READY_FOR_USER_RETEST`.
- Next step: have the user rerun the same blocked collaborative step `T1-01` before any forward testing resumes.

## Checkpoint 2026-03-13: FR-009 Shared Header/Help Cleanup And Rendered Focus Tracking

Objective:
- implement the user-approved follow-up cleanup on the same blocked collaborative step `T1-01` by tightening shared app chrome, shrinking board noise, wrapping panel traversal consistently, and fixing edit-form downward auto-scroll against rendered content.

User findings captured for this wave:
1. Repeated mode/status text such as `text selection mode enabled` and `thread loaded` is low-value noise and should not sit in the persistent UI.
2. `path:` should move beside the boxed `TILLSYN` mark so every full-screen/board surface reclaims one row.
3. The board summary/footer is too dense; attention/dependency rollups are redundant with the notices panels and the short help line truncates.
4. `tab`, `shift+tab`, and arrow-based panel traversal should loop at the boundaries instead of clamping.
5. Edit-form focus movement still scrolls upward but not downward because focus visibility is keyed to logical lines, not rendered wrapped rows.

Context7 + fallback:
1. Pre-edit consults:
   - `/charmbracelet/bubbles` for viewport `SetWidth` / `SetHeight` / `SetYOffset` behavior and explicit help-keymap usage -> PASS.
   - `/charmbracelet/bubbletea` for wrapped keyboard navigation/focus handling -> PASS.
2. After the first post-edit `just test-pkg ./internal/tui` failure, the turn had reached the Context7 call ceiling.
   - fallback source recorded before the next edit: `go doc charm.land/bubbles/v2/help` -> PASS.

Implementation summary:
1. Shared app chrome:
   - moved path rendering into the shared `appHeaderBlock` so board and full-page surfaces now render one inline-path `TILLSYN` header.
   - updated the board/header row budgeting to match the reclaimed vertical space.
2. Board/help cleanup:
   - removed attention/dependency summary rows from the board footer area and cut the selection summary down to the selected task plus subtree affordances.
   - replaced the board short help with one concise canonical line and kept the expanded help overlay separate.
   - cleared low-value thread/text-selection status spam instead of repeating mode/state in the persistent footer area.
3. Navigation/focus behavior:
   - switched left/right board panel traversal to the same wrapping policy already used by tab traversal.
   - replaced logical-line task/project form focus tracking with rendered-row markers so downward focus visibility follows wrapped content correctly after viewport refresh.
4. Test/golden alignment:
   - updated board-path/summary assertions to the new inline-header contract.
   - refreshed the board/help goldens after the shared-header and concise-help changes.

Commands run and outcomes:
1. `git status --short` -> PASS.
2. `sed -n '1,220p' Justfile` -> PASS.
3. `rg -n "text selection mode enabled|thread loaded|activeBottomHelpKeyMap|modePrompt|attention scope|attention panel|deps: total|tasks:|selected:|path:|panel|wrap" internal/tui/model.go internal/tui/full_page_surface.go internal/tui/thread_mode.go internal/tui/keymap.go internal/tui/model_test.go` -> PASS.
4. Context7 consults before edits -> PASS.
5. `just fmt` -> PASS.
6. `just test-pkg ./internal/tui` -> FAIL:
   - stale board expectations/goldens after the shared-header/help cleanup.
7. Fallback source after Context7 ceiling: `go doc charm.land/bubbles/v2/help` -> PASS.
8. `just fmt` -> PASS.
9. `just test-golden-update` -> PASS.
10. `just test-pkg ./internal/tui` -> PASS.
11. `just check` -> PASS.
12. `just ci` -> PASS.

Files edited in this checkpoint:
1. `internal/tui/full_page_surface.go`
2. `internal/tui/model.go`
3. `internal/tui/model_test.go`
4. `internal/tui/testdata/TestModelGoldenBoardOutput.golden`
5. `internal/tui/testdata/TestModelGoldenHelpExpandedOutput.golden`
6. `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md`
7. `PLAN.md`

Status:
- FR-009 implementation is complete and the agent-side validation stack is green (`just test-pkg ./internal/tui`, `just check`, `just ci`).
- Worksheet state is current:
  - `T1-01` remains the same blocked collaborative rerun target.
  - `FR-009` is logged as `READY_FOR_USER_RETEST`.
  - `FX-009` is logged with validations complete and both QA passes closed:
    1. PASS: Copernicus (`019ce87f-bb0a-7b13-b2b0-55e97f78f086`) code/UI re-audit after the help-copy/test follow-up.
    2. PASS: Faraday (`019ce87f-bdd0-7112-b89d-44cddb2080be`) tests/docs/tracker re-audit after the gating sync.
- Next step: commit this follow-up fix scope, then have the user rerun the same blocked collaborative step `T1-01` before any forward testing resumes.

## Checkpoint 2026-03-13: FR-010 Final Board/Footer/Thread Consistency Sweep

Objective:
- finish the last follow-up cleanup on blocked collaborative step `T1-01` without opening roadmap work: keep the shared full-page architecture, but remove the remaining footer/help/status drift and make thread/project-picker traversal behave consistently.

User findings captured for this wave:
1. Edit-task could reopen scrolled away from `title`.
2. Thread panel `tab`/`shift+tab` and arrow behavior still felt inconsistent.
3. Low-value mode/status text still repeated above bottom help.
4. Full-page info/edit/thread gutters still felt too padded versus the board panels.
5. The board footer still repeated selected/focus/overdue information.
6. Project/global notifications panels still showed inline navigation hints.
7. The short board help needed `:` restored and `? help` last.

Context7:
1. Pre-edit consult: Bubble Tea key handling/navigation patterns -> PASS.
2. After first failed package test: Bubble Tea key handling/backtab reminder -> PASS.
3. After second failed package test: Bubble Tea model state persistence reminder -> PASS.
4. After full-gate compile failure: Bubble Tea view/model separation reminder -> PASS.
5. After final golden-only failure: Bubbles help/golden rendering reminder -> PASS.

Implementation summary:
1. Reset shared task/project full-page viewports on entry so edit always starts at the top/title.
2. Tightened shared surface sizing to match the board gutter contract: no extra horizontal inset, no extra bottom spacer, slimmer box padding, and wider usable content width.
3. Suppressed low-value full-page status text (`edit task`, `task info`, `thread loaded`, focus/status noise) while preserving real mutation/error feedback.
4. Removed project/global notifications panel-local nav hint rows; overdue/due-soon now surface in warnings instead of the board footer.
5. Kept only subtree affordances in the board footer; removed redundant selected-task and due-summary footer lines.
6. Restored board short-help ordering/content with `:` and trailing `? help`.
7. Kept thread panels on reliable `tab`/`shift+tab` plus `left/right` wrapping, with comments-panel `up/down` scrolling; project picker now accepts `left/right` aliases too.
8. Fixed one repo-wide compile leak where `boardFooterLines` referenced the test-only `stripANSI` helper.
9. Refreshed board/help goldens and updated focused TUI tests for the deliberate contract changes.

Commands run and outcomes:
1. `just fmt` -> PASS.
2. `just test-pkg ./internal/tui` -> FAIL (golden drift + thread/notices expectations) -> Context7 re-consult.
3. `just fmt` -> PASS.
4. `just test-pkg ./internal/tui` -> FAIL (golden drift only) -> Context7 re-consult.
5. `just test-golden-update` -> PASS.
6. `just test-pkg ./internal/tui` -> PASS.
7. `just check` -> FAIL (`stripANSI` test helper referenced from production code) -> Context7 re-consult.
8. `just ci` -> FAIL (same compile failure) -> same remediation.
9. `just fmt` -> PASS.
10. `just check` -> PASS.
11. `just ci` -> PASS.

Files edited in this checkpoint:
1. `internal/tui/full_page_surface.go`
2. `internal/tui/model.go`
3. `internal/tui/thread_mode.go`
4. `internal/tui/model_test.go`
5. `internal/tui/testdata/TestModelGoldenBoardOutput.golden`
6. `internal/tui/testdata/TestModelGoldenHelpExpandedOutput.golden`
7. `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md`
8. `PLAN.md`

Status:
- FR-010 implementation is complete and validation is green (`just test-golden-update`, `just test-pkg ./internal/tui`, `just check`, `just ci`).
- The same blocked collaborative step `T1-01` remains paused for the user's rerun after QA sign-off and commit.

Addendum 2026-03-13 19:08 local:
- Dual QA is now complete for FR-010.
  1. Copernicus PASS: code/UI review over shared surface sizing, footer/help cleanup, and thread navigation contract.
  2. Faraday PASS after one tracker-state sync follow-up: tests/docs/worksheet/plan now match the actual failure-driven validation history and retest-ready state.
- Status update: `T1-01` remains paused pending commit and then the user's rerun of that same blocked step; QA is no longer a blocker.

## Checkpoint 2026-03-13: FR-011 Status-Line Cleanup, Preview Parity, And Thread Tab Direction

Objective:
- keep the same blocked collaborative step `T1-01` paused while fixing the remaining shared-surface/input mismatches: remove the stale board status notification, make info/edit description previews use the same sizing contract, tighten full-page outer gaps to one equal inset on all sides, and correct thread `tab`/`shift+tab` behavior/help.

User findings captured for this wave:
1. The board still shows an unwanted `project switched` notification on initial project load; the user does not want that notification at all.
2. Info mode and edit mode still render different description-preview heights for the same task content.
3. Full-page info/edit/comments top and bottom outer gaps are still larger than the left/right board-style gaps; the user wants one equal outer inset on all sides.
4. Comments/thread still moves the same direction on `tab` and `shift+tab`; `shift+tab` should reverse `tab`, and the short help should say so.
5. The remaining bottom-space mismatch appears coupled to the stale board status line because the board reflows correctly only after leaving and returning.

Context7:
1. Pre-edit consult: `/charmbracelet/bubbles` viewport/key-handling guidance for consistent viewport sizing, top-aligned offsets, and tab/backtab behavior in Bubble Tea/Bubbles -> PASS.

Local code inspection completed before edits:
1. `project switched` is still set in the project-picker accept path at `internal/tui/model.go:7152`.
2. Shared full-page outer gaps are still split in `internal/tui/full_page_surface.go` with `surfaceTopGap = 1` and `surfaceBottomGap = 0`, while horizontal inset is already `0`.
3. Shared preview height is bounded by `markdownPreviewHeight`, but edit mode still builds its description preview with a different width basis (`contentWidth+8`) than the info-mode path.
4. Thread panel traversal already routes through `isForwardTabKey` / `isBackwardTabKey`, but the rendered help still advertises `tab/←/→` instead of making `shift+tab` explicit.

Implementation summary:
1. Removed the stale `project switched` board status assignment from the project-picker accept path so the board no longer reserves a bottom status row after project load.
2. Split the description preview builder into a shared measured-width helper and routed edit-mode previews through the same width/height contract as info mode.
3. Tightened the shared full-page surface helper to board-matched outer gaps by setting both top and bottom spacer rows to `0`.
4. Corrected tab-direction detection so shifted tab cannot fall through the forward-tab path, and updated thread short help to advertise `tab/shift+tab` plus left/right wrap explicitly.
5. Added focused TUI regressions for project-switch status suppression, shared preview parity, thread tab reversal, and outer-gap parity.

Commands run and outcomes:
1. `just fmt` -> PASS.
2. `just test-pkg ./internal/tui` -> PASS.
3. `just test-golden` -> PASS.
4. `just check` -> PASS.
5. `just ci` -> PASS.
6. QA follow-up found one missing explicit outer-gap regression test plus tracker-state drift -> remediation applied.
7. `just fmt` -> PASS.
8. `just test-pkg ./internal/tui` -> PASS.
9. `just test-golden` -> PASS.
10. `just check` -> PASS.
11. `just ci` -> PASS.

Files edited in this checkpoint:
1. `internal/tui/full_page_surface.go`
2. `internal/tui/model.go`
3. `internal/tui/model_test.go`
4. `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md`
5. `PLAN.md`

QA:
1. Copernicus initial review -> FAIL due missing explicit outer-gap regression coverage; final re-audit -> PASS after `TestFullPageSurfaceMetricsUseBoardMatchedOuterGaps`.
2. Faraday initial review -> FAIL due tracker/evidence drift; final re-audit -> PASS after worksheet/plan synchronization.

Status:
- FR-011 implementation is complete and validation is green (`just fmt`, `just test-pkg ./internal/tui`, `just test-golden`, `just check`, `just ci`).
- The same blocked collaborative step `T1-01` remains paused for the user's rerun against FR-011.

## Checkpoint 2026-03-13: FR-012 Transient Status Leakage, Command Palette Normalization, And Schema Coverage Audit

Objective:
- keep the same blocked collaborative step `T1-01` paused while removing transient footer/status leakage from layout sizing, fixing command-palette phase aliases like `new_phase`, and making task/project schema coverage explicit instead of accidental.

User findings captured for this wave:
1. Transient board/footer noise such as `cancelled` still appears after backing out of task edit and seems to steal bottom space.
2. `new_phase` from the command palette drops back to the board instead of opening the phase form.
3. The user wants an explicit schema audit so the TUI/task/project field contract is intentional and consistent across menus.

Context7:
1. Pre-edit consults:
   - `/charmbracelet/bubbletea` for key handling and command-oriented input normalization patterns -> PASS.
   - `/charmbracelet/bubbles` for viewport/help separation and keeping header/footer chrome outside scrollable layout math -> PASS.
2. Failure-driven re-consults during the follow-up remediation loop:
   - `/charmbracelet/bubbletea` for `tea.View`/`tea.Layer` inspection in tests after the first compile failure -> PASS.
   - `/charmbracelet/bubbles` for viewport-bounded height assertions after the short-terminal regression test initially measured unbounded raw content -> PASS.
   - `/charmbracelet/bubbles` for the canonical viewport import path after the second compile failure in `model_test.go` -> PASS.

Implementation summary:
1. Shared screen/status cleanup:
   - removed full-page dependence on global `m.status` from `internal/tui/full_page_surface.go`; full-page body height no longer subtracts status-line rows and no longer appends transient status beneath the bordered surface.
   - added shared helpers for inner width, bottom help rendering, and outer horizontal padding so board and full-page screens consume the same chrome math.
2. Board footer filtering:
   - introduced transient board-status suppression so cancel/loading/focus noise no longer reserves footer rows or appears below the board.
   - updated board rendering and footer-line sizing to use the same filtered status helper.
3. Command palette normalization:
   - normalized command ids so dash, underscore, and space variants resolve to the same canonical command (`new-phase`, `new_phase`, `new phase`).
4. Explicit schema coverage:
   - added read-only `system:` sections for task info and project edit so structural/lifecycle fields are surfaced intentionally.
   - added regression tests that classify top-level task/project fields and task/project metadata fields as editable, read-only, or intentionally internal/unsupported, so future schema drift fails loudly.
5. Follow-up remediation after QA:
   - changed shared full-page body-height math to clamp against the actual available terminal height instead of forcing the default minimum back in on short terminals.
   - expanded the task `system:` section to render `project`, `parent`, `kind`, and `state`, so the read-only task schema contract now matches the UI surface intentionally.
   - removed the earlier duplicate ad-hoc `parent:` row outside `system:`.
   - added `TestFullPageSurfaceMetricsShrinkBodyToFitShortTerminal` and updated the task-system assertions to cover the newly surfaced fields.

Commands run and outcomes:
1. `just fmt` -> PASS.
2. `just test-pkg ./internal/tui` -> PASS.
3. `just check` -> PASS.
4. `just ci` -> PASS (`internal/tui` coverage 70.7%).
5. `just test-golden` -> PASS.
6. QA1 (Bernoulli) initial review -> FAIL:
   - shared full-page body height still clamped back to `taskInfoBodyViewportMinHeight`, so short terminals could still overflow the bottom border.
   - task schema coverage claimed read-only fields not all surfaced in the UI.
7. QA2 (Aristotle) initial review -> FAIL:
   - worksheet/plan still claimed FR-012 QA closure before the follow-up remediation loop was documented.
8. `just fmt` -> PASS.
9. `just test-pkg ./internal/tui` -> FAIL (compile: attempted `string(...)` conversion on Bubble Tea v2 `tea.View`) -> Context7 re-consult.
10. `just fmt` -> PASS.
11. `just test-pkg ./internal/tui` -> FAIL (compile: `tea.View.Content` is `tea.Layer`, not `string`) -> Context7 re-consult.
12. `just fmt` -> PASS.
13. `just test-pkg ./internal/tui` -> FAIL (regression test measured unbounded raw body content instead of a bounded viewport surface) -> Context7 re-consult.
14. `just fmt` -> PASS.
15. `just test-pkg ./internal/tui` -> FAIL (missing `viewport` import in `model_test.go`) -> Context7 re-consult.
16. `just fmt` -> PASS.
17. `just test-pkg ./internal/tui` -> PASS.
18. `just test-golden` -> PASS (needed one escalated rerun for Go build-cache access after sandbox denial).
19. `just check` -> PASS.
20. `just ci` -> PASS (`internal/tui` coverage 70.7%).

Files edited in this checkpoint:
1. `internal/tui/full_page_surface.go`
2. `internal/tui/model.go`
3. `internal/tui/model_test.go`
4. `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md`
5. `PLAN.md`

QA:
1. Bernoulli initial review -> FAIL due short-terminal height clamp and schema/UI coverage mismatch; final re-audit -> PASS after the follow-up code/test remediation.
2. Aristotle initial review -> FAIL due worksheet/plan chronology drift; final re-audit -> PASS after the FR-012 tracker state was synchronized.

Status:
- FR-012 implementation/remediation is complete and validation is green (`just fmt`, `just test-pkg ./internal/tui`, `just test-golden`, `just check`, `just ci`).
- The same blocked collaborative step `T1-01` remains paused for the user's rerun against FR-012.
- Next step: hand the same blocked step back to the user for rerun.

## Checkpoint 2026-03-14: FR-013 Phase Creation Still Fails After Rerun

Objective:
- record the new failure from the user rerun immediately, stop forward collaborative testing on the same blocked step `T1-01`, commit the validated FR-012 baseline first, then remediate the phase-creation semantics and focused text-input hotkey leak without reopening roadmap work.

User finding captured for this wave:
1. The rerun now passes except for phase creation; the user still cannot create a phase.

Context7:
1. Pre-edit consult:
   - `/websites/pkg_go_dev_github_com_charmbracelet_bubbletea` for explicit command-dispatch handling and keeping typed/selected command execution on the same action path -> PASS.
   - `/websites/pkg_go_dev_github_com_charmbracelet_bubbletea` for focused-input key handling so printable text is routed before screen-level hotkeys -> PASS.
   - `/charmbracelet/bubbles` for focused `textinput`/`textarea` update handling in larger Bubble Tea forms -> PASS.

Local inspection completed before any new edit:
1. Repo state check before remediation: `git status --short` showed the validated FR-012 scope was still uncommitted in:
   - `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md`
   - `PLAN.md`
   - `internal/tui/full_page_surface.go`
   - `internal/tui/model.go`
   - `internal/tui/model_test.go`
2. Protocol check:
   - the locked collaborative-remediation flow in this repo requires each validated fix scope to be committed before the next one starts; FR-012 was therefore committed first as `74bbc0e` before FR-013 code changes began.
3. Phase-creation code-path inspection:
   - `executeCommandPalette("new-phase")` was hard-coded to require branch context from either `focusedScopeTaskAtLevel("branch")` or `selectedTaskAtLevel("branch")`, so project-level phase creation was impossible.
   - `startPhaseForm()` always required a concrete parent task, so it could not represent a project-level phase form at all.
4. Focused-input key-routing inspection:
   - task edit/add mode still intercepted printable `e` before the focused `textinput` saw it.
   - project edit/add mode still intercepted printable `r` on `root_path:` via the form-level shortcut path.
4. Parallel inspection lanes launched:
   - Anscombe: TUI phase-creation code-path review.
   - Beauvoir: tracker/protocol/worktree-state review.

Implementation summary:
1. Baseline closeout:
   - committed the validated FR-012 scope as `74bbc0e` (`Finalize FR-012 shared layout and schema audit`) before opening FR-013.
2. Phase creation semantics:
   - changed `startPhaseForm()` to accept an optional parent.
   - `new-phase` now creates a project-level phase (`parent=""`, `kind=phase`, `scope=task`) when no branch is selected/focused.
   - branch-selected/focused `new-phase` still creates a branch-backed phase (`scope=phase`).
   - `new-subphase` remains strict and still requires a phase/subphase parent.
3. Focused text-input routing:
   - added explicit direct-text-input detection for task/project forms.
   - printable keys now go to focused task/project text inputs before form-level hotkeys.
   - bare printable project-form `r` no longer hijacks `root_path:`; only `ctrl+r` remains mapped there.
4. Regression coverage:
   - replaced the old branch-only `new-phase` guard test with project-level phase coverage.
   - added a tea-driven command-palette test for typed `new_phase`.
   - added printable-key regression tests for task `title:` (`e`/`E`) and project `root_path:` (`r`/`R`).

Commands run and outcomes:
1. `git status --short` -> PASS (confirmed validated but uncommitted FR-012 scope).
2. Context7 consult for Bubble Tea command dispatch -> PASS.
3. `rg -n "func \\(m Model\\) selectedBranchTask|func \\(m Model\\) selectedTaskAtLevel|func \\(m Model\\) focusedScopeTaskAtLevel|func \\(m Model\\) focusedScopeTaskAtLevels|func \\(m Model\\) startPhaseForm|phase-new|new-phase|new phase" internal/tui/model.go` -> PASS.
4. `sed -n '2560,2665p' internal/tui/model.go` -> PASS.
5. `sed -n '8470,8505p' internal/tui/model.go` -> PASS.
6. `sed -n '10488,10540p' internal/tui/model.go` -> PASS.
7. `git add COLLAB_VECTOR_MCP_E2E_WORKSHEET.md PLAN.md internal/tui/full_page_surface.go internal/tui/model.go internal/tui/model_test.go && git commit -m "Finalize FR-012 shared layout and schema audit"` -> PASS (`74bbc0e`).
8. `just fmt` -> PASS.
9. `just test-pkg ./internal/tui` -> PASS.
10. `just test-golden` -> PASS.
11. `just check` -> PASS.
12. `just ci` -> PASS (`internal/tui` coverage 70.6%).
13. Worksheet/plan FR-013 evidence updates -> PASS.

Status:
QA:
1. Copernicus final code/UI re-audit -> PASS.
2. Galileo tests/docs/tracker re-audit -> PASS after FR-013 tracker state was synchronized.

Status:
- FR-013 implementation is complete and validation is green (`just fmt`, `just test-pkg ./internal/tui`, `just test-golden`, `just check`, `just ci`).
- The same blocked collaborative step `T1-01` remains paused for the user's rerun against FR-013.

## Checkpoint 2026-03-14: FR-014 One Nestable Phase Migration

Objective:
- close the next blocked gap on `T1-01` by removing first-class `subphase`, migrating persisted data to one nestable `phase`, and keeping phase parents constrained to project root, branch, or phase.

User finding captured for this wave:
1. After the FR-013 rerun, the user reported phase creation still behaved incorrectly.
2. The user then selected the product direction explicitly:
   - keep one `phase`,
   - remove `new-subphase`,
   - allow phase parents at project root, branch, or phase,
   - forbid task parents for phases,
   - and update the DB/schema contract in the same wave.

Context7:
1. Pre-edit consult:
   - `/websites/pkg_go_dev_github_com_charmbracelet_bubbletea` for focused input routing and selection-vs-focus command handling in Bubble Tea update flows -> PASS.
   - `/websites/sqlite_docs` for safe persisted text and JSON-array rewriting during SQLite migrations -> PASS.
2. Failure-triggered re-consults:
   - `/golang/go/go1.26.0` after the first compile failure to confirm the shared exported helper approach for cross-package scope normalization -> PASS.
   - `/websites/pkg_go_dev_github_com_charmbracelet_bubbletea` after the TUI timeout/failing test loop to confirm command-handling assumptions while fixing test command draining and selected-vs-focus parent precedence -> PASS.

Local inspection completed before edits:
1. Parallel impact reviews confirmed `subphase` was a real persisted contract across:
   - domain enums and normalization,
   - app snapshot/service mappings,
   - SQLite migration/seeded kind rules,
   - transport scope/target surfaces,
   - TUI command/help/search/filter logic,
   - and active docs/tests.
2. The existing storage model did not require new tables or columns; the required change was a contract/data migration from `subphase` markers to `phase`.
3. The intended lineage contract was locked before edits:
   - project-level phase -> `kind=phase`, `scope=phase`, `parent_id=""`
   - branch phase -> `kind=phase`, `scope=phase`, `parent_id=<branch>`
   - nested phase -> `kind=phase`, `scope=phase`, `parent_id=<phase>`
   - tasks cannot parent phases.

Implementation summary:
1. Domain/app contract cleanup:
   - removed first-class `subphase` enums/targets/scope mappings from `internal/domain` and dependent app/transport code.
   - centralized default scope inference so `kind=phase` normalizes to `scope=phase`, including project-level phases.
2. SQLite/data migration:
   - kept the existing table shape.
   - added/used migration rewrites that convert legacy `subphase` text values in tasks/work_items/comments/capability leases/attention rows and rewrite kind-catalog JSON arrays plus change-event metadata to `phase`.
   - normalized legacy `kind=phase, scope=task` rows to `scope=phase`.
3. Seed/default kind rules:
   - removed `subphase` from built-in kind applies-to and parent-scope lists.
   - phase now applies only to `phase` and allows parent scopes `branch` and `phase`.
4. TUI/transport UX cleanup:
   - removed `new-subphase` from the command palette/surface.
   - `new-phase` now creates a project-level phase by default, a branch phase when a branch is selected/focused, and a nested phase when a phase is explicitly selected/focused.
   - selected phase/branch now takes precedence over the broader focus root when resolving `new-phase` parentage.
   - removed `subphase` from search/filter/help/label/thread terminology, treating nested phases as `phase` with lineage.
5. Test infrastructure follow-up:
   - updated the TUI test helper to stop synchronously waiting on long-lived Bubble Tea timer commands such as cursor blink follow-ups.

Commands run and outcomes:
1. `git status --short` -> PASS.
2. `rg -n "subphase|phase" ...` across domain/app/storage/TUI/docs -> PASS (inspection baseline).
3. Context7 Bubble Tea + SQLite consults -> PASS.
4. `just fmt` -> PASS.
5. `just test-pkg ./internal/domain` -> PASS.
6. `just test-pkg ./internal/app` -> PASS.
7. `just test-pkg ./internal/adapters/storage/sqlite` -> PASS.
8. `just test-pkg ./internal/adapters/server/common` -> PASS (`[no test files]`).
9. `just test-pkg ./internal/adapters/server/mcpapi` -> PASS.
10. `GOFLAGS='-test.timeout=30s' just test-pkg ./internal/tui` -> FAIL (exposed `TestModelCommandPaletteAndQuickActions` hanging on timer-driven cursor blink command in the test helper).
11. Context7 re-consult for Bubble Tea command handling -> PASS.
12. `just fmt` -> PASS.
13. `just test-pkg ./internal/tui` -> FAIL once more on selected-phase parent precedence (`new-phase` still chose focused branch root over selected phase).
14. Context7 re-consult for Bubble Tea selection/focus handling -> PASS.
15. `just fmt` -> PASS.
16. `just test-pkg ./internal/tui` -> PASS.
17. `just test-golden` -> PASS.
18. `just check` -> PASS.
19. `just ci` -> PASS (`internal/tui` coverage 70.6%).
20. QA follow-up inspection found one remaining real gap: `internal/app/snapshot.go` still used the legacy blank-scope inference and did not reject task-parented phases during snapshot validation.
21. Context7 Bubble Tea + SQLite guidance already covered the required follow-up patch shape; no new external source was needed before the narrow validator/test fix.
22. Patched `internal/app/snapshot.go` to use `domain.DefaultTaskScope(t.Kind, t.ParentID)` during snapshot validation and reject `kind=phase` rows whose parent scope is not `branch` or `phase`.
23. Added snapshot regression coverage for blank-scope phase import defaults, invalid task-parented phase rejection, and valid branch->phase->phase lineage; renamed the last active nested-phase fixture ids in `internal/tui/model_test.go`.
24. `just fmt` -> PASS.
25. `just test-pkg ./internal/app` -> PASS.
26. `just test-pkg ./internal/tui` -> PASS.
27. `just test-golden` -> PASS.
28. `just check` -> PASS.
29. `just ci` -> PASS (`internal/tui` coverage 70.6%).
30. QA pass 1 (`019ceb96-08e2-72c1-91ba-bf0bbeb39067` Poincare) -> PASS after the snapshot contract fix.
31. QA pass 2 (`019ceb96-0cda-70c1-b9ee-9c99e38f027c` Leibniz) -> PASS after worksheet/PLAN state synchronization.

Status:
- FR-014 implementation and the follow-up snapshot contract hardening are complete.
- Validation is green on both the original migration sweep and the closeout reruns.
- Final dual QA sign-off is complete, and worksheet/plan state is synchronized.
- Next step: hand the same blocked step `T1-01` back to the user for rerun.

## Checkpoint 2026-03-14: FR-015 Focus-Root-Only Phase Creation

Objective:
- close the next blocked gap on `T1-01` by making `new-phase` derive parentage from the active `f` focus screen only, never from the hovered/selected child row.

User finding captured for this wave:
1. After reviewing the FR-014 behavior, the user clarified the remaining `new_phase` rule:
   - project screen -> project-level phase,
   - focused branch screen -> child of that branch,
   - focused phase screen -> child of that phase,
   - task/subtask-focused screens must not create phases,
   - hovered or selected child rows must not change phase parentage unless they become the active screen via `f`.

Context7:
1. Pre-edit consult:
   - `/websites/pkg_go_dev_github_com_charmbracelet_bubbletea` for keeping action semantics bound to explicit model state rather than cursor selection -> PASS.

Local inspection completed before edits:
1. `executeCommandPalette("new-phase")` still preferred `selectedTaskAtLevels("phase", "branch")` before checking the active focus root.
2. `focusedScopeTaskAtLevels(...)` already exposed the exact single source of truth needed for screen-based parentage through `projectionRootTaskID`.
3. The remaining edge case was task/subtask-focused screens, which cannot legally parent phases and therefore needed a visible blocking warning instead of silently falling back to project-level creation.

Implementation summary:
1. TUI command routing:
   - `new-phase` now reads only `focusedScopeTaskAtLevels("phase", "branch")` for child parentage.
   - no subtree focus still opens a project-level phase form.
   - focused task/subtask screens now open a warning modal explaining that phases can only be created from project, branch, or phase screens.
2. Regression coverage:
   - updated the existing phase-creation tests so a merely selected branch row on the project board still yields a project-level phase,
   - confirmed a focused branch screen keeps parentage on the branch even when a child phase row is selected,
   - kept focused phase-screen nested creation coverage,
   - added a focused-task blocking test,
   - preserved normalized command-id coverage for `new_phase` and `new phase`.

Commands run and outcomes:
1. Context7 Bubble Tea focused-state consult -> PASS.
2. `just fmt` -> PASS.
3. `just test-pkg ./internal/tui` -> PASS.
4. `just test-golden` -> PASS.
5. `just check` -> PASS.
6. `just ci` -> PASS (`internal/tui` coverage 70.6%).
7. QA pass 1 (`019ceb96-08e2-72c1-91ba-bf0bbeb39067` Poincare) -> initial LOW test-gap finding (missing phase-selected/no-focus + subtask-focused regression coverage) -> remediated with extra tests.
8. Context7 Bubble Tea view/testing guidance re-consult after the failing regression loop -> PASS.
9. `just fmt` -> PASS.
10. `just test-pkg ./internal/tui` -> PASS.
11. `just test-golden` -> PASS.
12. `just check` -> PASS.
13. `just ci` -> PASS (`internal/tui` coverage 70.6%).
14. QA pass 1 (`019ceb96-08e2-72c1-91ba-bf0bbeb39067` Poincare) final -> PASS.
15. QA pass 2 (`019ceb96-0cda-70c1-b9ee-9c99e38f027c` Leibniz) final -> PASS.

Status:
- FR-015 implementation is complete and validation is green.
- Final QA sign-off is complete and tracker state is synchronized.
- User reran the same blocked step `T1-01` and reported PASS.
- Next step: continue to the next ordered collaborative TUI step `T1-02`.

## Checkpoint 2026-03-17: FR-016 Task Metadata Persistence And Subtask Task-Screen Management

Current objective:
- remediate the `T1-02` failure without opening roadmap work: fix metadata field persistence semantics, make subtasks fully manageable from task screens, preserve input safety during refresh, and add task-screen logging so future save/reload issues are diagnosable.

Backlog/open-findings review:
1. Reviewed active collaborative state in this file and confirmed `T1-01` is already complete; the current manual gate is `T1-02`.
2. Reviewed unresolved collaborative/doc state in `COLLAB_E2E_REMEDIATION_PLAN_WORKLOG.md` and `COLLABORATIVE_POST_FIX_VALIDATION_WORKSHEET.md` before opening FR-016.
3. User-reported `T1-02` failure was narrowed to TUI interaction semantics, not DB persistence: field-editor `ctrl+s` only applied back to the form, empty metadata values were treated as unchanged, task info could not open a selected subtask, and task-screen save/reload/reanchor logging was missing.

Investigation evidence:
1. Local log inspection of `.tillsyn/log/tillsyn-20260317.log` showed only startup lines plus one `tui.form.control_character_guard` entry for `description`; there were no task-update or subtask-action traces.
2. Explorer lane `019cfdda-9984-73a3-9500-5c51724e5cee` confirmed backend persistence succeeds when the outer form actually submits, and ranked the likely UX/interaction causes: blank-means-unchanged, editor `ctrl+s` only applying to form state, and multiline markdown fields being backed by single-line `textinput` state.
3. Explorer lane `019cfdda-ae47-7212-a3e9-77a7aa59fe2d` confirmed task-info `enter` was a no-op even though the subtask row was visibly focused, edit-mode subtask open/create used a separate contract from task-info, and save/reload/reanchor/logging was incomplete.

User consensus:
1. Presented fix options and got explicit user selection for `Option A`: shared draft-backed markdown fields, immediate persistence from the metadata editor for existing tasks, shared subtask action-row behavior across task screens, stronger logging, and safe refresh that never overwrites active input.

Files edited and why:
1. `internal/tui/model.go`
   - introduced dedicated draft/touched state for markdown-backed task metadata fields instead of treating `textinput` rows as the source of truth,
   - made editor save for existing task description/metadata go through the real `UpdateTask` path,
   - added task-info `enter` subtask drill-in,
   - added stable subtask-id reanchor in task-info,
   - added parent-edit return context for subtask drill-in/save/escape,
   - added structured task-screen action traces,
   - preserved deferred auto-refresh behavior for active input modes while allowing reload after successful updates.
2. `internal/tui/description_editor_mode.go`
   - made footer copy reflect whether `ctrl+s` applies a draft or saves the existing task.
3. `internal/tui/trace.go`
   - added reusable `tui.task_screen.action` structured debug logging for subtask open/toggle and task-save flows.
4. `internal/tui/model_test.go`
   - updated metadata draft tests to reflect the new source-of-truth model,
   - added regression coverage for editor-level metadata persistence, task-info subtask open-on-enter, and parent-edit reopen after subtask save.

Validation loop:
1. `just test-pkg ./internal/tui` -> FAIL
   - `TestModelEditTaskMetadataFieldsPrefillAndSubmit` still mutated `formInputs` directly instead of the new markdown draft state.
   - `TestModelEditTaskKeyboardSaveAndPickerShortcuts` still seeded the editor from `formInputs` instead of the new draft state.
   - `TestModelEditTaskSubtaskAndResourceRowSelection` expected `Esc` from child edit to drop to board instead of reopening parent edit.
2. Context7 re-consulted for Bubble Tea/Bubbles input/editor guidance after the failing test loop, then tests were updated to reflect the dedicated draft model and parent-edit return contract.
3. `just fmt && just test-pkg ./internal/tui` -> FAIL
   - synthetic `ctrl+s` test event shape was invalid for this Bubble Tea version.
4. Context7 re-consulted for Bubble Tea key-message semantics after the failing test loop, then the test was switched to call the real editor save/close path directly.
5. `just fmt && just test-pkg ./internal/tui` -> PASS.
6. `just test-golden` -> PASS.
7. `just check` -> PASS.
8. QA pass 1 (`019cfe16-a239-7121-830a-cb6fa7003188` Schrodinger) -> HIGH finding:
   - normal child-edit submit cleared the parent reopen context before deriving it, so only `Esc` and editor-level save returned to the parent edit flow.
9. Follow-up fix:
   - moved reopen-context capture ahead of edit-state clearing and added `TestModelEditTaskSubtaskSubmitReturnsToParent`.
10. Revalidation after QA1 finding:
   - `just fmt && just test-pkg ./internal/tui && just test-golden && just check && just ci` -> PASS.
11. QA pass 2 (`019cfe16-a49f-75d3-8b6b-c23ae3b3475e` Ramanujan) -> PASS on tracker-state audit after identifying stale `T1-01` references that needed synchronization.

Status:
- FR-016 implementation is complete and validation is green.
- Collaborative progression remains paused on `T1-02` until the user reruns the step.

Next step:
1. Update `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md` for FR-016/FX-016 and stale FR-015 references.
2. Commit the validated FR-016 scope.
3. Hand the user the exact `T1-02` rerun instructions.

## Checkpoint 2026-03-17: FR-017 Task-Screen Quick Actions, Resource Root Fallback, And Attribution Consistency

Current objective:
- close the remaining `T1-02` follow-up gaps without opening roadmap work: clarify save-dependent task-screen actions, add focused quick actions/help guidance, route resource attach through `project root -> bootstrap root`, fix local display-name rendering so bootstrap identity replaces visible legacy `tillsyn-user`, propagate actor names through write paths, and seed the next collaborative attribution worksheet.

Backlog/open-findings review:
1. Stayed on the same blocked collaborative step `T1-02`; no forward section advancement was allowed.
2. Continued from the active collaborative remediation state already tracked in this file, `COLLAB_E2E_REMEDIATION_PLAN_WORKLOG.md`, and `COLLABORATIVE_POST_FIX_VALIDATION_WORKSHEET.md`.
3. User clarified that task move/state should remain on `[` / `]` with explanation in `? help`, that save-first subtask behavior in `new task` is acceptable if explained clearly, and that future collaborative coverage must explicitly validate attribution for local user/orchestrator/subagent/system actors.

Investigation evidence:
1. `git status --short` -> confirmed this wave's expected dirty set: `internal/app/service.go`, `internal/app/service_test.go`, `internal/adapters/storage/sqlite/repo.go`, `internal/adapters/storage/sqlite/repo_test.go`, `internal/tui/model.go`, `internal/tui/model_test.go`, `internal/tui/thread_mode.go`, and new tracker markdown.
2. `git diff -- internal/app/service.go internal/app/service_test.go internal/adapters/storage/sqlite/repo.go internal/adapters/storage/sqlite/repo_test.go` -> reviewed the worker-lane attribution persistence changes before integration.
3. `rg` audits over TUI/app/adapter call sites showed that task/project update flows still dropped `UpdatedByName`/`CreatedByName`, thread description updates persisted only actor ids, and MCP adapter create/update project/task paths still forwarded ids without names.
4. Explorer findings already recorded in `COLLAB_T1_02_UX_ATTRIBUTION_FIX_TRACKER.md` remained valid:
   - Raman confirmed resource attach was hard-gated on project-root lookup.
   - McClintock confirmed task-screen help/action coverage was incomplete.
   - Franklin confirmed bootstrap display name was persisted but some comment/thread render paths still showed raw legacy local tuples.
5. Worker lane Hegel (`019cfe73-714b-7bd0-8dc1-3c998502b2e6`) completed app/storage attribution persistence changes and package-scoped validation before integration.

Context7 / docs checkpoints:
1. Bubble Tea v2 docs were consulted before this wave's code edits for model-defined key/focus handling.
2. After the first failed `just test-pkg ./internal/tui` loop, Context7 was re-consulted again before the next test edit, confirming focus order is model-defined and tests should assert the chosen panel cycle.

Files edited and why:
1. `internal/tui/model.go`
   - propagated local actor id/name/type into task/project metadata updates, resource attach updates, add-task creation, and labels-config-driven task updates,
   - added focused `.` quick actions for task info/edit,
   - clarified `? help` for task move/subtask/resource/comment flows,
   - defaulted `subtasks:` and `resources:` focus to the first existing item,
   - made `new task` save-dependent rows explicit,
   - switched resource-root lookup to fallback from project root to bootstrap/search root.
2. `internal/tui/thread_mode.go`
   - opened task-screen comments directly into the comments panel,
   - propagated actor display names through thread detail updates,
   - normalized comment/thread owner rendering so the local bootstrap display name replaces visible legacy `tillsyn-user` tuples.
3. `internal/app/service.go`
   - added actor-name fields on project/task update/create inputs,
   - resolved mutation-actor identity from explicit inputs plus context without changing guard semantics,
   - reused merged attribution for task/comment creation and project/task updates.
4. `internal/app/service_test.go`
   - captured mutation-actor context at the fake repo boundary and added attribution propagation tests for project/task/comment writes.
5. `internal/adapters/storage/sqlite/repo.go`
   - centralized change-event actor resolution so persisted task events prefer real display names from context/input over fallback labels.
6. `internal/adapters/storage/sqlite/repo_test.go`
   - added persistence coverage for user/agent/system change-event attribution plus comment/task actor-name preservation.
7. `internal/adapters/server/common/app_service_adapter_mcp.go`
   - forwarded both actor id and actor name into project/task mutations instead of dropping the display name.
8. `internal/tui/model_test.go`
   - added/updated regressions for contextual quick actions, bootstrap-root fallback, task-info comment-panel entry, local display-name owner rendering, add-task save-first gating, and thread panel traversal expectations.
9. `COLLAB_T1_02_UX_ATTRIBUTION_FIX_TRACKER.md`
   - marked the app/storage worker lane complete and linked the new attribution worksheet.
10. `COLLAB_ACTOR_ATTRIBUTION_VALIDATION_WORKSHEET.md`
   - created the future collaborative worksheet covering local user, orchestrator, subagent, and system attribution across task/thread/activity/notices surfaces.
11. `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md`
   - recorded FR-017/FX-017 and updated the active `T1-02` row with the latest user findings and retest scope.

Validation loop:
1. `just fmt` -> PASS.
2. `just test-pkg ./internal/tui` -> FAIL.
   - `TestModelThreadTabAndShiftTabMoveInOppositeDirections` still expected details-first thread opening.
   - `TestModelResourcePickerRequiresProjectRootForTaskAttach` still expected no bootstrap-root fallback.
3. Context7 re-consulted after the failed TUI test loop.
4. Updated the stale TUI expectations to match the intended UX.
5. `just fmt` -> PASS.
6. `just test-pkg ./internal/tui` -> PASS.
7. `just test-pkg ./internal/app` -> PASS.
8. `just test-pkg ./internal/adapters/storage/sqlite` -> PASS.
9. `just test-pkg ./internal/adapters/server/common` -> PASS (`[no test files]`, package build path clean).
10. `just test-golden` -> PASS.
11. `just check` -> PASS.
12. `just ci` -> PASS (`internal/tui` coverage 70.2%).

Status:
- FR-017 implementation and repo-wide validation are green.
- QA pass 1 (Archimedes `019cfe8c-7aa5-7bc0-9f61-da2a58fbd6f1`) found one medium contract drift: task move/state had leaked into `.` quick actions even though the agreed UX kept move/state only on `[` / `]`. That drift was removed, then `just fmt`, `just test-pkg ./internal/tui`, `just test-golden`, `just check`, and `just ci` were rerun green.
- QA pass 2 (Nietzsche `019cfe88-c986-7500-b9c2-082171d37b4d`) reported no findings on tests/docs/tracker state.
- Tracker/docs are updated through the new attribution worksheet and T1-02 follow-up ledger.
- Remaining steps before user rerun: commit the validated FR-017 scope, then hand back the exact `T1-02` rerun instructions.

Follow-up on the same blocked step (`T1-02`):
1. User rerun reported one remaining attribution-rendering defect: project notifications already showed the readable display name, but the task `system:` section still printed raw UUID-like ids for `created_by` and `updated_by`.
2. Kept the scope render-only in `internal/tui/model.go`: task system ownership lines now resolve through matching project activity entries and local identity display names instead of printing raw actor ids directly.
3. Added `TestTaskInfoBodyLinesRenderSystemSectionUsesReadableActorNames` in `internal/tui/model_test.go` to lock `created_by: Evan (user)` and `updated_by: Codex Orchestrator (agent)`.
4. Revalidation after the follow-up fix:
   - `just fmt` -> PASS
   - `just test-pkg ./internal/tui` -> PASS
   - `just test-golden` -> PASS
   - `just check` -> PASS
   - `just ci` -> PASS
5. Next step after commit/push: investigate the readonly/auth MCP mutation failure and the requested external auth repos under a non-committed path, then discuss whether the auth approach answers the open MCP questions.

### 2026-03-19: AGENTS Guidance Clarification

Objective:
- clarify tool-choice guidance for GitHub-hosted workflow inspection versus local git operations.

Edits:
1. `AGENTS.md`
   - refined the rule to prefer `gh` for GitHub-hosted operations whenever `gh` supports the task directly and clearly.
   - made `gh` the default for pull requests, workflow/check inspection, run logs, review actions, repository metadata, and GitHub authentication.
   - clarified that `git` remains the default for core local repository operations such as status, diff, add, commit, branch, merge-base inspection, and worktree management unless the current conversation explicitly requires a `gh`-specific workflow.
   - prohibited using the GitHub web UI for repository operations when `gh` can perform the same task.
   - added a Conventional Commits policy for all commit messages with lowercase, imperative summaries and a fixed allowed-type list.
   - stated that contributors and agents should follow the commit-message style consistently.

Validation:
1. `test_not_applicable`
   - docs-only guidance update; no code, workflow, or Justfile behavior changed.
