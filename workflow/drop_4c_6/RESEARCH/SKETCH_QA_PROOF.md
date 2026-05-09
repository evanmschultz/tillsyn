# SKETCH_QA_PROOF — Plan-QA Proof Pass on SKETCH.md v2.8.3 + Planner Draft

**Author**: go-qa-proof-agent (subagent)
**Pass**: plan-qa-proof axis — verify decomposition is well-formed, every wave's Specify block is testable, integration coverage is complete, no shipped-but-not-wired risks unmitigated
**Inputs**: `workflow/drop_4c_6/SKETCH.md` (v2.8.3 FINAL), `workflow/drop_4c_6/PROPOSED_AGENT_DRAFTS/till-go/planning-agent.md`, 9 RESEARCH deliverables
**Verdict**: **PASS** with two non-blocking quality findings (F1, F2) to absorb in a v2.8.4 micro-update or in plan-time decomposition.

---

## 1. Scope of this pass

This is a plan-QA-proof pass on a level-0 plan: the FINAL shape of Drop 4c.6 / 4c.7 / 4c.8 sketched in `SKETCH.md` v2.8.3 plus the proposed `planning-agent.md` draft that will land in Drop 4c.8. The proof axis (per `~/.claude/agents/go-qa-proof-agent.md`) is evidence-completeness on three dimensions:

1. **Atomic decomposition verification** — every wave's `AcceptanceCriteria` are testable; every Specify block is well-formed per §26 (Objective + AcceptanceCriteria + ValidationPlan + RiskNotes + ContextBlocks).
2. **Parallelization graph verification** — 4c.6 → 4c.7 → 4c.8 sequential dependencies stated correctly; no false claims of parallelism.
3. **Shipped-but-not-wired check** — every wave that closes a known shipped-but-not-wired gap (notably W7 wiring `ChildRulesFor` and W8 wiring `context.Resolve`) requires integration tests crossing schema → resolver → consumer → end-to-end.

Per the multi-level decomposition discipline encoded in `feedback_plan_down_build_up.md` and `feedback_decomp_small_parallel_plans.md`, this pass does NOT flag absence of per-wave droplet counts as a failure. Per-wave decomposition into droplets happens at planner-spawn time inside each drop, not in this sketch. The sketch's §25 explicitly defers droplet counts to plan-time.

Falsification work (counterexamples, hidden deps, contract attacks) is the sibling pass's responsibility and runs in parallel; this proof pass only verifies that the evidence supports the claim "the methodology is well-formed."

---

## 2. Decomposition shape — verified well-formed

### 2.1 Drop boundaries (§25)

The sketch locks Option A-split at §25 line 662 with three drops and clear sequential dependencies:

- **Drop 4c.6 — FOUNDATION (config + isolation)** — Waves W0, W0.5, W1, W2, W3, W5, W6.
- **Drop 4c.7 — CASCADE WIRING** — Waves W7, W8, W9, W10, W11.
- **Drop 4c.8 — DEFAULT PROMPTS + DOGFOOD** — Wave W4 (W4-A, W4-B, W4-C, W4-D).

Dependencies stated at §25 line 670: 4c.6 → 4c.7 (sequential; W3 frontmatter strip + bundle full content needed by W8 context preload). 4c.7 → 4c.8 (sequential; W7 + W8 must work for prompt drafts to assume auto-create + context preload). The dependency rationale is concrete and code-level — not hand-wavy. Verified.

### 2.2 Wave list — full coverage of §26 Specify blocks

Sixteen waves covered by named Specify blocks at §26:

| Wave | §26 line | Drop | Status |
|---|---|---|---|
| W0 | 746 | 4c.6 | well-formed |
| W0.5 | 766 | 4c.6 | well-formed |
| W1 | 789 | 4c.6 | well-formed |
| W2 | 805 | 4c.6 | well-formed |
| W3 | 831 | 4c.6 | well-formed |
| W5 | 856 | 4c.6 | well-formed |
| W6 | 875 | 4c.6 | well-formed |
| W7 | 892 | 4c.7 | well-formed |
| W8 | 913 | 4c.7 | well-formed |
| W9 | 935 | 4c.7 | well-formed (reserved-empty acceptable) |
| W10 | 948 | 4c.7 | well-formed |
| W11 | 968 | 4c.7 | well-formed |
| W4-A | 1013 | 4c.8 | well-formed |
| W4-B | 1030 | 4c.8 | well-formed |
| W4-C | 1046 | 4c.8 | well-formed |
| W4-D | 1062 | 4c.8 | well-formed |

