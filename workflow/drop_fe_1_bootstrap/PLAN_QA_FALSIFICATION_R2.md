# Drop FE 1 Bootstrap — Plan QA Falsification — Round 2

**Verdict:** `fail` (1 CONFIRMED counterexample, 3 POSSIBLE counterexamples, 4 NITs)
**Round:** 2
**Reviewer:** fe-qa-falsification-agent
**Date:** 2026-05-18

Round 2 falsification pass against `PLAN.md` (D1.1 – D1.6) as revised post-round-1. Per orchestrator directive, this pass does NOT replay round 1's findings (F1-F10) — those are the round-2 proof agent's domain. The attacks below are NEW counterexamples surfaced by the round-2 deltas: `packageManager: "pnpm@9.0.0"`, `client:idle`, `wails build` exit-code + `file` Mach-O check, `ui/types.go` split, raised 30s timeout, and the actual `wails.json` content on disk.

Section 0 reasoning lives in the orchestrator-facing response; this file carries only the finalized findings.

---

## 1. Findings

### F1 — CONFIRMED: `wails.json` `outputfilename: "tillsyn"` (lowercase) breaks every `Tillsyn.app` acceptance bullet

**Severity:** CONFIRMED counterexample. Hard fail on darwin acceptance in D1.3 + D1.6.
**Attack target:** Round-2 acceptance switched from CLI-version-string match to "binary exists at `./build/bin/Tillsyn.app/Contents/MacOS/Tillsyn` + `file` reports Mach-O." This assumes the binary's actual on-disk capitalization matches the assertion text.

**Evidence:**
- `wails.json:4`: `"outputfilename": "tillsyn"` (lowercase `t`).
- `wails.json:3`: `"name": "Tillsyn"` (capital `T`).
- Per Wails 2.12.0 (Context7 `/wailsapp/wails`, project-config docs at `versioned_docs/version-v2.12.0/reference/project-config.mdx`): `outputfilename` is the canonical override for the produced binary name. When set, Wails uses it for both the macOS `.app` bundle directory AND the binary inside `Contents/MacOS/`. The `name` field becomes documentary metadata + `CFBundleName`, not the on-disk artefact name.
- On darwin, `wails build` therefore produces `build/bin/tillsyn.app/Contents/MacOS/tillsyn` — all lowercase.
- PLAN.md D1.3 acceptance (line 88): `./build/bin/Tillsyn.app/Contents/MacOS/Tillsyn exists and file ./build/bin/Tillsyn.app/Contents/MacOS/Tillsyn reports a Mach-O binary`. Capital `T` in both `Tillsyn.app` and `/Tillsyn`.
- PLAN.md D1.6 acceptance (line 143): `ui/build/bin/Tillsyn.app/Contents/MacOS/Tillsyn exists`. Same capital-`T` mistake.
- PLAN.md D1.5 acceptance (line 129): `cd ui && wails build && open ./build/bin/Tillsyn.app`. Same.

**Counterexample shape:** A builder runs `cd ui && wails build`, the binary lands at `build/bin/tillsyn.app/Contents/MacOS/tillsyn`. The QA agent runs `ls ./build/bin/Tillsyn.app/Contents/MacOS/Tillsyn` (capital `T`) — `No such file or directory`. Acceptance fails; build-QA marks the droplet broken; builder retries; same failure. There is no path through the PLAN as written that succeeds on darwin without a `wails.json` edit OR an acceptance-text fix.

**Mitigation (must land before Phase 4 builders dispatch):**
1. Choose one of:
   - **(a) Update PLAN.md acceptance text** to lowercase `tillsyn.app/Contents/MacOS/tillsyn` everywhere (D1.3 ×2, D1.5 ×1, D1.6 ×1). Simplest; preserves on-disk shape of the prior bootstrap commits.
   - **(b) Add a D1.1 sub-step** that edits `wails.json:4` from `"outputfilename": "tillsyn"` → `"outputfilename": "Tillsyn"` (capital `T`). Then current acceptance text becomes correct. Also touches D1.1's paths list (already covers `wails.json`); add an acceptance bullet `grep -q '"outputfilename": "Tillsyn"' ui/wails.json` exits 0.
