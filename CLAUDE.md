# Tillsyn — Project CLAUDE.md

This file lives in the **`main/` worktree** at `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/`. `main/` is the `main`-branch checkout — real coding, building, testing, and committing against `main` happens here. **Drop orchs whose scope is the `main` branch launch from this directory.** STEWARD (the persistent MD-writing orchestrator) does NOT launch from `main/` — STEWARD launches from the bare root one directory up and edits `main/`'s files from there. The bare-root `CLAUDE.md` (one directory up) carries the same rules body; only the preamble differs.

## Coordination Model

Drops follow the MD-only cascade workflow. Phase sequence, file lifecycle, spawn contract, and restart recovery live in `workflow/example/drops/WORKFLOW.md` — drop-orchs read it at the start of every drop.

- **Drop artifacts** (plans, worklogs, QA rounds, closeout) live as MD under `workflow/drop_N/`. Tracked in git, reviewed in PR.
- **Tillsyn coordinates** auth sessions, capability leases, and inter-orch MCP surfaces — not drop artifacts. The action-item tree in § "Cascade Tree Structure" below is the post-cascade target state, not how today's drops are tracked.
- Do NOT use Claude Code's built-in `TaskCreate` / `TaskUpdate` / `TaskList` / `TaskGet` / `TaskStop` / `TaskOutput` — they evaporate on compaction/restart. Finer granularity goes in the drop's `PLAN.md` droplet rows.
- **Read `WIKI.md` + `PLAN.md` + `workflow/example/drops/WORKFLOW.md` at session start and after every compaction.** CLAUDE.md auto-loads; those three do not.

### Discussion Mode (Chat-Primary Until TUI Ergonomics Land)

Cross-cutting decisions still park on a Tillsyn action item (description = converged shape, comments = audit trail of direct quotes). But the actual dev ↔ orchestrator back-and-forth happens **in chat** until the TUI comment flow is ergonomic enough to drive decisions through. Surface the full substance in chat — open decisions, options, tradeoffs, blockers — not just status pings. After each round with concrete decisions, mirror the converged points back into the action-item description and post a short audit-trail comment capturing dev direct quotes on corrections.

## Cascade Plan

The cascade (state-triggered autonomous agent dispatch) is designed in `PLAN.md` (lives in this directory). That plan is the source of truth for cascade architecture, drop ordering, and hard prerequisites. This `CLAUDE.md` documents the **current pre-cascade workflow** the orchestrator uses today.

## Cascade Tree Structure (Template Architecture)

This is the cascade's template architecture by action-item `kind` — the **post-Drop-2 target state**. **Drop 3 encodes this tree as a template** and **Drop 4's dispatcher reads it** to bind agents, gates, and `child_rules`. Pre-cascade, the orchestrator approximates the same shape manually, but the `kind` values written into Tillsyn today are constrained by what Drop 2 Go can read — see "Pre-Drop-2 Creation Rule" below. The Kind Hierarchy / Agent Bindings sections describe the target shape, not the current runtime writes.

### Pre-Drop-1.75 Creation Rule (Current HEAD)

The Go identifier rename `Task → ActionItem` shipped pre-Drop-1.75 (2026-04-18), flipping the `kind`/`scope` default enum string from `"task"` to `"actionItem"` sitewide. Drop 1.75 is the **kind-collapse** drop (reduces `kind_catalog` to `{project, action_item}` and deletes the template_libraries paths). Until Drop 1.75 lands, **every new action item under a project is created with `kind='actionItem', scope='actionItem'`**. Do NOT use the other registered kinds (`build-actionItem`, `subtask`, `qa-check`, `plan-actionItem`, `commit-and-reingest`, `a11y-check`, `visual-qa`, `design-review`, `phase`, `branch`, any `*-phase` variant, `decision`, `note`) even though they remain in `kind_catalog` — `main/scripts/drops-rewrite.sql` (dev-run after Drop 1.75 Go ships) rewrites every non-project kind to `action_item`.

**Role on description prose, not metadata (pre-Drop-2):** note role in the description (`Role: builder`, `Role: qa-proof`, `Role: qa-falsification`, `Role: qa-a11y`, `Role: qa-visual`, `Role: design`, `Role: commit`, `Role: planner`). Drop 2 lands `metadata.role` as a first-class field; the SQL hydrates it from each item's pre-collapse `kind`.

