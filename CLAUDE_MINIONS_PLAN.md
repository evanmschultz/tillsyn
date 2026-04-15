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

---

## 1. Naming and Terminology

### 1.1. Finalized: "Agent"

The dispatched one-shot unit is called an **agent**. The term is already used throughout this codebase and the rollout (`go-builder-agent`, `go-qa-proof-agent`, subagent types), so introducing a new umbrella word would create more confusion than it solves. Agents are typed by role (planner, builder, QA-proof, QA-falsification, commit). The umbrella "agent" covers all of them. No new name.

- "The cascade dispatches a planning agent" — reads naturally
- "Build agent failed, QA agent passed" — reads naturally
- "The agent's auth was revoked" — reads naturally

### 1.2. The Cascade (Confirmed)

The execution pattern is a **cascade**: **design down, build up.** Planning decomposes downward through levels; completion propagates upward. State changes trigger the next step.

- **Cascade run** — a single top-to-bottom-to-top execution, starting from a slice moving to `in_progress`
- **Cascade tree** — the tree of plan items produced by a cascade run

### 1.3. Glossary

| Term | Definition |
|---|---|
| **Agent** | A one-shot autonomous unit dispatched by Tillsyn. Receives auth + task context, does work via MCP, moves its task to `complete`/`failed`, dies. Typed by role: planner, builder, QA, commit. |
| **Cascade** | The hierarchical execution pattern: planning decomposes downward ("design down"), completion propagates upward ("build up"). State changes trigger the next step. |
| **Cascade run** | A single execution of a cascade, from slice `in_progress` through completion or failure. |
| **Cascade tree** | The tree of plan items produced by a cascade run. |
| **Dispatcher** | The Tillsyn subsystem that watches for state changes and spawns agent processes. Not an agent — purely programmatic (except commit message formation, which uses a lightweight haiku agent). |
| **Gate** | A deterministic verification step (e.g., `mage ci`) that runs programmatically after an agent completes. No LLM involved. |
| **Slice** *(rename from phase)* | A nestable grouping of work. Contains a plan task, plan QA, and either sub-slices or build tasks. |
| **Task** | A unit of work. Contains QA checks as children. |
| **Check** | A QA verification step (proof or falsification). Child of a task. |

### 1.4. Cascade Addressing Vocabulary (Slice 0 Convergence)

Converged during Slice 0 closeout discussions. Pre-cascade these are conceptual; post-Slice-4 the dispatcher materializes them.

**Slice nesting + naming:**

- **Slices all the way down.** The `project` is NOT a slice — it is the root container. Immediate children of the project are **top-level slices** (slice_0, slice_1, …).
- A slice's children that are themselves slices are called **sub-slices**, addressed as `slice_sub_N` where `N` is the zero-indexed position among this slice's sub-slice children. A slice may have multiple non-slice children (plan-tasks, build-tasks, closeout); only other slice-kind children count toward `N`.
- Nesting is infinite. `slice_sub_1` of `slice_sub_0` is perfectly valid — the planner decomposes as far as the work demands.

**Dotted addresses (read-only shorthand):**

