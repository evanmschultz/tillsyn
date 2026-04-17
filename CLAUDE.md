# Tillsyn — Project CLAUDE.md (main worktree)

This file lives in the **`main/` worktree** at `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/`. This is the primary work checkout — all real coding, building, testing, and committing happens here. **The dev launches orchestrators from this directory.** The bare-root `CLAUDE.md` (one directory up) carries the same rules body; only the preamble differs.

## Tillsyn Is the System of Record

All work is tracked in Tillsyn. No exceptions.

- No markdown files for work tracking, coordination, worklogs, or execution state.
- **Tillsyn = durable truth. Every piece of work gets a Tillsyn plan item before it starts.**
- **Use Tillsyn exclusively for work tracking.** Do NOT use Claude Code's built-in `TaskCreate` / `TaskUpdate` / `TaskList` / `TaskGet` / `TaskStop` / `TaskOutput` — they are in-session-only and evaporate on compaction/restart. If a turn needs finer procedural granularity, decompose into child Tillsyn plan items rather than bolting on a parallel in-session tracker.
- **When work starts on a plan item, move it to `in_progress` immediately.** No items left in `todo` while being worked on.
- **Read `main/WIKI.md` at session start and after every compaction.** The wiki is the living best-practice snapshot for this project and changes drop-by-drop. CLAUDE.md is auto-loaded; WIKI.md is NOT — you must read it deliberately. On the first turn after cold-start or compaction, Read `WIKI.md` before substantive orchestration.

### Discussion Mode (Chat-Primary Until TUI Ergonomics Land)

Cross-cutting decisions still park on a Tillsyn plan item (description = converged shape, comments = audit trail of direct quotes). But the actual dev ↔ orchestrator back-and-forth happens **in chat** until the TUI comment flow is ergonomic enough to drive decisions through. Surface the full substance in chat — open decisions, options, tradeoffs, blockers — not just status pings. After each round with concrete decisions, mirror the converged points back into the plan-item description and post a short audit-trail comment capturing dev direct quotes on corrections.

## Cascade Plan

The cascade (state-triggered autonomous agent dispatch) is designed in `PLAN.md` (lives in this directory — moved from bare-root in Drop 0 so it is git-tracked with the rest of the repo; renamed from `CLAUDE_MINIONS_PLAN.md` during the 2026-04-16 consolidation pass). That plan is the source of truth for cascade architecture, drop ordering, and hard prerequisites. This `CLAUDE.md` documents the **current pre-cascade workflow** the orchestrator uses today.

## Pre-Consolidation Source Archive

Fold-source MD files from the 2026-04-16 consolidation pass live in `OLD_MDS/`. They are kept on disk (not deleted) so the dev can verify no content was lost before the eventual deletion. Contents:

- `OLD_MDS/HEADLESS_DISCUSSIONS.md` — folded into `PLAN.md` §4.1 / §19.10 / §23 / §24 and drop-4.5 scope.
- `OLD_MDS/TOS_COMPLIANCE.md` — folded into `PLAN.md` §22 (verbatim quote appendix §22.4.1–22.4.6) and §22.5 engineering-constraint carryover.
- `OLD_MDS/TOS_DISCUSSIONS.md` — converged rows folded into `PLAN.md` §22; pending Qs threaded into refinement-drop bullets under §19.10.
- `OLD_MDS/MINIONS_RESEARCH_2026-04-13.md` — folded into `PLAN.md` §20 open questions + §21 source links + risk register.
- `OLD_MDS/TILLSYN_PURPOSE_AND_INTEGRATION_FRAMING_2026-04-11.md` — folded into `README.md` §Integration Framing For MCP Clients.
- `OLD_MDS/temp.md` — the pre-apply Consolidation Decisions Ledger that drove this pass (preserved as audit trail).

When looking for pre-consolidation context that seems missing from `PLAN.md` or `README.md`, check `OLD_MDS/` first before assuming drift.

## Cascade Tree Structure (Template Architecture)

This is the cascade's template architecture by plan-item `kind` — the **post-Drop-2 target state**. **Drop 3 encodes this tree as a template** and **Drop 4's dispatcher reads it** to bind agents, gates, and `child_rules`. Pre-cascade, the orchestrator approximates the same shape manually, but the `kind` values written into Tillsyn today are constrained by what Drop 2 Go can read — see "Pre-Drop-2 Creation Rule" below. The Kind Hierarchy / Agent Bindings sections describe the target shape, not the current runtime writes.

