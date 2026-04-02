# tillsyn

A local-first Kanban TUI built with Bubble Tea v2, Bubbles v2, and Lip Gloss v2.

`tillsyn` uses the Swedish word `tillsyn` ("oversight/supervision").
The project/repo name is `tillsyn`, and the runtime command name is `till`.

`tillsyn` is designed as a better human-visible planning and verification surface than ad-hoc markdown checklists. The primary direction is human + coding-agent collaboration with explicit state, auditability, and clear completion gates, while still remaining useful as a standalone personal TUI task manager.
A core product purpose is maintaining one DB-backed source of truth for planning/execution state instead of fragmented markdown files.

Current scope:
- local tracking and planning workflows (human-operated TUI).
- local runtime diagnostics with styled logging and runtime log files under the active root, plus dev-mode workspace-local log placement.
- the active auth/runtime dogfood run is tracked in `PLAN.md`.
- advanced import/export transport-closure concerns (branch/commit-aware divergence reconciliation and richer conflict tooling) remain roadmap-only unless user re-prioritizes.

Contributor workflow and CI policy: `CONTRIBUTING.md`
Local branch/worktree workflow expectations are documented in `AGENTS.md`.

Local dogfood repo layout note:
- the bare control repo lives one directory above this checkout,
- `main/` is the operator/integration worktree,
- additional linked worktrees typically live under the bare root's `.tmp/`.

## Features
- Multi-project Kanban board.
- Launches into a project picker first (no auto-created default project).
- SQLite persistence (`modernc.org/sqlite`, no CGO).
- Keyboard navigation (`vim` keys + arrows) and mouse support.
- Archive-first delete flow with configurable defaults.
- Project and work-item thread mode with ownership-attributed markdown comments as the shared in-scope communication lane for human-to-agent and agent-to-agent coordination.
- Descriptions/comments are stored as markdown source fields and rendered in TUI views.
- MCP instruction tool for embedded docs plus scoped project/template/kind/node explanations (`till.get_instructions`).
- Raw stdio MCP via `./till mcp` as the primary local MCP transport.
- Secondary HTTP/API + HTTP MCP serve surface via `./till serve`.
- Project roots are real filesystem directory mappings; resource attachment is blocked outside the allowed root.
- Runtime kind-catalog + project allowlist validation for project/task mutations.
- Runtime JSON-schema validation for kind metadata payloads (with compiled-validator caching).
- Shared-DB `autent` integration for session-first MCP mutation auth.
- Capability leases retained as secondary local workflow/delegation guards while the auth UX is still being completed.
- JSON snapshot import/export.
- Configurable task field visibility.

## Active Status (2026-04-02)
Implemented now:
- Use `PLAN.md` as the active source of truth for the current dogfood auth/runtime wave.
- Local-only TUI + SQLite workflows (including startup bootstrap, project picker, threads/comments, and import/export snapshots).
- `./till`, `./till mcp`, and `./till serve` now share the same real default runtime unless the user explicitly opts into a different runtime.
- Local builds no longer silently force dev mode.
- `./till mcp` stays the raw stdio MCP server and shuts down cleanly on `Ctrl-C`.
- Shared-DB `autent` wiring is active for session-first MCP mutation auth.
- Raw-stdio MCP auth now supports local auth-context handles:
  - `till.auth_request(operation=claim|validate_session)` returns `auth_context_id` on the stdio runtime,
  - reduced mutation families accept `auth_context_id` instead of requiring inline `session_secret` on normal local MCP mutations,
  - and acting-session governance/delegation flows on `till.auth_request` accept `acting_auth_context_id` with the existing `acting_session_id`.
- Board info line includes hierarchy-aware focus guidance (`f` focus subtree, `F` return full board) with selected level and child counts for branch/phase navigation, including nested phases.
- Board scope rendering is level-scoped: project shows immediate project children, and focused branch/phase views show immediate children for that level (not full descendant dumps).
- Task-focused scope renders direct subtasks in the board so `f` on a task opens subtask-level board context.
- Board path context is always visible above columns (`path: project -> ...`) and updates on each `f` drill-down.
- Board cards now include hierarchy markers in metadata (`[branch|...]` / `[phase|...]`) so branch/phase rows are visually distinct from task rows.
- Wide layouts render a right-side notices panel with unresolved attention summary, selected-item context, and recent activity hints.
- Attention is now the durable inbox substrate for routed coordination:
  - comment mentions for `@dev`, `@builder`, `@qa`, `@orchestrator`, and `@research` materialize as role-targeted attention rows (`@dev` aliases to builder),
  - durable handoffs mirror into stable inbox attention rows for the target role,
  - and project/global notifications now load project-wide unresolved attention instead of only project-root attention.
