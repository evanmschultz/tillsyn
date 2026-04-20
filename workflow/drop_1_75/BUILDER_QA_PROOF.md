# DROP_1_75_KIND_COLLAPSE — Builder QA Proof

Append a `## Unit N.M — Round K` section per QA proof pass.

## Unit 1.1 — Round 1

**Verdict:** pass

## Summary

The builder's claim — 31 Go source files renamed across 8 narrow `rg`+`sd` passes for `WorkKind → Kind` and `WorkItemID → ActionItemID`, with PLAN.md §1.1 flipped to `done` and a factual append in BUILDER_WORKLOG.md — is fully supported by the evidence. All four acceptance gates re-ran clean from scratch. All 8 pass claims match post-state identifier counts and the `git diff` file list. All 3 disclosed deviations match diff reality and are consistent with PLAN.md §Scope guidance. File-count coherence (33 = 31 Go + 2 workflow MDs) holds. No Section 0 leakage into the worklog.

## Confirmed Claims

- **Gate 1 (fresh re-run):** `rg 'WorkKind' . --glob='!workflow/**' --count-matches` → 0 matches, rg exit 1.
- **Gate 2 (fresh re-run):** `rg 'type WorkItem |WorkItemKind|WorkItemID' . --glob='!workflow/**' --count-matches` → 0 matches, rg exit 1.
- **Gate 3 (fresh re-run):** `mage build` → `[SUCCESS] Built till from ./cmd/till`, exit 0.
- **Gate 4 (fresh re-run):** `mage test-pkg ./internal/domain` → 52 tests, 52 passed, 0 failed, 0 skipped, 1 package. Exit 0.
- **Pass 1 (`WorkKindActionItem → KindActionItem`):** `rg 'KindActionItem\b' . --glob='!workflow/**' -l | wc -l` = 24. Matches builder-claimed file count.
- **Pass 2 (`WorkKindSubtask → KindSubtask`):** `rg 'KindSubtask\b' . --glob='!workflow/**' -l | wc -l` = 15. Matches claim.
- **Pass 3 (`WorkKindPhase → KindPhase`):** `rg 'KindPhase\b' . --glob='!workflow/**' -l | wc -l` = 17. Matches claim.
- **Pass 4 (`WorkKindDecision → KindDecision`):** `rg 'KindDecision\b' . --glob='!workflow/**' -l | wc -l` = 7. Matches claim.
- **Pass 5 (`WorkKindNote → KindNote`):** `rg 'KindNote\b' . --glob='!workflow/**' -l | wc -l` = 7. Matches claim.
- **Pass 6 (`WorkKind\b → Kind`):** `rg 'WorkKind' . --glob='!workflow/**'` → 0 after passes complete. Bare type is correctly in use (e.g., `KindActionItem Kind = "actionItem"` const at `internal/domain/workitem.go:39`).
- **Pass 7 (test-name rename):** `rg 'TestCommentTargetTypeForWorkKindSupportsHierarchyKinds' . --glob='!workflow/**' --count-matches` → 0 matches (rg exit 1). `rg 'TestCommentTargetTypeForKindSupportsHierarchyKinds' . --glob='!workflow/**' --count-matches` → 2 hits in `internal/tui/thread_mode_test.go`. Matches builder claim exactly.
- **Pass 8 (`WorkItemID → ActionItemID`):** `rg 'WorkItemID\b' . --glob='!workflow/**' --count-matches` → 0 (rg exit 1). All 7 builder-named files (`internal/domain/change_event.go`, `internal/adapters/storage/sqlite/{repo.go,repo_test.go}`, `internal/tui/{model.go,model_test.go}`, `internal/app/service_test.go`, `internal/adapters/server/mcpapi/extended_tools_test.go`) appear in `git diff --name-only`. `ActionItemID\b` is present in 24 files.
- **File-count coherence:** `git diff --name-only | wc -l` = 33. Breakdown: 31 `.go` files + `workflow/drop_1_75/PLAN.md` + `workflow/drop_1_75/BUILDER_WORKLOG.md`. Matches builder claim.
- **PLAN.md state flip:** `git diff workflow/drop_1_75/PLAN.md` shows exactly one substantive change: §1.1 `**State:** todo` → `**State:** done`. No other edits.
- **BUILDER_WORKLOG.md append:** factual per-pass listing, acceptance-gate outcomes, two observations. No Section 0 reasoning leaked — `rg 'Section 0|Proposal|Convergence|QA Proof|QA Falsification' workflow/drop_1_75/BUILDER_WORKLOG.md` → 0 hits (rg exit 1).
- **Deviation 1 (`isValidWorkKind → isValidKind`):** `rg 'isValidWorkKind' . --glob='!workflow/**'` → 0 matches. `rg 'isValidKind' . --glob='!workflow/**' -n` shows definition at `internal/domain/workitem.go:195-196` and single call at `internal/domain/task.go:130`. The `WorkKind\b` pass correctly caught the private helper; the only caller was renamed by the same pass. No orphaned reference.
- **Deviation 2 (baseline zero on `type WorkItem ` / `WorkItemKind`):** `git show HEAD:internal/domain/workitem.go | rg 'WorkItemKind|type WorkItem '` → 0 matches at HEAD. Builder's claim that these patterns had zero baseline pre-unit is correct — the struct was already `type ActionItem` pre-Drop-1.75 per project CLAUDE.md "Pre-Drop-1.75 Creation Rule."
- **Deviation 3 (preserved `WorkItem*` out-of-scope):** surviving `WorkItem*` symbols (`IsValidWorkItemAppliesTo`, `validWorkItemAppliesTo`, `EmbeddingSearchTargetTypeWorkItem`, `EmbeddingSubjectTypeWorkItem`, `buildWorkItemEmbeddingContent`) remain across 21 files. These are explicitly out-of-scope per PLAN.md §Scope bullet 48 ("orphan-via-collapse refactor"). Acceptance gate 2 (narrow regex `type WorkItem |WorkItemKind|WorkItemID`) does not include them by design.
- **Case-insensitive sanity:** `rg -i 'WorkKind' . --glob='!workflow/**'` → 0 matches. No residual `WorkKind` in comments, string literals, or documentation.

## Hylla Feedback

N/A — this QA pass verified a pure Go-identifier rename via `rg` / `git diff` / `mage` gates. Hylla was not consulted because the evidence demanded by the acceptance gates is lexical (ripgrep) and semantic (Go compiler via `mage build` + `mage test-pkg`). No Hylla miss to record.
