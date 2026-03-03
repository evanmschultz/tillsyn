# COLLAB TEST 2026-03-02 (Dogfood Readiness)

## Scope

This worksheet is the active collaborative runbook for the markdown-first summary/details/comments wave.
Use section-by-section progression: do not advance until the current section is validated or a fix is landed and revalidated.

## Guardrails

- Repository scope only: `/Users/evanschultz/Documents/Code/hylla/tillsyn`
- Protocol validation in this worksheet is MCP-only (no HTTP/curl probes).
- Runtime for live validation: `just build` then `./till serve`.

## Baseline Agent-Only Validation (Completed)

| ID | Check | Status | Evidence |
|---|---|---|---|
| A-01 | `just test-pkg ./internal/domain` | PASS | integrator awaiter run (`2026-03-02`) |
| A-02 | `just test-pkg ./internal/app` | PASS | integrator awaiter run (`2026-03-02`) |
| A-03 | `just test-pkg ./internal/adapters/storage/sqlite` | PASS | integrator awaiter run (`2026-03-02`) |
| A-04 | `just test-pkg ./internal/adapters/server/mcpapi` | PASS | integrator awaiter run (`2026-03-02`) |
| A-05 | `just test-pkg ./internal/tui` | PASS | integrator awaiter run (`2026-03-02`) |
| A-06 | `just check` | PASS | integrator awaiter run (`2026-03-02`) |
| A-07 | `just ci` | PASS | integrator awaiter run (`2026-03-02`) |
| A-08 | `just vhs` (`board`, `regression_scroll`, `regression_subtasks`, `workflow`) | PASS | integrator awaiter run (`2026-03-02`) |
| A-09 | `gopls` workspace diagnostics | PASS | no blocking diagnostics (advisory hints may appear) |

## Collaborative Validation Queue (Run In Order)

### Section C1: Markdown-First Thread UX

| ID | Step | Expected | Status | Notes |
|---|---|---|---|---|
| C1-01 | Open task info (`enter`) then open thread (`c`) | Thread opens in read mode first (composer not auto-focused) | PENDING_USER | |
| C1-02 | Observe thread layout in read mode | Top bordered description/details pane consumes most height; bottom bordered comments pane consumes ~1/4; right bordered owner/history pane visible | PENDING_USER | |
| C1-03 | In thread read mode press `e` then `enter` | Details focus is active and `enter` transitions to edit form for underlying task/project | PENDING_USER | |
| C1-04 | Back to thread, press `i` | Composer activates as 2-line textarea; markdown paste works; submit on `ctrl+s` | PENDING_USER | |
| C1-05 | With details focused press `i` | Full-screen description editor opens with live Glamour markdown preview while typing | PENDING_USER | |
| C1-06 | In full-screen description editor press `ctrl+s` | Description saves and returns to thread details state; `esc` cancels unsaved edit | PENDING_USER | |

### Section C2: Summary/Details/Comments Visibility

| ID | Step | Expected | Status | Notes |
|---|---|---|---|---|
| C2-01 | Post a markdown comment with explicit summary via MCP | Thread shows `summary:` line plus rendered markdown body | PENDING_USER | |
| C2-02 | Open task info for same node | Task info shows recent comment preview lines with summary text | PENDING_USER | |
| C2-03 | Confirm long markdown details rendering | Markdown blocks/lists/headings remain readable in details/thread surfaces | PENDING_USER | |

### Section C3: Notifications/Global Panels Actionability

| ID | Step | Expected | Status | Notes |
|---|---|---|---|---|
| C3-01 | Focus notifications panel (`tab`) and move rows (`j/k`) | Section and row focus indicators stay stable | PENDING_USER | |
| C3-02 | Press `enter` on project warning/action row | Opens task info or scoped thread target (no dead-end action) | PENDING_USER | |
| C3-03 | Move to global notifications panel and `enter` | Deterministic cross-project open to task info or scoped thread | PENDING_USER | |
| C3-04 | Inspect board at wide layout with empty/short lists | Board columns + right-side panels share aligned bottom edge; no stray `…` truncation marker in panel body/footer fit | PENDING_USER | |
| C3-05 | Open thread from notifications and inspect right pane | Right pane shows owner identity and brief comment history entries with actor + summary | PENDING_USER | |

### Section C4: MCP Contract/Schema Behavior

| ID | Step | Expected | Status | Notes |
|---|---|---|---|---|
| C4-01 | `till.create_comment` without `summary` | Fails with deterministic required-arg error | PENDING_AGENT | |
| C4-02 | `till.create_comment` with markdown `summary` + `body_markdown` | Succeeds and returns summary/body | PENDING_AGENT | |
| C4-03 | Existing DB row migration check (legacy comments) | `comments.summary` populated from first non-empty body line | PENDING_AGENT | |
| C4-04 | `capture_state` includes non-zero `comment_overview` when comments exist | Comment counts/signal visible in MCP response | PENDING_AGENT | |

## Overflow / Unmapped Findings

| Timestamp | Surface | Observation | Expected | Severity | Mapped Section |
|---|---|---|---|---|---|
| 2026-03-03 | Board notifications layout | User observed unexplained `…` and perceived bottom-edge mismatch between board columns and right-side notifications stack. | Stable bottom alignment and no phantom truncation marker when no hidden content is expected. | High | C3-04 |

## Sign-Off

- Agent validation complete: `PENDING (C4 section)`
- User collaborative validation complete: `PENDING`
- Dogfood readiness verdict: `PENDING`
