# P3 Markdown + Scroll + Expand Audit (Read-Only)

Timestamp (UTC): 2026-03-05T03:26:55Z

## Scope + Method

- Read scope used:
  - `internal/tui/model.go`
  - `internal/tui/model_test.go`
  - `internal/tui/markdown_renderer.go`
  - `internal/tui/description_editor_mode.go`
  - `internal/tui/thread_mode.go`
- Write scope used:
  - `worklogs/VEC_FIX_PLAN_P3_MARKDOWN_SCROLL_AUDIT.md` only
- No production/test code was changed.

## Commands Run

- `rg --files internal/tui | rg 'markdown|viewport|model\.go|thread_mode|description_editor'`
- `rg -n "threadMarkdown\.render|glamour|markdownRenderer|taskInfoDetails|threadScroll|blocked_reason|objective|acceptance_criteria|validation_plan|risk_notes|modeDescriptionEditor|previewViewport|descriptionPreview" internal/tui/model.go internal/tui/model_test.go internal/tui/markdown_renderer.go internal/tui/description_editor_mode.go internal/tui/thread_mode.go`
- `nl -ba internal/tui/{markdown_renderer.go,description_editor_mode.go,thread_mode.go,model.go,model_test.go} | sed -n '<target ranges>'`
- Context7:
  - `resolve-library-id` for `charmbracelet/glamour` and `charmbracelet/bubbles`
  - `query-docs` for Glamour renderer setup/reuse and Bubbles viewport APIs

---

## 1) Current Markdown Rendering Paths (Glamour) + Plain vs Rendered

### 1.1 Core renderer path

- Shared renderer is `markdownRenderer` (`internal/tui/markdown_renderer.go:10-14`).
- Markdown render path trims input, enforces minimum wrap width, lazily (re)creates `glamour.NewTermRenderer`, and calls `Render` (`internal/tui/markdown_renderer.go:17-46`).
- Current renderer options: `WithStandardStyle(styles.DarkStyle)` + `WithWordWrap(wrapWidth)` (`internal/tui/markdown_renderer.go:29-34`).

### 1.2 Where markdown is rendered today

- Description editor preview: `threadMarkdown.render(...)` in preview builder (`internal/tui/description_editor_mode.go:252-259`).
- Thread top description/details panel renders markdown (`internal/tui/thread_mode.go:124-129`).
- Thread comments panel renders comment body markdown (`internal/tui/thread_mode.go:321-327`).
- Task-info details viewport renders task description markdown (`internal/tui/model.go:11852-11868`, `internal/tui/model.go:12413-12417`).
- Task-info comments preview renders comment markdown (`internal/tui/model.go:12490-12496`).
- Task-info structured metadata renders markdown for `objective`, `acceptance_criteria`, `validation_plan`, `risk_notes` (`internal/tui/model.go:12526-12538`).
- Board card description line renders markdown when card descriptions are enabled (`internal/tui/model.go:12278-12285`).

### 1.3 Rich text that is still plain (not markdown-rendered or not expandable)

- Task/project form rows are `textinput.Model` rows in modal input forms (`internal/tui/model.go:2195-2204`, `internal/tui/model.go:2468-2480`, `internal/tui/model.go:13271-13288`).
- Only description rows are routed to full markdown editor; other text fields are not (`internal/tui/model.go:7481-7485`, `internal/tui/model.go:7511-7512`, `internal/tui/model.go:7542-7550`, `internal/tui/model.go:7554-7555`).
- Description field in forms is shown as compact first-line summary, not full markdown (`internal/tui/model.go:3087-3119`).
- `blocked_reason` in task-info is rendered as plain hint text (`internal/tui/model.go:12464-12472`).
- Dependency inspector details render description as truncated plain text (`internal/tui/model.go:12945-12964`).

---

## 2) Current Scrolling/Viewport Mechanics (Info/Thread/Modal)

### 2.1 Full-screen description editor

- Uses `descriptionPreview viewport.Model` with `SoftWrap=true` (`internal/tui/model.go:800-803`).
- Preview viewport layout/content sync is centralized (`internal/tui/description_editor_mode.go:261-281`).
- Preview-mode scrolling keys: `j/k`, arrows, `pgup/pgdown`, `home/end` (`internal/tui/model.go:6188-6205`).
- Preview-mode mouse wheel scrolls viewport (`internal/tui/model.go:9020-9033`).

### 2.2 Task-info overlay

- Uses `taskInfoDetails viewport.Model` with `SoftWrap=true` (`internal/tui/model.go:804-807`).
- Details viewport is bounded by min/max height and synced per task/resize (`internal/tui/model.go:11844-11880`).
- Task-info keys scroll details viewport (`internal/tui/model.go:6398-6422`).
- Mouse wheel scrolls details viewport in task-info mode (`internal/tui/model.go:9058-9070`).
- Viewport offset resets on open/close/path step-back (`internal/tui/model.go:11805-11827`, `internal/tui/model.go:11920-11927`).

### 2.3 Thread mode

- Comment list scrolling is manual via integer `threadScroll`, not a viewport model (`internal/tui/model.go:630`, `internal/tui/thread_mode.go:170-176`).
- Paging step is `threadViewportStep()` (`internal/tui/thread_mode.go:717-723`), used by `pgup/pgdown` handlers (`internal/tui/model.go:6323-6334`).
- Mouse wheel adjusts `threadScroll` (`internal/tui/model.go:9049-9055`).
- Top description/details panel is fit-clipped (`fitLines`) and not independently scrollable (`internal/tui/thread_mode.go:130-145`).

