# Drop 4c — F.7.9 Builder QA Proof Review

**Verdict**: PROOF GREEN

**Droplet**: F.7.9 — Action-item metadata fields (`spawn_bundle_path`, `spawn_history[]`, `actual_cost_usd`).
**Mode**: filesystem-MD review, read-only. Round 1.
**Builder model**: opus.
**Reviewer model**: opus (qa-proof-agent).

REVISIONS-first compliance: REV-6 (JSON-blob, no new SQLite columns) and REV-9 (round-history-deferred, doc-comment required) checked first.

## 1. Findings

Per-criterion proof, each anchored to file:line.

### 1.1 Three new fields with explicit JSON tags + Go doc-comments — PASS

`internal/domain/workitem.go:201-238` adds `SpawnBundlePath`, `SpawnHistory`, `ActualCostUSD` to `ActionItemMetadata`.

- `SpawnBundlePath string` — `json:"spawn_bundle_path,omitempty"` — `internal/domain/workitem.go:210`. Doc-comment lines 201-209 explain dispatch-time-set / terminal-cleared semantics + cite REV-6.
- `SpawnHistory []SpawnHistoryEntry` — `json:"spawn_history,omitempty"` — `internal/domain/workitem.go:227`. Doc-comment lines 211-226 (covered separately under §1.2).
- `ActualCostUSD *float64` — `json:"actual_cost_usd,omitempty"` — `internal/domain/workitem.go:238`. Doc-comment lines 228-237 explain `*float64` rationale + cite REV-6.

`SpawnHistoryEntry` itself (`internal/domain/workitem.go:142-177`): five fields with explicit JSON tags + per-field doc-comments. `TotalCostUSD *float64` carries `omitempty` (line 176) — correct for the nil-vs-zero-cost edge.

The droplet brief mentioned "TOML/JSON tags" but `ActionItemMetadata` is a JSON-only domain surface (TOML lives under `internal/templates/`). JSON-only tags are the correct choice — verified by precedent: every existing field on the struct (e.g. `Objective`, `KindPayload`, `Outcome`) carries JSON tags only. NIT-grade ergonomic note, not a finding.

### 1.2 Doc-comment on `SpawnHistory` cites audit-only role + REV-9 / F.7.18 link — PASS

`internal/domain/workitem.go:211-226` carries the required prose verbatim:

> "AUDIT-ONLY ROLE: this field exists for ledger / dashboard rendering surfaces, NOT for re-prompting fix-builders. Round-history aggregation was DEFERRED in Drop 4c F.7.18 (REV-9) — if a use case for raw stream-json round-history surfaces post-Drop-5, add `prior_round_*` rules per F.7.18 commentary, NOT raw spawn_history reads. Doc-comment requirement enforced per planner-review P-§5.b (owner: F.7.18.6 per master PLAN §5)."

This satisfies REV-9 line 1062 verbatim ("Doc-comment on `spawn_history[]` MUST cite audit-only role + link to round-history-deferred decision in F.7.18 commentary"). Also cites P-§5.b (F.7.9 acceptance line 615) and the F.7.18.6 owner (master PLAN §5).

`AppendSpawnHistory`'s doc-comment (`internal/domain/action_item.go:635-647`) reinforces the no-dedupe rule and references back to `SpawnHistory`'s doc-comment for the audit-only role + REV-9 link — clean cross-reference rather than a copy-paste duplication.

### 1.3 NO new SQLite columns / DDL changes — PASS

`git diff --name-only -- internal/adapters/storage/sqlite/` produces a single file: `repo_test.go`. Storage source files (`repo.go`, `action_items.go`, schema-init paths) are untouched. The persistence path uses the existing JSON-encoded `metadata` text column.

REV-6 enforcement confirmed end-to-end: the diff does NOT touch any DDL surface, no `ALTER TABLE`, no `CREATE TABLE` action_items column add, no schema version bump. JSON blob path only.

### 1.4 NO migration script — PASS

`git diff --name-only HEAD` shows zero files matching `*migrate*`, `*migration*`, or any `.sql` script in `main/scripts/` or `internal/`. Pre-MVP "no migration logic in Go" rule (project CLAUDE.md memory `feedback_no_migration_logic_pre_mvp.md`) honored.

### 1.5 Five domain tests cover the required cases — PASS

`internal/domain/domain_test.go:1709-2031` adds five tests:

- **Test 1 — zero-value omitempty** (`TestActionItemMetadataSpawnFieldsZeroValueRoundTrips`, lines 1718-1745): asserts `spawn_bundle_path`, `spawn_history`, `actual_cost_usd` substrings ABSENT from JSON-encoded zero `ActionItemMetadata` AND decoded zero values round-trip clean.
- **Test 2 — populated round-trip with mixed `*float64`** (`TestActionItemMetadataSpawnFieldsPopulatedRoundTrip`, lines 1747-1812): two-entry `SpawnHistory` — entry 0 has `floatPtr(0.42)`, entry 1 has `nil` TotalCostUSD. Assertions explicitly distinguish nil-vs-non-nil at lines 1798-1805 (the four switch arms hit the edge cases mandated by F.7.9 AC line 614 + 623).
- **Test 3 — normalizer trim+UTC** (`TestNormalizeActionItemMetadataTrimsSpawnFields`, lines 1814-1879): trims `SpawnBundlePath` (line 1843), trims each entry's string fields (lines 1855-1859), UTC-converts `America/Los_Angeles` time-zoned input (lines 1860-1869), empty `SpawnHistory` round-trips as nil (lines 1873-1878).
- **Test 4 — three-append order incl. duplicate SpawnID** (`TestActionItemAppendSpawnHistoryAppendsInOrder`, lines 1881-1965): three sequential appends; the third (line 1957) appends `SpawnHistoryEntry{SpawnID: "spawn-1", Outcome: "killed"}` — duplicate of the first append's SpawnID; assertion on line 1959 confirms history length grows to 3 (no dedupe), assertion on line 1962 confirms the third entry's outcome is preserved.
- **Test 5 — append canonicalization** (`TestActionItemAppendSpawnHistoryCanonicalizes`, lines 1967-2031): trims string fields + UTC-converts times via `AppendSpawnHistory`, exercising the defensive layer in addition to the normalizer pass.

Five tests as claimed; all required cases covered.

### 1.6 SQLite round-trip test covers write → update-with-append → clear-but-keep-history — PASS

`internal/adapters/storage/sqlite/repo_test.go:3964-4200` adds `TestRepository_PersistsActionItemSpawnMetadata`:

- **Write phase** (lines 3995-4055): three cases — `ai-empty-spawnmeta`, `ai-bundle-no-history`, `ai-full-spawnmeta`. Each writes via `CreateActionItem`, reads back via `GetActionItem`, asserts deep-equal via `assertSpawnMetadataEqual`.
- **`ListActionItems` second-SELECT path** (lines 4062-4076): exercises the alternate read path (bonus rigor over the spec).
- **Update-with-append path** (lines 4080-4116): mutates `ai-full-spawnmeta` via `AppendSpawnHistory` for a third entry, persists via `UpdateActionItem`, reads back, asserts `len == 3` (line 4112) + order preserved (line 4115).
- **Clear path with history retention** (lines 4118-4144): re-reads the action item, sets `SpawnBundlePath = ""` + `ActualCostUSD = nil`, persists, reads back, asserts (a) bundle path cleared (line 4135), (b) cost cleared (line 4138), (c) history length still 3 (line 4143) — audit-trail survives the clear.

All three required transitions covered. Plus the helper `assertSpawnMetadataEqual` (lines 4163-4200) gives field-named failure messages rather than opaque `reflect.DeepEqual` diffs.

### 1.7 TUI schema-coverage guardrail updated — PASS

`internal/tui/model_test.go:14949-14958` adds the three new fields to the `internal` map in `TestActionItemMetadataSchemaCoverageIsExplicit`. Inline doc-comment (lines 14949-14955) documents the Drop 4.5+ TUI surfacing deferral. Classifying as `internal` (not `editable`, not `read-only`) is the correct call: the dispatcher writes them, the TUI does not render or edit them today.

`assertExplicitFieldCoverage` invocation on line 14960 will now exhaust the struct's reflect-discovered field list — adding any future spawn-metadata field without classifying it will fail this guardrail (the original purpose of the test).

### 1.8 `mage ci` green per worklog: 2286 tests across 21 packages — PASS (per worklog claim)

Worklog `workflow/drop_4c/4c_F7_9_BUILDER_WORKLOG.md:35-45` claims:

- `mage check` — 21/21 packages, 2286/2287 (1 skip pre-existing).
- `mage ci` — full green: tracked sources verified, listed Go files, format checked, test stream OK, coverage threshold met, binary built.
- Per-package: `mage test-pkg ./internal/domain` 271 tests, `mage test-pkg ./internal/adapters/storage/sqlite` 80 tests.

