# DROP_4c.6 — FOUNDATION_AGENTS_TOML_AND_ISOLATION

**State:** building
**Blocked by:** DROP_4c.5 (in build state at HEAD `3035ba0`; this drop is queued behind the 4c.5 close-out merge — see Notes § "Drop-4c.5 Sequencing").
**Paths (expected):** `internal/config/agents.go` (NEW), `internal/config/agents_test.go` (NEW), `internal/templates/load.go`, `internal/templates/load_test.go`, `internal/templates/embed.go`, `internal/templates/embed_test.go`, `internal/templates/builtin/agents/till-gen/*.md` (NEW dir), `internal/templates/builtin/agents/till-go/*.md` (NEW dir, placeholder content), `internal/templates/builtin/agents/till-gdd/*.md` (NEW dir, placeholder), `internal/templates/builtin/agents.example.toml` (NEW), `internal/templates/builtin/till-go.toml` (RENAME from `default-go.toml`), `internal/templates/builtin/till-gen.toml` (RENAME from `default-generic.toml`), `cmd/till/main.go`, `cmd/till/main_test.go`, `cmd/till/help.go`, `cmd/till/init_cmd.go` (NEW), `cmd/till/init_cmd_test.go` (NEW), `internal/app/dispatcher/binding_resolved.go`, `internal/app/dispatcher/binding_resolved_test.go`, `internal/app/dispatcher/cli_adapter.go` (BindingResolved struct definition site — ROUND-2 HF4 verified), `internal/app/dispatcher/cli_adapter_test.go` (BindingResolved struct-construction tests), `internal/app/dispatcher/cli_claude/render/render.go`, `internal/app/dispatcher/cli_claude/render/render_test.go`, `internal/app/dispatcher/cli_claude/env.go`, `internal/app/dispatcher/cli_claude/adapter.go` (env-var injection wiring), `internal/vendor/fsatomic/` (NEW vendored package), `internal/vendor/configmerge/` (NEW vendored package), `internal/vendor/VENDOR_SOURCE.md` (NEW), `AGENTS_CONFIG.md` (NEW top-level), `CASCADE_METHODOLOGY.md` (NEW top-level skeleton), `GDD_METHODOLOGY.md` (NEW top-level placeholder), `SPAWN_PIPELINE.md` (existing, doc-only updates — W6.D4 sole owner), `CLI_ADAPTER_AUTHORING.md` (existing, doc-only updates — W6.D4 sole owner), `README.md` (pointer additions only), `internal/app/service.go` (caller audit — `default-go.toml` comment at :383 + `default-generic.toml` comment at :384), `internal/app/service_test.go` (caller audit — load-bearing `default-go.toml` filesystem-path literal at :6534 + 7 doc-comment refs; ROUND-2 HF6 audit), `internal/app/auto_generate_steward_test.go` (caller audit — `default-go.toml` doc-comment at :18; ROUND-2 HF6 audit), `internal/adapters/server/common/mcp_surface.go` (caller audit — `default-go.toml` comment at :906 + `default-generic.toml` comment at :908; ROUND-2 HF6 audit, replaces over-claimed `template_service.go`), `internal/adapters/server/mcpapi/extended_tools.go` (caller audit for renamed builtin — `default-go.toml` comment at :1867).
**Packages (expected):** `internal/config` (NEW agents.go pair), `internal/templates`, `internal/app/dispatcher`, `internal/app/dispatcher/cli_claude`, `internal/app/dispatcher/cli_claude/render`, `internal/vendor/fsatomic` (NEW), `internal/vendor/configmerge` (NEW), `cmd/till`, `internal/app` (caller-audit only — read-only edits limited to import-string + call-site renames after F.5 lands), `internal/adapters/server/mcpapi` (caller-audit only — same scope), `internal/adapters/server/common` (caller-audit only — `mcp_surface.go` doc-comment refs to renamed builtins; ROUND-2 HF6).
**PLAN.md ref:** project-root `PLAN.md` (Drop 4c.6 row to be added at this drop's open; pre-Drop-2 PLAN.md isn't currently authoritative — same convention Drop 4c.5 follows).
**Workflow:** `workflow/example/drops/WORKFLOW.md`.
**Cascade concept:** `AGENT_CASCADE_DESIGN.md`.
**Started:** 2026-05-09.
**Closed:** —

## Scope

Foundation drop for the cascade-flavored SDD methodology trilogy (Drops 4c.6 / 4c.7 / 4c.8). Land the runtime-config layer (`agents.toml` + `agents.local.toml` schema + per-kind field-level inheritance + override merge), template-validation fail-loud at load time, embedded agent-dir scaffolding, the `till init` user-facing seeder, the bundle-full-content + isolation-enforcement fix wiring, the `default-*.toml` → `till-*.toml` rename + `tools` frontmatter migration, and the methodology-doc skeletons. Sketch scope is fixed by `workflow/drop_4c_6/SKETCH.md` v2.8.4 POST-QA FINAL §25.1: Waves W0 + W0.5 + W1 + W2 + W3 + W5 + W6. Out of scope: W7/W8/W9/W10/W11 (Drop 4c.7), W4-A/B/C/D substantive prompt content (Drop 4c.8) — this drop ships PLACEHOLDER prompts only at `internal/templates/builtin/agents/<group>/*.md` so the embed FS + path resolution can land without the prompt-authoring effort blocking.

The QA verdict on the sketch is locked (PASS on plan-QA-proof and plan-QA-falsification of v2.8.3, 5 minor findings applied to v2.8.4); this PLAN.md translates the sketch into atomic droplets, NOT a re-validation of the sketch.

## Per-Wave Source-of-Truth

This PLAN.md is the master index. Per-wave Specify blocks live in `SKETCH.md` § 26 (the dogfood SDD demonstration) — droplet-level Specify blocks below INHERIT from those wave-level blocks, scope-narrow to the droplet, and inline only the droplet-specific delta. Builders + plan-QA read both this file and the cited wave block in `SKETCH.md` § 26.

- W0 → `SKETCH.md` § 26.W0.
- W0.5 → `SKETCH.md` § 26.W0.5.
- W1 → `SKETCH.md` § 26.W1.
- W2 → `SKETCH.md` § 26.W2.
- W3 → `SKETCH.md` § 26.W3.
- W5 → `SKETCH.md` § 26.W5.
- W6 → `SKETCH.md` § 26.W6.

## Planner

### Decomposition shape — L1 mix of sub-plan containers and direct droplets

Per `~/.claude/agents/go-planning-agent.md` § "Multi-level decomposition — you do NOT plan all the way down in one spawn" (and per the v2.8.4 sketch §25 reaffirmation that decomposition is multi-pass), this PLAN.md is the L1 plan only. Waves whose work clearly exceeds the atomic-droplet sizing budget on first inspection emit a `kind=plan` sub-plan container row with its own scope statement; the orchestrator spawns a sub-planner agent against each sub-plan when its `blocked_by` clears, which authors the L2 PLAN.md inside the nested sub-drop directory. Waves that fit the atomic-droplet budget on first inspection emit `kind=build` droplet rows directly here.

| Wave  | L1 Shape                | Reason                                                                                                                            |
| ----- | ----------------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| W0    | sub-plan container      | New `agents.go` file with `Preset` defaults struct + per-kind inheritance merge + override semantics + frontmatter helpers + position-tracking errors estimates well past 120 LOC; multi-droplet decomposition required. |
| W0.5  | sub-plan container      | Six independent validators (`[[child_rules]]` cycle, `blocked_by` acyclicity, `agent_name` existence, kind enum, recursion-depth, claim-vs-impl) each ~40-80 LOC + tests; one validator per droplet decomposition is natural. |
| W1    | direct droplet (1)      | Placeholder agent files + `agents.example.toml` shipping is one mechanical embed-extension droplet; no semantic logic.            |
| W2    | sub-plan container      | TUI walk + group picker + 3-tier file copy + `.gitignore` ensure + MCP-config registration + project-DB record + JSON mode + `init-dev-config` removal + vendoring `fsatomic`/`configmerge` is multi-droplet by construction. |
| W3    | sub-plan container      | Five distinct ISOLATION_ENFORCEMENT_FIX changes D.1-D.5 across `binding_resolved.go` + `render.go` + `env.go` + doc-comment files; one droplet per change. |
| W5    | direct droplets (3)     | Two file renames + a frontmatter `tools` strip is small; each fits one droplet; fixture-references audit is mechanical.           |
| W6    | direct droplets (5)     | Five separate doc deliverables (`AGENTS_CONFIG.md`, `CASCADE_METHODOLOGY.md` skeleton, `GDD_METHODOLOGY.md` placeholder, `SPAWN_PIPELINE.md` + `CLI_ADAPTER_AUTHORING.md` updates, `README.md` pointers); pure markdown; one droplet per doc. |

**Vendoring W2.V (placed inside W2 sub-plan)**: vendoring `ta`'s `fsatomic` + `configmerge` packages is a sub-droplet of W2 because (a) it's W2's prerequisite, (b) it's mechanical file copy + import-rewrite + `VENDOR_SOURCE.md` provenance, and (c) the L2 sub-planner sees the full vendoring-vs-`till init` ordering shape. Documented here so plan-QA knows the vendor work is NOT a separate L1 droplet.

### Wave-by-wave decomposition

