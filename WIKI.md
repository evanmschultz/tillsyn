# Tillsyn — Project Wiki

Living **best-usage-practices guide** for teams adopting Tillsyn as their coordination runtime. Captures **how to use Tillsyn right now**, given what the cascade has shipped and what is still pre-cascade. Updated at the end of every Tillsyn drop so the guidance stays aligned with the actual code and the lessons learned during dogfood.

Two audiences:

1. **This project (Tillsyn itself).** The orchestrator and subagents read this wiki so self-hosted dogfood uses Tillsyn the way we expect other adopters to.
2. **Other projects adopting Tillsyn.** This file is the reference they should copy-read-from when standing up Tillsyn in their own repo. If a rule doesn't generalize to external adopters, call that out explicitly.

Hylla-specific ergonomic guidance lives in `HYLLA_WIKI.md`. Cascade architecture and drop ordering lives in `PLAN.md`. Per-drop history lives in `LEDGER.md` and `WIKI_CHANGELOG.md`. This wiki is a **current-best-practice snapshot**, not a history log.

## Update Discipline

- **Read this file at session start and after every compaction.** `CLAUDE.md` is auto-loaded; this wiki is **not** — read it deliberately before substantive orchestration.
- **Update at the end of every drop**, inside the `DROP <N> END — LEDGER UPDATE` task. If lessons from the drop change a best practice, rewrite the affected section **in place** — don't append `2026-04-XX update:` notes. Full audit trail lives in `REFINEMENTS.md` + `HYLLA_REFINEMENTS.md` + git history.
- Keep sections short and inspectable. If a section grows past ~30 lines, either split it or cut guidance that's no longer load-bearing.
- One-liner mirror per drop goes into `WIKI_CHANGELOG.md` so adopters can scan what changed.

## The Tillsyn Model (Node Types)

Tillsyn has exactly **two node types** you should use today:

1. **Project** — the root container. One per repo / product / coordination scope. Never nested inside another project.
2. **Drop** — every node below the project. Drops nest **infinitely**.

A "drop" is the Tillsyn-native word for a unit of work. In current runtime terms it is a plan item with `kind='task'` (the pre-Drop-2 creation rule — see `CLAUDE.md` § "Pre-Drop-2 Creation Rule"). Drop 2 of the cascade collapses every non-project kind to literal `kind='drop'`; for now, write `kind='task', scope='task'` and **refer to the node as a drop in prose**.

### Do Not Use Other Kinds Today

The `kind_catalog` still lists `build-task`, `subtask`, `qa-check`, `plan-task`, `commit-and-reingest`, `a11y-check`, `visual-qa`, `design-review`, `phase`, `branch`, `decision`, `note`. **Do not use them.** Drop 2's SQL rewrites every non-project node to `drop`. Pre-Drop-2, stick to plain `task` and keep the runtime writes consistent.

### Do Not Use Templates Right Now

Templates are part of the long-term cascade design, but **do not bind a template to new projects today**. The Tillsyn project itself is template-free (`template: none`). Templates will come back in Drop 3+ when `child_rules` can enforce required-QA subtasks and role gates. Until then, **the orchestrator enforces the tree shape manually** and this wiki is the specification for what that shape looks like.

## Level Addressing (0-Indexed)

Levels name depth from the project root down. **The project is level 0.** The first drop under the project is level 1. This is **0-indexed on purpose** — the whole DB zero-indexes everything, so levels do too. Use this language consistently:

- `project` — the root, **level 0**. Not a drop.
- `level_1` — every drop that sits directly under the project (first-child drops).
- `level_2` — drops one level below a level_1 drop.
- `level_N` — N steps deep from the project root.

Dotted addresses (`0.1.5.2`, `tillsyn-0.1.5.2`) are **read-only shorthand** — the TUI and logs use them for quick reference. **Mutations always take UUIDs**, never dotted addresses. Treat the dotted address the way you'd treat a breadcrumb path in a UI: fine for reading, never for writing.

## Tillsyn Is the System of Record

**Every action lives in Tillsyn.** This is the non-negotiable rule.

- Every piece of work gets a Tillsyn drop **before it starts**. Not retroactive.
- When work starts on a drop, move it to `in_progress` **immediately**. No `todo` items left while someone is working on them.
- **Do not use Claude Code's built-in `TaskCreate` / `TaskUpdate` / `TaskList` / `TaskGet` / `TaskStop` / `TaskOutput`.** They are in-session-only and evaporate on compaction or restart, leaving the session blind to its own procedural state. If a turn needs finer procedural granularity, decompose the work into **child Tillsyn drops** rather than bolting on a parallel in-session tracker.
- No markdown worklogs. No sticky notes. No "I'll track this in chat" handwave.
- If it's not in Tillsyn, it didn't happen.

External adopters: this rule generalizes. Any client (Claude, Codex, a CLI user) that uses Tillsyn for coordination must funnel all work state through Tillsyn. Client-local trackers drift; Tillsyn is durable.

## Drop Decomposition Rules

