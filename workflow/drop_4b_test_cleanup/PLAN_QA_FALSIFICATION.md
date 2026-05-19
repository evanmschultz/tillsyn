# PLAN_QA_FALSIFICATION — Drop 4b Test Cleanup (Round 1)

**Verdict:** FAIL — 4 CONFIRMED counterexamples, 4 POSSIBLE findings, 2 REFUTED, 0 NITs.

The plan has correct intent but two material correctness errors (F1 misrepresents existing code, F2 deflects R7.1/R7.2 in a way that defeats their purpose) plus several test-coverage gaps the planner explicitly listed in Notes but did not lock into acceptance criteria.

Severity legend: CONFIRMED = concrete counterexample with file:line evidence; POSSIBLE = real risk with credible repro path, builder should resolve; REFUTED = attack does not land; NIT = stylistic.

---

## 1. CONFIRMED Counterexamples

### F1 (CONFIRMED, HIGH) — D1.5 misrepresents existing code as fresh implementation work

The plan's D1.5 `app_service_adapter_mcp.go` section says "Implement `AppServiceAdapter.SupersedeActionItem`. Pattern: 1. Nil-guard on `a.service`. 2. Trim/validate `in.ActionItemID` non-empty. 3. Call `withMutationGuardContext` for guard-context setup. 4. Fetch the existing item via `a.service.GetActionItem` to run `assertOwnerStateGate`. 5. Check orchestrator-role ... 6. Call `a.service.SupersedeActionItem`. 7. Map errors via `mapAppError`. 8. Return `domain.ActionItem`."

**Evidence:** `AppServiceAdapter.SupersedeActionItem` ALREADY EXISTS at `internal/adapters/mcp_common/app_service_adapter_mcp.go:1051-1075`. Steps 1, 2, 3, 4, 6, 7, 8 are already implemented verbatim in the existing method. The only NEW logic in the planner's recipe is step 5 (orchestrator-role gate). The supporting `SupersedeActionItemRequest` boundary type at `mcp_surface.go:354` also already exists.

**Why this matters:** a builder reading D1.5 verbatim will either (a) try to add a function that already exists (compile error: duplicate function) or (b) silently rewrite the existing function, accidentally removing the existing STEWARD owner-state-gate (`assertOwnerStateGate` at line 1067) or the existing `Service.SupersedeActionItem` call (line 1070) and re-introducing them in altered form. The existing impl is exercised by `TestStewardIntegrationDropOrchSupersedeRejected` (`handler_steward_integration_test.go:466`) and any rewrite risks breaking that test silently.

**Recommended fix:** rewrite D1.5 to (a) acknowledge the adapter method already exists, (b) scope the change to "ADD orchestrator-role check (step 5) into the existing `SupersedeActionItem` method before the `Service.SupersedeActionItem` call," (c) ADD `SupersedeActionItem` to the `ActionItemService` interface at `mcp_surface.go:848-861` (this part of D1.5 is correct and needed), (d) keep the existing `TestStewardIntegrationDropOrchSupersedeRejected` test green as a regression guard, and (e) call out that `Service.SupersedeActionItem` already enforces "failed-only" so the new role gate is purely additive.

---

### F2 (CONFIRMED, HIGH) — D1.4 pre-authorizes R7.1/R7.2 scope deflation that defeats the refinement's purpose

D1.4 R7.1 acceptance includes: *"If reaching `applyCleanExitTransition` via the broker chain requires injecting a fake processMonitor (the real chain tries to spawn an agent process), builder must document the actual scope in the worklog — direct `applyCleanExitTransition` unit coverage via a constructed monitor is acceptable if the full chain is impractical in an in-package test."*

**Evidence:** the original REVISION_BRIEF R7.1 (lines 67-68) explicitly says: *"Gate-fail e2e test bypasses the broker chain entirely. ... **Fix:** add a real walker stub that returns an eligible item, drive the chain end-to-end, assert `metadata.outcome=failure` on the action item."* The point of R7.1 is "drive the chain end-to-end." The plan's pre-authorized deflation lets the builder reach for the easy path and recreate the exact gap R7.1 was filed against.

