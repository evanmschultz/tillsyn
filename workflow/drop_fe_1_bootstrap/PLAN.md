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

- **State:** done
- **Paths:**
  - moves: `main.go` → `ui/main.go`, `wails.json` → `ui/wails.json`, `frontend/` → `ui/frontend/` (entire tree, including `astro.config.mjs`, `package.json`, `tsconfig.json`, `src/`, `tests/`, `public/`)
  - writes: `ui/wails.json` (after relocation, change `"outputfilename": "tillsyn"` → `"outputfilename": "Tillsyn"` — see §N13 for the binary-capitalization rationale)
  - writes: `ui/frontend/package.json` (add `"packageManager": "pnpm@9.0.0"` field — see §N9 for rationale)
  - writes: `.gitignore` (add `ui/build/`, `ui/frontend/node_modules/`, `ui/frontend/dist/`, `ui/frontend/.astro/`; remove now-stale `frontend/*` entries if present; **MUST NOT** add `ui/frontend/pnpm-lock.yaml` — the lockfile is committed, see §N9)
  - writes: `ui/.gitignore` (Wails-specific artefacts: `build/bin/`, `build/darwin/`, `build/windows/`, `build/linux/`)
  - commits: `ui/frontend/pnpm-lock.yaml` (created by `pnpm install` during this droplet — see §N9 for the pin-not-switch rationale)
- **Sub-step ordering (build-time vs commit-time):** after `git mv` completes, run `cd ui/frontend && pnpm install` to materialize `node_modules/` AND generate `pnpm-lock.yaml`. THEN `git add ui/wails.json ui/frontend/pnpm-lock.yaml ui/frontend/package.json .gitignore ui/.gitignore` (plus the relocated tree from the `git mv`s). THEN commit. Sequence matters: skipping `pnpm install` would skip lockfile generation; staging before generation would commit nothing for the lockfile.
- **Packages:** `./ui` (new Go package, build-tag `wails`); `ui/frontend/` (Astro+Solid pnpm workspace).
- **Acceptance:**
  - `git ls-files ui/main.go ui/wails.json ui/frontend/astro.config.mjs ui/frontend/package.json` returns all four paths.
  - `git ls-files frontend/ main.go wails.json` returns empty (originals fully relocated).
  - `cat ui/wails.json` shows `"frontend:dir": "frontend"` and `"frontend:dev:serverUrl": "http://localhost:4321"` (relative to the wails.json location — i.e. `ui/frontend/`).
  - `grep -q '"outputfilename": "Tillsyn"' ui/wails.json` exits 0. The binary name is `Tillsyn` (capital T) for desktop-brand consistency with `Name: "Tillsyn"` — see §N13. Default scaffold-time `"tillsyn"` lowercase MUST be replaced.
  - `cat ui/main.go | head -1` shows `//go:build wails` build tag still present.
  - `grep -q '^//go:embed all:frontend/dist' ui/main.go` exits 0 — the embed directive is **literally unchanged** (still `//go:embed all:frontend/dist`, NOT helpfully-but-wrongly rewritten to `all:ui/frontend/dist`). Go resolves the path relative to `ui/main.go`'s dir, so `frontend/dist` correctly resolves to `ui/frontend/dist`. See §N10 for the relocation-trap rationale.
  - `grep -q '"packageManager": "pnpm@' ui/frontend/package.json` exits 0 — pnpm version pinned for Corepack auto-fetch.
  - `cd ui/frontend && pnpm install` exits 0 and produces `ui/frontend/node_modules/`. Lockfile is created at `ui/frontend/pnpm-lock.yaml`.
  - `git ls-files ui/frontend/pnpm-lock.yaml` returns the path (lockfile committed, reproducible installs).
  - `grep -q 'ui/frontend/pnpm-lock.yaml' .gitignore` exits **non-zero** (lockfile is NOT excluded).
  - `cd ui/frontend && pnpm run build` exits 0 — Astro's build pipeline runs to completion on the relocated tree (load-bearing signal). The prior bootstrap commits (`a9bac6c` / `8d33539`) did NOT scaffold `src/pages/`, so this droplet does not yet produce `dist/index.html` — D1.5 adds `ui/frontend/src/pages/index.astro` and `dist/index.html` materializes then. Build emits `dist/_astro/client.<hash>.js` and the carried-over `public/` assets, which is the right post-relocation state for a `src/pages/`-less tree.
  - `magefile.go` `verifySources()` (currently at L293 region — `git ls-files --error-unmatch magefile.go cmd/till/main.go cmd/till/main_test.go`) does NOT reference root `main.go`; the relocation does not break the source-tracking guard. Verified by `grep -n verifySources magefile.go`.
  - `mage format` reports no diff against `ui/main.go` after relocation (relocated file remains gofumpt-clean — `magefile.go` `formatCheck` stage scans every `*.go` via `trackedGoFiles()` regardless of build tag, so this is a real `mage ci` precondition).
  - `mage ci` still passes green (Go gates unaffected — the `wails` build tag fence keeps `ui/main.go` out of the default compile + test, and the format step is satisfied by the line above).
