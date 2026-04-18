# Drop 1 Unblock — Multi-Orch Auth Fix (Worklog)

**Date opened:** 2026-04-17
**Branch:** `main` (hotfix, not a drop branch)
**Orchestrator:** STEWARD
**Cross-reference:** `main/PLAN.md` §19.0.5

---

## 1. Problem Statement

The auth layer's "one active auth session per scope level" rule (documented in `main/CLAUDE.md` §"Auth and Leases") blocks concurrent orchestrators from coexisting at the same scope level. DROP_1_ORCH was given project-scope auth by the dev as the pre-Drop-2 workaround for the drop-collapse parity bug documented in `main/AUTH_LAYER_RESEARCH_2026-04-17.md`. That project-scope session blocks STEWARD from obtaining its own project-scope auth.

Consequence: Drop 1 cannot begin because the drop requires both orchestrators active — STEWARD for MD writes + per-drop findings routing, DROP_1_ORCH for code work on `drop/1`.

## 2. Acceptance Criteria

- 2.1 Two different orchestrator identities (different `principal_id`) can hold active sessions at the same scope level simultaneously.
- 2.2 A single orchestrator identity attempting to claim twice at the same scope level is still rejected (or returns the existing session — planner decides).
- 2.3 Revocation of one orchestrator's session does not affect the other's.
- 2.4 `mage ci` passes on `main`.
- 2.5 `main/CLAUDE.md` + bare-root `CLAUDE.md` "Auth and Leases" bullet updated to reflect the new rule.
- 2.6 No pre-existing sessions or leases invalidated by the change.

## 3. Open Questions — Planner Answers (2026-04-17)

- 3.1 **Enforcement locus** — **Go runtime check only.** `Service.ensureOrchestratorOverlapPolicy` at `internal/app/kind_capability.go:425-459`, called from `Service.IssueCapabilityLease` at `kind_capability.go:279-283` when `lease.Role == CapabilityRoleOrchestrator`. No DB UNIQUE index exists on `capability_leases` (`internal/adapters/storage/sqlite/repo.go:455-471`) or `auth_requests` (`repo.go:495-528`). No separate `auth_sessions` table. Session layer (`internal/app/auth_requests.go:173-300`, `internal/adapters/auth/autentauth/service.go:422-472`) has no overlap check at all — the gate is purely the capability-lease policy.
- 3.2 **DB impact** — **NO DB CHANGES.** No schema migration, no UNIQUE index, no new column, no data rewrite. `capability_leases.agent_name` already exists (`repo.go:458`) and is the identity column the fix needs. Existing leases + sessions unchanged.
- 3.3 **Re-claim semantics** — **Reject with existing `domain.ErrOrchestratorOverlap`.** Same-identity re-claim surfaces as a caller bug (forgot to revoke, or ignored hook-cached tuple). The existing project-policy `override_token` path (`kind_capability.go:443-456`) remains the intentional-replacement escape hatch. Silent-return is rejected because it would return old `lease_token` / `instance_id` with potentially mismatched TTL.
- 3.4 **Cross-role interaction** — **Fix stays orch-only. Non-orch roles not touched.** `ensureOrchestratorOverlapPolicy` is only invoked when `lease.Role == CapabilityRoleOrchestrator` AND short-circuits on `existing.Role != CapabilityRoleOrchestrator` (line 436). Builder / QA / research / planner leases already stack freely at any scope today. Behavior unchanged for them.
- 3.5 **Audit surface** — **No at-most-one assumptions to fix.** Downstream callers of `ListCapabilityLeasesByScope` all handle N leases gracefully: `capability_inventory.go:45` (list semantics), `snapshot.go:941` (dedup by `InstanceID`), `kind_capability.go:365` (`RevokeAllCapabilityLeases`). Revocation / heartbeat by `InstanceID` single-row operations unaffected — acceptance 2.3 structurally guaranteed.

## 4. Plan (Planner Output — Transcribed 2026-04-17)

### 4.1 Concrete Fix Scope

- 4.1.1 Production files to edit (exactly two):
  - `main/internal/app/kind_capability.go` — modify `ensureOrchestratorOverlapPolicy` at lines 425-459.
  - `main/internal/app/service_test.go` — add three new test functions + rewrite one existing test in the vicinity of line 3592+.
- 4.1.2 No domain-layer changes. No adapter changes. No schema / migration changes. No MCP tool schema changes. No CLI changes.

### 4.2 Exact Change in `ensureOrchestratorOverlapPolicy`

