# PLAN_QA_PROOF_R2 — drop_fe_1_bootstrap

**Verdict:** `pass` (with 2 info-level NITs; 0 blockers, 0 high, 0 med, 0 low, 2 info)
**Date:** 2026-05-18
**Round:** 2
**Reviewer:** fe-qa-proof-agent
**Working scope:** `workflow/drop_fe_1_bootstrap/PLAN.md` (round-2 revised) + `_BLOCKERS.toml` + `REVISION_BRIEF.md` (round-2 reconciled)
**Round-1 reference:** `PLAN_QA_PROOF.md` (15 findings) + `PLAN_QA_FALSIFICATION.md` (10 findings, 2 CONFIRMED blockers, 4 POSSIBLE, 4 NITs)

The round-2 plan addresses **every dev-approved finding** from round 1 — both proof side and falsification side — without inventing new droplets, new MCP surfaces, or new scope. Droplet count holds at 6. `_BLOCKERS.toml` edges unchanged. Locked decisions L1-L4 still honored. All new acceptance bullets are yes/no-checkable.

The two findings below are info-level NITs surfaced fresh by round 2's new content. Neither blocks Phase 4 dispatch.

---

## 1. Round-1 Finding Closure (Coverage Audit)

### 1.1 Round-1 PROOF findings → round-2 changes

| # | Severity | Round-1 finding | Round-2 change site | Verified |
|---|---|---|---|---|
| F1 | info | `verifySources()` interaction not asserted | D1.1 acceptance L57: `magefile.go verifySources() ... does NOT reference root main.go; ... Verified by grep -n verifySources magefile.go` | ✓ ADDRESSED |
| F2 | high | `//go:embed` directive preservation not asserted | D1.1 L51 + D1.3 L87 BOTH carry `grep -q '^//go:embed all:frontend/dist' ui/main.go` exits 0; §N10 documents the relocation trap | ✓ ADDRESSED |
| F3 | med | "a heading" fuzzy in D1.5 | D1.5 L127: "DOM contains (a) an `<h1>` or `<h2>` element with non-empty text content AND (b) a `<ul>` element ... When the DB is empty: ... literal string `No projects yet`" | ✓ ADDRESSED |
| F4 | high | DevTools probe not QA-agent-executable | D1.4 L99-107 adds `ui/app_test.go` with `TestApp_ListProjects_ReturnsDTOForExistingProject`; gate is `go test -tags wails ./ui/...` exits 0 (line 107) — the preferred option (a) from round 1 | ✓ ADDRESSED |
| F5 | low | Marker requirement phrased disjunctively | D1.5 L118: "MUST contain the line `// MIGRATION TARGET: @hylla/stil-solid` — mandatory per `ui/frontend/tests/migration-markers.test.ts:36-44`, absence fails `pnpm run test:unit`" — no "either/OR" phrasing | ✓ ADDRESSED |
| F6 | low | `Wails CLI v2.12.0` over-pinned | D1.6 L144 drops the version-string match entirely; only matches `[Wails] Dev mode`. §N11 documents | ✓ ADDRESSED |
| F7 | low | `mage -l` permissive OR | D1.2 L70: "`mage -l` lists `ciUI` **AND** lists `ci-ui`" — explicit AND | ✓ ADDRESSED |
| F8 | low | §N8 ASCII diagram inconsistent | §N8 L215-221 redrawn; D1.4 no longer shown as direct unblocker of D1.6; D1.6 ← {D1.2, D1.5} matches `_BLOCKERS.toml` | ✓ ADDRESSED |
| F9 | info | D1.3 "no new file" tightening | D1.3 L82: "If a local helper is added (e.g. `loadConfig()`), it lives as a `func` inside `ui/main.go`, NOT as a new file under `ui/`" | ✓ ADDRESSED |
| F10 | med | A3 `till project list` cross-check missing | D1.5 L128: full "CLI cross-check (A3 verification — F10 proof resolution)" bullet — capture both surfaces, assert set-equal modulo ordering | ✓ ADDRESSED |
| F11 | info | DTO field naming note | §N2 L178-179 records Go-idiomatic capitalized rationale + future-camelCase escape hatch | ✓ ADDRESSED |
| F12 | info | macOS-only path clarification | D1.3 L88 + D1.6 L143 add "macOS-only path; on Linux ... on Windows ...; Cross-platform packaging is out of scope per §N6" | ✓ ADDRESSED |
| F13 | info | L1-L4 locked-decision honor | Carried — L1 (ui/ top-level relocation D1.1), L2 (`@astrojs/solid-js` + `client:idle` §N3), L3 (no `internal/tui/` edits §N6), L4 (in-process `a.svc.ListProjects` D1.4 + §N6 explicit no-MCP) | ✓ CARRIED |
| F14 | info | Out-of-scope hygiene | §N6 list unchanged; no new scope; droplet count = 6 | ✓ CARRIED |
| F15 | info | Cited symbols | Round 2 reuses round-1's validated citations; spot-checks below | ✓ CARRIED |

