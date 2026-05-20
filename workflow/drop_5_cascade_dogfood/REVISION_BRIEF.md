# Drop 5 — Cascade Dogfood (Revision Brief — DRAFT)

**Status:** brief-drafting 2026-05-19, orch-direct from research request. Pre-planning. Open questions in §7 not yet ratified by dev.
**Drop scope (DRAFT — see §5):** transition Tillsyn's own drop coordination from MD-only filesystem drops (per `CLAUDE.md` Coordination Model + `workflow/example/drops/WORKFLOW.md`) to **Tillsyn-runtime-driven cascade dispatch** for at least one real follow-on drop, using the dispatcher + spawn pipeline + gate runner + commit agent that landed in Drops 4a → 4b → 4c.
**Out of scope (DRAFT):** any new dispatcher / gate / spawn pipeline feature work beyond what dogfooding surfaces as a blocker; Drop 4.5 FE/TUI overhaul (parallel track per `PLAN.md` §19.4.5 — DROP_FE_1_BOOTSTRAP is the parallel-track entry, already in flight); Drop 6 retry-tracking and Drop 7 error-observability work (those are the *next* cascade drops the dogfood feeds, not the dogfood itself).

## 1. Hard Prerequisites

**Already shipped (verified via Hylla + git log 2124d2c..HEAD):**

1. **Dispatcher core + auto-promotion** (Drop 4a + 4b). `internal/app/dispatcher/dispatcher.go`, `subscriber.go`, `walker.go`, `monitor.go`, `conflict.go`, `locks_file.go`, `locks_package.go`. `Start` / `Stop` / `RunOnce` / `PreviewSpawn` live. `ErrPromotionBlocked` + sibling-overlap detector + file/package lock managers exist. Continuous-mode subscriber-driven dispatch confirmed by `subscriber.go` doc-comments and 4a.23 falsification fix landed in `dispatcher.go`.
2. **Spawn pipeline** (Drop 4c). `BuildSpawnCommand`, `ResolveBinding`, CLI adapter (`cli_adapter.go`), Claude render package (`cli_claude/render/render.go`), bundle materialization (`bundle.go` + `Bundle.UpdateManifestPID`), orphan scan (`orphan_scan.go`), permission handshake (`handshake.go`), plugin preflight (`plugin_preflight.go`). `CLIKindClaude` default + binding override cascade live per `ResolveBinding` doc-comment.
3. **Gate runner + post-build pipeline** (Drop 4b). `gate_mage_ci.go`, `gate_mage_test_pkg.go`, `gate_commit.go`, `gate_push.go`, `gates.go`, plus `processMonitor.WireGates` injecting the runner into the clean-exit pipeline. `GateStatusFailed` → `transitionToFailed` path exists; D5 e2e tests in `dispatcher_e2e_test.go` (landed `752cb94`) exercise the full chain.
4. **Commit-message agent** (Drop 4c). `commit_agent.go` + `ErrCommitMessageTooLong`. Drop-4b post-build pipeline owns the retry / fail decision.
5. **Templates with agent_bindings + child_rules** (Drop 3, refined in 4a/4c). `internal/templates/` — `load.go` (109 KB), `agent_binding_test.go`, `child_rules.go`, `nesting.go`, `embed.go`. Three builtins: `till-fe`, `till-gen`, `till-go` per `Service.ListBuiltinTemplates`. Per-group agent files under `internal/templates/builtin/agents/{fe,gen,go,till-gdd}/`. Tillsyn project itself is template-free today (`CLAUDE.md` §"Tillsyn Project"): the dogfood subject project either keeps that or binds a template — see §7 Q3.
6. **Orch-self-approval for non-orch subagent auth** (Drop 4a Wave 3). Confirmed via `CLAUDE.md` §"Auth and Leases" + `WIKI.md` §"Auth Approval Cascade." Project-level `OrchSelfApprovalEnabled = *false` backstop is the total disable.
7. **MCP supersede operation** (Drop 4b cleanup R8, `89b4292` + `133d4c0`, currently superseding `failed → complete` for cleanup of auto-template-orphaned twins). Confirmed via `internal/adapters/mcp_rpc/extended_tools.go` + `extended_tools_test.go`.
8. **Hylla operational** (re-ingested 2026-05-18 per memory `feedback_hylla_disabled_for_now`). Mid-reingest at the moment this brief was drafted — last completed ingest pins HEAD `2124d2c`; today's HEAD `752cb94`. Drop 5 cannot start until Hylla is fully back at current HEAD (research-level Hylla queries used here all hit pre-reingest state for changed files — see §7 Q5).

