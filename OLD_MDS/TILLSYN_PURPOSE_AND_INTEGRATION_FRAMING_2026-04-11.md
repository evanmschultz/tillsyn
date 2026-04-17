# Tillsyn Purpose And Integration Framing

> **Audience:** anyone wiring an LLM coding agent (Codex, Claude Code, Cursor, custom MCP clients) to the Tillsyn MCP server. This document exists because the most common framing mistake — describing Tillsyn as "the system of record for planning" or "a passive ledger" — leads to under-using the runtime and produces brittle agent flows. Read this **before** writing skills, subagents, slash commands, hooks, or instruction files that interact with Tillsyn.

## TL;DR

Tillsyn is a **multi-actor coordination runtime**, not a passive planning ledger. It has:

1. A **role model** with distinct semantics per role.
2. A **template + child_rules engine** that auto-generates required gates as real blockers.
3. **Three coordination surfaces** (`till.comment`, `till.handoff`, `till.attention_item`) with distinct semantics that should not be flattened.
4. **Scoped auth** with delegated child sessions for cross-role mutations.
5. An explicit **restart recovery flow** that lets agents reattach to live state without losing context.

If your skill or subagent design treats Tillsyn as a place to dump status text, you have already lost most of what the runtime gives you.

---

## What Tillsyn Is Not

- **Not** "the system of record for planning." That description is technically true the way "an airplane is a metal tube" is technically true, and just as useless. Planning is one of many things Tillsyn coordinates.
- **Not** a passive append-only log of what happened. The runtime actively routes work and enforces gates.
- **Not** interchangeable with a markdown worklog. Worklogs are unstructured notes for a single author. Tillsyn carries typed state, role ownership, blocking relationships, template-generated subtasks, and addressable inboxes that worklogs cannot represent.
- **Not** a wrapper around `till.get_instructions`. Instructions return *policy context*. They are not a substitute for direct runtime state via `till.attention_item`, `till.handoff`, `till.comment`, `till.plan_item`, `till.kind`, etc.

## What Tillsyn Is

### 1. A role model

Tillsyn distinguishes coordination roles with distinct duties. A correctly designed flow names which role owns each step instead of flattening everything to "the agent":

- **Orchestrator** — plans, routes, delegates, cleans up. Owns the branch, the phase structure, and routing decisions. Does not normally implement code itself.
- **Builder** (`@builder`, aliased to `@dev`) — implements work inside a build-task. Hands back to orchestrator or qa when the task's definition of done is met.
- **QA** — verifies and either closes or returns work. QA is **two distinct asymmetric passes**:
  - `QA PROOF REVIEW` — verify evidence completeness, reasoning coherence, trace coverage, and fit with conventions/idioms.
  - `QA FALSIFICATION REVIEW` — actively try to break the conclusion via counterexamples, alternate traces, hidden dependencies, contract mismatches, and YAGNI pressure.
  These are not duplicate reviewers. The asymmetry is the point. Falsification needs fresh context (no parent hindsight bias) and is best run as an isolated subagent in clients that support it.
- **Research** — inspects code and runtime state to compile findings. Uses local MCP tools (Hylla, gopls, repo greps) plus external sources like Context7 / `go doc`. Produces conclusions with explicit unknowns, never silent assumptions.
- **Human** (`@human`) — the operator. Routed mentions land in the operator's viewer-scoped inbox.

When you write a skill, subagent, or slash command, **name the role that owns each step.** "the agent does X" is a smell.

### 2. A template + child_rules engine

Tillsyn's templates do not just describe shapes — they generate work. A `build-task` created from a template like `default-go` may auto-generate one or more required QA subtasks owned by `qa`. The asymmetric `QA PROOF REVIEW` and `QA FALSIFICATION REVIEW` gates fire as **template-generated blockers**, not as documentation a human is supposed to remember to do.

Treat them as real gates:

- They block phase/branch completion until they pass.
- They are owned by `qa`, not by whichever agent happens to be active.
- A builder who closes their own QA subtask is bypassing the gate, not satisfying it.
- The template's `child_rules` describe what gets generated and under what conditions; a flow that ignores them ends up regenerating ad-hoc QA scaffolding the runtime would have created for free.

