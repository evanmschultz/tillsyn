# W6 — PLAN_QA_FALSIFICATION — Round 1

**Wave:** Drop 4c.6.1 — W6 FE_SCAFFOLD
**Plan under attack:** `workflow/drop_4c_6_1/DROP_4c.6.1.W6_FE_SCAFFOLD/PLAN.md` (L2, 9 droplets D1–D9)
**Round:** 1
**Verdict:** **FAIL** — 4 FF (Falsification Findings) + 8 NITs

---

## TL;DR

- **FF1 (HIGH)** — `mage ci` IS NOT auto-excluded from `fe/`. Its `coverage` stage runs `go test ... -cover ./...` from Tillsyn root. Go's `./...` *does* skip nested modules, so `fe/go.mod` does isolate the FE Go code from `mage ci` discovery — BUT this is asserted, not tested. AC10 declares it as an outcome with no validation step. ABSORB: add an explicit "verify `mage ci` returns identical pass-set pre- and post-D1+D2" acceptance bullet to D1 (or D2 since D2 is the one that adds tested Go code under `fe/`).
- **FF2 (HIGH)** — Wails v2 canonical layout per Context7 (`/wailsapp/wails`) places `go.mod` + `main.go` at PROJECT ROOT, not in a subdirectory with `replace ../`. The plan's "separate `fe/go.mod` with `replace github.com/evanmschultz/tillsyn => ../`" is a non-canonical pattern. D1 RiskNote R1 even says "Wails v2 separate fe/go.mod with replace directive to ../. This is the standard Wails v2 pattern" — Context7 doesn't corroborate this as canonical. ABSORB: D1 RiskNotes must require builder to Context7-verify the pattern works with `wails dev` (specifically that the Wails CLI resolves bindings from a nested go.mod), and a fallback path is documented if the pattern breaks.
- **FF3 (HIGH)** — AC2 says "Inter/Iosevka fonts resolve" but `tokens.css` does NOT contain font files. Inter / Iosevka / JetBrains-Mono / Fira-Code ship as separate `@fontsource/*` pnpm packages in `stil/main/package.json:dependencies`. The plan's D3 only copies `tokens.css`; it does NOT add `@fontsource/*` deps to `fe/frontend/package.json`. AC2 will fail. ABSORB: D3 must add `@fontsource/inter`, `@fontsource/iosevka` (and the rest stil uses) as devDependencies in `fe/frontend/package.json`, OR AC2 must be reworded to "stil tokens load; font fallbacks match stil's stack."
- **FF4 (MEDIUM)** — D9 RiskNote on "engine.ts must load baseline.json" is unresolved at plan time. The note documents FOUR competing alternatives ("path traversal" / "Vite static import" / "fetch via Wails IPC `GetBindingsJSON`" / "fetch `/.tillsyn-bindings.json`") and ends with "Final v1 decision: ... engine.ts tries fetch('/.tillsyn/bindings.json'); if 404, uses baseline only." But the **paths list** for D9 specifies copying `stil-baseline.json` to `fe/frontend/public/` — implicitly choosing the static-import OR fetch-public path. The competing alternatives left in RiskNotes will mislead the builder. ABSORB: collapse D9 RiskNotes to ONE chosen path (static import via Astro public dir for baseline.json; HTTP fetch from `/.tillsyn-bindings.json` proxied via Vite/Astro middleware for local bindings) and delete the unchosen alternatives.

NITs cover: (N1) PLAN-QA-DISCIPLINE-R2 numeric audit, (N2) D2 GetBindingsJSON cross-droplet drift, (N3) D8 MainLayout.astro re-edit ordering, (N4) stil-baseline.json copy not in D3 paths, (N5) Vitest --passWithNoTests not committed, (N6) wails-keys.ts Linux platform-awareness, (N7) malformed .tillsyn/bindings.json parse-error handling, (N8) "close" command palette absence assertion.

---

## 1. Methodology

Attack vectors per spawn directive:

1. Missing `blocked_by` between droplets sharing files (beyond D4–D8 wails.ts chain).
2. Cycles.
3. PLAN.md ↔ `_BLOCKERS.toml` drift.
4. YAGNI — could any 2 droplets merge?
5. PLAN-QA-DISCIPLINE-R1: every NEW-behavior acceptance → test-runner blocked_by?
6. PLAN-QA-DISCIPLINE-R2: 9 narrated vs D1–D9 enumerated.
7. Contract mismatches.
8. L1 deviation soundness — D5/D6/D7/D8 serialized via wails.ts.
9. `fe/` separate-module isolation — does `mage ci` auto-exclude?
10. Wails v2 layout — separate `fe/go.mod` + `replace ../` canonical?
11. stil tokens copy-vs-symlink decision documented?
12. Astro+Solid+Vitest IPC roundtrip test exists?
13. Vim engine bindings.json — graceful fallback on MALFORMED file?
14. wails-keys filter platform-aware (Linux vs Mac)?
15. Vim palette command collision with original `close`?
16. Playwright via MCP — at least one droplet has `browser_snapshot` semantic check?

Evidence gathered:

- W6 PLAN.md (1 read, 692 lines).
- W6 `_BLOCKERS.toml` (1 read).
- L1 PLAN.md (W6 section + decomposition shape table).
- REVISION_BRIEF §2.15 + §2.19 (vim bindings architecture).
- SKETCH §5 + §10 (Wails layout + locked decisions).
- W5 sibling PLAN.md (cross-wave merge-semantic shape).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/magefile.go` (CI `coverage()` runs `go test ... -cover ./...`).
- `/Users/evanschultz/Documents/Code/hylla/stil/main/package.json` (font deps).
- `/Users/evanschultz/Documents/Code/hylla/stil/main/src/bindings/baseline.json` (verifies 4 tillsyn commands).
- Context7 `/wailsapp/wails` (project layout canonical pattern: go.mod at root).
- Context7 `/golang/go` (`./...` semantics around nested modules — well-established Go behavior; index returned only partial coverage but the rule is canonical).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/.gitignore` (verifies `dist/` matches at any level).

---

## 2. Findings

### 2.1 FF1 (HIGH) — `mage ci` auto-excludes `fe/` is ASSERTED, not TESTED

**Trace:**

- W6 PLAN.md line 124: `CompletionCriteria: ... mage ci exits 0 (main gate, fe/ excluded)`.
- W6 PLAN.md AC10 (line 29): `mage ci remains green; fe/ is excluded from main mage ci pre-MVP`.
- W6 PLAN.md ContextBlocks line 54: `[constraint severity=critical] fe/ excluded from main mage ci pre-MVP. mage ci-fe is the gate.`
- `magefile.go:267-309` `coverage()` runs `runGoTestCapture("-cover", "./...")` from the magefile working directory (Tillsyn root).
- Go behavior (canonical, not Context7-tested-in-isolation but widely documented): `./...` wildcard expansion does NOT descend into subdirectories that contain their own `go.mod`. Nested modules are opaque to the parent module's wildcard.
- Therefore the PLAN's exclusion claim is structurally correct — IF the `fe/go.mod` is well-formed AND in the tree before `mage ci` runs against the post-D1 state.

**Counterexample window:** D1 lands `fe/go.mod` BEFORE D2 adds `fe/app.go`. If a builder ever lands `fe/app.go` without `fe/go.mod` (e.g., a builder mistakenly drops `fe/go.mod` mid-build), `mage ci` `coverage` will try to compile `fe/app.go` as part of the main module, will fail because `fe/app.go` imports Wails (not in main `go.mod`), and `mage ci` will go red.

**Falsification:** the plan asserts `mage ci` exclusion as an outcome but provides NO explicit step for the builder to verify pre-build vs post-build that `mage ci` returns the identical pass-set. Builders accept the assertion on faith.

**Disposition:** ABSORB into round-2 plan. Add to D1 acceptance:

> `mage ci` from Tillsyn root before D1: capture `mage ci` exit code + duration. After D1 lands `fe/go.mod` + `fe/main.go` + `fe/wails.json`, run `mage ci` again. Diff: same exit code (0), same package set in coverage output, no `./fe/...` package appears. Record both runs in BUILDER_WORKLOG.

Same bullet should also apply at D2 close (Go code added in `fe/`).

---

### 2.2 FF2 (HIGH) — Wails v2 "separate `fe/go.mod` + `replace ../`" is NON-CANONICAL per Context7

**Trace:**

