# DROP_FE_1_BOOTSTRAP — Builder QA Falsification

Append a `## Droplet N.M — Round K` section per QA attempt. Each section records attacks attempted against the builder's claim, with CONFIRMED / POSSIBLE / NIT / REFUTED verdicts.

## Droplet 1.1 — Round 1

- **QA agent:** `fe-qa-falsification-agent`
- **Round:** 1
- **Verdict:** PASS with 1 NIT (counterexample count: 0 CONFIRMED, 0 POSSIBLE, 1 NIT, 9 REFUTED).
- **Builder claim under attack:** D1.1 successfully relocated `frontend/` → `ui/frontend/`, fixed the symlink target to `../../../.tillsyn/bindings.json`, pinned `packageManager: "pnpm@9.0.0"`, set `outputfilename: "Tillsyn"`, committed the lockfile, and `pnpm run build` exits 0.

### Attacks

**A1. Symlink target correctness + resolution + content. — REFUTED.**
- `readlink ui/frontend/public/.tillsyn-bindings.json` → `../../../.tillsyn/bindings.json` (three `../`, matches claim).
- Real file exists at repo-root `.tillsyn/bindings.json`, size 1113 bytes.
- Through-symlink `wc -c ui/frontend/public/.tillsyn-bindings.json` → `1113` (matches real file byte-for-byte).
- Head of resolved content is the expected `tillsyn-bindings` JSON schema.
- No counterexample.

**A2. Symlink stored as git symlink, not as regular file containing target string. — REFUTED.**
- `git ls-files --stage ui/frontend/public/.tillsyn-bindings.json` → `120000 f4f0668b... 0  ui/frontend/public/.tillsyn-bindings.json`.
- Mode `120000` is git's symlink mode (worklog noted the `git update-index --add --cacheinfo 120000,...` plumbing path; output confirms). The fallback path used did NOT degrade to `100644` regular-file. Wails build, deployment tooling, and any consumer treating the entry as a symlink will see a symlink.
- No counterexample.

**A3. `outputfilename: "Tillsyn"` actually applies to the binary. — REFUTED (with clarification).**
- Wails v2 docs (Context7 `/websites/wails_io` → "Project Config" → "outputfilename for the binary name"; v2 migration guide explicitly notes `binaryname` was RENAMED to `outputfilename` in v2): `outputfilename` controls the **compiled binary's filename**, NOT the macOS `.app` bundle name. The `.app` bundle name on darwin comes from `name` (which Wails uses to seed `Info.plist`).
- `ui/wails.json` has `"name": "Tillsyn"` AND `"outputfilename": "Tillsyn"` — both are consistent capitalization. So `wails build` on darwin produces `Tillsyn.app/Contents/MacOS/Tillsyn`. The change from `tillsyn` → `Tillsyn` in `outputfilename` is non-vacuous: it capitalizes the binary inside the bundle so it matches the bundle name. Without it, the bundle is `Tillsyn.app/Contents/MacOS/tillsyn` (lowercase) which can surface as a UX inconsistency in `ps`, Activity Monitor, and dock label tooltips.
- No counterexample to the claim "the edit applies". Builder may want to note in the droplet description that the bundle name is set by `name`, not `outputfilename`, but this is documentation polish — the edit itself is correct.

**A4. `packageManager: "pnpm@9.0.0"` enforcement — did install actually use 9.0.0? — REFUTED.**
- `head -1 ui/frontend/pnpm-lock.yaml` → `lockfileVersion: '9.0'`. pnpm 10.x emits `lockfileVersion: '10.0'` (or higher); pnpm 9.x emits `9.0`. The committed lockfile is the v9 dialect, which means Corepack DID intercept and run pnpm 9.0.0 as pinned — system 10.11.0 was bypassed.
- Reproducibility precondition (Corepack enabled) holds for the builder's dev machine; CI / new contributors who don't `corepack enable` will get the friendly warning + install completes via system pnpm. Behavior matches §N9 intent.
- No counterexample.

**A5. `.gitignore` regression — repo-root `node_modules/` no longer excluded after the rewrite. — REFUTED.**
- Read `.gitignore`: post-D1.1 entries scoped to `ui/build/`, `ui/frontend/node_modules/`, `ui/frontend/dist/`, `ui/frontend/.astro/`, `ui/frontend/wailsjs/`. There is NO bare `node_modules/` entry anywhere — neither pre- nor post-D1.1.
- This is technically NEW behavior (the old `.gitignore` had `frontend/node_modules/` only), but the FE work is now scoped to `ui/frontend/` and no sibling builder is creating stray `node_modules/` directories at repo root or elsewhere. Go builders don't run `npm`. Risk is hypothetical.
- No active counterexample, but see **NIT-1** below for a belt-and-suspenders suggestion.

**A6. Lockfile platform-specific entries — would CI re-resolve on Linux? — REFUTED.**
- `pnpm-lock.yaml` lines 160–500 enumerate `@esbuild/aix-ppc64`, `@esbuild/android-arm64`, `@esbuild/darwin-arm64`, `@esbuild/darwin-x64`, `@esbuild/linux-arm64`, `@esbuild/linux-x64`, `@esbuild/win32-x64`, etc. — ALL platforms listed simultaneously.
- pnpm lockfiles are cross-platform by design: `os:` / `cpu:` keys mark which platforms a given native binary applies to, but every variant is recorded. Linux CI resolving `@esbuild/linux-x64` from this lockfile is the documented happy path; no re-resolve triggered.
- No darwin-only entries that would block Linux resolution. No counterexample.

