# Spec-Driven Development — Cascade Prompt Review

Research deliverable for Drop 4c.6 sketch §11 pending update #1. Read-only analysis. Three parts: (1) what spec-driven development is, (2) what Tillsyn's current cascade prompts emphasize, (3) where SDD principles map onto the cascade kinds. **No specific prompt edits proposed** — that translation is the orchestrator's job after dev discussion.

---

## Part 1 — Spec-Driven Development Overview

### Working definition

Spec-driven development (SDD) treats a written, human-authored specification as the **primary artifact** of software work, with code as a derived expression of the spec rather than the source of truth. In AI-assisted workflows the spec is the contract the model reads to know *what* and *how* before writing any code. SDD is currently most prominent as an AI-coding-agent practice; the term is recent (2025-2026 industry adoption wave) and has multiple competing-but-overlapping definitions across vendors. It is not (yet) a single canonical methodology with a standards body.

Sources: [GitHub Spec-Kit announcement, Sept 2025](https://github.blog/ai-and-ml/generative-ai/spec-driven-development-with-ai-get-started-with-a-new-open-source-toolkit/); [GitHub Spec-Kit repo + spec-driven.md](https://github.com/github/spec-kit); [Martin Fowler — Understanding Spec-Driven Development: Kiro, spec-kit, Tessl](https://martinfowler.com/articles/exploring-gen-ai/sdd-3-tools.html); [Augment Code — What Is Spec-Driven Development](https://www.augmentcode.com/guides/what-is-spec-driven-development); [arxiv 2602.00180 — Spec-Driven Development: From Code to Contract](https://arxiv.org/html/2602.00180v1).

### Core principles (synthesized across sources)

1. **Spec is the source of truth.** Code is the spec's expression in a particular language/framework; maintaining software means evolving the spec, not the code (Spec-Kit `spec-driven.md`; Tessl's "spec-as-source" extreme).
2. **Gated phases.** Spec-Kit canonicalizes four phases — `/specify` (goals + user journeys) → `/plan` (architecture, stack, constraints) → `/tasks` (small reviewable units) → `/implement`. Human review at each phase boundary before advancing.
3. **Three rigor levels** (Fowler): **spec-first** (write spec, generate code once, edit code from then on); **spec-anchored** (keep spec for evolution); **spec-as-source** (only the spec is human-edited; code regenerates).
4. **Specs are precise enough to test against.** Per arxiv 2602.00180 — specifications are machine-readable contracts; conformance is verifiable.
5. **Small, iterative specs beat big upfront PRDs.** ThoughtWorks Technology Radar v33 explicitly warns against "bias toward heavy up-front specification and big-bang releases" — that's the headline anti-pattern.
6. **Traceability is the real work** (Fowler). Tooling UX for review, refinement, and round-trip code↔spec sync is what separates an SDD practice from a slide deck.
7. **Spec lifetime matches code lifetime.** A spec written, used once, then thrown away is a glorified prompt; an SDD spec is maintained alongside the code it generates.
8. **Human-in-the-loop at phase gates.** The spec, plan, and tasks each get human review. Implementation is the only autonomous phase, and it is bounded by the prior three artifacts.

### Anti-patterns

- **Heavy upfront waterfall specs** (ThoughtWorks Radar v33).
- **Spec-and-pray:** writing a spec, never validating it against the generated code, treating divergence as code's problem.
- **Spec-as-comment:** prose-only specs with no testable acceptance criteria — degenerates to expensive documentation.
- **Treating SDD as a replacement for TDD/BDD** rather than an architectural layer above them.

### Differences from TDD / BDD / traditional spec writing