### 3. Three coordination surfaces with distinct semantics

These are not three names for the same thing. They mean different things and should not be merged into a single "notification" concept:

- **`till.comment`** — shared, append-only thread lane attached to a project or work item. This is the discussion + status surface. Routed `@`-mentions inside a comment (`@human`, `@dev`, `@builder`, `@qa`, `@orchestrator`, `@research`) become viewer-scoped inbox comments for the addressed identity. In a UI, `@`-mention comments belong in a **Comments** notifications section, not under "Action Required" — they are FYI, not commitments to act.
- **`till.handoff`** — structured next-action lane. An open handoff addressed to a viewer **is** an Action Required row for that viewer. Handoffs carry typed intent (what's being handed over, why, what the receiver should do next). Closing a handoff is the canonical signal that an action item has been picked up.
- **`till.attention_item`** — durable inbox substrate underneath both. Attention items are how the runtime persists "this needs someone's attention" across restarts and across the comment/handoff distinction. `till.attention_item(operation=list, all_scopes=true)` is the right way to ask "what does this viewer have outstanding right now?"

A flow that only uses comments (treating Tillsyn as a thread log) loses the structured Action Required signal. A flow that only uses handoffs (treating Tillsyn as a ticketing system) loses the discussion lane. You need all three.

### 4. Scoped auth with delegated child sessions

Tillsyn auth is **scoped**, not global-by-default:

- **Global agent sessions** are for template/global admin and project creation. They should not be the default for in-project mutations.
- **Project-scoped sessions** are the normal vehicle for branch/phase/task/comment/handoff mutations inside a project.
- **Delegated child sessions** are how an orchestrator hands authority to a builder, qa, or research role for a bounded operation. The orchestrator calls `till.auth_request(operation=create, ...)` with the receiving role's `acting_session_id` and `acting_auth_context_id`. The child role then mutates under its own scoped authority.
- **Never reuse another agent's auth session.** It conflates audit trails, breaks role ownership, and can leak authority across roles.

If your skill design assumes "the agent has admin everywhere," you have a security problem and a coordination problem in the same place.

### 5. Explicit restart recovery

When an agent restarts (new conversation, process crash, lane switch), it should **not** start from a blank slate and try to reconstruct state from worklogs. The canonical recovery order is:

1. `till.capture_state` — get the current Tillsyn-side snapshot of who/what/where.
2. `till.attention_item(operation=list, all_scopes=true)` — see all outstanding items the runtime considers unresolved.
3. `till.handoff(operation=list)` — see open structured next-actions.
4. `till.comment(operation=list)` for any thread you need to resume — get the discussion context.

Doing this in this order, every time, is what makes Tillsyn safe to restart against. Skipping it produces "the agent forgot what it was doing" failures.

### 6. Real ordering uses `depends_on` / `blocked_by`

Visual board column position is for humans. Real prerequisite ordering between work items uses `depends_on` / `blocked_by` / `blocked_reason`. A flow that infers ordering from board position is brittle.

### 7. Scoped rules live in concrete places

Policy and rules in Tillsyn live in addressable, structured locations:

- Project-level `standards_markdown`
- Template descriptions
- Child rules (the template generation contract)
- Branch / phase / task metadata: `objective`, `acceptance criteria`, `definition of done`, `validation plan`
- Node-contract snapshots

`till.get_instructions` is a *navigation aid* over these — it returns the relevant policy context for a given focus. **It is not a replacement for reading the actual fields.** A skill that calls only `till.get_instructions` and never the typed state tools is operating on stale, summary-level context.

---

## How Tillsyn Relates To Semi-Formal Reasoning And Hylla

Many agent flows pair Tillsyn with two other components:

- **Semi-formal reasoning** (arxiv 2603.01896, "Agentic Code Reasoning"). The paper introduces a structured prompting shape — premises / execution traces / formal conclusions — that functions as a *certificate* and prevents agents from skipping cases or making unsupported claims. A common adaptation in deployed flows extends the paper's shape with two extra lines: **Evidence** (forces grounding in concrete repo/tooling artifacts rather than recall) and **Unknowns** (forces uncertainty to be stated explicitly rather than smoothed over).
- **Hylla** (or any committed-code evidence graph). Provides the substrate the **Evidence** line points to: deterministic, query-able state of the committed code itself, used to ground premises rather than relying on the model's recall.

Tillsyn is what makes the **Unknowns** line load-bearing instead of cosmetic. When an agent's certificate ends with "Unknowns: X, Y," that uncertainty has to go *somewhere* — otherwise "Unknowns" becomes a polite way of declaring readiness anyway. Tillsyn gives unknowns concrete destinations:

- A **comment thread** if the unknown is a discussion question.
- A **handoff** if the unknown blocks a different role's next action.
- An **attention item** if the unknown needs to persist across restarts.
- A **template-generated QA subtask** if the unknown is "is this actually right?"

**Corollary for skill and subagent design:** any skill that produces a semi-formal reasoning certificate must pair `Unknowns` with an explicit Tillsyn target. "Unknowns: …" with no Tillsyn destination is the failure mode this whole design exists to prevent.

Without Tillsyn, the certificate's Unknowns line is decoration. With Tillsyn, it is the routing signal that drives multi-role coordination.

---

## Recommended Reading Order For New Integrators

If you are wiring a new MCP client to Tillsyn, do this in order before you write any skills or subagents:

1. Read this document.
2. Call `till.get_instructions(focus=topic, topic=agents, mode=explain)` and read everything it returns.
3. Call `till.get_instructions(focus=topic, topic=workflows, mode=explain)` and read everything it returns.
4. Call `till.project(operation=list)` to see what projects exist locally.
5. For one real project, walk through the recovery order: `till.capture_state` → `till.attention_item(operation=list, all_scopes=true)` → `till.handoff(operation=list)` → `till.comment(operation=list)` on a live thread.
6. Read the project-level `standards_markdown` and at least one template's description + child rules.
7. *Then* design your skills.

Do **not** start by writing skills that wrap `till.get_instructions` and call it a day. That is exactly the failure mode this document exists to prevent.

---

## Common Anti-Patterns To Avoid

These are the failure modes most commonly observed when an agent client treats Tillsyn as a passive ledger:

- **Flattening roles to "the agent."** Skills that say "the agent creates a build-task and then the agent does QA on it" miss the entire role model. QA must be a different role than builder, and the falsification pass must be a different context than the proof pass.
- **Treating comments as Action Required.** Routed `@`-mentions in comments belong in a Comments inbox section. Open handoffs belong in Action Required. Merging them produces noise that humans learn to ignore.
- **Hand-rolling QA gates instead of using template child_rules.** If the template would have generated QA subtasks for you, generating ad-hoc ones manually means your QA can be silently skipped — the runtime does not know they exist.
- **Reusing one agent's auth session for another role.** Breaks the audit trail and the role model in a single move.
- **Reconstructing state from worklogs after a restart.** Tillsyn already has the state. Use the recovery flow.
- **Calling `till.get_instructions` and assuming it is the source of truth.** It is policy navigation, not state. Read the typed fields directly.
- **`Unknowns: …` with no Tillsyn destination.** The whole point of Unknowns is that it routes somewhere. If your skill produces Unknowns and has nowhere to route them, the skill is incomplete.
- **Treating board column position as ordering.** Use `depends_on` / `blocked_by` for real prerequisite relationships.

---

## History Of This Document

This document was written on 2026-04-11 after a Claude Code session described Tillsyn as "the system of record for planning" — a passive-ledger framing — and was corrected by the operator. The corrected understanding here was verified against direct `till.get_instructions(focus=topic, topic=agents|workflows)` calls and against live Tillsyn project state, not reconstructed from memory or recall. The intent of putting this in the Tillsyn repo (rather than only in the calling client's local notes) is so that future integrators of any MCP client get the framing right on the first attempt instead of repeating the same correction.

If you find that this document is stale relative to the current Tillsyn runtime — for example, if a new coordination surface has been added, a role has been renamed, or the recovery order has changed — please update it in place rather than leaving the drift to confuse the next integrator.