**Why this matters:** the broker chain IS reachable from in-package tests today — `TestDispatcherStartTriggersRunOnceOnEvent` (`subscriber_test.go:178`) and `TestHandleSubscriberEventInvokesRunOnceForEachEligibleItem` (`subscriber_test.go:445`) already drive `handleSubscriberEvent` end-to-end with stub walkers. The "real chain tries to spawn an agent process" concern is misframed — `handleSubscriberEvent` (`subscriber.go:175-193`) calls `d.RunOnce(ctx, item.ID, "")` which routes into the monitor; the monitor's spawn vs deterministic-path is configurable via test injection (the inline-fix at commit `d949f6f` already split the C1/C2 paths to make this testable). The original D5 falsification residue at `b12315c` was about timing, not unreachability.

**Recommended fix:** tighten D1.4 to REQUIRE the broker-chain reachable test as default acceptance; require explicit dev sign-off in the BUILDER_WORKLOG (not just "document in the worklog") for any direct-monitor unit-only fallback. Reference `TestDispatcherStartTriggersRunOnceOnEvent` and `TestHandleSubscriberEventInvokesRunOnceForEachEligibleItem` as the existing reachable-chain proof.

---

### F3 (CONFIRMED, MEDIUM) — D1.5 missing test case for supersede on non-failed item via MCP

