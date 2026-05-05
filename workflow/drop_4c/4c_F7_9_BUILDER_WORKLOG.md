# Drop 4c Droplet F.7.9 — Builder Worklog

**Droplet**: F.7.9 — Action-item metadata fields (`spawn_bundle_path`, `spawn_history[]`, `actual_cost_usd`).
**Mode**: filesystem-MD coordination (no Tillsyn writes).
**Builder model**: opus.

## Summary

Added three Drop 4c F.7.9 spawn-metadata fields to `domain.ActionItemMetadata` as JSON-blob entries (REV-6: no new SQLite columns). Audit-only `spawn_history[]` carries the planner-mandated doc-comment citing F.7.18 round-history-deferred decision (REV-9).

## Files Touched

- `internal/domain/workitem.go` — added `SpawnHistoryEntry` struct (5 fields with JSON tags), three new `ActionItemMetadata` fields (`SpawnBundlePath string`, `SpawnHistory []SpawnHistoryEntry`, `ActualCostUSD *float64`), and `normalizeSpawnHistory` helper. Extended `normalizeActionItemMetadata` to canonicalize the new fields (trim strings, UTC times, no dedupe on history).
- `internal/domain/action_item.go` — added `AppendSpawnHistory(entry, now)` method on `*ActionItem` that canonicalizes + appends the entry and bumps `UpdatedAt`.
- `internal/domain/domain_test.go` — five new tests covering (a) zero-value JSON omitempty round-trip, (b) populated round-trip with mixed `*float64` nil/non-nil cost, (c) normalizer trim+UTC behavior, (d) append-in-order semantics across three appends including duplicate `SpawnID`, (e) append canonicalization (trim + UTC).
- `internal/adapters/storage/sqlite/repo_test.go` — added `TestRepository_PersistsActionItemSpawnMetadata` exercising the JSON-blob round-trip via SQLite for three cases (empty, bundle-path-only, full populated) plus an UPDATE-then-RELOAD path that appends a third history entry and a clear path that resets bundle/cost without losing audit-trail entries.
- `internal/tui/model_test.go` — added the three new field names to the `internal` map in `TestActionItemMetadataSchemaCoverageIsExplicit`. The TUI does not render or edit them today; surfacing in dashboard / ledger is a Drop 4.5+ refinement, classified as internal scaffolding for now.

## Design Decisions

1. **JSON-blob via the existing `ActionItemMetadata` struct, not `map[string]any`.** The droplet spec hint says `Metadata` is "`map[string]any` today," but reading `internal/domain/workitem.go` showed it is a strongly-typed Go struct serialized via `json.Marshal` to a single SQLite text column. Adding typed fields directly to the struct rides the existing pipeline cleanly and matches the precedent set by `Outcome`, `KindPayload`, etc. No `map[string]any` complications, no JSON-double-marshal round-trips for callers.
2. **`*float64` for `ActualCostUSD` and `SpawnHistoryEntry.TotalCostUSD`.** F.7.9 acceptance criteria line 614 + edge case at line 623 both require nil-vs-zero to round-trip distinctly. REV-6's bare `float64` summary is a doc-level summary; the AC governs.
3. **`omitempty` on all three new metadata fields.** Backward-compat for Drop-4a-era items: zero-value items encode without the new keys, so a pre-Drop-4c reader that re-encodes never surfaces an empty `spawn_bundle_path`/`spawn_history`/`actual_cost_usd` it never wrote. Verified by `TestActionItemMetadataSpawnFieldsZeroValueRoundTrips`.
4. **`SpawnHistory` is never deduped.** Audit trail records every dispatch, including retries that reuse `(spawn_id, bundle_path)`. Verified in `TestActionItemAppendSpawnHistoryAppendsInOrder` third append.
5. **`AppendSpawnHistory` is a method on `*ActionItem`, not a free function.** Idiomatic Go; lives next to other lifecycle methods (`Move`, `UpdateDetails`, `SetLifecycleState`, `Restore`). Atomicity inherited from action-item-scoped lock — caller persists via `UpdateActionItem`, no new transaction wrapper.
6. **Spawn history clear-policy: never cleared.** Storage-layer test `TestRepository_PersistsActionItemSpawnMetadata` exercises the bundle-path-clear path and asserts history survives — the field exists for ledger / dashboard rendering, never as scratchpad.
7. **Defensive UTC + trim canonicalization in both the normalizer and `AppendSpawnHistory`.** Belt-and-suspenders: callers (Drop 4c F.7.4 monitor) can pass either UTC or local time without breaking `time.Equal` comparisons in the test suite. Verified by `TestActionItemAppendSpawnHistoryCanonicalizes` using `America/Los_Angeles`.

## Acceptance Criteria

- [x] Helper accessors / setters for `spawn_bundle_path`, `spawn_history`, `actual_cost_usd` added — direct struct fields with the existing `Metadata` mutation pattern. `AppendSpawnHistory` method covers the only non-trivial mutation (history append).
- [x] Doc-comment on `spawn_history[]` cites audit-only role + round-history-deferred reference. Wording: "AUDIT-ONLY ROLE: this field exists for ledger / dashboard rendering surfaces, NOT for re-prompting fix-builders. Round-history aggregation was DEFERRED in Drop 4c F.7.18 (REV-9) — if a use case for raw stream-json round-history surfaces post-Drop-5, add `prior_round_*` rules per F.7.18 commentary, NOT raw spawn_history reads."
- [x] Round-trip test: write the three fields → marshal to SQLite → read back → assert equality. `TestRepository_PersistsActionItemSpawnMetadata` covers three cases including the mixed nil/non-nil `*float64` edge.
- [x] `spawn_history` append semantics tested: starting empty, appending one entry, appending second entry preserves order. `TestActionItemAppendSpawnHistoryAppendsInOrder` asserts on three appends (including a duplicate `SpawnID` to confirm no-dedupe).
- [x] `mage check` passes — 2286 tests passed, 1 skipped (pre-existing). All 21 packages green.
- [x] `mage ci` passes — full pipeline green (format check, test stream, coverage threshold, build).
- [x] No new SQLite columns, no migration script. Confirmed via `git diff -- internal/adapters/storage/sqlite/` showing zero schema-keyword matches (`ALTER TABLE` / `CREATE TABLE` / `CREATE INDEX` / `spawn_bundle` / `spawn_history` / `actual_cost`). Only the test file is touched in that directory.
- [x] Worklog written.

## Verification

- `mage test-pkg ./internal/domain` — 271 tests pass.
- `mage test-pkg ./internal/adapters/storage/sqlite` — 80 tests pass.
- `mage check` — full green, 21/21 packages, 2286/2287 (1 skip pre-existing).
- `mage ci` — full green: tracked sources verified, listed Go files, format checked, test stream OK, coverage threshold met, binary built.

## Notes

- A pre-existing `internal/templates::TestTemplateTOMLRoundTrip` flake appeared in the first `mage check` invocation. It was unrelated to this droplet (F.7.9 doesn't touch the templates package), did not reproduce in `mage test-pkg ./internal/templates`, and the second full `mage check` run came up clean. Likely a transient race with another concurrent test path.
- The TUI guardrail `TestActionItemMetadataSchemaCoverageIsExplicit` enforces explicit classification of every `ActionItemMetadata` field as editable / read-only / internal. Adding the three new fields without classifying them was a real failure that caught my omission — the guardrail did its job. Now fixed by classifying all three as `internal` (per Drop 4.5+ refinement deferral).
