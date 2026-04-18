# DROP_1_75 — PLAN QA FALSIFICATION REVIEW (Round 5)

**Verdict:** FAIL — 1 blocking counterexample (F1) plus 2 material observations (O1, O2).

Round 4 caught F1 as a regression-from-triage. Round 5 catches an analogous shape in a different pass ordering: unit 1.2 deletes `ensureKindCatalogBootstrapped` without a workspace-compile waiver, leaving dangling callers in `template_library*.go` files that only die in unit 1.5. This is the same class of regression — "delete a symbol whose consumers die in a later unit without a waiver bridging the gap".

## Counterexamples

### F1 — Unit 1.2 deletes `ensureKindCatalogBootstrapped` with no compile waiver; circular block vs 1.5

**Severity:** blocking.
**Unit:** 1.2 — Delete app-layer kind-catalog seeder.

**Attack scenario.** Unit 1.2 deletes the `ensureKindCatalogBootstrapped` function (and `defaultKindDefinitionInputs`, `kindBootstrap` `sync.Once` field) from `internal/app/kind_capability.go:559-589`. Unit 1.2's own description explicitly says it will **skip** call sites inside files destined for deletion by unit 1.5 (`internal/app/template_library.go`, `template_library_builtin.go`, `template_contract.go`, `template_reapply.go`). Once 1.2 commits:

1. The function no longer exists.
2. `template_library.go:126`, `template_library_builtin.go:29`, `template_library_builtin.go:79` still call `s.ensureKindCatalogBootstrapped(ctx)`.
3. `internal/app` package does not compile.

