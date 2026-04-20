# DROP_1_75 — KIND_COLLAPSE

**State:** building
**Blocked by:** —
**Paths (expected):** `internal/domain/kind.go` (merge target — existing file absorbs `Kind` type from `workitem.go:35-44`), `internal/domain/action_item.go` (renamed from `task.go`), `internal/domain/workitem.go` (residual — lifecycle/context/actor/resource enums; rename pending; also loses `WorkKind` block at :35-44), `internal/domain/project.go` (strip `Kind` field), `internal/domain/template_library.go`, `internal/domain/template_reapply.go`, `internal/domain/builtin_template_library.go`, `internal/domain/*_test.go`, `internal/adapters/storage/sqlite/repo.go` (schema inline; `CREATE TABLE tasks` at :169, `CREATE TABLE action_items` at :198, `CREATE TABLE kind_catalog` at :316, `seedDefaultKindCatalog` at :1231-1301, `migrateTemplateLifecycle` at :1030-1055 + caller :650, `migratePhaseScopeContract` at :710-789, `bridgeLegacyActionItemsToWorkItems` at :1184-1228, `ALTER TABLE projects ADD COLUMN kind` at :588, 13 `ALTER TABLE tasks` at :592-604, 2 `UPDATE tasks` at :977-978, CREATE INDEXes at :558 :665, backfill helpers at :1057+), `internal/adapters/storage/sqlite/repo_test.go` (legacy tasks fixture at :1006-1049), `internal/adapters/storage/sqlite/template_library_test.go`, `internal/adapters/server/mcpapi/handler.go` (registerTemplateLibraryTools at :86, pickTemplateLibraryService at :1046-1054), `internal/adapters/server/mcpapi/extended_tools.go` (till.bind_project_template_library :2171+, till.get_template_library :2258+, till.upsert_template_library :2281+, till.ensure_builtin_template_library :2085+), `internal/adapters/server/mcpapi/extended_tools_test.go`, `internal/adapters/server/mcpapi/instructions_tool.go`, `internal/adapters/server/mcpapi/instructions_tool_test.go`, `internal/adapters/server/mcpapi/instructions_explainer.go`, `internal/adapters/server/mcpapi/handler_integration_test.go`, `internal/adapters/server/common/mcp_surface.go`, `internal/adapters/server/common/app_service_adapter_mcp.go`, `internal/adapters/server/common/app_service_adapter_mcp_actor_attribution_test.go`, `internal/adapters/server/common/app_service_adapter.go`, `internal/adapters/server/common/app_service_adapter_auth_context.go`, `internal/adapters/server/common/app_service_adapter_auth_context_test.go`, `internal/adapters/server/common/app_service_adapter_lifecycle_test.go`, `internal/adapters/server/httpapi/handler_integration_test.go`, `internal/app/kind_capability.go` (ensureKindCatalogBootstrapped :559-589 + sync.Once field + defaultKindDefinitionInputs :863-874), `internal/app/template_library.go`, `internal/app/template_library_builtin.go`, `internal/app/template_library_builtin_spec.go`, `internal/app/template_library_test.go`, `internal/app/template_contract.go`, `internal/app/template_contract_test.go`, `internal/app/template_reapply.go`, `internal/app/snapshot.go`, `internal/app/snapshot_test.go`, `internal/app/service.go`, `internal/app/service_test.go`, `internal/app/ports.go`, `internal/app/helper_coverage_test.go`, `internal/app/kind_capability_test.go`, `internal/tui/model.go` (9 hard refs: project.Kind at :4856, :18747; WorkKindSubtask at :5190 :5200 :14840 :17905 :19236; WorkKindPhase at :5227 :8957), `internal/tui/model_test.go`, `internal/tui/thread_mode.go`, `internal/tui/thread_mode_test.go`, `cmd/till/template_cli.go`, `cmd/till/template_builtin_cli_test.go`, `cmd/till/project_cli.go`, `cmd/till/project_cli_test.go`, `cmd/till/main.go`, `cmd/till/main_test.go`, `scripts/drops-rewrite.sql`, all `WorkKind` / `WorkItem` / `workitem` Go identifier references sitewide (multi-pass `rg`+`sd`; the `work_items → action_items` table rename already shipped via `scripts/rename-task-to-actionitem.sql`).
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

4. **`template_libraries` excision** — delete every file/symbol under the `template_library*` / `template_reapply` / `TemplateLibrar*` / `node_contract_snapshot*` / `template_node_template*` / `template_child_rule*` surface (42 code files). Domain: `template_library.go`, `template_reapply.go`, `builtin_template_library.go`, `template_library_test.go`. App: `template_library*.go`, `template_contract*.go`, `template_reapply.go`, plus `snapshot.go` / `snapshot_test.go` sections importing them, plus ports + service wiring. MCP: `handler.go:86 registerTemplateLibraryTools`, `handler.go:1046-1054 pickTemplateLibraryService` (per P9 line-drift correction), `extended_tools.go` tool registrations (`till.bind_project_template_library`, `till.get_template_library`, `till.upsert_template_library`, `till.ensure_builtin_template_library`), `extended_tools_test.go` coverage, `instructions_tool*.go` `TemplateLibraryID` arg surface, `instructions_explainer.go` template focus. Common: `mcp_surface.go`, `app_service_adapter*.go`. CLI: `template_cli.go`, `template_builtin_cli_test.go`. TUI: `model.go` / `model_test.go` / `thread_mode.go` hits. Also delete `migrateTemplateLifecycle` at `repo.go:1030-1055` + its caller at `:650` + its two helpers (`backfillTemplateLibraryRevisions`, `backfillProjectTemplateBindingSnapshots`) — these ALTER `template_libraries` and `project_template_bindings`, which die with the tables. Error sentinels in `internal/domain/errors.go:25-33` that reference template_libraries (`ErrTemplateLibraryNotFound`, `ErrInvalidTemplateLibrary`, `ErrInvalidTemplateLibraryScope`, `ErrInvalidTemplateStatus`, `ErrInvalidTemplateActorKind`, `ErrInvalidTemplateBinding`, `ErrBuiltinTemplateBootstrapRequired`, `ErrNodeContractForbidden`) die here; `ErrInvalidKindTemplate` at the same block is **preserved** — it's referenced by the surviving `KindDefinition.Template` / `normalizeKindTemplate` machinery per the F5 orphan classification above.

