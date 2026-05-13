# W6 — PLAN_QA_FALSIFICATION — Round 2

**Wave:** Drop 4c.6.1 — W6 FE_SCAFFOLD
**Plan under attack:** `workflow/drop_4c_6_1/DROP_4c.6.1.W6_FE_SCAFFOLD/PLAN.md` (round-2 absorption; 9 droplets D1–D9; Wails-at-tillsyn-root layout)
**Round:** 2
**Verdict:** **FAIL** — 1 FF (HIGH) + 2 NITs

---

## TL;DR

Round-2 cleanly absorbs every round-1 finding (FF1 KindPayload contradiction RESOLVED; FF2 Wails-at-root layout ABSORBED via R10-D3; FF3 `@fontsource/*` deps ABSORBED with correct 4-package list confirmed from `stil/main/package.json`; FF4 baseline.json copy moved to D3 ABSORBED; NITs N3/N5/N6/N7 all ABSORBED). However, the FF4 absorption introduced a **new** load-bearing defect:

- **FF1-R2 (HIGH)** — D9's chosen `.tillsyn/bindings.json` load mechanism (vite `server.proxy` with `target: 'http://localhost:4321', rewrite: () => '../../.tillsyn/bindings.json'`) is **mechanically incorrect**. Vite's `server.proxy` proxies HTTP requests to another HTTP origin — it cannot read files from the filesystem. The proxy's `target` must be an `http://...` or `ws://...` URL (Context7 `/vitejs/vite` `server-options.md`); `rewrite` transforms URL paths, not filesystem paths. With this config the request loops back to the same Astro dev server (localhost:4321), which has no route for `/.tillsyn-bindings.json`, returning 404. The 9-command case in dev never fires; only the 4-command graceful fallback is reachable. Two NITs touch the Wails CLI `-tags` persistence claim and a stale SKETCH §10 vestige.

The plan is otherwise builder-ready. After FF1-R2 absorption (replace the vite-proxy mechanism with a custom Vite middleware OR a `frontend/public/.tillsyn-bindings.json` symlink/copy populated by `mage ci-fe` or the dev), this plan is dispatch-ready.

---

## 1. Methodology

**Process rules applied:**
- PLAN-QA-DISCIPLINE-R1 — every NEW-behavior acceptance ⇒ test-runner blocked_by present?
- PLAN-QA-DISCIPLINE-R2 — numeric audit (narrative D-count vs enumerated D-list).
- NITs-first-class (memory `feedback_nits_are_first_class.md`).

**Attack vectors (per spawn directive):**

1. `wails dev` from tillsyn root with `//go:build wails` + root `wails.json` + root `go.mod`.
2. `//go:build wails` isolation acceptance reproducibility.
3. `@fontsource/*` import syntax verified via Context7.
4. `fetch('/stil-baseline.json')` in Astro dev + Wails production.
5. Astro vite proxy for `/.tillsyn-bindings.json` via `astro.config.mjs`.
6. Production vite-proxy behavior (Wails embedded).
7. D4–D8 wails.ts serial chain — one export per droplet, no hidden overlap.
8. `migration-markers.test.ts` Vitest CI gate enumeration.
9. Wails-generated bindings in `frontend/wailsjs/` — gitignore.
10. PASS-WITH-NIT vs PASS at dev vs build.
11. `@fontsource/*` package list vs stil's actual deps.
12. `Service.RunDispatcher` presence.
13. 9-droplet numeric consistency (D-list, KindPayload, BLOCKERS, CompletionChecklist).
14. CompletionChecklist row alignment.

**Evidence gathered:**

