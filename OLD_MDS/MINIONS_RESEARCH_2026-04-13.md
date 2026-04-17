# Tillsyn Minions Research Notes

Date: 2026-04-13

This memo is intentionally not a build plan. It is a research snapshot for discussion before implementation.

## 1. Scope And Current Assumptions

- The older Stripe memo in the repo root is useful as background, but its bare-repo and per-task worktree assumptions are not the current target.
- The current target is: Tillsyn owns the trigger, Tillsyn knows the project root, and Tillsyn launches minions by running Claude headless from that project directory.
- The main product goal is not "more autonomous agents." The goal is stricter truthfulness: state transitions, deterministic checks, and failure routing should make it harder for an agent to claim work is done when it is not.
- We are not using Tillsyn as the planning surface for this discussion round. This file is the working artifact by explicit request.

## 2. Confirmed External Facts

### 2.1 Claude Code Headless And Agent Invocation

- Claude Code docs explicitly say the CLI path formerly called "headless mode" is now the Agent SDK CLI, and "`-p` and all CLI options work the same way."
- The CLI reference documents `--agent` as "Specify an agent for the current session."
- Together, those two docs confirm that a defined agent can be invoked directly in headless mode.
- Practical form:

```bash
cd /path/to/project && claude -p --agent my-agent "task prompt"
```

- For scripted usage, Anthropic recommends `--bare`, but `--bare` skips hooks, skills, plugins, MCP auto-discovery, memory, and `CLAUDE.md`. That is useful only if Tillsyn passes everything explicitly.

### 2.2 Claude Agent Capability Controls

- Claude agent frontmatter supports `tools`, `disallowedTools`, `permissionMode`, `skills`, `mcpServers`, `hooks`, `initialPrompt`, `memory`, and `isolation`.
- `tools` is an allowlist.
- `disallowedTools` is a denylist.
- `permissionMode: acceptEdits` auto-accepts file edits and common filesystem commands in the working directory or additional directories.
- `permissionMode: plan` gives read-only exploration.
- `initialPrompt` is prepended when the agent is run as the main session agent via `--agent`.

### 2.3 Claude Hooks Are Real, But They Are The Wrong Primary Trigger Surface

- Claude hooks can run automatically on events like `SessionStart`, `SubagentStart`, `SubagentStop`, `TaskCreated`, `Stop`, and tool permission events.
- Hooks can approve or deny permissions, inject context, and trigger shell commands.
- If Claude starts with `claude --agent <name>`, hook payloads include `agent_type`.
- This is useful for local guardrails inside one Claude run.
- This is not sufficient as the main automation surface for Tillsyn, because hooks fire inside Claude session lifecycle, not on Tillsyn `plan_item` state transitions from the TUI or another orchestrator.

### 2.4 Stripe Minions Signals

- Stripe published two posts on `stripe.dev` on February 9, 2026 and February 19, 2026:
  - `Minions: Stripe’s one-shot, end-to-end coding agents`
  - `Minions: Stripe’s one-shot, end-to-end coding agents—Part 2`
- The accessible Stripe summaries confirm:
  - Minions are one-shot coding agents.
  - They are responsible for more than a thousand merged PRs per week.
  - Humans still review the code.
- The most detailed accessible public description I found was InfoQ's March 20, 2026 writeup, which attributes the following to Stripe's posts:
  - Minions execute one-shot end-to-end tasks.
  - Blueprints split work between deterministic routines and agent loops.
  - The system evolved from an internal fork of Goose.
  - CI, tests, and static analysis are core reliability controls.
- I am treating the Stripe blog itself as the primary source for existence, dates, titles, and high-level framing, and the InfoQ writeup as secondary support for details that were not exposed in the blog body through the tooling available here.

### 2.5 Semi-Formal Reasoning Paper

- The paper is `Agentic Code Reasoning` by Shubham Ugare and Satish Chandra (`arXiv:2603.01896`).
- The abstract defines semi-formal reasoning as structured prompting that requires explicit premises, execution traces, and conclusions.
- The paper's key idea is directly relevant here: the reasoning artifact acts like a certificate, making unsupported claims harder to hide.
- That maps well to QA minions, completion evidence, and failure comments in Tillsyn.

## 3. Current Tillsyn Code Reality

### 3.1 The Fix-Prompt Work Is Already Moving The Right Foundations

- The current codebase already contains a real `failed` lifecycle state in `internal/domain/workitem.go`.
- `internal/app/service.go` already:
  - maps move-to-failed to `CapabilityActionMarkFailed`,
  - blocks transitions out of terminal states without override auth,
  - uses hidden default failed columns.
- `internal/app/attention_capture.go` already counts `FailedItems`.
- The local uncommitted diff adds `metadata.outcome` to task metadata and validates allowed values at the MCP adapter boundary in `internal/adapters/server/common/app_service_adapter_mcp.go`.

### 3.2 Tillsyn Already Has The Event Primitive We Need

- `internal/app/live_wait.go` defines reusable event types:
  - `auth_request_resolved`
  - `attention_changed`
  - `handoff_changed`
  - `comment_changed`
