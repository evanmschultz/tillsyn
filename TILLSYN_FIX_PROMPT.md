# Tillsyn Agent — Coordination Model Fix

This document is the prompt for a Tillsyn agent to fix issues discovered during the Sjal Phase 1 scaffolding. It covers structural changes to Tillsyn's lifecycle model, auth model, agent coordination patterns, and documentation/instruction updates.

## Context — What Happened

During Phase 1 of the Sjal project (an Astro + SolidJS component library), the orchestrator ran into these concrete problems:

### Problem 1: Subagent Prompt Bloat

The orchestrator wrote 700+ word prompts to subagents (planner, QA proof, QA falsification) repeating all plan details, version tables, command lists, and task context. This is wrong — Tillsyn task details should carry the context. Subagent prompts should be ~100 words: auth credentials + task ID + "read your task details."

**Root cause**: No documented pattern for "task details are the prompt." Agents didn't know to read `plan_item(operation=get)` for their instructions. The `get_instructions` tool doesn't teach agents HOW to use Tillsyn operationally.

### Problem 2: No `failed` Lifecycle State

Tasks only have `todo`, `in_progress`, `done`. When work fails (QA falsification found a plan break, a build step errors), there's no way to represent "this attempt was made and failed." The workaround was creating new tasks, but the old task had no terminal failure state.

**Impact**: Orchestrators can't query for failed tasks. Task state doesn't distinguish success from failure. The outcome has to be inferred from comments or metadata.

### Problem 3: Role-Gated Task Completion Too Rigid

The orchestrator couldn't close tasks like "CONTEXT7 AND MDN RESEARCH" (completable by: human, research) even though the research subagent had clearly done the work and returned. The phase couldn't be marked `done` because these child tasks were stuck.

**Impact**: Orchestrator jams on role-gated tasks. Phase transitions stall. The orchestrator has to spin up brief research/qa sessions just to close tasks that are conceptually done.

### Problem 4: @Mention System Overhead for Ephemeral Agents

Subagents are ephemeral — they spawn, work, die. They never check attention items. The @mention system creates attention items that no one reads. The orchestrator's prompt told subagents to post `@orchestrator` comments, but the orchestrator just read the subagent's return value instead.

**Impact**: Unused attention items accumulate. The coordination model assumes persistent agents that poll, but subagents are ephemeral handlers.

### Problem 5: Auth Cleanup

Stale orchestrator leases from Phase 0 blocked Phase 1's orchestrator lease. There was no automatic cleanup when sessions expired or agents died.

**Impact**: "overlapping orchestrator lease blocked" error. Manual revoke_all needed.

### Problem 6: Bootstrap Flow Undocumented

The auth → lease → phase move → task update → subagent delegation flow had to be reinvented from scratch. No documented bootstrap sequence exists. The first attempt used a builder role for the parent session (wrong — should be orchestrator).

---

## Design Decisions Made

These decisions are confirmed by the dev and should be implemented:

### D1: Four Lifecycle States

Add `failed` as a fourth terminal state alongside `done`:

- `todo` → `in_progress` → `done` (success)
- `todo` → `in_progress` → `failed` (failure)

`failed` is a terminal state like `done` — work was attempted but didn't succeed. It is NOT the same as archived or cancelled. The task retains its full context (metadata, comments, history) for traceability.

**SQL/schema change**: Add `failed` to the lifecycle_state enum. Keep full history of state transitions. `failed` tasks should be queryable via `plan_item(operation=search, states=["failed"])`.

### D2: Task State as Signal (Not @Mentions)

Subagents are ephemeral. They don't poll attention items. The coordination model changes:

- **Subagents** read task details at spawn, do work, update task metadata with outcome, move task to `done`/`failed`, post a comment, die.
- **Orchestrator** reads task state + metadata after subagent returns. No @mention polling needed for subagent results.
- **Attention items** are used ONLY by orchestrators (for human approvals and inter-orchestrator communication).

**Template/instruction change**: Remove any guidance telling subagents to check attention items. Update role model docs to reflect ephemeral subagent pattern.

### D3: Item-List-Based Short-TTL Override Auth

When the orchestrator needs to close role-gated items (e.g. "completable by: research") or mark items `done` when children include `failed` tasks, it requests a **short-TTL, item-list-scoped auth** approved by a human.

Key design:

