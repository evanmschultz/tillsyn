# PLAN_QA_PROOF — DROP_4c.6.1.W6_FE_SCAFFOLD (Round 2)

**Reviewer:** L2 plan-QA proof agent (FE)
**Subject:** `workflow/drop_4c_6_1/DROP_4c.6.1.W6_FE_SCAFFOLD/PLAN.md` (9-droplet L2 plan for Wave W6, rewritten under R10-D3 Wails-at-root)
**Round:** 2
**Mode:** Filesystem-MD-only (no Tillsyn runtime); Hylla OFF; Context7 ON
**Verdict:** **PASS** — Round-1 FF1 (proof) and Fals FF1/FF2/FF3/FF4 absorbed; Round-1 NITs absorbed; 2 new minor NITs surfaced

---

## TL;DR

Round-2 absorbs all blocking findings from Round 1:

- **Proof FF1 (R1)** — `KindPayload.children.blocked_by` contradicted `_BLOCKERS.toml` for D5–D8. **RESOLVED** in PLAN.md `KindPayload` (lines 169–181) and `_BLOCKERS.toml` (lines 6–39). Serial chain D4→D5→D6→D7→D8 is now consistent across `KindPayload.children`, per-droplet "Blocked by" prose, Droplet Graph, Dispatch order, Dispatch Schedule, AND `_BLOCKERS.toml`.
- **Fals FF1 (R1, HIGH)** — `mage ci` exclusion was asserted not tested. **RESOLVED** via R10-D3: `//go:build wails` build-tag isolation replaces fragile separate-module pattern. D1 AC (line 273) explicitly requires builder to record pre-D1 and post-D1 `mage ci` output in BUILDER_WORKLOG to confirm IDENTICAL coverage pass-set. D2 AC (line 317) re-verifies. Builder verification path concrete and testable.
- **Fals FF2 (R1, HIGH)** — non-canonical Wails layout (`fe/go.mod` + `replace ../`). **RESOLVED** via R10-D3 Wails-at-repo-root. Context7 `/wailsapp/wails` confirms canonical layout: `go.mod` + `main.go` + `wails.json` + `frontend/` at project root. All `fe/` paths migrated; single shared `go.mod`; no replace directive; no placeholder require. `//go:build wails` is the sole CI-isolation mechanism.
- **Fals FF3 (R1, HIGH)** — `tokens.css` ships no font binaries. **RESOLVED.** D3 `package.json` shape_hint (line 392) declares `@fontsource/inter`, `@fontsource/iosevka`, `@fontsource/fira-code`, `@fontsource/jetbrains-mono`. D4 `MainLayout.astro` (line 429, 448) imports them as side-effect imports. Matches `stil/main/package.json:dependencies` (verified — 4 `@fontsource/*` entries, no font binaries in tokens.css).
- **Fals FF4 (R1, MEDIUM)** — D9 RiskNotes left 4 competing alternatives. **RESOLVED.** PLAN.md collapses to ONE path each (lines 680–681). Baseline: `fetch('/stil-baseline.json')` from `frontend/public/` (file moved to D3 paths, line 356). Local: `fetch('/.tillsyn-bindings.json')` via Astro vite proxy (D9 amends `astro.config.mjs`, line 687). Wails-IPC `GetBindingsJSON` alternative explicitly rejected (lines 41, 323).
- **R1 Proof NITs** — all absorbed (NIT2 `solidJs` import name; NIT3 D3 `env.d.ts` explicit KindPayload entry; NIT4 D9 marker phrasing tightened to "`//`-style line comment at top of file"; NIT5 D9 HTTP-fetch mechanism explicit; NIT6 D2 `RunDispatcher` stub-path explicit in AC; NIT7 migration-markers Vitest CI gate added as `tests/migration-markers.test.ts` in D3).
- **R1 Fals NITs** — N3 (D8 MainLayout.astro added to paths, line 600), N4 (baseline.json moved from D9 to D3, line 356), N5 (`--passWithNoTests` baked into D3 `test:unit` script, line 369), N6 (D9 AC7 macOS-only with FE-CROSS-PLATFORM-R1 refinement, line 65), N7 (malformed bindings.json → log + fallback, line 675), N8 (`close` absence assertion DEFERRED-AS-NIT — reasonable; REVISION_BRIEF §2.19 is the authoritative guard).

Two new minor NITs surfaced in this rewrite (R2-NIT1: vite proxy rewrite mechanism leaves filesystem mapping ambiguous; R2-NIT2: D1 `wails dev` smoke verification timing not constrained to pre-frontend-scaffold state).

**Counts verified internally consistent at 9 droplets:** title header (line 1), KindPayload (lines 169–181 → 9 entries), per-droplet sections (D1–D9, sections at 249 / 296 / 343 / 406 / 455 / 501 / 544 / 590 / 638), CompletionChecklist (lines 202–211 → 9 boxes), Dispatch Schedule (lines 764–782 → 9 droplets enumerated), `_BLOCKERS.toml` (7 entries for the 7 non-empty-blocked_by droplets — D1 + D3 are roots without blockers).

