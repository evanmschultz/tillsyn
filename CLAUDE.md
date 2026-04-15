# Tillsyn — Project CLAUDE.md (main worktree)

This file lives in the **`main/` worktree** at `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/`. This is the primary work checkout — all real coding, building, testing, and committing happens here. **The dev launches orchestrators from this directory.** The bare-root `CLAUDE.md` (one directory up) carries the same rules body; only the preamble differs.

## Tillsyn Is the System of Record

All work is tracked in Tillsyn. No exceptions.

- No markdown files for work tracking, coordination, worklogs, or execution state.
- **Tillsyn = durable truth. Every piece of work gets a Tillsyn plan item before it starts.**
- **Claude Code built-in `TaskCreate` / `TaskUpdate` / `TaskList` are in-session scratch — use them alongside Tillsyn** when an orchestrator turn spans ≥3 discrete actions (Tillsyn mutations, file edits, commits, etc.). Tillsyn holds the durable *what*; built-in todos hold the procedural *how* for one session. Not a replacement — belt-and-suspenders while Tillsyn behavior is still stabilizing.
- **When work starts on a plan item, move it to `in_progress` immediately.** No items left in `todo` while being worked on. Same status discipline applies to in-session todos.

### Discussion Mode (Chat-Primary Until TUI Ergonomics Land)

Cross-cutting decisions still park on a Tillsyn plan item (description = converged shape, comments = audit trail of direct quotes). But the actual dev ↔ orchestrator back-and-forth happens **in chat** until the TUI comment flow is ergonomic enough to drive decisions through. Surface the full substance in chat — open decisions, options, tradeoffs, blockers — not just status pings. After each round with concrete decisions, mirror the converged points back into the plan-item description and post a short audit-trail comment capturing dev direct quotes on corrections.

## Cascade Plan

The cascade (state-triggered autonomous agent dispatch) is designed in `CLAUDE_MINIONS_PLAN.md` (lives in this directory — moved from bare-root in Slice 0 so it is git-tracked with the rest of the repo). That plan is the source of truth for cascade architecture, slice ordering, and hard prerequisites. This `CLAUDE.md` documents the **current pre-cascade workflow** the orchestrator uses today.

## Cascade Tree Structure (Template Architecture)

This is the cascade's template architecture by plan-item `kind`. **Slice 3 encodes this tree as a template** and **Slice 4's dispatcher reads it** to bind agents, gates, and `child_rules`. Pre-cascade, the orchestrator approximates the same shape manually using whatever generic kinds the fresh project starts with — the structure, blockers, and agent roles below are the contract.

### Kind Hierarchy

```
project                                 kind: project
└── slice (infinitely nestable)         kind: slice
      ├── plan-task                     kind: plan-task          ─→ agent: go-planning-agent          (opus)
      │   ├── plan-qa-proof             kind: qa-check           ─→ agent: go-qa-proof-agent          (opus)
      │   └── plan-qa-falsification     kind: qa-check           ─→ agent: go-qa-falsification-agent  (opus)
      │
      ├── slice (sub-slice)             kind: slice               (same shape, recurses infinitely)
      │
      └── task (build-task)             kind: task               ─→ agent: go-builder-agent           (sonnet)
            ├── qa-proof                kind: qa-check           ─→ agent: go-qa-proof-agent          (sonnet)
            └── qa-falsification        kind: qa-check           ─→ agent: go-qa-falsification-agent  (sonnet)
```

### Required Children (Auto-Create Rules)

- **Every `slice`** auto-creates three children on creation: `plan-task`, `plan-qa-proof`, `plan-qa-falsification`. Manual today; template `child_rules`-enforced in Slice 3.
- **Every `task`** (build-task) auto-creates two children on creation: `qa-proof`, `qa-falsification`.
- `plan-qa-proof` and `plan-qa-falsification` are `blocked_by: plan-task` — they fire in parallel after the plan-task completes.
- `qa-proof` and `qa-falsification` under a build-task are `blocked_by: task` — they fire in parallel after the build-task completes **and** its post-build gates pass (see below).
- Slices nest infinitely. A planner creates sub-slices when decomposition needs to continue, or build-tasks when the work is granular enough.

### Agent Bindings

Pre-cascade: orchestrator spawns these manually via the `Agent` tool using Tillsyn auth credentials in the prompt.
Post-Slice-3: the template binds kinds → agents; the dispatcher spawns them on `in_progress` transitions.

