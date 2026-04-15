# TOS COMPLIANCE — Live Discussion Log

**Tracked in Tillsyn:** plan item `3b4052ef-300d-42de-8901-e22cecc9bea0` (kind `task` at project root of `a5e87c34-3456-4663-9f32-df1b46929e30`; will be relabelled `slice` once that kind lands — Slice 2).
**Companion analysis:** `TOS_COMPLIANCE.md` (frozen analysis snapshot — §1 Summary, §2 verbatim quotes, §3 in-bounds, §4 gray zones, §5 recommended adjustments, §6 open design questions, §7 decision log).
**Companion plan:** `CLAUDE_MINIONS_PLAN.md` (cascade design source of truth; §22 will land once Q1–Q5 converge).

## Purpose Of This File

Three writedowns exist for this work, each at a different granularity:

1. **`TOS_COMPLIANCE.md`** — frozen initial analysis. Only edited to record converged decisions in §7.
2. **`TOS_DISCUSSIONS.md`** (this file) — running back-and-forth on each of the five open design questions from `TOS_COMPLIANCE.md` §6, plus any cross-cutting questions that surface during discussion. Updated per round of convergence. Captures Claude's opening position, dev's response, counterpoints, tradeoffs considered, and the rationale behind whatever gets converged.
3. **Tillsyn plan item `3b4052ef-...` description** — terse mirror of converged shape only. Short on purpose — it's the third writedown, not the first.

Per the "Discuss-in-Comments, Edit-MD-After" memory rule: cross-cutting doc discussions happen chat-primary. Description = converged shape. Comments = audit trail of direct quotes. This MD is the second-tier writedown — longer than the plan item description, narrower than the full analysis, richer than chat.

## Table Of Contents

