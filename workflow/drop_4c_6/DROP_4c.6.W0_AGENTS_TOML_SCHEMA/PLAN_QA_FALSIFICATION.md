# Plan-QA-Falsification Round 1 — DROP_4c.6.W0

**Round:** 1
**Verdict:** fail
**Counterexample count:** 1 CONFIRMED (medium)
**Reviewer mode:** L2 plan-QA-falsification (parent kind = `plan`).
**Inputs read:**

- `workflow/drop_4c_6/DROP_4c.6.W0_AGENTS_TOML_SCHEMA/PLAN.md` (under attack)
- `workflow/drop_4c_6/DROP_4c.6.W0_AGENTS_TOML_SCHEMA/_BLOCKERS.toml`
- `workflow/drop_4c_6/PLAN.md` lines 51-67 (L1 W0 contract)
- `workflow/drop_4c_6/SKETCH.md` § 4, § 5, § 26.W0
- `~/.claude/agents/go-qa-falsification-agent.md` § "Plan-QA-Falsification axis"
- `workflow/example/drops/WORKFLOW.md` § "_BLOCKERS.toml — Sibling Blocker Ledger"

## 1. Findings

- 1.1 [Family: A2 Contract-mismatch + intra-droplet self-contradiction] [severity: medium] Droplet D3 acceptance bullet 8 (PLAN.md line 117) requires `TestMergeLocal_ToolsDenyRejected` to assert "error message contains the TOML line number," but at D3's stage `MergeLocal` operates over already-decoded `AgentsRegistry` structs (no TOML positions in scope), and **D5's own RiskNote at PLAN.md line 203 explicitly acknowledges this gap** ("`MergeLocal` doesn't currently parse TOML; the line number for `tools_deny` rejection comes from D3's load step, NOT from `MergeLocal`'s own logic. D5 either threads the source-line through `AgentsRegistry`... OR `MergeLocal`'s caller re-decodes the file..."). D5 then claims to REVISE this assertion at line 194 (`TestMergeLocal_ToolsDenyPositionWrapped (REVISES D3's TestMergeLocal_ToolsDenyRejected expectation)`). This puts the builder in a contradiction at D3's boundary: either (a) the builder ships D3 without the line-number assertion (which violates D3's AC bullet 8 as written, leaving D3's own droplet contract unsatisfiable on its own boundary — the very thing PLAN.md "Decomposition rationale" line 222 claims D5's last-place ordering preserves), or (b) the builder threads line-tracking through `AgentsRegistry` inside D3 (which D5 then reorganizes / duplicates, doubling the implementation). → **Repro:** PLAN.md:117 vs PLAN.md:194 vs PLAN.md:203. → **Fix hint:** edit D3 AC bullet 8 to drop the line-number clause and assert only `errors.Is(err, ErrToolsDenyNotOverridable)` plus a placeholder block-name in the message; move the line-number assertion exclusively to D5's `TestMergeLocal_ToolsDenyPositionWrapped`. Alternatively, lift line-tracking infrastructure (line-positions map on `AgentsRegistry`) into D1's scope so D3 can read it, and reframe D5 as wrapping the existing line-aware error rather than introducing the line-tracking infrastructure.

## 2. Counterexamples

- 2.1 **[CONFIRMED — medium]** D3 acceptance bullet 8 self-contradiction at the droplet boundary. See finding 1.1 above for full repro pointers. The plan as written cannot land each droplet self-consistently in turn — exactly the property PLAN.md line 222 claims D5's last-place sequencing preserves. The closest "drop the line clause from D3" reading is reasonable + matches D5's REVISES wording, but the plan needs to make that explicit so the D3 builder doesn't stamp the test with a line assertion that has no implementation backing.

## 3. Summary

**Verdict:** fail (1 CONFIRMED counterexample).