- `n` now respects active focus scope: in focused branch/phase it creates a child in that scope, and in focused task scope it creates a subtask.
- Kind-catalog bootstrap + project `allowed_kinds` enforcement is active for project/task write paths.
- Project-level `kind` and task-level `scope` persistence are active (`project|branch|phase|task|subtask` semantics enforced by kind rules, with nested phases inferred from parent lineage).
- Legacy kind-template compatibility paths still exist for create-time defaults and generated work when no template library is selected or when no matching bound node template exists.
- Locked next-step direction: templates are now intended to evolve into SQLite-backed workflow-and-authority contracts that define generated follow-up work, actor-kind edit/complete permissions, truthful completion gates, `system` audit provenance for generated nodes, and explicit global-to-project adopt/apply flows instead of silent backfill; the current planning contract is tracked in `TEMPLATING_DESIGN_MEMO.md`.
- Project creation can now optionally bind one approved global template library at create time, so project-scoped template defaults and root generated work start from the template-library model instead of the legacy kind-template path when the operator chooses that path.
- The intended auth model is now explicit:
  - global agent auth is for global catalog admin, template-library admin, and project creation/binding;
  - project-scoped agent auth is for guarded mutations inside that project;
  - narrower branch/phase/task auth should be used when the runtime can prove that path.
- Snapshot import/export now preserves template libraries, project bindings, and node-contract snapshots so generated workflow contracts round-trip with the work graph instead of being flattened back to legacy defaults.
- Current template-library enforcement slice is active for generated nodes:
  - create-child under a generated parent,
  - update / rename / reparent,
  - move-to-done,
  - archive / delete / restore.
  Stored node-contract snapshots now gate non-human actor kinds after the normal scope lease check, humans remain allowed, orchestrator completion still requires explicit per-rule override, and done transitions now honor required parent / containing-scope blockers from generated descendants instead of treating every child as an implicit blocker.
- Comments remain deliberately separate from template-contract mutation gating by design:
  - comments stay shared within the normal project/scope visibility model so humans can talk directly to subagents and agents can hand off to each other inside Tillsyn,
  - comment attribution/ownership remains first-class audit data,
  - and later targeted-routing or limit/configuration UX can build on that without turning comments into hidden per-role silos by default.
- Capability leases now normalize project scope ids, validate scope tuples on issuance, enforce bounded parent delegation, and apply builder/qa/orchestrator action checks in app/service write paths for non-user actors.

Still in progress for this dogfood wave:
- broader user-configurable policy/grant management beyond the current local dogfood request/session flow
- explicit anti-adoption gatekeeping for any future auth-context reuse or attachment flow beyond the requester-bound claim path
- richer disconnect-aware cleanup, broader continuous-listening/HTTP transport follow-through, and later OS-level notification ergonomics on top of the baseline-aware stdio watcher model
- final collaborative dogfood retest closeout and evidence capture in `PLAN.md`

Current MCP/runtime direction:
- `capture_state` is the summary-first recovery surface for level-scoped workflows; after client shutdown/restart, call it first to re-anchor project/scope context before resuming any watchers.
- `till.get_instructions` is the embedded-doc and scoped policy/explanation surface; it can return selected markdown docs plus agent-facing recommendations, it can explain one concrete project, template library, kind, or node from runtime state without turning into a raw schema browser, and `topic=bootstrap` is now the canonical richer bootstrap explanation path.
- `till.get_bootstrap_guide` remains on the frozen tool family as the lightweight compatibility wrapper for empty-instance and pre-approval flows.
- Attention/blocker signaling direction is node-scoped with user-action visibility and paginated scope queries for user/agent coordination.
- MCP mutation auth is session-first.
- transport-level lease/scope request contracts remain secondary local workflow guardrails for non-user mutations.
- MCP tool surface now includes:
  - instructions: `till.get_instructions`
  - bootstrap guidance: `till.get_bootstrap_guide`
  - auth requests: `till.auth_request`
  - projects and project-root reads/admin: `till.project`
  - tasks/work graph: `till.plan_item`
  - capture/attention: `till.capture_state`, `till.attention_item`
  - kinds/catalog admin: `till.kind`
  - template libraries/contracts: `till.template`
  - embeddings lifecycle: `till.embeddings`
  - capability leases: `till.capability_lease`
  - comments: `till.comment`
  - handoffs: `till.handoff`
  - empty-instance `capture_state` now returns deterministic `bootstrap_required` signaling, and agents can call `till.get_bootstrap_guide` for next steps.
  - recovery/watch guidance:
    - during active runs, use `wait_timeout` on `till.attention_item(operation=list)`, `till.comment(operation=list)`, and `till.handoff(operation=list)` to wait for the next change after the current baseline state instead of polling;
    - after client shutdown/restart, recover in this order: `till.capture_state`, `till.attention_item(operation=list, all_scopes=true)` for inbox state, `till.handoff(operation=list)` for durable coordination state, then `till.comment(operation=list)` for any thread you need to resume.
  - parity/guardrail notes:
    - `capture_state.state_hash` is stable across MCP/HTTP calls for unchanged underlying state (timestamp jitter excluded from hash input);
    - `till.capability_lease(operation=revoke_all)` fails closed on invalid/unknown scope tuples;
    - `till.comment(operation=create)` fails closed when the target does not exist in the referenced project;
    - `till.plan_item(operation=update)` title-only updates preserve existing priority when `priority` is omitted.

