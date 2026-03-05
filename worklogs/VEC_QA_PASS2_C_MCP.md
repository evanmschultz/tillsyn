# QA2-C MCP Search Schema/Forwarding Remediation Pass 2

Date: 2026-03-03
Lane: QA2-C
Objective: Independent QA pass 2 for MCP search schema/forwarding remediation completeness

## Verdict
PASS

## Findings by Severity

### High
- None.

### Medium
- None.

### Low
- Residual tagged-test caveat remains outside the required command scope.
  - `internal/adapters/server/common` tests are build-tagged with `commonhash`: `internal/adapters/server/common/app_service_adapter_mcp_guard_test.go:1`, `internal/adapters/server/common/app_service_adapter_mcp_actor_attribution_test.go:1`, `internal/adapters/server/common/app_service_adapter_test.go:1`.
  - Package-wide runs without the tag report no runnable tests for `common`: `.tmp/vec-wavef-evidence/20260303_180827/just_check.txt:7`.
  - Impact for this lane is low because required MCP search schema/forwarding checks are covered in `mcpapi` tests and passed.

## Required Check Results

1. Verify pagination schema constraints/defaults are encoded in tool schema: PASS.
- `till.search_task_matches` encodes numeric defaults/bounds directly in schema options:
  - `limit` includes `DefaultNumber(50)`, `Min(0)`, `Max(200)`: `internal/adapters/server/mcpapi/extended_tools.go:676`.
  - `offset` includes `DefaultNumber(0)`, `Min(0)`: `internal/adapters/server/mcpapi/extended_tools.go:683`.
- Schema assertions validate emitted `minimum`/`maximum`/`default` fields:
  - `limit`: `internal/adapters/server/mcpapi/extended_tools_test.go:818`.
  - `offset`: `internal/adapters/server/mcpapi/extended_tools_test.go:831`.

2. Verify mode/sort/levels/kinds/labels/limit/offset forwarding and tests: PASS.
- MCP tool schema includes filter/sort/mode/pagination fields:
  - `levels`, `kinds`, `labels_any`, `labels_all`, `mode`, `sort`, `limit`, `offset` at `internal/adapters/server/mcpapi/extended_tools.go:670`.
- Request parsing forwards all fields to transport request:
  - `internal/adapters/server/mcpapi/extended_tools.go:691`.
- Transport request type carries all forwarded fields:
  - `internal/adapters/server/common/mcp_surface.go:115`.
- Adapter forwards all fields into app-layer filter:
  - `internal/adapters/server/common/app_service_adapter_mcp.go:262`.
- Tests cover schema descriptions/enums/defaults and explicit/default forwarding:
  - schema checks: `internal/adapters/server/mcpapi/extended_tools_test.go:772`.
  - forwarding checks with explicit args and default pass-through: `internal/adapters/server/mcpapi/extended_tools_test.go:855`.

3. Residual caveats noted with severity: PASS.
- Tagged-test caveat recorded as Low (above).

## Commands / Tests Executed

1. `just test-pkg ./internal/adapters/server/mcpapi`
- Outcome: PASS (`ok   github.com/hylla/tillsyn/internal/adapters/server/mcpapi (cached)`).

2. Audit evidence checks
- `rg -n "//go:build" internal/adapters/server/common/*_test.go`
  - Outcome: PASS; confirmed `commonhash` tags on `common` tests.
- `nl -ba .tmp/vec-wavef-evidence/20260303_180827/test_pkg_internal_adapters_server_mcpapi.txt | sed -n '1,220p'`
  - Outcome: PASS; recorded non-cached package test evidence at line 1.
- `nl -ba .tmp/vec-wavef-evidence/20260303_180827/just_check.txt | sed -n '1,220p'`
  - Outcome: PASS; confirms `internal/adapters/server/common` shows `[no test files]` in default run at line 7.

## Context7 Compliance

- Context7 consulted before schema-semantics claims:
  - Library: `/mark3labs/mcp-go`.
  - Topic: numeric schema options/defaults/bounds for tool parameters (`WithNumber`, defaults, min/max).
- Context7 was available; no fallback source required for this requirement.
- No test/runtime failure occurred after Context7 consult, so failure-triggered re-consult was not required.

## Unresolved Risks

- Build-tagged `commonhash` tests in `internal/adapters/server/common` are not exercised by the required lane command.

## Exact Next Step

1. If you want explicit closure of the tagged-test caveat, run a tagged `common` test job via an approved `just` recipe (for example, `just test-pkg` variant with `commonhash`) and archive the output under `.tmp/vec-wavef-evidence/`.
