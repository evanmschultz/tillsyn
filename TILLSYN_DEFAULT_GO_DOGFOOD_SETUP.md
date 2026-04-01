# TILLSYN Default-Go Dogfood Setup

This document locks the agreed MCP-only setup for the first real Go template and the first real dogfood project.

Use this as the handoff spec for a fresh agent session once the `till_*` MCP tools are available in that session.

## Goal

Create one reusable Go project template library and use it to create the first real dogfood project:

- template library: `default-go`
- dogfood project: `TILLSYN`
- test-only project: `TEST_PROJECT`

The runtime setup should be performed through Tillsyn MCP tools, not through direct CLI mutation commands.

## Locked Decisions

### Naming

- Template library id: `default-go`
- Project kind: `go-project`
- Phase kind: `implementation-phase`
- Task kind: `build-task`
- Subtask kind: `qa-check`
- Dogfood project names should use all caps.

### Ownership / Actor-Kind Rules

- `builder`, `qa`, `research`, `orchestrator`, and `human` are actor kinds.
- Template ownership is by actor kind, not by a pre-known specific agent principal.
- Both QA subtasks should use actor kind `qa`.
- The stricter rule "two different QA principals must complete the two QA passes" is deferred to a later policy wave.
- Policy direction for the unified `plan_item` surface:
  - the responsible actor kind should be able to move its own work through ordinary active states such as `todo -> progress -> done` when the stored node contract allows it;
  - humans should remain allowed to perform those transitions;
  - `till.plan_item` should own same-noun reads as well as writes instead of keeping separate default read tools for the same noun.
  - `till.plan_item(operation=move_state)` is the preferred contract-aware state-transition shape instead of separate semantic `complete` / `reopen` verbs, so policy-gated forward and backward movement share one family shape.
  - delete/hard cleanup should remain human-only, and archive/final terminal transitions should stay more restricted than ordinary active-state moves.
  - comments should remain a separate append-only coordination family rather than being folded into `till.plan_item`.
  - parallel/sibling comments are expected when the approved scope already covers both nodes; otherwise escalation should use handoffs/attention instead of broadening scope implicitly.

### Auth / Scope Model

- This behavior is expected and desired:
  - global agent auth is for global catalog admin, template-library admin, and project creation/binding;
  - project-scoped agent auth is for guarded mutations inside that project;
  - branch/phase/task-scoped auth should be used when the runtime can prove the narrower path.
- Default MCP surface note:
  - `till.auth_request` is the preferred auth-request family for `create|list|get|claim|cancel`;
  - `till.project` is the preferred project-root family for `list|create|update|bind_template|get_template_binding|set_allowed_kinds|list_allowed_kinds|list_change_events|get_dependency_rollup`;
  - `till.plan_item` is the preferred plan-item read/mutation family for `get|list|search|create|update|move|move_state|delete|restore|reparent`;
  - `till.kind`, `till.template`, and `till.embeddings` are the preferred family tools for catalog/template/embedding lifecycle work;
  - `till.comment` is the preferred append-only coordination family for `create|list` and should not be folded into `plan_item`;
  - the older flat project/template/kind aliases are compatibility-only where still exposed and should not be treated as the preferred default surface.
- Agents and operators should not treat the global-to-project auth split as a bug.
- After creating a project with global auth, the next normal step is to claim or reuse a project-scoped session before creating guarded in-project work.
- Guarded agent lease identity should match the authenticated agent principal id; human-readable display names are attribution data, not the lease-match key.

### Standards / Repo Expectations

The Go standards payload should reflect how `tillsyn` itself is organized:

