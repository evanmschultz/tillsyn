# Drop FE 1 Bootstrap — Plan QA Falsification — Round 1

**Verdict:** `fail` (2 CONFIRMED counterexamples, 4 POSSIBLE counterexamples, 4 NITs)
**Round:** 1
**Reviewer:** fe-qa-falsification-agent
**Date:** 2026-05-18

Falsification pass against `PLAN.md` (D1.1 – D1.6). Goal: actively try to break the plan via concrete counterexamples backed by file:line evidence. Section 0 reasoning lives in the orchestrator-facing response and is intentionally not duplicated here.

---

## 1. Findings

### F1 — POSSIBLE: `//go:embed all:frontend/dist` in `main.go` — acceptance gap, not a breakage

**Severity:** POSSIBLE counterexample. Acceptance criterion is silent on the embed line; relocation coincidentally works.
**Attack target:** D1.1 claim that a clean `git mv` of `main.go` → `ui/main.go` + `frontend/` → `ui/frontend/` is a no-op-equivalent relocation.

**Evidence:**
- `main.go:16` declares `//go:embed all:frontend/dist`. The `//go:embed` directive resolves paths **relative to the source file's directory**, not relative to the module root.
- After `git mv main.go ui/main.go` + `git mv frontend ui/frontend`, the `//go:embed` directive in `ui/main.go` would resolve to `ui/frontend/dist` (relative to `ui/main.go`) — which is the correct new location.
- **BUT** D1.1 only renames; it does not edit `main.go`'s body. The directive text stays literally `all:frontend/dist`. The path `ui/frontend/dist` happens to satisfy `frontend/dist` relative to `ui/main.go` — so this part actually works by coincidence of the relative-path semantics. **Recheck:** with `ui/main.go` containing `//go:embed all:frontend/dist`, Go resolves `frontend/dist` against `ui/`'s file dir → `ui/frontend/dist`. That IS correct.
- **However**, D1.1 acceptance explicitly says: "`cat ui/main.go | head -1` shows `//go:build wails` build tag still present" — that line check passes, but the **embed-target preconditions** are not in acceptance. If a builder mistakenly thinks "the embed path needs updating to `ui/frontend/dist`" and edits the file, the embed breaks (becomes `ui/ui/frontend/dist`). The acceptance criterion is silent on whether the embed line should stay as `frontend/dist` (correct) or change to `ui/frontend/dist` (broken).

**Sharper attack — Wails `wails.json` `frontend:dir` field:**
- `wails.json:5` declares `"frontend:dir": "frontend"`. Per the Wails 2.x project-config docs (Context7, `/wailsapp/wails`): "`wails.json`, located in the project's root directory, serves as the central configuration hub" — i.e., `wails.json` IS the project root marker, and `frontend:dir` is resolved relative to the `wails.json` location.
- After `git mv wails.json ui/wails.json` + `git mv frontend ui/frontend`, the `frontend:dir` value `"frontend"` resolves to `ui/frontend/` (relative to `ui/wails.json`). D1.1's acceptance bullet correctly notes this: `"frontend:dir": "frontend" (relative to the wails.json location — i.e. ui/frontend/)`.
- **HOWEVER** this assumes Wails 2.12.0 resolves `frontend:dir` relative to `wails.json` parent dir and NOT relative to the **invocation cwd** (`cd ui && wails build`). The plan's D1.3 acceptance assumes `cd ui && wails build` works. If `wails` CLI uses the cwd instead of `wails.json` dir for resolution, the relocation still works because they coincide. But the planner has not verified this against the Wails 2.x source or docs. Context7 confirms `wails.json` is the "project root" — so this is likely fine, but D1.1 acceptance should add a positive smoke check: `cd ui && wails build` exits 0 (currently only `pnpm run build` is checked in D1.1; the `wails build` smoke is deferred to D1.3).

**Counterexample shape:** D1.1's claim is "git mv suffices + small `.gitignore` edits." But there is one path-relative concern lurking — the `//go:embed all:frontend/dist` line. Confirmed not-broken-by-the-rename, but D1.1's acceptance does NOT positively verify that `go build -tags wails ./ui/...` (or equivalent) finds the embed target. If the builder relocates BEFORE `ui/frontend/dist/` exists (i.e., before running `pnpm run build`), then any `go build -tags wails ./ui/...` invocation fails with `pattern frontend/dist: no matching files found`.