#### Wave W0 — `agents.toml` schema + override merge

##### 4c.6.W0 — sub-plan container

- **State:** done
- **Kind:** `plan` (sub-plan container; spawns its own L2 planner)
- **Directory:** `workflow/drop_4c_6/DROP_4c.6.W0_AGENTS_TOML_SCHEMA/` (created when sub-planner spawns)
- **Scope:** Land `internal/config/agents.go` with the `[agents]` defaults struct + per-kind override merge per `SKETCH.md` § 4 + § 5 + § 26.W0. New types: `AgentRuntime` (effective per-kind config), `AgentsRegistry` (loaded `agents.toml`), `Preset` (the `[agents]` block), per-kind `Override` partial-shape struct. Field-level inheritance: per-kind block overrides only fields it sets; absent fields fall through to `[agents]`. Map fields (`env_set` / `env_from_shell`) merge per-key. List fields (`cli_args` / `tools_allow` / `tools_deny` / `claude_md_addons`) full-replace. `agents.local.toml` deep-merge over the resolved `agents.toml`; `tools_deny` user-override REJECTED with structured error. TOML position-tracking errors via `pelletier/go-toml/v2` (existing dep — do NOT add a competing TOML lib per `SKETCH.md` § 26.W0 ContextBlocks). Render-time frontmatter `model:` / `tools:` strip helper exposed for W3 to call into.
- **Acceptance (L1 contract; L2 plan refines):**
  - `internal/config/agents.go` package-level types match § 4.1 + § 4.2 schema, with `pelletier/go-toml/v2` decode preserving line numbers for error reporting.
  - `Resolve(registry, kind)` returns the merged effective per-kind config; absent per-kind fields fall through to `[agents]`.
  - `Merge(localRegistry, projectRegistry)` deep-merges `.local.toml` over the resolved `agents.toml`; `tools_deny` set in `.local.toml` returns a closed sentinel error citing the TOML line.
  - Frontmatter strip helper accepts (frontmatter-string, has-model-in-toml, has-tools-in-toml) and returns frontmatter-string with the appropriate keys removed; pure function, no I/O.
  - `mage test-pkg ./internal/config` passes; coverage of override-merge is golden-fixture-based (one fixture per merge edge case).
- **Blocked by:** —
- **Source-of-truth Specify:** `SKETCH.md` § 26.W0.
- **L2 sub-planner spawn directive:** "Decompose the agents.toml schema layer into atomic droplets per `~/.claude/agents/go-planning-agent.md` § Atomic Droplet Sizing. Likely shape: D1 `Preset` + per-kind override structs + TOML decode wiring; D2 inheritance merge engine; D3 `agents.local.toml` deep-merge + `tools_deny` rejection; D4 frontmatter strip helper; D5 position-tracking error envelope. Wire `blocked_by` between droplets sharing `agents.go` (all of them — chain serializes). Tests co-located in `agents_test.go`."

#### Wave W0.5 — TEMPLATE VALIDATION + LOAD-TIME FAIL-LOUD

##### 4c.6.W0.5 — sub-plan container

- **State:** done
- **Kind:** `plan` (sub-plan container; spawns its own L2 planner)
- **Directory:** `workflow/drop_4c_6/DROP_4c.6.W0.5_TEMPLATE_VALIDATORS/`
- **Scope:** Add six load-time validators to `internal/templates/load.go` (alongside the existing `validateMapKeys` + `validateAgentBindingFiles` + `validateRequiredChildRules` + `validateChildRuleReachability` + `validateKindStructuralCoherence` chain landed in Drop 4c.5 F.5.1+F.5.2). Each new validator emits a closed sentinel error with TOML-line pointer + structured message "this template is broken because X; cannot ingest." Per `SKETCH.md` § 26.W0.5: (1) `[[child_rules]]` cycle detector (graph walk); (2) `blocked_by` acyclicity at load time (Drop 4a Wave 1.7 enforces at runtime — this duplicates the check at template-load, NOT at action-item-create); (3) `agent_name` existence across the 3-tier resolution priority (project `.tillsyn/agents/` → user `~/.tillsyn/agents/<group>/` → embedded); (4) kind closed-12-enum membership for every `[agents.<kind>]` and `[agent_bindings.<kind>]` block; (5) `[[child_rules]]` recursion-depth bound (default 5); (6) claim-vs-impl coherence (every claimed `[[child_rules]]` output kind matches the cascade-tree-shape rules in `CLAUDE.md` § Cascade Tree Structure). Each validator gets a malformed-template fixture + a passing-fixture per test.
- **Acceptance (L1 contract; L2 plan refines):**
  - All six validators land in `internal/templates/load.go` with closed sentinel errors and TOML-line pointers.
  - One malformed-template fixture per validator under `internal/templates/testdata/`, exercising the sentinel.
  - Each validator emits the structured "template is broken because X; cannot ingest" envelope.
  - Validator's claim-vs-impl coherence check (#6) maintains a Go-internal "known-wired consumer set" that includes the producers shipped today — Drop 4c.7 W7 + W8 will ADD to this set when those waves wire `ChildRulesFor` + `context.Resolve` consumers (per sketch finding F1). For Drop 4c.6 the set is empty (no consumers wired yet); the validator passes any template at load time without claim-vs-impl rejection. The set's existence + the test that exercises an "unknown-consumer claim" sentinel ARE in scope; the actual consumer additions are deferred to 4c.7.
  - `mage test-pkg ./internal/templates` passes; `mage ci` green.
- **Blocked by:** —
- **Source-of-truth Specify:** `SKETCH.md` § 26.W0.5.
- **L2 sub-planner spawn directive:** "Decompose six validators into one droplet per validator. Wire `blocked_by` between droplets sharing `internal/templates/load.go` — they ALL share that file, so a serial chain. `load_test.go` is also shared. Order should follow the dependency: kind-enum + agent_name first (load-time atomic), then cycles, then recursion-depth, then `blocked_by` acyclicity (load-time mirror of Drop 4a Wave 1.7), then claim-vs-impl last (depends on a known-wired set type). Tests use `internal/templates/testdata/` malformed fixtures — one per droplet."

#### Wave W1 — Embedded agent dirs + agents.example.toml

##### 4c.6.W1.D1 — Scaffold embedded agent dirs (placeholder content) + ship `agents.example.toml`

- **State:** done
- **Kind:** `build` (atomic droplet; `Irreducible: true`)
- **Paths:** `internal/templates/builtin/agents/till-gen/planning-agent.md` (NEW), `internal/templates/builtin/agents/till-gen/builder-agent.md` (NEW), `internal/templates/builtin/agents/till-gen/qa-proof-agent.md` (NEW), `internal/templates/builtin/agents/till-gen/qa-falsification-agent.md` (NEW), `internal/templates/builtin/agents/till-gen/research-agent.md` (NEW), `internal/templates/builtin/agents/till-gen/closeout-agent.md` (NEW), `internal/templates/builtin/agents/till-gen/commit-message-agent.md` (NEW), same 7 names under `internal/templates/builtin/agents/till-go/` and `internal/templates/builtin/agents/till-gdd/`, `internal/templates/builtin/agents.example.toml` (NEW), `internal/templates/embed.go`, `internal/templates/embed_test.go`.
- **Packages:** `internal/templates`.
- **Acceptance:**
  - 21 placeholder agent .md files shipped (3 groups × 7 standard names per `SKETCH.md` § 11.1 closing note: planning, builder, qa-proof, qa-falsification, research, closeout, commit-message). Each file body: `# PLACEHOLDER — substantive content lands in Drop 4c.8 W4` plus the YAML frontmatter `name: <agent-name>` + `description: ...placeholder...`. Unambiguous "do not treat as production prompt."
  - `agents.example.toml` ships at `internal/templates/builtin/agents.example.toml` with sane Anthropic-direct defaults per `SKETCH.md` § 4.2 (planner+builder default sonnet; QA pair opus; commit haiku; `tools_allow` per kind; `tools_deny` empty default).
  - `//go:embed` directive in `internal/templates/embed.go` extends to include `builtin/agents/till-gen/*.md`, `builtin/agents/till-go/*.md`, `builtin/agents/till-gdd/*.md`, `builtin/agents.example.toml`. Explicit per-file list (NOT glob `**/*.md`) per F.2.1 falsification-mitigation pattern carried forward from Drop 4c.5.
  - `embed_test.go` adds an FS-introspection test asserting all 21 placeholder paths + `agents.example.toml` resolve via `DefaultTemplateFS.Open`.
  - `mage test-pkg ./internal/templates` passes.
