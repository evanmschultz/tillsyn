# PLAN_QA_PROOF — drop_fe_1_bootstrap

**Verdict:** `pass` (with 15 findings — 0 blockers, 1 high, 5 med, 6 low, 3 info)
**Date:** 2026-05-18
**Round:** 1
**Reviewer:** fe-qa-proof-agent
**Working scope:** `workflow/drop_fe_1_bootstrap/PLAN.md` + `_BLOCKERS.toml` against `REVISION_BRIEF.md`

The plan is structurally sound. Acceptance criteria A1-A7 from the brief are all mapped to droplets; `blocked_by` wiring is a clean DAG with no cycles; PLAN.md inline bullets and `_BLOCKERS.toml` are in sync; locked decisions L1-L4 are honored; cited symbols (`internal/app/service.go:2252 ListProjects`, `cmd/till/main.go:2314 sqlite.Open`, `cmd/till/main.go:2414 app.NewService`, `magefile.go:234 CiFe`, commits `a9bac6c` + `8d33539`, `frontend/package.json` `@astrojs/solid-js@^4.4.0`) all check out against repo HEAD.

Findings below are clarifications, acceptance tightenings, and one explicit verification-mechanism gap (F2 + F10). None require redesign. Suggested fixes are inline; the orchestrator should hand the planner a brief at Phase 3 with the high + med items folded in.

---

## 1. Acceptance Criteria Coverage (A1-A7 → Droplets)

Mapping each brief §6 criterion to droplet(s):

| Criterion | Mapped Droplet(s) | Verifying acceptance bullet(s) |
|---|---|---|
| **A1** — `wails dev` opens window; `wails build` produces runnable binary | D1.3, D1.6 | D1.3: "`cd ui && wails build` produces a binary; … starts without panic" + "`cd ui && wails dev` connects to the Astro dev server"; D1.6: "`mage uiBuild` exits 0 and produces a binary at `ui/build/bin/Tillsyn.app/...`" |
| **A2** — window renders one Astro page mounting ≥1 SolidJS island; visible content | D1.5 | "the launched window shows: a heading, a `<ul>` with one `<li>` per non-archived project" + "`<div id="...solid-island...">` hydration marker" |
| **A3** — SolidJS calls Wails Go method → `internal/app/*` → SQLite; matches `till project list` output | D1.4, D1.5 | D1.4: "body calls `a.svc.ListProjects(a.ctx, false)` and maps each `domain.Project` to `ProjectDTO`" + "DB has ≥1 non-archived project, the array has length ≥1"; D1.5: "the launched window shows: a `<ul>` with one `<li>` per non-archived project" |
| **A4** — `mage ci` continues green; new targets additive | every droplet | D1.1/D1.2/D1.3/D1.4/D1.5/D1.6 all assert "`mage ci` continues green" / "(build-tag `wails` keeps this file out of the default Go build)" |
| **A5** — no CLI / TUI / MCP modified, deprecated, or duplicated | every droplet via paths + §N6 | Paths declare "No edits inside `cmd/` or `internal/` beyond read-only consumption"; §N6 explicitly excludes TUI parity + MCP-from-Wails + auth + write ops |
| **A6** — `ui/README.md` exists with orientation + run commands | D1.6 | "`ui/README.md` exists and contains at minimum: 'in-process Go bindings', 'read-only this drop', 'see REVISION_BRIEF.md', and the two mage target names" |
| **A7** — `.gitignore` excludes Wails / Node build artifacts | D1.1 | "writes: `.gitignore` (add `ui/build/`, `ui/frontend/node_modules/`, `ui/frontend/dist/`, `ui/frontend/.astro/`; remove now-stale `frontend/*` entries if present)" + "writes: `ui/.gitignore` (Wails-specific artefacts: `build/bin/`, `build/darwin/`, ...)" |

**All 7 criteria mapped.** No criterion is orphaned; no droplet is acceptance-orphan (every droplet contributes to at least one A or is structural prerequisite).

---

## 2. Findings

### 2.1 F1 — `verifySources()` and CI workflow do not reference root `main.go` (info)

