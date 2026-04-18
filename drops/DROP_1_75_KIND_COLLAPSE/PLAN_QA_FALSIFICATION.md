# DROP_1_75_KIND_COLLAPSE — Plan QA Falsification (Round 2)

**Verdict:** fail

Scope now covers the 5 locked decisions coherently, but at least 9 hidden dependencies and cross-file coupling gaps would make the plan break `mage ci` on Day 1 of Phase 4 if executed verbatim. The Round 1 QA PROOF already flagged the Kind vs KindID collision and the seedDefaultKindCatalog line-range drift; those remain unresolved. This round adds 7 new counterexamples the planner must incorporate into the Phase 1 unit breakdown before Phase 3 cleanup.

All counterexamples below are grounded to concrete `file:line` evidence inside `drop/1.75/` HEAD.

## Findings

### F1. A second, parallel kind-catalog seeder exists in `internal/app` — plan only deletes one of two

- **Claim under attack:** PLAN Scope item 1: "Delete the `seedDefaultKindCatalog` function at `internal/adapters/storage/sqlite/repo.go:1231-1247`" is treated as THE seeder deletion.
- **Counterexample:** There is a SECOND seeder. `internal/app/kind_capability.go:863-874` defines `defaultKindDefinitionInputs()` returning the SAME 7 legacy kinds (`project`, `actionItem`, `subtask`, `phase`, `branch`, `decision`, `note`). `ensureKindCatalogBootstrapped(ctx)` at `kind_capability.go:559-589` iterates this list and `repo.CreateKindDefinition(ctx, kind)` for any missing entry. This bootstrap is called from:
  - `kind_capability.go:99` (`ListKindDefinitions`)
  - `kind_capability.go:161` (`SetProjectAllowedKinds`)
  - `kind_capability.go:593` (`resolveProjectKindDefinition`)
  - `kind_capability.go:636` (`resolveActionItemKindDefinition`)
  - `service.go:192` and `service.go:244`
  - `template_library.go:126`
  - `template_library_builtin.go:29` and `:79`
- **Impact:** If the planner follows PLAN literally and only deletes the `repo.go` seeder + bakes `{project, actionItem}` into the initial schema, the app-layer seeder will happily re-seed `subtask`, `phase`, `branch`, `decision`, `note` into `kind_catalog` on the NEXT service startup when the catalog has anything missing. The `kind_catalog_only_project_and_drop = 2` assertion in `drops-rewrite.sql` will pass on the dev DB, and then the next `till` boot silently re-hydrates 5 more rows. Counterexample is reproducible: drop the repo-layer seeder, boot the binary, call `ListKindDefinitions` once — catalog has 7 rows again.
- **Mitigation:** Planner Unit N.X MUST enumerate BOTH seed surfaces and delete both in the same unit (or serialize with `blocked_by` so the app-layer seeder dies before the repo-layer seeder). `kind_capability_test.go:996` also references `defaultKindDefinitionInputs()` and must be updated.

### F2. `migrateTemplateLifecycle` runs on every DB open and will crash after `template_libraries` is dropped

- **Claim under attack:** PLAN Scope item 4: "delete every file/symbol under the `template_library*`... surface". Plan enumerates 41 files but says nothing about `initSchema` migration hooks that `ALTER TABLE template_libraries`.
- **Counterexample:** `internal/adapters/storage/sqlite/repo.go:650` calls `r.migrateTemplateLifecycle(ctx)` during `initSchema`. The migration body at `repo.go:1030-1055` executes 10 `ALTER TABLE` statements against `template_libraries` and `project_template_bindings`. If the unit that excises `template_libraries` drops those tables (or even the `CREATE TABLE` statements for them) but leaves `migrateTemplateLifecycle` intact in the initSchema call chain, the NEXT DB open raises `no such table: template_libraries`. `backfillTemplateLibraryRevisions` (`repo.go:1057-1097`) and `backfillProjectTemplateBindingSnapshots` (`repo.go:1099+`) have the same shape.
- **Impact:** `mage ci` broken (all tests that touch `Open(dbPath)` fail); dev binary broken; `drops-rewrite.sql` can't run because the Go binary won't start to prepare the DB.
- **Mitigation:** Plan must explicitly enumerate and remove `r.migrateTemplateLifecycle(ctx)` call + function + `backfillTemplateLibraryRevisions` + `backfillProjectTemplateBindingSnapshots` + their test callers, and do so in the same unit that removes the `CREATE TABLE template_libraries` + `CREATE TABLE project_template_bindings` statements.

