# Plan-QA-Falsification Round 2 — DROP_4c.6.W0

**Round:** 2
**Verdict:** fail
**Counterexample count:** 1 CONFIRMED (medium)
**Reviewer mode:** L2 plan-QA-falsification (parent kind = `plan`).
**Inputs read:**

- `workflow/drop_4c_6/DROP_4c.6.W0_AGENTS_TOML_SCHEMA/PLAN.md` (round-2 revised — under attack)
- `workflow/drop_4c_6/DROP_4c.6.W0_AGENTS_TOML_SCHEMA/PLAN_QA_FALSIFICATION.md` (round-1 verdict)
- `workflow/drop_4c_6/DROP_4c.6.W0_AGENTS_TOML_SCHEMA/_BLOCKERS.toml`
- `workflow/drop_4c_6/PLAN.md` lines 51-67 (L1 W0 contract)
- `workflow/drop_4c_6/SKETCH.md` § 4, § 5, § 26.W0
- `~/.claude/agents/go-qa-falsification-agent.md` § "Plan-QA-Falsification axis"
- `workflow/example/drops/WORKFLOW.md` § "_BLOCKERS.toml — Sibling Blocker Ledger"

## 1. Round-1 Counterexample Verification

**FF1 (D3 AC bullet 8 ↔ D5 RiskNote line-number contradiction): PARTIAL — fixed in PROSE; residual drift in KindPayload JSON.**

Round-1 finding 1.1 named five drift sites the planner needed to rework. All five PROSE sites in round-2 PLAN.md now consistently push the file/line/block wrapping to D5 and scope D3 to bare-sentinel rejection:

| Site | Line | Status | Verification |
| --- | --- | --- | --- |
| AC2 ("returns `ErrToolsDenyNotOverridable`...") | 111 | FIXED | "TOML-position wrapping is added by D5's envelope; D3 raises the bare sentinel." |
| AC3 (bare sentinel message shape) | 112 | FIXED | Bare reads `"tools_deny is not user-overridable; remove the field"` (no file/line/block prefix at the D3 boundary); D5 wraps for user-facing render. |
| AC8 (`TestMergeLocal_ToolsDenyRejected` expectation) | 117 | FIXED | "asserts `errors.Is(err, ErrToolsDenyNotOverridable)` succeeds against the sentinel...this bullet covers only the sentinel-rejection contract D3 owns." Explicit forward-reference to D5's `TestMergeLocal_ToolsDenyPositionWrapped`. |
| RiskNote ("`tools_deny` rejection MUST surface...") | 128 | FIXED | "D3 raises the bare sentinel; D5's envelope adds the file/block/line wrapping. D3's tests assert sentinel-rejection only; D5's tests assert the wrapping." |
| ContextBlock (constraint critical at D3) | 132 | FIXED | "...via the closed sentinel `ErrToolsDenyNotOverridable` (D3) wrapped into a position-aware `*ConfigError` envelope (D5). Safety floor per `SKETCH.md` § 4.3.1." |

D5 RiskNote at line 203 is internally consistent with the reworded D3: it correctly observes `MergeLocal` doesn't parse TOML and the line number must come from D1's `LoadRegistry` step (via a `tools_deny_line` field or similar threading on `AgentsRegistry`). No D3↔D5 contradiction remains in PROSE.

**However, the rework did NOT propagate into D3's KindPayload JSON at line 137.** That JSON is the structured spec the builder reads to decide what to ship; mismatch between PROSE acceptance and KindPayload `shape_hint` strings re-creates the FF1 contradiction at a different surface. See Counterexample 2.1 below.

## 2. Fresh Counterexamples

- 2.1 **[CONFIRMED — medium] [Family: A2 Contract-mismatch + intra-droplet self-contradiction]** D3 KindPayload `shape_hint` strings retain the original FF1 line-number language that AC2/AC3/AC8/RiskNote/ContextBlock now explicitly forbid at D3's boundary. Two sub-clauses are load-bearing:

  - **(a)** `KindPayload.changes[0].shape_hint = "MergeLocal(project, local) deep-merges local registry over project; rejects local tools_deny with sentinel error citing TOML line; reuses D2's mergeMaps helper"` (PLAN.md:137) — the phrase `"sentinel error citing TOML line"` directly contradicts AC2 (line 111: "TOML-position wrapping is added by D5's envelope; D3 raises the bare sentinel") and AC3 (line 112: "no file/line/block prefix at the D3 boundary").
  - **(b)** `KindPayload.changes[1].shape_hint = "table-driven; tools_deny test asserts errors.Is(err, ErrToolsDenyNotOverridable) + line-number presence in message"` (PLAN.md:137) — the phrase `"+ line-number presence in message"` directly contradicts AC8 (line 117: "this bullet covers only the sentinel-rejection contract D3 owns") and the D3 RiskNote (line 128: "D3's tests assert sentinel-rejection only").

  → **Repro:** PLAN.md:111 vs PLAN.md:137 (production shape-hint clause); PLAN.md:117 vs PLAN.md:137 (test shape-hint clause).
  → **Fix hint:** in D3's KindPayload at PLAN.md:137, edit (a) `shape_hint` for `MergeLocal, ErrToolsDenyNotOverridable` to delete `"; rejects local tools_deny with sentinel error citing TOML line"` and replace with `"; rejects local tools_deny by returning bare ErrToolsDenyNotOverridable sentinel (D5 wraps with file/line/block)"`. Edit (b) `shape_hint` for the test list to delete `"+ line-number presence in message"` and replace with `"; line-number wrapping asserted separately by D5's TestMergeLocal_ToolsDenyPositionWrapped"`. Same Option-α intent as the round-1 fix; planner missed propagating into the JSON `shape_hint` strings the builder consumes.

  → **Why this is a real counterexample, not a stylistic nit:** the `KindPayload` JSON is the planner's structured handoff to the builder. `~/.claude/agents/go-planning-agent.md` § "Tillsyn-Flavored Specify Pass" treats it as a first-class metadata field; builders inspect `shape_hint` strings to disambiguate when prose acceptance is multi-bullet. A builder reading PROSE-first and KindPayload-second sees consistent intent; a builder reading KindPayload-first (or treating it as a tie-breaker) reproduces the original FF1 contradiction by stamping line-number assertions D3 has no infrastructure to back. The plan's own "Decomposition rationale" line 222 still claims the serial chain "preserves each prior droplet's CI-passable boundary" — that property remains broken at the JSON layer until the propagation lands.

