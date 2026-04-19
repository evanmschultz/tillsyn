# Plan QA Falsification — Round 6

**Verdict:** fail

## Summary

Two blocker-severity counterexamples found against Drop 1.75's Round-5-triage PLAN.md. Both concern **unit 1.5's Paths list being incomplete to discharge the 1.4 waiver through `mage ci`**:

- **F1** — three files that reference `domain.ErrBuiltinTemplateBootstrapRequired` (a sentinel 1.4 deletes) are omitted from 1.5's Paths.
- **F2** — `internal/app/kind_capability.go` holds `*domain.TemplateLibrary` in two function signatures (`templateDerivedProjectAllowedKindIDs` and `initializeProjectAllowedKinds`) and is not in 1.5's Paths; after 1.4 deletes `domain.TemplateLibrary`, those signatures will not compile.

Either gap alone prevents 1.5's `mage ci` acceptance criterion from passing, leaving unit 1.4's waiver undischarged and the workspace compile-broken. Plus one major SQL correctness gap (F3), one minor FK-toggle ambiguity (F4), and one editorial gap (F5).

## Findings

### F1 — 1.5's Paths omit three files that reference `ErrBuiltinTemplateBootstrapRequired`

**Severity:** blocker
**Unit:** 1.5
**Attacks:** missing-Paths, waiver-discharge

**Counterexample.** Unit 1.4 deletes `domain.ErrBuiltinTemplateBootstrapRequired` from `internal/domain/errors.go` (its Acceptance bullet explicitly requires `rg 'ErrBuiltinTemplateBootstrapRequired|ErrNodeContractForbidden' internal/domain/errors.go` returns 0). Unit 1.5 is responsible for re-greening the workspace via `mage ci`. `grep` of `ErrBuiltinTemplateBootstrapRequired` across the tree yields references in these files:

- `internal/app/template_library_builtin.go:101` — **in 1.5 Paths** ✓
- `internal/app/template_library_test.go:659-660` — **in 1.5 Paths** ✓
- `internal/adapters/server/common/mcp_surface.go:14-15` (the `var ErrBuiltinTemplateBootstrapRequired = errors.New(...)` re-export) — **in 1.5 Paths** ✓
- `internal/adapters/server/common/app_service_adapter.go:597-598` — **in 1.5 Paths** ✓
- `internal/adapters/server/mcpapi/handler.go:855` — **in 1.5 Paths** ✓
- `internal/adapters/server/common/app_service_adapter_helpers_test.go:259` — **NOT in 1.5 Paths** — uses both `domain.ErrBuiltinTemplateBootstrapRequired` AND the common re-export.
- `internal/adapters/server/mcpapi/handler_test.go:938` — **NOT in 1.5 Paths** — `errors.Join(common.ErrBuiltinTemplateBootstrapRequired, ...)`.
- `internal/adapters/server/httpapi/handler.go:425` — **NOT in 1.5 Paths** — `case errors.Is(err, common.ErrBuiltinTemplateBootstrapRequired):`.

1.5's Paths list for `common/` and `httpapi/` names only `common/mcp_surface.go`, `common/app_service_adapter.go`, `common/app_service_adapter_mcp.go`, `common/app_service_adapter_auth_context.go`, `common/app_service_adapter_auth_context_test.go`, `common/app_service_adapter_mcp_actor_attribution_test.go`, `common/app_service_adapter_lifecycle_test.go`, and `httpapi/handler_integration_test.go`. Neither `common/app_service_adapter_helpers_test.go` nor `httpapi/handler.go` (production handler, non-integration) nor `mcpapi/handler_test.go` appears.

After 1.5's scheduled edits, these three omitted files still reference the deleted `ErrBuiltinTemplateBootstrapRequired` sentinel (directly via `domain.` or transitively via `common.` — and both sources die: the `common.` re-export at `mcp_surface.go:14-15` is itself inside a file 1.5 deletes). `mage ci` will fail compile on all three packages (`common`, `httpapi`, `mcpapi`), so 1.5's `mage ci` succeeds acceptance bullet is unreachable, the 1.4 waiver is undischarged, and every downstream unit inherits a compile-broken workspace.

**Fix direction.** Add `internal/adapters/server/common/app_service_adapter_helpers_test.go`, `internal/adapters/server/mcpapi/handler_test.go`, and `internal/adapters/server/httpapi/handler.go` to 1.5's Paths.

---

### F2 — 1.5's Paths omit `internal/app/kind_capability.go` whose signatures reference `*domain.TemplateLibrary`

