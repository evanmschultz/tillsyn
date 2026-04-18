# Drop 1 Prerequisite — ActionItem-Scope Orchestrator Auth

**Date**: 2026-04-17
**Branch**: `main` (prerequisite hotfix, same pattern as `DROP_1_UNBLOCK_MULTI_ORCH_AUTH_2026-04-17.md`)
**Status**: SCOPE DEFINED — awaiting QA review of this worklog
**Prior art**: `DROP_1_UNBLOCK_MULTI_ORCH_AUTH_2026-04-17.md` (multi-orch same-scope coexistence, landed in `7781c02`), `AUTH_LAYER_RESEARCH_2026-04-17.md` (broader auth-layer survey).

## 1. Purpose

Let orchestrator-role auth sessions resolve at **actionItem scope**, exactly the same way they resolve at project scope today. Nothing more. No new semantics.

Why: post-Drop-2, every coordination boundary below the project is a nested `actionItem` (kind collapse to `drop` is deferred). Today, the auth layer collapses every actionItem-scope request back to project scope, so every orch lives at project scope. That forces STEWARD + DROP_1_ORCH + DROP_1.5_ORCH + future drop orchs to all pile up at `project/<id>`. ActionItem-scope orch auth lets each drop orch scope itself to the actionItem subtree it actually owns.

## 2. Scope

### In

- 2.1 `internal/domain/auth_request.go:163-184` — path parser accepts `/actionItem/<actionItem-id>` segments after `/project/<id>`. Literal vocabulary `actionItem` (no rename to `drop`).
- 2.2 `internal/domain/auth_request.go:210-274` + `276-296` + `299-329` — `Normalize`, `String`, `LevelTuple` produce / round-trip `ScopeLevelActionItem`.
- 2.3 `internal/app/auth_scope.go:164-168` — stop collapsing actionItem scope back to project. Preserve actionItem-scope context in `AuthScopeContext`.
- 2.4 `internal/app/auth_scope.go:134-170` (`authScopeContextFromActionItemLineage`) — return actionItem lineage without forcing project-scope flattening.
- 2.5 `internal/adapters/server/common/app_service_adapter_mcp.go:496-547` (`authRequestPathWithin`) — compare actionItem segment array by prefix, same way branch/phase arrays are compared today.
- 2.6 `internal/adapters/auth/autentauth/service.go:~1090` (clone of `authRequestPathWithin`) — same fix.
- 2.7 `internal/adapters/server/mcpapi/handler.go:107` — `till_auth_request` MCP `path` parameter docstring mentions `/actionItem/<id>`.
- 2.8 Tests: parser round-trip, path-within containment, `ResolveAuthScopeContext` preserves actionItem lineage, CLI `approve` lifecycle with `--path project/<id>/actionItem/<id>`, SQLite round-trip for actionItem-scope auth request row.

### Out (explicitly deferred)

- 2.9 **Subtask vocabulary** — dev directive: actionItem only. Not adding `/subtask/<id>` path support.
- 2.10 **`drop` rename** — deferred to Drop 2's kind collapse. Keep literal `actionItem` vocabulary.
- 2.11 **Nested-actionItem overlap semantics** — orch at `actionItem:A` vs. orch at `actionItem:B` nested under A. Today's overlap check buckets leases by exact `(scopeType, scopeID)` tuple (`ensureOrchestratorOverlapPolicy` at `internal/app/kind_capability.go:425-466`), so ancestor/descendant pairs never meet. **This carries forward unchanged** — same semantics as today's project-scope behavior. Nested-overlap policy is a future design question, not this change.
- 2.12 **Overlap-by-ancestor widening** — no new repo query walking actionItem parent chains.
- 2.13 **Session-row schema promotion** — `auth_sessions` continues to carry scope only via `ApprovedPath` TEXT. No new `ScopeType`/`ScopeID` columns.
- 2.14 **TUI styled rendering of actionItem-scope paths** — `authRequestPathDisplay` at `internal/tui/model.go:16429-16460` already falls back to raw-string display on parse error. Once the parser accepts actionItem, the breadcrumb renders as raw `project/<id>/actionItem/<id>` — readable, unstyled. Styled segment rendering deferred.
- 2.15 **MCP `approve` operation** — separate auth-layer refinement per prior research §5.3; independent of scope.

## 3. Design Decisions (Pre-Resolved By Dev)

- 3.1 **Vocabulary**: keep `actionItem`. No `drop` rename.
- 3.2 **Subtask**: out. ActionItem only.
- 3.3 **Nested overlap**: carry today's semantics forward — different `(scopeType, scopeID)` tuples never collide in the overlap check. Explicit policy ("project-scope orch subsumes descendants") is a future question.
- 3.4 **Session schema**: keep text `ApprovedPath`. No column promotion.
- 3.5 **Ship location**: on `main` as a second prerequisite hotfix (same pattern as multi-orch fix).

## 4. What ActionItem Scope Already Gives Us For Free

- 4.1 Lease validator `internal/app/kind_capability.go:401-422` already accepts `CapabilityScopeActionItem` when `actionItem.Scope==KindAppliesToActionItem`. No change needed.
- 4.2 SQLite `auth_requests.scope_type` is TEXT, no enum constraint (`internal/adapters/storage/sqlite/repo.go:495-528`). Stores `"actionItem"` as-is, no migration.
- 4.3 `authSessionApprovedPath` at `internal/adapters/server/common/app_service_adapter_mcp.go:484-493` re-parses from TEXT at read-time — once the grammar accepts actionItem, sessions round-trip automatically.
- 4.4 `kind_catalog` already lets tasks carry any `agent_type`; the orch role reaches actionItem scope through the auth path, not through kind binding.
- 4.5 Proof the lease layer works: `internal/app/template_contract_test.go:134, 198, 223, 258` already issue `CapabilityScopeActionItem` leases in-fixture. The gate is purely reachability from the auth-path layer.

