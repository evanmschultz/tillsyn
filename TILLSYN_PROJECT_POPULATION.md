# TILLSYN Project Population

Status: active migration/planning worksheet for moving the canonical `TILLSYN` project backlog into Tillsyn itself.

Purpose:
- keep one clean, current file for populating the in-app `TILLSYN` project;
- point back to older markdown sources instead of duplicating them blindly;
- record current user+agent consensus for what belongs in Tillsyn now;
- keep open questions visible until they are resolved in-project with the user.

## Source Pointers

Use these older files as source material, not as the long-term execution surface once the canonical in-app project is populated.

### Primary Sources

- `TILLSYN_DEFAULT_GO_DOGFOOD_SETUP.md`
  - goal and locked naming/auth/setup contract: lines 7-88
  - template contract and project setup expectations: lines 90-219
  - plan/build/closeout/cleanup contract and initial dogfood tree: lines 221-315
  - MCP-only execution sequence and validation checklist: lines 322-405
- `README.md`
  - current dogfood order and role/coordination/instructions/template state: lines 221-340
  - cleanup/refinement follow-through and known later dogfood TODOs: lines 342-406
- `TEMPLATE_AGENT_CONSENSUS.md`
  - product framing, node/agent type systems, communication, truth, and policy direction: lines 15-220
- `TEMPLATING_DESIGN_MEMO.md`
  - deeper template model, authority/contract framing, and structured-vs-text contract direction: lines 12-220

### Supporting Repo Guidance

- `AGENTS.md`
  - tracked repo guidance now says that once the canonical in-app `TILLSYN` project exists, markdown planning docs are migration/input material rather than the default working backlog.

## Current Consensus

### Canonical Working Surface

- The long-term working backlog should live in Tillsyn itself, not in sprawling repo markdown.
- Existing markdown files are intake material to review, discuss, migrate, trim, or retire.
- Human remains in the loop for migration decisions, template updates, policy changes, and important workflow shifts.

### Current Builtin Template Reality

- The shipped builtin `default-go` now generates:
  - project-root `PROJECT SETUP`;
  - branch-lane `PLAN`, `BUILD`, `CLOSEOUT`, and `BRANCH CLEANUP`;
  - task-level QA generation for `build-task` via `QA PASS 1` and `QA PASS 2`.
- This contract is real and executable today.
- During dogfooding, we must keep watching for workflow gaps and decide what belongs in the template versus what should remain manual planning.

### One-Time Onboarding Exception

- For this fresh canonical `TILLSYN` setup, do not execute Hylla work yet.
- Hylla-related setup items remain part of the contract and should not be forgotten.
- The temporary rule is:
  - keep Hylla items visible during onboarding,
  - skip actually running them until the project setup discussion says it is time.

### Non-Branch Dogfood Shape

- The first real dogfood run should not assume a real git branch/worktree lane when one is not intended.
- For that reason, the initial execution shape should be allowed to use project-root phases rather than forcing a branch plus `BRANCH CLEANUP`.
- This is a template/product TODO, not a reason to silently repurpose the existing branch template contract.

### Markdown Intake / Cleanup Expectations

- For existing projects with code already present, one early onboarding phase should explicitly:
  - review all markdown files in the bare repo and active worktree;
  - review relevant YAML or other worklog-like files too;
  - use subagents to inspect the codebase and determine what is already done, what is stale, what is still needed, and what needs updating;
  - produce a human-reviewed migration list before moving backlog/rules into Tillsyn;
  - clean worklog/planning sprawl out of the repo once the relevant content is safely captured in Tillsyn.

## Proposed Population Order

This is the current recommended order for filling the in-app `TILLSYN` project.

### 1. Use The Generated `PROJECT SETUP` Phase First

Keep the generated setup phase as the root onboarding area.

Already generated there:
1. `TEMPLATE FIT REVIEW`
2. `HYLLA INGEST MODE DECISION`
3. `HYLLA INITIAL INGEST OR REFRESH`
4. `HYLLA VS GIT FRESHNESS CHECK`
5. `PROJECT METADATA AND STANDARDS LOCK`
6. `CREATE OR CONFIRM FIRST BRANCH LANE`
7. `CREATE OR CONFIRM FIRST PLAN PHASE`

These should be reviewed with the user before adding more execution work.

### 2. Add Manual Onboarding Phases At Project Root

These should be treated as phases, not one-off tasks.

#### Phase: `MARKDOWN INTAKE AND CLEANUP`

Purpose:
- review all legacy planning/worklog docs;
- compare them against current code;
- identify what is done, stale, still needed, or should be updated;
- discuss with the user what belongs in Tillsyn and what should be cleaned up.

