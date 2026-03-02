# COLLAB TEST 2026-03-02

## Objective
- Re-run all remaining dogfooding blockers and changed paths after the 2026-03-02 remediation wave.
- Execute every agent-only test now, record evidence inline, then hand off only the required collaborative/manual checks.

## Sources Pulled Forward
- `COLLABORATIVE_POST_FIX_VALIDATION_WORKSHEET.md`
- `MCP_DOGFOODING_WORKSHEET.md`
- `COLLAB_E2E_REMEDIATION_PLAN_WORKLOG.md`
- `PLAN.md`

## Carry-Forward Remaining Items
- `MCP_DOGFOODING_WORKSHEET.md`: `M3.1 -> C-01`, `M3.2 -> C-02`, `M4.1 -> C-03`, `M4.2 -> C-04`.
- `COLLABORATIVE_POST_FIX_VALIDATION_WORKSHEET.md`: Section 4 regression sweep -> `C-05`; Section 5 archived/search/keybinding checks -> `C-06`; Section 6 manual notifications bubbling check -> `C-07`; Section 6 sink-parity rerun -> `E-08`; Section 8 restore rerun -> `E-07`.

## Environment
- Workspace: `/Users/evanschultz/Documents/Code/hylla/tillsyn`
- Date label: `2026-03-02`
- Evidence dir: `.tmp/collab-test-2026-03-02/`

## Test Matrix
| ID | Area | Mode | Status | Evidence | Notes |
|---|---|---|---|---|---|
| A-01 | `just check` gate | Agent-only | PASS | `.tmp/collab-test-2026-03-02/a01_just_check.txt` | Baseline gate passed. |
| A-02 | `just ci` gate | Agent-only | PASS | `.tmp/collab-test-2026-03-02/a02_just_ci.txt` | Full gate passed (with non-fatal Go stat-cache permission warning). |
| A-03 | `just test-golden` | Agent-only | PASS | `.tmp/collab-test-2026-03-02/a03_test_golden.txt` | Golden fixture test passed. |
| A-04 | CLI help smoke (`./till --help`, `./till serve --help`) | Agent-only | PASS | `.tmp/collab-test-2026-03-02/a04_help_root.txt`, `.tmp/collab-test-2026-03-02/a04_help_serve.txt`, `.tmp/collab-test-2026-03-02/a04_help_summary.txt` | Both help commands produced usage with `0` stderr bytes. |
| A-05 | Startup config seeding behavior | Agent-only | PASS | `.tmp/collab-test-2026-03-02/a05_startup_seed_check.txt` | `cmd/till` package tests pass (includes startup seeding regression tests). |
| E-01 | Start isolated serve runtime | Agent-only | PASS | `.tmp/collab-test-2026-03-02/e01_serve_start.log`, `.tmp/collab-test-2026-03-02/e01_healthz.json` | Runtime started and `/healthz` returned `{"status":"ok"}`. |
| E-02 | MCP initialize + tools/list | Agent-only | PASS | `.tmp/collab-test-2026-03-02/e02_mcp_initialize.json`, `.tmp/collab-test-2026-03-02/e02_mcp_tools_list.json` | MCP initialized (`protocolVersion=2024-11-05`) and tools list returned (`30` tools). |
| E-03 | Fixture seed for scoped checks (project->branch->phase->task) | Agent-only | PASS | `.tmp/collab-test-2026-03-02/e03_fixture_seed.json` | Deterministic fixture IDs created for guardrail checks. |
| E-04 | Guardrail M2.1 (non-user mutation without tuple fails closed) | Agent-only | PASS | `.tmp/collab-test-2026-03-02/e04_guard_m21.json` | Fail-closed confirmed (`isError=true`, `invalid_request`). |
| E-05 | Guardrail M2.2 (scope mismatch fails closed) | Agent-only | PASS | `.tmp/collab-test-2026-03-02/e05_guard_m22.json` | Fail-closed confirmed (`isError=true`, `not_found`). |
| E-06 | Guardrail M2.3 (completion blocked by unresolved blocker) | Agent-only | PASS | `.tmp/collab-test-2026-03-02/e06_guard_m23.md` | First completion transition blocked, second succeeded after blocker resolution. |
| E-07 | Focused `till_restore_task` rerun (historical fail row) | Agent-only | PASS | `.tmp/collab-test-2026-03-02/e07_restore_rerun.json` | Restore rerun succeeded in transport flow (`isError=null`). |
| E-08 | Logging sink parity rerun (mapped MCP/HTTP errors show in runtime sink) | Agent-only | PASS | `.tmp/collab-test-2026-03-02/e08_rerun_v2_summary.log`, `.tmp/collab-test-2026-03-02/e08_rerun_v2_serve_stderr.log`, `.tmp/collab-test-2026-03-02/e08_rerun_v2_mcp_invalid_resp.json`, `.tmp/collab-test-2026-03-02/e08_rerun_v2_http_invalid.json` | Rerun confirms mapped MCP + HTTP errors incremented in `.tillsyn/log` and surfaced on serve stderr. |
| S-01 | Gatekept subagent lane (in-scope mutation) | Agent + subagent | PASS | `.tmp/collab-test-2026-03-02/s01_subagent_in_scope.md` | Valid task-scoped lease allowed in-scope mutation (`actual_is_error=false`). |
| S-02 | Gatekept subagent lane (out-of-scope mutation) | Agent + subagent | PASS | `.tmp/collab-test-2026-03-02/s02_subagent_out_scope.md` | Scope-mismatched mutation failed closed (`actual_is_error=true`, `guardrail_failed`). |
| C-01 | TUI warning indicator + compact panel (M3.1) | Collaborative/manual | PENDING_USER | `TBD` | Requires user + running TUI. |
| C-02 | Resolve parity transport + TUI (M3.2) | Collaborative/manual | PENDING_USER | `TBD` | Requires user + running TUI. |
| C-03 | Level-scoped search/filter coverage (M4.1) | Collaborative/manual | PENDING_USER | `TBD` | Requires user + TUI parity observation. |
| C-04 | Search/filter parity API/MCP vs TUI (M4.2) | Collaborative/manual | PENDING_USER | `TBD` | Requires user + TUI. |
| C-05 | TUI regression sweep Section 4 (C4/C6/C9/C10/C11/C12/C13) | Collaborative/manual | PENDING_USER | `TBD` | User-driven validation required. |
| C-06 | Archived/search/keybinding targeted checks Section 5 | Collaborative/manual | PENDING_USER | `TBD` | User-driven validation required. |
| C-07 | Notifications bubbling + quick-info parity (Section 6 manual UI check) | Collaborative/manual | PENDING_USER | `TBD` | Requires user-visible TUI confirmation of warning/error surfacing. |

