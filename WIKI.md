# Tillsyn — Project Wiki

Living **best-usage-practices guide** for teams adopting Tillsyn as their coordination runtime. Captures **how to use Tillsyn right now**, given what the cascade has shipped and what is still pre-cascade. Updated at the end of every Tillsyn drop so the guidance stays aligned with the actual code and the lessons learned during dogfood.

Two audiences:

1. **This project (Tillsyn itself).** The orchestrator and subagents read this wiki so self-hosted dogfood uses Tillsyn the way we expect other adopters to.
2. **Other projects adopting Tillsyn.** This file is the reference they should copy-read-from when standing up Tillsyn in their own repo. If a rule doesn't generalize to external adopters, call that out explicitly.

Cascade architecture and drop ordering live in `PLAN.md`. Per-drop history lives in git log + the drop's Tillsyn `kind=closeout` action_item comments. This wiki is a **current-best-practice snapshot**, not a history log.

## Update Discipline

- **Read this file at session start and after every compaction.** `CLAUDE.md` is auto-loaded; this wiki is **not** — read it deliberately before substantive orchestration.
- **Update at the end of every drop** by editing this file in place on the drop branch. If lessons from the drop change a best practice, rewrite the affected section **in place** — don't append `2026-04-XX update:` notes. Full audit trail lives in the drop's `kind=closeout` action_item comments + git history.
- Keep sections short and inspectable. If a section grows past ~30 lines, either split it or cut guidance that's no longer load-bearing.

## The Tillsyn Model (Node Types)

Tillsyn has exactly **two node tables** in the runtime:

1. **`projects`** — the root container. One per repo / product / coordination scope. Never nested inside another project. No `kind` column post-Drop-1.75.
2. **`action_items`** — every node below the project. Nest **infinitely**. The `kind` column is a closed 12-value enum (see `CLAUDE.md` § "Post-Drop-1.75 Creation Rule" for the full enum + creator-chooses-explicitly rule).

In prose we call any non-project node a **cascade node**, classified along three orthogonal axes — `kind` (what work), `metadata.role` (who does it), and `metadata.structural_type` (where it sits in the cascade flow: `cascade | drop | segment | confluence | droplet`). The level-1 unit (direct child of the project) is a **cascade** — the whole tree of work; a **drop** is any vertical decomposition step BELOW level-1. See § Cascade Vocabulary below for the full structural_type enum and orthogonality table.

### Closed 12-Value `kind` Enum (Post-Drop-1.75)

`action_items.kind` is the closed 12-value enum: `plan`, `research`, `build`, `plan-qa-proof`, `plan-qa-falsification`, `build-qa-proof`, `build-qa-falsification`, `closeout`, `commit`, `refinement`, `discussion`, `human-verify`. There is no inferred default and no fallback kind — the creator picks explicitly at create time. Old kinds (`actionItem`, `build-actionItem`, `subtask`, `qa-check`, `plan-actionItem`, `commit-and-reingest`, `a11y-check`, `visual-qa`, `design-review`, `phase`, `branch`, `decision`, `note`) were rewritten by `main/scripts/drops-rewrite.sql` during Drop 1.75 and are no longer accepted.

### Do Not Use Templates Right Now

Templates are part of the long-term cascade design, but **do not bind a template to new projects today**. The Tillsyn project itself is template-free (`template: none`). Templates land in Drop 3+ when `child_rules` enforce required-QA children and role gates. Until then, **the orchestrator enforces the tree shape manually** and this wiki is the specification for what that shape looks like.

## Cascade Vocabulary

The cascade tree's shape vocabulary is a closed 5-value enum that describes **where a node sits in the work flow's structure**, independent of what kind of work it is (`metadata.kind`) or who does the work (`metadata.role`). Picture water flowing down a series of waterfalls: a **cascade** is the whole waterfall sequence (the level-1 tree of work); a **drop** is one vertical step within the cascade; **segments** are parallel streams within a drop; **confluences** are merge points where streams rejoin; **droplets** are atomic, indivisible units that finish in one shot. The metaphor orients the vocabulary; enforcement happens at the `till.action_item(operation=create|update)` boundary.