| Kind | Agent | Model | Role | Edits Code? |
|---|---|---|---|---|
| `plan-task` (slice-level) | `go-planning-agent` | opus | `planner` | No |
| `qa-check` under `plan-task` | `go-qa-proof-agent` / `go-qa-falsification-agent` | opus | `qa` | No |
| `task` (build-task) | `go-builder-agent` | sonnet | `builder` | **Yes** |
| `qa-check` under `task` | `go-qa-proof-agent` / `go-qa-falsification-agent` | sonnet | `qa` | No |
| commit-agent *(Slice-4+, post-build gate)* | `commit-message-agent` | haiku | `commit` | No |

### Post-Build Gates (Deterministic, Between Build-Task And Its QA)

After a build-task reports success, before its `qa-*` children become eligible, gates run programmatically. No LLM except the commit agent.

1. **`mage ci`** — on fail, the build-task moves to `failed`, gate output posted as a comment.
2. **Commit** — commit-agent (haiku) forms the message; system runs `git add` + `git commit`. Pre-cascade: orchestrator + dev do this manually (see Git Management (Pre-Cascade) below).
3. **Push** — `git push` when the template's `auto_push = true`. Pre-cascade: manual.
4. **Hylla reingest** — NOT per-task. Slice-end only, orchestrator-run, after `gh run watch --exit-status` is green. See "Cascade Ledger + Hylla Feedback" + "Slice End — Ledger Update Task" below. Agents never call `hylla_ingest`.

Only after all gates pass do the build-task's QA children fire.

### Blocker Semantics

- **Parent-child** — a parent cannot move to `complete` while any child is incomplete or `failed`. Always-on parent-blocks-on-failed-child arrives in Slice 1.
- **`blocked_by`** — the only sibling and cross-slice ordering primitive. Planner sets these at creation time; dispatcher adds runtime blockers when file/package locks conflict (Slice 4+).
- **File- and package-level blocking** — sibling build-tasks sharing a file in `paths` OR a package in `packages` MUST have an explicit `blocked_by` between them. Plan QA falsification attacks missing blockers. Package-level locking exists because a single Go package (e.g. `internal/domain` with ~25 files) shares one compile — editing different files in the same package still breaks the other agent's test run.

### State-Trigger Dispatch

Moving a plan item to `in_progress` is the dispatch trigger (Slice 4+). Pre-cascade, the orchestrator IS the dispatcher — it reads the kind, picks the binding above, moves the item to `in_progress`, and spawns the subagent via the `Agent` tool with Tillsyn auth credentials and Hylla artifact ref in the prompt.

## Tillsyn Project

The tillsyn project was **reset in Slice 0** — the prior messy project (`a0cfbf87-b470-45f9-aae0-4aa236b56ed9`, `default-go` template) was renamed to `TILLSYN-OLD` and a fresh, template-free project was created. Retiring `TILLSYN-OLD` via delete or archive is a Slice 10 refinement (project lifecycle ops bullet).

- **Project ID**: `a5e87c34-3456-4663-9f32-df1b46929e30`
- **Template**: none (fresh project, no template bound)
- **Slug**: `tillsyn`
- **Kind**: `go-project`

## Hylla Baseline

- **Artifact ref**: `github.com/evanmschultz/tillsyn@main` — Hylla resolves `@main` to the latest ingest automatically. Do not track snapshot numbers or commit hashes here.
- **Also stored on the Tillsyn project metadata** under `metadata.hylla_artifact_ref` so planners read it programmatically rather than copy-paste from this file.
- **Ledger**: `main/LEDGER.md` tracks per-slice cost, node count (total / code / tests / packages), orphan deltas, refactors, and slice descriptions. Populated by the orchestrator during the per-slice `SLICE <N> END — LEDGER UPDATE` task after ingest completes.
- **Hylla feedback**: `main/HYLLA_FEEDBACK.md` aggregates subagent-reported Hylla ergonomics and search-quality issues. Subagents report misses in their closing comment; the orchestrator rolls them up at slice end before running the slice-end ingest.

### Code Understanding Rules

