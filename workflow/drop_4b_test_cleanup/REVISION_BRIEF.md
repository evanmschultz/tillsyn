# Drop 4b Test Cleanup — Deferred R5/R6/R7/R8 Absorption (Revision Brief)

**Status:** revision-brief authoring 2026-05-18 (orch-direct from `project_drop_4b_refinements_raised.md`).
**Drop scope (LOCKED):** land four deferred refinements from Drop 4b. Two test-gap groups (R6, R7) + two MCP coordination-surface fixes (R5, R8). R1/R2/R3/R4 stay parked for a separate template-validation hardening drop.
**Out of scope:** template-validation hardening (R1/R2/R3/R4); D5 design rework; any non-test changes to `internal/app/dispatcher/` beyond what R6/R7 require.

## 1. Hard Prerequisites

- HEAD `2124d2c` on `main`. Git clean.
- Drop 4b cascade fully closed (it is).
- Hylla back on per dev 2026-05-18 — planners + builders + QA may use Hylla MCP normally for Go code understanding.

## 2. Goal

Absorb four deferred refinements raised during Drop 4b plan-QA + build-QA rounds. Each refinement has a documented fix path; this drop wires them in and adds the tests that pin the new behavior. No new feature work, no architecture changes.

## 3. Pre-MVP Rules In Force

- Filesystem-MD only for drop coordination. No per-droplet Tillsyn action items.
- Tillsyn-flow output style + Section 0 SEMI-FORMAL REASONING in every subagent response.
- Single-line conventional commits. ≤72 chars.
- NEVER raw `go test` / `go build` / `go vet` / `mage install`. Always `mage <target>`.
- Hylla MCP is back on; use it as primary code-understanding source. Record any miss in `## Hylla Feedback` per builder worklog.
- No closeout MD rollups pre-dogfood.

## 4. Refinement Breakdowns

### R5 — `till.comment` target_type schema-vs-runtime drift

**Surface:** `till.comment` MCP tool.

**Observed:** Both `actionItem` (post-Drop-1.75 cascade-correct vocab) and `branch` (pre-Drop-1.75 vocab) return `internal_error: create comment: invalid target type`. Schema enum claims to accept `project|branch|phase|actionItem|subtask|decision|note`. Runtime accepts only some subset (TBD by investigation).

**Impact:** Orchestrators cannot post per-action-item comments via MCP. Built-in coordination surface broken on this lane.

**Workaround in place:** Stash closing content in `completion_contract.completion_notes` on the parent build action item.

**Recommended fix path (planner refines):**
1. Investigate `internal/app/comments.go` (or wherever `CreateComment` lives) — find the actual accepted target-type vocab in the validation code.
2. Reconcile: either (a) update server-side accepted-vocab to match the schema enum, or (b) shrink the schema enum to match runtime + add migration for old vocab tokens. Default choice: option (a) — the schema enum is the documented contract; runtime should honor it.
3. Add a regression test that creates a comment with `target_type=actionItem` against a live action item and asserts success.
4. Add a regression test for each schema-enum value the runtime claims to accept.

**Severity:** Medium — coordination surface degradation, not a hard cascade-blocker.

### R6 — D5 e2e scope-clarity findings (QA proof, low-severity)

**Source:** `workflow/drop_4b/D5_BUILDER_QA_PROOF.md` (or whatever the exact filename — planner verifies).

**Findings:**

- **6.1** Test `TestAutoDispatchE2EGatePassViaNewDispatcher` (in `internal/app/dispatcher/subscriber_test.go`) implies fuller E2E than the trace actually covers — `d.gates.Run(...)` is invoked directly at line ~600, bypassing the production subscriber → walker → RunOnce → monitor pipeline. The broker → subscriber half is only observed as "no panic / no deadlock." Test prose at lines 586-589 honestly discloses this.
  - **Fix options:** (a) rename to `TestAutoDispatch_NewDispatcherGateWiring` to match actual scope; (b) enrich walker stub so `EligibleForPromotion` returns a candidate and let RunOnce drive into monitor's gate-runner invocation site. Planner picks based on size — (a) is smaller; (b) covers more.

- **6.2** No explicit goroutine-leak assertion. `t.Cleanup` calls `d.Stop` and `-race`/`mage ci` did not flag leaks across 3379 tests, but no `runtime.NumGoroutine()` bracket. Optional `goleak.VerifyNone(t)` or `assertNoGoroutineLeak(t, baseline)` helper would tighten.
  - **Fix:** add the helper to the relevant tests. `go.uber.org/goleak` may need adding to `go.mod` (dev runs `go get`).

- **6.3** Action-item state-transition verification (`todo → in_progress → complete`) not asserted. The check-list bullet "state transitions verified" maps only to lifecycle pin `lister.calls == 1`.
  - **Fix:** clarify D5 spec — does "state transitions" mean dispatcher lifecycle (Start/Stop) or action-item state? Extend pass test to drive real RunOnce path that promotes an item AND assert state moves through `todo → in_progress → complete`.

**Severity:** Low for all three. PASS verdict on D5 stands.

### R7 — D5 e2e falsification residue (deferred from b12315c)

**Source:** `workflow/drop_4b/D5_FIX_QA_FALSIFICATION.md` (or filename equivalent). `b12315c` addressed the timing-flake (R7.0 / FAL 2.3 + 2.7) inline via `waitForSubscriberDelivery` helper. The following deferred:

- **7.1 (test-gap, low):** Gate-fail e2e test bypasses the broker chain entirely. `TestAutoDispatchE2EGateFailViaNewDispatcher` only verifies `d.gates.Run` returns `ErrGateNotRegistered` in isolation; it does NOT exercise `subscriber → handleSubscriberEvent → RunOnce → monitor → applyCleanExitTransition → gateRunner.Run → GateStatusFailed → transitionToFailed`. **Fix:** add a real walker stub that returns an eligible item, drive the chain end-to-end, assert `metadata.outcome=failure` on the action item.
- **7.2 (test-gap, low):** C1/C2/N2 inline fix at commit `d949f6f` is unit-tested in `monitor_test.go` but NOT reached via the integration chain. **Fix:** add an integration test that drives `applyCleanExitTransition` via the subscriber chain, covering at least the Skipped vs Failed disambiguation (C1) and pre-loop ctx-cancel (C2) paths.
- **7.3 (NIT):** `stubE2ETemplateResolver.GetProjectTemplate` ignores `projectID`. Mask risk for production-resolver misrouting bugs. **Fix:** parameterize by project in the stub OR add a separate test that asserts per-project routing via the real `dispatcherTemplateResolver`.
- **7.4 (NIT):** File naming inconsistency — `subscriber_test.go` now contains both subscriber unit tests AND D5 e2e tests. **Fix:** rename to `dispatcher_e2e_test.go` for the e2e portion OR split into two files.

**Severity:** All low. PASS verdict on D5 fix stands.

### R8 — `outcome=superseded` auto-fails to terminal `failed` via MCP

**Surface:** `till.action_item operation=update` with `metadata.outcome: "superseded"`.

**Observed:** Setting `outcome=superseded` on a `todo`-state action item automatically moves it to `failed` (terminal). Subsequent `move_state` to `complete` rejects with `cannot transition from terminal state "failed" without override auth`. Only the human-only CLI `till action_item supersede <id> --reason "..."` can unstick `failed → complete`.

**Impact:** Orchestrators cannot fully resolve auto-template-created duplicate twins via MCP — must dev-route every duplication event through CLI. Brittle when duplicates accumulate.

**Recommended fix path (planner refines):**

1. Expose an orch-level supersede operation on the action_item MCP surface, gated by orchestrator role + lease-subtree scope, mirroring the orch-self-approval pattern from Drop 4a Wave 3. New `mcp__tillsyn__till_action_item operation=supersede` accepts `action_item_id` + `reason`.
2. Document the auto-fail-on-superseded transition in the existing schema field description (`metadata.outcome` enum + its observed side effects).
3. Tests:
   - MCP supersede with valid orchestrator session + in-subtree action item: succeeds, item moves to `complete` from `failed`.
   - MCP supersede without orchestrator role: rejected.
   - MCP supersede on out-of-subtree action item: rejected.

**Alternative:** make `outcome=superseded` NOT auto-fail (leave state malleable until explicit `move_state`). Planner evaluates: which costs less complexity? The orch-MCP supersede operation is the safer choice — preserves the terminal-on-failure invariant + adds a structured unstick path.

**Severity:** Medium — recurring blocker, dev-CLI workaround exists.

## 5. Acceptance Criteria (Drop-Level — Planner Decomposes Per Droplet)

- **A1.** `mage ci` green.
- **A2.** R5 — `till.comment` with `target_type=actionItem` succeeds against a real action item. Regression test pins this + the other schema-enum values that should work.
- **A3.** R6 — D5 e2e tests renamed OR enriched per 6.1; goleak helper added per 6.2; state-transition assertions per 6.3.
- **A4.** R7 — Gate-fail e2e test exercises the full broker chain (7.1); applyCleanExitTransition reached via integration chain (7.2); stub project-routing parameterized or asserted via real resolver (7.3); file rename or split (7.4).
- **A5.** R8 — Orch-level supersede on action_item MCP surface; tests for role-gating, subtree-gating, and the happy path.
- **A6.** No regressions in existing Drop 4b tests; the 3379-test baseline holds or grows.

## 6. Out Of Scope (Hard)

- R1/R2/R3/R4 from Drop 4b (template-validation hardening — separate drop).
- Any rewrite of the D5 implementation (only test absorbtion + naming + assertions).
- Any change to the auth-revoke / git-status / auto-promotion behavior (Drop 4b shipped these; this drop touches only their tests via R6/R7).
- New gate kinds.
- New MCP surfaces beyond R5 and R8.

## 7. Open Questions For The Planner

- **Q1 — R5 fix direction.** Default option (a): expand runtime accepted-vocab to match schema. Confirm-or-deviate.
- **Q2 — R6 fix option for 6.1.** Rename (smaller) vs enrich walker stub (more coverage). Planner picks based on coverage gain vs LOC budget.
- **Q3 — R8 MCP supersede shape.** New `operation=supersede` on `till_action_item`, or new field on `operation=update`? Default: new operation — clearer audit, less coupling to update's general path.
- **Q4 — R7.4 file rename vs split.** Planner picks based on what other tests already live in `subscriber_test.go`.

## 8. Approximate Size

- R5: ~50-100 LOC (investigation + reconcile + tests).
- R6: ~80-150 LOC (test additions + goleak wiring + state-transition assertions).
- R7: ~150-250 LOC (4 test additions + 1 file rename or split).
- R8: ~150-250 LOC (new MCP operation + adapter + service-layer wiring + tests).

Total: ~430-750 LOC across ~4-7 droplets. Most LOC is test code.

## 9. Cross-References

- Memory `project_drop_4b_refinements_raised.md` — original refinement entries (R5/R6/R7/R8).
- `workflow/drop_4b/D5_BUILDER_QA_PROOF.md` — R6 source.
- `workflow/drop_4b/D5_FIX_QA_FALSIFICATION.md` — R7 source.
- `workflow/drop_4b/PLAN.md` — Drop 4b plan (closed).
- `workflow/example/drops/WORKFLOW.md` — per-drop lifecycle.