- **Blocked by:** 4c.6.W0.5 (ROUND-2 HF2 — both edit `internal/templates` Go package, must serialize on package compile/test unit; W0.5's load-time validators are foundational — claim-vs-impl validator's Go-internal known-wired-set type lands first; W1.D1's embed-list extension lands on top of that compile state).
- **Specify (droplet-scope; inherits `SKETCH.md` § 26.W1):**
  - **Objective:** Scaffold the embedded agent-dir tree + ship `agents.example.toml` so W2's `till init` has files to copy, W3's render layer has files to resolve, and Drop 4c.8 W4 has filenames + paths to author into. Placeholder content only; substantive prompt authoring is Drop 4c.8.
  - **AcceptanceCriteria:** see Acceptance bullets above.
  - **ValidationPlan:** `mage test-pkg ./internal/templates`; `mage ci` green; manual `git ls-files internal/templates/builtin/agents/` shows 21 files.
  - **RiskNotes:** Glob-vs-explicit-list pattern: re-confirm explicit list per F.2.1 falsification mitigation #2 — never `**/*.md`. Placeholder body wording must include "PLACEHOLDER" string-match so a builder mistakenly committing a stub here can't pass embedded-FS introspection silently. `agents.example.toml` semantic correctness is gated by W0's loader (chicken/egg: this droplet ships the file but does NOT load it; W0 droplets exercise the loader against this fixture later).
  - **ContextBlocks:**
    - `decision` (normal): till-gdd ships placeholder only (post-Hylla-rev — see `SKETCH.md` § 14.2 / § 21.6).
    - `constraint` (high): explicit per-file embed list — never glob.
    - `reference` (normal): `internal/templates/embed.go` is the F.2.1 + F.2.2 + F.1.3 home — read those droplet doc-comments before extending.
    - `warning` (high): the 7 standard agent names (planning, builder, qa-proof, qa-falsification, research, closeout, commit-message) MUST match the names referenced by W3's resolver and Drop 4c.8 W4's prompt-authoring scope. Mismatch = silent miss at spawn.
  - **KindPayload:** `{"changes":[{"file":"internal/templates/embed.go","symbol":"DefaultTemplateFS","action":"modify","shape_hint":"extend //go:embed list with 21 placeholder agent .md files + agents.example.toml"},{"file":"internal/templates/builtin/agents/till-gen/*","symbol":"7 placeholder agent .md files","action":"add","shape_hint":"frontmatter name+description; body PLACEHOLDER marker"},{"file":"internal/templates/builtin/agents/till-go/*","symbol":"7 placeholder agent .md files","action":"add","shape_hint":"same shape, till-go group"},{"file":"internal/templates/builtin/agents/till-gdd/*","symbol":"7 placeholder agent .md files","action":"add","shape_hint":"same shape, till-gdd group"},{"file":"internal/templates/builtin/agents.example.toml","symbol":"runtime-config example","action":"add","shape_hint":"sketch §4.2 sane defaults: sonnet planner+builder, opus QA, haiku commit"}]}`

#### Wave W2 — `till init` command

##### 4c.6.W2 — sub-plan container

- **State:** done
- **Kind:** `plan` (sub-plan container; spawns its own L2 planner)
- **Directory:** `workflow/drop_4c_6/DROP_4c.6.W2_TILL_INIT/`
- **Scope:** Land `till init` per `SKETCH.md` § 9 + § 26.W2 — TUI walk (project name + group picker), copy embedded `internal/templates/builtin/agents/<group>/*.md` → `<project>/.tillsyn/agents/*.md` FLAT, copy `agents.example.toml` → `<project>/agents.toml`, ensure `agents.local.toml` in `.gitignore`, optional `.mcp.json` registration, project-DB record creation, Laslig success message, JSON mode (`--json '{...}'`) with identical behavior, re-run safety (never overwrites). Plus: vendor `fsatomic` (52 LOC, zero deps) + `configmerge` (~12kB + tests, one dep already in Tillsyn) from `ta` to `internal/vendor/` with `VENDOR_SOURCE.md` provenance per `SKETCH.md` § 9.6. Plus: REMOVE `till init-dev-config` command from `cmd/till/main.go` (lines 1885-1901, 2039-2040+ per `git grep` confirmation 2026-05-09) — fold install-time config setup into `till install`. JSON-mode + TUI behaviors must be IDENTICAL apart from input source.
- **Acceptance (L1 contract; L2 plan refines):**
  - `till init` command is registered in `cmd/till/main.go`'s root command tree, with help text + JSON mode flag.
  - `internal/vendor/fsatomic/` + `internal/vendor/configmerge/` exist with provenance in `internal/vendor/VENDOR_SOURCE.md` (cites ta commit hash + future `hylla-shared` migration plan per `SKETCH.md` § 9.6).
  - `till init` (TUI mode) on an empty dir: copies 7 agent .md files (FLAT — no group prefix in destination), copies `agents.toml`, updates `.gitignore`, optionally writes `.mcp.json`, creates project record, prints Laslig success.
  - `till init` (JSON mode `till init --json '{"name":"...","group":"till-go","mcp":true}'`) on an empty dir: same effects; no TUI prompts.
  - Re-run safety: invocation #2 reports added=0, skipped=N already-present, no overwrites.
  - `till init-dev-config` removed from CLI surface; `cmd/till/help.go` updated accordingly; `cmd/till/main_test.go` references updated to `till install` for any test that previously exercised `init-dev-config`'s config-creation behavior.
  - `mage test-pkg ./cmd/till/...` passes; integration test on empty project dir; re-run-on-existing test; JSON-mode equivalence test.
- **Blocked by:** 4c.6.W1.D1 (W2 copies the agent .md files shipped by W1; without W1's embedded scaffolding there's nothing to copy).
- **Source-of-truth Specify:** `SKETCH.md` § 26.W2 + § 9.
- **L2 sub-planner spawn directive:** "Decompose `till init` into atomic droplets per `~/.claude/agents/go-planning-agent.md` § Atomic Droplet Sizing. Likely shape: D1 vendor `fsatomic` (separate package, no `cmd/till` collision) — **every vendored Go file MUST carry a 2-3 line block-comment header `// DO NOT EDIT — re-vendor from upstream` plus pointer to `internal/vendor/VENDOR_SOURCE.md`** (ROUND-2 OQ#2 disposition); D2 vendor `configmerge` (same header rule); D3 `cmd/till/init_cmd.go` skeleton + flag wiring + JSON mode parser; D4 TUI walk (group picker + project name) using existing bubbletea infrastructure; D5 file-copy + `.gitignore` ensure (uses fsatomic — `blocked_by` D1); D6 `.mcp.json` optional registration (verify shape via Context7 against Claude Code's expected schema); D7 project-DB record creation; **D8 remove `init-dev-config` from `main.go` + `help.go` + caller-audit `main_test.go` — PRECONDITION (ROUND-2 OQ#3 disposition): the L2 W2 sub-planner MUST verify `till install` already covers (or is extended in this sub-plan to cover) the dev-config-creation behavior currently in `cmd/till/main.go:2039 runInitDevConfig` (creates dev config file + enforces debug logging) BEFORE D8 finalizes the removal. If `till install` does NOT cover this behavior, the L2 sub-plan MUST add a new droplet that extends `till install` first. Premature `init-dev-config` removal = behavior regression and must be caught at L2 plan-QA**. Wire `blocked_by` between droplets sharing `cmd/till/init_cmd.go` (D3, D4, D5, D6, D7 — serial chain) AND between droplets sharing `cmd/till/main.go` (D3 [register init], D8 [remove init-dev-config] — order: D3 first since D8 only removes; safer than reverse). The `internal/vendor/fsatomic` + `internal/vendor/configmerge` droplets are NEW packages so they parallel-run with everything in `cmd/till` until the file-copy droplet (D5) needs them."

#### Wave W3 — SystemPromptTemplatePath plumbing + bundle full content + isolation fix

##### 4c.6.W3 — sub-plan container

- **State:** done
- **Kind:** `plan` (sub-plan container; spawns its own L2 planner)
- **Directory:** `workflow/drop_4c_6/DROP_4c.6.W3_BUNDLE_AND_ISOLATION/`
- **Scope:** Land the five ISOLATION_ENFORCEMENT_FIX changes D.1-D.5 per `SKETCH.md` § 18 / § 26.W3, plus the cross-cutting `SystemPromptTemplatePath` end-to-end plumbing. (D.1) ship full agent body via `//go:embed` + 3-tier resolution priority (`<project>/.tillsyn/agents/<name>.md` → `~/.tillsyn/agents/<group>/<name>.md` → embedded `till-<group>/<name>.md`) — replaces today's stub at `render.go:assembleAgentFileBody` (lines 340-364 per current file inspection). (D.2) frontmatter `model:` + `tools:` render-time strip when `agents.toml` has them set, per `SKETCH.md` § 4.4. (D.3) inject defense-in-depth env vars `CLAUDE_CODE_DISABLE_BACKGROUND_TASKS=1`, `CLAUDE_CODE_FORK_SUBAGENT=0`, `DISABLE_AUTOUPDATER=1`, `DISABLE_TELEMETRY=1` in `internal/app/dispatcher/cli_claude/env.go`. (D.4) post-render validator wired into the render path (NOT shipped as an unwired helper — see Acceptance bullet) — fails loud on empty / stub-shaped / missing-required-frontmatter bundle agent body. (D.5) doc-comment corrections at `render.go:307-319` (the `renderAgentFile` + `assembleAgentFileBody` block currently describing the stub design) ONLY; **`SPAWN_PIPELINE.md:24-31` rewrite is W6.D4's sole responsibility** (ROUND-2 HF3). Plus: propagate `SystemPromptTemplatePath` through `BindingResolved` (struct definition site is `internal/app/dispatcher/cli_adapter.go:102`, NOT `binding_resolved.go` — ROUND-2 HF4 verified; W3 paths therefore include `cli_adapter.go` + `cli_adapter_test.go`). Sentinel-injection integration test scaffolding lives here too (real sentinel tests get fleshed out by Drop 4c.8 W4-D, but a minimal "bundle body is non-empty after render" assertion ships in this wave).
- **Acceptance (L1 contract; L2 plan refines):**
  - `dispatcher.BindingResolved.SystemPromptTemplatePath` field exists and is populated by the resolver.
  - `render.assembleAgentFileBody` resolves project → user-local → embedded; writes substantive content (not the F.7.3b stub).
  - Render-time strip of frontmatter `model:` / `tools:` when `agents.toml` has the corresponding key set — uses the helper exposed by W0.
  - Defense-in-depth env vars injected in `cli_claude/env.go`'s closed POSIX baseline.
  - Post-render validator is **wired into the render path at `render.Render`'s exit** (or whichever render call site produces the bundle) — NOT shipped as a separate exported helper that no caller invokes (ROUND-2 HF8 — wiring contract sharpened to prevent shipped-but-not-wired anti-pattern per `feedback_tillsyn_enforces_templates.md`). The wired validator errors out if `<bundle>/plugin/agents/<name>.md` is empty, is the OLD F.7.3b stub shape, or is missing `name` / `description` frontmatter; minimal sentinel-style assertion in `render_test.go` exercises the failure path.
  - Doc-comment corrections at `render.go:307-319` (Go-source doc-comment adjacent to the resolver work) reflect `--bare`-collapsed isolation per `RESEARCH/ISOLATION_ENFORCEMENT_FIX.md` §D.5 (the prior research's "two-paths model" framing was misleading without a `--bare` qualifier; current argv shape is correct — no new flags). **`SPAWN_PIPELINE.md:24-31` rewrite is OUT OF SCOPE for W3** — sole owner is W6.D4 (ROUND-2 HF3 — duplicate-ownership cut; W6.D4's `Blocked by: 4c.6.W3` ordering is justified by "doc reflects W3-shipped reality," not "doc finishes what W3 started").
  - Minimal sentinel-style integration test: assert spawned `<bundle>/plugin/agents/<name>.md` body length > stub-threshold (e.g. > 200 chars); the FULL sentinel-injection-into-Path-B / system-CLAUDE.md test suite ships in Drop 4c.8 W4-D.
  - `mage test-pkg ./internal/app/dispatcher/cli_claude/...` passes; `mage ci` green.
- **Blocked by:** 4c.6.W1.D1 (W3's resolver consumes the embedded `till-<group>/<name>.md` files); 4c.6.W0 (W3's frontmatter strip helper is exposed by W0).
- **Source-of-truth Specify:** `SKETCH.md` § 26.W3 + `RESEARCH/ISOLATION_ENFORCEMENT_FIX.md`.
- **L2 sub-planner spawn directive:** "Decompose D.1-D.5 into atomic droplets per `~/.claude/agents/go-planning-agent.md` § Atomic Droplet Sizing. Likely shape: D1 propagate `SystemPromptTemplatePath` through `BindingResolved` — NOTE: `BindingResolved` struct lives at `internal/app/dispatcher/cli_adapter.go:102`, NOT `binding_resolved.go` (ROUND-2 HF4 verified via `git grep -n 'type BindingResolved' internal/`); D1 paths therefore include `internal/app/dispatcher/cli_adapter.go` + `internal/app/dispatcher/cli_adapter_test.go` alongside `binding_resolved.go` / `binding_resolved_test.go`; D2 3-tier agent-body resolver in `render.go:assembleAgentFileBody`; D3 frontmatter strip wiring (calls into W0's helper); D4 defense-in-depth env vars in `cli_claude/env.go` and adapter-wire-up in `cli_claude/adapter.go` if env injection lands at the adapter call site (sub-planner verifies); D5 post-render validator + minimal sentinel-style assertion in `render_test.go` — **the validator MUST be wired into `render.Render`'s exit path** (or whichever render call site produces the bundle), NOT shipped as an unwired exported helper, so every spawned bundle is validated before the spawn proceeds (ROUND-2 HF8 — wiring contract sharpened to prevent shipped-but-not-wired anti-pattern per `feedback_tillsyn_enforces_templates.md`); D6 doc-comment corrections at `render.go:307-319` ONLY (Go-source side; `SPAWN_PIPELINE.md:24-31` is W6.D4's sole responsibility per ROUND-2 HF3 — strip from D6 scope). Wire `blocked_by`: D1 → D2 (D2 reads the field D1 plumbs); D2 → D3 (D3 mutates the body D2 produces); D2 → D5 (D5 validates the body D2 produces); D4 parallels D1-D3 (different file `cli_claude/env.go`); D6 parallels everything (doc-only)."

#### Wave W5 — Template thinning + rename

##### 4c.6.W5.D1 — Rename `default-go.toml` → `till-go.toml` (file move + embed.go + caller audit)

- **State:** done
- **Kind:** `build` (atomic droplet; `Irreducible: true`)
- **Paths:** `internal/templates/builtin/till-go.toml` (RENAMED from `default-go.toml`), `internal/templates/embed.go` (//go:embed list + `LoadDefaultTemplateForLanguage` switch + `BuiltinTemplateNames`), `internal/templates/embed_test.go`, plus caller-audit edits (string literal updates only) at `internal/app/service.go`, `internal/app/service_test.go`, `internal/app/auto_generate_steward_test.go`, `internal/adapters/server/common/mcp_surface.go`, `internal/adapters/server/mcpapi/extended_tools.go` per `RESEARCH/CASCADE_ENFORCEMENT_AND_CONTEXT_PRELOAD.md` §C audit + ROUND-2 HF6 regenerated audit (caller list verified independently in ROUND-3 via `git grep "default-go.toml" cmd/ internal/` against HEAD `f32b9d8`; over-claimed `internal/app/auto_generate_steward.go` and `internal/app/template_service.go` REMOVED — both have zero `default-go.toml` refs at HEAD; load-bearing sites: `embed.go:34` directive, `embed.go:136-138` resolver switch, `embed.go:178` `BuiltinTemplateNames` literal, `internal/app/service_test.go:6534` fixture).
- **Packages:** `internal/templates`, `internal/app` (caller audit), `internal/adapters/server/mcpapi` (caller audit).
- **Acceptance:**
  - `internal/templates/builtin/default-go.toml` renamed to `internal/templates/builtin/till-go.toml`. (Use `git mv` for history continuity per `feedback_never_remove_workflow_files.md` discipline carried over to source files.)
  - `//go:embed` directive in `embed.go` references `builtin/till-go.toml` (NOT `builtin/default-go.toml`).
  - `LoadDefaultTemplateForLanguage("go")` switch case returns the new path; `BuiltinTemplateNames()` returns `["default-generic", "till-go"]` after this droplet (W5.D2 lands the second rename).
  - `git grep "default-go.toml"` returns zero hits in **non-doc-comment locations** (string literals, embed directives, switch cases, `BuiltinTemplateNames` literal entries) within `cmd/`, `internal/`, top-level Go files. Doc-comment / workflow-MD / historical-rename-record hits ARE retained per `feedback_never_remove_workflow_files.md` and the rename-history rule (ROUND-2 HF5 — phrasing refined to make this grep-able + contradiction-free with bullet on retained historical refs below).
  - Caller-audit edits in 5 sites (ROUND-2 HF6 regenerated audit list — `git grep "default-go.toml" cmd/ internal/` against current HEAD):
    - `internal/app/service.go:383` — doc-comment forward-looking → update to `till-go.toml`.
    - `internal/app/service_test.go` — 7 hits: line 6534 is a load-bearing `filepath.Join("..", "templates", "builtin", "default-go.toml")` literal (will break test at run if not renamed); other 6 hits are doc-comments — line 6525, 6529, 6537 (error message string), 6551, 6552, 6713 forward-looking → update.
    - `internal/app/auto_generate_steward_test.go:18` — doc-comment forward-looking → update to `till-go.toml`.
    - `internal/adapters/server/common/mcp_surface.go:906` — doc-comment forward-looking → update.
    - `internal/adapters/server/mcpapi/extended_tools.go:1867` — doc-comment forward-looking → update.
    - `internal/templates/load.go` (3 hits in doc-comments at line 255, 592, 735) + `internal/templates/load_test.go` (2 hits at 1709, 1927) + `internal/templates/embed.go` (5 hits including the `//go:embed` directive at :34, switch case at :138, plus historical doc-comments at :17, :62, :106) + `internal/templates/embed_test.go` (multiple) — these are all touched by the rename itself (embed.go directive + switch); doc-comments classified as historical (Drop 4c.5 F.2.1 rebadge history) are RETAINED verbatim per `feedback_never_remove_workflow_files.md` extension; doc-comments forward-looking (e.g. `embed.go:106`'s "→ loads `builtin/default-go.toml`") are UPDATED.
    - `internal/templates/builtin/default-generic.toml` doc-comments referencing `default-go.toml` (lines 3, 7, 35, 40, 253, 261, 273, 312) — historical / cross-reference; retained verbatim until W5.D2's renaming pass updates them in lockstep with the file's own rename.
    - **Removed from caller-audit list (ROUND-2 HF6)**: `internal/app/auto_generate_steward.go` (zero `default-go.toml` refs verified via `git grep`), `internal/app/template_service.go` (zero refs verified). Both were over-claimed in Round-1 PLAN.md.
  - `mage ci` green; integration test exercises template loading via new name.
- **Blocked by:** 4c.6.W1.D1 (ROUND-2 HF1 — both edit `internal/templates/embed.go` + `embed_test.go` AND share `internal/templates` package compile/test unit; W1.D1 lands the embed-list extension first because W5.D1 is a rename within the existing list shape; W5.D1 then rebases its rename onto the post-W1.D1 directive. Transitively chains to 4c.6.W0.5 via W1.D1's blocker).
- **Specify (droplet-scope; inherits `SKETCH.md` § 26.W5):**
  - **Objective:** Rename the Go-flavored builtin to align with the `till-` prefix family per `SKETCH.md` § 3.5.1 / § 21.6 — communicates "shipped from Tillsyn binary."
  - **AcceptanceCriteria:** see Acceptance bullets above.
  - **ValidationPlan:** `git grep "default-go"` post-edit; `mage ci` green; `mage test-pkg ./internal/templates`; `mage test-pkg ./internal/app`; `mage test-pkg ./internal/adapters/server/mcpapi`.
  - **RiskNotes:** 140 `workflow/` MD references to old name — leave alone per `feedback_never_remove_workflow_files.md`. Test fixtures may hardcode template names — audit during rename. `BuiltinTemplateNames` wire-protocol breakage for any external consumer hardcoding old names — explicitly out of scope (no external consumers pre-MVP). Drop 4c.5 F.2.1 doc-comment in `embed.go` LINES 16-23 references "rebadged from `default.toml` to `default-go.toml`" — update those to record the second rebadge "to `till-go.toml`" per dual-history note.
  - **ContextBlocks:**
    - `decision` (normal): `till-` prefix per `SKETCH.md` § 3.5.1 / § 21.6.
    - `constraint` (normal): no top-level Go-source references to old names post-rename.
    - `reference` (normal): `feedback_never_remove_workflow_files.md` — workflow MD references retained.
    - `reference` (normal): `internal/templates/embed.go` lines 16-23 + 34 + 136-138 + 178 — load-bearing sites per CASCADE_ENFORCEMENT research §C.
  - **KindPayload:** `{"changes":[{"file":"internal/templates/builtin/default-go.toml","symbol":"file","action":"rename_to:internal/templates/builtin/till-go.toml","shape_hint":"git mv preserves history"},{"file":"internal/templates/embed.go","symbol":"DefaultTemplateFS","action":"modify","shape_hint":"//go:embed builtin/default-go.toml → builtin/till-go.toml; switch case 'go' → builtin/till-go.toml; BuiltinTemplateNames literal default-go → till-go"},{"file":"internal/templates/embed_test.go","symbol":"FS introspection tests","action":"modify","shape_hint":"path-string updates only"},{"file":"<4 caller audit sites>","symbol":"string literal references","action":"modify","shape_hint":"forward-looking comments → till-go; historical droplet refs → retained verbatim"}]}`

##### 4c.6.W5.D2 — Rename `default-generic.toml` → `till-gen.toml` (file move + embed.go + caller audit)

- **State:** done
- **Kind:** `build` (atomic droplet; `Irreducible: true`)
- **Paths:** `internal/templates/builtin/till-gen.toml` (RENAMED from `default-generic.toml`), `internal/templates/embed.go`, `internal/templates/embed_test.go`, plus caller-audit edits regenerated via ROUND-2 HF6 `git grep "default-generic.toml" cmd/ internal/`:
  - `internal/app/service.go:384` — doc-comment forward-looking → update to `till-gen.toml`.
  - `internal/app/service_test.go:6849` — doc-comment forward-looking → update.
  - `internal/app/auto_generate_steward_test.go:18` — same line as W5.D1's audit (single doc-comment mentions both `default-go.toml` and `default-generic.toml`); W5.D1 lands the `default-go.toml` half first, W5.D2 follows up with the `default-generic.toml` half — strict serial via `Blocked by` chain.
  - `internal/adapters/server/common/mcp_surface.go:908` — doc-comment forward-looking → update.
  - `internal/templates/load.go` (lines 255, 592) + `internal/templates/embed.go` (lines 20, 34, 59, 66, 103, 136) + `internal/templates/embed_test.go` (multiple) + `internal/templates/builtin/default-go.toml` (line 6) — touched by the rename itself (embed.go directive + switch case at :136); doc-comments classified as historical retained verbatim, forward-looking updated.
  - **Removed from caller-audit list (ROUND-2 HF6)**: `internal/app/template_service.go` (zero refs to `default-generic.toml` verified via `git grep`).
- **Packages:** `internal/templates`, `internal/app` (caller audit), `internal/adapters/server/common` (caller audit — `mcp_surface.go`; ROUND-2 HF6).
- **Acceptance:**
  - `internal/templates/builtin/default-generic.toml` renamed to `internal/templates/builtin/till-gen.toml` via `git mv`.
  - `//go:embed` directive in `embed.go` references `builtin/till-gen.toml`.
  - `LoadDefaultTemplateForLanguage("")` switch case returns the new path; `BuiltinTemplateNames()` returns `["till-gen", "till-go"]` after this droplet (sketch § 3.5.1 stable lexical order maintained).
  - `git grep "default-generic.toml"` returns zero hits in **non-doc-comment locations** (string literals, embed directives, switch cases, `BuiltinTemplateNames` literal entries) within `cmd/`, `internal/`, top-level Go files. Doc-comment / workflow-MD / historical-rename-record hits ARE retained per `feedback_never_remove_workflow_files.md` and the rename-history rule (ROUND-2 HF5 mirror — phrasing analogous to W5.D1's bullet).
  - `mage ci` green.
- **Blocked by:** 4c.6.W5.D1 (both droplets edit `internal/templates/embed.go` + `embed_test.go` — package-lock chain).
- **Specify (droplet-scope; inherits `SKETCH.md` § 26.W5):**
  - **Objective:** Mirror W5.D1's rename for the language-agnostic builtin, completing the `till-` prefix family.
  - **AcceptanceCriteria:** see Acceptance bullets above.
  - **ValidationPlan:** `git grep "default-generic"`; `mage ci`; `mage test-pkg ./internal/templates`.
  - **RiskNotes:** Same caller-audit discipline as W5.D1. Workflow MD references to `default-generic.toml` retained per `feedback_never_remove_workflow_files.md`. `BuiltinTemplateNames()` ordering: stable lexical (`till-gen` < `till-go` so order is preserved post-rename).
  - **ContextBlocks:**
    - `decision` (normal): `till-` prefix per `SKETCH.md` § 3.5.1.
    - `constraint` (normal): `BuiltinTemplateNames()` stable lexical order.
    - `reference` (normal): Drop 4c.5 F.2.2 + F.1.3 droplet doc-comments record the prior rename.
  - **KindPayload:** `{"changes":[{"file":"internal/templates/builtin/default-generic.toml","symbol":"file","action":"rename_to:internal/templates/builtin/till-gen.toml","shape_hint":"git mv"},{"file":"internal/templates/embed.go","symbol":"DefaultTemplateFS","action":"modify","shape_hint":"//go:embed list update; switch case '' → till-gen.toml; BuiltinTemplateNames literal default-generic → till-gen"},{"file":"<caller audit sites>","symbol":"forward-looking string literal references","action":"modify","shape_hint":"forward-looking → till-gen; historical → retained"}]}`

##### 4c.6.W5.D3 — Drop `go-` prefix from agent_name in renamed `till-go.toml` + remove `tools` from frontmatter

- **State:** done
- **Kind:** `build` (atomic droplet; `Irreducible: true`)
- **Paths:** `internal/templates/builtin/till-go.toml` (NOTE: post-W5.D1 rename), `internal/templates/builtin/till-gen.toml` (post-W5.D2 rename), plus the placeholder agent .md files shipped in W1 — strip `tools:` / `model:` from their frontmatter so agents.toml is sole authority per `SKETCH.md` § 4.4 + § 15. **W5-D2-FF1 ROUND-2 ABSORPTION:** also `internal/templates/load.go` (lines 388 + 1240, paired historical doc-comments referencing both `default-go.toml + default-generic.toml`) and `internal/app/auto_generate_steward.go` (line 108, short-name historical doc-comment `default-generic vs default-go`) — these doc-comment-only sites were deferred from W5.D1 + W5.D2 routed Unknowns and absorbed here so the rebadge cleanup closes in this droplet.
- **Packages:** `internal/templates`.
- **Acceptance:**
  - In `till-go.toml`, every `[agent_bindings.<kind>] agent_name = "go-<name>"` becomes `agent_name = "<name>"` (per `SKETCH.md` § 7) — Go specialization comes from group choice at init time, not from agent name.
  - In `till-gen.toml`, `agent_name` values follow the same convention (no group prefix).
  - In every `internal/templates/builtin/agents/<group>/*.md` placeholder file shipped by W1, frontmatter is `name` + `description` ONLY — no `model:`, no `tools:`, no `allowedTools:`, no `disallowedTools:` per `SKETCH.md` § 15.
  - Existing `tools` field on `agent_bindings` at runtime: per `SKETCH.md` § 4.1 + § 4.2, `tools_allow` + `tools_deny` MOVE from agent .md frontmatter / `[agent_bindings]` to `agents.toml`. After this droplet, `[agent_bindings.<kind>]` blocks have ONLY cascade-structural fields (`agent_name`, `commit_agent`, `[…context]`, etc.); runtime fields (`model`, `tools`, etc.) are absent. NOTE: any actual schema-level changes to `templates.AgentBinding` to drop fields are OUT OF SCOPE for this droplet — those would orphan tests + downstream consumers; the field-removal-from-schema is a Drop 4c.7+ concern. THIS droplet only ensures the SHIPPED `till-*.toml` files don't SET those fields, even though the schema still accepts them.
  - `mage test-pkg ./internal/templates` passes; `mage ci` green; integration test exercising template loading via `till-go.toml` confirms no `tools` field is read from `[agent_bindings]`.
  - **W5-D2-FF1 ROUND-2 ABSORBED:** `internal/templates/load.go:388` + `:1240` paired historical doc-comments updated to read `till-gen.toml + till-go.toml` (current names, with optional `← default-generic.toml + default-go.toml` rebadge note matching the dual-history pattern from W5.D1/W5.D2). `internal/app/auto_generate_steward.go:108` short-name doc-comment updated to `till-gen vs till-go` (with optional rebadge note). After this droplet, `git grep "default-generic"` returns ZERO non-historical hits and `git grep "default-go"` returns ZERO non-historical hits across `cmd/`, `internal/`, `*.go` — rebadge cleanup is complete.
- **Blocked by:** 4c.6.W5.D1, 4c.6.W5.D2, 4c.6.W1.D1 (this droplet edits the renamed files from D1+D2 AND the placeholder agent .md files from W1).
- **Specify (droplet-scope; inherits `SKETCH.md` § 26.W5 + § 4.4 + § 15):**
  - **Objective:** Migrate runtime fields (`model`, `tools_allow/deny`) out of template-defined surfaces (`[agent_bindings]` and agent .md frontmatter) and into `agents.toml`'s sole authority. Drops `go-` prefix from `agent_name` so group choice (not name) carries language specialization.
  - **AcceptanceCriteria:** see Acceptance bullets above.
  - **ValidationPlan:** `mage test-pkg ./internal/templates`; integration test exercising template-load against post-rename files; `mage ci`.
  - **RiskNotes:** Schema-level field removal from `templates.AgentBinding` is OUT OF SCOPE — would break tests + adapter contracts; **deferred to Drop 4c.7** (ROUND-2 OQ#5 disposition pinned the deferral target — was previously phrased as "Drop 4c.7 or later," now hard-pinned to 4c.7). This droplet ONLY edits the shipped `till-*.toml` files + placeholder agent .md frontmatter. Frontmatter strip (W3.D3) handles RUN-TIME removal of `model:` / `tools:` if any user-supplied agent .md in `<project>/.tillsyn/agents/` carries them.
  - **ContextBlocks:**
    - `decision` (normal): runtime fields move to `agents.toml`; templates carry only cascade-structural fields per `SKETCH.md` § 7.
    - `constraint` (high): schema-level field removal deferred — this droplet edits SHIPPED files only.
    - `warning` (high): existing tests that load `till-go.toml` and assert `[agent_bindings.<kind>] tools = [...]` MUST be updated to assert tools is empty/absent post-this-droplet.
    - `reference` (normal): `SKETCH.md` § 4.4 frontmatter strip happens at render time (W3.D3); this droplet handles SOURCE files only.
  - **KindPayload:** `{"changes":[{"file":"internal/templates/builtin/till-go.toml","symbol":"[agent_bindings.<kind>] agent_name + tools","action":"modify","shape_hint":"drop go- prefix from agent_name; remove tools field; remove model field"},{"file":"internal/templates/builtin/till-gen.toml","symbol":"[agent_bindings.<kind>] agent_name + tools","action":"modify","shape_hint":"same as till-go"},{"file":"internal/templates/builtin/agents/till-{gen,go,gdd}/*.md","symbol":"placeholder frontmatter","action":"modify","shape_hint":"frontmatter is name+description ONLY; no model/tools/allowedTools/disallowedTools"}]}`

#### Wave W6 — Methodology + config docs

##### 4c.6.W6.D1 — `AGENTS_CONFIG.md` (new top-level doc)

- **State:** done
- **Kind:** `build` (atomic droplet; doc-only — `Irreducible: true`)
- **Paths:** `AGENTS_CONFIG.md` (NEW top-level), `README.md` (single pointer link added — touches a different droplet, see W6.D5).
- **Packages:** none (markdown only).
- **Acceptance:**
  - `AGENTS_CONFIG.md` shipped at repo root, ≥ 200 lines, sections: schema (§ 4 of sketch), override semantics (§ 5), `env_set` vs `env_from_shell` (§ 4.5), `tools_allow` vs `tools_deny` override scope (§ 4.3.1), frontmatter strip behavior (§ 4.4), `claude_md_addons` (§ 12), worked examples (Bedrock / Vertex / OpenRouter / Ollama Cloud — borrow from `SKETCH.md` § 6 indirectly).
  - Cross-references to `CASCADE_METHODOLOGY.md` skeleton (W6.D2) + `SPAWN_PIPELINE.md` (existing) + `CLI_ADAPTER_AUTHORING.md` (existing).
  - `mage ci` green (no Go code changed; doc lint if any catches it).
- **Blocked by:** 4c.6.W0 (sketches the schema + override semantics that this doc describes — must land before docs accurately describe them).
- **Specify (droplet-scope; inherits `SKETCH.md` § 26.W6):**
  - **Objective:** Adopter-facing reference for `agents.toml` schema, override semantics, env-handling, frontmatter-strip behavior, `claude_md_addons` injection. Single source for "how do I configure my agents per-machine."
  - **AcceptanceCriteria:** see Acceptance bullets above.
  - **ValidationPlan:** doc review pass; `mage ci`.
  - **RiskNotes:** Doc may drift from W0's actual implementation. Mitigation: doc cites symbol names + file paths from `internal/config/agents.go` so a future refactor's renames surface as broken citations.
  - **ContextBlocks:**
    - `constraint` (high): doc accuracy gates dogfood — adopters must be able to follow `AGENTS_CONFIG.md` and get a working setup.
    - `reference` (normal): `SKETCH.md` § 4-6 carries the canonical schema + examples.
  - **KindPayload:** `{"changes":[{"file":"AGENTS_CONFIG.md","symbol":"new top-level doc","action":"add","shape_hint":"sections per sketch §4-6 + §12; ≥200 lines; cross-refs to CASCADE_METHODOLOGY.md and SPAWN_PIPELINE.md"}]}`

##### 4c.6.W6.D2 — `CASCADE_METHODOLOGY.md` skeleton

- **State:** done
- **Kind:** `build` (atomic droplet; doc-only — `Irreducible: true`)
- **Paths:** `CASCADE_METHODOLOGY.md` (NEW top-level skeleton).
- **Packages:** none.
- **Acceptance:**
  - `CASCADE_METHODOLOGY.md` shipped at repo root with skeleton structure: leading "Plan Down, Build Up" section per `feedback_plan_down_build_up.md`; followed by sections covering kind closed-12-enum, role enum, `metadata.structural_type` (drop / segment / confluence / droplet), agent shape, Section 0 5-pass / 4-pass certificate, Tillsyn-flavored Specify pass, TN-per-section response style, Hylla-first evidence ordering, TDD requirement, QA proof-vs-falsification asymmetry, `blocked_by` ordering, parent-children-complete invariant, isolation enforcement.
  - Each section is a placeholder with 1-3 paragraphs and a `<!-- TODO populate post-dogfood with measured benchmarks -->` marker per `SKETCH.md` § 14.1.
  - The first `##` H2 section after the H1 title is `## Plan Down, Build Up` per `SKETCH.md` § 26.W6 ContextBlocks `constraint`(high) — testable via doc inspection / `awk`-or-equivalent first-H2 check (ROUND-2 HF9 — pinned to grep-able assertion to replace the un-testable "leads the doc" phrasing).
  - Cross-references to `AGENTS_CONFIG.md` (W6.D1) + `GDD_METHODOLOGY.md` (W6.D3).
- **Blocked by:** —
- **Specify (droplet-scope; inherits `SKETCH.md` § 26.W6 + § 14.1):**
  - **Objective:** Skeleton for the cascade-methodology canonical doc — fleshed out post-dogfood with measured benchmarks per `project_methodology_docs_tracker.md`. MVP-release blocker.
  - **AcceptanceCriteria:** see Acceptance bullets above.
  - **ValidationPlan:** doc review pass; `mage ci`.
  - **RiskNotes:** Skeleton must be complete enough for future methodology-comparison articles to cite; depth not required pre-dogfood, but ALL cascade vocabulary (kind / role / structural_type) MUST be cross-referenced to `WIKI.md § Cascade Vocabulary` rather than redefined here.
  - **ContextBlocks:**
    - `constraint` (high): "Plan Down, Build Up" leads the doc per `feedback_plan_down_build_up.md`.
    - `decision` (normal): three docs (this + `GDD_METHODOLOGY.md` + `AGENTS_CONFIG.md`) mandatory before MVP per `project_methodology_docs_tracker.md`.
    - `reference` (normal): `WIKI.md § Cascade Vocabulary` is canonical for vocabulary; this doc cross-references rather than duplicates.
  - **KindPayload:** `{"changes":[{"file":"CASCADE_METHODOLOGY.md","symbol":"new top-level doc skeleton","action":"add","shape_hint":"leads with Plan Down Build Up; placeholder sections per sketch §26.W6 acceptance"}]}`

##### 4c.6.W6.D3 — `GDD_METHODOLOGY.md` placeholder

- **State:** done
- **Kind:** `build` (atomic droplet; doc-only — `Irreducible: true`)
- **Paths:** `GDD_METHODOLOGY.md` (NEW top-level placeholder).
- **Packages:** none.
- **Acceptance:**
  - `GDD_METHODOLOGY.md` shipped at repo root as a stub: title, 1-paragraph description ("Graph-Driven Development methodology — populated post-Hylla-rev / post-dogfood per `project_methodology_docs_tracker.md` and `SKETCH.md` § 14.2"), `<!-- TODO populate post-dogfood -->` marker, prior-art research note placeholder per `SKETCH.md` § 14.2.1.
- **Blocked by:** —
- **Specify (droplet-scope; inherits `SKETCH.md` § 26.W6 + § 14.2):**
  - **Objective:** Placeholder for the GDD-methodology doc. MVP-release blocker per `project_methodology_docs_tracker.md` — actual content lands post-Hylla-rev.
  - **AcceptanceCriteria:** see Acceptance bullets above.
  - **ValidationPlan:** doc review pass; `mage ci`.
  - **RiskNotes:** Placeholder MUST clearly mark itself as "populate post-dogfood" so adopters don't expect substantive content here.
  - **ContextBlocks:**
    - `decision` (normal): placeholder only; substantive content post-dogfood per `project_methodology_docs_tracker.md`.
    - `reference` (normal): `SKETCH.md` § 14.2.1 prior-art research note still applies.
  - **KindPayload:** `{"changes":[{"file":"GDD_METHODOLOGY.md","symbol":"placeholder doc","action":"add","shape_hint":"~30-line stub; populate post-Hylla-rev"}]}`

##### 4c.6.W6.D4 — Update `SPAWN_PIPELINE.md` + `CLI_ADAPTER_AUTHORING.md` for `--bare`-collapsed isolation

- **State:** done
- **Kind:** `build` (atomic droplet; doc-only — `Irreducible: true`)
- **Paths:** `SPAWN_PIPELINE.md` (existing), `CLI_ADAPTER_AUTHORING.md` (existing).
- **Packages:** none.
- **Acceptance:**
  - `SPAWN_PIPELINE.md:24-31` — the prior "two paths" framing — rewritten to reflect `--bare`-collapsed isolation per `RESEARCH/ISOLATION_ENFORCEMENT_FIX.md` §D.5. The corrected framing: Tillsyn's `--bare --plugin-dir <bundle>/plugin --agent <name> --setting-sources "" --strict-mcp-config --settings ... --mcp-config ...` argv enforces isolation per Anthropic's documented `--bare` behavior; Path B / system CLAUDE.md / skills / project CLAUDE.md / hooks / `~/.claude/settings.json` / system plugins are ALL skipped by Claude Code. The actual gap that Drop 4c.6 W3 closes is that the bundle's `<bundle>/plugin/agents/<name>.md` USED to ship a one-liner stub instead of substantive content; W3 closes that hole.
  - `CLI_ADAPTER_AUTHORING.md` — note appended documenting the same `--bare`-collapsed-isolation correction so future CLI adapters (e.g. Drop 4d's codex adapter) inherit the correct mental model.
- **Blocked by:** 4c.6.W3 (this droplet documents what W3 actually shipped — must come AFTER W3 lands so the doc reflects reality).
- **Specify (droplet-scope; inherits `SKETCH.md` § 26.W3 § D.5 + § 26.W6):**
  - **Objective:** Correct existing `SPAWN_PIPELINE.md` + `CLI_ADAPTER_AUTHORING.md` doc-comments that misled the prior research's "two-paths model" framing.
  - **AcceptanceCriteria:** see Acceptance bullets above.
  - **ValidationPlan:** doc review pass; `mage ci`; `git grep "two paths"` returns zero hits in spawn / adapter docs post-edit.
  - **RiskNotes:** The `--bare`-collapsed framing depends on Claude Code's documented `--bare` behavior staying stable across versions. Mitigation: the corrected text cites Claude Code docs (Context7 reference) rather than relying on undocumented behavior.
  - **ContextBlocks:**
    - `constraint` (high): doc must reflect the post-W3 reality, not the pre-W3 stub-shape architecture.
    - `reference` (normal): `RESEARCH/ISOLATION_ENFORCEMENT_FIX.md` §D.5 is the source-of-truth for the corrected framing.
  - **KindPayload:** `{"changes":[{"file":"SPAWN_PIPELINE.md","symbol":"lines 24-31 (two-paths block)","action":"modify","shape_hint":"rewrite to --bare-collapsed isolation framing per ISOLATION_ENFORCEMENT_FIX §D.5"},{"file":"CLI_ADAPTER_AUTHORING.md","symbol":"isolation-discipline note","action":"modify","shape_hint":"append note documenting --bare-collapsed inheritance for future adapters"}]}`

##### 4c.6.W6.D5 — `README.md` pointer additions to new docs

- **State:** done
- **Kind:** `build` (atomic droplet; doc-only — `Irreducible: true`)
- **Paths:** `README.md` (existing — pointer additions only).
- **Packages:** none.
- **Acceptance:**
  - `README.md` adds three short bullets (or a "Methodology Docs" section) pointing to `AGENTS_CONFIG.md`, `CASCADE_METHODOLOGY.md`, `GDD_METHODOLOGY.md`. No restructuring of existing README content.
  - Bullet text mentions each doc's purpose in 1 line; cross-referenced to its top-level path.
- **Blocked by:** 4c.6.W6.D1, 4c.6.W6.D2, 4c.6.W6.D3 (READMEs cite docs that must exist).
- **Specify (droplet-scope; inherits `SKETCH.md` § 26.W6):**
  - **Objective:** Make new methodology + config docs discoverable from the README so new adopters reach them in their first read.
  - **AcceptanceCriteria:** see Acceptance bullets above.
  - **ValidationPlan:** doc review pass; `mage ci`.
  - **RiskNotes:** README pointer hygiene must be idempotent — re-running this droplet should not double-add the pointers. Mitigation: builder uses Read+Edit (not Write) and verifies the bullets don't already exist before adding.
  - **ContextBlocks:**
    - `decision` (normal): pointer only; do NOT inline-document the new docs in README (keeps README terse).
    - `reference` (normal): `project_methodology_docs_tracker.md` lists three docs as MVP-release blockers.
  - **KindPayload:** `{"changes":[{"file":"README.md","symbol":"new methodology-docs section or bullet additions","action":"modify","shape_hint":"3 short pointers to AGENTS_CONFIG.md / CASCADE_METHODOLOGY.md / GDD_METHODOLOGY.md"}]}`

### `blocked_by` graph summary

The graph below mirrors the `Blocked by:` bullets above and the `_BLOCKERS.toml` ledger. `→` = "blocked by."

```
4c.6.W0       (no blockers — Wave A head)
4c.6.W0.5     (no blockers — Wave A head, parallel with W0)
4c.6.W6.D2    (CASCADE_METHODOLOGY.md skeleton — no blockers — Wave A)
4c.6.W6.D3    (GDD_METHODOLOGY.md placeholder — no blockers — Wave A)

4c.6.W1.D1    → 4c.6.W0.5                         (ROUND-2 HF1+HF2 — internal/templates package serialization head)
4c.6.W2       → 4c.6.W1.D1                        (W2 copies W1's embedded files; transitively waits on W0.5)
4c.6.W3       → 4c.6.W1.D1, 4c.6.W0               (W3 resolver consumes W1's files; W3 frontmatter strip uses W0's helper)
4c.6.W5.D1    → 4c.6.W1.D1                        (ROUND-2 HF1+HF2 — both edit embed.go + embed_test.go AND share internal/templates package; transitively chains via W1.D1 → W0.5)
4c.6.W5.D2    → 4c.6.W5.D1                        (both edit internal/templates/embed.go; transitively → W1.D1 → W0.5 via HF7-derivative — HF7 auto-resolved by HF2's chain)
4c.6.W5.D3    → 4c.6.W5.D1, 4c.6.W5.D2, 4c.6.W1.D1 (edits the renamed files + W1's placeholders; transitively all internal/templates work serialized)
4c.6.W6.D1    → 4c.6.W0                           (AGENTS_CONFIG.md describes W0's schema)
4c.6.W6.D4    → 4c.6.W3                           (doc reflects W3-shipped reality; W6.D4 is sole owner of SPAWN_PIPELINE.md:24-31 + CLI_ADAPTER_AUTHORING.md per ROUND-2 HF3)
4c.6.W6.D5    → 4c.6.W6.D1, 4c.6.W6.D2, 4c.6.W6.D3 (README cites docs that must exist)
```

Wave A (parallel — no blockers): 4c.6.W0, 4c.6.W0.5, 4c.6.W6.D2, 4c.6.W6.D3.
Wave B (after Wave A): 4c.6.W1.D1 (after W0.5), 4c.6.W6.D1 (after W0).
Wave C (after Wave B): 4c.6.W2 (after W1.D1), 4c.6.W3 (after W1.D1 + W0), 4c.6.W5.D1 (after W1.D1).
Wave D (after Wave C): 4c.6.W5.D2 (after W5.D1), 4c.6.W6.D4 (after W3).
Wave E (after Wave D): 4c.6.W5.D3 (after W5.D1 + W5.D2 + W1.D1), 4c.6.W6.D5 (after W6.D1 + W6.D2 + W6.D3 — note W6.D5 only requires W6.D1 which lives in Wave B; it is launchable as soon as Wave B closes, but holding it here in Wave E for graphical clarity since W6.D1 is its sole-Wave-B parent and the others are Wave A).

ROUND-2 graph implications: the `internal/templates` package now serializes through a single chain W0.5 → W1.D1 → W5.D1 → W5.D2 → W5.D3 (HF1 + HF2 enforced; HF7 transitively resolved). This adds depth (5 sequential steps along the longest internal/templates chain) but eliminates the package-compile race that Round 1 plan-QA-falsification flagged at 1.1 + 1.2 + 1.3. The wall-clock bottleneck is now the longer of (a) W2 sub-plan after W1.D1 closes, (b) W3 sub-plan after W1.D1 + W0 close, or (c) the W5.D1 → W5.D2 → W5.D3 chain. W0 + W0.5 still run in parallel at drop start.

## Notes

### Sub-plan vs direct-droplet ratio + L2 spawn cadence

L1 emits 4 sub-plan containers (W0, W0.5, W2, W3) and 9 direct droplets (W1.D1, W5.D1-D3, W6.D1-D5). The orchestrator spawns the L2 sub-planners against each sub-plan when its `blocked_by` clears — W0 + W0.5 spawn immediately at drop start (no blockers); W2 spawns after W1.D1 closes; W3 spawns after both W1.D1 and W0 close. L2 sub-planners author their own `workflow/drop_4c_6/DROP_4c.6.W<X>_<NAME>/PLAN.md` per WORKFLOW.md § Sub-Drops; L2 plan-QA fires per WORKFLOW.md Phase 2 against each sub-plan independently.

### Drop-4c.5 sequencing

Drop 4c.5 is in `building` state per `workflow/drop_4c_5/PLAN.md` line 3. This drop (4c.6) cannot start build-phase work until 4c.5 closes (its `Blocked by:` line above is updated when the orch transitions 4c.6 from `planning` → `building`). The plan-phase work in this PLAN.md is independent of 4c.5's close and may be QA'd / discussed / refined while 4c.5 is still building.

### Out-of-scope items routed away

- **W7 / W8 / W9 / W10 / W11** — Drop 4c.7 (cascade wiring; NOT this drop). Sketch §25.2.
- **W4-A / W4-B / W4-C / W4-D** — Drop 4c.8 (substantive prompt content + dogfood overrides + end-to-end isolation tests). Sketch §25.3.
- **Schema-level removal of `tools` / `model` fields from `templates.AgentBinding`** — **Drop 4c.7** (ROUND-2 OQ#5 disposition — hard-pinned to 4c.7, was previously "4c.7+"). W5.D3 only edits the SHIPPED template files; the schema still accepts the fields.
- **`till-gdd` substantive content** — post-Hylla-rev, post-dogfood. W1.D1 ships placeholder only.
- **Sentinel-injection integration test suite (full)** — Drop 4c.8 W4-D. W3 includes only a minimal "bundle body is non-empty" assertion.
- **Per-spawn rate limits / concurrency caps / provider-side health checks** — pre-MVP out of scope per `SKETCH.md` § 27.
- **`hylla-shared` repo extraction** — post-MVP migration of vendored `fsatomic` + `configmerge` per `SKETCH.md` § 9.6.

### Sketch-vs-handoff scope notes

The orchestrator handoff inferred Drop 4c.6's scope as "W0 + W0.5 + W1 + W2 + W3 + W5 + W6 + possibly W9" with W9 in question. Sketch §25.1 EXCLUDES W9 from 4c.6 (W9 is "subsumed into 4c.6 W5 (rename happens there alongside template thinning); reserved for any wiring follow-ups discovered during W7/W8" — i.e., it's a Drop 4c.7 reservation wave, not a 4c.6 wave). I trust the sketch and excluded W9 from 4c.6's L1 droplets.

### Pre-MVP rules carried over from Drop 4c.5

Planner + builder run `model: sonnet`; QA pair runs `model: opus` (per system frontmatter at `~/.claude/agents/go-*.md`; agents.toml runtime config — Drop 4c.6 W1 schema + Drop 4c.8 W4 population — makes this declarative going forward). Filesystem-MD mode, no Tillsyn-runtime per-droplet plan items, no closeout MD rollups, single-line conventional commits ≤72 chars, never raw `go test` / `go build` / `go vet` / `mage install`. Builder spawn prompts MUST include "do NOT commit" directive — orch commits after each droplet closes per WORKFLOW.md Phase 4. Each builder reads any sub-plan-level `REVISIONS POST-AUTHORING` section first if the L2 plan adds one. Section 0 SEMI-FORMAL REASONING in every subagent response, but Section 0 stays in the orchestrator-facing response — never in `PLAN.md` / `BUILDER_WORKLOG.md` / `BUILDER_QA_*.md` / `PLAN_QA_*.md` / `CLOSEOUT.md`.

### Locked architectural decisions (inherited from Drop 4c)

L1 (no secrets), L2 (no command override), L3 (POSIX-only), L4 (closed env baseline), L11 (CLI-agnostic monitor), L13 (context aggregator OPTIONAL), L20 (commit + push gates default OFF). All non-negotiable.

### Builder spawn-prompt template

Each builder spawn for a Drop 4c.6 droplet uses the WORKFLOW.md § "Agent Spawn Contract" preamble plus the per-role appendix in WORKFLOW.md § "Per-Role Spawn Appendices" § Builder. Concretely the appendix carries: droplet ID (e.g. `4c.6.W1.D1`), the droplet row excerpt from THIS PLAN.md (or the L2 sub-plan's PLAN.md for sub-plan droplets), the drop's `BUILDER_WORKLOG.md` path (`workflow/drop_4c_6/BUILDER_WORKLOG.md` for L1 droplets; nested sub-drop path for L2 droplets), round number, and the working-dir absolute path (`/Users/evanschultz/Documents/Code/hylla/tillsyn/main`).

### Open questions — Phase 3 dispositions

All Round-1 open questions resolved during Phase 3 dev discussion (2026-05-09). Recorded inline below for the audit trail.

1. **W0.5 claim-vs-impl validator's known-wired set in 4c.6 (empty) vs 4c.7 (populated by W7+W8).** **RESOLVED 2026-05-09 — empty-set-as-authored.** The validator + sentinel-style "unknown-consumer claim" test SHIP in this drop with the known-wired set empty. No `// TODO Drop 4c.7` no-op stub; the validator runs against the empty set and any template that claims `[[child_rules]]` outputs MUST currently match the (empty) consumer set or fail load. Drop 4c.7 W7 + W8 additions to the set land in their own droplets without re-touching the validator scaffolding.
2. **W2 vendoring vs `hylla-shared` repo migration.** **RESOLVED 2026-05-09 — vendored files get a stronger "DO NOT EDIT — re-vendor from upstream" header comment** in addition to `VENDOR_SOURCE.md`. Header lands at the top of every vendored Go file (a 2-3 line block-comment directive); `VENDOR_SOURCE.md` retains the upstream commit hash + future `hylla-shared` migration plan per `SKETCH.md` § 9.6. L2 W2 sub-planner adds this to its vendoring droplet acceptance.
3. **W2 `till init-dev-config` removal — historical config-file behavior.** **RESOLVED 2026-05-09 — L2 W2 sub-planner MUST verify `till install` covers the dev-config-creation behavior currently in `cmd/till/main.go:2039 runInitDevConfig` (creates dev config file + enforces debug logging) BEFORE finalizing `init-dev-config` removal. If `till install` does NOT already cover this behavior, the L2 sub-plan MUST expand W2 scope to extend `till install` accordingly BEFORE removing `init-dev-config`.** Premature removal = behavior regression; this gate must be caught at L2 plan-QA. Routed to: L2 W2 sub-planner directive (extends "L2 sub-planner spawn directive" section in W2 above implicitly via this Notes resolution).
4. **W3.D5 post-render validator stub-shape detection signature.** **RESOLVED 2026-05-09 — deferred to L2 W3 sub-planner.** Lean toward body length threshold + frontmatter presence + stub-key-phrase absence (3-signal approach); L2 settles concrete thresholds. L2 plan-QA attacks the choice.
5. **W5.D3 schema-level field-removal deferral.** **RESOLVED 2026-05-09 — schema-level removal of `tools` / `model` fields from `templates.AgentBinding` is assigned to Drop 4c.7.** The pinning to 4c.7 (rather than "4c.7 or later") gives Drop 4c.7's planner a concrete inheritable deferral. Mirrored below in W5.D3's RiskNotes + the "Out-of-scope items routed away" subsection of these Notes.
6. **W6.D2 `CASCADE_METHODOLOGY.md` skeleton depth.** **RESOLVED 2026-05-09 — keep as-authored.** "1-3 paragraphs per section + `<!-- TODO populate post-dogfood -->` markers" is sufficient depth pre-dogfood per `project_methodology_docs_tracker.md`; substantive depth lands post-measured-benchmarks. No change to W6.D2 acceptance.