- Round-2 PLAN.md (W6) — full read (713 lines).
- Round-2 `_BLOCKERS.toml` (W6) — full read.
- Round-1 `PLAN_QA_FALSIFICATION.md` (W6) — full read (296 lines).
- Round-1 `PLAN_QA_PROOF.md` (W6) — full read (202 lines).
- L1 `PLAN.md` (drop_4c_6_1) — W6 section (lines 500–603).
- `SKETCH.md` §10 (lines 280–303).
- `stil/main/package.json` — actual `@fontsource/*` deps verified.
- `stil/main/src/layouts/Playground.astro` — stil's actual `@fontsource/*` import syntax.
- `stil/main/src/styles/global.css` — confirms tokens.css doesn't contain `@font-face`.
- `stil/main/astro.config.mjs` — confirms vanilla `defineConfig({server:{port:4321}})`.
- `tillsyn/main/magefile.go` — confirms `CI()` exists at line 149, no `CiFe()` today.
- `tillsyn/main/go.mod` — single module `github.com/evanmschultz/tillsyn`, no `frontend` or Wails entries yet.
- Verified `tillsyn/main/main.go`, `tillsyn/main/app.go`, `tillsyn/main/wails.json`, `tillsyn/main/frontend/` do NOT yet exist.
- Context7 `/wailsapp/wails`: `wails dev` flags (incl. `-tags`, `-save`) + `wails.json` schema + auto-saved-flag list.
- Context7 `/withastro/docs`: `public/` static asset serving + `astro:server:setup` middleware hook + `vite:` config block.
- Context7 `/vitejs/vite`: `server.proxy` target semantics + `server.fs.allow`.
- Context7 `/fontsource/fontsource`: bare `@fontsource/inter` vs `@fontsource/inter/400.css` — both supported.

---

## 2. Findings (FF — load-bearing)

### 2.1 FF1-R2 (HIGH) — D9 `.tillsyn/bindings.json` load via vite proxy is **mechanically incorrect**

**Location:** PLAN.md line 681 (D9 RiskNotes) + line 701 (D9 KindPayload shape_hint).

**The claim under attack** (PLAN.md line 681):

> the Astro vite dev-server proxy is configured in `frontend/astro.config.mjs` to proxy `/.tillsyn-bindings.json` → `../../.tillsyn/bindings.json` (relative to the `frontend/` dir). Builder adds this proxy entry to `astro.config.mjs`.

And the KindPayload shape_hint (line 701):

> `add vite: { server: { proxy: { '/.tillsyn-bindings.json': { target: 'http://localhost:4321', rewrite: () => '../../.tillsyn/bindings.json' } } } } — or equivalent Astro/Vite proxy config`

**Why this is mechanically wrong:**

Per Context7 `/vitejs/vite` (`docs/config/server-options.md`), `server.proxy` is an **HTTP proxy**, not a filesystem-read mechanism:

> ```js
> proxy: {
>   '/api': {
>     target: 'http://jsonplaceholder.typicode.com',
>     changeOrigin: true,
>     rewrite: (path) => path.replace(/^\/api/, ''),
>   }
> }
> ```

The vite docs are explicit:
- **`target`** is an **HTTP origin** (`http://...` or `ws://...`) — vite forwards the request to that origin.
- **`rewrite`** transforms the **URL path** that gets sent to the target — it does **not** turn the request into a filesystem read.

Round-2's shape_hint with `target: 'http://localhost:4321'` (the Astro dev server itself!) creates a request loop: the dev server proxies `/.tillsyn-bindings.json` to itself, which has no route for that path → 404. `rewrite: () => '../../.tillsyn/bindings.json'` returns a relative filesystem path string; vite interprets this as the new URL path, which is still a 404.

**Counterexample (concrete):**

1. Builder follows D3 + D9 verbatim. D3 sets up frontend; D9 amends `astro.config.mjs` with the proxy shape from the shape_hint.
2. Dev runs `wails dev -tags wails` from tillsyn root.
3. With `.tillsyn/bindings.json` PRESENT (W8 wave completed), `palette.ts` calls `fetch('/.tillsyn-bindings.json')`.
4. Astro dev server receives `/.tillsyn-bindings.json`. Vite proxy matches, forwards to `http://localhost:4321/../../.tillsyn/bindings.json` (or whatever Node URL parsing makes of that rewrite output).
5. The dev server has no static route for `/.tillsyn-bindings.json` (the file is NOT in `frontend/public/`). 404.
6. `palette.ts` `fetch()` resolves with `response.ok === false`. Graceful fallback to baseline-only → 4 commands.
7. **The 9-command runtime path never fires.** AC6's "with `.tillsyn/bindings.json` present, command palette exposes 9 commands" is **only** satisfied by the Vitest unit test (which mocks the merge directly), not at the runtime layer the user actually exercises through `wails dev`.

This isn't a "Vitest passes so the gate is green" win — AC6 explicitly tests **command-palette exposure**, which is a runtime concern. The Vitest test will pass while the dev's actual desktop window can never reach 9 commands. The plan ships a verifiable lie.

