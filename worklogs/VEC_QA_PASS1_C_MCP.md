# QA1-C MCP Search Schema/Forwarding Review

Date: 2026-03-03
Lane: QA1-C
Scope: independent QA pass for MCP search schema guardrails + forwarding tests

## Verdict
PASS

## Findings by Severity

### High
- None.

### Medium
- None.

### Low
- Residual tagged-test coverage caveat remains for adjacent package tests.
  - Evidence: `internal/adapters/server/common` has build-tagged tests (`//go:build commonhash`) at [app_service_adapter_mcp_guard_test.go](/Users/evanschultz/Documents/Code/hylla/tillsyn/internal/adapters/server/common/app_service_adapter_mcp_guard_test.go:1), [app_service_adapter_mcp_actor_attribution_test.go](/Users/evanschultz/Documents/Code/hylla/tillsyn/internal/adapters/server/common/app_service_adapter_mcp_actor_attribution_test.go:1), and [app_service_adapter_test.go](/Users/evanschultz/Documents/Code/hylla/tillsyn/internal/adapters/server/common/app_service_adapter_test.go:1).
  - The required run `just test-pkg ./internal/adapters/server/mcpapi` does not execute those `commonhash`-tagged tests.
  - Additional note: grep over `internal/adapters/server/common/*test.go` found no search forwarding assertions in that tagged suite, so impact to this lane’s MCP search remediation is low.

## Required Check Results

1. Verify tool schema encodes pagination guardrails as schema constraints (not description-only): PASS.
- Tool schema now sets numeric constraints/defaults directly in `mcp.WithNumber`:
  - `limit`: `mcp.DefaultNumber(50)`, `mcp.Min(0)`, `mcp.Max(200)` at [extended_tools.go](/Users/evanschultz/Documents/Code/hylla/tillsyn/internal/adapters/server/mcpapi/extended_tools.go:676).
  - `offset`: `mcp.DefaultNumber(0)`, `mcp.Min(0)` at [extended_tools.go](/Users/evanschultz/Documents/Code/hylla/tillsyn/internal/adapters/server/mcpapi/extended_tools.go:683).
- Schema assertions verify emitted fields (not just descriptions):
  - helper at [extended_tools_test.go](/Users/evanschultz/Documents/Code/hylla/tillsyn/internal/adapters/server/mcpapi/extended_tools_test.go:430).
  - assertions for `minimum/maximum/default` at [extended_tools_test.go](/Users/evanschultz/Documents/Code/hylla/tillsyn/internal/adapters/server/mcpapi/extended_tools_test.go:818).

2. Verify mode/sort/levels/kinds/labels/limit/offset are forwarded and tested: PASS.
- Request forwarding wiring includes all fields at [extended_tools.go](/Users/evanschultz/Documents/Code/hylla/tillsyn/internal/adapters/server/mcpapi/extended_tools.go:691).
- Transport contract includes fields in request struct at [mcp_surface.go](/Users/evanschultz/Documents/Code/hylla/tillsyn/internal/adapters/server/common/mcp_surface.go:115).
- App adapter forwards all fields into `app.SearchTasksFilter` at [app_service_adapter_mcp.go](/Users/evanschultz/Documents/Code/hylla/tillsyn/internal/adapters/server/common/app_service_adapter_mcp.go:262).
- MCP tests assert forwarding of `mode`, `sort`, `levels`, `kinds`, `labels_any`, `labels_all`, `limit`, `offset` and default pass-through behavior at [extended_tools_test.go](/Users/evanschultz/Documents/Code/hylla/tillsyn/internal/adapters/server/mcpapi/extended_tools_test.go:839).

3. Verify residual tagged-test coverage caveat is called out clearly: PASS.
- Caveat documented in Findings (Low) above with explicit file evidence and impact.

## Context7 Compliance
- Consulted Context7 for `mcp-go` schema option semantics prior to claims.
  - Library: `/mark3labs/mcp-go`.
  - Confirmed tool schema option patterns for numeric defaults and bounds.
- Supplementary fallback confirmation from local module source (for exact symbol names used in this repo):
  - `DefaultNumber`, `Min`, `Max` definitions at `/Users/evanschultz/go/pkg/mod/github.com/mark3labs/mcp-go@v0.44.0/mcp/tools.go:1085`.

## Commands / Tests Executed
- `just test-pkg ./internal/adapters/server/mcpapi` -> PASS (`ok   github.com/hylla/tillsyn/internal/adapters/server/mcpapi (cached)`).
- Evidence cross-check reviewed:
  - `.tmp/vec-wavef-evidence/20260303_175936/test_pkg_internal_adapters_server_mcpapi.txt:1`
  - `.tmp/vec-wavef-evidence/20260303_175936/just_check.txt:9`
  - `.tmp/vec-wavef-evidence/20260303_175936/just_ci.txt:9`

## Unresolved Risks
- The required QA command is package-scoped and cache-eligible; it does not exercise build-tagged tests in adjacent `common` package by default.

## Exact Next Step
1. If explicit closure of the tagged-test caveat is desired for this wave, run `just test-pkg ./internal/adapters/server/common` with `-tags commonhash` wiring via an approved `just` recipe and archive output under `.tmp/vec-wavef-evidence/`.
