# Wails v2 E2E Testing Best Practices — Playwright Methodology

Date: 2026-05-22
Audience: Tillsyn + Hylla polyglot-foundation + any Wails-fronted project under `hylla/`
Authoritative as of: late 2025 / 2026
Mirror copy: `hylla/hylla/polyglot-foundation/docs/wails-e2e-playwright-best-practices-2026-05-22.md`

## 1. Bottom Line

**Wails v2 has no canonical e2e harness for the built binary.** The only first-party guidance — surfaced via the Wails v3 e2e docs (dated 2026-02-28) and reaffirmed in `wailsapp/wails Discussion #4205` (July 2025) — is **run `wails dev` and point Playwright at the Wails dev-server port** (default `34115`).

**Critical correction to prior Tillsyn methodology**: pointing Playwright at the Astro standalone dev server (`localhost:51428` in our setup) does NOT work — that port serves the bare Astro pages without the Wails-injected `window.go.main.App` IPC bindings. Agents driving Playwright at 51428 see empty data, falsely-PASSING console-error-free renders, and cannot exercise the actual IPC-dependent UI.

The correct surface is the **Wails AssetServer** at `localhost:34115` (Wails default). `wails dev` proxies the Astro frontend at 51428 to 34115 and injects bindings against the live Go backend at 34115. Both data AND IPC fidelity come for free.

## 2. The IPC Bridge Question

In a Wails v2 app, `window.go.main.App.*` IPC bindings exist only inside the Wails runtime — either the native window OR the dev AssetServer at 34115. They are NOT present in:

- Plain Astro dev server (`pnpm run dev` standalone on 51428).
- Static built frontend served via any non-Wails HTTP server.
- Playwright bundled Chromium navigating directly to 51428.

