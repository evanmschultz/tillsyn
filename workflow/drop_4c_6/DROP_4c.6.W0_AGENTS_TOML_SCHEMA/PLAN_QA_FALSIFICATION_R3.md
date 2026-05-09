# Plan-QA-Falsification Round 3 — DROP_4c.6.W0

**Round:** 3
**Verdict:** pass
**Counterexample count:** 0
**Reviewer mode:** L2 plan-QA-falsification (parent kind = `plan`).
**Inputs read:**

- `workflow/drop_4c_6/DROP_4c.6.W0_AGENTS_TOML_SCHEMA/PLAN.md` (round-3 revised — under attack)
- `workflow/drop_4c_6/DROP_4c.6.W0_AGENTS_TOML_SCHEMA/PLAN_QA_FALSIFICATION_R2.md` (round-2 verdict)
- `workflow/drop_4c_6/DROP_4c.6.W0_AGENTS_TOML_SCHEMA/PLAN_QA_FALSIFICATION.md` (round-1 verdict, context)
- `workflow/drop_4c_6/DROP_4c.6.W0_AGENTS_TOML_SCHEMA/_BLOCKERS.toml`
- `workflow/drop_4c_6/PLAN.md` lines 51-67 (L1 W0 contract)
- `workflow/drop_4c_6/SKETCH.md` § 4, § 5, § 26.W0 (referenced via R1/R2 trail)
- `~/.claude/agents/go-qa-falsification-agent.md` § "Plan-QA-Falsification axis"
- `workflow/example/drops/WORKFLOW.md` § "_BLOCKERS.toml — Sibling Blocker Ledger"

## 1. Round-2 Counterexample Verification

**W0-FF2-R2 (D3 KindPayload `shape_hint` retains line-number language): FIXED.**

R2 finding 2.1 named two sub-clauses at PLAN.md:137 the planner needed to propagate. R3 PLAN.md:137 applied both fixes verbatim per the R2 fix-hint replacement strings:

| Sub-clause | R2 mandate (exact replacement string) | R3 actual at PLAN.md:137 | Status |
| --- | --- | --- | --- |
| (a) Production `shape_hint` for `MergeLocal, ErrToolsDenyNotOverridable` | `"; rejects local tools_deny by returning bare ErrToolsDenyNotOverridable sentinel (D5 wraps with file/line/block)"` | `"...; rejects local tools_deny by returning bare ErrToolsDenyNotOverridable sentinel (D5 wraps with file/line/block); reuses D2's mergeMaps helper"` | FIXED — verbatim insertion; trailing `"; reuses D2's mergeMaps helper"` was already present in the original and survives the edit (does not introduce drift). |
| (b) Test-list `shape_hint` for `TestMergeLocal_*` | `"; line-number wrapping asserted separately by D5's TestMergeLocal_ToolsDenyPositionWrapped"` | `"table-driven; tools_deny test asserts errors.Is(err, ErrToolsDenyNotOverridable); line-number wrapping asserted separately by D5's TestMergeLocal_ToolsDenyPositionWrapped"` | FIXED — verbatim insertion replacing the old `"+ line-number presence in message"` clause. |

Both the offending phrases R2 flagged (`"sentinel error citing TOML line"` in (a) and `"+ line-number presence in message"` in (b)) are now absent from PLAN.md:137. The new wording is internally consistent with the AC2 (line 111), AC3 (line 112), AC8 (line 117), RiskNote (line 128), and ContextBlock (line 132) prose — all of which were verified FIXED in R2 round-1 verification and remain unchanged in R3.

**Cross-check: no fresh D3↔D5 line-number drift introduced elsewhere.** I scanned every D3 surface (AC bullets at 109-120, Specify Objective at 123, AcceptanceCriteria pointer at 124, ValidationPlan at 125, RiskNotes at 126-130, ContextBlocks at 131-136, KindPayload at 137) and every D5 surface (AC bullets at 185-195, RiskNotes at 201-205, ContextBlocks at 206-211, KindPayload at 212). The only line-number-aware language in D3 is the explicit forward-reference to D5's wrapping (intentional, contractually consistent) and the references to D5's `TestMergeLocal_ToolsDenyPositionWrapped`. No D3-owned bullet asserts D3 produces line-numbered output. D5 RiskNote at 203 still correctly observes `MergeLocal` doesn't parse TOML and the line number must come from D1's `LoadRegistry` step. No new contradiction.

## 2. Fresh Counterexamples

None CONFIRMED. Per Round-2 verdict §3 table, families A1, A3, A4, A5, A6, A7 were exhausted with no counterexample at R2; the R3 reword touched only the two `shape_hint` clauses at PLAN.md:137 inside D3's KindPayload — strictly local edits with no surface area for new attacks against the larger plan structure (decomposition shape, serial chain, symbol set, test set, cross-wave consumer wiring, sizing, blocker graph). I re-attacked each family below to confirm no regression slipped in alongside the targeted fix.

