# DROP_1_75 — KIND_COLLAPSE

**State:** planning
**Blocked by:** —
**Paths (expected):** `internal/domain/kind.go` (merge target — existing file absorbs `Kind` type from `workitem.go:35-44`), `internal/domain/action_item.go` (renamed from `task.go`), `internal/domain/workitem.go` (residual — lifecycle/context/actor/resource enums; rename pending; also loses `WorkKind` block at :35-44), `internal/domain/project.go` (strip `Kind` field), `internal/domain/template_library.go`, `internal/domain/template_reapply.go`, `internal/domain/builtin_template_library.go`, `internal/domain/*_test.go`, `internal/adapters/storage/sqlite/repo.go` (schema inline; `CREATE TABLE tasks` at :169, `CREATE TABLE action_items` at :198, `CREATE TABLE kind_catalog` at :316, `seedDefaultKindCatalog` at :1231-1301, `migrateTemplateLifecycle` at :1030-1055 + caller :650, `migratePhaseScopeContract` at :710-789, `bridgeLegacyActionItemsToWorkItems` at :1184-1228, `ALTER TABLE projects ADD COLUMN kind` at :588, 13 `ALTER TABLE tasks` at :592-604, 2 `UPDATE tasks` at :977-978, CREATE INDEXes at :558 :665, backfill helpers at :1057+), `internal/adapters/storage/sqlite/repo_test.go` (legacy tasks fixture at :1006-1049), `internal/adapters/storage/sqlite/template_library_test.go`, `internal/adapters/server/mcpapi/handler.go` (registerTemplateLibraryTools at :86, pickTemplateLibraryService at :1045-1050), `internal/adapters/server/mcpapi/extended_tools.go` (till.bind_project_template_library :2171+, till.get_template_library :2258+, till.upsert_template_library :2281+, till.ensure_builtin_template_library :2085+), `internal/adapters/server/mcpapi/extended_tools_test.go`, `internal/adapters/server/mcpapi/instructions_tool.go`, `internal/adapters/server/mcpapi/instructions_tool_test.go`, `internal/adapters/server/mcpapi/instructions_explainer.go`, `internal/adapters/server/mcpapi/handler_integration_test.go`, `internal/adapters/server/common/mcp_surface.go`, `internal/adapters/server/common/app_service_adapter_mcp.go`, `internal/adapters/server/common/app_service_adapter_mcp_actor_attribution_test.go`, `internal/adapters/server/common/app_service_adapter.go`, `internal/adapters/server/common/app_service_adapter_auth_context.go`, `internal/adapters/server/common/app_service_adapter_auth_context_test.go`, `internal/adapters/server/common/app_service_adapter_lifecycle_test.go`, `internal/adapters/server/httpapi/handler_integration_test.go`, `internal/app/kind_capability.go` (ensureKindCatalogBootstrapped :559-589 + sync.Once field + defaultKindDefinitionInputs :863-874), `internal/app/template_library.go`, `internal/app/template_library_builtin.go`, `internal/app/template_library_builtin_spec.go`, `internal/app/template_library_test.go`, `internal/app/template_contract.go`, `internal/app/template_contract_test.go`, `internal/app/template_reapply.go`, `internal/app/snapshot.go`, `internal/app/snapshot_test.go`, `internal/app/service.go`, `internal/app/service_test.go`, `internal/app/ports.go`, `internal/app/helper_coverage_test.go`, `internal/app/kind_capability_test.go`, `internal/tui/model.go` (9 hard refs: project.Kind at :4856, :18747; WorkKindSubtask at :5190 :5200 :14840 :17905 :19236; WorkKindPhase at :5227 :8957), `internal/tui/model_test.go`, `internal/tui/thread_mode.go`, `internal/tui/thread_mode_test.go`, `cmd/till/template_cli.go`, `cmd/till/template_builtin_cli_test.go`, `cmd/till/project_cli.go`, `cmd/till/project_cli_test.go`, `cmd/till/main.go`, `cmd/till/main_test.go`, `scripts/drops-rewrite.sql`, all `WorkKind` / `WorkItem` / `workitem` Go identifier references sitewide (multi-pass `rg`+`sd`; the `work_items → action_items` table rename already shipped via `scripts/rename-task-to-actionitem.sql`).
**Packages (expected):** `internal/domain`, `internal/app`, `internal/adapters/storage/sqlite`, `internal/adapters/server/mcpapi`, `internal/adapters/server/common`, `internal/adapters/server/httpapi`, `internal/tui`, `cmd/till`.
**Workflow:** drops/WORKFLOW.md
**Started:** 2026-04-18
**Closed:** —

## Scope

Collapse the `kind_catalog` to exactly two kinds — `project` and `actionItem` (camelCase, matching the Go constant `WorkKindActionItem = "actionItem"` at `internal/domain/workitem.go:39` and the live `kind_catalog` row) — delete every code path related to the `template_libraries` / template-binding subsystem, rename stale Go identifiers to match the post-`Task → ActionItem` state (the `work_items → action_items` **table** rename already shipped via `scripts/rename-task-to-actionitem.sql` on 2026-04-18 — only Go identifier renames remain: `WorkKind → Kind`, `WorkItem*` stale refs, `workitem` filename/package refs), drop the `projects.kind` column, and excise the legacy `tasks` compat table. This unblocks the simpler post-Drop-2 cascade tree where role lives on `metadata.role` rather than being encoded in the kind slug, and it removes the dual-kind-system hazard where Go persists `actionItem` but SQL still carries a `WorkKind`-indexed column layout.

