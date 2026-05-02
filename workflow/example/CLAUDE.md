# `<PROJECT>` — Project CLAUDE.md (Cascade Workflow, MD-Only)

> **Generic template.** This file is a project-level `CLAUDE.md` for a project
> running the cascade workflow with Markdown files instead of a coordination
> runtime (no Tillsyn, no action-item DB, no MCP dispatcher). Copy into your
> project's primary worktree (e.g. `main/`) and replace every `<PROJECT>` /
> `<PACKAGE>` / `<TECH>` placeholder with your project's values.
>
> The cascade concept source is `AGENT_CASCADE_DESIGN.md`. The per-drop
> lifecycle is `drops/WORKFLOW.md`. This file owns orchestrator role
> boundaries, agent bindings, evidence sources, and language-specific quality
> rules — it does not re-describe per-phase mechanics.

This file lives in the **primary work checkout** at `<PROJECT_MAIN_PATH>` (e.g. `/path/to/<project>/main/`). Real coding, building, testing, and committing happens here. **The dev launches work orchestrators from this directory.** Sessions launched from a bare-root one level up (if any) are **steward orchestrators** with a different prompt and a different scope — cross-worktree oversight and merge-conflict help, not feature work.

## Coordination Model — At a Glance

This project does **not** use any coordination runtime. Three documents own the coordination model; they do not duplicate each other:

- **`PLAN.md`** (at project root or `main/`) — overarching drop tree (level_1 container drops + state + `blocked_by` + per-drop dir link). Updated *after* a drop closes or *after* a planner restructures the tree. Not edited mid-build.
- **`drops/WORKFLOW.md`** — canonical per-drop lifecycle (planner → plan-QA → discuss → revise → builder → build-QA → verify → closeout). Owns: drop directory shape, file lifecycles, phase order, the **Agent Spawn Contract** (preamble pasted into every subagent spawn), restart recovery.
- **`CLAUDE.md`** (this file) — orchestrator role boundaries, agent bindings, evidence sources, language quality rules, build discipline, commit format, safety. Does not own per-phase mechanics — those live in `drops/WORKFLOW.md`.

Per-drop work artifacts live under `drops/DROP_N_<NAME>/`. The directory is stamped from `drops/_TEMPLATE/` at Phase 1 start and persists through closeout.

- **Read `WIKI.md` + `PLAN.md` + `drops/WORKFLOW.md` at session start and after every compaction.** CLAUDE.md auto-loads; the other three do not — read them deliberately on the first turn after cold-start or compaction before substantive orchestration.
- **Use any in-session TodoWrite-style tracker for nothing.** Do NOT use Claude Code's built-in `TaskCreate` / `TaskUpdate` / `TaskList` / `TaskGet` / `TaskStop` / `TaskOutput` — they evaporate on compaction/restart. Decompose finer procedural granularity into atomic units inside the active drop's `PLAN.md` instead.
- **No markdown files outside `drops/` for work tracking.** Per-drop dirs are the worklog substrate.

## Drops and Droplets

- A **drop** is a container unit of work — one row in `PLAN.md`, one directory under `drops/`. Drops are declared in `PLAN.md` and refined in their own dir.
- A **droplet** is the cascade's atomic build unit — the smallest work a builder can finish in one shot. Target: one source file + its co-located test, a few blocks of change (~80 LOC net typical, ~200 LOC soft ceiling, ≤3 files). Droplets are **sub-package** — many droplets can target files in one Go package, and they serialize with explicit `blocked_by` because a package shares one compile.
- **Planner nodes** live at package level and above. **LLM QA** (plan-QA + build-QA) fires at package level and above. **Automated QA** (`mage ci` or its language equivalent) runs at package level, once per package, covering every droplet that touched the package.
- Drops nest infinitely. A planner either decomposes a drop into sub-drops or emits droplets at the leaves. Planner-calls-planner is the cascade's recursion primitive.
- **Atomic-granularity rule:** a droplet is "atomic" when a single builder subagent can finish it cleanly, its acceptance criteria are yes/no-verifiable, and its `paths` / `packages` footprint is explicit. If a droplet is too large, **add more droplets inside its parent plan** rather than stretching one droplet.
- Ordering: parent-child nesting (a parent cannot close while any child is incomplete) + `blocked_by` for sibling and cross-node ordering. Each dir with >1 immediate child also carries a `_BLOCKERS.toml` ledger mirroring the `PLAN.md` `Blocked by:` bullets — see `drops/WORKFLOW.md` § "`_BLOCKERS.toml` — Sibling Blocker Ledger." No `depends_on` field.
- State: per-drop `state` in the drop dir's `PLAN.md` header (`planning` / `building` / `done` / `blocked`); per-droplet `state` in the Planner section's row (`todo` / `in_progress` / `done` / `blocked`); container-level `state` in the project-root `PLAN.md`.

