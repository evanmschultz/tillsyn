# Tillsyn — Project CLAUDE.md

`main/` is the `main`-branch checkout — real coding/building/testing/committing happen here. Orchestrators launch from this directory. CLAUDE.md is ta-managed (`agents_md` schema, H2 = record); edit via `mcp__ta__*`, not raw Edit/Write.

## Hard Rules (Inviolable)

Inviolable across every session/drop/agent/surface. Add freely; removing needs dev sign-off.

- **Tillsyn is the work-tracking source of truth.** Cascade state (action_items, comments, QA verdicts) lives in Tillsyn — never chat-only or MD-checklist. Built-in `TaskCreate`/`TaskUpdate` are fine for a subagent's granular sub-steps or tiny orch reminders, but they evaporate on compaction — anything durable goes in a Tillsyn child action_item. [`feedback_todowrite_tillsyn_dual_use`]
- **Mage targets only for Go gates.** Never raw `go test/build/vet/run`, `gofmt`, `gofumpt`. Always `mage <target>`; if a target is missing, ADD it — never bypass. [see Build Verification]
- **bin/sh dispatch is the sanctioned interim surface.** `bin/agent-dispatch.sh` + `bin/agent-audit-toon.py` live in `main/` and ARE used for agent dispatch until Tillsyn's Go adapter framework (`internal/app/dispatcher`) — built FROM these proven bin/sh patterns — is complete. Keep them current; do not remove. The Go adapter supersedes them only once it dogfoods the same chains end-to-end.
- **No arbitrary-argv knobs on `BindingResolved`.** Templates declare `cli_kind`; adapters encapsulate argv (REV-1 killed `Command`/`ArgsPrefix`). New CLI families = new adapters, not template argv.
- **Cascade = plan down, build up, recurse on atomicity** (full spec: `CASCADE_METHODOLOGY.md §Plan Down, Build Up` — follow it, don't re-derive). The recursive flow, captured here so it is never missed:
  - **Plan top-down.** Each planner decomposes its scope into a SMALL set of children and RECURSES: a non-atomic sub-scope becomes a `kind=plan` child dispatched to its OWN sub-planner (which decomposes again), all the way down until LEAF planners emit only atomic `kind=build` droplets. A planner NEVER flattens a big set of builds in one pass — push depth into sub-plans.
  - **No child cap; the only cap is atomic-droplet sizing, MEASURED not labelled.** A droplet = 1-2 small code blocks (≤80 LOC incl. tests, ≤3 files). A *code block* = one new/changed top-level production symbol OR one cohesive same-purpose edit cluster. Plan-QA-falsification COUNTS the distinct production symbols a droplet names (tests excluded) and FAILS the plan on any droplet ≥3 symbols / >80 LOC / >3 files — never trusts the planner's label; re-measures EVERY droplet on any amendment. A 3-block "droplet" is the anti-pattern → emit a `kind=plan` sub-plan. "One coherent concern" is NOT a budget exception.
  - **Tree is ASYMMETRIC.** Depth is per-branch. A shared type/interface needed early sits as a SHALLOW leaf with `blocked_by` edges from the deeper branches that consume it; unrelated branches nest deeper. Real cross-branch ordering = `blocked_by` (concrete shared file/package or must-exist-first symbol), never artificial nesting or forced serialization.
  - **Build bottom-up.** Atoms land first (parallel where `blocked_by` allows); integration/confluence nodes follow once inputs are green.
  - **Gates (per-branch, never a global phase barrier):** a node's plan-QA pair (proof + falsification) BOTH PASS before it launches child planners OR transitions any build to `in_progress`; `mage ci` + build-QA gate each build's completion. A plan-QA FAIL → wipe-and-replan that subtree.
  - **Parallelize everything not `blocked_by`** — sub-planners, plan-QA pairs, builders, build-QA pairs all in flight at once across code-independent branches. **Auto-advance:** drive the cascade to completion; never ask permission per tick. Stop ONLY for a genuine fork the spec/methodology/memory can't resolve, a hard blocker, a QA-FAIL needing a design ruling, or a destructive/outward action (push/PR/ingest).
- **ALWAYS parallelize FE + core** (`ui/**` vs `internal/**`/`cmd/**`) — disjoint, no lock contention. Serialize only on an explicit `blocked_by` naming a real cross-lane symbol. [`feedback_parallelize_unblocked_default`]
- **Multi-backend dogfood = cost-relief.** Route per the Agent Bindings table (Cascade Architecture); canonical spec `HYLLA_BIN.md §2`. [`project_multi_backend_dogfood_direction`]
- **Playwright MANDATORY for FE** at `http://localhost:34115` (Wails AssetServer w/ IPC bindings — NOT `:51428`, the bindings-less bare Astro). Required every FE spawn: navigate + snapshot + fullPage screenshot→`.playwright-mcp/` + `browser_console_messages level=error` (0) + computed-style token check + visible-error check (`[role="alert"],[data-tone="error"]` count — `createResource` swallows throws). `mage uiDev` running first. Not optional/deferrable; if tool-blocked, agent reports BLOCKED + orch runs it. [`docs/wails-e2e-playwright-best-practices-2026-05-22.md`]
- **Responsive-first FE** — mobile 375 / tablet 768 / desktop 1280+; mobile-first CSS + `@media min-width`; stil canonical breakpoint tokens (no Tillsyn-local values); Playwright `browser_resize` + screenshot at all three. [`feedback_responsive_first_fe`]
- **Subagent discipline (2026-05-27).** Planners + plan-QA: Hylla MANDATORY-PRIMARY (zero Hylla in `## Hylla Feedback` = FAIL). Builders + build-QA: `mage test-func <pkg> <Func>` ONLY (never `mage ci`/`test-pkg`/raw go/gofmt). Plan-QA: `mage test-pkg <pkg>` only. Closeout: `mage ci` once. Failure-attribution: a `mage test-*` error outside your `paths` → `BLOCKED-by-sibling-WIP` + stop. No self-rescoping (>80 LOC / >3 files / ≥3 symbols → BLOCKED for re-split; never partial-ship + grade COMPLETE). Plan-QA-falsification Rule 3.5: `hylla_node_full` every integration seam (~30 lines) hunting `// TODO`/`// DEFERRED`/"blocked on" — any active deferral = FAIL; plus family-existence checks. Closing comments MUST carry `## Hylla Feedback` + `## Tools Used` (every call by full name + `wc -l` LOC). Orch jq-audits EVERY agent EVERY time. [`feedback_subagent_scope_tightening` + `CASCADE_METHODOLOGY.md §Subagent Discipline`]

## Coordination Model

Cascades live in Tillsyn as action_item subtrees (the system of record). Pre-cascade MD-per-drop (`workflow/drop_N/*.md`) is historical scaffolding — kept in tree as audit, never extended.

- **Cascade** = level-1 node (`kind=plan`, directly under the project, `parent_id=""`). New titles use `CASCADE <NAME>`; legacy `DROP_<NAME>` stays historical. **Drop** = level-2+ vertical step. **Droplet** = `kind=build` leaf (declares `paths` + `packages` + acceptance prose).
- **Builder outputs** → `till.comment` on the build (Hylla feedback + verdict + files-touched + mage output). **QA verdicts** → `till.comment` on the QA twins. **Closeout** → `till.comment` on the cascade root + `kind=closeout`/`refinement` state moves. **Cross-cutting decisions** → `kind=discussion` (description = converged shape, comments = audit trail). **Dev action items** → `till.attention_item`, never MD rows.
- **Read `WIKI.md` + `PLAN.md` at session start + after every compaction** (CLAUDE.md auto-loads; those two don't).

## Tillsyn Project

- **Project ID**: `5d9b530c-b568-4830-9e16-058c957cfc05`
- **Slug**: `tillsyn`
- **Template**: none (fresh, template-free)
- **Hylla artifact**: `github.com/evanmschultz/tillsyn@main` (`@main` → latest ingest)

Projects have no `kind` column post-Drop-1.75; language/stack lives in project `metadata`.

## Cascade Architecture

Every non-project node is classified on three orthogonal axes: `kind` (what work), `metadata.role` (who), `metadata.structural_type` (where — `cascade | drop | segment | confluence | droplet`). Canonical vocabulary: `WIKI.md § Cascade Vocabulary` — never redefine. `cascade`-in-the-Go-enum tracks at action_item `62569299-6522-401e-a15b-c6f61e2dc609`; until it lands, level-1 uses `structural_type=drop` as placeholder.

### Closed 12-Kind Enum

```toon
kinds[12]{kind,purpose}:
  plan,planning-dominant — decomposes into children; nests infinitely; auto-creates plan-qa-* twins
  research,read-only investigation — compiles findings posts dies (no QA children)
  build,code-changing leaf — auto-creates build-qa-* twins; no further children
  plan-qa-proof,proof-completeness QA on a plan parent; blocked_by parent
  plan-qa-falsification,falsification QA on a plan parent; blocked_by parent
  build-qa-proof,proof-completeness QA on a build parent; blocked_by parent + post-build gates
  build-qa-falsification,falsification QA on a build parent; blocked_by parent + post-build gates
  closeout,drop-end coordination aggregation
  commit,commit action — template-triggered under plan at level ≥ 2
  refinement,perpetual / long-lived tracking umbrella
  discussion,cross-cutting decision park — description=converged shape comments=audit trail
  human-verify,dev sign-off hold point — attention items + checklist children no plan/QA
```

### Agent Bindings (Cascade Defaults)

Canonical routing = `HYLLA_BIN.md §2`; this table mirrors it — keep in sync. `codex-gpt5` rows dispatch via hermetic `codex exec`; `opus`/`sonnet`/`haiku` rows via the built-in Agent tool (OAuth). Builder = `haiku` + `sonnet` fallback for over-envelope droplets.

```toon
agent_bindings[13]{kind,agent_name,model,role,edits_code,axis}:
  plan_go,ta-go-planning,codex-gpt5,planner,no,go-decomposition
  plan_fe,ta-fe-planning,codex-gpt5,planner,no,fe-decomposition
  plan-qa-proof_go,ta-go-plan-qa-proof,opus,qa-proof,no,plan-axis
  plan-qa-falsification_go,ta-go-plan-qa-falsification,codex-gpt5,qa-falsification,no,plan-axis
  plan-qa-proof_fe,ta-fe-plan-qa-proof,opus,qa-proof,no,plan-axis
  plan-qa-falsification_fe,ta-fe-plan-qa-falsification,codex-gpt5,qa-falsification,no,plan-axis
  build_go,ta-go-builder,haiku,builder,yes,go-implementation
  build_fe,ta-fe-builder,haiku,builder,yes,fe-implementation
  build-qa-proof_go,ta-go-build-qa-proof,sonnet,qa-proof,no,build-axis
  build-qa-falsification_go,ta-go-build-qa-falsification,codex-gpt5,qa-falsification,no,build-axis
  build-qa-proof_fe,ta-fe-build-qa-proof,sonnet,qa-proof,no,build-axis
  build-qa-falsification_fe,ta-fe-build-qa-falsification,codex-gpt5,qa-falsification,no,build-axis
  closeout,ta-closeout,opus,closeout,no,post-build-wrap
```

QA personas are split by axis (plan vs build) AND language (go vs fe) — persona-per-axis, no in-prompt branching. Agent names resolve 3-tier: project `.tillsyn/agents/<group>/<name>.md` → user `~/.tillsyn/...` → embedded `internal/templates/builtin/agents/...`. Pre-cascade today: Claude Code `Agent` tool with `subagent_type` matching `.claude/agents/ta-*.md`. Hylla MCP READ-ONLY for all agents; FE personas treat Hylla as Go-only.

### Required Children + Gates

- `kind=plan` auto-creates `plan-qa-proof` + `plan-qa-falsification`; `kind=build` auto-creates `build-qa-proof` + `build-qa-falsification` (blocked_by parent). `research`/`discussion`/`closeout`/`refinement`/`human-verify` are standalone.
- **Parent-child**: parent can't reach terminal-success while any child is incomplete/`failed`/`blocked` (always-on).
- **`blocked_by`**: sibling + cross-drop ordering primitive. File-level + package-level locks auto-insert runtime `blocked_by` on conflict.
- **Atomicity**: see Hard Rules + `CASCADE_METHODOLOGY.md` (measured, not labelled).
- **Post-build gates** (between a build and its QA children): `mage ci` (fail → build `failed`) → commit → push (cascade-end) → Hylla reingest (cascade-end). See Build-QA-Commit Discipline.

## Action-Item Lifecycle

States: `todo` / `in_progress` / `complete` / `failed`.

- **Success**: `metadata.outcome="success"` + `completion_notes` → `complete`.
- **Failure**: `metadata.outcome="failure"` + `completion_notes` → `failed`.
- **Blocked**: `metadata.outcome="blocked"` + `metadata.blocked_reason` → report to orch + stop.
- **Supersede**: dev-only `till action_item supersede <id> --reason "..."` unsticks `failed → complete`.

No parent reaches terminal-success with any `failed`/`blocked` child (always-on).

## Paths and Packages

Planners set `paths` + `packages` at creation. Builders restrict edits to declared `paths`; reference-only material → `files`. Sibling builds sharing a `paths` file OR a `packages` import MUST carry explicit `blocked_by`; per-package compile collisions block at `in_progress` promotion via runtime `blocked_by`. Cross-ref: `WIKI.md § Atomic Drop Granularity`.

## Orchestrator + Subagent Roles

**Orchestrator** = the parent Claude Code session (plans, routes, delegates, cleans up; reads code + Hylla; creates action_items; spawns subagents). **Code-edit rule**: PREFERS cascade builders; MAY edit code directly for trivial fixes / mid-flight stabilization / NIT absorptions — run gates + commit per discipline even then.

**Subagent roles**: Builder (ONLY role editing source). QA proof/falsification (read-verify-comment-die; never edit; parallel pairs). Planning (decomposes; never edits). Research (built-in `Explore`).

```toon
surfaces[6]{tool,subagent_use,orch_use}:
  till.action_item,read+update own item,create/update + read state + move phases
  till.comment,result comments on own item,guidance before spawns + drop-end aggregation
  till.attention_item,never,inbox for human approvals + dev-action routing
  till.handoff,never,structured next-action routing
  till.auth_request,claim only,create+claim+approve (orch-self-approval for non-orch subagents)
  till.capture_state,never,re-anchor scope on session start/restart
```

Subagents do NOT use attention_items / handoffs / @mentions / downward signaling.

## Auth and Leases

**Orch PROVISIONS per-agent auth for every subagent — NEVER shares its own session tuple** (wrong attribution, no gating, connection-bound `auth_context_id` doesn't transfer). Canonical sequence = `project_steward_auth_bootstrap` S1→S2→S3:

1. Orch **create-on-behalf**: `till.auth_request(operation=create, acting_session_id=<orch>, acting_session_secret, acting_auth_context_id, principal_id=<agent>, principal_type=agent, principal_role=builder|qa|research, path=project/<id>, requested_ttl=72h)` → `request_id` + `resume_token`.
2. Orch **self-approve**: `till.auth_request(operation=approve, request_id, acting_session_id=<orch>, …, agent_instance_id=<orch>, lease_token=<orch>)`.
3. Spawn prompt hands the agent ONLY its `request_id` + `resume_token` (+ `principal_id`/`client_id`); the agent **claims** (own `auth_context_id`) + **self-issues** a `capability_lease` (role + narrower scope than the orch's project lease).

Spawn prompts MUST instruct: claim → issue lease → use own 5-tuple; and **NEVER `till.auth_request(operation=create)` to renew on a transient error — report BLOCKED**. Always set `wait_timeout` on create. One active session per scope level. Orch cleans up child sessions/leases at phase end; manually revoke stale ones via `operation=revoke`. **Always report the session ID + request ID to dev** on create/claim. Project-level `OrchSelfApprovalEnabled=false` is the total backstop.

## Build-QA-Commit Discipline

No build droplet is `complete` without per-droplet QA passing. Push + `gh run watch` + Hylla reingest are cascade-end only.

**Per-droplet**: build → QA proof + falsification (parallel, both pass; respawn builder + re-QA on fail) → `mage ci` green → commit (`git add` specific files, conventional one-line; no push).

**Cascade-end**: `mage ci` clean → push + `gh run watch --exit-status` (no ingest on red) → Hylla reingest from the remote.

**Hylla ingest invariants (inviolable)**: always `enrichment_mode=full_enrichment` (never `structural_only`); always from the GitHub remote (`github.com/evanmschultz/tillsyn@main`); never before push + green CI; only the cascade-orch calls `hylla_ingest` (subagents never).

Subagent closing comments carry `## Hylla Feedback` (each miss: Query / Missed-because / Worked-via / Suggestion, or "None"). Orch aggregates at cascade-end.

## Git Management (Pre-Cascade)

Orch + dev manage git manually until the dispatcher takes over commits. Clean git state for an action item's `paths` is a creation precondition (orch checks `git status --porcelain <paths>`; asks dev to clean if dirty).

**Post-merge cleanup** (after a cascade PR merges): `gh pr merge <N> --merge --delete-branch` (preserve history; NOT squash/rebase) → if local sync fails, verify server-side merge via `gh pr view <N> --json state,mergeCommit` then `git push origin --delete <branch>` → `cd main/` (NEVER clean up from inside the worktree being removed) → `git fetch && git pull --ff-only` → `git worktree remove <path>` (investigate before `--force`) → `git branch -D <branch>` → verify `git worktree list` + `git branch -a`. Commit/stash all working-dir changes before marking a cascade closed.

## Recovery After Session Restart

```toon
recovery_steps[5]{step,action}:
  1,till.capture_state(project_id=...) to re-anchor scope
  2,till.attention_item(operation=list, all_scopes=true) for inbox
  3,check in_progress tasks for staleness
  4,revoke orphaned auth sessions/leases
  5,resume from current action-item state
```

## Skill and Slash Command Routing

```toon
commands[4]{command,when_to_use}:
  /plan-from-hylla,Hylla-grounded planning
  /qa-proof,Proof-oriented QA
  /qa-falsification,Falsification-oriented QA
  semi-formal-reasoning,Explicit reasoning certificate for semantic/high-risk work
```

## Section 0 Response Shape

Every substantive response opens with `# Section 0 — SEMI-FORMAL REASONING`. Orchestrator-facing: 5 passes (Planner / Builder / QA Proof / QA Falsification / Convergence). Subagent-facing: 4 (Proposal / QA Proof / QA Falsification / Convergence). Each pass uses the 5-field certificate (Premises / Evidence / Trace or cases / Conclusion / Unknowns). **Canonical spec: `~/.claude/CLAUDE.md § Semi-Formal Reasoning`.** Three project rules: (1) on every substantive response (trivial lookups exempt); (2) stays in the orchestrator-facing response ONLY — never in any Tillsyn `description`/`metadata.*`/`completion_notes`/comment/handoff; (3) subagent spawn prompts carry the directive verbatim (subagents don't inherit CLAUDE.md).

## Evidence and Code Understanding

Evidence order: (1) **Hylla** (`hylla_search`/`hylla_node_full`/`hylla_search_keyword`/`hylla_refs_find`/`hylla_graph_nav`) — primary for committed Go; exhaust every search mode before LSP/Read/Grep/Glob; record every miss in `## Hylla Feedback`. (2) **`git diff`** — files changed since last ingest (Hylla is stale for those). (3) **Context7 + `go doc` + gopls/`LSP`** — external/language/tooling semantics + live/uncommitted symbols.

Non-Go code (markdown/TOML/YAML/magefile/SQL): `Read`/`Grep`/`Glob`/`Bash` directly. **Hylla indexes Go only.**

## Project Structure

```toon
packages[8]{path,purpose}:
  cmd/till,CLI/TUI entrypoint + MCP server
  internal/domain,core entities and invariants
  internal/app,application services + use-cases (hexagonal core)
  internal/adapters/storage/sqlite,SQLite persistence
  internal/adapters/server/mcpapi,MCP handler
  internal/config,TOML loading + defaults + validation
  internal/platform,OS-specific paths
  internal/tui,Bubble Tea / Bubbles / Lip Gloss
```

`magefile.go` at repo root is canonical build/test automation. `ui/` is the Wails+Astro+Solid desktop FE.

## Tech Stack

```toon
stack[10]{component,version_or_lib}:
  language,Go 1.26+
  tui_framework,Bubble Tea v2 + Bubbles v2 + Lip Gloss v2
  sqlite,modernc.org/sqlite (no CGO)
  toml,github.com/pelletier/go-toml/v2
  cli,Fang + Laslig
  logger,github.com/charmbracelet/log
  fe_framework,SolidJS + Astro
  fe_host,Wails v2 (ui/main.go + ui/wails.json)
  fe_dev_port,localhost:34115 (Wails AssetServer w/ window.go IPC bindings — Playwright target; :51428 is bindings-less bare Astro)
  fe_pkg_manager,pnpm (pinned via packageManager field)
```

## Dev MCP Server

Test against `tillsyn-dev` (or a worktree-specific MCP name pointing at that worktree's built binary). Setup: `CONTRIBUTING.md § Dev MCP Server Setup`.

## ta MCP — Structured MD Editing

`ta` exposes MD files as schema'd records. Use `mcp__ta__*` (NOT raw Edit/Write) for any MD with a registered schema. Schema: `<project>/.ta/schema.toml` — registers `agents_md` (**CLAUDE.md + AGENTS.md**, H2-section records), `contributing`, `claude_agents` (`.claude/agents/*.md`), + cascade dbs (`discussions`/`plans`/`project`). If `mcp__ta__*` errors with "index missing", run `ta index rebuild` once.

```toon
ta_tools[7]{tool,purpose}:
  mcp__ta__schema,inspect/mutate the resolved schema
  mcp__ta__list_sections,enumerate record ids under a scope
  mcp__ta__get,read record(s) by id (raw or fields)
  mcp__ta__create,create a new record (fails if id exists)
  mcp__ta__update,PATCH-style partial overlay + atomic re-validation
  mcp__ta__delete,remove a record by id OR whole file by prefix+force
  mcp__ta__search,structured + regex search across records
```

Workflow: `list_sections` → `get` the section id → `update` the `body` field. The H2 heading IS the id (`CLAUDE.section.<slug>`). `ta` is in `.mcp.json` (`--project /abs/path`); permissions in `.claude/settings.json` (machine-local).

## Build Verification

Before any `build` action item is `complete`: all relevant mage targets pass; **NEVER raw Go toolchain** (always `mage <target>`; fix a buggy target, never bypass); template QA subtasks complete. `mage install` is orch-only (install, not verification). Coverage <70% is a hard failure. Run `mage ci` before push.

Canonical 12-target shape (P6 — shared across all sibling projects so agents always know the gate name):

```
TestFunc(pkg, fn)  builder + build-QA   go test -run "^<Func>$" -count=1 -race <pkg>
TestPkg(pkg)       plan-QA read-only    go test -count=1 <pkg>
Test               closeout/orch        go test ./...
RacePkg(pkg)       build-QA             go test -race -count=1 <pkg>
Race               closeout/orch        go test -race ./...
FormatFile(file)   builder + build-QA   gofumpt -w <file>
Format             closeout/orch        gofumpt -w .
FormatCheck        ci                   gofumpt -l . && fail if non-empty
VetPkg(pkg)        builder + build-QA   go vet <pkg>
Vet                closeout/orch        go vet ./...
Tidy               orch-only            go mod tidy + git-diff --exit-code
CI                 closeout/orch        Sources + FormatCheck + Vet + (Race+Coverage) + Tidy + Build + Integration
```

Tillsyn-specific extras: `mage build`/`run`/`dev`/`install`/`testGolden`/`testGoldenUpdate`/`testIntegration`/`uiDev`/`uiBuild`/`ciUI`/`uiA11y`. Hyphenated aliases preserved (`check`, `ci-ui`, `test-pkg`, `test-func`, `race-pkg`, `vet-pkg`, `format-check`, `format-file`, `fmt`, `ui-dev`, `ui-build`, `ui-a11y`, `test-golden`, `test-golden-update`, `test-integration`).

## Go Development Rules

- **Hexagonal**, interface-first, dependency inversion. **TDD-first** where practical; smallest concrete design (no speculative abstraction). **Idiomatic Go** — naming, package structure, stdlib/third-party/local import grouping. Go doc comments on every top-level decl + method.
- **Errors**: wrap with `%w`, bubble at clean boundaries, log context-rich at adapter/runtime edges, don't swallow. **Logger**: `github.com/charmbracelet/log` (dev-mode logs to `.tillsyn/log/`). **Tests**: co-located `*_test.go`, table-driven, `-race` via mage.
- **After touching Go**: `mage ci` before handoff (also for `.github/workflows/` or `magefile.go` changes); after pushing to fix CI, `gh run watch --exit-status` until green. **Dependencies**: ask dev to run `go get`/module updates (no `GOPROXY=direct`/`GOSUMDB=off`/checksum bypass). **Context7** before any code + after any test failure. **Clarification**: goal-alignment questions first, then implementation detail.