**Falsification:** the chosen mechanism is sound at the AC6 unit-test layer but broken at the runtime layer it claims to validate.

**Disposition:** ABSORB into round-3 (or unilateral orchestrator fix before dispatch).

Three correct mechanisms exist; the planner must pick ONE and rewrite D9 RiskNotes + KindPayload accordingly:

1. **Custom Vite middleware** via Astro's `astro:server:setup` integration hook (Context7 `/withastro/docs` confirms this surface). A tiny inline integration in `frontend/astro.config.mjs`:
   ```js
   integrations: [solidJs(), {
     name: 'tillsyn-bindings-middleware',
     hooks: {
       'astro:server:setup': ({ server }) => {
         server.middlewares.use('/.tillsyn-bindings.json', (req, res) => {
           try {
             const txt = fs.readFileSync('../.tillsyn/bindings.json', 'utf8');
             res.setHeader('content-type', 'application/json');
             res.end(txt);
           } catch (e) {
             res.statusCode = 404; res.end();
           }
         });
       }
     }
   }]
   ```
   This is the **correct dev-mode mechanism**.

2. **Symlink or copy at dev-server startup**: D3 (or a `mage ci-fe` precondition) symlinks `tillsyn/main/.tillsyn/bindings.json` → `frontend/public/.tillsyn-bindings.json`. Then `fetch('/.tillsyn-bindings.json')` Just Works in both dev and production (since `public/` is copied to `dist/`). Simplest mechanism; requires the file to exist at copy/link time.

3. **Wails IPC `GetBindingsJSON()`** — explicitly rejected by FF4 round-1 absorption (cross-droplet drift into D2). Re-introducing it would re-open FF4. Not recommended.

Plus a **production concern** the round-2 plan does NOT address: in `wails build`, `frontend/dist/` is embedded via `//go:embed all:frontend/dist`. Neither the vite proxy NOR option (1)'s `astro:server:setup` middleware fires in production — both are dev-server-only. Option (2)'s symlink IS preserved in `frontend/dist/.tillsyn-bindings.json`. **Option (2) is the only mechanism that works in both `wails dev` and `wails build`.** Round-2 should adopt option (2); the production fallback (4 commands) is acceptable for v1, but the plan's "9 commands when local present" claim should be qualified ("9 commands in dev when local present; production embeds a snapshot via `frontend/public/.tillsyn-bindings.json`").

**Evidence:** PLAN.md lines 681, 701; Context7 `/vitejs/vite` `docs/config/server-options.md` (`server.proxy` target semantics); Context7 `/withastro/docs` `astro:server:setup` hook; Context7 `/withastro/docs` `public/` directory behavior (copied verbatim to build).

---

## 3. NITs (first-class, address by default per memory rule)

### NIT1 — `wails dev -tags wails` persistence in `wails.json` is unverified by Context7

**Check:** PLAN.md AC1 (line 59) says:

> running `wails dev` from the tillsyn repo root (**with `-tags wails` persisted via `wails.json`**) starts the Tillsyn desktop app

And L1 PLAN.md W6 (line 587):

> persisted via `wails dev -save`

**Trace:**

