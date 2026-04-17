# HEADLESS DISCUSSIONS

Orchestrator-scope discussion doc. Source of the "Tillsyn-defined agents dispatched via `claude -p`" architecture thread plus any follow-on open questions that arise from it. **Not a Tillsyn plan item** — this is chat-primary discussion substrate until it's ready to be folded into `PLAN.md` under an explicit refinement drop. Keep description-final-shape / comments-audit-trail discipline here too: the sections below are the converged shape; quote-blocks are direct dev or orchestrator lines.

Related: `PLAN.md` §1.4 (Cascade Addressing Vocabulary — already converged), §2 (Hierarchy Refactor), §19.4 (drop 4 Dispatcher Core — currently specifies `claude --agent <type> --bare -p "..."` for spawn).

---

## 1. Research Chain — Why Headless Dispatch Is The Right Path

### 1.1 Problem

- 1.1.1 The orchestrator's system prompt loads every `.md` frontmatter from `~/.claude/agents/` and `./claude/agents/` at startup. With Go + FE variants (builder, QA proof, QA falsification, planner × 2 languages + closeout + inline research) that's ~8+ agent descriptions in the orchestrator context permanently.
- 1.1.2 Motivation to scope: (a) context bloat for agents the orchestrator never dispatches in this project, (b) desire to define cascade-specialist agents that only fire headlessly in dispatcher-driven runs, (c) ability to make specialists more narrowly typed without polluting the orchestrator's picker.

### 1.2 What Does NOT Work

- 1.2.1 **`permissions.deny: ["Agent(name)"]`** — runtime invocation gate only. Agent frontmatter descriptions still load into the system prompt before the permission check runs. Zero context savings, only blocks the call. Docs: https://code.claude.com/docs/en/permissions.md (deny→ask→allow order is a runtime ordering, not a schema filter).
- 1.2.2 **Subfolders inside `.claude/agents/`** — discovery is flat. `~/.claude/agents/cascade/*.md` is not scanned. No `**/*.md` recursion documented.
- 1.2.3 **Custom agent-path setting** — no `additionalAgentDirs`, no path-override option in `settings.json`. Hardcoded scope list: managed → `--agents` CLI flag → project `.claude/agents/` → user `~/.claude/agents/` → plugins.
- 1.2.4 **Plugins as a filter** — plugins CAN carry agents and can be scoped per-project via `enabledPlugins` in `.claude/settings.json`. But all agents inside an enabled plugin load together — no per-agent gating within a plugin. Helps if cascade specialists are their own plugin; doesn't help if we want fine-grained per-agent visibility.
- 1.2.5 Sources: https://code.claude.com/docs/en/sub-agents.md, https://code.claude.com/docs/en/settings.md, https://code.claude.com/docs/en/plugins.md.

### 1.3 What DOES Work — `claude -p` Headless CLI Flags

The orchestrator's in-session `Agent` tool does not take per-call tool restrictions (frontmatter only). But shelling out via Bash to a fresh `claude -p` process is a completely different story — every flag is available at launch:

- 1.3.1 `--tools "Read,Edit,Bash,mcp__tillsyn__*,mcp__hylla__*"` — **hard availability gate**. Tools not on the list cannot be called, period.
- 1.3.2 `--allowedTools "..."` / `--disallowedTools "..."` — soft permission rules (skip prompts in `default` mode).
- 1.3.3 `--append-system-prompt "$SPEC"` / `--system-prompt "$SPEC"` — inject arbitrary system prompt body. Replaces or appends the agent's behavioral spec at launch. Mutually exclusive.
- 1.3.4 `--agents '{"name":{"description":"...","prompt":"..."}}'` — inject full agent JSON at launch, per-invocation only.
- 1.3.5 `--model sonnet|opus|haiku|...` — model tier.
- 1.3.6 `--settings ./path-or-json` — custom settings file at launch.
- 1.3.7 `--permission-mode default|acceptEdits|plan|auto|dontAsk|bypassPermissions`.
- 1.3.8 MCP tools gate by name or prefix: `mcp__tillsyn__till_plan_item`, `mcp__hylla__*`.
- 1.3.9 Source: https://code.claude.com/docs/en/cli-reference.md.

