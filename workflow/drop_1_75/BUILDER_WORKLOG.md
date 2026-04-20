# DROP_1_75_KIND_COLLAPSE — Builder Worklog

Append a `## Unit N.M — Round K` section per build attempt. See `drops/WORKFLOW.md` § "Phase 4 — Build (per unit)" for what each section should contain.

<!-- First unit appended after planner decomposition + plan-QA green. -->

## Unit 1.1 — Round 1

**Date:** 2026-04-19
**Outcome:** success
**Files touched:** 31 Go source files across `internal/domain`, `internal/app`, `internal/adapters/storage/sqlite`, `internal/adapters/server/{mcpapi,common,httpapi}`, `internal/tui`, `cmd/till`.

### Per-identifier pass results

Narrow `rg -l | xargs sd` per identifier, in ordering-safe sequence:

1. `WorkKindActionItem\b` → `KindActionItem` — 24 files touched, 0 residual refs.
2. `WorkKindSubtask\b` → `KindSubtask` — 15 files touched, 0 residual refs.
3. `WorkKindPhase\b` → `KindPhase` — 17 files touched, 0 residual refs.
4. `WorkKindDecision\b` → `KindDecision` — 7 files touched, 0 residual refs.
5. `WorkKindNote\b` → `KindNote` — 7 files touched, 0 residual refs.
6. `WorkKind\b` → `Kind` (bare type, run AFTER constants so prefixes don't get corrupted) — 21 files touched, 0 residual `\bWorkKind\b` matches afterward.
7. `TestCommentTargetTypeForWorkKindSupportsHierarchyKinds` → `TestCommentTargetTypeForKindSupportsHierarchyKinds` (narrow test-name fix — the middle-of-identifier case missed by `\b` boundary; hit 2 lines in `internal/tui/thread_mode_test.go`). After this, `rg 'WorkKind' --glob='!workflow/**'` returns 0.
8. `WorkItemID\b` → `ActionItemID` — 7 files touched (`internal/domain/change_event.go`, `internal/adapters/storage/sqlite/{repo.go,repo_test.go}`, `internal/tui/{model.go,model_test.go}`, `internal/app/service_test.go`, `internal/adapters/server/mcpapi/extended_tools_test.go`), 0 residual refs.

Acceptance-gate patterns `type WorkItem ` and `WorkItemKind` had 0 baseline occurrences before the sweep — `type WorkItem ` was already removed when the struct became `type ActionItem`, and `WorkItemKind` never existed in this tree.

### Acceptance gate outcomes

- **Gate 1** `rg 'WorkKind' . --glob='!workflow/**'` → 0 matches (ripgrep exit 1).
- **Gate 2** `rg 'type WorkItem |WorkItemKind|WorkItemID' . --glob='!workflow/**'` → 0 matches (ripgrep exit 1).
- **Gate 3** `mage build` → SUCCESS. Built `till` from `./cmd/till`.
- **Gate 4** `mage test-pkg ./internal/domain` → SUCCESS. 52 tests passed, 0 failed, 0 skipped, 1 package.

### Deviations from plan

None material. Two observations:

1. The `isValidWorkKind` private function / `// isValidWorkKind` comment pair in `internal/domain/workitem.go` was caught by the `WorkKind\b` pass and now reads `isValidKind`. Consistent with the rename intent (function validates a `Kind` value) — no caller issue since the only caller `internal/domain/task.go:130` was renamed by the same pass. Called out because the "bare type" pass also renamed a private function name — no deviation from the narrow-regex contract, but worth recording.
2. Acceptance-gate-out-of-scope `WorkItem*` symbols remain in place per the plan's explicit "narrow regex only, no catch-all sweep" guidance: `IsValidWorkItemAppliesTo` + `validWorkItemAppliesTo` (domain/kind.go), `EmbeddingSearchTargetTypeWorkItem` + `EmbeddingSubjectTypeWorkItem` + `buildWorkItemEmbeddingContent` + `"work_item"` literal (app/search_embeddings.go + adapters). These are not in the acceptance regex `type WorkItem |WorkItemKind|WorkItemID` and are deferred to later refinement work. The worklog notes them so later units know they exist.

### Hylla Feedback

N/A — task touched non-Go-content semantics only incidentally; all rename decisions were driven by `rg` pattern matches + `sd` substitution + `mage` compile/test gates. No Hylla query was needed to verify call-site correctness because Go's compiler is the authoritative reference for identifier resolution, and `mage build` + `mage test-pkg ./internal/domain` confirmed zero breakage sitewide. Recording `None — Hylla answered everything needed.` as the closing stance.
