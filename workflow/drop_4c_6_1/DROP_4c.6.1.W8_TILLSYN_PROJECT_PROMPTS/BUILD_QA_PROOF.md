# W8 D1-D20 — BUILD-QA-PROOF Verdict

**Date:** 2026-05-13
**Reviewer:** go-qa-proof-agent (opus)
**Overall verdict:** PASS

## Coverage Summary

- **Files audited:** 20 / 20 (10 Go + 10 FE).
- **Signal B (frontmatter):** 20 / 20 PASS — all start with `---\n`, contain `name:`, `description:`, `model:`, `tools:` keys, and close with `---\n`.
- **Signal A (body length > 200 bytes):** 20 / 20 PASS — all bodies >> 1000 chars (range 3000-6000+ chars post-frontmatter; total file sizes 4013-6953 bytes).
- **Signal C (`## Role` header):** 20 / 20 PASS — visually confirmed in each file via Read.
- **Migration marker (verbatim ASCII):** 20 / 20 PASS — line 8 of each file carries the exact marker `<!-- Tillsyn-project-local; lifted from ~/.claude/agents/ and adapted for Tillsyn's workflow. Future projects use embedded defaults shipped in Drop 4c.8. -->` with ASCII apostrophe in `Tillsyn's`.
- **Section 0 NOT at top of file:** 20 / 20 PASS — every file's first line is `---` (frontmatter), Section 0 appears only as a directive instruction in the body for the consumer subagent to render.
- **Frontmatter model values per cascade-model-policy table:** 20 / 20 PASS — exact alignment (planning/qa/research = opus, builder = sonnet, commit-message = haiku, closeout / orchestrator-managed = orchestrator-managed).
- **Per-droplet differentiation (D3/D5, D4/D6, D13/D15, D14/D16):** 4 / 4 pairs PASS — substantively different Evidence Sources and What To Check / Attack Vectors sections.
- **Source provenance (D1-D7 + D11-D17 adapted from `~/.claude/agents/`):** 14 / 14 PASS — Tillsyn-specific specializations layered over global agent semantics.
- **FROM-SCRATCH discipline (D8/D9/D10 + D18/D19/D20):** 6 / 6 PASS — cite CLAUDE.md / WORKFLOW.md / WIKI.md; no false source-adaptation claims.

## Per-droplet coverage

