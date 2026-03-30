# Template, Agent, And Communication Consensus

Status: working consensus draft for discussion only.

Purpose:
- capture the current user+agent consensus before updating `PLAN.md`, `README.md`, or other canonical docs
- keep one temporary place for the missing product-contract details around templates, agent types, communication, and truthful completion
- avoid losing nuance while we refine the post-dogfood roadmap language

Scope note:
- this file is a temporary consensus worksheet
- it does not replace the active run contract in `PLAN.md`
- once the open questions below are resolved, this material should be folded back into the main docs and this file can be reduced or removed

## Core Value Prop

Tillsyn is not just kanban plus auth.

Tillsyn is a local-first human+agent planning, execution, oversight, tracking, and communication system that replaces messy markdown-file-first workflows with one structured, inspectable, auditable source of truth.

The value prop is:
- humans can actually see what is planned, what is being worked, what is blocked, what changed, and why
- orchestrators can coordinate and monitor work without loading giant markdown files into context
- subagent work becomes visible and reviewable instead of opaque
- communication becomes explicit, structured, inspectable, and less opaque for both humans and orchestrators
- required checks, subtasks, evidence, and approvals can be enforced instead of relying on trust or memory
- planning, worklogs, comments, review notes, approvals, and handoffs stay attached to the right nodes instead of being scattered across repo markdown
- repo roots stay cleaner because planning and execution state move out of sprawling ad-hoc markdown files

Core framing:
- Tillsyn is the structured planning, execution, oversight, and communication layer for human+agent work
- node types define work structure
- agent types define allowed authority and delegation
- completion contracts enforce truth
- MCP long-lived coordination provides the live wait/resume/notify channel, while persisted Tillsyn state remains the durable record

UX/Surface framing:
- the TUI should stay logical, styled, and user-friendly for the workflows it exposes
- the TUI should prefer reusable shared surfaces/components instead of bespoke duplicated behavior
- the CLI should expose full capability even when the interaction style differs from the TUI
- the product should avoid a split where important power or recovery actions exist only in one surface

## Type Systems

There are two distinct type systems.

### 1. Node Types

Node types define the work contract for:
- project
- branch
- phase
- task
- subtask

Node-type templates may define:
- metadata schema and default values
- default paradigms, workflows, test expectations, and QA expectations
- default child work such as phases, tasks, subtasks, review steps, and checklist items
- managed markdown guidance sections
- required completion criteria, evidence, and child-completion rules

### 2. Agent Types

Agent types define the authority and workflow role for principals.

MVP default roles:
- orchestrator
- builder
- qa
- system

Future expansion may add roles such as:
- reviewer
- researcher
- docs
- security
- release

Agent-type policy may define:
- allowed read and write scopes
- allowed node kinds and work categories
- allowed delegation targets
- what can be auto-approved within template policy bounds
- what always requires user review
- what counts as valid signoff or evidence from that agent type

## Template Goal

Templates are not meant to be overly opinionated or force one workflow.

The product goal is:
- give users strong defaults that can replace markdown-first planning
- let users configure node-type and agent-type behavior
- let orchestrators help create, update, and maintain templates through MCP and TUI/CLI flows
- keep the user as the final approval authority for impactful policy/template changes

Configuration principle:
- Tillsyn should not force one rigid workflow
- Tillsyn should provide understandable defaults, configurable template behavior, and clear reviewable controls
- users may choose configuration directly or with orchestrator assistance, but important changes should remain user-reviewable

## Current Intent For Template Behavior

When a node is created, the template for that node type should be able to:
- auto-populate metadata and default values
- seed paradigms, validation plans, and workflow expectations
- create default child items
- attach default completion criteria and evidence expectations
- attach managed agent-facing guidance sections where appropriate

Examples already agreed in discussion:
- a Go project can start with language-specific metadata, testing expectations, and default planning structure
- a branch can auto-create CI or verification tasks such as `mage ci`
- a phase can auto-create QA or review tasks
- a task can auto-create subtasks for package-level tests, QA checks, or validation follow-ups

## Completion Honesty Goal

The system should make it harder for agents to claim completion when required work has not actually been finished.

High-level intent:
- required generated subtasks/checklists/evidence should matter
- node completion should be gated by explicit completion contracts
- humans should be able to see why something is blocked
- orchestrators should be able to monitor compliance without loading large markdown plans into context

Open policy detail:
- which rules should block completion vs warn only
- when downstream QA failure should reopen upstream work
- which overrides should be available to users and how they are audited