In-scope:

1. **Kind catalog collapse** — keep only `{project, actionItem}` rows. Delete `seedDefaultKindCatalog` at `internal/adapters/storage/sqlite/repo.go:1231-1301` (the full range — `:1231-1247` is the seed-record table, `:1264-1272` is the `INSERT OR IGNORE`, `:1274-1299` is the merge/upsert block). Delete the parallel app-layer seeder `ensureKindCatalogBootstrapped` at `internal/app/kind_capability.go:559-589` plus its `sync.Once` struct field and its input source `defaultKindDefinitionInputs` at `:863-874`. Bake the two remaining rows into `CREATE TABLE kind_catalog` (`repo.go:316`) as inline `INSERT` statements so DB open is idempotent without any seed loop.

2. **Go identifier rename** — multi-pass `rg`+`sd` sweep for `WorkKind` → `Kind` (40 files), `WorkItem` → `ActionItem` where stale (separate narrow pass), `workitem` package/filename refs. One narrow regex per identifier; no catch-all passes. The SQL-layer `work_items → action_items` rename already shipped via `rename-task-to-actionitem.sql`; this scope is Go-identifier-only.

3. **File + type renames** — `internal/domain/task.go` → `internal/domain/action_item.go` (filename only; struct is already `type ActionItem`). Move `WorkKind` + its 5 constants from `internal/domain/workitem.go:35-44` into the **existing** `internal/domain/kind.go` (merging into the file that already contains `KindID`, `KindAppliesTo`, `KindDefinition`). New `type Kind string` stays distinct from existing `type KindID string` — different semantics (`Kind` = action_item row kind; `KindID` = catalog lookup id). `KindID(Kind(x))` conversion is the de facto pattern already (see `kind_capability.go:867`). Renaming `workitem.go` itself is out of scope.

4. **`template_libraries` excision** — delete every file/symbol under the `template_library*` / `template_reapply` / `TemplateLibrar*` / `node_contract_snapshot*` / `template_node_template*` / `template_child_rule*` surface (42 code files). Domain: `template_library.go`, `template_reapply.go`, `builtin_template_library.go`, `template_library_test.go`. App: `template_library*.go`, `template_contract*.go`, `template_reapply.go`, plus `snapshot.go` / `snapshot_test.go` sections importing them, plus ports + service wiring. MCP: `handler.go:86 registerTemplateLibraryTools`, `handler.go:1045 pickTemplateLibraryService`, `extended_tools.go` tool registrations (`till.bind_project_template_library`, `till.get_template_library`, `till.upsert_template_library`, `till.ensure_builtin_template_library`), `extended_tools_test.go` coverage, `instructions_tool*.go` `TemplateLibraryID` arg surface, `instructions_explainer.go` template focus. Common: `mcp_surface.go`, `app_service_adapter*.go`. CLI: `template_cli.go`, `template_builtin_cli_test.go`. TUI: `model.go` / `model_test.go` / `thread_mode.go` hits. Also delete `migrateTemplateLifecycle` at `repo.go:1030-1055` + its caller at `:650` + its two helpers (`backfillTemplateLibraryRevisions`, `backfillProjectTemplateBindingSnapshots`) — these ALTER `template_libraries` and `project_template_bindings`, which die with the tables.

5. **Drop `projects.kind` column** — strip `Kind` field from `type Project` at `internal/domain/project.go:16`, delete the assignment at `:85`, delete `ALTER TABLE projects ADD COLUMN kind` at `repo.go:588` (else migration re-adds the column on every DB open after `scripts/drops-rewrite.sql` runs), strip all MCP project handler filters / CLI `project_cli.go` usage / `project_cli_test.go` coverage / `internal/tui/model.go` readbacks at `:4856, :18747` / `thread_mode.go` references / 11 files total per `rg 'projects\.kind|project\.Kind'`.

6. **`scripts/drops-rewrite.sql` rewrite** — schema-only collapse, replacing the current 296-line multi-phase script. Target end-state + minimum 5 assertions:
   - `DELETE FROM kind_catalog WHERE id NOT IN ('project', 'actionItem')`.
   - `DROP TABLE template_libraries`, `template_node_templates`, `template_child_rules`, `template_child_rule_editor_kinds`, `template_child_rule_completer_kinds`, `project_template_bindings`, `node_contract_snapshots`, `node_contract_editor_kinds`, `node_contract_completer_kinds`.
   - `ALTER TABLE projects DROP COLUMN kind` (or SQLite equivalent: `CREATE TABLE projects_new` + copy + drop + rename).
   - `DROP TABLE tasks`.
   - `UPDATE action_items SET kind='actionItem', scope='actionItem'` (all 115 rows currently `task/task`).
   - Assertion block: `SELECT COUNT(*) FROM kind_catalog` = 2; `SELECT COUNT(*) FROM sqlite_master WHERE name LIKE 'template_%'` = 0; `SELECT COUNT(*) FROM sqlite_master WHERE name = 'tasks'` = 0; `SELECT COUNT(*) FROM pragma_table_info('projects') WHERE name = 'kind'` = 0; `SELECT COUNT(*) FROM action_items WHERE kind NOT IN ('project','actionItem')` = 0.
   - PHASE 6 role hydration and PHASE 7 multi-kind rewrite from the main-branch script are deleted outright — dev-DB cleanup on 2026-04-18 removed every row they'd touch.

