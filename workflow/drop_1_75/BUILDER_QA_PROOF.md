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

## Unit 1.2 — Round 1

**Verdict:** PASS

## Summary

Builder's claim — delete the app-layer kind-catalog seeder (`ensureKindCatalogBootstrapped` + `defaultKindDefinitionInputs` + `kindBootstrap` field + orphaned `kindBootstrapState` type), drop the now-unused `"sync"` import from `kind_capability.go`, and strip the 6 in-scope guard-block callers (4 in `kind_capability.go` + 2 in `service.go`) — is fully supported by the evidence. The acceptance `rg` returns 0. The three dangling `template_library*.go` references are the documented Unit 1.5 waiver, not defects. `git diff HEAD` matches the worklog's "Files touched" list (3 Go files + 2 MD files) exactly.

## Confirmed Claims

- **Acceptance rg (fresh re-run):** `rg 'ensureKindCatalogBootstrapped|defaultKindDefinitionInputs|kindBootstrap' . --glob='!workflow/**' --glob='!internal/app/template_library*.go' --glob='!internal/app/template_contract*.go' --glob='!internal/app/template_reapply.go'` → 0 matches (rg exit 1). **Pass.**
- **`ensureKindCatalogBootstrapped` definition removed:** pre-diff hunk shows the full function body at `kind_capability.go:559-589` deleted, matching the worklog's `:559-589` range claim.
- **`defaultKindDefinitionInputs` removed:** pre-diff hunk shows the function body (returns 7 built-in kind inputs) deleted, matching the worklog's `:863-874` range claim (line numbers approximate — span deleted is 12 lines, close to the claimed 12-line span).
- **`kindBootstrap` field removed from `Service`:** `git diff internal/app/service.go` shows `-	kindBootstrap      kindBootstrapState` struck from the struct. Builder claim of pre-removal at `:109` matches `git show HEAD:internal/app/service.go | rg kindBootstrap` → `109:	kindBootstrap      kindBootstrapState`.
- **`kindBootstrapState` struct type removed (orphaned):** `git diff internal/app/kind_capability.go` shows the 6-line block (comment + struct with `once sync.Once`, `err error`) deleted. `rg 'kindBootstrapState' . --glob='!workflow/**' --glob='!.git/**'` → 0 matches.
- **`"sync"` import dropped from `kind_capability.go`:** diff shows `-	"sync"` removed from the import block. `rg '\bsync\.' internal/app/kind_capability.go` → 0 matches (no remaining consumers in the file). `"sync"` import intact in `service.go` because `schemaCacheMu sync.RWMutex` still uses it (verified via `rg '\bsync\.' internal/app/service.go` → `108:	schemaCacheMu      sync.RWMutex`).
- **In-scope callers stripped — `kind_capability.go` (4 sites):** `git show HEAD:internal/app/kind_capability.go | rg 'ensureKindCatalogBootstrapped' -n` pre-unit returned definition at `:559-560` + callers at `:99, :161, :593, :636`. Diff hunks show guard blocks stripped from `ListKindDefinitions` (pre-`:99-101`), `SetProjectAllowedKinds` (pre-`:161-163`), `resolveProjectKindDefinition` (pre-`:593-595`), `resolveActionItemKindDefinition` (pre-`:636-638`). All 4 guard blocks removed.
- **In-scope callers stripped — `service.go` (2 sites):** pre-unit refs at `:201, :253`. Diff hunks show guard blocks stripped from `EnsureDefaultProject` (pre-`:201-203`) and `CreateProjectWithMetadata` (pre-`:253-255`). Both callers cleaned.
- **Test deletion:** `TestDefaultKindDefinitionInputsIncludeNestedPhaseSupport` removed from `kind_capability_test.go` (diff shows 24 lines deleted at the end of the file). `rg 'TestDefaultKindDefinitionInputsIncludeNestedPhaseSupport' . --glob='!workflow/**'` → 0 matches.
- **`"slices"` import retained:** `rg 'slices\.' internal/app/kind_capability_test.go` → `:63`, `:961` — both still used. Matches worklog.
- **`service_test.go` untouched (correctly):** `git show HEAD:internal/app/service_test.go | rg 'defaultKindDefinitionInputs|kindBootstrap|ensureKindCatalogBootstrapped'` → 0 matches pre-unit. Plan listed it defensively; builder correctly identified no edits were needed. Current state also 0 matches.
- **Waiver-documented dangling refs present and exactly 3:** `rg 'ensureKindCatalogBootstrapped' internal/app/template_library.go internal/app/template_library_builtin.go -n` →
  - `internal/app/template_library.go:126`
  - `internal/app/template_library_builtin.go:29`
  - `internal/app/template_library_builtin.go:79`

  All three match the plan's "intentionally skip" clause verbatim. **Not a finding** — these die wholesale in Unit 1.5 (plan §1.5 Paths list includes both files for deletion). Confirmed per PLAN.md §1.2 waiver and task description's "Intentionally skipped (per plan, NOT defects)" bullet.
- **`mage test-pkg ./internal/app` + `mage ci` waiver honored:** worklog explicitly notes both were waived and not run. Matches plan §1.2 "mage test-pkg ./internal/app and mage ci are waived for this unit only." No violation.
- **`git diff HEAD --stat` file coherence:** 3 Go source files (`internal/app/kind_capability.go`, `internal/app/kind_capability_test.go`, `internal/app/service.go`) + 2 MD files (`workflow/drop_1_75/BUILDER_WORKLOG.md` append, `workflow/drop_1_75/PLAN.md` state flip). Matches worklog's "Files touched" list exactly.
- **PLAN.md state flip:** `git diff workflow/drop_1_75/PLAN.md` shows exactly one change: §1.2 `**State:** todo` → `**State:** done`. No drift.
- **Deviation 1 (`kindBootstrapState` orphan removal):** worklog explicitly disclosed; plan's §1.2 acceptance rg includes `kindBootstrap` as a target pattern, so the struct type (which matches `kindBootstrap*`) would have been caught by the same gate anyway. Deleting it is strictly cleaner than leaving it and is covered by the gate pattern.
- **Deviation 2 (`"sync"` import drop):** worklog explicitly disclosed; safe because the only `sync.` consumer in `kind_capability.go` was `kindBootstrapState.once sync.Once`. No remaining consumers (`rg '\bsync\.' internal/app/kind_capability.go` → 0).

## Missing Evidence

- None. All 6 plan-specified acceptance items (including the 3 waived-compile-failure items) have direct diff/rg evidence.

## P-Findings (Proof Gaps)

- None.

## Hylla Feedback

None — Hylla answered everything needed. This QA pass verified symbol deletions via `git diff HEAD` + `rg` over committed Go source (Hylla is stale for files touched after the last ingest, so `git diff` is the correct evidence source here per project CLAUDE.md rule #2). No Hylla query was attempted because the evidence demanded is lexical (has-the-identifier-disappeared-from-these-specific-files) and the diff is the authoritative record. No miss to record.
