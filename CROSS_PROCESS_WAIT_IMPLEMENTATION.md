# Cross-Process Wait Implementation

Status: temporary detailed implementation/reference file for the current dogfood-readiness wave.

Purpose:
- capture the fully detailed consensus for cross-process live waiting and the adjacent planning items that must not be lost while the implementation is in flight,
- give builders and QA a concrete execution target without replacing `PLAN.md` as the active ledger,
- keep the immediate auth/MCP slice aligned with the broader human+agent communication direction.

Authority:
- `PLAN.md` remains the active source of truth for status, acceptance, test evidence, and closeout,
- this file is a temporary detailed reference for the current wave and should be folded back into canonical docs once the work is complete.

## 1. Product Interpretation

Tillsyn is not just task storage or auth gatekeeping.

The intended product is:
- a structured human+agent planning and execution system,
- a human oversight surface that is less opaque than markdown-first workflows,
- a truthful completion/governance layer that keeps agents honest,
- a communication substrate where humans and agents can wait, respond, hand off, and resume work without losing context.

Immediate implication:
- the current same-process auth wake path is not enough,
- dogfood readiness requires the real default human-in-the-loop path:
  - `./till mcp` waits in one process,
  - the human reviews and resolves from TUI or CLI in another process,
  - the waiting agent resumes from that same blocked line of work without manual status re-checking.

## 2. What Already Exists

Landed already:
- pre-session auth requests with continuation metadata,
- TUI and CLI auth review flows,
- durable auth-request state in SQLite,
- attention items and notification surfaces for user action,
- a same-process `LiveWaitBroker` abstraction in `internal/app`,
- auth-specific live wake in one process without app-layer polling,
- runtime log persistence and cleaner operator CLI surfaces.

Current limitation:
- the existing live wake path only works when the waiter and resolver share the same in-memory Go runtime,
- it does not cover the default local dogfood path where `./till mcp` waits in one process and TUI/CLI resolves in another.

## 3. Locked Near-Term Goal

Deliver one local cross-process waiting substrate that:
- is local-only for this wave,
- is used by stdio MCP first,
- keeps SQLite as durable truth,
- keeps TUI and CLI DRY on the human-resolution side,
- makes auth the first real consumer,
- is reusable later for comments and handoffs.

Not in this slice:
- HTTP/continuous-listening transport support,
- broad MCP outbound notification fanout for every surface,
- remote/team tenancy,
- task/subtask auth-path support,
- generalized workflow-template execution for every future agent flow.

## 4. Design Principles

1. Durable truth stays in SQLite.
- approvals, denials, comments, handoffs, and similar workflow facts are durable first.

2. Live wake is a transport convenience layer on top.
- the live channel wakes blocked callers quickly,
- reconnect/recovery still comes from durable state.

3. Human review is one shared logical path.
- TUI and CLI are two presentations over the same review/resolve logic,
- the business rules and event publication path should not diverge.

4. Auth is the first consumer, not the final destination.
- the same substrate should later support comments, handoffs, and other guarded waits.

5. Keep the current wave local-first and stdio-first.
- no HTTP dependency for dogfood readiness,
- no remote coordinator/server requirement for this phase.

## 5. Recommended Cross-Process Model

### 5.1 Transport Shape

Use a local IPC-style wake path between processes rather than app-layer polling.

Recommended minimal shape for this wave:
- each waiting process opens one local listener,
- the waiting process registers its wait interest durably in SQLite,
- the resolving process writes the durable auth decision,
- then it reads matching wait registrations and sends a wake message directly,
- the waiting process wakes immediately and rereads durable state.

This gives:
- same-machine coordination,
- no separate always-on broker daemon,
- no extra remote transport contract,
- reusable event types without making SQLite itself the live channel.

### 5.2 Durable Data Responsibilities

SQLite should hold:
- auth request truth,
- a durable latest-event/outbox surface for replay/recovery,
- waiter registrations for currently live blocked callers.

SQLite should not become:
- a polling loop pretending to be a live channel,
- the only delivery mechanism for the normal success path.

### 5.3 Live Delivery Responsibilities

The live channel should:
- wake waiting callers quickly,
- allow the blocked MCP tool call to return immediately once the human resolves the request,
- tolerate stale registrations and clean them up,
- remain generic enough for future `comment` and `handoff` consumers.

## 6. Why Auth Is The First Consumer

Auth is the cleanest first consumer because:
- it already has a long-lived wait call: `claim_auth_request(wait_timeout=...)`,
- the lifecycle is simple and well-defined,
- success is easy to verify,
- it is already part of the current dogfood flow.

What “auth first consumer” means:
- build the generic waiter/event substrate once,
- plug auth into it first,
- prove:
  - pending request,
  - blocked wait in `./till mcp`,
  - human approval/deny/cancel in TUI or CLI,
  - immediate wake/return in the waiting requester.

After that, reuse the same substrate for:
- comments:
  - human or another agent comments,
  - waiting agent wakes and continues,
- handoffs:
  - builder waits on QA,
  - QA changes handoff status or returns notes,
  - builder wakes and continues,
- later guarded workflow events defined by templates/policy.

## 7. Dry Human Review Requirement

