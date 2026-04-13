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

### Problem 7: Orchestrator Has No Auth Approval Loop

When the orchestrator creates an auth request (e.g. for override auth, subagent delegation), it has no mechanism to wait for the human to approve it. The orchestrator creates the request and then… what? Without a `/loop` polling for auth request state changes, the orchestrator can't know when the human approved the request in the TUI. The session stalls.

**Impact**: Auth-gated workflows break in practice. The orchestrator can't proceed after requesting override auth because it doesn't know when approval arrives. This blocks the entire D3 override auth pattern and any auth request flow.

**Root cause**: No documented or enforced requirement that orchestrators enter a `/loop` when waiting for auth approval. The auth request → approval → claim flow assumes the orchestrator is polling, but nothing ensures it.

### Problem 8: Human Can't Move Tasks via MCP Either

The `completable_by` role gate (`ensureTaskCompletableByNodeContract`) doesn't just block orchestrators — it blocks the human user too when operating via MCP. If a task's `NodeContractSnapshot` says `completable_by: [research]`, the human user session (actor type `human`) is also rejected. The human can only move tasks via the TUI (which bypasses the MCP adapter layer), creating an inconsistency.

**Impact**: The dev can't use `plan_item(operation=move_state, state=done)` via MCP to close tasks that are role-gated. Combined with Problem 3, both the orchestrator AND the human are blocked from completing role-gated tasks via MCP. The override auth mechanism (D3) must serve both principals, not just the orchestrator.

### Problem 9: No TUI Surface for Failed Tasks

When `failed` state is added (D1), there's no TUI surface for viewing or managing failed tasks. Failed tasks shouldn't appear in the board columns like `todo`/`in_progress`/`done` — they need their own notification-style surface, similar to how archived tasks are handled but with higher visibility since failed tasks are action items, not archived history.

**Impact**: Without a TUI surface, the human can't discover or triage failed tasks. The orchestrator can query for them via MCP, but the human reviewing work in the TUI would have no way to see what failed.

---

## Design Decisions Made

These decisions are confirmed by the dev and should be implemented:

### D1: Four Lifecycle States

Add `failed` as a fourth terminal state alongside `done`:

- `todo` → `in_progress` → `done` (success)
- `todo` → `in_progress` → `failed` (failure)
- `todo` → `failed` (discovered invalid before work starts — valid transition)

`failed` is a terminal state like `done` — work was attempted or determined unnecessary/invalid. It is NOT the same as archived or cancelled. The task retains its full context (metadata, comments, history) for traceability.

**Valid transitions**:
- `todo → in_progress` (start work)
- `in_progress → done` (success)
- `in_progress → failed` (failure during work)
- `todo → failed` (invalid/unnecessary before work starts)
- `failed → done` (supersede — **requires override auth D3**)
- All other transitions from terminal states (`done`, `failed`) are blocked without override auth.

**SQL/schema change**: Add `failed` to the lifecycle_state enum. Keep full history of state transitions. `failed` tasks should be queryable via `plan_item(operation=search, states=["failed"])`. Add a hidden "failed" column to every project's column list (same pattern as the existing "archived" column). This column is hidden from the TUI board view but exists in the database so that `resolveTaskColumnIDForState` and `lifecycleStateForColumnID` can resolve `failed` ↔ column mappings. Without this hidden column, the MCP `move_state(state=failed)` path fails because column resolution returns an error.

**Transition guard**: `failed` is terminal. Moving a task FROM `failed` to any other state (including `done` for supersede) requires override auth (D3). The domain layer (`SetLifecycleState`) currently has no transition matrix — it accepts any valid state as target. A transition guard must be added, either in the domain layer or in `MoveTask` at the app service layer, to enforce terminal-state semantics. Without this, anyone with edit access can move `failed` back to `todo`, violating the spec.

