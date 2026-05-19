# DROP_FE_1_BOOTSTRAP — Builder QA Proof

Append a `## Droplet N.M — Round K` section per QA attempt. See `workflow/example/drops/WORKFLOW.md` § "Phase 5 — Build QA" for what each section should contain.

## Droplet 1.1 — Round 1

- **Reviewer:** `fe-qa-proof-agent`
- **Verdict:** **PASS**
- **Date:** 2026-05-18
- **Scope:** D1.1 acceptance bullets 1-12 (QA-agent-executable subset); bullet 13 (`mage format`) + bullet 14 (`mage ci`) **deferred** per spawn-prompt rule (parallel-builder WIP in `internal/` would falsely contaminate `mage ci`); confirmed deferred-not-failed.
- **Files reviewed:** `ui/wails.json`, `ui/main.go`, `ui/frontend/package.json`, `.gitignore`, `ui/.gitignore`, `ui/frontend/public/.tillsyn-bindings.json` (symlink), `magefile.go` (verifySources), `BUILDER_WORKLOG.md`, `PLAN.md` (acceptance bullets), `git status --porcelain`.

### Premises

P1. `ui/wails.json` exists, has `"outputfilename": "Tillsyn"` (capital T), `frontend:dir: "frontend"`, `frontend:dev:serverUrl: "http://localhost:4321"`.
P2. `ui/main.go` exists, line 1 is `//go:build wails`, line 16 reads literally `//go:embed all:frontend/dist` (NOT `all:ui/frontend/dist`).
P3. `ui/frontend/package.json` has `"packageManager": "pnpm@9.0.0"` field.
P4. `ui/frontend/pnpm-lock.yaml` is git-tracked AND not gitignored.
P5. `.gitignore` excludes `ui/build/`, `ui/frontend/node_modules/`, `ui/frontend/dist/`, `ui/frontend/.astro/`, `ui/frontend/wailsjs/`; stale `frontend/*` entries are gone.
P6. `ui/.gitignore` exists with Wails artefact patterns.
P7. Original paths (`frontend/`, `main.go`, `wails.json`) are no longer git-tracked at repo root.
P8. `ui/frontend/public/.tillsyn-bindings.json` is a symlink resolving to `../../../.tillsyn/bindings.json` (three `../`) and through-symlink read returns the real `.tillsyn/bindings.json` byte count (1113).
P9. `cd ui/frontend && pnpm install` exits 0 with lockfile up to date.
P10. `cd ui/frontend && pnpm run build` exits 0 (pipeline runs to completion per the tightened acceptance bullet 11; `dist/index.html` is NOT expected — D1.5 adds `src/pages/index.astro`).
P11. `magefile.go verifySources()` (L290-295) does NOT reference root `main.go` — guard is unaffected by relocation.
P12. `git status --porcelain` scope is clean for D1.1 paths (only declared moves + edits); sibling `internal/` WIP is expected per parallel-builders directive, not a D1.1 scope violation.

### Evidence

E1. `Read ui/wails.json`:
```
"name": "Tillsyn",
"outputfilename": "Tillsyn",
"frontend:dir": "frontend",
"frontend:install": "pnpm install",
"frontend:dev:serverUrl": "http://localhost:4321"
```
→ P1 verified.

E2. `Read ui/main.go` lines 1-17:
```
//go:build wails
package main
import ( ... "github.com/evanmschultz/tillsyn/internal/app" "github.com/wailsapp/wails/v2" ... )
//go:embed all:frontend/dist
var assets embed.FS
```
→ P2 verified. Build tag preserved, embed directive verbatim, NOT rewritten to `all:ui/frontend/dist` (§N10 trap variant 1 avoided).

E3. `Read ui/frontend/package.json` line 6: `"packageManager": "pnpm@9.0.0"` → P3 verified.

E4. `git ls-files ui/frontend/pnpm-lock.yaml` → `ui/frontend/pnpm-lock.yaml` (tracked). `Read .gitignore`: no `ui/frontend/pnpm-lock.yaml` exclusion line. → P4 verified.

E5. `Read .gitignore` lines 38-43:
```
# Wails build outputs and generated bindings (Drop 4c.6.1 W6.D1; relocated under ui/ at drop_fe_1_bootstrap D1.1)
ui/build/
ui/frontend/node_modules/
ui/frontend/dist/
ui/frontend/.astro/
ui/frontend/wailsjs/
```
No `frontend/*` patterns elsewhere in file. → P5 verified.

E6. `Read ui/.gitignore`:
```
build/bin/
build/darwin/
build/windows/
build/linux/
```
Plus comment lines explaining pnpm-lock.yaml is NOT ignored. → P6 verified.

E7. `git ls-files frontend/ main.go wails.json` → empty output → P7 verified.

E8. `readlink ui/frontend/public/.tillsyn-bindings.json` → `../../../.tillsyn/bindings.json` (three `../` — correct for `ui/frontend/public/` → repo-root depth). `cat ui/frontend/public/.tillsyn-bindings.json | wc -c` → `1113`. `cat .tillsyn/bindings.json | wc -c` → `1113`. Through-symlink read returns identical byte count to the real file. `stat -L` shows `1113 bytes` for the symlink-resolved path. → P8 verified. §N10 variant 2 trap (the symlink relocation trap) is correctly fixed.

E9. `pnpm --dir ui/frontend install` output: `Lockfile is up to date, resolution step is skipped` / `Already up to date` / `Done in 320ms`. Exit 0. → P9 verified.

E10. `pnpm --dir ui/frontend run build` output: `[build] ✓ Completed` / `[vite] ✓ built in 242ms` / `[build] 0 page(s) built` / `[build] Complete!`. Exit 0. Astro produces `dist/_astro/client.CJIGP0w7.js` (16 KB) and copies `public/` assets (including the symlink-resolved `.tillsyn-bindings.json` at 1.1K) into `dist/`. → P10 verified (and the acceptance bullet 11 tightening fits exactly — "build pipeline runs to completion on the relocated tree", no `dist/index.html` expected).

E11. `magefile.go:290-295`:
```go
// verifySources ensures the required automation and CLI entrypoint sources are still tracked.
func verifySources() error {
    printer := newMagePrinter()
    _, err := captureCommandWithProgress(printer, "Verifying tracked sources", "Verified tracked sources", "git", "ls-files", "--error-unmatch", "magefile.go", "cmd/till/main.go", "cmd/till/main_test.go")
    return err
}
```
References `magefile.go`, `cmd/till/main.go`, `cmd/till/main_test.go` only. No reference to root `main.go`. → P11 verified.

E12. `git status --porcelain ui/ frontend/ main.go wails.json .gitignore`:
```
M  .gitignore
D  frontend/public/.tillsyn-bindings.json
A  ui/.gitignore
R  frontend/astro.config.mjs -> ui/frontend/astro.config.mjs
R  frontend/package.json -> ui/frontend/package.json
A  ui/frontend/pnpm-lock.yaml
A  ui/frontend/public/.tillsyn-bindings.json
R  frontend/public/stil-baseline.json -> ui/frontend/public/stil-baseline.json
R  frontend/public/stil-tokens.css -> ui/frontend/public/stil-tokens.css
R  frontend/src/env.d.ts -> ui/frontend/src/env.d.ts
R  frontend/src/layouts/MainLayout.astro -> ui/frontend/src/layouts/MainLayout.astro
R  frontend/tests/migration-markers.test.ts -> ui/frontend/tests/migration-markers.test.ts
R  frontend/tsconfig.json -> ui/frontend/tsconfig.json
R  main.go -> ui/main.go
R  wails.json -> ui/wails.json
```
All within declared D1.1 paths. Sibling `internal/domain/comment.go`, `internal/adapters/mcp_rpc/extended_tools.go`, `internal/app/dispatcher/*_test.go` are dirty per spawn-prompt parallel-builder warning — expected, not D1.1 scope violation. → P12 verified.

### Trace or cases

Acceptance bullet → evidence mapping (PLAN.md D1.1 lines 49-62):

| # | Bullet | Verified | Evidence |
|---|---|---|---|
| 1 | `git ls-files ui/main.go ui/wails.json ui/frontend/astro.config.mjs ui/frontend/package.json` returns all 4 | YES | `git ls-files` returned all 4 paths |
| 2 | `git ls-files frontend/ main.go wails.json` empty | YES | empty output |
| 3 | `cat ui/wails.json` shows `frontend:dir: "frontend"` + `frontend:dev:serverUrl: http://localhost:4321` | YES | E1 |
| 4 | `grep -q '"outputfilename": "Tillsyn"' ui/wails.json` exits 0 | YES | E1 |
| 5 | `head -1 ui/main.go` shows `//go:build wails` | YES | E2 |
| 6 | `grep -q '^//go:embed all:frontend/dist' ui/main.go` exits 0 | YES | E2 — directive verbatim, not rewritten |
| 7 | `grep -q '"packageManager": "pnpm@' ui/frontend/package.json` exits 0 | YES | E3 |
| 8 | `pnpm install` exits 0 and produces `node_modules/` + lockfile | YES | E9 |
| 9 | `git ls-files ui/frontend/pnpm-lock.yaml` returns path | YES | E4 |
| 10 | `grep -q 'ui/frontend/pnpm-lock.yaml' .gitignore` exits non-zero | YES | E4 — no match in `.gitignore` |
| 11 | `pnpm run build` exits 0 (tightened: pipeline-completes, no `dist/index.html` expected pre-D1.5) | YES | E10 — `[build] Complete!` exit 0 |
| 12 | `magefile.go verifySources()` does not reference root `main.go` | YES | E11 |
| 13 | `mage format` reports no diff against `ui/main.go` | DEFERRED | Per spawn-prompt rule; parallel-builder WIP would contaminate |
| 14 | `mage ci` still passes green | DEFERRED | Per spawn-prompt rule; drop-end gate |

Additional verifications beyond the literal bullets:

- Symlink trap fix (§N10 variant 2): `readlink` shows three `../`; through-symlink byte count matches real file → E8.
- Scope cleanliness vs sibling WIP: only declared D1.1 paths show in `git status --porcelain` for the declared scope → E12.
- `//go:embed` trap (§N10 variant 1): directive remains literally `all:frontend/dist` — Go resolves relative to file's dir → resolves to `ui/frontend/dist`, which is where Astro writes its output. Correct.

### Conclusion

**PASS.** All 12 QA-agent-executable acceptance bullets verified. Bullets 13 (`mage format`) and 14 (`mage ci`) are properly deferred per the parallel-builder spawn-prompt rule, not failed.

The builder's claims in `BUILDER_WORKLOG.md` are accurate:
- `git mv` relocations preserved as renames in `git status`.
- `outputfilename` bump to `Tillsyn` landed.
- `packageManager: "pnpm@9.0.0"` pin landed.
- `pnpm-lock.yaml` is committed (not gitignored).
- `//go:embed all:frontend/dist` directive preserved verbatim (§N10 variant 1 trap avoided).
- Symlink at `ui/frontend/public/.tillsyn-bindings.json` correctly re-pointed with three `../` (§N10 variant 2 trap fixed); through-symlink read returns the real bindings.json byte count (1113).
- `pnpm install` and `pnpm run build` both exit 0 on the relocated tree.
- Acceptance bullet 11 tightening (drop `dist/index.html` expectation pre-D1.5) is appropriate — the bootstrap tree carries no `src/pages/` yet; D1.5 will add `src/pages/index.astro` and `dist/index.html` materializes then.

The §N10 §"variant 2 — relocation traps that move a path-bearing file deeper" expansion in PLAN.md is well-warranted by the actual symlink trap encountered; the orchestrator's edit to §N10 reflects load-bearing learning.

### Unknowns

U1. **pnpm 10.11.0 ran instead of pinned 9.0.0.** `pnpm --dir ui/frontend install` was executed by the dev-machine's system pnpm v10.11.0 (per the final `Done in 320ms using pnpm v10.11.0` line). The `packageManager: "pnpm@9.0.0"` field is honored by Corepack-aware Node when `corepack enable` has been run; that does not appear to have been done on this dev machine. The install completed successfully — the lockfile produced by the prior pnpm 9.0.0 run (during builder execution) is still v9 format and pnpm 10 read it fine. This is consistent with §N9's intent ("developers on system-pnpm-10.x without Corepack enabled may see a friendly pin warning but the install still completes"). **Not a D1.1 failure** — N9 anticipates this case — but worth flagging for §N7 (dev-machine prerequisite check) on the next FE drop: the orch should verify `corepack enable` has been run on the dev machine before D1.6's `mage ui-dev` smoke test, or pin-enforcement may drift further. Route: include in CLOSEOUT.md refinements list.

U2. **`mage ci` not run in QA.** Per spawn-prompt rule, `mage ci` and `mage format` are deferred to drop-end CI. The proof of "D1.1 doesn't break the Go side" is structural — `//go:build wails` keeps `ui/main.go` out of the default Go build, and `verifySources()` doesn't reference root `main.go`. The actual `mage ci` verdict won't be known until drop-end after sibling-builder commits land. Route: drop-end CI gate.

U3. **Symlink-as-`git update-index` blob tracking.** The builder used `git update-index --add --cacheinfo 120000,<blob>,<path>` to add the symlink because `ln -s` was unavailable in the environment. `git status` records this as a `D` + `A` pair rather than a clean rename. Functionally equivalent (the symlink would also have shown content-edit + rename even via `ln -s`), but slightly worse `git log --stat` readability. Route: documentary note only — no D1.1 failure.

### Hylla Feedback

N/A — FE droplet, Hylla is Go-only per the current `feedback_hylla_go_only_today.md` rule (also: Hylla is OFF entirely per `feedback_hylla_disabled_for_now.md` 2026-05-18). No Hylla queries attempted, no fallback recorded. `Read`, `Bash` (rtk-proxied grep + readlink + pnpm + git ls-files + wc), and `Edit`/`Write` were the right primitives.

## Droplet 1.2 — Round 1

- **Reviewer:** `fe-qa-proof-agent`
- **Verdict:** **PASS**
- **Date:** 2026-05-18
- **Scope:** D1.2 acceptance bullets 1, 2, 4 (rename + alias map + literal-substring policy); bullet 3 (`mage ci` continues green) **deferred** per spawn-prompt directive (parallel-builder WIP in `internal/adapters/mcp_rpc/`, `internal/app/dispatcher/`, `ui/main.go` would falsely contaminate); confirmed deferred-not-failed.
- **Files reviewed:** `magefile.go` (full read, lines 1-707), `BUILDER_WORKLOG.md` § "Droplet 1.2 — Round 1", `git status --porcelain magefile.go workflow/drop_fe_1_bootstrap/`.

### Premises

P1. `func CiUI() error` exists in `magefile.go` with body running `pnpm run test:unit` then `pnpm run build` inside `ui/frontend/`.
P2. `func CiFe` does NOT exist anywhere in `magefile.go`.
P3. `Aliases` map contains `"ci-ui": CiUI,` entry, positioned alphabetically.
P4. `CiUI` doc comment + stage titles reference `ui/` (or `ui/frontend/`), not bare `frontend/`.
P5. `mage -l` output lists `ciUI` and does NOT list `ciFe`.
P6. `mage -h ciUI` shows `Aliases: ci-ui`.
P7. `mage -h ciFe` and `mage -h ci-fe` return exit 2 with "Unknown target".
P8. `mage ciUI` exits 0; vitest runs in `ui/frontend/`, astro build completes.
P9. Bare `"frontend"` substring (outside `ui/frontend` token) does NOT appear in `magefile.go`. Remaining `frontend` occurrences are confined to the doc comment + the consolidated `"ui/frontend"` path token.
P10. `git status --porcelain magefile.go workflow/drop_fe_1_bootstrap/` shows only D1.2-scoped modifications.

### Evidence

E1. `Read magefile.go` lines 232-264 verify the `CiUI` function:
- L232-234 doc comment: `// CiUI runs the UI continuous-integration gate: Vitest unit tests followed / // by an Astro static build, both executed inside the `ui/frontend/` directory. / // Playwright e2e tests are excluded — those run via MCP during QA agent passes.`
- L235: `func CiUI() error {`
- L237-240: `wd, err := os.Getwd(); ...` cwd resolution preserved.
- L241: `uiDir := filepath.Join(wd, "ui/frontend")` — single consolidated path token.
- L246-250: stage `{title: "UI Unit Tests", run: func() error { return runCommandInDir(uiDir, "pnpm", "run", "test:unit") }}`.
- L252-256: stage `{title: "UI Build", run: func() error { return runCommandInDir(uiDir, "pnpm", "run", "build") }}`.
→ P1, P4 verified.

E2. Full `Read magefile.go` (707 lines) shows no `func CiFe` declaration anywhere — only `CiUI` exists. → P2 verified.

E3. `Read magefile.go` lines 26-37 (Aliases map):
```
var Aliases = map[string]interface{}{
    "check":              CI,
    "ci-ui":              CiUI,
    "dev":                Dev,
    "test-golden":        TestGolden,
    ...
}
```
→ P3 verified. `"ci-ui": CiUI,` sits between `"check"` and `"dev"` — alphabetical placement.

E4. `mage -l` output (captured):
```
ci*                 runs the canonical full gate.
ciUI                runs the UI continuous-integration gate: Vitest unit tests
                    followed by an Astro static build, both executed inside the
                    'ui/frontend/' directory.
```
No `ciFe` row appears. → P5 verified.

E5. `mage -h ciUI` output (captured):
```
CiUI runs the UI continuous-integration gate: Vitest unit tests followed by an
Astro static build, both executed inside the 'ui/frontend/' directory.
Playwright e2e tests are excluded — those run via MCP during QA agent passes.

Usage:

    mage ciui

Aliases: ci-ui
```
→ P6 verified — `Aliases: ci-ui` line present.

E6. `mage -h ci-fe` returned exit 2 with stderr `Unknown target: "ci-fe"`. `mage -h ciFe` returned exit 2 with stderr `Unknown target: "cife"` (mage lowercases for resolution). → P7 verified.

E7. `mage ciUI` execution (captured tail):
```
UI Unit Tests
> tillsyn-fe@0.0.1 test:unit /Users/evanschultz/Documents/Code/hylla/tillsyn/main/ui/frontend
> vitest run --passWithNoTests

 RUN  v3.2.4 /Users/evanschultz/Documents/Code/hylla/tillsyn/main/ui/frontend
 ↓ tests/migration-markers.test.ts (2 tests | 2 skipped)
 Test Files  1 skipped (1)

UI Build
> tillsyn-fe@0.0.1 build /Users/evanschultz/Documents/Code/hylla/tillsyn/main/ui/frontend
> astro build
...
21:40:41 [build] Complete!
```
Working directory `.../main/ui/frontend` (NOT bare `frontend/`); both stages completed cleanly; mage process exited 0 (tail captured success-line, no error annotation). → P8 verified.

E8. Static substring read of `magefile.go` via full-file Read shows two `frontend` occurrences total:
- Line 233 (doc comment): `... inside the `ui/frontend/` directory.` — `frontend` appears only inside `ui/frontend/` token. Explicitly permitted by the acceptance bullet ("doc comment that points to `ui/frontend`").
- Line 241 (path construction): `uiDir := filepath.Join(wd, "ui/frontend")` — `frontend` appears only inside `"ui/frontend"` string literal token.
No bare `"frontend"` (with surrounding double quotes and no `ui/` prefix) appears anywhere in the file. The builder consolidated `filepath.Join(wd, "ui", "frontend")` → `filepath.Join(wd, "ui/frontend")` precisely to eliminate the standalone `"frontend"` string. → P9 verified.