**Open / unverified (must close before planner spawns):**

9. **`till project create` template-binding ergonomics.** The dogfood subject needs a Tillsyn project with the right template attached. Hylla returned no direct CLI-level evidence of `till project create --template <name>` ergonomics being exercised end-to-end against `till-go`. `CreateProject` exists in `internal/app` (referenced by `AppServiceAdapter.UpdateProject`'s sibling code path) but the CLI surface isn't probed in this brief — `cmd/till/main.go` got minor edits in `89b4292`. **Planner verifies.**
10. **`till.action_item` + `metadata.role` post-Drop-2 wiring.** `CLAUDE.md` says role lives in description prose pre-Drop-2 and lands on `metadata.role` post-Drop-2. Drop 2 status not explicitly confirmed in `PLAN.md` reading. **Planner verifies whether Drop 2 has landed, or whether dogfood operates with role-in-prose.**
11. **Dispatcher manual-trigger vs auto-promote-mode for dogfood.** `till dispatcher run --action-item <id>` is the documented Drop 4a manual entry; auto-promotion-on-state-change is documented as landed in Drop 4b. Dogfood scope choice (§5) hinges on which mode we drive.
12. **Closeout / refinement / discussion / human-verify kinds — operator-managed.** Per `CLAUDE.md` "Agent Bindings" table, these four kinds have `agent: orchestrator-managed` — they don't auto-dispatch. The dogfood drop needs explicit human-orchestrator (or STEWARD-equivalent) ownership of those nodes.

## 2. Goal

Validate that the cascade — dispatcher + spawn pipeline + gate runner + commit agent + templates — can drive a real Tillsyn-repo drop end-to-end **with Tillsyn action items as the system of record for work tracking**, not just as auth/credential plumbing alongside `workflow/drop_N/` MDs.

Success looks like: a small, well-scoped follow-on drop runs through `plan → plan-qa-(proof|falsification) → build → build-qa-(proof|falsification) → commit → push → CI green → closeout` with action_items in Tillsyn driving each transition, agents spawned by the dispatcher (not the parent orchestrator session), gates running between build and build-QA, and the commit agent forming the message.

Failure modes Drop 5 catches:
- Spawn pipeline edge cases against real-world prompts / real CLAUDE.md sizes / real auth tokens.
- Gate runner sequencing against actual `mage ci` + `git push` against the live remote.
- Template `agent_bindings` mismatched against the cascade-default agent name resolution.
- Auth flow at non-trivial fan-out (planner + 2 plan-QA + N builders + 2N build-QA + commit + closeout — order-of-magnitude more sessions than orch-direct).
- Orphan-scan + cleanup behavior under real spawn churn.

## 3. Pre-MVP Rules In Force

- Hylla MCP is the primary Go code-understanding source. Record misses in subagent closing comments under `## Hylla Feedback`.
- Filesystem-MD for drop coordination **transitions to Tillsyn-runtime for this drop and onward** if §5 option (a) lands. Until that flip, MD-only.
- Tillsyn-flow output style + Section 0 SEMI-FORMAL REASONING in every subagent response.
- Single-line conventional commits, ≤72 chars; commit agent enforces.
- Never raw `go test` / `go build` / `go vet`; always `mage <target>`.
- No closeout MD rollups pre-dogfood — but Drop 5 IS the first dogfood, so this rule begins to retire here. Planner decides what gets rolled up vs lives in Tillsyn-only.

## 4. Component Breakdowns (Definition of Cascade Dogfood)

This section answers the dev's research Question 1: *what concretely changes when we flip the switch?*

### 4.1 Project Setup in Tillsyn

**Today:** Tillsyn project `a5e87c34-3456-4663-9f32-df1b46929e30` exists with `template: none` (per `CLAUDE.md` §"Tillsyn Project"). Drop coordination lives in `workflow/drop_N/` MDs; Tillsyn action_items hold auth-bootstrap nodes (STEWARD persistent parents `DISCUSSIONS` / `HYLLA_FINDINGS` / `LEDGER` / `WIKI_CHANGELOG` / `REFINEMENTS` / `HYLLA_REFINEMENTS`) but not per-drop work.

**Dogfood change:** bind a template (likely `till-go` per `Service.ListBuiltinTemplates`) to the Tillsyn project, OR create a sub-project / parallel project for the dogfood subject. See §7 Q3 for the trade-off.

### 4.2 Templates + Agent Bindings + child_rules

**Today:** `till-go` template exists in `internal/templates/builtin/` and references agents under `agents/go/`. `agent_binding_test.go` + `child_rules.go` + `nesting.go` ship. The template's `child_rules` should encode "every `plan` auto-creates `plan-qa-proof` + `plan-qa-falsification`" + "every `build` auto-creates `build-qa-proof` + `build-qa-falsification`."

**Dogfood verification:** planner reads `till-go.toml` (25.2 KB per ls) and confirms the child_rules match the `CLAUDE.md` "Required Children (Auto-Create Rules)." Mismatch is a hard planner-stop blocker.

### 4.3 Action-Item Tree Creation Flow

**Today (pre-cascade pattern, per `CLAUDE.md` §"Cascade Tree Structure"):** the orchestrator creates the level_1 drop + planner action_item + the plan-QA twins by hand via MCP `till.action_item operation=create` calls.

**Dogfood change:** the planner agent itself creates child action_items (per `PLAN.md` §19.5 line 1700: *"Planner agent integration: agent creates child action items via MCP"*). This requires (a) planner subagent has MCP write capability to `till.action_item operation=create`, and (b) the planner's spawn prompt includes the action-item-tree-creation directive. Both are templates / agent-binding concerns.

### 4.4 Dispatcher Trigger Mode

**Today:** two paths exist — manual (`till dispatcher run --action-item <id>`) and auto-promote (Drop 4b's subscriber-driven loop, started via `dispatcher.Start`).

**Dogfood choice:** likely auto-promote — that's the cascade's whole point. But manual-trigger is the safer rollout: dev runs `till dispatcher run` per dispatched node, catches surprises one node at a time. Recommended dogfood sequence: first 2-3 nodes manual; once the loop is observed clean, flip to auto-promote for the rest of the drop. **Planner picks per §7 Q1.**

### 4.5 Gate Execution

**Today:** `processMonitor.WireGates` injects the gate runner + template resolver into the monitor's clean-exit pipeline. Gates resolved per template, executed between agent-clean-exit and `transitionToFailed | transitionToComplete`.

**Dogfood verification:** the till-go template specifies which gates fire per kind. Likely: `build` → `mage_test_pkg` (per-package test gate per `gate_mage_test_pkg.go`); drop-end `closeout` → `mage_ci` + `commit` + `push`. Planner verifies template-vs-CLAUDE.md alignment. Missing gates or extra gates are hard blockers.

### 4.6 Commit + Push Gates

**Today:** `gate_commit.go` invokes the commit agent (`commit_agent.go`); `gate_push.go` runs `git push origin <branch>`. Commit message comes from the haiku-bound commit-message-agent.

**Dogfood verification:** confirm the commit-agent spawn pipeline works end-to-end. Per `commit_agent.go` doc, `ErrCommitMessageTooLong` flags >72-char messages — dogfood may hit this if the agent gets verbose on a real diff. Planner schedules a smoke retry plan in case.

### 4.7 Closeout

**Today:** drop-orch owns closeout per `workflow/example/drops/WORKFLOW.md` Phase 7 — aggregate Hylla feedback, refinements, ledger entry, wiki changelog into `CLOSEOUT.md`. Hylla reingest is drop-end-only, from remote, after CI green.

**Dogfood change:** closeout becomes a Tillsyn `closeout` kind action_item under the level_1 drop; it's `orchestrator-managed` (no LLM agent). The orchestrator (parent CC session OR STEWARD-equivalent) closes it after the drop-end ledger/Hylla/refinements rolls land. **MD files in `workflow/drop_N/` still get written** because they're the durable on-disk source — Tillsyn closeout-kind action_item just tracks state, MD files carry content. (Per `WIKI.md` §"Drop Orch Cross-Subtree Exception" — the Tillsyn description holds a short summary + pointer into `workflow/drop_N/` files.)

## 5. Scope Shape (Recommendation: option B + thin slice of option A)

This answers the dev's research Question 5.

### Option A — Full Infrastructure + Run One Dogfood Drop End-to-End

- Audit + close all §1 prereqs.
- Bind template to Tillsyn project (or create dogfood-subject project).
- Wire planner-agent action-item creation if not already.
- Run a small follow-on drop (candidates in §6) through the full cascade.
- Capture findings, refinements, hylla feedback.

**Pros:** ships the dogfood signal the cascade was designed to produce. PLAN.md §19.5 explicitly says *"From here on, the cascade can build itself."*
**Cons:** large + integrative + high uncertainty surface. Each unverified prereq (§1 items 9–12) could derail. Failure-mode-hypothesis (§6) lists 4 concrete ways this goes wrong.

### Option B — Audit Prereqs + Write Dogfood Playbook + One Trivial Smoke Run

- Audit all §1 prereqs systematically (each gets a build-qa-equivalent verification step).
- Write `workflow/drop_5_cascade_dogfood/DOGFOOD_PLAYBOOK.md` — the operator's checklist for running cascade-driven drops.
- Run **one minimal smoke drop** end-to-end (e.g., a 1-droplet `chore(docs)` change to confirm the loop works) before declaring Drop 5 complete.
- Defer the "first real cascade-managed feature drop" to Drop 5.5 / 6.

**Pros:** captures everything Drop 5 was supposed to capture (the prereq-audit + playbook + signal), but bounds the uncertainty. The smoke run de-risks Drop 6/7's first real cascade work. Doesn't pre-commit to a specific dogfood subject.
**Cons:** technically delays the "real" dogfood by one drop. May feel like ceremony.

### Recommendation

**Option B, with a tight smoke-run requirement.** Rationale:
- §1 items 9–12 are unverified — Option A bets the drop on those resolving cleanly.
- The cascade has shipped a lot of code in 4a/4b/4c that has never been exercised at full fan-out under real-world prompt/auth/token volumes. Smoke-run catches the obvious; real drop catches the subtle.
- Dev's documented "decomp into small parallel planners" preference (memory `feedback_decomp_small_parallel_plans`) maps better to Option B: prereq-audit is naturally parallelizable, playbook authoring is single-threaded, smoke-run is sequential. Option A's "run one big dogfood drop" is harder to parallelize and harder to recover from mid-drop.
- Option A is the obvious sequel — Drop 6 / 7 in `PLAN.md` are the natural homes for "first real cascade-managed feature drop" once Drop 5's playbook lands.

**If dev prefers Option A,** the brief expands §6 (failure-mode-hypothesis) into explicit mitigation playbooks, and §7 Q1 becomes load-bearing.

## 6. Failure-Mode Hypothesis (Ranked, Most → Least Likely)

This answers the dev's research Question 4. All four are stated as hypotheses; Drop 5's smoke run / dogfood is what proves or refutes them.

### 6.1 Auth flow at scale (HIGH likelihood)

The cascade spawns far more sessions than orch-direct: one orchestrator + one planner + 2 plan-QA + N builders + 2N build-QA + 1 commit agent + closeout. For a 5-droplet build drop that's ~15 sessions vs the pre-cascade ~5. Orch-self-approval is in place (Drop 4a W3), but it's been exercised only under MD-coordination spawn volumes. Likely failure: token-claim races, lease-issue ordering, orch session expiry mid-fan-out.

**Mitigation surfaced in dogfood:** record per-session timing + auth-claim latency in the smoke run; if the orch-self-approval path is slow, queue or batch.

### 6.2 Templates not matching dispatcher expectations (HIGH likelihood)

`till-go.toml` ships a binding table; the dispatcher's catalog (`internal/templates/catalog.go`) reads it. Drift between the template's agent names and the cascade's bare-name convention (`builder-agent` / `build-qa-proof-agent` / etc.) would fail at agent-resolution time. The 3-tier walk (project → user → embedded) means a stale project-level override could silently win.

**Mitigation:** Drop 5 prereq-audit explicitly cross-checks `till-go.toml` agent_bindings against `CLAUDE.md` §"Agent Bindings" table. Mismatch = hard stop.

### 6.3 Gate runner edge cases (MEDIUM likelihood)

Gates run as black-box exec; failures wrap into sentinel errors. Likely problems: gate timeout on a slow `mage ci`, gate stdout/stderr handling weirdness (per memory `feedback_mage_precommit_ci_parity`), commit-agent length cap hit on a verbose diff.

**Mitigation:** smoke-run inspects gate logs; first dogfood drop is small enough that gates run fast.

### 6.4 Spawn-pipeline subprocess crashes (LOW likelihood)

Drop 4c shipped thoroughly tested; orphan-scan handles dead PIDs; bundle materialization has its own error contract. Less likely to surface as a *new* failure mode.

**Mitigation:** orphan-scan runs at dispatcher start; smoke-run confirms it doesn't false-positive on a clean spawn.

## 7. Open Questions for Planner (DEV RATIFIES BEFORE PLANNING STARTS)

- **Q1 — Scope shape.** Option A (full dogfood) or Option B (audit + playbook + smoke run)? Brief recommends B. *Dev decides.*
- **Q2 — Dogfood subject (if option A or if option B's smoke run needs a non-trivial target).** Candidates:
  - **(a)** Absorb Drop 4b carry-forward refinements R1/R2/R3/R4 (template-validation hardening, deferred from `drop_4b_test_cleanup`'s scope) — small, well-defined, test-heavy. Lowest risk.
  - **(b)** A tiny `ui/` follow-on (parallel-track FE work — but FE is being run via DROP_FE_1_BOOTSTRAP separately, may not be cascade-managed yet).
  - **(c)** A synthetic scoping drop (e.g., add a missing `mage` target) — bounded, no domain risk, but feels artificial.
  - **(d)** A "drop 5.5 — cascade ergonomics first-pass refinements" — capture whatever the dogfood surfaces and absorb it inline.
  - *Brief recommends (a) — has clearest scope + acceptance criteria.*
- **Q3 — Template binding strategy.** Three options:
  - **(i)** Bind `till-go` directly to project `a5e87c34-…` (the live Tillsyn project). Risk: mid-dogfood template glitch contaminates the live project.
  - **(ii)** Create a parallel "tillsyn-dogfood" project, bind `till-go` there, dogfood there, retire when stable. Cost: extra project; need `till project create` ergonomics to work cleanly (§1 item 9 unverified).
  - **(iii)** Build a custom Drop-5-specific template that bakes in the verified-good `agent_bindings` only, no `child_rules` complexity. Cost: extra template authoring, but lowest blast radius.
  - *Brief recommends (ii) — isolation + tests `till project create` as a side effect.*
- **Q4 — Dispatcher trigger mode for the smoke run.** Manual (`till dispatcher run --action-item <id>` per node), auto-promote (set `Start`, walk away), or hybrid (manual for the first ~3 nodes, auto-promote for the rest)? Brief recommends hybrid.
- **Q5 — Hylla reingest gating.** Hylla is mid-reingest at brief-draft time. Drop 5 cannot start until Hylla is back at current HEAD — confirm with dev before planner spawns. If Hylla is still down when planning begins, switch to LSP + `git diff` fallback for prereq-audit (planner work doesn't strictly require Hylla, but the dogfood subagents do).
- **Q6 — `workflow/example/CLAUDE.md` drift remediation.** The example/ CLAUDE.md is explicitly MD-only with a paradigm-override preamble that tells subagents to ignore `till_*` / `auth_request` / etc. This is **correct for external adopters** (per the doc's own framing) but **diverges from Tillsyn-self post-Drop-5**. Options:
  - **(i)** Leave `workflow/example/` as-is (it's an adopter reference, not Tillsyn-self).
  - **(ii)** Add a second example variant `workflow/example_cascade/` (Tillsyn-runtime-driven) alongside the MD-only one. Adopters pick the one matching their setup.
  - **(iii)** Update `workflow/example/` to be cascade-runtime-by-default, and demote MD-only to a fallback note.
  - *Brief recommends (ii) — preserves both audiences. Could be a Drop 5.5 / 6 follow-on, not necessarily Drop 5 scope.*
- **Q7 — `closeout` / `refinement` / `discussion` / `human-verify` kind handling.** These are `orchestrator-managed` (no LLM agent). Who runs them in the cascade-driven world?
  - **(i)** Parent Claude Code session (today's pattern).
  - **(ii)** A persistent "STEWARD-equivalent" orchestrator agent that lives across drops.
  - **(iii)** Dev manually via TUI / CLI.
  - *Brief notes: per `WIKI.md` §"Drop Orch Cross-Subtree Exception" the existing STEWARD pattern points at (ii). Dev confirms.*
- **Q8 — Tillsyn-runtime vs MD-only handoff.** Once Tillsyn-runtime drives a drop, what happens to `workflow/drop_5_cascade_dogfood/PLAN.md` and the other MD artifacts?
  - **(i)** They keep getting written (per WIKI exception rule — MD on disk is source of truth for content, Tillsyn description carries summary + pointer).
  - **(ii)** They're retired in favor of Tillsyn descriptions / completion_notes.
  - *Brief recommends (i) — the audit trail + post-merge STEWARD splice flow already depends on the MDs; retiring them is a Drop 8+ wiki-system concern.*
- **Q9 — Concurrency cap.** `PLAN.md` line 1023 mentions a hard-coded soft cap of N=6 concurrent active agents. Drop 5's smoke run should test this — does a 5-droplet build drop with parallel-eligible droplets actually run at N=6, or does it serialize?
- **Q10 — Failed-state cleanup.** If a dogfood action_item lands in `failed` terminal state, the dev-CLI `till action_item supersede` unsticks it. The MCP `supersede` operation just landed (`89b4292`). Confirm planner has access to that operation OR document the failed-state recovery flow.

## 8. Approximate Size

**Option B (recommended):**
- Prereq audit (parallelizable across §1 items): ~5-8 droplets, each ~30-100 LOC of test or doc.
- DOGFOOD_PLAYBOOK.md authoring: ~1 orch-direct MD droplet, ~200-400 lines.
- Smoke-run setup (project + template + auth): ~2-3 droplets, mostly orchestration.
- Smoke-run execution: 1 dogfood subject drop (Q2 (a) recommended = R1/R2/R3/R4 absorption) — itself ~4-7 droplets via cascade, ~400-800 LOC.
- Drop 5 closeout + ledger entry: ~1 orch-direct droplet.

Total: ~12-20 droplets in Drop 5 if Option B. Approximately 2-3x larger if Option A and the dogfood subject is non-trivial.

## 9. Cross-References

- `PLAN.md` §19.5 (Drop 5 — Cascade Planning, original scope bullets) — source of `[ ] Planner agent integration` / `[ ] Planning QA integration` / etc.
- `PLAN.md` §19.4.5 (Drop 4.5 parallel track) — FE/TUI overhaul concurrent with Drop 5; explicit dev direction "starts early to inform TUI direction."
- `CLAUDE.md` §"Cascade Tree Structure" — agent bindings table + required-children rules.
- `CLAUDE.md` §"Coordination Model" — current MD-only stance + post-Drop-2 Tillsyn-runtime transition pointer.
- `WIKI.md` §"Coordination Model" + §"Drop Orch Cross-Subtree Exception" — drop-orch authoring patterns + STEWARD post-merge collation.
- `workflow/example/CLAUDE.md` — adopter-facing MD-only reference (drift discussion §7 Q6).
- `workflow/example/drops/WORKFLOW.md` §"Agent Spawn Contract" — paradigm-override preamble.
- Memory `project_drop_4b_refinements_raised.md` — Drop 4b refinement carry-forward catalog (R1/R2/R3/R4 candidate dogfood subjects per §7 Q2(a)).
- Memory `feedback_decomp_small_parallel_plans` — orch decomposition default (drives Option B preference).
- Memory `feedback_parallelize_unblocked_default` — concurrency directive.
- `internal/app/dispatcher/` — full dispatcher + spawn pipeline + gate runner surface.
- `internal/templates/builtin/till-go.toml` — likely dogfood-subject template (planner verifies).
- `cmd/till/main.go` — CLI surface entry; `till project create` + `till dispatcher run` paths live here.

## 10. Out of Scope (Hard)

- Drop 4.5 FE/TUI overhaul work (parallel track; not gated on Drop 5).
- Drop 6 retry-tracking + max-tries semantics.
- Drop 7 external-failure detection + attention-item routing for failure types.
- Drop 8 wiki/ledger system (the Tillsyn-internal one, distinct from `LEDGER.md` aggregation).
- Drop 10 refinement cleanup (post-dogfood by definition; this drop produces the refinement list, doesn't absorb it).
- Any new dispatcher feature work (Drop 5 exercises what 4a/4b/4c shipped; bugs surfaced get filed as Drop 5.5 / 6 refinements, not absorbed inline unless trivial).
- Migration logic — pre-MVP no-migration rule applies (memory `feedback_no_migration_logic_pre_mvp`). If dogfood reveals schema drift, dev deletes `~/.tillsyn/tillsyn.db` and re-runs.
