# Plan QA Proof — Round 6

**Verdict:** pass

## Summary

Every Round-5 triage addition verifies against HEAD. Blocker chains, waiver discharge graph, SQL phase ordering, and acceptance criteria are coherent. Two editorial-grade observations noted but neither breaks plan correctness.

## Findings

### P1 — `till.ensure_builtin_template_library` is not a separate MCP tool

**Severity:** editorial
**Unit:** 1.5 (acceptance), and Scope bullets §4 + §line-5/23/135/296 Paths prose
**Claim in plan:** Plan repeatedly cites `till.ensure_builtin_template_library` at `extended_tools.go:2085` as one of four dying MCP tools, and the 1.5 acceptance alternation rg pattern includes `till\.ensure_builtin_template_library` as if it were a registered tool name.
**Evidence:** `extended_tools.go:2015-2028` registers ONE tool `till.template` with `operation` enum including `"ensure_builtin"` (and `"upsert"`, `"list"`, `"get"`, `"get_builtin_status"`, `"get_node_contract"`). The string at `:2085` is the internal audit label `"ensure_builtin_template_library"` passed to `authorizeMCPMutation`, not an MCP tool name. A sibling `grep` for `till\.ensure_builtin_template_library` across the codebase returns 0 matches today — the rg alternation is already vacuous pre-deletion. The three other tools (`till.bind_project_template_library` :2171, `till.get_template_library` :2258, `till.upsert_template_library` :2281) ARE legacy-alias separate tool registrations that the 1.5 rg will correctly verify.
**Gap:** The acceptance rg pattern is not wrong (correctly returns 0 matches at unit completion), but it includes a term that never had a match, creating a misleading "four tools dying" story. The actual deletion surface is: (1) three legacy-alias tools (bind/get/upsert), (2) the `till.template` tool's `ensure_builtin`/`upsert`/`get_builtin_status`/`get_node_contract` operations (effectively the whole `till.template` tool dies), and (3) the audit-label string.
**Suggested fix:** Next-round planner edit can either (a) drop the `|till\.ensure_builtin_template_library` term from the 1.5 rg, (b) add a separate acceptance line asserting `rg 'till\.template' internal/adapters/server/mcpapi/extended_tools.go` returns 0 to cover the till.template tool's removal, or (c) leave as-is and accept the wording drift. No impact on builder correctness — `mage ci` remains the load-bearing re-green gate.

### P2 — `scanProject` starts at `:3974`, not `:3970`

**Severity:** editorial
**Unit:** 1.3
**Claim in plan:** "Second project-read query at `:3970-4000` — same shape (Scan at `:3984`, `p.Kind = ...` block at `:3990-3992`)."
**Evidence:** `repo.go:3974` declares `func scanProject(s scanner)`. Line :3970 is inside a prior loop (`out = append(out, normalized)` at :3969, blank line at :3972, comment at :3974). The Scan at :3984 and `p.Kind = NormalizeKindID(...)` at :3990-3992 with default-assignment through :3993 all match.
**Gap:** The range start `:3970` precedes the scanProject symbol by 4 lines. Internal line numbers (`:3984`, `:3990-3992`) are correct.
**Suggested fix:** Next-round planner edit can change `:3970-4000` to `:3974-4003` to bracket the full `scanProject` function. No builder-behavior impact — the function is uniquely named and builder will target-match on symbol name.

## Confirmed Claims

Round-5 triage claims verified end-to-end:

- Unit 1.2 F1 Option A — `mage test-pkg ./internal/app` + `mage ci` waiver mirrors 1.4 / 1.6 waiver pattern. Verified: PLAN line 83 contains waiver prose with discharge to unit 1.5 ("the re-green gate that discharges this waiver"). `rg` acceptance at line 82 correctly excludes `internal/app/template_library*.go`, `internal/app/template_contract*.go`, `internal/app/template_reapply.go` per the "intentionally skip" clause at line 85.
- Unit 1.3 P2 + P4 heading rename — heading at line 87 now reads "Bake kind_catalog rows + delete SQLite seeder + strip projects.kind schema (DDL + SQL queries + Go wrappers)". Verified.
- Unit 1.3 CreateProject site — `repo.go:1345-1360`. Verified: line 1345 is `// CreateProject creates project.`, :1351-1354 hold `kindID := domain.NormalizeKindID(p.Kind)` + fallback, :1356 INSERT column list includes `kind`, :1358 passes `string(kindID)` positionally.
- Unit 1.3 UpdateProject site — `repo.go:1362-1383`. Verified: line 1362 is `// UpdateProject updates state for the requested operation.`, :1368-1371 hold `kindID := domain.NormalizeKindID(p.Kind)` + fallback, :1374 SET clause includes `kind = ?`, :1376 positional arg.
- Unit 1.3 ensureGlobalAuthProject site — `repo.go:1455-1473`. Verified: line 1455 is `// ensureGlobalAuthProject creates the hidden project row...`, :1458 INSERT column list includes `kind`, :1465 positional arg is `string(domain.DefaultProjectKind)`. Function-retention note at plan line 109 correctly preserves the self-healing-bootstrap function.
- Unit 1.3 list-projects query — `repo.go:1418-1452`. Verified: :1428 declares `kindRaw string`, :1434 `rows.Scan(..., &kindRaw, ...)`, :1437-1440 hold `p.Kind = domain.NormalizeKindID(domain.KindID(kindRaw))` + `if p.Kind == "" { p.Kind = domain.DefaultProjectKind }`.
- Unit 1.3 second project-read query — `scanProject` at `:3974` (plan cites `:3970` — see P2 finding). Inner citations :3984 (Scan) and :3990-3992 (Kind assignment block) verify correctly.
- Unit 1.3 test-site strip — `repo_test.go:2369-2371` holds `project.SetKind("project-template", now)` + `t.Fatalf`; `:2378-2381` holds `loadedProject.Kind != domain.KindID("project-template")` + `t.Fatalf`. Verified verbatim.
- Unit 1.3 new rg acceptance lines — plan lines 97 (`kindRaw|NormalizeKindID\(p\.Kind\)|p\.Kind\s*=`) and 98 (`INSERT INTO projects\([^)]*kind|UPDATE projects[^;]*kind\s*=|SELECT[^;]*kind[^;]*FROM projects` with `-U`) correctly target the Go-wrapper and SQL-query strips.
- Unit 1.6 P1 snapshot.go additions — all four citations verify:
  - `:41` — `Kind domain.KindID` field declaration on `SnapshotProject` ✓
  - `:395-397` — normalization loop `if domain.NormalizeKindID(p.Kind) == "" { p.Kind = domain.DefaultProjectKind; s.Projects[i].Kind = p.Kind }` ✓
  - `:1230-1237` — `snapshotProjectFromDomain` returns `SnapshotProject{..., Kind: p.Kind, ...}` with Kind at :1235 ✓
  - `:1589-1603` — `toDomain` has `kind := domain.NormalizeKindID(p.Kind)` at :1589, fallback at :1590-1592, `Kind: kind` at :1598 ✓