Full per-drop lifecycle is in `drops/WORKFLOW.md`. Cascade concept source is `AGENT_CASCADE_DESIGN.md`.

## Orchestrator-as-Hub

The parent Claude Code session launched by the dev from this directory is always **the orchestrator**. Every other role (builder, qa-proof, qa-falsification, planning, research) is a subagent spawned via the `Agent` tool.

**CRITICAL: The orchestrator NEVER writes source code.** The parent session must not use `Edit`, `Write`, or any other tool to modify source or test files. Every code change — every single one — goes through a builder subagent. Orchestrator reads code for planning/research and edits markdown only (this file, `WIKI.md`, `PLAN.md`, drop dir mds, `LEDGER.md`, `README.md`, agent `.md` files).

### Agent Bindings (Generic Cascade Defaults)

The cascade binds kinds → agents → models. Swap the language variant to match your project.

| Role | Agent (Go variant) | Agent (FE variant) | Model | Edits Source? |
|---|---|---|---|---|
| Planner | `go-planning-agent` | `fe-planning-agent` | sonnet (opus at L1 if budget allows) | No |
| Plan QA Proof | `go-qa-proof-agent` | `fe-qa-proof-agent` | opus | No |
| Plan QA Falsification | `go-qa-falsification-agent` | `fe-qa-falsification-agent` | opus | No |
| Builder | `go-builder-agent` | `fe-builder-agent` | sonnet | **Yes** (only role that does) |
| Build QA Proof | `go-qa-proof-agent` | `fe-qa-proof-agent` | sonnet (opus if droplet is high-risk) | No |
| Build QA Falsification | `go-qa-falsification-agent` | `fe-qa-falsification-agent` | sonnet (opus if droplet is high-risk) | No |
| Research | Claude's built-in `Explore` subagent | same | sonnet | No |
| Commit (post-Drop-N) | `commit-message-agent` | same | haiku | No |

The agents are **global** (`~/.claude/agents/`) and some reference coordination-runtime tools (e.g. `till_*`) this project does not use. Every spawn carries the override preamble from `drops/WORKFLOW.md` § "Agent Spawn Contract" — single canonical source, do not duplicate it here.

## Build-QA-Commit Loop

Per-drop lifecycle is canonical in `drops/WORKFLOW.md` (Phases 1–7: plan, plan-QA, discuss + cleanup, build, build-QA, verify, closeout). This file does not duplicate the phase steps.

**Follow WORKFLOW.md's phases in order, exactly as written. No skipped phases. No reordered phases. No shortcut paths.** If a phase looks redundant for a particular drop, return the question to the dev — do not unilaterally drop it.

**Code is NEVER committed or pushed without per-droplet QA passing first**, and **code-understanding index reingest is drop-end only** — both rules are enforced inside WORKFLOW.md's phases. Subagents never trigger reingest.

## Code-Understanding Index Baseline (Go Projects)

For Go projects, the primary committed-code evidence source is a code-understanding index (Hylla: `hylla_search`, `hylla_node_full`, `hylla_search_keyword`, `hylla_refs_find`, `hylla_graph_nav`). Non-Go projects substitute the language-appropriate index or fall back to Read/Grep/Glob + LSP.

- **Artifact ref** (Go + Hylla): `github.com/<org>/<project>@main`. Replace with your project's remote.
- **Ingest is drop-end only**, not per-droplet. Only the orchestrator triggers ingest. Always `enrichment_mode=full_enrichment`, always from the GitHub remote, never before `git push` + CI green. Subagents never trigger ingest.

### Code Understanding Rules

