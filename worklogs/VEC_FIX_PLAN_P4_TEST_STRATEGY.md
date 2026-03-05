# P4 Test Strategy: Unified Modal + Comments + Markdown
Timestamp: 2026-03-05 03:26:08 UTC
Scope: read-only analysis of `internal/tui/model_test.go` and in-file test helpers; no code edits.

## Commands and Outcomes
1. `cat Justfile`
Outcome: confirmed `just test-pkg` and `just test-golden` are the relevant package-level recipes.
2. `wc -l internal/tui/model_test.go`
Outcome: file size is 9,381 lines.
3. `rg -n "modal|modeTaskInfo|modeEditTask|comments|thread|markdown|metadata|description editor|details" internal/tui/model_test.go`
Outcome: identified modal/comments/markdown coverage anchors.
4. `rg -n "^func TestModel..." internal/tui/model_test.go` (thread/task-info/edit/description-editor tests)
Outcome: enumerated primary tests for info/edit modal + comments + markdown behavior.
5. `nl -ba internal/tui/model_test.go | sed -n '1,240p'`
Outcome: verified fake service + comment helper paths and test harness structure.
6. `nl -ba internal/tui/model_test.go | sed -n '940,1660p'`
Outcome: verified thread/task-info/comment-preview/metadata-prefill coverage details.
7. `nl -ba internal/tui/model_test.go | sed -n '2468,2795p'`
Outcome: verified description-editor modal behavior and task-info-to-editor transitions.
8. `nl -ba internal/tui/model_test.go | sed -n '4600,4695p'`
Outcome: verified render-mode overlay coverage for `New Task`, `Edit Task`, and `Task Info` modal titles/hints.
9. `nl -ba internal/tui/model_test.go | sed -n '9280,9381p'`
Outcome: verified reusable test helpers (`loadReadyModel`, `applyMsg`, `applyCmd`, `stripANSI`, `keyRune`).
10. `rg -n "older comments|comments unavailable|\(no comments yet\)|summary fallback" internal/tui/model_test.go`
Outcome: no direct assertions found for these task-info comment states.
11. `just test-pkg ./internal/tui`
Outcome: `ok   github.com/hylla/tillsyn/internal/tui	103.630s`.

## 1) Existing Coverage Map (Info/Edit Modal + Comments + Metadata Rendering)
1. Test harness and comment plumbing exist and are reusable.
Evidence: `internal/tui/model_test.go:23`, `internal/tui/model_test.go:43`, `internal/tui/model_test.go:105`, `internal/tui/model_test.go:130`, `internal/tui/model_test.go:9299`, `internal/tui/model_test.go:9310`, `internal/tui/model_test.go:9321`.

2. Thread modal comment flows are covered for read-first behavior, identity attribution/fallback, and posting.
Evidence: `internal/tui/model_test.go:955`, `internal/tui/model_test.go:1093`, `internal/tui/model_test.go:1140`, `internal/tui/model_test.go:1184`.

3. Thread details modal and transitions into edit/project-edit are covered.
Evidence: `internal/tui/model_test.go:1217`, `internal/tui/model_test.go:1263`, `internal/tui/model_test.go:1299`.

4. Task-info modal coverage includes comment preview, markdown details visibility, structured metadata sections, and details viewport scrolling.
Evidence: `internal/tui/model_test.go:1375`, `internal/tui/model_test.go:1429`, `internal/tui/model_test.go:1463`, `internal/tui/model_test.go:1570`, `internal/tui/model_test.go:2724`.

5. Edit-task metadata prefill and submit behavior exists for objective/acceptance/validation/risk, including one clear sentinel (`-`) path.
Evidence: `internal/tui/model_test.go:1506`, `internal/tui/model_test.go:1532`, `internal/tui/model_test.go:1546`, `internal/tui/model_test.go:1558`.

6. Full-screen markdown editor behavior is covered for task/project targets, preview mode, undo/redo, layout, and save/cancel loops.
Evidence: `internal/tui/model_test.go:2482`, `internal/tui/model_test.go:2518`, `internal/tui/model_test.go:2555`, `internal/tui/model_test.go:2581`, `internal/tui/model_test.go:2616`, `internal/tui/model_test.go:2689`, `internal/tui/model_test.go:2777`.

7. Overlay-level modal title/hint smoke checks exist for `New Task`, `Edit Task`, and `Task Info`.
Evidence: `internal/tui/model_test.go:4601`, `internal/tui/model_test.go:4640`, `internal/tui/model_test.go:4653`, `internal/tui/model_test.go:4675`.

