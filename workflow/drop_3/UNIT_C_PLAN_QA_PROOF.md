# DROP 3 — UNIT C PLAN — QA PROOF REVIEW (ROUND 1)

**Reviewer:** `go-qa-proof-agent` (subagent, opus)
**Target:** `workflow/drop_3/UNIT_C_PLAN.md` (Round 1 author: `go-planning-agent`).
**Verdict:** **PASS with minor nits** (no blockers; recommended tightenings only).
**Date:** 2026-05-02
**Review mode:** plan-QA proof — evidence completeness against the spec, claim-to-source verification, no falsification.

## 1. Required Proof Checks

### 1.1 `principal_type` extension to 5th value (`steward`)

**Premise:** the planner identifies the existing 4-case switch and the one-line addition correctly.

**Evidence:** Read `internal/domain/auth_request.go:596-608` directly. Today's switch has 4 cases (counting the `""/"user"` fallthrough as one): `""/"user" → "user"`, `"agent" → "agent"`, `"service"/"system" → "service"`, default → `ErrInvalidActorType`. UNIT_C_PLAN.md droplet 3.C.3 (line 108) cites the exact line range `:596-608` and describes the addition as a fifth case `"steward" → "steward", nil`.

**Trace:** plan adds one switch arm; surrounding validation in `NewAuthRequest` (`:388-405`) is correctly identified — the plan calls out lines `:393-405`. Today's block treats `principal_type == "agent"` specially (must carry a `principalRole`); the plan extends that pattern to `steward` requiring `principal_role: orchestrator`.

**Conclusion:** correctly identified. The plan's note that `steward` follows the same "non-empty role required" pattern as `agent` is consistent with the existing code at `:393-405`.

**Unknowns:** the plan does not explicitly say whether `principal_role: orchestrator` is the SOLE accepted role or one of several — re-reading line 108 and acceptance bullet at line 125 confirms "orchestrator-only role accepted; reject any other role" — proof clean.

**Verdict:** **PASS.**

### 1.2 Auth-layer state-lock integration point

**Premise:** `MoveActionItem` and `MoveActionItemState` are the only two state-mutation paths the plan needs to gate.

**Evidence:** Read `internal/adapters/server/common/app_service_adapter_mcp.go:728` (`MoveActionItem`) and `:744` (`MoveActionItemState`). Both call `withMutationGuardContext` (`:1807-1859`), which populates `domain.AuthenticatedCaller` at `:1852` with `PrincipalID`, `PrincipalName`, `PrincipalType ActorType`. Plan correctly cites all three line numbers and the wiring path.

**Trace:** plan adds an `assertOwnerStateGate(ctx, item)` helper invoked from both move paths AFTER fetching the item (line 110-113). For `MoveActionItemState`, the natural fetch point is `a.service.GetActionItem` at `:760` (verified in source). For `MoveActionItem`, the fetch happens inside `a.service.MoveActionItem` at `:736` — the helper would need to fetch separately or have the underlying service expose the fetched item. **Nit (see § 3.1):** plan is slightly fuzzy on the `MoveActionItem` (column-move) gate's fetch point — it succeeds in `MoveActionItemState` because of the explicit `GetActionItem` call at `:760`, but `MoveActionItem` calls into `a.service.MoveActionItem` directly without a pre-fetch. Builder will need to add a `GetActionItem` call (or refactor to expose the fetched item from the service layer) before the gate fires. This is a builder-level detail, not a planner-level miss; the plan's acceptance bullet (line 127) does cover "MoveActionItem ... reject state transitions on Owner = STEWARD" so the builder gets the directive.

**Conclusion:** the two state-mutation entry points are correctly identified as the ONLY paths needing the gate. `UpdateActionItem`, `CreateActionItem`, `ReparentActionItem` are correctly carved out (line 120-121, 127).

**Unknowns:** plan flags but does not lock the exact wiring shape for `MoveActionItem`'s pre-fetch — surfaced as nit § 3.1.

**Verdict:** **PASS.**

### 1.3 `Owner` + `DropNumber` first-class fields mirroring Drop 2.3's Role pattern

**Premise:** the planner correctly mirrored Drop 2.3's `Role` field landing pattern.