There is no §26.W4 master block; instead W4 is decomposed at §25.3 into W4-A/B/C/D each with its own Specify block, which is the correct shape under the per-drop split. Deliberate, documented, coherent.

### 2.3 Specify-block well-formedness — per §26

Each block is required to carry five fields (Objective / AcceptanceCriteria / ValidationPlan / RiskNotes / ContextBlocks) — the same shape the planner-agent will use at decomposition time per §10.1. Every wave above carries all five fields with material content. Spot checks below.

**W0 (line 746-764)**: Objective is forward-looking and goal-shaped (line 748). AcceptanceCriteria are 6 bullets, each independently testable via `mage test-pkg ./internal/config` or golden-fixture override-merge tests (lines 750-755). ValidationPlan names exact mage targets (line 756). RiskNotes flag three concrete coupling risks (lines 757-760). ContextBlocks use the typed enum correctly (`reference` / `decision` / `constraint` with severity per §10.1.7) (lines 761-764). Well-formed.

**W3 (line 831-854)**: Most-loaded wave. Eight AcceptanceCriteria covering schema field, resolver, 3-tier priority, frontmatter strip, defense-in-depth env vars, post-render validator, doc-comment corrections, and sentinel-injection integration test. Each AC maps to a concrete code site or test target (lines 835-842). RiskNotes call out four concrete failure modes (lines 844-848). ContextBlocks include the critical-severity invariant "bundle MUST carry full body" (line 850). Well-formed.

**W7 (line 892-911)**: AcceptanceCriteria explicitly require integration tests crossing schema → resolver → consumer for the auto-create wiring (lines 898-901). Atomic-transaction acceptance criterion is coherent with `Service.CreateActionItem`'s existing transactional shape per `RESEARCH/CASCADE_ENFORCEMENT_AND_CONTEXT_PRELOAD.md §A.2`. RiskNotes correctly anchor the recursion-depth bound on W0.5's validator (line 906). Well-formed.

**W8 (line 913-933)**: AcceptanceCriteria explicitly require per-cascade-kind integration tests at line 923. New `all_peer_children` rule is correctly distinguished from existing `siblings_by_kind` (which per `RESEARCH/CASCADE_ENFORCEMENT_AND_CONTEXT_PRELOAD.md §B.1` row 8 picks "latest round per matching kind"). The semantics-distinction risk is captured at line 927. Well-formed.

**W11 (line 968-1011)**: Largest scope expansion (per v2.8.2 changelog at sketch line 14). AcceptanceCriteria split structural rejects (lines 985-992) from advisory warnings (lines 993-996). Comment role-gating added as fresh structural rule (line 991) — research-grounded per `RESEARCH/AUTENT_ARCHIVE_QA_PRIOR_PLANNING.md §A.3` confirming zero role-gating today. The `allow` policy correctly DROPPED per dev §3.1 (line 1005). RiskNotes flag role-resolution dependency on auth-context shape (line 1002). Well-formed.

**W4-A / W4-B / W4-C / W4-D**: Each carries the 5 fields. W4-A and W4-B differ on Go-specific tuning per §3.5.1 / §15. W4-C is Tillsyn's own dogfood overrides at `<repo>/.tillsyn/agents/`. W4-D is integration-test-only. All four well-formed.

### 2.4 Multi-level decomposition discipline — respected

Sketch §25 line 672 is explicit: "Droplet counts are NOT specified at this sketch level per `feedback_plan_down_build_up.md` — the planner-agent decomposes each wave into however many droplets fit the work; plan-QA verifies the decomposition is well-formed at each level."

This is the system-as-designed discipline: top-level plan authors waves with Specify blocks; per-wave plan-spawn handles atomic-droplet decomposition. The sketch correctly stays at level-0 and does NOT pre-decompose droplet counts. Per the proof axis "Multi-level decomposition: this is a level-0 plan; per-wave sub-decomposition into droplets happens at planner-spawn time, NOT in this sketch. Do NOT flag absence-of-droplet-counts as failure" — confirmed compliant.

---

## 3. Shipped-but-not-wired coverage — verified