Important expectations:
- subagents should be used to inspect code and help classify old planning items;
- the orchestrator should produce a complete inventory:
  - done things;
  - stale things;
  - things that need updating;
  - things still needed;
- the human reviews that inventory before migration decisions are finalized.

#### Phase: `TILLSYN POPULATION DECISIONS`

Purpose:
- decide where migrated items belong inside Tillsyn;
- decide what remains permanent docs;
- decide what should be removed or reduced from markdown.

This is distinct from markdown intake:
- the intake phase gathers and classifies;
- this phase decides placement and actual migration shape inside Tillsyn.

#### Phase: `TEMPLATE IMPROVEMENT WATCH`

Purpose:
- keep a running in-project list of template friction, missing structure, and policy gaps found during dogfooding.

It should include at least:
- phase/task/subtask depth expectations;
- non-branch execution needs;
- template update/rebind behavior;
- onboarding template improvements for existing repos;
- how `AGENTS.md`, skills, and client-native guidance should evolve for coding clients such as Codex CLI and Claude Code while leveraging Tillsyn.

#### Phase: `ONBOARDING GAP WATCH`

Purpose:
- track first-run setup and bootstrap problems found while onboarding real projects into Tillsyn.

Examples:
- empty TUI kind/template pickers after fresh DB reset;
- missing TUI builtin-refresh action;
- unclear project creation or template adoption flow.

#### Phase: `IDENTITY-BASED MENTIONS AND NOTIFICATIONS`

Purpose:
- replace role-bucket mentions with specific actor/user targeting.

Current consensus:
- `@human` is not good enough; it should become `@user`;
- `@qa`, `@builder`, and similar role buckets are insufficient for multiple agents or future multi-dev teams;
- mentions should target unique mapped identities by name, and the TUI should show those names clearly;
- the system must prevent ambiguous duplicate agent names once identity-based mentions exist.

This should be treated as its own phase because it is a substantial product/workflow change, not a one-line follow-up.

### 3. First Real Dogfood Execution Shape

After onboarding/setup decisions are done, prefer a project-root phase-based execution shape first:

1. `DOGFOOD RUN`
2. inside that:
   - `PLAN`
   - `BUILD`
   - `CLOSEOUT`

Do not force a branch lane or `BRANCH CLEANUP` unless we actually intend to use a real git branch/worktree lane.

## Historical Build Candidates To Recheck

These came from the older dogfood setup markdown and must be checked against current code before we recreate them as live work:

1. `FIX TILLSYN MCP DISCOVERY`
2. `LOCK DEFAULT ENABLED TOOLS`
3. `RENAME MCP NOUNS TO PLAN_ITEM FAMILY`
4. `REDUCE PROJECT TOOL REDUNDANCY`
5. `REDUCE PLAN_ITEM TOOL REDUNDANCY`
6. `REDUCE HANDOFF TOOL REDUNDANCY`
7. `REDUCE ATTENTION TOOL REDUNDANCY`
8. `REDUCE LEASE TOOL VISIBILITY`
9. `ALIGN README WITH MCP SURFACE`
10. `ALIGN BOOTSTRAP GUIDE WITH MCP SURFACE`

These should not be loaded blindly.

Before adding them to Tillsyn:
- compare them to current code and docs;
- decide whether each item is:
  - already done;
  - partially done;
  - stale/superseded;
  - or still real remaining work.

## Open Questions

### Population / Workflow

- Which onboarding phases should exist in the template eventually versus staying manual for `TILLSYN` first?
- Should the initial real dogfood run stay fully phase-based at project root, or should we later introduce a real branch/worktree lane?
- How much old markdown should remain after migration versus being trimmed or deleted?

### Template / Product

- Should onboarding-existing-project behavior become part of the shipped template contract, including markdown intake and migration discussion?
- How should non-branch execution be represented in templates without overloading the current branch contract?
- Do generated phases need deeper default task/subtask structure, or should more of that stay manual?

### Identity / Coordination

- What is the exact unique-name model for agent/user mentions?
- How should duplicate-name prevention work for orchestrators, builders, QA, research, and users?
- How should identity-based mentions interact with future multi-user teams?

### Client / Guidance

- How should `AGENTS.md`, client-native skills, and Tillsyn instructions evolve so coding clients get a more native Tillsyn-assisted workflow?
- Which parts of that should live in template rules, which in instructions, and which in per-client skills/guidance?

## Immediate Next Step

Use this file as the clean checklist/consensus source while populating the in-app `TILLSYN` project.

Do not add new in-app items blindly.

For each proposed phase/task:
1. review it with the user;
2. decide whether it belongs in Tillsyn now;
3. decide whether it is:
   - onboarding/setup,
   - real dogfood execution,
   - template/product follow-up,
   - or documentation cleanup.