**Evidence:** Read `internal/domain/action_item.go:25-83` (struct + input) and `:96-219` (`NewActionItem`). The `Role` field appears at `:33` on `ActionItem` and `:67` on `ActionItemInput`. Validation block at `:150-158` (the plan cites `:155-158`, which is correct for the active `NormalizeRole` + `IsValidRole` lines; the comment block starts at `:150`). The struct-literal return is at `:195-218` with `Role: in.Role` at `:201`. Plan cites `:33`, `:67`, `:155-158`, `:201` — every cite matches.

**Trace:** plan adds two new fields next to `Role`. New error `ErrInvalidDropNumber` mirrors `ErrInvalidRole` (line 71). Owner is trimmed, no closed enum (line 70 — matches the orchestrator-locked architectural decision in the spawn brief). DropNumber rejects `< 0` (line 71). Test cases cover empty/whitespace/round-trip/negative-rejection (line 65, 73-74).

**Conclusion:** Drop 2.3 pattern correctly mirrored. Both fields land on `ActionItem` + `ActionItemInput` + `NewActionItem` validation + struct-literal return — same five touchpoints Drop 2.3 used for `Role`.

**Unknowns:** none.

**Verdict:** **PASS.**

### 1.4 Template `[child_rules]` consumer hook — hard blocker on Unit B's rule-engine droplet

**Premise:** Droplet 3.C.4 is correctly hard-blocked on Unit B's rule-engine droplet.

**Evidence:** Read `workflow/drop_3/UNIT_B_PLAN.md`. Unit B's rule-engine surface is **`3.B.4 — Template.ChildRulesFor`** (lines 96-112 in UNIT_B_PLAN.md). UNIT_C_PLAN.md droplet 3.C.4's `Blocked by` line (line 187) reads `3.C.3, 3.B.<final-rule-engine-droplet>` — the planner correctly defers the exact ID to orchestrator synthesis time, knowing the cross-unit ID couldn't be resolved at planning time. The Cross-Unit Dependencies table (line 47) explicitly calls this out as **Hard** with a note: "orch wires this at synthesis after Unit B's PLAN.md surfaces the specific droplet ID."

**Trace:** plan correctly identifies that **`3.B.4`** (`Template.ChildRulesFor`) is the rule-engine surface Unit C consumes — though the plan uses placeholder `<final-rule-engine-droplet>`. **Nit (see § 3.2):** Unit B's PLAN.md is now available alongside Unit C's, so the placeholder could be resolved to `3.B.4` directly. The orchestrator should make this substitution at synthesis. This is NOT a planner miss — the planner correctly flagged it for orchestrator wiring.

**Conclusion:** hard blocker correctly identified, deferred to synthesis-time wiring per the spec.

**Unknowns:** none — the placeholder mechanism is correct.

**Verdict:** **PASS.**

### 1.5 5 STEWARD level_2 findings + refinements-gate

**Premise:** all 5 findings parents named + the gate item present.