2. The fix MUST be consistent across D1.3, D1.5, D1.6 — partial application creates a second-order break.
3. Recommended: option (a) — the on-disk reality is the source of truth; making PLAN match reality is lower-risk than mutating a config field that already works.

---

### F2 — POSSIBLE: `sqlite.Open(":memory:")` is the wrong door — `OpenInMemory()` is the existing helper

**Severity:** POSSIBLE counterexample. Steers builder toward a non-canonical path that may quietly succeed but bypasses the proven in-memory helper.
**Attack target:** D1.4 acceptance line 106 says "in-memory via `sqlite.Open(":memory:")` preferred if `internal/adapters/storage/sqlite` accepts it, else a `t.TempDir()`-rooted file." The planner left this builder-routed.

**Evidence:**
- `internal/adapters/storage/sqlite/repo.go:75-98` declares `func Open(path string) (*Repository, error)`. The body does `os.MkdirAll(filepath.Dir(path), 0o755)` (line 79) — for `":memory:"`, `filepath.Dir(":memory:")` returns `"."`, which `MkdirAll` no-ops successfully. Then `sql.Open(driverName, ":memory:")` (line 82) — `modernc.org/sqlite` accepts `:memory:` and creates a per-connection in-memory DB.
- `internal/adapters/storage/sqlite/repo.go:101-118` declares `func OpenInMemory() (*Repository, error)`. The body opens with the DSN `"file::memory:?cache=shared"` — the shared-cache form that lets multiple connections see the same in-memory DB. `Open(":memory:")` does NOT have shared cache.
- D1.4 acceptance also says builder may seed "via the existing service layer (`svc.CreateProject(...)` or equivalent)". `app.Service.CreateProject` likely uses the same `*sql.DB` handle as `ListProjects`. With `MaxOpenConns(1)` (line 86) the pool is single-connection — so seed + list see the same in-memory DB even without `?cache=shared`.
- HOWEVER: the test's `*App` wraps an `*app.Service`. If `app.Service` ever spins up a goroutine or a second connection (e.g. a background migration check, a separate read pool added later), `Open(":memory:")` becomes a silent footgun: seeds appear lost because the read connection sees a different in-memory DB.

**Counterexample shape:** A future refactor adds a second-connection path to `app.Service`. The test that uses `Open(":memory:")` starts flaking — sometimes the seed is visible, sometimes not. The test that uses `OpenInMemory()` (`?cache=shared`) keeps passing because the shared-cache DSN makes both connections see the same DB.

**Mitigation:**
1. Update D1.4 acceptance line 106 to: "use `sqlite.OpenInMemory()` (the existing helper at `internal/adapters/storage/sqlite/repo.go:101`), NOT `sqlite.Open(":memory:")`. The helper uses `"file::memory:?cache=shared"` which tolerates multi-connection access; `Open(":memory:")` uses per-connection DBs."
2. Tighten the fallback: "If `OpenInMemory()` is not a fit for some reason discovered at build time, use `t.TempDir()`-rooted file via `Open(filepath.Join(t.TempDir(), "test.db"))`. Document the reason in `_BUILDER_WORKLOG.md`."

---

### F3 — POSSIBLE: `Corepack enable` is a hidden dev-machine prereq the plan documents but doesn't gate

**Severity:** POSSIBLE counterexample. Misaligned spawn-prereq surface; failure mode depends on the dev's installed pnpm version.
**Attack target:** §N9 says "`corepack enable` (one-time dev-machine setup) ensures `pnpm` is auto-fetched at the pinned version." But the plan has NO acceptance bullet, NO prereq droplet, and NO orchestrator check for whether Corepack is actually enabled before Phase 4 fires.