**Timestamp handling**: `SetLifecycleState` manages `CompletedAt` (for `done`), `ArchivedAt` (for `archived`), and `StartedAt` (for `progress`). For `failed`, reuse `CompletedAt` — failed means work-completed-unsuccessfully, so the completion timestamp is semantically correct. Do NOT add a separate `FailedAt` field. The `metadata.outcome` field (D6) distinguishes success from failure.

**Implementation note for `CompletedAt` reuse**: `SetLifecycleState` currently has two branches: (1) `if prev != StateDone && state == StateDone { t.CompletedAt = &ts }` — sets the timestamp, and (2) `if state != StateDone { t.CompletedAt = nil }` — clears it. **Both branches must be updated atomically.** Branch 1 becomes `(prev != StateDone && state == StateDone) || (prev != StateFailed && state == StateFailed)`. Branch 2 becomes `state != StateDone && state != StateFailed`. If a developer updates one and misses the other, `CompletedAt` will be set and immediately nilled in the same call — and tests that only check `LifecycleState` (not `CompletedAt`) will pass. The TUI task info view should render the timestamp label as "completed_at" for `done` tasks and "failed_at" for `failed` tasks (conditional label, same underlying field).

**Snapshot version**: `Snapshot.Validate()` has an exhaustive switch on lifecycle states. Adding `failed` requires updating this switch and bumping the snapshot version.

**Full change surface** (QA-verified — 13+ functions across 5 packages):
- Domain: `isValidLifecycleState`, `normalizeLifecycleState`, `SetLifecycleState` (timestamp branch)
- App: `normalizeStateID`, `lifecycleStateForColumnID`, `sanitizeStateTemplates`, `Snapshot.Validate()`
- Adapters: `normalizeTaskStateInput`, `normalizeStateLikeID`, `taskLifecycleStateForColumnName`, `canonicalLifecycleState`, `buildWorkOverview` (needs new `WorkOverview.FailedTasks` counter — the current `default` branch silently counts unrecognized states as `TodoTasks`)
- TUI: `lifecycleStateForColumnName`, `lifecycleStateLabel`, `normalizeColumnStateID`, `dependencyStateIDForTask`, plus 15+ sites with `== domain.StateDone` checks
- Config: `Config.Validate` checks search states against known lifecycle states

**TUI rendering of `failed`**: Failed tasks do NOT appear as a board column (the "failed" column is hidden, like "archived"). Instead, they appear in the **notifications panel** — both project-scoped and propagated to global notifications using existing notification infrastructure. The notification panel is `Model.renderOverviewPanel` (project) and `Model.renderGlobalNoticesPanel` (global), with sections built by `Model.noticesPanelSections`. Enter-key transitions on global items go through `Model.beginGlobalNoticeTransition`. In the project notifications panel, failed tasks appear **above warnings** as a new section with a count badge (e.g. "3 Failed Tasks"). Selecting the failed-tasks notification entry opens a list screen using the **same component and rendering pattern that coordination uses** (handoffs/attention items list via `Model.renderCoordinationDetailModeView`). Each failed task in the list can be opened with Enter to see the full task info view. This reuses the existing TUI component infrastructure — a new notification section wired into `noticesPanelSections`, no new screen types needed.

**Naming note**: The domain constant is `StateProgress` with value `"progress"` (not `"in_progress"`). The MCP adapter `normalizeTaskStateInput` normalizes `"in_progress"` → `"progress"`. The same normalization pattern applies to `"failed"` — the domain constant will be `StateFailed` with value `"failed"`.

### D2: Task State as Signal (Not @Mentions)

Subagents are ephemeral. They don't poll attention items. The coordination model changes:

- **Subagents** read task details at spawn, do work, update task metadata with outcome, move task to `done`/`failed`, post a comment, die.
- **Orchestrator** reads task state + metadata after subagent returns. No @mention polling needed for subagent results.
- **Attention items** are used ONLY by orchestrators (for human approvals and inter-orchestrator communication).