Current auth note:
- Normal TUI users should not need to manually issue themselves auth sessions for routine TUI use.
- `till auth request create|list|show|approve|deny|cancel` and `till auth session list|validate|revoke` are now active for dogfood/operator use.
- Auth request scopes now support:
  - `project/<project-id>[/branch/<branch-id>[/phase/<phase-id>...]]`,
  - `projects/<project-id-a>,<project-id-b>...`,
  - `global`;
  with multi-project/general scope reserved for orchestrators.
- TUI auth-request notifications route to focused-project vs global panels, and `enter` opens auth review directly instead of a generic thread fallback.
- TUI auth review now uses a dedicated full-screen review surface with visible decision controls, human-readable scope labels, explicit confirm-before-apply for both approve and deny, and optional notes that start blank instead of prefilled audit prose.
- TUI auth inventory distinguishes pending requests, resolved requests, and active approved sessions, but the active-session revoke path is still less discoverable than it should be; CLI is the clearer operator revoke path for now.
- TUI project edit now includes a focusable `comments` row that opens the project thread directly on the comments panel and returns cleanly to edit mode on `esc`.
- CLI auth inventory supports project/global request and session listing so operators can inspect and revoke without guesswork.
- MCP requesters can now resume approved requests through `till.auth_request(operation=claim)` using the requester-owned `resume_token` returned by `till.auth_request(operation=create)`; when callers provide custom continuation metadata, `continuation_json.resume_token` must still be present and non-empty. For delegated on-behalf-of approvals, the approved child principal/client now owns the continuation claim directly.
- MCP requesters can now also withdraw their own pending requests through `till.auth_request(operation=cancel)` using that same requester-owned continuation proof (`request_id`, `resume_token`, `principal_id`, and `client_id`), and cancel ownership stays separate from child self-claim.
- On raw stdio MCP, callers should prefer auth-context handles over repeating secrets:
  - `till.auth_request(operation=claim|validate_session)` returns `auth_context_id`,
  - mutation-family tools such as `till.project`, `till.plan_item`, `till.comment`, `till.handoff`, `till.attention_item`, `till.kind`, `till.template`, and `till.capability_lease` accept `session_id` plus `auth_context_id`,
  - acting-session flows on `till.auth_request` accept `acting_session_id` plus `acting_auth_context_id`,
  - inline `session_secret` and `acting_session_secret` remain compatibility fallbacks,
  - and the local auth-context handle path is currently a stdio-runtime feature, not the preferred HTTP serve flow yet.
- The reduced auth family now also exposes approved-session governance for MCP/runtime dogfood:
  - use `till.auth_request(operation=list_sessions|check_session_governance|revoke_session)` with an acting approved session whose approved path already contains the governed session scope;
  - simple project overlap is not enough to govern a broader global or multi-project session;
  - explicit multi-project orchestrator approvals may govern matching project or multi-project child sessions without gaining global reach;
  - session-governance policy:
    - orchestrators may inspect/check/revoke any session at their approved scope or below;
    - non-orchestrator agents may only inspect/check/revoke the exact same session they are currently using for self-cleanup;
    - do not widen self-cleanup to sibling or descendant session governance;
    - prefer the non-destructive `check_session_governance` operation for negative proof and UI/runtime explanation paths instead of relying on failed destructive revoke attempts;
    - do not make ordinary task completion implicitly destroy the caller's auth session by default;
    - if future workflow-specific auto-cleanup exists, keep it opt-in and limited to explicitly ephemeral sessions created for one bounded unit of work.
  - use `till.auth_request(operation=validate_session)` when you already possess the target session secret and need to inspect the resolved session identity/details.
