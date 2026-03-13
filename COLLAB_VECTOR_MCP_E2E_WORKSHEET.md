# Collaborative Vector + MCP E2E Worksheet

Created: 2026-03-04
Status: Active
Owner: User + Codex (single-writer updates by Codex)

## 1) Purpose

Validate the vector-search wave end-to-end together, with:
1. user-driven TUI validation,
2. agent-driven MCP validation,
3. shared pass/fail decisions,
4. strict fail-stop remediation loop before moving to the next item.

## 2) Locked Collaboration Protocol

For every failing step, use this exact sequence before any next step:
1. Stop progression immediately on first fail.
2. Record the user's exact wording in this worksheet.
3. Spawn subagents to investigate and propose fix options.
4. Discuss options with user and reach explicit consensus.
5. Implement fix with scoped worker subagents.
6. Run required package tests (`just test-pkg ...`) and required gates as applicable.
7. Run independent QA pass 1 and QA pass 2.
8. Re-run the same collaborative step and capture fresh evidence.
9. Only then mark that step complete and proceed.

## 3) Evidence Destinations

Primary collaborative evidence files:
1. `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md` (this file)
2. `COLLAB_TEST_2026-03-02_DOGFOOD.md` (collaborative dogfood record)
3. `MCP_DOGFOODING_WORKSHEET.md` (MCP transport corroboration)

Command/test artifacts:
1. `.tmp/vec-collab-e2e-<timestamp>/...`
2. `.tmp/vec-wavef-evidence/20260303_175936/...`
3. `.tmp/vec-wavef-evidence/20260303_180827/...`

## 4) Session Setup

### 4.1 Agent Preflight

| ID | Command | Expected | Status | Evidence | Notes |
|---|---|---|---|---|---|
| S-01 | `just build` | Build succeeds | PASS | `.tmp/vec-collab-e2e-20260304_191626/just_build.txt` | Initial sandbox run hit cache permission; rerun outside sandbox passed. |
| S-02 | `./till serve --help` | Serve help is visible | PASS | `.tmp/vec-collab-e2e-20260304_191626/till_serve_help.txt` | Help output includes serve flags and endpoints. |
| S-03 | `just check` | PASS | PASS | `.tmp/vec-collab-e2e-20260304_191626/just_check.txt` | Cross-package check suite passed for this session. |

### 4.2 User Runtime Setup

| ID | Action | Expected | Status | Evidence | Notes |
|---|---|---|---|---|---|
| U-01 | Start TUI runtime (`just run` or built binary flow) | App opens without panic | PASS | User quotes: "the server is running and the tui is running." / "ok, I started the server, and the tui, with a fresh db. lets do our tests." | Session kickoff confirmation captured verbatim. |
| U-02 | Open project/board that contains vector-indexed tasks | Board loads with tasks | PENDING_USER | | Awaiting explicit board/task visibility confirmation. |

### 4.3 Fresh-DB Fixture Seed (Required Before T1/T2)

| ID | Action | Expected | Status | Evidence | Notes |
|---|---|---|---|---|---|
| F-01 | Create one project (`p` then `N`) | Project appears in picker/board | PENDING_USER | | |
| F-02 | Create one task (`n`) with non-empty title + description | Task appears in first column | PENDING_USER | | |
| F-03 | Edit same task (`e`) and fill metadata fields (`objective`, `acceptance_criteria`, `validation_plan`, `risk_notes`, `blocked_reason`) | Save succeeds and values persist | PENDING_USER | | |
| F-04 | Re-open task info (`i` or `enter`) | Metadata fields visibly render | PENDING_USER | | |

## 5) Collaborative TUI E2E Queue (Run In Order)

### Section T1: Metadata Accessibility (Wave E)

| ID | Step | Expected | Status | User Detailed Findings | Evidence | Notes |
|---|---|---|---|---|---|---|
| T1-01 | Open edit task form for existing task | `objective`, `acceptance_criteria`, `validation_plan`, `risk_notes`, `blocked_reason` are visible/editable | PENDING_USER | | | |
| T1-02 | Save updates for all above fields | Values persist after save/re-open | PENDING_USER | | | |
| T1-03 | Open task info overlay | All above fields render in info view | PENDING_USER | | | |

### Section T2: TUI Search Behavior