**Template/instruction change**: Remove any guidance telling subagents to check attention items. Update role model docs to reflect ephemeral subagent pattern.

### D3: Item-List-Based Short-TTL Override Auth

Override auth serves two principal types through two different paths:

**Human user in TUI**: The TUI shows a **warning modal** when the human tries to complete a role-gated task or a task with failed children. The modal explains the override and lets the human proceed directly. No auth request needed — the TUI bypasses the MCP adapter layer's `ensureTaskCompletableByNodeContract` guard. The human is always the ultimate authority.

**Orchestrator (or human) via MCP**: When the orchestrator (or the human user operating via MCP) needs to close role-gated items or mark items `done` when children include `failed` tasks, it requests a **short-TTL, item-list-scoped auth** approved by the human in the TUI.

Key design:

- **Item-list-based**: The auth request specifies the **full list of item IDs** it applies to. The auth is scoped to exactly those items — not a blanket role escalation.
- **Short TTL (~3 minutes)**: The auth expires automatically. The requester can't hold it open.
- **Human-approved**: Every override auth requires human approval in the TUI. No auto-approval.
- **Applies to ALL plan items**: Any plan item with `require_children_done` can't be marked `done` when children include `failed` items without this auth. This is not phase-specific — it applies at every level of the hierarchy.
- **Both orchestrator and human can create override auth requests**: The `auth_request(operation=create)` validates that the requesting principal is `orchestrator` OR `human`. Both actor types can request override auth via MCP. The human in TUI has the separate warning-modal path that doesn't require auth requests.

Use cases:
1. **Orchestrator via MCP**: Closing a role-gated task when the work was done by a different role (e.g. orchestrator closing a "completable by: research" task after research subagent returned). Orchestrator requests override auth → human approves in TUI → orchestrator claims and mutates.
2. **Orchestrator via MCP**: Marking a parent `done` when some children are `failed` and the orchestrator has determined the failures are superseded or no longer needed.
3. **Human via MCP**: Closing tasks that are role-gated when operating outside the TUI — the dev can use `plan_item(operation=move_state, state=done)` after obtaining override auth.
4. **Human in TUI**: Warning modal lets the human proceed directly. No auth request flow needed.

**Implementation**: The auth request includes a `target_items: [id1, id2, ...]` field. Tillsyn validates that the requesting principal has `orchestrator` or `human` actor type. The human reviews the item list in the TUI and approves. The resulting session can only mutate state on the listed items. TTL is enforced server-side (~3 minutes).

**TUI override auth approval**: The approval screen renders the `target_items` as a list of task titles + current states alongside the request reason, reusing existing auth request approval components and styling. The warning modal for direct human override (TUI-only bypass path) also reuses existing modal patterns and styling for consistency. No new screen types — wire into existing component infrastructure.

**Completion fix**: The `ensureTaskCompletableByNodeContract` check in `Service.MoveTask` must be updated to check for active override auth on the target item. If override auth exists for this item ID and hasn't expired, the `completable_by` guard is bypassed. The override check goes **inside** `ensureTaskCompletableByNodeContract` as an "override auth bypasses this guard" branch — the guard already knows it's blocking, so it's the natural place to check.

**`target_items` enforcement**: The override auth session carries its `target_items` directly (copied from the auth request at session creation time, not looked up via join). This enables single-lookup guard checks without joining through the auth_request table. The session is short-lived (~3 min TTL) so duplication is minimal.

**`target_items` validation on create**: `auth_request(operation=create)` validates only that every item ID in `target_items` exists and belongs to the requesting session's project scope. It does NOT validate that items are in a state that needs override — items may change state between request creation and human approval.

