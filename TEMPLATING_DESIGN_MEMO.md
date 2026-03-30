# Templating System Design Memo

Status: design proposal for `agent/templating-design`

Date: 2026-03-29

Scope:
- design only
- no implementation in this lane
- intended to lock the product model before a later implementation branch

## Purpose

Define templating in Tillsyn correctly.

The earlier draft leaned too hard toward "scaffolding defaults" and repo-file authoring. The corrected model is:
- templates are workflow-and-authority contracts first
- scaffolding is only part of the contract
- SQLite should be the active source of truth for those contracts

Source note:
- Context7 was used before this doc update for SQLite storage assumptions.
- Context7 still did not resolve the Go stdlib template package cleanly, so local `go doc text/template` remains the bounded fallback reference for trusted plain-text/markdown rendering mechanics.

## Corrected Core Definition

Templates in Tillsyn are not just "how to prefill a node".

Templates are the system that says:
- when a node of this type is created, which follow-up nodes are auto-generated
- what node kind or type those generated nodes are
- which actor kind is responsible for each generated node
- which actor kinds may edit each generated node
- which actor kinds may mark each generated node complete
- whether orchestrator completion is allowed as an override for that generated node
- whether a human may always override and close it
- whether that generated node blocks parent completion
- whether that generated node blocks branch or phase completion

So the real goal is:
- keep work small and trackable
- make required follow-up work explicit
- keep agents out of giant markdown plans
- prevent agents from claiming completion when required role-specific follow-up work is still open

That is the first and foremost meaning of templating in Tillsyn.

## What This Means For The Product Model

The cleaner model is:
- `scope level` = project, branch, phase, task, subtask
- `node kind/type` = research, build, qa-pass, release-check, design-review, and so on
- `actor kind/type` = human, orchestrator, builder, qa, research, and so on
- `system actor` = the internal runtime actor that creates generated nodes and records audit provenance
- `template` = generated work graph plus role contract plus completion gates
- `auth/policy` = whether the current actor/session actually has the required actor kind and scope
- `completion gate` = whether the required generated work was completed by an allowed actor or a human

The critical correction is that `scope level` is not enough.

Today the system is mostly level-based:
- can this actor touch project vs branch vs phase vs task scope?

The clarified product needs to move toward:
- can this actor kind perform this action
- on this node kind
- at this scope path
- under this stored node contract?

That is a stricter and more useful model.

Important clarification:
- `system` should exist as an internal actor for audit/provenance
- but `system` should not be treated as the workflow owner kind for generated work
- generated work should instead carry a required or responsible actor kind such as `builder` or `qa`
- in other words:
  - `created_by = system`
  - `responsible_actor_kind = qa`
  - these are not the same field

## What Templates Should Mean In Tillsyn

### 1. Project Templates

Project templates should define:
- default project metadata
- default project standards/guidance
- which node kinds are available by default in that project
- default child rules for project-root creation
- default template rules for branch, phase, task, and subtask creation inside that project

Examples:
- a `go-service` project template
- a `python-service` project template
- a `research-project` template

These are not just style presets. They are project-level collaboration rules.

### 2. Branch / Phase / Task / Subtask Templates

Node templates should define:
- what child work gets auto-generated
- what role owns each generated child
- who may edit it
- who may complete it
- whether it blocks its parent or containing scope
- any default metadata or guidance attached to it

Examples:
- a build task auto-generates a QA pass child
- a research phase auto-generates evidence review and signoff tasks
- a release branch auto-generates validation and coordination tasks

### 3. Standards / Docs Payloads

Standards and docs still matter, but they are secondary outputs:
- project standards markdown
- role-facing notes or guidance
- human-readable explanations of what the generated work means

These should help humans understand the contract, not replace the contract.

### 4. Completion Truth

Templates should make completion truthful by default.

If a template-generated child is marked:
- required for parent done
- or required for branch/phase done

then the parent or containing scope should not move to done until that child is done by:
- an allowed actor kind with valid scope auth
- or a human
- or an explicitly allowed orchestrator override, if the rule permits it

## Current Product Implication

The current code already has pieces of this idea:
- kind catalog
- kind payload validation
- create-time child generation
- completion contract checks
- scope-based auth/lease checks

But those pieces are not joined cleanly enough yet.