| ID  | Path                                                 | Bytes | Signal A | Signal B | Signal C | Marker (ASCII) | Frontmatter model      | Section 0 placement | Differentiation       | Provenance discipline | Verdict |
| --- | ---------------------------------------------------- | ----- | -------- | -------- | -------- | -------------- | ---------------------- | ------------------- | --------------------- | --------------------- | ------- |
| D1  | `.tillsyn/agents/go/planning-agent.md`               | 6953  | PASS     | PASS     | PASS     | PASS           | `opus`                 | body directive only | n/a                   | ADAPTED-FROM-SOURCE   | PASS    |
| D2  | `.tillsyn/agents/go/builder-agent.md`                | 5823  | PASS     | PASS     | PASS     | PASS           | `sonnet`               | body directive only | n/a                   | ADAPTED-FROM-SOURCE   | PASS    |
| D3  | `.tillsyn/agents/go/plan-qa-proof-agent.md`          | 6241  | PASS     | PASS     | PASS     | PASS           | `opus`                 | body directive only | distinct from D5      | ADAPTED-FROM-SOURCE   | PASS    |
| D4  | `.tillsyn/agents/go/plan-qa-falsification-agent.md`  | 6677  | PASS     | PASS     | PASS     | PASS           | `opus`                 | body directive only | distinct from D6      | ADAPTED-FROM-SOURCE   | PASS    |
| D5  | `.tillsyn/agents/go/build-qa-proof-agent.md`         | 6115  | PASS     | PASS     | PASS     | PASS           | `opus`                 | body directive only | distinct from D3      | ADAPTED-FROM-SOURCE   | PASS    |
| D6  | `.tillsyn/agents/go/build-qa-falsification-agent.md` | 6782  | PASS     | PASS     | PASS     | PASS           | `opus`                 | body directive only | distinct from D4      | ADAPTED-FROM-SOURCE   | PASS    |
| D7  | `.tillsyn/agents/go/research-agent.md`               | 5226  | PASS     | PASS     | PASS     | PASS           | `opus`                 | body directive only | n/a                   | ADAPTED-FROM-SOURCE   | PASS    |
| D8  | `.tillsyn/agents/go/closeout-agent.md`               | 4565  | PASS     | PASS     | PASS     | PASS           | `orchestrator-managed` | body directive only | n/a                   | FROM-SCRATCH          | PASS    |
| D9  | `.tillsyn/agents/go/commit-message-agent.md`         | 4013  | PASS     | PASS     | PASS     | PASS           | `haiku`                | n/a (no Section 0)  | n/a                   | FROM-SCRATCH          | PASS    |
| D10 | `.tillsyn/agents/go/orchestrator-managed.md`         | 5695  | PASS     | PASS     | PASS     | PASS           | `orchestrator-managed` | n/a (no Section 0)  | n/a                   | FROM-SCRATCH          | PASS    |
| D11 | `.tillsyn/agents/fe/planning-agent.md`               | 5782  | PASS     | PASS     | PASS     | PASS           | `opus`                 | body directive only | n/a                   | ADAPTED-FROM-SOURCE   | PASS    |
| D12 | `.tillsyn/agents/fe/builder-agent.md`                | 5422  | PASS     | PASS     | PASS     | PASS           | `sonnet`               | body directive only | n/a                   | ADAPTED-FROM-SOURCE   | PASS    |
| D13 | `.tillsyn/agents/fe/plan-qa-proof-agent.md`          | 5622  | PASS     | PASS     | PASS     | PASS           | `opus`                 | body directive only | distinct from D15     | ADAPTED-FROM-SOURCE   | PASS    |
| D14 | `.tillsyn/agents/fe/plan-qa-falsification-agent.md`  | 6042  | PASS     | PASS     | PASS     | PASS           | `opus`                 | body directive only | distinct from D16     | ADAPTED-FROM-SOURCE   | PASS    |
| D15 | `.tillsyn/agents/fe/build-qa-proof-agent.md`         | 5731  | PASS     | PASS     | PASS     | PASS           | `opus`                 | body directive only | distinct from D13     | ADAPTED-FROM-SOURCE   | PASS    |
| D16 | `.tillsyn/agents/fe/build-qa-falsification-agent.md` | 6251  | PASS     | PASS     | PASS     | PASS           | `opus`                 | body directive only | distinct from D14     | ADAPTED-FROM-SOURCE   | PASS    |
| D17 | `.tillsyn/agents/fe/research-agent.md`               | 5499  | PASS     | PASS     | PASS     | PASS           | `opus`                 | body directive only | n/a                   | ADAPTED-FROM-SOURCE   | PASS    |
| D18 | `.tillsyn/agents/fe/closeout-agent.md`               | 4982  | PASS     | PASS     | PASS     | PASS           | `orchestrator-managed` | n/a (no Section 0)  | n/a                   | FROM-SCRATCH          | PASS    |
| D19 | `.tillsyn/agents/fe/commit-message-agent.md`         | 4350  | PASS     | PASS     | PASS     | PASS           | `haiku`                | n/a (no Section 0)  | n/a                   | FROM-SCRATCH          | PASS    |
| D20 | `.tillsyn/agents/fe/orchestrator-managed.md`         | 5801  | PASS     | PASS     | PASS     | PASS           | `orchestrator-managed` | n/a (no Section 0)  | n/a                   | FROM-SCRATCH          | PASS    |

## Differentiation evidence (D3/D5, D4/D6, D13/D15, D14/D16)

