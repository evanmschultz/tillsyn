# Cascade Methodology

This is the canonical methodology document for the **Cascade SDD** approach Tillsyn implements and dogfoods. It describes how work decomposes top-down through a recursive plan/build/QA tree, how each node is classified along three orthogonal axes, how agents reason and converge, and what gates keep the cascade honest.

This document is a **skeleton** today. Each section captures the methodology's *shape* in 1-3 paragraphs, with placeholder markers indicating where post-dogfood measurement and benchmark data will be filled in. Adopters can read this end-to-end to understand the methodology; depth and worked benchmarks land after the first dogfood cycles produce real numbers.

For the canonical vocabulary used throughout (drop / segment / confluence / droplet, the closed `kind` enum, the `role` enum), see `WIKI.md` § "Cascade Vocabulary" — that section is the single source of truth and this document cross-references rather than redefines it. Companion docs: `AGENTS_CONFIG.md` (per-machine `agents.toml` configuration reference) and `GDD_METHODOLOGY.md` (Graph-Driven Development methodology, which composes with this one post-Hylla-rev).

<!-- TODO populate post-dogfood with measured benchmarks -->

## Plan Down, Build Up

The methodology's spine is a single rule: **plan top-down, build bottom-up.** Planning starts at the highest level of the work — a drop, a feature, a release — and decomposes recursively into smaller plans, sub-plans, and finally atomic build droplets. Building inverts this: the smallest droplets land first, integration nodes follow once their inputs are green, and higher-level deliverables emerge from the bottom up. This is not a waterfall — every level of decomposition has its own QA pair, every level can fail and trigger a wipe-and-replan, and the recursion depth is bounded only by atomic-droplet sizing rules set per-template.

There is **no cap on the number of children at any planning level.** A planner is free to emit two children or twenty, depending on what the work needs. The only hard constraint is *atomic-droplet sizing* — each leaf `build` droplet must be small enough that one builder agent can finish it cleanly in one shot (the till-go template defaults to 1-4 code blocks / 80-120 LOC + tests, but those numbers are template-defined, not methodology-hardcoded; adopters running other templates may differ). When work exceeds the atomic budget, planners emit a sub-plan child instead of a build child, and the sub-plan recurses with its own planner agent. Multi-level decomposition is the norm, not the exception.

The build phase reverses the flow. Atomic droplets at the deepest level run first, in parallel where their `blocked_by` graph allows. Their outputs feed integration / confluence nodes that merge sibling streams. Each level's QA pair gates the next: a level-2 plan cannot be marked `complete` until every level-3 droplet under it is `complete`, and the level-2's own plan-QA-proof + plan-QA-falsification both PASS. This bottom-up assembly is what produces "atoms first, integration next" — the methodology's hedge against integration risk, because every atom is independently verified before it joins anything larger.

<!-- TODO populate post-dogfood with measured benchmarks -->

## Three Orthogonal Axes — `kind` × `metadata.role` × `metadata.structural_type`

Every non-project node in the cascade is classified along three independent axes, set explicitly at create time. None of them are inferred from the others. Templates' `child_rules`, gate rules, and agent bindings dispatch on combinations of all three. The orthogonality matters: collapsing any two axes into one produces ambiguity at the dispatch layer and breaks plan-QA's ability to attack misclassification.

The three axes are: **`kind` (what work)** — the closed 12-value enum (`plan`, `build`, `research`, `plan-qa-proof`, `plan-qa-falsification`, `build-qa-proof`, `build-qa-falsification`, `closeout`, `commit`, `refinement`, `discussion`, `human-verify`); **`metadata.role` (who does it)** — the closed role enum (`builder`, `qa-proof`, `qa-falsification`, `qa-a11y`, `qa-visual`, `design`, `commit`, `planner`, `research`); and **`metadata.structural_type` (where it sits)** — the closed 4-value cascade-shape enum (`drop`, `segment`, `confluence`, `droplet`). The dual `kind` + `role` axes earn their keep on QA kinds where parent context disambiguates: `build-qa-proof` and `plan-qa-proof` both carry `role=qa-proof`, but the QA agent's verification axis differs based on parent kind.