### 1.4 Conclusion

- 1.4.1 The orchestrator keeps only the agents it actually uses in-session as files. Cascade specialists that the dispatcher fires headlessly via `claude -p` don't need to live in `~/.claude/agents/` at all — they can be Tillsyn-stored specs injected at launch.
- 1.4.2 This aligns with §19.4 drop 4 Dispatcher Core's already-documented spawn command:
  `claude --agent <type> --bare -p "..." --mcp-config <per-run mcp.json> --strict-mcp-config --permission-mode acceptEdits --max-budget-usd <N> --max-turns <N>`.
  The refinement is: replace `--agent <type>` (which implies a file lookup) with `--append-system-prompt "$(tillsyn read agent-spec ...)"` + `--tools "..."` + `--model <tier>` so the spec comes from Tillsyn.

---

## 2. Proposed Architecture — Tillsyn-Defined Agents

### 2.1 Where Specs Live

- 2.1.1 **Dev idea (this turn)**: `~/.tillsyn/agents/*.json` (or `*.md` with YAML frontmatter) managed by Tillsyn. Not inside `.claude/agents/` so nothing loads into the orchestrator's system prompt. Dispatcher reads from here at spawn time.
- 2.1.2 **Alternative (fully DB-backed)**: agent specs as first-class records on the Tillsyn project node or template, queried via an MCP op (`till.agent_spec(operation=get, name=...)`). No filesystem artifact. Cleanest from a "Tillsyn is the system of record" standpoint.
- 2.1.3 **Hybrid (likely pragmatic)**: Tillsyn DB is authoritative; dispatcher materializes the per-run spec into a cache dir (e.g. `~/.tillsyn/agents/runs/<run-uuid>/<agent>.md`) for audit and for passing a stable file path to `--append-system-prompt` via a here-doc or `$(...)`.

### 2.2 Spec Shape (Sketch)

```yaml
name: go-builder-cascade
description: Ephemeral code-edit builder, cascade-dispatched only
model: sonnet
tools: [Read, Edit, Write, Glob, Grep, Bash, LSP, mcp__tillsyn__*, mcp__hylla__*]
permission_mode: acceptEdits
max_turns: 30
system_prompt: |
  <behavioral spec; tool discipline; Tillsyn auth + state management;
   mage-only build rules; Hylla-first code understanding; etc.>
```

### 2.3 Dispatch Path

```bash
claude -p "$task_prompt" \
  --append-system-prompt "$spec_system_prompt" \
  --tools "$spec_tools" \
  --model "$spec_model" \
  --permission-mode "$spec_permission_mode" \
  --mcp-config "$per_run_mcp_json" --strict-mcp-config \
  --output-format stream-json
```

### 2.4 Tradeoffs vs. File-Based Frontmatter

- 2.4.1 **Gains**: zero context bloat in the orchestrator; specs live alongside kind/template bindings in Tillsyn; versioned with the project; hard tool gating via `--tools`; model tiering preserved; cascade agents stay narrow without polluting the orchestrator picker.
- 2.4.2 **Losses**: no per-agent `mcpServers` scoping from frontmatter (must inline in the per-run `mcp-config`); no per-agent `hooks` (lifecycle hooks) — if cascade agents need hooks, we need another path; no per-agent `skills` injection — inline as needed.
- 2.4.3 **Neutral**: `--append-system-prompt` vs. `--system-prompt` — probably prefer append so the Claude Code default sub-agent scaffolding stays in place and the Tillsyn spec layers on top.

### 2.5 Migration / Phasing

