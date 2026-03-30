# Templating System Design Memo

Status: design proposal for `agent/templating-design`

Date: 2026-03-29

Scope:
- design only
- no implementation in this lane
- intended as a handoff artifact for a later implementation branch

## Purpose

Define what "templating" should mean in Tillsyn without collapsing it into the current `kind` defaults model or overreaching into a full workflow engine.

The memo is grounded in the current codebase:
- `KindDefinition.Template` already drives create-time project metadata defaults, task metadata defaults, completion checklist defaults, and auto-created child work.
- project and task creation already validate `kind` and `kind_payload` at create time.
- CLI and MCP expose `kind` management; TUI mostly does not.
- `AgentsFileSections` and `ClaudeFileSections` exist in the domain model but are not currently applied or surfaced.
- service-level `StateTemplates` currently control board columns and should stay separate from node templating in the first wave.

Source note:
- Context7 did not resolve the Go stdlib template package cleanly for this topic, so I used local `go doc text/template` as a bounded fallback to confirm the trusted-author, plain-text-generation model for parameterized markdown/text payloads.

## Current-State Read

The current system already has two different concepts mixed together under `kind`:

1. Taxonomy and validation
- what scopes a kind applies to
- allowed parent scopes
- `kind_payload` schema validation
- project allowlists

2. Create-time scaffolding
- default project metadata
- default task metadata
- completion checklist defaults
- auto-created children

That is workable for the current MVP slice, but it will become confusing if Tillsyn expands templating to:
- project scaffolds
- richer branch/phase/task/subtask defaults
- standards or guidance payloads
- managed docs sections
- later reseeding and audit flows

The clean split is:
- `kind` means classification, constraints, and schema.
- `template` means deterministic scaffolding and managed generated content.

The first implementation wave should preserve compatibility by building on the current `KindTemplate` behavior, but the product language and future API should stop treating all templating as "kind templates".

Current user-facing creation flow gaps:
- TUI project creation captures metadata and root path, but not project `kind` or template choice.
- TUI task creation infers child kind/scope mostly from focused board scope, not from a template-selection flow.
- CLI project creation supports `--kind`, metadata flags, and standards markdown, but not a higher-level template-set concept.
- MCP `create_project` and `create_task` support `kind` and metadata, but not template preview or binding semantics.
- CLI and MCP expose `kind` catalog management today; TUI does not.

## What Templating Should Mean In Tillsyn

Templating should mean:

- deterministic, auditable create-time scaffolding
- optional explicit later reapply or migration
- reviewable generated structure and text
- no hidden continuous mutation after creation

It should not mean:

- a general-purpose workflow engine
- arbitrary code execution
- silent retroactive rewrites when definitions change
- a second permissions system

### 1. Project Templates

Project templates should define the initial shape of a project:
- default project `kind`
- default project metadata
- standards or guidance markdown
- allowed kind presets for the project
- root child structure, such as the default branch or initial phases
- default node-template bindings for later branch/phase/task/subtask creation

Project templates are best modeled as a `template set`, not as a single leaf template. The project is the binding point for a set of related node templates.

### 2. Branch / Phase / Task / Subtask Templates

Node templates should define create-time scaffolding for one scope and kind combination:
- title and description defaults
- default labels
- default task metadata
- completion criteria/checklists/evidence expectations
- child skeletons
- managed guidance payloads

These are not replacements for `kind`. They are the scaffold chosen after the kind and scope are known.

### 3. Kind Defaults

Kind defaults should remain narrow and structural:
- applies-to constraints
- allowed parent scopes
- schema validation for `kind_payload`
- possibly a tiny amount of kind-native defaulting for schema-shaped payload fields

Kind defaults should not become the long-term home for:
- project standards markdown
- product-specific docs payloads
- large child hierarchies
- template inheritance policy
- reseeding policy

In other words, kind answers "what is this node allowed to be?", while template answers "what should this node look like when created in this project?"

### 4. Standards / Docs Payloads

