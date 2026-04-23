# Tillsyn Minions: Architectural Plan

**Date:** 2026-04-13
**Status:** Active design. Research complete. No code yet. Not tracked in Tillsyn yet.
**Scope:** Feature addition to Tillsyn — state-triggered autonomous agent dispatch with hierarchical planning cascade.

---

## Table of Contents

1. [Naming and Terminology](#1-naming-and-terminology)
2. [Hierarchy Refactor](#2-hierarchy-refactor)
3. [The Cascade Model](#3-the-cascade-model)
4. [Dispatch Mechanism](#4-dispatch-mechanism)
5. [File- and Package-Level Blocking](#5-file--and-package-level-blocking)
6. [Template Configuration](#6-template-configuration)
7. [Agent Types and Model Assignment](#7-agent-types-and-model-assignment)
8. [Auth and Lifecycle](#8-auth-and-lifecycle)
9. [Gates, Commits, and Deterministic Steps](#9-gates-commits-and-deterministic-steps)
10. [Trust Model](#10-trust-model)
11. [Semi-Formal Reasoning Integration](#11-semi-formal-reasoning-integration)
12. [Concurrency Model](#12-concurrency-model)
13. [Hylla Integration](#13-hylla-integration)
14. [Escalation and Retry Policy](#14-escalation-and-retry-policy)
15. [Wiki / Ledger System](#15-wiki--ledger-system)
16. [Quality and Vulnerability Checking](#16-quality-and-vulnerability-checking)
17. [Prerequisites](#17-prerequisites)
18. [Pre-Build Preparation](#18-pre-build-preparation)
19. [Development Order](#19-development-order)
20. [Open Questions](#20-open-questions)
21. [Resources](#21-resources)
22. [Account Tier, Auth, and ToS Posture](#22-account-tier-auth-and-tos-posture)
23. [Mention Routing](#23-mention-routing)
24. [File Viewer (TUI) with charmbracelet/glamour](#24-file-viewer-tui-with-charmbrachelet-glamour)

---

## 1. Naming and Terminology

### 1.1. Finalized: "Agent"

The dispatched one-shot unit is called an **agent**. The term is already used throughout this codebase and the rollout (`go-builder-agent`, `go-qa-proof-agent`, subagent types), so introducing a new umbrella word would create more confusion than it solves. Agents are typed by role (planner, builder, QA-proof, QA-falsification, commit). The umbrella "agent" covers all of them. No new name.

- "The cascade dispatches a planning agent" — reads naturally
- "Build agent failed, QA agent passed" — reads naturally
- "The agent's auth was revoked" — reads naturally

### 1.2. The Cascade (Confirmed)

The execution pattern is a **cascade**: **design down, build up.** Planning decomposes downward through levels; completion propagates upward. State changes trigger the next step.

- **Cascade run** — a single top-to-bottom-to-top execution, starting from a drop moving to `in_progress`
- **Cascade tree** — the tree of action items produced by a cascade run

### 1.3. Glossary

| Term | Definition |
|---|---|
| **agent** | A one-shot autonomous unit dispatched by Tillsyn. Receives auth + action-item context, does work via MCP, moves its action item to `complete`/`failed`, dies. Typed by role: planner, builder, QA, commit. |
| **cascade** | The hierarchical execution pattern: planning decomposes downward ("design down"), completion propagates upward ("build up"). State changes trigger the next step. |
| **cascade run** | A single execution of a cascade, from drop `in_progress` through completion or failure. |
| **cascade tree** | The tree of action items produced by a cascade run. |
| **dispatcher** | The Tillsyn subsystem that watches for state changes and spawns agent processes. Not an agent — purely programmatic (except commit message formation, which uses a lightweight haiku agent). |
| **gate** | A deterministic verification step (e.g., `mage ci`) that runs programmatically after an agent completes. No LLM involved. |
| **Drop** *(workflow unit — capitalized in prose)* | A release/lane coordination unit (`Drop 1.75`, `Drop 2`, etc.) — one PR, one drop-orch, one branch. NOT an `action_items.kind` value post-Drop-1.75; the in-tree grouping primitive is the `plan` kind. See §1.4 for the kind vocabulary and §2.2 for the tree shape. |
| **action item** | A row in the `action_items` table. `kind` is a closed 12-value enum (`plan`, `research`, `build`, `plan-qa-proof`, `plan-qa-falsification`, `build-qa-proof`, `build-qa-falsification`, `closeout`, `commit`, `refinement`, `discussion`, `human-verify`). `plan` is the grouping primitive — nests infinitely and contains `build` / `research` / sub-`plan` children plus auto-created QA twins. `build` is a leaf with `build-qa-*` children. |

### 1.4. Cascade Addressing Vocabulary (drop 0 Convergence)

Converged during drop 0 closeout discussions. Pre-cascade these are conceptual; post-drop-4 the dispatcher materializes them.

**Workflow-drop nesting + naming:**

- **Drops all the way down at the workflow level.** The `project` is NOT a workflow drop — it is the root container. Immediate children of the project are **top-level Drops** (Drop 0, Drop 1, …). In the action-items tree this is the root `plan` node for that workflow drop.
- Workflow Drops may contain sub-`plan`s (recursive decomposition) and `build` leaves. Every `plan` nests infinitely when the work needs deeper breakdown.
- Dotted addresses (below) let readers reference any node in the tree by position regardless of how deep nesting goes.

**Dotted addresses (read-only shorthand):**

- Top-level Drop `N` is addressed as `N` (e.g. `0` is Drop 0).
- Sub-node `M` of node `N` is `N.M` (e.g. `0.1` is Drop 0's second sub-item).
- `0.1.5.2` = project's Drop 0 → its sub-item_1 → that sub-item's sub-item_5 → that sub-item's sub-item_2.
- Project-qualified form: `<proj_name>-<dotted>` (e.g. `tillsyn-0.1.5.2`) for unambiguous cross-project references.
- **Dotted addresses are read-only.** For mutations, always use the UUID action-item id. Dotted addresses are unstable under re-parenting and should never be load-bearing in scripts.

**`action_items.kind` — closed 12-value enum (post-Drop-1.75 kind catalog):**

Drop 1.75 collapses the old `kind_catalog` (16+ mixed kinds + `project`) into two node types — `projects` (table, no kind column) and `action_items` (table, 12-value `kind` enum). The kind axis is what templates bind `child_rules`, gate rules, and agent bindings against. No fallback kind — every `action_items` row sets `kind` explicitly at creation.

| Table          | Kind                     | Purpose                                                                                                                                |
| -------------- | ------------------------ | -------------------------------------------------------------------------------------------------------------------------------------- |
| `projects`     | _(no kind column)_       | Root container per project. Project-kind info (language/stack) lives in `projects.metadata`, not as a catalog value.                   |
| `action_items` | `plan`                   | Planning-dominant — decomposes work into children. The grouping primitive; nests infinitely. Auto-creates `plan-qa-proof` + `plan-qa-falsification` children. |
| `action_items` | `research`               | Read-only investigation. Standalone — no auto-QA children. Posts findings in a closing comment and dies.                                |
| `action_items` | `build`                  | Code-changing leaf. Auto-creates `build-qa-proof` + `build-qa-falsification` children. Cannot contain further children.                |
| `action_items` | `plan-qa-proof`          | Proof-completeness QA pass on a `plan` parent. `blocked_by: <plan-parent>`.                                                            |
| `action_items` | `plan-qa-falsification`  | Falsification QA pass on a `plan` parent. `blocked_by: <plan-parent>`.                                                                 |
| `action_items` | `build-qa-proof`         | Proof-completeness QA pass on a `build` parent. `blocked_by: <build-parent>` + post-build gates (mage ci).                             |
| `action_items` | `build-qa-falsification` | Falsification QA pass on a `build` parent. Same blockers as `build-qa-proof`.                                                          |
| `action_items` | `closeout`               | Drop-end coordination — ledger update, Hylla reingest, WIKI_CHANGELOG line. Orchestrator-managed. `blocked_by: <every-other-drop-item>`. |
| `action_items` | `commit`                 | Commit action. Template-triggered under `plan` at level ≥ 2 with a `complete` `build` descendant (see Commit Cadence below). Post-Drop-4. |
| `action_items` | `refinement`             | Perpetual / long-lived tracking umbrella — drop-end entries roll up here. Persistent STEWARD-owned drops (REFINEMENTS, HYLLA_REFINEMENTS) use this kind. |
| `action_items` | `discussion`             | Cross-cutting decision park — description holds converged shape, comments hold audit trail. Pre-cascade discussion happens in chat (see `CLAUDE.md` §"Discussion Mode"); this kind formalizes the artifact. |
| `action_items` | `human-verify`           | Dev sign-off hold point — attention items + checklist children, NO plan/QA children. Used for `DROP N START — PLANNING CONFIRMATION WITH DEV` and `DROP N END — REVIEW DONE + CORRECT`. |

**Customization (future drop — NOT Drop 1.75).** Projects and orchestrators author custom kinds via template that attach as sub-action-items of specific generic kinds. Examples: a custom `ledger-update` under `closeout`; a custom `ledger-aggregation` or `refinement-entry` under `refinement`. Drop 1.75 ships only the 12 generics plus the template hook. Project templates will also carry an allowed-kinds enum + a `disallow_generics` bool to restrict which kinds a project accepts.

**Commit Cadence (Drop 2 discussion seed — see §19.2).** Current commit-every-build-task rate is too aggressive. Proposed default: auto-generate a `commit` child under any `plan` at level ≥ 2 whose subtree contains at least one `build` in `complete` state, fired at plan close-time. Templates can set `auto_commit = false` per-kind for local-only outputs (SQL migration scripts never committed, scratch work). Edge cases to resolve in the Drop 2 discussion: multi-PR drops (one commit per plan vs one per Drop), cascade-owned vs dev-owned commit, partial-success subtrees.

**Workflow-Drop ↔ `plan` action-item bridge.** Workflow Drops (Drop 0 / Drop 1 / Drop 1.75 / …) materialize as a root `plan` per-Drop in the action_items tree: that root `plan` holds the Drop's human-verify start, planning children, build children, sub-`plan`s, closeout, and human-verify end. A Drop's numbered name (`DROP 1.75`) is stable and human-facing; its dotted address (`1.75`, `1.75.0`, `<proj>-1.75.0.2`) is position-derived and unstable under re-parenting. Use named form in prose and docs; UUIDs for all mutations; dotted form only for quick cross-references in chat. Post-Drop-2 the resolver (§19.2) accepts either UUID or dotted form on read paths.

**Adhoc vs. template-generated action items:**

- **Template-generated** — a `plan`'s `child_rules` create sibling `plan-qa-proof` and `plan-qa-falsification` automatically (Drop 3+). Same for `build` → `build-qa-*`. This is the cascade's default flow.
- **Adhoc** — a `refinement` or `discussion` action item created manually by the orchestrator outside any cascade flow, typically because the work is cross-cutting or long-lived. Per drop 0 5.2 decision: persistent refinement trees (`REFINEMENTS.MD`, `HYLLA_REFINEMENTS.MD`) use `kind: refinement` + adhoc creation pre-Drop-3, and existing items get updated in place rather than re-created.

---

## 2. Hierarchy Refactor

### 2.1. Pre-Drop-1.75 State — Confused Primitives (Being Collapsed Now)

Pre-Drop-1.75 Tillsyn had: `project > branch > phase > actionItem > subtask` in `kind_catalog`, plus `build-actionItem`, `qa-check`, `plan-actionItem`, `commit-and-reingest`, `a11y-check`, `visual-qa`, `design-review`, `decision`, `note`. In practice:

- **Branch** existed to map to git worktrees. With file-level gating (Section 5), worktrees are unnecessary.
- **Phase** and **branch** were used interchangeably and inconsistently.
- The 16+ kind_catalog values carried no orthogonal structural meaning — most were role-flavored `actionItem` variants. The `plan-actionItem`/`build-actionItem`/`qa-check` split conflated "what role does this agent play?" with "where does this node sit in the tree?"
- **`depends_on`** overlaps with the parent-child hierarchy (children must complete before parent = implicit depends_on).
- **`done`** should be `complete` — more descriptive, clearer intent.

**Drop 1.75 is the collapse drop.** `kind_catalog` becomes a closed 12-value enum on `action_items.kind`; `projects.kind` column is dropped entirely; `scope` is mirrored from `kind` for this drop (column removal deferred to a future refinement). Role moves from kind to description prose (pre-Drop-2) and to `metadata.role` (post-Drop-2).

### 2.2. Proposed Hierarchy (Post-Drop-1.75 Target)

Two node tables. Tree uses `action_items.kind` on every non-project node. `plan` is the grouping primitive (the old `drop` kind collapses into it).

```
project                                                          (table: projects — no kind column)
  └── plan (root plan — one per workflow Drop; nests infinitely) kind: plan
        ├── human-verify (DROP N START — PLANNING CONFIRMATION)  kind: human-verify    (dev-gated, first child)
        ├── human-verify (DROP N START — REFINEMENT REVIEW)      kind: human-verify    (sibling of start confirmation)
        ├── plan-qa-proof                                        kind: plan-qa-proof   (auto-created child of this plan)
        ├── plan-qa-falsification                                kind: plan-qa-falsification
        ├── research  (optional, adhoc)                          kind: research
        ├── build  (leaf work)                                   kind: build
        │     ├── build-qa-proof                                 kind: build-qa-proof
        │     └── build-qa-falsification                         kind: build-qa-falsification
        ├── plan  (sub-plan — infinite nesting)                  kind: plan
        │     └── ... same structure with its own start/end human-verify bracketing ...
        ├── discussion (cross-cutting decision park)             kind: discussion
        ├── refinement (adhoc perpetual entries)                 kind: refinement
        ├── commit (template-triggered, post-Drop-4)             kind: commit
        ├── DROP_N_REFINEMENTS_GATE_BEFORE_DROP_N+1              kind: refinement       (STEWARD-owned, §15.8)
        ├── closeout (DROP N END — LEDGER UPDATE)                kind: closeout         (blocked_by every other drop item)
        └── human-verify (DROP N END — REVIEW DONE + CORRECT)    kind: human-verify     (dev-gated, blocked_by closeout + refinements-gate)
```

**Rules:**

- **Every workflow Drop starts with a `human-verify` (`DROP N START — PLANNING CONFIRMATION WITH DEV`)** — dev-gated first child that captures sign-off on scope, plan, and agent/system-prompt direction before any planning agent fires. Absorbs the existing `DROP N START — REFINEMENT REVIEW` as a sibling `human-verify` inside the bracket — the refinement review feeds the planning confirmation, they happen together at Drop start.
- **Every workflow Drop ends with a `human-verify` (`DROP N END — REVIEW DONE + CORRECT`)** — dev-gated sign-off that all work landed correctly. `blocked_by` both the `closeout` (`DROP N END — LEDGER UPDATE`, so the ledger entry is in place) AND `DROP_N_REFINEMENTS_GATE_BEFORE_DROP_N+1` (so STEWARD's post-merge MD writes + refinements-gate discussion have completed — see §15.7–§15.8). Nothing in the Drop moves to `complete` until this final `human-verify` is signed off.
- **Every `plan`** auto-creates `plan-qa-proof` + `plan-qa-falsification` children (blocked_by the `plan`, firing in parallel after plan completes). `plan` contains either `build` leaves, sub-`plan`s, or both, bracketed by the start/end `human-verify` items.
- **Every `build`** auto-creates `build-qa-proof` + `build-qa-falsification` children. `build` is leaf-level — no nesting beyond one level of QA checks.
- **`plan` nests infinitely.** A planner at one level creates sub-`plan`s if the work needs further decomposition, or `build`s if the work is granular enough. Sub-`plan`s carry their own start/end `human-verify` bracketing — the pattern is recursive, not root-only.
- **`human-verify`** does NOT carry `plan`/QA children — it is a dev-gated hold point whose internal structure is attention item(s) + checklist action-item(s). No planner fires for it.
- **`research`, `discussion`, `refinement`, `closeout`, `commit`** do NOT auto-create QA children. Review is via comment thread (`research`), description convergence (`discussion`), orchestrator-managed aggregation (`closeout`, `refinement`), or template-triggered post-build gating (`commit`).
- A planner can create a `build` directly (small enough, no further decomposition needed) — it just creates the `build` with its QA children.

**Why bracketing, not just a wrap-up note.** The prior "per-drop wrap-up" was documentation-level — nothing structurally prevented a Drop from completing without dev sign-off. The start/end `human-verify` bracketing makes dev sign-off a real `blocked_by` edge the dispatcher enforces. It is the primary hook for the Discussion Mode rule in `CLAUDE.md` ("cross-cutting decisions happen in chat, converged shape lands on the action item") — the start `human-verify` is where those chats are scheduled, the end `human-verify` is where they're validated.

### 2.3. Remove `branch`

`branch` was a primitive for git worktree mapping. With file-level gating (Section 5), worktrees are unnecessary. Branches add a hierarchy level that creates confusion with phases.

**Action:** Remove `branch` from the kind catalog. Pre-existing `branch` nodes are rewritten to `plan` by `main/scripts/drops-rewrite.sql` (Drop 1.75). Remove from schema, domain, TUI, templates.

### 2.4. Remove `phase` and `branch` — Use `plan` as the Grouping Primitive

"Phase" implied temporal ordering (phase 1, phase 2). "Branch" was a worktree-mapping holdover. Neither survives Drop 1.75: both collapse into `plan`, which is the natural grouping primitive because a `plan` is where decomposition decisions land (agent: planner) and it's where the cascade's `blocked_by` edges are set. Siblings can run in parallel when paths/packages don't overlap — no temporal-ordering word needed in the schema.

**Action:** Drop 1.75 rewrites all pre-existing `phase` and `branch` nodes to `plan` via `main/scripts/drops-rewrite.sql`. Remove `phase` and `branch` from schema, domain, TUI, templates, documentation. The word "Drop" survives as workflow-level prose ("Drop 1.75") but is NOT an `action_items.kind` value.

### 2.5. Rename `done` → `complete`

`complete` is more descriptive. "This action item is complete" reads better than "this action item is done." Aligns with `completion_contract`, `completion_notes`, `CompletedAt`.

**Lifecycle states:** `todo` → `in_progress` → `complete` | `failed`

**Action:** Rename in schema (DB column values), domain constants (`StateComplete`), TUI labels, MCP adapter normalization, templates, documentation. This touches the same surfaces as the `failed` state addition — combine the migration.

### 2.6. Simplify `depends_on` and `blocked_by`

| Mechanism | Keep? | Rationale |
|---|---|---|
| **Parent-child** | **Yes** | Core hierarchy. `require_children_done` enforces completion ordering. |
| **`blocked_by`** | **Yes** | For sibling ordering (drop-2 blocked_by drop-1) and cross-branch blocking (file-level conflicts). Essential for cascade scheduling. |
| **`depends_on`** | **Remove (last drop)** | Redundant with parent-child + `blocked_by`. Planned ordering is `blocked_by` set at creation time. Remove in final dogfooding drop as a real test of the cascade system. |

**Action:** Keep `depends_on` functional during build. Remove in the final cleanup drop during dogfooding. This serves as a good integration test of the cascade.

### 2.7. Incremental Migration Strategy

This refactor touches schema, domain, app, adapters, TUI, templates, and documentation. It must be done incrementally:

1. Each rename/removal is a small, testable increment
2. After each increment: `mage ci`, confirm no orphaned code, confirm nothing that worked before is broken. Hylla reingest is drop-end only (§9.7), not per-increment.
3. Existing data migrations for DB schema changes
4. Template updates accompany each schema change

**This is the touchiest part of the build.** Every change ripples through 5+ packages. Plan each increment carefully.

---

## 3. The Cascade Model

### 3.1. Plan Down, Build Up

The cascade is recursive. **Design down:** planning decomposes work into smaller pieces, level by level. **Build up:** execution happens at the leaf level, and completion propagates upward.

At every level, the pattern is:

1. **Plan** — a planner agent decomposes the work into children
2. **Plan QA** — two QA agents verify the plan (proof + falsification)
3. **Execute** — children fire (which may themselves be cascades or leaf-level builds)
4. **Verify** — gates and QA verify the execution
5. **Complete** — completion propagates upward to parent

### 3.2. ASCII Art — Full Cascade

```
ROOT PLAN: "Add failed lifecycle state" ← kind: plan ← human/orchestrator moves to in_progress
│
│  ┌──────────────────────────────────────────────────────────────┐
│  │ TEMPLATE AUTO-CREATES (on plan creation):                    │
│  │  • PLAN-QA-PROOF           (kind: plan-qa-proof)             │
│  │  • PLAN-QA-FALSIFICATION   (kind: plan-qa-falsification)     │
│  │ (No separate planner action item — the `plan` kind IS the    │
│  │  planner's work unit. The planner agent runs against THIS    │
│  │  action item.)                                               │
│  └──────────────────────────────────────────────────────────────┘
│
├── ROOT PLAN runs the planner → go-planning-agent (opus, high effort)
│   │  Work: reads plan goal, decomposes into sub-`plan`s or `build`s,
│   │        fills out scope/context/affected areas for each,
│   │        sets blocked_by between sequential children,
│   │        sets paths[] + packages[] on each child for file-/package-level gating
│   │  Output: creates SUB-PLAN-1, SUB-PLAN-2 (kind: plan) and/or
│   │          BUILD-1, BUILD-2 (kind: build) as children of ROOT PLAN
│   │          via till.action_item(operation=create)
│   │  Terminal: moves ROOT PLAN to complete → auth revoked → killed
│   │
│   ├── PLAN-QA-PROOF ← kind: plan-qa-proof ← blocked_by: ROOT PLAN
│   │   Agent: go-qa-proof-agent (opus, high effort)
│   │   Checks: plan completeness, evidence grounding, consistency
│   │   PASS → complete │ FAIL → failed + comment
│   │
│   └── PLAN-QA-FALSIFICATION ← kind: plan-qa-falsification ← blocked_by: ROOT PLAN
│       Agent: go-qa-falsification-agent (opus, high effort)
│       Checks: vagueness, missing cases, incorrect assumptions, YAGNI,
│               FILE-/PACKAGE-LEVEL CONFLICTS (must verify no two siblings share
│               paths[] or packages[] without explicit blocked_by between them!)
│       PASS → complete │ FAIL → failed + comment
│
│   ┌──────────────────────────────────────────────────────────────┐
│   │ ALL PLAN QA PASS → child plans / builds become eligible      │
│   │                                                              │
│   │ ANY PLAN QA FAIL → failed + comment with findings            │
│   │   try 1 of max-tries=2:                                      │
│   │     ROOT PLAN re-runs planner with failure context           │
│   │     plan QA runs again on the revised plan                   │
│   │   try 2 fails → attention item to orchestrator AND human     │
│   │     full stop until human intervenes                         │
│   └──────────────────────────────────────────────────────────────┘
│
├── SUB-PLAN-1 ← kind: plan ← no blocked_by → auto in_progress when plan QA passes
│   │
│   │  ┌─ TEMPLATE AUTO-CREATES plan-qa-proof + plan-qa-falsification children ─┐
│   │
│   ├── SUB-PLAN-1 runs its planner → go-planning-agent (opus)
│   │   │  Work: decomposes sub-plan into 1-4 granular builds
│   │   │        fills out: description, paths[], packages[],
│   │   │        acceptance_criteria, test targets
│   │   │        sets blocked_by for builds sharing files OR packages
│   │   │  Output: creates BUILD-1, BUILD-2 (kind: build) as children
│   │   │
│   │   ├── PLAN-QA-PROOF (kind: plan-qa-proof)
│   │   └── PLAN-QA-FALSIFICATION (kind: plan-qa-falsification)
│   │
│   │   ON PLAN QA PASS → builds eligible:
│   │
│   ├── BUILD-1 ← kind: build ← no blocked_by → auto in_progress
│   │   │  Agent: go-builder-agent (sonnet, standard effort)
│   │   │  File gating: can only edit files listed in action item paths[]
│   │   │  Pre-check: system confirms assigned files have clean git status
│   │   │  Work: implements code, runs mage test-func on affected funcs
│   │   │  max-tries=2 (builder can retry once on test failure)
│   │   │  Terminal: moves to complete via MCP → auth revoked → killed
│   │   │
│   │   │  ┌─ ON BUILDER COMPLETE: ─────────────────────────────────┐
│   │   │  │ GATE: mage ci (deterministic, system runs it)          │
│   │   │  │ Gate pass → QA fires (proof + falsification parallel)  │
│   │   │  │ Gate fail → action_item moves to failed + gate comment │
│   │   │  │   → try 1 of max-tries: new builder fires              │
│   │   │  │   → try 2 fails: escalate (re-plan or human)           │
│   │   │  └────────────────────────────────────────────────────────┘
│   │   │
│   │   ├── BUILD-QA-PROOF ← kind: build-qa-proof ← fires after mage ci passes
│   │   │   Agent: go-qa-proof-agent (sonnet, medium effort)
│   │   │   Checks: evidence completeness, reasoning, trace coverage
│   │   │
│   │   └── BUILD-QA-FALSIFICATION ← kind: build-qa-falsification ← fires in parallel
│   │       Agent: go-qa-falsification-agent (sonnet, medium effort)
│   │       Checks: counterexamples, hidden deps, contract mismatches
│   │
│   │   ALL QA PASS → commit-cadence check (§CLAUDE.md Commit Cadence):
│   │                 IF parent plan at level ≥ 2 with ≥1 build complete →
│   │                   template auto-creates COMMIT (kind: commit)
│   │                 commit agent (haiku) forms message
│   │                 → system commits + optionally pushes (template cfg)
│   │                 → gh run watch --exit-status until CI green
│   │                 → BUILD-1 complete
│   │                 (NO per-build hylla reingest — ingest is Drop-end only,
│   │                  owned by the Drop's closeout action item; see §15.7)
│   │   ANY QA FAIL → BUILD-1 failed → attention item
│   │     QA never retries. QA failure → escalation or human.
│   │
│   ├── BUILD-2 ← kind: build ← blocked_by: BUILD-1 (if shares files/packages)
│   │   │                       OR no blocked_by (if disjoint → parallel)
│   │   └── ... same build+gate+commit+QA flow ...
│   │
│   │  ┌─ SUB-PLAN COMPLETION CHECK: ──────────────────────────┐
│   │  │ System checks for uncommitted/unpushed changes         │
│   │  │ If any found → attention item to orchestrator          │
│   │  └────────────────────────────────────────────────────────┘
│   │
│   └── ALL BUILDS complete → SUB-PLAN-1 complete
│
├── SUB-PLAN-2 ← kind: plan ← blocked_by: SUB-PLAN-1 (or parallel if disjoint)
│   │  Auto in_progress when SUB-PLAN-1 completes
│   │  Same cascade flow
│   └── ...
│
├── CLOSEOUT ← kind: closeout ← blocked_by: every other ROOT PLAN child
│   │  Orchestrator-managed. Aggregates ledger / refinements / Hylla findings.
│   │  Triggers Hylla reingest at Drop end (full_enrichment, from remote).
│
└── ALL CHILDREN complete → ROOT PLAN complete
```

### 3.3. Key Properties

**Design down (decomposition):**
- drop → Sub-drops → Build tasks
- At each level, a planner agent does the decomposition
- Planning QA verifies the decomposition before execution proceeds
- Planner must explicitly set file paths and `blocked_by` for file-level conflicts
- Plan QA falsification specifically checks for missing file-level blockers
- The planner creates child action items via `till.action_item(operation=create)`
- Template `child_rules` auto-create QA children for each created item

**Build up (completion):**
- `build` action item complete → sub-drop checks all children → sub-drop complete → drop checks → complete
- Uses `require_children_done` — parent can't complete until all children complete
- No `depends_on` needed for parent-child — the hierarchy IS the dependency
- `blocked_by` is for siblings and file-level conflicts

**Parallel execution (natural concurrency):**
- When a parent moves to `in_progress`, ALL children without blockers auto-fire simultaneously
- BUILD-TASK-1 and BUILD-TASK-2 fire in parallel if they don't share files
- QA-PROOF and QA-FALSIFICATION always fire in parallel
- No parallelism config — just absence of `blocked_by`. If it's not blocked, it fires.

### 3.4. `blocked_by` — The Only Sibling Ordering Primitive

With the hierarchy refactor (Section 2.6), `depends_on` is marked for removal in the final dogfooding drop. Until then, both exist. After removal:

| Mechanism | What It Means | Where It Applies |
|---|---|---|
| **Parent-child hierarchy** | Child must complete before parent can complete | Built into cascade. Uses `require_children_done`. |
| **`blocked_by`** | Sibling or cross-drop item must be `complete` before this item can move to `in_progress` | Sequential drops, file-level conflicts between tasks, cross-drop dependencies. |

The planner sets `blocked_by` at creation time for planned ordering. Runtime discoveries (unexpected file conflicts detected by the dispatcher) add `blocked_by` dynamically.

---

## 4. Dispatch Mechanism

### 4.1. How It Works

The dispatcher is a **programmatic subsystem inside Tillsyn**, not a separate process or CLI command. It watches for lifecycle state transitions and spawns agent processes.

```
State change detected: action_item moved to in_progress
  │
  ├── Does the item's kind have an agent binding in the template?
  │   NO → nothing happens (manual work, or deterministic gate)
  │   YES ↓
  │
  ├── FILE-LEVEL PRE-CHECK (for builders only):
  │   ├── Are all assigned files clean in git status?
  │   │   NO → block dispatch, post comment with dirty-file list,
  │   │        attention item to orchestrator
  │   │   YES ↓
  │   ├── Is any assigned file currently claimed by another active agent?
  │   │   YES → add dynamic blocked_by, defer dispatch
  │   │   NO ↓
  │   └── Claim file locks for this agent
  │
  ├── Issue auth session for the agent (role from template, no human approval)
  │
  ├── Resolve project directory path (from project metadata)
  │
  ├── Spawn: cd <project_dir> && claude --agent <agent_type> \
  │     --bare -p "<task_context>" \
  │     --permission-mode acceptEdits \
  │     --max-budget-usd <from_template> \
  │     --max-turns <from_template> \
  │     --output-format stream-json \
  │     --mcp-config <agent_mcp_config>
  │
  ├── Monitor: subscribe to LiveWaitBroker for this action item's state changes
  │
  └── On state change to complete|failed:
      ├── Auth auto-revoked (state-machine hook in MoveActionItem)
      ├── Release file locks held by this agent
      ├── Kill the claude CLI process
      ├── Cascade: check for newly-eligible children/siblings
      └── Fire next eligible items
```

### 4.2. No `till run` Command

There is no `till run` CLI command. The dispatch is triggered by state changes. The state change can come from:

- **TUI** — human moves a action item to `in_progress`
- **MCP** — orchestrator calls `till.action_item(operation=move_state, state=in_progress)`
- **Dispatcher itself** — when a blocker clears, the dispatcher auto-moves eligible items to `in_progress`

The dispatcher is always running as part of the Tillsyn process (serve, serve-mcp, or TUI). It subscribes to `LiveWaitBroker` state-change events.

### 4.3. Auto-Promotion of Eligible Items

When any action item moves to `complete`:
1. Dispatcher checks all items that have `blocked_by` pointing to the completed item
2. For each: if ALL blockers are now `complete`, auto-move to `in_progress`
3. The `in_progress` transition triggers dispatch (4.1)

This creates the natural concurrency: items without blockers fire immediately when their parent moves to `in_progress`.

### 4.4. Project Directory Resolution

Tillsyn already has path logic in the project creation / bootstrap flow (TUI). This needs to be:

1. **Refactored** from TUI to the backend (`internal/app/` or `internal/domain/`) so it's reusable
2. **Stored** in project metadata (a `project_dir` field on the project record)
3. **Used** by the dispatcher to `cd` before spawning claude

**Early TODO:** Audit the existing path logic in TUI bootstrap flow. Refactor to backend. Add `project_dir` to project metadata if not already present.

### 4.5. Agent Process Lifecycle

```
Dispatcher spawns claude CLI process
  │
  ├── Agent boots, claims auth via till.auth_request(operation=claim)
  │   (auth was pre-created by dispatcher, no human approval needed)
  │
  ├── Agent reads action-item details via action_item(get) — its working brief
  │
  ├── Agent does work:
  │   ├── Planner: creates child action items via MCP
  │   ├── Builder: edits files (gated to allowed paths), runs mage test-func
  │   └── QA: reads code, verifies, writes certificate
  │
  ├── Agent calls till.action_item(operation=move_state, state=complete|failed)
  │   ├── Includes metadata.outcome, completion notes, comments
  │   └── This is the terminal MCP call
  │
  ├── MoveActionItem fires:
  │   ├── Auth auto-revoked
  │   ├── LiveWaitBroker publishes state-change event
  │   └── `require_children_done` checks fire (children blocking parent)
  │
  └── Dispatcher receives LiveWait event:
      ├── Kills claude CLI process (cleanup)
      ├── Releases file locks
      ├── Checks for newly-eligible items
      └── Fires next cascade step
```

**Key:** The agent interacts with Tillsyn directly via MCP tools. No JSON parsing of agent output. No structured response format from the agent. The MCP tool calls ARE the structured output. The system knows what happened because the agent moved its own state.

---

## 5. File- and Package-Level Blocking

### 5.1. The Problem

Multiple build agents may run concurrently in the same project directory. Without coordination, two agents editing the same file will produce merge conflicts, stale reads, or silent overwrites. A subtler problem: two agents editing **different files in the same Go package** can break the package for everyone — changes to one file in `internal/domain` can cause compilation errors in the other 24 files of that package, which breaks the other agent's `mage test-pkg` or `mage test-func` mid-run. File-level gating is necessary but not sufficient.

### 5.2. One Builder Per Package and Per File

Two levels of mutual exclusion:

- **File-level write lock.** Only one active build agent holds the write lock on any file at a time.
- **Package-level build lock.** Only one active build agent holds the "may break this package's compile" token at a time. The agent that holds the package lock is allowed to add/remove symbols, change signatures, or otherwise invalidate the package's API while it works. Other agents editing different packages proceed normally. An agent that only needs to **read** a locked package to call its API is fine — the lock is on writes, not reads.

Examples why this matters:
- `internal/domain` is ~25 files in one Go package. Two agents each editing one file still share a compile. If agent A renames a type, agent B's test compile fails.
- `internal/adapters/server/common` has multiple MCP adapter files sharing private helpers. Same problem.

The planner sets **both** `paths` (the specific files) and `packages` (the Go packages those files belong to) on each `build` action item. The dispatcher enforces mutual exclusion:

1. Before dispatching a builder, check all assigned files against active file locks AND all assigned packages against active package locks.
2. If any file or package is locked by another active agent → add dynamic `blocked_by`, defer dispatch.
3. If all files and packages are free → acquire both sets of locks, dispatch.

### 5.3. Planner Responsibility

The planner **must** set file paths and package paths on every `build` action item it creates. This is not optional — it's how the cascade prevents file and package conflicts.

Plan QA falsification **specifically checks** for:
- Action items with missing or incomplete `paths`
- Action items with missing or inconsistent `packages` (packages must cover every file in `paths`)
- Two sibling tasks sharing a file OR a package without an explicit `blocked_by` between them
- File paths that don't match the drop's stated scope

If the planner misses a path or a package, the dispatcher catches it at dispatch time via git status checks (Section 5.4) and the package-lock check, but relying on runtime detection is the fallback, not the plan.

### 5.4. Git Status Pre-Check

Before a builder agent starts, the dispatcher confirms:

```
git status --porcelain -- <file1> <file2> ...
```

If any assigned files have uncommitted changes (dirty git status):
- **Block dispatch** — do not start the builder
- Post a comment on the action item listing the dirty files
- Fire an attention item to the orchestrator
- The orchestrator or human must resolve the dirty state before the builder can proceed

This catches two problems:
1. Files dirtied by a previous agent that crashed before commit
2. Files manually edited by the human outside the cascade

### 5.5. Dispatcher Auto-Detection of Conflicts

Even if the planner forgot to set `blocked_by` between two tasks sharing files or packages, the dispatcher detects it at dispatch time:

1. Builder A dispatches with files `[internal/domain/a.go]`, packages `[internal/domain]` → both locks acquired.
2. Builder B tries to dispatch with files `[internal/domain/b.go]`, packages `[internal/domain]`. File `b.go` is free, but the package `internal/domain` is already locked by Builder A.
3. Dispatcher adds dynamic `blocked_by: Builder-A's-action-item` to Builder B's action item.
4. Builder B defers until Builder A completes and releases both locks.

This is the safety net. The planner should set `blocked_by` explicitly; the dispatcher catches what the planner missed, at either the file OR package level.

### 5.6. File Path Gating via `--allowedTools`

The dispatcher translates file paths to Claude Code tool restrictions:

```
--allowedTools "Edit(<path1>),Write(<path1>),Edit(<path2>),Write(<path2>),Read,Grep,Glob,Bash(mage *)"
```

The agent has full read access to the entire project directory but can only **edit** files in its assigned paths. This prevents an agent from accidentally modifying files outside its scope, even if the LLM decides to.

### 5.7. Why Not Worktrees

Worktrees add complexity without proportional benefit:
- Git worktree management overhead
- Hylla branch isolation overhead
- Merge conflict risk when worktrees land on main
- No benefit when file path gating prevents cross-agent interference

File path gating is simpler and sufficient:
- Each builder edits only its assigned files
- Multiple builders work in the same checkout simultaneously
- No merge step needed — all changes are in the same tree
- `mage test-func` prevents test interference

---

## 6. Template Configuration

### 6.1. What Templates Define

Templates are the configuration layer for the cascade. They define:

| Configuration | What It Controls | Example |
|---|---|---|
| **Agent binding** | Which agent type fires for this kind | `build` → `go-builder-agent` |
| **Model** | Which Claude model to use | `opus`, `sonnet`, `haiku` |
| **Effort** | Claude Code effort level | `high`, `standard`, `low` |
| **Tools** | Allowed and disallowed tools | `allowedTools: [Read, Edit, Bash, Grep]` |
| **Budget** | Max cost per invocation | `max_budget_usd: 5.00` |
| **Turns** | Max conversation turns | `max_turns: 20` |
| **Max-tries** | Total attempts before permanent failure | `max_tries: 2` |
| **Gates** | Deterministic verification steps | `[{command: "mage ci", on_fail: "fail_task"}]` |
| **Child rules** | Auto-created children on item creation | `build` → auto-create `build-qa-proof`, `build-qa-falsification` |
| **Trigger state** | Which state transition fires the agent | `in_progress` (default for all) |
| **Escalation** | Whether failures re-trigger planning | `escalation_enabled: true` |
| **Push policy** | Whether to push after commit | `auto_push: true` (default), `auto_push: false` |
| **Commit message** | Always formed by the haiku commit agent. No deterministic fallback. Style rules hardcoded in the agent's prompt details. | `commit_agent: true` |

### 6.2. Example Template Kind Definition (Sketch)

Templates bind against the 12-value `action_items.kind` enum. Example spec for the `build`, `plan`, and `commit` kinds:

```toml
[kinds.build]
agent_name = "go-builder-agent"
model = "sonnet"
effort = "standard"
max_budget_usd = 5.00
max_turns = 20
max_tries = 2
trigger_state = "in_progress"
auto_commit = true      # overridable per project or per kind instance
commit_agent = true

[kinds.build.tools]
allowed = ["Read", "Edit", "Write", "Bash", "Grep", "Glob"]
disallowed = ["Agent"]

[[kinds.build.gates]]
name = "ci"
command = "mage ci"
on_fail = "fail_task"

[kinds.build.child_rules]
auto_create = ["build-qa-proof", "build-qa-falsification"]

[kinds.plan]
agent_name = "go-planning-agent"
model = "opus"
effort = "high"
max_budget_usd = 10.00
max_turns = 30
max_tries = 2
trigger_state = "in_progress"
escalation_enabled = true

[kinds.plan.tools]
allowed = ["Read", "Grep", "Glob", "Bash"]
disallowed = ["Edit", "Write", "Agent"]

[kinds.plan.child_rules]
auto_create = ["plan-qa-proof", "plan-qa-falsification"]
# commit-cadence rule (see §19.2 discussion): at plan close-time, if this plan
# is at level ≥ 2 AND has ≥1 `build` child in state `complete` AND auto_commit = true,
# auto-generate a `commit` child.

[kinds.commit]
agent_name = "commit-message-agent"
model = "haiku"
effort = "low"
max_budget_usd = 0.10
max_turns = 3
```

`plan-qa-proof` / `plan-qa-falsification` / `build-qa-proof` / `build-qa-falsification` bind to their matching go-qa-*-agent with matched model (opus for plan-QA, sonnet for build-QA). `research`, `closeout`, `refinement`, `discussion`, `human-verify` do not bind to an agent at the template level (orchestrator-managed or template-triggered post-Drop-4).

### 6.3. Per-Project Template Overrides

The template defines defaults. Projects can override per-kind settings (e.g., a project with expensive CI might increase `max_budget_usd`). Override mechanism TBD — likely a project-level config section.

---

## 7. Agent Types and Model Assignment

### 7.1. Agent Inventory

The cascade uses these agent types:

| Agent Type | Agent File | Role | Edits Code? |
|---|---|---|---|
| **Planner** | `go-planning-agent.md` | Decomposes work into action items | No |
| **Plan QA Proof** | `go-qa-proof-agent.md` | Verifies plan completeness/consistency | No |
| **Plan QA Falsification** | `go-qa-falsification-agent.md` | Attacks plan for vagueness/errors | No |
| **Builder** | `go-builder-agent.md` | Implements code changes | **Yes** |
| **Build QA Proof** | `go-qa-proof-agent.md` | Verifies build evidence/reasoning | No |
| **Build QA Falsification** | `go-qa-falsification-agent.md` | Attacks build for counterexamples | No |
| **Commit Agent** | `commit-message-agent.md` *(new)* | Forms commit messages from git diff | No |
| **Wiki** | TBD | Maintains running work ledger | No |
| **Quality/Vuln** | TBD | Graph-based code quality checks | No |

### 7.2. Model and Effort Assignment

| Agent Type | Model | Effort | Rationale |
|---|---|---|---|
| **Planner** | opus | high | Decomposition requires deep understanding. Quality here bounds everything downstream. "Spec quality bounds output quality." |
| **Plan QA** | opus | high | Must catch planner errors. Cheap QA on an expensive plan is false economy. |
| **Builder** | sonnet | standard | Code generation is well-bounded by the plan. The plan already did the thinking. |
| **Build QA Proof** | sonnet | medium | Evidence verification is structured. Semi-formal certificate guides the work. |
| **Build QA Falsification** | sonnet | medium | Counterexample search is structured. Certificate + `paths`/`packages` scope the search. |
| **Commit Agent** | haiku | low | Reads `git diff`, action item title, and template commit format rules. Forms a conventional-commit message. Extremely narrow scope. |
| **Wiki** | haiku | low | Summarization action item. Absorb child wikis, produce summary. |
| **Quality/Vuln** | sonnet | high | Deep graph analysis. Needs careful reasoning about resource lifecycles. |

**Open question:** Is sonnet sufficient for builders? The plan provides detailed instructions, file paths, acceptance criteria. The builder's job is execution, not design. But complex code changes may need opus. **Suggestion:** Default to sonnet, template-configurable to opus for complex kinds.

---

## 8. Auth and Lifecycle

### 8.0. Pre-Cascade Auth Flow (Today, Pre-Drop-4)

Until Drop 4 ships the dispatcher and Drop 1.6 ships the auth-approval cascade, auth flows manually:

1. **Orchestrator auth** — dev approves every orchestrator launch in the TUI (STEWARD, DROP_N_ORCH). 12h TTL, 4h approval window. STEWARD pre-stages drop-orch auth requests on behalf of the dev so each orch launch is one TUI tap rather than a cold create-then-approve cycle.
2. **Subagent auth (canonical post-§19.1.6)** — orch approves every non-orch subagent (planner, QA, builder, research, commit) auth request itself, scoped within the orch's subtree. Dev never sees subagent auth requests in the TUI. See §19.1.6 for the auth-layer rule and §8.1 below for the post-Drop-4 dispatcher-issued variant of the same idea.
3. **Subagent auth (pre-§19.1.6 workaround, today)** — today's auth layer still gates approvals to the dev. Both `STEWARD_ORCH_PROMPT.md` §8.1 and `DROP_N_ORCH_PROMPT.md` Section A2 carry an S2 dev-fallback: orch attempts the approve call; if rejected, surface to dev in chat for manual TUI approval. Documented friction; resolved when §19.1.6 ships.

The pre-cascade orchestrator IS the dispatcher — it picks the kind, picks the agent variant, spawns the subagent, provisions the auth, approves the auth (post-§19.1.6), and watches state transitions manually. Drop 4 replaces the orch-as-dispatcher loop with a real dispatcher; auth flow §8.1 below describes that target.

### 8.1. Auth Flow for Agents

Agent auth is **pre-approved by the system**. No human approval step because:

1. The human already approved the cascade by moving the parent to `in_progress`
2. The template defines which agents fire — the human approved the template
3. Each agent gets auth scoped to its specific action item (or parent drop for planners)

```
Dispatcher detects: action_item moved to in_progress
  ├── Creates auth session: role=<from_template>, scope=<from_template>, item=<action_item_id>
  │   TTL: from template (default 30 min)
  │   No human approval — system-issued
  ├── Passes auth credentials in agent prompt
  └── Agent claims auth at boot
```

### 8.2. Auth Revocation

Auth is revoked when:
- The agent moves its action item to `complete` or `failed` (auto-revocation)
- The TTL expires (agent took too long)
- The dispatcher kills the agent (e.g., budget exceeded)

### 8.3. Agent Terminal Action

When an agent calls `till.action_item(operation=move_state, state=complete|failed)`:

1. MoveActionItem validates the transition
2. Auth is auto-revoked
3. LiveWaitBroker publishes state-change event
4. Dispatcher kills the CLI process
5. Next cascade step fires

The agent does NOT need to call any other "cleanup" tool. The terminal state move IS the signal.

---

## 9. Gates, Commits, and Deterministic Steps

### 9.1. What Gates Are

Gates are deterministic verification steps that run **programmatically by the dispatcher**, not by any agent. No LLM is involved (except the commit agent, which is a tightly-scoped haiku agent). Gates run after a build agent reports success.

### 9.2. Gate Execution Flow

```
Build agent moves action item to complete
  │
  ├── Dispatcher receives state-change event
  ├── Checks template: does this kind have gates?
  │   YES ↓
  │
  ├── GATE 1: mage ci
  │   ├── Pass → continue
  │   └── Fail → action item moves to failed (override auth) + gate output as comment
  │
  ├── GATE 2: QA agents fire (proof + falsification, parallel)
  │   ├── QA runs against the code as-built, before any commit.
  │   │   (CLAUDE.md Build-QA-Commit Discipline: QA precedes commit.)
  │   ├── ALL PASS → continue
  │   └── ANY FAIL → action item moves to failed + QA output as comment
  │         → escalate (re-plan or human). QA never retries per-action-item.
  │
  ├── GATE 3: commit + push + CI green (deterministic + commit agent)
  │   ├── git add <affected files>
  │   ├── Spawn commit agent (haiku):
  │   │     reads git diff, action item title, commit style rules embedded
  │   │     in the agent's prompt (hardcoded in the agent file's details),
  │   │     outputs a single conventional-commit message
  │   ├── System validates structure and length only (non-empty, within
  │   │     a hard length cap). NO regex style validation. NO deterministic
  │   │     fallback message.
  │   ├── On structure/length fail: re-spawn commit agent with the rejection
  │   │     reason. Max 2 tries. After 2 fails: escalate to orchestrator.
  │   │     Orchestrator receives the git diff command and the git commit
  │   │     command, runs them, dispatcher re-validates, commits if green.
  │   ├── System runs: git commit -m "<message>"
  │   ├── If auto_push=true in template: git push
  │   ├── gh run watch --exit-status until CI lands green
  │   └── Fail → action item moves to failed + commit/CI output as comment
  │
  └── All gates pass → action item complete
      (NO per-action-item hylla reingest. Ingest is drop-end only, owned by
       DROP N END — LEDGER UPDATE. See §9.7 + §15.7 + CLAUDE.md.)
```

### 9.3. The Commit Agent

The commit agent is a special lightweight agent:

- **Model:** haiku (cheapest, fastest)
- **Scope:** Reads `git diff --cached`, the action item's title and description, and outputs a single commit message string.
- **Commit style rules are hardcoded in the agent file's prompt details**, not templated per-project. The agent knows the conventional-commit format, length caps, and repo conventions because its system prompt says so.
- **No file edits.** No MCP mutations. No state changes. Pure text generation.
- **System validates structure and length only** — non-empty, within a hard length cap. There is **no regex style validator** and **no deterministic fallback**. If the message fails structure/length, the system re-spawns the commit agent with the rejection reason.
- **Max 2 tries.** On second failure, the system escalates to the orchestrator: posts an attention item with the `git diff --cached` command and the `git commit -m "<message>"` command. The orchestrator forms a message, runs the commit command, the dispatcher re-validates, and commits if green.
- **System makes the actual `git commit` and `git push` calls.** The commit agent only forms the message — it never touches git directly.

This keeps the commit flow 95% deterministic (system runs git) with a thin LLM layer for message quality.

### 9.4. Push Configurability

Whether to auto-push after commit is **template-configurable**:

```toml
[kinds.build]
auto_push = true   # push after every successful commit (default)
# auto_push = false  # commit only, push at drop completion
```

When `auto_push = false`, the dispatcher commits locally but defers push. The **drop completion check** (Section 9.6) catches unpushed commits.

### 9.5. Gate Output

Gate stdout/stderr is captured and posted as a `till.comment` on the action item. On failure:
- Action item moves to `failed` (dispatcher uses override auth for `complete → failed` transition)
- `metadata.outcome: "failure"`
- `metadata.gate_name: "<which gate failed>"`
- Gate output posted as comment
- Retry logic fires (Section 14)

### 9.6. drop Completion Check

When all children of a drop are `complete`, before the drop itself moves to `complete`, the dispatcher runs a completion check:

```
All children complete → drop completion check:
  ├── git status: any uncommitted changes in drop scope?
  │   YES → attention item to orchestrator
  ├── git log @{push}..: any unpushed commits?
  │   YES → if auto_push=true: push now. If auto_push=false: attention item.
  └── All clean → drop moves to complete
```

This catches edge cases where a builder's commit was deferred or a gate partially completed.

### 9.7. Hylla Reingest — Drop-End Only, Never Per-Action-Item

Hylla reingest is a deterministic step, **not a per-action-item gate**. It fires **once per Drop** inside the `closeout` action item, owned by the drop-orch pre-merge on the drop branch. Never per `build`.

```
DROP N END — LEDGER UPDATE (drop-orch) →
  confirm all sibling tasks complete →
  git push (if anything un-pushed) → gh run watch --exit-status green →
  hylla_ingest(artifact_ref=github.com/evanmschultz/tillsyn@main,
               enrichment_mode=full_enrichment) →
  finalize five level_2 findings-drop descriptions → close LEDGER UPDATE
```

Invariants (mirrored from CLAUDE.md + `feedback_orchestrator_runs_ingest.md`):

- **Always** `enrichment_mode=full_enrichment`. Never `structural_only`.
- **Always** source from the GitHub remote artifact ref, never from a local working copy.
- **Never** before `git push` + `gh run watch --exit-status` green.
- **Only the drop-orch** calls `hylla_ingest`. Build agents, QA agents, commit agent, STEWARD — never. No exceptions.

**QA agents verify against the code as-built, not against a freshly-ingested Hylla graph.** QA reads the current working tree (LSP + git diff) for uncommitted-since-last-ingest deltas. Keeping ingest drop-end only avoids (a) ingest-flooding the Hylla artifact with per-action-item snapshots and (b) making QA cost proportional to action-item count. Full drop-end ingest feeds STEWARD's MD writes + planner context for the next drop.

The template carries Hylla settings (artifact ref, enrichment mode) so the drop-orch reads them programmatically rather than hard-coding per-drop. See §15.7 for STEWARD's consumption of the ingest results via the five level_2 findings drops.

---

## 10. Trust Model

### 10.1. The Problem

Agents confabulate, skip cases, claim tests pass without running them. Unstructured agents are wrong 20-22% of the time on code verification tasks (semi-formal reasoning paper, arXiv 2603.01896).

In current Tillsyn usage: agents forget to move action items, claim work is complete but don't update state, orchestrators skip QA, and completion claims are unverifiable.

### 10.2. How the Cascade Solves It

At **no point** does the system trust any agent's self-assessment. Every claim is independently verified:

```
PLANNER claims plan is good
  → Planning QA independently verifies (proof + falsification)
  → Plan QA FAILS? → planner was wrong, provably

BUILDER claims code is complete
  → Deterministic gates verify (mage ci, tests, lint)
  → Build QA independently verifies (proof + falsification)
  → Gates FAIL? → builder was wrong, provably
  → QA FAILS? → builder's claim was unsupported

QA claims verification is complete
  → Semi-formal certificate structure forces evidence citation
  → Mandatory enumeration prevents case-skipping
  → Unknowns are first-class outputs, routed to orchestrator
```

### 10.3. Layers of Verification

| Layer | What | Who | Trust Level |
|---|---|---|---|
| 1 | Plan decomposition | Planner agent | **Untrusted** — verified by plan QA |
| 2 | Plan quality | Plan QA agents (proof + falsification) | **Semi-trusted** — structured by certificate |
| 3 | Code implementation | Build agent | **Untrusted** — verified by gates + build QA |
| 4 | Mechanical correctness | Deterministic gates | **Fully trusted** — deterministic, no LLM |
| 5 | Semantic correctness | Build QA agents | **Semi-trusted** — structured by certificate |
| 6 | Final approval | Human review | **Fully trusted** — ultimate authority |

### 10.4. Why This Is Stronger Than Current Practice

Currently the orchestrator must remember to call QA, and agents must remember to update state. Both fail regularly (see Section 10.1). The cascade makes these structural:

- **QA is automatic** — template auto-creates QA children, dispatcher auto-fires them
- **State updates are the terminal action** — the agent's MCP call IS the state update
- **Gates are deterministic** — no agent can bypass `mage ci`
- **Hierarchy enforces completion** — parent can't be `complete` until all children are `complete` (via `require_children_done`)

### 10.5. Comparison to Stripe

Stripe has: deterministic gates + human review.

We add:
- **Semi-formal reasoning certificates** (evidence-grounded, counterexample-obligated)
- **Asymmetric QA** (proof verifies evidence; falsification attacks conclusions)
- **Hierarchical planning QA** (plans are verified before execution, not just code)
- **Unknowns as first-class outputs** routed into Tillsyn coordination
- **Template-driven configuration** (gates, agents, models are project-specific)
- **Full audit trail** (every state transition, comment, gate output is persistent)

Stripe can rely on thousands of human reviewers to catch what gates miss. We compensate with QA agents.

### 10.6. Minions + Semi-Formal Full-Benefit Rule

**Cascade design rule:** the Stripe Minions pattern and the semi-formal reasoning certificate (arXiv 2603.01896) are used **to the full extent of their benefit** — not cited as inspiration, not partially adopted, not treated as optional scaffolding. Structurally enforced by template config (§6), gate placement (§9), QA agent prompts (§11), and dispatcher-owned commit flow (§9.3).

Concretely this means every cascade run must exhibit:

- **Deterministic-agentic-deterministic sandwich.** Every `build` action item is bracketed by deterministic gates on both sides: git-status pre-check + package-lock acquire (in) → builder agent (agentic middle) → `mage ci` + QA agents + commit-agent + push + CI-green watch (deterministic out). No agentic step is ever the final arbiter of "done." The deterministic bookends are what make the agentic middle affordable. (Hylla reingest is drop-end only, not a per-action-item bookend — see §9.7.)
- **2-CI-round hard cap per try, max 2 tries.** From Stripe's Minions: if `mage ci` fails twice on the same try, the try is done; retry policy lives in §14, not inside the agent. Agents do not loop on CI internally.
- **Mandatory certificate structure on every QA claim.** Premises / Evidence / Trace-or-cases / Conclusion / Unknowns — no batch assertions, no "all tests pass" without enumeration (§11.2). Enforced by QA agent prompts and verified by orchestrator review on escalation.
- **Hypothesis-refinement loop on every falsification pass.** HYPOTHESIS → EVIDENCE → STATUS (CONFIRMED | REFUTED | REFINED) → REVISION (§11.3). Each hypothesis gets its own comment; the reasoning chain is inspectable after the fact.
- **Evidence grounding on every claim.** Every certificate line that asserts a fact must cite Hylla / `git diff` / Context7 / `go doc` / gopls. Prose without citations is treated as unverified and demoted to Unknowns.
- **Unknowns as routable coordination state.** Unknowns are not erased — they become Tillsyn comments, handoffs, or attention items so the orchestrator can route them (§11.4).

**Non-negotiables:** these shapes are enforced by the template and by the QA agent prompts. A cascade run where any of them is absent is a cascade design bug, not a style variation. Plan QA falsification specifically checks for missing certificate structure or missing evidence citations on sibling QA outputs.

---

## 11. Semi-Formal Reasoning Integration

### 11.1. The Certificate Structure

Every QA agent (plan QA and build QA) must produce a certificate:

```
PREMISES: [what must be true]
EVIDENCE: [Hylla citations, file:line references, git diff excerpts]
TRACE/CASES: [per-case execution path analysis]
CONCLUSION: [the claim, derived from above]
UNKNOWNS: [what remains uncertain — routed to Tillsyn]
```

### 11.2. Mandatory Enumeration

QA agents must **enumerate** the cases they verify and trace each one separately. No batch assertions ("all tests pass"). The certificate template forces per-case analysis.

This is the semi-formal paper's strongest finding: mandatory enumeration prevents case-skipping, which causes the majority of false-positive verifications.

### 11.3. Hypothesis-Refinement Loop

For QA falsification, formalize the hypothesis loop:

```
HYPOTHESIS: [what the QA agent believes might be wrong]
EVIDENCE: [code citations from Hylla, test output, graph nav results]
STATUS: CONFIRMED | REFUTED | REFINED
REVISION: [if REFINED, what changed and why]
```

Track this in the action item's comments so the orchestrator can see the reasoning chain. Each hypothesis gets its own comment entry with the status update.

### 11.4. Pre-Build Preparation — Update Existing Agents and CLAUDE.md

**Before building the cascade**, update the existing agent files and CLAUDE.md to enforce semi-formal reasoning in the current manual workflow:

1. Update `~/.claude/agents/go-qa-proof-agent.md` and `go-qa-falsification-agent.md` with explicit certificate templates and mandatory enumeration
2. Update `~/.claude/agents/go-planning-agent.md` with planning certificate requirements
3. Update project CLAUDE.md files to explain semi-formal reasoning for the orchestrator's current manual workflow
4. Update CLAUDE.md to clearly document how the orchestrator should use agents as they stand today (without the cascade), so the workflow works during the build phase

This serves two purposes:
- Immediately improves current agent quality
- Validates the certificate structure before automating it

### 11.5. Full-Benefit Rule (Tie-in to §10.6)

See §10.6 "Minions + Semi-Formal Full-Benefit Rule." The certificate shape here (§11.1), the mandatory enumeration rule (§11.2), and the hypothesis-refinement loop (§11.3) are all **structurally enforced**, not stylistic suggestions:

- Template config (§6.2) binds QA kinds to agent prompts that contain the certificate template verbatim.
- Post-build gates (§9.2) run deterministic verification around the agentic middle so the certificate never has to carry mechanical correctness claims alone.
- Plan QA falsification (§3.2) specifically checks sibling QA outputs for missing certificate fields, missing evidence citations, and ungrouded prose.
- Unknowns are first-class outputs routed via `till.comment` / `till.handoff` / `till.attention_item` — they are not prose footnotes, they are coordination state (§11 Unknowns line).

**If a QA output lacks certificate structure, it is treated as "no QA was done" — not a style nit.** That's what "full extent of benefit" means for this rollout.

---

## 12. Concurrency Model

### 12.1. Hard-Coded Concurrency Cap (Dogfood) — N=6

The dispatcher enforces a **hard-coded soft cap of N=6 concurrent active agents** across the whole cascade during dogfood. This is a single global semaphore around `dispatch()`, not per-kind or per-drop. When N agents are in flight, new eligible items wait in a FIFO queue keyed by the state-transition event that made them eligible.

- **Why hard-coded (not template-configurable yet).** Dogfood needs one predictable value so failure modes come from the cascade itself, not from a knob being set too high on one day and too low on another. Config surface adds ambiguity when we have zero production data to tune against.
- **Why N=6 specifically.** Matches the practical ceiling for one Max $200 subscription running pure-headless sessions (see §22 ToS posture) without hitting per-account rate limits hard enough to poison the cascade's failure signal. Lower values starve parallelism on embarrassingly-parallel drops; higher values start queuing at the API boundary and look like cascade regressions.
- **Refinement plan.** §19.10 drop 10 bullet "Cascade concurrency soft-cap enforcement mechanism" promotes this from a hard-coded constant to a template-configurable field with an account-tier-aware default (Max $100 subscribers need a lower cap, API-key-backed installs need a different cap entirely — see §22).

**Parallelism within the cap.** Any eligible item without blockers fires when a dispatch slot opens; the cascade still naturally parallelizes. The cap bounds total outstanding agents, not the structural parallelism — so sibling QA-proof + QA-falsification still run concurrently, and independent build tasks in separate packages still run concurrently, as long as the total under-the-cap.

### 12.2. Resource Contention

Multiple build agents may run in the same project directory simultaneously. This requires:

1. **File-level blocking** (Section 5) — each builder is restricted to specific files, one builder per file
2. **Per-function mage targets** — builders run `mage test-func <pkg> <func>` to test their own changes without interfering with other builders testing in the same package

**Early TODO:** Expand `magefile.go` to support `mage test-func <pkg> <func>` so multiple builders can test their own changes in the same package without interfering. The full `mage ci` gate runs after the builder completes — that's the system's job, not the builder's.

### 12.3. Builder Test Flow

Builders and the system have distinct testing responsibilities:

| Who | What | When |
|---|---|---|
| **Builder** | `mage test-func <pkg> <func>` on affected functions | During implementation, as a rapid feedback loop |
| **System (gate)** | `mage ci` (full suite: build, vet, lint, test, coverage) | After builder marks complete, before commit |

This split lets builders iterate fast on narrow tests while the system ensures the full suite passes before committing.

### 12.4. Failure Due to External Limits

Concurrent agents may fail due to:
- Network errors (API rate limits, timeouts)
- Claude API usage limits
- Local resource exhaustion (disk, memory)

These failures must be:
1. Detected by the dispatcher (process exit with non-zero, no state change in Tillsyn)
2. Recorded as `failed` with `metadata.outcome: "blocked"` and `metadata.blocked_reason: "external_limit"`
3. Surfaced as an attention item to the human with the specific error
4. Distinguishable from "agent tried and failed" (which is `metadata.outcome: "failure"`)

### 12.5. Stale Process Detection

If an agent's auth TTL expires without a state change:
1. Dispatcher kills the process
2. Action item moves to `failed` with `metadata.outcome: "blocked"`, `metadata.blocked_reason: "timeout"`
3. File locks released
4. Attention item fires

---

## 13. Hylla Integration

### 13.1. Agent Access to Hylla

All agents get Hylla MCP access for code understanding. The dispatcher provides an `agent-mcp.json` config that includes:

- **Tillsyn MCP** — for action-item mutations (move state, create items, post comments)
- **Hylla MCP** — for code understanding (search, graph nav, node full)
- **Context7** — for library docs (optional, configurable)

gopls is excluded (too stateful, slow initialization, not needed for one-shot work).

### 13.2. Hylla Reingest

Reingest is a **drop-end programmatic step** (Section 9.7), owned by the drop-orch inside the Drop's `closeout` action item. Not a per-`build` gate, not an agent action item. The dispatcher calls `hylla_ingest` programmatically from the remote artifact ref after `git push` + `gh run watch --exit-status` green:

```
hylla_ingest(
  artifact_ref = github.com/<org>/<repo>@<branch>,
  enrichment_mode = full_enrichment
)
```

Template configures:
- Artifact ref (remote GitHub ref — never local path)
- Enrichment mode (always `full_enrichment`)

**QA agents do NOT depend on fresh Hylla graph.** QA reads the working tree via LSP + git diff for uncommitted-since-last-ingest deltas. Drop-end reingest feeds the next drop's planner context + STEWARD's MD-write inputs, not mid-drop QA.

### 13.3. Hylla for Planning

Planner agents use Hylla extensively:
- `hylla_search` / `hylla_search_keyword` to understand current code
- `hylla_graph_nav` to trace dependencies
- `hylla_refs_find` to find all references to symbols being changed
- `hylla_node_full` to read function implementations

The planner's job is to decompose work based on the actual code structure, not assumptions. Hylla provides ground truth.

---

## 14. Escalation and Retry Policy

### 14.1. `max-tries` Semantics

`max-tries=2` means **2 total attempts**. Not 2 retries after the initial.

- First attempt: agent fires, does work, succeeds or fails
- If fails: second attempt fires with failure context from the first
- If second attempt fails: permanent failure, attention item to orchestrator/human

### 14.2. What Counts as a Try

| Event | Counts as a try? |
|---|---|
| Build agent fails | Yes |
| Gate (`mage ci`) fails | Yes (same try as the builder — builder produced bad code) |
| QA fails | **No** — QA failure changes state to `failed`, but does not trigger a retry. QA never talks to builder. The orchestrator/escalation system handles QA failures. |
| External failure (network, limits) | **No** — this is a `blocked` outcome, not a `failure`. Different retry semantics (see 14.3). |

### 14.3. External Failure Retries (max-tries + 2)

External failures (`metadata.outcome: "blocked"`) don't count against `max-tries` but **are capped at +2 additional attempts**:

- `max-tries=2` gives 2 normal attempts for agent-caused failures
- +2 additional attempts for external/blocked failures (network errors, API limits, timeouts)
- Total theoretical maximum: 4 attempts (2 normal + 2 blocked retries)
- Blocked retries fire after a cooldown (configurable, default 60s)
- If all blocked retries exhaust → permanent failure, attention item to human

This prevents infinite retry loops on infrastructure issues while giving transient failures a fair chance to resolve.

### 14.4. Escalation Up

When a `build` action item fails after `max-tries` exhausted:

1. The `build` action item is permanently `failed`
2. Attention item fires to the plan-level orchestrator
3. If `escalation_enabled: true` in template:
   a. A new `plan` action item is created at the drop level with the failure context
   b. A planner agent fires to re-plan the drop, incorporating the failure
   c. Planning QA verifies the revised plan
   d. New build tasks are created from the revised plan
   e. `max-tries=2` for the escalation cycle itself
4. If escalation also fails (2 tries), attention item to human. Full stop.

### 14.5. Escalation Tracking

**Nothing happens in the dark.** Every escalation must produce:

1. A diff of how the plan changed from the original
2. Why the plan changed (failure context from the failed `build` action item)
3. What the planner tried differently
4. A comment on the original failed action item linking to the new plan

The planner agent is responsible for documenting this in its `plan` action item's comments. QA verifies the documentation is present and accurate.

### 14.6. Template Configuration

```toml
[kinds.build]
max_tries = 2
blocked_retries = 2          # additional attempts for external failures
blocked_retry_cooldown = 60  # seconds between blocked retries
escalation_enabled = true    # on permanent failure, re-plan at parent level

[kinds.plan]
max_tries = 2
escalation_enabled = false   # plan failure goes to human, not further re-planning
```

Escalation can be disabled per-kind in the template. When disabled, permanent failure goes directly to an attention item for the orchestrator/human.

---

## 15. Wiki / Ledger System

**Status:** Partial resolution (drop 0 convergence 5.3 + temp.md Q3). Wiki infra seeded in drop 0 (`WIKI.md`, `WIKI_CHANGELOG.md`, `HYLLA_WIKI.md`); wiki-agent automation (Section 15.1) lands post-drop-4. Per-drop MD cleanup subdrop structural rule added in 15.6 below.

### 15.6. Per-drop MD Cleanup Subdrop (drop 0 Convergence)

Every drop's closeout (`closeout` action item per Section 1.4, previously "drop <N> END — LEDGER UPDATE") MUST include an **MD cleanup action item** that:

1. Scans the drop's shipped work against current MD files (`PLAN.md`, `REFINEMENTS.md`, `HYLLA_REFINEMENTS.md`, `CLAUDE.md`).
2. Trims entries that landed in this drop — replace the long-form refinement entry with a one-line summary pointing to the drop's `WIKI_CHANGELOG.md` line and the commit SHA.
3. Removes stale sections where a design question got resolved by the drop's work.
4. Runs BEFORE the wiki-updater action item (when that lands post-drop-4) so the wiki aggregator sees current MD state.

This prevents `PLAN.md` and the refinement logs from accreting resolved cruft. The commit history + `WIKI_CHANGELOG.md` holds the full audit trail, so trimming is safe.

Pre-cascade (now): the orchestrator performs the MD cleanup action item manually during drop-end closeout.
Post-cascade (drop 4+): a dedicated `md-cleanup-agent` subtype fires under the closeout drop and is verified by QA.

### 15.1. Concept

A wiki agent maintains a running summary of all work done at its level. It fires twice:
1. After the plan is accepted (initial wiki entry)
2. After the level is marked `complete` (final wiki entry)

Not on failure — the orchestrator gets failure info directly via attention items.

### 15.2. What the Wiki Contains

- Affected code blocks (from `paths` / `packages`)
- Action item IDs and their current states
- Code still to be affected (open items)
- Summary of changes made (from completed items' comments)

### 15.3. Hierarchical Absorption

Child wikis are absorbed by parent wikis:
- Build-level wikis are detailed (exact files, symbols, line ranges)
- drop-level wikis summarize build-level wikis (file groups, feature areas)
- Parent-drop wikis summarize child-drop wikis (feature descriptions, architectural changes)

The further up the tree, the more summarized. This gives the orchestrator a quick view without drowning in detail.

### 15.4. Storage

**Open question:** Where do wikis live?
- Option A: As comments on the action item (simple, uses existing infrastructure)
- Option B: As a dedicated `wiki` field in action item metadata (queryable, structured)
- Option C: As a separate wiki table in the DB (most flexible, most work)

**Leaning toward:** Option A (comments) for initial drop, Option B (metadata) for later. Comments are append-only and human-readable. Metadata is structured and queryable.

### 15.5. Orchestrator Memory Compaction

When the orchestrator compacts memory, it should absorb the wiki summaries. The wiki provides a structured, pre-summarized view that's cheaper to load than re-reading all action items and comments.

**Open question:** How does wiki content integrate with orchestrator memory management? This needs design work.

### 15.7. Drop-Orch + STEWARD MD Ownership Split (Post-2026-04-19)

**Supersedes the earlier "STEWARD owns all MD writes" framing.** Dev directive 2026-04-19 during Drop 1.5 `DROP_END_LEDGER_UPDATE` execution. Applies every drop going forward. See `STEWARD_ORCH_PROMPT.md` §1.3 for the canonical MD ownership map and memory `project_drop_pr_flow_and_workflow_architecture.md` for the full flow. Memory `feedback_steward_owns_md_writes.md` is marked **SUPERSEDED**.

**Drop-orch (`DROP_N_ORCH`) owns on the drop branch:**

- All drop-lifetime artifact content — LEDGER entry, REFINEMENTS raised, HYLLA feedback, WIKI changelog, DISCUSSIONS. Written to files under `main/workflow/drop_N/` (see §15.9 for layout) on the drop branch; flows to `main` via drop merge.
- Architecture-MD edits when the drop's scope touches process (`CLAUDE.md`, `PLAN.md`, `AGENT_CASCADE_DESIGN.md`, `STEWARD_ORCH_PROMPT.md`, `workflow/README.md`, `workflow/example/drops/_TEMPLATE/*`).
- Rebase onto `origin/main` + conflict resolution (Go conflicts → builder subagent; MD conflicts → drop-orch directly).
- PR creation + dev-approved merge.
- Remote branch + local branch ref cleanup (`git push origin --delete drop/N` + `git branch -D drop/N`).

**STEWARD owns post-merge, running from `main/` (not bare root):**

- Reads `main/workflow/drop_N/` content post-merge; discusses with dev; collates into `main/LEDGER.md`, `main/REFINEMENTS.md`, `main/HYLLA_FEEDBACK.md`, `main/WIKI_CHANGELOG.md`, `main/HYLLA_REFINEMENTS.md`.
- Continues to curate `main/WIKI.md` between drops.
- Runs `git worktree remove drop/N` for local cleanup — **nothing else**. STEWARD does **not** delete branches (remote or local); drop-orch already did that.

**Six persistent level_1 STEWARD-owned drops.** Direct children of the project; never close. The set is subject to refinement over time — dev intent (2026-04-16): *"we will need to refine steward each drop too. So, that may change as we develop this system."*

| Persistent drop | Feeds MD file in `main/` |
|---|---|
| `DISCUSSIONS` | (cross-cutting audit trail; no single MD) |
| `HYLLA_FINDINGS` | `HYLLA_FEEDBACK.md` |
| `LEDGER` | `LEDGER.md` |
| `WIKI_CHANGELOG` | `WIKI_CHANGELOG.md` |
| `REFINEMENTS` | `REFINEMENTS.md` |
| `HYLLA_REFINEMENTS` | `HYLLA_REFINEMENTS.md` |

**Tillsyn L1/L2 structure still tracks the work.** Drop-orch still creates the per-drop level_2 findings drops under STEWARD's persistent L1 parents for Tillsyn-native visibility. The **content** source of truth is the on-disk files under `main/workflow/drop_N/` (§15.9) — drop-orch writes to disk during the drop, and STEWARD reads from disk post-merge. Level_2 drop descriptions may carry short pointers into the disk MDs but are no longer the canonical content store.

**Drop-close sequence (load-bearing).**

1. All build + QA action items in drop N → `done`.
2. Drop-orch works `DROP N END — LEDGER UPDATE` pre-merge on the drop branch: finalizes the drop-end artifact MDs under `main/workflow/drop_N/` → `git push` → `gh run watch --exit-status` green → `hylla_ingest` (full enrichment, remote ref) → `gh pr create --base main --head drop/N` → dev-approved merge.
3. Drop-orch post-merge: `git push origin --delete drop/N` + `git branch -D drop/N` → post `till.handoff` to `@STEWARD` naming `main/workflow/drop_N/` → close `DROP N END — LEDGER UPDATE`.
4. STEWARD on `main/`: reads `main/workflow/drop_N/`, discusses with dev, writes the corresponding MDs on `main`, commits docs-only with single-line conventional-commits, pushes, closes the level_2 findings drops.
5. STEWARD works the refinements-gate item inside drop N's tree (§15.8) — discusses next-drop refinements + STEWARD-self refinement with dev, applies agreed changes, closes the gate.
6. Only after the refinements-gate closes can drop N's level_1 close (parent-blocks-on-incomplete-child).
7. STEWARD runs `git worktree remove drop/N` locally.
8. Drop N+1 starts.

**Pre-Drop-3 enforcement = honor-system.** Drop 3 enforcement = template auto-generation of per-drop `workflow/drop_N/` scaffolding + refinements-gate + new `steward` orch `principal_type` with auth-level state-lock. See §19.3.

### 15.8. Per-Drop Refinements-Gate + STEWARD-Self Refinement

**Every numbered level_1 drop must carry a STEWARD-owned refinements-gate item** inside its own tree, named `DROP_N_REFINEMENTS_GATE_BEFORE_DROP_N+1`. Created by drop-orch at drop spin-up. `blocked_by` every other Drop N item + the five level_2 findings drops. Worked by STEWARD post-merge. Blocks the numbered drop's level_1 closure until STEWARD closes it.

When STEWARD works the refinements-gate post-merge, the conversation covers **two prompts**:

1. **Next-drop refinements** — which of drop N's refinements-raised entries (captured in `DROP_N_REFINEMENTS_RAISED` + `DROP_N_HYLLA_REFINEMENTS_RAISED`) should be applied to drop N+1's action items before N+1 starts? Apply agreed refinements directly to the level_2 items under drop N+1 (creating N+1's parent if the dev is ready to spin it up).
2. **STEWARD-self refinement** — does STEWARD's scope, prompt, persistent-drop set, or per-drop flow need refinement from drop N's lessons? Dev quote (2026-04-16): *"every drop the amount will be a refinement thing, lol."* Expect non-zero STEWARD-self refinement every drop. Common outcomes: add/rename a persistent drop; adjust drop-close sequence; update memory; edit `STEWARD_ORCH_PROMPT.md`.

Closing the refinements-gate unblocks the numbered drop's level_1 closure. STEWARD summarizes the gate's decisions in `completion_notes`.

### 15.9. Per-Drop `workflow/` Dir Architecture

**`workflow/` is git-tracked** and lives in `main/workflow/`. Bare-root `workflow/` is retired. STEWARD ran the one-time migration at commit `effaad9` on main. See `AGENT_CASCADE_DESIGN.md` §8.4 for the atomic-small-things rendering contract.

**Per-drop subdir:** `main/workflow/drop_N/` — created by drop-orch at drop spin-up, mirrors `workflow/example/` + `workflow/example/drops/_TEMPLATE/` shape (atomic-small-things discipline — many small MDs, not monoliths). Drop-orch writes directly to files under this subdir as the drop progresses; all of `workflow/drop_N/` flows to `main` via the drop's PR merge.

**`failures/` subdir at each branched level of `drop_N/`.** Never delete QA / plan / build artifacts. Failed QA, plan, or build content moves into `failures/` so the next iteration's plan / QA files can learn from and count them. Retention = forever. **Forward-only** — no retroactive backfill for pre-2026-04-19 drops (dev directive 2026-04-19).

**Only edit workflow-process flow from `AGENT_CASCADE_DESIGN.md` + `workflow/example/drops/_TEMPLATE/`.** Those two are the canonical atomic-small-things source; `workflow/drop_N/` for any specific N is a rendering of that template shape for drop N's specific work.

**Flow during drop N (pre-merge, on the drop branch):**

1. Drop-orch spin-up: create `main/workflow/drop_N/` mirroring `_TEMPLATE`.
2. Planners, builders, QA populate per-unit MDs under their respective subdirs as work progresses.
3. Failed iterations move to `failures/` at the appropriate level; never deleted.
4. At drop end, the `DROP_N_END_LEDGER_UPDATE` subdir holds the five drop-end artifact MDs (hylla findings, ledger entry, wiki changelog entry, refinements raised, Hylla refinements raised).

**Flow post-merge (STEWARD on `main/`):**

1. Reads `main/workflow/drop_N/` content (primarily the drop-end artifact MDs).
2. Discusses with dev, splices content into `main/LEDGER.md` / `main/REFINEMENTS.md` / `main/HYLLA_FEEDBACK.md` / `main/WIKI_CHANGELOG.md` / `main/HYLLA_REFINEMENTS.md`.
3. Commits docs-only on `main` with single-line conventional-commits, pushes.
4. Works drop N's refinements-gate (§15.8) — discussing next-drop + STEWARD-self refinements with dev.
5. Closes drop N's level_2 findings drops + level_1.
6. Runs `git worktree remove drop/N` for local cleanup. (Remote + local branch refs were already deleted by drop-orch pre-STEWARD.)

**Rebase + Hylla staleness.** During a main-diverged rebase, use `git diff` as primary evidence, **not Hylla**. Hylla reflects `main` at time of last ingest; post-rename (e.g. Drop 1's `task → action_item`, `plan_item → action_item`), Hylla's node names drift from current committed state. Re-ingest on `main` post-drop-merge restores Hylla freshness for next drop. Subagents report "Hylla stale vs rebase target; used `git diff`" as expected pattern during rebase, not a miss.

---

## 16. Quality and Vulnerability Checking

**Status:** **DEFERRED** (drop 0 convergence 5.1, 2026-04-14). Dev direction: "let's defer small wins tracking to later." Quality / vulnerability checking as a third QA step is post-dogfood territory — revisit after the cascade is self-hosting. The design below is preserved for when we pick it back up; nothing in drops 1–9 depends on it shipping. drop 10 refinement cleanup is the earliest realistic landing window.

### 16.1. Concept

A third QA step after build that uses Hylla's graph navigation to check for structural code issues:

- Resource lifecycle: opened file → is it closed? Is the close deferred?
- Memory management (for languages that need it)
- Error handling: returned error → is it checked?
- Interface contracts: does the implementation satisfy the interface?
- Goroutine lifecycle: spawned goroutine → is it joined or cancelled?

### 16.2. How It Works

```
Builder complete → Gates pass → QA Proof + QA Falsification pass
  → Quality/Vuln check fires (third QA step)
  → Uses hylla_graph_nav to trace resource lifecycles
  → N agents do the same checks (configurable redundancy)
  → All must pass for the `build` action item to be fully verified
```

### 16.3. Configurable Redundancy

The template configures how many agents run each quality check:

```toml
[kinds.quality-check]
agent_name = "go-quality-agent"
model = "sonnet"
effort = "high"
replicas = 3  # 3 agents do the same check independently
consensus = "all_pass"  # all must pass (vs "majority_pass")
```

More replicas = higher chance of catching issues = higher cost. Template-configurable tradeoff.

### 16.4. Standalone Mode

Quality/vuln checks can also run independently, not as part of a build cascade:

```
Human creates a quality-check action item → fires quality agents
  → Agents scan specified code using Hylla graph nav
  → Report findings as comments
```

Useful for periodic codebase health scans.

### 16.5. Language-Specific

Currently Go-only. Each language will need its own quality agent with language-specific checks. Add a action item to support more languages when Hylla supports them.

---

## 17. Prerequisites

### 17.1. Hard Prerequisites (drop 1 — the fresh-project equivalent of "D1 done right")

| Feature | What | Why Required for Cascade |
|---|---|---|
| **Failed lifecycle state** | Fourth terminal state `failed` across domain / app / adapters / storage / config / capture / snapshot | Agents must represent failure. Gates must move tasks to failed. |
| **Outcome required on `failed`** | Moving an action item to `failed` requires a non-empty `metadata.outcome` (one of `failure`/`blocked`/`superseded`). Empty `outcome` on terminal `failed` is a domain-level validation error. | Without it, a `failed` action item is indistinguishable from a `failed-for-unknown-reason` action item, and the cascade can't route. |
| **Parent-blocks-on-failed-child (always-on, not configurable)** | Parent cannot move to `complete` while any child is `failed`. **Not a template flag. Not a policy option.** Always-on built-in behavior. Bypass only via the supersede path (human CLI, orchestrator version post-dogfood). | Core cascade integrity. A configurable version (`require_children_done`) can be set to false and breaks the cascade silently. Remove the knob. |
| **Human supersede CLI** | `till action_item supersede <id> --reason "..."` — human-only command that marks a `failed` action item as `superseded` in `metadata.outcome` and transitions `failed → complete`. Bypasses the terminal-state guard because the CLI asserts human intent at the binary boundary. | Currently the human has no way to resolve stuck `failed` items. Before any cascade runs, the human needs to be able to unstick things. |
| **Auth auto-revoke on terminal state** | Auth session ends when the action item moves to `complete` or `failed` | Dead agent auth sessions must clean up. "One auth per scope" constraint. |
| **Action-item details as prompt** | Agent reads action-item detail fields as its working brief | Simplifies agent prompts — the action item IS the prompt. |
| **Action-item `paths` as first-class field** | `paths []string` on the action item, planner-set, readable by builder + QA. Domain-level, not buried in metadata JSON. | Plan-QA falsification needs to query siblings' paths to detect cross-action-item file conflicts (Section 5.3). Without a first-class field, QA has no data to check. Replaces the removed D10 "affected_artifacts". |
| **Action-item `files` as first-class field (read-only reference)** | `files []string` on the action item, planner-set via TUI file-picker, distinct from `paths` (which is edit-scope). Holds reference files the builder should read but not edit. | The drop 4.5 file-viewer (§24) and mention-routing (§23) both read `files` to render attached material. Without a first-class field, reference attachments leak into metadata JSON and can't be rendered in the TUI. Validation enforces files exist in the repo at creation time. |
| **Action-item `start_commit` / `end_commit`** | Two fields on the action item. `start_commit` set at creation (current HEAD). `end_commit` set at move-to-complete (current HEAD). Domain-level. | Needed before the dispatcher takes over commits. Pre-dogfood: orchestrator + dev manage git manually, these fields just record the boundary. Post-dogfood: dispatcher reads these to decide reingest/commit scope. |
| **Creation gated on clean git for declared paths** | At action-item creation, if any path in `paths` is dirty in `git status --porcelain`, creation fails with an error telling the orchestrator to clean up git first. | Without this gate, a cascade agent (or orchestrator) inherits uncommitted state and silently mixes it into its work. Always-on behavior. |
| **Orchestrator supersede auth (deferred, post-dogfood)** | Programmatic supersede via orchestrator auth (not human CLI). | Only needed when the orchestrator has to unstick things autonomously. Pre-dogfood, the human CLI is enough. Keep this out of drop 1; it ships in drop 11. |
| **`human-verify` kind (landed in Drop 1.75)** | Dev-gated action items that bracket every Drop (START — PLANNING CONFIRMATION, END — REVIEW DONE + CORRECT — see §2.2). Shape: `human-verify` action item with an attention item child addressed to `@dev`, `blocked_by` semantics so parent `plan` cannot complete until it's signed off. | The START/END bracketing rule (§2.2) needs a real kind to hang off of, not prose. Drop 1.75's kind collapse lands `human-verify` as one of the 12 closed `action_items.kind` values — orchestrator creates these with `kind='human-verify'` directly, no `Role:` prose workaround. Drop 3 adds template-level auto-generation of start/end `human-verify` items when a top-level `plan` is created. |

### 17.2. Not Prerequisites (Removed from cascade scope)

| Feature | Why Not Needed |
|---|---|
| **Auth claim response enrichment** | Designed for orchestrator-triggered non-headless agents. Headless cascade dispatch passes everything needed in the spawn prompt — no claim-time enrichment required. |
| **`require_children_done` as a configurable policy** | Removed as a knob. Replaced by the always-on behavior in 17.1. Having it as a setting meant the default could (and did) ship as `false`, silently breaking cascade integrity. |
| **Level-based signaling** | Agents fail and die. Dispatcher reads failure. No runtime signaling. |
| **Auth approval loop for cascade agents** | Agent auth is system-issued inside the cascade, no human approval step. |
| **TUI rendering of `failed` (deferred)** | Post-dogfood. Pre-dogfood: the orchestrator exposes failures to the human via a CLI subcommand (`till action_item list --state failed` or `till failures list`). TUI rendering is nice-to-have, not load-bearing. |

---

## 18. Pre-Build Preparation

Before building the cascade, update the existing workflow to enforce the patterns the cascade will automate.

### 18.1. Update Agent Files for Semi-Formal Reasoning

Update `~/.claude/agents/`:

- `go-qa-proof-agent.md` — add explicit certificate template with mandatory enumeration
- `go-qa-falsification-agent.md` — add hypothesis-refinement loop structure
- `go-planning-agent.md` — add planning certificate with scope/evidence requirements
- `go-builder-agent.md` — add `paths` / `packages` reporting requirements (update once the fields land in drop 1)

### 18.2. Update CLAUDE.md Files

Update project and global CLAUDE.md to:
- Document semi-formal reasoning for the orchestrator's current manual workflow
- Document how the orchestrator should use agents as they stand today
- Note that the cascade isn't built yet, so the orchestrator must manually trigger agents
- Define the certificate structure the orchestrator should expect from agents

### 18.3. Expand Mage Targets

Add `mage test-func <pkg> <func>` so multiple builders can test individual functions without interfering. This is needed for concurrent builders in the same package.

### 18.4. Audit Path Logic

Examine the existing path resolution in TUI bootstrap flow. Plan refactoring to backend so the dispatcher can reuse it.

### 18.5. Add `mage install` with Dev-Promoted Commit Pinning

Add a `mage install` target that installs from a specific commit hash, not from HEAD:

```
mage install → git checkout <pinned-commit> -- && go install . && embed version
```

**Dev-promoted per drop.** The dev is the only actor that can bump the pinned commit. The promotion happens at a clean boundary — after a drop completes and its QA has cleared. The dev runs `mage install COMMIT=<hash>` explicitly; the cascade never promotes itself.

This is critical for safe dogfooding: when the cascade system is building Tillsyn itself, the installed `till` binary must be a known-good version. If a cascade agent produces broken code and the binary is rebuilt from HEAD, the broken binary could corrupt the cascade. Pinning to a dev-promoted commit breaks that loop.

### 18.6. MCP Passthrough for Headless Agents (Resolved)

Confirmed: `claude --bare -p "..." --mcp-config <path> --strict-mcp-config` accepts an ad-hoc MCP server list and ignores the dev's `settings.json`. See Section 20.5 for evidence and flag details. No pre-build research remaining — this flows directly into the dispatcher design in the cascade drops.

### 18.7. CI Cleanup — Mac-Only Workflows

`.github/workflows/` currently exercises Linux + Windows + macOS matrix runs. Pre-cascade we only dogfood on Mac; the Linux/Windows legs are noise (slower feedback, flaky runner pool, no deployment target uses them). Strip them. Keep only the macOS job(s). Review every workflow file — `ci.yml`, any release / matrix / nightly workflow — and delete Linux/Windows branches, matrix entries, conditional runners, and any OS-specific scripts they reference. If the change removes the only consumer of an action or cached dependency, remove those too. After the cut, `mage ci` locally + a triggered GH Actions run on the cleaned workflows must both stay green. Builder subagent owns this; QA-proof + QA-falsification pass both before commit. Windows/Linux support can return if/when a real deployment target needs it — don't preemptively re-add it.

### 18.8. Dev MCP Server Setup

Each worktree needs its own locally-registered MCP server pointing at the worktree's built `till` binary — orchestrators test against the dev version, not the installed one, and unique per-worktree names prevent binaries from different worktrees from colliding. Full setup instructions (`claude mcp add` command template, per-worktree naming scheme, active registrations in this repo's dev environment) live in `main/CONTRIBUTING.md` § "Dev MCP Server Setup". Both `CLAUDE.md` files carry only a one-line summary and a pointer — the canonical text is in CONTRIBUTING.

---

## 19. Development Order

**drop sequencing principle.** Waves are gone. Everything is a drop, including the prerequisite work. drop 0 is a fresh-project reset; drop 1 is the fresh-project equivalent of "D1 done right" from the old plan. Dogfooding turns on as soon as the dispatcher can actually dispatch something (drop 5).

**Every drop is bracketed by dev-gated START/END subdrops (applies to every drop below).** See §2.2 for the structural rule.

- **`DROP N START — PLANNING CONFIRMATION WITH DEV`** — first child of the drop. Dev reviews scope, plan shape, and any agent/system-prompt decisions before the planning agent fires. Absorbs the `DROP N START — REFINEMENT REVIEW` action item as a sibling inside the bracket (the refinement review feeds the planning confirmation, they happen at the same drop-start checkpoint).
- **Five STEWARD-scope level_2 findings drops + one STEWARD-owned refinements-gate** — drop-orch creates these at drop spin-up (see §15.7 + §15.8). Level_2 findings drops sit under the persistent level_1 STEWARD parents (`HYLLA_FINDINGS` / `LEDGER` / `WIKI_CHANGELOG` / `REFINEMENTS` / `HYLLA_REFINEMENTS`). The refinements-gate item `DROP_N_REFINEMENTS_GATE_BEFORE_DROP_N+1` sits inside drop N's tree, `blocked_by` every other drop N item + the five level_2 findings drops. STEWARD works the refinements-gate post-merge (covers both next-drop refinements and STEWARD-self refinement — expect non-zero STEWARD-self refinement every drop). The refinements-gate blocks drop N's level_1 closure until STEWARD closes it.
- **`DROP N END — REVIEW DONE + CORRECT`** — last child of the drop. `blocked_by` `DROP N END — LEDGER UPDATE` + the refinements-gate. Dev confirms all work landed correctly; covers the doc-review checklist below.

At the END subdrop, orchestrator + dev review and update:
- Bare-root `CLAUDE.md`
- `main/CLAUDE.md`
- `~/.claude/agents/*.md` (agent files the cascade or orchestrator actually uses)
- `~/.claude/CLAUDE.md` only if a global rule changed

**No subagents on this review.** Orchestrator and dev decide directly. Keep the docs aligned with current code state so the next drop's subagents aren't briefed on stale rules.

**Template constraint (applies throughout).** The `default-go` template structure gets trimmed to what the cascade actually reads. Templates bind kinds to: agents, models, effort levels, tools, budgets, turns, gates, child-rules, trigger state, escalation, and push policy — nothing more. Don't add fields the dispatcher doesn't consume.

**Git management pre-dispatcher.** Until the dispatcher's commit logic lands (post-dogfood refinement, drop 11), the **orchestrator and dev manage git manually**: the orchestrator reminds the dev to clean up dirty paths before a action item is created, and the dev handles the actual commits. Action items still carry `start_commit` / `end_commit` fields from drop 1, but those fields are records, not triggers, until the dispatcher is wired up.

### 19.0. drop 0 — Project Reset + Docs Cleanup

Before any cascade code lands:

- [x] Delete the current messy Tillsyn project in Tillsyn (`a0cfbf87-b470-45f9-aae0-4aa236b56ed9`) — renamed to `TILLSYN-OLD` and replaced by fresh project `a5e87c34-3456-4663-9f32-df1b46929e30`. Hard-delete deferred to drop 10 (project lifecycle ops).
- [x] Create a **fresh Tillsyn project with NO template bound.** Done — `a5e87c34-3456-4663-9f32-df1b46929e30`.
- [x] Full rewrite of bare-root `/Users/evanschultz/Documents/Code/hylla/tillsyn/CLAUDE.md` and `main/CLAUDE.md` — landed across `1a63cc5`, `1825d78`, `48e91ea`, `aef9482`, `8bad5ea`, `9cf1037`, `870de3e`, `b411b48`, `d32680f`. Bare-root + `main/` bodies aligned, both ~200+ lines (target relaxed to fit the cascade architecture sections).
- [x] Full rewrite of `~/.claude/agents/*.md` for the cascade-dispatched agents — 18.2 (`f4334081`) shipped the rewrite of `go-builder-agent.md`, `go-qa-proof-agent.md`, `go-qa-falsification-agent.md`, `go-planning-agent.md`. Stale D1–D10 vocabulary removed; spawn contract + self-managed lifecycle baked in.
- [x] Update agent files for semi-formal reasoning specifics (18.1 refinements on top of the rewrite) — folded into 18.2.
- [x] Add `mage test-func` target (18.3) — landed; visible in `mage -l` as `testFunc`.
- [x] Audit path logic in TUI, plan backend refactoring (18.4) — audit landed; refactor itself deferred to drop 1+ (TUI bootstrap path resolution stays put pre-cascade).
- [x] Add `mage install` with dev-promoted commit pinning (18.5) — **superseded** in `d4fd2c2` to a simplified dev-only build-and-save target (`refactor(install): simplify mage install to dev-only build-and-save`). Dev-promoted commit pinning deferred to §19.10 refinement bullet — pre-cascade dogfood doesn't need the pin yet.
- [x] MCP passthrough for headless agents — **already resolved** (Sections 18.6, 20.6). No pre-build research remaining.
- [x] CI cleanup — strip Linux/Windows from `.github/workflows/`, keep macOS only (18.7) — landed in `08cb397` (`fix(ci): cold-cache mage ci parity and macos-only matrix`).
- [x] **Mid-drop additions** *(added during drop 0 execution — detail tracked in Tillsyn action items, not re-described here)*:
  - **18.10 gofumpt adoption** — committed `d684dcb`; required 18.10B follow-up because of cold-cache leak.
  - **18.10B fix cold-cache `mage ci` gofumpt gate** (`runGofumptList` + `trackedGoFiles` stdout/stderr split; `wrapCommandErrorWithStderr` for error paths). Ships with 18.7 in a single push so post-push CI is macos-only and green.
  - **18.11 auth-cache `SessionStart`-hook MVP** — shipped; read-side cache-inject on resume/compact/startup. **18.11B `PostToolUse`-hook auto-persist** — shipped; removes manual-Write discipline. Retroactively captured as action items post-ship.
  - **18.12 fix gopls build-tags for `magefile.go`** — **closed without shipping (2026-04-14)**. Initial builder landed `.vscode/settings.json` with `gopls.build.buildFlags = ["-tags=mage"]`, but the premise was wrong: the dev uses nvim (not VS Code) on this repo, and gopls does not auto-read project-root `.vscode/settings.json` — its config comes from the LSP client (nvim-lspconfig for the dev, the `gopls-lsp@claude-plugins-official` plugin for Claude Code). The checked-in file would not have affected either runtime. File was reverted; `.vscode/` is now ignored alongside other editor cruft. Real fix (if still needed) belongs in editor-side config, not the repo tree.
- [ ] **Per-drop wrap-up:** confirm the rewritten CLAUDE.md and agent files match the plan post-cleanup.

### 19.0.5. Drop 1 Prerequisite — Multi-Orch Auth (Lands on `main` Before Drop 1)

Sequenced **after Drop 0** and **before Drop 1 can start**. This is a hotfix on `main`, not a drop branch — STEWARD orchestrates it from the bare root; code lands directly on `main` while DROP_1_ORCH is paused. `drop/1.5` continues on its own worktree in parallel; any merge conflicts with this hotfix resolve at Drop 1.5 merge time (dev accepted the risk 2026-04-17).

**Problem:** The auth layer enforces "one active auth session per scope level" — documented in `main/CLAUDE.md` §"Auth and Leases." The dev gave DROP_1_ORCH project-scope auth as the pre-Drop-2 workaround for the drop-collapse parity bug (see `AUTH_LAYER_RESEARCH_2026-04-17.md`). That project-scope session blocks STEWARD (and any other orchestrator) from obtaining its own project-scope auth simultaneously. Drop 1 cannot move forward without both orchestrators participating — STEWARD owns MD writes + per-drop findings routing, DROP_1_ORCH owns the drop 1 code work.

**Scope (planner refines):**

- [ ] **Identify enforcement locus.** Planner's first job: determine whether "one per level" is enforced by (a) DB UNIQUE index on `auth_sessions` / `capability_leases`, (b) Go runtime check in `Service.ClaimAuthRequest` / `IssueCapabilityLease` / equivalent, (c) both, or (d) CLAUDE.md documentation only with no code enforcement. Report DB impact before builder spawns.
- [ ] **Allow multiple concurrent orchestrator sessions at the same scope level, keyed on distinct principal identity.** Two orchestrators with different `principal_id` must be able to hold active sessions at the same scope level (e.g. project-scope) simultaneously.
- [ ] **Preserve per-identity single-session invariant.** A single orchestrator identity (same `principal_id`) must still hold at most one active session per scope level — re-claim should either return the existing session or reject the duplicate, planner picks the semantics and justifies.
- [ ] **Schema migration only if strictly needed.** Existing sessions + leases remain valid. No data rewrite. If the constraint is pure Go, no DB touch at all.
- [ ] **Test coverage.** New tests verifying: (1) two different `principal_id` orchestrators claim project-scope concurrently → both succeed; (2) same `principal_id` re-claims project-scope → rejected-or-returns-existing per planner's choice; (3) revoking one orchestrator's session does not affect the other's.
- [ ] **Post-merge CLAUDE.md update.** Change "One active auth session per scope level at a time" in both `main/CLAUDE.md` and bare-root `CLAUDE.md` to reflect the new rule. STEWARD owns this edit — lands as part of the hotfix commit set, not deferred.
- [ ] **`mage ci` passes on `main`** before the hotfix is considered done.

**Worklog:** `main/DROP_1_UNBLOCK_MULTI_ORCH_AUTH_2026-04-17.md` (plan + QA verdicts + build log + CI verdict).

**Workflow (STEWARD-orchestrated):**

1. STEWARD writes PLAN.md stub (this section) + worklog MD skeleton.
2. STEWARD spawns `go-planning-agent` — investigates enforcement locus, writes concrete fix plan into the worklog, reports DB impact.
3. STEWARD spawns `go-qa-proof-agent` + `go-qa-falsification-agent` in parallel on the plan. Loop back to planner if any unmitigated counterexample.
4. STEWARD reports DB-impact answer to dev + awaits approval before builder spawns.
5. STEWARD spawns `go-builder-agent` with the converged plan.
6. STEWARD spawns QA proof + QA falsification in parallel on the build. Loop back to builder if needed.
7. STEWARD runs `mage ci` in `main/` — gate, must pass.
8. STEWARD commits (code + PLAN.md + worklog MD + CLAUDE.md updates), pushes, runs `gh run watch --exit-status`.
9. STEWARD reports green CI to dev.

**Why this slot:** DROP_1_ORCH is blocked behind it today. Drop 1.5 continues in parallel; merge-resolution deferred. Drop 1.6 (§19.1.6) builds on the multi-orch assumption (STEWARD + drop orchs approving their own subagent auth concurrently), so this lands first.

**Pre-fix workaround (current state, pre-merge of this hotfix):** Only one orch at a time can hold project-scope auth. Dev rotates STEWARD vs DROP_1_ORCH sessions manually. This hotfix ends that rotation.

### 19.1. drop 1 — Failed Lifecycle State (Fresh-Project "D1 Done Right")

The hard prerequisites from Section 17.1, shipped cleanly against the fresh project. This is the foundation drop 2+ sits on. Each item must pass `mage ci` + QA proof + QA falsification before it's marked complete.

- [ ] **`go.mod` `replace` directive cleanup** *(drop 1 first action item, before any lifecycle work)*. Strip every `replace` directive in `go.mod` except the fantasy-fork replacement (dev maintains a personal fork of `go-fantasy` / equivalent for this project). Grep-audit `go.mod` for every `replace (...)` stanza, delete any that point at local filesystem paths left over from experimentation, delete any that pin an upstream to an old version for reasons nobody still remembers, and keep only the fantasy-fork line documented inline with a `// fantasy-fork: <rationale>` comment. After edits: `go mod tidy` (via mage wrapper if one exists; if not, this is the one raw-`go` exception justified by being a module-file operation the dev runs, not an agent), `mage ci`. QA-proof + QA-falsification required — a stray `replace` that points at a missing path silently breaks every downstream build. Motivation: dev direction (2026-04-16) — "only replace is supposed to be my fork of fantasy."
- [ ] **Install local git hooks for gofumpt + `mage ci` parity** *(drop 0 refinement, scheduled here as drop 1 first item)*. Add committed `.githooks/pre-commit` that runs a new `mage format-check` target (public wrapper around the existing private `formatCheck()` in `magefile.go:218-236`) and `.githooks/pre-push` that runs `mage ci` in full. Add a `mage install-hooks` target that sets `core.hooksPath = .githooks` so the tracked hook scripts become the active hooks for any fresh clone. Also fix the `mage format` no-arg ergonomics wart discovered in drop 0 closeout: `func Format(path string) error` (`magefile.go:200`) requires a positional arg, making the `path == "" || path == "."` branch in the body unreachable from CLI (`mage format` errors with "not enough arguments"); split into `Format()` (no-arg = whole tree via `trackedGoFiles()`) and `FormatPath(path string)` (scoped), or adopt a variadic form. Motivation: drop 0 surfaced that gofumpt drift landed on `main` because no local gate catches it pre-commit — `mage ci` is the CI-parity gate but runs too late to prevent red pushes. Hooks must remain bypassable via `--no-verify` per existing discipline (global CLAUDE.md rule: never bypass without explicit dev instruction). QA-proof + QA-falsification required — the hook scripts are the local build gate, can't silently break.
- [ ] `failed` lifecycle state (fourth terminal state) across domain / app / adapters / storage / config / capture / snapshot. Fix the HEAD gaps (gofmt regression in `app_service_adapter_outcome_test.go`, empty-outcome acceptance in `validateMetadataOutcome`).
- [ ] Require non-empty `metadata.outcome` on any transition to `failed` (domain-level validation error, not just value whitelist).
- [ ] **Remove** `require_children_done` as a configurable option. Replace with always-on parent-blocks-on-failed-child behavior enforced at every hierarchy level. No template flag, no policy knob. Bypass only via the supersede path below.
- [ ] Human supersede CLI: `till action_item supersede <id> --reason "..."` — marks `failed` action item as `metadata.outcome: "superseded"` and transitions `failed → complete`. Bypasses the terminal-state guard because the CLI asserts human intent at the binary boundary.
- [ ] Auth auto-revoke on terminal state (`complete` or `failed`).
- [ ] **Server-infer `client_type` on auth request create** *(gap surfaced in drop 0)*. Remove `client_type` from the `till.auth_request(operation=create)` MCP tool schema — callers shouldn't declare transport; the server knows. Entrypoint adapters stamp it: MCP-stdio adapter stamps `"mcp-stdio"`, TUI stamps `"tui"`, CLI stamps `"cli"`. Tighten `app.Service.CreateAuthRequest` to reject empty `ClientType` at create time (matches the existing approve-path check in `autentauth.Service.ensureClient`) so the asymmetric validation bug that bit drop 0 — create accepted empty, approve rejected empty with `ErrInvalidClientType` — is structurally unreachable. Governance + display still consume `client_type` as a first-class field; only the caller responsibility moves server-side. MCP-layer tests drop the field; domain-layer tests keep it on `CreateAuthRequestInput` since that's the domain boundary. `client_id` stays caller-supplied (same transport can come from different software).
- [ ] **Reject unknown keys across all MCP mutation paths** *(gap surfaced in drop 0)*. `till.project(operation=create)` silently dropped every non-schema key in my drop-0 metadata payload — caller thought fields landed, they didn't. Same asymmetric-validation pattern as the `client_type` bug above. Audit every `till.*` mutation tool (`till.project`, `till.action_item`, `till.comment`, `till.handoff`, `till.attention_item`, `till.capability_lease`, `till.kind`, `till.template`) and every nested metadata/extension object each one accepts. Every MCP handler must reject unknown keys with a structured error naming the offending key and the accepted schema — never silent-drop. If extension-style freeform fields are wanted for any surface, add an explicit named `extensions map[string]string` (or equivalent) to the domain type so it's documented and validated, not an anything-goes sink. Add golden tests asserting the error shape for each handler. Scope note: this is the *validation* fix; adding new first-class cascade fields to the project node is drop 4's dispatcher prerequisite, not this item.
- [ ] **PATCH semantics on all update handlers — no more silent full-replace** *(gap surfaced in drop 0; second repro confirmed in 18.2 closeout)*. `till.project(operation=update)` with a partial payload (only `name` + `metadata`) wiped the stored `description` back to empty string — the handler is full-replace without documenting it. Second silent-data-loss bug in the same family as unknown-key drop above. **Live second repro from 18.2 closeout (2026-04-14)**: `till.action_item(operation=update)` on action item `f4334081-84ad-47a4-bcf9-238c2f915ad2` passing only `title` + `metadata` wiped `description` (full rewrite contract) and `labels` (`["agents","docs","orchestrator-scope","drop-0"]`). Confirms the behavior is handler-family-wide, not project-only. Audit every `till.*` update/mutation handler for the same behavior (`till.project`, `till.action_item`, `till.comment`, `till.handoff`, `till.attention_item`, `till.kind`, `till.template`). Pick ONE semantics per handler and enforce it: either (a) true PATCH — only provided fields change, omitted fields preserved — which matches caller intuition and is strongly preferred, or (b) explicit full-PUT with a required `replace_all: true` flag that forces the caller to acknowledge they are overwriting. Never silently wipe fields because the caller didn't repeat them. Preserve the drop 0 precedent in tests: `update(name, metadata)` must leave `description` intact; `till.action_item.update(title, metadata)` must leave `description` + `labels` intact. **Third repro (2026-04-14, 18.10B closeout)**: builder on 18.10B + 18.7 hit it again — a `till.action_item.update(title, ...)` with no `description` arg cleared 18.7's stored description; builder worked around by re-calling update with the full original description restored. Evidence that every orchestrator / builder round-trip through update is a latent data-loss risk until this lands.
- [ ] **Accept `state` in place of `column_id` on `till.action_item(operation=create)`; stop leaking column UUIDs into the agent contract** *(gap surfaced in drop 0)*. Fresh project had auto-seeded default columns (`To Do`, `In Progress`, `Done`) but `till.action_item(op=create)` rejects the call unless the caller passes the literal column UUID — and no MCP op exposes column UUIDs (`till.capture_state` loads them for state-hashing but does not surface them; there is no `list_columns` operation). An orchestrator following MCP-only discipline has no way to discover the UUID. Column identity is a UI/layout concern; agents only care about lifecycle state. Fix: `till.action_item(op=create)` must accept `state` (`todo` / `in_progress` / `done` / `failed` once drop 1 adds it) and resolve the column UUID server-side via the existing `resolveActionItemColumnIDForState` helper (`internal/adapters/server/common/app_service_adapter_mcp.go:811`). Keep `column_id` accepted for TUI drag-and-drop callers that genuinely know the UUID, but make `state` the documented agent-facing input and reject the call only when *both* are empty. Same cleanup on `till.action_item(op=move)` where `to_column_id` currently faces the same leak — accept `state` and resolve internally. Add a golden test proving an orchestrator with no column knowledge can create an action item purely by `state`. No column-listing MCP op needs to be added; the goal is to make column IDs invisible to the agent surface, not to expose them. **Second + third repro (2026-04-17)**: both the `rak` and `fckin` template-free projects blocked their DROP_1_ORCH / DROP_1.5_ORCH-equivalent launches the same way — agent couldn't create any action item because `column_id` is required at the MCP boundary and no discovery op exists; dev had to hand-surface column UUIDs via direct sqlite query and paste them into the orchestrator prompt. Confirms the fix is launch-gating for every fresh project using the cascade model, not just drop 0.
- [ ] Action-item details as prompt (agent reads action-item fields as working brief).
- [ ] First-class `paths []string` field on action items (planner-set, readable by builder + QA). Domain-level field, not buried in metadata JSON. Replaces the removed `affected_artifacts`.
- [ ] First-class `packages []string` field on action items (covers every file in `paths`). Used by package-level blocking (Section 5.2).
- [ ] First-class `files []string` field on action items — set of files **attached to** the drop (distinct from `paths`). Populated by the planner via the TUI file-picker (drop 4.5 §24) so a drop can carry reference material (existing code the builder should read, prior-design docs, sibling-drop output) without those files being counted as edit-scope. `files` is **read-only reference**; `paths` is **edit-scope**. The drop 4.5 file-viewer (§24) reads `files` to render attached content with `charmbracelet/glamour` and show `git diff` against `start_commit`. Validation: every `files` entry must exist in the repo at creation time; duplicates across `files` + `paths` are allowed (a file can be both edit-scope and reference). QA-proof + QA-falsification verify that the planner populated `files` where reference material is needed (Plan QA falsification treats missing reference attachments on work that depends on external context as a plan gap).
- [ ] First-class `start_commit` / `end_commit` fields on action items. `start_commit` set at creation (current HEAD). `end_commit` set at move-to-complete (current HEAD).
- [ ] Creation gated on clean git for declared paths: if any path in `paths` is dirty in `git status --porcelain`, creation fails with an error telling the orchestrator/dev to clean git first. Always-on.
- [ ] CLI failure listing: `till action_item list --state failed` (or `till failures list`) so the human can see `failed` tasks without TUI rendering. TUI rendering of `failed` is deferred post-dogfood.
- [ ] **Deferred post-dogfood (documented here, not built yet):** orchestrator programmatic supersede via system-issued auth. Human CLI is enough for Wave-1-equivalent scope.
- [ ] **Per-drop wrap-up:** update CLAUDE.md + agent files to reflect the new required fields, the always-on block behavior, and the supersede CLI.

### 19.1.6. drop 1.6 — Auth Approval Cascade (Orch Self-Approves Non-Orch Subagents)

Sequenced **after Drop 1 + Drop 1.5** and **before Drop 2**. This drop unblocks the canonical orch-spawn-subagent flow documented in `STEWARD_ORCH_PROMPT.md` §8.1 and `DROP_N_ORCH_PROMPT.md` Section A2 — today the system gates every auth approval to the dev TUI, which makes orchs that need to spawn 5+ subagents per drop (planner / QA proof / QA falsification / builder / commit / research) untenable for the dev's approval bandwidth. The pre-fix workaround in those prompts is "if `approve` is rejected by today's guardrails, surface to dev in chat for manual TUI approval" — that workaround disappears when this drop ships.

**Scope:**

- [ ] **Auth-layer rule: orchs may approve non-orch subagent auth requests scoped within their own subtree.** The auth layer must accept `till_auth_request operation=approve` from a session whose `principal_role: orchestrator` AND whose lease scope encompasses the request's `path`, when the request's `principal_role` is non-orch (`planner | qa | builder | research | commit`) — `research` lands as a first-class principal role in Drop 2 (§19.2) and is listed here so the approval rule covers it from day one. Reject orch-self-approval (an orch cannot approve another orchestrator's auth — that stays a dev-only operation). Cross-orch approval is also rejected (DROP_1_ORCH cannot approve DROP_1.5_ORCH's subagent requests; STEWARD cannot approve DROP_1_ORCH's subagent requests unless explicit dev opt-in lands in a later refinement).
- [ ] **STEWARD cross-subtree exception:** STEWARD's project-scoped lease covers all six persistent level_1 parents, so STEWARD's approve calls cover any subagent request whose path roots under those parents. Drop orchs' branch-scoped leases cover only their own drop subtree — they cannot approve subagent auth for items they don't own. Drop orchs may add level_2 nodes under STEWARD's persistent parents (per Drop Orch Cross-Subtree Exception in `WIKI.md`), but subagent auth for work on those added nodes still routes through the drop orch (which created them) for approval, not STEWARD.
- [ ] **No configurability in this drop.** Threshold knobs (e.g. "auto-approve only QA, route builder approval to dev"; "max subagents auto-approved per hour"; "specific subagent roles always go to dev") are explicitly deferred to a later refinement drop in §19.10. This drop ships the binary capability: orch-approves-non-orch-subagent-in-subtree, full-stop.
- [ ] **Dev opt-out switch (project-scope):** project-scope toggle `metadata.orch_self_approval_enabled: bool` (default `true` once the capability lands) so a dev who wants every approval to flow through TUI for a given project can flip it off without rebuilding. Backstop, not the everyday path.
- [ ] **Audit trail:** every orch-approved auth request must record the approving orch's `agent_instance_id` + `lease_token` + `principal_id` in the auth approval row so post-hoc audit shows "STEWARD approved STEWARD_PLANNER_DISCUSSIONS_TYPE_OVERHAUL on 2026-04-22" rather than just "approved". Surface in the TUI auth log so the dev can scan recent approvals.
- [ ] **MCP-layer test coverage:** new golden tests for the four interesting cases — (1) orch-in-subtree approves non-orch in same subtree → success; (2) orch-in-subtree tries to approve another orchestrator → rejected; (3) orch-A tries to approve orch-B's subagent in B's subtree → rejected; (4) STEWARD approves a subagent under one of its persistent parents → success.
- [ ] **Prompt updates after the capability lands:** delete the S2 dev-fallback paragraph from `STEWARD_ORCH_PROMPT.md` §8.1 + `DROP_N_ORCH_PROMPT.md` Section A2 (replace with a one-line "S2 always succeeds — no dev hop"). Update memory `feedback_steward_spawn_drop_orch_flow.md` and `project_steward_auth_bootstrap.md` to drop the dev-fallback caveat. STEWARD owns the prompt + memory edits per the standard post-merge MD-write flow.
- [ ] **Per-drop wrap-up:** update CLAUDE.md (Auth and Leases section) + agent files (auth-claim sections) to reflect the new approval flow.

**Why this slot:** Drop 1 ships the `failed` lifecycle + paths/packages + auth auto-revoke — those primitives are all that the approval-cascade rule needs to wire against. Drop 1.5 ships the TUI work that surfaces the new auth-log audit trail cleanly. Drop 2 starts hierarchy refactor (kind collapse + drop rename); doing the auth fix before Drop 2 keeps the approval-rule tests stable on today's kind vocabulary, then Drop 2 sweeps the renames through them. Slotting between 1.5 and 2 is the lowest-risk window.

**Pre-1.6 workaround (current state, pre-merge of this drop):** STEWARD and drop orchs surface every subagent auth request to the dev in chat when today's `approve` call is rejected. Friction is real but bounded — each cycle is one chat round-trip. The workaround is documented in both orch prompts under the relevant auth sections so it survives across compactions.

### 19.2. drop 2 — Hierarchy Refactor

This is the touchiest code change. Each step ripples through 5+ packages. Incremental, `mage ci` after each step. Hylla reingest remains drop-end only (§9.7) — run the full drop-2 reingest inside Drop 2's `closeout` action item, not per-schema-migration-step.

**Pre-Drop-2 state (landed in Drop 1.75):** `kind_catalog` is a closed 12-value enum on `action_items.kind`; `projects.kind` column is dropped; `branch` and `phase` kinds are rewritten to `plan` by `main/scripts/drops-rewrite.sql`; `scope` is mirrored from `kind`; `metadata.role` is NOT yet a first-class field (role still lives in description prose). Drop 2 picks up from there.

- [ ] **Promote `metadata.role` to a first-class domain field.** Values: `builder`, `qa-proof`, `qa-falsification`, `qa-a11y`, `qa-visual`, `design`, `commit`, `planner`, `research`. Migration: parse `Role: <name>` from every existing action_item description and hydrate `metadata.role` at rename-time. `research` has no pre-collapse kind to hydrate from — new research action items post-Drop-2 set it at creation time.
- [ ] **Rename `done` → `complete`**: DB column values, domain `StateComplete`, TUI labels, MCP normalization, templates, docs. Combine with any leftover `failed` state migration since they touch the same surfaces.
- [ ] **Allow infinite `plan` nesting**: update domain validation to allow `plan`-under-`plan`. Update TUI tree rendering. (Kind collapse in Drop 1.75 already removed the branch/phase nesting barrier; Drop 2 enforces the remaining invariants on `action_items.kind`: `build` is leaf-only, `human-verify` carries no `plan`/QA children, `plan-qa-*`/`build-qa-*` appear only as auto-children of their matching parent kind.)
- [ ] **Dotted-address fast-nav (CLI + MCP read paths, TUI bindings in drop 4.5)** *(§1.4 convergence landing in drop 2)*. Implement dotted-address resolution (`N`, `N.M`, `N.M.K`, `<proj_name>-<dotted>`) as a pure resolver in `internal/domain` or `internal/app`: takes a dotted string + project context, returns the UUID of the matching node (or an error if ambiguous/missing). Wire into `till.action_item(operation=get)` and any other MCP read operation that takes an action-item identifier — accept either UUID or dotted form. Wire into CLI read commands the same way. **Dotted addresses remain read-only**: all mutation paths (`till.action_item(op=update|move|create)`, `till.comment(op=create)`, etc.) reject dotted form and require UUIDs.
- [ ] **Commit cadence — discussion item (`kind: discussion`).** Current default commits after every build QA pass, which is too aggressive. Converged proposal: auto-generate a `commit` child under any `plan` at level ≥ 2 whose subtree contains at least one `build` in `complete` state, fired at plan close-time (not creation) so no-build plans (pure research / discussion / refinement) don't accrue orphan commit items. Templates set `auto_commit = false` per-kind for local-only outputs (SQL migration scripts, scratch work). **Edge cases to resolve in the discussion**: multi-PR Drops (one commit per plan vs one per Drop), cascade-owned vs dev-owned commit, partial-success subtrees (some `build` children complete, some failed), and the hard-fast-rule-vs-template-override balance. Convergence lands on the discussion item's description before Drop 2 `commit` wiring in Drop 4 (§19.4).
- [ ] **`scope` column — defer removal to a future refinement drop.** Drop 1.75 mirrored `scope` from `kind` to avoid a ~20-file Go ripple (`ActionItem` struct, `NewActionItem` validator, `DefaultActionItemScope`, `IsValidWorkItemAppliesTo`, SQL column + indexes, MCP request schemas). Drop 2 adds a follow-up refinement bullet (log in REFINEMENTS) to strip the column Hylla-assisted in a later dedicated drop. No work in Drop 2 itself.
- [ ] **Research as a standalone kind in documentation and agent prompts.** `research` is its own `action_items.kind` (not a role on a plan) — read-only investigation, no auto-QA children, posts findings in a closing comment, dies. Surface this in agent-file docs, `go-research-agent` subagent prompt, and the §19.1.6 auth approval cascade's non-orch role list so orch self-approval covers research subagent auth from day one.
- [ ] **Per-drop wrap-up:** update CLAUDE.md + agent files for the `metadata.role` promotion and `done → complete` rename.

**Order matters:** `metadata.role` first (no state-machine changes, pure metadata migration), then `done → complete` (touches state machine + `failed` state), then infinite-nesting validation (depends on the new kind vocabulary already in place from Drop 1.75).

### 19.3. drop 3 — Template Configuration

**Cascade Vocabulary Adoption (dev direction, 2026-04-17).** Before template configuration can be encoded cleanly, the node vocabulary needs a stable classification that templates can bind to. "Drops all the way down" makes the shape unambiguous but is painful to talk about — dotted addresses (`0.1.5.2`) help readers but not speakers. Drop 3 adopts a **waterfall metaphor** that aligns the branding (`Tillsyn Cascade`) with the node-type axis templates need. Concretely: `drop` remains the level_1 cascade step (parallelizable, one bare worktree + branch + drop-orch per drop); a new `segment` concept names a parallel execution stream within a drop (fan-out); a `confluence` is a merge/integration item where multiple segments or drops converge; a `droplet` is the atomic, indivisible leaf action. These four classifications live on `metadata.structural_type` (closed enum, NOT open-ended) and are orthogonal to `metadata.role` (builder / qa-proof / qa-falsification / planner / commit / design / …). Templates `child_rules` and gate rules bind on `structural_type`, not on the collapsed `kind=drop`. This lands at the **start of drop 3** because every bullet below depends on having a stable classification vocabulary to bind rules against.

- [ ] **Add `metadata.structural_type` as a first-class enum field on every non-project node.** Closed 4-value enum: `drop | segment | confluence | droplet`. Not customizable initially — branding cohesion and agent-context-budget discipline win over adopter flexibility. Escape hatch (`metadata.structural_subtype` free-form string) stays deferred until a concrete adopter use case forces it. Validation at the `till.action_item(operation=create|update)` boundary rejects unknown values. Default is NOT inferred — the creator (planner / orch / dev) chooses explicitly. **`action_item` is NOT a structural_type value** — it is the generic node concept (renamed from `action_item`, see next bullet).
- [ ] **Define cascade semantics per structural_type (waterfall metaphor — single canonical source in WIKI glossary, pointers from every other doc).**
  - `drop` — vertical cascade step. Level_1 children of the project are always drops; deeper drops are sub-cascades. Parallelizable across siblings when path/package blockers allow. Best practice: one bare worktree + branch + drop-orch per level_1 drop.
  - `segment` — parallel execution stream within a drop. The fan-out unit. Segments within a drop run in parallel; segments across drops coordinate via `till.handoff`. A segment may recurse (segment within segment) when a sub-stream needs its own fan-out.
  - `confluence` — merge/integration node. Pulls work from multiple segments or sibling drops and produces the integrated output. Always has non-empty `blocked_by` naming the upstream segments/drops. The plan-QA-falsification pass attacks empty-`blocked_by` confluences and confluences whose `blocked_by` doesn't cover every upstream contributor.
  - `droplet` — atomic, indivisible leaf action. MUST have zero children. The plan-QA-falsification pass attacks droplets-with-children (misclassification → should be segment or drop).
- [ ] **Rename `action_item → action_item` across every surface.** DB schema (table + every FK + every index referencing the old name), Go domain types (`ActionItem → ActionItem`), MCP tool names (`till.action_item → till.action_item`), CLI commands, every doc (PLAN / WIKI / README / CLAUDE / STEWARD_ORCH_PROMPT / every agent file under `.claude/agents/` / every memory file), every in-tree script. **Ordering matters: run the SQL schema migration against every live DB BEFORE bringing up the renamed binary, otherwise the new code boots against an unmigrated schema and crashes.** Migration sequence: (1) dev writes + stages the migration SQL, (2) dev stops every running `till serve-mcp` process across every worktree, (3) dev runs the SQL against every affected DB (dev workstation live DB + any fixture DBs used by tests + any worktree-local DB), (4) dev applies the code rename + MD sweep, (5) new binary comes up against the already-migrated schema. Single migration droplet inside drop 3 owning the full sweep — do not partial-migrate and leave both names live. Backward-compat shim for the old MCP name stays off the table because the tool is self-hosted dogfood — only this project consumes it today.
- [ ] **Template binding by `structural_type`, not by `kind`.** After Drop 2's kind collapse (everything non-project is `kind=drop`), templates bind `child_rules`, gate rules, validation constraints, and agent bindings on the `structural_type` axis instead. A template declares: "a `drop` auto-creates a `planner` droplet + `qa-proof` droplet + `qa-falsification` droplet"; "a build `droplet` auto-creates `qa-proof` + `qa-falsification` sibling droplets"; "a `confluence` requires non-empty `blocked_by`"; and so on. Kind stays binary (`project | drop`); structural_type carries the semantic shape.
- [ ] **Retroactive classification of existing action_items (one-shot SQL + TUI verification).** Every non-project node alive pre-drop-3 gets a `metadata.structural_type` assignment:
  - Generic containers without a clean cascade shape — especially STEWARD's six persistent level_1 parents (`DISCUSSIONS`, `HYLLA_FINDINGS`, `LEDGER`, `WIKI_CHANGELOG`, `REFINEMENTS`, `HYLLA_REFINEMENTS`) — stay as plain `action_item`s with `metadata.persistent: true`. They are NOT drops, NOT segments, NOT confluences, NOT droplets. They are long-lived coordination anchors and don't need structural_type at all. The plan-QA-falsification pass accepts "no structural_type + metadata.persistent=true" as a valid shape.
  - Numbered level_1 drops (`DROP_1`, `DROP_1.5`, etc.) → `structural_type: drop`.
  - Level_2 findings items (`DROP_N_HYLLA_FINDINGS`, `DROP_N_LEDGER_ENTRY`, etc.) under STEWARD's persistent parents → plain `action_item` with `metadata.persistent: false` + `metadata.dev_gated: false` — they're feed slots, not cascade nodes.
  - Current per-drop tasks (planner / builder / qa-*) → `structural_type: droplet`.
  - Per-drop integration/merge nodes (if any) → `structural_type: confluence`.
  - Deferred: segments don't exist in today's data (no explicit fan-out nodes pre-drop-3), so no retroactive segment classification.
- [ ] **WIKI glossary as single canonical source for cascade vocabulary.** Add a dedicated `## Cascade Vocabulary` section to `main/WIKI.md` owning: structural_type enum + each value's definition + atomicity rules (droplet has zero children, confluence has non-empty `blocked_by`, segment can recurse) + relationship to `metadata.role` (orthogonal axes) + worked examples ("make a confluence action_item at `0.2.5` merging work from `0.2.3` and `0.2.4` to produce the integrated test run"). Every other doc — PLAN, README, CLAUDE, STEWARD_ORCH_PROMPT, agent prompt files, bootstrap skills — holds a **pointer** to the WIKI section, not a duplicate definition. Drift risk mitigation: QA-falsification sweeps any vocabulary redefinition outside the glossary as a mitigation-required finding.
- [ ] **Plan-QA-falsification attack surface additions.** Teach the plan-qa-falsification pass (prompt + checklist) these new attack vectors:
  - **Droplet-with-children** — any `structural_type=droplet` action_item with one or more children is a misclassification; real shape is segment or drop.
  - **Segment path/package overlap without `blocked_by`** — sibling segments within the same drop sharing `paths[]` entries or `packages[]` entries without an explicit `blocked_by` between them are a race waiting to fire. Same rule as file/package locking on `build` action items, applied at the segment level.
  - **Empty-`blocked_by` confluence** — confluence without non-empty `blocked_by` is a definitional contradiction. Flag and refuse.
  - **Confluence with partial upstream coverage** — confluence whose `blocked_by` doesn't name every segment/drop it claims to integrate. Planner must list every upstream.
  - **Role/structural_type contradiction** — role=`qa-proof` on a non-droplet; role=`builder` on a confluence; role=`planner` on a droplet without a downstream integration target. Each combination has narrow legitimate shapes.
- [ ] **Adopter bootstrap updates (`go-project-bootstrap` + `fe-project-bootstrap` skills + every `CLAUDE.md` template).** Every new project adopting Tillsyn post-drop-3 must inherit the cascade glossary pointer at bootstrap time. Bootstrap writes a WIKI scaffolding with the `## Cascade Vocabulary` section pre-filled from a template-controlled source, plus a CLAUDE.md pointer line: *"Cascade vocabulary canonical: `WIKI.md` §`Cascade Vocabulary`."* Every agent file under `.claude/agents/` (builder / qa / planning variants for both languages) gets a one-line reminder in its frontmatter body: *"Structural classifications (drop | segment | confluence | droplet) live in WIKI glossary — never redefine."*
- [ ] **Per-drop wrap-up for cascade vocabulary specifically.** After the rename + enum + template binding land, sweep every lingering `action_item` / `action-item` / `action item` / `ActionItem` string across docs, agent prompts, slash-command files, skill files, and memory files. Update `metadata.role` vs `metadata.structural_type` crosswalk wherever docs previously conflated role with kind. Commit the sweep as a final docs-only droplet under drop 3.

- [ ] Add agent binding fields to kind definitions (`agent_name`, `model`, `effort`, `tools`, etc.).
- [ ] Add gate definitions to kind templates.
- [ ] Add `max_tries`, `max_budget_usd`, `max_turns`, `auto_push`, `commit_agent` to kind definitions.
- [ ] Add `blocked_retries`, `blocked_retry_cooldown` to kind definitions.
- [ ] Template parsing and validation for new fields.
- [ ] Build a fresh `default-go` template (or equivalent) aligned to the cascade — do not resurrect the current bloated one.
- [ ] **New `steward` orch `principal_type` + auth-level state-lock** (see §15.7). Add `principal_type: steward` to Tillsyn's auth model as an orchestrator variant distinct from the generic `agent` principal type. Enforce at the auth layer: sessions whose `principal_type != steward` are rejected when attempting any `till.action_item` state transition on an item whose `metadata.owner = STEWARD`. Drop-orchs keep `create` + `update(description/details/metadata)` permissions on STEWARD-owned items but literally cannot move them through state. Replaces the pre-Drop-3 honor-system rule in the prompts.
- [ ] **Template auto-generation of STEWARD-scope items on every numbered-drop creation.** When `DROP_N_ORCH` creates a new level_1 numbered drop, template `child_rules` must auto-create the five level_2 findings drops under the persistent STEWARD parents (`DROP_N_HYLLA_FINDINGS` / `DROP_N_LEDGER_ENTRY` / `DROP_N_WIKI_CHANGELOG_ENTRY` / `DROP_N_REFINEMENTS_RAISED` / `DROP_N_HYLLA_REFINEMENTS_RAISED`) AND the refinements-gate item inside the drop's tree (`DROP_N_REFINEMENTS_GATE_BEFORE_DROP_N+1`). Each auto-generated item lands with `metadata.owner = STEWARD`, `metadata.drop_number = N`, and the correct `blocked_by` wiring on the refinements-gate (every other drop N item + the five level_2 findings drops).
- [ ] **Template-defined STEWARD-owned drop kind(s).** Templates must allow marking specific kinds as STEWARD-owned — drop-orchs can create + edit `description` on them, but only `steward`-principal sessions can transition their state. Pair with the `principal_type: steward` gate above.
- [ ] **Per-drop wrap-up:** update CLAUDE.md + agent files.

### 19.4. drop 4 — Dispatcher Core

The minimal dispatch loop, now that lifecycle + template fields exist.

- [ ] Refactor path logic from TUI to backend (18.4 output).
- [ ] **First-class project-node fields the dispatcher reads** *(prerequisite; replaces the old single-field `project_dir` bullet)*. Add these as domain-level fields on `Project`, not metadata JSON, each with explicit validation: `hylla_artifact_ref` (string, e.g. `github.com/evanmschultz/tillsyn@main`), `repo_bare_root` (abs path), `repo_primary_worktree` (abs path — the `cd` target for dispatched agents; supersedes the old `project_dir` concept), `language` (enum matching agent variants: `go`, `fe`, …), `build_tool` (string: `mage`, `npm`, `cargo`, …), `dev_mcp_server_name` (string — which MCP server dispatched agents register against). Planner fills these at project create; dispatcher reads them to spawn agents with correct `cd`, correct `{lang}-builder-agent` / `{lang}-qa-*-agent` variant, correct artifact ref in the prompt, correct MCP server registration. Fields *not* on the project node: `agent_bindings` + `post_build_gates` + kind vocabulary → template-scope (drop 3); `current_drop` → already encoded by the `kind=drop` action item in `state=in_progress`, no field needed; `go_version` → derive from `go.mod`, don't duplicate. Depends on the drop 1 metadata-validation tightening so unknown keys surface as errors instead of silent drops.
- [ ] Implement dispatcher: LiveWaitBroker subscription for state changes.
- [ ] Implement agent spawn: `cd <project_dir> && claude --agent <type> --bare -p "..." --mcp-config <per-run mcp.json> --strict-mcp-config --permission-mode acceptEdits --max-budget-usd <N> --max-turns <N>`.
- [ ] Implement agent lifecycle: auth issuance, process monitoring, cleanup on state change.
- [ ] Implement auto-promotion: when blockers clear, move eligible items to `in_progress`.
- [ ] File-level AND package-level blocking: lock acquisition, conflict detection, dynamic `blocked_by` (Section 5).
- [ ] Git status pre-check before builder dispatch (reuse the drop 1 gate).
- [ ] Gate execution: run template-defined gates after builder completion.
- [ ] Commit agent (haiku) integration for commit message formation.
- [ ] Git commit + optionally push as programmatic step — with the no-fallback, orchestrator-escalation path from Section 9.3. **Dispatcher writing to git is still opt-in at this drop;** dev can leave it off and keep managing git manually until confidence lands.
- [ ] Hylla reingest programmatic integration — **drop-end only**, as part of the Drop's `closeout` action item flow. Not a per-`build` gate (see §9.7).
- [ ] **Per-drop wrap-up:** update CLAUDE.md + agent files.

### 19.4.5. drop 4.5 — Frontend + TUI Overhaul (Concurrent with drop 5 Dogfooding)

**Track model.** drop 4.5 runs on a parallel track, kicked off alongside drop 5. Depends on drop 1 (`paths` / `packages` / `files` / `failed` state / `start_commit`-`end_commit` / creation-gated-on-clean-git) and drop 4 (dispatcher core — dispatch events are the event source the TUI subscribes to). Does **not** block drop 5 from starting, but drop 5's dogfooding informs drop 4.5's TUI ergonomics and vice versa. Dev direction (2026-04-16) for the early start: "starts early to inform TUI direction."

**Scope.** All TUI + mention-routing UX work consolidates here so the TUI ergonomics land as a coherent overhaul rather than scattered bullets across drops 5-9.

- [ ] **File viewer with `charmbracelet/glamour`** *(§24)*. TUI gains a file-viewer pane that renders drop-attached files via `glamour` — markdown with theme-aware rendering, syntax-highlighted fenced code blocks via glamour's chroma integration. Reads the drop's `files []string` (drop 1). Keybindings: open file-picker to attach, jump to attached file list, cycle through attached files, toggle glamour render vs. raw view. See §24 for the full viewer design.
- [ ] **Path picker + file picker (TUI)** *(§24)*. Two distinct pickers: path-picker for edit-scope (`paths`), file-picker for reference attachments (`files`). Both navigate the repo tree, filter by glob, multi-select. Path-picker enforces file-exists validation and package inference (derive `packages` from selected paths for `build` action-item creation). File-picker is laxer — any repo-tracked file is attachable.
- [ ] **Git-diff-per-action-item against `start_commit`** *(§24)*. For any drop with `start_commit` set, render a split-view diff of the drop's `paths` against `start_commit..HEAD` (or `start_commit..end_commit` if complete). Uses go-git or shells out to `git diff`; rendering via glamour's diff lexer. Live-updates on file-watcher events while the drop is `in_progress`.
- [ ] **Dotted-address fast-nav TUI bindings** *(§1.4 + drop 2 follow-through)*. Consume the read-only dotted-address resolver landed in drop 2. Keybinding (default `g` for "go-to"): prompt accepts `0.1.5.2` or `tillsyn-0.1.5.2`, resolves to action-item UUID, jumps focus + scrolls tree to that node. Handles `<proj_name>-<dotted>` cross-project form by switching project context first.
- [ ] **Mention-routing UX** *(§23)*. TUI surface for the `@`-mention routing system defined in §23. Inline comment composer that autocompletes `@dev` / `@builder` / `@qa` / `@qa-proof` / `@qa-falsification` / `@orchestrator` / `@research` / `@human`; on post, the comment is wired into the Tillsyn mention-routing backend so addressed parties see it in their Action Required list. **Requires a planning subdrop** (per §2.2 START bracketing rule) because the system-prompt decisions for dispatched cascade agents to answer `@`-addressed comments are load-bearing and must be confirmed with dev before builder fires.
- [ ] **Per-drop wrap-up:** update CLAUDE.md + agent files for the new TUI surface.

### 19.5. drop 5 — Cascade Planning (Dogfooding Begins)

From here on, the cascade can build itself.

- [ ] Planner agent integration: agent creates child action items via MCP.
- [ ] Planning QA integration: plan-qa-proof and plan-qa-falsification auto-fire.
- [ ] Plan acceptance flow: plan QA pass → children become eligible.
- [ ] Template child_rules for `plan` → `plan-qa-proof` + `plan-qa-falsification`, and `build` → `build-qa-proof` + `build-qa-falsification`.
- [ ] Validate the cascade flow end-to-end on real Tillsyn work (start dogfooding).
- [ ] **Per-drop wrap-up:** update CLAUDE.md + agent files.

### 19.6. drop 6 — Escalation

- [ ] Retry tracking per action item (attempt count in metadata).
- [ ] Re-fire on failure (up to max-tries).
- [ ] External failure detection + blocked retries (max-tries + 2).
- [ ] Escalation up: failed `build` → new `plan` action item at parent level.
- [ ] Escalation tracking: plan diff, failure context documentation.
- [ ] Template configuration for escalation_enabled.
- [ ] **Per-drop wrap-up:** update CLAUDE.md + agent files.

### 19.7. drop 7 — Error Handling and Observability

- [ ] Detect external failures (network, API limits, resource exhaustion).
- [ ] Distinguish `blocked` (external) from `failure` (agent error).
- [ ] Stale process detection (auth TTL expiry).
- [ ] Attention item routing for different failure types.
- [ ] Failure communication to human (specific error details).
- [ ] **Per-drop wrap-up:** update CLAUDE.md + agent files.

### 19.8. drop 8 — Wiki / Ledger System

- [ ] Design wiki storage (comments vs metadata vs dedicated table).
- [ ] Wiki agent: fires after plan acceptance and after completion.
- [ ] Hierarchical wiki absorption (child → parent summarization).
- [ ] Orchestrator memory integration.
- [ ] **Per-drop wrap-up:** update CLAUDE.md + agent files.

### 19.9. drop 9 — Quality and Vulnerability Checking

- [ ] Design quality check agent and certificate.
- [ ] Hylla graph-nav-based resource lifecycle verification.
- [ ] Configurable replicas (N agents, consensus policy).
- [ ] Standalone mode (independent quality scans).
- [ ] Go-specific checks (goroutine lifecycle, error handling, defer patterns).
- [ ] **Per-drop wrap-up:** update CLAUDE.md + agent files.

### 19.10. drop 10 — Refinement Cleanup (post initial dogfood)

After real dogfooding reveals what works and what doesn't.

- [ ] **Cascade concurrency soft-cap — promote from hard-coded N=6 to template-configurable field** *(§12.1 refinement; ToS convergence §22)*. Replace the hard-coded constant in the dispatcher with a template field (`max_concurrent_agents`) that defaults to N=6 for Max $200 subscribers and to a lower value for Max $100 subscribers. The default is read from the account-tier signal the dispatcher infers from `claude auth status` at startup (if inference is unreliable, fall back to a configured per-install default). QA-proof + QA-falsification required — misconfigured caps are the fastest path to ToS-grey-zone behavior.
- [ ] **API-key backend support for users without a Max subscription** *(§22 refinement)*. Add a dispatcher path that spawns agents via the Anthropic API directly when `CLAUDE_API_KEY` is configured on the project, rather than via `claude` CLI + `setup-token`. Covers users without Max $100/$200 subscriptions. Requires an API-specific prompt adapter (the CLI's system prompt + tool affordances differ from raw API) and a cost-tracking adapter since the API bills per-token instead of per-subscription. Gate: user explicitly sets `use_api_key: true` in project config; the dispatcher never falls through to API-key mode silently.
- [ ] **OpenAI-compatible models via Claude Agent SDK as alternate backend** *(§22 refinement)*. Add a third backend option: dispatch agents via the Claude Agent SDK pointed at an OpenAI-compatible model endpoint (e.g. local Ollama, OpenRouter, third-party providers). Same cost-tracking + prompt-adaptation work as the API-key backend. Scope note: this is about *Tillsyn being model-agnostic for users who can't or don't want to use Anthropic*, not about abandoning Claude for the cascade's canonical implementation — the cascade's QA certificate structure and prompt engineering stay Claude-tuned; OpenAI-compat models are best-effort.
- [ ] **Headless-only-for-Max-plans gating in user-facing compliance doc** *(§22 refinement)*. Create `main/TOS_USER_COMPLIANCE.md` (or similar) that states plainly: pure-headless cascade dispatch requires a Max $100 or $200 subscription; other users must use the API-key or OpenAI-compat backends (bullets above). Link from `main/README.md` and `main/CONTRIBUTING.md`. Tillsyn does not itself enforce this — the user is responsible for their Anthropic ToS posture — but the docs must not be ambiguous about it.
- [ ] **User-side ToS compliance story in README + CONTRIBUTING** *(§22 refinement)*. Thread a short "Anthropic ToS posture when using Tillsyn with Claude" section into `main/README.md` and `main/CONTRIBUTING.md`: explains the three backends (Max CLI, API key, OpenAI-compat via Agent SDK), which ones sit in Anthropic's supported use-case envelope vs. user-responsibility territory, how training opt-out should be verified, and where the authoritative verbatim Anthropic quotes live (§22 of this plan). Keeps the user-facing framing honest without turning `README.md` into a legal doc.
- [ ] Second-pass review of the cascade-bound template: trim unused fields, align with what the cascade actually reads after shipping.
- [ ] Second-pass review of `~/.claude/agents/*.md`: trim to what cascade-dispatched agents actually need, keeping orchestrator-side agents separate.
- [ ] Second-pass review of `CLAUDE.md` (global + project): remove items the cascade now handles.
- [ ] Shrink orchestrator-side slash-command + skill surface to the minimum needed after cascade takes over most coordination.
- [ ] **Full `magefile.go` cleanup + refine pass** *(deferred from drop 0)*. drop 0 added `mage test-func` (18.3) and `mage install` with commit pinning (18.5) as point additions without touching the rest. Do a full sweep: consolidate duplicated invocation helpers, normalize target naming (`test-pkg` vs `test-func` vs `test-golden` vs `ci` — are the prefixes + argument shapes consistent?), prune any dead or stub targets, verify every target has a one-line `mg:` doc comment, confirm no target shells out to a raw `go` command (force everything through a single `runGo` helper), and make sure `mage -l` output reads like a coherent menu. QA-proof + QA-falsification required — the magefile is the build gate, can't silently break.
- [ ] **`mage install` post-MVP retire-or-reshape** *(deferred from drop 0 fix-finish)*. drop 0's closeout reshaped `mage install` to the minimum dogfood-safe shape: single required positional arg `sha` (never resolved from `git HEAD`, empty errors out), temp `git worktree add --detach <sha>`, `go build` with `-X ...buildinfo.Commit=<sha>` ldflag, install binary to hardcoded `$HOME/.tillsyn/till` colocated with `config.toml` / `tillsyn.db` / `logs/`, defer cleanup of both the worktree and the temp root. **Enforcement for "dev-only, never agents" is docstring + `CLAUDE.md` rule only** — no tool-permission deny, no env-marker guard, no code-level check. This target exists solely because we're pre-MVP and need a stable `till` on the dev box to orchestrate the cascade against. Once MVP ships with proper release artifacts (goreleaser snapshot is already wired in CI), decide: retire `mage install` entirely in favor of `gh release download` + install, or keep it as a dev convenience with the shape above. Whichever: no pin-log file, no printer notice, no helpers — if it stays, it stays at ~30 lines.
- [ ] **Agent tool-permission deny for `mage install`** *(refinement-only, add when needed)*. If an agent ever tries to invoke `mage install` despite the CLAUDE.md rule, add `Bash(mage install*)` to the `tools: { deny: [...] }` block in every `~/.claude/agents/*.md` (builder, QA, research, planning variants) and record the incident so we know the docstring-only enforcement was insufficient. Until that happens, the docstring rule is load-bearing and good enough — no proactive guardrail work.
- [ ] **Project lifecycle operations — delete + archive** *(gap surfaced in drop 0)*. Add `till.project(operation=delete)` MCP op and corresponding `till project delete` CLI. Guard: project must have no active auth sessions or leases, no in-flight cascade runs. Must cascade-clean action items, comments, handoffs, attention items, template bindings, embeddings, capture snapshots — audit FK coverage and add explicit cleanup where `ON DELETE CASCADE` is missing. Also add `till.project(operation=archive)` MCP op + `till project archive` CLI that flips the archived flag already surfaced in `include_archived` list filter (preserves data, hides from default list). drop 0 worked around the missing delete by renaming the messy pre-cascade project to `TILLSYN-OLD`; retire that renamed project via `delete` (or `archive` if we want to keep the old data for comparison) once these ops ship.
- [ ] **Compaction-resilient auth cache — refinement pass** *(MVP shipped in drop 0 as 18.11; this is the follow-up hardening)*.

  **Background — what drop 0 shipped.** The MVP solves orchestrator auth loss across context compaction using a `SessionStart` hook + a per-project file cache. No keychain. The full design:

  - **Cache file**: `~/.claude/tillsyn-auth/<project-uuid>.json`, mode 0600, inside `~/.claude/tillsyn-auth/` mode 0700. One file per `(project-uuid, role)`, overwrite-in-place on every fresh claim, no stale accumulation.
  - **Payload**: `{project_id, role, session_id, session_secret, auth_context_id, agent_instance_id, lease_token, request_id, expires_at, claimed_at}`. Timestamps are RFC3339 UTC `Z` form with fractional seconds stripped.
  - **Write path — orchestrator behavioral rule**: on every successful orchestrator-role `till.auth_request(operation=claim)` + subsequent `till.capability_lease(operation=issue)`, the orchestrator `Write`s the full bundle to cache **before** any other Tillsyn work. Enforced by auto-memory `feedback_orchestrator_auth_cache.md` — orchestrator-scope only; subagents never load this memory, never write or read the cache.
  - **Read path — SessionStart hook**: `~/.claude/hooks/session_start_tillsyn_auth_inject.sh` (registered in `~/.claude/settings.json` under `hooks.SessionStart` with `matcher: "startup|resume|compact"`). On fire, the script scans the cache dir, parses `expires_at`, deletes expired entries (reactive GC), and for valid entries emits `{"hookSpecificOutput": {"hookEventName": "SessionStart", "additionalContext": "<tillsyn-auth-cache>...</tillsyn-auth-cache>"}}`. The orchestrator sees the bundle in its first turn after resume/compact.
  - **Orchestrator use**: on seeing the injected `<tillsyn-auth-cache>` block, validate via `till.auth_request(operation=validate_session)` before use. On validation failure (revoked, server-side expired), delete the cache file and fall through to asking the dev — captured in `feedback_auth_after_compaction.md`.
  - **Subagent isolation model**: nondiscoverability, not access control. The cache path, hook script path, and read/write commands appear only in orchestrator auto-memory under `~/.claude/projects/<parent-session-hash>/memory/`. They are **never** in any `CLAUDE.md` (global or project), any `~/.claude/agents/*.md`, any slash command, or any spawn prompt template. `SessionStart` hooks fire only for top-level CLI sessions — Agent-tool subagents don't fire them, so `additionalContext` injection reaches only the orchestrator. A Bash-equipped subagent could still `ls ~/.claude/` and find it; isolation is sufficient for accident prevention, not against a rogue subagent.
  - **TTL'd secret**: every entry carries `expires_at` and self-deletes on expired read; even a leaked cache file is short-lived.
  - **Why file + hook and not keychain**: keychain has no native TTL, macOS-only, touches the user's personal keychain, and the ACL-by-app advantage doesn't apply from inside the Bash tool. File-backed is simpler, portable across platforms, auditable, naturally TTL'd, and has equivalent practical isolation.

  **This is the documented generalized solution for any Tillsyn auth-loss-through-compaction issue.** Any future role (builder, qa, research) facing the same problem uses the same pattern — different filename suffix, same hook script, same behavioral rule in a role-specific memory file.

  **Refinement work for this drop (not MVP):**

  - [ ] **Extend to subagent roles** if subagent compaction becomes a real pattern. Today subagents are per-spawn and don't compact, so builder/qa/research caches are out of scope; revisit if that changes.
  - [ ] **Harden subagent isolation**: consider whether to move cache under a less-enumerable path, or whether to use a deny-list in subagent permissions to block `Bash:ls ~/.claude/tillsyn-auth/` and `Read:~/.claude/tillsyn-auth/**`. Probably YAGNI unless we see a real leak; recorded here so the option is known.
  - [ ] **Expired-entry reaper**: today GC is reactive (on read). Add a lightweight sweep hook (`SessionEnd` or cron-equivalent) if files pile up — also probably YAGNI given overwrite-in-place semantics, but recorded.
  - [ ] **Cache-hit telemetry**: a `SessionStart` emit line the hook writes to `~/.claude/hooks/hook-execution.log` so we can see how often the cache saves a round-trip vs. falls through. Useful for validating the design holds up.
  - [ ] **Cross-project orchestrator sessions**: today the hook injects every valid bundle. If the orchestrator works on multiple projects concurrently it gets multiple bundles injected; fine today, may want cwd-scoped filtering later.
  - [ ] **Port to Linux / Windows** if the project ever runs there — the file semantics are portable; only the `date -j -u -f` parsing in the hook script is macOS-specific and needs a GNU-date alt path.
  - [ ] **Integration with `till auth session show`**: today the CLI's `show` command redacts the secret. If a user wants to seed the cache for a pre-existing session, they can't. Add `till auth session reveal --session-id <id>` (interactive, requires TTY, logs an attention item) so seeding works without re-claiming. Lower priority — the cache is normally written at claim time where the secret is already in hand.
- [ ] **Cascade Tree Structure docs relocation — MVP docs prep** *(surfaced during 2026-04-17 CLAUDE.md size-cleanup sweep)*. The authoritative "Cascade Tree Structure (Template Architecture)" section — kind hierarchy diagram, required-children rules, agent bindings table, post-build gates, blocker semantics, state-trigger dispatch, pre-Drop-2 creation rule — currently lives in both `main/CLAUDE.md` and the bare-root `CLAUDE.md`. It's not duplicated anywhere in `PLAN.md` today. Before MVP release, relocate the canonical text into `PLAN.md` (natural home: a new top-level section near §3 "The Cascade Model" or as an explicit §3.x "Template Architecture by Kind") and shrink both `CLAUDE.md` bodies to a short summary + pointer. Rationale: `PLAN.md` is the documented source of truth for cascade architecture (per each `CLAUDE.md`'s "Cascade Plan" pointer); having the tree structure live in `CLAUDE.md` instead is a structural inversion that will confuse OSS readers once MVP docs are written. QA-proof + QA-falsification: verify `PLAN.md` version is complete (no content dropped), both `CLAUDE.md` pointers resolve correctly, and WIKI.md `Related Files` is updated. Deferred from the 2026-04-17 sweep because doing it then would have required a large `PLAN.md` edit alongside the CLAUDE.md shrink; the sweep preserved Cascade Tree Structure in both `CLAUDE.md` files pending this refinement.
- [ ] **Cascade granularity refinements (source: `main/AGENT_CASCADE_DESIGN.md` 2026-04-18).** The design doc sets starting values hardcoded into the dogfood build and lists every dial that should become dev-tunable once we have metrics data:
  - [ ] **Role→model bindings configurable by path + action-item kind.** `AGENT_CASCADE_DESIGN.md` §3 ships a fixed binding (planner=sonnet, all QA=opus, builder=sonnet, commit=haiku). Refinement: template/project-config fields so the dev can override per path glob + per action-item kind. Configuration surface design is open — see §20.10 Q12.
  - [ ] **Nested planners + QA inside Go packages.** Pre-dogfood rule (`AGENT_CASCADE_DESIGN.md` §2.2): planners and LLM QA stop at the Go-package boundary; droplets go sub-package with `blocked_by` serialization on shared compile. Refinement: figure out how to nest planners + QA *inside* a package keyed on file-clusters or feature-slices. Unblocks finer parallelism and tighter-scoped LLM QA for large packages. Data from dogfood decides if it's worth building.
  - [ ] **Global plan-QA sweep depth threshold configurable.** `AGENT_CASCADE_DESIGN.md` §4.4 hardcodes depth ≥ 3 as the trigger for the second plan-QA pass. Refinement: promote to a template field. Starter value stays at 3; real drops tell us where the threshold actually earns its cost.
  - [ ] **Droplet LOC/file ceilings configurable per action-item kind, with planner-asks-for-permission flow.** `AGENT_CASCADE_DESIGN.md` §2.1 ships soft ~80 LOC target / ~200 LOC ceiling / ~3 files. Refinement: template fields per action-item kind (SQL migration droplet may want different ceilings than a unit-test droplet than a TUI-component droplet). When a planner genuinely can't decompose under the ceiling, the workflow should let the planner *request* permission to exceed it — captured as an attention item or structured handoff — rather than silently ignoring the rule or forcing a contrived split.
  - [ ] **Audit-trail storage strategy evaluation.** `AGENT_CASCADE_DESIGN.md` §8 ships Option X (full snapshot per change) on YAGNI grounds. Refinement: if dogfood shows edit-count × node-size bounds out uncomfortably, evaluate Option Y (diff-per-change) or Option Z (snapshot + diffs). Do not optimize until data says to.
  - [ ] **Metrics catalog → structured ledger emission.** `AGENT_CASCADE_DESIGN.md` §13 lists the metrics we want (per-droplet build-green rate, per-planner-node plan-QA pass rate, per-drop cost by tier, re-QA frequency, parallelism extraction rate, etc.). Today the ledger captures this as prose inside `DROP_N_LEDGER_ENTRY`. Refinement: define a structured JSON/TOML block embedded in each ledger entry with the full metric set, so we can aggregate cleanly for the eventual comparative benchmarks (§12 of the design doc — arxiv 2603.01896 framework).
  - [ ] **Split `AGENT_CASCADE_DESIGN.md` into concept + operations before MVP public release.** Today the doc is unified-for-now to prevent drift between the conceptual explanation (audiences: people running a similar cascade with plain Markdown + off-the-shelf subagents, and the future blog-post/article source material) and the internal pre-dogfood MD-only operations content (dev + STEWARD day-to-day reference during Drops 1.75 → 4). Before MVP, split into `docs/cascade-concept.md` (public-facing: §1 thesis, §2 droplet shape, §3 role/model bindings, §4 QA placement, §5 nesting, §6 failure handling, §7 blocker re-QA, §8 audit trail, §9 cascade tree ASCII art, §12 benchmarking framework, §13 metrics catalog) and `docs/cascade-operations.md` (internal: §10 dogfood plan, §11 affected cascade drops, §14 open questions). QA-proof + QA-falsification: verify no content dropped; every cross-reference in `PLAN.md`, `CLAUDE.md`, and other MDs updated to the new locations.
  - [x] **De-Rak-ify `main/workflow/example/` for public-release shipping.** *(Landed 2026-04-19.)* `main/workflow/example/` is now a generic cascade-workflow reference — every file uses `<PROJECT>` / `<package>` / `<org>` placeholders, no project-specific names remain. Content aligned with `AGENT_CASCADE_DESIGN.md` §2 (droplet shape), §4 (QA placement), §5 (sub-drop nesting / planner-calls-planner), §7 (ancestor re-QA on blocker failure). Structure: `example/CLAUDE.md` (generic project CLAUDE), `example/drops/WORKFLOW.md` (cascade-aware 7-phase lifecycle), `example/drops/_TEMPLATE/` (per-drop scaffold), `example/drops/DROP_N_EXAMPLE/` (concrete pedagogical walkthrough of one closed drop in a fictional generic Go project). Double-nested `drops/drops/` import bug flattened to `drops/` at the same time.
- [ ] **Per-drop wrap-up:** update CLAUDE.md + agent files.

### 19.11. drop 11 — Dispatcher Git Ownership (Post-Dogfood Refinement)

Move git responsibility from orchestrator+dev to the dispatcher.

- [ ] Dispatcher performs all commits for action items it dispatched (commit agent handles the message; no deterministic fallback; commit-agent failure escalates to orchestrator CLI tool).
- [ ] Dispatcher reads `start_commit` / `end_commit` fields to decide reingest scope.
- [ ] Orchestrator CLI tool for manual commit override when the commit agent escalates.
- [ ] Orchestrator programmatic supersede via system-issued auth (the post-dogfood supersede path from drop 1's deferred list).
- [ ] TUI rendering of `failed` tasks (deferred from drop 1).
- [ ] Update CLAUDE.md to remove the "orchestrator + dev manage git manually" language.
- [ ] **Per-drop wrap-up:** update CLAUDE.md + agent files.

### 19.12. drop 12 — `depends_on` Removal (Dogfooding Test)

- [ ] Remove `depends_on` from schema, domain, app, adapters, TUI.
- [ ] Confirm `blocked_by` + parent-child hierarchy fully replaces it.
- [ ] Intentionally last — it's a real integration test of the cascade system itself building a cascade-relevant change.
- [ ] **Per-drop wrap-up:** update CLAUDE.md + agent files.

---

## 20. Open Questions

### 20.1. Dispatcher Process Model

**Q1:** Where does the dispatcher run? Options:
- a) Inside `till serve` (HTTP server process) — available when MCP server is running
- b) Inside `till serve-mcp` (stdio MCP) — available during claude sessions
- c) Inside the TUI process — available when dev is using TUI
- d) A dedicated `till dispatch` daemon process
- e) All of the above (dispatcher is a library, embedded in all surfaces)

**Leaning:** (e) — the dispatcher is a library (`internal/dispatch/`) that any Tillsyn process can embed. It subscribes to LiveWaitBroker events, which are already cross-process.

### 20.2. Wiki Design

**Q2:** What is the wiki's shape and storage? See Section 15.4. Needs design work.

**Q3:** How does wiki content integrate with orchestrator memory compaction?

### 20.3. Quality Check Design

**Q4:** What specific Go quality checks should be in the initial set? Resource lifecycle, error handling, goroutine safety — what else?

**Q5:** How do we handle false positives in quality checks? The graph analysis might flag valid patterns as issues. What's the escalation path?

### 20.4. Escalation Depth

**Q6:** How deep can escalation nest? If a build fails → re-plan → build fails again → ?

Current design: `max-tries=2` at each level, then attention item. But what about the re-plan level? Can the re-plan itself fail and escalate further?

**Suggestion:** One level of escalation. Build fails → re-plan → build fails → human. No deeper nesting. Configurable in template.

### 20.5. MCP Config Passthrough for Headless Agents (RESOLVED)

**Q7:** Can `claude --bare -p "..." --mcp-config <path>` accept an ad-hoc MCP server list that is **not** in the dev's `settings.json`?

**Resolution: Yes.** Claude Code's headless CLI supports this directly. The flag pair to use is:

```
claude --bare -p "..." \
  --mcp-config /path/to/agent-mcp.json \
  --strict-mcp-config \
  --permission-mode acceptEdits \
  --max-budget-usd <N> --max-turns <N>
```

- `--mcp-config <path>` loads the MCP server definitions from the given JSON file (same shape as the `mcpServers` block in `settings.json`).
- `--strict-mcp-config` tells Claude Code to use **only** the servers defined in that JSON and to **ignore** `settings.json` / global / project-scoped MCP definitions. This is exactly the isolation the cascade needs.

Source: Claude Code official CLI docs via Context7 (`/websites/code_claude`).

Implication: the dispatcher writes a per-run `agent-mcp.json` (Tillsyn + Hylla + Context7, tool allow-lists tailored per agent type) and passes it via the two flags. The orchestrator's `settings.json` is untouched, and the agent sees a clean, minimal MCP surface. This lets us separate the agent MCP surface from the orchestrator MCP surface cleanly — no settings.json forking, no env-var hacks.

### 20.6. Template Kind Expansion (Resolved in Drop 1.75)

**Q8 (original):** The cascade introduces new kind types that may not exist in `default-go` yet: `plan-actionItem`, `plan-qa`, `quality-check`, `commit-agent`.

**Resolution (Drop 1.75, 2026-04-21):** `kind_catalog` collapses to a closed 12-value `action_items.kind` enum (see §1.4): `plan`, `research`, `build`, `plan-qa-proof`, `plan-qa-falsification`, `build-qa-proof`, `build-qa-falsification`, `closeout`, `commit`, `refinement`, `discussion`, `human-verify`. Plan-QA splits into its own pair of kinds (`plan-qa-proof` / `plan-qa-falsification`) rather than reusing a generic `qa-check` — same for build-QA — so template `child_rules` can bind distinct agents and gate rules per role. Quality/vuln checking (Section 16, deferred) has no dedicated kind today; when it lands it attaches as a custom sub-kind under `build` via the template-customization hook (see §1.4 Customization).

Drop 3 picks up `child_rules` wiring against this fixed 12-kind enum — no per-kind vocabulary debate left, just binding agents + gates to known kinds.

### 20.7. Orchestrator Role in Cascade

**Q9:** What does the orchestrator do during an active cascade?

Current design: orchestrator runs `/loop` polling for attention items (failures, escalations). The cascade runs autonomously until something fails. The orchestrator's job is:
- Start cascades (move drop to `in_progress`)
- Handle failures (review attention items, decide fix vs. supersede)
- Review wiki summaries
- Make design decisions the cascade can't

Is this sufficient? Does the orchestrator need more visibility during a running cascade?

### 20.8. Plan Item State Machine for Gates (RESOLVED)

**Q10:** When a build agent moves its action item to `complete` and then a gate fails, how does the action item get to `failed`?

**Resolution:** The builder moves to `complete`. Gates run. If a gate fails, the dispatcher uses **override auth** to move `complete → failed`. The `complete → failed` transition requires override auth, which the dispatcher has (system-issued). This uses existing mechanisms — no intermediate states, no new state transitions. Override auth is already designed for exactly this kind of system-level state correction.

### 20.9. Planning Agent Auth Scope

**Q11:** The planner agent needs to create child action items via `till.action_item(operation=create)`. But its auth is scoped to its own `plan` action item. Creating children on the drop (the `plan` action item's parent) requires broader scope.

Options:
- a) Planner's auth is scoped to the drop, not just its `plan` action item
- b) Planner creates children under its own `plan` action item, and the dispatcher re-parents them
- c) Planner creates children under the drop via a dedicated "create-child-on-parent" MCP operation

**Leaning:** (a) — the planner needs drop-scoped auth because its job is to decompose the drop. Template configuration specifies the auth scope for each kind.

### 20.10. Cascade Granularity Configuration (source: `main/AGENT_CASCADE_DESIGN.md`)

**Q12:** How are role→model bindings configured per path + per action-item kind? `AGENT_CASCADE_DESIGN.md` §3 hardcodes the bindings (planner=sonnet, all QA=opus, builder=sonnet, commit=haiku) for the dogfood phase. Options for the configurable form:

- a) Template field: `role_model_bindings: { planner: "sonnet", plan_qa: "opus", ... }` on the project's template binding. One binding set per template.
- b) Project-level settings: a `model_bindings.toml` (or block in `tillsyn.toml`) that lives at the project root. Per-path glob overrides inside.
- c) Per-drop override: drop kind or drop metadata can shadow the template/project default for that drop's subtree.
- d) All three layered: template default → project override → drop override → per-action-item override.

**Leaning:** (d) with sensible precedence — template sets the baseline, project overrides for the project's characteristic work (e.g., "this project does a lot of TUI work, bump builder to opus for `internal/tui/**`"), drop metadata overrides for an unusual drop, per-action-item for final surgery. Ship template-only in Drop 10's first refinement pass; add layers as real use-cases demand.

**Q13:** How do we record the dogfood metrics catalog from `AGENT_CASCADE_DESIGN.md` §13 in a machine-aggregable form? The per-droplet / per-planner-node / per-drop / comparative metrics need structured retention, not prose. Options:

- a) JSON block embedded in each ledger `description` under a stable `## Metrics` heading.
- b) A sibling file `main/METRICS/<drop-slug>.json` updated by the drop-orch at close.
- c) A dedicated `till.metrics` MCP operation that writes to a dedicated table.

**Leaning:** (a) first (cheapest, no schema work, data lives next to the narrative). Promote to (c) once the aggregator needs typed queries.

**Q14:** Droplet ceiling breach workflow. When a planner genuinely can't decompose a droplet under the ceiling (`AGENT_CASCADE_DESIGN.md` §2.1), what's the approval path?

- a) Planner marks `irreducible: true` + justification. Plan-QA validates.
- b) Planner opens an attention item asking the dev to ratify the breach.
- c) Planner creates a structured handoff to the drop-orch with the breach request.

**Leaning:** (a) is the baseline — plan-QA falsification should be able to reject unjustified irreducibility. Escalate to (b) or (c) only if we see planners abusing `irreducible: true` as an easy out.

**Q15:** Workflow-MD exit criteria. When do we retire `drops/` MDs in favor of direct Tillsyn writes? `AGENT_CASCADE_DESIGN.md` §14 Q1 recommends: after Drop 4 dispatcher lands AND at least 3 workflow-MD drops have completed. Confirm the 3-drop floor; document the retirement trigger as a refinement action item when we reach it.

---

## 21. Resources

### 21.1. Stripe Minions

- [Minions Part 1](https://stripe.dev/blog/minions-stripes-one-shot-end-to-end-coding-agents) — Stripe Dev Blog. Primary source. "The walls matter more than the model."
- [Minions Part 2](https://stripe.dev/blog/minions-stripes-one-shot-end-to-end-coding-agents-part-2) — Architecture deep-dive. Blueprints as state machines. Deterministic-agentic-deterministic sandwich. 2-CI-round hard cap.
- [Stripe Engineers Deploy Minions](https://www.infoq.com/news/2026/03/stripe-autonomous-coding-agents/) — InfoQ. Scale: 1,300+ merged PRs/week.
- [Deconstructing Stripe's Minions](https://www.sitepoint.com/stripe-minions-architecture-explained/) — SitePoint. Architecture walkthrough.
- [Blueprint Architecture Deep-Dive](https://www.mindstudio.ai/blog/stripe-minions-blueprint-architecture-deterministic-agentic-nodes) — MindStudio. Deterministic vs. agentic nodes.
- [How Stripe's Minions Ship 1,300 PRs a Week](https://blog.bytebytego.com/p/how-stripes-minions-ship-1300-prs) — ByteByteGo.
- [The walls matter more than the model](https://www.anup.io/stripes-coding-agents-the-walls-matter-more-than-the-model/) — Independent analysis.
- [Steve Kaliski Podcast](https://podcasts.apple.com/us/podcast/how-stripe-built-minions-ai-coding-agents-that-ship/id1809663079?i=1000757255000) — Stripe engineer interview.
- [Block goose](https://github.com/block/goose) — The open-source agent Stripe forked.

### 21.2. Semi-Formal Reasoning

- [Agentic Code Reasoning (arXiv 2603.01896)](https://arxiv.org/abs/2603.01896) — Ugare & Chandra, Meta. Certificates force evidence-grounded reasoning. 88.8% accuracy vs 78.2% without.
- [arXiv HTML version](https://arxiv.org/html/2603.01896v1) — Full text with appendices.
- [Emergent Mind analysis](https://www.emergentmind.com/papers/2603.01896)
- [VentureBeat coverage](https://venturebeat.com/orchestration/metas-new-structured-prompting-technique-makes-llms-significantly-better-at) — "Meta's structured prompting technique."

### 21.3. Claude Code Headless

- [Run Claude Code programmatically](https://code.claude.com/docs/en/headless) — Official docs. `-p`, `--bare`, streaming.
- [CLI reference](https://code.claude.com/docs/en/cli-reference) — All flags: `--agent`, `--worktree`, `--output-format`, `--max-budget-usd`.
- [Claude Code Subagents](https://www.morphllm.com/claude-subagents) — Agent file format and invocation.

### 21.4. Internal

- `STRIPE_MINIONS_FOR_TILLSYN_HYLLA_CONCEPT_AND_PLAN_2026-04-11.md` — Previous design doc. Mapping of Stripe concepts. Resource inventory. Partially superseded by this document.
- `MINIONS_RESEARCH_AND_FINDINGS_2026-04-13.md` — Research compilation. Source material analysis. Open questions (partially answered in this document).
- `main/TILLSYN_FIX_PROMPT.md` — Historical document listing the pre-cascade fix decisions (failed lifecycle state, outcome metadata, override auth, auth auto-revoke, `require_children_done`, action-item details as prompt). Hard prerequisites for the cascade.
- `~/.claude/agents/*.md` — Current 8+2 agent file inventory.

### 21.5. Tillsyn Code Surfaces (from Hylla)

- `internal/app/live_wait.go` — `LiveWaitBroker` interface, `LiveWaitEvent` struct, `inProcessLiveWaitBroker.Wait` method
- `internal/adapters/livewait/localipc/broker.go` — Cross-process SQLite-backed broker. `Broker.Close` manages subscribers.
- `internal/domain/capability.go` — `CapabilityLease`, `NewCapabilityLease`, `CapabilityLease.Renew`, `CapabilityLease.MatchesScope`
- `internal/app/kind_capability.go` — `HeartbeatCapabilityLeaseInput`, `RevokeAllCapabilityLeasesInput`
- `internal/domain/errors.go` — `ErrTransitionBlocked` sentinel
- `internal/adapters/server/common/auth.go` — `MutationAuthorizer` interface
- `internal/adapters/auth/autentauth/service.go` — `IssueSessionInput`
- `internal/adapters/server/server.go` — `Run`, `NewHandler` (HTTP server setup)
- `internal/adapters/server/mcpapi/handler.go` — `Handler.ServeHTTP` (MCP request handler)
- `magefile.go` — `CI` target (canonical gate)

---

## 22. Account Tier, Auth, and ToS Posture

**Status:** Converged 2026-04-15/16 (consolidation pass). Folds the content previously scattered across `main/TOS_COMPLIANCE.md` (verbatim Anthropic quote appendix) and `main/TOS_DISCUSSIONS.md` Q3 + Cross-cutting A into the plan. Pending items (Q1, Q2, Q4, Q5 from `TOS_DISCUSSIONS.md`) are threaded as refinement bullets under §19.10.

### 22.1. Dogfood Backend: Pure-Headless via Max $200

- **Subscription tier**: the cascade dogfood runs on a **Max $200** Anthropic subscription. Max $100 is a fallback (lower concurrency cap per §12.1 refinement), not the primary target. Below Max, users run the API-key or OpenAI-compat backends (§19.10 refinement bullets).
- **Dispatch mode**: pure-headless via `claude --bare -p ... --mcp-config <path> --strict-mcp-config --permission-mode acceptEdits --max-budget-usd <N> --max-turns <N>`. No interactive sessions. No human-in-the-loop prompts inside the dispatched process — all dev interaction happens on the orchestrator side via Tillsyn comments / handoffs / attention items.
- **Auth path**: `claude setup-token` on the dev's box, credentials available to the subscription's headless flow. Dispatcher never embeds or persists the Anthropic API key itself; dispatch reads whatever Claude CLI's auth store already holds.
- **Training opt-out**: verified ON at the Anthropic account level. Dogfood cannot ship with training opted-in because cascade-generated work touches in-progress architecture decisions and partially-landed dev code.

### 22.2. Concurrency + ToS

- Hard-coded N=6 during dogfood (§12.1) matches the practical ceiling for one Max $200 account running pure-headless without triggering per-account rate-limit responses that would poison the cascade's failure signal. This is not an Anthropic-published cap; it's an empirical dogfood ceiling.
- Promotion to template-configurable with account-tier-aware defaults lives in §19.10.
- Users who exceed the cap or run on lower tiers are responsible for their own account-level ToS posture — Tillsyn does not inspect or enforce Anthropic account state.

### 22.3. User-Side Posture

Tillsyn ships three dispatch backends (after §19.10 refinement lands):

| Backend | Requires | Tillsyn ToS Posture |
|---|---|---|
| **Max CLI** (pure-headless) | Max $100 / $200 subscription + `claude setup-token` | Primary path. Sits in Anthropic's supported use-case envelope for Max subscribers. |
| **Anthropic API key** | `CLAUDE_API_KEY` env + OpenRouter/direct API billing | Supported. Per-token billing; no subscription gating. User responsible for rate limits and billing. |
| **OpenAI-compat via Agent SDK** | OpenAI-compatible endpoint (Ollama local, OpenRouter, third-party) | Best-effort. Cascade's QA certificate + prompt engineering is Claude-tuned; non-Claude models may regress on structured reasoning. User responsible for model quality. |

The user is responsible for their own ToS compliance on the chosen backend. `main/TOS_USER_COMPLIANCE.md` (landing in §19.10) holds the user-facing compliance doc; §22.4 below holds the authoritative verbatim Anthropic quotes that informed the dogfood posture.

### 22.4. Verbatim Anthropic Quote Appendix

The authoritative verbatim quotes from Anthropic's published ToS / Usage Policy / Claude Code docs that informed the dogfood posture. These quotes were originally collected in `main/TOS_COMPLIANCE.md §2` and are folded here as evidence — they are not editorial summary, must not be paraphrased, and must be preserved verbatim with their source URLs intact. Retrieval date: 2026-04-14.

#### 22.4.1. Consumer Terms of Service — `https://www.anthropic.com/legal/consumer-terms`

Automation / non-human-access clause:

> "Except when you are accessing our Services via an Anthropic API Key or where we otherwise explicitly permit it, to access the Services through automated or non-human means, whether through a bot, script, or otherwise."

Training-opt-out language:

> "We may use Materials to provide, maintain, and improve the Services and to develop other products and services, including training our models, unless you opt out of training through your account settings."

#### 22.4.2. Usage Policy — `https://www.anthropic.com/legal/aup`

Agentic-use passthrough:

> "Agentic use cases must still comply with the Usage Policy."

Guardrail-bypass prohibition:

> "Intentionally bypass capabilities, restrictions, or guardrails established within our products."

Coordination / circumvention:

> "Coordinate malicious activity across multiple accounts to avoid detection or circumvent product guardrails."

#### 22.4.3. Agentic-Use support article — `https://support.claude.com/en/articles/12005017-using-agents-according-to-our-usage-policy`

The article enumerates prohibited outcomes (surveillance, phishing, scaled abuse, unauthorized system access) and does not address human-oversight requirements, autonomy bounds, or multi-agent dispatch protocols. Agentic use is permitted as long as the Usage Policy itself is respected:

> "All uses of agents and agentic features must continue to adhere to Anthropic's Usage Policy."

#### 22.4.4. Commercial Terms of Service — `https://www.anthropic.com/legal/commercial-terms`

Competing-product restriction (§D.4):

> "Customer may not and must not attempt to (a) access the Services to build a competing product or service, including to train competing AI models or resell the Services except as expressly approved by Anthropic; (b) reverse engineer or duplicate the Services; or (c) support any third party's attempt at any of the conduct restricted in this sentence."

Training on customer content (§B):

> "Anthropic may not train models on Customer Content from Services."

#### 22.4.5. Claude Code permission-modes — `https://code.claude.com/docs/en/permission-modes`

`acceptEdits`:

> "`acceptEdits` mode lets Claude create and edit files in your working directory without prompting. … In addition to file edits, `acceptEdits` mode auto-approves common filesystem Bash commands: `mkdir`, `touch`, `rm`, `rmdir`, `mv`, `cp`, and `sed`. … Paths outside that scope, writes to protected paths, and all other Bash commands still prompt."

`bypassPermissions` / `--dangerously-skip-permissions`:

> "`bypassPermissions` mode disables permission prompts and safety checks so tool calls execute immediately. Writes to protected paths are the only actions that still prompt. Only use this mode in isolated environments like containers, VMs, or devcontainers without internet access, where Claude Code cannot damage your host system."

> "`bypassPermissions` offers no protection against prompt injection or unintended actions. For background safety checks without prompts, use auto mode instead."

Auto-mode availability:

> "Auto mode is available only when your account meets all of these requirements: Plan: Team, Enterprise, or API. Not available on Pro or Max. … Model: Claude Sonnet 4.6 or Opus 4.6. Not available on Haiku or claude-3 models. Provider: Anthropic API only. Not available on Bedrock, Vertex, or Foundry."

Auto-mode rules dropped on entry:

> "On entering auto mode, broad allow rules that grant arbitrary code execution are dropped: Blanket `Bash(*)`, Wildcarded interpreters like `Bash(python*)`, Package-manager run commands, `Agent` allow rules. Narrow rules like `Bash(npm test)` carry over."

Auto-mode on subagents:

> "The classifier checks subagent work at three points: Before a subagent starts, the delegated task description is evaluated, so a dangerous-looking task is blocked at spawn time. While the subagent runs, each of its actions goes through the classifier with the same rules as the parent session, and any `permissionMode` in the subagent's frontmatter is ignored. When the subagent finishes, the classifier reviews its full action history; if that return check flags a concern, a security warning is prepended to the subagent's results."

Auto-mode fallback and headless interaction:

> "If the classifier blocks an action 3 times in a row or 20 times total, auto mode pauses and Claude Code resumes prompting. … In non-interactive mode with the `-p` flag, repeated blocks abort the session since there is no user to prompt."

Protected paths (always prompt in any mode):

> ".git, .vscode, .idea, .husky, .claude (except for .claude/commands, .claude/agents, .claude/skills, and .claude/worktrees) … .gitconfig, .gitmodules, .bashrc, .bash_profile, .zshrc, .zprofile, .profile, .ripgreprc, .mcp.json, .claude.json"

#### 22.4.6. Claude Code CLI reference — `https://code.claude.com/docs/en/cli-reference`

Long-lived auth for CI / scripts:

> "`claude setup-token` — Generate a long-lived OAuth token for CI and scripts. Prints the token to the terminal without saving it. Requires a Claude subscription."

Headless flags used by the plan:

> "`--bare` — Minimal mode: skip auto-discovery of hooks, skills, plugins, MCP servers, auto memory, and CLAUDE.md so scripted calls start faster."
>
> "`--max-budget-usd` — Maximum dollar amount to spend on API calls before stopping (print mode only)."
>
> "`--max-turns` — Limit the number of agentic turns (print mode only). Exits with an error when the limit is reached. No limit by default."
>
> "`--dangerously-skip-permissions` — Skip permission prompts. Equivalent to `--permission-mode bypassPermissions`."

Multi-agent support (Claude Code overview — `https://code.claude.com/docs/en/overview`):

> "Spawn multiple Claude Code agents that work on different parts of a task simultaneously. A lead agent coordinates the work, assigns subtasks, and merges results."

> "For fully custom workflows, the Agent SDK lets you build your own agents powered by Claude Code's tools and capabilities, with full control over orchestration, tool access, and permissions."

### 22.5. Gray-Zone Analysis Carryover (From TOS_COMPLIANCE §4)

The original `TOS_COMPLIANCE.md` §4 identified six gray zones with the pre-convergence cascade plan. The convergence summary on 2026-04-15 (retracted the legal framing in favor of the operational one — see `TOS_DISCUSSIONS.md` cross-cutting A) resolved most of these. The remaining operational / safety-posture concerns are carried forward here as engineering constraints, not legal risks:

- **Subscription-tier rate limits (§4.1 → §22.2 + §12.1).** Pure-headless under Max weekly Opus quotas + 5-hour session windows is the operational ceiling, not the legal ceiling. N=6 concurrency cap (§12.1) is calibrated against this.
- **Auto-mode classifier omitted (§4.2 → §10.6 + cascade safety layers).** Auto mode is unavailable on Max tier. The cascade's own safety layers (per-path `--allowedTools`, file + package locks, `max_tries`, `max_budget_usd`, deterministic CI gates, asymmetric QA with enforced certificate structure) are the intentional substitute. Documented explicitly in §10.6 Full-Benefit Rule.
- **`Bash(mage *)` wide door (§4.3 → §19.10 refinement).** Replace `Bash(mage *)` with per-kind explicit allowlists (`Bash(mage test-func *), Bash(mage test-pkg *), Bash(mage ci)` for `build`; `Bash(mage test-golden *), Bash(mage ci)` for `build-qa-proof`/`build-qa-falsification`/`plan-qa-proof`/`plan-qa-falsification`; no Bash for `plan` and `commit`). Scheduled for drop 10 refinement; not drop-1 blocker but must land before broader external use.
- **Host-machine posture vs sandbox (§4.4 → future drop).** Dogfood runs in the shared `main/` checkout, gated only by file + package locks. Sandboxing options (per-run git worktree, `sandbox-exec` on macOS, devcontainer, Firecracker VM) are explicit future drops, not drop-1 scope. Accepted dogfood risk.
- **Cascade-run budget ceiling + escalation-depth cap (§4.5 → §14 + §19.10).** Per-invocation `max_budget_usd` + `max_turns` exist; the summed cascade-run ceiling and escalation-depth cap are drop 10 refinement items.
- **Training-opt-out posture (§4.6 → §22.1).** Explicitly ON at the dev's Anthropic account. Documented in §22.1 as a dogfood precondition.

---

## 23. Mention Routing

**Status:** Converged 2026-04-15/16. Lineage: this section is NOT a fresh-invented system; it is the formalization of the mention-routing model the orchestrator has been using in Tillsyn since before this plan existed. The existing `@human` / `@dev` / `@builder` / `@qa` / `@orchestrator` / `@research` addressing vocabulary is established (see bare-root + `main/CLAUDE.md` Discussion Mode, coordination-surfaces rows, and the user-global Tillsyn-First Coordination rule). This section consolidates the routing rules, adds cascade-agent system-prompt semantics (via `claude -p --append-system-prompt`), and inserts the TUI/dispatcher hooks.

### 23.1. Lineage From Existing Design

Source lineage, in order:

- **Bare-root + `main/CLAUDE.md`** — documents the active mention vocabulary, the dev=builder alias, and the "Open handoffs are the primary Action Required rows for the addressed viewer" rule.
- **`main/HEADLESS_DISCUSSIONS.md` §3.1** (folded into this plan per consolidation ledger) — Tillsyn-defined agents via `claude -p --append-system-prompt`, the mention-routing model, and inter-orchestrator comms.
- **Existing `till.comment` + `till.handoff` + `till.attention_item` MCP ops** — the routing substrate already exists in Tillsyn; §23 formalizes the UX layer (drop 4.5 TUI surface) and the cascade-agent response path.

### 23.2. The Mention Vocabulary

Routed `@`-mentions, in order of specificity:

| Mention | Routes To | Notes |
|---|---|---|
| `@human` | Dev (primary human operator) | Breakglass channel — reserved for decisions the orchestrator can't make. |
| `@dev` | Dev — aliases to `builder` for role-dispatch purposes | Vocabulary kept for human-facing comment threads. |
| `@builder` | Active builder agent on the target drop, if any | If no builder is active, the comment is held until a builder spawns. |
| `@qa` | Both qa-proof and qa-falsification on the target drop | Broad address — use for questions both passes should see. |
| `@qa-proof` / `@qa-falsification` | Specific QA role | Narrow address — use when only one pass should respond. |
| `@orchestrator` | Active orchestrator session on the target project | Cross-orchestrator comms when multiple orchestrators run. |
| `@research` | Built-in Explore subagent spawned on demand | Research requests are comment-triggered. |

### 23.3. Routing Mechanics

Two coordination surfaces carry mentions with different semantics:

- **`till.comment`** — shared append-only thread lane on the action item. Mentions surface in the addressed viewer's Comments notifications section. Lightweight discussion, question-and-answer, audit-trail.
- **`till.handoff`** — structured next-action routing. Mentions surface in the addressed viewer's **Action Required** list. Used when a concrete next action is expected.

Rule of thumb: questions and audit-trail → comments; routed next-action → handoffs. `till.attention_item` is the orchestrator's inbox substrate and is not addressed via `@`-mentions in user text.

### 23.4. Cascade-Agent System-Prompt Integration

Dispatched cascade agents need a system-prompt addendum that teaches them how to:

1. **Detect** `@`-mentions addressed to their role in comments on the action item they own.
2. **Respond** via a new comment on the same action item, echoing the addresser so the thread reads naturally.
3. **Escalate** to `till.handoff` or `till.attention_item` when the question requires a decision the agent cannot make.

The addendum is passed via `claude -p --append-system-prompt <text>` (Anthropic Claude Code flag — see §20.5 resolution for pass-through mechanics). The text is hardcoded per-role in the agent's `.md` file, not per-dispatch.

**Planning subdrop required before any of §23 ships in code.** Per §2.2 START bracketing rule, the system-prompt text is load-bearing — it defines how cascade agents interpret and respond to human direction during their short lifetime. Dev confirmation on exact wording is required before the drop 4.5 builder fires.

### 23.5. Where This Lands

- **TUI mention composer**: drop 4.5 (§19.4.5 / §24).
- **Cascade-agent system-prompt addenda**: drop 4.5 (same planning subdrop).
- **Routing backend** (the MCP-level logic that identifies `@`-mentions and writes notifications): existing substrate, extended incrementally as drop 4.5's UX reveals needs.

---

## 24. File Viewer (TUI) with `charmbracelet/glamour`

**Status:** Converged 2026-04-15/16. Scope expanded from the original git-diff-only concept (`main/HEADLESS_DISCUSSIONS.md` §3.2) to a full file-viewing surface built on the already-vendored `charmbracelet/glamour` dependency. Lands in drop 4.5 (§19.4.5).

### 24.1. Scope

Four TUI surfaces, all reading the same drop-node data (`paths` / `packages` / `files` / `start_commit` / `end_commit` from drop 1):

1. **Attached-file viewer** — reads `files []string`, renders each attached file via `glamour` (markdown-rendered if `.md`, syntax-highlighted fenced code block wrapped in a styled pane if `.go` / `.sql` / `.toml` / etc.).
2. **Path-picker** — TUI file-tree for populating `paths` on drop creation. Multi-select, glob-filter, package-derivation (selected paths auto-populate `packages` via `internal/domain` resolver).
3. **File-picker** — TUI file-tree for populating `files` (reference attachments). Looser validation than path-picker — any repo-tracked file is attachable.
4. **Git-diff pane** — for any drop with `start_commit` set, renders `git diff <start_commit>..HEAD` (or `..end_commit` for complete drops) scoped to the drop's `paths`. Live-updates on file-watcher events during `in_progress`.

### 24.2. Why Glamour

- Already vendored (`charmbracelet/glamour` is a Bubble Tea-native stack dependency — no new third-party).
- Theme-aware (lip gloss integration already wired for the rest of the TUI).
- Handles markdown + chroma-backed syntax highlighting + diff lexer in one library. No per-file-type rendering logic to maintain.

### 24.3. Keybinding Sketch

- `f` — open attached-file viewer (on selected drop).
- `F` — open file-picker to attach a file to the selected drop.
- `p` — open path-picker to edit-scope paths on the selected drop.
- `d` — open git-diff pane for the selected drop.
- `g <dotted-address>` — fast-nav via drop-2 dotted-address resolver.
- `m` — open mention composer (§23).

Keybindings subject to clash-audit against existing TUI bindings during drop 4.5's planning subdrop.

### 24.4. Dependencies

- Drop 1: `paths`, `packages`, `files`, `start_commit`, `end_commit` all first-class.
- Drop 2: dotted-address resolver (for `g` keybinding).
- Drop 4: dispatcher core (for live state-change events the viewer subscribes to).
- Drop 4.5: this section (scheduling container).

### 24.5. Out of Scope

- Editing from within the viewer. The viewer is read-only; edits go through the builder agent path, not the TUI. (Rationale: orchestrator-never-edits-Go-code rule in `CLAUDE.md` applies to the TUI as well when the TUI is run from the orchestrator session.)
- Remote / cross-project file rendering. The viewer operates on the active project's primary worktree only.
- In-TUI git operations (stage / commit / push). The viewer shows diffs; mutations go through the existing git management path.
