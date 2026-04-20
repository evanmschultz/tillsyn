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
