# tillsyn

A local-first Kanban TUI built with Bubble Tea v2, Bubbles v2, and Lip Gloss v2.

`tillsyn` uses the Swedish word `tillsyn` ("oversight/supervision").
The project/repo name is `tillsyn`, and the runtime command name is `till`.

`tillsyn` is designed as a better human-visible planning and verification surface than ad-hoc markdown checklists. The primary direction is human + coding-agent collaboration with explicit state, auditability, and clear completion gates, while still remaining useful as a standalone personal TUI task manager.
A core product purpose is maintaining one DB-backed source of truth for planning/execution state instead of fragmented markdown files.

Current scope:
- local tracking and planning workflows (human-operated TUI).
- local runtime diagnostics with styled logging and dev-mode local log files.
- the active auth/runtime dogfood run is tracked in `PLAN.md`.
- advanced import/export transport-closure concerns (branch/commit-aware divergence reconciliation and richer conflict tooling) remain roadmap-only unless user re-prioritizes.

Contributor workflow and CI policy: `CONTRIBUTING.md`

## Features
- Multi-project Kanban board.
- Launches into a project picker first (no auto-created default project).
- SQLite persistence (`modernc.org/sqlite`, no CGO).
- Keyboard navigation (`vim` keys + arrows) and mouse support.
- Archive-first delete flow with configurable defaults.
- Project and work-item thread mode with ownership-attributed markdown comments.
- Descriptions/comments are stored as markdown source fields and rendered in TUI views.
- MCP instruction tool for embedded docs + agent recommendations (`till.get_instructions`).
- Raw stdio MCP via `./till mcp` as the primary local MCP transport.
- Secondary HTTP/API + HTTP MCP serve surface via `./till serve`.
- Project roots are real filesystem directory mappings; resource attachment is blocked outside the allowed root.
- Runtime kind-catalog + project allowlist validation for project/task mutations.
- Runtime JSON-schema validation for kind metadata payloads (with compiled-validator caching).
- Shared-DB `autent` integration for session-first MCP mutation auth.
- Capability leases retained as secondary local workflow/delegation guards while the auth UX is still being completed.
- JSON snapshot import/export.
- Configurable task field visibility.

## Active Status (2026-03-21)
Implemented now:
- Use `PLAN.md` as the active source of truth for the current dogfood auth/runtime wave.
- Local-only TUI + SQLite workflows (including startup bootstrap, project picker, threads/comments, and import/export snapshots).
- `./till`, `./till mcp`, and `./till serve` now share the same real default runtime unless the user explicitly opts into a different runtime.
- Local builds no longer silently force dev mode.
- `./till mcp` stays the raw stdio MCP server and shuts down cleanly on `Ctrl-C`.
- Shared-DB `autent` wiring is active for session-first MCP mutation auth.
- Board info line includes hierarchy-aware focus guidance (`f` focus subtree, `F` return full board) with selected level and child counts for branch/phase navigation, including nested phases.
- Board scope rendering is level-scoped: project shows immediate project children, and focused branch/phase views show immediate children for that level (not full descendant dumps).
- Task-focused scope renders direct subtasks in the board so `f` on a task opens subtask-level board context.
- Board path context is always visible above columns (`path: project -> ...`) and updates on each `f` drill-down.
- Board cards now include hierarchy markers in metadata (`[branch|...]` / `[phase|...]`) so branch/phase rows are visually distinct from task rows.
- Wide layouts render a right-side notices panel with unresolved attention summary, selected-item context, and recent activity hints.
- `n` now respects active focus scope: in focused branch/phase it creates a child in that scope, and in focused task scope it creates a subtask.
- Kind-catalog bootstrap + project `allowed_kinds` enforcement is active for project/task write paths.
- Project-level `kind` and task-level `scope` persistence are active (`project|branch|phase|task|subtask` semantics enforced by kind rules, with nested phases inferred from parent lineage).
- Kind template system actions can auto-append checklist items and auto-create child work items during task creation; the broader post-dogfood hierarchy-wide template/agent/communication direction is tracked in `TEMPLATE_AGENT_CONSENSUS.md`.
- Capability leases now normalize project scope ids, validate scope tuples on issuance, enforce bounded parent delegation, and apply builder/qa/orchestrator action checks in app/service write paths for non-user actors.