- **D3 (plan-qa-proof) vs D5 (build-qa-proof) Go:**
  - D3 Evidence Sources: `PLAN.md`, `REVISION_BRIEF.md`, `SKETCH.md`, `_BLOCKERS.toml`; explicitly excludes Go source / test output / `git diff`.
  - D5 Evidence Sources: Go source declared in `paths`, `git diff`, `mage test-pkg` output, PLAN.md droplet section, Hylla; explicitly excludes REVISION_BRIEF / SKETCH as primary.
  - D3 What To Check covers 7 axes (atomic decomposition, parallelization graph, specify-block well-formedness, structural type consistency, multi-level decomposition discipline, paths/packages declared, scope alignment).
  - D5 What To Check covers 9 axes (impl matches AcceptanceCriteria, test coverage, scope compliance, no TODO/FIXME stubs, `mage ci` evidence, error handling, KindPayload vs diff, ContextBlocks invariants, no `mage install`).
- **D4 (plan-qa-fals) vs D6 (build-qa-fals) Go:**
  - D4 Attack Vectors target plan-graph integrity: missing `blocked_by`, blocker cycles, `_BLOCKERS.toml` drift, structural type violations, untestable AcceptanceCriteria, decomposition over/under-sizing, scope creep beyond REVISION_BRIEF, multi-level decomposition violation, `mage install` in ValidationPlan, over-blocked_by serialization.
  - D6 Attack Vectors target code/runtime: concurrency/race safety, interface misuse, error swallowing, false-positive tests, test residue, `mage install` invocation, raw `go` bypass, scope leakage, KindPayload vs diff drift, init() side effects, leaked goroutines, YAGNI pressure (12 vectors total).
- **D13 (plan-qa-proof) vs D15 (build-qa-proof) FE:**
  - D13 Evidence Sources: PLAN.md, REVISION_BRIEF.md, SKETCH.md, `_BLOCKERS.toml`; not FE source or screenshots.
  - D15 Evidence Sources: FE `.tsx`/`.astro`/`.css`/`.ts` source, `git diff`, Playwright MCP output (`browser_snapshot`, `browser_take_screenshot`), Vitest results, PLAN.md acceptance criteria.
  - D13 What To Check (9 axes): component boundary isolation, a11y coverage in plan, responsive coverage in plan, parallelization graph, specify-block well-formedness, Wails IPC dependency, stil tokens path, migration marker coverage, scope alignment.
  - D15 What To Check (9 axes): Playwright pass rates, a11y no new violations, TypeScript strict, ESLint clean, scope compliance, stil tokens usage, migration markers present, zero-JS discipline, build gates evidence.
- **D14 (plan-qa-fals) vs D16 (build-qa-fals) FE:**
  - D14 Attack Vectors (11) target FE plan integrity: missing a11y plan coverage, missing responsive coverage, missing `blocked_by` between FE siblings sharing TS modules/CSS, hidden Wails IPC dependency, stil `dist/` path reference, island justification gap, missing migration marker requirement, untestable AcceptanceCriteria, vim engine changes without `wails-keys.ts` filter awareness, `_BLOCKERS.toml` drift, scope creep beyond REVISION_BRIEF §2.15.
  - D16 Attack Vectors (12) target FE code/runtime: visual regression not caught by text assertions, a11y violation in `browser_snapshot`, Wails IPC error path not tested, intermediate-viewport break, stil token drift hardcoded, missing migration marker on new component, false-positive visual test, unjustified `client:load` island, scope leakage, TypeScript `any` without docs, plain JS file introduced, `dist/tokens.css` reference.

All four pairs are substantively differentiated, not near-identical copies.

## Source provenance spot-check (D1-D7 + D11-D17)

