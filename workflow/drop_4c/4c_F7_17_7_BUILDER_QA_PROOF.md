# Drop 4c — F.7.17.7 Builder QA-Proof Review

**Reviewer role:** go-qa-proof-agent (subagent, read-only).
**Droplet:** `4c.F.7.17.7` — `permission_grants` SQLite table + storage adapter.
**Verdict:** **PROOF GREEN** with 2 nits (deviation acknowledged + sound; SKETCH.md tracked-edit out-of-list-but-justified).

## Acceptance Criteria — File:Line Evidence

### 1. Builder deviation is consistency-driven (TEXT IDs, not BLOB/uuid.UUID)

Read of `internal/adapters/storage/sqlite/repo.go:280-449` confirms every existing table uses `id TEXT PRIMARY KEY` and TEXT timestamps via the shared `ts()` helper:

- `comments`: `id TEXT PRIMARY KEY` + `created_at TEXT NOT NULL`, `updated_at TEXT NOT NULL` (`repo.go:292-303`).
- `kind_catalog`: `id TEXT PRIMARY KEY` + TEXT timestamps (`repo.go:323-332`).
- `project_allowed_kinds`: composite `(project_id TEXT, kind_id TEXT)` PK + TEXT `created_at` (`repo.go:342-348`).
- `capability_leases`: `instance_id TEXT PRIMARY KEY` + TEXT timestamps (`repo.go:349-365`).
- `attention_items`: `id TEXT PRIMARY KEY` + TEXT timestamps (`repo.go:366-388`).
- `auth_requests`: `id TEXT PRIMARY KEY` + TEXT timestamps + nullable TEXT `resolved_at` / `expires_at` (`repo.go:389-422`).
- `handoffs`: `id TEXT PRIMARY KEY` + TEXT timestamps (`repo.go:423-450`).

Builder's deviation (`worklog Schema choice rationale §1`) is consistent with **every existing storage table**. The codebase has zero precedent for BLOB IDs or DATETIME columns. Adopting BLOB/uuid.UUID for `permission_grants` alone would create a style island. **Deviation accepted.**

### 2. No `google/uuid` direct import added

`git diff --stat go.mod go.sum` returns clean — no edits to module files. `git status` lists no go.mod / go.sum delta. Builder's claim that promoting `google/uuid` to a direct dep was avoided is verified: `go.mod` unchanged.

Imports in the five new files (read in full):

- `permission_grant.go`: `strings`, `time` (only).
- `permission_grant_test.go`: `errors`, `testing`, `time` (only).
- `permission_grants_store.go`: `context`, `internal/domain` (only).
- `permission_grants_repo.go`: `context`, `database/sql`, `errors`, `strings`, `internal/app`, `internal/domain` (only).
- `permission_grants_repo_test.go`: `context`, `errors`, `path/filepath`, `testing`, `time`, `internal/app`, `internal/domain` (only).

Zero `github.com/google/uuid` imports in any new file. **Verified.**

### 3. Schema columns match contract

`internal/adapters/storage/sqlite/repo.go:451-468` (the `git diff` hunk):

```
CREATE TABLE IF NOT EXISTS permission_grants (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    kind TEXT NOT NULL,
    rule TEXT NOT NULL,
    cli_kind TEXT NOT NULL,
    granted_by TEXT NOT NULL,
    granted_at TEXT NOT NULL,
    UNIQUE (project_id, kind, rule, cli_kind),
    FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
);
```

Plus `CREATE INDEX IF NOT EXISTS idx_permission_grants_lookup ON permission_grants (project_id, kind, cli_kind);` at `repo.go:488` (in the trailing index block).

All seven required columns present (id, project_id, kind, rule, cli_kind, granted_by, granted_at), all NOT NULL where required, UNIQUE composite on the four-tuple, FK to projects with CASCADE. The deviation point (TEXT instead of BLOB / DATETIME) does not change semantic shape. **Verified.**

### 4. FK cascade-delete to projects

Same DDL block, `repo.go:467`:

```
FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
```

Identical to the cascade used by every other tenant-scoped table. **Verified.**

### 5. `domain.PermissionGrant` struct + `NewPermissionGrant` validation

`internal/domain/permission_grant.go:24-32` defines the seven-field struct (ID, ProjectID, Kind, Rule, CLIKind, GrantedBy, GrantedAt).

`internal/domain/permission_grant.go:70-109` — `NewPermissionGrant` rejects:

- Empty / whitespace ID (`:71-74`) → `ErrInvalidID`.
- Empty / whitespace ProjectID (`:76-79`) → `ErrInvalidID`.
- Invalid Kind via `IsValidKind` (`:81-83`) → `ErrInvalidKind`. Confirmed `IsValidKind` exists at `internal/domain/kind.go:50-52` (closed 12-value enum check with TrimSpace + ToLower).
- Empty / whitespace Rule (`:85-88`) → `ErrInvalidPermissionGrantRule`.
- Empty / whitespace CLIKind (`:90-93`) → `ErrInvalidPermissionGrantCLIKind`. CLIKind is also lowercased on the way through, locking the storage UNIQUE composite to a single canonical form.
- Empty / whitespace GrantedBy (`:95-98`) → `ErrInvalidPermissionGrantGrantedBy`.
- Stamps `GrantedAt = now.UTC()` (`:107`).

