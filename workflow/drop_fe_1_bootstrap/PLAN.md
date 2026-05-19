# DROP_FE_1_BOOTSTRAP — Wails + SolidJS + Astro Skeleton, Minimal Read-Only Proof-Of-Wiring View

**State:** planning
**Blocked by:** —
**Paths (expected):** `ui/` (new top-level dir relocated from existing repo-root `frontend/` + `main.go` + `wails.json` — see Notes §N1), `magefile.go` (rename `CiFe` -> `CiUI` + repath), `.gitignore` updates, `wails.json` (relocated). No edits inside `cmd/` or `internal/` beyond read-only consumption of `internal/app.Service`.
**Packages (expected):** Go: `./ui` (Wails main, build-tag `wails`); FE: `ui/frontend/` (Astro + `@astrojs/solid-js`); build: `magefile.go`.
**PLAN.md ref:** new parallel FE lane; not numbered into the Go cascade sequence
**Workflow:** `workflow/example/drops/WORKFLOW.md`
**Cascade concept:** `AGENT_CASCADE_DESIGN.md`
**Started:** 2026-05-18
**Closed:** —

## Scope

First drop of the FE lane. Stand up a Wails + SolidJS + Astro desktop-app skeleton at `ui/` top-level, with one end-to-end minimal read-only view that proves in-process Go ↔ JS wiring against the existing Tillsyn SQLite DB via the same `internal/app/*` services the CLI uses. Nothing about this drop competes with the TUI — the FE earns replacement on merit later.

Locked design decisions (orch + dev, 2026-05-18 chat):

1. **Location:** `ui/` top-level peer to `cmd/`, `internal/`.
2. **Bootstrap:** `wails init -n tillsyn-ui -t solidjs` into `ui/`, then add Astro for static framing of `ui/frontend/`.
3. **TUI relationship:** coexist; FE earns replacement long-term.
4. **Wiring:** in-process Go bindings — Wails exposes Go services to JS directly. NOT MCP-from-Wails.

See `REVISION_BRIEF.md` for the full locked-decisions rationale, acceptance criteria, out-of-scope hard list, and open planner questions.

## Planner

**Round 1 — L1 (orch-dispatched, 2026-05-18).** Decomposition into 6 atomic droplets. The repo already carries a partial Wails+Astro+Solid scaffold at the repo root (commits `a9bac6c` "feat(fe): wails bootstrap at repo root with build-tag isolation" and `8d33539` "feat(fe): astro solid scaffold with stil tokens and mage ci-fe"), but the locked decision L1 mandates `ui/` top-level. This drop **relocates** the existing scaffold from `frontend/`/`main.go`/`wails.json` into `ui/frontend/`/`ui/main.go`/`ui/wails.json` and then adds the one read-only proof view. See `## Notes` §N1 for the brief-vs-reality decision fork the orchestrator must clear with the dev BEFORE Phase 4 builders fire.

The proof view binds against `app.Service.ListProjects(ctx, includeArchived=false) ([]domain.Project, error)` (`internal/app/service.go:2252`) — that signature is the smallest read-only surface that exercises the full chain: Wails IPC → Go method → `internal/app.Service` → `internal/adapters/storage/sqlite` → live `.tillsyn/tillsyn.db`. Project rows have `ID` + `Name` already on `domain.Project`; no projection layer needed for this drop.

Droplets:

---

### Droplet 1.1 — Relocate existing FE scaffold from repo root into `ui/` subtree

- **State:** todo
- **Paths:**
  - moves: `main.go` → `ui/main.go`, `wails.json` → `ui/wails.json`, `frontend/` → `ui/frontend/` (entire tree, including `astro.config.mjs`, `package.json`, `tsconfig.json`, `src/`, `tests/`, `public/`)
  - writes: `.gitignore` (add `ui/build/`, `ui/frontend/node_modules/`, `ui/frontend/dist/`, `ui/frontend/.astro/`; remove now-stale `frontend/*` entries if present)
  - writes: `ui/.gitignore` (Wails-specific artefacts: `build/bin/`, `build/darwin/`, `build/windows/`, `build/linux/`)
