# VEC Audit 02: SQLite + ncruces + sqlite-vec
Timestamp: 2026-03-03T17:34:36-0700 (2026-03-04T00:34:36Z)

## Commands Run And Outcomes
- `rg -n ... VECTOR_SEARCH_EXECUTION_PLAN.md` + `nl -ba VECTOR_SEARCH_EXECUTION_PLAN.md` (sections 2 and 6 inspected): `ok`
- Context7 `resolve-library-id` + `query-docs`:
  - `/ncruces/go-sqlite3` (driver/runtime config/memory defaults): `ok`
  - `/asg017/sqlite-vec-go-bindings` (ncruces binding integration): `ok`
- `rg -n ... internal/adapters/storage/sqlite/repo.go internal/adapters/storage/sqlite/repo_test.go`: `ok`
- `nl -ba internal/adapters/storage/sqlite/repo.go` (runtime/migrate/vector method paths): `ok`
- `nl -ba internal/adapters/storage/sqlite/repo_test.go` (coverage and vector tests): `ok`
- `nl -ba go.mod` + `nl -ba go.sum` (relevant pins): `ok`
- `just test-pkg ./internal/adapters/storage/sqlite`: `ok` (`cached`)

## Findings By Severity

### High
1. Plan section 6 schema scope is only partially implemented in storage adapter.
- Evidence:
  - Plan expects search docs + FTS + vec table ensures (`VECTOR_SEARCH_EXECUTION_PLAN.md:108-112`).
  - Adapter currently ensures `task_embeddings` as a regular table and indexes (`internal/adapters/storage/sqlite/repo.go:169-180`), then performs brute-force cosine computation (`internal/adapters/storage/sqlite/repo.go:1430-1440`).
- Risk:
  - No FTS ensure and no explicit `vec0` virtual table ensure in this scoped adapter path; ranking/recall/performance behavior may diverge from planned storage design as data grows.

2. No adapter-level vec capability guard before vec SQL functions.
- Evidence:
  - Upsert calls `vec_f32(?)` directly (`internal/adapters/storage/sqlite/repo.go:1375-1377`).
  - Search calls `vec_distance_cosine(..., vec_f32(?))` directly (`internal/adapters/storage/sqlite/repo.go:1433-1435`).
  - No `vec_version()` probe or feature-check path in `Open/migrate`.
- Risk:
  - If vec functions are unavailable at runtime, vector upsert/search fail at call sites; resilience depends on higher-layer best-effort behavior rather than storage-layer capability handling.

### Medium
1. Vec-unavailable behavior is not explicitly covered in adapter tests.
- Evidence:
  - Only happy-path vector roundtrip test is present (`internal/adapters/storage/sqlite/repo_test.go:110-191`).
  - No test asserting graceful adapter behavior when vec functions are missing.
- Risk:
  - Regression can silently break semantic retrieval/write paths without dedicated adapter-level safety tests.

2. Foreign-key enablement is set via PRAGMA during migrate execution, not explicitly encoded in DSN.
- Evidence:
  - `PRAGMA foreign_keys = ON;` issued in migrate statements (`internal/adapters/storage/sqlite/repo.go:92`).
  - DB opened with plain `sql.Open("sqlite3", path)` (`internal/adapters/storage/sqlite/repo.go:58`).
- Risk (inference):
  - SQLite PRAGMA foreign-key behavior is connection-scoped; with pooled `database/sql` usage, FK/cascade assumptions for embedding rows can drift if not consistently applied per connection.

## Completeness Checklist Vs Plan (Sections 2, 6)
- Section 2.1 driver/storage direction (`ncruces` + sqlite-vec ncruces binding): `PASS`
  - `internal/adapters/storage/sqlite/repo.go:16-23`, `go.mod:14,21`
- Section 2.3 forward-only idempotent ensures/no legacy backfill framework: `PASS`
  - `CREATE TABLE IF NOT EXISTS`/`CREATE INDEX IF NOT EXISTS` pattern in `internal/adapters/storage/sqlite/repo.go:90-279`
- Section 2.4 fallback policy (keyword when semantic path unavailable): `PASS (outside storage core; inferred from app behavior)`
  - Adapter methods still hard-fail on missing vec functions (`internal/adapters/storage/sqlite/repo.go:1375-1377`, `1433-1435`)
- Section 6.1 schema ensures (search docs + FTS + vec tables + optional queue): `FAIL (partial implementation only)`
  - Implemented: `task_embeddings` table/indexes (`internal/adapters/storage/sqlite/repo.go:169-180`)
  - Missing in scoped adapter file: explicit FTS ensure and explicit vec virtual-table ensure.
- Section 6.2 keep existing migrations intact: `PASS`
  - Existing legacy migration paths remain in place (`internal/adapters/storage/sqlite/repo.go:286-349`)
- Section 6.3 vec unavailable should not break open + keyword operational: `PASS WITH QA GAP`
  - Open/migrate path does not execute vec SQL functions (`internal/adapters/storage/sqlite/repo.go:90-349`)
  - No adapter test explicitly validates vec-unavailable scenario (`internal/adapters/storage/sqlite/repo_test.go:110-191`)

## Residual Risks
- Storage layer lacks an explicit vec capability negotiation path; vec function failures are deferred to runtime calls.
- Adapter test suite does not currently validate vec-unavailable behavior or capability fallback at adapter boundary.
- Schema plan conformance risk remains for FTS/vec-table ensure completeness versus planned model in section 6.

## Explicit Final Verdict
- **Conditional pass for pre-dogfood storage/runtime baseline; not complete against full Section 6 schema plan.**
- **Primary QA blockers to full compliance:** missing explicit FTS/vec-table ensure coverage and missing adapter-level vec-unavailable test coverage.