7. **Legacy `tasks` table excision** — 26 references in `repo.go`: `CREATE TABLE tasks` (`:169`), `CREATE INDEX idx_tasks_project_column_position` (`:558`), `CREATE INDEX idx_tasks_project_parent` (`:665`), 13 `ALTER TABLE tasks` (`:592-604`), `UPDATE tasks` in `migratePhaseScopeContract` (`:717, :732-733`), 2 `UPDATE tasks` in actor-name migration (`:977-978`), `bridgeLegacyActionItemsToWorkItems` at `:1184-1228` (uses `FROM tasks t` at `:1218`). Delete `migratePhaseScopeContract` entirely (`:710-789`) — the kind_catalog subphase rewrite it performs becomes unreachable post-collapse. Delete test fixture `repo_test.go:1006-1049` that creates a legacy `tasks` table + inserts a row.

8. **Tests + fixtures** — update every test file in the `rg -l 'work_items|WorkKind[A-Z]|WorkKindPhase|WorkKindSubtask|WorkKindDecision|WorkKindNote|projects\.kind|template_librar|bridgeLegacyActionItems|seedDefaultKindCatalog|ensureKindCatalogBootstrapped'` surface (16 test files): `internal/tui/{thread_mode_test,model_test}.go`, `internal/domain/domain_test.go`, `internal/app/{template_library_test,template_contract_test,snapshot_test,service_test,kind_capability_test}.go`, `internal/adapters/storage/sqlite/repo_test.go`, `internal/adapters/server/mcpapi/{handler_integration_test,extended_tools_test}.go`, `internal/adapters/server/httpapi/handler_integration_test.go`, `internal/adapters/server/common/{app_service_adapter_mcp_actor_attribution_test,app_service_adapter_lifecycle_test,app_service_adapter_auth_context_test}.go`, `cmd/till/main_test.go`. Plus `cmd/till/project_cli_test.go` for `projects.kind` surface. Goldens: the 4 `.golden` files in `internal/tui/testdata/` do not reference the dying surface (verified via `grep`); no golden updates needed unless the TUI render output changes.

Out-of-scope:

- `metadata.role` first-class field and role hydration (Drop 2; today role stays in description prose).
- Cascade dispatcher + state-trigger engine (Drop 4+).
- Template system return (future drop — user/orch-defined templates, not `kind_catalog`-coupled).
- Per-row role hydration / multi-kind rewrite (dead code for this drop — dev DB row-level cleanup 2026-04-18 left only uniform `task/task` rows across 115 action_items).
- `project_id nullable` and any remaining auth schema changes (Drop 1).
- Renaming `internal/domain/workitem.go` itself (the residual lifecycle/context/actor/resource file) to a less-stale name (defer).
- **Orphan-via-collapse refactor.** The following four sites are **deferred out-of-scope** and stay in place as dead/partial code. Dev direct quote: *"I say we leave and just orphan. in the refinment drops we will be refactoring and cleaning up using hylla, would that be simplest? we don't want them actually running if we can help it"*. Per-site classification:
  - `KindAppliesTo` constants at `internal/domain/kind.go:22-28` (Project/Branch/Phase/ActionItem/Subtask): **mixed** — `Project` + `Branch` + `ActionItem` runtime-live; `Phase` + `Subtask` naturally unreachable post-collapse (no kind_catalog rows and no action_item rows will ever carry them).
  - `WorkKind` non-actionItem variants at `internal/domain/workitem.go:35-44` (`WorkKindSubtask`, `WorkKindPhase`, `WorkKindDecision`, `WorkKindNote`): **naturally unreachable** — no DB rows will ever contain these strings post drops-rewrite.sql.
  - `capabilityScopeTypeForActionItem` at `internal/app/kind_capability.go:409-423`: **mixed** — `Branch` branch (`:414-415`) is **runtime-live** because drop-scoped auth path uses `/branch/<drop-id>` per the pre-Drop-2 auth-path-branch-quirk rule; `Phase/Subtask` branches are naturally unreachable; `Project` + `default→ActionItem` are live.
  - `AuthRequestPathKind` + constants at `internal/domain/auth_request.go:43-49`: **all live** — auth-request path kinds (`project`, `projects`, `global`) are orthogonal to action_item kinds. No action needed; refinement drop reviews fit.

Verification: `mage ci` green from `drop/1.75/`, `gh run watch --exit-status` green on `drop/1.75` branch, and dev re-runs `scripts/drops-rewrite.sql` against `~/.tillsyn/tillsyn.db` cleanly.

## Planner

Atomic units of work. Each unit mutates its `state` field in place during Phase 4. Blocker semantics per `CLAUDE.md` § "Blocker Semantics" — sibling units sharing a package in `Packages` OR a file in `Paths` must have an explicit `blocked_by`.

### 1.1 — Rename `WorkKind → Kind` + stale `WorkItem*` / `workitem` identifier references

**State:** todo
**Paths:** `internal/domain/workitem.go`, `internal/domain/kind.go`, `internal/domain/task.go`, `internal/domain/template_library.go`, `internal/domain/template_reapply.go`, `internal/domain/comment.go`, `internal/domain/change_event.go`, `internal/domain/attention_level_test.go`, `internal/domain/domain_test.go`, `internal/app/*.go`, `internal/adapters/storage/sqlite/*.go`, `internal/adapters/server/mcpapi/*.go`, `internal/adapters/server/common/*.go`, `internal/adapters/server/httpapi/*.go`, `internal/tui/*.go`, `cmd/till/*.go` (any file in the 40-file surface of `rg 'WorkKind|WorkItem[^a-z]|workitem'`, excluding `drops/**`)
**Packages:** `internal/domain`, `internal/app`, `internal/adapters/storage/sqlite`, `internal/adapters/server/mcpapi`, `internal/adapters/server/common`, `internal/adapters/server/httpapi`, `internal/tui`, `cmd/till`
**Blocked by:** —
**Acceptance:**
- `rg 'WorkKind' drop/1.75/ --glob='!drops/**'` returns 0 matches.
- `rg 'type WorkItem |WorkItemKind|WorkItemID' drop/1.75/ --glob='!drops/**'` returns 0 matches (the table `work_items` is already `action_items`; only stale Go symbol refs die).
- `mage build` succeeds from `drop/1.75/`.
- `mage test-pkg ./internal/domain` passes.

