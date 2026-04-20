# DROP_1_75 — Builder QA Falsification

Append a `## Unit N.M — Round K` section per attempt.

## Unit 1.1 — Round 1

**Verdict:** pass-with-findings

## Summary

Attempted ten structural attacks against the `WorkKind → Kind` + `WorkItemID → ActionItemID` rename sweep. All four plan-binding acceptance gates pass (`rg WorkKind` = 0, `rg 'type WorkItem |WorkItemKind|WorkItemID'` = 0, `mage build` green, `mage test-pkg ./internal/domain` green). I additionally ran `mage test-pkg` on every other package in the rename surface (`internal/app`, `internal/tui`, `internal/adapters/storage/sqlite`, `internal/adapters/server/{common,mcpapi,httpapi}`, `cmd/till`) — all green, 1005 tests passing collectively. Private-helper collision check (`isValidKind`), pre-existing `Kind*` surface collision check, string-literal preservation check, and the explicit preservation-list check all passed. No CONFIRMED counterexample. Two non-blocking findings recorded — both are naming-inconsistency residues the plan's narrow acceptance regex did not cover, properly deferrable to a refinement drop.

## Findings

### F1 — Residual middle-of-identifier `WorkItem` names in production symbols

- **Severity:** editorial / defer to refinement drop
- **Where:**
  - `internal/app/embedding_runtime.go:457, 502` — `processWorkItemClaim` method (receiver `*EmbeddingWorker`).
  - `internal/tui/thread_mode.go:359-360` — `startSelectedWorkItemThread` method (plus caller at `internal/tui/model.go:13106` and test-site at `internal/tui/model_test.go:2095`).
- **Counterexample:** none — these are not plan-binding. Plan §1.1 line 73 narrows to "`WorkItem`-prefixed symbols renamed to `ActionItem`-prefixed," and both of these are `WorkItem`-infix (method names prefixed with `process` / `startSelected`), not `WorkItem`-prefixed at position 0. The four acceptance-gate regexes (`type WorkItem `, `WorkItemKind`, `WorkItemID`, `WorkKind`) do not cover infix matches.
- **Suggested fix:** enumerate on the next kind/action-item-naming pass (refinement drop) — `processWorkItemClaim → processActionItemClaim`, `startSelectedWorkItemThread → startSelectedActionItemThread`. Not a Drop 1.75 unit-1.1 regression.

### F2 — Residual local-variable `workItem*` names

- **Severity:** editorial / cosmetic
- **Where:**
  - `internal/tui/model.go:3215-3236` — `workItemID` local in `jumpToActivityNode`.
  - `internal/app/search_embeddings_test.go:44, 52, 53` — `workItemType`, `workItemID` locals.
  - `internal/app/service_test.go:2843, 2852, 2865` — `seenWorkItem` local.
  - `internal/adapters/storage/sqlite/repo.go:611, 626` — `workItemAlterStatements` local inside `ALTER TABLE tasks` block (dies in unit 1.7 per plan §Scope bullet 7 — acceptable churn-avoidance).
  - `internal/adapters/storage/sqlite/repo_test.go:1080-1085` — `workItemCount` local in legacy-tasks-fixture block (also dies in 1.7).
- **Counterexample:** none — plan §1.1 scopes "symbols," not slot-scoped locals. Gates pass. The 1.7-scoped hits are correctly deferred per the same waiver pattern 1.2/1.4/1.6 use.
- **Suggested fix:** clean up during the same refinement drop as F1, or during unit 1.13's TUI test sweep (`m.model_test.go:2095` already in unit 1.13's paths).

## Attacks Attempted (No Counterexample Found)

- **Attack 1 (over-reach on bare-type Pass 6 / pre-existing `Kind*` collision).** `internal/domain/kind.go` holds `type KindID string`, `KindAppliesTo`, `KindDefinition`, `KindTemplate`, `KindTemplateChildSpec` — all semantically distinct from the newly-introduced `type Kind string` at `internal/domain/workitem.go:35`. No name collision (different type identifiers). No shadowing (both in same package at top level, different names). `ActionItem.Kind Kind` field at `task.go:28` correctly resolves against the new `Kind` type. `isValidKind(kind Kind)` at `workitem.go:196` is called from `task.go:130` with `in.Kind` (field type `Kind`) — type-checks.
- **Attack 2 (middle-of-identifier misses).** `rg -i 'workkind'` across the non-workflow tree returns 0 matches. The 7th-pass fix for `TestCommentTargetTypeForWorkKindSupportsHierarchyKinds` is present at `internal/tui/thread_mode_test.go:9-10` as `TestCommentTargetTypeForKindSupportsHierarchyKinds`. No other WorkKind substring residue.
- **Attack 3 (string-literal rot).** `internal/domain/workitem.go:39-43` — `KindActionItem = "actionItem"`, `KindSubtask = "subtask"`, `KindPhase = "phase"`, `KindDecision = "decision"`, `KindNote = "note"`. All RHS string values preserved exactly. `sd` did not touch the RHS literals.
- **Attack 4 (`WorkItem*` preservation list intact).** `rg 'IsValidWorkItemAppliesTo|validWorkItemAppliesTo|EmbeddingSearchTargetTypeWorkItem|EmbeddingSubjectTypeWorkItem|buildWorkItemEmbeddingContent'` returns hits across `internal/domain/{kind.go, task.go, attention_level_test.go, template_library.go}`, `internal/app/{search_embeddings.go, embedding_runtime.go, service.go, snapshot.go, *_test.go}`, `internal/adapters/storage/sqlite/{repo.go, repo_test.go, embedding_lifecycle_adapter*.go, embedding_jobs_test.go}`, `internal/tui/{model.go, model_test.go, model_teatest_test.go}`, `internal/adapters/server/common/app_service_adapter_lifecycle_test.go`. `"work_item"` string literal preserved at `internal/app/search_embeddings.go:25`, `internal/app/embedding_runtime.go:19`, `internal/adapters/storage/sqlite/embedding_lifecycle_adapter.go:446,460`, `internal/adapters/storage/sqlite/embedding_jobs.go:1095`, `cmd/till/embeddings_cli_test.go:99`, `internal/adapters/server/mcpapi/extended_tools_test.go:563,585,2750`.
- **Attack 5 (private-helper collision `isValidKind`).** Only one declaration sitewide: `internal/domain/workitem.go:196 func isValidKind(kind Kind) bool`. Only caller: `internal/domain/task.go:130 if !isValidKind(in.Kind)`. No duplicate symbol.
- **Attack 6 (cross-package ripple).** Ran `mage test-pkg` on all eight affected packages: `internal/domain` (52 tests), `internal/app` (212 tests), `internal/tui` (368 tests), `internal/adapters/storage/sqlite` (69 tests), `internal/adapters/server/common` (130 tests), `internal/adapters/server/mcpapi` (92 tests), `internal/adapters/server/httpapi` (56 tests), `cmd/till` (226 tests). All green. 1205 tests passing collectively — no silent cross-package breakage.
- **Attack 7 (acceptance-regex gap — broader `WorkItem`-prefix enumeration).** `rg '\bWorkItem[A-Z]'` remaining hits are all on the preserved list from Attack 4 (EmbeddingSubjectTypeWorkItem, EmbeddingSearchTargetTypeWorkItem, buildWorkItemEmbeddingContent, IsValidWorkItemAppliesTo, validWorkItemAppliesTo) plus `bridgeLegacyActionItemsToWorkItems` (dies in unit 1.7 per plan §Scope bullet 7). No unclassified leftover top-level symbols. Infix-WorkItem symbols logged as F1/F2.
- **Attack 8 (orphan-collapse drift — `WorkKindSubtask/Phase/Decision/Note`).** All four renamed consistently to `KindSubtask/KindPhase/KindDecision/KindNote` at `internal/domain/workitem.go:40-43`. `rg WorkKind -i` = 0 across the non-workflow tree — both declarations and every reference reached by the sweep. Plan §Scope bullet 48 classified these as "naturally unreachable" post-drops-rewrite.sql; the rename itself is still in-scope because the constants appear in the rename regex `WorkKind\b`, which the acceptance gate forces to 0.
- **Attack 9 (worklog factuality).** BUILDER_WORKLOG.md §"Unit 1.1 — Round 1" states 31 Go source files touched. `git diff --name-only` shows exactly 31 `.go` files modified + 2 MD files (PLAN.md + BUILDER_WORKLOG.md itself). Per-pass file count claims check out: e.g. pass 6 `WorkKind\b → Kind` touching 21 files matches the surface that remained after passes 1-5 stripped the 5 constants from 24/15/17/7/7 files. Baseline claim "`type WorkItem ` and `WorkItemKind` had 0 baseline occurrences" verified via `git stash` + `rg` against HEAD (both returned 0).
- **Attack 10 (PLAN.md state atomicity).** `grep -n '^\*\*State:' PLAN.md`: one `building` (drop-level, line 3), one `done` (unit 1.1, line 63), fifteen `todo` (units 1.2-1.16). `git diff workflow/drop_1_75/PLAN.md` shows exactly the single state flip on line 63. No collateral state mutation.

## Hylla Feedback

None — task touched pure Go identifier-rename semantics verified via `rg` / `sd` / `mage` gates. Hylla was not the right tool for this verification (the authoritative reference is the Go compiler + test runner, which `mage build` + `mage test-pkg` exercise directly). No Hylla query was attempted and none was needed.

## Unit 1.2 — Round 1

**Verdict:** PASS

## Summary