**Mitigation (must land before Phase 4):**
1. Add explicit D1.1 acceptance line: "`grep -n '//go:embed' ui/main.go` shows the directive text is **literally unchanged** — still `//go:embed all:frontend/dist`, NOT `all:ui/frontend/dist`. Confirms the path is correctly relative to `ui/main.go`'s dir, not to the module root."
2. Add explicit D1.1 acceptance line: "After `cd ui/frontend && pnpm install && pnpm run build`, `ls ui/frontend/dist/index.html` exists — so the `//go:embed` target is materialized for any downstream `go build -tags wails`."
3. Move the `wails build` smoke check from D1.3 to D1.1 (or add a redundant smoke in D1.1) so the relocation itself is proven against the Wails CLI before any code change lands on top.

---

### F2 — CONFIRMED: No `pnpm-lock.yaml`, no `packageManager` field — `pnpm install` is fragile + REVISION_BRIEF says "Node + npm"

**Severity:** CONFIRMED counterexample. Blocks D1.1 acceptance + D1.5 + D1.6.
**Attack target:** Plan assumes `pnpm` is available; REVISION_BRIEF §1 says "Node + npm available on dev machine".

**Evidence:**
- `REVISION_BRIEF.md:13` reads: "Node + npm available on dev machine (Wails frontend templates use them)." Explicitly names `npm`, NOT `pnpm`.
- `wails.json:6-9` declares: `"frontend:install": "pnpm install"`, `"frontend:build": "pnpm run build"`, `"frontend:dev:watcher": "pnpm run dev"`. Wails 2.12.0 default template uses `npm install` / `npm run build` (Context7 — `/wailsapp/wails` shows default `"frontend:install": "npm install"`). The repo's `wails.json` has already been customized to pnpm by the prior bootstrap commits (`a9bac6c` / `8d33539`), but the REVISION_BRIEF was authored without that customization being surfaced.
- `frontend/package.json:1-28` has **no `packageManager` field**. Node 16.13+ honours `packageManager` as a Corepack hint; without it, `pnpm install` on a dev machine that has only `npm` available fails immediately.
- **`ls frontend/` shows no `pnpm-lock.yaml`** (only `astro.config.mjs`, `package.json`, `tsconfig.json`, `public/`, `src/`, `tests/`). pnpm projects normally check in `pnpm-lock.yaml`; its absence means either (a) the prior bootstrap commits did not actually run `pnpm install` against this checkout, or (b) `pnpm-lock.yaml` is in `.gitignore`. Either way, the dev installing pnpm fresh and running `pnpm install` will resolve from scratch with no lock pin — version drift is uncontrolled.
- D1.1 acceptance line 49: `cd ui/frontend && pnpm install exits 0`. D1.2 acceptance: `pnpm run test:unit` + `pnpm run build`. D1.5 acceptance: `pnpm run build` + `pnpm run test:unit`. D1.6 acceptance: `mage uiDev` runs `wails dev` which invokes `pnpm run dev`. Five droplets directly depend on `pnpm` working.

**Counterexample shape:** Dev confirms "wails CLI is on PATH" per Note N7. Dev does NOT confirm pnpm. Phase 4 builders fire. D1.1 builder runs `cd ui/frontend && pnpm install` → command-not-found → builder reports failure → orchestrator must escalate to dev → drop stalls before any code change.

**Mitigation (must land before Phase 4):**
1. EITHER pin `pnpm` via `packageManager` field in `frontend/package.json` (e.g. `"packageManager": "pnpm@9.0.0"`) so Corepack auto-fetches it; D1.1 includes this edit in scope.
2. OR switch all FE scripting to `npm` (matching REVISION_BRIEF §1 verbatim): `wails.json` `frontend:install/build/dev:watcher` change to `npm install` / `npm run build` / `npm run dev`; D1.2/D1.5/D1.6 acceptance updated to `npm run`.
3. OR add explicit "dev confirms pnpm is on `$PATH`" line to Note N7 alongside the existing `wails` CLI check, and add `mage` precondition probe target.
4. Independent of choice, add `pnpm-lock.yaml` (or `package-lock.json`) under version control — currently absent.

