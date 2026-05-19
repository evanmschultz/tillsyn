# Drop FE 1 — Bootstrap (Revision Brief)

**Status:** revision-brief authoring 2026-05-18 (orch-direct from chat-mode design discussion with dev).
**Drop scope (LOCKED):** stand up the FE lane. Wails + SolidJS + Astro skeleton at `ui/` + one read-only end-to-end view proving in-process Go ↔ JS wiring.
**Out of scope:** anything competing with the existing TUI; auth flow; MCP-over-the-Wails-bridge; write operations from the FE; production styling; performance work; tests for the TUI itself.

## 1. Hard Prerequisites

- HEAD `2124d2c` on `main`. Git clean.
- Hylla is back on per dev 2026-05-18 (irrelevant to this drop — FE agents do not use Hylla; Hylla is Go-only today per `feedback_hylla_go_only_today.md`).
- Dev has the Wails CLI installed (or will install once before build droplets fire). The planner does NOT spawn `wails init` itself — droplets do, and the dev's environment must have `wails` on `$PATH`. **Confirm-once with dev BEFORE Phase 4 builders start.**
- Node + npm available on dev machine (Wails frontend templates use them).
- No existing `ui/` directory in the repo (verified at brief authoring time).

## 2. Goal

Make the first FE drop boring. Bootstrap the four-stack combo (Wails shell + Astro static framing + SolidJS islands + Go bindings) and prove end-to-end wiring with ONE minimal read-only view that pulls real data from the existing Tillsyn SQLite DB via the same `internal/app/*` services the CLI uses. Future FE drops add features on a working skeleton; this drop ships a skeleton.