Attempted ten targeted falsification attacks against the app-layer kind-catalog seeder deletion. Zero CONFIRMED counterexamples. All in-scope surviving references to the deleted symbols are either the three documented Unit 1.5 waiver sites (`template_library.go:126`, `template_library_builtin.go:29`, `template_library_builtin.go:79`) or unrelated domain-layer types (`KindDefinitionInput` lives in `internal/domain/kind.go` — not affected). The deleted test's property (nested phase support in seeded defaults) is preserved by a sibling test at the repo layer (`internal/adapters/storage/sqlite/repo_test.go:2333 TestRepository_SeedDefaultKindsIncludeNestedPhaseSupport`) that drives through `seedDefaultKindCatalog` in `repo.go` — coverage is not lost, only relocated to the layer that still owns the seed path. Guard-block strips are behavior-preserving modulo the bootstrap call itself. The two out-of-plan deletions (`kindBootstrapState` type + `"sync"` import) are forced by the primary deletion and strictly cleaner than leaving them; both are covered by the plan's acceptance regex anyway. No counterexample found; unit holds.

## Attacks Attempted

### A1 — Hidden callers outside the plan's scoped excludes (REFUTED)

Ran the acceptance `rg` **without** the scoped excludes:

```
rg -n 'ensureKindCatalogBootstrapped|defaultKindDefinitionInputs|kindBootstrap|kindBootstrapState' . --glob='!workflow/**' --glob='!.git/**'
```

Three hits, all in the documented waiver files:
- `internal/app/template_library.go:126`
- `internal/app/template_library_builtin.go:29`
- `internal/app/template_library_builtin.go:79`

Zero unexpected callers. PLAN.md §1.2 line 85 names exactly these three; BUILDER_WORKLOG.md Unit 1.2 Round 1 names exactly these three; both match diff reality. No counterexample.

### A2 — Stealth orphans (REFUTED)

Probed for dead helpers the deletion might have stranded:
- `defaultKindDefinitions` (without `Inputs` suffix) — `rg` returns 0 matches. Not a latent name.
- `kindDefinitionInput` (lowercase) — `rg` returns 0 matches. Not a latent name.
- `KindDefinitionInput` (exported) — 34 hits across `internal/domain/kind.go` (declaration at `:79-80`), plus test-site usages in `internal/domain/`, `internal/app/`, `internal/tui/`, `internal/adapters/**`, `cmd/till/`. All reference the **domain-layer** type, which is orthogonal to the deleted `defaultKindDefinitionInputs` app-layer function. Type remains live and load-bearing (used by `UpsertKindDefinition` at `kind_capability.go:108`).

No stealth orphans. The deleted function was the only consumer specific to the bootstrap path.

### A3 — Sync-import fallout (REFUTED)

Two sub-probes:
- `rg '\bsync\.' internal/app/kind_capability.go` → 0 matches. No surviving `sync.` consumer in the file. Import drop is correct.
- `rg '\bsync\.' internal/app/service.go` → `108: schemaCacheMu sync.RWMutex`. `"sync"` import in `service.go` is still needed and was not (and should not have been) touched. Verified `git diff internal/app/service.go` shows no import-block edit.

No other file imports `sync` via a re-export pattern — `sync` is a stdlib package with no aliasing convention in this repo (`rg '"sync"' internal/app/ --glob='*.go'` → only `service.go:<import block>` remains).

### A4 — Service field init (REFUTED)

Exhaustive search for `kindBootstrap` field init across `NewService` and every test helper:
- `rg 'kindBootstrap|KindBootstrap' . --glob='!workflow/**' --glob='!.git/**'` → 0 matches post-diff.
- `NewService` at `service.go:120` uses composite literal `&Service{...}` with named fields — `kindBootstrap` never explicitly initialized pre-diff (relied on zero-value of `kindBootstrapState`). Post-diff struct literal at `:165-186` cleanly omits the field. No init site breaks.
- 40+ `NewService(repo, ...)` call sites in test files — none pass `kindBootstrap` (it was never a constructor arg, only a struct field). No test-helper fixture touches it.

No dangling init site.

### A5 — Test-coverage drop (REFUTED — COVERAGE PRESERVED AT LOWER LAYER)

Deleted test `TestDefaultKindDefinitionInputsIncludeNestedPhaseSupport` (kind_capability_test.go:994-1017 pre-diff) asserted:
- `phase` kind exists in defaults with `AppliesTo` containing `KindAppliesToPhase`.
- `subtask` kind exists with `AllowedParentScopes` containing `KindAppliesToPhase`.

Sibling coverage at the **repo layer**: `internal/adapters/storage/sqlite/repo_test.go:2333-2354 TestRepository_SeedDefaultKindsIncludeNestedPhaseSupport` asserts:
- `phase.AppliesToScope(domain.KindAppliesToPhase)` — same property.
- `phase.AllowsParentScope(domain.KindAppliesToPhase)` — same property.

The repo-layer test drives through `seedDefaultKindCatalog` in `repo.go:1231-1301`, which independently seeds the phase kind with matching applies_to + parent_scope values (`repo.go:1244` → `parentScope: []domain.KindAppliesTo{domain.KindAppliesToBranch, domain.KindAppliesToPhase}`). This is a separate seed code path (SQLite migration-driven) from the deleted `ensureKindCatalogBootstrapped` (app-layer runtime-driven). Unit 1.3 will bake the same two kinds directly into the `CREATE TABLE kind_catalog` block and delete the repo-layer seed; at that point the repo-layer test evolves to assert the baked rows.

**Not a coverage gap.** The deleted test was redundant with the repo-layer test, and the repo-layer test survives both deletions (app-layer now, repo-layer seeder next).

One nuance: the deleted test asserted `subtask.AllowedParentScopes` includes `KindAppliesToPhase`, but the repo-layer test only asserts it for `phase`. Not a regression — the `subtask` assertion becomes moot because Unit 1.14's drops-rewrite.sql collapses `subtask` kind out of existence (per plan §1.14 F3 decision) and the Unit 1.3-baked catalog has only `project` + `actionItem`. Once 1.3 lands, `subtask` is no longer a runtime-live kind, so asserting its parent scopes would be asserting a dead branch.

### A6 — Caller-rewrite correctness (REFUTED)

Inspected all 6 stripped guard blocks for off-by-one deletion (did any accidentally remove a subsequent error return or cache read?):

- `ListKindDefinitions` at `kind_capability.go:91-103` — guard was 3 lines `if err := ...; err != nil { return nil, err }`, post-diff the function flows directly from receiver-signature into `repo.ListKindDefinitions`. No error-path or cache-access was bundled into the guard.
- `SetProjectAllowedKinds` at `:143-164` — guard was sandwiched between `GetProject` error check and `normalizeKindIDList` call. Post-diff, both surrounding blocks are intact at `:148-150` and `:151-153`. No collateral deletion.
- `resolveProjectKindDefinition` at `:546-572` — guard was at the function head. Post-diff, `kindID = domain.NormalizeKindID(kindID)` at `:548` is the new first line. Identical behavior modulo bootstrap.
- `resolveActionItemKindDefinition` at `:586-620` — same shape. Post-diff head is `kindID = domain.NormalizeKindID(kindID)` at `:588`. Clean.
- `EnsureDefaultProject` at `service.go:198-` — guard was 3 lines before `repo.ListProjects`. Post-diff, the function flows directly into `projects, err := s.repo.ListProjects(ctx, false)`. Clean.
- `CreateProjectWithMetadata` at `service.go:246-` — guard was 3 lines before `withResolvedMutationActor`. Post-diff, that call is the new first line. Clean.

No stripped guard carried additional error-return or cache-read responsibilities. The guard blocks were pure bootstrap-invocation wrappers, and removing them is behavior-preserving modulo the bootstrap itself.

### A7 — Plan-vs-diff alignment: 4 vs 6 guard blocks (REFUTED — NOT OVERREACH)

PLAN.md §1.2 line 85 says "Update every caller (`resolveProjectKindDefinition` at `:592-596` and similar) to skip the bootstrap call." The plan's Paths field (line 78) explicitly includes `internal/app/service.go` and notes "remove `kindBootstrap` struct field if declared there." The 2 service.go guard-block strips are consequences of removing the method — you cannot delete `ensureKindCatalogBootstrapped` from `kind_capability.go` while leaving callers in `service.go`; the compile would fail even within scope.

`git show HEAD:internal/app/service.go | rg ensureKindCatalogBootstrapped` confirms exactly 2 pre-diff hits (`:201`, `:253`) in service.go, and the diff deletes exactly those. No overreach. The "4 callers" mention in the plan is listing callers in `kind_capability.go` specifically; service.go strips are covered by the plan's Paths field and the "similar" wording. No deviation from plan intent.

### A8 — Angles the proof twin structurally cannot catch (REFUTED)

Proof twin (BUILDER_QA_PROOF.md Unit 1.2 Round 1) verified evidence completeness and reasoning coherence. Angles proof cannot attack directly:
- **Silent coverage migration** — A5 addresses this (property lives at repo layer, not lost).
- **Field-init dead-store** — A4 addresses this (no constructor sites break).
- **Import-fallout cascade** — A3 addresses this (sync is stdlib, no re-export).
- **Struct-size ABI drift** — not applicable to Go (no stable ABI, no external consumers of `Service` struct layout).
- **Reflection/interface assertion** — `rg 'reflect\.|TypeOf.*kindBootstrap|interface\{\}.*kindBootstrap'` → 0 matches. `kindBootstrapState` was never reflected on. A10 confirms.

No angle the proof twin missed.

### A9 — Waiver abuse: does `mage build ./internal/app` hide defects beyond the 3 documented template_library sites? (REFUTED)

The plan's waiver rationale: the 3 template_library.go sites keep `internal/app` compile-broken until Unit 1.5 deletes those files wholesale. Post-diff, would `mage build ./internal/app` surface anything beyond those 3?

- **Undeclared references:** `rg 'ensureKindCatalogBootstrapped|defaultKindDefinitionInputs|kindBootstrap|kindBootstrapState' . --glob='!workflow/**'` returns exactly the 3 documented sites. Nothing else.
- **Unused-import diagnostic:** `go vet` would flag a truly-unused `"sync"` import. `sync` was dropped from `kind_capability.go` but retained in `service.go` (still used). No dangling import.
- **Unused-variable diagnostic:** no surviving references to `kindBootstrap` field or `kindBootstrapState` type — both gone cleanly.
- **Interface-compliance drift:** `Service` has no interface assertions (`var _ XXX = (*Service)(nil)`) that would break when a field disappears. `rg 'var _ .* = \(\*Service\)' internal/app/` → 0 matches.