What is missing conceptually:
- actor kind restrictions on generated nodes
- node-kind-aware auth rules beyond level-only scope checks
- stored node contracts that preserve what was required at creation time
- clear human-readable TUI and CLI surfaces for "who can do what and why is this blocked?"

## Recommended Type Systems

Tillsyn should explicitly recognize three related but separate type systems.

### 1. Scope Level

This stays structural:
- project
- branch
- phase
- task
- subtask

### 2. Node Kind / Node Type

This is the work category:
- build
- qa-pass
- research
- design-review
- release-check
- docs-update

This is broader than just branch/phase/task/subtask.

### 3. Actor Kind / Actor Type

This is the authority category:
- human
- orchestrator
- builder
- qa
- research

For MVP, I would keep this list small and explicit. I would push back on fully user-defined actor kinds in the first wave, because that explodes the auth matrix too early.

The internal `system` actor should remain separate from this workflow actor-kind list:
- it is for automatic generation, migrations, and audit trails
- it is not a normal owner kind that humans assign to work
- it should not receive general-purpose auth the way orchestrator/builder/qa do

## Static Defaults Vs Parameterized Templates

### Static Contract Data

Keep these static and stored structurally:
- generated child rules
- node kind defaults
- actor kind permissions
- blocker flags
- allowed editor kinds
- allowed completer kinds
- orchestrator override policy
- metadata defaults
- completion policy flags

These should be stored as structured DB data, not rendered text.

### Parameterized Text

Allow parameterization only for text-like fields:
- titles
- descriptions
- standards markdown
- checklist text
- guidance text

Parameterization should stay narrow in MVP:
- trusted authors only
- plain text / markdown only
- no arbitrary scripting language
- no hidden repository inspection

## Where Templates Should Live

The active source of truth should be SQLite.

That is the strongest correction to the earlier draft.

### Why SQLite Should Be Canonical

Templates in this clarified model are live operational rules:
- they affect node creation
- they affect auth checks
- they affect completion gates
- they affect readable UI state
- they need transactional updates with node creation and completion changes

SQLite fits that model because Tillsyn already uses one local, self-contained, transactional database and these rules are app state, not just import/export artifacts.

So my updated recommendation is:
- one DB file
- one authoritative template/rule store
- one authoritative binding/provenance store

### What Should Not Be Canonical In MVP

Not canonical:
- JSON files
- config files
- multiple saved locations for the same active rules

Those can exist later as:
- export/import
- backup
- diff/review helper
- library sharing

But not as the primary live source of truth in MVP.

### Recommended Storage Namespaces

Inside SQLite, support a small number of namespaces:
- `global`
- `project`
- `draft`

Recommended MVP behavior:
- `global` templates are reusable base templates in the DB
- `project` templates are project-scoped copies or customizations
- `draft` templates are unapproved proposals, typically created by an orchestrator or a user during editing

Promotion should happen inside the DB:
- clone global to project
- clone project to global only with human approval
- promote draft to active only with human approval

That avoids multi-location drift.

Recommended update model:
- a project may customize its active project-scoped library without affecting any other project
- a newer global library revision should not silently alter existing projects
- instead, a human or an explicitly allowed orchestrator should run an adoption flow:
  - preview global vs project differences
  - choose to clone/update into the project library
  - then optionally apply the updated rules to existing managed/generated nodes

### Focused MVP SQLite Layout

For MVP, the active template contract should be relational inside SQLite.

Do not store the live template system as one large `template_json` blob.

Recommended canonical tables:
- `template_libraries`
  - one row per global, project, or draft library
  - includes `scope`, `project_id`, `status`, `source_library_id`, human approval fields, and audit fields
- `project_template_bindings`
  - one active library binding per project
  - `project_id` should be unique in MVP
- `node_templates`
  - one row per `(library_id, scope_level, node_kind_id)`
  - stores human-readable display name plus default guidance/default payload references
- `node_template_guidance_sections`
  - managed human-readable sections such as standards notes, checklist explanations, or role guidance
- `template_child_rules`
  - one row per generated child rule
  - stores child scope level, child node kind, responsible actor kind, blocker flags, orchestrator override flag, ordering, and title/description templates
- `template_child_rule_edit_actor_kinds`
  - normalized allowed-editor actor kinds per child rule
- `template_child_rule_complete_actor_kinds`
  - normalized allowed-completer actor kinds per child rule
- `node_contract_snapshots`
  - one row per generated node that captures the resolved rule at creation time
  - includes source library/template/rule ids, audit actor identity, responsible actor kind, blocker flags, and override policy