For the canonical definitions of each enum value, the worked-combinations table, and atomicity rules (e.g. "`droplet` MUST have zero children" / "`confluence` MUST have non-empty `blocked_by`"), see `WIKI.md` § "Cascade Vocabulary." This methodology doc cross-references rather than duplicates that vocabulary, per the single-canonical-source rule in the wiki.

<!-- TODO populate post-dogfood with measured benchmarks -->

## Closed 12-Value `kind` Enum

`action_items.kind` is a closed 12-value enum, chosen by the creator at create time. There is no inferred default and no fallback kind. The enum partitions cascade work into named work-types: planning-dominant decomposition (`plan`); read-only investigation (`research`); code-changing leaf work (`build`); QA passes attached to both planning and build parents (`plan-qa-proof`, `plan-qa-falsification`, `build-qa-proof`, `build-qa-falsification`); drop-end coordination (`closeout`, `commit`); long-lived umbrellas and decision parks (`refinement`, `discussion`); and dev sign-off hold points (`human-verify`).

Each kind has specific structural rules. `plan` and `build` auto-create QA-twin children via template `[[child_rules]]` — every `plan` gets `plan-qa-proof` + `plan-qa-falsification`, every `build` gets `build-qa-proof` + `build-qa-falsification`, both pairs `blocked_by` their parent. `research` does NOT auto-create QA twins — research outputs are findings, not implementation claims, so the proof/falsification asymmetry doesn't apply; the orchestrator reviews findings via comment thread. `closeout` / `refinement` / `discussion` / `human-verify` are standalone — they don't auto-create QA twins either; they have their own bespoke gates.

The 12-value enum is closed: extension happens via templates that register custom kinds attaching as sub-action-items of specific generics (e.g. a custom `ledger-update` under `closeout`). Adopters never modify the core 12-value Go enum; they extend via template customization. This is the canonical extension surface and is documented in the templates layer (see `WIKI.md` § "Closed 12-Value `kind` Enum" for the full reference).

<!-- TODO populate post-dogfood with measured benchmarks -->

## `metadata.role` Enum

The `role` axis names *who does the work* — a closed enum: `builder`, `qa-proof`, `qa-falsification`, `qa-a11y`, `qa-visual`, `design`, `commit`, `planner`, `research`. Roles bind to agent definitions via the `[agent_bindings]` section of a template's TOML — the dispatcher reads `(kind, role)` and looks up the corresponding agent file (e.g. `(build, builder) → till-go/builder-agent.md`). The same role can apply to multiple kinds: `qa-proof` applies to both `plan-qa-proof` and `build-qa-proof`, and the QA agent branches on `parent.kind` at runtime to pick the correct verification axis.

The dual-axis `(kind, role)` design is what enables a single agent file to serve two kinds. The `qa-proof-agent.md` and `qa-falsification-agent.md` files in the till-go group are each authored once, and the agent reads `parent.kind` in its system prompt to determine whether to apply the plan-QA verification axis (atomic decomposition + parallelization graph + Specify-block well-formedness) or the build-QA axis (acceptance-criteria conformance + KindPayload-vs-diff drift + adversarial DecisionLog review).

Pre-Drop-2 the role lives in description prose (`Role: builder`, `Role: qa-proof`); post-Drop-2 it lands on `metadata.role` as a first-class field. Either way, the role is set at create time and is mandatory — no inferred defaults, just like `kind` and `structural_type`.

<!-- TODO populate post-dogfood with measured benchmarks -->

## `metadata.structural_type` Enum (drop / segment / confluence / droplet)

`structural_type` names *where a node sits in the cascade flow's structure*, independent of what kind of work it is or who does it. Picture water flowing down a series of waterfalls: a **drop** is one vertical step that may decompose into more steps; **segments** are parallel streams within a drop; **confluences** are merge points where streams rejoin; **droplets** are atomic, indivisible units that finish in one shot. The metaphor orients the vocabulary; enforcement happens at the create/update boundary.

The 4-value enum is mandatory on every non-project node, validated at the create/update boundary post-Drop-3. Atomicity rules: `droplet` MUST have zero children (any child indicates misclassification — the parent should be `segment` or `drop`); `confluence` MUST have non-empty `blocked_by` (empty is a definitional contradiction since a confluence merges upstream streams); `segment` may recurse and contain droplets, sub-segments, or confluences; `drop` is the level-1 cascade step under the project root.