**Note (2026-05-21):** `cascade` is the 5th value of `structural_type`, added per dev directive. Go enum work tracked at Tillsyn action_item `62569299-6522-401e-a15b-c6f61e2dc609`. Until that lands, level-1 nodes carry `structural_type=drop` as a placeholder; the new vocabulary is documented here so plan-QA-falsification can attack misuse going forward.

### `metadata.structural_type` Enum

Closed 5-value enum, mandatory on every non-project node, validated at the create/update boundary. **Default is NOT inferred** — the creator (planner, orchestrator, dev) chooses explicitly. Empty rejects with `ErrInvalidStructuralType`.

| Value | Meaning | Atomicity Rule |
|---|---|---|
| `cascade` | Level-1 cascade tree. Direct child of the project; the whole sequence of drops + segments + confluences + droplets that delivers one coherent change. | **MUST have empty `parent_id`** (level-1 only — the project is not modeled as a parent action_item). Decomposes into drops / segments / sub-droplets. |
| `drop` | Vertical decomposition step within a cascade. Level-2+ only — direct children of the project are cascades, not drops. | Decomposes recursively into segments, confluences, or sub-drops. |
| `segment` | Parallel execution stream within a drop — the fan-out unit. May recurse. | May contain droplets, sub-segments, or confluences. |
| `confluence` | Merge / integration node where multiple upstream streams rejoin. | **MUST have non-empty `blocked_by`** naming every upstream contributor. Empty `blocked_by` is a definitional contradiction. |
| `droplet` | Atomic, indivisible leaf — one builder agent finishes it in one shot. | **MUST have zero children.** Any child indicates misclassification: should be `segment`, `drop`, or `cascade`. |

### Orthogonality With `metadata.role`

`structural_type` (where) and `metadata.role` (who) are independent axes. Worked combinations:

- `(structural_type=droplet, role=builder)` — canonical build leaf: one file's worth of code change.
- `(structural_type=droplet, role=qa-proof)` — canonical QA leaf: one verification pass against one build droplet.
- `(structural_type=droplet, role=qa-falsification)` — canonical attack leaf.
- `(structural_type=confluence, role=orchestrator)` — integration point at the bottom of fan-out.
- `(structural_type=segment, role=planner)` — a planning sub-stream that fans out further work.
- `(structural_type=cascade, role=orchestrator)` — the level-1 root for a named cascade.
- `(structural_type=drop, role=orchestrator)` — a level-2+ vertical step inside a cascade.

### Worked Examples

1. **Single-package change.** Level-1 `cascade` named `CASCADE_3`. Inside it, a `segment` named "Unit A — Cascade Vocabulary Foundation" that fans out into 7 sibling `droplet` children (each one a builder + QA-proof droplet + QA-falsification droplet). The droplets close concurrently where their `blocked_by` allows.

2. **Cross-package change.** Level-1 `cascade` named `CASCADE_4`. Inside, two parallel `segment` siblings — "App Plumbing" and "Schema Plumbing" — each with droplet children. A `confluence` named "Integration" sits at the bottom with `blocked_by` listing every droplet under both segments. The confluence is the natural close-of-cascade checkpoint.

3. **Refinements gate.** A `confluence` named `CASCADE_3_REFINEMENTS_GATE_BEFORE_CASCADE_4` with non-empty `blocked_by` enumerating every level_2 finding drop + every other level_2 child of `CASCADE_3`. Dev closes it after the per-cascade refinements pass.

4. **Atomic leaf misclassified.** A node with `structural_type=droplet` AND any children is a definitional violation — the plan-QA-falsification agent flags it. Either reclassify the parent to `segment`, `drop`, or `cascade` depending on its role in the tree.

5. **Existing names stay historical.** Action items with titles like `DROP 4D CODEX MULTI BACKEND` keep their original names; only new cascades use the `CASCADE_<NAME>` prefix going forward. The `structural_type` value updates regardless of title — the Tillsyn-internal `structural_type=cascade` migration covers historical items via DB wipe per pre-MVP "No Migration Logic" rule.