- Expected scoped-auth workflow:
  - use global approved agent sessions for template-library admin and `till.project(operation=create)`;
  - once the project exists, use a project-scoped approved agent session for guarded in-project mutations such as `till.plan_item(operation=create)`;
  - on raw stdio MCP, first claim or validate the acting session to get `auth_context_id`, then prefer `session_id` + `auth_context_id` for subsequent mutation calls;
  - when an orchestrator needs child builder/qa/research auth, create that child request through `till.auth_request(operation=create, acting_session_id=..., acting_auth_context_id=...)` so requester ownership is derived from the acting orchestrator session and the child path stays bounded within the acting approved path;
  - builder, qa, and research agents may still request their own single-project rooted auth directly, but they must not mint sibling or broader child sessions for other principals;
  - do not treat the global-to-project auth split as a runtime bug.
- Auth hygiene expectations:
  - never use another agent's or user's auth session, session secret, or `auth_context_id`;
  - never paste auth material into comments, handoffs, task metadata, docs, or external logs;
  - always request the narrowest scope and shortest reasonable lifetime for the work being done;
  - orchestrators are responsible for child-session cleanup at the end of a run unless a stricter project/template rule says otherwise;
  - builders, qa, and research agents may clean up only their own sessions and must not govern sibling or broader sessions;
  - after a run ends, do not leave muddy auth state behind: pending requests, active child sessions, stale leases, and stale coordination rows should all be cleaned up truthfully;
  - after restart, recover current auth and coordination state before minting replacement sessions so the runtime is not polluted with duplicate leftovers.
- Guarded agent lease identity should be rooted in the authenticated agent principal id; display names are for attribution, not lease matching.
- `till.capability_lease(operation=issue)` now follows that same rule directly:
  - for authenticated agent sessions, the issued lease identity is derived from the authenticated principal instead of trusting a caller-supplied `agent_name`;
  - explicit `agent_name` remains relevant only for non-agent/operator lease issuance paths.
- Default surface note:
  - `till.auth_request` now owns auth-request create, list, get, claim, and cancel plus auth-session list, validate, governance-check, and revoke;
  - `till.project` now owns project-root mutations such as create, update, template bind, and allowed-kinds updates;
  - `till.project` also owns project-root reads such as list, template binding lookup, allowed-kinds lookup, change events, and dependency rollups;
  - `till.plan_item` now owns plan-item reads and mutations such as get, list, search, create, update, move, move_state, delete, restore, and reparent;
  - `till.kind` now owns kind catalog list/upsert, `till.template` now owns template-library list/get/upsert plus node-contract lookup, `till.embeddings` now owns status/reindex, and `till.comment` now owns comment create/list;
  - only selected older flat project/template/kind aliases remain behind explicit legacy config for compatibility testing.
- Policy direction for the unified `plan_item` surface:
  - the responsible actor kind should be able to move its own work through ordinary active states such as `todo -> progress -> done` when the stored node contract allows it;
  - humans remain allowed to perform those transitions;
  - `till.plan_item(operation=move_state)` is the preferred contract-aware state-transition shape for policy-gated forward and backward workflow movement;
  - builders should not gain unrestricted power over terminal cleanup just because they can progress their own work, and QA/orchestrator/human transitions should still be gated by stored policy and scope;
  - destructive or terminal cleanup actions such as delete, hard cleanup, and final archive remain more restricted and should not default to agent autonomy.
- Comment-family direction:
  - comments should not be folded into `till.plan_item`; they are a separate coordination/threading type.
  - the default comment-family shape is `till.comment(operation=create|list)`.
  - comments should stay append-only by default in the first family pass; agent comment editing is intentionally deferred so the coordination log remains trustworthy.
  - comments should be allowed anywhere inside the caller's approved scope subtree, which means parallel/sibling commenting is fine when the approved scope already covers both nodes.
  - if a caller does not hold scope broad enough for the affected sibling/parallel node, the preferred escalation path is still handoff or attention rather than silently widening comment reach.