- **Item-list-based**: The auth request specifies the **full list of item IDs** it applies to. The auth is scoped to exactly those items — not a blanket role escalation.
- **Short TTL (2–5 minutes)**: The auth expires automatically. The orchestrator can't hold it open.
- **Human-approved**: Every override auth requires human approval. No auto-approval.
- **Applies to ALL plan items**: Any plan item with `require_children_done` can't be marked `done` when children include `failed` items without this auth. This is not phase-specific — it applies at every level of the hierarchy.

Use cases:
1. Closing a role-gated task when the work was done by a different role (e.g. orchestrator closing a "completable by: research" task after research subagent returned).
2. Marking a parent `done` when some children are `failed` and the orchestrator has determined the failures are superseded or no longer needed.

**Implementation**: The auth request includes a `target_items: [id1, id2, ...]` field. Tillsyn validates that the requesting principal has orchestrator role. Human reviews the item list and approves. The resulting session can only mutate state on the listed items. TTL is enforced server-side (2–5 minutes).

### D4: Auth Revocation on Terminal State

When a task/level is moved to `done` or `failed`, all auth sessions scoped to that level should be immediately revoked.

- Task moves to `done` → revoke any active auth sessions scoped to that task.
- Phase moves to `done` → revoke any active auth sessions scoped to that phase.
- If an agent dies without moving its task, the orchestrator revokes the auth manually before spawning a replacement.

**Implementation**: This could be a trigger/hook in Tillsyn's state machine, or documented as an orchestrator responsibility. Automatic revocation on state change is cleanest.

**Constraint**: Only **one active auth session per scope level** at a time. Attempting to create a second auth at the same level should fail unless the first is revoked.

### D5: Task Details Are the Prompt (via Auth Claim Enrichment)

Before spawning a subagent, the orchestrator writes concrete instructions into the task:

- `description` — what to do
- `metadata.implementation_notes_agent` — specific agent guidance, commands, files
- `metadata.command_snippets` — exact commands to run
- `metadata.acceptance_criteria` — what success looks like
- `metadata.validation_plan` — how to verify

With D7 (auth claim response enrichment), the subagent receives these details **in the auth claim response** — no separate `plan_item(operation=get)` call needed. The subagent prompt is just auth credentials + task ID. The claim response is the bootstrap.

**Template/instruction change**: Update `get_instructions` to teach agents this pattern. Document the metadata fields that carry instructions. Note that `plan_item(operation=get)` is still available for re-reads during execution, but the initial load comes from the claim response.

### D6: Outcome in Metadata

Standardize where agents put their results:

- `metadata.outcome` — `"success"`, `"failure"`, `"blocked"`, or `"superseded"` (the flag)
- `metadata.blocked_reason` — when outcome is `"blocked"`, describes the specific issue
- `completion_contract.completion_notes` — result summary (human-readable)
- `metadata.transition_notes` — any context for why the state changed

Outcome values:
- `"success"` — work completed successfully (task moves to `done`)
- `"failure"` — work attempted but didn't succeed (task moves to `failed`)
- `"blocked"` — couldn't start or continue due to external issue (task moves to `failed`, subagent signals UP to orchestrator)
- `"superseded"` — orchestrator determined the failure is no longer relevant (task moves from `failed` to `done` via override auth)

The orchestrator reads `plan_item(operation=get)` and has everything in one call. Comments are used for thread history but the essential result is in metadata.

### D7: Auth Claim Response Enrichment

When an auth session is claimed, the response body includes **contextual information for the scope level** so the agent doesn't need separate `plan_item(get)` and `plan_item(list)` calls at bootstrap. The response varies by role and scope:

**Response format conventions:**
- **Children** are returned as a key-value map: `{name: id, name: id, ...}` — not a list. This keeps the payload compact and directly addressable.
- **Comments** on the scoped plan item are included if any exist, so agents have full thread context without a separate `comment(operation=list)` call.

**Global-scoped orchestrator** (template/admin work):
- Bootstrap information (project template bindings, system config)
- Key-value map of project names → IDs

**Project-scoped (or below) orchestrator** (branch, phase, or project scope):
- Full `plan_item` details for the scoped level (description, metadata, completion_contract, etc.)
- Immediate children as `{name: id}` map with current lifecycle state
- Comments on the scoped plan item
- This gives the orchestrator everything it needs to begin routing work — including knowing to set up `/loop` polling immediately.