Still in progress for this dogfood wave:
- broader user-configurable policy/grant management beyond the current local dogfood request/session flow
- orchestrator/builder/qa scoped-auth choreography, including orchestrator-only multi-project/general scope enforcement and bounded delegation
- explicit anti-adoption gatekeeping for any future auth-context reuse or attachment flow beyond the requester-bound claim path
- final collaborative dogfood retest closeout and evidence capture in `PLAN.md`

Current MCP/runtime direction:
- `capture_state` is a summary-first recovery surface for level-scoped workflows.
- Attention/blocker signaling direction is node-scoped with user-action visibility and paginated scope queries for user/agent coordination.
- MCP mutation auth is session-first.
- transport-level lease/scope request contracts remain secondary local workflow guardrails for non-user mutations.
- MCP tool surface now includes:
  - instructions: `till.get_instructions`
  - bootstrap guidance: `till.get_bootstrap_guide`
  - projects: `till.list_projects`, `till.create_project`, `till.update_project`
  - tasks/work graph: `till.list_tasks`, `till.create_task`, `till.update_task`, `till.move_task`, `till.delete_task`, `till.restore_task`, `till.reparent_task`, `till.list_child_tasks`, `till.search_task_matches`
  - capture/attention: `till.capture_state`, `till.list_attention_items`, `till.raise_attention_item`, `till.resolve_attention_item`
  - change/dependency context: `till.list_project_change_events`, `till.get_project_dependency_rollup`
  - kinds/allowlists: `till.list_kind_definitions`, `till.upsert_kind_definition`, `till.set_project_allowed_kinds`, `till.list_project_allowed_kinds`
  - capability leases: `till.issue_capability_lease`, `till.heartbeat_capability_lease`, `till.renew_capability_lease`, `till.revoke_capability_lease`, `till.revoke_all_capability_leases`
  - comments: `till.create_comment`, `till.list_comments_by_target`
  - empty-instance `capture_state` now returns deterministic `bootstrap_required` signaling, and agents can call `till.get_bootstrap_guide` for next steps.
  - parity/guardrail notes:
    - `capture_state.state_hash` is stable across MCP/HTTP calls for unchanged underlying state (timestamp jitter excluded from hash input);
    - `till.revoke_all_capability_leases` fails closed on invalid/unknown scope tuples;
    - `till.create_comment` fails closed when the target does not exist in the referenced project;
    - `till.update_task` title-only updates preserve existing priority when `priority` is omitted.

Current auth note:
- Normal TUI users should not need to manually issue themselves auth sessions for routine TUI use.
- `till auth request create|list|show|approve|deny|cancel` and `till auth session list|validate|revoke` are now active for dogfood/operator use.
- Auth request scopes now support:
  - `project/<project-id>[/branch/<branch-id>[/phase/<phase-id>...]]`,
  - `projects/<project-id>,<project-id>...`,
  - `global`;
  with multi-project/general scope reserved for orchestrators.
- TUI auth-request notifications route to focused-project vs global panels, and `enter` opens auth review directly instead of a generic thread fallback.
- TUI auth review now uses a dedicated full-screen review surface with visible decision controls, human-readable scope labels, and a simpler default approve flow; deny remains note-first with explicit confirm/cancel.
- TUI auth inventory distinguishes pending requests, resolved requests, and active approved sessions, and supports direct revoke for active sessions.
- CLI auth inventory supports project/global request and session listing so operators can inspect and revoke without guesswork.
- MCP requesters can now resume approved requests through `till.claim_auth_request` when they created the original request with continuation metadata that includes a requester-owned `resume_token`.
- The lower-level `till auth issue-session` seam still exists as a temporary operator/developer escape hatch, but it is no longer the primary documented flow.