### F3. `bridgeLegacyActionItemsToWorkItems` is not the only code path touching the `tasks` table — 26 references remain

- **Claim under attack:** PLAN Notes line 47: "Dropping the table removes the shim's reason to exist" — implying the `tasks` table deletion is a single-bullet operation gated on the bridge removal.
- **Counterexample:** `grep "tasks\b" internal/adapters/storage/sqlite/repo.go | wc -l` → 26 hits. Enumerated:
  - `repo.go:169` — `CREATE TABLE IF NOT EXISTS tasks (...)` (initSchema creates it every open)
  - `repo.go:558` — `CREATE INDEX ... ON tasks(project_id, column_id, position)`
  - `repo.go:592-604` — 13 `ALTER TABLE tasks` idempotent migration statements
  - `repo.go:665` — `CREATE INDEX ... ON tasks(project_id, parent_id)`
  - `repo.go:717` — `UPDATE tasks SET scope = ... WHERE scope = 'subphase'` (inside `migratePhaseScopeContract`)
  - `repo.go:732-734` — `UPDATE tasks SET scope = ? WHERE kind = ? AND scope = ?`
  - `repo.go:977-978` — `UPDATE tasks SET created_by_name ...` / `updated_by_name ...` (inside `migrateActionItemActorNames`)
  - `repo.go:1218` — `FROM tasks t` (inside the bridge)
  - `repo_test.go:1006-1033` — a TEST FIXTURE that `CREATE TABLE tasks` + `INSERT INTO tasks` + `Open(dbPath)` to verify the bridge. Deleting the bridge without deleting this test leaves a test that creates an orphan `tasks` table and then asserts migration behavior.
- **Impact:** Dropping only the bridge while leaving any of the other 24 `tasks` references produces either (a) a dead `tasks` table re-created on every open (if `CREATE TABLE` survives), (b) a build failure on test files referencing `CREATE TABLE tasks`, or (c) a migration path that runs `UPDATE tasks SET ...` against a non-existent table and crashes at startup. The plan reads like "delete one function" when it is actually "delete 26 call sites across init-schema + migrations + indexes + tests".
- **Mitigation:** Planner Unit N.X enumeration MUST include all 26 `tasks` references. The test at `repo_test.go:1006-1049` that tests the bridge pathway must be deleted or rewritten to target `action_items` directly.

### F4. `ALTER TABLE action_items ADD COLUMN kind ...` migration statements are IDEMPOTENT HISTORY — not safe to delete outright

- **Claim under attack:** PLAN Scope item 6: "`scripts/drops-rewrite.sql` = schema-only... rewritten against the `action_items` (post-rename) table".
- **Counterexample:** `repo.go:611-625` contains 13 `ALTER TABLE action_items` statements that run on EVERY DB open and add columns with `IF NOT EXISTS`-style idempotence via `isDuplicateColumnErr` catch. These exist because the `action_items` table was renamed from `work_items` and those columns had to be back-compatibly added to preserve older installs. If the planner's "bake into initial schema migration" for item 1 also bakes these 13 columns into the `CREATE TABLE action_items` statement AND removes the `ALTER TABLE` idempotence block, then a fresh install works, but an UPGRADE from a dev DB sitting on an older `action_items` CREATE (pre these columns) silently falls through — the `CREATE TABLE IF NOT EXISTS action_items` preserves the old shape, no ALTER runs, and column-missing errors surface at first write.
- **Impact:** Silent schema drift on upgrade. Dev DB on 2026-04-18 cleanup (Notes line 43) was already purged; no upgrade risk in the immediate dev flow. But any fresh repo clone that has a `tillsyn.db` from an earlier drop hits this. CI DBs (created fresh) are fine. The risk is localized but not-documented.
- **Mitigation:** Planner must explicitly document in the unit description: are we (a) keeping the ALTER idempotence block because dev DBs exist from before, or (b) removing it because Drop 1.75 assumes fresh-install semantics only. PLAN currently says neither.