| ID | Step | Expected | Status | User Detailed Findings | Evidence | Notes |
|---|---|---|---|---|---|---|
| T2-01 | Run task search for known keyword in title/description | Relevant matches returned and stable ordering | PENDING_USER | | | |
| T2-02 | Run search for metadata text (`objective` etc.) | Match includes task containing metadata phrase | PENDING_USER | | | |
| T2-03 | Navigate multi-result search pages | Deterministic behavior with explicit limit/offset defaults | PENDING_USER | | | |
| T2-04 | Use dependency inspector search | Results remain consistent with explicit mode/sort/limit/offset defaults | PENDING_USER | | | |

### Section T3: Regression Safety in TUI Flows

| ID | Step | Expected | Status | User Detailed Findings | Evidence | Notes |
|---|---|---|---|---|---|---|
| T3-01 | Edit task metadata, then open thread/comments flow | No overlay/layout regression | PENDING_USER | | | |
| T3-02 | Switch projects/scopes and repeat search | No stale/incorrect search carryover | PENDING_USER | | | |

## 6) Collaborative MCP E2E Queue (Run In Order)

### Section M1: Tool Schema + Guardrails

| ID | MCP Check | Expected | Status | User Detailed Findings | Evidence | Notes |
|---|---|---|---|---|---|---|
| M1-01 | `till.search_task_matches` tool schema inspection | Contains `mode`, `sort`, `levels`, `kinds`, `labels_any`, `labels_all`, `limit`, `offset` | PENDING_AGENT | | | |
| M1-02 | Schema numeric constraints | `limit` default 50, min 0, max 200; `offset` default 0, min 0 | PENDING_AGENT | | | |
| M1-03 | Invalid pagination input check | Invalid values fail with deterministic validation behavior | PENDING_AGENT | | | |

### Section M2: Query Mode Behavior

| ID | MCP Check | Expected | Status | User Detailed Findings | Evidence | Notes |
|---|---|---|---|---|---|---|
| M2-01 | `mode=keyword` call | Returns lexical matches for query | PENDING_AGENT | | | |
| M2-02 | `mode=semantic` call | Returns semantic matches or keyword fallback when semantic unavailable | PENDING_AGENT | | | |
| M2-03 | `mode=hybrid` call | Combined behavior with stable ranking response shape | PENDING_AGENT | | | |

### Section M3: Filters, Sorting, Pagination

| ID | MCP Check | Expected | Status | User Detailed Findings | Evidence | Notes |
|---|---|---|---|---|---|---|
| M3-01 | `levels` + `kinds` filters | Result set constrained correctly | PENDING_AGENT | | | |
| M3-02 | `labels_any` + `labels_all` filters | Taxonomy filter behavior correct | PENDING_AGENT | | | |
| M3-03 | `sort=rank_desc|title_asc|created_at_desc|updated_at_desc` | Sort order deterministic and valid | PENDING_AGENT | | | |
| M3-04 | `limit` + `offset` paging calls | Stable slices of total candidate set | PENDING_AGENT | | | |

## 7) Findings + Remediation Ledger