For the orthogonality table showing how `structural_type` composes with `metadata.role` (canonical combinations like `(droplet, builder)` for build leaves and `(confluence, orchestrator)` for integration points), see `WIKI.md` § "Cascade Vocabulary" — that section is the single canonical source. This methodology doc holds the methodology-level explanation of *why* the axis exists separately from `kind` and `role`; the wiki holds the enforced vocabulary.

<!-- TODO populate post-dogfood with measured benchmarks -->

## Agent Shape

Each cascade kind binds to a specific agent at dispatch time. Agents are defined in template-shipped Markdown files under `internal/templates/builtin/agents/<group>/<name>.md` (where `<group>` is `till-gen`, `till-go`, or `till-gdd` and `<name>` is one of the 7 standard agent names: `planning`, `builder`, `qa-proof`, `qa-falsification`, `research`, `closeout`, `commit-message`). Adopters can override per-project at `<project>/.tillsyn/agents/<name>.md` or per-user at `~/.tillsyn/agents/<group>/<name>.md`; the resolver checks project → user → embedded in priority order.

Agent files carry YAML frontmatter (`name`, `description`) and a substantive body. Runtime configuration — model choice, tool allowlists, environment variables, MCP config — lives in `agents.toml` (per-project) or `agents.local.toml` (per-machine, `.gitignore`d), NOT in agent-file frontmatter. The `tools_allow` list is overridable per-machine; `tools_deny` is a safety floor and rejects user override at startup. Frontmatter `model:` and `tools:` keys, if present in agent files, are stripped at render time when `agents.toml` has the corresponding key set — see `AGENTS_CONFIG.md` for the full configuration reference.

Each agent has a tightly scoped role — planners decompose and never edit code; builders implement leaf work and never spawn other agents; QA agents read and verify but never edit. This separation-of-concerns is hardcoded structural invariant, not template-customizable. Cross-role boundary violations are rejected at the dispatch / MCP layer with structured errors citing the violated invariant.

<!-- TODO populate post-dogfood with measured benchmarks -->

## Section 0 — Semi-Formal Reasoning Certificate

Every substantive agent response begins with a `# Section 0 — SEMI-FORMAL REASONING` block before the response body. The block contains either 5 named passes (orchestrator-facing: `Planner` / `Builder` / `QA Proof` / `QA Falsification` / `Convergence`) or 4 named passes (subagent-facing: `Proposal` / `QA Proof` / `QA Falsification` / `Convergence`). Each pass uses a 5-field certificate: **Premises** (what must hold), **Evidence** (grounded sources, not implicit background), **Trace or cases** (concrete paths through the reasoning), **Conclusion** (the claim), and **Unknowns** (what's still uncertain, routed via comment / handoff / attention item or explicitly accepted).

The shape is adapted from Ugare & Chandra's *Agentic Code Reasoning* (arxiv 2603.01896, Meta, 4 Mar 2026) — the paper shows structured certificates reduce patch-equivalence errors substantially — with two methodology-level extensions: **Evidence** and **Unknowns** as first-class fields, and an explicit **adversarial QA Falsification pass** the paper does not include. The falsification extension targets the paper's §4.3 residual failure mode where elaborate but incomplete reasoning chains produce confident but wrong answers; a dedicated adversarial pass is the methodology's hedge against that mode.

Convergence is the gate: an agent declares Convergence only when (a) QA Falsification produced no unmitigated counterexample, (b) QA Proof confirmed evidence completeness across every claim, and (c) remaining Unknowns are explicit and routed. If any of (a)/(b)/(c) fail, the agent loops back to the earliest pass needing rework before declaring Convergence. The reasoning lives in the orchestrator-facing response only — it never gets written into Tillsyn `description`, `metadata.*`, `completion_contract.completion_notes`, comments, or any other durable artifact. Tillsyn stores finalized artifacts, not process.

<!-- TODO populate post-dogfood with measured benchmarks -->

## Tillsyn-Flavored Specify Pass

Each plan and build node carries a structured **Specify** block in its description (or post-Drop-2, in `metadata.specify`). The block has five fields: **Objective** (what this node accomplishes, in one sentence); **AcceptanceCriteria** (a bulleted, testable list of what "done" looks like); **ValidationPlan** (concrete commands or steps to verify acceptance); **RiskNotes** (known hazards and mitigations); and **ContextBlocks** (typed references to upstream decisions / constraints / warnings / reference docs that bound the work).

The shape is inspired by spec-driven development frameworks (Specify, GitHub Spec Kit) but Tillsyn-flavored: the block sits inside the cascade tree as first-class metadata, not in a separate spec file. Plan-QA-proof verifies AcceptanceCriteria support the Objective and that the ValidationPlan exercises every criterion. Plan-QA-falsification attacks the Specify for under-constraining Objectives, over-constraining AcceptanceCriteria (untestable bullets), missing RiskNotes, and ContextBlocks that don't bound the cited risks.

Specify blocks compose with the cascade's recursive structure: a level-2 plan's Specify constrains its level-3 children's Specifies, and child Specifies inherit ContextBlocks from their parents (the dispatcher's context aggregator merges them at spawn time). This makes the Specify pass scale with decomposition depth — high-level intent flows down without being repeated, and low-level details bubble up through QA back to the parent level.

