# Drop 4c Droplet F.7.9 — QA Falsification Review

**Droplet**: F.7.9 — Action-item metadata fields (`spawn_bundle_path`, `spawn_history[]`, `actual_cost_usd`).
**Mode**: filesystem-MD coordination, read-only adversarial review.
**Reviewer model**: opus.

## 1. Findings

### 1.1 Per-attack verdicts (A1-A12 + A13 self-attack)

| ID  | Attack                                                                 | Verdict   |
| --- | ---------------------------------------------------------------------- | --------- |
| A1  | `*float64 ActualCostUSD` omitempty backwards-compat                    | REFUTED   |
| A2  | `SpawnHistory []SpawnHistoryEntry` empty-vs-nil round-trip             | REFUTED   |
| A3  | `AppendSpawnHistory` concurrent-call race                              | REFUTED   |
| A4  | `normalizeSpawnHistory` UTC + trim + caller-mutation                   | REFUTED   |
| A5  | `SpawnHistoryEntry` 6-field set per master PLAN                        | REFUTED   |
| A6  | `SpawnHistory` audit-only doc-comment + F.7.18 citation                | REFUTED   |
| A7  | TUI schema-coverage guardrail classification                           | REFUTED   |
| A8  | `ActualCostUSD` nil-vs-zero pointer-bool semantics doc                 | REFUTED   |
| A9  | Single-text-column JSON-blob storage assumption                        | REFUTED   |
| A10 | `mage ci` green claim — test-count plausibility                        | NIT       |
| A11 | No-migration-pre-MVP rule compliance                                   | REFUTED   |
| A12 | Forward-compat with F.7.18 round-history aggregation                   | REFUTED   |
| A13 | Slice-aliasing in `AppendSpawnHistory` (self-attack)                   | REFUTED   |

### 1.2 Evidence trail

