# DROP_1_75_KIND_COLLAPSE — Builder QA Proof

Append a `## Unit N.M — Round K` section per QA proof pass.

## Unit 1.1 — Round 1

**Verdict:** pass

## Summary

The builder's claim — 31 Go source files renamed across 8 narrow `rg`+`sd` passes for `WorkKind → Kind` and `WorkItemID → ActionItemID`, with PLAN.md §1.1 flipped to `done` and a factual append in BUILDER_WORKLOG.md — is fully supported by the evidence. All four acceptance gates re-ran clean from scratch. All 8 pass claims match post-state identifier counts and the `git diff` file list. All 3 disclosed deviations match diff reality and are consistent with PLAN.md §Scope guidance. File-count coherence (33 = 31 Go + 2 workflow MDs) holds. No Section 0 leakage into the worklog.

## Confirmed Claims

- **Gate 1 (fresh re-run):** `rg 'WorkKind' . --glob='!workflow/**' --count-matches` → 0 matches, rg exit 1.
- **Gate 2 (fresh re-run):** `rg 'type WorkItem |WorkItemKind|WorkItemID' . --glob='!workflow/**' --count-matches` → 0 matches, rg exit 1.
- **Gate 3 (fresh re-run):** `mage build` → `[SUCCESS] Built till from ./cmd/till`, exit 0.
- **Gate 4 (fresh re-run):** `mage test-pkg ./internal/domain` → 52 tests, 52 passed, 0 failed, 0 skipped, 1 package. Exit 0.
- **Pass 1 (`WorkKindActionItem → KindActionItem`):** `rg 'KindActionItem\b' . --glob='!workflow/**' -l | wc -l` = 24. Matches builder-claimed file count.
- **Pass 2 (`WorkKindSubtask → KindSubtask`):** `rg 'KindSubtask\b' . --glob='!workflow/**' -l | wc -l` = 15. Matches claim.
- **Pass 3 (`WorkKindPhase → KindPhase`):** `rg 'KindPhase\b' . --glob='!workflow/**' -l | wc -l` = 17. Matches claim.
- **Pass 4 (`WorkKindDecision → KindDecision`):** `rg 'KindDecision\b' . --glob='!workflow/**' -l | wc -l` = 7. Matches claim.
- **Pass 5 (`WorkKindNote → KindNote`):** `rg 'KindNote\b' . --glob='!workflow/**' -l | wc -l` = 7. Matches claim.
- **Pass 6 (`WorkKind\b → Kind`):** `rg 'WorkKind' . --glob='!workflow/**'` → 0 after passes complete. Bare type is correctly in use (e.g., `KindActionItem Kind = "actionItem"` const at `internal/domain/workitem.go:39`).
- **Pass 7 (test-name rename):** `rg 'TestCommentTargetTypeForWorkKindSupportsHierarchyKinds' . --glob='!workflow/**' --count-matches` → 0 matches (rg exit 1). `rg 'TestCommentTargetTypeForKindSupportsHierarchyKinds' . --glob='!workflow/**' --count-matches` → 2 hits in `internal/tui/thread_mode_test.go`. Matches builder claim exactly.
- **Pass 8 (`WorkItemID → ActionItemID`):** `rg 'WorkItemID\b' . --glob='!workflow/**' --count-matches` → 0 (rg exit 1). All 7 builder-named files (`internal/domain/change_event.go`, `internal/adapters/storage/sqlite/{repo.go,repo_test.go}`, `internal/tui/{model.go,model_test.go}`, `internal/app/service_test.go`, `internal/adapters/server/mcpapi/extended_tools_test.go`) appear in `git diff --name-only`. `ActionItemID\b` is present in 24 files.
- **File-count coherence:** `git diff --name-only | wc -l` = 33. Breakdown: 31 `.go` files + `workflow/drop_1_75/PLAN.md` + `workflow/drop_1_75/BUILDER_WORKLOG.md`. Matches builder claim.
- **PLAN.md state flip:** `git diff workflow/drop_1_75/PLAN.md` shows exactly one substantive change: §1.1 `**State:** todo` → `**State:** done`. No other edits.
- **BUILDER_WORKLOG.md append:** factual per-pass listing, acceptance-gate outcomes, two observations. No Section 0 reasoning leaked — `rg 'Section 0|Proposal|Convergence|QA Proof|QA Falsification' workflow/drop_1_75/BUILDER_WORKLOG.md` → 0 hits (rg exit 1).
- **Deviation 1 (`isValidWorkKind → isValidKind`):** `rg 'isValidWorkKind' . --glob='!workflow/**'` → 0 matches. `rg 'isValidKind' . --glob='!workflow/**' -n` shows definition at `internal/domain/workitem.go:195-196` and single call at `internal/domain/task.go:130`. The `WorkKind\b` pass correctly caught the private helper; the only caller was renamed by the same pass. No orphaned reference.
- **Deviation 2 (baseline zero on `type WorkItem ` / `WorkItemKind`):** `git show HEAD:internal/domain/workitem.go | rg 'WorkItemKind|type WorkItem '` → 0 matches at HEAD. Builder's claim that these patterns had zero baseline pre-unit is correct — the struct was already `type ActionItem` pre-Drop-1.75 per project CLAUDE.md "Pre-Drop-1.75 Creation Rule."
- **Deviation 3 (preserved `WorkItem*` out-of-scope):** surviving `WorkItem*` symbols (`IsValidWorkItemAppliesTo`, `validWorkItemAppliesTo`, `EmbeddingSearchTargetTypeWorkItem`, `EmbeddingSubjectTypeWorkItem`, `buildWorkItemEmbeddingContent`) remain across 21 files. These are explicitly out-of-scope per PLAN.md §Scope bullet 48 ("orphan-via-collapse refactor"). Acceptance gate 2 (narrow regex `type WorkItem |WorkItemKind|WorkItemID`) does not include them by design.
- **Case-insensitive sanity:** `rg -i 'WorkKind' . --glob='!workflow/**'` → 0 matches. No residual `WorkKind` in comments, string literals, or documentation.

