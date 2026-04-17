# Claude Code ToS Compliance — Cascade Plan Review

**Date:** 2026-04-14
**Status:** Initial analysis complete. Recommendations and forward options pending dev discussion.
**Scope:** Review of `PLAN.md` against Anthropic's Consumer Terms, Usage Policy, Agentic-Use guidance, Commercial Terms, Claude Code permission-modes documentation, and CLI reference — identify ToS-adjacent risks and sketch compliance-tightening options.
**Companion to:** `PLAN.md`.
**Tracked in Tillsyn:** plan item `3b4052ef-300d-42de-8901-e22cecc9bea0` (kind `task` pre-Drop-2, will be relabelled `drop` once that kind lands) — project `a5e87c34-3456-4663-9f32-df1b46929e30`. Description = converged shape; comments = audit trail.

---

## Table of Contents

1. [Summary](#1-summary)
2. [Documents reviewed, with verbatim quotes](#2-documents-reviewed-with-verbatim-quotes)
3. [Where the plan is clearly in bounds](#3-where-the-plan-is-clearly-in-bounds)
4. [Gray zones and open risks](#4-gray-zones-and-open-risks)
5. [Recommended plan adjustments](#5-recommended-plan-adjustments)
6. [Open design questions (for Tillsyn discussion)](#6-open-design-questions-for-tillsyn-discussion)
7. [Decision log](#7-decision-log)

---

## 1. Summary

The cascade plan in `PLAN.md` does not frontally violate Anthropic's Consumer Terms of Service, Commercial Terms of Service, Usage Policy, or the Agentic-Use support-article guidance. All of the flags the plan intends to use (`--bare`, `-p`, `--mcp-config`, `--strict-mcp-config`, `--permission-mode acceptEdits`, `--allowedTools`, `--max-budget-usd`, `--max-turns`) are documented, supported, and explicitly intended for headless / programmatic use. Multi-agent dispatch is explicitly supported — the Claude Code overview markets it as "Run agent teams and build custom agents."

However, several design choices in the current plan sit in a gray band where the ToS-compliance outcome depends on decisions the plan has not yet made:

1. The **account tier** the dispatcher authenticates as is never pinned down. The Consumer Terms permit automated access only "when you are accessing our Services via an Anthropic API Key or where we otherwise explicitly permit it" — headless Claude Code is explicitly permitted, but a long-running autonomous dispatcher spawning dozens of parallel processes is closer to the clause's intent than a one-shot `claude -p` in CI is.
2. The plan locks out Anthropic's **auto-mode classifier** (the one Anthropic-provided autonomous-safety layer) by running `acceptEdits` instead — and auto mode requires a Team / Enterprise / API tier that the plan never commits to.
3. The plan runs agents in the **live `main/` checkout** rather than an isolated container / VM. Anthropic's recommended posture for any loose permission mode is "isolated containers and VMs only."
4. The plan has **no cascade-run-level budget ceiling** and no explicit statement on training-data opt-out. Rate-limit failures are handled reactively (as `blocked` outcomes) but cost-runaway is not structurally bounded.

None of these are ToS violations today. All of them are decisions the plan should make explicitly before Drop 5 (dogfooding begins).

---

## 2. Documents reviewed, with verbatim quotes

### 2.1 Consumer Terms of Service — `https://www.anthropic.com/legal/consumer-terms`

Automation / non-human-access clause:

> "Except when you are accessing our Services via an Anthropic API Key or where we otherwise explicitly permit it, to access the Services through automated or non-human means, whether through a bot, script, or otherwise."

Training-opt-out language:

> "We may use Materials to provide, maintain, and improve the Services and to develop other products and services, including training our models, unless you opt out of training through your account settings."

### 2.2 Usage Policy — `https://www.anthropic.com/legal/aup`

Agentic-use passthrough:

> "Agentic use cases must still comply with the Usage Policy."

Guardrail-bypass prohibition:

> "Intentionally bypass capabilities, restrictions, or guardrails established within our products."

Coordination / circumvention:

> "Coordinate malicious activity across multiple accounts to avoid detection or circumvent product guardrails."

### 2.3 Agentic-Use support article — `https://support.claude.com/en/articles/12005017-using-agents-according-to-our-usage-policy`

The article enumerates prohibited outcomes (surveillance, phishing, scaled abuse, unauthorized system access) and does not address human-oversight requirements, autonomy bounds, or multi-agent dispatch protocols. Agentic use is permitted as long as the Usage Policy itself is respected:

> "All uses of agents and agentic features must continue to adhere to Anthropic's Usage Policy."

### 2.4 Commercial Terms of Service — `https://www.anthropic.com/legal/commercial-terms`

Competing-product restriction (§D.4):

> "Customer may not and must not attempt to (a) access the Services to build a competing product or service, including to train competing AI models or resell the Services except as expressly approved by Anthropic; (b) reverse engineer or duplicate the Services; or (c) support any third party's attempt at any of the conduct restricted in this sentence."

Training on customer content (§B):

> "Anthropic may not train models on Customer Content from Services."

### 2.5 Claude Code permission-modes — `https://code.claude.com/docs/en/permission-modes`

`acceptEdits`:

> "`acceptEdits` mode lets Claude create and edit files in your working directory without prompting. … In addition to file edits, `acceptEdits` mode auto-approves common filesystem Bash commands: `mkdir`, `touch`, `rm`, `rmdir`, `mv`, `cp`, and `sed`. … Paths outside that scope, writes to protected paths, and all other Bash commands still prompt."

`bypassPermissions` / `--dangerously-skip-permissions`:

> "`bypassPermissions` mode disables permission prompts and safety checks so tool calls execute immediately. Writes to protected paths are the only actions that still prompt. Only use this mode in isolated environments like containers, VMs, or devcontainers without internet access, where Claude Code cannot damage your host system."

> "`bypassPermissions` offers no protection against prompt injection or unintended actions. For background safety checks without prompts, use auto mode instead."

Auto-mode availability:

> "Auto mode is available only when your account meets all of these requirements: Plan: Team, Enterprise, or API. Not available on Pro or Max. … Model: Claude Sonnet 4.6 or Opus 4.6. Not available on Haiku or claude-3 models. Provider: Anthropic API only. Not available on Bedrock, Vertex, or Foundry."

Auto-mode rules dropped on entry:

> "On entering auto mode, broad allow rules that grant arbitrary code execution are dropped: Blanket `Bash(*)`, Wildcarded interpreters like `Bash(python*)`, Package-manager run commands, `Agent` allow rules. Narrow rules like `Bash(npm test)` carry over."

Auto-mode on subagents:

> "The classifier checks subagent work at three points: Before a subagent starts, the delegated task description is evaluated, so a dangerous-looking task is blocked at spawn time. While the subagent runs, each of its actions goes through the classifier with the same rules as the parent session, and any `permissionMode` in the subagent's frontmatter is ignored. When the subagent finishes, the classifier reviews its full action history; if that return check flags a concern, a security warning is prepended to the subagent's results."

Auto-mode fallback and headless interaction:

> "If the classifier blocks an action 3 times in a row or 20 times total, auto mode pauses and Claude Code resumes prompting. … In non-interactive mode with the `-p` flag, repeated blocks abort the session since there is no user to prompt."

Protected paths (always prompt in any mode):

> ".git, .vscode, .idea, .husky, .claude (except for .claude/commands, .claude/agents, .claude/skills, and .claude/worktrees) … .gitconfig, .gitmodules, .bashrc, .bash_profile, .zshrc, .zprofile, .profile, .ripgreprc, .mcp.json, .claude.json"

### 2.6 Claude Code CLI reference — `https://code.claude.com/docs/en/cli-reference`

Long-lived auth for CI / scripts:

> "`claude setup-token` — Generate a long-lived OAuth token for CI and scripts. Prints the token to the terminal without saving it. Requires a Claude subscription."

Headless flags used by the plan:

> "`--bare` — Minimal mode: skip auto-discovery of hooks, skills, plugins, MCP servers, auto memory, and CLAUDE.md so scripted calls start faster."
>
> "`--max-budget-usd` — Maximum dollar amount to spend on API calls before stopping (print mode only)."
>
> "`--max-turns` — Limit the number of agentic turns (print mode only). Exits with an error when the limit is reached. No limit by default."
>
> "`--dangerously-skip-permissions` — Skip permission prompts. Equivalent to `--permission-mode bypassPermissions`."

Multi-agent support (Claude Code overview, `https://code.claude.com/docs/en/overview`):

> "Spawn multiple Claude Code agents that work on different parts of a task simultaneously. A lead agent coordinates the work, assigns subtasks, and merges results."

> "For fully custom workflows, the Agent SDK lets you build your own agents powered by Claude Code's tools and capabilities, with full control over orchestration, tool access, and permissions."

---

## 3. Where the plan is clearly in bounds

### 3.1 Multi-agent dispatch is officially supported

The Claude Code overview explicitly sells "Run agent teams and build custom agents" and points to the Agent SDK for "fully custom workflows … with full control over orchestration, tool access, and permissions." Tillsyn's cascade is within the intended envelope. No ToS or policy clause restricts the number of parallel agents a dev may run.

### 3.2 Headless dispatch pattern is documented

`--bare -p ... --mcp-config ... --strict-mcp-config` is the documented headless invocation pattern. `PLAN.md` §4.1 and §19.4 use it verbatim.

### 3.3 `acceptEdits` is not a bypass

`acceptEdits` is materially weaker than `default` but materially stronger than `bypassPermissions`. It still enforces protected paths (`.git`, `.claude`, `.mcp.json`, shell rc files, etc.) and still prompts on any Bash command outside the small filesystem whitelist (`mkdir`, `touch`, `rm`, `rmdir`, `mv`, `cp`, `sed`). The plan never uses `--dangerously-skip-permissions` / `bypassPermissions`. This is the correct mode choice for the posture.

### 3.4 Git stays in the dispatcher, not the agent

`PLAN.md` §9.3 holds `git add`, `git commit`, and `git push` in deterministic dispatcher code. The commit agent (haiku) only produces a message string. This keeps agents out of the protected-path prompt path for `.git/` writes and avoids the class of bug where an LLM agent guesses a git command.

### 3.5 Commercial-Terms §D.4 ("competing product") is not triggered

Tillsyn orchestrates Claude Code. It does not train a competing AI model and does not resell inference. The plan's positioning does not implicate the "build a competing product" clause.

### 3.6 Usage-Policy guardrail-bypass clause is not implicated

The "do not intentionally bypass capabilities, restrictions, or guardrails" clause applies to circumventing Claude Code's built-in safety controls (prompt filters, protected paths, auto-mode classifier). The plan operates inside those controls — it configures the controls via documented flags, it does not bypass them.

---

## 4. Gray zones and open risks

### 4.1 Subscription tier is a load-bearing, unresolved decision

The plan does not say whether the dispatcher authenticates as:

- a Claude subscription OAuth login (Pro / Max) — same credentials as the dev's interactive session;
- a `claude setup-token` long-lived OAuth token for CI and scripts;
- an Anthropic API key (Console billing) under the Commercial Terms;
- a third-party provider (Bedrock, Vertex, Foundry) or non-Anthropic model through the Agent SDK.

Each has a different ToS posture:

- **Pro / Max OAuth.** The Consumer-Terms automation clause is satisfied by the "explicitly permitted" carve-out — headless `claude -p` is documented and thus permitted. But a long-running dispatcher spawning dozens of parallel processes, autonomously re-planning on failure, and doing so without human-in-the-loop approval per dispatch, is closer to the spirit of the automation clause than one-shot `claude -p` usage. Max 20x fair-use / weekly Opus quotas will throttle this profile aggressively.
- **`claude setup-token` OAuth.** Documented for CI and scripts. Clearest signal to Anthropic that the automation is intentional and within the permitted carve-out. Still subject to subscription-plan rate limits.
- **Anthropic API key (Commercial Terms).** Removes the Consumer-Terms automation ambiguity entirely. Subjects the deployment to Commercial Terms including "Anthropic may not train models on Customer Content," which is favorable for a system generating large volumes of proprietary code. Cost is per-token rather than flat.
- **Third-party provider.** Bedrock / Vertex / Foundry have their own ToS stacks on top of Anthropic's, but the Consumer-Terms automation clause is not applicable. Auto mode is not available on these providers per Anthropic's docs.

The plan should state which tier(s) the cascade targets, per project, in a new section of `PLAN.md` (proposed §22, see §5.1 below).

### 4.2 Auto-mode classifier is locked out by the tier + permission-mode combination

Auto mode is the only Anthropic-provided autonomous-safety layer. It runs a Sonnet 4.6 classifier on every subagent action, drops arbitrary-code-execution allow rules on entry, and inspects subagent return paths for escalation. The current plan cannot use it because:

- The auto-mode tier requirement (Team / Enterprise / API) is never committed to.
- Even on an eligible tier, the dispatcher starts agents with `--permission-mode acceptEdits`, not `auto`.

This is a defensible choice — the cascade has its own non-classifier safety layers (per-path Edit/Write gating, `--allowedTools` scoping, file + package locks, `max_tries`, `max_budget_usd`, deterministic CI gates, asymmetric QA) — but the plan should state explicitly that the classifier is intentionally replaced by these layers, rather than simply omitted.

Auto-mode-in-headless has a specific failure mode worth noting: "In non-interactive mode with the `-p` flag, repeated blocks abort the session since there is no user to prompt." A cascade run under auto mode that trips the classifier 3 times in a row (or 20 times total) would terminate without a human fallback — the cascade's attention-item flow would need to treat this as a first-class `blocked` outcome.

### 4.3 `Bash(mage *)` is the wide door under `acceptEdits`

`PLAN.md` §5.6 restricts Edit/Write to per-path allowlists but permits `Bash(mage *)`. Claude Code Bash allow rules are not path-gated the way Edit/Write are — any `mage <target>` matches the pattern and runs without prompting.

Prompt-injection vectors:

- Hostile content inside a Tillsyn comment that an agent reads.
- Hostile content inside a Hylla-returned node summary.
- Hostile content in `git diff` output if the agent is reading commits from a malicious upstream.

Each can attempt to coerce a builder into running `mage <any-target-with-side-effects>`. `mage install`, a future `mage ingest`, or any magefile target that talks to the network is reachable through this single Bash pattern.

Auto mode would defend against this class of issue; `acceptEdits` does not. Auto mode also drops `Bash(*)`-style rules on entry specifically because they are the known escape hatch.

Mitigation: replace `Bash(mage *)` with an explicit per-kind allowlist — for example, `Bash(mage test-func *), Bash(mage test-pkg *), Bash(mage ci)` for a build-task kind, and `Bash(mage test-golden *)` for a QA kind.

### 4.4 Host-machine posture vs Anthropic's "isolated environment" recommendation

`PLAN.md` §5.7 is explicit: multiple builders share the single `main/` checkout, gated only by file- and package-level dispatcher locks. No OS-level sandbox, no filesystem isolation, no egress control.

Anthropic's explicit guidance for loose permission modes is "containers, VMs, or devcontainers without internet access, where Claude Code cannot damage your host system." `acceptEdits` is looser than `default` and the plan effectively runs it unattended; the posture gap is real even though the plan never touches `bypassPermissions`.

Blast-radius scenarios:

- A dispatcher bug that releases a file / package lock early lets two builders touch the same file with no OS-level second line of defense.
- An agent coerced via prompt injection into calling a mage target outside its path allowlist (see §4.3) has write access to every file the host user can touch.
- A cascade run producing broken code that gets auto-pushed and auto-reingested could corrupt the very binary running the cascade — which `PLAN.md` §18.5 ("Add `mage install` with Dev-Promoted Commit Pinning") already acknowledges as a real dogfooding risk.

Mitigation options, from cheapest to most thorough:

1. Replace the shared `main/` checkout with a per-run git worktree for cascade agents (keep the current checkout for the dev and orchestrator).
2. Wrap agent subprocesses in `sandbox-exec` on macOS or a Linux namespace sandbox.
3. Run agents inside a devcontainer or Firecracker VM.
4. Require explicit egress allowlisting per agent kind.

### 4.5 Concurrency, rate limits, and cascade-run budget

`PLAN.md` §12.1 — "There is no cap on concurrent agents" — collides with the practical reality of subscription-plan quotas and API-key cost ceilings.

- A typical cascade-run peak: one Opus planner + two Opus plan-QA + N Sonnet builders + 2N Sonnet build-QA + a Haiku commit agent. For N = 3 that is 10 concurrent agents at peak; N = 5 reaches 14.
- Pro / Max plans have 5-hour session windows and weekly Opus quotas. Multiple peak-concurrency cascade runs inside a single session will trip the quota.
- API-key usage has no per-session quota but real per-token cost. Opus-high planner and Opus-high plan-QA at 30-turn budgets can each cost on the order of a dollar per agent; a cascade-run with a failed drop that escalates and re-plans can spiral.

The plan has `max_budget_usd` per invocation, `max_turns` per invocation, `max_tries` per failure, and `blocked_retries` for external failures. It does not have:

- A cascade-run-level budget ceiling (sum of all child invocations).
- A plan-level escalation-depth cap (planner re-plans → builder retries → planner re-plans again is theoretically unbounded under some escalation configurations).
- A per-drop budget ceiling.

### 4.6 Training-data posture is never chosen

Consumer-plan Materials are used for training "unless you opt out of training through your account settings." Commercial Terms (API key) state "Anthropic may not train models on Customer Content from Services." Under a multi-agent cascade emitting large code diffs and QA prose, this is a meaningful data-exposure axis. It belongs in the plan, not implicit.

---

## 5. Recommended plan adjustments

### 5.1 Add "Section 22: Account tier, auth, and data posture" to `PLAN.md`

A per-project, explicit statement of:

- Which subscription / API path the dispatcher uses (Pro / Max OAuth, `setup-token`, API key, Bedrock, Vertex, Foundry).
- Which model provider the cascade targets (Anthropic direct, Bedrock, Vertex, Foundry, or a non-Anthropic model via the Agent SDK).
- Training-opt-out status (consumer plan: explicitly opted out; API key: covered by Commercial Terms).
- Long-lived auth path (`claude setup-token` vs ambient interactive auth).
- Whether auto mode is used (tier must support it) or intentionally omitted in favor of the cascade's own safety layers.

### 5.2 Pre-Drop-5 decision on auto-mode eligibility

Before Drop 5 (dogfooding begins), the plan must choose one of:

- Upgrade the cascade's target deployment to Team / Enterprise / API tier AND switch dispatched agents to `--permission-mode auto` (classifier replaces the raw Bash allowlist).
- Stay on `acceptEdits` AND document in §10 (Trust Model) that the classifier is intentionally replaced by per-path `--allowedTools`, file/package locks, `max_tries`, deterministic CI gates, and asymmetric QA.

### 5.3 Tighten Bash allowlists per kind

Replace `Bash(mage *)` with explicit per-kind patterns:

- `build-task`: `Bash(mage test-func *), Bash(mage test-pkg *), Bash(mage ci)`.
- `qa-check`: `Bash(mage test-golden *), Bash(mage ci)` — read-only verification commands.
- `plan-task`: no Bash mage at all, or a minimal read-only subset.
- `commit-agent`: no Bash at all — commit agent only produces a message string.

Document the allowlist per kind in the template definition.

### 5.4 Add a cascade-run-level budget ceiling

Sum of all child invocations' `max_budget_usd` must not exceed a per-cascade-run ceiling. On exceedance, the dispatcher surfaces an attention item rather than spawning the next agent. This closes the runaway-escalation cost hole that per-invocation budgets do not cover.

Complementary: an escalation-depth cap so planner→builder→planner cycles terminate deterministically.

### 5.5 Flag sandboxing as a future drop

Not required day 1, but a concrete drop in the development order that moves cascade agents into a sandbox (per-run git worktree, `sandbox-exec` on macOS, devcontainer, or Firecracker VM) closes the §4.4 gap. Pre-sandbox posture becomes an explicit risk the plan has accepted; post-sandbox posture aligns with Anthropic's published recommendation.

### 5.6 Make training opt-out explicit

Section 22 (from §5.1) should include one-liners: "Subscription with training opt-out Y/N" or "API key under Commercial Terms Y/N." Auditable.

### 5.7 Make the orchestrator-vs-dispatcher distinction explicit

The Claude Code overview frames multi-agent use as "A lead agent coordinates the work, assigns subtasks, and merges results." The cascade as currently designed is mostly headless dispatch with no lead-agent context; a hybrid model that lets the orchestrator session spawn in-session subagents for interactive work (and reserves headless dispatch for scheduled / unattended runs) fits both the Anthropic framing and the developer's experience better. See §6.3 for the two-model comparison.

---

## 6. Open design questions (for Tillsyn discussion)

### 6.1 API-key path

- Does the dispatcher support per-project API-key credentials?
- How are API keys stored (macOS keychain, env-vars passed to the subprocess, per-project config file under `~/.claude/tillsyn-auth/`, or an explicit `till secret` subcommand)?
- Cost observability: does the dispatcher surface per-cascade-run cost back to the dev via a Tillsyn comment on the cascade drop?
- Does the project's `kind` template override the default (OAuth) credential path, or is the credential path a project-level setting?

### 6.2 Non-Anthropic models

- The Agent SDK supports Bedrock, Vertex, Foundry, and provider bridges. Should the template kind bind a provider (Anthropic / Bedrock / Vertex / Foundry / custom) in addition to a model name?
- Auto mode is Anthropic-API-only. A non-Anthropic path is non-classifier by construction — the plan's own safety layers become the only defense.
- What does a `go-builder-agent on Bedrock Sonnet` spawn line look like (env vars, endpoint configuration, region, IAM role)?
- Do non-Anthropic providers enforce Anthropic's Usage Policy or their own — and how does the template surface the difference?

### 6.3 Orchestrator-called subagents vs cascade headless dispatch

Two fundamentally different execution models:

- **Cascade headless dispatch (current plan).** State change triggers a fresh `claude --bare -p ...` subprocess via the dispatcher. No parent conversation context. The agent reads its task via MCP and terminates when it moves the task to `complete` / `failed`.
- **Orchestrator-called subagents.** The orchestrator session invokes the built-in `Agent` tool with `subagent_type: go-builder-agent` (or equivalent Agent-SDK call). The subagent runs inside the orchestrator's conversation tree (fresh context but same process tree) and reports back via tool result.

Tradeoffs:

| Axis | Cascade headless | Orchestrator-called |
|---|---|---|
| Parallelism | High — no shared conversation state | Moderate — bounded by the parent session |
| Auth | Per-agent Tillsyn auth, pre-issued | Inherits parent session auth scope |
| Safety posture | Fully configurable `--permission-mode` per spawn | Parent session's permission mode applies; auto-mode classifier handles subagents natively |
| Cost accounting | Per-subprocess `max_budget_usd` | Parent-session budget |
| ToS posture | Closer to Consumer-Terms automation clause; benefits from explicit API-key path | Inside the parent session; fits "lead agent coordinates" overview language |
| Observability | Requires streaming + log capture | Native via tool result |
| Compaction | Subprocess is ephemeral — no context to preserve | Parent-session compaction applies |

A hybrid is plausible and probably desirable: orchestrator-called subagents for interactive dev-driven sessions, headless dispatch for unattended / scheduled runs. The template could bind the dispatch mode per kind.

### 6.4 Tier-dependent feature gating

- If a dev is on Pro / Max, auto mode is unavailable — the cascade's own safety layers are mandatory, and the template should refuse to bind `permission_mode = auto`.
- If a dev is on API key, auto mode is available — the template may optionally enable it for cascade runs.
- The dispatcher should read the tier from project configuration and refuse to start if the configured permission mode is incompatible with the tier.

### 6.5 User-facing compliance story

Tillsyn is headed for external users. Each user will bring their own tier, auth posture, training-opt-out decision, and model preference. The plan should define:

- A clear per-project configuration surface for all of the above.
- A pre-flight check that the configured tier / mode / provider combination is compatible before the first dispatch.
- Documentation the user can hand to their own security / legal review.

---

## 7. Decision log

Decisions accrue here as they converge. Source of truth is the Tillsyn drop (comments capture the audit trail; this section mirrors the converged shape).

- **2026-04-14** — ToS analysis complete; six gray zones identified (§4). Seven recommended adjustments proposed (§5). Five open design questions surfaced (§6). Pending dev discussion before any `PLAN.md` edits.
- **2026-04-15** — Tillsyn tracking task created: `3b4052ef-300d-42de-8901-e22cecc9bea0` (kind `task` at project root, project `a5e87c34-3456-4663-9f32-df1b46929e30`). Will be relabelled `drop` once that kind is added to the project's allowed_kinds (Drop 2). Discussion flow: chat-primary; converged points mirrored back into this MD's §7; audit trail on the plan item via `till.comment`; full discussion writeup in `TOS_DISCUSSIONS.md`.
- **2026-04-15** — **Cross-cutting A converged**: pure-headless Claude Code on Max $200 subscription using documented flags carries no meaningful ToS-violation risk. Automation-clause carve-out ("or where we otherwise explicitly permit it") is explicitly populated by `claude setup-token`, `--bare`, `-p`, `--max-budget-usd`, `--max-turns`, and the marketed multi-agent feature. Account-action triggers (AUP: guardrail bypass, scaled abuse, multi-account evasion, competing-product §D.4) do not fire for single-account dev use. Real risks are operational (Max weekly Opus quota / 5-hour session windows), not legal. Training opt-out verified ON on dev's account. The earlier "gray zone" framing in §4 of this doc overstated the legal axis — retracted; the gray zone is operational only.
- **2026-04-15** — **Q3 converged**: day-one Drop 4 dispatch is **pure-headless** via `claude --bare -p ...` under dev's Max subscription. **Concurrency soft-cap N = 4 hard-coded** during dogfood period. Pre-Drop-4 orchestrator-called flow (per `CLAUDE.md`) unchanged through Drop 3. Refinement drops will layer on, in rough order: API-key path (Q1), CLI-app dispatch-approve flow, notification system, non-Anthropic providers (Q2), tier-dependent feature gating (Q4), user-facing compliance story (Q5). Configurability of the concurrency cap deferred to refinement. Claude's earlier "hybrid-day-one" recommendation withdrawn — it was a usability preference, not a compliance requirement.
- **Pending** — Q1 (API-key path) discussion opens next.
- **Pending** — Q2, Q4, Q5 convergence.
- **Pending** — `PLAN.md` §22 draft (records chosen posture + refinement-drop order; will land as a builder-gated edit after all five questions converge, per `TOS_COMPLIANCE.md` §5 recommendations).
