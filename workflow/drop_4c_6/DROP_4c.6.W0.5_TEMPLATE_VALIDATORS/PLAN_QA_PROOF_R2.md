# Plan-QA-Proof Round 2 — DROP_4c.6.W0.5

**Reviewer:** L2 plan-QA-proof (Round 2)
**Target:** `workflow/drop_4c_6/DROP_4c.6.W0.5_TEMPLATE_VALIDATORS/PLAN.md` (revised; 6 droplets D1-D6)
**Round-1 verdict context:**
- `PLAN_QA_PROOF.md` — Round 1 PASS (proof side)
- `PLAN_QA_FALSIFICATION.md` — Round 1 PASS-header with 3 medium findings (FF1 TOML-line disclosure, FF2 D2/W1.D1 misattribution, FF3 sorted-key DFS) + low-severity context items 1.3/1.4/1.6/1.7/1.8 + 4 PASS rows (1.9 / 1.10 / 1.11 / 1.12) + EXHAUSTED 1.13.
**Inputs cross-checked:**
- L1 contract: `workflow/drop_4c_6/PLAN.md:71-85` (W0.5 sub-plan container) + `:95-100` (W1.D1 acceptance — for FF2 verification)
- Sketch source: `SKETCH.md` § 26.W0.5
- Sibling ledger: `workflow/drop_4c_6/DROP_4c.6.W0.5_TEMPLATE_VALIDATORS/_BLOCKERS.toml`
- Existing chain reality: `internal/templates/load.go:462-580` (canonicalizeMapKeys + validateChildRuleCycles + formatCyclePath); `:472` for TOML-line gap anchor; `:553` for non-deterministic map iteration anchor.

## 1. Round-1 Findings Verification

### FF1 — TOML-line disclosure gap on each of D1-D6 — VERDICT: FIXED

**Round-1 claim:** L1 W0.5 acceptance bullet 1 (`workflow/drop_4c_6/PLAN.md:78`) requires "TOML-line pointers"; the existing chain at `internal/templates/load.go:472` cannot deliver this for post-decode validators because `pelletier/go-toml/v2` only emits position-aware `DecodeError` during the decode pass. None of D1-D6's acceptance bullets propagated the hedge.

**Round-2 evidence — per-droplet verification:**

| Droplet | Warning ContextBlock | `:472` cite | Field-path mitigation |
| ------- | -------------------- | ----------- | --------------------- |
| D1      | line 80              | yes         | "field-path (e.g., `agents.<kind>` or `agents.totally-bogus`)" |
| D2      | line 114             | yes         | "field-path (e.g., `agent_bindings.<kind>.agent_name = \"no-such-agent\"`)" |
| D3      | line 147             | yes         | "cycle-path rendering (`kindA -> kindB -> kindA [parent->child]`)" |
| D4      | line 181             | yes         | "path rendering (`k0 -> k1 -> ... -> k6`)" |
| D5      | line 214             | yes         | "path rendering mirrors D3's `formatCyclePath` shape (`kindA -> kindB -> kindA [blocked_by]`)" |
| D6      | line 249             | yes         | "consumer-identifier string (e.g., `unknown_consumer`)" |

All 6 D-blocks contain a `warning` (high) ContextBlock that (a) cites `pelletier/go-toml/v2`'s post-decode validator limitation, (b) names `internal/templates/load.go:472` as the verified anchor, (c) declares the L1 W0.5 acceptance bullet 1 "TOML-line pointer" wording aspirational pending an upstream API extension, (d) provides per-droplet field-path mitigation prose. Disclosure floor reached at acceptance level (RiskNote-equivalent ContextBlock semantics). **FIXED.**

### FF2 — D2 misattribution to W1.D1 — VERDICT: FIXED