**15/15 proof findings addressed.**

### 1.2 Round-1 FALSIFICATION findings → round-2 changes

| # | Severity | Round-1 finding | Round-2 change site | Verified |
|---|---|---|---|---|
| F1-fals | POSSIBLE | `//go:embed` acceptance gap | §N10 + D1.1 L51 + D1.3 L87 grep guards (same as proof F2) | ✓ ADDRESSED |
| F2-fals | **CONFIRMED** | pnpm vs npm mismatch, no lockfile | D1.1 L42 adds `"packageManager": "pnpm@9.0.0"`; L44 commits `pnpm-lock.yaml`; L52-55 acceptance verifies pin + lockfile committed + `.gitignore` non-exclusion; §N9 documents decision (pin, not switch); REVISION_BRIEF §1 L12 updated to "Node + pnpm" | ✓ ADDRESSED |
| F3-fals | POSSIBLE | `client:load` vs `client:idle` | D1.5 L119, L123 uses `<ProjectList client:idle />`; §N3 L185 documents reset from `client:load` per FE doctrine | ✓ ADDRESSED |
| F4-fals | POSSIBLE | Inline DTO bloats `ui/main.go` | D1.4 L100 splits `ProjectDTO` into new `ui/types.go` (package main, build-tag wails); acceptance L105, L109 verifies split; §N2 L177 documents resolution | ✓ ADDRESSED |
| F5-fals | **CONFIRMED** | Wails CLI version unpinned | §N11 documents Go-binding pin vs CLI uncontrol; D1.3 L88 + D1.6 L144 drop version-string matches; rely on `wails build` exit 0 + Mach-O `file` check + `[Wails] Dev mode` literal | ✓ ADDRESSED |
| F6-fals | POSSIBLE | `wails build` smoke verification-mode contradiction | D1.3 L88-89 narrows QA-agent path to exit 0 + binary file + Mach-O check; runtime window-open explicitly demoted to Phase 6 dev-launch confirmation (NOT D1.3 acceptance) — single verification mode per role | ✓ ADDRESSED |
| F7-fals | NIT | Cross-drop hygiene with `drop_4b_test_cleanup/` | No round-2 change needed; round-1 verdict (file-disjoint, no overlap) holds | ✓ CARRIED |
| F8-fals | NIT | `gofumpt`-vs-compile mechanism | D1.1 L58: "`mage format` reports no diff against `ui/main.go` after relocation (relocated file remains gofumpt-clean — `magefile.go` `formatCheck` stage scans every `*.go` via `trackedGoFiles()` regardless of build tag)" | ✓ ADDRESSED |
| F9-fals | NIT | 10s timeout cold-cache fragile | D1.6 L144 raises to 30s; §N12 documents | ✓ ADDRESSED |
| F10-fals | NIT | Lockfile lifecycle | D1.1 L54: `git ls-files ui/frontend/pnpm-lock.yaml` returns path (committed); L55: `grep -q 'ui/frontend/pnpm-lock.yaml' .gitignore` exits non-zero | ✓ ADDRESSED |