### F5. `AuthRequestPathKind`, `CapabilityScopePhase/Subtask/Branch`, and `KindAppliesToBranch/Phase/Subtask` survive kind_catalog collapse — implicit dual-vocabulary debt

- **Claim under attack:** PLAN out-of-scope line 28 implies `metadata.role` hydration and cascade dispatcher are Drop 2/4+. PLAN Scope says `kind_catalog` collapses to `{project, actionItem}`.
- **Counterexample:** Three separate `branch/phase/subtask` vocabulary surfaces survive independent of `kind_catalog`:
  - `internal/domain/kind.go:22-28` — `KindAppliesToProject/Branch/Phase/ActionItem/Subtask` (5 values). After collapse, `branch/phase/subtask` are still valid `applies_to` values on `KindDefinition` rows, but no kind in the catalog uses them.
  - `internal/domain/kind.go:40-45` — `validWorkItemAppliesTo = []KindAppliesTo{Branch, Phase, ActionItem, Subtask}`. Still matches work-item rows.
  - `internal/domain/capability.go` (16 file match on `CapabilityScope*`) — `CapabilityScopeBranch/Phase/Subtask/ActionItem` are mutation-guard scope values used by `capabilityScopeTypeForActionItem` at `kind_capability.go:409-423`, which switches on `actionItem.Scope` ∈ {`Project`, `Branch`, `Phase`, `Subtask`, default=`ActionItem`}.
  - `internal/domain/auth_request.go:33-49` — `AuthRequestPathKind` is `Project/Projects/Global` (a different kind vocabulary entirely). The memory note ("Auth Path Branch Quirk Pre-Drop-2") confirms live auth uses `/branch/<drop-id>` and lease `scope_type: branch` TODAY and that this is "revisit at Drop 2."
- **Impact:** Plan does not clarify which of these vocabularies die in Drop 1.75 vs survive. If the builder interprets "collapse to {project, actionItem}" as "delete `KindAppliesToBranch/Phase/Subtask` constants" (a natural read of `WorkKind → Kind` item 3), the live auth path `/branch/<drop-id>` immediately fails validation at `auth_request.go:250-280` and all drop-scoped auth breaks. If the builder leaves them in, but an agent or test picks a kind of e.g. `phase`, `resolveActionItemKindDefinition` at `kind_capability.go:655` fails with `KindNotFound` because `phase` is gone from `kind_catalog`. Either reading is wrong.
- **Mitigation:** Plan MUST explicitly state: (a) which of the three vocabularies survive (answer per memory note and per `capabilityScopeTypeForActionItem`: `KindAppliesTo*` and `CapabilityScope*` MUST survive for auth scope to work pre-Drop-2; `AuthRequestPathKind` is untouched), and (b) what happens when a test tries to create an actionItem with `kind='phase'` — answer: kind_capability.go returns `KindNotFound`. All tests using `phase/subtask/decision/note` kinds (enumerated in F6 below) must be updated or deleted.

### F6. Test suite contains hundreds of references to the doomed kinds — identifier rename is not enough