- 2.5.1 Pre-drop-4: orchestrator still uses file-based agents via the in-session `Agent` tool (no dispatcher yet).
- 2.5.2 drop 4: dispatcher lands; this is where Tillsyn-defined-agent support gets wired. Refine §19.4 to spec `claude -p --append-system-prompt` over `--agent <type>`.
- 2.5.3 drop 10 (Refinement Cleanup): second-pass review of `~/.claude/agents/*.md` can then prune anything the orchestrator no longer needs in-session, now that cascade specialists are Tillsyn-defined.

### 2.6 Open Sub-Questions

- 2.6.1 Does the orchestrator still need file-based variants for in-session spawns (`Agent` tool)? Yes for pre-cascade planning agents the orchestrator wants to invoke directly. Pragmatic answer: keep generalists as files, move specialists to Tillsyn.
- 2.6.2 How do we audit what spec was used on a given run? Answer: dispatcher writes the resolved spec + hash into the plan item's metadata on spawn, and into the run-cache dir. Matches §19.4 auth issuance / process monitoring pattern.
- 2.6.3 How do cascade agents see the Tillsyn spec at runtime (not just the orchestrator that launched them)? They see it as their own `--append-system-prompt` — they don't need to know it came from Tillsyn. No read-back required.

---

## 3. Additional Questions (Q1–Q3 from this turn)

### 3.1 Q1 — @-Mention System Preservation

**Dev question (quoted):** *"we aren't fully eliminating the @ mentioning system right? we will still need it for communication between orchestrators update and down and dev ↔ orchestrator communication and dev ↔ in the future and all would go through attention_items. we will need to make sure we don't eliminate the at mention system in our predogfood drops."*

- 3.1.1 **Status in `PLAN.md`**: plan covers `attention_item` as the escalation substrate (lines 237, 286, 295, 360, 499, 732, 779, 781, 953, 1013, 1032, 1048, 1075, 1103, 1464 — escalations, auth-related, push-deferred pings, gate failures). **But** there's no explicit `@dev` / `@orchestrator` / `@qa` / `@builder` / `@human` mention-routing design section. The mentions are referenced in global `~/.claude/CLAUDE.md` ("Routed `@`-mentions are `@human`, `@dev`, `@builder`, `@qa`, `@orchestrator`, `@research`") and in the project `main/CLAUDE.md` Coordination Surfaces, but not fleshed out as a first-class design in the plan.
- 3.1.2 **Not being eliminated — confirmed.** Dev-↔-orchestrator, orchestrator-↔-orchestrator, and future inter-project orchestrator communication all require it. Attention items are the inbox substrate; mentions are the routing keys.
- 3.1.3 **Gap to close**: add a `Mention Routing + Inter-Orchestrator Communication` subsection to the plan's refinement section. Needs design for: (a) mention → attention_item fan-out rules; (b) orchestrator-to-orchestrator signaling when multiple orchestrators run across projects; (c) `@dev` vs. `@human` semantics (dev is one role of human; might be the same addressee today but not forever); (d) whether mentions route via `till.handoff` (structured next-action) or `till.attention_item` (inbox) — they're different surfaces today and the semantics need tightening.
- 3.1.4 **Pre-dogfood drops must preserve it.** drop 1 (failed lifecycle) and drop 4 (dispatcher) must both keep attention_item + handoff + mention routing intact; none of the hierarchy refactor (drop 2) or template config (drop 3) removes or replaces them.
- 3.1.5 **Refinement section target**: new entry under §15 / §17 range, or a dedicated section "Mention Routing + Inter-Orchestrator Communication (design pending)". Target drop: drop 6 (Escalation) or drop 7 (Error Handling + Observability) — both already touch attention_item routing, natural place to design this cleanly.

### 3.2 Q2 — Git Diff View Per Plan Item

**Dev question (quoted):** *"do we have the git diff view for each plan_item in the refinement drop?"*

