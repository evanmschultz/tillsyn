# T1-02 UX And Attribution Remediation Tracker

Status: In progress
Step: `T1-02`
Opened: 2026-03-17

## Objective

Close the remaining `T1-02` UX and attribution gaps without opening roadmap work:

1. make task-screen help and quick actions explain subtask/resource/comment flows clearly,
2. keep task state movement on `[` / `]` and teach it in `? help` instead of adding a new visible control,
3. make save-dependent action rows explicit in `new task`,
4. use `project root -> bootstrap default root` for task resource attach/display fallbacks,
5. remove visible local-user `tillsyn-user` rendering when bootstrap identity already has a display name,
6. preserve correct actor attribution for user, orchestrator, subagent, and system writes,
7. add future collaborative attribution coverage in a worksheet-style markdown file.

## Acceptance Criteria

- `? help` for task info/edit explains:
  - subtask flow,
  - resource flow,
  - comments flow,
  - task move/state flow via `[` / `]`,
  - focused quick actions via `.`.
- `.` opens focused/contextual quick actions without bloating bottom help.
- Task edit row-lists are clearer:
  - `subtasks:` and `resources:` default to the first existing item when present,
  - `left/right` still traverses rows including `+ create` / `+ attach`.
- `new task` clearly disables save-dependent rows until the task exists.
- Resource attach uses project root when configured, otherwise the bootstrap default path.
- Resource display remains project-root-relative when possible.
- Comment/task activity owner labels render the configured local user display name instead of legacy `tillsyn-user`.
- Mutation persistence stores actor identity/name/type correctly for non-user actors too.
- New collaborative worksheet coverage exists for attribution validation.

## Scope

### TUI lane

Allowed files:
- `internal/tui/model.go`
- `internal/tui/thread_mode.go`
- `internal/tui/model_test.go`
- `internal/tui/description_editor_mode.go`
- `internal/tui/trace.go`

### App/storage lane

Allowed files:
- `internal/app/service.go`
- `internal/app/service_test.go`
- `internal/adapters/storage/sqlite/repo.go`
- `internal/adapters/storage/sqlite/repo_test.go`

### Tracker/docs lane

Allowed files:
- `PLAN.md`
- `COLLAB_VECTOR_MCP_E2E_WORKSHEET.md`
- `COLLAB_T1_02_UX_ATTRIBUTION_FIX_TRACKER.md`
- `COLLAB_ACTOR_ATTRIBUTION_VALIDATION_WORKSHEET.md`

## Investigation Notes

- Explorer `Raman` (`019cfe4a-2412-7f60-9dd7-15692f03190f`) confirmed task resource attach is blocked when the current project lacks a configured project root and that current tests cover direct focus manipulation more than real user discoverability.
- Explorer `McClintock` (`019cfe4a-26c1-7242-beeb-c8368e12635d`) confirmed there is no explicit task-state control beyond `[` / `]`, that task-info comment entry is details-first, and that help text does not fully explain the current subtask/move flow.
- Explorer `Franklin` (`019cfe4a-2966-7601-bab6-bb2f368f4fd2`) confirmed bootstrap/config already persist `display_name`, but comment/thread read paths still render stored legacy actor tuples literally; write-side actor-name persistence is also incomplete for some mutations.

## Lane Status

| Lane | Owner | Scope | Status | Notes |
| --- | --- | --- | --- | --- |
| `W0` | Orchestrator | TUI hotspot integration | IN_PROGRESS | Shared task-screen help, quick actions, row-list clarity, resource-root fallback, owner-label rendering, tests. |
| `W1` | Hegel (`019cfe73-714b-7bd0-8dc1-3c998502b2e6`) | App/storage attribution persistence | COMPLETE | Persisted actor_name with local/agent/system mutations and strengthened service/sqlite attribution tests. |
| `QA1` | Archimedes (`019cfe8c-7aa5-7bc0-9f61-da2a58fbd6f1`) | Code/UI review | COMPLETE | Initial medium finding: task move/state leaked into `.` quick actions; follow-up removal verified clean. |
| `QA2` | Nietzsche (`019cfe88-c986-7500-b9c2-082171d37b4d`) | Tracker/docs review | COMPLETE | No findings; tracker/docs/test state coherent for FR-017. |

## Validation Plan

Worker/package scope:

- `just test-pkg ./internal/tui`
- `just test-pkg ./internal/app`
- `just test-pkg ./internal/adapters/storage/sqlite`

Integrator/full scope:

- `just test-golden`
- `just check`
- `just ci`

## Future Collaborative Coverage To Add

The next collaborative worksheet extension must cover attribution correctness across:

1. local user actions rendered as the bootstrap display name,
2. orchestrator-generated mutations rendered with orchestrator identity,
3. subagent-generated mutations rendered with worker identity,
4. system-origin mutations rendered as system,
5. task/thread/activity/notices surfaces agreeing on actor display.

Planned worksheet:
- `COLLAB_ACTOR_ATTRIBUTION_VALIDATION_WORKSHEET.md`