- `internal/adapters/livewait/localipc/broker.go` already provides a cross-process SQLite-backed broker.
- This is the correct substrate for "fire a minion when Tillsyn state changes."

### 3.3 Tillsyn Already Has A Good Internal Worker Pattern

- `internal/app/embedding_runtime.go` is the best existing precedent for minions.
- It already has:
  - durable pending/running/ready/failed state,
  - claim loops,
  - heartbeats,
  - recovery for expired claims,
  - logging,
  - success/failure transitions,
  - retry behavior.
- If minions are added, this runtime pattern should be copied or generalized rather than inventing a different orchestration style.

### 3.4 Templates Already Contain Part Of The Contract Surface

- `internal/domain/template_library.go` already carries:
  - child rules,
  - responsible actor kind,
  - editable/completable actor kinds,
  - required-for-parent-done flags.
- This is already enough to model "build-task done causes qa-proof and qa-falsification children to matter."
- What does not exist yet is a first-class template-level automation contract for "when state becomes X, run minion Y with gate set Z and parse output as contract K."

### 3.5 Auth Claim Enrichment Still Looks Thin

- `internal/app/auth_requests.go` currently returns `ClaimedAuthRequestResult{Request, SessionSecret, Waiting}`.
- That is not yet the richer bootstrap shape described in `TILLSYN_FIX_PROMPT.md`.
- If minions are ordinary external actors that mutate Tillsyn directly, D7 still matters.
- If Tillsyn becomes the sole controller of state mutation and Claude is only a subprocess worker, D7 becomes helpful but no longer a hard prerequisite for the first minion drop.

## 4. Main Architectural Conclusion

- Tillsyn should own minion orchestration.
- Claude should not own minion orchestration.
- Claude hooks are useful as agent-local guardrails, not as the authoritative workflow engine.

Why:

- Tillsyn is the surface where the state change happens.
- Tillsyn must react whether the state change came from the TUI, MCP, or a future orchestrator.
- Tillsyn already owns durable coordination state, event delivery, comments, handoffs, and attention items.
- Tillsyn can make the final state decision based on command exit status, gate results, and parsed minion output instead of trusting the minion's self-report.

This is the key correction to the older Stripe memo: the right import from Stripe is "blueprints combine deterministic routines and agent calls," not "copy Stripe's exact runtime topology."

## 5. What A Tillsyn Minion System Should Probably Be

### 5.1 Trigger Model

- Template says which kinds are minion-runnable.
- Entering `in_progress` on one of those kinds triggers a run.
- Parent completion is still governed by existing child/blocker/depends_on semantics.
- Read-only QA and research minions can run in parallel.
- Write-capable builder minions are a concurrency risk if they share one checkout.

### 5.2 Execution Model

- Tillsyn launches the minion from the project root:

```bash
cd /project/root && claude -p --agent go-builder-agent "..."
```

- Tillsyn should capture:
  - stdout and stderr,
  - exit code,
  - timestamps,
  - task id,
  - agent name,
  - model,
  - files changed summary,
  - gate results,
  - parsed structured outcome.

### 5.3 Output Contract

- Do not parse freeform prose if you can avoid it.
- Prefer structured outputs:
  - Claude `--json-schema` for the final result payload.
  - `--output-format stream-json` when event-level telemetry is needed.
- A minion result contract should at minimum include:
  - `outcome`
  - `summary`
  - `evidence`
  - `affected_artifacts`
  - `follow_up_needed`
  - `comment_body`

### 5.4 Deterministic Gates

- The template should define gate steps and parse rules.
- For this repo, gates likely include things like `mage test-pkg`, `mage ci`, or later Hylla ingest checks.
- Tillsyn, not Claude, should evaluate whether those gates passed.
- On failure, Tillsyn should:
  - mark the plan item `failed`,
  - set `metadata.outcome`,
  - add a comment with condensed failure detail,
  - raise an attention item for orchestrator and user channels when appropriate.

### 5.5 Suggested Data Model Additions

- A durable `minion_run` or `automation_run` table is likely needed.
- Minimum fields:
  - `run_id`
  - `project_id`
  - `plan_item_id`
  - `trigger_event`
  - `template_rule_id`
  - `agent_name`
  - `command_line`
  - `cwd`
  - `started_at`
  - `finished_at`
  - `status`
  - `parsed_outcome`
  - `stdout_excerpt`
  - `stderr_excerpt`
  - `comment_id`
  - `attention_item_id`

### 5.6 Suggested Template Contract Additions

- Each automatable kind likely needs something like:
  - `minion_trigger`: `on_enter_state=in_progress`
  - `minion_actor_kind`: `builder|qa|research`
  - `claude_agent_name`
  - `permission_profile`
  - `prompt_contract`
  - `result_schema`
  - `gate_steps`
  - `gate_parsers`
  - `on_success`
  - `on_failure`

I would keep this separate from child rules. Child rules describe hierarchy. Minion rules describe execution.

## 6. How This Connects To TILLSYN_FIX_PROMPT.md

### 6.1 Likely Prerequisites