### Adjacent Domain Primitives

Two boolean flags on every cascade node:

- **`metadata.persistent`** — when `true`, the node is retained as a long-lived anchor across drops. Used for perpetual umbrellas (refinement queues, discussion parks) that outlive any single drop. Default `false`.
- **`metadata.dev_gated`** — when `true`, state transitions on the node require dev sign-off (the refinements-gate confluence is the canonical consumer). Default `false`.

### Single-Canonical-Source Rule

This section is **the** canonical definition for cascade vocabulary. Every other doc — `PLAN.md`, `CLAUDE.md`, agent prompt files, bootstrap skills, memory files — holds a **pointer** to this section, not a duplicate definition. The `plan-qa-falsification` agent attacks any cascade-vocabulary redefinition outside this section.

## Level Addressing (0-Indexed)

Levels name depth from the project root down. **The project is level 0.** The first drop under the project is level 1. This is **0-indexed on purpose** — the whole DB zero-indexes everything, so levels do too. Use this language consistently:

- `project` — the root, **level 0**. Not a drop.
- `level_1` — every drop that sits directly under the project (first-child drops).
- `level_2` — drops one level below a level_1 drop.
- `level_N` — N steps deep from the project root.

Dotted addresses (`0.1.5.2`, `tillsyn-0.1.5.2`) are **read-only shorthand** — the TUI and logs use them for quick reference. **Mutations always take UUIDs**, never dotted addresses. Treat the dotted address the way you'd treat a breadcrumb path in a UI: fine for reading, never for writing.

## Coordination Model

**Tillsyn IS the work-tracking substrate.** Drop 2 closed; templates + `child_rules` + role gating + auto-QA-twin spawning + first-class `paths` / `packages` fields shipped. Work-state lives in Tillsyn action_items, not in MD drop directories. Dogfooding Tillsyn means using Tillsyn end-to-end for its own development.

- **A drop = a Tillsyn action_item subtree.** Root is `kind=plan`, `structural_type=drop`, directly under the project. Template auto-creates `plan-qa-proof` + `plan-qa-falsification` children.
- **Droplet rows = `kind=build` action_items** as children of the root, with `paths` / `packages` declared and acceptance criteria in description prose. Template auto-creates `build-qa-proof` + `build-qa-falsification` children per build.
- **Worklogs, QA verdicts, closeout findings = `till.comment` on the relevant action_item.** No standalone `*.md` files inside the drop dir for these — comments are the durable audit trail.
- **Cross-cutting decisions = `kind=discussion` action_item.** Description = converged shape; comments = audit trail of dev direct quotes.
- **Dev actions = `till.handoff` addressed to dev.** Not MD checklist rows.
- **Do NOT use Claude Code's built-in `TaskCreate` / `TaskUpdate` / `TaskList` / `TaskGet` / `TaskStop` / `TaskOutput`.** They evaporate on compaction or restart. If a turn needs finer procedural granularity, decompose into child Tillsyn action_items.
- **Existing `workflow/drop_N/` MD directories from pre-migration drops stay in tree as historical audit** per `feedback_never_remove_workflow_files.md`. Do NOT create new MD content for new drops — Tillsyn-native is the system of record going forward.

**External adopters:** the pattern generalizes. Work-state MUST be durable across compaction / restart / multi-session — Tillsyn (or an equivalent durable runtime in your stack) is the right substrate. In-session trackers drift and evaporate.

**For adopters who don't yet have Tillsyn installed**, the MD-bridge pattern documented at `main/workflow/example/drops/WORKFLOW.md` + `main/workflow/example/CLAUDE.md` is the pre-Tillsyn scaffolding. Tillsyn-the-project does not follow its own adopter-bridge template — we use Tillsyn.

## Drop Decomposition Rules

### Every Cascade Opens With A Planning Drop + Dev Discussion

