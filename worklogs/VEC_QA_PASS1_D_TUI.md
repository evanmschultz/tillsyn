# QA1-D: Independent QA Pass 1 (TUI metadata/search/dependency)

Date: 2026-03-03  
Lane: `QA1-D`  
Scope checked: `internal/tui/**`, `VECTOR_SEARCH_EXECUTION_PLAN.md`, `.tmp/vec-wavef-evidence/20260303_175936/**`  
Write scope used: this file only.

## Context7 Usage

- Not used in this pass.
- Reason: no external framework/library behavior claims were required; conclusions are based on repository code/tests and local evidence artifacts.

## Required Check Results

1. Metadata fields (`objective`, `acceptance_criteria`, `validation_plan`, `risk_notes`) view/edit + tests:
   - `PASS` for implemented behavior and direct test coverage.
2. Dependency inspector explicit forwarding (`mode`, `sort`, `limit`, `offset`, default `levels`):
   - `PASS`.
3. Pagination/sorting defaults and plan-vs-implementation drift:
   - `PARTIAL` (implementation explicit; one documented plan/default drift remains).

## Findings By Severity

### Medium

1. Plan-vs-implementation drift on default search `limit`.
   - Plan says default `limit=50` (`VECTOR_SEARCH_EXECUTION_PLAN.md:59-62`).
   - TUI implementation default is `defaultSearchResultsLimit = 200` (`internal/tui/model.go:185-186`), used in dependency inspector search (`internal/tui/model.go:4456`) and other TUI search requests (`internal/tui/model.go:1856`, `internal/tui/model.go:2013`).
   - Impact: contract/documentation mismatch for default pagination behavior.

### Low

1. Coverage is strong for edit/view, but create-path assertion for these four metadata fields is not directly present.
   - Edit + clear semantics covered in `TestModelEditTaskMetadataFieldsPrefillAndSubmit` (`internal/tui/model_test.go:1506-1567`).
   - View semantics covered in `TestModelTaskInfoShowsStructuredMetadataSections` (`internal/tui/model_test.go:1462-1503`).
   - No direct add-task submit assertion found for the same four fields in this pass.

### High

- None.

## Evidence (File:Line)

- Metadata fields present in task form schema/indexes:
  - `internal/tui/model.go:103-117`
  - `internal/tui/model.go:128-141`
- Metadata fields editable/prefilled in task form:
  - `internal/tui/model.go:2468-2480`
  - `internal/tui/model.go:2509-2519`
  - `internal/tui/model.go:3315-3350`
- Metadata fields viewable in task-info overlay:
  - `internal/tui/model.go:12526-12538`
- Metadata tests:
  - `internal/tui/model_test.go:1462-1503`
  - `internal/tui/model_test.go:1506-1567`
- Dependency inspector forwarding now explicit:
  - `internal/tui/model.go:4447-4457`
- Dependency inspector forwarding tests:
  - `internal/tui/model_test.go:8531-8551`
- Plan/default drift reference:
  - `VECTOR_SEARCH_EXECUTION_PLAN.md:59-62`
  - `internal/tui/model.go:185-186`

## Commands / Tests Executed

1. Repository/evidence inspection commands (selected):
   - `rg -n ... internal/tui/model.go internal/tui/model_test.go`
   - `nl -ba internal/tui/model.go | sed -n ...`
   - `nl -ba internal/tui/model_test.go | sed -n ...`
   - `nl -ba VECTOR_SEARCH_EXECUTION_PLAN.md | sed -n '1,320p'`
   - `sed -n '1,120p' .tmp/vec-wavef-evidence/20260303_175936/test_pkg_internal_tui.txt`
2. Required test command:
   - `just test-pkg ./internal/tui`
   - Outcome: `ok   github.com/hylla/tillsyn/internal/tui  (cached)`

## Verdict

- `FAIL` for QA1-D due to unresolved Medium drift (plan default `limit=50` vs implementation default `200`).

## Unresolved Risks and Exact Next Step

- Unresolved risk: default-pagination contract mismatch may cause confusion across lanes/reviewers and inconsistent expectations for search behavior.
- Exact next step:
  1. Align one source of truth by either:
     - updating `VECTOR_SEARCH_EXECUTION_PLAN.md` defaults to `limit=200`, or
     - changing TUI default limit constant/call sites to `50`.
  2. Re-run `just test-pkg ./internal/tui`.
  3. Re-run QA pass and update this QA artifact with final PASS/FAIL.