### Pre-Drop-2 Creation Rule (Current HEAD)

Until Drop 2 ships the Go kind-collapse + `Task → Drop` rename, **every new plan item under a project is created with `kind='task', scope='task'`**. Do NOT use the other registered kinds (`build-task`, `subtask`, `qa-check`, `plan-task`, `commit-and-reingest`, `a11y-check`, `visual-qa`, `design-review`, `phase`, `branch`, any `*-phase` variant, `decision`, `note`) even though they remain in `kind_catalog` — `main/scripts/drops-rewrite.sql` (dev-run after Drop 2 Go ships) rewrites every non-project kind to `drop`.

**Role on description prose, not metadata (pre-Drop-2):** note role in the description (`Role: builder`, `Role: qa-proof`, `Role: qa-falsification`, `Role: qa-a11y`, `Role: qa-visual`, `Role: design`, `Role: commit`, `Role: planner`). Drop 2 lands `metadata.role` as a first-class field; the SQL hydrates it from each item's pre-collapse `kind`.

**Same-scope nesting is allowed.** A `task` drop may nest under another `task` drop — `task` kind has no parent-scope restriction in `kind_catalog`. Same-scope nesting has live precedent (`subtask@subtask` under `subtask@subtask` in TILLSYN as of 2026-04-16). If the first nested `task@task` create is rejected by the MCP layer, fall back to `kind='subtask', scope='subtask'` for nested layers and flag the rejection.

### Kind Hierarchy

```
project                                 kind: project
└── drop (infinitely nestable)         kind: drop
      ├── plan-task                     kind: plan-task          ─→ agent: go-planning-agent          (opus)
      │   ├── plan-qa-proof             kind: qa-check           ─→ agent: go-qa-proof-agent          (opus)
      │   └── plan-qa-falsification     kind: qa-check           ─→ agent: go-qa-falsification-agent  (opus)
      │
      ├── drop (sub-drop)             kind: drop               (same shape, recurses infinitely)
      │
      └── task (build-task)             kind: task               ─→ agent: go-builder-agent           (sonnet)
            ├── qa-proof                kind: qa-check           ─→ agent: go-qa-proof-agent          (sonnet)
            └── qa-falsification        kind: qa-check           ─→ agent: go-qa-falsification-agent  (sonnet)
```

### Required Children (Auto-Create Rules)

- **Every `drop`** auto-creates three children on creation: `plan-task`, `plan-qa-proof`, `plan-qa-falsification`. Manual today; template `child_rules`-enforced in Drop 3.
- **Every `task`** (build-task) auto-creates two children on creation: `qa-proof`, `qa-falsification`.
- `plan-qa-proof` and `plan-qa-falsification` are `blocked_by: plan-task` — they fire in parallel after the plan-task completes.
- `qa-proof` and `qa-falsification` under a build-task are `blocked_by: task` — they fire in parallel after the build-task completes **and** its post-build gates pass (see below).
- Drops nest infinitely. A planner creates sub-drops when decomposition needs to continue, or build-tasks when the work is granular enough.

### Agent Bindings

Pre-cascade: orchestrator spawns these manually via the `Agent` tool using Tillsyn auth credentials in the prompt.
Post-Drop-3: the template binds kinds → agents; the dispatcher spawns them on `in_progress` transitions.

| Kind | Agent | Model | Role | Edits Code? |
|---|---|---|---|---|
| `plan-task` (drop-level) | `go-planning-agent` | opus | `planner` | No |
| `qa-check` under `plan-task` | `go-qa-proof-agent` / `go-qa-falsification-agent` | opus | `qa` | No |
| `task` (build-task) | `go-builder-agent` | sonnet | `builder` | **Yes** |
| `qa-check` under `task` | `go-qa-proof-agent` / `go-qa-falsification-agent` | sonnet | `qa` | No |
| commit-agent *(Drop-4+, post-build gate)* | `commit-message-agent` | haiku | `commit` | No |

### Post-Build Gates (Deterministic, Between Build-Task And Its QA)

After a build-task reports success, before its `qa-*` children become eligible, gates run programmatically. No LLM except the commit agent.