Three new sentinels added in `internal/domain/errors.go:81-95` (verified via `git diff`). All carry doc comments. **Verified.**

13 validation cases tested: `internal/domain/permission_grant_test.go:53-140` — empty + whitespace × {ID, ProjectID, Rule, CLIKind, GrantedBy} (10 cases), empty Kind, unknown Kind (2 cases), plus the happy-path (1 case in a separate test). Whitespace-only inputs covered for every required string field, including the recently-added robustness against `   `-only inputs. **Verified.**

UTC-normalization tested explicitly: `permission_grant_test.go:144-168` uses `America/New_York` and asserts `Location() == time.UTC` post-construction. **Verified.**

### 6. `app.PermissionGrantsStore` interface declares the three required methods

`internal/app/permission_grants_store.go:30-45`:

- `InsertGrant(ctx context.Context, grant domain.PermissionGrant) error` (`:33`).
- `ListGrantsForKind(ctx context.Context, projectID string, kind domain.Kind, cliKind string) ([]domain.PermissionGrant, error)` (`:40`).
- `DeleteGrant(ctx context.Context, id string) error` (`:44`).

Idempotency / determinism / ErrNotFound contracts documented in interface doc comment (`:9-29`). The interface lives in a separate file from `ports.go`, mirroring the pre-existing `HandoffRepository` optional-port pattern. **Verified.**

### 7. SQLite adapter idempotent-insert via `ON CONFLICT DO NOTHING`

`internal/adapters/storage/sqlite/permission_grants_repo.go:37-52`:

```
INSERT INTO permission_grants (
    id, project_id, kind, rule, cli_kind, granted_by, granted_at
)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(project_id, kind, rule, cli_kind) DO NOTHING
```

Conflict on the UNIQUE composite is a noop returning `nil` — no error wrapping, no surprise. Adapter additionally re-applies trim+lowercase on every field (`:23-28`) defending against direct callers that bypass `NewPermissionGrant`. Empty-after-trim values rejected with `ErrInvalidID` (`:30-32`); zero `GrantedAt` rejected (`:33-35`). **Verified.**

### 8. Seven new tests pass

Verified via `mage testPkg ./internal/adapters/storage/sqlite/` (87 tests, 0 fail; was 80 in prior commit) and `mage testPkg ./internal/domain/` (286 tests, 0 fail; was 283 in prior commit). Net delta: +7 sqlite tests + 3 domain tests = +10 actual tests (worklog claimed +7 sqlite tests; that count matches sqlite-package-only — domain tests are a separate +3).

Specific tests located:

- `TestRepositoryPermissionGrantsSchemaAndIndex` — `permission_grants_repo_test.go:17-84` — schema columns, lookup index, UNIQUE autoindex.
- `TestRepositoryPermissionGrantsRoundTrip` — `:88-169` — happy-path insert+list+delete with ordering + ErrNotFound on missing id.
- `TestRepositoryPermissionGrantsIdempotentInsert` — `:174-238` — duplicate-tuple insert returns nil; original `GrantedBy` + `GrantedAt` preserved (the load-bearing invariant).
- `TestRepositoryPermissionGrantsCrossProjectIsolation` — `:242-288` — project A's grant invisible to project B.
- `TestRepositoryPermissionGrantsCrossCLIIsolation` — `:295-367` — claude grant invisible to codex lookup; both can coexist for same rule (Drop 4d-critical).
- `TestRepositoryPermissionGrantsKindFilter` — `:372-430` — `KindBuild` grants don't leak into `KindBuildQAProof` lookups.
- `TestRepositoryPermissionGrantsValidationErrors` — `:438-484` — adapter-level fail-closed for empty ID, zero GrantedAt, empty cliKind on List, empty id on Delete.

Plus `mustCreateProject` helper at `:489-499` satisfies the FK so tests aren't trivially insertable. **All 7 verified.**

### 9. Worklog includes explicit fresh-DB callout

`workflow/drop_4c/4c_F7_17_7_BUILDER_WORKLOG.md:16-32` — explicit heading "Dev FRESH-DB CALLOUT (DO THIS BEFORE NEXT `mage ci`)" with the literal command:

```sh
rm -f ~/.tillsyn/tillsyn.db
```

Plus a careful disambiguation that `mage ci`'s ephemeral `t.TempDir()` + `OpenInMemory()` paths don't touch the host DB — only the dev's local `till` install needs the wipe. **Verified.**

### 10. NO commit by builder

`git log --oneline -3`:

```
e19e9f0 docs(drop-4c): add f.7.2 qa proof and falsification artifacts
f6aec8b feat(templates): add tool-gating + sandbox + sysprompt fields (4c F.7.2)
31700b6 feat(dispatcher): add claude adapter implementing cli adapter interface
```