Standards and guidance payloads should be treated as first-class managed template outputs:
- project standards markdown
- agent-facing guidance
- later CLAUDE/AGENTS-style managed sections

The current `AgentsFileSections` and `ClaudeFileSections` names are too product-specific for the long term. The cleaner noun is `managed sections` or `managed documents`.

In the first implementation wave:
- support templated markdown payloads attached to project or node records
- do not write files into the repo automatically
- do not implement bidirectional sync with `AGENTS.md` or `CLAUDE.md`

### 5. Checklists / Contracts

Completion contracts should be a templating concern, but a narrow one:
- create-time default checklist items
- create-time default completion criteria
- create-time evidence expectations
- create-time tightening of policies such as `require_children_done`

This is already close to what the current code does. The main change is conceptual clarity and provenance, not a new behavior model.

## Static Defaults Vs Parameterized Templates

The split should be conservative.

### Static Defaults

Use static defaults for fields where determinism and merge behavior matter more than expressiveness:
- `kind` and scope defaults
- schema-shaped `kind_payload` defaults
- project metadata defaults like icon, color, tags, homepage
- task metadata defaults like validation plan, risk notes, command snippets
- allowed kind presets
- completion policy booleans
- fixed child topology

These should merge structurally, not through text rendering.

### Parameterized Templates

Use parameterized templates only for text-like payloads:
- project standards markdown
- node titles
- node descriptions
- checklist text
- completion notes
- managed guidance sections

Recommended first-wave rendering model:
- plain text / markdown only
- no HTML templating
- trusted authors only
- minimal deterministic context object
- minimal function surface

`text/template` is appropriate for this class of output because the outputs are plain text or markdown and template authors are trusted. It is not appropriate as an untrusted input language.

Recommended first-wave render context:

```json
{
  "project": {
    "id": "p1",
    "name": "Go API",
    "slug": "go-api",
    "kind": "go-service",
    "metadata": {
      "owner": "platform",
      "tags": ["go", "api"]
    }
  },
  "parent": {
    "id": "branch-1",
    "title": "Main Branch",
    "kind": "branch",
    "scope": "branch"
  },
  "node": {
    "kind": "implementation",
    "scope": "task"
  },
  "params": {
    "primary_check": "just check",
    "language": "go"
  }
}
```

Explicitly out of scope for first-wave parameterization:
- conditional child counts
- loops that create variable numbers of nodes
- arbitrary custom functions
- live repository inspection from template execution
- auth-policy decisions driven by rendered text

## How Templates Should Interact With Kinds And Project Metadata

Recommended rule:

- kind owns structural validity
- template owns scaffold resolution
- project metadata supplies context for template rendering

### Resolution Order

For new nodes, the resolver should apply this order:

1. Kind structural defaults and validation
2. Project template-set defaults for the requested scope and kind
3. Optional parent-derived defaults when explicitly configured
4. Caller-supplied fields and metadata
5. System-only fallbacks, such as current user owner fill-in

Scalar override rule:
- caller values win over template defaults

Safety-tightening rule:
- completion policy may tighten but should not silently relax

Structured payload rule:
- `kind_payload` should stay a schema-validated merge target, not a generic text template bag

### Project Metadata Interaction

Project metadata should have two roles:

1. Stored project state
- owner
- tags
- icon
- color
- homepage
- `kind_payload`
- standards markdown

2. Template render context
- values that parameterize later branch/phase/task/subtask templates

That makes project metadata the stable source for project-specific substitution values without inventing a separate ad hoc variable store for the first wave.

### Allowed Kinds Interaction

Project templates should be able to seed the project allowlist, but they should not bypass it.

Recommended behavior:
- binding a project template can set a default allowed-kind preset
- actual creation still validates against the resolved allowlist
- explicit project allowlist edits remain authoritative

## Where Templates Should Live

The long-term answer should be a mixed model.

### DB Only

Pros:
- fast runtime lookup
- easy to query from TUI/CLI/MCP
- easy to snapshot/export