**Severity:** info
**Description:** `magefile.go:293` calls `git ls-files --error-unmatch magefile.go cmd/till/main.go cmd/till/main_test.go` — the relocation of root `main.go` → `ui/main.go` does NOT trip this guard. Same for `.github/workflows/ci.yml:50` which references only `cmd/till/main.go`. Verified by reading both files. The planner did not assert this nor flag it as a risk; the path is safe, but the absence of an explicit "`mage ci` continues green because `verifySources()` doesn't reference root `main.go`" assertion in D1.1 makes the green-CI claim less self-evidence.
**Suggested fix:** D1.1 acceptance gains one bullet: "`magefile.go:293 verifySources()` references `cmd/till/main.go`, NOT root `main.go` — confirmed via `rg verifySources magefile.go`; root `main.go` relocation does not break the source-tracking guard."

### 2.2 F2 — `//go:embed all:frontend/dist` directive must remain intact in `ui/main.go` (high)

**Severity:** high
**Description:** Root `main.go` line 16 has `//go:embed all:frontend/dist`. After D1.1 relocates `main.go` → `ui/main.go` and `frontend/` → `ui/frontend/`, the directive `all:frontend/dist` is path-relative to the file's NEW location (`ui/`) — so it correctly resolves to `ui/frontend/dist`. **But only because the `frontend/` tree also moved.** If a future refactor splits these two relocations across droplets, the embed directive silently breaks (Go's `//go:embed` raises a compile error if the path doesn't exist; in `wails` build-tag context this surfaces only when `wails build` is invoked). D1.1 acceptance does not assert the embed directive survives intact. D1.3 rewrites `ui/main.go` and could accidentally remove or alter the directive.
**Suggested fix:** D1.1 acceptance gains: "`grep -q '//go:embed all:frontend/dist' ui/main.go` exits 0 (relocated file preserves the embed directive verbatim)." D1.3 acceptance gains the same assertion to guard against accidental removal during the `NewApp(nil)` → real-service rewrite.

### 2.3 F3 — D1.5 acceptance "a heading" is fuzzy (med)

**Severity:** med
**Description:** D1.5 acceptance bullet: "the launched window shows: a heading, a `<ul>` with one `<li>` per non-archived project". "A heading" is yes/no-checkable in principle but ambiguous — does an `<h1 />` with empty text content count? A `<div>` styled to look like a heading? The empty-state assertion is also good but lacks an HTML-element-level grounding.
**Suggested fix:** Tighten to: "the launched window's DOM contains (a) an `<h1>` or `<h2>` element with non-empty text content AND (b) a `<ul>` element. When DB has projects: the `<ul>` has ≥1 `<li>` children, each containing both the project's `ID` and `Name` as visible text. When DB is empty: the page contains the literal string `No projects yet` (or planner-chosen empty-state copy, declared verbatim in this acceptance)."

### 2.4 F4 — D1.4 DevTools probe is human-or-graphical, not QA-agent-executable (high)

**Severity:** high
**Description:** D1.4 acceptance:
> "`cd ui && wails dev` — opening the dev window, then in the browser DevTools console: `await window.go.main.App.ListProjects()` returns an array."

A FE QA agent in this drop's lifecycle is a Claude subagent with no desktop/GUI access — it cannot launch a Wails dev window and type into DevTools. The acceptance is yes/no-checkable **by a human**, but the spawn contract for Phase 5 build-QA describes a subagent verifying the droplet. This creates an evidence gap: the QA agent's pass verdict will rely on dev-or-orchestrator manual verification rather than the agent's own observation. Same issue at the bottom of the same acceptance: "the same method on `window.go.main.App` (verified by QA agent via `wails dev`...)" — the QA agent cannot actually do this.
**Suggested fix:** Either —
(a) **Preferred:** add a Go-side smoke test inside `ui/` (build-tag `wails_test` or similar) that constructs `*App` against a temp SQLite DB pre-seeded with one project, calls `App.ListProjects()` directly, and asserts `len(result) == 1 && result[0].ID != "" && result[0].Name != ""`. This is QA-agent-executable via `mage test-pkg ./ui` or equivalent.
(b) **Acceptable fallback:** explicitly mark the DevTools probe as "verified by dev or orchestrator manually before Phase 7 closeout; the QA agent verifies via `wails build` exit 0 + the static method existence check `grep -q 'func (a \\*App) ListProjects()' ui/main.go`". This narrows what the QA agent attests vs what the dev attests.

### 2.5 F5 — D1.5 migration-marker requirement is mandatory, not optional (low)

