# Collaborative Parallel Fix Tracker (2026-03-02)

This tracker is the execution log for parallel remediation waves on 2026-03-02.
Reference policy: `PARALLEL_AGENT_RUNBOOK.md`.

## Goals

1. Close collaborative remediation gaps with lock-scoped parallel lanes.
2. Preserve independent QA sign-off + gate evidence before marking any section complete.
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
- Sign-off status (Wave 2 only): COMPLETE. Wave 3 remains active below.

## Wave 3: Notifications + Global Panel Redesign (User-Requested)

### Lock Table (Wave 3)

| Lane ID | Owner | Lock Scope | Out of Scope | Objective | Status |
|---|---|---|---|---|---|
| LANE-NOTIFICATIONS-REDESIGN | worker-subagent | `internal/tui/model.go`, `internal/tui/model_test.go` | non-`internal/tui/*` paths, docs/worklogs | Implement requested notifications redesign: remove redundant header rows, preserve focusable section lists, and add separate global notifications panel below project notifications with selectable rows/actions. | COMPLETE (superseded by remediation lane) |
| LANE-NOTIFICATIONS-REMEDIATION | worker-subagent | `internal/tui/model.go`, `internal/tui/model_test.go` | non-`internal/tui/*` paths, docs/worklogs | Address QA findings (actionability, stable-key selection, partial-fetch resilience, edge-case tests, and gate recovery). | COMPLETE |
| REVIEW-NOTIFICATIONS-REDESIGN | explorer-subagent | `LANE-NOTIFICATIONS-REDESIGN` + remediation delta + affected collaborative markdown docs | direct code edits | Independent QA review for correctness, UX parity with user intent, test adequacy, and markdown/worklog consistency before sign-off. | COMPLETE (initial FAIL, final PASS after remediation) |

### Checkpoint 10 - Wave 3 Kickoff

- Trigger: user reported zero visible change for requested notifications/global-panel redesign.
- Scope locked to TUI panel architecture and navigation behavior for this remediation section.
- Completion gate (explicit):
  1. worker implementation handoff,
  2. independent QA subagent sign-off on code + markdown docs,
  3. passing tests (`just test-pkg ./internal/tui`, `just check`, `just ci`),
  4. user-run confirmation before marking section complete.

### Checkpoint 11 - Initial Worker Handoff + Independent QA (FAIL)

- Worker handoff received for `LANE-NOTIFICATIONS-REDESIGN` with TUI model/test changes.
- Independent QA outcomes:
  - code QA: FAIL (global Enter actionability/stability/resilience gaps + red TUI package gate),
  - docs QA: FAIL (Wave 3 acceptance coverage and tracker/worklog consistency gaps).
- Decision: dispatched remediation lane before any completion mark.

### Checkpoint 12 - Remediation + Integrator Gates (Green, Pending Final QA/User)

- Remediation handoff received for `LANE-NOTIFICATIONS-REMEDIATION`:
  - stable-key global row identity + selection re-anchor,
  - deterministic non-task Enter modal fallback,
  - resilient non-active project global fetch handling with partial-results signaling,
  - expanded edge-case TUI tests.
- Integrator gate evidence after remediation:
  - `just test-pkg ./internal/tui` PASS
  - `just check` PASS
  - `just ci` PASS
- Wave 3 status: implementation ready, final QA sign-off + user-run confirmation still required before closeout.

### Checkpoint 13 - Final QA Outcomes

- Independent code QA (`REVIEW-NOTIFICATIONS-FINAL-CODE`): PASS (no remaining high/medium findings in TUI scope).
- Independent docs/process QA initially flagged:
  1. missing explicit scrollable acceptance row in active collaborative worksheet,
  2. wording drift on QA-pending vs user-confirmation-pending state.
- Doc gaps remediated:
  - added explicit manual row for notifications scrolling behavior in `COLLAB_TEST_2026-03-02.md` (`C-12`),
  - synchronized wording to reflect QA pass and remaining user confirmation gate.
- Current Wave 3 closeout state: QA complete, tests green, awaiting user-run collaborative confirmation before final completion mark.

### Checkpoint 14 - Final Docs Recheck Interpretation

- Final docs recheck confirms no remaining process/doc drift defects.
- Reported FAIL is expected because user confirmation rows (`C-08`..`C-12`) are intentionally still `PENDING_USER` until collaborative run completes.
- Wave 3 remains intentionally open pending user-run evidence capture.

## Wave 4: Markdown-First Summary/Details/Comments Closeout

### Lock Table (Wave 4)