- **Claim under attack:** PLAN Scope item 7: "update every domain/app/adapter test, every MCP integration test, every fixture, every golden file that references a non-{`project`,`actionItem`} kind". Listed as a single bullet.
- **Counterexample:** Grep for `KindAppliesToBranch|KindAppliesToPhase|KindAppliesToSubtask|WorkKindSubtask|WorkKindPhase|WorkKindDecision|WorkKindNote` across the worktree (excluding `drops/`): 373 occurrences across 34 files. Partial enumeration of test callers:
  - `internal/domain/kind_capability_test.go:4 hits` — tests asserting applies-to validation with phase/subtask
  - `internal/domain/attention_level_test.go:3 hits` — scope-level tests
  - `internal/app/template_contract_test.go:5 hits` — template-contract tests (dies with templates anyway per F2 scope)
  - `internal/app/snapshot_test.go:6 hits` — snapshot tests for phase/subtask promotion
  - `internal/app/service_test.go:20 hits` — end-to-end service tests
  - `internal/app/kind_capability_test.go:16 hits` — kind-capability unit tests (tests the very logic PLAN is touching)
  - `internal/app/helper_coverage_test.go` — in the match list
  - `internal/adapters/storage/sqlite/repo_test.go:9 hits` — storage persistence tests with phase/subtask rows
  - `internal/tui/model_test.go:92 hits` — TUI tests with `subtask` show/hide behavior
  - `internal/adapters/server/mcpapi/extended_tools_test.go:5 hits` — MCP tool integration tests
  - `internal/adapters/server/mcpapi/handler_integration_test.go:3 hits`
  - `internal/adapters/server/common/app_service_adapter_lifecycle_test.go:29 hits`
  - `internal/adapters/server/common/app_service_adapter_auth_context_test.go:4 hits`
- **Impact:** Bullet 7 in PLAN is a single one-liner but the underlying work is hundreds of test-file updates. A builder hitting a "Phase 4 unit: update tests" with scope = "every test touching doomed kinds" will collide with tests the planner didn't identify — a unit that claims `internal/domain/*_test.go` will also have to touch `internal/app/kind_capability_test.go` (because the kind-capability unit test calls `defaultKindDefinitionInputs()`) and `internal/adapters/storage/sqlite/repo_test.go` (same test file that tests the deleted bridge).
- **Mitigation:** Planner MUST (a) enumerate a per-file test update matrix, or (b) split item 7 into per-package units (`internal/domain` tests, `internal/app` tests, `internal/adapters/storage/sqlite` tests, `internal/adapters/server/*` tests, `internal/tui` tests, `cmd/till` tests), and wire `blocked_by` so tests update AFTER the symbol rename lands.

### F7. JSON template fixtures at `templates/builtin/*.json` carry the doomed kinds — 133 references the plan doesn't enumerate

- **Claim under attack:** PLAN Paths line 5 enumerates `internal/app/*template_librar*`, `internal/domain/builtin_template_library.go`, etc. as the template excision surface — but says nothing about the JSON fixtures the Go files load.
- **Counterexample:** `templates/builtin/default-go.json:72 matches` and `templates/builtin/default-frontend.json:61 matches` reference the legacy kind vocabulary (`build-task`, `qa-check`, `plan-task`, `subtask`, `phase`, `commit-and-reingest`). These files are READ by `internal/domain/builtin_template_library.go` (14 hits) and `internal/app/template_library_builtin.go` (47 hits) at runtime to synthesize the built-in template library. If the plan excises `template_library*` Go code but leaves the JSON fixtures in the repo, (a) the JSONs become orphan files (cosmetic), and (b) `mage ci` still passes in a worktree where the JSONs are never parsed — UNLESS the `rg` + `sd` rename pass incidentally touches `subtask` / `phase` strings inside JSON, which would introduce unintended string mutations in fixture files not meant to change.
- **Impact:** Primarily hygiene (orphan fixture files after Go code deletion). But the `rg` + `sd` codebase-wide sweep for e.g. `WorkKind` → `Kind` strings is a narrow enough regex to skip JSON, while `work_items` → `action_items` in raw-text mode could accidentally mutate JSON comments / description fields. Plan's "narrow regex per identifier" (Notes line 45) mitigates this IF the planner explicitly excludes `templates/**/*.json` — but that exclusion is not documented.
- **Mitigation:** Plan unit for `rg` + `sd` rename must (a) document exclusion globs (`--glob '!templates/**'`, `--glob '!drops/**'`, `--glob '!docs/**'`, `--glob '!*.md'`), and (b) include a final unit that deletes the orphan `templates/builtin/*.json` files once `template_library*` Go is gone.