- Mentions/notifications direction:
  - routed inbox attention is now landed on the frozen surface without adding a new top-level tool:
    - comment mentions for `@dev`, `@builder`, `@qa`, `@orchestrator`, and `@research` sync into role-targeted `attention` rows,
    - durable handoffs sync into one stable target-role inbox attention row,
    - `till.attention_item(operation=list)` now supports project-wide reads plus `target_role` filtering through `all_scopes` and `target_role`,
    - and the TUI notices panels consume project-wide unresolved attention so routed inbox items surface naturally in project/global notifications.
  - TUI notifications should now distinguish inbox-style comment mentions from generic warning/action rows:
    - routed comment mentions belong in a dedicated `Comments` section instead of the generic `Warnings` section,
    - `Action Required` should be reserved for structured handoff/action routing that is explicitly addressed to the current viewer,
    - lingering handoffs for other roles should stay visible as oversight warnings/coordination state instead of looking like human work items,
    - the board/TUI should only surface comment-mention inbox rows that match the current viewer identity (`human` for ordinary user sessions; explicit agent-role matches when the local identity is configured that way),
    - and comment inbox rows should be clearable one at a time after review instead of being treated as ambient warning clutter.
  - Live status:
    - bounded builder, QA, and research agent sessions can now post real project-thread comments and durable handoffs on the same project scope,
    - those comments fan out into role-targeted mention inbox rows exactly once per mentioned role,
    - per-item clear works on routed comment mentions without clearing unrelated handoffs or older mentions in the same role inbox,
    - auth wait wake is proven live through `till.auth_request(operation=claim, wait_timeout=...)`,
    - local cross-process runtime coverage now also proves comment, attention, and handoff waiters wake on the next newer change after the current baseline state,
    - and the fresh native MCP rerun now also proves live auth wake, routed attention wake, handoff wake, and comment wake on a clean empty task thread.
  - current transport caveat:
    - stdio waitable watcher plumbing is now landed for attention, comments, and handoffs with baseline-aware “next change” semantics,
    - local runtime tests now cover auth plus coordination wake end to end across broker instances,
    - and the native MCP proof is now re-closed on the patched runtime,
    - but this is still not a full push-notification transport for disconnected clients, HTTP listeners, or OS notifications;
    - after restart, agents should recover durable state with `capture_state`, `attention_item(list)`, `handoff(list)`, and thread `comment(list)` reads before resuming watchers.
- Current remaining dogfood order:
  - first close with one final collaborative dogfood hardening pass,
  - and only after that slice finishes, run one explicit cleanup/refinement wave for the production dogfood dataset, notification polish, rendering polish, OS-level notification ergonomics, and richer rule/template composition UX.
- The lower-level `till auth issue-session` seam still exists as a temporary operator/developer escape hatch, but it is no longer the primary documented flow.
- Current continuation status: `till.auth_request(operation=claim)` now uses a runtime-local cross-process live wake path for local dogfood runs, so TUI or CLI approve/deny/cancel in one process can wake a waiting requester in another process without app-layer polling; delegated child approvals now support direct child claim while requester cleanup remains separate and requester-bound.
- Current bounded-delegation status: `till.auth_request(operation=create)` now also supports explicit child delegation through `acting_session_id` and `acting_session_secret`; when used, requester attribution is derived from the acting session, child paths must stay within the acting approved path, and only orchestrators may create sibling child auth for other principals.
- Current cancel constraint: the MCP cancel path is requester-bound and continuation-bound. It is meant for orchestrator/requester cleanup of pending requests, not human/operator review cancellation or descendant-session management, and it should not be used as a claim-ownership proof path.
- Active approved-session shutdown is a separate path:
  - use `till.auth_request(operation=revoke_session)` for live session revocation,
  - and keep `till.auth_request(operation=cancel)` limited to requester-owned pending-request cleanup.
- Role-model note:
  - the domain model already includes `research` alongside `orchestrator`, `builder`, and `qa`;
  - current MCP auth/lease surfaces now expose `orchestrator|builder|qa|research`;
  - `planner` is not yet a first-class runtime role and should not be added casually without deciding whether it is a constrained orchestration role, a research/planning hybrid, or just a naming alias.
- Role-purpose note:
  - `orchestrator` plans, routes, delegates, and cleans up coordination state.
  - `builder` implements and reports progress with comments/handoffs.
  - `qa` verifies outcomes, returns/reopens work when needed, and closes verification handoffs.
  - `research` inspects code and runtime state, compiles findings or bug inventories, and may use local MCP tools plus Context7 to gather evidence before handing results back.
- Coordination primer:
  - `till.comment` is the shared append-only discussion lane for humans and agents on the same in-scope node or subtree.
  - Supported routed mentions are `@human`, `@dev`, `@builder`, `@qa`, `@orchestrator`, and `@research`; `@dev` aliases to builder.
  - Mentioned comments create routed inbox rows in the `Comments` notifications section for the matching viewer/role.
  - `till.handoff` is the structured next-action lane; open handoffs should appear in `Action Required` only for the addressed viewer and otherwise remain visible as coordination/oversight warnings until the receiving agent resolves them.
  - `attention` is the durable inbox substrate underneath routed mentions, handoffs, and other notification-worthy rows.
  - If you see `Action Required`, assume there is an open handoff or similarly explicit work-request row for the current viewer, not just a plain comment.