## Hylla Feedback

N/A — this QA pass verified a pure Go-identifier rename via `rg` / `git diff` / `mage` gates. Hylla was not consulted because the evidence demanded by the acceptance gates is lexical (ripgrep) and semantic (Go compiler via `mage build` + `mage test-pkg`). No Hylla miss to record.

## Unit 1.2 — Round 1

**Verdict:** PASS

## Summary

Builder's claim — delete the app-layer kind-catalog seeder (`ensureKindCatalogBootstrapped` + `defaultKindDefinitionInputs` + `kindBootstrap` field + orphaned `kindBootstrapState` type), drop the now-unused `"sync"` import from `kind_capability.go`, and strip the 6 in-scope guard-block callers (4 in `kind_capability.go` + 2 in `service.go`) — is fully supported by the evidence. The acceptance `rg` returns 0. The three dangling `template_library*.go` references are the documented Unit 1.5 waiver, not defects. `git diff HEAD` matches the worklog's "Files touched" list (3 Go files + 2 MD files) exactly.

## Confirmed Claims

- **Acceptance rg (fresh re-run):** `rg 'ensureKindCatalogBootstrapped|defaultKindDefinitionInputs|kindBootstrap' . --glob='!workflow/**' --glob='!internal/app/template_library*.go' --glob='!internal/app/template_contract*.go' --glob='!internal/app/template_reapply.go'` → 0 matches (rg exit 1). **Pass.**
- **`ensureKindCatalogBootstrapped` definition removed:** pre-diff hunk shows the full function body at `kind_capability.go:559-589` deleted, matching the worklog's `:559-589` range claim.
- **`defaultKindDefinitionInputs` removed:** pre-diff hunk shows the function body (returns 7 built-in kind inputs) deleted, matching the worklog's `:863-874` range claim (line numbers approximate — span deleted is 12 lines, close to the claimed 12-line span).
- **`kindBootstrap` field removed from `Service`:** `git diff internal/app/service.go` shows `-	kindBootstrap      kindBootstrapState` struck from the struct. Builder claim of pre-removal at `:109` matches `git show HEAD:internal/app/service.go | rg kindBootstrap` → `109:	kindBootstrap      kindBootstrapState`.
- **`kindBootstrapState` struct type removed (orphaned):** `git diff internal/app/kind_capability.go` shows the 6-line block (comment + struct with `once sync.Once`, `err error`) deleted. `rg 'kindBootstrapState' . --glob='!workflow/**' --glob='!.git/**'` → 0 matches.
- **`"sync"` import dropped from `kind_capability.go`:** diff shows `-	"sync"` removed from the import block. `rg '\bsync\.' internal/app/kind_capability.go` → 0 matches (no remaining consumers in the file). `"sync"` import intact in `service.go` because `schemaCacheMu sync.RWMutex` still uses it (verified via `rg '\bsync\.' internal/app/service.go` → `108:	schemaCacheMu      sync.RWMutex`).
- **In-scope callers stripped — `kind_capability.go` (4 sites):** `git show HEAD:internal/app/kind_capability.go | rg 'ensureKindCatalogBootstrapped' -n` pre-unit returned definition at `:559-560` + callers at `:99, :161, :593, :636`. Diff hunks show guard blocks stripped from `ListKindDefinitions` (pre-`:99-101`), `SetProjectAllowedKinds` (pre-`:161-163`), `resolveProjectKindDefinition` (pre-`:593-595`), `resolveActionItemKindDefinition` (pre-`:636-638`). All 4 guard blocks removed.
- **In-scope callers stripped — `service.go` (2 sites):** pre-unit refs at `:201, :253`. Diff hunks show guard blocks stripped from `EnsureDefaultProject` (pre-`:201-203`) and `CreateProjectWithMetadata` (pre-`:253-255`). Both callers cleaned.
- **Test deletion:** `TestDefaultKindDefinitionInputsIncludeNestedPhaseSupport` removed from `kind_capability_test.go` (diff shows 24 lines deleted at the end of the file). `rg 'TestDefaultKindDefinitionInputsIncludeNestedPhaseSupport' . --glob='!workflow/**'` → 0 matches.
- **`"slices"` import retained:** `rg 'slices\.' internal/app/kind_capability_test.go` → `:63`, `:961` — both still used. Matches worklog.
- **`service_test.go` untouched (correctly):** `git show HEAD:internal/app/service_test.go | rg 'defaultKindDefinitionInputs|kindBootstrap|ensureKindCatalogBootstrapped'` → 0 matches pre-unit. Plan listed it defensively; builder correctly identified no edits were needed. Current state also 0 matches.
- **Waiver-documented dangling refs present and exactly 3:** `rg 'ensureKindCatalogBootstrapped' internal/app/template_library.go internal/app/template_library_builtin.go -n` →
  - `internal/app/template_library.go:126`
  - `internal/app/template_library_builtin.go:29`
  - `internal/app/template_library_builtin.go:79`

  All three match the plan's "intentionally skip" clause verbatim. **Not a finding** — these die wholesale in Unit 1.5 (plan §1.5 Paths list includes both files for deletion). Confirmed per PLAN.md §1.2 waiver and task description's "Intentionally skipped (per plan, NOT defects)" bullet.
