# Collaborative Parallel Fix Tracker (2026-03-02)

This tracker is the execution log for the active parallel remediation wave.
Reference policy: `PARALLEL_AGENT_RUNBOOK.md`.

## Goals

1. Fix comment coverage gaps across all node types used in scope hierarchy.
2. Fix TUI right-side notices workflow into focusable/scrollable/selectable triage lists.
3. Keep work non-overlapping by lane lock scope and integrate with targeted package checks.

## Lock Table

| Lane ID                 | Owner             | Lock Scope                                                                                                                                                                                                                                                                                                    | Out of Scope                                                                                       | Objective                                                                                      | Status   |
| ----------------------- | ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- | -------- |
| LANE-COMMENT-TARGETS    | worker-subagent   | `internal/domain/comment.go`, `internal/domain/comment_test.go`, `internal/app/service_test.go`, `internal/app/snapshot.go`, `internal/adapters/server/mcpapi/extended_tools.go`, `internal/adapters/server/mcpapi/extended_tools_test.go`, `internal/tui/thread_mode.go`, `internal/tui/thread_mode_test.go` | `internal/tui/model.go`, `internal/tui/model_test.go`                                              | Add branch/subphase comment target support end-to-end (domain/app/MCP/TUI mapping).            | COMPLETE |
| LANE-NOTICES-PANEL      | worker-subagent   | `internal/tui/model.go`, `internal/tui/model_test.go`, `internal/tui/keymap.go`                                                                                                                                                                                                                               | `internal/domain/*`, `internal/app/*`, `internal/adapters/server/*`, `internal/tui/thread_mode.go` | Convert notices panel to section-focusable, scrollable, selectable list UX with Enter actions. | COMPLETE |
| LANE-OWNERSHIP-PROPOSAL | explorer-subagent | docs-only analysis                                                                                                                                                                                                                                                                                            | any code edits                                                                                     | Confirm ownership tracking sufficiency and propose schema upgrade path for stable actor IDs.   | COMPLETE |

## Checkpoint Log

### Checkpoint 1 - Baseline

- Completed:
    - Created `COMMENT_SCHEMA_COVERAGE_AND_BUILD_PLAN.md`.
    - Created this tracker file.
    - Confirmed current gaps:
        - missing comment target types for `branch` and `subphase`,
        - TUI thread mapping excludes branch/subphase,
        - notices panel currently non-list sections.
- Commands:
    - `sed`, `rg` code audit commands only.
- Status:
    - Ready to dispatch non-overlapping subagent lanes.

### Checkpoint 2 - Lane Dispatch

- Launched:
    - `LANE-COMMENT-TARGETS` worker (`019cad0b-e43a-7dc3-bcd2-06bf67b292a7`)
    - `LANE-NOTICES-PANEL` worker (`019cad0b-e681-70a2-8d6d-0ed6186eb30a`)
    - `LANE-OWNERSHIP-PROPOSAL` explorer (`019cad0b-e893-7630-9349-d27a4636d2a9`)
- Status:
    - All lanes running under non-overlapping lock scopes.

### Checkpoint 3 - Lane Handoffs Received

- `LANE-COMMENT-TARGETS` (`019cad0b-e43a-7dc3-bcd2-06bf67b292a7`)
    - Delivered:
        - domain target-type additions (`branch`, `subphase`) + normalization tests.
        - snapshot mapping updates + app coverage tests.
        - MCP comment tool enum expansion + transport-level pass-through tests.
        - TUI thread mapping updates + new thread mapping tests.
    - Worker-scoped tests reported:
        - `just test-pkg ./internal/domain` PASS
        - `just test-pkg ./internal/app` PASS
        - `just test-pkg ./internal/adapters/server/mcpapi` PASS
        - `just test-pkg ./internal/tui` PASS
- `LANE-NOTICES-PANEL` (`019cad0b-e681-70a2-8d6d-0ed6186eb30a`)
    - Delivered:
        - section-level notices focus model (`warnings`, `agent/user action`, `selection`, `recent activity`),
        - per-section cursor and windowed list scrolling,
        - Enter activation for selected notices row (task info or activity detail),
        - updated board help copy and keymap wording.
    - Worker-scoped tests reported:
        - `just test-pkg ./internal/tui` PASS