Instruction-tool usage guidance:
- `till.get_instructions` is intended for missing/stale/ambiguous policy context, not mandatory on every step.
- Keep context bounded with `doc_names` and `max_chars_per_doc`.
- Use `include_markdown=false` for inventory checks and `include_markdown=true` when full markdown text is required.
- Descriptions/details and comment summary/body fields are markdown-first authoring surfaces.

Roadmap-only in the active wave (explicitly deferred):
- advanced import/export transport closure concerns (branch/commit-aware divergence reconciliation and conflict tooling),
- remote/team auth-tenancy expansion and additional security hardening,
- hierarchy-wide node-type templates, agent-type policy, truthful completion enforcement, first-class handoffs, durable wait/recovery coordination, and broader template-library expansion.

Current post-dogfood consensus note:
- the detailed working consensus for that template/agent/communication scope is tracked in `TEMPLATE_AGENT_CONSENSUS.md` until it is folded back into the canonical docs.

Dangerous limitation note (pre-hardening, design warning):
- In future policy-controlled override flows, orchestrator calls may receive override-token material.
- That design currently assumes orchestrator adherence to user policy/guidance; treat overrides as explicit user-approved actions only.

## Run
```bash
just run
```

Or build once and run the binary:
```bash
just build
./till
```

## Startup Behavior
- TUI launch opens the project picker before normal board mode.
- If no projects exist yet, the picker stays open and supports `N` to create the first project.
- Normal TUI startup seeds a missing resolved config file from `config.example.toml` when that template is available in the current workspace root.
- On TUI startup, missing required bootstrap fields are prompted and persisted:
  - `identity.display_name`
  - one default path (stored as the single active entry in `paths.search_roots`)

## CLI Commands
Export current data:
```bash
./till export --out /tmp/till.json
```

Snapshot export includes:
- projects, columns, tasks/work-items
- kind catalog definitions + project allowed-kind closure
- comments/threads
- capability leases

Import snapshot:
```bash
./till import --in /tmp/till.json
```

Include only active records in export:
```bash
./till export --out /tmp/till-active.json --include-archived=false
```

Start the raw stdio MCP server:
```bash
./till mcp
```

Start the secondary HTTP/API + HTTP MCP server:
```bash
./till serve
```

Dogfood auth request/session commands:
```bash
./till auth request create --path project/<project-id> --principal-id review-agent --principal-type agent --client-id till-mcp-stdio --reason "dogfood request"
./till auth request approve --request-id <request-id> --note "approved for dogfood"
./till auth session validate --session-id <session-id> --session-secret <session-secret>
./till auth session revoke --session-id <session-id> --reason operator_revoke
```

Dogfood MCP continuation pattern:
```json
{
  "tool": "till.create_auth_request",
  "arguments": {
    "path": "project/<project-id>",
    "principal_id": "review-agent",
    "principal_type": "agent",
    "client_id": "till-mcp-stdio",
    "reason": "dogfood request",
    "continuation_json": "{\"resume_token\":\"opaque-requester-token\",\"resume_tool\":\"till.raise_attention_item\"}"
  }
}
```

After the user approves the request in the TUI, the requester can claim the approved session through MCP with the same `request_id` plus that `resume_token` using `till.claim_auth_request`.

Current auth caveat:
- the request/session commands above are the primary operator dogfood path
- `till auth issue-session` remains a lower-level temporary operator/developer seam
- the intended end-user flow is request-and-approval inside the product, not routine manual session minting from the shell

## Config
`till` loads TOML config from platform defaults, or from `--config` / `TILL_CONFIG`.
Help-only paths (`--help`) render usage without running runtime bootstrap side effects (including config seeding).

Database path precedence:
1. `--db`
2. `TILL_DB_PATH`
3. TOML `database.path`
4. platform default path

Path resolution controls:
- `--app` / `TILL_APP_NAME` to namespace paths (default `tillsyn`)
- `--dev` / `TILL_DEV_MODE` to explicitly use `<app>-dev` path roots
- `./till`, `./till mcp`, and `./till serve` all use the same default platform runtime when `--dev` is not enabled
- `till paths` prints `app`, `root`, `config`, `database`, `logs`, and `dev_mode` in that order
  - `root` is the active runtime root
  - `database` is the effective sqlite path after CLI/env/config resolution
  - `logs` follows the active runtime root by default and lands under `<root>/logs`