No new `F.7.17.7`-related commit. `git status --porcelain` shows the 7 listed builder-output files as untracked + 2 modifications (`internal/adapters/storage/sqlite/repo.go`, `internal/domain/errors.go`) staged for the orchestrator's commit step. **Verified.**

### 11. Scope: only the listed files touched (one nit)

`git status --porcelain` modified files:

- `internal/adapters/storage/sqlite/repo.go` (DDL append — listed).
- `internal/domain/errors.go` (3 sentinels appended — listed).
- `workflow/drop_4c/SKETCH.md` (NIT — not listed in builder's 8-file declaration).

Untracked:

- `internal/domain/permission_grant.go` (listed).
- `internal/domain/permission_grant_test.go` (listed).
- `internal/app/permission_grants_store.go` (listed).
- `internal/adapters/storage/sqlite/permission_grants_repo.go` (listed).
- `internal/adapters/storage/sqlite/permission_grants_repo_test.go` (listed).
- `workflow/drop_4c/4c_F7_17_7_BUILDER_WORKLOG.md` (listed).

Plus several pre-existing untracked workflow / dispatcher artifacts from concurrent F.7 droplets — not authored by THIS builder.

**Nit N1**: `workflow/drop_4c/SKETCH.md` shows a one-line edit (`~25–35 droplets` → `~28–40 droplets ... CLI adapter seam + ... context aggregator added 2026-05-04`) plus a multi-line F.7.17 SKETCH expansion. Per `git log` and the SKETCH content, this edit predates this droplet — it documents the F.7 wave architecture and was authored alongside `F7_17_CLI_ADAPTER_PLAN.md`. The SKETCH edit is appropriate context for F.7.17.7 (it explains why `cli_kind` exists at all) but it falls outside this droplet's declared 8-file scope. Recommend the orchestrator confirm the SKETCH delta belongs with this droplet's commit or with a planning droplet that already landed.

## Hard-constraint check

- **No Hylla calls used to mutate state.** Two read-only `hylla_search_keyword` calls were attempted for symbol verification (`translateNoRows`, `IsValidKind`, `scanner`); a third hybrid call hit "enrichment still running" and was abandoned in favor of `git grep` and direct `Read`. No code edits.
- **No build-runner overrides.** All verification went through `mage ci`, `mage testPkg`. No raw `go test`.

## Summary

**PROOF GREEN.** Every acceptance criterion has direct file:line evidence. Builder deviation (TEXT IDs over BLOB/uuid.UUID) is consistency-driven across seven prior tables and avoids promoting `google/uuid` to a direct dep — accepted as sound. `mage ci` green at 2415 tests with all coverage gates met. The seven new tests exercise the load-bearing invariants (idempotency preserves original audit trail, cross-project isolation, cross-CLI isolation enabling Drop 4d, kind filter, full validation matrix). One nit (N1: SKETCH.md edit not in declared file list) is informational — orchestrator decides commit grouping.

## Hylla Feedback

- **Query**: `hylla_search` (hybrid mode) for `IsValidKind closed enum membership domain Kind`.
  - **Missed because**: Returned `enrichment still running for github.com/evanmschultz/tillsyn@main`. Hybrid-mode call rejects on incomplete enrichment even when keyword sub-mode could still answer.
  - **Worked via**: `Read` of `internal/domain/kind.go:50-52` (located via `git grep`).
  - **Suggestion**: When enrichment is partial, allow `hylla_search` to fall back to keyword-only and return that subset rather than failing the whole call.
- **Query**: `hylla_search_keyword` for `translateNoRows`.
  - **Missed because**: Returns five `Repository.*` methods (DeleteProject, DeleteTask, UpdateTask, ResolveAttentionItem, UpdateCapabilityLease) that USE `translateNoRows` but not the helper itself. The helper is unexported (`func translateNoRows`) — keyword-content search treats unexported helpers like Repository methods. The search hit body-text mentions, not the declaration site.
  - **Worked via**: `git grep -n "translateNoRows" -- 'internal/adapters/storage/sqlite/'` then `Read repo.go:3359-3369`.
  - **Suggestion**: Add a `definition_only` filter that ranks the symbol's declaration above usage sites; or expose `block_kind=function` filtering in `hylla_search_keyword`.
- **Query**: `hylla_search` with `field: "content"` (singular field shape).
  - **Missed because**: Tool returned `field must be summary, content, or docstring` — but my call passed `fields: ["content"]` (the plural form). Plural form documented in tool schema; one of those branches still validates the singular `field` parameter even when the plural is supplied.
  - **Worked via**: Switched to `hylla_search_keyword` with `fields: ["content"]`.
  - **Suggestion**: Pick one shape (plural array) and remove the singular validation path so the error message stops contradicting the documented schema.

Three Hylla misses. Two are ergonomics (fallback-on-partial-enrichment, definition-vs-usage ranking). One is a parameter-shape contradiction.