- **TDD = SDD at the unit level.** A failing test is a micro-spec for a single function. TDD tests are executable; they don't enumerate user journeys or architectural choices.
- **BDD = SDD's most direct ancestor.** Gherkin scenarios are executable specs at the behavior level (Kinde, testRigor). SDD generalizes BDD upward — beyond per-feature scenarios, into architecture / stack / project-level specs that AI agents read as one durable contract.
- **Traditional spec writing** (waterfall RUP / IEEE 830) produced static documents human engineers translated by hand. SDD specs are read by an AI implementer and intentionally drive the implementation; that round-trip changes what the spec must contain (sufficient detail for the agent to act, not just for the human to approve).
- SDD **complements** TDD/BDD; it does not replace them (Augment Code; Kinde). You can run TDD inside an SDD `/implement` phase.

### Roles in a coding-agent cascade most affected by SDD

- **Planner** — strongest fit. SDD's `/specify` + `/plan` + `/tasks` phases map onto cascade planning almost 1:1.
- **Builder** — affected indirectly. The spec that drove the plan also constrains the build's acceptance criteria.
- **QA (proof + falsification)** — affected when the spec carries verifiable acceptance criteria; QA can attack the spec itself, not just the implementation.
- **Research** — neutral. Research compiles findings; it does not produce a spec or code.
- **Commit** — neutral; commits are derivative of the build artifact.

---

## Part 2 — Current Agent Prompt Inventory

### How Tillsyn ships prompt content today

**Critical finding:** Tillsyn does NOT ship the substantive cascade-agent system prompts in-repo. The `[agent_bindings.<kind>]` blocks in `internal/templates/builtin/default-go.toml` carry **structural / runtime fields only** (`agent_name`, `model`, `effort`, `tools`, `max_tries`, `max_budget_usd`, `max_turns`, `auto_push`, `commit_agent`, `blocked_retries`, `blocked_retry_cooldown`) plus a `[agent_bindings.<kind>.context]` sub-block (parent / parent_git_diff / ancestors_by_kind / delivery / max_chars / max_rule_duration). No prompt-shaping fields, no spec template, no acceptance-criteria scaffold — see `default-go.toml:388-599`.

The substantive prompt content (role definition, evidence order, tool discipline, output format) lives at `~/.claude/agents/<name>.md` on the dev's machine and is loaded by the Claude CLI from the **system-installed plugin path** (Path B per `SPAWN_PIPELINE.md`'s "Two Plugin Paths" section). The per-spawn bundle Tillsyn writes contains only a **minimal stub** at `<bundle>/plugin/agents/<name>.md` whose body is a one-liner pointer (`render.go:assembleAgentFileBody`, ~ line 339). The cross-CLI `system-prompt.md` Tillsyn does generate carries action-item structural fields only — `task_id`, `project_id`, `project_dir`, `kind`, `title`, `paths`, `packages`, plus a generic move-state directive (`render.go:assembleSystemPromptBody`, ~ line 246). No role definition, no Section 0 scaffold, no spec scaffold.

This means **changing the cascade-agent role contract today is a `~/.claude/agents/*.md` edit**, not a Tillsyn-repo edit. `internal/app/dispatcher/binding_resolved.go` has a `SystemPromptTemplatePath` field per render.go:323 ("future evolution") but the F.7.3b stub does not yet read from it.

### Per-agent summaries (Go variants — Tillsyn is Go-only today)

The agent files at `~/.claude/agents/*.md` are not readable from this research session (sandbox scope is the project root). Summaries below are reconstructed from the orchestration-process documentation in `CLAUDE.md`, `SEMI-FORMAL-REASONING.md`, `SPAWN_PIPELINE.md`, `AGENTS.md`, `WIKI.md`, and (for `go-research-agent`) the agent's own definition loaded into this research session's system prompt at spawn time. Where a field is reconstructed-not-direct, it is flagged.

#### `go-builder-agent`