### F8. `task_embeddings → action_item_embeddings` table rename did not finish at the identifier layer — 83 residual `WorkItem`/`Work_items` identifier hits

- **Claim under attack:** PLAN Scope item 2: "one multi-pass `rg` + `sd` sweep over the whole repo: `work_items` → `action_items` (SQL + strings), `WorkItem` → `ActionItem` where still stale, `WorkKind` → `Kind`, `workitem` package/filename references". Phrased as a clean sweep.
- **Counterexample:** Grep for `Work_items|WorkItems|WorkItem\b` returns 83 hits across 15 files. Partial enumeration:
  - `internal/app/search_embeddings.go:2` — embedding search path
  - `internal/app/embedding_runtime.go:6`
  - `internal/app/embedding_runtime_test.go:6`
  - `internal/app/search_embeddings_test.go:1`
  - `internal/adapters/storage/sqlite/embedding_lifecycle_adapter.go:1`
  - `internal/adapters/storage/sqlite/embedding_lifecycle_adapter_test.go:10`
  - `internal/adapters/storage/sqlite/embedding_jobs_test.go:1`
  - `internal/adapters/storage/sqlite/repo_test.go:15`
  - `internal/adapters/storage/sqlite/repo.go:5`
  - `internal/app/service.go:5`
  - `internal/app/service_test.go:19`
  - `internal/tui/model.go:4` / `internal/tui/model_test.go:6`
- **Further:** `grep -c 'task_embeddings|task_id|action_item_embeddings|action_item_id'` returns 97 occurrences across 16 files, concentrated in `internal/adapters/storage/sqlite/embedding_lifecycle_adapter_test.go`, `repo.go`, and the embedding-lifecycle tests. The `rename-task-to-actionitem.sql` script claims (lines 6-8) that it renamed `task_embeddings → action_item_embeddings` and `task_id → action_item_id`, but Go-level string/identifier cleanup is incomplete — many callers still use `task_id` / `WorkItem*` identifiers.
- **Impact:** A "one `sd` pass" framing understates the real surface. The rename sweeps must include at least 4 distinct identifier chains: (a) `work_items → action_items` (table + strings), (b) `WorkItem → ActionItem` (struct / references), (c) `WorkKind → Kind` (type + constants), (d) residual `task_embeddings → action_item_embeddings` + `task_id → action_item_id` column identifiers in embedding code. Missing any chain leaves a broken reference at `mage ci` time.
- **Mitigation:** Planner must break item 2 into 4-5 sub-units with explicit per-regex scope (or a single unit with a documented multi-pass script running all 4-5 regexes before any commit). Each pass must be inspectable via `git diff` per Notes line 45.

### F9. MCP tool surface (`instructions_tool`, `extended_tools`) likely exposes `template_library*` commands to agents — the plan doesn't name the tool-list excision

- **Claim under attack:** PLAN Scope item 4: "delete every file/symbol under the `template_library*`... surface across `internal/domain`, `internal/app`, `internal/adapters/*`, `cmd/till`, tests, and any TUI references."
- **Counterexample:** `internal/adapters/server/mcpapi/extended_tools.go` has 25 matches for `template`, and `internal/adapters/server/mcpapi/instructions_tool.go` has 5. `instructions_explainer.go` has 8. These are likely the MCP tool registrations that expose `till.template_library_*` commands to agents. PLAN mentions `internal/adapters/server/mcpapi/*` in Paths but does not enumerate tool deletions or the `handler.go` wiring that registers them.
- **Impact:** If the builder deletes `internal/app/template_library*.go` but leaves the MCP tool handlers in `extended_tools.go` + their `handler.go` registration, the code won't compile (import unresolved). If they delete the handlers but miss the registration in `handler.go`, the registration compiles against a deleted handler — another build break. Also: the MCP tool DESCRIPTIONS (string literals like `"list template libraries"`) are user-visible and need deletion, not rename.
- **Mitigation:** Plan unit for template excision must list `extended_tools.go`, `instructions_tool.go`, `instructions_explainer.go`, `handler.go` tool-registration lines, and their test files (`extended_tools_test.go` — 33 matches, `instructions_tool_test.go` — 3 matches) explicitly.