**10/10 falsification findings addressed (CONFIRMED blockers resolved; POSSIBLE counterexamples mitigated per dev calls; NITs absorbed per `feedback_nits_are_first_class.md`).**

### 1.3 ABSORB NITs spot-check (3 of 8)

- **F1 NIT (`verifySources()`):** D1.1 L57 explicit assertion ✓
- **F6 NIT (CLI version-string drop):** §N11 + D1.6 L144 ✓
- **F9 NIT (helper-no-new-file):** D1.3 L82 ✓

All 3 spot-checked NITs land cleanly.

---

## 2. Round-2 Acceptance Verifiability

Every new acceptance bullet introduced in round 2 is yes/no-checkable from a QA-agent command line:

| Acceptance bullet | Checkability mechanism | Verdict |
|---|---|---|
| `grep -q '^//go:embed all:frontend/dist' ui/main.go` exits 0 | exit code | ✓ |
| `grep -q '"packageManager": "pnpm@' ui/frontend/package.json` exits 0 | exit code | ✓ |
| `git ls-files ui/frontend/pnpm-lock.yaml` returns path | non-empty stdout | ✓ |
| `grep -q 'ui/frontend/pnpm-lock.yaml' .gitignore` exits non-zero | exit code | ✓ |
| `mage format` reports no diff against `ui/main.go` | exit code + no `ui/main.go` in `gofumpt -l` output | ✓ |
| `file ./build/bin/Tillsyn.app/Contents/MacOS/Tillsyn` reports Mach-O binary | stdout substring match on `Mach-O` (well-defined for the `file` command) | ✓ |
| `go test -tags wails ./ui/...` exits 0 | exit code | ✓ |
| `grep -q 'type ProjectDTO struct' ui/types.go` exits 0 + same against `ui/main.go` exits non-zero | exit codes | ✓ |
| 30s timeout with literal `[Wails] Dev mode` substring | timeout + grep | ✓ |
| `TestApp_ListProjects_ReturnsDTOForExistingProject` named exactly + assertions (a)(b)(c)(d) | test runner + table-driven assertions | ✓ |
| `mage -l` lists `ciUI` AND `ci-ui`, NOT `ci-fe` | stdout text match | ✓ |
| CLI cross-check: `till project list` set-equal to rendered `<li>` set modulo ordering | deep-equal-modulo-ordering on two JSON-or-table captures | ✓ |

No new fuzziness introduced. The hedge in D1.4 about "in-memory or temp-file SQLite (builder picks impl; in-memory preferred if accepts it, else `t.TempDir()`-rooted file)" leaves a builder-flight choice but the gate (`go test -tags wails ./ui/...` exits 0) is unambiguous. See §3.2 for an info-level NIT on this hedge.

---

## 3. Findings

### 3.1 F1-R2 — REVISION_BRIEF.md was actually updated; §N9 drift caveat is slightly stale (info)

**Severity:** info
**Description:** §N9 L243-244 reads:
> **Brief-vs-plan drift flag for the orchestrator:** `REVISION_BRIEF.md` §1 still says "Node + npm available". The orchestrator should reconcile this in a follow-up post-round-2...

Verified by Read of `REVISION_BRIEF.md` L12: "Node + pnpm available on dev machine. Plan pins `packageManager: \"pnpm@9.0.0\"` in `frontend/package.json` and commits `pnpm-lock.yaml` per Round 2 PLAN.md §N9 decision." — the brief has been updated to match the round-2 decision. §N9's drift-flag prose is correct-in-spirit (the planner was forbidden from editing the brief) but factually stale: the orch did the reconciliation already.

**Suggested fix:** Optional one-line tightening in §N9 — replace "still says 'Node + npm available'" with "originally said 'Node + npm available' before this round-2 reconciliation; if you read this note in a future drop and the brief has drifted again, treat the dev's 2026-05-18 round-2 decision as the source of truth." Or simply delete the drift-flag paragraph since the reconciliation has already happened. Not a blocker either way.