- Unit 1.6 waiver discharge — line 154 lists units 1.11 (app), 1.12 (sqlite + mcpapi + httpapi + common), 1.13 (tui + cmd/till) as per-package re-green gates. 1.6 waiver discharge through 1.11/1.12/1.13 maps cleanly.
- Unit 1.6 Project struct verification — `domain/project.go:16` holds `Kind KindID`, `:60` holds `Kind: DefaultProjectKind,` in NewProject literal, `:79-88` holds `SetKind(kind KindID, now time.Time) error` with assignment at :85. All citations verify.
- Unit 1.11 compile-restoration note — plan line 218 explicitly names `Project.Kind + SnapshotProject.Kind + SetKind` as test-site strip scope; discharges 1.2 waiver transitively; line 220 rg widens to `SnapshotProject\{[^}]*Kind|SetKind`. Verified.
- Unit 1.12 test-site migration — plan line 228 notes `:2368-2381` migrated to 1.3; line 232 notes residual test-site responsibility for remaining `Project.Kind`/`SetKind` in the sqlite package.
- Unit 1.14 SQL phase ordering — plan line 268 reads: pre-flight counts → DROP template cluster (9 tables) → DELETE kind_catalog rows → SQLite table-rebuild to drop projects.kind → DROP tasks → UPDATE action_items → assertions. Phase order is correct; DROP template tables runs BEFORE the kind_catalog row DELETE per Round-5 editorial fix to avoid `ON DELETE RESTRICT` FK hazard. The 9 template tables in the DROP phase match the F9 list in PLAN Scope §6.
- Unit 1.14 new O1 assertions — plan lines 261-262 assert `node_contract_%` → 0 (covers `node_contract_snapshots`, `node_contract_editor_kinds`, `node_contract_completer_kinds`) and `project_template_bindings` exact-name → 0 (neither `template_%` nor `node_contract_%` catches this exact name).
- Unit 1.14 O2 SQL 3-valued logic fix — plan line 265 asserts `kind NOT IN ('project','actionItem') OR kind IS NULL` → 0. Correctly closes the `NOT IN (list)` + NULL semantic gap.
- Blocker chains:
  - 1.7 `blocked_by: 1.1, 1.2, 1.3, 1.5` — correct. 1.3 owns sqlite schema changes including `ALTER TABLE projects ADD COLUMN kind` strip; 1.7 owns tasks-table excision in the same file. Explicit 1.3→1.7 edge required and present.
  - 1.12 `blocked_by: 1.3, 1.5, 1.6, 1.7` — correct. Both 1.3 and 1.7 mutate `repo_test.go`; 1.6 deletes `Project.Kind` field that 1.12 test-site strip discharges.
  - 1.14 `blocked_by: 1.1…1.13` — correct; SQL rewrite is a sink blocking on every code unit.
  - No missing 1.6→1.7 edge: 1.7 does not consume `Project.Kind`; `bridgeLegacyActionItemsToWorkItems` (line 1184-1228) and `migratePhaseScopeContract` (line 710-789) are tasks-table shims that don't touch project.kind.
- Waiver discharge graph is complete:
  - 1.2 waiver (app `test-pkg` + `mage ci`) → discharged by 1.5's `mage ci` re-green gate (explicit) AND 1.11's `mage test-pkg ./internal/app` at line 218 (transitive).
  - 1.4 waiver (`mage build` + `mage ci`) → discharged by 1.5's `mage ci`.
  - 1.6 waiver (`mage build` + `mage ci`) → discharged by 1.11 (app), 1.12 (sqlite + mcpapi + httpapi + common), 1.13 (tui + cmd/till).
- Hypothetical caller reachability:
  - `ensureKindCatalogBootstrapped` — 11 call sites across `kind_capability.go` (6), `template_library.go` (1), `template_library_builtin.go` (2), `service.go` (2). Template-family callers die with 1.5 (wholesale file deletion); non-template callers are reachable from 1.2's declared work (`resolveProjectKindDefinition + callers`) and 1.5 Paths ("strip template service fields + bindings" on service.go).
  - `kindAppliesToEqual` — 2 callers in `repo.go` (:765 `upsertKindDefinition`, :1280 `seedDefaultKindCatalog`). 1.3 deletes the seeder; the :765 caller is a survivor (1.3 acceptance correctly allows "the helpers' remaining uses outside the deleted seeder").
  - `resolveProjectKindDefinition` — 2 call sites (`kind_capability.go:624`, `service.go:260`), both covered by 1.6 Paths ("strip resolveProjectKindDefinition + callers" and "strip project.Kind references").

## Hylla Feedback

N/A — Hylla not used for this review. All evidence was gathered via Read and Grep because (a) the Drop 1.75 worktree has not been reingested since kind collapse started and would return stale results, and (b) every verification required fine-grained line-number resolution inside specific Go files, which the Read tool supports directly and efficiently without a Hylla round-trip. No Hylla misses to report.
