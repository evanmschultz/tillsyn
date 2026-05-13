# DROP 4c.6.1.W6 — FE SCAFFOLD (Wails v2 at tillsyn repo root + Astro + Solid + stil tokens + vim engine)

**State:** planning
**Kind:** plan (L2 sub-plan; children are atomic build droplets)
**Parent plan:** `workflow/drop_4c_6_1/PLAN.md` § Wave W6
**Source-of-truth scope:** REVISION_BRIEF §2.15 (updated Round 10 for Wails-at-root); SKETCH §5 + §10
**Wave position:** Wave A head — no external blockers; blocks nothing external (FE is independent of Go cmd/till waves)
**Hylla:** OFF (non-Go files / build-tagged root files; LSP + Context7 used for layout verification)

---

## Round 2 Changes (absorbed 2026-05-12)

**Changelog from round-1 plan-QA (proof + falsification) + R10-D3 Wails-at-root decision:**

### RESOLVED (orchestrator-fixed unilaterally before round-2 dispatch)
- **Proof FF1 RESOLVED**: `KindPayload.children` `blocked_by` contradiction with `_BLOCKERS.toml` for D5/D6/D7/D8. Orchestrator fixed unilaterally: KindPayload.children, Droplet Graph, and Dispatch order now all declare the serial chain D4→D5→D6→D7→D8 consistent with `_BLOCKERS.toml`. Preserved in this rewrite.

### ABSORBED — R10-D3 Wails-at-root (major path rewrite)
- **Fals FF2 ABSORBED**: Non-canonical Wails layout absorbed by R10-D3 decision (Context7 `/wailsapp/wails` confirms canonical: `go.mod` + `main.go` + `wails.json` + `frontend/` all at project root). All `fe/` paths changed:
  - `fe/go.mod` → REMOVED (no separate module; single shared `go.mod`)
  - `fe/main.go` → `main.go` (root; `//go:build wails`)
  - `fe/app.go` → `app.go` (root; `//go:build wails`)
  - `fe/wails.json` → `wails.json`
  - `fe/frontend/` → `frontend/`
  - All D1–D9 path declarations updated accordingly.
  - D1 RiskNote R2 (replace directive + placeholder `require`) removed entirely — no separate module.

### ABSORBED — Proof NITs
- **NIT1**: Spawn-prompt surface count (6 surfaces / 5 D-droplets) — orchestrator-side note only; no PLAN.md change.
- **NIT2**: D3 shape_hint `solid` → `solidJs` to match Context7 canonical `@astrojs/solid-js` import name.
- **NIT3**: D3 `env.d.ts` — added explicit `KindPayload.changes` entry (deliberate authoring step).
- **NIT4**: D9 migration marker phrasing — tightened to "`//`-style line comment at the top of the file, before any imports."
- **NIT5**: D9 RiskNote HTTP-fetch mechanism — explicit in D9 AC: `fetch('/stil-baseline.json')` for baseline, `fetch('/.tillsyn-bindings.json')` for local extension; 404 → baseline fallback.
- **NIT6**: D2 AC — `RunDispatcher` stub-path made explicit in AC ("delegates to `Service.RunDispatcher` OR returns wrapped `ErrNotImplemented`; both are acceptable v1 wiring").
- **NIT7**: Migration-marker CI gate — D3 adds a `tests/migration-markers.test.ts` Vitest test that walks `src/components/` + `src/lib/vim/` and asserts marker substring presence.

### ABSORBED — Fals FFs and NITs
- **Fals FF1 ABSORBED**: `mage ci` exclusion now via `//go:build wails` isolation. D1 acceptance explicitly requires: "`go test ./...` from tillsyn root WITHOUT `-tags wails` completes cleanly; root `main.go` + `app.go` are skipped by build tag; no `./main` or `./app` packages appear in coverage output." Verified at D1 close (no D2 delta needed since D2 adds to the same wails-tagged root files).
- **Fals FF3 ABSORBED**: D3 `package.json` now includes `@fontsource/inter`, `@fontsource/iosevka`, `@fontsource/fira-code`, `@fontsource/jetbrains-mono` as pnpm deps (confirmed from `stil/main/package.json` deps). D4 `MainLayout.astro` imports them via `@fontsource/inter` etc.
- **Fals FF4 ABSORBED (revised post-R2 fals FF1-R2)**: D9 RiskNotes collapsed to ONE chosen path: baseline.json via `fetch('/stil-baseline.json')` (file in `frontend/public/`; copied by D3); `.tillsyn/bindings.json` via `fetch('/.tillsyn-bindings.json')` from `frontend/public/.tillsyn-bindings.json` (D3 creates this as a symlink to `../../.tillsyn/bindings.json` OR a copy — Astro dev follows the symlink; `pnpm build` resolves it into the static `dist/` bundle for production). **NO vite-server proxy** (per W6 round-2 fals FF1-R2: `vite.server.proxy` is HTTP-to-HTTP only; `rewrite` transforms URL paths not filesystem paths — original round-2 mechanism mechanically wrong). Wails IPC `GetBindingsJSON` alternative remains explicitly refuted. **Limitation accepted for v1**: production binary captures `.tillsyn/bindings.json` at BUILD time; runtime edits require rebuild. v2 may reintroduce IPC `GetBindingsJSON` for live-reload.
- **Fals N3 ABSORBED**: D8 paths now include `frontend/src/layouts/MainLayout.astro` (MODIFY — add settings nav link).
- **Fals N4 ABSORBED**: `frontend/public/stil-baseline.json` copy moved from D9 to D3 paths + D3 KindPayload.
- **Fals N5 ABSORBED**: D3 AC pre-bakes `vitest run --passWithNoTests` in `test:unit` script.
- **Fals N6 ABSORBED**: D9 AC7 qualified as macOS-only; FE-CROSS-PLATFORM-R1 refinement added.
- **Fals N7 ABSORBED**: D9 AC adds malformed `.tillsyn/bindings.json` handling: log parse error to console + fallback to baseline-only.
- **Fals N8 DEFERRED-AS-NIT**: `close` command absence assertion — deferred. The closed 9-command set has no `close`; a vacuous assertion adds no signal. The REVISION_BRIEF §2.19 history is the authoritative guard.

---

## Specify

### Objective

Bootstrap the Wails v2 desktop application for Tillsyn at the tillsyn repo root (canonical Wails v2 layout per Context7). `main.go` and `app.go` live at the repo root with `//go:build wails` to isolate CGO from default `mage ci` builds. Deliver 6 v1 surfaces (project list, project detail + action item tree, action item create dialog, dispatcher trigger + spawn output viewer, settings panel) backed by real Wails IPC calls to `internal/app.Service`. Add a TypeScript-side vim keybinding engine (`engine.ts` + `types.ts` + `wails-keys.ts` + `palette.ts`) with ID-based deep merge of stil baseline commands (4) + Tillsyn-local extensions (5) = 9 commands. Add `mage ci-fe` target to `magefile.go` covering Vitest unit tests + pnpm build. All FE component files carry `// MIGRATION TARGET: @hylla/stil-solid`; all vim engine files carry `// MIGRATION TARGET: github.com/hylla-org/ro-vim`.

### AcceptanceCriteria

- AC1 (`wails dev`): running `wails dev -tags wails` from the tillsyn repo root starts the Tillsyn desktop app without errors; the project list page is reachable at the Astro dev server URL. **Note** (W6 fals NIT1, R2): Wails v2 docs do NOT list `tags` in `wails dev -save`'s auto-persisted-flag set per Context7 `/wailsapp/wails` `project-config.mdx`; builder treats `-tags wails` as a per-invocation requirement. If empirical testing confirms `-save` does persist it (docs may be partial), builder may update `wails.json` manually with the documented schema field — otherwise dev workflow documents `wails dev -tags wails` as the standard invocation.
- AC2 (stil tokens + fonts): stil design tokens load from `frontend/public/stil-tokens.css` (copied from `stil/main/src/styles/tokens.css`); `--carl` accent variable is present; Inter/Iosevka/Fira-Code/JetBrains-Mono fonts resolve via `@fontsource/*` pnpm packages declared in `frontend/package.json` and imported in `frontend/src/layouts/MainLayout.astro`.
- AC3 (project list IPC): `ProjectList.tsx` island calls `window.go.main.App.ListProjects()` via Wails IPC and renders the result list.
- AC4 (create dialog IPC): action item create dialog submits via `window.go.main.App.CreateActionItem(...)` Wails IPC call.
- AC5 (migration markers): every `frontend/src/components/*.tsx` file has `// MIGRATION TARGET: @hylla/stil-solid` as a `//`-style line comment at the top of the file before any imports; every `frontend/src/lib/vim/*.ts` file has `// MIGRATION TARGET: github.com/hylla-org/ro-vim` as a `//`-style line comment at the top of the file before any imports. A Vitest test in `frontend/tests/migration-markers.test.ts` walks both directories and asserts the marker substring.
- AC6 (vim engine merge): `palette.ts` implements ID-based deep merge; with `.tillsyn/bindings.json` present, command palette exposes 9 commands; with it absent, falls back to 4 baseline commands.
- AC7 (wails-keys filter macOS): `wails-keys.ts` intercepts macOS Cmd+Q, Cmd+M, Cmd+W, Cmd+H at document level and prevents them from reaching `engine.ts`. Linux/Windows OS-key filtering is deferred (FE-CROSS-PLATFORM-R1).
- AC8 (Vitest): `mage ci-fe` runs `pnpm run test:unit` in `frontend/` and it passes (vim engine + migration marker unit tests green).
- AC9 (pnpm build): `mage ci-fe` runs `pnpm run build` in `frontend/` and it exits 0.
- AC10 (main mage ci): `mage ci` remains green; `//go:build wails`-tagged root files (`main.go`, `app.go`) are skipped; `go test ./...` from tillsyn root WITHOUT `-tags wails` completes cleanly.