## 2) Missing Tests for Requested Unified Modal/Comments/Markdown Behavior
1. Missing task-info comment-state coverage for `comments unavailable`, `(no comments yet)`, and older-comment truncation indicator.
Evidence gap: no direct assertions found by pattern scan for `older comments`, `comments unavailable`, `(no comments yet)` in `internal/tui/model_test.go`; nearest positive coverage is `internal/tui/model_test.go:1375`.

2. Missing explicit summary-fallback assertion when comment `Summary` is empty and preview should derive from markdown body.
Evidence gap: existing summary assertions only use explicit summary strings (`internal/tui/model_test.go:1052`, `internal/tui/model_test.go:1420`).

3. Missing unified backstack test that composes full flow in one scenario: `task info -> thread -> details modal -> description editor -> esc -> thread -> esc -> task info`.
Evidence: partial coverage exists, but split across separate tests (`internal/tui/model_test.go:1058`, `internal/tui/model_test.go:1217`, `internal/tui/model_test.go:2724`).

4. Missing table-driven metadata patch semantics for each structured field under `blank keep`, `"-" clear`, and `non-empty set`.
Evidence: single-case coverage exists (`internal/tui/model_test.go:1506`) but only `acceptance` clear is asserted (`internal/tui/model_test.go:1558`) and no per-field blank-preserve matrix exists.

5. Missing richer markdown-parity checks between thread/details/task-info surfaces (heading/list/code-link combinations and multiline body semantics).
Evidence: current assertions mostly check token presence for simple markdown snippets (`internal/tui/model_test.go:1047`, `internal/tui/model_test.go:1418`, `internal/tui/model_test.go:1424`, `internal/tui/model_test.go:1463`).

## 3) Proposed Table-Driven Test Matrix and Exact Test Names
### Add
1. `func TestModelTaskInfoCommentPreviewStates_TableDriven(t *testing.T)`
Matrix dimensions: `comment_list_state`, `summary_present`, `comment_count`, `body_markdown_shape`.
Core cases: list error, zero comments, one comment with summary, one comment without summary, >preview-limit comments.
Key assertions: shows `comments unavailable`, `(no comments yet)`, fallback summary text, and `+N older comments` indicator.

2. `func TestModelUnifiedTaskInfoThreadDetailsBackstack_TableDriven(t *testing.T)`
Matrix dimensions: `entry_path`, `details_opened`, `editor_opened`, `esc_count`.
Core cases: task-info->thread->esc; task-info->thread->details->esc; full chain through editor and back.
Key assertions: deterministic return mode and stable focused task/thread target.

3. `func TestModelEditTaskMetadataPatchSemantics_TableDriven(t *testing.T)`
Matrix dimensions: `field` (`objective|acceptance|validation|risk`), `input_mode` (`blank|dash|set`), `initial_value`.
Core cases: preserve on blank, clear on `-`, replace on non-empty.
Key assertions: post-submit metadata exactness per field.

4. `func TestModelThreadAndTaskInfoMarkdownParity_TableDriven(t *testing.T)`
Matrix dimensions: `markdown_fixture` (`heading/list/code/link/blockquote`), `surface` (`thread_description|task_info_details|comment_preview`).
Core cases: same markdown body rendered consistently across surfaces.
Key assertions: canonical tokens visible on each surface; no accidental loss during modal transitions.

### Update
1. Update `TestModelTaskInfoShowsCommentPreview` (`internal/tui/model_test.go:1375`) to keep it as smoke coverage and delegate edge states to the new table-driven preview-state test.
2. Update `TestModelThreadReadModeRequiresExplicitComposer` (`internal/tui/model_test.go:1140`) to include explicit negative assertion that save action while composer is inactive does not create comments.
3. Update `TestModelEditTaskMetadataFieldsPrefillAndSubmit` (`internal/tui/model_test.go:1506`) to remain as happy-path smoke while matrixed semantics move to the new table-driven patch test.

## 4) Suggested Package-Level Command List for Verification
1. `just test-pkg ./internal/tui`
Use for all modal/comments/markdown unit coverage verification.
2. `just test-golden`
Run when overlay rendering text layout or golden fixtures are touched.
3. `just test-pkg ./internal/tui`
Re-run as final package gate after test updates to ensure deterministic pass.