- **D1 (Go planning)** references `~/.claude/agents/go-planning-agent.md`-style content (Hylla evidence order, mage-first build gates, plan-down/build-up methodology, atomic-droplet sizing) plus Tillsyn-specific additions (12-kind enum, `paths`/`packages` mandate, PLAN.md droplet shape).
- **D2 (Go builder)** matches the global `go-builder-agent.md` shape (mage-first, TDD-red-green-refactor, `%w` wrapping, `charmbracelet/log`, table-driven tests, `-race`) plus Tillsyn additions (CONSUMER-TIE contract, atomic-droplet sizing awareness, `mage install` blocker, builder-only-edits-Go discipline).
- **D7 (Go research)** matches global research-agent semantics (read-only, Hylla evidence order, findings-to-orchestrator) plus Tillsyn additions (no `hylla_ingest`, no mage execution beyond `mage -l`).
- **D11/D12 (FE planning + builder)** match global `fe-planning-agent.md` / `fe-builder-agent.md` shape (CSS-first, zero-JS, a11y, Playwright/Vitest, responsive 3-viewport) plus Tillsyn additions (Wails v2 + Astro + SolidJS specifically, `src/styles/tokens.css` NOT `dist/`, vim engine at `fe/frontend/src/lib/vim/`, `wails-keys.ts` filter awareness, migration markers).
- **D13-D17 (FE plan-QA + build-QA + research)** mirror the global FE-QA agent attack vectors and adapt for Tillsyn FE specifics (Wails IPC dependencies, stil token path, intermediate viewport breaks).

## FROM-SCRATCH discipline (D8/D9/D10 + D18/D19/D20)

- **D8 (Go closeout)** cites WORKFLOW.md §"Phase 7 — Closeout", CLAUDE.md §"Cascade Ledger + Hylla Feedback", `hylla_ingest` invariants, STEWARD boundary. No claim of `~/.claude/agents/closeout-agent.md` source.
- **D9 (Go commit-message)** cites CLAUDE.md §"Git Commit Format", project commit-style memory. References Tillsyn commit history examples. No false source claim.
- **D10 (Go orchestrator-managed)** cites CLAUDE.md §"Orchestrator-as-Hub Architecture", 12-kind enum semantics for `closeout`/`refinement`/`discussion`/`human-verify`, MD-doc ownership split between drop-orch and STEWARD. Notes ORCH-MANAGED-R1 deferral to Drop 4c.8.
- **D18 (FE closeout)** explicitly states "Relationship to go/closeout-agent.md: FE closeout follows the same structure with FE-specific aggregation" — correct structural-template framing. Cites WORKFLOW.md Phase 7 + FE extensions. No false source claim.
- **D19 (FE commit-message)** mirrors D9 structure with FE scope tokens (`fe`, `fe/vim`, `fe/components`, `fe/styles`, `fe/wails`, `fe/tests`). FE-specific examples included.
- **D20 (FE orchestrator-managed)** mirrors D10 with FE-specific notes (Playwright coverage summary, visual regression audit, stil consistency, intermediate-viewport gaps). Explicit "Relationship to go/orchestrator-managed.md" framing.

## NITs

NIT-1 [severity: low] — `Hylla` listed in `tools:` of all Go and FE QA / planning / research agents, but in 8 of 10 FE files the body explicitly states "Hylla indexes Go only today" and the agent should never call `hylla_*`. The declared `tools:` capability is therefore vestigial for the FE group — included for cascade-model-policy alignment per round-2 Proof FF1.2 absorption (PLAN.md line 15). This is consistent with the plan's directive that the tools list governs authored prompts for future dogfood when Hylla is operational, but a reader could find it confusing. Recommendation: leave as-is per the round-2 decision; no fix required for W8.

NIT-2 [severity: low] — D8 references `$HOME/.tillsyn/till` as the `mage install` target in its body (`"never call hylla_ingest"` section), and D10 doesn't reference the target path at all. CLAUDE.md §"Build Verification" describes `mage install` as promoting "a binary to $HOME/.local/bin/till" (specifically `~/.local/bin/till` not `~/.tillsyn/till`). D1 and D2 also reference `$HOME/.tillsyn/till`. The Tillsyn-project-local prompts use `.tillsyn/till` which is a plausible Tillsyn-aware home but doesn't match CLAUDE.md verbatim. Recommendation: defer to a follow-up; the rule (NEVER `mage install`) is preserved, only the example path differs. No build / QA gate impact.

