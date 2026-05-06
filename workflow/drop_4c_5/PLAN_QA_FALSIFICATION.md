# Drop 4c.5 Master PLAN — QA Falsification Round 1

**Reviewer:** go-qa-falsification-agent
**Date:** 2026-05-05
**Verdict:** NEEDS-REWORK

## 1. Attack Inventory

(Per attack categories 1-12 from the orchestrator brief. Outcome: YES = counterexample landed; NO = mitigated by plan; N/A = out-of-scope.)

- **1.1 — Cross-theme blocked_by (especially `internal/app/service.go` + `app_service_adapter_mcp.go`).** **YES.** Multiple file-level collisions across themes that the master plan does not surface. See §2 finding F1 (A.1 ↔ C.1) and F2 (B.1 ↔ C.1).

- **1.2 — Cycles in blocked_by graph.** **NO.** I extracted every blocked_by edge across Chain 1-4 plus cross-chain edges (A.2→A.1, F.1.1→F.2.1, F.1.2→F.1.3, F.2.4→F.2.1/F.2.2, F.3.1→F.2.1/F.2.2/F.1.2, F.3.2→F.5.1, F.3.3→F.1.2, E.6→F.1.3 — already serial, F.5.1→E.6 — already serial). No cycle exists. Master plan §"Cross-Theme Blocked-By Justifications" + per-chain ordering is acyclic.

- **1.3 — Package-lock chain ordering invalid.** **YES (partial).** See §2 finding F3 — F.1.2 in Master Chain 1 has `blocked_by: F.1.1, F.1.3` listed correctly in the table, but the master plan's **Chain 4 ordering shows F.1.3 lands at slot 3** (after F.2.1, F.2.2). Chain 1 reaches F.1.2 only at slot 10 (after F.6.1 → F.1.1). So F.1.3 lands well before F.1.2 — ordering valid. However, the master Chain 1 row for **F.2.4** lists `blocked_by: F.1.2, F.2.1, F.2.2` and OMITS the direct dependency on **F.1.3** that Theme F's per-droplet acceptance requires (F.2.4 calls `LoadDefaultTemplateForLanguage`, which lands in F.1.3). F.1.3 is satisfied transitively via F.1.2 → F.1.3 in the dep graph, so the chain order is correct, but the explicit edge is missing for traceability.

- **1.4 — Q1-Q5 resolutions missed counterexamples.** **NO with one risk.** Q4 ("adapter-stamp is sufficient") explicitly carves out a future dispatcher direct-call path. Theme A planner's Q-A-4 also flags this. The CURRENT dispatcher dogfood loop (per `feedback_drop_orch_spinup_discipline.md`) provisions auth via the `till` CLI, so adapter-stamping covers it. Q1 (defer FE), Q2 (warn-only), Q3 (correctness mandatory), Q5 (A+B mandatory) all hold under the evidence. The one residual risk: **Q4's "out of scope for 4c.5" depends on dispatcher implementation NOT changing during the drop**. If F.7's spawn pipeline (already shipped Drop 4c) gains a non-CLI auth path mid-drop, Q4's resolution leaks. Master plan's mitigation: tracked as "Drop 4d / Drop 5 follow-up" — sufficient for pre-MVP.