The principle from `feedback_tillsyn_enforces_templates.md` is: every claimed deliverable must trace schema → resolver → consumer → integration test. The two highest-risk waves (W7, W8) close known shipped-but-not-wired gaps. Plus W0.5 / W11 add new "fail loud" enforcement layers.

### 3.1 W7 — auto-create wiring (closes Drop 3 droplet 3.20)

`RESEARCH/CASCADE_ENFORCEMENT_AND_CONTEXT_PRELOAD.md §A.1-A.4` confirms today's reality: schema, resolver, and TOML rules all committed and pass tests, but no production code path consumes `Template.ChildRulesFor`. The orchestrator hand-waves QA twin creation. Drop 3 droplet 3.20 was the original consumer-binding droplet that never landed.

W7 AcceptanceCriteria (sketch line 894-902):
- Schema → resolver layer: already present per research §A.1.
- **Consumer**: AC line 896 — `Service.CreateActionItem` calls `tpl.ChildRulesFor(kind)` and creates declared children atomically. Single consumer integration point identified (`internal/app/service.go:1180-1199` per research §A.2).
- **Integration tests**: AC lines 898-899 require integration tests for `kind=plan` and `kind=build` auto-firing twins. These cross schema → resolver → consumer → behavioral assertion. Verified.

**Coverage adequate for the proof axis.**

### 3.2 W8 — context-preload wiring (closes shipped-but-never-called engine)

`RESEARCH/CASCADE_ENFORCEMENT_AND_CONTEXT_PRELOAD.md §B.1-B.3` confirms: full aggregator engine at `internal/app/dispatcher/context/aggregator.go:243-453` with rules, caps, timeouts, file-vs-inline delivery, greedy-fit, markers, plus 5 per-rule resolvers in `rules.go`. Zero call sites in `render.go`. Spawn pipeline today emits a 6-field stub with no parent / ancestors / siblings content per research §B.

W8 AcceptanceCriteria (sketch line 915-923):
- Schema → engine: present per research §B.1.
- **Consumer**: AC line 917 — `render.go:assembleSystemPromptBody` calls `context.Resolve(binding, item, project)`. Single consumer integration point at `render.go:246-279`.
- **New rule**: AC line 918 — `all_peer_children` rule (schema + resolver) for plan-QA's "all children of parent plan" need per §11.1.
- **Integration tests**: AC line 923 — "spawn each cascade kind; assert system-prompt content matches §11.1 declared bundle." Per-cascade-kind coverage. Verified.

**Coverage adequate.**

### 3.3 W0.5 — load-time template validation (NEW)

W0.5 closes the "schema-shipped-but-not-validated" gap per `feedback_tillsyn_enforces_templates.md`. AcceptanceCriteria (sketch line 768-777) enumerate six concrete validators each with structured-error contract. ValidationPlan (line 777) requires "malformed-template-fixture test PER validator" — one fixture per error case. Coverage adequate.

### 3.4 W10 — wipe-and-replan (closes the failed-QA flow)

`RESEARCH/CASCADE_ENFORCEMENT_AND_CONTEXT_PRELOAD.md §D` confirms `domain.ActionItem.Archive` is single-row only; no cascade method exists. W10 AcceptanceCriteria (sketch line 951-955) require:
- New typed `metadata.failure_history` field on domain layer.
- `Service.WipeChildrenAndRePlan` with five concrete steps (collect QA findings → synthesize summary → archive atomically → append to failure_history → transition parent).
- Integration test for full failed-plan → wipe → re-plan → green cycle.

The render-layer hook for "Prior Attempt Failed" depends on W8 wiring — correctly noted in RiskNotes line 961. Cross-wave dependency surfaced. Coverage adequate.

### 3.5 W11 — runtime fail-loud at MCP boundary (NEW; v2.8.2 expansion)

W11 enforces structural separation-of-concerns at MCP boundary. The split between hardcoded structural rejects (line 985-992) and template-defined warnings (line 993-996) is well-grounded in `feedback_tillsyn_enforces_templates.md` interpretation. The five hardcoded rejects (planner-creates-QA, builder-decomposes, no-update-on-system-QA, file-write-outside-paths, no-MCP-delete) are cascade-architecture invariants, not adopter-tunables.

Comment role-gating (AC line 991) is FRESH structural rule per `RESEARCH/AUTENT_ARCHIVE_QA_PRIOR_PLANNING.md §A.3` row 9 confirming zero role-gating today. Scope is concrete and bounded.