- `node_contract_edit_actor_kinds`
  - normalized allowed-editor actor kinds per generated node snapshot
- `node_contract_complete_actor_kinds`
  - normalized allowed-completer actor kinds per generated node snapshot

Recommended relational rule:
- policy columns and actor-kind permissions should live in rows and join tables
- small leaf JSON should be allowed only where the product already has JSON-shaped payloads, such as kind-payload defaults
- the canonical contract should not depend on a single serialized template blob

This is the key difference from the current `kind_catalog.template_json` design.
- `kind_catalog` should continue to own node-kind registry concerns:
  - allowed levels,
  - allowed parent levels,
  - payload schema
- template tables should own generated work, actor rules, and completion gates

## Recommended Nouns And Data Shapes

### Product Nouns

- `template library`: a named collection of template rules
- `node template`: the rule for creating one node kind at one scope level
- `child rule`: one auto-generated child contract
- `template binding`: which project uses which template library
- `node contract snapshot`: the resolved rule persisted on a created node
- `actor kind`: the authority category used by auth and completion rules
- `responsible actor kind`: the workflow role expected to do the work
- `audit actor`: who or what created/approved/bound/applied the rule or node

Recommended wording:
- prefer `responsible actor kind` over `owner kind`
- `owner` already reads like human ownership or assignment
- the template contract needs a clearer workflow term

### Suggested Go Shapes

```go
type TemplateLibrary struct {
	ID          string
	Scope       string // global|project|draft
	ProjectID   string
	Name        string
	Description string
	Status      string // draft|approved|archived
	CreatedBy   string
	ApprovedBy  string
	CreatedAt   time.Time
	ApprovedAt  *time.Time
}

type NodeTemplate struct {
	ID               string
	LibraryID        string
	AppliesToLevel   domain.KindAppliesTo
	NodeKindID       domain.KindID
	DisplayName      string
	MetadataDefaults domain.TaskMetadata
	ManagedGuidance  []ManagedSectionTemplate
	ChildRules       []ChildRule
}

type ChildRule struct {
	ID                        string
	NodeKindID                domain.KindID
	AppliesToLevel            domain.KindAppliesTo
	TitleTemplate             string
	DescriptionTemplate       string
	RequiredActorKind         string
	EditableByActorKinds      []string
	CompletableByActorKinds   []string
	OrchestratorMayComplete   bool
	RequiredForParentDone     bool
	RequiredForContainingDone bool
}

type TemplateBinding struct {
	ProjectID        string
	TemplateLibraryID string
	BoundBy          string
	BoundAt          time.Time
}

type NodeContractSnapshot struct {
	NodeID                     string
	SourceTemplateLibraryID    string
	SourceNodeTemplateID       string
	SourceChildRuleID          string
	CreatedByActorType         string
	CreatedByActorID           string
	NodeKindID                 domain.KindID
	RequiredActorKind          string
	EditableByActorKinds       []string
	CompletableByActorKinds    []string
	OrchestratorMayComplete    bool
	RequiredForParentDone      bool
	RequiredForContainingDone  bool
}
```

The important piece is `NodeContractSnapshot`.

Do not rely only on the live template definition after creation. Persist the resolved contract on the generated node so the truth of that node does not silently change later.

The important identity split is:
- audit actor: who or what created this row
- responsible actor kind: which workflow role is meant to do the work

For generated nodes, those will usually be:
- audit actor = `system`
- responsible actor kind = `builder` or `qa` or another workflow actor kind

## Auth And Completion Model

The auth model needs to expand beyond level-only checks.

Recommended evaluation shape:
- actor kind
- scope path
- node kind
- action
- stored node contract

Example actions:
- create child
- edit node
- mark in progress
- mark complete
- reopen
- override complete

Recommended rule:
- scope authorization remains the outer boundary
- node-contract authorization becomes the inner boundary
- audit actor identity does not grant workflow authority by itself

That gives a sane MVP path:
- keep current level-based auth as the coarse scope check
- add node-kind and actor-kind checks for templated/generated work

Clarification on level vs kind:
- a QA task inside a phase is still `level=task`
- the thing that makes it QA rather than build is `node kind`
- so level does not cover QA-vs-build by itself
- it is always the combination of:
  - scope level,
  - node kind,
  - and actor kind

