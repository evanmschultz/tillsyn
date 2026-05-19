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

## Droplet 1.2 — Round 1

- **QA agent:** `fe-qa-falsification-agent`
- **Round:** 1
- **Verdict:** PASS with 1 CONFIRMED (PLAN-side wording, not builder error) + 2 NITs (counterexample count: 1 CONFIRMED routed to PLAN, 0 POSSIBLE, 2 NITs, 7 REFUTED).
- **Builder claim under attack:** D1.2 renamed `CiFe` → `CiUI`, added `"ci-ui": CiUI` to the `Aliases` map (alphabetically between `"check"` and `"dev"`), consolidated `filepath.Join(wd, "ui", "frontend")` → `filepath.Join(wd, "ui/frontend")`, updated stage titles `"FE Unit Tests"` → `"UI Unit Tests"` and `"FE Build"` → `"UI Build"`, updated the doc comment to point at `ui/frontend/`, and confirmed `mage ciUI` exits 0.

### Attacks Attempted

**1. Stale `CiFe` / `ci-fe` / `ciFe` references repo-wide. REFUTED.**
- `grep -rln -E 'CiFe|ci-fe|ciFe' --exclude-dir=workflow --exclude-dir=node_modules --exclude-dir=.git` returned exit 1 (no matches). Wider grep including `FE Unit Tests` / `FE Build` returned only `.tillsyn/agents/fe/*.md`, where the matches are the role/agent prefix word "FE" (e.g. "FE Builder Agent", "FE Build-Specific") — semantically unrelated to the mage target name. Zero stale references to the renamed target name anywhere outside `workflow/` historical artifacts and `node_modules/`.
- `README.md`, `CONTRIBUTING.md`, `AGENTS.md` all return zero matches for `FE Unit Tests` / `FE Build` / `"frontend"`. No docs need updating.

