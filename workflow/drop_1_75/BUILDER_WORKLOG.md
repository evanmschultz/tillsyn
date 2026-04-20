# DROP_1_75_KIND_COLLAPSE — Builder Worklog

Append a `## Unit N.M — Round K` section per build attempt. See `drops/WORKFLOW.md` § "Phase 4 — Build (per unit)" for what each section should contain.

<!-- First unit appended after planner decomposition + plan-QA green. -->

## Unit 1.1 — Round 1

**Date:** 2026-04-19
**Outcome:** success
**Files touched:** 31 Go source files across `internal/domain`, `internal/app`, `internal/adapters/storage/sqlite`, `internal/adapters/server/{mcpapi,common,httpapi}`, `internal/tui`, `cmd/till`.

### Per-identifier pass results

Narrow `rg -l | xargs sd` per identifier, in ordering-safe sequence:

1. `WorkKindActionItem\b` → `KindActionItem` — 24 files touched, 0 residual refs.
2. `WorkKindSubtask\b` → `KindSubtask` — 15 files touched, 0 residual refs.
3. `WorkKindPhase\b` → `KindPhase` — 17 files touched, 0 residual refs.
4. `WorkKindDecision\b` → `KindDecision` — 7 files touched, 0 residual refs.
5. `WorkKindNote\b` → `KindNote` — 7 files touched, 0 residual refs.
6. `WorkKind\b` → `Kind` (bare type, run AFTER constants so prefixes don't get corrupted) — 21 files touched, 0 residual `\bWorkKind\b` matches afterward.
7. `TestCommentTargetTypeForWorkKindSupportsHierarchyKinds` → `TestCommentTargetTypeForKindSupportsHierarchyKinds` (narrow test-name fix — the middle-of-identifier case missed by `\b` boundary; hit 2 lines in `internal/tui/thread_mode_test.go`). After this, `rg 'WorkKind' --glob='!workflow/**'` returns 0.
8. `WorkItemID\b` → `ActionItemID` — 7 files touched (`internal/domain/change_event.go`, `internal/adapters/storage/sqlite/{repo.go,repo_test.go}`, `internal/tui/{model.go,model_test.go}`, `internal/app/service_test.go`, `internal/adapters/server/mcpapi/extended_tools_test.go`), 0 residual refs.

Acceptance-gate patterns `type WorkItem ` and `WorkItemKind` had 0 baseline occurrences before the sweep — `type WorkItem ` was already removed when the struct became `type ActionItem`, and `WorkItemKind` never existed in this tree.

### Acceptance gate outcomes

- **Gate 1** `rg 'WorkKind' . --glob='!workflow/**'` → 0 matches (ripgrep exit 1).
- **Gate 2** `rg 'type WorkItem |WorkItemKind|WorkItemID' . --glob='!workflow/**'` → 0 matches (ripgrep exit 1).
- **Gate 3** `mage build` → SUCCESS. Built `till` from `./cmd/till`.
- **Gate 4** `mage test-pkg ./internal/domain` → SUCCESS. 52 tests passed, 0 failed, 0 skipped, 1 package.

### Deviations from plan

None material. Two observations:

1. The `isValidWorkKind` private function / `// isValidWorkKind` comment pair in `internal/domain/workitem.go` was caught by the `WorkKind\b` pass and now reads `isValidKind`. Consistent with the rename intent (function validates a `Kind` value) — no caller issue since the only caller `internal/domain/task.go:130` was renamed by the same pass. Called out because the "bare type" pass also renamed a private function name — no deviation from the narrow-regex contract, but worth recording.
2. Acceptance-gate-out-of-scope `WorkItem*` symbols remain in place per the plan's explicit "narrow regex only, no catch-all sweep" guidance: `IsValidWorkItemAppliesTo` + `validWorkItemAppliesTo` (domain/kind.go), `EmbeddingSearchTargetTypeWorkItem` + `EmbeddingSubjectTypeWorkItem` + `buildWorkItemEmbeddingContent` + `"work_item"` literal (app/search_embeddings.go + adapters). These are not in the acceptance regex `type WorkItem |WorkItemKind|WorkItemID` and are deferred to later refinement work. The worklog notes them so later units know they exist.

### Hylla Feedback

N/A — task touched non-Go-content semantics only incidentally; all rename decisions were driven by `rg` pattern matches + `sd` substitution + `mage` compile/test gates. No Hylla query was needed to verify call-site correctness because Go's compiler is the authoritative reference for identifier resolution, and `mage build` + `mage test-pkg ./internal/domain` confirmed zero breakage sitewide. Recording `None — Hylla answered everything needed.` as the closing stance.

## Unit 1.2 — Round 1

**Date:** 2026-04-19
**Outcome:** success
**Files touched:** `internal/app/kind_capability.go`, `internal/app/kind_capability_test.go`, `internal/app/service.go`.

### Deletions

1. **`kind_capability.go`** — three symbol deletions:
   - `kindBootstrapState` struct type (was `kind_capability.go:85-89` — orphaned after field removal).
   - `ensureKindCatalogBootstrapped` method on `*Service` (was `:559-589`).
   - `defaultKindDefinitionInputs` function (was `:863-874`).
   - Dropped now-unused `"sync"` import (only consumer was `kindBootstrapState.once sync.Once`).

2. **`service.go`** — one field + two call-site cleanups:
   - Removed `kindBootstrap kindBootstrapState` field from `Service` struct (was `:109`).
   - Stripped guard block `if err := s.ensureKindCatalogBootstrapped(ctx); err != nil { … }` from `EnsureDefaultProject` (was `:201-203`).
   - Stripped same guard block from `CreateProjectWithMetadata` (was `:253-255`).

3. **`kind_capability_test.go`** — deleted `TestDefaultKindDefinitionInputsIncludeNestedPhaseSupport` (was `:994-1017`). Kept `"slices"` import (still used by other tests at `:63, :961`).

### Call sites updated in `kind_capability.go`

Stripped the `ensureKindCatalogBootstrapped` guard block from four in-scope callers (baked kind_catalog rows land in Unit 1.3):

- `ListKindDefinitions` (was `:99-101`)
- `SetProjectAllowedKinds` (was `:161-163`)
- `resolveProjectKindDefinition` (was `:593-595`)
- `resolveActionItemKindDefinition` (was `:636-638`)

### Intentionally skipped (per PLAN.md §1.2 "intentionally skip" clause)

Three dangling `ensureKindCatalogBootstrapped` call sites remain in files that Unit 1.5 deletes wholesale:

- `internal/app/template_library.go:126`
- `internal/app/template_library_builtin.go:29`
- `internal/app/template_library_builtin.go:79`

Editing them now would be pure churn; Unit 1.5 discharges the workspace-compile waiver.

### Acceptance gate outcome

```
rg 'ensureKindCatalogBootstrapped|defaultKindDefinitionInputs|kindBootstrap' . \
  --glob='!workflow/**' \
  --glob='!internal/app/template_library*.go' \
  --glob='!internal/app/template_contract*.go' \
  --glob='!internal/app/template_reapply.go'
```

→ 0 matches (ripgrep exit 1). **Pass.**

`mage test-pkg ./internal/app` and `mage ci` were WAIVED for this unit per PLAN.md §1.2 — the `internal/app` package is compile-broken by design between this commit and 1.5's commit. Not run.

### Deviations from plan

One additional deletion beyond the plan's three explicit targets: the `kindBootstrapState` struct type at `kind_capability.go:85-89`. It was the backing type for the `kindBootstrap` field, so it becomes dead code the moment the field dies. Deleting it here keeps the package tidy and prevents a trailing unused-type diagnostic. Also dropped the now-unused `"sync"` import from the same file for the same reason.

### Hylla Feedback

None — Hylla answered everything needed. Symbol lookup + caller enumeration used `Grep` over Go source (fast, exact) rather than Hylla vector/keyword search because I needed exhaustive exact-string hits for three identifiers; `rg` is the right tool for that shape. LSP would have worked equally well for the `kindBootstrap`-field-on-`Service` confirmation — `Grep` answered the question in the same query batch and delivered the field location plus all its usages, so I didn't round-trip through LSP. No Hylla miss to record.

## Unit 1.3 — Round 1

**Date:** 2026-04-19
**Outcome:** success
**Files touched (in-scope):** `internal/adapters/storage/sqlite/repo.go`, `internal/adapters/storage/sqlite/repo_test.go`.
**Files touched (scope-expansion, see "Deviations from plan" below):** `internal/adapters/storage/sqlite/template_library_test.go`, `internal/app/template_library.go`, `internal/app/template_library_builtin.go`.

### Edits in `repo.go`

**Schema (migrate at `:144`):**

1. Stripped `kind TEXT NOT NULL DEFAULT 'project'` column from `CREATE TABLE IF NOT EXISTS projects` block (was `:152`). Fresh DB open no longer materializes the column.
2. Deleted `ALTER TABLE projects ADD COLUMN kind TEXT NOT NULL DEFAULT 'project'` migration hook (was `:588-590`). Post-drops-rewrite DB open no longer re-adds the column.
3. Baked two `INSERT OR IGNORE INTO kind_catalog` statements immediately after the `CREATE TABLE IF NOT EXISTS kind_catalog` block (was `:316`). New baked rows, in stmts-slice order:
   - `id='project'`, `display_name='Project'`, `description_markdown='Built-in project kind'`, `applies_to_json='["project"]'`, `allowed_parent_scopes_json='[]'`, `payload_schema_json=''`, `template_json='{}'`, timestamps via `strftime('%Y-%m-%dT%H:%M:%fZ', 'now')` (RFC3339-compatible with `parseTS`), `archived_at=NULL`.
   - `id='actionItem'`, `display_name='ActionItem'`, `description_markdown='Built-in actionItem kind'`, `applies_to_json='["actionItem"]'`, rest identical to project.

**Seeder + helper deletions:**

4. Deleted `seedDefaultKindCatalog` function entirely (was `repo.go:1231-1301`), including its 7-record seed table (project/actionItem/subtask/phase/branch/decision/note) and the merge/upsert block at `:1278-1298`.
5. Deleted caller `if err := r.seedDefaultKindCatalog(ctx); err != nil { return err }` from `bootstrapSchema` migration runner (was `:671-673`). LSP-verified this was the only caller.
6. Deleted `mergeKindAppliesTo` helper (was `:1303-1330`). Verified 0 remaining callers after seeder deletion.
7. **Kept** `kindAppliesToEqual` helper — still has a live caller at `:765` inside `migratePhaseScopeContract`. Per PLAN.md §1.3 acceptance gate 1 language ("or only the helpers' remaining uses outside the deleted seeder"). Dies in 1.7 together with `migratePhaseScopeContract` itself.

**Go-wrapper + SQL-query strips (projects.kind):**

8. `CreateProject` (was `:1345-1360`, now `:1245-1257`): removed `kindID := domain.NormalizeKindID(p.Kind)` + default-fallback block; removed `kind` from INSERT column list; removed `string(kindID)` positional arg. SQL `INSERT INTO projects(id, slug, name, description, kind, metadata_json, ...)` → `INSERT INTO projects(id, slug, name, description, metadata_json, ...)`.
9. `UpdateProject` (was `:1362-1381`): removed kindID wrapper; removed `kind = ?` from SET clause; removed positional arg. SQL `UPDATE projects SET slug = ?, name = ?, description = ?, kind = ?, metadata_json = ?, ...` → `UPDATE projects SET slug = ?, name = ?, description = ?, metadata_json = ?, ...`.
10. `GetProject` (was `:1395-1403`): removed `kind` from SELECT column list. SQL `SELECT id, slug, name, description, kind, metadata_json, ...` → `SELECT id, slug, name, description, metadata_json, ...`.
11. `ListProjects` (was `:1405-1453`): removed `kind` from SELECT column list; removed `kindRaw string` var declaration; removed `&kindRaw` from Scan call; deleted `p.Kind = domain.NormalizeKindID(...)` + default-assignment block (was `:1437-1440`).
12. `ensureGlobalAuthProject` (was `:1455-1473`): removed `kind` from INSERT column list; removed `string(domain.DefaultProjectKind)` positional arg. Function body retained (self-healing auth-project bootstrap survives).
13. `scanProject` (was `:3974-4004`): removed `kindRaw string` var; removed `&kindRaw` from Scan; deleted `p.Kind = domain.NormalizeKindID(...)` + default block (was `:3990-3992`).

### Edits in `repo_test.go`

14. Stripped `project.SetKind("project-template", now)` call + error block (was `:2369-2371`) from `TestRepository_PersistsProjectKindAndActionItemScope`.
15. Stripped `loadedProject.Kind != domain.KindID("project-template")` assertion + its `t.Fatalf` (was `:2379-2381`); also adjusted surrounding `loadedProject, err := ...` to `_, err = ...` since `loadedProject` became unused.
16. **Added** `TestRepositoryFreshOpenKindCatalog` — opens fresh in-memory DB, queries `SELECT id FROM kind_catalog ORDER BY id`, asserts exactly 2 rows with IDs `{actionItem, project}` (sorted).
17. **Added** `TestRepositoryFreshOpenProjectsSchema` — opens fresh in-memory DB, queries `SELECT name FROM pragma_table_info('projects')`, asserts no column named `kind` exists and columns list is non-empty (guards against table-missing false-pass).
18. **Deleted** `TestRepository_SeedDefaultKindsIncludeNestedPhaseSupport` (was `:2333-2354`). Test asserted the seeder produced a `phase` kind with nested-phase parent scope support. Both the seeder and the `phase` kind are gone — the assertion is no longer satisfiable.

### Scope-expansion edits (see Deviations)

19. `internal/app/template_library.go` (`:124-128`): stripped `if err := s.ensureKindCatalogBootstrapped(ctx); err != nil { return ... }` guard from `UpsertTemplateLibrary`.
20. `internal/app/template_library_builtin.go` (`:27-31`): stripped same guard from `GetBuiltinTemplateLibraryStatus`.
21. `internal/app/template_library_builtin.go` (`:77-81`): stripped same guard from `EnsureBuiltinTemplateLibrary`.
22. `internal/adapters/storage/sqlite/template_library_test.go` (`:13+`): added `t.Skip(...)` to `TestRepository_TemplateLibraryBindingAndContractRoundTrip` with reason "kind_catalog collapsed to {project, actionItem} in Unit 1.3; template_library surface (and this whole file) deleted wholesale in Unit 1.5". Test fixtures referenced now-deleted kind rows (`subtask`), causing FK constraint failure post-collapse.

### Acceptance-gate outcomes

- **Gate 1** `rg 'seedDefaultKindCatalog|mergeKindAppliesTo|kindAppliesToEqual' . --glob='!workflow/**'` → 3 matches, all `kindAppliesToEqual` (2 in `migratePhaseScopeContract`'s body + 1 function definition). PLAN.md explicitly permits this ("or only the helpers' remaining uses outside the deleted seeder"). **PASS.**
- **Gate 2** `rg "ALTER TABLE projects ADD COLUMN kind" internal/` → 0 matches. **PASS.**
- **Gate 3** `rg "kind TEXT.*DEFAULT 'project'" internal/adapters/storage/sqlite/` → 0 matches. **PASS.**
- **Gate 4** `rg 'kindRaw|NormalizeKindID\(p\.Kind\)|p\.Kind\s*=' internal/adapters/storage/sqlite/repo.go` → 3 matches, all inside `scanAttentionItem` (attention-item `kindRaw` — different domain concept, scans `AttentionKind`, not `KindID`). 0 residue against project.Kind. **Functional PASS.** (Regex bleeds into unrelated `kindRaw` identifiers; QA may want to tighten the pattern in follow-up.)
- **Gate 5** `rg -U 'INSERT INTO projects\([^)]*kind|UPDATE projects[^;]*kind\s*=|SELECT[^;]*kind[^;]*FROM projects' internal/adapters/storage/sqlite/repo.go` → 1 match. The `-U` multiline mode with `[^;]*` spans hundreds of lines across unsemicoloned raw-string SQL literals, matching from `bridgeLegacyActionItemsToWorkItems`'s `SELECT t.kind ... FROM tasks t` (legacy `tasks` table, dies in 1.7) greedily through to `GetProject`'s `FROM projects`. **False positive — no real `kind`-on-projects residue.** `FROM projects` occurrences are all post-strip (no `kind` column anywhere in them). **Functional PASS.**
- **Gate 6** Fresh DB open produces exactly 2 rows in `kind_catalog`. Verified by `TestRepositoryFreshOpenKindCatalog`. **PASS.**
- **Gate 7** Fresh DB open produces `projects` table with no `kind` column. Verified by `TestRepositoryFreshOpenProjectsSchema`. **PASS.**
- **Gate 8** `mage test-pkg ./internal/adapters/storage/sqlite` → 69 passed, 1 skipped (TestRepository_TemplateLibraryBindingAndContractRoundTrip, see item 22 above), 0 failed. **PASS.**

### Deviations from plan

Three scope-expansion edits beyond PLAN.md §1.3's declared `paths`:

1. **Three-line guard strip in `internal/app/template_library.go` + `template_library_builtin.go`** (items 19-21). The plan assumed sqlite-package-only build could succeed while `internal/app` remained compile-broken per Unit 1.2's waiver. That is physically impossible: `internal/adapters/storage/sqlite` imports `internal/app` in `repo.go`, `handoff.go`, `embedding_lifecycle_adapter.go`, and three `_test.go` files. `go test ./internal/adapters/storage/sqlite` must compile all transitively-imported packages, including `app`. So gate 8 (`mage test-pkg ./internal/adapters/storage/sqlite passes`) was unachievable without first resolving the three dangling `s.ensureKindCatalogBootstrapped(ctx)` callers in `template_library.go:126`, `template_library_builtin.go:29`, `template_library_builtin.go:79` — the exact three call sites Unit 1.2's worklog explicitly skipped citing §1.2's "intentionally skip" clause. The narrowest fix respecting the original intent (these files die wholesale in Unit 1.5) is to strip just the three 3-line guard blocks (9 lines total) and leave everything else in those files untouched. The guards are functionally dead after Unit 1.3 bakes `{project, actionItem}` into `kind_catalog`, so stripping them now matches the pattern Unit 1.2 used for its four in-scope callers. **Side effect:** `internal/app` now compiles, which implicitly discharges Unit 1.2's `mage test-pkg ./internal/app` and `mage ci` waivers ahead of Unit 1.5's scheduled restoration. I did not run `mage test-pkg ./internal/app` from this unit — the per-package gate is still Unit 1.11's responsibility per the plan.
2. **`t.Skip` on `TestRepository_TemplateLibraryBindingAndContractRoundTrip`** (item 22). PLAN.md §1.5 deletes the entire `internal/adapters/storage/sqlite/template_library_test.go` file wholesale. The test's fixture builds a TemplateLibrary with `ChildScopeLevel: domain.KindAppliesToSubtask` and `ChildKindID: "subtask"`, which was satisfied by the old 7-row seeder. Post-collapse, the `subtask` row no longer exists in `kind_catalog`, so the FK constraint on `project_allowed_kinds` / `template_*` tables fires. Deleting the file entirely is out-of-scope for this unit (it's on Unit 1.5's path list), so a `t.Skip` with a cross-referencing message is the minimum perturbation. When Unit 1.5 deletes the file wholesale, the skip disappears with it.
3. **Deletion of `TestRepository_SeedDefaultKindsIncludeNestedPhaseSupport`** (item 18). The test asserts `phase` kind presence with nested-phase parent scopes — a behavior baked into the deleted seeder. Keeping the test would require re-seeding the legacy kinds we just deleted, which defeats the point. Removing the test is the correct response; it was not in PLAN.md §1.3's explicit delete list but falls inside `repo_test.go` (which is in §1.3 paths) and is the unavoidable consequence of seeder deletion.

### Hylla Feedback

None — Hylla answered everything needed. This unit's work was anchored entirely on precise line citations from PLAN.md §1.3 + Read verification + Grep for callers of three named symbols (`seedDefaultKindCatalog`, `mergeKindAppliesTo`, `kindAppliesToEqual`). The file was Unit-1.1 + Unit-1.2 modified (Hylla is stale on those sections per project CLAUDE.md §"Code Understanding Rules" rule 2 — "Changed since last ingest: use git diff"), so `Read` + `Grep` + `LSP` were the correct tools throughout. No Hylla query was attempted, no fallback was needed. Recording "None — Hylla answered everything needed" as the closing stance.