Integration tests required per AC line 999. Coverage adequate.

---

## 4. Planner draft — verified hardening list applied

Sketch §17.1 lists nine hardening items as "APPLIED." Cross-checking against `PROPOSED_AGENT_DRAFTS/till-go/planning-agent.md`:

| Hardening item | Sketch §17.1 | Planner-draft evidence |
|---|---|---|
| Section 0 stdout-only | line 567 | line 10 ("Section 0 reasoning is your stdout output BEFORE you make any Tillsyn MCP tool call") |
| HARD RULE — Think before authoring with "STOP" wording | line 568 | line 12 ("STOP. That is a discipline violation") |
| Edit existing children allowed | line 569 | reframed to "Wipe-and-restart on plan-failed (system-managed)" at lines 153-178 — supersedes per §25.5 |
| Tools removed from frontmatter | line 570 | frontmatter at lines 1-4 has only `name` + `description` |
| `model:` removed from frontmatter | line 571 | confirmed at lines 1-4 |
| Planner does NOT transition parent to `complete` | line 572 | line 184 ("You do NOT move the parent plan to `complete`") |
| Common failure modes section | line 573 | lines 188-197, 7 failure modes enumerated |
| Karpathy four baked into Working Principles | line 574 | lines 40-45, all four principles named |
| Archive over delete for child cleanup | line 575 | line 207 ("You do not delete children — archive instead, to preserve the audit trail") |

All nine hardening items verified present in the planner draft. Sketch §17.1 claim of "APPLIED" is accurate.

### 4.1 §25.5 follow-up updates — already applied

Sketch §25.5 at line 731-735 says "Two minor updates needed" to the planner draft after Option A-split:
1. "Edit existing children (fix-up scenarios)" → reframe as "Wipe-and-restart on plan-failed (system-managed)."
2. "Planner NEVER creates, edits, or archives plan-qa-* / build-qa-* action items."

Both are ALREADY APPLIED in the current planner-draft v4 on disk:
- Item 1: planner draft lines 153-178 carry the "Wipe-and-restart on plan-failed" section with full system-managed framing including `failure_history` synthesis. The "Edit existing children" framing is gone.
- Item 2: planner draft line 208 verbatim ("You do not create, edit, or archive any QA action item"). Plus the "You NEVER" list at lines 170-176.

The §25.5 description is mildly stale; it reads as if the updates are still pending. This does not affect decomposition correctness — the planner draft is already in the desired post-Option-A shape. **Noted for sketch maintenance** (see F2 below).

---

## 5. Cross-wave dependency graph — coherent

The sketch states explicit cross-drop dependencies. Tracing the load-bearing ones:

- **W3 → W8**: W3 ships full agent body via `//go:embed` and frontmatter strip; W8 needs the bundle's substantive content to make context-preload meaningful (otherwise context lands in a stub). Correctly stated at §25 line 670.
- **W7 → W10**: W10's wipe-and-restart relies on W7's auto-create to spawn fresh QA-twins on the post-wipe fresh children. Correctly stated at §11.2 step 6 (sketch line 416) and §26.W10 ContextBlocks (line 962).
- **W8 → W10**: W10's "Prior Attempt Failed" render hook depends on W8's context-preload wiring being live in `render.go`. Correctly stated at §26.W10 RiskNotes line 961.
- **W0.5 → W7, W8**: W0.5's claim-vs-impl coherence validator must know which kinds W7+W8 wire. Stated as risk at §26.W0.5 line 782 ("`Shipped-but-not-wired` load-time check requires consumer-discovery interlock with W7 + W8 outcomes"). See F1 below for the residual gap.
- **W11 → auth context shape (Drop 4a Wave 3)**: W11's role-resolution depends on auth context carrying role + owner. Drop 4a Wave 3 delivered orch-self-approval / role-resolution per `feedback_steward_spawn_drop_orch_flow.md`. Correctly noted at §26.W11 RiskNotes line 1002.

**Cross-drop coherence is sound.** Each cross-wave dependency is either named in AcceptanceCriteria, RiskNotes, or ContextBlocks.

---

## 6. Findings (non-blocking)

These are quality-tier findings — not counterexamples to PASS. The plan is sound; these surfaced during falsification but do not invalidate decomposition correctness. They should be absorbed in a v2.8.4 micro-update or in plan-time droplet decomposition.