- **Packages:** `./ui` (new Go package, build-tag `wails`); `ui/frontend/` (Astro+Solid pnpm workspace).
- **Acceptance:**
  - `git ls-files ui/main.go ui/wails.json ui/frontend/astro.config.mjs ui/frontend/package.json` returns all four paths.
  - `git ls-files frontend/ main.go wails.json` returns empty (originals fully relocated).
  - `cat ui/wails.json` shows `"frontend:dir": "frontend"` and `"frontend:dev:serverUrl": "http://localhost:4321"` (relative to the wails.json location — i.e. `ui/frontend/`).
  - `cat ui/main.go | head -1` shows `//go:build wails` build tag still present.
  - `cd ui/frontend && pnpm install` exits 0 and produces `ui/frontend/node_modules/`.
  - `cd ui/frontend && pnpm run build` exits 0 and produces `ui/frontend/dist/index.html`.
  - `mage ci` still passes green (Go gates unaffected — the `wails` build tag fence keeps `ui/main.go` out of the default build).
- **Blocked by:** —

---

### Droplet 1.2 — Rename `magefile.go` `CiFe` target → `CiUI` repathed at `ui/frontend/`

- **State:** todo
- **Paths:** `magefile.go` (edit only the `CiFe` func, its alias map entry if any, and any doc comment that mentions `frontend/`).
- **Packages:** root `magefile.go` (build-tag `mage`).
- **Acceptance:**
  - `mage -l` lists `ci-ui` (or `ciUI`) and **does NOT** list `ci-fe` (renamed, not added alongside — avoids alias drift).
  - `mage ciUI` runs `pnpm run test:unit` and `pnpm run build` inside `ui/frontend/` (not `frontend/`). On a freshly-relocated tree (post-D1.1), both stages exit 0.
  - `mage ci` continues green (CI target itself unchanged in scope).
  - `magefile.go` source no longer contains the substring `"frontend"` outside an explicit comment that points to `ui/frontend`.
  - Single-line conventional commit: `refactor(mage): rename ci-fe to ci-ui and repath under ui/frontend/`.
- **Blocked by:** 1.1 (the path `ui/frontend/` must exist before the target is repointed; same file `magefile.go` is exclusive to D1.2 — no overlap with D1.1).

---

### Droplet 1.3 — Wire `ui/main.go` to construct a real `*app.Service` against `.tillsyn/tillsyn.db`

- **State:** todo
- **Paths:** `ui/main.go` (replace the `NewApp(nil)` placeholder with real service construction — see `cmd/till/main.go:2314` and `:2414` for the existing pattern: `sqlite.Open(cfg.Database.Path)` then `app.NewService(repo, uuid.NewString, nil, app.ServiceConfig{...})`). May add one local helper inside `ui/main.go` (single file in droplet to keep scope tight); do NOT introduce a new `ui/bridge/` package this drop (see `## Notes` §N2 — Q2 resolution).
- **Packages:** `./ui` (Wails main, build-tag `wails`); read-only deps on `github.com/evanmschultz/tillsyn/internal/app`, `github.com/evanmschultz/tillsyn/internal/adapters/storage/sqlite`, `github.com/evanmschultz/tillsyn/internal/config`, `github.com/google/uuid`.
- **Acceptance:**
  - `ui/main.go` no longer contains the literal `NewApp(nil)` line; instead constructs `*app.Service` and passes it to `NewApp(svc)`.
  - The constructed service opens against the same DB path the CLI uses (`config.Load` + `cfg.Database.Path` resolution — mirror the `cmd/till/main.go` resolution, do not hardcode a path).
  - On a dev machine with `.tillsyn/tillsyn.db` populated by at least one prior `till project create` invocation: `cd ui && wails build` produces a binary; `./build/bin/Tillsyn.app/Contents/MacOS/Tillsyn` (or the platform equivalent) starts without panic. Verified manually by orch-or-dev launching the binary; QA agent verifies via `wails build` exit code 0 + presence of the output binary.
  - `cd ui && wails dev` connects to the Astro dev server at `http://localhost:4321` and opens a Wails window. JS console shows no IPC errors related to missing bindings.
  - No changes outside `ui/main.go`. `mage ci` remains green (build-tag `wails` keeps this file out of the default Go build).
- **Blocked by:** 1.1 (the file `ui/main.go` is created by D1.1 via relocation; D1.3 is the next writer of that file — file-shared, must serialize).

---

### Droplet 1.4 — Expose `ListProjects` IPC method on the Wails App struct