- **`mage test-pkg ./internal/app` + `mage ci` waiver honored:** worklog explicitly notes both were waived and not run. Matches plan §1.2 "mage test-pkg ./internal/app and mage ci are waived for this unit only." No violation.
- **`git diff HEAD --stat` file coherence:** 3 Go source files (`internal/app/kind_capability.go`, `internal/app/kind_capability_test.go`, `internal/app/service.go`) + 2 MD files (`workflow/drop_1_75/BUILDER_WORKLOG.md` append, `workflow/drop_1_75/PLAN.md` state flip). Matches worklog's "Files touched" list exactly.
- **PLAN.md state flip:** `git diff workflow/drop_1_75/PLAN.md` shows exactly one change: §1.2 `**State:** todo` → `**State:** done`. No drift.
- **Deviation 1 (`kindBootstrapState` orphan removal):** worklog explicitly disclosed; plan's §1.2 acceptance rg includes `kindBootstrap` as a target pattern, so the struct type (which matches `kindBootstrap*`) would have been caught by the same gate anyway. Deleting it is strictly cleaner than leaving it and is covered by the gate pattern.
- **Deviation 2 (`"sync"` import drop):** worklog explicitly disclosed; safe because the only `sync.` consumer in `kind_capability.go` was `kindBootstrapState.once sync.Once`. No remaining consumers (`rg '\bsync\.' internal/app/kind_capability.go` → 0).