<!-- TODO populate post-dogfood with measured benchmarks -->

## TN-Per-Section Response Style

Substantive agent responses follow a stable numbered-Markdown shape: top-level sections are `## 1. <Title>` / `## 2. <Title>` / etc., with sub-bullets `- 1.1 <text>` / `- 1.2 <text>` / etc. The response closes with a `## TL;DR` containing **one `TN` item per top-level section** (`T1` summarizing section `1`, `T2` summarizing section `2`, etc.) — no extras, no gaps. This pairs with Section 0 to make responses both auditable (every claim has evidence) and addressable (every section has a stable reference like "T2.1 in the plan").

The TN-per-section invariant is enforced at the orchestrator-facing layer; subagents inherit the convention via spawn-prompt directives. Trivial responses (one-line factual lookups, terse confirmations, simple yes/no answers) skip both Section 0 and the numbered body — the rule prevents premature judgment on substantive work, not ceremony for small answers.

The shape is what makes long agent threads navigable. Devs reviewing a plan-QA-proof verdict can address `T3 in the QA verdict` and the agent or orchestrator knows exactly what's being cited. Without stable numbering, address-by-quote degrades into address-by-paraphrase, and audit trails get lossy.

<!-- TODO populate post-dogfood with measured benchmarks -->

## Hylla-First Evidence Ordering

Agents working on Tillsyn use **Hylla** (the project's graph-of-symbols indexer) as the primary source for committed-code understanding. The ordering is: (1) Hylla for committed Go code; (2) `git diff` for files changed since the last Hylla ingest; (3) Context7 + `go doc` + LSP for external library / language / tooling semantics the repo can't answer itself. Non-Go files (markdown, TOML, YAML, magefile, SQL) fall through directly to `Read` / `Grep` / `Glob` since Hylla today indexes Go only.

Hylla-first matters because the indexer's graph traversal (`hylla_graph_nav`, `hylla_refs_find`) surfaces relationships LSP and grep miss — call-graph queries, symbol summaries, semantic search across the committed graph. Agents that skip Hylla and reach for grep tend to find string matches but miss the actual call-graph dependency. When a Hylla query misses (the indexer doesn't return what was needed), agents are required to record the miss in their closing comment under a `## Hylla Feedback` heading; the orchestrator aggregates these at drop end to drive Hylla improvements.

For projects that aren't Tillsyn (Hylla is Tillsyn-internal today), the equivalent rule is: use the project's primary semantic-graph indexer first, fall back to LSP and grep for misses, and record misses for tooling improvement. The principle is: prefer graph queries to string matches when a graph index is available.

<!-- TODO populate post-dogfood with measured benchmarks -->

## TDD Requirement

