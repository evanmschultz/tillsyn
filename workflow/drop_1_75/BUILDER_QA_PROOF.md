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

## Unit 1.4 — Round 1

**Verdict:** PASS

### Summary

Unit 1.4 (domain-layer `TemplateLibrary` / `TemplateReapply` / `NodeContractSnapshot` / `BuiltinTemplate` excision) is fully supported by the evidence. All 6 in-scope ripgrep gates re-run clean. `mage test-pkg ./internal/domain` is green (49/49 passing). All 4 target files are deleted. The one disclosed relocation — `canonicalizeActionItemToken` moved byte-identical from deleted `template_library.go` into `kind.go` — verifies byte-identical and remains the sole declaration in the package, with the consumer (`NormalizeKindID` at `kind.go:176`) compiling cleanly under the package test run. No stealth orphans remain. Gate 7 (`mage build` / `mage ci`) is explicitly waived by PLAN.md §1.4 and was not executed, per the orchestrator instruction. Downstream compile errors in `internal/app/**` and `internal/adapters/**` are the expected Unit 1.5 surface and are not counted as findings.

### Gate Evidence

1. **Gate 1** `rg 'TemplateLibrary|TemplateReapply|NodeContractSnapshot|BuiltinTemplate' internal/domain/` → 0 matches (rg exit 1). **PASS.**
2. **Gate 2** `mage test-pkg ./internal/domain` → `49 tests passed across 1 package`, 0 failures, 0 skipped, 0 build errors. Exit 0. **PASS.**
3. **Gate 3** `rg -F 'ErrTemplateLibraryNotFound' internal/domain/errors.go` → 0 matches (rg exit 1). **PASS.**
4. **Gate 4** `rg 'ErrInvalidTemplate(Library|LibraryScope|Status|ActorKind|Binding)' internal/domain/errors.go` → 0 matches (rg exit 1). **PASS.**
5. **Gate 5** `rg 'ErrBuiltinTemplateBootstrapRequired|ErrNodeContractForbidden' internal/domain/errors.go` → 0 matches (rg exit 1). **PASS.**
6. **Gate 6** `rg 'ErrInvalidKindTemplate' internal/domain/errors.go` → exactly 1 match (line 25, `ErrInvalidKindTemplate = errors.New("invalid kind template")`). **PASS.**
7. **Gate 7** `mage build` / `mage ci` **waived** per PLAN.md §1.4 Acceptance bullet ("workspace compile-broken between this unit's commit and 1.5's commit by design"). Not executed. **HONORED.**

### File-Deletion Verification

| File | `ls` result | `git status` |
| --- | --- | --- |
| `internal/domain/template_library.go` | No such file or directory | `D  internal/domain/template_library.go` |
| `internal/domain/template_library_test.go` | No such file or directory | `D  internal/domain/template_library_test.go` |
| `internal/domain/template_reapply.go` | No such file or directory | `D  internal/domain/template_reapply.go` |
| `internal/domain/builtin_template_library.go` | No such file or directory | `D  internal/domain/builtin_template_library.go` |

All 4 files absent from working tree; all 4 staged as deletions in git index. Rerunnable: `ls internal/domain/template_library.go internal/domain/template_library_test.go internal/domain/template_reapply.go internal/domain/builtin_template_library.go` returns 4 "No such file or directory" errors; `git status --porcelain -- internal/domain/` lists 4 `D` lines for those paths.

### Relocation Soundness Check (`canonicalizeActionItemToken`)

- **Single declaration.** `rg -c '^func canonicalizeActionItemToken' internal/domain/*.go` → `internal/domain/kind.go:1`. Sole definition in the package.
- **Byte-identical move (visual diff).** Dumped `git show HEAD:internal/domain/template_library.go` to `/tmp/old_tl.go` and compared the `canonicalizeActionItemToken` block (HEAD `:270-300`) against the current `internal/domain/kind.go:179-209` block segment-by-segment:
  - Doc comment: identical 4-line block ("`canonicalizeActionItemToken` rewrites the lowercase `actionitem` token…Token boundaries are start-of-string, end-of-string, `-`, and `_`.").
  - Signature: `func canonicalizeActionItemToken(lowered string) string` — identical.
  - Constants `token = "actionitem"`, `canonical = "actionItem"` — identical.
  - Control flow (early-return guard, `strings.Builder` + `Grow(len(lowered))`, for-i loop, boundary-check `leftOK`/`rightOK`, `WriteString(canonical)` / `WriteByte(lowered[i])`, final `return b.String()`) — identical statement-for-statement.
  - No semantic change. Relocation is byte-identical per builder claim.
- **Consumer compiles.** `internal/domain/kind.go:176` (`return KindID(canonicalizeActionItemToken(strings.ToLower(trimmed)))`) is inside `NormalizeKindID`; `mage test-pkg ./internal/domain` (Gate 2) returned 49/49 pass with 0 build errors, proving the call site compiles and the function remains callable. Test `TestNormalizeKindID` exercises this code path per the package's standing `domain_test.go` coverage.
- **No collision with `NormalizeTemplateLibraryID`.** The old sibling caller (`NormalizeTemplateLibraryID` at HEAD `template_library.go:262-268`) died with the file. `NormalizeKindID` is now the sole consumer, matching the builder's relocation rationale.

### Stealth-Orphan Sweep

- Enumerated every top-level declaration in the 4 deleted files via `git show HEAD:...` — 24 exported + 14 unexported decls in `template_library.go`, 7 types in `template_reapply.go`, 3 types in `builtin_template_library.go`, 0 test decls worth checking in `template_library_test.go` (tests are self-contained).
- `rg 'Template|NodeContractSnapshot|BuiltinTemplate|cloneProjectMetadata|cloneActionItemMetadata|normalizeTemplate|cloneNodeTemplates|cloneTemplateLibrary|cloneOptionalTemplateLibrary|newNodeTemplate|newTemplateChildRule|normalizeTemplateActorKinds|normalizeTemplateActorType|normalizeTemplateNullableTS|validTemplateLibraryScopes|validTemplateLibraryStatuses|validTemplateActorKinds' internal/domain/` returns only hits on `KindTemplate` / `KindTemplateChildSpec` / `normalizeKindTemplate` / `ErrInvalidKindTemplate` — all **intentionally preserved** `Kind*Template*` symbols in `kind.go`, distinct from the deleted `TemplateLibrary` family. Per PLAN.md §1.4 F5 classification, `ErrInvalidKindTemplate` is kept for `normalizeKindTemplate` (naturally unreachable but retained until a refinement drop). No match on any of the 24 + 14 + 7 + 3 deleted symbols — no stealth orphan.
- `rg 'cloneProjectMetadata|cloneActionItemMetadata' internal/domain/` → 0 matches. Both helpers dead with their file.
- `rg 'ReferencedKindIDs' internal/domain/` → 0 matches. The `TemplateLibrary.ReferencedKindIDs` method died with its receiver type.