- W6 PLAN.md ContextBlocks line 67: `[decision severity=normal] Wails v2 separate fe/go.mod with replace directive to ../. This is the standard Wails v2 pattern.`
- W6 PLAN.md D1 RiskNotes: `fe/go.mod must declare require github.com/wailsapp/wails/v2 v2.x.x ... The replace github.com/evanmschultz/tillsyn => ../ directive maps the main module.`
- Context7 query `/wailsapp/wails` for "Wails v2 project layout go.mod location relative to project root" returned multiple canonical "Project structure rundown" snippets from `wailsapp/wails/website/.../firstproject.mdx`. All describe the standard layout as: `<project>/go.mod`, `<project>/main.go`, `<project>/frontend/`. The Wails template generator's default puts `go.mod` at the project root, NOT in a subdirectory.
- The `replace` directive Wails docs DO show is for `replace github.com/wailsapp/wails/v2 => /path/to/clonedir/v2` — pinning a local Wails fork. That is NOT the pattern in W6 PLAN.md's `replace github.com/evanmschultz/tillsyn => ../`.
- The "separate fe/go.mod + replace parent" pattern is plausible (Go modules support it), but it's not documented as canonical Wails v2 in the wails docs Context7 indexed.

**Risk:** the Wails CLI runs `wails dev` from inside `fe/`. It expects `wails.json` at the same level as `go.mod`. The plan does put both at `fe/`, so that part works. But `wails dev` triggers `wails generate module` which scans the bound `App` struct via Go's `packages.Load`. With `replace github.com/evanmschultz/tillsyn => ../`, `packages.Load` from inside `fe/` should resolve types in `internal/app` correctly. Should — but only if `fe/go.mod` declares `require github.com/evanmschultz/tillsyn v0.0.0-00000000000000-000000000000` (a placeholder version) alongside the `replace`. The PLAN.md shape_hint does NOT show this `require`. This is a likely build-break.

**Falsification:** D1 RiskNotes confidently asserts the pattern is "standard Wails v2" with no Context7 citation. The pattern is uncommon enough that the dev (or a builder) will hit a friction point during `wails dev`.

**Disposition:** ABSORB.

1. D1 RiskNotes must require the builder to Context7-verify the pattern before authoring `fe/go.mod`. Specifically: does Wails v2's `wailsjs/go/main/App.js` generator handle a `replace ../` parent module correctly?
2. D1 KindPayload shape_hint for `fe/go.mod` must include the `require github.com/evanmschultz/tillsyn v0.0.0-00000000000000-000000000000` alongside the `replace`, OR explicitly document that Go module resolution for a `replace`-only directive (no `require`) works.
3. Provide a fallback path if the pattern breaks: collapse to single root `go.mod` (move `cmd/till` to top-level + Wails main.go at top-level + `frontend/` at top-level). Or move `fe/go.mod` to `tillsyn/fe/go.mod` outside the main worktree (unlikely; out of scope).

---

### 2.3 FF3 (HIGH) — AC2 "Inter/Iosevka fonts resolve" — missing `@fontsource/*` deps

**Trace:**

- W6 PLAN.md AC2 (line 21): `stil design tokens load from fe/frontend/public/stil-tokens.css (copied from stil/main/src/styles/tokens.css); Inter/Iosevka fonts resolve; --carl accent variable is present.`
- W6 PLAN.md D3 paths: `fe/frontend/public/stil-tokens.css (NEW — copied from stil/main/src/styles/tokens.css)`.
- W6 PLAN.md D3 package.json shape_hint: `{name: tillsyn-fe, private: true, scripts: {...}, devDependencies: {astro, @astrojs/solid-js, solid-js, vitest, @playwright/test, typescript}}`. NO `@fontsource/*` entries.
- `stil/main/package.json:dependencies`: `@fontsource/fira-code`, `@fontsource/inter`, `@fontsource/iosevka`, `@fontsource/jetbrains-mono`. Fonts ship as separate npm/pnpm packages, NOT bundled into `tokens.css`.
- `stil/main/src/styles/tokens.css` (read first 40 lines): contains `--carl`, spacing scale, radius scale. No `@font-face` rules.
- `stil/main/src/styles/global.css` (18.9K — not read in detail) likely contains `@import` rules that pull `@fontsource/*` files. The Tillsyn `fe/frontend/` does NOT consume `global.css`.

**Counterexample:** builder follows D3 verbatim. Copies `tokens.css`. Adds `<link rel=stylesheet href=/stil-tokens.css>` to `MainLayout.astro` (D3 stub or D4 fill). Runs `wails dev`. Browser loads page. CSS variables `--carl` resolve. Inter font does NOT load because no `@font-face` rule exists; browser falls back to default sans-serif. AC2 "Inter/Iosevka fonts resolve" FAILS.

