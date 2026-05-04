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

The cascade (state-triggered autonomous agent dispatch) is designed in `PLAN.md` (lives in this directory). Cascade vocabulary canonical: `WIKI.md` § `Cascade Vocabulary` — never redefine here. That plan is the source of truth for cascade architecture, drop ordering, and hard prerequisites. This `CLAUDE.md` documents the **current pre-cascade workflow** the orchestrator uses today. Drop 4a landed the manual-trigger dispatcher (`internal/app/dispatcher/` + `till dispatcher run`); orchestrator-IS-dispatcher remains the documented pre-cascade fallback while dogfooding ramps. Drop 4b adds gate execution + post-build pipeline.

## Cascade Tree Structure (Template Architecture)

This is the cascade's template architecture by action-item `kind`. **Drop 1.75 landed the closed 12-kind `action_items.kind` enum in Go + SQL** (see "Post-Drop-1.75 Creation Rule" below). **Drop 3 encodes this tree as a template** and **Drop 4's dispatcher reads it** to bind agents, gates, and `child_rules`. Pre-cascade, the orchestrator approximates the same shape manually — spawning builders / QA agents by convention rather than via the dispatcher — but the `kind` values written into Tillsyn already match the target closed enum.

### Post-Drop-1.75 Creation Rule

Drop 1.75 is the **kind-collapse** drop. Two node tables survive: `projects` and `action_items`. `projects` has no `kind` column post-collapse. `action_items.kind` is a closed 12-value enum, chosen by the creator at creation time — there is no inferred default and no `actionItem` fallback kind. Old kinds (`task`, `actionItem`, `build-actionItem`, `subtask`, `qa-check`, `plan-actionItem`, `commit-and-reingest`, `a11y-check`, `visual-qa`, `design-review`, `phase`, `branch`, any `*-phase` variant, `decision`, `note`) are rewritten by `main/scripts/drops-rewrite.sql` into the new vocabulary; `action_items.scope` is mirrored from `kind` (scope removal lives in a future refinement drop).

`action_items.kind` closed enum (12 values):

| Kind                     | Purpose                                                                 |
| ------------------------ | ----------------------------------------------------------------------- |
| `plan`                   | Planning-dominant — planner agent decomposes work into children.        |
| `research`               | Read-only investigation — research agent compiles findings, posts, dies. |
| `build`                  | Code-changing leaf — builder agent implements, tests, commits.          |
| `plan-qa-proof`          | Proof-completeness QA pass on a `plan` parent.                          |
| `plan-qa-falsification`  | Falsification QA pass on a `plan` parent.                               |
| `build-qa-proof`         | Proof-completeness QA pass on a `build` parent.                         |
| `build-qa-falsification` | Falsification QA pass on a `build` parent.                              |
| `closeout`               | Drop-end coordination — aggregates ledger / refinements / findings.     |
| `commit`                 | Commit action — template-triggered under `plan` at level ≥ 2 (see Commit Cadence below). |
| `refinement`             | Perpetual / long-lived tracking umbrella — drop-end entries roll up here. |
| `discussion`             | Cross-cutting decision park — description = converged shape, comments = audit trail. |
| `human-verify`           | Dev sign-off hold point — attention items + checklist children, no plan/QA. |

**Customization (future drop — NOT Drop 1.75):** projects and orchestrators author custom kinds via template that attach as sub-action-items of specific generic kinds (e.g., a custom `ledger-update` under `closeout`, a custom `ledger-aggregation` under `refinement`). Drop 1.75 ships only the 12 generics plus the template hook. Drop 2 lands `metadata.role` as a first-class field; until then, role goes in description prose (`Role: builder`, `Role: qa-proof`, `Role: qa-falsification`, `Role: qa-a11y`, `Role: qa-visual`, `Role: design`, `Role: commit`, `Role: planner`, `Role: research`).

**Project-level kind restriction (future drop).** Project templates will carry an allowed-kinds enum + a `disallow_generics` bool to restrict which kinds a project accepts — lands alongside template customization. Drop 1.75 treats all 12 kinds as allowed on every project.

### Three Orthogonal Axes — `kind` × `metadata.role` × `metadata.structural_type`

Every non-project node is classified along three independent axes, set explicitly at create time. None of them are inferred from the others. Templates `child_rules`, gate rules, and agent bindings dispatch on combinations of all three.

