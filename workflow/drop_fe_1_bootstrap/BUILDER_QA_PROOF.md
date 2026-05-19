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