Examples:
- `phase / qa-phase / qa`
  - a phase-level QA review lane
- `task / qa-pass / qa`
  - a task-level QA pass under a build phase
- `subtask / evidence-capture / research`
  - a subtask-level research evidence requirement under another task

So in your example:
- a QA run inside a phase is usually still `level=task`
- if it is a child of a build task, it might be `level=subtask`
- `qa-pass` vs `build-task` is the node-kind distinction
- level only tells the structural slot

## Focused Runtime Flow

The MVP runtime should work like this.

### 1. Create Flow

When a human, orchestrator, or builder creates a node:
- resolve the active project template library
- resolve the matching `node_template` by:
  - project binding,
  - scope level,
  - node kind
- merge human-entered metadata with template defaults
- create the requested node
- generate each required child rule as a real node inside the same transaction
- stamp each generated node with:
  - audit actor identity
  - source library/template/rule ids
  - responsible actor kind
  - allowed editor actor kinds
  - allowed completer actor kinds
  - parent/scope blocker flags

Generated nodes should normally look like this:
- `created_by_actor_type = system`
- `created_by_actor_id = tillsyn-system-template`
- `responsible_actor_kind = qa` or `builder` or another workflow role

### 2. Edit Flow

When someone edits a generated node:
- check scope auth first
- then check stored node-contract permissions
- humans stay allowed
- non-human actors must match allowed actor kind plus scope

### 3. Complete Flow

When someone marks a node done:
- check scope auth first
- check node-contract complete permissions
- allow human override-complete always
- allow orchestrator complete only if the stored rule explicitly allows it
- deny completion if required generated blockers are still open
- show the specific blocking nodes and role requirements in the error and in UI

### 4. Adopt / Apply Flow

When a project wants newer global rules:
- preview global library vs project library
- choose whether to adopt into a new project draft
- human approves the updated draft
- bind the project to the approved project library
- optionally run explicit apply/update on existing managed/generated nodes

Important MVP rule:
- no silent backfill
- no hidden reseed
- updating existing nodes must be an explicit command or wizard step
- only a human or an explicitly allowed orchestrator may run that apply/update flow

Recommended MVP apply scope:
- existing managed/generated nodes first
- do not auto-create missing required children under arbitrary old user-created nodes in the first wave

### 5. Audit Rule

Every meaningful template action should emit durable change history:
- draft created
- draft edited
- draft approved
- project bound
- project adopted from global
- apply/update run on existing nodes

That keeps the feature understandable and reviewable.

## Comments And Collaboration

Templates should not turn comments into private per-role silos.

Recommended default rule:
- comments remain the shared in-scope communication layer
- humans should be able to talk directly to subagents on the relevant node
- agents should be able to hand off to each other in the same work graph
- comment attribution should stay first-class so actor identity and intent remain auditable
- this shared comment layer is a core value-add over external markdown plans
- template contracts should gate workflow mutations and done truth, not discussion

Recommended later-wave direction:
- add optional comment routing or addressing metadata if needed
- add optional limits/configuration for projects that need tighter communication policy
- do not use template-contract ownership rules to hide discussion by default

## TUI / CLI / MCP Flow Direction

Readability matters as much as enforcement.

The TUI and CLI must be idiot-proof for humans.

They should always make these things visible:
- what template library is active
- what generated children will appear
- which actor kind owns each generated child
- who may edit it
- who may complete it
- what it blocks
- why the current node cannot move to done

### TUI MVP

TUI should support:
- browse template libraries
- inspect one template library in plain language
- create or edit project-scoped drafts
- approve drafts as a human
- bind a project to one active template library
- inspect node contract snapshots on generated nodes

Recommended MVP screens:
- `Template Libraries`
  - tabs or filters for `global`, `project`, and `draft`
- `Template Library Detail`
  - plain-language summary of generated work and blockers
- `Template Draft Editor`
  - primary MVP authoring surface
- `Template Approval Review`
  - human review before publish/bind
- `Project Template Binding / Adopt Preview`
  - compare active project library vs selected global library
- `Node Contract Detail`
  - show responsible actor kind, editable/completable actor kinds, and blocker behavior
- `Blocked Completion Detail`
  - show why the node cannot move to done in plain language

Recommended TUI readability rule:
- always render contract summaries in sentences a human can skim
- example:
  - `When a build task is created, generate one qa-pass child.`
  - `Only qa may edit or complete it.`
  - `Human override is allowed.`
  - `This child blocks parent done.`