- Bare control repo with visible sibling worktrees.
- One dedicated `gopls` MCP server per active worktree.
- Use Hylla MCP first for Tillsyn code understanding; use other tools only as needed.
- Use Context7 and `go doc` during planning before building, before fixing tests after failures, and during QA.
- MCP-first dogfooding for runtime and operator workflows.
- `mage` is the canonical build/test gate.
- Laslig styling for all Mage functions and CLI output.
- Fang for help and CLI command surfaces.
- Bubble Tea v2, Bubbles v2, Lip Gloss v2, and Charmbracelet stack on v2.
- GitHub Actions CI and release snapshot checks must stay green.
- No ad-hoc `.codex/` directories inside worktrees.
- Follow `AGENTS.md` workflow and worktree rules.
- Comments and handoffs are the coordination layer.
- Template-generated QA blockers must be completed truthfully before done.

## Project Description

Use this description for `TILLSYN`:

> Local-first human-agent planning and execution workspace with MCP-first dogfooding, scoped auth, template-driven workflow contracts, shared comments and handoffs, and semantic project search.

## Template Contract

### Project-Level Behavior

For `go-project`:

- auto-create one phase:
  - `IMPLEMENTATION TRACK`

That phase should be:

- `child_scope_level = "phase"`
- `child_kind_id = "implementation-phase"`
- `responsible_actor_kind = "builder"`
- `editable_by_actor_kinds = ["builder", "orchestrator"]`
- `completable_by_actor_kinds = ["builder", "human"]`
- `required_for_parent_done = true`

### Lifecycle Contract

For normal branch/work execution inside an existing Go project, the default-go lifecycle should be:

- `PLAN`
- `BUILD`
- `CLOSEOUT`
- `BRANCH CLEANUP`

`PROJECT SETUP` is project-only work. It is not part of every branch lane. Use it only when bootstrapping a new project or onboarding an existing repo into Tillsyn.

The intended operator flow is:

- create or confirm a `PLAN` phase first;
- if the branch/phase already exists because the work is obvious, create the branch/phase and immediately use the `PLAN` phase to fill out the full task tree before broad implementation starts;
- do not treat ad-hoc building without a populated `PLAN` phase as the preferred path.

### Project-Only Setup Contract

New projects should get one setup phase before normal work begins:

- `PROJECT SETUP`

That phase should cover at least:

- template fit review with the dev;
- Hylla ingest-mode decision with the dev:
  - `structural_only`
  - `embeddings_only`
  - `full_enrichment`
- initial Hylla ingest or refresh;
- git vs Hylla freshness confirmation for the intended ref;
- project metadata / standards lock;
- creation of the first `PLAN` phase.

Existing-project onboarding should use the same setup contract. It should also confirm whether the currently bound template still matches the project's needs and discuss template updates with the dev before changing the workflow contract.

### Default-Go Lifecycle Management

`default-go` should be treated as a builtin-managed approved global template library, not as a one-shot bootstrap artifact.

The lifecycle contract should be:

- install `default-go` when missing;
- refresh it explicitly when the repo-backed builtin definition changes;
- keep provenance metadata clear enough to explain whether the installed library is current or drifted from the builtin source;
- keep approval state explicit rather than silently mutating libraries in the background;
- preserve existing project bindings until a dev explicitly chooses to reapply or upgrade them.

Refreshing the library definition is allowed to update the library row and its provenance metadata. It is not allowed to silently rewrite already-created project state.

### Project-Template Update / Reapply Contract

If `default-go` changes later, bound projects should stay stable until a dev explicitly reapplies or upgrades the binding.

That reapply path should be available through both:

- the TUI; and
- MCP/CLI surfaces.

The reapply flow should:

- show that the bound library changed;
- show which project-level defaults and generated-node contracts would change;
- let the dev approve the new binding intentionally instead of silently adopting it.

After a dev-approved reapply:

- future generated nodes may use the newly approved contract immediately;
- existing generated nodes must not be silently rewritten;
- existing template-owned nodes may be proposed for migration only when the runtime can still prove they are template-owned and unmodified.

Migration review should be a first-class dev approval flow:

- the dev should be able to approve individual migrations;
- the dev should also have an explicit `approve all` option in the TUI and CLI/MCP-facing operator flow;
- the orchestrator may prepare, explain, or queue those migration proposals, but final approval remains with the dev.
- the TUI presentation should use the normal Tillsyn interaction model and existing React-style component language so template review feels like ordinary review/approval work instead of a separate admin console.
- the operator flow should stay simple: drift summary first, then proposed migrations, then per-item approve/skip plus `approve all` when the dev wants bulk adoption.