- 4.2.1 Update doc comment at line 425: `// ensureOrchestratorOverlapPolicy enforces project policy for overlapping orchestrator leases held by a DIFFERENT agent identity. Same-identity overlap continues to block unless the project override policy is satisfied.`
- 4.2.2 Inside the loop at lines 432-457, immediately after the three existing short-circuits (`existing.InstanceID == next.InstanceID` at 433, `existing.Role != CapabilityRoleOrchestrator` at 436, `!existing.IsActive(now)` at 439), add a fourth branch:
  - Compute `sameIdentity := strings.TrimSpace(existing.AgentName) != "" && strings.TrimSpace(existing.AgentName) == strings.TrimSpace(next.AgentName)`.
  - If `!sameIdentity`: **`continue`** (allow the second orch identity at the same scope).
  - If `sameIdentity`: fall through to the existing policy block (`AllowOrchestratorOverride` + `OrchestratorOverrideToken` path at 443-456) unchanged.
- 4.2.3 Function signature, return value, caller contract — all unchanged.

### 4.3 Test Additions / Modifications in `service_test.go`

- 4.3.1 **NEW** `TestIssueCapabilityLeaseAllowsDistinctOrchestratorIdentities` — two orchs with `AgentName="orch-a"` and `AgentName="orch-b"` issue project-scope orch leases back-to-back. Both succeed. Assert both rows exist via `ListCapabilityLeases`. Asserts acceptance 2.1.
- 4.3.2 **NEW** `TestIssueCapabilityLeaseRejectsSameIdentityReclaim` — one orch `AgentName="orch-a"` issues project-scope orch lease successfully. Second issue with same `AgentName="orch-a"`, different `AgentInstanceID`, fails with `domain.ErrOrchestratorOverlap` (or `ErrOverrideTokenRequired` if project has `AllowOrchestratorOverride: true` without a token — match existing error shape). Asserts acceptance 2.2.
- 4.3.3 **NEW** `TestIssueCapabilityLeaseRevokeOneIdentityLeavesOthers` — orchs A and B both hold project-scope orch leases. Revoke A. Assert `ListCapabilityLeases(IncludeRevoked=false)` returns B only and B is active. Asserts acceptance 2.3.
- 4.3.4 **NEW** `TestIssueCapabilityLeaseOverlapDifferentIdentitiesNoTokenRequired` — two distinct identities, same project-scope, no override policy set, both issues succeed. Cements acceptance 2.1 against the policy-less default.
- 4.3.5 **REWRITE — HIGH RISK** `TestIssueCapabilityLeaseOverlapPolicy` at `service_test.go:3619-3659` currently uses FOUR DISTINCT `AgentName` values (`orch-a`, `orch-b`, `orch-c`, `orch-d`) and expects the second+third+fourth issues to hit the override-token policy. Under the new rule those would all succeed without tokens. Builder must rewrite this test to use the SAME `AgentName` across all four subcases (e.g. `orch-alpha`) while keeping `AgentInstanceID` distinct per issue. Preserve all four original assertions: first-succeeds / second-requires-token / third-invalid-token / fourth-override-succeeds. Builder must NOT delete this test — it protects the same-identity override-token lane.
- 4.3.6 **NEW** `TestIssueCapabilityLeaseSameInstanceIDRetry` — same `AgentName="orch-a"` AND same `AgentInstanceID="inst-a"` retry. Exercises the `existing.InstanceID == next.InstanceID` short-circuit at `kind_capability.go:433` which runs BEFORE the new identity check. Assert behavior is whatever the existing short-circuit already produces today (fake repo silently overwrites, SQLite PK errors — builder confirms actual repo behavior and asserts accordingly). Flag divergence between fake-repo and sqlite behavior as an unknown if it surfaces. Closes falsification coverage gap 3.1.
- 4.3.7 **NEW** `TestIssueCapabilityLeaseSameIdentityAfterExpiry` — one orch `AgentName="orch-a"` issues a project-scope lease with short TTL. Advance clock past `ExpiresAt`. Same identity re-issues. Succeeds (via `!existing.IsActive(now)` short-circuit at `kind_capability.go:439`). Closes falsification coverage gap 3.2.
- 4.3.8 **NEW** `TestIssueCapabilityLeaseSameIdentityAfterRevoke` — one orch `AgentName="orch-a"` issues a project-scope lease successfully. Explicitly `RevokeCapabilityLease` on that lease's instance_id. Same identity re-issues without override token. Succeeds (via same line-439 short-circuit, since revoked leases are not `IsActive`). Closes falsification coverage gap 3.3.
- 4.3.9 **NEW** `TestIssueCapabilityLeaseDistinctIdentitiesBranchScope` — repeat §4.3.1 but at `CapabilityScope: CapabilityScopeBranch` (or `CapabilityScopeTask`; builder picks whichever matches existing test fixtures). Proves the fix is scope-type-agnostic, not project-scope-only by accident of the test data. Closes falsification coverage gap 3.4.