- **1.5 — Cross-cutting design decisions wrong (E.6 post-decode canonicalization).** **NO.** The decision is well-reasoned (matches `domain.IsValidKind`'s case-fold tolerance, surfaces `[gates.BUILD] AND [gates.build]` collision via post-canonicalization detection). Theme C+E planner's Note 2 explicitly defends it. Hidden cost surfaced: **signature change `validateMapKeys(tpl Template)` → `validateMapKeys(tpl *Template)` ripples to caller at `load.go:125`**. Master plan §"Cross-Theme Cross-Cutting Decisions" calls this out. Mitigated.

- **1.6 — Scope creep / YAGNI (F.5.2 reachability "vacuously true").** **NO with documented tradeoff.** Theme F planner Note 1 admits `validateChildRuleReachability` is vacuously true on the embedded default. The droplet earns its keep on adopter templates that strip child_rules — a real future case post-MVP marketplace (F.4 → Drop 4d-prime). REVISION_BRIEF §3.6 lists it explicitly in scope; planner's stance ("ship the validator anyway; the alternative loses ground later") is defensible. `validateKindStructuralCoherence` adds the THIN cross-axis wedge — also planner-defended as a wedge for future drops. NOT a falsification BLOCKER, but I note it as a mitigated tradeoff in §3.

- **1.7 — Wave A feasibility (7 parallel droplets).** **YES.** Master Plan §"Wave Structure" claims Wave A foundations are "no cross-chain blockers; all `blocked_by: —`". The list includes A.1, C.1, C.4, D.1, E.1, F.2.1, AND F.2.3. The plan immediately self-corrects ("F.2.3 ... blocked_by F.2.1 since it copies the rebadged content; lands in Wave B"). Editorial inconsistency — F.2.3 should NOT be listed under Wave A. Additionally see F1: A.1 and C.1 collide on `app_service_adapter_mcp.go` so they cannot truly parallelize. Wave A is **really 5 droplets**, not 7.

- **1.8 — Hidden dependencies via shared types.** **NO (mitigated).** A.1 changes `UpdateActionItemInput` to pointer-sentinel for Title/Description/Priority/DueAt/Labels. `rg UpdateActionItemInput` shows 4+ TUI construction sites in `internal/tui/model.go` (lines 6106, 8045, 8591, 11635, 19840), `internal/tui/thread_mode.go:510`, `internal/app/service_test.go` (3 sites), `internal/app/kind_capability_test.go:902`, `internal/adapters/server/common/app_service_adapter_mcp.go:855`, plus `internal/tui/model.go:65` (interface). Theme A planner mitigation #3 catches this: "compile error surfaces every call site." Mitigated by Go's type system but the `internal/tui` work is uncounted in the Theme A surface. Builder MUST touch every site or the build breaks.

- **1.9 — Test fixture / golden file collisions.** **NO.** No droplet touches shared golden files. F.2.1 byte-copies `default.toml` → `default-go.toml`, but the golden tests in `embed_test.go` are renamed in the same droplet. F.2.2 ships a NEW file. F.2.3 ships a NEW repo-root file. Independent.

- **1.10 — Blocker-graph correctness via tooling.** **NO.** Built the DAG mentally from per-chain tables + cross-chain justifications; no cycle. Some edges are listed transitively-only (F.2.4 should also list F.1.3 directly per §1.3 above), but that's a redundancy gap, not a correctness gap.

- **1.11 — Cross-theme service.go collision (F.6.1 chain ordering vs Theme F Note 9 "parallel-safe").** **NO.** Master plan correctly serializes F.6.1 in Chain 1 between E.8 and F.1.1. Theme F planner Note 9 says F.6.1 is parallel-safe IFF the orchestrator's file-lock manager handles multi-file atomic edits. The dispatcher's lock manager (Drop 4a Wave 2) DOES handle file-level locks; package-level locks add another serialization point. Either way, master plan's choice (serialize) is conservative-safe. Theme F planner's "parallel-safe" claim is correct only under a specific lock-manager configuration, and chain serialization is the safer pre-cascade default.

- **1.12 — Builder spawn-prompt completeness.** **YES.** Master Plan §"Pre-MVP Rules" enumerates: opus, filesystem-MD mode, no-commit directive, single-line conventional commits, never raw `go test`/`mage install`, Section 0 reasoning. **However, master plan does NOT explicitly tell the builder which Theme MD to read.** Per droplet-row in master Chain 1-4 tables, the `Source PLAN MD` column points at the per-theme MD — that's good. But there is no spawn-prompt template showing how to compose: REVISION_BRIEF + theme MD + droplet-specific section. A builder spawned for E.6 needs to read THEME_CE_PLAN.md §"E.6" PLUS the Master Plan's E.6 row PLUS REVISION_BRIEF §3.5 4b R1. Master Plan should add a one-line spawn-prompt pattern under "Pre-MVP Rules." See finding F4.

## 2. Counterexamples (CONFIRMED BLOCKERS)

### F1 — A.1 and C.1 share `internal/adapters/server/common/app_service_adapter_mcp.go`

- **What breaks.** Master Plan Chain 1 lists A.1 as the head of Chain 1 with `blocked_by: —`. Master Plan §"Independent" lists C.1 with `blocked_by: —`. Master Plan §"Wave Structure" puts BOTH in Wave A as "parallel-launches." But A.1's per-theme description (THEME_A_PLAN.md line 46) edits `internal/adapters/server/common/app_service_adapter_mcp.go` (the adapter-side `UpdateActionItem` mapping at line 855), and C.1 (THEME_CE_PLAN.md line 14-15) edits the same file (`assertOwnerStateGateUpdateFields` body + call site at line 845-852).
- **Reproduction trace.** `rg -n UpdateActionItemInput internal/adapters/server/common/app_service_adapter_mcp.go` → line 855. `rg -n assertOwnerStateGateUpdateFields internal/adapters/server/common/app_service_adapter_mcp.go` → lines 820-852. Both droplets edit lines 845-855 area of the same 2252-line file.
- **Recommended fix.** Add `C.1 blocked_by: A.1` (or `A.1 blocked_by: C.1`) to Master Plan. The package `internal/adapters/server/common` has BOTH droplets touching it; per CLAUDE.md "Paths and Packages" rule, sibling builds sharing a package MUST have explicit `blocked_by`. Master Plan should also fix its Wave A list to remove this implicit parallelization.

### F2 — B.1 also touches `app_service_adapter_mcp.go` (Theme C+E planner flagged this; master missed it)

- **What breaks.** B.1 (THEME_BD_PLAN.md line 28) adds `SupersedeActionItem` MCP-adapter passthrough method to `internal/adapters/server/common/app_service_adapter_mcp.go`. Master Chain 1 sequences B.1 after A.4. C.1 is "Independent" with no blocker to B.1.
- **Reproduction trace.** THEME_BD_PLAN.md line 28: "internal/adapters/server/common/app_service_adapter_mcp.go — add MCP-adapter passthrough method `SupersedeActionItem`...". THEME_CE_PLAN.md line 14: "internal/adapters/server/common/app_service_adapter_mcp.go (function body + call site at line 845-852)". B.1 + C.1 both edit the same file with no `blocked_by` between them.
- **Recommended fix.** Either chain C.1 into Chain 1 (as `blocked_by: B.1` between B.1 and C.2), or add explicit `B.1 blocked_by: C.1` (since C.1 has no other deps and lands fast) so C.1 lands in Wave A and B.1 in Wave B. Theme C+E planner already flagged this risk explicitly: "Master synthesizer should also flag any cross-theme collisions (e.g. Theme A may also touch `internal/app/service.go` → C.2 / C.3 / E.8 / E.9 may need cross-theme blockers)" (THEME_CE_PLAN.md line 453). Same logic applies to the adapter file.

### F3 — Master Chain 1 row for F.2.4 omits direct `blocked_by: F.1.3`

- **What breaks.** Theme F's per-droplet F.2.4 acceptance: "Blocked by: F.1.3, F.2.1, F.2.2." Master Plan Chain 1 row for F.2.4 lists `Blocked by: F.1.2, F.2.1, F.2.2`. F.1.3 is reachable transitively (F.1.2 → F.1.3) so the actual ordering is correct. But the master table loses traceability for the direct edge.
- **Reproduction trace.** THEME_F_PLAN.md line 277: "**Blocked by:** F.1.3, F.2.1, F.2.2." Master Plan Chain 1 table line 62: "F.2.4 ... Blocked by F.1.2, F.2.1, F.2.2".
- **Recommended fix.** Update Master Plan Chain 1 row for F.2.4 to read `Blocked by: F.1.2, F.1.3, F.2.1, F.2.2` (explicit) or document the transitive reasoning inline so future drop-orchs reading the plan don't have to chase the per-theme MD. Minor severity but a traceability gap.

### F4 — No spawn-prompt template in Master Plan

- **What breaks.** The Master Plan's Pre-MVP Rules section (§"Notes" subsection) lists requirements (opus, no-commit, Section 0) but does not show a spawn-prompt template. The builder for, say, E.6 needs three sources: (1) THEME_CE_PLAN.md §"E.6" full droplet spec, (2) Master Plan's E.6 row + cross-cutting decision §"Cross-Theme Cross-Cutting Decisions" line E.6 ("post-decode canonicalization NOT exact-match"), (3) REVISION_BRIEF §3.5 4b R1. Without a template, drop-orchs may forget to chain the cross-cutting-decision into the prompt — leading the builder to default to exact-match.
- **Reproduction trace.** Master Plan §"Notes / Pre-MVP Rules" lines 170-173. No template shown. Each per-theme MD has its own §References but no aggregated spawn-prompt example.
- **Recommended fix.** Add to Master Plan a one-paragraph "Spawn-prompt content for any droplet" template:
  - "Spawn the appropriate go-builder-agent with `model: opus`. Provide REVISION_BRIEF.md, the per-theme PLAN MD with the droplet's section highlighted, and the Master Plan §`Cross-Theme Cross-Cutting Decisions` row for the droplet (if any). Include 'do NOT commit' directive. Builder reads droplet's `Falsification mitigations` section before any code edits."

## 3. Mitigated Attacks

- **A.1 wire-shape interaction with A.2 strict decoder.** Theme A planner Q-A-1 raises this directly; Master Plan §"Cross-Theme Blocked-By Justifications" (line 109) wires `A.2 → A.1` cross-chain. Mitigated.
- **A.4 strict-failure-outcome-enum check.** Master Plan §"Cross-Theme Cross-Cutting Decisions" (line 39) explicitly INCLUDES the switch on `outcome ∈ {failure, blocked, superseded}`. Mitigated.
- **E.6 post-decode canonicalization vs exact-match.** Master Plan defends with 4-bullet rationale (line 40); Theme C+E planner §"E.6 fix-path decision" backs it. Plan-QA may flip if reasoning is rejected, but rationale is sound.
- **E.9 placement (`internal/utils/` vs `internal/platform/`).** Master Plan picks `internal/platform/gitenv` per CLAUDE.md "Project Structure" guidance (line 41). Mitigated.
- **F.6.1 vs Theme F Note 9 "parallel-safe" claim.** Master Plan serializes F.6.1 in Chain 1 between E.8 and F.1.1 (file-lock conservative). Mitigated even though Theme F planner suggests parallel-safe.
- **TUI breakage from A.1 pointer-sentinel shape change.** Theme A planner mitigation #3 catches it via compile-error surface; builder MUST grep + update every TUI site (model.go × 4, thread_mode.go × 1). Type system enforces it. Mitigated by Go semantics.
- **F.5.2 reachability "vacuously true".** Theme F planner Note 1 documents the tradeoff and defends shipping. REVISION_BRIEF §3.6 explicitly lists it. Mitigated by scope authority.
- **B.1 supersede cascading-children attack.** Theme B+D planner §3.1 explicitly resolves: NO cascade. Documented in droplet acceptance + falsification mitigations. Mitigated.
- **Wave A claim's F.2.3 listing.** Master Plan self-corrects inline ("F.2.3 lands in Wave B") — editorial inconsistency only, not a correctness blocker. Mitigated by self-correction.

## 4. Conclusion

**Verdict: NEEDS-REWORK.**

Master plan has TWO unmitigated cross-theme file collisions on `internal/adapters/server/common/app_service_adapter_mcp.go` (F1: A.1 ↔ C.1; F2: B.1 ↔ C.1) that violate the CLAUDE.md "Paths and Packages" rule: "sibling `build` action items sharing a file in `paths` OR a package in `packages` MUST have an explicit `blocked_by`." Theme C+E planner explicitly flagged this risk at line 453 of THEME_CE_PLAN.md and the master synthesizer missed acting on it.

Two minor traceability gaps (F3: F.2.4 missing direct edge to F.1.3; F4: no spawn-prompt template) should be closed during the same revision pass.

**Required actions:**

1. Add `C.1 blocked_by: B.1` (or chain C.1 in front of B.1 with `B.1 blocked_by: C.1`) — closes F1 and F2 simultaneously.
2. Update Master Plan's Wave A list to exclude C.1 (since it'll now serialize after A.1) and remove the F.2.3 misclassification.
3. Update Master Plan Chain 1 row for F.2.4 to show explicit `blocked_by: F.1.2, F.1.3, F.2.1, F.2.2`.
4. Add a spawn-prompt content template to Master Plan §Notes.

Falsification round 1 should be re-run after these fixes to confirm no new collisions surface.
