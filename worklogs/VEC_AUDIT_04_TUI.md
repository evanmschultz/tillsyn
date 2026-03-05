# Lane QA-AUDIT-4: TUI Metadata + Search UX Audit
Timestamp: 2026-03-04T00:35:44Z

## Commands Run and Outcomes
- `sed -n '110,190p' VECTOR_SEARCH_EXECUTION_PLAN.md`
  - Outcome: confirmed Section 7 metadata requirements and Wave E acceptance targets.
- `sed -n '32,74p' VECTOR_SEARCH_EXECUTION_PLAN.md`
  - Outcome: confirmed search pagination contract (`limit=50` default, `limit` max `200`, `offset=0` default).
- `rg -n "objective|acceptance_criteria|validation_plan|blocked_reason|risk_notes|SearchTaskMatches|defaultSearchResultsLimit|dependency" internal/tui/model.go internal/tui/model_test.go`
  - Outcome: mapped implementation and test coverage anchors.
- `nl -ba internal/tui/model.go | sed -n '96,130p'`, `... '180,210p'`, `... '2460,2538p'`, `... '3296,3355p'`, `... '12398,12548p'`, `... '1836,2020p'`, `... '4430,4515p'`
  - Outcome: collected exact line refs for editable/viewable fields, search forwarding, and dependency inspector behavior.
- `nl -ba internal/tui/model_test.go | sed -n '770,822p'`, `... '8330,8360p'`, `... '8414,8435p'`
  - Outcome: confirmed current assertions for search forwarding and dependency inspector limits.
- `just test-pkg ./internal/tui`
  - Outcome: `ok   github.com/hylla/tillsyn/internal/tui	(cached)`.

## Findings by Severity

### High
1. Missing high-risk automated coverage for 4/5 Wave-E metadata fields (`objective`, `acceptance_criteria`, `validation_plan`, `risk_notes`).
- Edit/persist/render implementations exist:
  - `internal/tui/model.go:2477`
  - `internal/tui/model.go:2478`
  - `internal/tui/model.go:2479`
  - `internal/tui/model.go:2480`
  - `internal/tui/model.go:3315`
  - `internal/tui/model.go:3324`
  - `internal/tui/model.go:3333`
  - `internal/tui/model.go:3342`
  - `internal/tui/model.go:12531`
  - `internal/tui/model.go:12532`
  - `internal/tui/model.go:12533`
  - `internal/tui/model.go:12534`
- In current TUI tests, metadata visibility assertions only cover dependency metadata + `blocked_reason`:
  - `internal/tui/model_test.go:8341`
  - `internal/tui/model_test.go:8344`
  - `internal/tui/model_test.go:8347`
- QA risk: regressions for 4 metadata fields can pass current suite.

### Medium
1. Search default-limit contract drift vs plan.
- Plan default: `limit=50` (max `200`):
  - `VECTOR_SEARCH_EXECUTION_PLAN.md:61`
  - `VECTOR_SEARCH_EXECUTION_PLAN.md:64`
- Code default is `200` and forwarded in primary search paths:
  - `internal/tui/model.go:186`
  - `internal/tui/model.go:1856`
  - `internal/tui/model.go:2013`
- Tests enforce code constant, not plan default:
  - `internal/tui/model_test.go:809`
  - `internal/tui/model_test.go:810`

### Low
1. Dependency-inspector forwarding shape is less explicit than primary search paths.
- Inspector forwards query/scope/archive/states/limit only:
  - `internal/tui/model.go:4447`
  - `internal/tui/model.go:4453`
- Primary search paths explicitly set mode/sort/limit/offset and broader filters:
  - `internal/tui/model.go:1844`
  - `internal/tui/model.go:1854`
  - `internal/tui/model.go:1855`
  - `internal/tui/model.go:1857`
  - `internal/tui/model.go:2001`
  - `internal/tui/model.go:2011`
  - `internal/tui/model.go:2012`
  - `internal/tui/model.go:2014`
- Current test for inspector path confirms limit, not explicit offset/mode/sort semantics:
  - `internal/tui/model_test.go:8423`

## Completeness Checklist vs Plan (Pass/Fail)
- Section 7 / Checkpoint 1 (editable fields): **PASS**
  - Evidence: `internal/tui/model.go:112`, `internal/tui/model.go:113`, `internal/tui/model.go:114`, `internal/tui/model.go:115`, `internal/tui/model.go:116`, `internal/tui/model.go:2476`, `internal/tui/model.go:2477`, `internal/tui/model.go:2478`, `internal/tui/model.go:2479`, `internal/tui/model.go:2480`.
- Section 7 / Checkpoint 2 (viewable in task-info): **PASS**
  - Evidence: `internal/tui/model.go:12468`, `internal/tui/model.go:12531`, `internal/tui/model.go:12532`, `internal/tui/model.go:12533`, `internal/tui/model.go:12534`.
- Section 7 / Checkpoint 3 (DRY/reuse patterns): **PASS**
  - Evidence: `internal/tui/model.go:103`, `internal/tui/model.go:3036`, `internal/tui/model.go:3301`, `internal/tui/model.go:13271`, `internal/tui/model.go:12522`.
- Checkpoint 4 (search limits + forwarding shape): **FAIL**
  - Forwarding is explicit in main search paths, but default limit differs from plan (`200` vs planned `50`).
- Checkpoint 5 (dependency inspector behavior/limit): **PASS (with caveat)**
  - Evidence: `internal/tui/model.go:4453`, `internal/tui/model.go:4501`, `internal/tui/model.go:12898`, `internal/tui/model_test.go:8423`.
- Checkpoint 6 (missing high-risk test coverage): **FAIL**
  - No direct assertions found for `objective`, `acceptance_criteria`, `validation_plan`, `risk_notes` in TUI tests.

## Residual Risks
- Wave-E metadata functionality is implemented but under-tested for four metadata fields.
- Search default-limit behavior currently deviates from documented contract.
- Dependency inspector behavior depends on narrower forwarding assumptions than primary search paths.

## Explicit Final Verdict
**FAIL for QA completeness sign-off.**
Core Wave-E behavior appears implemented, but unresolved high/medium QA risks remain (metadata test gaps and search default-limit contract drift).