- 3.2.1 **Current state in plan**: `start_commit` / `end_commit` are drop 1 first-class fields (lines 1338, 1371 project-field context). Having the two commits means a diff is always derivable — but there is no dedicated "show me the diff for this plan item" surface today.
- 3.2.2 **What's missing**: a user-facing view that computes `git diff <start_commit>..<end_commit> -- <paths>` for a plan item and renders it. Could be: CLI (`till task diff <id>`), MCP op (`till.plan_item(operation=diff)`), TUI pane, all three.
- 3.2.3 **Why it matters for refinement drop**: QA falsification + proof agents, dev review, and per-drop ledger entries all benefit from a canonical diff view tied to plan-item identity. Right now QA agents reconstruct this manually from metadata.
- 3.2.4 **Recommend**: add as a refinement drop entry targeted at drop 6–7 range (alongside the mention-routing design). Needs: MCP op spec, CLI surface, TUI view, tests. Scope'd small because the data (`start_commit`, `end_commit`, `paths`) is already there post-drop-1.
- 3.2.5 **Bonus**: include a `till task diff <id> --vs-live` that shows `<end_commit> .. HEAD` for plan items that shipped earlier in the drop — useful when reviewing drift during an in-flight drop.

### 3.3 Q3 — "drops All The Way Down" Coverage

**Dev question (quoted):** *"look again at the main/PLAN.md full, does it say to change to only one plan_item type, task turns into drop, so it is drops all the way down, but projects are different? like how a branch can have branches, but no branch is a tree, and a tree isnt a branch. does it talk about levels, drop_sub_n which is the depth, but we create the all levels as ordered lists (which are mutable so id is the real tracker, but we have convenience calls (cli, tui all based on ordinal 0.2.5 eg.)"*

- 3.3.1 **Yes — fully covered in `PLAN.md` §1.4 (Cascade Addressing Vocabulary, lines 66–103).** Converged during drop 0 closeout. Contains:
  - "drops all the way down. The `project` is NOT a drop — it is the root container." (line 72)
  - `drop_sub_N` zero-indexed depth/position labels (line 73).
  - Dotted addresses `0.1.5.2` described as read-only shorthand, with project-qualified form `<proj_name>-<dotted>` (lines 76–81).
  - Mutations use UUID only; dotted addresses are unstable under re-parenting (line 82).
  - Type-drop kinds: `plan-drop`, `build-drop`, `qa-drop`, `closeout-drop`, `refinement-drop`, `human-verify-drop`, `discussion-drop` (lines 84–98).
- 3.3.2 **`temp.md` coverage** (lines 8–24): same vocabulary converged interactively, later folded into `PLAN.md` §1.4. Not out of sync.
- 3.3.3 **What's NOT yet in the plan**: the `task → drop` structural rename (turning the `task` kind into a leaf-drop of kind `build-drop` so it's drops all the way down at the schema level too) is referenced in §19.2 drop 2 Hierarchy Refactor but not explicitly as "rename `task` to `build-drop` leaf". Current drop 2 bullets only rename `phase → drop` and `done → complete` + remove `branch`. **Gap**: add a drop 2 bullet to rename or recategorize the `task` kind so leaf build-tasks are canonically `build-drop` per §1.4. Or leave `task` as a legacy alias and add `build-drop` as the canonical name.
- 3.3.4 **Ordinal-based convenience lookup (CLI/TUI/MCP)**: not yet in plan. Proposal — `till view <proj>-<dotted>` and `till comment <proj>-<dotted> --body "..."` read operations use the dotted address; mutations require UUID (matches §1.4 line 82). MCP tools: `till.plan_item(operation=resolve, address="0.1.5.2")` returns the UUID; no mutation ops accept dotted addresses. Worth adding to the refinement section as a quality-of-life item.

---

## 4. Refinement Expansion (No Particular Order)

### 4.1 Web Frontend → TUI Overhaul

**Dev quote:** *"last we will add a webfront end. refine the design and take our lessons from that to overhaul the tui design so it is more user friendly."*