### ValidationPlan

- AC1: builder runs `wails dev` from tillsyn repo root (manual smoke; noted in BUILDER_WORKLOG).
- AC2: inspect `frontend/public/stil-tokens.css` contents for `--carl`; browser devtools in Playwright confirm Inter font resolves; `@fontsource/*` deps present in `frontend/package.json`.
- AC3–AC4: Playwright MCP `browser_snapshot` on project list page confirms project rows in accessibility tree; `browser_snapshot` on create dialog confirms form fields present.
- AC5: `mage ci-fe` executes the migration-marker Vitest test which walks `src/components/` + `src/lib/vim/`.
- AC6: Vitest test for `palette.ts` with both present and absent local file.
- AC7: Vitest test for `wails-keys.ts` using JSDOM key event dispatch with `metaKey: true`.
- AC8–AC9: `mage ci-fe` exit code 0 is the gate.
- AC10: builder runs `go test ./...` from tillsyn root WITHOUT `-tags wails`; records exit code + package set in BUILDER_WORKLOG; confirms no `//go:build wails`-tagged files appear in coverage output.

### RiskNotes

- R1 (Wails v2 dev mode): `//go:embed all:frontend/dist` in `main.go` is compile-time only. `wails dev` serves from the Astro dev server (`frontend:dev:serverUrl` in `wails.json`), NOT from the embed. `dist/` does not need to exist for `wails dev` acceptance. Builder must NOT add `wails build` as an acceptance target in v1.
- R2 (Wails at repo root — single module): `main.go` and `app.go` join the existing `github.com/evanmschultz/tillsyn` module. They import `github.com/evanmschultz/tillsyn/internal/app` directly (no replace directive needed). The `//go:build wails` tag is the ONLY isolation mechanism from default `mage ci` builds. Builder must confirm the tag is the FIRST line of each file (before the `package` declaration) and that `mage ci` pass-set is unchanged after D1 lands.
- R3 (Playwright via MCP, not CI): Playwright tests run via MCP `mcp__plugin_playwright_playwright__*` by QA agents — NOT in `mage ci-fe`. `mage ci-fe` runs only `pnpm run test:unit` (Vitest) + `pnpm run build`. Builder writes the Playwright test file; QA agents execute it.
- R4 (D4–D8 layout dependency): D3 creates `frontend/src/layouts/MainLayout.astro` as an empty stub; D4 fills it. D5–D8 are blocked_by D4 (serial chain via wails.ts file lock), so they all see the filled layout. D8 modifies it again (settings nav link) — safe because D8 is serialized after D4.
- R5 (Go test coverage for Wails root files): default `mage ci` excludes `main.go` + `app.go` via `//go:build wails`. Go tests for Wails IPC methods are out of scope for v1. Accepted risk; tracked as FE-GO-TEST-R1.

### ContextBlocks

```
[constraint severity=critical]
Wails files (main.go, app.go) at tillsyn repo root with //go:build wails. NEVER add -tags wails to mage ci.
mage ci-fe is the FE gate. mage ci is the main Go gate. They are SEPARATE targets.

[constraint severity=critical]
NEVER run mage install. Acceptance verification uses mage ci-fe and mage ci only.

[constraint severity=critical]
//go:build wails MUST be the first line of main.go and app.go (before the package declaration).
Omitting the build tag breaks the CI isolation guarantee — mage ci will try to compile Wails deps.

[constraint severity=high]
Every frontend/src/components/*.tsx file MUST carry: // MIGRATION TARGET: @hylla/stil-solid
as a //-style line comment at the top of the file before any imports.
Every frontend/src/lib/vim/*.ts file MUST carry: // MIGRATION TARGET: github.com/hylla-org/ro-vim
as a //-style line comment at the top of the file before any imports.
These are load-bearing audit markers. A Vitest test in D3 enforces them as a CI gate.

[constraint severity=high]
stil tokens consumed from stil/main/src/styles/tokens.css (the source path).
dist/tokens.css does NOT exist pre-build (stil's pnpm build:tokens produces dist/tokens.json).
The copy target in D3 copies src/styles/tokens.css → frontend/public/stil-tokens.css.

[constraint severity=high]
Fonts ship as @fontsource/* pnpm packages — they are NOT bundled in tokens.css.
D3 frontend/package.json MUST include @fontsource/inter, @fontsource/iosevka, @fontsource/fira-code, @fontsource/jetbrains-mono.
D4 MainLayout.astro MUST import them (e.g., import '@fontsource/inter').

[decision severity=normal]
Canonical Wails v2 layout at tillsyn repo root. Single shared go.mod. No fe/ subfolder.
No separate module. No replace directive. Verified via Context7 /wailsapp/wails docs.

[decision severity=normal]
mage ci-fe runs: (1) pnpm run test:unit in frontend/ (Vitest with --passWithNoTests); (2) pnpm run build in frontend/.
Playwright is MCP-only for QA agents; NOT in mage ci-fe. Avoids browser install in CI.

[decision severity=normal]
D3 creates frontend/src/layouts/MainLayout.astro as an empty stub. D4 fills it. D8 adds settings nav link.
Serialized via blocked_by chain — safe from concurrent edits.

[decision severity=normal]
baseline.json loaded via fetch('/stil-baseline.json') from frontend/public/ (file copied in D3).
.tillsyn/bindings.json loaded via fetch('/.tillsyn-bindings.json') from frontend/public/.tillsyn-bindings.json (D3 creates this as a symlink to ../../.tillsyn/bindings.json per R2 FF1-R2; pnpm build resolves into static dist/). NO vite-proxy (HTTP-to-HTTP only; cannot rewrite to filesystem paths per Context7 /vitejs/vite).
404 on either → graceful fallback. GetBindingsJSON IPC method NOT used (explicitly deferred).

[reference severity=normal]
stil baseline.json: /Users/evanschultz/Documents/Code/hylla/stil/main/src/bindings/baseline.json
product_extensions.tillsyn has 4 commands: new-drop, complete-drop, handoff, comment.

[reference severity=normal]
Local bindings extension: .tillsyn/bindings.json adds 5 commands: dispatch, plan, archive, settings, help.
ID-based deep merge → 9 commands total when local present; 4 when absent (graceful fallback).

[warning severity=high]
window.go.main.App.* is the Wails-generated IPC namespace. The struct bound in main.go is *App (from root).
Service methods are on *App (not *app.Service directly). App struct holds a *app.Service field; App methods delegate to service.

[warning severity=high]
D9 Playwright test scope: navigate to project list page, verify vim key dispatch (j/k navigation) in accessibility tree.
Do NOT scope to data rendering (that's D4's concern). D9 and D4 run in parallel; D4 data may not be present during D9 QA.

[warning severity=high]
//go:build wails must be placed on a line BY ITSELF before the package clause. The Go toolchain requires
the build constraint to be followed by a blank line before the package declaration. Omitting the blank
line makes the build tag a comment, not a constraint — the file will compile without -tags wails.

[note severity=normal]
pnpm is the package manager for frontend/. NEVER use npm. Every frontend/ command uses pnpm.

[note severity=normal]
wails-keys.ts is macOS-scoped in v1. Linux uses Ctrl instead of Cmd; Windows uses Alt+F4.
Cross-platform filter deferred to FE-CROSS-PLATFORM-R1 refinement.
```

### KindPayload

```json
{
  "children": [
    {"kind": "build", "title": "W6.D1 WAILS BOOTSTRAP (main.go + wails.json + .gitignore)", "blocked_by": []},
    {"kind": "build", "title": "W6.D2 GO SERVICE BINDINGS (app.go)", "blocked_by": ["W6.D1"]},
    {"kind": "build", "title": "W6.D3 ASTRO + SOLID SETUP + STIL TOKENS + FONTS + mage ci-fe (frontend/ scaffold + magefile.go)", "blocked_by": []},
    {"kind": "build", "title": "W6.D4 PROJECT LIST PAGE (projects.astro + ProjectList.tsx + MainLayout.astro filled + @fontsource imports)", "blocked_by": ["W6.D2", "W6.D3"]},
    {"kind": "build", "title": "W6.D5 PROJECT DETAIL + ACTION ITEM TREE (project-detail.astro + ActionItemTree.tsx)", "blocked_by": ["W6.D2", "W6.D3", "W6.D4"]},
    {"kind": "build", "title": "W6.D6 ACTION ITEM CREATE DIALOG (ActionItemCreateDialog.tsx)", "blocked_by": ["W6.D5"]},
    {"kind": "build", "title": "W6.D7 DISPATCHER TRIGGER + SPAWN OUTPUT VIEWER (DispatcherTrigger.tsx + SpawnOutputViewer.tsx)", "blocked_by": ["W6.D6"]},
    {"kind": "build", "title": "W6.D8 SETTINGS PANEL (settings.astro + SettingsPanel.tsx)", "blocked_by": ["W6.D7"]},
    {"kind": "build", "title": "W6.D9 VIM ENGINE (engine.ts + types.ts + wails-keys.ts + palette.ts + Vitest + Playwright)", "blocked_by": ["W6.D3"]}
  ]
}
```