- **State:** todo
- **Paths:** `ui/main.go` (add one exported method `func (a *App) ListProjects() ([]ProjectDTO, error)` and one tiny DTO type `ProjectDTO struct { ID string; Name string }` to keep the JS-side shape stable across `domain.Project` schema churn). Same file as D1.3; serialize via `blocked_by`.
- **Packages:** `./ui`.
- **Acceptance:**
  - `ui/main.go` defines `App.ListProjects() ([]ProjectDTO, error)` whose body calls `a.svc.ListProjects(a.ctx, false)` and maps each `domain.Project` to `ProjectDTO{ID: p.ID, Name: p.Name}`.
  - `cd ui && wails dev` — opening the dev window, then in the browser DevTools console: `await window.go.main.App.ListProjects()` returns an array. When the local DB has ≥1 non-archived project, the array has length ≥1 and each element has non-empty `ID` and `Name`. When the DB is empty (cold dev machine), the call returns `[]` (not `null`, not throw) — this is the "DB-empty-but-wiring-works" acceptance shape.
  - `cd ui && wails build` exits 0. Resulting binary, when launched, exposes the same method on `window.go.main.App` (verified by QA agent via `wails dev` since `wails build` headless inspection requires a runtime probe — the build-success + dev-mode probe pair is sufficient acceptance).
  - DTO is defined inside `ui/main.go` (no new `ui/dto/` package this drop — same YAGNI rationale as N2).
- **Blocked by:** 1.3 (D1.4 extends the same `ui/main.go` D1.3 just rewrote; serialize).

---

### Droplet 1.5 — Solid component + Astro page rendering the project list

- **State:** todo
- **Paths:**
  - writes: `ui/frontend/src/components/ProjectList.tsx` (Solid component, ~30-60 LOC: `createResource` against `window.go.main.App.ListProjects`, plain `<ul><li>` render of `id` + `name`, simple loading + error states).
  - writes: `ui/frontend/src/pages/index.astro` (single Astro page using `MainLayout.astro` and mounting `<ProjectList client:load />` — see Notes §N3 on the `client:load` choice).
  - writes: `ui/frontend/src/types/wails.d.ts` (one tiny ambient declaration so the SolidJS component compiles cleanly under TypeScript: `declare global { interface Window { go: { main: { App: { ListProjects(): Promise<{ ID: string; Name: string }[]> } } } } }`).