**Same-scope nesting is allowed.** A `actionItem` drop may nest under another `actionItem` drop — `actionItem` kind has no parent-scope restriction in `kind_catalog`. Same-scope nesting has live precedent (`subtask@subtask` under `subtask@subtask` in TILLSYN as of 2026-04-16). If the first nested `actionItem@actionItem` create is rejected by the MCP layer, fall back to `kind='subtask', scope='subtask'` for nested layers and flag the rejection.

### Kind Hierarchy

```
project                                 kind: project
└── drop (infinitely nestable)         kind: drop
      ├── plan-actionItem                     kind: plan-actionItem          ─→ agent: go-planning-agent          (opus)
      │   ├── plan-qa-proof             kind: qa-check           ─→ agent: go-qa-proof-agent          (opus)
      │   └── plan-qa-falsification     kind: qa-check           ─→ agent: go-qa-falsification-agent  (opus)
      │
      ├── drop (sub-drop)             kind: drop               (same shape, recurses infinitely)
      │
      └── actionItem (build-actionItem)             kind: actionItem               ─→ agent: go-builder-agent           (sonnet)
            ├── qa-proof                kind: qa-check           ─→ agent: go-qa-proof-agent          (sonnet)
            └── qa-falsification        kind: qa-check           ─→ agent: go-qa-falsification-agent  (sonnet)
```

### Required Children (Auto-Create Rules)

- **Every `drop`** auto-creates three children on creation: `plan-actionItem`, `plan-qa-proof`, `plan-qa-falsification`. Manual today; template `child_rules`-enforced in Drop 3.
- **Every `actionItem`** (build-actionItem) auto-creates two children on creation: `qa-proof`, `qa-falsification`.
- `plan-qa-proof` and `plan-qa-falsification` are `blocked_by: plan-actionItem` — they fire in parallel after the plan-actionItem completes.
- `qa-proof` and `qa-falsification` under a build-actionItem are `blocked_by: actionItem` — they fire in parallel after the build-actionItem completes **and** its post-build gates pass (see below).
- Drops nest infinitely. A planner creates sub-drops when decomposition needs to continue, or build-tasks when the work is granular enough.

### Agent Bindings

Pre-cascade: orchestrator spawns these manually via the `Agent` tool using Tillsyn auth credentials in the prompt.
Post-Drop-3: the template binds kinds → agents; the dispatcher spawns them on `in_progress` transitions.

| Kind                                      | Agent                                             | Model  | Role      | Edits Code? |
| ----------------------------------------- | ------------------------------------------------- | ------ | --------- | ----------- |
| `plan-actionItem` (drop-level)            | `go-planning-agent`                               | opus   | `planner` | No          |
| `qa-check` under `plan-actionItem`        | `go-qa-proof-agent` / `go-qa-falsification-agent` | opus   | `qa`      | No          |
| `actionItem` (build-actionItem)           | `go-builder-agent`                                | sonnet | `builder` | **Yes**     |
| `qa-check` under `actionItem`             | `go-qa-proof-agent` / `go-qa-falsification-agent` | sonnet | `qa`      | No          |
| commit-agent _(Drop-4+, post-build gate)_ | `commit-message-agent`                            | haiku  | `commit`  | No          |

### Post-Build Gates (Deterministic, Between Build-ActionItem And Its QA)

After a build-actionItem reports success, before its `qa-*` children become eligible, gates run programmatically. No LLM except the commit agent.

1. **`mage ci`** — on fail, the build-actionItem moves to `failed`, gate output posted as a comment.
2. **Commit** — commit-agent (haiku) forms the message; system runs `git add` + `git commit`. Pre-cascade: orchestrator + dev do this manually (see Git Management (Pre-Cascade) below).
3. **Push** — `git push` when the template's `auto_push = true`. Pre-cascade: manual.
4. **Hylla reingest** — NOT per-actionItem. Drop-end only, orchestrator-run, after `gh run watch --exit-status` is green. See "Cascade Ledger + Hylla Feedback" + "Drop Closeout" below. Agents never call `hylla_ingest`.