**Severity:** blocker
**Unit:** 1.5
**Attacks:** missing-Paths, waiver-discharge, compile-window

**Counterexample.** Unit 1.4 deletes `domain.TemplateLibrary` (the `internal/domain/template_library.go` deletion is the scoped 1.4 excision). `internal/app/kind_capability.go` holds two function signatures that take `*domain.TemplateLibrary`:

- Line `:762` — `func templateDerivedProjectAllowedKindIDs(projectKind domain.KindID, library *domain.TemplateLibrary) []domain.KindID`
- Line `:776` — `func (s *Service) initializeProjectAllowedKinds(ctx context.Context, project domain.Project, library *domain.TemplateLibrary) error`

Unit 1.2's Paths include `internal/app/kind_capability.go`, but 1.2's scoped edits (deletion of `ensureKindCatalogBootstrapped` at `:559-589`, `sync.Once` field, `defaultKindDefinitionInputs` at `:863-874`, and caller updates) do **not** touch these two signatures. 1.2 also runs strictly before 1.4 (1.2 blocked_by: 1.1 only; 1.4 blocked_by: 1.1 only; but 1.5 is blocked_by: 1.2 AND 1.4, so compile-window ordering is 1.1 → {1.2, 1.4} → 1.5). At 1.2 commit time, `domain.TemplateLibrary` still exists, so the signatures compile fine — 1.2 has no reason to touch them.

Unit 1.5's Paths for `internal/app` list `template_library.go`, `template_library_builtin.go`, `template_library_builtin_spec.go`, `template_library_test.go`, `template_contract.go`, `template_contract_test.go`, `template_reapply.go`, `snapshot.go`, `snapshot_test.go`, `service.go`, `service_test.go`, `ports.go`, `helper_coverage_test.go` — but **not** `kind_capability.go`. 1.5's prose describes "strip template service fields + bindings" in `service.go` but nowhere requires the 1.5 builder to edit `kind_capability.go`.

At 1.5 commit time, `domain.TemplateLibrary` is gone (1.4 deleted it) and the two signatures at `kind_capability.go:762,:776` reference a nonexistent type. `mage ci` fails compile on `internal/app` before it can reach any downstream package. 1.5's `mage ci` succeeds acceptance bullet is unreachable; 1.4's waiver is undischarged.

Note also the caller sites: `internal/app/service.go:211` and `:304` call `initializeProjectAllowedKinds(ctx, project, nil)` and `initializeProjectAllowedKinds(ctx, project, allowlistLibrary)` respectively. `service.go` is in 1.5's Paths, but fixing only the callers (dropping the `nil`/`allowlistLibrary` arg) does not help until the callee signature is also changed — and the callee is in `kind_capability.go`, not `service.go`.

Worth noting: `internal/app/template_library.go:299, :462` also reference these functions, but that whole file is deleted in 1.5, so those callers are moot.

**Fix direction.** Add `internal/app/kind_capability.go` to 1.5's Paths with explicit prose directing the builder to strip the `library *domain.TemplateLibrary` parameter from both function signatures (and simplify `initializeProjectAllowedKinds`' body to drop the template branch entirely — every call now passes `nil`-equivalent).

---

### F3 — 1.14 has no re-seed for `project_allowed_kinds` leaving the live `tillsyn` project's allowlist empty post-script

**Severity:** major
**Unit:** 1.14
**Attacks:** 1.14 SQL correctness, blocker-chain

**Counterexample.** `scripts/drops-rewrite.sql` on the current `drop/1.75` branch contains `project_allowed_kinds_only_valid` assertion logic. The live dev DB was manually purged pre-drop (per PLAN Notes, 2026-04-18) but the single surviving `tillsyn` project retains `project_allowed_kinds` rows pointing at legacy kind ids (`task`, `build-task`, `qa-check`, etc.) — the post-rename schema uses `action_items` table with `kind='task'` rows, but `project_allowed_kinds.kind_id` still references the legacy string ids.

1.14's Phase 3 (`DELETE FROM kind_catalog WHERE id NOT IN ('project', 'actionItem')`) deletes every non-surviving kind row from `kind_catalog`. If `project_allowed_kinds.kind_id` has an FK `ON DELETE CASCADE` to `kind_catalog.id` (which it does per the current schema), then every `project_allowed_kinds` row referencing a legacy kind id (`task`, `qa-check`, etc.) silently CASCADE-deletes. Result: the `tillsyn` project ends with zero `project_allowed_kinds` rows — it cannot create any new `actionItem` children until a human re-seeds the table.