**Falsification:** plan promises font resolution but its implementation only copies tokens, not fonts.

**Disposition:** ABSORB. Two options for round-2:

1. **Option A (faithful)**: D3 adds `@fontsource/inter` and `@fontsource/iosevka` (and `@fontsource/jetbrains-mono` if used) to `fe/frontend/package.json:devDependencies`. D3 OR D4 (whoever fills `MainLayout.astro`) adds `@import` statements in a global stylesheet for the fonts.
2. **Option B (deferral)**: rewrite AC2 to "stil tokens load; `--carl` accent variable resolves. Font support deferred to a post-v1 droplet (track as FE-FONTS-R1 in Refinements Logged)."

Plan-QA cannot choose between A and B unilaterally — that's a dev call. Round-2 should surface both options.

---

### 2.4 FF4 (MEDIUM) — D9 RiskNotes loads-baseline.json conflict — 4 alternatives left undecided

**Trace:**

- W6 PLAN.md D9 RiskNotes section (lines 574–576) describes the problem in detail:
  > engine.ts must load stil/main/src/bindings/baseline.json at RUNTIME in the browser. The canonical path in the build system is ../../../../../../stil/main/src/bindings/baseline.json relative to fe/frontend/. This is impractical for a browser build. Resolution for v1: bundle baseline.json via a Vite/Astro static import...
- Then: "**D9 vs D3 `public/` file addition**: D9 needs to add `fe/frontend/public/stil-baseline.json` ... Add `fe/frontend/public/stil-baseline.json` to D9's paths."
- Then for `.tillsyn/bindings.json`: "the path is `../../.tillsyn/bindings.json` relative to `fe/frontend/`. Use a fetch or Vite plugin. ... **Final v1 decision**: fetch `/.tillsyn-bindings.json` via HTTP. ... OR the simplest: `engine.ts` tries `fetch('/.tillsyn/bindings.json')`; if 404, uses baseline only."
- W6 PLAN.md **Updated paths for D9** confirms: `fe/frontend/public/stil-baseline.json (NEW)` is the chosen path.
- But the RiskNotes prose left FOUR competing alternatives in the text: (i) Vite/Astro static import; (ii) HTTP fetch via Astro public dir; (iii) Wails IPC `GetBindingsJSON()` method on `App` (with cross-droplet impact on D2); (iv) `engine.ts tries fetch('/.tillsyn/bindings.json')`.

**Counterexample:** D9 builder reads RiskNotes, sees four paths sketched, picks one. Builder picks (iii) — adds `GetBindingsJSON()` to `fe/app.go`. But D2 is already complete and didn't include that method. Builder either: (a) goes back and adds to D2 (cross-droplet drift — D2's blocked_by chain doesn't account for this); (b) violates the "D9 doesn't touch Go files" constraint by editing `fe/app.go` directly.

**Falsification:** the plan provides four alternatives, declares one "final" near the end, but the prose still confuses. Plans must converge on ONE alternative before the builder is dispatched.

**Disposition:** ABSORB. D9 RiskNotes must collapse to ONE path:

- baseline.json: D3 copies to `fe/frontend/public/stil-baseline.json` (move this from D9's paths to D3's paths to honor the "D3 sets up frontend dev environment" boundary).
- `.tillsyn/bindings.json`: `engine.ts` calls `fetch('/.tillsyn-bindings.json')`. Astro's dev middleware proxies `/.tillsyn-bindings.json` → `${cwd}/.tillsyn/bindings.json`. Production: Wails embeds the file (later concern; v1 is `wails dev` only).
- Delete RiskNote prose about Wails IPC `GetBindingsJSON` (D2 impact) and direct path-traversal.

---

## 3. NITs (Process-rule discipline — all first-class)

### N1 — PLAN-QA-DISCIPLINE-R2 numeric audit: 9 narrated vs D1–D9 enumerated

**Check:** does narrative count match the enumerated D-list?

- KindPayload: D1–D9 (9 entries). ✓
- Droplet Graph section: D1, D2, D3, D4, D5, D6, D7, D8, D9. ✓
- Acceptance bullets: AC1–AC10 (10 ACs, not 9 — but ACs ≠ droplets; this is fine).
- CompletionChecklist: D1 through D9 checkboxes. ✓
- Dispatch Schedule: D1 + D3 batch 1; D2 sequential; D4 + D9 batch 2; D5→D6→D7→D8 serial. Enumerates all 9. ✓
- Objective bullet (line 16): "Deliver 6 v1 surfaces ... Add a TypeScript-side vim keybinding engine ... Add `mage ci-fe` target."