Only after all gates pass do the build-actionItem's QA children fire.

### Blocker Semantics

- **Parent-child** — a parent cannot move to `complete` while any child is incomplete or `failed`. Always-on parent-blocks-on-failed-child arrives in Drop 1.
- **`blocked_by`** — the only sibling and cross-drop ordering primitive. Planner sets these at creation time; dispatcher adds runtime blockers when file/package locks conflict (Drop 4+).
- **File- and package-level blocking** — sibling build-tasks sharing a file in `paths` OR a package in `packages` MUST have an explicit `blocked_by` between them. Plan QA falsification attacks missing blockers. Package-level locking exists because a single Go package (e.g. `internal/domain` with ~25 files) shares one compile — editing different files in the same package still breaks the other agent's test run.

### State-Trigger Dispatch

Moving an action item to `in_progress` is the dispatch trigger (Drop 4+). Pre-cascade, the orchestrator IS the dispatcher — it reads the kind, picks the binding above, moves the item to `in_progress`, and spawns the subagent via the `Agent` tool with Tillsyn auth credentials and Hylla artifact ref in the prompt.

## Tillsyn Project

The tillsyn project was **reset in Drop 0** — the prior messy project (`a0cfbf87-b470-45f9-aae0-4aa236b56ed9`, `default-go` template) was renamed to `TILLSYN-OLD` and a fresh, template-free project was created. Retiring `TILLSYN-OLD` via delete or archive is a Drop 10 refinement (project lifecycle ops bullet).

- **Project ID**: `a5e87c34-3456-4663-9f32-df1b46929e30`
- **Template**: none (fresh project, no template bound)
- **Slug**: `tillsyn`
- **Kind**: `go-project`

## Hylla Baseline

- **Artifact ref**: `github.com/evanmschultz/tillsyn@main` — Hylla resolves `@main` to the latest ingest automatically. Do not track snapshot numbers or commit hashes here.
- **Also stored on the Tillsyn project metadata** under `metadata.hylla_artifact_ref` so planners read it programmatically rather than copy-paste from this file.
- **Ledger**: `LEDGER.md` tracks per-drop cost, node count (total / code / tests / packages), orphan deltas, refactors, and drop descriptions. Populated by STEWARD post-merge from the drop-orch's finalized `DROP_N_LEDGER_ENTRY` description.
- **Hylla feedback**: `HYLLA_FEEDBACK.md` aggregates subagent-reported Hylla ergonomics and search-quality issues. Subagents report misses in their closing comment; drop-orch rolls them up into the `DROP_N_HYLLA_FINDINGS` description at drop end; STEWARD writes the MD post-merge.

### Code Understanding Rules

1. **All Go code**: use Hylla MCP (`hylla_search`, `hylla_node_full`, `hylla_search_keyword`, `hylla_refs_find`, `hylla_graph_nav`) as the primary source for committed-code understanding. If Hylla does not return the expected result on the first search, exhaust every Hylla search mode — vector (`hylla_search` with `search_types: ["vector"]`), keyword (`hylla_search_keyword`), graph-nav (`hylla_graph_nav`), refs (`hylla_refs_find`) — before falling back to `LSP`, `Read`, `Grep`, `Glob`. **Whenever a Hylla miss forces a fallback, the subagent must record the miss in its closing comment** under a `## Hylla Feedback` heading so the orchestrator can aggregate it at drop end.
2. **Changed since last ingest**: use `git diff` for files touched after the last Hylla ingest. Hylla is stale for those files until reingest.
3. **Non-Go code** (markdown, TOML, YAML, magefile, SQL, etc.): use `Read`, `Grep`, `Glob`, `Bash` directly. Hylla doesn't cover non-Go files.
4. **External semantics**: Context7 + `go doc` + `LSP` for library and language questions the repo can't answer itself.
5. **`LSP` tool** (gopls-backed, provided by the `gopls-lsp@claude-plugins-official` plugin): symbol search, references, diagnostics, rename safety, definitions for live / uncommitted code. Auto-targets the active checkout (`main/`). Subagents: use `LSP` rather than shelling out to `gopls` or scraping with `grep`/`rg`.

## Build-QA-Commit Discipline