Recommended TUI implementation rule:
- reuse existing TUI components and interaction contracts wherever possible
- extend the current full-page surfaces, confirm flows, quick-action patterns, input controls, and viewport helpers already used in `internal/tui/model.go`
- do not introduce a second bespoke template-only UI language if an existing pattern already fits
- new template screens should feel like the rest of Tillsyn, not like a sidecar tool bolted on later

Golden requirement:
- any implementation slice that changes TUI output must run `just test-golden`
- if goldens need updating, run `just test-golden-update`
- then manually inspect the changed golden output and screen behavior to confirm it still looks right against current Tillsyn conventions
- do not treat golden updates as mechanical-only approval

### CLI MVP

CLI should support:
- list libraries
- show library
- clone global to project
- preview adopt-from-global
- apply adopted rules to existing managed/generated nodes
- create project draft library
- approve draft library
- bind project to approved library
- show node contract for one node

Recommended CLI command family:
- `till template library list`
- `till template library show --library-id <id>`
- `till template library clone --from-library <id> --project-id <id> --as draft`
- `till template library approve --library-id <id>`
- `till template project bind --project-id <id> --library-id <id>`
- `till template project preview-adopt --project-id <id> --from-library <id>`
- `till template project adopt --project-id <id> --from-library <id> --as draft`
- `till template project apply --project-id <id> --managed-only`
- `till template node contract --node-id <id>`

MVP pushback:
- I do not recommend full matrix-style template authoring through long CLI flag lists in wave 1
- that would be error-prone and hard to read
- use TUI as the primary human authoring surface
- use CLI for inspect, clone, approve, bind, preview, and apply operations

### MCP MVP

MCP should support:
- inspect libraries
- inspect node contract previews
- propose drafts
- propose adopt-from-global previews
- never publish or approve without human approval

Recommended MCP posture:
- MCP may propose draft changes
- MCP may inspect previews and current bindings
- MCP should not be the final approval or publish surface in MVP
- the human should approve in TUI or CLI

## Documentation And Guidance Surfaces

The first implementation wave should not stop at tables and enforcement.

It also needs canonical human and agent guidance surfaces so people can actually use the feature correctly.

### README Requirement

README should eventually include concrete examples for the best-supported workflows:
- create a project and bind one project template library
- create a build task that auto-generates QA follow-up work
- inspect why a node is blocked from done
- approve or bind a draft library
- adopt newer global rules into one project
- explicitly apply/update managed/generated nodes

README should not just describe nouns.
It should show the expected happy-path workflow in short, copyable examples.

### MCP Instruction / Bootstrap Requirement

The existing MCP guidance tools should be expanded as part of the implementation wave:
- `till.get_instructions`
- `till.get_bootstrap_guide`

Those tools should include:
- the recommended Tillsyn workflow order for humans and orchestrators
- short examples of common template/library operations
- best-practice suggestions for keeping external agent setup aligned with Tillsyn
- explicit reminders that impactful policy/template changes still need human approval

### AGENTS / CLAUDE / Skills Guidance Requirement

The guidance surfaces should explicitly help operators align surrounding agent setup with Tillsyn.

That includes recommendations such as:
- update `AGENTS.md` to reflect the project's Tillsyn workflow, approval policy, and validation rules
- update `CLAUDE.md` or equivalent agent-policy docs so interaction rules match Tillsyn's authority model
- create or refine skills that reflect the project's repeated branch/phase/task workflow inside Tillsyn
- keep those files descriptive and human-reviewable instead of hiding process in scattered prompt fragments

Recommended MVP rule:
- Tillsyn should suggest these changes
- but should not silently rewrite those external files
- humans stay the approvers for those policy/doc updates

## Example Contract

Example:
- project template library: `go-service`
- node template: `build-task`
- when a `build-task` is created:
  - generate child `qa-pass`
  - required actor kind: `qa`
  - editable by: `qa`
  - completable by: `qa`
  - orchestrator may complete: optional per rule
  - human may complete: always
  - blocks parent done: yes
  - blocks containing phase done: optional per rule

That is the kind of explicitness the system needs.

## Customizability And MVP Pushback

Your direction makes sense, but MVP needs guardrails.

### What I Think Should Be In MVP