### `git diff HEAD` vs Worklog Cross-Check

- Worklog §"Files deleted (wholesale, via `git rm`)" table lists 4 files with line counts 786, 113, 98, 39 (total 1036). `git diff HEAD --stat` reports: `template_library.go | 786 ----`, `template_library_test.go | 113 ----`, `template_reapply.go | 98 ----`, `builtin_template_library.go | 39 --`. **Exact match.**
- Worklog §"errors.go — sentinels removed (8)" lists 8 error-var names. `git diff HEAD -- internal/domain/errors.go` shows a single deletion hunk removing exactly 8 `Err*` lines in the sentinel block:  `ErrInvalidTemplateLibrary`, `ErrInvalidTemplateLibraryScope`, `ErrInvalidTemplateStatus`, `ErrInvalidTemplateActorKind`, `ErrInvalidTemplateBinding`, `ErrBuiltinTemplateBootstrapRequired`, `ErrTemplateLibraryNotFound`, `ErrNodeContractForbidden`. **Exact match.**
- Worklog §"Relocation repair" documents `canonicalizeActionItemToken` addition to `kind.go`. `git diff HEAD -- internal/domain/kind.go` shows a single insertion hunk of exactly the function + its 4-line doc comment immediately after `NormalizeKindID` at `:179`. **Exact match.**
- `git status --porcelain -- internal/domain/` reports 6 entries (4 `D`, 2 `M` on `errors.go` + `kind.go`). Worklog §"Files touched" claims "5 files in `internal/domain` (4 deleted, 1 edited, 1 relocation-repair into `kind.go`)". The "1 edited" + "1 relocation-repair" reads as two distinct file edits (which matches `errors.go` + `kind.go`), though the "5 files" arithmetic counts the relocation as an overlap of the kind.go edit — minor cosmetic wording, not a substantive miscount. Not a finding.

### Findings

- **P-findings:** none.
- **Informational:** Worklog's "5 files" count wording in §"Files touched" could be more precise (`errors.go` edit + `kind.go` edit = 2 distinct modifications; total touched paths = 6, not 5). Purely cosmetic; does not impact verdict.

### Hylla Feedback

N/A — task touched non-Go files only from the Unit-1.1–1.3 edit window where Hylla is stale per project CLAUDE.md rule #2 ("Changed since last ingest: use `git diff`"). This QA pass verified excisions + relocation via `git diff HEAD`, `git show HEAD:`, `rg`, `Read`, and `mage test-pkg` over committed-since-ingest Go source. No Hylla query was issued; no fallback was forced. Recording "None — Hylla answered everything needed" as the closing stance would also be accurate — both framings agree there is no miss to record.

## Unit 1.5 — Round 1 — QA Proof

**Verdict:** PASS-WITH-NOTES

### Summary

Unit 1.5's core claim — workspace compile-restoration burden discharged, template_library call-site excision complete across all 7 packages in §1.5 Paths — is fully supported by the committed + staged evidence. `mage build` succeeds clean (Unit 1.4 waiver discharged). Zero dangling TemplateLibrary/TemplateReapply/NodeContractSnapshot/BuiltinTemplate references in any Go source file. All 21 test failures are reproducible, correctly classified, and correctly routed to Units 1.11 (13 in `internal/app`) / 1.12 (8 across `mcpapi`/`httpapi`/`common`). Scope-expansion edits (help.go restore, service_test.go fakeRepo seeding, project_cli_test.go imports) are all minimum-necessary to discharge §1.5's gate. Two informational findings: (a) the builder's worklog understates one substantive transformation — `ensureActionItemCompletionBlockersClear` was **simplified** (not just relocated) from 53 LOC with node-contract blocker collection to 16 LOC with only `CompletionCriteriaUnmet` checks; this is correct-by-construction after NodeContractSnapshot deletion but worth documenting; (b) one scope-expansion file (`cmd/till/cli_render.go` orphan case-label strip) was edited but not mentioned in the worklog.

### Confirmed Claims

- **Gate 1 (fresh re-run):** `rg 'TemplateLibrary|TemplateReapply|NodeContractSnapshot|BuiltinTemplate' drop/1.75/ --glob='*.go'` → 0 files found. **Pass.**
- **Gate 1 extended (repo identifiers):** `rg 'template_librar|node_contract_snapshot|project_template_binding|template_node_template|template_child_rule' drop/1.75/ --glob='*.go'` → 0 files found. **Pass.**
- **Gate 2 (MCP tool deregistration):** `rg 'till\.bind_project_template_library|till\.get_template_library|till\.upsert_template_library|till\.ensure_builtin|"ensure_builtin"' drop/1.75/internal/adapters/server/mcpapi/` → 0 files found. **Pass.**
- **Gate 4 (`mage ci` — PARTIAL, per builder):** Verified by decomposition:
  - `mage build` → `[SUCCESS] Built till from ./cmd/till`, exit 0. **Unit 1.4 compile waiver is DISCHARGED — zero build errors sitewide.**
  - `mage testPkg ./internal/app` → 179 tests, 166 passed, **13 failed**, 0 skipped. All 13 failures cite `kind definition not found: "subtask" | "branch" | "phase"`.
  - `mage testPkg ./internal/adapters/server/mcpapi` → 87 tests, 85 passed, **2 failed** (`kind definition not found: "branch"`).
  - `mage testPkg ./internal/adapters/server/httpapi` → **2 failed** (kind-not-found variants).
  - `mage testPkg ./internal/adapters/server/common` → 96 tests, 92 passed, **4 failed** (`kind definition not found: "branch"`).
  - Total: 13 + 2 + 2 + 4 = **21 failures — exact match with builder's claim.**
- **21-failure routing (sampled by `Grep func <name>`):**
  - 7 in `internal/app/kind_capability_test.go` → §1.11 Paths (PLAN.md:217). ✓
  - 5 in `internal/app/service_test.go` → §1.11 Paths. ✓
  - 1 in `internal/app/snapshot_test.go` → §1.11 Paths. ✓
  - 2 in `internal/adapters/server/mcpapi/handler_integration_test.go` → §1.12 Paths (PLAN.md:231). ✓
  - 2 in `internal/adapters/server/httpapi/handler_integration_test.go` → §1.12 Paths. ✓
  - 3 in `internal/adapters/server/common/app_service_adapter_auth_context_test.go` → §1.12 Paths. ✓
  - 1 in `internal/adapters/server/common/app_service_adapter_lifecycle_test.go` → §1.12 Paths. ✓
  - **All 21 lie in §1.11 or §1.12 Paths. None are §1.5's responsibility.**