**CRITICAL: No droplet is `done` without per-droplet QA passing. Push + `gh run watch` + Hylla reingest are drop-end only — full sequence in `workflow/example/drops/WORKFLOW.md` Phases 4–7.**

Per-droplet (Phases 4–5):

1. **Build** — builder subagent implements the droplet.
2. **QA Proof + Falsification (parallel)** — both must pass.
3. **Fix** — if either QA fails, respawn builder, re-run QA until both green.
4. **Commit** — `git add` the specific changed files, commit with conventional-commit format. No push yet.

Drop-end (Phases 6–7):

5. **`mage ci` locally** — must pass clean.
6. **Push + `gh run watch --exit-status`** — once, for the whole drop's work. No ingest on red CI.
7. **Hylla reingest** — drop-end only, from the remote, `enrichment_mode=full_enrichment`.

No skipped QA. No per-droplet push. No claiming done without both QA passes.

## Cascade Ledger + Hylla Feedback

Drop-orch owns the drop end-to-end per `workflow/example/drops/WORKFLOW.md` Phases 1–7, including closeout (Phase 7) — aggregating Hylla feedback, refinements, ledger entry, and wiki changelog into `workflow/drop_N/CLOSEOUT.md` before the PR merges.

**STEWARD is post-merge consolidation + validation.** After the drop's PR merges, STEWARD runs on `main`: reads `workflow/drop_N/CLOSEOUT.md`, splices the aggregated content into the top-level MDs (`LEDGER.md`, `REFINEMENTS.md`, `HYLLA_FEEDBACK.md`, `WIKI_CHANGELOG.md`, `HYLLA_REFINEMENTS.md`, `WIKI.md`), validates the splice, `git worktree remove drop/N`. That's it. STEWARD does not touch drop branches, does not run CI, does not call `hylla_ingest`. Full STEWARD spec: `STEWARD_ORCH_PROMPT.md`.

**Subagent responsibility:** every closing comment includes a `## Hylla Feedback` section. `None — Hylla answered everything needed.` if clean; otherwise record each miss:

- **Query**: tool name + key inputs.
- **Missed because**: your hypothesis (wrong search mode, schema gap, missing summary, stale ingest, etc.).
- **Worked via**: the fallback tool + inputs that found the thing.
- **Suggestion**: one-liner for what Hylla could do better.

Explicit "no miss" is still useful signal. Ergonomic-only gripes (awkward parameters, confusing response shapes, weird IDs) also go here.

## Drop Closeout

Drop-orch closes the drop per `workflow/example/drops/WORKFLOW.md` Phase 7 — aggregate Hylla feedback + refinements, write `CLOSEOUT.md`, flip drop state to `done`, merge PR.

**Hylla ingest invariants (inviolable):**

- Always `enrichment_mode=full_enrichment`. Never `structural_only`.
- Always source from the GitHub remote (`github.com/evanmschultz/tillsyn@main`). Never from a local working copy.
- Never before `git push` + `gh run watch --exit-status` green.
- Only the drop-orch calls `hylla_ingest`. Subagents never do. STEWARD never does.

## Git Management (Pre-Cascade)

Until the cascade dispatcher takes over commits (`PLAN.md` Drop 11), **orchestrator + dev manage git manually**. The orchestrator does not commit from its own session — it asks the dev, or spawns a builder subagent when code changes are needed. Clean git state (for the files an action item declares) is a precondition for creating an action item; the orchestrator checks `git status --porcelain <paths>` before creation and asks the dev to clean up if dirty.

## Orchestrator-as-Hub Architecture

The parent Claude Code session launched by the dev from this directory is always **the orchestrator**. There is no `.claude/agents/orchestration-agent.md` file — the orchestrator is defined by the invocation context, not by a markdown spec. Every other role (builder, qa, planner, closeout, research) is a subagent spawned via the `Agent` tool.

**CRITICAL: The orchestrator NEVER writes Go code.** The parent session must not use `Edit`, `Write`, or any other tool to modify `.go` source or test files. Every code change — every single one — goes through a builder subagent via the `Agent` tool. Orchestrator reads code for planning and research only.

