# Tillsyn — Project CLAUDE.md

This file lives in `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/`. `main/` is the `main`-branch checkout — real coding, building, testing, and committing happen here. Orchestrators launch from this directory.

## Hard Rules (Inviolable)

Applies across every session, drop, agent, and surface in this project. Adding to this list is fine; removing requires explicit dev sign-off.

- **No human time estimates — use cascade-shape work estimates.** NEVER say "this will take 1-2 days" / "a few hours" / "a week of work" / "medium lift" / "should take a while." Agents run on a different clock than human devs; the framing is wrong AND annoying. Estimate in cascade-shape units: cascades, drops, segments, confluences, droplets, plans. Examples: "≈3 build droplets across 2 packages," "one plan-QA pair + 4 build-QA pairs," "one cascade with W1/W2 parallel sub-planners + 6 build droplets." Applies to chat responses, plan-item descriptions, agent prompts, Tillsyn comments, and every other surface. See `feedback_no_human_time_estimates.md` memory for anti-examples.
- **Tillsyn-only for work tracking.** No Claude Code built-in `TaskCreate` / `TaskUpdate` / `TaskList` / `TaskGet` / `TaskStop` / `TaskOutput` — they evaporate on compaction/restart. Finer granularity goes in child action items.
- **Mage targets only for Go gates.** Never `go test`, `go build`, `go vet`, `gofmt`, `gofumpt`. Always `mage <target>`. If a target is missing, ADD the target; never bypass.
- **No bash-dispatcher bridges in this repo.** Tillsyn's adapter framework (`internal/app/dispatcher/cli_adapter.go` + `RegisterAdapter` + per-CLI packages) is the dispatch surface. Do not ship `bin/agent-dispatch.sh`-style shell scripts inside `main/` or as adopter examples. Sandbox is declarative (template + `agents.toml` validators at `internal/templates/load.go`); process isolation is OS-level (PATH-shadowed shim, container).
- **No arbitrary-argv knobs on `BindingResolved`.** REV-1 supersession explicitly killed `Command []string` and `ArgsPrefix []string`. Templates declare `cli_kind`; adapters encapsulate argv. Do not reintroduce. New CLI families get new adapters, not template-supplied argv strings.
- **Atomicity is a planner-prompt concern, not dispatcher Go code.** Builders' droplet sizing is enforced structurally via `paths` + `packages` declarations + file/package lock manager (Drop 4a Wave 2), AND numerically via the planner prompt rule "**1-2 small code blocks per build droplet** (≤80 LOC incl. tests), declare paths + packages." When a sub-goal would EXCEED 1-2 blocks, the planner emits a `kind=plan` child instead of an oversize `kind=build` — recursion is the norm, not the exception (see `CASCADE_METHODOLOGY.md` "Plan Down, Build Up"). Do not bake numeric atomicity into Go code.
- **Multi-backend dogfood is the cost-relief mechanism.** Anthropic-only spend is unsustainable. Route `plan` + `*-qa-falsification` to Codex (gpt-5.x with reasoning-effort knobs); route `*-qa-proof` to claude-opus (specialist verification); route `build` + `commit` to claude-haiku (or `claude --bare` → ollama-localhost for tier-2 cheap-builder). See `project_multi_backend_dogfood_direction.md` memory for the full routing thesis + scope of `drop_4d_multi_backend`.
- **ALWAYS parallelize FE and core work — NEVER serialize without a real dependency.** Cross-lane work (FE in `ui/**` + core Go in `internal/**` / `cmd/**`) is disjoint by package and has zero file-level lock contention; the cascade's file/package lock manager (Drop 4a Wave 2) blocks same-lane sibling conflicts, NOT cross-lane work. Dispatch `ta-fe-builder` + `ta-go-builder` concurrently against unblocked droplets every cascade tick. The only legitimate serialization between FE and core is an explicit `blocked_by` edge naming a specific cross-lane symbol dependency (e.g. FE needs a new Go IPC method to exist before its component can call it). Default = parallel; serialization is the exception that requires justification. Same rule applies WITHIN a lane: anything not explicitly `blocked_by` another sibling runs concurrently. See `feedback_parallelize_unblocked_default.md` memory for the dev directive backing this.
- **Playwright MANDATORY for FE work.** Every fe-builder / fe-qa spawn prompt MUST require: `mcp__plugin_playwright_playwright__browser_navigate` to `http://localhost:34115` (Wails dev AssetServer with `window.go.main.App.*` IPC bindings injected — NOT `localhost:51428`, which is the bare Astro standalone WITHOUT bindings and produces false-PASS empty-state coverage), `browser_snapshot`, `browser_take_screenshot` (fullPage + saved to `.playwright-mcp/`), `browser_console_messages level=error` (0 errors required), `browser_evaluate` for computed-style token verification, AND a visible-error check via `[role="alert"], [data-tone="error"]` element count (SolidJS `createResource` swallows throws silently — console-error count alone is insufficient). `mage uiDev` (→ `wails dev`) MUST be running before any browser_navigate. NOT optional. NOT deferable to dev. If subagent tool allowlist blocks Playwright MCP, agent reports BLOCKED and orch runs the verification itself — never fabricated, never silently skipped. Dev called this out 2026-05-21 after multiple agents skipped visual verification. Full Wails+Playwright methodology + Chromium-vs-WKWebView fidelity caveat at `docs/wails-e2e-playwright-best-practices-2026-05-22.md`.
- **Responsive-first FE — mobile + tablet + desktop breakpoints from day one.** Per `feedback_responsive_first_fe.md`. Desktop Wails users resize their window freely; the layout MUST adapt at standard breakpoints (mobile 375x667, tablet 768x1024, desktop 1280x800+). Build mobile-first CSS, layer wider rules via `@media (min-width: ...)`. NavRail collapses to bottom-tabs / horizontal strip at narrow widths. Topbar drops subtitle at narrow widths. Use stil's canonical breakpoint tokens (from `/Users/evanschultz/Documents/Code/hylla/stil/main/src/styles/tokens.css`) — do NOT invent Tillsyn-local breakpoint values. Playwright verification MUST include `browser_resize` at all three breakpoints + screenshot at each. Why: (a) real desktop UX (resize-friendly is table stakes), (b) cross-platform leverage (patterns built here inform future `stil-swift` iOS + Android native ports — Hylla's design-system paradigm is "stil = canonical tokens; per-platform adapters render"). drop_fe_3 trimmed stil's mobile patterns from `global.css` (172-line vendored subset vs 708-line upstream); the first follow-up FE drop should restore those patterns by re-vendoring the full file.
- **Plan-QA twins MUST close BEFORE sibling build droplets start.** Cascade discipline: plan-QA-proof + plan-QA-falsification on the cascade root (or any nested plan node) run to completion (with revision rounds if needed) BEFORE any `kind=build` child transitions to `in_progress`. drop_4d_codex's whole shipped-but-not-wired failure was caught only post-hoc because plan-QA fired AFTER 8 droplets shipped. The methodology spine is "plan down, build up" — plan-QA gates the build phase.

