# Plan QA — Proof Review (Round 5)

**Plan commit reviewed:** `d256f2e` ("docs(drop-1-75): plan round 4 triage fixes")
**Verdict:** `FAIL`

## Summary

The 5 Round-4 triage fixes (F1, P1, P2, O1, O2) are all applied correctly in `d256f2e` and are technically sound. However, Round 5's evidence-completeness pass uncovered **two blocking gaps in unit 1.6's Paths coverage** for the `projects.kind` column strip. Production code in `internal/app/snapshot.go` and `internal/adapters/storage/sqlite/repo.go` holds `domain.Project.Kind` readbacks/writes that are not listed in any unit's Paths and will fail to compile when unit 1.6 deletes `Project.Kind`. These are not mere oversights — they are production-code consumers outside the unit 1.6 workspace-compile-restoration carveout (which names only `sqlite` + `mcpapi` + `tui` + `cmd/till` **test** packages via units 1.12/1.13). Without explicit coverage, unit 1.12 will face more work than its description states, or the workspace will remain compile-broken through drop close.

Round-4 triage fixes F1, P1, P2, O1, O2 verified — see §Triage Verification below.

## Triage Verification

- **F1 (Option B — 1.6 waiver + 1.12/1.13 burden):** `d256f2e` adds the `**mage build` and `mage ci` are waived for this unit only` paragraph to unit 1.6 at PLAN.md line 143, mirroring the existing 1.4 waiver. Units 1.12 and 1.13 acceptance clauses strengthened to name the workspace-compile-restoration burden explicitly (PLAN.md :221, :222, :234, :235). `rg` guards extended to include `project\.Kind|SetKind` at :225 and :237. ✓ Applied.
- **P1 (`:1045-1050` → `:1046-1054`):** 3 sites corrected in `d256f2e`:
  - Top-of-file Paths header at PLAN.md :5 — now reads `pickTemplateLibraryService at :1046-1054`. ✓
  - Unit 1.5 Paths bullet at PLAN.md :124 — now reads `delete pickTemplateLibraryService at :1046-1054`. ✓
  - Notes bullet at PLAN.md :283 (was :1045-1050) — now reads `:1046-1054`. ✓
  - Direct source verification: `Read` of `internal/adapters/server/mcpapi/handler.go` confirms `pickTemplateLibraryService` function body spans lines 1046-1054. ✓
- **P2 (intentional dead-code removal marker):** PLAN.md :145 adds "The `Project.Kind` field + `NewProject` kind arg + `SetKind` method deletion is **intentional dead-code removal — no behavior change**" followed by an enumeration of downstream test-site consumers. ✓ Applied.
- **O1 (`-U` multi-line flag):** PLAN.md :141 now reads `rg -U 'project\.Kind|projects\.kind|Project\{[^}]*Kind' ...` with explanatory suffix "`-U` enables multi-line matching so `Project{...\n...Kind:...}` struct literals are caught". Verified functional — invoking the multi-line grep locally returns 4 files (project.go, snapshot.go, extended_tools_test.go, project_cli_test.go). ✓ Applied.
- **O2 (1.4 → 1.5 cross-link):** PLAN.md :117 now reads "Per-unit `mage build` gate is deferred to 1.5 (see unit 1.5 Acceptance — `"mage ci` succeeds from `drop/1.75/`" is the re-green gate that discharges this waiver)". ✓ Applied.

All 5 triage fixes are visible in `git show d256f2e`, present at the expected PLAN.md line offsets, and source-citation-accurate.

## Findings

### P1 — `internal/app/snapshot.go` production-code `Project.Kind` consumers missing from unit 1.6 Paths

- **Severity:** `blocking`
- **Unit:** 1.6 (Drop `projects.kind` column)
- **What's unproven:** Unit 1.6 Paths for the `internal/app` package lists only `kind_capability.go`, `service.go`, and `template_reapply.go`. `internal/app/snapshot.go` is NOT listed. But `snapshot.go` contains production-code consumers of `domain.Project.Kind` that will fail to compile when unit 1.6 deletes the field:
  - `snapshot.go:395-397` — `if domain.NormalizeKindID(p.Kind) == "" { p.Kind = domain.DefaultProjectKind; s.Projects[i].Kind = p.Kind }`. Note: here `p` is `SnapshotProject` (which has its own `Kind domain.KindID` field at `snapshot.go:41`), so this site may actually be orphan-safe once SnapshotProject.Kind is stripped, but SnapshotProject.Kind strip is ALSO not named in any unit.
  - `snapshot.go:1235` — `SnapshotProject{...Kind: p.Kind, ...}` struct literal (`p` is `domain.Project`).
  - `snapshot.go:1589` — `kind := domain.NormalizeKindID(p.Kind)` (`p` is `SnapshotProject`, converting to domain).
  - `snapshot.go:1593-1603` — `return domain.Project{...Kind: kind, ...}`. **This is the killer**: it's a `domain.Project` construction with `Kind:` field assignment. Unit 1.6 deletes `Project.Kind`, so this line fails to compile.