### 3.2 F2-R2 — D1.4 `ui/app_test.go` SQLite-backing hedge (info)

**Severity:** info
**Description:** D1.4 L106 acceptance for `ui/app_test.go`:
> The test constructs `*app.Service` against an in-memory or temp-file SQLite DB (builder picks impl; in-memory via `sqlite.Open(":memory:")` preferred if `internal/adapters/storage/sqlite` accepts it, else a `t.TempDir()`-rooted file)...

The "preferred if accepts it" hedge is reasonable for a builder-flight choice — but a builder lacking domain familiarity could spend time discovering whether `modernc.org/sqlite` honors `:memory:` via the `sqlite.Open` wrapper. Per `cmd/till/main.go` patterns (cited in the same droplet), the canonical path uses `cfg.Database.Path` against a real file; whether the `sqlite.Open` wrapper accepts the literal `:memory:` is not asserted in PLAN.md.

The end-state gate (`go test -tags wails ./ui/...` exits 0) is unambiguous, so this does NOT block acceptance — the builder picks the impl, the gate verifies success. Worth a hint, not a re-plan.

**Suggested fix:** Optional one-line addition: "If `sqlite.Open(\":memory:\")` is rejected by the adapter, fall back to `t.TempDir()` + `cfg.Database.Path` immediately; do not spend builder cycles probing memory-mode support — the gate doesn't care." Or leave as-is and rely on the builder's judgment + `BUILDER_WORKLOG.md` hindsight.

---

## 4. Cited Symbols + Commands Spot-Check (Round-2 New Additions)

Round 2 reuses round-1's validated citations and adds a handful of new commands. Spot-check:

- `//go:embed all:frontend/dist` — Go spec: directive resolves path relative to source file's directory. After `git mv main.go ui/main.go`, the path `frontend/dist` resolves against `ui/` → `ui/frontend/dist` (the new location of the embed target). Confirmed by Go documentation; §N10 documents the trap correctly.
- `"packageManager": "pnpm@9.0.0"` — npm/Corepack standard field, honored by Node 16.13+. `corepack enable` auto-fetches the pinned pnpm version. Correct syntax.
- `pnpm install` (D1.1 L53) — standard pnpm CLI; produces `pnpm-lock.yaml` on first run.
- `file ./build/bin/Tillsyn.app/Contents/MacOS/Tillsyn` — macOS `file(1)` command. Output for a Wails macOS binary contains the literal `Mach-O` (e.g. `Mach-O 64-bit executable arm64` on Apple Silicon). Substring match is reliable.
- `go test -tags wails ./ui/...` — Go test syntax with build-tag selector; `./ui/...` walks the `ui/` subtree. Correct.
- `mage format` (D1.1 L58) — defined in `magefile.go`; runs `gofumpt` over `trackedGoFiles()`. Correct.
- `grep -n verifySources magefile.go` (D1.1 L57) — standard grep, line-number flag. Correct.
- `[Wails] Dev mode` literal (D1.6 L144) — Wails 2.x runtime emission marker. Documented in Wails 2.x source; QA-agent grep is reliable.

No fabricated or syntactically incorrect commands.

---

## 5. `_BLOCKERS.toml` Integrity Cross-Check

Inline `Blocked by:` bullets in PLAN.md vs `_BLOCKERS.toml` rows:

| Droplet | PLAN.md inline | `_BLOCKERS.toml` row | Match |
|---|---|---|---|
| D1.1 | — | (not listed — D1.1 has no blockers) | ✓ |
| D1.2 | `1.1` | `node="1.2", blocked_by=["1.1"]` | ✓ |
| D1.3 | `1.1` | `node="1.3", blocked_by=["1.1"]` | ✓ |
| D1.4 | `1.3` | `node="1.4", blocked_by=["1.3"]` | ✓ |
| D1.5 | `1.2, 1.4` | `node="1.5", blocked_by=["1.2", "1.4"]` | ✓ |
| D1.6 | `1.2, 1.5` | `node="1.6", blocked_by=["1.2", "1.5"]` | ✓ |

