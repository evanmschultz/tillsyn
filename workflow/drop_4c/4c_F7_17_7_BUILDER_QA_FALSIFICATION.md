# Drop 4c — F.7.17.7 Builder QA Falsification (Round 1)

**Droplet:** `4c.F.7.17.7` — `permission_grants` SQLite table + storage adapter
**Reviewer mode:** read-only adversarial.
**Verdict:** **PASS-WITH-NITS** (no CONFIRMED counterexamples; three NITs worth surfacing for the Round 2 builder if the orchestrator wants to fold them in).

## Section 0 — orchestrator-facing reasoning

(Lives only in the chat reply — Section 0 doesn't get embedded in artifact MD per CLAUDE.md doctrine.)

## 1. Files reviewed

- `internal/domain/permission_grant.go` — new struct + constructor + validation.
- `internal/domain/permission_grant_test.go` — 13-case validation matrix + UTC normalization + happy path.
- `internal/domain/errors.go` — three new sentinels appended.
- `internal/app/permission_grants_store.go` — port interface (3 methods).
- `internal/adapters/storage/sqlite/permission_grants_repo.go` — adapter.
- `internal/adapters/storage/sqlite/permission_grants_repo_test.go` — 7 test funcs + `mustCreateProject` helper.
- `internal/adapters/storage/sqlite/repo.go` — DDL appended at lines 451-468 (table) + line 491 (index).
- `workflow/drop_4c/4c_F7_17_7_BUILDER_WORKLOG.md` — builder's narrative.
- Cross-checked against `workflow/drop_4c/F7_17_CLI_ADAPTER_PLAN.md` body + REVISIONS, `workflow/drop_4c/F7_CORE_PLAN.md` F.7.5 schema spec.

## 2. Per-attack verdicts

### A1. Deviation justified? — **REFUTED**

Spawn prompt called for `BLOB PRIMARY KEY` + `uuid.UUID`; builder shipped `TEXT PRIMARY KEY` + `string`.

Evidence — every existing table primary key in `internal/adapters/storage/sqlite/repo.go` uses `id TEXT PRIMARY KEY`. `rg "id (TEXT|BLOB) PRIMARY KEY" repo.go` returns 12 hits, ALL TEXT, ZERO BLOB:

- lines 148, 165, 176, 217, 292, 324, 350, 367, 390, 424, 459 (this droplet's own), 647.

Plus the cross-plan F.7-CORE F.7.5 acceptance criteria itself specifies `id TEXT PRIMARY KEY` (`F7_CORE_PLAN.md:405`). The spawn prompt's BLOB/uuid.UUID phrasing was pre-existing-codebase-blind; the builder's deviation aligns with both the codebase and the F.7-CORE F.7.5 schema. Deviation is documented in the worklog "Schema choice rationale" section so it's auditable.

### A2. `ON CONFLICT DO NOTHING` semantics — **REFUTED**

Adapter (`permission_grants_repo.go:42`):
```sql
INSERT INTO permission_grants (...) VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(project_id, kind, rule, cli_kind) DO NOTHING
```

`modernc.org/sqlite` is a pure-Go SQLite re-implementation of the upstream SQLite engine; UPSERT clause `ON CONFLICT(<conflict-target>) DO NOTHING` has been supported by SQLite since 3.24.0 (2018). The conflict target matches the `UNIQUE (project_id, kind, rule, cli_kind)` constraint declared at `repo.go:466`, so the conflict is recognized and the second insert returns nil with zero rows affected — no error.

`TestRepositoryPermissionGrantsIdempotentInsert` exercises this end-to-end (lines 199-220 of `permission_grants_repo_test.go`): insert original → insert duplicate (different ID, granted_at, granted_by; same project+kind+rule+cli_kind tuple) → assert `err == nil`. The test would fail if SQLite returned an error.

### A3. Idempotent insert preserves original row — **REFUTED**

`TestRepositoryPermissionGrantsIdempotentInsert` lines 222-237:
- After both inserts, `ListGrantsForKind` returns exactly 1 row.
- Asserts `got[0].ID == original.ID` (original survives).
- Asserts `got[0].GrantedBy == original.GrantedBy` ("STEWARD", not "OTHER_PRINCIPAL").
- Asserts `got[0].GrantedAt.Equal(original.GrantedAt)` (NOT `now.Add(time.Hour)`).

The dup grant intentionally uses a different ID, GrantedBy, AND GrantedAt to make the assertion load-bearing. PASS.

### A4. FK cascade-delete — **REFUTED**

`PRAGMA foreign_keys = ON` is set at the connection level — `repo.go:133` (in the `Open` path's pragma list) AND `repo.go:146` (in the explicit pragma block). `repo_test.go:37-41` and `:71-75` already assert the pragma is on for both `Open()` and `OpenInMemory()` connections.

Schema (`repo.go:467`): `FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE`.

Combined: project deletion will cascade to permission_grants. There is no test exercising the cascade in the new test file, but the FK pragma is verified at repo-init level by the existing `repo_test.go` tests.

NIT — see §3.

### A5. Lookup index correctness — **REFUTED**

Index: `idx_permission_grants_lookup ON permission_grants (project_id, kind, cli_kind)` (`repo.go:491`).

Query (`permission_grants_repo.go:71-72`):
```sql
WHERE project_id = ? AND kind = ? AND cli_kind = ?
ORDER BY granted_at ASC, id ASC
```

The three equality predicates align as a perfect index-prefix match, so the SQLite query planner uses the index for filtering. ORDER BY is on `granted_at, id` which is NOT in the index, so the planner sorts the post-filter rowset in memory — acceptable for permission_grants cardinality (low: per (project, kind, cli_kind) the row count is bounded by approved rules, not transactional volume).

No table scan. No NIT.

### A6. `NewPermissionGrant` validation strictness — **REFUTED**

`permission_grant.go:71-98`:
- ID, ProjectID, Rule, GrantedBy: `strings.TrimSpace` then non-empty check.
- CLIKind: `strings.TrimSpace(strings.ToLower(...))` then non-empty.
- Kind: `IsValidKind(in.Kind)` — closed-enum membership check (`kind.go:50-52` does `slices.Contains(validKinds, ...)` with TrimSpace+ToLower).

`TestNewPermissionGrantValidationRejections` covers all 13 cases:
- empty ID, whitespace ID
- empty ProjectID, whitespace ProjectID
- empty Kind, unknown Kind ("not-a-real-kind")
- empty Rule, whitespace Rule
- empty CLIKind, whitespace CLIKind
- empty GrantedBy, whitespace GrantedBy

Whitespace-only inputs are correctly rejected because TrimSpace runs before the non-empty check. Unknown Kind is rejected because `IsValidKind` is membership-against-closed-12-enum, not just `len > 0`.

### A7. `cli_kind` validation at this layer — **REFUTED (with explicit narrow-scope justification)**

The domain layer accepts ANY non-empty CLIKind string after trim+lower. Rule rationale (worklog "Schema choice rationale" + `permission_grant.go:34-41` doc comment): closed-enum CLI vocabulary lives at the templates / dispatcher boundary (per F7_17 plan REV-3 + `IsValidCLIKind` on `CLIKind` type per F.7.17.1 acceptance criteria). The domain just refuses "" so the storage adapter's UNIQUE composite stays stable.

This means `cli_kind = "bogus"` would persist if a direct caller bypassed the templates layer. The test asserts `cli_kind` is lowercased to `"claude"` from input `"Claude"` (`permission_grant_test.go:37-39`), but doesn't reject `"bogus"`.

NIT — see §3.

### A8. `granted_at` precision — **REFUTED**

Builder uses the shared `ts(t time.Time) string` helper at `repo.go:3505`:
```go
func ts(t time.Time) string {
    return t.UTC().Format(time.RFC3339Nano)
}
```

This is the SAME format every other table in the repo uses (`comments.created_at`, `handoffs.updated_at`, `attention_items.created_at`, etc. — all RFC3339Nano text). `parseTS` (line 3518) parses the same format. UTC enforced both on write (`ts()` uses `t.UTC()`) and on the domain constructor (`NewPermissionGrant` does `now.UTC()`).

The test `TestNewPermissionGrantUTCNormalization` (lines 144-168) verifies non-UTC `now` (America/New_York) is converted to UTC on the persisted record. Cross-table format consistency is preserved.

### A9. Cross-project isolation test rigor — **REFUTED**

`TestRepositoryPermissionGrantsCrossProjectIsolation` (lines 242-288):
- Creates project A AND project B as DIFFERENT real projects via `mustCreateProject` (helper at line 489).
- Inserts ONE grant under project A.
- Asserts `ListGrantsForKind(A, build, claude)` returns `[grantA]`.
- Asserts `ListGrantsForKind(B, build, claude)` returns empty (`len(gotB) == 0`).

Both projects exist as real DB rows so the FK on `permission_grants.project_id` doesn't reject the insert. The assertion is precise on both sides (A returns the grant; B returns empty).

### A10. Cross-CLI isolation test rigor — **REFUTED**

`TestRepositoryPermissionGrantsCrossCLIIsolation` (lines 295-367):
- Inserts claude grant.
- Asserts ListGrantsForKind(project, build, "claude") returns 1 row.
- Asserts ListGrantsForKind(project, build, "codex") returns 0 rows — even though project + kind + rule match. (line 339-342)
- THEN inserts a codex grant with the same rule under same project + kind.
- Asserts the codex grant lands successfully (UNIQUE composite is per-CLI; codex insert does NOT collide with claude insert) — line 357-358.
- Asserts post-insert codex query returns the codex grant.

This is the load-bearing test for Drop 4d's codex landing: same rule must be grantable independently per-CLI. PASS.

### A11. Schema-init `CREATE TABLE IF NOT EXISTS` — **REFUTED**

`repo.go:458`: `CREATE TABLE IF NOT EXISTS permission_grants (...)`.
`repo.go:491`: `CREATE INDEX IF NOT EXISTS idx_permission_grants_lookup ...`.

Both idempotent. Re-running migrate against an already-migrated DB is a noop. Fresh-DB-friendly per pre-MVP rule.

### A12. Compile-time interface assertion — **NIT (not CONFIRMED)**

`embedding_lifecycle_adapter.go:16` declares `var _ app.EmbeddingLifecycleStore = (*Repository)(nil)` — precedent for compile-time interface assertions in this package.

`permission_grants_repo.go` does NOT declare `var _ app.PermissionGrantsStore = (*Repository)(nil)`. Without this, an interface drift (e.g. someone changes `app.PermissionGrantsStore.InsertGrant` to take a different signature) would NOT fail at adapter compile time — only at the next call site that uses the interface. Adding the assertion is one line; the precedent exists in the package.

This is the only refactor-safety hole in the implementation. Builder didn't claim it ships in the worklog acceptance checklist, so it isn't a regression — but it's a reuse-of-existing-pattern miss.

Severity: NIT. Doesn't break correctness, doesn't fail tests, but is a one-line guard the package's own conventions invite.

### A13. UNIQUE constraint shape — **REFUTED**

UNIQUE composite: `(project_id, kind, rule, cli_kind)`. Two grants differing only by `granted_by` or `granted_at` will collide on the UNIQUE composite. The idempotent insert test (A3) explicitly exercises this case (different granted_by, different granted_at, same composite quadruple → second insert is a noop, original survives). Per spec, this IS the intended semantic. PASS.

### A14. No-commit by builder — **REFUTED**

`git log --oneline -5` head:
```
e19e9f0 docs(drop-4c): add f.7.2 qa proof and falsification artifacts
f6aec8b feat(templates): add tool-gating + sandbox + sysprompt fields (4c F.7.2)
31700b6 feat(dispatcher): add claude adapter implementing cli adapter interface
e6cd71c feat(templates): add tillsyn top-level globals struct
16b86cb feat(templates): add context sub-struct to agent binding
```

None reference F.7.17.7. `git status --short` shows the new files as `??` (untracked) and `repo.go` / `errors.go` as ` M` (modified, unstaged). Builder did NOT commit. Worklog "Conventional commit message (for orchestrator post-QA)" line confirms intent: "for orchestrator post-QA" — orchestrator drives commits. PASS.

### A15. Memory-rule conflicts — **REFUTED**

- `feedback_no_migration_logic_pre_mvp.md` — DDL is appended in the `Init()` migrate-on-fresh-DB path; no migration code, no `ALTER TABLE`, no schema-version tracking. Worklog explicitly cites the rule and includes a fresh-DB callout for the dev. PASS.
- `feedback_subagents_short_contexts.md` — single-domain task (one table + adapter + tests); no multi-package fan-out. PASS.
- `feedback_orchestrator_no_build.md` — builder is a subagent, not the orchestrator. PASS.
- `feedback_use_typed_agents.md` — `go-builder-agent` was used. PASS (orchestrator concern, not builder).

## 3. NITs (non-blocking; surface to orchestrator if Round 2 happens)

### N1. No `var _ app.PermissionGrantsStore = (*Repository)(nil)` compile-time assertion

**Where:** `internal/adapters/storage/sqlite/permission_grants_repo.go`.
**Pattern precedent:** `embedding_lifecycle_adapter.go:16`.
**Fix:** add `var _ app.PermissionGrantsStore = (*Repository)(nil)` at the top of `permission_grants_repo.go` (file-level declaration). Catches interface drift at compile time.
**Severity:** NIT — doesn't affect correctness; just refactor safety.

### N2. No FK cascade-delete test for `permission_grants → projects`

**Where:** `internal/adapters/storage/sqlite/permission_grants_repo_test.go`.
**What's missing:** a test that creates a project, inserts a grant under it, deletes the project, and asserts the grant is GONE (FK cascade fired).
**Why it's a NIT:** the FK pragma is enforced at connection level (`repo.go:133` + `:146`) and `repo_test.go` already verifies the pragma is on. The schema declares `ON DELETE CASCADE` (`repo.go:467`). The cascade WILL fire correctly. A direct test of the cascade is belt-and-suspenders rigor — useful but not load-bearing.
**Severity:** NIT — extra assertion; current coverage is implicit-via-pragma-test.

### N3. `cli_kind` accepts arbitrary non-empty strings at the domain layer

**Where:** `internal/domain/permission_grant.go:90-93`.
**Behavior:** `cli_kind = "bogus"` would persist if a direct caller bypasses the templates layer (which holds the closed-enum check via `IsValidCLIKind` per F.7.17.1).
**Risk:** low — every production write path goes through the dispatcher, which goes through the resolved binding's `CLIKind` (validated at template-load time per F.7.17.1). A direct domain-layer caller is the failure shape, and there is one in this codebase: tests. But the lowercasing normalization mitigates the most likely shape ("Claude" vs "claude").
**Trade-off:** the worklog and `permission_grant.go` doc comment both explicitly justify this as "domain accepts free-form, templates layer holds the closed enum" — a deliberate split. If the orchestrator wants belt-and-suspenders, F.7.17.7 could call `domain.IsValidCLIKind` from `NewPermissionGrant` once F.7.17.1 has landed `IsValidCLIKind` in the domain package. Today the closed-enum lives in `internal/templates`, so calling it from `internal/domain` would create an upward dependency.
**Severity:** NIT — accepted-by-design with explicit doc comment; upgrade contingent on F.7.17.1's `IsValidCLIKind` location.

## 4. Scope check (cross-plan ownership)

**Plan body** says F.7.17.7 ADDS `cli_kind` to a pre-existing `permission_grants` table that **F.7-CORE F.7.5** owns (`F7_CORE_PLAN.md:381-410`). At the time of this droplet, F.7.5 has NOT shipped — `git status` shows no F.7.5 worklog under `workflow/drop_4c/`, no `permission_grants` table exists pre-droplet.

The builder's worklog asserts: "F.7.17.7 is the SOLE OWNER of the `permission_grants` table including the `cli_kind` column from inception." This is consistent with the orchestrator's spawn prompt phrasing ("`permission_grants` SQLite table + storage adapter") rather than the plan body's "schema gets cli_kind column."

The scope re-bundling is the orchestrator's call (it issued the spawn prompt). Two follow-on consequences worth surfacing:

- **F.7-CORE F.7.5** acceptance criteria now overlap with this droplet's shipped surface. F.7.5 will need re-scoping at sibling-coordination time to NOT re-declare the table.
- The plan REVISIONS section runs only through REV-8; the orchestrator's spawn prompt referenced "REV-13" but the plan file does not contain REV-9 through REV-13. Either (a) further revisions exist somewhere not yet committed, or (b) the spawn prompt's "REV-13" was a forward-looking marker. Either way, the implementation matches the orchestrator's stated scope.

This is not a counterexample against the implementation — it's a cross-plan coordination flag. Severity: NIT for orchestrator's awareness on F.7.5.

## 5. Verdict

**PASS-WITH-NITS.** Three NITs (N1 / N2 / N3) plus one cross-plan coordination flag (§4). None are CONFIRMED counterexamples. All 15 attacks REFUTED. Implementation is safe to commit at the orchestrator's discretion.

If the orchestrator wants to fold N1 (compile-time interface assertion) into Round 2 — that's a one-line, zero-risk add. N2 and N3 are deferrable to the F.7.5 / F.7.17.1 landing droplets where they fit more naturally.

## 6. Hylla Feedback

N/A — this droplet's review touched only Go files added or modified within the current uncommitted working set. Hylla's index is stale for these files until the next reingest. All evidence came from `Read` + `rg` + `git log` / `git status` against the live working tree, which is the correct evidence path for post-build pre-commit review per the project's "git diff for changed-since-ingest" rule.

## TL;DR

- **T1.** Reviewed 8 files (5 new, 2 modified, 1 worklog) against the plan body + REVISIONS + F.7-CORE F.7.5 spec.
- **T2.** All 15 attacks REFUTED. Three NITs surfaced (compile-time interface assertion missing; no direct FK-cascade test; `cli_kind` free-form at domain layer by deliberate design). Plus one cross-plan scope flag (§4) — F.7-CORE F.7.5 will need re-scoping since this droplet shipped the entire table, not just the `cli_kind` column.
- **T3.** **PASS-WITH-NITS** — no counterexamples, safe to commit at orchestrator's discretion. N1 is the only NIT worth a one-line Round-2 fix (`var _ app.PermissionGrantsStore = (*Repository)(nil)`).