Build droplets follow test-driven development. Tests are authored alongside or before production code, not after. Coverage gates are enforced at the canonical `mage ci` path — the till-go template defaults to ≥70% line coverage on touched packages; below threshold is a hard failure. Coverage thresholds are template-defined (not methodology-hardcoded), so adopters running other templates may differ; the methodology-level rule is *coverage gates exist and are enforced at CI*.

Tests are table-driven where the work admits it, behavior-oriented (not implementation-coupled), and run with `-race` via mage targets. Builders never invoke raw `go test` / `go build` / `go vet` — every build / test / lint goes through `mage <target>` so the canonical CI path is the canonical local verification path. If a mage target is broken, builders fix the target rather than bypass it. Cold-cache equivalence between local `mage ci` and CI `mage ci` is the standard for "ready to push."

The TDD rule composes with the cascade: a `build` droplet's `build-qa-proof` child verifies that every AcceptanceCriteria bullet has a corresponding test, and `build-qa-falsification` attacks the test suite for missing edge cases, untestable assertions, and silent skips. TDD without QA enforcement degrades into "I wrote some tests"; the QA twin makes the rule load-bearing.

<!-- TODO populate post-dogfood with measured benchmarks -->

## QA Proof vs Falsification — Asymmetric Verification

QA in the cascade is **two distinct passes, not duplicate reviewers.** Proof and falsification are asymmetric by design and run in parallel as separate agent contexts so each gets a fresh window without parent-hindsight bias.

**QA Proof** verifies evidence completeness, reasoning coherence, trace coverage, and that the parent's claim is actually supported by the current code and evidence. Plan-QA-proof checks atomic decomposition + parallelization graph + Specify-block well-formedness + multi-level decomposition discipline. Build-QA-proof checks AcceptanceCriteria conformance + KindPayload-vs-diff alignment + CompletionContract checklist + DecisionLog evidence chains. The proof axis is: *"is the claim supported?"*

**QA Falsification** actively tries to break the parent's conclusion via counterexamples, alternate traces, hidden dependencies, contract mismatches, and YAGNI pressure. Plan-QA-falsification attacks over-decomposition, under-decomposition, missing `blocked_by`, over-`blocked_by`, untestable Specify bullets, and cascade-tree misclassification. Build-QA-falsification attacks KindPayload-vs-final-code drift, silently dropped acceptance criteria, parent-plan contract mismatches, and adversarial DecisionLog review. The falsification axis is: *"can the claim be false?"*

Both passes must PASS for the parent to close. A failed proof OR a failed falsification re-routes the work — for `kind=plan` failures, the system wipes the children atomically and respawns the planner with synthesized failure context; for `kind=build` failures, the build re-spawns with QA findings injected into its system prompt. The asymmetric pair is what catches both insufficient evidence (proof's lane) and over-confident reasoning (falsification's lane).

<!-- TODO populate post-dogfood with measured benchmarks -->

## `blocked_by` Ordering Primitive

`blocked_by` is the **only sibling and cross-drop ordering primitive** in the cascade. Planners set `blocked_by` at creation time on every child that depends on a sibling's completion before its own work can start. The dispatcher reads `blocked_by` to gate `in_progress` transitions: a child cannot move to `in_progress` while any node in its `blocked_by` list is incomplete or `failed`.

`blocked_by` operates at two levels: planner-set (static, declared at creation) and dispatcher-inserted (dynamic, runtime). The dispatcher's lock manager inserts runtime `blocked_by` on `in_progress` promotion when sibling locks conflict — for example, when two `build` droplets share a Go package and would race on the package's compile/test unit, the dispatcher inserts a `blocked_by` between them so they serialize. Planners are required to set static `blocked_by` whenever sibling droplets share a file (`paths` overlap) or a package (`packages` overlap); plan-QA-falsification attacks missing static `blocked_by` as a primary risk.

The `blocked_by` primitive is what makes parallel builds safe. Without it, two builders touching the same Go package would race and both fail their compile / test runs. With it, the cascade fans out work as wide as the lock graph allows and serializes only where necessary. This is the fan-out enabler — the methodology's response to "how do you parallelize without losing correctness."

<!-- TODO populate post-dogfood with measured benchmarks -->

## Parent-Children-Complete Invariant