## Coordination Model

**Cascades live in Tillsyn as action_item subtrees.** The MD-per-drop pattern (`workflow/drop_N/PLAN.md`, `BUILDER_WORKLOG.md`, `CLOSEOUT.md`) was pre-cascade scaffolding. Drop 2 closed long ago; work-state lives in Tillsyn as the system of record. Dogfooding Tillsyn means **using Tillsyn for work tracking**, full stop.

- **Cascade = level-1 node.** A cascade is the whole tree of work that lives directly under the project. Going forward, new cascade titles use `CASCADE <NAME>` prefix (e.g. `CASCADE FOO BAR`); existing `DROP_<NAME>` titles stay historical and are not renamed.
- **Root of a cascade** → `kind=plan`, `structural_type=cascade` action_item directly under the project (`parent_id=""` — the project is NOT modeled as a parent action_item; level-1 means empty parent_id). Template auto-creates `plan-qa-proof` + `plan-qa-falsification` children. Until the Go enum work at Tillsyn action_item `62569299-6522-401e-a15b-c6f61e2dc609` lands, level-1 still uses `structural_type=drop` as a placeholder.
- **Drop = level-2+ vertical step.** "Drop" describes a vertical decomposition step inside a cascade. Most planning emits drops as planner children of the cascade root.
- **Droplet rows** → `kind=build` action_items as descendants of the cascade root. Each declares `paths`, `packages`, description prose (acceptance criteria + role + scope). Template auto-creates `build-qa-proof` + `build-qa-falsification` children per build.
- **Builder outputs** → `till.comment` on the build action_item. Includes Hylla feedback section, build verdict, files-touched list, `mage` output.
- **QA round verdicts** → `till.comment` on the QA twin action_items (proof + falsification).
- **Cascade-end closeout** → `till.comment` on the cascade root + state moves on `kind=closeout` / `kind=refinement` action_items addressing per-cascade aggregation.
- **Cross-cutting decisions** → `kind=discussion` action_item: description = converged shape, comments = audit trail of dev quotes.
- **Dev action items** → `till.attention_item` addressed to the dev. NOT MD checklist rows.
- Do NOT use Claude Code's built-in `TaskCreate` / `TaskUpdate` / `TaskList` / `TaskGet` / `TaskStop` / `TaskOutput` — they evaporate on compaction/restart. Finer granularity goes in child action_items.
- **Read `WIKI.md` + `PLAN.md` at session start and after every compaction.** CLAUDE.md auto-loads; those two do not.
- **Existing `workflow/drop_N/` MD directories stay in tree as historical audit** per `feedback_never_remove_workflow_files.md`. Do NOT create new MD content for new cascades — Tillsyn-native is the system of record going forward.