E9. `git status --porcelain magefile.go workflow/drop_fe_1_bootstrap/`:
```
 M magefile.go
 M workflow/drop_fe_1_bootstrap/BUILDER_WORKLOG.md
 M workflow/drop_fe_1_bootstrap/PLAN.md
```
Only the three D1.2-scoped files are modified. Sibling-builder WIP (`ui/main.go`, `internal/adapters/mcp_rpc/`, `internal/app/dispatcher/`) lives outside this scoped filter — confirms D1.2 has not strayed. → P10 verified.

### Trace or cases

T1. **Function rename completeness.** `CiUI` is the only `Ci*` function with FE/UI semantics. The doc comment (L232-234), function signature (L235), body's path variable name (`uiDir`, L241), stage titles (`"UI Unit Tests"` L247, `"UI Build"` L253) all use the new `ui` / `UI` naming. No `Fe`/`fe` token survives in this function's source span. The legacy `CiFe` symbol is fully absent (E2).

T2. **Alias registration.** Aliases map is keyed by hyphenated alias → canonical function reference. `"ci-ui": CiUI,` is the sole `ci-*` entry; the old `"ci-fe"` is gone. Alphabetical placement between `"check"` and `"dev"` matches the existing alias-map convention (`"format-path"`, `"test-pkg"`, etc. are similarly clustered by prefix). Functional verification: `mage -h ciUI` reports `Aliases: ci-ui` (E5) — mage's reflective alias-discovery successfully sees the new map entry.

T3. **Legacy-target negative path.** Two distinct legacy names checked: `ciFe` (the old canonical) and `ci-fe` (the old alias). Both return exit 2 + "Unknown target" (E6). This proves no shim/back-compat layer survived — old call sites would fail loudly rather than silently no-op. The builder worklog corroborates: "Renaming therefore needed only the single function definition + alias-map entry. No cascading edits required" (no callers of `CiFe` existed inside the `CI` aggregate, so removing the symbol leaves no compile-time hole).

T4. **Runtime execution path.** `mage ciUI` enters `runStage(printer, "UI Unit Tests", run)` → `runCommandInDir(uiDir, "pnpm", "run", "test:unit")` (L249). `uiDir = filepath.Join(wd, "ui/frontend")` resolves to absolute `.../main/ui/frontend`. vitest output confirms the working directory at the top of its banner (E7) — `RUN  v3.2.4 /Users/evanschultz/Documents/Code/hylla/tillsyn/main/ui/frontend`. Both stages succeed; mage exits 0 implicitly via tail-line `[build] Complete!` and no error wrap.

T5. **Literal-substring sweep.** The acceptance bullet 4 enforces a strict no-bare-`"frontend"` policy. Full-file Read confirmed exactly two `frontend` occurrences, both qualified within the `ui/frontend` token (doc comment at L233 and string literal at L241). The builder's deliberate consolidation `filepath.Join(wd, "ui", "frontend")` → `filepath.Join(wd, "ui/frontend")` (recorded in BUILDER_WORKLOG.md "Notes" entry on bullet 4) eliminates the only remaining standalone `"frontend"` quote-delimited token. Cross-platform safety: `filepath.Join` normalizes embedded forward-slash on Windows, so this is portable.

T6. **Scope-clean signal.** `git status --porcelain magefile.go workflow/drop_fe_1_bootstrap/` (E9) returns exactly the three expected modifications (the magefile + worklog + plan). Sibling-builder WIP outside this filter is not D1.2's responsibility per spawn-prompt parallel-builder rule.

### Conclusion

**PASS.** All proof premises verified by direct evidence. D1.2 cleanly renamed `CiFe` → `CiUI`, registered the `ci-ui` alias, eliminated the bare `"frontend"` substring, and proved runtime correctness via `mage ciUI` execution (exit 0, vitest cwd = `ui/frontend`, astro build complete). Bullet 3 (`mage ci`) is explicitly deferred per spawn-prompt directive — that deferral is correct (parallel-builder WIP would otherwise falsely fail the gate); the deferral itself is not a failure mode.

### Unknowns

U1. **`mage ci` not run.** Per spawn-prompt rule, `mage ci` is deferred to drop-end CI after all parallel builders complete and commits land. The D1.2-only proof of "Go side unaffected" is structural: `CiUI` is invoked only when called explicitly (it is NOT part of the `CI` aggregate at L213-229, which runs `verifySources` / `formatCheck` / `coverage` / `Build` / `TestIntegration`). Renaming a non-aggregate target cannot break the aggregate gate. Route: drop-end CI verifies.

U2. **Alphabetical placement is a builder stylistic choice.** The acceptance bullet doesn't specify ordering. The builder placed `"ci-ui"` between `"check"` and `"dev"` for grouping with future `ci-*` aliases. No QA action required; documentary note only.

U3. **`mage -h <alias>` does not resolve aliases.** Empirically observed: `mage -h ci-ui` returns exit 2 "Unknown target" even though `mage ci-ui` execution would succeed. This is mage's documented behavior — `-h` takes canonical target names. The alias is properly registered (E5 shows `Aliases: ci-ui` line under `mage -h ciUI`) and would dispatch correctly at execution time. Not a D1.2 defect; surfaced for QA-falsification awareness in case the sibling attempts to use `mage -h <alias>` as a falsification probe.

### Hylla Feedback

N/A — FE droplet, Hylla is Go-only per `feedback_hylla_go_only_today.md` rule (and Hylla is OFF entirely per `feedback_hylla_disabled_for_now.md` 2026-05-18). `Read`, `Bash` (mage targets + git status), and full-file source inspection were the right primitives. The literal-substring check could not use `grep` directly (the rtk-proxied `grep -n` Bash form was sandboxed-denied this round), but full-file `Read` of all 707 lines provided equivalent ground truth.

## Droplet 1.3 — Round 1

- **Reviewer:** `fe-qa-proof-agent`
- **Verdict:** **PASS**
- **Date:** 2026-05-18
- **Scope:** D1.3 acceptance bullets — `NewApp(nil)` removal + real-service wiring + embed-directive preservation + path-resolution mirror of `cmd/till/main.go` (PLAN.md D1.3 rows 84-90). `wails build` exit-0 acceptance gate **routed to Phase 6** (sandbox-denied for both builder + QA — see Unknowns). `mage ci` **deferred** per parallel-builder spawn-prompt rule (sibling builders dirty in `internal/adapters/mcp_rpc/`, `internal/app/dispatcher/`, `magefile.go`).
- **Files reviewed:** `ui/main.go` (full read, 98 lines), `git diff ui/main.go`, `git show HEAD:ui/main.go` (baseline embed-directive comparison), `internal/platform/paths.go`, `internal/config/config.go` (signature spans), `internal/adapters/storage/sqlite/repo.go` (`Open` + `Close` spans), `internal/app/service.go` (`NewService` + `IDGenerator` + `Clock` + `DeleteMode` definitions), `cmd/till/main.go:2415` + `:3500` (cross-reference call sites), `BUILDER_WORKLOG.md` § "Droplet 1.3 — Round 1", `git status --porcelain`.

### Premises