Cons:
- weak reviewability
- poor collaboration ergonomics for authored template changes
- difficult to tie template revisions to repo commits or code review

DB only is not enough.

### Config Only

Pros:
- easy machine-local defaults

Cons:
- wrong scope for project collaboration
- poor auditability
- not shareable enough

Config should not be the primary home for template bodies.

### Repo Files Only

Pros:
- versioned with project work
- reviewable in git
- good for human and agent collaboration

Cons:
- awkward runtime mutation flow
- expensive or repetitive parsing unless imported
- harder to attach provenance to existing records

Repo files alone are also not enough.

### Recommended Mixed Model

Use:
- repo files as authored template definitions
- DB as imported runtime records and binding/provenance state
- config only for discovery paths and operator defaults

Recommended responsibilities:

- Repo files:
  - canonical authored template definitions
  - versioned and reviewable
  - best for team or repo-specific packs

- DB:
  - imported template revisions
  - active project bindings
  - node-level provenance
  - export/import survival

- Config:
  - template search roots
  - optional default template set id for new repos or operators

### Recommended File Shape

For the first wave, prefer JSON template files over inventing a new authoring grammar:
- it matches current CLI/MCP JSON payload shapes
- it keeps import/export straightforward
- it avoids adding YAML-specific complexity

Human ergonomics are worse than TOML or YAML, but the tradeoff is acceptable for wave 1 because the first authoring surface should still be CLI/repo-review-heavy rather than TUI-heavy.

Recommended repo path:
- `templates/*.json`

## Recommended Nouns And API Shapes

### Product Nouns

- `kind`: taxonomy, schema, scope constraints
- `template set`: project-level collection of related template definitions
- `node template`: one create-time scaffold for a specific scope and kind
- `template binding`: a project or node pointing at a template revision plus parameter values
- `managed section`: generated markdown or guidance payload with provenance
- `template preview`: the resolved output before persistence

### Suggested Go Shapes

```go
type TemplateSet struct {
	ID          string
	Name        string
	Description string
	Revision    string
	Source      TemplateSource
	Project     ProjectTemplateSpec
	Nodes       []NodeTemplate
}

type ProjectTemplateSpec struct {
	DefaultKind     domain.KindID
	Metadata        domain.ProjectMetadata
	AllowedKindIDs  []domain.KindID
	ManagedSections []ManagedSectionTemplate
	RootChildren    []ChildTemplateSpec
}

type NodeTemplate struct {
	ID               string
	AppliesTo        domain.KindAppliesTo
	KindID           domain.KindID
	DisplayName      string
	ParametersSchema string
	TitleTemplate    string
	DescriptionTmpl  string
	MetadataDefaults domain.TaskMetadata
	ManagedSections  []ManagedSectionTemplate
	Completion       domain.CompletionContract
	Children         []ChildTemplateSpec
}

type ManagedSectionTemplate struct {
	ID              string
	Target          string
	Heading         string
	BodyTemplate    string
	RequireUserSync bool
}

type TemplateBinding struct {
	TemplateSetID string
	Revision      string
	Parameters    map[string]string
	BoundBy       string
	BoundAt       time.Time
}
```

### Suggested CLI / MCP Shapes

Prefer a new `template` namespace instead of further overloading `kind`.

CLI:
- `till template list`
- `till template show --id`
- `till template validate --path templates/go-service.json`
- `till template import --path templates/go-service.json`
- `till template bind project --project-id p1 --template-set go-service/v1`
- `till project create --template-set go-service/v1 --name "Go API"`

MCP:
- `till.list_templates`
- `till.get_template`
- `till.import_template`
- `till.bind_project_template`
- `till.preview_template_resolution`

Mutation calls should later accept:
- `template_set_id`
- `template_id`
- `template_params`

But in wave 1, project creation only needs `template_set_id` and optional parameter values.

## Example Template Objects

### Example Template Set