| Lane ID | Owner | Lock Scope | Objective | Status |
|---|---|---|---|---|
| LANE-COMMENT-MODEL | worker-subagent | `internal/domain/comment.go`, `internal/domain/comment_test.go`, `internal/app/service.go`, `internal/app/service_test.go`, `internal/app/snapshot.go`, `internal/app/snapshot_test.go`, `internal/app/ports.go` | Add canonical comment `summary` field semantics (fallback from body) through domain/app/snapshot. | COMPLETE |
| LANE-SQLITE-MIGRATION | worker-subagent | `internal/adapters/storage/sqlite/repo.go`, `internal/adapters/storage/sqlite/repo_test.go` | Add `comments.summary` schema + migration/backfill + persistence read/write coverage. | COMPLETE |
| LANE-MCP-CONTRACT-MD | worker-subagent | `internal/adapters/server/common/*`, `internal/adapters/server/mcpapi/*` (scoped set from lane prompt) | Add summary-aware comment MCP contracts and markdown-rich schema guidance; populate capture comment overview from real data. | COMPLETE |
| LANE-TUI-MD-UX-PASSA | worker-subagent | `internal/tui/model.go`, `internal/tui/model_test.go`, `internal/tui/thread_mode.go`, `internal/tui/thread_mode_test.go`, `internal/tui/keymap.go`, `internal/tui/model_teatest_test.go`, `internal/tui/testdata/**` | Read-first markdown UX for details/comments, comment summary visibility, and preserved notification actionability. | COMPLETE |

### Checkpoint W4-01 - User Constraint Sync

- Added explicit constraints in `AGENTS.md`:
  - never operate outside `/Users/evanschultz/Documents/Code/hylla/tillsyn`,
  - MCP-only protocol validation (no HTTP/curl probes),
  - `just build` + `./till serve` permitted for runtime MCP checks,
  - never push unless explicitly requested.

### Checkpoint W4-02 - Lane Execution + Integrator Verification

- Worker lanes completed with non-overlapping lock scopes.
- Integrator reran package checks for non-TUI lanes:
  - `just test-pkg ./internal/domain` PASS
  - `just test-pkg ./internal/app` PASS
  - `just test-pkg ./internal/adapters/storage/sqlite` PASS
  - `just test-pkg ./internal/adapters/server/mcpapi` PASS
- Integrator reran TUI package gate after TUI lane:
  - `just test-pkg ./internal/tui` PASS
- Integrator full gates:
  - `just check` PASS
  - `just ci` PASS
- VHS sweep:
  - `just vhs` PASS (`board`, `regression_scroll`, `regression_subtasks`, `workflow`)

### Checkpoint W4-03 - Commit + Collaborative Worksheet Refresh

- Committed non-TUI wave integration:
  - `f28e9f3` — `Add markdown-first comment summary schema and MCP contracts`
- Created active collaborative worksheet for this wave:
  - `COLLAB_TEST_2026-03-02_DOGFOOD.md`
- Remaining closeout gate:
  - independent QA sign-off on code + docs for Wave 4,
  - collaborative user run of worksheet sections (`C1`-`C3`) and agent completion of section `C4` before final dogfood-ready mark.

### Checkpoint W4-04 - Live TUI Regression Remediation (Thread Editors + Notices Fit)

- Triggered by live collaborative run feedback:
  1. thread description shown as `(no description)` in notification-opened thread contexts,
  2. comments/details needed independent multiline editors,
  3. user-visible `…` truncation confusion and board/notices bottom-fit mismatch concerns.
- Integrated fixes in TUI scope:
  - `internal/tui/model.go`
  - `internal/tui/thread_mode.go`
  - `internal/tui/model_test.go`
  - `internal/tui/testdata/TestModelGoldenBoardOutput.golden`
  - `internal/tui/testdata/TestModelGoldenHelpExpandedOutput.golden`
- Validation evidence:
  - `just fmt` PASS
  - `just test-golden-update` PASS
  - `just test-pkg ./internal/tui` PASS
  - `just check` PASS
  - `just ci` PASS
- Worksheet synchronization:
  - updated `COLLAB_TEST_2026-03-02_DOGFOOD.md`:
    - `C1-04` control expectation updated (`ctrl+s` submit),
    - new `C1-05` details-editor validation row,
    - new `C3-04` alignment/truncation validation row,
    - overflow row logged for this regression.
- Status:
  - awaiting user collaborative rerun confirmation for C1/C3 rows before final wave closeout.

### Checkpoint W4-05 - Thread Workspace UX Redesign (Description/Comments/History)

- User UX directive implemented:
  1. top bordered description/details pane taking majority of thread viewport,
  2. bottom bordered comments pane at ~25% viewport with 2-line composer,
  3. right bordered owner + brief history pane,
  4. full-screen description markdown editor with live Glamour preview.
- Files changed:
  - `internal/tui/thread_mode.go`
  - `internal/tui/model.go`
  - `internal/tui/markdown_renderer.go`
- Validation:
  - `just fmt && just test-pkg ./internal/tui` FAIL (initial visibility assertion mismatch),
  - Context7 re-consult after failure (required),
  - remediation patch to actor-tagged/history summary visibility,
  - `just fmt && just test-pkg ./internal/tui` PASS,
  - `just check` PASS,
  - `just ci` PASS.
  - post-renderer-style update:
    - `just fmt && just test-pkg ./internal/tui` PASS
    - `just check && just ci` PASS
- Worksheet sync:
  - `COLLAB_TEST_2026-03-02_DOGFOOD.md` updated with:
    - `C1-02` pane-layout validation,
    - `C1-05` live-preview editor validation,
    - `C1-06` save/cancel semantics,
    - `C3-05` right-pane owner/history validation.