- 4.1.1 **Ordering**: web frontend lands LAST (post-drop 10? likely drop 12+), after the cascade is self-hosting and stable.
- 4.1.2 **Method**: design the web frontend from scratch with current-state usage knowledge — what the dev + orchestrators actually do frequently, what the TUI makes awkward.
- 4.1.3 **Feedback loop**: web frontend design reveals which flows are genuinely user-hostile in the current TUI (not "different", actually *worse*). Use those specific findings to drive a TUI overhaul.
- 4.1.4 **Plan placement**: add as a new §19.12+ drop "Web Frontend + TUI Overhaul". Needs planning-drop before build-drops.

### 4.2 Refactor Flow + Live Tests

**Dev quote (edited):** *"design and setup refactor flow, reqs/ideas: settable stopping points for parity and/or improvement verification through actual interaction in dev version and compare with stable as needed, e2e for all things that could possibly be effected by drop, then when green commit and push and reingest, reinstall stable with new binary. what should we call those live tests and verification stuff?"*

- 4.2.1 **Concept**: refactor drops need programmable stopping points where the dev (or a verification agent) exercises the dev binary against the stable binary to confirm parity or measure improvement, scoped to just the surface area the drop touched.
- 4.2.2 **Naming candidates** (pick one at the refinement drop):
  - **drop Impact Probes** — emphasizes scoped-to-drop + probing behavior.
  - **Parity Checkpoints** — emphasizes dev-vs-stable comparison.
  - **Surface Verification Runs** — emphasizes the scoped e2e surface.
  - **Live Parity Gates** — gate-like, matches the post-build-gate vocabulary already in plan.
  - **drop Acceptance Probes** — ties to `completion_contract`.
- 4.2.3 **Orchestrator recommendation**: "Live Parity Gates" or "drop Impact Probes" — the first emphasizes they're gates (block drop completion), the second emphasizes they're scoped to the drop's affected surface. Probably combine: **"drop Impact Gates"** or **"Impact-Scoped Parity Gates"** (mouthful). Leaning **drop Impact Gates** for brevity.
- 4.2.4 **Mechanics sketch**:
  - Planner agent identifies the blast radius at drop-plan time: paths + packages + downstream callers + TUI views + CLI subcommands affected.
  - Closeout-drop gains a `drop-impact-gate` child task that runs BEFORE commit + reingest.
  - Dev (or a human-verify sub-drop if interactive) walks through the affected surface on both dev and stable binaries. Comparison + findings logged.
  - On pass: proceed to commit → push → CI → reingest → `mage install` promote.
  - On fail: back to builder; drop doesn't close.
- 4.2.5 **Plan placement**: new drop in the refinement region — likely §19.10 drop 10 (post-initial-dogfood) is the right home, since refactor discipline is exactly what drop 10 is for.

---

## 5. Parking Lot (Not Addressed This Turn)

- 5.1 Batch-mutation MCP ops for plan items (dev mentioned in Q3 context — *"a refinement thing we will want is batch operations on plan_item nodes"*). Separate refinement item.
- 5.2 Migrating `TILLSYN-OLD` project (drop 10 cleanup — already in plan).
- 5.3 Cross-project orchestrator session coordination (plan §19.10 subagent-isolation bullet, line 1462). Related to 3.1.3 inter-orchestrator communication.

---

## 6. Status + Next Actions

- 6.1 This doc is the scratch. When ready, the orchestrator folds the converged points into `PLAN.md` under drop 4 (dispatcher refinement for headless agent dispatch), drop 6–7 (mention routing, git-diff view), drop 10 (refactor flow + drop Impact Gates, second-pass `~/.claude/agents/` cleanup), and drop 12+ (web frontend + TUI overhaul).
- 6.2 Pre-dogfood drops (1–3) must not break: attention_item, handoff, @-mention routing, existing file-based agents in `~/.claude/agents/`.
- 6.3 No Tillsyn plan item created from this research session — the orchestrator session will create the `UPDATE PLAN.md — HEADLESS DISPATCH + MENTION ROUTING + REFACTOR FLOW` discussion plan item when it picks this up.