D1.6 (which calls into D1.5's adapter) acceptance lists 4 test cases: orchestrator + failed = success, missing reason = invalid_request, non-orchestrator = auth_denied, unknown id = not_found. There is NO test verifying the failed-only invariant is preserved when called through the new MCP operation.

**Evidence:** `Service.SupersedeActionItem` at `service.go:1843-1845` rejects non-failed items with `ErrTransitionBlocked`. This rejection is the safety invariant the REVISION_BRIEF R8 describes ("preserves the terminal-on-failure invariant + adds a structured unstick path"). But D1.6 has no acceptance bullet verifying this passes through the MCP boundary intact. Without that test, a future refactor could silently weaken the invariant (e.g., move the failed-only check into the CLI runner instead of the service layer, breaking it for MCP callers).

**Why this matters:** the whole point of R8's "MCP supersede" design is "preserves the terminal-on-failure invariant." A regression test pinning that invariant at the new MCP layer is load-bearing — without it, the invariant only holds because the service still enforces it, not because the MCP-surface acceptance demands it.

**Recommended fix:** add a 5th acceptance bullet to D1.6: *"orchestrator session + `todo`-state item (or `in_progress`, `complete`, archived) + non-empty reason → typed error wrapping `ErrTransitionBlocked` with the 'supersede only applies to failed items' hint; lifecycle_state and outcome unchanged after rejection."*

---

### F4 (CONFIRMED, LOW) — D1.3 R6.2 deferral inconsistent with project policy

D1.3 R6.2 says: *"goroutine-leak detection deferred; requires go.uber.org/goleak (not in go.mod). t.Cleanup(d.Stop) is the structural drain guard."*

**Evidence:** `go.mod` does not contain `go.uber.org/goleak` (verified at `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/go.mod`). CLAUDE.md Go Development Rules (Dependencies section) reads: *"Dependencies: ask the dev to run `go get` / module updates in their own shell."* This is an established pattern — the planner's recommended path for a one-line `go get` addition is to ASK THE DEV, not unilaterally defer the underlying refinement.

The REVISION_BRIEF R6.2 (lines 55-56) explicitly says: *"`go.uber.org/goleak` may need adding to `go.mod` (dev runs `go get`)."* The brief itself anticipated this path. D1.3 silently chose the deferral option without surfacing the dev-prompt choice (Notes §"Open questions" does flag it as a question, but the droplet acceptance already commits to the deferred-comment path).

**Why this matters:** the refinement R6.2 is filed precisely because the broker→subscriber chain *can* leak goroutines under failure paths, and t.Cleanup(d.Stop) is a STRUCTURAL drain guard (proves Stop is called) not a goroutine-count assertion. A `goleak.VerifyNone(t)` call after Stop would catch a future regression where Stop fails to drain a goroutine. Deferring without dev confirmation accepts that gap permanently as a comment.

**Recommended fix:** before D1.3 ships, the orchestrator asks the dev to `go get go.uber.org/goleak` in the dev's shell, and D1.3 wires `goleak.VerifyNone(t)` into `TestAutoDispatch_NewDispatcherGateWiring` and `TestAutoDispatchE2EGateFailViaNewDispatcher`. If the dev declines, D1.3 reverts to the comment-only deferred path and adds an explicit "deferred per dev decision YYYY-MM-DD" annotation in the comment.

---

## 2. POSSIBLE Findings

### F5 (POSSIBLE, MEDIUM) — D1.4 R7.3 only tests the stub, not the real production resolver

D1.4 R7.3 says: *"Parameterize `stubE2ETemplateResolver`. Add field `tplByProject map[string]templates.Template` ... Add one table-driven test `TestStubE2ETemplateResolverRoutesPerProject` asserting per-project routing returns the configured per-project template."*

**Evidence:** the original R7.3 (REVISION_BRIEF lines 69-70) says: *"`stubE2ETemplateResolver.GetProjectTemplate` ignores `projectID`. Mask risk for production-resolver misrouting bugs. Fix: parameterize by project in the stub OR add a separate test that asserts per-project routing via the real `dispatcherTemplateResolver`."*

The real `dispatcherTemplateResolver` lives at `cmd/till/main.go:2704` (package `main`) — it's NOT in `internal/app/dispatcher`. A test in `dispatcher_e2e_test.go` (package `dispatcher`) cannot reach it. The planner chose the easier "parameterize the stub" half of the OR clause; the test `TestStubE2ETemplateResolverRoutesPerProject` only proves the test stub works correctly, not that the production resolver routes per project.

**Why this matters:** R7.3's "mask risk for production-resolver misrouting bugs" is the load-bearing concern. Testing the stub's per-project routing proves the STUB does what the stub claims; it does not pin the production behavior. The existing `TestDispatcherTemplateResolverAdapter` at `cmd/till/main_test.go:3363` already tests `dispatcherTemplateResolver` (positive case) — extending it to assert per-project routing (or adding `TestDispatcherTemplateResolverPerProjectRouting`) is the load-bearing test.

**Recommended fix:** keep D1.4's stub parameterization (it's harmless and lets future e2e tests route per-project), but ADD a sibling droplet or a paths-extension to D1.4 that touches `cmd/till/main_test.go` to add a per-project routing assertion on the real `dispatcherTemplateResolver`. Cross-package edit but small; alternatively, scope a new droplet D1.7 owning `cmd/till/main_test.go`.

---

### F6 (POSSIBLE, MEDIUM) — D1.1 alias semantics test the domain layer but not the SQLite persistence path

D1.1 adds the `"actionitem"` → `"action_item"` alias inside `NormalizeCommentTargetType`. The acceptance covers `mage test-pkg internal/domain`. There is no acceptance for the SQLite repo path.

**Evidence:** `internal/adapters/storage/sqlite/repo.go:3095` reads `comment.TargetType = domain.NormalizeCommentTargetType(domain.CommentTargetType(targetTypeRaw))`. With D1.1's alias in place, if SQLite has stored rows with `target_type='actionItem'` (the schema-declared form), the load-path will now normalize to `'action_item'` — DIFFERENT from what was stored. Read-after-write would observe a different value than what was inserted. The `IsValidCommentTargetType(comment.TargetType)` check on line 3096 still passes, but the in-memory `comment.TargetType` is now `'action_item'` while the DB row still says `'actionItem'`. Anything that later does an exact-string match on the round-trip value (e.g. a UNIQUE constraint, a filter, a future migration script) sees the drift.