1. **`mage ci`** — on fail, the build-task moves to `failed`, gate output posted as a comment.
2. **Commit** — commit-agent (haiku) forms the message; system runs `git add` + `git commit`. Pre-cascade: orchestrator + dev do this manually (see Git Management (Pre-Cascade) below).
3. **Push** — `git push` when the template's `auto_push = true`. Pre-cascade: manual.
4. **Hylla reingest** — NOT per-task. Drop-end only, orchestrator-run, after `gh run watch --exit-status` is green. See "Cascade Ledger + Hylla Feedback" + "Drop End — Ledger Update Task" below. Agents never call `hylla_ingest`.

Only after all gates pass do the build-task's QA children fire.

### Blocker Semantics

- **Parent-child** — a parent cannot move to `complete` while any child is incomplete or `failed`. Always-on parent-blocks-on-failed-child arrives in Drop 1.
- **`blocked_by`** — the only sibling and cross-drop ordering primitive. Planner sets these at creation time; dispatcher adds runtime blockers when file/package locks conflict (Drop 4+).
- **File- and package-level blocking** — sibling build-tasks sharing a file in `paths` OR a package in `packages` MUST have an explicit `blocked_by` between them. Plan QA falsification attacks missing blockers. Package-level locking exists because a single Go package (e.g. `internal/domain` with ~25 files) shares one compile — editing different files in the same package still breaks the other agent's test run.

### State-Trigger Dispatch

Moving a plan item to `in_progress` is the dispatch trigger (Drop 4+). Pre-cascade, the orchestrator IS the dispatcher — it reads the kind, picks the binding above, moves the item to `in_progress`, and spawns the subagent via the `Agent` tool with Tillsyn auth credentials and Hylla artifact ref in the prompt.

## Tillsyn Project

The tillsyn project was **reset in Drop 0** — the prior messy project (`a0cfbf87-b470-45f9-aae0-4aa236b56ed9`, `default-go` template) was renamed to `TILLSYN-OLD` and a fresh, template-free project was created. Retiring `TILLSYN-OLD` via delete or archive is a Drop 10 refinement (project lifecycle ops bullet).

- **Project ID**: `a5e87c34-3456-4663-9f32-df1b46929e30`
- **Template**: none (fresh project, no template bound)
- **Slug**: `tillsyn`
- **Kind**: `go-project`

## Hylla Baseline

- **Artifact ref**: `github.com/evanmschultz/tillsyn@main` — Hylla resolves `@main` to the latest ingest automatically. Do not track snapshot numbers or commit hashes here.
- **Also stored on the Tillsyn project metadata** under `metadata.hylla_artifact_ref` so planners read it programmatically rather than copy-paste from this file.
- **Ledger**: `main/LEDGER.md` tracks per-drop cost, node count (total / code / tests / packages), orphan deltas, refactors, and drop descriptions. Populated by the orchestrator during the per-drop `DROP <N> END — LEDGER UPDATE` task after ingest completes.
- **Hylla feedback**: `main/HYLLA_FEEDBACK.md` aggregates subagent-reported Hylla ergonomics and search-quality issues. Subagents report misses in their closing comment; the orchestrator rolls them up at drop end before running the drop-end ingest.

### Code Understanding Rules

1. **All Go code**: use Hylla MCP (`hylla_search`, `hylla_node_full`, `hylla_search_keyword`, `hylla_refs_find`, `hylla_graph_nav`) as the primary source for committed-code understanding. If Hylla does not return the expected result on the first search, exhaust every Hylla search mode — vector (`hylla_search` with `search_types: ["vector"]`), keyword (`hylla_search_keyword`), graph-nav (`hylla_graph_nav`), refs (`hylla_refs_find`) — before falling back to `LSP`, `Read`, `Grep`, `Glob`. **Whenever a Hylla miss forces a fallback, the subagent must record the miss in its closing comment** under a `## Hylla Feedback` heading so the orchestrator can aggregate it into `main/HYLLA_FEEDBACK.md` at drop end.
2. **Changed since last ingest**: use `git diff` for files touched after the last Hylla ingest. Hylla is stale for those files until reingest.
3. **Non-Go code** (markdown, TOML, YAML, magefile, SQL, etc.): use `Read`, `Grep`, `Glob`, `Bash` directly. Hylla doesn't cover non-Go files.
4. **External semantics**: Context7 + `go doc` + `LSP` for library and language questions the repo can't answer itself.
5. **`LSP` tool** (gopls-backed, provided by the `gopls-lsp@claude-plugins-official` plugin): symbol search, references, diagnostics, rename safety, definitions for live / uncommitted code. Auto-targets the active checkout (`main/`). Subagents: use `LSP` rather than shelling out to `gopls` or scraping with `grep`/`rg`.

