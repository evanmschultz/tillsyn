# DROP_1_75_KIND_COLLAPSE — Plan QA Proof (Round 2)

**Verdict:** pass (with non-blocking accuracy notes — see P4, P5, P10)

Scope is coherent, in-scope items cover the 5 locked decisions, and every load-bearing citation verifies against `drop/1.75/` HEAD. Three citations drift cosmetically from reality (line-range end, "new" tag on an existing file, grep count) but none of them alter the scope shape or block the Phase 1 planner. Planner section remains a placeholder — acceptable per the round-2 contract which QAs scope, not the unit list.

## Findings

### P1. camelCase `actionItem` decision is grounded in live code

- **Claim:** PLAN Scope line 13: "camelCase, matching the Go constant `WorkKindActionItem = \"actionItem\"` at `internal/domain/workitem.go:39` and the live `kind_catalog` row".
- **Evidence:** `internal/domain/workitem.go:39` reads `WorkKindActionItem WorkKind = "actionItem"` exactly as cited.
- **Verdict:** verified. Lock decision 1 (camelCase) is correctly anchored to a concrete, unchanged constant.

### P2. `internal/domain/task.go` → `action_item.go` rename target is accurate

- **Claim:** PLAN Scope item 3: "`internal/domain/task.go` → `internal/domain/action_item.go` (file is already `type ActionItem struct{}`, only the filename is stale)".
- **Evidence:** `internal/domain/task.go:24` reads `type ActionItem struct {`. `internal/domain/action_item.go` does not exist. Grep confirms `task.go:1` match for `work_items|WorkItem` and `task.go:38` match for `WorkKind|type Kind`, consistent with a file that already holds `ActionItem` as its primary type but references `WorkKind` in fields.
- **Verdict:** verified.

### P3. `seedDefaultKindCatalog` start line is accurate

- **Claim:** PLAN Scope item 1 + Notes line 46: function lives at `internal/adapters/storage/sqlite/repo.go:1231-1247` and "re-seeds 7 legacy kinds on every DB open".
- **Evidence:** `repo.go:1231` is the `func (r *Repository) seedDefaultKindCatalog(ctx context.Context) error {` signature. Lines 1240–1247 enumerate exactly 7 records: `project`, `actionItem`, `subtask`, `phase`, `branch`, `decision`, `note`. 7 kinds confirmed. `seedDefaultKindCatalog` is called from the open path (Grep finds references in `repo.go` and the migration orchestration).
- **Verdict:** verified (with a refinement in P4 about where the function *ends*).

### P4. `seedDefaultKindCatalog` end-line citation is slightly wrong (non-blocking)

- **Claim:** PLAN Scope item 1 + Notes line 46: function range is `repo.go:1231-1247`.
- **Evidence:** The function body actually extends to line 1301 (`return nil\n}` at 1300–1301). Lines 1231–1247 are only the `records := []seedRecord{ ... }` literal. Lines 1248–1300 contain the `appliesJSON` marshal loop, the `INSERT OR IGNORE`, a `GetKindDefinition` round-trip, a `mergeKindAppliesTo` call, and an `UPDATE kind_catalog` merge step. Deleting the function means deleting ~70 lines, not ~17.
- **Verdict:** needs-evidence (correction, not rejection). Builder must not stop the delete at line 1247. Recommend PLAN be edited by planner in Phase 1 to cite `repo.go:1231-1301` (or simply "the full `seedDefaultKindCatalog` function") to avoid a partial excise in Phase 4.

### P5. `internal/domain/kind.go` is NOT "new" — file already exists

- **Claim:** PLAN Paths line 5: "`internal/domain/kind.go` (new, split from `workitem.go`)".
- **Evidence:** `internal/domain/kind.go` already exists (Glob confirmed). Its current content defines `KindID`, `KindAppliesTo`, `KindTemplateChildSpec`, `KindTemplate`, `KindDefinition`, `KindDefinitionInput` — all kind-catalog infrastructure types, none of which are the `WorkKind` enum currently in `workitem.go`. Scope item 3 reads "move `WorkKind` (renamed `Kind`) + its constants into `internal/domain/kind.go`" — correctly implying a *merge into the existing file*, not a new-file creation.
- **Verdict:** needs-evidence (Paths-line tag drift). The Paths annotation `(new, split from workitem.go)` contradicts the Scope body. Planner must reconcile in Phase 1: the file is pre-existing; what is "new" is the arrival of `WorkKind → Kind` constants inside it.

### P6. `Kind` rename will sit alongside existing `KindID` (hidden coupling to surface in Phase 1)