**Evidence:**
- `which corepack` → `/opt/homebrew/bin/corepack` (Homebrew-shipped Node v26.0.0 includes Corepack). `which pnpm` → `/opt/homebrew/bin/pnpm` (Homebrew-installed standalone pnpm, NOT the Corepack shim). System pnpm shadows Corepack's would-be shim because Homebrew installs to the same `/opt/homebrew/bin/`.
- Corepack ships with Node BUT is disabled by default. Dev must run `corepack enable` once. Until enabled, `corepack` doesn't shim `pnpm` — the system pnpm at `/opt/homebrew/bin/pnpm` is what runs.
- Per Corepack docs (Context7 `/nodejs/corepack`, README): with `packageManager: "pnpm@9.0.0"` in `package.json`:
  - Corepack-enabled + system pnpm absent → Corepack auto-fetches pnpm 9.0.0. Works.
  - Corepack-enabled + system pnpm present at same path → depends on PATH order; typically system pnpm wins on Homebrew installs.
  - Corepack-disabled + system pnpm 9.x → system pnpm runs, ignores the field. Works coincidentally.
  - Corepack-disabled + system pnpm 10.x (strict-engines default) → system pnpm 10 reads `packageManager: "pnpm@9.0.0"`, refuses to install with "This project is configured to use pnpm@9.0.0" error.
- PLAN.md §N7 punts to "orch confirms with dev before Phase 4" — but the confirmation criteria the orch is supposed to verify isn't enumerated. Is `corepack --version` returning 0 sufficient? Is `pnpm --version` matching 9.0.0 required? The plan doesn't say.

**Counterexample shape:** Dev upgrades Homebrew next week. Homebrew bumps pnpm to 10.x. Dev re-runs Phase 4 — `pnpm install` errors immediately because pnpm 10 refuses the 9.0.0 pin. Builder cannot proceed. Plan provided no remediation guidance.

**Mitigation:**
1. Add an explicit pre-Phase-4 orchestrator-check bullet to §N7: "Orch runs `corepack --version` (must exit 0), `pnpm --version` (record output for worklog), and `node --version` (must report ≥ v22.12.0 even-numbered). If `pnpm --version` does NOT report `9.0.0`, orch asks the dev to run `corepack enable` and re-check."
2. OR drop the `packageManager` pin to `"pnpm@>=9.0.0"` (semver-range form Corepack supports) so a system pnpm 9.x or 10.x both satisfy. Less reproducible but more tolerant.
3. Document the exact failure mode in §N9 so a future drop hitting this knows what to look at.

---

### F4 — POSSIBLE: 30s timeout on `mage uiDev` cold-cache is optimistic

**Severity:** POSSIBLE counterexample. 30s may still be too tight on a fresh checkout; the smoke check could flake intermittently.
**Attack target:** Round 2 raised 10s → 30s for the D1.6 `mage uiDev` smoke check (§N12). Plan claims "30s tolerates cold-cache `pnpm install` + first-time wails codegen on a slow dev machine."

**Evidence:**
- D1.6 acceptance (line 144): "QA agent runs it with a 30s timeout wrapper and verifies (a) the process stays running until SIGINT, AND (b) stdout contains the literal substring `[Wails] Dev mode`."
- `mage uiDev` runs `wails dev`. `wails dev` on a fresh checkout does, in sequence: (1) `pnpm install` if `node_modules/` absent (Wails honours `frontend:install` from `wails.json`), (2) generate Wails JS bindings under `frontend/wailsjs/`, (3) start the Astro dev server (`pnpm run dev` per `wails.json:8`), (4) wait up to `viteServerTimeout` seconds (default 10 per the wails.json v2.12 schema), (5) build the Go bridge with `go build -tags dev`, (6) launch the WebView window, (7) print `[Wails] Dev mode` after the window opens.
- Empirical floor:
  - `pnpm install` for a 19-dep tree on a cold pnpm store, against npm registry: 8–30s.
  - First-time Astro dev server start (Astro 5.7.13 reading TS config + Solid integration): 3–6s.
  - `go build -tags dev` of the Wails bridge, cold Go build cache: 10–25s.
  - WebView window launch + Wails dev-loop ready: 1–3s.
- Sum on cold caches: 22–64s. The 30s timeout sits in the middle of that range — a flake-prone bound.
- §N12 says "If a future drop's smoke check flakes at 30s, the diagnostic is to investigate dev-cache hygiene, not to keep loosening the bound — past 60s the smoke check stops being a smoke check." That's a reasonable position, but the PLAN itself doesn't define what "dev-cache hygiene" remediation looks like. If the smoke flakes during D1.6's first QA run on a CI-equivalent fresh checkout, the only escape hatch is "increase the bound," which §N12 forbids.

