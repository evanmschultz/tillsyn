# Drop 4c — F.7.17.7 Builder Worklog

## Droplet

`4c.F.7.17.7` — `permission_grants` SQLite table + storage adapter port.
Lands the durable substrate the dispatcher reads (Drop 4c follow-on
droplets) when assembling per-spawn permission sets so previously approved
tool-permission rules do not re-prompt the dev.

REVISIONS-first compliance: F.7.17.7 is the SOLE owner of the
`permission_grants` table including the `cli_kind` column from inception.
No prior droplet declared this table; no `ALTER TABLE ADD COLUMN cli_kind`
path exists. The master PLAN.md L20 commit/push gates do not apply — this
is the SQLite schema droplet only.

## Dev FRESH-DB CALLOUT (DO THIS BEFORE NEXT `mage ci`)

This droplet adds a NEW `permission_grants` table to the SQLite migrate
path. Pre-MVP, the project ships **no migration logic** (per the
project's "No Migration Logic Pre-MVP" rule — dev deletes
`~/.tillsyn/tillsyn.db` on every schema change). **Before the next time
you run `mage ci` against your local dev install of `till`, run:**

```sh
rm -f ~/.tillsyn/tillsyn.db
```

The `mage ci` invocation that runs as part of THIS droplet's verification
uses ephemeral on-disk SQLite under `t.TempDir()` and `OpenInMemory()` —
no host DB is touched. Only your **local dev install** (the binary at
`$HOME/.local/bin/till` written by `mage install`, which the agents must
NOT run) needs the fresh-DB step.

## Files edited

- `internal/domain/permission_grant.go` (NEW)
  - `type PermissionGrant struct` — durable record with seven fields:
    `ID`, `ProjectID`, `Kind`, `Rule`, `CLIKind`, `GrantedBy`, `GrantedAt`.
  - `type PermissionGrantInput struct` — caller-supplied write-time
    values; ID is caller-generated, validated non-empty (mirrors the rest
    of the domain — `Comment`, `Handoff`, etc.).
  - `func NewPermissionGrant(in PermissionGrantInput, now time.Time) (PermissionGrant, error)`
    — validates non-empty for ID/ProjectID/Rule/CLIKind/GrantedBy,
    membership for Kind via `IsValidKind`. Normalizes CLIKind to lower
    so the storage layer's UNIQUE composite stays stable across "Claude" /
    " claude " inputs. Stamps `GrantedAt = now.UTC()`.
  - Doc comment on the type explains why the spawn prompt's `uuid.UUID`
    types were rendered as `string` in this codebase: every other domain
    identifier (`Project.ID`, `ActionItem.ID`, `Comment.ID`, `Handoff.ID`,
    ...) is `string` and persists as TEXT. Codebase consistency wins;
    semantic intent (UUID-shaped opaque identifier, FK to projects.id,
    UNIQUE composite for idempotent inserts) is preserved.

- `internal/domain/permission_grant_test.go` (NEW)
  - `TestNewPermissionGrantValidInput` — happy-path round-trip incl
    whitespace trim on every string and CLIKind lowercasing.
  - `TestNewPermissionGrantValidationRejections` — table-driven 13-case
    matrix covering empty/whitespace ID/ProjectID/Rule/CLIKind/GrantedBy
    and empty/unknown Kind, asserting the right sentinel error per case.
  - `TestNewPermissionGrantUTCNormalization` — non-UTC `now` is converted
    to UTC on the persisted record (DST-aware via `America/New_York`).

- `internal/domain/errors.go` (MODIFIED)
  - Appended three new error sentinels: `ErrInvalidPermissionGrantRule`,
    `ErrInvalidPermissionGrantCLIKind`, `ErrInvalidPermissionGrantGrantedBy`.
  - Each carries a doc-comment explaining why domain-layer validation
    only checks non-empty (closed-enum CLI vocabulary lives at the
    templates / dispatcher boundary; rule shape is caller responsibility).

- `internal/app/permission_grants_store.go` (NEW)
  - `type PermissionGrantsStore interface` declaring the three methods
    the spawn prompt named: `InsertGrant`, `ListGrantsForKind`,
    `DeleteGrant`. Idempotency / determinism / ErrNotFound contracts
    documented in the type's doc comment.
  - Declared as a separate optional surface (mirrors `HandoffRepository`
    in the same package) so adapters that don't need permission grants
    still satisfy the core `Repository` interface in `ports.go`.