### CompletionContract

**StartCriteria:**
- `main.go`, `app.go`, `wails.json`, `frontend/` do not exist at the tillsyn repo root (verified: all files NEW).
- `mage ci` is green on HEAD before any W6 work begins.
- `stil/main/src/styles/tokens.css` exists and is readable; `--carl` accent variable present.
- `stil/main/src/bindings/baseline.json` exists and `product_extensions.tillsyn` has exactly 4 commands.
- `.tillsyn/bindings.json` is authored in W8 wave — W8 is Wave A parallel. If not yet present when D9 builder runs, D9 graceful-fallback test covers the absent case.

**CompletionCriteria:**
- All 9 build droplets reach state `done`.
- `mage ci-fe` exits 0 (Vitest + pnpm build).
- `mage ci` exits 0 (main gate; Wails-tagged root files excluded via `//go:build wails`).
- `wails dev` from tillsyn repo root starts without errors (manual smoke noted in BUILDER_WORKLOG).
- Playwright MCP test navigates to project list page and verifies accessibility tree (QA agents run this).
- Every component file carries migration marker; every vim engine file carries migration marker (Vitest CI gate).
- Fonts resolve at runtime (Inter/Iosevka/Fira-Code/JetBrains-Mono via `@fontsource/*` pnpm deps).

