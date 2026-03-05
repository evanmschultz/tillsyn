# QA-AUDIT-3 MCP/Transport Search Surface Audit
Timestamp: 2026-03-03 17:33:36 MST

## Commands run and outcomes
1. `date '+%Y-%m-%d %H:%M:%S %Z'`
   - `2026-03-03 17:33:36 MST`
2. Read-only scope inspection:
   - `nl -ba VECTOR_SEARCH_EXECUTION_PLAN.md | sed -n '1,260p'`
   - `nl -ba internal/adapters/server/common/mcp_surface.go | sed -n '1,260p'`
   - `nl -ba internal/adapters/server/common/app_service_adapter_mcp.go | sed -n '240,340p'`
   - `nl -ba internal/adapters/server/mcpapi/extended_tools.go | sed -n '646,722p'`
   - `nl -ba internal/adapters/server/mcpapi/extended_tools_test.go | sed -n '717,905p'`
   - `rg` verification over search-tool fields/descriptions/tests in scoped files
3. Context7 checks (pre-claim protocol/library grounding):
   - `resolve-library-id`: `mark3labs/mcp-go` -> `/mark3labs/mcp-go`
   - `query-docs`: tool parameter types/options/getters (`WithNumber`, `WithInteger`, `Minimum/Maximum`, getters)
4. Scoped tests (just recipes only):
   - `just test-pkg ./internal/adapters/server/mcpapi` -> `ok   github.com/hylla/tillsyn/internal/adapters/server/mcpapi (cached)`
   - `just test-pkg ./internal/adapters/server/common` -> `?    github.com/hylla/tillsyn/internal/adapters/server/common [no test files]`

## Findings by severity
### Medium
1. Schema guardrails for pagination are description-only, not schema-enforced.
   - Evidence:
     - `internal/adapters/server/mcpapi/extended_tools.go:676`
     - `internal/adapters/server/mcpapi/extended_tools.go:677`
     - `internal/adapters/server/mcpapi/extended_tools_test.go:784`
     - `internal/adapters/server/mcpapi/extended_tools_test.go:788`
   - Detail:
     - `limit`/`offset` are declared with `mcp.WithNumber(...)` and descriptive text (`default 50`, `max 200`, `offset >= 0`) but no schema-level `minimum/maximum/default` constraints.
     - Tests assert description text presence, not schema constraint enforcement.
   - Risk:
     - Clients can send values outside described guardrails; enforcement relies on downstream app validation/defaulting path.

2. `common` adapter forwarding path has no coverage in default lane test invocation due build tags.
   - Evidence:
     - `internal/adapters/server/common/app_service_adapter_mcp.go:262`
     - `internal/adapters/server/common/app_service_adapter_mcp_guard_test.go:1`
     - `internal/adapters/server/common/app_service_adapter_mcp_actor_attribution_test.go:1`
     - `internal/adapters/server/common/app_service_adapter_test.go:1`
   - Detail:
     - Scoped command result was `[no test files]`; existing `_test.go` files are behind `//go:build commonhash`.
   - Risk:
     - Regressions in `SearchTasksRequest -> app.SearchTasksFilter` mapping could evade default package test runs for this lane.

### Low
1. Plan section 5 ranking semantics are only partially exposed at MCP tool schema level.
   - Evidence:
     - `VECTOR_SEARCH_EXECUTION_PLAN.md:97`
     - `VECTOR_SEARCH_EXECUTION_PLAN.md:104`
     - `internal/adapters/server/mcpapi/extended_tools.go:664`
     - `internal/adapters/server/mcpapi/extended_tools.go:675`
   - Detail:
     - MCP schema exposes `sort=rank_desc` and mode options, but not explicit weight composition/tie-breaker notes from plan section 5.
   - Risk:
     - Primarily expectation/documentation mismatch risk for integrators, not a functional forwarding break.

## Completeness checklist vs plan (sections 3, 5)
- Section 3.1 Query modes (`keyword|semantic|hybrid`) exposed in tool schema: **PASS**
  - `internal/adapters/server/mcpapi/extended_tools.go:674`
  - `internal/adapters/server/mcpapi/extended_tools_test.go:750`
- Section 3.2 Filters (`project_id`, `states`, `include_archived`, `levels`, `kinds`, `labels_any`, `labels_all`) exposed and forwarded: **PASS**
  - schema: `internal/adapters/server/mcpapi/extended_tools.go:665`, `:669`, `:670`, `:671`, `:672`, `:673`
  - forwarding to transport request: `internal/adapters/server/mcpapi/extended_tools.go:681`, `:685`, `:686`, `:687`, `:688`, `:689`
  - forwarding to app filter: `internal/adapters/server/common/app_service_adapter_mcp.go:263`, `:267`, `:268`, `:269`, `:270`, `:271`
- Section 3.3 Sort options exposed and forwarded: **PASS**
  - `internal/adapters/server/mcpapi/extended_tools.go:675`
  - `internal/adapters/server/mcpapi/extended_tools.go:691`
  - `internal/adapters/server/common/app_service_adapter_mcp.go:273`
- Section 3.4 Pagination/limits/default/guardrail disclosure: **PASS (with caveat)**
  - disclosure present: `internal/adapters/server/mcpapi/extended_tools.go:676`, `:677`
  - forwarded: `internal/adapters/server/mcpapi/extended_tools.go:692`, `:693`, `internal/adapters/server/common/app_service_adapter_mcp.go:274`, `:275`
  - caveat: constraints are not schema-enforced (finding M1).
- Section 5 ranking/dedup exposure in MCP contract: **PARTIAL**
  - rank sort is exposed (`rank_desc`), but weighting/tie-breaker semantics from plan are not explicitly surfaced in schema text.

## Residual risks
1. Pagination guardrails are advisory in schema text rather than encoded constraints.
2. `internal/adapters/server/common` default test path does not exercise tagged tests in this lane command profile.
3. Consumers depending on explicit rank internals may need non-schema docs/source references.

## Final verdict
**CONDITIONALLY PASS** for MCP/transport search contract exposure.

Core surface completeness and forwarding for modes/filters/sort/pagination are present and mcpapi tests pass. Main QA risks are schema-level guardrail enforceability and tagged-test coverage visibility for the common adapter path.