---

## 1. Round-2 Absorption Verification

### 1.1 R10-D3 Wails-at-tillsyn-root absorption

**Premise:** Every `fe/` path must migrate to repo root; no `fe/go.mod`, no `replace` directive; `//go:build wails` on root `main.go` + `app.go`.

**Evidence:**

- `git grep "fe/" workflow/drop_4c_6_1/DROP_4c.6.1.W6_FE_SCAFFOLD/PLAN.md` returns nothing (clean migration).
- `git grep "fe/" workflow/drop_4c_6_1/DROP_4c.6.1.W6_FE_SCAFFOLD/_BLOCKERS.toml` returns nothing.
- D1 Paths (lines 257–259): `main.go` (NEW at root), `wails.json` (NEW at root), `.gitignore` (EXISTING — add wails ignore patterns). NO `fe/go.mod`, NO replace directive.
- D2 Paths (line 304): `app.go` (NEW at root).
- D1 KindPayload shape_hint (line 287): import path `github.com/evanmschultz/tillsyn/internal/app` (single module — no replace needed).
- D1 AC (line 268): "`main.go` exists at tillsyn repo root with `//go:build wails` as the FIRST LINE."
- D2 AC (line 313): "`app.go` exists at tillsyn repo root with `//go:build wails` as the FIRST LINE."
- D1 RiskNotes (line 277): "`//go:build wails` MUST be the very first line of `main.go` ... followed by a blank line, then `package main`. If the blank line is missing, Go treats the constraint as a comment and the file compiles unconditionally."
- ContextBlocks (lines 99–101): "`//go:build wails` MUST be the first line of main.go and app.go (before the package declaration)."
- ContextBlocks (line 153–156, severity=high warning): explicit warning that constraint must be followed by a blank line before `package`.
- ContextBlocks (lines 120–121): "Canonical Wails v2 layout at tillsyn repo root. Single shared go.mod. No fe/ subfolder. No separate module. No replace directive. Verified via Context7 /wailsapp/wails docs."

**Conclusion:** R10-D3 absorbed cleanly. **PASS.**

### 1.2 Proof FF1 (R1) preservation — KindPayload ↔ _BLOCKERS ↔ per-droplet consistency

**Premise:** The Round-1 finding — `KindPayload.children.blocked_by` for D5/D6/D7/D8 contradicted `_BLOCKERS.toml` and per-droplet "Blocked by" prose — must remain resolved in R2.

**Evidence:**

| Droplet | `KindPayload.children` (PLAN.md lines 169–181) | Per-droplet header "Blocked by" | `_BLOCKERS.toml` |
|---|---|---|---|
| D1 | `[]` (line 171) | — (line 255 — Wave A head) | not listed (root) |
| D2 | `["W6.D1"]` (line 172) | `W6.D1` (line 302) | `["W6.D1"]` (line 8) |
| D3 | `[]` (line 173) | — (line 349 — Wave A parallel) | not listed (root) |
| D4 | `["W6.D2", "W6.D3"]` (line 174) | `W6.D2, W6.D3` (line 412) | `["W6.D2", "W6.D3"]` (line 13) |
| D5 | `["W6.D2", "W6.D3", "W6.D4"]` (line 175) | `W6.D2, W6.D3, W6.D4` (line 461) | `["W6.D2", "W6.D3", "W6.D4"]` (line 18) |
| D6 | `["W6.D5"]` (line 176) | `W6.D5` (line 507) | `["W6.D5"]` (line 23) |
| D7 | `["W6.D6"]` (line 177) | `W6.D6` (line 550) | `["W6.D6"]` (line 28) |
| D8 | `["W6.D7"]` (line 178) | `W6.D7` (line 596) | `["W6.D7"]` (line 33) |
| D9 | `["W6.D3"]` (line 179) | `W6.D3` (line 644) | `["W6.D3"]` (line 38) |

All three surfaces agree, row by row.

Droplet Graph (lines 222–235) and Dispatch order (lines 237–245) and Dispatch Schedule (lines 764–782) all narrate the same serial D4→D5→D6→D7→D8 chain.

**Conclusion:** R1 Proof FF1 preserved in R2. **PASS.**

### 1.3 Fals FF1 (R1) — `mage ci` exclusion verification

**Premise:** "`mage ci` exclusion of root `main.go` + `app.go` via `//go:build wails`" must have a concrete verification step, not a bare assertion.

**Evidence:**

- D1 AC (line 273): "`mage ci` remains green (without `-tags wails`; build tag excludes main.go + app.go from default compile; `go test ./...` shows IDENTICAL package set as pre-D1). Builder records pre-D1 and post-D1 `mage ci` output in BUILDER_WORKLOG to confirm identical coverage pass-set."
- D2 AC (line 317): "`mage ci` remains green (build tag unchanged; same package set as post-D1)."
- ValidationPlan AC10 (line 79): "builder runs `go test ./...` from tillsyn root WITHOUT `-tags wails`; records exit code + package set in BUILDER_WORKLOG; confirms no `//go:build wails`-tagged files appear in coverage output."
- ContextBlocks (lines 99–101): "`//go:build wails` MUST be the first line of main.go and app.go (before the package declaration). Omitting the build tag breaks the CI isolation guarantee — mage ci will try to compile Wails deps."