## Missing Evidence

- None. All 6 plan-specified acceptance items (including the 3 waived-compile-failure items) have direct diff/rg evidence.

## P-Findings (Proof Gaps)

- None.

## Hylla Feedback

None — Hylla answered everything needed. This QA pass verified symbol deletions via `git diff HEAD` + `rg` over committed Go source (Hylla is stale for files touched after the last ingest, so `git diff` is the correct evidence source here per project CLAUDE.md rule #2). No Hylla query was attempted because the evidence demanded is lexical (has-the-identifier-disappeared-from-these-specific-files) and the diff is the authoritative record. No miss to record.

## Unit 1.3 — Round 1

**Verdict:** PASS

## Summary

Builder's claim — bake `project` + `actionItem` rows into `kind_catalog`, delete `seedDefaultKindCatalog` + `mergeKindAppliesTo`, strip `projects.kind` at both DDL sites (`:152` primary + `:588` ALTER), strip `kind` from all 6 SQL functions and their Go wrappers, add two new invariant tests, and scope-expand 9 lines in `internal/app/template_library*.go` to make gate 8 achievable — is fully supported by the evidence. All 8 acceptance gates re-ran clean. The two builder-flagged false positives (gate 4 `scanAttentionItem`-scope, gate 5 multi-line regex bleed through unsemicoloned raw-string SQL literals) are correctly classified. Scope-expansion is real (sqlite-package imports `internal/app` in 8 files), minimum-necessary (9 lines = narrowest reachable; stub-reinstatement would re-introduce deleted code), and intent-preserving (mirrors Unit 1.2's in-scope-caller treatment for files that die wholesale in Unit 1.5).

## Gate-by-gate evidence