- **help.go restoration (`--template-json` compatibility-only guidance):** The hidden flag still exists at `cmd/till/main.go:850-851` (`kindUpsertCmd.Flags().StringVar(&kindUpsertOpts.templateJSON, "template-json", ...)` + `mustMarkFlagHidden(kindUpsertCmd, "template-json")`). Parser `parseOptionalKindTemplateJSON` at `main.go:2927` still consumes it and returns `domain.KindTemplate`. Restored help text at `help.go:130-131` truthfully reflects runtime state. Test `cmd/till/main_test.go:625` asserts `"compatibility-only"` is present in the long-form help output — the restoration makes this test pass without re-enabling any deleted code path. **Legitimate §1.5 scope** (needed to make `mage testPkg ./cmd/till` compile + test-pass).
- **service_test.go fakeRepo seeding:** `internal/app/service_test.go:45-76` (`newFakeRepo`) now seeds `{project, actionItem}` into `kindDefs`. This mirrors the real sqlite schema bootstrap at `internal/adapters/storage/sqlite/repo.go` (Unit 1.3's baked `INSERT OR IGNORE INTO kind_catalog` after `CREATE TABLE`). The docstring at `:38-44` accurately explains the invariant: `ensureKindCatalogBootstrapped` (deleted Unit 1.2) used to perform this seed at service-first-use; the in-memory fake never runs SQL migrations, so it needs an equivalent compile-time seed. **Not masking real bootstrap behavior** — real DBs still bootstrap via Unit 1.3's baked rows. The 13 app-package failures listed above hit this seed and STILL fail (because they try to create items with `"subtask"` / `"phase"` kinds that the seed intentionally doesn't carry), confirming the seed is narrow and not suppressing the §1.11-owned test-site issues. **Legitimate invariant-preserving fix.**
- **project_cli_test.go scope-expansion (5 unused imports):** Unstaged-layer diff strips `context`, `path/filepath`, `sqlite`, `config`, `uuid` imports. These became orphans because the staged-layer diff deleted `TestRunProjectCreateUsesTemplateLibrary` (which exercised `app.UpsertTemplateLibraryInput{...}` via sqlite-backed fixtures) — a template-library test whose body is legitimately §1.5 territory. Stripping only the orphan imports (not the full file) is the minimum-necessary fix. `project_cli_test.go` IS listed in §1.6 Paths (line 149) and §1.13 Paths (line 244) for Project.Kind strips and Kind-test-fixture work respectively; Unit 1.5's edits are import-strip-only and **do not pre-empt** later units' substantive edits. **Legitimate minimum-necessary scope-expansion** per the orchestrator directive.
- **go.mod/go.sum state:** `git status go.mod go.sum` → clean (no changes). `Grep` of live Go source confirms `chroma` (in `internal/tui/gitdiff/highlighter.go`, `internal/tui/file_viewer_renderer.go`), `glamour` (in `internal/tui/markdown_renderer.go`), `douceur` (transitive), `x/exp/golden` (in `third_party/teatest_v2/teatest.go`), and `golang.org/x/exp` (transitive) are still live-referenced by production code. The "go.mod diagnostics" mentioned in the spawn prompt are **not reflected in on-disk `go.mod`** — they appear to be a transient gopls/IDE state, not a real `go mod tidy` divergence. **No go.mod churn needed for §1.5.**
- **Wholesale file deletions verified:** `git status --short` confirms D marks on `cmd/till/template_cli.go`, `cmd/till/template_builtin_cli_test.go`, `internal/app/template_{library,library_builtin,library_builtin_spec,library_test,contract,contract_test,reapply}.go`, `internal/adapters/storage/sqlite/template_library_test.go`. All match §1.5 Paths.
- **Snapshot struct field strip:** `internal/app/snapshot.go` diff (lines 17-33 of diff): `TemplateLibraries []domain.TemplateLibrary`, `ProjectBindings []domain.ProjectTemplateBinding`, `NodeContracts []domain.NodeContractSnapshot` fields all removed from `type Snapshot struct`. Matches §1.5 Paths directive ("strip `TemplateLibraries` field + `snapshotTemplateLibraryFromDomain` + `upsertTemplateLibrary` + `normalizeSnapshotTemplateLibrary` sections").
- **Repository interface strip:** `internal/app/ports.go` diff removes 9 lines. Matches §1.5 Paths directive ("strip the 9 `TemplateLibrary*` / `NodeContractSnapshot*` / `ProjectTemplateBinding*` methods").

### Informational Findings

- **I-1 (undisclosed simplification):** `ensureActionItemCompletionBlockersClear` relocated from deleted `internal/app/template_contract.go:74-125` (53 LOC, collected node-contract blockers from children + descendants, sorted + joined via `; `) to `internal/app/mutation_guard.go:22-38` (16 LOC, only retains the `actionItem.CompletionCriteriaUnmet(activeChildren)` check). The transformation is **correct-by-construction** — `NodeContractSnapshot` is deleted, so the blocker-collection branch was unreachable — and is invariant-preserving for the surviving `CompletionCriteriaUnmet` code path. **Builder's worklog understates this**: §1.5 worklog mentions "stripped template service fields + bindings" but does not call out the 53→16 LOC simplification of a method still used by `service.go:667`. This is **not a correctness finding** (the simplified version is what the code must be post-NodeContractSnapshot-deletion), but it's a worklog-completeness gap. **Route:** no action required; flagging for future worklog discipline.
- **I-2 (undocumented scope-expansion file):** `cmd/till/cli_render.go` had 9 orphan case labels stripped from `commandProgressLabel` (`template.library.list`, `template.library.show`, `template.library.upsert`, `template.builtin.status`, `template.builtin.ensure`, `template.project.bind`, `template.project.binding`, `template.project.preview`, `template.contract.show`). File is NOT in §1.5 Paths and is NOT mentioned in the worklog. The edit is legitimate scope-expansion — the CLI template commands were deleted wholesale in §1.5, so their `commandProgressLabel` branches are unreachable. **Not a correctness finding** (the strip matches reality); **worklog disclosure gap** only. **Route:** no action required.
- **I-3 (Gate 1 doc-file residuals):** Builder's worklog notes non-Go `TemplateLibrary` / etc. refs remain in `AGENT_CASCADE_DESIGN.md`, `CLAUDE.md`, `DROP_1_75_ORCH_PROMPT.md`, `README.md`. Verified — `rg 'TemplateLibrary' drop/1.75/ --glob='*.md'` returns hits in those four files. The "pure-collapse, no doc churn" scope-boundary reading is defensible; doc scrubs are a separate concern from the Go-collapse work. **No §1.5 finding.** **Route:** consider a doc-scrub refinement drop or a Unit 1.15 acceptance-gate clarification.

### Attack Surface Verdicts

- **Gate 2 PARTIAL acceptability:** **PASS.** All 21 failures reproducibly hit files that PLAN.md explicitly assigns to §1.11 / §1.12 Paths. Failure signatures (`kind definition not found: "branch" | "phase" | "subtask"`) are the exact shape Unit 1.3 engineered via its catalog-collapse (deleting those rows from `kind_catalog`). Routing is accurate; no mis-classified failure belongs to §1.5.
- **help.go restore:** **PASS.** Hidden `--template-json` flag is live; restored guidance is truthful; test `main_test.go:625` requires it.
- **service_test.go fakeRepo seeding:** **PASS.** Intent-preserving invariant fix mirroring real sqlite bootstrap; 13 residual app-package failures still surface `kind definition not found` errors for other kinds, proving the seed doesn't suppress real test issues.
- **Scope-expansion for project_cli_test.go:** **PASS.** Import-strip-only; test-body deletion happened in staged layer and was legitimately §1.5 scope (template-library test).
- **go.mod diagnostics:** **PASS.** `go.mod` is clean on-disk; all flagged packages are live-referenced from production code. Diagnostic signal is orch-side/transient, not a real tidy divergence. **No action needed in §1.5.**
- **Completeness:** **PASS.** Zero dangling template_library/NodeContractSnapshot/BuiltinTemplate refs across all `.go` files in `drop/1.75/`.
- **Unit 1.4 waiver discharge claim:** **PASS.** `mage build` runs clean with exit 0.

### Overall Verdict

**PASS-WITH-NOTES.** Unit 1.5's declared objective (workspace compile-restoration) is fully discharged. The "21 failures out-of-scope" routing is audit-clean. Two informational findings (I-1, I-2) are worklog-completeness gaps, not correctness defects — neither blocks §1.5 `done` state. §1.11 and §1.12 inherit the 21 failures as planned; no plan rework needed.

### Route to Later Units

- **Unit 1.11** (app test-site updates): owns 13 failures — 7 in `kind_capability_test.go`, 5 in `service_test.go`, 1 in `snapshot_test.go`. All `kind definition not found: "subtask" | "phase" | "branch"` variants. Also owns coverage restoration (`internal/app` at 69.3%, threshold 70%).
- **Unit 1.12** (adapter + MCP test-site updates): owns 8 failures — 2 in `mcpapi/handler_integration_test.go`, 2 in `httpapi/handler_integration_test.go`, 3 in `common/app_service_adapter_auth_context_test.go`, 1 in `common/app_service_adapter_lifecycle_test.go`. Also owns coverage restoration (`internal/adapters/server/common` at 62.7%).

### Hylla Feedback

N/A — task touched files wholly within the Unit 1.1–1.5 edit window where Hylla is stale per project CLAUDE.md rule #2 ("Changed since last ingest: use `git diff`"). This QA pass verified excisions, scope-expansion boundaries, test-failure routing, and helper-relocation semantics entirely via `git diff HEAD`, `git show HEAD:`, `Grep`, `Read`, `mage build`, and `mage testPkg`. No Hylla query was issued; no fallback was forced.

## Unit 1.7 — Round 1 — QA Proof

**Date:** 2026-04-20
**Target:** §1.7 (Legacy `tasks` table excision) — `internal/adapters/storage/sqlite/repo.go`, `internal/adapters/storage/sqlite/repo_test.go`.

### Re-Verified Gates

| Gate | Command | Builder Result | QA Re-Run |
| ---- | ------- | -------------- | --------- |
| 1 | `rg 'CREATE TABLE( IF NOT EXISTS)? tasks\|ALTER TABLE tasks\|UPDATE tasks\|FROM tasks\|INSERT INTO tasks\|idx_tasks_' drop/1.75/internal/` | 0 matches | 0 matches — PASS |
| 2 | `rg 'bridgeLegacyActionItemsToWorkItems\|migratePhaseScopeContract' drop/1.75/internal/` | 0 matches | 0 matches — PASS |
| 3 | `mage test-pkg ./internal/adapters/storage/sqlite` | 68/68 pass, 0 skip, 1.01s | 68/68 pass, 0 skip, 0.99s — PASS |

All three declared §1.7 acceptance gates re-confirmed against the working tree.

### Scope Bounds Verified

- §1.7 Paths: `repo.go`, `repo_test.go`. `git diff --stat` on these two files: `-223` / `-220` lines, pure deletions, zero additions. Scope-correct.
- Other modified files (`cmd/till/project_cli.go`, `internal/app/*`, `internal/domain/project.go`, `internal/tui/*`, `internal/adapters/server/mcpapi/*`) match Unit 1.6's declared scope verbatim (BUILDER_WORKLOG.md lines 440-469). Unit 1.6 + 1.7 share one uncommitted staging window — last commit `06e98a0` ("excise template_libraries from call sites") is Unit 1.5. No §1.7 bleed into non-§1.7 files.

### Deviation D1 Verdict — Test Function-Wide Delete (PASS)

Builder deleted the whole `TestRepository_MigratesLegacyActionItemsTable` function (220 lines) rather than just the PLAN-cited fixture range `:1006-1049`. Reviewed the full deleted test body via `git diff`:

- **Assertion 1** (lines ~1053-1081 of deleted block — `PRAGMA table_info(tasks)` + `seenParentID` check): validates the ALTER-TABLE-parent_id migration on the `tasks` table. Dies with the tasks table by definition.
- **Assertion 2** (lines ~1083-1099 of deleted block — `COUNT(*) FROM action_items WHERE id='t1'` + `GetActionItem("t1")` + kind/state checks): validates the bridge's row-copy behavior. Dies with `bridgeLegacyActionItemsToWorkItems`.
- **Assertion 3** (`change_events` table exists + `actor_name` column): covered by `TestRepository_MigratesLegacyCommentAndEventOwnership` at `:975-1094` — that test inserts into `change_events` with an `actor_id, actor_type` legacy shape, then asserts the post-migrate `actor_name` read-back (`:1088-1092`). Equivalent direct coverage.
- **Assertion 4** (`comments` table exists + `actor_id`, `actor_name`, `summary` columns, no `author_name`): also covered by `TestRepository_MigratesLegacyCommentAndEventOwnership` — it reads `actor_id, actor_name, summary` off `comments` post-migration for c1/c2/c3 fixtures. Equivalent direct coverage.
- **Assertion 5** (`attention_items` table exists): indirect coverage via `TestRepository_AttentionItemRoundTrip` (`:1772`), `TestRepository_AttentionItemValidationErrors` (`:1878`), `TestRepository_AttentionItemProjectWideRoleFilterAndUpsert` (`:2012`) — all would fail on a missing table. Not a direct `sqlite_master` existence check, but functionally equivalent.
- **Assertion 6** (`idx_comments_project_target_created_at` + `idx_attention_scope_state_created_at`): indirect coverage via the round-trip tests above (the comment-target index drives `ListCommentsByTarget` performance but not correctness — missing the index wouldn't fail the test, only slow it). Direct existence assertion is lost.

**Net coverage loss:** the direct `sqlite_master WHERE name='idx_*'` existence assertions for the two indexes are no longer checked. This is low-severity — missing indexes are performance regressions, not correctness regressions, and would not have been caught by a green 68/68 test run anyway. No functional test is orphaned. Builder's rationale ("the test exists solely to validate the bridge migration we are removing") is accurate for the primary assertions and defensible for the incidental ones.

### Deviation D2 Verdict — Orphan Helper Delete (PASS)

- `rewriteSubphaseKindAppliesTo` — full-tree grep (`rg 'rewriteSubphaseKindAppliesTo' drop/1.75/`) returns 3 hits, all in `workflow/drop_1_75/BUILDER_WORKLOG.md` prose. Zero source-code references. Safe to delete.
- `kindAppliesToEqual` — full-tree grep (`rg 'kindAppliesToEqual' drop/1.75/`) returns 10 hits across `PLAN.md`, `BUILDER_WORKLOG.md`, `BUILDER_QA_PROOF.md` prose only. Zero source-code references. Safe to delete.
- D2 was explicitly anticipated in Unit 1.3 Round 1 worklog (`BUILDER_WORKLOG.md:130` — "Dies in 1.7 together with `migratePhaseScopeContract` itself"). This is planned continuation, not an unplanned expansion.

### Migration-Runner Caller Sites

Confirmed via `git diff internal/adapters/storage/sqlite/repo.go`:

- `r.migratePhaseScopeContract(ctx)` — deletion at diff line 80 (was `repo.go:542`). Gone.
- `r.bridgeLegacyActionItemsToWorkItems(ctx)` — deletion at diff line 92 (was `repo.go:554`). Gone.
- Function bodies at diff lines 102-275 — fully deleted including all internal call sites (`rewriteSubphaseKindAppliesTo` invocations at diff lines 155-156, `kindAppliesToEqual` invocations at diff line 157).

No orphan function-call sites remain.

### Coverage

PLAN.md §1.7 Acceptance does not specify coverage. Project CLAUDE.md sets 70% as a workspace-wide `mage ci` invariant, but `mage ci` is the full gate (compile-waived for §1.5-1.13 per PLAN.md §1.6 Acceptance). `mage test-pkg` does not emit a per-package coverage percentage in normal mode; the test-green signal (68/68, zero skip) is the documented §1.7 bar. Builder met PLAN contract. Coverage re-green is a §1.11/§1.12 concern per prior rounds' routing.

### Overall Verdict

**PASS.** All three declared §1.7 gates re-verified green. Both deviations are correctness-equivalent, PLAN-consistent continuations (D2 was anticipated explicitly in Unit 1.3's worklog; D1 is the minimal internally-consistent extension of PLAN.md's literal `:1006-1049` range to preserve compileable test). Migration-runner caller sites gone. Helpers confirmed zero-caller before delete. No scope bleed. Mage `test-pkg` green, zero skipped tests, no test-escape patterns. Unit 1.7's excision leaves the `sqlite` package functionally equivalent for all non-legacy-bridge surfaces.

Low-severity note on coverage completeness: direct `sqlite_master` existence assertions for `idx_comments_project_target_created_at` + `idx_attention_scope_state_created_at` are no longer asserted post-delete. Not a blocker — the indexes remain declared in `repo.go:424, :429` and the round-trip tests would catch missing *tables* they depend on. File as a future-refinement only if deemed worth adding a one-off schema-presence test to `TestRepositoryFreshOpenProjectsSchema` at `:2388`.

### Hylla Feedback

N/A — Unit 1.7's subject is deletion of exact Go identifiers + exact SQL literals in exactly two files. The question shape is deterministic string-match, not semantic search; `rg` / `Grep` over a known path set is strictly faster and more precise than Hylla for this. Additionally, the code under review is within the Unit 1.1-1.7 staging window where Hylla is stale per CLAUDE.md rule #2. No Hylla query was issued; no fallback was forced.

## Unit 1.6 — Round 1 — QA Proof

**Date:** 2026-04-20
**Reviewer:** go-qa-proof-agent
**Verdict:** **PASS-WITH-NOTES**

### Scope

Unit 1.6 strips `projects.kind` (column already baked out of domain.Project in Unit 1.3's commit; this unit removes downstream Go-layer readbacks). PLAN.md §1.6 paths cover 5 packages: `internal/domain`, `internal/app`, `internal/adapters/server/mcpapi`, `internal/tui`, `cmd/till`. `mage build` / `mage ci` explicitly WAIVED for this unit per PLAN.md §1.6 acceptance (workspace compile-broken between 1.6 and 1.11/1.12/1.13 by design).

### Gate Re-Runs (Independent Verification)

All 4 gates re-run from `/Users/evanschultz/Documents/Code/hylla/tillsyn/drop/1.75/` via `Grep`:

- **Gate 1** `rg -U 'project\.Kind|projects\.kind|Project\{[^}]*Kind' --glob='!workflow/**'` → 1 hit at `scripts/drops-rewrite.sql:196` (SQL comment; excluded by gate's `--glob='!scripts/drops-rewrite.sql'`). **PASS.**
- **Gate 2** `rg 'projectFieldKind'` → 3 hits, all in `workflow/drop_1_75/PLAN.md` and `workflow/drop_1_75/BUILDER_WORKLOG.md` (documentation). Source tree clean. **PASS** (with note — see D3 below; the gate regex as written in PLAN.md lacks the `--glob='!workflow/**'` exclude Gate 1 has).
- **Gate 3** `rg 'tillsyn\.snapshot\.v4' internal/app/` → 0 matches. **PASS.**
- **Gate 4** `rg 'tillsyn\.snapshot\.v5' internal/app/snapshot.go` → exactly 1 hit at `:16` (`const SnapshotVersion = "tillsyn.snapshot.v5"`). **PASS.**

### Per-Surface Evidence

**Domain (`internal/domain/project.go`):**
- `type Project struct` (lines 11-20) reads `ID, Slug, Name, Description, Metadata, CreatedAt, UpdatedAt, ArchivedAt` only. Zero `Kind` field. Verified via Read.
- `func NewProject(id, name, description string, now time.Time)` (line 42) — no kind arg. Verified.
- `SetKind` method absent. `rg 'SetKind' --glob='!workflow/**'` → 0 hits.

**App (`internal/app/snapshot.go`):**
- `type SnapshotProject struct` (lines 33-42) — zero `Kind` field. Verified via Read.
- `snapshotProjectFromDomain` (lines 1050-1061) — no `.Kind` read or write. Verified.
- `SnapshotProject.toDomain()` (lines 1256-1271) — no `.Kind` read or write. Verified.
- `SnapshotVersion` bumped to `tillsyn.snapshot.v5` at line 16. `Snapshot.Version` populated from constant at `:180`; validation at `:326` compares against same constant. No hardcoded `v4` anywhere in tree.
- Remaining `.Kind` hits in snapshot.go (grep: `Kind` → lines 61, 87, 90-91, 100-103, 192, 204, 259, 265, 399-404, 445, 450+, 481-495, 903+, 983+, 1083, 1109-1115, 1294-1296, 1357-1370, etc.) are all `ActionItem.Kind` / `KindDefinition.ID` / `KindAppliesTo` / `ProjectAllowedKinds.KindIDs` — NOT `Project.Kind`. Correct.

**App (`internal/app/service.go`):**
- `CreateProjectInput.Kind` (line 231) and `UpdateProjectInput.Kind` (line 286) — **dead fields, see D1 below**.
- `CreateProjectWithMetadata` (lines 247-279) body never reads `in.Kind`. Verified via Read.
- `UpdateProject` (lines 294-324) body never reads `in.Kind`. Verified via Read.
- Only `in.Kind` reads in service.go are lines 525-526 (`CreateActionItem` / `UpdateActionItem` path — ActionItemInput.Kind, not ProjectInput.Kind). Correct domain separation.

**App (`internal/app/kind_capability.go`):**
- `resolveProjectKindDefinition` absent. `rg 'resolveProjectKindDefinition|validateProjectKind' --glob='!workflow/**'` → 0 hits. Verified deleted.

**MCP adapter (`internal/adapters/server/mcpapi/instructions_explainer.go`):**
- Diff confirms 3 strip sites (rule-append branch, overview interpolation, buildProjectWhyItApplies kind-baseline entry). Per git diff --stat: 6 line delta.
- No remaining `project.Kind` reads in `internal/adapters/server/mcpapi/`. `rg 'project\.Kind' --glob='!workflow/**'` → 0 matches.

**TUI (`internal/tui/model.go`):**
- `rg 'projectKindPicker|modeProjectKindPicker|projectFieldKind' internal/tui/` → 0 hits. Subsystem fully excised — no orphan consts, types, Model fields, handlers, view overlays, or help-panel entries.
- Classification block deletion (D4) examined via `git diff` at `model.go:17830+`. The deleted block renders `kind:` input + `project_kinds: <summary>` list header + per-row iterations. Both depend on `project.Kind` domain field and `projectKindSummaryRows()` / `projectKindPickerOptions()` methods (deleted). Leaving the `classification` section header with no contents would render a visually-broken headerless empty block. **Intent-preserving.**
- `modeAddProject` / `modeEditProject` prompt rewrites (D2) examined via `git diff` at `model.go:20463+`. The `kind opens picker on enter/e/type` clause was surgically removed; all other verbiage (`enter saves`, `i edits description`, `r picks root_path`, `comments opens thread`, `esc cancels`) preserved verbatim. **No UX regression beyond the intentional kind-picker removal.**

**TUI (`internal/tui/thread_mode.go`):**
- `UpdateProjectInput{...}` literal at `:561-569` no longer carries `Kind:`. Verified.

**CLI (`cmd/till/project_cli.go`):**
- `CreateProjectInput{...}` literal at `:143-146` no longer carries `Kind:`. Verified.
- `writeProjectList` / `writeProjectDetail` / `writeProjectReadiness` have no `project.Kind` references. `rg 'project\.Kind' cmd/till/` → 0 hits.

### Deviation Adjudication

- **D1 (CreateProjectInput.Kind / UpdateProjectInput.Kind dead fields):** Confirmed dead in application layer — neither `CreateProjectWithMetadata` nor `UpdateProject` read the field. Two external writers still populate it (adapter layer `internal/adapters/server/common/app_service_adapter_mcp.go:559, :585` and one test fixture `internal/app/kind_capability_test.go:523`), but these writes land in a field no consumer reads. Wire-format consideration: `CreateProjectInput` / `UpdateProjectInput` are unexported-JSON-tag-free domain structs, not HTTP/MCP over-the-wire DTOs; the separate `mcpapi.CreateProjectRequest` struct carries the MCP surface contract. So no "leak as optional empty-string" concern. **Defensible for the §1.6 waiver window.** 1.11 (app package re-green) is the logical cleanup point; PLAN.md §1.6 Paths does not explicitly list these Input struct fields, so leaving them is consistent with the literal scope. **Note** (not blocker): recommend Unit 1.11's description add an explicit bullet to strip these two fields and the three call-site writes at that time.
- **D2 (prompt string rewrites):** Verified surgical — only the kind-picker clause removed; remainder verbatim. Leaving a prompt that advertises a deleted behavior would be a user-visible lie. **Intent-preserving, in scope.**
- **D3 (Gate 2 regex lacks workflow exclude):** Confirmed — the 3 remaining hits are all documentation prose (PLAN.md invariant rows + BUILDER_WORKLOG.md audit trail). Not source code. PLAN drafting oversight, not a builder error. **Accept; flag as a PLAN refinement candidate.**
- **D4 (classification section + system kind: row delete wider than PLAN literal line numbers):** Verified. The deleted content is entirely dependent on `project.Kind` domain field or `projectKindPicker` methods, all of which the unit removes. The `classification` section header is purely contextual for the kind subsection; the system section's `kind: <value>` row reads the deleted domain field. **Intent-preserving.**
- **D5 (template_reapply.go listed in PLAN but already deleted in 1.5):** Confirmed — file absent from working tree. `ls internal/app/template_reapply.go` → not found. **No action needed; flag as a PLAN refinement candidate.**

### Scope-Bound Check

`git status --porcelain` lists modifications to 13 Go files + 2 MD files:

- **Unit 1.6 scope (per PLAN.md §1.6 Paths):** `internal/domain/project.go`, `internal/app/kind_capability.go`, `internal/app/service.go`, `internal/app/snapshot.go`, `internal/adapters/server/mcpapi/instructions_explainer.go`, `internal/adapters/server/mcpapi/extended_tools_test.go`, `internal/tui/model.go`, `internal/tui/model_test.go`, `internal/tui/thread_mode.go`, `cmd/till/project_cli.go`, `cmd/till/project_cli_test.go` — 11 files, all in-scope.
- **Out-of-scope but present in working tree:** `internal/adapters/storage/sqlite/repo.go` (223 deletions) and `internal/adapters/storage/sqlite/repo_test.go` (220 deletions). `git diff` sample confirms these are `CREATE TABLE tasks` + related legacy-tasks-table excision — **Unit 1.7's** scope (PLAN.md §1.7 Paths: `internal/adapters/storage/sqlite/repo.go`, `repo_test.go`). Also confirms zero `project.Kind` touches in sqlite (grep → 0 matches).

**Scope-bleed adjudication:** the sqlite changes are Unit 1.7 work staged alongside Unit 1.6 in the same working tree. Unit 1.6's builder did not touch sqlite (PLAN.md §1.6 explicitly forbids it by omission). Unit 1.7 QA owns verification of those changes. **Not a Unit 1.6 QA blocker** — but the dev should be aware that the pre-commit working tree mixes multiple units' deltas, so the Unit 1.6 commit must stage only the 11 in-scope Go files + the 2 MD files.

### Section 0 Convergence

- (a) **QA Falsification — no unmitigated counterexample.** Attacks A1 (stub Kind refs), A2 (wider form-field removal), A3 (missed golden fixture), A4 (dead field wire-leak), A5 (forbidden-file edit), A6 (gate regex weakening) — each verified clear. D1's dead fields are genuinely unread in-service; D4's wider delete is strictly domain-field-dependent; A3 confirmed via `rg 'tillsyn\.snapshot\.v4'` over the entire tree; A5 confirmed via per-file diff review.
- (b) **QA Proof — evidence complete.** Each claim (4 gates, 5 deviations, 7 surface-level strips, 2 input-field deads, 1 scope-bleed) has direct Grep/Read/git-diff citation above.
- (c) **Unknowns routed.** None unresolved for Unit 1.6. One refinement candidate flagged: PLAN §1.6 Gate 2 regex should gain `--glob='!workflow/**'` exclude (D3 / routed to dev as PLAN refinement). One coordination hint routed: Unit 1.11 description should explicitly list the two `CreateProjectInput.Kind` / `UpdateProjectInput.Kind` struct-field strips + the three call-site writes at `adapter_mcp.go:559, :585` + `kind_capability_test.go:523` (D1 cleanup).

### Verdict

**PASS-WITH-NOTES.**

All 4 gates verified green. All 5 declared deviations adjudicated as defensible within PLAN.md §1.6 waiver window. `projectKindPicker` subsystem fully excised with zero orphan references. SnapshotVersion bump clean. No forbidden-file touches attributable to Unit 1.6. D1's dead fields are a legitimate scoping trade-off the PLAN paths permit — flag for Unit 1.11 explicit cleanup.

**Notes (non-blocking):**
1. Unit 1.11's description should list `CreateProjectInput.Kind` / `UpdateProjectInput.Kind` field removal + 3 call-site writes as explicit strip targets.
2. PLAN.md §1.6 Gate 2 regex should gain `--glob='!workflow/**'` to parallel Gate 1.
3. PLAN.md §1.6 Paths list should drop `internal/app/template_reapply.go` (D5 — already deleted in 1.5).
4. Working tree mixes Unit 1.6 + Unit 1.7 deltas; dev must stage Unit 1.6's 11 Go files + 2 MD files as a discrete commit before reviewing Unit 1.7.

### Hylla Feedback

N/A — Unit 1.6's subject is exact-string removal (Go identifiers, constants, struct fields) across a finite enumerated file list. Question shapes were `which files contain 'project.Kind'`, `is there still a projectKindPicker definition`, `does CreateProjectInput still have a Kind field`. All resolved by `rg` / `Grep` / `Read` over the PLAN-listed paths. Additionally, Unit 1.6's edits are uncommitted at review time, so Hylla's committed-code index is stale per CLAUDE.md rule #2 — fallback to `git diff` + `Read` is expected and correct. No Hylla query was issued; no fallback was forced.

## Unit 1.9 — Round 1 — QA Proof — 2026-04-20

**Verdict:** PASS

## Summary

The builder's claim — 11 lines (`type Kind string` + 5-constant block with godoc) moved from `internal/domain/workitem.go` to `internal/domain/kind.go` — is fully supported by the evidence. All 6 proof-completeness checks pass. The placement deviation (block landed at `kind.go:18-28` after `DefaultProjectKind` at line 16, rather than between `KindID` at line 13 and `DefaultProjectKind` at line 16 as the plan literally directed) is assessed as a correct interpretation of plan intent — builder's placement preserves the idiomatic Go "type followed by its own constants" adjacency pattern that the surrounding file already follows three times (`KindID`+`DefaultProjectKind`, `Kind`+5-const-block, `KindAppliesTo`+5-const-block). The plan's literal instruction would have split `KindID` from its own `DefaultProjectKind`. No remediation needed; flag as a minor PLAN-authoring imprecision, not a builder error.

## Confirmed Claims

- **Check 1 (kind.go contents):** `internal/domain/kind.go:19` contains exactly one `type Kind string` declaration; lines 22-28 contain exactly one const block with `KindActionItem Kind = "actionItem"`, `KindSubtask Kind = "subtask"`, `KindPhase Kind = "phase"`, `KindDecision Kind = "decision"`, `KindNote Kind = "note"`. Values intact, types tagged correctly. **PASS**
- **Check 2 (workitem.go absence):** `internal/domain/workitem.go` lines 1-100 contain no `type Kind string` declaration and no `KindActionItem`/`KindSubtask`/`KindPhase`/`KindDecision`/`KindNote` constants. `git diff internal/domain/workitem.go` confirms a 12-line deletion (original lines 34-44: doc comment + type + const block). **PASS**
- **Check 3 (package-wide uniqueness):** `Grep 'type Kind string' *.go` across entire drop returns exactly 1 hit at `internal/domain/kind.go:19`. `Grep 'KindActionItem\s*Kind\s*=|KindSubtask\s*Kind\s*=|KindPhase\s*Kind\s*=|KindDecision\s*Kind\s*=|KindNote\s*Kind\s*='` returns exactly 5 hits, all at `internal/domain/kind.go:23-27`. Zero duplicate declarations anywhere in Go tree. **PASS**
- **Check 4 (domain tests):** `mage test-pkg ./internal/domain` → `[PKG PASS] github.com/evanmschultz/tillsyn/internal/domain (0.00s)`, 49 tests, 49 passed, 0 failed, 0 skipped, 1 package. Exit 0. **PASS**
- **Check 5 (downstream build):** `mage build` → `[SUCCESS] Built till from ./cmd/till`, exit 0. No compilation breakage in any downstream package referencing `domain.Kind` or the 5 constants. **PASS**
- **Check 6 (godoc preservation):** `git diff internal/domain/kind.go` shows the added block has doc comment `// Kind represents a configurable item kind.` and `// Built-in kind defaults.` byte-identical to the pre-move version in `git show HEAD:internal/domain/workitem.go | head -50` (lines 34-35 of HEAD). Constant declarations within the block retain gofmt-aligned tab spacing. **PASS**

## Placement Deviation Assessment

Plan directed placement **between** `KindID` (line 13) and `DefaultProjectKind` (line 16). Builder placed **after** `DefaultProjectKind`, at lines 18-28.

Analysis:
- Plan's literal instruction would split `KindID` (the type) from `DefaultProjectKind` (the only constant of that type). That is anti-idiomatic — every surrounding block in `kind.go` pairs a type immediately with its related constants (lines 12-16 `KindID`+`DefaultProjectKind`; lines 30-40 `KindAppliesTo`+5-const-block; lines 47-55 same pattern for ContextImportance equivalent in workitem.go would be the norm).
- Builder's placement preserves the type+constants adjacency idiom and produces a clean cadence: `KindID → DefaultProjectKind → Kind → KindActionItem-block → KindAppliesTo → KindAppliesTo-block`.
- Godoc output ordering: Go's `godoc` sorts types and their associated constants by declaration order within the file but groups constants under their declared type; placement inside the same file does not affect the rendered doc for package `domain`.
- Readability: Builder's placement reads cleanly top-to-bottom as three paired type+constants groupings.

Verdict on deviation: **acceptable, objectively better than literal plan text**. No remediation.

**Notes (non-blocking):**
1. PLAN.md §1.9 Paths line (line 194) describes placement as "near top, distinct from existing `type KindID string`" — this wording is consistent with builder's placement. The more specific "between KindID and DefaultProjectKind" framing (if it exists in round notes) is the minor imprecision; builder correctly followed the spirit.
2. No Unit 1.9 commit yet — working tree has Unit 1.9 deltas (`kind.go`, `workitem.go`) plus the staged rename `task.go → action_item.go` likely from Unit 1.8. Dev will stage Unit 1.9's 2 Go files as a discrete commit before advancing.

### Hylla Feedback

N/A — Unit 1.9's subject is an exact file-to-file move of 11 lines within the same package. Evidence needs were: (a) verify presence/absence in two specific files, (b) verify uniqueness sitewide, (c) verify godoc preservation via `git show HEAD`, (d) verify compile+test green via mage. All resolved by `Read` / `Grep` / `Bash git diff` / `Bash mage`. Additionally, Unit 1.9's edits are uncommitted at review time, so Hylla's committed-code index is stale per CLAUDE.md rule #2 — fallback to `git diff` + `Read` is expected and correct. No Hylla query was issued; no fallback was forced.

## Unit 1.8 — Round 1 — QA Proof — 2026-04-20

**Verdict:** PASS

**Summary:** Pure `git mv internal/domain/task.go internal/domain/action_item.go` with zero content change. `similarity index 100%`, blob hashes byte-identical, `mage test-pkg ./internal/domain` green (49/49), `mage build` green, no stale `task.go` references. All 6 PLAN §1.8 proof-completeness checks satisfied with concrete evidence.

### Confirmed Claims

- **Check 1 (rename semantics):** `git status` → `renamed:    internal/domain/task.go -> internal/domain/action_item.go` under "Changes to be committed". Staged as a git rename, not a delete+add. Git rename detection confirmed by both porcelain output and `git diff --cached --stat` showing `internal/domain/{task.go => action_item.go} | 0` (zero line changes).
- **Check 2 (byte-identical content):** `git diff --cached -- internal/domain/action_item.go internal/domain/task.go` emits only `similarity index 100%` + `rename from`/`rename to` headers — no `@@` content hunks. Strongest-possible cross-check: `git hash-object internal/domain/action_item.go` = `0fb8f3f55368426519802bdd726fe2ce2539d289`; `git rev-parse HEAD:internal/domain/task.go` = `0fb8f3f55368426519802bdd726fe2ce2539d289`. Blob hashes match byte-for-byte. Line-count parity: both 384 lines.
- **Check 3 (`mage test-pkg ./internal/domain` green):** Output shows `tests: 49 / passed: 49 / failed: 0 / skipped: 0`, `[SUCCESS] All tests passed`, `[PKG PASS] github.com/evanmschultz/tillsyn/internal/domain`. Matches PLAN §1.8 "49/49 pass" expectation exactly.
- **Check 4 (`mage build` green):** Output `[RUNNING] Building till from ./cmd/till` → `[SUCCESS] Built till from ./cmd/till`. Full downstream link succeeds — no compilation breakage from the rename.
- **Check 5 (no stale `task.go` references in Go source):** `Grep` for `task.go` across `*.go` files project-wide returns zero matches. No test fixture, log message, embedded asset path, or comment still references the old filename.
- **Check 6 (package decl + exported symbols intact):** Head of `internal/domain/action_item.go` shows `package domain` on line 1 and exports `Priority` + constants `PriorityLow`/`PriorityMedium`/`PriorityHigh`, `ActionItem` struct (with full field list), and `ActionItemInput` struct. All expected exports present. Blob-hash identity (Check 2) guarantees every other symbol in the file is unchanged as well.

### Scope Hygiene

- **Unit 1.8 staged delta is isolated to the rename.** `git diff --cached` at the Unit 1.8 scope shows only the rename header; no content modifications staged under this unit.
- **Working tree contains unstaged sibling modifications** (`internal/domain/kind.go`, `internal/domain/workitem.go`, `workflow/drop_1_75/BUILDER_WORKLOG.md`, `workflow/drop_1_75/PLAN.md`) — these belong to Unit 1.9 (`Merge WorkKind block into kind.go`) per PLAN §1.9 and are out of scope for this review. `mage test-pkg` and `mage build` passing in the presence of those unstaged edits is additional defense-in-depth: the rename holds up even with 1.9 in-flight.
- **No forbidden-file touches attributable to Unit 1.8.** The rename is the only staged change.

### Falsification Attempts (All Mitigated)

- *Could git's similarity index misreport a content change as 100% identical?* → Refuted by direct blob-hash identity (`0fb8f3f5...`), which is independent of the similarity heuristic.
- *Could downstream packages break from the rename?* → Refuted by `mage build` success. Go resolves by package path, not filename; `package domain` is preserved (per file head), so zero linkage breakage is expected.
- *Could `mage test-pkg` be green because tests were skipped?* → Refuted; output explicitly reports `skipped: 0`.
- *Could the rename be delete+add disguised?* → Refuted; both `git status` (`renamed:`) and `git diff --cached` (`similarity index 100%`, `rename from`/`rename to`) tag rename semantics explicitly.
- *Could unstaged 1.9 edits be masking a regression?* → Accepted but non-blocking. Verdict scope is the Unit 1.8 staged rename; the 1.9 sibling work runs under its own QA pass.

### Notes (non-blocking)

None. PLAN §1.8's three acceptance gates (`ls internal/domain/task.go` fails, `ls internal/domain/action_item.go` succeeds, `mage test-pkg ./internal/domain` passes) are all satisfied, plus the four additional proof-completeness checks the spawn prompt layered on (byte-identical content, mage build, no stale references, symbol preservation).

### Hylla Feedback

N/A — Unit 1.8 is a pure file rename; Hylla's committed-code index predates the rename and would describe the symbols under the old filename regardless. Question shapes were `does git recognize this as a rename`, `is the content byte-identical`, `does the package still compile and test green`, `are there stale filename references`. All resolved by `git status` / `git diff` / `git hash-object` / `git rev-parse` / `mage` / `Grep`. No Hylla query was issued; no fallback was forced.