| Finding ID | Section/Step | Severity | User Detailed Findings | Agent Rephrase | Decision | Status | Evidence |
|---|---|---|---|---|---|---|---|
| FR-001 | T1/T3 (UI architecture + info modal) | High | User requested: unified modal/component across create/edit/display for all node types; markdown (Glamour) rendering for rich text; scrollable/wrapping long fields; expandable text sections (description-like behavior) for non-date textual fields; comments list restored in info modal with ownership metadata and relevant fields. | Consolidated node modal framing (`nodeModalBoxStyle`), added full task-info body viewport scroll wiring, kept Glamour markdown rendering paths, exposed full comments list with owner metadata rows, and made node-type-aware info/edit headers. | Accepted for immediate remediation | READY_FOR_USER_RETEST | Code: `internal/tui/model.go`, Tests: `internal/tui/model_test.go`, QA: agents Darwin + Ampere |
| FR-002 | T1/T3 (modal parity follow-up) | High | User reported info and edit modals still differ and requested both be full-screen (description-preview sized) and use the same modal/component with edit-only interaction differences. | Routed task info and add/edit node modes through the same full-screen viewport modal renderer (`renderNodeModalViewport`) with shared frame style and dimensions. | Accepted for immediate remediation | READY_FOR_USER_RETEST | Code: `internal/tui/model.go`; validation: `just test-pkg ./internal/tui`, `just check`, `just ci`; QA: Darwin + Ampere |
| FR-003 | T1/T3 (exact info/edit section parity) | High | User confirmed full-screen parity was close but requested edit modal be fully overhauled to match info modal design exactly, reusing the same component/section layout and converting sections to editable form (not a separate edit design). | Replaced legacy flat edit-form rendering with section-based task/project bodies routed through shared full-screen node modal renderer, aligned task edit sections with task info ordering, added edit comments metadata section visibility, and refreshed tests to assert section-order parity via helper body content. | Accepted for immediate remediation | READY_FOR_USER_RETEST | Code: `internal/tui/model.go`, `internal/tui/model_test.go`; validation: `just test-pkg ./internal/tui`, `just check`, `just ci`; QA: Darwin + Ampere |
| FR-004 | T1/T3 (edit/info UX policy + interaction cleanup) | High | User requested final cleanup before resuming collaborative steps: info/edit should be full-page screens (not modal framing), description directly under title with navigation aligned to display order, remove inherited/effective labels blocks, rename dependency placeholder text to `csv task`, simplify edit hotkeys (`ctrl+s` save, no edit `d`/`ctrl+r`/`ctrl+s` subtask), show subtasks in both info/edit with edit-mode create path, open `blocked_reason/objective/acceptance_criteria/validation_plan/risk_notes` in full-screen editor, wrap top/bottom edit navigation, make info description height auto-grow with cap, and split info metadata onto separate priority/due/labels lines. | Implemented shared full-page node surface rendering for task/project info+edit modes, reordered section layout (title -> description first), removed inherited/effective label rendering blocks, switched placeholders to `csv task`, refactored edit-key routing (`ctrl+s` save + `enter/e` action routing with typing-safe labels/dependency handling), added virtual subtasks/resources edit sections, routed metadata fields through shared markdown editor path, enabled edit wrap navigation, auto-sized info description viewport to content height with max cap, and split info metadata lines. | Accepted for immediate remediation | READY_FOR_USER_RETEST | Code: `internal/tui/model.go`, Tests: `internal/tui/model_test.go`; validation: `just test-pkg ./internal/tui`, `just check`, `just ci`; QA: Hooke + Galileo (final pass after follow-up regression fix + help/hint sync) |
| FR-005 | T1/T3 (follow-up UX polish before collaborative rerun) | High | User follow-up requested: clarify blank-field behavior, move `kind/state/complete/mode` into info/edit header, make resources and subtasks selectable/editable rows, stop `j/k` boundary wrap so `k` can be typed in title, and add border + persistent `TILLSYN` header on full-page node screens. | Implemented header metadata lines for info/edit node surfaces, switched edit traversal to arrow-wrap only while preserving typing for `j/k`, added selectable row cursors for subtasks/resources with create/edit flows (`enter/e`), clarified blank-value guidance copy, restored bordered node surface framing, and kept `TILLSYN` header visible on full-page node views; added direct tests for header/border composition and blank-guidance text. | Accepted for immediate remediation | READY_FOR_USER_RETEST | Code: `internal/tui/model.go`, Tests: `internal/tui/model_test.go`; validation: `just test-pkg ./internal/tui`, `just check`, `just ci`; QA: Feynman + Planck final re-audit PASS against explicit user requirements |

## 8) Subagent Fix Planning Record (Populate On First Failure)