**CompletionChecklist:**
- [ ] D1 done: main.go (//go:build wails + App struct stub + wails.Run) + wails.json authored; go test ./... without -tags wails still passes; .gitignore updated.
- [ ] D2 done: app.go (//go:build wails) authored; IPC methods delegate to internal/app.Service.
- [ ] D3 done: frontend/ scaffold complete; @fontsource/* deps in package.json; stil-tokens.css + stil-baseline.json copied; mage ci-fe target added; empty layout stub present; migration-marker Vitest test authored.
- [ ] D4 done: projects.astro + ProjectList.tsx + MainLayout.astro (filled + @fontsource imports); migration markers present.
- [ ] D5 done: project-detail.astro + ActionItemTree.tsx; migration markers present.
- [ ] D6 done: ActionItemCreateDialog.tsx; migration markers present.
- [ ] D7 done: DispatcherTrigger.tsx + SpawnOutputViewer.tsx; migration markers present.
- [ ] D8 done: settings.astro + SettingsPanel.tsx; MainLayout.astro settings nav link added; migration markers present.
- [ ] D9 done: engine.ts + types.ts + wails-keys.ts + palette.ts + Vitest tests + Playwright test; migration markers present; malformed-bindings fallback test present.
- [ ] mage ci-fe green.
- [ ] mage ci green.

---

## Planner

### Droplet Graph

```
D1 ──────────────────────────────────────────────────── (Wave A head, no blockers)
        │
        ▼
D2 (blocked_by D1)
   (app.go; same root module package + build-tag; serialize after D1)

D3 ──────────────────────────────────────────────────── (Wave A parallel, no blockers)
   (pure frontend/ files + magefile.go CiFe addition)

D4 (blocked_by D2 + D3) ─── D5 (blocked_by D4) ─── D6 (blocked_by D5) ─── D7 (blocked_by D6) ─── D8 (blocked_by D7)
    serial chain: D4–D8 all edit frontend/src/lib/wails.ts; concurrent dispatch would conflict

D9 (blocked_by D3) ──────────────────────────────────── (parallel to the D4–D8 chain; only edits frontend/src/lib/vim/)
```

Dispatch order:
1. D1 and D3 in parallel (Wave A entry).
2. D2 after D1 (root module; Wails build-tag isolation; D1 creates main.go with App struct; D2 adds app.go methods).
3. D4 after D2 + D3 (needs IPC types from D2; needs frontend dev environment from D3).
4. D5 after D4 (wails.ts file lock — D5 adds listActionItems to file D4 created).
5. D6 after D5 (wails.ts file lock — D6 adds createActionItem).
6. D7 after D6 (wails.ts file lock — D7 adds runDispatcher).
7. D8 after D7 (wails.ts file lock — D8 adds getAgentsConfig + getTemplateConfig; also modifies MainLayout.astro).
8. D9 after D3 only (vim engine writes to frontend/src/lib/vim/ only — parallel to D4–D8 chain; no IPC dependency).

---

### W6.D1 — WAILS BOOTSTRAP

- **State:** todo
- **Kind:** build
- **Irreducible:** true
- **Structural type:** droplet
- **Blocked by:** — (none; Wave A head)
- **Paths (all NEW unless noted):**
  - `main.go` (NEW — `//go:build wails` + App struct + wails.Run; at tillsyn repo root)
  - `wails.json` (NEW — Wails config; at tillsyn repo root)
  - `.gitignore` (EXISTING — add frontend/node_modules/, frontend/dist/, build/bin/, frontend/wailsjs/ to ignore rules)
- **Packages:** none new (root module; `//go:build wails` excludes from default compile; no new package)
- **Mage verification:** `mage ci` (main gate must stay green; build tag isolates wails files)

#### Specify

**Objective:** Create the minimal Wails v2 project skeleton at the tillsyn repo root that `wails dev` can read. `main.go` declares the `App` struct (holds a `*app.Service` field), `startup` lifecycle hook, and `wails.Run(...)` with default native menu (no custom items). `wails.json` configures the Astro dev server URL for dev mode (port 4321) and the frontend build command for production. Both files carry `//go:build wails` as the first non-comment line. `.gitignore` updated with Wails artifact patterns.

**AcceptanceCriteria:**
- `main.go` exists at tillsyn repo root with `//go:build wails` as the FIRST LINE (blank line, then `package main`).
- `main.go` compiles when `-tags wails` is passed (builder verifies manually: `go build -tags wails .` from tillsyn root — notes result in BUILDER_WORKLOG).
- `wails.json` is valid JSON with `name`, `outputfilename`, `frontend:dir`, `frontend:install`, `frontend:build`, `frontend:dev:watcher`, `frontend:dev:serverUrl` (set to `http://localhost:4321`) fields.
- `wails dev` from tillsyn repo root does not panic on startup (smoke; builder notes result in BUILDER_WORKLOG).
- `.gitignore` contains lines for `frontend/node_modules/`, `frontend/dist/`, `build/bin/`, `frontend/wailsjs/`.
- `mage ci` remains green (without `-tags wails`; build tag excludes main.go + app.go from default compile; `go test ./...` shows IDENTICAL package set as pre-D1). Builder records pre-D1 and post-D1 `mage ci` output in BUILDER_WORKLOG to confirm identical coverage pass-set.

**RiskNotes:**
- `//go:embed all:frontend/dist` in `main.go` is compile-time only. `wails dev` does NOT require `dist/` to exist. Builder must NOT attempt `wails build` as a verification step.
- `//go:build wails` MUST be the very first line of `main.go` (and `app.go` in D2), followed by a blank line, then `package main`. If the blank line is missing, Go treats the constraint as a comment and the file compiles unconditionally — breaking CI isolation.
- `App` struct in `main.go` is NEW. It holds a `*app.Service` field (type from `github.com/evanmschultz/tillsyn/internal/app`). Builder must read `internal/app/service.go` to confirm the correct `Service` type name before writing. **Symbol `App` is new, not yet in tree.**
- Wails native menu: v1 uses the DEFAULT Wails native menu only. No `Menu:` field in `options.App{}`. Wails provides Quit/About/Hide/Minimize automatically on macOS.
- `wails.json` `frontend:dev:serverUrl` must match Astro's dev server port. Astro default is `http://localhost:4321`. Builder sets this here; D3 sets the Astro port to match.
- Context7 `/wailsapp/wails` canonical layout confirmed: `go.mod` at project root, `main.go` at root, `wails.json` at root, `frontend/` at root. This drop follows that layout exactly.

**KindPayload (changes):**
```json
{
  "changes": [
    {"file": "main.go", "symbol": "App", "action": "add", "shape_hint": "//go:build wails\n\npackage main\n\nimport (\n  \"github.com/evanmschultz/tillsyn/internal/app\"\n  \"github.com/wailsapp/wails/v2\"\n  \"github.com/wailsapp/wails/v2/pkg/options\"\n)\n\ntype App struct { ctx context.Context; svc *app.Service }\nfunc NewApp(svc *app.Service) *App\nfunc (a *App) startup(ctx context.Context)\nfunc main() { ... wails.Run(&options.App{Bind: []interface{}{app}}) }"},
    {"file": "wails.json", "symbol": "wails config", "action": "add", "shape_hint": "{\"name\":\"Tillsyn\",\"outputfilename\":\"tillsyn\",\"frontend:dir\":\"frontend\",\"frontend:install\":\"pnpm install\",\"frontend:build\":\"pnpm run build\",\"frontend:dev:watcher\":\"pnpm run dev\",\"frontend:dev:serverUrl\":\"http://localhost:4321\"}"},
    {"file": ".gitignore", "symbol": "wails artifacts", "action": "modify", "shape_hint": "add: frontend/node_modules/, frontend/dist/, build/bin/, frontend/wailsjs/ to existing .gitignore"}
  ]
}
```

---

### W6.D2 — GO SERVICE BINDINGS

- **State:** todo
- **Kind:** build
- **Irreducible:** true
- **Structural type:** droplet
- **Blocked by:** W6.D1
- **Paths (all NEW, not yet in tree):**
  - `app.go` (NEW — `//go:build wails`; at tillsyn repo root)
- **Packages:** none new (root module, same package `main`; `//go:build wails` isolates; package-level build-lock with D1 → blocked_by D1)
- **Mage verification:** `mage ci` (main gate stays green; wails-tagged files excluded from default compile)

#### Specify

**Objective:** Author `app.go` — the `App` struct's IPC methods that Wails binds to `window.go.main.App.*`. Each method delegates to the `*app.Service` field set in D1's `startup` hook. Wails IPC contract: `ListProjects`, `GetProject`, `ListActionItems`, `CreateActionItem`, `RunDispatcher`, `GetAgentsConfig`, `GetTemplateConfig`.

**AcceptanceCriteria:**
- `app.go` exists at tillsyn repo root with `//go:build wails` as the FIRST LINE.
- `app.go` compiles when `-tags wails` is passed (part of the same `go build -tags wails .` smoke that D1 established; builder re-verifies after D2 to confirm no new errors).
- Exported methods on `App` match the IPC contract: `ListProjects() ([]domain.Project, error)`, `GetProject(id string) (domain.Project, error)`, `ListActionItems(projectID string) ([]domain.ActionItem, error)`, `CreateActionItem(req domain.CreateActionItemRequest) (domain.ActionItem, error)`, `RunDispatcher(actionItemID string) error` (delegates to `Service.RunDispatcher` OR returns wrapped `ErrNotImplemented` — both are acceptable v1 wiring; document which in BUILDER_WORKLOG), `GetAgentsConfig() (string, error)`, `GetTemplateConfig() (string, error)`.
- Every exported method has a Go doc comment starting with the method name.
- `mage ci` remains green (build tag unchanged; same package set as post-D1).

**RiskNotes:**
- Builder must read `internal/app/service.go` and `internal/domain/` to discover correct type names (`Project`, `ActionItem`, `CreateActionItemRequest`) before writing. **All symbols on `App` are new, not yet in tree.** Domain types are in `github.com/evanmschultz/tillsyn/internal/domain` — same module, no replace directive needed.
- `RunDispatcher`: read `internal/app/service.go` to check if a `RunDispatcher(ctx context.Context, actionItemID string) error` method exists. If it does, delegate. If not, stub as `return fmt.Errorf("dispatcher not yet wired: %w", ErrNotImplemented)` and mark TODO in BUILDER_WORKLOG. Either is acceptable v1 wiring.
- `GetAgentsConfig` + `GetTemplateConfig`: return TOML as raw string for the settings panel. Stub as reading `<cwd>/.tillsyn/agents.toml` or returning a placeholder string. Document the stub clearly.
- `GetBindingsJSON` IPC method is explicitly NOT in scope for D2. The vim engine (D9) loads bindings via HTTP fetch from `/.tillsyn-bindings.json`. Do not add `GetBindingsJSON` to `app.go`.
- Go test coverage for Wails root files deferred pre-MVP. Accepted risk logged as FE-GO-TEST-R1.

**KindPayload (changes):**
```json
{
  "changes": [
    {"file": "app.go", "symbol": "App.ListProjects", "action": "add", "shape_hint": "//go:build wails\n\npackage main\n\nfunc (a *App) ListProjects() ([]domain.Project, error) { return a.svc.ListProjects(a.ctx) }"},
    {"file": "app.go", "symbol": "App.GetProject", "action": "add", "shape_hint": "func (a *App) GetProject(id string) (domain.Project, error)"},
    {"file": "app.go", "symbol": "App.ListActionItems", "action": "add", "shape_hint": "func (a *App) ListActionItems(projectID string) ([]domain.ActionItem, error)"},
    {"file": "app.go", "symbol": "App.CreateActionItem", "action": "add", "shape_hint": "func (a *App) CreateActionItem(req domain.CreateActionItemRequest) (domain.ActionItem, error)"},
    {"file": "app.go", "symbol": "App.RunDispatcher", "action": "add", "shape_hint": "func (a *App) RunDispatcher(actionItemID string) error — delegates to Service.RunDispatcher OR returns ErrNotImplemented stub"},
    {"file": "app.go", "symbol": "App.GetAgentsConfig", "action": "add", "shape_hint": "func (a *App) GetAgentsConfig() (string, error)"},
    {"file": "app.go", "symbol": "App.GetTemplateConfig", "action": "add", "shape_hint": "func (a *App) GetTemplateConfig() (string, error)"}
  ]
}
```

---

### W6.D3 — ASTRO + SOLID DEV SETUP + STIL TOKENS + FONTS + mage ci-fe

- **State:** todo
- **Kind:** build
- **Irreducible:** true
- **Structural type:** droplet
- **Blocked by:** — (none; parallel to D1/D2; pure frontend files + magefile.go addition)
- **Paths (all NEW unless noted):**
  - `frontend/package.json` (NEW)
  - `frontend/astro.config.mjs` (NEW)
  - `frontend/tsconfig.json` (NEW)
  - `frontend/pnpm-lock.yaml` (NEW — generated by pnpm install)
  - `frontend/public/stil-tokens.css` (NEW — copied from `stil/main/src/styles/tokens.css`)
  - `frontend/public/stil-baseline.json` (NEW — copied from `stil/main/src/bindings/baseline.json`; consumed by D9 vim engine via `fetch('/stil-baseline.json')`)
  - `frontend/public/.tillsyn-bindings.json` (NEW — **symlink** to `../../.tillsyn/bindings.json` per R2 FF1-R2 absorption; consumed by D9 vim engine via `fetch('/.tillsyn-bindings.json')`. Astro dev follows symlink to serve live file; `pnpm build` resolves symlink and copies content into `dist/`. If symlink creation fails on Windows or the dev's filesystem, builder falls back to copy + documents the limitation.)
  - `frontend/src/layouts/MainLayout.astro` (NEW — EMPTY STUB with slot; D4 fills with nav + chrome + @fontsource imports)
  - `frontend/src/env.d.ts` (NEW — Astro type reference triple-slash directive)
  - `frontend/tests/migration-markers.test.ts` (NEW — Vitest test; walks src/components/ + src/lib/vim/ and asserts MIGRATION TARGET marker substring; green at D3 with empty dirs; remains green as D4–D9 populate files)
  - `magefile.go` (EXISTING — add `CiFe()` function only; all other content untouched)
- **Packages:** none (pure FE files + magefile addition; `magefile.go` is `//go:build mage`, separate from Go module)
- **Mage verification:** `mage ci-fe` (new target, added in this droplet); `mage ci` (existing gate must remain green)

#### Specify

**Objective:** Initialize the Astro + SolidJS frontend project at `frontend/`. Install Astro, `@astrojs/solid-js`, SolidJS, Vitest, `@playwright/test`, and the four `@fontsource/*` packages (Inter, Iosevka, Fira-Code, JetBrains-Mono). Copy stil tokens CSS and stil baseline.json to `frontend/public/`. Add `CiFe()` target to `magefile.go`. Create empty `MainLayout.astro` stub and `env.d.ts`. Add `migration-markers.test.ts` Vitest CI gate.

**AcceptanceCriteria:**
- `frontend/package.json` exists with: `name: "tillsyn-fe"`, `private: true`; `devDependencies` including `astro`, `@astrojs/solid-js`, `solid-js`, `vitest`, `@playwright/test`, `typescript`; `dependencies` (or `devDependencies`) including `@fontsource/inter`, `@fontsource/iosevka`, `@fontsource/fira-code`, `@fontsource/jetbrains-mono`; scripts include `dev`, `build`, `test:unit: "vitest run --passWithNoTests"`, `test:e2e`, `test`.
- `frontend/astro.config.mjs` imports `solidJs` from `@astrojs/solid-js` (canonical name per Context7); configures `integrations: [solidJs()]`; `output: 'static'`; server port 4321.
- `frontend/public/stil-tokens.css` is a verbatim copy of `stil/main/src/styles/tokens.css`; `:root { --carl: #dd9f57; ... }` present.
- `frontend/public/stil-baseline.json` is a verbatim copy of `stil/main/src/bindings/baseline.json`; `product_extensions.tillsyn.commands` has exactly 4 entries.
- `frontend/public/.tillsyn-bindings.json` exists as a symlink to `../../.tillsyn/bindings.json` (or copy if symlinks unsupported). `ls -la frontend/public/.tillsyn-bindings.json` shows the symlink target; `cat frontend/public/.tillsyn-bindings.json` resolves to the project-local bindings file content. R2 FF1-R2 absorption.
- `frontend/src/layouts/MainLayout.astro` exists (content: minimal `<!DOCTYPE html>` skeleton with `<link rel="stylesheet" href="/stil-tokens.css">` and `<slot />`; D4 extends it).
- `frontend/src/env.d.ts` exists with `/// <reference types="astro/client" />` (Astro triple-slash type reference).
- `frontend/tests/migration-markers.test.ts` exists; it imports `fs` + `path` (or `node:fs` + `node:path`); walks `src/components/` and `src/lib/vim/`; for each `.tsx` and `.ts` file asserts the appropriate MIGRATION TARGET marker substring is present. At D3 time the dirs are empty so no assertions fire — test passes with `--passWithNoTests`. As D4–D9 add files, assertions accumulate.
- `magefile.go` has new `CiFe()` exported function (added after existing `CI()` target; before helper funcs); runs `pnpm run test:unit` then `pnpm run build` in `frontend/`; short doc comment.
- `mage ci-fe` exits 0 (`vitest run --passWithNoTests` passes on empty test suite; `pnpm run build` exits 0).
- `mage ci` exits 0 (existing gate unchanged).

**RiskNotes:**
- `pnpm install` must be run inside `frontend/` to generate `pnpm-lock.yaml`. Builder runs `pnpm install` from `frontend/`. Note in BUILDER_WORKLOG.
- Astro 5.x uses `output: 'static'` by default. Wails embeds `frontend/dist` — so static output is correct.
- `mage ci-fe` shape: the `CiFe` function uses the same `runCommandInDir` or equivalent pattern as existing helpers. Builder reads existing `magefile.go` helpers (`runStage`, `runCommand`, etc.) before authoring. Target must change directory to `frontend/` before invoking pnpm.
- `vitest run --passWithNoTests` is PRE-BAKED in the `test:unit` script (not discovered at runtime). Vitest 1.x/2.x default exits non-zero when no tests found.
- The `@astrojs/solid-js` Astro integration canonical import name is `solidJs` per Context7 `/wailsapp/wails` + Context7 `/withastro/docs`. The shape_hint reflects this.
- **`magefile.go` is a SHARED file.** D3 is the ONLY W6 droplet touching `magefile.go`. D1 and D2 do NOT touch `magefile.go`.

**KindPayload (changes):**
```json
{
  "changes": [
    {"file": "frontend/package.json", "symbol": "pnpm project config", "action": "add", "shape_hint": "{name: tillsyn-fe, private: true, scripts: {dev: astro dev, build: astro build, 'test:unit': 'vitest run --passWithNoTests', 'test:e2e': 'playwright test', test: 'pnpm run test:unit && pnpm run test:e2e'}, devDependencies: {astro, '@astrojs/solid-js', 'solid-js', vitest, '@playwright/test', typescript, '@fontsource/inter', '@fontsource/iosevka', '@fontsource/fira-code', '@fontsource/jetbrains-mono'}}"},
    {"file": "frontend/astro.config.mjs", "symbol": "defineConfig", "action": "add", "shape_hint": "import solidJs from '@astrojs/solid-js'; export default defineConfig({integrations: [solidJs()], output: 'static', server: {port: 4321}})"},
    {"file": "frontend/public/stil-tokens.css", "symbol": "design tokens", "action": "add", "shape_hint": "verbatim copy of stil/main/src/styles/tokens.css"},
    {"file": "frontend/public/stil-baseline.json", "symbol": "baseline bindings", "action": "add", "shape_hint": "verbatim copy of stil/main/src/bindings/baseline.json; product_extensions.tillsyn.commands has 4 entries"},
    {"file": "frontend/public/.tillsyn-bindings.json", "symbol": "local bindings symlink", "action": "add", "shape_hint": "symlink to ../../.tillsyn/bindings.json per R2 FF1-R2; created via `ln -sf ../../.tillsyn/bindings.json frontend/public/.tillsyn-bindings.json`; Astro dev follows symlink (live); pnpm build resolves + copies into dist/ (static snapshot for Wails-embedded production); fallback to copy if filesystem doesn't support symlinks"},
    {"file": "frontend/src/layouts/MainLayout.astro", "symbol": "MainLayout", "action": "add", "shape_hint": "<!DOCTYPE html><html><head><link rel=stylesheet href=/stil-tokens.css></head><body><slot /></body></html> — empty stub; D4 fills with nav chrome + @fontsource imports"},
    {"file": "frontend/src/env.d.ts", "symbol": "Astro types", "action": "add", "shape_hint": "/// <reference types=\"astro/client\" />"},
    {"file": "frontend/tests/migration-markers.test.ts", "symbol": "migration marker CI gate", "action": "add", "shape_hint": "import {readdirSync, readFileSync} from 'node:fs'; import {join} from 'node:path'; describe('migration markers', () => { it('components have @hylla/stil-solid marker', ...); it('vim files have ro-vim marker', ...); }) — walks src/components/ and src/lib/vim/; asserts marker substring; passes with --passWithNoTests when dirs are empty"},
    {"file": "magefile.go", "symbol": "CiFe", "action": "add", "shape_hint": "// CiFe runs the FE CI gate: pnpm test:unit + pnpm build in frontend/.\nfunc CiFe() error { ... run pnpm run test:unit in frontend/; run pnpm run build in frontend/ ... }"}
  ]
}
```

---

### W6.D4 — PROJECT LIST PAGE

- **State:** todo
- **Kind:** build
- **Irreducible:** true
- **Structural type:** droplet
- **Blocked by:** W6.D2, W6.D3
- **Paths (all NEW unless noted):**
  - `frontend/src/pages/projects.astro` (NEW)
  - `frontend/src/components/ProjectList.tsx` (NEW)
  - `frontend/src/lib/wails.ts` (NEW — Wails IPC type wrappers; shared by D5–D8 via serial chain)
  - `frontend/src/layouts/MainLayout.astro` (EXISTING after D3 stub; D4 fills with nav + chrome + `@fontsource/*` imports)
- **Packages:** none (pure FE; no Go package)
- **Mage verification:** `mage ci-fe` (Vitest + pnpm build)

#### Specify

**Objective:** Implement the project list page. `projects.astro` renders using `MainLayout` and mounts the `ProjectList` SolidJS island. `ProjectList.tsx` calls `window.go.main.App.ListProjects()` via Wails IPC on mount, renders rows. `wails.ts` provides typed wrappers for each IPC call. Fills `MainLayout.astro` with styled nav chrome + `<slot />` AND `@fontsource/*` import statements (so Inter/Iosevka/etc. load on every page).

**AcceptanceCriteria:**
- `ProjectList.tsx` first line is `// MIGRATION TARGET: @hylla/stil-solid` (`//`-style comment; before any imports).
- `wails.ts` first line is `// MIGRATION TARGET: @hylla/stil-solid`.
- `projects.astro` imports `ProjectList` from components and `MainLayout` from layouts; mounts `<ProjectList client:load />` inside `<MainLayout>`.
- `MainLayout.astro` imports all four `@fontsource/*` packages (e.g. `import '@fontsource/inter'; import '@fontsource/iosevka'; import '@fontsource/fira-code'; import '@fontsource/jetbrains-mono';`) in the `<style>` block or as a side-effect import in the frontmatter.
- `MainLayout.astro` has styled nav header (app name + keyboard shortcut hint); `<slot />` for page content; `<link rel="stylesheet" href="/stil-tokens.css">` in `<head>`.
- Vitest: at least one unit test for `ProjectList.tsx` (mock `window.go.main.App.ListProjects`, assert rows render).
- `mage ci-fe` exits 0 (migration-markers test green because `ProjectList.tsx` + `wails.ts` carry the marker).
- Playwright (MCP): `browser_snapshot` on the projects page URL shows a list container in accessibility tree.

**RiskNotes:**
- Wails generates `frontend/wailsjs/go/main/App.js` + `frontend/wailsjs/runtime/` during `wails dev`. `window.go.main.App.*` is available ONLY after `wails dev` runs once and generates bindings. For unit testing (Vitest + JSDOM), IPC calls must be mocked. Builder provides a mock for `window.go.main.App.ListProjects` in the Vitest setup file (e.g., `frontend/src/test-setup.ts`).
- `wails.ts` wrapper functions handle the Wails IPC promise shape: `window.go.main.App.ListProjects()` returns `Promise<Project[]>`. Wrappers re-export with TypeScript types.
- `MainLayout.astro` is EXISTING after D3 creates the stub. D4 edits it (adds nav + styles + font imports). Since D4 is blocked_by D3 (which creates the file), this is safe.
- Migration marker in `wails.ts`: `wails.ts` is a Tillsyn FE IPC helper — migration target is `@hylla/stil-solid` (same as components), not `ro-vim`.

**KindPayload (changes):**
```json
{
  "changes": [
    {"file": "frontend/src/pages/projects.astro", "symbol": "ProjectsPage", "action": "add", "shape_hint": "---\nimport ProjectList from '../components/ProjectList.tsx';\nimport MainLayout from '../layouts/MainLayout.astro';\n---\n<MainLayout><ProjectList client:load /></MainLayout>"},
    {"file": "frontend/src/components/ProjectList.tsx", "symbol": "ProjectList", "action": "add", "shape_hint": "// MIGRATION TARGET: @hylla/stil-solid\nimport { createSignal, onMount } from 'solid-js';\nexport default function ProjectList() { const [projects, setProjects] = createSignal([]); onMount(async () => { setProjects(await listProjects()); }); return <ul>...</ul>; }"},
    {"file": "frontend/src/lib/wails.ts", "symbol": "listProjects", "action": "add", "shape_hint": "// MIGRATION TARGET: @hylla/stil-solid\nexport async function listProjects(): Promise<Project[]> { return (window as any).go.main.App.ListProjects(); }"},
    {"file": "frontend/src/layouts/MainLayout.astro", "symbol": "MainLayout", "action": "modify", "shape_hint": "add nav header with app name + vim shortcut hints; @fontsource/* imports (import '@fontsource/inter'; import '@fontsource/iosevka'; import '@fontsource/fira-code'; import '@fontsource/jetbrains-mono'); slot for content; links stil-tokens.css in head"}
  ]
}
```

---

### W6.D5 — PROJECT DETAIL + ACTION ITEM TREE

- **State:** todo
- **Kind:** build
- **Irreducible:** true
- **Structural type:** droplet
- **Blocked by:** W6.D2, W6.D3, W6.D4
- **Paths (all NEW, not yet in tree):**
  - `frontend/src/pages/project-detail.astro` (NEW)
  - `frontend/src/components/ActionItemTree.tsx` (NEW)
- **Also modifies (shared file — hence blocked_by D4):**
  - `frontend/src/lib/wails.ts` (EXISTING after D4; D5 adds `listActionItems` wrapper)
- **Packages:** none (pure FE)
- **Mage verification:** `mage ci-fe`

#### Specify

**Objective:** Implement the project detail page with a two-pane layout: left pane = collapsible action item tree (SolidJS island), right pane = action item detail view (initially empty until an item is focused). `ActionItemTree.tsx` calls `window.go.main.App.ListActionItems(projectID)` via the `wails.ts` wrapper (D5 adds the wrapper; D4 created the file). Project ID passed as Astro page URL param.

**AcceptanceCriteria:**
- `ActionItemTree.tsx` first line is `// MIGRATION TARGET: @hylla/stil-solid`.
- Two-pane layout is present in `project-detail.astro` (left + right pane structure via CSS grid or flexbox).
- `ActionItemTree.tsx` renders a tree from flat `ActionItem[]` (parent/child by `parent_id` or equivalent field — builder reads `internal/domain/action_item.go` to confirm).
- `wails.ts` has `export async function listActionItems(projectID: string): Promise<ActionItem[]>` added.
- Vitest: at least one unit test for tree flattening / rendering logic.
- `mage ci-fe` exits 0.
- Playwright (MCP): `browser_snapshot` on project detail URL shows tree container in accessibility tree.

**RiskNotes:**
- `ActionItem` domain type has a `parent_id` field (or similar). Builder reads `internal/domain/action_item.go` to confirm the exact JSON-serialized field name. Wails marshals Go struct fields by JSON tag; builder confirms JSON tags before writing the TS interface.
- D5 adds `listActionItems` to `wails.ts` (file created by D4). D5 is blocked_by D4 to serialize this edit. The file-level lock is correct — no concurrent edit risk.
- TypeScript `ActionItem` interface: manually defined in `wails.ts` matching the Go struct's exported fields (Wails generates this in `wailsjs/go/main/` during dev; for Vitest the builder defines the interface manually).

**KindPayload (changes):**
```json
{
  "changes": [
    {"file": "frontend/src/pages/project-detail.astro", "symbol": "ProjectDetailPage", "action": "add", "shape_hint": "two-pane layout: left=ActionItemTree island, right=detail pane (empty slot)"},
    {"file": "frontend/src/components/ActionItemTree.tsx", "symbol": "ActionItemTree", "action": "add", "shape_hint": "// MIGRATION TARGET: @hylla/stil-solid\nCollapsible tree rendering ActionItem[]; calls listActionItems(projectID)"},
    {"file": "frontend/src/lib/wails.ts", "symbol": "listActionItems", "action": "modify", "shape_hint": "add: export async function listActionItems(projectID: string): Promise<ActionItem[]> { return (window as any).go.main.App.ListActionItems(projectID); }"}
  ]
}
```

---

### W6.D6 — ACTION ITEM CREATE DIALOG

- **State:** todo
- **Kind:** build
- **Irreducible:** true
- **Structural type:** droplet
- **Blocked by:** W6.D5
- **Paths (all NEW, not yet in tree):**
  - `frontend/src/components/ActionItemCreateDialog.tsx` (NEW)
- **Also modifies (shared file — hence chained after D5):**
  - `frontend/src/lib/wails.ts` (EXISTING after D5; D6 adds `createActionItem` wrapper)
- **Packages:** none (pure FE)
- **Mage verification:** `mage ci-fe`

#### Specify

**Objective:** Implement the action item create dialog as a SolidJS modal component. Fields: kind picker (dropdown), title input, paths input (comma-separated), description textarea. Submits via `window.go.main.App.CreateActionItem(...)` through the `wails.ts` wrapper.

**AcceptanceCriteria:**
- `ActionItemCreateDialog.tsx` first line is `// MIGRATION TARGET: @hylla/stil-solid`.
- Form has kind picker, title, paths, description fields.
- Submit calls `createActionItem(req)` from `wails.ts`.
- `wails.ts` has `export async function createActionItem(req: CreateActionItemRequest): Promise<ActionItem>` added.
- Vitest: test for form validation (title required, kind required).
- `mage ci-fe` exits 0.
- Playwright (MCP): `browser_snapshot` shows dialog form fields in accessibility tree when dialog is open.

**RiskNotes:**
- `CreateActionItemRequest` Go struct: builder reads `internal/domain/` to discover exported fields + JSON tags. Common fields: `project_id`, `kind`, `title`, `description`, `paths`, `packages`. Map JSON tags → TypeScript camelCase interface fields.
- D6 chained after D5 (which is after D4) to serialize `wails.ts` edits. No concurrent file access.

**KindPayload (changes):**
```json
{
  "changes": [
    {"file": "frontend/src/components/ActionItemCreateDialog.tsx", "symbol": "ActionItemCreateDialog", "action": "add", "shape_hint": "// MIGRATION TARGET: @hylla/stil-solid\nModal dialog; kind picker + title + paths + description; submit calls createActionItem(req)"},
    {"file": "frontend/src/lib/wails.ts", "symbol": "createActionItem", "action": "modify", "shape_hint": "add: export async function createActionItem(req: CreateActionItemRequest): Promise<ActionItem> { return (window as any).go.main.App.CreateActionItem(req); }"}
  ]
}
```

---

### W6.D7 — DISPATCHER TRIGGER + SPAWN OUTPUT VIEWER

- **State:** todo
- **Kind:** build
- **Irreducible:** true
- **Structural type:** droplet
- **Blocked by:** W6.D6
- **Paths (all NEW, not yet in tree):**
  - `frontend/src/components/DispatcherTrigger.tsx` (NEW)
  - `frontend/src/components/SpawnOutputViewer.tsx` (NEW)
- **Also modifies (shared file — hence chained after D6):**
  - `frontend/src/lib/wails.ts` (EXISTING after D6; D7 adds `runDispatcher` wrapper)
- **Packages:** none (pure FE)
- **Mage verification:** `mage ci-fe`

#### Specify

**Objective:** Implement a "Run Dispatcher" button (`DispatcherTrigger.tsx`) that calls `window.go.main.App.RunDispatcher(actionItemID)` via `wails.ts` wrapper. `SpawnOutputViewer.tsx` displays spawn status — for v1, uses a simple status indicator (idle/running/done/error) via SolidJS reactive state. Full live-tail via Wails `runtime.EventsOn` is a future enhancement (FE-LIVE-TAIL-R1). `wails.ts` gets `runDispatcher` wrapper.

**AcceptanceCriteria:**
- `DispatcherTrigger.tsx` first line is `// MIGRATION TARGET: @hylla/stil-solid`.
- `SpawnOutputViewer.tsx` first line is `// MIGRATION TARGET: @hylla/stil-solid`.
- `DispatcherTrigger.tsx` accepts `actionItemID: string` prop; on click calls `runDispatcher(actionItemID)`.
- `SpawnOutputViewer.tsx` shows status indicator (idle/running/done/error) using SolidJS `createSignal`.
- `wails.ts` has `export async function runDispatcher(id: string): Promise<void>` added.
- Vitest: test for button click firing IPC call (mock).
- `mage ci-fe` exits 0.

**RiskNotes:**
- Live spawn output tailing via `runtime.EventsOn` is a future enhancement (FE-LIVE-TAIL-R1). v1 uses simple status state.
- `RunDispatcher` in `app.go` (D2) may return an error if dispatcher is not fully wired. v1 behavior: show error status to user if call fails.
- D7 chained after D6 to serialize `wails.ts` edits.

**KindPayload (changes):**
```json
{
  "changes": [
    {"file": "frontend/src/components/DispatcherTrigger.tsx", "symbol": "DispatcherTrigger", "action": "add", "shape_hint": "// MIGRATION TARGET: @hylla/stil-solid\n<button onClick={async () => { await runDispatcher(props.actionItemID); }}>Run</button>"},
    {"file": "frontend/src/components/SpawnOutputViewer.tsx", "symbol": "SpawnOutputViewer", "action": "add", "shape_hint": "// MIGRATION TARGET: @hylla/stil-solid\nStatus indicator: idle/running/done/error; SolidJS createSignal"},
    {"file": "frontend/src/lib/wails.ts", "symbol": "runDispatcher", "action": "modify", "shape_hint": "add: export async function runDispatcher(id: string): Promise<void> { return (window as any).go.main.App.RunDispatcher(id); }"}
  ]
}
```

---

### W6.D8 — SETTINGS PANEL

- **State:** todo
- **Kind:** build
- **Irreducible:** true
- **Structural type:** droplet
- **Blocked by:** W6.D7
- **Paths (all NEW unless noted):**
  - `frontend/src/pages/settings.astro` (NEW)
  - `frontend/src/components/SettingsPanel.tsx` (NEW)
  - `frontend/src/layouts/MainLayout.astro` (EXISTING — D8 adds settings nav link)
- **Also modifies (shared file — hence chained after D7):**
  - `frontend/src/lib/wails.ts` (EXISTING after D7; D8 adds `getAgentsConfig` + `getTemplateConfig` wrappers)
- **Packages:** none (pure FE)
- **Mage verification:** `mage ci-fe`

#### Specify

**Objective:** Implement the settings page. `settings.astro` mounts `SettingsPanel` island. `SettingsPanel.tsx` shows read-only views of `agents.toml` content and `template.toml` content (fetched via `getAgentsConfig()` + `getTemplateConfig()` IPC wrappers from `wails.ts`). Groups management is a future enhancement; v1 is view-only. D8 also adds the settings nav link to `MainLayout.astro` (D4 created it; D8 is serialized after D4–D7 so this is safe).

**AcceptanceCriteria:**
- `SettingsPanel.tsx` first line is `// MIGRATION TARGET: @hylla/stil-solid`.
- `SettingsPanel.tsx` calls `getAgentsConfig()` and `getTemplateConfig()` on mount; renders results in `<pre>` blocks.
- `wails.ts` has `export async function getAgentsConfig(): Promise<string>` and `export async function getTemplateConfig(): Promise<string>` added.
- `frontend/src/layouts/MainLayout.astro` has a settings nav link (e.g., `<a href="/settings">Settings</a>` in the nav header).
- Vitest: test for settings panel rendering with mocked config strings.
- `mage ci-fe` exits 0.
- Playwright (MCP): `browser_snapshot` on settings URL shows config content areas in accessibility tree.

**RiskNotes:**
- `getAgentsConfig()` and `getTemplateConfig()` in `app.go` (D2) return raw TOML strings. v1 renders them verbatim in `<pre>` blocks. No parsing, no editing.
- D8 modifies `MainLayout.astro` (created by D3, filled by D4). Since D8 is serialized after D4–D7 via the blocked_by chain, there is no concurrent edit risk.
- D8 chained after D7 to serialize `wails.ts` edits.

**KindPayload (changes):**
```json
{
  "changes": [
    {"file": "frontend/src/pages/settings.astro", "symbol": "SettingsPage", "action": "add", "shape_hint": "import SettingsPanel; import MainLayout; <MainLayout><SettingsPanel client:load /></MainLayout>"},
    {"file": "frontend/src/components/SettingsPanel.tsx", "symbol": "SettingsPanel", "action": "add", "shape_hint": "// MIGRATION TARGET: @hylla/stil-solid\nFetches agents.toml + template.toml; renders in <pre> blocks"},
    {"file": "frontend/src/lib/wails.ts", "symbol": "getAgentsConfig + getTemplateConfig", "action": "modify", "shape_hint": "add: export async function getAgentsConfig(): Promise<string>; export async function getTemplateConfig(): Promise<string>"},
    {"file": "frontend/src/layouts/MainLayout.astro", "symbol": "settings nav link", "action": "modify", "shape_hint": "add <a href='/settings'>Settings</a> to nav header"}
  ]
}
```

---

### W6.D9 — VIM ENGINE

- **State:** todo
- **Kind:** build
- **Irreducible:** true
- **Structural type:** droplet
- **Blocked by:** W6.D3
- **Paths (all NEW, not yet in tree):**
  - `frontend/src/lib/vim/types.ts` (NEW)
  - `frontend/src/lib/vim/engine.ts` (NEW)
  - `frontend/src/lib/vim/wails-keys.ts` (NEW)
  - `frontend/src/lib/vim/palette.ts` (NEW)
  - `frontend/src/lib/vim/engine.test.ts` (NEW — co-located Vitest unit tests)
  - `frontend/src/lib/vim/wails-keys.test.ts` (NEW — co-located Vitest unit tests)
  - `frontend/src/lib/vim/palette.test.ts` (NEW — co-located Vitest unit tests)
  - `frontend/tests/vim-keybind.spec.ts` (NEW — Playwright e2e test; authored, run via MCP by QA agents)
- **Packages:** none (pure FE + `frontend/src/lib/vim/` module)
- **Mage verification:** `mage ci-fe` (Vitest unit tests for engine + wails-keys + palette; migration-markers test also covers vim/*.ts files)

#### Specify

**Objective:** Implement the TypeScript vim keybinding engine at `frontend/src/lib/vim/`. `types.ts` defines binding/mode/dispatch types. `engine.ts` implements the mode state machine consuming stil baseline (via `fetch('/stil-baseline.json')`) + `.tillsyn/bindings.json` extension (via `fetch('/.tillsyn-bindings.json')`) at startup; dispatches key events. `wails-keys.ts` runs at document level, filters macOS OS-level keys (Cmd+Q/M/W/H), passes everything else to `engine.ts`. `palette.ts` implements the `:` command palette backed by ID-based deep merge of baseline's 4 `product_extensions.tillsyn.commands` + local's 5 additions = 9 total (graceful fallback to 4 when local absent). Vitest unit tests for all three. Playwright test (authored; executed by QA agents via MCP).

**Evidence for command counts (direct file read):**
- `stil/main/src/bindings/baseline.json` `product_extensions.tillsyn.commands` (confirmed): 4 commands — `new-drop`, `complete-drop`, `handoff`, `comment`.
- `.tillsyn/bindings.json` (authored in W8; graceful fallback if absent): 5 commands — `dispatch`, `plan`, `archive`, `settings`, `help`.
- Total with merge: 9. Without local file: 4.

**AcceptanceCriteria:**
- All 4 vim engine files have `// MIGRATION TARGET: github.com/hylla-org/ro-vim` as a `//`-style line comment at the TOP OF THE FILE before any imports. (Migration-markers Vitest test from D3 enforces this automatically.)
- `types.ts` exports: `VimMode` (`'nav' | 'insert' | 'visual' | 'visual-block' | 'command' | 'hint'`), `Binding` interface, `Command` interface, `EngineState` interface, `DispatchHandler` type.
- `engine.ts` exports `VimEngine` class: `constructor(baseline: Command[], local?: Command[])`, `handleKey(event: KeyboardEvent): void`, `getMode(): VimMode`, `onDispatch(handler: DispatchHandler): void`.
- `wails-keys.ts` exports `installWailsKeyFilter(engine: VimEngine): () => void` — installs document-level keydown listener, blocks macOS Cmd+Q/Cmd+M/Cmd+W/Cmd+H (via `event.metaKey` + key check), passes rest to `engine.handleKey(event)`. Returns a cleanup function. Linux/Windows OS-key filtering is out of scope for v1 (FE-CROSS-PLATFORM-R1).
- `palette.ts` exports `CommandPalette` class: `constructor(baseline: Command[], local?: Command[])`, `mergeCommands(baseline: Command[], local: Command[]): Command[]` (local wins on ID collision), `getCommands(): Command[]` (returns 9 or 4), `execute(id: string, ...args: string[]): Promise<void>`.
- Vitest (`engine.test.ts`): `j` in nav mode dispatches `next-item`; `Esc` in insert mode returns to nav mode; unknown key in nav mode is silently ignored.
- Vitest (`wails-keys.test.ts`): `Cmd+Q` keydown (with `metaKey: true`) is NOT passed to engine; plain `j` keydown IS passed to engine. JSDOM supports `metaKey` in `KeyboardEvent` constructor — builder confirms.
- Vitest (`palette.test.ts`): with local commands present, `getCommands()` returns 9 unique IDs; with local absent, returns 4; local wins on ID collision (`mergeCommands` test).
- Vitest (`palette.test.ts`): with malformed local bindings JSON (invalid UTF-8 / truncated): `getCommands()` returns 4 baseline commands; parse error is logged to console (not thrown/fail-loud).
- Playwright (`vim-keybind.spec.ts`): navigate to `http://localhost:4321`; install the engine; dispatch `j` key; verify `getMode()` remains `nav` and next-item handler fired. Uses `browser_navigate` + `browser_snapshot` via MCP. (Authored; run by QA agents.)
- `mage ci-fe` exits 0 (all Vitest tests pass).

**RiskNotes:**
- **Baseline loading (ONE chosen path):** `engine.ts` fetches baseline at startup: `await fetch('/stil-baseline.json')` (file is in `frontend/public/`, copied by D3 — D3 must complete before D9 runs, ensured by D9 `blocked_by D3`). Parses JSON, extracts `product_extensions.tillsyn.commands` (4 entries). If fetch fails: fallback to an empty baseline-only mode (log error to console). Alternative paths (direct path traversal, Wails IPC `GetBindingsJSON`) are explicitly rejected: path traversal is impractical in browser context; IPC `GetBindingsJSON` adds cross-droplet scope to D2 unnecessarily. Use `fetch('/stil-baseline.json')` only.
- **Local bindings loading (ONE chosen path — REVISED per R2 fals FF1-R2):** `palette.ts` fetches local extension: `await fetch('/.tillsyn-bindings.json')`. If 404: graceful fallback to baseline-only (4 commands). If present + valid: ID-merge with baseline. If present + malformed: log parse error to console + fall back to baseline-only (NOT fail-loud). **Mechanism**: D3 creates `frontend/public/.tillsyn-bindings.json` as a symlink to `../../.tillsyn/bindings.json` (D3 paths + KindPayload). Astro dev server follows the symlink to serve the live project-local file; `pnpm build` resolves the symlink and copies the file content into `frontend/dist/`. This works in BOTH `wails dev` (symlink-served, live updates visible after page refresh) AND `wails build` (static snapshot embedded in production binary). **Limitation (v1, accepted)**: production binary captures `.tillsyn/bindings.json` at build time; runtime edits to that file require a rebuild. v2 may reintroduce a Wails IPC `GetBindingsJSON` method for live-reload (refinement raised separately). **NO vite-server-proxy** — original round-2 design used `vite.server.proxy` which is HTTP-to-HTTP only per Context7 `/vitejs/vite` and cannot rewrite to filesystem paths (R2 fals FF1-R2). `frontend/astro.config.mjs` does NOT require a proxy entry; D9 does NOT modify `astro.config.mjs`.
- **`window.go.main.App.*` NOT used by D9.** The vim engine is a pure TypeScript module with no Wails IPC dependency.
- `wails-keys.ts` test environment (Vitest + JSDOM): `new KeyboardEvent('keydown', {key: 'q', metaKey: true})` works in JSDOM. Builder confirms via code.
- D9 is parallel with D4–D8 (D9 blocked_by D3 only; D4–D8 are serialized separately). No conflict: D9 writes ONLY to `frontend/src/lib/vim/`; D4–D8 write to `frontend/src/components/`, `frontend/src/pages/`, and `frontend/src/lib/wails.ts`. **REVISED (R2 fals FF1-R2)**: D9 no longer amends `frontend/astro.config.mjs` — the symlink approach for `.tillsyn-bindings.json` makes the proxy unnecessary. D3 creates the symlink alongside other public-dir setup; D9 just fetches.

**D9 paths (R2 FF1-R2 revised — no `astro.config.mjs` amendment; symlink lives in D3 scope):**
- `frontend/src/lib/vim/types.ts` (NEW)
- `frontend/src/lib/vim/engine.ts` (NEW)
- `frontend/src/lib/vim/wails-keys.ts` (NEW)
- `frontend/src/lib/vim/palette.ts` (NEW)
- `frontend/src/lib/vim/engine.test.ts` (NEW)
- `frontend/src/lib/vim/wails-keys.test.ts` (NEW)
- `frontend/src/lib/vim/palette.test.ts` (NEW)
- `frontend/tests/vim-keybind.spec.ts` (NEW — Playwright; run via MCP by QA agents)

**KindPayload (changes):**
```json
{
  "changes": [
    {"file": "frontend/src/lib/vim/types.ts", "symbol": "VimMode + Binding + Command + EngineState + DispatchHandler", "action": "add", "shape_hint": "// MIGRATION TARGET: github.com/hylla-org/ro-vim\nexport type VimMode = 'nav'|'insert'|'visual'|'visual-block'|'command'|'hint';\nexport interface Binding {...}; export interface Command { id: string; ... }"},
    {"file": "frontend/src/lib/vim/engine.ts", "symbol": "VimEngine", "action": "add", "shape_hint": "// MIGRATION TARGET: github.com/hylla-org/ro-vim\nexport class VimEngine { constructor(baseline: Command[], local?: Command[]); handleKey(e: KeyboardEvent): void; getMode(): VimMode; onDispatch(h: DispatchHandler): void }"},
    {"file": "frontend/src/lib/vim/wails-keys.ts", "symbol": "installWailsKeyFilter", "action": "add", "shape_hint": "// MIGRATION TARGET: github.com/hylla-org/ro-vim\nexport function installWailsKeyFilter(engine: VimEngine): () => void — macOS Cmd+Q/M/W/H blocked (event.metaKey + key); rest → engine.handleKey; returns cleanup fn"},
    {"file": "frontend/src/lib/vim/palette.ts", "symbol": "CommandPalette", "action": "add", "shape_hint": "// MIGRATION TARGET: github.com/hylla-org/ro-vim\nexport class CommandPalette { constructor(baseline: Command[], local?: Command[]); mergeCommands(b: Command[], l: Command[]): Command[]; getCommands(): Command[]; execute(id: string, ...args: string[]): Promise<void> }"},
    {"file": "frontend/src/lib/vim/engine.test.ts", "symbol": "VimEngine tests", "action": "add", "shape_hint": "j→next-item dispatch; Esc in insert→nav; unknown key silently ignored"},
    {"file": "frontend/src/lib/vim/wails-keys.test.ts", "symbol": "wailsKeyFilter tests", "action": "add", "shape_hint": "Cmd+Q (metaKey:true, key:q) blocked; j (no metaKey) passed through"},
    {"file": "frontend/src/lib/vim/palette.test.ts", "symbol": "CommandPalette tests", "action": "add", "shape_hint": "9 commands with local; 4 without; local wins on ID collision; malformed local JSON → log + fallback to 4"},
    {"file": "frontend/tests/vim-keybind.spec.ts", "symbol": "playwright vim test", "action": "add", "shape_hint": "// Playwright: browser_navigate localhost:4321; dispatch j key; verify nav mode + next-item fired; run via MCP browser_snapshot"}
  ]
}
```

---

## _BLOCKERS.toml Cross-Reference

(Mirrors PLAN.md inline `Blocked by:` bullets — PLAN.md is truth.)

```toml
# _BLOCKERS.toml — DROP_4c.6.1.W6_FE_SCAFFOLD/
# Immediate-children sibling blocker ledger.
# Mirrors inline Blocked by: bullets in PLAN.md; PLAN.md is truth.

[[blockers]]
node = "W6.D2"
blocked_by = ["W6.D1"]
reason = "D2 (app.go) is in the same root Go module as D1 (main.go); both carry //go:build wails; package-level compile lock requires serialization"

[[blockers]]
node = "W6.D4"
blocked_by = ["W6.D2", "W6.D3"]
reason = "D4 needs IPC method signatures from D2 (wails.ts wrapper types) and frontend dev environment + pnpm setup from D3"

[[blockers]]
node = "W6.D5"
blocked_by = ["W6.D2", "W6.D3", "W6.D4"]
reason = "D5 adds listActionItems to wails.ts (file created by D4); file-level conflict on frontend/src/lib/wails.ts — serialized after D4"

[[blockers]]
node = "W6.D6"
blocked_by = ["W6.D5"]
reason = "D6 adds createActionItem to wails.ts; serialized after D5 to avoid concurrent wails.ts edits"

[[blockers]]
node = "W6.D7"
blocked_by = ["W6.D6"]
reason = "D7 adds runDispatcher to wails.ts; serialized after D6 to avoid concurrent wails.ts edits"

[[blockers]]
node = "W6.D8"
blocked_by = ["W6.D7"]
reason = "D8 adds getAgentsConfig + getTemplateConfig to wails.ts and adds settings nav link to MainLayout.astro (created by D4); serialized after D7"

[[blockers]]
node = "W6.D9"
blocked_by = ["W6.D3"]
reason = "D9 (vim engine) requires Vitest config + frontend dev environment from D3; baseline.json in frontend/public/ from D3; no IPC dependency; parallel to D4-D8 chain"
```

---

## Dispatch Schedule

```
Parallel batch 1 (Wave A entry):
  W6.D1  (Wails bootstrap — main.go + wails.json + .gitignore)
  W6.D3  (Astro + Solid + fonts + stil + mage ci-fe)

Sequential after D1:
  W6.D2  (Go service bindings — app.go; blocked_by D1)

Parallel batch 2 (after D2 + D3 both done):
  W6.D4  (project list page; blocked_by D2 + D3)
  W6.D9  (vim engine; blocked_by D3 only — runs in parallel with D4)

Sequential after D4 (wails.ts serial chain):
  W6.D5  (project detail; blocked_by D2 + D3 + D4)
  W6.D6  (create dialog; blocked_by D5)
  W6.D7  (dispatcher trigger; blocked_by D6)
  W6.D8  (settings panel; blocked_by D7)
```

**Acyclicity check:** D1 → D2; D3 (independent); D2+D3 → D4; D3 → D9; D4 → D5 → D6 → D7 → D8. No cycle.

**Parallelism:** D1 ‖ D3; D4 ‖ D9 (after D2+D3); D5–D8 serial (wails.ts additions).

---

## Refinements Logged

| ID | Description | Trigger |
|---|---|---|
| FE-GO-TEST-R1 | Go test coverage for Wails root files (main.go, app.go) deferred pre-MVP; excluded from mage ci via //go:build wails | Accepted risk |
| FE-LAYOUT-R1 | MainLayout.astro created as stub in D3, filled in D4, nav-link extended in D8 — consider a dedicated layout droplet if it grows further | Future |
| FE-LIVE-TAIL-R1 | SpawnOutputViewer v1 uses status indicator only; live tail via Wails runtime.EventsOn is a future enhancement | Post-v1 |
| FE-BINDINGS-FETCH-R1 | .tillsyn/bindings.json loaded via symlink at `frontend/public/.tillsyn-bindings.json` (R2 FF1-R2 absorption) + HTTP fetch in vim engine v1; pnpm build resolves symlink into static dist/ for Wails-embedded production. **Limitation**: production binary captures bindings at build time; runtime edits require rebuild. GetBindingsJSON IPC method is a future enhancement for live-reload + offline robustness | Post-v1 |
| FE-BASELINE-COPY-R1 | stil-baseline.json copied to frontend/public/ for browser bundling; replace with pnpm workspace link when stil-solid publishes | Post-dogfood |
| FE-WAILS-TS-SERIAL-R1 | wails.ts is a shared file touched by D4-D8; serialized via blocked_by chain; consider splitting into per-domain files (wails-projects.ts, wails-action-items.ts, etc.) in a future refactor | Post-v1 |
| FE-CROSS-PLATFORM-R1 | wails-keys.ts is macOS-scoped in v1 (Cmd+Q/M/W/H); Linux Ctrl-key and Windows Alt+F4 filtering deferred | Post-v1 |