- **Evidence checked:** `Read internal/app/snapshot.go:1230-1250, :380-400, :1585-1605`, `Grep 'p\.Kind|project\.Kind|\.SetKind\(|Project\.Kind' internal/app/*.go`, PLAN.md :137 (unit 1.6 Paths).
- **Fix recommendation:** Add `internal/app/snapshot.go` to unit 1.6 Paths (and to unit 1.5 Paths if the `SnapshotProject.Kind` field also needs to disappear — scope §5 says "audit every `Project{...}` struct literal construction sitewide" so snapshot.go is in-scope by scope text but missing from unit Paths). Alternatively, split `snapshot.go`'s `SnapshotProject` work into unit 1.5 (since snapshot.go already is in 1.5 Paths for `TemplateLibraries` strip) while keeping the `domain.Project` construction strip in 1.6. Explicitly name `SnapshotProject.Kind` field deletion as in-scope.

### P2 — `internal/adapters/storage/sqlite/repo.go` production-code `Project.Kind` consumers straddle units 1.3 and 1.6 without explicit coverage

- **Severity:** `blocking`
- **Unit:** 1.3 + 1.6 boundary gap
- **What's unproven:** Unit 1.3's Paths name `repo.go` for schema-site strips (`seedDefaultKindCatalog`, `CREATE TABLE kind_catalog`, `ALTER TABLE projects ADD COLUMN kind`, `CREATE TABLE IF NOT EXISTS projects` block). Unit 1.6's Paths name several Go packages but NOT `internal/adapters/storage/sqlite/repo.go`. `repo.go` contains at least two production-code `domain.Project.Kind` readback sites that will fail to compile when unit 1.6 deletes `Project.Kind`:
  - `repo.go:1424-1450` — the `readProjects`-style loop that scans `kindRaw` from the SQL `kind` column and assigns `p.Kind = domain.NormalizeKindID(...)` at `:1437`, checks `p.Kind == ""` at `:1438`.
  - `repo.go:3986-3993` — another `domain.Project{}` construction site with `p.Kind = domain.NormalizeKindID(...)` at `:3990-3993`.
  - Additionally, `repo.go:1458` `INSERT INTO projects(id, slug, name, description, kind, ...)` references the `kind` column. This is a quoted DDL string, not a Go-type ref — but it must also die when the column does.
- **Evidence checked:** `Read internal/adapters/storage/sqlite/repo.go:1420-1460, :3985-4000`, PLAN.md :90-91 (unit 1.3 Paths), PLAN.md :137-138 (unit 1.6 Paths + Packages). Unit 1.6 Packages lists `internal/adapters/server/mcpapi` but NOT `internal/adapters/storage/sqlite`.
- **Fix recommendation:** Either (a) add `internal/adapters/storage/sqlite/repo.go` to unit 1.6 Paths AND add `internal/adapters/storage/sqlite` to unit 1.6 Packages, explicitly naming the `readProjects`/`getProject` scan-and-assign sites and the `INSERT INTO projects` column list. Or (b) expand unit 1.3's Paths explicitly to include the SELECT/INSERT statements on the `projects` table (currently 1.3 covers DDL but not DML), and shift the field-strip burden for `repo.go`-side consumers into 1.3. Either way, the scope §5 "strip `Kind KindID` field from `type Project`" work has to pair with stripping every consumer, and unit 1.6's current Paths leave `repo.go` unclaimed.

### P3 — `SnapshotProject.Kind` field deletion not explicitly named