**Markdown doc ownership is split between drop-orch (drop branch) and STEWARD (`main` post-merge).** Drop-orchs (`DROP_N_ORCH`) own per-drop artifact content in `main/workflow/drop_N/` and any architecture-MD edits (`CLAUDE.md`, `PLAN.md`, `AGENT_CASCADE_DESIGN.md`, `STEWARD_ORCH_PROMPT.md`, `workflow/README.md`, `workflow/example/**`) when the drop's scope touches process — all on the drop branch, flowing to `main` via PR merge. STEWARD (persistent continuation orchestrator — `STEWARD_ORCH_PROMPT.md`) runs post-merge on `main`, reads `main/workflow/drop_N/` content, collates it into the six top-level MDs (`LEDGER.md`, `REFINEMENTS.md`, `HYLLA_FEEDBACK.md`, `WIKI_CHANGELOG.md`, `HYLLA_REFINEMENTS.md`, plus `WIKI.md` curation), then `git worktree remove drop/N` after drop-orch has deleted the remote + local branch refs.

### How It Works

1. Orchestrator plans, routes, delegates, and cleans up. Reads code + Hylla for research. Creates Tillsyn action items. Spawns subagents. Coordinates results.
2. Subagents are ephemeral — they spawn, read their actionItem, do work, update the actionItem, die.
3. ActionItem state is the signal. On terminal state, the subagent sets `metadata.outcome` and moves to `done` or `failed` (once Drop 1 lands, `failed` will be a real terminal state; until then, failures are represented in metadata).
4. Subagents do not poll or watch anything. Read actionItem at spawn, execute, update, return.
5. Only the orchestrator uses attention items (human approval + inter-orchestrator coordination).

### Agent State Management

Every subagent manages its own Tillsyn action-item state. The orchestrator can't move role-gated items.

**Spawn prompt vs. action-item description split:** the spawn prompt carries ephemeral fields (action_item_id, auth credentials, working directory, move-state directive); the action-item description carries durable content (Hylla artifact ref, paths, packages, acceptance criteria, mage targets, cross-references). Rule: if a field changes every spawn, put it in the prompt; if it's stable across time and authors, put it in the description.

Full contract — exact spawn-prompt fields, exact description fields, spawn-gate checks — lives in each agent file at `~/.claude/agents/*.md` under "Required Prompt Fields" and "Spawn Prompt vs Action-Item Description Split." Don't duplicate it here.

## ActionItem Lifecycle (Current HEAD)

Three terminal-reachable states today: `todo`, `in_progress`, `done`. A fourth state `failed` lands in Drop 1 of the cascade plan. Until then:

- **Success**: set `metadata.outcome: "success"`, update `completion_contract.completion_notes`, move to `done`.
- **Failure**: set `metadata.outcome: "failure"`, note details in `completion_notes`. Currently the actionItem stays in `in_progress` with a failure-flavored outcome; Drop 1 adds the real `failed` transition.
- **Blocked**: set `metadata.outcome: "blocked"` + `metadata.blocked_reason`, report to orchestrator, stop.
- **Supersede** (post-Drop-1): human-only CLI `till actionItem supersede <id> --reason "..."` unsticks `failed → complete`. Before Drop 1 this doesn't exist.

No parent can move to terminal-success if any child is in a failure/blocked state — enforcement becomes always-on in Drop 1.

## Paths and Packages (Drop-1 Target)

Today, builders and planners track affected code loosely in metadata. In Drop 1, `paths []string` and `packages []string` become first-class domain fields on every action item, set by the planner, readable by builder + QA, and required for the file- and package-level blocking the cascade relies on. Until Drop 1 ships, note affected paths in `completion_notes` — the cascade plan (`PLAN.md`, Section 5 + Section 17.1) is the contract.

## Auth and Leases

- One active auth session per scope level at a time.
- Orchestrator cleans up all child auth sessions and leases at end of phase/run.
- Auth auto-revoke on terminal state is a Drop-1 item; until then, the orchestrator manually revokes stale sessions.
- **Always report the auth session ID to the dev** when requesting or claiming auth. The dev needs visibility into active sessions.

## Coordination Surfaces

**Subagents:**

- `till.action_item` — read actionItem, update metadata, move state.
- `till.comment` — result comments on their own actionItem.
- No attention_items, no handoffs, no @mentions, no downward/sideways signaling.