**A7. Stale `frontend/` references elsewhere in the repo. — REFUTED (D1.2 scoped reference confirmed; no orphaned mentions).**
- Grep across `.go`, `.md`, `.yml`, `.yaml`, `.json`, `.toml`, `.mjs` files (excluding `ui/`, `node_modules/`, `.git/`, `workflow/`):
  - `magefile.go:232,240` — `CiFe()` doc comment + `feDir := filepath.Join(wd, "frontend")`. **This is D1.2's scope per BUILDER_WORKLOG.md and PLAN.md** (the worklog explicitly defers `mage ci` to drop-end because `CiFe` still points at `frontend/`).
  - `README.md:316,325,374,375` — references to `default-frontend` template name, not paths. False positive.
  - `service_test.go`, `.tillsyn/agents/fe/*.md`, `.claude/settings*.json` — all reference `frontend/` as a noun (template, doc context), not as a real filesystem path.
- `ui/frontend/tsconfig.json` — `"include": [".astro/types.d.ts", "**/*"]`, `"exclude": ["dist"]` — no hardcoded paths referencing the old location.
- `ui/frontend/astro.config.mjs` — no `srcDir` / `outDir` / `root` overrides; Astro uses defaults relative to the config file's directory.
- `.github/workflows/ci.yml` and `release.yml` — zero `frontend` references.
- `wails.json` field `"frontend:dir": "frontend"` — this is RELATIVE to the `wails.json` file's directory. `wails.json` lives at `ui/wails.json`, so the path resolves to `ui/frontend/`. Correct.
- No counterexample. The only outstanding `frontend/` reference is the intentionally-deferred `magefile.go` `CiFe` target, scoped to D1.2.

**A8. Astro `prepareOutDir` symlink-stat failure — verify build still passes. — REFUTED (via direct evidence, build re-run blocked by sandbox).**
- Direct build re-run via `pnpm run build` denied by sandbox (logged in Unknowns).
- Indirect refutation: the symlink target is verified correct (A1), it is a real symlink in git (A2), and `statSync` of a resolvable symlink returns the target's stat result — `prepareOutDir` (Vite's `fs.statSync` over `public/` entries before clearing `dist/`) cannot ENOENT on a resolvable symlink. The failure mode the worklog describes (broken target ENOENT-ing `statSync`) is mechanically impossible against the current target.
- Worklog claims the build was run and exited 0 with `_astro/client.<hash>.js` emitted. Symlink correctness is independent evidence the build path is unblocked.
- No counterexample.

**A9. Astro 0-pages warning — does it escalate to non-zero exit under any flag? — REFUTED.**
- `ui/frontend/astro.config.mjs` has NO flags that escalate warnings (no `vite.build.emptyOutDir`, no strict-mode equivalent, no `output: 'server'` that would require pages).
- Context7 lookup against Astro docs (`/websites/astro_build_en`): "Missing pages directory" is a build-time warning; only `MissingIndexForInternationalization` (which fires only when `prefixDefaultLocale` is set in `i18n` config) escalates to a hard error. No `i18n` config in this project.
- No counterexample.

**A10. `.tillsyn/` cleanup semantics — symlink target may temporarily 404. — REFUTED (not D1.1's concern).**
- Test suite searches: no test under `internal/` or anywhere else removes/recreates the repo-root `.tillsyn/bindings.json`. The Tillsyn DB at `.tillsyn/tillsyn.db` is touched by some tests via `t.TempDir()` patterns, but `bindings.json` is treated as a static resource.
- Even if a future test cleaned `.tillsyn/`, this would 404 the symlink only during that test's window; FE builds don't run concurrently with that test.
- Out of D1.1 scope. No counterexample.

### NITs

**NIT-1. `.gitignore` no longer carries a bare `node_modules/` global pattern.**
- The pre-D1.1 `.gitignore` had `frontend/node_modules/` (scoped). The post-D1.1 `.gitignore` has `ui/frontend/node_modules/` (also scoped). Neither version carries a bare global `node_modules/`.
- Risk: hypothetical future tooling creating `node_modules/` outside `ui/frontend/` (e.g. an `examples/` subdir, a test fixture) would NOT be ignored.
- Severity: NIT. No current code path produces stray `node_modules/`; Go work doesn't touch pnpm; FE work is scoped to `ui/frontend/`.
- Mitigation (optional): add a bare `node_modules/` line to `.gitignore` as belt-and-suspenders. Not blocking D1.1 acceptance.

### Cross-Cutting

- **Parallel-builder isolation confirmed.** `git status --porcelain` shows D1.1's staged changes (all under `ui/`, `.gitignore`, `frontend/*` deletes, `ui/.gitignore`) cleanly disjoint from sibling-builder unstaged changes in `internal/adapters/mcp_rpc/*`, `internal/domain/comment*.go`, `internal/app/dispatcher/subscriber_test.go`. No cross-contamination.
- **Did NOT run `mage ci`** — would see sibling WIP per spawn-prompt rule. Verified per-attack evidence directly.

### Convergence

(a) No CONFIRMED or POSSIBLE counterexample produced. One NIT (.gitignore belt-and-suspenders) is non-blocking and optional.
(b) Evidence completeness: `readlink`, `git ls-files --stage`, `head -1` on lockfile, `wc -c`, Read of all relevant config files, Grep across non-excluded file types, Context7 lookups for Wails `outputfilename` semantics and Astro warning escalation behavior. All 10 attacks have concrete evidence.
(c) Unknowns: build re-run via `pnpm run build` was sandbox-denied; refuted Attack 8 via mechanical/structural evidence instead. Routing back to orchestrator: if dev wants the build re-run as independent confirmation, it can be done manually outside the sandbox. Not blocking the PASS verdict.

### Verdict

**PASS** — Builder D1.1's claim survives all 10 attacks. One optional NIT logged for `.gitignore` belt-and-suspenders. Ready for build-QA-proof sibling result + dev acceptance.