- **Claim:** PLAN Scope item 3 implicitly: `WorkKind` (string-alias type for kind-as-string like `"actionItem"`) renamed to `Kind`, placed in `kind.go`.
- **Evidence:** `internal/domain/kind.go:13` already declares `type KindID string`. `KindID` is the same conceptual payload (kind-as-string identifier); `WorkKindActionItem WorkKind = "actionItem"` and `DefaultProjectKind KindID = "project"` both hold kind strings but under two different Go types. Renaming `WorkKind → Kind` without deciding the `Kind` vs `KindID` relationship will either produce two redundant type aliases (`Kind` and `KindID`) or force a collapse that the current scope doesn't spell out.
- **Verdict:** unsupported (hidden-dependency). Not a blocking defect for this plan-QA round — "pass" stands — but planner in Phase 1 MUST either (a) collapse `KindID` and `Kind` to one type, or (b) explicitly document why both types survive. Surfaced here so Phase 1 catches it before Phase 4.

### P7. `bridgeLegacyActionItemsToWorkItems` citation verifies

- **Claim:** PLAN Scope item 2 + Notes line 47: shim at `repo.go:1184-1228`, translates legacy `tasks` into `action_items`.
- **Evidence:** `repo.go:1184` reads `func (r *Repository) bridgeLegacyActionItemsToWorkItems(ctx context.Context) error {`. Body is an `INSERT INTO action_items(...) SELECT ... FROM tasks t WHERE NOT EXISTS(...)` ending at line 1228 with `return nil\n}`. Matches the cited range exactly.
- **Verdict:** verified.

### P8. `ensureGlobalAuthProject` self-healing claim verifies

- **Claim:** PLAN Notes line 44: `repo.go:1455-1473`, runs on every DB open with `INSERT ... ON CONFLICT(id) DO NOTHING`.
- **Evidence:** `repo.go:1455` reads the function signature `func (r *Repository) ensureGlobalAuthProject(ctx context.Context) error {`. Body contains the exact `INSERT INTO projects(...) VALUES(...) ON CONFLICT(id) DO NOTHING` pattern, closing at line 1473 with `return nil\n}`. Matches cited range exactly.
- **Verdict:** verified. Dev-cleanup of the `__global__` row is safe — this migration hook rebuilds it.

### P9. Schema `CREATE TABLE` locations match claims about what needs to change

- **Claim:** PLAN Paths line 5: `internal/adapters/storage/sqlite/schema.go` (or wherever `CREATE TABLE work_items` lives).
- **Evidence:** Grep for `CREATE TABLE IF NOT EXISTS (kind_catalog|work_items|action_items|tasks)` in `internal/adapters/storage/sqlite/` returns only `repo.go` hits: `tasks` at line 169, `action_items` at line 198, `kind_catalog` at line 316. No separate `schema.go` — schema is inline in `repo.go`. No `CREATE TABLE work_items` hit at all, suggesting the rename from `work_items → action_items` has already substantially landed at the DDL level.
- **Verdict:** verified (with refinement). Planner in Phase 1 should reconcile Paths line: `schema.go` doesn't exist; the CREATEs live in `repo.go`. The `work_items` residue is at Go-identifier / string-reference level (2127 occurrences of `action_items|ActionItem` across 40 files vs 144 of `work_items|WorkItem` across 24 files), not at the DDL level. This narrows the `rg` + `sd` sweep to identifier strings, not schema SQL.

### P10. `template_librar*` surface count drift is cosmetic

- **Claim:** PLAN Scope item 4: "Grep today returns 44 files."
- **Evidence:** Grep for the 6 template-surface patterns across the worktree (excluding `drops/**`) returns 41 files. Including the drop's own coordination files (PLAN.md, DROP_1_75_ORCH_PROMPT.md, README.md, CLAUDE.md) brings the total to 42. The "44" figure in PLAN is 2–3 files high.
- **Verdict:** needs-evidence (cosmetic). Scope item intent is clear — planner will enumerate the final deletion list during Phase 1. The precise count is irrelevant to the scope shape, but PLAN should stop citing "44" once the planner produces the enumerated list.

### P11. `projects.kind` column drop is backed by a real migration step

- **Claim:** PLAN Scope item 5: strip column from schema, `type Project`, MCP handlers, filters, tests, TUI views.
- **Evidence:** Grep returns `repo.go:589` with `migrate sqlite add projects.kind`, confirming the column was added by a migration (so drop needs a symmetric migration). Grep also returns `scripts/drops-rewrite.sql:196` referencing `work_items.kind/projects.kind are plain TEXT without FK`, confirming the SQL script surface where the `ALTER TABLE projects DROP COLUMN kind` will land.
- **Verdict:** verified.

### P12. Seven in-scope items are individually addressable and collectively sufficient