1.14's assertion block does NOT check post-script `project_allowed_kinds` state. 1.14 also has no explicit `INSERT INTO project_allowed_kinds (project_id, kind_id) VALUES ('<tillsyn-uuid>', 'actionItem'), ('<tillsyn-uuid>', 'project')` re-seed. On dev's DB, next `till actionItem create ...` attempt against the `tillsyn` project fails with "kind not allowed" — production blast radius.

If the FK is `ON DELETE RESTRICT` instead, the 1.14 Phase 3 DELETE fails outright with an FK constraint error, and the script halts mid-transaction (arguably a better failure mode than silent allowlist emptying, but still a broken end state — 1.14 still never verifies the post-state is allowlist-correct).

**Fix direction.** 1.14 Phase 3 should be split: (a) delete all legacy kind rows from `kind_catalog`, (b) explicitly `DELETE FROM project_allowed_kinds WHERE kind_id NOT IN ('project', 'actionItem')` before Phase 3, (c) explicitly re-seed `INSERT OR IGNORE INTO project_allowed_kinds (project_id, kind_id) SELECT p.id, 'actionItem' FROM projects p` (idempotent re-seed for every surviving project). Plus an assertion: `SELECT COUNT(*) FROM project_allowed_kinds WHERE kind_id NOT IN ('project','actionItem')` returns 0, and `SELECT COUNT(*) FROM project_allowed_kinds WHERE kind_id = 'actionItem'` returns >= 1 (every surviving project can create action_items).

---

### F4 — 1.14 Phase 4 SQLite projects-rebuild unclear on `PRAGMA foreign_keys` handling

**Severity:** minor
**Unit:** 1.14
**Attacks:** 1.14 SQL correctness, phase-order

**Counterexample.** 1.14 Phase 4 (per prose: "SQLite table-rebuild to drop `projects.kind` column") uses the canonical SQLite pattern: CREATE new table → COPY rows → DROP old → RENAME new. If `PRAGMA foreign_keys = ON` at time of DROP old, any FK pointing at `projects.id` with `ON DELETE CASCADE` cascades into child tables — and `action_items.project_id`, `project_allowed_kinds.project_id`, etc. all have FKs to `projects.id`. CASCADE on the temporary DROP destroys every child row.

Canonical SQLite guidance (sqlite.org 12-step pattern) requires `PRAGMA foreign_keys = OFF` before the rebuild and `ON` after. 1.14's prose does not specify this toggle. If the builder writes the rebuild without the toggle (or with toggle in wrong order), every `action_items` row on dev's DB is deleted mid-script.

**Fix direction.** 1.14 prose should explicitly require `PRAGMA foreign_keys = OFF;` at Phase 4 start, the 12-step rebuild, then `PRAGMA foreign_keys = ON;` + `PRAGMA foreign_key_check;` at Phase 4 end. Plus an assertion: `SELECT COUNT(*) FROM action_items` matches pre-Phase-4 count (prove no silent CASCADE losses).

---

### F5 — 1.5's prose does not explicitly flag stripping `ErrBuiltinTemplateBootstrapRequired` var declaration at `mcp_surface.go:14-15`

**Severity:** editorial
**Unit:** 1.5
**Attacks:** description-drift

**Counterexample.** 1.5 lists `internal/adapters/server/common/mcp_surface.go` in Paths, but prose nowhere surfaces the `var ErrBuiltinTemplateBootstrapRequired = errors.New("...")` re-export at `:14-15`. A builder strictly reading prose could interpret "Template libraries app + adapter + CLI excision" to mean only tool-registration and service-wiring surface, missing the error-sentinel re-export in an adapter file whose filename (`mcp_surface.go`) doesn't obviously signal "template library error exports."

**Fix direction.** Add a one-line prose callout: "`common/mcp_surface.go:14-15` — strip the `ErrBuiltinTemplateBootstrapRequired` re-export var; the domain-side sentinel dies in 1.4, this common-layer re-export dies in 1.5."

---

## Attacks Attempted (No Counterexample Found)

These attacks were run and did not produce counterexamples.

- **A1 — Table enumeration under 1.14 DROP TABLE pass.** The 9 template-cluster tables (`template_libraries`, `template_node_templates`, `template_child_rules`, `template_child_rule_editor_kinds`, `template_child_rule_completer_kinds`, `project_template_bindings`, `node_contract_snapshots`, `node_contract_editor_kinds`, `node_contract_completer_kinds`) are all listed. Compared against `internal/adapters/storage/sqlite/repo.go` CREATE TABLE statements at :336, :362, :377, :395, :403, :411, :426, :443, :449 — all 9 covered.