Recommended default model:
- hard-block completion on unfinished required children, unfinished required checklist items, missing required evidence, and missing required approvals
- soft-warn on optional or informational items
- default QA failure behavior should reopen the validated node and block affected parents rather than automatically reopening the full ancestor chain
- users may choose stricter cascade behavior through policy/template configuration
- TUI and CLI surfaces should explain blocked state clearly so users can quickly see what remains and why completion is denied

Current consensus:
- this default honesty model is the intended MVP direction

## Agent Communication Goal

Tillsyn should make agent-to-agent, human-to-agent, and human-to-subagent communication first-class and less opaque.

Desired outcomes:
- humans can see what subagents are doing
- humans can comment directly on work being done by subagents if desired
- orchestrators can coordinate other agents without relying on giant markdown files
- comments, attention items, approvals, review notes, and handoff state stay attached to the relevant nodes
- long-running wait/resume flows let agents pause for approvals, QA, or dependent work instead of faking completion
- the system should reduce opacity for humans rather than increasing it

## MCP / Transport Understanding

Current consensus:
- MCP and `mcp-go` are transport/channel mechanisms for Tillsyn logic
- Tillsyn owns the domain rules for templates, auth, delegation, completion truth, and coordination semantics
- long-lived open wait/resume behavior is required for the agent communication/coordination model
- the user used webhook language as an analogy, not as a claim that MCP already provides a business-level webhook workflow primitive
- persisted Tillsyn state should remain the source of truth; long-lived MCP wait/listen behavior is the live transport layer, not the authoritative workflow record

## Auth And Delegation Intent

The auth model should go beyond the current path-only shape.

Desired direction:
- orchestrators may request or create narrower subagent auth only within template-defined and policy-defined bounds
- those bounds should be shaped by node type, node scope, agent type, and user-approved policy
- anything outside those bounds should escalate to the user for review
- builder vs qa vs reviewer style separation should be possible through configurable policy, not hard-coded one-size-fits-all product rules

Current agreement:
- the product should ship good defaults
- users should be able to configure the rules
- orchestrators may help manage these templates/policies, but impactful changes should remain user-reviewable

## Template Inheritance And Reseeding

Current consensus direction:
- inheritance and override behavior should be configurable rather than rigidly opinionated
- users should be able to choose how templates combine across levels
- defaults should be reasonable and easy to understand
- orchestrators may help apply or preview inheritance/override choices, but user review should remain available for impactful changes

Likely configuration primitives:
- merge
- replace
- append
- inherit-none

Current reseeding direction:
- default behavior is that existing nodes stay as created
- if templates change later, users may choose to update or migrate existing nodes
- MVP should include a usable TUI and CLI path for fast bulk or whole-scope template updates by type/scope
- reseeding should be previewable before application where practical

## Agent Authority Direction

Current consensus direction:
- authority should be policy-driven and configurable, not hard-coded to one opinionated workflow
- the product still needs explicit action categories so policy remains understandable and enforceable

Likely action categories:
- read
- comment
- create-child
- edit-node
- request-auth
- approve-auth-within-bounds
- mark-in-progress
- mark-complete
- reopen
- attach-evidence
- signoff
- resolve-attention
- archive-or-cleanup

Current consensus:
- these action categories are the intended MVP policy vocabulary

Direction:
- keep the action vocabulary small for MVP
- separate node scope, node type, and action class in policy evaluation
- allow later expansion if users need more granular roles or permissions

## Waiting And Recovery Direction

Current consensus direction:
- agents should be able to wait on approvals, QA, research, and dependent work through long-lived MCP transport behavior
- waiting should not depend on one fragile open connection being the only record of state
- persisted wait/handoff/dependency state should survive disconnects
- reconnect and polling/capture-based fallback should remain available when live waiting is interrupted

Recommended default model:
- explicit wait states persisted in Tillsyn
- heartbeats for live agent sessions where relevant
- explicit timeout and cancellation behavior
- reconnect-and-resume when the transport drops
- notifications for live routing, with comments/attention/handoff state preserved durably
- hanging waits, stale heartbeats, abandoned approvals, and incomplete delegated work should be discoverable after restart so a new orchestrator or replacement subagent is not blind to unfinished state
- the system should expose restart-safe discovery surfaces for orphaned or waiting work so orchestrators can recover when an LLM session collapses, an agent process dies, or a transport/session expires
- default safety direction should avoid letting authority or waiting state silently linger forever without visibility
- durable recovery must persist beyond auth/session timeout; expired sessions must not erase the visibility of unfinished or hanging work