- `identity.default_actor_type` (`user|agent|system`) + `identity.display_name` are defaults for new thread comment ownership
- `paths.search_roots` stores one active default path used by bootstrap and path-pickers
- task resource attachments require a configured per-project root mapping (`project_roots`)
- dev mode logging writes to the shared runtime `logs/` directory under the resolved app root when `logging.dev_file.enabled = true`
  - explicit relative dev log dir overrides are still anchored to the nearest workspace root marker (`go.mod` or `.git`)
- logging level is controlled by TOML `logging.level` (`debug|info|warn|error|fatal`)

Example:
```toml
[database]
path = ""

[delete]
default_mode = "archive" # archive | hard

[task_fields]
show_priority = true
show_due_date = true
show_labels = true
show_description = false

[board]
show_wip_warnings = true
group_by = "none" # none | priority | state

[search]
cross_project = false
include_archived = false
states = ["todo", "progress", "done"] # plus optional "archived"

[identity]
display_name = "" # required at TUI startup bootstrap
default_actor_type = "user" # user | agent | system

[paths]
search_roots = [] # bootstrap writes one active default path entry

[logging]
level = "info"

[logging.dev_file]
enabled = true
# Default sentinel resolves to the shared runtime root logs directory.
# Explicit relative overrides are still workspace-root-relative.
dir = ".tillsyn/log"
```

Full template: `config.example.toml`

## Key Controls
- `h/l` or `←/→`: move column
- `j/k` or `↓/↑`: move task
- `n`: new task
- `e`: edit task
- `i` or `enter`: task info modal
- `c` (in task info): open thread for the selected work item
- `d` (in new-task due field): open due-date picker (`enter`/`e` in edit-task due field)
- `f`: focus selected subtree (including empty scopes)
- `F`: return to full board
- `p`: project picker
- `N` (in project picker): new project
- `:`: command palette
- `/`: search
- `d`: delete using configured default mode
- `.`: open quick actions (archive/restore and context actions)
- `a`: archive task
- `D`: hard delete task
- `u`: restore task
- `t`: toggle archived visibility
- `ctrl+y`: toggle text-selection mode (copy-friendly mouse selection)
- `?`: toggle expanded help
- `q`: quit

Command palette highlights:
- `new-branch`, `edit-branch`, `archive-branch`, `restore-branch`, `delete-branch`
- `new-phase`
- `new-project`, `edit-project`, `archive-project`, `restore-project`, `delete-project`
- while subtree focus is active, `new-branch` is blocked and shows a warning modal; clear focus (`F`) first

## Thread Mode
- Open project thread from command palette with `thread-project` (`project-thread` alias).
- Open selected work-item thread with `thread-item` (`item-thread` / `task-thread` aliases), or `c` from task info.
- Supported thread targets: project, task, subtask, phase, decision, and note.
- New comments use configured identity defaults and should render readable actor names when available.

## Fang Context
Fang is Charmbracelet's experimental batteries-included wrapper for Cobra CLIs.
`tillsyn` does not currently integrate Fang or Cobra for CLI command execution.
Current usage is Fang-inspired help copy/style in the in-app command reference overlay.

## Developer Workflow
Primary commands:
```bash
just fmt
just test-pkg ./internal/app
just check
just test
just ci
```

For contribution policy, pre-push expectations, and branch-protection recommendations, see `CONTRIBUTING.md`.

VHS visual regression captures:
```bash
just vhs
just vhs vhs/regression_subtasks.tape
just vhs vhs/regression_scroll.tape
```

Golden tests:
```bash
just test-golden
just test-golden-update
```

## CI
GitHub Actions runs split gates:
- matrix smoke checks on macOS/Linux/Windows via `just check`
- full Linux gate via `just ci`
- Goreleaser snapshot validation after the full Linux gate