1. **All source code (primary language)**: use the code-understanding index first. Exhaust every search mode the index offers (vector, keyword, graph-nav, refs) before falling back to `LSP`, `Read`, `Grep`, `Glob`. **Whenever an index miss forces a fallback, the subagent records the miss in its closing comment** under a `## Hylla Feedback` heading inside the drop's `BUILDER_WORKLOG.md`.
2. **Changed since last ingest**: use `git diff`. The index is stale for those files until reingest.
3. **Non-primary-language files** (markdown, TOML, YAML, magefile/makefile, SQL): use `Read`, `Grep`, `Glob`, `Bash` directly.
4. **External semantics**: Context7 + language-native doc tool (`go doc`, MDN, `python -m pydoc`, etc.) + `LSP` for library and language questions the repo can't answer itself.
5. **`LSP` tool** (gopls / tsserver / pyright / etc. backed): symbol search, references, diagnostics, rename safety for live / uncommitted code. Auto-targets the active checkout.

## Evidence Sources

In order:

1. **Code-understanding index** (Hylla for Go) — committed repo-local code.
2. **`git diff`** — uncommitted local deltas / files changed since last ingest.
3. **Context7 + language-native doc tool + `LSP`** — external / language / tooling semantics.

## Semi-Formal Reasoning

For semantic, high-risk, or ambiguous work:

- **Premises** — what must be true.
- **Evidence** — grounded in the index / `git diff` / Context7 / doc tool / LSP.
- **Trace or cases** — concrete paths through the code.
- **Conclusion** — the claim.
- **Unknowns** — what remains uncertain. Routed to the orchestrator (subagents return Unknowns in their final response; orchestrator surfaces to dev).

Short and inspectable. Full Section 0 spec lives in `~/.claude/output-styles/tillsyn-flow.md § "Section 0 — SEMI-FORMAL REASONING (Pre-Body Block)"` — 5 passes for orchestrators (Planner / Builder / QA Proof / QA Falsification / Convergence), 4 passes for subagents (Proposal / QA Proof / QA Falsification / Convergence). The Agent Spawn Contract preamble (in WORKFLOW.md) requires Section 0 from every subagent — but Section 0 stays in the orchestrator-facing response **only**, never inside `PLAN.md` / `BUILDER_WORKLOG.md` / `BUILDER_QA_*.md` / `PLAN_QA_*.md` / `CLOSEOUT.md`.

## QA Discipline

**No build droplet is `done` without per-droplet QA passing.** This is a gate, not a suggestion. Two asymmetric passes, not duplicates:

- **QA Proof** — evidence completeness, reasoning coherence, trace coverage. Asks: *"does the evidence support the claim?"*
- **QA Falsification** — counterexamples, alternate traces, hidden dependencies, contract mismatches, YAGNI pressure. Asks: *"can I construct a case where this is wrong?"*

Plan-QA and build-QA both run as parallel proof + falsification spawns. Plan-QA writes round-suffixed files (`PLAN_QA_PROOF.md` for round 1, `PLAN_QA_PROOF_R2.md` for round 2, etc.; same pattern for falsification). Build-QA appends rounds to single durable files (`BUILDER_QA_PROOF.md`, `BUILDER_QA_FALSIFICATION.md`) using `## Droplet N.M — Round K` headings. **Never `git rm` QA files between rounds — every round stays in tree for audit.** Full file-lifecycle table in `drops/WORKFLOW.md`.

**Planner-level LLM QA.** LLM QA fires at planner nodes (package level and above), not below. A droplet that passes its package's automated gate AND passes the planner-above's build-QA is `done`. If a planner's descendants fail their gates, the planner is re-QA'd (ancestor re-QA).

## Orchestrator Role Boundaries

- **Orchestrator** (this parent Claude Code session) — plans, routes, delegates, cleans up. **Never edits source code.** May edit markdown docs (this file, `WIKI.md`, `PLAN.md`, drop dir mds, `README.md`, `LEDGER.md`, `REFINEMENTS.md`, agent `.md` files).
- **Builder subagent** — the ONLY role that edits source code. Spawned via the `Agent` tool with the spawn contract preamble + builder appendix.
- **QA subagents** (proof + falsification) — gated to QA roles. Read, verify, write to their own `*_QA_*.md` file, return verdict to orch, die. Never edit code.
- **Planner subagent** — fills the drop's `PLAN.md` Planner section (Phase 1) and revises it across plan-QA rounds (Phase 3). May spawn sub-planners (cascade recursion). Never edits code.
- **Dev / human** — approves design calls during plan-QA discussion (Phase 3), reviews build-QA findings (Phase 5).

## Project Structure

> Replace this section with your project's actual package map. The cascade
> doesn't prescribe a layout; it only assumes code is organized into packages
> (Go) / modules (FE) / equivalents that match the droplet sub-package rule.

