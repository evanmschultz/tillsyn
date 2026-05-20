# Drop FE 2 — Action-Item Listing (Revision Brief)

**Status:** revision-brief authoring 2026-05-19 (orch-direct survey, research-agent recommendation).
**Drop scope (PROPOSED — locks at dev sign-off):** read-only action-item listing for a selected project. Project click in the existing `ProjectList` navigates to an action-items view that renders the project's action-item tree with kind / state / role / structural-type indicators.
**Out of scope:** write IPC (create / edit / move / archive), filtering / search UI, comment thread display, dispatcher status, MCP-over-Wails, auth flows, design-system polish, production styling, virtualized scrolling.

## 1. Hard Prerequisites

- `drop_fe_1_bootstrap` merged on `main`. The FE shell (`ui/main.go`, `ui/types.go`, `ui/frontend/`) and `mage uiBuild` / `mage uiDev` targets exist (verified: `magefile.go:37-38, 268-289`).
- HEAD on `main`, `git status` clean for `ui/` + `magefile.go`.
- Dev has `wails`, `pnpm`, Node available (carry-over from drop_fe_1).
- Tillsyn SQLite DB at the platform default path holds at least one project with non-trivial action-item children for visual smoke verification. The four candidate projects (live tillsyn project, TILLSYN-OLD, etc.) easily satisfy this.

## 2. Goal

Prove the **read-list-with-navigation** pattern on the FE skeleton landed in drop_fe_1. Adds:

1. A second Wails IPC method (`ListActionItems`) on the `App` struct, returning a domain-projected `ActionItemDTO`.
2. A second Solid component (`ActionItemList.tsx`) that renders a project's action items.
3. A second Astro page (or a single-page state-toggle — planner decides) that mounts the new component.
4. A click affordance on `ProjectList` rows that navigates to the action-item view scoped to the clicked project.

The "navigation" word is load-bearing — drop_fe_1 was a single-page mount. This drop is the first that has more than one view, which surfaces the router-vs-state question (Open Question Q1). We deliberately pick **the smallest workable answer** for this drop and defer routing-library adoption to a later drop if the friction is high enough.

## 3. Why This Candidate (vs The Other Three)

Survey conducted 2026-05-19. Four candidates considered:

- **C1 — Action-item listing (RECOMMENDED).** Builds directly on drop_fe_1's pattern (`Service.ListProjects` → `Service.ListActionItems`). All required read methods already exist on `internal/app.Service` (verified: `service.go:2286, 2541`). Single new DTO. Single new Solid component. Single new IPC method. Adds navigation surface for the first time, which is genuinely new (proves the next pattern) but the smallest viable answer is in-component-state, not a router. Estimated 5-7 droplets.
- **C2 — Dispatcher status pane.** Read-only, conceptually attractive. But `internal/app/dispatcher/dispatcher.go` exposes NO public introspection API — `Dispatcher` interface is `RunOnce` / `Start` / `Stop` only (verified: `dispatcher.go:115-135`). Building C2 requires designing + landing new Go surface inside the dispatcher package FIRST, then the FE on top. That's two cross-cutting drops, not one — Go domain decisions (what status is, how it's published, lock state surfacing) deserve their own planner pass. **Defer until after a dedicated `drop_5_dispatcher_status_api` Go drop lands the introspection surface.**
- **C3 — Editable forms (create project / create action item).** Re-opens the in-process-vs-MCP question for write paths, which is an unlocked architectural decision. Bigger droplet count (form state + validation + error surfaces + bidirectional binding + the architectural decision itself). **Defer until a dedicated discussion item converges on write-path policy.**
- **C4 — MCP-from-Wails.** Explicitly re-litigates Locked Decision L4 from drop_fe_1. Out of scope by bias. **Defer to a dedicated discussion drop with a clear motivation (e.g. "browser target arrives", "Wails-binary-and-web parity needed").**

Bias applied: smaller / builds-on-pattern / defers re-litigation of locked decisions. C1 wins on all three axes.

## 4. Locked Architectural Decisions Carried Forward From drop_fe_1

- **L1-L4 unchanged.** `ui/` top-level, Wails+SolidJS+Astro, coexist with TUI, in-process Go bindings (NOT MCP-from-Wails). This drop adds a second IPC method along the same in-process pattern.
- **L5 (new, proposed for dev sign-off) — Navigation via Solid signal, NOT a router library, for this drop.** Rationale: one project list and one action-item view do not justify a router dependency. If a third view lands and the signal-based navigation becomes awkward, the *next* drop introduces a router (likely `@solidjs/router`) as a focused single-droplet change. Defer the dependency until the friction is visible.

## 5. Pre-MVP Rules In Force (Carried Forward)

- Filesystem-MD only for drop coordination. No per-droplet Tillsyn action items.
- Tillsyn-flow output style + Section 0 SEMI-FORMAL REASONING in every subagent response.
- Single-line conventional commits ≤72 chars.
- FE agents do NOT use Hylla MCP (Hylla is Go-only). Use Read / Grep / Glob, Context7 for Wails / SolidJS / Astro, MDN / CanIUse for browser APIs.
- No closeout MD rollups pre-dogfood.
- Builders edit only the files their droplet declares in `paths`.
- Acceptance criteria are yes/no-verifiable from the running artifact.

