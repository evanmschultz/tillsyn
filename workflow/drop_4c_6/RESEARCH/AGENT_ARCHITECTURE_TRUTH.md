# Agent Architecture Truth — Per-Project Agent Definition Shipping

Read-only research deliverable. Re-investigation of the dev's dispute over the previous research finding ("Tillsyn ships zero substantive prompt content in-repo today"). Code citations replace inferences.

---

## 1. Executive truth-finding

**Per-project agent definition shipping is ABSENT from the codebase today.** Not built, not partially built, not even planned in any tracked workflow drop. A `system_prompt_template_path` field exists on `templates.AgentBinding` (TOML schema layer) as a deferred future seam, but:

- No code reads from it (`grep` of `internal/app/dispatcher/`: only doc-comment references in `render.go:323` and `spawn.go:533`, both flagging it as a "follow-up droplet" deferral).
- It does NOT propagate to `BindingResolved` (the resolved struct the render layer consumes — see `binding_resolved.go:117-153`, no `SystemPromptTemplatePath` field).
- Neither `default-go.toml` nor `default-generic.toml` populates it on any of their 12 `[agent_bindings.<kind>]` blocks (`grep` returns zero hits in `internal/templates/builtin/`).
- The validator (`load.go:1031-1055`) constrains it to a project-relative path under `.tillsyn/` — meaning the schema EXPECTS a per-project file under `.tillsyn/`, but no code provisions, seeds, copies, or reads such a file.

The dev's expected architecture — `.tillsyn/agents/<name>.md` per-project agent system prompts seeded from a binary-embedded default at init — is **not implemented in any form**. There is no `.tillsyn/agents/` reference anywhere in the tracked tree, no `till init` / `till bootstrap` command, no agent-file seeding logic. The previous research's claim is correct in substance but understates the situation: it's not just that the per-spawn bundle ships a stub — there is no per-project source for the bundle to copy from in the first place.

---

## 2. Bundle content inventory

`Render` at `internal/app/dispatcher/cli_claude/render/render.go:125-179` writes exactly five files into the per-spawn bundle. Every byte is generated in code; nothing is read from any project-local file.

### 2.1 `<bundle>/system-prompt.md`

Written by `renderSystemPrompt` (line 220) calling `assembleSystemPromptBody` (line 246). Body is built by `strings.Builder` from the action item and project struct only:

- `task_id: <item.ID>` (line 248-250)
- `project_id: <project.ID>` (line 251-253)
- `project_dir: <project.RepoPrimaryWorktree>` (line 254-256)
- `kind: <item.Kind>` (line 257-259)
- `title: <item.Title>` (line 260-264, conditional on non-empty)
- `paths: <comma-joined item.Paths>` (line 265-269, conditional)
- `packages: <comma-joined item.Packages>` (line 270-274, conditional)
- A literal three-line move-state directive (lines 275-277).

**No role definition, no tool discipline prose, no Section 0 scaffold, no acceptance-criteria scaffold, no spec template.** The doc-comment at line 219 explicitly states "Hylla awareness is deliberately omitted per F.7.10."

### 2.2 `<bundle>/plugin/.claude-plugin/plugin.json`

Written by `renderPluginManifest` (line 294). Single JSON field `name: "spawn-<bundle.SpawnID>"` (line 299). Cosmetic plugin-manifest scaffolding for `claude` to recognize the bundle as a plugin tree.

### 2.3 `<bundle>/plugin/agents/<binding.AgentName>.md`

Written by `renderAgentFile` (line 326) calling `assembleAgentFileBody` (line 340). Body is hard-coded text:

```
---
name: <binding.AgentName>
description: Tillsyn-spawned <binding.AgentName> subagent.
allowedTools: <comma-joined binding.ToolsAllowed>     (conditional)
disallowedTools: <comma-joined binding.ToolsDisallowed>  (conditional)
---

Tillsyn-spawned subagent stub. Behavior loaded from the canonical <binding.AgentName>
template at the system-installed plugin path.
```

**This is a one-liner pointer stub.** The doc-comment at lines 307-319 explicitly states the canonical templates at `~/.claude/agents/<name>.md` "remain the source of truth for behavior — they are loaded by claude from the system-installed plugin path (Path B per memory §1), not from the per-spawn plugin (Path A)." No project-local file is consulted.

