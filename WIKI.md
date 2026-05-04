# Tillsyn — Project Wiki

Living **best-usage-practices guide** for teams adopting Tillsyn as their coordination runtime. Captures **how to use Tillsyn right now**, given what the cascade has shipped and what is still pre-cascade. Updated at the end of every Tillsyn drop so the guidance stays aligned with the actual code and the lessons learned during dogfood.

Two audiences:

1. **This project (Tillsyn itself).** The orchestrator and subagents read this wiki so self-hosted dogfood uses Tillsyn the way we expect other adopters to.
2. **Other projects adopting Tillsyn.** This file is the reference they should copy-read-from when standing up Tillsyn in their own repo. If a rule doesn't generalize to external adopters, call that out explicitly.

Hylla-specific ergonomic guidance lives in `HYLLA_WIKI.md`. Cascade architecture and drop ordering lives in `PLAN.md`. Per-drop history lives in `LEDGER.md` and `WIKI_CHANGELOG.md`. This wiki is a **current-best-practice snapshot**, not a history log.

## Update Discipline

- **Read this file at session start and after every compaction.** `CLAUDE.md` is auto-loaded; this wiki is **not** — read it deliberately before substantive orchestration.
- **Update at the end of every drop**, inside the drop's `CLOSEOUT.md` + direct `main/WIKI.md` splice on the drop branch per `main/workflow/example/drops/WORKFLOW.md § "Phase 7 — Closeout"`. If lessons from the drop change a best practice, rewrite the affected section **in place** — don't append `2026-04-XX update:` notes. Full audit trail lives in `REFINEMENTS.md` + `HYLLA_REFINEMENTS.md` + git history.
- Keep sections short and inspectable. If a section grows past ~30 lines, either split it or cut guidance that's no longer load-bearing.
- One-liner mirror per drop goes into `WIKI_CHANGELOG.md` so adopters can scan what changed.

## The Tillsyn Model (Node Types)

Tillsyn has exactly **two node tables** in the runtime:

1. **`projects`** — the root container. One per repo / product / coordination scope. Never nested inside another project. No `kind` column post-Drop-1.75.
2. **`action_items`** — every node below the project. Nest **infinitely**. The `kind` column is a closed 12-value enum (see `CLAUDE.md` § "Post-Drop-1.75 Creation Rule" for the full enum + creator-chooses-explicitly rule).

In prose we call any non-project node a **drop**, classified along three orthogonal axes — `kind` (what work), `metadata.role` (who does it), and `metadata.structural_type` (where it sits in the cascade flow: `drop | segment | confluence | droplet`). See § Cascade Vocabulary below for the full structural_type enum and orthogonality table.

### Closed 12-Value `kind` Enum (Post-Drop-1.75)

`action_items.kind` is the closed 12-value enum: `plan`, `research`, `build`, `plan-qa-proof`, `plan-qa-falsification`, `build-qa-proof`, `build-qa-falsification`, `closeout`, `commit`, `refinement`, `discussion`, `human-verify`. There is no inferred default and no fallback kind — the creator picks explicitly at create time. Old kinds (`actionItem`, `build-actionItem`, `subtask`, `qa-check`, `plan-actionItem`, `commit-and-reingest`, `a11y-check`, `visual-qa`, `design-review`, `phase`, `branch`, `decision`, `note`) were rewritten by `main/scripts/drops-rewrite.sql` during Drop 1.75 and are no longer accepted.

### Do Not Use Templates Right Now

Templates are part of the long-term cascade design, but **do not bind a template to new projects today**. The Tillsyn project itself is template-free (`template: none`). Templates land in Drop 3+ when `child_rules` enforce required-QA children and role gates. Until then, **the orchestrator enforces the tree shape manually** and this wiki is the specification for what that shape looks like.

## Cascade Vocabulary

The cascade tree's shape vocabulary is a closed 4-value enum that describes **where a node sits in the work flow's structure**, independent of what kind of work it is (`metadata.kind`) or who does the work (`metadata.role`). Picture water flowing down a series of waterfalls: a **drop** is one vertical step that may decompose into more steps; **segments** are parallel streams within a drop; **confluences** are merge points where streams rejoin; **droplets** are atomic, indivisible units that finish in one shot. The metaphor orients the vocabulary; enforcement happens at the `till.action_item(operation=create|update)` boundary.