The first child of every **cascade** (i.e. every immediate child of the project) is a **planning drop**. Its job is a dev ↔ orchestrator discussion that:

1. Confirms the cascade's scope is well-understood.
2. Decomposes the cascade into **atomic nested drops** (the work units a single builder subagent can finish cleanly).
3. Sets `blocked_by` across siblings where ordering matters.
4. Files any cross-cutting decisions as `kind=discussion` action_items under the project.

**Until the planning drop is `complete`, no build droplet under the cascade is eligible to start.** This is how we guarantee decomposition actually happens instead of drifting into ad-hoc "I'll figure out the next step as I go" execution.

Nested drops (level_2 and deeper) do **not** universally require their own planning drop — but if a nested drop is itself ambiguous or large enough to need decomposition, add a planning drop under it too. The recursive pattern follows the same plan-then-build rhythm at every level.

### Atomic Droplet Granularity

A droplet is "atomic" when:

- One builder subagent (or one orchestrator + dev pairing) can finish it in one working session.
- Its acceptance criteria are concrete and verifiable — a QA subagent can make a yes/no call.
- It has a clear `paths` / `packages` footprint so file- and package-level blocking can work.

If a leaf is too large to fit those constraints, **nest further** rather than stretching the droplet — emit a `drop` or `segment` parent and decompose underneath.

### Cascade Sizing + Parallelism (Best Practices, Not Hard Rules)

These are adopter best practices for how the orchestrator + dev shape the cascade tree. Guidance, not gates — override when the domain genuinely demands it.

- **Cascades should be small and domain-specific.** One cascade = one coherent chunk of change (one package, one subsystem, one cross-cutting concern). If a cascade starts pulling in a second unrelated domain, prefer splitting into two cascades.
- **Nested drops (level_2 and deeper) bottom out at atomic `structural_type=droplet` nodes.** One builder subagent (or one orchestrator + dev pairing) finishes the leaf cleanly — see "Atomic Droplet Granularity" above.
- **Run cascades in parallel when their domains don't overlap.** Two cascades whose `paths` / `packages` / coordination surfaces don't touch each other SHOULD run concurrently, each under its own cascade-orch. If they touch — shared packages, shared runtime surfaces, shared auth flow — serialize with explicit `blocked_by`, coordinate via `till.handoff`, or merge-and-respin.
- **When parallel cascades complete, each cascade-orch owns its own close-out end-to-end.** Each cascade-orch runs its own cascade-end sequence on its branch (rebase, PR, merge, delete remote + local branch refs, move cascade root action_item to `complete`). Post-merge cleanup (worktree removal) happens from `main/`, never from inside the worktree being removed.
- **Motivating constraint: cascade-orch context budget.** The sizing + parallelism rules exist so each cascade stays small enough for one cascade-orch to manage end-to-end without overloading context. A cascade so big that its full findings set can't fit into one coherent review session is too big — split it.

Treat these as defaults. If a cascade genuinely has to be large and monolithic (e.g. a single atomic schema migration), accept that and plan context budget accordingly. If two touching cascades have to run in parallel for schedule reasons, invest heavily in `blocked_by` + handoff discipline.

### Ordering: Use `blocked_by`, Not `depends_on`

Tillsyn has two primitives for "this comes after that":

1. **Parent-child nesting** — a parent drop cannot move to `complete` while any child is incomplete. **This is what `depends_on` would be for.** You get it for free by nesting. Do not layer a `depends_on` field on top of nesting.
2. **`blocked_by`** — the **only** sibling and cross-cascade ordering primitive. Planners set `blocked_by` at creation time; Wave 2 of Drop 4a delivered the dispatcher's lock manager and conflict detector — runtime `blocked_by` insertion fires on `in_progress` promotion when sibling locks conflict (file or package).

**Rule of thumb:** if X should finish before Y and they're **siblings** (or in different subtrees), use `blocked_by`. If X should finish before Y and Y's completion genuinely depends on X's result, **make Y a child of X** instead of siblings-with-blocked_by, so the parent-child rule does the work.