- **Claim:** PLAN line 51 (Notes): "One drop, not split. All seven in-scope items above ship in this drop."
- **Evidence / Trace:** 
  - Item 1 (kind catalog collapse) is a schema migration + single-function excise. 
  - Item 2 (table rename + `WorkKind → Kind` + `tasks` drop) is an `rg` + `sd` sweep with known-shape precedent (`plan_items → action_items`).
  - Item 3 (file + type rename) is a `git mv` + `split` — directly verifiable against `task.go`'s current content.
  - Item 4 (template_libraries excision) is a 41-file delete — Phase 1 planner enumerates exactly.
  - Item 5 (projects.kind drop) has a verified migration surface (P11).
  - Item 6 (drops-rewrite.sql rewrite) depends on items 1–5 landing first — implicit `blocked_by` the Phase 1 planner must wire.
  - Item 7 (tests + fixtures) is diffuse but gated by `mage ci` green.
- **Verdict:** verified. Items are coupled (renames enable SQL script rewrite enables migration), but the coupling is sequential not circular — planner can decompose into units with `blocked_by` chains.

### P13. Out-of-scope items are genuinely deferrable

- **Claim:** PLAN lines 25–32 lists 6 out-of-scope deferrals: `metadata.role` (Drop 2), cascade dispatcher (Drop 4+), template system return (future drop), per-row role hydration (dead-code today), `project_id nullable` (Drop 1), rename of `workitem.go` residual.
- **Evidence / Trace:** Dev DB cleanup (Notes line 43) left only uniform `kind='task', scope='task'` rows, which justifies deferring per-row role hydration (no data to hydrate). `metadata.role` hydration explicitly belongs to Drop 2 per inherited cascade plan. `project_id nullable` is documented as a Drop 1 item in `main`'s CLAUDE.md (project scope). `workitem.go` residual rename is noted as a pure cosmetic follow-up — lifecycle/context/actor/resource enums surviving in that file do not block the `WorkKind → Kind` migration because `workitem.go` is an edit surface, not a rename target.
- **Verdict:** verified. No hidden coupling forces a deferred item back into Drop 1.75's surface.

### P14. Packages line is accurate

- **Claim:** PLAN line 6: Packages = `internal/domain`, `internal/app`, `internal/adapters/storage/sqlite`, `internal/adapters/server/mcpapi`, `cmd/till`, `internal/tui` (conditional).
- **Evidence:** P10's 41-file list spans exactly those packages. Grep for `template_librar*` hits `internal/tui/model.go` and `internal/tui/model_test.go`, so the conditional `internal/tui` qualifier is warranted.
- **Verdict:** verified.

### P15. Verification line is consistent with WORKFLOW Phase 6

- **Claim:** PLAN line 34: "`mage ci` green from `drop/1.75/`, `gh run watch --exit-status` green on `drop/1.75` branch, and dev re-runs `scripts/drops-rewrite.sql` against `~/.tillsyn/tillsyn.db` cleanly."
- **Evidence:** `drops/WORKFLOW.md` Phase 6 specifies `mage ci` + `git push` + `gh run watch --exit-status` as drop-end verification. The extra dev-DB cleanup step is documented in Notes line 50 as same-pattern as `scripts/rename-task-to-actionitem.sql`, which Glob confirms exists alongside `drops-rewrite.sql` in `scripts/`.
- **Verdict:** verified.

## Summary

**Passed:** All 5 round-1 locked decisions (camelCase `actionItem`, `task.go → action_item.go` rename, `seedDefaultKindCatalog` delete, codebase-wide `work_items → action_items` rename, `projects.kind` column drop) are each anchored to a concrete, verifiable code citation. Out-of-scope deferrals hold up. The 7-item scope is exhaustive and sequentially decomposable. Packages line is accurate. Verification line aligns with WORKFLOW Phase 6.

**Non-blocking findings for Phase 1 planner:**
- **P4** — `seedDefaultKindCatalog` spans `repo.go:1231-1301`, not `1231-1247`. Planner should correct the line range when decomposing the unit so the builder deletes the full function (including the merge/update block).
- **P5** — Paths-line annotation `internal/domain/kind.go (new, split from workitem.go)` is wrong; the file already exists with `KindID` + friends. Planner should clarify: "existing file; receiving `WorkKind`/`Kind` constants from `workitem.go`."
- **P6** — `WorkKind → Kind` rename will produce a new `Kind` type adjacent to existing `KindID` in the same file. Planner must either collapse the two or explicitly document why both survive.
- **P9** — No `internal/adapters/storage/sqlite/schema.go` exists; schema `CREATE TABLE`s are inline in `repo.go`. Planner should narrow Paths accordingly.
- **P10** — `template_librar*` file count is 41 today, not 44. Purely cosmetic; the planner-enumerated deletion list supersedes the estimate.

**Does not block Phase 1 planner kickoff.** Scope is strong enough for the planner to decompose into atomic units. The five non-blocking corrections are refinement items the planner naturally handles when building the unit list.

