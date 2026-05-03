# UNIT C — PLAN QA FALSIFICATION — ROUND 1

**Reviewer:** go-qa-falsification-agent (subagent)
**Round:** 1
**Verdict:** **FAIL** — three CONFIRMED counterexamples, one CONFIRMED scope hole, multiple HIGH-severity nits
**Date:** 2026-05-02

## Verdict Summary

The 6-droplet decomposition shape is sensible and matches the Drop 2.3 / 2.4 / 2.5 / 2.6 precedents. But Droplet 3.C.3's auth-gate design has **three confirmed bypasses** and **one confirmed external-library blocker** that no droplet addresses, plus several smaller scope/wiring gaps. The plan must be revised before any builder spawns.

---

## C1 — CONFIRMED (HIGH): `UpdateActionItem` becomes a state-lock bypass once `Owner` is updatable via 3.C.5

**The attack.** Plan 3.C.3 § Acceptance test 5 says: *"`UpdateActionItem` (description/details/metadata) on `Owner = "STEWARD"` with `AuthPrincipalType = "agent"` → SUCCEEDS. The gate is state-only; drop-orchs retain content-edit rights."* Plan 3.C.5 then adds `Owner` and `DropNumber` to `CreateActionItemRequest`, `UpdateActionItemRequest`, and `internal/adapters/server/common/app_service_adapter_mcp.go` `UpdateActionItem` plumbing (around `:661`).

That combination makes `Owner` itself an agent-mutable column. Concrete reproduction:

```
# Drop-orch session, AuthPrincipalType = "agent"
till.action_item(operation=update, action_item_id=<DROP_3_HYLLA_FINDINGS>, owner="")
# → succeeds per 3.C.3 acceptance #5 (UpdateActionItem stays open)
# → 3.C.5 plumbs owner through, no validation blocks empty value (3.C.1: empty owner is the dominant case)

till.action_item(operation=move_state, action_item_id=<DROP_3_HYLLA_FINDINGS>, state=complete)
# → succeeds, gate sees Owner == "" not "STEWARD"
```

The gate is now sidestepped in two MCP calls. Test 1 / test 4 in Droplet 3.C.3 only verify the gate fires when `Owner == "STEWARD"` at gate-fire time. They don't verify the gate is unbypassable.

