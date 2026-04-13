# Tillsyn — Project CLAUDE.md

## Tillsyn Is the System of Record

All work is tracked in Tillsyn. No exceptions.

- Never use markdown files for work tracking, coordination, worklogs, or execution state.
- Claude Code's built-in task tools (TaskCreate, TaskUpdate, etc.) are fine for local progress tracking within a conversation. Use them alongside Tillsyn — they complement each other. Tillsyn is the cross-session source of truth; built-in tasks are ephemeral per-conversation aids.
- Every piece of work gets a Tillsyn plan item before it starts.
- **When work starts on a plan item, move it to `in_progress` immediately** so the dev can see what is actively being worked on. Do not leave items in `todo` while working on them.
- PLAN.md and other markdown planning docs are frozen reference material, not live trackers.

## Tillsyn Project

- **Project ID**: `a0cfbf87-b470-45f9-aae0-4aa236b56ed9`
- **Template**: `default-go` (revision 3, scope global)
- **Slug**: `tillsyn`
- **Kind**: `go-project`

## Hylla Baseline

- **Artifact ref**: `github.com/evanmschultz/tillsyn@main`
- Hylla resolves `@main` to the latest ingest automatically. Do not track snapshot numbers or commit hashes here.

### Code Understanding Rules

1. **All Go code**: use Hylla MCP (`hylla_search`, `hylla_node_full`, `hylla_search_keyword`, `hylla_refs_find`, `hylla_graph_nav`) as the primary source of truth for committed code understanding. Do not use `cat`, `grep`, `Read`, or other file tools for Go code discovery or navigation when Hylla can answer the question. **If Hylla does not return the expected code on the first search, exhaust all Hylla search modes before falling back to standard tools**: try vector similarity search (`hylla_search` with `search_types: ["vector"]`), keyword search across content/summary/docstring fields (`hylla_search_keyword`), graph navigation (`hylla_graph_nav`), and reference lookup (`hylla_refs_find`). Only after multiple Hylla search strategies fail may you use `Read`, `Grep`, or `Glob` for Go code.
2. **Changed since last ingest**: if Go code has been modified since the last Hylla ingest (check via `git diff`), use `git diff` for those specific deltas. Hylla is stale for those files until reingest.
3. **Non-Go code** (markdown, TOML, YAML, magefile, templates, SQL, etc.): use normal tools (Read, Grep, Glob, Bash) freely. Hylla does not cover non-Go files.
4. **External semantics**: use Context7, `go doc`, and gopls MCP for library docs, language semantics, and tooling questions the repository itself cannot prove.
5. **gopls MCP**: use for symbol search, references, diagnostics, rename safety, and workspace understanding. gopls must target the active visible checkout, not the bare root.

### Build-QA-Commit Discipline

**CRITICAL: Code is NEVER committed or pushed without QA completing first.** The sequence is:

1. **Build** — builder subagent implements the increment.
2. **QA Proof** — `go-qa-proof-agent` verifies evidence completeness and design support.
3. **QA Falsification** — `go-qa-falsification-agent` actively tries to break the conclusion.
4. **Fix** — if QA finds issues, spawn another builder to fix. Repeat QA.
5. **Commit** — only after BOTH QA passes clear: `git add` the specific changed files, commit with conventional-commit format.
6. **Push** — `git push` to the remote so CI runs and the remote is current.
7. **Reingest Hylla** — **do NOT run `hylla_ingest` yourself.** Ask the dev to run the ingest. NEVER use `structural_only` mode — full enrichment is the only acceptable ingest. Wait for the dev to confirm ingest is complete before proceeding.
8. **Update Tillsyn** — update the plan item's checklist, metadata, and lifecycle state to reflect what happened. If it's not in Tillsyn, it didn't happen.
9. **Move on** — only after the dev confirms reingest and Tillsyn reflects the completed state do you proceed to the next task.

Do not batch commits. Do not defer pushes. Do not skip QA. Do not skip reingest. Do not claim completion in chat without Tillsyn reflecting it.