Avoid using `depends_on` at all. It's redundant with nesting and the cascade runtime does not honor it as a separate primitive.

## QA Discipline — Every Build Droplet Gets QA

**No build droplet is `complete` without QA passing.** This is a gate, not a suggestion.

Every build droplet (any node whose `kind=build` — i.e., the leaf that actually edits code) has **two QA children**:

1. **`build-qa-proof`** (role: `qa-proof`) — verifies evidence completeness, reasoning coherence, trace coverage. Asks: *"does the evidence support the claim?"*
2. **`build-qa-falsification`** (role: `qa-falsification`) — tries to break the conclusion via counterexamples, alternate traces, hidden dependencies, contract mismatches, YAGNI pressure. Asks: *"can I construct a case where this is wrong?"*

Both run in parallel after the build droplet completes (`blocked_by: <build droplet>`). **Both must pass** before the droplet is eligible to close. If either finds issues, the droplet stays `in_progress`, the finding is recorded, a fix runs, and QA re-runs. The same proof+falsification pair applies to `kind=plan` parents via `plan-qa-proof` + `plan-qa-falsification` — every plan node gates descent on its plan-QA twins per the "Plan-QA twins MUST close BEFORE sibling build droplets start" Hard Rule.

External adopters: run QA even when you don't have `go-qa-*-agent` subagents — adapt the pattern to your language stack. The proof/falsification split is language-agnostic; it's an epistemic discipline, not a Go-ism.

## Build-QA-Commit Loop

The parent orchestrator session — or the cascade dispatcher (`till dispatcher run --action-item <id>`) when binding-resolved — runs this loop. The dispatcher automates the spawn + lock + auth-provision steps; the loop body is identical either way.

1. **Plan** — planning-agent (or orchestrator + dev, for trivial cascades) decomposes into atomic droplets with `paths` / `packages` / acceptance criteria.
2. **Build** — builder-agent subagent implements the increment. Builder moves its own droplet to `in_progress` at start, commits evidence to `implementation_notes_agent` + `completion_notes`, moves to `complete` at end, and closes with a `## Hylla Feedback` section.
3. **QA proof + QA falsification** — parallel subagent spawn, each with fresh context. Each moves its own QA droplet to `in_progress` at start, `complete` on pass, or leaves `in_progress` + posts findings on fail.
4. **Fix** — if either QA fails, respawn the builder, re-run QA.
5. **Commit** — after both QA pass, orchestrator + dev commit with conventional-commit format. `git add <paths>` — never `git add .`.
6. **Push + CI green** — `git push` then `gh run watch --exit-status` until green.
7. **Update Tillsyn** — checklist + metadata + terminal state.

**No batched commits. No deferred pushes. No skipped QA. No skipped CI watch.**

Hylla reingest is **cascade-end only** — once per cascade, after the cascade root action_item moves to `complete`, full enrichment from the GitHub remote, only after `git push` + `gh run watch --exit-status` green. Cascade-orch runs it. Subagents never call `hylla_ingest`.

## End-Of-Cascade Findings Log

Every cascade ends with findings collected in Tillsyn-native form, not in MD aggregation files.

### 1. Usage Findings — What Went Well, What Hurt

Aggregate the cascade's actual usage experience — the kind of thing you can only learn by working through the cascade:

- **Ergonomic wins** — patterns / MCP shapes / CLI commands / TUI flows that felt natural.
- **Ergonomic pain** — awkward parameters, confusing response shapes, opaque IDs, workflows that fought us.
- **Bugs** — hit or worked-around during the cascade, with enough detail to file a real fix cascade later.
- **Usage lessons** — wiki edits that came out of the cascade (role model, naming rules, blocker semantics, etc.).

These land as Tillsyn action_items:

- **Subagent `## Hylla Feedback` sections** in closing `till.comment`s on each build action_item — cascade-orch aggregates at cascade-end into a `kind=closeout` comment summary.
- **Tillsyn product / CLI / TUI / MCP refinements** → new `kind=refinement` action_items under the project (carry-forward queue for later cascades).
- **Hylla search-quality / ergonomics refinements** → new `kind=refinement` action_items, labeled `hylla` so they can be filtered.
- **Best-practice shifts** → direct edits to this wiki on the cascade branch.