**Builder/research agents** (typically task-scoped):
- Full `plan_item` details for their task (the task details ARE the prompt)
- Immediate children as `{name: id}` map with current lifecycle state
- Comments on the task
- Scoped to only what their auth level allows.

**QA agents** (qa-check, visual-qa, a11y-check, qa-proof, qa-falsification):
- Full `plan_item` details for their QA subtask
- Comments on their subtask
- **Parent build-task details** including `metadata.affected_artifacts` (D10) + parent comments — so QA knows exactly which code to verify

**Design rationale**: One claim call = fully bootstrapped agent. QA agents get the affected code locations from their parent build-task automatically. The scope-based filtering is enforced by the auth model and template role routing.

**Cross-item context is the orchestrator's job, not the claim response's.** Sibling build-task details are NOT included in the QA claim — that would muddy context and couple QA agents to the shape of the entire phase. Instead, when the orchestrator creates or updates a plan_item that is a dependency or blocker of another item, the orchestrator checks the related items' comments and state, then updates the affected item's details and metadata so the agent starts correctly when spawned. This keeps routing intelligence in the orchestrator and agent context clean.

**Relationship to `capture_state`**: Auth claim enrichment provides scope-level context (one plan item + its immediate children). `capture_state` still serves a different purpose — it gives the broader project view so an orchestrator can pick which scope to focus on. An orchestrator managing a project with 5 phases needs `capture_state` to see all phases before deciding which one to work on. The claim response only has what's at the claimed scope level. Do not remove or replace `capture_state`.

### D8: Level-Based Signaling

Agents communicate upward via a level-based signaling mechanism. The rules vary by role:

**Non-orchestrator agents** (builder, qa, research):
- Can signal **UP one level** to their nearest orchestrator only.
- Use case: permission issues (client blocking a tool call), unexpected blockers discovered before or during work.
- **After signaling UP, the subagent fails the task and dies.** It does NOT wait for a response. The subagent sets `metadata.outcome: "blocked"`, `metadata.blocked_reason` with the issue details, moves the task to `failed`, posts a comment explaining the blocker, and returns. The orchestrator receives the signal via attention items, resolves the issue, revokes the dead agent's auth, and creates a new task. The `failed` task blocks its parent's `done` state (D9) until the orchestrator resolves it.
- Cannot signal down or sideways. All other information flows through plan items.

**Orchestrators**:
- Can signal **UP** (escalate to parent orchestrator) or **DOWN** (direct to child orchestrator).
- Can only signal to other orchestrators. Communication to builders/QA/research is exclusively through plan items (task details, metadata, comments).
- Receiver polls via `/loop` at 60–120s intervals.

**All downward communication to non-orchestrator agents is through plan items.** The orchestrator writes task details, metadata, and comments BEFORE spawning the agent. There is no runtime push-notification to a running subagent — if the orchestrator needs to change course, it waits for the subagent to finish (or fail) and spawns a new one with updated context.

**No agent ID required**: Signals are structural, based on hierarchy position. "Up" resolves to the nearest orchestrator in the parent chain. "Down" resolves to the child orchestrator at the specified scope.

**Implementation**: Either a new `till.signal(direction=up|down, scope_id=..., message=...)` tool, or a mode on `till.handoff(direction=up|down)`. Signals arrive as attention items for the receiving orchestrator. Non-orchestrator "up" signals also arrive as orchestrator attention items.

**Known limitation**: Claude Code sessions can't hold connections open indefinitely. `/loop` with 60–120s intervals for active work is the best approximation.

### D10: Affected Artifacts Tracking

Builders and planners track which code files, symbols, or blocks they affect (or plan to affect). This information flows automatically to QA agents via auth claim enrichment (D7) so they search the exact right things.

**Field**: `metadata.affected_artifacts` on plan items that involve code changes.

**Structure**:
```json
[
  {
    "path": "src/components/Button.tsx",
    "symbols": ["Button", "ButtonProps"],
    "lines": "42-67",
    "change_type": "create",
    "description": "New button component with hover states"
  }
]
```