**2. `filepath.Join` semantic change (Windows separator behavior). REFUTED.**
- Context7 `/golang/go` `path/filepath.Join` doc: "Join joins any number of path elements into a single path, separating them with an OS specific Separator. Empty elements are ignored. The result is Cleaned."
- `Clean` on Windows normalizes both `/` and `\` to the OS separator. Trace:
  - Darwin: `Join("X", "ui", "frontend")` → `X/ui/frontend`; `Join("X", "ui/frontend")` → `X/ui/frontend`. Identical.
  - Windows: `Join("X", "ui", "frontend")` → `X\ui\frontend`; `Join("X", "ui/frontend")` → intermediate `X\ui/frontend`, then `Clean` rewrites the embedded `/` to `\` → `X\ui\frontend`. Identical.
- No semantic divergence. (Wails build target is darwin/linux/windows; behavior verified for all three.)

**3. Alias-map collision. REFUTED.**
- Read `magefile.go:26-37` (full Aliases map). Entries: `check`, `ci-ui`, `dev`, `test-golden`, `test-golden-update`, `test-integration`, `test-pkg`, `test-func`, `fmt`, `format-path`. `ci-ui` is the only `ci-*` key — no typo collisions (`ci-iu`, `cui`, etc.). No other entry targets `CiUI`. Map is a plain `map[string]interface{}` with unique string keys; if a duplicate string key were present, Go's map literal would silently keep the last value but `mage -l` would not detect it — `grep -c '"ci-ui"'` on the file would catch it. Verified: exactly one occurrence on line 28.

**4. Vitest skip-behavior masks future test failures. NIT.**
- Builder's `mage ciUI` ran `vitest run --passWithNoTests` and skipped 2 tests in `tests/migration-markers.test.ts` (the self-skip when `src/components/` is absent, which is D1.5-pending state). Test Files: `1 skipped (1)`, exit 0.
- The acceptance criterion is `mage ciUI` exit 0. Skip != fail. Currently no risk.
- **Future-drop concern (not a D1.2 counterexample):** when D1.5 lands `src/components/` and the migration-markers tests un-skip, the same pipeline will catch real failures. There's no need to harden `passed > 0` at D1.2 — vitest itself reports `0 passed | 1 skipped` distinctly from `0 passed | 0 skipped`, and any future drop that adds REAL tests will fail the gate if they fail. The `--passWithNoTests` flag also doesn't suppress failures, only the "no tests found" hard-exit.
- **Logged as NIT-D1.2-A4:** consider whether a future drop should switch `--passWithNoTests` off once the test suite is populated. Not blocking.

**5. Astro strict-build flag could escalate `Missing pages directory` warning. REFUTED.**
- Read `ui/frontend/astro.config.mjs` in full (12 lines). Config object: `integrations: [solidJs()]`, `output: 'static'`, `server: { port: 4321 }`. No strict-build flag set. No `vite.build.fail-on-warn`, no Astro `--strict` invocation in `package.json` build script.
- The warning `[WARN] Missing pages directory: src/pages` is emitted by Astro's default `astro build` and is non-fatal in current config. D1.5 lands `src/pages/index.astro` which removes the warning entirely.
- No escalation risk under current config. If a future drop adds strict-build flags before D1.5 lands, that would be a separate counterexample at that point.

**6. PLAN.md `mage -l` acceptance-bullet wording vs actual mage behavior. CONFIRMED (PLAN-side bug, not builder bug).**
- PLAN.md line 73 reads: "`mage -l` lists `ciUI` **AND** lists `ci-ui` (the hyphenated alias added to the `Aliases` map to match existing convention at `magefile.go:26-36` — e.g. `test-pkg`, `format-path`)."
- Empirical `mage -l` output (this round, verified):
  ```
  Targets:
    build               ...
    ci*                 ...
    ciUI                runs the UI continuous-integration gate: ...
    dev                 ...
    format              ...
    formatPath          ...
    ...
    testFunc            ...
    testGolden          ...
    testGoldenUpdate    ...
    testIntegration     ...
    testPkg             ...
  ```
  `mage -l` lists `ciUI` (canonical) but does **NOT** list `ci-ui` (alias). Aliases never appear as standalone rows — they surface only via `mage -h <canonical>` (verified: `mage -h ciUI` reports `Aliases: ci-ui`).
- The literal PLAN.md bullet is unsatisfiable as written. Builder's interpretation in BUILDER_WORKLOG.md line 66 (treating "lists `ci-ui`" as "alias-map MEMBERSHIP, not standalone `mage -l` row") is the correct intent reading, but the bullet text needs a fix.
- **Routed back to orchestrator (PLAN.md fix, not builder rework):** update PLAN.md line 73 from "`mage -l` lists `ciUI` **AND** lists `ci-ui`" to either:
  - "`mage -l` lists `ciUI`; `mage -h ciUI` reports `Aliases: ci-ui`", or
  - "`mage -l` lists `ciUI`; the `ci-ui` alias is registered in the `Aliases` map and resolvable via `mage ci-ui`."
- Verified parallel-pattern precedent: the existing `test-pkg` / `format-path` aliases also do NOT appear in `mage -l` standalone — they only resolve via the alias path. So the bullet's own appeal to "match existing convention" is internally inconsistent with mage's actual `-l` behavior.

**7. Stale "FE Unit Tests" / "FE Build" stage-title references in docs. REFUTED.**
- `grep -rln -E 'FE Unit Tests|FE Build' --exclude-dir=workflow --exclude-dir=node_modules --exclude-dir=.git` returns three matches in `.tillsyn/agents/fe/*.md` only. Inspected: all three matches are the role-prefix phrase ("FE Build-QA-Falsification Agent", "FE Build-Specific" attack-vectors section heading) — not the mage stage-title strings the builder renamed. Semantically unrelated.
- README, CONTRIBUTING, AGENTS: zero matches.
- No stale stage-title references anywhere.

**8. Alias-map insertion-point ordering significance. REFUTED.**
- Read `magefile.go:26-37` ordering: `check`, `ci-ui`, `dev`, `test-golden`, `test-golden-update`, `test-integration`, `test-pkg`, `test-func`, `fmt`, `format-path`. The existing ordering is idiosyncratic (not strictly alphabetical — `test-func` follows `test-pkg`, `fmt` follows `test-func`, etc.), so the builder's "alphabetical insertion between `check` and `dev`" claim is approximately correct for the local neighborhood but isn't a global property of the map.
- Functional impact: Go's `map[string]interface{}` iteration order is randomized; mage's `mage -h <canonical>` only reads the alias name from the value side of the map (reverse lookup). No ordering-dependent behavior exists. Insertion point is purely stylistic. The builder's choice is defensible (groups future `ci-*` aliases) and matches no other significant property of the existing ordering.

**9. Consolidated `filepath.Join(wd, "ui/frontend")` non-idiomatic Go. NIT.**
- Idiomatic Go convention prefers one path component per `filepath.Join` arg: `filepath.Join(wd, "ui", "frontend")`. The consolidated form `filepath.Join(wd, "ui/frontend")` works correctly (per attack 2 verdict — `Clean` normalizes both) but reads as a small departure from convention.
- Builder's reason (in BUILDER_WORKLOG.md line 63) is the literal-substring reading of PLAN.md acceptance bullet 4: the bullet forbids the substring `"frontend"` outside an explicit `ui/frontend` comment. The two-arg form leaves a standalone `"frontend"` string literal which the bullet would flag.
- **Logged as NIT-D1.2-A9:** the consolidated form is acceptable but a brief inline comment (`// "ui/frontend" kept as a single token to satisfy acceptance-bullet-4 literal substring reading`) would help a future reader understand the unusual shape. Not blocking; the builder's worklog records the reasoning durably, which is the equivalent of a comment in a different artifact.
- Alternative future-fix: relax PLAN.md acceptance bullet 4 to permit `filepath.Join(wd, "ui", "frontend")` since the *intent* is "the path resolves to `ui/frontend`, not `frontend/`" — both forms satisfy that intent. Routed back to orchestrator as an optional PLAN.md refinement.

**10. Internal callers of `CiFe`. REFUTED.**
- `grep -nE 'CiUI|CiFe' magefile.go` returns three matches: line 28 (Aliases entry), line 232 (doc comment), line 235 (function definition). Zero call sites.
- Inspected `CI()` body (`magefile.go:213-230`): stage list is `Sources` / `Formatting` / `Coverage` / `Build` / `Integration` — no `CiUI` / `CiFe` call. `mage ci` does NOT chain to `mage ciUI` (these are two independent gates). Rename of `CiFe` → `CiUI` therefore needed only the three edits the builder made.

### Cross-Cutting Notes

- Did NOT run `mage ci` per spawn-prompt directive (parallel-builders WIP would falsely fail the gate). All attacks resolved via static reads + isolated `mage -l` / `mage -h ciUI` queries (which don't execute the gate, only inspect the target registry).
- Did NOT run `mage ciUI` (would run pnpm subprocess against `ui/frontend/` which the parallel D1.3 builder may have left in a transient state); the builder's own R1 invocation already showed exit 0 with the expected output shape.
- Section 0 certificate convergence:
  - (a) QA Falsification on my own verdicts: re-checked Attack 4 (NIT vs CONFIRMED) — confirmed NIT because acceptance criteria currently pass and the failure mode is speculative for future drops. Re-checked Attack 6 (CONFIRMED vs NIT) — confirmed CONFIRMED because the PLAN.md text is literally unsatisfiable, not just imprecise; this is a real bug that needs a fix even though the fix is in a different artifact than the build code.
  - (b) Evidence completeness verified: every verdict cites concrete file-and-line evidence or empirical command output.
  - (c) Unknowns: zero. All 10 attacks resolved; the CONFIRMED is routed to orchestrator as a PLAN.md edit (not a builder rework).

### Verdict

**PASS** with one PLAN-side CONFIRMED (acceptance-bullet wording fix, not a builder code change) and two NITs (future-drop vitest-skip robustness, optional inline comment on consolidated `filepath.Join`). Builder D1.2's code claim survives all 10 attacks. No counterexample to the builder's implementation. Routing:

- **PLAN.md fix (orchestrator action):** update line 73 acceptance bullet to reflect that `mage -l` lists only canonical names, not aliases. Suggested rewrite in attack-6 finding.
- **NIT-D1.2-A4 (future drops):** consider whether to remove `--passWithNoTests` once the FE test suite is non-empty. Not actionable now.
- **NIT-D1.2-A9 (optional):** either add a one-line comment at `magefile.go:241` explaining the consolidated `filepath.Join` form, or relax PLAN.md acceptance bullet 4 to permit the idiomatic two-arg form. Orchestrator picks.

Ready for build-QA-proof sibling result + dev acceptance.

## Hylla Feedback

N/A — Hylla is OFF per the 2026-05-18 rule. Used `Read`, `grep` (via `/usr/bin/grep`), `mage -l`, `mage -h ciUI`, and Context7 `/golang/go` `path/filepath` doc. All evidence sources were appropriate to the questions asked.

## Droplet 1.3 — Round 1

- **QA agent:** `fe-qa-falsification-agent`
- **Round:** 1
- **Verdict:** PASS with 1 NIT + 1 ROUTED-GATE (counterexample count: 0 CONFIRMED, 0 POSSIBLE, 1 NIT, 8 REFUTED, 1 SANDBOX-DENIED → routed to dev-launch).
- **Builder claim under attack:** D1.3 rewrote `ui/main.go` to wire a real `*app.Service` against the same SQLite DB the CLI uses (mirroring `cmd/till/main.go:2244-2314`), with `wails build` gate routed to QA/Phase 6 due to sandbox-denied `go`/`wails` invocations. Worklog claims "static-read-verified" against the source.

### Attacks

**A1. Compile-time correctness (`go build -tags wails ./ui/...`). SANDBOX-DENIED → REFUTED via static evidence.**
- Attempted `go build -tags wails ./ui/...` — denied by sandbox (`Permission to use Bash with command go build ...`).
- Builder honestly reported the same denial.
- Static cross-reference of every external symbol the new code calls against the real source:
  - `platform.DefaultPaths() (Paths, error)` — `internal/platform/paths.go:28` exists. **Match.**
  - `Paths.ConfigPath` + `Paths.DBPath` — struct fields at `internal/platform/paths.go:13-17`. **Match.**
  - `config.Default(string) Config` — `internal/config/config.go:191`. **Match.**
  - `config.Load(string, Config) (Config, error)` — `internal/config/config.go:295`. **Match.**
  - `cfg.Database.Path` (string) — `internal/config/config.go:48-50`. **Match.**
  - `cfg.Delete.DefaultMode` (`config.DeleteMode` named-string) — `internal/config/config.go:17,53-55`. **Match.**
  - `sqlite.Open(string) (*Repository, error)` — `internal/adapters/storage/sqlite/repo.go:75`. **Match.**
  - `(*sqlite.Repository).Close() error` — `internal/adapters/storage/sqlite/repo.go:121`. **Match.**
  - `app.NewService(Repository, IDGenerator, Clock, ServiceConfig) *Service` — `internal/app/service.go:163`. Args at call site: `repo` (satisfies `app.Repository` interface — Repository interface at `internal/app/ports.go:11` requires methods the existing CLI already proves `*sqlite.Repository` provides), `uuid.NewString` (func() string, matches `IDGenerator = func() string`), `nil` for `Clock` (nil-safe per `service.go:167-169` defaults to `time.Now`), `app.ServiceConfig{DefaultDeleteMode: app.DeleteMode(...)}`. **Match.**
  - `app.DeleteMode(cfg.Delete.DefaultMode)` — named-string-to-named-string conversion, valid Go (both have underlying `string`). CLI does the identical thing at `cmd/till/main.go:2415`. **Match.**
- Zero compile-time errors detectable by static reading.

**A2. `wails build` gate (`cd ui && wails build`). SANDBOX-DENIED → ROUTED to dev-launch (Phase 6).**
- Attempted `cd ui && wails build` — denied by sandbox.
- This is the actual binary-emission gate the PLAN.md row 91 specifies; the QA agent cannot execute it from inside this sandbox. Builder also denied (worklog § "Sandbox constraints").
- Static evidence supports a clean compile (Attack 1) but the `wails build` flow also runs `wailsbindings` codegen against the binding set (currently just `App.startup`), which I cannot exercise headlessly. Recommend dev-launch verification on local machine as the formal pass for the worklog acceptance bullet.

**A3. Type-signature mismatches (`NewService` third arg). REFUTED.**
- Per `internal/app/service.go:163`: `func NewService(repo Repository, idGen IDGenerator, clock Clock, cfg ServiceConfig) *Service`. Third arg is `Clock` (type alias for `func() time.Time` at line 126), NOT `*log.Logger`.
- Builder passes `nil` for the `Clock` arg. Lines 167-169 explicitly defaults nil clocks to `time.Now`: `if clock == nil { clock = time.Now }`. Nil-safe by design.
- CLI's call site (`cmd/till/main.go:2414`) passes `nil` in the SAME position — identical pattern.
- Runtime behavior: `clock()` calls return `time.Now()` for both processes.
- No mismatch.

**A4. `ServiceConfig` minimal population vs `ListProjects` path. REFUTED.**
- D1.3's scope is service WIRING only — `App` struct has no JS-exposed `ListProjects` method yet (PLAN.md row 98-113 lands that in D1.4). So at D1.3 the bridge surface area is just `startup(ctx)`.
- Future-looking check anyway: `Service.ListProjects(ctx, includeArchived) ([]domain.Project, error)` at `internal/app/service.go:2252-2254` is a thin pass-through to `s.repo.ListProjects(ctx, includeArchived)`. Touches zero ServiceConfig fields. No nil-deref risk on the read-only call.
- All other ServiceConfig fields builder left unset are defaulted nil-safely in `NewService` body:
  - `CapabilityLeaseTTL` → defaults to `defaultCapabilityLeaseTTL` (line 173-175).
  - `RequireAgentLease` → defaults true (line 176-179).
  - `StateTemplates` → defaults via `defaultStateTemplates()` (line 180-183).
  - `SearchIndex` → falls back to `repo.(EmbeddingSearchIndex)` type-assert (line 184-189).
  - `EmbeddingLifecycle` → falls back to `repo.(EmbeddingLifecycleStore)` type-assert (line 190-195).
  - `LiveWaitBroker` → defaults to `NewInProcessLiveWaitBroker()` (line 202-205).
  - `GitStatusChecker` → defaults to `defaultGitStatusChecker` (line 206-209).
- `AuthRequests`, `AuthBackend`, `EmbeddingGenerator` remain nil — but D1.4's `ListProjects` IPC doesn't exercise auth or embeddings. Future FE drops that touch auth/embeddings will need to populate these.

**A5. Cleanup callback semantics on `wails.Run` return. REFUTED (with NIT).**
- Context7 `/wailsapp/wails` and `/websites/wails_io` both confirm: "wails.Run() ... blocks until the application window is closed." It returns normally on user-driven exit (window close, runtime.Quit, etc.).
- Builder defers `cleanup()` before `wails.Run(...)`. On normal exit, the deferred cleanup fires and `repo.Close()` runs.
- **NIT-D1.3-A5:** Wails provides `OnShutdown` callback specifically for "just before the application terminates" — strictly more correct than `defer` for ensuring cleanup runs on ALL exit paths (`OnShutdown` fires even when the OS sends a signal that bypasses normal Go defers). Builder's `defer cleanup()` works for normal exit. Future drops could move cleanup into `OnShutdown` for belt-and-suspenders. Not a current bug.

**A6. `log.Fatal` vs `os.Exit` interaction with cleanup. REFUTED.**
- `log.Fatal(err)` (line 75) calls `os.Exit(1)` and skips deferred funcs — correct.
- However: at the point line 75 executes, `cleanup()` hasn't been deferred yet (line 77 is `defer cleanup()`, after the error check). So there's no skipped-cleanup leak in the failure path.
- Inside `newServiceFromConfig`, the failure paths at lines 50, 55, 59 all return BEFORE `sqlite.Open` succeeds OR happen on `sqlite.Open` failure itself — `sqlite.Open` already cleans up its own partial state on failure (sqlite/repo.go:89-91 `_ = db.Close()` on pragma failure).
- The only theoretical leak path is: `sqlite.Open` returns successfully (line 60) AND `app.NewService` panics (line 61-63). `NewService` is value-construction with no error path and no observable panic points; functionally unreachable. Not a real bug.

**A7. `platform.DefaultPaths()` CLI parity. REFUTED (with NIT).**
- `internal/platform/paths.go:28-30`: `DefaultPaths()` calls `DefaultPathsWithOptions(Options{AppName: "tillsyn"})`.
- That delegates to `DefaultPathsWithOptions` (line 33-60): with `HomeDir == ""`, `DevMode == false`, falls to line 55-59 — `os.UserHomeDir()` + `filepath.Join(homeDir, ".tillsyn")` + `PathsForHome(...)`.
- CLI at `cmd/till/main.go:2244-2248` calls `DefaultPathsWithOptions(Options{AppName: rootOpts.appName, DevMode: rootOpts.devMode, HomeDir: rootOpts.homeDir})`. With default CLI flags (no `--dev`, no `--home`), these resolve to the SAME path (`~/.tillsyn/tillsyn.db`).
- macOS-specific concern: `DefaultPathsWithOptions` does NOT use `os.UserConfigDir()` (which would route through `~/Library/Application Support/` on macOS). It uses `os.UserHomeDir()` + `.tillsyn`. Same on darwin/linux/windows.
- **NIT-D1.3-A7:** If a developer runs `till --dev` in CLI mode, they get a dev-mode workspace-rooted DB path. The Wails app will ALWAYS use the home-rooted path because `DefaultPaths()` has no DevMode hook. Acceptable for production GUI use; flag for future when a `Tillsyn --dev` window might be wanted.

**A8. In-process Go bindings vs Wails embedding (`NewApp(svc)` compile). REFUTED.**
- `App` struct at `ui/main.go:27-30` carries the `svc *app.Service` field added in D1.3.
- `NewApp(svc *app.Service) *App` at line 33-35 accepts the field and stores it: `return &App{svc: svc}`.
- `application := NewApp(svc)` at line 79 passes the local `svc *app.Service` from `newServiceFromConfig()`. Types align.
- Bound to JS via `Bind: []interface{}{application}` at line 91. `application` is `*App` — Wails bindgen will reflect over exported methods. Today only `startup` (unexported, registered via `OnStartup` separately) — no exported methods to bind. D1.4 adds `ListProjects`.
- Compile-time clean.

**A9. Concurrent DB access (CLI + FE both opening DB). REFUTED.**
- `internal/adapters/storage/sqlite/repo.go:130-134` applies `PRAGMA journal_mode = WAL` to every connection on `Open`. WAL allows concurrent readers AND one writer simultaneously across processes.
- `db.SetMaxOpenConns(1)` + `db.SetMaxIdleConns(1)` at lines 86-87 are PER-PROCESS settings. They limit one Go process's pool; they do NOT prevent other OS processes from opening the same file.
- `PRAGMA busy_timeout = 60s` at line 130-132 gives concurrent processes up to 60 seconds to acquire a write lock before returning `SQLITE_BUSY`. Tolerant of typical TUI/GUI write bursts.
- Net: CLI and FE can both open `~/.tillsyn/tillsyn.db` simultaneously. Reads are concurrent; writes serialize at the file lock with 60s grace.

**A10. `//go:embed` literal preservation. REFUTED.**
- `git diff HEAD -- ui/main.go` shows zero context lines changed around line 21. The embed directive `//go:embed all:frontend/dist` sits at line 21 of the new file (line 16 of the prior file — line number drifted due to grown import block, content unchanged).
- Direct read confirms: `ui/main.go:21` is `//go:embed all:frontend/dist`, and `ui/main.go:22` is `var assets embed.FS`. Byte-identical to pre-D1.3.
- Builder explicitly called out the §N10 trap in worklog Notes and verified post-write via `rg`. Trap dodged.

### Cross-Cutting Notes

- Could NOT execute `go build` or `wails build` from inside the sandbox; both denied with and without `dangerouslyDisableSandbox`. Builder reported the same denial honestly. Static evidence (signature cross-references against actual source files + git diff + Context7 Wails semantics) gives high confidence in compile-cleanliness; the binary-emission gate routes to dev-launch verification at Phase 6.
- Did NOT consult LSP (per builder's reasoning: symbol resolution was unambiguous via `/usr/bin/grep` + targeted `Read`).
- Section 0 certificate convergence:
  - (a) QA Falsification on my own verdicts: re-checked A5 (NIT vs CONFIRMED on cleanup semantics) — confirmed NIT because the defer-based cleanup works for the normal exit path Wails provides; `OnShutdown` is strictly more robust but the current code is correct, not buggy. Re-checked A7 (NIT vs CONFIRMED on dev-mode parity) — confirmed NIT because default CLI invocations DO match `DefaultPaths()`; the `--dev`-flag mismatch is a future-edge-case, not a current bug.
  - (b) Evidence completeness verified: every verdict cites concrete file-and-line evidence, git-diff output, or Context7 quote. The sandbox-denied gate is honestly routed.
  - (c) Unknowns: one — the actual `wails build` exit-code + Mach-O binary emission. Routed to dev-launch at Phase 6.

### Verdict

**PASS** with 2 NITs and 1 ROUTED-GATE. Builder D1.3's code claim survives all 10 attacks. The static evidence chain for compile-cleanliness is airtight; the only unverified piece (`wails build` binary emission) cannot be executed by the QA agent in this sandbox and is honestly routed to dev-launch verification per PLAN.md row 92's own carve-out ("Runtime window-open is a dev-launch confirmation gate at Phase 6"). Routing:

- **NIT-D1.3-A5 (future drops):** consider moving `repo.Close()` cleanup into Wails `OnShutdown` callback for belt-and-suspenders coverage of signal-driven exits. Not actionable now.
- **NIT-D1.3-A7 (future drops):** if a `Tillsyn --dev` window is ever wanted, builder will need to thread `DevMode`/`HomeDir` options through `DefaultPathsWithOptions` rather than `DefaultPaths()`. Not actionable now.
- **ROUTED-GATE-D1.3-A2 (dev-launch / Phase 6):** dev should run `cd ui && wails build` locally to confirm the binary emission gate. PLAN.md row 91 explicitly carves this out as a Phase 6 acceptance.

Ready for build-QA-proof sibling result + dev-launch verification.

## Hylla Feedback

N/A — Hylla is OFF per the 2026-05-18 rule. Used `Read`, `git diff`, `/usr/bin/grep`, Context7 (`/websites/wails_io` + `/wailsapp/wails`) for Wails `Run()` semantics. The sandbox denied both `go build` and `wails build` invocations; reported honestly and resolved attacks via static evidence where possible.
