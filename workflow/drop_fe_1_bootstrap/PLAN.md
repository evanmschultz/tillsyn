# DROP_FE_1_BOOTSTRAP — Wails + SolidJS + Astro Skeleton, Minimal Read-Only Proof-Of-Wiring View

**State:** planning
**Blocked by:** —
**Paths (expected):** `ui/` (new top-level dir), root build/glue files (`magefile.go` if a `mage fe-*` target lands, `.gitignore` additions), no edits inside `cmd/` or `internal/` for this drop
**Packages (expected):** `ui/` (Wails project), `ui/frontend/` (Astro + SolidJS), thin Go bridge (location to be set by planner — likely `ui/app.go` or `ui/bridge/`; reads from existing `internal/app/*` services in-process)
**PLAN.md ref:** new parallel FE lane; not numbered into the Go cascade sequence
**Workflow:** `workflow/example/drops/WORKFLOW.md`
**Cascade concept:** `AGENT_CASCADE_DESIGN.md`
**Started:** 2026-05-18
**Closed:** —

## Scope

First drop of the FE lane. Stand up a Wails + SolidJS + Astro desktop-app skeleton at `ui/` top-level, with one end-to-end minimal read-only view that proves in-process Go ↔ JS wiring against the existing Tillsyn SQLite DB via the same `internal/app/*` services the CLI uses. Nothing about this drop competes with the TUI — the FE earns replacement on merit later.

Locked design decisions (orch + dev, 2026-05-18 chat):

1. **Location:** `ui/` top-level peer to `cmd/`, `internal/`.
2. **Bootstrap:** `wails init -n tillsyn-ui -t solidjs` into `ui/`, then add Astro for static framing of `ui/frontend/`.
3. **TUI relationship:** coexist; FE earns replacement long-term.
4. **Wiring:** in-process Go bindings — Wails exposes Go services to JS directly. NOT MCP-from-Wails.

See `REVISION_BRIEF.md` for the full locked-decisions rationale, acceptance criteria, out-of-scope hard list, and open planner questions.

## Planner

<Filled by `fe-planning-agent` in Phase 1. Read `REVISION_BRIEF.md` first.>

## Notes

<Filled by planner if useful.>
