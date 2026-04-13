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
- Phase kinds:
  - `project-setup-phase`
  - `plan-phase`
  - `build-phase`
  - `closeout-phase`
  - `branch-cleanup-phase`
- Task kind: `build-task`
- Phase kind: `refactor-phase`
- Phase kind: `dogfood-refactor-phase`
- Task kind: `refactor-task`
- Task kind: `dogfood-refactor-task`
- Subtask kind: `qa-check`
- Subtask kind: `commit-and-reingest`
- Dogfood project names should use all caps.

### Ownership / Actor-Kind Rules

- `builder`, `qa`, `research`, `orchestrator`, and `human` are actor kinds.
- Template ownership is by actor kind, not by a pre-known specific agent principal.
- Both QA review subtasks should use actor kind `qa`.
- The stricter rule "two different QA principals must complete the proof-oriented and falsification-oriented QA reviews" is deferred to a later policy wave.
- The template should still distinguish proof-oriented QA from falsification-oriented QA and explicitly describe the intended named agent pattern:
  - `qa-proof-agent`
  - `qa-falsification-agent`
- Shared cross-client agent names should be treated as guidance in template text and docs now, even though runtime routing is still by actor kind:
  - `orchestration-agent`
  - `planning-agent`
  - `research-agent`
  - `builder-agent`
  - `qa-proof-agent`
  - `qa-falsification-agent`
  - `closeout-agent`
  - `commit-and-reingest-agent`
  - `gopls-worktree-agent`
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
  - a capability lease does not upgrade a human session into an agent session.
- Default MCP surface note:
  - `till.auth_request` is the preferred auth-request family for `create|list|get|claim|cancel`;
  - `till.project` is the preferred project-root family for `list|create|update|bind_template|get_template_binding|set_allowed_kinds|list_allowed_kinds|list_change_events|get_dependency_rollup`;
  - `till.plan_item` is the preferred plan-item read/mutation family for `get|list|search|create|update|move|move_state|delete|restore|reparent`;
  - `till.kind`, `till.template`, and `till.embeddings` are the preferred family tools for catalog/template/embedding lifecycle work;
  - `till.comment` is the preferred append-only coordination family for `create|list` and should not be folded into `plan_item`;
  - the older flat project/template/kind aliases are compatibility-only where still exposed and should not be treated as the preferred default surface.
- Agents and operators should not treat the global-to-project auth split as a bug.
- After creating a project with global auth, the next normal step is to claim or reuse a project-scoped approved agent session before creating guarded in-project work.
- Guarded agent lease identity should match the authenticated agent principal id; human-readable display names are attribution data, not the lease-match key.

### Standards / Repo Expectations

The Go standards payload should reflect how `tillsyn` itself is organized:

- Bare control repo with visible sibling worktrees.
- One dedicated `gopls` MCP server per active worktree.
- Use Hylla MCP first for Tillsyn code understanding; use other tools only as needed.
- Use Context7 and `go doc` during planning before building, before fixing tests after failures, and during QA.
- For semantic, high-risk, or ambiguous work, record a semi-formal reasoning certificate covering premises, evidence, trace or cases, conclusion, and unknowns.
- Use Hylla for committed repo-local evidence, use git diff for uncommitted local deltas, and use Context7 only for external semantics the repo cannot prove.
- Build in small tested increments and prefer TDD where practical.
- Maintain the repo coverage gate and use the canonical `mage` verification flow.
- Prefer the smallest concrete design that satisfies the current requirement.
- Reuse or refactor existing code when that is the best option.
- No implementation, cleanup, QA, parity-check, or repair work should happen without an explicit task or subtask at the correct level.
- If tests, CI, or QA fail, create a new explicit fix task or subtask before repair work begins. Tillsyn does not auto-create or force that repair item today, so the orchestrator or human must add it explicitly.
- If additional repair is needed after a task or subtask was already completed, create a new explicit item at that same level instead of silently reusing the completed one. The shipped default workflow treats that as policy, not as an automatically enforced reopen ban.
- Every phase should end with explicit push-and-reingest confirmation or an explicit no-repo-delta record before downstream work treats the baseline as current.
- Refactor work should use the shipped `refactor-phase`, `dogfood-refactor-phase`, `refactor-task`, and `dogfood-refactor-task` contracts when that workflow is the real fit.
- For refactor and dogfood-refactor work, builders should update slice and phase descriptions with git-diff line deltas, before-and-after repo and Hylla counts, timing windows, and cleanup/security findings after QA or validation, and orchestrators should roll those metrics up to the parent phase description plus the report artifact. Tillsyn generates the explicit metrics checkpoints for that work, but it does not auto-verify every metric field or rollup total today.
- Do not add abstraction for hypothetical future variation.
- Prefer idiomatic Go naming, package structure, interface placement, error handling, logging boundaries, and test shape.
- Wrap and bubble errors with context instead of swallowing them.
- Unresolved uncertainty must become explicit coordination state instead of optimistic completion.
- Confirmed-good build work must be committed, pushed, and refreshed into Hylla where needed before downstream reasoning treats the graph as current.
- MCP-first dogfooding for runtime and operator workflows.
- `mage` is the canonical build/test gate.
- Laslig styling for all Mage functions and CLI output.
- Fang for help and CLI command surfaces.
- Bubble Tea v2, Bubbles v2, Lip Gloss v2, and Charmbracelet stack on v2.
- GitHub Actions CI and release snapshot checks must stay green.
- No ad-hoc `.codex/` directories inside worktrees.
- Follow `AGENTS.md` workflow and worktree rules.
- Comments, handoffs, and attention are distinct coordination surfaces: comments are shared discussion, handoffs are explicit next-action routing, and attention is the durable inbox substrate.
- Template-generated QA blockers must be completed truthfully before done.

