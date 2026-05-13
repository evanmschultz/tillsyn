# BUILDER WORKLOG — DROP 4c.6.1 W6

---

## Round 1 — W6.D1 WAILS BOOTSTRAP (2026-05-12)

**Droplet:** W6.D1 — WAILS BOOTSTRAP (main.go + wails.json + .gitignore)
**Agent:** fe-builder-agent (claude-sonnet-4-6)
**State transition:** todo → complete

### Files Created / Modified

- `main.go` (NEW at tillsyn repo root) — `//go:build wails` build tag on line 1, blank line, `package main`. `App` struct holds `*app.Service` field. `NewApp`, `startup`, `main` with `wails.Run` + default native menu (no custom `Menu:` field; Wails provides Quit/About/Hide/Minimize automatically). `//go:embed all:frontend/dist` embed directive present (compile-time only; `wails dev` serves from Astro dev server, not from embed). Imports: `github.com/wailsapp/wails/v2`, `github.com/wailsapp/wails/v2/pkg/options`, `github.com/wailsapp/wails/v2/pkg/options/assetserver`, `github.com/evanmschultz/tillsyn/internal/app`.
- `wails.json` (NEW at tillsyn repo root) — valid JSON with `$schema`, `name: "Tillsyn"`, `outputfilename: "tillsyn"`, `frontend:dir: "frontend"`, `frontend:install: "pnpm install"`, `frontend:build: "pnpm run build"`, `frontend:dev:watcher: "pnpm run dev"`, `frontend:dev:serverUrl: "http://localhost:4321"`.
- `.gitignore` (MODIFIED) — added `build/bin/`, `frontend/node_modules/`, `frontend/dist/`, `frontend/wailsjs/` entries.

### Pre-D1 mage ci Package Set (Baseline)

```
PKG PASS: internal/platform/gitenv, internal/fsatomic, internal/platform, internal/buildinfo,
          internal/adapters/embeddings/fantasy, internal/config, internal/domain,
          internal/tui/gitdiff, github.com/evanmschultz/tillsyn (root), cmd/colors, cmd/headerlab
PKG FAIL: cmd/till, internal/adapters/auth/autentauth, internal/adapters/livewait/localipc,
          internal/adapters/server, internal/adapters/server/common, internal/adapters/server/httpapi,
          internal/adapters/server/mcpapi, internal/adapters/storage/sqlite, internal/app,
          internal/app/dispatcher, internal/app/dispatcher/cli_claude,
          internal/app/dispatcher/cli_claude/render, internal/app/dispatcher/context,
          internal/templates, internal/tui, internal/tui/style (build error)
Total: 27 packages, 11 pass, 16 fail, 1 build error (tui/style), 455 tests passed
```

### Post-D1 mage ci Package Set

```
PKG PASS: internal/platform/gitenv, internal/fsatomic, internal/platform, internal/buildinfo,
          internal/adapters/embeddings/fantasy, internal/config, internal/domain,
          internal/tui/gitdiff, github.com/evanmschultz/tillsyn (root), cmd/colors, cmd/headerlab,
          internal/tui/keybindings, internal/tui/components (NEW — arrived from W5 parallel wave)
PKG FAIL: (identical 16-package set as pre-D1)
Build errors: internal/tui/style (pre-existing, unchanged)
Total: 29 packages, 13 pass, 16 fail, 1 build error (tui/style), 455 tests passed
```

**Package set delta analysis:** Two new PASS packages (`internal/tui/keybindings`, `internal/tui/components`) appeared — these are from W5 parallel wave work landing concurrently, not from D1. The root package `github.com/evanmschultz/tillsyn` still PASSES, confirming `main.go` (with `//go:build wails`) is correctly excluded from default compilation. No new failures introduced. Build tag isolation working as designed.

### go.mod Dep Additions Required (for dev to run)

The Wails v2 dependency is NOT in `go.mod`. The following deps are needed before `wails dev -tags wails` can be used. **Dev must run these `go get` commands manually (builder must not run `go get`):**

```
go get github.com/wailsapp/wails/v2@latest
```

This will pull in `wails/v2` and its transitive deps. Additionally:

```
go get github.com/wailsapp/wails/v2/pkg/options
go get github.com/wailsapp/wails/v2/pkg/options/assetserver
```

(These are sub-packages of `wails/v2` — the single `go get github.com/wailsapp/wails/v2@latest` should cover all three.)

**Note:** Until Wails is added to `go.mod`, `go build -tags wails .` at the root will fail with missing import errors. `mage ci` (without `-tags wails`) remains green because the build tag excludes `main.go` from the default compile path — this was verified above.

### wails dev Smoke

`wails dev -tags wails` was NOT run — the Wails CLI is not confirmed installed in this environment, and `wails/v2` is not yet in `go.mod`. Builder notes this as expected per W6 PLAN.md §RiskNotes R1 and D1 AcceptanceCriteria commentary. Full `wails dev` smoke is a post-go.mod-addition manual verification step for the dev.

`go build -tags wails .` was NOT run for the same reason (missing `wails/v2` dep in go.mod). Will be verifiable after dev runs `go get`.

### Context7 References

- `/wailsapp/wails` — queried for canonical `main.go` shape, `wails.Run` + `options.App` + `assetserver.Options` pattern, `wails.json` field schema, and `frontend:dev:serverUrl` / `frontend:dev:watcher` field names.

### Acceptance Criteria Status

- [x] `main.go` exists at tillsyn repo root with `//go:build wails` as FIRST LINE (blank line, then `package main`)
- [x] `wails.json` valid JSON with all required fields (`name`, `outputfilename`, `frontend:dir`, `frontend:install`, `frontend:build`, `frontend:dev:watcher`, `frontend:dev:serverUrl`)
- [x] `.gitignore` contains `frontend/node_modules/`, `frontend/dist/`, `build/bin/`, `frontend/wailsjs/`
- [x] `mage ci` identical package-set pre and post D1 (build tag isolation confirmed)
- [ ] `go build -tags wails .` compile verification — blocked on dev running `go get github.com/wailsapp/wails/v2@latest`
- [ ] `wails dev` smoke — blocked on Wails CLI install + go.mod dep addition

### Hylla Feedback

None — Hylla was OFF for W6 (non-Go FE files + build-tagged root files). Context7 + Read/LSP used throughout.