- **Gate 1** `rg 'seedDefaultKindCatalog|mergeKindAppliesTo|kindAppliesToEqual' . --glob='!workflow/**'` → 3 matches, all `kindAppliesToEqual`: 1 function definition at `internal/adapters/storage/sqlite/repo.go:1237` and 2 call-site invocations on `repo.go:770` (same line, inside `migratePhaseScopeContract`). PLAN.md §1.3 gate 1 language explicitly permits these ("or only the helpers' remaining uses outside the deleted seeder"). `migratePhaseScopeContract` is scheduled for deletion in Unit 1.7. **PASS.**
- **Gate 2** `rg "ALTER TABLE projects ADD COLUMN kind" . --glob='!workflow/**'` → 0 matches. Workflow MD mentions are specs/worklogs, not source. **PASS.**
- **Gate 3** `rg "kind TEXT.*DEFAULT 'project'" internal/adapters/storage/sqlite/` → 0 matches. **PASS.**
- **Gate 4** `rg 'kindRaw|NormalizeKindID\(p\.Kind\)|p\.Kind\s*=' internal/adapters/storage/sqlite/repo.go` → 3 matches, all inside `scanAttentionItem` at `:4290` (var decl), `:4306` (Scan positional), `:4329` (`item.Kind = domain.NormalizeAttentionKind(domain.AttentionKind(kindRaw))`). This is the `AttentionKind` domain concept — `kindRaw` scans an attention-item kind column, not a project kind. Zero residue against `project.Kind`. Builder's false-positive classification verified. **PASS.** (Regex could be tightened in follow-up, e.g. `p\.Kind\b\s*=` with negative lookbehind, but Go re2 doesn't support lookbehind; the PLAN regex is a best-effort lexical proxy.)
- **Gate 5** `rg -U 'INSERT INTO projects\([^)]*kind|UPDATE projects[^;]*kind\s*=|SELECT[^;]*kind[^;]*FROM projects' internal/adapters/storage/sqlite/repo.go` → 1 match spanning from `bridgeLegacyActionItemsToWorkItems`'s raw-string `SELECT t.kind ... FROM tasks t` body (starts `:1197`) greedily through `kindAppliesToEqual` func body, past `CreateProject` / `UpdateProject` / `DeleteProject`, and ending in `GetProject`'s `FROM projects` raw-string at `:1295`. Multi-line mode + `[^;]*` with no SQL-literal-boundary discipline creates the bleed. Tighter replacement `rg -U 'SELECT[^;]*\bkind\b[^;]*FROM projects\b' repo.go` → 0 matches. Manual inspection of all 6 project-SQL sites (`INSERT INTO projects` at `:1256` + `:1349`, `UPDATE projects` at `:1269`, `SELECT ... FROM projects` at `:1294` + `:1304`) confirms zero `kind` column references. Builder's false-positive classification verified. **Functional PASS.**
- **Gate 6** `TestRepositoryFreshOpenKindCatalog` added in `repo_test.go` (diff lines +2566 through +2605). Opens fresh in-memory DB, queries `SELECT id FROM kind_catalog ORDER BY id`, asserts exactly 2 rows with IDs `["actionItem", "project"]`. Test passes under `mage test-pkg ./internal/adapters/storage/sqlite`. **PASS.**
- **Gate 7** `TestRepositoryFreshOpenProjectsSchema` added in `repo_test.go` (diff lines +2607 through +2644). Opens fresh in-memory DB, queries `SELECT name FROM pragma_table_info('projects')`, rejects any `kind` column, guards against 0-column false-pass via `len(columns) == 0` check. Test passes. **PASS.**
- **Gate 8** `mage test-pkg ./internal/adapters/storage/sqlite` → 70 total tests, 69 passed, 0 failed, 1 skipped (`TestRepository_TemplateLibraryBindingAndContractRoundTrip` — documented 1.5-scheduled deletion, skip reason cross-references Unit 1.3/1.5). **PASS.**

## Scope-expansion minimum-necessity check

Builder stripped 3 × 3-line guard blocks (9 lines total) in files PLAN §1.2 explicitly marked "intentionally skip":

- `internal/app/template_library.go:124-128` → guard removed from `UpsertTemplateLibrary`.
- `internal/app/template_library_builtin.go:27-31` → guard removed from `GetBuiltinTemplateLibraryStatus`.
- `internal/app/template_library_builtin.go:74-81` → guard removed from `EnsureBuiltinTemplateLibrary`.