- Current live-transport caveat: waitable stdio watchers are now landed for auth, attention, comments, and handoffs, but they still depend on an active waiting client. This is not yet a disconnected push-notification system, richer session-aware cleanup layer, or HTTP/continuous-listening transport.
- Product expectation note: humans and orchestrators are expected to keep active plans current inside Tillsyn itself. When plans change, the corresponding nodes should be updated or archived in Tillsyn so humans and agents are not coordinating against stale markdown drift.
- Bootstrap/instructions status:
  - `till.get_instructions(topic=bootstrap)` is now the canonical richer bootstrap explanation surface.
  - `till.get_bootstrap_guide` remains the dedicated lightweight compatibility wrapper on the frozen MCP family.
  - the current scoped explanation surface already accepts optional `project_id`, `template_library_id`, `kind_id`, and `node_id` plus `focus=project|template|kind|node|topic`.
  - project scope now explains project standards, allowed kinds, template binding, and project-local workflow expectations.
  - template scope now explains node-template descriptions, child rules, responsible actor kinds, blocker rules, and migration/reapply context.
  - branch/phase/task/subtask scope now explains the concrete node's description plus metadata fields such as objective, implementation notes, acceptance criteria, definition of done, and validation plan, together with any stored node-contract snapshot.
  - this explanation layer prefers persisted runtime policy sources such as `standards_markdown`, template descriptions, task metadata, project bindings, and node-contract snapshots before falling back to generic embedded docs.

Template-library operator examples:
- SQLite is the live source of truth. JSON is the stable CLI/MCP transport for template-library reads and writes, while the TUI is the primary human review/approval/editor surface.
- CLI template operators now follow the same laslig-style human output contract as the rest of the operator surface:
  - `--spec-json` remains the machine-friendly ingestion path,
  - list/show/bind/contract commands render human-readable tables/detail views for auditability instead of raw JSON blobs.
- TUI surfaces now expose the same contract model without a separate template UI stack:
  - project create/edit includes a project-kind picker and an approved-library picker plus approved-library hints,
  - task info shows the active project library and any generated-node contract snapshot,
  - and comments remain shared regardless of template ownership.
- Template child rules are the contract mechanism:
  - a node template can auto-generate follow-up work,
  - assign each generated node to a responsible actor kind,
  - restrict edit/complete actions per actor kind,
  - and mark specific generated nodes as required blockers for parent or containing-scope completion.
- Current default-go workflow direction:
  - `PROJECT SETUP` is project-only onboarding work for new or adopted projects,
  - normal branch/work execution should flow through `PLAN`, `BUILD`, `CLOSEOUT`, and `BRANCH CLEANUP`,
  - the preferred operator flow is to create or confirm `PLAN` before broad implementation begins,
  - and the fuller lifecycle contract for project setup, branch setup, plan/build/closeout/cleanup, generated QA work, and the initial `TILLSYN` dogfood tree is locked in `TILLSYN_DEFAULT_GO_DOGFOOD_SETUP.md`.
- Example shape:
  - a `build-task` template can generate two `qa-check` children with different titles, both owned by `qa`, both `required_for_parent_done: true`, and both still commentable because comments remain the shared coordination lane.
- Default-go lifecycle management direction:
  - `default-go` should be treated as a builtin-managed template library rather than a one-off bootstrap artifact,
  - refresh/install should be explicit and auditable,
  - the runtime should be able to show whether the builtin library is missing, current, or drifted from the repo-backed builtin source,
  - and refreshing the library definition must not silently rewrite already-bound project state.
- Project-template update/reapply direction:
  - if `default-go` or another bound approved library changes later, existing projects should stay stable until a dev explicitly reapplies or upgrades the binding,
  - TUI and MCP/CLI should both expose that reapply path,
  - the reapply flow should show what changed in project defaults and generated-node contracts before apply,
  - future generated nodes may start using the newly approved contract after that explicit reapply,
  - existing generated nodes must not be silently rewritten,
  - any migration of existing template-owned nodes should be surfaced as dev-approved action-required work with both per-item approval and an explicit `approve all` affordance,
  - and the review UI should use the normal TUI/React-style component language already used elsewhere in Tillsyn rather than inventing a separate template-admin presentation model.
