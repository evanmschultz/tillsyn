# Drop 4c.6 / 4c.7 / 4c.8 — Per-Project Agents + Runtime Config + Cascade Wiring + Default Prompts (v2.8.4 — POST-QA FINAL)

**Status**: pre-planning sketch — **v2.8.4 POST-QA FINAL**. Both plan-QA-proof and plan-QA-falsification PASS verdicts achieved on v2.8.3; 5 minor findings applied → v2.8.4. Decision LOCKED: Option A-split (3 drops, close all gaps). All three must ship before dogfood. Wave breakdown in §25; per-wave Specify blocks in §26.

**v2.8.4 changelog from v2.8.3 (post-QA fixes)**:

- **F1 (proof)**: §26.W7 + §26.W8 AcceptanceCriteria add explicit "W0.5 claim-vs-impl validator's known-wired set updated to include this wave's newly-wired consumer" bullet — closes silent-staleness risk where a wave wires a consumer but doesn't update the validator's coverage map.
- **F2 (proof)**: §25.5 rewritten — was listing "two minor updates needed" to planner draft that were already applied; now correctly reads "Planner draft updates — APPLIED" with cross-reference to disk state.
- **C1 (falsification)**: §26.W7 AcceptanceCriteria add `metadata.auto_created_via_child_rule: true` marker requirement on every auto-created child — required by §26.W11's "non-system actor updating system-auto-created QA → reject" runtime check; without this marker W11 cannot discriminate auto-created from agent-created QA items.
- **C2 (falsification)**: planner-draft (both `~/.claude/agents/go-planning-agent.md` and `workflow/drop_4c_6/PROPOSED_AGENT_DRAFTS/till-go/planning-agent.md`) "Cascade Design — Atomic Droplets + Parallelization (HARD RULES)" section reframed: atomic-droplet EXISTENCE is structural invariant (hardcoded methodology); SPECIFIC SIZING NUMBERS (1-4 code blocks, 80-120 LOC) are till-go template values that adopters running other templates may differ on. Per `feedback_tillsyn_enforces_templates.md` structural-vs-semantic split.
- **C3 (falsification)**: removed the "OLD §25 Decision asked" historical block; renumbered trailing sections — old §23 "Out of scope" → §27; old §24 "Pending updates" → §28. Sketch numbering now matches file order.
- **3 Unknowns routed to dogfood measurement** (per falsification §3): token-cost ratios for failure synthesis vs archived-children dump; N=3 escalation right-sizing; compaction-resilience under wipe-and-replan + multi-round failure synthesis. Not blocking — measured post-dogfood.

**v2.8.3 changelog from v2.8.2**:

- §26 W10 N-failure escalation default = **N=3** (1st attempt + 2 retries → escalate). Configurable per template post-MVP.
- §11.1 clarified: `kind=research` does NOT auto-create QA twins; orchestrator reviews findings at comment level. `research-agent` IS in W4 prompt-authoring scope (one of the 7 standard agent names).
- Memory `project_team_aware_architecture.md` updated: fork-and-PR model (per dev §6) — anyone with maintainer-granted access to a section forks that section automatically; works on fork; submits PR-style merge request; maintainer reviews + accepts/rejects/requests-changes. NOT a "publish-as-permission" model. Mental model is `git fork → branch → commit → push → PR → review → merge`.
- Decision LOCKED: Option A (remove `till.action_item.delete` MCP tool entirely) per dev §6 of last turn ("we just need to wipe and redo"). Sketch §26 W11 already reflects.

**v2.8.2 changelog from v2.8**:

- §26 W11 reworked per `RESEARCH/AUTENT_ARCHIVE_QA_PRIOR_PLANNING.md`:
  - REMOVE `till.action_item.delete` MCP tool entirely (today exists with `mode=archive|hard`; W11 strips MCP exposure; Go method stays for internal callers).
  - REMOVE archive from MCP agent surface entirely (today archive is `delete mode=archive`); becomes human-only via UI/CLI.
  - ADD comment role-gating (today no role-gating; QA-only-on-own-action-item is fresh structural rule).
  - ADD `Service.RestoreActionItem` role-gate verification + fix if gap exists (research flagged).
  - DROP `allow` violation policy (per dev §3.1 prior turn); keep `reject` (hardcoded structural) + `warn` (template advisory).
  - **Separation-of-concerns moved from template-defined to HARDCODED structural** (per dev §3.2 prior turn). Planners NEVER touch QA. Builders NEVER decompose. These are cascade-architecture invariants.
- §26 W10 expanded with N-failure escalation (NEW per research §3.3 — unspecced before).
- §22 research-deliverable summary updated with autent + capability-lease split + delete/archive truth + comment-gating gap.
- Memories written: `feedback_prompt_injection_team.md` + `project_team_aware_architecture.md` (latter updated to clarify autent vs capability-lease split per research).
- W11 LOC estimate revised UP from ~80-150 to ~250-360 LOC due to delete/archive removal + comment gating + restore gap fix.

**v2.8 changelog from v2.7**:

- §25.1 Drop 4c.6: NEW wave **W0.5 (TEMPLATE VALIDATION + LOAD-TIME FAIL-LOUD)** — closes the gap dev called out: schema-shipped-but-not-validated. Validators for `[[child_rules]]` cycles, `blocked_by` acyclicity, `agent_name` existence, kind closed-enum membership, recursion-depth bounds, claim-vs-implementation coherence. Each emits structured error with TOML-line pointer + "this template is broken because X; cannot ingest" message. Per `feedback_tillsyn_enforces_templates.md`: Tillsyn enforces; fails LOUD on bad templates.
- §25.2 Drop 4c.7: NEW wave **W11 (RUNTIME FAIL-LOUD ON TEMPLATE-ENFORCEMENT VIOLATIONS)** — enforce template rules at the MCP boundary. Planner creating QA action item → reject. Builder editing outside `paths` → reject. Builder running `mage test-pkg` instead of `mage test-func` → soft-fail (template-configurable). Schema-shipped feature with no consumer wired (load-time check) → reject template. Each rejection emits structured error citing the template rule violated.
- §26 NEW: **Per-Wave Specify Blocks (SDD-Inspired Demonstration)** — each wave (W0 → W11 across 3 drops) gets a Specify block (Objective + AcceptanceCriteria + ValidationPlan + RiskNotes + ContextBlocks) authored in the same shape the planner-agent will use for action-item metadata. The sketch eats its own dogfood: methodology document authored using the methodology it documents.
- This addresses the dev's sharp question: "does the plan state, demonstrate, AND enact (via SDD) the ethos?" Prior versions stated; v2.8 demonstrates + enacts.

**v2.7 changelog from v2.6**:

- §11.2 reworked again: planner is now **BLIND to archived children** on plan-failed. System synthesizes a `failure_context` summary from QA findings BEFORE archiving; injects into the fresh planner's system-prompt.md. Planner authors fresh decomposition from synthesized failure context only — no MCP-fetching of archived children, no partial revival, simplest-possible reset. New `metadata.failure_history` typed field on `ActionItemMetadata` (list — supports multiple cycles). Net tokens DOWN (synthesis ~200-500 tokens vs ~2000-5000 archived-child dump).
- §25 Drop 4c.7 W10 scope EXPANDED: `Service.WipeChildrenAndRePlan` now (a) collects QA findings BEFORE archive, (b) synthesizes failure_context summary, (c) appends to `parent.metadata.failure_history`, (d) archives + transitions. Plus new `metadata.failure_history` field on domain layer (~30 LOC). Plus render-layer hook (W3 + W8 already in scope) to include "Prior Attempt Failed" section in system-prompt.md when `failure_history` non-empty.
- Memory: `feedback_opus_builders_pre_mvp.md` retired; `feedback_cascade_model_policy.md` written. Cascade-dogfooding model policy (sonnet planners/builders, opus QA, haiku commit) now lives in current memory.
- Planner draft v4 on disk: "Wipe-and-restart" section reworked — planner is BLIND to archived children; consumes failure context only via system-prompt's synthesized section. "You NEVER read archived children" added to forbidden list.

**v2.6 changelog from v2.5**:

- §25 LOCKED: Option A-split chosen. Three drops — 4c.6 (foundation: config + isolation), 4c.7 (cascade wiring: auto-create + context-preload + wipe-and-replan), 4c.8 (default prompts + dogfood overrides). Sequential dependencies. Droplet counts NOT specified at sketch level per `feedback_plan_down_build_up.md` — planner determines per-level decomposition; plan-QA verifies. Bias toward smaller per-droplet sizes per dev's "don't let stuff fall through cracks" is a quality guideline, not a count cap.
- §11.2 reworked: failed-QA flow is **SYSTEM-managed**. Planner agents NEVER create / edit / archive QA action items. System (`Service.WipeChildrenAndRePlan`) archives all children atomically; planner spawns fresh and authors new build/sub-plan children; template `[[child_rules]]` auto-create fresh QA-twins on the new children. Planner can READ archived children for reference but creates fresh action items (no resurrection in MVP).
- New principle baked into §2 / scope thinking: **Tillsyn ENFORCES template rules; doesn't HARDWIRE behavior.** Templates define, Tillsyn enforces and errors loud. Any "the system handles X" assumption MUST be wired end-to-end (schema → resolver → consumer → integration test). Memory entry written: `feedback_tillsyn_enforces_templates.md`.

**v2.5 changelog from v2.4**:

- §22: `CASCADE_ENFORCEMENT_AND_CONTEXT_PRELOAD.md` landed. Verdicts: A (auto-create) NOT enforced; B (context preload) NOT wired; C (rename) ~30 edits; D (wipe-vs-edit) wipe-and-restart for MVP at ~150-230 LOC.
- §25 NEW: **Decision asked** — close the gaps in Drop 4c.6 (~400-600 LOC, ~6-8 additional droplets) vs hedge prompts (cascade depends on orchestrator hand-work permanently). Recommendation: close.

**v2.4 changelog from v2.3**:

- §3.5 / §7: rename intent — `default-go.toml` → `till-go.toml` and `default-generic.toml` → `till-gen.toml` for symmetry with `till-` prefix used by embedded agent groups. Rename cost pending §22 audit.
- §11.2 NEW: failed-QA handling — wipe-and-restart strategy for MVP (planner archives existing children + qa-twins; creates fresh decomposition). Audit trail preserved (archive, never delete). Edit-existing remains a future optimization.
- §22: cascade-enforcement + context-preload + rename-audit research dispatched (`CASCADE_ENFORCEMENT_AND_CONTEXT_PRELOAD.md`). Verifies code state on (a) auto-creation of plan-qa-twins / build-qa-twins, (b) context preloading per cascade kind into spawn prompts, (c) every reference to `default-*.toml` for rename cost.
- §11.1 clarification: cascade tree shape per CLAUDE.md `Cascade Tree Structure` section is canonical. `kind=plan` action item is parent; its children at level N+1 include `plan-qa-proof` + `plan-qa-falsification` (auto-created on plan creation) + `build` children + `sub-plan` children + `research` children. The plan-qa-twins ARE siblings of the builds + sub-plans — both are children of the plan. "Children" terminology is correct.

**v2.3 changelog from v2.2**:

- §4.2: cascade-dogfooding model assignments — planners + builders run **sonnet** (default in `[agents]`); QA pair (proof + falsification, plan-side + build-side) runs **opus**; commit-message-agent runs **haiku**. Supersedes prior memory's "opus builders pre-dogfood" rule once dogfooding starts.
- §11.1: NEW — Plan-QA vs Build-QA verification axes. Same agent .md files (`qa-proof-agent.md` / `qa-falsification-agent.md`); branch on `parent.kind`. Plan-QA verifies atomic decomposition + parallelization + cascade-tree shape; Build-QA verifies spec conformance + evidence + counterexample attacks.
- `PROPOSED_AGENT_DRAFTS/till-go/planning-agent.md`: NEW "Cascade Design — atomic droplets + parallelization (HARD RULES)" section added. Hard sizing constraints (1-4 code blocks, 80-120 LOC + tests, ideally one production file). Multi-level decomposition rules (one level per spawn). Parallelization rules (lock-graph wiring). Plan-QA-falsification heads-up — write the plan to survive falsification.
- "Common failure modes to avoid" expanded — droplet-too-big, droplet-too-small, decomposing-all-the-way-down-in-one-spawn, missing-parallelization.

**v2.2 changelog from v2.1**:

- §2 Boundary: corrected isolation framing — argv-level isolation is ALREADY enforced by `--bare` per Anthropic docs (Path B / system CLAUDE.md / skills / project CLAUDE.md / hooks all skipped). Prior research's "two-paths model" was misleading without a `--bare` qualifier. Actual gap: bundle stub body, not Path B leak.
- §17.1: planner draft hardening list APPLIED. Draft on disk at `workflow/drop_4c_6/PROPOSED_AGENT_DRAFTS/till-go/planning-agent.md` for review.
- §18 W3 Wave: detailed 5 isolation-fix changes D.1-D.5 (full agent body via `//go:embed` + per-project override; frontmatter strip; defense-in-depth env vars; post-render validator; doc-comment corrections). No new argv flag needed.
- §22: ISOLATION_ENFORCEMENT_FIX.md landed; all research deliverables now in.

**v2.1 changelog from v2**:

- §2 Boundary: hardened isolation invariant added (NO inheritance from system / project / user; bundle is the entire prompt surface; all `tools_allow` / `tools_deny` enforced with NO possible extras).
- §4 Schema: `[agents]` defaults block + per-kind field-level inheritance (each kind overrides only the fields it cares about; absent fields fall through to defaults). `tools_allow` + `tools_deny` MOVED from agent .md frontmatter to `agents.toml`. `agents.local.toml` MAY override `tools_allow` (system-tool availability) but MUST NOT override `tools_deny` (safety floor).
- §4.4 NEW: render-time strip of `model:` from agent .md frontmatter when `agents.toml` has the field set; documented behavior so adopters know `agents.toml` is authoritative.
- §5 Override semantics: tools_allow / tools_deny rules added.
- §9.6 ta vendor: migration path is `hylla-shared` repo (NOT making ta packages public). Org-wide pattern for all hylla projects.
- §15 Frontmatter: shape narrowed to `name` + `description` only. No `model:`, no `tools:` (those move to agents.toml).
- §16 Tillsyn dogfood overrides: `## Hylla Feedback` section requirement REMOVED.
- §17 Planner draft: hardening list — Section 0 is stdout-only; NEVER appears in any Tillsyn artifact (description / metadata / completion_notes / comments); planner can EDIT existing build / sub-plan children (fix-up scenarios), not just create.
- §22 Pending research: `ISOLATION_ENFORCEMENT_FIX.md` dispatched.
- §14.5 memory: `project_methodology_docs_tracker.md` WRITTEN (dev approved).

---

## 1. Goal

Land everything Tillsyn needs to dogfood-on-itself with arbitrary backends, SDD-inspired prompts, and **fully isolated agent execution**:

1. `agents.toml` + `agents.local.toml` runtime config (per cascade kind: model / endpoint / retries / budgets / env handling / tools).
2. `<project>/.tillsyn/agents/<name>.md` per-project agent def shipping.
3. Embedded default agent prompts in `internal/templates/builtin/agents/<group>/<name>.md` shipped via `//go:embed`.
4. `till init` command — seeds project from embedded defaults, walks user via TUI, optionally registers Tillsyn MCP server in `.mcp.json`.
5. Default agent prompts that absorb SDD-inspired structuring guidance — planner populates typed `ActionItemMetadata` primitives; builder consumes them; QA verifies against them.
6. `SystemPromptTemplatePath` plumbing — propagate through `BindingResolved`; `render.go` reads project-local agent def and writes substantive content into bundle (instead of today's redirect stub).
7. **Isolation enforcement** — bundle is the ENTIRE prompt surface; spawned cascade agents inherit NOTHING from system, project, or user (no Path B fallthrough, no skills, no CLAUDE.md, no MCPs other than `tillsyn`, no settings beyond bundle's). Drop 4c.6 must close this enforcement; today's bundle stub leaves a hole.

---

## 2. Boundary

- **Cascade agents inherit NOTHING outside the bundle (HARD INVARIANT).** Spawned `claude` (or future `codex`) processes MUST NOT load: system `~/.claude/agents/<name>.md` (Path B), system `~/.claude/CLAUDE.md`, system `~/.claude/skills/`, project `<project>/.claude/CLAUDE.md`, project `<project>/.mcp.json`, project `<project>/.claude/agents/`, user `~/.tillsyn/CLAUDE.md` overlays, system hooks. The bundle's files are the COMPLETE prompt + settings + MCP + agent surface. `tools_allow` and `tools_deny` declared in `agents.toml` are enforced with NO possible extras. **GOOD NEWS PER §22 RESEARCH**: Tillsyn's CURRENT argv (`--bare --plugin-dir <bundle>/plugin --agent <name> --setting-sources "" --strict-mcp-config --settings ... --mcp-config ...`) already enforces this isolation per Anthropic's documented `--bare` behavior — Path B / system CLAUDE.md / skills / project CLAUDE.md / hooks / `~/.claude/settings.json` / system plugins are ALL skipped by Claude Code itself. The previous research's "two-paths model" framing was misleading without a `--bare` qualifier. The actual gap is that the bundle's `<bundle>/plugin/agents/<name>.md` ships a one-liner redirect stub instead of substantive content (`render.go:340-364`); shipping full content closes the loop. Defense-in-depth env vars (`CLAUDE_CODE_DISABLE_BACKGROUND_TASKS=1`, `CLAUDE_CODE_FORK_SUBAGENT=0`, `DISABLE_AUTOUPDATER=1`, `DISABLE_TELEMETRY=1`) and a post-render validator (fail loud on empty bundle agent body) backstop the architecture.
- **Tillsyn never holds secret VALUES.** Only env-var NAMES. Values live in dev's shell.
- **Tillsyn never validates** model name reachability, endpoint validity, key correctness. Run-time errors come back from the spawned CLI; Tillsyn surfaces them with a TOML-line pointer.
- **Tillsyn never enumerates supported providers.** Whatever combination of env-var-NAMES + base_url + model the dev declares, Tillsyn passes through. If the spawned CLI accepts it, it works.
- **Tillsyn never enumerates supported drop types.** Templates define their own work-type enums (open key-value maps with name + description). Planners / orch / dev see them via MCP. Tillsyn enforces structural rules (cascade tree shape, blocked_by ordering, kind closed-12-enum, parent-children-complete). Tillsyn does NOT dictate semantic work types.
- **Tillsyn defaults stay language-generic; specializations live in groups.** `till-gen` group is generic, `till-go` adds Go+mage discipline, future `till-gdd` adds Hylla-graph-driven flow. Adopters pick a group at `till init` time. Project's `.tillsyn/agents/` is FLAT (no group prefix); group prefix lives only on the built-in side and on the user's `~/.tillsyn/agents/<group>/` library.

---

## 3. File layout — at project root after `till init`

```
<project>/
├── .tillsyn/
│   └── agents/
│       ├── planning-agent.md         # seeded from chosen group at init
│       ├── builder-agent.md
│       ├── qa-proof-agent.md
│       ├── qa-falsification-agent.md
│       ├── research-agent.md
│       ├── closeout-agent.md
│       └── commit-message-agent.md
├── agents.toml                       # runtime config, git-tracked
├── agents.local.toml                 # gitignored user override
└── .gitignore                        # contains `agents.local.toml`
```

- 3.1 **`.tillsyn/` is git-tracked by default** (project's recommended agents). Adopters review them in PRs.
- 3.2 **`.tillsyn/agents/` is FLAT** — agent file names carry NO language prefix. Specialization is baked at init time via group choice.
- 3.3 **`agents.toml` is required** at project root. `agents.local.toml` is optional.

### 3.4 User-machine library at `~/.tillsyn/agents/<group>/<name>.md`

Per-machine agent override library. Resolution priority at spawn:

1. **Project local** — `<project>/.tillsyn/agents/<name>.md` (highest priority).
2. **User local** — `~/.tillsyn/agents/<group>/<name>.md` (per-machine library).
3. **Embedded default** — `internal/templates/builtin/agents/<group>/<name>.md` (lowest priority, fallback).

If all three miss for a referenced agent name → fail loud at spawn. The spawned `claude` process NEVER falls through to `~/.claude/agents/<name>.md`.

### 3.5 Built-in groups (in-binary, in-repo at `internal/templates/builtin/agents/`)

```
internal/templates/builtin/
├── agents/
│   ├── till-gen/                     # language-generic; no mage / Hylla / Go specifics
│   ├── till-go/                      # Go+mage tuning; NO Hylla (Hylla is GDD only)
│   └── till-gdd/                     # POST-DOGFOOD; Hylla-graph-driven; placeholder
├── default-go.toml                   # existing — agent_bindings cascade structure
├── default-generic.toml              # existing
├── agents.example.toml               # NEW — runtime config example
└── embed.go                          # extends //go:embed
```

- 3.5.1 **Group naming**: `till-` prefix communicates "shipped from Tillsyn binary." Lean accepted.
- 3.5.2 **Tillsyn's own `<this-repo>/.tillsyn/agents/` overrides** the `till-go` defaults with Hylla-first content. The Tillsyn-specific overrides are git-tracked here and serve as the dogfood reference implementation.

---

## 4. `agents.toml` schema — defaults + per-kind field-level inheritance

### 4.1 Top-level `[agents]` defaults block

```toml
# agents.toml — project recommended defaults, git-tracked

[agents]
# Defaults every cascade kind inherits from. Per-kind blocks override only the fields they care about.
client = "claude"
model = "claude-sonnet-4-6"        # planner + builder default for cascade dogfooding
effort = "medium"
max_tries = 3
max_budget_usd = 5.0
max_turns = 50
blocked_retries = 0
blocked_retry_cooldown = "30s"
auto_push = false
env_set = {}
env_from_shell = { ANTHROPIC_API_KEY = "ANTHROPIC_API_KEY" }
cli_args = []
tools_allow = ["Read", "Grep", "Glob", "Bash", "LSP"]
tools_deny = []
claude_md_addons = []
```

### 4.2 Per-kind blocks override only what differs (cascade-dogfooding model assignments)

```toml
[agents.plan]
# inherits model = sonnet
max_budget_usd = 8.0
max_turns = 80
tools_allow = ["Read", "Grep", "Glob", "Bash", "LSP", "mcp__plugin_context7_context7__resolve-library-id", "mcp__plugin_context7_context7__query-docs", "mcp__tillsyn__till_action_item", "mcp__tillsyn__till_comment"]

[agents.build]
# inherits model = sonnet
tools_allow = ["Read", "Edit", "Write", "Grep", "Glob", "Bash", "LSP", "mcp__tillsyn__till_action_item", "mcp__tillsyn__till_comment"]

# QA agents run on opus — deeper falsification + proof reasoning warrants the bigger model.
[agents.plan-qa-proof]
model = "claude-opus-4-7"
max_budget_usd = 4.0
tools_allow = ["Read", "Grep", "Glob", "Bash", "LSP", "mcp__tillsyn__till_action_item", "mcp__tillsyn__till_comment"]

[agents.plan-qa-falsification]
model = "claude-opus-4-7"
max_budget_usd = 4.0
tools_allow = ["Read", "Grep", "Glob", "Bash", "LSP", "mcp__tillsyn__till_action_item", "mcp__tillsyn__till_comment"]

[agents.build-qa-proof]
model = "claude-opus-4-7"
max_budget_usd = 3.0
tools_allow = ["Read", "Grep", "Glob", "Bash", "LSP", "mcp__tillsyn__till_action_item", "mcp__tillsyn__till_comment"]

[agents.build-qa-falsification]
model = "claude-opus-4-7"
max_budget_usd = 3.0
tools_allow = ["Read", "Grep", "Glob", "Bash", "LSP", "mcp__tillsyn__till_action_item", "mcp__tillsyn__till_comment"]

[agents.research]
# inherits model = sonnet
max_budget_usd = 3.0

[agents.commit]
model = "claude-haiku-4-5-20251001"   # commits run cheap+fast
max_budget_usd = 0.10
max_turns = 5
tools_allow = ["Read", "Bash"]
```

**Cascade-dogfooding model policy** (supersedes prior memory's "opus builders pre-dogfood" rule once dogfooding starts): planners + builders run **sonnet** for speed/cost; QA pair (proof + falsification, both plan-side and build-side) runs **opus** for deeper reasoning; commit-message-agent runs **haiku** for cheap-and-fast commit message generation.

- 4.2.1 **Field-level inheritance**: every `[agents.<kind>]` field is optional. Absent fields fall through to `[agents]` defaults. Only fields that DIFFER from default need to be present.
- 4.2.2 **Map fields (`env_set`, `env_from_shell`)**: per-key inheritance. A per-kind block sets `env_set = { K = "v" }`; the resulting effective `env_set` is the per-kind override merged onto the defaults map (per-kind keys win; defaults keys absent in per-kind survive).
- 4.2.3 **List fields (`cli_args`, `tools_allow`, `tools_deny`, `claude_md_addons`)**: full-replace if present in per-kind block; inherit from defaults if absent.

### 4.3 `agents.local.toml` user override

Field-level deep-merge OVER `agents.toml` (which itself is `[agents]` defaults + per-kind merge). Per-kind block in `.local.toml` overrides specific fields:

```toml
# agents.local.toml — gitignored

[agents.build]
model = "anthropic/claude-opus-4"
env_set = { ANTHROPIC_BASE_URL = "https://openrouter.ai/api/v1" }
env_from_shell = { ANTHROPIC_API_KEY = "OPENROUTER_API_KEY" }
tools_allow = ["Read", "Edit", "Write", "Glob", "Bash", "LSP", "mcp__tillsyn__till_action_item", "mcp__tillsyn__till_comment"]   # I don't have rg installed; using Glob+Bash instead of Grep
```

#### 4.3.1 `tools_allow` vs `tools_deny` override scope

- **`tools_allow`** — `agents.local.toml` MAY override (system-tool availability varies per machine; e.g., user without `rg` swaps Grep → Glob+Bash).
- **`tools_deny`** — `agents.local.toml` MUST NOT override. Safety floor. The project's `agents.toml` declared denial holds across all contributors regardless of machine. Reason: `tools_deny` typically encodes "this agent must not run `rm -rf`" or similar — relaxing it per-machine breaks the security model.
- Implementation: `agents.local.toml`'s `tools_deny` field, if set, fails loud at startup (`agents.local.toml [agents.<kind>] tools_deny is not user-overridable; remove the field`).

### 4.4 Frontmatter `model:` strip (render-time)

When the agent .md (whichever wins via §3.4 priority) has `model:` in YAML frontmatter AND `agents.toml` (effective resolution) has `model =` set for the kind, the render layer **strips `model:`** from the frontmatter that goes into the bundle's `<bundle>/plugin/agents/<name>.md`. `agents.toml` is authoritative; the spawned CLI sees `--model <m>` argv only.

If `agents.toml` does NOT set `model =` (neither in `[agents]` defaults nor `[agents.<kind>]`), the frontmatter `model:` survives — adopter is opting into "let the CLI's default win."

Documented in `AGENTS_CONFIG.md` (§14 ledger).

### 4.5 `env_set` vs `env_from_shell` (unchanged)

Two orthogonal fields. `env_set` carries literal k=v (non-secret runtime config). `env_from_shell` is a rename map: KEY = spawn env-var name; VALUE = orch's shell env-var name. Tillsyn does `os.Getenv(<shell-name>)` and injects under `<spawn-name>`.

OpenRouter:
```toml
env_set = { ANTHROPIC_BASE_URL = "https://openrouter.ai/api/v1" }
env_from_shell = { ANTHROPIC_API_KEY = "OPENROUTER_API_KEY" }
```

Validation: keys + values match `^[A-Za-z][A-Za-z0-9_]*$`. Missing shell var at spawn → fail loud. Tillsyn validates schema only — never the model name, endpoint, or key.

---

## 5. Override semantics summary

`agents.local.toml` field-level deep-merges OVER the resolved `agents.toml` (which is `[agents]` defaults + per-kind merge):

- **Top-level fields**: per-field replacement (local wins if present).
- **`env_set` / `env_from_shell`**: per-key merge.
- **`cli_args` / `tools_allow` / `claude_md_addons`**: full replacement.
- **`tools_deny`**: NOT user-overridable; setting it in `.local.toml` fails loud.
- **Per-kind**: if `[agents.build]` is absent in `.local.toml`, project resolution is used unchanged.

---

## 6. Examples

(OpenRouter / Bedrock / Vertex / Ollama-Cloud full examples in v1; same `env_set` / `env_from_shell` patterns apply against the `[agents]` defaults block now.)

---

## 7. `default-go.toml` after migration (cascade-structural only)

```toml
[agent_bindings.build]
agent_name = "builder-agent"             # NO `go-` prefix; Go-ness comes from group choice at init
commit_agent = "commit-message-agent"     # cascade structure — STAYS IN TEMPLATE [AGREED §21.3]

[agent_bindings.build.context]
parent = "summary"
parent_git_diff = "summary"
ancestors_by_kind = { plan = "summary" }
delivery = "system_prompt"
max_chars = 60000
max_rule_duration = "5s"
```

GONE from template (now in `agents.toml`): `model`, `effort`, `max_tries`, `max_budget_usd`, `max_turns`, `auto_push`, `blocked_retries`, `blocked_retry_cooldown`, **`tools` (now `tools_allow` + `tools_deny` in agents.toml)**. Plus `agent_name` field values renamed to drop the `go-` prefix.

---

## 8. Resolution order (deterministic, three layers)

1. **At project load**: read `agents.toml` (required — fail loud if missing). Resolve effective per-kind config = `[agents]` defaults + per-kind block. If `agents.local.toml` exists, field-level deep-merge per §5. Cache resolved per-kind structs.
2. **At spawn**: dispatcher reads `binding.agent_name` + `binding.context` from template; reads runtime block from cached `agents.toml` resolution by kind. Resolves agent .md path: project `.tillsyn/agents/<name>.md` → user `~/.tillsyn/agents/<group>/<name>.md` → embedded `till-<group>/<name>.md`. Reads body + frontmatter; render-time-strips `model:` if `agents.toml` set it (§4.4). Constructs bundle's `<bundle>/plugin/agents/<name>.md` with the substantive body. Constructs `cmd.Env` (closed POSIX baseline + `env_set` + `env_from_shell` resolved). Constructs `cmd.Args` (`--model <model>` + `--effort <effort>` + `cli_args` + `--allowed-tools <tools_allow>` + isolation flags TBD by §22 research).
3. **At failure**: surface error with `agents.toml [agents.<kind>]:<line>` pointer.

**`SystemPromptTemplatePath` plumbing**: today the field exists at `templates.AgentBinding.SystemPromptTemplatePath` (`schema.go:556`) with a validator constraining it to `.tillsyn/`-relative paths, but it's NOT propagated through `BindingResolved` and NOT consumed in `render.go`. Drop 4c.6 wires this end-to-end so the bundle's `<bundle>/plugin/agents/<name>.md` carries substantive content.

---

## 9. `till init` command (replaces `till init-dev-config`)

- 9.1 `till init-dev-config` is removed; install-time setup (DB creation, default config) folds into `till install`.
- 9.2 `till init` purpose: seed a project. Run from project root.
- 9.3 Behavior:
  - Infers project name from working-dir name; offers it as default in TUI walk.
  - User picks group (default: `till-gen`, also `till-go`; `till-gdd` greyed out until post-Hylla-rev).
  - Copies `internal/templates/builtin/agents/<group>/*.md` → `<project>/.tillsyn/agents/*.md` (flat — no group prefix in destination).
  - Copies `internal/templates/builtin/agents.example.toml` → `<project>/agents.toml`.
  - Adds `agents.local.toml` to `.gitignore`.
  - Optionally registers Tillsyn MCP server in `<project>/.mcp.json` (Claude Code's MCP config). Future Drop 4d adds equivalent codex config registration.
  - Creates the project record in the Tillsyn DB.
  - Closes with a Laslig success message.
- 9.4 Re-run safety: never overwrites existing files. Adds missing only.
- 9.5 JSON mode: `till init --json '{ ... }'` for non-interactive scripting.
- 9.6 **File-copy implementation**: vendor `fsatomic` (52 LOC, zero deps) + `configmerge` (~12kB + tests, one dep already in Tillsyn) from `ta` into Tillsyn's `internal/`. Add `VENDOR_SOURCE.md` provenance note pointing to ta's commit hash. Migration path: when `ta` reaches MVP, extract these shared packages into a NEW `hylla-shared` repo (the org-wide pattern for shared code across hylla projects). Both `ta` AND `tillsyn` then import from `hylla-shared`. **Do NOT make `ta`'s packages public** — `hylla-shared` is the canonical home for cross-project DRY code.

---

## 10. SDD-inspired spec convention (no schema changes)

Action items already carry SDD-flavored primitives. Use them. NO new schema fields.

### 10.1 Where the spec lives — co-located with the action item

- 10.1.1 **`Description`** — free-form markdown prose. Narrative.
- 10.1.2 **`Metadata.Objective`** — typed string. What this work accomplishes and why.
- 10.1.3 **`Metadata.AcceptanceCriteria`** — testable bullets. The contract.
- 10.1.4 **`Metadata.ValidationPlan`** — how each acceptance criterion is verified.
- 10.1.5 **`Metadata.RiskNotes`** — what could go wrong.
- 10.1.6 **`Metadata.DecisionLog`** — design decisions made during work.
- 10.1.7 **`Metadata.ContextBlocks`** — typed enum × severity: `note` / `constraint` / `decision` / `reference` / `warning` / `runbook` × `low` / `normal` / `high` / `critical`.
- 10.1.8 **`Metadata.KindPayload`** — `json.RawMessage` free-form. Per-kind escape hatch.
- 10.1.9 **`Metadata.CompletionContract.{StartCriteria, CompletionCriteria, CompletionChecklist}`** — `[]ChecklistItem`. Executable acceptance gate.
- 10.1.10 **`Paths` / `Files` / `Packages`** — write-scope / read-attention / lock-domain.

### 10.2 What does NOT need to be on the checklist

`CompletionContract.CompletionChecklist` MUST NOT include "all child action items in `complete` state" — domain enforces unconditionally per `action_item.go:599-616 CompletionCriteriaUnmet` (Drop 4a Wave 1.7 removed the `RequireChildrenComplete` policy bit). Children must be `complete` or `archived`.

### 10.3 Spec scales with droplet size

(See v2 §10.3 — unchanged.)

### 10.4 Relationship to Section 0

Section 0 reasoning is the agent's **stdout output** at run time. It NEVER appears inside any Tillsyn artifact (description / metadata / completion_notes / comments). The spec is cited in Section 0 Premises + Conclusion of every pass; the action item's metadata is the conclusion's PERSISTED form. Section 0 is the reasoning trace; the action item is the artifact.

### 10.5 NOT vanilla SDD — Tillsyn-flavored

No separate `<id>.spec.md` file. No project-wide source-of-truth. Specs scoped to the action item they're about. Inspired by SDD, adapted to Tillsyn's primitives.

---

## 11. Build agents — TDD requirement

Default `builder-agent.md` (in `till-go` group; identical principle in `till-gen`, mage-specific calls drop out):

1. **Read** the action-item spec.
2. **Write or update tests first** — for THIS droplet's specific functions only.
3. **Run `mage test-func <pkg> <fn>`** — confirm RED for the right reason.
4. **Implement** the change.
5. **Run `mage test-func <pkg> <fn>`** again — confirm GREEN.
6. **Mark `CompletionContract.CompletionChecklist` items `Complete: true`** as you go.
7. **Append `Metadata.DecisionLog` entries** for non-trivial design decisions.
8. **NEVER** run `mage test-pkg` or `mage ci` for verification — that's QA's scope.

### 11.2 Failed-QA handling — system-managed wipe-and-restart with synthesized failure context (MVP)

**Critical principles**:

1. **Planner agents NEVER write or affect QA action items** in default templates. QA-twins are created and managed by THE SYSTEM via template `[[child_rules]]`.
2. **The new planner does NOT see archived children at all.** No reading. No reference. No partial revival. Cleanest possible reset.
3. **The system synthesizes a failure-context prompt section** (template-customizable post-MVP) and injects it into the new planner's spawn prompt. The planner authors a fresh decomposition informed ONLY by (a) the parent plan's current state + (b) the system-supplied failure context.

When `plan-qa-proof` or `plan-qa-falsification` returns a failure verdict, the parent plan transitions to `failed` state. The orchestrator (or future dispatcher) triggers the system-managed wipe + re-plan flow:

1. **System** (via `Service.WipeChildrenAndRePlan` — Wave W10 work):
   - Collects QA failure findings from the failed QA-twins' closing comments (BEFORE archiving them).
   - Archives ALL children of the parent plan in one transaction — plan-qa-twins, builds, sub-plans, research. `LifecycleState = archived` (preserves audit trail; **NEVER delete**).
   - Synthesizes a `failure_context` artifact from the collected QA findings — list of (a) what was attempted, (b) why QA flagged it, (c) "don't repeat these mistakes" framing. Stores on the parent plan's `metadata.failure_history` (list — supports multiple cycles).
   - Transitions parent plan back to `in_progress`.
2. **Render layer** (W3 + W8 work): when the fresh planner-agent spawn fires, the system-prompt.md includes a "Prior Attempt Failed" section synthesized from `metadata.failure_history[<latest>]` — current MVP framing is hardcoded simple prose; future template `[failed_plan_prompt_template]` field lets templates customize the framing.
3. **Planner agent** reads its system-prompt + own action item content. **The planner does NOT see archived children directly** — they are NOT in its preloaded context, and the planner's prompt explicitly forbids reading them via MCP.
4. **Planner agent** authors fresh Specify block with `Metadata.RiskNotes` entries derived from the failure-context section: `"Prior attempt: <what was tried>; failed because: <reason>; this attempt avoids: <approach>."`
5. **Planner agent** creates fresh build/sub-plan/research children with corrected decomposition. **Planner does NOT create or touch QA-twins.**
6. **Template `[[child_rules]]`** fire on each new child creation: fresh plan-qa-twins on any new sub-plan, fresh build-qa-twins on any new build. System-managed; planner uninvolved.
7. Cascade dispatcher fires the new children + their fresh QA-twins. Cycle continues.

**Why "new planner is blind to archived children"**:

- **Simplest possible reset.** Zero risk of partial revival. Zero risk of the planner anchoring on bad prior decisions. Cleanest QA story (the new plan is judged on its own merits, not as a delta).
- **Saves tokens net-net.** Counter-intuitively: not loading archived children's full content (descriptions, metadata, KindPayload, etc.) into the planner's context outweighs the cost of regenerating decomposition prose. Failure-context synthesis is short (~200-500 tokens) vs full archived-children dump (~2000-5000 tokens depending on decomposition size).
- **Prevents missing things.** The dev's load-bearing concern: surgical cherry-picking from archived children invites "I'll just keep this one and update that one" cognitive load that misses subtle issues. Fresh decomposition forces full re-evaluation.
- **Template-customizable post-MVP.** Adopters who WANT surgical revival can author a template-defined `[failed_plan_prompt_template]` that includes archived-children data. Default template: clean reset.

**Same pattern for build failures** — a `build-qa-*` failure transitions the build to `failed`. System collects QA findings, archives the failed build's QA-twins, synthesizes failure_context, spawns fresh builder. Builder doesn't see archived QA-twins; sees synthesized "prior attempt failed because X" in system prompt.

**What planner agents NEVER do** (default templates):
- Create QA action items.
- Edit QA action items.
- Archive QA action items.
- Read archived children of any kind (system filters them out of context preload; planner's prompt explicitly forbids MCP-fetching them).

**What planner agents NEVER do** (always):
- Delete action items. Archive only.

**System-side responsibility** (Wave W10 work — scope expanded from prior version):
- `Service.WipeChildrenAndRePlan(parent_id)`:
  - Step 1: collect QA failure findings (from non-archived QA-twins of `parent_id`).
  - Step 2: synthesize `failure_context` summary string.
  - Step 3: archive all non-archived children atomically.
  - Step 4: append `failure_context` to `parent.metadata.failure_history` (new typed field — sized 1-N entries).
  - Step 5: transition parent back to `in_progress`.
- Render layer (W3 + W8 hooks): when assembling system-prompt.md for a fresh planner spawn AND `parent.metadata.failure_history` is non-empty, include a "Prior Attempt Failed" section with the latest entry's content. Future: template `[failed_plan_prompt_template]` customizes framing.
- `metadata.failure_history` is a NEW typed field on `ActionItemMetadata`. Adds ~30 LOC to domain layer.

### 11.1 Plan-QA vs Build-QA — different verification axes

The QA pair runs at TWO different cascade levels with DIFFERENT verification axes. Same agent .md files (`qa-proof-agent.md` / `qa-falsification-agent.md`) — the agents BRANCH on `parent.kind` to apply the right axis.

**`plan-qa-proof` axis** (parent is `kind=plan`):
- Verify decomposition produces atomic droplets per cascade-design rules in `planning-agent.md` (1-4 code blocks, 80-120 LOC + tests, ideally one production file).
- Verify parallelization graph: siblings with disjoint `paths`/`packages` have NO `blocked_by`; siblings with overlap have correct `blocked_by`.
- Verify each child's Specify block is well-formed (Objective + AcceptanceCriteria + ValidationPlan + KindPayload + ContextBlocks present and coherent).
- Verify multi-level decomposition discipline — top planner authored ONE level; sub-plans handle their own decomposition.

**`plan-qa-falsification` axis** (parent is `kind=plan`):
- Attack the decomposition: over-decomposed (too many tiny builds), under-decomposed (one giant build hiding risk).
- Attack the parallelization: missing `blocked_by` (concurrency hazards), over-`blocked_by` (artificial serialization).
- Attack the Specify blocks: under-constraining Objectives, over-constraining AcceptanceCriteria, untestable bullets, missing RiskNotes.
- Attack the cascade-tree shape: builds that should be sub-plans, sub-plans that should be builds, missing intermediate segments.

**`build-qa-proof` axis** (parent is `kind=build`):
- Verify implementation matches `metadata.AcceptanceCriteria` (each bullet — code inspection or `mage <target>` evidence).
- Verify spec-conformance: `KindPayload.changes` entries match the actual diff; `ContextBlocks.constraint` invariants are preserved.
- Audit `CompletionContract.CompletionChecklist` — every `Complete: true` has evidence; every `Complete: false` is explicitly explained (failed/deferred/N/A with reason).
- Review `Metadata.DecisionLog` — every decision traceable to evidence.

**`build-qa-falsification` axis** (parent is `kind=build`):
- Counterexample / hidden dep / contract mismatch / YAGNI attacks (existing).
- Spec-attack family: `KindPayload`-vs-final-code drift, silently dropped acceptance criteria, contract mismatches with parent plan.
- Adversarial DecisionLog review: every decision attacked; can it be false? steelman against?

The agent .md prompt for each (`qa-proof-agent.md` and `qa-falsification-agent.md`) starts by reading `parent.kind` and branching on it. Drafts after planner sign-off.

**Note on `kind=research`**: research action items do NOT auto-create QA twins (per CLAUDE.md `Required Children`). `research-agent` runs read-only investigation and posts findings via comment on its own action item. Reviewed by orchestrator at the comment level, not by spawned QA agents. This is intentional — research outputs are findings, not implementation claims, so the proof/falsification asymmetry doesn't apply. `research-agent` IS one of the 7 standard agent names in W4-A / W4-B prompt authoring scope.

---

## 12. Karpathy-style content-injection (`claude_md_addons`)

Optional list-of-paths field on `agents.toml` per kind (or in defaults). Tillsyn loads listed files at spawn time and concatenates bodies into the agent's system prompt (after the agent .md body).

**However**: the four Karpathy principles (Think Before Coding / Simplicity First / Surgical Changes / Goal-Driven Execution) BAKE into the agent .md body itself, NOT just via addons. The `claude_md_addons` field stays available for adopters who want additional behavioral overlays without forking the agent .md.

---

## 13. Drop-type customization (template-defined enums, NOT hardcoded)

(Unchanged from v2 — templates define their own work-type enums; Tillsyn enforces structural rules but NOT semantic work types. Defer implementation to post-Drop-4c.6.)

---

## 14. Methodology + tracking docs

The dev wants three durable docs to track these methodologies — for internal reference, future articles, eventual release.

### 14.1 `CASCADE_METHODOLOGY.md` (top-level, write skeleton in Drop 4c.6 W6)

(Unchanged from v2.)

### 14.2 `GDD_METHODOLOGY.md` (placeholder; populate post-dogfood)

(Unchanged from v2; §14.2.1 prior-art research note still applies.)

### 14.3 `README.md` / `docs/best-practices/` — adopter-facing

(Unchanged from v2.)

### 14.4 Methodology comparison + benchmarks

(Unchanged from v2.)

### 14.5 Memory entry — WRITTEN

`project_methodology_docs_tracker.md` written this turn (dev approved). Captures the three docs as MVP-release blockers + benchmark goal + arxiv-style ambition. Indexed in `MEMORY.md`.

---

## 15. Default agent prompt structure — proposed shape

Frontmatter standard (YAML on Markdown) — narrowed:

```yaml
---
name: <agent-name>
description: <one-line — used for routing / search>
---
```

That's it. **No `model:`** (agents.toml authoritative; render-strip if both present per §4.4). **No `tools:`** (moved to agents.toml as `tools_allow` + `tools_deny`). **No `allowedTools` / `disallowedTools`** (same).

**Body sections** (in order, every default agent .md):

1. **Role one-liner** — what this agent does in the cascade.
2. **Output format** — Section 0 4-pass (subagent variant) or 5-pass (orchestrator variant); 5-field certificate; **Section 0 is stdout-only, NEVER in Tillsyn artifacts**.
3. **Working principles** — Karpathy four (Simplicity First / Surgical Changes / Goal-Driven Execution / Think Before Coding) baked in, 4-6 lines.
4. **Evidence sources, in order** — generic for `till-gen`; mage-specific for `till-go`; Hylla-first for `till-gdd` and Tillsyn dogfood overrides.
5. **Tillsyn-flavored Specify pass** (planner only) — populate Objective / AcceptanceCriteria / ValidationPlan / RiskNotes / ContextBlocks / KindPayload / CompletionContract.
6. **Spec consumption pass** (builder + QA only) — read parent's typed metadata; halt if missing.
7. **TDD pass** (builder only) — `mage test-func` per-function red-green-refactor for `till-go`; placeholder for `till-gen`.
8. **Tillsyn coordination** — move state / update metadata / closing comment.
9. **What you do NOT do** — explicit boundary list.

---

## 16. Tillsyn dogfood overrides — `<this-repo>/.tillsyn/agents/`

Tillsyn-the-project IS a Go project. Its `<repo>/.tillsyn/agents/<name>.md` files override the `till-go` defaults with:

- **Hylla-first evidence sourcing** (the only place pre-GDD where Hylla is recommended — Tillsyn's own dogfood path).
- **Tillsyn-specific mage targets** (`mage test-func`, `mage test-pkg`, `mage ci`, never raw `go`).
- **Cross-references** to `WIKI.md § Atomic Drop Granularity` + `CLAUDE.md § Code Understanding Rules`.

These overrides are git-tracked in this repo and serve as the dogfood reference implementation.

(`## Hylla Feedback` section requirement REMOVED per dev §8.5 — closing comments stay tight, no explicit feedback discipline.)

---

## 17. Proposed default agent prompt drafts (DO NOT apply yet — for review)

### 17.1 Planner draft — APPLIED

Hardened planner draft written to `workflow/drop_4c_6/PROPOSED_AGENT_DRAFTS/till-go/planning-agent.md` (this turn). Hardening list applied:

- ✓ **Section 0 is stdout-only** — explicit "BEFORE you make any Tillsyn MCP tool call" framing; "NEVER appears inside any Tillsyn artifact" hard rule.
- ✓ **HARD RULE — Think before authoring** with the "STOP — that's a discipline violation" wording.
- ✓ **Edit existing children allowed** for fix-up scenarios (under "Decomposition rules → Edit existing children").
- ✓ **Tools removed from frontmatter** — frontmatter is `name` + `description` only.
- ✓ **`model:` removed from frontmatter** — same.
- ✓ **Planner does NOT transition parent to `complete`** — explicit; children + QA-twins gate that.
- ✓ **Common failure modes** section added (empty Objective / untestable AcceptanceCriteria / missing blocked_by / over- and under-decomposition / Section 0 leaking into description).
- ✓ **Karpathy four** baked into Working Principles section.
- ✓ **Archive over delete** for child cleanup (audit trail).

### 17.2 Other 3 agents (drafts pending planner sign-off)

Same shape applies to `builder-agent.md`, `qa-proof-agent.md`, `qa-falsification-agent.md` with role-specific bodies. Drafts written after dev signs off on the planner shape — waiting before mirroring the pattern across three more files.

---

## 18. Implementation plan — Drop 4c.6

| Wave | Scope                                                                                                                              |
| ---- | ---------------------------------------------------------------------------------------------------------------------------------- |
| W0   | `agents.toml` schema (defaults + per-kind inheritance) + override merge + position-tracking errors. New `internal/config/agents.go`. |
| W1   | Embedded defaults at `internal/templates/builtin/agents/till-gen/*` + `till-go/*` (`till-gdd` placeholder). `//go:embed`.            |
| W2   | `till init` command — TUI walk, group picker, file copy (vendored fsatomic + configmerge), `.gitignore` ensure, MCP-config optional, project-DB record. |
| W3   | `SystemPromptTemplatePath` plumbing — propagate through `BindingResolved`; `render.go` reads project-local + user-local + embedded with priority order; bundle ships substantive content (not stub). Render-time strip of frontmatter `model:` per §4.4. **Apply isolation enforcement fix per §22 research findings (5 changes D.1-D.5)**: D.1.c ship full agent body via `//go:embed` + per-project override (resolves the 1-line stub); D.2 strip + inject frontmatter `model:` / `tools:` so `agents.toml` is sole runtime authority; D.3 inject defense-in-depth env vars (`CLAUDE_CODE_DISABLE_BACKGROUND_TASKS=1`, `CLAUDE_CODE_FORK_SUBAGENT=0`, `DISABLE_AUTOUPDATER=1`, `DISABLE_TELEMETRY=1`) in `cli_claude/env.go`; D.4 post-render validator that fails loud on empty bundle agent body; D.5 correct misleading doc-comments at `render.go:307-319` and `SPAWN_PIPELINE.md:24-31` to reflect `--bare`-collapsed isolation. **No new argv flag needed** — current argv shape is correct. |
| W4   | Default agent prompt content drafted across both groups + Tillsyn dogfood overrides.                                               |
| W5   | `default-go.toml` template thinning + agent-name renames (drop `go-` prefix); `tools` field removal (now in agents.toml).         |
| W6   | Docs: top-level `AGENTS_CONFIG.md` + `CASCADE_METHODOLOGY.md` skeleton + `GDD_METHODOLOGY.md` placeholder + README pointers.       |

**Estimated**: 24-32 droplets, ~3-5 days. Big drop — see §20 Path 1 vs Path 2 question. **Adjusted upward from v2's 22-30 to absorb §22 isolation-enforcement fix work**.

---

## 19. Test plan

(Unchanged from v2; ADD)

- Inheritance: `[agents]` defaults + per-kind override; map-key merge; list full-replace; absent kind falls through.
- `tools_deny` user-override rejection: `agents.local.toml [agents.<kind>] tools_deny = [...]` fails loud.
- Frontmatter strip: agent .md with `model:` + agents.toml with `model =` → bundle's stub frontmatter has no `model:`.
- **Isolation enforcement** (per §22): inject sentinels into `~/.claude/CLAUDE.md`, `~/.claude/agents/<name>.md`, `<project>/.claude/CLAUDE.md`; verify spawned process's actual prompt does NOT contain sentinels. Concrete test patterns from §22 research.

---

## 20. Drop sequencing — pick a path

(Unchanged from v2; estimates updated per §18.)

- 20.1 **Path 1 — Drop 4c.6 absorbs everything (~24-32 droplets, ~3-5 days, single PR).**
- 20.2 **Path 2 — Split into 4c.6 + 4c.7.**
- 20.3 **My lean: Path 1.**

---

## 21. Open questions — need dev decision

- 21.1 **`tools_allow` / `tools_deny` location** — RESOLVED: moved to `agents.toml` per §4 v2.1. Frontmatter NO longer carries tools.
- 21.2 **`auto_push` location** — AGREED: `agents.toml` defaults block.
- 21.3 **`commit_agent` location** — AGREED: stays in template.
- 21.4 **Override granularity** — AGREED: field-level deep-merge.
- 21.5 **`cli_args` merge semantics** — AGREED: full replacement.
- 21.6 **Group naming** — AGREED: `till-gen` / `till-go` / `till-gdd` (`till-` prefix).
- 21.7 **Path 1 vs Path 2 sequencing** — Lean Path 1 for dogfood urgency; pending dev call.
- 21.8 **`claude_md_addons` field** — AGREED: optional list-of-paths field per agent kind.
- 21.9 **Resolution priority** — AGREED: project `.tillsyn/agents/` → user `~/.tillsyn/agents/<group>/` → embedded default; fail loud if all miss.
- 21.10 **`till init` re-run safety** — AGREED: never overwrites; only adds missing.
- 21.11 **`till init` MCP-config update** — AGREED in spirit; exact UX TBD.
- 21.12 **ta versioning + reuse** — AGREED: vendor `fsatomic` + `configmerge` for now; future migration to `hylla-shared` repo (NOT making ta packages public).
- 21.13 **Frontmatter strip behavior** — AGREED: render-strips `model:` if agents.toml sets it.
- 21.14 **`tools_allow` user override** — AGREED: allowed (system-tool variation per machine).
- 21.15 **`tools_deny` user override** — AGREED: forbidden (safety floor); fails loud at startup.
- 21.16 **Isolation enforcement details** — pending §22 research deliverable.
- 21.17 **Planner draft full sign-off** — pending hardening (§17.1) applied; ready for dev re-review then.

---

## 22. Research deliverables — ALL LANDED

- ✓ `RESEARCH/SPEC_DRIVEN_REVIEW.md`
- ✓ `RESEARCH/AGENT_ARCHITECTURE_TRUTH.md`
- ✓ `RESEARCH/TILLSYN_SPEC_SHAPE_PROTOTYPE.md`
- ✓ `RESEARCH/TA_AND_KARPATHY_REVIEW.md`
- ✓ `RESEARCH/SPAWN_PIPELINE_TRUTH.md`
- ✓ `RESEARCH/TA_VERSIONING_AND_REUSE.md`
- ✓ `RESEARCH/ISOLATION_ENFORCEMENT_FIX.md`
- ✓ `RESEARCH/AUTENT_ARCHIVE_QA_PRIOR_PLANNING.md` — **landed v2.8.2**. Headline: today's reality differs from prior assumed state. (a) Autent today is permissive `allow *` (dogfood rule); REAL gating at capability-lease layer (`internal/app/kind_capability.go`). (b) `till.action_item.delete` MCP tool EXISTS today with `mode=archive|hard`; archive is folded into delete-mode, not separate. (c) Comments have ZERO role-gating today; "QA only comments on own action item" is a FRESH structural-rule candidate. (d) `Service.RestoreActionItem` may have a role-gate gap. (e) QA wipe-and-replan flow specced in §11.2 but NOT built; N-failure escalation unspecced. W11 + W10 scopes expanded accordingly.
- ✓ `RESEARCH/CASCADE_ENFORCEMENT_AND_CONTEXT_PRELOAD.md` — **landed v2.5**. Headline: pattern of shipped-but-never-wired. Schema + resolver + tests for both `ChildRulesFor` (auto-create) and `context.Resolve` (preload aggregator engine) are present in code; production consumers were never built. Drop 3 droplet 3.20 was supposed to land the auto-create consumer; it never materialized. Today's bundle's `system-prompt.md` is a 6-field stub with no parent / ancestors / siblings content. Verdicts:
  - **A — auto-create**: NOT enforced today. Schema + resolver shipped (`internal/templates/child_rules.go:96-123`); zero call sites in `internal/app` / `internal/adapters` / `cmd`. Orchestrator hand-creates twins.
  - **B — context preload**: NOT wired today. Aggregator engine fully implemented (`internal/app/dispatcher/context/aggregator.go:243-453`); zero call sites in production code. `render.go:assembleSystemPromptBody` (lines 246-279) authors the 6-field stub with no parent / ancestors / siblings.
  - **C — rename audit**: 12 source files + ~30 string edits. Load-bearing sites: `internal/templates/embed.go:34` (`//go:embed`), `embed.go:136-138` (resolver switch), `embed.go:178` (`BuiltinTemplateNames` wire literal), `internal/app/service_test.go:6534` (fixture). Zero top-level MD references. ~30-60 minutes mechanical edit.
  - **D — wipe-vs-edit**: `domain.ActionItem.Archive` is single-row only (`action_item.go:619-624`); no cascade method exists. Wipe-and-restart needs ~150-230 LOC for `Service.WipeChildrenAndRePlan` + tests + optional CLI. Edit-existing has zero Go cost but pushes 200-400 prompt words + subtle-failure risk. SKETCH §11.2 wipe-for-MVP call is well-founded.

**Synthesis**: closing A + B + D gaps costs ~400-600 LOC + tests. Drop 4c.6 prompt-draft assumptions ("system properly auto triggers", "agents auto get all the context") are NOT supported by today's code. Decision needed: close the gaps in Drop 4c.6 (unlock prompts as drafted) OR hedge the prompts (cascade depends on orchestrator hand-work permanently). My recommendation: close the gaps — see §25.

---

## 25. Decision LOCKED — Option A-split (3 drops, close all gaps)

Dev's call: close the gaps. Three drops:

- **Drop 4c.6** — config layer + isolation fix. Waves W0 + W1 + W2 + W3 + W5 + W6.
- **Drop 4c.7** — cascade wiring. Waves W7 + W8 + W9 + W10.
- **Drop 4c.8** — default agent prompts + Tillsyn dogfood overrides. Wave W4.

All three must ship before dogfood. Dependencies: 4c.6 → 4c.7 (sequential; W3 frontmatter strip + bundle full content needed by W8 context preload). 4c.7 → 4c.8 (sequential; W7+W8 must work for prompt drafts to assume auto-create + context preload).

**Droplet counts are NOT specified at this sketch level** per `feedback_plan_down_build_up.md` — the planner-agent decomposes each wave into however many droplets fit the work; plan-QA verifies the decomposition is well-formed at each level. Per-wave time estimates below are PROVISIONAL guidance based on wave scope, NOT caps on actual droplet counts. Actual decomposition happens during the cascade-flavored plan-QA pass on this sketch.

### 25.1 — Drop 4c.6 (FOUNDATION)

**Scope**: config layer + isolation fix. Waves W0 + W1 + W2 + W3 + W5 + W6.

| Wave | Scope                                                                                                                              |
| ---- | ---------------------------------------------------------------------------------------------------------------------------------- |
| W0   | `agents.toml` schema (defaults + per-kind inheritance) + override merge + `tools_allow`/`tools_deny` + frontmatter-strip render-time + position-tracking errors. New `internal/config/agents.go`. |
| W0.5 | **TEMPLATE VALIDATION + LOAD-TIME FAIL-LOUD**. Validators: `[[child_rules]]` cycle detector; `blocked_by` acyclicity invariant (load-time + runtime); `agent_name` existence across agent groups + project `.tillsyn/agents/`; kind closed-12-enum membership in `[agents.<kind>]` + `[agent_bindings.<kind>]`; `[[child_rules]]` recursion-depth bound (qa-twins-of-qa-twins style); claim-vs-implementation coherence (every claimed `[[child_rules]]` output kind supported by cascade tree shape). Each validator emits structured error with TOML-line pointer + "template is broken because X; cannot ingest" message. Closes the schema-shipped-but-not-validated gap per `feedback_tillsyn_enforces_templates.md`. |
| W1   | Embedded defaults dirs at `internal/templates/builtin/agents/till-gen/` + `till-go/` + `till-gdd/` (placeholders/scaffolds, NO substantive content yet — that's Drop 4c.8). `//go:embed` extension. `agents.example.toml` shipped. |
| W2   | `till init` command — TUI walk, group picker, file copy (vendored `fsatomic` + `configmerge` from `ta`), `.gitignore` ensure, MCP-config optional, project-DB record. **Eliminate `till init-dev-config`** (folds into `till install`). |
| W3   | `SystemPromptTemplatePath` plumbing + bundle full content per ISOLATION fix D.1-D.5: full agent body via `//go:embed` + per-project override; frontmatter `model:` + `tools:` strip; defense-in-depth env vars (`CLAUDE_CODE_DISABLE_BACKGROUND_TASKS=1`, `CLAUDE_CODE_FORK_SUBAGENT=0`, `DISABLE_AUTOUPDATER=1`, `DISABLE_TELEMETRY=1`); post-render validator (fail loud on empty bundle agent body); doc-comment corrections at `render.go:307-319` and `SPAWN_PIPELINE.md:24-31`. |
| W5   | Template thinning: `default-go.toml` → `till-go.toml`, `default-generic.toml` → `till-gen.toml` rename + agent-name renames (drop `go-` prefix in default-go.toml's `[agent_bindings.<kind>] agent_name = "..."`). `tools` field removal from frontmatter (now in `agents.toml`). |
| W6   | Docs: top-level `AGENTS_CONFIG.md` + `CASCADE_METHODOLOGY.md` skeleton + `GDD_METHODOLOGY.md` placeholder + README pointers + update `SPAWN_PIPELINE.md` + `CLI_ADAPTER_AUTHORING.md`. |

**Estimated wave scope**: ~3 days; droplet count determined by planner during plan-down (no cap).

### 25.2 — Drop 4c.7 (CASCADE WIRING)

**Scope**: close the shipped-but-not-wired gaps. Waves W7 + W8 + W9 + W10.

| Wave | Scope                                                                                                                              |
| ---- | ---------------------------------------------------------------------------------------------------------------------------------- |
| W7   | Auto-create wiring: wire `templates.ChildRulesFor` consumer into `Service.CreateActionItem`. Tests: integration test that creating a `kind=plan` action item auto-fires `plan-qa-proof` + `plan-qa-falsification` per template rules. ~50-100 LOC + tests. |
| W8   | Context-preload wiring: wire `dispatcher/context.Resolve` consumer into `render.go:assembleSystemPromptBody`. Add new context rule for "all peer-children of parent plan" (plan-QA needs this — today's `siblings_by_kind` picks "latest round per matching kind", not "all children"). Tests: integration test that each cascade kind gets its declared context preloaded. ~100-200 LOC + tests + new rule schema. |
| W9   | Subsumed into 4c.6 W5 (rename happens there alongside template thinning). Reserved for any wiring follow-ups discovered during W7/W8. |
| W10  | `Service.WipeChildrenAndRePlan(parent_id)` — archives all non-archived children atomically; transitions parent back to `in_progress`; emits metadata note. Plus orchestrator/dispatcher hook that calls it on plan-failed transitions. Tests: integration test that QA-failure → wipe → fresh planner spawn → fresh QA-twins via auto-create rules. ~150-230 LOC + tests. |
| W11  | **RUNTIME FAIL-LOUD ON TEMPLATE-ENFORCEMENT VIOLATIONS**. Enforce template rules at the MCP boundary. Planner-role agent attempting `till.action_item.create` of `kind=plan-qa-*`/`build-qa-*` → reject with "template forbids planner agents from creating QA action items; QA-twin lifecycle is system-managed via `[[child_rules]]`." Builder-role agent attempting `till.action_item.create` → reject. Any agent attempting `till.action_item.update` on a system-auto-created QA action item → reject. File-write attempts outside declared `paths` → reject at spawn-pipeline layer. `till.action_item.delete` from any agent → reject (only archive allowed). Template-defined "violation policy" enum (`reject` / `warn` / `allow`) configurable per kind; default `reject`. Each rejection emits structured error with action-item ID + role + attempted operation + template-rule citation. |

**Estimated wave scope**: ~2 days; droplet count determined by planner during plan-down (no cap). Per dev's "don't let stuff fall through cracks" guidance, the planner is biased toward smaller per-droplet sizes — but that's a quality guideline, not a count cap.

### 25.3 — Drop 4c.8 (DEFAULT PROMPTS + DOGFOOD)

**Scope**: substantive agent prompt content. Wave W4.

| Wave | Scope                                                                                                                              |
| ---- | ---------------------------------------------------------------------------------------------------------------------------------- |
| W4-A | `till-gen` agent prompts (planning + builder + qa-proof + qa-falsification + research + closeout + commit-message). Language-generic; no mage / Hylla / Go specifics. |
| W4-B | `till-go` agent prompts (same 7). Go+mage tuning; NO Hylla (Hylla in till-gdd post-dogfood + Tillsyn dogfood overrides). |
| W4-C | Tillsyn dogfood overrides at `<this-repo>/.tillsyn/agents/` — Hylla-first variants for Tillsyn's own dogfood. |
| W4-D | End-to-end spawn integration tests — sentinel-injection into Path B + system CLAUDE.md + project CLAUDE.md, assert spawned process never sees them; argv negative-assertions; env-var injection assertions. |

**Estimated wave scope**: ~3 days; droplet count determined by planner during plan-down (no cap). Bulk is prompt authoring (14 prompts in W4-A + W4-B at ~100-200 lines each), plus W4-C dogfood overrides, plus W4-D integration tests — but each of those decomposes into however many droplets the planner determines fit.

### 25.4 Dev's `~/.claude/agents/*.md` updates

Separate from Drop 4c.6/7/8. Happens AFTER 4c.8 lands so dev's machine agents match the new paradigm. Dev updates their own `~/.claude/agents/go-planning-agent.md` etc. to use:
- Cascade design HARD RULES (1-4 code blocks, 80-120 LOC max + tests).
- Tillsyn-flavored Specify pass.
- Section 0 stdout-only + hardened think-before-authoring.
- Plan agents NEVER touch QA action items.
- Wipe-and-restart on plan-failed (system-managed; planner just authors fresh decomposition).
- TN-per-numbered-section response style.

This is a per-machine update; happens once after 4c.8 ships, then doesn't need re-doing per drop.

### 25.5 Planner draft updates — APPLIED

Planner draft at `workflow/drop_4c_6/PROPOSED_AGENT_DRAFTS/till-go/planning-agent.md` v5 reflects all hardening + Option A-split semantics. Both items below are APPLIED on disk (not pending — per proof finding F2):

1. **"Edit existing children" section reframed as "Wipe-and-restart on plan-failed (system-managed; You Are BLIND to Archived Children)"** — planner does NOT read archived children; consumes failure context only via system-supplied `failure_context` section in spawn prompt's "Prior Attempt Failed" block.
2. **Explicit "You NEVER create, edit, or archive any QA action item"** in the "What you do NOT do" section. QA-twin lifecycle is system-managed via template `[[child_rules]]`.

Plus the v2.8.4 falsification-fix C2: cascade design HARD RULES section reframed to clarify that atomic-droplet EXISTENCE is the structural invariant; the SPECIFIC SIZING NUMBERS (1-4 code blocks, 80-120 LOC) are till-go template values that adopters running other templates may differ on. Per `feedback_tillsyn_enforces_templates.md` structural-vs-semantic split.

---

## 26. Per-Wave Specify Blocks (SDD-Inspired Demonstration)

This section authors each wave's Specify block in the same shape the planner-agent will use for action-item metadata at decomposition time. The sketch eats its own dogfood: methodology document authored using the methodology it documents.

Per `feedback_plan_down_build_up.md`: each wave's Specify is independently verifiable; plan-QA-proof verifies AcceptanceCriteria support the wave's Objective; plan-QA-falsification attacks each Specify for under/over-constraint, missing testability, integration coverage gaps, and `feedback_tillsyn_enforces_templates`-style "shipped-but-not-wired" risks.

### 26.W0 — agents.toml schema + override merge

- **Objective**: Land the runtime-config schema layer (`agents.toml` + `agents.local.toml`) with field-level inheritance (`[agents]` defaults + per-kind overrides), `tools_allow`/`tools_deny` migration from frontmatter, render-time `model:` strip, and position-tracking error reporting. Foundation for everything else.
- **AcceptanceCriteria**:
  - `internal/config/agents.go` defines `AgentRuntime` + `AgentsRegistry` types matching schema in §4.1.
  - TOML loader produces TOML-line-aware errors for missing required fields, malformed env-var names, duplicate keys.
  - Override merge implements field-level + map-level + list-replace semantics per §5; covered by golden-fixture tests.
  - `agents.local.toml` `tools_deny` field rejected with structured "not user-overridable" error.
  - Render-time frontmatter `model:` / `tools:` stripped when agents.toml has them defined.
  - `mage test-pkg ./internal/config` passes; `mage ci` green.
- **ValidationPlan**: `mage test-pkg ./internal/config`; golden-fixture override-merge tests; `mage ci`.
- **RiskNotes**:
  - TOML position-tracking via `pelletier/go-toml/v2` may have nested-map edge cases — verify with fixtures.
  - Frontmatter YAML lib choice: pick smallest dep.
  - Override merge nested-map recursion: guard against infinite loops.
- **ContextBlocks**:
  - `reference` (normal): `pelletier/go-toml/v2` is existing dep; do NOT add competing TOML lib.
  - `decision` (normal): `tools_deny` is NOT user-overridable.
  - `constraint` (high): override-merge semantics per §5 are load-bearing for §3 schema.

### 26.W0.5 — TEMPLATE VALIDATION + LOAD-TIME FAIL-LOUD (NEW)

- **Objective**: Validate templates for SEMANTIC correctness at load time. Tillsyn fails LOUDLY on circular / claim-vs-impl-mismatch / malformed templates rather than silently loading and discovering at runtime. Closes the gap that Drop 3 droplet 3.20 left implicit: schema-shipped but unvalidated.
- **AcceptanceCriteria**:
  - `[[child_rules]]` cycle detector: graph walk detects A→B→A cycles; emits structured error with TOML-line pointer for both rules in the cycle.
  - `blocked_by` acyclicity validator at template load (Drop 4a Wave 1.7 enforces at runtime; this adds load-time enforcement).
  - `agent_name` existence: every `[agent_bindings.<kind>] agent_name = "..."` resolves to a real agent .md file via 3-tier priority; missing → fail loud naming the binding's TOML line.
  - kind closed-12-enum membership: every `[agents.<kind>]` block + every `[agent_bindings.<kind>]` uses a kind from the closed enum; non-member → fail loud.
  - `[[child_rules]]` recursion-depth bound: child rules cannot trigger more than N levels deep (default 5; configurable post-MVP via template).
  - Claim-vs-impl coherence: every claimed `[[child_rules]]` output kind is supported by cascade-tree-shape rules per `CLAUDE.md § Cascade Tree Structure`.
  - Each validator emits structured error: TOML-line pointer + "template is broken because X; cannot ingest" message.
- **ValidationPlan**: malformed-template-fixture test PER validator (one fixture per error case); `mage test-pkg ./internal/templates`; integration test confirms loader rejects each fixture with correct error shape.
- **RiskNotes**:
  - Cycle-detection determinism: sorted-key traversal for reproducible error messages.
  - Recursion-depth bound: dev may want configurable per template; default 5 sufficient pre-MVP.
  - Claim-vs-impl check requires modeling cascade tree shape rules in Go — duplicate-with-CLAUDE.md risk; cite the doc as source of truth.
  - "Shipped-but-not-wired" load-time check requires consumer-discovery interlock with W7 + W8 outcomes.
- **ContextBlocks**:
  - `constraint` (critical): every malformed-template error MUST include TOML-line pointer.
  - `decision` (normal): default recursion-depth bound is 5.
  - `reference` (normal): `feedback_tillsyn_enforces_templates.md` — load-bearing principle.
  - `warning` (high): claim-vs-impl coherence requires cross-referencing CLAUDE.md cascade tree rules.

### 26.W1 — Embedded defaults dirs

- **Objective**: Ship embedded agent dirs at `internal/templates/builtin/agents/{till-gen,till-go,till-gdd}/` via `//go:embed`. Placeholder content only (substantive content lands in Drop 4c.8 W4). Plus ship `agents.example.toml`.
- **AcceptanceCriteria**:
  - Three group dirs created with placeholder `*.md` files for the 7 standard agent names.
  - `//go:embed` directive in `internal/templates/builtin/embed.go` includes the new dirs.
  - `agents.example.toml` ships with sane Anthropic-direct defaults per §4.2.
  - `mage test-pkg ./internal/templates` confirms embed FS contains expected paths.
- **ValidationPlan**: `mage test-pkg ./internal/templates`; `mage ci`.
- **RiskNotes**:
  - Placeholder content must be unambiguous (`# PLACEHOLDER — substantive content lands in Drop 4c.8 W4`); never accidentally treated as production prompt.
  - `//go:embed` glob must include only `*.md`, exclude `.DS_Store` etc.
- **ContextBlocks**:
  - `decision` (normal): till-gdd ships placeholder only (post-Hylla-rev).
  - `reference` (normal): future migration of vendored ta packages to `hylla-shared` repo (per §9.6).

### 26.W2 — `till init` command

- **Objective**: Land `till init` command. Replace `till init-dev-config` (folded into `till install`). TUI walk for project name + group choice; copy embedded defaults to `<project>/.tillsyn/agents/`; create `agents.toml`; update `.gitignore`; optionally register MCP server in `.mcp.json`; create project DB record.
- **AcceptanceCriteria**:
  - `till init` command added to `cmd/till/`; uses vendored `fsatomic` + `configmerge` from ta.
  - TUI walk via existing bubbletea infrastructure: project name (cwd-name default), group picker (till-gen / till-go default + till-gdd greyed).
  - File copy: embedded `internal/templates/builtin/agents/<group>/*.md` → `<project>/.tillsyn/agents/*.md` (FLAT, no group prefix in destination).
  - `agents.toml` copied from `agents.example.toml`.
  - `.gitignore` updated to include `agents.local.toml` (creates `.gitignore` if absent; idempotent).
  - Optional `.mcp.json` registration (TUI confirms before mutating).
  - Project DB record created so it appears in TUI.
  - Closes with Laslig success message.
  - Re-run safety: never overwrites existing files; reports added vs skipped vs already-present.
  - JSON mode: `till init --json '{...}'` for non-interactive scripting; same behaviors as TUI.
  - `till init-dev-config` removed; install-time setup folds into `till install`.
- **ValidationPlan**: `mage test-pkg ./cmd/till/...`; integration test on empty project dir; re-run-on-existing test; JSON-mode equivalence test.
- **RiskNotes**:
  - ta vendor: include `VENDOR_SOURCE.md` provenance pointing to ta's commit hash.
  - `.gitignore` mutation MUST be idempotent.
  - `.mcp.json` shape must match Claude Code's expected schema (verify via Context7).
  - TUI vs JSON mode behaviors must be IDENTICAL apart from input source.
- **ContextBlocks**:
  - `constraint` (high): `till init` re-runnable without overwriting existing files.
  - `decision` (normal): vendor `fsatomic` + `configmerge`; future migration to `hylla-shared`.
  - `reference` (normal): `feedback_md_update_qa.md` — atomic write discipline.

### 26.W3 — SystemPromptTemplatePath + bundle full content + isolation fix

- **Objective**: Wire `SystemPromptTemplatePath` end-to-end. Bundle agent file ships SUBSTANTIVE content (not stub). Apply ISOLATION_ENFORCEMENT_FIX D.1-D.5: full body via `//go:embed` + per-project override; frontmatter strip; defense-in-depth env vars; post-render validator; doc-comment corrections.
- **AcceptanceCriteria**:
  - `BindingResolved.SystemPromptTemplatePath` field added; populated by resolver.
  - `render.go:assembleAgentFileBody` reads project-local → user-local → embedded with priority order; writes substantive content.
  - 3-tier priority resolution implemented + tested.
  - Frontmatter strip: parses YAML; if agents.toml has `model =` set, strips `model:`; same for `tools:` → `tools_allow`. Documented in `AGENTS_CONFIG.md`.
  - Defense-in-depth env vars (`CLAUDE_CODE_DISABLE_BACKGROUND_TASKS=1`, `CLAUDE_CODE_FORK_SUBAGENT=0`, `DISABLE_AUTOUPDATER=1`, `DISABLE_TELEMETRY=1`) injected via `cli_claude/env.go`.
  - Post-render validator: fails loud if bundle's `<bundle>/plugin/agents/<name>.md` body is empty / stub-shaped / missing required frontmatter.
  - Doc-comment corrections at `render.go:307-319` and `SPAWN_PIPELINE.md:24-31` (per ISOLATION research §D.5).
  - Integration test: sentinel injection into Path B + system CLAUDE.md + project CLAUDE.md; assert spawned process never reads them.
- **ValidationPlan**: `mage test-pkg ./internal/app/dispatcher/cli_claude/...`; sentinel-injection integration test; `mage ci`.
- **RiskNotes**:
  - Frontmatter YAML parsing: pick smallest dep; verify edge cases.
  - Resolution priority must NOT fall through to Path B — `--bare` blocks it but verify via test.
  - `CLAUDE_CODE_FORK_SUBAGENT=0` may break if future Claude version requires it; document trade-off.
  - Stub-shape detection in validator: must match OLD stub's body shape.
- **ContextBlocks**:
  - `constraint` (critical): bundle MUST carry full body, not stub.
  - `constraint` (high): `--bare` argv flag MUST stay in argv assembly.
  - `decision` (normal): no new argv flag needed — current shape is correct.
  - `reference` (normal): `RESEARCH/ISOLATION_ENFORCEMENT_FIX.md`.
  - `warning` (critical): regression in isolation = adopters' agents inherit system state.

### 26.W5 — Template thinning + rename

- **Objective**: Rename `default-go.toml` → `till-go.toml` and `default-generic.toml` → `till-gen.toml`. Strip runtime fields from `[agent_bindings]` blocks (now in `agents.toml`). Drop `go-` prefix from `agent_name` values.
- **AcceptanceCriteria**:
  - Templates renamed (12 source files updated, ~30 string edits per CASCADE_ENFORCEMENT research §C).
  - `[agent_bindings.<kind>]` blocks have only cascade-structural fields (`agent_name`, `tools_allow`, `tools_deny`, `commit_agent`, `[…context]` rules). Runtime fields gone.
  - `agent_name` values renamed (drop `go-` prefix).
  - `BuiltinTemplateNames` wire literals updated.
  - `mage ci` green.
- **ValidationPlan**: `git grep` confirms zero references to old filenames; `mage ci` green; integration test exercises template loading via new names.
- **RiskNotes**:
  - 140 workflow MD references to old names — leave alone per "never remove workflow drop files" memory.
  - Test fixtures may hardcode template names — audit during rename.
  - `BuiltinTemplateNames` wire-protocol breakage for any external consumer hardcoding old names.
- **ContextBlocks**:
  - `decision` (normal): `till-` prefix per §3.5.1 / §21.6.
  - `constraint` (normal): no top-level MD references to old names.
  - `reference` (normal): `feedback_never_remove_workflow_files.md`.

### 26.W6 — Docs

- **Objective**: Land doc skeletons + adopter-facing reference. `AGENTS_CONFIG.md`, `CASCADE_METHODOLOGY.md` skeleton, `GDD_METHODOLOGY.md` placeholder, README pointers, updates to `SPAWN_PIPELINE.md` + `CLI_ADAPTER_AUTHORING.md`.
- **AcceptanceCriteria**:
  - `AGENTS_CONFIG.md` written: schema, override semantics, env_set/env_from_shell, Bedrock/Vertex/OpenRouter/Ollama Cloud examples, frontmatter strip behavior, `claude_md_addons`.
  - `CASCADE_METHODOLOGY.md` skeleton: kind enum / role enum / structural-type / agent shape / Section 0 / Tillsyn-flavored Specify / TN-per-section / Hylla-first / TDD / QA proof-vs-falsification / blocked_by / parent-children-complete / isolation enforcement / **plan down build up spine** (lead).
  - `GDD_METHODOLOGY.md` placeholder per `project_methodology_docs_tracker.md`.
  - `SPAWN_PIPELINE.md` + `CLI_ADAPTER_AUTHORING.md` updated for `--bare`-collapsed isolation.
- **ValidationPlan**: doc review pass; `mage ci`.
- **RiskNotes**:
  - `CASCADE_METHODOLOGY.md` skeleton must be complete enough for the methodology article to cite; flesh out post-dogfood with measured benchmarks.
  - `SPAWN_PIPELINE.md` "two paths" section misleading without `--bare` qualifier — fix here.
- **ContextBlocks**:
  - `decision` (normal): three docs mandatory before MVP per `project_methodology_docs_tracker.md`.
  - `constraint` (high): "plan down build up" leads `CASCADE_METHODOLOGY.md`.
  - `reference` (normal): `project_methodology_docs_tracker.md`.

### 26.W7 — Auto-create wiring

- **Objective**: Wire `templates.ChildRulesFor` consumer into `Service.CreateActionItem` so QA-twins are auto-created when their parents are created. Closes Drop 3 droplet 3.20 gap.
- **AcceptanceCriteria**:
  - `Service.CreateActionItem` calls `tpl.ChildRulesFor(kind)` and creates declared children atomically.
  - Children get parent's project_id, parent_id, paths/packages inheritance per template, template-defined kind.
  - **Each auto-created child carries `metadata.auto_created_via_child_rule: true` marker** — required by W11's "non-system actor updating system-auto-created QA → reject" runtime check (per falsification finding C1; without this marker, W11 cannot discriminate auto-created from agent-created QA items).
  - **W0.5's claim-vs-impl validator's known-wired set updated to include `ChildRulesFor` consumer** — prevents silent staleness (per proof finding F1).
  - Integration test: creating `kind=plan` auto-fires `plan-qa-proof` + `plan-qa-falsification` with `blocked_by` pointing at the plan; both children carry `auto_created_via_child_rule: true`.
  - Integration test: creating `kind=build` auto-fires `build-qa-proof` + `build-qa-falsification`; both carry the marker.
  - Atomic transaction: parent + children either all created or none (no partial states).
  - `mage test-pkg ./internal/app` green; `mage ci` green.
- **ValidationPlan**: integration tests for auto-create on plan + build creation; failure-path test (template returns malformed rules → fail loud per W0.5 validators); `mage ci`.
- **RiskNotes**:
  - Atomic transaction: SQLite vs domain-layer boundary — clarify.
  - Children's `paths`/`packages`: inherit from parent or stay empty? Template specifies.
  - Recursion: W0.5 recursion-depth bound prevents infinite loops.
- **ContextBlocks**:
  - `constraint` (critical): atomic create — no partial state.
  - `constraint` (high): cycle detection from W0.5 prevents infinite recursion.
  - `reference` (normal): `RESEARCH/CASCADE_ENFORCEMENT_AND_CONTEXT_PRELOAD.md` §A.
  - `warning` (high): Drop 3 droplet 3.20 anti-pattern; integration test mandatory per `feedback_tillsyn_enforces_templates.md`.

### 26.W8 — Context-preload wiring + new "all peer-children" rule

- **Objective**: Wire `dispatcher/context.Resolve` consumer into `render.go:assembleSystemPromptBody`. Add new "all peer-children of parent plan" context rule for plan-QA. Per-cascade-kind context bundles per §11.1 land in spawned agents' system prompts.
- **AcceptanceCriteria**:
  - `render.go:assembleSystemPromptBody` calls `context.Resolve(binding, item, project)` and includes its output in system prompt body.
  - New context rule `all_peer_children` defined in schema; resolver implementation added; preserves token cap + timeout.
  - Plan-QA agents get parent plan + ALL its children (builds + sub-plans + research) per §11.1.
  - Build-QA agents get parent build + grandparent plan.
  - Builders get parent plan's spec (Objective / AcceptanceCriteria / ValidationPlan / KindPayload / ContextBlocks / CompletionContract).
  - Failure-context section for plan-failed scenarios (per §11.2): when `parent.metadata.failure_history` non-empty, render layer includes "Prior Attempt Failed" section.
  - **W0.5's claim-vs-impl validator's known-wired set updated to include `context.Resolve` consumer** — prevents silent staleness (per proof finding F1).
  - Integration test: spawn each cascade kind; assert system-prompt content matches §11.1 declared bundle.
- **ValidationPlan**: integration tests per cascade kind; render-layer unit tests for `all_peer_children` rule; failure-context render test; `mage ci`.
- **RiskNotes**:
  - Token cap: large parent + many siblings can blow budget; aggregator's greedy-fit + max_chars must hold.
  - `all_peer_children` semantics: distinguish "children of parent" (plan-QA needs) from "siblings of self" (`siblings_by_kind`).
  - Failure-context rendering MUST NOT leak archived children's data — only synthesized summary.
- **ContextBlocks**:
  - `constraint` (critical): plan-QA agents see ALL parent's children, not just same-kind siblings.
  - `constraint` (high): aggregator token cap enforced.
  - `decision` (normal): failure-context framing hardcoded MVP; template-customizable post-MVP.
  - `reference` (normal): `RESEARCH/CASCADE_ENFORCEMENT_AND_CONTEXT_PRELOAD.md` §B; sketch §11.1.

### 26.W9 — Wiring follow-ups (reserved)

- **Objective**: Reserved for any wiring follow-ups discovered during W7/W8 plan-QA. Rename absorbed in 4c.6 W5; if W7+W8 surface unforeseen integration gaps, this wave catches them.
- **AcceptanceCriteria**:
  - Any test failures from W7+W8 integration tests resolved.
  - Any ergonomic gaps surfaced during W7+W8 plan-QA addressed.
  - `mage ci` green across the cascade.
- **ValidationPlan**: regression suite; cascade-flavored plan-QA on W7+W8 deliverables.
- **RiskNotes**:
  - Empty wave acceptable: if W7+W8 land cleanly, drop this wave; don't manufacture work.
- **ContextBlocks**:
  - `decision` (normal): empty wave acceptable.

### 26.W10 — Service.WipeChildrenAndRePlan + failure-history field + N-failure escalation

- **Objective**: Implement system-managed failed-plan wipe per §11.2. New `Service.WipeChildrenAndRePlan(parent_id)` method. New `metadata.failure_history` typed field on `ActionItemMetadata`. Hook into orchestrator's plan-failed transition. Plus N-failure escalation (NEW per research §3.3 — currently unspecced) — after **N=3 wipe-and-replan cycles** (default; template-tunable post-MVP) without green QA, system emits attention-item to orchestrator + human. N=3 means: 1st planning attempt + 2 retries → escalate. Bounded token cost; humans don't get pinged for trivial failures.
- **AcceptanceCriteria**:
  - `metadata.failure_history` typed field added (list type; supports multiple cycles).
  - `Service.WipeChildrenAndRePlan`: collects QA findings → synthesizes `failure_context` summary → archives all children atomically → appends to `parent.metadata.failure_history` → transitions parent to `in_progress`.
  - Orchestrator (or future dispatcher) hooks into plan-failed transition.
  - Integration test: QA fails plan → wipe fires → fresh planner spawn sees synthesized failure_context in system prompt → planner authors fresh decomposition without reading archived children → fresh QA-twins auto-fire on new children.
  - `mage ci` green.
- **ValidationPlan**: integration test for full failed-plan → wipe → re-plan → green cycle; `mage ci`.
- **RiskNotes**:
  - `failure_history` MUST NOT include archived children's full content — only synthesized summary.
  - Atomic archive: SQLite transaction across all children + parent state transition.
  - Render-layer hook for "Prior Attempt Failed" depends on W8 wiring.
- **ContextBlocks**:
  - `constraint` (critical): planner BLIND to archived children.
  - `constraint` (high): atomic archive — no partial wipe.
  - `decision` (normal): failure-context framing hardcoded MVP; template-customizable post-MVP.
  - `reference` (normal): sketch §11.2.

### 26.W11 — RUNTIME FAIL-LOUD: HARDCODED STRUCTURAL + TEMPLATE-DEFINED WARNINGS (NEW; UPDATED v2.8.2 per AUTENT_ARCHIVE_QA_PRIOR_PLANNING research)

**Research-driven scope expansion** (per `RESEARCH/AUTENT_ARCHIVE_QA_PRIOR_PLANNING.md`):

Today's reality differs from prior assumed state:
- `till.action_item.delete` MCP tool EXISTS today (`extended_tools.go:1297-1344` with `mode=archive|hard`); role-gated to orchestrator+system but MCP-callable.
- `archive` is folded into `delete` (`mode=archive`) — NOT a separate operation. Domain `Archive` method called via service mode-archive branch.
- Comments have ZERO role-gating today (`extended_tools.go:2233-2363`); any session with `comment` capability posts on any action item.
- `Service.RestoreActionItem` may have a role-gate gap (research flagged for verification).
- Autent today is permissive `allow *` (dogfood rule); REAL gating lives at capability-lease layer (`internal/app/kind_capability.go`).

This means W11 has MORE work than v2.8.1 implied. Scope expanded below.

- **Objective**: Enforce structural separation-of-concerns at runtime via the MCP boundary. STRUCTURAL violations (separation-of-concerns, audit-trail integrity) are HARDCODED rejects — templates cannot override them, because they're cascade-architecture invariants not adopter-tunables. SEMANTIC violations (workflow preferences) get template-defined warnings (advisory log entries; operation proceeds). This split honors the dev's "structural rules hardcoded; semantic rules template-defined" axis (see `feedback_tillsyn_enforces_templates.md` interpretation refinement).
- **AcceptanceCriteria**:
  - **REMOVE MCP tool `till.action_item.delete` entirely** (`extended_tools.go:1297-1344` + legacy `till.delete_task` alias at `:1578-1593`). No more MCP delete exposure. `Service.DeleteActionItem` Go method stays for internal callers but loses its MCP entry point. Audit-trail invariant.
  - **REMOVE MCP archive from agent surface**: archive (today `delete mode=archive`) becomes human-only via UI/CLI. No agent-role can call it. `domain.ActionItem.Archive` stays as Go method for internal/system callers (e.g. `Service.WipeChildrenAndRePlan` from W10).
  - **Hardcoded structural rejects (NOT template-overridable)**:
    - Planner-role agent attempting `till.action_item.create/update` of `kind=plan-qa-*` / `build-qa-*` → reject. Separation-of-concerns invariant (planners NEVER touch QA — prevents prompt-injection compromise per `feedback_prompt_injection_team.md`).
    - Builder-role agent attempting `till.action_item.create/update` of any QA kind → reject.
    - Builder-role agent attempting `till.action_item.create` of any kind → reject. Builders implement; they don't decompose.
    - Any non-system actor attempting `till.action_item.update` on system-auto-created QA action item → reject.
    - File-write attempts outside action item's declared `paths` → reject at spawn-pipeline layer. Security boundary.
    - **Comment role-gating** (NEW per research §4): QA-role agents can ONLY comment on action items they OWN (their own `build-qa-*` / `plan-qa-*` items); cannot comment on parent plan/build action items. Planners and builders comment only on items they own. Schema/service today has no role-gating — fresh implementation.
    - **`Service.RestoreActionItem` role-gate verification + fix if gap exists** per research §2.3 (flagged as possible gap).
  - **Template-defined warnings (semantic, advisory)**:
    - Schema in template for advisory-warning rules: `[[advisory_warnings]] op = "..." role = "..." message = "..."`.
    - When a non-structural pattern fires (e.g., builder runs `mage test-pkg` instead of `mage test-func`), the operation PROCEEDS but a warning is logged. Adopters tune which non-structural patterns warn.
    - Warnings do NOT block operations. They flag for humans to review.
  - Each hardcoded rejection emits structured error: `{action_item_id, actor_role, attempted_op, rule_violated, recommended_fix}`.
  - Each template-defined warning emits structured log entry with template-rule citation (file:line in the template TOML).
  - Integration tests for each rejection + each warning path.
- **ValidationPlan**: integration tests for each violation; `mage test-pkg ./internal/adapters/server/mcpapi`; `mage test-pkg ./internal/app/policy`; `mage ci`.
- **RiskNotes**:
  - Role-resolution at MCP boundary: depends on auth context carrying role + owner (Drop 4a Wave 3 provides today; verify with §22 research).
  - Performance: every MCP call does structural-rule check; must be fast (in-memory map lookup; ~ns).
  - Hardcoded rejects must NOT include things adopters legitimately want to tune. Limit hardcoded set to "separation-of-concerns + audit-trail invariants" — anything else is template territory.
  - **`allow` policy DROPPED from prior design** per dev §3.1: silent allow = lossy semantics. Only `reject` (hard) and `warn` (soft) exist. Absence of a rule = no policy applies = operation proceeds without log noise.
- **ContextBlocks**:
  - `constraint` (critical): structural separation-of-concerns is HARDCODED — planners NEVER touch QA, builders NEVER decompose, no actor deletes, no MCP archive. These are cascade-architecture invariants.
  - `constraint` (high): template-defined warnings are advisory only — never block operations.
  - `decision` (normal): `allow` policy dropped; only `reject` (hardcoded structural) and `warn` (template advisory) remain.
  - `reference` (normal): `feedback_tillsyn_enforces_templates.md` (structural vs semantic split); `feedback_prompt_injection_team.md` (separation-of-concerns prevents compromise); `project_team_aware_architecture.md` (archive auth gated by ownership; MCP archive not exposed).
  - `warning` (high): role-resolution + owner-resolution at MCP boundary depend on auth context shape; team-aware extension (per `project_team_aware_architecture.md`) lands later but schema must accommodate now.

### 26.W4-A — till-gen agent prompts

- **Objective**: Author substantive prompt content for the till-gen embedded agents. Language-generic; no mage / Hylla / Go specifics. Seven prompts: planning, builder, qa-proof, qa-falsification, research, closeout, commit-message.
- **AcceptanceCriteria**:
  - All 7 prompts authored at `internal/templates/builtin/agents/till-gen/<name>.md`.
  - All prompts comply with: Karpathy four / Section 0 stdout-only / TN-per-N response style / Tillsyn-flavored Specify Pass (planner) / cascade design HARD RULES (planner) / TDD discipline placeholder (builder) / parent-kind branching (QA pair).
  - Frontmatter is `name` + `description` only (no model, no tools).
  - `mage test-pkg ./internal/templates` confirms embed FS includes all 7 files.
  - Prompts pass plan-QA review (separate pass before merge).
- **ValidationPlan**: prompt-content review (plan-QA on the prompts themselves, NOT just structural CI); `mage ci`.
- **RiskNotes**:
  - Prompt drift between till-gen / till-go / till-gdd variants — same shape, different specifics; ensure consistency.
  - `till-gen` "language-generic" must be truly language-agnostic; verify by mentally running against a non-Go project.
- **ContextBlocks**:
  - `constraint` (high): NO Hylla / NO mage / NO Go specifics in till-gen.
  - `reference` (normal): `workflow/drop_4c_6/PROPOSED_AGENT_DRAFTS/till-go/planning-agent.md` — canonical example for till-go shape; till-gen drops Go-specific bits.

### 26.W4-B — till-go agent prompts

- **Objective**: Author till-go variants of the 7 agents. Go+mage tuning. NO Hylla (Hylla in till-gdd post-dogfood + Tillsyn dogfood overrides only).
- **AcceptanceCriteria**:
  - All 7 prompts authored at `internal/templates/builtin/agents/till-go/<name>.md`.
  - All prompts include Go-specific evidence sources (LSP / git diff / Context7 + go doc / Read for non-Go).
  - Builder prompt includes `mage test-func <pkg> <fn>` per-function red-green-refactor TDD discipline.
  - Planner prompt includes cascade design HARD RULES + Tillsyn-flavored Specify Pass.
  - Prompts pass plan-QA review.
- **ValidationPlan**: prompt-content review; `mage ci`.
- **RiskNotes**:
  - NO Hylla in till-go defaults — adopters who want Hylla either fork or use till-gdd post-dogfood.
- **ContextBlocks**:
  - `constraint` (high): NO Hylla in till-go default.
  - `reference` (normal): `PROPOSED_AGENT_DRAFTS/till-go/planning-agent.md`.

### 26.W4-C — Tillsyn dogfood overrides

- **Objective**: Author Tillsyn project's own `<this-repo>/.tillsyn/agents/` overrides. Hylla-first variants; serve as dogfood reference implementation. Per dev §8.5: NO Hylla Feedback section requirement.
- **AcceptanceCriteria**:
  - All 7 (or fewer if some not needed) overrides at `<this-repo>/.tillsyn/agents/<name>.md`.
  - Overrides ADD Hylla as top-priority evidence source; ADD `mcp__hylla__*` to tools_allow; ADD cross-references to `WIKI.md` + `CLAUDE.md`.
  - NO `## Hylla Feedback` closing-comment requirement.
  - Overrides are git-tracked + flagged as the dogfood reference for adopters who want similar.
- **ValidationPlan**: prompt-content review; spawn integration test using overrides (sentinels confirm overrides take priority over till-go defaults).
- **RiskNotes**:
  - Overrides must NOT pollute embedded till-go content — separate file copies in different paths.
- **ContextBlocks**:
  - `constraint` (high): overrides at project path (`<repo>/.tillsyn/agents/`), NOT in `internal/templates/builtin/`.
  - `decision` (normal): NO Hylla Feedback section requirement (per §8.5).
  - `reference` (normal): sketch §16.

### 26.W4-D — End-to-end spawn integration tests

- **Objective**: Land integration tests that exercise the full spawn pipeline with substantive prompts. Sentinel injection into Path B + system CLAUDE.md + project CLAUDE.md confirms isolation. Argv negative-assertions + env-var injection assertions confirm spawn-shape correctness.
- **AcceptanceCriteria**:
  - Sentinel test: write `SENTINEL_USER_AGENT_INHERITED_LEAK` into fake `~/.claude/agents/<name>.md` under tempdir-rerooted `HOME`; render; assert spawned `claude` invocation never sees the sentinel.
  - Equivalent sentinels in `~/.claude/CLAUDE.md`, `<project>/.claude/CLAUDE.md`, `~/.claude/skills/`.
  - Argv negative-assertions: `--setting-sources ""` always present and never set to `user` / `project` / `local`.
  - Env-var injection assertions: all 4 defense-in-depth env vars present in every spawn.
  - Post-render validator tests: empty body fails loud; stub-shaped body fails loud; valid body passes.
  - `mage ci` green.
- **ValidationPlan**: dedicated integration test suite at `internal/app/dispatcher/cli_claude/render/render_isolation_test.go` (or similar); `mage ci`.
- **RiskNotes**:
  - Tempdir-rerooted HOME setup: must ensure tests don't pollute the real `~/.claude/` (use `t.TempDir()` + `os.Setenv("HOME", ...)`).
  - Sentinel leak detection: any sentinel string in any bundle file or argv element fails the test.
- **ContextBlocks**:
  - `constraint` (critical): integration tests are the load-bearing assurance against isolation regression.
  - `reference` (normal): `RESEARCH/ISOLATION_ENFORCEMENT_FIX.md` §E.
  - `warning` (high): regression in isolation enforcement = adopters' agents inherit system state. CI MUST catch.

---

## 27. Out of scope

- Codex CLIAdapter (Drop 4d).
- Per-spawn rate limits / concurrency caps.
- Provider-side health checks at startup.
- Migration tooling for existing deployments (pre-MVP).
- Skills / SKILL.md format adoption (post-dogfood; see §13).
- GDD methodology doc population (post-dogfood).
- `~/.claude/agents/*.md` edits to dev's machine — REVERSED per §25.4: dev's system agents WERE updated this conversation as the orchestrator's working subagents; embedded Tillsyn-shipped defaults land in Drop 4c.8 W4.
- Team-aware architecture build (per `project_team_aware_architecture.md`) — schema accommodations land in Drop 4c.6+ but full feature is post-Drop-4c.8.

---

## 28. Pending updates

This sketch will be re-updated after:

1. ✓ All research deliverables landed (§22).
2. ✓ Plan-QA-proof + plan-QA-falsification PASS verdicts on sketch v2.8.3 (5 minor findings applied → v2.8.4).
3. Planner draft v5 sign-off — pending dev review (file at `workflow/drop_4c_6/PROPOSED_AGENT_DRAFTS/till-go/planning-agent.md` post-C2-fix).
4. Other 3 main agent drafts (builder + qa-proof + qa-falsification) — drafted after planner sign-off in same shape.
5. Then: planner spawn on Drop 4c.6 actual implementation.

NO production changes until §28.3 + §28.4 land.