1. [Q1 — API-key path](#q1--apikey-path)
2. [Q2 — Non-Anthropic models](#q2--non-anthropic-models)
3. [Q3 — Orchestrator-called subagents vs cascade headless dispatch](#q3--orchestrator-called-subagents-vs-cascade-headless-dispatch)
4. [Q4 — Tier-dependent feature gating](#q4--tier-dependent-feature-gating)
5. [Q5 — User-facing compliance story](#q5--user-facing-compliance-story)
6. [Cross-cutting A — Pure-headless on Max $200: is there ANY ToS risk?](#cross-cutting-a--pure-headless-on-max-200-is-there-any-tos-risk)
7. [Decision log (running mirror of `TOS_COMPLIANCE.md` §7)](#decision-log-running-mirror-of-tos_compliancemd-7)

---

## Q1 — API-key path

**Status:** not yet discussed. Opens after Q3 + Cross-cutting A converge.

### Questions on the table (from `TOS_COMPLIANCE.md` §6.1)

- Does the dispatcher support per-project API-key credentials?
- How are API keys stored — macOS keychain, env vars passed to the subprocess, per-project config file under `~/.claude/tillsyn-auth/`, or an explicit `till secret` subcommand?
- Cost observability — does the dispatcher surface per-cascade-run cost back to the dev via a Tillsyn comment on the cascade slice?
- Does the project's `kind` template override the default (OAuth) credential path, or is the credential path a project-level setting?

### Pre-discussion framing

- Pre-Slice-5 dogfooding uses the dev's existing subscription auth (OAuth inherited from the interactive Claude Code session). API-key path lands as a refinement slice when external users show up.
- Per dev's stated plan: refinement slice adds API-key support, CLI-app-driven dispatch-approve flow, and notification system — all layered on top of a dogfooded headless baseline.

### Open sub-decisions

- (waiting on Q3 + Cross-cutting A convergence)

---

## Q2 — Non-Anthropic models

**Status:** not yet discussed. Lower priority — can be deferred to post-Slice-5 refinement.

### Questions on the table (from `TOS_COMPLIANCE.md` §6.2)

- The Agent SDK supports Bedrock, Vertex, Foundry, and provider bridges. Should the template kind bind a provider (Anthropic / Bedrock / Vertex / Foundry / custom) in addition to a model name?
- Auto mode is Anthropic-API-only. A non-Anthropic path is non-classifier by construction — the cascade's own safety layers become the only defense.
- What does a `go-builder-agent on Bedrock Sonnet` spawn line look like — env vars, endpoint configuration, region, IAM role?
- Do non-Anthropic providers enforce Anthropic's Usage Policy or their own — and how does the template surface the difference?

### Pre-discussion framing

- Defer to post-Slice-5 refinement unless a concrete user demand surfaces earlier.
- The template binding should be orthogonal — `kind` → `agent` is one axis, `agent` → `provider+model` is a separate axis. Adding a third `provider` column to the binding table is the likely shape.

---

## Q3 — Orchestrator-called subagents vs cascade headless dispatch

**Status:** Converged 2026-04-15 — pure-headless day-one, refinement slices expand. Dev decision + full convergence below.

### Claude's opening (2026-04-15, captured verbatim from chat)

#### 1. What the current plan actually does

- 1.1 `CLAUDE_MINIONS_PLAN.md` §4.1 and §19.4 describe a **pure headless dispatcher**: state change on a Tillsyn plan item → dispatcher spawns a fresh `claude --bare -p "..." --permission-mode acceptEdits --max-budget-usd N --max-turns N --allowedTools ...` subprocess. The agent reads its task via MCP, executes, mutates the task to a terminal state, and the subprocess exits. No parent conversation context at any point.
- 1.2 The reason given in the plan is parallelism without shared conversation state — N builders + 2N QA + planners can run simultaneously without collision on a single parent session's context window or rate-limit bucket.
- 1.3 The orchestrator-as-hub rule in `CLAUDE.md` says the parent session plans / routes / delegates and never edits Go code. Today that parent session spawns via the `Agent` tool (orchestrator-called). The cascade plan replaces that spawn path with headless dispatch once Slice 4 lands.

#### 2. Case for orchestrator-called subagents

- 2.1 **Fits Anthropic's overview framing.** The Claude Code docs literally say "A lead agent coordinates the work, assigns subtasks, and merges results." Orchestrator-called matches that one-for-one. Headless dispatch is officially supported but framed as a CI / automation use case — not the "lead agent coordinates" pattern.
- 2.2 **Auto-mode classifier works natively.** If the dev is on Team / Enterprise / API tier, auto-mode's classifier runs on every subagent action, inspects return paths for escalation, and is the one Anthropic-provided autonomous-safety layer. Headless dispatch with `acceptEdits` can't use it. Orchestrator-called inherits it for free.
- 2.3 **Auth is simpler.** Parent session already has an orchestrator-role Tillsyn session; subagents inherit the scope they need. No per-spawn auth request dance. Today's pre-cascade flow already works this way — it's proven.
- 2.4 **Observability is native.** Tool results stream back into the parent conversation. The dev sees what the subagent did without needing streamed log capture or per-subprocess structured logging.
- 2.5 **Compaction protection for the parent is real but bounded.** The subagent runs in a fresh context window that doesn't bloat the parent. Only the tool result lands in the parent.

#### 3. Case for headless dispatch (why the plan chose it)

- 3.1 **True parallelism.** `Agent` tool calls in a parent session run in sequence from the parent's perspective — even when the harness executes them concurrently, the parent has to wait for every tool call to resolve. A headless dispatcher can spawn 10 concurrent subprocesses and only intervene when state changes.
- 3.2 **Unattended runs.** A cascade that runs overnight, on schedule, triggered by webhook, or from a cron doesn't have a parent session. Headless is the only model that works there.
- 3.3 **Budget isolation.** Each subprocess has its own `--max-budget-usd` / `--max-turns` enforced by Claude Code itself. In orchestrator-called, the parent session's budget is shared.
- 3.4 **Fresh-context purity.** Headless subprocesses truly have zero inherited context — the `--bare` flag even skips auto-discovery of hooks / skills / MCP / CLAUDE.md. This matters for falsification QA specifically, where parent-hindsight bias would pollute the review.

#### 4. Hybrid model (originally recommended)

- 4.1 **Template binds dispatch mode per kind, not per project.** Planner kinds (`plan-task`, `plan-qa-*`) → orchestrator-called (the dev is usually interacting anyway; auto-mode classifier adds a real safety layer on high-stakes planning). Builder kinds (`task` / `build-task`) → headless when the cascade is running unattended, orchestrator-called when the dev is interactive. QA kinds → headless (falsification-context-purity wins).
- 4.2 **Selector is a project / run-level flag, not a dispatch-time decision.** A `cascade_mode: interactive | unattended` config on the project, or a `--mode=unattended` CLI flag on `till cascade run`. Dispatcher reads it and picks the spawn path per kind accordingly.
- 4.3 **Both paths converge on the same agent files.** `~/.claude/agents/go-builder-agent.md` already describes the spawn-prompt contract. Both orchestrator-called and headless use that contract. Memory rule "spawn prompt + plan-item description" already splits durable vs ephemeral correctly — it doesn't need changes.
- 4.4 **Plan item state machine is unchanged.** Whether spawned in-session or headlessly, the agent's first action is `till_plan_item(operation=move_state, state=in_progress)` and its last is `move_state=done/failed`. Tillsyn doesn't care which spawn path.

#### 5. What changes in the plan if we went hybrid

- 5.1 **`CLAUDE_MINIONS_PLAN.md` §4.1 reframes** — dispatch becomes a strategy interface with two implementations (`in_session_subagent` via `Agent` tool, `headless_subprocess` via `claude --bare -p`). §19.4 gains a flag-check.
- 5.2 **Slice 4 dispatcher gains a mode selector.** Slight scope bump; not a reshape.
- 5.3 **Tier-gating (Q4) inherits from this choice.** Auto-mode only kicks in on orchestrator-called + API-tier. Headless + acceptEdits stays the unattended default regardless of tier.
- 5.4 **User-facing compliance story (Q5) is cleaner.** "You can run Tillsyn interactively with your existing Claude Code session — or configure it for unattended runs with an API key." Two supported modes, each with its own ToS posture.

#### 6. Sub-decisions this forces

- 6.1 Do we commit to supporting both models, or do we pick one?
- 6.2 If hybrid: per-kind binding or per-run mode selector (or both)?
- 6.3 Does Slice 4's dispatcher land with only one implementation, with the other added in a later slice — or both day one?
- 6.4 Does the interactive / orchestrator-called path remain the pre-Slice-5 "current workflow" (as `CLAUDE.md` already documents) with headless layering on top in Slice 4+?

### Dev response (2026-04-15, paraphrased — full source in Tillsyn chat)

- Pushes back on hybrid-day-one. Proposes:
  - Start **pure-headless** in Slice 4+. This is the dogfood target — validate the headless dispatcher pattern end-to-end on the live project.
  - Expand in **refinement slices** (to be added in `CLAUDE_MINIONS_PLAN.md`) to:
    - Add API-key path (Q1).
    - Add "subs from Claude CLI app with dispatch-approve" — interactive approval loop driven by Claude Code's native notification / approval surface rather than an inline orchestrator-called subagent. This sits between pure-headless and the original hybrid — preserves the parent-session / lead-agent framing but decouples approval from the dispatcher's state loop.
    - Add the notification system discussed earlier.
  - Dogfood pure-headless all the way through to get the refinement slices.
- Raises a hard gating concern: "I have a Max $200 plan. Am I at risk of violating ToS if we start pure-headless? Is there literally ANY doubt in your mind? Because I can't risk losing my Claude access!"

### Where the discussion lands before Claude responds

- The dev's plan is architecturally sound if Cross-cutting A resolves favorably (pure-headless on Max $200 is not a ToS risk). Under that outcome, the original "hybrid-day-one" recommendation dissolves — dogfood headless first, then expand.
- If Cross-cutting A resolves unfavorably (any real ToS risk on Max $200 pure-headless), the recommendation shifts back to orchestrator-called-first OR immediate API-key migration. The dev has explicitly said they can't risk account loss.
- **Q3 convergence is downstream of Cross-cutting A.** Resolve A first.

### Dev decision (2026-04-15, direct quotes)

> "7.1 yes pure headless the rest goes in refinement slices"
> "7.2 yes, 4 is good, hard coded, configurability is a refinement slice thing"
> "7.3 I think i opted out, tell me how to confirm"
> "7.4 agreed after I confirm my whole claude account is training opt out"
> "cool privacy is right."

### Converged (2026-04-15)

- **Day-one Slice 4 dispatch: pure headless** via `claude --bare -p ...` under the dev's Max $200 subscription. No orchestrator-called subagents in the dispatcher. Pre-Slice-4 orchestrator-called flow (per `CLAUDE.md`) unchanged.
- **Concurrency soft-cap N = 4 hard-coded** during dogfood period. Configurability deferred to a refinement slice.
- Refinement slices will layer on, in rough order:
  - API-key path (Q1).
  - CLI-app dispatch-approve flow (interactive approval surface from Claude Code's own notification path — sits between pure-headless and in-session subagents; preserves the "lead agent coordinates" framing without reshaping the dispatcher's state loop).
  - Notification system.
  - Non-Anthropic providers (Q2).
  - Tier-dependent feature gating (Q4).
  - User-facing compliance story (Q5).
- **Training opt-out**: confirmed ON on dev's account (claude.ai → Settings → Privacy).

### Sub-decisions resolution

- 6.1 **Pure-headless only day-one.** Not hybrid, not orchestrator-called-only.
- 6.2 **Deferred to refinement.** Per-kind vs per-run selector is a refinement-slice design question, not a Slice 4 decision.
- 6.3 **Slice 4 scope: headless-only.** No hybrid stub. Keeps Slice 4 tight and aligns with dogfood-then-expand discipline.
- 6.4 **Pre-Slice-4 unchanged.** Orchestrator-called (current `CLAUDE.md` flow) continues through Slice 3.

---

## Q4 — Tier-dependent feature gating

**Status:** not yet discussed. Likely resolves naturally from Q1 + Q3.

### Questions on the table (from `TOS_COMPLIANCE.md` §6.4)

- Pro / Max: auto mode unavailable — cascade's own safety layers mandatory, template must refuse `permission_mode = auto`.
- API key: auto mode available — template may opt in.
- Dispatcher reads tier from project configuration and refuses to start if the configured permission mode is incompatible with the tier.

### Pre-discussion framing

- Under the dev's proposed trajectory (pure-headless-first on Max subscription, API-key expansion in refinement), Q4 sits purely in the refinement slices — the Max-subscription dogfood period never uses auto mode.
- Template-level enforcement (refuse to bind incompatible mode + tier combos) is a natural Slice-3-or-later template-refinement.

---

## Q5 — User-facing compliance story

**Status:** not yet discussed. Cross-references Q1, Q2, Q3, Q4.

### Questions on the table (from `TOS_COMPLIANCE.md` §6.5)

- A clear per-project configuration surface for tier / auth / training-opt-out / model.
- A pre-flight check that the configured tier / mode / provider combination is compatible before the first dispatch.
- Documentation the user can hand to their own security / legal review.

### Pre-discussion framing

- Only meaningful after Q1–Q4 settle. The user-facing story is the composition of those answers.
- Shape will likely be a `till doctor compliance` subcommand or similar — pre-flight check that surfaces the current tier + auth + mode + provider + training-opt-out state and flags incompatibilities.

---

## Cross-cutting A — Pure-headless on Max $200: is there ANY ToS risk?

**Status:** Converged 2026-04-15 — no meaningful ToS-violation risk on pure-headless Max $200. Verdict + dev acceptance below.

### Claim under test

> Pure-headless Claude Code dispatch, running on a Max $200 subscription, using documented flags (`-p`, `--bare`, `--permission-mode acceptEdits`, `--allowedTools`, `--max-budget-usd`, `--max-turns`), carries no meaningful ToS-violation risk.

### Evidence inventory (from `TOS_COMPLIANCE.md` §2)

1. **Consumer Terms automation clause** — "Except when you are accessing our Services via an Anthropic API Key **or where we otherwise explicitly permit it**, to access the Services through automated or non-human means, whether through a bot, script, or otherwise." Emphasis added.
2. **Claude Code CLI reference** documents `-p`, `--bare`, `--permission-mode`, `--allowedTools`, `--max-budget-usd`, `--max-turns` as supported flags for scripted use. `--bare` is described as "Minimal mode: skip auto-discovery … so scripted calls start faster." `-p` (print mode) is the headless invocation mode. These are official features.
3. **`claude setup-token`** — "Generate a long-lived OAuth token for CI and scripts. Prints the token to the terminal without saving it. Requires a Claude subscription." Anthropic explicitly ships a token-generation flow designed for subscription-authenticated CI / script use. This is direct evidence that subscription + headless + scripted usage is within the permitted envelope.
4. **Claude Code overview** — "Spawn multiple Claude Code agents that work on different parts of a task simultaneously. A lead agent coordinates the work, assigns subtasks, and merges results." Multi-agent is a marketed feature of Claude Code, not a workaround.
5. **Usage Policy** — "Agentic use cases must still comply with the Usage Policy" and "Intentionally bypass capabilities, restrictions, or guardrails established within our products." Nothing here prohibits the dispatch pattern; it prohibits bypassing guardrails. Configuring guardrails via documented flags is not bypassing them.
6. **Support article on agentic use** — "All uses of agents and agentic features must continue to adhere to Anthropic's Usage Policy." Permissive.

### Analysis to be surfaced when Claude answers

- Map the clause language literally: is subscription-authenticated headless Claude Code "explicitly permitted" by Anthropic? Evidence 2–4 are direct yeses. The automation clause is aimed at bots scraping claude.ai, third-party scripts hitting Anthropic APIs without an API key, or pretending-to-be-human automation — NOT at using Claude Code's own documented flags.
- Separate the legal axis (ToS) from the operational axis (rate limits, weekly Opus quotas on Max plans). A cascade that trips Max-plan quotas is not a ToS violation — it's a throttled / failed run. The worst-case operational outcome on Max is rate-limit errors, not account action.
- Enumerate what actually triggers account action under the AUP:
  - Intentionally bypassing guardrails — N/A (we configure via documented flags).
  - Coordinating across multiple accounts to avoid detection — N/A (single dev account).
  - Running bots or scrapers — N/A (we invoke official Claude Code CLI).
  - Using Services to build a competing AI product — N/A (Commercial Terms §D.4 — Tillsyn orchestrates Claude Code, does not train a competing model).
  - Scaled abuse, surveillance, phishing — N/A.
- Surface the only remaining gray band: "reasonable use" interpretation. A cascade spawning dozens of concurrent subprocesses continuously might look anomalous to Anthropic's internal detection, but anomaly detection is an operational friction (rate limiting, throttling, possible outreach) not a ToS violation. And the published Max plan does not cap concurrency numerically.
- Arrive at Claude's convergence position: **no, there is no meaningful ToS-violation risk in dogfooding pure-headless on Max $200**. The real risks are operational and the dev's "can't risk losing Claude access" concern is addressable by caveats + visibility, not by abandoning the headless trajectory.

### Caveats Claude will want to lock in alongside a "no risk" verdict

- Record the chosen posture explicitly in `CLAUDE_MINIONS_PLAN.md` §22 (new) so future contributors / users inherit the analysis rather than re-deriving it.
- Keep training-opt-out explicit in `CLAUDE_MINIONS_PLAN.md` §22 (Consumer plans: verify opt-out on the dev's account settings).
- During dogfood period, keep concurrency visible and capped — `CLAUDE_MINIONS_PLAN.md` §12.1 ("no cap on concurrent agents") should get a soft cap configurable per-run, even pre-refinement. Not for ToS reasons — for cost + quota hygiene.
- If Anthropic ever narrows the automation clause or publishes new guidance, revisit immediately. The analysis is against the terms as they stood at `TOS_COMPLIANCE.md` date (2026-04-14).

### Verdict (2026-04-15)

- **Pure-headless Claude Code on a Max $200 subscription using documented flags carries no meaningful ToS-violation risk.** Evidence basis:
  - Consumer Terms automation clause has an explicit carve-out ("or where we otherwise explicitly permit it") populated by `claude setup-token`, `--bare`, `-p`, `--max-budget-usd`, `--max-turns` — all documented, marketed, and shipped by Anthropic for subscription-authenticated scripted use.
  - `claude setup-token` is direct evidence: Anthropic explicitly ships a long-lived OAuth token generator "for CI and scripts" that "requires a Claude subscription." If subscription-authenticated scripted access violated the automation clause, this feature could not exist.
  - Multi-agent dispatch is a marketed Claude Code feature ("Spawn multiple Claude Code agents that work on different parts of a task simultaneously. A lead agent coordinates the work, assigns subtasks, and merges results.").
  - Account-action triggers (AUP) — intentional guardrail bypass, scaled abuse, coordinated multi-account evasion, competing-product (Commercial Terms §D.4) — none fire for a dev running Tillsyn on their own single account.
- Real risks are **operational** — Max weekly Opus quotas + 5-hour session windows — not legal. Already modeled as `blocked` outcomes with retries in `CLAUDE_MINIONS_PLAN.md`.
- Residual irreducible uncertainty: ToS could change; Anthropic's dispute-reading could differ from this analysis. Neither rises to "literally any doubt" — they are the irreducible uncertainty of any ToS analysis against published-at-a-point-in-time terms.
- Caveats accepted alongside the verdict:
  - `CLAUDE_MINIONS_PLAN.md` §22 (new, to land) records the chosen posture explicitly so future contributors inherit the analysis.
  - Concurrency soft-cap N = 4 hard-coded during dogfood (Q3 convergence).
  - Training opt-out verified ON on dev's account.
  - Revisit immediately if Anthropic narrows the automation clause or publishes new guidance.
- Withdrawn: Claude's earlier "hybrid-day-one" recommendation was a usability / safety-belt preference, not a compliance requirement. The initial §4 "gray zone" framing in `TOS_COMPLIANCE.md` overstated the legal axis — the gray zone is operational (rate limits / quotas), not legal.

### Dev acceptance (2026-04-15, direct quotes)

> "7.4 agreed after I confirm my whole claude account is training opt out"
> "cool privacy is right."

### Status

- Converged. Cross-cutting A closed. §7 of `TOS_COMPLIANCE.md` updated with the convergence entry.

---

## Decision log (running mirror of `TOS_COMPLIANCE.md` §7)

- **2026-04-14** — ToS analysis complete; six gray zones identified (§4). Seven recommended adjustments proposed (§5). Five open design questions surfaced (§6). Pending dev discussion before any `CLAUDE_MINIONS_PLAN.md` edits.
- **2026-04-15** — Tillsyn tracking task created: `3b4052ef-300d-42de-8901-e22cecc9bea0` (kind `task` at project root, project `a5e87c34-3456-4663-9f32-df1b46929e30`). Will be relabelled `slice` once that kind is added to the project's allowed_kinds (Slice 2). Discussion flow: chat-primary; converged points mirrored back into `TOS_COMPLIANCE.md` §7; audit trail on the plan item via `till.comment`; full discussion writeup in `TOS_DISCUSSIONS.md`.
- **2026-04-15** — Q3 opened. Claude's initial recommendation (hybrid-day-one) met with dev pushback in favor of pure-headless-first + refinement-slice expansion. Convergence gated on Cross-cutting A (pure-headless-on-Max-$200 ToS risk).
- **2026-04-15** — Cross-cutting A converged: no meaningful ToS-violation risk for pure-headless on Max $200 using documented flags. Evidence: automation-clause carve-out is explicitly populated by `claude setup-token`, `--bare`, `-p`, multi-agent marketing. Training opt-out verified ON on dev's account. Real risks are operational (Max weekly Opus quota / session windows), not legal. Claude's earlier "gray zone" framing in `TOS_COMPLIANCE.md` §4 overstated the legal axis — retracted.
- **2026-04-15** — Q3 converged: day-one Slice 4 dispatch is pure-headless via `claude --bare -p ...` under dev's Max subscription. Concurrency soft-cap N = 4 hard-coded during dogfood. Pre-Slice-4 orchestrator-called flow unchanged. Refinement slices will layer on: API-key path (Q1), CLI-app dispatch-approve flow, notification system, non-Anthropic providers (Q2), tier-gating (Q4), user-facing compliance story (Q5).
- **Pending** — Q1, Q2, Q4, Q5 convergence.
- **Pending** — `CLAUDE_MINIONS_PLAN.md` §22 draft (records chosen posture + refinement-slice order).