Recovery visibility direction:
- full recovery discovery should be orchestrator-first by default
- replacement subagents should not independently discover broad hanging state by default, because they may be exposed to stale or no-longer-relevant work after plans changed
- when an orchestrator requests a new subagent for a scope that has hanging or orphaned delegated work, the system should surface that fact to the orchestrator first
- the orchestrator should then either:
  - restart/replace the subagent with the relevant recovery context when the plan is still valid, or
  - stop and involve the user in cleaning up/archive/removal of stale Tillsyn state when the plan changed or ambiguity exists

Staleness direction:
- distinguish agent/session staleness from work-state staleness
- an expired session or dead heartbeat should invalidate authority, but not erase unfinished work, waits, or handoff obligations
- default behavior should favor visibility and orchestrator/user review over silent cleanup
- session timeout must not make recovery state disappear

Automatic restart direction:
- automatic restart is allowed only when failure is clearly operational rather than semantic
- examples include transport drops, LLM/session crashes, expired sessions, or heartbeat loss
- before restart, the orchestrator should receive recovery discovery information
- if the hanging state still matches the active plan, the orchestrator may restart a replacement subagent with recovery context
- if the state is ambiguous, stale, or inconsistent with the current plan, the orchestrator should stop and involve the user
- default safety direction should allow limited automatic recovery for operational failures while requiring human review when plan intent may have changed

Likely MVP default:
- at most one automatic replacement attempt per hanging delegated lane when the system classifies the failure as operational and sees no conflicting stale state
- otherwise require orchestrator review, and often user review

## Communication Surface Direction

Current consensus direction:
- communication should be first-class, visible, and attached to nodes
- humans must be able to monitor and join communication with subagents if desired
- orchestrators must be able to coordinate without loading giant markdown plans into context

Likely surface roles:
- comments for discussion and reasoning
- attention items for actionable blockers or required follow-up
- notifications for routing and awareness
- approval notes for auth and policy decisions
- completion evidence for proof of work
- handoff state for structured “finished / waiting / next actor” coordination

Usability direction:
- communication and recovery surfaces should be understandable from both TUI and CLI
- users should be able to see what subagents are doing, what they are waiting on, and what needs intervention without reading giant markdown files

Current handoff direction:
- handoff state should be a first-class structured surface, not just an implicit convention inside comments
- this is the intended MVP direction for a first-class handoff object
- humans and orchestrators should be able to scan handoff state quickly without parsing long prose

Likely handoff object shape:
- source node
- source agent / role
- target node or target role
- status such as ready, waiting, blocked, failed, returned, superseded
- reason / summary
- required next action
- required evidence or missing evidence
- related approval or dependency references
- created_at / updated_at
- resolved_at / resolution note

## Agent Policy Action Categories

Purpose:
- agent policy needs explicit action categories so permissions stay understandable, auditable, and configurable
- this avoids vague policy like “builder can do implementation things” without defining what that means in practice

Likely MVP action categories:
- read
- comment
- create-child
- edit-node
- request-auth
- approve-auth-within-bounds
- mark-in-progress
- mark-complete
- reopen
- attach-evidence
- signoff
- resolve-attention
- archive-or-cleanup

Direction:
- keep the action vocabulary small for MVP
- separate node scope, node type, and action class in policy evaluation
- allow later expansion if users need more granular roles or permissions

## MVP Role Matrix Direction

Current consensus:
- keep MVP role defaults simple
- the primary default agent roles should be `orchestrator`, `builder`, and `qa`
- note explicitly that future users may expand this with additional roles, but MVP should not scare users away with too much complexity

High-level default role intent:
- orchestrator:
  - coordinate work
  - request and manage bounded subagent auth
  - monitor recovery/hanging state
  - route handoffs and approvals
- builder:
  - implement and update assigned work within allowed scope
  - attach evidence and comments
  - request narrower help when allowed
- qa:
  - validate work
  - attach QA evidence
  - reopen or fail validated work according to policy
  - avoid broad implementation mutation by default

More explicit MVP default role behavior:
- orchestrator:
  - broad read within allowed scope
  - comment
  - create-child where allowed
  - request auth
  - approve bounded subagent auth
  - monitor recovery and hanging state
  - route handoffs
  - perform limited cleanup/archive actions with reviewable audit trail
- builder:
  - read assigned scope
  - comment
  - create child work where allowed
  - edit implementation nodes in assigned scope
  - attach evidence
  - mark complete on builder-owned work where policy allows
  - should not sign off QA by default