### F10. Package-level blockers for `internal/domain` are underspecified — builder collision between rename and split units

- **Claim under attack:** PLAN Notes line 51 says "planner decomposes into units with `blocked_by` wiring (e.g. table rename unblocks SQL-script update; file rename unblocks any unit touching those files)" — generic hand-wave.
- **Counterexample:** `internal/domain` package holds at least 4 distinct transformations this drop touches:
  1. `task.go` → `action_item.go` file rename (item 3)
  2. `workitem.go` split: move `WorkKind` → `Kind` into `kind.go` (item 3) — AND `kind.go` already has `KindID` (Round 1 P6 unresolved hidden coupling)
  3. `WorkKind → Kind` identifier rename sweep (item 2) — touches files in 1 AND 2
  4. `template_library*.go`, `builtin_template_library.go`, `template_reapply.go` deletions (item 4) — inside the same package
  5. `project.go` `Kind` field deletion (item 5) — same package
  
  Any two of these units running concurrently in parallel builders break because Go packages compile as one unit. Per CLAUDE.md §"Blocker Semantics": "sibling build-tasks sharing a file in `paths` OR a package in `packages` MUST have an explicit `blocked_by` between them. Plan QA falsification attacks missing blockers." This is exactly the attack.
- **Impact:** If the planner splits item 2 (identifier rename) + item 3 (file rename) + item 4 (template excision) + item 5 (project.Kind drop) into 4 parallel units all touching `internal/domain`, two parallel builders will compile-clash. The cascade pre-Drop-4 is manual orchestration, but the planner still has to declare `blocked_by` because the WORKFLOW doesn't enforce file-level locking automatically.
- **Mitigation:** Planner MUST enforce strict `internal/domain` serialization: one unit at a time touching that package. Suggest a unit ordering like: (N.1) template excision in `internal/domain` → (N.2) `task.go → action_item.go` + `workitem.go` split into `kind.go` + `WorkKind → Kind` rename (one unit — these are inseparable in `internal/domain`) → (N.3) `project.Kind` field + `SetKind` method deletion. Each `blocked_by` the previous. Same serialization discipline for `internal/app` (multiple template files + `kind_capability.go` + `snapshot.go` + `service.go` touched), `internal/adapters/storage/sqlite` (repo.go + tests), and `internal/tui` (model.go + model_test.go both 4k+ LOC).

### F11. `drops-rewrite.sql` assertions need an update the plan doesn't spec

- **Claim under attack:** PLAN Scope item 6: "`UPDATE action_items SET kind='actionItem', scope='actionItem'` (all 115 live rows are currently `task/task`), and assert `SELECT COUNT(*) FROM kind_catalog = 2`."
- **Counterexample:** The current `scripts/drops-rewrite.sql:249-254` already has three matching assertions:
  - `kind_catalog_only_project_and_drop` expected 2 (but for `{project, drop}`)
  - `kind_catalog_has_drop` expected 1 (row id = 'drop')
  - `kind_catalog_has_project` expected 1 (row id = 'project')
  
  And `project_allowed_kinds_only_valid` checks `kind_id NOT IN ('project', 'drop')`. PLAN says the new target is `{project, actionItem}` — so every assertion needs its literal value flipped from `'drop'` to `'actionItem'`. PLAN doesn't call this out explicitly; item 6 says "assert `kind_catalog` row count = 2" but the LABELS of the assertions still reference `drop`. Label-only drift is cosmetic; the real risk is forgetting to update the `id = 'drop'` → `id = 'actionItem'` literal on line 251.