### 4.4 Verification Target

- 4.4.1 `mage test-pkg ./internal/app` for focused package run during development.
- 4.4.2 `mage ci` as the full gate before push (per `main/CLAUDE.md` §Build Verification).

### 4.5 Acceptance Criteria Mapping

- 4.5.1 → `TestIssueCapabilityLeaseAllowsDistinctOrchestratorIdentities` + `TestIssueCapabilityLeaseOverlapDifferentIdentitiesNoTokenRequired`.
- 4.5.2 → `TestIssueCapabilityLeaseRejectsSameIdentityReclaim` + rewritten `TestIssueCapabilityLeaseOverlapPolicy`.
- 4.5.3 → `TestIssueCapabilityLeaseRevokeOneIdentityLeavesOthers`.
- 4.5.4 → STEWARD runs `mage ci` post-build.
- 4.5.5 → STEWARD post-merge edits both CLAUDE.md files: replace `main/CLAUDE.md:276` (and bare-root equivalent) "One active auth session per scope level at a time." with "One active orchestrator lease per scope level per orchestrator **identity** (the lease's `AgentName` — bound to `session.PrincipalID` for agent-session issuers per `internal/adapters/server/mcpapi/extended_tools.go:228-229`). Two different orchestrator identities can hold active leases at the same scope concurrently; same-identity re-issue is rejected with `ErrOrchestratorOverlap` unless the project policy's `override_token` is supplied (see `internal/app/kind_capability.go:443-456`). `AgentName` is a coordination primitive, not a security boundary — a future Drop 1.6 deliverable will bind it to the approve-cascade audit trail."
- 4.5.6 → Structurally guaranteed by "no schema change, no data rewrite, runtime-only."

### 4.6 YAGNI Boundary

- 4.6.1 No new MCP operation. `till_capability_lease` enum unchanged.
- 4.6.2 No new CLI command.
- 4.6.3 No new domain type. No new error value — reuse `ErrOrchestratorOverlap` + `ErrOverrideTokenRequired`.
- 4.6.4 No new config knob / project metadata field.
- 4.6.5 No new adapter layer. No new interface method.
- 4.6.6 No new DB column. `agent_name` already exists.
- 4.6.7 No Drop 1 / Drop 1.6 / Drop 2 dependency.
- 4.6.8 Net diff shape: ~5 production lines + 3 new test functions + 1 rewrite.

### 4.7 Risk Register

- 4.7.1 **R1 — Test rewrite hazard (HIGH).** Builder may read §4.3.5 as a pure addition and miss that `TestIssueCapabilityLeaseOverlapPolicy` requires modification, not just new tests. QA falsification must verify rewrite preserves all four original assertions under the new same-identity pattern.
- 4.7.2 **R2 — `AgentName` forgeability (ACCEPTED OUT-OF-SCOPE).** DROP_1_ORCH could pass `AgentName="STEWARD"` and bypass uniqueness. Documented in CLAUDE.md update as a coordination primitive, not a security boundary. Drop 1.6 ships the approve-cascade audit binding.
- 4.7.3 **R3 — TrimSpace normalization mismatch (LOW).** `NewCapabilityLease` already trims `AgentName` at `capability.go:129`, so stored values are pre-trimmed. Using `TrimSpace` on both sides is defensive + consistent with the override-token comparison pattern at lines 447 + 454. Zero cost.

## 5. QA Planning Verdict

**Converged 2026-04-17 after two-pass review.** Both passes spawned with fresh context windows and the `go-qa-proof-agent` / `go-qa-falsification-agent` subagent types per `main/CLAUDE.md` §QA Discipline.

### 5.1 QA Proof — Verdict: CONFIRMED

Proof pass verified evidence completeness across all 17 concrete file:line citations in §3 and §4 (function names, line ranges, error constants, short-circuit conditions, test fixtures, MCP agent-session call site, SQLite schema shape). Every claim is backed by an inspectable Go symbol; no citation is fabricated or misaligned. The "no DB change" conclusion is structurally sound — no UNIQUE index exists on `capability_leases` or `auth_requests`, and `agent_name` is already present at `repo.go:458`. The fix surface (~5 production lines) and test coverage (acceptance → test mapping in §4.5) are complete for the proposed change. No proof-side findings requiring plan revision.