The SQLite repo's `CreateComment` path (line 1913-1917) calls `NormalizeCommentTarget` BEFORE insert, so going forward all NEW rows will store the canonical `'action_item'`. But any EXISTING stored rows in dev or test DBs that used the old schema enum form `'actionItem'` will now round-trip differently than they were written. CLAUDE.md memory rule `feedback_no_migration_logic_pre_mvp.md`: "Dev deletes ~/.tillsyn/tillsyn.db on schema/state-vocab change." So the dev's instinct is "blow away the DB," which mitigates production risk — but the planner did not surface this gotcha or add a regression test pinning the round-trip behavior.

**Why this matters:** Drop 1.75's collapse already had to confront stale stored vocabularies; doing it again silently invites the same class of bug. A regression test in `internal/adapters/storage/sqlite/repo_test.go` exercising insert-with-camelCase, read-back-as-snake_case would pin the behavior intentionally.

**Recommended fix:** EITHER (a) extend D1.1's paths/packages to include `internal/adapters/storage/sqlite/repo_test.go` and add a regression test exercising the persistence round-trip with the new alias, OR (b) add a Notes bullet to PLAN.md acknowledging the round-trip drift and explicitly accepting it because dev wipes DB on vocab change.

---

### F7 (POSSIBLE, LOW) — D1.2 schema enum description string still lists pre-Drop-1.75 tokens

D1.2 says: *"Replace with `mcp.Enum("project", "action_item", "actionItem")` and update the description string for `target_type`."*

**Evidence:** the description string at `extended_tools.go:2243` reads `"project|branch|phase|actionItem|subtask|decision|note"`. D1.2 acceptance bullet 2 covers the enum change ("excludes stale pre-1.75 tokens") but the acceptance does NOT specify the new description string. A builder reading D1.2 verbatim could update the enum to `("project", "action_item", "actionItem")` while leaving the description string showing `"project|branch|phase|actionItem|subtask|decision|note"` — passing both acceptance bullets (mage test-pkg passes; enum contains the right values; enum excludes the old tokens). The description-vs-enum drift then mirrors the original R5 bug.

**Why this matters:** schema description is what an MCP caller reads to learn the contract. Leaving it stale recreates the exact "schema-vs-runtime drift" R5 was filed against.

**Recommended fix:** add an explicit acceptance bullet to D1.2: *"The `target_type` description string lists exactly `'project|action_item|actionItem'` and contains no stale pre-1.75 tokens (`branch`, `phase`, `subtask`, `decision`, `note`)."*

---

### F8 (POSSIBLE, LOW) — D1.5 step 5 orchestrator-role check uses a hand-rolled check instead of the existing `authSessionRoleMayGovernOthers` helper

D1.5 step 5 says: *"Check orchestrator-role: if the authenticated caller (from context) has `principal_role != \"orchestrator\"`, return `ErrAuthorizationDenied` with a clear message."*

**Evidence:** `authSessionRoleMayGovernOthers` (referenced at `app_service_adapter_mcp.go:275-277`) is the existing helper used by `ApproveAuthRequest` for an identical "orchestrator-only" gate. The plan's hand-rolled string-equality check duplicates that helper's logic instead of reusing it.

**Why this matters:** two implementations of the same role gate are prone to drift. If the role enum vocabulary expands (e.g. "orchestrator-steward" subsumes "orchestrator"), the helper gets updated and the hand-rolled check silently lags.

**Recommended fix:** D1.5 step 5 says explicitly: *"Reuse the existing `authSessionRoleMayGovernOthers(actingSession)` helper (line 275 pattern); do not hand-roll the role check."*

---

## 3. REFUTED Attacks

### Attack 2 (REFUTED) — MCP schema enum drift across tools

