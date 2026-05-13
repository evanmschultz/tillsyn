# SKETCH.md — Drop 4c.6.1 — User Surface + Multi-Group + FE Bootstrap

**Status:** SKETCH (informal scope-decision capture). Companion to REVISION_BRIEF.md. Codifies the design discussions from 2026-05-12's post-shipped-Drop-4c.6 dogfood-ramp session.

## §0 Why this sketch exists

Drop 4c.6 shipped the architectural slab (resolvers, validators, isolation argv, embedded scaffolding). The user-surface layer that would make a project actually dogfood-ready was deferred to "later drops" that hadn't been planned in concrete scope. When the dev tried to dogfood Tillsyn against itself, every step from `till init` to dispatcher-spawn surfaced a gap.

This sketch captures the architectural decisions made during dogfood-ramp discovery, so the planner subagent has design-level grounding before emitting PLAN.md.

## §1 The 3-tier resolution story (the architectural unification)

Drop 4c.6 shipped a 3-tier resolver for **agent bodies** (project → user → embedded). The dogfood-ramp session revealed this should apply CONSISTENTLY to:

1. **Agent bodies** (`.md` files): project local → user HOME → embedded — ✓ Drop 4c.6 shipped this.
2. **Templates** (`template.toml`): project local → user HOME → embedded — Drop 4c.6 shipped only project + embedded. **THIS DROP adds user HOME tier.**
3. **Runtime config** (`agents.toml`): project local → embedded example. Drop 4c.6 ships per-project. **No user HOME tier needed** per dev — `agents.toml` is per-project runtime, `template.toml` is the right portability surface.
4. **agents.local.toml** (gitignored project-local override): unchanged from Drop 4c.6.

### §1.1 Why agents.toml doesn't get a HOME tier

`agents.toml` carries project-specific runtime knobs (which kinds use which models, which tools each kind has, env vars). These vary per project (a tools-research project might need different env than a UI project). `template.toml` is the structural cascade definition (agent_bindings, child_rules, validators) — that's what users want to share across projects.

So:
- Users save `template.toml` to HOME → reused across projects via `till init`.
- Users save agent BODIES to HOME → reused across projects via `till init`.
- Users do NOT save `agents.toml` to HOME — each project authors its own runtime config (often by copying the embedded example).

## §2 Groups are first-class

Drop 4c.6 had a single-group model: `till init --group till-go` copies one group's agents. Reality demands multi-group:

- Tillsyn itself = Go backend + FE app = wants `go` group AND `fe` group.
- A user's project might be Go-only, or FE-only, or both, or even three.
- Groups must compose, not exclude.

### §2.1 Group composability

Each group is a **collection of agent prompts + a template** that handles a domain (Go work, FE work, generic work, future: Rust/Python/whatever).

- Built-in groups: `go`, `gen`, `fe` (this drop ships these three; future groups added later).
- Groups are **disjoint at the file-system level**: each lives at `agents/<group>/<name>.md`. No file-name collisions across groups.
- The standard 9-agent set per group: `planning-agent`, `builder-agent`, `plan-qa-proof-agent`, `plan-qa-falsification-agent`, `build-qa-proof-agent`, `build-qa-falsification-agent`, `research-agent`, `closeout-agent`, `commit-message-agent`.
- A project picks ONE or MORE groups at `till init`. The selected groups' agents AND template entries get aggregated into the project.
- Each agent's group is determined by its subdir, NOT by a prefix in its filename. The `go-` prefix from Drop 4c.6 W5.D3 is dropped entirely.

### §2.2 Built-in vs project-specific

Two distinct things share the same file layout:

- **Built-in groups** (shipped in the binary, embedded in `internal/templates/builtin/agents/<group>/`): generic concerns. `fe` group's built-ins are FE-stack-agnostic — Playwright, a11y, visual QA, responsive design. Not specific to Astro/Solid/Wails.
- **Project-specific overrides** (project local + user HOME): Tillsyn's own FE agents (which DO know about Astro/Solid/Wails/stil) live in `tillsyn/main/.tillsyn/agents/fe/`, optionally saved to `~/.tillsyn/agents/fe/` so other hylla projects can use them.