**Orchestrator (this session):**

- `till.action_item` — create/update tasks, read state, move phases.
- `till.comment` — guidance before spawning subagents.
- `till.attention_item` — inbox for human approvals.
- `till.handoff` — structured next-action routing.
- `/loop` polling (60-120s cadence) for attention items during long-running work.

## Role Model

- **Orchestrator** — the human-launched CLI session. Plans, routes, delegates, cleans up. Never edits Go code. Drop-orchs edit MDs on their drop branch (artifact content in `main/workflow/drop_N/` + architecture MDs when scope touches process); STEWARD edits MDs on `main` post-merge (collation into top-level MDs + worktree cleanup).
- **Builder** — subagent. The ONLY role that edits Go code. Reads actionItem, implements, updates, dies.
- **QA Proof / QA Falsification** — subagents. Ephemeral. Read actionItem, review, update with verdict, die.
- **Planning** — subagent. Decomposes a drop into tasks with paths/packages/acceptance criteria.
- **Research** — Claude's built-in `Explore` subagent.
- **Human** — approves auth, reviews results, makes design decisions.

## Recovery After Session Restart

1. `till.capture_state` — re-anchor project and scope context.
2. `till.attention_item(operation=list, all_scopes=true)` — inbox state.
3. Check `in_progress` tasks for staleness.
4. Revoke orphaned auth sessions/leases.
5. Resume from current actionItem state.

## Claude Code Agents (Go Project)

Spawn via the `Agent` tool with `subagent_type`. There is no orchestration-agent row — the orchestrator is the parent session, not a subagent.

| Agent                | Subagent Type               | Purpose                                                 |
| -------------------- | --------------------------- | ------------------------------------------------------- |
| **Builder**          | `go-builder-agent`          | Ephemeral builder — the only role that edits Go code    |
| **Planning**         | `go-planning-agent`         | Hylla-first planning grounded in committed code reality |
| **QA Proof**         | `go-qa-proof-agent`         | Proof-completeness check — evidence supports the claim  |
| **QA Falsification** | `go-qa-falsification-agent` | Falsification attempt — try to break the conclusion     |

Inline (no subagent file):

- **research-agent** — Claude's built-in `Explore` subagent.

### QA Discipline

Two asymmetric passes, not duplicates:

- **QA Proof** (`go-qa-proof-agent`, `/qa-proof`) — evidence completeness, reasoning coherence, trace coverage.
- **QA Falsification** (`go-qa-falsification-agent`, `/qa-falsification`) — counterexamples, hidden deps, contract mismatches, YAGNI.

Run both for every build-actionItem. They are asymmetric — proof checks whether the evidence supports the claim; falsification tries to construct a counterexample. Spawn them as parallel subagents so each gets a fresh context window.

## Skill and Slash Command Routing

| Command                 | When to Use                                                |
| ----------------------- | ---------------------------------------------------------- |
| `/plan-from-hylla`      | Hylla-grounded planning                                    |
| `/qa-proof`             | Proof-oriented QA                                          |
| `/qa-falsification`     | Falsification-oriented QA                                  |
| `semi-formal-reasoning` | Explicit reasoning certificate for semantic/high-risk work |

## Semi-Formal Reasoning — Section 0 Response Shape

Every substantive response begins with a `# Section 0 — SEMI-FORMAL REASONING` block before the normal response body. Orchestrator-facing responses use five passes (`Planner` / `Builder` / `QA Proof` / `QA Falsification` / `Convergence`); subagent-facing responses use four (`Proposal` / `QA Proof` / `QA Falsification` / `Convergence`). Each pass uses the 5-field certificate: **Premises** / **Evidence** / **Trace or cases** / **Conclusion** / **Unknowns**.

**Canonical spec: `SEMI-FORMAL-REASONING.md`** (this directory). Read that file for the full rules, the subagent pass-through directive, and the Tillsyn artifact boundary (Section 0 reasoning lives in the orchestrator-facing response ONLY — never in Tillsyn descriptions, metadata, completion_notes, or comments).

Trivial-answer carve-out: one-line factual lookups and terse confirmations skip both Section 0 and the numbered body.