### 2. Cross-Project Improvement Prompt (When Tillsyn Is Used Externally)

**When Tillsyn is being used by a project that is NOT this repo**, the adopting project's cascade-end closeout has one additional deliverable: **a prompt written to give back to Tillsyn itself** so the Tillsyn team can improve the runtime based on real external usage.

The prompt should capture:

- **Context** — what kind of project is using Tillsyn, what language stack, what team size, what role mix.
- **Friction** — the concrete moments during the cascade when Tillsyn got in the way: schema confusion, missing primitives, MCP call ergonomics, handoff/attention/comment semantics that didn't fit.
- **Workarounds** — what the adopting team did to route around the friction.
- **Requests** — ranked list of what would remove the friction in future Tillsyn releases.
- **Evidence** — pointers to specific cascades / comments / handoffs in the adopter's Tillsyn project that illustrate each friction point.

The adopting project files this prompt back to the Tillsyn team (via issue, PR, or `till.handoff` to a Tillsyn-team orchestrator identity, once that routing exists). **This is the primary feedback loop that keeps Tillsyn honest about external usability** — without it, we only see self-hosted dogfood signal, which overfits to the Tillsyn team's own habits.

Self-hosted dogfood cascades (i.e., cascades of the Tillsyn repo itself) skip step 2 — the findings from step 1 already flow into Tillsyn refinement action_items + this wiki directly.

## Orchestrator Role Boundaries

- **Orchestrator** (the parent Claude Code session) — plans, routes, delegates, cleans up. PREFERS cascade builder subagents for code changes (cascade enforces atomic-droplet sizing + plan-QA + asymmetric build-QA). May edit Go (or other) code directly when cascade adds overhead without value: trivial typo fixes, single-constant updates, mid-flight build-green stabilization, NIT-class absorptions surfaced by build-QA. Always edits markdown docs directly (this wiki, `CLAUDE.md`, `PLAN.md`, agent `.md` files) on the cascade branch.
- **Builder subagent** — the ONLY role that edits language code. Spawned via the `Agent` tool with Tillsyn auth credentials in the prompt.
- **QA subagents** — gated to `qa` role. Read, verify, verdict, die. Never edit code.
- **Planner subagent** — decomposes a cascade into atomic nested drops + droplets. Never edits code.
- **Dev / human** — approves **orchestrator** auth, reviews results, makes design calls that the orchestrator files as discussion action_items. Per the auth-approval cascade below, the dev does **not** approve non-orch subagent auth (planner / QA / builder / research).

External adopters: mirror this split even if you're using a single Claude session end-to-end — keeping "who is allowed to edit code" explicit makes QA gates meaningful instead of ceremonial.

## Auth Approval Cascade

**Dev approves orchestrator auth. Orchestrators approve their own non-orch subagent auth.**

The dev only ever sees orchestrator auth requests in the TUI. Planner / QA / builder / research auth is **provisioned and approved by the orch that spawns the subagent**, never by the dev. This keeps the dev's approval surface bounded to the active cascade orchestrators instead of fanning out to every short-lived subagent inside every cascade.

**Approval scope.** An orchestrator may approve a non-orch auth request when **all** of the following hold:

1. The request's `path` resolves to a node inside the orch's lease subtree.
2. The request's `principal_role` is **not** `orchestrator`. Orch-spawning-orch is out of scope; orch chains require dev approval at every step.
3. The orch claims the approval action through its own session tuple — no acting-on-behalf-of for approval.

**Capability landing.** Wave 3 of Drop 4a landed the orch-self-approves-non-orch-subagent capability. Orch-side approval is the canonical path; cross-orch and orch-spawning-orch still route through the dev TUI. Project-level `OrchSelfApprovalEnabled = *false` toggle is the total backstop (reverts ALL approves under that project to dev-TUI approval).