- `LANE-OWNERSHIP-PROPOSAL` (`019cad0b-e893-7630-9349-d27a4636d2a9`)
    - Delivered:
        - ownership audit confirms current attribution is `actor_type + author_name`,
        - no stable `actor_id` column/path yet,
        - proposal options documented; recommended low-risk wave decision is to defer schema migration.

### Checkpoint 4 - Integrator Verification

- Integrator review:
    - lock scopes were respected (no cross-lane file collisions),
    - compiled lane diffs reviewed and merged in current branch worktree.
- Integrator validation commands:
    1. `just test-pkg ./internal/domain` PASS
    2. `just test-pkg ./internal/app` PASS
    3. `just test-pkg ./internal/adapters/server/mcpapi` PASS
    4. `just test-pkg ./internal/tui` PASS
    5. `just ci` PASS
- Wave outcome:
    - comment coverage now includes all hierarchy node types used by TUI/MCP/domain snapshot flows.
    - notices panel now supports focusable, scrollable, selectable list sections under one panel border.
    - ownership tracking remains display-name based; stable actor ID is intentionally deferred to a follow-up wave.

## Lane Handoffs and Verification

### LANE-COMMENT-TARGETS

- Handoff: received
- Files changed:
    - `internal/domain/comment.go`
    - `internal/domain/comment_test.go`
    - `internal/app/snapshot.go`
    - `internal/app/service_test.go`
    - `internal/adapters/server/mcpapi/extended_tools.go`
    - `internal/adapters/server/mcpapi/extended_tools_test.go`
    - `internal/tui/thread_mode.go`
    - `internal/tui/thread_mode_test.go`
- Targeted tests:
    - `just test-pkg ./internal/domain` PASS
    - `just test-pkg ./internal/app` PASS
    - `just test-pkg ./internal/adapters/server/mcpapi` PASS
    - `just test-pkg ./internal/tui` PASS
- Integrator verification: PASS
- Status: COMPLETE

### LANE-NOTICES-PANEL

- Handoff: received
- Files changed:
    - `internal/tui/model.go`
    - `internal/tui/model_test.go`
    - `internal/tui/keymap.go`
- Targeted tests:
    - `just test-pkg ./internal/tui` PASS
- Integrator verification:
    - `just test-pkg ./internal/tui` PASS
- Status: COMPLETE

### LANE-OWNERSHIP-PROPOSAL

- Handoff: received
- Summary:
    - ownership persists as `actor_type + author_name` (no stable actor ID today),
    - schema migration for actor ID is optional follow-up, not required for current dogfood wave closeout.
- Status: COMPLETE

## Wave 2: Ownership + TUI Closeout (No Legacy Shim Policy)

### Lock Table (Wave 2)

| Lane ID | Owner | Lock Scope | Out of Scope | Objective | Status |
|---|---|---|---|---|---|
| LANE-OWNERSHIP-CORE | worker-subagent | `internal/domain/comment.go`, `internal/domain/comment_test.go`, `internal/domain/change_event.go`, `internal/app/service.go`, `internal/app/service_test.go`, `internal/app/snapshot.go`, `internal/app/snapshot_test.go`, `internal/adapters/storage/sqlite/repo.go`, `internal/adapters/storage/sqlite/repo_test.go`, `internal/adapters/server/common/mcp_surface.go`, `internal/adapters/server/common/app_service_adapter_mcp.go`, `internal/adapters/server/mcpapi/extended_tools.go`, `internal/adapters/server/mcpapi/extended_tools_test.go` | `internal/tui/*`, `internal/config/*`, `cmd/till/*` | Implement immutable ownership tuple for comments/events and migrate schema cleanly without legacy dual-path shims. | COMPLETE |
| LANE-TUI-FLOWS | worker-subagent | `internal/tui/model.go`, `internal/tui/model_test.go`, `internal/tui/options.go`, `internal/tui/thread_mode.go`, `internal/tui/thread_mode_test.go`, `internal/tui/keymap.go` | non-`internal/tui/*` paths | Consume ownership tuple in TUI render/actions and complete outstanding notices/global panel UX fixes that are TUI-local. | COMPLETE |
| LANE-CONFIG-IDENTITY | worker-subagent | `internal/config/config.go`, `internal/config/config_test.go`, `cmd/till/main.go`, `cmd/till/main_test.go`, `config.example.toml` | non-config/cmd paths | Add immutable `identity.actor_id` generation/persistence and runtime wiring. | COMPLETE |
| LANE-REVIEW-REMEDIATION | worker-subagent | `cmd/till/main.go`, `cmd/till/main_test.go`, `internal/tui/options.go`, `internal/tui/model.go`, `internal/tui/model_test.go`, `internal/app/mutation_guard.go`, `internal/app/service.go`, `internal/app/service_test.go`, `internal/adapters/server/common/app_service_adapter_mcp.go`, `internal/adapters/server/mcpapi/extended_tools_test.go`, `internal/adapters/storage/sqlite/repo.go`, `internal/adapters/storage/sqlite/repo_test.go`, `internal/app/snapshot.go`, `internal/app/snapshot_test.go` | docs/plan files and unrelated packages | Resolve independent-review blockers and re-validate end-to-end gates. | COMPLETE |

