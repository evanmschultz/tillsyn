# Plan QA Falsification — Round 8

**Verdict:** fail

## Summary

Round 8 produces two major unmitigated counterexamples and two minor/editorial findings against the post-Round-7-triage plan. The two majors are: (F1) the DEV REMINDER Phase 6 anchor is ambiguous/contradictory — the parenthetical places it at Phase 6 *exit*, the prose at Phase 6 *entry*, and WORKFLOW.md Phase 6 Verify never actually includes the dev running `drops-rewrite.sql` against the real DB, so the true trigger is a gap between Phase 6 exit and Phase 7 Closeout; and (F2) unit 1.3's site-list for SQL-query strips omits `GetProject` SELECT at `repo.go:1398` from the named sub-bullets even though the `rg` acceptance catches it. Every F2-series attack on DROP COLUMN itself (SQLite version, hidden deps, migration-replay, schema-healing) was refuted.

## Findings

### F1 — DEV REMINDER Phase 6 anchor is ambiguous and contradicts WORKFLOW.md

**Severity:** major
**Unit:** 1.14
**Counterexample:** Orchestrator re-reads PLAN.md unit 1.14 at drop-end and tries to surface the F3 re-prompt. The plan line 275 says:

> "surface this callout at the transition into Phase 6 Verify (after unit 1.15's `mage ci` + push + `gh run watch --exit-status` returns green, before dev runs the one-shot `sqlite3 ~/.tillsyn/tillsyn.db < scripts/drops-rewrite.sql` step)"

Two internal contradictions:

1. **Prose vs parenthetical disagree.** "Transition *into* Phase 6 Verify" means Phase 6 entry (before `mage ci` runs). The parenthetical "after... `gh run watch` returns green" means Phase 6 *exit* (after CI green). These are opposite ends of Phase 6. An orchestrator reading this cannot decide which is authoritative.
2. **WORKFLOW.md Phase 6 never includes the dev-DB script run.** WORKFLOW.md:148-158 defines Phase 6 Verify as exactly three machine-checkable steps: `mage ci` from `main/`, `git push`, `gh run watch --exit-status`. The dev running `sqlite3 ~/.tillsyn/tillsyn.db < scripts/drops-rewrite.sql` is **not inside Phase 6**. It happens after Phase 6 completes, before or during Phase 7 Closeout, but WORKFLOW.md doesn't name it as a phase step anywhere. The plan anchors to "Phase 6 Verify" but the actual trigger moment (dev types the sqlite3 command against the real DB) is outside WORKFLOW's named phase transitions.

Concrete failure shape: if the orchestrator interprets "transition *into* Phase 6" literally, the reminder fires before unit 1.15's CI watch even starts — too early, dev has no reason to look at the SQL script yet. If the orchestrator follows the parenthetical, the reminder fires at Phase 6 exit but there is no WORKFLOW-defined phase transition at that point (Phase 6 flows directly into Phase 7 Closeout with no named sub-step for the dev-DB run). A future orchestrator with only the plan (no chat context) cannot derive the intended trigger moment.

**Evidence:**
- Read `drops/WORKFLOW.md:148-158` — Phase 6 Verify steps exhaustively listed; no dev-DB SQL step.
- Read PLAN.md:275 — anchor text reads "at the transition into Phase 6 Verify" but parenthetical times it "after... `gh run watch` returns green", which is Phase 6 *exit*.
- Read PLAN.md:55 — the Scope line says "dev re-runs `scripts/drops-rewrite.sql` against `~/.tillsyn/tillsyn.db` cleanly" as a final verification, not attached to any phase.