## Orchestrator-as-Hub Architecture

The parent Claude Code session is always the **orchestrator**. All other roles (builder, qa, research) are ephemeral subagents.

**CRITICAL: The orchestrator NEVER writes code.** The parent session must NEVER use Edit, Write, or any other tool to modify Go source files, test files, or production code. All code changes — every single one — go through a builder subagent spawned via the Agent tool. The orchestrator reads code for planning and research only. If you catch yourself about to edit a `.go` file from the parent session, stop and spawn a subagent instead.

### How It Works

1. **Orchestrator** (parent session) plans, routes, delegates, and cleans up. It does NOT implement. It does NOT edit code. It reads code and Hylla for research, creates Tillsyn plan items, spawns subagents, and coordinates results.
2. **Subagents** are ephemeral — they spawn, read their task, do work, update the task, die. Builder subagents are the ONLY actors that edit code.
3. **Task state is the signal.** When a subagent finishes, it moves the task to `done` or `failed` and puts results in task metadata. The orchestrator reads the task state to know what happened.
4. **No subagent polls or watches anything.** Subagents read their task details at spawn, execute, update, return.
5. **Only the orchestrator uses attention items** — for human approval requests and inter-orchestrator communication.

### Agent State Management — CRITICAL

**Every subagent MUST manage its own Tillsyn plan item state.** The orchestrator cannot move role-gated items (e.g., QA subtasks gated to `qa` role).

**Before spawning any subagent:**
1. Move the target plan item to `in_progress` if the orchestrator has permission. If not (role-gated), the agent prompt MUST instruct the subagent to move it themselves.
2. Include in the agent prompt: the Tillsyn task ID, auth credentials (session_id, session_secret, auth_context_id, agent_instance_id, lease_token), and explicit instructions to move state.
3. Include the Hylla artifact ref (`github.com/evanmschultz/tillsyn@main`) so the subagent can query Hylla for Go code understanding. Omit `snapshot` — Hylla resolves `@main` to the latest ingest automatically.

**Every subagent prompt MUST include these instructions:**
- "Move your Tillsyn task to `in_progress` immediately when you start work."
- "When done: update metadata with results, move to `done`."
- "If you find issues that need fixing: leave in `in_progress`, update metadata with findings, return to orchestrator."

**For QA subagents specifically:**
- QA subtasks are gated to the `qa` role. The orchestrator must request a `qa`-role auth session and pass those credentials to the QA agent.
- The QA agent moves its own subtask to `in_progress` at start and `done` on PASS.
- On findings that need fixes: leave in `in_progress`, report findings, orchestrator spawns builder to fix, then re-runs QA.

**If a subagent fails to update state:** The orchestrator must get auth as the appropriate role and fix the state before proceeding. This is a recovery path, not the normal flow — fix the agent prompts so it doesn't happen again.

## Task Lifecycle

Four lifecycle states:

- **`todo`** — work not started
- **`in_progress`** — work actively being done
- **`done`** — work completed successfully
- **`failed`** — work completed unsuccessfully (attempt made, didn't succeed) *(being implemented — D1)*

### Success and Failure

- On success: set `metadata.outcome: "success"`, update `completion_contract.completion_notes`, move to `done`.
- On failure: set `metadata.outcome: "failure"`, update notes, move to `failed`.
- On blocked: set `metadata.outcome: "blocked"` + `metadata.blocked_reason`, signal UP, move to `failed`, die.
- On supersede: orchestrator (with override auth) sets `metadata.outcome: "superseded"`, moves `failed` to `done`.

### Failure Handling

`failed` tasks stay with full context. The orchestrator creates a new task with failure context and `depends_on` pointing to the failed task. No plan item can be marked `done` if any child is `failed` — this applies at ALL hierarchy levels.

## Auth and Lease Lifecycle

- **One active auth session** per scope level at a time.
- Auth is **immediately revoked** when a task/level is marked `done` or `failed` *(being implemented — D4)*.
- Orchestrator cleans up ALL child auth sessions and leases at end of phase/run.
- Auth claim response includes contextual data for the scope level *(being implemented — D7)*.
- **Always report the auth session ID to the dev** when requesting or claiming auth via `till.auth_request`. The dev needs visibility into which auth sessions are active.

## Affected Artifacts Tracking

Builders and planners track which code they affect via `metadata.affected_artifacts` *(being implemented — D10)*:

```json
[{"path": "internal/domain/lifecycle.go", "symbols": ["LifecycleState"], "change_type": "modify"}]
```

- **Planners** set `change_type: "planned"` during planning.
- **Builders** update with `create`/`modify`/`delete` during implementation.
- **QA agents** receive parent build-task's `affected_artifacts` via claim response.

Cross-item context is the orchestrator's job — update dependent items' details before spawning agents.

## Ordering and Dependencies

- **`depends_on`** — planned prerequisite ordering at creation. Tillsyn enforces this.
- **`blocked_by`** + **`blocked_reason`** — dynamic runtime blockers discovered during execution.

## Coordination Surfaces

### Subagents (Builder, QA, Research)

- `till.plan_item` — read task details, update metadata, move state
- `till.comment` — post result comments on their task
- Signal UP to orchestrator if blocked *(being implemented — D8)*

Subagents do NOT use attention_items, handoffs, @mentions, or downward/sideways signaling.

### Orchestrators

- `till.plan_item` — create/update tasks, read state, move phases
- `till.comment` — post guidance before spawning subagents
- `till.attention_item` — check inbox for human approvals
- `till.handoff` — structured next-action routing
- Level-based signaling *(being implemented — D8)*
- `/loop` polling at 60-120s for attention items

## Role Model

- **Orchestrator** (parent session) — plans, routes, delegates, cleans up. Owns phase transitions. **NEVER edits code. NEVER writes to source files.** All code changes are delegated to builder subagents.
- **Builder** (subagent) — ephemeral. The ONLY role that edits code. Reads task, implements, updates task, dies.
- **QA** (subagent) — ephemeral. Reads task, reviews, updates task with verdict, dies.
- **Research** (subagent) — ephemeral. Reads task, gathers evidence, updates task, dies.
- **Human** — approves auth requests, reviews results, makes design decisions.

## Recovery After Session Restart

1. `till.capture_state` — re-anchor project and scope context
2. `till.attention_item(operation=list, all_scopes=true)` — inbox state
3. Check for `in_progress` tasks that may be stale
4. Revoke any orphaned auth sessions/leases
5. Resume from current task state

## Allowed Kinds

Only create plan items with these kinds from the `default-go` template:

`branch`, `branch-cleanup-phase`, `build-phase`, `build-task`, `closeout-phase`, `commit-and-reingest`, `decision`, `dogfood-refactor-phase`, `dogfood-refactor-task`, `go-project`, `note`, `phase`, `plan-phase`, `project`, `project-setup-phase`, `qa-check`, `refactor-phase`, `refactor-task`, `subtask`, `task`

## Claude Code Agents

These agents are available via the `Agent` tool with `subagent_type`:

| Agent | Subagent Type | Purpose |
|---|---|---|
| **Orchestration** | `orchestration-agent` | Tillsyn system of record, routing planning/QA/closeout through skills |
| **Builder** | `go-builder-agent` | Ephemeral builder — the ONLY role that edits code |
| **Planning** | `go-planning-agent` | Hylla-first planning grounded in committed code reality |
| **QA Proof** | `go-qa-proof-agent` | Proof-completeness check — verify evidence supports the claim |
| **QA Falsification** | `go-qa-falsification-agent` | Falsification attempt — actively try to break the conclusion |
| **Closeout** | `closeout-agent` | Coordinate QA, freshness, and final baseline updates |
| **Gopls Worktree** | `gopls-worktree-agent` | Keep gopls MCP pointed at the active visible checkout |

Additional inline roles (no separate subagent file):
- **research-agent** — uses Claude's built-in `Explore` subagent
- **commit-and-reingest-agent** — parent role via `/commit-and-reingest`

### QA Discipline

QA has two distinct, asymmetric passes — they are not duplicate reviewers:

- **QA PROOF REVIEW** (`go-qa-proof-agent`, `/qa-proof`) — verify evidence completeness, reasoning coherence, trace coverage.
- **QA FALSIFICATION REVIEW** (`go-qa-falsification-agent`, `/qa-falsification`) — counterexamples, hidden deps, contract mismatches, YAGNI.
- **QA Sweep** (`/qa-sweep`) — coordinate both passes for closeout.

Prefer subagents for QA when fresh-context isolation matters.

## Skill and Slash Command Routing

| Command | When to Use |
|---|---|
| `/tillsyn-bootstrap` | Tillsyn project setup checks |
| `/plan-from-hylla` | Hylla-grounded planning |
| `/qa-proof` | Proof-oriented QA |
| `/qa-falsification` | Falsification-oriented QA |
| `/qa-sweep` | Coordinated QA across both passes |
| `/commit-and-reingest` | Confirmed-good baseline updates (commit + push + Hylla reingest) |
| `/select-checkout` | Checkout selection in bare-root setup |
| `/gopls-sync` | gopls MCP hygiene after checkout changes |
| `semi-formal-reasoning` | Explicit reasoning certificate for semantic/high-risk work |

## Semi-Formal Reasoning

For semantic, high-risk, or ambiguous work, use this reasoning shape:

- **Premises** — what must be true
- **Evidence** — grounded in Hylla / `git diff` / Context7
- **Trace or cases** — concrete paths through the code
- **Conclusion** — the claim
- **Unknowns** — what remains uncertain, routed into Tillsyn

Keep certificates short and inspectable.

## Evidence Sources

Use these in order:

1. **Hylla** for committed repo-local code understanding (always use latest snapshot, filter `snapshot=<current>`).
2. **`git diff`** for uncommitted local deltas or files changed since last ingest.
3. **Context7** for external semantics. Also use `go doc` and **gopls MCP**.

## Project Structure

- `cmd/till`: CLI/TUI entrypoint
- `internal/domain`: core entities and invariants
- `internal/app`: application services and use-cases (ports-first, hexagonal core)
- `internal/adapters/storage/sqlite`: SQLite persistence adapter
- `internal/adapters/server/mcpapi`: MCP API handler layer
- `internal/config`: TOML loading, defaults, validation
- `internal/platform`: OS-specific config/data/db path resolution
- `internal/tui`: Bubble Tea/Bubbles/Lip Gloss presentation layer
- `.artifacts/`: generated local outputs
- `magefile.go`: canonical build/test automation

## Tech Stack

- Go 1.26+
- Bubble Tea v2, Bubbles v2, Lip Gloss v2
- SQLite (`modernc.org/sqlite`, no CGO)
- TOML config (`github.com/pelletier/go-toml/v2`)
- Laslig for Mage and CLI styling
- Fang for CLI help surfaces

## Agent Selection

This is a Go project. Use `go-*` agent variants:

- Builder: `go-builder-agent`
- QA Proof: `go-qa-proof-agent`
- QA Falsification: `go-qa-falsification-agent`
- Planning: `go-planning-agent`
- Closeout: `closeout-agent` (shared, lang-aware)

## Dev MCP Server

Every worktree needs a local dev MCP server pointing at its own built binary so changes can be tested against the dev version, not the installed one. This is mandatory — every branch gets its own dev binary and MCP server.

### Setup

```bash
# 1. Build the binary in the worktree
mage build

# 2. Add the dev MCP server scoped to the worktree
claude mcp add --scope local tillsyn-dev -- /path/to/worktree/till serve-mcp
```

### Current Dev Servers

- **main**: `tillsyn-dev` -> `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/till serve-mcp`

### Rules

- After every `mage build`, the dev binary is updated in place — the MCP server picks up changes on next invocation.
- When creating a new branch/worktree, `mage build` and `claude mcp add` are part of the setup before any testing begins.
- Always test against `tillsyn-dev`, not the installed `till` binary.
- When retiring a branch, remove its dev MCP server entry.

## Build Verification

Before any build-task can be marked done:

1. All mage verification targets pass (discover via `mage -l`).
2. Never use raw `go build`, `go test`, `go vet` — always mage targets.
3. All template-generated QA subtasks completed.

**CRITICAL: NEVER run `go test`, `go build`, `go run`, or any raw `go` toolchain command directly.** All Go operations go through `mage` targets. This applies to the orchestrator, builder subagents, QA subagents — everyone. No exceptions. If a `mage` target fails, investigate and fix the target or the code, do not bypass it with a raw `go` command.

- `mage run` — run from source
- `mage build` — build local binary `./till`
- `mage test-pkg <pkg>` — test a specific package
- `mage test-golden` / `mage test-golden-update` — golden fixture validation
- `mage ci` — canonical full gate (source verification, gofmt, coverage, build)

Run `mage ci` before push. Coverage below 70% is a hard failure.

## Go Development Rules

You are a senior Go dev. These rules are always active:

### Context7 and Documentation

- ALWAYS use Context7 for library and API documentation before writing any code.
- ALWAYS re-run Context7 after any test failure or runtime error before making the next edit.
- If Context7 is unavailable (quota, network, outage), record the fallback source before proceeding (e.g. official docs, `go doc`, or package-local docs).

### Tillsyn Instructions

- Use `till.get_instructions` on-demand (missing/stale/ambiguous guidance), not on every step.
- Keep context bounded: set `doc_names` explicitly, use `max_chars_per_doc` on long docs, use `include_markdown=false` for inventory checks and `true` only when full text is needed.

### Code Style and Idioms

- Hexagonal architecture, interface-first boundaries, dependency inversion.
- TDD-first where practical. Ship small, testable increments.
- Prefer smallest concrete design; no abstraction for hypothetical future variation.
- Idiomatic Go naming, package structure, import grouping (stdlib, third-party, local).
- Write idiomatic Go doc comments for all top-level declarations and methods in production and test code. Add inline comments for non-obvious behavior blocks (including in `*_test.go`).
- Treat all project/task details and thread comment content as markdown-first authoring surfaces.
- In MCP calls, write markdown-formatted content for `description`, `summary`, and `body_markdown` fields.

### Error Handling and Logging

- Wrap errors with `%w`.
- Return errors upward at clean boundaries.
- Log context-rich failures at adapter/runtime edges instead of swallowing errors.
- Use `github.com/charmbracelet/log` as the canonical logger for application/runtime logs.
- Keep colored/styled console output enabled for local developer ergonomics.
- In dev mode, write logs to `.tillsyn/log/` so logs are easy to inspect during debugging.
- Log meaningful runtime operations and failures (startup paths/config load, persistence/migrations, mutating actions, recoverable/non-recoverable errors).
- During troubleshooting, inspect recent local log files before proposing fixes and include relevant findings in reasoning.

### Build, Test, and Mage

- Review `magefile.go` at startup and use its targets as the source of truth for local automation.
- **NEVER run `go test`, `go build`, `go run`, or any raw `go` toolchain command.** Always use the corresponding `mage` target. If a `mage` target has a bug, fix the target — do not fall back to raw `go` commands.
- Run `mage` targets from the worktree root as plain `mage <target>` without `GOCACHE=...` or other cache-path env overrides unless the user explicitly asks.
- Do not create workspace-local ad-hoc Go cache directories (e.g. `.go-cache-*`).
- During implementation loops, run `mage test-pkg <pkg>` after meaningful increments.
- When you touch Go code, finish by running `mage ci` unless the user approves a narrower suite.
- Before asking the user to push or opening/refreshing a PR, run `mage ci` and report results.
- After pushing a change to fix or validate CI, run `gh run watch --exit-status` and do not claim CI passes until the remote run finishes green.
- If you touch `.github/workflows/` or `magefile.go`, run `mage ci` before handoff.
- Add package-scoped Mage targets only when they materially simplify the repo.
- Coverage below 70% is a hard failure.

### Testing Guidelines

- Tests are co-located as `*_test.go`.
- Prefer table-driven tests and behavior-oriented assertions.
- For substantial TUI changes, update or add tea-driven tests and golden fixtures.

### GitHub and Git

- Prefer `gh` for GitHub-hosted operations (PRs, workflow/check inspection, run logs, review actions, repo metadata, auth).
- Invoke `gh` directly without wrapping in `/bin/zsh -lc` layers.
- Use `git` for core local operations (status, diff, add, commit, branch, merge-base).
- Do not use the GitHub web UI when `gh` can do the same task.

### Git Commit Format

Use conventional-commit style: `type(scope): message`. All lowercase except proper nouns, acronyms, or terms that are conventionally capitalized (e.g. HTTP, TUI, WASM). Keep messages concise and human — describe what changed, not how.

Format: `type(scope): short message`

Types: `feat`, `fix`, `refactor`, `chore`, `docs`, `test`, `ci`, `style`, `perf`

Examples:
- `feat(ingest): add per-file progress reporting`
- `fix(tui): correct viewport wrap on narrow terminals`
- `chore(deps): update to charm/v2`
- `refactor(core): split parse and render phases`
- `docs(readme): add quickstart section`

No co-authored-by trailers. No period at the end. No capitalized first word after the colon unless it's a proper noun or acronym.

### Dependencies

- If dependency updates need network access, ask the user to run `go get` and module update commands in their own shell.
- Never use dependency-fetch bypasses (`GOPROXY=direct`, `GOSUMDB=off`, checksum bypass flags).

### Safety

- Never delete files or directories without explicit user approval.
- Never run commands outside the repository root: `/Users/evanschultz/Documents/Code/hylla/tillsyn`.
- Never push to any remote unless the user explicitly requests it in the current conversation.
- Keep secrets out of config files committed to the repository.

### Dogfooding

- For live-runtime dogfooding, project setup, auth setup, and operator workflow validation, use MCP surfaces by default unless the user explicitly asks to validate the CLI.
- For runtime/protocol validation, run MCP-only checks (no HTTP/curl validation probes).
- It is allowed to `mage build` and run `./till serve` locally for MCP-side validation.

### Clarification Protocol

- When clarification is needed, ask in two stages: first ask general goal-alignment questions and lock shared objectives, only then ask specific implementation-detail questions.

## Bare-Root and Worktree Discipline

- Bare repo at `/Users/evanschultz/Documents/Code/hylla/tillsyn` is the orchestration root, not a coding checkout.
- Real work happens in `/Users/evanschultz/Documents/Code/hylla/tillsyn/main` (or other visible worktrees).
- Always confirm and `cd` into the intended checkout before edits, tests, commits, or gopls work.
- After switching/creating/retiring a checkout, use `/gopls-sync`.

## Features Being Implemented (from TILLSYN_FIX_PROMPT.md)

These design decisions are confirmed and being built. Sections above marked *(being implemented)* reference these:

- **D1**: `failed` lifecycle state (fourth terminal state)
- **D2**: Task state as signal (not @mentions) for ephemeral subagents
- **D3**: Item-list-based short-TTL override auth (human-approved, scoped to specific items)
- **D4**: Auth revocation on terminal state
- **D5**: Task details are the prompt (via auth claim enrichment)
- **D6**: Standardized outcome in metadata
- **D7**: Auth claim response enrichment (bootstrap context in claim response)
- **D8**: Level-based signaling (UP/DOWN between orchestrators, UP-only for subagents)
- **D9**: `require_children_done` blocks on `failed` children at all levels
- **D10**: Affected artifacts tracking in `metadata.affected_artifacts`

See `TILLSYN_FIX_PROMPT.md` for full specifications.