A parent node cannot be marked `complete` while any child is incomplete, `failed`, or `blocked`. This is an **always-on invariant** — not a policy bit, not template-configurable, not relaxable. The rule applies recursively: a level-2 plan can't close until every level-3 droplet under it is `complete`; a level-1 drop can't close until every level-2 node is `complete`; and the entire cascade rolls up cleanly only when every leaf has finished and every parent's QA twins have passed.

The invariant is enforced at the domain layer in `internal/domain/action_item.go` and at the `till.action_item(operation=update)` MCP boundary. Attempts to mark a parent `complete` with incomplete children return a closed sentinel error citing the offending children. Pre-Drop-1 the rule was policy-toggled; post-Drop-1 it's hardcoded.

This invariant composes with the `failed` terminal state landed in Drop 1: when a `build` droplet's QA twin returns a falsification verdict, the build moves to `failed`, the parent plan can't close, and the wipe-and-replan flow fires. Without the parent-children-complete invariant, the cascade would silently close partial trees and lose audit trail.

<!-- TODO populate post-dogfood with measured benchmarks -->

## Isolation Enforcement

Spawned subagents run in **bundle-isolated contexts** — they never see the orchestrator's `~/.claude/CLAUDE.md`, the project's `.claude/CLAUDE.md`, system skills, project plugins, or hooks. The isolation is enforced at the spawn layer: Tillsyn assembles a per-spawn bundle directory containing only the files the agent's role needs (the agent's own `<bundle>/plugin/agents/<name>.md`, the rendered system prompt, MCP config, settings) and invokes `claude --bare --plugin-dir <bundle>/plugin --agent <name> --setting-sources "" --strict-mcp-config --settings ... --mcp-config ...`. The `--bare` flag tells Claude Code to skip every normal context source (Path B / system CLAUDE.md / skills / project CLAUDE.md / hooks / `~/.claude/settings.json` / system plugins).

Isolation matters because the orchestrator's CLAUDE.md and skills are tuned for orchestration — they're loaded with planning rules, dispatch rules, multi-agent coordination semantics. A spawned QA agent reading those would be confused into orchestrator-shaped reasoning. By stripping the entire normal context surface and hand-assembling exactly what the agent needs, Tillsyn guarantees each spawned agent sees only its role-tuned context — no leakage from the orchestrator's configuration.

Sentinel-injection integration tests verify isolation end-to-end: synthetic "BLEED_SENTINEL" strings injected into `~/.claude/CLAUDE.md`, system agents, project CLAUDE.md, and hooks are asserted absent in the spawned process's actual prompt. The tests fail loudly if any normal context source leaks through. The post-render bundle validator (Drop 4c.6 W3) checks the spawned bundle's agent body is non-empty and substantive (not the prior stub-shape) before the spawn proceeds — a defense-in-depth gate against silent isolation regressions.

<!-- TODO populate post-dogfood with measured benchmarks -->

## Cross-References

Companion methodology and configuration docs live alongside this one at the repo root:

- **`AGENTS_CONFIG.md`** — adopter-facing reference for `agents.toml` schema, override semantics, env_set vs env_from_shell, tools_allow vs tools_deny scope, frontmatter strip behavior, `claude_md_addons`, and worked Bedrock / Vertex / OpenRouter / Ollama Cloud examples.
- **`GDD_METHODOLOGY.md`** — Graph-Driven Development methodology (Hylla-flavored). Composes with Cascade Methodology: cascade describes *how work decomposes and verifies*; GDD describes *how knowledge is graph-indexed and traversed*. Substantive content lands post-Hylla-rev / post-dogfood per `project_methodology_docs_tracker.md`.
- **`SPAWN_PIPELINE.md`** — the per-spawn bundle assembly pipeline (env vars, settings, MCP config, agent body resolution). Cited from the Isolation Enforcement section above.
- **`CLI_ADAPTER_AUTHORING.md`** — guide for authoring new CLI adapters (today: Claude Code; future: Codex, others). Inherits the `--bare`-collapsed isolation framing.
- **`WIKI.md` § "Cascade Vocabulary"** — single canonical source for the `kind` enum, `role` enum, `structural_type` enum, and the orthogonality table. This methodology doc cross-references rather than duplicates that vocabulary.