## Execution Log
- 2026-03-02: Worksheet created.
- 2026-03-02: `A-01` PASS (`just check`).
- 2026-03-02: `A-02` PASS (`just ci`), includes non-fatal Go stat-cache permission warning in this environment.
- 2026-03-02: `A-03` PASS (`just test-golden`).
- 2026-03-02: `A-04` PASS (`./till --help`, `./till serve --help` with zero stderr bytes).
- 2026-03-02: `A-05` PASS (`just test-pkg ./cmd/till`; seeding regression coverage included in package tests).
- 2026-03-02: `E-01` to `E-07` PASS via isolated live runtime transport checks (health, MCP init/list, guardrails M2.1/M2.2/M2.3, restore rerun).
- 2026-03-02: `E-08` rerun PASS after runtime default logger sink-bridge fix. Live probe shows both mapped MCP + HTTP error lines in `.tillsyn/log` and stderr.
- 2026-03-02: Initial `S-01`/`S-02` probe attempt blocked by sandbox bind restrictions; rerun with escalated local bind permissions completed and both checks passed.

## User Notes Overflow
Use this section for findings that do not map directly to `C-01` through `C-07` during collaborative/manual validation.

| Timestamp | Surface | Observation | Expected | Severity | Mapped Section |
|---|---|---|---|---|---|
| 2026-03-02T04:56Z | MCP tools | On clean DB, `till.create_task` requires `column_id` but MCP surface used in this run does not expose a `list_columns`-style tool to discover initial column IDs. | MCP-only bootstrap path should allow task creation from zero state without manual TUI assist. | medium | `UNMAPPED` |

## Live MCP + TUI Joint Run (2026-03-02)

| Step | Action | Status | Evidence | Watch In TUI |
|---|---|---|---|---|
| L-01 | MCP baseline check (`till_get_bootstrap_guide`, `till_list_projects`) | PASS | bootstrap output + project list (tool transcript) | Project list state at startup/empty flow. |
| L-02 | Create collaborative run project via MCP (`collab-live-2026-03-02`) | PASS | Project ID `bb6ecc18-d978-4d91-a51f-c65dbea189ef` | New project appears in picker/list. |
| L-03 | Attempt zero-state hierarchy seeding via MCP-only `till_create_task` | BLOCKED | MCP error: `invalid_request: required argument "column_id" not found` | No task created; awaiting seed-column workaround. |
| L-04 | Raise unresolved attention item via MCP (`approval_required`) | PASS | Attention ID `82dd0391-7ed5-4c02-9816-d7d78c69cc81`; capture_state `open_count=1`, `requires_user_action=1` | Warning/attention indicators should appear for project scope. |
| L-05 | Resolve same attention item via MCP and verify state | PASS | `till_resolve_attention_item` + capture_state `open_count=0` | Warning/attention indicators should clear. |