- **`kind` (what work)** — the closed 12-value enum above (post-Drop-1.75). Names the kind of work the agent does (`build` / `plan` / `build-qa-proof` / …).
- **`metadata.role` (who does it)** — closed enum (post-Drop-2): `builder`, `qa-proof`, `qa-falsification`, `qa-a11y`, `qa-visual`, `design`, `commit`, `planner`, `research`. The dual-axis design earns its keep on QA kinds where parent context disambiguates (`build-qa-proof` vs `plan-qa-proof` both carry `role=qa-proof`); non-QA role values exist for agent-binding-lookup symmetry. Pre-Drop-2 the role lives in description prose (`Role: builder`, `Role: qa-proof`, …); post-Drop-2 it lands on `metadata.role`.
- **`metadata.structural_type` (where it sits in the cascade flow)** — closed 4-value enum landing in Drop 3: `drop | segment | confluence | droplet`. Names the node's cascade shape independent of role / kind. Atomicity rules: `droplet` MUST have zero children; `confluence` MUST have non-empty `blocked_by`; `segment` may recurse; `drop` is the level_1 cascade step.

**Cascade vocabulary canonical: `WIKI.md` § `Cascade Vocabulary` — the worked-combinations table, atomicity rules, and orthogonality with `metadata.role` live there. Do not redefine the structural_type vocabulary in this file or any other doc.** Pre-Drop-3, `metadata.structural_type` is not yet validated at the create/update boundary; the orchestrator chooses values consistent with the WIKI definition and Drop 3 lands the validation. Adopters mirror the same canonical-pointer rule in their own project `CLAUDE.md` (see `workflow/example/CLAUDE.md` line 27).

### Kind Hierarchy

Two node types, one enum. `project` is a table, not a kind. Everything below is `action_items.kind`:

```
project                                                    (table: projects)
└── plan (infinitely nestable)                             kind: plan                    ─→ agent: go-planning-agent          (opus)
      ├── plan-qa-proof                                    kind: plan-qa-proof           ─→ agent: go-qa-proof-agent          (opus)
      ├── plan-qa-falsification                            kind: plan-qa-falsification   ─→ agent: go-qa-falsification-agent  (opus)
      │
      ├── research                                         kind: research                ─→ agent: go-research-agent          (opus)
      │
      ├── build (leaf)                                     kind: build                   ─→ agent: go-builder-agent           (sonnet)
      │     ├── build-qa-proof                             kind: build-qa-proof          ─→ agent: go-qa-proof-agent          (sonnet)
      │     └── build-qa-falsification                     kind: build-qa-falsification  ─→ agent: go-qa-falsification-agent  (sonnet)
      │
      ├── plan (sub-plan — infinite nesting)               kind: plan                    (same shape, recurses)
      │
      ├── closeout                                         kind: closeout                (drop-end aggregation)
      ├── commit                                           kind: commit                  (template-triggered, post-Drop-4)
      ├── discussion                                       kind: discussion              (converged shape + audit trail)
      ├── refinement                                       kind: refinement              (perpetual rollup)
      └── human-verify                                     kind: human-verify            (dev sign-off hold point)
```

### Required Children (Auto-Create Rules)

- **Every `plan`** auto-creates two children on creation: `plan-qa-proof`, `plan-qa-falsification`. Manual today; template `child_rules`-enforced in Drop 3.
- **Every `build`** auto-creates two children on creation: `build-qa-proof`, `build-qa-falsification`.
- `plan-qa-proof` and `plan-qa-falsification` are `blocked_by: plan` — they fire in parallel after the `plan` completes.
- `build-qa-proof` and `build-qa-falsification` are `blocked_by: build` — they fire in parallel after the `build` completes **and** its post-build gates pass (see below).
- `plan` nests infinitely. A planner creates sub-`plan`s when decomposition continues, or `build`s when the work is granular enough.
- `research`, `discussion`, `closeout`, `refinement`, `human-verify` do NOT auto-create QA children — they are standalone action items. Research gets reviewed via comment thread; discussion converges in description; human-verify gates on attention-item sign-off.

### Commit Cadence (Drop 2 Discussion Seed)

`commit` is a real kind but current commit-every-`build` rate is too high. Proposed default heuristic for Drop 2 discussion:

- **Auto-generate a `commit` child under any `plan` at level ≥ 2 whose subtree contains at least one `build` child in `complete` state.** Generation fires at plan close-time (not creation), so no-build plans (pure research / discussion / refinement) don't accrue orphan commit items.
- **Template override** — templates can set `auto_commit = false` on a `plan` kind for local-only outputs (SQL migration scripts that never touch tracked code, scratch/throwaway work).
- **Edge cases to resolve in the Drop 2 discussion item**: multi-PR drops (one commit per plan vs one per drop), cascade-owned commit vs dev-owned commit, partial-success subtrees (some `build` children complete, some failed).

### Agent Bindings

Pre-cascade: orchestrator spawns these manually via the `Agent` tool using Tillsyn auth credentials in the prompt.
Post-Drop-3: the template binds kinds → agents; the dispatcher spawns them on `in_progress` transitions.
Post-Drop-4a: Wave 2 delivered the dispatcher loop with manual-trigger CLI (`till dispatcher run --action-item <id>`); automatic dispatch on `in_progress` transitions lands in Drop 4b.

| Kind                      | Agent                         | Model  | Role               | Edits Code? |
| ------------------------- | ----------------------------- | ------ | ------------------ | ----------- |
| `plan`                    | `go-planning-agent`           | opus   | `planner`          | No          |
| `plan-qa-proof`           | `go-qa-proof-agent`           | opus   | `qa-proof`         | No          |
| `plan-qa-falsification`   | `go-qa-falsification-agent`   | opus   | `qa-falsification` | No          |
| `research`                | `go-research-agent`           | opus   | `research`         | No          |
| `build`                   | `go-builder-agent`            | sonnet | `builder`          | **Yes**     |
| `build-qa-proof`          | `go-qa-proof-agent`           | sonnet | `qa-proof`         | No          |
| `build-qa-falsification`  | `go-qa-falsification-agent`   | sonnet | `qa-falsification` | No          |
| `commit` _(Drop-4+)_      | `commit-message-agent`        | haiku  | `commit`           | No          |
| `closeout` / `refinement` / `discussion` / `human-verify` | orchestrator-managed | —   | orchestrator       | No          |

### Post-Build Gates (Deterministic, Between `build` And Its QA Children)

After a `build` action item reports success, before its `build-qa-proof` / `build-qa-falsification` children become eligible, gates run programmatically. No LLM except the commit agent.

1. **`mage ci`** — on fail, the `build` moves to `failed`, gate output posted as a comment. Wave 0 of Drop 4a wired `mage ci` into `.githooks/pre-push`, so a clean push is itself the smoke check (dev runs `mage install-hooks` once per fresh clone).
2. **Commit** — commit-agent (haiku) forms the message; system runs `git add` + `git commit`. Pre-cascade: orchestrator + dev do this manually (see Git Management (Pre-Cascade) below). Commit cadence follows the rule in "Commit Cadence" above — not every `build` generates a `commit` child.
3. **Push** — `git push` when the template's `auto_push = true`. Pre-cascade: manual.
4. **Hylla reingest** — NOT per-`build`. Drop-end only, orchestrator-run, after `gh run watch --exit-status` is green. See "Cascade Ledger + Hylla Feedback" + "Drop Closeout" below. Agents never call `hylla_ingest`.

Only after all gates pass do the `build`'s QA children fire.

### Blocker Semantics

- **Parent-child** — a parent cannot move to `complete` while any child is incomplete or `failed`. Always-on invariant — Wave 1 of Drop 4a removed the `RequireChildrenComplete` policy bit; the rule is unconditional.
- **`blocked_by`** — the only sibling and cross-drop ordering primitive. Planner sets these at creation time; Wave 2 of Drop 4a delivered the dispatcher's lock manager (file + package), and the conflict detector inserts runtime `blocked_by` on `in_progress` promotion when sibling locks conflict.
- **File- and package-level blocking** — sibling `build` action items sharing a file in `paths` OR a package in `packages` MUST have an explicit `blocked_by` between them. Plan QA falsification attacks missing blockers. Package-level locking exists because a single Go package (e.g. `internal/domain` with ~25 files) shares one compile — editing different files in the same package still breaks the other agent's test run.

### State-Trigger Dispatch