**Severity:** low
**Description:** D1.5 acceptance phrases the marker requirement as "either passes (no marker required on `ProjectList.tsx`) OR `ProjectList.tsx` carries the … marker; the planner picks: **carry the marker**". Verified against `frontend/tests/migration-markers.test.ts:36-46`: when `files.length > 0`, every `.tsx`/`.ts` file in `src/components/` MUST contain `// MIGRATION TARGET: @hylla/stil-solid` — no opt-out. The "either ... OR" phrasing is misleading; the planner's pick (carry the marker) is the ONLY viable path that doesn't fail `pnpm run test:unit`.
**Suggested fix:** Replace the disjunctive phrasing with: "`ui/frontend/src/components/ProjectList.tsx` MUST contain the line `// MIGRATION TARGET: @hylla/stil-solid` (mandatory per `ui/frontend/tests/migration-markers.test.ts:36-44`; absence fails `pnpm run test:unit`). Acceptance: `grep -q 'MIGRATION TARGET: @hylla/stil-solid' ui/frontend/src/components/ProjectList.tsx` exits 0."

### 2.6 F6 — D1.6 `mage uiDev` version-string assertion is over-pinned (low)

**Severity:** low
**Description:** D1.6 acceptance: "verifies stdout shows `Wails CLI v2.12.0` + `[Wails] Dev mode` markers within 10s." The literal `v2.12.0` couples the acceptance to one Wails CLI patch version; if the dev's machine has v2.13.x (current as of 2026-05-18 per Wails release cadence) or v2.12.1, the assertion fails even though wiring is correct.
**Suggested fix:** Loosen to regex `Wails CLI v2\.` (or `Wails CLI v` for full major-version flexibility) and `[Wails] Dev mode` literal. The point of the assertion is "wails dev started successfully", not "we're on this exact patch."

### 2.7 F7 — D1.2 `mage -l` listing assertion uses permissive OR (low)

**Severity:** low
**Description:** D1.2 acceptance: "`mage -l` lists `ci-ui` (or `ciUI`) and does NOT list `ci-fe`". The "or" makes this two acceptances in one — passes if either appears. Existing alias style in `magefile.go:26-36` uses hyphenated aliases (`test-pkg`, `format-path`) consistently; adding `ci-ui` as an alias is the precedent-conforming choice.
**Suggested fix:** Pin both: "`mage -l` lists `ciUI` AND `mage -l` lists `ci-ui` (added to `Aliases` map at `magefile.go:26` to match the existing alias convention). `mage -l` does NOT list `ci-fe` (renamed, not added alongside)."

### 2.8 F8 — N8 blocker-graph ASCII diagram is mildly inconsistent with prose (low)

**Severity:** low
**Description:** §N8 ASCII:
```
D1.1 ──────────────────────┬─▶ D1.2 ──────────┬─▶ D1.5
                            │                  │
                            └─▶ D1.3 ─▶ D1.4 ──┘
                                              └─▶ D1.6
                            │
                            └────────────────────▶ D1.6 (via D1.2 + D1.5)
```
The diagram suggests D1.4 directly unblocks D1.6 (`D1.4 ──┘ └─▶ D1.6`) and that D1.2 ALSO unblocks D1.6 via a separate trailing line. The prose below correctly states "D1.6 waits on D1.2 AND D1.5", and `_BLOCKERS.toml` `node="1.6", blocked_by=["1.2", "1.5"]` is the truth. So D1.6 is NOT directly blocked by D1.4 — only transitively through D1.5. The diagram's `└─▶ D1.6` indented under the D1.4 branch is visually misleading.
**Suggested fix:** Redraw — D1.4 only unblocks D1.5; D1.6 only depends on D1.2 + D1.5:
```
D1.1 ──┬─▶ D1.2 ──┬─▶ D1.5 ──▶ D1.6
       │          │             ▲
       │          └─────────────┘
       └─▶ D1.3 ──▶ D1.4 ──▶ D1.5
```
Or just delete the ASCII and lean on the prose + `_BLOCKERS.toml`.

### 2.9 F9 — D1.3 acceptance "no changes outside `ui/main.go`" is correct but worth tightening (info)

**Severity:** info
**Description:** D1.3 says "May add one local helper inside `ui/main.go` (single file in droplet to keep scope tight); do NOT introduce a new `ui/bridge/` package this drop." Then later: "No changes outside `ui/main.go`. `mage ci` remains green". Both bullets are consistent; the YAGNI rationale in §N2 is sound. NIT: the planner might want to assert that the helper, if added, does not introduce a new file (which would make D1.3 a 2-file droplet).
**Suggested fix:** Optional tightening: "If a local helper is added (e.g. `loadConfig()`), it lives as a `func` inside `ui/main.go`, NOT a new file under `ui/`."