- **Blocked by:** —

---

### Droplet 1.2 — Rename `magefile.go` `CiFe` target → `CiUI` repathed at `ui/frontend/`

- **State:** done
- **Paths:** `magefile.go` (edit only the `CiFe` func, its alias map entry if any, and any doc comment that mentions `frontend/`).
- **Packages:** root `magefile.go` (build-tag `mage`).
- **Acceptance:**
  - `mage -l` lists `ciUI` (canonical name) and **does NOT** list `ci-fe` or `ciFe` (renamed, not added alongside — avoids alias drift). The `ci-ui` alias is registered in the `Aliases` map at `magefile.go:26-36` (matching existing convention — e.g. `test-pkg`, `format-path`) and surfaces via `mage -h ciUI` reporting `Aliases: ci-ui`. Mage's `-l` output lists only canonical names; aliases dispatch at execution time (verified via `mage ci-ui` running the same body as `mage ciUI`).
  - `mage ciUI` runs `pnpm run test:unit` and `pnpm run build` inside `ui/frontend/` (not `frontend/`). On a freshly-relocated tree (post-D1.1), both stages exit 0.
  - `mage ci` continues green (CI target itself unchanged in scope).
  - `magefile.go` source no longer contains the substring `"frontend"` outside an explicit comment that points to `ui/frontend`.
  - Single-line conventional commit: `refactor(mage): rename ci-fe to ci-ui and repath under ui/frontend/`.
- **Blocked by:** 1.1 (the path `ui/frontend/` must exist before the target is repointed; same file `magefile.go` is exclusive to D1.2 — no overlap with D1.1).

---

### Droplet 1.3 — Wire `ui/main.go` to construct a real `*app.Service` against `.tillsyn/tillsyn.db`