Read-only review — I did not re-run `mage ci`. Worklog also notes a transient flake on `internal/templates::TestTemplateTOMLRoundTrip` that resolved on second invocation (worklog line 49). The flake is unrelated to F.7.9 — F.7.9 does not touch `internal/templates/`. Recorded as NIT under §3 below.

### 1.9 Scope: only the 5 listed files + worklog touched — PASS

`git diff --name-only HEAD` returns 11 modified files + 1 untracked worklog. F.7.9-claimed files (5):

- `internal/domain/workitem.go`
- `internal/domain/action_item.go`
- `internal/domain/domain_test.go`
- `internal/adapters/storage/sqlite/repo_test.go`
- `internal/tui/model_test.go`

The other 6 modified files (`internal/templates/*` × 5 + `workflow/drop_4c/SKETCH.md`) belong to the sibling F.7.17.1 droplet — confirmed via `workflow/drop_4c/4c_F7_17_1_BUILDER_WORKLOG.md` lines 9-26. The shared working tree carries both droplets' deltas concurrently; F.7.9's scope discipline holds.

Plus the F.7.9 worklog `workflow/drop_4c/4c_F7_9_BUILDER_WORKLOG.md` — present and well-structured.

## 2. Missing Evidence

- 2.1 None. All 9 review criteria have direct file:line evidence.

## 3. Nits (Non-Blocking)

- 3.1 **Worklog flake on `internal/templates::TestTemplateTOMLRoundTrip`** (`workflow/drop_4c/4c_F7_9_BUILDER_WORKLOG.md:49`). Builder noted a transient first-invocation failure that did not reproduce. Out of F.7.9's scope (F.7.9 does not edit `internal/templates/`); flagged here for sibling F.7.17.1 builder to be aware of since their droplet DOES edit that package. Not a F.7.9 finding.
- 3.2 **Droplet brief mentioned "TOML/JSON tags"** but `ActionItemMetadata` is a JSON-only domain struct (TOML decoding lives in `internal/templates/`). Builder correctly used JSON-only tags matching the existing struct precedent (`Objective`, `KindPayload`, `Outcome`, etc.). The brief's wording was imprecise; builder's interpretation is correct. Not a finding.
- 3.3 **`AppendSpawnHistory` cross-references `SpawnHistory`'s doc-comment for audit-only role rather than copy-pasting**. This is good practice (single source of truth) — flagging here as a design call worth preserving in future builds: REV-9 mandate sits on the field, not on the helper.

## 4. Summary

PROOF GREEN.

- Premises (the 9 review criteria) all held: typed fields with JSON tags + doc-comments; REV-6 (no DDL) verified by single-file `git diff` on storage; REV-9 (audit-only doc-comment + F.7.18 link) verified verbatim; tests cover all required cases (5 domain + 1 storage round-trip with three transitions); TUI guardrail correctly updated; scope discipline held (sibling droplet deltas accounted for); `mage ci` green per worklog.
- Evidence: file:line cites in §1.1-§1.9.
- Trace / cases: write → update-append → clear paths covered both at the domain layer (5 tests) and the storage layer (1 multi-phase test). `*float64` nil-vs-zero edge explicitly tested at both layers.
- Conclusion: F.7.9 is ready to mark complete.
- Unknowns: I did not re-run `mage ci` myself (read-only review); proof leans on the worklog's claim. The transient `internal/templates::TestTemplateTOMLRoundTrip` flake is the only soft spot but is sibling-droplet territory, not F.7.9.

## TL;DR

- **T1**: F.7.9 PROOF GREEN — three typed fields land in `ActionItemMetadata` with explicit JSON tags + doc-comments, REV-6 (no DDL) and REV-9 (audit-only doc-comment with F.7.18 link) both honored.
- **T2**: No missing evidence; all 9 criteria have file:line proof in §1.
- **T3**: Three NITs noted, none blocking — sibling-droplet templates flake, JSON-vs-TOML brief wording, and a praise-note for `AppendSpawnHistory` cross-referencing rather than duplicating doc-comments.
- **T4**: Verdict: PROOF GREEN — F.7.9 is ready for `complete`. `mage ci` claim taken at face value (read-only review).

## Hylla Feedback

`N/A — F.7.9 review touched non-Go files only via Read (the worklog MD + plan MD). Go source review used `git diff` on the 5 listed files plus targeted `Read` on `workitem.go` and `model_test.go` for context. No Hylla queries issued; per project CLAUDE.md "Hylla Indexes Only Go Files Today" + the working-tree state (uncommitted work since last ingest, so Hylla would be stale anyway), `git diff` was the right primary source. No miss to record.`