**Suggested fix:** Option A — rewrite the trigger anchor as "After Phase 6 Verify completes (CI green on `drop/1.75`) and before Phase 7 Closeout begins, surface this callout as the final drop-orch act in the gap where the dev runs the one-shot `sqlite3 ~/.tillsyn/tillsyn.db < scripts/drops-rewrite.sql` step against the real dev DB." Option B — add an explicit "Phase 6.5 — Dev DB migration" step to WORKFLOW.md and anchor the reminder to it. Option A is cheaper and captures the actual handoff moment; Option B is structurally cleaner but requires a WORKFLOW.md edit that STEWARD must own (PLAN changes don't touch WORKFLOW.md). Prefer Option A.

### F2 — Unit 1.3 sub-bullets omit `GetProject` SELECT at `repo.go:1398`

**Severity:** major
**Unit:** 1.3
**Counterexample:** Unit 1.3's sub-bullets (PLAN.md:107-111) name five specific SQL-query strip sites:

- `CreateProject` at `:1345-1360`
- `UpdateProject` at `:1362-1383+`
- `ensureGlobalAuthProject` at `:1455-1473`
- "List-projects query at `:1418-1452`"
- Second project-read query (`scanProject`) at `:3974-4000`

But `GetProject` at `repo.go:1395-1403` has its OWN SELECT string at `:1398`: `SELECT id, slug, name, description, kind, metadata_json, created_at, updated_at, archived_at FROM projects WHERE id = ?`. This is a distinct SELECT literal, separate from both `ListProjects` (`:1408`) and `scanProject` (a scan function, not a SELECT site). A builder strictly following the sub-bullet list strips the five named sites, leaves `:1398` intact, commits, and the `mage test-pkg ./internal/adapters/storage/sqlite` gate fails at runtime with `no such column: kind` on any `GetProject` call post unit 1.3 (because unit 1.3 also strips the DDL column from `repo.go:152`).

The acceptance regex at PLAN.md:98 (`rg -U 'INSERT INTO projects\([^)]*kind|UPDATE projects[^;]*kind\s*=|SELECT[^;]*kind[^;]*FROM projects' ...`) would catch the miss, but only after the build breaks. Since builders read sub-bullets for site enumeration and use the `rg` gate as a final check, the missing enumeration forces an extra round-trip.

**Evidence:**
- Read `internal/adapters/storage/sqlite/repo.go:1395-1403` — `GetProject` wraps a SELECT string at `:1398` that explicitly names `kind`. `scanProject` consumes the row but does not itself hold the SELECT literal.
- `scanProject` at `:3974-4004` contains only a `Scan(...)` call; the SELECT string lives one layer up in each caller (`GetProject` at `:1398` and `ListProjects` at `:1408`).
- Plan sub-bullet at PLAN.md:111 only names scanProject's internal strip, not the SELECT callers above it.

**Suggested fix:** add one sub-bullet to unit 1.3 between the `ensureGlobalAuthProject` bullet and the list-projects bullet: *"`GetProject` SELECT at `:1398` — remove `kind` from the `SELECT id, slug, name, description, kind, ...` column list (the `scanProject` helper at `:3974` handles the Scan side for both `GetProject` and any other scanProject caller, but the SELECT literal itself lives in each caller)."* Alternatively, rewrite the sub-bullets as "strip `kind` from every SELECT/INSERT/UPDATE `projects` literal in `repo.go` — verified 5 sites: `:1356 (INSERT in CreateProject)`, `:1374 (UPDATE in UpdateProject)`, `:1398 (SELECT in GetProject)`, `:1408 (SELECT in ListProjects)`, `:1458 (INSERT in ensureGlobalAuthProject)`".

### F3 — `SnapshotProject.Kind` field removal does not bump `SnapshotVersion`

**Severity:** minor
**Unit:** 1.6 (or plan-wide)
**Counterexample:** `internal/app/snapshot.go:15-16` defines `const SnapshotVersion = "tillsyn.snapshot.v4"` as the canonical snapshot schema version, validated at `:377`. Unit 1.6 strips the `Kind` field from `SnapshotProject` struct per Round-5 P1, but no unit in the plan bumps `SnapshotVersion` to `v5`. JSON round-trip is soft-compatible (dropped fields decode silently, added fields ignore unknown), so this doesn't produce a hard failure. However, the version string now falsely claims "v4 schema" for a struct whose domain-mapped shape no longer matches the pre-drop v4 `SnapshotProject`. Snapshot consumers that validate strictly (or tools external to the binary that pin to v4) would see semantic drift without a version cue.

**Evidence:**
- Read `internal/app/snapshot.go:15-16` — `SnapshotVersion` constant.
- Read `internal/app/snapshot.go:377-378` — validation rejects non-matching versions.
- Read PLAN.md:148 — unit 1.6 Paths touches `snapshot.go` but does not mention version bump.
- `cmd/till/main.go:1899` registers `snapshot export` CLI command, so snapshots ARE persisted and transported across binary versions.

**Suggested fix:** add a sub-bullet to unit 1.6: *"Bump `SnapshotVersion` from `tillsyn.snapshot.v4` to `tillsyn.snapshot.v5` at `internal/app/snapshot.go:16` — the removed `SnapshotProject.Kind` field is a schema change even though JSON round-trip is soft-compatible; strict validators need a version cue. Update `snapshot_test.go` fixtures to expect v5."* Or accept the drift explicitly in the plan's Notes section and document that v4 is a schema-stable alias across the projects.kind removal.

### F4 — `sqlite-vec-go-bindings/ncruces` overrides ncruces's embedded SQLite version (documentation gap, not runtime risk)

**Severity:** editorial
**Unit:** 1.14 (or plan-wide)
**Counterexample:** PLAN.md:277 documents the SQLite version floor as "SQLite 3.35.0 / March 2021; `modernc.org/sqlite` is well past that threshold". But this project uses **`github.com/ncruces/go-sqlite3`** as the direct driver (per `go.mod:65`), not `modernc.org/sqlite` (`modernc.org/sqlite v1.46.1` appears as a transitive indirect at `go.mod:112`). Worse, `repo.go:16` also imports `_ "github.com/asg017/sqlite-vec-go-bindings/ncruces"` which registers a custom SQLite WASM binary via `init()` that **overrides** `sqlite3.Binary` from the ncruces embed (per `sqlite-vec-go-bindings@v0.1.6/ncruces/init.go:20-22`).

Runtime impact: the Go binary's migration code never runs `ALTER TABLE ... DROP COLUMN` (that's dev's one-shot CLI). The dev's sqlite3 CLI is 3.51.0 (verified via `sqlite3 -version`). So the actual script execution works fine. But the plan's version-floor justification cites the wrong driver. If someone later re-examines the floor, they will chase `modernc.org/sqlite` and miss the sqlite-vec WASM override layer entirely.

**Evidence:**
- Read `drop/1.75/go.mod:65` — `github.com/ncruces/go-sqlite3 v0.23.3` as direct require.
- Read `drop/1.75/go.mod:112` — `modernc.org/sqlite v1.46.1 // indirect` (transitive, not direct).
- Read `internal/adapters/storage/sqlite/repo.go:16-20` — project imports `ncruces/go-sqlite3` + `sqlite-vec-go-bindings/ncruces`.
- Read `/Users/evanschultz/go/pkg/mod/github.com/ncruces/go-sqlite3@v0.23.3/embed/README.md:3` — ncruces embeds SQLite 3.49.1.
- Read `/Users/evanschultz/go/pkg/mod/github.com/asg017/sqlite-vec-go-bindings@v0.1.6/ncruces/init.go:20-22` — sqlite-vec's init overrides `sqlite3.Binary`.
- `sqlite3 -version` → `3.51.0` (dev's CLI that actually runs the script).

**Suggested fix:** update PLAN.md:277 to cite the actual driver chain: *"SQLite 3.35.0 / March 2021. Dev runs `drops-rewrite.sql` via their local `sqlite3` CLI (3.51.0, verified), not via the Go binary. The Go binary's driver chain is `ncruces/go-sqlite3 v0.23.3` + `sqlite-vec-go-bindings/ncruces v0.1.6` override — both well past 3.35.0 — but the binary never runs DROP COLUMN itself, only the dev CLI does."*

## Attacks Attempted (No Counterexample Found)

- **F2 hidden-deps on `projects.kind`.** No CREATE TRIGGER, CREATE VIEW, CHECK constraint, generated column, or FK references `projects.kind`. `rg 'CREATE TRIGGER|CREATE VIEW|CHECK \(.*kind|GENERATED ALWAYS|AFTER (INSERT|UPDATE) ON projects'` against `internal/adapters/storage/sqlite/` returns zero hits. The Round-7 F2 triage is structurally sound.
- **F2 SQLite version floor.** ncruces bundles SQLite 3.49.1 (verified via embed/README.md); dev's sqlite3 CLI is 3.51.0 (verified); both support `DROP COLUMN` (requires 3.35.0+). The sqlite-vec WASM override is presumed to be a similarly-recent build (v0.1.6 released post-2024, well after 3.35). Not a runtime risk even if the plan cites the wrong driver (that's editorial — F4 above).
- **F2 migration-replay attack.** Plan unit 1.3 explicitly strips BOTH the `:152` primary schema column `kind TEXT NOT NULL DEFAULT 'project'` AND the `:588` migration-hook `ALTER TABLE projects ADD COLUMN kind`. No other schema-healing logic exists — `rg 'pragma_table_info|isDuplicateColumnErr|missingColumn|hasColumn|columnExists'` against the sqlite adapter confirms only the explicit `:588` hook adds columns. Fresh installs post-drop skip the column entirely; pre-existing migrated DBs get the column dropped via the dev's script; no mechanism re-adds it.
- **F2 DB-already-migrated attack.** Same result — no schema-validation "healing" logic in `internal/adapters/storage/sqlite/` scans `pragma_table_info` and re-adds missing columns. The `isDuplicateColumnErr` helper is only used by explicit ALTER TABLE calls, all of which unit 1.3 strips for `projects.kind`.
- **F4 memory-drift.** `project_drop_1_75_unit_1_14_f3_decision.md` exists under `~/.claude/projects/.../memory/` and is indexed in `MEMORY.md:45`. MEMORY.md is auto-loaded on every session start, so the memory survives compaction.
- **P1 line offsets at `repo_test.go:2365-2385`.** Verified directly via Read. `SetKind` call at `:2369`, error check `t.Fatalf` at `:2370`, closing `}` at `:2371`. Assertion `if loadedProject.Kind != ...` starts at `:2379`, `t.Fatalf` body at `:2380`, closing `}` at `:2381`. Round-7 P1 correction stands.
- **Round 6 F1/F2 citation drift.** Read and verified all four citations:
  - `internal/adapters/server/common/app_service_adapter_helpers_test.go:259` — `domain.ErrBuiltinTemplateBootstrapRequired` case entry. ✓
  - `internal/adapters/server/mcpapi/handler_test.go:938` — `errors.Join(common.ErrBuiltinTemplateBootstrapRequired, ...)`. ✓
  - `internal/adapters/server/httpapi/handler.go:425` — `errors.Is(err, common.ErrBuiltinTemplateBootstrapRequired)`. ✓
  - `internal/app/kind_capability.go:762` and `:776` — `templateDerivedProjectAllowedKindIDs` and `initializeProjectAllowedKinds`. ✓
- **Unit 1.5 atomicity split.** Unit 1.5 lists ~45 files across 7 packages but is defensibly atomic: `internal/adapters/server/mcpapi` imports `internal/app` imports `internal/domain`, and the template-library surface spans all three. Splitting into 1.5a (domain), 1.5b (app+adapter), 1.5c (CLI) would leave intermediate unsatisfied-reference compile states worse than the current single unit. The plan acknowledges this explicitly at PLAN.md:143. The waiver strategy (1.4 waives app-package compile until 1.5 re-greens) is sound.
- **INSERT-default reliance.** Test INSERTs at `repo_test.go:1028, :1245, :1357` and `embedding_lifecycle_adapter_test.go:322` omit the `kind` column from the column list (they rely on the NOT NULL DEFAULT). Post-drop these continue to work because the column is gone entirely — no default needed. No plan change required.
- **`project_allowed_kinds` orthogonality.** FK from `project_allowed_kinds.project_id` → `projects(id)` and `project_allowed_kinds.kind_id` → `kind_catalog(id)`. Zero FK references `projects(kind)`. Unrelated to DROP COLUMN.

## Hylla Feedback

N/A — task touched non-Go files only (PLAN.md, WORKFLOW.md) and read-only Go code inspection used `Read` + `Grep` + Context7 as designed. Hylla is Go-only today and is not the right tool for verifying PLAN.md line offsets against live code. One shape gripe: Context7 for `/ncruces/go-sqlite3` returned API docs but not a straightforward "bundled SQLite version" answer; had to fall through to reading the module cache's `embed/README.md` directly. Suggestion for Context7 ergonomics: include a "bundled runtime versions" doc snippet on drivers that embed external runtimes (SQLite WASM, lua, etc.) so the version floor lookup doesn't require a module cache fallback.