## Tillsyn Project

- **Project ID**: `5d9b530c-b568-4830-9e16-058c957cfc05`
- **Slug**: `tillsyn`
- **Template**: none (fresh project, template-free)
- **Hylla artifact**: `github.com/evanmschultz/tillsyn@main` (Hylla resolves `@main` to latest ingest)

Projects have no `kind` column post-Drop-1.75. Language/stack info lives in `metadata` on the project.

## Cascade Architecture

Cascades live as Tillsyn action_item subtrees. Every non-project node is classified along three orthogonal axes: `kind` (what work), `metadata.role` (who does it), `metadata.structural_type` (where it sits — `cascade | drop | segment | confluence | droplet`). Canonical vocabulary lives in `WIKI.md § Cascade Vocabulary` — never redefine. The 5th value `cascade` is the level-1 unit; `drop` is the level-2+ vertical step. Adding `cascade` to the Go `StructuralType` enum tracks at action_item `62569299-6522-401e-a15b-c6f61e2dc609`; until that lands, level-1 nodes carry `structural_type=drop` as a placeholder.

### Closed 12-Kind Enum

```toon
kinds[12]{kind,purpose}:
  plan,planning-dominant — decomposes work into children; nests infinitely; auto-creates plan-qa-* twins
  research,read-only investigation — agent compiles findings posts dies (no QA children)
  build,code-changing leaf — auto-creates build-qa-* twins; cannot contain further children
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

Pre-cascade: orchestrator spawns these via the `Agent` tool with Tillsyn auth in the prompt. Post-Drop-4a: dispatcher reads template bindings + spawns on `in_progress` transition.

```toon
agent_bindings[13]{kind,agent_name,model,role,edits_code,axis}:
  plan_go,ta-go-planning,opus,planner,no,go-decomposition
  plan_fe,ta-fe-planning,opus,planner,no,fe-decomposition
  plan-qa-proof_go,ta-go-plan-qa-proof,opus,qa-proof,no,plan-axis
  plan-qa-falsification_go,ta-go-plan-qa-falsification,codex-gpt5,qa-falsification,no,plan-axis
  plan-qa-proof_fe,ta-fe-plan-qa-proof,opus,qa-proof,no,plan-axis
  plan-qa-falsification_fe,ta-fe-plan-qa-falsification,codex-gpt5,qa-falsification,no,plan-axis
  build_go,ta-go-builder,sonnet,builder,yes,go-implementation
  build_fe,ta-fe-builder,sonnet,builder,yes,fe-implementation
  build-qa-proof_go,ta-go-build-qa-proof,sonnet,qa-proof,no,build-axis
  build-qa-falsification_go,ta-go-build-qa-falsification,codex-gpt5,qa-falsification,no,build-axis
  build-qa-proof_fe,ta-fe-build-qa-proof,sonnet,qa-proof,no,build-axis
  build-qa-falsification_fe,ta-fe-build-qa-falsification,codex-gpt5,qa-falsification,no,build-axis
  closeout,ta-closeout,haiku,closeout,no,post-build-wrap