1. **All Go code**: use Hylla MCP (`hylla_search`, `hylla_node_full`, `hylla_search_keyword`, `hylla_refs_find`, `hylla_graph_nav`) as the primary source for committed-code understanding. If Hylla does not return the expected result on the first search, exhaust every Hylla search mode — vector (`hylla_search` with `search_types: ["vector"]`), keyword (`hylla_search_keyword`), graph-nav (`hylla_graph_nav`), refs (`hylla_refs_find`) — before falling back to `LSP`, `Read`, `Grep`, `Glob`. **Whenever a Hylla miss forces a fallback, the subagent must record the miss in its closing comment** under a `## Hylla Feedback` heading so the orchestrator can aggregate it into `main/HYLLA_FEEDBACK.md` at slice end.
2. **Changed since last ingest**: use `git diff` for files touched after the last Hylla ingest. Hylla is stale for those files until reingest.
3. **Non-Go code** (markdown, TOML, YAML, magefile, SQL, etc.): use `Read`, `Grep`, `Glob`, `Bash` directly. Hylla doesn't cover non-Go files.
4. **External semantics**: Context7 + `go doc` + `LSP` for library and language questions the repo can't answer itself.
5. **`LSP` tool** (gopls-backed, provided by the `gopls-lsp@claude-plugins-official` plugin): symbol search, references, diagnostics, rename safety, definitions for live / uncommitted code. Auto-targets the active checkout (`main/`). Subagents: use `LSP` rather than shelling out to `gopls` or scraping with `grep`.

## Build-QA-Commit Discipline

**CRITICAL: Code is NEVER committed or pushed without QA completing first. Hylla ingest is slice-end only, not per-task.**

1. **Build** — builder subagent implements the increment.
2. **QA Proof** — `go-qa-proof-agent` verifies evidence completeness.
3. **QA Falsification** — `go-qa-falsification-agent` tries to break the conclusion.
4. **Fix** — if QA finds issues, spawn another builder to fix, then re-run QA.
5. **Commit** — only after both QA passes clear: `git add` the specific changed files, commit with conventional-commit format (pre-cascade: orchestrator + dev; post-Slice-4: commit-agent).
6. **Push** — `git push` so CI runs.
7. **CI green** — `gh run watch --exit-status` until CI lands green. If CI fails, fix before continuing — no ingest on a red commit.
8. **Update Tillsyn** — checklist + metadata + lifecycle state. If it's not in Tillsyn, it didn't happen.
9. **Move on to the next task.** Per-task Hylla reingest does NOT happen. Ingest happens once per slice, at slice end, inside the `SLICE <N> END — LEDGER UPDATE` task — see "Cascade Ledger + Hylla Feedback" and "Slice End — Ledger Update Task" below.

No batched commits. No deferred pushes. No skipped QA. No skipped CI watch. No claiming done in chat without Tillsyn reflecting it.

## Cascade Ledger + Hylla Feedback

Two per-slice artifacts live in `main/`:

- **`main/LEDGER.md`** — per-slice snapshot of project state, cost, and code-quality deltas. Populated by the orchestrator at slice end. Fields per slice: closed date, slice plan-item ID, ingest snapshot, ingest cost + cost-to-date, node counts (total / code / tests / packages), orphan delta, refactors, description, commit SHAs, notable plan-item IDs, unknowns forwarded.
- **`main/HYLLA_FEEDBACK.md`** — running log of Hylla ergonomics and search-quality feedback from subagents and the orchestrator. Each subagent that falls back from Hylla to `LSP` / `Read` / `Grep` / `Glob` records the miss in its closing comment under a `## Hylla Feedback` heading. The orchestrator aggregates those entries into `HYLLA_FEEDBACK.md` during the slice-end task, before calling ingest.

**Subagent responsibility:** in every closing comment, always include a `## Hylla Feedback` section. If you had no Hylla misses, write `None — Hylla answered everything needed.`. If you did, record each miss with:

- **Query**: tool name + key inputs.
- **Missed because**: your hypothesis (wrong search mode, schema gap, missing summary, stale ingest, etc.).
- **Worked via**: the fallback tool + inputs that found the thing.
- **Suggestion**: one-liner for what Hylla could do better.

Explicit "no miss" is still useful signal. Ergonomic-only gripes (awkward parameters, confusing response shapes, weird IDs) also go here.

## Slice End — Ledger Update Task

Every slice gets a final task named `SLICE <N> END — LEDGER UPDATE`. Orchestrator-role-gated. `blocked_by` every other task in the slice. Only runs once all siblings are `done`.

**Orchestrator steps when working this task:**