- custom node kinds/types
- SQLite-stored global and project template libraries
- orchestrator-created drafts
- human approval before a draft becomes active
- explicit global-to-project adopt/update flow
- auto-generated child work
- actor-kind edit and complete restrictions
- parent and containing-scope completion blockers
- explicit apply/update commands for existing managed/generated nodes
- clear TUI and CLI explanation surfaces

### What I Would Push Out Of MVP

- fully user-defined actor kinds
- multiple layered libraries active at once per project
- arbitrary file-based source-of-truth syncing
- generic copy/sync across many storage backends
- per-field permission matrices
- full reseeding of existing nodes when a library changes

Recommended MVP simplifications:
- one active template library per project
- fixed small actor-kind set
- global -> project clone flow
- previewed global -> project adopt/update flow
- draft -> approved promotion flow
- generated-node contract snapshots
- explicit apply/update only by human or explicitly allowed orchestrator

That is enough to prove the model without turning it into a giant policy system all at once.

## Phased Implementation Plan

### Phase 0: Lock The Model

Agree on:
- templates as workflow-and-authority contracts
- SQLite as the active source of truth
- one active library per project
- fixed MVP actor kinds

### Phase 1: DB Model

Add:
- template library records
- node template records
- child rule records
- template binding records
- node contract snapshot records
- normalized actor-kind permission tables
- migration path that keeps `kind_catalog.template_json` as a compatibility seam only
- library lineage fields so project copies can trace which global library they came from

### Phase 2: Runtime Enforcement

Add:
- actor-kind-aware edit/complete checks for templated nodes
- parent and containing-scope done gates
- readable blocked-state reasons
- `system` audit actor creation for generated nodes
- explicit apply/update path for existing managed/generated nodes
- transaction and savepoint boundaries so create/adopt/apply flows fail cleanly

### Phase 3: Human Flows

Add:
- TUI browse/approve/bind flows
- CLI browse/clone/approve/bind flows
- previewed adopt/update flows for project-scoped libraries
- MCP draft proposal and preview flows
- node-contract detail and blocked-state explanation surfaces
- README workflow examples for the canonical happy paths
- richer `till.get_instructions` and `till.get_bootstrap_guide` guidance for templates, workflows, and surrounding agent-policy files
- TUI implementation discipline:
  - reuse existing shared components and interaction logic,
  - keep full-page surface behavior aligned with current modes,
  - and clear golden coverage with human review of changed output
- cleanup of old CLI/TUI wording that still implies templates are only kind defaults

### Phase 4: Later Wave

Later:
- library export/import
- better diffing
- reseeding/migration tools
- broader actor-kind flexibility if still needed

## Focused MVP Planning Checklist

Once implementation planning starts, keep it to these slices:

1. SQLite schema and storage model
- template libraries
- node templates
- child rules
- template bindings
- node contract snapshots
- normalized editor/completer actor-kind tables
- foreign-key and transaction boundaries for create/bind/approve flows
- savepoint-friendly update/apply flows so partial failures do not leave half-updated template adoption state

2. Runtime resolution and snapshotting
- resolve the active project template library
- resolve the node template for a requested scope level + node kind
- generate child rules into concrete child nodes
- persist the resolved rule as a node contract snapshot on every generated node
- record audit actor identity separately from responsible actor kind

3. Auth and completion enforcement
- keep current scope checks as the coarse boundary
- add actor-kind and node-kind checks from the stored node contract
- add parent and containing-scope done gates using the stored contract
- keep human override-complete allowed
- keep orchestrator override-complete opt-in per rule and default off

4. Human-readable TUI and CLI flows
- library browse/show
- project bind flow
- draft create/edit/approve flow
- global-to-project adopt preview flow
- explicit apply/update flow for managed/generated nodes
- node blocked-state explanation flow
- node contract inspection flow
- TUI work must reuse existing shared components/surfaces instead of inventing a parallel template UI stack
- TUI work must pass `just test-golden`, and changed golden output must be manually reviewed for fit and readability

5. Documentation and MCP guidance
- README examples for the highest-frequency template workflows
- `till.get_instructions` updates that explain the best-supported workflow order and related operator guidance
- `till.get_bootstrap_guide` updates for first-run template/library setup
- explicit suggestions for aligning skills and `AGENTS.md` / `CLAUDE.md`-style files with Tillsyn policy