```json
{
  "id": "go-service/v1",
  "name": "Go Service",
  "description": "Starter scaffold for a local Go service project",
  "revision": "git:75aa5c4",
  "project": {
    "default_kind": "go-service",
    "metadata": {
      "owner": "platform",
      "tags": ["go", "service"],
      "standards_markdown": "Primary validation path: `{{index .params \"primary_check\"}}`.\nKeep project standards current in Tillsyn."
    },
    "allowed_kind_ids": ["project", "branch", "phase", "implementation", "qa-check", "note"],
    "root_children": [
      {
        "kind": "branch",
        "applies_to": "branch",
        "title": "Main Branch",
        "description": "Default implementation branch for {{.project.name}}",
        "labels": ["templated"]
      }
    ]
  },
  "nodes": [
    {
      "id": "implementation-task",
      "applies_to": "task",
      "kind_id": "implementation",
      "display_name": "Implementation Task",
      "title_template": "{{.project.name}} implementation",
      "description_template": "Ship the scoped change and validate with {{index .params \"primary_check\"}}.",
      "metadata_defaults": {
        "validation_plan": "Run {{index .params \"primary_check\"}} before handoff.",
        "acceptance_criteria": "Behavior is verified and documented."
      },
      "completion": {
        "completion_checklist": [
          {"id": "tests-green", "text": "Primary validation is green", "done": false}
        ],
        "policy": {
          "require_children_done": true
        }
      },
      "children": [
        {
          "kind": "qa-check",
          "applies_to": "subtask",
          "title": "QA Check",
          "description": "Verify the implementation before completion."
        }
      ]
    }
  ]
}
```

### Example Project Binding

```json
{
  "project_id": "p-go-api",
  "template_set_id": "go-service/v1",
  "revision": "git:75aa5c4",
  "parameters": {
    "primary_check": "just check"
  },
  "bound_by": "evan",
  "bound_at": "2026-03-29T16:10:00Z"
}
```

## CLI / TUI / MCP Surfacing

### CLI

CLI should be the first-class authoring and inspection surface in wave 1.

Recommended first-wave CLI flow:

1. Import or inspect template sets
2. Bind one template set to a project at create time
3. Show the resolved preview before applying

Example:

```bash
till template import --path templates/go-service.json
till project create --name "Go API" --template-set go-service/v1 --template-param primary_check="just check"
```

CLI should also expose read-only provenance:
- which template set a project uses
- which node template generated a node
- which revision and parameters were applied

### TUI

TUI should be consumer-first in wave 1, not authoring-first.

Recommended first-wave TUI surfaces:
- project create picker includes template-set choice
- project create review panel summarizes:
  - resulting project kind
  - metadata defaults
  - root children to be created
  - standards/guidance sections to attach
- task or branch create flow shows a compact "template preview" summary if a non-default node template will apply
- project info and task info show template provenance badges

Avoid in wave 1:
- rich template authoring screens
- full reseed diff UIs
- inline editing of template definitions

### MCP

MCP should support discovery, preview, and bounded application.

Recommended first-wave MCP capabilities:
- inspect template inventory
- get one template definition
- preview resolution for a proposed project or node creation
- create project with a template-set binding

Defer direct broad template-authoring mutation over MCP until the repo-file and audit model is clearer. It is better for wave 1 if agents can:
- discover templates
- propose one
- bind or apply an approved existing template

than if they can freely rewrite template libraries through MCP.

## UX Flows

### Flow 1: Create A Project From A Template Set

1. User or agent chooses `go-service/v1`
2. System previews:
   - resulting project kind
   - standards markdown
   - allowed kind preset
   - generated root branch
3. User confirms
4. System creates:
   - project
   - template binding
   - generated root children
   - change-event audit rows

### Flow 2: Create A Task Under A Scoped Parent

1. User focuses a branch or phase and starts create-task
2. System resolves:
   - parent scope
   - chosen kind
   - project template-set binding
   - matching node template
3. Form opens with rendered defaults and a short preview note
4. User edits any explicit fields
5. System persists:
   - final node values
   - provenance for the resolved template revision
   - generated child subtasks if applicable

### Flow 3: Inspect Provenance During Review

1. User opens task info
2. System shows:
   - kind
   - template origin
   - revision
   - generated checklist items
   - whether any managed section was detached or edited
3. Reviewer can tell the difference between:
   - user-authored content
   - template-generated content
   - manually detached overrides

## Collaboration Flows, Ownership, And Auditability

Templating should improve collaboration only if provenance is explicit.

Recommended rules:

- template definition changes are reviewed like code or docs
- template application is a separate auditable event
- existing nodes do not silently change when template files change
- generated children and managed sections must carry provenance
- impactful template binding changes should be user-reviewable

### Ownership

Recommended ownership split:
- repo template authors own template definitions
- project owners own template-set binding choices
- agents may propose template use or apply approved templates within allowed workflows
- node ownership stays with the actor who created or updated the node, not the template author

### Audit Fields

Persist enough to answer:
- what template generated this?
- which revision?
- with which parameters?
- who bound or applied it?
- when?
- which outputs remain managed vs detached?

At minimum, persist:
- template set id
- node template id
- revision
- parameters hash or explicit parameter map
- actor id / actor type
- applied timestamp
- generated child ids

### Change Feed

Template-related change events should be first-class:
- template imported
- project bound to template set
- node created from template
- managed section detached
- later reapply requested or previewed

This is important for both user trust and agent recovery.

## Migration And Compatibility Strategy

The first implementation wave should be compatibility-first.

### Recommended Approach

1. Keep current `KindDefinition.Template` behavior working
- do not break the existing create-time defaults path
- do not force an immediate data rewrite

2. Add a template resolution layer above it
- project template sets resolve to node-template defaults
- unresolved fields can still fall back to existing `KindTemplate` values

3. Preserve current `kind` CLI and MCP surfaces
- `kind upsert` remains valid
- template-bearing `kind` fields become legacy-compatible inputs, not the future primary API

4. Add explicit project template binding
- new projects may opt into a template set
- old projects continue to work with current kind defaults only

5. Do not retroactively mutate existing nodes in wave 1
- only new project and node creation uses the new template resolver

### Snapshot Compatibility

Snapshot export/import should eventually include:
- imported template definitions
- project template bindings
- node provenance

During transition:
- continue exporting current kind-template data
- add new template records alongside it
- prefer additive compatibility over immediate field removal

### Deprecation Direction

Long term:
- `kind.template` should shrink toward structural defaults or disappear
- `template set` and `node template` should become the primary authored scaffold model

But that is not a first-wave requirement.

## Explicitly Out Of Scope For The First Implementation Wave

Keep first-wave scope tight.

Out of scope:
- full template authoring UI in the TUI
- retroactive reseeding or bulk migration of existing nodes
- file-writing sync to `AGENTS.md`, `CLAUDE.md`, or repo docs
- arbitrary custom template functions
- untrusted template execution
- dynamic child multiplicity based on repo inspection
- agent-role policy templating
- workflow wait-state templating
- board-column templating beyond the existing `StateTemplates` behavior
- fine-grained merge conflict UI for reapply or detach flows
- remote template registries
- multi-repo template distribution semantics

## Risks And Tradeoffs

### 1. Kind vs Template Confusion

Risk:
- users and code paths may not know when to choose `kind` vs `template`

Mitigation:
- keep `kind` as classification and validation
- keep `template` as scaffold and generated content
- add provenance and preview surfaces

### 2. Mixed Storage Complexity

Risk:
- repo files plus DB records can drift

Mitigation:
- import template revisions as immutable records
- bind projects to explicit revisions
- require explicit re-import or rebind when repo files change

### 3. Parameterization Becoming A Scripting Language

Risk:
- template logic grows opaque and hard to audit

Mitigation:
- limit parameterization to string fields
- use a small context surface
- avoid custom functions in wave 1

### 4. Managed Docs Synchronization Ambiguity

Risk:
- users will not trust templates if edits get overwritten

Mitigation:
- do not implement file sync in wave 1
- treat generated markdown as stored managed content with explicit detach semantics later

### 5. Overbuilding The TUI Too Early

Risk:
- template authoring UI could consume the whole implementation wave

Mitigation:
- CLI and MCP first for authoring and inspection
- TUI first only for choosing and viewing template results

## Phased Implementation Plan

### Phase 0: Lock Design Terms

Goals:
- agree on nouns
- agree on first-wave scope
- agree that kind and template are separate concepts

Deliverables:
- this memo
- acceptance notes for the next branch

### Phase 1: Resolver And Project Template Binding

Goals:
- add `template set` and `template binding` concepts
- keep existing `kind` behavior intact
- resolve project and node defaults through a compatibility-first resolver

Suggested implementation slice:
- domain types for template set, node template, template binding
- repo support for imported template sets and project bindings
- project create path accepts `template_set_id` and parameters
- create-time project and task flows attach provenance
- export/import carries new records

### Phase 2: CLI / MCP Consumption Surfaces

Goals:
- make template discovery and preview practical
- let projects bind existing templates explicitly

Suggested implementation slice:
- `template list/show/import/validate`
- `bind_project_template`
- `preview_template_resolution`
- project create template flags and MCP args

### Phase 3: TUI Consumption Surfaces

Goals:
- let normal users pick a project template set
- expose provenance in project/task info surfaces

Suggested implementation slice:
- project create template picker
- compact preview summary
- provenance badges and detail panels

### Phase 4: Later Wave, Explicitly Deferred

Goals:
- reseeding
- detachment / managed section lifecycle
- richer authoring UI
- policy integration

This later wave should not start until wave 1 usage validates the data model.

## Recommendation

Recommended product direction:

1. Define templating as deterministic create-time scaffolding plus explicit later reapply, not hidden automation.
2. Keep `kind` for classification, schema, and placement rules.
3. Introduce `template set` as the project-level concept and `node template` as the scope/kind-specific concept.
4. Use a mixed storage model:
   - repo files for authored definitions
   - DB for imported revisions, bindings, and provenance
   - config only for discovery paths and operator defaults
5. Keep parameterization narrow and text-only in wave 1.
6. Keep the first implementation wave compatibility-first by layering on top of current `KindTemplate` behavior instead of rewriting it all at once.
7. Start with CLI and MCP authoring/inspection; keep TUI mostly to selection and provenance.

## Memo Summary

- Templating should mean deterministic, auditable scaffolding at project and node creation time.
- `kind` and `template` should be separated conceptually now, even if the first implementation wave reuses current `KindTemplate` storage internally.
- The right first-wave storage model is mixed: repo-file definitions plus DB bindings and provenance.
- Parameterized rendering should stay narrow, trusted, and markdown/text-only.
- The first wave should focus on resolver, project template binding, preview, provenance, and CLI/MCP support, while explicitly deferring reseeding, file sync, and rich TUI authoring.

## Open Questions

1. Should first-wave project template files be imported explicitly, or should Tillsyn auto-discover `templates/*.json` in the active repo root?
2. Should `project.Metadata.StandardsMarkdown` remain the storage field for generated standards content, or should standards move into a more generic managed-sections store immediately?
3. Is it acceptable for first-wave node creation to resolve templates implicitly from project binding plus kind, or does the user want explicit per-create template selection from day one?
4. Should template-set binding also seed the project `allowed_kinds` closure automatically, or should that remain a separate explicit step for clarity?
5. Does the user want template authoring over MCP in wave 1, or only discovery/preview/binding of repo-authored templates?

## Recommended Next Step

Create a follow-on implementation branch that:
- introduces the template-set and binding domain types
- adds a compatibility-first resolver above the current `KindTemplate` path
- exposes CLI and MCP preview/bind flows before attempting any broader TUI or reseed work
