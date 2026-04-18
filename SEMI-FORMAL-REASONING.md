# Semi-Formal Reasoning — Section 0 Response Shape

This document is the **canonical specification** for the Section 0 semi-formal reasoning scaffold used by every substantive response in Tillsyn-coordinated projects. It is language-agnostic, orchestration-model-agnostic, and intended to travel with any project that adopts Tillsyn as its coordination runtime.

The shape is an adaptation of the reasoning certificate from arxiv 2603.01896 ("Agentic Code Reasoning," Ugare & Chandra, Meta, 4 Mar 2026), extended with:

- Two additional first-class fields — **Evidence** and **Unknowns**.
- An explicit multi-role self-review loop the paper does not include. This targets the paper's §4.3 residual failure mode: *"elaborate but incomplete reasoning chains ... leading to a confident but wrong answer."* A single writer can converge confidently on a wrong answer; a dedicated adversarial pass is the hedge.

## Section 0 Structure

Every substantive response (anything beyond a trivial one-line answer or factual lookup) begins with a `# Section 0 — SEMI-FORMAL REASONING` block, then the normal response body in the `tillsyn-flow` numbered format.

Section 0 contains named passes as `##` subsections, in order.

**Orchestrator-facing responses — 5 passes:**

- `## Planner` — frame the goal, gather evidence, enumerate open questions.
- `## Builder` — construct the proposed answer, design, or edit.
- `## QA Proof` — verify every claim is backed by evidence and the trace covers every case.
- `## QA Falsification` — actively attack the proposal via counterexamples, hidden dependencies, contract mismatches, YAGNI pressure, memory-rule conflicts. Each attack either mitigates or is explicitly accepted.
- `## Convergence` — declare: (a) QA Falsification produced no unmitigated counterexample, (b) QA Proof confirmed evidence completeness, (c) remaining Unknowns are explicit and routed. If any of (a)/(b)/(c) fail, loop back to the earliest pass that needs re-work before declaring Convergence.

**Subagent-facing responses — 4 passes:**

- `## Proposal` — the subagent's specialized role already defines the "framing" half, so Planner and Builder collapse into one Proposal pass.
- `## QA Proof`
- `## QA Falsification`
- `## Convergence`

## The 5-Field Certificate

Each pass uses the 5-field certificate where applicable. Not every pass needs all five, but the bundle as a whole must cover all five before Convergence:

- **Premises** — what must be true.
- **Evidence** — grounded in Hylla / `git diff` / Context7 / `go doc` / gopls / MDN / CanIUse / cited papers. Not implicit background.
- **Trace or cases** — concrete paths through the reasoning.
- **Conclusion** — the claim.
- **Unknowns** — what remains uncertain, routed as a Tillsyn comment / handoff / attention item or explicitly accepted.

## Body After Section 0

After `# Section 0` closes, the response body uses the `tillsyn-flow` output style unchanged (`## 1. Section`, `- 1.1`, `## TL;DR`, `TN`). **Section 0 precedes the numbered body; it does not replace it.** `tillsyn-flow` remains the canonical source for body format.

## Trivial-Answer Carve-Out

One-line factual lookups, terse confirmations, and simple yes/no answers skip BOTH Section 0 AND the numbered body. The rule prevents premature judgment on substantive work — it is not ceremony for small answers.

## Greedy-Decoding Compatibility

The multi-role loop is role-driven error correction, not stochastic sampling variance. Individual tokens within each pass still follow the top-1 / temperature-0 policy. The multi-role structure changes frame (planner → QA falsification), not decoding policy. No conflict.

## Subagent Pass-Through

Subagents do NOT inherit the orchestrator's `CLAUDE.md` or output style. When delegating substantive work (planning, QA, build with design judgment), the spawn prompt MUST include the directive verbatim:

> *"Render your response beginning with a `# Section 0 — SEMI-FORMAL REASONING` block containing `## Proposal`, `## QA Proof`, `## QA Falsification`, and `## Convergence` passes before your final output. Each pass uses the 5-field certificate (Premises / Evidence / Trace or cases / Conclusion / Unknowns) where applicable. Convergence must declare (a) QA Falsification found no unmitigated counterexample, (b) QA Proof confirmed evidence completeness, (c) remaining Unknowns are routed. If any fail, loop back before Convergence."*

## Tillsyn Artifact Boundary (Project Rule)

**Section 0 reasoning lives in the orchestrator-facing response ONLY.** Do NOT write Proposal / Planner / Builder / QA / Convergence pass text into Tillsyn `description`, `metadata.*`, `completion_contract.completion_notes`, closing comments, or any other Tillsyn artifact. Tillsyn stores **finalized artifacts**, not process.

Finalized closing certificates specialized to the role (builder's `## Implementation Notes`, QA's verdict + findings, planner's decomposition rationale) still go in the Tillsyn closing comment — just not the multi-pass Section 0 scaffold itself.

## Why This Shape

Unstructured chain-of-thought lets the model skip cases or make unsupported claims. The paper reports structured certificates reduce patch-equivalence errors from 78.2% → 88.8% on RubberDuckBench — roughly half the remaining errors removed by requiring explicit evidence per claim. Additional gains reported: +9-12pp on Defects4J fault localization.

The multi-role extension adds adversarial review because a single writer can converge confidently on a wrong answer. A dedicated falsification pass is the hedge against that failure mode.

The **Unknowns** field is load-bearing specifically for Tillsyn adopters: it gives every uncertainty a durable routing target (comment / handoff / attention item) instead of evaporating into optimistic completion.

Keep each pass short and inspectable — the point is auditable claims, not verbose prose.

## Adopter Requirements (Cross-Project)

Every project adopting Tillsyn as a coordination runtime MUST:

1. **Carry a pointer to this spec in the project `CLAUDE.md`.** CLAUDE.md is auto-loaded on session start; any substantive-response-producing session needs to see the Section 0 rules without extra lookup. A short pointer with the TL;DR + subagent-pass-through reminder + Tillsyn-artifact-boundary reminder is sufficient.
2. **Repeat the pointer in every worktree-root `CLAUDE.md`.** If the repo uses bare-root + worktree layout (e.g. `main/`, `drop/<N>/`), each worktree `CLAUDE.md` needs the same pointer. Worktrees boot orchestrators independently; a worktree with a stale CLAUDE.md silently loses the scaffold for any session launched from it.
3. **Activate the `tillsyn-flow` output style.** Set `outputStyle: tillsyn-flow` in `~/.claude/settings.json` (or the project-local equivalent). The output style file (`~/.claude/output-styles/tillsyn-flow.md`) carries the body format rules + Section 0 pre-block spec.
4. **Include the Section 0 directive verbatim in every subagent spawn prompt.** Subagents do NOT inherit CLAUDE.md or the output style. See "Subagent Pass-Through" above.
5. **Enforce the Tillsyn artifact boundary.** See "Tillsyn Artifact Boundary" above. Process never lands in durable Tillsyn storage.

## Bootstrap Checklist For A New Adopter Project

When standing up Tillsyn in a new project:

- Copy this file (`SEMI-FORMAL-REASONING.md`) into the project root or `docs/` directory.
- Add a short pointer section to the project `CLAUDE.md` referencing this file.
- If the project uses worktrees, ensure every worktree-root `CLAUDE.md` carries the same pointer.
- Confirm `~/.claude/settings.json` has `outputStyle: tillsyn-flow` enabled for the launching user.
- If the project delegates to subagents, ensure every `Agent` tool spawn includes the Section 0 directive verbatim.

## Related Reading

- arxiv 2603.01896 — "Agentic Code Reasoning," Ugare & Chandra, Meta (4 Mar 2026). Canonical source for the certificate shape and the §4.3 residual failure mode this doc extends.
- `~/.claude/output-styles/tillsyn-flow.md` — body format rules.
- Project `CLAUDE.md` — where the pointer to this file lives.
- Project `WIKI.md` — living best-practice snapshot; references this file from its Response Shape section.