- **A2 — SQL 3-valued logic on action_items kind column.** PLAN 1.14 Acceptance line 265 already covers `OR kind IS NULL` — Round-5 O2 triage addressed this. Attack REFUTED.

- **A3 — Phase-order attack on 1.14 `DROP TABLE` before `DELETE FROM kind_catalog`.** PLAN 1.14 prose line 268 explicitly orders Phase 2 (DROP template tables) before Phase 3 (DELETE kind_catalog rows) to avoid `ON DELETE RESTRICT` trips. Attack REFUTED.

- **A4 — 1.3 test-site coverage.** Unit 1.3 Acceptance line 112 migrates the `repo_test.go:2369-2381` round-trip assertion into 1.3 (not 1.12) so 1.3's `mage test-pkg ./internal/adapters/storage/sqlite` gate stays green. Round-5 P2 addressed this. Attack REFUTED.

- **A5 — 1.6 → 1.7 transitive compile safety.** 1.6 waives `mage ci`, 1.7 runs against sqlite package only (`mage test-pkg ./internal/adapters/storage/sqlite`), so the waiver applies. 1.7 does not depend on workspace-wide compile. Attack REFUTED.

- **A6 — 1.11 Paths completeness for `internal/app/*_test.go` references to deleted symbols.** Grep of `internal/app/*_test.go` for `WorkKind|TemplateLibrary|project\.Kind|SnapshotProject\{.*Kind|SetKind|ensureKindCatalogBootstrapped` returned hits only in the four files 1.11 lists (`kind_capability_test.go`, `service_test.go`, `snapshot_test.go`, `helper_coverage_test.go`) plus `template_library_test.go` and `template_contract_test.go` (both deleted wholesale by 1.5). `search_embeddings_test.go:59` uses `domain.Project{...}` with NO `Kind` field — safe. `embedding_runtime_test.go`, `handoffs_test.go`, `auth_requests_test.go`, `auth_scope_test.go`, `capability_inventory_test.go`, `mutation_guard_test.go`, `attention_capture_test.go`, `schema_validator_test.go`, `live_wait_test.go` all returned zero matches. Attack REFUTED.

- **A7 — YAGNI + complexity pressure on KindDefinition.Template preservation.** PLAN line 53 F5 orphan classification intentionally defers `KindDefinition.Template` / `KindTemplate` / `KindTemplateChildSpec` / `validateKindTemplateExpansion` / `normalizeKindTemplate` / `ErrInvalidKindTemplate` deletion to a refinement drop. Dev direct quote in line 48 approves the orphan-via-collapse pattern. Not YAGNI — the retention is deliberate and documented. Attack REFUTED.

- **A8 — Waiver discharge completeness.** 1.2 waiver (`mage test-pkg ./internal/app` + `mage ci`) discharged by 1.5 (`mage ci`) + 1.11 (`mage test-pkg ./internal/app`). 1.4 waiver (`mage build` + `mage ci`) discharged by 1.5. 1.6 waiver (`mage build` + `mage ci`) discharged by 1.11 (app), 1.12 (sqlite, mcpapi, httpapi, common), 1.13 (tui, cmd/till). Each waiver has an explicit re-green gate in a named later unit's Acceptance. Attack REFUTED — except that F1 + F2 above show 1.5 cannot actually reach its `mage ci` re-green as-written.

- **A9 — Compile-window holes on 1.8 / 1.9 / 1.10 (domain-package subsequent units).** 1.8 (rename task.go → action_item.go) blocked_by 1.1, 1.4, 1.6 ensures domain package is compile-stable post-1.6 waiver-discharge via the test units. Wait — 1.8 blocked_by includes 1.6 but not 1.11/1.12/1.13 which actually discharge 1.6's waiver. `mage test-pkg ./internal/domain` in 1.8 still works because domain package itself is not waived — only `mage build`/`mage ci` (workspace-wide) was waived on 1.6. Domain package compiles in isolation. Attack REFUTED.

- **A10 — `embedding_jobs_test.go` / `embedding_lifecycle_adapter_test.go` / `handoff_test.go` test-site references.** These files are in 1.12's Paths (line 228). No omissions detected. Attack REFUTED.

## Hylla Feedback

N/A — task touched non-Go files only (plan MD review against committed Go code via Read / Grep / LSP / Glob — Hylla's Go-code graph wasn't queried because the scope is plan-document correctness against the current working tree, not committed-code structure understanding).