1. Move the task to `in_progress`.
2. Confirm all sibling tasks in the slice are `done`. Confirm `git status --porcelain` is clean.
3. Confirm every commit from this slice has landed on the remote branch.
4. Run `gh run watch --exit-status` on the latest CI run. Do NOT proceed unless CI is green.
5. Aggregate `## Hylla Feedback` sections from every subagent closing comment in this slice into `main/HYLLA_FEEDBACK.md` under a new `## Slice <N>` heading.
6. Call `hylla_ingest` on the remote ref `github.com/evanmschultz/tillsyn@main`. **ALWAYS full enrichment. NEVER `structural_only`. NEVER from a local working copy — always from remote, after push + CI green.**
7. Poll `hylla_run_get` via `/loop 120` while ingest progresses. When the run reports "nearly done" (enrichment stage entered), kill the loop and `ScheduleWakeup` once for the estimated remaining time.
8. When ingest completes, read `hylla_run_get` final result. Extract: ingest snapshot, cost (this run + lineage-to-date), node counts (total / code / tests / packages), orphan delta.
9. Append a new `## Slice <N> — <Title>` entry to `main/LEDGER.md` with the format shown there.
10. Mark the slice-end task `done`; reference the ledger entry in `completion_notes`.

**Hylla ingest invariants (repeat for emphasis):**

- Always `enrichment_mode=full_enrichment`.
- Always source from the GitHub remote.
- Never before `git push` + `gh run watch --exit-status` green.
- Only the orchestrator calls `hylla_ingest`. Subagents never do.

## Git Management (Pre-Cascade)

Until the cascade dispatcher takes over commits (`CLAUDE_MINIONS_PLAN.md` Slice 11), **orchestrator + dev manage git manually**. The orchestrator does not commit from its own session — it asks the dev, or spawns a builder subagent when code changes are needed. Clean git state (for the files a plan item declares) is a precondition for creating a plan item; the orchestrator checks `git status --porcelain <paths>` before creation and asks the dev to clean up if dirty.

## Orchestrator-as-Hub Architecture

The parent Claude Code session launched by the dev from this directory is always **the orchestrator**. There is no `.claude/agents/orchestration-agent.md` file — the orchestrator is defined by the invocation context, not by a markdown spec. Every other role (builder, qa, planner, closeout, research) is a subagent spawned via the `Agent` tool.

**CRITICAL: The orchestrator NEVER writes Go code.** The parent session must not use `Edit`, `Write`, or any other tool to modify `.go` source or test files. Every code change — every single one — goes through a builder subagent via the `Agent` tool. Orchestrator reads code for planning and research only. Markdown documentation edits (this file, `CLAUDE_MINIONS_PLAN.md`, agent `.md` files) are orchestrator-scope.

### How It Works

1. Orchestrator plans, routes, delegates, and cleans up. Reads code + Hylla for research. Creates Tillsyn plan items. Spawns subagents. Coordinates results.
2. Subagents are ephemeral — they spawn, read their task, do work, update the task, die.
3. Task state is the signal. On terminal state, the subagent sets `metadata.outcome` and moves to `done` or `failed` (once Slice 1 lands, `failed` will be a real terminal state; until then, failures are represented in metadata).
4. Subagents do not poll or watch anything. Read task at spawn, execute, update, return.
5. Only the orchestrator uses attention items (human approval + inter-orchestrator coordination).

### Agent State Management — Critical

Every subagent manages its own Tillsyn plan item state. The orchestrator can't move role-gated items (e.g. QA subtasks gated to `qa`).

**Split of concerns — spawn prompt vs. plan-item description:**

- The **spawn prompt** (what the orchestrator passes to the `Agent` tool) carries only spawn-unique and ephemeral fields. It does NOT duplicate content already in the plan-item description or project metadata.
- The **plan-item description** (what the agent reads via `till.auth_request(operation=claim)`) carries the durable task content: what to do, acceptance criteria, Hylla artifact ref, paths, packages, mage targets, cross-references.
- Rule of thumb: if a field changes every spawn, put it in the prompt. If it's stable across time and authors, put it in the description.

**Spawn prompt must include (ephemeral / spawn-unique):**

- Tillsyn `task_id` of the plan item the agent owns.
- Auth credentials: `session_id`, `session_secret`, `auth_context_id`, `agent_instance_id`, `lease_token`.
- Project working directory: absolute path to `main/` (`/Users/evanschultz/Documents/Code/hylla/tillsyn/main`). The agent `cd`s into this before any file or mage work.
- Move-state directive:
  - "Move your Tillsyn task to `in_progress` immediately when you start."
  - "When done: update metadata, move to terminal state."
  - "If you find issues that need fixing: leave `in_progress`, update metadata with findings, return to orchestrator."
- Short pointer: "Everything else is in your task description — follow it."

**Plan-item description must include (durable / authored):**