- Current lifecycle implementation status:
  - template libraries now carry revision/provenance metadata,
  - project bindings pin a bound library snapshot plus the bound revision instead of following the mutable latest library row,
  - binding reads now surface drift/current state against the latest approved library revision,
  - builtin `default-go` lifecycle state is now surfaced explicitly as missing/current/update-available,
  - builtin install/refresh now has an explicit ensure path rather than relying on ad-hoc generic upserts,
  - project reapply preview is now surfaced explicitly before adoption,
  - existing-node migration approvals are now available through MCP/CLI as an explicit per-item or `approve all` action against the drift preview,
  - TUI project edit now routes same-library drifted saves through a dedicated migration-review screen before rebinding,
  - that review screen shows drift summary, proposed existing-node migrations, per-item selection, `approve all`, and explicit skip,
  - explicit reapply still uses the existing bind flow against the latest approved library rather than a separate new verb,
  - and the remaining follow-through has shifted from missing TUI reapply UX to broader dogfood hardening and live parity on the refreshed runtime.
- CLI examples:
  - `till project create --name "Go Service" --kind go-service --template-library-id go-defaults`
  - `till.project(operation=create, name="Go Service", kind="go-service", template_library_id="go-defaults", ...)`
  - `till template library list --scope global --status approved`
  - `till template builtin status --library-id default-go`
  - `till template builtin ensure --library-id default-go`
  - `till template library show --library-id go-defaults`
  - `till template library upsert --spec-json '{"id":"go-defaults","scope":"global","name":"Go Defaults","status":"approved","node_templates":[{"id":"tmpl-build-task","scope_level":"task","node_kind_id":"build-task","display_name":"Build Task","child_rules":[{"id":"qa-pass-1","position":1,"child_scope_level":"subtask","child_kind_id":"qa-check","title_template":"QA pass 1","responsible_actor_kind":"qa","editable_by_actor_kinds":["qa"],"completable_by_actor_kinds":["qa","human"],"required_for_parent_done":true},{"id":"qa-pass-2","position":2,"child_scope_level":"subtask","child_kind_id":"qa-check","title_template":"QA pass 2","responsible_actor_kind":"qa","editable_by_actor_kinds":["qa"],"completable_by_actor_kinds":["qa","human"],"required_for_parent_done":true}]}]}'`
  - `till template project bind --project-id <project-id> --library-id go-defaults`
  - `till template project binding --project-id <project-id>`
  - `till template project preview --project-id <project-id>`
  - `till template project approve-migrations --project-id <project-id> --all`
  - `till template contract show --node-id <node-id>`
- Kind catalog note:
  - `till kind` is now the node-kind registry/allowlist surface.
  - template-library workflow contracts should be created and inspected through `till template`, the TUI project form, or MCP JSON transport instead of the legacy kind-template seam.
- Documentation expectations:
  - keep README workflow examples aligned with the actor kinds and generated blocker rules that Tillsyn actually enforces.
  - keep at least one canonical example that shows multi-child gatekeeping such as a build task that auto-generates multiple QA subtasks.
  - keep examples readable enough for humans to audit quickly in the TUI and CLI; template contracts should clarify ownership and completion gates instead of hiding them in large markdown files.
  - keep the docs explicit that comments are the durable shared communication layer inside Tillsyn, which is a core value-add over external markdown plans for human-to-agent and agent-to-agent coordination.

Instruction-tool usage guidance:
- `till.get_instructions` is intended for missing/stale/ambiguous policy context, not mandatory on every step.
- Keep context bounded with `doc_names` and `max_chars_per_doc`.
- Use `include_markdown=false` for inventory checks and `include_markdown=true` when full markdown text is required.
- Descriptions/details and comment summary/body fields are markdown-first authoring surfaces.
- Use `focus=project|template|kind|node` with `project_id`, `template_library_id`, `kind_id`, or `node_id` when you need scoped rules for a real runtime object instead of generic doc guidance.

Roadmap-only in the active wave (explicitly deferred):
- advanced import/export transport closure concerns (branch/commit-aware divergence reconciliation and conflict tooling),
- remote/team auth-tenancy expansion and additional security hardening,
- template-library authoring/approval/binding UX, richer actor-kind-aware template-policy surfaces, stronger truthful-completion surfacing, durable wait/recovery UX, broader template-library expansion, broader session-aware MCP wait/notify reuse for comments and handoffs, richer human+agent search/filtering (keyword/path/vector/hybrid with deduped provenance-aware results), and HTTP/continuous-listening support for a later wave.

After the current active slices close, run one cleanup/refinement wave focused on real dogfood usage:
- clean/refresh the local dogfood DB and create the canonical `tillsyn` project/task tree that will be used for real collaborative dogfooding.
- add intuitive non-JSON TUI/operator surfaces for viewing and editing workflow rules and template policy:
  - global template rules,
  - project-scoped rules,
  - branch/phase/task/subtask rule overlays,
  - and the effective inherited rule view for one concrete node.