**Auth handoff to the subagent.** After the orch creates and approves the request, the orch passes `request_id` + `resume_token` + `path` + `principal_id` + `client_id` to the subagent in the spawn prompt — **never** the orch's own session tuple. The subagent runs `till.auth_request(operation=claim)` itself and issues its own scope-appropriate lease.

External adopters: this rule generalizes. Any orchestrator-shaped session that fans out to short-lived sub-sessions should provision + approve those sub-sessions itself — pushing every approval onto the human is the antipattern.

## Response Shape — Section 0 Semi-Formal Reasoning

**Canonical spec: `~/.claude/CLAUDE.md § "Semi-Formal Reasoning — Section 0 Response Shape"`** (global, mirrored across all projects). That section is the source of truth for the scaffold — adopter requirements, subagent pass-through, Tillsyn artifact boundary, bootstrap checklist. The project-level `CLAUDE.md § "Section 0 Response Shape"` enforces three load-bearing rules locally. This section is a quick-reference summary; read the canonical text before extending or adapting the shape.

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

## Cascade-End Closeout Checklist

Cascade-close is cascade-orch-owned end-to-end. The cascade root `kind=plan` action_item moves to `complete` only after every step below lands.

**Cascade-orch steps (pre-merge, on the cascade branch):**

1. All sibling droplets `complete`. `git status --porcelain` clean.
2. All commits on remote. CI green (`gh run watch --exit-status`).
3. Aggregate per-subagent `## Hylla Feedback` sections from build action_item closing comments into a single `till.comment` on the cascade root (or on a dedicated `kind=closeout` child).
4. Surface cascade refinements as new `kind=refinement` action_items under the project (Tillsyn product / CLI / TUI / MCP) — labeled `hylla` when the candidate is Hylla-specific.
5. If this is an external adopter: write the cross-project improvement prompt to a `kind=research` action_item under the project and route it to the Tillsyn team via `till.handoff` or external issue/PR.
6. `hylla_ingest` — full enrichment, from the GitHub remote, after CI green (Go projects only; Hylla indexes Go today).
7. If any best-practice shifted, edit `WIKI.md` in place on the cascade branch.
8. Rebase onto `origin/main`, resolve conflicts (Go via builder subagent using `git diff`; MDs directly). `mage ci` green locally.
9. Force-push. CI green. Open PR. Dev-approved merge.
10. Post-merge: delete remote + local branch refs. Move the cascade root action_item to `complete`. `git worktree remove cascade/N` from `main/` (never from inside the worktree being removed).

## Related Files

- `CLAUDE.md` — canonical project rules + Hard Rules. Auto-loaded on every session start.
- `PLAN.md` — cascade architecture and drop ordering. Source of truth for the cascade build.
- `AGENT_CASCADE_DESIGN.md` — cascade-agent role definitions and spawn pipeline.
- `CASCADE_METHODOLOGY.md` — long-form methodology doc (plan-down / build-up, atomicity-via-planner-prompt).
- `CLI_ADAPTER_AUTHORING.md` — guide for adding a new CLI adapter (codex, ollama-bridge, etc.) under the dispatcher's `CLIAdapter` interface.
- `CONTRIBUTING.md` — dev environment + MCP setup. Sections managed via `ta` MCP (`.ta/schema.toml` registers the `contributing` db).
- `AGENTS.md` / `AGENTS_CONFIG.md` — agent role definitions + configuration shape.
- `GDD_METHODOLOGY.md` — game-design-doc methodology placeholder; populated post-dogfood.
- `workflow/` (git-tracked under `main/workflow/`) — per-drop atomic-small-things MDs. `workflow/drop_N/` is the on-disk historical audit for drops closed under the pre-Tillsyn-native scaffolding (Drop 2 and earlier). `workflow/example/drops/_TEMPLATE/` + `workflow/example/CLAUDE.md` are the **adopter onramp** for projects standing up Tillsyn from scratch — the MD-bridge pattern adopters use until they install Tillsyn itself.
- `README.md` — public-facing project overview.