- Hylla artifact ref (`github.com/evanmschultz/tillsyn@main`). Also retrievable via Tillsyn project metadata (`metadata.hylla_artifact_ref`); planners copy it into each child description for convenience.
- Paths (post-Slice-1, `paths []string`) or affected files (pre-Slice-1, in prose).
- Packages (post-Slice-1, `packages []string`).
- Acceptance criteria.
- Mage targets for verification (discover via `mage -l`).
- Cross-references to sibling tasks, blockers, or upstream plan items.

**Before spawning any subagent:**

- Move the target item to `in_progress` if permission allows; otherwise the agent prompt's move-state directive instructs the subagent to do it itself.
- Verify the plan-item description carries everything the agent needs — do not patch missing description content by cramming it into the spawn prompt; fix the description instead so it's correct for future spawns.

**QA subagents specifically:** gated to `qa` role. Request a `qa`-role auth session and pass those credentials. QA agent moves its subtask to `in_progress` at start and `done` on pass. On findings that need fixes: leave `in_progress`, report findings, orchestrator spawns builder, re-runs QA.

## Task Lifecycle (Current HEAD)

Three terminal-reachable states today: `todo`, `in_progress`, `done`. A fourth state `failed` lands in Slice 1 of the cascade plan. Until then:

- **Success**: set `metadata.outcome: "success"`, update `completion_contract.completion_notes`, move to `done`.
- **Failure**: set `metadata.outcome: "failure"`, note details in `completion_notes`. Currently the task stays in `in_progress` with a failure-flavored outcome; Slice 1 adds the real `failed` transition.
- **Blocked**: set `metadata.outcome: "blocked"` + `metadata.blocked_reason`, report to orchestrator, stop.
- **Supersede** (post-Slice-1): human-only CLI `till task supersede <id> --reason "..."` unsticks `failed → complete`. Before Slice 1 this doesn't exist.

No parent can move to terminal-success if any child is in a failure/blocked state — enforcement becomes always-on in Slice 1.

## Paths and Packages (Slice-1 Target)

Today, builders and planners track affected code loosely in metadata. In Slice 1, `paths []string` and `packages []string` become first-class domain fields on every plan item, set by the planner, readable by builder + QA, and required for the file- and package-level blocking the cascade relies on. Until Slice 1 ships, note affected paths in `completion_notes` — the cascade plan (`CLAUDE_MINIONS_PLAN.md`, Section 5 + Section 17.1) is the contract.

## Auth and Leases

- One active auth session per scope level at a time.
- Orchestrator cleans up all child auth sessions and leases at end of phase/run.
- Auth auto-revoke on terminal state is a Slice-1 item; until then, the orchestrator manually revokes stale sessions.
- **Always report the auth session ID to the dev** when requesting or claiming auth. The dev needs visibility into active sessions.

## Coordination Surfaces

**Subagents:**
- `till.plan_item` — read task, update metadata, move state.
- `till.comment` — result comments on their own task.
- No attention_items, no handoffs, no @mentions, no downward/sideways signaling.

**Orchestrator (this session):**
- `till.plan_item` — create/update tasks, read state, move phases.
- `till.comment` — guidance before spawning subagents.
- `till.attention_item` — inbox for human approvals.
- `till.handoff` — structured next-action routing.
- `/loop` polling (60-120s cadence) for attention items during long-running work.

## Role Model

- **Orchestrator** — the human-launched CLI session. Plans, routes, delegates, cleans up. Never edits Go code. May edit markdown docs (this file, plan docs, agent files).
- **Builder** — subagent. The ONLY role that edits Go code. Reads task, implements, updates, dies.
- **QA Proof / QA Falsification** — subagents. Ephemeral. Read task, review, update with verdict, die.
- **Planning** — subagent. Decomposes a slice into tasks with paths/packages/acceptance criteria.
- **Research** — Claude's built-in `Explore` subagent.
- **Human** — approves auth, reviews results, makes design decisions.

## Recovery After Session Restart

1. `till.capture_state` — re-anchor project and scope context.
2. `till.attention_item(operation=list, all_scopes=true)` — inbox state.
3. Check `in_progress` tasks for staleness.
4. Revoke orphaned auth sessions/leases.
5. Resume from current task state.

## Claude Code Agents (Go Project)

Spawn via the `Agent` tool with `subagent_type`. There is no orchestration-agent row — the orchestrator is the parent session, not a subagent.