**Round-1 claim:** D2 RiskNote asserted "(W1.D1 explicitly carries the validator-rewire in its own droplet acceptance)"; cross-checked at `workflow/drop_4c_6/PLAN.md:95-100` (W1.D1's 5 acceptance bullets) — none of those bullets mention rewiring D2's default `lookupFn`. Falsification recommended either (a) redesign D2's default to walk `embed.FS` unconditionally, or (b) add the rewire as an explicit W1.D1 acceptance bullet.

**Round-2 evidence — option (a) chosen:**

- AC bullet 3 (line 91): "D2's default `lookupFn func(name string) bool` walks the embedded `internal/templates/builtin/agents/{till-gen,till-go,till-gdd}/<name>.md` FS unconditionally — it does NOT depend on W1.D1 rewiring anything. The walker is self-contained ... Pre-W1.D1, the FS contains no `*.md` files at those paths, so the default walker returns false for every name ... Post-W1.D1, the same default walker finds the real placeholder files W1.D1 ships and resolves real agent_name references at Load. **No rewire from W1.D1 is needed — the validator code is final on D2 land.**"
- RiskNote line 105 reinforces: "Default `lookupFn` walks `embed.FS` unconditionally; no rewire from W1.D1 needed." Cross-references W1.D1's 5 acceptance bullets at `workflow/drop_4c_6/PLAN.md:95-100` and asserts "none is 'rewire D2's default lookupFn' — and none needs to be."
- Warning ContextBlock line 112: "pre-W1.D1 the default `lookupFn` walks the embed.FS path that contains no agent `*.md` files yet ... The validator code itself does not change between pre-W1.D1 and post-W1.D1 — only the FS contents change."

The misattribution is replaced by an explicit decoupling claim grounded in the actual W1.D1 acceptance set. The validator's production code path is exercisable at D2 land time (via injected `lookupFn`); post-W1.D1 the same default walker resolves real names automatically because W1.D1 ships files into the FS path the walker already queries. **FIXED.**

### FF3 — Sorted-key DFS determinism omission across D3/D4/D5 — VERDICT: FIXED

**Round-1 claim:** SKETCH §26.W0.5 RiskNote mandates "Cycle-detection determinism: sorted-key traversal for reproducible error messages." Existing `validateChildRuleCycles` at `internal/templates/load.go:553` uses `for node := range graph` (Go-map non-deterministic). D3 / D4 / D5 inherit the bug without acceptance-bullet remediation.

**Round-2 evidence — D3 lands the fix; D4/D5 inherit via blocked_by chain:**

- D3 AC bullet 6 (line 128): "**Cycle-DFS shared helper iterates root-set in sorted-key order.** D3 lands a private generic helper `dfsDetectCycle[K comparable](graph map[K][]K) (cyclePath []K, found bool)` that sorts the input graph's root keys (`sort.Strings` over `[]string` projection of `domain.Kind` keys) before iterating. Reproducible error messages — the same cyclic graph produces the same `cyclePath` rendering across runs / OSes / Go map-iteration orderings. The current `validateChildRuleCycles` at `internal/templates/load.go:553` uses `for node := range graph` (non-deterministic Go map iteration); D3 fixes that as part of the helper extraction. D4 and D5 reuse the same helper via the blocked_by chain (D4 blocked_by D3, D5 blocked_by D4) and inherit the sorted-key determinism for free."
- D3 ContextBlock constraint (high) line 146: "cycle-DFS shared helper iterates root-set in sorted-key order ... The existing `validateChildRuleCycles` at `internal/templates/load.go:553` uses non-deterministic `for node := range graph` and is fixed as part of D3's helper extraction. Same helper used by D3/D4/D5 — D3 lands the helper; D4/D5 call into it via the blocked_by chain."
- D4 inheritance — KindPayload (line 182): "reuses graph helper D3 produces"; RiskNote (line 175): "graph-build helper that D3 produces (or D4 refactors out of D3) is shared between D3 and D4."
- D5 inheritance — RiskNote (line 207): "Builder DOES NOT copy-paste — extracts the colored-DFS into a private generic helper `dfsDetectCycle[K comparable](graph map[K][]K) (cyclePath []K, found bool)` and reuses it across D3 + D5."; ContextBlock constraint (high) line 212: "D5's DFS pattern matches D3's colored-DFS; builder extracts a shared private helper rather than copy-pasting."

The fix is anchored at D3 (chain entry point for the DFS pattern) with explicit `:553` non-determinism callout, the helper signature spelled out (`dfsDetectCycle[K comparable]`), and the inheritance mechanism (blocked_by chain) made explicit on D4 and D5. The existing chain's bug is repaired in the same droplet that introduces the shared helper. **FIXED.**

### Round-1 Findings Verification Summary

| Round-1 finding | Severity | Round-2 verdict |
| ---- | ---- | ---- |
| FF1 — TOML-line disclosure gap (per D1-D6) | medium | FIXED |
| FF2 — D2 W1.D1 misattribution | medium | FIXED |
| FF3 — sorted-key DFS determinism (D3/D4/D5) | medium | FIXED |

All 3 medium findings applied at acceptance / RiskNote / ContextBlock level with concrete anchors in the live source tree (`load.go:472` and `load.go:553`).

## 2. Fresh Findings

Fresh 5-axis pass on the revised plan per `~/.claude/agents/go-qa-proof-agent.md`: atomic-decomposition / parallelization-graph / specify-block-well-formedness / multi-level-decomposition / shipped-but-not-wired.

- 2.1 [Axis: atomic-decomposition] [severity: low] D3's scope grew in Round 2 to include "extract a shared `dfsDetectCycle[K comparable]` generic helper" + "fix the existing `:553` non-determinism" + "extend the cycle DFS to walk `BlockedByParent` edges" + 4-row table-driven test + 2 fixtures. Plan estimate at line 258 is "~30-80 LOC validator + ~30-50 LOC test + ~10-20 LOC TOML" (70-150 LOC total). The helper-extraction plus the determinism fix is small (~10-15 LOC delta on top of the existing 60-LOC `validateChildRuleCycles` body), and the additional fixture is modest (~20 LOC TOML). D3 still lands within the project CLAUDE.md atomic budget (80-120 LOC + tests), but it's now at the upper boundary. → evidence: `internal/templates/load.go:517-561` is 45 LOC for the existing cycle validator; helper extraction adds ~10 LOC; sorted-key fix adds ~3 LOC; edge-type wrapper adds ~5-10 LOC. Total D3 production diff ~60-70 LOC. Test code ~50 LOC across 4 rows. Two fixtures ~30 LOC. Total ~140-150 LOC + tests — at the upper bound. → fix_hint: log as low-severity context for build-QA atomicity check; NOT a blocker. If D3's diff overshoots the budget at build time, the build-QA falsifier flags it via the existing atomicity rule.

- 2.2 [Axis: specify-block-well-formedness] [severity: low] D3's ContextBlock list (lines 141-147) carries 7 entries (reference + reference + decision + constraint + warning + constraint + warning). The constraint duplication (line 144 = "preserve the colored-DFS pattern" + line 146 = "sorted-key root iteration") is a partial split — the constraint at line 146 explicitly inherits the colored-DFS pattern from line 144 by referencing "extracted into a private generic helper." Reviewer might claim a single merged constraint block would read cleaner. → evidence: D3 ContextBlocks at lines 141-147; the split is intentional (line 144 = pattern preservation; line 146 = determinism / helper-extraction wiring) but the two constraints share the same scope (the cycle DFS helper). → fix_hint: REFUTED — the split is fine; one constraint covers the pattern (colored-DFS, the established style at load.go:527-549), the other covers the determinism + extraction wiring. Two distinct invariants on the same target. No fix.

- 2.3 [Axis: parallelization-graph] [severity: PASS] Chain D1→D2→D3→D4→D5→D6 is strictly linear; `_BLOCKERS.toml` matches PLAN.md inline `Blocked by:` bullets at lines 66, 98, 128, 161, 193, 226. All 6 droplets share `internal/templates/load.go` + `internal/templates/load_test.go` + `internal/templates` package, so the serial chain is mandatory per project CLAUDE.md § "Blocker Semantics." → evidence: `_BLOCKERS.toml:23-46` enumerates 5 explicit blocker rows D2→D1, D3→D2, D4→D3, D5→D4, D6→D5 (D1 has no blocker). Cross-walk between PLAN.md and `_BLOCKERS.toml` clean. → fix_hint: nothing to fix. PASS.

- 2.4 [Axis: multi-level-decomposition] [severity: PASS] Each droplet ships ~30-80 LOC of validator code per the plan estimate (line 258); no droplet exceeds the atomic budget enough to warrant L3 sub-planning (see 2.1 caveat for D3's borderline case, but still single-level). The L1 W0.5 contract (`workflow/drop_4c_6/PLAN.md:71-85`) decomposes cleanly into 6 droplets, one per validator, matching the L1 spawn directive verbatim. → evidence: chain order D1=kind-enum / D2=agent_name / D3=cycles / D4=recursion-depth / D5=blocked_by / D6=claim-vs-impl matches the L1 spawn directive at `workflow/drop_4c_6/PLAN.md:85` ("kind-enum + agent_name first ... then cycles, then recursion-depth, then `blocked_by` acyclicity ... then claim-vs-impl last"). → fix_hint: nothing to fix. PASS.

- 2.5 [Axis: shipped-but-not-wired] [severity: PASS] D6's empty `knownWiredConsumers` set is L1-grounded (`workflow/drop_4c_6/PLAN.md:81` acceptance bullet 4); D5's degenerate-graph forward-looking validator is L1-mandated (`workflow/drop_4c_6/PLAN.md:78` acceptance bullet 1 "blocked_by acyclicity"); D6's LOUD WARNING doc-comment + W7/W8 hook are in scope (D6 RiskNote line 242). The shipped-but-not-wired anti-pattern is the EXACT thing D6 is designed to prevent — and the plan ships the floor (sentinel + test + scaffolding) explicitly to enable W7/W8 to wire the first real producers. → evidence: D6 acceptance bullet 4 (line 226) + RiskNote line 239 + LOUD WARNING line 242 + L1 anchor `workflow/drop_4c_6/PLAN.md:907-908` cited via D6's RiskNote bullet on "Future drops MUST update the known-wired set." → fix_hint: nothing to fix. PASS.

## 3. Missing Evidence

None. Round-2 fixes carry the load-bearing source-tree anchors (`internal/templates/load.go:472` for FF1, `:553` for FF3, `workflow/drop_4c_6/PLAN.md:95-100` for FF2). Cross-references between droplets (D3 helper → D4/D5 inheritance) are explicit in both PLAN.md prose and `_BLOCKERS.toml` blocker rationale. No proof gaps that block PASS.

## 4. Summary

**Verdict: PASS**

| Axis                              | Result | Notes |
| --------------------------------- | ------ | ------------------------------------------------------------- |
| Atomic decomposition              | PASS   | All 6 droplets within atomic budget; D3 borderline (2.1) — log only. |
| Parallelization graph             | PASS   | Strictly serial chain D1→D2→D3→D4→D5→D6; `_BLOCKERS.toml` matches PLAN.md inline. |
| Specify-block well-formedness     | PASS   | All 6 droplets carry Objective / AcceptanceCriteria / ValidationPlan / RiskNotes / ContextBlocks / KindPayload; constraint-split in D3 (2.2) is intentional. |
| Multi-level decomposition         | PASS   | One droplet per L1-mandated validator; chain order matches L1 spawn directive verbatim. |
| Shipped-but-not-wired             | PASS   | D5/D6 forward-looking scaffolding L1-grounded with explicit W7/W8 hook + LOUD WARNING. |
| Round-1 finding verification      | PASS   | FF1 / FF2 / FF3 all FIXED with concrete source-tree anchors. |

**Counterexample count: 0 hard.** 2 low-severity context findings (2.1 atomicity-borderline on D3, 2.2 constraint-split style) logged for build-QA awareness; neither blocks dispatch.

Round-1 verdict was PASS-with-medium-drift on the falsification side; Round 2 closed all three medium drift findings with anchored evidence in the live source tree. Round-2 fresh pass identifies no new hard findings. The L2 plan is clean for build dispatch.

## 5. Hylla Feedback

N/A — review touched non-Go files (PLAN.md, PLAN_QA_FALSIFICATION.md, _BLOCKERS.toml) plus narrow line-range pulls from `internal/templates/load.go` to verify the `:472` and `:553` anchors. Hylla today indexes Go only, but the questions on this review were "is the round-2 plan revision correctly cited against the live source tree" — which is a line-anchor verification, not a symbol-graph navigation. Read tool was the right primary; no Hylla query was warranted and no fallback miss to log.