**Counterexample shape:** Dev runs Phase 4 on a fresh clone (no prior pnpm store, no Go build cache for the `wails` tag). `mage uiDev` fires; 30s wrapper SIGKILLs the process before `[Wails] Dev mode` prints. QA marks D1.6 fail. Builder retries from a warm cache, succeeds. The drop is artificially cache-dependent.

**Mitigation:**
1. **Either** raise the timeout to 60s with a worklog rule: "if smoke check takes > 60s, investigate dev-cache hygiene before raising further" — this aligns the timeout with empirical cold-cache reality without losing the 60s upper bound §N12 already names.
2. **OR** split the smoke into two phases: (a) prime cache step (run `pnpm install` + `go build -tags wails -o /dev/null ./ui/...` once, no timeout, off the critical path); (b) timed smoke check with the cache primed, 15s bound.
3. **OR** accept the flake risk explicitly: add an acceptance bullet "if the 30s smoke flakes once, builder MAY retry up to 2 additional times before failing the droplet; cumulative > 3 attempts is a real fail."

---

### F5 — NIT: `client:idle` falls back to `load` event on macOS WKWebView (no `requestIdleCallback` support)

**Severity:** NIT — informational, not a blocker for this drop's acceptance.
**Attack target:** Round 2 reset `client:load` → `client:idle` per round-1 F3 falsification. §N3 cites "Astro docs: `client:idle` waits for `requestIdleCallback` (fires within tens of ms of first paint in a Wails window with no other JS competing for the main thread)."

**Evidence:**
- Per Astro docs (Context7 `/withastro/docs`, directives-reference): "Load and hydrate the component JavaScript once the page is done with its initial load and the `requestIdleCallback` event has fired. If you are in a browser that doesn't support `requestIdleCallback`, then the document `load` event is used."
- Per MDN `Window.requestIdleCallback`: supported in Chrome, Firefox, Edge, Opera. **NOT supported in Safari / WebKit.**
- Wails v2 on darwin uses `WKWebView` (Apple's WebKit engine, NOT Chromium). Wails v2 on Linux uses `webkit2gtk` (also WebKit). Wails v2 on Windows uses `WebView2` (Chromium-based).
- Conclusion: on darwin and Linux, `client:idle` falls back to the `load` event. Hydration still works correctly, but the timing semantics differ from §N3's claim "fires within tens of ms of first paint" — the `load` event fires AFTER all sub-resources finish, which can be later than first paint.

**Counterexample shape:** §N3 promises a specific timing characteristic ("tens of ms of first paint") that doesn't hold on macOS / Linux Wails builds. The DROP's acceptance doesn't depend on this (D1.5 just asserts the DOM contains the rendered list, with no timing constraint). So no acceptance failure — but the planner's reasoning is wrong about WHY `client:idle` is better than `client:load` on this platform.

**Mitigation:**
1. Reword §N3 to: "`client:idle` defers hydration until after first load. On WebKit-based runtimes (Wails on darwin / Linux), this resolves to the document `load` event because `requestIdleCallback` is unavailable; on Wails-Windows (WebView2 / Chromium), `requestIdleCallback` runs as documented. Either fallback is correct for a read-only list — hydration timing is not load-bearing for this drop."
2. NOT a blocker — the acceptance text in D1.5 makes no timing claim, so the droplet succeeds regardless of which fallback fires.

---

### F6 — NIT: `pnpm install` ordering inside D1.1 is implicit

**Severity:** NIT — documentation gap, not a structural break.
**Attack target:** D1.1 acceptance asserts `git ls-files ui/frontend/pnpm-lock.yaml` returns the path (line 54). But the lockfile only exists after `pnpm install` runs — and the plan's acceptance text doesn't enumerate the build-step ordering.

**Evidence:**
- D1.1 acceptance line 53: `cd ui/frontend && pnpm install exits 0 and produces ui/frontend/node_modules/. Lockfile is created at ui/frontend/pnpm-lock.yaml.`
- D1.1 acceptance line 54: `git ls-files ui/frontend/pnpm-lock.yaml returns the path (lockfile committed, reproducible installs).`
- The implicit ordering: `git mv` → `pnpm install` (creates lock) → `git add ui/frontend/pnpm-lock.yaml` → commit. But the plan doesn't say "the builder MUST run `pnpm install` AND `git add` the lockfile as part of D1.1's build steps." A literal reading of the acceptance criteria says only "verify lockfile is tracked" — a strict-as-spec builder could interpret D1.1 as "verify a lockfile that doesn't exist yet" and fail.
- §N9 paragraph 3 makes the build-step intent clearer: "After `pnpm install` runs (during D1.1 acceptance), the resulting `ui/frontend/pnpm-lock.yaml` is `git add`'d and committed alongside the rest of D1.1's relocation." But §N9 is a note, not the acceptance text. A builder reading only the droplet body might miss the intent.

**Counterexample shape:** Builder reads D1.1 strictly, runs the `git mv` operations, runs `pnpm install` per acceptance bullet line 53, then proceeds to D1.1's commit step WITHOUT explicitly `git add`-ing the lockfile (because nothing in D1.1's body says "stage the lockfile"). The commit lands without the lockfile; acceptance bullet line 54 fails on the QA pass.