- **Packages:** `ui/frontend/` (Astro+Solid pnpm workspace). Does NOT touch any Go code.
- **Acceptance:**
  - `cd ui/frontend && pnpm run build` exits 0. `ui/frontend/dist/index.html` exists and contains a `<div id="...solid-island...">` or equivalent hydration marker for the `<ProjectList client:load />` island.
  - `cd ui && wails dev` — the launched window shows: a heading, a `<ul>` with one `<li>` per non-archived project in the dev DB, each rendering `ID` and `Name`. When the dev DB is empty, the window shows a visible "No projects yet" empty-state string (component must render an empty-state, not a blank `<ul>` — that's the only way the QA agent can distinguish "loaded zero rows" from "binding crashed silently").
  - `cd ui && wails build && open ./build/bin/Tillsyn.app` — same rendering inside the built binary.
  - `migration-markers.test.ts` either passes (no marker required on `ProjectList.tsx`) OR `ProjectList.tsx` carries the `// MIGRATION TARGET: @hylla/stil-solid` marker; the planner picks: **carry the marker** to stay consistent with the migration-target convention already encoded in `ui/frontend/tests/migration-markers.test.ts`. Acceptance: `cd ui/frontend && pnpm run test:unit` exits 0 and the migration-markers test reports `ProjectList.tsx contains migration marker` as a passing case (not a vacuous skip).
- **Blocked by:** 1.2 (mage target must run inside `ui/frontend/` so `pnpm run test:unit` resolves the relocated tree), 1.4 (the JS binding `window.go.main.App.ListProjects` must exist before the Solid component can `createResource` against it; runtime-only blocker, but the type declaration in `wails.d.ts` is the planner's compile-time pin).

---

### Droplet 1.6 — `ui/README.md` orientation doc + `mage ui-dev` and `mage ui-build` targets

- **State:** todo
- **Paths:**
  - writes: `ui/README.md` (~30-60 lines: one paragraph on what `ui/` is, one paragraph on how to run it (`mage ui-dev` for hot-reload Wails+Astro dev loop; `mage ui-build` for the production binary), one paragraph on the in-process binding wiring and the read-only-this-drop guarantee, link to `REVISION_BRIEF.md`).
  - writes: `magefile.go` (adds two new exported functions `UIDev() error` and `UIBuild() error` running `wails dev` and `wails build` respectively from the `ui/` directory; adds `ui-dev` / `ui-build` aliases to the `Aliases` map; existing `CiUI` from D1.2 stays as-is).
- **Packages:** root (`magefile.go` + `ui/README.md`). Same file `magefile.go` as D1.2 → must serialize.
- **Acceptance:**
  - `mage -l` lists `ui-dev`, `ui-build`, and the unchanged `ci-ui` from D1.2.
  - `mage uiBuild` (or alias `mage ui-build`) exits 0 on a clean checkout and produces a binary at `ui/build/bin/Tillsyn.app/Contents/MacOS/Tillsyn` (or platform-equivalent path; the magefile resolves `${GOOS}` to pick the right directory).
  - `mage uiDev` (or alias `mage ui-dev`) starts the Wails dev loop without erroring out of the gate (process stays running until SIGINT — QA agent runs it with a short timeout wrapper and verifies stdout shows `Wails CLI v2.12.0` + `[Wails] Dev mode` markers within 10s).
  - `ui/README.md` exists and contains at minimum: "in-process Go bindings", "read-only this drop", "see REVISION_BRIEF.md", and the two mage target names.
  - `mage ci` remains green.
- **Blocked by:** 1.2 (same `magefile.go` file; serialize), 1.5 (the `mage ui-build` smoke check exercises the full UI build — the proof view must render real data before the README can honestly claim "in-process bindings work").

---

## Notes

### N1 — Brief-vs-reality decision fork (REQUIRES DEV CLEARANCE BEFORE PHASE 4)

The `REVISION_BRIEF.md` §1 "Hard Prerequisites" states "No existing `ui/` directory in the repo (verified at brief authoring time)." That is true — `ui/` does not exist. **But the brief implicitly assumes the repo is FE-virgin**, and that assumption is false. The repo already carries:

- `main.go` (build tag `wails`) wired to `internal/app.Service` with a placeholder `NewApp(nil)`, ready to be made real.
- `wails.json` at repo root pointing at `frontend/`.
- `frontend/` directory with Astro + `@astrojs/solid-js` + Vitest + Playwright scaffold, `MainLayout.astro`, `stil-tokens.css`, `stil-baseline.json`, `.tillsyn/bindings.json` symlink, and a `migration-markers.test.ts` that pins component files to the `// MIGRATION TARGET: @hylla/stil-solid` marker convention.
- `magefile.go` `CiFe` target that runs `pnpm run test:unit` + `pnpm run build` inside `frontend/`.
- Two prior commits already on `main`: `a9bac6c` and `8d33539`.

**The brief's locked decision L1 says `ui/` top-level.** The on-disk state says `frontend/` at repo root. These conflict.

This planner cannot relitigate L1 unilaterally (it is a locked architectural decision). It also cannot ignore the on-disk state (the conflict is factual). The plan above honors L1 — droplet D1.1 relocates the existing scaffold into `ui/`, treating the prior bootstrap commits as a head start rather than the final shape.

**Dev decision needed BEFORE Phase 4 builder dispatches:** one of —

- **(A) Confirm L1 stands** → the plan as written runs. D1.1 does a clean `git mv` relocation; the prior commits remain valid history; the drop completes by adding the proof view inside the relocated tree. **(Recommended default — preserves L1 + uses the prior bootstrap commits as work already done.)**
- **(B) Revise L1 to "FE lives at repo root (`frontend/` + `main.go` + `wails.json`)"** → orch rewrites this `PLAN.md` (drop D1.1 and D1.2; renumber D1.3-D1.6 to operate on the in-place root paths; keep `CiFe` name; D1.6 README lands at `README_UI.md` or `frontend/README.md`).
- **(C) Hybrid** → some files stay at root (`main.go`, `wails.json`) for Wails-toolchain compatibility, others move (`frontend/` → `ui/frontend/`). The orchestrator should NOT pick this option without dev sign-off; it generates the worst of both worlds (split locations, dual-path build commands).

**Default if dev does not respond before Phase 4: A** (honor the locked decision).

### N2 — Q2 (bridge package location) — resolved: directly in `ui/main.go`, no `ui/bridge/` this drop

The brief's open question Q2 asks whether the binding glue lives in `ui/main.go` or a new `ui/bridge/` package. For this drop's scope (one read-only method, one DTO), a separate `ui/bridge/` package is YAGNI — it adds an import boundary that buys nothing when there is exactly one method to expose. The DTO + method definition land inline in `ui/main.go` (D1.4). When the FE grows to ≥3 IPC methods or starts needing typed projections that the JS-side wants to share across components, the next FE drop can extract `ui/bridge/` cleanly via a single-file refactor — at that point the boundary will be earned by real surface area.

### N3 — Q3 (Astro + SolidJS integration shape) — resolved on-disk: `@astrojs/solid-js` integration + `client:load` directive

The brief's open question Q3 asks how Astro mounts SolidJS islands. **On-disk evidence resolves this for us**: `frontend/package.json` (now `ui/frontend/package.json` post-D1.1) already depends on `@astrojs/solid-js@^4.4.0` and `solid-js@^1.9.7`, and `frontend/astro.config.mjs` already wires `integrations: [solidJs()]`. D1.5 uses the standard Astro island pattern: `<ProjectList client:load />` inside `index.astro`. The `client:load` directive (rather than `client:idle` or `client:visible`) is chosen because the proof view is the **only** content on the page and we want eager hydration to make the QA acceptance check ("the list renders within 2 seconds of window open") deterministic. Future FE drops should default to `client:idle` per FE-planning-agent doctrine; the eager-load choice here is a one-drop acceptance-determinism call.

### N4 — Q1 (Go workspace integration) — resolved on-disk: same root module

The brief's open question Q1 asks whether `ui/` should be its own Go module. **On-disk evidence resolves this for us**: the existing `main.go` uses `package main` and imports `github.com/evanmschultz/tillsyn/internal/app` directly — i.e. it sits inside the existing root `go.mod` module. The `//go:build wails` tag is the isolation mechanism: the Wails main is excluded from the default Go build but included when `wails build` invokes Go with the `wails` tag. Relocating `main.go` to `ui/main.go` keeps the same module + same build-tag isolation; no `ui/go.mod` is needed. This was the brief's default and on-disk reality confirms it.

### N5 — Q4 (mage target naming) — resolved: `ui-*` prefix, with `ci-fe` → `ci-ui` rename

The brief's open question Q4 asks `fe-*` vs `ui-*`. Pick `ui-*` to match the locked-decision directory name `ui/` — keeps the naming surface coherent for the next FE drop. D1.2 renames the existing `CiFe` → `CiUI`. D1.6 adds `UIDev` + `UIBuild`. Single rename now, not a `fe-*`/`ui-*` split later.

### N6 — Out-of-scope (re-affirmed from brief §7)

This plan does NOT add:
- Auth flow in the FE, anywhere.
- MCP-over-the-Wails-bridge (Locked Decision L4).
- Write operations (the proof method `ListProjects` is read-only; no `CreateProject` / `UpdateProject` / etc.).
- TUI feature parity. No FE surface duplicates any TUI screen this drop.
- Cross-platform packaging (`.dmg` / `.exe` / `.AppImage`).
- Tests for SolidJS components (per brief §7 — `migration-markers.test.ts` already exists from prior commits and stays; no new SolidJS unit tests this drop).
- Edits inside `cmd/` or `internal/`. The bridge consumes `internal/app.Service` and `internal/adapters/storage/sqlite` as already-public read-only APIs; no new exports added.

### N7 — `wails` CLI prerequisite check (REQUIRES DEV CONFIRMATION BEFORE PHASE 4)

`REVISION_BRIEF.md` §1 says the dev confirms once that `wails` and `pnpm` are on `$PATH` before Phase 4. Orchestrator must check this **before** spawning the first builder. A simple `wails doctor` + `pnpm --version` invocation by the orch (or a dev confirmation in chat) suffices. The plan does NOT add a droplet for "install wails" — that's a dev-machine prerequisite, not a drop artifact.

### N8 — Blocker graph summary

```
D1.1 ──────────────────────┬─▶ D1.2 ──────────┬─▶ D1.5
                            │                  │
                            └─▶ D1.3 ─▶ D1.4 ──┘
                                              └─▶ D1.6
                            │
                            └────────────────────▶ D1.6 (via D1.2 + D1.5)
```

Concretely:
- D1.1 unblocks D1.2 and D1.3 (both edit different files post-relocation).
- D1.2 (magefile.go) and D1.3 (ui/main.go) edit disjoint files, both depend on D1.1 alone — they MAY run in parallel.
- D1.3 unblocks D1.4 (same file `ui/main.go`, serialize).
- D1.5 (frontend code) waits on D1.2 (mage target points at relocated tree) AND D1.4 (JS-side binding must exist).
- D1.6 (magefile.go + README) waits on D1.2 (same file `magefile.go`) AND D1.5 (so the README's wiring claims are evidence-backed).

Mirrored into `_BLOCKERS.toml` alongside this file.