### Review Lanes (Wave 2)

| Review Lane | Owner | Scope | Requirement | Status |
|---|---|---|---|---|
| REVIEW-OWNERSHIP-CORE | explorer-subagent | `LANE-OWNERSHIP-CORE` changed files | Validate correctness, migration safety, architecture boundaries, and test evidence quality. | COMPLETE |
| REVIEW-TUI-FLOWS | explorer-subagent | `LANE-TUI-FLOWS` changed files | Validate focus/navigation behavior, ownership rendering semantics, and regression risk. | COMPLETE (remediation required) |
| REVIEW-CONFIG-IDENTITY | explorer-subagent | `LANE-CONFIG-IDENTITY` changed files | Validate immutable-id bootstrap logic, config persistence correctness, and startup behavior. | COMPLETE (remediation required) |
| REVIEW-REMEDIATION-PASS2 | explorer-subagent | remediation delta | Confirm blocker fixes and gate readiness after remediation lane. | COMPLETE (PASS) |

### Checkpoint 5 - Wave 2 Initial Handoffs

- Received handoffs for:
  - `LANE-OWNERSHIP-CORE` (including recovery pass),
  - `LANE-TUI-FLOWS`,
  - `LANE-CONFIG-IDENTITY`.
- Initial scoped tests were green in lane scopes except one integration compile break surfaced in `cmd/till`.

### Checkpoint 6 - Independent Review Findings

- `REVIEW-CONFIG-IDENTITY`: found critical compile break in bootstrap callback (`cfg.Identity` access on `tui.BootstrapConfig`).
- `REVIEW-TUI-FLOWS`: found high-risk actor-id startup wiring drift.
- `REVIEW-OWNERSHIP-CORE`: found contract mismatch where task `actor_name` was accepted at surface but not preserved in events, plus snapshot-versioning concern.
- Integration status after first review pass: FAIL (targeted remediation required).

### Checkpoint 7 - Targeted Remediation

- Dispatched `LANE-REVIEW-REMEDIATION` and fixed:
  1. bootstrap callback compile break in `cmd/till`,
  2. startup/runtime propagation for `identity.actor_id` into TUI model,
  3. change-event preservation of mutation `actor_name`,
  4. explicit snapshot version semantics for ownership shape change (`v2`, strict import match).
- Remediation lane command evidence:
  - `just test-pkg ./internal/tui` PASS
  - `just test-pkg ./internal/adapters/storage/sqlite` PASS
  - `just test-pkg ./internal/app` PASS
  - `just test-pkg ./internal/adapters/server/mcpapi` PASS
  - `just test-pkg ./cmd/till` PASS
  - `just ci` PASS

### Checkpoint 8 - Final Independent Review + Integrator Gates

- `REVIEW-REMEDIATION-PASS2`: PASS (no remaining high/medium findings in reviewed scope).
- Integrator reran full gates:
  - `just check` PASS
  - `just ci` PASS
- Wave 2 closeout status: READY FOR COLLAB TEST CONTINUATION.

### Checkpoint 9 - Sign-Off Revalidation (Post-Integration)

- Revalidated gate health after final tracker/doc synchronization:
  - `just check` PASS
  - `just ci` PASS
- Sign-off status: COMPLETE. Parallel lanes + independent review lanes are closed with green integrator gates.