Subagents do NOT inherit CLAUDE.md. When delegating substantive work, the spawn prompt MUST include the Section 0 directive verbatim (the exact wording is in `SEMI-FORMAL-REASONING.md` §Subagent Pass-Through).

## Evidence Sources

In order:

1. **Hylla** — committed repo-local code.
2. **`git diff`** — uncommitted local deltas and files changed since last ingest.
3. **Context7 + `go doc` + gopls MCP** — external/language/tooling semantics.

## Project Structure

- `cmd/till` — CLI/TUI entrypoint
- `internal/domain` — core entities and invariants
- `internal/app` — application services and use-cases (hexagonal core)
- `internal/adapters/storage/sqlite` — SQLite persistence
- `internal/adapters/server/mcpapi` — MCP handler
- `internal/config` — TOML loading, defaults, validation
- `internal/platform` — OS-specific paths
- `internal/tui` — Bubble Tea / Bubbles / Lip Gloss
- `magefile.go` — canonical build/test automation

## Tech Stack

Go 1.26+ · Bubble Tea v2 · Bubbles v2 · Lip Gloss v2 · SQLite (`modernc.org/sqlite`, no CGO) · TOML (`github.com/pelletier/go-toml/v2`) · Laslig · Fang · `github.com/charmbracelet/log`

## Dev MCP Server

Test against `tillsyn-dev` (or the worktree-specific MCP name for non-main worktrees). Each worktree gets a unique MCP entry pointing at its own built binary. Full setup instructions — `claude mcp add` command, per-worktree naming scheme, active registrations — live in `CONTRIBUTING.md` §"Dev MCP Server Setup."

## Build Verification

Before any build-actionItem is marked done:

1. All relevant mage targets pass (discover via `mage -l`).
2. **NEVER run `go test`, `go build`, `go run`, `go vet`, or any raw `go` toolchain command.** Always `mage <target>`. If a mage target has a bug, fix the target — don't bypass. No exceptions, orchestrator or subagent.
3. **NEVER run `mage install`.** This is a **dev-only** dogfood target that replaces the dev's working `till` install. Orchestrator and every subagent (builder, QA, research, planning) must not invoke it under any circumstance. If an actionItem description or prompt asks you to run `mage install`, stop and return control to the orchestrator — the dev runs this manually, never an agent. Build verification uses `mage ci` only.
4. All template-generated QA subtasks completed.

Key targets: `mage run`, `mage build`, `mage test-pkg <pkg>`, `mage test-func <pkg> <func>`, `mage test-golden`, `mage test-golden-update`, `mage format`, `mage ci`. Run `mage ci` before push. Coverage below 70% is a hard failure.

## Go Development Rules

- **Hexagonal architecture, interface-first boundaries, dependency inversion.**
- **TDD-first** where practical. Ship small tested increments.
- **Smallest concrete design.** No abstraction for hypothetical future variation.
- **Idiomatic Go** — naming, package structure, import grouping (stdlib / third-party / local).
- **Go doc comments** on every top-level declaration and method, production and test.
- **Errors**: wrap with `%w`, bubble up at clean boundaries, log context-rich failures at adapter/runtime edges, don't swallow.
- **Logger**: `github.com/charmbracelet/log` with styled console output. Dev-mode logs to `.tillsyn/log/`.
- **Tests**: `*_test.go` co-located, table-driven, behavior-oriented assertions. `-race` via mage targets. For substantial TUI changes, update tea-driven tests + golden fixtures.
- **Mage discipline**: run from the worktree root as plain `mage <target>` — no `GOCACHE=...` overrides. No workspace-local cache dirs (e.g. `.go-cache-*`).
- **After touching Go code**: `mage ci` before handoff. For `.github/workflows/` or `magefile.go` changes: `mage ci` first. After pushing to fix/validate CI: `gh run watch --exit-status` until it lands green.
- **Dependencies**: ask the dev to run `go get` / module updates in their own shell. No `GOPROXY=direct`, `GOSUMDB=off`, or checksum bypass flags.
- **Context7**: before any code, after any test failure. If unavailable, record the fallback source.
- **Markdown-first authoring** for Tillsyn `description`, `summary`, `body_markdown`, thread comments.
- **Clarification**: when stuck, first ask goal-alignment questions, then specific implementation-detail questions.