- **A1.** `ActualCostUSD *float64 \`json:"actual_cost_usd,omitempty"\`` (`workitem.go:238`). Go's `encoding/json` treats nil pointer as empty under `omitempty` — the key is omitted at marshal. `TestActionItemMetadataSpawnFieldsZeroValueRoundTrips` (`domain_test.go:1722-1748`) asserts the zero-value blob does NOT contain `"actual_cost_usd"`. Backwards-compat preserved: a pre-Drop-4c reader marshalling a never-dispatched item never emits the key.
- **A2.** `SpawnHistory []SpawnHistoryEntry \`json:"spawn_history,omitempty"\`` (`workitem.go:227`). Per the Go spec, `omitempty` omits zero-length slices regardless of nil-vs-non-nil-empty. `normalizeSpawnHistory` (`workitem.go:710-713`) explicitly returns nil on empty input, so the canonicalized form is uniformly nil. Same zero-value test asserts no `"spawn_history"` key on encode.
- **A3.** `AppendSpawnHistory` performs `append(t.Metadata.SpawnHistory, entry)` (`action_item.go:654`) without internal locking — concurrent invocation on the same `*ActionItem` would race. The method's doc-comment (`action_item.go:635-647`) and `SpawnHistory`'s field doc (`workitem.go:226`) BOTH explicitly document the single-writer-via-action-item-scoped-lock contract ("atomicity is inherited from the action-item-scoped lock — Drop 4a Wave 2 lock manager"). This matches the contract every other `*ActionItem` mutator follows (`Move`, `UpdateDetails`, `SetLifecycleState`, `Restore`). Contract is explicit; not a counterexample.
- **A4.** `normalizeSpawnHistory` (`workitem.go:710-724`) iterates with `for _, entry := range in` — `entry` is a value copy; caller-owned input is untouched. Output is a fresh `make([]SpawnHistoryEntry, 0, len(in))` slice. `entry.StartedAt.UTC()` and `entry.TerminatedAt.UTC()` call `time.Time.UTC()`, which returns a new `time.Time` representing the same instant in UTC — a `time.Local` input round-trips to its UTC-equivalent instant without losing the absolute timestamp. `TestNormalizeActionItemMetadataTrimsSpawnFields` (`domain_test.go:1835-1895`) verifies America/Los_Angeles input → UTC output via `time.Equal`.
- **A5.** `SpawnHistoryEntry` (`workitem.go:148-177`) declares all 6 spec fields: `SpawnID string`, `BundlePath string`, `StartedAt time.Time`, `TerminatedAt time.Time`, `Outcome string`, `TotalCostUSD *float64`. Types and JSON tags match the master PLAN F.7.9 spec.
- **A6.** `SpawnHistory` field doc (`workitem.go:217-222`) reads: "AUDIT-ONLY ROLE: this field exists for ledger / dashboard rendering surfaces, NOT for re-prompting fix-builders. Round-history aggregation was DEFERRED in Drop 4c F.7.18 (REV-9) — if a use case for raw stream-json round-history surfaces post-Drop-5, add `prior_round_*` rules per F.7.18 commentary, NOT raw spawn_history reads. Doc-comment requirement enforced per planner-review P-§5.b (owner: F.7.18.6 per master PLAN §5)." Satisfies REV-9 + planner-review P-§5.b verbatim.
- **A7.** `model_test.go:14937-14959` adds the three new fields to the `internal` map of `TestActionItemMetadataSchemaCoverageIsExplicit` with an explanatory comment citing Drop 4.5+ TUI deferral. Guardrail's contract — every metadata field classified as editable/readOnly/internal — is satisfied.
- **A8.** `ActualCostUSD` doc (`workitem.go:228-238`) is explicit: "Pointer because the stream-jsonl event MAY omit cost — nil round-trips as 'cost not reported' and is meaningfully different from `*float64`-of-0 ('zero-cost spawn')." Same distinction echoed in `SpawnHistoryEntry.TotalCostUSD` doc (`workitem.go:171-176`). Future callers reading the doc cannot reasonably conflate nil with 0.
- **A9.** Storage layer (`repo.go`): schema declares `metadata_json TEXT NOT NULL DEFAULT '{}'` (lines 152, 175); insert path runs `json.Marshal(t.Metadata)` into the single text column (`repo.go:1210`); update path mirrors at `repo.go:1322`; read path runs `json.Unmarshal([]byte(metadataRaw), &t.Metadata)` (`repo.go:3011`). No code in `repo.go` references `SpawnHistory`/`SpawnBundlePath`/`ActualCostUSD` symbolically. Single-blob persistence claim verified.
- **A10.** Worklog reports 2286 pass / 1 skip on `mage check`. The diff adds 5 new domain test functions (`TestActionItemMetadataSpawnFieldsZeroValueRoundTrips`, `TestActionItemMetadataSpawnFieldsPopulatedRoundTrip`, `TestNormalizeActionItemMetadataTrimsSpawnFields`, `TestActionItemAppendSpawnHistoryAppendsInOrder`, `TestActionItemAppendSpawnHistoryCanonicalizes`) + 1 SQLite test function (`TestRepository_PersistsActionItemSpawnMetadata`), matching the 5+1 worklog count. As a read-only QA reviewer I cannot independently re-run `mage ci`. The pre-existing `internal/templates::TestTemplateTOMLRoundTrip` flake the worklog notes is unrelated to F.7.9 (no F.7.9 file touches `internal/templates`); `git status --porcelain` shows separate uncommitted modifications under `internal/templates/` that pre-date or post-date this droplet. Recommended: orchestrator independently confirms `mage ci` green from a clean state before commit.
- **A11.** `git diff --stat HEAD` for F.7.9 paths shows: `internal/adapters/storage/sqlite/repo_test.go +237` (test only), `internal/domain/action_item.go +23`, `internal/domain/domain_test.go +324` (test only), `internal/domain/workitem.go +101`, `internal/tui/model_test.go +10` (test only). Total 695 insertions, 0 deletions. `repo.go` is NOT modified. The pre-existing `ALTER TABLE action_items ADD COLUMN metadata_json` at `repo.go:509` shipped in an earlier drop and is idempotent (`isDuplicateColumnErr`). No new schema, no migration script, no DDL — `feedback_no_migration_logic_pre_mvp.md` honored.
- **A12.** `SpawnHistoryEntry` is exported with 6 stable fields. F.7.18 round-history aggregation, when reconsidered post-Drop-5, would land as new derived metadata fields (`prior_round_*` per the doc-comment) or a new aggregator type — neither requires `SpawnHistoryEntry` mutation. Forward-compat clean.
- **A13 (self-attack).** `t.Metadata.SpawnHistory = append(t.Metadata.SpawnHistory, entry)` (`action_item.go:654`) reassigns the slice header to `t.Metadata.SpawnHistory` after append. Standard Go slice-aliasing semantics: any external pointer to the prior header sees the prior length, but the storage layer's `json.Marshal(t.Metadata)` reads through `t.Metadata.SpawnHistory` (the new header) — persistence is correct. No counterexample.