### 5.2 QA Falsification — Verdict: PLAN-MUST-REVISE (All 4 Action Items Applied)

Falsification pass attempted four counterexamples. All applied to the plan:

- **5.2.1 Coverage gap — same-InstanceID retry.** Plan did not test the `existing.InstanceID == next.InstanceID` short-circuit at `kind_capability.go:433`, which runs BEFORE the new identity check. Applied as §4.3.6 `TestIssueCapabilityLeaseSameInstanceIDRetry`.
- **5.2.2 CLAUDE.md wording ambiguity.** §4.5.5 said "orchestrator identity" without binding "identity" to a concrete field. Applied — §4.5.5 amended to cite `AgentName` and the `session.PrincipalID` binding at `extended_tools.go:228-229` explicitly, plus flag `AgentName` as a coordination primitive (not a security boundary) with the Drop 1.6 audit binding called out.
- **5.2.3 Coverage gap — expiry + revoke interactions.** Plan did not test that the `!existing.IsActive(now)` short-circuit at `kind_capability.go:439` still permits same-identity re-issue after TTL expiry or explicit revoke. Applied as §4.3.7 `TestIssueCapabilityLeaseSameIdentityAfterExpiry` + §4.3.8 `TestIssueCapabilityLeaseSameIdentityAfterRevoke`.
- **5.2.4 TOCTOU window pre-existing but unflagged.** Plan did not document that `ensureOrchestratorOverlapPolicy` + `CreateCapabilityLease` are not transactional. Applied as §10.3 with explicit ACCEPTED / out-of-scope rationale — fix does not expand the race surface; concurrency hardening deferred to Drop 1 proper.

Scope-type coverage was also tightened as a belt-and-suspenders item (§4.3.9 `TestIssueCapabilityLeaseDistinctIdentitiesBranchScope`) to prove the fix is scope-type-agnostic, not an artifact of the project-scope test data.

### 5.3 Convergence

Proof pass CONFIRMED evidence completeness. Falsification pass produced no unmitigated counterexample — all 4 action items applied before builder spawn. Plan is ready for build.

## 6. Build Log

**Builder:** `go-builder-agent` · **Completed:** 2026-04-17.

### 6.1 Files Edited

- `main/internal/app/kind_capability.go` — `ensureOrchestratorOverlapPolicy` (lines 425-459 → 425-463 after edit).
  - Doc comment at line 425 expanded per §4.2.1 to name the different-vs-same identity distinction.
  - Added `sameIdentity` branch between the `!existing.IsActive(now)` short-circuit and the project-policy override block. `sameIdentity` is true only when both `existing.AgentName` and `next.AgentName` are non-empty (post `strings.TrimSpace`) and equal. When `!sameIdentity`, the loop `continue`s the existing lease row (different orchestrator identity is permitted to coexist). When `sameIdentity`, control falls through to the pre-existing `AllowOrchestratorOverride` + `OrchestratorOverrideToken` policy block, unchanged.
  - Function signature, return types, caller contract unchanged. No new imports (`strings` already imported).
- `main/internal/app/service_test.go` — rewrote one existing test, added eight new tests below.

### 6.2 Test Rewrite (§4.3.5, HIGH RISK per §4.7.1)

`TestIssueCapabilityLeaseOverlapPolicy` (was lines 3592-3660; now 3592-3665):

- **Old `AgentName` pattern:** four distinct values (`orch-a`, `orch-b`, `orch-c`, `orch-d`) each with a matching `AgentInstanceID`.
- **New `AgentName` pattern:** single shared value `orch-alpha` across all four sub-cases, with `AgentInstanceID` rotated per-issue (`orch-alpha-1`, `orch-alpha-2`, `orch-alpha-3`, `orch-alpha-4`).
- **All four assertions preserved identically:**
  1. First issue: `err == nil` (no existing overlap).
  2. Second issue (no override token): `err == domain.ErrOverrideTokenRequired`.
  3. Third issue (wrong override token): `err == domain.ErrOverrideTokenInvalid`.
  4. Fourth issue (correct override token `override-123`): `err == nil`.
- **Reason for rewrite:** under the new `sameIdentity` gate, distinct `AgentName` values bypass the override-token policy entirely (the new `continue` in `ensureOrchestratorOverlapPolicy`). The original test's 2nd-4th sub-cases would have started succeeding silently without exercising the override-token lane. Using one shared `AgentName` keeps the override-token code path under test. Doc comment above the function annotates the invariant.