Moving an action item to `in_progress` is the dispatch trigger (Drop 4+). Pre-cascade, the orchestrator IS the dispatcher — it reads the kind, picks the binding above, moves the item to `in_progress`, and spawns the subagent via the `Agent` tool with Tillsyn auth credentials and Hylla artifact ref in the prompt. Drop 4a delivered the manual-trigger dispatcher: `till dispatcher run --action-item <id>` (`cmd/till`) reads the same template `agent_bindings`, acquires file/package locks via the lock manager (`internal/app/dispatcher/locks_file.go` + `locks_package.go`), spawns the subagent via `claude --agent <name>`, and provisions auth via Wave-3's orch-self-approval flow. Per-drop work currently dogfoods both paths until Drop 4b's gate runner ships.

## Tillsyn Project

The tillsyn project was **reset in Drop 0** — the prior messy project (`a0cfbf87-b470-45f9-aae0-4aa236b56ed9`, `default-go` template) was renamed to `TILLSYN-OLD` and a fresh, template-free project was created. Retiring `TILLSYN-OLD` via delete or archive is a Drop 10 refinement (project lifecycle ops bullet).

- **Project ID**: `a5e87c34-3456-4663-9f32-df1b46929e30`
- **Template**: none (fresh project, no template bound)
- **Slug**: `tillsyn`

(Projects have no `kind` column post-Drop-1.75 — `action_items.kind` is the only kind enum. Language/stack info lives in `metadata` on the project.)

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

**CRITICAL: No droplet is `complete` without per-droplet QA passing. Push + `gh run watch` + Hylla reingest are drop-end only — full sequence in `workflow/example/drops/WORKFLOW.md` Phases 4–7.**

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

Drop-orch closes the drop per `workflow/example/drops/WORKFLOW.md` Phase 7 — aggregate Hylla feedback + refinements, write `CLOSEOUT.md`, flip drop state to `complete`, merge PR.

**Hylla ingest invariants (inviolable):**

- Always `enrichment_mode=full_enrichment`. Never `structural_only`.
- Always source from the GitHub remote (`github.com/evanmschultz/tillsyn@main`). Never from a local working copy.
- Never before `git push` + `gh run watch --exit-status` green.
- Only the drop-orch calls `hylla_ingest`. Subagents never do. STEWARD never does.

## Git Management (Pre-Cascade)

Until the cascade dispatcher takes over commits (`PLAN.md` Drop 11), **orchestrator + dev manage git manually**. The orchestrator does not commit from its own session — it asks the dev, or spawns a builder subagent when code changes are needed. Clean git state (for the files an action item declares) is a precondition for creating an action item; the orchestrator checks `git status --porcelain <paths>` before creation and asks the dev to clean up if dirty.

### Post-Merge Branch Cleanup (Drop Closeout)

After a drop PR merges, the closing orchestrator MUST run the cleanup sequence below **in this order**. Skipping steps leaves stale worktrees or branch refs that block future drops (the Drop 1.75 close-out hit this: a leftover `drop/1.5` worktree still had `main` checked out with uncommitted files, which blocked `gh pr merge`'s local sync + blocked the next drop's worktree from checking out `main`).

1. **Merge with history preserved.** `gh pr merge <N> --merge --delete-branch` — `--merge` creates a merge commit that preserves every commit in the drop branch (NOT `--squash`, NOT `--rebase`). `--delete-branch` removes the remote ref when the local sync step succeeds.
2. **If `gh pr merge`'s local sync step fails** (usually because another worktree has `main` checked out with uncommitted work), the server-side merge still succeeded — verify with `gh pr view <N> --json state,mergeCommit`. Then delete the remote branch explicitly: `git push origin --delete <branch>`.
3. **`cd` into the `main/` worktree — NEVER run cleanup from inside the worktree you're about to remove.** Removing the worktree you're standing in pulls the rug out from the current shell.
4. **Fast-forward main:** `git fetch origin && git pull --ff-only` in `main/`. Confirm the merge commit is at HEAD.
5. **Remove the drop worktree:** `git worktree remove /path/to/drop/N` from `main/` or the bare root. If it refuses because of staged/unstaged changes, INVESTIGATE before `--force`-ing — those changes may be real work someone forgot to commit.
6. **Delete the local branch ref:** `git branch -D drop/N`.
7. **Verify clean:** `git worktree list` should show only bare + `main` (+ any live concurrent drops). `git branch -a` should show no stale local drop branches.