The "minimal read-only view" is the falsification probe of the wiring choice (Locked Decision #4): if in-process Go bindings can't satisfy a trivial read use case, that surfaces NOW, not after we've built features on top.

## 3. Locked Architectural Decisions (Chat-Mode With Dev, 2026-05-18)

- **L1 — FE location: `ui/` top-level peer to `cmd/`, `internal/`.** Why: Wails wants its own project root with `wails.json` + `frontend/`; forcing it under `cmd/` cross-cuts the existing Go-binary convention.
- **L2 — Wails bootstrap: `wails init -n tillsyn-ui -t solidjs` in `ui/`, then add Astro for static framing of `ui/frontend/`.** Why: Wails has SolidJS as a first-class template; Astro doesn't have a Wails template today, so Astro layers on top of the SolidJS scaffold for the static-framing story.
- **L3 — FE coexists with TUI permanently in the near term.** FE earns replacement on merit; this drop does NOT deprecate, hide, or duplicate any TUI surface. The CLI stays the scripting surface unconditionally.
- **L4 — In-process Go bindings, NOT MCP-from-Wails.** Wails Go-side calls the existing `internal/app/*` services directly in-process. Same binary, same process, same DB handle. Reserve MCP for non-Wails clients (CLI tools, other Claude sessions, eventual web target). This is a Wails-binary-only decision; nothing forecloses an HTTP/MCP surface for browser-target FE later.

## 4. Pre-MVP Rules In Force

- Filesystem-MD only for drop coordination. No per-droplet Tillsyn action items.
- Tillsyn-flow output style + Section 0 SEMI-FORMAL REASONING in every subagent response.
- Single-line conventional commits. ≤72 chars.
- FE agents do NOT use Hylla MCP (Hylla is Go-only today). Use Read / Grep / Glob, Context7 for Wails / SolidJS / Astro library semantics, MDN / CanIUse for browser APIs.
- No closeout MD rollups pre-dogfood (per `feedback_no_closeout_md_pre_dogfood.md`).
- Builders edit only the files their droplet declares in `paths`.
- Acceptance criteria are yes/no-verifiable from the running artifact (`wails dev`, `wails build` produces a binary, the read-only view renders real Tillsyn data).

## 5. Indicative Scope (Planner Decides Actual Decomposition)

A starting hypothesis — the planner refines, splits, or merges as needed.

1. **Skeleton: `wails init` + Astro overlay.** Get a `wails dev` that opens a blank window and `wails build` that produces a binary. Files: `ui/wails.json`, `ui/main.go`, `ui/app.go`, `ui/frontend/*`, `ui/go.mod` (or workspace integration), `.gitignore` additions.
2. **Astro pages + SolidJS island wiring.** One Astro page with one SolidJS island that mounts in the Wails window. Files: `ui/frontend/src/pages/index.astro`, `ui/frontend/src/components/ProjectList.solid.tsx` (or equivalent name), Astro config tweaks.
3. **Go-side bridge to `internal/app/*` services.** Expose a single read-only method on the Wails App struct that calls `Service.ListProjects` (or similar existing reader) and returns a JSON-marshallable result type. Files: `ui/app.go` (Wails App methods), small bridge package if the planner deems it warranted.
4. **End-to-end read view.** SolidJS island calls the Wails-exposed Go method, renders the project list (id + name) in a plain HTML list. No styling beyond defaults. Files: SolidJS component + Astro page.
5. **Build target glue.** A `mage fe-dev` / `mage fe-build` target wrapping `wails dev` / `wails build` for parity with existing build discipline. Files: `magefile.go`.
6. **README + `.gitignore`.** `ui/README.md` with one-paragraph orientation + run commands. `.gitignore` for Wails build artifacts (`ui/build/`, `ui/frontend/node_modules/`, `ui/frontend/dist/`).

The planner may collapse 1+2 or split 3+4. The planner decides droplet boundaries based on package-level locking (`ui/`, `ui/frontend/`, `magefile.go` may each be a compile/lock unit) and atomic-droplet sizing.

## 6. Acceptance Criteria (Drop-Level — Planner Decomposes Per Droplet)

- **A1.** `cd ui && wails dev` opens a window; `cd ui && wails build` produces a binary that runs.
- **A2.** The window renders one Astro page that mounts at least one SolidJS island. Verified by visible content in the dev window.
- **A3.** The SolidJS island calls a Wails-exposed Go method, which calls the existing `internal/app/*` service layer, which reads from the existing Tillsyn DB. Verified by the rendered list matching `till project list` output (or equivalent CLI surface) against the same DB.
- **A4.** `mage ci` continues to pass green on `main` — the FE work doesn't break existing Go gates. New `mage fe-*` targets are additive.
- **A5.** No CLI / TUI / MCP surface is modified, deprecated, or duplicated by this drop.
- **A6.** `ui/README.md` exists with one-paragraph orientation + run commands.
- **A7.** `.gitignore` excludes Wails / Node build artifacts.

## 7. Out Of Scope (Hard)

- Any FE feature beyond A1-A3 (no detail views, no nav, no filtering, no editing, no auth, no MCP).
- Production styling, design system, theming, dark mode.
- TUI feature parity, TUI deprecation, TUI hiding, TUI competition.
- Cross-platform packaging (`.dmg` / `.exe` / `.AppImage`).
- Auto-update, telemetry, error reporting.
- Tests for SolidJS components — Phase 5 build-QA verifies acceptance criteria by running the artifact, not by adding component unit tests in this bootstrap drop. Test infrastructure lands in a later drop if/when it makes sense.
- Edits inside `cmd/` or `internal/` packages, except read-only consumption of `internal/app/*` services via the bridge. The bridge package may live under `ui/` (planner decides).
- MCP-over-the-Wails-bridge (Locked Decision L4 — explicitly excluded).

## 8. Open Questions For The Planner

- **Q1 — Go workspace integration.** Should `ui/` be its own Go module (`ui/go.mod`) or part of the existing `go.mod` at repo root via Wails' embedded-Go pattern? Planner inspects `wails init` output and chooses the smaller-blast-radius option. Default: same module if Wails accepts it; separate sub-module only if Wails refuses.
- **Q2 — Bridge package location.** Bridge between Wails App struct and `internal/app/*` services: directly in `ui/app.go`, or a thin `ui/bridge/` package? Planner picks based on idiomatic Wails project layout + the one-method scope of this drop.
- **Q3 — Astro + SolidJS integration shape.** Does Astro mount SolidJS islands via the official `@astrojs/solid-js` integration, or does Wails' SolidJS template come with Vite-native SolidJS and we wrap Astro around it? Planner reads Wails SolidJS template structure + Astro docs (Context7), picks the path with the least configuration surgery.
- **Q4 — mage target naming.** `mage fe-dev` / `mage fe-build` vs `mage ui-dev` / `mage ui-build`? Planner picks consistent with existing target naming in `magefile.go`.

## 9. Approximate Size

5-7 droplets, ~300-600 LOC across Go + JS + config + docs. Most of the LOC is config/glue, not novel logic. The "novel" code is the Wails App method + the SolidJS component — both trivial. Plan-QA twins fire against this brief before any builder spawns.

## 10. Cross-References

- `workflow/example/drops/WORKFLOW.md` — canonical per-drop lifecycle.
- `workflow/drop_fe_1_bootstrap/PLAN.md` — drop's PLAN.md (planner fills the Planner section).
- `CLAUDE.md` § Tech Stack — current Go stack.
- Future drops: `drop_fe_2_*/` for the next FE feature once this skeleton lands.