- design and implement composable template layering rather than one flat template choice:
  - one project should be able to inherit general `go` rules plus a narrower layer such as `go cli/tui`, `go backend`, or `go wasm`,
  - child template layers should be able to override or extend parent defaults instead of forcing duplicated whole-template copies,
  - the effective rule-precedence model should be explicit, for example: global template base -> subtype overlays -> project rule overrides -> node-local contract/metadata,
  - and the human-facing UI should make inherited vs overridden rule sources obvious.
- move `Action Required` to the top of the project notifications panel.
- add one configurable notifications accent/color, dogfood orange first, and apply it consistently to attention-worthy notification surfaces such as comments, warnings, and action-required rows before choosing a long-term default.
- highlight `@human` mentions in rendered thread/comment markdown so human-directed asks stand out immediately.
- connect Tillsyn notifications to the local terminal/OS notification path for `@mentions`, `Action Required`, and other attention-worthy events.
- read the full `Agentic Code Reasoning` paper (`arXiv:2603.01896`) and explicitly review how its semi-formal reasoning model should reshape default templates, generated workflow contracts, and scoped instructions before the real dogfood template set is finalized.
- plan and discuss, in detail, which parts of that paper should map into Tillsyn template phases, research/qa handoffs, and instruction guidance rather than treating the paper as a one-off reading note.
- keep the notifications panel labeling explicit enough that `comment mention`, `handoff`, and other attention kinds cannot be mistaken for each other during dogfood.
- group global notifications by project instead of scattering repeated project headers through the panel.
- clear comment-notification rows from notices after the viewer opens/reviews them so old mentions do not keep muddying the project/global notifications panels.
- add unread/new cues plus history/audit-friendly wording for the same notification surfaces so terminal dings still have an in-product trace.
- add notification-noise controls such as per-kind mute/quieting or dedupe rules if real dogfood shows repeated routed rows becoming distracting.

Current post-dogfood consensus note:
- the detailed working consensus for that template/agent/communication scope is tracked in `TEMPLATE_AGENT_CONSENSUS.md` until it is folded back into the canonical docs.

Dangerous limitation note (pre-hardening, design warning):
- In future policy-controlled override flows, orchestrator calls may receive override-token material.
- That design currently assumes orchestrator adherence to user policy/guidance; treat overrides as explicit user-approved actions only.

## Run
```bash
mage run
```

Or build once and run the binary:
```bash
mage build
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
- template libraries + project template bindings + node-contract snapshots
- comments/threads
- capability leases
- handoffs

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
./till auth request create --path project/<project-id> --principal-id <principal-id> --principal-type agent --client-id <client-id> --reason "dogfood request"
./till auth request approve --request-id <request-id> --note "approved for dogfood"
./till auth session validate --session-id <session-id> --session-secret <session-secret>
./till auth session revoke --session-id <session-id> --reason operator_revoke
```

Dogfood MCP continuation pattern:
```json
{
  "tool": "till.auth_request",
  "arguments": {
    "operation": "create",
    "path": "project/<project-id>",
    "principal_id": "<principal-id>",
    "principal_type": "agent",
    "client_id": "<client-id>",
    "reason": "dogfood request",
    "continuation_json": "{\"resume_token\":\"opaque-requester-token\",\"resume_tool\":\"till.attention_item\"}"
  }
}
```

If `continuation_json` is omitted, `till.auth_request(operation=create)` now auto-generates a requester-owned `resume_token` and returns it in the create result. If `continuation_json` is provided, `continuation_json.resume_token` must still be present and non-empty.

After the user approves the request in the TUI, the requester can claim the approved session through MCP with the same `request_id` plus that `resume_token` using `till.auth_request(operation=claim)`.

If the requester needs to withdraw a still-pending request, it can call `till.auth_request(operation=cancel)` with that same `request_id` plus the requester-owned `resume_token`, `principal_id`, and `client_id`.

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
mage test-pkg ./internal/app
mage ci
mage build
```

For contribution policy, pre-push expectations, and branch-protection recommendations, see `CONTRIBUTING.md`.

VHS visual regression captures:
```bash
vhs vhs/regression_subtasks.tape
vhs vhs/regression_scroll.tape
```

Golden tests:
```bash
mage test-golden
mage test-golden-update
```

## CI
GitHub Actions runs `mage ci` on macOS/Linux/Windows, then validates a Goreleaser snapshot.