**Why the plan misses this.** Droplet 3.C.3 is authored before the Droplet 3.C.5 MCP plumbing exists, so the planner reasoned about gate semantics against the pre-3.C.5 world (where `Owner` couldn't be sent via MCP at all). The plan never returns to 3.C.3 after 3.C.5 widens the input surface.

**Evidence.**
- `UpdateActionItem` adapter at `internal/adapters/server/common/app_service_adapter_mcp.go:692-725`: no field-level write gate today — every named input field flows through.
- Plan 3.C.5 line 197: "thread `Owner` + `DropNumber` through `CreateActionItem` (around `:620`, mirror Drop 2.5's `Role` plumbing) and `UpdateActionItem` (around `:661`)."
- Plan 3.C.3 acceptance test 5 (line 120) explicitly green-lights `UpdateActionItem` for agents on STEWARD-owned items.

**Fix (orch routes back).** Drop 3.C.3 (or a new sub-droplet bundled with 3.C.5) must add an `UpdateActionItem` field-level guard: when `existing.Owner == "STEWARD"` and `caller.AuthPrincipalType != "steward"`, reject any `UpdateActionItemRequest` whose `Owner` or `DropNumber` differ from the existing values. Otherwise `Owner` becomes a self-clearing field and the entire gate is decorative.

Most-damaging-counterexample candidate.

---

## C2 — CONFIRMED (HIGH): `ReparentActionItem` is unguarded — drop-orchs can extract STEWARD-owned children from STEWARD persistent parents

**The attack.** Plan 3.C.3 § Architectural Decisions #4: *"Drop-orchs retain `UpdateActionItem`, `CreateActionItem`, `ReparentActionItem` rights on STEWARD-owned items."* Plan never re-evaluates whether `Reparent` is safe.

```
# Drop-orch session, AuthPrincipalType = "agent"
till.action_item(operation=reparent,
                 action_item_id=<DROP_3_HYLLA_FINDINGS>,
                 parent_id=<some-node-in-DROP_3-tree>)
# → succeeds; STEWARD's level_2 finding is now reparented out of HYLLA_FINDINGS
```

Consequences:

1. **Auto-generator parent-id-lookup breaks**: Droplet 3.C.4's `parent_id_lookup = "owner=STEWARD,title=DROP_${N}_HYLLA_FINDINGS-parent"` resolves to the persistent parent `HYLLA_FINDINGS`, but if a prior drop's level_2 child has been reparented under it from elsewhere or the existing one extracted, ordering / counting / `blocked_by` calculation drifts.
2. **STEWARD's post-merge collation flow** depends on level_2 findings being children of the canonical persistent parents (`STEWARD_ORCH_PROMPT.md` § 1.2: "Five level_2 findings drops — one under each non-`DISCUSSIONS` persistent parent above"). Reparenting silently breaks that contract — STEWARD's audit listing under the persistent parent will miss the moved item.
3. **The refinements-gate `blocked_by` resolver** (3.C.4) builds its block list from `WHERE drop_number = N`. A reparented item still has `drop_number = N` (reparent doesn't change that), so the gate still blocks on it — but the item is no longer reachable via the canonical parent path. Closing the gate now requires STEWARD to find the orphaned item out-of-band.

**Evidence.** `internal/app/service.go:1106-1156` (`ReparentActionItem`) — runs only `enforceMutationGuardAcrossScopes` capability check, no Owner-aware reject. The adapter at `app_service_adapter_mcp.go:810-823` does not invoke any STEWARD gate.

**Fix.** Add `assertOwnerStateGate`-equivalent to `ReparentActionItem` adapter path — OR ban Owner change via reparent by checking the existing item's Owner before allowing reparent when AuthPrincipalType != "steward". The plan's "ReparentActionItem stays open" decision needs counterexample-driven revision.

---

## C3 — CONFIRMED (HIGH): `principal_type: steward` collides with the upstream `autent` external library's closed enum

**The attack.** Plan 3.C.3 line 108: *"extend `normalizeAuthRequestPrincipalType` at `:596-608` ... to add a fifth case: `"steward" → "steward", nil`"*. Plan never addresses the autent boundary.

`autent` (`/Users/evanschultz/go/pkg/mod/github.com/evanmschultz/autent@v0.1.1/domain/principal.go`) declares the closed `validPrincipalTypes = []PrincipalType{user, agent, service}` and `IsValidPrincipalType` rejects anything else with `ErrInvalidPrincipalType`. The autent library is an external module — tillsyn cannot extend it from inside this repo.

The call path that breaks:

```
ApproveAuthRequest
  → ... (issues autent session via the AuthBackend)
  → autentauth.Service.IssueSession (internal/adapters/auth/autentauth/service.go:183)
    → ensurePrincipal(ctx, principalID, principalType=NormalizePrincipalType("steward"), ...)
      → service.RegisterPrincipal(...)
        → autentdomain.NewPrincipal(...)
          → IsValidPrincipalType(steward) == false
          → returns ErrInvalidPrincipalType
```

So an `auth_request` with `principal_type=steward` would pass tillsyn's `NewAuthRequest` validation but fail at session-issuance time when autent rejects the principal type. The plan never names this boundary nor specifies the mapping.

**Evidence.**
- `/Users/evanschultz/go/pkg/mod/github.com/evanmschultz/autent@v0.1.1/domain/principal.go:22-26, 56-58, 67-69`.
- `internal/adapters/auth/autentauth/service.go:191`: `ensurePrincipal(...principalType...)` — direct passthrough.
- `internal/adapters/auth/autentauth/service.go:803-812`: `principalTypeToActorType` maps unknown → `domain.ActorTypeUser`. So even if a steward session somehow existed, the validation path would resolve `AuthenticatedCaller.PrincipalType = ActorTypeUser` — not `"steward"`.

**Fix.** Plan must add a sub-droplet (or extend 3.C.3) covering the autent-boundary mapping. Two viable shapes:

1. **Boundary-mapped:** at the autentauth layer, map `principal_type=steward` → `autentdomain.PrincipalTypeAgent` for autent's purposes; keep `steward` only in tillsyn's `auth_requests` table + propagate via the new `AuthSession.PrincipalType` field + new `AuthenticatedCaller.AuthPrincipalType` field. Document that `steward` is a tillsyn-internal axis distinct from autent's principal_type.
2. **Vendored fork:** patch the vendored autent library to add `steward`. Pre-MVP, probably overkill.

The plan implicitly assumes (1) but never spells it out. A builder reading 3.C.3 will write `IssueSession(...PrincipalType: "steward"...)` and hit autent rejection at runtime.

---

## C4 — CONFIRMED (MEDIUM): `MoveActionItem` adapter doesn't pre-fetch the action item; Droplet 3.C.3 helper-design is wrong for that path

**The attack.** Plan 3.C.3 line 110: *"the helper consults the loaded action item's `Owner` field, which means the helper must run AFTER the action item is fetched but BEFORE the move SQL fires; on `MoveActionItemState` the existing `a.service.GetActionItem` call at `:760` is the natural fetch point — call the gate immediately after."*

Confirmed at `app_service_adapter_mcp.go:760`: `MoveActionItemState` does pre-fetch. **But `MoveActionItem` at `:728-741` does NOT pre-fetch** — it directly calls `a.service.MoveActionItem(ctx, ID, columnID, position)`. The fetch happens inside the service layer at `internal/app/service.go:611`. So the plan's "natural fetch point" is `MoveActionItemState`-only. For `MoveActionItem` the helper has nothing to gate on without adding a new pre-fetch.

The plan's "BEFORE the move SQL fires" requirement forces ONE of:

1. **Add a pre-fetch in adapter `MoveActionItem`** — N+1 latency for every column move (every state change goes through column moves).
2. **Push the gate down into `service.MoveActionItem`** — cleaner architecturally, but the gate needs `AuthPrincipalType` which lives on `AuthenticatedCaller` (a context value), not on the service input.
3. **Skip gating `MoveActionItem`** — but the plan's acceptance test 4 requires `MoveActionItem` to reject. Inconsistent.

**Evidence.** Read above; service layer fetches at `internal/app/service.go:611`, adapter does not.

**Fix.** Plan must specify which option. Option (2) is the architecturally-correct answer (gate at the service layer, where every callsite that mutates state passes through; the HTTP path also benefits) — but that's a different droplet shape than what 3.C.3 describes. Option (1) keeps the adapter-only gate but adds a pre-fetch.

---

## C5 — CONFIRMED (HIGH): Droplet 3.C.4 + Droplet 3.C.6 both require STEWARD persistent parents to exist; neither droplet's plan covers seeding

**The attack vector required by the spawn brief.** "Template auto-gen race. When `DROP_N_ORCH` creates a level_1 numbered drop, the auto-gen rule fires and creates 5 STEWARD level_2 items. What if the persistent STEWARD parents don't exist yet?"

**Confirmed.** Droplet 3.C.4 line 175: *"Numbered drop `N=3` creation when `Owner = "STEWARD"` is missing on a persistent parent → rule-engine returns a clear error (the auto-generator fails fast; STEWARD persistent parents must exist before any numbered drop spawns)."* — the plan flags the failure but does not name who creates the STEWARD persistent parents in the first place.

Per `STEWARD_ORCH_PROMPT.md` § 5.0, the dev's STEWARD orch creates them via `till.action_item(operation=create)` at first-session time. But:

1. **No droplet in Unit C creates them.** Droplet 3.C.6 (integration tests) line 224 says: *"Setup: project root + 5 STEWARD persistent parents seeded in the test fixture."* Test fixtures are seeded by test setup code that does not exist yet — the plan does not specify which file owns that setup.
2. **No droplet documents the boot-time invariant.** When the auto-generator fires for the FIRST numbered drop on a fresh project, the STEWARD parents must already exist. There is no Tillsyn-side mechanism in Unit C that asserts this — the failure mode is "rule-engine returns a clear error" and then... what? The drop-orch is already mid-creation of `DROP_N`. Atomicity of the rule fan-out vs. the parent creation is not addressed.
3. **Cold-start lockout.** On a fresh `~/.tillsyn/tillsyn.db` (the pre-MVP rule), the dev fresh-DBs and then must manually re-seed the 5 STEWARD persistent parents before any `DROP_N` can be created. The plan does not surface this dev-experience cliff.

**Evidence.**
- `STEWARD_ORCH_PROMPT.md:135-152` — STEWARD seeds the parents, no Tillsyn-side enforcement.
- `STEWARD_ORCH_PROMPT.md:135-136` notes the *hard sequencing dependency*: "this step must close before any numbered-drop orchestrator ... spins up. Drop-orchs create level_2 findings drops as children of the five persistent parents below; those children cannot exist until the parents do."
- Droplet 3.C.4 makes this hard sequencing dependency a runtime failure mode but doesn't elevate it to a unit-level acceptance criterion.

**Fix.** Either (a) add a sub-droplet that seeds the 5 STEWARD persistent parents on project creation (template-driven, since this IS what Drop 3 is about), OR (b) explicitly document this as a STEWARD-runtime invariant that drop-orchs must check before creating numbered drops, with a Tillsyn-side reject when the parents are missing — and add a 3.C.6 integration test that exercises the missing-parent failure end-to-end (not just at the rule-engine layer).

---

## C6 — CONFIRMED (MEDIUM): Refinements-gate `blocked_by` dynamic-at-create + manual-update has a known drop-orch-forgets-to-update failure mode

**Spawn-brief vector 6.** "Construct a scenario where drop-orch forgets to update the gate's `blocked_by` mid-drop."

**Confirmed.** Plan 3.C.4 acceptance: *"Numbered drop `N=3` creation → 5 level_2 findings created under correct STEWARD parents + refinements-gate created with correct `blocked_by` covering every Drop 3 item + the 5 findings just spawned."* Plan § Architectural Decisions #5: *"Refinements-gate `blocked_by` is dynamic at rule-fire time, NOT static at create. ... Items that come into existence AFTER the gate is created (e.g., a mid-drop refinement plan-item) are NOT auto-added to the gate's blocked_by — drop-orch must manually update the gate's blocked_by list."*

**Concrete failure scenario:**

```
# Hour 0: DROP_3_ORCH creates DROP_3 (level_1).
# Auto-generator fires. Refinements-gate created with blocked_by = [
#   DROP_3_PLAN, DROP_3_HYLLA_FINDINGS, DROP_3_LEDGER_ENTRY,
#   DROP_3_WIKI_CHANGELOG_ENTRY, DROP_3_REFINEMENTS_RAISED, DROP_3_HYLLA_REFINEMENTS_RAISED
# ]
# Hour 5: DROP_3_ORCH discovers a missing dependency, creates DROP_3_DEPS_FIX (level_2 inside DROP_3 tree, drop_number=3).
# DROP_3_ORCH does NOT update the refinements-gate blocked_by (forgot, or never got the rule).
# Hour 10: All originally-listed children close. Refinements-gate becomes unblocked.
# STEWARD closes the gate.
# But DROP_3_DEPS_FIX is still in_progress.
# DROP_3_ORCH attempts to close DROP_3 (level_1) — succeeds because parent-blocks-on-incomplete-child sees the gate as closed.
# (Gate's blocked_by didn't include DROP_3_DEPS_FIX.)
# DROP_3 closes prematurely with an in-progress child.
```

This is exactly the failure the plan's architectural decision warned about. The plan proposes manual-update as the mitigation but provides no Tillsyn-side enforcement for when the manual-update doesn't happen.

**Evidence.**
- `STEWARD_ORCH_PROMPT.md:1.2` line: "the refinements-gate item ... blocks level_1 drop N's closure".
- `main/PLAN.md` § 19.3 (per Unit C plan ref) — STEWARD gate is the load-bearing closure block.
- Drop 1's always-on `parent-blocks-on-failed-child` does NOT cover this case because the orphaned child is `in_progress`, not `failed` — and the gate is `closed`.

**Fix.** This is not a Unit C blocker per se — the architectural-decision's tradeoff was deliberate. But Unit C's tests (3.C.6) MUST include the regression case: "drop_orch creates a mid-drop child after the gate, gate closes without the new child blocking, level_1 closes prematurely" — and assert whether this is acceptable behavior or a bug. Either accept the manual-update contract explicitly with a runtime warning, or upgrade to rule-fire-on-every-new-drop-N-child. Currently the plan defers without test coverage.

---

## N1 — NIT (MEDIUM): `MoveActionItem` (column-only) gate semantics — what if column move doesn't change lifecycle state?

**Spawn-brief vector 3.** *"What if a column move doesn't change lifecycle state — does the gate fire spuriously?"*

Plan 3.C.3 acceptance test 4: *"`MoveActionItem` (column-level move, not state-level) on `Owner = "STEWARD"` with `AuthPrincipalType = "agent"` → must also reject (the state-lock applies to ANY `LifecycleState` transition, and `MoveActionItem` is the column-move path that can change state when the destination column maps to a different lifecycle)."*

But `internal/app/service.go:627-630`:

```go
toState := lifecycleStateForColumnID(columns, toColumnID)
if toState == "" {
    toState = fromState
}
```

A column-only move where `toState == fromState` (e.g., reordering position within the same lifecycle column, or moving between two columns mapped to the same state) is purely cosmetic. **The plan's gate fires on `Owner == "STEWARD"` regardless of state delta** — so drop-orchs can no longer reorder STEWARD items even when the move is state-neutral. That's stricter than § 19.3 bullet 7 which says drop-orchs "cannot move STEWARD items through state."

**Fix.** Either (a) accept the stricter "no column moves at all" semantics and document that or (b) gate on `fromState != toState` so position-only moves stay open. Option (a) is simpler / safer; option (b) preserves drop-orch ergonomics. Plan must pick one, not leave both interpretations in play.

---

## N2 — NIT (LOW): STEWARD locking itself out — no scenario found

**Spawn-brief vector 2.** *"Verify the gate logic doesn't accidentally reject STEWARD's own state transitions."*

**Verified clean.** Gate is: `Owner == "STEWARD" AND AuthPrincipalType != "steward"`. STEWARD sessions claim with `AuthPrincipalType = "steward"`, so the predicate is false on STEWARD's own moves. No counterexample found.

One residual question: what if STEWARD spawns a non-orch subagent (per `STEWARD_ORCH_PROMPT.md` § 8.1) — a `STEWARD_PLANNER_<TOPIC>` or similar — that needs to update content on STEWARD-owned items? Per the spec, those subagents claim auth as `principal_type: agent` (not `steward`). They'd hit the gate when moving state. Plan does not address whether STEWARD-spawned subagents should claim as `steward` or `agent`. This is a STEWARD-prompt-spec question, not a Unit C planning bug, but the boundary should be flagged for the dev.

---

## N3 — NIT (LOW): `AuthPrincipalType` field name collision — verified orthogonal but adds future-confusion risk

**Spawn-brief vector 7.** *"Verify orthogonality."*

**Verified.** Plan 3.C.3 § Architectural Decisions #3 (line 245) correctly distinguishes:

- `domain.AuthenticatedCaller.PrincipalType` (existing) is `ActorType` — `user|agent|system`, sourced from autent's principal type after `principalTypeToActorType` mapping (line 803-812).
- `domain.AuthenticatedCaller.AuthPrincipalType` (new) is the auth-request principal_type string — `user|agent|service|steward`, sourced directly from the issued session's `PrincipalType` field.

The names are confusable. Recommendation: rename the new field to something like `AuthRequestPrincipalType` or `SessionPrincipalClass` to make the distinction obvious in code review. Not blocking, but every future reader of `caller.PrincipalType` vs `caller.AuthPrincipalType` will need a mental check. Spec the rename now or accept the confusion forever.

Also: the existing `domain.ActorType` enum is `{user, agent, system}` (per `internal/domain/auth_request.go:415-420` validation). The plan's auth-request principal_type is `{user, agent, service, steward}`. Note the `system` vs `service` mismatch — that's pre-existing tech debt in the codebase, not Unit C's bug, but it crosses Unit C's terrain. Worth flagging for the dev.

---

## N4 — NIT (LOW): `Owner` + `DropNumber` field placement decision is reasonable but the rollback claim understates cost

**Spawn-brief vector 5.** *"What's the rollback cost if dev wants metadata JSON instead?"*

Plan 3.C.1 line 79: *"If dev pushes back to metadata JSON, only 3.C.1 + 3.C.2 change shape; the auth gate (3.C.3) reads only `Owner`, not `DropNumber`."*

**Counterexample.** That claim is too narrow. Rollback to metadata JSON would also affect:

1. **Droplet 3.C.4** auto-generator's `parent_id_lookup = "owner=STEWARD,title=..."` and `blocked_by_lookup = "every_other_drop_${N}_item"` resolvers — both would need to JSON-decode every action_item row instead of using indexed columns.
2. **Droplet 3.C.5** MCP plumbing: `Owner string` and `DropNumber int` would move from request struct fields to nested `metadata` keys, changing every test's input shape.
3. **The new index** `idx_action_items_project_owner_drop_number` (3.C.2) wouldn't exist — its query patterns become full table scans (per project) or require JSON-extract indexes (SQLite supports those, but it's a different schema shape).
4. **`AuthenticatedCaller.AuthPrincipalType`** is unaffected (auth axis, not action_item).

The actual rollback touches at minimum droplets 3.C.1, 3.C.2, 3.C.4, 3.C.5 (and any tests in 3.C.6 that key off the field names). The plan's "only 3.C.1 + 3.C.2 change shape" claim is wrong.

**Fix.** Update the rollback statement to "first-class is recommended; if dev pushes back to metadata JSON, every consumer except the auth gate changes shape — non-trivial backout" or run the dev decision before any droplet starts.

---

## N5 — NIT (LOW): SQLite index design mismatch with auto-generator queries

**Plan 3.C.2** proposes `CREATE INDEX idx_action_items_project_owner_drop_number ON action_items(project_id, owner, drop_number)`. **Plan 3.C.4** describes two query shapes:

1. `WHERE project_id = ? AND owner = ? AND title = ?` — find STEWARD persistent parent.
2. `WHERE project_id = ? AND drop_number = ?` — find every drop N item (regardless of owner; the refinements-gate's blocked_by spans the whole drop's children).

For query (1): index covers `project_id, owner` prefix; the `title` predicate is then a non-indexed filter on the remaining matches. For most dev workflows (5 STEWARD parents per project) this is fine — but the plan claims "the new index covers this query" which is only partially true.

For query (2): the index's `(project_id, owner, drop_number)` ordering means filtering by `(project_id, drop_number)` requires either a full prefix scan over all owners (today: empty + STEWARD = 2 partitions; trivial) or a different index. Acceptable now; not future-proof.

**Fix.** Either swap the index column order to `(project_id, drop_number, owner)` so query (2) is a clean prefix scan and query (1) drops to a slightly less-clean scan, OR add a second index on `(project_id, drop_number)` — both would be defensible. The plan should pick deliberately, not by accident.

---

## C7 — CONFIRMED (MEDIUM): 6-droplet decomposition has a gap on PLAN.md § 19.3 bullet 9 ("Template-defined STEWARD-owned drop kind(s)")

**Spawn-brief vector 8.** *"Find anything in PLAN.md § 19.3 + § 15.7 that isn't covered."*

The Unit C scope (line 13) lists:

> "Three concrete primitives (`metadata.owner` first-class field, `metadata.drop_number` first-class field, `principal_type: steward` enum value), one auth-layer enforcement path ..., and one template-driven auto-generation hook"

But Drop 3's `PLAN.md` lists nine bullet items. § 19.3 bullet **9** ("Template-defined STEWARD-owned drop kind(s). Templates allow marking specific kinds as STEWARD-owned. Pairs with the `principal_type: steward` gate.") is **not in Unit C's four-bullet enumeration** at line 17.

The Unit C plan claims: *"The four PLAN.md § 19.3 bullets Unit C covers"* and lists 1-4 — but the actual § 19.3 bullets it covers are 7, 8, 9, **and `metadata.owner`** is hinted in § 19.3 implicitly via the STEWARD discussion. § 19.3 bullet 9 (template marks specific KINDS as STEWARD-owned, e.g. `closeout` kind always lands `Owner = "STEWARD"`) is **missing from the droplet decomposition**.

The auto-generator (Droplet 3.C.4) writes `Owner = "STEWARD"` only for the 6 specific drops in the template's `[child_rules]` body. It doesn't address "template kind X is always STEWARD-owned." The closed 12-kind enum (Drop 1.75) includes `closeout`, `refinement`, `commit`, `human-verify` — Drop 3 § 19.3 bullet 9 implies templates can declare "kind=closeout always Owner=STEWARD on creation." None of the 6 droplets covers that mechanism.

**Evidence.**
- Drop 3 `PLAN.md:23` bullet 9: *"Template-defined STEWARD-owned drop kind(s). Templates allow marking specific kinds as STEWARD-owned."*
- Unit C plan line 17: lists 4 bullets that are actually a re-enumeration of bullets 7-8 + their internals; bullet 9's "kind-level STEWARD-owned marking" is absent.

**Fix.** Either (a) add Droplet 3.C.7 (or fold into 3.C.4) covering kind-level Owner-default — when template defines `kind.steward_owned = true`, every action_item created with that kind gets `Owner = "STEWARD"` regardless of who creates it. OR (b) explicitly punt bullet 9 to Drop 3+ / Unit B revision and document the carve-out.

---

## C8 — CONFIRMED (LOW): No droplet covers the Drop 1 `failed` state interaction with STEWARD gate

**Plan dependency.** Drop 1 (per Drop 1's PLAN.md) lands always-on `failed` lifecycle state. The Unit C plan never says what happens when:

1. STEWARD-owned item is in state `done` and the always-on terminal-state guard rejects transitions out of it (`internal/app/service.go:646-648`).
2. A `failed` STEWARD-owned item — does the gate fire on the supersede path (post-Drop-1 CLI `till action_item supersede`)? The plan never discusses supersede.

Pre-Drop-1 this is not a problem (no `failed` state). Post-Drop-1, the supersede path becomes a state-transition route the plan doesn't cover.

**Fix.** Add a sub-acceptance to 3.C.3 for the supersede path — since Drop 1 has already landed (commit `0a7ba80`, per Drop 3 PLAN.md "Blocked by: DROP_2 (closed at 0a7ba80)"), this is live code. Test that `supersede` on a STEWARD-owned item is gated identically to `MoveActionItemState`.

---

## Summary Of Required Plan Revisions Before Building

Routed back to orchestrator. Each one needs an explicit decision before any 3.C.* droplet starts.

1. **C1** — `UpdateActionItem` field-level guard for `Owner` and `DropNumber` when `existing.Owner == "STEWARD"` and `caller.AuthPrincipalType != "steward"`. **Blocking.**
2. **C2** — Decide whether `ReparentActionItem` is gated. **Blocking** for the auto-generator's parent-id-lookup contract.
3. **C3** — Specify the autent boundary mapping for `principal_type=steward`. **Blocking** — without this, builders write code that fails at runtime.
4. **C4** — Pick adapter-pre-fetch vs service-layer-gate for `MoveActionItem`. **Blocking** for Droplet 3.C.3 implementation.
5. **C5** — Specify how/when the 5 STEWARD persistent parents are seeded. **Blocking** for Droplet 3.C.4 + 3.C.6.
6. **C6** — Add 3.C.6 integration test for the gate's "manual-update misses mid-drop child" failure mode.
7. **C7** — Cover § 19.3 bullet 9 (kind-level STEWARD-owned marking) or explicitly defer.
8. **C8** — Cover the post-Drop-1 supersede path.

Plus the lower-severity nits (N1 column-only move semantics, N3 field naming, N4 rollback statement, N5 index design).

## Most Damaging Counterexample

**C1 (UpdateActionItem becomes a state-lock bypass).** The whole point of Unit C's auth-layer enforcement is to end the honor-system. C1 leaves the honor-system in place — a hostile or buggy drop-orch session simply clears the `Owner` field via update, then transitions state with no further gate. C3 (autent boundary) is also high-impact but a builder would catch it at first test run; C1 is a silent-bypass class bug that integration tests as currently sketched in 3.C.6 wouldn't even detect (3.C.6 only tests Owner=STEWARD against gate-fire, not Owner-clearing-via-update). Fix C1 first.

## Hylla Feedback

N/A — task touched non-Go files only. The falsification reasoning ran against Go source via `LSP` (workspaceSymbol / findReferences / goToDefinition) and direct `Read` on the Go source files cited above (`auth_request.go`, `authenticated_caller.go`, `app_service_adapter_mcp.go`, `service.go`, `snapshot.go`, vendored `autent` library). No Hylla `hylla_*` queries were attempted because the live source-code reads were already deterministic and faster than vector search for confirming concrete line numbers and call paths in already-cited code. This is consistent with the planner's note (Unit C PLAN.md line 263) that direct reads were the most efficient evidence path for this specific plan.