5. **Drop `projects.kind` column** — strip `Kind KindID` field from `type Project` at `internal/domain/project.go:16`, delete the `Kind: DefaultProjectKind,` entry in `NewProject`'s struct literal at `:60`, delete the `SetKind(kind KindID, ...)` method entirely at `:79-88` (with its assignment at `:85`), delete `ALTER TABLE projects ADD COLUMN kind` at `repo.go:588` AND strip `kind TEXT NOT NULL DEFAULT 'project'` from the `CREATE TABLE IF NOT EXISTS projects` block at `repo.go:152` (inside `migrate()` at `:144` — else every fresh DB in CI or new-user first-run re-materializes the column, violating the drop's end-state invariant), strip all MCP project handler filters / CLI `project_cli.go` usage / `project_cli_test.go` coverage / `internal/tui/model.go` readbacks at `:4856, :18747` / `thread_mode.go` references / audit every `Project{...}` struct literal construction sitewide (including tests) to remove `Kind:` field references / 11 files total per `rg 'projects\.kind|project\.Kind'`.

6. **`scripts/drops-rewrite.sql` rewrite** — schema-only collapse, replacing the current 296-line multi-phase script. Target end-state + minimum 5 assertions:
   - `DELETE FROM kind_catalog WHERE id NOT IN ('project', 'actionItem')`.
   - `DROP TABLE template_libraries`, `template_node_templates`, `template_child_rules`, `template_child_rule_editor_kinds`, `template_child_rule_completer_kinds`, `project_template_bindings`, `node_contract_snapshots`, `node_contract_editor_kinds`, `node_contract_completer_kinds`.
   - `ALTER TABLE projects DROP COLUMN kind`.
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
  - `KindDefinition.Template` / `KindTemplate` / `KindTemplateChildSpec` / `validateKindTemplateExpansion` at `internal/domain/kind.go:57-73` + `internal/app/kind_capability.go:977-1010` (plus `normalizeKindTemplate` at `kind.go:262-275` and the `ErrInvalidKindTemplate` sentinel): **naturally unreachable** post-collapse — `kind_catalog` bakes empty `auto_create_children` for both surviving rows so the expansion path never fires. Retention intentional; refinement drop deletes.

Verification: `mage ci` green from `drop/1.75/`, `gh run watch --exit-status` green on `drop/1.75` branch, and dev re-runs `scripts/drops-rewrite.sql` against `~/.tillsyn/tillsyn.db` cleanly.

## Planner

Atomic units of work. Each unit mutates its `state` field in place during Phase 4. Blocker semantics per `CLAUDE.md` § "Blocker Semantics" — sibling units sharing a package in `Packages` OR a file in `Paths` must have an explicit `blocked_by`.

### 1.1 — Rename `WorkKind → Kind` + stale `WorkItem*` / `workitem` identifier references

**State:** done
**Paths:** `internal/domain/workitem.go`, `internal/domain/kind.go`, `internal/domain/task.go`, `internal/domain/template_library.go`, `internal/domain/template_reapply.go`, `internal/domain/comment.go`, `internal/domain/change_event.go`, `internal/domain/attention_level_test.go`, `internal/domain/domain_test.go`, `internal/app/*.go`, `internal/adapters/storage/sqlite/*.go`, `internal/adapters/server/mcpapi/*.go`, `internal/adapters/server/common/*.go`, `internal/adapters/server/httpapi/*.go`, `internal/tui/*.go`, `cmd/till/*.go` (any file in the 40-file surface of `rg 'WorkKind|WorkItem[^a-z]|workitem'`, excluding `drops/**`)
**Packages:** `internal/domain`, `internal/app`, `internal/adapters/storage/sqlite`, `internal/adapters/server/mcpapi`, `internal/adapters/server/common`, `internal/adapters/server/httpapi`, `internal/tui`, `cmd/till`
**Blocked by:** —
**Acceptance:**
- `rg 'WorkKind' drop/1.75/ --glob='!workflow/**'` returns 0 matches.
- `rg 'type WorkItem |WorkItemKind|WorkItemID' drop/1.75/ --glob='!workflow/**'` returns 0 matches (the table `work_items` is already `action_items`; only stale Go symbol refs die).
- `mage build` succeeds from `drop/1.75/`.
- `mage test-pkg ./internal/domain` passes.

Multi-pass `rg`+`sd` sweep. Narrow regexes per identifier: `WorkKind\b → Kind`, `WorkKindActionItem → KindActionItem`, `WorkKindSubtask → KindSubtask`, `WorkKindPhase → KindPhase`, `WorkKindDecision → KindDecision`, `WorkKindNote → KindNote`. `WorkItem`-prefixed symbols renamed to `ActionItem`-prefixed. `workitem` package filename refs kept as-is (renaming `workitem.go` is out of scope). This unit establishes the `Kind` type surface every downstream unit depends on — it is the single non-blocked unit.

### 1.2 — Delete app-layer kind-catalog seeder

**State:** done
**Paths:** `internal/app/kind_capability.go`, `internal/app/kind_capability_test.go`, `internal/app/service.go` (remove `kindBootstrap` struct field if declared there), `internal/app/service_test.go`
**Packages:** `internal/app`
**Blocked by:** 1.1
**Acceptance:**
- `rg 'ensureKindCatalogBootstrapped|defaultKindDefinitionInputs|kindBootstrap' drop/1.75/ --glob='!workflow/**' --glob='!internal/app/template_library*.go' --glob='!internal/app/template_contract*.go' --glob='!internal/app/template_reapply.go'` returns 0 matches (the `template_*.go` files are intentionally excluded per the "intentionally skip" clause below — they die wholesale in unit 1.5; dangling callers there are expected between 1.2 and 1.5).
- **`mage test-pkg ./internal/app` and `mage ci` are waived for this unit only.** The `internal/app` package is compile-broken between this unit's commit and 1.5's commit by design — callers at `internal/app/template_library.go:126`, `internal/app/template_library_builtin.go:29`, `internal/app/template_library_builtin.go:79` keep referencing `ensureKindCatalogBootstrapped` until 1.5 deletes those files wholesale. Per-unit `mage test-pkg` gate is deferred to 1.5 (see unit 1.5 Acceptance — "`mage ci` succeeds from `drop/1.75/`" is the re-green gate that discharges this waiver). Builder honors this waiver; QA does not fail the unit on app-package compile/test failure. Same pattern as the 1.4 / 1.6 waivers.

