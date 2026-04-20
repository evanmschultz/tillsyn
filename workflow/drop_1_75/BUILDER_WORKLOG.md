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

## Unit 1.2 — Round 1

**Date:** 2026-04-19
**Outcome:** success
**Files touched:** `internal/app/kind_capability.go`, `internal/app/kind_capability_test.go`, `internal/app/service.go`.

### Deletions

1. **`kind_capability.go`** — three symbol deletions:
   - `kindBootstrapState` struct type (was `kind_capability.go:85-89` — orphaned after field removal).
   - `ensureKindCatalogBootstrapped` method on `*Service` (was `:559-589`).
   - `defaultKindDefinitionInputs` function (was `:863-874`).
   - Dropped now-unused `"sync"` import (only consumer was `kindBootstrapState.once sync.Once`).

2. **`service.go`** — one field + two call-site cleanups:
   - Removed `kindBootstrap kindBootstrapState` field from `Service` struct (was `:109`).
   - Stripped guard block `if err := s.ensureKindCatalogBootstrapped(ctx); err != nil { … }` from `EnsureDefaultProject` (was `:201-203`).
   - Stripped same guard block from `CreateProjectWithMetadata` (was `:253-255`).

3. **`kind_capability_test.go`** — deleted `TestDefaultKindDefinitionInputsIncludeNestedPhaseSupport` (was `:994-1017`). Kept `"slices"` import (still used by other tests at `:63, :961`).

### Call sites updated in `kind_capability.go`

Stripped the `ensureKindCatalogBootstrapped` guard block from four in-scope callers (baked kind_catalog rows land in Unit 1.3):

- `ListKindDefinitions` (was `:99-101`)
- `SetProjectAllowedKinds` (was `:161-163`)
- `resolveProjectKindDefinition` (was `:593-595`)
- `resolveActionItemKindDefinition` (was `:636-638`)

### Intentionally skipped (per PLAN.md §1.2 "intentionally skip" clause)

Three dangling `ensureKindCatalogBootstrapped` call sites remain in files that Unit 1.5 deletes wholesale:

- `internal/app/template_library.go:126`
- `internal/app/template_library_builtin.go:29`
- `internal/app/template_library_builtin.go:79`

Editing them now would be pure churn; Unit 1.5 discharges the workspace-compile waiver.

### Acceptance gate outcome

```
rg 'ensureKindCatalogBootstrapped|defaultKindDefinitionInputs|kindBootstrap' . \
  --glob='!workflow/**' \
  --glob='!internal/app/template_library*.go' \
  --glob='!internal/app/template_contract*.go' \
  --glob='!internal/app/template_reapply.go'
```

→ 0 matches (ripgrep exit 1). **Pass.**

`mage test-pkg ./internal/app` and `mage ci` were WAIVED for this unit per PLAN.md §1.2 — the `internal/app` package is compile-broken by design between this commit and 1.5's commit. Not run.

### Deviations from plan

One additional deletion beyond the plan's three explicit targets: the `kindBootstrapState` struct type at `kind_capability.go:85-89`. It was the backing type for the `kindBootstrap` field, so it becomes dead code the moment the field dies. Deleting it here keeps the package tidy and prevents a trailing unused-type diagnostic. Also dropped the now-unused `"sync"` import from the same file for the same reason.

### Hylla Feedback

None — Hylla answered everything needed. Symbol lookup + caller enumeration used `Grep` over Go source (fast, exact) rather than Hylla vector/keyword search because I needed exhaustive exact-string hits for three identifiers; `rg` is the right tool for that shape. LSP would have worked equally well for the `kindBootstrap`-field-on-`Service` confirmation — `Grep` answered the question in the same query batch and delivered the field location plus all its usages, so I didn't round-trip through LSP. No Hylla miss to record.