- **Is the import dependency real?** YES. `rg '"github.com/evanmschultz/tillsyn/internal/app"' internal/adapters/storage/sqlite/ -l` returns 8 files (3 production: `repo.go`, `handoff.go`, `embedding_lifecycle_adapter.go`; 5 tests). Gate 8's `mage test-pkg ./internal/adapters/storage/sqlite` requires all transitively-imported packages to compile, including `internal/app`. With Unit 1.2 having deleted `ensureKindCatalogBootstrapped`, the three dangling callers made `internal/app` uncompilable, making gate 8 unreachable. Unit 1.2's waiver covered `mage test-pkg ./internal/app` + `mage ci`, but NOT the transitive compile requirement induced by Unit 1.3's own gate 8. The planner's implicit assumption — that per-package gate isolation would let 1.2's waiver stand through 1.3 — is physically impossible given the import graph.
- **Is a narrower fix available?** NO. The only alternatives to stripping the guards are: (a) reinstate `ensureKindCatalogBootstrapped` as a stub (reverses Unit 1.2's deletion, adds dead code, creates a follow-up delete burden in 1.5), (b) accept that gate 8 is untestable (violates PLAN acceptance). Stripping 9 lines of now-dead guards in files slated for wholesale deletion in Unit 1.5 is the minimum reachable fix.
- **Is the PLAN §1.2 'intentionally skip' intent preserved?** YES. §1.2's intent was "don't touch these files — they die wholesale in 1.5, edits are pure churn". The edits are limited to removing 9 lines of dead-after-Unit-1.2 guard code; no new symbols, no refactors, no API surface churn. When Unit 1.5 deletes the files wholesale, the 9 lines would have been deleted anyway. This matches the same surgical pattern Unit 1.2 used for its four in-scope callers (strip the guard, leave everything else).
- **Side effect disclosure:** stripping the 3 guards implicitly discharges Unit 1.2's `mage test-pkg ./internal/app` and `mage ci` waivers ahead of Unit 1.5's scheduled restoration. Builder explicitly did NOT run `mage test-pkg ./internal/app` from Unit 1.3 (per-package gate remains Unit 1.11's responsibility per plan). No invariant violation.

## Schema-site verification

- **Primary DDL at `repo.go:144-156`** (inside `migrate()`): `CREATE TABLE IF NOT EXISTS projects (id, slug, name, description, metadata_json, created_at, updated_at, archived_at)` — 8 columns, no `kind`. Confirmed via Read at `:144-156`.
- **ALTER site at `:588-591`** (post-migrate hook range): the statement list at `:580-586` contains index creates; the subsequent `ALTER TABLE projects ADD COLUMN metadata_json` at `:593` is unrelated; the old `ALTER TABLE projects ADD COLUMN kind` is gone. Confirmed.
- **Baked INSERTs at `:315-338`**: `CREATE TABLE IF NOT EXISTS kind_catalog` block at `:315-326` is immediately followed by two `INSERT OR IGNORE INTO kind_catalog` statements for `'project'` (`:327-332`) and `'actionItem'` (`:333-338`). Both use RFC3339-compatible `strftime('%Y-%m-%dT%H:%M:%fZ', 'now')` timestamps (parseTS-compatible). Both live in the same `stmts` slice as the DDL, so the schema migration runs them atomically. **Confirmed.**

## SQL-query strip verification (6 functions)

- **`CreateProject` at `:1249-1260`** — INSERT columns `(id, slug, name, description, metadata_json, created_at, updated_at, archived_at)`, 8 `?` placeholders, 8 args. No `kind`. **Confirmed.**
- **`UpdateProject` at `:1262-1277`** — SET `slug = ?, name = ?, description = ?, metadata_json = ?, updated_at = ?, archived_at = ?`, 6 args + id. No `kind`. **Confirmed.**
- **`GetProject` at `:1291-1299`** — SELECT `id, slug, name, description, metadata_json, created_at, updated_at, archived_at`. No `kind`. **Confirmed.**
- **`ListProjects` at `:1301-...`** — SELECT same 8 columns at `:1304-1305`. No `kindRaw` var, no `&kindRaw` Scan handle, no `p.Kind = domain.NormalizeKindID(...)` block. **Confirmed.**
- **`ensureGlobalAuthProject` at `:1347-1363`** — INSERT columns `(id, slug, name, description, metadata_json, created_at, updated_at, archived_at)` at `:1349`, 8 placeholders, 5 actual args + 3 inline literals (`''`, `'{}'`, `NULL`). No `kind`. Function preserved (self-healing auth-project bootstrap). **Confirmed.**
- **`scanProject` at `:3865-3889`** — Scan over `&p.ID, &p.Slug, &p.Name, &p.Description, &metadataRaw, &createdRaw, &updatedRaw, &archived` (8 handles). No `kindRaw`. No `p.Kind = ...` assignment. **Confirmed.**

## Test-site strip + new-test verification

- **`project.SetKind("project-template", now)` call at old `:2369-2371`**: deleted. Diff shows the full 3-line `if err := project.SetKind(...); err != nil { t.Fatalf(...) }` block removed from `TestRepository_PersistsProjectKindAndActionItemScope`. **Confirmed.**
- **`loadedProject.Kind` assertion at old `:2379-2381`**: deleted. Diff shows the 3-line `if loadedProject.Kind != domain.KindID("project-template") { t.Fatalf(...) }` block removed; `loadedProject, err := ...` rewritten to `_, err = ...` since the var is otherwise unused. **Confirmed.**
- **`TestRepository_SeedDefaultKindsIncludeNestedPhaseSupport` at old `:2333-2354`**: deleted wholesale (22 lines in diff). The test asserted `phase` kind presence with nested-phase parent scopes; post-collapse the `phase` row no longer exists in `kind_catalog`. Deleting this test is the unavoidable consequence of seeder deletion; builder flagged this in Deviation #3. **Correct classification.**
- **New `TestRepositoryFreshOpenKindCatalog`** (diff +2566 through +2605): asserts exactly 2 rows, IDs `["actionItem", "project"]` in sorted order, guards against both over-seed and under-seed failure modes. **Verified by reading the test body.**
- **New `TestRepositoryFreshOpenProjectsSchema`** (diff +2607 through +2644): queries `pragma_table_info('projects')`, rejects any column named `kind`, and guards against the 0-column false-pass (table-missing-entirely would otherwise silently satisfy the "no kind column" assertion). **Verified.**

## Git diff cross-check vs worklog

`git diff HEAD --stat` shows exactly 7 files modified:

1. `internal/adapters/storage/sqlite/repo.go` — 161 line churn — matches worklog's edit items 1-13.
2. `internal/adapters/storage/sqlite/repo_test.go` — 110 line churn — matches items 14-18.
3. `internal/adapters/storage/sqlite/template_library_test.go` — 1 line add (the `t.Skip` at `:14`) — matches item 22.
4. `internal/app/template_library.go` — 3 lines removed — matches item 19.
5. `internal/app/template_library_builtin.go` — 6 lines removed (2 × 3-line guards) — matches items 20-21.
6. `workflow/drop_1_75/BUILDER_WORKLOG.md` — 71 lines added — the Unit 1.3 Round 1 worklog entry.
7. `workflow/drop_1_75/PLAN.md` — 2 lines churn — single `todo → done` flip on `### 1.3` header.

No unexplained files touched. No files missed. Worklog file list matches diff exactly.

## P-Findings (Proof Gaps)

- None. All 8 gates PASS, all scope-expansion claims verified, schema-site + SQL-query + test-site verifications all clean.

**Informational (not a 1.3 finding):** `scripts/drops-rewrite.sql:230` contains `SELECT COUNT(*) FROM projects WHERE kind <> 'project'` — an invariant check in the post-migration rewrite. This is a dev-run SQL script outside Unit 1.3's declared paths. It will break when executed against a post-1.3 schema (no `kind` column). PLAN.md §1.3 paths list only `repo.go` + `repo_test.go`, and drops-rewrite.sql appears to be planned under Unit 1.14 per §1.14 (dev-run after Unit 1.75 Go ships). Flagged for drop-orch awareness, not a Unit 1.3 defect.

## Hylla Feedback

None — Hylla answered everything needed. This QA pass verified schema/SQL/test-site strips via `git diff HEAD` + `rg` + `Read` over committed Go source. The touched files are all in the Unit 1.1 + Unit 1.2 + Unit 1.3 edit window (Hylla is stale for those files per project CLAUDE.md rule #2, "Changed since last ingest: use git diff"), so lexical tools are correct. Gate 8 was verified via `mage test-pkg` rather than any Hylla query because the gate is a runtime compile + test pass, not a symbol-query. No Hylla miss to record.