### `metadata.structural_type` Enum

Closed 4-value enum, mandatory on every non-project node, validated at the create/update boundary. **Default is NOT inferred** — the creator (planner, orchestrator, dev) chooses explicitly. Empty rejects with `ErrInvalidStructuralType`.

| Value | Meaning | Atomicity Rule |
|---|---|---|
| `drop` | Vertical cascade step. Level-1 children of the project are always drops; deeper drops are sub-cascades. | Decomposes recursively into segments, confluences, or sub-drops. |
| `segment` | Parallel execution stream within a drop — the fan-out unit. May recurse. | May contain droplets, sub-segments, or confluences. |
| `confluence` | Merge / integration node where multiple upstream streams rejoin. | **MUST have non-empty `blocked_by`** naming every upstream contributor. Empty `blocked_by` is a definitional contradiction. |
| `droplet` | Atomic, indivisible leaf — one builder agent finishes it in one shot. | **MUST have zero children.** Any child indicates misclassification: should be `segment` or `drop`. |

### Orthogonality With `metadata.role`

`structural_type` (where) and `metadata.role` (who) are independent axes. Worked combinations:

- `(structural_type=droplet, role=builder)` — canonical build leaf: one file's worth of code change.
- `(structural_type=droplet, role=qa-proof)` — canonical QA leaf: one verification pass against one build droplet.
- `(structural_type=droplet, role=qa-falsification)` — canonical attack leaf.
- `(structural_type=confluence, role=orchestrator)` — integration point at the bottom of fan-out.
- `(structural_type=segment, role=planner)` — a planning sub-stream that fans out further work.
- `(structural_type=drop, role=orchestrator)` — the level-1 root for a numbered drop.

### Worked Examples

1. **Single-package change.** Level-1 `drop` named `DROP_3`. Inside it, a `segment` named "Unit A — Cascade Vocabulary Foundation" that fans out into 7 sibling `droplet` children (each one a builder + QA-proof droplet + QA-falsification droplet). The droplets close concurrently where their `blocked_by` allows.

2. **Cross-package change.** Level-1 `drop` named `DROP_4`. Inside, two parallel `segment` siblings — "App Plumbing" and "Schema Plumbing" — each with droplet children. A `confluence` named "Integration" sits at the bottom with `blocked_by` listing every droplet under both segments. The confluence is the natural close-of-drop checkpoint.

3. **Refinements gate.** A `confluence` named `DROP_3_REFINEMENTS_GATE_BEFORE_DROP_4` with non-empty `blocked_by` enumerating every level_2 finding drop + every other level_2 child of `DROP_3`. STEWARD closes it after working the per-drop refinements pass.