## Build-QA-Commit Discipline

**CRITICAL: Code is NEVER committed or pushed without QA completing first. Hylla ingest is drop-end only, not per-task.**

1. **Build** — builder subagent implements the increment.
2. **QA Proof** — `go-qa-proof-agent` verifies evidence completeness.
3. **QA Falsification** — `go-qa-falsification-agent` tries to break the conclusion.
4. **Fix** — if QA finds issues, spawn another builder to fix, then re-run QA.
5. **Commit** — only after both QA passes clear: `git add` the specific changed files, commit with conventional-commit format (pre-cascade: orchestrator + dev; post-Drop-4: commit-agent).
6. **Push** — `git push` so CI runs.
7. **CI green** — `gh run watch --exit-status` until CI lands green. If CI fails, fix before continuing — no ingest on a red commit.
8. **Update Tillsyn** — checklist + metadata + lifecycle state. If it's not in Tillsyn, it didn't happen.
9. **Move on to the next task.** Per-task Hylla reingest does NOT happen. Ingest happens once per drop, at drop end, inside the `DROP <N> END — LEDGER UPDATE` task — see "Cascade Ledger + Hylla Feedback" and "Drop End — Ledger Update Task" below.

No batched commits. No deferred pushes. No skipped QA. No skipped CI watch. No claiming done in chat without Tillsyn reflecting it.

## Cascade Ledger + Hylla Feedback

Per-drop artifact MDs live in `main/`. **All MD writes route through `STEWARD`** (the persistent continuation orchestrator — see `STEWARD_ORCH_PROMPT.md`). Numbered-drop orchestrators (`DROP_N_ORCH`) never edit MDs — they file per-drop content into `description` fields of **level_2 findings drops** under STEWARD's persistent level_1 parents, and STEWARD writes the MDs on `main` post-merge.

STEWARD-owned per-drop MDs:

- **`LEDGER.md`** — per-drop snapshot of project state, cost, and code-quality deltas. Fields per drop: closed date, drop plan-item ID, ingest snapshot, ingest cost + cost-to-date, node counts (total / code / tests / packages), orphan delta, refactors, description, commit SHAs, notable plan-item IDs, unknowns forwarded. Fed by `DROP_N_LEDGER_ENTRY.description`.
- **`HYLLA_FEEDBACK.md`** — running log of Hylla ergonomics and search-quality feedback from subagents. Fed by `DROP_N_HYLLA_FINDINGS.description`.
- **`WIKI_CHANGELOG.md`** — per-drop wiki deltas. Fed by `DROP_N_WIKI_CHANGELOG_ENTRY.description`.
- **`REFINEMENTS.md`** — deferred refinements raised per drop. Fed by `DROP_N_REFINEMENTS_RAISED.description`.
- **`HYLLA_REFINEMENTS.md`** — Hylla-scoped deferred refinements. Fed by `DROP_N_HYLLA_REFINEMENTS_RAISED.description`.
- **`WIKI.md`** — living best-practice snapshot; STEWARD curates between drops.

**Flow:** during drop N, drop-orch populates the five level_2 findings-drop descriptions incrementally. At drop end, drop-orch runs ingest, finalizes the descriptions, and closes `DROP <N> END — LEDGER UPDATE` before merge. Post-merge on `main`, STEWARD reads the level_2 descriptions, discusses with dev, writes the corresponding MDs, commits docs-only on `main`, and closes the level_2 drops. The six persistent level_1 parents (`DISCUSSIONS`, `HYLLA_FINDINGS`, `LEDGER`, `WIKI_CHANGELOG`, `REFINEMENTS`, `HYLLA_REFINEMENTS`) never close. See `STEWARD_ORCH_PROMPT.md` §10 + `feedback_steward_owns_md_writes.md` memory.

**Subagent responsibility:** in every closing comment, always include a `## Hylla Feedback` section. If you had no Hylla misses, write `None — Hylla answered everything needed.`. If you did, record each miss with:

- **Query**: tool name + key inputs.
- **Missed because**: your hypothesis (wrong search mode, schema gap, missing summary, stale ingest, etc.).
- **Worked via**: the fallback tool + inputs that found the thing.
- **Suggestion**: one-liner for what Hylla could do better.

Explicit "no miss" is still useful signal. Ergonomic-only gripes (awkward parameters, confusing response shapes, weird IDs) also go here.

## Drop End — Ledger Update Task

Every drop gets a final drop-orch-owned task named `DROP <N> END — LEDGER UPDATE`. `blocked_by` every other task in the drop. Runs once all siblings are `done`. Closed by drop-orch **before the drop branch merges to `main`**. Drop-orch owns ingest + level_2 findings-drop description finalization; STEWARD owns the MD writes post-merge.

**Drop-orch steps (pre-merge, on the drop branch):**

1. Move the task to `in_progress`.
2. Confirm all sibling tasks in the drop are `done`. Confirm `git status --porcelain` clean.
3. Confirm every commit from this drop has landed on the remote drop branch.
4. Run `gh run watch --exit-status` on the latest CI run. Do NOT proceed unless CI is green.
5. Call `hylla_ingest` on the remote ref `github.com/evanmschultz/tillsyn@main`. **ALWAYS full enrichment. NEVER `structural_only`. NEVER from a local working copy — always from remote, after push + CI green.**
6. Poll `hylla_run_get` via `/loop 120` while ingest progresses. When the run reports "nearly done" (enrichment stage entered), kill the loop and `ScheduleWakeup` once for the estimated remaining time.
7. When ingest completes, read `hylla_run_get` final result. Extract: ingest snapshot, cost (this run + lineage-to-date), node counts (total / code / tests / packages), orphan delta.
8. **Finalize each of the five level_2 findings-drop descriptions** drop-orch created at drop spin-up — `DROP_N_HYLLA_FINDINGS`, `DROP_N_LEDGER_ENTRY`, `DROP_N_WIKI_CHANGELOG_ENTRY`, `DROP_N_REFINEMENTS_RAISED`, `DROP_N_HYLLA_REFINEMENTS_RAISED` — with drop-in-ready content STEWARD will splice into the MDs post-merge. The `DROP_N_LEDGER_ENTRY.description` must carry a fully-formatted `## Drop <N> — <Title>` block (closed date, plan-item ID, ingest snapshot, cost, node counts, orphan delta, refactors, description, commit SHAs, notable IDs, unknowns forwarded).
9. Post a `till.handoff` to `@STEWARD` with `next_action_type: post-merge-md-write` naming the five level_2 drops.
10. Close `DROP <N> END — LEDGER UPDATE` with `metadata.outcome: "success"` and the five level_2 drop IDs in `completion_notes`.
11. **Do NOT write any MD file.** STEWARD writes all per-drop MDs on `main` post-merge.
12. Signal the dev the drop branch is ready to merge.

**Post-merge (STEWARD, on `main`):**

STEWARD reads each level_2 findings-drop description, discusses with dev, writes the corresponding MD on `main`, commits docs-only with single-line conventional-commits, pushes, and closes the level_2 drops. STEWARD then works the `DROP_N_REFINEMENTS_GATE_BEFORE_DROP_N+1` item inside drop N's tree — discussing next-drop refinements + STEWARD-self refinement with the dev, applying agreed changes, closing the gate. Closing the refinements-gate unblocks drop N's level_1 closure. Full sequence in `STEWARD_ORCH_PROMPT.md` §10.

**Hylla ingest invariants (repeat for emphasis):**

- Always `enrichment_mode=full_enrichment`.
- Always source from the GitHub remote.
- Never before `git push` + `gh run watch --exit-status` green.
- Only the drop-orch calls `hylla_ingest`. Subagents never do. STEWARD never does.

## Git Management (Pre-Cascade)

Until the cascade dispatcher takes over commits (`PLAN.md` Drop 11), **orchestrator + dev manage git manually**. The orchestrator does not commit from its own session — it asks the dev, or spawns a builder subagent when code changes are needed. Clean git state (for the files a plan item declares) is a precondition for creating a plan item; the orchestrator checks `git status --porcelain <paths>` before creation and asks the dev to clean up if dirty.

## Orchestrator-as-Hub Architecture

The parent Claude Code session launched by the dev from this directory is always **the orchestrator**. There is no `.claude/agents/orchestration-agent.md` file — the orchestrator is defined by the invocation context, not by a markdown spec. Every other role (builder, qa, planner, closeout, research) is a subagent spawned via the `Agent` tool.

