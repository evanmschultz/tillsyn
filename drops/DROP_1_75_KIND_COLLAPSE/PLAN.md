# DROP_1_75 — KIND_COLLAPSE

**State:** planning
**Blocked by:** —
**Paths (expected):** `internal/domain/kind.go` (new, split from `workitem.go`), `internal/domain/action_item.go` (renamed from `task.go`), `internal/domain/workitem.go` (residual — lifecycle/context/actor/resource enums; rename pending), `internal/domain/*_test.go`, `internal/adapters/storage/sqlite/repo.go`, `internal/adapters/storage/sqlite/schema.go` (or wherever `CREATE TABLE work_items` lives), `internal/adapters/storage/sqlite/*_test.go`, `internal/adapters/server/mcpapi/*` (handlers referencing `projects.kind` / `template_librar*`), `internal/app/*template_librar*`, `internal/app/template_reapply.go`, `internal/app/snapshot.go`, `internal/domain/template_library*.go`, `internal/domain/builtin_template_library.go`, `internal/domain/template_reapply.go`, `cmd/till/template_cli.go`, `cmd/till/template_builtin_cli_test.go`, `scripts/drops-rewrite.sql`, any `work_items` / `WorkKind` string references sitewide (multi-pass `rg` + `sd`), tests referencing non-{`project`,`actionItem`} kinds, tests referencing `projects.kind`
**Packages (expected):** `internal/domain`, `internal/app`, `internal/adapters/storage/sqlite`, `internal/adapters/server/mcpapi`, `cmd/till`, `internal/tui` (only if `template_librar*` or `projects.kind` references surface there)
**Workflow:** drops/WORKFLOW.md
**Started:** 2026-04-18
**Closed:** —

## Scope

Collapse the `kind_catalog` to exactly two kinds — `project` and `actionItem` (camelCase, matching the Go constant `WorkKindActionItem = "actionItem"` at `internal/domain/workitem.go:39` and the live `kind_catalog` row) — delete every code path related to the `template_libraries` / template-binding subsystem, rename stale files + types to match the post-`Task → ActionItem` identifier state, rename the `work_items` table and every in-code `work_items` / `WorkKind` string reference to `action_items` / `Kind` via a codebase-wide `rg` + `sd` pass (same pattern as the shipped `plan_items → action_items` rename), and drop the `projects.kind` column. This unblocks the simpler post-Drop-2 cascade tree where role lives on `metadata.role` rather than being encoded in the kind slug, and it removes the dual-kind-system hazard where Go thinks it persists `actionItem` but SQL still carries a `WorkKind`-indexed column layout.

In-scope:

1. **Kind catalog collapse** — keep only `{project, actionItem}` rows. Delete the `seedDefaultKindCatalog` function at `internal/adapters/storage/sqlite/repo.go:1231-1247` (runs on every DB open and re-seeds 7 legacy kinds). Bake the two remaining rows into the `CREATE TABLE kind_catalog` / initial schema migration so DB open is idempotent without a seed loop.
2. **`work_items` → `action_items` table + identifier rename** — one multi-pass `rg` + `sd` sweep over the whole repo: `work_items` → `action_items` (SQL + strings), `WorkItem` → `ActionItem` where still stale, `WorkKind` → `Kind`, `workitem` package/filename references. Drive with narrow, inspectable regexes per identifier; never a single catch-all pass. Drop the legacy `tasks` compat table and remove `bridgeLegacyActionItemsToWorkItems` (`repo.go:1184-1228`) — the bridge only exists to translate the old `tasks` table into `work_items`, and `tasks` is empty in the dev DB.
3. **File + type renames** — `internal/domain/task.go` → `internal/domain/action_item.go` (file is already `type ActionItem struct{}`, only the filename is stale). Split `internal/domain/workitem.go`: move `WorkKind` (renamed `Kind`) + its constants into `internal/domain/kind.go`; leave lifecycle/actor/context/resource enums behind (renaming `workitem.go` itself is a follow-up drop — out of scope here unless the split forces it).
4. **`template_libraries` excision** — delete every file/symbol under the `template_library*`, `template_reapply`, `TemplateLibrar*`, `node_contract_snapshot*`, `template_node_template*`, `template_child_rule*` surface across `internal/domain`, `internal/app`, `internal/adapters/*`, `cmd/till`, tests, and any TUI references. Grep today returns 44 files. Planner must enumerate the final deletion list and any call-site updates required to keep the rest of the package compiling.
5. **Drop `projects.kind` column** — templates are a user/orch concern, not a system-tracked project attribute. Strip the column from the schema, the `type Project` struct, all MCP project handlers, all project filters, all tests, and the TUI project views. Any `WHERE projects.kind = ?` / `SELECT kind FROM projects` goes.
6. **`scripts/drops-rewrite.sql` = schema-only** — rewritten against the `action_items` (post-rename) table: drop the non-{`project`,`actionItem`} rows from `kind_catalog`, drop all `template_librar*` / `node_contract_snapshot*` / `template_node_template*` / `template_child_rule*` tables, `ALTER TABLE projects DROP COLUMN kind`, drop the empty legacy `tasks` table, run a single `UPDATE action_items SET kind='actionItem', scope='actionItem'` (all 115 live rows are currently `task/task`), and assert `SELECT COUNT(*) FROM kind_catalog = 2`. PHASE 6 role hydration and PHASE 7 multi-kind rewrite from the main-branch script are deleted outright — the dev-DB cleanup on 2026-04-18 removed every row they'd touch. Dev applies this SQL manually to `~/.tillsyn/tillsyn.db` at drop end, same pattern as `scripts/rename-task-to-actionitem.sql`.
7. **Tests + fixtures** — update every domain/app/adapter test, every MCP integration test, every fixture, every golden file that references a non-{`project`,`actionItem`} kind, the `work_items` table, `WorkKind`, or `projects.kind`.