## Project Description

Use this description for `TILLSYN`:

> Local-first human-agent planning and execution workspace with MCP-first dogfooding, scoped auth, template-driven workflow contracts, shared comments and handoffs, durable attention/inbox state, and semantic project search.

## Template Contract

### Project-Level Behavior

For `go-project`:

- auto-create one phase:
  - `PROJECT SETUP`

That phase should be:

- `child_scope_level = "phase"`
- `child_kind_id = "project-setup-phase"`
- `responsible_actor_kind = "orchestrator"`
- `editable_by_actor_kinds = ["orchestrator"]`
- `completable_by_actor_kinds = ["orchestrator", "human"]`
- `required_for_parent_done = true`

### Branch-Level Behavior

For `branch` lanes:

- auto-create these phases in order:
  - `PLAN`
  - `BUILD`
  - `CLOSEOUT`
  - `BRANCH CLEANUP`

Those phases should be generated by the branch template itself so normal lane execution begins only after a branch/worktree lane exists.

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
- project reapply preview should be explicit and drift-aware before adoption:
  - `till template project preview --project-id PROJECT_ID`;
  - `till.project(operation=preview_template_reapply, project_id=...)`;
- explicit existing-node migration approval should be available through both:
  - `till template project approve-migrations --project-id PROJECT_ID --task-id TASK_ID|--all`; and
  - `till.project(operation=approve_template_migrations, project_id=..., task_ids=[...]|approve_all=true)`;
- builtin lifecycle status and explicit install/refresh should be available through both:
  - `till template builtin status|ensure`; and
  - `till.template(operation=get_builtin_status|ensure_builtin)`;
- explicit builtin ensure should fail loudly when required kinds are still missing instead of silently installing a partial contract;
- explicit builtin ensure guidance should tell agents to run builtin status first and treat required/missing kinds as a runtime-DB bootstrap mismatch, not as the builtin template being absent;
- explicit reapply may use the existing bind/update path as long as it remains visibly dev-approved and drift-aware;
- in the TUI, saving a project with the same selected drifted library should open a drift summary + migration-review step before rebinding future generated work.
- that TUI review step should expose per-item selection, explicit `approve all`, and explicit skip for existing generated nodes while preserving dev approval as the final gate.

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
- explicit task and subtask creation before any implementation, QA, parity-check, or repair work begins.

### Build-Phase Contract

The `BUILD` phase holds the actual implementation tasks.

Each concrete implementation task should normally be a `build-task`, `refactor-task`, or `dogfood-refactor-task`, and nested refactor work should use `refactor-phase` or `dogfood-refactor-phase` when that is the real fit.

Each generated phase should also include:

- `PHASE PUSH AND REINGEST CONFIRMATION`

`build-task` auto-generates:

- `QA PROOF REVIEW`
- `QA FALSIFICATION REVIEW`
- `COMMIT PUSH AND REINGEST`

`refactor-task` adds:

- `PARITY VALIDATION IN ACTION`
- `METRICS CAPTURE AND REPORT`

`dogfood-refactor-task` adds:

- `TEST AGAINST DEV VERSION`
- `CONFIRM LOCAL USED VERSION UPDATED`
- `METRICS CAPTURE AND REPORT`

For refactor and dogfood-refactor work, the builder should update the slice task description after QA and parity or dev validation with truthful metrics such as `git diff` added/removed/net lines, tracked-source line counts before and after, touched files/packages, Hylla node/block/orphan counts before and after, active/wait/ingest timing windows, ingest cost when available, and cleanup/security findings. The orchestrator should then roll those values up into the parent refactor phase description and the markdown report artifact. Tillsyn generates the explicit metrics checkpoints for that work, but it does not auto-verify every metric field or rollup total today.

Any failing tests, CI, QA, parity validation, or dev-version validation should create a new explicit follow-up task or subtask before repair work begins. Tillsyn does not auto-create or force that follow-up item today, so the orchestrator or human must add it explicitly.

### Closeout-Phase Contract

Each build lane should get one `CLOSEOUT` phase after `BUILD`.

That phase should include tasks for at least:

- all required `mage` tests/checks passing;
- local commit recorded;
- push confirmed when a new baseline exists;
- Hylla artifact ingested or refreshed and confirmed current to git;
- proof-oriented QA across completed work, using the done tasks plus Hylla-backed code understanding;
- falsification-oriented QA across completed work, using the done tasks plus Hylla-backed code understanding;
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
  - `QA PROOF REVIEW`
  - `QA FALSIFICATION REVIEW`
  - `COMMIT PUSH AND REINGEST`

For every `refactor-task`:

- auto-create:
  - `QA PROOF REVIEW`
  - `QA FALSIFICATION REVIEW`
  - `PARITY VALIDATION IN ACTION`
  - `COMMIT PUSH AND REINGEST`
  - `METRICS CAPTURE AND REPORT`

For every `dogfood-refactor-task`:

- auto-create:
  - `QA PROOF REVIEW`
  - `QA FALSIFICATION REVIEW`
  - `TEST AGAINST DEV VERSION`
  - `CONFIRM LOCAL USED VERSION UPDATED`
  - `COMMIT PUSH AND REINGEST`
  - `METRICS CAPTURE AND REPORT`

Both QA subtasks should be:

- `responsible_actor_kind = "qa"`
- `editable_by_actor_kinds = ["qa"]`
- `completable_by_actor_kinds = ["qa", "human"]`
- `required_for_parent_done = true`

The `COMMIT PUSH AND REINGEST` subtask should be:

- `responsible_actor_kind = "builder"`
- `editable_by_actor_kinds = ["builder", "orchestrator"]`
- `completable_by_actor_kinds = ["builder", "human"]`
- `required_for_parent_done = true`
- focused on committing confirmed-good work, pushing the baseline that downstream tooling should rely on, triggering Hylla refresh, waiting for ingest completion, and recording the resulting freshness evidence

## Exact Template Object

The executable builtin template source now lives in:

- [default-go.json](/Users/evanschultz/Documents/Code/hylla/tillsyn/main/templates/builtin/default-go.json)

That repo file is the authoritative shipped contract. It now contains:

- project root generation for `PROJECT SETUP`;
- project-setup task generation for template fit, Hylla decisions, metadata lock, first branch lane, and first `PLAN` phase preparation;
- branch-lane generation for `PLAN`, `BUILD`, `CLOSEOUT`, and `BRANCH CLEANUP`;
- phase-level `PHASE PUSH AND REINGEST CONFIRMATION` generation;
- closeout and branch-cleanup default task generation; and
- `build-task`, `refactor-task`, and `dogfood-refactor-task` generated blocker work.

## Initial Dogfood Tree

Inside the generated `PROJECT SETUP` phase, complete the setup items first. Then create the first real branch lane and use its generated `PLAN` and `BUILD` phases.

Inside that first branch lane's generated `BUILD` phase, create these initial `build-task` items:

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

- `QA PROOF REVIEW`
- `QA FALSIFICATION REVIEW`
- `COMMIT PUSH AND REINGEST`

## MCP-Only Execution Sequence

In a fresh agent session where the `till_*` MCP tools are callable, execute this sequence:

1. Verify current runtime state through MCP.
2. Confirm `TEST_PROJECT` exists only as a test project and that `TILLSYN` does not already exist.
3. Obtain or confirm the expected auth scope:
   - use global approved agent auth for kind/template-library admin and project creation/binding;
   - after `TILLSYN` exists, use project-scoped approved agent auth for guarded in-project mutations.