The R1 falsification's exact requested fix — "verify `mage ci` returns identical pass-set pre- and post-D1+D2" — is encoded verbatim in D1 AC and re-verified in D2 AC.

**Go build-constraint semantics** (canonical, well-documented): a `//go:build` directive on a line above (blank line then) `package` clause excludes the file from the default build set. `go test ./...` from the module root will silently skip files whose build constraints aren't satisfied. With `mage ci` running `go test ./...` without `-tags wails`, root `main.go` and `app.go` will not be compiled or counted in coverage.

**Conclusion:** R1 Fals FF1 RESOLVED. **PASS.**

### 1.4 Fals FF2 (R1) — Wails canonical layout

**Premise:** R10-D3 chose Wails canonical layout (Option (c) per L1 PLAN.md line 18). Plan must reflect this with single `go.mod`, no `replace`, no `fe/` subfolder.

**Evidence:**

- L1 PLAN.md line 18 R10-D3 decision: "Option (c) Wails at tillsyn repo root — canonical Wails v2 layout. `main.go` + `app.go` + `wails.json` + `frontend/` all at the tillsyn repo root, sharing the existing single `go.mod`. NO separate `fe/go.mod`, NO `replace` directive, NO `fe/` subfolder."
- L1 PLAN.md line 924 (locked decision): "Wails at tillsyn repo root — canonical Wails v2 layout."
- Context7 `/wailsapp/wails` "Creating a Project > Project Layout > Project structure rundown": canonical layout is `go.mod` + `main.go` + `wails.json` + `frontend/` + `build/` at project root. Confirms R10-D3 choice.
- W6 PLAN.md ContextBlocks (lines 120–121): "Canonical Wails v2 layout at tillsyn repo root. Single shared go.mod. No fe/ subfolder. No separate module. No replace directive. Verified via Context7 /wailsapp/wails docs."
- W6 PLAN.md D1 RiskNotes (line 281): "Context7 `/wailsapp/wails` canonical layout confirmed: `go.mod` at project root, `main.go` at root, `wails.json` at root, `frontend/` at root. This drop follows that layout exactly."
- W6 PLAN.md Round 2 Changes block (lines 19–27): every `fe/` path explicitly migrated to root; `D1 RiskNote R2 (replace directive + placeholder require) removed entirely — no separate module.`

**Conclusion:** R1 Fals FF2 RESOLVED. **PASS.**

### 1.5 Fals FF3 (R1) — fonts via `@fontsource/*`

**Premise:** `stil/main/src/styles/tokens.css` declares font families but ships no binaries. Plan must include `@fontsource/*` pnpm deps + side-effect imports.

**Evidence:**

- Direct read of `stil/main/src/styles/tokens.css` (lines 1–96): no `@font-face` rules, no `@fontsource` imports. Only `--font-sans`, `--font-serif`, `--font-mono` CSS variables declaring font-family stacks.
- Direct read of `stil/main/package.json`: `dependencies` includes `@fontsource/fira-code` (^5.2.7), `@fontsource/inter` (^5.2.8), `@fontsource/iosevka` (^5.2.5), `@fontsource/jetbrains-mono` (^5.2.8). Confirmed 4 packages.
- W6 PLAN.md D3 KindPayload shape_hint (line 392): `"...devDependencies: {astro, '@astrojs/solid-js', 'solid-js', vitest, '@playwright/test', typescript, '@fontsource/inter', '@fontsource/iosevka', '@fontsource/fira-code', '@fontsource/jetbrains-mono'}"`. All four declared.
- W6 PLAN.md D3 AC (line 369): "`devDependencies` ... including `@fontsource/inter`, `@fontsource/iosevka`, `@fontsource/fira-code`, `@fontsource/jetbrains-mono`."
- W6 PLAN.md D4 AC (line 429): "`MainLayout.astro` imports all four `@fontsource/*` packages (e.g. `import '@fontsource/inter'; import '@fontsource/iosevka'; import '@fontsource/fira-code'; import '@fontsource/jetbrains-mono';`) in the `<style>` block or as a side-effect import in the frontmatter."
- W6 PLAN.md D4 KindPayload (line 448): `MainLayout.astro modify shape_hint` includes `@fontsource/* imports`.
- W6 PLAN.md ContextBlocks (lines 115–118): "Fonts ship as @fontsource/* pnpm packages — they are NOT bundled in tokens.css. D3 frontend/package.json MUST include @fontsource/inter, @fontsource/iosevka, @fontsource/fira-code, @fontsource/jetbrains-mono. D4 MainLayout.astro MUST import them."
- AC2 (line 60): "Inter/Iosevka/Fira-Code/JetBrains-Mono fonts resolve via `@fontsource/*` pnpm packages declared in `frontend/package.json` and imported in `frontend/src/layouts/MainLayout.astro`."