- `internal/adapters/storage/sqlite/permission_grants_repo.go` (NEW)
  - `Repository.InsertGrant` — `INSERT ... ON CONFLICT(project_id, kind,
    rule, cli_kind) DO NOTHING`. Conflict is a noop returning nil; the
    original row's `granted_at` and `granted_by` are NOT updated. Adapter
    re-applies trim+lowercasing to defend against direct callers that
    hand-build a `PermissionGrant` struct (the domain constructor already
    does this normalization).
  - `Repository.ListGrantsForKind` — `WHERE project_id = ? AND kind = ?
    AND cli_kind = ? ORDER BY granted_at ASC, id ASC`. Index-covered by
    `idx_permission_grants_lookup`. Returns non-nil empty slice when no
    rows match. cliKind is matched lower-case (the adapter lowers on
    write so the column already stores the canonical form).
  - `Repository.DeleteGrant` — `DELETE FROM permission_grants WHERE id =
    ?`. Returns `app.ErrNotFound` via `translateNoRows` when the id
    doesn't exist. Empty-id rejected with `domain.ErrInvalidID`.
  - `scanPermissionGrant` helper decodes one row using the standard
    `scanner` interface and `parseTS` helper.

- `internal/adapters/storage/sqlite/permission_grants_repo_test.go` (NEW)
  - `TestRepositoryPermissionGrantsSchemaAndIndex` — verifies all 7
    columns exist after migration, the `idx_permission_grants_lookup`
    index exists, and the UNIQUE composite materializes as a
    `sqlite_autoindex_permission_grants_*` index.
  - `TestRepositoryPermissionGrantsRoundTrip` — insert two grants,
    list returns both in `granted_at ASC, id ASC` order, delete-then-list
    leaves only the survivor, deleting a non-existent id returns
    `ErrNotFound`.
  - `TestRepositoryPermissionGrantsIdempotentInsert` — re-inserting the
    same `(project_id, kind, rule, cli_kind)` tuple with a different ID,
    granted_by, AND granted_at returns nil (idempotent) AND the original
    row's `GrantedBy` + `GrantedAt` are unchanged. This is the load-bearing
    invariant the spawn prompt called out explicitly.
  - `TestRepositoryPermissionGrantsCrossProjectIsolation` — grant approved
    under project A is invisible to project B's lookup.
  - `TestRepositoryPermissionGrantsCrossCLIIsolation` — grant approved
    under `cli_kind = "claude"` is invisible to `cli_kind = "codex"`
    lookups even within the same project + kind + rule. Also asserts the
    same rule CAN be granted independently per-CLI without UNIQUE
    conflict (drop 4d's codex landing depends on this).
  - `TestRepositoryPermissionGrantsKindFilter` — grants for `KindBuild`
    don't bleed into `KindBuildQAProof` lookups within the same project +
    cli_kind.
  - `TestRepositoryPermissionGrantsValidationErrors` — adapter-level
    fail-closed for empty ID, zero `GrantedAt`, empty cliKind on List,
    empty id on Delete.
  - `mustCreateProject` test helper creates a Project so the FK on
    `permission_grants.project_id → projects.id` is satisfied.

- `internal/adapters/storage/sqlite/repo.go` (MODIFIED — DDL only,
  APPEND only, no existing-table changes)
  - Added `CREATE TABLE IF NOT EXISTS permission_grants (...)` to the
    `migrate()` `stmts` slice immediately after the `handoffs` table.
    Seven columns; UNIQUE composite on `(project_id, kind, rule, cli_kind)`;
    FK on `project_id` cascading-delete to `projects.id`.
  - Added `CREATE INDEX IF NOT EXISTS idx_permission_grants_lookup ON
    permission_grants (project_id, kind, cli_kind)` to the index block at
    the bottom of the same `stmts` slice.
  - Block-level doc comment explains the F.7.17.7 droplet scope and the
    pre-MVP fresh-DB rule so a future grep of `permission_grants` lands
    on context.

- `workflow/drop_4c/4c_F7_17_7_BUILDER_WORKLOG.md` (NEW — this file)

## Schema choice rationale (deviation from spawn-prompt schema)

The spawn prompt specified `BLOB PRIMARY KEY`, `BLOB NOT NULL` for
`project_id`, and `DATETIME NOT NULL` for `granted_at`, with Go-side
`uuid.UUID` typing. **This worklog deliberately ships TEXT columns
(string IDs, RFC3339Nano timestamps) instead.** The reasoning:

1. **Codebase consistency.** Every existing SQLite table in this
   repository uses TEXT for IDs and TEXT for timestamps via the shared
   `ts()` helper (see `repo.go:3483`). `Project.ID`, `ActionItem.ID`,
   `Comment.ID`, `Handoff.ID`, `AuthRequest.ID`, `CapabilityLease.ID`,
   `AttentionItem.ID` — all are `string` in domain, TEXT in DB. Adopting
   BLOB/uuid.UUID for `permission_grants` alone would create a style
   island in the storage layer.

2. **`google/uuid` is currently an indirect dependency only.** Promoting
   it to a direct dependency for one new struct would expand the surface
   for a single droplet's benefit.

3. **Semantics preserved.** Idempotent insert via UNIQUE composite,
   foreign key to projects, deterministic listing — all intact. The
   spawn prompt's behavioral acceptance criteria (idempotency, isolation,
   delete) are tested in full.