- **Severity:** `non-blocking` (inferrable from scope §5 and unit 1.5's snapshot.go Path, but not explicitly enumerated)
- **Unit:** 1.5 (Paths already touches snapshot.go) or 1.6
- **What's unproven:** `internal/app/snapshot.go:41` defines `Kind domain.KindID` as a field on `SnapshotProject`. If `domain.Project.Kind` dies, `SnapshotProject.Kind` should die too (otherwise the snapshot serialization layer keeps a phantom field). No unit explicitly names this field deletion in its Acceptance regex. Unit 1.5's Paths for `snapshot.go` only says "strip `TemplateLibraries` field + `snapshotTemplateLibraryFromDomain` + `upsertTemplateLibrary` + `normalizeSnapshotTemplateLibrary` sections".
- **Evidence checked:** `Grep 'type SnapshotProject' internal/app/snapshot.go:36-46`, PLAN.md :124 (unit 1.5 Paths for snapshot.go).
- **Fix recommendation:** Add explicit Acceptance bullet to unit 1.6 (or 1.5): `rg 'SnapshotProject\{[^}]*Kind|SnapshotProject\.\s*Kind' returns 0 matches`. Confirm snapshot JSON schema change is intentional (the `kind,omitempty` JSON tag at :41 means the serialized schema stays backward-compatible on null/missing, so no migration concern).

### P4 — `ensureGlobalAuthProject` INSERT DML references `kind` column not named in unit 1.3's DDL-focused Paths

- **Severity:** `non-blocking` (will be discovered by unit 1.3's builder when `mage test-pkg ./internal/adapters/storage/sqlite` fails, but better to name explicitly)
- **Unit:** 1.3
- **What's unproven:** `repo.go:1458` contains `INSERT INTO projects(id, slug, name, description, kind, metadata_json, created_at, updated_at, archived_at)` — a DML that references the `kind` column. When unit 1.3 strips the column from the schema, this INSERT will break. Unit 1.3's Paths and Acceptance focus on DDL sites (`CREATE TABLE`, `ALTER TABLE`, `seedDefaultKindCatalog`) and do not name DML sites.
- **Evidence checked:** `Read internal/adapters/storage/sqlite/repo.go:1455-1473`, PLAN.md :88-101 (unit 1.3 body).
- **Fix recommendation:** Add to unit 1.3 Paths prose: "also strips the `kind` column reference from `ensureGlobalAuthProject` INSERT at :1458 and any other DML referencing `projects.kind`". Strengthen unit 1.3 Acceptance with: `rg "INSERT INTO projects\([^)]*\bkind\b" drop/1.75/internal/adapters/storage/sqlite/` returns 0 matches.

## Non-Blocking Observations

- **O1 (informational) — `p.Kind` in snapshot.go :203, :289:** These sites use `kind` as a loop variable on `[]SnapshotKindDefinition` (unrelated to `Project.Kind`). No action needed.
- **O2 (informational) — `template_library_test.go` cites `domain.WorkKind("project-setup-phase")`:** Unit 1.5 deletes `template_library_test.go` entirely (Paths line 124), so these references die with the file. ✓ Covered.
- **O3 (informational) — kind_capability.go :777, :780 `project.Kind` refs:** Covered by unit 1.6 Paths (`kind_capability.go` is explicitly named). ✓.
- **O4 (informational) — service.go :257, :260, :264, :365, :372, :378 `project.Kind`/`project.SetKind`:** Covered by unit 1.6 Paths (`service.go` is explicitly named). ✓.
- **O5 (informational) — template_reapply.go :63 `project.Kind`:** Covered by unit 1.6 Paths (`template_reapply.go` is explicitly named) AND unit 1.5 (dies entirely as `internal/app/template_reapply.go` is in 1.5 Paths). Double-covered, which is fine given blocker 1.6 is `blocked_by 1.5` — by the time 1.6 runs, the file no longer exists.

## Scope-to-Unit Coverage

All 8 scope items (§1 through §8) map to at least one unit:

- Scope §1 (Kind catalog collapse) → units 1.2, 1.3, 1.14 ✓
- Scope §2 (Go identifier rename) → unit 1.1 ✓
- Scope §3 (File + type renames) → units 1.8, 1.9 ✓
- Scope §4 (template_libraries excision) → units 1.4, 1.5 ✓
- Scope §5 (projects.kind column) → units 1.3, 1.6 (partial — see P1, P2) ⚠
- Scope §6 (drops-rewrite.sql rewrite) → unit 1.14 ✓
- Scope §7 (Legacy tasks table) → unit 1.7 ✓
- Scope §8 (Tests + fixtures) → units 1.10, 1.11, 1.12, 1.13 ✓

Scope §5 is flagged ⚠ rather than ✓ because of P1 and P2 above.

## Blocker Chain Validation

Walked PLAN.md blocker chain:

- 1.1 (no blockers) — only unblocked root.
- 1.2 ← 1.1 ✓
- 1.3 ← 1.1, 1.2 ✓
- 1.4 ← 1.1 ✓
- 1.5 ← 1.1, 1.2, 1.3, 1.4 ✓ (big atomic excision, correctly serialized)
- 1.6 ← 1.1-1.5 ✓
- 1.7 ← 1.1, 1.2, 1.3, 1.5 ✓ (note: 1.4 not a blocker, which is correct — template-domain excision and tasks table are orthogonal)
- 1.8 ← 1.1, 1.4, 1.6 ✓ (domain file rename after all domain-package edits are final)
- 1.9 ← 1.1, 1.4, 1.6, 1.8 ✓
- 1.10 ← 1.1, 1.4, 1.6, 1.8, 1.9 ✓ (domain tests trail domain code)
- 1.11 ← 1.2, 1.5, 1.6 ✓ (app tests trail app code)
- 1.12 ← 1.3, 1.5, 1.6, 1.7 ✓
- 1.13 ← 1.5, 1.6 ✓
- 1.14 ← 1.1-1.13 ✓ (SQL script is sink)
- 1.15 ← 1.14 ✓ (drop-end verification)

Chain is acyclic and complete. Package-level serialization inside `internal/domain` (units 1.1 → 1.4 → 1.6 → 1.8 → 1.9) honors CLAUDE.md §Blocker Semantics. `internal/app` package serialization (1.2 → 1.5) honors the same rule. No missing sibling blockers detected for package-sharing units.

## Hylla Feedback

N/A — Hylla is a Go-only index today and Drop 1.75's review here centered on markdown PLAN reasoning and cross-checking the live Go workspace via `Read` / `Grep`. No Hylla queries needed or attempted; no fallback path to record.