### Package Map (Generic Example)

- `<entrypoint-package>/` — CLI or UI entry. All flag / route wiring here; dispatches into internal packages.
- `internal/<domain-package>/` — one package per bounded domain. Each package is an implementation detail — nothing here is a public API.
- `<build-automation-file>` at repo root — build automation (`magefile.go`, `justfile`, `Makefile`, `package.json` scripts, etc.).

### Import DAG

> Diagram your project's import graph. Keep it a DAG — no cycles, strictly
> layered. The cascade relies on package-level blocking, which requires
> clean package boundaries.

### File Breakdown

> Per-file LOC estimates help the planner size droplets. A file under ~400 LOC
> is usually one droplet; over ~400 LOC probably needs splitting before build.
> Update this table as real LOC land.

## Tech Stack

> Replace with your project's actual tech stack. Examples of what to list:

- **Language and version** — e.g. `Go 1.26+`, `Node 20+`, `Python 3.12+`.
- **Production dependencies** — CLI / UI / protocol libraries your code compiles against.
- **Dev tooling** — formatters, linters, test runners, coverage tools. Invoked via the build-automation tool (mage / just / make / npm scripts), not directly.

## Build Verification

Per-droplet verification (during build-QA, Phase 5 of WORKFLOW.md): builder runs the language's per-package build + test for the touched package. Drop-end verification (after all droplets pass build-QA, Phase 6 of WORKFLOW.md): the project's CI target (`mage ci`, `just ci`, `npm run ci`, etc.) from the primary worktree, then `git push`, then `gh run watch --exit-status` until CI green.

1. All relevant build-automation targets pass (discover via `mage -l`, `just --list`, `npm run`, etc.).
2. **NEVER run raw compiler / test-runner commands** (e.g. `go test`, `go build`, `go vet`, raw `pytest`, raw `vitest`). Always route through the build-automation tool. If a target has a bug, fix the target — don't bypass. No exceptions, orchestrator or subagent.
3. **NEVER run dev-only dogfood targets** (e.g. `mage install` for tools that promote a binary to `$GOBIN`). Orchestrator and every subagent must not invoke them. If a droplet description asks for it, stop and return control to the orchestrator — the dev runs these manually.
4. All build-QA rounds for every droplet have closed green.

Run the CI target before every push. Coverage gates (if any) flip on in a late-stage refinement drop; treat coverage as report-only until the gate lands.

## Language Development Rules

> Fill this section per your project's language. Preserve the categories:
> Structure + Style, Errors, Concurrency (if applicable), Tests, Build Discipline,
> Dependencies, Reference Lookups. Examples below are Go-flavored; swap for
> your language's idioms.

### Structure + Style

- **Interface-first boundaries** (Go: interfaces; FE: component/hook APIs), dependency inversion where warranted.
- **Smallest concrete design.** No abstraction for hypothetical future variation.
- **TDD-first** where practical. Ship small tested increments.
- **Idiomatic style** — follow your language's community conventions. Formatters and linters enforce layout; static analyzers catch the rest.
- **Doc comments** on every exported identifier, starting with the identifier name (Go: `// Name …`; TS: JSDoc `/** ... */`).

### Errors

- **Wrap** with language-appropriate error wrapping (`fmt.Errorf("context: %w", err)` in Go; `new Error(..., { cause })` in TS). Never flatten an error to a string when callers need to inspect.
- **Sentinels** / typed errors for expected conditions callers want to branch on.
- **Inspect** with structural matching (`errors.Is` / `errors.As` in Go; `instanceof` in TS) — never string-match error messages.
- **Never swallow.** If you genuinely want to discard an error, assign to `_` / `void` with a one-line comment explaining why.

### Concurrency (If Applicable)

- **Goroutines / async tasks are bounded.** No unbounded fire-and-forget. Use an `errgroup` / `Promise.all` with a concurrency limit.
- **Every async task is context-cancellable.** Long-running loops check their cancellation signal.
- **Cleanup in `defer` / `finally`.** File closers, mutex unlocks, cancel funcs — always on the line after the resource acquisition.
- **No shared mutable state without synchronization.**
- **Race detector / strict-mode always on** in tests.

### Tests