### F1. W7/W8 AcceptanceCriteria do NOT include "update W0.5's claim-vs-impl validator with newly-wired kinds"

W0.5 AcceptanceCriteria line 775 demands a "claim-vs-impl coherence" validator: "every claimed `[[child_rules]]` output kind is supported by cascade-tree-shape rules." W0.5 RiskNotes line 782 acknowledges the cross-drop dependency: "consumer-discovery interlock with W7 + W8 outcomes."

But neither W7's nor W8's AcceptanceCriteria mention updating W0.5's validator post-wiring. Once W7 wires `ChildRulesFor` consumer in `Service.CreateActionItem`, the set of kinds the validator considers "wired" changes. Same for W8 wiring `context.Resolve`. Without an explicit acceptance criterion in W7/W8 to update the validator's known-wired set, there's a silent-staleness risk: validator passes today (everything looks unwired so nothing trips claim-vs-impl) and silently misses future regressions.

**Recommendation**: add to both W7 and W8 AcceptanceCriteria one bullet: "W0.5's claim-vs-impl validator updated to include this wave's newly-wired consumers; integration test confirms claim-vs-impl now passes for the wired kinds and would fail-loud if the consumer regresses." Single-line addition per wave. Roughly mechanical.

### F2. §25.5 stale — planner draft updates already applied, but description reads as pending

Sketch §25.5 (line 731-735) lists two "minor updates needed" to the planner draft. Both are already applied in `PROPOSED_AGENT_DRAFTS/till-go/planning-agent.md` v4 on disk (verified §4.1 above).

**Recommendation**: in the v2.8.4 micro-update, change §25.5 from imperative ("two minor updates needed") to indicative ("two minor updates applied; verified at lines 153-178 and line 208 of the draft"). Documentation hygiene only.

---

## 7. Methodology meta-note — the sketch eats its own dogfood

Sketch §26 explicitly demonstrates SDD by authoring each wave's Specify block in the same shape the planner-agent will use at decomposition time (Objective + AcceptanceCriteria + ValidationPlan + RiskNotes + ContextBlocks). This is a self-referential demonstration: the methodology document is authored using the methodology it documents.

Consequences for this proof pass:
- The sketch's Specify-block shape IS the canonical reference for what plan-QA verifies in actual planner output.
- A flaw in the sketch's Specify-block discipline would propagate forward — every planner-spawn would inherit the same shape gap.
- Inversely, a clean sketch raises the bar for plan-time output: planners should match the sketch's level of rigor in AcceptanceCriteria testability, ContextBlocks severity tagging, and RiskNotes specificity.

Verified: the sketch's discipline is high. AcceptanceCriteria are mage-target-anchored. ContextBlocks use the typed enum. RiskNotes name specific coupling concerns. The dogfood-on-itself loop is healthy.

This is the **first cascade-flavored QA pass on a methodology-defining drop**. The methodology is being tested end-to-end on the methodology drop itself. PASS verdict carries weight beyond a normal plan-QA — it's also a self-consistency check on the methodology.

---

## 8. Verdict

**PASS** with two non-blocking quality findings (F1, F2) for sketch maintenance.

The sketch is well-formed across all three plan-QA-proof axes:
1. **Atomic decomposition**: 16 Specify blocks, all 5-field-complete, all testable.
2. **Parallelization graph**: cross-drop dependencies stated correctly with code-site grounding.
3. **Shipped-but-not-wired**: every closing-the-gap wave (W0.5, W7, W8, W10, W11) requires integration tests crossing schema → resolver → consumer → behavioral assertion.

Planner draft hardening list verified applied (9 of 9 items present in draft). §25.5 follow-up updates verified applied (description is mildly stale; F2).

Findings F1 and F2 do NOT block the plan. They should land as a v2.8.4 micro-update before the Drop 4c.6 plan-spawn, or be picked up by the plan-time planner as part of decomposition.

The sibling `plan-qa-falsification` pass runs in parallel and may surface counterexamples this proof pass did not consider; both passes must converge before the methodology-defining drop locks.

---

## Hylla Feedback

N/A — this pass reviewed methodology MD documents (sketch + planner draft + research deliverables); no Go code was queried. Hylla is Go-only today per `feedback_hylla_go_only_today.md`, so this pass appropriately used `Read` directly and did not invoke any Hylla tool. No Hylla miss to report.