```

**8-persona QA split (2026-05-21)**: per dev directive, the QA personas are SPLIT by axis (plan vs build) AND by language (go vs fe). Each persona's body focuses on its own axis — `ta-go-plan-qa-proof.md` carries plan-decomposition + parallelization-graph + Specify-block proof rules; `ta-go-build-qa-proof.md` carries acceptance-criteria + KindPayload-vs-diff + mage-gate proof rules. Same shape for falsification + FE. No more in-prompt branching; persona-per-axis is the canonical shape.

Agent names resolve via 3-tier walk: project `.tillsyn/agents/<group>/<name>.md` → user `~/.tillsyn/agents/<group>/<name>.md` → embedded `internal/templates/builtin/agents/<group>/<name>.md`. Pre-cascade today: Claude Code session uses `Agent` tool with `subagent_type` matching `.claude/agents/ta-*.md` names. Hylla MCP is READ-ONLY for all agents; FE personas apply the "Hylla = Go-only" doctrine (use normal tools for Astro / SolidJS / CSS / TOML).

### Required Children (Auto-Create)

- Every `kind=plan` auto-creates `plan-qa-proof` + `plan-qa-falsification` (blocked_by parent).
- Every `kind=build` auto-creates `build-qa-proof` + `build-qa-falsification` (blocked_by parent + post-build gates).
- `research`, `discussion`, `closeout`, `refinement`, `human-verify` are standalone — no auto-QA.

### Blockers + Atomicity

- **Parent-child**: parent cannot move to `complete` while any child is incomplete or `failed`. Always-on.
- **`blocked_by`**: sibling + cross-drop ordering primitive. File-level + package-level locks (Drop 4a Wave 2) auto-insert runtime `blocked_by` on conflict.
- **Atomicity**: planner prompt enforces "**1-2 small code blocks per build droplet** (≤80 LOC incl. tests), declare paths + packages." Sub-goals exceeding 1-2 blocks MUST be emitted as `kind=plan` children, NOT inlined as oversize `kind=build` droplets. Multi-level recursive decomposition is the norm — a 3-block "build droplet" is the anti-pattern. Plan-QA falsification attacks missing blockers + over-sized droplets + missing sub-plan recursion.

### Post-Build Gates (Between `build` And Its QA Children)

1. `mage ci` — on fail, build moves to `failed`. (Wave 0 wired into `.githooks/pre-push`; dev runs `mage install-hooks` once per fresh clone.)
2. Commit — commit-message-agent (haiku) forms message; orchestrator + dev run `git add` + `git commit` pre-cascade.
3. Push — `git push` when template `auto_push=true`. Pre-cascade: manual.
4. Hylla reingest — NOT per-build. Drop-end only, orchestrator-run, after `gh run watch --exit-status` green. Subagents never call `hylla_ingest`.

Only after all gates pass do the `build`'s QA children fire.

## Action-Item Lifecycle

Four states: `todo`, `in_progress`, `complete`, `failed` (Drop 1 landed `failed` as a real terminal state).

- **Success**: `metadata.outcome="success"` + `completion_notes` + move to `complete`.
- **Failure**: `metadata.outcome="failure"` + details in `completion_notes` + move to `failed`.
- **Blocked**: `metadata.outcome="blocked"` + `metadata.blocked_reason` + report to orch + stop.
- **Supersede**: dev-only CLI `till action_item supersede <id> --reason "..."` unsticks `failed → complete`.

No parent reaches terminal-success if any child is `failed` or `blocked` (always-on invariant).

## Paths and Packages

Wave 1 of Drop 4a landed `paths []string`, `packages []string`, `files []string`, `start_commit string`, `end_commit string` as first-class fields on every ActionItem. Planners set `paths` + `packages` at creation. Builders restrict edits to declared `paths`; reference-only material goes in `files`.

Sibling builds sharing a file in `paths` OR a package in `packages` MUST have explicit `blocked_by`. Per-package compile collisions block at `in_progress` promotion via runtime `blocked_by` insertion. Cross-reference: `WIKI.md § "Atomic Drop Granularity"`.

## Orchestrator + Subagent Roles

**Orchestrator** = the parent Claude Code session launched by the dev from this directory. Plans, routes, delegates, cleans up. Reads code + Hylla for research. Creates Tillsyn action items. Spawns subagents. Coordinates results.

**Code-edit rule**: orchestrator PREFERS cascade subagents for code changes (cascade enforces atomic-droplet sizing + plan-QA + asymmetric build-QA). Orchestrator MAY edit Go (or other) code directly when cascade adds overhead without value: trivial typo fixes, single-constant updates, mid-flight build-green stabilization, NIT-class absorptions surfaced by build-QA. When in doubt, prefer the builder. Even when editing directly, run verification gates (`mage ci`, etc.) and commit per build-QA-commit discipline.

**Subagent roles**:

- **Builder** — ONLY role that edits source code. Spawned via `Agent` tool with Tillsyn auth credentials in the prompt.
- **QA Proof / Falsification** — gated to QA. Read, verify, write closing comment, die. Never edit code. Run as parallel spawns (fresh context per pass).
- **Planning** — decomposes a drop into atomic build droplets with `paths`/`packages`/acceptance. Never edits code.
- **Research** — Claude's built-in `Explore` subagent for read-only investigation.

**Coordination surfaces**:

```toon
surfaces[6]{tool,subagent_use,orch_use}:
  till.action_item,read+update own item,create/update tasks + read state + move phases
  till.comment,result comments on own item,guidance before spawns + drop-end aggregation
  till.attention_item,never,inbox for human approvals + dev-action routing
  till.handoff,never,structured next-action routing
  till.auth_request,claim only,create+claim+approve (orch-self-approval for non-orch subagents)
  till.capture_state,never,re-anchor scope on session start / restart