CLI and TUI should share:
- the same app/service auth resolution calls,
- the same decision rules,
- the same event publication path,
- the same audit/note semantics.

They should differ only in:
- layout,
- navigation,
- local interaction model.

This is important because:
- the human process is one product concept,
- TUI and CLI are just two ways to exercise it,
- drift here would make debugging and policy behavior harder.

## 8. Dogfood-Ready Acceptance For This Slice

The slice is dogfood-ready only if all of the following are true:

1. `./till mcp` can open a blocked auth continuation wait in one process.
2. TUI approval in another process wakes that wait immediately.
3. CLI denial/cancel in another process wakes that wait immediately.
4. No manual “check status again” step is required for the normal path.
5. If the live wake path fails, durable state still reflects the correct auth decision.
6. Stale wait registrations are cleaned up or ignored safely.
7. `mage ci` and remote GitHub Actions all pass.
8. The collaborative E2E auth/MCP worksheet is updated before manual rerun.

## 9. Implementation Slices

### Slice A: Local Cross-Process Wait Adapter

Build a new adapter that implements `app.LiveWaitBroker` with cross-process semantics.

Responsibilities:
- durable waiter registration,
- durable latest-event storage for replay,
- local listener lifecycle for blocked waiters,
- publish-to-waiter wake delivery,
- stale registration cleanup.

Likely package shape:
- `internal/adapters/livewait/localipc`

Acceptance:
- one broker instance can wait,
- another broker instance in the same runtime/DB can publish,
- the waiter wakes without polling loops.

### Slice B: App/Auth Integration

Keep auth on the existing app seam and swap in the cross-process broker at runtime bootstrap.

Responsibilities:
- preserve current auth lifecycle semantics,
- make cross-process waiting the default live path in real runs,
- keep same-process behavior valid in tests/fallbacks,
- keep durable replay behavior correct.

Acceptance:
- `ClaimAuthRequest(wait_timeout=...)` uses the cross-process broker in normal runtime construction,
- approve/deny/cancel publish through the same broker path,
- existing auth request semantics do not regress.

### Slice C: Runtime Wiring And Docs

Wire the runtime to construct the cross-process broker and update the active docs.

Responsibilities:
- inject the broker when building the app service,
- keep operator/runtime logs meaningful,
- document what is now complete vs what is still deferred,
- refresh the collab worksheet for the next manual run.

Acceptance:
- `./till`, `./till mcp`, and CLI auth commands use the same broker-backed runtime,
- docs no longer overstate or understate the current behavior.

## 10. QA Plan

Required lane pattern for this wave:
- 2 QA reviewers per builder lane,
- 1 final QA reviewer after integration,
- repo gates after integration,
- remote GitHub Actions watch after push.

QA focus areas:
- lost-wakeup races,
- stale registration cleanup,
- auth state/audit regressions,
- CLI/TUI resolution parity,
- docs accurately matching the landed behavior.

## 11. Manual Collaborative Test Focus After Landing

Primary manual checks:
- pending auth request appears in TUI,
- waiting requester in `./till mcp` stays blocked,
- TUI approval wakes the requester,
- CLI denial wakes the requester,
- CLI cancel wakes the requester,
- resulting request/session state is readable from inventory/capture-state surfaces,
- no misleading “manual re-check” step is needed for the normal path.

## 12. Open Follow-On Work That Must Not Be Forgotten

These are intentionally not all MVP blockers for this slice, but they are part of the broader direction.

### 12.1 Comments And Handoffs

After auth:
- comments should be able to wake waiting humans/agents,
- handoff transitions should be able to wake the next actor,
- these should reuse the same substrate rather than inventing separate wait systems.

### 12.2 Task/Subtask Auth Paths

Current auth path scope remains:
- project,
- project/branch,
- project/branch/nested phase lineage.

Deferred but important:
- more granular task/subtask-level authority models,
- likely needed for builder vs QA separation on fine-grained work,
- TUI/path-picking UX for that will need careful redesign before it is safe and understandable.

### 12.3 Node-Type And Agent-Type Templates/Policy

Still part of the broader value prop:
- node-type templates define work structure, metadata, required checks, default child work, and completion rules,
- agent-type policy defines authority, delegation, and approval bounds,
- the wait/communication substrate must stay generic enough to serve those future template-defined flows.

### 12.4 Plan Hygiene In Tillsyn

The intended product behavior is:
- the active plan lives in Tillsyn,
- orchestrators and users keep it current there,
- when the plan changes, outdated nodes are updated or archived in Tillsyn,
- this reduces markdown drift, hidden state, and missed work for both humans and agents.

### 12.5 Search Roadmap

This is not the current implementation slice, but it is necessary follow-on planning:
- keyword search,
- scoped/path-aware search,
- semantic/vector search,
- hybrid/deduped search,
- provenance metadata showing which search modes matched each node,
- rich filtering across project/scope/path/state/kind/labels/metadata and similar,
- useful surfaces for both humans and agents.

## 13. What This File Does Not Change

This file does not replace:
- `PLAN.md` as the active execution ledger,
- the existing section-by-section collaborative remediation flow,
- the requirement to update README/collab docs when behavior changes land,
- the requirement to ask the user whether to delete this temporary file once its contents are fully folded back into canonical docs.