- Co-located test files. Table-driven for anything with input variants. Behavior-oriented assertions.
- Prefer in-memory fakes (`testing/fstest.MapFS`, mock service workers, etc.) over real IO; reserve real fixtures for one end-to-end integration test per binary / app.
- **Fixture-directory rule.** When a package needs real-file fixtures, they live next to the test that reads them (Go: `testdata/`; JS: `__fixtures__/`). No shared top-level fixtures dir — keep fixtures local to the test that owns them.

### Build Discipline

- Plain `<build-tool> <target>` from the repo root. No ad-hoc env overrides (no `GOCACHE=...`, no workspace-local cache dirs).
- If a target is missing or broken, add/fix the target — never bypass with a raw toolchain command.

### After Touching Code

- Run the CI target before handoff at drop-end (Phase 6 of WORKFLOW.md). After pushing: `gh run watch --exit-status` until green.

### Dependencies

- Ask the dev to run dependency updates (`go get`, `npm install <pkg>`, `pip install <pkg>`). No checksum bypass flags (`GOSUMDB=off`, `--no-verify`, `--trusted-host`).

### Reference Lookups

- **Context7** + language doc tool + `LSP` before any unfamiliar external API usage, after any test failure.

### Markdown Authoring

- Drop dir mds are markdown-first. Use fenced code blocks for snippets, tables for structured data, headings per `drops/WORKFLOW.md`. No HTML.

## Skill and Slash Command Routing

| Command | When to Use |
|---|---|
| `/qa-proof` | Proof-oriented QA (used inside subagent definitions; orchestrator typically just spawns the agent) |
| `/qa-falsification` | Falsification-oriented QA (same) |
| `/select-checkout` | Confirm the active visible checkout |
| `/gopls-sync` (Go-only) | Verify gopls targets the active worktree |
| `semi-formal-reasoning` | Explicit reasoning certificate (Section 0 shape) |

## Git Commit Format

Conventional-commit: `type(scope): message`. All lowercase except proper nouns and acronyms (HTTP, CLI, JSON, TUI). Concise — describe what changed, not how.

**Subject-line only. No body. No bullet lists in the commit message.** The diff records what changed file-by-file; the subject line carries the human summary. Do not enumerate per-file changes in a body — that content belongs in the PR description, WIKI changelog, or LEDGER entry, not in `git log`.

Types: `feat`, `fix`, `refactor`, `chore`, `docs`, `test`, `ci`, `style`, `perf`.

Examples:
- `feat(<scope>): <short description of the change>`
- `fix(<scope>): <what broke + the verb that fixed it>`
- `chore(deps): add <library>`
- `docs(drop-N): planner decompose into N droplets`
- `docs(drop-N): clear plan qa round K, route to planner`

No co-authored-by trailers. No period at end. No capitalized first word after the colon unless proper noun/acronym. Keep the subject under ~72 chars when possible — if it won't fit, the change is probably too bundled and should be two commits.

## Safety

- Never delete files or directories without explicit dev approval.
- Never run commands outside the project's worktree root.
- Never push to any remote without explicit request.
- Keep secrets out of committed config files.

## Bare-Root and Worktree Discipline

> If your project uses a bare-root + visible-worktree layout (orchestration
> root at the bare repo, one or more named worktrees for real work), document
> the split here. If your project uses a single checkout, delete this section.

- **Bare repo** (if any) — steward orchestration root, not a coding checkout.
- **Primary worktree** (e.g. `main/`) — real coding / building / testing / committing happens here.
- Always confirm `pwd` is the intended worktree before edits, tests, commits, or LSP work.
- If checkout context is unclear, use `/select-checkout`.
- Multi-lane setups (parallel feature worktrees) coordinate via WORKFLOW.md's restart recovery + a steward orchestrator. Single-worktree projects skip this.

## Recovery After Session Restart

Filesystem + git, no coordination-runtime calls. Full procedure in `drops/WORKFLOW.md` § "Recovery After Restart". Quick form:

1. `git status` — uncommitted work.
2. `git log --oneline -20` — recent commits.
3. Read `PLAN.md` — container states.
4. List `drops/*/PLAN.md` headers — per-drop phase state.
5. Per active drop: presence of `PLAN_QA_*.md` = mid-plan-QA loop; absence + `BUILDER_WORKLOG.md` exists = mid-build; `CLOSEOUT.md` with `state: done` = drop closed.
6. Per active droplet: scan latest `## Droplet N.M — Round K` heading in `BUILDER_WORKLOG.md` + both `BUILDER_QA_*.md` to figure out next step.
