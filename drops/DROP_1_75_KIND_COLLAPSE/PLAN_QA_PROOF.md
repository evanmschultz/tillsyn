# Plan QA Proof — Round 8

**Verdict:** pass

## Summary

Every Round-7 triage claim (F2 native DROP COLUMN, F4 Phase-6 anchor, P1 line-offset fix) verifies against the committed tree. One editorial residue survives in the Scope section (line 30) and should be swept for consistency with the Round-7 F2 decision, but it is non-blocking — no schema dependency, coherence, or line-offset claim fails.

## Findings

### P1 — Scope line 30 still cites the deprecated "SQLite equivalent" rebuild path
**Severity:** editorial
**Unit:** plan-wide (Scope section, not unit 1.14)
**Claim in plan:** `scripts/drops-rewrite.sql` rewrite bullet at line 30:
```
- `ALTER TABLE projects DROP COLUMN kind` (or SQLite equivalent: `CREATE TABLE projects_new` + copy + drop + rename).
```
**Evidence:** Read `drops/DROP_1_75_KIND_COLLAPSE/PLAN.md:30`. The parenthetical was written pre-Round-7 F2, when the plan still assumed a table-rebuild was needed. Round-7 F2 concluded native `ALTER TABLE projects DROP COLUMN kind;` works directly because `projects.kind` has no dependencies (verified below), so the rebuild path is no longer needed.
**Gap:** The Scope bullet contradicts unit 1.14's Phase 4 prose (PLAN.md:277), which explicitly rejects the rebuild path and cites "`modernc.org/sqlite` is well past that threshold". A careful reader now has two conflicting instructions 247 lines apart. This is the kind of drift that Round-7 triage was supposed to clean up.
**Suggested fix:** Strip the `(or SQLite equivalent: ...)` parenthetical from line 30 so the Scope bullet reads `- \`ALTER TABLE projects DROP COLUMN kind\`.` — matches unit 1.14 prose and matches the assertion at line 264.

## Confirmed Claims

