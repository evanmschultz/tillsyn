# PLAN_QA_PROOF — DROP_4c.6.1.W6_FE_SCAFFOLD (Round 1)

**Reviewer:** L2 plan-QA proof agent (FE)
**Subject:** `workflow/drop_4c_6_1/DROP_4c.6.1.W6_FE_SCAFFOLD/PLAN.md` (9-droplet L2 plan for Wave W6 — FE Scaffold)
**Mode:** Filesystem-MD-only (no Tillsyn runtime); Hylla OFF; Context7 ON
**Verdict:** **FAIL** — 1 FF (load-bearing internal inconsistency in `blocked_by` declarations) + 7 NITs

A single load-bearing finding blocks builder dispatch: the PLAN's `KindPayload.children` `blocked_by` list (the manifest that creates the build action items) declares D5/D6/D7/D8 parallel-after-D2+D3, while the per-droplet "Updated blocked_by" prose and `_BLOCKERS.toml` declare them serialized on the shared `fe/frontend/src/lib/wails.ts` file. Both views cannot be true. Without resolution, the plan ships with two contradictory creation contracts and the dispatcher will pick one of them at random, producing concurrent-edit conflicts the planner specifically diagnosed and tried to prevent.

Everything else is sound: Wails v2 layout matches Context7, Astro 5 port 4321 default matches Context7 (`/withastro/docs`), the stil tokens source path `stil/main/src/styles/tokens.css` is the correct source-of-truth path (verified by direct file read; `--carl` on line 15; `Inter` + `Iosevka` in font stacks; `dist/tokens.css` does not exist — stil's `pnpm build:tokens` produces `dist/tokens.json`), the baseline.json `product_extensions.tillsyn.commands` has exactly 4 commands (`new-drop`, `complete-drop`, `handoff`, `comment`) matching the L2 plan's claim, `magefile.go` has `CI()` and no `CiFe()` (correctly identified as NEW), and migration markers are encoded as load-bearing acceptance gates on every component / vim engine file.

---

## 1. Findings (FF — load-bearing)

### FF1 — `KindPayload.children.blocked_by` contradicts the per-droplet "Updated blocked_by" prose and `_BLOCKERS.toml` for D5/D6/D7/D8

**Location:** PLAN.md lines 96–110 (`KindPayload.children`) vs. lines 388 (D5 "Updated blocked_by"), 415–417 (D6 narrative), 453 (D7 declared `blocked_by: W6.D6`), 496 (D8 declared `blocked_by: W6.D7`), and `_BLOCKERS.toml` lines 14–34.

**The contradiction:**

| Droplet | `KindPayload.children` (line 103–106) | Per-droplet section | `_BLOCKERS.toml` |
|---|---|---|---|
| D5 | `["W6.D2", "W6.D3"]` | `W6.D2, W6.D3, W6.D4` (line 388) | `["W6.D2", "W6.D3", "W6.D4"]` |
| D6 | `["W6.D2", "W6.D3"]` | `W6.D5` (line 417) | `["W6.D5"]` |
| D7 | `["W6.D2", "W6.D3"]` | `W6.D6` (line 453) | `["W6.D6"]` |
| D8 | `["W6.D2", "W6.D3"]` | `W6.D7` (line 496) | `["W6.D7"]` |

The `Droplet Graph` block at lines 148–158 also says D4–D8 run in parallel after D2+D3, agreeing with `KindPayload` and disagreeing with the per-droplet sections + `_BLOCKERS.toml`. The `Dispatch Schedule` block at lines 658–675 agrees with the per-droplet sections + `_BLOCKERS.toml` (serial D4→D5→D6→D7→D8).

**Why this is load-bearing, not cosmetic:**

The PLAN's own RiskNotes diagnose the exact problem:

- D5 RiskNotes (line 386): "D5 is parallel to D4 (both blocked_by D2+D3, not by each other). If D4 and D5 dispatch concurrently, D4's `wails.ts` may not be authored yet. **Mitigation**: D5 builder creates its own `listActionItems` wrapper in `wails.ts` (adding to the file created by D4). This requires `wails.ts` to exist. Since D5 is parallel to D4 and both target `fe/frontend/src/lib/wails.ts`, there is a **file-level conflict** between D4 and D5. **Resolution**: D5 is blocked_by D4."
- D6 (line 415): "**Final decision**: chain D5 → D6 → D7 → D8 on `wails.ts` additions to avoid concurrent file edits."
- D8 RiskNotes (line 517): also serially adds settings nav link to `MainLayout.astro` (touched by D4) — same concurrent-edit class of bug.

The CLAUDE.md "File- and package-level blocking" rule and the WIKI's atomic-drop granularity guidance both require explicit `blocked_by` between droplets sharing a file. The planner correctly identified the conflict and authored the serialization in `_BLOCKERS.toml` + the per-droplet sections, then failed to back-port the corrections into the `KindPayload`. The `KindPayload` is what an automated dispatcher (4a/4b) reads to create the build children — if today's dispatcher reads from `KindPayload` (or a future tool emits Tillsyn action items from the JSON manifest), it will create D5/D6/D7/D8 with `blocked_by: ["W6.D2", "W6.D3"]` and dispatch them in parallel, producing the very file-conflict the plan tried to prevent. `_BLOCKERS.toml` is a mirror-of-truth artifact per its own header comment ("Mirrors `Blocked by:` bullets in PLAN.md; PLAN.md is truth") — but PLAN.md has TWO truths.

**Required fix (Round 2):** rewrite the `KindPayload.children` array to match `_BLOCKERS.toml`:

```json
{"kind": "build", "title": "W6.D5 ...", "blocked_by": ["W6.D2", "W6.D3", "W6.D4"]},
{"kind": "build", "title": "W6.D6 ...", "blocked_by": ["W6.D5"]},
{"kind": "build", "title": "W6.D7 ...", "blocked_by": ["W6.D6"]},
{"kind": "build", "title": "W6.D8 ...", "blocked_by": ["W6.D7"]},
```

Also rewrite the `Droplet Graph` block at lines 148–158 to reflect the serial chain, AND the `Dispatch order:` enumeration at lines 160–163 to match the `Dispatch Schedule` at lines 658–675.

**Evidence:** PLAN.md lines 96–110, 148–163, 388, 415–417, 453, 496, 517, 658–675; `_BLOCKERS.toml` lines 14–34; Tillsyn `main/CLAUDE.md` "File- and package-level blocking" rule.

---

## 2. NITs (first-class, address by default per memory rule)

### NIT1 — Spawn-prompt UI-surface count vs PLAN narrative

Spawn prompt says "5 UI surfaces from SKETCH §5.3 covered by D4-D8"; SKETCH §5.3 lists 6 surfaces (project list / project detail+tree / create dialog / dispatcher trigger / spawn output viewer / settings panel). L2 PLAN's Objective line 16 correctly says "Deliver 6 v1 surfaces." D7 covers two surfaces (DispatcherTrigger + SpawnOutputViewer), so 5 droplets correctly cover 6 surfaces. Trace coverage is satisfied — no FF. NIT is purely on the spawn-prompt phrasing; orchestrator may want to fix the spawn-prompt boilerplate for future plan-QA spawns.

**Evidence:** SKETCH.md lines 146–156; PLAN.md line 16; PLAN.md D4 (line 312), D5 (line 359), D6 (line 403), D7 (line 447 — *two* components), D8 (line 490).

### NIT2 — Astro `solid` vs `solidJs` import name in D3 shape_hint

D3 KindPayload shape_hint (line 302) writes `import solid from @astrojs/solid-js; ... integrations: [solid()]`. Context7 `/withastro/docs` shows the canonical import as `solidJs` (`import solidJs from '@astrojs/solid-js'; integrations: [solidJs()]`). Default-import names are builder-discretion, so this is non-binding, but matching the official docs reduces cognitive friction. Suggest updating the shape_hint to `solidJs` for clarity.

**Evidence:** PLAN.md line 302; Context7 `/withastro/docs` "Configure SolidJS in astro.config.mjs".

### NIT3 — D3 `fe/frontend/src/env.d.ts` listed under Paths but never referenced in KindPayload changes

D3 Paths block (line 271) lists `fe/frontend/src/env.d.ts` as NEW, but the `KindPayload.changes` array (lines 299–307) has no entry for it. Either drop it from Paths (Astro CLI may generate it automatically) or add an explicit `KindPayload.changes` entry so the builder treats it as a deliberate authoring step. Minor — builder will likely add it idiomatically — but the L2 contract should be tight.

**Evidence:** PLAN.md line 271 vs. lines 299–307.

### NIT4 — D9 acceptance bullet says "first line of their JSDoc/TS doc comment" but TS doesn't have JSDoc-as-first-line semantics

D9 AC (line 561) requires the migration marker as "the first line of their JSDoc/TS doc comment." TS files use `//` line comments or `/** ... */` block comments; either can host the migration marker. Builders writing `// MIGRATION TARGET: ...` as the first line of the file (above any imports) is the simpler reading and matches the constraint at lines 60–61 ("Every `fe/frontend/src/lib/vim/*.ts` file MUST carry: `// MIGRATION TARGET: github.com/hylla-org/ro-vim`"). Suggest tightening the AC to "as a `//`-style line comment at the top of the file (before any imports)" so build-QA-falsification doesn't get caught chasing a JSDoc-block-vs-line-comment red herring.

**Evidence:** PLAN.md lines 60–61, 332, 350, 351, 377, 424, 561, 596–602.

### NIT5 — D9 RiskNote R1 + R2 leave the `.tillsyn/bindings.json` load mechanism partially open

D9 RiskNotes R1+R2 (lines 574–576) propose three alternatives for loading the local bindings file (Wails IPC `GetBindingsJSON`, HTTP fetch via Astro public dir, runtime fetch with graceful fallback), and end with "**Final v1 decision**: fetch `/.tillsyn-bindings.json` via HTTP." But the AC (line 569) only states "with `.tillsyn/bindings.json` present, command palette exposes 9 commands; with it absent, falls back to 4." The mechanism (HTTP fetch vs IPC) is left implicit. Suggest making the AC mention "loaded via `fetch('/.tillsyn-bindings.json')`; 404 ⇒ baseline-only fallback" so build-QA-proof can verify the exact mechanism. Also: if HTTP fetch is the chosen path, D3 needs to either (a) symlink `<repo>/.tillsyn/bindings.json` → `fe/frontend/public/.tillsyn-bindings.json` or (b) copy it at `wails dev` start. D3's paths block (line 264–273) does NOT include this — orphan acceptance bullet at the D3/D9 boundary.

**Evidence:** PLAN.md lines 264–273, 569, 574–576.

### NIT6 — D2 `RunDispatcher` stub-path leaks into AC

D2 RiskNotes (line 236) says "If `Service` doesn't have a `RunDispatcher` method, stub it as `return fmt.Errorf("dispatcher not yet wired: %w", ErrNotImplemented)` and mark TODO." But D2 AC (line 230) declares `RunDispatcher(actionItemID string) error` as a required exported method. The stub-path is a sensible fallback, but it should be explicit in the AC ("either delegates to `Service.RunDispatcher` or returns a wrapped `ErrNotImplemented` sentinel; this is acceptable v1 wiring") so build-QA-proof has a clear gate.

**Evidence:** PLAN.md lines 230 vs. 236.

### NIT7 — Migration-marker grep validation is acceptance but not a `mage ci-fe` gate

AC5 (line 24) requires every component and vim engine file to carry the migration marker, with validation via `grep -r "MIGRATION TARGET" fe/frontend/src/` (line 36). This is a human/QA-agent check, not part of `mage ci-fe`. For the markers to be load-bearing audit gates (per the spawn prompt's "hard acceptance gates per droplet"), a Vitest test or a tiny mage helper that greps the dir should fail the gate when markers are missing. Suggest adding a one-test guard in D3's `mage ci-fe` (e.g., a `tests/migration-markers.test.ts` that walks `src/components/` + `src/lib/vim/`, asserts the marker substring) — cheap belt-and-suspenders, no Playwright needed.

**Evidence:** PLAN.md lines 24, 36, 60–62 (constraint severity=high); spawn prompt §8.

---

## 3. Proof Checks — Pass Inventory

### 3.1 Migration markers as hard acceptance gates (spawn-prompt §8)

`// MIGRATION TARGET: @hylla/stil-solid` declared on every component file: D4 line 24, 332, 350, 351; D5 line 377; D6 line 424; D7 line 465–466; D8 line 508. `// MIGRATION TARGET: github.com/hylla-org/ro-vim` declared on every vim engine file: D9 line 561 (collectively for `types.ts`, `engine.ts`, `wails-keys.ts`, `palette.ts`). ContextBlocks constraint severity=high (lines 60–62) makes them load-bearing. AC5 (line 24) makes them per-acceptance gates. **PASS.** (Caveat: NIT7 — they are *acceptance* gates but not *CI* gates.)

### 3.2 stil tokens path uses `src/styles/tokens.css` not `dist/tokens.css` (spawn-prompt §9)

D3 KindPayload (line 303) says "verbatim copy of stil/main/src/styles/tokens.css"; AC2 (line 21) says "copied from `stil/main/src/styles/tokens.css`"; ContextBlocks (lines 64–65) say "stil tokens consumed from stil/main/src/styles/tokens.css (the source path). dist/tokens.css does NOT exist pre-build." Verified by direct read: `/Users/evanschultz/Documents/Code/hylla/stil/main/src/styles/tokens.css` exists (5.2K); `--carl: #dd9f57` on line 15; `Inter` + `Iosevka` in `--font-sans` / `--font-mono` (lines 43, 45). `stil/main/package.json` `pnpm build:tokens` script runs `tsx scripts/build-tokens.ts` (script exists at `stil/main/scripts/build-tokens.ts`, 4.4K) — the spawn-prompt §9 hypothesis that build:tokens produces `dist/tokens.json` (not `dist/tokens.css`) is consistent with the stil README pattern. **PASS.**

### 3.3 baseline.json `product_extensions.tillsyn` has exactly 4 commands (spawn-prompt cross-planner)

Verified by direct read of `/Users/evanschultz/Documents/Code/hylla/stil/main/src/bindings/baseline.json` lines 100–109: `tillsyn` block carries `new-drop`, `complete-drop`, `handoff`, `comment` — exactly 4 IDs. PLAN.md D9 (lines 556–558) + REVISION_BRIEF §2.15 (lines 390–405) both restate 4 baseline + 5 local = 9 merged. W5 (L1 PLAN line 547) uses the identical merge semantic for the Go-side TUI dispatcher. **PASS.** Cross-surface consistency: same `j` does next-item, same `:dispatch` works in both surfaces.

### 3.4 L1 deviation justification (spawn-prompt §10)

L1 PLAN (line 555) said "D4-D8 each blocked by D2+D3." L2 PLAN serializes D5→D6→D7→D8 on the shared `wails.ts` file. The deviation IS documented in PLAN.md D5 RiskNotes (line 386) and D6 narrative (line 415), with the rationale "file-level conflict on `fe/frontend/src/lib/wails.ts` between concurrent additions." The CLAUDE.md "File- and package-level blocking" rule mandates serialization for shared-file pairs. Deviation is justified. **PASS — but its representation in `KindPayload` is inconsistent (see FF1).**

### 3.5 `fe/go.mod` `replace` directive standard pattern (spawn-prompt §11)

Context7 `/wailsapp/wails` confirms Wails v2 standard `wails.Run(&options.App{Bind: []interface{}{app}})` plus `//go:embed all:frontend/dist`. The `replace github.com/evanmschultz/tillsyn => ../` directive is a standard Go module pattern (not Wails-specific) used to wire a sub-module to a parent in-tree. Context7 doesn't show the `replace` form explicitly in Wails samples (Wails docs assume single-module projects), but the pattern is idiomatic Go and matches the "separate `fe/go.mod`, no `go.work`, no pnpm workspace" decision (PLAN.md line 67). **PASS.** Builder must still verify the in-tree `internal/app.Service` type signature before writing `fe/app.go` (D2 R2 risk note explicitly calls this out — good).

### 3.6 `wails dev` acceptance bullet + test path (spawn-prompt §12)

L1 PLAN (line 541) acceptance §5.10: "`wails dev` in `fe/` launches Tillsyn desktop app showing project list." L2 PLAN's AC1 (line 20) restates "running `wails dev` from `fe/` starts the Tillsyn desktop app without errors; the project list page is reachable at the Astro dev server URL." Verification path (line 33): "builder runs `wails dev` from `fe/` (manual smoke; noted in BUILDER_WORKLOG)." This is a manual smoke gate, not a `mage ci-fe` gate (Wails dev mode requires Wails binary + browser, not feasible in CI). RiskNote R1 (line 44) explicitly forbids `wails build` as a v1 acceptance target (correct — `//go:embed all:frontend/dist` is compile-time, but dev mode serves from the Astro server). **PASS.**

### 3.7 W5 vs W6 vim merge consistency (spawn-prompt cross-planner)

W5 (L1 PLAN line 547) Go-side: "loads baseline's `product_extensions.tillsyn.commands` (4), merges local's 5 additions; falls back gracefully." W6 (L2 PLAN D9 lines 556–558) TS-side: same semantic — "Total with merge: 9. Without local file: 4." Both surfaces target the same `<project>/.tillsyn/bindings.json` file (REVISION_BRIEF §2.19 line 386) and the same baseline.json. Cross-surface test (REVISION_BRIEF §2.19 line 441): "both surfaces dispatch `:dispatch <action-item-id>` to a handler that invokes Tillsyn's dispatcher Service. Cross-surface consistency." **PASS.**

### 3.8 `_BLOCKERS.toml` mirrors PLAN.md (spawn-prompt §7)

`_BLOCKERS.toml` (37 lines) carries entries for D2/D4/D5/D6/D7/D8/D9 — these are the seven nodes with non-empty `blocked_by`. PLAN.md per-droplet sections agree with `_BLOCKERS.toml`. **PASS — but PLAN.md `KindPayload` disagrees with both (FF1).** Resolution requires editing PLAN.md, not `_BLOCKERS.toml`.

### 3.9 Acyclicity of the `_BLOCKERS.toml`-canonical graph

D1 → D2. D3 independent. D2+D3 → D4. D4 → D5 → D6 → D7 → D8. D3 → D9. D9 parallel to D4–D8. No cycle. Topological sort: (D1, D3), D2, D4, (D5, D9), D6, D7, D8. **PASS.**

### 3.10 PLAN-QA-DISCIPLINE-R1 — NEW-behavior acceptance → test-runner `blocked_by`

Every D4–D8 acceptance bullet declaring "Vitest test for X" requires the Vitest runner, which ships in D3 (`mage ci-fe`). All five droplets carry `blocked_by D3` (canonically per `_BLOCKERS.toml`). D9 Playwright AC tests at `http://localhost:4321`, which requires D3's Astro setup (port 4321). D9 `blocked_by D3` — correct. D1's `wails dev` is NOT a Playwright-test prerequisite (Playwright drives the Astro dev server, not Wails). **PASS — no R1 violation.**

### 3.11 PLAN-QA-DISCIPLINE-R2 — narrative count matches D-list enumeration

L1 PLAN W6 row (line 148): "Wails setup + Astro config + 6 FE pages + Go bindings + stil token integration + vim engine (4 TS files + tests); clearly multi-droplet." This is qualitative, not numeric — no count claim to falsify. L2 PLAN title says "9 droplets" (header line 1 + W6 dispatch section); `KindPayload.children` has exactly 9 entries (D1–D9); per-droplet sections have exactly 9 (D1–D9); `_BLOCKERS.toml` has 7 entries (the 7 non-empty blocked_by nodes — D1 and D3 are roots, no entry needed). Counts internally consistent at 9. **PASS — no R2 violation.**

### 3.12 `mage ci-fe` target added in D3 (spawn-prompt §2)

D3 declares `magefile.go` modification adding `CiFe()` function (KindPayload line 305; Paths line 272; AC line 285–287). Direct read of `magefile.go` confirms `CI()` exists at line 149 and no `CiFe()` exists today — D3 correctly identifies it as NEW. Implementation pattern noted in RiskNotes (line 292): "use the same `runCommand` helper pattern as existing `CI()` to stay idiomatic." Builder will read existing helpers (`captureCommandWithProgress`, `runStage`, `newMagePrinter`) before authoring. **PASS.**

---

## 4. Coverage Map

| SKETCH §5.3 surface | Droplet | Component file(s) |
|---|---|---|
| Project list page | D4 | `ProjectList.tsx` + `projects.astro` |
| Project detail / action item tree | D5 | `ActionItemTree.tsx` + `project-detail.astro` |
| Action item create dialog | D6 | `ActionItemCreateDialog.tsx` |
| Dispatcher trigger button | D7 | `DispatcherTrigger.tsx` |
| Spawn output viewer | D7 | `SpawnOutputViewer.tsx` |
| Settings panel | D8 | `SettingsPanel.tsx` + `settings.astro` |

6 surfaces, 5 droplets (D7 carries two surfaces — same `Cmd+R` semantic of dispatcher action + its output viewer, sensible to co-locate). Trace coverage complete.

---

## 5. Cross-Planner Consistency (W5 ↔ W6)

| Concern | W5 (Go-side TUI) | W6 (TS-side FE) | Consistent? |
|---|---|---|---|
| baseline.json source | `stil/main/src/bindings/baseline.json` | `stil/main/src/bindings/baseline.json` | YES |
| Local extension file | `<project>/.tillsyn/bindings.json` | `<project>/.tillsyn/bindings.json` | YES |
| Merge semantic | ID-based deep merge; local wins | ID-based deep merge; local wins | YES |
| Baseline cmd count | 4 (verified) | 4 (verified) | YES |
| Local cmd count | 5 | 5 | YES |
| Merged cmd count | 9 | 9 | YES |
| Graceful fallback | baseline-only on absent local | baseline-only on absent local | YES |
| Migration target | `github.com/hylla-org/lykta` (KEYBIND-R1) | `github.com/hylla-org/ro-vim` (KEYBIND-R2) | Intentionally different per SKETCH §10 |

No cross-planner inconsistency.

---

## 6. Verdict

**FAIL — Round 2 required.**

Single load-bearing FF: FF1 (`KindPayload.children.blocked_by` inconsistency with `_BLOCKERS.toml` and per-droplet "Updated blocked_by"). The plan has two contradictory creation manifests for D5/D6/D7/D8. Builder dispatch under the wrong manifest produces the concurrent-edit failure the planner already diagnosed.

Round 2 scope:
1. **FF1**: rewrite `KindPayload.children` to match `_BLOCKERS.toml`; rewrite `Droplet Graph` (lines 148–158) and `Dispatch order:` enumeration (lines 160–163) to match `Dispatch Schedule` (lines 658–675). Single source of truth.
2. **NIT1**: spawn-prompt UI-surface count — orchestrator-side; no PLAN.md edit needed.
3. **NIT2**: D3 shape_hint `solid` → `solidJs` to match Context7 canonical import name.
4. **NIT3**: D3 `env.d.ts` — either drop from Paths or add KindPayload entry.
5. **NIT4**: D9 AC migration-marker phrasing — tighten to `//`-style line comment at top of file.
6. **NIT5**: D9 RiskNote — make HTTP-fetch mechanism explicit in AC; either add public-dir copy step to D3 OR confirm D9 owns the copy/symlink.
7. **NIT6**: D2 AC — make `RunDispatcher` stub-path explicit ("delegates to `Service.RunDispatcher` OR returns wrapped `ErrNotImplemented`").
8. **NIT7**: D3 — optional one-line Vitest test guarding migration-marker presence (cheap belt-and-suspenders; promotes the audit gate from human-grep to CI gate).

After Round 2 absorption, this plan is dispatch-ready.