| Family | Attack-pass result | Notes |
| --- | --- | --- |
| A1 Concurrency / `blocked_by` cycles, drift, sibling overlap | EXHAUSTED — no counterexample | Strict serial chain D1→D2→D3→D4→D5 unchanged. `_BLOCKERS.toml` rows mirror PLAN.md `Blocked by:` bullets exactly (D2→D1, D3→D2, D4→D3, D5→D4). No cycle. No sibling-path overlap (each droplet sole owner of the file lines it adds/modifies). Package-shared-compile rule honored by full serialization per `~/.claude/agents/go-planning-agent.md` HARD RULES. |
| A2 Contract-mismatch L1 AC vs L2 droplet AC + intra-droplet | EXHAUSTED — no counterexample | All five L1 W0 AcceptanceCriteria bullets at parent PLAN.md:60-64 still covered by D1-D5. The R2 CONFIRMED finding (FF2) is now FIXED per §1 above. L1↔L2 function-name drift (`Merge(localRegistry, projectRegistry)` at L1 vs `MergeLocal(project, local)` at L2) noted in R1/R2 remains low-noise — L1 AC is shape-level, L2 binds the exact symbol; not a counterexample. |
| A3 Hidden-coupling — cross-wave consumers named | EXHAUSTED — no counterexample | D1 types → D2/D3 named (in-W0). D2 `Resolve` → W3 named (PLAN.md:87 + :228). D3 `MergeLocal` → W3 named (PLAN.md:228). D4 `StripFrontmatterKeys` → W3 named (PLAN.md:170 + :230). D5 `ConfigError` → W11 named (PLAN.md:231). Every shipped symbol has a named consumer. |
| A4 YAGNI / premature abstraction | EXHAUSTED — no counterexample | Every droplet maps to a specific L1 W0 AC bullet. No interfaces with one implementation. Pointer-field `Override` shape justified by absent-vs-zero discrimination per §4.2.1. `mergeMaps` / `replaceList` factored once, reused across D2+D3 — DRY, not premature. |
| A5 Shipped-but-not-wired | EXHAUSTED — no counterexample | Every new symbol has a named consumer. Plan calls out W2/W3/W11 explicitly under "Out of scope for W0" (PLAN.md:228-231), preserving the cross-wave wiring chain. |
| A6 Atomicity sizing | EXHAUSTED — no counterexample | Sizing unchanged from R2. D5 still revises D1+D3 prior-test assertions; planner self-justifies last-place ordering at PLAN.md:222. Total touched-LOC stays in the 80-120 LOC + tests budget per `~/.claude/agents/go-planning-agent.md` § Atomic Droplet Sizing. |
| A7 Prompt-injection (DORMANT pre-team-feature) | EXHAUSTED — gated dormant | Per `~/.claude/agents/go-qa-falsification-agent.md` § "Prompt-Injection Attack Family (Dormant Pre-Team-Feature)" — activation gate is post-MVP team feature, not yet landed. No attack surface. |
| Phase-2-step-1 required: missing `blocked_by` | EXHAUSTED — no counterexample | Every D2-D5 has explicit `Blocked by:` bullet to its predecessor; D1 has `Blocked by: —` (root of W0 chain) which is correct because W0 sub-plan itself sits at L1 with `Blocked by: —`. |
| Phase-2-step-1 required: cycles in `blocked_by` | EXHAUSTED — no counterexample | Linear chain D1→D2→D3→D4→D5; no cycles (acyclic by construction). |
| Phase-2-step-1 required: `_BLOCKERS.toml` ↔ PLAN.md drift | EXHAUSTED — no counterexample | PLAN.md (truth): D1=—, D2=D1, D3=D2, D4=D3, D5=D4. `_BLOCKERS.toml` records 4 `[[blockers]]` rows mirroring exactly. D1 correctly omitted (root). R3 reword did not touch blocker structure. |
| W0-FF2-R2 propagation regression-check | EXHAUSTED — no counterexample | R2 finding fully propagated; both targeted `shape_hint` clauses at PLAN.md:137 carry the verbatim replacement language from R2's fix-hint. No residual line-number assertion in any D3-owned surface. |

## 3. Summary

**Verdict:** pass — 0 CONFIRMED counterexamples.

| Family | Result |
| --- | --- |
| W0-FF2-R2 verification (R2 counterexample) | FIXED (verbatim) |
| A1 Concurrency / `blocked_by` cycles, drift, sibling overlap | EXHAUSTED |
| A2 Contract-mismatch L1↔L2 + intra-droplet | EXHAUSTED |
| A3 Hidden-coupling — cross-wave consumers named | EXHAUSTED |
| A4 YAGNI / premature abstraction | EXHAUSTED |
| A5 Shipped-but-not-wired | EXHAUSTED |
| A6 Atomicity sizing | EXHAUSTED |
| A7 Prompt-injection (dormant) | EXHAUSTED — gated |
| Phase-2: missing `blocked_by` | EXHAUSTED |
| Phase-2: `blocked_by` cycles | EXHAUSTED |
| Phase-2: `_BLOCKERS.toml` ↔ PLAN.md drift | EXHAUSTED |
| W0-FF2-R2 propagation regression-check | EXHAUSTED |

The R3 plan converges. The R2 CONFIRMED counterexample (W0-FF2-R2) is verbatim FIXED at PLAN.md:137 per the R2 fix-hint replacement strings. The fresh-attack pass surfaced no new counterexample across the seven attack families plus the three Phase-2-step-1 required attacks. Decomposition shape, serial chain, symbol set, test set, cross-wave consumer wiring, sizing, and blocker graph are all internally consistent and L1-AC-covering. No further plan-QA-falsification round is required for W0.

## 4. Hylla Feedback

N/A — review touched only Markdown plan artifacts (PLAN.md, _BLOCKERS.toml, prior-round verdict files, parent PLAN.md, agent definition). Hylla is Go-only today per `feedback_hylla_go_only_today.md`; no Hylla query was needed and no fallback miss to log.