### §2.3 `fe` built-in scope

The `fe` group's built-in 9 agent prompts cover generic FE work:

- Builder: knows Vitest, Playwright, ESLint, TS, CSS, accessibility basics.
- Planner: knows component architecture, design tokens, responsive design.
- Plan-QA: reviews FE plans (component boundaries, accessibility coverage, responsive coverage).
- Build-QA: reviews FE code (Playwright pass rate, a11y violations, type errors, test coverage).
- Research, Closeout, Commit-message: generic.

The Tillsyn-specific FE agents that know Wails/Astro/Solid/stil land in `tillsyn/main/.tillsyn/agents/fe/` and `~/.tillsyn/agents/fe/` as project + HOME overrides on top.

## §3 Plan-QA vs Build-QA — separate prompts

Drop 4c.6 shipped 2 QA agents (`qa-proof-agent`, `qa-falsification-agent`) covering 4 cascade kinds. The dev's directive: split into 4 separate agent files per group.

### §3.1 Why split

Different work, different evidence sources, different attack angles:

- **plan-qa-proof**: verifies the plan is complete + evidence-backed. Reviews PLAN.md droplet decomposition, `blocked_by` graph correctness, paths/packages declarations, acceptance bullets, surface boundaries.
- **plan-qa-falsification**: attacks the plan. Finds hidden dependencies, missing droplets, scope creep, blocker cycles, paths/packages overlaps not declared as blocked_by.
- **build-qa-proof**: verifies the build matches the plan. Reviews actual code changes, test pass rates, evidence for each acceptance bullet, no scope creep beyond the droplet's paths.
- **build-qa-falsification**: attacks the build. Finds counterexamples to test claims, race conditions, edge cases the tests miss, false-passes, test residue, security gaps.

Same name (`qa-proof`) covering both layers conflates evidence sources (PLAN.md text vs Go test output) and attack angles. Separate prompts let each agent specialize.

### §3.2 Model assignment per kind

CLAUDE.md cascade table assigns:

- `plan` → planning-agent (opus)
- `plan-qa-proof` → plan-qa-proof-agent (opus)
- `plan-qa-falsification` → plan-qa-falsification-agent (opus)
- `build` → builder-agent (sonnet)
- `build-qa-proof` → build-qa-proof-agent (opus per `feedback_cascade_model_policy.md`)
- `build-qa-falsification` → build-qa-falsification-agent (opus per `feedback_cascade_model_policy.md`)
- `research` → research-agent (opus per `feedback_cascade_model_policy.md`)
- `closeout` → orchestrator-managed (or closeout-agent when authored)
- `commit-message` → commit-message-agent (haiku per `feedback_cascade_model_policy.md`)

## §4 TUI components — inline first, extract later

### §4.1 Decision: build in `tillsyn/main/internal/tui/components/`

Per dev call. Alternative was bootstrapping `lykta/main/` as a separate Go module. Rejected for now because:

- Faster to dogfood (no module bootstrap, no go.work coordination).
- Migration to lykta is a clean later drop once lykta first publishes.
- Every component carries `// MIGRATION TARGET: github.com/hylla-org/lykta` so future extraction is mechanical.

Refinement entry **EXTRACT-R1** tracks this.

### §4.2 Component set

Initial set (matches CLI needs):

- `confirm.go` — y/n prompt with default. Used by till init's `.mcp.json` confirm (closes NIT1) and any future destructive-action confirms.
- `textinput.go` — single-line text input with validation. Used by till init's project name field.
- `picker_single.go` — styled single-select list. Used by future `till template show --source` choice etc.
- `picker_multi.go` — styled multi-select list. Used by till init's multi-group picker.
- `header.go` / `footer.go` — styled chrome.
- `progress.go` — single-step status line.

### §4.3 Style system

At `internal/tui/style/`:

- `palette.go` — colors via lipgloss styles, semantic names (primary, accent, success, warning, error, muted).
- `spacing.go` — padding / margin constants for consistency.
- `typography.go` — text style helpers (heading, body, label, code).

Brand consistency: where possible, the TUI palette mirrors `stil`'s color tokens (translated from CSS to lipgloss). Future: `lykta` package consumes stil tokens directly via a Go-side adapter when stil-solid ships its tokens.json.

## §5 FE — Wails desktop, inline first

### §5.1 Decision stack

- **Wails v2** (desktop framework, Go backend + WebView frontend, cross-platform via OS WebView).
- **Astro** as the frontend framework producing the static bundle Wails embeds.
- **SolidJS** for islands (per stil-solid's planned shape).
- **stil tokens** consumed from `/Users/evanschultz/Documents/Code/hylla/stil/main/dist/tokens.css` (built artifact) or via pnpm linked path when stil-solid ships.
- **No till serve** — Wails IPC handles Go ↔ JS directly. No HTTP layer needed for the desktop app.

### §5.2 Why Wails over Tauri/Electron

- **Wails**: Go backend native (no FFI, no RPC shim). ~15MB bundle. OS WebView per platform (WebKit/WebView2/WebKitGTK). Cross-platform desktop without separate Rust/Node toolchain.
- **Tauri 2.0**: would require Rust backend + FFI to Go. More polyglot, more glue.
- **Electron**: ~70MB bundle, Node backend, more glue to Go.

Wails wins on "least new tech in our stack" criterion.

### §5.3 v1 desktop surfaces

Minimum surfaces for Tillsyn dogfood-via-desktop-UI:

- Project list page (table with archived filter, create button).
- Project detail / action item tree (left pane: collapsible tree; right pane: action item detail).
- Action item create dialog (kind picker, paths input, description editor).
- Dispatcher trigger button (per action item, "Run" → calls `dispatcher.RunOnce` via Wails IPC).
- Spawn output viewer (live tail of subagent output).
- Settings panel (view/edit agents.toml, view template.toml, manage groups).

**Out of v1**: log viewer history, embeddings management, capability lease UI, auth flow UI, multi-project orchestration views. Those use the CLI until later.

### §5.4 Wails project layout

```
tillsyn/main/fe/
├── main.go                  # Wails main + Service bindings
├── wails.json               # Wails config
├── frontend/
│   ├── package.json         # Astro + Solid + dev deps
│   ├── astro.config.mjs
│   ├── pnpm-lock.yaml
│   ├── public/
│   │   └── stil-tokens.css  # built artifact from stil (or linked)
│   └── src/
│       ├── pages/           # Astro pages (project list, detail, settings)
│       ├── components/      # Tillsyn FE components (per-component MIGRATION TARGET comments)
│       ├── layouts/         # Astro layouts
│       └── lib/             # client-side helpers, Wails IPC wrappers
└── go.mod                   # Wails wraps the existing Service from internal/app
```

### §5.5 Brand consistency

- Color palette derived from stil tokens.
- Typography per stil's font choices (Inter, JetBrains Mono per stil's package.json).
- Spacing scale per stil.
- Component primitives (button, modal, input) follow stil's design language even when Tillsyn-built (later extraction to stil-solid via EXTRACT-R2).

### §5.6 Size-adaptive from day 1

Per dev directive — every component built responsive:

- Container queries where possible (component-driven layout).
- Responsive typography (clamp/min/max).
- Min/max widths so layouts work in compact-IDE-pane (300px wide) AND full-screen-desktop (1920px wide) AND eventually mobile.

This isn't extra work; it's the baseline.

### §5.7 Testing layered