## 5. Change Set

### 5.1 Production Files (6)

| File | Change |
|---|---|
| `internal/domain/auth_request.go` | Parser accepts `/actionItem/<id>`; `Normalize`, `String`, `LevelTuple` round-trip `ScopeLevelActionItem`. |
| `internal/app/auth_scope.go` | Remove actionItem→project collapse at line 164-168; preserve actionItem lineage in `AuthScopeContext`. |
| `internal/adapters/server/common/app_service_adapter_mcp.go` | `authRequestPathWithin` compares actionItem segment arrays. |
| `internal/adapters/auth/autentauth/service.go` | Same fix to the clone. |
| `internal/adapters/server/mcpapi/handler.go` | MCP `path` docstring update. |
| (no TUI change) | Raw-string fallback already graceful. |

Net expected: ~100-150 production LoC.

### 5.2 Test Files (~5)

| File | Test |
|---|---|
| `internal/domain/auth_request_test.go` | `TestParseAuthRequestPath` adds actionItem cases: `project/p1/actionItem/t1`, `project/p1/actionItem/t1/actionItem/t2`, `project/p1/branch/b1/actionItem/t1`. `TestAuthRequestPathRoundTripAndLevelTuple` round-trips each. |
| `internal/adapters/auth/autentauth/service_test.go` | `authRequestPathWithin` actionItem-array containment: `project/p1/actionItem/t1` contains `project/p1/actionItem/t1/actionItem/t2`; does NOT contain `project/p1/actionItem/t2`. |
| `internal/app/auth_scope_test.go` | `TestResolveAuthScopeContextPreservesActionItemLineage` — no branch ancestor, actionItem scope flows through. |
| `internal/adapters/storage/sqlite/repo_test.go` | `TestRepository_AuthRequestRoundTripActionItemScope` — seed + read actionItem-scope auth request row end-to-end. |
| `cmd/till/main_test.go` | `TestRunAuthRequestApproveLifecycleActionItemScope` — CLI approve path with `--path project/<id>/actionItem/<id>`. |

Net expected: ~150-250 test LoC.

## 6. Risk

- 6.1 **Load-bearing collapse at `auth_scope.go:164`.** Every current caller assumes actionItem/subtask scope gets flattened to project. Removing that collapse needs a read of every callsite of `ResolveAuthScopeContext` to confirm no caller depends on the flatten.
  - Mitigation: planner verifies all callsites via `LSP` references before builder starts. QA Falsification attacks this directly.
- 6.2 **Subtask collapse still active.** Per §2.9, we do NOT touch subtask. The collapse of subtask→project or subtask→actionItem remains. This is intentional — only actionItem scope is in scope.
  - Mitigation: the test in §5.2 for `ResolveAuthScopeContext` asserts subtask-scope callers still flatten as before (no regression in the out-of-scope path).
- 6.3 **Same-package concurrency with Drop 1 work.** Drop 1 touches `internal/app` (auth TTL, lifecycle). This change touches `internal/app/auth_scope.go`. Same package = same compile unit. Coordinate via explicit sequencing: this hotfix lands on `main` first, Drop 1 branches off the post-fix `main`.
- 6.4 **Path storage text-format change for stored sessions.** Not applicable: no existing stored session has an actionItem-scope path today (the collapse prevented creation). All existing stored `ApprovedPath` values are project/branch/phase shape. New parser is additive — old paths still parse.

## 7. Ordering

- 7.1 Atomic commit required: grammar (`auth_request.go`) + lineage (`auth_scope.go`) + path-within (`app_service_adapter_mcp.go` + autent clone). Any one alone leaves actionItem-scope paths half-working.
- 7.2 MCP docstring (`handler.go:107`) is additive — can ship in the same atomic commit or separately.
- 7.3 Tests land in the same commit as the production change they cover.

## 8. Verification

- 8.1 `mage ci` green before push. Coverage threshold 70% per package.
- 8.2 **Live smoke**: after `mage build`, file a fresh orch auth request at `path=project/<id>/actionItem/<actionItem-id>` via `mcp__tillsyn-dev__till_auth_request(operation=create)`. Dev approves via CLI: `main/till auth request approve --request-id <id>`. Claim. Issue a capability lease at `scope_type=actionItem, scope_id=<actionItem-id>`. Verify lease list returns the actionItem-scope orch lease alongside existing project-scope leases. Revoke + cleanup. Mirror of the multi-orch smoke test.

## 9. QA Verdicts

*To be filled by `go-qa-proof-agent` + `go-qa-falsification-agent` against this worklog before any builder work begins.*

### 9.1 QA Proof — *PENDING*

### 9.2 QA Falsification — *PENDING*

### 9.3 Convergence — *PENDING*

## 10. Unknowns / Follow-Ups

- 10.1 (Design, deferred per §2.11) Nested-actionItem orch overlap semantics. Needs a separate decision before multiple orchs start claiming nested actionItem scopes in practice.
- 10.2 (Design, deferred per §2.12) Overlap-by-ancestor widening — should project-scope orch lease block actionItem-scope orch lease under it?
- 10.3 (Engineering, deferred per §2.15) MCP `till_auth_request` lacks `approve` operation — CLI-only today.
- 10.4 (Drop 2 rider) When kind collapse renames `actionItem` → `drop` in the kind layer, auth-path vocabulary either renames in lockstep (`project/<id>/drop/<id>`) or stays `actionItem` indefinitely. Decision sits with Drop 2.

## 11. Hylla Feedback (From Agents)

*Aggregated from each agent's closing `## Hylla Feedback` section at hotfix close.*