**`MoveTask` capability action for `failed`**: Moving a task TO `failed` uses `CapabilityActionMarkFailed` (new capability action, parallel to `CapabilityActionMarkComplete`). Moving a task FROM `failed` to `done` (supersede) uses `CapabilityActionMarkComplete` + override auth check. The `MoveTask` switch becomes:
```
case toState == StateDone:  CapabilityActionMarkComplete
case toState == StateFailed: CapabilityActionMarkFailed
case fromState == StateTodo && toState == StateProgress: CapabilityActionMarkInProgress
default: CapabilityActionEditNode
```

**`CapabilityActionMarkFailed` role assignment**: ALL roles get `CapabilityActionMarkFailed` in `DefaultCapabilityActions` — orchestrator, builder, QA, and research. Every role must be able to fail its own tasks (the ephemeral subagent pattern requires subagents to mark their own tasks as `failed` and die). This matches the existing pattern where all 4 roles already have `CapabilityActionMarkComplete` and `CapabilityActionMarkInProgress`. The new constant must also be added to `validCapabilityActions` and `IsValidCapabilityAction`.

### D3a: Orchestrator Auth Approval Loop

**ALL orchestrator auth requests MUST be paired with a `/loop` polling pattern.** When the orchestrator creates an auth request that requires human approval, it MUST immediately enter a `/loop` at 60–120s intervals polling for the auth request's status change. Without this, the orchestrator creates the request and has no way to know when the human approved it.