## 3. Summary

**Verdict:** fail (1 CONFIRMED counterexample, medium — same FF1 family, residual at JSON shape-hint layer not reached by the round-2 prose reword).

| Family | Result | Notes |
| --- | --- | --- |
| FF1 verification (round-1 counterexample) | PARTIAL FIX | 5/5 prose drift sites cleanly reworded; 2/2 KindPayload `shape_hint` clauses at PLAN.md:137 retain the original drift. Re-emerges as Counterexample 2.1. |
| A1 Concurrency / `blocked_by` cycles, drift, sibling overlap | EXHAUSTED — no counterexample | Strict serial chain D1→D2→D3→D4→D5 unchanged from round 1; `_BLOCKERS.toml` `[[blockers]]` rows mirror PLAN.md `Blocked by:` bullets exactly (D2→D1, D3→D2, D4→D3, D5→D4). No cycles. No sibling shares a path entry (each droplet sole owner of its added/modified file lines). Package-shared-compile rule honored by full serialization per `~/.claude/agents/go-planning-agent.md` HARD RULES. |
| A2 Contract-mismatch L1 AC vs L2 droplet AC + intra-droplet | 1 CONFIRMED (medium) — see 2.1 | All five L1 W0 AcceptanceCriteria bullets at `workflow/drop_4c_6/PLAN.md:60-64` still covered. L1↔L2 function-name/parameter-order drift (`Merge(localRegistry, projectRegistry)` at L1 vs `MergeLocal(project, local)` at L2) remains low-noise as in round 1. The CONFIRMED counterexample is the residual D3 KindPayload drift — same FF1 family migrated to JSON. |
| A3 Hidden-coupling — cross-wave consumers named | EXHAUSTED — no counterexample | D1 types → D2/D3 named (in-W0). D2 `Resolve` → W3 named (PLAN.md:87 + :228). D3 `MergeLocal` → W3 named (PLAN.md:228). D4 `StripFrontmatterKeys` → W3 named (PLAN.md:170 + :230). D5 `ConfigError` → W11 named (PLAN.md:231). Every shipped symbol has at least one named consumer in the plan or downstream wave reference. |
| A4 YAGNI / premature abstraction | EXHAUSTED — no counterexample | Every droplet maps to a specific L1 W0 AC bullet. No interfaces with one implementation. The pointer-field `Override` shape is justified by the absent-vs-zero discrimination required by §4.2.1. The `mergeMaps` / `replaceList` helpers are factored once and reused across D2+D3 — DRY, not premature. |
| A5 Shipped-but-not-wired | EXHAUSTED — no counterexample | Every new symbol has a named consumer. The plan calls out W2/W3/W11 explicitly under "Out of scope for W0" (PLAN.md:228-231), preserving the cross-wave wiring chain. |
| A6 Atomicity sizing | EXHAUSTED — no counterexample | Sizing unchanged from round 1. D5 still revises D1+D3 prior-test assertions; planner self-justifies last-place ordering at PLAN.md:222. Total touched-LOC stays in budget. |
| A7 Prompt-injection (DORMANT pre-team-feature) | EXHAUSTED — gated dormant | Per `~/.claude/agents/go-qa-falsification-agent.md` § "Prompt-Injection Attack Family (Dormant Pre-Team-Feature)" — activation gate is post-MVP team feature, not yet landed. No attack surface to exercise. |
| Phase-2-step-1 required: missing `blocked_by` | EXHAUSTED | Every D2-D5 has explicit `Blocked by:` bullet to its predecessor; D1 has `Blocked by: —` (root of W0 chain) which is correct because W0 sub-plan itself sits at L1 with `Blocked by: —`. |
| Phase-2-step-1 required: cycles in `blocked_by` | EXHAUSTED | Linear chain D1→D2→D3→D4→D5; no cycles. |
| Phase-2-step-1 required: `_BLOCKERS.toml` ↔ PLAN.md drift | EXHAUSTED | PLAN.md (truth): D1=—, D2=D1, D3=D2, D4=D3, D5=D4. `_BLOCKERS.toml` records 4 `[[blockers]]` rows mirroring exactly. D1 correctly omitted (root). Round-2 prose reword did not touch blocker structure. |

The CONFIRMED counterexample at 2.1 is editorial — same character of fix as round-1's Option α (rewording prose-vs-impl drift), this time on the JSON `shape_hint` strings the round-1 reword missed. It does NOT block the plan's overall decomposition shape, the serial chain, the symbol set, the test set, or the cross-wave consumer wiring. Once D3's KindPayload `shape_hint` clauses are propagated to match the AC2/AC3/AC8/RiskNote prose, the plan converges PASS.

## 4. Hylla Feedback

N/A — review touched only Markdown plan artifacts (PLAN.md, _BLOCKERS.toml, prior-round verdict, SKETCH.md, parent PLAN.md, agent definition). Hylla is Go-only today per `feedback_hylla_go_only_today.md`; no Hylla query was needed and no fallback miss to log.