Fields:
- `path` (required) — file path relative to project root
- `symbols` (optional) — function names, class names, exports affected
- `lines` (optional) — line range, useful for reviews
- `change_type` (required) — `create`, `modify`, `delete`, or `planned` (for plan-phase items that haven't been built yet)
- `description` (optional) — human-readable note on what changed or will change

**Who writes it:**
- **Planners** set `affected_artifacts` with `change_type: "planned"` during planning — describes what WILL be affected.
- **Builders** update `affected_artifacts` with actual `create`/`modify`/`delete` entries during/after implementation — describes what WAS affected.
- **Orchestrators** may seed `affected_artifacts` when creating tasks from plan context.

**Who reads it:**
- **QA agents** receive the parent build-task's `affected_artifacts` automatically in their auth claim response (D7). This tells them exactly which files and symbols to verify.
- **Orchestrators** read `affected_artifacts` when reviewing outcomes, planning next steps, and updating dependent items' context.

**Orchestrator cross-item responsibility**: When a plan_item that is a blocker or dependency of another item is created or completed, the orchestrator checks the related items' comments, state, and `affected_artifacts`, then updates the dependent item's details so the agent for that item starts with correct context. This keeps cross-cutting awareness in the orchestrator (where routing decisions belong) rather than overloading individual agent claim responses with sibling data.

**Template integration**: The `affected_artifacts` field uses the existing `metadata` infrastructure — no new schema type needed. It's a convention enforced by documentation and `get_instructions` guidance, not a hard schema constraint. Templates should document that `build-task` and `plan-phase` task kinds expect this field.

**Why this matters**: Without affected artifacts, QA agents have to guess where to look or ask the orchestrator. With it, the builder's actual changes flow directly to QA through the hierarchy — parent build-task → QA subtask auth claim. Cross-cutting concerns between sibling build-tasks are managed by the orchestrator updating task details before spawning agents, not by dumping sibling data into claim responses.

### D9: `require_children_done` Blocks on `failed` Children at All Levels

Any plan item with `require_children_done: true` **cannot be marked `done`** if any child is in `failed` state. This applies universally — tasks, phases, branches, project — not just phases.

The orchestrator's options when children have failed:
1. **Create a fix task** to address the failure, get it to `done`, then close the parent.
2. **Supersede the failure** via item-list-based short-TTL override auth (D3): request human-approved auth scoped to the specific failed items, move them to `done` with `metadata.outcome = "superseded"` and `completion_contract.completion_notes` explaining why, then close the parent normally.

`failed` is a terminal state that means "attempted and didn't succeed." It is never silently ignored. Resolution is always explicit and traceable.

---

## What Needs to Change in Tillsyn

### Schema Changes

1. **Add `failed` lifecycle state** to the state enum. Update state machine to allow `in_progress` → `failed` transition. `failed` is terminal (like `done`).
2. **Add `metadata.outcome` field** (or validate that arbitrary metadata keys are supported). Values: `"success"`, `"failure"`, `"blocked"`, `"superseded"`. Also `metadata.blocked_reason` for the `"blocked"` case.
3. **`require_children_done` policy update — all levels**: ANY plan item (not just phases) with `require_children_done: true` can be marked `done` only if all children are `done` (not `failed`). `failed` children block parent completion at every hierarchy level. Resolution requires either a fix task or item-list-based override auth (D3).

### Auth Changes

4. **One auth per level enforcement**: Reject `auth_request(operation=create)` if an active session already exists at the same scope level, unless the existing one is revoked first.
5. **Auto-revocation on terminal state**: When `plan_item(operation=move_state, state=done|failed)` succeeds, automatically revoke all auth sessions scoped to that task/level. (Or document this as orchestrator responsibility if auto-revocation is too complex.)
6. **Item-list-based short-TTL override auth** (D3): Support `auth_request(operation=create)` with a `target_items: [id1, id2, ...]` field, short TTL (2–5 minutes), requiring human approval. The resulting session can only mutate the listed items. Used for: closing role-gated tasks, superseding failed items.
7. **Auth claim response enrichment** (D7): When `auth_request(operation=claim)` succeeds, the response body includes scope-appropriate contextual data:
   - All scopes: plan_item details + immediate children as `{name: id}` map + comments on the item
   - Global orchestrator: bootstrap info + project `{name: id}` map
   - Project/below orchestrator: plan_item details for scope + children map with states
   - Builder/research agents: task details + children map
   - **QA agents**: own subtask details + **parent build-task details including `affected_artifacts`** + parent comments (D10). No sibling data — cross-item context is orchestrator-managed via plan_item updates.
   - This eliminates separate plan_item(get), plan_item(list), and comment(list) calls at agent bootstrap.

### Communication Changes

8. **Level-based signaling** (D8): New tool or mode for upward/downward communication. Non-orchestrator agents can signal UP one level to nearest orchestrator (e.g. permission issues, blockers). Orchestrators can signal UP or DOWN but only to other orchestrators. All downward communication to non-orchestrator agents is through plan items only. Signals arrive as attention items.

### Tool/Instruction Changes

9. **Update `get_instructions`** to return operational usage guidance for agents:
   - "Your auth claim response contains your task details, children, and comments — read those first."
   - "Builders/planners: update `metadata.affected_artifacts` with the files/symbols you change or plan to change."
   - "Update `metadata.outcome` with success, failure, or superseded."
   - "Move task to `done` or `failed` when finished."
   - "Post a comment with your result summary."
   - "Do NOT poll attention items (you are ephemeral), but you CAN signal UP to your orchestrator if you hit a blocker before starting work."
   - "QA agents: your claim response includes parent build-task `affected_artifacts` — use these to search the exact right code."
   - "Orchestrators: set up `/loop` polling for attention items immediately after claiming auth."
   - "Orchestrators: when creating/completing a plan_item that is a dep/blocker of another item, check related items and update the dependent item's details so the agent starts correctly."
10. **Update template `completable_by`** on all task kinds to include orchestrator (with item-list-based short-TTL auth as the guard).
11. **Update AGENTS.md and README** with:
   - Orchestrator-as-hub pattern
   - Ephemeral subagent lifecycle
   - Task-details-as-prompt pattern (now simplified: auth claim response includes the details)
   - Auth delegation and cleanup
   - Bootstrap sequence (simplified by auth claim enrichment)
   - Failure handling (failed state + fix task or supersede via override auth)
   - Level-based orchestrator communication
   - `/loop` polling pattern for orchestrators

### Bootstrap Sequence Documentation

12. **Document the standard bootstrap flow** (in AGENTS.md or README):
   ```
   Orchestrator Bootstrap:
   1. Read project CLAUDE.md for project-specific rules
   2. till.get_instructions(focus=project, project_id=...) for template rules
   3. till.auth_request(operation=create) for orchestrator auth (project-scoped)
   4. Human approves
   5. till.auth_request(operation=claim) → response includes:
      - plan_item details for scope level
      - immediate children (names, IDs, states)
      - everything needed to begin routing work
   6. till.capability_lease(operation=issue) for orchestrator lease
   7. Set up /loop polling for attention_items (human approvals, inter-orchestrator signals)
   8. Begin routing: move phases to in_progress, update task details, spawn subagents

   Builder/Research Subagent Bootstrap:
   1. Receive auth credentials + task ID in prompt
   2. till.auth_request(operation=claim) → response includes:
      - task details (description, metadata, acceptance criteria)
      - immediate children as {name: id} map
      - comments on the task
   3. If blocked (e.g. client permission issue): signal UP to orchestrator → set outcome "blocked" → fail task → post comment → die
   4. Execute work based on task details from claim response
   5. Update metadata.affected_artifacts with files/symbols changed
   6. Update metadata.outcome with success or failure
   7. Move task to done or failed
   8. Post comment with result summary
   9. Return (die)

   QA Subagent Bootstrap:
   1. Receive auth credentials + subtask ID in prompt
   2. till.auth_request(operation=claim) → response includes:
      - own QA subtask details + comments
      - parent build-task details including metadata.affected_artifacts + comments
   3. If blocked: signal UP to orchestrator → set outcome "blocked" → fail subtask → post comment → die
   4. Use parent's affected_artifacts to search the exact right files/symbols
   5. Execute QA verification
   6. Update metadata.outcome with success or failure
   7. Move subtask to done or failed
   8. Post comment with QA verdict + evidence
   9. Return (die)
   ```


## Stack Rules

{Project-specific technology rules}

## Build Verification

{Project-specific build verification checklist}

## Commit Format

Conventional commits: `type(scope): message`. All lowercase except proper nouns/acronyms. No co-authored-by trailers. No period at the end.

Types: `feat`, `fix`, `refactor`, `chore`, `docs`, `test`, `ci`, `style`, `perf`
```

---

## Resolved Questions

These were open questions during design and have been decided:

### RQ1: Inter-Agent Communication → D8 (Level-Based Signaling)

**Decision**: Level-based signaling with role-differentiated access:
- Non-orchestrator agents can signal UP one level to nearest orchestrator (for blockers, permission issues)
- Orchestrators can signal UP or DOWN, but only to other orchestrators
- All downward communication to non-orchestrator agents is through plan items only (task details, metadata, comments)
- Signals arrive as attention items. Orchestrator polls via `/loop` at 60–120s intervals.

**Known limitation**: Claude Code can't hold connections open indefinitely. `/loop` is the approximation.

### RQ2: `depends_on` vs `blocked_by` → Keep Both

**Decision**: Keep both. Same enforcement mechanism (block `in_progress` until resolved), different semantics:
- `depends_on` = planned prerequisite, set at creation, structural
- `blocked_by` + `blocked_reason` = dynamic runtime blocker, set when problems occur, cleared when resolved

The distinction matters for orchestrator decision-making: planned dependency vs. unexpected problem.

### RQ4: Polling Viability → `/loop` + Auth Claim Enrichment

**Decision**: `/loop` at 60–120s intervals for active orchestrators. Auth claim response enrichment (D7) means the orchestrator knows its scope and children immediately on claim, so it can set up polling right away. `wait_timeout` on `attention_item(operation=list)` should be supported to reduce polling frequency.

### RQ5: `failed` State and Completion → D9 (Blocks at All Levels)

**Decision**: `require_children_done` blocks on `failed` children at ALL plan item levels, not just phases. Resolution: create a fix task, or supersede via item-list-based short-TTL override auth (D3) with human approval. `failed` items moved to `done` get `metadata.outcome = "superseded"`.

---

## Verification Gaps

These are not open design questions — the design is decided. But the implementation may have gaps that the Tillsyn agent should verify during planning:

### VG1: Template `child_rules` on Runtime Task Creation

**Expected behavior**: When the orchestrator creates a new `build-task` via `plan_item(operation=create)` to replace a failed one, template `child_rules` should fire and auto-generate the 5 QA subtasks (proof, falsification, visual-qa, a11y-check, commit-and-reingest).

**Risk**: `child_rules` might only fire during initial project population (when the project is first created from the template), not on subsequent runtime `plan_item(operation=create)` calls.

**Action for Tillsyn agent**: Verify this works during planning. If `child_rules` don't fire on runtime task creation, this is a bug that must be fixed — it's load-bearing for the failure recovery pattern (D1 + failure handling). Flag this to the human during planning if behavior is uncertain.

---

## Summary of Changes

| # | Area | Change | Priority |
|---|---|---|---|
| 1 | Schema | Add `failed` lifecycle state | High |
| 2 | Schema | Add/support `metadata.outcome` field (success, failure, superseded) | High |
| 3 | Schema | `require_children_done` blocks on `failed` at ALL levels | High |
| 4 | Auth | One active auth per scope level | High |
| 5 | Auth | Revoke on terminal state (done/failed) | High |
| 6 | Auth | Item-list-based short-TTL override auth (D3) | High |
| 7 | Auth | Auth claim response enrichment: details + children map + comments (D7) | High |
| 8 | Auth | QA claim includes parent build-task `affected_artifacts` (D7+D10) | High |
| 9 | Communication | Level-based signaling: subagents UP to orchestrator, orchestrators UP/DOWN to orchestrators (D8) | High |
| 9a | Convention | Cross-item context is orchestrator-managed via plan_item updates, not claim response | High |
| 10 | Convention | `metadata.affected_artifacts` field on build-tasks and plan tasks (D10) | High |
| 11 | Tool | Update `get_instructions` operational guidance (incl. affected_artifacts) | High |
| 12 | Templates | Add orchestrator to `completable_by` with override guard | High |
| 13 | Docs | Bootstrap sequence (simplified by auth claim enrichment) | High |
| 14 | Docs | Ephemeral subagent pattern | High |
| 15 | Docs | Task-details-as-prompt (now: auth claim includes details + comments) | High |
| 16 | Docs | Level-based orchestrator communication + `/loop` polling | Medium |
| 17 | Docs | Failure handling (failed state + fix task or supersede) | High |
| 18 | Docs | Affected artifacts tracking pattern (builders write, QA reads) | High |
| 19 | Verify | `child_rules` fire on runtime `plan_item(create)` (VG1) | High |