The plan does not pick among (1)/(2)/(3). It silently assumes pnpm works.

---

### F3 — POSSIBLE: `client:load` choice contradicts FE-planning-agent doctrine; Note N3's "acceptance-determinism" rationale is weak

**Severity:** POSSIBLE counterexample. Non-blocker; orchestrator/dev judgment call.
**Attack target:** D1.5 `<ProjectList client:load />` directive + Note N3's justification.

**Evidence:**
- D1.5 acceptance lines 104-106 + Note N3 lines 162-164 specify `client:load` "because the proof view is the **only** content on the page and we want eager hydration to make the QA acceptance check ('the list renders within 2 seconds of window open') deterministic."
- Astro client-directive docs (Context7, `/withastro/docs`): "`client:load` immediately loads and hydrates the component's JavaScript when the page loads. It is ideal for high-priority UI elements that require immediate interactivity." `client:idle` waits for `requestIdleCallback`. `client:visible` waits for IntersectionObserver.
- The ProjectList component is **read-only** — there is no interactivity that justifies "high-priority." The async `createResource` call against `window.go.main.App.ListProjects` does not start until the component hydrates. With `client:idle`, hydration runs after the browser is idle (typically a few hundred ms after first paint) — still well within "2 seconds." With `client:visible`, it would run as soon as the component scrolls into view (irrelevant here since it's the only content above the fold).
- Note N3's own conclusion concedes: "Future FE drops should default to `client:idle` per FE-planning-agent doctrine; the eager-load choice here is a one-drop acceptance-determinism call."
- The "deterministic acceptance" argument is weak: `wails dev` window opens with the ProjectList above the fold, so `client:visible` and `client:idle` both fire within tens of ms of window paint. The plan does not include a runtime measurement showing `client:load` is actually faster or more deterministic than `client:idle` in the Wails-windowed context.
- **Pattern risk:** D1.5 sets the FIRST island in the codebase. The next FE drop will pattern-match on D1.5. If D1.5 says `client:load`, the next drop's planner / builder will reach for `client:load` too — making "future drops use `client:idle`" a paper rule rather than encoded discipline. The cheap fix is to set the right default NOW, not later.

**Counterexample shape:** A future FE QA falsification round attacks the cargo-culted `client:load` usage propagated from D1.5; the team has to retro-fix every island instead of having set the right default in the bootstrap drop.

**Mitigation (recommend dev decide):**
1. Switch D1.5 to `<ProjectList client:idle />`, drop the "acceptance-determinism" justification, and add an acceptance bullet that the list renders within 2 s of window open regardless of directive (which `client:idle` satisfies).
2. OR keep `client:load` but add an explicit `// reason: acceptance-determinism for bootstrap drop only; future islands default to client:idle` comment in `index.astro` so the pattern doesn't propagate silently.
3. (Not recommended) Keep `client:load` with no comment.

---

### F4 — POSSIBLE: Inline DTO in `ui/main.go` couples IPC surface to entrypoint file; YAGNI cuts both ways

**Severity:** POSSIBLE counterexample. NIT-leaning; non-blocker.
**Attack target:** D1.4 inline `ProjectDTO struct { ID string; Name string }` in `ui/main.go`. Note N2's YAGNI rationale.

**Evidence:**
- D1.4 lines 89-95 + Note N2 lines 158-160 argue: "for this drop's scope (one read-only method, one DTO), a separate `ui/bridge/` package is YAGNI."
- The Wails `Bind` mechanism (`main.go:51-53`) requires the bound struct (`App`) and its exported methods to live in `package main` so the codegen `wailsbindings` tool can locate them. Splitting `App` itself into `ui/bridge/` is non-trivial — agreed, YAGNI for this drop.
- **But the DTO is separable.** `ProjectDTO` is a pure data type with no Wails-binding dependency. A `ui/dto.go` file (still `package main`, same dir) would hold `ProjectDTO` without an import boundary, keeping `main.go` smaller. The plan opts for inline. Note N2 conflates "no `ui/bridge/` package" (justified) with "inline DTO in `main.go`" (separate decision, less justified).
- **Sibling churn risk:** D1.3 and D1.4 both edit `ui/main.go` serially (D1.4 `blocked_by` D1.3). When the next FE drop adds 3 more IPC methods, each one extends `ui/main.go` further, sequentially. By drop 3 of the FE lane, `ui/main.go` will be ~200 LOC of mixed wiring + DTOs + methods.
- Not a blocker; refactoring out a `ui/dto.go` later is one droplet of work. But the plan could pre-empt the bloat with a 2-LOC choice now.

**Counterexample shape:** Future FE drop's plan-QA falsification attacks `ui/main.go` LOC bloat and recommends an extract that this drop's plan-QA could have pre-empted.

**Mitigation (recommend dev decide):**
1. Move DTO type into a sibling `ui/types.go` (still `package main`, same dir, no import boundary). D1.4 writes both `ui/main.go` and `ui/types.go`. Adds 1 file to D1.4 paths.
2. OR accept the inline DTO + plan a future split when LOC warrants. Note N2 already implicitly accepts this — but it should be explicit, not buried.

---

### F5 — CONFIRMED (low-impact): `wails` CLI version is not pinned anywhere

**Severity:** CONFIRMED counterexample. Non-blocker for D1.1-D1.5 acceptance (Go bindings tolerant); blocker for D1.6's "Wails CLI v2.12.0" stdout marker check.
**Attack target:** Plan acceptance that assumes specific Wails CLI version.

**Evidence:**
- `go.mod:93` pins `github.com/wailsapp/wails/v2 v2.12.0` as an indirect Go dependency (Wails Go bindings).
- D1.6 acceptance line 127: "QA agent runs it with a short timeout wrapper and verifies stdout shows `Wails CLI v2.12.0` + `[Wails] Dev mode` markers within 10s."
- **There is no `wails` CLI version pin anywhere in the repo** (no `.wails-version` file, no `tool.wails` block in any TOML, no `wails-cli` `package.json` dep, no version constraint in Note N7 or REVISION_BRIEF). The dev's installed `wails` CLI could be v2.9, v2.10, v2.11, v2.13, etc.
- Wails CLI ↔ Go-bindings version compatibility: Wails 2.x maintains forward compatibility within a minor band, but mismatched major.minor pairings can fail silently or produce subtle codegen drift. The `wails build` step relies on the CLI's `wailsbindings` codegen running against the installed Go-bindings module — version drift here causes "App is bound but `window.go.main.App.ListProjects` is undefined" errors that look like Go-side bugs.
- D1.6 acceptance pins the **string match** to `Wails CLI v2.12.0`. If the dev has CLI v2.11.x or v2.13.x installed, the QA check fails the string match even though the binary builds and runs correctly.

**Counterexample shape:** Dev runs `wails doctor` per Note N7, sees green output, Phase 4 builders fire. D1.6's QA agent runs `mage uiDev`, captures stdout, fails the `Wails CLI v2.12.0` substring match because the dev's installed CLI is v2.11 or v2.13. Builder respawns, fixes nothing real, QA still fails. Drop stalls on a string match, not a wiring problem.

**Mitigation (must land before Phase 4):**
1. Soften D1.6 acceptance to match `Wails CLI v2\.\d+\.\d+` (regex) — only the major version is load-bearing for the wiring test.
2. AND pin the expected CLI minor version somewhere — Note N7 should add: "Dev confirms `wails version` output starts with `v2.12` (matching the Go-bindings `v2.12.0` in `go.mod`). If newer/older, the plan's D1.6 acceptance regex needs updating, NOT relax-to-any."
3. The version pin lives in the brief, not in code today (Wails CLI doesn't have a project-local version-pin mechanism analogous to `packageManager`). Accept the gap with documentation.

---

### F6 — POSSIBLE: D1.3's `wails build` smoke check is environment-fragile (codesigning + macOS-only)

**Severity:** POSSIBLE counterexample. Non-blocker but acceptance is too tight.
**Attack target:** D1.3 acceptance "`cd ui && wails build` produces a binary; `./build/bin/Tillsyn.app/Contents/MacOS/Tillsyn` starts without panic."

**Evidence:**
- D1.3 line 79 hardcodes the macOS `.app` bundle path. The CI matrix at `.github/workflows/ci.yml:18-21` runs only `macos-latest` today, so macOS-only is operationally OK for THIS drop.
- BUT `wails build` on macOS, even unsigned, can fail with "App is damaged" or Gatekeeper rejection depending on macOS minor version + dev's xattr state. The acceptance says "starts without panic" but does NOT say "passes Gatekeeper" or "user has dismissed `xattr -d com.apple.quarantine`."
- D1.3's acceptance "verified manually by orch-or-dev launching the binary" pushes the hardest check (does the window actually open) to a human eyeball — but the planner has also assigned that check to the QA agent via "QA agent verifies via `wails build` exit code 0 + presence of the output binary." This is two contradictory acceptance modes for the same line.
- D1.4 line 94 then says "wails build headless inspection requires a runtime probe — the build-success + dev-mode probe pair is sufficient acceptance" — implicitly admitting `wails build` cannot be fully QA-agent-verified.

**Counterexample shape:** QA agent verifies `wails build` exit 0 and `./build/bin/Tillsyn.app/Contents/MacOS/Tillsyn` exists. Build is signed correctly enough to pass that file existence check, but launching the binary hits a Gatekeeper or runtime symbol-resolution error that the QA agent does not surface. Drop closes with broken-on-dev-machine binary; orchestrator only finds out at drop close when dev tries it.

**Mitigation:**
1. Split D1.3 acceptance into "QA-agent-verifiable" (build exit 0 + binary file exists + `file` command reports Mach-O) and "dev-runtime-verifiable" (binary launches, window opens). Make the dev check explicit in Phase 7, not implicit in D1.3 acceptance.
2. OR drop the `wails build` requirement from D1.3 entirely and rely on `wails dev` for IPC wiring proof. The `wails build` smoke move to D1.6 (which already runs it) — D1.3 only needs to prove the Go-side construction compiles + `wails dev` opens.

---

### F7 — NIT: Cross-drop `_BLOCKERS.toml` integrity — `drop_4b_test_cleanup/` does not touch this drop's paths, OK

**Severity:** NIT. Not a counterexample; recorded for completeness.
**Attack target:** Cross-drop file overlap with parallel drop `drop_4b_test_cleanup/`.

**Evidence:**
- `drop_4b_test_cleanup/PLAN.md` (read first 60 lines) declares paths under `internal/domain/`, `internal/adapters/mcp_rpc/`, `internal/app/`, `internal/app/dispatcher/`. No overlap with `ui/`, `main.go`, `wails.json`, `frontend/`, or `magefile.go`'s `CiFe` function body.
- `drop_fe_1_bootstrap/PLAN.md` declares `ui/`, `magefile.go` (specifically `CiFe`→`CiUI` rename), `.gitignore`. The `magefile.go` edit is the only potential collision surface; `drop_4b_test_cleanup` does not touch `magefile.go`.

**Counterexample:** None. The two parallel drops are file-disjoint.

**Mitigation:** None needed. NIT only — note in PLAN.md cross-references section that `drop_4b_test_cleanup/` is concurrent but file-disjoint, so no cross-drop `blocked_by` required.

---

### F8 — NIT: `mage ci` does NOT compile `ui/main.go` even without the build tag — Go's package-discovery semantics

**Severity:** NIT. Confirms the plan's claim; documents the mechanism explicitly.
**Attack target:** Plan repeatedly claims `mage ci` "stays green via build-tag `wails` fence" (D1.1, D1.3, D1.4, D1.6 acceptance).

**Evidence:**
- `magefile.go:212-229` (the `CI()` function body) stages are: `verifySources`, `formatCheck`, `coverage` (runs `go test -cover ./...`), `Build` (runs `go build -o ./till ./cmd/till`), `TestIntegration` (`go test -tags integration ./internal/templates/...`).
- `coverage` invokes `go test ./...` which DOES walk the entire module tree. Without the `wails` build tag set in the default test flags, files with `//go:build wails` are excluded from compilation. ✅ Plan claim holds.
- BUT: after D1.1's `git mv main.go ui/main.go`, the file `ui/main.go` is the only Go file in `ui/` (D1.2 doesn't add another). `go test ./...` discovers `./ui` as a package. With `//go:build wails`, all files in `./ui` are excluded → Go reports `no Go files in ./ui` → `go test` treats this as a non-test package, **not an error**.
- However, `go vet` and `gofumpt -l` (`formatCheck` calls `go tool gofumpt -l <tracked files>`) DO scan files regardless of build tags. `magefile.go:412-426` `trackedGoFiles()` returns every `*.go` file via `git ls-files *.go`, including `ui/main.go`. `gofumpt -l` should format-check `ui/main.go` regardless of the build tag.
- **Sub-attack:** if `ui/main.go` is not gofumpt-clean, `mage ci` fails at the `formatCheck` stage, NOT at compile. The plan's repeated "build-tag fence keeps `mage ci` green" claim is true for compile but not automatically true for format. **Verify** that the pre-existing `main.go` is gofumpt-clean today — if it is, `git mv` preserves clean state.

**Counterexample shape:** None concrete. Mechanism is "watch the gofumpt step, not the compile step" — recorded so the build-QA agent doesn't miss it.

**Mitigation:** None needed. Confirms the plan. Add one line to D1.1 acceptance: "`mage format` reports no diff against `ui/main.go` after relocation."

---

### F9 — NIT: D1.6's `mage UIDev` 10-second timeout probe is heuristic and CI-fragile

**Severity:** NIT. Mostly cosmetic.
**Attack target:** D1.6 acceptance line 127.

**Evidence:**
- D1.6 expects `mage uiDev` to print stdout markers `Wails CLI v2.12.0` and `[Wails] Dev mode` within 10s. On a cold-cache `pnpm install` (no `pnpm-lock.yaml` — see F2) plus first-time `wails dev` codegen, 10s is tight on a slow dev machine.
- This is not a blocker; if it flakes, the QA agent reruns. But it's an acceptance criterion that's environment-dependent.

**Mitigation:** Raise the timeout to 30s OR explicitly drop the time bound and replace with "process stays running until SIGINT, AND stdout contains both markers."

---

### F10 — NIT: `pnpm-lock.yaml` / `package-lock.json` lifecycle not addressed by D1.1's `.gitignore` updates

**Severity:** NIT. Related to F2.
**Attack target:** D1.1 `.gitignore` edits.

**Evidence:**
- D1.1 line 41-42 adds `ui/build/`, `ui/frontend/node_modules/`, `ui/frontend/dist/`, `ui/frontend/.astro/`. Does NOT mention `pnpm-lock.yaml` or `package-lock.json`.
- Best practice: check in the lockfile (pnpm or npm). Today the lockfile is absent on disk. Either:
  - the bootstrap commits (`a9bac6c` / `8d33539`) never ran `pnpm install`, leaving lockfile uncreated, OR
  - the lockfile was created and then `.gitignore`'d, but no current `.gitignore` rule shows this.
- D1.1's first acceptance bullet "`cd ui/frontend && pnpm install` exits 0 and produces `ui/frontend/node_modules/`" creates a `pnpm-lock.yaml` as a side effect — but the plan doesn't say whether to `git add` it.

**Mitigation:** Add D1.1 acceptance line: "`ui/frontend/pnpm-lock.yaml` (or `package-lock.json` if npm-pivoted per F2) is committed to git after the first `pnpm install`. Reproducible installs require a lockfile."

---

## 2. Counterexamples

### Counterexample CE1 (CONFIRMED — F2)

**Setup:** Dev confirms `wails doctor` clean per Note N7. Dev's machine has `node v22` + `npm v10` per REVISION_BRIEF §1. Dev does NOT have `pnpm` installed.

**Trace:**
1. Phase 4 dispatches D1.1 builder.
2. Builder runs `git mv main.go ui/main.go`, `git mv frontend/ ui/frontend/`, `git mv wails.json ui/wails.json`. Updates `.gitignore`. Commits.
3. Builder runs D1.1 acceptance: `cd ui/frontend && pnpm install`.
4. Shell: `pnpm: command not found`.
5. Builder reports failure. Drop stalls before any code change.

**Why this trace is forced:** REVISION_BRIEF says "npm available." Plan says "pnpm install." There is no precondition probe between them.

### Counterexample CE2 (CONFIRMED — F5)

**Setup:** Dev has `wails` CLI v2.11.4 installed (last stable before 2.12). All other Phase 4 prereqs green.

**Trace:**
1. D1.6 builder lands `mage uiBuild` + `mage uiDev` targets.
2. D1.6 build-QA falsification agent runs `mage uiDev` with 10s timeout.
3. Stdout shows `Wails CLI v2.11.4` + `[Wails] Dev mode`.
4. Acceptance regex/substring `Wails CLI v2.12.0` fails the match.
5. QA marks D1.6 fail. Builder respawns, can't fix (real version is 2.11.4). Drop stalls on a string.

**Why this trace is forced:** Acceptance hardcodes v2.12.0 against an unpinned CLI.

### Counterexample CE3 (POSSIBLE — F3)

**Setup:** D1.5 ships with `client:load`. Next FE drop adds a "ProjectDetail" island.

**Trace:**
1. Next-drop planner reads `ui/frontend/src/pages/index.astro` for pattern.
2. Sees `<ProjectList client:load />` with no comment qualifying the choice.
3. Patterns the new island as `<ProjectDetail client:load />`.
4. ProjectDetail is below-the-fold; `client:visible` would have been correct. Hydration JS ships eagerly for a component the user may never see.

**Why this trace is forced:** D1.5 sets the pattern; Note N3's "future drops use `client:idle`" lives in MD, not in code.

---

## 3. Summary

**Verdict:** `fail` — at least one CONFIRMED counterexample exists that blocks Phase 4 dispatch (F2).

**Total findings:** 10
- **CONFIRMED counterexamples (blocker):** 2 — F2 (pnpm vs npm mismatch), F5 (wails CLI version pin)
- **POSSIBLE counterexamples (dev decision):** 4 — F1 (embed acceptance gap), F3 (`client:load`), F4 (inline DTO), F6 (`wails build` smoke fragility)
- **NITs:** 4 — F7 (cross-drop hygiene), F8 (gofumpt-vs-compile), F9 (10s timeout), F10 (lockfile commit)

**Blocking issues that must be resolved before Phase 4 dispatch:**
1. **F2** — pick one: pin pnpm via `packageManager` field; OR switch to npm; OR add explicit "dev confirms pnpm on `$PATH`" check + commit lockfile.
2. **F5** — soften D1.6 acceptance to regex `Wails CLI v2\.\d+\.\d+` OR pin the CLI version explicitly in Note N7.

**Recommended dev decisions for non-blockers:**
- F3 — switch to `client:idle` OR add justifying comment in `index.astro`.
- F4 — accept inline DTO + plan future split (cheaper) OR move DTO to `ui/types.go` now (cleaner).
- F6 — split D1.3 acceptance into QA-agent-verifiable vs dev-runtime-verifiable.

**No issues found in:**
- Cross-drop file overlap with `drop_4b_test_cleanup/` (F7 — disjoint).
- `mage ci` green claim via build-tag fence (F8 — confirmed by reading `magefile.go:212-229`).
- `magefile.go` CiFe→CiUI rename (D1.2) — no external references to `CiFe` outside the magefile (verified via Read of magefile.go; CI workflow at `.github/workflows/ci.yml:33-35` only invokes `mage ci`, not `mage ci-fe`, so the rename is safe).

**Loop:** Orchestrator routes findings to dev. Plan-revision round 2 spawns a planner with these findings as the brief; planner edits PLAN.md addressing F1/F2/F5 minimally + dev's calls on F3/F4/F6.

---

## Hylla Feedback

N/A — Hylla is OFF per `feedback_hylla_disabled_for_now.md` (2026-05-18), and this drop is FE-only (Hylla is Go-only when on). All evidence gathered via Read / Bash grep (where permitted) / Context7. No Hylla queries attempted.