```

Subagents do NOT use attention_items / handoffs / @mentions / downward-sideways signaling.

## Auth and Leases

- **The orchestrator PROVISIONS per-agent auth for every dispatched subagent — NEVER shares its own session tuple.** Sharing the orch's `session_id`/`session_secret`/`auth_context_id`/`agent_instance_id`/`lease_token` with subagents is a hard anti-pattern (wrong attribution, no real gating, connection-bound `auth_context_id` doesn't transfer). The system has the primitives to do this right (all implemented today): for each subagent the orch runs **create-on-behalf → orch-self-approve**, then hands the agent ONLY its `request_id` + `resume_token` (+ its `principal_id`/`client_id`) in the spawn prompt. The agent then **claims** its own session (own connection-bound `auth_context_id`) and **self-issues** a `capability_lease` (role + `actionItem` scope, narrower than the orch's project lease to avoid the equal-scope overlap guard) for its own `agent_instance_id`/`lease_token`. Canonical sequence = `project_steward_auth_bootstrap` S1→S2→S3. Orch flow proved working 2026-05-24 (issued session `594e308e`).
  - Orch create-on-behalf: `till.auth_request(operation=create, acting_session_id=<orch>, acting_session_secret, acting_auth_context_id, principal_id=<agent>, principal_type=agent, principal_role=builder|qa|research, path=project/<id>, requested_ttl=72h)` → returns `request_id` + `resume_token`.
  - Orch approve (self-approval gate): `till.auth_request(operation=approve, request_id, acting_session_id=<orch>, acting_session_secret, acting_auth_context_id, agent_instance_id=<orch>, lease_token=<orch>)` → issues the agent's session.
  - Spawn prompt MUST instruct: claim → issue lease → use own 5-tuple; and **NEVER call `till.auth_request(operation=create)` to renew on a transient error — report BLOCKED instead** (prevents the pending-request pile-up + self-renewal hangs).
  - HEADLESS/dispatcher-spawned agents get this same provisioning from the system itself once `AuthBundle` (spawn.go, currently an empty Wave-3 stub) is populated — Track B `D-AUTH-INJECT` automates exactly the manual flow above. Until then the orch does it per dispatch.
- One active auth session per scope level at a time.
- Orchestrator cleans up child sessions + leases at end of phase/run.
- Auth auto-revoke on terminal state lands in Drop 4b. Pre-Drop-4b, manually revoke stale sessions via `till.auth_request(operation=revoke)`.
- Orchestrators approve their own non-orch subagent auth requests scoped within their lease subtree (Wave 3 of Drop 4a). Cross-orch + orch-spawning-orch approvals route through dev TUI.
- Project-level `OrchSelfApprovalEnabled = *false` is the total backstop (reverts ALL approves under that project to dev-TUI).
- **Always report the auth session ID + request ID to dev** when requesting or claiming auth.

## Build-QA-Commit Discipline

**No build droplet is `complete` without per-droplet QA passing.** Push + `gh run watch` + Hylla reingest are cascade-end only.

**Per-droplet**:

1. Build — builder subagent implements the droplet.
2. QA Proof + Falsification (parallel) — both must pass.
3. Fix — if either QA fails, respawn builder + re-run QA until green.
4. Commit — `git add` specific changed files, conventional-commit format. No push yet.

**Cascade-end**:

5. `mage ci` locally — must pass clean.
6. Push + `gh run watch --exit-status` — once for the whole cascade. No ingest on red CI.
7. Hylla reingest — cascade-end only, from the remote, `enrichment_mode=full_enrichment`.

**Subagent closing comments include `## Hylla Feedback` section.** Record each Hylla miss: Query + Missed because + Worked via + Suggestion. Or `None — Hylla answered everything needed.` if clean. Orchestrator aggregates at cascade-end.