**Numeric trip:** Objective says "6 v1 surfaces" — counted: project list (D4), project detail + action item tree (D5), action item create dialog (D6), dispatcher trigger + spawn output viewer (D7), settings panel (D8). That's 5 droplets covering 5+1=6 SURFACES (D7 packs two surfaces). 5 droplets × surfaces match prose "6 v1 surfaces." OK.

**Verdict: PASS.** No numeric drift.

**Disposition:** No action.

### N2 — D2 GetBindingsJSON cross-droplet drift mention

**Check:** D9 RiskNotes line 576 explicitly says: "expose a `GetBindingsJSON() (string, error)` method on `App` (in `fe/app.go` D2) and call it from `palette.ts` via Wails IPC. ... If D2 is already done when D9 is discovered: add `GetBindingsJSON` in D9 (same package `fe`; but D9 doesn't touch Go files)."

This text leaves the builder unsure whether to edit `fe/app.go` from D9. Per FF4, this whole alternative path should be deleted.

**Disposition:** ABSORB in FF4 resolution.

### N3 — D8 MainLayout.astro re-edit ordering

**Check:** D8 RiskNotes line 517: "settings.astro page must be reachable from MainLayout.astro nav (D4 adds nav; D8 builder may need to add a settings nav link to MainLayout.astro). Since D8 is serialized after D4–D7, editing MainLayout.astro again is safe. Builder adds the nav link."

D4 writes the layout. D8 modifies it. Both touch the same file. The `blocked_by` chain D5→D6→D7→D8 covers this via transitivity (D8 blocked_by D7 blocked_by D6 blocked_by D5 blocked_by D4). So OK.

**BUT** the D8 paths list (line 497–499) does NOT include `MainLayout.astro` as a path. D8's paths only list `settings.astro` + `SettingsPanel.tsx`. The "file footprint declaration" rule (cascade discipline) requires every modified file to appear in paths.

**Disposition:** ABSORB minor. Add `fe/frontend/src/layouts/MainLayout.astro` (MODIFY) to D8 paths.

### N4 — stil-baseline.json copy not in D3 paths

**Check:** D9 RiskNotes resolves to D9 adding `fe/frontend/public/stil-baseline.json`. Per FF4, this should move to D3 (the frontend-setup droplet). D3 paths (lines 264–274) do NOT currently include `stil-baseline.json`. If we keep D9 as the owner, D9 adds the file; if we move ownership to D3, D3 paths must update.

**Disposition:** Tied to FF4 resolution. Move `stil-baseline.json` copy to D3 in round-2.

### N5 — Vitest `--passWithNoTests` flag not committed

**Check:** D3 RiskNote line 294: "pnpm run test:unit on an empty test suite should exit 0 with 'no test files found' message. If Vitest exits non-zero on empty suite, add `--passWithNoTests` flag. Builder discovers and documents."

Vitest 1.x and 2.x DEFAULT to non-zero exit when no tests found. The flag is required, not optional. Leaving it "builder discovers" causes a likely D3 build-QA fail-loud cycle.

**Disposition:** ABSORB. Pre-bake the flag in D3 acceptance: "`fe/frontend/package.json` scripts: `test:unit: vitest run --passWithNoTests`." This is determined, not discovered.

### N6 — wails-keys.ts platform awareness (Linux vs Mac)

**Check:** spawn directive vector 14: "What about platform-specific differences (Linux vs Mac)? Is the filter platform-aware?"

D9 AC7 says: `wails-keys.ts intercepts Cmd+Q, Cmd+M, Cmd+W, Cmd+H at document level...`. On Linux the OS modifier is `Ctrl`, not `Cmd`. On Windows the closest analog is `Alt+F4` for quit, `Win` for system menu. The plan only addresses macOS keys.

Wails v2 supports Linux (WebKitGTK) and Windows (WebView2). Tillsyn dogfoods on macOS first per dev's environment (Darwin per env block), but the migration target `ro-vim` will be cross-platform. The filter being mac-only IS acceptable for v1 (acknowledged scope), but the plan should make this explicit.