`wails dev` is the surface that lights up bindings. It:
1. Starts the Go host process with the `wails` build tag.
2. Spawns Astro at `frontend:dev:serverUrl` (51428 in Tillsyn's wails.json).
3. Serves the AssetServer on 34115 (Wails v2 default).
4. Proxies frontend assets from 51428 → 34115 and injects `window.go` into HTML responses at 34115.
5. Opens the native WebView window pointing at 34115.

Playwright (or any browser) hitting 34115 sees: real Astro UI + real Go-backed IPC. Playwright hitting 51428 sees: Astro UI + missing bindings + UI in error / empty-state branches.

## 3. Can Playwright Drive The Built Wails Binary?

Per-platform breakdown:

- **macOS (WKWebView)**: NO. WKWebView has no CDP endpoint. Wails v2 added `OpenInspectorOnStartup` + the `isInspectable` property exposing Safari Web Inspector, but Inspector is GUI-only — not automatable by Playwright, chromedp, or go-rod. A third-party WKWebView WebDriver exists for Tauri ([danielraffel.me 2026-02-14](https://danielraffel.me/2026/02/14/i-built-a-webdriver-for-wkwebview-tauri-apps-on-macos/)) but it is experimental and Tauri-specific.
- **Windows (WebView2)**: YES. Set `WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS=--remote-debugging-port=9222` then `browserType.connectOverCDP()`. Documented at [Playwright WebView2](https://playwright.dev/docs/webview2) and [Microsoft Learn](https://learn.microsoft.com/en-us/microsoft-edge/webview2/how-to/playwright). Wails v2 accepts `AdditionalBrowserArgs: []string{"--remote-debugging-port=9222"}`.
- **Linux (WebKitGTK)**: `WebKitWebDriver` works in principle; no shipped Wails integration.

## 4. Fidelity Gap (Chromium vs WKWebView)

Playwright bundled Chromium ≠ macOS WKWebView. Different engines (Blink vs WebKit2), different font metrics, different gamma/contrast, different `:has()` and CSS-feature timing. Practical leaks: subpixel layout shifts, scrollbar widths, font-fallback chains, occasional CSS-feature edge cases.

Not catastrophic for component logic / a11y / interaction testing. Real risk for pixel-diff visual regressions.

The fidelity gap is the accepted state-of-art across the Wails community. No automatable WKWebView driver ships for macOS as of late 2025. Layered defense:

1. **Playwright vs `wails dev`** — primary harness for component, layout, interaction, a11y.
2. **Manual macOS visual smoke** post-build — launch the built `.app`, take screenshots via AppleScript/`screencapture`, diff against committed baselines. Catches WKWebView-only rendering drift.
3. **Optional Windows CDP smoke lane** — if Win shipping matters, `--remote-debugging-port=9222` + `connectOverCDP` against the built binary.

## 5. Alternatives Ranked

| Approach | Real-Wails UX Fidelity | MCP-Driveable | CI-Friendly | Verdict |
|---|---|---|---|---|
| Playwright vs `wails dev` (port 34115) | Low (Chromium ≠ WKWebView) | Yes | Trivial | **Best practical default** |
| Playwright + CDP vs built binary | High (Win only) | Yes | Hard (signing, headless) | Win-only smoke lane |
| chromedp / go-rod vs Wails | Same CDP constraints as Playwright | Indirect | Same | No advantage over Playwright |
| Appium + `appium-mac2-driver` | XCUITest can't see WKWebView DOM | Possible | Heavy | Useless for content |
| Visual regression (Percy/Chromatic) on built binary | High | No | Manual screenshots only | Pixel-diff layer only |

## 6. Component-Level Testing (Vitest + Stub)

Consensus for unit/component tests in jsdom: stub the IPC bridge as a Vitest global setup.

```ts
// vitest.setup.ts
import { vi } from "vitest";

vi.stubGlobal("go", {
  main: {
    App: {
      ListProjects: vi.fn().mockResolvedValue([
        { ID: "p-1", Name: "Test Project" },
      ]),
      ListActionItems: vi.fn().mockResolvedValue([]),
    },
  },
});
```

No Wails-published mock package exists — everyone hand-rolls. Co-locate the mock with the IPC type definitions so TypeScript flags shape drift at compile time.

## 7. Recommendations For Tillsyn (Active)

1. **All FE Playwright work targets `http://localhost:34115`**, NOT `localhost:51428`. The latter is the Astro-only port without bindings.
2. **`mage uiDev` must be running** before any Playwright walk. `mage uiDev` invokes `wails dev` from `ui/`.
3. **FE agent definitions** (`.claude/agents/ta-fe-*.md`) reference `34115` as the canonical Playwright target.
4. **CLAUDE.md** `fe_dev_port` field carries `localhost:34115` (Wails AssetServer with bindings), with `localhost:51428` noted as the underlying Astro-only port.
5. **Bridge guards** in `ProjectList.tsx` + `useActionItems.ts` stay as defense-in-depth (cheap, document the gap, protect against misconfiguration drift).
6. **Future**: add manual macOS visual-smoke step post-build as a layered defense for WKWebView-only rendering bugs. Lower priority than the methodology fix above.
7. **Future**: add Vitest + `vi.stubGlobal('go', ...)` component-test layer for unit isolation. Separate concern from e2e.

## 8. Recommendations For Hylla / Polyglot-Foundation

The same methodology applies to any Wails-fronted UI under `hylla/`:

- Default Wails v2 AssetServer = port `34115`. Verify per-project via `ui/wails.json` + the project's mage/`wails dev` invocation.
- Never point Playwright at the Astro/Vite/Webpack standalone port — bindings live in the Wails AssetServer only.
- If sharing FE QA agent definitions across Hylla projects, parametrize the Wails port (don't hardcode 34115 — projects may remap).
- Component tests use the Vitest stubGlobal pattern; reuse the same shape across projects for consistency.
- WKWebView fidelity gap is universal — adopt the macOS manual visual-smoke layered defense at project level.

## 9. Sources

- [Wails v3 E2E Testing guide](https://v3alpha.wails.io/guides/e2e-testing/) — first-party canonical, dated 2026-02-28.
- [wailsapp/wails Discussion #4205](https://github.com/wailsapp/wails/discussions/4205) — v2 e2e testing thread (July 2025).
- [wailsapp/wails Issue #4115](https://github.com/wailsapp/wails/issues/4115) — v2 e2e testing pattern.
- [Playwright WebView2 docs](https://playwright.dev/docs/webview2).
- [Microsoft Learn — Playwright + WebView2](https://learn.microsoft.com/en-us/microsoft-edge/webview2/how-to/playwright).
- [WebKit blog — Enabling Web Inspector in apps](https://webkit.org/blog/13936/enabling-the-inspection-of-web-content-in-apps/).
- [Tauri WebDriver docs (Linux/Win only)](https://v2.tauri.app/develop/tests/webdriver/).
- [WKWebView WebDriver for Tauri on macOS — third-party](https://danielraffel.me/2026/02/14/i-built-a-webdriver-for-wkwebview-tauri-apps-on-macos/) — experimental.
- [Appium issue: XCUITest can't see WKWebView](https://github.com/appium/appium-for-mac/issues/15).
- [shinshin86/wails-with-playwright-sample](https://github.com/shinshin86/wails-with-playwright-sample) — stock template, no tests committed.
- [dAppServer/wails-build-action](https://github.com/dAppServer/wails-build-action) — CI build only, no e2e.
- [Vitest globals mocking](https://vitest.dev/guide/mocking/globals).

## 10. Provenance

Research dispatched 2026-05-22 by Tillsyn STEWARD orch after a side-by-side Wails vs Playwright screenshot revealed the empty-state gap. Sources are primary (official Wails docs + GitHub issues/discussions + Playwright + Microsoft Learn). The "shim/fixtures" workaround initially proposed was rejected as it would mask the underlying methodology bug.

## 11. Concrete Agent Update Patterns (Apply Per Project)

These are the exact edit patterns Tillsyn applied to its FE agents + CLAUDE.md + magefile. Replicate the shape in any Hylla project with Wails-fronted FE.

### 11.1 FE Builder Agent — Playwright Section

**Before**:

```md
## Playwright MCP — MANDATORY at 3 Breakpoints

For EVERY FE build droplet before declaring done:
- `browser_navigate http://localhost:51428` (Wails dev server).
- For each breakpoint {375x667, 768x1024, 1280x800}: ...
```

**After**:

```md
## Playwright MCP — MANDATORY at 3 Breakpoints

For EVERY FE build droplet before declaring done:
- Pre-flight: `mage uiDev` (or project equivalent invoking `wails dev`) MUST be
  running. Starts the Wails AssetServer at `http://localhost:34115` with
  `window.go.main.App.*` IPC bindings injected against the live Go backend.
  `http://localhost:51428` is the bare Astro standalone dev server WITHOUT
  bindings — never navigate there for verification.
- `browser_navigate http://localhost:34115` (Wails dev AssetServer).
- For each breakpoint {375x667, 768x1024, 1280x800}: ...
- Rendering-engine fidelity caveat: Playwright bundled Chromium ≠ macOS
  WKWebView in production. Component / layout / a11y / interaction coverage is
  honest; WKWebView-only pixel-diffs are not.
```

### 11.2 FE Build-QA-Proof Agent — Playwright Verification Section

**Before**:

```md
- `browser_navigate http://localhost:51428`
- `browser_console_messages level=error` — MUST be 0.
```

**After**:

```md
- Pre-flight: confirm `mage uiDev` is running.
- `browser_navigate http://localhost:34115` (Wails dev AssetServer).
- `browser_console_messages level=error` — MUST be 0.
- Visible-error verification (not just console): query for
  `[role="alert"], [data-tone="error"]` element count. SolidJS createResource
  catches throws silently — the UI can render an error pill while
  console.error stays clean. If the build claims an error-free UI and you find
  rendered error elements, FAIL.
- If builder navigated to `localhost:51428` instead of `34115` for the
  verification walk, FAIL — the binding-less surface gives false-PASS
  empty-state coverage.
```

### 11.3 FE Build-QA-Falsification Agent — Counterexample Section

**Add** (in addition to the existing attack vectors):

```md
- Pre-flight: confirm `mage uiDev` is running. Canonical Playwright target is
  `http://localhost:34115` (Wails dev AssetServer, IPC bindings injected).
  `localhost:51428` is bindings-less Astro — a build verified there ALONE is
  a critical finding.
- Visible-error attack: query
  `document.querySelectorAll('[role="alert"], [data-tone="error"]').length`.
  SolidJS createResource swallows thrown errors silently — the UI renders an
  error pill while console.error is clean. Builds passing on console-only
  verification can be hiding visible errors.
```

### 11.4 FE Planning Agent — Pre-Plan Surface Reference

**Before**:

```md
- `browser_navigate http://localhost:51428`
```

**After**:

```md
- Pre-flight: confirm `mage uiDev` is running. Canonical Playwright target is
  `http://localhost:34115` (Wails dev AssetServer with `window.go` injected).
  `http://localhost:51428` is the bare Astro standalone WITHOUT bindings —
  never plan against the binding-less surface.
- `browser_navigate http://localhost:34115`
```

### 11.5 Project CLAUDE.md — Playwright MANDATORY Rule

**Before** (single line in hard-rules section):

```md
- Playwright MANDATORY for FE work. Every fe-builder / fe-qa spawn prompt
  MUST require: browser_navigate to http://localhost:51428, ...
```

**After**:

```md
- Playwright MANDATORY for FE work. Every fe-builder / fe-qa spawn prompt
  MUST require: browser_navigate to http://localhost:34115 (Wails dev
  AssetServer with window.go.main.App.* IPC bindings injected — NOT
  localhost:51428, which is the bare Astro standalone without bindings and
  produces false-PASS empty-state coverage), browser_snapshot,
  browser_take_screenshot (fullPage + saved to .playwright-mcp/),
  browser_console_messages level=error (0 errors required), browser_evaluate
  for computed-style token verification, AND a visible-error check via
  [role="alert"], [data-tone="error"] element count (SolidJS createResource
  swallows throws silently — console-error count alone is insufficient).
  mage uiDev (→ wails dev) MUST be running before any browser_navigate.
```

### 11.6 Project CLAUDE.md — Stack Table fe_dev_port Row

**Before**:

```
fe_dev_port,localhost:51428
```

**After**:

```
fe_dev_port,localhost:34115 (Wails AssetServer with window.go IPC bindings —
canonical Playwright target; localhost:51428 is the bare Astro dev server
without bindings)
```

### 11.7 Mage UIDev Doc Comment

**Before** (Go doc comment on the UIDev target):

```go
// UIDev launches the Wails live-development loop from the `ui/` subtree.
// The command is long-running: it starts the Astro dev server on
// `http://localhost:4321`, compiles the Go host with the `wails` build tag,
// opens a native WebView window, ...
```

**After**:

```go
// UIDev launches the Wails live-development loop from the `ui/` subtree.
// The command is long-running: it starts the Astro dev server on
// `http://localhost:51428` (per `ui/wails.json` frontend:dev:serverUrl),
// compiles the Go host with the `wails` build tag, runs the Wails
// AssetServer on `http://localhost:34115` with `window.go.main.App.*` IPC
// bindings injected against the live Go backend, opens a native WebView
// window pointing at 34115, ...
//
// Playwright / FE QA target: `http://localhost:34115` (NOT 51428 — the bare
// Astro standalone has no bindings and silently produces empty-state
// false-PASSES).
```

### 11.8 FE Bridge Guards — Keep As Defense-In-Depth

Code added in earlier Tillsyn drops to short-circuit `window.go`-missing calls:

```ts
if (typeof window === "undefined" || !window.go?.main?.App) {
  return [];
}
```

These were added to make `localhost:51428` not throw a TypeError. With the methodology fix, the canonical Playwright surface (`localhost:34115`) always has `window.go` injected — the guards become dead code on the canonical path.

**Recommendation**: keep them. They are cheap, document the IPC-bridge dependency, and protect against future drift (someone forgetting to run `wails dev`, or pointing automation at the wrong port). Add a code comment noting the canonical surface so future readers understand they are defense-in-depth, not the primary path.

### 11.9 Audit Checklist For A Hylla Project

For each Wails-fronted Hylla project, audit:

1. `**/*.md` for hardcoded references to the bare Astro / Vite / Webpack dev port — replace with the Wails AssetServer port. Verify per-project (default `34115`; some projects may remap via Wails options).
2. FE agent definition files (`*.claude/agents/*-fe-*.md` or equivalent) — apply the patterns in §11.1-§11.4.
3. Project CLAUDE.md / AGENTS.md — apply §11.5-§11.6.
4. magefile.go / build scripts — apply §11.7.
5. FE source code — check for `window.go`-related guards (§11.8) and audit comments + doc strings for stale port references.
6. CI configs — any Playwright invocations need the canonical port + `wails dev` running.
7. Any existing Tillsyn / project-tracker action items referencing the old port — update or annotate with the methodology correction.

## 12. Staged Adoption Plan (Committed E2E + axe-core a11y Gate)

For Wails-fronted projects adopting the ta-style committed Playwright + axe-core a11y CI gate, the rollout is **phased** to measure Wails-specific CI cold-start cost before mandating branch protection.

### 12.1 Phase 1 — Substrate (do first)

Create `ui/frontend/a11y/` with the following 4 files:

- **`package.json`** — devDependencies `@playwright/test`, `@axe-core/playwright`, `axe-core`. Pin versions matching ta's reference for stability.
- **`playwright.config.ts`** — Wails-adapted from ta's template. Key deltas vs ta:
  - `webServer.command: 'mage uiDev'` (project equivalent invoking `wails dev`)
  - `webServer.cwd: '..'` (mage runs from repo root)
  - `webServer.url: 'http://127.0.0.1:34115/'` (Wails AssetServer, NOT bare Astro)
  - `webServer.timeout: 60_000` (60s for `wails dev` cold start; ta uses 30s for HTTP-server startup)
  - `webServer.reuseExistingServer: !process.env.CI`
  - `use.baseURL: 'http://127.0.0.1:34115'`
  - `projects`: mobile-375, tablet-768, desktop-1280 (per ta's pattern)
- **`tests/smoke.spec.ts`** — 1 axe spec for the primary route. Severity filter: `serious + critical` block; `moderate + minor` reported but don't fail.
- **`.gitignore`** — `node_modules/`, `playwright-report/`, `test-results/`, `.playwright/`.

Add a `mage UIA11y` target with env-split:
- CI hard-fail on missing toolchain or violations.
- Local warn-skip when `ui/frontend/a11y/node_modules` absent (print install command).
- Local with toolchain installed: full run, hard-fail on violations.

Run locally, measure cold-start time, report to dev.

### 12.2 Phase 1 Explicitly Does NOT Include

- `.github/workflows/ci.yml` a11y job (defer to Phase 2 after timing measured).
- Branch protection on a new a11y check.
- Vitest + `vi.stubGlobal('go', ...)` component-test layer (separate concern, file when relevant gap surfaces).
- FE QA persona migration to committed-spec-as-floor model (Phase 3).

### 12.3 Phase 2 — CI Gate (after Phase 1 ships clean for 1-2 cascades)

- Add the `a11y` CI job (ta's exact shape, Wails timeout bumped to 60s, Go-build cache pre-warmed).
- Measure actual warm CI run time. If <2 min, mandate via branch protection. If >3 min, narrow scope (1 route smoke instead of full per-route suite) before mandating.

### 12.4 Phase 3 — FE QA Persona Migration

- Migrate FE QA personas from "interactive Playwright MCP per droplet" to "committed specs are the regression floor; Playwright MCP is exploratory for net-new UI only."
- build-qa-proof verifies committed specs ran + axe results.
- build-qa-falsification attacks the committed suite + adds new tests for the surface under attack.

### 12.5 Scope By Project Shape

- **Wails app** (Tillsyn, Hylla polyglot): this plan applies. `wails dev` + 34115 + `window.go` IPC.
- **HTTP-server** (ta): different `webServer.command`, same shape — ta's pattern.
- **Web FE** (future Tillsyn/Hylla web build): same shape, `webServer.command` points at the web binary.
- **Mobile native** (`stil-swift` iOS, future Android): principle carries (live-backend a11y gate is mandatory), tooling does not — Playwright + axe-core don't run on native iOS/Android. Use XCUITest + Accessibility Inspector on iOS; Espresso + UIAutomator + Accessibility Scanner on Android.

### 12.6 Cost + Fidelity Considerations

- **Wails CI cold-start**: estimated 60-90s warm (vs ta's 30-45s). Verify on first deployment before mandating branch protection.
- **macOS WKWebView fidelity gap**: Linux CI Playwright/Chromium gate is honest for structural a11y + IPC contract + computed-style assertions; NOT honest for WKWebView-specific rendering. Manual macOS visual smoke per release tag stays the layered defense.

### 12.7 Anti-Pattern To Avoid During Adoption

When migrating an existing project that already has a `playwright.config.ts` pointing at the bare Astro / Vite dev server with a `stubWailsBridge`-style mock, FIX the config first (retarget `webServer.command` + `baseURL` per §12.1) and DELETE the mock — do not preserve the mock layer alongside the new live-backend config. The §3 anti-pattern (shim/fixtures duplicating the IPC contract) drifts from real Go signatures and produces false-PASSes against a bridgeless mock.