**Hylla ingest invariants (inviolable)**:

- Always `enrichment_mode=full_enrichment`. Never `structural_only`.
- Always source from the GitHub remote (`github.com/evanmschultz/tillsyn@main`). Never from a local working copy.
- Never before `git push` + `gh run watch --exit-status` green.
- Only the cascade-orch calls `hylla_ingest`. Subagents never do.

## Git Management (Pre-Cascade)

Until the cascade dispatcher takes over commits, orchestrator + dev manage git manually. Clean git state for an action item's declared `paths` is a precondition for creation; orch checks `git status --porcelain <paths>` before creation and asks dev to clean up if dirty.

### Post-Merge Branch Cleanup

After a cascade PR merges:

1. `gh pr merge <N> --merge --delete-branch` — preserves cascade's commit history (NOT --squash / --rebase).
2. If local sync fails (another worktree has main checked out with uncommitted work), the server-side merge still succeeded — verify with `gh pr view <N> --json state,mergeCommit`. Then `git push origin --delete <branch>` to clean the remote.
3. `cd` into `main/` worktree — NEVER cleanup from inside the worktree being removed.
4. `git fetch origin && git pull --ff-only` in `main/`.
5. `git worktree remove /path/to/cascade/N` from `main/` or bare root. If refuses, INVESTIGATE before `--force`.
6. `git branch -D cascade/N` (or `drop/N` for historical branches).
7. Verify clean: `git worktree list` + `git branch -a`.