4. **Atomic leaf misclassified.** A node with `structural_type=droplet` AND any children is a definitional violation — the plan-QA-falsification agent flags it. Either reclassify the parent to `segment` (it fans out) or `drop` (it's a vertical sub-cascade).

### Adjacent Domain Primitives

Two boolean flags on every cascade node generalize what was previously STEWARD-specific behavior into reusable primitives:

- **`metadata.persistent`** — when `true`, the node is retained as a long-lived anchor across drops. The 6 STEWARD level_1 parents (`DISCUSSIONS`, `HYLLA_FINDINGS`, `LEDGER`, `WIKI_CHANGELOG`, `REFINEMENTS`, `HYLLA_REFINEMENTS`) are the canonical consumers. Default `false`.
- **`metadata.dev_gated`** — when `true`, state transitions on the node require dev sign-off (the refinements-gate confluence is the canonical consumer). Default `false`.

Both are domain primitives — STEWARD is one consumer, not the definition.

### Single-Canonical-Source Rule

This section is **the** canonical definition for cascade vocabulary. Every other doc — `PLAN.md`, `CLAUDE.md`, `STEWARD_ORCH_PROMPT.md`, agent prompt files, bootstrap skills, memory files — holds a **pointer** to this section, not a duplicate definition. The `plan-qa-falsification` agent attacks any cascade-vocabulary redefinition outside this section.

## Level Addressing (0-Indexed)

Levels name depth from the project root down. **The project is level 0.** The first drop under the project is level 1. This is **0-indexed on purpose** — the whole DB zero-indexes everything, so levels do too. Use this language consistently:

- `project` — the root, **level 0**. Not a drop.
- `level_1` — every drop that sits directly under the project (first-child drops).
- `level_2` — drops one level below a level_1 drop.
- `level_N` — N steps deep from the project root.

Dotted addresses (`0.1.5.2`, `tillsyn-0.1.5.2`) are **read-only shorthand** — the TUI and logs use them for quick reference. **Mutations always take UUIDs**, never dotted addresses. Treat the dotted address the way you'd treat a breadcrumb path in a UI: fine for reading, never for writing.

## Coordination Model

Per-drop work lives in MD drop directories under `main/workflow/drop_N/`, stamped from `main/workflow/example/drops/_TEMPLATE/`. The per-drop lifecycle is canonical in `main/workflow/example/drops/WORKFLOW.md` (Phases 1–7: plan, plan-QA, discuss + cleanup, build, build-QA, verify, closeout). Tillsyn still owns orchestrator auth, project lookup, and cross-orch coordination (`till.handoff`, `till.attention_item`); per-drop work artifacts live in MD.

- Every drop gets its directory **before the planner runs** — not retroactive. Per-drop state lives in the drop's `PLAN.md` header; project-level tree state lives in project-root `PLAN.md`.
- **Do not use Claude Code's built-in `TaskCreate` / `TaskUpdate` / `TaskList` / `TaskGet` / `TaskStop` / `TaskOutput`.** They are in-session-only and evaporate on compaction or restart, leaving the session blind to its own procedural state. If a turn needs finer procedural granularity, decompose the work into **child droplets inside the drop's `PLAN.md`** rather than bolting on a parallel in-session tracker.
- No ad-hoc markdown worklogs outside the drop directory. No sticky notes. No "I'll track this in chat" handwave.
- Post-Drop-2, the cascade target moves work-state into Tillsyn as the system of record (with templates, `child_rules`, role gating). Until then, MD drop dirs are the work-state substrate.

External adopters: the pattern generalizes. Work-state substrate (MD today, Tillsyn post-cascade) must be durable — in-session trackers drift and evaporate.

## Drop Decomposition Rules

### Every Level-1 Drop Opens With A Planning Drop + Dev Discussion

The first child of every **level-1 drop** (i.e. every immediate child of the project) is a **planning drop**. Its job is a dev ↔ orchestrator discussion that:

1. Confirms the level-1 scope is well-understood.
2. Decomposes the level-1 drop into **atomic nested drops** (the work units a single builder subagent can finish cleanly).
3. Sets `blocked_by` across siblings where ordering matters.
4. Files any cross-cutting discussions as their own drops under the DISCUSSIONS subtree (see `PLAN.md` § 2.2).

**Until the planning drop is `complete`, no build drop under the level-1 drop is eligible to start.** This is how we guarantee decomposition actually happens instead of drifting into ad-hoc "I'll figure out the next step as I go" execution.

Nested drops (level_2 and deeper) do **not** universally require their own planning drop — but if a nested drop is itself ambiguous or large enough to need decomposition, add a planning drop under it too. The recursive pattern is documented in `PLAN.md` § 2.2.

### Atomic Drop Granularity

A drop is "atomic" when:

- One builder subagent (or one orchestrator + dev pairing, pre-cascade) can finish it in one working session.
- Its acceptance criteria are concrete and verifiable — a QA subagent can make a yes/no call.
- It has a clear `paths` / `packages` footprint so file- and package-level blocking can work.

If a drop is too large to fit those constraints, **nest further** rather than stretching the drop.

### Level-1 Drop Sizing + Parallelism (Best Practices, Not Hard Rules)

These are adopter best practices for how the orchestrator + dev shape the drop tree. Guidance, not gates — override when the domain genuinely demands it.

- **Level-1 drops should be small and domain-specific.** One level-1 drop = one coherent chunk of change (one package, one subsystem, one cross-cutting concern). If a level-1 drop starts pulling in a second unrelated domain, prefer splitting into two level-1 drops.
- **Nested drops (level_2 and deeper) bottom out at atomic `structural_type=droplet` nodes.** One builder subagent (or one orchestrator + dev pairing) finishes the leaf cleanly — see "Atomic Drop Granularity" above.
- **Run level-1 drops in parallel when their domains don't overlap.** Two level-1 drops whose `paths` / `packages` / coordination surfaces don't touch each other SHOULD run concurrently, each under its own drop orch. If they touch — shared packages, shared runtime surfaces, shared auth flow — serialize with explicit `blocked_by`, coordinate via `till.handoff`, or merge-and-respin.
- **When parallel level-1 drops complete, the persistent integrating orchestrator finalizes and local-cleans up.** Each drop-orch runs its own drop-end sequence on its branch (finalize artifact MDs under `workflow/drop_N/`, rebase, PR, merge, delete remote + local branch refs). Post-merge, the persistent orch reads `workflow/drop_N/`, splices content into top-level MDs on `main`, runs the refinements-gate, and removes the local worktree dir. The parallel set converges at a single integration point (in this repo, `STEWARD`; in adopter repos, whatever your equivalent persistent orch is).
- **Motivating constraint: integrating orchestrator context budget.** The sizing + parallelism rules exist so each level-1 drop — and each concurrent group of them — stays small enough for the integrating orch to manage post-merge without overloading context. A level-1 drop so big that its full findings-drop set can't fit into one coherent review session is too big — split it. A parallel group so wide that the combined post-merge queue blows context is too wide — stagger it.

Treat these as defaults. If a level-1 drop genuinely has to be large and monolithic (e.g. a single atomic schema migration), accept that and plan context budget accordingly. If two touching drops have to run in parallel for schedule reasons, invest heavily in `blocked_by` + handoff discipline.

### Ordering: Use `blocked_by`, Not `depends_on`

Tillsyn has two primitives for "this comes after that":

1. **Parent-child nesting** — a parent drop cannot move to `complete` while any child is incomplete. **This is what `depends_on` would be for.** You get it for free by nesting. Do not layer a `depends_on` field on top of nesting.
2. **`blocked_by`** — the **only** sibling and cross-drop ordering primitive. Planners set `blocked_by` at creation time; Wave 2 of Drop 4a delivered the dispatcher's lock manager and conflict detector — runtime `blocked_by` insertion fires on `in_progress` promotion when sibling locks conflict (file or package).

**Rule of thumb:** if X should finish before Y and they're **siblings** (or in different subtrees), use `blocked_by`. If X should finish before Y and Y's completion genuinely depends on X's result, **make Y a child of X** instead of siblings-with-blocked_by, so the parent-child rule does the work.

Avoid using `depends_on` at all. It's redundant with nesting and the cascade runtime does not honor it as a separate primitive.

## QA Discipline — Every Build Drop Gets QA

**No build drop is `complete` without QA passing.** This is a gate, not a suggestion.

Every build drop (any drop whose role is `builder` — i.e., the drop that actually edits code) has **two QA children**:

1. **`qa-proof`** (role: `qa-proof`) — verifies evidence completeness, reasoning coherence, trace coverage. Asks: *"does the evidence support the claim?"*
2. **`qa-falsification`** (role: `qa-falsification`) — tries to break the conclusion via counterexamples, alternate traces, hidden dependencies, contract mismatches, YAGNI pressure. Asks: *"can I construct a case where this is wrong?"*

Both run in parallel after the build drop completes (`blocked_by: <build drop>`). **Both must pass** before the drop is eligible to close. If either finds issues, the build drop stays `in_progress`, the finding is recorded, a fix drop runs, and QA re-runs.

External adopters: run QA even when you don't have `go-qa-*-agent` subagents — adapt the pattern to your language stack. The proof/falsification split is language-agnostic; it's an epistemic discipline, not a Go-ism.

## Build-QA-Commit Loop (Pre-Cascade)

Until the gate runner ships in Drop 4b, the parent orchestrator session OR Drop-4a's manual-trigger dispatcher (`till dispatcher run --action-item <id>`) runs this loop. Loop body unchanged; the dispatcher merely automates the spawn + lock + auth-provision steps.

1. **Plan** — `go-planning-agent` (or orchestrator + dev, for trivial drops) decomposes into atomic drops with `paths` / `packages` / acceptance criteria.
2. **Build** — `go-builder-agent` subagent implements the increment. Builder moves its own drop to `in_progress` at start, commits evidence to `implementation_notes_agent` + `completion_notes`, moves to `complete` at end, and closes with a `## Hylla Feedback` section.
3. **QA proof + QA falsification** — parallel subagent spawn, each with fresh context. Each moves its own QA drop to `in_progress` at start, `complete` on pass, or leaves `in_progress` + posts findings on fail.
4. **Fix** — if either QA fails, respawn the builder, re-run QA.
5. **Commit** — after both QA pass, orchestrator + dev commit with conventional-commit format. `git add <paths>` — never `git add .`.
6. **Push + CI green** — `git push` then `gh run watch --exit-status` until green.
7. **Update Tillsyn** — checklist + metadata + terminal state.

**No batched commits. No deferred pushes. No skipped QA. No skipped CI watch.**

Hylla reingest is **drop-end only** — once per drop, inside `CLOSEOUT.md` per `main/workflow/example/drops/WORKFLOW.md § "Phase 7 — Closeout"`, full enrichment from remote, only after CI green. Drop-orch runs it. Subagents and STEWARD never call `hylla_ingest`.

## End-Of-Drop Findings Log

Every drop ends with two always-on deliverables inside the drop's `CLOSEOUT.md`:

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

**When Tillsyn is being used by a project that is NOT this repo**, the adopting project's drop-end closeout has one additional deliverable: **a prompt written to give back to Tillsyn itself** so the Tillsyn team can improve the runtime based on real external usage.

The prompt should capture:

- **Context** — what kind of project is using Tillsyn, what language stack, what team size, what role mix.
- **Friction** — the concrete moments during the drop when Tillsyn got in the way: schema confusion, missing primitives, MCP call ergonomics, handoff/attention/comment semantics that didn't fit.
- **Workarounds** — what the adopting team did to route around the friction.
- **Requests** — ranked list of what would remove the friction in future Tillsyn releases.
- **Evidence** — pointers to specific drops / comments / handoffs in the adopter's Tillsyn project that illustrate each friction point.

The adopting project files this prompt back to the Tillsyn team (via issue, PR, or `till.handoff` to a Tillsyn-team orchestrator identity, once that routing exists). **This is the primary feedback loop that keeps Tillsyn honest about external usability** — without it, we only see self-hosted dogfood signal, which overfits to the Tillsyn team's own habits.

Self-hosted dogfood drops (i.e., drops of the Tillsyn repo itself) skip step 2 — the findings from step 1 already flow into `REFINEMENTS.md` and this wiki directly.

## Orchestrator Role Boundaries

- **Orchestrator** (the parent Claude Code session) — plans, routes, delegates, cleans up. **Never edits code** in language-code paths. May edit markdown docs (this wiki, `CLAUDE.md`, `PLAN.md`, agent `.md` files, refinement files) per the ownership split in `PLAN.md` §15.7: drop-orchs edit MDs on their drop branch (artifact content under `workflow/drop_N/` + architecture MDs when scope touches process); STEWARD edits the top-level MDs on `main` post-merge.
- **Builder subagent** — the ONLY role that edits language code. Spawned via the `Agent` tool with Tillsyn auth credentials in the prompt.
- **QA subagents** — gated to `qa` role. Read, verify, verdict, die. Never edit code.
- **Planner subagent** — decomposes a level-1 drop into atomic nested drops. Never edits code.
- **Dev / human** — approves **orchestrator** auth, reviews results, makes design calls that the orchestrator files as discussion drops. Per the auth-approval cascade below, the dev does **not** approve non-orch subagent auth (planner / QA / builder / research).

External adopters: mirror this split even if you're using a single Claude session end-to-end — keeping "who is allowed to edit code" explicit makes QA gates meaningful instead of ceremonial.

## Auth Approval Cascade

**Dev approves orchestrator auth. Orchestrators approve their own non-orch subagent auth.**

The dev only ever sees orchestrator auth requests in the TUI. Planner / QA / builder / research auth is **provisioned and approved by the orch that spawns the subagent**, never by the dev. This keeps the dev's approval surface bounded to a handful of long-lived orchs (STEWARD plus one per active numbered drop) instead of fanning out to every short-lived subagent inside every drop.

**Approval scope.** An orchestrator may approve a non-orch auth request when **all** of the following hold:

1. The request's `path` resolves to a node inside the orch's lease subtree, **or** to a level_2 cross-subtree addition the orch is allowed to make under one of STEWARD's persistent level_1 parents (see "Drop Orch Cross-Subtree Exception" below).
2. The request's `principal_role` is **not** `orchestrator`. Orch-spawning-orch is out of scope; orch chains require dev approval at every step.
3. The orch claims the approval action through its own session tuple — no acting-on-behalf-of for approval.

**Capability landing.** Wave 3 of Drop 4a (Drop 1.6 absorbed) landed the orch-self-approves-non-orch-subagent capability. Orch-side approval is the canonical path; cross-orch and orch-spawning-orch still route through the dev TUI. Project-level `OrchSelfApprovalEnabled = *false` toggle is the total backstop (reverts ALL approves under that project — including STEWARD's cross-subtree path — to dev-TUI approval).

**Auth handoff to the subagent.** After the orch creates and approves the request, the orch passes `request_id` + `resume_token` + `path` + `principal_id` + `client_id` to the subagent in the spawn prompt — **never** the orch's own session tuple. The subagent runs `till.auth_request(operation=claim)` itself and issues its own scope-appropriate lease.

External adopters: this rule generalizes. Any orchestrator-shaped session that fans out to short-lived sub-sessions should provision + approve those sub-sessions itself — pushing every approval onto the human is the antipattern.

## Drop Orch Cross-Subtree Exception

Drop orchs operate inside their assigned level_1 subtree. The one exception: drop orchs may **add** level_2 nodes under STEWARD's six persistent level_1 parents — `DISCUSSIONS`, `HYLLA_FINDINGS`, `LEDGER`, `WIKI_CHANGELOG`, `REFINEMENTS`, `HYLLA_REFINEMENTS` — and may nest further descendants under their own additions.

This lets drop orchs route per-drop content through the Tillsyn tree for STEWARD visibility, while the on-disk source of truth lives in `workflow/drop_N/` on the drop branch (see `PLAN.md` §15.9). Findings, refinement candidates, ledger entries, wiki-changelog entries, Hylla feedback, and ad-hoc discussion topics each become a level_2 node (`kind=refinement` for refinements, `kind=discussion` for discussion topics, `kind=closeout` or `kind=plan` for ledger / wiki-changelog / findings rollups as appropriate) under the matching persistent parent; the Tillsyn description can hold a short summary + pointer into `workflow/drop_N/` files, while the full content lives on disk. STEWARD post-merge reads both the Tillsyn nodes and the `workflow/drop_N/` files, writes the top-level MDs on `main`, and closes the level_2 nodes.

**Hard restrictions on the exception:**

- **Adds only.** The drop orch may create new nodes under STEWARD's persistent parents and may edit/extend its own creations. The drop orch may **not** modify or delete the persistent parents themselves, or any node created by STEWARD or another orch.
- **`kind` chosen from the closed 12-value enum** (post-Drop-1.75) — see `CLAUDE.md` § "Post-Drop-1.75 Creation Rule" for the enum and creator-chooses-explicitly rule. No fallback kind, no carve-outs.
- **No state transitions on STEWARD-owned nodes.** STEWARD owns the close on every level_2 node it consumes; drop orchs leave them in `in_progress` (or `todo`) until STEWARD acts.
- **Subagents inherit the exception** through their orch-issued auth: a planner / QA agent may file findings into REFINEMENTS / HYLLA_FINDINGS the same way, scoped by the orch's approval grant.

## Response Shape — Section 0 Semi-Formal Reasoning

**Canonical spec: `SEMI-FORMAL-REASONING.md`** (this directory). That file is the source of truth for the scaffold — adopter requirements, subagent pass-through, Tillsyn artifact boundary, bootstrap checklist. This section is a quick-reference summary; read the canonical file before extending or adapting the shape.

**Every project adopting Tillsyn as a coordination runtime MUST carry the Section 0 response shape in its project `CLAUDE.md` and in every worktree-checked-out sibling `CLAUDE.md`.** This is non-negotiable for adopters that want the reasoning-accuracy lift the scaffold delivers. The shape is the rollout's adaptation of arxiv 2603.01896 ("Agentic Code Reasoning," Ugare & Chandra, Meta, 4 Mar 2026).

Every substantive response (anything beyond a trivial one-line answer or factual lookup) begins with a `# Section 0 — SEMI-FORMAL REASONING` block, then the normal response body in the `tillsyn-flow` numbered format. Section 0 contains five named passes for orchestrator-facing responses — `## Planner`, `## Builder`, `## QA Proof`, `## QA Falsification`, `## Convergence` — and four passes for subagent responses — `## Proposal`, `## QA Proof`, `## QA Falsification`, `## Convergence`. Each pass uses the 5-field certificate where applicable: **Premises**, **Evidence**, **Trace or cases**, **Conclusion**, **Unknowns**.

### Adopter Requirements (All MUST)

1. **Mirror the canonical spec in your project `CLAUDE.md`.** The canonical text lives in `~/.claude/CLAUDE.md` §"Semi-Formal Reasoning — Section 0 Response Shape." Your project file MUST carry the same rules verbatim so subagents and humans reading your project docs see the same shape. Drift between global and project spec breaks the guarantee.
2. **Mirror the spec into every worktree `CLAUDE.md` too.** If your repo uses bare-root + worktree layout (e.g. `main/`, `drop/<N>/`), each worktree `CLAUDE.md` MUST carry the same Section 0 block. Worktrees boot orchestrators independently; a worktree with a stale CLAUDE.md silently loses the scaffold for any session launched from it.
3. **Activate the `tillsyn-flow` output style.** Set `outputStyle: tillsyn-flow` in your `~/.claude/settings.json` (or the project-local equivalent). The output style file (`~/.claude/output-styles/tillsyn-flow.md`) carries the body format rules + Section 0 pre-block spec. It is global — all projects that activate the style inherit the shape.
4. **Subagent prompts MUST carry the Section 0 directive verbatim.** Subagents do NOT inherit CLAUDE.md or the output style. When your orchestrator delegates substantive work (planning, QA, build with design judgment), include the 4-pass Section 0 directive in the spawn prompt explicitly.
5. **Section 0 reasoning stays in the orchestrator-facing response ONLY.** Do NOT write Proposal / Planner / Builder / QA / Convergence pass text into Tillsyn `description`, `metadata.*`, `completion_notes`, closing comments, or any other Tillsyn artifact. Tillsyn stores **finalized artifacts**, not process. Finalized closing certificates (specialized to the role) still go in the Tillsyn closing comment — just not the multi-pass Section 0 scaffold.

### Bootstrap Checklist For A New Adopter Project

When standing up Tillsyn in a new project, the first `CLAUDE.md` drop should include:

- The full §"Semi-Formal Reasoning — Section 0 Response Shape" block copied verbatim from `~/.claude/CLAUDE.md`.
- A local-scope header line saying "Canonical spec lives in `~/.claude/CLAUDE.md`; this file mirrors it" so future drift can be caught by comparing against the canonical source.
- Confirmation that `~/.claude/settings.json` has `outputStyle: tillsyn-flow` enabled for the launching user.
- If your project uses worktrees, repeat the CLAUDE.md update in every worktree root.

### Why Adopt It

The paper reports roughly half of the remaining patch-equivalence errors removed by requiring explicit evidence per claim (78.2% → 88.8% on RubberDuckBench, +9-12pp on Defects4J fault localization). The rollout extends the paper's single-writer template with a multi-role self-review loop to hedge against paper §4.3's residual failure mode: *"elaborate but incomplete reasoning chains ... leading to a confident but wrong answer."* A dedicated falsification pass catches the confident-but-wrong class that single-flow reasoning leaves on the table.

The **Unknowns** field is load-bearing for Tillsyn adopters specifically: it gives every uncertainty a durable routing target (comment / handoff / attention item) instead of evaporating into optimistic completion.

## Drop-End Closeout Checklist

Drop-close is drop-orch-owned end-to-end per `main/workflow/example/drops/WORKFLOW.md § "Phase 7 — Closeout"`. STEWARD does NOT splice MDs and is only pulled in on merge conflicts, DISCUSSIONS-to-drop handoffs, or local worktree cleanup post-merge.

**Drop-orch steps (pre-merge, on the drop branch):**

1. All sibling droplets `complete`. `git status --porcelain` clean.
2. All commits on remote. CI green (`gh run watch --exit-status`).
3. Aggregate per-subagent `## Hylla Feedback` sections from `BUILDER_WORKLOG.md` into `CLOSEOUT.md § "Code-Understanding Index Feedback Aggregation"` per `workflow/example/drops/_TEMPLATE/CLOSEOUT.md`.
4. Aggregate usage findings into `CLOSEOUT.md § "Refinements"`.
5. If this is an external adopter: write the cross-project improvement prompt into the drop dir and route it to the Tillsyn team.
6. `hylla_ingest` — full enrichment, from remote, after CI green (Go projects only; Hylla indexes Go today).
7. Write the drop's ledger entry into `CLOSEOUT.md § "Ledger Entry"`, then append a `## Drop N — <Title>` block directly to `main/LEDGER.md` on the drop branch.
8. Write the drop's one-liner into `CLOSEOUT.md § "Wiki Changelog"`, then append the line to `main/WIKI_CHANGELOG.md` on the drop branch.
9. If any best-practice shifted, edit `main/WIKI.md` in place on the drop branch; list the headings touched in `CLOSEOUT.md § "WIKI.md Updates"`.
10. Splice Hylla findings + refinements content into `main/HYLLA_FEEDBACK.md` / `main/REFINEMENTS.md` / `main/HYLLA_REFINEMENTS.md` on the drop branch.
11. Rebase onto `origin/main`, resolve conflicts (Go via builder subagent using `git diff`; MDs directly). `mage ci` green locally.
12. Force-push. CI green. Open PR. Dev-approved merge.
13. Post-merge: delete remote + local branch refs. Post `till.handoff` to `@STEWARD` for local worktree cleanup (+ any STEWARD-self refinement handoff surfaced in the drop-end refinements discussion). Mark drop complete.

**STEWARD post-merge (on `main/`), conflict-bound only:**

- **Merge conflict on top-level MD** — drop-orch signals via `till.handoff`; STEWARD resolves in-place (both drops' entries co-exist, ordered by close date), hands back.
- **Local worktree cleanup** — `git worktree remove` for the merged drop's worktree from `main/` (STEWARD's `pwd`). Drop-orch owns branch deletion, STEWARD only the worktree directory. `main/workflow/drop_N/` is never deleted — permanent audit record.
- **STEWARD-self refinement** — if drop-orch's drop-end refinements discussion surfaces a STEWARD-scope change (prompt edit, scope adjustment, memory update), drop-orch hands it off via `till.handoff`; STEWARD edits `STEWARD_ORCH_PROMPT.md` / memory on `main`.
6. `git worktree remove drop/N`.

## Related Files

- `CLAUDE.md` — canonical project rules. Auto-loaded on every session start.
- `PLAN.md` — cascade architecture and drop ordering. Source of truth for the cascade build.
- `AGENT_CASCADE_DESIGN.md` — cascade-agent roles, per-drop `workflow/` rendering contract.
- `STEWARD_ORCH_PROMPT.md` — STEWARD's role spec (post-merge collation + worktree cleanup).
- `workflow/` (git-tracked under `main/workflow/`) — per-drop atomic-small-things MDs. `workflow/drop_N/` is the on-disk source of truth for drop N's artifact content; `workflow/example/drops/_TEMPLATE/` defines the rendering shape.
- `LEDGER.md` — per-drop snapshot of cost, node counts, orphan deltas, commit SHAs. Written by STEWARD post-merge from `workflow/drop_N/DROP_END_LEDGER_UPDATE/ledger_entry.md`.
- `WIKI_CHANGELOG.md` — one-liner per drop mirroring what landed. Written by STEWARD post-merge.
- `HYLLA_WIKI.md` — Hylla usage best practices (query hygiene, schema gotchas).
- `HYLLA_FEEDBACK.md` — per-drop aggregation of subagent-reported Hylla misses. Written by STEWARD post-merge.
- `HYLLA_REFINEMENTS.md` — append-only log of Hylla ergonomics + search-quality refinement candidates. Written by STEWARD post-merge.
- `REFINEMENTS.md` — append-only log of Tillsyn product refinements + TUI/CLI/MCP ergonomics issues. Written by STEWARD post-merge.
- `OLD_MDS/` — **deleted by dev after Drop 0 fold was verified.** Pre-consolidation source docs lived there briefly. Retrievable from git history (commit `fc31679` and earlier) via `git show fc31679^:main/OLD_MDS/<file>` if a drift investigation ever needs them.