- qa:
  - read assigned scope
  - comment
  - attach QA evidence
  - mark QA work complete or failed according to policy
  - reopen validated upstream work when QA fails
  - should not broadly rewrite implementation work by default

## MVP Vs Roadmap Split

MVP expectations:
- safe and previewable template apply/update flows
- scope-aware bulk apply for future nodes or existing scope
- restart-safe wait/recovery visibility
- first-class structured handoff state
- simple default role matrix (`orchestrator`, `builder`, `qa`)
- discoverable bootstrap/help surfaces for users and orchestrators

Roadmap, not required for MVP:
- richer diff/preview UX for template reseeding and migration
- highly granular field-by-field merge conflict resolution for retroactive template updates
- more advanced role catalogs beyond the MVP default roles
- deeper handoff automation and advanced dependency orchestration
- more sophisticated policy editors and visual authority inspectors
- broader template libraries and richer preset packs

## Remaining Questions For Main-Docs Fold-In

Most major product-contract questions above are now settled for this worksheet.

Remaining items to finalize when folding this into `PLAN.md` and `README.md`:

1. Override and audit detail:
- who may override blocked completion
- what override reasons or evidence must be recorded
- how override visibility should appear in TUI, CLI, and MCP surfaces

2. Cleanup and stale-state handling detail:
- exact user-facing behavior for cleaning up stale waits, stale handoffs, or superseded delegated work
- exact archive/delete/revoke flows when plans changed after hanging work remained behind

3. Bootstrap presentation detail:
- README wording and positioning
- TUI bootstrap copy and discovery flow
- MCP helper surfaces for “learn Tillsyn”, template setup, and policy refinement

4. MVP acceptance detail:
- what exact TUI/CLI/MCP capabilities must be present before this template/agent/communication scope is called MVP feature-complete

Current bootstrap direction:
- the system should be useful with no mandatory extra setup beyond normal startup
- strong defaults should let users start immediately
- enough guidance should exist in README, TUI, and MCP helper surfaces so users and orchestrators can refine the setup later without loading unnecessary context upfront
- an MCP surface for learning about Tillsyn itself is valuable so orchestrators can discover product rules and help users configure templates/policy only when needed

Current MVP bootstrap direction:
- day-one use should work with strong defaults and no mandatory advanced setup
- a user should be able to start `till` and get immediate value
- a Codex-oriented or similar agent workflow should be supported through a discoverable preset or clear default behavior, not a heavy configuration burden

## Implementation Slices

Execution direction:
- this scope should be delivered in small validated slices
- each implementation slice should favor one clear primary outcome
- test and commit after meaningful validated slices
- when implementation work is parallelized, use one builder lane plus two independent QA/review lanes for each builder lane

### Slice 1: Shared Contracts And Durable State

Goal:
- establish the missing domain/app contract for hierarchy-wide templates, structured handoffs, and durable wait/recovery state without overextending the UI yet

Focus:
- node-template contract expansion
- handoff object contract
- persisted waiting/recovery/discovery contract
- policy action vocabulary

### Slice 2: Agent Policy And Bounded Delegation

Goal:
- land MVP agent-type policy with simple default roles and bounded orchestrator delegation/recovery behavior

Focus:
- default MVP roles (`orchestrator`, `builder`, `qa`)
- action-category evaluation
- bounded subagent auth/delegation rules
- stale authority vs durable work-state handling

### Slice 3: Template Application Beyond Today’s Task-Centric Path

Goal:
- extend template behavior beyond the current task-only action path into the intended hierarchy-aware node flow

Focus:
- hierarchy-wide template application model
- inheritance/override primitives
- reseeding/apply-scope behavior
- truthful completion-contract seeding across levels

### Slice 4: TUI And CLI Product Surfaces

Goal:
- expose the new template/policy/handoff/recovery capability through logical, reusable TUI and full-capability CLI surfaces

Focus:
- shared TUI form/screen components for new template/policy workflows
- clear blocked-state and recovery-state UX
- first-class handoff and waiting visibility
- full CLI capability for inspection, apply, recovery, cleanup, and policy/template operations

### Slice 5: Bootstrap, Discovery, And Dogfood Validation

Goal:
- make the system understandable and usable with strong defaults, good bootstrap/discovery guidance, and real collaborative validation

Focus:
- bootstrap defaults and presets
- README / TUI / MCP discovery surfaces
- “learn Tillsyn” / helper-tool direction
- collaborative E2E dogfood validation for real human+agent workflows