The doc-comment at line 321-324 names the deferred future evolution: `binding.SystemPromptTemplatePath` "F.7.2 landed the field; F.7.3b's MINIMAL stub does not consult it. A follow-up droplet adds template-rendering against the path." That follow-up has not landed.

### 2.4 `<bundle>/plugin/.mcp.json`

Written by `renderMCPConfig` (line 391). Hard-coded JSON registering `till serve-mcp` as a stdio MCP server named `tillsyn` (lines 396-401). Identical for every spawn.

### 2.5 `<bundle>/plugin/settings.json`

Written by `renderSettings` (line 462). Generates `permissions.{allow, ask, deny}` JSON. `allow` is `binding.ToolsAllowed` plus any persisted permission grants merged from the lister (lines 462-541). `deny` mirrors `binding.ToolsDisallowed`. `ask` is an explicit empty array (line 483).

**No project-local file is read at any point in the bundle render path.** The render is purely a function of `(item, project, binding, persisted-permission-grants)`. No filesystem read against `<project>/.tillsyn/agents/` or any equivalent location.

---

## 3. `.tillsyn/` directory references

Exhaustive `git grep -n "\.tillsyn/"` across the tree returns content under exactly these subpaths:

| Subpath | Purpose | Citations |
| --- | --- | --- |
| `.tillsyn/template.toml` | Per-project Drop 3 template override; tracked in git per `.gitignore:13-19`. | `mcp_surface.go:902-904`, `extended_tools.go:1920`, `auto_generate_steward.go:58`, `app_service_adapter_mcp.go:1941` |
| `.tillsyn/spawns/` | Per-spawn bundle root in project-mode (vs default `os.TempDir()`). Auto-added to `.gitignore` once per process. | `bundle.go:30, 54, 261, 279, 475-501, 567`, `SPAWN_PIPELINE.md:55`, `bundle_test.go:95` |
| `.tillsyn/log/` | Dev-mode log directory. | `AGENTS.md:55`, `CLAUDE.md:400`, `config.example.toml:107` |
| `~/.tillsyn/` | Stable runtime home (config.toml, tillsyn.db, logs). | `README.md:596-610, 708-712`, `cmd/till/main.go:1845` |
| `.tillsyn/tillsyn.db` | SQLite DB in dev/stable runtime homes. | `repo.go:322, 457` |

**There is NO `.tillsyn/agents/` reference anywhere in the tree.** Zero hits for `.tillsyn/agents`, `ProjectAgentsDir`, `agents.toml`, `tillsyn-dir`, `TillsynDir`. The only `agents.toml` reference is in `workflow/drop_4c_6/SKETCH.md` — a pre-planning sketch for a future runtime-config drop, NOT shipping today.

There is no `cmd/till/init.go` or `cmd/till/bootstrap.go`. The only init-style command is `till init-dev-config` (`cmd/till/help.go:377`), which writes a config TOML — not agent files.

---

## 4. Workflow MD prior planning

I searched `workflow/drop_4a/`, `workflow/drop_4b/`, `workflow/drop_4c/`, `workflow/drop_4c_5/`, `workflow/drop_4c_6/`, and `workflow/example/drops/WORKFLOW.md` for any planned per-project agent definition shipping. Results:

- **Drop 4c shipped** the per-spawn render pipeline via F.7.3b (`render.go`, this droplet's deliverable). The plan explicitly carves out `binding.SystemPromptTemplatePath` consumption as a "follow-up droplet" — see `render.go:321-324` doc-comment. That follow-up was NOT itself planned in any of Drops 4a/4b/4c/4c.5/4c.6. It is a vague "next droplet" pointer with no scoped owner, no plan item, no acceptance criteria.
- **Drop 4c.6 SKETCH** at `workflow/drop_4c_6/SKETCH.md` proposes an `agents.toml` runtime-config split (model / endpoint / env / retries / budgets / turn caps), but explicitly **defers prompt-shaping to a different drop**. SKETCH §7 question 6: *"What about the agent prompts themselves? This sketch covers RUNTIME config (model, endpoint, retries, budgets). It does NOT cover the agent system prompts (the markdown that defines builder/qa/planner behavior). PENDING: research subagent's findings on spec-driven development to decide whether prompt-shaping fields belong in `agents.toml` or stay separate."* SKETCH §10 explicitly lists "Agent system-prompt overhauls — pending research findings" under Out-of-scope.
- **Drop 4c.5** (`workflow/drop_4c_5/`) is template-ergonomics + audit-debt work — no agent-prompt content scope.
- **No workflow MD** mentions copying `.tillsyn/agents/` content, seeding agent files at project-init, or shipping a default agent-prompt set in the binary.

The closest the codebase has come to landing this is the `templates.AgentBinding.SystemPromptTemplatePath` field (`schema.go:556`), which expects a path "relative to `.tillsyn/`" per the validator. That field is the deferred seam for per-project agent prompt files, but neither the schema nor any builtin template populates it, no render code consumes it, and no creation/seeding/copy logic provisions a target file.

---

## 5. ActionItemMetadata field inventory

`ActionItemMetadata` is defined at `internal/domain/workitem.go:180-239`. Every field, classified:

| Field | Type | Classification | Doc-comment summary |
| --- | --- | --- | --- |
| `Objective` | `string` | scalar | Trim-only free-form planning objective. |
| `ImplementationNotesUser` | `string` | scalar | User-authored notes. |
| `ImplementationNotesAgent` | `string` | scalar | Agent-authored notes. |
| `AcceptanceCriteria` | `string` | scalar | Acceptance criteria prose. |
| `DefinitionOfDone` | `string` | scalar | DoD prose. |
| `ValidationPlan` | `string` | scalar | Validation plan prose. |
| `BlockedReason` | `string` | scalar | Blocked-state reason. |
| `RiskNotes` | `string` | scalar | Risk-notes prose. |
| `CommandSnippets` | `[]string` | list | Free-form command examples. |
| `ExpectedOutputs` | `[]string` | list | Expected-output descriptions. |
| `DecisionLog` | `[]string` | list | Decision-log entries. |
| `RelatedItems` | `[]string` | list | Related-item references. |
| `TransitionNotes` | `string` | scalar | State-transition notes. |
| `DependsOn` | `[]string` | list | Dependency references. |
| `BlockedBy` | `[]string` | list | Blocker references. |
| `ContextBlocks` | `[]ContextBlock` | closed-shape struct list | Typed planning context (Title, Body, Type, Importance — closed enums). |
| `ResourceRefs` | `[]ResourceRef` | closed-shape struct list | Typed resource references (ID, ResourceType, Location, PathMode, BaseAlias, Title, Notes, Tags, LastVerifiedAt). |
| **`KindPayload`** | **`json.RawMessage`** | **free-form JSON blob** | **"kind_payload" — opaque JSON validated only for syntactic validity** (`workitem.go:286-289`: `bytes.TrimSpace` + `json.Valid`). Deep-merged into defaults via `mergeKindPayloadValue` (line 574) which recursively fills missing object fields — i.e. it treats the blob as a map at any depth. |
| `CompletionContract` | `CompletionContract` | closed-shape struct | StartCriteria + CompletionCriteria + CompletionChecklist + CompletionEvidence + CompletionNotes. |
| `Outcome` | `string` | scalar | "success", "failure", "blocked", etc. — open enum. |
| `SpawnBundlePath` | `string` | scalar | Current-spawn bundle path. |
| `SpawnHistory` | `[]SpawnHistoryEntry` | closed-shape struct list | Append-only audit trail of dispatcher spawns. |
| `ActualCostUSD` | `*float64` | scalar (optional) | Most-recent spawn cost in USD. |

### 5.1 Strongest candidate map field for SDD-inspired key-value spec content

**`KindPayload json.RawMessage`** is the only true free-form map / key-value field on `ActionItemMetadata`. From the doc-comment context (the `mergeKindPayloadValue` recursive merger at lines 573-594):

> *"recursively fills missing object fields from defaults"*

It's a JSON blob whose only domain validation is `json.Valid`. Any kind can stash a custom-shaped payload here, and the merge layer treats it as a recursive map at every depth. It is the natural seam for a per-kind spec (e.g., `kind=plan` could carry `{specify: {goals: [...], journeys: [...]}, plan: {arch: ..., stack: ...}, tasks: [...]}` per the SDD `/specify` → `/plan` → `/tasks` shape).

Trade-off: there is no schema enforcement on `KindPayload` content. If the cascade wants typed spec sub-structures (e.g. `[plan_spec]` block with closed fields), the more idiomatic move is to extend `ActionItemMetadata` with new typed fields per kind rather than overloading `KindPayload`. But for an SDD pilot that wants to evolve the spec shape iteratively, `KindPayload` is the right escape hatch.

Note: `Description string` on `ActionItem` itself (`action_item.go:159`) is the current de-facto spec carrier — planners author paths/packages/criteria as Markdown prose there. It is honor-system, not template-validated.

---

## 6. Cross-check of previous research's specific claims

| Claim from `SPEC_DRIVEN_REVIEW.md` Part 2 | Verdict | Code evidence |
| --- | --- | --- |
| The per-spawn bundle's `<bundle>/plugin/agents/<name>.md` is a one-liner pointer stub. | **CONFIRMED.** | `render.go:340-364` `assembleAgentFileBody`: hard-coded frontmatter + a single body line "Tillsyn-spawned subagent stub. Behavior loaded from the canonical … template at the system-installed plugin path." |
| The cross-CLI `system-prompt.md` carries action-item structural fields only. | **CONFIRMED.** | `render.go:246-279` `assembleSystemPromptBody`: emits `task_id`, `project_id`, `project_dir`, `kind`, `title`, `paths`, `packages`, plus a generic move-state directive. No role definition, no tool discipline, no Section 0 scaffold. |
| `SystemPromptTemplatePath` field per render.go:323 ("future evolution") but the F.7.3b stub does not yet read from it. | **CONFIRMED with caveat.** | The field exists on `templates.AgentBinding` (`schema.go:556`), NOT on `dispatcher.BindingResolved` (per `binding_resolved.go:117-153`'s `ResolveBinding` body which does not copy it through). `render.go:323` doc-comment names the deferral. The previous research said "binding.SystemPromptTemplatePath" without disambiguating which binding type — strictly speaking the field lives on the raw template binding only and would need to be propagated through `ResolveBinding` before render can see it. **Caveat is structural-pedantry, not a substantive correction.** |
| Tillsyn does NOT ship the substantive cascade-agent system prompts in-repo. | **CONFIRMED and STRENGTHENED.** | Beyond what the previous research said: there is no `.tillsyn/agents/` reference anywhere in the tree, no `till init` / `till bootstrap` command, no agent-file seeding logic, no per-project agent-file source (let alone shipping path). The only project-relative `.tillsyn/` content the codebase provisions is `template.toml` (Drop 3) and `spawns/` (Drop 4c). The agent-prompt content lives exclusively at `~/.claude/agents/<name>.md` on the dev's machine, and the bundle's stub frontmatter explicitly points there. |

The previous research's substantive finding holds. The dev's intuition that `.tillsyn/agents/` "is SUPPOSED to already have the .claude/agents/*.md stuff" is **a design intent that has not been built**. There is no partial implementation, no half-shipped seed, no failing test pointing at a missing path. The gap is total.

---

## 7. Recommended next step

This is research output, not a decision — options framed neutrally for orchestrator + dev to choose between.

### Option A — Drop 4c.6 absorbs per-project agent-file shipping

Pros: keeps prompt-shaping near `agents.toml` runtime-config (one drop touches both surfaces). Resolves SKETCH §7 question 6 in the same breath as §10's deferred "Agent system-prompt overhauls" item.

Required scope additions to the SKETCH:

1. Define the per-project agent-file location. The schema seam suggests `<project_root>/.tillsyn/agents/<name>.md`, matching `system_prompt_template_path`'s validator constraint of "relative to `.tillsyn/`."
2. Decide seeding model: (a) embed default prompts (e.g. `internal/templates/agents/default-go/<name>.md`) in the binary, copy to `.tillsyn/agents/` at project init or first dispatch if absent; or (b) keep `~/.claude/agents/` as authoritative and add tooling to copy to `.tillsyn/` on demand.
3. Wire `templates.AgentBinding.SystemPromptTemplatePath` through `ResolveBinding` to `BindingResolved`, then have `render.assembleAgentFileBody` consult it. Today it stops at the schema layer.
4. Add a `till init` (or equivalent) command that materializes `.tillsyn/agents/` at project bootstrap.

Cost: SKETCH currently estimates ~6 droplets / ~1 day for runtime-config alone. Adding agent-file shipping likely doubles that scope (+4-6 droplets for embed-FS, copy logic, render plumbing, init command, gitignore handling).

### Option B — Defer per-project agent-file shipping to a dedicated drop after 4c.6

Pros: keeps Drop 4c.6 narrow (runtime-config only). Lets the agent-prompt content question be answered with a clean spec scope of its own (with SDD-shape input from `SPEC_DRIVEN_REVIEW.md`).

Drop content for the deferred drop:

- Embedded default agent prompts under `internal/templates/agents/default-go/` (Go) and a copy mechanism similar to `default-go.toml`'s embed.FS.
- Per-project `.tillsyn/agents/` seeding at project init (new `till init` subcommand or extend an existing project create/load path).
- Render-layer plumbing of `SystemPromptTemplatePath` from `templates.AgentBinding` → `dispatcher.BindingResolved` → `render.assembleAgentFileBody`.
- Optional: add an SDD-style `[spec]` block in `KindPayload` per `plan` / `build` kind, validated at the template layer (lighter weight than restructuring `ActionItemMetadata`).

### Option C — Reject the dev's expected architecture; keep `~/.claude/agents/` as the source of truth

Rationale: `.claude/agents/<name>.md` is the system-installed plugin path Claude reads natively. The per-spawn bundle's stub plus the `--allowed-tools`/`--disallowed-tools` overlay is sufficient for tool-gating, and per-spawn isolation already prevents project state from leaking. The dev's machine becomes the canonical source.

Costs: this is what's shipped today, but **it doesn't survive multi-dev or CI**. Every dev needs to install the canonical agent files into their own `~/.claude/agents/`. CI runners need the same. There is no in-repo source for new contributors to reference. If the dev's machine is the only place the cascade-agent contracts exist, the cascade is undocumented from any other dev's perspective.

### Recommendation framing (neutral)

The previous research's finding stands: the substantive prompt content is not in-repo. Options A and B both land per-project agent-file shipping; the question is scope of Drop 4c.6 vs a follow-up. Option C is the do-nothing path and is honest about today's state but creates a multi-dev / CI hole. The orchestrator + dev should decide based on (i) how soon multi-dev or CI dogfood comes online, and (ii) whether SDD-shaped spec content is going to evolve in lockstep with prompt content or independently.

---

## Hylla Feedback

Hylla was unavailable for this research session — `hylla_search_keyword` returned `"enrichment still running for github.com/evanmschultz/tillsyn@main"` for every query attempted. All findings rely on `git grep` and direct file reads.

Specific misses (logged for the drop-end aggregator):

- **Query**: `hylla_search_keyword(query="ProjectAgentsDir", artifact_ref="github.com/evanmschultz/tillsyn@main")` — and three other variants for `.tillsyn/agents`, `TillsynDir`, `agents.toml`.
- **Missed because**: enrichment-still-running. Recent ingest had not completed full enrichment by session start.
- **Worked via**: `git grep -n "<term>"` from the worktree root, which returned definitive zero/non-zero hits across `.go`, `.md`, and `.toml` files in seconds.
- **Suggestion**: Hylla could surface a "structural-only mode" or "best-effort partial results" when full enrichment is mid-flight rather than rejecting all queries — for code-existence questions like "does this string appear anywhere in the tree", grep-equivalent results would be enormously useful even without enrichment metadata. Alternatively, expose ingest-status and ETA so the caller can make an informed fallback decision.

Bash sandbox note: the agent's bash policy denied `find` and `grep -rn` against the project root (permission-denied) but allowed `git grep`. `git grep` was strictly sufficient for this research, but the asymmetry between which read-only search tools are reachable is worth noting — `git grep` only sees tracked files, which is fine here but would miss generated/untracked content for other research questions.