NIT-3 [severity: low] — D17 (FE research) `description:` line claims `Read/Grep/Glob + Context7/MDN/CanIUse + Playwright MCP inspection` but the `tools:` line declares `Read, Grep, Glob, Hylla` only — Context7, MDN/CanIUse, Playwright MCP, WebFetch are described in the body but not in the formal frontmatter `tools:` declaration. This is consistent with the spec (Common Frontmatter table fixes `tools:` per role) and the actual call to external tools happens via the subagent's frontmatter at spawn time, not the prompt body. Recommendation: leave as-is; the body documents which tools the spawned subagent will need, the frontmatter `tools:` field declares what THIS prompt's tier-1 override should pass to the resolver. No fix required.

NIT-4 [severity: low] — D9 / D19 (commit-message-agent) body sections "What You Do NOT Do" include "Do NOT run `mage install` or any mage build target" (D9) and "Do NOT run any npm commands, build tools, or Playwright MCP" (D19) — these are correct given the haiku model + `Read`-only tools list, but D9 / D19 don't include a Section 0 directive in the body. Per spec line 76 ("Prompts INSTRUCT subagents to render Section 0 in THEIR responses"), the requirement is conditional on the prompt being for a "substantive response" agent. Commit-message authoring at haiku is mechanical / non-substantive per the trivial-answer carve-out in CLAUDE.md §"Semi-Formal Reasoning — Section 0 Response Shape". The omission is consistent. Marking as PASS but flagging in NITs in case future review intends to require Section 0 universally.

NIT-5 [severity: low] — D10 (Go orchestrator-managed) and D20 (FE orchestrator-managed) similarly omit Section 0 directive. Same reasoning as NIT-4 — the orchestrator's procedural action items for `closeout` / `refinement` / `discussion` / `human-verify` are coordination-level work where the orchestrator IS the agent. The Section 0 discipline is documented in CLAUDE.md and applies to the orchestrator's substantive responses; embedding a directive inside the orchestrator-managed prompt is reasonable to omit. Marking as PASS, flagged as NIT.

NIT-6 [severity: low] — Each ADAPTED-FROM-SOURCE file (D1-D7, D11-D17) keeps the closing `## Hylla Feedback` section requirement, including FE files (D11-D17) where the body explicitly states Hylla doesn't index FE code today. This produces `N/A — ...` answers, but the contract holds — drop-orch closeout aggregation works regardless. Consistent with the round-2 Proof FF1.2 absorption. No fix required.

## Verdict rationale

All 20 prompt files pass the Common Validator Requirements (Signal A + B + C), the per-droplet AcceptanceCriteria for frontmatter shape, the verbatim migration-marker check with ASCII apostrophe, and the per-droplet differentiation requirements (D3/D5, D4/D6, D13/D15, D14/D16). FROM-SCRATCH discipline is intact for D8/D9/D10/D18/D19/D20 — these cite CLAUDE.md / WORKFLOW.md / WIKI.md rather than fabricating a non-existent source. ADAPTED-FROM-SOURCE prompts (D1-D7, D11-D17) preserve global-agent semantics while layering Tillsyn-specific overrides (mage-first, atomic-droplet sizing, MD-only workflow mode, CONSUMER-TIE, stil-tokens path, Wails IPC, vim engine awareness, migration markers).

Section 0 placement is correct in all 20 files — every file's first line is the frontmatter delimiter `---`, with Section 0 appearing as a body directive instructing the consuming subagent to render Section 0 in their responses (per spec line 76). The 6 files that legitimately omit a Section 0 directive (D9, D10, D18, D19, D20) are haiku / orchestrator-managed roles where the trivial-answer or coordination-work carve-out applies.

`mage test-pkg ./internal/app/dispatcher/cli_claude/render` is reported green by builder (83/83 PASS). `mage ci` is reported GREEN. D21 (Wave C smoke test) is not in scope for this QA pass — it's blocked by W1 completion and will be exercised separately.

Recommendation: PASS. NITs are documented for future-drop awareness; none block W8 closeout.