Out-of-scope:

- `metadata.role` first-class field and role hydration (Drop 2; today role stays in description prose).
- Cascade dispatcher + state-trigger engine (Drop 4+).
- Template system return (future drop — user/orch-defined templates, not `kind_catalog`-coupled).
- Per-row role hydration / multi-kind rewrite (dead code for this drop — dev DB row-level cleanup 2026-04-18 left only uniform `task/task` rows across 115 action_items).
- `project_id nullable` and any remaining auth schema changes (Drop 1).
- Renaming `internal/domain/workitem.go` itself (the residual lifecycle/context/actor/resource file) to a less-stale name (defer).

Verification: `mage ci` green from `drop/1.75/`, `gh run watch --exit-status` green on `drop/1.75` branch, and dev re-runs `scripts/drops-rewrite.sql` against `~/.tillsyn/tillsyn.db` cleanly.

## Planner

<Filled by go-planning-agent in Phase 1. Atomic units of work below. Each unit's state is mutated in place by the builder during Phase 4. See drops/WORKFLOW.md § "Phase 1 — Plan" for deliverable rules.>

## Notes

- The existing `drop/1.75/PLAN.md` (repo root, not this file) is the inherited big tillsyn cascade plan from the `main` branch. Do NOT edit it for coordination — it merges back to main unchanged. All per-drop coordination lives here.
- **Pre-drop dev DB cleanup (2026-04-18)**: dev purged the live `~/.tillsyn/tillsyn.db` down to a single project (`tillsyn`) and 115 action_items, all with uniform `kind='task', scope='task'`. Every legacy kind (`build-task`, `qa-check`, `subtask`, `project-setup-phase`) and every non-tillsyn project was deleted. Backups sit at `~/.tillsyn/tillsyn.db.pre-*-purge`. This dramatically narrows what `drops-rewrite.sql` has to do.
- **`__global__` auth project is self-healing.** `internal/adapters/storage/sqlite/repo.go:1455-1473` `ensureGlobalAuthProject` runs on every DB open with `INSERT ... ON CONFLICT(id) DO NOTHING`. Dev deleting the `__global__` row during pre-drop cleanup is fine — it rebuilds on next binary startup. No migration work here.
- **`work_items → action_items` rename uses the `rg` + `sd` pattern the `plan_items → action_items` rename shipped with.** Multi-pass, narrow regex per identifier (`work_items` table name, `WorkItem` struct, `WorkKind` type, `workitem` package/filename refs). Single-catch-all passes are banned — every pass must be inspectable in `git diff` before commit.
- **`seedDefaultKindCatalog` dies.** The function at `repo.go:1231-1247` re-seeds 7 kinds on every DB open, which is the mechanism by which legacy kinds keep materializing. Replacement: bake the two surviving rows (`project`, `actionItem`) into the initial `CREATE TABLE kind_catalog` schema migration so DB open is idempotent without a seed loop.
- **`bridgeLegacyActionItemsToWorkItems` dies with the `tasks` table.** The shim at `repo.go:1184-1228` exists only to translate the long-empty legacy `tasks` table into `work_items`. Dropping the table removes the shim's reason to exist.
- **`projects.kind` column is user/orch-facing metadata, not a system tracking concern.** Projects get template suggestions from the user or orch at creation and across the project's life; the system doesn't need to persist a kind. Column gets dropped along with every Go reference, MCP handler filter, and TUI view that reads it.
- `scripts/drops-rewrite.sql` on `main` currently rewrites every non-project node's `kind` to `drop` (Drop-2 vocabulary) and hydrates 8 role variants from legacy kinds. Drop 1.75's builder **replaces** that script with a schema-only collapse against the post-rename `action_items` table: `kind_catalog` → `{project, actionItem}`, `template_librar*` / `node_contract_snapshot*` / `template_node_template*` / `template_child_rule*` wipe, `ALTER TABLE projects DROP COLUMN kind`, drop legacy `tasks` table, one-line `UPDATE action_items SET kind='actionItem', scope='actionItem'`, assert `kind_catalog` row count = 2. PHASE 6 role hydration and PHASE 7 multi-kind rewrite from the old script are deleted.
- Dev manually applies `scripts/drops-rewrite.sql` against `~/.tillsyn/tillsyn.db` at drop end (same pattern as `scripts/rename-task-to-actionitem.sql`). Agents never touch the dev DB.
- **One drop, not split.** All seven in-scope items above ship in this drop. The planner decomposes into units with `blocked_by` wiring (e.g. table rename unblocks SQL-script update; file rename unblocks any unit touching those files).