4. Upsert these kinds:
   - `go-project`
   - `project-setup-phase`
   - `plan-phase`
   - `build-phase`
   - `closeout-phase`
   - `branch-cleanup-phase`
   - `build-task`
   - `refactor-phase`
   - `dogfood-refactor-phase`
   - `refactor-task`
   - `dogfood-refactor-task`
   - `qa-check`
   - `commit-and-reingest`
5. Upsert the global template library `default-go`.
6. Create `TILLSYN` with:
   - `kind = "go-project"`
   - the locked description above
   - the standards payload in the template metadata defaults
7. Bind `default-go` during project creation if the current MCP/project-create path supports it.
   - If create-time bind is not exposed cleanly in the live MCP session, create the project and then bind the template immediately afterward.
8. Confirm the project-level template generated:
   - `PROJECT SETUP`
9. Create the first real branch lane and confirm it auto-generated:
   - `PLAN`
   - `BUILD`
   - `CLOSEOUT`
   - `BRANCH CLEANUP`
10. Under the generated `BUILD` phase, create the initial build tasks listed above.
11. For each build task, confirm the template-generated subtasks appear:
   - `QA PROOF REVIEW`
   - `QA FALSIFICATION REVIEW`
   - `COMMIT PUSH AND REINGEST`
12. Confirm the generated QA subtasks carry the expected node contract:
   - `responsible_actor_kind = "qa"`
   - `editable_by_actor_kinds = ["qa"]`
   - `completable_by_actor_kinds = ["qa", "human"]`
   - `required_for_parent_done = true`

## Validation Checklist

The executing agent should verify all of the following through MCP-visible state:

- `default-go` exists as an approved global template library.
- `go-project`, `project-setup-phase`, `plan-phase`, `build-phase`, `closeout-phase`, `branch-cleanup-phase`, `build-task`, `refactor-phase`, `dogfood-refactor-phase`, `refactor-task`, `dogfood-refactor-task`, and `qa-check` exist in the kind catalog.
- `commit-and-reingest` exists in the kind catalog.
- `TILLSYN` exists with kind `go-project`.
- `TILLSYN` has the agreed project description.
- `TILLSYN` has the agreed standards markdown.
- `TILLSYN` is bound to `default-go`.
- `PROJECT SETUP` exists under `TILLSYN`.
- A branch lane exists under `TILLSYN`.
- That branch lane auto-generated `PLAN`, `BUILD`, `CLOSEOUT`, and `BRANCH CLEANUP`.
- Each generated phase includes `PHASE PUSH AND REINGEST CONFIRMATION`.
- Each initial build task exists under the generated `BUILD` phase.
- Each build task auto-generated `QA PROOF REVIEW`, `QA FALSIFICATION REVIEW`, and `COMMIT PUSH AND REINGEST`.
- Each QA subtask has the expected contract snapshot.

## Deferred / Not In Scope Yet

- Enforcing that proof-oriented QA and falsification-oriented QA must be completed by two distinct QA principals.
- Final default `enabled_tools` cleanup for the Tillsyn MCP server.
- Archive/delete policy changes beyond the current accepted temporary behavior.
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
3. Upsert kinds `go-project`, `project-setup-phase`, `plan-phase`, `build-phase`, `closeout-phase`, `branch-cleanup-phase`, `build-task`, `refactor-phase`, `dogfood-refactor-phase`, `refactor-task`, `dogfood-refactor-task`, `qa-check`, and `commit-and-reingest`.
4. Upsert the approved global template library `default-go` exactly as specified in the markdown file.
5. Create project `TILLSYN` in all caps as kind `go-project`, using the locked description and standards.
6. Bind `default-go` during project creation if possible, otherwise bind immediately after creation.
7. Confirm the template generated `PROJECT SETUP`.
8. Create the first branch lane and confirm it auto-generated `PLAN`, `BUILD`, `CLOSEOUT`, and `BRANCH CLEANUP`.
9. Under the generated `BUILD` phase, create these build-task items:
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
10. Verify each build-task auto-generated `QA PROOF REVIEW`, `QA FALSIFICATION REVIEW`, and `COMMIT PUSH AND REINGEST`.
11. Verify the generated QA subtasks have the expected node contract.

Do not use CLI mutation commands. Report exact MCP results, any schema mismatches, and any missing tool/path needed to complete the setup.
```