The only compile failures `mage build ./internal/app` would surface are the 3 documented waiver sites. The waiver is honest — not hiding latent defects.

### A10 — kindBootstrapState orphan removal safety (REFUTED)

`kindBootstrapState` (pre-diff at `kind_capability.go:85-89`) was defined as:
```
type kindBootstrapState struct {
    once sync.Once
    err  error
}
```

Attack surfaces for orphan type removal:
- **Reflection / runtime type access:** `rg 'reflect\.|TypeOf|\.String\(\)|"kindBootstrapState"'` across the tree → no hits referencing the type name. Not reflected upon.
- **Interface embedding:** struct had no methods; no interface could have required it. `rg 'kindBootstrapState\b'` → 0 post-diff.
- **Generic constraint:** Go generics didn't parameterize on this type (`rg '\[.*kindBootstrapState.*\]'` → 0).
- **Exported alias / type assertion:** lowercase, package-private, no external consumer.
- **Test-helper mock:** no test file constructed or asserted on this type.

Safe to delete. The deletion is actually forced — once the `kindBootstrap` field on `Service` is removed (plan-mandated at line 85 "`sync.Once` struct field `kindBootstrap` (declared on `Service`)"), the struct type becomes dead code. Leaving it would produce a `declared and not used` diagnostic from stricter linters.

## F-Findings (Falsification Findings)

- **None.** Ten attack attempts, zero CONFIRMED counterexamples. All attacks REFUTED with concrete evidence.

## Classification

- No findings to classify. Unit 1.2 is **PASS** — no findings block, no findings to defer, no findings editorial-only.

**Do not block Unit 1.2.** Proceed to Unit 1.3.

## Hylla Feedback