- **Evidence sources required**: Hylla (committed Go), `git diff` (uncommitted), Context7 + `go doc` + gopls/`LSP` (external/library), `Read`/`Grep`/`Glob` for non-Go files. Hylla-first per `CLAUDE.md` §"Code Understanding Rules". Cite Hylla misses in closing comment.
- **Output format**: Section 0 5-pass certificate (subagent variant — 4 passes: Proposal / QA Proof / QA Falsification / Convergence) before action; finalized closing certificate posts to Tillsyn closing comment.
- **Verification gates**: `mage ci` before any `build` action item moves to complete (`CLAUDE.md` §"Build Verification"). `mage` targets only — never raw `go build`/`go test`. `mage install` forbidden for any agent.
- **Spec/pre-build contract**: action-item `description` carries `paths`/`packages`/`acceptance criteria`/`mage targets`/`cross-references` per `CLAUDE.md` §"Agent State Management" and the WIKI's atomic-droplet rule. **Closest current analogue to a spec.** No template enforces shape; planner authoring is honor-system.
- **Domain**: Go-only.

#### `go-planning-agent`

- **Evidence sources required**: Hylla-first for committed-code grounding, `git diff` for staleness deltas, Context7 for external semantics. `/plan-from-hylla` slash command is the canonical entry.
- **Output format**: Section 0 5-pass; planner produces a decomposition into sub-`plan` / `build` action items with `paths`/`packages`/`blocked_by` wiring (`CLAUDE.md` §"Drop-Orch Spin-Up Checklist + Planner Contract" memory; `feedback_drop_orch_spinup_discipline`).
- **Verification gates**: pairing — every plan auto-creates `plan-qa-proof` + `plan-qa-falsification` children (`default-go.toml:[[child_rules]]`). Planner cannot "complete" without those passing.
- **Spec/pre-build contract**: planner OUTPUT is the closest thing the cascade has to an executable spec (action-item descriptions with paths/packages/criteria become the build-agent's contract). But planner INPUT today is ad-hoc orchestrator-supplied directive prose; no template-enforced spec scaffold.
- **Domain**: Go-only.

#### `go-qa-proof-agent` / `go-qa-falsification-agent`

- **Evidence sources required**: Hylla / git diff / `LSP` / Context7. QA bindings deliberately do NOT pre-stage `parent_git_diff` per `default-go.toml:535-541` ("REV-4: independent verification is load-bearing"). QA fetches its own evidence.
- **Output format**: Section 0 4-pass; verdict + findings posted to Tillsyn closing comment.
- **Verification gates**: QA itself is the gate; both QA passes must be `complete` for a `plan` or `build` to terminal-complete (`CLAUDE.md` §"Required Children", §"Action-Item Lifecycle").
- **Spec/pre-build contract**: QA agents check the **claim** in the parent's closing certificate against evidence. They do not check against an explicit upstream spec — they check whether the claim is supported. If a spec existed, QA could check both (claim-vs-spec AND claim-vs-evidence).
- **Asymmetry**: proof attempts to verify the claim is supported; falsification attempts to break it via counterexample / hidden dependency / contract mismatch / YAGNI pressure (`CLAUDE.md` §"QA Discipline"). Two separate fresh-context spawns.
- **Domain**: Go-only.

#### `go-research-agent` (loaded into THIS session)

- **Evidence sources required**: Hylla → `LSP`/gopls → `git diff` → `Read`/`Grep`/`Glob` for non-Go → Context7 + `go doc` → WebSearch/WebFetch. Hylla-first hard rule; misses go in `## Hylla Feedback`.
- **Output format**: Section 0 4-pass; closing certificate with Premises / Evidence / Trace / Conclusion / Unknowns.
- **Verification gates**: none — research is read-only. No QA twins auto-created.
- **Spec/pre-build contract**: research description must carry the **research question** + evidence-order expectations + deliverable shape. This IS a kind of spec (a research-output contract) but small in scope.
- **Notable**: explicit "options + trade-offs, NOT decisions" rule. Research never recommends.
- **Domain**: Go-only labeling, but most of the work transfers to other languages.

#### `closeout-agent`, `gopls-worktree-agent`, `commit-message-agent`

- `closeout-agent`: shared (lang-aware), orchestrator-managed today (`default-go.toml:[agent_bindings.closeout]` is `agent_name = "orchestrator-managed"`). Aggregates ledger / refinements / Hylla feedback at drop end. No spec contract — closeout is rollup work.
- `gopls-worktree-agent`: Go-only utility for keeping the gopls daemon in sync across worktrees. Not a substantive cascade role.
- `commit-message-agent`: referenced in every binding's `commit_agent = "commit-message-agent"` field but the agent definition file is not confirmed present on this machine (Drop-4+ scope per `CLAUDE.md` Agent Bindings table; `model = "haiku"`). When it lands it will read git diff + structural fields and emit a Conventional-Commits message; no spec required, no Section 0 scaffold.

#### FE counterparts (mentioned only)

`fe-builder-agent`, `fe-planning-agent`, `fe-qa-proof-agent`, `fe-qa-falsification-agent` — symmetric Go agents with FE evidence sources (MDN / CanIUse / ESLint / Vitest / Playwright) instead of Go ones. Not in Tillsyn's current scope (Tillsyn is Go-only today).

### Cascade-structural fields on the bindings (default-go.toml)

Each `[agent_bindings.<kind>]` carries: `agent_name`, `model`, `effort`, `tools` (placeholder; Drop 4 gates), `max_tries`, `max_budget_usd`, `max_turns`, `auto_push`, `commit_agent`, `blocked_retries`, `blocked_retry_cooldown`. Each `[agent_bindings.<kind>.context]` carries: `parent`, optionally `parent_git_diff` (build only — REV-4 excludes QA), `ancestors_by_kind`, `delivery`, `max_chars`, `max_rule_duration`.

**No prompt-shaping field exists today.** No `spec_template`, no `acceptance_criteria_required = true`, no `pre_build_artifacts = [...]`. The closest thing is the `[context]` rules (parent + ancestors-by-kind), which are evidence-routing rules, not spec rules.

The `agents.toml` sketch (drop_4c_6/SKETCH.md) §3 lists runtime-config fields only; §7 open question 6 explicitly defers the prompt-shaping question to this research deliverable.

### `default-generic.toml`

Intentionally omits `[agent_bindings]` entirely (`default-generic.toml:326-336`) — agent identities are language-specific. So the SDD question is currently "what should `default-go.toml`'s bindings carry, plus what should the canonical `~/.claude/agents/go-*.md` files carry."

---

## Part 3 — Mapping SDD Principles onto Cascade Kinds

Per kind: **already aligned** / **gaps** / **tensions** / **where the spec would live**.

### `plan`

- **Already aligned**: planner OUTPUT is structurally close to an SDD task-list (action items with `paths`, `packages`, `blocked_by`, acceptance criteria as prose). `plan` auto-creates `plan-qa-proof` + `plan-qa-falsification` — that IS the gated-phase + human/QA review pattern. Hylla-first evidence sourcing parallels SDD's "spec must reflect reality, not vibes." `[context]` rules with `ancestors_by_kind = ["plan"]` give a sub-plan upward visibility into its enclosing plan — the spec hierarchy SDD relies on.
- **Gaps**: planner INPUT has no enforced spec shape. Current entry is orchestrator prose; nothing checks for goals / user journeys / architecture / stack / constraints. `/plan-from-hylla` is a slash command, not a contract. No round-trip from spec → plan → build → done verification (Tessl's spec-as-source extreme).
- **Tensions**:
  - With **YAGNI / smallest-concrete-design**: SDD's "write a spec first" can become heavy-upfront-design (the ThoughtWorks anti-pattern). Tillsyn explicitly prefers small, iterative droplets. Mitigation: bias to spec-first or spec-anchored, not spec-as-source.
  - With **Tillsyn-first coordination**: a separate spec file outside Tillsyn duplicates state. If specs land, they belong as an action-item description field (post-Drop-1.75 metadata) or a `[agents.<kind>].spec_template = "..."` referenced from an in-repo `workflow/` file — not as a free-floating doc.
  - With **Section 0**: low tension. Section 0 is a reasoning trace; a spec is a contract. They compose: the Proposal pass cites the spec as a Premise.
- **Where the spec would live** (analysis only): four candidates — (i) action-item `description` shape enforced by template (planner contract); (ii) new `[agents.plan].spec_template` field referencing an in-repo `workflow/specs/<id>.md`; (iii) new `metadata.spec_ref` pointer on the action item; (iv) external file the planner agent reads at spawn (closest to Spec-Kit's `/specify` artifact). Each has different cost / discoverability / round-trip implications.

### `build`

- **Already aligned**: builder reads action-item description (paths/packages/criteria) — that is the closest thing to a spec the cascade has today. `mage ci` enforces an executable-conformance gate (~ SDD's "spec must be testable"). QA twins close the loop on whether the build matched the description.
- **Gaps**: description is honor-system prose; nothing enforces it carries acceptance criteria the build can be checked against. `parent_git_diff = true` pre-stages diff but does not pre-stage a spec. No "this build implements spec section X" linkage.
- **Tensions**:
  - With **Hylla-first**: SDD specs may stipulate behavior not yet in the codebase; Hylla can't ground forward-looking specs. Mitigation: spec lives in the action-item description, Hylla grounds the existing-code parts the build extends.
  - With **YAGNI**: SDD's "spec as source-of-truth" pressure can encourage over-specification. The build kind already has the YAGNI guardrail in CLAUDE.md ("smallest concrete design"); a spec scaffold would need to inherit it.
  - With **Greedy decoding**: no tension; spec doesn't change decoding.
  - With **Section 0**: low tension; spec becomes an Evidence cite.
- **Where the spec would live**: the action-item description today carries the build contract. SDD-style elevation would either (i) formalize description sub-fields (acceptance criteria block; conformance check block) per a template-validated shape, or (ii) reference an external `workflow/specs/<id>.md` from the description.

### `plan-qa-proof`

- **Already aligned**: QA proof verifies the planner's claim is supported by evidence — the same shape as SDD's "review the spec for completeness." Hylla-first plus independent evidence pull (no `parent_git_diff` per REV-4).
- **Gaps**: proof attacks the planner's reasoning, not an upstream spec. If a spec existed, proof could verify spec-coverage (does the plan address every spec item?) in addition to evidence-completeness.
- **Tensions**: low — QA proof gains a second axis (claim-vs-evidence AND claim-vs-spec) without losing the first.
- **Where the spec would live**: same as `plan`; QA proof reads the same artifact upstream.

### `plan-qa-falsification`

- **Already aligned**: explicitly attacks via counterexample / hidden dependency / YAGNI pressure. Falsification's natural target list extends cleanly to spec attacks (does the spec under-constrain? does it over-constrain? are there spec sections the plan silently drops?).
- **Gaps**: same as plan-qa-proof — no upstream spec to falsify against.
- **Tensions**: low. Adding spec-attack to the existing list does not remove existing attacks.
- **Where the spec would live**: same as `plan`.

### `build-qa-proof`

- **Already aligned**: verifies the build's claim against evidence; mage-ci pass is the executable-conformance proof-point.
- **Gaps**: verifies against the build's closing certificate, not against an upstream spec. Spec-conformance check (does the implementation satisfy each spec acceptance criterion?) would be a new axis.
- **Tensions**:
  - With the **REV-4 "QA fetches its own diff" rule**: minimal — QA already fetches independently; reading a spec file is one more read.
  - With **YAGNI**: if the spec is just a duplicate of the action-item description, the second axis is busywork. Mitigation: spec exists only when the description references one.
- **Where the spec would live**: same as `build`.

### `build-qa-falsification`

- **Already aligned**: counterexample / hidden-dep / contract-mismatch attacks. Spec-attack extends the same family naturally.
- **Gaps**: no spec-conformance attack today.
- **Tensions**: same low-tension story as build-qa-proof.
- **Where the spec would live**: same as `build`.

### `research`

- **Already aligned**: research description carries question + evidence-order + deliverable shape — already SDD-shaped (a small, scoped spec for a single research question).
- **Gaps**: minor. Research is read-only and produces findings, not implementations. SDD's gated-phase machinery doesn't add much.
- **Tensions**:
  - With **"options not decisions"**: SDD's spec-as-source extreme would let a research finding *be* the next spec. Tillsyn's research rule is "compile findings; orchestrator decides." Keep that boundary; SDD does not require research to decide.
- **Where the spec would live**: research already has its spec-equivalent (the action-item description). No new field needed.

### `commit`

- **Already aligned**: Conventional Commits is itself a tiny spec for the message format. `commit_agent = "commit-message-agent"` is wired template-side.
- **Gaps**: none meaningful — commits are derivative, not generative.
- **Tensions**: none.
- **Where the spec would live**: not applicable. The only "spec" is the Conventional Commits format string in `AGENTS.md` lines 43-49.

---

## Cross-Kind Themes

1. **The cascade already practices a weak form of SDD.** Action-item descriptions are de-facto specs; QA twins are de-facto phase gates; Hylla-first is de-facto evidence-grounding. What's missing is a formalized **shape** for the description (acceptance-criteria sub-block, conformance-check sub-block) and an enforced **link** from spec to plan to build.

2. **Tillsyn ships no prompt content today.** Substantive role definitions live in `~/.claude/agents/*.md` on the dev's machine. Tillsyn ships per-spawn structural data (`system-prompt.md` body) plus an empty agent stub (`plugin/agents/<name>.md`). Any "make the cascade more SDD" decision splits along that fault line: prompt-content edits are `~/.claude/agents/*.md` (out of repo); prompt-routing or spec-shape edits are template / agents.toml / action-item-metadata work (in repo).

3. **`SystemPromptTemplatePath` already exists as a binding field** (`render.go:323` "future evolution"). It is the natural seam where an `agents.toml`-defined or template-defined spec scaffold could land without breaking existing render flow.

4. **Section 0, Hylla-first, Tillsyn-first, Greedy decoding, YAGNI all compose with SDD at low cost** — none of them conflict with spec-first or spec-anchored. They all conflict with spec-as-source (Tessl extreme) because that mode pressures big-bang spec authoring and round-trip code regeneration that Tillsyn's pre-MVP discipline rules out.

5. **SDD adds the most leverage at `plan`** — the cascade's entry phase is where SDD's `/specify` + `/plan` + `/tasks` shape would slot in cleanly. Build / QA gain a second verification axis (spec-conformance) but do not change shape. Research / commit are roughly unaffected.

---

## Sources

- [GitHub — Spec-driven development with AI: Get started with a new open source toolkit (Sept 2025)](https://github.blog/ai-and-ml/generative-ai/spec-driven-development-with-ai-get-started-with-a-new-open-source-toolkit/)
- [GitHub Spec-Kit repo](https://github.com/github/spec-kit) and [`spec-driven.md`](https://github.com/github/spec-kit/blob/main/spec-driven.md)
- [Martin Fowler — Understanding Spec-Driven Development: Kiro, spec-kit, and Tessl](https://martinfowler.com/articles/exploring-gen-ai/sdd-3-tools.html)
- [Augment Code — What Is Spec-Driven Development?](https://www.augmentcode.com/guides/what-is-spec-driven-development)
- [Augment Code — Claude Code for Spec-Driven Development: Capabilities and Limits](https://www.augmentcode.com/guides/claude-code-spec-driven-development)
- [arxiv 2602.00180 — Spec-Driven Development: From Code to Contract in the Age of AI Coding Assistants](https://arxiv.org/html/2602.00180v1)
- [Microsoft — Diving Into Spec-Driven Development With GitHub Spec Kit](https://developer.microsoft.com/blog/spec-driven-development-spec-kit)
- [InfoWorld — Spec-driven AI coding with GitHub's Spec Kit](https://www.infoworld.com/article/4062524/spec-driven-ai-coding-with-githubs-spec-kit.html)
- [Visual Studio Magazine — GitHub Open Sources Kit for Spec-Driven AI Development](https://visualstudiomagazine.com/articles/2025/09/03/github-open-sources-kit-for-spec-driven-ai-development.aspx)
- [Kinde — Beyond TDD: Why Spec-Driven Development is the Next Step](https://www.kinde.com/learn/ai-for-software-engineering/best-practice/beyond-tdd-why-spec-driven-development-is-the-next-step/)
- [testRigor — TDD vs BDD vs SDD](https://testrigor.com/blog/what-is-test-driven-development-tdd-vs-bdd-vs-sdd/)
- [The AI Agent Factory — Chapter 16: Spec-Driven Development with Claude Code](https://agentfactory.panaversity.org/docs/General-Agents-Foundations/spec-driven-development)
- [alexop.dev — Spec-Driven Development with Claude Code in Action](https://alexop.dev/posts/spec-driven-development-claude-code-in-action/)

In-repo references (read at research time):

- `internal/templates/builtin/default-go.toml:388-599` — `[agent_bindings.<kind>]` blocks.
- `internal/templates/builtin/default-generic.toml:326-336` — agent_bindings intentionally omitted.
- `internal/app/dispatcher/cli_claude/render/render.go:209-339` — system-prompt body + agent-stub assembly.
- `SPAWN_PIPELINE.md` § "Two Plugin Paths" + § "Per-Spawn Bundle Layout".
- `SEMI-FORMAL-REASONING.md` (canonical Section 0 spec).
- `CLAUDE.md` § "Cascade Tree Structure", § "QA Discipline", § "Code Understanding Rules", § "Build Verification".
- `workflow/drop_4c_6/SKETCH.md` § 7 open question 6 (the question this research answers).

## Hylla Feedback

- **Query**: `hylla_search_keyword` for `system_prompt SystemPrompt AgentPrompt prompt_template render_prompt` against `github.com/evanmschultz/tillsyn@main`.
  - **Missed because**: Hylla index reported `enrichment still running for github.com/evanmschultz/tillsyn@main` — i.e. the artifact was not query-ready during this research session.
  - **Worked via**: direct `Read` against `internal/app/dispatcher/cli_claude/render/render.go` (lines 1-340 in two reads) plus `Read` of `internal/templates/builtin/default-go.toml:380-599`, `default-generic.toml`, `SPAWN_PIPELINE.md`, `CLI_ADAPTER_AUTHORING.md`, `SEMI-FORMAL-REASONING.md`, `AGENTS.md`, `CLAUDE.md`, `workflow/drop_4c_6/SKETCH.md`. Bash `grep` was permission-denied for the orchestrator session, so `Grep`/`Glob` would have been first-line — `Grep` returned `No such tool available`.
  - **Suggestion**: Hylla "enrichment still running" is a cold-start condition — research subagents should fall back fast (this one did, ~1 query before giving up). Long-term: a per-artifact `is_query_ready` boolean returned by `hylla_artifact_overview` would let agents skip the doomed search call. Also, the research-agent toolchain in this session lacked `Grep` (returned "No such tool available") which forced a Bash-grep attempt that the sandbox denied — the research-agent definition advertises `Grep`/`Glob` but the runtime grant did not include them.

- **Ergonomic gripe**: `hylla_search_vector` rejected my call with `field must be summary, content, or docstring` when I passed only `search_types`/`fields` to `hylla_search`; the error message could clarify it was the embedding-`field` parameter (singular) that was missing, since `fields` (plural) was present in the call.