**Disposition:** ABSORB minor. D9 AC7 reworded: "wails-keys.ts intercepts macOS Cmd+Q/M/W/H. Linux/Windows OS-key filtering deferred to a future cross-platform droplet (refinement FE-CROSS-PLATFORM-R1)."

### N7 — Malformed `.tillsyn/bindings.json` handling

**Check:** spawn directive vector 13. D9 AC for graceful fallback (line 565): "`palette.ts` ... with `.tillsyn/bindings.json` present, command palette exposes 9 commands; with it absent, falls back to 4 baseline commands."

What about: file PRESENT but malformed (truncated JSON, schema-version mismatch, bad UTF-8)? The plan doesn't say. Common patterns:

- Fail-loud (throw, refuse to load app): not user-friendly for a runtime config.
- Log + fall back to baseline-only: matches the "graceful fallback" spirit.
- Fail-loud but clear error message in command palette: middle path.

**Disposition:** ABSORB minor. Add D9 AC: "with malformed `.tillsyn/bindings.json`: log parse error to console + fall back to baseline-only (same as absent file). NOT fail-loud."

### N8 — Vim palette command `close` absence assertion

**Check:** spawn directive vector 15: "Vim palette command collision with `close`: SKETCH says original `close` DROPPED (redundant with stil's `complete-drop`). Does D9's `palette.ts` actually drop it?"

The dropped command was in the OLD 6-command proposal that R3-FF2 superseded. The current 9-command set (4 baseline + 5 local) does NOT contain `close`. Verified against:

- `stil/main/src/bindings/baseline.json:product_extensions.tillsyn.commands` — 4 commands: `new-drop`, `complete-drop`, `handoff`, `comment`. No `close`.
- REVISION_BRIEF §2.19 Tillsyn-local additions — 5 commands: `dispatch`, `plan`, `archive`, `settings`, `help`. No `close`.

The plan is fine. There is no current `close` command to drop. But the plan does NOT have an explicit acceptance asserting `close` is absent — which makes a regression possible if a future R-loop reintroduces it accidentally.

**Disposition:** DEFERRED-AS-NIT — reason: belt-and-suspenders assertion for a non-existent command is low value; the REVISION_BRIEF history is sufficient guard.

---

## 4. Spawn directive vector-by-vector audit

| Vector | Question | Verdict | Finding |
|---|---|---|---|
| 1 | Hidden file-shares beyond wails.ts? | PASS | D4–D8 wails.ts chain identified; MainLayout.astro D4-creates / D8-modifies handled via D5→D6→D7→D8 transitivity. (N3 minor: D8 paths missing MainLayout.astro.) |
| 2 | Cycles? | PASS | Acyclicity: D1→D2; D3 independent; D2+D3→D4; D4→D5→D6→D7→D8; D3→D9. No cycle. |
| 3 | PLAN.md ↔ `_BLOCKERS.toml` drift? | PASS | `_BLOCKERS.toml` entries for D2/D4/D5/D6/D7/D8/D9 all match PLAN.md inline `Blocked by:` bullets. |
| 4 | YAGNI — could 2 droplets merge? | PASS | D5+D6 are both forms in same project-detail context but they target different files (ActionItemTree vs ActionItemCreateDialog) and wails.ts is serialized; merging risks cross-concern. D7's two components (DispatcherTrigger + SpawnOutputViewer) are already merged in one droplet. |
| 5 | NEW-behavior acceptance → test-runner blocked_by? | PASS | AC3/AC4 (IPC calls), AC6 (vim engine merge), AC7 (wails-keys filter), AC8 (Vitest) all map to `mage ci-fe` as the test-runner gate. D9 has Vitest tests for all three concerns. |
| 6 | PLAN-QA-DISCIPLINE-R2 numeric audit | PASS (N1) | See N1: 9 narrated ↔ D1–D9 enumerated, no drift. |
| 7 | Contract mismatches? | FAIL (FF1, FF2, FF3, FF4) | Multiple — see findings above. |
| 8 | L1 deviation: D5–D8 wails.ts serialization soundness | PASS | The deviation is well-justified (file-level conflict on wails.ts). FE-WAILS-TS-SERIAL-R1 refinement tracks post-v1 split. Critical path: D1→D2→D4→D5→D6→D7→D8 = 7-step longest chain, acceptable for ~9 droplets. |
| 9 | `fe/` separate-module isolation from `mage ci` | FAIL (FF1) | Structurally correct but unverified. Add validation step. |
| 10 | Wails v2 separate `fe/go.mod` + `replace ../` canonical? | FAIL (FF2) | Context7 doesn't corroborate; plan asserts as canonical. |
| 11 | stil tokens copy-vs-symlink trade-off documented? | PASS | PLAN.md explicitly chose "copy" via `Read` + `Write` (D3 RiskNote line 293) and noted that source-path consumption avoids requiring a stil build pre-step. The trade-off (copy goes stale vs symlink leaks dependency) is documented in SKETCH §10 row "Stil tokens consumption path." |
| 12 | Astro+Solid+Vitest IPC roundtrip test? | PARTIAL | D4 AC line 336 + RiskNote line 341 require mocking `window.go.main.App.*` in Vitest. This tests the component-side wiring, NOT the Wails IPC roundtrip. A genuine end-to-end IPC test requires running `wails dev` + Playwright, which IS in D9's Playwright test scope. Acceptable for v1 since Playwright is run via MCP by QA agents. |
| 13 | Malformed bindings.json fallback? | FAIL (N7) | See N7. |
| 14 | wails-keys platform-aware? | FAIL (N6) | See N6. |
| 15 | `close` command palette absence assertion? | PASS (N8 deferred) | See N8. |
| 16 | Playwright via MCP `browser_snapshot` semantic check? | PASS | D4 AC (line 338), D5 AC (line 382), D6 AC (line 430), D8 AC (line 513) all use `browser_snapshot`. D9 Playwright test (line 570) uses `browser_navigate` + `browser_snapshot`. ✓ |