6. Migration and compatibility
- do not silently mutate existing nodes
- keep legacy kind-template behavior working during transition
- make new enforcement depend on stored node contracts, not live template lookups only
- plan a single explicit apply/update command path for existing managed/generated nodes, not hidden backfill
- move project creation onto template-library resolution before rewriting snapshot import/export, so new live data uses the new model before the transport shape is updated around it
- then update snapshot import/export to carry template libraries, bindings, and node-contract snapshots without freezing the old split model in place

## Concrete Cleanup Map

When implementation starts, these are the most obvious old seams that should be removed or clearly quarantined.

### Domain / App Seams

- `internal/domain/kind.go`
  - `KindTemplate`, `KindTemplateChildSpec`, and related template-file section fields currently bundle node-kind registry and template behavior together
  - new work should move live contract behavior into template-library tables and leave `kind` focused on node-kind registry/schema concerns
  - `AgentsFileSections` and `ClaudeFileSections` currently look like legacy template-sidecar fields and should either gain a clear managed-docs home or be removed from the live template path
- `internal/app/kind_capability.go`
  - `mergeTaskMetadataWithKindTemplate`
  - `applyKindTemplateSystemActions`
  - `applyProjectKindTemplateSystemActions`
  - `validateKindTemplateExpansion`
  - these are the main legacy create-time template engine paths and should be replaced by the library resolver + snapshot pipeline
  - `normalizeTaskScopeForKind` is also a cleanup seam because it still infers structural level from specific kind ids like `branch`, `phase`, and `subtask`
- `internal/app/service.go`
  - `createTaskWithTemplates` currently calls legacy kind-template expansion directly
  - the new path should resolve project library -> node template -> child rules -> contract snapshots
- `internal/app/mutation_guard.go`
  - the current internal-template bypass should not remain a hidden authority seam once generated-node contracts exist
- `internal/domain/project.go`
  - `MergeProjectMetadata` should remain a merge helper, but not the canonical home of workflow-template policy
- `internal/domain/workitem.go`
  - `MergeTaskMetadata` and `MergeCompletionContract` should remain bounded merge helpers, but should no longer be the whole template system
- `internal/domain/task.go`
  - `CompletionCriteriaUnmet` currently only sees checklist/children-done state
  - it will need to work with stored contract snapshots and actor-kind-aware blockers

### Storage / Transport Seams

- `internal/adapters/storage/sqlite/repo.go`
  - `kind_catalog.template_json` is the main legacy storage seam
  - keep it readable during transition, but stop treating it as the active authored template system
  - new template tables should become canonical
- `internal/app/snapshot.go`
  - snapshot import/export currently carries kind-template blobs and project allowlists
  - the snapshot format will need an explicit compatibility update so template libraries, bindings, and node contract snapshots are not silently lost
- `cmd/till/main.go`
  - `--template-json` on `till kind upsert` is a legacy authoring surface that should not remain the primary template path
  - `kind upsert` should become node-kind registry management only
- `internal/adapters/server/common/mcp_surface.go`
  - `UpsertKindDefinitionRequest` currently carries template fields through the kind path
  - the transport should grow separate template-library surfaces instead of overloading kind upsert forever
- `internal/adapters/server/common/app_service_adapter_mcp.go`
  - bootstrap and help text that still advertise template-driven child/checklist auto-actions through the kind path will need to move to the new library wording
- `internal/adapters/server/mcpapi/extended_tools.go`
  - current `till.upsert_kind_definition` and `till.list_kind_definitions` should remain for node-kind registry work
  - separate template-library MCP tools should carry the new contract model
- `internal/adapters/server/mcpapi/instructions_tool.go`
  - instruction guidance already talks about `AGENTS.md`, `CLAUDE.md`, and recommended doc usage
  - it should gain template-workflow examples and explicit Tillsyn-alignment suggestions instead of staying generic

### Tests / Docs Seams

- `internal/domain/kind_capability_test.go`
  - current expectations around `KindDefinition.Template` will need to narrow to kind-registry behavior or move to template-library tests
- `internal/app/kind_capability_test.go`
  - these tests currently prove legacy checklist merge and auto-child creation through kind templates
  - they should be replaced or rewritten around template libraries and node contract snapshots
- `cmd/till/main_test.go`
  - help output and command coverage that mention `--template-json` will need replacement
- `internal/tui/model.go`
  - current form/search/rendering paths still infer structural labels from kind ids in several places
  - those read/write seams should be reviewed when `kind` and `level` are separated more cleanly
  - template flows should be added through the existing mode/surface helpers instead of building disconnected one-off widgets