The pattern:
1. Orchestrator calls `till.auth_request(operation=create, ...)` — gets back a pending auth request ID.
2. Orchestrator enters `/loop` polling `till.auth_request(operation=get, id=...)` at 60–120s intervals.
3. Human approves (or denies) the auth request in the TUI.
4. On next `/loop` tick, orchestrator reads the auth request status:
   - `status: approved` → proceed to `till.auth_request(operation=claim, ...)`, exit loop.
   - `status: denied` → read `denial_reason` from the response, log/report the denial, exit loop. The denial reason (already supported by Tillsyn's existing denial reason logic) tells the orchestrator why and what to do next. The system response includes a signal to kill the loop.
   - `status: expired` → the request timed out without human action, exit loop, handle as a soft failure.
5. `/loop` exits on ANY terminal status (approved, denied, expired). The orchestrator handles each case.

This is not optional. Every auth request flow — override auth (D3), subagent delegation auth, phase-level auth — requires this pattern. The `get_instructions` guidance (D5) must teach this explicitly. The bootstrap sequence documentation must include it.

**Known limitation**: Claude Code `/loop` has a minimum interval of 60s. For auth approvals that happen quickly (human is watching), this means up to 60s latency between approval and orchestrator notice. This is acceptable — the alternative (no loop) is permanent stall.

### D4: Auth Revocation on Terminal State

When a task/level is moved to `done` or `failed`, all auth sessions scoped to that level should be immediately revoked.

- Task moves to `done` → revoke any active auth sessions scoped to that task.
- Phase moves to `done` → revoke any active auth sessions scoped to that phase.
- If an agent dies without moving its task, the orchestrator revokes the auth manually before spawning a replacement.

**Implementation**: Automatic revocation via a state-machine hook in `Service.MoveTask`. When `MoveTask` successfully transitions a task to `done` or `failed`, it revokes all auth sessions scoped to that task/level as part of the same operation. This is cleaner and more reliable than requiring every orchestrator to remember to revoke manually.

**Concurrency note**: SQLite write serialization (single-process, `IMMEDIATE` transactions) means concurrent `move_state` calls are serialized at the database level. The first write wins; the second sees the updated state and either succeeds (if the transition is still valid) or fails (if the state already changed). This is sufficient for single-process Tillsyn. If Tillsyn ever supports multi-process access to the same database, explicit optimistic concurrency control (e.g. version column check on `UpdateTask`) would be needed.

**Constraint**: Only **one active auth session per scope level** at a time. Attempting to create a second auth at the same level should fail unless the first is revoked.

**D4/D3 dependency**: D4 MUST be implemented AFTER D3 (override auth) within Wave 1. Revocation on both `done` and `failed` is safe once override auth exists, because the supersede path (`failed → done`) goes through override auth which creates a fresh session. Implementation order within Wave 1: D1 → D6 → D3 → D3a → D4 (revoke on both `done` and `failed`) → D9. This eliminates the auth leak problem where failed tasks leave orphaned sessions blocking the "one auth per scope level" constraint.

### D5: Task Details Are the Prompt (via Auth Claim Enrichment)

Before spawning a subagent, the orchestrator writes concrete instructions into the task:

- `description` — what to do
- `metadata.implementation_notes_agent` — specific agent guidance, commands, files
- `metadata.command_snippets` — exact commands to run
- `metadata.acceptance_criteria` — what success looks like
- `metadata.validation_plan` — how to verify

With D7 (auth claim response enrichment, **Wave 2**), the subagent receives these details **in the auth claim response** — no separate `plan_item(operation=get)` call needed. The subagent prompt is just auth credentials + task ID. The claim response is the bootstrap.

**Without D7 (Wave 1)**: The "task details are the prompt" pattern still works — the orchestrator writes everything into the task, and the subagent calls `plan_item(operation=get)` at spawn to read it. D7 optimizes this from two calls (claim + get) to one (claim includes details). The pattern is the same; D7 is the ergonomic improvement, not the prerequisite.

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

**Validation**: Outcome values are validated at the **MCP adapter layer** (`normalizeTaskStateInput` or a sibling function in `common/app_service_adapter_mcp.go`). The adapter rejects any `move_state` or `update` that sets an unrecognized `metadata.outcome` value before it reaches the service layer. This is intentionally a boundary constraint — the domain metadata remains freeform, but external input is validated at the system boundary. The valid set (`success | failure | blocked | superseded`) is small and stable; if it needs to grow, the adapter validation is the single place to update.

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

**Soft enforcement**: The MCP adapter layer should emit a **warning** (not a block) when a `build-task` is moved to `done` with an empty `affected_artifacts` field. This makes missing data visible without breaking the workflow. The warning appears in the `move_state` response and in the TUI's project notifications (Warnings section). Without this, QA agents silently receive empty artifact lists and have to guess where to look.

**Why this matters**: Without affected artifacts, QA agents have to guess where to look or ask the orchestrator. With it, the builder's actual changes flow directly to QA through the hierarchy — parent build-task → QA subtask auth claim. Cross-cutting concerns between sibling build-tasks are managed by the orchestrator updating task details before spawning agents, not by dumping sibling data into claim responses.

### D9: `require_children_done` Blocks on `failed` Children at All Levels

Any plan item with `require_children_done: true` **cannot be marked `done`** if any child is in `failed` state. This applies universally — tasks, phases, branches, project — not just phases.

The orchestrator's options when children have failed:
1. **Create a fix task** to address the failure, get it to `done`, then close the parent.
2. **Supersede the failure** via item-list-based short-TTL override auth (D3): request human-approved auth scoped to the specific failed items, move them to `done` with `metadata.outcome = "superseded"` and `completion_contract.completion_notes` explaining why, then close the parent normally.

`failed` is a terminal state that means "attempted and didn't succeed." It is never silently ignored. Resolution is always explicit and traceable.

**Error message differentiation**: `CompletionCriteriaUnmet` currently produces "child item %q is not done" for any non-done child. This is confusing for `failed` children — the message should distinguish `failed` from `todo`/`in_progress`. Update to: "child item %q is failed" vs "child item %q is not done (currently %s)". The `ensureTaskCompletionBlockersClear` check (app layer, via `NodeContractSnapshot.RequiredForParentDone`) also produces a similar generic message — both layers need differentiation.

**MoveTask guard for `failed → done`**: The `MoveTask` method in `Service` currently only has special handling for `toState == StateDone` (completion checks) and `todo → progress` (start criteria). It needs an additional guard: if the task is currently `failed` and the target is `done`, require override auth (D3). Without this, the transition is unrestricted once `failed` is a valid state.

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
   - **"Orchestrators: when creating ANY auth request, immediately enter a `/loop` polling for auth request status at 60–120s intervals. Do NOT proceed without the loop — the session will stall waiting for human approval."** (D3a)
10. **Update template `completable_by`** on all task kinds to include orchestrator (with item-list-based short-TTL auth as the guard).
11. **Update AGENTS.md and README** with:
   - Orchestrator-as-hub pattern
   - Ephemeral subagent lifecycle
   - Task-details-as-prompt pattern (now simplified: auth claim response includes the details)
   - Auth delegation and cleanup
   - Bootstrap sequence (simplified by auth claim enrichment + mandatory auth approval loop)
   - Failure handling (failed state + fix task or supersede via override auth)
   - Level-based orchestrator communication
   - `/loop` polling pattern for orchestrators (both attention items AND auth approvals)

### Bootstrap Sequence Documentation

12. **Document the standard bootstrap flow** (in AGENTS.md or README):
   ```
   Orchestrator Bootstrap:
   1. Read project CLAUDE.md for project-specific rules
   2. till.get_instructions(focus=project, project_id=...) for template rules
   3. till.auth_request(operation=create) for orchestrator auth (project-scoped)
   4. IMMEDIATELY enter /loop polling till.auth_request(operation=get, id=...) at 60-120s
      - DO NOT proceed without the loop — the session will stall
   5. Human approves (orchestrator gets notice on next /loop tick)
   6. till.auth_request(operation=claim) → response includes:
      - plan_item details for scope level
      - immediate children (names, IDs, states)
      - everything needed to begin routing work
   7. till.capability_lease(operation=issue) for orchestrator lease
   8. Set up /loop polling for attention_items (human approvals, inter-orchestrator signals)
   9. Begin routing: move phases to in_progress, update task details, spawn subagents

   Orchestrator Override Auth Flow (D3):
   1. Orchestrator identifies items needing override (role-gated tasks, failed → done)
   2. till.auth_request(operation=create, target_items=[id1, id2, ...], ttl=300)
   3. IMMEDIATELY enter /loop polling till.auth_request(operation=get, id=...) at 60-120s
   4. Human reviews item list in TUI, approves
   5. Orchestrator sees approved status on next /loop tick
   6. till.auth_request(operation=claim) → gets short-TTL override session
   7. Mutate the listed items (move_state, update) within TTL window
   8. Session auto-expires after TTL

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

### VG1: Template `child_rules` on Runtime Task Creation — RESOLVED (Not a Gap)

**Status**: CLOSED — verified via Hylla code analysis (QA proof + QA falsification, 2026-04-12).

**Finding**: `child_rules` DO fire on runtime `plan_item(operation=create)`. The `Service.createTaskWithTemplates` method (lines 696-705 of `internal/app/service.go`) calls `applyTemplateChildRules` for every task creation with a bound NodeTemplate. The function the original analysis flagged — `applyProjectTemplateChildRules` — is a **separate** function that only fires during `CreateProjectWithMetadata` (initial project population). But `applyTemplateChildRules` (without the "Project" prefix) fires on every runtime create via `createTaskWithTemplates`.

**Conclusion**: The failure recovery pattern (create replacement `build-task`, auto-generate QA subtasks) works correctly. No fix needed.

---

## QA Findings

These findings were produced by independent QA proof and QA falsification reviews across 3 rounds (2026-04-12/13). They are incorporated into the design decisions above but documented here for traceability.

### QAF1: D1 Blast Radius Underestimated

The original plan identified ~5 normalizers to update for `failed` state. Both QA agents independently found **13+ functions across 5 packages** that hardcode state awareness. The full change surface is documented in D1 above. Key additions: `isValidLifecycleState`, `SetLifecycleState` (timestamp branch), `Snapshot.Validate()`, `Config.Validate`, and 15+ TUI sites with `== StateDone` checks.

### QAF2: No Transition Matrix in Domain Layer

`SetLifecycleState` accepts any valid state as target — no transition guards. Adding `failed` as valid means `failed → todo` is unrestricted in the domain. Terminal-state enforcement must be added. Addressed in D1 "Transition guard" section above.

### QAF3: D4/D3 Hidden Coupling — RESOLVED

Implementing D4 (auth revocation on `failed`) before D3 (override auth) would create a deadlock. Resolved by enforcing internal Wave 1 ordering: D3 ships before D4. D4 now revokes on BOTH `done` and `failed` within Wave 1 since D3 provides the override auth path for the `failed → done` supersede transition.

### QAF4: D9 Error Message Confusion

`CompletionCriteriaUnmet` produces a generic "child item X is not done" for failed children. Both QA agents flagged this as a UX issue. Addressed in D9 "Error message differentiation" section above.

### QAF5: D8 YAGNI Risk

Both QA agents agreed D8 (level-based signaling) survives on correctness but carries YAGNI risk. The current workaround (subagent fails task, orchestrator reads failure) works via D1+D2. D8 adds value only if the orchestrator needs to act before the subagent finishes failing — which is rare for ephemeral subagents. D8 is safely deferrable to a later wave.

### QAF6: D6 Needs Outcome Validation — RESOLVED

Nothing prevents setting `metadata.outcome: "banana"`. Resolved: validation at MCP adapter layer (added to D6 "Validation" section). Adapter rejects unrecognized outcome values at the system boundary.

### QAF7: No Hidden `failed` Column Breaks MCP `move_state` — RESOLVED (Round 3)

`resolveTaskColumnIDForState` requires a column for every lifecycle state. Without a hidden "failed" column, the MCP `move_state(state=failed)` path dies on column resolution. Resolved: add hidden "failed" column to every project's column list, same pattern as existing "archived" column. Added to D1 "SQL/schema change" section.

### QAF8: `buildWorkOverview` Default Branch Misclassifies `failed` — RESOLVED (Round 3)

The `default` case in `buildWorkOverview` increments `TodoTasks++` for unrecognized states. Resolved: add explicit `failed` case and new `WorkOverview.FailedTasks` counter field. Added to D1 change surface.

### QAF9: `CapabilityActionMarkFailed` Role Assignment Unspecified — RESOLVED (Round 3)

The spec defined the constant but not which roles get it. Resolved: all roles (orchestrator, builder, QA, research) get `CapabilityActionMarkFailed`, matching the existing pattern where all roles have `MarkComplete`. Added to D3.

### QAF10: D4 Auto vs Manual Revocation Ambiguous — RESOLVED (Round 3)

D4 left the choice between state-machine hook and orchestrator responsibility open. Resolved: auto-revocation via state-machine hook in `Service.MoveTask`. SQLite write serialization handles concurrent `move_state` races. Added to D4.

### QAF11: D5/D7 Wave Mismatch — RESOLVED (Round 3)

D5 referenced D7 claim enrichment as if it existed in Wave 1, but D7 is Wave 2. Resolved: added explicit note that D5 without D7 uses `plan_item(get)` — the pattern works either way, D7 is ergonomic improvement not prerequisite. Added to D5.

### QAF12: TUI Notification Panel Exists — Confirmed (Round 3)

QA falsification claimed the notification panel was not a reusable component. This was wrong — confirmed via Hylla: `Model.renderOverviewPanel` (project), `Model.renderGlobalNoticesPanel` (global), `Model.noticesPanelSections` (section builder), `Model.beginGlobalNoticeTransition` (Enter-key handler). The existing infrastructure supports adding new notification categories. Updated D1 TUI section with specific function names.

---

## Summary of Changes

| # | Area | Change | Priority | Wave |
|---|---|---|---|---|
| 1 | Schema | Add `failed` lifecycle state (D1) — 13+ functions, transition guards, timestamps, snapshot version, hidden column | High | 1 |
| 2 | Schema | Add/validate `metadata.outcome` field — success, failure, blocked, superseded (D6). Validate at MCP adapter layer. | High | 1 |
| 3 | Auth | Item-list-based short-TTL override auth — both orchestrator and human can request (D3). `AuthRequest` schema extension with `TargetItems` field. | High | 1 |
| 4 | Auth | Orchestrator auth approval `/loop` with denial/expiry termination (D3a) | High | 1 |
| 5 | Auth | Completion fix — `ensureTaskCompletableByNodeContract` checks override auth, TUI warning modal for human (D3) | High | 1 |
| 6 | Auth | `MoveTask` new `CapabilityActionMarkFailed` (all roles) + `failed → done` override auth guard + `validCapabilityActions` + `DefaultCapabilityActions` | High | 1 |
| 7 | Auth | One active auth per scope level | High | 1 |
| 8 | Auth | Auto-revoke on terminal state via state-machine hook — both `done` and `failed` (D4). Ships after D3 within Wave 1. | High | 1 |
| 9 | Schema | `require_children_done` blocks on `failed` at ALL levels, with error message differentiation (D9) | High | 1 |
| 10 | TUI | Failed tasks in notifications panel — project + global, above warnings, count badge, list-detail view (D1) | High | 1 |
| 11 | TUI | Warning modal for human override of `completable_by` guard (D3) | High | 1 |
| 12 | Auth | Auth claim response enrichment: details + children map + comments (D7) | High | 2 |
| 13 | Auth | QA claim includes parent build-task `affected_artifacts` (D7+D10) | High | 2 |
| 14 | Convention | `metadata.affected_artifacts` field on build-tasks and plan tasks (D10) + soft enforcement warning on empty `affected_artifacts` at `done` | High | 2 |
| 15 | Communication | Level-based signaling — deferred, YAGNI risk (D8) | Medium | 3 |
| 16 | Convention | Cross-item context is orchestrator-managed via plan_item updates, not claim response | High | 1 |
| 17 | Tool | Update `get_instructions` operational guidance (incl. affected_artifacts, auth loop, denial handling) | High | 1 |
| 18 | Templates | Add orchestrator to `completable_by` with override guard | High | 1 |
| 19 | Docs | Bootstrap sequence (incl. mandatory auth loop, override auth flow, denial/expiry handling) | High | 1 |
| 20 | Docs | Ephemeral subagent pattern | High | 1 |
| 21 | Docs | Task-details-as-prompt (now: auth claim includes details + comments) | High | 2 |
| 22 | Docs | Level-based orchestrator communication + `/loop` polling | Medium | 3 |
| 23 | Docs | Failure handling (failed state + fix task or supersede) | High | 1 |
| 24 | Docs | Affected artifacts tracking pattern (builders write, QA reads) | High | 2 |
| ~~25~~ | ~~Verify~~ | ~~`child_rules` fire on runtime `plan_item(create)` (VG1) — RESOLVED, not a gap~~ | ~~Done~~ | ~~—~~ |

**Wave 1 internal dependency order**: D1 → D6 → D3 (incl. `AuthRequest` schema extension) → D3a → D4 → D9 → TUI (notifications + warning modal) → docs/templates/instructions

**Wave summary**:
- **Wave 1** (lifecycle + override auth): D1, D6, D3, D3a, D4 (both terminal states), D9, TUI updates. Internal order above. This fixes the most acute coordination problems — no failed state, can't complete role-gated tasks, orchestrator stalls on auth approval, auth leak for failed tasks.
- **Wave 2** (auth enrichment + artifacts): D5, D7, D10. Requires D3 to be stable first.
- **Wave 3** (signaling — deferrable): D8. YAGNI risk — only build if D1+D2 workaround proves insufficient.
- **D2** (task state as signal): Convention/docs change, can be done alongside any wave.