| Fix ID | Finding ID | Planning Subagents | Candidate Options | User-Selected Option | Notes |
|---|---|---|---|---|---|
| FX-001 | FR-001 | Planning: `worklogs/VEC_FIX_PLAN_P1_INFO_MODAL_AUDIT.md`, `worklogs/VEC_FIX_PLAN_P2_EDIT_MODAL_AUDIT.md`, `worklogs/VEC_FIX_PLAN_P3_MARKDOWN_SCROLL_AUDIT.md`, `worklogs/VEC_FIX_PLAN_P4_TEST_STRATEGY.md` | Option A selected: add reusable node modal frame + full task-info content viewport + full comments metadata list + node-type-aware headers; validate with targeted + full gates. | Option A (consensus by user directive to proceed) | No schema/migration changes required for this UX fix scope. |
| FX-002 | FR-002 | Follow-up QA audit only (Darwin + Ampere) | Option A selected: share one full-screen modal renderer between task info and add/edit node modes, preserving mode-specific interactions. | Option A | Focused UI parity change; no storage/schema impact. |
| FX-003 | FR-003 | Follow-up QA audit only (Darwin + Ampere) | Option A selected: keep one shared full-screen modal renderer and replace edit-body internals with sectioned info-style structure; add parity assertions in tests. | Option A | UI architecture parity refinement; no storage/schema impact. |
| FX-004 | FR-004 | Parallel workers: `LANE-UX-EDIT-NODE-TESTS`, `LANE-UX-EDIT-NODE-CODE-R2`; dual QA: Hooke + Galileo | Option A selected: keep full-page shared node component, finish interaction-policy cleanup with typing-safe `e` behavior, then align tests/help text and rerun full gates. | Option A (consensus by user directive to proceed immediately) | Included one follow-up remediation loop after QA-found `e` regression; resolved before final QA sign-off. |
| FX-005 | FR-005 | Parallel QA re-audit: Feynman + Planck (`019cbd28-39ce-7411-bca7-8366d7183f73`, `019cbd28-3bf8-7aa1-9de0-2047c96776a6`) | Option A selected: apply user-requested polish exactly (header metadata move, selectable subtask/resource rows, arrow-only boundary wrap, border + persistent TILLSYN header) and harden with focused tests/help-copy sync. | Option A (explicit user directive: "fix all of this please") | One QA pass initially flagged overlay-guardrail conflict; resolved as intentional by explicit user requirement for full-page node surfaces, then both QA passes signed off. |

## 9) Validation After Fix Record

| Fix ID | Package Tests | QA Pass 1 | QA Pass 2 | Collaborative Re-test Step(s) | Final Result |
|---|---|---|---|---|---|
| FX-001 | `just test-pkg ./internal/tui` PASS; `just check` PASS; `just ci` PASS | PASS (`019cbc1c-aaaa-75f1-8be4-d333d98e6e3d` Darwin) | PASS (`019cb5c7-4141-76f2-a48e-70bb889ed054` Ampere) | Pending user collaborative rerun of FR-001 scenario | READY_FOR_USER_RETEST |
| FX-002 | `just test-pkg ./internal/tui` PASS; `just check` PASS; `just ci` PASS | PASS (`019cbc1c-aaaa-75f1-8be4-d333d98e6e3d` Darwin) | PASS (`019cb5c7-4141-76f2-a48e-70bb889ed054` Ampere) | Pending user collaborative rerun of modal parity/full-screen scenario | READY_FOR_USER_RETEST |
| FX-003 | `just test-pkg ./internal/tui` PASS; `just check` PASS; `just ci` PASS | PASS (`019cbc1c-aaaa-75f1-8be4-d333d98e6e3d` Darwin) | PASS (`019cb5c7-4141-76f2-a48e-70bb889ed054` Ampere) | Pending user collaborative rerun of exact info/edit section parity scenario | READY_FOR_USER_RETEST |
| FX-004 | `just test-pkg ./internal/tui` PASS; `just check` PASS; `just ci` PASS | PASS (`019cbcdc-df40-7113-b70c-a03ebd889523` Hooke, final) | PASS (`019cbcdc-e153-7b92-8696-a8a5025ad824` Galileo, final) | Pending user collaborative rerun of updated edit/info UX policy scope (T1-01 rerun) | READY_FOR_USER_RETEST |
| FX-005 | `just test-pkg ./internal/tui` PASS; `just check` PASS; `just ci` PASS | PASS (`019cbd28-39ce-7411-bca7-8366d7183f73` Feynman, final re-audit) | PASS (`019cbd28-3bf8-7aa1-9de0-2047c96776a6` Planck, final re-audit) | Pending user collaborative rerun of the same blocked T1-01 scope after follow-up polish | READY_FOR_USER_RETEST |

## 10) Sign-Off

- Collaborative TUI queue complete: `PENDING`
- Collaborative MCP queue complete: `PENDING`
- Open High findings: `PENDING`
- Open Medium findings: `PENDING`
- Final user+agent dogfood verdict: `PENDING`

## 11) Discussion Log