**Guardrail against "uncommitted work in a stale worktree" recurrence:** every drop orchestrator MUST commit or explicitly stash all working-dir changes before marking its drop closed. A stale worktree holding `main` with staged files is an anti-pattern — if work isn't ready to commit, it should sit on its own named branch, not on `main` in a drop worktree. If a close-out orchestrator finds pending changes during step 5 above, they must route them (commit to a preservation branch, hand back to dev, or explicitly destroy with dev sign-off) — never silently force-remove.

## Orchestrator-as-Hub Architecture

The parent Claude Code session launched by the dev from this directory is always **the orchestrator**. There is no `.claude/agents/orchestration-agent.md` file — the orchestrator is defined by the invocation context, not by a markdown spec. Every other role (builder, qa, planner, closeout, research) is a subagent spawned via the `Agent` tool.

**CRITICAL: The orchestrator NEVER writes Go code.** The parent session must not use `Edit`, `Write`, or any other tool to modify `.go` source or test files. Every code change — every single one — goes through a builder subagent via the `Agent` tool. Orchestrator reads code for planning and research only.

**Markdown doc ownership is split between drop-orch (drop branch) and STEWARD (`main` post-merge).** Drop-orchs (`DROP_N_ORCH`) own per-drop artifact content in `main/workflow/drop_N/` and any architecture-MD edits (`CLAUDE.md`, `PLAN.md`, `AGENT_CASCADE_DESIGN.md`, `STEWARD_ORCH_PROMPT.md`, `workflow/README.md`, `workflow/example/**`) when the drop's scope touches process — all on the drop branch, flowing to `main` via PR merge. STEWARD (persistent continuation orchestrator — `STEWARD_ORCH_PROMPT.md`) runs post-merge on `main`, reads `main/workflow/drop_N/` content, collates it into the six top-level MDs (`LEDGER.md`, `REFINEMENTS.md`, `HYLLA_FEEDBACK.md`, `WIKI_CHANGELOG.md`, `HYLLA_REFINEMENTS.md`, plus `WIKI.md` curation), then `git worktree remove drop/N` after drop-orch has deleted the remote + local branch refs.

### How It Works

1. Orchestrator plans, routes, delegates, and cleans up. Reads code + Hylla for research. Creates Tillsyn action items. Spawns subagents. Coordinates results.
2. Subagents are ephemeral — they spawn, read their action item, do work, update the action item, die.
3. Action-item state is the signal. On terminal state, the subagent sets `metadata.outcome` and moves to `complete` or `failed` (once Drop 1 lands, `failed` will be a real terminal state; until then, failures are represented in metadata).
4. Subagents do not poll or watch anything. Read the action item at spawn, execute, update, return.
5. Only the orchestrator uses attention items (human approval + inter-orchestrator coordination).

### Agent State Management

Every subagent manages its own Tillsyn action-item state. The orchestrator can't move role-gated items.

**Spawn prompt vs. action-item description split:** the spawn prompt carries ephemeral fields (action_item_id, auth credentials, working directory, move-state directive); the action-item description carries durable content (Hylla artifact ref, paths, packages, acceptance criteria, mage targets, cross-references). Rule: if a field changes every spawn, put it in the prompt; if it's stable across time and authors, put it in the description.

Full contract — exact spawn-prompt fields, exact description fields, spawn-gate checks — lives in each agent file at `~/.claude/agents/*.md` under "Required Prompt Fields" and "Spawn Prompt vs Action-Item Description Split." Don't duplicate it here.

## Action-Item Lifecycle (Current HEAD)

Four terminal-reachable states: `todo`, `in_progress`, `complete`, `failed` (Drop 1 landed `failed` as a real terminal state):

- **Success**: set `metadata.outcome: "success"`, update `completion_contract.completion_notes`, move to `complete`.
- **Failure**: set `metadata.outcome: "failure"`, note details in `completion_notes`, transition to `failed` (real terminal state since Drop 1).
- **Blocked**: set `metadata.outcome: "blocked"` + `metadata.blocked_reason`, report to orchestrator, stop.
- **Supersede** (post-Drop-1): human-only CLI `till action_item supersede <id> --reason "..."` unsticks `failed → complete`.