No drift. `_BLOCKERS.toml` reasons mirror the inline rationale in each droplet's `Blocked by:` bullet. **Planner's claim "no `blocked_by` edges changed in round 2" verified.**

---

## 6. Locked-Decision Conformance (Re-Audit)

- **L1 (`ui/` top-level peer to `cmd/`, `internal/`):** D1.1 relocates `main.go`→`ui/main.go`, `wails.json`→`ui/wails.json`, `frontend/`→`ui/frontend/`. ✓
- **L2 (Wails+SolidJS+Astro overlay):** §N3 confirms on-disk `@astrojs/solid-js@^4.4.0` + Astro integration; D1.5 mounts `<ProjectList client:idle />`. ✓
- **L3 (FE coexists with TUI; no replacement this drop):** §N6 explicitly excludes TUI feature parity, deprecation, hiding; no paths declared under `internal/tui/` or `cmd/`. ✓
- **L4 (in-process Go bindings, NOT MCP-from-Wails):** D1.4 L104: `a.svc.ListProjects(a.ctx, false)` — direct in-process Go-method call; §N6 explicitly excludes MCP-over-Wails. ✓

No locked decision violated; no locked decision mis-cited.

---

## 7. Scope-Creep Check

- **Droplet count:** 6 (D1.1, D1.2, D1.3, D1.4, D1.5, D1.6). **Unchanged from round 1.** ✓
- **New files added by round 2:** `ui/types.go` (D1.4) + `ui/app_test.go` (D1.4) — both within D1.4's existing scope, both addressing F4-proof + F4-fals. ✓ Not new droplets.
- **New mage targets:** None beyond round-1's `ciUI` + `uiDev` + `uiBuild`. ✓
- **New MCP surfaces:** None. L4 still excludes MCP-over-Wails. ✓
- **New TUI / auth / write-IPC:** None. ✓

**No scope creep.**

---

## 8. Missing Evidence / Verification Gaps

None blocker-class. Two info-level NITs surfaced in §3 above (REVISION_BRIEF.md drift-flag stale, D1.4 SQLite hedge). Both are post-PASS readability tightenings, not gaps in the plan's evidence chain.

---

## TL;DR

- **T1** — All 15 round-1 PROOF findings (F1-F15) addressed in round-2 plan changes; all 10 round-1 FALSIFICATION findings (F1-fals through F10-fals) addressed including both CONFIRMED blockers (F2-fals pnpm + F5-fals Wails CLI version).
- **T2** — All 3 spot-checked ABSORB NITs (F1, F6, F9) land cleanly; 12 of 12 round-2 §N1-§N12 Notes present, with §N9-§N12 new and content matching dev-approved direction.
- **T3** — Droplet count holds at 6; no new droplets, no new MCP surfaces, no scope creep; `_BLOCKERS.toml` edges unchanged and in sync with PLAN.md inline `Blocked by:` bullets across all 5 edges.
- **T4** — All new acceptance bullets are yes/no-checkable by a headless QA agent (embed grep, `packageManager` grep, `pnpm-lock.yaml` ls-files, `mage format` exit, `file` Mach-O substring, `go test -tags wails` exit, DTO-split greps, 30s `[Wails] Dev mode` match, named-exactly Go test function); cited new commands all syntactically valid.
- **T5** — **Verdict: pass.** 2 findings, both info-level NITs (F1-R2 §N9 drift-flag stale post-brief-update; F2-R2 D1.4 SQLite hedge). Neither blocks Phase 4 dispatch. Locked decisions L1-L4 honored; round-2 plan ready for Phase 4 builder spawn.

## Hylla Feedback

N/A — Hylla is OFF per `feedback_hylla_disabled_for_now.md` (2026-05-18). This is an FE drop; Hylla is Go-only when on. Read / Grep / Glob covered everything this review needed. No Hylla queries attempted.