Multi-pass `rg`+`sd` sweep. Narrow regexes per identifier: `WorkKind\b → Kind`, `WorkKindActionItem → KindActionItem`, `WorkKindSubtask → KindSubtask`, `WorkKindPhase → KindPhase`, `WorkKindDecision → KindDecision`, `WorkKindNote → KindNote`. `WorkItem`-prefixed symbols renamed to `ActionItem`-prefixed. `workitem` package filename refs kept as-is (renaming `workitem.go` is out of scope). This unit establishes the `Kind` type surface every downstream unit depends on — it is the single non-blocked unit.

### 1.2 — Delete app-layer kind-catalog seeder

**State:** todo
**Paths:** `internal/app/kind_capability.go`, `internal/app/kind_capability_test.go`, `internal/app/service.go` (remove `kindBootstrap` struct field if declared there), `internal/app/service_test.go`
**Packages:** `internal/app`
**Blocked by:** 1.1
**Acceptance:**
- `rg 'ensureKindCatalogBootstrapped|defaultKindDefinitionInputs|kindBootstrap' drop/1.75/ --glob='!drops/**'` returns 0 matches.
- `mage test-pkg ./internal/app` passes.

Delete `ensureKindCatalogBootstrapped` at `kind_capability.go:559-589`, its `sync.Once` struct field `kindBootstrap` (declared on `Service` — builder confirms via `LSP` before deletion), and `defaultKindDefinitionInputs` at `:863-874`. Update every caller (`resolveProjectKindDefinition` at `:592-596` and similar) to skip the bootstrap call — built-in rows live in the `CREATE TABLE kind_catalog` baked inserts after 1.3 ships.

### 1.3 — Bake kind_catalog rows + delete SQLite seeder + drop project.kind ALTER