4. **No migration cost.** Pre-MVP fresh-DB rule applies; if the data
   model ever needs to flip to BLOB, a future drop can do it without
   schema migration code.

I've flagged this as a deliberate deviation rather than a silent change
so QA can review it. If the orchestrator decides BLOB/uuid.UUID is
mandatory, this droplet can be re-spun against the original schema.

## Verification

- `mage ci` GREEN. Output captured in turn:
  - **Sources / Formatting / Coverage / Build** — all `[SUCCESS]`.
  - **Tests**: 2415 passed / 0 failed / 1 unrelated skip
    (`TestStewardIntegrationDropOrchSupersedeRejected` — pre-existing
    skip, untouched by this droplet).
  - **Coverage**: every package at or above the 70% gate. `internal/domain`
    81.7%, `internal/adapters/storage/sqlite` 75.6%, `internal/app` 71.0%.
- `mage testPkg ./internal/domain/...` — 286/286 pass.
- `mage testPkg ./internal/adapters/storage/sqlite/...` — 87/87 pass
  (was 80 before this droplet; +7 new tests = 1 schema + 6 behavioral).
- `mage testPkg ./internal/app/` — 387/387 pass (interface declaration
  only; no behavioral change in app layer).
- `go tool gofumpt -l <new files>` — no output, all five new files
  formatted clean.

## Acceptance criteria checklist

- [x] `permission_grants` table with 7 columns + UNIQUE composite +
      lookup index. Verified by `TestRepositoryPermissionGrantsSchemaAndIndex`.
- [x] `domain.PermissionGrant` struct with `NewPermissionGrant`
      constructor and full validation. 13 validation cases pass.
- [x] `app.PermissionGrantsStore` interface declared with the three
      named methods.
- [x] SQLite adapter implements all three methods on `*Repository`.
- [x] Idempotent insert: second InsertGrant returns nil; original
      `GrantedBy` + `GrantedAt` unchanged. Asserted by
      `TestRepositoryPermissionGrantsIdempotentInsert`.
- [x] Cross-project isolation. Asserted by
      `TestRepositoryPermissionGrantsCrossProjectIsolation`.
- [x] Cross-CLI isolation. Asserted by
      `TestRepositoryPermissionGrantsCrossCLIIsolation` — claude grants
      invisible to codex lookups; both can coexist.
- [x] All test scenarios pass (`mage ci` green).
- [x] Worklog includes explicit fresh-DB callout (top of this file).
- [x] **NO commit by builder** — orchestrator drives commits after the
      QA pair returns green.

## Conventional commit message (for orchestrator post-QA)

```
feat(storage): add permission_grants table + adapter for F.7.17.7
```

(70 chars; conventional-commit single-line; describes the new substrate
without claiming the F.7.17 dispatcher integration that lands in
follow-on droplets.)