### Every Level-1 Drop Opens With A Planning Drop + Dev Discussion

The first child of every **level-1 drop** (i.e. every immediate child of the project) is a **planning drop**. Its job is a dev ↔ orchestrator discussion that:

1. Confirms the level-1 scope is well-understood.
2. Decomposes the level-1 drop into **atomic nested drops** (the work units a single builder subagent can finish cleanly).
3. Sets `blocked_by` across siblings where ordering matters.
4. Files any cross-cutting discussions as their own drops under the DISCUSSIONS subtree (see `PLAN.md` § 2.2).

**Until the planning drop is `done`, no build drop under the level-1 drop is eligible to start.** This is how we guarantee decomposition actually happens instead of drifting into ad-hoc "I'll figure out the next step as I go" execution.

Nested drops (level_2 and deeper) do **not** universally require their own planning drop — but if a nested drop is itself ambiguous or large enough to need decomposition, add a planning drop under it too. The recursive pattern is documented in `PLAN.md` § 2.2.

### Atomic Drop Granularity

A drop is "atomic" when:

- One builder subagent (or one orchestrator + dev pairing, pre-cascade) can finish it in one working session.
- Its acceptance criteria are concrete and verifiable — a QA subagent can make a yes/no call.
- It has a clear `paths` / `packages` footprint so file- and package-level blocking can work.

If a drop is too large to fit those constraints, **nest further** rather than stretching the drop.

### Ordering: Use `blocked_by`, Not `depends_on`

Tillsyn has two primitives for "this comes after that":

1. **Parent-child nesting** — a parent drop cannot move to `done` while any child is incomplete. **This is what `depends_on` would be for.** You get it for free by nesting. Do not layer a `depends_on` field on top of nesting.
2. **`blocked_by`** — the **only** sibling and cross-drop ordering primitive. Planners set `blocked_by` at creation time; the dispatcher adds runtime blockers when file/package locks conflict (Drop 4+).

**Rule of thumb:** if X should finish before Y and they're **siblings** (or in different subtrees), use `blocked_by`. If X should finish before Y and Y's completion genuinely depends on X's result, **make Y a child of X** instead of siblings-with-blocked_by, so the parent-child rule does the work.

Avoid using `depends_on` at all. It's redundant with nesting and the cascade runtime does not honor it as a separate primitive.

## QA Discipline — Every Build Drop Gets QA

**No build drop is `done` without QA passing.** This is a gate, not a suggestion.

Every build drop (any drop whose role is `builder` — i.e., the drop that actually edits code) has **two QA children**:

1. **`qa-proof`** (role: `qa-proof`) — verifies evidence completeness, reasoning coherence, trace coverage. Asks: *"does the evidence support the claim?"*
2. **`qa-falsification`** (role: `qa-falsification`) — tries to break the conclusion via counterexamples, alternate traces, hidden dependencies, contract mismatches, YAGNI pressure. Asks: *"can I construct a case where this is wrong?"*

Both run in parallel after the build drop completes (`blocked_by: <build drop>`). **Both must pass** before the drop is eligible to close. If either finds issues, the build drop stays `in_progress`, the finding is recorded, a fix drop runs, and QA re-runs.

External adopters: run QA even when you don't have `go-qa-*-agent` subagents — adapt the pattern to your language stack. The proof/falsification split is language-agnostic; it's an epistemic discipline, not a Go-ism.

## Build-QA-Commit Loop (Pre-Cascade)

Until the cascade dispatcher ships (Drop 4+), the parent orchestrator session runs this loop manually:

1. **Plan** — `go-planning-agent` (or orchestrator + dev, for trivial drops) decomposes into atomic drops with `paths` / `packages` / acceptance criteria.
2. **Build** — `go-builder-agent` subagent implements the increment. Builder moves its own drop to `in_progress` at start, commits evidence to `implementation_notes_agent` + `completion_notes`, moves to `done` at end, and closes with a `## Hylla Feedback` section.
3. **QA proof + QA falsification** — parallel subagent spawn, each with fresh context. Each moves its own QA drop to `in_progress` at start, `done` on pass, or leaves `in_progress` + posts findings on fail.
4. **Fix** — if either QA fails, respawn the builder, re-run QA.
5. **Commit** — after both QA pass, orchestrator + dev commit with conventional-commit format. `git add <paths>` — never `git add .`.
6. **Push + CI green** — `git push` then `gh run watch --exit-status` until green.
7. **Update Tillsyn** — checklist + metadata + terminal state.

**No batched commits. No deferred pushes. No skipped QA. No skipped CI watch.**

Hylla reingest is **drop-end only** — once per drop, inside the `DROP <N> END — LEDGER UPDATE` task, full enrichment from remote, only after CI green. Subagents never call `hylla_ingest`.

## End-Of-Drop Findings Log

Every drop ends with two always-on deliverables inside the `DROP <N> END — LEDGER UPDATE` task:

### 1. Usage Findings — What Went Well, What Hurt

Aggregate the drop's actual usage experience — the kind of thing you can only learn by working through the drop:

- **Ergonomic wins** — patterns / MCP shapes / CLI commands / TUI flows that felt natural.
- **Ergonomic pain** — awkward parameters, confusing response shapes, opaque IDs, workflows that fought us.
- **Bugs** — hit or worked-around during the drop, with enough detail to file a real fix drop later.
- **Usage lessons** — wiki edits that came out of the drop (role model, naming rules, blocker semantics, etc.).

These land in:

- `HYLLA_FEEDBACK.md` for Hylla-specific feedback (aggregated from subagent `## Hylla Feedback` sections in closing comments).
- `REFINEMENTS.md` for Tillsyn product / CLI / TUI / MCP ergonomics findings.
- `HYLLA_REFINEMENTS.md` for Hylla search-quality / ergonomics findings.
- Direct edits to this wiki for rules that changed.

### 2. Cross-Project Improvement Prompt (When Tillsyn Is Used Externally)

**When Tillsyn is being used by a project that is NOT this repo**, the adopting project's drop-end task has one additional deliverable: **a prompt written to give back to Tillsyn itself** so the Tillsyn team can improve the runtime based on real external usage.

The prompt should capture:

- **Context** — what kind of project is using Tillsyn, what language stack, what team size, what role mix.
- **Friction** — the concrete moments during the drop when Tillsyn got in the way: schema confusion, missing primitives, MCP call ergonomics, handoff/attention/comment semantics that didn't fit.
- **Workarounds** — what the adopting team did to route around the friction.
- **Requests** — ranked list of what would remove the friction in future Tillsyn releases.
- **Evidence** — pointers to specific drops / comments / handoffs in the adopter's Tillsyn project that illustrate each friction point.

The adopting project files this prompt back to the Tillsyn team (via issue, PR, or `till.handoff` to a Tillsyn-team orchestrator identity, once that routing exists). **This is the primary feedback loop that keeps Tillsyn honest about external usability** — without it, we only see self-hosted dogfood signal, which overfits to the Tillsyn team's own habits.

Self-hosted dogfood drops (i.e., drops of the Tillsyn repo itself) skip step 2 — the findings from step 1 already flow into `REFINEMENTS.md` and this wiki directly.

## Orchestrator Role Boundaries

- **Orchestrator** (the parent Claude Code session) — plans, routes, delegates, cleans up. **Never edits code** in language-code paths. May edit markdown docs (this wiki, `CLAUDE.md`, `PLAN.md`, agent `.md` files, refinement files).
- **Builder subagent** — the ONLY role that edits language code. Spawned via the `Agent` tool with Tillsyn auth credentials in the prompt.
- **QA subagents** — gated to `qa` role. Read, verify, verdict, die. Never edit code.
- **Planner subagent** — decomposes a level-1 drop into atomic nested drops. Never edits code.
- **Dev / human** — approves auth, reviews results, makes design calls that the orchestrator files as discussion drops.

External adopters: mirror this split even if you're using a single Claude session end-to-end — keeping "who is allowed to edit code" explicit makes QA gates meaningful instead of ceremonial.

## Drop-End Closeout Checklist

Every drop's final task is `DROP <N> END — LEDGER UPDATE`. Orchestrator-role-gated. `blocked_by` every other drop in the tree.

1. All sibling drops `done`. `git status --porcelain` clean.
2. All commits on remote. CI green (`gh run watch --exit-status`).
3. Aggregate per-subagent `## Hylla Feedback` sections into `HYLLA_FEEDBACK.md`.
4. Aggregate usage findings into `REFINEMENTS.md` / `HYLLA_REFINEMENTS.md`.
5. If this is an external adopter: write the cross-project improvement prompt and route it to the Tillsyn team.
6. `hylla_ingest` — full enrichment, from remote, after CI green (Go projects only; Hylla indexes Go today).
7. Append entry to `LEDGER.md`.
8. Append one-liner to `WIKI_CHANGELOG.md`.
9. Update the relevant section(s) of this wiki if anything shipped that changed best practice.

## Related Files

- `CLAUDE.md` — canonical project rules. Auto-loaded on every session start.
- `PLAN.md` — cascade architecture and drop ordering. Source of truth for the cascade build.
- `LEDGER.md` — per-drop snapshot of cost, node counts, orphan deltas, commit SHAs.
- `WIKI_CHANGELOG.md` — one-liner per drop mirroring what landed.
- `HYLLA_WIKI.md` — Hylla usage best practices (query hygiene, schema gotchas).
- `HYLLA_FEEDBACK.md` — per-drop aggregation of subagent-reported Hylla misses.
- `HYLLA_REFINEMENTS.md` — append-only log of Hylla ergonomics + search-quality refinement candidates.
- `REFINEMENTS.md` — append-only log of Tillsyn product refinements + TUI/CLI/MCP ergonomics issues.
- `OLD_MDS/` — pre-consolidation source docs; folded into this wiki + `PLAN.md` + `README.md` in Drop 0. Kept as an audit trail until dev-verified safe to delete.