**Conclusion:** R1 Fals FF3 RESOLVED. **PASS.**

### 1.6 Fals FF4 (R1) — D9 bindings load — ONE path

**Premise:** R1 Fals FF4 required collapsing D9 RiskNotes to ONE chosen path for baseline.json AND `.tillsyn/bindings.json`. Alternative paths must be explicitly refuted, not left implicit.

**Evidence:**

- W6 PLAN.md D9 RiskNotes "Baseline loading (ONE chosen path)" (line 680): "`engine.ts` fetches baseline at startup: `await fetch('/stil-baseline.json')` (file is in `frontend/public/`, copied by D3 — D3 must complete before D9 runs, ensured by D9 `blocked_by D3`). ... Alternative paths (direct path traversal, Wails IPC `GetBindingsJSON`) are explicitly rejected: path traversal is impractical in browser context; IPC `GetBindingsJSON` adds cross-droplet scope to D2 unnecessarily. Use `fetch('/stil-baseline.json')` only."
- W6 PLAN.md D9 RiskNotes "Local bindings loading (ONE chosen path)" (line 681): "`palette.ts` fetches local extension: `await fetch('/.tillsyn-bindings.json')`. If 404: graceful fallback to baseline-only (4 commands). If present + valid: ID-merge with baseline. If present + malformed: log parse error to console + fall back to baseline-only (NOT fail-loud). The Astro vite dev-server proxy is configured in `frontend/astro.config.mjs` to proxy `/.tillsyn-bindings.json` → `../../.tillsyn/bindings.json` ..."
- W6 PLAN.md D2 RiskNotes (line 323): "`GetBindingsJSON` IPC method is explicitly NOT in scope for D2. The vim engine (D9) loads bindings via HTTP fetch from `/.tillsyn-bindings.json`. Do not add `GetBindingsJSON` to `app.go`."
- W6 PLAN.md Round 2 Changes Fals FF4 row (line 41): "Three alternate paths (path-traversal, Wails IPC `GetBindingsJSON`, direct public copy without proxy) explicitly refuted. `GetBindingsJSON` IPC alternative deleted from D2 scope."
- W6 PLAN.md N4 absorption (line 43): "Fals N4 ABSORBED: `frontend/public/stil-baseline.json` copy moved from D9 to D3 paths + D3 KindPayload." Verified at D3 Paths line 356 + D3 KindPayload line 395.
- D9 paths block (lines 645–653, 687–695): `frontend/public/stil-baseline.json` is NOT in D9 paths (it's in D3). `frontend/astro.config.mjs` is in D9 paths as MODIFY (line 687).

**Conclusion:** R1 Fals FF4 RESOLVED — ONE path each, alternatives explicitly refuted. **PASS.**

### 1.7 R1 Proof NITs (all 7)

| R1 NIT | Resolution | Evidence |
|---|---|---|
| NIT1 (spawn-prompt UI-surface count 5/6) | Orchestrator-side; no PLAN.md edit needed. ABSORBED (line 30). | PLAN.md line 16 Objective correctly says "6 v1 surfaces". |
| NIT2 (`solid` → `solidJs` import name) | ABSORBED. | PLAN.md D3 AC line 370 + KindPayload line 393: "`import solidJs from '@astrojs/solid-js'`"; line 32. |
| NIT3 (D3 `env.d.ts` not in KindPayload) | ABSORBED. | PLAN.md D3 paths line 358 + KindPayload line 397 explicit entry. |
| NIT4 (D9 marker phrasing) | ABSORBED — tightened to `//`-style line comment at top of file. | PLAN.md AC5 line 63: "as a `//`-style line comment at the top of the file before any imports"; D9 AC line 667. |
| NIT5 (D9 HTTP-fetch mechanism + D3 public-dir copy) | ABSORBED. Baseline.json moved from D9 to D3 paths; AC explicit. | PLAN.md D3 paths line 356; D9 RiskNotes lines 680–681. |
| NIT6 (D2 `RunDispatcher` stub-path explicit in AC) | ABSORBED. | PLAN.md D2 AC line 315: "delegates to `Service.RunDispatcher` OR returns wrapped `ErrNotImplemented` — both are acceptable v1 wiring; document which in BUILDER_WORKLOG"; line 35. |
| NIT7 (migration-markers Vitest CI gate) | ABSORBED. D3 adds `tests/migration-markers.test.ts`. | PLAN.md D3 paths line 359; D3 AC line 375; D3 KindPayload line 398; line 36. |

**Conclusion:** All 7 R1 Proof NITs absorbed. **PASS.**

### 1.8 R1 Falsification NITs (N3–N8)

| R1 Fals NIT | Resolution | Evidence |
|---|---|---|
| N3 (D8 paths missing MainLayout.astro) | ABSORBED. | PLAN.md D8 paths line 600: `frontend/src/layouts/MainLayout.astro` (EXISTING — D8 adds settings nav link); line 42. |
| N4 (stil-baseline.json copy not in D3 paths) | ABSORBED — moved from D9 to D3. | PLAN.md D3 paths line 356; line 43. |
| N5 (Vitest `--passWithNoTests` baked) | ABSORBED — pre-baked in `test:unit` script. | PLAN.md D3 AC line 369: "`test:unit: \"vitest run --passWithNoTests\"`"; line 44. |
| N6 (wails-keys macOS-only + FE-CROSS-PLATFORM-R1) | ABSORBED. | PLAN.md AC7 line 65; D9 AC line 670; refinement table line 800. |
| N7 (malformed bindings.json → log + fallback) | ABSORBED. | PLAN.md D9 AC line 675: "with malformed local bindings JSON ... `getCommands()` returns 4 baseline commands; parse error is logged to console (not thrown/fail-loud)"; D9 RiskNotes line 681; line 46. |
| N8 (`close` command absence assertion) | DEFERRED-AS-NIT with reason. | PLAN.md line 47: "vacuous assertion adds no signal. The REVISION_BRIEF §2.19 history is the authoritative guard." Reasonable deferral. |

**Conclusion:** All 6 R1 Fals NITs either absorbed (N3–N7) or deferred-with-reason (N8). **PASS.**

---

## 2. New Round-2 Proof Checks

### 2.1 Serial chain D4→D5→D6→D7→D8 on `frontend/src/lib/wails.ts`

**Premise:** Five droplets edit the same `frontend/src/lib/wails.ts` file; must be serialized via `blocked_by`.

**Evidence:**

- D4 Paths line 416: `frontend/src/lib/wails.ts (NEW — Wails IPC type wrappers; shared by D5–D8 via serial chain)`.
- D5 also modifies line 466: `frontend/src/lib/wails.ts (EXISTING after D4; D5 adds listActionItems wrapper)`. blocked_by `W6.D4` (line 461).
- D6 also modifies line 511: `frontend/src/lib/wails.ts (EXISTING after D5; D6 adds createActionItem wrapper)`. blocked_by `W6.D5` (line 507).
- D7 also modifies line 555: `frontend/src/lib/wails.ts (EXISTING after D6; D7 adds runDispatcher wrapper)`. blocked_by `W6.D6` (line 550).
- D8 also modifies line 602: `frontend/src/lib/wails.ts (EXISTING after D7; D8 adds getAgentsConfig + getTemplateConfig wrappers)`. blocked_by `W6.D7` (line 596).
- `_BLOCKERS.toml` mirrors each (lines 18, 23, 28, 33).

**Conclusion:** Serial chain preserved correctly. **PASS.**

### 2.2 `_BLOCKERS.toml` mirrors PLAN.md

**Premise:** `_BLOCKERS.toml` is mirror-of-truth; PLAN.md is authoritative.

**Evidence:** Cross-checked in §1.2 table. Every PLAN.md `Blocked by:` per-droplet header agrees with `_BLOCKERS.toml`. Both agree with `KindPayload.children.blocked_by`. **PASS.**

### 2.3 PLAN-QA-DISCIPLINE-R1 (every NEW-behavior AC → test-runner blocked_by)

**Premise:** Every "Vitest test" or "Playwright test" AC must trace to a `blocked_by` chain reaching D3 (the droplet that adds `mage ci-fe` and the test runners).

**Evidence:**

- D4 AC (line 431): "Vitest: at least one unit test for `ProjectList.tsx`". D4 `blocked_by W6.D2, W6.D3` (line 412). Reaches D3 directly. ✓
- D5 AC (line 479): "Vitest: at least one unit test for tree flattening / rendering logic". D5 `blocked_by W6.D2, W6.D3, W6.D4`. Reaches D3 directly. ✓
- D6 AC (line 524): "Vitest: test for form validation". D6 `blocked_by W6.D5`. D5 reaches D3 transitively. ✓
- D7 AC (line 569): "Vitest: test for button click firing IPC call". D7 → D6 → D5 → D3 transitively. ✓
- D8 AC (line 615): "Vitest: test for settings panel rendering". D8 → D7 → D6 → D5 → D3 transitively. ✓
- D9 ACs (lines 672–675): Vitest tests for engine/wails-keys/palette. D9 `blocked_by W6.D3` (line 644). Direct. ✓
- D9 Playwright AC (line 676): runs against `http://localhost:4321` (Astro dev server from D3). D9 → D3. ✓
- D3 migration-markers Vitest test (line 375) — D3 is the source droplet, no blocker needed.

**Conclusion:** No R1 violation. Every test AC reaches D3. **PASS.**

### 2.4 PLAN-QA-DISCIPLINE-R2 (narrated count vs enumerated count)

**Premise:** Every numeric count claim in the plan must match D-list enumeration.

**Evidence:**

- Title (line 1): no numeric claim ("Wails v2 ... Astro + Solid + stil tokens + vim engine") — no count to falsify.
- Objective (line 55): "Deliver 6 v1 surfaces (project list, project detail + action item tree, action item create dialog, dispatcher trigger + spawn output viewer, settings panel)". Enumerated: 1 list + 1 detail-w-tree + 1 dialog + 1 trigger + 1 viewer + 1 settings = 6. ✓ Maps to 5 droplets (D4–D8); D7 carries 2 surfaces (DispatcherTrigger + SpawnOutputViewer). Trace coverage complete.
- Objective (line 55): "(`engine.ts` + `types.ts` + `wails-keys.ts` + `palette.ts`)" — 4 vim engine files. D9 paths (lines 646–649) enumerate exactly 4 source files. ✓
- Objective (line 55): "stil baseline commands (4) + Tillsyn-local extensions (5) = 9 commands". `stil/main/src/bindings/baseline.json` direct-read confirms 4 in `product_extensions.tillsyn`. REVISION_BRIEF §2.19 names 5 local extensions. 4 + 5 = 9. ✓
- D9 Evidence block (lines 662–665): "4 commands ... 5 commands ... Total with merge: 9. Without local file: 4." Numbers match.
- `KindPayload.children` (lines 169–181): 9 entries (D1–D9). CompletionChecklist (lines 202–211): 9 boxes. Per-droplet sections: D1, D2, D3, D4, D5, D6, D7, D8, D9 — 9 sections. Dispatch Schedule (lines 764–782): 9 droplets enumerated.

**Conclusion:** All numeric claims consistent. **PASS.**

### 2.5 Migration markers on every FE component file + vim engine file

**Premise:** Every `frontend/src/components/*.tsx` file carries `// MIGRATION TARGET: @hylla/stil-solid` as first line; every `frontend/src/lib/vim/*.ts` file carries `// MIGRATION TARGET: github.com/hylla-org/ro-vim` as first line. Migration-markers Vitest test enforces this as CI gate.

**Evidence:**

- AC5 (line 63): per-file migration marker requirement; Vitest test in `frontend/tests/migration-markers.test.ts` walks both directories.
- D3 paths line 359: `frontend/tests/migration-markers.test.ts` (NEW — Vitest test; walks src/components/ + src/lib/vim/ and asserts MIGRATION TARGET marker substring).
- D3 AC line 375: explicit description of the test logic.
- D3 KindPayload line 398: shape_hint for the test.
- Per-droplet AC bullets requiring marker:
  - D4 line 426: `ProjectList.tsx` first line marker.
  - D4 line 427: `wails.ts` first line marker (`@hylla/stil-solid`).
  - D5 line 475: `ActionItemTree.tsx` first line marker.
  - D6 line 520: `ActionItemCreateDialog.tsx` first line marker.
  - D7 line 564: `DispatcherTrigger.tsx` first line marker.
  - D7 line 565: `SpawnOutputViewer.tsx` first line marker.
  - D8 line 611: `SettingsPanel.tsx` first line marker.
  - D9 line 667: all 4 vim engine files have `// MIGRATION TARGET: github.com/hylla-org/ro-vim` as a `//`-style line comment at the top of the file before any imports.
- ContextBlocks (lines 103–108, severity=high): "Every frontend/src/components/*.tsx file MUST carry: // MIGRATION TARGET: @hylla/stil-solid ... Every frontend/src/lib/vim/*.ts file MUST carry: // MIGRATION TARGET: github.com/hylla-org/ro-vim ... These are load-bearing audit markers. A Vitest test in D3 enforces them as a CI gate."

Caveat: `wails.ts` is at `frontend/src/lib/wails.ts` (NOT under `src/lib/vim/`). D4's AC (line 427) says it carries the `@hylla/stil-solid` marker. The migration-markers Vitest test walks `src/components/` + `src/lib/vim/` — it does NOT walk `src/lib/wails.ts`. The test would not catch a missing marker on `wails.ts`. This is consistent with the constraint at lines 103–104 ("Every frontend/src/components/*.tsx file MUST carry: // MIGRATION TARGET: @hylla/stil-solid") — `wails.ts` is not in that path glob. D4's AC line 427 makes it an authoring requirement but not a CI gate. Not a load-bearing FF — `wails.ts` is consumed only by components, which DO carry markers; the audit chain remains intact through the components even if `wails.ts` itself is unmarked. See R2-NIT2 below.

**Conclusion:** Migration markers covered as load-bearing per-droplet AC + Vitest CI gate. **PASS** (with R2-NIT2).

### 2.6 D9 also touches `frontend/astro.config.mjs` (vite proxy)

**Premise:** D9 RiskNotes mandates a vite proxy for `/.tillsyn-bindings.json` → `../../.tillsyn/bindings.json`, which requires editing `astro.config.mjs` (created by D3). D9 must declare this file in paths and the blocker chain must serialize the edit.

**Evidence:**

- D9 "Updated paths" (line 687): `frontend/astro.config.mjs` (EXISTING — D9 adds vite proxy entry for `/.tillsyn-bindings.json`).
- D9 KindPayload (line 701): `{"file": "frontend/astro.config.mjs", "symbol": "vite proxy", "action": "modify", ...}`.
- D9 RiskNotes (line 681 + 684): "The Astro vite dev-server proxy is configured in `frontend/astro.config.mjs` to proxy `/.tillsyn-bindings.json` → `../../.tillsyn/bindings.json` (relative to the `frontend/` dir). Builder adds this proxy entry to `astro.config.mjs`. ... Add `frontend/astro.config.mjs` to D9 paths as MODIFY."
- D9 `blocked_by W6.D3` (line 644). D3 creates `astro.config.mjs` (D3 KindPayload line 393).
- D4–D8 do NOT modify `astro.config.mjs` (D4 KindPayload lines 444–449; D5 lines 489–495; D6 lines 532–538; D7 lines 577–584; D8 lines 624–632). No concurrent edit risk.

**Conclusion:** File-conflict properly serialized. **PASS** (with R2-NIT1 — see below — about the proxy rewrite mechanism plausibility).

---

## 3. Round-2 NITs (new, minor)

### R2-NIT1 — D9 vite proxy `rewrite` semantics ambiguous

**Location:** PLAN.md line 701 (D9 KindPayload "vite proxy" shape_hint).

**Issue:** The shape_hint reads:

```
vite: { server: { proxy: { '/.tillsyn-bindings.json': { target: 'http://localhost:4321', rewrite: () => '../../.tillsyn/bindings.json' } } } } — or equivalent Astro/Vite proxy config; builder picks idiomatic Astro v5 form
```

Vite's `proxy.rewrite` produces a URL path, NOT a filesystem path. A `rewrite` returning `'../../.tillsyn/bindings.json'` would still send an HTTP request to the proxy target — the rewrite doesn't translate URL → filesystem. To serve a file from outside `frontend/public/` at dev-time, Astro/Vite needs either:

- a custom middleware via the `astro:server:setup` integration hook (Context7 `/withastro/docs` confirms this is the canonical Astro v5 pattern for dev-server middleware injection), or
- a static-server route mapping (no canonical Vite primitive for cross-fs serving without middleware).

The plan's shape_hint says "or equivalent Astro/Vite proxy config; builder picks idiomatic Astro v5 form" — leaving the actual mechanism to builder discovery. The graceful-fallback baseline-only behavior (D9 AC line 668; RiskNote line 681) covers the case where the fetch fails, so a working baseline-only fallback ships even if the dev-time proxy is mis-authored.

**Severity:** Low — non-blocking. The mechanism is technically open but the plan has a robust fallback (baseline-only) when the dev-time mechanism doesn't deliver. Builder will discover during D9 implementation; build-QA-falsification can attack at build close.

**Suggested fix (optional):** PLAN.md could pre-bake the canonical Astro v5 form. Recommended phrasing: "use an `astro:server:setup` integration hook in `astro.config.mjs` that injects a Vite middleware reading `path.resolve(__dirname, '../../.tillsyn/bindings.json')` and responding with the file contents on GET `/.tillsyn-bindings.json`; 404 if file absent." This would replace the proxy-rewrite shape_hint.

**Disposition:** NIT — DEFERRED-WITH-REASON (graceful fallback is the safety net; builder discovery is acceptable; rewriting the shape_hint now would force a third proof round for a single-symbol edit). Surface as build-QA-falsification target during D9 close-out.

### R2-NIT2 — `frontend/src/lib/wails.ts` is outside the migration-markers Vitest test walk

**Location:** PLAN.md AC5 (line 63), D3 AC (line 375), D3 KindPayload (line 398).

**Issue:** AC5 requires markers on `frontend/src/components/*.tsx` and `frontend/src/lib/vim/*.ts`. The migration-markers Vitest test (`frontend/tests/migration-markers.test.ts`) walks exactly those two directories. But `frontend/src/lib/wails.ts` is in `frontend/src/lib/` — not in `frontend/src/lib/vim/` — and per D4 AC line 427 the file MUST also carry `// MIGRATION TARGET: @hylla/stil-solid`. The test won't enforce it. A future R-loop could drop the marker from `wails.ts` and it would silently slip past CI.

**Severity:** Low — the marker is still required by D4's per-droplet AC (an authoring-time gate). Loss of CI enforcement is the gap.

**Suggested fix (optional):** Extend the Vitest test to also walk `frontend/src/lib/` non-recursively (or recursively but with a vim-vs-non-vim path-prefix branch on the asserted marker substring). The cheapest is: walk all `.ts`/`.tsx` files under `frontend/src/`, derive expected-marker from path: `lib/vim/` → `ro-vim`; everywhere else → `@hylla/stil-solid`.

**Disposition:** NIT — DEFERRED-WITH-REASON (D4 AC remains the authoring gate; the omission affects future refactors more than the v1 ship). Surface in W6 build-QA-proof close-out.

---

## 4. Verdict

**PASS — Round 2 absorption complete.**

All Round-1 findings resolved:

- Proof FF1 (KindPayload internal drift): RESOLVED (orchestrator-fixed pre-R2; preserved cleanly).
- Fals FF1 (`mage ci` exclusion unverified, HIGH): RESOLVED via R10-D3 + D1 AC concrete pre/post comparison.
- Fals FF2 (Wails layout non-canonical, HIGH): RESOLVED via R10-D3 canonical-root layout.
- Fals FF3 (fonts missing, HIGH): RESOLVED via D3 `@fontsource/*` deps + D4 imports.
- Fals FF4 (D9 4-alternative drift, MEDIUM): RESOLVED via ONE-path-each + explicit refutation of alternatives.
- All R1 Proof NITs (7): absorbed.
- All R1 Fals NITs (6): 5 absorbed (N3–N7); N8 deferred-with-reason.

Two new minor R2 NITs (R2-NIT1 vite proxy `rewrite` ambiguity; R2-NIT2 `wails.ts` outside migration-markers test walk) are non-blocking and DEFERRED-WITH-REASON to build-QA close-out. Plan is dispatch-ready.

---

## 5. Closing Certificate

**Premises:**

- R10-D3 (Wails at tillsyn repo root) must be reflected in every path declaration; no `fe/go.mod`, no `replace`, no `fe/` subfolder; `//go:build wails` on root `main.go` + `app.go` with the constraint as the FIRST line followed by a blank line.
- Proof FF1 (KindPayload ↔ _BLOCKERS ↔ per-droplet `blocked_by` consistency) must remain resolved post-rewrite.
- Fals FF1/FF2/FF3/FF4 from R1 must each map to a concrete plan change with file:line evidence.
- 9 droplets enumerated must equal 9 droplets narrated must equal 9 droplets blocker-tracked.
- All FE component files carry `// MIGRATION TARGET: @hylla/stil-solid`; all vim engine files carry `// MIGRATION TARGET: github.com/hylla-org/ro-vim`; load-bearing as audit markers + CI gate via D3 Vitest test.

**Evidence:**

- W6 PLAN.md (rewritten): lines 1–801.
- W6 `_BLOCKERS.toml`: lines 1–40.
- W6 PLAN_QA_PROOF.md (Round 1): lines 1–203.
- W6 PLAN_QA_FALSIFICATION.md (Round 1): lines 1–296.
- L1 PLAN.md: lines 12–34 (Round 10 changelog), 520–605 (W6 spec), 910–937 (locked decisions).
- REVISION_BRIEF.md §2.15, §2.19.
- SKETCH.md §10.
- Direct file reads: `stil/main/src/styles/tokens.css` (lines 1–96, no `@font-face`), `stil/main/package.json` (4 `@fontsource/*` deps), `stil/main/src/bindings/baseline.json` (lines 100–109, 4 commands in `product_extensions.tillsyn`).
- `magefile.go` line 149 (`CI()` exists); no `CiFe()` today (confirmed via `git grep`).
- Context7 `/wailsapp/wails` "Project structure rundown" (canonical layout: `go.mod` + `main.go` + `wails.json` + `frontend/` at root).
- Context7 `/withastro/docs` `astro:server:setup` hook (vite middleware injection canonical pattern).

**Trace or cases:**

- 9 droplets present: D1 line 249, D2 line 296, D3 line 343, D4 line 406, D5 line 455, D6 line 501, D7 line 544, D8 line 590, D9 line 638.
- 7 blocker entries in `_BLOCKERS.toml` matching 7 non-root droplets (D2, D4, D5, D6, D7, D8, D9). D1 and D3 are Wave-A roots with no blockers.
- KindPayload `blocked_by` cross-checked row-by-row against per-droplet headers and `_BLOCKERS.toml` (§1.2 table) — all 9 rows agree.
- R10-D3 path migration verified: `git grep "fe/" PLAN.md` returns nothing; `git grep "fe/" _BLOCKERS.toml` returns nothing.
- R1 FF / NIT absorption table verified row-by-row (§1.7, §1.8).
- Migration-markers AC traced through every droplet that adds component or vim files (§2.5).
- D9 vite proxy ambiguity flagged as R2-NIT1 with graceful-fallback safety net; non-blocking.

**Conclusion:** **PASS.** Round 2 absorbs every blocking finding from Round 1, preserves Proof FF1 resolution, and surfaces only minor build-QA-falsification surface area. Plan is dispatch-ready for the build phase.

**Unknowns:**

- Astro v5 idiomatic form for the `/.tillsyn-bindings.json` middleware (R2-NIT1) — builder will discover; graceful-fallback baseline-only is the safety net.
- `frontend/src/lib/wails.ts` marker enforcement gap (R2-NIT2) — D4 per-droplet AC is the authoring gate; CI enforcement could be extended in a future refinement.
- Cross-wave consistency test for W5 (Go TUI vim) vs W6 (FE TS vim) command-set drift remains deferred per W6 R1 falsification §5 — belongs in W2 init smoke or Drop 4c.7. Not a W6 concern.

---

## 6. Hylla Feedback

Hylla OFF per spawn directive. No Hylla calls attempted. No miss to record.