### 6.3 Test Functions Added

All eight are top-level `func Test...(t *testing.T)` functions in `main/internal/app/service_test.go`, appended after the rewritten `TestIssueCapabilityLeaseOverlapPolicy` (before `TestCreateTaskMutationGuardRequiredForAgent`).

| Test | Asserts |
|---|---|
| `TestIssueCapabilityLeaseAllowsDistinctOrchestratorIdentities` (§4.3.1) | Two orchestrator leases with `AgentName` `orch-a` and `orch-b` at the same project scope both succeed without override-token. `ListCapabilityLeases` returns both rows; both names present. Acceptance 2.1. |
| `TestIssueCapabilityLeaseRejectsSameIdentityReclaim` (§4.3.2) | Same `AgentName="orch-a"` reclaim with different `AgentInstanceID` returns `domain.ErrOverrideTokenRequired` on an override-enabled project with no token supplied. Acceptance 2.2. |
| `TestIssueCapabilityLeaseRevokeOneIdentityLeavesOthers` (§4.3.3) | Orchs A and B hold concurrent project-scope leases. `RevokeCapabilityLease(orch-a)` leaves `ListCapabilityLeases(IncludeRevoked=false)` returning only `orch-b` with `RevokedAt == nil`. Acceptance 2.3. |
| `TestIssueCapabilityLeaseOverlapDifferentIdentitiesNoTokenRequired` (§4.3.4) | Two distinct `AgentName`s on a project with `AllowOrchestratorOverride=false` (default) both succeed. Policy block never consulted on the non-matching-identity lane. Belt-and-suspenders cement of acceptance 2.1. |
| `TestIssueCapabilityLeaseSameInstanceIDRetry` (§4.3.6) | Same `AgentName` and same `AgentInstanceID` retry succeeds via the `existing.InstanceID == next.InstanceID` short-circuit at line 433 (runs before the new identity check). Second call returns the retry's own fresh `LeaseToken`; fake repo's idempotent `CreateCapabilityLease` ends at exactly one row. Closes falsification gap 3.1. |
| `TestIssueCapabilityLeaseSameIdentityAfterExpiry` (§4.3.7) | Same `AgentName="orch-a"` re-issues after the clock advances past the first lease's `ExpiresAt` (5 min TTL, 10 min advance). Succeeds via the `!existing.IsActive(now)` short-circuit at line 439. Closes falsification gap 3.2. |
| `TestIssueCapabilityLeaseSameIdentityAfterRevoke` (§4.3.8) | Same `AgentName="orch-a"` re-issues after `RevokeCapabilityLease` on the first instance. Succeeds via the same line-439 short-circuit (revoked lease is not `IsActive`). Closes falsification gap 3.3. |
| `TestIssueCapabilityLeaseDistinctIdentitiesBranchScope` (§4.3.9) | Same shape as §4.3.1 but with `ScopeType: CapabilityScopeBranch` and a synthesized branch row via `CreateTask(Scope: KindAppliesToBranch)`. Two distinct identities coexist on a branch-scope lease without override token. `ListCapabilityLeases(branch, scopeID=branch.ID)` returns both. Proves scope-type-agnostic. Closes falsification gap 3.4. |

### 6.4 Verification Output

**`mage test-pkg ./internal/app`:**

```
[RUNNING] Running go test ./internal/app
[SUCCESS] Test stream detected
[PKG PASS] github.com/evanmschultz/tillsyn/internal/app (1.40s)

Test summary
  tests: 212
  passed: 212
  failed: 0
  skipped: 0
  packages: 1

[SUCCESS] All tests passed
  212 tests passed across 1 package.
```

Each of the 9 changed/new tests was additionally run via `mage test-func ./internal/app <TestName>` — all 9 green individually, confirming the new functions are reached by Go's test runner and exercise the intended code path.

**`mage ci`:**

Four stages, all green:

1. **Sources** — tracked sources verified.
2. **Formatting** — gofmt clean across tracked `.go` files.
3. **Coverage** — `go test -cover ./...` across 19 packages, 1261 tests, all passed. Minimum package coverage 70.0% threshold met. `internal/app` coverage 71.3%. Highest: `internal/buildinfo` 100.0%; lowest above threshold: `internal/tui` 70.6%. No package below threshold.
4. **Build** — `till` binary built from `./cmd/till` successfully.

### 6.5 Unknowns Surfaced