Unit 1.2's Acceptance includes `mage test-pkg ./internal/app passes` **and** a tree-wide `rg 'ensureKindCatalogBootstrapped|defaultKindDefinitionInputs|kindBootstrap' drop/1.75/ --glob='!drops/**'` = 0 matches. Both fail:

- `mage test-pkg ./internal/app` cannot pass because the package won't compile while template_library*.go still calls the deleted function.
- `rg 'ensureKindCatalogBootstrapped' ...` returns ≥3 matches from the surviving template_library family.

Unit 1.5 is the only unit that deletes those caller files — but 1.5's `blocked_by: 1.1, 1.2, 1.3, 1.4`. Circular: 1.2 cannot pass acceptance until 1.5 runs; 1.5 cannot run until 1.2 passes acceptance.

**Evidence.**
- `internal/app/template_library.go:126` — `if err := s.ensureKindCatalogBootstrapped(ctx); err != nil {` (live, called from a non-test function in the file 1.5 deletes).
- `internal/app/template_library_builtin.go:29` and `:79` — same call.
- Plan.md unit 1.2 description: "**Intentionally skip** call sites inside files destined for deletion by unit 1.5 (`internal/app/template_library_builtin.go:29, :79`, `template_library.go`, `template_contract.go`, `template_reapply.go`) — 1.5's wholesale file deletion moots them".
- Plan.md unit 1.2 Acceptance lines 82-83: tree-wide rg returns 0 **and** `mage test-pkg ./internal/app` passes.
- Plan.md unit 1.5 Blocked by (line 126): `1.1, 1.2, 1.3, 1.4` — 1.5 cannot start until 1.2 is done.

**Regression class.** This is the same shape Round 4 caught against unit 1.6. Round-4 triage added a per-unit `mage build` + `mage ci` waiver to 1.6 (and to 1.4), with re-green discharged by a later unit (1.12 + 1.13 for 1.6; 1.5 for 1.4). Unit 1.2 has the same setup — delete symbol X, leave callers alive in files a later unit will delete — but no waiver and a stricter acceptance regex than 1.4/1.6 had.

**Fix recommendation — Option A (mirrors the 1.4 / 1.6 waiver pattern; minimal disruption):**

Add to unit 1.2:

> **`mage build` and `mage ci` are waived for this unit only.** The workspace is compile-broken between this unit's commit and 1.5's commit by design (`internal/app` package has dangling calls to the deleted `ensureKindCatalogBootstrapped` from template_library*.go until 1.5 removes those files). Per-unit `mage build` + `mage test-pkg ./internal/app` gates are deferred to 1.5 (see unit 1.5 Acceptance — `mage ci` succeeds from `drop/1.75/` is the re-green gate that discharges this waiver). Builder honors this waiver; QA does not fail the unit on workspace-compile failure.

Rewrite unit 1.2 Acceptance to scope the `rg` check:

- Change line 82 from `rg 'ensureKindCatalogBootstrapped|defaultKindDefinitionInputs|kindBootstrap' drop/1.75/ --glob='!drops/**' returns 0` to `rg 'ensureKindCatalogBootstrapped|defaultKindDefinitionInputs|kindBootstrap' drop/1.75/ --glob='!drops/**' --glob='!internal/app/template_library*.go' --glob='!internal/app/template_contract*.go' --glob='!internal/app/template_reapply.go' returns 0. The full tree-wide sweep moves to unit 1.5 Acceptance (which must add this regex).` Then add the same regex to unit 1.5 Acceptance as a post-deletion invariant.
- Remove `mage test-pkg ./internal/app passes` from unit 1.2 Acceptance; restore it on unit 1.5.

Also extend unit 1.5 Acceptance to add the 1.2-scoped regex as a full-tree invariant (since 1.5 is the re-green point):

- `rg 'ensureKindCatalogBootstrapped|defaultKindDefinitionInputs|kindBootstrap' drop/1.75/ --glob='!drops/**' returns 0 matches.`

**Fix recommendation — Option B (merge 1.2 into 1.5):**

Delete unit 1.2 outright. Move all of 1.2's scope (deleting `ensureKindCatalogBootstrapped` + `defaultKindDefinitionInputs` + `kindBootstrap` `sync.Once` + the 6 in-`internal/app` callers at kind_capability.go:99/161/593/636 + service.go:192/244) into unit 1.5's deletion bundle. Renumber downstream units (1.3→1.2, 1.4→1.3, …) or leave gap.

Less surgical, but avoids the waiver-pattern spread across three units (1.2 / 1.4 / 1.6) that all do the same "delete, wait for callers to die, re-green later" dance.

**Option A is preferred** — it mirrors the existing 1.4 / 1.6 waiver treatment and is the smallest delta to the committed d256f2e plan.

## Observations

### O1 — Unit 1.14 assertion block doesn't cover `node_contract_*` or `project_template_bindings`

**Severity:** non-blocking.
**Unit:** 1.14 — `scripts/drops-rewrite.sql` rewrite.

**Attack.** Unit 1.14 drops 9 tables (`template_libraries`, `template_node_templates`, `template_child_rules`, `template_child_rule_editor_kinds`, `template_child_rule_completer_kinds`, `project_template_bindings`, `node_contract_snapshots`, `node_contract_editor_kinds`, `node_contract_completer_kinds`). The assertion block only checks:

- `SELECT COUNT(*) FROM sqlite_master WHERE name LIKE 'template_%'` = 0.

This catches 5 of the 9 tables (the `template_*` family) but not `project_template_bindings` nor the 3 `node_contract_*` tables. A silent survival of any of those 4 would evade verification.

**Fix recommendation.** Extend the assertion block with two additional checks:

- `SELECT COUNT(*) FROM sqlite_master WHERE name LIKE 'node_contract_%'` returns 0.
- `SELECT COUNT(*) FROM sqlite_master WHERE name = 'project_template_bindings'` returns 0.

Now a minimum of 7 assertions (was 5). The plan currently says "Assertion block: ... = 2; ... LIKE 'template_%' = 0; ... = 'tasks' = 0; ... pragma_table_info kind = 0; ... NOT IN = 0" and "minimum 5 assertions" — bump both to 7.

### O2 — Unit 1.14 assertion #5 (`kind NOT IN (...)`) is NULL-blind

**Severity:** non-blocking.
**Unit:** 1.14 — `scripts/drops-rewrite.sql` rewrite.

**Attack.** Assertion `SELECT COUNT(*) FROM action_items WHERE kind NOT IN ('project','actionItem')` returns 0 relies on SQL 3-valued logic: `NULL NOT IN ('project','actionItem')` evaluates to `NULL`, not `TRUE`. A row with `kind IS NULL` evades the assertion. Current dev DB is 115 rows with uniform `kind='task'` per the plan note, so the UPDATE on line 32 covers them. But if any row slips in with a NULL kind (edge case during the window between the dev cleanup and the script run), the assertion would silently pass despite the invariant being broken.

**Fix recommendation.** Change the assertion to catch NULL too:

- `SELECT COUNT(*) FROM action_items WHERE kind NOT IN ('project','actionItem') OR kind IS NULL` returns 0.

Or equivalently add a tightening pre-update:

- Before the UPDATE in phase 6, `SELECT COUNT(*) FROM action_items WHERE kind IS NULL OR kind = ''` and fail the script if non-zero (pre-flight shape check).

This mirrors phase (1) "pre-flight counts" that the plan already mentions but doesn't specify.

## What I checked and did NOT find

- **Waiver soundness for unit 1.6 (F1 from Round 4).** Verified. Every `Project.Kind` / `SetKind` reference lives in a file covered by one of: unit 1.6 (production strip), unit 1.11 (app tests — `kind_capability_test.go`), unit 1.12 (sqlite + mcpapi + httpapi + common tests), or unit 1.13 (tui + cmd/till tests). No fourth package holds orphaned references. The one remaining question — whether `app.CreateProjectInput.Kind` / `app.UpdateProjectInput.Kind` / `common.CreateProjectRequest.Kind` / `common.UpdateProjectRequest.Kind` are stripped or left as orphans — is acceptable either way per the F5 orphan-via-collapse policy; unit 1.6's acceptance regex `Project\{[^}]*Kind` only catches literal `Project{...Kind...}` struct literals, so orphan retention passes. Flag this as an **Unknown** for the builder: decide orphan-retain vs full-strip at build time, document the choice in BUILDER_WORKLOG.md, and either way the workspace compiles.
- **Blocker graph completeness for the rest of the plan.** Walked every package-sharing pair:
  - `internal/domain`: 1.1 → 1.4 → 1.6 → 1.8 → 1.9 → 1.10. Linear, correct.
  - `internal/app`: 1.1 → 1.2 → 1.5 → 1.6 → 1.11. Correct (modulo F1 above).
  - `internal/adapters/storage/sqlite`: 1.1 → 1.3 → 1.5 → 1.7 → 1.12. Correct.
  - `internal/adapters/server/mcpapi`: 1.1 → 1.5 → 1.6 → 1.12. Correct.
  - `internal/adapters/server/common`: 1.1 → 1.5 → 1.12. Correct.
  - `internal/adapters/server/httpapi`: 1.1 → 1.5 → 1.12. Correct.
  - `internal/tui`: 1.1 → 1.5 → 1.6 → 1.13. Correct.
  - `cmd/till`: 1.1 → 1.5 → 1.6 → 1.13. Correct.
  - `scripts/drops-rewrite.sql`: 1.14 blocks on all code units (1.1–1.13). Correct.
- **Regex correctness for unit 1.6.** `rg -U 'project\.Kind|projects\.kind|Project\{[^}]*Kind'` with `-U` enables multiline; `[^}]*` in `-U` mode DOES match newlines (character classes always match newlines in ripgrep's multiline mode). The regex correctly catches `Project{\n...\nKind: ...\n}` multi-line struct literals. ✓
- **Scope §1–8 → unit mapping.** Every in-scope item maps to at least one unit. §1 kind catalog collapse → 1.2 + 1.3. §2 Go identifier rename → 1.1. §3 file + type renames → 1.8 + 1.9. §4 template_libraries excision → 1.4 + 1.5. §5 projects.kind drop → 1.3 (schema) + 1.6 (Go). §6 drops-rewrite.sql → 1.14. §7 legacy tasks table → 1.7. §8 tests + fixtures → 1.10 + 1.11 + 1.12 + 1.13. ✓
- **Hylla-ingest discipline.** Unit 1.15 correctly gates ingest behind `gh run watch --exit-status` green. Orchestrator-run per workflow. ✓
- **`mage install` contamination.** No unit references `mage install`. ✓
- **Phase ordering for unit 1.14 vs FK constraints.** Initially looked like a blocker (DELETE FROM kind_catalog before DROP template_* risks FK RESTRICT failure on `template_node_templates.node_kind_id` / `template_child_rules.child_kind_id`). Downgraded to non-blocking because: (a) SQLite defaults `PRAGMA foreign_keys = OFF`, so the restrict only fires if dev explicitly enables it; (b) post dev-DB cleanup on 2026-04-18 the template_* tables likely hold 0 rows in the live DB; (c) phase (3)'s DROP TABLE of template_* tables removes all rows holding RESTRICT-FK references before the DELETE FROM kind_catalog. The plan ORDER shown in the description is (2) DELETE kind_catalog, (3) DROP template → which is the risky order. Re-reading line 255: "Phases: (1) pre-flight counts, (2) DELETE FROM kind_catalog WHERE id NOT IN ('project', 'actionItem'), (3) DROP TABLE the template cluster (9 tables per F9)". This order IS dangerous if FK enforcement is on and template tables still have rows. **Not upgrading to blocker** because the combination of OFF-default FKs + already-empty template tables makes it extremely unlikely to fire. But would recommend the builder swap phases (2) and (3): drop template tables first (removing all rows that reference kind_catalog via FK-RESTRICT), then delete the 5 legacy rows from kind_catalog. Cost: 2 lines swapped. Benefit: eliminates the FK-RESTRICT failure mode entirely. Call it out as a strong editorial suggestion in the builder brief.

## Hylla Feedback

N/A — this review examined non-Go markdown (`PLAN.md`, `WORKFLOW.md`, `CLAUDE.md`) plus Go source + SQL schema that fall under the filesystem-md + `Grep`/`Read`/`LSP` fallback lane. The drop does not use Tillsyn or Hylla ingest during plan-QA — Hylla ingest happens only at Phase 7 Closeout post-CI-green per WORKFLOW.md. No Hylla queries ran.