| Family | Result | Notes |
| --- | --- | --- |
| A1 Concurrency / `blocked_by` cycles, drift, sibling overlap | EXHAUSTED — no counterexample | Strict serial chain D1→D2→D3→D4→D5; no cycle; PLAN.md `Blocked by:` bullets and `_BLOCKERS.toml` `[[blockers]]` rows mirror exactly; no sibling shares a path (each droplet sole owner of its added-or-modified file lines), and the package-shared-compile rule is honored by full serialization per `~/.claude/agents/go-planning-agent.md` HARD RULES. |
| A2 Contract-mismatch L1 AC vs L2 droplet AC | 1 CONFIRMED (medium) — finding 1.1 | All five L1 W0 AcceptanceCriteria bullets at `workflow/drop_4c_6/PLAN.md:60-64` covered (types → D1; `Resolve` → D2; local merge + `tools_deny` reject → D3; frontmatter strip → D4; `mage test-pkg` + golden fixtures → D1-D5). Function-naming/parameter-order refinement (`Merge(localRegistry, projectRegistry)` at L1 vs `MergeLocal(project, local)` at L2) is conventional, low-noise, NOT confirmed. The CONFIRMED counterexample is the intra-D3 contradiction, not the L1↔L2 surface drift. |
| A3 Hidden-coupling — cross-wave consumers named | EXHAUSTED — no counterexample | D1 types → D2/D3 named (in-W0). D2 `Resolve` → W3 named (PLAN.md:87 + :228). D3 `MergeLocal` → W3 named (PLAN.md:228). D4 `StripFrontmatterKeys` → W3 named (PLAN.md:170 + :230). D5 `ConfigError` → W11 named (PLAN.md:231). Every shipped symbol has at least one named consumer in the plan or downstream wave reference. |
| A4 YAGNI / premature abstraction | EXHAUSTED — no counterexample | Every droplet maps to a specific L1 W0 AC bullet. No interfaces with one implementation. No abstractions ahead of the consumer. The pointer-field `Override` shape is justified by the absent-vs-zero discrimination required by §4.2.1. The `mergeMaps` / `replaceList` helpers are factored once and reused across D2+D3 — DRY, not premature. |
| A5 Shipped-but-not-wired | EXHAUSTED — no counterexample | See A3 — every new symbol has a named consumer. The plan calls out W2/W3/W11 explicitly under "Out of scope for W0" (PLAN.md:228-231), preserving the cross-wave wiring chain. |
| A6 Atomicity sizing | EXHAUSTED — no counterexample | D1: types + LoadRegistry + 2 tests + 1 fixture (~120 LOC + tests, 1 production file). D2: Resolve + helpers + 5 tests + 4 fixtures (~80-100 LOC + tests, 1 production file). D3: MergeLocal + sentinel + 4 tests + 3 fixtures (~80 LOC + tests, 1 production file). D4: StripFrontmatterKeys + 6 tests (~80-100 LOC + tests, 1 production file). D5: ConfigError + WrapWithPosition + LoadRegistry/MergeLocal mods + 4 new + 2 revised tests (largest droplet — ~80-100 LOC plus 2 revised assertions; planner self-justifies the last-place ordering at PLAN.md:222 to make the revisions sit on prior tests' boundary). All within the 80-120 LOC + tests + 1-production-file till-go template budget per `~/.claude/agents/go-planning-agent.md` § "Atomic droplet sizing." D5's "revises D1+D3 prior tests" is the planner's deliberate sequencing choice (PLAN.md:222), not an atomicity violation — the prior tests' assertions get tightened, not deleted; total touched-LOC stays in budget. |
| A7 Prompt-injection (DORMANT pre-team-feature) | EXHAUSTED — gated dormant | Per `~/.claude/agents/go-qa-falsification-agent.md` § "Prompt-Injection Attack Family (Dormant Pre-Team-Feature)" — activation gate is post-MVP team feature, not yet landed. No attack surface to exercise. |
| Phase-2-step-1 required: missing `blocked_by` | EXHAUSTED | Every D2-D5 has explicit `Blocked by:` bullet to its predecessor in the serial chain; D1 has `Blocked by: —` (root of W0 chain) which is correct because W0 sub-plan itself sits at L1 with `Blocked by: —`. |
| Phase-2-step-1 required: cycles in `blocked_by` | EXHAUSTED | Linear chain D1→D2→D3→D4→D5; no cycles. |
| Phase-2-step-1 required: `_BLOCKERS.toml` ↔ PLAN.md drift | EXHAUSTED | PLAN.md (truth) says: D1=—, D2=D1, D3=D2, D4=D3, D5=D4. `_BLOCKERS.toml` records 4 `[[blockers]]` rows for D2/D3/D4/D5 each pointing to its predecessor; D1 correctly omitted (root). Mirror is exact. |

The counterexample at finding 1.1 does NOT block the plan's overall decomposition shape, the serial chain, the symbol set, the test set, or the cross-wave consumer wiring — it requires an editorial fix to D3's AC bullet 8 (drop the line-number clause OR move line-tracking infrastructure into D1) so D3's builder can stamp a self-consistent test at D3's own boundary. Once that single AC bullet is reworded, the plan converges PASS.

## 4. Hylla Feedback

N/A — review touched only Markdown plan artifacts (PLAN.md, _BLOCKERS.toml, SKETCH.md, agent definition). Hylla is Go-only today per `feedback_hylla_go_only_today.md`; no Hylla query was needed and no fallback miss to log.