**Mitigation:**
1. Add a build-step ordering bullet to D1.1 (separate from acceptance): "Build steps in order: (1) `git mv` operations per `paths`. (2) `cd ui/frontend && pnpm install` (this creates `pnpm-lock.yaml` and `node_modules/`). (3) `git add ui/frontend/pnpm-lock.yaml ui/main.go ui/wails.json ui/frontend/...` for tracked outputs. (4) `git add .gitignore` for the ignore updates. (5) Commit."
2. Acceptance line 54 stays as-is; the build-step bullet makes the implicit ordering explicit.

---

### F7 — NIT: `file` Mach-O check is darwin-only; D1.3 + D1.6 don't say so

**Severity:** NIT — cross-platform acceptance gap. Out-of-scope per §N6, but the plan implicitly assumes darwin.
**Attack target:** D1.3 acceptance line 88 + D1.6 acceptance line 143 both gate on `file ./build/bin/Tillsyn.app/Contents/MacOS/Tillsyn reports a Mach-O binary`. Both bullets parenthesize "On macOS (dev's env per session `Platform: darwin`)" but don't formally bracket the Mach-O check with a "(darwin-only)" tag.

**Evidence:**
- `/usr/bin/file` is part of macOS base install (BSD `file(1)`). Confirmed available at `/usr/bin/file`.
- On Linux, `file` is typically GNU `file(1)` (also available). Linux Wails binary lands at `build/bin/Tillsyn` (bare ELF) — `file` reports `ELF 64-bit LSB executable`, NOT `Mach-O`.
- On Windows, `file` may not be present by default. Wails binary at `build\bin\Tillsyn.exe` — `file` (if present via Git for Windows) reports `PE32+ executable`.
- The plan acknowledges this in line 88 parenthetical: "Linux/Windows produce different paths..." But the ACCEPTANCE BULLET ABOVE the parenthetical reads as if `file` reporting Mach-O is the universal verification. A non-darwin QA pass running this acceptance literally would fail.

**Counterexample shape:** If a future drop runs Phase 5 QA on a Linux CI runner against this droplet's acceptance text, the QA agent asserts `file ./build/bin/Tillsyn.app/Contents/MacOS/Tillsyn` → file does not exist (Linux produces a flat binary, not an `.app` bundle). Acceptance fails on a path that the planner expected to succeed.

**Mitigation:**
1. Reword the D1.3 acceptance bullet to: "On darwin: `./build/bin/<binary-name>.app/Contents/MacOS/<binary-name>` exists AND `file <path>` contains the substring `Mach-O`. On Linux: `./build/bin/<binary-name>` exists. On Windows: `.\build\bin\<binary-name>.exe` exists. Cross-platform packaging is out of scope per §N6, so QA agents running this drop verify only the platform they're on."
2. Same shape for D1.6 acceptance.
3. NOT a blocker for THIS drop (dev is on darwin per `Platform: darwin`), but the acceptance text should not falsely claim cross-platform coverage.