---

## 5. Cross-wave attack — W5 vs W6 vim engine consistency

W5 (TUI keybindings/) and W6 (FE vim engine) both implement ID-based deep merge of stil baseline 4 commands + Tillsyn-local 5 commands. The implementations are independent (Go vs TypeScript) but the SEMANTICS must match for cross-surface consistency (REVISION_BRIEF KEYBIND-CONSIST-R1).

Test surface for cross-wave consistency: does `:dispatch <action-item-id>` produce IDENTICAL dispatcher invocations from both surfaces? REVISION_BRIEF §2.19 line 441 acknowledges this and assigns it as a TEST. But neither W5 nor W6 PLAN.md carries a droplet for this cross-surface consistency test. It's deferred — fine for now (the test would require both W5 and W6 to be complete first), but no acceptance criterion captures the gap.

**Disposition:** DEFERRED-AS-NIT — reason: cross-wave integration test belongs in W2 (init smoke) OR Drop 4c.7 (cascade wiring). Not W5 or W6's responsibility individually.

---

## 6. Verdict

**FAIL** — 4 FFs (FF1 HIGH, FF2 HIGH, FF3 HIGH, FF4 MEDIUM) + 8 NITs (N1 PASS-with-audit, N2 absorbed in FF4, N3 ABSORB minor, N4 absorbed in FF4, N5 ABSORB, N6 ABSORB, N7 ABSORB, N8 DEFERRED).

**Recommended round-2 absorption checklist:**

1. FF1 — D1 + D2 acceptance: explicit `mage ci` pre/post run with diff confirming identical pass-set.
2. FF2 — D1 RiskNotes: Context7-verify `replace ../` Wails pattern; document `require <main_module> v0.0.0-...` shape; provide fallback.
3. FF3 — D3 OR AC2 — add `@fontsource/*` deps OR reword AC2 to "tokens load; fonts deferred to FE-FONTS-R1."
4. FF4 — D9 RiskNotes — collapse to ONE bindings-load path; move baseline.json copy to D3.
5. N3 — D8 paths — add MainLayout.astro.
6. N5 — D3 — `vitest run --passWithNoTests` pre-baked, not discovered.
7. N6 — D9 AC7 — qualify macOS-only with FE-CROSS-PLATFORM-R1 refinement.
8. N7 — D9 AC — malformed bindings.json = log + fall back, NOT fail-loud.

**Pattern observation for PLAN-QA-DISCIPLINE-R3 (proposed):** when a plan's RiskNotes documents MULTIPLE competing alternatives without explicitly closing on ONE, the plan is not yet builder-ready. Future plan-QA falsification rounds should attack any RiskNote that enumerates ≥2 unchoosen paths. (FF4 is the example.)

---

## Hylla Feedback

Hylla OFF per spawn directive. No Hylla calls attempted.