| Timestamp (local) | Speaker | Detailed Statement | Notes |
|---|---|---|---|
| 2026-03-04 19:31 | User | "yeah, but what about the extra fields for the embedings? the schema itself had to be changed!" | Clarification requested on in-place schema migration behavior for embeddings. |
| 2026-03-04 19:34 | User | "where are we saving the log file, is relative to the tillsyn dir or in the application support dir?" | Clarification requested on dev file log path resolution. |
| 2026-03-04 19:37 | User | "ok, I started the server, and the tui, with a fresh db. lets do our tests." | Collaborative test execution start on fresh DB. |
| 2026-03-04 19:43 | User | Node info/edit UX should be unified and reusable across all node types; rich text fields should support wrapping+scrolling+expand/markdown rendering; comments list with ownership metadata must be visible in info modal; prefer one shared display/input modal style with interaction-mode differences only. | Collaborative failure report; remediation loop initiated. |
| 2026-03-05 11:05 | Agent | Implemented FR-001 remediation in TUI code and tests: reusable node modal frame, task-info full-body viewport scroll wiring, full comment metadata list in task-info, and node-type-aware info/edit headers. | Awaiting user collaborative rerun confirmation for closure. |
| 2026-03-05 11:19 | Agent | Dual QA passes completed by subagents Darwin and Ampere; both approved user retest with one low test-gap note, then test assertions were tightened for owner metadata rows. | FR-001 moved to `READY_FOR_USER_RETEST`. |
| 2026-03-05 12:03 | User | Requested follow-up: info/edit modals must both be full-screen and be the same modal/component; edit mode should only differ by interactive ability to update fields. | Logged as FR-002 remediation scope. |
| 2026-03-05 12:22 | Agent | Implemented shared full-screen node modal renderer for task info + add/edit node flows and revalidated with package/full gates and dual QA passes. | FR-002 moved to `READY_FOR_USER_RETEST`. |
| 2026-03-05 13:18 | User | Requested stricter parity: edit modal should be completely overhauled to use the exact same info-modal design, reusing components and converting each section to editable form (not a separate edit design). | Logged as FR-003 remediation scope. |
| 2026-03-05 15:02 | Agent | Implemented FR-003 section-parity refactor (task/project edit section bodies via shared modal renderer), added section-order parity assertions in TUI tests, reran package/full gates, and completed dual QA passes with Darwin + Ampere. | FR-003 moved to `READY_FOR_USER_RETEST`. |
| 2026-03-05 00:08 | User | Requested pre-test UX cleanup: full-page info/edit screens, description ordering parity, remove inherited/effective labels display, simplify edit hotkeys, rename `csv task ids`, metadata editor reuse, wrap-around edit navigation, subtasks visibility/create path, and info metadata/description sizing refinements. | Logged as FR-004 remediation scope. |
| 2026-03-05 00:39 | Agent | Implemented FR-004 in `internal/tui/model.go` + `internal/tui/model_test.go`, reran package/full gates, and ran dual QA; QA initially flagged `e`-key regression and help-copy drift, then follow-up fix/remediation loop completed with fresh green gates. | FX-004 remediation loop completed; awaiting user rerun. |
| 2026-03-05 00:50 | Agent | Final dual QA recheck after follow-up fixes: Hooke PASS and Galileo PASS; scope moved to collaborative user retest gate. | FR-004/FX-004 set to `READY_FOR_USER_RETEST`. |
| 2026-03-05 16:12 | User | Asked for clarification on `blank values keep current task value`, requested header metadata placement in info/edit, reported resources not focusable/editable, asked to remove `j/k` boundary behavior so typing `k` works, and asked to test bordered node pages while keeping `TILLSYN` header. | Logged as FR-005 follow-up remediation scope. |
| 2026-03-05 16:28 | User | Confirmed final direction: “it feels like a blank field should make it clear if wasn't empty before… let's do the border and the tillsyn header! fix all of this please!” | Explicit implementation go-ahead for FR-005. |
| 2026-03-05 17:04 | Agent | Implemented FR-005 follow-up polish in TUI code + tests (header metadata, selectable subtask/resource rows, arrow-only boundary wrap with typing-safe `j/k`, bordered full-page node surface with persistent `TILLSYN` header, clearer blank-value copy) and reran package/full gates. | Awaiting collaborative user rerun of T1-01 before forward progression. |
| 2026-03-05 17:18 | Agent | Ran parallel QA re-audit with Feynman + Planck. Initial Planck note about overlay-vs-replacement was classified as intentional per explicit user requirement for full-page screens; final re-audit PASS/PASS against user requirements. | FX-005 moved to `READY_FOR_USER_RETEST`. |