**Guardrail**: every cascade-orch MUST commit or explicitly stash all working-dir changes before marking the cascade closed. A stale worktree holding `main` with staged files is an anti-pattern.

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

Every substantive response begins with a `# Section 0 — SEMI-FORMAL REASONING` block. Orchestrator-facing: 5 passes (`Planner` / `Builder` / `QA Proof` / `QA Falsification` / `Convergence`). Subagent-facing: 4 passes (`Proposal` / `QA Proof` / `QA Falsification` / `Convergence`). Each pass uses the 5-field certificate: **Premises** / **Evidence** / **Trace or cases** / **Conclusion** / **Unknowns**.

**Canonical full spec lives in `~/.claude/CLAUDE.md § "Semi-Formal Reasoning — Section 0 Response Shape"`** (global, mirrored across all projects). This project file enforces three load-bearing rules:

1. Section 0 on every substantive response (skip only for trivial one-line lookups).
2. Section 0 stays in the orchestrator-facing response ONLY — never inside Tillsyn `description` / `metadata.*` / `completion_notes` / comments / handoffs / attention items.
3. Subagent spawn prompts MUST carry the Section 0 directive verbatim (subagents don't inherit CLAUDE.md).

## Evidence Sources

In order:

1. **Hylla** — committed repo-local code.
2. **`git diff`** — uncommitted local deltas / files changed since last ingest.
3. **Context7** + **`go doc`** + **gopls MCP** — external / language / tooling semantics.

## Code Understanding Rules

1. **All Go code**: use Hylla MCP (`hylla_search`, `hylla_node_full`, `hylla_search_keyword`, `hylla_refs_find`, `hylla_graph_nav`) as the primary source. Exhaust every search mode (vector + keyword + graph-nav + refs) before falling back to `LSP`, `Read`, `Grep`, `Glob`. **Record every Hylla miss** in the subagent's closing comment under `## Hylla Feedback`.
2. **Changed since last ingest**: use `git diff` — Hylla is stale for those files until reingest.
3. **Non-Go code** (markdown, TOML, YAML, magefile, SQL): use `Read`, `Grep`, `Glob`, `Bash` directly.
4. **External semantics**: Context7 + `go doc` + `LSP` for library / language questions the repo can't answer itself.
5. **`LSP` tool** (gopls-backed): symbol search, references, diagnostics, rename safety for live / uncommitted code. Auto-targets `main/`.

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

`magefile.go` at repo root is the canonical build/test automation. `ui/` is the Wails+Astro+Solid desktop FE.

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
  fe_dev_port,localhost:34115 (Wails AssetServer with window.go IPC bindings — canonical Playwright target; localhost:51428 is the bare Astro dev server without bindings)
  fe_pkg_manager,pnpm (pinned via packageManager field)
```

## Dev MCP Server

Test against `tillsyn-dev` (or worktree-specific MCP name for non-main worktrees). Each worktree gets a unique MCP entry pointing at its own built binary. Setup instructions in `CONTRIBUTING.md § "Dev MCP Server Setup"`.

## ta MCP — Structured MD Editing

`ta` is a tiny MCP server that exposes MD files as structured records with schemas. Use `mcp__ta__*` tools (NOT raw `Edit` / `Write`) when modifying MD files that have a ta schema registered. Schema lives at `<project>/.ta/schema.toml` — currently registers `contributing` (CONTRIBUTING.md sections) + cascade-tree dbs (`discussions`, `plans`, `project`). Adding schemas for CLAUDE.md / README.md / WIKI.md is a near-term task (sign-off required before schema edits land).

```toon
ta_tools[7]{tool,purpose}:
  mcp__ta__schema,inspect or mutate the resolved schema (db / type / field)
  mcp__ta__list_sections,enumerate record ids under a scope (file-parse order)
  mcp__ta__get,read one record by id (raw bytes or structured fields) or every record under a prefix
  mcp__ta__create,create a new record — fails if id exists — type required (db.type)
  mcp__ta__update,PATCH-style update of existing record — partial overlay + atomic re-validation
  mcp__ta__delete,remove a record by id OR whole file by id prefix
  mcp__ta__search,structured + regex search across records under a scope
```

**Workflow** for editing a ta-managed MD: `list_sections` to see the structure → `get` the section by id → `update` the body field with PATCH-style overlay. The bracket header IS the id (e.g. `[contributing.section-installation]` → id `contributing.section-installation`). Validation failures return structured JSON naming the field + rule that failed.

**Registration**: `ta` is in `.mcp.json` at project root with `--project /abs/path` arg pinned. Tool permissions in `.claude/settings.json` (machine-local — not in git). Run `claude mcp list` to verify after session restart.

**NOT ta-managed** today: any MD without an entry in `.ta/schema.toml`. Use `Read` / `Edit` / `Write` for those. Aspirational: migrate the load-bearing docs (CLAUDE.md, README.md, WIKI.md) to ta-schema management over the next drops so all MD edits flow through validated structured surfaces.

## Build Verification

Before any `build` action item is `complete`:

1. All relevant mage targets pass (`mage -l` for the list).
2. **NEVER raw Go toolchain** (`go test` / `go build` / `go run` / `go vet`). Always `mage <target>`. If a target has a bug, fix the target — don't bypass.
3. `mage install` is allowed for the orchestrator when dev needs the `till` binary refreshed locally. Build verification still uses `mage ci`; `mage install` is install not verification.
4. All template-generated QA subtasks completed.

Key targets: `mage run`, `mage build`, `mage test-pkg <pkg>`, `mage test-func <pkg> <func>`, `mage test-golden`, `mage test-golden-update`, `mage format`, `mage ci`, `mage uiDev`, `mage uiBuild`, `mage ciUI`. Run `mage ci` before push. Coverage below 70% is a hard failure.

## Go Development Rules

- **Hexagonal architecture**, interface-first boundaries, dependency inversion.
- **TDD-first** where practical. Ship small tested increments.
- **Smallest concrete design.** No abstraction for hypothetical future variation.
- **Idiomatic Go** — naming, package structure, import grouping (stdlib / third-party / local).
- **Go doc comments** on every top-level declaration and method.
- **Errors**: wrap with `%w`, bubble at clean boundaries, log context-rich failures at adapter/runtime edges, don't swallow.
- **Logger**: `github.com/charmbracelet/log` with styled console output. Dev-mode logs to `.tillsyn/log/`.
- **Tests**: `*_test.go` co-located, table-driven, behavior-oriented. `-race` via mage targets.
- **Mage discipline**: plain `mage <target>` from worktree root. No `GOCACHE=...` overrides or workspace-local cache dirs.
- **After touching Go code**: `mage ci` before handoff. For `.github/workflows/` or `magefile.go` changes: `mage ci` first. After pushing to fix CI: `gh run watch --exit-status` until green.
- **Dependencies**: ask dev to run `go get` / module updates in their own shell. No `GOPROXY=direct`, `GOSUMDB=off`, or checksum bypass.
- **Context7**: before any code, after any test failure.
- **Markdown-first authoring** for Tillsyn `description`, `summary`, `body_markdown`, comments.
- **Clarification**: when stuck, ask goal-alignment questions first, then implementation-detail questions.