N/A — this falsification pass verified a pure app-layer symbol deletion via `git diff HEAD` + `rg` over committed Go source. Hylla is stale for files edited after the last ingest (per project CLAUDE.md rule #2), so the authoritative evidence for this unit is the diff itself plus post-diff `rg` sweeps. No Hylla query was attempted and none was needed — the deletion's blast radius is lexical (who-references-these-identifiers), not semantic (what-does-this-function-mean). `rg` is the right tool shape. No miss to record.

## Unit 1.3 — Round 1

**Verdict:** PASS

## Summary

Ran 13 targeted attacks against the `projects.kind` strip + `kind_catalog` bake + seeder deletion + scope-expansion claim. Zero CONFIRMED counterexamples. Fresh rerun of `mage test-pkg ./internal/adapters/storage/sqlite` is still green (69 pass / 1 skip). The INSERT statements are schema-complete (10 columns match 10 values), the `ensureGlobalAuthProject` INSERT is well-formed (5 placeholders = 5 args, 8 columns = 5 placeholders + 3 inline literals), the baked rows sort correctly (`actionItem` before `project` in ASCII), the `pragma_table_info` assertion uses exact-match (`c == "kind"`) against SQLite's declared-lowercase name, the stripped guards in `template_library*.go` had no behavior beyond bootstrap gating, and every listed attack surface routes either to "mitigated here" or "owned by a later unit." The one pre-existing DB concern (orphaned `kind` column from a past `ALTER TABLE` run) is harmless because NOT NULL is satisfied by the declared DEFAULT and no code path reads it; `scripts/drops-rewrite.sql` owns the physical column drop per §1.14. Scope-expansion is the minimum reachable fix — it's physically impossible to run `mage test-pkg ./internal/adapters/storage/sqlite` with `internal/app` in a compile-broken state because sqlite imports app in 8 files.

## Attacks Attempted

### A1 — Migration round-trip on pre-existing DB with `kind` column (REFUTED)

Scenario: user has pre-1.3 `tillsyn.db` where `ALTER TABLE projects ADD COLUMN kind TEXT NOT NULL DEFAULT 'project'` already ran. Post-1.3 code opens it.

- `CREATE TABLE IF NOT EXISTS projects` — no-op (table exists).
- `ALTER TABLE projects ADD COLUMN metadata_json` at `repo.go:593` still fires under `isDuplicateColumnErr` guard — safe idempotent.
- The physical `kind` column remains. No code path reads it: post-1.3 SELECTs in `CreateProject` / `UpdateProject` / `GetProject` / `ListProjects` / `scanProject` / `ensureGlobalAuthProject` all omit it (verified via direct read of `repo.go:1249-1363` and `:3865-3889`).
- `CreateProject` INSERT: `INSERT INTO projects(id, slug, name, description, metadata_json, created_at, updated_at, archived_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)` — 8 columns, 8 placeholders. When the table also physically has `kind TEXT NOT NULL DEFAULT 'project'`, SQLite fills it from the column DEFAULT. NOT NULL satisfied. Insert succeeds.
- Orphaned column stays as dead weight until `scripts/drops-rewrite.sql` §1.14 runs `ALTER TABLE projects DROP COLUMN kind`.

Not a defect. Documented path.

### A2 — Pre-existing DB with 7 legacy `kind_catalog` rows (REFUTED — OUT OF SCOPE)

Scenario: user has pre-collapse DB with rows `{project, actionItem, subtask, phase, branch, decision, note}` (7 rows from old seeder).

- `CREATE TABLE IF NOT EXISTS kind_catalog` — no-op.
- Baked `INSERT OR IGNORE INTO kind_catalog` for `project` + `actionItem` — IGNORED (rows already present).
- Result: user has 7 rows, not 2. `TestRepositoryFreshOpenKindCatalog` is a fresh-open test (`OpenInMemory()`) so it only exercises the baked-two-row path.

Handled by §1.14 — `DELETE FROM kind_catalog WHERE id NOT IN ('project', 'actionItem')` collapses the set. No code in Unit 1.3 requires exactly 2 rows at runtime (only the test does, and the test only uses in-memory). Not a Unit 1.3 defect.

### A3 — INSERT OR IGNORE re-fire semantics (REFUTED)

`migrate()` runs on every `OpenRepository` call. The `stmts` slice contains `CREATE TABLE IF NOT EXISTS kind_catalog` immediately followed by two `INSERT OR IGNORE INTO kind_catalog` statements. Every DB open re-runs them.

- First open: `CREATE TABLE` creates table, INSERTs add rows with `strftime('%Y-%m-%dT%H:%M:%fZ', 'now')` timestamps.
- Subsequent opens: `CREATE TABLE IF NOT EXISTS` → no-op. `INSERT OR IGNORE` → primary-key conflict on `id='project'` / `id='actionItem'` → row ignored, existing row untouched. Timestamps stay frozen to first-open.

Behavior is idempotent and timestamp-stable. Not a defect.

### A4 — Row-ordering in `TestRepositoryFreshOpenKindCatalog` (REFUTED)

Test at `repo_test.go:2567-2605` uses `SELECT id FROM kind_catalog ORDER BY id` and expects `want := []string{"actionItem", "project"}`. ASCII: `'a' (0x61) < 'p' (0x70)`, so `actionItem` sorts before `project`. `ORDER BY id` in SQLite on TEXT columns uses the BINARY collation by default — strict byte order. Test expectation matches actual sort order regardless of INSERT order (builder inserted project first, actionItem second).

### A5 — `pragma_table_info` case-sensitivity / normalization (REFUTED)

Test at `repo_test.go:2636-2640` does `if c == "kind"`. SQLite's `pragma_table_info('projects')` returns column names as declared in the CREATE TABLE statement. The column was declared lowercase `kind TEXT NOT NULL DEFAULT 'project'`. SQLite preserves declared case. No case-folding, no Unicode normalization — pure byte comparison via Go's `==` on `string`. The assertion would still catch a hypothetical `KIND` or `Kind` column declaration. There is also a `len(columns) == 0` guard at `:2641-2643` blocking the table-missing false-pass. Assertion is robust.

### A6 — Scope-expansion guard strip: runtime-behavior change? (REFUTED)

Read post-strip `template_library.go:124-128` + `template_library_builtin.go:27-31, 74-81`. Each post-strip block proceeds directly from the receiver signature into the method body. The pre-strip guard was uniformly `if err := s.ensureKindCatalogBootstrapped(ctx); err != nil { return ..., err }` — pure invocation wrapper. The method did ONE thing: lazily seed kind_catalog rows. Post-1.3, those rows live in DDL. So the guard is functionally equivalent to a no-op for the post-baked state. No runtime-behavior change. Post-strip files compile (`mage test-pkg ./internal/adapters/storage/sqlite` transitively compiled all of `internal/app`).

### A7 — `mage ci` waiver-discharge claim (REFUTED — CORRECTLY SCOPED)

Worklog Deviation #3: "Unit 1.2's waiver is functionally discharged ahead of schedule." The claim that is actually being made:

- `internal/app` now compiles (3 guard-sites stripped, no dangling `ensureKindCatalogBootstrapped` callers).
- This implicitly satisfies Unit 1.2's `mage test-pkg ./internal/app` compile precondition.
- Builder explicitly did NOT claim `mage ci` runs green — that gate is 1.5's charge (restoring workspace compile) and 1.15's charge (whole-drop verification).

Claim is scoped to "the transitive compile through sqlite works" (proven by gate 8 passing). Not an overclaim. No counterexample.

### A8 — `kind_catalog` INSERT schema-column constraint satisfaction (REFUTED)

`kind_catalog` DDL at `repo.go:315-326` has 10 columns:
1. `id TEXT PRIMARY KEY`
2. `display_name TEXT NOT NULL`
3. `description_markdown TEXT NOT NULL DEFAULT ''`
4. `applies_to_json TEXT NOT NULL DEFAULT '[]'`
5. `allowed_parent_scopes_json TEXT NOT NULL DEFAULT '[]'`
6. `payload_schema_json TEXT NOT NULL DEFAULT ''`
7. `template_json TEXT NOT NULL DEFAULT '{}'`
8. `created_at TEXT NOT NULL`
9. `updated_at TEXT NOT NULL`
10. `archived_at TEXT` (nullable)

Baked INSERT at `:327-332` (project) and `:333-338` (actionItem) list 10 columns and 10 VALUES each. Manually mapped:
- `id`: 'project' / 'actionItem' — NOT NULL satisfied.
- `display_name`: 'Project' / 'ActionItem' — NOT NULL satisfied.
- `description_markdown`: 'Built-in project kind' / 'Built-in actionItem kind' — NOT NULL.
- `applies_to_json`: '["project"]' / '["actionItem"]' — NOT NULL.
- `allowed_parent_scopes_json`: '[]' — NOT NULL.
- `payload_schema_json`: '' — NOT NULL (empty string is NOT NULL).
- `template_json`: '{}' — NOT NULL.
- `created_at` / `updated_at`: `strftime('%Y-%m-%dT%H:%M:%fZ', 'now')` — NOT NULL.
- `archived_at`: NULL — nullable, fine.

All NOT NULL constraints satisfied. No silent suppression via `OR IGNORE`. Verified by the passing `TestRepositoryFreshOpenKindCatalog` test (insert succeeded, row retrievable).

### A9 — Seeder-caller side effects (REFUTED)

Pre-diff `repo.go:671-673` (via `git show HEAD`) shows the seeder caller was a single 3-line `if err := r.seedDefaultKindCatalog(ctx); err != nil { return err }`. No adjacent side effects packaged with it. Immediate context:

```
bridgeLegacyActionItemsToWorkItems()
seedDefaultKindCatalog()      ← deleted
ensureGlobalAuthProject()
migrateFailedColumn()
```

The surrounding migration calls are independent. Deletion removes only the seeder invocation — `ensureGlobalAuthProject` still runs, `bridgeLegacyActionItemsToWorkItems` still runs, `migrateFailedColumn` still runs. No collateral migration step deleted.

### A10 — Test-site false-green from deleting `TestRepository_SeedDefaultKindsIncludeNestedPhaseSupport` (REFUTED)

Test asserted that the seeder produced a `phase` kind with nested-phase parent-scope support. Both the seeder AND the `phase` kind are gone post-1.3. The assertion is literally unsatisfiable — there is no `phase` row to query. Preserving the test would require either (a) re-seeding the legacy kinds (reverts the collapse), or (b) rewriting the assertion against a non-existent row (guaranteed fail). Builder's deletion is the only coherent response. The "phase" domain concept itself is planned for domain-layer deletion in §1.9. No functional regression — the property being tested has no meaningful referent post-collapse.

### A11 — `ensureGlobalAuthProject` INSERT arity (REFUTED)

Read `repo.go:1347-1363`. SQL:
```
INSERT INTO projects(id, slug, name, description, metadata_json, created_at, updated_at, archived_at)
VALUES (?, ?, ?, '', '{}', ?, ?, NULL)
ON CONFLICT(id) DO NOTHING
```

- 8 columns.
- 5 `?` placeholders + 3 inline literals ('', '{}', NULL) = 8 values total.
- 5 Go args passed: `AuthRequestGlobalProjectID`, `globalAuthProjectSlug`, `globalAuthProjectName`, `globalAuthProjectCreatedAt`, `globalAuthProjectCreatedAt`.

Arity matches (5 placeholders = 5 args). Column-to-value mapping:
- `id` ← `AuthRequestGlobalProjectID` (TEXT)
- `slug` ← `globalAuthProjectSlug` (TEXT)
- `name` ← `globalAuthProjectName` (TEXT)
- `description` ← '' (TEXT)
- `metadata_json` ← '{}' (TEXT)
- `created_at` ← `globalAuthProjectCreatedAt` (TEXT)
- `updated_at` ← `globalAuthProjectCreatedAt` (TEXT)
- `archived_at` ← NULL (TEXT nullable)

All types align. `ON CONFLICT(id) DO NOTHING` handles the self-healing repeat-open case. Valid INSERT.

### A12 — `scripts/drops-rewrite.sql:230` stale `projects.kind` ref (REFUTED — OWNED BY §1.14)

Line 230 is `(SELECT COUNT(*) FROM projects WHERE kind <> 'project')` inside an assertion block. This script is the CURRENT pre-rewrite version — §1.14 replaces the entire file wholesale. The file is outside Unit 1.3's declared paths (`repo.go` + `repo_test.go` only). There is no execution path in Drop 1.75 where the current (stale) script runs against a post-1.3 schema:

- Drop phase order: units 1.1-1.13 (Go) → unit 1.14 (rewrite SQL) → unit 1.15 (mage ci + push) → dev applies NEW rewrite to real DB.
- Current dev DB still physically has `kind` column (pre-1.3 state) — the stale script would run fine against it IF it ran today, but no workflow step calls for that.

Classified EDITORIAL-ONLY / DEFER-TO-UNIT-1.14. Matches proof twin's informational-only classification.

### A13 — Regex-bleed alternate forms (REFUTED)

Reran with tighter bounds that proof twin might have missed:

- `rg -U 'SELECT[^;]*\bkind\b[^;]*FROM projects\b' repo.go` → 0 matches (the one greedy-bleed match from gate 5 requires `FROM tasks` as the FROM clause to intersect with `t.kind` and hit "FROM" bookends).
- `rg -U 'INSERT INTO projects\([^)]*\bkind\b[^)]*\)' repo.go` (word-boundary on `kind`) → 0 matches. Confirms no `INSERT INTO projects(...kind...)` residue.
- `rg -U 'UPDATE projects[^;]*\bkind\b\s*=' repo.go` → 0 matches. Confirms no `UPDATE projects ... kind = ...` residue.
- Same 3 regexes on `kindRaw|NormalizeKindID\(p\.Kind\)|p\.Kind\s*=` — still only `scanAttentionItem` matches (lines 4290, 4306, 4329). `scanAttentionItem` scans the `AttentionKind` domain, not `projects.kind`.
- Case-insensitive probe: `rg -i 'projects\.kind|p\.kind\b'` across `internal/` (excluding tests, workflow) → 0 matches. No overlooked casing.

Every tighter regex variant that might catch a real residue returns zero. Only the builder-documented false-positives remain.

## F-Findings (Falsification Findings)

- **None.** 13 attacks attempted, 0 CONFIRMED counterexamples. All attacks REFUTED with concrete evidence.

## Classification

- No F-findings to classify. Unit 1.3 is **PASS**.

**Do not block Unit 1.3.** Proceed to Unit 1.4.

## Hylla Feedback

None — Hylla answered everything needed. This falsification pass verified schema/SQL/test-site strips and INSERT arity via `git diff HEAD` + `Read` + `Grep` over committed Go source, plus a rerun of `mage test-pkg ./internal/adapters/storage/sqlite` for gate 8 fresh-eyes verification. Hylla is stale for files edited after the last ingest (project CLAUDE.md rule #2) — all Unit 1.3 edits are post-ingest, so lexical tools + `git show HEAD:...` for pre-diff context are the authoritative evidence. No Hylla query was attempted; none was needed. The falsification shape here is "does the diff produce any counterexample state," not "where else in committed code does X appear" — the former is diff-bound, the latter is Hylla's sweet spot. No miss to record.

## Unit 1.4 — Round 1

**Verdict:** PASS-WITH-FINDINGS

## Summary

Ran 15 targeted attacks against the Unit 1.4 domain-layer template excision. Zero CONFIRMED blocking counterexamples. Byte-compared `canonicalizeActionItemToken` between pre-delete `template_library.go:274-300` and current `kind.go:183-209` — character-for-character identical (doc comment, signature, consts, control flow, return). `kind.go` required no new imports (the function only uses `strings.Builder` / `strings.Contains`, both already covered by the file's existing `"strings"` import). Fresh rerun of `mage test-pkg ./internal/domain` = 49/49 pass, matching the pre-`-race` count and the pre-unit 52 − 3 (deleted `template_library_test.go` `Test*` funcs) expectation exactly. Waiver scope is intact: every residual template-symbol caller outside `internal/domain/` appears in PLAN.md §1.5's explicit Paths list. Three non-blocking findings surfaced — two EDITORIAL-ONLY (proof twin text inaccuracy; worklog "5 files" wording), one DEFER-TO-LATER-UNIT (`cmd/till/project_cli_test.go` references dead-after-1.4 `domain.TemplateLibraryScope*` / `TemplateActorKind*` consts but is listed in PLAN §1.6 Paths, not §1.5 — §1.5 planner needs to add it or §1.5's `mage ci` restoration gate fails).

## Attacks Attempted

### A1 — Byte-identity of relocated `canonicalizeActionItemToken` (REFUTED)

Dumped `git show HEAD:internal/domain/template_library.go` to `/tmp/old_tl.go`, then character-compared the 31-line `canonicalizeActionItemToken` block:

- Pre (HEAD `template_library.go:270-300`): doc comment (4 lines), `func canonicalizeActionItemToken(lowered string) string {` signature, `const (token = "actionitem", canonical = "actionItem")`, `if !strings.Contains(...)` early-return, `var b strings.Builder`, `b.Grow(len(lowered))`, `i := 0`, `for i < len(lowered) {` loop with inner `lowered[i:i+len(token)] == token` check, `leftOK`/`rightOK` boundary logic (`i == 0 || lowered[i-1] == '-' || lowered[i-1] == '_'`), `b.WriteString(canonical)` / `b.WriteByte(lowered[i])` branches, `return b.String()`.
- Post (`internal/domain/kind.go:179-209`): identical 31 lines — same doc comment, signature, consts, loop structure, branch predicates, return.

No semantic change, no whitespace drift, no subtle re-indent. Byte-identical. `rg -c '^func canonicalizeActionItemToken' internal/domain/*.go` → `kind.go:1` (sole declaration).

### A2 — Doc-comment / signature / receiver-status drift (REFUTED)

- **Doc comment:** preserved verbatim — same 4-line block describing token rewriting + boundary semantics.
- **Signature:** `func canonicalizeActionItemToken(lowered string) string` — identical parameter name, parameter type, return type.
- **Export status:** lowercase `c` in both locations → unexported in both → no accidental export during move.
- **No method conversion:** still a free function, not a method on any type (original wasn't either — it's a package-private helper).

### A3 — Import fallout from relocation (REFUTED)

Enumerated symbols used by `canonicalizeActionItemToken`: `strings.Builder`, `strings.Contains`. Both live in stdlib `strings`. `kind.go`'s pre-diff import block (lines 3-10) already includes `"strings"` (used by `NormalizeKindID` / `NormalizeKindAppliesTo` etc.). No new imports needed; diff shows 0 import-block edits in `kind.go`. Old `template_library.go` imported `bytes, crypto/sha256, encoding/hex, encoding/json, fmt, slices, sort, strings, time` — none of these were referenced by `canonicalizeActionItemToken`, so their absence from `kind.go` is irrelevant to the relocated function.

### A4 — Error-sentinel preservation correctness (REFUTED)

Preserved sentinel: `ErrInvalidKindTemplate` (line 25). Verified it's still referenced **inside** `internal/domain/kind.go` at 7 call sites (lines 262, 265, 271, 274, 281, 288, 296) inside `normalizeKindTemplate` and related validators — preservation is grounded. The 8 deleted sentinels (`ErrInvalidTemplateLibrary`, `ErrInvalidTemplateLibraryScope`, `ErrInvalidTemplateStatus`, `ErrInvalidTemplateActorKind`, `ErrInvalidTemplateBinding`, `ErrBuiltinTemplateBootstrapRequired`, `ErrTemplateLibraryNotFound`, `ErrNodeContractForbidden`) have zero remaining references inside `internal/domain/` (verified `rg` → 0 matches). Residual references exist in `internal/app/**`, `internal/adapters/server/**` — all 13 files appear in PLAN §1.5 Paths (`template_library.go`, `template_library_builtin.go`, `template_library_test.go`, `template_contract.go`, `template_contract_test.go`, `template_reapply.go`, `template_library_builtin_spec.go`, `app_service_adapter.go`, `handler.go` [mcpapi + httpapi], `handler_test.go`, `mcp_surface.go`, `app_service_adapter_helpers_test.go`). Every residual caller dies wholesale or has its branch stripped in §1.5.

Sweep against remaining `errors.go`: no other `Err*` was removed. The 8 deleted lines (via `git diff`) match the 8 worklog entries exactly — no over-deletion.

### A5 — Test-file deletion helper/fixture leakage (REFUTED)

Read `git show HEAD:internal/domain/template_library_test.go` — file contained exactly 3 `func Test*` definitions:
- `TestNewTemplateLibraryNormalizesNestedRules`
- `TestNewTemplateLibraryRejectsDuplicateScopeKind`
- `TestNewNodeContractSnapshotDefaultsActorKinds`

No `TestMain`, no helper functions, no package-level vars, no shared fixtures. File was self-contained. Wholesale `git rm` is safe — no sibling `*_test.go` in `internal/domain` referenced symbols from this file. `rg '^func TestMain' internal/domain/` → 0 matches (no TestMain anywhere in the domain package).

### A6 — `builtin_template_library.go` data references (REFUTED)

Read `git show HEAD:internal/domain/builtin_template_library.go` — contained 3 types: `BuiltinTemplateLibraryState`, `BuiltinTemplateLibraryStatus`, plus one status const block. No hardcoded template data (that lives in `internal/app/template_library_builtin_spec.go` + `internal/app/embedded/*.json`, not in the domain file). No registry append pattern. `rg 'BuiltinTemplateLibrary' internal/domain/` → 0 matches post-delete. Safe wholesale deletion.

### A7 — `NodeContractSnapshot` method-signature residue (REFUTED)

`rg 'NodeContractSnapshot' internal/domain/` → 0 matches. No remaining domain type still has a method parameter or return type referencing `NodeContractSnapshot`. The struct lived in `template_library.go` (verified at HEAD lines ~400+) with its own methods; all died with the file. Gate 1 `rg 'TemplateLibrary|TemplateReapply|NodeContractSnapshot|BuiltinTemplate' internal/domain/` returned 0.

### A8 — Type aliases / re-exports dangling (REFUTED)

Searched for `type X = Y` alias patterns in `internal/domain/` at HEAD that might have aliased any deleted type. `git show HEAD:internal/domain/*.go | rg 'type \w+ = '` — no hits on deleted types. No `= TemplateLibrary`, `= NodeTemplate`, `= NodeContractSnapshot`, or `= TemplateActorKind` aliases anywhere in the domain package. Nothing dangling.

### A9 — Const/var blocks in deleted files used elsewhere (REFUTED — ALL WITHIN §1.5 WAIVER)

Enumerated const/var blocks in deleted `template_library.go` from HEAD:
- `TemplateLibraryScope*` consts (3: Global, Project, Draft)
- `TemplateLibraryStatus*` consts (3: Draft, Approved, Archived)
- `ProjectTemplateBindingDrift*` consts (3: Current, UpdateAvailable, LibraryMissing)
- `TemplateActorKind*` consts (5: Human, Orchestrator, Builder, QA, Research)
- `validTemplateLibraryScopes`, `validTemplateLibraryStatuses`, `validTemplateActorKinds` vars (3)

`rg 'TemplateLibraryScope|TemplateLibraryStatus|ProjectTemplateBindingDrift|TemplateActorKind' drop/1.75/**/*.go -l` → 27 files. Cross-checked every one against PLAN §1.5 Paths:
- `internal/app/*` template files: all in §1.5 delete list.
- `internal/app/{snapshot.go, snapshot_test.go, service_test.go, helper_coverage_test.go}`: all in §1.5 Paths (strip surfaces).
- `internal/adapters/storage/sqlite/{repo.go, template_library_test.go}`: in §1.5 Paths.
- `internal/adapters/server/common/{mcp_surface.go, app_service_adapter.go, app_service_adapter_mcp.go, app_service_adapter_lifecycle_test.go}`: in §1.5 Paths.
- `internal/adapters/server/mcpapi/{extended_tools.go, extended_tools_test.go, instructions_explainer.go}`: in §1.5 Paths.
- `internal/tui/{model.go, model_test.go}`: in §1.5 Paths.
- `cmd/till/{template_cli.go, template_builtin_cli_test.go, main.go, main_test.go}`: in §1.5 Paths.

**Exception flagged** (becomes F3 below): `cmd/till/project_cli_test.go` references `domain.TemplateLibraryScopeGlobal` (`:180`), `domain.TemplateLibraryStatusApproved` (`:183`), `domain.TemplateActorKindBuilder` (`:205`), `domain.TemplateActorKind` (`:206`), `domain.TemplateActorKindHuman` (`:207`). This file is listed in PLAN §1.6 Paths, not §1.5 — §1.6 strips `Project.Kind`, not template types. §1.5's `mage ci` restoration acceptance bullet would fail at this file unless §1.5's planner adds it to the strip list OR §1.5 gets a targeted scope expansion. Not a §1.4 defect, but a latent §1.5 planning gap.

### A10 — `-race` re-run (REFUTED)

Ran `mage testFunc ./internal/domain TestNormalizeKindID` (mage's race-enabled test-function target) → package compiles + tests run green, but 0 matching tests found (no `TestNormalizeKindID` exists in the package — see F2 below). Separately, `mage testPkg ./internal/domain` → 49/49 pass cleanly, 0.00s real time. Race detector does not hide anything for this unit — the excision is file-deletion + 31-line function move with zero goroutine/channel/shared-state surface. Safe.

### A11 — Waiver-scope discipline (PARTIAL — FLAGS F3)

Sweep of every non-domain file still referencing deleted types: all route into PLAN §1.5 Paths except one — `cmd/till/project_cli_test.go` (see A9). Listed under F3 as DEFER-TO-LATER-UNIT (§1.5 planning gap, not §1.4 defect).

`rg 'ErrInvalidTemplateLibrary|ErrInvalidTemplateLibraryScope|ErrInvalidTemplateStatus|ErrInvalidTemplateActorKind|ErrInvalidTemplateBinding|ErrBuiltinTemplateBootstrapRequired|ErrTemplateLibraryNotFound|ErrNodeContractForbidden' drop/1.75/**/*.go -l` → 13 files, every one in §1.5 Paths. Error-sentinel waiver is clean.

### A12 — Test-count delta sanity (REFUTED)

Unit 1.1 Round 1 reports 52 tests passed in `./internal/domain` pre-`template_library_test.go` deletion. Deleted file had 3 `func Test*`. Post-1.4 count: 52 − 3 = 49. Observed: 49. Exact match.

Note: `rg '^func Test' internal/domain/*_test.go` returns 41 top-level Test declarations across 6 files, but the test runner count of 49 includes subtests via `t.Run(...)`, which is why 41 < 49. Ratio is stable across the 1.4 diff — the deleted file added no subtests either (all 3 were flat `Test*`), so 49 = 52 − 3 checks out arithmetically.

### A13 — `mage test-pkg ./internal/domain` fresh rerun (REFUTED)

Ran `mage testPkg ./internal/domain` from `/Users/evanschultz/Documents/Code/hylla/tillsyn/drop/1.75/`:
```
[PKG PASS] github.com/evanmschultz/tillsyn/internal/domain (0.00s)
  tests: 49, passed: 49, failed: 0, skipped: 0, packages: 1
```
Exit 0. Matches builder/proof-twin claim exactly.

### A14 — Orphan-via-collapse discipline (REFUTED)

Per `feedback_orphan_via_collapse_defer_refinement.md`: catalog/enum collapse should leave orphan-downstream vocabulary alone, deferring cleanup to a refinement drop. Unit 1.4 does NOT violate this — it excises **type definitions and their error sentinels** that are wholly dead in the post-collapse world (template-library subsystem is going away entirely, not being replaced). This is NOT orphan-via-collapse; it's direct subsystem excision. The orphan-via-collapse rule applies to things like `KindSubtask/KindPhase/KindDecision/KindNote` consts which remain declared (see Unit 1.1 F2). Unit 1.4 is structurally different — it removes an entire feature surface.

No dead code accidentally left behind inside `internal/domain/`: `rg -i 'template|nodecontract|builtin.*template' internal/domain/` → matches only `KindTemplate`, `KindTemplateChildSpec`, `normalizeKindTemplate`, `ErrInvalidKindTemplate` — all intentionally preserved per PLAN §1.4 F5 classification (naturally unreachable post-drops-rewrite but kept until a dedicated refinement drop).

### A15 — Proof twin blind spots (PARTIAL — FLAGS F2)

Proof twin (BUILDER_QA_PROOF.md Unit 1.4 Round 1) verified byte-identity, gate outcomes, file-deletion presence, and stealth-orphan absence. It did NOT verify one claim it made: "Test `TestNormalizeKindID` exercises this code path per the package's standing `domain_test.go` coverage."

Direct check: `rg 'TestNormalizeKindID|^func TestNormalize' internal/domain/` returns only `TestNormalizeHandoffListFilter`, `TestNormalizeAttentionListFilter`, `TestNormalizeCommentTarget`, `TestNormalizeCommentTargetSupportsHierarchyNodes`. **No `TestNormalizeKindID` exists.** `rg 'actionitem|canonicalize' internal/domain/*_test.go` → 0 matches. `canonicalizeActionItemToken` has zero direct unit tests; `NormalizeKindID` has zero direct unit tests. The coverage it relies on is transitive (other domain tests that construct kinds/projects hit `NormalizeKindID` via `project.SetKind`, `KindDefinition.Input` path) and those transitive tests did pass. So the relocated helper is exercised, but proof twin's specific "TestNormalizeKindID" citation is false.

This is a pre-existing test-coverage gap, NOT introduced by Unit 1.4. The relocation doesn't change it. EDITORIAL-ONLY against the proof twin text.

## F-Findings (Falsification Findings)

### F1 — Worklog "5 files" wording imprecise

- **Severity:** EDITORIAL-ONLY
- **Where:** BUILDER_WORKLOG.md Unit 1.4 Round 1, "Files touched: 5 files in `internal/domain` (4 deleted, 1 edited, 1 relocation-repair into `kind.go`)."
- **What:** The count resolves to 6 distinct file paths (4 deletions + `errors.go` modification + `kind.go` modification) but the prose reads "5 files ... 1 edited, 1 relocation-repair" which conflates `errors.go` edit with `kind.go` relocation. `git status --porcelain -- internal/domain/` shows 6 entries (4 D + 2 M).
- **Counterexample status:** REFUTED as a blocker — the underlying work is correct, only the file-count wording is loose. Proof twin flagged the same cosmetic issue (BUILDER_QA_PROOF.md Unit 1.4 "Informational" line 221).
- **Fix:** reword to "6 files in `internal/domain` (4 deleted, 2 edited: `errors.go` sentinel strip + `kind.go` relocation-repair)." Do not block Unit 1.4.

### F2 — Proof twin claim of `TestNormalizeKindID` coverage is inaccurate

- **Severity:** EDITORIAL-ONLY (against proof-twin text, not against builder work)
- **Where:** BUILDER_QA_PROOF.md Unit 1.4 Round 1, under "Relocation Soundness Check", final bullet: "Test `TestNormalizeKindID` exercises this code path per the package's standing `domain_test.go` coverage."
- **What:** No test named `TestNormalizeKindID` exists in `internal/domain/`. Verified via `rg 'TestNormalizeKindID'` → 0 hits. The proof twin's phrasing should be "transitively exercised by kind/project construction tests" rather than citing a named test. `NormalizeKindID` and `canonicalizeActionItemToken` have no direct unit tests.
- **Counterexample status:** REFUTED as a blocker — the relocated function IS compiled + transitively exercised (49/49 pass proves it compiles; construction paths in `kind_test.go` / `project.go` invocations of `NormalizeKindID` exercise it indirectly). The coverage gap is pre-existing, not introduced here.
- **Fix:** Either (a) correct the proof-twin text in a follow-up, OR (b) add a direct `TestNormalizeKindID` / `TestCanonicalizeActionItemToken` in a refinement drop. Don't block Unit 1.4.

### F3 — `cmd/till/project_cli_test.go` has template-type references but is NOT in PLAN §1.5 Paths

- **Severity:** DEFER-TO-LATER-UNIT (§1.5 planning gap)
- **Where:** `cmd/till/project_cli_test.go:180, 183, 205, 206, 207` — references to `domain.TemplateLibraryScopeGlobal`, `domain.TemplateLibraryStatusApproved`, `domain.TemplateActorKindBuilder`, `domain.TemplateActorKind`, `domain.TemplateActorKindHuman`.
- **What:** PLAN §1.5 Paths lists `cmd/till/template_cli.go`, `cmd/till/template_builtin_cli_test.go`, `cmd/till/main.go`, `cmd/till/main_test.go` for cmd/till. It does NOT list `cmd/till/project_cli_test.go`. PLAN §1.6 lists `cmd/till/project_cli_test.go` (for `project.Kind` stripping). But the template types die in §1.4 + §1.5, not §1.6 — meaning `project_cli_test.go` will reference dead `domain.TemplateLibraryScope*` / `TemplateActorKind*` symbols at §1.5's "`mage ci` restoration" acceptance gate. §1.5 will either fail its own acceptance or need a targeted scope-expansion edit. (Parallel shape to Unit 1.3's own documented 9-line scope expansion into `template_library*.go`.)
- **Counterexample status:** NOT a Unit 1.4 blocker — Unit 1.4 explicitly waives `mage build` / `mage ci`. The workspace is expected to be compile-broken through §1.5. This finding is aimed at §1.5's planner.
- **Fix:** §1.5 planner should add `cmd/till/project_cli_test.go` to §1.5 Paths (template-surface strip only; `project.Kind` strip remains in §1.6). OR, if the §1.5 planner prefers, surface this as a scope-expansion deviation during §1.5 execution (matches the 1.3 pattern). Route to ORCHESTRATOR-ATTENTION so the drop-orch updates §1.5 Paths before spawning the §1.5 builder.

## Classification

| Finding | Classification | Blocks 1.4? |
| --- | --- | --- |
| F1 | EDITORIAL-ONLY | No |
| F2 | EDITORIAL-ONLY | No |
| F3 | DEFER-TO-LATER-UNIT + ORCHESTRATOR-ATTENTION | No |

**Do not block Unit 1.4.** All three findings are non-blocking. F3 needs the orchestrator to flag §1.5's Paths for update before the §1.5 builder spawn — propagate via a drop-orch comment on the §1.5 unit description or a scope-expansion note, whichever matches the current MD-workflow convention.

## Hylla Feedback

N/A — task touched non-Go content (PLAN.md, worklog MDs) and Go files inside the Unit-1.1–1.3 edit window where Hylla is stale per project CLAUDE.md rule #2 ("Changed since last ingest: use `git diff`"). Verification evidence was `git diff HEAD` + `git show HEAD:<path>` + `Read` + `Grep` + a fresh `mage testPkg` rerun. No Hylla query was attempted and none was needed — byte-identity verification, sentinel-removal verification, and waiver-scope cross-reference are all lexical / diff-bound operations that Hylla can't out-perform on stale-since-ingest files. Recording "None — Hylla answered everything needed" as the closing stance.

## Unit 1.5 — Round 1 — QA Falsification

**Verdict:** PASS-WITH-ACCEPTED-RISK (plus 2 non-blocking findings routed to `§1.5-related-cleanup` follow-up; plus 1 DEFER-TO-LATER-UNIT finding aimed at §1.11/§1.12/§1.13 planners)

## Summary

Ran 11 targeted attacks against the Unit 1.5 template-library / node-contract-snapshot excision across 7 packages. Zero CONFIRMED blocking counterexamples against the unit's stated objective ("workspace-compile restoration burden — the 1.4 waiver discharge"). `mage build` succeeds, `mage ci` produces 0 build errors across 20 packages, the three §1.5 acceptance regexes (Gate 1/2/3) all return 0 matches on Go files, all 21 remaining test failures classify cleanly into §1.11 / §1.12 Paths as documented. Two non-blocking findings surfaced: (F1) nine dead string literals in `internal/adapters/server/common/app_service_adapter_mcp.go` + `internal/adapters/server/mcpapi/instructions_tool.go` + `internal/tui/model.go` advertising removed template-library functionality — not caught by Gate 1 regex because they use `template-library` (dash) not `template_librar` (underscore); these are live MCP/TUI introspection surfaces, not doc files. (F2) one previously-flagged scope-expansion into `cmd/till/project_cli_test.go` (already documented in Round 1 §1.4 F3) — builder applied the minimum-necessary strip cleanly, no further action needed. One DEFER observation (F3): the §1.11/§1.12/§1.13 planners must handle the 21 routed test failures as already Paths'd — orchestrator attention not needed.

## Attacks Attempted

### A1 — Gate 2 test-failure mis-routing (REFUTED)

Cross-referenced every one of the 21 test failures surfaced by fresh `mage ci` rerun against §1.11 / §1.12 Paths:

**§1.11 Paths** (`internal/app/kind_capability_test.go`, `internal/app/service_test.go`, `internal/app/snapshot_test.go`, `internal/app/helper_coverage_test.go`):
- kind_capability_test.go: 7 failures (TestServiceSetAndListProjectAllowedKindsValidation, TestServiceEnforceMutationGuardBranches, TestCreateActionItemAppliesKindTemplateActions, TestCreateProjectAppliesKindTemplateDefaultsAndChildren, TestCreateActionItemCascadesChildKindTemplateDefaults, TestCreateActionItemRejectsRecursiveTemplateBeforePersistence, TestIssueCapabilityLeaseParentDelegationPolicy) — ALL in Paths. ✓
- service_test.go: 5 failures (TestIssueCapabilityLeaseDistinctIdentitiesBranchScope, TestScopedLeaseAllowsLineageMutations, TestScopedLeaseRejectsSiblingMutations, TestCreateActionItemKindPayloadValidation, TestReparentActionItemRejectsCycle) — ALL in Paths. ✓
- snapshot_test.go: 1 failure (TestExportSnapshotIncludesExpectedData) — in Paths. ✓

**§1.12 Paths** (`internal/adapters/storage/sqlite/*_test.go`, `internal/adapters/server/{mcpapi,httpapi,common}/*_test.go`):
- mcpapi/handler_integration_test.go: 2 failures (TestHandlerUpdateHandoffResolvesApprovedPathContext, TestHandlerUpdateHandoffOutOfScopeApprovedPathDenied) — in Paths. ✓
- httpapi/handler_integration_test.go: 2 failures (TestHandlerResolveAttentionItemApprovedPath, TestHandlerResolveAttentionItemOutOfScopeApprovedPathDenied) — in Paths. ✓
- common/app_service_adapter_auth_context_test.go: 3 failures (TestAppServiceAdapterAuthorizeMutationApprovedPathLookupBackedResources, TestAppServiceAdapterAuthorizeMutationApprovedPathPolicySplit, TestAppServiceAdapterAuthorizeMutationApprovedPathExplicitScopeResources) — in Paths. ✓
- common/app_service_adapter_lifecycle_test.go: 1 failure (TestAppServiceAdapterProjectActionItemCommentLifecycle) — in Paths. ✓

Total: 7+5+1+2+2+3+1 = 21. Classification tight. Worklog's routing table matches fresh `mage ci` output exactly. No counterexample — every failure is owned by a later unit's Paths.

### A2 — `help.go` compat-docstring restore smells like hedging (REFUTED — FLAG IS LIVE)

Traced `--template-json` end-to-end post-§1.5:

- `cmd/till/main.go:850` — `kindUpsertCmd.Flags().StringVar(&kindUpsertOpts.templateJSON, "template-json", "", "Optional kind template JSON")`.
- `cmd/till/main.go:851` — `mustMarkFlagHidden(kindUpsertCmd, "template-json")`.
- `cmd/till/main.go:2581` — `template, err := parseOptionalKindTemplateJSON(opts.templateJSON)` — flag value is consumed.
- `cmd/till/main.go:2593` — `Template: template` — passed into `UpsertKindDefinition`.
- `cmd/till/main.go:2926-2937` — `parseOptionalKindTemplateJSON` unmarshals raw JSON into `domain.KindTemplate`; `mage build` confirms `domain.KindTemplate` still exists (§1.4 F5 preservation).

The hidden flag is functional end-to-end. `TestRunSubcommandHelp/kind_upsert` at `cmd/till/main_test.go:623-625` asserts `"compatibility-only"` appears in the help output. `TestRunKindAndAllowlistCommands` at `main_test.go:1812` literally exercises `--template-json "{}"`. Builder's help-text restore is necessary to keep these tests passing — not hedging, not a bug-in-disguise. The docstring is truthful.

### A3 — `service_test.go` `newFakeRepo` seeding masks bugs (REFUTED)

Tracked every remaining caller of the deleted `ensureKindCatalogBootstrapped`:

- `git grep -n "ensureKindCatalogBootstrapped" -- internal/ cmd/` → 1 hit, a comment-only reference in `internal/app/service_test.go:40` explaining why the seed block exists.
- Zero production callers.

The seeding replaces the EFFECT of the deleted bootstrap (kind_catalog has `{project, actionItem}` at test time) not a CALL. The real sqlite repo seeds via `CREATE TABLE kind_catalog` + inline `INSERT OR IGNORE` (§1.3) at every DB open — orthogonal code path. The fake-repo seeding is the in-memory equivalent; no production code path is bypassed or masked.

Seed block at `service_test.go:45-76` constructs minimal `domain.KindDefinition` records with `AppliesTo: [project]` / `AppliesTo: [actionItem]`, `CreatedAt: time.Now().UTC()`, `UpdatedAt: time.Now().UTC()` — matches post-collapse catalog shape exactly. No counterexample.

### A4 — Scope-expansion into `cmd/till/project_cli_test.go` (REFUTED — MINIMUM NECESSARY)

`git diff c84d281 -- cmd/till/project_cli_test.go`:
- Deleted `TestRunProjectCreateUsesTemplateLibrary` function (~100 lines) that constructed `app.UpsertTemplateLibraryInput` with `domain.TemplateLibraryScopeGlobal`, `domain.TemplateLibraryStatusApproved`, `domain.TemplateActorKindBuilder`, `domain.TemplateActorKind`, `domain.TemplateActorKindHuman` — all types removed in §1.4.
- Removed 5 unused imports that the deleted test was the sole consumer of: `context`, `path/filepath`, `internal/adapters/storage/sqlite`, `internal/config`, `github.com/google/uuid`.

Verified no other test in `project_cli_test.go` uses these imports — Go's unused-import error is a hard build failure, so stripping them is forced by the test deletion. This is a single indivisible unit of work. Already previewed by §1.4 Round 1 F3 (F3 routed as DEFER-TO-LATER-UNIT to §1.5 planner OR scope-expansion). Builder took the scope-expansion path — matches the §1.3 pattern. No over-reach.

### A5 — Deletion completeness sweep (REFUTED for symbols; FLAGS F1 for string literals)

Exhaustive case-insensitive grep across the repo excluding workflow + drops-rewrite.sql:

**Code-identifier sweep (§1.5 Gate 1 regex):** `rg 'TemplateLibrary|TemplateReapply|NodeContractSnapshot|BuiltinTemplate|node_contract_snapshot|template_librar|template_node_template|template_child_rule|project_template_binding' --glob='*.go'` → 0 matches. Gate 1 clean. ✓

**Extended symbol sweep:** `rg 'TemplateLibrary|NodeContractSnapshot|ProjectTemplateBinding|TemplateActorKind' --glob='*.go' --glob='!workflow/**' --glob='!scripts/drops-rewrite.sql'` → 0 matches.

**Dash-form string-literal sweep** (NOT covered by Gate 1 regex — `template_librar` with underscore does not match `template-library` with dash): `rg -nI 'template-library\|"template library"' --glob='*.go'` found:

- `internal/adapters/server/common/app_service_adapter_mcp.go:33` — `"Kind catalog plus template-library-driven generated follow-up work and node-contract snapshots"` — live `WhatTillsynIs` capability string served via MCP introspection.
- `internal/adapters/server/mcpapi/instructions_tool.go:330, 358, 359, 385, 386, 401` — six instances across `AGENTS.md` / `README.md` / `SKILL.md` advice blocks advertising "template-library workflows", "template-library changes", "template-library examples", "template-library-driven features". Served as MCP `till.instructions` tool output.
- `internal/tui/model.go:16851, 16861` — hardcoded help-text strings inside `modeAddProject` / `modeEditProject` describing a "template library field opens the approved-library picker" UI that was removed (per §1.5 F7 confirming no TemplateLibrary readbacks).

Total: 9 dead string literals across 3 live user-facing surfaces. Classified as F1 below.

### A6 — `go.mod` diagnostics false positive (REFUTED — ORCHESTRATOR SPAWN PROMPT WAS WRONG)

Spawn prompt claimed "New go.mod diagnostics post-builder-return: `github.com/charmbracelet/x/exp/golden` missing, `golang.org/x/exp` missing; chroma / glamour / douceur unused."

Evidence:
- `git diff c84d281 -- go.mod` → EMPTY. go.mod unchanged.
- `git grep -nI "charmbracelet/glamour"` → 12 hits across `internal/tui/markdown_renderer.go`, `file_viewer_mode.go`, `file_viewer_renderer.go`, `model.go`. Glamour is heavily used by TUI — not orphaned.
- `git grep -nI "charmbracelet/x/exp/golden"` → 3 hits in `third_party/teatest_v2/teatest.go:16` + `go.mod`/`go.sum`. Used by teatest.
- `mage ci` completed `gofumpt` + `go vet` + `go test -cover` with 0 build errors sitewide.

The spawn prompt's diagnostic was a gopls-cache artifact, not a real build defect. No counterexample against the builder.

### A7 — §1.4 waiver discharge verification (REFUTED — WORKSPACE COMPILES)

Ran `mage build` → `[SUCCESS] Built till from ./cmd/till`. Then re-ran `mage ci` → "0 build errors across 20 packages" in the test summary (only 21 test failures, all in §1.11/§1.12 Paths per A1).

Sample package-level verification (via `mage ci` coverage pass): all 20 packages produced `[PKG PASS]` for compile or `[PKG FAIL]` with coverage-% (not build-error) detail. Every package compiles. §1.4 waiver DISCHARGED.

### A8 — YAGNI / over-deletion of §1.6+ dependencies (REFUTED)

Verified symbols §1.6+ needs still exist:
- `resolveProjectKindDefinition` — still at `kind_capability.go:547`. §1.6 Paths: "strip `resolveProjectKindDefinition` + callers" — §1.6 will delete it, needs it present first.
- `resolveActionItemKindDefinition` — still at `kind_capability.go:587`. Used by `CreateActionItem`.
- `initializeProjectAllowedKinds` — still at `kind_capability.go:711`, `library *domain.TemplateLibrary` param correctly stripped (per §1.5 F2 Round-6 directive).
- `templateDerivedProjectAllowedKindIDs` — DELETED. Per §1.5 F2 directive: "if the functions become trivial after the parameter removal, delete them and their callers instead of keeping stub shells". Correct disposition.
- `domain.KindTemplate` — PRESERVED (§1.4 F5). Still referenced at `cmd/till/main.go:2927, 2932`. ✓
- `ErrInvalidKindTemplate` — PRESERVED (§1.4 F5). Still at `internal/domain/errors.go:25` with 7 call-sites in `kind.go`. ✓

No over-deletion. §1.6+ prerequisites intact.

### A9 — Orphan-via-collapse classification (REFUTED)

Per `feedback_orphan_via_collapse_defer_refinement.md`: catalog/enum collapse should leave orphan-downstream vocabularies alone, deferring cleanup to refinement. Unit 1.5 does NOT violate — it excises an entire feature surface (template_libraries + node_contract_snapshots), not vocabulary rows. This is direct subsystem excision, same pattern as §1.4.

Zero dead code left inside `internal/app` / `internal/adapters/server/*` under deleted type names — final `git grep -n "TemplateLibrary|NodeContractSnapshot|ProjectTemplateBinding|TemplateActorKind" --glob='*.go'` returns 0 matches. The 9 dash-form string literals (F1) are user-facing copy, not code; they describe features that no longer exist, but the code itself doesn't reference the deleted types.

### A10 — `ensureActionItemCompletionBlockersClear` relocation is a semantic-preserving collapse (REFUTED)

`git diff c84d281 -- internal/app/mutation_guard.go` shows NEW functions `currentMutationActorType` + `ensureActionItemCompletionBlockersClear` added to this file. These were NOT in §1.5 Paths for `mutation_guard.go`.

Baseline location: `git grep -n "^func currentMutationActorType" c84d281 -- internal/app/` → `c84d281:internal/app/template_contract.go:22`. Same for `ensureActionItemCompletionBlockersClear` at `template_contract.go:74`.

The functions were defined in `template_contract.go` (deleted wholesale per §1.5 Paths) but CALLED from `service.go` at 9 call sites. Deleting `template_contract.go` wholesale without relocating them would break compile. Forced relocation, same pattern as §1.4's `canonicalizeActionItemToken` relocation (template_library.go → kind.go).

**However, the relocated `ensureActionItemCompletionBlockersClear` is NOT byte-identical to the original** — the original (template_contract.go:74-125, 51 lines) walked children + descendants, loaded `nodeContractSnapshotForActionItem` per child, checked `RequiredForParentDone` + `RequiredForContainingDone` via `appendBlocker`, then finally ran `CompletionCriteriaUnmet`. The new version (mutation_guard.go:22-47, 25 lines) keeps ONLY the `CompletionCriteriaUnmet` tail.

This collapse is SEMANTICALLY CORRECT: `nodeContractSnapshotForActionItem` depends on `NodeContractSnapshot` which was deleted. The node-contract enforcement path has no data to operate on post-§1.4. The remaining behavior (completion_criteria on active non-archived children) is the only enforcement that survives the template excision. Caller semantics at `service.go:667` (inside `MoveActionItem` → `StateDone` transition) preserves completion-blocker enforcement at the state-transition grain, just without the dead node-contract branch.

Not documented in the worklog, but not a bug. Recorded under F2 below so §1.11 planner knows the behavior-surface changed.

### A11 — `cli_render.go` + `mutation_guard.go` + `change_event.go` + ports.go scope creep (REFUTED — FORCED BY DELETIONS)

Non-Paths files touched in diff:
- `cmd/till/cli_render.go` — deleted 9 `case "template.library.*"` + `case "template.builtin.*"` + `case "template.project.*"` + `case "template.contract.*"` branches from `commandProgressLabel`. All reference removed MCP tool operations. Forced by CLI surface removal.
- `internal/app/mutation_guard.go` — relocation target for `currentMutationActorType` + `ensureActionItemCompletionBlockersClear` (per A10) + deletion of `withInternalTemplateMutation` + `internalTemplateMutationAllowed` + `internalTemplateMutationContextKey` (context helpers only used by deleted template_contract code paths). Forced by template_contract.go deletion.
- `internal/domain/change_event.go` — gofumpt whitespace realignment only (struct field alignment). No semantic change. Byproduct of `mage format` on adjacent edits.
- `internal/app/ports.go` — IS in §1.5 Paths (confirmed).
- `internal/adapters/server/common/app_service_adapter.go` — IS in §1.5 Paths (confirmed).

Scope-creep into cli_render.go + mutation_guard.go + change_event.go is forced-by-deletion. None of it is discretionary. No counterexample.

## F-Findings (Falsification Findings)

### F1 — Dead string literals advertising removed template-library functionality

- **Severity:** EDITORIAL / DEFER-TO-REFINEMENT-DROP (non-blocking for §1.5)
- **Where:**
  - `internal/adapters/server/common/app_service_adapter_mcp.go:33` — live MCP `WhatTillsynIs` capability string mentions "template-library-driven generated follow-up work and node-contract snapshots".
  - `internal/adapters/server/mcpapi/instructions_tool.go:330, 358, 359, 385, 386, 401` — six instances in `till.instructions` tool output advertising "template library" / "template-library" / "template-bound" workflows.
  - `internal/tui/model.go:16851, 16861` — two hardcoded help strings in `modeAddProject` / `modeEditProject` describing removed "template library field" UI.
- **Counterexample status:** NOT a §1.5 blocker — §1.5 Gate 1 acceptance regex uses `template_librar` (underscore) which does not match `template-library` (dash). Worklog explicitly scoped non-Go-code copy to "doc-scrub is a different drop per the 'pure-collapse, no doc churn' scope boundary." However, these are LIVE Go-source string literals in `internal/` not MD docs — they're served back to orchestrators through MCP introspection and to users through TUI help text. Calling them "docs" stretches the scope-boundary rule.
- **Proposed fix:** (a) Strip the 9 string literals in a follow-up refinement drop (`drop 2+` vocabulary scrub), OR (b) add them to §1.11 / §1.12 / §1.13 Paths explicitly during the planner's re-review (these surfaces are in `internal/adapters/server/common`, `internal/adapters/server/mcpapi`, `internal/tui` — already in §1.11/§1.12/§1.13 packages). Recommend (b) — they're in Paths already; just add the string-strip task.
- **Do not block §1.5.** Gate 1 acceptance is met against its literal wording.

### F2 — `ensureActionItemCompletionBlockersClear` relocation is not byte-identical to baseline

- **Severity:** EDITORIAL / UNDOCUMENTED-BUT-CORRECT (non-blocking)
- **Where:** Original at `template_contract.go:74-125` (51 lines, walks children + descendants, loads node-contract snapshots, enforces `RequiredForParentDone` / `RequiredForContainingDone`, appends blockers via `appendBlocker`, finally runs `CompletionCriteriaUnmet`). New at `mutation_guard.go:22-47` (25 lines, enforces `CompletionCriteriaUnmet` on active non-archived children only).
- **What:** Builder worklog describes this as part of the "template-contract deletion" but does not flag that the function's INTERNAL BEHAVIOR collapsed (not just the relocation). The relocated form is semantically correct — `NodeContractSnapshot` is deleted, so the node-contract enforcement branch has no data to operate on — but the worklog's Sub-Pass E (§1.5 Round 1) mentions only "stripped template service fields + bindings" without flagging this specific behavior collapse.
- **Counterexample status:** NOT a bug — the collapse is forced by §1.4's `NodeContractSnapshot` type deletion. The completion-criteria enforcement path at `service.go:667` (MoveActionItem → StateDone) still fires; only the dead node-contract branch is gone.
- **Proposed fix:** Add one sentence to Sub-Pass E in BUILDER_WORKLOG.md Unit 1.5 Round 1: "`ensureActionItemCompletionBlockersClear` retains only its `CompletionCriteriaUnmet` tail; node-contract enforcement branch removed alongside `NodeContractSnapshot` type deletion."
- **Do not block §1.5.** Behavior-preserving collapse; worklog text improvement only.

### F3 — Test failures route cleanly to §1.11 / §1.12 / §1.13 Paths (INFORMATIONAL)

- **Severity:** INFORMATIONAL (no action needed for §1.5)
- **What:** All 21 test failures are in files already listed in §1.11 / §1.12 / §1.13 Paths. No orchestrator rework needed; the planner's Paths decomposition correctly anticipated the test-site work. This finding is recorded so §1.11/§1.12/§1.13 builders know their Paths contain live test breakage expected to be resolved by their unit.
- **Counterexample status:** None — this is the plan working as designed.
- **Proposed fix:** None. Proceed to §1.6 or §1.11 per orchestrator decision.

## Classification

| Finding | Classification                          | Blocks 1.5? |
| ------- | --------------------------------------- | ----------- |
| F1      | EDITORIAL / DEFER-TO-REFINEMENT-DROP    | No          |
| F2      | EDITORIAL / UNDOCUMENTED-BUT-CORRECT    | No          |
| F3      | INFORMATIONAL                           | No          |

**Do not block Unit 1.5.** All three findings are non-blocking. §1.5's stated objective — discharge §1.4's workspace-compile waiver via `mage ci` success — is met: 0 build errors, 20/20 packages compile, three Gate regexes return 0 matches on Go files. The 21 remaining test failures are correctly owned by §1.11 / §1.12 / §1.13 per their existing Paths lists.

Accepted-risk: F1's nine dead string literals remain in live MCP/TUI surfaces. §1.5's acceptance wording ("pure-collapse, no doc churn") reads ambiguously against live-Go-source introspection strings; builder's interpretation (strip type/symbol refs only, leave string copy for a later vocabulary scrub) is defensible but creates a minor user-facing lie-in-docstring surface until a refinement drop. Flagged for the orchestrator to decide whether to fold into §1.11/§1.12/§1.13 Paths or defer to a dedicated vocabulary drop.

## Hylla Feedback

None — Hylla answered everything needed. This falsification pass verified template-library excision via `git diff c84d281` + `Read` + `Grep` over committed + uncommitted Go source, plus a fresh `mage ci` rerun for gate 4 / failure-routing verification. Hylla is stale for files edited after the last ingest (project CLAUDE.md §Code Understanding Rules rule 2) — every touched file in §1.5 is post-c84d281 so lexical tools + `git show c84d281:<path>` for pre-diff context are the authoritative evidence. No Hylla query was issued; none was needed. The falsification shape here is "does the post-deletion state leave any symbol/string residue that breaks the gate or advertises removed functionality," not "where else in committed code does X appear" — the former is diff-bound, the latter is Hylla's sweet spot. No miss to record.