No parent can move to terminal-success if any child is in `failed` or `blocked` state — always-on invariant (Wave 1 of Drop 4a removed the `RequireChildrenComplete` policy bit; the rule is unconditional).

## Paths and Packages

Wave 1 of Drop 4a landed `paths []string`, `packages []string`, `files []string`, `start_commit string`, and `end_commit string` as first-class fields on every `ActionItem` (`internal/domain/action_item.go`). Planners set `paths` + `packages` at creation; dispatcher's lock manager (Wave 2) reads `packages` for package-level locks and `paths` for file-level locks. Builders restrict edits to declared `paths`; reference-only material lives in `files`. `start_commit` / `end_commit` are opaque caller-populated strings (orchestrator pre-cascade; dispatcher post-Wave-2). Per-package compile collisions are blocked at `in_progress` promotion via runtime `blocked_by` insertion when a sibling holds the same package lock. Cross-reference: `WIKI.md § "Atomic Drop Granularity"` for the planner-side rule of thumb.

## Auth and Leases

- One active auth session per scope level at a time.
- Orchestrator cleans up all child auth sessions and leases at end of phase/run.
- Auth auto-revoke on terminal state lands in Drop 4b. Pre-Drop-4b, orchestrators (and STEWARD post-merge) manually revoke stale sessions via `till.auth_request operation=revoke`.
- Orchestrators approve their own non-orch subagent auth requests scoped within their lease subtree (Wave 3 of Drop 4a). Cross-orch and orch-spawning-orch approvals still route through the dev TUI. Project-level `OrchSelfApprovalEnabled = *false` toggle is the total backstop (reverts ALL approves under that project to dev-TUI).
- **Always report the auth session ID to the dev** when requesting or claiming auth. The dev needs visibility into active sessions.

## Coordination Surfaces

**Subagents:**

- `till.action_item` — read the action item, update metadata, move state.
- `till.comment` — result comments on their own action item.
- No attention_items, no handoffs, no @mentions, no downward/sideways signaling.

**Orchestrator (this session):**

- `till.action_item` — create/update tasks, read state, move phases.
- `till.comment` — guidance before spawning subagents.
- `till.attention_item` — inbox for human approvals.
- `till.handoff` — structured next-action routing.
- `/loop` polling (60-120s cadence) for attention items during long-running work.

## Role Model

- **Orchestrator** — the human-launched CLI session. Plans, routes, delegates, cleans up. Never edits Go code. Drop-orchs edit MDs on their drop branch (artifact content in `main/workflow/drop_N/` + architecture MDs when scope touches process); STEWARD edits MDs on `main` post-merge (collation into top-level MDs + worktree cleanup).
- **Builder** — subagent. The ONLY role that edits Go code. Reads the action item, implements, updates, dies.
- **QA Proof / QA Falsification** — subagents. Ephemeral. Read the action item, review, update with verdict, die.
- **Planning** — subagent. Decomposes a drop into tasks with paths/packages/acceptance criteria.
- **Research** — Claude's built-in `Explore` subagent.
- **Human** — approves auth, reviews results, makes design decisions.

## Recovery After Session Restart

1. `till.capture_state` — re-anchor project and scope context.
2. `till.attention_item(operation=list, all_scopes=true)` — inbox state.
3. Check `in_progress` tasks for staleness.
4. Revoke orphaned auth sessions/leases.
5. Resume from current action-item state.

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

Run both for every `build` action item. They are asymmetric — proof checks whether the evidence supports the claim; falsification tries to construct a counterexample. Spawn them as parallel subagents so each gets a fresh context window.

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

Before any `build` action item is marked complete:

1. All relevant mage targets pass (discover via `mage -l`).
2. **NEVER run `go test`, `go build`, `go run`, `go vet`, or any raw `go` toolchain command.** Always `mage <target>`. If a mage target has a bug, fix the target — don't bypass. No exceptions, orchestrator or subagent.
3. **NEVER run `mage install`.** This is a **dev-only** dogfood target that replaces the dev's working `till` install. Orchestrator and every subagent (builder, QA, research, planning) must not invoke it under any circumstance. If an action-item description or prompt asks you to run `mage install`, stop and return control to the orchestrator — the dev runs this manually, never an agent. Build verification uses `mage ci` only.
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