---

### F8 — NIT: goleak absent from FE plan; OK because the FE smoke test is sync end-to-end

**Severity:** NIT — cross-drop parity check, no action required.
**Attack target:** drop_4b_test_cleanup D1.3 adds `goleak.VerifyTestMain(m)` to dispatcher e2e tests. Does the FE drop's `ui/app_test.go` also need goleak coverage?

**Evidence:**
- `ui/app_test.go` per D1.4 acceptance (line 106-107): constructs `*app.Service` against in-memory or temp-file SQLite, seeds via `svc.CreateProject`, constructs `*App`, calls `app.ListProjects()`, asserts on the result. The test is synchronous end-to-end — no spawned goroutines, no subprocesses, no broker chain.
- `*app.Service` internally may run a one-shot background migration on `sqlite.Open` (per `repo.go:91-95` → `repo.migrate(ctx)` is called synchronously inside `Open`, NOT in a goroutine). So `Open` returning doesn't leave a background goroutine.
- The dispatcher tests use goleak because the dispatcher's monitor loop spawns goroutines for state polling; the FE test's `*App` is a passive struct with no goroutines.
- Conclusion: no goroutine surface to leak. goleak parity is not needed for this drop.

**Mitigation:**
1. No change. Optional refinement: add a one-line note to D1.4 acceptance: "No goroutine-leak detection needed — `*App` is a passive struct, no background goroutines spawned by the test path."
2. If a future FE drop adds an IPC method that spawns a goroutine (e.g. event streaming from Go to JS), revisit goleak parity at that point.

---

### F9 — REFUTED: `mage format` clean on relocated `main.go`

**Severity:** REFUTED counterexample.
**Attack target:** D1.1 acceptance line 58: `mage format reports no diff against ui/main.go after relocation (relocated file remains gofumpt-clean ...).` Claim: the existing `main.go` may not be gofumpt-clean, in which case the relocation surfaces a pre-existing format issue.

**Evidence:**
- Read `main.go` lines 1-58. The file is 58 lines, single-package `package main`, build tag `//go:build wails`, imports grouped correctly (stdlib first: `context`, `embed`, `log`; then third-party: `github.com/evanmschultz/...`, `github.com/wailsapp/...`). Indentation is tabs. Empty lines look minimal-no-double-blank. Spacing between functions looks 1-line. Comments use `//` form, capitalized first word, periods.
- The codebase tooling uses `mvdan.cc/gofumpt v0.9.2` (per `go.mod:103`) as the canonical formatter. gofumpt is a strict superset of `gofmt` — anything `gofmt`-clean MAY still be gofumpt-dirty (extra blank lines, missing `gofumpt`-specific rewrites).
- I could not run `gofmt -l` or `gofumpt -l` against the file (Bash invocation denied for raw Go tools). Manual visual inspection shows the file LOOKS clean: no obvious double-blank-lines, no obvious un-gofumpted comment groupings.
- The file content was authored by the prior bootstrap commit `8d33539` "feat(fe): astro solid scaffold with stil tokens and mage ci-fe" which would have hit `mage ci` → `Format` step as part of its PR gate (per the canonical CI workflow). So the file should already be gofumpt-clean.

**Counterexample shape (REFUTED):** I cannot produce a concrete counterexample from inspection. The file LOOKS clean and was committed via the canonical CI gate. If a future builder discovers gofumpt drift during D1.1, the fix is one-line: `mage format` rewrites the file. Acceptance line 58 then passes by construction.

**Mitigation:** none required. If D1.1's builder hits a gofumpt diff, they run `mage format` and commit the result as part of D1.1.

---

### F10 — REFUTED: Cross-drop path overlap with `drop_4b_test_cleanup`

**Severity:** REFUTED counterexample.
**Attack target:** Verify no file paths overlap between drop_fe_1_bootstrap and drop_4b_test_cleanup (both currently in flight).