**State:** todo
**Paths:** `internal/adapters/storage/sqlite/repo.go`, `internal/adapters/storage/sqlite/repo_test.go`
**Packages:** `internal/adapters/storage/sqlite`
**Blocked by:** 1.1, 1.2
**Acceptance:**
- `rg 'seedDefaultKindCatalog|mergeKindAppliesTo|kindAppliesToEqual' drop/1.75/ --glob='!drops/**'` returns 0 matches (or only the helpers' remaining uses outside the deleted seeder).
- `rg "ALTER TABLE projects ADD COLUMN kind" drop/1.75/` returns 0 matches.
- Fresh DB open produces exactly 2 rows in `kind_catalog` (`project`, `actionItem`). Verified by test `TestRepositoryFreshOpenKindCatalog` (builder adds this test to `repo_test.go`).
- `mage test-pkg ./internal/adapters/storage/sqlite` passes.

Delete `seedDefaultKindCatalog` at `repo.go:1231-1301` (full range including merge/upsert block). Delete caller from `bootstrapSchema` / migration runner (builder verifies via `LSP`). Bake two `INSERT OR IGNORE INTO kind_catalog` statements for `project` + `actionItem` directly inside `CREATE TABLE kind_catalog` block (`:316`). Delete `ALTER TABLE projects ADD COLUMN kind` at `:588` (per F4 — else migration re-adds the column on every DB open post-drops-rewrite). Drop `mergeKindAppliesTo` + `kindAppliesToEqual` helpers if no other caller survives after 1.1.

### 1.4 — Template libraries + node_contract_snapshots domain excision

**State:** todo
**Paths:** `internal/domain/template_library.go`, `internal/domain/template_library_test.go`, `internal/domain/template_reapply.go`, `internal/domain/builtin_template_library.go`, `internal/domain/errors.go` (remove template-library error sentinels)
**Packages:** `internal/domain`
**Blocked by:** 1.1
**Acceptance:**
- `rg 'TemplateLibrary|TemplateReapply|NodeContractSnapshot|BuiltinTemplate' drop/1.75/internal/domain/` returns 0 matches.
- `mage test-pkg ./internal/domain` passes.
- `grep -c 'ErrTemplate' internal/domain/errors.go` returns 0.

Delete the four files; strip `ErrTemplate*` sentinels from `errors.go`. `internal/app` and `internal/adapters/*` still hold template references but they die in 1.5–1.7 before the compile gate trips (this unit only deletes domain-layer types; downstream packages won't compile until 1.5 runs).

### 1.5 — Template libraries app + adapter + CLI excision

**State:** todo
**Paths:** `internal/app/template_library.go`, `internal/app/template_library_builtin.go`, `internal/app/template_library_builtin_spec.go`, `internal/app/template_library_test.go`, `internal/app/template_contract.go`, `internal/app/template_contract_test.go`, `internal/app/template_reapply.go`, `internal/app/snapshot.go` (strip `TemplateLibraries` field + `snapshotTemplateLibraryFromDomain` + `upsertTemplateLibrary` + `normalizeSnapshotTemplateLibrary` sections), `internal/app/snapshot_test.go`, `internal/app/service.go` (strip template service fields + bindings), `internal/app/service_test.go`, `internal/app/ports.go` (strip `TemplateLibraryRepo` port), `internal/app/helper_coverage_test.go`, `internal/adapters/storage/sqlite/repo.go` (delete `migrateTemplateLifecycle` at `:1030-1055`, caller at `:650`, `backfillTemplateLibraryRevisions` at `:1057+`, `backfillProjectTemplateBindingSnapshots`; delete all `TemplateLibrary` repo methods: `ListTemplateLibraries`, `CreateTemplateLibrary`, `UpdateTemplateLibrary`, `DeleteTemplateLibrary`, `GetTemplateLibrary`, `UpsertTemplateLibrary`, and every `NodeContractSnapshot*` + `ProjectTemplateBinding*` repo method), `internal/adapters/storage/sqlite/template_library_test.go`, `internal/adapters/server/common/mcp_surface.go`, `internal/adapters/server/common/app_service_adapter.go`, `internal/adapters/server/common/app_service_adapter_mcp.go`, `internal/adapters/server/common/app_service_adapter_auth_context.go`, `internal/adapters/server/common/app_service_adapter_auth_context_test.go`, `internal/adapters/server/common/app_service_adapter_mcp_actor_attribution_test.go`, `internal/adapters/server/common/app_service_adapter_lifecycle_test.go`, `internal/adapters/server/mcpapi/handler.go` (delete `pickTemplateLibraryService` at `:1045-1050` + call at `:66, :72, :86`), `internal/adapters/server/mcpapi/extended_tools.go` (delete `till.bind_project_template_library` at `:2171`, `till.get_template_library` at `:2258`, `till.upsert_template_library` at `:2281`, `till.ensure_builtin_template_library` at `:2085`, plus all `TemplateLibraryID` argument handling at `:435, :457, :595, :604, :840, :853`), `internal/adapters/server/mcpapi/extended_tools_test.go`, `internal/adapters/server/mcpapi/instructions_tool.go` (strip `TemplateLibraryID` field + arg), `internal/adapters/server/mcpapi/instructions_tool_test.go`, `internal/adapters/server/mcpapi/instructions_explainer.go` (strip `template_library_description` + template focus branch at `:296, :338`), `internal/adapters/server/mcpapi/handler_integration_test.go`, `internal/adapters/server/httpapi/handler_integration_test.go`, `internal/tui/model.go` (strip any `TemplateLibrary` readbacks — F7 says none; builder verifies), `internal/tui/model_test.go`, `internal/tui/thread_mode.go`, `cmd/till/template_cli.go` (delete entire file), `cmd/till/template_builtin_cli_test.go` (delete entire file), `cmd/till/main.go` (strip template-cli command registration), `cmd/till/main_test.go`
**Packages:** `internal/app`, `internal/adapters/storage/sqlite`, `internal/adapters/server/common`, `internal/adapters/server/mcpapi`, `internal/adapters/server/httpapi`, `internal/tui`, `cmd/till`
**Blocked by:** 1.1, 1.2, 1.3, 1.4
**Acceptance:**
- `rg 'TemplateLibrary|TemplateReapply|NodeContractSnapshot|BuiltinTemplate|node_contract_snapshot|template_librar|template_node_template|template_child_rule|project_template_binding' drop/1.75/ --glob='!drops/**' --glob='!scripts/drops-rewrite.sql'` returns 0 matches.
- `mage ci` succeeds from `drop/1.75/`.
- MCP tools `till.bind_project_template_library`, `till.get_template_library`, `till.upsert_template_library`, `till.ensure_builtin_template_library` absent from registered tools (verify via `rg 'till\.(bind_project_template_library|get_template_library|upsert_template_library|ensure_builtin_template_library)' internal/adapters/server/mcpapi/` returns 0).

The big atomic excision. Must be one unit because MCP imports app, which imports domain — splitting by package leaves intermediate compile-broken states. Builder deletes in bottom-up order (sqlite → app → common → mcpapi → httpapi → tui → cmd) so each package compiles at the end of its sub-pass. `snapshot.go` is the trickiest surface: strip the `TemplateLibraries` field from `Snapshot` struct and every reference to template serialization in validation / upsert / normalize helpers.

### 1.6 — Drop `projects.kind` column (domain + app + MCP + TUI + CLI)

**State:** todo
**Paths:** `internal/domain/project.go` (strip `Kind KindID` field at `:16`, assignment at `:85`), `internal/app/kind_capability.go` (strip `resolveProjectKindDefinition` + callers), `internal/app/service.go` (strip project.Kind references), `internal/app/template_reapply.go` (strip — partly duplicated w/ 1.5 deletion), `internal/adapters/server/mcpapi/instructions_explainer.go` (strip project.Kind readback), `internal/tui/model.go` (strip `project.Kind` readbacks at `:4856, :18747` and `projectFieldKind` form input if present), `internal/tui/thread_mode.go`, `cmd/till/project_cli.go`, `cmd/till/project_cli_test.go`
**Packages:** `internal/domain`, `internal/app`, `internal/adapters/server/mcpapi`, `internal/tui`, `cmd/till`
**Blocked by:** 1.1, 1.2, 1.3, 1.4, 1.5
**Acceptance:**
- `rg 'project\.Kind|projects\.kind|Project\{[^}]*Kind' drop/1.75/ --glob='!drops/**' --glob='!scripts/drops-rewrite.sql'` returns 0 matches.
- `rg 'projectFieldKind' drop/1.75/` returns 0 matches.
- `mage ci` succeeds.

The column strip is deferred to this unit (after 1.5) because `internal/tui/model.go` and `cmd/till/project_cli.go` hold `project.Kind` readbacks the template system touches — ordering them before template excision risks re-introducing compile cycles.

### 1.7 — Legacy `tasks` table excision

**State:** todo
**Paths:** `internal/adapters/storage/sqlite/repo.go`, `internal/adapters/storage/sqlite/repo_test.go`
**Packages:** `internal/adapters/storage/sqlite`
**Blocked by:** 1.1, 1.2, 1.3, 1.5
**Acceptance:**
- `rg 'CREATE TABLE( IF NOT EXISTS)? tasks|ALTER TABLE tasks|UPDATE tasks|FROM tasks|INSERT INTO tasks|idx_tasks_' drop/1.75/internal/` returns 0 matches.
- `rg 'bridgeLegacyActionItemsToWorkItems|migratePhaseScopeContract' drop/1.75/` returns 0 matches.
- `mage test-pkg ./internal/adapters/storage/sqlite` passes.

Delete `CREATE TABLE tasks` at `:169`, `CREATE INDEX idx_tasks_project_column_position` at `:558`, `CREATE INDEX idx_tasks_project_parent` at `:665`, 13 `ALTER TABLE tasks` at `:592-604`, `UPDATE tasks` in `migratePhaseScopeContract` at `:717, :732-733`, 2 `UPDATE tasks` in actor-name migration at `:977-978`, `bridgeLegacyActionItemsToWorkItems` entire function at `:1184-1228`, plus its caller in the migration runner. Delete `migratePhaseScopeContract` entirely at `:710-789` — the subphase→phase rewrite it performs is unreachable after 1.3 bakes `{project, actionItem}` into kind_catalog. Delete test fixture `repo_test.go:1006-1049` that creates a legacy `tasks` table + inserts a row.

### 1.8 — Rename `internal/domain/task.go → action_item.go`

**State:** todo
**Paths:** `internal/domain/task.go` (delete), `internal/domain/action_item.go` (create with identical content)
**Packages:** `internal/domain`
**Blocked by:** 1.1, 1.4, 1.6
**Acceptance:**
- `ls internal/domain/task.go` fails (file absent).
- `ls internal/domain/action_item.go` succeeds.
- `mage test-pkg ./internal/domain` passes.

File-only rename via `git mv`. No content changes. Ordered after 1.1, 1.4, 1.6 so the file's content is final before the rename.

### 1.9 — Merge `WorkKind` block into existing `internal/domain/kind.go`

**State:** todo
**Paths:** `internal/domain/workitem.go` (delete `:35-44` block: `type WorkKind` + 5 constants; file is renamed `Kind` post-1.1), `internal/domain/kind.go` (absorb the block — place `type Kind string` + 5 `KindActionItem` / `KindSubtask` / etc. constants near top, distinct from existing `type KindID string`)
**Packages:** `internal/domain`
**Blocked by:** 1.1, 1.4, 1.6, 1.8
**Acceptance:**
- `grep 'type Kind string' internal/domain/kind.go` returns 1 match.
- `grep 'type KindID string' internal/domain/kind.go` returns 1 match.
- `grep 'type Kind string\|type WorkKind' internal/domain/workitem.go` returns 0 matches.
- `grep -c 'KindActionItem\|KindSubtask\|KindPhase\|KindDecision\|KindNote' internal/domain/kind.go` returns at least 5.
- `mage test-pkg ./internal/domain` passes.

Moves the renamed block from `workitem.go:35-44` (post-1.1 it reads `type Kind string` + `KindActionItem`/... constants) into `kind.go`, preserving both `Kind` and `KindID` as distinct types per the P6 decision. After this unit, `workitem.go` retains only lifecycle/context/actor/resource enums and becomes a residual rename-target (deferred to a future drop).

### 1.10 — Domain test updates

**State:** todo
**Paths:** `internal/domain/domain_test.go`, `internal/domain/attention_level_test.go`
**Packages:** `internal/domain`
**Blocked by:** 1.1, 1.4, 1.6, 1.8, 1.9
**Acceptance:**
- `mage test-pkg ./internal/domain` passes with no skipped tests.
- `rg 'WorkKind|TemplateLibrary|project\.Kind' internal/domain/*_test.go` returns 0 matches.

Domain tests trail the domain-package unit chain to pick up the last stable state.

### 1.11 — App test updates

**State:** todo
**Paths:** `internal/app/kind_capability_test.go`, `internal/app/service_test.go`, `internal/app/snapshot_test.go`, `internal/app/helper_coverage_test.go`, `internal/app/search_embeddings_test.go`, `internal/app/embedding_runtime_test.go`
**Packages:** `internal/app`
**Blocked by:** 1.2, 1.5, 1.6
**Acceptance:**
- `mage test-pkg ./internal/app` passes.
- Coverage for `internal/app` ≥ 70% (project CLAUDE.md Build Verification rule).
- `rg 'WorkKind|TemplateLibrary|ensureKindCatalogBootstrapped' internal/app/*_test.go` returns 0 matches.

### 1.12 — Adapter + MCP test updates

**State:** todo
**Paths:** `internal/adapters/storage/sqlite/repo_test.go`, `internal/adapters/storage/sqlite/embedding_jobs_test.go`, `internal/adapters/storage/sqlite/embedding_lifecycle_adapter_test.go`, `internal/adapters/storage/sqlite/handoff_test.go`, `internal/adapters/server/mcpapi/handler_integration_test.go`, `internal/adapters/server/mcpapi/extended_tools_test.go`, `internal/adapters/server/mcpapi/instructions_tool_test.go`, `internal/adapters/server/httpapi/handler_integration_test.go`, `internal/adapters/server/common/app_service_adapter_mcp_actor_attribution_test.go`, `internal/adapters/server/common/app_service_adapter_lifecycle_test.go`, `internal/adapters/server/common/app_service_adapter_auth_context_test.go`
**Packages:** `internal/adapters/storage/sqlite`, `internal/adapters/server/mcpapi`, `internal/adapters/server/httpapi`, `internal/adapters/server/common`
**Blocked by:** 1.3, 1.5, 1.7
**Acceptance:**
- `mage test-pkg ./internal/adapters/storage/sqlite` passes.
- `mage test-pkg ./internal/adapters/server/mcpapi` passes.
- `mage test-pkg ./internal/adapters/server/httpapi` passes.
- `mage test-pkg ./internal/adapters/server/common` passes.
- `rg 'WorkKind|TemplateLibrary|template_librar|bridgeLegacyActionItems|seedDefaultKindCatalog|FROM tasks|projects\.kind' drop/1.75/internal/adapters/ --glob='*_test.go'` returns 0 matches.

### 1.13 — TUI + CLI test updates

**State:** todo
**Paths:** `internal/tui/model_test.go`, `internal/tui/thread_mode_test.go`, `internal/tui/model_teatest_test.go`, `internal/tui/description_editor_mode.go` (strip dead WorkKind refs if any), `cmd/till/main_test.go`, `cmd/till/project_cli_test.go`
**Packages:** `internal/tui`, `cmd/till`
**Blocked by:** 1.5, 1.6
**Acceptance:**
- `mage test-pkg ./internal/tui` passes.
- `mage test-pkg ./cmd/till` passes.
- `mage test-golden` passes (goldens unchanged per F7 — regenerate via `mage test-golden-update` only if TUI render drift is proven).
- `rg 'WorkKind|TemplateLibrary|project\.Kind|projectFieldKind' drop/1.75/internal/tui/ drop/1.75/cmd/till/` returns 0 matches.

### 1.14 — `scripts/drops-rewrite.sql` rewrite

**State:** todo
**Paths:** `scripts/drops-rewrite.sql`
**Packages:** — (non-Go)
**Blocked by:** 1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 1.7, 1.8, 1.9, 1.10, 1.11, 1.12, 1.13
**Acceptance:**
- Script runs cleanly against dev's `~/.tillsyn/tillsyn.db` (dev-applied at drop end; builder verifies on a copy: `cp ~/.tillsyn/tillsyn.db /tmp/verify.db && sqlite3 /tmp/verify.db < scripts/drops-rewrite.sql`).
- Post-run assertions (all must pass):
  - `SELECT COUNT(*) FROM kind_catalog` returns 2.
  - `SELECT COUNT(*) FROM sqlite_master WHERE name LIKE 'template_%'` returns 0.
  - `SELECT COUNT(*) FROM sqlite_master WHERE name = 'tasks'` returns 0.
  - `SELECT COUNT(*) FROM pragma_table_info('projects') WHERE name = 'kind'` returns 0.
  - `SELECT COUNT(*) FROM action_items WHERE kind NOT IN ('project','actionItem')` returns 0.
- Rollback on assertion failure (via `BEGIN TRANSACTION` + `SELECT RAISE(ROLLBACK, ...)` guards).

Schema-only collapse replacing the current 296-line multi-phase script. Phases: (1) pre-flight counts, (2) `DELETE FROM kind_catalog WHERE id NOT IN ('project', 'actionItem')`, (3) `DROP TABLE` the template cluster (9 tables per F9), (4) SQLite table-rebuild to drop `projects.kind` column, (5) `DROP TABLE tasks`, (6) `UPDATE action_items SET kind='actionItem', scope='actionItem'`, (7) assertion block. PHASE 6 role hydration + PHASE 7 multi-kind rewrite from the main-branch script are deleted outright (dev-DB cleanup 2026-04-18 removed every row they'd touch).

### 1.15 — Drop-end `mage ci` gate

**State:** todo
**Paths:** — (verification-only, no code edits)
**Packages:** — (workspace-wide)
**Blocked by:** 1.14
**Acceptance:**
- `mage ci` succeeds from `drop/1.75/` (format, vet, test, build, lint — per CLAUDE.md § Build Verification).
- `git push` + `gh run watch --exit-status` returns green on branch `drop/1.75`.
- Coverage ≥ 70% workspace-wide.
- `rg 'WorkKind|TemplateLibrary|template_librar|node_contract_snapshot|seedDefaultKindCatalog|ensureKindCatalogBootstrapped|bridgeLegacyActionItems|migratePhaseScopeContract|migrateTemplateLifecycle|projects\.kind|project\.Kind|FROM tasks' drop/1.75/ --glob='!drops/**' --glob='!scripts/drops-rewrite.sql'` returns 0 matches (end-state invariant).

Drop-end gate. No code edits — pure verification that the workspace is clean after all 14 prior units commit. Orchestrator runs this before any `hylla_ingest` call (ingest is drop-end only, per Phase 7 Closeout).

## Notes

- The existing `drop/1.75/PLAN.md` (repo root, not this file) is the inherited big tillsyn cascade plan from the `main` branch. Do NOT edit it for coordination — it merges back to main unchanged. All per-drop coordination lives here.
- **Pre-drop dev DB cleanup (2026-04-18)**: dev purged the live `~/.tillsyn/tillsyn.db` down to a single project (`tillsyn`) and 115 action_items, all with uniform `kind='task', scope='task'`. Every legacy kind (`build-task`, `qa-check`, `subtask`, `project-setup-phase`) and every non-tillsyn project was deleted. Backups sit at `~/.tillsyn/tillsyn.db.pre-*-purge`. This dramatically narrows what `drops-rewrite.sql` has to do.
- **`__global__` auth project is self-healing.** `internal/adapters/storage/sqlite/repo.go:1455-1473` `ensureGlobalAuthProject` runs on every DB open with `INSERT ... ON CONFLICT(id) DO NOTHING`. Dev deleting the `__global__` row during pre-drop cleanup is fine — it rebuilds on next binary startup. No migration work here.
- **Go identifier rename scope narrower than the brief suggested.** The `work_items → action_items` **table** rename shipped pre-Drop-1.75 on 2026-04-18 via `scripts/rename-task-to-actionitem.sql`. `repo.go:198` already reads `CREATE TABLE action_items`. Only Go-identifier renames remain (`WorkKind → Kind`, any stale `WorkItem*` refs, `workitem` package/filename refs).
- **`KindID` vs `Kind` decision.** Post-rename `type Kind string` stays distinct from existing `type KindID string` — different semantics (catalog lookup id vs action_item row kind slug). Conversions remain explicit (`domain.KindID(domain.Kind(x))`), matching the de facto pattern in `kind_capability.go:867`.
- **`seedDefaultKindCatalog` dies.** The function at `repo.go:1231-1301` re-seeds 7 kinds on every DB open, which is the mechanism by which legacy kinds keep materializing. Replacement: bake the two surviving rows (`project`, `actionItem`) into the initial `CREATE TABLE kind_catalog` schema migration so DB open is idempotent without a seed loop.
- **`bridgeLegacyActionItemsToWorkItems` dies with the `tasks` table.** The shim at `repo.go:1184-1228` exists only to translate the long-empty legacy `tasks` table into `work_items`. Dropping the table removes the shim's reason to exist.
- **`template_libraries` migration hook dies.** `migrateTemplateLifecycle` at `repo.go:1030-1055` + its caller at `:650` + helpers `backfillTemplateLibraryRevisions` and `backfillProjectTemplateBindingSnapshots` die together with the tables they alter.
- **`ALTER TABLE projects ADD COLUMN kind` at `repo.go:588` must die with the column strip.** Otherwise every DB open after drops-rewrite.sql re-adds the column.
- **MCP tool deregistration surface.** At least 4 MCP tools in `extended_tools.go`: `till.bind_project_template_library` (`:2171`), `till.get_template_library` (`:2258`), `till.upsert_template_library` (`:2281`), `till.ensure_builtin_template_library` (`:2085`). All die, along with their test coverage in `extended_tools_test.go:1679+`. `instructions_tool.go` also exposes `TemplateLibraryID` arg (`:65, :126, :146, :300, :318`) — dies. `handler.go` wiring at `:86, :1045-1050` dies. `common/mcp_surface.go` dies.
- **Goldens are not affected.** The 4 `.golden` files in `internal/tui/testdata/` do not reference `template_librar` / `node_contract_snapshot` / `WorkKindPhase` / `WorkKindSubtask` / `projects.kind` per `grep`. Regeneration via `mage test-golden-update` only if TUI render output changes after F11 hits are removed.
- **`projects.kind` column is user/orch-facing metadata, not a system tracking concern.** Projects get template suggestions from the user or orch at creation and across the project's life; the system doesn't need to persist a kind. Column gets dropped along with every Go reference, MCP handler filter, and TUI view that reads it.
- **F10 package-level blocker contract.** `internal/domain` has 5 in-drop units (Go identifier rename 1.1, template_libraries domain excision 1.4, projects.kind strip 1.6, action_item.go filename 1.8, kind.go merge 1.9) that share the package compile. Per `CLAUDE.md` §Blocker Semantics, sibling units sharing a package require explicit `blocked_by`. The Planner section above serializes these five via a linear `blocked_by` chain. `internal/app` units (1.2 / 1.5 / 1.7) similarly chain within the app package.
- `scripts/drops-rewrite.sql` on `main` currently rewrites every non-project node's `kind` to `drop` (Drop-2 vocabulary) and hydrates 8 role variants from legacy kinds. Drop 1.75's builder **replaces** that script with a schema-only collapse against the post-rename `action_items` table: `kind_catalog` → `{project, actionItem}`, `template_librar*` / `node_contract_snapshot*` / `template_node_template*` / `template_child_rule*` wipe, `ALTER TABLE projects DROP COLUMN kind`, drop legacy `tasks` table, one-line `UPDATE action_items SET kind='actionItem', scope='actionItem'`, assert `kind_catalog` row count = 2 plus 4 more end-state invariants. PHASE 6 role hydration and PHASE 7 multi-kind rewrite from the old script are deleted.
- Dev manually applies `scripts/drops-rewrite.sql` against `~/.tillsyn/tillsyn.db` at drop end (same pattern as `scripts/rename-task-to-actionitem.sql`). Agents never touch the dev DB.
- **One drop, not split.** All eight in-scope items above ship in this drop across 15 atomic units (1.1–1.15). The planner decomposed into units with `blocked_by` wiring (package-level serialization enforced in `internal/domain` + `internal/app`; `scripts/drops-rewrite.sql` is a sink blocking on every code unit; `mage ci` gate is the final drop-end invariant).