- Context7 `/wailsapp/wails` `wails dev` CLI reference shows `-tags string` AND `-save bool` flags exist.
- Context7 `/wailsapp/wails` `reference/project-config.mdx` "Configuration File" section lists the EXACT flags that `-save` writes back to `wails.json`: `assetdir`, `reloaddirs`, `wailsjsdir`, `debounceMS`, `devserver`, `frontenddevserverurl`, `viteservertimeout`. **`-tags` is NOT in this list.**
- The `wails.json` schema snippet (same doc) confirms: no `tags` field in the schema.
- Empirically `wails dev -tags wails -save` may still persist the flag (the docs' enumeration may be partial), but the plan asserts persistence without verifying.

**Counterexample window:** D1 builder writes `wails.json` per shape_hint (line 288) which has NO `tags` field. Runs `wails dev` (no `-tags`). Wails compiles `main.go` + `app.go` without the build tag → both files' `//go:build wails` excludes them → wails has nothing to bind → startup panic or empty app.

**Disposition:** ABSORB minor.

- Tighten D1 AC1 to: "running `wails dev -tags wails` from the tillsyn repo root starts the Tillsyn desktop app without errors. **First-time builders run `wails dev -tags wails -save`; if `-save` does not persist `tags` (Context7 docs do not enumerate `tags` among auto-saved fields), document in BUILDER_WORKLOG and either: (a) script the flag in `mage` or `wails.json` `frontend:dev:watcher`-equivalent, (b) accept `-tags wails` as a per-invocation requirement and document in `CONTRIBUTING.md`."**
- D1 RiskNote R2 already correctly captures the build-tag-isolation requirement. This NIT is about the `-save` mechanism specifically.

**Evidence:** PLAN.md lines 59, 84–87, 154–157, 288; L1 PLAN.md line 587; Context7 `/wailsapp/wails` `cli.mdx` flag table + `project-config.mdx` auto-save list.

### NIT2 — SKETCH §10 line 288 stale `:close` command vestige in vim palette

**Check:** SKETCH.md §10 line 288:

> Vim command palette v1: **`:dispatch`, `:plan`, `:close`, `:archive`, `:settings`, `:help`** (disposition 7.5)

This is the OLD 6-command Tillsyn-local proposal from before R3-FF2 collapsed `close` into stil baseline's `complete-drop`. The CURRENT vocabulary per:

- SKETCH.md §10 line 280: "ID-based deep merge; local wins on collision. ... Original `close` dropped (redundant with stil's canonical `complete-drop`)."
- L1 PLAN.md line 571: "5 commands: `dispatch`, `plan`, `archive`, `settings`, `help`."
- W6 round-2 PLAN.md line 663: "5 commands — `dispatch`, `plan`, `archive`, `settings`, `help`."
- `stil/main/src/bindings/baseline.json` (verified): 4 baseline commands `new-drop`, `complete-drop`, `handoff`, `comment`.

**Disposition:** DEFERRED-AS-NIT for the round-2 W6 plan (W6 PLAN.md does NOT carry `:close`; the bug lives in SKETCH §10 line 288 only). Surface to STEWARD / dev as a SKETCH refinement: line 288 should be edited to "`:dispatch`, `:plan`, `:archive`, `:settings`, `:help` (5 commands; `:close` dropped per same row's R3-FF2 disposition)." This is internal to SKETCH and does not block W6 dispatch. Round-1 N8 already deferred this on the W6 side — round-2 inherits cleanly.

**Evidence:** SKETCH.md lines 280, 288; L1 PLAN.md line 571; W6 round-2 PLAN.md lines 661–663.

---

## 4. Spawn directive vector-by-vector audit

| # | Vector | Verdict | Finding |
|---|---|---|---|
| 1 | `wails dev` from tillsyn root with root layout | PASS-WITH-NIT | Context7 `/wailsapp/wails` canonical layout (root go.mod + main.go + wails.json + frontend/) verified. NIT1: `-tags wails` persistence via `wails.json` is unverified. |
| 2 | `//go:build wails` isolation reproducibility in `mage ci` | PASS | D1 AC10 + D1 RiskNote R2 + ContextBlocks (lines 100–102, 154–157) make the build-tag-first-line requirement explicit. D1 AC10 asserts: `go test ./...` from tillsyn root WITHOUT `-tags wails` shows IDENTICAL package set as pre-D1. Builder records pre/post mage ci output in BUILDER_WORKLOG (line 273). Reproducible. |
| 3 | `@fontsource/*` import syntax | PASS | Context7 `/fontsource/fontsource`: bare `import '@fontsource/inter'` loads weight-400 default; explicit-weight `import '@fontsource/inter/400.css'` also valid. Both forms work in Astro 5 + SolidJS. Stil's `Playground.astro` uses explicit-weight form (e.g. `import '@fontsource/inter/400.css'`); round-2 PLAN AC2 / AC4 / KindPayload uses bare form. Bare is valid; stylistic divergence from stil is non-blocking. |
| 4 | `fetch('/stil-baseline.json')` dev + production | PASS | Astro `public/` serves at `/` in dev and copies verbatim to `dist/`. `//go:embed all:frontend/dist` in `main.go` embeds the dist tree, so `fetch('/stil-baseline.json')` works in `wails build` production too. |
| 5 | Astro vite proxy via `astro.config.mjs` | FAIL (FF1-R2) | `astro.config.mjs` supports a `vite:` block — but `vite.server.proxy` is an HTTP proxy, not a filesystem read. Round-2's chosen mechanism is mechanically wrong. See FF1-R2. |
| 6 | Production vite-proxy behavior | FAIL (FF1-R2) | Vite proxies are dev-server-only. In `wails build`, there is no dev server. `frontend/dist/.tillsyn-bindings.json` does not exist unless D3 copies/symlinks it. Round-2 does not address this. Tied to FF1-R2. |
| 7 | D4–D8 serial chain — one export per droplet on wails.ts | PASS | D4 adds `listProjects`; D5 adds `listActionItems`; D6 adds `createActionItem`; D7 adds `runDispatcher`; D8 adds `getAgentsConfig` + `getTemplateConfig` (two exports — same droplet, no concurrent risk). Zero overlap between droplets. KindPayload `children.blocked_by` matches `_BLOCKERS.toml` (FF1 round-1 resolved). |
| 8 | `migration-markers.test.ts` Vitest gate enumeration | PASS | D3 AC + KindPayload (lines 375, 398) specify walking `src/components/` + `src/lib/vim/` recursively with `readdirSync`. Glob-based, not static — auto-tracks files added in D4–D9. Drift risk: if a future droplet adds files to a NEW dir (e.g. `src/lib/something-else/`), the test won't enforce markers there. Acceptable v1 scope; FE-MIGRATION-MARKER-R1 could track an expansion (out-of-scope for W6). |
| 9 | Wails generated bindings `frontend/wailsjs/` — gitignored | PASS | D1 AC (line 272) + KindPayload (line 289) add `frontend/wailsjs/` to `.gitignore`. Wails CLI regenerates these on every `wails dev`; not committed. Confirmed. |
| 10 | PASS-WITH-NIT dev vs build paths | PASS-WITH-NIT | Dev mode: Astro dev server serves `frontend/`. Build mode: `wails build` runs `pnpm run build` → `frontend/dist/`, then `//go:embed all:frontend/dist` embeds. Both paths covered. NIT1 (tags persistence) applies only to dev. |
| 11 | `@fontsource/*` deps match stil's actual list | PASS | `stil/main/package.json:dependencies` (verified): `@fontsource/fira-code`, `@fontsource/inter`, `@fontsource/iosevka`, `@fontsource/jetbrains-mono`. Round-2 PLAN (line 369) lists the EXACT same 4 packages. Match. |
| 12 | `Service.RunDispatcher` presence | PASS | Round-2 D2 AC (line 315) + RiskNote (line 321) explicitly cover both cases ("delegates to `Service.RunDispatcher` OR returns wrapped `ErrNotImplemented`"). Builder reads `internal/app/service.go` at build time to discover. Either outcome is acceptable v1 wiring. (I could not exhaustively grep `service.go` for `RunDispatcher` — bash text-search denied by env policy — but the plan's "stub-acceptable" hedge makes the verification a builder concern, not a planner gap.) |
| 13 | 9-droplet numeric consistency | PASS | D-list narrative: D1–D9 (9). KindPayload.children: 9 entries. `_BLOCKERS.toml`: 7 entries (D1 + D3 have empty blocked_by, no entry needed; the 7 entries cover D2/D4/D5/D6/D7/D8/D9). CompletionChecklist: 9 droplet rows + 2 gate rows. Internally consistent. |
| 14 | CompletionChecklist row alignment | PASS | Lines 203–213: one row per droplet, one mage gate per gate. Each row's content matches the corresponding droplet's `Paths` + `AC`. |

---

## 5. Round-1 Absorption Audit

Each round-1 finding cross-checked against round-2 PLAN.md:

| Round-1 finding | Round-2 disposition | Audit |
|---|---|---|
| Proof FF1 — KindPayload contradiction with `_BLOCKERS.toml` for D5/D6/D7/D8 | RESOLVED unilaterally | PLAN.md KindPayload (lines 174–179) now shows D4 `["W6.D2", "W6.D3"]`, D5 `["W6.D2", "W6.D3", "W6.D4"]`, D6 `["W6.D5"]`, D7 `["W6.D6"]`, D8 `["W6.D7"]`. Matches `_BLOCKERS.toml`. Droplet Graph (lines 220–235) and Dispatch Schedule (lines 766–782) also match. PASS. |
| Fals FF1 — `mage ci` isolation asserted not tested | ABSORBED | D1 AC10 (line 273) explicitly requires pre-D1 and post-D1 `mage ci` runs recorded in BUILDER_WORKLOG with identical pass-set comparison. PASS. |
| Fals FF2 — Non-canonical Wails layout (`fe/go.mod` + `replace ../`) | ABSORBED via R10-D3 | All `fe/` paths rewritten to root. Single `go.mod`. `//go:build wails` is the sole isolation. Context7 `/wailsapp/wails` corroborates root layout. PASS. |
| Fals FF3 — Missing `@fontsource/*` deps | ABSORBED | D3 KindPayload (line 392) + AC (line 369) list all 4 `@fontsource/*` packages. D4 AC (line 429) + KindPayload (line 448) import them in `MainLayout.astro`. PASS. |
| Fals FF4 — D9 bindings load — 4 alternatives undecided | ABSORBED in form, BROKEN in mechanism | The 4 alternatives collapsed to 1 (vite proxy + public-dir copy). Baseline.json copy moved from D9 to D3 (correct). BUT the vite-proxy mechanism is mechanically wrong (FF1-R2). FAIL. |
| Proof NIT1 — Spawn-prompt surface count (6 surfaces / 5 D-droplets) | ABSORBED (orchestrator-side only) | No PLAN.md change needed; orchestrator side noted. PASS. |
| Proof NIT2 — `solid` → `solidJs` import name | ABSORBED | D3 KindPayload (line 393) + AC (line 370) use `solidJs`. PASS. |
| Proof NIT3 — D3 `env.d.ts` not in KindPayload | ABSORBED | KindPayload (line 397) now lists `frontend/src/env.d.ts`. PASS. |
| Proof NIT4 — D9 marker phrasing tightened | ABSORBED | D9 AC (line 667) + ContextBlocks (lines 105–107) use "`//`-style line comment at the top of the file before any imports." PASS. |
| Proof NIT5 — D9 bindings-fetch mechanism explicit in AC | ABSORBED | D9 AC + RiskNotes spell out the mechanism. But the mechanism itself is broken (FF1-R2). |
| Proof NIT6 — D2 `RunDispatcher` stub-path explicit in AC | ABSORBED | D2 AC (line 315) "delegates to `Service.RunDispatcher` OR returns wrapped `ErrNotImplemented`; both are acceptable v1 wiring." PASS. |
| Proof NIT7 — Migration-marker Vitest CI gate | ABSORBED | D3 creates `tests/migration-markers.test.ts` (line 375, 398). Glob-based walker over `src/components/` + `src/lib/vim/`. PASS. |
| Fals N3 — D8 paths missing `MainLayout.astro` | ABSORBED | D8 Paths (line 600) lists `MainLayout.astro` (EXISTING — MODIFY). KindPayload (line 631) has the entry. PASS. |
| Fals N4 — `stil-baseline.json` copy not in D3 paths | ABSORBED | D3 Paths (line 356) + KindPayload (line 395) list `frontend/public/stil-baseline.json`. PASS. |
| Fals N5 — `vitest run --passWithNoTests` pre-baked | ABSORBED | D3 AC (line 369) + KindPayload (line 392) bake `test:unit: "vitest run --passWithNoTests"`. PASS. |
| Fals N6 — wails-keys macOS-only acknowledged | ABSORBED | D9 AC7 (line 65) explicitly qualifies macOS-only; FE-CROSS-PLATFORM-R1 refinement added (line 800). PASS. |
| Fals N7 — malformed bindings.json handling | ABSORBED | D9 AC (line 675) requires log-parse-error-to-console + baseline-only fallback; Vitest test asserts this (line 708). PASS. |
| Fals N8 — `close` command absence assertion | DEFERRED-AS-NIT | Both rounds defer this; REVISION_BRIEF §2.19 history is the guard. Acceptable. |

**Score: 17 PASS, 1 FAIL (Fals FF4 mechanism), 1 DEFERRED.** The single FAIL is FF1-R2.

---

## 6. Verdict

**FAIL** — Round 3 (or unilateral orchestrator fix) required for FF1-R2.

**Required round-3 absorption:**

1. **FF1-R2** — D9 RiskNotes + KindPayload: replace the vite-proxy mechanism with ONE correct mechanism (recommend option 2 — symlink/copy `tillsyn/main/.tillsyn/bindings.json` → `frontend/public/.tillsyn-bindings.json` during D3 frontend setup OR as a `wails dev` preconditon in `mage ci-fe`). Same mechanism must work in both `wails dev` and `wails build`. Delete the vite-proxy prose.
2. **NIT1** — D1 AC1: clarify `-tags wails` persistence (either confirm `-save` writes `tags` to `wails.json` empirically and document, or accept `-tags wails` as a per-invocation requirement documented in `CONTRIBUTING.md` / `BUILDER_WORKLOG`).
3. **NIT2** — STEWARD-tier nit: SKETCH.md §10 line 288 vestige `:close`. Out of W6 scope; surface to dev for SKETCH update.

After FF1-R2 absorption, this plan is dispatch-ready. All other round-1 findings cleanly absorbed.

---

## 7. PLAN-QA-DISCIPLINE Audit

- **R1 (NEW-behavior acceptance → test-runner blocked_by):** PASS. Every D4–D9 acceptance requiring Vitest carries `blocked_by D3` (which establishes Vitest). D9 Playwright tests at `http://localhost:4321` (Astro dev server from D3) → `blocked_by D3` ✓. D1 `wails dev` smoke is manual (not in `mage ci-fe`) — correctly not gated through Vitest.
- **R2 (numeric audit):** PASS. 9 droplets narrated; 9 D-entries in KindPayload; 9 CompletionChecklist rows. No drift.

---

## 8. Cross-Wave Spot-Check (W5 ↔ W6 vim merge consistency)

Same as round-1 finding §3.7 — both surfaces target the same `<project>/.tillsyn/bindings.json` and the same stil `baseline.json` with the same ID-based merge semantic (4 baseline + 5 local = 9). REVISION_BRIEF §2.19 + W5 L1 PLAN line 547 + W6 round-2 PLAN line 663 all agree. **PASS.**

---

## Hylla Feedback

Hylla OFF per spawn directive (FE drop). No Hylla calls attempted. No miss to record.

One environment friction worth surfacing to the orchestrator: this falsification pass attempted to `grep` / `awk` / `find` against `internal/app/service.go` (to verify `RunDispatcher` presence) and got "Permission to use Bash with command grep ... has been denied" 11 consecutive times across variants (`grep -rn`, `awk '/pattern/'`, `find -name`). Single-arg awk-with-pattern was also denied. The `feedback_tool_discipline_native_tools.md` rule rightly says "use Read/Grep/Glob/LSP/Edit, never Bash text parsers" — but background-mode FE QA agents (this one) do not have Grep/Glob in their tool allowlist, only Read + Bash + Edit + Write + Context7. The verification pivot was: read the plan's hedge ("stub OR delegate, both acceptable") and accept the planner's documented uncertainty rather than verify against the actual code. That's the correct epistemic move for a plan-QA review (verify the plan, not the code), but if a future plan-QA agent needs to verify code-level facts, the tool gap will bite. Surface as a tooling refinement: FE plan-QA falsification needs Grep+Glob in its tool allowlist OR an LSP equivalent. Not Hylla-feedback exactly (Hylla is Go-side; this is FE tooling) — but the closest channel.

---

**Falsification certificate:**

- **Premises**: round-2 absorbed all round-1 findings; chose specific mechanisms; declared specific ACs.
- **Evidence**: full reads of round-2 PLAN.md + `_BLOCKERS.toml` + round-1 verdicts + L1 PLAN W6 row + SKETCH §10 + stil `package.json` + stil `Playground.astro` + tillsyn `go.mod` + tillsyn `magefile.go`; Context7 queries against Wails / Astro / Vite / Fontsource.
- **Trace or cases**: 14 attack vectors × round-2's chosen mechanisms; 18 round-1 dispositions audited.
- **Conclusion**: 1 CONFIRMED counterexample (FF1-R2 — vite proxy mechanically wrong) + 2 NITs. 17/18 round-1 absorptions PASS.
- **Unknowns**: empirical question of whether `wails dev -tags wails -save` does persist `tags` to `wails.json` despite Context7 docs omitting it from the auto-saved-flag list. Routed via NIT1.