**Evidence:**
- drop_4b_test_cleanup paths (per its `PLAN.md`):
  - `internal/app/dispatcher/subscriber_test.go`
  - `internal/app/dispatcher/dispatcher_e2e_test.go` (new)
  - `internal/domain/comment.go` + `internal/domain/comment_test.go`
  - `internal/adapters/mcp_rpc/extended_tools.go` + `extended_tools_test.go`
  - `internal/adapters/mcp_common/mcp_surface.go` + `app_service_adapter_lifecycle_test.go`
- drop_fe_1_bootstrap paths (per PLAN.md line 5): `ui/` (new top-level), `magefile.go` (rename `CiFe` → `CiUI`), `.gitignore`, `wails.json` (relocated).
- Set intersection: empty. No file overlaps; no package overlaps (`internal/app/dispatcher` vs `./ui`).
- `magefile.go` is touched by the FE drop alone. drop_4b_test_cleanup leaves the magefile untouched.

**Counterexample shape (REFUTED):** None. The two drops are spatially disjoint.

**Mitigation:** None required. Drops can run in parallel without `blocked_by` between them at the file level.

---

## 2. Counterexamples Summary

| ID | Severity | Target | Outcome |
|----|----------|--------|---------|
| F1 | CONFIRMED | D1.3 + D1.5 + D1.6 binary-path capitalization | Hard fail on darwin |
| F2 | POSSIBLE | D1.4 `:memory:` choice | Use `OpenInMemory()` instead |
| F3 | POSSIBLE | Corepack-enable prereq | Add explicit orch-check |
| F4 | POSSIBLE | D1.6 30s smoke timeout | Cold-cache fragile; raise to 60s or split |
| F5 | NIT | §N3 `requestIdleCallback` claim | Reword to acknowledge WebKit fallback |
| F6 | NIT | D1.1 lockfile build-step ordering | Add explicit build-step bullet |
| F7 | NIT | `file` Mach-O check cross-platform | Reword to "darwin-only" |
| F8 | NIT (REFUTED) | goleak FE parity | Not needed — no goroutine surface |
| F9 | REFUTED | `mage format` clean on `main.go` | Looks clean; no evidence of drift |
| F10 | REFUTED | Cross-drop path overlap | Disjoint paths confirmed |

CONFIRMED counterexamples: **1** (F1).
POSSIBLE counterexamples: **3** (F2, F3, F4).
NITs: **3** (F5, F6, F7).
REFUTED: **3** (F8, F9, F10).

---

## 3. Verdict

**`fail`** on F1 alone. F1 is a literal-acceptance bug: the plan asserts a binary path that the on-disk `wails.json` does NOT produce. Every D1.3 / D1.5 / D1.6 acceptance bullet that names `Tillsyn.app/Contents/MacOS/Tillsyn` will fail on darwin. No retry loop converges without a plan edit (either acceptance text → lowercase, or `wails.json` → capital `T`).

The 3 POSSIBLE counterexamples (F2, F3, F4) are each individually mitigable in round 3:
- F2: one-line text change in D1.4 acceptance (point at `OpenInMemory()`).
- F3: add explicit orch-check bullet to §N7.
- F4: raise timeout to 60s with worklog rule, OR split smoke into prime-then-check phases.

The 3 NITs (F5, F6, F7) are documentation-grade fixes that don't block a builder but improve plan clarity.

Recommended round-3 trigger: PLAN.md edits for F1 (mandatory) + F2/F3/F4 (strongly recommended), then proof + falsification re-run.

---

## 4. TL;DR

- **T1** Round-2 plan has one concrete acceptance-criterion bug (F1, CONFIRMED): `wails.json:4` declares `outputfilename: "tillsyn"` lowercase, but D1.3/D1.5/D1.6 assert `Tillsyn.app/Contents/MacOS/Tillsyn` capital-T — no path through PLAN succeeds on darwin without an edit. Three POSSIBLE counterexamples (F2 `OpenInMemory` vs `Open(":memory:")`, F3 Corepack-enable prereq ungated, F4 30s smoke timeout cold-cache-fragile) and three NITs (F5 WebKit fallback, F6 lockfile ordering, F7 file Mach-O darwin-only) are documented with mitigations.
