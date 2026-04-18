# DROP_1_75 — KIND_COLLAPSE

**State:** planning
**Blocked by:** —
**Paths (expected):** `internal/domain/kind_*.go`, `internal/domain/kind_*_test.go`, `internal/adapters/storage/sqlite/*kind*`, `internal/adapters/storage/sqlite/seed/*kind*`, `scripts/drops-rewrite.sql`, `templates/**` (templates path deletion), tests referencing non-{`project`,`action_item`} kinds
**Packages (expected):** `internal/domain`, `internal/app`, `internal/adapters/storage/sqlite`, `internal/adapters/server/mcpapi`
**Workflow:** drops/WORKFLOW.md
**Started:** 2026-04-18
**Closed:** —

## Scope

Collapse the `kind_catalog` to exactly two kinds — `project` and `action_item` — and delete every code path related to the `template_libraries` / template-binding subsystem. This unblocks the simpler post-Drop-2 cascade tree where role lives on `metadata.role` rather than being encoded in the kind slug. In-scope: remove non-{`project`,`action_item`} kind definitions from the catalog and seed data, **simplify `scripts/drops-rewrite.sql` to schema-only work** — kind_catalog collapse to `{project, action_item}`, `template_libraries` excision, and a single-statement `UPDATE action_items SET kind='action_item', scope='action_item'` — excise template_libraries CRUD / storage / MCP handlers, update domain tests, update MCP integration tests, update any hard-coded kind fixtures. Out-of-scope: `metadata.role` first-class field (Drop 2), cascade dispatcher (Drop 4+), template-system return (future drop), per-row role hydration (dev DB row-level cleanup on 2026-04-18 left only uniform `task/task` rows, so the legacy PHASE 6 role-hydration + PHASE 7 multi-kind rewrite from the main-branch `drops-rewrite.sql` become dead code for this drop). Verification is `mage ci` green + the dev's `~/.tillsyn/tillsyn.db` migration script re-run cleanly.

## Planner

<Filled by go-planning-agent in Phase 1. Atomic units of work below. Each unit's state is mutated in place by the builder during Phase 4. See drops/WORKFLOW.md § "Phase 1 — Plan" for deliverable rules.>

## Notes

- The existing `drop/1.75/PLAN.md` is the inherited big tillsyn cascade plan from the `main` branch. Do NOT edit it for coordination — it merges back to main unchanged. All per-drop coordination lives here.
- **Pre-drop dev DB cleanup (2026-04-18)**: dev purged the live `~/.tillsyn/tillsyn.db` down to a single project (`tillsyn`) and 115 action_items, all with uniform `kind='task', scope='task'`. Every legacy kind (`build-task`, `qa-check`, `subtask`, `project-setup-phase`) and every non-tillsyn project was deleted. Backups sit at `~/.tillsyn/tillsyn.db.pre-*-purge`. This dramatically narrows what `drops-rewrite.sql` has to do.
- `scripts/drops-rewrite.sql` on `main` currently rewrites every non-project node's `kind` to `drop` (Drop-2 vocabulary) and hydrates 8 role variants from legacy kinds. Drop 1.75's builder **replaces** that script with a schema-only collapse: kind_catalog delete-all-but-`{project, action_item}`, `template_libraries` wipe (cascades `template_*` + `node_contract_snapshots`), one-line `UPDATE action_items SET kind='action_item', scope='action_item'`, drop the empty legacy `tasks` table, two-kind-catalog assertions. PHASE 6 role hydration and PHASE 7 multi-kind rewrite from the old script are deleted.
- Dev manually applies `scripts/drops-rewrite.sql` against `~/.tillsyn/tillsyn.db` at drop end (same pattern as `main/scripts/rename-task-to-actionitem.sql`). Agents never touch the dev DB.