None. §4.3.6 "divergence between fake-repo and SQLite behavior" (flagged in the plan as a potential unknown) was confirmed on the fake-repo side: `fakeRepo.CreateCapabilityLease` at `service_test.go:965-968` silently overwrites by `InstanceID`, and the test asserts exactly that behavior (single row after same-InstanceID retry, rotated `LeaseToken`). SQLite-side behavior for the same retry was NOT exercised in this hotfix — the production code path relies on the app-layer short-circuit at `kind_capability.go:433` to keep the SQLite adapter from ever seeing a duplicate-PK conflict on a legitimate retry. If a caller somehow reaches `CreateCapabilityLease` with a clashing InstanceID, the SQLite adapter's behavior is out-of-scope for this hotfix (no production call-site does so today — `IssueCapabilityLease` is the sole issuer path and it consults `ensureOrchestratorOverlapPolicy` first).

## 6a. Hylla Feedback

None — Hylla answered everything needed. This hotfix worked entirely from `Read` + `Grep` against committed files because the investigation was file- and line-range-scoped (exact function rewrite plus new tests inside a single known `*_test.go`). No query was attempted against Hylla, so no miss was logged. Consistent with the AUTH_LAYER_RESEARCH §9 note that grammar/switch-statement-shaped investigations are better served by keyword + read than vector search.

## 6b. Integration Test Build Log

Closes coverage gap flagged in §7.2.2: add one SQLite-backed integration test that exercises the distinct-identity orchestrator overlap fix through a real `Service` backed by `OpenInMemory()`, not the fake repo used in `internal/app` service-level coverage.

### 6b.1 File Edited

- `main/internal/adapters/storage/sqlite/repo_test.go` — added top-level test `TestRepository_CapabilityLeaseDistinctOrchestratorIdentitiesAtProjectScope` at lines 1869-1991 (122 new lines, inserted between `TestRepository_CapabilityLeaseRoundTrip` at 1809-1866 and `TestRepository_AttentionItemRoundTrip` at the next block so the capability-lease tests stay co-located). No other file touched in the SQLite adapter package. No helper added — the test reuses the existing `OpenInMemory()` + `app.NewService(repo, idGen, clock, cfg)` idiom that already appears in the same file at lines 666 + 1703.

### 6b.2 Assertion Trace

Step-by-step, in order:

1. Build a real SQLite-backed `Service` via `OpenInMemory()` + `app.NewService` with a deterministic 3-slot id generator (`p-multi-orch`, `lease-token-steward`, `lease-token-drop-1`) and a fixed UTC clock (`2026-04-17T12:00:00Z`). Matches the existing service-layer pattern at `repo_test.go:666+`.
2. `svc.CreateProject(ctx, "Multi-Orch Project", "")` creates the project row (consumes the first id).
3. `svc.IssueCapabilityLease` with `AgentName: "orch-steward"`, `AgentInstanceID: "orch-steward-inst"`, `Role: CapabilityRoleOrchestrator`, `ScopeType: CapabilityScopeProject` — assert `err == nil`, `LeaseToken != ""`, `InstanceID != ""`. Exercises the sameIdentity=false branch's `continue` lane (first lease has no prior row to skip past, but the policy block runs).
4. `svc.IssueCapabilityLease` with `AgentName: "orch-drop-1"`, `AgentInstanceID: "orch-drop-1-inst"`, same project+scope+role — assert `err == nil`, `LeaseToken` distinct from the first lease's token. THIS is the core assertion: under the hotfix, distinct `AgentName` at the same project scope must succeed without an override token. Exercises `ensureOrchestratorOverlapPolicy` with `existing=orch-steward` and `next=orch-drop-1`, hitting the `!sameIdentity → continue` branch at `kind_capability.go:444-448`.
5. `svc.ListCapabilityLeases` with `ProjectID=project.ID`, `ScopeType=CapabilityScopeProject`, default `IncludeRevoked=false` — assert exactly 2 rows returned, both with `RevokedAt == nil`, and both `AgentName` values present via a `map[string]bool` check. Proves the SQLite adapter persists both rows without structural rejection (no UNIQUE collision on `project_id, scope_type, scope_id, role`) and that the service-level active-only filter surfaces both.
6. `svc.RevokeCapabilityLease(AgentInstanceID: leaseSteward.InstanceID, Reason: "done")` — assert `err == nil`. Explicitly revokes the first identity.
7. `svc.ListCapabilityLeases` again (active only) — assert exactly 1 row, `AgentName == "orch-drop-1"`, `RevokedAt == nil`, `ExpiresAt.After(now)` (lease TTL defaults to `defaultCapabilityLeaseTTL = 24h` via `kind_capability.go:92-95`, so `ExpiresAt` is 24h ahead of the fixed clock). Proves per-instance revoke does NOT cascade to the peer identity — acceptance 2.3 at the SQLite persistence boundary.
8. `repo.GetCapabilityLease(ctx, leaseSteward.InstanceID)` (direct-repo read, bypassing service filter) — assert `RevokedAt != nil`. Confirms the revoke actually persisted in the row, not merely masked by the list filter.