**CRITICAL: The orchestrator NEVER writes Go code.** The parent session must not use `Edit`, `Write`, or any other tool to modify `.go` source or test files. Every code change — every single one — goes through a builder subagent via the `Agent` tool. Orchestrator reads code for planning and research only.

**Markdown documentation edits route through `STEWARD`.** STEWARD (the persistent continuation orchestrator — `STEWARD_ORCH_PROMPT.md`) is the only orchestrator that edits MD files in this repo. Numbered-drop orchestrators (`DROP_N_ORCH`) never touch MDs — they file per-drop artifact content into level_2 findings-drop descriptions under STEWARD's persistent level_1 parents. STEWARD writes the MDs on `main` post-merge. See "Cascade Ledger + Hylla Feedback" and "Drop End — Ledger Update Task" above.

### How It Works

1. Orchestrator plans, routes, delegates, and cleans up. Reads code + Hylla for research. Creates Tillsyn plan items. Spawns subagents. Coordinates results.
2. Subagents are ephemeral — they spawn, read their task, do work, update the task, die.
3. Task state is the signal. On terminal state, the subagent sets `metadata.outcome` and moves to `done` or `failed` (once Drop 1 lands, `failed` will be a real terminal state; until then, failures are represented in metadata).
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
- Paths (post-Drop-1, `paths []string`) or affected files (pre-Drop-1, in prose).
- Packages (post-Drop-1, `packages []string`).
- Acceptance criteria.
- Mage targets for verification (discover via `mage -l`).
- Cross-references to sibling tasks, blockers, or upstream plan items.

**Before spawning any subagent:**

- Move the target item to `in_progress` if permission allows; otherwise the agent prompt's move-state directive instructs the subagent to do it itself.
- Verify the plan-item description carries everything the agent needs — do not patch missing description content by cramming it into the spawn prompt; fix the description instead so it's correct for future spawns.

**QA subagents specifically:** gated to `qa` role. Request a `qa`-role auth session and pass those credentials. QA agent moves its subtask to `in_progress` at start and `done` on pass. On findings that need fixes: leave `in_progress`, report findings, orchestrator spawns builder, re-runs QA.

## Task Lifecycle (Current HEAD)

Three terminal-reachable states today: `todo`, `in_progress`, `done`. A fourth state `failed` lands in Drop 1 of the cascade plan. Until then:

- **Success**: set `metadata.outcome: "success"`, update `completion_contract.completion_notes`, move to `done`.
- **Failure**: set `metadata.outcome: "failure"`, note details in `completion_notes`. Currently the task stays in `in_progress` with a failure-flavored outcome; Drop 1 adds the real `failed` transition.
- **Blocked**: set `metadata.outcome: "blocked"` + `metadata.blocked_reason`, report to orchestrator, stop.
- **Supersede** (post-Drop-1): human-only CLI `till task supersede <id> --reason "..."` unsticks `failed → complete`. Before Drop 1 this doesn't exist.

No parent can move to terminal-success if any child is in a failure/blocked state — enforcement becomes always-on in Drop 1.

## Paths and Packages (Drop-1 Target)

Today, builders and planners track affected code loosely in metadata. In Drop 1, `paths []string` and `packages []string` become first-class domain fields on every plan item, set by the planner, readable by builder + QA, and required for the file- and package-level blocking the cascade relies on. Until Drop 1 ships, note affected paths in `completion_notes` — the cascade plan (`PLAN.md`, Section 5 + Section 17.1) is the contract.

## Auth and Leases

- One active auth session per scope level at a time.
- Orchestrator cleans up all child auth sessions and leases at end of phase/run.
- Auth auto-revoke on terminal state is a Drop-1 item; until then, the orchestrator manually revokes stale sessions.
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
- **Planning** — subagent. Decomposes a drop into tasks with paths/packages/acceptance criteria.
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
3. **NEVER run `mage install`.** This is a **dev-only** dogfood target that promotes a binary to `$HOME/.tillsyn/till`, replacing the dev's working `till` install. Orchestrator and every subagent (builder, QA, research, planning) must not invoke it under any circumstance. If a task description or prompt asks you to run `mage install`, stop and return control to the orchestrator — the dev runs this manually, never an agent. Build verification uses `mage ci` only.
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