The goal for MVP is explicit adoption with no silent drift, not automatic bulk rewriting of live project work.

Current implementation direction for this contract:

- template-library rows should carry explicit revision/provenance metadata;
- project bindings should pin a bound revision and a bound library snapshot for stable future generation;
- binding reads should expose whether the project is current or has an update available;
- builtin lifecycle status and explicit install/refresh should be available through both:
  - `till template builtin status|ensure`; and
  - `till.template(operation=get_builtin_status|ensure_builtin)`;
- explicit builtin ensure should fail loudly when required kinds are still missing instead of silently installing a partial contract;
- explicit reapply may use the existing bind/update path as long as it remains visibly dev-approved and drift-aware.

### Plan-Phase Contract

Every branch/work lane should get one `PLAN` phase before `BUILD`.

The `PLAN` phase should cover at least:

- Hylla-first code understanding;
- Context7 and `go doc` research before implementation;
- scope confirmation with the dev;
- detailed build-task and subtask creation for the upcoming `BUILD` phase;
- validation-plan definition;
- branch/worktree setup planning when needed;
- closeout and cleanup expectations defined up front.

### Build-Phase Contract

The `BUILD` phase holds the actual implementation tasks.

Each concrete implementation task should normally be a `build-task`, so it auto-generates:

- `QA PASS 1`
- `QA PASS 2`

### Closeout-Phase Contract

Each build lane should get one `CLOSEOUT` phase after `BUILD`.

That phase should include tasks for at least:

- all required `mage` tests/checks passing;
- local commit recorded;
- Hylla artifact ingested or refreshed and confirmed current to git;
- QA sweep 1 across completed work, using the done tasks plus Hylla-backed code understanding;
- QA sweep 2 across completed work, using the done tasks plus Hylla-backed code understanding;
- dev review;
- orchestrator + dev collaborative testing;
- push / PR / handoff readiness.

### Branch-Cleanup Contract

Each branch/work lane should end with one `BRANCH CLEANUP` phase.

That phase should include at least:

- confirm closeout work completed truthfully;
- remove linked worktree when the lane is done;
- remove the branch when it is no longer needed;
- remove the lane-specific `gopls` MCP entry;
- rerun `codex mcp list` and confirm the stale MCP server is gone;
- any other cleanup required by the bare-repo `AGENTS.md` worktree rules.

### Branch-Setup Contract

When a new branch/worktree lane is created, the branch setup work should be tracked explicitly before `BUILD`.

That setup work should cover at least:

- create a visible sibling worktree from the bare control repo;
- use the exact worktree path, not `.tmp/`, unless the dev explicitly points at a legacy `.tmp` lane;
- add one unique `gopls` MCP server entry for that worktree in the bare-root Codex config;
- rerun `codex mcp list` and confirm the new server is visible;
- create or confirm the `PLAN` phase for that branch lane.

### Task-Level Behavior

For every `build-task`:

- auto-create:
  - `QA PASS 1`
  - `QA PASS 2`

Both QA subtasks should be:

- `responsible_actor_kind = "qa"`
- `editable_by_actor_kinds = ["qa"]`
- `completable_by_actor_kinds = ["qa", "human"]`
- `required_for_parent_done = true`

## Exact Template Object