### 6b.3 `mage test-pkg ./internal/adapters/storage/sqlite`

```
[RUNNING] Running go test ./internal/adapters/storage/sqlite
[SUCCESS] Test stream detected
[PKG PASS] github.com/evanmschultz/tillsyn/internal/adapters/storage/sqlite (1.16s)

Test summary
  tests: 69
  passed: 69
  failed: 0
  skipped: 0

[SUCCESS] All tests passed
  69 tests passed across 1 package.
```

Package test count moved from 68 → 69 (one new top-level test). Additionally ran `mage test-func ./internal/adapters/storage/sqlite TestRepository_CapabilityLeaseDistinctOrchestratorIdentitiesAtProjectScope` in isolation with `-race` enabled (mage wraps `-race -count=1`) — 1/1 green in 8.31s. Race-clean on the fresh SQLite handle.

### 6b.4 `mage ci`

Four stages, all green. Verified on current working tree.

1. **Sources** — tracked sources verified.
2. **Formatting** — gofmt clean.
3. **Coverage** — `go test -cover ./...` across **19 packages, 1262 tests, 1262 passed, 0 failed, 0 skipped**. Up from 1261 in §6.4 — exact +1 delta matches the single new test. Minimum package-coverage floor 70.0% met everywhere. `internal/adapters/storage/sqlite` coverage **72.3%**, measured delta **+0.2 pp** (pre-test baseline on the same tree was 72.1% via a `git stash` round-trip). Other package coverages unchanged from §6.4: `internal/app` 71.3%, `internal/buildinfo` 100.0%, `internal/tui` 70.6% (lowest above floor).
4. **Build** — `till` binary built from `./cmd/till` successfully.

### 6b.5 Unknowns / Hylla Feedback

**Unknowns:** None blocking. One notable (ACCEPTED, out-of-scope): the hotfix-invariant from §6.5 that `existing.InstanceID == next.InstanceID` short-circuits at `kind_capability.go:433` before the SQLite adapter sees a PK collision on `capability_leases.instance_id` is NOT exercised by this integration test — it is fake-repo-covered only at `service_test.go:3900+`. Adding the same coverage at the SQLite boundary was out-of-scope for this gap-closing task; file under Drop 1 test-coverage if desired.

**Hylla Feedback:** None — Hylla answered everything needed. This work used `Grep` + `Read` against committed files because the question was "what is the existing helper pattern for service-layer SQLite tests and the existing capability-lease test in this package?" — a scope-bounded pattern-match best served by keyword search, not semantic retrieval. No Hylla query was attempted, so no miss was logged. Consistent with §6a.



## 7. QA Build Verdict

Both passes spawned in parallel with fresh context windows per `main/CLAUDE.md` §QA Discipline.

### 7.1 QA Proof — Verdict: CONFIRMED

Re-verified every claim in §6 against on-disk state. Production edit at `internal/app/kind_capability.go:425-466` matches §4.2 exactly (doc comment verbatim, `sameIdentity` branch at lines 444-448, no new imports, no signature change, no caller change). All 8 new test functions present at cited line numbers; rewrite at `internal/app/service_test.go:3597-3664` preserves all four override-token assertions under shared `AgentName="orch-alpha"` with distinct `AgentInstanceID`s. Scope hygiene clean (`git diff --name-only` = `PLAN.md` + the two Go files only; the two worklog/research MDs untracked and authorized). Independent `mage test-pkg ./internal/app` = 212/212 green; independent `mage ci` = 1261/1261 green across 19 packages, `internal/app` coverage 71.3% exact-match for §6.4. YAGNI boundary intact — zero new top-level declarations. No proof gaps.

### 7.2 QA Falsification — Verdict: BUILD-CONFIRMED (2 accepted-with-caveats)

Ten attack surfaces probed, all refuted or reduced to soft caveats:

- Rewrite still gates the override-token lane (sub-cases 2/3/4 reach the policy block because `sameIdentity=true` under shared `orch-alpha`).
- No hidden test in the repo relied on old distinct-identity blocking behavior; `ErrOrchestratorOverlap` has zero test-assertion uses pre- or post-fix, so no test breaks.
- Single caller of `ensureOrchestratorOverlapPolicy` (`kind_capability.go:280`); signature unchanged.
- Single MCP source for `AgentName` assignment (`extended_tools.go:228-229`); no competing handler.
- No new concurrency surface — diff is 5 lines inside an existing for-loop, no goroutines.
- `mage ci` reproduces builder claims byte-for-byte.
- Neither `mage install` nor raw `go` invocations used.

Two accepted-with-caveats, both pre-existed the hotfix scope:

- **7.2.1 (Soft / pre-existing)** `ErrOrchestratorOverlap` reachable at `kind_capability.go:452` when `sameIdentity && !AllowOrchestratorOverride`, but no test exercises it. The rewritten `TestIssueCapabilityLeaseOverlapPolicy` always sets `AllowOrchestratorOverride: true` (inherited from the pre-fix shape). Not a regression. Optional follow-up: add one test with `AllowOrchestratorOverride: false` + same-identity reclaim asserting `ErrOrchestratorOverlap`. File under Drop 1 test-coverage.
- **7.2.2 (Soft / pre-existing)** No SQLite-backed integration test exercises two distinct-identity orchestrator leases at the same scope. Schema (`repo.go:455-471`: `instance_id PRIMARY KEY` only) confirms SQLite admits the new behavior, and `CreateCapabilityLease` (`repo.go:3232`) is a plain INSERT. Production SQLite is structurally compatible. Optional follow-up: add a SQLite-backed integration test. Not a blocker for the hotfix.

One cosmetic note: the rewritten `TestIssueCapabilityLeaseOverlapPolicy` keeps a 9-entry `ids` slice but now consumes only 5 slots after the rewrite. Dead slots are harmless; no fix required.

### 7.3 Convergence

Proof CONFIRMED. Falsification BUILD-CONFIRMED. Two caveats both pre-existed the fix and are logged as optional Drop 1 follow-ups. Hotfix is ready for commit.

## 8. CI Gate

- 8.1 `mage ci` run twice independently — once by builder, once by QA Proof — both green. 1261/1261 tests across 19 packages. `internal/app` coverage 71.3%, minimum-package floor (70.0%) met everywhere. Sources/Formatting/Coverage/Build stages all green. `till` binary built.
- 8.2 Pre-existing lint informational diagnostics in files NOT touched by this hotfix (`handoffs.go:265`, `service.go:1892`, `kind_capability.go:857`, `kind_capability.go:1040`, `service_test.go:411/759/770/906/2022`) are informational only, not gate failures — `mage ci` returned SUCCESS. None were introduced by the hotfix diff; they precede this change and are out-of-scope per §4.6.

## 9. Commit + Push

*Pending CI green. STEWARD commits code + PLAN.md + worklog + CLAUDE.md updates, pushes, runs `gh run watch --exit-status`.*

## 10. Unknowns / Follow-Ups

- 10.1 (Planner, non-blocking) Whether to emit a structured `charmbracelet/log` line when same-identity retry is rejected vs different-identity accepted, for future audit visibility. Current code does not log at this site. Default for this hotfix: no new log lines. Dev preference decision, not a correctness question.
- 10.2 (Planner, non-blocking) Whether the CLAUDE.md wording should also mention that `AgentName` is the lease-level identity field so future planners know which field to inspect. **Resolved by QA Falsification Action Item 2 — §4.5.5 amended to cite `AgentName` + `extended_tools.go:228-229` binding explicitly.**
- 10.3 (QA Falsification, ACCEPTED — pre-existing, out-of-scope) **TOCTOU window preserved.** The fix modifies `ensureOrchestratorOverlapPolicy` but does NOT wrap that check + the subsequent `CreateCapabilityLease` call (at `internal/app/kind_capability.go:284+`) in a single DB transaction. Two concurrent orchestrators passing the overlap check simultaneously could race to `CreateCapabilityLease`. This TOCTOU window exists today for the override-token path and is unchanged by this hotfix — the fix is not expanding the race surface. Addressing it needs a wrapping transaction + probable interface changes on the capability-lease repo; explicitly out-of-scope for the hotfix. File under Drop 1 concurrency hardening.

## 11. Hylla Feedback (From Agents)

*Aggregated from each agent's closing `## Hylla Feedback` section. Rolled up into `main/HYLLA_FEEDBACK.md` at hotfix close.*
