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

## Recommended Nouns And Data Shapes

### Product Nouns

- `template library`: a named collection of template rules
- `node template`: the rule for creating one node kind at one scope level
- `child rule`: one auto-generated child contract
- `template binding`: which project uses which template library
- `node contract snapshot`: the resolved rule persisted on a created node
- `actor kind`: the authority category used by auth and completion rules

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

That gives a sane MVP path:
- keep current level-based auth as the coarse scope check
- add node-kind and actor-kind checks for templated/generated work

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

### CLI MVP

CLI should support:
- list libraries
- show library
- clone global to project
- create project draft library
- approve draft library
- bind project to approved library
- show node contract for one node

### MCP MVP

MCP should support:
- inspect libraries
- inspect node contract previews
- propose drafts
- never publish or approve without human approval

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
- auto-generated child work
- actor-kind edit and complete restrictions
- parent and containing-scope completion blockers
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
- draft -> approved promotion flow
- generated-node contract snapshots

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

### Phase 2: Runtime Enforcement

Add:
- actor-kind-aware edit/complete checks for templated nodes
- parent and containing-scope done gates
- readable blocked-state reasons

### Phase 3: Human Flows

Add:
- TUI browse/approve/bind flows
- CLI browse/clone/approve/bind flows
- MCP draft proposal and preview flows

### Phase 4: Later Wave

Later:
- library export/import
- better diffing
- reseeding/migration tools
- broader actor-kind flexibility if still needed

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

1. MVP actor kinds: should the fixed initial set be `human`, `orchestrator`, `builder`, `qa`, and `research`, or should it be even smaller?
2. Should humans always retain unconditional override-complete power on generated blockers? My recommendation is yes.
3. Should orchestrator completion on builder/qa-generated blockers default to off and require explicit per-rule opt-in? My recommendation is yes.
4. Should a project bind exactly one active template library in MVP, with global libraries only serving as clone sources? My recommendation is yes.
5. Should current `kind` evolve into the canonical node-kind registry while scope level remains a separate field? My recommendation is yes.
6. Should draft libraries be project-local only in MVP, even if an orchestrator proposed them from a global pattern? My recommendation is yes.

## Recommended Next Step

Lock this corrected model first, then do a small follow-on planning pass that defines:
- the exact MVP actor-kind set
- the SQLite table layout
- the project/global/draft clone-and-approve flow
- the minimum TUI and CLI screens needed to make the rules readable to humans