I searched for other MCP tools using `target_type` as a parameter. Only `till.comment` (`extended_tools.go:2243`) declares a `target_type` field. `till.attention_item` and `till.auth_request` do not use a `target_type` field — they use `scope_type` (a different vocabulary). No cross-tool schema drift.

### Attack 8 (REFUTED) — External referent for renamed test function

I searched for `TestAutoDispatchE2EGatePassViaNewDispatcher` across the entire repo. The only references are: (a) the definition + comment in `subscriber_test.go`, (b) a historical worklog at `workflow/drop_4b/D5_BUILDER_WORKLOG.md` (audit trail, durable, not a live referent), (c) PLAN.md + REVISION_BRIEF.md of this drop. No CI workflow, no README, no inline doc reference. Renaming is safe.

---

## 4. Cross-Drop and Cross-Package Verification

### Attack 9 — Package boundaries

D1.3 + D1.4 both edit `internal/app/dispatcher`. The blocked_by chain `1.4 blocked_by 1.3` is correct (1.4 builds on 1.3's new file). D1.5 edits `internal/adapters/mcp_common` (different package). D1.6 edits `internal/adapters/mcp_rpc` (different package). D1.6 is `blocked_by 1.2, 1.5` correctly. D1.1 + D1.2 edit different packages (`internal/domain` vs `internal/adapters/mcp_rpc`). No package-boundary violations.

### Attack 10 — Cross-drop with `drop_fe_1_bootstrap`

`drop_fe_1_bootstrap/` touches `ui/`, `magefile.go`, `.gitignore`, `wails.json`, root-level files relocated to `ui/`. Disjoint from this drop's `internal/domain/`, `internal/adapters/mcp_rpc/`, `internal/adapters/mcp_common/`, `internal/app/dispatcher/` paths. No interaction.

---

## 5. Hylla Feedback

N/A — action item touched non-Go files only for the PLAN.md / REVISION_BRIEF.md review portion, and Hylla is OFF for the Go-file evidence-gathering portion (per `feedback_hylla_disabled_for_now.md` 2026-05-18). Fell back to `rg` + `Read` + `Grep` throughout. No Hylla queries issued; no miss patterns to report.

---

## 6. Verdict Summary

| Attack | Verdict | Severity |
|---|---|---|
| F1 — D1.5 misrepresents existing code | CONFIRMED | HIGH |
| F2 — D1.4 R7.1/R7.2 pre-authorized scope deflation | CONFIRMED | HIGH |
| F3 — D1.5/D1.6 missing supersede-non-failed regression | CONFIRMED | MEDIUM |
| F4 — D1.3 R6.2 deferral inconsistent with policy | CONFIRMED | LOW |
| F5 — D1.4 R7.3 only tests stub not real resolver | POSSIBLE | MEDIUM |
| F6 — D1.1 alias persistence round-trip drift | POSSIBLE | MEDIUM |
| F7 — D1.2 description string drift gap | POSSIBLE | LOW |
| F8 — D1.5 step 5 reinvents existing helper | POSSIBLE | LOW |
| Attack 2 — schema drift across tools | REFUTED | — |
| Attack 8 — external referent for renamed test | REFUTED | — |
| Attack 9 — package boundaries | EXHAUSTED (clean) | — |
| Attack 10 — cross-drop interaction | EXHAUSTED (clean) | — |

**Recommended round-2 actions:**

1. Rewrite D1.5 to acknowledge the existing adapter method (F1) and reference the existing helper for role-check (F8).
2. Tighten D1.4 R7.1/R7.2 acceptance to require the broker-chain test as default (F2) and explicit dev sign-off for fallback.
3. Add the supersede-non-failed regression bullet to D1.6 (F3).
4. Surface the goleak choice to the dev BEFORE building D1.3 (F4).
5. Extend D1.4 R7.3 with a real-resolver test in `cmd/till` (F5), either by extending D1.4 paths or by adding a D1.7.
6. Surface the alias persistence round-trip risk in D1.1 Notes (F6).
7. Tighten D1.2 acceptance with explicit description-string check (F7).