### 2.10 F10 — A3 verifies "matches `till project list` output" — D1.5 acceptance is silent on the cross-check (med)

**Severity:** med
**Description:** Brief A3: "Verified by the rendered list matching `till project list` output (or equivalent CLI surface) against the same DB." D1.5 acceptance specifies "one `<li>` per non-archived project … each rendering `ID` and `Name`" — but does NOT instruct the QA agent to also run `till project list` and confirm row equivalence. This leaves room for the FE to render a different set (e.g. including archived projects, or duplicating rows) and still "pass" the local D1.5 check while failing the brief-level A3.
**Suggested fix:** D1.5 acceptance adds: "Cross-check with CLI: `till project list --no-archived` (or equivalent CLI flag for non-archived listing) produces the SAME set of (ID, Name) pairs as the rendered `<li>` elements. Same ordering not required; same set required. QA-agent path: capture `till project list` JSON output, capture `await window.go.main.App.ListProjects()` JSON output, assert deep-equal modulo ordering."

### 2.11 F11 — D1.4 DTO field naming should match Go conventions (info)

**Severity:** info
**Description:** D1.4 proposes `ProjectDTO struct { ID string; Name string }`. Wails serializes Go structs to JS as `{"ID": "...", "Name": "..."}` by default (capitalized keys, matching exported Go field names). The JS-side TypeScript declaration in `ui/frontend/src/types/wails.d.ts` (D1.5) types this as `Promise<{ ID: string; Name: string }[]>` — capitalized. This is internally consistent and follows Go conventions, BUT it diverges from JS idiom (camelCase). This is a one-drop, no-styling-yet decision — accept-as-is, but worth a note for future drops to revisit (perhaps via Wails struct tags or a JSON marshaller).
**Suggested fix:** No change to acceptance. Optional: §N2 note: "DTO uses Go-idiomatic capitalized field names (`ID`, `Name`); JS side carries these unchanged. Future FE drops may add `json:` tags to camelCase the wire format if the JS surface accumulates enough sprawl to warrant the conversion."

### 2.12 F12 — `wails build` macOS-only acceptance path (info)

**Severity:** info
**Description:** D1.3 + D1.6 both assert `./build/bin/Tillsyn.app/Contents/MacOS/Tillsyn` — this is the macOS bundle path. On Linux: `./build/bin/Tillsyn` (just the binary). On Windows: `.\build\bin\Tillsyn.exe`. The dev's env is Darwin (verified from session env `Platform: darwin`), so the acceptance works for THIS dev's machine; cross-platform packaging is out-of-scope per §N6 anyway.
**Suggested fix:** Optional clarification, not change: D1.3 acceptance gains "(macOS bundle path — dev's env is Darwin; Linux/Windows produce `./build/bin/Tillsyn{,.exe}` instead. Cross-platform packaging out of scope per §N6)."

### 2.13 F13 — Locked-decision conformance: all four honored (info)

**Severity:** info
**Description:** Audit of L1-L4:
- **L1 (`ui/` top-level peer to `cmd/`, `internal/`):** D1.1 relocates to `ui/` ✓
- **L2 (`wails init -n tillsyn-ui -t solidjs` + Astro overlay):** §N3 confirms on-disk `@astrojs/solid-js@^4.4.0` + `solid-js@^1.9.7` + `astro@^5.7.13` (verified at `frontend/package.json`); D1.5 uses standard `<Component client:load />` Astro+Solid island pattern (Context7-confirmed) ✓
- **L3 (FE coexists with TUI; no replacement this drop):** §N6 explicitly excludes TUI feature parity / deprecation / hiding; paths declare "No edits inside `cmd/`" — TUI is at `internal/tui/` and is untouched ✓
- **L4 (in-process Go bindings, no MCP-from-Wails):** D1.4 acceptance shows `a.svc.ListProjects(a.ctx, false)` — direct Go method call against `*app.Service`, in-process; §N6 explicitly excludes "MCP-over-the-Wails-bridge" ✓

No locked decision is violated; no locked decision is mis-cited.

### 2.14 F14 — Out-of-scope discipline: no sneak-ins (info)

