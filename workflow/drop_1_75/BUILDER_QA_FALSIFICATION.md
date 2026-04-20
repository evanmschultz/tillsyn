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