```json
{
  "id": "default-go",
  "scope": "global",
  "name": "Default Go",
  "status": "approved",
  "node_templates": [
    {
      "id": "go-project-template",
      "scope_level": "project",
      "node_kind_id": "go-project",
      "display_name": "Go Project",
      "project_metadata_defaults": {
        "owner": "Evan",
        "standards_markdown": "- Bare control repo with visible sibling worktrees.\n- One dedicated `gopls` MCP server per active worktree.\n- Use Hylla MCP first for Tillsyn code understanding; use other tools only as needed.\n- Use Context7 and `go doc` during planning before building, before fixing tests after failures, and during QA.\n- MCP-first dogfooding for runtime and operator workflows.\n- `mage` is the canonical build/test gate.\n- Laslig styling for all Mage functions and CLI output.\n- Fang for help and CLI command surfaces.\n- Bubble Tea v2, Bubbles v2, Lip Gloss v2, and Charmbracelet stack on v2.\n- GitHub Actions CI and release snapshot checks must stay green.\n- No ad-hoc `.codex/` directories inside worktrees.\n- Follow `AGENTS.md` workflow and worktree rules.\n- Comments and handoffs are the coordination layer.\n- Template-generated QA blockers must be completed truthfully before done."
      },
      "child_rules": [
        {
          "id": "implementation-track",
          "position": 1,
          "child_scope_level": "phase",
          "child_kind_id": "implementation-phase",
          "title_template": "IMPLEMENTATION TRACK",
          "description_template": "Primary implementation and verification phase for the project.",
          "responsible_actor_kind": "builder",
          "editable_by_actor_kinds": ["builder", "orchestrator"],
          "completable_by_actor_kinds": ["builder", "human"],
          "required_for_parent_done": true
        }
      ]
    },
    {
      "id": "build-task-template",
      "scope_level": "task",
      "node_kind_id": "build-task",
      "display_name": "Build Task",
      "child_rules": [
        {
          "id": "qa-pass-1",
          "position": 1,
          "child_scope_level": "subtask",
          "child_kind_id": "qa-check",
          "title_template": "QA PASS 1",
          "description_template": "First QA pass for the parent build task.",
          "responsible_actor_kind": "qa",
          "editable_by_actor_kinds": ["qa"],
          "completable_by_actor_kinds": ["qa", "human"],
          "required_for_parent_done": true
        },
        {
          "id": "qa-pass-2",
          "position": 2,
          "child_scope_level": "subtask",
          "child_kind_id": "qa-check",
          "title_template": "QA PASS 2",
          "description_template": "Second QA pass for the parent build task.",
          "responsible_actor_kind": "qa",
          "editable_by_actor_kinds": ["qa"],
          "completable_by_actor_kinds": ["qa", "human"],
          "required_for_parent_done": true
        }
      ]
    }
  ]
}
```

## Initial Dogfood Phase

Inside the generated `IMPLEMENTATION TRACK` phase, create these initial `build-task` items:

- `FIX TILLSYN MCP DISCOVERY`
- `LOCK DEFAULT ENABLED TOOLS`
- `RENAME MCP NOUNS TO PLAN_ITEM FAMILY`
- `REDUCE PROJECT TOOL REDUNDANCY`
- `REDUCE PLAN_ITEM TOOL REDUNDANCY`
- `REDUCE HANDOFF TOOL REDUNDANCY`
- `REDUCE ATTENTION TOOL REDUNDANCY`
- `REDUCE LEASE TOOL VISIBILITY`
- `ALIGN README WITH MCP SURFACE`
- `ALIGN BOOTSTRAP GUIDE WITH MCP SURFACE`

Because these are `build-task` items, each one should auto-generate:

- `QA PASS 1`
- `QA PASS 2`

## MCP-Only Execution Sequence

In a fresh agent session where the `till_*` MCP tools are callable, execute this sequence:

1. Verify current runtime state through MCP.
2. Confirm `TEST_PROJECT` exists only as a test project and that `TILLSYN` does not already exist.
3. Obtain or confirm the expected auth scope:
   - use global approved agent auth for kind/template-library admin and project creation/binding;
   - after `TILLSYN` exists, use project-scoped approved agent auth for guarded in-project mutations.
4. Upsert these kinds:
   - `go-project`
   - `implementation-phase`
   - `build-task`
   - `qa-check`
5. Upsert the global template library `default-go`.
6. Create `TILLSYN` with:
   - `kind = "go-project"`
   - the locked description above
   - the standards payload in the template metadata defaults