### 2.4 Create/edit modal surfaces

- Add/edit task and project modal forms render each field via `textinput` row, with no viewport/wrapped block region (`internal/tui/model.go:13271-13343`).
- Non-description long text fields therefore do not get wrap+scroll in form context.

---

## 3) Concrete Implementation Shape: Wrap + Scroll + Expand for Non-Date Text Fields in Unified Modal

### 3.1 Boundary-preserving approach

- Keep all changes inside TUI adapter layer (`internal/tui/*`); do not alter domain invariants or app service contracts.
- Reuse existing full-screen `modeDescriptionEditor` surface as the unified rich-text modal to minimize new state surface area (`internal/tui/model.go:82`, `internal/tui/model.go:2673-2773`, `internal/tui/model.go:2860-2932`).

### 3.2 Target fields for upgrade

- Task form long text fields:
  - `description` (already upgraded)
  - `blocked_reason`
  - `objective`
  - `acceptance_criteria`
  - `validation_plan`
  - `risk_notes`
- Keep date/enum/list-like fields (`due`, `priority`, labels/task-id CSV refs) as inline modal fields.

### 3.3 Recommended implementation steps

1. Generalize editor target state from description-only to rich-text field targeting.
   - Current target enum is only task/project/thread description (`internal/tui/model.go:86-93`).
   - Extend target model with field key/index so one editor surface can save to multiple fields.

2. Add generic launch/save helpers for rich-text fields.
   - Today `saveDescriptionEditor()` only writes task/project description or thread markdown (`internal/tui/model.go:2860-2876`).
   - Introduce field-aware save routing for task metadata text fields while retaining existing description behavior.

3. Route non-date long text task fields to unified modal.
   - Mirror existing description routing behavior currently wired at `taskFieldDescription` only (`internal/tui/model.go:7481-7485`, `internal/tui/model.go:7511-7512`).
   - Apply same trigger semantics (`enter`, `i`, typed-seed) to targeted long text fields.

4. Keep compact modal rows, but render a stable summary string for every rich-text field.
   - Reuse/extend summary helper pattern (`descriptionFormDisplayValue`) for non-description rich-text rows (`internal/tui/model.go:3087-3119`).

5. Make task-info long text display consistent and expandable.
   - Convert `blocked_reason` display from plain hint row to markdown-rendered section, matching other metadata sections (`internal/tui/model.go:12464-12472`, `internal/tui/model.go:12526-12538`).
   - Add one explicit “expand rich text” action from task-info to open unified modal in preview mode for currently focused details context (same scroll contract as description preview).

6. Optional second wave (if UX wants parity in thread read mode):
   - Add a read-only expand shortcut for thread top description/details panel, since panel body is currently clipped (`internal/tui/thread_mode.go:130-145`).

### 3.4 Why this is lowest-risk

- Existing editor already provides wrap/scroll/mouse behavior and has tests (`internal/tui/model.go:6188-6205`, `internal/tui/model.go:9020-9033`, `internal/tui/model_test.go:2615-2811`).
- Reusing that path avoids introducing a second markdown viewport subsystem.

---

## 4) Test Additions Needed

### 4.1 Existing coverage to preserve

- Description editor route + save/cancel from forms (`internal/tui/model_test.go:2481-2552`).
- Description preview scroll behavior (keyboard + mouse, narrow viewport) (`internal/tui/model_test.go:2688-2811`).
- Task-info description details viewport scrolling (`internal/tui/model_test.go:1569-1642`).
- Structured metadata sections present in task-info (`internal/tui/model_test.go:1462-1503`).

### 4.2 New tests to add

1. `TestModelTaskRichTextFieldRoutesToUnifiedEditor` (table-driven)
- fields: `blocked_reason`, `objective`, `acceptance_criteria`, `validation_plan`, `risk_notes`
- assert `enter`/`i` route to full-screen editor and preserve return focus.

2. `TestModelTaskRichTextFieldSaveCancelSemantics`
- assert save writes back to proper metadata field; cancel leaves field unchanged.

3. `TestModelTaskInfoBlockedReasonRendersAsMarkdownSection`
- assert blocked reason participates in markdown section rendering path (not plain hint-only row).

4. `TestModelTaskInfoRichTextExpandUsesUnifiedPreviewViewport`
- assert expand action opens unified modal in preview mode and supports keyboard + mouse scrolling.

5. `TestModelThreadDescriptionExpandPreview` (optional second wave)
- if thread read-mode expand is introduced, assert clipped panel can open full scrollable preview.

### 4.3 Coverage gap observed now

- No dedicated thread-mode tests for top description panel clipping/expand/scroll semantics in `internal/tui/thread_mode_test.go` (file currently focused on comment-target mapping only: `internal/tui/thread_mode_test.go:1-85`).

---

## Context7 Compliance Note

- Context7 was consulted before making external library behavior claims.
- Libraries consulted:
  - `/charmbracelet/glamour` (renderer setup/reuse, word wrap/style options)
  - `/charmbracelet/bubbles` (viewport sizing/content/scroll APIs used by current TUI implementation)
- Recommendations above align with repository usage and the referenced Context7 API patterns.