P1. `ui/main.go` contains zero occurrences of `NewApp(nil)`.
P2. `ui/main.go` contains exactly one `NewApp(svc)` call site at the wails-startup junction.
P3. The `//go:embed all:frontend/dist` directive is byte-identical to its pre-D1.3 baseline content (line position may shift; content must not).
P4. `newServiceFromConfig() (*app.Service, func(), error)` exists in `ui/main.go` and orchestrates the documented chain: `platform.DefaultPaths` → `config.Default(paths.DBPath)` → `config.Load(paths.ConfigPath, defaultCfg)` → `sqlite.Open(cfg.Database.Path)` → `app.NewService(repo, uuid.NewString, nil, app.ServiceConfig{...})`.
P5. Each call-site signature in P4 matches the production signature in `internal/{platform,config,adapters/storage/sqlite,app}/`.
P6. `main()` defers the cleanup callback BEFORE invoking `wails.Run` (so the SQLite handle closes on normal exit).
P7. The single `app.ServiceConfig` field set (`DefaultDeleteMode`) is the minimal viable choice for an FE bootstrap that does not exercise auth, embeddings, or live-wait — and `app.NewService` defaults every unset field per `service.go:163-211`.
P8. `app.DeleteMode(cfg.Delete.DefaultMode)` is a valid Go cross-package named-string conversion (both types share underlying `string`) and mirrors `cmd/till/main.go:2415` exactly.
P9. `git status --porcelain` for D1.3 scope is `ui/main.go` + `workflow/drop_fe_1_bootstrap/{BUILDER_WORKLOG.md,PLAN.md}` only.
P10. `wails build` is sandbox-denied for the QA agent (matching the builder's sandbox), so the build-gate acceptance bullet is route-to-Phase-6, not buildable-locally.

### Evidence

E1. `rg -c 'NewApp\(nil\)' ui/main.go` → `0`. → P1 verified.

E2. `rg -n "NewApp" ui/main.go` shows three lines: L32 doc comment, L33 function signature, L79 call site `application := NewApp(svc)`. The only invocation is at L79 with the real service. → P2 verified.

E3. Embed-directive baseline comparison via `git show HEAD:ui/main.go > /tmp/ui_main_baseline.go; rg -n "go:embed" /tmp/ui_main_baseline.go ui/main.go`:
```
/tmp/ui_main_baseline.go:16://go:embed all:frontend/dist
ui/main.go:21://go:embed all:frontend/dist
```
Line moved 16→21 because the import block grew (5 new imports: `fmt`, `sqlite`, `config`, `platform`, `uuid`). Directive content **byte-identical** to baseline. The §N10 trap variant 1 (a "helpful" rewrite to `all:ui/frontend/dist`) is dodged. → P3 verified.

E4. `Read ui/main.go` L47-70 (the helper):
```go
func newServiceFromConfig() (*app.Service, func(), error) {
    paths, err := platform.DefaultPaths()
    if err != nil { return nil, nil, fmt.Errorf("resolve runtime paths: %w", err) }
    defaultCfg := config.Default(paths.DBPath)
    cfg, err := config.Load(paths.ConfigPath, defaultCfg)
    if err != nil { return nil, nil, fmt.Errorf("load config %q: %w", paths.ConfigPath, err) }
    repo, err := sqlite.Open(cfg.Database.Path)
    if err != nil { return nil, nil, fmt.Errorf("open sqlite repository %q: %w", cfg.Database.Path, err) }
    svc := app.NewService(repo, uuid.NewString, nil, app.ServiceConfig{
        DefaultDeleteMode: app.DeleteMode(cfg.Delete.DefaultMode),
    })
    cleanup := func() {
        if closeErr := repo.Close(); closeErr != nil {
            log.Printf("warning: close sqlite repository: %v", closeErr)
        }
    }
    return svc, cleanup, nil
}
```
Helper returns `(*app.Service, func(), error)`; chain matches the claim verbatim. → P4 verified.

E5. Signature alignment (Read each production span):

- `internal/platform/paths.go:28` — `func DefaultPaths() (Paths, error)` with `Paths{ConfigPath, DataDir, DBPath, LogsDir}` defined at L11-17. Helper uses `paths.DBPath` (L52) + `paths.ConfigPath` (L53) — both valid fields. **Match.**
- `internal/config/config.go:191` — `func Default(dbPath string) Config`. Helper passes `paths.DBPath` (a `string` per `Paths.DBPath`). **Match.**
- `internal/config/config.go:295` — `func Load(path string, defaults Config) (Config, error)`. Helper passes `paths.ConfigPath` + `defaultCfg`. Missing-file is non-fatal per L302-306 (`os.ErrNotExist` → returns defaults cleanly), so first-run users with no `~/.tillsyn/config.toml` get the bootstrap path. **Match.**
- `internal/adapters/storage/sqlite/repo.go:75` — `func Open(path string) (*Repository, error)`. Helper passes `cfg.Database.Path` (a `string` per `DatabaseConfig.Path` at config.go:48-50). **Match.**
- `internal/app/service.go:163` — `func NewService(repo Repository, idGen IDGenerator, clock Clock, cfg ServiceConfig) *Service`. Helper passes 4 args: `repo` (a `*sqlite.Repository` which satisfies `app.Repository` — same interface satisfaction the CLI relies on at `cmd/till/main.go:2314`), `uuid.NewString` (matches `type IDGenerator func() string` at L123), `nil` clock (defaulted to `time.Now` at L167-169), and `app.ServiceConfig{DefaultDeleteMode: ...}`. **Match.**
- `internal/adapters/storage/sqlite/repo.go:121` — `func (r *Repository) Close() error`. Cleanup callback at L65 calls `repo.Close()`, logs the error. **Match.**

→ P5 verified.

E6. `Read ui/main.go` L72-93 (the `main` body):
```go
func main() {
    svc, cleanup, err := newServiceFromConfig()
    if err != nil { log.Fatal(err) }
    defer cleanup()

    application := NewApp(svc)

    err = wails.Run(&options.App{ ... })
    if err != nil { log.Fatal(err) }
}
```
`defer cleanup()` sits at L77, **before** `NewApp(svc)` at L79 and **before** `wails.Run` at L81. On normal `wails.Run` return, the deferred cleanup closes the SQLite handle. → P6 verified.

E7. `Read internal/app/service.go:163-211` (NewService body):
- L167-169: `if clock == nil { clock = time.Now }` → `nil` clock is safe.
- L170-172: `if cfg.DefaultDeleteMode == "" { cfg.DefaultDeleteMode = DeleteModeArchive }` → empty default-delete-mode is safe.
- L173-175: `CapabilityLeaseTTL` defaults if `<=0` → safe with zero-value.
- L176-179: `RequireAgentLease` defaults to `true` when nil.
- L180-183: `StateTemplates` defaults to `defaultStateTemplates()` if empty.
- L184-189: `SearchIndex` defaults from repo's `EmbeddingSearchIndex` interface satisfaction (sqlite.Repository may or may not satisfy; either way, no crash).
- L196: `HandoffRepository` defaults from repo's interface satisfaction (type-assert with `_, ok :=` — safe even if not satisfied).
- L202-204: `LiveWaitBroker` defaults to `NewInProcessLiveWaitBroker()`.
- L206-208: `GitStatusChecker` defaults to `defaultGitStatusChecker`.

All unset ServiceConfig fields are defaulted to non-nil sentinels by the constructor. The FE bootstrap's `ListProjects` call path will not nil-deref. → P7 verified.

E8. Cross-package `DeleteMode` conversion:
- `internal/config/config.go:17` — `type DeleteMode string`.
- `internal/app/service.go:22` — `type DeleteMode string`.
- Both share underlying `string`. Go permits the cross-package named-string conversion `app.DeleteMode(cfg.Delete.DefaultMode)` (cfg.Delete.DefaultMode is `config.DeleteMode`; both fundamentally `string`).
- `rg -n "DeleteMode\(cfg\.Delete\.DefaultMode\)" cmd/ internal/ ui/` → 3 matches: `cmd/till/main.go:2415`, `cmd/till/main.go:3500`, `ui/main.go:62`. Builder's call site is **structurally identical** to the two pre-existing production sites.

→ P8 verified.

E9. `git status --porcelain` (full repo) shows:
```
 M magefile.go                                                   ← D1.2 (sibling, complete)
 M ui/main.go                                                    ← D1.3 (this droplet)
 M workflow/drop_fe_1_bootstrap/BUILDER_WORKLOG.md               ← D1.3 (this droplet)
 M workflow/drop_fe_1_bootstrap/PLAN.md                          ← D1.3 (this droplet)
```
D1.3's code-side scope is exactly `ui/main.go`. Sibling Go-side builders' working-tree state lives in their own dispatched contexts (not visible in this filter); the spawn-prompt warned of `internal/adapters/mcp_rpc/`, `internal/app/dispatcher/` activity outside D1.3's surface. **No cross-contamination from D1.3.** → P9 verified.

E10. `wails build` attempt: invoking `cd ui && wails build` was sandbox-denied for the QA agent ("Permission to use Bash has been denied"). Same denial pattern as builder's attempt. `wails --version` likewise denied. Build-gate is not locally executable from this QA agent; routed to Phase 6 dev-launch. → P10 verified (sandbox denial is honest, not invented).

### Trace or cases

T1. **`NewApp(nil)` removal.** `rg -c 'NewApp\(nil\)' ui/main.go` returns 0. `rg -n NewApp ui/main.go` shows L32 (doc comment), L33 (declaration), L79 (call site `NewApp(svc)`). Three references; one is the call site; the call site uses the real service. Acceptance bullet "no longer contains `NewApp(nil)`" → **MET**.

T2. **Embed-directive preservation (§N10 trap dodged).** Baseline `git show HEAD:ui/main.go` at L16 reads `//go:embed all:frontend/dist`. Post-D1.3 `ui/main.go` at L21 reads `//go:embed all:frontend/dist`. Byte-identical. The new helper sits at L42-70, BETWEEN the embed directive (L21) and `main()` (L72) — directly in the "edit blast radius" where a `Write`-tool full-file rewrite could have drifted the directive. Builder explicitly defended against this in BUILDER_WORKLOG.md ("Embed-trap §N10 awareness" note); the defense held.

T3. **Path-resolution mirror of `cmd/till/main.go`.** Helper chain (E4):
   - `platform.DefaultPaths()` — same call shape `cmd/till/main.go` uses for non-dev-mode invocations; `appName="tillsyn"` default; resolves to `~/.tillsyn/{config.toml,tillsyn.db,logs/}` on macOS/Linux.
   - `config.Default(paths.DBPath)` → seeds `Database.Path` with the platform-resolved DB location.
   - `config.Load(paths.ConfigPath, defaultCfg)` → reads user config TOML; missing-file is non-fatal (E5).
   - `sqlite.Open(cfg.Database.Path)` → opens the SAME DB file the CLI opens (no hardcoded path; goes through `cfg.Database.Path` which is the platform-resolved value unless the user overrode it in `config.toml`).
   - `app.NewService(repo, uuid.NewString, nil, ServiceConfig{DefaultDeleteMode: ...})` → builds the service against that DB. Minimal `ServiceConfig` is appropriate because FE bootstrap only exercises `ListProjects`-like read paths (no auth, no embeddings, no live-wait).
   Acceptance bullet "DB path resolution mirrors `cmd/till/main.go` (`config.Load` → `cfg.Database.Path`); no hardcoded path" → **MET**.

T4. **Cleanup discipline + error path.** `main` defers `cleanup()` immediately after the helper returns success (L77). Normal `wails.Run` return → defer fires → `repo.Close()` runs. `wails.Run` error → `log.Fatal(err)` calls `os.Exit(1)` which skips deferred funcs — but the SQLite WAL pragma (`PRAGMA journal_mode = WAL` at `repo.go:132`) means the DB is durable across process-killed exits; no data loss. This mirrors `cmd/till/main.go:2319-2323`'s identical pattern. The `log.Fatal`-skips-defer behavior is well-documented Go semantics + the cost is bounded (one open `*sql.DB` handle, no in-flight transactions on the FE startup path). **Acceptable.**

T5. **Minimal `ServiceConfig` field set.** Only `DefaultDeleteMode` is set. All other fields (`AuthRequests`, `EmbeddingGenerator`, `LiveWaitBroker`, `CapabilityLeaseTTL`, `RequireAgentLease`, `StateTemplates`, search weights, `BootstrapProjectHooks`, etc.) are defaulted by `NewService`'s constructor body (E7). The bootstrap droplet's IPC surface — limited to `ListProjects` per D1.4 — does not touch any of these subsystems. The minimal set is appropriate; future FE drops (D2.x onwards) will populate matching fields when they wire auth/embeddings/IPC-mediated waits.

T6. **Build-tag fence isolates `ui/main.go` from `mage ci`.** The file's first line is `//go:build wails`. Without `-tags wails`, the Go toolchain excludes `ui/main.go` from compilation. `mage ci` does not pass `-tags wails`, so it compiles the non-wails view of the project — which D1.3 does not touch. This structural property means D1.3's code-side diff cannot break `mage ci` regardless of correctness inside the wails-tagged file. The drop-end `mage ci` gate (after sibling commits land) is preserved as a safety net but is not the load-bearing acceptance for D1.3 specifically.

T7. **Sandbox-denied build gate routed honestly.** `wails build` denial recorded clearly (E10); no fabricated PASS, no glossed-over "deferred" without explanation. Phase 6 dev-launch will execute the build + open-window verification on the dev's local machine where `wails` is installed.

### Conclusion

**PASS.** All proof premises verified by direct evidence. The builder's claims in `BUILDER_WORKLOG.md` § "Droplet 1.3 — Round 1" hold:

- `NewApp(nil)` is absent; `NewApp(svc)` appears once at the canonical wails-startup junction (T1, E2).
- The `newServiceFromConfig` helper exists, mirrors `cmd/till/main.go:2244-2314 + :2414`, and every call site matches the corresponding production signature (E4, E5, T3).
- The `//go:embed all:frontend/dist` directive is byte-identical to baseline despite the surrounding edit (E3, T2). §N10 variant-1 trap correctly dodged.
- Cleanup callback closes the SQLite handle on normal exit (E6, T4); error path is acceptable per CLI parity.
- `app.ServiceConfig` minimal-field choice is correct given the bootstrap's IPC scope (T5, E7).
- D1.3's diff is structurally compatible with `mage ci` via the `//go:build wails` fence (T6); drop-end CI provides the deterministic safety net.

The single unverified acceptance bullet (`cd ui && wails build` exits 0 + Mach-O binary at `ui/build/bin/Tillsyn.app/Contents/MacOS/Tillsyn`) is **route-to-Phase-6**, not a defect: the QA agent's sandbox denied the same invocation it denied for the builder. This is a known cascade-shape gap that the drop's Phase 6 dev-launch verification is designed to close.

**Findings count: 0 PASS-blocking.** 1 routed Unknown (sandbox-denied build gate).

### Unknowns

U1. **`wails build` gate sandbox-denied — routed to Phase 6.** Both the builder agent and this QA agent received "Permission to use Bash has been denied" on every attempted invocation of `wails build`, `wails --version`, and `go build -tags wails ./ui/...`. The acceptance bullet "`cd ui && wails build` exits 0 and produces `ui/build/bin/Tillsyn.app/Contents/MacOS/Tillsyn` as a Mach-O binary" cannot be checked from inside the cascade today. Mitigations layered on top of the denial:
   1. **Static code review** confirms the file compiles in principle — every call site matches a real signature (E5), no unreachable types, no undeclared identifiers (verified via Read of all production spans).
   2. **Symbol-level cross-reference** confirms the call pattern is identical to `cmd/till/main.go:2415` (E8) which compiles in `mage ci` today.
   3. **Build-tag fence** keeps `ui/main.go` out of `mage ci`'s default compile (T6), so the structural integrity of the Go module is preserved even if D1.3 had a wails-only compile defect.
   The actual exit-0 + Mach-O production check is **Phase 6 dev-launch responsibility**. Route: orchestrator surfaces this in CLOSEOUT.md as "FE drop's wails-build acceptance requires dev-machine verification because the agent sandbox denies the `wails` toolchain."

U2. **`mage ci` deferred to drop-end per spawn-prompt rule.** Sibling parallel builders are dirty in `magefile.go` (D1.2 done but uncommitted), `internal/adapters/mcp_rpc/`, `internal/app/dispatcher/`. Running `mage ci` from inside this QA agent would either (a) include sibling WIP and produce a false-positive fail, or (b) silently see sibling WIP via shared filesystem state (per `feedback_parallel_builders_share_worktree.md`) and produce a non-deterministic result. The build-tag fence (T6) is the structural reason D1.3's correctness is independent of the drop-end `mage ci` verdict. Route: drop-end CI gate, post-sibling-commit.

U3. **`log.Fatal` skips deferred cleanup.** If `wails.Run` returns an error, `log.Fatal` calls `os.Exit(1)` which bypasses `defer cleanup()`. The SQLite handle is not closed cleanly in that path. Mitigated by WAL mode (`repo.go:132`'s `PRAGMA journal_mode = WAL`) — the DB stays durable. The CLI has the identical property at `cmd/till/main.go:2319-2323`, so this is project-policy precedent, not a D1.3-introduced regression. **Accepted.** Not a refinement candidate — matches established pattern.

U4. **`cfg.Delete.DefaultMode` may be empty on first run.** If `config.Load` finds no `~/.tillsyn/config.toml` and `config.Default("")` does not populate `Delete.DefaultMode`, the conversion `app.DeleteMode(cfg.Delete.DefaultMode)` yields an empty string. `app.NewService` then defaults it to `DeleteModeArchive` at `service.go:170-172`. **Self-healing.** No counterexample.

### Hylla Feedback

N/A — FE droplet, Hylla is OFF entirely per `feedback_hylla_disabled_for_now.md` (2026-05-18). Used `Read`, `rg` (rtk-proxied — exact `grep` invocations were sandbox-denied), `git diff`, `git show HEAD:`, and direct production-file inspection. The narrow signature-lookup pattern was well-served by `Read` + `rg`. `LSP` not consulted because the symbols in `internal/{platform,config,adapters/storage/sqlite,app}/` are clearly named and `Read`+`rg` resolution was sufficient.

## Droplet 1.4 — Round 1

- **Reviewer:** `fe-qa-proof-agent`
- **Verdict:** **PASS** (with one Unknown routed to dev: `go test -tags wails ./ui/...` execution)
- **Date:** 2026-05-18
- **Scope:** D1.4 acceptance bullets — new `App.ListProjects()` IPC method + `ProjectDTO` in a separate `ui/types.go` + `ui/app_test.go` smoke test; embed-directive preservation; DTO not leaked into `ui/main.go`. `go test -tags wails ./ui/...` execution **routed to dev** (sandbox-denied for builder and QA, matching D1.3 pattern). `mage ci` **deferred** per parallel-builder spawn-prompt rule.
- **Files reviewed:** `ui/main.go` (full read, 116 lines), `ui/types.go` (full read, 18 lines, NEW), `ui/app_test.go` (full read, 113 lines, NEW), `git diff HEAD -- ui/main.go`, `git status --porcelain ui/ workflow/drop_fe_1_bootstrap/`, `internal/adapters/storage/sqlite/repo.go` (`OpenInMemory` signature), `internal/app/service.go` (`ListProjects`, `CreateProject`, `NewService` signatures), `BUILDER_WORKLOG.md` § "Droplet 1.4 — Round 1".

### Premises

P1. `ui/main.go` contains `func (a *App) ListProjects() ([]ProjectDTO, error)` between `(*App).startup` and `newServiceFromConfig`. Body calls `a.svc.ListProjects(a.ctx, false)` and maps each `domain.Project` to `ProjectDTO{ID: p.ID, Name: p.Name}` via a pre-allocated slice.
P2. The `//go:embed all:frontend/dist` directive at `ui/main.go:21` is byte-identical to its pre-D1.4 state; the only diff entries are the new `ListProjects` method body.
P3. `ui/types.go` exists as a NEW file with `//go:build wails` on line 1, `package main` on line 3, and `type ProjectDTO struct { ID string; Name string }` as its only declaration. No `json:` tags (per PLAN.md §N2 design call).
P4. `ui/app_test.go` exists as a NEW file with `//go:build wails` on line 1, `package main` on line 3, and contains `TestApp_ListProjects_ReturnsDTOForExistingProject` plus a test-local `itoa` helper.
P5. The test constructs the service via `sqlite.OpenInMemory()` (NOT raw `sqlite.Open(":memory:")`), seeds via `svc.CreateProject(ctx, ...)`, asserts (a) `err == nil`, (b) `len(result) >= 1`, (c) every DTO has non-empty `ID` + `Name`, (d) the seeded `(ID, Name)` appears in the result set.
P6. `type ProjectDTO struct` appears **zero** times in `ui/main.go` and **exactly once** in `ui/types.go` (no inline DTO leak).
P7. Call-site signature `a.svc.ListProjects(a.ctx, false)` matches `internal/app/service.go:2252` (`func (s *Service) ListProjects(ctx context.Context, includeArchived bool) ([]domain.Project, error)`).
P8. `sqlite.OpenInMemory()` exists in the production source tree at `internal/adapters/storage/sqlite/repo.go:101`.
P9. `git status --porcelain ui/ workflow/drop_fe_1_bootstrap/` shows exactly the D1.4 scope: `ui/main.go` modified, `ui/types.go` + `ui/app_test.go` new, `BUILDER_WORKLOG.md` + `PLAN.md` modified. No stray files.
P10. `go test -tags wails ./ui/...` is sandbox-denied for the QA agent (matching the builder's denial); the execution gate is route-to-dev, not buildable from the cascade.

### Evidence

E1. `Read ui/main.go` L42-58 (the new method):
```go
// ListProjects is the Wails IPC method exposed to the frontend as
// window.go.main.App.ListProjects(). Returns every non-archived project on
// the underlying SQLite store projected into the JS-friendly ProjectDTO
// shape. Read-only — never mutates the store. Errors from the service layer
// surface verbatim (Wails serializes (T, error) returns as a JS promise that
// rejects on non-nil error).
func (a *App) ListProjects() ([]ProjectDTO, error) {
    projects, err := a.svc.ListProjects(a.ctx, false)
    if err != nil {
        return nil, err
    }
    dtos := make([]ProjectDTO, 0, len(projects))
    for _, p := range projects {
        dtos = append(dtos, ProjectDTO{ID: p.ID, Name: p.Name})
    }
    return dtos, nil
}
```
Method body matches the claim verbatim: `a.svc.ListProjects(a.ctx, false)` call, pre-allocated `make([]ProjectDTO, 0, len(projects))`, field-for-field projection. → P1 verified.

E2. `git diff HEAD -- ui/main.go` shows **only** an 18-line insertion at L42-58 (the new method + surrounding doc comment). The diff hunk header is `@@ -39,6 +39,24 @@` — the change sits between L39 (existing `(*App).startup` close) and the pre-existing `newServiceFromConfig`. The `//go:embed all:frontend/dist` directive at L21 is **outside** the diff hunk → byte-identical to pre-D1.4 state. → P2 verified. §N10 variant-1 trap correctly dodged (matching D1.3's same dodge).

E3. `Read ui/types.go` (full file, 18 lines):
- L1: `//go:build wails`
- L2: blank
- L3: `package main`
- L4: blank
- L5-13: doc comment explaining Wails wire-format defaults and pointing at the future D1.5 `wails.d.ts` ambient declaration.
- L14-17: `type ProjectDTO struct { ID   string; Name string }` (literal field layout: capitalized `ID` and `Name`, bare `string` types, no `json:` tags).
- L18: file ends after struct close brace.
Only declaration in the file is `ProjectDTO`. → P3 verified. PLAN.md §N2 "no `json:` tags" design call honored.

E4. `Read ui/app_test.go` (full file, 113 lines):
- L1: `//go:build wails`
- L3: `package main`
- L5-13: imports (`context`, `strings`, `testing`, `time`, `internal/adapters/storage/sqlite`, `internal/app`).
- L26: `func TestApp_ListProjects_ReturnsDTOForExistingProject(t *testing.T) {`
- L27-33: `repo, err := sqlite.OpenInMemory()` + `t.Cleanup(func() { _ = repo.Close() })`.
- L38-47: deterministic counter-based `idGen` + monotonic `clk`.
- L48: `svc := app.NewService(repo, idGen, clk, app.ServiceConfig{})`.
- L52-59: `svc.CreateProject(ctx, "Tillsyn FE Smoke", "in-memory seed for App.ListProjects bridge test")` + non-empty `seeded.ID` assertion.
- L65-66: `application := NewApp(svc); application.startup(ctx)` — sets `a.ctx` before the IPC call.
- L68-71: `result, err := application.ListProjects()` + `err == nil` assertion (assert (a)).
- L72-74: `if len(result) < 1` → fatal (assert (b)).
- L77-84: per-DTO loop asserting non-empty `ID` + `Name` (assert (c)).
- L87-97: linear scan for seeded `(ID, Name)` pair (assert (d)).
- L100-113: test-local `itoa` helper (avoids `strconv` import for ID padding).
→ P4, P5 verified. Assertions (a)-(d) all present and match PLAN.md acceptance bullets.

E5. DTO leak check via `Read` of both files:
- `ui/main.go` (full read) contains **zero** lines matching `type ProjectDTO struct`. The string `ProjectDTO` appears only as a return type at L48 (`func (a *App) ListProjects() ([]ProjectDTO, error)`), a slice element type at L53 (`make([]ProjectDTO, 0, len(projects))`), and a struct literal at L55 (`ProjectDTO{ID: p.ID, Name: p.Name}`).
- `ui/types.go` contains **exactly one** line matching `type ProjectDTO struct` at L14.
→ P6 verified. (Raw `grep` Bash invocation was sandboxed-denied for this round, but full-file Read of both files provides equivalent ground truth.)

E6. `rg -n 'func.*Service.*ListProjects' internal/app/service.go` → `service.go:2252: func (s *Service) ListProjects(ctx context.Context, includeArchived bool) ([]domain.Project, error)`. Signature matches the call `a.svc.ListProjects(a.ctx, false)` precisely (`a.ctx` is `context.Context`, `false` is `includeArchived`). → P7 verified.

E7. `rg -n 'func OpenInMemory' internal/adapters/storage/sqlite/` → `repo.go:101: func OpenInMemory() (*Repository, error)`. The test's `repo, err := sqlite.OpenInMemory()` call site matches. → P8 verified.

E8. `git status --porcelain ui/ workflow/drop_fe_1_bootstrap/`:
```
 M ui/main.go
 M workflow/drop_fe_1_bootstrap/BUILDER_WORKLOG.md
 M workflow/drop_fe_1_bootstrap/PLAN.md
?? ui/app_test.go
?? ui/types.go
```
Exactly D1.4's declared scope: one modified Go source (`ui/main.go`), two new Go sources (`ui/types.go`, `ui/app_test.go`), and the two workflow MDs. No stray files. Sibling Go-D1.4 / Go-D1.5 working-tree state in `internal/` is invisible to this filter — properly out-of-scope per parallel-builders spawn-prompt rule. → P9 verified.

E9. `go test -tags wails ./ui/...` invocation attempted: `Permission to use Bash with command go test ... has been denied`. Same denial pattern as builder's attempt (BUILDER_WORKLOG.md L123 records this verbatim). → P10 verified (denial is honest, not invented).

### Trace or cases

T1. **`App.ListProjects` IPC contract.** The method has zero parameters and returns `([]ProjectDTO, error)` — the exact shape Wails serializes as `Promise<ProjectDTO[]>` on the JS side. `a.ctx` is read from the App struct (set by `startup` at L37-40 of `ui/main.go`); JS callers cannot pass a Go `context.Context`, so reading from `a.ctx` is the only correct pattern. Body delegates to `a.svc.ListProjects(a.ctx, false)` — `false` means "exclude archived" per the production signature, which is the correct default for the FE bootstrap's "show me real projects" semantics.

T2. **Slice pre-allocation.** `make([]ProjectDTO, 0, len(projects))` — capacity sized to source length; `append` cannot trigger reslice during the loop. For an empty source slice (`projects == nil` or `len == 0`), the result is `[]ProjectDTO{}` (zero-length but non-nil). Wails serializes both `[]` and `null` cleanly for JS consumption, but a deterministic empty-slice return matches the test's `len(result) >= 1` assertion semantics when ≥1 project is seeded.

T3. **Embed-directive preservation.** Same proof shape as D1.3 T2: `git diff HEAD -- ui/main.go` shows the diff hunk is bounded to L42-58 (the new method); the embed directive at L21 is outside the hunk and therefore byte-identical to baseline. The builder's edit tool used a small-context match around `(*App).startup`; the directive sits five lines above, in untouched territory.

T4. **`ProjectDTO` placement and content.**
- File: `ui/types.go` (NEW, dedicated DTO file per the §N2 round-2 F4-fals discussion in BUILDER_WORKLOG.md L121).
- Build tag: `//go:build wails` → DTO is excluded from non-wails builds (same fence as the rest of `ui/`).
- Package: `package main` → shares the same compile unit as `ui/main.go`, so `ProjectDTO` is reachable without import.
- Fields: `ID string` and `Name string` only. No `json:` tags (PLAN.md §N2 design call: Wails uses Go field names as-is for its IPC bindings, so JSON-tag round-trip semantics don't apply; the JS side gets `{ID, Name}` literally).
- Doc comment explicitly notes the JS-side capitalization and points at the D1.5 TypeScript ambient declaration.

T5. **Test setup pattern matches established repo conventions.** The counter-based `idGen` + monotonic `clk` shape exactly mirrors `cmd/till/action_item_cli_test.go:194-204` and `internal/app/attention_capture_test.go:18-28` per BUILDER_WORKLOG.md L116. The `app.NewService(repo, idGen, clk, app.ServiceConfig{})` constructor call uses the zero-value `ServiceConfig`, which `NewService` defaults out per D1.3's E7 (auth, embeddings, live-wait, etc. all defaulted to safe sentinels). This is sound because `ListProjects` and `CreateProject` do not exercise auth, embeddings, or live-wait paths.

T6. **`sqlite.OpenInMemory()` vs raw `sqlite.Open(":memory:")` — round-2 F2-fals resolution.** The test uses `sqlite.OpenInMemory()` (production helper at `repo.go:101`), NOT raw `sqlite.Open(":memory:")`. Per PLAN.md round-2 F2-fals (referenced in BUILDER_WORKLOG.md L25), `OpenInMemory()` is the canonical multi-connection-safe DSN `"file::memory:?cache=shared"` factory — future-proof if a refactor ever lifts the `MaxOpenConns(1)` cap. The test's `t.Cleanup(func() { _ = repo.Close() })` properly disposes the handle.

T7. **Assertion coverage matches PLAN.md acceptance bullets.** Per PLAN.md row 109 (referenced in BUILDER_WORKLOG.md L120), the four assertions are:
- (a) `err == nil` after `application.ListProjects()` → L68-71.
- (b) `len(result) >= 1` → L72-74.
- (c) every DTO has non-empty `ID` + `Name` → L77-84.
- (d) seeded `(ID, Name)` appears in the result set → L87-97.
All four assertions present, each with a distinct fatal/error trigger.

T8. **`itoa` helper isolation.** The test-local `itoa` (L100-113) is package-private + build-tag-gated (`//go:build wails`). It does not collide with `strconv.Itoa` because it's unexported and lives in the same package as the test that uses it. Non-wails builds (i.e., everything outside `ui/`) never compile this helper; production code (the CLI binary) does not include `ui/app_test.go` at all because Go's test toolchain excludes `_test.go` files from production builds. No production-binary contamination.

T9. **DTO non-leak into `ui/main.go`.** The acceptance bullet "`grep 'type ProjectDTO struct' ui/main.go` returns nothing" is verified via E5: full Read of `ui/main.go` (116 lines) shows zero matches for `type ProjectDTO struct`. The struct is referenced (return type at L48, slice element at L53, literal at L55) but not **declared**. The matching positive check (`grep 'type ProjectDTO struct' ui/types.go` returns one match) is also verified via E5: L14 is the sole declaration.

T10. **Scope-clean signal.** `git status --porcelain ui/ workflow/drop_fe_1_bootstrap/` (E8) returns exactly the five expected entries. Sibling-builder WIP outside `ui/` is properly invisible to this filter. Builder-claimed scope and on-disk reality agree.

T11. **Build-tag fence still isolates `ui/` from `mage ci`.** Same structural argument as D1.3 T6: `//go:build wails` on all three files (`ui/main.go`, `ui/types.go`, `ui/app_test.go`) keeps them out of the default Go compile that `mage ci` triggers. D1.4's diff cannot break `mage ci` for the non-wails build path. The drop-end CI gate remains the safety net for any hypothetical interaction with sibling Go work.

T12. **Sandbox-denied test execution routed honestly.** `go test -tags wails ./ui/...` denial recorded clearly (E9). No fabricated PASS, no glossed-over deferral. Builder did the same; QA mirrors the denial; dev runs the test locally.

### Conclusion

**PASS.** All proof premises verified by direct evidence. The builder's claims in `BUILDER_WORKLOG.md` § "Droplet 1.4 — Round 1" hold:

- `App.ListProjects()` exists with the claimed signature (`() ([]ProjectDTO, error)`) and body (delegates to `a.svc.ListProjects(a.ctx, false)`, maps domain projects to DTOs via a pre-allocated slice) — E1, T1, T2.
- `//go:embed all:frontend/dist` at `ui/main.go:21` is byte-identical to pre-D1.4 baseline — E2, T3. §N10 variant-1 trap dodged.
- `ui/types.go` is a clean NEW file: `//go:build wails`, `package main`, single `type ProjectDTO struct { ID string; Name string }` declaration, no `json:` tags — E3, T4.
- `ui/app_test.go` is a clean NEW file: `//go:build wails`, `package main`, `TestApp_ListProjects_ReturnsDTOForExistingProject` with all four required assertions, uses `sqlite.OpenInMemory()` (NOT raw `sqlite.Open(":memory:")`), seeds via `svc.CreateProject`, calls `application.startup(ctx)` before the IPC call — E4, T5, T6, T7.
- `type ProjectDTO struct` appears zero times in `ui/main.go` and exactly once in `ui/types.go` — E5, T9.
- Call-site signatures align with production (`ListProjects`, `OpenInMemory`, `CreateProject`, `NewService` all match) — E6, E7.
- Git scope is clean — E8, T10.
- Build-tag fence preserves `mage ci` insulation — T11.

The single unverified acceptance bullet (`go test -tags wails ./ui/...` exits 0) is **route-to-dev**, not a defect — the QA agent's sandbox denies the same invocation the builder's sandbox denied. Static cross-references prove every signature compiles in principle; the dynamic green-bar verification belongs to dev or to a future cascade-shape upgrade (rooting agents in a sandbox that permits `go test`).

**Findings count: 0 PASS-blocking.** 1 routed Unknown (sandbox-denied test execution).

### Unknowns

U1. **`go test -tags wails ./ui/...` sandbox-denied — routed to dev.** Both the D1.4 builder and this QA agent received "Permission to use Bash has been denied" on `go test -tags wails ./ui/...`. The acceptance bullet "wails-tagged tests exit 0 with `TestApp_ListProjects_ReturnsDTOForExistingProject` PASS" cannot be checked from inside the cascade today. Mitigations layered on top of the denial:
   1. **Static signature cross-references** confirm every external symbol the test invokes is real: `sqlite.OpenInMemory()` at `repo.go:101`, `app.NewService(repo, idGen, clk, ServiceConfig{})` at `service.go:163`, `svc.CreateProject(ctx, name, desc)` at `service.go:313`, `application.ListProjects()` per E1.
   2. **Type-system consistency** confirmed via Read: `ProjectDTO.ID` is `string`, `domain.Project.ID` is `string` (per D1.3 E5), `ProjectDTO.Name` is `string`, `domain.Project.Name` is `string` — no type mismatch in the projection.
   3. **Test logic flow** is internally consistent: seed succeeds → result must contain seed → linear-scan loop finds match → assertion (d) passes.
   The actual exit-0 + PASS check is **dev-machine responsibility**. Recommended command: `cd /Users/evanschultz/Documents/Code/hylla/tillsyn/main && go test -tags wails ./ui/...`. Route: orchestrator surfaces this in CLOSEOUT.md alongside D1.3's `wails build` denial as a class of "wails-tagged gates require dev-machine verification."

U2. **`mage ci` deferred per spawn-prompt rule.** Identical structural argument to D1.3 U2: sibling parallel builders are dirty in `internal/`; running `mage ci` here would either include sibling WIP (false-positive fail) or produce non-deterministic results. Build-tag fence (T11) preserves D1.4's structural independence from `mage ci`'s non-wails view. Route: drop-end CI gate.

U3. **`a.ctx == nil` when called outside the Wails runtime lifecycle.** If a future test or caller invokes `application.ListProjects()` without first calling `application.startup(ctx)`, `a.ctx` is the zero value (`nil`). `a.svc.ListProjects(nil, false)` would propagate the nil context to the repository layer; SQLite's driver typically tolerates nil context (treats as `context.Background()`) but the production contract is "non-nil context required." The D1.4 test correctly calls `application.startup(ctx)` first (E4 L66), and the Wails runtime always calls `startup(ctx)` at window-open. **Accepted — not a D1.4 defect, just a property of the IPC contract.** Future test rounds that add more `App.*` methods should mirror the `startup(ctx)` precondition.

U4. **No archived-project test case.** The test seeds one non-archived project and asserts it appears. There is no negative-path coverage proving the `false` argument actually filters archived projects (would require seeding an archived project and asserting it's absent). Acceptable for D1.4 because the filter semantics are tested at the service layer in `internal/app/*_test.go` (BUILDER_WORKLOG.md L138 — "service-layer filter semantics are tested in `internal/app/*_test.go`"); D1.4's bridge test exercises the read-bridge plumbing, not the filter. Future drops that expose `ListProjects(includeArchived=true)` would add the matching test. **Accepted — out-of-scope per PLAN.md.**

### Hylla Feedback

N/A — FE droplet, Hylla is OFF entirely per `feedback_hylla_disabled_for_now.md` (2026-05-18). Used `Read`, `Bash` (rtk-proxied `rg` — raw `grep` was sandbox-denied this round), `git diff`, and `git status`. Narrow signature lookups (`OpenInMemory`, `ListProjects`, `CreateProject`) were handled cleanly by `rg` against `internal/`. `LSP` not consulted because all referenced symbols are clearly named and `Read`+`rg` resolution was sufficient.

## Droplet 1.5 — Round 1

- **Reviewer:** `fe-qa-proof-agent`
- **Verdict:** **PASS**
- **Date:** 2026-05-18
- **Scope:** D1.5 acceptance bullets per spawn-prompt checks 1-6 (file correctness, build exit-0 + hydration marker, vitest non-vacuous, scope cleanliness). Dev-launched runtime checks (Wails window + CLI cross-check) are explicitly Phase 6 and out-of-scope.

### Evidence Bundle

E1. **`ui/frontend/src/components/ProjectList.tsx` (Read full file, 57 lines)** —
  - Line 1: `// MIGRATION TARGET: @hylla/stil-solid` (exact literal match — spawn-prompt check 1a).
  - Line 8: `import { createResource, For, Show } from 'solid-js';` — SolidJS primitives imported correctly.
  - Lines 12-23: `fetchProjects(): Promise<Project[]>` — SSR guard at lines 19-21: `if (typeof window === 'undefined') { return []; }` (exact match — spawn-prompt check 1b).
  - Line 22: `return window.go.main.App.ListProjects();` — exercises the Wails-injected bridge.
  - Line 26: `const [projects] = createResource<Project[]>(fetchProjects);` — SolidJS resource pattern (spawn-prompt check 1c).
  - Lines 31-54: Triple-nested `<Show>` — outer gates on `!projects.loading` with `fallback={<p>Loading…</p>}` (line 33); middle gates on `!projects.error` with `fallback={<p role="alert">Error: {String(projects.error)}</p>}` (line 37); inner gates on `(projects() ?? []).length > 0` with `fallback={<p>No projects yet</p>}` (line 41). Covers loading + error + empty + data states (spawn-prompt check 1d).
  - Line 41: `<p>No projects yet</p>` — exact empty-state literal (spawn-prompt check 1e).
  - Lines 43-51: `<ul><For each={projects()}>{(project) => (<li>{project.ID} — {project.Name}</li>)}</For></ul>` — `<ul><li>` structure displaying `ID + Name` per project (spawn-prompt check 1f).

E2. **`ui/frontend/src/pages/index.astro` (Read full file, 11 lines)** —
  - Line 2: `import MainLayout from '../layouts/MainLayout.astro';` — uses `MainLayout` (spawn-prompt check 2a).
  - Line 3: `import ProjectList from '../components/ProjectList';` — imports the new component correctly (spawn-prompt check 2c).
  - Line 6: `<MainLayout>` wrapper.
  - Line 9: `<ProjectList client:idle />` — `client:idle` directive, NOT `client:load` (spawn-prompt check 2b).

E3. **`ui/frontend/src/types/wails.d.ts` (Read full file, 18 lines)** —
  - Line 6: `export {};` — makes the file a module (required for `declare global` augmentation).
  - Lines 8-18: `declare global { interface Window { go: { main: { App: { ListProjects(): Promise<{ ID: string; Name: string }[]> } } } } }` — exact shape required by spawn-prompt check 3a.
  - DTO field names `ID` and `Name` capitalized at line 13, matching `ProjectDTO` in `ui/types.go` lines 14-17 (`type ProjectDTO struct { ID string; Name string }`). Cross-surface name parity confirmed (spawn-prompt check 3b).

E4. **Astro build via `pnpm --dir /Users/evanschultz/Documents/Code/hylla/tillsyn/main/ui/frontend run build` — exit 0.**
  - 6 modules transformed; emits `dist/_astro/ProjectList.BBbcfCQW.js` (1.09 kB / gzip 0.54 kB), `dist/_astro/client.CEmo_1HW.js` (6.11 kB / gzip 2.56 kB), `dist/_astro/web.Cx_12A-G.js` (13.86 kB / gzip 5.73 kB).
  - `▶ src/pages/index.astro` → `/index.html (+5ms)`; `1 page(s) built in 454ms`; `[build] Complete!`
  - **`dist/index.html` Read directly.** Contains `<astro-island uid="ZlVu73" data-solid-render-id="s0" component-url="/_astro/ProjectList.BBbcfCQW.js" component-export="default" renderer-url="/_astro/client.CEmo_1HW.js" props="{}" ssr client="idle" opts="…" await-children>` — exact `<astro-island` substring with `client="idle"` attribute (spawn-prompt check 4).
  - The SSR'd markup inside the island is `<section data-hk="s00001"><h2>Projects</h2><!--$--><p data-hk="s00002000">No projects yet</p><!--/--></section>` — the empty-state branch rendered during SSR (the SSR guard returned `[]` → `length > 0 === false` → `fallback={<p>No projects yet</p>}` taken). Confirms the guard works and the empty-state path is exercised before hydration.

E5. **Vitest via `pnpm --dir /Users/evanschultz/Documents/Code/hylla/tillsyn/main/ui/frontend run test:unit` — exit 0.**
  - Output: `✓ tests/migration-markers.test.ts (2 tests | 1 skipped) 1ms`; `Test Files 1 passed (1)`; `Tests 1 passed | 1 skipped (2)`.
  - **Non-vacuous verification.** Read `tests/migration-markers.test.ts` (68 lines). Logic at lines 34-49: `const files = collectFiles(componentsDir, ['.tsx', '.ts'])`. With `ProjectList.tsx` now present under `src/components/`, `files.length === 1`, so the `else` branch (lines 41-48) iterates calling `it(\`${path.relative(frontendDir, file)} contains migration marker\`, …)` — a real `it()` registration, NOT `it.skip()`. The vitest summary `1 passed | 1 skipped (2)` matches exactly: the passing test is the `ProjectList.tsx` marker assertion (lines 42-47: `expect(content).toContain(COMPONENT_MARKER)` where `COMPONENT_MARKER = '// MIGRATION TARGET: @hylla/stil-solid'`); the skipped test is the vim-engine branch at line 55 (`src/lib/vim/` is still empty, expected D9 territory). Spawn-prompt check 5 met — non-vacuous.

E6. **Scope via `git -C <worktree> status --porcelain ui/frontend/src/`.**
  - Output: `?? ui/frontend/src/components/` + `?? ui/frontend/src/pages/` + `?? ui/frontend/src/types/` — three NEW directories (each containing exactly one new file per E1/E2/E3 reads). No `M` rows for `MainLayout.astro`, `astro.config.mjs`, or `tsconfig.json`. Existing files untouched (spawn-prompt check 6).

### Spawn-Prompt Checks Cross-Reference

| Check | Requirement | Evidence | Status |
| --- | --- | --- | --- |
| 1a | Line 1 = `// MIGRATION TARGET: @hylla/stil-solid` | E1 line 1 | PASS |
| 1b | SSR guard `if (typeof window === 'undefined') return [];` | E1 lines 19-21 | PASS |
| 1c | `createResource` against `window.go.main.App.ListProjects` | E1 lines 22, 26 | PASS |
| 1d | `<Show>` covers loading + error + empty + data | E1 lines 31-54 (triple-nested) | PASS |
| 1e | Literal `No projects yet` | E1 line 41 | PASS |
| 1f | `<ul><li>` rendering `id + name` | E1 lines 43-51 | PASS |
| 2a | Uses `MainLayout` | E2 lines 2, 6 | PASS |
| 2b | Mounts `<ProjectList client:idle />` (NOT `client:load`) | E2 line 9 | PASS |
| 2c | ProjectList imported correctly | E2 line 3 | PASS |
| 3a | `declare global` with `Window.go.main.App.ListProjects(): Promise<{ ID: string; Name: string }[]>` | E3 lines 8-18 | PASS |
| 3b | Capitalized `ID`/`Name` matching `ProjectDTO` in `ui/types.go` | E3 line 13 + ui/types.go lines 14-17 | PASS |
| 4 | `pnpm run build` exit 0 + `dist/index.html` + `<astro-island … client="idle">` | E4 (build output + dist read) | PASS |
| 5 | Migration-markers test PASSING non-vacuously | E5 (vitest `1 passed | 1 skipped` + test-source review) | PASS |
| 6 | Only 3 new files; no edits to existing | E6 (`git status` output) | PASS |

### Falsification Attacks Considered (mitigated)

A1. **SSR guard might still leave a build-time error.** `pnpm run build` exit 0 with no warnings about the island; `dist/index.html` contains the SSR'd empty-state markup cleanly (E4). The guard returns `[]` synchronously, so `createResource`'s loader resolves immediately during the SSR pass; the triple `<Show>` then evaluates loading=false → error=undefined → `length > 0 === false` → empty-state fallback rendered. **Mitigated.**

A2. **Empty-state literal in source might differ from what SSR emits.** E1 line 41 source = `<p>No projects yet</p>`; E4 dist HTML = `<p data-hk="s00002000">No projects yet</p>` (text content identical, Astro adds the `data-hk` hydration key). **Mitigated.**

A3. **`client:idle` might be silently downgraded by Astro 5.** E4 dist HTML shows `client="idle"` attribute on the `<astro-island>` web component; the inline hydration runtime in `dist/index.html` uses `"requestIdleCallback"in window?window.requestIdleCallback(i,s):setTimeout(i,s.timeout||200)` — that's Astro 5's canonical `client:idle` hydration impl. **Mitigated.**

A4. **TypeScript ambient declaration might collide with existing `ui/frontend/src/env.d.ts`.** E3 uses `export {}` to make the file a module + `declare global` to augment the `Window` interface — standard Astro/TS pattern. `env.d.ts` was not touched (E6 scope check). Augmentations are additive, no collision detected. **Mitigated.**

A5. **DTO capitalization parity Go-side vs TS-side.** E3 line 13 declares `ID: string; Name: string`; cross-read of `ui/types.go` lines 14-17 (`type ProjectDTO struct { ID string; Name string }`, no `json:"…"` tags, so Wails serializes with default Go-export-case names `ID`/`Name`). Both surfaces agree. **Mitigated.**

A6. **vitest `1 passed | 1 skipped` might not actually be the `ProjectList.tsx` case.** Reading the test source (E5) — `collectFiles(componentsDir, ['.tsx', '.ts'])` returns exactly the new `ProjectList.tsx`; the `else` branch fires a real `it()`. The vim branch (`src/lib/vim/`) is still empty so it goes through `it.skip()`. The arithmetic matches: 1 passed (`ProjectList.tsx contains migration marker`) + 1 skipped (vim-engine vacuous) = 2 total. **Mitigated.**

A7. **`<Show>` might short-circuit incorrectly during SSR causing wrong markup.** SSR pass: `fetchProjects()` returns `[]` synchronously (guard). SolidJS `createResource` initializes with loader's synchronous return → `projects.loading === false`, `projects.error === undefined`, `projects()` returns `[]`. Outer `<Show when={!projects.loading}>` → true (enter); middle `<Show when={!projects.error}>` → true (enter); inner `<Show when={(projects() ?? []).length > 0}>` → false (take fallback `<p>No projects yet</p>`). Result matches E4 dist HTML exactly. **Mitigated.**

A8. **Scope might silently include edits to existing files.** E6 `git status --porcelain ui/frontend/src/` returns only three `??` rows, no `M` rows. **Mitigated.**

A9. **The `?? []` defensive default at line 40 might mask a runtime bug.** When `projects()` returns the resolved array, `?? []` short-circuits because the LHS is non-nullish. When the resource is in flight, the outer `<Show>` gate prevents the inner reads. The defensive default only fires in the synchronous initial-render edge where the SolidJS reactive system hasn't yet propagated, and `[]` is the correct semantic identity for "no data" — same path as the SSR guard. No observable runtime divergence. **Mitigated.**

### Unknowns

U1. **Dev-launched runtime cross-check (Wails window + CLI parity).** Spawn-prompt note: "The CLI cross-check (A3 — comparing rendered list to `till project list` output) is dev-launched at Phase 6." This QA verifies the static + SSR'd surface; the dynamic hydrated DOM with real `window.go.main.App.ListProjects()` data is dev-launched. **Routed — out-of-scope per spawn prompt.**

U2. **`mage ci` deferred per spawn-prompt rule.** Sibling Go D1.4 + D1.6 builders are concurrently dirty in `internal/`; running `mage ci` here would either include sibling WIP (false-positive fail) or produce non-deterministic results. `mage ciUI` (which the builder ran successfully per BUILDER_WORKLOG.md line 153) is the correctly-scoped FE-only gate. **Routed — drop-end CI gate.**

U3. **SolidJS reactive identity across resource refetches.** Builder noted `<For>` keying assumes "the same `{ID, Name}` shapes should produce identical references" across refetches. In practice, each `fetchProjects()` call produces a fresh array of fresh object literals, so `<For>` will see new reference identities and re-render rows. This is correct/expected SolidJS behavior — `<For>` keys on reference equality, but the diff is shallow on the array level: removing/adding rows is detected by identity comparison, and within-row text updates trigger `<li>` content re-renders. Not a defect; a non-issue for D1.5's empty-or-one-item rendering path. **Accepted — design judgment, not a D1.5 defect.**

### Hylla Feedback

N/A — FE droplet, Hylla is OFF entirely per `feedback_hylla_disabled_for_now.md` (2026-05-18). Used `Read` (4 file reads — `BUILDER_WORKLOG.md`, `ProjectList.tsx`, `index.astro`, `wails.d.ts`, plus `dist/index.html`, `migration-markers.test.ts`, `ui/types.go`), `Bash` (`pnpm --dir` build + test, `git status --porcelain`). No Hylla calls attempted (FE files aren't indexed today per `feedback_hylla_go_only_today.md`). No Context7 calls needed — SolidJS `createResource` + `<Show>` semantics and Astro `client:idle` hydration confirmed via reading the emitted dist HTML's inline hydration runtime, which encodes the exact `requestIdleCallback` strategy.