| Agent | Subagent Type | Purpose |
|---|---|---|
| **Builder** | `go-builder-agent` | Ephemeral builder — the only role that edits Go code |
| **Planning** | `go-planning-agent` | Hylla-first planning grounded in committed code reality |
| **QA Proof** | `go-qa-proof-agent` | Proof-completeness check — evidence supports the claim |
| **QA Falsification** | `go-qa-falsification-agent` | Falsification attempt — try to break the conclusion |

Inline (no subagent file):
- **research-agent** — Claude's built-in `Explore` subagent.

### QA Discipline

Two asymmetric passes, not duplicates:

- **QA Proof** (`go-qa-proof-agent`, `/qa-proof`) — evidence completeness, reasoning coherence, trace coverage.
- **QA Falsification** (`go-qa-falsification-agent`, `/qa-falsification`) — counterexamples, hidden deps, contract mismatches, YAGNI.

Run both for every build-task. They are asymmetric — proof checks whether the evidence supports the claim; falsification tries to construct a counterexample. Spawn them as parallel subagents so each gets a fresh context window.

## Skill and Slash Command Routing

| Command | When to Use |
|---|---|
| `/plan-from-hylla` | Hylla-grounded planning |
| `/qa-proof` | Proof-oriented QA |
| `/qa-falsification` | Falsification-oriented QA |
| `semi-formal-reasoning` | Explicit reasoning certificate for semantic/high-risk work |

## Semi-Formal Reasoning

For semantic, high-risk, or ambiguous work:

- **Premises** — what must be true
- **Evidence** — grounded in Hylla / `git diff` / Context7 / `go doc` / gopls
- **Trace or cases** — concrete paths through the code
- **Conclusion** — the claim
- **Unknowns** — what remains uncertain, routed into Tillsyn as a comment, handoff, or attention item

Short and inspectable.

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
- `.artifacts/` — generated local outputs
- `magefile.go` — canonical build/test automation

## Tech Stack

Go 1.26+ · Bubble Tea v2 · Bubbles v2 · Lip Gloss v2 · SQLite (`modernc.org/sqlite`, no CGO) · TOML (`github.com/pelletier/go-toml/v2`) · Laslig · Fang · `github.com/charmbracelet/log`

## Dev MCP Server

Every worktree needs a local dev MCP server pointing at its own built binary — test against the dev version, not the installed one.

```bash
mage build
claude mcp add --scope local tillsyn-dev -- /path/to/worktree/till serve-mcp
```

- **main**: `tillsyn-dev` → `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/till serve-mcp`

After every `mage build`, the dev binary is updated in place; MCP picks up changes on next invocation. Always test against `tillsyn-dev`, not the installed `till`. When retiring a branch, remove its dev MCP entry.

## Build Verification

Before any build-task is marked done:

1. All relevant mage targets pass (discover via `mage -l`).
2. **NEVER run `go test`, `go build`, `go run`, `go vet`, or any raw `go` toolchain command.** Always `mage <target>`. If a mage target has a bug, fix the target — don't bypass. No exceptions, orchestrator or subagent.
3. All template-generated QA subtasks completed.

Key targets: `mage run`, `mage build`, `mage test-pkg <pkg>`, `mage test-golden`, `mage test-golden-update`, `mage ci`. Run `mage ci` before push. Coverage below 70% is a hard failure.

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

## Git Commit Format

Conventional-commit: `type(scope): message`. All lowercase except proper nouns, acronyms (HTTP, TUI, WASM). Concise, describe what changed, not how.

Types: `feat`, `fix`, `refactor`, `chore`, `docs`, `test`, `ci`, `style`, `perf`

Examples:
- `feat(ingest): add per-file progress reporting`
- `fix(tui): correct viewport wrap on narrow terminals`

No co-authored-by trailers. No period at end. No capitalized first word after the colon unless proper noun/acronym.

## Safety

- Never delete files or directories without explicit dev approval.
- Never run commands outside the repo root `/Users/evanschultz/Documents/Code/hylla/tillsyn`.
- Never push to any remote without explicit request.
- Keep secrets out of committed config files.

## Bare-Root and Worktree Discipline

- The bare repo at `/Users/evanschultz/Documents/Code/hylla/tillsyn` (one level up) is the orchestration root — **not** a coding checkout.
- This directory (`main/`) is the primary work checkout. Real coding / building / testing / committing happens here.
- Always confirm `pwd` is this checkout before edits, tests, commits, or gopls work.
- **Dev launches orchestrators from here** — this is the canonical orchestrator working directory.
- This project uses a single visible checkout during cascade development. Additional worktrees and gopls-sync tooling are not needed — dispatched cascade agents `cd` into this directory directly.