- **Impact:** A builder doing a quick `sd 'drop' 'actionItem' scripts/drops-rewrite.sql` is too broad (it hits `DROP TABLE`, `DROP COLUMN`, comment text like "Drop 1.75", and the file's own title). A careful hand-edit needs the planner to enumerate exactly which tokens change.
- **Mitigation:** Planner's unit for `drops-rewrite.sql` rewrite must list:
  - `kind_catalog_only_project_and_drop` (label rename) and `kind_id NOT IN ('project', 'drop')` (value update)
  - `kind_catalog_has_drop` → `kind_catalog_has_actionItem`
  - `WHERE id = 'drop'` → `WHERE id = 'actionItem'`
  - Removing PHASE 5 (introduce `drop` kind) entirely — replace with INSERT of `actionItem` kind if needed
  - Rewriting PHASE 6 role hydration (plan says DELETE this phase)
  - Rewriting PHASE 7 `UPDATE work_items SET kind='drop', scope='task'` → `UPDATE action_items SET kind='actionItem', scope='actionItem'`
  - Table name in the pre-flight snapshot at line 49: `FROM work_items` → `FROM action_items`

## Summary

**Failed:** 11 concrete counterexamples, each grounded in file:line evidence, demonstrate the plan as written would crash `mage ci` or silently drift semantics on execution.

**What must change before Phase 3 exit:**
1. **F1** — Enumerate BOTH seed surfaces (`repo.go` + `kind_capability.go:863-874 defaultKindDefinitionInputs` + its bootstrap call sites).
2. **F2** — Explicitly list `migrateTemplateLifecycle` + backfill helpers for deletion in the same unit as the `template_libraries` table excision.
3. **F3** — Enumerate all 26 `tasks` table references; include `repo_test.go:1006-1049` fixture + bridge test deletion.
4. **F4** — Document fresh-install vs upgrade semantics for the `ALTER TABLE action_items` idempotence block.
5. **F5** — Name which of `KindAppliesTo*` / `CapabilityScope*` / `AuthRequestPathKind` survive; confirm `KindAppliesToBranch/Phase/Subtask` + `CapabilityScope*` SURVIVE Drop 1.75 (per "Auth Path Branch Quirk" memory).
6. **F6** — Break item 7 into per-package test-update units with explicit file enumeration.
7. **F7** — Exclude `templates/**`, `drops/**`, `docs/**`, `*.md` from `rg` + `sd` sweeps; explicitly delete orphan `templates/builtin/*.json` in a final unit.
8. **F8** — Split item 2 (`work_items → action_items` rename) into 4-5 sub-passes per identifier chain.
9. **F9** — Enumerate MCP tool file deletions in `extended_tools.go`, `instructions_tool.go`, `instructions_explainer.go`, `handler.go` + their tests.
10. **F10** — Serialize `internal/domain` units (one at a time) via explicit `blocked_by`; same for `internal/app`, `internal/adapters/storage/sqlite`, `internal/tui`.
11. **F11** — Enumerate exact `drops-rewrite.sql` assertion edits (label + value + table-name).

**Unresolved round-1 items still outstanding** (raised in PLAN_QA_PROOF.md, unaddressed in PLAN.md body):
- P4 (seedDefaultKindCatalog line range `1231-1247` → `1231-1301`).
- P5 (`internal/domain/kind.go` "new" tag contradicts scope body).
- P6 (`Kind` vs `KindID` collision — two types for the same conceptual payload).
- P9 (`schema.go` doesn't exist; schema lives in `repo.go`).
- P10 (`template_librar*` file count 44 is off — but cosmetic).

The scope is the right SHAPE but needs a substantially more-enumerated Phase 1 unit breakdown before a builder can execute it safely.
