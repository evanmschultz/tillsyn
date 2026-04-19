# Plan QA Proof — Round 7

**Verdict:** pass

## Summary

Every Round 6 triage claim (F1-F5, P1-P2) verified end-to-end against the committed worktree. Line numbers match, completeness greps clean against the updated 1.5 Paths, F3 column-naming sanity check holds (`kind_id`, not `kind`), F4 pragma-wrap ordering is coherent across all drops-rewrite.sql phases, and the Round 5 regression sample (snapshot.go, repo.go, repo_test.go) remains accurate.

## Findings

None — verdict is pass. One editorial nit recorded below for completeness.

### P1 — repo_test.go:2378 cite is one line off (editorial)
**Severity:** editorial
**Unit:** 1.3
**Claim in plan:** "Test-site strip: `repo_test.go:2369-2371` (the `project.SetKind(...)` call) and `:2378-2381` (the `loadedProject.Kind != domain.KindID("project-template")` assertion and its `t.Fatalf`)."
**Evidence:** Read `repo_test.go:2363-2382`. The `SetKind` call lives at `:2369` (inside the `if` at `:2369-2371`) — matches. The assertion `if loadedProject.Kind != domain.KindID("project-template")` starts at `:2379`, not `:2378`. The `t.Fatalf` is at `:2380`, closing `}` at `:2381`.
**Gap:** Range `:2378-2381` is inclusive of one preceding blank/brace line; not load-bearing but drifts by one line.
**Suggested fix:** adjust to `:2379-2381` next round (optional — builder can locate via `rg "loadedProject\\.Kind != domain\\.KindID"`).

## Confirmed Claims

**F1 — Three cited lines exist with expected content:**
- `internal/adapters/server/common/app_service_adapter_helpers_test.go:259` — `{name: "builtin template bootstrap", err: domain.ErrBuiltinTemplateBootstrapRequired, wantErr: ErrBuiltinTemplateBootstrapRequired}` — exact match.
- `internal/adapters/server/mcpapi/handler_test.go:938` — `err: errors.Join(common.ErrBuiltinTemplateBootstrapRequired, errors.New("missing typed kinds")),` — exact match.
- `internal/adapters/server/httpapi/handler.go:425` — `case errors.Is(err, common.ErrBuiltinTemplateBootstrapRequired):` — exact match.

**F1 — Completeness grep (`ErrBuiltinTemplateBootstrapRequired` sitewide, excluding `drops/**`):** 11 hits across 9 files. Each hit accounted for:
- `internal/domain/errors.go:31` — definition; handled by 1.4 (errors.go listed; preserves `ErrInvalidKindTemplate` only).
- `internal/app/template_library_builtin.go:101` — wholesale delete in 1.5.
- `internal/app/template_library_test.go:659-660` — wholesale delete in 1.5.
- `internal/adapters/server/mcpapi/handler_test.go:938` — in 1.5 Paths (F1).
- `internal/adapters/server/common/app_service_adapter_helpers_test.go:259` — in 1.5 Paths (F1).
- `internal/adapters/server/mcpapi/handler.go:855` — in 1.5 Paths (explicit).
- `internal/adapters/server/httpapi/handler.go:425` — in 1.5 Paths (F1).
- `internal/adapters/server/common/app_service_adapter.go:597-598` — in 1.5 Paths (explicit).
- `internal/adapters/server/common/mcp_surface.go:14-15` — in 1.5 Paths (F5 explicit).

**F2 — Both function signatures confirmed at cited lines:**
- `kind_capability.go:762` — `func templateDerivedProjectAllowedKindIDs(projectKind domain.KindID, library *domain.TemplateLibrary) []domain.KindID` — exact match.
- `kind_capability.go:776` — `func (s *Service) initializeProjectAllowedKinds(ctx context.Context, project domain.Project, library *domain.TemplateLibrary) error` — exact match.

**F2 — Caller sweep (`templateDerivedProjectAllowedKindIDs|initializeProjectAllowedKinds`):** 4 external callers. All covered:
- `internal/app/service.go:211` — `initializeProjectAllowedKinds(ctx, project, nil)`; service.go in 1.5 Paths.
- `internal/app/service.go:304` — `initializeProjectAllowedKinds(ctx, project, allowlistLibrary)`; service.go in 1.5 Paths.
- `internal/app/template_library.go:299` — wholesale delete in 1.5.
- `internal/app/template_library.go:462` — wholesale delete in 1.5.

Plan's "if the functions become trivial after the parameter removal, delete them and their callers instead of keeping stub shells" clause covers the post-1.5 state where `*domain.TemplateLibrary` has been deleted by 1.4 and both functions lose their template-derived branches.