- Playwright (via MCP) drives Astro dev mode at `localhost:3000` (or Wails dev port). Visual regression + interaction tests for FE components.
- **Agent perception is fully automated — no dev-side screenshot capture required.** FE agents use the Playwright MCP toolset to "see" the UI in two complementary ways:
  - `browser_snapshot` returns the page's accessibility tree as text. Agent reads it semantically: button labels, ARIA roles, headings, form fields, table structure. Tells the agent WHAT'S on screen without rendering pixels.
  - `browser_take_screenshot` returns the rendered image as an attachment. Multimodal Claude (the FE agents) consumes the image directly — literally sees the UI. Used for visual-correctness checks (alignment, color, spacing, brand consistency with stil tokens).
  - Combined: agent edits a component → starts dev server (Bash, run_in_background) → `browser_navigate` to local port → `browser_snapshot` for semantic check → `browser_take_screenshot` for visual check → iterate until correct → write a Playwright test asserting the state → run via Bash.
- Vitest for component unit tests.
- TS strict + ESLint.
- Go tests for the Service methods Wails IPC binds.
- Manual smoke of the built Wails binary for the integration layer (the rare case where dev-side eyes-on is the only path — Playwright doesn't drive Wails binaries directly, only the embedded dev-mode Astro server).

## §6 till serve — delete now, rebuild later

Drop 4c.6 inherited a malformed `till serve` from prior drops. Per dev directive: delete entirely.

- Remove CLI subcommand from `cmd/till/main.go`.
- Remove `internal/adapters/server/` package and its tests.
- Refinement **TILL-SERVE-R1** tracks the future rebuild from scratch as a proper HTTP/MCP server (prereq for web variant + teams feature).

## §7 Parallel within one drop

Per dev directive — parallelism is at the **droplet** level within this single drop, NOT at the cross-drop level (everything is in this repo).

Disjoint surfaces that can dispatch in parallel:

- TUI components + style system (`internal/tui/`)
- FE scaffold (`fe/`)
- Bake walker HOME tier (`internal/app/service.go`)
- 9-agent restructure (`internal/templates/builtin/agents/<group>/`)
- CLI surface (`cmd/till/`)
- agents.toml + template.toml schema shift (`internal/templates/builtin/*.toml`)
- till serve deletion (`cmd/till/main.go` + `internal/adapters/server/`)

Cross-cutting (serial-ish):

- CLAUDE.md cascade table update (only after agent restructure lands so the right names are referenced)
- agents.example.toml update (only after schema shift lands)

The planner subagent will work out the exact droplet decomposition and `blocked_by` graph.

## §8 Refinements — future work tracked

| ID | What | When |
|---|---|---|
| EXTRACT-R1 | Move `internal/tui/components/` + `internal/tui/style/` to `github.com/hylla-org/lykta` | post-dogfood, lykta first-publish |
| EXTRACT-R2 | Move Tillsyn-generic FE components to `@hylla/stil-solid` | post-dogfood, stil-solid first-publish |
| TILL-SERVE-R1 | Rebuild HTTP/MCP server from scratch | prereq for web variant + teams feature |
| METHO-R1 | Methodology docs (`CASCADE_METHODOLOGY.md`, `AGENTS_CONFIG.md`, `GDD_METHODOLOGY.md`) substantive content | Drop 4c.8 alongside prompt-authoring |
| A1-R1 | Update `stil/README.md` to drop the `stil-rust` adapter mention | low priority cosmetic |
| D7-R5 | Dev MCP pollutes workspace `.tillsyn/` runtime files in `--dev` mode | post-dogfood UX cleanup |
| FE-MOBILE-R1 | Capacitor wrap for mobile when needed | post-Wails-desktop-stable |
| FE-WEB-R1 | Web variant via proper HTTP server | post-till-serve-rebuild |

## §9 Acceptance bar for dogfood-readiness

After this drop ships green on main, the dev should be able to:

1. `till init --groups go,fe` in any directory → fully populated project, ready for dispatcher.
2. `till action_item create --project-id X --kind build --title "..." --paths "..." --description "..."` → action item created.
3. `till dispatcher run --action-item <id> --dry-run` → clean spawn descriptor, no SQL hacks, no manual `.mcp.json` writes.
4. Drop the `--dry-run` → real spawn fires; the spawned Claude Code subagent reads the 3-tier-resolved agent body (project-local override > HOME > embedded), edits code per acceptance, returns clean.
5. `till template save --group go` → user's HOME tier populated; next `till init` reads it.
6. `wails dev` in `tillsyn/main/fe/` → Tillsyn desktop app launches, shows project list, can trigger dispatcher.

When all 6 work end-to-end without papercuts, Tillsyn is dogfood-ready. Drop 4c.7 (cascade wiring) starts then, using the dogfooded cascade.

## §10 Architectural decisions explicit

For posterity (and so the planner subagent has them grounded):

| Decision | Value | Why |
|---|---|---|
| Multi-group projects | YES, composable | Tillsyn = go + fe; users will mix. |
| Agent file naming | drop `go-` prefix; subdir-per-group is the distinguisher | Cleaner; dev directive. |
| Plan-QA vs Build-QA agents | SPLIT into 4 separate files per group | Different work, different prompts. |
| Agent count per group | **10 (9 standard + `orchestrator-managed.md`)** | FF3 disposition — keep orchestrator-managed for closeout/refinement/discussion/human-verify kinds until Drop 4c.8 splits them. ORCH-MANAGED-R1 tracks. |
| FLAT-to-subdir migration | **NO migration code** | FF2 disposition — clean cutover; existing FLAT projects manually rebuilt. Builder fails loud on detected FLAT layout. D7-R6 tracks the manual cleanup of existing dogfood-ramp projects. |
| `till action_item create --structural-type` | **Smart-default per kind, override via flag** | FF4 disposition. plan/refinement → segment; all other kinds → droplet. Override validates against the closed enum. |
| TUI components location | inline `tillsyn/main/internal/tui/components/` for now | Faster to dogfood; migration tracked. |
| FE location | **Wails at tillsyn repo root (canonical Wails v2 layout)** — Round 10 W6 fals FF2 | `main.go` + `app.go` + `wails.json` + `frontend/` live at the tillsyn repo root, sharing the existing single `go.mod` with `cmd/till/` + `internal/...`. NO separate `fe/go.mod`, NO `replace` directive, NO `fe/` subfolder. Wails-tagged files use `//go:build wails` to isolate CGO from default `mage ci` builds. Future surfaces: web variant = new `cmd/till-web/main.go` in same module; mobile = Capacitor wraps `frontend/dist/` (no extra Go); cloud auth server = SEPARATE hylla-org repo. |
| FE framework | Wails + Astro + Solid + stil tokens | Go backend native; no till-serve dep; lean stack. |
| Mobile / web variants | Deferred (refinements logged) | Desktop first per dev. |
| `till serve` deletion | **INVERTED CARVING: 4-droplet sequence — Inventory → Extract-everything-not-HTTP → Delete-residue → CLAUDE.md update** | Round-3 R3-FF1 found a THIRD missed dependency (`mcpapi/`, 16K LOC) after rounds 1 + 2 patched `till mcp` + `common/`. Dev disposition: invert the carving discipline. W7.D1 produces an INVENTORY MD doc classifying every file/symbol in `internal/adapters/server/` as http-residue / stdio-relevant / transport-neutral with full consumer map. W7.D2 extracts everything-not-HTTP per inventory. W7.D3 deletes only the http-residue + runs `mage ci` as a belt-and-suspenders check (failure surfaces any missed extraction). W7.D4 = CLAUDE.md update. TILL-SERVE-R1 tracks rebuild. |
| Vim bindings merge semantic | **ID-based deep merge; local wins on collision** | R3-FF2 found stil baseline.json ALREADY has `product_extensions.tillsyn` with 4 commands (`new-drop`, `complete-drop`, `handoff`, `comment`). Dev disposition: union by command ID; local wins on collision. v1 command palette = 9 commands (stil's 4 + Tillsyn-local 5: `dispatch`, `plan`, `archive`, `settings`, `help`). Original `close` dropped (redundant with stil's canonical `complete-drop`). Loaders gracefully fall back to baseline-only when local file absent (not fail-loud). |
| Stil tokens consumption path | **`stil/main/src/styles/tokens.css` (source-of-truth)** | R3-NIT7 found `dist/tokens.css` doesn't exist pre-build (stil's `pnpm build:tokens` produces it). Consume `src/` directly to avoid requiring a stil build pre-step in Tillsyn's build flow. When stil-solid publishes as pnpm package, switch to the linked artifact path. |
| Tillsyn-project-local agent prompts | **Authored as W8 wave parallel with infrastructure** | Per-disposition 7.1 — bundled in Drop 4c.6.1. ~22 prompts at `tillsyn/main/.tillsyn/agents/{go,fe}/`. Source: `~/.claude/agents/` + project CLAUDE.md + WORKFLOW.md + WIKI.md + memories. Enables Tillsyn-on-Tillsyn dogfood immediately post-4c.6.1. Skip `gen/` per 7.6. |
| `till agents bootstrap` CLI | **Folded into W3 (disposition 7.2)** | Subcommand under `till agents`. Maps `~/.claude/agents/<group>-<role>-agent.md` → `~/.tillsyn/agents/<group>/<role>-agent.md`. 2-into-4 QA fan-out from 2-file system shape to 4-file Tillsyn shape. Onboarding UX, not the primary content source. |
| Vim keybindings — overall architecture | **Single source-of-truth via stil/bindings/baseline.json + Tillsyn-local extension** | Both TUI (W5) and FE (W6) consume stil baseline AND `<project>/.tillsyn/bindings.json`'s product_extensions.tillsyn. Cross-surface consistency: same `j` does next-item everywhere. Refinement BIND-CONSIST-R1 tracks test. |
| Vim keybindings — TUI (W5) | **`internal/tui/keybindings/` dispatcher consuming stil baseline + Tillsyn-local extension** | Go-side dispatcher with bubbletea key event dispatch. Mode state machine (nav/insert/visual/visual-block/command/hint). Migration target lykta (KEYBIND-R1). |
| Vim keybindings — FE/desktop (W6) | **`frontend/src/lib/vim/` engine + wails-keys filter + palette** (Round 10 path update — Wails at repo root) | TS-side engine inside WebView. `wails-keys.ts` filters OS-level keys (Cmd+Q/M/W/H stay native) before dispatching to engine. Migration target ro-vim (KEYBIND-R2). Vitest + Playwright tests. |
| Vim product_extensions location | **Tillsyn-local file at `<project>/.tillsyn/bindings.json` (disposition 7.4 option b)** | Faster than a stil-side drop. Refinement KEYBIND-R3 moves to stil's baseline.json `product_extensions.tillsyn` when stil-solid lands. |
| Vim command palette v1 | **`:dispatch`, `:plan`, `:archive`, `:settings`, `:help` (disposition 7.5)** — `:close` DROPPED per R3-FF2 (redundant with stil baseline's `complete-drop`) | Minimum set. Tillsyn-local additions are these 5 commands; merged with stil baseline's 4 (`new-drop`, `complete-drop`, `handoff`, `comment`) via ID-based deep merge = 9 total. Future commands added as Tillsyn surfaces grow. |
| Wails native menu | **Default Wails menu in v1** | Quit/About/Hide/Minimize wire automatically. No custom items. Refinement NATIVE-MENU-R1 for future vim-aware menu items. |
| `.gitignore` for Tillsyn-project-local artifacts | **Re-include `.tillsyn/agents/**/*.md` + `.tillsyn/bindings.json`** | Alongside existing `!.tillsyn/template.toml` re-include. Runtime files (`config.toml`, `tillsyn.db*`, `logs/`, `livewait.secret`) stay ignored. |
| Recursive dogfood for embedded built-ins | **Drop 4c.8 authors 30 substantive prompts via the now-mature cascade** | Refinement EMBED-PROMPTS-R1. 10 prompts × 3 groups (gen + go + fe). Replaces 4c.6.1's placeholders so other projects dogfood out-of-box. |
| `internal/config/agents.go` decoder shape | **Multi-group `[<group>]` / `[<group>.<kind>]` deep-merge** | Plan-QA proof-FF1 disposition — adds REVISION_BRIEF §2.12a covering decoder + `agents.local.toml` deep-merge over the new shape. Blocks §2.12 TOML file rewrites (decoder shape is the contract). |
| Methodology docs | Defer substantive content to Drop 4c.8 | Substantive content lands alongside prompts. |
| `stil-rust` adapter | Drop from plan entirely | Wails covers Linux via WebKitGTK. |
| `agents.toml` HOME tier | NO | Per-project runtime, not shared. `template.toml` IS the portability surface. |
| Schema shape | `[<group>]` and `[<group>.<kind>]` (no `agents.` / `template.` prefix) | File name already says it. |
| Cross-drop coordination | NONE this drop | Everything inline. |
| go.work / pnpm workspaces | Skip for now | No cross-module deps; faster to dogfood. |
| **Embedded-template subdir names (Round 10 W1+W2 fals FF1)** | **Canonical `go/` + `gen/` + `fe/` — NO `till-` prefix** | W4.D1 performs `git mv till-go → go` + `git mv till-gen → gen` to align with SKETCH §2.1 + new `[<group>.<kind>]` schema. `agentBodyDefaultGroup` constant updates `"till-go"` → `"go"` (and fallback `"till-gen"` → `"gen"`) in `render.go` via W1.D3 scope. `till-gdd/` subdir stays — it's a template-family identifier, not a group. |
| **`ProjectMetadata.Groups []string` typed field (Round 10, 5x confirmation across W1/W2/W3 plan-QA)** | **W1.D2 ships typed field; W2.D7 + W3.D1 consume directly — no JSON stopgap, no TODO fallback** | `domain.ProjectMetadata` gets `Groups []string` field in W1.D2. `W2.D7` writes `Metadata.Groups = payload.Groups` directly. `W3.D1` (`till project update --add-group/--remove-group`) modifies the typed field. W2-GROUPS-R1 refinement RESOLVED inline. |
| **`@fontsource/*` font packages (Round 10 W6 fals FF3)** | **`@fontsource/inter`, `@fontsource/iosevka`, `@fontsource/fira-code`, `@fontsource/jetbrains-mono` in `frontend/package.json`** | stil `tokens.css` declares `--font-family-sans: 'Inter'` etc. but ships fonts via `@fontsource/*` pnpm deps — NOT inline. Without these deps + a `MainLayout.astro` import, browser falls back to system fonts and AC2 fails. |
| **Bubble Tea v2 sub-component shape (Round 10 W5 fals FF1+FF2)** | **W5 components are sub-models composed by outer `tea.Model`; not `tea.Model` themselves; never `return tea.Quit` from sub-component** | `tea.Model` in Bubble Tea v2 (`charm.land/bubbletea/v2@v2.0.0-rc.2/tea.go:52-63`) requires `View() View` struct, NOT `View() string`. W5 components have `View() string` returns and don't satisfy the interface — that's fine because they're composed by an outer `tea.Model` at `internal/tui/model.go`. Spec drops the `var _ tea.Model = (*ConfirmModel)(nil)` claim. Components use `return nil` + `Done()`/`Cancelled()` accessors (NOT `tea.Quit` which kills the parent TUI). |
| **W8 prompt frontmatter `model:` field (Round 10 W8 proof+fals FF)** | **Bare aliases: `model: sonnet`/`opus`/`haiku`/`orchestrator-managed`** | Matches live `~/.claude/agents/go-*-agent.md` system frontmatter. Auto-tracks Claude Code's model-resolver upgrades. Versioned IDs (e.g. `claude-sonnet-4-6`) rot fast as model families advance. |

## §11 What dev signs off on

Reading this sketch + the REVISION_BRIEF, dev should confirm:

- §2 scope (every numbered subsection in 2.1–2.16).
- §3 out-of-scope deferrals.
- §4 refinements list.
- §5 acceptance criteria.
- §10 architectural decisions.

Once converged, planner subagent emits PLAN.md decomposing §2 into atomic droplets with blocked_by graph + paths/packages declarations.