- **F2 precondition — no PK on `kind`.** `internal/adapters/storage/sqlite/repo.go:147-157` (`CREATE TABLE IF NOT EXISTS projects`): `id TEXT PRIMARY KEY` on `:148`. `kind TEXT NOT NULL DEFAULT 'project'` at `:152` carries only a `DEFAULT`, no key modifier.
- **F2 precondition — no UNIQUE involving `kind`.** `rg 'UNIQUE.*kind|UNIQUE\(.*kind' internal/adapters/storage/sqlite/repo.go` returns one hit at `:372` (`UNIQUE(library_id, scope_level, node_kind_id)` on `template_node_templates`, unrelated to `projects.kind`).
- **F2 precondition — no FK referencing or referenced by `projects.kind`.** `rg 'REFERENCES projects' repo.go` returns 17 hits, all `REFERENCES projects(id)`. Zero reference `projects(kind)`. Verified at `:167, :195, :224, :235, :250, :284, :299, :314, :333, :358, :423, :440, :470, :493, :527, :555, :860`.
- **F2 precondition — no index on `projects.kind`.** `rg 'CREATE INDEX' repo.go` returns 32 hits; none reference the `projects` table at all. The `projects` table has zero secondary indexes anywhere in the file.
- **F2 precondition — no trigger or view referencing `kind`.** `rg 'CREATE TRIGGER|CREATE VIEW' repo.go` returns zero matches. The file has no triggers and no views.
- **F2 precondition — SQLite version ≥ 3.35.0.** `go.mod:112` pins `modernc.org/sqlite v1.46.1 // indirect`. `~/go/pkg/mod/modernc.org/sqlite@v1.46.1/CHANGELOG.md:20` records v1.44.2 (2026-01-18) upgrade to SQLite 3.51.2. v1.46.1 ≥ v1.44.2, so the vendored SQLite is ≥ 3.51.2, comfortably past the 3.35.0 / March 2021 threshold for native `DROP COLUMN`.
- **F2 coherence — Phase 4 prose uses single-statement DROP COLUMN.** `PLAN.md:277` (unit 1.14 prose) contains `(4) \`ALTER TABLE projects DROP COLUMN kind;\` — per Round-7 F2, use SQLite's native \`DROP COLUMN\` ...`.
- **F2 coherence — 12-step rebuild and PRAGMA wrapper excised from unit 1.14.** `rg 'PRAGMA foreign_keys = OFF|12-step' PLAN.md` returns only line 277, which uses both phrases **in contrast** ("instead of the 12-step ... rebuild", "eliminates the Round-6 F4 `PRAGMA foreign_keys = OFF/ON` wrapper entirely") rather than prescribing them. No residual prescriptive mention.
- **F2 coherence — Round-6 F4 moot rationale present.** `PLAN.md:277` includes "which was unsound anyway, because `PRAGMA foreign_keys` inside an open `BEGIN TRANSACTION` is a silent no-op per SQLite docs" and "The Round-7 falsification F1 blocker ... is dissolved because no table rebuild happens." Both rationales visible to future readers.
- **F4 — DEV REMINDER Trigger subsection present and anchored to Phase 6 Verify.** `PLAN.md:275` contains `**Trigger (Round-7 F4 anchor):** surface this callout at the transition into Phase 6 Verify (after unit 1.15's \`mage ci\` + push + \`gh run watch --exit-status\` returns green, before dev runs the one-shot \`sqlite3 ~/.tillsyn/tillsyn.db < scripts/drops-rewrite.sql\` step)`. Phase name verified against `drops/WORKFLOW.md:148` — `## Phase 6 — Verify`. Names match.
- **P1 — `repo_test.go` line offsets correct.** `internal/adapters/storage/sqlite/repo_test.go:2369` is `if err := project.SetKind("project-template", now); err != nil {` (SetKind call block spans :2369-:2371). `:2379` is `if loadedProject.Kind != domain.KindID("project-template") {`. `:2380` is `t.Fatalf("expected persisted project kind, got %q", loadedProject.Kind)`. `:2381` is the closing `}`. All three citations in unit 1.3's Paths + the `1.3` description at PLAN.md:112 match the tree.
- **Round 5/6 survivors — `internal/app/snapshot.go`.** `:41` = `Kind domain.KindID \`json:"kind,omitempty"\``. `:395-397` = `NormalizeKindID(p.Kind)` / `p.Kind = domain.DefaultProjectKind` / `s.Projects[i].Kind = p.Kind`. `:1230-1237` = `snapshotProjectFromDomain` struct literal with `Kind: p.Kind` at `:1235`. `:1589-1603` = `kind := domain.NormalizeKindID(p.Kind)` at `:1589`, `Kind: kind` at `:1598`. All four citations land on the claimed code.
- **Round-6 F2 survivors — `internal/app/kind_capability.go`.** `:762` = `func templateDerivedProjectAllowedKindIDs(projectKind domain.KindID, library *domain.TemplateLibrary) []domain.KindID`. `:776` = `func (s *Service) initializeProjectAllowedKinds(ctx context.Context, project domain.Project, library *domain.TemplateLibrary) error`. Both signatures still carry the `library *domain.TemplateLibrary` parameter unit 1.5 is slated to strip. Citations accurate.
- **Round-6 F1 survivors — adapter test + httpapi handler.** `internal/adapters/server/common/app_service_adapter_helpers_test.go:259` = test-table entry `{name: "builtin template bootstrap", err: domain.ErrBuiltinTemplateBootstrapRequired, wantErr: ErrBuiltinTemplateBootstrapRequired}`. `internal/adapters/server/mcpapi/handler_test.go:938` = `err: errors.Join(common.ErrBuiltinTemplateBootstrapRequired, errors.New("missing typed kinds"))`. `internal/adapters/server/httpapi/handler.go:425` = `case errors.Is(err, common.ErrBuiltinTemplateBootstrapRequired):`. All three citations verified.
- **Scripts file present with expected size.** `scripts/drops-rewrite.sql` = 296 lines, matching unit 1.14's "296-line multi-phase script" description.
- **Legacy `tasks` index citations — `:558, :665`.** `repo.go:558` = `CREATE INDEX IF NOT EXISTS idx_tasks_project_column_position ON tasks(...)`. `repo.go:665` = `CREATE INDEX IF NOT EXISTS idx_tasks_project_parent ON tasks(...)`. Unit 1.7 deletion targets verified.
- **`ALTER TABLE projects ADD COLUMN kind` at `:588`.** `repo.go:588` = `ALTER TABLE projects ADD COLUMN kind TEXT NOT NULL DEFAULT 'project'`. Unit 1.3 deletion target verified — matches the `F2` quoted-DDL guard at unit 1.15's acceptance (line 290).

## Hylla Feedback

N/A — task touched non-Go files only. PLAN.md and WORKFLOW.md are markdown, and the Go code was browsed via `Read` / `Grep` for specific line-anchor verification (the line numbers encoded in the plan are the primary artifact under review, and `Read` with explicit offsets is the right tool for that). Hylla's summary-level views would not have surfaced `:2379` vs `:2378` line-level drift.