- **State:** done
- **Paths:** `ui/main.go` (replace the `NewApp(nil)` placeholder with real service construction — see `cmd/till/main.go:2314` and `:2414` for the existing pattern: `sqlite.Open(cfg.Database.Path)` then `app.NewService(repo, uuid.NewString, nil, app.ServiceConfig{...})`). May add one local helper inside `ui/main.go` (single file in droplet to keep scope tight); do NOT introduce a new `ui/bridge/` package this drop (see `## Notes` §N2 — Q2 resolution). If a local helper is added (e.g. `loadConfig()`), it lives as a `func` inside `ui/main.go`, NOT as a new file under `ui/`.
- **Packages:** `./ui` (Wails main, build-tag `wails`); read-only deps on `github.com/evanmschultz/tillsyn/internal/app`, `github.com/evanmschultz/tillsyn/internal/adapters/storage/sqlite`, `github.com/evanmschultz/tillsyn/internal/config`, `github.com/google/uuid`.
- **Acceptance:**
  - `ui/main.go` no longer contains the literal `NewApp(nil)` line; instead constructs `*app.Service` and passes it to `NewApp(svc)`.
  - The constructed service opens against the same DB path the CLI uses (`config.Load` + `cfg.Database.Path` resolution — mirror the `cmd/till/main.go` resolution, do not hardcode a path).
  - `grep -q '^//go:embed all:frontend/dist' ui/main.go` exits 0 — the embed directive is preserved verbatim across the `NewApp(nil)` → real-service rewrite (D1.3 must not helpfully-but-wrongly rewrite the path; the directive is path-relative to the file's dir, not the module root). This is the same guard as D1.1, repeated here to catch builder regressions.
  - `cd ui && wails build` exits 0 AND the output binary exists at the expected platform path. On macOS (dev's env per session `Platform: darwin`): `./build/bin/Tillsyn.app/Contents/MacOS/Tillsyn` exists and `file ./build/bin/Tillsyn.app/Contents/MacOS/Tillsyn` reports a Mach-O binary. Note: `./build/bin/Tillsyn.app/...` is a **macOS-only path**; on Linux the binary lands at `./build/bin/Tillsyn`, on Windows at `.\build\bin\Tillsyn.exe`. Cross-platform packaging is out of scope per §N6.
  - `cd ui && wails dev` connects to the Astro dev server at `http://localhost:4321` and opens a Wails window. JS console shows no IPC errors related to missing bindings. (Runtime window-open is a dev-launch confirmation gate at Phase 6, NOT this droplet's acceptance — QA agent's acceptance is `wails build` exit-code 0 + binary file exists + Mach-O check, all of which are QA-agent-executable headlessly.)
  - No changes outside `ui/main.go`. `mage ci` remains green (build-tag `wails` keeps this file out of the default Go build).
- **Blocked by:** 1.1 (the file `ui/main.go` is created by D1.1 via relocation; D1.3 is the next writer of that file — file-shared, must serialize).

---

### Droplet 1.4 — Expose `ListProjects` IPC method on the Wails App struct + Go-side smoke test

- **State:** done
- **Paths:**
  - writes: `ui/main.go` — add one exported method `func (a *App) ListProjects() ([]ProjectDTO, error)`. Same file as D1.3; serialize via `blocked_by`.
  - writes: `ui/types.go` (new file, `package main`, build-tag `//go:build wails` for symmetry with `ui/main.go`) — defines the DTO type: `type ProjectDTO struct { ID string; Name string }`. Splitting the DTO out of `ui/main.go` pre-empts entrypoint-file bloat as more IPC methods land in future drops (see §N2 — F4-fals resolution; zero import-boundary cost since both files are `package main`).
  - writes: `ui/app_test.go` (new file, `//go:build wails`) — Go-side smoke test for the bridge, see acceptance below.
- **Packages:** `./ui`.
- **Acceptance:**
  - `ui/main.go` defines `App.ListProjects() ([]ProjectDTO, error)` whose body calls `a.svc.ListProjects(a.ctx, false)` and maps each `domain.Project` to `ProjectDTO{ID: p.ID, Name: p.Name}`.
  - `ui/types.go` exists, declares `package main`, carries `//go:build wails`, and defines `ProjectDTO` with exactly fields `ID string` and `Name string`. No other types in this drop.
  - `ui/app_test.go` exists with `//go:build wails`, declares `package main`, and contains a test function named exactly `TestApp_ListProjects_ReturnsDTOForExistingProject`. The test constructs `*app.Service` against an in-memory SQLite DB via the existing `sqlite.OpenInMemory()` helper (`internal/adapters/storage/sqlite/repo.go:101-118`) — NOT raw `sqlite.Open(":memory:")` (works coincidentally today via `MaxOpenConns(1)` but breaks silently if a future refactor adds a second connection; `OpenInMemory()` uses the canonical multi-connection-safe `"file::memory:?cache=shared"` DSN). Seeds at least one non-archived project via the existing service layer (`svc.CreateProject(...)` or equivalent), constructs `*App` against the seeded service, calls `app.ListProjects()`, and asserts: (a) `err == nil`, (b) `len(result) >= 1`, (c) every returned `ProjectDTO` has non-empty `ID` and `Name`, (d) the seeded project's `ID` + `Name` appear in the result set.
  - `go test -tags wails ./ui/...` exits 0. This is the QA-agent-executable smoke gate — runs headlessly, no Wails CLI / dev-window / DevTools-console probe needed. (Default `mage test-pkg` does NOT include the `wails` build tag, so this test is invoked explicitly via the tagged command; future FE drops may add a `mage test-ui` target wrapping it.)
  - `cd ui && wails build` exits 0; the output binary exposes `window.go.main.App.ListProjects` (verified transitively via the wailsbindings codegen succeeding during `wails build` — binding-generation failure surfaces as a non-zero exit code).
  - The DTO is defined in `ui/types.go`, NOT inline in `ui/main.go`. `grep -q 'type ProjectDTO struct' ui/main.go` exits **non-zero**; `grep -q 'type ProjectDTO struct' ui/types.go` exits 0.
- **Blocked by:** 1.3 (D1.4 extends the same `ui/main.go` D1.3 just rewrote; serialize). Note: `ui/types.go` and `ui/app_test.go` are new files D1.3 doesn't touch, but they are part of the same `./ui` package compile unit as `ui/main.go` — the serialize-on-`ui/main.go`-rewrite rule covers package-level locking too.

---

### Droplet 1.5 — Solid component + Astro page rendering the project list

- **State:** todo
- **Paths:**
  - writes: `ui/frontend/src/components/ProjectList.tsx` (Solid component, ~30-60 LOC: `createResource` against `window.go.main.App.ListProjects`, plain `<ul><li>` render of `id` + `name`, simple loading + error states; literal empty-state string `No projects yet` rendered when the resource resolves to an empty array). MUST contain the line `// MIGRATION TARGET: @hylla/stil-solid` — mandatory per `ui/frontend/tests/migration-markers.test.ts:36-44`, absence fails `pnpm run test:unit`.
  - writes: `ui/frontend/src/pages/index.astro` (single Astro page using `MainLayout.astro` and mounting `<ProjectList client:idle />` — see Notes §N3 on the `client:idle` choice).
  - writes: `ui/frontend/src/types/wails.d.ts` (one tiny ambient declaration so the SolidJS component compiles cleanly under TypeScript: `declare global { interface Window { go: { main: { App: { ListProjects(): Promise<{ ID: string; Name: string }[]> } } } } }`).
- **Packages:** `ui/frontend/` (Astro+Solid pnpm workspace). Does NOT touch any Go code.
- **Acceptance:**
  - `cd ui/frontend && pnpm run build` exits 0. `ui/frontend/dist/index.html` exists and contains a hydration marker for the `<ProjectList client:idle />` island (Astro renders `client:idle` directives with a `<astro-island ... client="idle">` web-component wrapper — see Context7 `/withastro/docs` § "Client Directives" + `island.client` runtime tag).
  - `grep -q '// MIGRATION TARGET: @hylla/stil-solid' ui/frontend/src/components/ProjectList.tsx` exits 0 (marker is mandatory, AND — not OR — per migration-markers test).
  - `grep -q 'No projects yet' ui/frontend/src/components/ProjectList.tsx` exits 0 (literal empty-state string is present in source).
  - `cd ui/frontend && pnpm run test:unit` exits 0. The migration-markers test reports `ProjectList.tsx` as a passing case (not a vacuous skip — `files.length > 0` branch must exercise).
  - `cd ui && wails dev` — the launched window DOM contains (a) an `<h1>` or `<h2>` element with non-empty text content AND (b) a `<ul>` element. When the DB has projects: the `<ul>` has ≥1 `<li>` children, each containing both the project's `ID` and `Name` as visible text. When the DB is empty: the page contains the literal string `No projects yet` (component renders an empty-state, not a blank `<ul>` — that's the only way the QA agent can distinguish "loaded zero rows" from "binding crashed silently").
  - **CLI cross-check (A3 verification — F10 proof resolution).** Against the SAME `.tillsyn/tillsyn.db` the dev window opens: capture `till project list` output (or equivalent CLI surface for non-archived listing) and the rendered `<li>` set from the dev window. Assert: the set of (ID, Name) pairs matches exactly modulo ordering. Same ordering not required; same set required. QA-agent path: capture `till project list` machine-readable output (JSON if available; else parse the table), capture the Solid-rendered list via a DOM dump or the dev-mode `await window.go.main.App.ListProjects()` JSON, deep-equal-modulo-ordering.
  - `cd ui && wails build && open ./build/bin/Tillsyn.app` — same rendering inside the built binary (dev-launch confirmation at Phase 6, not this droplet's machine-checkable acceptance — the dev opens the binary and confirms visually).
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
  - `mage uiBuild` (or alias `mage ui-build`) exits 0 on a clean checkout AND the output binary exists at the expected platform path. On macOS (dev's env per session `Platform: darwin`): `ui/build/bin/Tillsyn.app/Contents/MacOS/Tillsyn` exists. (Linux/Windows produce different paths — `ui/build/bin/Tillsyn` and `ui\build\bin\Tillsyn.exe` respectively; cross-platform packaging is out of scope per §N6.) No `wails` CLI version-string match — the Wails CLI version is dev-machine-controlled (see §N11); we depend only on the Go binding pin in `go.mod`.
  - `mage uiDev` (or alias `mage ui-dev`) starts the Wails dev loop without erroring out of the gate. QA agent runs it with a 60s timeout wrapper and verifies (a) the process stays running until SIGINT (no immediate-exit failure), AND (b) stdout contains the literal substring `[Wails] Dev mode` (the dev-mode startup marker — this is a Wails-runtime emission, not a CLI-version-string check). 60s tolerates cold-cache `pnpm install` (8-30s) + Astro start (3-6s) + Go bridge build (10-25s) + WebView launch (1-3s) — sum range 22-64s, so 60s covers the upper end of a worst-case cold-cache dev machine without going past the §N12 smoke-vs-build cap.
  - `ui/README.md` exists and contains at minimum: "in-process Go bindings", "read-only this drop", "see REVISION_BRIEF.md", and the two mage target names.
  - `mage ci` remains green.
- **Blocked by:** 1.2 (same `magefile.go` file; serialize), 1.5 (the `mage ui-build` smoke check exercises the full UI build — the proof view must render real data before the README can honestly claim "in-process bindings work"). Note: D1.3 and D1.4 are transitive blockers via D1.5 (D1.5 blocked_by D1.4; D1.4 blocked_by D1.3); D1.6's explicit `blocked_by` list contains only the immediate `[1.2, 1.5]` per the `_BLOCKERS.toml` immediate-children rule.

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

### N2 — Q2 (bridge package location) — resolved: `App` methods in `ui/main.go`, DTOs in `ui/types.go`, no `ui/bridge/` package this drop

The brief's open question Q2 asks whether the binding glue lives in `ui/main.go` or a new `ui/bridge/` package. For this drop's scope (one read-only method, one DTO), a separate `ui/bridge/` package is YAGNI — it adds an import boundary that buys nothing when there is exactly one method to expose. **However**, the DTO itself is separable from the entrypoint at zero import-boundary cost: per round-1 falsification F4, splitting `ProjectDTO` into a sibling `ui/types.go` file (still `package main`, same dir, same build tag `//go:build wails`) pre-empts entrypoint bloat as future drops add more IPC methods — without paying for a new package. D1.4 writes `ui/types.go` for the DTO and keeps `App.ListProjects` in `ui/main.go`. The full `ui/bridge/` extraction (its own package, exported types, import boundary) waits until the FE accumulates ≥3 IPC methods + typed projections shared across components.

DTO field naming follows Go conventions: `ID string; Name string` (capitalized — exported). Wails serializes Go structs to JS as `{"ID": "...", "Name": "..."}` by default; the JS-side TypeScript declaration in `wails.d.ts` types this as `Promise<{ ID: string; Name: string }[]>` to match. Capitalized field names on the JS wire diverge from JS-idiom camelCase, but converting via `json:` tags adds friction now for zero benefit. Future FE drops MAY add struct tags to camelCase the wire format if the JS surface accumulates enough sprawl to warrant the conversion.

### N3 — Q3 (Astro + SolidJS integration shape) — resolved on-disk: `@astrojs/solid-js` integration + `client:idle` directive

The brief's open question Q3 asks how Astro mounts SolidJS islands. **On-disk evidence resolves this for us**: `frontend/package.json` (now `ui/frontend/package.json` post-D1.1) already depends on `@astrojs/solid-js@^4.4.0` and `solid-js@^1.9.7`, and `frontend/astro.config.mjs` already wires `integrations: [solidJs()]`. D1.5 uses the standard Astro island pattern: `<ProjectList client:idle />` inside `index.astro`.

**Round-1 falsification F3 reset the directive choice from `client:load` to `client:idle`.** Per Astro docs (Context7 `/withastro/docs` § "Client Directives"): `client:load` immediately loads and hydrates the component's JavaScript when the page loads — "ideal for high-priority UI elements that require immediate interactivity." `client:idle` waits for `requestIdleCallback`. On WebKit-based Wails (darwin/Linux), `requestIdleCallback` is NOT natively supported — Astro's runtime polyfills via a fallback to the `load` event, so hydration still fires reliably (just bound to first paint rather than browser-idle). The ProjectList is a read-only list with no immediate-interactivity requirement; `client:load` is cargo-culted eager hydration. `client:idle` is the FE-planning-agent doctrine default and is the right choice for this bootstrap drop. Setting it correctly here keeps the pattern clean for future islands instead of relying on a paper "future drops should …" rule that subsequent planners must catch.

QA acceptance (D1.5: "the launched window DOM contains …") remains satisfiable under `client:idle` — Playwright-style probes (`networkidle`, `domcontentloaded`) wait for hydration before asserting DOM content.

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

### N7 — Dev-machine prerequisite check (REQUIRES DEV CONFIRMATION BEFORE PHASE 4)

`REVISION_BRIEF.md` §1 says the dev confirms once that `wails` and `pnpm` are on `$PATH` before Phase 4. Orchestrator must check this **before** spawning the first builder. The full check matrix the orch (or dev-in-chat) verifies:

- `wails doctor` — exit 0; reports a Wails CLI v2.x (no strict version-string match; see §N11).
- `node --version` — exit 0; reports v16.13.0 or newer (Corepack requires this minimum for `packageManager` field honoring).
- `corepack --version` — exit 0. If exit non-zero or `corepack` not found, dev runs `corepack enable` once (ships with Node 16.13+ but may need explicit enable on some distros).
- `pnpm --version` — exit 0. If `corepack enable` ran above, Corepack auto-fetches the pinned `pnpm@9.0.0` at first `pnpm install`; if dev has a system pnpm 9.x installed, it works silently; if dev has a system pnpm 10.x WITHOUT Corepack enabled, the pin enforcement may strict-fail (re-check after `corepack enable`).

The plan does NOT add a droplet for "install wails" or "install pnpm" — these are dev-machine prerequisites, not drop artifacts. If any check above fails, dev resolves locally before builder dispatches.

### N8 — Blocker graph summary

```
D1.1 ──┬─▶ D1.2 ───────────────┬─▶ D1.5 ──▶ D1.6
       │                       │             ▲
       │                       │             │
       └─▶ D1.3 ──▶ D1.4 ──────┘             │
                                             │
       D1.2 ────────────────────────────────-┘
```

Concretely:
- D1.1 unblocks D1.2 and D1.3 (both edit different files post-relocation).
- D1.2 (magefile.go) and D1.3 (ui/main.go) edit disjoint files, both depend on D1.1 alone — they MAY run in parallel.
- D1.3 unblocks D1.4 (same file `ui/main.go`, serialize).
- D1.5 (frontend code) waits on D1.2 (mage target points at relocated tree) AND D1.4 (JS-side binding must exist). D1.5 does NOT directly depend on D1.3 — the dependency is transitive through D1.4.
- D1.6 (magefile.go + README) waits on D1.2 (same file `magefile.go`, serialize) AND D1.5 (so the README's wiring claims are evidence-backed). D1.6 does NOT directly depend on D1.3 or D1.4 — those are transitive through D1.5.

Mirrored into `_BLOCKERS.toml` alongside this file. No `blocked_by` edges changed in round 2 — only the diagram was redrawn to remove the misleading `D1.4 ──┘ └─▶ D1.6` indentation and to accurately show D1.6's two-edge dependency on D1.2 + D1.5.

### N9 — Package manager decision (round-2 F2 resolution): pin pnpm via `packageManager`, commit lockfile

Round-1 falsification F2 surfaced a CONFIRMED blocker: `REVISION_BRIEF.md` §1 reads "Node + npm available on dev machine" but `wails.json` (already on disk from prior bootstrap commits `a9bac6c` / `8d33539`) hardwires `"frontend:install": "pnpm install"` + `"frontend:build": "pnpm run build"` + `"frontend:dev:watcher": "pnpm run dev"`, the prior `frontend/package.json` carries no `packageManager` field, and there is no `pnpm-lock.yaml` on disk.

**Dev decision (2026-05-18, round-2 brief):** pnpm + pin, NOT switch-to-npm. The prior bootstrap commits chose pnpm; this drop honors that choice and pins it deterministically.

D1.1 ships two coupled edits to resolve F2:

1. **Pin pnpm.** `ui/frontend/package.json` gains a `"packageManager": "pnpm@9.0.0"` field. Node 16.13+ honours `packageManager` as a Corepack hint — `corepack enable` (one-time dev-machine setup) ensures `pnpm` is auto-fetched at the pinned version regardless of what the dev manually installed. Removes the "dev has npm but plan says pnpm" environment gap.
2. **Commit `pnpm-lock.yaml`.** After `pnpm install` runs (during D1.1 acceptance), the resulting `ui/frontend/pnpm-lock.yaml` is `git add`'d and committed alongside the rest of D1.1's relocation. Reproducible installs require a lockfile; absent the lock, every fresh `pnpm install` re-resolves the dependency graph from scratch, drifting silently. `.gitignore` must NOT exclude `ui/frontend/pnpm-lock.yaml` — D1.1 acceptance asserts this.

**Brief-vs-plan drift resolved.** `REVISION_BRIEF.md` §1 was reconciled by the orchestrator after round-2 planning to read "Node + pnpm available on dev machine. Plan pins `packageManager: \"pnpm@9.0.0\"` in `frontend/package.json` and commits `pnpm-lock.yaml` per Round 2 PLAN.md §N9 decision." Brief and plan now agree.

### N10 — Path-relative-resolution relocation traps (round-1 F2-proof / F1-fals resolution + D1.1-build symlink discovery)

Three traps share a structural shape: a directive/string holds a relative path that resolves against SOMETHING-other-than-the-module-root, and moving the holder file silently breaks resolution unless the path string is recomputed for the new depth.

**Variant 1 — `//go:embed` directives (Go).** `main.go` line 16 declares `//go:embed all:frontend/dist`. Go resolves `//go:embed` paths **relative to the source file's directory**. After `git mv main.go ui/main.go` + `git mv frontend ui/frontend`, the directive in `ui/main.go` resolves `frontend/dist` against `ui/`'s file directory → `ui/frontend/dist`, which is correct. A well-meaning builder reading the relocation diff may "fix" the directive to `//go:embed all:ui/frontend/dist` to "match the new path". That rewrite breaks the embed (Go would then look for `ui/ui/frontend/dist` relative to `ui/main.go`). The directive must stay **literally unchanged** across D1.1's relocation AND across D1.3's `NewApp(nil)` → real-service rewrite AND across D1.4's `App.ListProjects` method addition. D1.1 and D1.3 acceptance both carry `grep -q '^//go:embed all:frontend/dist' ui/main.go` exits 0.

**Variant 2 — Relative symlinks (filesystem).** A relative-target symlink resolves against the symlink's OWN directory, not the source-file's directory and not the module root. `frontend/public/.tillsyn-bindings.json` was a symlink with target string `../../.tillsyn/bindings.json` — from `frontend/public/` that resolves to repo-root `.tillsyn/bindings.json`. After `git mv frontend/public/ ui/frontend/public/`, the same target string resolves to `ui/.tillsyn/bindings.json` (one level deeper), which doesn't exist. Fix: recompute the target as `../../../.tillsyn/bindings.json` (one extra `../` per added directory depth). Discovered during D1.1 round-1 build when Vite's `prepareOutDir` `statSync`'d every `public/` entry before clearing `dist/` and hard-failed on the broken symlink. If `ln -s` is sandboxed, the equivalent fix goes through git plumbing: `git hash-object -w --stdin` to write the new target string as a blob, then `git update-index --add --cacheinfo 120000,<sha>,<path>` to install it as a symlink mode entry.

**Variant 3 — Config-file relative paths (Astro / Vite / tsconfig).** `astro.config.mjs` and `tsconfig.json` may carry relative paths (`./src`, `./public`) that resolve against the config file's location. After relocation those typically remain correct (the relocation preserves the relative tree structure around the config file). But ABSOLUTE-from-repo-root paths in code or config (e.g. `publicDir: "/frontend/public"` if such ever existed) would silently break — the resolution baseline changed. D1.1 acceptance covers this indirectly via `pnpm run build` exit 0: a broken absolute path would fail the build. Future FE drops adding such paths should bracket the absolute-vs-relative choice in worklog notes.

**General rule:** any string holding a relative path resolves against SOMETHING — file dir, working dir, config dir, symlink dir. Whenever a holder file moves, audit every relative-path-bearing string for the new depth. The acceptance grep guards above are belt-and-suspenders against well-meaning rewrites; the variant 2 fix above is the corrective action when discovery happens at build time.

### N11 — Wails CLI version is dev-machine-controlled; only the Go binding pin is contractual (round-1 F5-fals resolution)

`go.mod:93` pins `github.com/wailsapp/wails/v2 v2.12.0` — that's the Go-bindings module version this codebase compiles against, fully reproducible across machines via `go.sum`. The `wails` CLI binary itself is installed via `go install github.com/wailsapp/wails/v2/cmd/wails@latest` (or a specific version) into the dev's `$GOBIN` / `$PATH`; there is no `.wails-version` file, no project-local CLI version pin, no Corepack-equivalent mechanism for Go CLI tools.

**The contract this drop depends on is the Go binding pin in `go.mod`, NOT the dev's installed CLI version.** Round-1 acceptance bullets that grep stdout for `Wails CLI v2.12.0` are removed in round 2:

- D1.3 acceptance: drops the CLI version-string match entirely; uses `wails build` exit-code 0 + binary file existence + Mach-O `file` check.
- D1.6 acceptance: greps stdout only for the literal `[Wails] Dev mode` marker (a Wails-runtime emission unaffected by CLI version), with a 30s timeout window (raised from round-1's 10s — see §N12).

The Wails CLI version contract is documentary, not machine-checked: the dev's installed CLI should be a reasonably recent v2.x — version drift in the CLI rarely breaks codegen against a fixed Go-bindings pin, and when it does, the failure surfaces as `wails build` non-zero exit, which D1.3 + D1.6 acceptance catches.

If a future drop discovers CLI/binding-version drift causing real codegen issues, the response is to either (a) re-pin `go.mod` to match the dev's CLI, or (b) document a "tested CLI versions" range in `ui/README.md`. Not to grep stdout strings.

### N12 — Smoke-test 60s timeout (round-2 F4-fals resolution)

Round-1 D1.6 acceptance set a 10s timeout. Round-1 falsification F9 raised it to 30s. Round-2 falsification F4 measured the cold-cache sum at 22-64s (`pnpm install` 8-30s + Astro start 3-6s + Go bridge build 10-25s + WebView launch 1-3s) — 30s sits in the middle of that range and flakes. Round 2 raises to **60s** to cover the upper end of a worst-case cold-cache machine. 60s is the smoke-vs-build cap: past 60s the gate stops measuring "did dev mode start" and starts measuring "did the whole world rebuild." If a future drop's smoke check flakes at 60s, the diagnostic is dev-cache hygiene OR a real Wails/Astro regression, not loosening the bound further.

### N13 — Binary capitalization (round-2 F1-fals resolution)

Round-2 falsification F1 surfaced a CONFIRMED blocker: `wails.json` (carried in via the prior bootstrap commits) declares `"outputfilename": "tillsyn"` (lowercase) — so `wails build` emits `tillsyn.app/Contents/MacOS/tillsyn` by default. The round-1+round-2 plan acceptance asserts `Tillsyn.app/.../Tillsyn` (capital T). Mismatch.

**Dev decision (2026-05-18, post-round-2): option (b) — bump `wails.json` `outputfilename` to `"Tillsyn"`.** Best-practice for desktop apps is to match the bundle name to the app's display name (Discord, Slack, etc. all ship capitalized `.app` bundles). The `till` CLI stays lowercase per Unix convention; the desktop app is `Tillsyn.app` for branding consistency with the project name. D1.1 paths + acceptance updated to perform the edit + verify via grep.

`wails.json:4` `"name": "Tillsyn"` is the `CFBundleName` (display name in Finder/Launchpad) and was already capitalized — the bug was only in `outputfilename`.