<!-- TODO populate post-dogfood with measured benchmarks -->

## Comparison Surface

Cascade Methodology sits in the same problem space as Spec-Driven Development (Specify, GitHub Spec Kit), Plan-Decompose-Execute agent frameworks, and the broader Agentic Code Reasoning literature. The methodology's distinguishing commitments are: (1) **closed-12-kind classification** with no inferred defaults, forcing every node to be explicitly typed; (2) **three orthogonal axes** (`kind` / `role` / `structural_type`) that templates and dispatchers compose on independently; (3) **proof-and-falsification QA asymmetry** running in parallel as separate agent contexts; (4) **bottom-up build assembly** with atomic-droplet sizing as the only recursion-cap; (5) **bundle-isolated agent spawns** that strip every normal context source and hand-assemble role-tuned prompts.

These commitments shape the comparison axes. Against pure Spec-Driven approaches, Cascade Methodology adds the recursive cascade tree + dynamic `blocked_by` lock graph + system-managed wipe-and-replan on failure. Against pure agentic frameworks, it adds the structured-Specify pass + Section 0 5-pass certificate + the asymmetric QA pair. Against feature-flagged "AI coding assistants," it adds end-to-end cascade-driven dispatch (no orchestrator hand-work in the steady state) and isolation-enforcement at the spawn boundary.

Concrete benchmark axes — token cost per drop, end-to-end wall-clock per drop, error-rate after wipe-and-replan, integration-defect rate at confluences, escalation rate to human review — will be populated post-dogfood per `project_methodology_docs_tracker.md`'s benchmark plan. The methodology is benchmark-aimed; the skeleton ships before the numbers, and the numbers ship after the first dogfood cycles produce them.

<!-- TODO populate post-dogfood with measured benchmarks -->

## Provenance

This methodology is the synthesis of multiple threads:

- **Plan-Down-Build-Up spine** — derived from the rollout's `feedback_plan_down_build_up.md` memory entry; refined through Drop 4c.6 / 4c.7 / 4c.8 sketch-and-plan iteration.
- **Section 0 certificate shape** — Ugare & Chandra, *Agentic Code Reasoning* (arxiv 2603.01896, Meta, 4 Mar 2026), with Tillsyn's two extensions (Evidence + Unknowns as first-class fields, plus the adversarial QA Falsification pass).
- **Tillsyn-flavored Specify pass** — inspired by Spec-Driven Development frameworks (Specify, GitHub Spec Kit) and adapted to live in cascade-tree metadata rather than separate spec files.
- **Closed-12-kind enum + orthogonal-axes design** — Tillsyn-internal, landed in Drop 1.75 with the kind-collapse migration; documented canonically in `WIKI.md` § "Cascade Vocabulary."
- **`--bare`-collapsed isolation enforcement** — Anthropic's documented Claude Code `--bare` flag behavior; verified per `RESEARCH/ISOLATION_ENFORCEMENT_FIX.md` and shipped end-to-end in Drop 4c.6 W3.
- **Atomic-droplet sizing rules** — till-go template values (1-4 code blocks, 80-120 LOC + tests) tuned through Drop 4c iterations; adopters using other templates may differ. The methodology-level invariant is *atomic sizing exists*, not the specific till-go numbers.

The methodology is intentionally template-customizable at the semantic edges (sizing numbers, model assignments, tool allowlists) and hardcoded-structural at the invariant edges (closed enums, parent-children-complete, isolation enforcement, separation-of-concerns between roles). The split follows Tillsyn's "templates define semantic behavior; Tillsyn enforces structural invariants" rule (`feedback_tillsyn_enforces_templates.md`).

The skeleton in this document is intentionally evergreen at the methodology-shape level — the rules that change with measurement (sizing thresholds, escalation N-counts, token caps) are deferred to template config and post-dogfood numbers, while the rules that anchor the methodology (closed enums, asymmetric QA, plan-down-build-up, isolation-by-default) ship as load-bearing skeleton text. Future revisions will populate the post-dogfood benchmark sections and refine the template-defined edges in lockstep with measured outcomes.

<!-- TODO populate post-dogfood with measured benchmarks -->