- Top-level slice `N` is addressed as `N` (e.g. `0` is slice 0).
- Sub-slice `M` of slice `N` is `N.M` (e.g. `0.1` is slice 0's second sub-slice).
- `0.1.5.2` = project's slice_0 → its sub-slice_1 → that sub-slice's sub-slice_5 → that sub-slice's sub-slice_2.
- Project-qualified form: `<proj_name>-<dotted>` (e.g. `tillsyn-0.1.5.2`) for unambiguous cross-project references.
- **Dotted addresses are read-only.** For mutations, always use the UUID plan-item id. Dotted addresses are unstable under re-parenting and should never be load-bearing in scripts.

**Type-slice kinds (post-Slice-3 template vocabulary):**

A slice has a **primary type** recorded in its `kind` (or equivalent metadata pre-template). Typed slices let the dispatcher route agents and the planner apply different `child_rules`. Canonical set:

| Kind | Purpose | Typical children |
|---|---|---|
| `plan-slice` | Planning-dominant work — decomposing and specifying future work. | plan-task + plan-qa-proof + plan-qa-falsification. |
| `build-slice` | Build-dominant work — tasks that change code. | build-task(s) + qa children; optional sub-slices if decomposition continues. |
| `qa-slice` | QA-dominant work — review, audit, verification without code change. | qa-check children (proof / falsification). |
| `closeout-slice` | Slice-end ledger update, Hylla reingest, WIKI_CHANGELOG line. | The `SLICE <N> END — LEDGER UPDATE` task. |
| `refinement-slice` | Perpetual / long-lived tracking slice for accumulated refinement entries (e.g. `REFINEMENTS.MD`, `HYLLA_REFINEMENTS.MD`). Typically adhoc-created, not template-generated. | Whatever the refinement calls for (usually notes / discussion, occasionally build tasks). |
| `human-verify-slice` | Dev must inspect and ack before the slice can close. | Attention item(s) + checklist task(s). |
| `discussion-slice` | Cross-cutting decision park — description holds the converged shape, comments hold the audit trail. Pre-cascade, discussion happens in chat (see `CLAUDE.md` §"Discussion Mode"); this kind formalizes the pattern. | Notes / decision items. |

Pre-Slice-3, the project has **no template bound** and these types exist only as labels + naming conventions. The cascade-tree architecture in `CLAUDE.md` still applies; type-slice kinds extend it with orthogonal routing hints. Slice 3 encodes them as template kinds; Slice 4's dispatcher reads them.

**Adhoc vs. template-generated slices:**

- **Template-generated** — a `build-slice`'s `plan-task` child-rule creates sibling `plan-qa-proof` and `plan-qa-falsification` automatically. This is the cascade's default flow.
- **Adhoc** — a refinement or discussion slice created manually by the orchestrator outside any cascade flow, typically because the work is cross-cutting or long-lived. Per Slice 0 5.2 decision: refinement slices (`REFINEMENTS.MD`, `HYLLA_REFINEMENTS.MD`) use the generic `slice` kind + adhoc creation pre-Slice-3, and existing slices get updated in place rather than re-created.

---

## 2. Hierarchy Refactor

### 2.1. Current State — Confused Primitives

Today Tillsyn has: `project > branch > phase > task > subtask`. In practice:
- **Branch** exists to map to git worktrees. With file-level gating instead of worktrees, branches are unnecessary.
- **Phase** and **branch** are used interchangeably and inconsistently.
- **`depends_on`** overlaps with the parent-child hierarchy (children must complete before parent = implicit depends_on).
- **`done`** should be `complete` — more descriptive, clearer intent.

### 2.2. Proposed Hierarchy

```
project
  └── slice (nestable — was "phase" and "branch")
        ├── plan-task (always present in a slice)
        ├── plan-qa-proof (always present)
        ├── plan-qa-falsification (always present)
        ├── task (build task — leaf-level work)
        │     ├── qa-proof (always present under task)
        │     └── qa-falsification (always present under task)
        └── slice (sub-slice — infinite nesting)
              └── ... same structure ...
```

**Rules:**
- **Slices** always have: plan-task + plan-qa-proof + plan-qa-falsification. They contain either tasks or sub-slices (or both).
- **Tasks** always have: qa-proof + qa-falsification as children. Tasks are leaf-level — no nesting beyond one level of checks.
- **Slices can nest infinitely.** A planner at one level creates sub-slices if the work needs further decomposition, or tasks if the work is granular enough.
- A planner can create a task directly (small enough, no further decomposition needed) — it just creates the task with its QA children.

### 2.3. Remove `branch`

Branch was a primitive for git worktree mapping. With file-level gating (Section 5), worktrees are unnecessary. Branches add a hierarchy level that creates confusion with phases.

**Action:** Remove `branch` from the hierarchy. Migrate existing branches to slices. Remove from schema, domain, TUI, templates.

### 2.4. Rename `phase` → `slice`

"Phase" implies temporal ordering (phase 1, phase 2). "Slice" implies a bounded chunk of work that can be ordered OR parallel. Better fit for the cascade model where sibling slices might run in parallel.

**Action:** Rename in schema, domain, TUI, templates, documentation. Migration: existing phases become slices.

### 2.5. Rename `done` → `complete`

`complete` is more descriptive. "This task is complete" reads better than "this task is done." Aligns with `completion_contract`, `completion_notes`, `CompletedAt`.

**Lifecycle states:** `todo` → `in_progress` → `complete` | `failed`

**Action:** Rename in schema (DB column values), domain constants (`StateComplete`), TUI labels, MCP adapter normalization, templates, documentation. This touches the same surfaces as the `failed` state addition — combine the migration.

### 2.6. Simplify `depends_on` and `blocked_by`

| Mechanism | Keep? | Rationale |
|---|---|---|
| **Parent-child** | **Yes** | Core hierarchy. `require_children_done` enforces completion ordering. |
| **`blocked_by`** | **Yes** | For sibling ordering (SLICE-2 blocked_by SLICE-1) and cross-branch blocking (file-level conflicts). Essential for cascade scheduling. |
| **`depends_on`** | **Remove (last slice)** | Redundant with parent-child + `blocked_by`. Planned ordering is `blocked_by` set at creation time. Remove in final dogfooding slice as a real test of the cascade system. |

**Action:** Keep `depends_on` functional during build. Remove in the final cleanup slice during dogfooding. This serves as a good integration test of the cascade.

### 2.7. Incremental Migration Strategy

This refactor touches schema, domain, app, adapters, TUI, templates, and documentation. It must be done incrementally:

1. Each rename/removal is a small, testable increment
2. After each increment: `mage ci`, Hylla reingest, confirm no orphaned code, confirm nothing that worked before is broken
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
SLICE: "Add failed lifecycle state" ← human/orchestrator moves to in_progress
│
│  ┌──────────────────────────────────────────────────────────┐
│  │ TEMPLATE AUTO-CREATES:                                   │
│  │  • PLAN-TASK (kind: plan-task)                           │
│  │  • PLAN-QA-PROOF (kind: qa-check, role: qa)              │
│  │  • PLAN-QA-FALSIFICATION (kind: qa-check, role: qa)      │
│  └──────────────────────────────────────────────────────────┘
│
├── PLAN-TASK ← no blockers → auto in_progress → planner agent fires
│   │  Agent: go-planning-agent (opus, high effort)
│   │  Work: reads slice goal, decomposes into 2-4 sub-slices,
│   │        fills out scope/context/affected areas for each,
│   │        sets blocked_by between sequential slices,
│   │        sets file paths on each task for file-level gating
│   │  Output: creates SUB-SLICE-1, SUB-SLICE-2 as children of SLICE
│   │          via till.plan_item(operation=create)
│   │  Terminal: moves own task to complete via MCP → auth revoked → killed
│   │
│   ├── PLAN-QA-PROOF ← blocked_by: PLAN-TASK → fires when plan completes
│   │   Agent: go-qa-proof-agent (opus, high effort)
│   │   Checks: plan completeness, evidence grounding, consistency
│   │   PASS → moves to complete │ FAIL → moves to failed + comment
│   │
│   └── PLAN-QA-FALSIFICATION ← blocked_by: PLAN-TASK → fires in parallel
│       Agent: go-qa-falsification-agent (opus, high effort)
│       Checks: vagueness, missing cases, incorrect assumptions, YAGNI,
│               FILE-LEVEL CONFLICTS (must verify no two tasks share files
│               without explicit blocked_by between them!)
│       PASS → moves to complete │ FAIL → moves to failed + comment
│
│   ┌──────────────────────────────────────────────────────────────┐
│   │ ALL PLAN QA PASS → sub-slices become eligible                │
│   │                                                              │
│   │ ANY PLAN QA FAIL → failed + comment with findings            │
│   │   try 1 of max-tries=2:                                     │
│   │     new PLAN-TASK created with failure context                │
│   │     planner re-plans incorporating QA feedback               │
│   │     plan QA runs again on the revised plan                   │
│   │   try 2 fails → attention item to orchestrator AND human     │
│   │     full stop until human intervenes                         │
│   └──────────────────────────────────────────────────────────────┘
│
├── SUB-SLICE-1 ← no blocked_by → auto in_progress when plan QA passes
│   │
│   │  ┌─ TEMPLATE AUTO-CREATES plan-task + plan-qa children ─┐
│   │
│   ├── PLAN-TASK ← planner agent fires
│   │   │  Agent: go-planning-agent (opus, high effort)
│   │   │  Work: decomposes sub-slice into 1-4 granular build tasks
│   │   │        fills out: description, paths, packages,
│   │   │        acceptance_criteria, test targets
│   │   │        sets blocked_by for tasks sharing files OR packages
│   │   │  Output: creates BUILD-TASK-1, BUILD-TASK-2 as children
│   │   │
│   │   ├── PLAN-QA-PROOF (checks plan + file conflict detection)
│   │   └── PLAN-QA-FALSIFICATION (attacks plan + file conflict detection)
│   │
│   │   ON PLAN QA PASS → build tasks eligible:
│   │
│   ├── BUILD-TASK-1 ← no blocked_by → auto in_progress
│   │   │  Agent: go-builder-agent (sonnet, standard effort)
│   │   │  File gating: can only edit files listed in plan item paths
│   │   │  Pre-check: system confirms assigned files have clean git status
│   │   │  Work: implements code, runs mage test-func on affected funcs
│   │   │  max-tries=2 (builder can retry once on test failure)
│   │   │  Terminal: moves to complete via MCP → auth revoked → killed
│   │   │
│   │   │  ┌─ ON BUILDER COMPLETE: ───────────────────────────────┐
│   │   │  │ GATE: mage ci (deterministic, system runs it)         │
│   │   │  │ Gate pass → commit agent (haiku) forms message     │
│   │   │  │   → system commits + optionally pushes (template cfg) │
│   │   │  │   → hylla reingest (system, deterministic)            │
│   │   │  │   → task stays complete → QA fires                   │
│   │   │  │ Gate fail → task moves to failed + gate output comment │
│   │   │  │   → try 1 of max-tries: new builder fires            │
│   │   │  │   → try 2 fails: escalate (re-plan or human)         │
│   │   │  └──────────────────────────────────────────────────────┘
│   │   │
│   │   ├── QA-PROOF ← fires after build + gates + commit complete
│   │   │   Agent: go-qa-proof-agent (sonnet, medium effort)
│   │   │   Checks: evidence completeness, reasoning, trace coverage
│   │   │
│   │   └── QA-FALSIFICATION ← fires in parallel with QA-PROOF
│   │       Agent: go-qa-falsification-agent (sonnet, medium effort)
│   │       Checks: counterexamples, hidden deps, contract mismatches
│   │
│   │   ALL QA PASS → BUILD-TASK-1 complete
│   │   ANY QA FAIL → BUILD-TASK-1 failed → attention item
│   │     QA never retries. QA failure → escalation or human.
│   │
│   ├── BUILD-TASK-2 ← blocked_by: BUILD-TASK-1 (if shares files)
│   │   │              OR no blocked_by (if different files → parallel)
│   │   └── ... same build+gate+commit+QA flow ...
│   │
│   │  ┌─ SLICE COMPLETION CHECK: ─────────────────────────────┐
│   │  │ System checks for uncommitted/unpushed changes         │
│   │  │ If any found → attention item to orchestrator          │
│   │  └───────────────────────────────────────────────────────┘
│   │
│   └── ALL BUILD-TASKS complete → SUB-SLICE-1 complete
│
├── SUB-SLICE-2 ← blocked_by: SUB-SLICE-1 (or parallel if no conflict)
│   │  Auto in_progress when SUB-SLICE-1 completes
│   │  Same cascade flow
│   └── ...
│
└── ALL SUB-SLICES complete → SLICE complete
```

### 3.3. Key Properties

**Design down (decomposition):**
- Slice → Sub-slices → Build tasks
- At each level, a planner agent does the decomposition
- Planning QA verifies the decomposition before execution proceeds
- Planner must explicitly set file paths and `blocked_by` for file-level conflicts
- Plan QA falsification specifically checks for missing file-level blockers
- The planner creates child plan items via `till.plan_item(operation=create)`
- Template `child_rules` auto-create QA children for each created item

**Build up (completion):**
- Build task complete → sub-slice checks all children → sub-slice complete → slice checks → complete
- Uses `require_children_done` — parent can't complete until all children complete
- No `depends_on` needed for parent-child — the hierarchy IS the dependency
- `blocked_by` is for siblings and file-level conflicts

**Parallel execution (natural concurrency):**
- When a parent moves to `in_progress`, ALL children without blockers auto-fire simultaneously
- BUILD-TASK-1 and BUILD-TASK-2 fire in parallel if they don't share files
- QA-PROOF and QA-FALSIFICATION always fire in parallel
- No parallelism config — just absence of `blocked_by`. If it's not blocked, it fires.

### 3.4. `blocked_by` — The Only Sibling Ordering Primitive

With the hierarchy refactor (Section 2.6), `depends_on` is marked for removal in the final dogfooding slice. Until then, both exist. After removal:

| Mechanism | What It Means | Where It Applies |
|---|---|---|
| **Parent-child hierarchy** | Child must complete before parent can complete | Built into cascade. Uses `require_children_done`. |
| **`blocked_by`** | Sibling or cross-slice item must be `complete` before this item can move to `in_progress` | Sequential slices, file-level conflicts between tasks, cross-slice dependencies. |

The planner sets `blocked_by` at creation time for planned ordering. Runtime discoveries (unexpected file conflicts detected by the dispatcher) add `blocked_by` dynamically.

---

## 4. Dispatch Mechanism

### 4.1. How It Works

The dispatcher is a **programmatic subsystem inside Tillsyn**, not a separate process or CLI command. It watches for lifecycle state transitions and spawns agent processes.

```
State change detected: plan_item moved to in_progress
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
  ├── Monitor: subscribe to LiveWaitBroker for this plan item's state changes
  │
  └── On state change to complete|failed:
      ├── Auth auto-revoked (state-machine hook in MoveTask)
      ├── Release file locks held by this agent
      ├── Kill the claude CLI process
      ├── Cascade: check for newly-eligible children/siblings
      └── Fire next eligible items
```

### 4.2. No `till run` Command

There is no `till run` CLI command. The dispatch is triggered by state changes. The state change can come from:

- **TUI** — human moves a plan item to `in_progress`
- **MCP** — orchestrator calls `till.plan_item(operation=move_state, state=in_progress)`
- **Dispatcher itself** — when a blocker clears, the dispatcher auto-moves eligible items to `in_progress`

The dispatcher is always running as part of the Tillsyn process (serve, serve-mcp, or TUI). It subscribes to `LiveWaitBroker` state-change events.

### 4.3. Auto-Promotion of Eligible Items

When any plan item moves to `complete`:
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
  ├── Agent reads task details via plan_item(get) — its working brief
  │
  ├── Agent does work:
  │   ├── Planner: creates child plan items via MCP
  │   ├── Builder: edits files (gated to allowed paths), runs mage test-func
  │   └── QA: reads code, verifies, writes certificate
  │
  ├── Agent calls till.plan_item(operation=move_state, state=complete|failed)
  │   ├── Includes metadata.outcome, completion notes, comments
  │   └── This is the terminal MCP call
  │
  ├── MoveTask fires:
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

The planner sets **both** `paths` (the specific files) and `packages` (the Go packages those files belong to) on each build task. The dispatcher enforces mutual exclusion:

1. Before dispatching a builder, check all assigned files against active file locks AND all assigned packages against active package locks.
2. If any file or package is locked by another active agent → add dynamic `blocked_by`, defer dispatch.
3. If all files and packages are free → acquire both sets of locks, dispatch.

### 5.3. Planner Responsibility

The planner **must** set file paths and package paths on every build task it creates. This is not optional — it's how the cascade prevents file and package conflicts.

Plan QA falsification **specifically checks** for:
- Tasks with missing or incomplete `paths`
- Tasks with missing or inconsistent `packages` (packages must cover every file in `paths`)
- Two sibling tasks sharing a file OR a package without an explicit `blocked_by` between them
- File paths that don't match the slice's stated scope

If the planner misses a path or a package, the dispatcher catches it at dispatch time via git status checks (Section 5.4) and the package-lock check, but relying on runtime detection is the fallback, not the plan.

### 5.4. Git Status Pre-Check

Before a builder agent starts, the dispatcher confirms:

```
git status --porcelain -- <file1> <file2> ...
```

If any assigned files have uncommitted changes (dirty git status):
- **Block dispatch** — do not start the builder
- Post a comment on the plan item listing the dirty files
- Fire an attention item to the orchestrator
- The orchestrator or human must resolve the dirty state before the builder can proceed

This catches two problems:
1. Files dirtied by a previous agent that crashed before commit
2. Files manually edited by the human outside the cascade

### 5.5. Dispatcher Auto-Detection of Conflicts

Even if the planner forgot to set `blocked_by` between two tasks sharing files or packages, the dispatcher detects it at dispatch time:

1. Builder A dispatches with files `[internal/domain/a.go]`, packages `[internal/domain]` → both locks acquired.
2. Builder B tries to dispatch with files `[internal/domain/b.go]`, packages `[internal/domain]`. File `b.go` is free, but the package `internal/domain` is already locked by Builder A.
3. Dispatcher adds dynamic `blocked_by: Builder-A's-task` to Builder B's task.
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
| **Agent binding** | Which agent type fires for this kind | `build-task` → `go-builder-agent` |
| **Model** | Which Claude model to use | `opus`, `sonnet`, `haiku` |
| **Effort** | Claude Code effort level | `high`, `standard`, `low` |
| **Tools** | Allowed and disallowed tools | `allowedTools: [Read, Edit, Bash, Grep]` |
| **Budget** | Max cost per invocation | `max_budget_usd: 5.00` |
| **Turns** | Max conversation turns | `max_turns: 20` |
| **Max-tries** | Total attempts before permanent failure | `max_tries: 2` |
| **Gates** | Deterministic verification steps | `[{command: "mage ci", on_fail: "fail_task"}]` |
| **Child rules** | Auto-created children on item creation | `build-task` → auto-create `qa-proof`, `qa-falsification` |
| **Trigger state** | Which state transition fires the agent | `in_progress` (default for all) |
| **Escalation** | Whether failures re-trigger planning | `escalation_enabled: true` |
| **Push policy** | Whether to push after commit | `auto_push: true` (default), `auto_push: false` |
| **Commit message** | Always formed by the haiku commit agent. No deterministic fallback. Style rules hardcoded in the agent's prompt details. | `commit_agent: true` |

### 6.2. Example Template Kind Definition (Sketch)

```toml
[kinds.build-task]
agent_name = "go-builder-agent"
model = "sonnet"
effort = "standard"
max_budget_usd = 5.00
max_turns = 20
max_tries = 2
trigger_state = "in_progress"
auto_push = true
commit_agent = true

[kinds.build-task.tools]
allowed = ["Read", "Edit", "Write", "Bash", "Grep", "Glob"]
disallowed = ["Agent"]

[[kinds.build-task.gates]]
name = "ci"
command = "mage ci"
on_fail = "fail_task"

[kinds.build-task.child_rules]
auto_create = ["qa-proof", "qa-falsification"]

[kinds.plan-task]
agent_name = "go-planning-agent"
model = "opus"
effort = "high"
max_budget_usd = 10.00
max_turns = 30
max_tries = 2
trigger_state = "in_progress"
escalation_enabled = true

[kinds.plan-task.tools]
allowed = ["Read", "Grep", "Glob", "Bash"]
disallowed = ["Edit", "Write", "Agent"]

[kinds.commit-agent]
agent_name = "commit-message-agent"
model = "haiku"
effort = "low"
max_budget_usd = 0.10
max_turns = 3
```

### 6.3. Per-Project Template Overrides

The template defines defaults. Projects can override per-kind settings (e.g., a project with expensive CI might increase `max_budget_usd`). Override mechanism TBD — likely a project-level config section.

---

## 7. Agent Types and Model Assignment

### 7.1. Agent Inventory

The cascade uses these agent types:

| Agent Type | Agent File | Role | Edits Code? |
|---|---|---|---|
| **Planner** | `go-planning-agent.md` | Decomposes work into plan items | No |
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
| **Commit Agent** | haiku | low | Reads `git diff`, plan item title, and template commit format rules. Forms a conventional-commit message. Extremely narrow scope. |
| **Wiki** | haiku | low | Summarization task. Absorb child wikis, produce summary. |
| **Quality/Vuln** | sonnet | high | Deep graph analysis. Needs careful reasoning about resource lifecycles. |

**Open question:** Is sonnet sufficient for builders? The plan provides detailed instructions, file paths, acceptance criteria. The builder's job is execution, not design. But complex code changes may need opus. **Suggestion:** Default to sonnet, template-configurable to opus for complex kinds.

---

## 8. Auth and Lifecycle

### 8.1. Auth Flow for Agents

Agent auth is **pre-approved by the system**. No human approval step because:

1. The human already approved the cascade by moving the parent to `in_progress`
2. The template defines which agents fire — the human approved the template
3. Each agent gets auth scoped to its specific plan item (or parent slice for planners)

```
Dispatcher detects: plan_item moved to in_progress
  ├── Creates auth session: role=<from_template>, scope=<from_template>, item=<plan_item_id>
  │   TTL: from template (default 30 min)
  │   No human approval — system-issued
  ├── Passes auth credentials in agent prompt
  └── Agent claims auth at boot
```

### 8.2. Auth Revocation

Auth is revoked when:
- The agent moves its task to `complete` or `failed` (auto-revocation)
- The TTL expires (agent took too long)
- The dispatcher kills the agent (e.g., budget exceeded)

### 8.3. Agent Terminal Action

When an agent calls `till.plan_item(operation=move_state, state=complete|failed)`:

1. MoveTask validates the transition
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
Build agent moves task to complete
  │
  ├── Dispatcher receives state-change event
  ├── Checks template: does this kind have gates?
  │   YES ↓
  │
  ├── GATE 1: mage ci
  │   ├── Pass → continue
  │   └── Fail → task moves to failed (override auth) + gate output as comment
  │
  ├── GATE 2: commit (deterministic + commit agent)
  │   ├── git add <affected files>
  │   ├── Spawn commit agent (haiku):
  │   │     reads git diff, plan item title, commit style rules embedded
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
  │   └── Fail → task moves to failed + commit output as comment
  │
  ├── GATE 3: hylla reingest (deterministic)
  │   ├── hylla inspect --source-dir <project_dir> --enrichment-mode full
  │   └── Fail → attention item (non-blocking for QA, but flagged)
  │
  └── All gates pass → task stays complete → QA agents fire
```

### 9.3. The Commit Agent

The commit agent is a special lightweight agent:

- **Model:** haiku (cheapest, fastest)
- **Scope:** Reads `git diff --cached`, the plan item's title and description, and outputs a single commit message string.
- **Commit style rules are hardcoded in the agent file's prompt details**, not templated per-project. The agent knows the conventional-commit format, length caps, and repo conventions because its system prompt says so.
- **No file edits.** No MCP mutations. No state changes. Pure text generation.
- **System validates structure and length only** — non-empty, within a hard length cap. There is **no regex style validator** and **no deterministic fallback**. If the message fails structure/length, the system re-spawns the commit agent with the rejection reason.
- **Max 2 tries.** On second failure, the system escalates to the orchestrator: posts an attention item with the `git diff --cached` command and the `git commit -m "<message>"` command. The orchestrator forms a message, runs the commit command, the dispatcher re-validates, and commits if green.
- **System makes the actual `git commit` and `git push` calls.** The commit agent only forms the message — it never touches git directly.

This keeps the commit flow 95% deterministic (system runs git) with a thin LLM layer for message quality.

### 9.4. Push Configurability

Whether to auto-push after commit is **template-configurable**:

```toml
[kinds.build-task]
auto_push = true   # push after every successful commit (default)
# auto_push = false  # commit only, push at slice completion
```

When `auto_push = false`, the dispatcher commits locally but defers push. The **slice completion check** (Section 9.6) catches unpushed commits.

### 9.5. Gate Output

Gate stdout/stderr is captured and posted as a `till.comment` on the plan item. On failure:
- Task moves to `failed` (dispatcher uses override auth for `complete → failed` transition)
- `metadata.outcome: "failure"`
- `metadata.gate_name: "<which gate failed>"`
- Gate output posted as comment
- Retry logic fires (Section 14)

### 9.6. Slice Completion Check

When all children of a slice are `complete`, before the slice itself moves to `complete`, the dispatcher runs a completion check:

```
All children complete → slice completion check:
  ├── git status: any uncommitted changes in slice scope?
  │   YES → attention item to orchestrator
  ├── git log @{push}..: any unpushed commits?
  │   YES → if auto_push=true: push now. If auto_push=false: attention item.
  └── All clean → slice moves to complete
```

This catches edge cases where a builder's commit was deferred or a gate partially completed.

### 9.7. Hylla Reingest as a Gate

Hylla reingest is a deterministic step, not an agent task. It fires automatically after successful commit+push:

```
Build agent complete → mage ci passes → commit + push → hylla reingest
```

The template configures Hylla settings per project:
- Source directory
- Branch (main or worktree branch)
- Enrichment mode (always full, never structural_only)

**No agent runs hylla reingest.** The dispatcher calls it programmatically. QA agents need the fresh graph to verify against.

---

## 10. Trust Model

### 10.1. The Problem

Agents confabulate, skip cases, claim tests pass without running them. Unstructured agents are wrong 20-22% of the time on code verification tasks (semi-formal reasoning paper, arXiv 2603.01896).

In current Tillsyn usage: agents forget to move plan items, claim work is complete but don't update state, orchestrators skip QA, and completion claims are unverifiable.

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

Track this in the plan item's comments so the orchestrator can see the reasoning chain. Each hypothesis gets its own comment entry with the status update.

### 11.4. Pre-Build Preparation — Update Existing Agents and CLAUDE.md

**Before building the cascade**, update the existing agent files and CLAUDE.md to enforce semi-formal reasoning in the current manual workflow:

1. Update `~/.claude/agents/go-qa-proof-agent.md` and `go-qa-falsification-agent.md` with explicit certificate templates and mandatory enumeration
2. Update `~/.claude/agents/go-planning-agent.md` with planning certificate requirements
3. Update project CLAUDE.md files to explain semi-formal reasoning for the orchestrator's current manual workflow
4. Update CLAUDE.md to clearly document how the orchestrator should use agents as they stand today (without the cascade), so the workflow works during the build phase

This serves two purposes:
- Immediately improves current agent quality
- Validates the certificate structure before automating it

---

## 12. Concurrency Model

### 12.1. No Artificial Limits

There is no cap on concurrent agents. The cascade naturally parallelizes: any item without blockers fires when its parent moves to `in_progress`.

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
2. Task moves to `failed` with `metadata.outcome: "blocked"`, `metadata.blocked_reason: "timeout"`
3. File locks released
4. Attention item fires

---

## 13. Hylla Integration

### 13.1. Agent Access to Hylla

All agents get Hylla MCP access for code understanding. The dispatcher provides an `agent-mcp.json` config that includes:

- **Tillsyn MCP** — for task mutations (move state, create items, post comments)
- **Hylla MCP** — for code understanding (search, graph nav, node full)
- **Context7** — for library docs (optional, configurable)

gopls is excluded (too stateful, slow initialization, not needed for one-shot work).

### 13.2. Hylla Reingest

Reingest is a programmatic gate (Section 9.7), not an agent task. The dispatcher calls:

```bash
hylla inspect --source-dir <project_dir> --enrichment-mode full
```

Template configures:
- Source directory path
- Branch name
- Enrichment mode (always full)

Reingest fires after successful commit+push, before QA agents start. QA agents need the fresh graph to verify against.

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

When a build task fails after `max-tries` exhausted:

1. Build task is permanently `failed`
2. Attention item fires to the plan-level orchestrator
3. If `escalation_enabled: true` in template:
   a. A new plan task is created at the slice level with the failure context
   b. A planner agent fires to re-plan the slice, incorporating the failure
   c. Planning QA verifies the revised plan
   d. New build tasks are created from the revised plan
   e. `max-tries=2` for the escalation cycle itself
4. If escalation also fails (2 tries), attention item to human. Full stop.

### 14.5. Escalation Tracking

**Nothing happens in the dark.** Every escalation must produce:

1. A diff of how the plan changed from the original
2. Why the plan changed (failure context from the failed build task)
3. What the planner tried differently
4. A comment on the original failed task linking to the new plan

The planner agent is responsible for documenting this in its plan task's comments. QA verifies the documentation is present and accurate.

### 14.6. Template Configuration

```toml
[kinds.build-task]
max_tries = 2
blocked_retries = 2          # additional attempts for external failures
blocked_retry_cooldown = 60  # seconds between blocked retries
escalation_enabled = true    # on permanent failure, re-plan at parent level

[kinds.plan-task]
max_tries = 2
escalation_enabled = false   # plan failure goes to human, not further re-planning
```

Escalation can be disabled per-kind in the template. When disabled, permanent failure goes directly to an attention item for the orchestrator/human.

---

## 15. Wiki / Ledger System

**Status:** Partial resolution (Slice 0 convergence 5.3 + temp.md Q3). Wiki infra seeded in Slice 0 (`WIKI.md`, `WIKI_CHANGELOG.md`, `HYLLA_WIKI.md`); wiki-agent automation (Section 15.1) lands post-Slice-4. Per-slice MD cleanup subslice structural rule added in 15.6 below.

### 15.6. Per-Slice MD Cleanup Subslice (Slice 0 Convergence)

Every slice's closeout (`closeout-slice` per Section 1.4, previously "SLICE <N> END — LEDGER UPDATE") MUST include an **MD cleanup task** that:

1. Scans the slice's shipped work against current MD files (`CLAUDE_MINIONS_PLAN.md`, `REFINEMENTS.md`, `HYLLA_REFINEMENTS.md`, `CLAUDE.md`).
2. Trims entries that landed in this slice — replace the long-form refinement entry with a one-line summary pointing to the slice's `WIKI_CHANGELOG.md` line and the commit SHA.
3. Removes stale sections where a design question got resolved by the slice's work.
4. Runs BEFORE the wiki-updater task (when that lands post-Slice-4) so the wiki aggregator sees current MD state.

This prevents `CLAUDE_MINIONS_PLAN.md` and the refinement logs from accreting resolved cruft. The commit history + `WIKI_CHANGELOG.md` holds the full audit trail, so trimming is safe.

Pre-cascade (now): the orchestrator performs the MD cleanup task manually during slice-end closeout.
Post-cascade (Slice 4+): a dedicated `md-cleanup-agent` subtype fires under the closeout slice and is verified by QA.

### 15.1. Concept

A wiki agent maintains a running summary of all work done at its level. It fires twice:
1. After the plan is accepted (initial wiki entry)
2. After the level is marked `complete` (final wiki entry)

Not on failure — the orchestrator gets failure info directly via attention items.

### 15.2. What the Wiki Contains

- Affected code blocks (from `paths` / `packages`)
- Plan item IDs and their current states
- Code still to be affected (open items)
- Summary of changes made (from completed items' comments)

### 15.3. Hierarchical Absorption

Child wikis are absorbed by parent wikis:
- Build-level wikis are detailed (exact files, symbols, line ranges)
- Slice-level wikis summarize build-level wikis (file groups, feature areas)
- Parent-slice wikis summarize child-slice wikis (feature descriptions, architectural changes)

The further up the tree, the more summarized. This gives the orchestrator a quick view without drowning in detail.

### 15.4. Storage

**Open question:** Where do wikis live?
- Option A: As comments on the plan item (simple, uses existing infrastructure)
- Option B: As a dedicated `wiki` field in plan item metadata (queryable, structured)
- Option C: As a separate wiki table in the DB (most flexible, most work)

**Leaning toward:** Option A (comments) for initial slice, Option B (metadata) for later. Comments are append-only and human-readable. Metadata is structured and queryable.

### 15.5. Orchestrator Memory Compaction

When the orchestrator compacts memory, it should absorb the wiki summaries. The wiki provides a structured, pre-summarized view that's cheaper to load than re-reading all plan items and comments.

**Open question:** How does wiki content integrate with orchestrator memory management? This needs design work.

---

## 16. Quality and Vulnerability Checking

**Status:** **DEFERRED** (Slice 0 convergence 5.1, 2026-04-14). Dev direction: "let's defer small wins tracking to later." Quality / vulnerability checking as a third QA step is post-dogfood territory — revisit after the cascade is self-hosting. The design below is preserved for when we pick it back up; nothing in Slices 1–9 depends on it shipping. Slice 10 refinement cleanup is the earliest realistic landing window.

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
  → All must pass for the build task to be fully verified
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
Human creates a quality-check plan item → fires quality agents
  → Agents scan specified code using Hylla graph nav
  → Report findings as comments
```

Useful for periodic codebase health scans.

### 16.5. Language-Specific

Currently Go-only. Each language will need its own quality agent with language-specific checks. Add a plan item to support more languages when Hylla supports them.

---

## 17. Prerequisites

### 17.1. Hard Prerequisites (Slice 1 — the fresh-project equivalent of "D1 done right")

| Feature | What | Why Required for Cascade |
|---|---|---|
| **Failed lifecycle state** | Fourth terminal state `failed` across domain / app / adapters / storage / config / capture / snapshot | Agents must represent failure. Gates must move tasks to failed. |
| **Outcome required on `failed`** | Moving a task to `failed` requires a non-empty `metadata.outcome` (one of `failure`/`blocked`/`superseded`). Empty `outcome` on terminal `failed` is a domain-level validation error. | Without it, a `failed` task is indistinguishable from a `failed-for-unknown-reason` task, and the cascade can't route. |
| **Parent-blocks-on-failed-child (always-on, not configurable)** | Parent cannot move to `complete` while any child is `failed`. **Not a template flag. Not a policy option.** Always-on built-in behavior. Bypass only via the supersede path (human CLI, orchestrator version post-dogfood). | Core cascade integrity. A configurable version (`require_children_done`) can be set to false and breaks the cascade silently. Remove the knob. |
| **Human supersede CLI** | `till task supersede <id> --reason "..."` — human-only command that marks a `failed` task as `superseded` in `metadata.outcome` and transitions `failed → complete`. Bypasses the terminal-state guard because the CLI asserts human intent at the binary boundary. | Currently the human has no way to resolve stuck `failed` items. Before any cascade runs, the human needs to be able to unstick things. |
| **Auth auto-revoke on terminal state** | Auth session ends when task moves to `complete` or `failed` | Dead agent auth sessions must clean up. "One auth per scope" constraint. |
| **Task details as prompt** | Agent reads task detail fields as its working brief | Simplifies agent prompts — the task IS the prompt. |
| **Plan-item `paths` as first-class field** | `paths []string` on the plan item, planner-set, readable by builder + QA. Domain-level, not buried in metadata JSON. | Plan-QA falsification needs to query siblings' paths to detect cross-task file conflicts (Section 5.3). Without a first-class field, QA has no data to check. Replaces the removed D10 "affected_artifacts". |
| **Plan-item `start_commit` / `end_commit`** | Two fields on the plan item. `start_commit` set at creation (current HEAD). `end_commit` set at move-to-complete (current HEAD). Domain-level. | Needed before the dispatcher takes over commits. Pre-dogfood: orchestrator + dev manage git manually, these fields just record the boundary. Post-dogfood: dispatcher reads these to decide reingest/commit scope. |
| **Creation gated on clean git for declared paths** | At plan-item creation, if any path in `paths` is dirty in `git status --porcelain`, creation fails with an error telling the orchestrator to clean up git first. | Without this gate, a cascade agent (or orchestrator) inherits uncommitted state and silently mixes it into its work. Always-on behavior. |
| **Orchestrator supersede auth (deferred, post-dogfood)** | Programmatic supersede via orchestrator auth (not human CLI). | Only needed when the orchestrator has to unstick things autonomously. Pre-dogfood, the human CLI is enough. Keep this out of Slice 1; it ships in Slice 11. |

### 17.2. Not Prerequisites (Removed from cascade scope)

| Feature | Why Not Needed |
|---|---|
| **Auth claim response enrichment** | Designed for orchestrator-triggered non-headless agents. Headless cascade dispatch passes everything needed in the spawn prompt — no claim-time enrichment required. |
| **`require_children_done` as a configurable policy** | Removed as a knob. Replaced by the always-on behavior in 17.1. Having it as a setting meant the default could (and did) ship as `false`, silently breaking cascade integrity. |
| **Level-based signaling** | Agents fail and die. Dispatcher reads failure. No runtime signaling. |
| **Auth approval loop for cascade agents** | Agent auth is system-issued inside the cascade, no human approval step. |
| **TUI rendering of `failed` (deferred)** | Post-dogfood. Pre-dogfood: the orchestrator exposes failures to the human via a CLI subcommand (`till task list --state failed` or `till failures list`). TUI rendering is nice-to-have, not load-bearing. |

---

## 18. Pre-Build Preparation

Before building the cascade, update the existing workflow to enforce the patterns the cascade will automate.

### 18.1. Update Agent Files for Semi-Formal Reasoning

Update `~/.claude/agents/`:

- `go-qa-proof-agent.md` — add explicit certificate template with mandatory enumeration
- `go-qa-falsification-agent.md` — add hypothesis-refinement loop structure
- `go-planning-agent.md` — add planning certificate with scope/evidence requirements
- `go-builder-agent.md` — add `paths` / `packages` reporting requirements (update once the fields land in Slice 1)

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

**Dev-promoted per slice.** The dev is the only actor that can bump the pinned commit. The promotion happens at a clean boundary — after a slice completes and its QA has cleared. The dev runs `mage install COMMIT=<hash>` explicitly; the cascade never promotes itself.

This is critical for safe dogfooding: when the cascade system is building Tillsyn itself, the installed `till` binary must be a known-good version. If a cascade agent produces broken code and the binary is rebuilt from HEAD, the broken binary could corrupt the cascade. Pinning to a dev-promoted commit breaks that loop.

### 18.6. MCP Passthrough for Headless Agents (Resolved)

Confirmed: `claude --bare -p "..." --mcp-config <path> --strict-mcp-config` accepts an ad-hoc MCP server list and ignores the dev's `settings.json`. See Section 20.6 for evidence and flag details. No pre-build research remaining — this flows directly into the dispatcher design in the cascade slices.

### 18.7. CI Cleanup — Mac-Only Workflows

`.github/workflows/` currently exercises Linux + Windows + macOS matrix runs. Pre-cascade we only dogfood on Mac; the Linux/Windows legs are noise (slower feedback, flaky runner pool, no deployment target uses them). Strip them. Keep only the macOS job(s). Review every workflow file — `ci.yml`, any release / matrix / nightly workflow — and delete Linux/Windows branches, matrix entries, conditional runners, and any OS-specific scripts they reference. If the change removes the only consumer of an action or cached dependency, remove those too. After the cut, `mage ci` locally + a triggered GH Actions run on the cleaned workflows must both stay green. Builder subagent owns this; QA-proof + QA-falsification pass both before commit. Windows/Linux support can return if/when a real deployment target needs it — don't preemptively re-add it.

---

## 19. Development Order

**Slice sequencing principle.** Waves are gone. Everything is a slice, including the prerequisite work. Slice 0 is a fresh-project reset; Slice 1 is the fresh-project equivalent of "D1 done right" from the old plan. Dogfooding turns on as soon as the dispatcher can actually dispatch something (Slice 5).

**Per-slice wrap-up task (applies to every slice below).** At slice end, orchestrator + dev review and update:
- Bare-root `CLAUDE.md`
- `main/CLAUDE.md`
- `~/.claude/agents/*.md` (agent files the cascade or orchestrator actually uses)
- `~/.claude/CLAUDE.md` only if a global rule changed

**No subagents on this wrap-up.** Orchestrator and dev decide directly. Keep the docs aligned with current code state so the next slice's subagents aren't briefed on stale rules.

**Template constraint (applies throughout).** The `default-go` template structure gets trimmed to what the cascade actually reads. Templates bind kinds to: agents, models, effort levels, tools, budgets, turns, gates, child-rules, trigger state, escalation, and push policy — nothing more. Don't add fields the dispatcher doesn't consume.

**Git management pre-dispatcher.** Until the dispatcher's commit logic lands (post-dogfood refinement, Slice 11), the **orchestrator and dev manage git manually**: the orchestrator reminds the dev to clean up dirty paths before a plan item is created, and the dev handles the actual commits. Plan items still carry `start_commit` / `end_commit` fields from Slice 1, but those fields are records, not triggers, until the dispatcher is wired up.

### 19.0. Slice 0 — Project Reset + Docs Cleanup

Before any cascade code lands:

- [ ] Delete the current messy Tillsyn project in Tillsyn (`a0cfbf87-b470-45f9-aae0-4aa236b56ed9`).
- [ ] Create a **fresh Tillsyn project with NO template bound.** The cascade plan itself will define what template (if any) gets attached later; starting without one avoids inheriting `default-go`'s bloat into the planning work.
- [ ] Full rewrite of bare-root `/Users/evanschultz/Documents/Code/hylla/tillsyn/CLAUDE.md` and `main/CLAUDE.md`:
  - Remove references to `till.get_instructions`, `/tillsyn-bootstrap`, and the "bootstrap guide" flow.
  - Remove "Features Being Implemented (D1-D10)" section — replace with a one-line pointer to `CLAUDE_MINIONS_PLAN.md` for the cascade plan.
  - Tighten the Tillsyn-usage and Go-idiom sections; drop anything the cascade plan now covers.
  - Make explicit that orchestrator + dev manage git manually until the dispatcher lands (Slice 11+).
  - Target size: bare-root ~200 lines, `main/CLAUDE.md` ~40 lines.
- [ ] Full rewrite of `~/.claude/agents/*.md` for the cascade-dispatched agents:
  - Remove any references to `till.get_instructions` / bootstrap.
  - Bake in the semi-formal reasoning certificate template (Section 11.1) where it belongs.
  - Tighten headless-invocation expectations (agents run in `--bare -p "..."` with a scoped `--mcp-config`).
  - Target size per agent: ~60-80 lines.
- [ ] Update agent files for semi-formal reasoning specifics (18.1 refinements on top of the rewrite).
- [ ] Add `mage test-func` target (18.3).
- [ ] Audit path logic in TUI, plan backend refactoring (18.4).
- [ ] Add `mage install` with dev-promoted commit pinning (18.5).
- [ ] MCP passthrough for headless agents — **already resolved** (Sections 18.6, 20.6). No pre-build research remaining.
- [ ] CI cleanup — strip Linux/Windows from `.github/workflows/`, keep macOS only (18.7). Full QA (proof + falsification) because CI is load-bearing.
- [ ] **Mid-slice additions** *(added during Slice 0 execution — detail tracked in Tillsyn plan items, not re-described here)*:
  - **18.10 gofumpt adoption** — committed `d684dcb`; required 18.10B follow-up because of cold-cache leak.
  - **18.10B fix cold-cache `mage ci` gofumpt gate** (`runGofumptList` + `trackedGoFiles` stdout/stderr split; `wrapCommandErrorWithStderr` for error paths). Ships with 18.7 in a single push so post-push CI is macos-only and green.
  - **18.11 auth-cache `SessionStart`-hook MVP** — shipped; read-side cache-inject on resume/compact/startup. **18.11B `PostToolUse`-hook auto-persist** — shipped; removes manual-Write discipline. Retroactively captured as plan items post-ship.
  - **18.12 fix gopls build-tags for `magefile.go`** — **closed without shipping (2026-04-14)**. Initial builder landed `.vscode/settings.json` with `gopls.build.buildFlags = ["-tags=mage"]`, but the premise was wrong: the dev uses nvim (not VS Code) on this repo, and gopls does not auto-read project-root `.vscode/settings.json` — its config comes from the LSP client (nvim-lspconfig for the dev, the `gopls-lsp@claude-plugins-official` plugin for Claude Code). The checked-in file would not have affected either runtime. File was reverted; `.vscode/` is now ignored alongside other editor cruft. Real fix (if still needed) belongs in editor-side config, not the repo tree.
- [ ] **Per-slice wrap-up:** confirm the rewritten CLAUDE.md and agent files match the plan post-cleanup.

### 19.1. Slice 1 — Failed Lifecycle State (Fresh-Project "D1 Done Right")

The hard prerequisites from Section 17.1, shipped cleanly against the fresh project. This is the foundation Slice 2+ sits on. Each item must pass `mage ci` + QA proof + QA falsification before it's marked complete.

- [ ] **Install local git hooks for gofumpt + `mage ci` parity** *(Slice 0 refinement, scheduled here as Slice 1 first item)*. Add committed `.githooks/pre-commit` that runs a new `mage format-check` target (public wrapper around the existing private `formatCheck()` in `magefile.go:218-236`) and `.githooks/pre-push` that runs `mage ci` in full. Add a `mage install-hooks` target that sets `core.hooksPath = .githooks` so the tracked hook scripts become the active hooks for any fresh clone. Also fix the `mage format` no-arg ergonomics wart discovered in Slice 0 closeout: `func Format(path string) error` (`magefile.go:200`) requires a positional arg, making the `path == "" || path == "."` branch in the body unreachable from CLI (`mage format` errors with "not enough arguments"); split into `Format()` (no-arg = whole tree via `trackedGoFiles()`) and `FormatPath(path string)` (scoped), or adopt a variadic form. Motivation: Slice 0 surfaced that gofumpt drift landed on `main` because no local gate catches it pre-commit — `mage ci` is the CI-parity gate but runs too late to prevent red pushes. Hooks must remain bypassable via `--no-verify` per existing discipline (global CLAUDE.md rule: never bypass without explicit dev instruction). QA-proof + QA-falsification required — the hook scripts are the local build gate, can't silently break.
- [ ] `failed` lifecycle state (fourth terminal state) across domain / app / adapters / storage / config / capture / snapshot. Fix the HEAD gaps (gofmt regression in `app_service_adapter_outcome_test.go`, empty-outcome acceptance in `validateMetadataOutcome`).
- [ ] Require non-empty `metadata.outcome` on any transition to `failed` (domain-level validation error, not just value whitelist).
- [ ] **Remove** `require_children_done` as a configurable option. Replace with always-on parent-blocks-on-failed-child behavior enforced at every hierarchy level. No template flag, no policy knob. Bypass only via the supersede path below.
- [ ] Human supersede CLI: `till task supersede <id> --reason "..."` — marks `failed` task as `metadata.outcome: "superseded"` and transitions `failed → complete`. Bypasses the terminal-state guard because the CLI asserts human intent at the binary boundary.
- [ ] Auth auto-revoke on terminal state (`complete` or `failed`).
- [ ] **Server-infer `client_type` on auth request create** *(gap surfaced in Slice 0)*. Remove `client_type` from the `till.auth_request(operation=create)` MCP tool schema — callers shouldn't declare transport; the server knows. Entrypoint adapters stamp it: MCP-stdio adapter stamps `"mcp-stdio"`, TUI stamps `"tui"`, CLI stamps `"cli"`. Tighten `app.Service.CreateAuthRequest` to reject empty `ClientType` at create time (matches the existing approve-path check in `autentauth.Service.ensureClient`) so the asymmetric validation bug that bit Slice 0 — create accepted empty, approve rejected empty with `ErrInvalidClientType` — is structurally unreachable. Governance + display still consume `client_type` as a first-class field; only the caller responsibility moves server-side. MCP-layer tests drop the field; domain-layer tests keep it on `CreateAuthRequestInput` since that's the domain boundary. `client_id` stays caller-supplied (same transport can come from different software).
- [ ] **Reject unknown keys across all MCP mutation paths** *(gap surfaced in Slice 0)*. `till.project(operation=create)` silently dropped every non-schema key in my Slice-0 metadata payload — caller thought fields landed, they didn't. Same asymmetric-validation pattern as the `client_type` bug above. Audit every `till.*` mutation tool (`till.project`, `till.plan_item`, `till.comment`, `till.handoff`, `till.attention_item`, `till.capability_lease`, `till.kind`, `till.template`) and every nested metadata/extension object each one accepts. Every MCP handler must reject unknown keys with a structured error naming the offending key and the accepted schema — never silent-drop. If extension-style freeform fields are wanted for any surface, add an explicit named `extensions map[string]string` (or equivalent) to the domain type so it's documented and validated, not an anything-goes sink. Add golden tests asserting the error shape for each handler. Scope note: this is the *validation* fix; adding new first-class cascade fields to the project node is Slice 4's dispatcher prerequisite, not this item.
- [ ] **PATCH semantics on all update handlers — no more silent full-replace** *(gap surfaced in Slice 0; second repro confirmed in 18.2 closeout)*. `till.project(operation=update)` with a partial payload (only `name` + `metadata`) wiped the stored `description` back to empty string — the handler is full-replace without documenting it. Second silent-data-loss bug in the same family as unknown-key drop above. **Live second repro from 18.2 closeout (2026-04-14)**: `till.plan_item(operation=update)` on task `f4334081-84ad-47a4-bcf9-238c2f915ad2` passing only `title` + `metadata` wiped `description` (full rewrite contract) and `labels` (`["agents","docs","orchestrator-scope","slice-0"]`). Confirms the behavior is handler-family-wide, not project-only. Audit every `till.*` update/mutation handler for the same behavior (`till.project`, `till.plan_item`, `till.comment`, `till.handoff`, `till.attention_item`, `till.kind`, `till.template`). Pick ONE semantics per handler and enforce it: either (a) true PATCH — only provided fields change, omitted fields preserved — which matches caller intuition and is strongly preferred, or (b) explicit full-PUT with a required `replace_all: true` flag that forces the caller to acknowledge they are overwriting. Never silently wipe fields because the caller didn't repeat them. Preserve the Slice 0 precedent in tests: `update(name, metadata)` must leave `description` intact; `till.plan_item.update(title, metadata)` must leave `description` + `labels` intact. **Third repro (2026-04-14, 18.10B closeout)**: builder on 18.10B + 18.7 hit it again — a `till.plan_item.update(title, ...)` with no `description` arg cleared 18.7's stored description; builder worked around by re-calling update with the full original description restored. Evidence that every orchestrator / builder round-trip through update is a latent data-loss risk until this lands.
- [ ] **Accept `state` in place of `column_id` on `till.plan_item(operation=create)`; stop leaking column UUIDs into the agent contract** *(gap surfaced in Slice 0)*. Fresh project had auto-seeded default columns (`To Do`, `In Progress`, `Done`) but `till.plan_item(op=create)` rejects the call unless the caller passes the literal column UUID — and no MCP op exposes column UUIDs (`till.capture_state` loads them for state-hashing but does not surface them; there is no `list_columns` operation). An orchestrator following MCP-only discipline has no way to discover the UUID. Column identity is a UI/layout concern; agents only care about lifecycle state. Fix: `till.plan_item(op=create)` must accept `state` (`todo` / `in_progress` / `done` / `failed` once Slice 1 adds it) and resolve the column UUID server-side via the existing `resolveTaskColumnIDForState` helper (`internal/adapters/server/common/app_service_adapter_mcp.go:811`). Keep `column_id` accepted for TUI drag-and-drop callers that genuinely know the UUID, but make `state` the documented agent-facing input and reject the call only when *both* are empty. Same cleanup on `till.plan_item(op=move)` where `to_column_id` currently faces the same leak — accept `state` and resolve internally. Add a golden test proving an orchestrator with no column knowledge can create a plan item purely by `state`. No column-listing MCP op needs to be added; the goal is to make column IDs invisible to the agent surface, not to expose them.
- [ ] Task details as prompt (agent reads task fields as working brief).
- [ ] First-class `paths []string` field on plan items (planner-set, readable by builder + QA). Domain-level field, not buried in metadata JSON. Replaces the removed `affected_artifacts`.
- [ ] First-class `packages []string` field on plan items (covers every file in `paths`). Used by package-level blocking (Section 5.2).
- [ ] First-class `start_commit` / `end_commit` fields on plan items. `start_commit` set at creation (current HEAD). `end_commit` set at move-to-complete (current HEAD).
- [ ] Creation gated on clean git for declared paths: if any path in `paths` is dirty in `git status --porcelain`, creation fails with an error telling the orchestrator/dev to clean git first. Always-on.
- [ ] CLI failure listing: `till task list --state failed` (or `till failures list`) so the human can see `failed` tasks without TUI rendering. TUI rendering of `failed` is deferred post-dogfood.
- [ ] **Deferred post-dogfood (documented here, not built yet):** orchestrator programmatic supersede via system-issued auth. Human CLI is enough for Wave-1-equivalent scope.
- [ ] **Per-slice wrap-up:** update CLAUDE.md + agent files to reflect the new required fields, the always-on block behavior, and the supersede CLI.

### 19.2. Slice 2 — Hierarchy Refactor

This is the touchiest code change. Each step ripples through 5+ packages. Incremental, `mage ci` + reingest after each step.

- [ ] **Remove `branch`**: delete from schema, domain, TUI, templates. Migrate existing branches to top-level slices.
- [ ] **Rename `phase` → `slice`**: schema migration, domain constants, TUI labels, MCP adapter normalization, templates, docs.
- [ ] **Rename `done` → `complete`**: DB column values, domain `StateComplete`, TUI labels, MCP normalization, templates, docs. Combine with any leftover `failed` state migration since they touch the same surfaces.
- [ ] **Allow infinite slice nesting**: update domain validation to allow slice-under-slice. Update TUI tree rendering.
- [ ] **Per-slice wrap-up:** update CLAUDE.md + agent files for the new vocabulary.

**Order matters:** Remove `branch` first (least entangled), then `phase → slice` (more entangled but no state-machine changes), then `done → complete` (most entangled, touches state machine + `failed` state).

### 19.3. Slice 3 — Template Configuration

- [ ] Add agent binding fields to kind definitions (`agent_name`, `model`, `effort`, `tools`, etc.).
- [ ] Add gate definitions to kind templates.
- [ ] Add `max_tries`, `max_budget_usd`, `max_turns`, `auto_push`, `commit_agent` to kind definitions.
- [ ] Add `blocked_retries`, `blocked_retry_cooldown` to kind definitions.
- [ ] Template parsing and validation for new fields.
- [ ] Build a fresh `default-go` template (or equivalent) aligned to the cascade — do not resurrect the current bloated one.
- [ ] **Per-slice wrap-up:** update CLAUDE.md + agent files.

### 19.4. Slice 4 — Dispatcher Core

The minimal dispatch loop, now that lifecycle + template fields exist.

- [ ] Refactor path logic from TUI to backend (18.4 output).
- [ ] **First-class project-node fields the dispatcher reads** *(prerequisite; replaces the old single-field `project_dir` bullet)*. Add these as domain-level fields on `Project`, not metadata JSON, each with explicit validation: `hylla_artifact_ref` (string, e.g. `github.com/evanmschultz/tillsyn@main`), `repo_bare_root` (abs path), `repo_primary_worktree` (abs path — the `cd` target for dispatched agents; supersedes the old `project_dir` concept), `language` (enum matching agent variants: `go`, `fe`, …), `build_tool` (string: `mage`, `npm`, `cargo`, …), `dev_mcp_server_name` (string — which MCP server dispatched agents register against). Planner fills these at project create; dispatcher reads them to spawn agents with correct `cd`, correct `{lang}-builder-agent` / `{lang}-qa-*-agent` variant, correct artifact ref in the prompt, correct MCP server registration. Fields *not* on the project node: `agent_bindings` + `post_build_gates` + kind vocabulary → template-scope (Slice 3); `current_slice` → already encoded by the `kind=slice` plan item in `state=in_progress`, no field needed; `go_version` → derive from `go.mod`, don't duplicate. Depends on the Slice 1 metadata-validation tightening so unknown keys surface as errors instead of silent drops.
- [ ] Implement dispatcher: LiveWaitBroker subscription for state changes.
- [ ] Implement agent spawn: `cd <project_dir> && claude --agent <type> --bare -p "..." --mcp-config <per-run mcp.json> --strict-mcp-config --permission-mode acceptEdits --max-budget-usd <N> --max-turns <N>`.
- [ ] Implement agent lifecycle: auth issuance, process monitoring, cleanup on state change.
- [ ] Implement auto-promotion: when blockers clear, move eligible items to `in_progress`.
- [ ] File-level AND package-level blocking: lock acquisition, conflict detection, dynamic `blocked_by` (Section 5).
- [ ] Git status pre-check before builder dispatch (reuse the Slice 1 gate).
- [ ] Gate execution: run template-defined gates after builder completion.
- [ ] Commit agent (haiku) integration for commit message formation.
- [ ] Git commit + optionally push as programmatic step — with the no-fallback, orchestrator-escalation path from Section 9.3. **Dispatcher writing to git is still opt-in at this slice;** dev can leave it off and keep managing git manually until confidence lands.
- [ ] Hylla reingest as programmatic gate.
- [ ] **Per-slice wrap-up:** update CLAUDE.md + agent files.

### 19.5. Slice 5 — Cascade Planning (Dogfooding Begins)

From here on, the cascade can build itself.

- [ ] Planner agent integration: agent creates child plan items via MCP.
- [ ] Planning QA integration: plan-qa-proof and plan-qa-falsification auto-fire.
- [ ] Plan acceptance flow: plan QA pass → children become eligible.
- [ ] Template child_rules for plan-task → plan-qa-proof + plan-qa-falsification.
- [ ] Validate the cascade flow end-to-end on real Tillsyn work (start dogfooding).
- [ ] **Per-slice wrap-up:** update CLAUDE.md + agent files.

### 19.6. Slice 6 — Escalation

- [ ] Retry tracking per plan item (attempt count in metadata).
- [ ] Re-fire on failure (up to max-tries).
- [ ] External failure detection + blocked retries (max-tries + 2).
- [ ] Escalation up: failed build → new plan task at parent level.
- [ ] Escalation tracking: plan diff, failure context documentation.
- [ ] Template configuration for escalation_enabled.
- [ ] **Per-slice wrap-up:** update CLAUDE.md + agent files.

### 19.7. Slice 7 — Error Handling and Observability

- [ ] Detect external failures (network, API limits, resource exhaustion).
- [ ] Distinguish `blocked` (external) from `failure` (agent error).
- [ ] Stale process detection (auth TTL expiry).
- [ ] Attention item routing for different failure types.
- [ ] Failure communication to human (specific error details).
- [ ] **Per-slice wrap-up:** update CLAUDE.md + agent files.

### 19.8. Slice 8 — Wiki / Ledger System

- [ ] Design wiki storage (comments vs metadata vs dedicated table).
- [ ] Wiki agent: fires after plan acceptance and after completion.
- [ ] Hierarchical wiki absorption (child → parent summarization).
- [ ] Orchestrator memory integration.
- [ ] **Per-slice wrap-up:** update CLAUDE.md + agent files.

### 19.9. Slice 9 — Quality and Vulnerability Checking

- [ ] Design quality check agent and certificate.
- [ ] Hylla graph-nav-based resource lifecycle verification.
- [ ] Configurable replicas (N agents, consensus policy).
- [ ] Standalone mode (independent quality scans).
- [ ] Go-specific checks (goroutine lifecycle, error handling, defer patterns).
- [ ] **Per-slice wrap-up:** update CLAUDE.md + agent files.

### 19.10. Slice 10 — Refinement Cleanup (post initial dogfood)

After real dogfooding reveals what works and what doesn't.

- [ ] Second-pass review of the cascade-bound template: trim unused fields, align with what the cascade actually reads after shipping.
- [ ] Second-pass review of `~/.claude/agents/*.md`: trim to what cascade-dispatched agents actually need, keeping orchestrator-side agents separate.
- [ ] Second-pass review of `CLAUDE.md` (global + project): remove items the cascade now handles.
- [ ] Shrink orchestrator-side slash-command + skill surface to the minimum needed after cascade takes over most coordination.
- [ ] **Full `magefile.go` cleanup + refine pass** *(deferred from Slice 0)*. Slice 0 added `mage test-func` (18.3) and `mage install` with commit pinning (18.5) as point additions without touching the rest. Do a full sweep: consolidate duplicated invocation helpers, normalize target naming (`test-pkg` vs `test-func` vs `test-golden` vs `ci` — are the prefixes + argument shapes consistent?), prune any dead or stub targets, verify every target has a one-line `mg:` doc comment, confirm no target shells out to a raw `go` command (force everything through a single `runGo` helper), and make sure `mage -l` output reads like a coherent menu. QA-proof + QA-falsification required — the magefile is the build gate, can't silently break.
- [ ] **`mage install` post-MVP retire-or-reshape** *(deferred from Slice 0 fix-finish)*. Slice 0's closeout reshaped `mage install` to the minimum dogfood-safe shape: single required positional arg `sha` (never resolved from `git HEAD`, empty errors out), temp `git worktree add --detach <sha>`, `go build` with `-X ...buildinfo.Commit=<sha>` ldflag, install binary to hardcoded `$HOME/.tillsyn/till` colocated with `config.toml` / `tillsyn.db` / `logs/`, defer cleanup of both the worktree and the temp root. **Enforcement for "dev-only, never agents" is docstring + `CLAUDE.md` rule only** — no tool-permission deny, no env-marker guard, no code-level check. This target exists solely because we're pre-MVP and need a stable `till` on the dev box to orchestrate the cascade against. Once MVP ships with proper release artifacts (goreleaser snapshot is already wired in CI), decide: retire `mage install` entirely in favor of `gh release download` + install, or keep it as a dev convenience with the shape above. Whichever: no pin-log file, no printer notice, no helpers — if it stays, it stays at ~30 lines.
- [ ] **Agent tool-permission deny for `mage install`** *(refinement-only, add when needed)*. If an agent ever tries to invoke `mage install` despite the CLAUDE.md rule, add `Bash(mage install*)` to the `tools: { deny: [...] }` block in every `~/.claude/agents/*.md` (builder, QA, research, planning variants) and record the incident so we know the docstring-only enforcement was insufficient. Until that happens, the docstring rule is load-bearing and good enough — no proactive guardrail work.
- [ ] **Project lifecycle operations — delete + archive** *(gap surfaced in Slice 0)*. Add `till.project(operation=delete)` MCP op and corresponding `till project delete` CLI. Guard: project must have no active auth sessions or leases, no in-flight cascade runs. Must cascade-clean plan items, comments, handoffs, attention items, template bindings, embeddings, capture snapshots — audit FK coverage and add explicit cleanup where `ON DELETE CASCADE` is missing. Also add `till.project(operation=archive)` MCP op + `till project archive` CLI that flips the archived flag already surfaced in `include_archived` list filter (preserves data, hides from default list). Slice 0 worked around the missing delete by renaming the messy pre-cascade project to `TILLSYN-OLD`; retire that renamed project via `delete` (or `archive` if we want to keep the old data for comparison) once these ops ship.
- [ ] **Compaction-resilient auth cache — refinement pass** *(MVP shipped in Slice 0 as 18.11; this is the follow-up hardening)*.

  **Background — what Slice 0 shipped.** The MVP solves orchestrator auth loss across context compaction using a `SessionStart` hook + a per-project file cache. No keychain. The full design:

  - **Cache file**: `~/.claude/tillsyn-auth/<project-uuid>.json`, mode 0600, inside `~/.claude/tillsyn-auth/` mode 0700. One file per `(project-uuid, role)`, overwrite-in-place on every fresh claim, no stale accumulation.
  - **Payload**: `{project_id, role, session_id, session_secret, auth_context_id, agent_instance_id, lease_token, request_id, expires_at, claimed_at}`. Timestamps are RFC3339 UTC `Z` form with fractional seconds stripped.
  - **Write path — orchestrator behavioral rule**: on every successful orchestrator-role `till.auth_request(operation=claim)` + subsequent `till.capability_lease(operation=issue)`, the orchestrator `Write`s the full bundle to cache **before** any other Tillsyn work. Enforced by auto-memory `feedback_orchestrator_auth_cache.md` — orchestrator-scope only; subagents never load this memory, never write or read the cache.
  - **Read path — SessionStart hook**: `~/.claude/hooks/session_start_tillsyn_auth_inject.sh` (registered in `~/.claude/settings.json` under `hooks.SessionStart` with `matcher: "startup|resume|compact"`). On fire, the script scans the cache dir, parses `expires_at`, deletes expired entries (reactive GC), and for valid entries emits `{"hookSpecificOutput": {"hookEventName": "SessionStart", "additionalContext": "<tillsyn-auth-cache>...</tillsyn-auth-cache>"}}`. The orchestrator sees the bundle in its first turn after resume/compact.
  - **Orchestrator use**: on seeing the injected `<tillsyn-auth-cache>` block, validate via `till.auth_request(operation=validate_session)` before use. On validation failure (revoked, server-side expired), delete the cache file and fall through to asking the dev — captured in `feedback_auth_after_compaction.md`.
  - **Subagent isolation model**: nondiscoverability, not access control. The cache path, hook script path, and read/write commands appear only in orchestrator auto-memory under `~/.claude/projects/<parent-session-hash>/memory/`. They are **never** in any `CLAUDE.md` (global or project), any `~/.claude/agents/*.md`, any slash command, or any spawn prompt template. `SessionStart` hooks fire only for top-level CLI sessions — Agent-tool subagents don't fire them, so `additionalContext` injection reaches only the orchestrator. A Bash-equipped subagent could still `ls ~/.claude/` and find it; isolation is sufficient for accident prevention, not against a rogue subagent.
  - **TTL'd secret**: every entry carries `expires_at` and self-deletes on expired read; even a leaked cache file is short-lived.
  - **Why file + hook and not keychain**: keychain has no native TTL, macOS-only, touches the user's personal keychain, and the ACL-by-app advantage doesn't apply from inside the Bash tool. File-backed is simpler, portable across platforms, auditable, naturally TTL'd, and has equivalent practical isolation.

  **This is the documented generalized solution for any Tillsyn auth-loss-through-compaction issue.** Any future role (builder, qa, research) facing the same problem uses the same pattern — different filename suffix, same hook script, same behavioral rule in a role-specific memory file.

  **Refinement work for this slice (not MVP):**

  - [ ] **Extend to subagent roles** if subagent compaction becomes a real pattern. Today subagents are per-spawn and don't compact, so builder/qa/research caches are out of scope; revisit if that changes.
  - [ ] **Harden subagent isolation**: consider whether to move cache under a less-enumerable path, or whether to use a deny-list in subagent permissions to block `Bash:ls ~/.claude/tillsyn-auth/` and `Read:~/.claude/tillsyn-auth/**`. Probably YAGNI unless we see a real leak; recorded here so the option is known.
  - [ ] **Expired-entry reaper**: today GC is reactive (on read). Add a lightweight sweep hook (`SessionEnd` or cron-equivalent) if files pile up — also probably YAGNI given overwrite-in-place semantics, but recorded.
  - [ ] **Cache-hit telemetry**: a `SessionStart` emit line the hook writes to `~/.claude/hooks/hook-execution.log` so we can see how often the cache saves a round-trip vs. falls through. Useful for validating the design holds up.
  - [ ] **Cross-project orchestrator sessions**: today the hook injects every valid bundle. If the orchestrator works on multiple projects concurrently it gets multiple bundles injected; fine today, may want cwd-scoped filtering later.
  - [ ] **Port to Linux / Windows** if the project ever runs there — the file semantics are portable; only the `date -j -u -f` parsing in the hook script is macOS-specific and needs a GNU-date alt path.
  - [ ] **Integration with `till auth session show`**: today the CLI's `show` command redacts the secret. If a user wants to seed the cache for a pre-existing session, they can't. Add `till auth session reveal --session-id <id>` (interactive, requires TTY, logs an attention item) so seeding works without re-claiming. Lower priority — the cache is normally written at claim time where the secret is already in hand.
- [ ] **Per-slice wrap-up:** update CLAUDE.md + agent files.

### 19.11. Slice 11 — Dispatcher Git Ownership (Post-Dogfood Refinement)

Move git responsibility from orchestrator+dev to the dispatcher.

- [ ] Dispatcher performs all commits for plan items it dispatched (commit agent handles the message; no deterministic fallback; commit-agent failure escalates to orchestrator CLI tool).
- [ ] Dispatcher reads `start_commit` / `end_commit` fields to decide reingest scope.
- [ ] Orchestrator CLI tool for manual commit override when the commit agent escalates.
- [ ] Orchestrator programmatic supersede via system-issued auth (the post-dogfood supersede path from Slice 1's deferred list).
- [ ] TUI rendering of `failed` tasks (deferred from Slice 1).
- [ ] Update CLAUDE.md to remove the "orchestrator + dev manage git manually" language.
- [ ] **Per-slice wrap-up:** update CLAUDE.md + agent files.

### 19.12. Slice 12 — `depends_on` Removal (Dogfooding Test)

- [ ] Remove `depends_on` from schema, domain, app, adapters, TUI.
- [ ] Confirm `blocked_by` + parent-child hierarchy fully replaces it.
- [ ] Intentionally last — it's a real integration test of the cascade system itself building a cascade-relevant change.
- [ ] **Per-slice wrap-up:** update CLAUDE.md + agent files.

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

### 20.3. Wiki Design

**Q2:** What is the wiki's shape and storage? See Section 15.4. Needs design work.

**Q3:** How does wiki content integrate with orchestrator memory compaction?

### 20.4. Quality Check Design

**Q4:** What specific Go quality checks should be in the initial set? Resource lifecycle, error handling, goroutine safety — what else?

**Q5:** How do we handle false positives in quality checks? The graph analysis might flag valid patterns as issues. What's the escalation path?

### 20.5. Escalation Depth

**Q6:** How deep can escalation nest? If a build fails → re-plan → build fails again → ?

Current design: `max-tries=2` at each level, then attention item. But what about the re-plan level? Can the re-plan itself fail and escalate further?

**Suggestion:** One level of escalation. Build fails → re-plan → build fails → human. No deeper nesting. Configurable in template.

### 20.6. MCP Config Passthrough for Headless Agents (RESOLVED)

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

### 20.7. Template Kind Expansion

**Q8:** The cascade introduces new kind types that may not exist in `default-go` yet:
- `plan-task` (planner agent fires here)
- `plan-qa` (planning QA agent fires here — or reuse existing `qa-check`?)
- `quality-check` (quality/vuln agent fires here — see Section 16, DEFERRED)
- `commit-agent` (haiku commit message agent)

**Partial resolution (Slice 0 5.2, 2026-04-14):** the **type-slice kinds** (`plan-slice`, `build-slice`, `qa-slice`, `closeout-slice`, `refinement-slice`, `human-verify-slice`, `discussion-slice` — see Section 1.4) are the authoritative vocabulary. Dev direction: pre-Slice-3, use the **generic `slice` kind + adhoc creation** for refinement and discussion work — update existing slices in place rather than fragmenting. Slice 3 promotes the type-slice kinds to real template kinds with `child_rules`.

Still open for Slice 3: the per-kind `child_rules` wiring (e.g. does `plan-qa` reuse `qa-check` or become its own kind), and how the dispatcher chooses the agent binding from the slice type.

### 20.8. Orchestrator Role in Cascade

**Q9:** What does the orchestrator do during an active cascade?

Current design: orchestrator runs `/loop` polling for attention items (failures, escalations). The cascade runs autonomously until something fails. The orchestrator's job is:
- Start cascades (move slice to `in_progress`)
- Handle failures (review attention items, decide fix vs. supersede)
- Review wiki summaries
- Make design decisions the cascade can't

Is this sufficient? Does the orchestrator need more visibility during a running cascade?

### 20.9. Plan Item State Machine for Gates (RESOLVED)

**Q10:** When a build agent moves its task to `complete` and then a gate fails, how does the task get to `failed`?

**Resolution:** The builder moves to `complete`. Gates run. If a gate fails, the dispatcher uses **override auth** to move `complete → failed`. The `complete → failed` transition requires override auth, which the dispatcher has (system-issued). This uses existing mechanisms — no intermediate states, no new state transitions. Override auth is already designed for exactly this kind of system-level state correction.

### 20.10. Planning Agent Auth Scope

**Q11:** The planner agent needs to create child plan items via `till.plan_item(operation=create)`. But its auth is scoped to its own plan-task. Creating children on the SLICE (the plan-task's parent) requires broader scope.

Options:
- a) Planner's auth is scoped to the slice, not just its plan-task
- b) Planner creates children under its own plan-task, and the dispatcher re-parents them
- c) Planner creates children under the slice via a dedicated "create-child-on-parent" MCP operation

**Leaning:** (a) — the planner needs slice-scoped auth because its job is to decompose the slice. Template configuration specifies the auth scope for each kind.

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
- `main/TILLSYN_FIX_PROMPT.md` — Historical document listing the pre-cascade fix decisions (failed lifecycle state, outcome metadata, override auth, auth auto-revoke, `require_children_done`, task details as prompt). Hard prerequisites for the cascade.
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