7. Bind `default-go` during project creation if the current MCP/project-create path supports it.
   - If create-time bind is not exposed cleanly in the live MCP session, create the project and then bind the template immediately afterward.
8. Confirm the project-level template generated:
   - `IMPLEMENTATION TRACK`
9. Under that phase, create the initial build tasks listed above.
10. For each build task, confirm the template-generated subtasks appear:
   - `QA PASS 1`
   - `QA PASS 2`
11. Confirm the generated QA subtasks carry the expected node contract:
   - `responsible_actor_kind = "qa"`
   - `editable_by_actor_kinds = ["qa"]`
   - `completable_by_actor_kinds = ["qa", "human"]`
   - `required_for_parent_done = true`

## Validation Checklist

The executing agent should verify all of the following through MCP-visible state:

- `default-go` exists as an approved global template library.
- `go-project`, `implementation-phase`, `build-task`, and `qa-check` exist in the kind catalog.
- `TILLSYN` exists with kind `go-project`.
- `TILLSYN` has the agreed project description.
- `TILLSYN` has the agreed standards markdown.
- `TILLSYN` is bound to `default-go`.
- `IMPLEMENTATION TRACK` exists under `TILLSYN`.
- Each initial build task exists under `IMPLEMENTATION TRACK`.
- Each build task auto-generated `QA PASS 1` and `QA PASS 2`.
- Each QA subtask has the expected contract snapshot.

## Deferred / Not In Scope Yet

- Enforcing that `QA PASS 1` and `QA PASS 2` must be completed by two distinct QA principals.
- Final default `enabled_tools` cleanup for the Tillsyn MCP server.
- Archive/delete policy changes beyond the current accepted temporary behavior.
- The exact expanded JSON object for the full `PLAN` / `BUILD` / `CLOSEOUT` / `BRANCH CLEANUP` lifecycle contract and project-setup/onboarding contract.
- Exact final field-by-field payload shape for the migration-review queue and lifecycle export objects.

## Handoff Prompt For A Fresh Agent

Use this exact prompt in a fresh Codex session launched from:

- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main`

Prompt:

```text
Use only the Tillsyn MCP tools in this session. Follow AGENTS.md. Read /Users/evanschultz/Documents/Code/hylla/tillsyn/main/TILLSYN_DEFAULT_GO_DOGFOOD_SETUP.md first and treat it as the setup contract. Then:

1. Verify current runtime state through MCP only.
2. Use the expected auth scopes:
   - global approved agent auth for kind/template-library admin and project creation/binding;
   - project-scoped approved agent auth for guarded in-project mutations after the project exists.
3. Upsert kinds `go-project`, `implementation-phase`, `build-task`, and `qa-check`.
4. Upsert the approved global template library `default-go` exactly as specified in the markdown file.
5. Create project `TILLSYN` in all caps as kind `go-project`, using the locked description and standards.
6. Bind `default-go` during project creation if possible, otherwise bind immediately after creation.
7. Confirm the template generated `IMPLEMENTATION TRACK`.
8. Under that phase, create these build-task items:
   - FIX TILLSYN MCP DISCOVERY
   - LOCK DEFAULT ENABLED TOOLS
   - RENAME MCP NOUNS TO PLAN_ITEM FAMILY
   - REDUCE PROJECT TOOL REDUNDANCY
   - REDUCE PLAN_ITEM TOOL REDUNDANCY
   - REDUCE HANDOFF TOOL REDUNDANCY
   - REDUCE ATTENTION TOOL REDUNDANCY
   - REDUCE LEASE TOOL VISIBILITY
   - ALIGN README WITH MCP SURFACE
   - ALIGN BOOTSTRAP GUIDE WITH MCP SURFACE
9. Verify each build-task auto-generated `QA PASS 1` and `QA PASS 2`.
10. Verify the generated QA subtasks have the expected node contract.

Do not use CLI mutation commands. Report exact MCP results, any schema mismatches, and any missing tool/path needed to complete the setup.
```
