# DROP_1_75 — KIND_COLLAPSE

**State:** planning
**Blocked by:** —
**Paths (expected):** `internal/domain/kind_*.go`, `internal/domain/kind_*_test.go`, `internal/adapters/storage/sqlite/*kind*`, `internal/adapters/storage/sqlite/seed/*kind*`, `scripts/drops-rewrite.sql`, `templates/**` (templates path deletion), tests referencing non-{`project`,`action_item`} kinds
**Packages (expected):** `internal/domain`, `internal/app`, `internal/adapters/storage/sqlite`, `internal/adapters/server/mcpapi`
**Workflow:** drops/WORKFLOW.md
**Started:** 2026-04-18
**Closed:** —

## Scope

Collapse the `kind_catalog` to exactly two kinds — `project` and `action_item` — and delete every code path related to the `template_libraries` / template-binding subsystem. This unblocks the simpler post-Drop-2 cascade tree where role lives on `metadata.role` rather than being encoded in the kind slug. In-scope: remove non-{`project`,`action_item`} kind definitions from the catalog and seed data, retarget `scripts/drops-rewrite.sql` to rewrite every non-project node to `action_item` (today it targets `drop`), excise template_libraries CRUD / storage / MCP handlers, update domain tests, update MCP integration tests, update any hard-coded kind fixtures. Out-of-scope: `metadata.role` first-class field (Drop 2), cascade dispatcher (Drop 4+), template-system return (future drop). Verification is `mage ci` green + the dev's `~/.tillsyn/tillsyn.db` migration script re-run cleanly.

## Planner

<Filled by go-planning-agent in Phase 1. Atomic units of work below. Each unit's state is mutated in place by the builder during Phase 4. See drops/WORKFLOW.md § "Phase 1 — Plan" for deliverable rules.>

## Notes

- The existing `drop/1.75/PLAN.md` is the inherited big tillsyn cascade plan from the `main` branch. Do NOT edit it for coordination — it merges back to main unchanged. All per-drop coordination lives here.
- `scripts/drops-rewrite.sql` currently rewrites every non-project node's `kind` to `drop`; Drop 1.75's builder must retarget that script to `action_item` **and** ensure its before/after assertions match the new two-kind catalog.
- Dev manually applies `scripts/drops-rewrite.sql` against `~/.tillsyn/tillsyn.db` at drop end (same pattern as `main/scripts/rename-task-to-actionitem.sql`). Agents never touch the dev DB.