- `README.md`
  - once implementation lands, remove transitional wording that still describes active behavior as kind-template-backed defaults without the new contract path beside it
- `PLAN.md`
  - implementation slices should explicitly record when the old kind-template path is removed or quarantined

The point of this map is:
- no orphaned `kind template` authoring path
- no duplicate old/new template engines living side by side longer than necessary
- no stale help text that teaches the wrong model

### Current Level vs Kind Conflation To Watch

These current seams are especially likely to create subtle bugs if they survive the migration:
- structural level inference from specific kind ids such as `branch`, `phase`, and `subtask`
- TUI creation defaults that pick both kind and level from current board position
- search/filter/read models that still treat `phase` as both a level word and a kind word
- MCP surfaces that expose parallel `levels` and `kinds` filters while built-in values still overlap heavily

Implementation rule:
- when the new contract lands, audit every place that infers level from kind or kind from level
- preserve only explicit level fields plus explicit node-kind fields

## Implementation Hygiene Requirement

When the implementation branch starts, treat DRY cleanup as part of the plan, not follow-up polish.

Required planning rule:
- every implementation slice must explicitly identify the old code paths, commands, UI copy, tests, and data assumptions it replaces
- and must either remove them in the same slice or mark them as temporary compatibility seams with a named cleanup follow-up

In practice that means:
- update or delete superseded code blocks instead of layering duplicates beside them
- avoid parallel old/new template resolution paths unless the compatibility seam is deliberate and documented
- avoid orphaned CLI commands, TUI copy, schema fields, or tests that still describe templates as simple kind defaults after the contract model lands
- include code-search cleanup checks in each implementation slice so stale wording and dead branches are caught deliberately

## Locked Consensus

These points are now treated as agreed:
- templates are workflow-and-authority contracts first
- SQLite is the active single source of truth
- one active template library per project in MVP
- global libraries act as clone/adopt sources; project libraries can diverge locally without affecting other projects
- human override-complete is always allowed
- orchestrator complete on builder/QA blockers defaults to off and is opt-in per rule
- `kind` becomes the node-kind registry while scope level remains separate
- `system` remains an internal audit/provenance actor, not the workflow owner kind for generated work

## README / PLAN Handoff Note

Once this model is locked, README and this branch's `PLAN.md` should stop describing templates primarily as "kind templates" or "metadata/checklist defaults".

They should instead describe the clarified contract:
- `kind` as structure
- `template` as generated work graph plus role contract
- auth as actor-kind plus scope plus node-kind enforcement
- done gating as required generated work completed by an allowed actor or a human

I do not recommend editing README and `PLAN.md` further until this updated model is approved, because that wording should be folded in once, cleanly, instead of churned across multiple partial drafts.

## Memo Summary

- Templates are workflow-and-authority contracts first, scaffolding second.
- Scope level, node kind, and actor kind must be treated as separate dimensions.
- Level-only auth is not enough; templated nodes need actor-kind and node-kind checks.
- SQLite should be the active single source of truth for templates, bindings, and node contract snapshots.
- MVP should support fixed actor kinds, custom node kinds, one active template library per project, human approval, generated child blockers, and idiot-proof TUI/CLI visibility.

## Open Questions

1. Should explicit apply/update on existing nodes target only:
   - managed/generated nodes,
   - or also optionally create missing required children under existing user-created parents?
   My recommendation is: managed/generated nodes first, with missing-child creation as an explicit opt-in later.
2. Should project adopt/update from a global library replace the whole project library, or patch selected node templates/rules only?
   My recommendation is: whole-library preview plus explicit project-side confirmation in MVP.
3. Should wave 1 support full CLI authoring of template rules, or keep TUI as the primary authoring surface?
   My recommendation is: keep TUI primary, CLI operational.

## Recommended Next Step

Treat this planning pass as the lock for MVP direction, then start the later implementation branch with this exact first slice:
- add relational template-library tables beside the current kind catalog
- build one compatibility-first resolver that prefers project template libraries and falls back to legacy `kind_catalog.template_json`
- store node contract snapshots on generated nodes
- land TUI inspect/approve/bind flows plus CLI inspect/clone/approve/bind/apply flows
- remove or quarantine the legacy template authoring seams called out in the cleanup map