### 1.3 Sanity checks not in spawn prompt

- **`Outcome string \`json:"outcome"\`` in `SpawnHistoryEntry` lacks `omitempty`** — line 170. Empty `Outcome` would emit `"outcome":""`. This is an internal-not-yet-omitted field, but the SpawnHistory slice itself is `omitempty`, so a never-dispatched item still doesn't surface the inner field. **NIT, deferred** — not a backwards-compat hazard because entries only exist after dispatch.
- **`TotalCostUSD` IS `omitempty`** at the inner level (line 176), correctly distinguishing nil from `*float64`-of-0 in the audit-trail entries.
- **`Outcome` on `ActionItemMetadata` (the pre-existing field)** uses `omitempty` (line 200), so the new spawn fields are consistent with the closest precedent.

## 2. Counterexamples

None CONFIRMED. The implementation tracks the F.7.9 spec, REV-6 (JSON-blob, no schema change), REV-9 (round-history deferred + audit-only doc), and the no-migration-pre-MVP memory rule. All 12 spawn-prompt attack vectors plus one self-attack vector either landed REFUTED or surfaced as a non-blocking NIT.

## 3. Summary

**Verdict: PASS-WITH-NITS.**

- A10: I cannot independently re-verify `mage ci` green from this read-only QA pass. Worklog's self-report (2286 pass / 1 skip) plus the unrelated-templates-flake disclaimer is plausible, but a clean-state reproduction by the orchestrator before commit is the responsible mitigation.
- 1.3 minor consistency note: inner `Outcome` field lacks `omitempty` (the outer slice's `omitempty` makes it harmless today, but a future direct caller emitting a single `SpawnHistoryEntry` outside the metadata wrapper would surface `"outcome":""`). Not a counterexample, just a future-proofing flag.

Builder discovered REV-6 / REV-9 correctly, used the existing typed-struct path rather than a `map[string]any` route, satisfied the audit-only doc-comment contract verbatim, and added the TUI guardrail classification proactively (the worklog notes the guardrail caught the omission on first run).

## TL;DR

- T1: All 12 spawn-prompt attacks plus 1 self-attack land REFUTED except A10, which is a NIT (cannot independently rerun `mage ci`); inner-field NIT (`Outcome` lacks `omitempty` but is harmless behind the slice-level `omitempty`) flagged but non-blocking.
- T2: No CONFIRMED counterexamples — JSON-blob persistence, omitempty backwards-compat, UTC normalization, audit-only doc-comment, TUI guardrail, and no-migration-rule all hold.
- T3: PASS-WITH-NITS — orchestrator should reproduce `mage ci` green from a clean state before commit; otherwise droplet is ready to land.
