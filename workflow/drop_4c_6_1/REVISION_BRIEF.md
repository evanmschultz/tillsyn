# Drop 4c.6.1 — User Surface + Multi-Group + FE Bootstrap

**Status:** REVISION_BRIEF (pre-planner). Replace with planner-emitted PLAN.md once dev signs off on this brief.
**Authored:** 2026-05-12 (post-dogfood-ramp discovery session)
**Drop kind:** user-surface completion + new FE scaffold + agent set restructure
**Workspace:** `tillsyn/main` (single-repo, no cross-module deps for this drop — see §10)
**Blocked by:** none (Drop 4c.6 closed)
**Blocks:** Drop 4c.7 (cascade wiring needs this drop's user surface to be dogfood-friendly first)

## 1. Why this drop exists

Drop 4c.6 shipped the load-bearing architectural surfaces for `till init`, the 3-tier agent body resolver, the post-render validator, isolation argv, defense-in-depth env, and embedded scaffolding. But the **user surface** for getting a project into a dogfood-ready state was treated as later-drop work that didn't materialize in 4c.6 or in any planned drop.

When the dev tried to dogfood Tillsyn against itself on 2026-05-12, the following gaps surfaced:

- `till init` doesn't populate `RepoPrimaryWorktree` / `RepoBareRoot` / `Language` on the project record. The dispatcher REQUIRES `RepoPrimaryWorktree` and errors `ErrInvalidSpawnInput` without it.
- No CLI exists to update an existing project (no `till project update`).
- No CLI exists to create an action item (`till action_item` has `delete / get / list / move / move_state / reparent / restore / supersede / update` but NO `create`).
- TUI mode hardwires `mcp = false`, so `till init` interactive runs never write `.mcp.json`. Means Claude Code can't auto-load the per-project Tillsyn MCP server.
- `till init` doesn't write `<project>/.tillsyn/template.toml` from binary defaults or user HOME tier.
- No `~/.tillsyn/templates/<group>.toml` middle tier exists in the bake walker — only project + embedded.
- No save-back flow (`till template save`, `till agents save`) to push project customization to HOME.
- 12 embedded agent .md files in till-go group include 5 orphaned `go-*-agent.md` files from Drop 4c.6 W5.D3 collapse — never cleaned up.
- The shipped 2-agent (qa-proof + qa-falsification) model covers 4 cascade kinds (plan-qa-proof, plan-qa-falsification, build-qa-proof, build-qa-falsification). Different work, should be separate prompts.
- agents.toml schema is single-group (`[agents.build]`), no story for multi-group projects.
- `till serve` is malformed; needs delete-and-rebuild.

Without these gaps closed, **Tillsyn can't dogfood itself**. The dispatcher pipeline functions but every step from "I just made a project" to "the dispatcher fires" requires manual SQL hacks, manual file writes, and Claude Code restart juggling.

## 2. Drop scope (numbered for planner consumption)

### 2.1 Bake walker — 3-tier template resolution (HOME tier middle layer)

- Extend `bakeProjectKindCatalog` (`internal/app/service.go:bakeProjectKindCatalog`) to walk:
  1. `<RepoBareRoot>/.tillsyn/template.toml` (project bare-root override)
  2. `<RepoPrimaryWorktree>/.tillsyn/template.toml` (project worktree override)
  3. **NEW**: `~/.tillsyn/templates/<group>.toml` per selected group(s) (user-global override)
  4. Embedded `till-<group>.toml` (binary default)
- For multi-group projects, walk tier 3 for EACH selected group, aggregating bindings + child_rules from each.
- Tier 1 + 2 + 4 already shipped per Drop 4c.6.
- Tier 3 is new. Lives in user's `~/.tillsyn/templates/<group>.toml` (created via `till template save` — see 2.7).

### 2.2 Group-aware agent body resolver

- Existing 3-tier resolver in `internal/app/dispatcher/cli_claude/render/render.go` walks:
  1. `<project>/.tillsyn/agents/<name>.md` (project local, FLAT)
  2. `~/.tillsyn/agents/<group>/<name>.md` (user, subdir-per-group)
  3. Embedded `agents/<group>/<name>.md`
- **Update tier 1**: change FLAT to subdir-per-group: `<project>/.tillsyn/agents/<group>/<name>.md`.
- For multi-group projects, resolver searches the project's selected groups in order (declared at project creation or via `till project update --groups go,fe`).
- Cross-group fallback to `gen` group as last-resort (preserves the W3-FF7 cross-group fallback shipped in Drop 4c.6).

### 2.3 Multi-group `till init`

- CLI flag: `--group <name>` (repeated cobra flag — `till init --group go --group fe`).
- JSON payload shape: `{"name": "...", "groups": ["go", "fe"], "mcp": true}`. The `group` (singular) field is removed; tests + acceptance update.
- Default behavior: if no `--group` passed in TUI, prompt user to pick one OR multiple via multi-select picker. Default group selection: `gen` (language-agnostic).
- For each selected group:
  - Copy embedded `agents/<group>/*.md` to `<project>/.tillsyn/agents/<group>/*.md` (subdir-per-group, NOT flat).
  - Aggregate the group's bindings into `<project>/agents.toml` under `[<group>]` and `[<group>.<kind>]` sections.
  - Aggregate the group's template into `<project>/.tillsyn/template.toml` under `[<group>]` sections.

**FF2 disposition (no migration)**: subdir-per-group is the ONLY supported shape. NO FLAT support, NO migration code. Existing FLAT projects from earlier dogfood-ramp sessions require manual cleanup by the dev (`rm -rf <project>/.tillsyn/agents` then re-run `till init`). Documented as refinement **D7-R6 (manual cleanup of existing FLAT projects)**. Builder MUST NOT write migration code — fail loud if a FLAT layout is detected (with a clear error message pointing to the rm-and-reinit fix).

### 2.4 `till init` writes `<project>/.tillsyn/template.toml`

- Currently NOT written. This drop adds it.
- Source: `~/.tillsyn/templates/<group>.toml` (user override) if exists, else embedded `till-<group>.toml`.
- Aggregated across selected groups.
- Idempotent — re-run skips if file exists.

### 2.5 `till init` populates project record fields

- Currently passes only `name + description` to `CreateProject`.
- This drop: switch to `CreateProjectWithMetadata` with:
  - `RepoPrimaryWorktree = <cwd>` (absolute path)
  - `RepoBareRoot = <bare-root detect or empty>` (use `git rev-parse --git-common-dir` or similar)
  - `Language = <primary group's language>` (e.g., "go" for till-go primary)
  - `HyllaArtifactRef = ""` (user sets later via `till project update`)
  - `Metadata.groups = [...]` (the selected groups, persisted on project record)
- This closes D7-R2.

### 2.6 TUI MCP confirm prompt (closes NIT1 + D7-R4)

- Currently `runInitTUI` (`cmd/till/init_cmd.go:229`) hardwires `MCP = false`.
- This drop: add a bubbletea step after group selection that asks y/n for `.mcp.json` registration.
- **Default = YES** per dev directive.
- JSON mode: respects `mcp` boolean as before; default true if absent.

### 2.7 Template & agent save-flow CLIs

- `till template save --from-project <project-id> --group <group>` — read project's `<project>/.tillsyn/template.toml`'s `[<group>]` block, write to `~/.tillsyn/templates/<group>.toml`. Idempotent (overwrites). Confirms with user before overwrite.
- `till template list` — show user's HOME templates + embedded defaults side-by-side.
- `till template show --group <group> --source {home|embedded}` — print the toml content.
- `till template diff --group <group>` — show diff between HOME tier and embedded default.
- `till template restore --group <group>` — copy embedded default to HOME, overwriting user's HOME tier (with confirmation).
- `till agents save --from-project <project-id> --group <group>` — read project's `<project>/.tillsyn/agents/<group>/*.md` files, write to `~/.tillsyn/agents/<group>/*.md`. Idempotent.
- `till agents list` — show user's HOME agents + embedded defaults.
- `till agents show --group <group> --agent <name> --source {home|embedded}` — print agent body.
- `till agents diff --group <group> --agent <name>` — diff HOME vs embedded.

### 2.8 `till project update` CLI

- Add subcommand under `till project`.
- Flags: `--root-path`, `--bare-root`, `--language`, `--add-group <name>`, `--remove-group <name>`, `--hylla-artifact-ref`, `--description`, `--owner`, `--homepage`, `--icon`, `--color`, `--tag`.
- Project ID required.
- Validates fields per existing Project domain rules.
- Closes D7-R3 + D7-R2 (for projects already created without root_path).

### 2.9 `till action_item create` CLI

- Add subcommand under `till action_item`.
- Required flags: `--project-id`, `--kind`, `--title`, `--description`.
- Optional flags: `--paths <comma-separated>`, `--packages <comma-separated>`, `--files <comma-separated>`, `--blocked-by <id>` (repeated), `--metadata-json`, `--parent-id`, `--structural-type <type>` (override; smart-default per kind).
- **FF4 disposition (smart-default `--structural-type` per kind)**: when `--structural-type` is not passed, derive default from `--kind`:
  - `plan` → `segment`
  - `refinement` → `segment`
  - `research`, `build`, `plan-qa-proof`, `plan-qa-falsification`, `build-qa-proof`, `build-qa-falsification`, `closeout`, `commit`, `discussion`, `human-verify` → `droplet`
  - `--structural-type` flag, when passed, overrides the default. CLI validates the override is a valid enum value (`drop`/`segment`/`confluence`/`droplet`) — fail loud otherwise.
- Returns the created action item's ID (also dotted address for convenience).
- Help text documents the smart-default mapping so users know the default before deciding to override.
- Closes R1 from earlier this session + FF4 from plan-QA round 1.

### 2.10 Project lifecycle CLIs

- `till project delete --project-id <id>` — hard delete (with `--confirm` flag required).
- `till project archive --project-id <id>` — soft archive (sets archived flag).
- `till project restore --project-id <id>` — un-archive.
- `till project rename --project-id <id> --name <new-name>` — rename + reslug.

### 2.11 Agent set restructure (closes D7-R5 and orphan-via-collapse)

- Delete the 5 orphaned `go-*-agent.md` files from `internal/templates/builtin/agents/go/` (Drop 4c.6 W5.D3 carryforward).
- Split the qa-proof + qa-falsification placeholders into 4 separate files per group:
  - `plan-qa-proof-agent.md`
  - `plan-qa-falsification-agent.md`
  - `build-qa-proof-agent.md`
  - `build-qa-falsification-agent.md`
- **FF3 disposition (keep `orchestrator-managed.md`)**: `till-go.toml` lines 599 / 624 / 637 / 650 currently bind `closeout` / `refinement` / `discussion` / `human-verify` to `orchestrator-managed`. Splitting that into role-specific agents is Drop 4c.8 prompt-authoring work, not 4c.6.1's scope. KEEP `orchestrator-managed.md` as a 10th file per group (9 standard + 1 special). Refinement **ORCH-MANAGED-R1** tracks future split into dedicated `closeout-agent` / `refinement-agent` / `discussion-agent` / `human-verify-agent` prompts during Drop 4c.8.
- Final agent set per group (**10 files**: 9 standard + 1 special):
  1. `planning-agent.md`
  2. `builder-agent.md`
  3. `plan-qa-proof-agent.md`
  4. `plan-qa-falsification-agent.md`
  5. `build-qa-proof-agent.md`
  6. `build-qa-falsification-agent.md`
  7. `research-agent.md`
  8. `closeout-agent.md`
  9. `commit-message-agent.md`
  10. `orchestrator-managed.md` (the special 10th — covers closeout/refinement/discussion/human-verify kinds until Drop 4c.8 splits)
- Add the new `fe/` group dir with 10 placeholder agent files (substantive content lands in Drop 4c.8 W4).
- Confirm `gen/` group dir has its 10 placeholders.
- Update `internal/templates/builtin/embed.go` //go:embed list explicitly per file (per Drop 4c.6 F.2.1 falsification mitigation).

### 2.12a `internal/config/agents.go` decoder + `agents.local.toml` deep-merge for new schema (plan-QA proof-FF1)

**Plan-QA proof-FF1 disposition**: the planner missed assigning the Go-side decoder + deep-merge update to any wave. Adding it here as a sibling to §2.12 (schema-shift on the TOML files themselves). The decoder lives at `internal/config/agents.go`; the deep-merge lives there too (per REVISION_BRIEF §2.12 last bullet, originally overlooked).

Scope:
- Update `internal/config/agents.go` `AgentsRegistry` / `Preset` / `Override` struct definitions to support the new `[<group>]` / `[<group>.<kind>]` shape (multi-group at the schema level).
- Update `Resolve(registry, group, kind)` to support a group dimension. Previous signature was `Resolve(registry, kind)` (single-group); new signature accepts group AND kind.
- Update `Merge(localRegistry, projectRegistry)` deep-merge logic over the new shape — per-group + per-kind override resolution.
- Golden-fixture tests for each merge case (group default, group override, project override, local override).
- `mage test-pkg ./internal/config` green.
- Blocked by: must land BEFORE §2.12 (the TOML file rewrites) because the decoder shape is the contract.

### 2.12 agents.toml + template.toml schema shift (multi-group)

- Drop the `agents.` prefix from agents.toml sections (file name already says it).
- New schema for multi-group:
  ```toml
  # Group-level defaults
  [go]
  model = "sonnet"
  tools = ["Read", "Edit", "Grep", "Glob"]
  
  # Per-kind override within group
  [go.plan-qa-proof]
  model = "opus"
  tools = ["Read", "Grep", "Glob"]
  
  [fe]
  model = "sonnet"
  tools = ["Read", "Edit", "Grep", "Glob"]
  
  [fe.build-qa-proof]
  model = "opus"
  tools = ["Read", "Grep", "Glob", "mcp__plugin_playwright_playwright__*"]
  ```
- Same `[<group>]` and `[<group>.<rest>]` shape for `template.toml`.
- Update `internal/templates/builtin/agents.example.toml` to the new schema.
- Update `internal/templates/builtin/till-go.toml` and `till-gen.toml` to the new schema.
- Bake walker (§2.1) reads the new schema.
- agents.local.toml deep-merge logic updates to handle the new schema.

### 2.13 CLAUDE.md cascade-table corrections

- Current cascade table at `CLAUDE.md` references `go-builder-agent`, `go-planning-agent`, etc. Update to drop the `go-` prefix.
- Add 4 separate rows for the QA agents (plan-qa-proof, plan-qa-falsification, build-qa-proof, build-qa-falsification).
- Confirm the 9-agent set per group is documented.

### 2.14 TUI components + vim keybinding dispatcher (inline in tillsyn — see §10)

**Components** at `tillsyn/main/internal/tui/components/`:

- `confirm.go` — y/n prompt (used by NIT1 fix + other CLI confirmations)
- `textinput.go` — single-line text input with validation
- `picker_single.go` — single-select styled list
- `picker_multi.go` — multi-select styled list (used by multi-group selection)
- `header.go` / `footer.go` — styled chrome
- `progress.go` — single-step progress / status line

**Style system** at `tillsyn/main/internal/tui/style/`:

- `palette.go` — colors, semantic names (mirrors stil's color tokens where applicable)
- `spacing.go` — padding / margin constants
- `typography.go` — text styles

**Vim keybinding dispatcher** at `tillsyn/main/internal/tui/keybindings/`:

- `dispatcher.go` — Go-side keybinding dispatcher that consumes `/Users/evanschultz/Documents/Code/hylla/stil/main/src/bindings/baseline.json` AND the Tillsyn-local `<project>/.tillsyn/bindings.json` extensions (per §2.19) at startup, dispatches bubbletea key events to handlers per mode (nav / insert / visual / visual-block / command / hint).
- `loader.go` — JSON-decode the baseline + Tillsyn-local extension files into the dispatcher's binding table.
- `modes.go` — mode-state machine.
- `dispatcher_test.go` — table-tested per-binding dispatch.
- All TUI components route key events through this dispatcher.

**Migration markers**: every file carries `// MIGRATION TARGET: github.com/hylla-org/lykta` doc-comment. Refinement EXTRACT-R1 (components + style + keybindings) tracks the move-when-stable.

**Refactor** `runInitTUI` to consume these components AND the dispatcher (so till init walks use vim-style keys consistent with the rest of the TUI).

### 2.15 FE scaffold + vim keybinding engine (inline in tillsyn — see §10)

**Stack**: **Wails v2 desktop** (per dev decision) + **Astro** + **SolidJS islands** + **stil tokens** (consumed from `/Users/evanschultz/Documents/Code/hylla/stil/`).

**Build at** `tillsyn/main/fe/`.

**v1 surfaces**:

- Project list page
- Project detail / action item tree
- Action item create dialog
- Dispatcher trigger button (calls dispatcher via Wails IPC)
- Spawn output viewer (live tail)
- Settings panel (view/edit agents.toml, view template.toml, manage groups)

**Wails project structure**:

- `fe/main.go` — Wails main + Service bindings + DEFAULT NATIVE MENU (Quit / About / Hide / Minimize / etc.; no custom menu items in v1)
- `fe/frontend/` — Astro project (package.json, astro.config.mjs, src/)
- `fe/frontend/src/components/` — Tillsyn-specific components (consume stil tokens)
- `fe/frontend/src/lib/vim/` — vim keybinding engine + Wails-aware key handler (see below)
- `fe/wails.json` — Wails config

**Vim keybinding engine** at `tillsyn/main/fe/frontend/src/lib/vim/`:

- `engine.ts` — TS-side vim engine. Consumes `/Users/evanschultz/Documents/Code/hylla/stil/main/src/bindings/baseline.json` AND the Tillsyn-local `<project>/.tillsyn/bindings.json` (per §2.19) at startup. Dispatches key events through mode state machine.
- `types.ts` — types for bindings, modes, dispatch handlers.
- `wails-keys.ts` — Wails-aware keypress filter that runs at document level inside the WebView. Filters OS-level keys (Cmd+Q quit, Cmd+M minimize, Cmd+W close window, Cmd+H hide, Cmd+Tab) so the OS / Wails default menu handles them. Passes everything else to `engine.ts`.
- `palette.ts` — command palette (`:` command mode) backed by the Tillsyn `product_extensions.tillsyn.commands` list per §2.19.
- Vitest unit tests for the engine + wails-keys filter.
- Playwright (via MCP) tests in dev mode for end-to-end keybinding behavior.

**Migration markers**: `engine.ts` + `wails-keys.ts` + `palette.ts` carry `// MIGRATION TARGET: github.com/hylla-org/ro-vim`. UI components carry `// MIGRATION TARGET: @hylla/stil-solid`. Refinement EXTRACT-R2 + KEYBIND-R2 track the moves-when-stable.

**Brand consistency**: consume stil tokens directly from `/Users/evanschultz/Documents/Code/hylla/stil/main/src/styles/tokens.css` (the source-of-truth path; `dist/tokens.css` does NOT exist pre-build per `stil/main/package.json`'s `pnpm build:tokens` step). Consuming `src/` directly avoids requiring a stil build pre-step in Tillsyn's build flow. When stil-solid lands as a pnpm package, switch to the linked path.

**Size-adaptive CSS from day 1** — every component uses container queries / responsive units so future web + mobile wraps work without rework.

**Testing**: Vitest for component unit tests, Playwright (via MCP) for FE integration tests in dev mode. Per the agent-perception model: agents use `browser_snapshot` (accessibility tree) + `browser_take_screenshot` (multimodal Claude sees images) — zero dev-side screenshot capture required.

### 2.16 Refactor `internal/adapters/server/` — INVERSE DISCIPLINE (W7 4-step restructure)

**Disposition history**:
- Round 1 FF1: original "delete entire `internal/adapters/server/`" broke `till mcp`.
- Round 2 R2-FF1: 2-step refactor missed `common/` dependency for `till capture-state` + auth tests.
- Round 3 R3-FF1: refactor missed `mcpapi/` (16K LOC) that `RunStdio` strictly depends on.
- **Pattern observed**: deletion-side enumeration keeps missing dependencies because we don't have Hylla graph-nav available. Each round patches one gap; the next round finds another.

**Disposition (round-3 dev call)**: invert the carving discipline. Specify the RESIDUE (HTTP-specific bits), extract everything else. This is structural, not iterative.

**Four-droplet sequence**:

- **W7.D1 — INVENTORY** (`internal/adapters/server/` audit; produces an MD doc, no code changes)
  - Builder reads every file in `internal/adapters/server/` (top-level files + `mcpapi/` + `common/` + any other subdirs).
  - For each file/symbol, classify into ONE of:
    - **`http-residue`**: HTTP server transport / handler / wire-protocol-specific. Stays in `internal/adapters/server/` for W7.D3 deletion.
    - **`stdio-relevant`**: stdio MCP transport code. Extracts to `internal/adapters/mcp_stdio/` in W7.D2.
    - **`transport-neutral`**: shared scaffolding (Service adapter, auth helpers, MCP RPC handlers like `ServeStdio`, MCP types like `Config`). Extracts to one of: `internal/adapters/mcp_common/` (Service adapter + auth) or `internal/adapters/mcp_rpc/` (MCP RPC engine like the current `mcpapi/`).
  - Use `git grep -n "internal/adapters/server/"` and LSP `findReferences` on every exported symbol in the package to build the dependency-consumer map. Document EVERY consumer in `cmd/till/`, `internal/`, and `*_test.go` files.
  - Output: `workflow/drop_4c_6_1/W7_INVENTORY.md` with three categorized file/symbol lists + the full consumer map. Treats this as a load-bearing artifact W7.D2 and W7.D3 builders consume.
  - Acceptance: every file in `internal/adapters/server/` is classified into exactly one category; every consumer of exported symbols is enumerated with file:line citations; `mage ci` GREEN (no code touched).

- **W7.D2 — EXTRACT EVERYTHING-NOT-HTTP** (per W7.D1's inventory)
  - Create whichever new packages the inventory says are needed (likely `internal/adapters/mcp_stdio/`, `internal/adapters/mcp_common/`, `internal/adapters/mcp_rpc/` from `mcpapi/`).
  - Move every file/symbol classified as `stdio-relevant` or `transport-neutral` to the appropriate new package.
  - Update ALL importers per the consumer map from W7.D1 (`cmd/till/main.go` + tests + any other internal/* file).
  - `mage ci` GREEN after this step. `till mcp` works. `till capture-state` works. Auth-mutation tests pass.

- **W7.D3 — DELETE HTTP RESIDUE** (per W7.D1's inventory)
  - Delete every file/symbol classified as `http-residue`.
  - Remove `till serve` CLI subcommand from `cmd/till/main.go`.
  - Delete HTTP server tests.
  - After deletion, run `mage ci` — failure surfaces ANY missed extraction in W7.D2 (mandatory belt-and-suspenders check).
  - If `internal/adapters/server/` becomes empty after deletion, remove the directory.
  - Acceptance: `git grep -n "internal/adapters/server/"` returns ZERO matches (or only acceptable test-data fixtures); `till serve` no longer in `till --help`; `mage ci` GREEN.

- **W7.D4 — CLAUDE.md cascade table corrections** (renumbered from old W7.D3; unchanged scope)
  - Update cascade table to drop go- prefix, split plan-qa/build-qa rows, document 10-agent set per group, reflect orchestrator-managed as the 10th file kept.

**Pattern discipline (encoded for future drops)**:

- **Specify the residue, not the whole directory.** Carve from the HTTP-side; everything else automatically survives.
- **Two-layer defense**: W7.D1 inventory + W7.D3 final `mage ci` check. Even if inventory misses something, the build failure surfaces it before close.
- **No `mage ci` red between droplets**. Each of W7.D1 → W7.D2 → W7.D3 lands `mage ci` GREEN. Build never breaks.

**Refinement TILL-SERVE-R1**: rebuild a proper HTTP/MCP server from scratch (prereq for web variant + teams feature) — separate later drop.

### 2.17 `till agents bootstrap` CLI (W3 scope-add per disposition 7.2)

- Add subcommand under `till agents`.
- Required flags: `--from <path>` (source dir, e.g., `~/.claude/agents`), `--to <path>` (destination, default `~/.tillsyn/agents`).
- Optional flags: `--dry-run` (preview the cp plan; no writes), `--force` (overwrite existing destination files).
- Behavior:
  - Reads source dir for files matching pattern `<group>-<role>-agent.md` (where group ∈ {go, fe, gen} and role ∈ {builder, planning, qa-proof, qa-falsification, research, closeout, commit-message, etc.}).
  - Writes to `<to>/<group>/<role>-agent.md`.
  - **2-into-4 QA fan-out**: when source has `<group>-qa-proof-agent.md`, write BOTH `<to>/<group>/plan-qa-proof-agent.md` AND `<to>/<group>/build-qa-proof-agent.md` from the same source. Same for `qa-falsification`. Drop 4c.8 properly splits these into distinct prompts (refinement **QA-SPLIT-R1**).
  - **Group-agnostic agents** (e.g., `closeout-agent.md`, `commit-message-agent.md` if no group prefix): copy to each known group dir AND to `gen/`. User can hand-tune per-group later.
  - **Missing files**: report which files were not found (e.g., `orchestrator-managed.md` likely doesn't exist in `~/.claude/agents/`; bootstrap writes a 1-paragraph starter for it).
  - Default destination is `~/.tillsyn/agents/` (HOME tier of the 3-tier resolver).
- Tests: `mage test-pkg ./cmd/till` exercise dry-run + actual copy + 2-into-4 fan-out + missing-file reporting.
- Help text documents the canonical use case (`till agents bootstrap` to populate HOME tier from `~/.claude/agents/` for dogfood) AND notes it's onboarding UX, not the primary content source — substantive embedded defaults ship in Drop 4c.8.

### 2.18 Tillsyn-project-local agent prompts (NEW WAVE W8, parallel with infrastructure)

**Source-of-truth**: this is the work that makes Tillsyn-on-Tillsyn dogfood viable post-4c.6.1.

**Scope**: author substantive prompts for Tillsyn's own project work at `tillsyn/main/.tillsyn/agents/{go,fe}/<name>.md`. These are TILLSYN-AWARE: they encode mage discipline, Section 0 reasoning, MD-only workflow mode, plan-down/build-up, atomic-droplet sizing, Hylla usage, CONSUMER-TIE test contract, our QA disciplines.

**Files to author** (~22 prompts total):

```
tillsyn/main/.tillsyn/agents/go/
  ├── planning-agent.md
  ├── builder-agent.md
  ├── plan-qa-proof-agent.md
  ├── plan-qa-falsification-agent.md
  ├── build-qa-proof-agent.md
  ├── build-qa-falsification-agent.md
  ├── research-agent.md
  ├── closeout-agent.md
  ├── commit-message-agent.md
  └── orchestrator-managed.md
tillsyn/main/.tillsyn/agents/fe/
  └── (same 10 files; FE-flavored content: Playwright, a11y, Astro/Solid, Wails IPC, stil tokens awareness, vim keybinding integration)
```

Per disposition 7.6: SKIP `gen/` for Tillsyn-project-local — Tillsyn's own work is go + fe only.

**Source material per prompt**:

- `~/.claude/agents/<group>-<role>-agent.md` (the dev's system agents — production-grade starting point).
- Project `CLAUDE.md` (cascade tree, agent bindings, build discipline).
- `workflow/example/drops/WORKFLOW.md` (methodology).
- `WIKI.md` (cascade vocabulary, structural_type axis).
- Memory entries: `feedback_plan_down_build_up.md`, `feedback_decomp_small_parallel_plans.md`, `feedback_subagents_background_default.md`, `feedback_section_0_required.md`, `feedback_hylla_go_code.md`, `feedback_cascade_model_policy.md`, `feedback_use_typed_agents.md`, `feedback_commit_style.md`, `feedback_tool_discipline_native_tools.md`, etc.
- Drop 4c.6 + Drop 4c.6.1 worklog patterns (CONSUMER-TIE, atomic-droplet sizing, plan-QA asymmetry, single-line conventional commits).

**Per-prompt acceptance**:

- Frontmatter: `name`, `description`, `model` (per cascade-model-policy), `tools` (per role).
- Body: substantive (>= 1000 chars; not stub). Encodes the role's discipline.
- Passes post-render validator (per Drop 4c.6 W3.D5).
- No Section 0 leakage in committed file.
- Cited memory rules / WORKFLOW.md rules where applicable.

**QA pair per prompt**: build-qa-proof + build-qa-falsification (parallel) per prompt. Proof: each promised discipline encoded; Falsification: can a builder reading this prompt go wrong despite following it?

**Parallel-with-infrastructure**: W8 droplets are file-independent of W0–W7 work (different files). Can dispatch fully parallel with all other waves. Wave A entry.

**Migration markers**: each prompt carries a doc-comment note at the top — `<!-- Tillsyn-project-local; lifted from ~/.claude/agents/ and adapted for Tillsyn's workflow. Future projects use embedded defaults shipped in Drop 4c.8. -->`.

### 2.19 Tillsyn-local vim bindings extension file (per disposition 7.4)

**Decision**: option (b) — Tillsyn-local bindings file, NOT stil-side drop. Faster; stil side stays untouched this drop.

**File**: `tillsyn/main/.tillsyn/bindings.json` (tracked in git per §2.20 .gitignore update).

**Schema**: extends stil's baseline shape per `stil/main/src/bindings/baseline.json`'s `product_extensions.<product>` pattern.

**R3-FF2 disposition (dev call)**: stil baseline.json ALREADY ships `product_extensions.tillsyn` with 4 commands (`new-drop`, `complete-drop`, `handoff`, `comment`). My original 6-command proposal didn't account for this. Resolution:

- **Merge semantic: ID-based deep merge.** Local file's commands UNION with baseline's by command `id`. Local-wins on collision (`local.id == baseline.id`). Loaders in W5 + W6 implement this merge.
- **Final command set v1 (9 commands, deduped)**:
  - **From stil baseline** (4 commands, kept verbatim):
    - `new-drop` — Create a new drop / action item.
    - `complete-drop` — Mark the focused drop / action item complete.
    - `handoff` — Open the handoff panel for the focused project / action item.
    - `comment` — Open the comment thread for the focused project / action item.
  - **Tillsyn-local additions** (5 commands, in `<project>/.tillsyn/bindings.json`):
    - `dispatch` — Trigger dispatcher on the focused action item.
    - `plan` — Open the planner for the focused project or sub-plan.
    - `archive` — Archive the focused project or action item.
    - `settings` — Open the settings panel (agents.toml + template.toml + groups).
    - `help` — Open the help panel (keybinding reference + tips).
- **Dropped**: my original `close` (was redundant with stil's `complete-drop`; stil's name is canonical).

**Tillsyn-local file** at `<project>/.tillsyn/bindings.json`:

```json
{
  "schema_version": 1,
  "name": "tillsyn-bindings",
  "description": "Tillsyn project-local vim bindings extension. ID-merges with stil baseline; local wins on collision.",
  "extends": "stil-baseline",
  "extends_path": "../../../stil/main/src/bindings/baseline.json",
  "product_extensions": {
    "tillsyn": {
      "description": "Tillsyn-specific commands ADDED to baseline (ID-based deep merge; local wins on collision).",
      "commands": [
        { "id": "dispatch", "command": "dispatch", "description": "Trigger dispatcher on the focused action item — accepts optional flags." },
        { "id": "plan",     "command": "plan",     "description": "Open the planner for the focused project or sub-plan." },
        { "id": "archive",  "command": "archive",  "description": "Archive the focused project or action item." },
        { "id": "settings", "command": "settings", "description": "Open the settings panel (agents.toml + template.toml + groups)." },
        { "id": "help",     "command": "help",     "description": "Open the help panel — keybinding reference + tips." }
      ]
    }
  }
}
```

**Loader contract (consumed by W5 + W6)**:
1. Load `stil/main/src/bindings/baseline.json` → seed binding-set + `product_extensions.tillsyn.commands` (the 4 from stil).
2. Load `<project>/.tillsyn/bindings.json` if present → merge into seed by ID; local wins on collision.
3. If `<project>/.tillsyn/bindings.json` ABSENT → use baseline-only (4 stil commands). NOT a fail-loud condition (graceful default).
4. Resulting command palette exposes the UNION of unique-ID commands. v1: 9 commands.

**Consumed by**:

- W5 (TUI vim dispatcher) — loader reads baseline.json + this file, merges product_extensions.tillsyn.commands into the command palette.
- W6 (FE vim engine) — same.
- Test: both surfaces dispatch `:dispatch <action-item-id>` to a handler that invokes Tillsyn's dispatcher Service. Cross-surface consistency.

**Refinement KEYBIND-R3**: when stil-solid lands, move Tillsyn's product_extensions block from this local file INTO `stil/main/src/bindings/baseline.json` so other Hylla products inherit the canonical Tillsyn commands. The local file becomes a no-op or is deleted.

### 2.20 `.gitignore` adjustment for project-local agents + bindings

Current `.gitignore` excludes `.tillsyn/*` and re-includes only `.tillsyn/template.toml`. With §2.18 (Tillsyn-project-local prompts) + §2.19 (Tillsyn-local bindings file), we need MORE re-includes:

```gitignore
# Re-include the project-local agent prompts (Tillsyn-aware overrides).
!.tillsyn/agents/
!.tillsyn/agents/**/*.md

# Re-include the project-local bindings extension.
!.tillsyn/bindings.json
```

Place these alongside the existing `!.tillsyn/template.toml` re-include.

**Runtime state files** (`.tillsyn/config.toml`, `.tillsyn/tillsyn.db`, `.tillsyn/logs/`, `.tillsyn/livewait.secret`) stay ignored. Only `.tillsyn/agents/**/*.md`, `.tillsyn/template.toml`, and `.tillsyn/bindings.json` are tracked.

Folds into W7.D3 (CLAUDE.md update) or its own small droplet — planner's call.

## 3. Out of scope (explicit deferrals)

- **Cascade wiring** (state-trigger autonomous dispatch, full gate runner, post-build pipeline) — Drop 4c.7.
- **Substantive agent prompts** for the 9 × 3 = 27 placeholder files — Drop 4c.8 W4.
- **`till serve` rebuild** — separate drop, prereq for web variant + teams.
- **Web variant of FE** — Wails desktop first; web wrap later when Wails is stable.
- **Mobile (Capacitor) wrap** — later still.
- **Methodology docs** (`CASCADE_METHODOLOGY.md`, `AGENTS_CONFIG.md`, `GDD_METHODOLOGY.md`) substantive content — Drop 4c.8 alongside prompt-authoring.
- **Hylla reset** — orthogonal one-off the dev runs; doesn't gate this drop.
- **Extract TUI components to lykta** — REFINEMENT (post-dogfood).
- **Extract FE components to stil-solid** — REFINEMENT (post-dogfood).

## 4. Refinements logged this session

| ID | Description | Trigger |
|---|---|---|
| D7-R1 | createProjectDBRecord opts into AutoCreateProjectColumns but not AutoSeedStewardAnchors | absorbed into §2.5 |
| D7-R2 | till init doesn't populate RepoPrimaryWorktree | closed by §2.5 |
| D7-R3 | No `till project update` CLI surface | closed by §2.8 |
| D7-R4 | TUI hardwires mcp:false (NIT1) | closed by §2.6 |
| D7-R5 | Dev MCP pollutes workspace `.tillsyn/` runtime files | post-dogfood — separate concern |
| T1-R1 | till serve malformed; delete entirely; rebuild from scratch | closed by §2.16 (delete only; rebuild is later drop) |
| A1-R1 | Drop `stil-rust` adapter (Wails covers Linux) | confirmed; stil README to be updated |
| METHO-R1 | Methodology docs need substantive content during/after Drop 4c.8 | tracked |
| EXTRACT-R1 | Extract `internal/tui/components/` + `internal/tui/style/` to `github.com/hylla-org/lykta` after dogfood is working | tracked; migration markers in each file |
| EXTRACT-R2 | Extract Tillsyn FE components generic enough to be shared into `@hylla/stil-solid` after dogfood is working | tracked; migration markers in each file |
| TILL-SERVE-R1 | Rebuild HTTP/MCP server from scratch as prereq for web variant + teams feature | tracked |
| TUI-R1 | TUI components built tillsyn-internal-first per dev call; pull into lykta once lykta's first published release lands | tracked |
| D7-R6 | Manual cleanup of FLAT agent dirs in existing projects from dogfood-ramp session (TILLSYN-TEST, TILLSYN, /tmp/tillsyn-dogfood-smoke). Dev runs `rm -rf <project>/.tillsyn/agents && till init`. No migration code shipped. | manual; user action |
| ORCH-MANAGED-R1 | Split `orchestrator-managed.md` into role-specific agents (closeout-agent, refinement-agent, discussion-agent, human-verify-agent) during Drop 4c.8 prompt-authoring | tracked; deferred to 4c.8 |
| D-NIL-R1 | gopls warning `nil == nil` at `cmd/till/main.go:3895:73` (tautological condition) — likely dead code or bug. Investigate + fix in separate small drop or fold into next Tillsyn build session | tracked |
| GOPLS-NITS-R1 | Several gopls unused-param / unused-func / refactor-suggestion findings (8 warnings) in `main.go`, `init_cmd.go`, `kind_capability.go`, `handoffs.go`. Pre-existing tech debt. Clean up in a future refinement drop. | tracked |
| BOOTSTRAP-R1 | `till agents bootstrap` extends to OTHER non-`~/.claude/agents/` sources post-MVP (e.g., per-org template libraries, marketplace pulls) | tracked; UX-driven |
| KEYBIND-R1 | Extract `internal/tui/keybindings/` to `github.com/hylla-org/lykta` when lykta publishes | tracked; co-extracts with EXTRACT-R1 |
| KEYBIND-R2 | Extract `fe/frontend/src/lib/vim/` (engine + wails-keys + palette) to `github.com/hylla-org/ro-vim` when ro-vim publishes | tracked |
| KEYBIND-R3 | Move Tillsyn `product_extensions` block from `<project>/.tillsyn/bindings.json` (option b) INTO `stil/main/src/bindings/baseline.json` `product_extensions.tillsyn` when stil-solid lands | tracked; canonicalize after stil drop |
| BIND-CONSIST-R1 | Cross-surface keybinding consistency test (same `j` does next-item in BOTH TUI's action item list AND desktop FE's project list) | tracked; tests TBD-W5+W6 boundary |
| NATIVE-MENU-R1 | Wails native menu integration with vim command dispatch (e.g., File→Open Project triggers same handler as `:plan` vim command) | tracked; post-4c.7 mature |
| QA-SPLIT-R1 | Drop 4c.8 properly authors 4 distinct QA prompts per group (replacing the 2-into-4 duplicate seed from `till agents bootstrap`) | tracked; 4c.8 work |
| EMBED-PROMPTS-R1 | Drop 4c.8 — substantive content for embedded `internal/templates/builtin/agents/<group>/*.md` defaults (30 prompts = 10 × 3 groups). Replaces placeholders. Lets OTHER projects (without `~/.claude/agents/` setup) dogfood out-of-box. | tracked; Drop 4c.8 scope |
| CASCADE-WIRING-R1 | Drop 4c.7 — state-trigger autonomous dispatch on `in_progress`, full gate runner, post-build pipeline. Cascade-driven dogfood. | tracked; Drop 4c.7 scope |
| FE-MOBILE-R1 | Capacitor wrap for mobile when needed | tracked; post-Wails-desktop-stable |
| FE-WEB-R1 | Web variant via proper HTTP server post-till-serve-rebuild | tracked; depends on TILL-SERVE-R1 |

## 5. Acceptance criteria (drop-level)

- 5.1 `till init` in a clean dir with `--groups go` creates a project that the dispatcher can spawn against WITHOUT manual SQL or manual `.mcp.json` writes.
- 5.2 `till init --groups go,fe` works (multi-group), and `<project>/.tillsyn/agents/{go,fe}/<name>.md` subdirs are populated.
- 5.3 `till template save / list / show / diff / restore` work against a real project and HOME tier.
- 5.4 `till agents save / list / show / diff` work.
- 5.5 `till project update --root-path` updates an existing project's worktree path.
- 5.6 `till project delete / archive / restore / rename` work.
- 5.7 `till action_item create` creates an action item under a project from CLI flags.
- 5.8 TUI prompt for `.mcp.json` registration (default yes).
- 5.9 The 9-agent set per group is shipped: file structure correct, no go- prefix orphans, plan-qa-* split from build-qa-*.
- 5.10 fe/ scaffold has a runnable Wails dev mode (`wails dev` works) with stil tokens loading.
- 5.11 `till serve` removed from CLI + server code removed.
- 5.12 `mage ci` green.
- 5.13 The dogfood-readiness test passes: SQL-free, manual-edit-free path from `till init` → `till action_item create` → `till dispatcher run --dry-run` → spawn descriptor renders cleanly.

## 6. Plan-QA gates for this drop

- Plan-qa-proof: every section in §2 has paths declared, packages declared, acceptance bullets, blocked_by between droplets. (Done by planner subagent.)
- Plan-qa-falsification: attack every claim, find counterexamples.

## 7. Implementation guidance

- All work happens in `tillsyn/main` per dev decision. No bootstrapping of separate lykta / stil-solid modules tonight.
- Migration markers (`// MIGRATION TARGET: ...`) on every component that will eventually extract.
- Per-droplet branches optional; pre-cascade we work on main per established pattern (drop-orch + dev manage git directly).
- Parallel work across surfaces is encouraged: TUI components, FE scaffold, CLI surface, agent set restructure are all disjoint enough to dispatch in parallel.

## 8. Test discipline

- TUI: golden tests via `teatest_v2` per existing patterns.
- FE: Playwright via MCP for visual + interaction. Vitest for component unit tests. TS strict.
- CLI: existing `run(ctx, args, &out, io.Discard)` end-to-end pattern (CONSUMER-TIE TEST CONTRACT inherited from Drop 4c.6).
- Backend: Go tests for Service methods; `mage ci` for full gate.
- Cross-suite coverage gate: ≥70% per package (existing rule).

## 9. Coordination

- Pre-cascade per-droplet QA pairs (proof + falsification) for every `build` action item.
- Per dev directive on parallelism: TUI work, FE work, CLI work all parallel (different files / packages).
- Cross-drop coordination NONE — everything in this repo. No `CROSS_DROP_DEPS.md` needed.

## 10. Single-repo (no go.work, no pnpm workspace)

Per dev directive: build TUI components and FE inline in `tillsyn/main`. No bootstrapping of `lykta/` or `stil-solid/` modules tonight. Migration to those modules is REFINEMENT (EXTRACT-R1, EXTRACT-R2) tracked for post-dogfood.

Rationale: faster path to dogfood, migration is a clean later drop, zero cross-repo coordination overhead for this drop. The dev's eventual hylla-dir multi-project workflow can use go.work later when lykta/stil-solid are first published.

## 11. Process from here

- 11.1 Dev reviews this REVISION_BRIEF in chat + iterates with me.
- 11.2 Once converged, spawn planner subagent → emits `workflow/drop_4c_6_1/PLAN.md` with droplet decomposition.
- 11.3 Plan-QA pair (proof + falsification) on the PLAN.md.
- 11.4 Build phase: dispatch builder subagents per droplet (manually via Agent tool pre-cascade-autonomy; via `till dispatcher run` once 4c.7 lands).
- 11.5 Per-droplet QA pairs (build-qa-proof + build-qa-falsification).
- 11.6 Closeout per cascade methodology (manual pre-dogfood, automated post-dogfood).
- 11.7 After this drop ships green + lands on main: Tillsyn can dogfood itself. Start Drop 4c.7 (cascade wiring) using the dogfooded cascade.