**Severity:** info
**Description:** Audit of out-of-scope hard list (brief §7) vs droplet paths/acceptance:
- No FE feature beyond A1-A3 ✓ (only `ListProjects`)
- No production styling/theming/dark mode ✓ (D1.5 says "plain `<ul><li>` render", "simple loading + error states", no CSS file edits)
- No TUI parity / deprecation / hiding ✓ (see F13)
- No cross-platform packaging ✓ (Mac-only paths; see F12)
- No auto-update / telemetry / error reporting ✓ (no IPC methods for these)
- No SolidJS component tests this drop ✓ (D1.5 only references the existing migration-markers test; no new `.test.tsx` files)
- No edits inside `cmd/` or `internal/` ✓ (every droplet's paths declares this)
- No MCP-over-Wails ✓ (see L4 in F13)

Clean. No sneak-ins.

### 2.15 F15 — Cited symbol evidence chain (info)

**Severity:** info
**Description:** Spot-checked planner citations against repo HEAD:
- `internal/app/service.go:2252` — `ListProjects(ctx, includeArchived bool) ([]domain.Project, error)` ✓ (read the file; signature exact match)
- `cmd/till/main.go:2314` — `repo, err := sqlite.Open(cfg.Database.Path)` ✓ (read the file; pattern exact match)
- `cmd/till/main.go:2414` — `svc := app.NewService(repo, uuid.NewString, nil, app.ServiceConfig{...})` ✓ (read the file; construction pattern exact match)
- Commits `a9bac6c` ("feat(fe): wails bootstrap at repo root with build-tag isolation") + `8d33539` ("feat(fe): astro solid scaffold with stil tokens and mage ci-fe") ✓ (verified via `git log --oneline`)
- Root `main.go` line 1 — `//go:build wails` ✓
- Root `wails.json` line 5 — `"frontend:dir": "frontend"` ✓
- `frontend/package.json` — `@astrojs/solid-js@^4.4.0`, `solid-js@^1.9.7`, `astro@^5.7.13` ✓
- `magefile.go:234` — `func CiFe() error` ✓
- `frontend/tests/migration-markers.test.ts:36-44` — component-marker assertion ✓ (read the file)

Every cited symbol exists at the line/path claimed. No fabricated citations.

---

## 3. Missing Evidence / Verification Gaps

- **F2 (high)** — embed-directive preservation not asserted in any droplet's acceptance.
- **F4 (high)** — DevTools probe verification mechanism (human vs QA agent) not made explicit; risks Phase-5 QA agents either skipping the probe or making unfounded claims about JS-runtime behavior they can't observe.
- **F10 (med)** — brief A3's "matches `till project list` output" cross-check has no acceptance bullet — pure D1.5-local rendering acceptance leaves room for set divergence.
- **F1 (info)** — `verifySources()` and `.github/workflows/ci.yml:50` interaction with the root `main.go` relocation is correct but not asserted; readers must reverify on their own.

---

## TL;DR

- **T1** — 7/7 brief acceptance criteria mapped to droplets; no orphan acceptance, no orphan droplet.
- **T2** — `blocked_by` graph is a clean DAG (1.1 → {1.2, 1.3}; 1.3 → 1.4; {1.2, 1.4} → 1.5; {1.2, 1.5} → 1.6); inline PLAN.md bullets match `_BLOCKERS.toml` exactly; no cycles.
- **T3** — locked decisions L1-L4 honored; out-of-scope §7 list clean; no sneak-ins of TUI / auth / write-IPC / MCP / production styling.
- **T4** — every cited symbol (`internal/app/service.go:2252`, `cmd/till/main.go:2314+:2414`, `magefile.go:234`, commits `a9bac6c` + `8d33539`, `frontend/package.json` + `astro.config.mjs`) verified against repo HEAD; no fabricated citations.
- **T5** — **verdict: pass.** 15 findings: 0 blockers, 2 high (F2 embed-directive assertion missing; F4 DevTools-probe verification-mechanism gap), 5 med (F3 heading fuzziness; F10 A3 cross-check missing; F8 ASCII diagram drift; F5 marker requirement; F7 alias listing), 6 low/info. Recommended Phase-3 brief: fold F2, F4, F10, F3, F5 into the planner's revision; F1, F6, F7, F8, F9, F11, F12 are NIT-class but worth absorbing per `feedback_nits_are_first_class.md`.

## Hylla Feedback

N/A — FE drop. Hylla is Go-only today per `feedback_hylla_go_only_today.md` + Hylla currently OFF per `feedback_hylla_disabled_for_now.md`. No FE-side index to provide feedback on. Read / Grep / rg / Context7 covered everything this review needed.