**F2 — Broader `*domain.TemplateLibrary` / `domain.TemplateLibrary` sweep:** 60+ hits across `internal/app/{service.go, service_test.go, template_library.go, template_reapply.go, snapshot.go, ports.go}`, `internal/tui/{model.go, model_test.go}`, `internal/adapters/storage/sqlite/repo.go`, `internal/adapters/server/mcpapi/{instructions_explainer.go, extended_tools_test.go}`, `internal/adapters/server/common/{app_service_adapter_mcp.go, mcp_surface.go}`. Every file is in 1.5 Paths (either explicit file mention or wholesale file deletion via 1.5's template_library*/template_contract*/template_reapply clauses).

**F3 — Column name confirmed:** `CREATE TABLE IF NOT EXISTS project_allowed_kinds` at `repo.go:328-335` declares columns `project_id TEXT NOT NULL, kind_id TEXT NOT NULL, created_at TEXT NOT NULL`. The Round-6 F3 Option A assertion uses `kind_id NOT IN ('project','actionItem') OR kind_id IS NULL` — matches schema exactly.

**F4 — Pragma-wrap ordering is coherent:**
- Phase 2 (DROP TABLE template cluster): with template child tables dropped before parent `template_libraries`, RESTRICT FKs (`project_template_bindings.library_id → template_libraries` at `repo.go:424`, `template_node_templates.node_kind_id → kind_catalog` at `:374`, `template_child_rules.child_kind_id → kind_catalog` at `:392`) never fire because child rows are gone first. Pragma OFF/ON not needed here.
- Phase 3 (DELETE FROM kind_catalog WHERE id NOT IN ...): `project_allowed_kinds.kind_id → kind_catalog(id) ON DELETE CASCADE` (`:334`) fires cleanly by design; F3's Option A assertion at Phase 7 catches the residue.
- Phase 4 (projects table rebuild via 12-step): `PRAGMA foreign_keys = OFF` prevents CASCADE-destruction of `action_items`, `project_allowed_kinds`, and every other `REFERENCES projects(id) ON DELETE CASCADE` child (21 such FKs in `repo.go`). `PRAGMA foreign_keys = ON` closes the wrap before Phase 5.
- Phase 5 (DROP TABLE tasks): no FK points at `tasks` (action_items references `projects`, not `tasks`). Safe with FKs ON.
- Phase 6 (UPDATE action_items SET kind='actionItem'): non-destructive, FK-agnostic.
- Phase 7 assertions: run with FKs ON (post-wrap), including F3's new `project_allowed_kinds` check.

No other phase needs the pragma toggle.

**F5 — Line numbers verified:**
- `mcp_surface.go:14` — doc comment `// ErrBuiltinTemplateBootstrapRequired reports that builtin template operations hit a runtime DB missing prerequisite kinds.`
- `mcp_surface.go:15` — `var ErrBuiltinTemplateBootstrapRequired = errors.New("builtin template bootstrap is required")`

**P1 — All four operation-string literals exist in `extended_tools.go`:**
- `"ensure_builtin"` at `:2019` (Enum), `:2028` (auth doc), `:2071` (switch case).
- `"bind_project_template_library"` at `:604` (enum literal), `:2203` (operation name); tool name `"till.bind_project_template_library"` at `:2171`.
- `"get_template_library"` at `:2273` (encode-error fmt); tool name `"till.get_template_library"` at `:2258`.
- `"upsert_template_library"` at `:2129`, `:2310`, `:2326`; tool name `"till.upsert_template_library"` at `:2281`.

Acceptance grep is non-vacuous.

**P2 — `scanProject` function location verified:**
- `repo.go:3974` — `// scanProject handles scan project.` (leading comment).
- `repo.go:3975` — `func scanProject(s scanner) (domain.Project, error) {`.
- `:3978` — `kindRaw string` local.
- `:3984` — `s.Scan(&p.ID, &p.Slug, &p.Name, &p.Description, &kindRaw, ...)`.
- `:3990-3992` — `p.Kind = domain.NormalizeKindID(domain.KindID(kindRaw))` + default-fallback block.

Plan cites `:3974-4000` with `p.Kind = ...` at `:3990-3992`. Both correct.

**Round 5 regression re-sample:**
- `snapshot.go:41` — `Kind domain.KindID` field in `SnapshotProject` ✓
- `snapshot.go:395-397` — `domain.NormalizeKindID(p.Kind)` + `p.Kind = domain.DefaultProjectKind` + `s.Projects[i].Kind = p.Kind` ✓
- `snapshot.go:1230-1237` — `snapshotProjectFromDomain` returns struct with `Kind: p.Kind` at `:1235` ✓
- `snapshot.go:1589-1603` — `toDomain` with `kind := domain.NormalizeKindID(p.Kind)` at `:1589`, `Kind: kind` at `:1598` ✓
- `repo.go:1345-1360` — `CreateProject`: `kindID` at `:1351-1354`, INSERT column list includes `kind` at `:1356`, positional arg `string(kindID)` at `:1358` ✓
- `repo.go:1418-1452` — list query: `kindRaw` local at `:1428`, `Scan` at `:1434`, `p.Kind = ...` + default fallback at `:1437-1440` ✓
- `repo_test.go:2368-2381` — `project.SetKind("project-template", now)` at `:2369` and `loadedProject.Kind != domain.KindID("project-template")` assertion at `:2379-2381` ✓ (line `:2378` is inclusive of prior blank; editorial nit above)

**Plan-wide structural checks:**
- `ports.go:24-32` — 9 methods as claimed: `UpsertTemplateLibrary`, `GetTemplateLibrary`, `ListTemplateLibraries`, `UpsertProjectTemplateBinding`, `GetProjectTemplateBinding`, `DeleteProjectTemplateBinding`, `CreateNodeContractSnapshot`, `UpdateNodeContractSnapshot`, `GetNodeContractSnapshot` ✓
- `blocked_by` chain remains acyclic and package-level-blocker-compliant from Round 5.
- DEV REMINDER callout for F3 Option A re-surfacing is narratively intact.

## Hylla Feedback

N/A — task involved reading committed source (primarily Go) but the actual queries were straight file Reads + Greps rather than Hylla semantic searches. Hylla would have been the right tool for "who else imports `*domain.TemplateLibrary`" but the plan cited exact lines, and Read/Grep answered every check. No Hylla miss to report.