## 6. Indicative Scope (Planner Decides Actual Decomposition)

Hypothesis — planner refines.

1. **Go-side IPC + DTO.** Add `ActionItemDTO` to `ui/types.go` (fields: `ID`, `ParentID`, `Title`, `Kind`, `Role`, `StructuralType`, `LifecycleState`, `Position`, plus whatever's needed for child rendering — planner decides what to project). Add `func (a *App) ListActionItems(projectID string) ([]ActionItemDTO, error)` to `ui/main.go` calling `a.svc.ListActionItems(a.ctx, projectID, false)`. Files: `ui/types.go`, `ui/main.go`.
2. **TS ambient declaration update.** Extend `ui/frontend/src/types/wails.d.ts` to include the new `ListActionItems` signature.
3. **Solid component — flat list first.** `ui/frontend/src/components/ActionItemList.tsx` that takes a `projectId` prop, calls `window.go.main.App.ListActionItems(projectId)`, renders flat with `kind` / `state` / `role` / `structural_type` columns. Same SSR guard pattern as `ProjectList.tsx` (`typeof window === 'undefined'` returns empty).
4. **Tree assembly.** Convert the flat list to a tree via `ParentID` linking, render nested. Indentation by depth. Planner decides whether to fold into droplet 3 or split — the tree assembly is itself a small unit of logic worth its own droplet if visual polish is wanted.
5. **Navigation surface.** Add a Solid signal `currentView: 'projects' | 'action-items'` + `selectedProjectId: string | null` in `index.astro` or a small `App.tsx` wrapper. `ProjectList` rows become clickable (button or `<li>` with click handler — accessibility planner decides). Click sets state, `ActionItemList` mounts.
6. **Back navigation.** "Back to projects" affordance on the action-items view. Plain `<button>` for this drop.
7. **(Optional) Dotted-address display.** Compute dotted address FE-side from the tree (parent + position). Skip if it pushes droplet count over 8 — defer to a follow-on.

The planner may collapse 1+2, may split 5+6, may absorb tree assembly into the component. Package-level locking: `ui/` and `ui/frontend/` are likely the two compile/lock units.

## 7. Acceptance Criteria (Drop-Level — Planner Decomposes Per Droplet)

- **A1.** `mage uiBuild` succeeds; `mage uiDev` opens the window and shows the project list.
- **A2.** Clicking a project row navigates to a new view that shows that project's action items.
- **A3.** The action-item view renders the items in tree shape (children indented under parents) with visible `kind` / `state` / `role` indicators. `StructuralType` rendered or not at planner's choice (graceful degradation acceptable).
- **A4.** A "back to projects" affordance returns to the project list view, preserving no client-side state expectation (re-fetch is fine for this drop).
- **A5.** The rendered tree matches `till action_item list --project <id>` (or equivalent CLI surface) for the same DB, on at least one verification project.
- **A6.** `mage ci` continues to pass green. New code does not touch `cmd/`, `internal/`, or any non-`ui/` package except read-only consumption of `internal/app.Service`.
- **A7.** No CLI / TUI / MCP surface is modified.
- **A8.** SSR guard pattern (`typeof window === 'undefined'`) is consistent with `ProjectList.tsx`.
- **A9.** Error surfaces (empty list, fetch error, archived project) render gracefully — no "undefined" leaking to the DOM.

## 8. Out Of Scope (Hard)

- Action-item mutation of any kind (create / move / archive / rename / supersede / reparent / metadata edits).
- Comment thread display, comment posting.
- Filtering, sorting, search, pagination, virtualization.
- Dispatcher state, lock state, auth state, attention items, handoffs.
- Production styling, design tokens, dark mode, `@hylla/stil-solid` migration.
- Cross-window state, multi-project picker, project search.
- Router library adoption (`@solidjs/router` etc.) — explicitly deferred (see §4 L5).
- Tests for SolidJS components — drop_fe_1 deferred this; drop_fe_2 carries the deferral forward.
- Edits inside `cmd/` or `internal/` packages beyond read-only `Service.ListActionItems` consumption.

## 9. Open Questions For The Planner

- **Q1 — Routing-vs-signal navigation shape.** §4 L5 proposes Solid signal in a single page. Confirm vs `@solidjs/router`. Planner reads Context7 (`@solidjs/router`) and Astro's view-transition story; if both views in the same Astro page suffice (likely — Astro renders one `.astro` route that hosts the SolidJS island, which then renders one of N children based on signal state), L5 stands. If multi-page (two Astro pages with full reload between) is more idiomatic for Astro, planner proposes that variant. **Bias: signal-in-single-island, defer router.**
- **Q2 — DTO shape.** Which `domain.ActionItem` fields project into `ActionItemDTO`? Minimum: `ID`, `ParentID`, `Title`, `Kind`, `LifecycleState`, `Position`. Likely additions: `Role`, `StructuralType`. Open: `Owner`, `DropNumber`, `Persistent`, `DevGated`, `Labels`, `Description`, timestamps. Planner picks the smallest set that satisfies A3; later drops widen the projection.
- **Q3 — Tree assembly: server-side or client-side?** `Service.ListActionItems` returns a flat slice (`service.go:2286`). Two options: (a) FE assembles the tree from `ParentID` (one IPC call, FE work), (b) add a server-side helper that returns a nested shape (one IPC call, one new Go function). **Bias: FE assembly** — smaller blast radius, no new Go surface, fits the read-only-mirror pattern.
- **Q4 — Dotted-address display.** `domain.ActionItem` has no `DottedAddress` field (verified: `action_item.go:24-175`). Two options: (a) FE computes from tree position post-assembly, (b) defer to a follow-on drop. **Bias: defer** if droplet count tightens; include if it lands cheap inside the tree-assembly droplet.
- **Q5 — Empty / error / archived state design.** What does the action-item view show when `selectedProjectId` is set but the project has zero action items? When the project is archived? When the IPC call fails? Plain text fallbacks (mirror `ProjectList.tsx`'s "No projects yet" pattern) are sufficient for this drop. Confirm with dev that no styling effort goes into error UX in this drop.
- **Q6 — Accessibility floor.** `<li>` with click handler is not keyboard accessible; `<button>` inside `<li>` is. Planner picks the simpler-still-accessible shape and documents the choice; full a11y audit deferred.
- **Q7 — Vitest / Playwright coverage.** drop_fe_1 deferred component tests entirely. Planner confirms drop_fe_2 ALSO defers, or proposes a minimal smoke (`mage ciUI` already runs Playwright per `magefile.go:237`). **Bias: defer like drop_fe_1.**

## 10. Approximate Size

5-7 droplets, ~250-450 LOC across Go + TSX + Astro + config.

Most "novel" logic concentrates in the tree assembly + navigation signal — both are small. The bulk of the LOC is DTO field projection + plain JSX markup. Plan-QA twins fire against this brief before any builder spawns.

## 11. Risk Profile

- **R1 — Navigation pattern locks in awkwardly.** If §4 L5 (signal-based) feels wrong by droplet 5, planner has the option to escalate to dev for the router decision mid-build. Lower-risk path: ship signal-based; let drop_fe_3 introduce a router if needed.
- **R2 — Tree depth in real data.** Tillsyn projects have deeply-nested action-item trees (drop 4b has 30+ children, some of which are themselves parents). Indentation-only rendering risks horizontal overflow. Mitigation: planner specifies `max-width` + visual depth cap (e.g. visually flatten beyond depth N) OR documents the deferral explicitly.
- **R3 — `ListActionItems` returns a large list.** `Service.ListActionItems(projectID, false)` for the live tillsyn project may return hundreds of items. No pagination, no virtualization. Mitigation: explicit in §8 out-of-scope; verify smoke is "renders without crashing" not "renders fast".
- **R4 — DTO field churn.** If droplet 1 picks a minimal DTO and droplet 3 needs more, droplet 1 reopens. Mitigation: pick a slightly-wider DTO up-front (carry `Role` + `StructuralType` even if not displayed initially); cost is ~5 LOC.
- **R5 — Wails IPC arg passing for `projectID string`.** Verify that Wails serializes a single-string argument over IPC cleanly. Drop_fe_1's `ListProjects` had zero args; this is the first arg-bearing IPC method. Planner reads Wails docs (Context7) for parameter binding semantics and confirms or surfaces.

## 12. Cross-References

- `workflow/example/drops/WORKFLOW.md` — canonical per-drop lifecycle.
- `workflow/drop_fe_1_bootstrap/REVISION_BRIEF.md` — template + lock decisions L1-L4 this drop carries forward.
- `workflow/drop_fe_2_action_item_listing/PLAN.md` — drop's PLAN.md (planner fills the Planner section).
- Evidence sources read for this survey:
  - `ui/main.go` — current bridge pattern.
  - `ui/types.go`, `ui/frontend/src/types/wails.d.ts` — DTO + TS ambient shape.
  - `ui/frontend/src/components/ProjectList.tsx`, `ui/frontend/src/pages/index.astro` — FE shape.
  - `internal/app/service.go` — `ListActionItems` (line 2286), `ListChildActionItems` (line 2541), `GetProject` (line 2262).
  - `internal/domain/action_item.go` — `ActionItem` struct shape (lines 24-175).
  - `internal/app/dispatcher/dispatcher.go` — `Dispatcher` interface (lines 115-135) confirmed NO public introspection API → Candidate 2 deferred.
  - `internal/app/dotted_address.go` — `ResolveDottedAddress` exists (resolve → UUID); no inverse "compute dotted from UUID" helper → Open Question Q4.
  - `magefile.go` — `UIBuild` / `UIDev` / `CiUI` targets (lines 268, 274, 287, 237).
  - `ui/frontend/package.json` — Astro 5.7 + SolidJS 1.9 + Vitest + Playwright present; no router.
