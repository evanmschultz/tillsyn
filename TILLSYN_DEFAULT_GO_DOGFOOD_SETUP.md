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

Inside `TILLSYN`, create this first phase:

- `MCP TOOL SURFACE RATIONALIZATION`

Under that phase, create these initial `build-task` items:

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
3. Upsert these kinds:
   - `go-project`
   - `implementation-phase`
   - `build-task`
   - `qa-check`
4. Upsert the global template library `default-go`.
5. Create `TILLSYN` with:
   - `kind = "go-project"`
   - the locked description above
   - the standards payload in the template metadata defaults
6. Bind `default-go` during project creation if the current MCP/project-create path supports it.
   - If create-time bind is not exposed cleanly in the live MCP session, create the project and then bind the template immediately afterward.
7. Confirm the project-level template generated:
   - `IMPLEMENTATION TRACK`
8. Under that phase, create the initial build tasks listed above.
9. For each build task, confirm the template-generated subtasks appear:
   - `QA PASS 1`
   - `QA PASS 2`
10. Confirm the generated QA subtasks carry the expected node contract:
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
- The future MCP noun-family refactor from `task`-heavy naming to `plan_item`.
- Final default `enabled_tools` cleanup for the Tillsyn MCP server.
- Archive/delete policy changes beyond the current accepted temporary behavior.

## Handoff Prompt For A Fresh Agent

Use this exact prompt in a fresh Codex session launched from:

- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main`

Prompt:

```text
Use only the Tillsyn MCP tools in this session. Follow AGENTS.md. Read /Users/evanschultz/Documents/Code/hylla/tillsyn/main/TILLSYN_DEFAULT_GO_DOGFOOD_SETUP.md first and treat it as the setup contract. Then:

1. Verify current runtime state through MCP only.
2. Upsert kinds `go-project`, `implementation-phase`, `build-task`, and `qa-check`.
3. Upsert the approved global template library `default-go` exactly as specified in the markdown file.
4. Create project `TILLSYN` in all caps as kind `go-project`, using the locked description and standards.
5. Bind `default-go` during project creation if possible, otherwise bind immediately after creation.
6. Confirm the template generated `IMPLEMENTATION TRACK`.
7. Under that phase, create these build-task items:
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
8. Verify each build-task auto-generated `QA PASS 1` and `QA PASS 2`.
9. Verify the generated QA subtasks have the expected node contract.

Do not use CLI mutation commands. Report exact MCP results, any schema mismatches, and any missing tool/path needed to complete the setup.
```