- D1 `failed` state: required.
- D6 `metadata.outcome`: required.
- D9 failed children block parent done: required.
- D4 auth/session cleanup on terminal state: strongly recommended.
- D10 `affected_artifacts`: strongly recommended for QA minions.

### 6.2 Helpful But Not Strictly Required For First Minion Cut

- D7 auth claim enrichment is very helpful if minions operate as independent Tillsyn actors.
- D8 level-based signaling is not a first-cut requirement.
- D3 item-scoped override auth matters for orchestrator and human repair flows, but not necessarily for the first "Tillsyn-controlled subprocess runner" drop.

### 6.3 Important Design Fork

- If minions are treated as ordinary external actors that claim auth and mutate Tillsyn themselves, the fix-prompt auth work is central to the first implementation.
- If minions are treated as subprocesses controlled by Tillsyn, then Tillsyn can remain the sole writer of task state and thread surfaces, which lowers the first-cut auth burden substantially.

My current recommendation is the second option for the first implementation.

## 7. Suggested Implementation Order After The Fix-Prompt Work

### 7.1 M0: Decide The Safety Model

- Decide whether write-capable minions are allowed to share one checkout.
- If yes, serialize all builder minions per project.
- If no, add isolated project-managed working copies even if Tillsyn is launched from the project root.

### 7.2 M1: Non-Claude Automation First

- Add a template-driven automation runner for deterministic commands only.
- Trigger on state change.
- Run `mage` or other commands.
- Parse success/failure.
- Move item to `done` or `failed`.
- Emit comments and attention.

This validates the trigger, ledger, parser, and notification model before Claude is involved.

### 7.3 M2: Claude Read-Only QA Minions

- Add `qa-proof` and `qa-falsification` first.
- They are lower risk because they should not edit code.
- Feed them `affected_artifacts`, comments, and task details.
- Require structured output.

### 7.4 M3: Claude Builder Minions

- Only after the runner, ledger, gate parsing, and failure routing are trustworthy.
- Use a strict result schema and deterministic post-run gates.
- Tillsyn decides final state.

### 7.5 M4: Parent Rollup And Auto-Fanout

- Builder done triggers QA children in parallel.
- Parent becomes eligible for `done` only when required children are done and no failed child remains unresolved.

## 8. Major Risks

### 8.1 Parallel Writers In One Checkout

- This is the biggest technical risk in the current target shape.
- If two write-capable minions run concurrently in the same project directory, they can trample each other, invalidate diffs, and make gate results meaningless.
- Read-only QA can run in parallel much more safely than builders.

### 8.2 Overloading Templates With Parser DSL Complexity

- "Template defines how command responses are parsed" is the right direction.
- But a fully generic parser DSL will get complicated quickly.
- Prefer a small set of parser types first:
  - `exit_zero`
  - `json_schema`
  - `contains_text`
  - `mage_ci`
  - `hylla_ingest`

### 8.3 Claude Session Configuration Drift

- If minion behavior depends on project `.claude` auto-discovery, local config drift can change behavior.
- If minion behavior depends on `--bare`, Tillsyn must pass every required agent/config/skill explicitly.
- This needs one deliberate product decision, not an accidental mix.

### 8.4 Diff Truthfulness

- You explicitly want the git diff view to matter before the end of the phase.
- That means diff capture should probably become a run artifact or run-detail view, not just a final review step.
- Tillsyn should likely record at least `git diff --stat` and maybe a truncated patch snapshot per run.

## 9. Open Questions

- Should first-cut builder minions be serialized per project, or do you want Tillsyn-managed isolated working copies even though the launch point is still the project directory?
- Do you want minion definitions to live primarily in Tillsyn templates, in `.claude/agents`, or in both with one as the source of truth?
- For the first cut, should Tillsyn be the only writer of task state and comments after a minion run, or do you want minions to call Tillsyn directly as authenticated actors?
- Should read-only QA minions ship before builder minions?
- Do you want gate parsing to begin with a small typed set of parsers, or do you want a more general template DSL immediately?
- Should `mage ci` be a post-run gate only, or can templates define pre-run, mid-run, and post-run command stages?
- Do you want a run-detail TUI surface from the first minion drop, especially for diff, stdout, stderr, and parsed result?
- Do you want minions to depend on project-local Claude config discovery, or should Tillsyn construct a deterministic explicit Claude invocation for every run?

## 10. Source Links

- Stripe primary:
  - https://stripe.dev/blog/minions-stripes-one-shot-end-to-end-coding-agents
  - https://stripe.dev/blog/minions-stripes-one-shot-end-to-end-coding-agents-part-2
  - https://stripe.dev/blog
- Claude Code primary:
  - https://code.claude.com/docs/en/headless
  - https://code.claude.com/docs/en/cli-usage
  - https://code.claude.com/docs/en/sub-agents
  - https://code.claude.com/docs/en/hooks
- Semi-formal reasoning paper:
  - https://arxiv.org/abs/2603.01896
- Secondary support for Stripe details not exposed in the retrieved blog body:
  - https://www.infoq.com/news/2026/03/stripe-autonomous-coding-agents/