**Evidence:** Read `main/PLAN.md` § 15.7 lines 1265-1276 (the persistent-drop table) and § 19.3 lines 1660-1661 (template auto-generation bullets). The 6 STEWARD persistent level_1 drops are: `DISCUSSIONS`, `HYLLA_FINDINGS`, `LEDGER`, `WIKI_CHANGELOG`, `REFINEMENTS`, `HYLLA_REFINEMENTS`. Per § 19.3 line 1660, auto-generation creates **5** level_2 findings (NOT 6 — `DISCUSSIONS` is excluded because it's cross-cutting audit trail without a single MD; verified at line 1269): `DROP_N_HYLLA_FINDINGS`, `DROP_N_LEDGER_ENTRY`, `DROP_N_WIKI_CHANGELOG_ENTRY`, `DROP_N_REFINEMENTS_RAISED`, `DROP_N_HYLLA_REFINEMENTS_RAISED`.

**Trace:** UNIT_C_PLAN.md scope paragraph (line 22) names all 5 findings drops verbatim AND the refinements-gate (`DROP_N_REFINEMENTS_GATE_BEFORE_DROP_N+1`). The auto-generator template sketch (lines 142-169) covers all 6 spawns (5 findings + 1 gate). Acceptance bullet (line 180) confirms "5 STEWARD level_2 findings auto-create on numbered-drop creation" + "refinements-gate auto-creates inside the drop's tree."

**Conclusion:** all 5 findings + refinements-gate correctly enumerated. `DISCUSSIONS` correctly excluded (matches spec at PLAN.md line 1269).

**Unknowns:** none.

**Verdict:** **PASS.**

### 1.6 Same-file race between 3.C.3 and 3.C.5 on `app_service_adapter_mcp.go`

**Premise:** the planner correctly serialized the two droplets via explicit `Blocked by`.

**Evidence:** UNIT_C_PLAN.md "Same-Package Compile Race Wiring" section (lines 256-259) explicitly addresses this: "3.C.3 and 3.C.5 both touch `internal/adapters/server/common/app_service_adapter_mcp.go`. Hard same-file race. 3.C.5 is `Blocked by: 3.C.3` (transitively via 3.C.2 → 3.C.3 → 3.C.5)." Confirmed by reading droplet 3.C.5's `Blocked by` line (line 212) which reads `3.C.2`, and the parallel-with note (line 214) which states "ordered AFTER 3.C.3, NOT in parallel."

**Trace:** the dependency chain is `3.C.2 → 3.C.3 → 3.C.5` (3.C.3 blocks on 3.C.2 at line 132; 3.C.5 blocks on 3.C.2 at line 212). **Nit (see § 3.3):** the plan's claim that 3.C.5 is "transitively" blocked by 3.C.3 via the chain `3.C.2 → 3.C.3 → 3.C.5` is INCORRECT — 3.C.5's `Blocked by` lists only `3.C.2`, NOT `3.C.3`. The transitive ordering only holds if 3.C.3 also `Blocked by: 3.C.2` AND 3.C.5 is scheduled AFTER 3.C.3 by some other mechanism. In practice, 3.C.3 and 3.C.5 both share `3.C.2` as their only upstream blocker — meaning a parallel scheduler could fire BOTH after 3.C.2 completes, hitting the same-file race. **The fix is to make 3.C.5's `Blocked by` line read `3.C.3` instead of (or in addition to) `3.C.2`.** This is a minor planner nit — the prose is correct, but the formal `Blocked by` field must agree.

**Conclusion:** the intent is correct (3.C.5 must follow 3.C.3 due to same-file race), but the formal `Blocked by` wiring on 3.C.5 omits the explicit 3.C.3 blocker — surfaced as nit § 3.3 below.

**Unknowns:** whether the orchestrator's synthesis pass will catch this and fix the `Blocked by` declaration before dispatching builders.

**Verdict:** **PASS-WITH-NIT.** Intent verified, formal wiring needs a one-line fix.

### 1.7 Pre-MVP no-migration honored

**Premise:** schema additions trigger fresh-DB; no SQL backfill.

**Evidence:** UNIT_C_PLAN.md "Pre-MVP rules in effect" block (lines 32-38): "Schema additions (`owner`, `drop_number` columns on `action_items`; `steward` value in the principal_type enum check) are pre-MVP — **dev fresh-DBs `~/.tillsyn/tillsyn.db`** between schema-touching droplets. No migration logic in Go, no `till migrate` subcommand, no SQL backfill script."

**Trace:** every droplet's `DB action` line is consistent with this rule:

- 3.C.1 (line 76): `DB action: NONE` — struct-only.
- 3.C.2 (line 98): `DB action: DELETE ~/.tillsyn/tillsyn.db BEFORE running mage ci for this droplet (schema change).`
- 3.C.3 (line 131): `DB action: DELETE ~/.tillsyn/tillsyn.db BEFORE running mage ci (auth_request principal_type accepted set widens, although no schema column changes — defensive).`
- 3.C.4 (line 186): `DB action: DELETE ~/.tillsyn/tillsyn.db BEFORE running mage ci (rule-engine integration may add new rows on existing fixtures; safer to fresh-DB).`
- 3.C.5 (line 211): `DB action: NONE (data-shape only; schema lands in 3.C.2).`
- 3.C.6 (line 234): `DB action: DELETE ~/.tillsyn/tillsyn.db BEFORE running mage ci.`

**Conclusion:** rule honored across all 6 droplets. No SQL backfill, no migration code. Verified against the project memory `feedback_no_migration_logic_pre_mvp.md` rule.

**Unknowns:** none.

**Verdict:** **PASS.**

### 1.8 6 droplets cover the full PLAN.md § 19.3 + § 15.7 STEWARD scope

**Premise:** nothing in PLAN.md § 19.3 bullets 7-9 + § 15.7 STEWARD bullets is missed.

**Evidence:** Spec coverage matrix:

| PLAN.md ref | Spec bullet | UNIT_C_PLAN droplet |
|---|---|---|
| § 19.3 line 1659 (bullet 7) | `principal_type: steward` enum + auth-level state-lock + drop-orch-keep-create+update perms | 3.C.3 (auth gate); 3.C.5 (MCP plumbing for owner field that the gate reads) |
| § 19.3 line 1660 (bullet 8) | Template auto-generation of 5 level_2 findings + refinements-gate, `metadata.owner = STEWARD`, `metadata.drop_number = N`, blocked_by wiring | 3.C.4 (auto-gen consumer); 3.C.1 (Owner+DropNumber domain fields); 3.C.2 (SQLite columns + index) |
| § 19.3 line 1661 (bullet 9) | Template-defined STEWARD-owned drop kind(s) — drop-orchs create + edit description, only `steward`-principal moves state | 3.C.3 + 3.C.4 — bullet 9 is "templates allow marking specific kinds as STEWARD-owned" which Unit B's `KindRule.Owner string` field (UNIT_B_PLAN line 45) handles on the schema side; Unit C consumes it via the auth gate. |
| § 15.7 line 1278-1287 | Drop-close sequence: STEWARD closes 5 level_2 findings + refinements-gate; parent-blocks-on-incomplete-child | 3.C.6 integration test case 3 (line 226) |
| § 15.7 STEWARD persistent parents | Auto-gen children land under correct STEWARD parents | 3.C.4 `parent_id_lookup = "owner=STEWARD,title=..."` (line 171) |
| § 15.8 refinements-gate (PLAN line 1293) | `blocked_by` every other Drop N item + 5 findings | 3.C.4 `blocked_by_lookup` (line 167, 172) + acceptance bullet (line 181) |

**Trace:** all 8 PLAN.md bullets in scope are covered. **Nit (see § 3.4):** PLAN.md § 19.3 bullet 9 line 1661 ("Template-defined STEWARD-owned drop kind(s)") is not explicitly enumerated as a separate Unit C deliverable — the plan handles it implicitly via Unit B's `KindRule.Owner` schema field + Unit C's auth gate keying on `Owner == "STEWARD"`. This works (and the plan's "Out of scope" line 27 correctly delegates the schema slot to Unit B), but a one-line acknowledgment in Unit C's scope would strengthen the cross-reference. Plan is correct as-is; this is a doc-tightening nit only.

**Conclusion:** all 6 droplets together cover the full Unit C scope per the spawn brief and PLAN.md § 19.3 bullets 7-9 + § 15.7. No spec bullet is unmapped.

**Unknowns:** the plan's Architectural Decision #5 (line 247) routes a dynamic-vs-static `blocked_by` resolution question back to dev — this is correctly flagged as an open question routed to orchestrator; not a proof miss.

**Verdict:** **PASS.**

## 2. Cross-Cutting Coverage

### 2.1 Drop 2.3 Role precedent fidelity

The plan repeatedly invokes Drop 2.3's Role-field landing pattern as the architectural precedent for Owner + DropNumber. Verified by reading `internal/domain/action_item.go` directly:

- Struct field at `:33`: ✓
- Input field at `:67`: ✓
- Validation block at `:150-158`: ✓
- Struct-literal return at `:201`: ✓
- Sentinel error `ErrInvalidRole` (mirror for `ErrInvalidDropNumber`): ✓ (referenced in PLAN.md § 19.2 bullet 1)

The plan correctly mirrors all 5 touchpoints for Owner + DropNumber.

### 2.2 `AuthenticatedCaller` extension architecture

UNIT_C_PLAN.md line 111 makes a load-bearing claim: that the existing `domain.AuthenticatedCaller.PrincipalType` field is `ActorType` (user/agent/system) and is **distinct** from the auth-request `principal_type` (`user|agent|service|steward`).

**Verified** by reading `internal/domain/authenticated_caller.go:8-13` and `:21-29`: today's `PrincipalType` field is typed `ActorType`, with the switch at `:21` accepting only `ActorTypeUser`, `ActorTypeAgent`, `ActorTypeSystem`. The plan's introduction of a new `AuthPrincipalType string` field (carrying `user|agent|service|steward`) is architecturally sound — conflating the two would require adding `steward` to `ActorType`, which would ripple into `change_events.actor_type` + `created_by_type` columns (correctly noted in plan line 245).

### 2.3 Index design

Droplet 3.C.2 line 88 introduces `CREATE INDEX IF NOT EXISTS idx_action_items_project_owner_drop_number ON action_items(project_id, owner, drop_number)`. The composite key matches the auto-generator's two cross-row queries (line 171-172):

1. `WHERE project_id = ? AND owner = 'STEWARD' AND title = ?` — covered by the index's `(project_id, owner)` prefix.
2. `WHERE project_id = ? AND drop_number = ?` — partially covered; the leftmost prefix `(project_id, owner, drop_number)` does NOT skip-scan efficiently in SQLite without a leading `owner` filter. **Nit (see § 3.5):** the `ListActionItemsByDropNumber` query (line 172) lacks an `owner` filter, so the index degrades to a `project_id`-prefix scan with a filter on `drop_number`. This still works for moderate row counts but isn't optimally covered. Either:

   - Add a second index `(project_id, drop_number)` for that specific query, OR
   - Tighten `ListActionItemsByDropNumber`'s SQL to also filter on `owner != ''` if the auto-generator only ever queries owned items (semantically tighter), OR
   - Accept the partial coverage given the small expected row counts (typically < 50 items per drop).

This is a builder-level optimization detail, not a planner miss; the plan correctly identifies that an index is needed.

## 3. Nits (Non-Blocking)

### 3.1 `MoveActionItem` (column-move) gate fetch point

Droplet 3.C.3 line 110-113 describes the gate helper invocation but is fuzzy on `MoveActionItem`'s pre-fetch — `MoveActionItemState` has an explicit `GetActionItem` at `:760` to consult, but `MoveActionItem` (`:728`) goes straight into `a.service.MoveActionItem` without a domain-level fetch. Builder will need to add a `GetActionItem` call OR refactor the service layer. **Recommendation:** orchestrator surfaces this to the builder as a synthesis-time clarification — "for MoveActionItem, add a `GetActionItem` call before the gate fires; do NOT skip the gate just because the column-move path doesn't pre-fetch today."

### 3.2 Resolve `<final-rule-engine-droplet>` placeholder to `3.B.4`

Droplet 3.C.4's `Blocked by` line (line 187) carries `3.B.<final-rule-engine-droplet>` as a placeholder. Unit B's PLAN.md is now available; the rule-engine surface is `3.B.4` (`Template.ChildRulesFor`). Orchestrator should substitute at synthesis. Cross-unit dependencies table at line 47 also carries the placeholder.

### 3.3 3.C.5's `Blocked by` should explicitly include 3.C.3

Droplet 3.C.5's `Blocked by` (line 212) reads `3.C.2` — but per the same-file race on `internal/adapters/server/common/app_service_adapter_mcp.go` (lines 256-259), 3.C.5 must run AFTER 3.C.3, not in parallel with it. The plan's prose at line 214 ("ordered AFTER 3.C.3, NOT in parallel") is correct but contradicts the formal `Blocked by` field. **Recommended fix:** change line 212 to `Blocked by: 3.C.3` (which transitively includes 3.C.2 since 3.C.3 blocks on 3.C.2). This is a one-line edit.

### 3.4 PLAN.md § 19.3 bullet 9 (template-defined STEWARD-owned kinds) cross-ref

Bullet 9 is implicitly handled via Unit B's `KindRule.Owner string` schema field + Unit C's auth gate. The plan's "Out of scope" line 27 correctly delegates the schema slot to Unit B, but a one-sentence forward-reference in Unit C's scope ("Unit B's `KindRule.Owner` schema field is the consumer that template-defined STEWARD-owned kinds bind into; Unit C's auth gate enforces the `Owner == STEWARD` ⇒ `AuthPrincipalType == steward` rule against any kind whose template marks it owned") would tighten the cross-unit cross-reference. **Recommendation:** doc-tightening only; no architectural change.

### 3.5 Index coverage for `ListActionItemsByDropNumber`

The composite index `(project_id, owner, drop_number)` doesn't optimally cover queries that filter only on `(project_id, drop_number)` without an `owner` prefix. Builder should either add a second index OR tighten the query to include `owner`. **Recommendation:** acceptance criterion in 3.C.2 could be tightened to specify "the auto-generator's two cross-row queries are fully index-covered" rather than just naming the index. Pre-MVP performance is unlikely to bite, but the design intent should match the wired index.

## 4. Open Questions Routed Back To Orchestrator

The plan flags 5 architectural decisions (lines 243-247) as locked-at-planner-time but pending dev sign-off at synthesis:

1. **Owner as first-class string (NOT closed enum).** Locked per orchestrator-locked architectural decision in the spawn brief. ✓ — proof confirms scope-brief alignment.
2. **DropNumber as first-class int (NOT metadata JSON).** Locked-in by planner; routed to dev. Recommended rationale: read frequency at rule-fire + cross-row-query time. ✓ — sound.
3. **`AuthPrincipalType` distinct from `PrincipalType ActorType` on `AuthenticatedCaller`.** Locked-in. ✓ — verified against `internal/domain/authenticated_caller.go` source.
4. **Auth gate fires ONLY on state-transition mutations.** Locked-in per § 19.3 bullet 7 line 1659 ("drop-orchs keep `create` + `update(description/details/metadata)` permissions"). ✓ — matches spec.
5. **Refinements-gate `blocked_by` is dynamic-at-create + manual-update, NOT rule-fire-on-every-new-drop-N-child.** Locked-in by planner; routed to dev. ✓ — sound recommendation.

All 5 decisions are correctly flagged for orchestrator ↔ dev sync at synthesis time.

## 5. Hylla Feedback

N/A — task touched non-Go files only (planner authored MD; QA reviewer's evidence work was Read-tool against existing Go files plus MD spec readout). Hylla today indexes Go committed code only per `feedback_hylla_go_only_today.md`; the planning-review surface is MD-dominant. No Hylla queries attempted because direct `Read` calls on the cited files (`internal/domain/auth_request.go`, `internal/domain/authenticated_caller.go`, `internal/domain/action_item.go`, `internal/adapters/server/common/app_service_adapter_mcp.go`) plus `Read` on PLAN.md and the four drop-3 unit plans were the most efficient evidence path. No Hylla miss to record.

Ergonomic note: planner-side `Hylla Feedback: N/A — non-Go` and reviewer-side `Hylla Feedback: N/A — non-Go MD review` patterns are now common for plan-QA passes. Aggregator should accept this as expected signal rather than flagging.

## 6. Verdict Summary

**PASS** — Round 1 plan ready for build phase, conditional on the 5 nits below being addressed at synthesis time:

1. **§ 3.1** — Orchestrator clarifies `MoveActionItem` pre-fetch wiring for builder.
2. **§ 3.2** — Substitute `3.B.<final-rule-engine-droplet>` with `3.B.4` in Unit C plan.
3. **§ 3.3** — Tighten 3.C.5's `Blocked by` to include `3.C.3` explicitly.
4. **§ 3.4** — Add one-sentence forward-reference to Unit B's `KindRule.Owner` for PLAN.md bullet 9 coverage.
5. **§ 3.5** — Tighten index acceptance criterion in 3.C.2 to "cross-row queries fully index-covered."

None are blocking. None require re-planning. All are synthesis-time tightenings the orchestrator can apply when wiring cross-unit dependencies. The 6-droplet decomposition correctly covers the full Unit C scope, mirrors Drop 2.3's Role precedent faithfully, and honors all pre-MVP rules.

**No Round 2 required pending falsification sibling's verdict.**
