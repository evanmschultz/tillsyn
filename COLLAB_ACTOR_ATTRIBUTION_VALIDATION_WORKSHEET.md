# Collaborative Actor Attribution Validation Worksheet

Created: 2026-03-17
Status: Planned
Owner: User + Codex

## 1) Purpose

Validate that actor attribution is persisted and rendered correctly for the next collaborative dogfood sweep across:
1. local user actions,
2. orchestrator actions,
3. subagent actions,
4. system-origin actions,
5. task/thread/activity/notices surfaces.

## 2) Guardrails

1. Run one step at a time.
2. Stop on first fail and remediate before proceeding.
3. Record the user's detailed finding, not just pass/fail.
4. Capture both persistence evidence and rendered-surface evidence.

## 3) Preconditions

| ID | Action | Expected | Status | Evidence | Notes |
|---|---|---|---|---|---|
| A-SETUP-01 | Bootstrap local identity with display name | Local user display name is non-empty | PENDING_USER | | Use current bootstrap display name. |
| A-SETUP-02 | Prepare one task with thread/comments visible in TUI | Task/thread/activity surfaces all reachable | PENDING_USER | | |
| A-SETUP-03 | Prepare one MCP-capable runtime for orchestrator/subagent mutations | Agent-written mutations available for inspection | PENDING_AGENT | | |

## 4) Local User Attribution

| ID | Step | Expected | Status | User Detailed Findings | Evidence | Notes |
|---|---|---|---|---|---|---|
| A-U-01 | Create/update one task from the TUI as the local user | Persisted task/change event uses local `actor_id`, `actor_name`, `actor_type=user` | PENDING_USER | | | |
| A-U-02 | Add one task/thread comment from the TUI | Comment owner renders the bootstrap display name, not legacy fallback ids | PENDING_USER | | | |
| A-U-03 | Inspect task info, thread, board activity/notices for the same mutation | All rendered surfaces agree on the local user display name | PENDING_USER | | | |

## 5) Orchestrator Attribution

| ID | Step | Expected | Status | User Detailed Findings | Evidence | Notes |
|---|---|---|---|---|---|---|
| A-O-01 | Perform one guarded MCP mutation as orchestrator | Persisted attribution uses orchestrator identity/name/type | PENDING_AGENT | | | |
| A-O-02 | Inspect TUI surfaces for that mutation | Task/thread/activity/notices all show orchestrator identity consistently | PENDING_USER | | | |

## 6) Subagent Attribution

| ID | Step | Expected | Status | User Detailed Findings | Evidence | Notes |
|---|---|---|---|---|---|---|
| A-S-01 | Perform one guarded MCP mutation as subagent/worker | Persisted attribution uses worker identity/name/type | PENDING_AGENT | | | |
| A-S-02 | Inspect TUI surfaces for that mutation | Task/thread/activity/notices all show worker identity consistently | PENDING_USER | | | |

## 7) System Attribution

| ID | Step | Expected | Status | User Detailed Findings | Evidence | Notes |
|---|---|---|---|---|---|---|
| A-Y-01 | Trigger one system-origin mutation or seeded system event | Persisted attribution uses `actor_type=system` with system display name | PENDING_AGENT | | | |
| A-Y-02 | Inspect TUI surfaces for that event | Rendered surfaces show system ownership consistently | PENDING_USER | | | |

## 8) Failure Ledger

| Finding ID | Section/Step | Severity | User Detailed Findings | Agent Rephrase | Decision | Status | Evidence |
|---|---|---|---|---|---|---|---|

## 9) Validation Record

| Fix ID | Commands / Checks | QA1 | QA2 | User Retest | Final Status |
|---|---|---|---|---|---|

## 10) Discussion Log

| Timestamp (local) | Speaker | Detailed Statement | Notes |
|---|---|---|---|
| 2026-03-17 00:00 | Agent | Created this worksheet to cover future collaborative identity rendering and persistence checks for local user, orchestrator, subagent, and system actors. | Seeded from the active collaborative remediation discussion. |