Delete `ensureKindCatalogBootstrapped` at `kind_capability.go:559-589`, its `sync.Once` struct field `kindBootstrap` (declared on `Service` — builder confirms via `LSP` before deletion), and `defaultKindDefinitionInputs` at `:863-874`. Update every caller (`resolveProjectKindDefinition` at `:592-596` and similar) to skip the bootstrap call — built-in rows live in the `CREATE TABLE kind_catalog` baked inserts after 1.3 ships. **Intentionally skip** call sites inside files destined for deletion by unit 1.5 (`internal/app/template_library_builtin.go:29, :79`, `template_library.go`, `template_contract.go`, `template_reapply.go`) — 1.5's wholesale file deletion moots them, so edits here would be pure churn. The waived Acceptance bullets above are the contract side of that "intentionally skip" choice.

### 1.3 — Bake kind_catalog rows + delete SQLite seeder + strip projects.kind schema (DDL + SQL queries + Go wrappers)

**State:** done
**Paths:** `internal/adapters/storage/sqlite/repo.go`, `internal/adapters/storage/sqlite/repo_test.go`
**Packages:** `internal/adapters/storage/sqlite`
**Blocked by:** 1.1, 1.2
**Acceptance:**
- `rg 'seedDefaultKindCatalog|mergeKindAppliesTo|kindAppliesToEqual' drop/1.75/ --glob='!workflow/**'` returns 0 matches (or only the helpers' remaining uses outside the deleted seeder).
- `rg "ALTER TABLE projects ADD COLUMN kind" drop/1.75/` returns 0 matches.
- `rg "kind TEXT.*DEFAULT 'project'" drop/1.75/internal/adapters/storage/sqlite/` returns 0 matches (per F2 — the quoted-DDL column strip).
- `rg 'kindRaw|NormalizeKindID\(p\.Kind\)|p\.Kind\s*=' drop/1.75/internal/adapters/storage/sqlite/repo.go` returns 0 matches (per Round-5 P2 — Go-wrapper strip verification).
- `rg -U 'INSERT INTO projects\([^)]*kind|UPDATE projects[^;]*kind\s*=|SELECT[^;]*kind[^;]*FROM projects' drop/1.75/internal/adapters/storage/sqlite/repo.go` returns 0 matches (per Round-5 P2 — SQL-query strip verification; `-U` for multi-line SQL literals).
- Fresh DB open produces exactly 2 rows in `kind_catalog` (`project`, `actionItem`). Verified by test `TestRepositoryFreshOpenKindCatalog` (builder adds this test to `repo_test.go`).
- Fresh DB open produces a `projects` table with **no** `kind` column. Verified by test `TestRepositoryFreshOpenProjectsSchema` (builder adds this test to `repo_test.go`) asserting `pragma_table_info('projects')` does not include `kind`.
- `mage test-pkg ./internal/adapters/storage/sqlite` passes.

Delete `seedDefaultKindCatalog` at `repo.go:1231-1301` (full range including merge/upsert block). Delete caller from `bootstrapSchema` / migration runner (builder verifies via `LSP`). Bake two `INSERT OR IGNORE INTO kind_catalog` statements for `project` + `actionItem` directly inside `CREATE TABLE kind_catalog` block (`:316`). Strip `projects.kind` at **both** schema sites: (a) delete `ALTER TABLE projects ADD COLUMN kind` at `:588` (migration hook — else every DB open post drops-rewrite re-adds the column), and (b) delete `kind TEXT NOT NULL DEFAULT 'project'` from the `CREATE TABLE IF NOT EXISTS projects (...)` block at `repo.go:152` inside `migrate()` at `:144` (the primary schema — else every fresh CI or new-user DB re-materializes the column, invalidating unit 1.14's end-state invariant). Drop `mergeKindAppliesTo` + `kindAppliesToEqual` helpers if no other caller survives after 1.1.

**Per Round-5 P2 + P4**, also strip every SQL query reference to the `kind` column in `repo.go` **plus their Go-level wrappers** so this unit's `mage test-pkg` gate stays green once the DDL column is gone (DDL-only strips would leave runtime `"no such column: kind"` failures on every project query):

- `CreateProject` at `:1345-1360` — remove `kindID := domain.NormalizeKindID(p.Kind)` + default-fallback block at `:1351-1354`; remove `kind` from INSERT column list at `:1356`; remove the `string(kindID)` positional arg at `:1358`.
- `UpdateProject` at `:1362-1383+` — same shape (remove kindID wrapper, `kind = ?` from SET clause, positional arg).
- `ensureGlobalAuthProject` at `:1455-1473` — remove `kind` from INSERT column list at `:1458` + the `string(domain.DefaultProjectKind)` positional arg at `:1465`. (P4 scope bullet.) Note: the function itself stays — it is self-healing auth-project bootstrap per project CLAUDE.md; only the column reference dies.
- `GetProject` SELECT at `:1398` — remove `kind` from the `SELECT id, slug, name, description, kind, metadata_json, created_at, updated_at, archived_at FROM projects WHERE id = ?` column list. `scanProject` handles the Scan side for both `GetProject` and `ListProjects`, but each caller holds its own SELECT literal.
- List-projects query at `:1418-1452` — remove `kind` from SELECT column list, drop `kindRaw` var + `&kindRaw` Scan handle, delete the `p.Kind = domain.NormalizeKindID(...)` + default-assignment block at `:1437-1440`.
- Second project-read query (`scanProject`) at `:3974-4000` — same shape (Scan at `:3984`, `p.Kind = ...` block at `:3990-3992`; function start is `:3974`, not `:3970` per Round-6 P2).
- **Test-site strip**: `repo_test.go:2369-2371` (the `project.SetKind("project-template", now)` call) and `:2379-2381` (the `loadedProject.Kind != domain.KindID("project-template")` assertion and its `t.Fatalf`; Round-7 P1 corrected from `:2378-2381` — the `if` statement actually starts at `:2379`). Stripped here rather than in unit 1.12 so that 1.3's `mage test-pkg ./internal/adapters/storage/sqlite` gate stays green — the `SetKind` method still exists at this unit's point (it dies in 1.6), so compiling the call is fine; only the round-trip *assertion* needs to go because the `kind` column no longer round-trips. Unit 1.12's `repo_test.go` Paths note now refers to other Project/Kind test sites only (see unit 1.12).

### 1.4 — Template libraries + node_contract_snapshots domain excision

**State:** done
**Paths:** `internal/domain/template_library.go`, `internal/domain/template_library_test.go`, `internal/domain/template_reapply.go`, `internal/domain/builtin_template_library.go`, `internal/domain/errors.go` (remove template-library error sentinels — preserve `ErrInvalidKindTemplate` per F5 classification)
**Packages:** `internal/domain`
**Blocked by:** 1.1
**Acceptance:**
- `rg 'TemplateLibrary|TemplateReapply|NodeContractSnapshot|BuiltinTemplate' drop/1.75/internal/domain/` returns 0 matches.
- `mage test-pkg ./internal/domain` passes.
- Precise error-sentinel check (per F6):
  - `rg -F 'ErrTemplateLibraryNotFound' internal/domain/errors.go` returns 0.
  - `rg 'ErrInvalidTemplate(Library|LibraryScope|Status|ActorKind|Binding)' internal/domain/errors.go` returns 0.
  - `rg 'ErrBuiltinTemplateBootstrapRequired|ErrNodeContractForbidden' internal/domain/errors.go` returns 0.
  - `rg 'ErrInvalidKindTemplate' internal/domain/errors.go` returns 1 (intentionally preserved for `kind.go:normalizeKindTemplate`, which is F5-classified as naturally unreachable but kept until refinement drop).
- **`mage build` and `mage ci` are waived for this unit only.** The workspace is compile-broken between this unit's commit and 1.5's commit by design (domain types deleted before app-layer consumers). Per-unit `mage build` gate is deferred to 1.5 (see unit 1.5 Acceptance — "`mage ci` succeeds from `drop/1.75/`" is the re-green gate that discharges this waiver). Builder honors this waiver; QA does not fail the unit on workspace-compile failure.

Delete the four files; strip template_libraries `Err*` sentinels from `errors.go` while preserving `ErrInvalidKindTemplate`. `internal/app` and `internal/adapters/*` still hold template references but they die in 1.5–1.7 before the compile gate trips (this unit only deletes domain-layer types; downstream packages won't compile until 1.5 runs — this intermediate state is expected and waived above).

### 1.5 — Template libraries app + adapter + CLI excision

**State:** todo
**Paths:** `internal/app/template_library.go`, `internal/app/template_library_builtin.go`, `internal/app/template_library_builtin_spec.go`, `internal/app/template_library_test.go`, `internal/app/template_contract.go`, `internal/app/template_contract_test.go`, `internal/app/template_reapply.go`, `internal/app/snapshot.go` (strip `TemplateLibraries` field + `snapshotTemplateLibraryFromDomain` + `upsertTemplateLibrary` + `normalizeSnapshotTemplateLibrary` sections), `internal/app/snapshot_test.go`, `internal/app/service.go` (strip template service fields + bindings), `internal/app/service_test.go`, `internal/app/ports.go` (strip the 9 `TemplateLibrary*` / `NodeContractSnapshot*` / `ProjectTemplateBinding*` methods from the unified `Repository` interface at `:24-32` — per P11, there is no standalone `TemplateLibraryRepo` port; the methods live on `Repository`), `internal/app/helper_coverage_test.go`, `internal/app/kind_capability.go` (per Round-6 F2 — strip the `library *domain.TemplateLibrary` parameter from `templateDerivedProjectAllowedKindIDs` at `:762` and `initializeProjectAllowedKinds` at `:776`; if the functions become trivial after the parameter removal, delete them and their callers instead of keeping stub shells — dead after 1.4 drops `domain.TemplateLibrary`), `internal/adapters/storage/sqlite/repo.go` (delete `migrateTemplateLifecycle` at `:1030-1055`, caller at `:650`, `backfillTemplateLibraryRevisions` at `:1057+`, `backfillProjectTemplateBindingSnapshots`; delete all `TemplateLibrary` repo methods: `ListTemplateLibraries`, `CreateTemplateLibrary`, `UpdateTemplateLibrary`, `DeleteTemplateLibrary`, `GetTemplateLibrary`, `UpsertTemplateLibrary`, and every `NodeContractSnapshot*` + `ProjectTemplateBinding*` repo method), `internal/adapters/storage/sqlite/template_library_test.go`, `internal/adapters/server/common/mcp_surface.go` (per Round-6 F5 — explicitly delete `ErrBuiltinTemplateBootstrapRequired` var + doc at `:14-15`, along with the rest of the template-surface re-exports the file carries), `internal/adapters/server/common/app_service_adapter.go` (strip `errors.Is(err, domain.ErrBuiltinTemplateBootstrapRequired)` branch + `errors.Join(ErrBuiltinTemplateBootstrapRequired, err)` wrap at `:597-598`), `internal/adapters/server/common/app_service_adapter_mcp.go`, `internal/adapters/server/common/app_service_adapter_auth_context.go`, `internal/adapters/server/common/app_service_adapter_auth_context_test.go`, `internal/adapters/server/common/app_service_adapter_mcp_actor_attribution_test.go`, `internal/adapters/server/common/app_service_adapter_lifecycle_test.go`, `internal/adapters/server/common/app_service_adapter_helpers_test.go` (per Round-6 F1 — strip the test table entry referencing `domain.ErrBuiltinTemplateBootstrapRequired` + `ErrBuiltinTemplateBootstrapRequired` re-export at `:259`), `internal/adapters/server/mcpapi/handler.go` (delete `pickTemplateLibraryService` at `:1046-1054` + call at `:66, :72, :86`; strip `errors.Is(err, common.ErrBuiltinTemplateBootstrapRequired)` branch at `:855`), `internal/adapters/server/mcpapi/handler_test.go` (per Round-6 F1 — strip the test case that constructs `errors.Join(common.ErrBuiltinTemplateBootstrapRequired, ...)` at `:938`), `internal/adapters/server/mcpapi/extended_tools.go` (delete `till.bind_project_template_library` at `:2171`, `till.get_template_library` at `:2258`, `till.upsert_template_library` at `:2281`, and every `operation=ensure_builtin` / `"ensure_builtin"` branch on `till.template` at `:2085` — per Round-6 P1, `ensure_builtin` is an operation on the unified `till.template` tool, not a standalone `till.ensure_builtin_template_library` tool name, plus all `TemplateLibraryID` argument handling at `:435, :457, :595, :604, :840, :853`), `internal/adapters/server/mcpapi/extended_tools_test.go`, `internal/adapters/server/mcpapi/instructions_tool.go` (strip `TemplateLibraryID` field + arg), `internal/adapters/server/mcpapi/instructions_tool_test.go`, `internal/adapters/server/mcpapi/instructions_explainer.go` (strip `template_library_description` + template focus branch at `:296, :338`), `internal/adapters/server/mcpapi/handler_integration_test.go`, `internal/adapters/server/httpapi/handler.go` (per Round-6 F1 — strip `errors.Is(err, common.ErrBuiltinTemplateBootstrapRequired)` branch at `:425`), `internal/adapters/server/httpapi/handler_integration_test.go`, `internal/tui/model.go` (strip any `TemplateLibrary` readbacks — F7 says none; builder verifies), `internal/tui/model_test.go`, `internal/tui/thread_mode.go`, `cmd/till/template_cli.go` (delete entire file), `cmd/till/template_builtin_cli_test.go` (delete entire file), `cmd/till/main.go` (strip template-cli command registration), `cmd/till/main_test.go`
**Packages:** `internal/app`, `internal/adapters/storage/sqlite`, `internal/adapters/server/common`, `internal/adapters/server/mcpapi`, `internal/adapters/server/httpapi`, `internal/tui`, `cmd/till`
**Blocked by:** 1.1, 1.2, 1.3, 1.4
**Acceptance:**
- `rg 'TemplateLibrary|TemplateReapply|NodeContractSnapshot|BuiltinTemplate|node_contract_snapshot|template_librar|template_node_template|template_child_rule|project_template_binding' drop/1.75/ --glob='!workflow/**' --glob='!scripts/drops-rewrite.sql'` returns 0 matches.
- `mage ci` succeeds from `drop/1.75/`. **This unit carries the workspace-compile-restoration burden** — the 1.4 waiver expects 1.5 to re-green the workspace via this check.
- MCP tools `till.bind_project_template_library`, `till.get_template_library`, `till.upsert_template_library` absent from registered tools (verify via `rg 'till\.(bind_project_template_library|get_template_library|upsert_template_library)' internal/adapters/server/mcpapi/` returns 0). Per Round-6 P1, `ensure_builtin` was an `operation` on the unified `till.template` tool, not a standalone tool name — verify via `rg '"ensure_builtin"|"bind_project_template_library"|"get_template_library"|"upsert_template_library"' internal/adapters/server/mcpapi/` returns 0 (catches the operation-string literals on `till.template` if any residue remains).

The big atomic excision. Must be one unit because MCP imports app, which imports domain — splitting by package leaves intermediate compile-broken states. Builder deletes in bottom-up order (sqlite → app → common → mcpapi → httpapi → tui → cmd) so each package compiles at the end of its sub-pass. `snapshot.go` is the trickiest surface: strip the `TemplateLibraries` field from `Snapshot` struct and every reference to template serialization in validation / upsert / normalize helpers.

### 1.6 — Drop `projects.kind` column (domain Go + app + MCP + TUI + CLI)

**State:** todo
**Paths:** `internal/domain/project.go` (strip `Kind KindID` field from `type Project` at `:16`; delete the `Kind: DefaultProjectKind,` entry in `NewProject`'s struct literal at `:60`; delete the entire `SetKind(kind KindID, ...)` method at `:79-88`; audit every `Project{...}` struct literal sitewide to remove `Kind:` field references — per F5 the bare `:85` citation was ambiguous), `internal/app/kind_capability.go` (strip `resolveProjectKindDefinition` + callers), `internal/app/service.go` (strip project.Kind references), `internal/app/snapshot.go` (per Round-5 P1 — strip `Kind domain.KindID` field from `type SnapshotProject` at `:41`, strip the `Projects[i].Kind` normalization loop at `:395-397` which reads `domain.Project.Kind`, strip `Kind: p.Kind` from `snapshotProjectFromDomain` at `:1230-1237`, strip the `kind := domain.NormalizeKindID(p.Kind)` + `Kind: kind` lines from `SnapshotProject.toDomain` at `:1589-1603`; also bump `SnapshotVersion` from `tillsyn.snapshot.v4` to `tillsyn.snapshot.v5` at `:16` — schema-honest signal for strict consumers even though JSON round-trip is soft-compatible), `internal/app/snapshot_test.go` (update fixtures and any golden test data pinning the `tillsyn.snapshot.v4` literal to `tillsyn.snapshot.v5`), `internal/app/template_reapply.go` (strip — partly duplicated w/ 1.5 deletion), `internal/adapters/server/mcpapi/instructions_explainer.go` (strip project.Kind readback), `internal/tui/model.go` (strip `project.Kind` readbacks at `:4856, :18747` and `projectFieldKind` form input if present), `internal/tui/thread_mode.go`, `cmd/till/project_cli.go`, `cmd/till/project_cli_test.go`
**Packages:** `internal/domain`, `internal/app`, `internal/adapters/server/mcpapi`, `internal/tui`, `cmd/till`
**Blocked by:** 1.1, 1.2, 1.3, 1.4, 1.5
**Acceptance:**
- `rg -U 'project\.Kind|projects\.kind|Project\{[^}]*Kind' drop/1.75/ --glob='!workflow/**' --glob='!scripts/drops-rewrite.sql'` returns 0 matches (`-U` enables multi-line matching so `Project{...\n...Kind:...}` struct literals are caught, not just same-line hits).
- `rg 'projectFieldKind' drop/1.75/` returns 0 matches.
- `rg 'tillsyn\.snapshot\.v4' drop/1.75/internal/app/` returns 0 matches (SnapshotVersion bumped to v5 per the schema-change acceptance).
- `rg 'tillsyn\.snapshot\.v5' drop/1.75/internal/app/snapshot.go` returns exactly 1 match (the `const SnapshotVersion` line at `:16`).
- **`mage build` and `mage ci` are waived for this unit only.** The workspace is compile-broken between this unit's commit and units 1.11 / 1.12 / 1.13's commits by design (domain `Project.Kind` field + `SetKind` method deleted before test-site updates in `internal/app` / `sqlite` / `mcpapi` / `tui` / `cmd/till` packages). Per-unit `mage build` gate is deferred to units 1.11 (app), 1.12 (sqlite + mcpapi + httpapi + common), and 1.13 (tui + cmd/till) (see their Acceptance — `mage test-pkg` on each affected package is the per-package re-green gate that discharges this waiver). Builder honors this waiver; QA does not fail the unit on workspace-compile failure.

The column strip is deferred to this unit (after 1.5) because `internal/tui/model.go` and `cmd/till/project_cli.go` hold `project.Kind` readbacks the template system touches — ordering them before template excision risks re-introducing compile cycles. The `Project.Kind` field + `NewProject` kind arg + `SetKind` method deletion is **intentional dead-code removal — no behavior change**; downstream test-site references in `internal/app/snapshot_test.go`, `internal/adapters/server/mcpapi/extended_tools_test.go:98`, `internal/tui/model_test.go:15199-15207`, and `cmd/till/project_cli_test.go` are the only remaining consumers and get stripped in units 1.11 (app), 1.12 (mcpapi), and 1.13 (tui + cmd/till). The `repo_test.go:2368-2381` assertion migrated out of 1.12 into unit 1.3 (per Round-5 P2 — 1.3's own `mage test-pkg` gate required the test-site strip to stay green).

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
**Paths:** `internal/app/kind_capability_test.go`, `internal/app/service_test.go`, `internal/app/snapshot_test.go` (per Round-5 P1 — strip any `SnapshotProject.Kind` / `domain.Project{...Kind:}` / `domain.KindID("project-template")` round-trip assertions that reference the deleted fields), `internal/app/helper_coverage_test.go`
**Packages:** `internal/app`
**Blocked by:** 1.2, 1.5, 1.6
**Acceptance:**
- `mage test-pkg ./internal/app` passes. **This unit carries part of unit 1.6's workspace-compile-restoration burden** for the `internal/app` package — test-site references to the deleted `Project.Kind` + `SnapshotProject.Kind` + `SetKind` must be removed here so this package compiles + tests. It also discharges unit 1.2's `mage test-pkg ./internal/app` waiver transitively (the 1.2 waiver was held open against 1.5's `mage ci` restoration, which this unit's per-package test-pkg confirms at the app-package grain).
- Coverage for `internal/app` ≥ 70% (project CLAUDE.md Build Verification rule).
- `rg 'WorkKind|TemplateLibrary|ensureKindCatalogBootstrapped|SnapshotProject\{[^}]*Kind|SetKind' internal/app/*_test.go` returns 0 matches.
- `kind_capability_test.go` cases that assert non-empty `AutoCreateChildren` are either rewritten to assert empty (matching the F5 `KindTemplate` classification — post-collapse `kind_catalog` rows carry empty children) or deleted if purely template-library-coupled.

Per P10: `search_embeddings_test.go` and `embedding_runtime_test.go` are **not** in Paths — verified 0 hits for `WorkKind`/`TemplateLibrary`/`ensureKindCatalogBootstrapped` and no edits needed.

### 1.12 — Adapter + MCP test updates

**State:** todo
**Paths:** `internal/adapters/storage/sqlite/repo_test.go` (any residual `Project.Kind` / `SetKind` test-site references other than `:2368-2381`, which was migrated into unit 1.3 per Round-5 P2 — 1.3's own test-pkg gate required the strip), `internal/adapters/storage/sqlite/embedding_jobs_test.go`, `internal/adapters/storage/sqlite/embedding_lifecycle_adapter_test.go`, `internal/adapters/storage/sqlite/handoff_test.go`, `internal/adapters/server/mcpapi/handler_integration_test.go`, `internal/adapters/server/mcpapi/extended_tools_test.go` (also strips `Project.Kind` test-site reference at `:98` per unit 1.6 waiver discharge), `internal/adapters/server/mcpapi/instructions_tool_test.go`, `internal/adapters/server/httpapi/handler_integration_test.go`, `internal/adapters/server/common/app_service_adapter_mcp_actor_attribution_test.go`, `internal/adapters/server/common/app_service_adapter_lifecycle_test.go`, `internal/adapters/server/common/app_service_adapter_auth_context_test.go`
**Packages:** `internal/adapters/storage/sqlite`, `internal/adapters/server/mcpapi`, `internal/adapters/server/httpapi`, `internal/adapters/server/common`
**Blocked by:** 1.3, 1.5, 1.6, 1.7
**Acceptance:**
- `mage test-pkg ./internal/adapters/storage/sqlite` passes. **This unit carries part of unit 1.6's workspace-compile-restoration burden** for the `sqlite` package — any residual test-site references to the deleted `Project.Kind` + `SetKind` (other than `:2368-2381`, handled in 1.3) must be removed here so this package compiles.
- `mage test-pkg ./internal/adapters/server/mcpapi` passes. **This unit carries part of unit 1.6's workspace-compile-restoration burden** for the `mcpapi` package — test-site references to the deleted `Project.Kind` must be removed here so this package compiles.
- `mage test-pkg ./internal/adapters/server/httpapi` passes.
- `mage test-pkg ./internal/adapters/server/common` passes.
- `rg 'WorkKind|TemplateLibrary|template_librar|bridgeLegacyActionItems|seedDefaultKindCatalog|FROM tasks|projects\.kind|project\.Kind|SetKind' drop/1.75/internal/adapters/ --glob='*_test.go'` returns 0 matches.

### 1.13 — TUI + CLI test updates

**State:** todo
**Paths:** `internal/tui/model_test.go` (also strips `Project.Kind` / `SetKind` test-site references at `:15199-15207` per unit 1.6 waiver discharge), `internal/tui/thread_mode_test.go`, `internal/tui/model_teatest_test.go`, `internal/tui/description_editor_mode.go` (strip dead WorkKind refs if any), `cmd/till/main_test.go`, `cmd/till/project_cli_test.go` (also strips any residual `Project.Kind` test-site references per unit 1.6 waiver discharge — unit 1.6 already touches `project_cli_test.go` for production-side readbacks, this unit closes out the test-fixture side)
**Packages:** `internal/tui`, `cmd/till`
**Blocked by:** 1.5, 1.6
**Acceptance:**
- `mage test-pkg ./internal/tui` passes. **This unit carries part of unit 1.6's workspace-compile-restoration burden** for the `tui` package — test-site references to the deleted `Project.Kind` + `SetKind` must be removed here so this package compiles.
- `mage test-pkg ./cmd/till` passes. **This unit carries part of unit 1.6's workspace-compile-restoration burden** for the `cmd/till` package — any residual test-site references to the deleted `Project.Kind` must be removed here so this package compiles.
- `mage test-golden` passes (goldens unchanged per F7 — regenerate via `mage test-golden-update` only if TUI render drift is proven).
- `rg 'WorkKind|TemplateLibrary|project\.Kind|projectFieldKind|SetKind' drop/1.75/internal/tui/ drop/1.75/cmd/till/` returns 0 matches.

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
  - `SELECT COUNT(*) FROM sqlite_master WHERE name LIKE 'node_contract_%'` returns 0 (per Round-5 O1 — `template_%` catches 5 of 9 tables; `node_contract_%` covers the snapshot cluster).
  - `SELECT COUNT(*) FROM sqlite_master WHERE name = 'project_template_bindings'` returns 0 (per Round-5 O1 — neither `template_%` nor `node_contract_%` matches this exact name).
  - `SELECT COUNT(*) FROM sqlite_master WHERE name = 'tasks'` returns 0.
  - `SELECT COUNT(*) FROM pragma_table_info('projects') WHERE name = 'kind'` returns 0.
  - `SELECT COUNT(*) FROM action_items WHERE kind NOT IN ('project','actionItem') OR kind IS NULL` returns 0 (per Round-5 O2 — SQL 3-valued logic: a row with `kind IS NULL` silently passes a bare `NOT IN` check; the `OR kind IS NULL` clause closes that gap).
  - `SELECT COUNT(*) FROM project_allowed_kinds WHERE kind_id NOT IN ('project','actionItem') OR kind_id IS NULL` returns 0 (per Round-6 F3 Option A — catches any allowlist row pointing at a deleted legacy kind; if the assertion fires, the script aborts cleanly via `RAISE(ROLLBACK)` and dev handles the residue manually).
- Rollback on assertion failure (via `BEGIN TRANSACTION` + `SELECT RAISE(ROLLBACK, ...)` guards).

**DEV REMINDER AT RUN-TIME (Round-6 F3):** After WORKFLOW.md Phase 6 Verify completes (CI green on `drop/1.75`) and before Phase 7 Closeout begins — specifically, before the dev executes `scripts/drops-rewrite.sql` against `~/.tillsyn/tillsyn.db` as part of the drop-end sequence — the orchestrator MUST re-surface the F3 decision and the three options considered so the dev can reconsider with fresh context. The chosen path is **Option A (assert-only)**. Restate the trade-off verbatim:

- **Option A (chosen, encoded above):** assert `project_allowed_kinds` has no legacy-kind rows post-delete; if it does, `RAISE(ROLLBACK)` aborts and dev handles the miss manually. Cheapest, diagnostic-first.
- **Option B (not chosen):** re-seed via `INSERT OR IGNORE INTO project_allowed_kinds (project_id, kind_id) SELECT id, 'actionItem' FROM projects` after Phase 3. Guarantees every project keeps an allowlist row post-collapse, but adds a behavior-change that is outside the current plan's pure-collapse scope.
- **Option C (not chosen):** both A and B — belt-and-suspenders.

**Trigger (Round-7 F4 anchor):** surface this callout after Phase 6 Verify completes (CI green on `drop/1.75`) and before Phase 7 Closeout begins — the drop-orch's final act before closeout is to present this reminder. The dev then runs the one-shot `sqlite3 ~/.tillsyn/tillsyn.db < scripts/drops-rewrite.sql` step against the real dev DB. The orchestrator cross-references the project memory `project_drop_1_75_unit_1_14_f3_decision.md` for the rationale and triggers the reminder on any mention of running this script. Dev asked for the re-prompt explicitly on 2026-04-18; honor it even if the decision looks settled.

Schema-only collapse replacing the current 296-line multi-phase script. Phases: (1) pre-flight counts, (2) `DROP TABLE` the template cluster (9 tables per F9) — **ordered before the kind_catalog row delete per Round-5 editorial note so that any FK from `template_node_templates.node_kind_id` / `template_child_rules.child_kind_id` to `kind_catalog.id` cannot trip `ON DELETE RESTRICT` if dev DB happens to run with `PRAGMA foreign_keys = ON`**, (3) `DELETE FROM kind_catalog WHERE id NOT IN ('project', 'actionItem')` (safe after template_* tables are gone), (4) `ALTER TABLE projects DROP COLUMN kind;` — **per Round-7 F2, use SQLite's native `DROP COLUMN` (available since SQLite 3.35.0 / March 2021). The dev runs `drops-rewrite.sql` via their local `sqlite3` CLI (3.51.0), not via the Go binary. The Go binary uses `github.com/ncruces/go-sqlite3 v0.23.3` as driver, with `github.com/asg017/sqlite-vec-go-bindings/ncruces v0.1.6` blank-imported at `internal/adapters/storage/sqlite/repo.go:16` — the sqlite-vec binding replaces the ncruces/go-sqlite3 embed (per `~/go/pkg/mod/github.com/asg017/sqlite-vec-go-bindings@v0.1.6/README.md:85` "do NOT include them both"), shipping its own embedded SQLite WASM well past the 3.35.0 floor. Dev runs `drops-rewrite.sql` via local `sqlite3 3.51.0` CLI, also well past the floor. Use the native form instead of the 12-step `CREATE NEW + INSERT-SELECT + DROP OLD + RENAME` rebuild. `projects.kind` is a plain column with no PK / UNIQUE / FK / index / trigger / view dependencies so the native form works directly. This eliminates the Round-6 F4 `PRAGMA foreign_keys = OFF/ON` wrapper entirely — which was unsound anyway, because `PRAGMA foreign_keys` inside an open `BEGIN TRANSACTION` is a silent no-op per SQLite docs. The Round-7 falsification F1 blocker (CASCADE through 17 child tables from `DROP TABLE projects_old` with FK enforced) is dissolved because no table rebuild happens.**, (5) `DROP TABLE tasks`, (6) `UPDATE action_items SET kind='actionItem', scope='actionItem'`, (7) assertion block. PHASE 6 role hydration + PHASE 7 multi-kind rewrite from the main-branch script are deleted outright (dev-DB cleanup 2026-04-18 removed every row they'd touch).

### 1.15 — Drop-end verification (`mage ci` + push + CI watch)

**State:** todo
**Paths:** — (verification-only, no code edits)
**Packages:** — (workspace-wide)
**Blocked by:** 1.14
**Acceptance:**
- `mage ci` succeeds from `drop/1.75/` (format, vet, test, build, lint — per CLAUDE.md § Build Verification).
- `git push` + `gh run watch --exit-status` returns green on branch `drop/1.75`.
- Coverage ≥ 70% workspace-wide.
- `rg 'WorkKind|TemplateLibrary|template_librar|node_contract_snapshot|seedDefaultKindCatalog|ensureKindCatalogBootstrapped|bridgeLegacyActionItems|migratePhaseScopeContract|migrateTemplateLifecycle|projects\.kind|project\.Kind|FROM tasks' drop/1.75/ --glob='!workflow/**' --glob='!scripts/drops-rewrite.sql'` returns 0 matches (end-state invariant).
- Quoted-DDL guard (per F2 — catches schema substrings inside Go string literals): `rg "kind TEXT.*DEFAULT 'project'" drop/1.75/ --glob='!workflow/**' --glob='!scripts/drops-rewrite.sql'` returns 0 matches.

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
- **MCP tool deregistration surface.** At least 4 MCP tools in `extended_tools.go`: `till.bind_project_template_library` (`:2171`), `till.get_template_library` (`:2258`), `till.upsert_template_library` (`:2281`), `till.ensure_builtin_template_library` (`:2085`). All die, along with their test coverage in `extended_tools_test.go:1679+`. `instructions_tool.go` also exposes `TemplateLibraryID` arg (`:65, :126, :146, :300, :318`) — dies. `handler.go` wiring at `:86, :1046-1054` dies. `common/mcp_surface.go` dies.
- **Goldens are not affected.** The 4 `.golden` files in `internal/tui/testdata/` do not reference `template_librar` / `node_contract_snapshot` / `WorkKindPhase` / `WorkKindSubtask` / `projects.kind` per `grep`. Regeneration via `mage test-golden-update` only if TUI render output changes after F11 hits are removed.
- **`projects.kind` column is user/orch-facing metadata, not a system tracking concern.** Projects get template suggestions from the user or orch at creation and across the project's life; the system doesn't need to persist a kind. Column gets dropped along with every Go reference, MCP handler filter, and TUI view that reads it.
- **F10 package-level blocker contract.** `internal/domain` has 5 in-drop units (Go identifier rename 1.1, template_libraries domain excision 1.4, projects.kind strip 1.6, action_item.go filename 1.8, kind.go merge 1.9) that share the package compile. Per `CLAUDE.md` §Blocker Semantics, sibling units sharing a package require explicit `blocked_by`. The Planner section above serializes these five via a linear `blocked_by` chain. `internal/app` units (1.2 / 1.5 / 1.7) similarly chain within the app package.
- `scripts/drops-rewrite.sql` on `main` currently rewrites every non-project node's `kind` to `drop` (Drop-2 vocabulary) and hydrates 8 role variants from legacy kinds. Drop 1.75's builder **replaces** that script with a schema-only collapse against the post-rename `action_items` table: `kind_catalog` → `{project, actionItem}`, `template_librar*` / `node_contract_snapshot*` / `template_node_template*` / `template_child_rule*` wipe, `ALTER TABLE projects DROP COLUMN kind`, drop legacy `tasks` table, one-line `UPDATE action_items SET kind='actionItem', scope='actionItem'`, assert `kind_catalog` row count = 2 plus 4 more end-state invariants. PHASE 6 role hydration and PHASE 7 multi-kind rewrite from the old script are deleted.
- Dev manually applies `scripts/drops-rewrite.sql` against `~/.tillsyn/tillsyn.db` at drop end (same pattern as `scripts/rename-task-to-actionitem.sql`). Agents never touch the dev DB.
- **One drop, not split.** All eight in-scope items above ship in this drop across 15 atomic units (1.1–1.15). The planner decomposed into units with `blocked_by` wiring (package-level serialization enforced in `internal/domain` + `internal/app`; `scripts/drops-rewrite.sql` is a sink blocking on every code unit; `mage ci` gate is the final drop-end invariant).
