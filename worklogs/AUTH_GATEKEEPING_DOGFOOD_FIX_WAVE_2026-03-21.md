# Auth Gatekeeping Dogfood Fix Wave

Created: 2026-03-21
Updated: 2026-03-21
Status: Implemented locally; local gates green with broader orchestrator scope support, pending commit/push/remote CI watch

## Purpose

This file is the lane and QA contract for the next auth gatekeeping remediation wave.

`PLAN.md` remains the canonical run ledger.
This file exists because the user explicitly requested one separate markdown file that captures all remaining auth/gatekeeping fixes before implementation.

## Locked User Findings

1. Auth approval should move away from the current cramped confirm-modal interaction and become a more intuitive review surface.
2. Auth approval should use one dedicated full-screen review surface rather than feeling like a confirm modal with extra fields bolted on.
3. The happy path should be simple:
   - default decision is `approve`,
   - approve should mostly be `approve -> confirm`,
   - cancel should always remain obvious.
4. Deny should branch cleanly:
   - choose `deny`,
   - write an optional explanation,
   - confirm or cancel.
5. Auth review should not depend on `h/l`-style confirm switching or other vim-style bindings that conflict with typing.
6. Once deny is selected, the flow should center the note editor first and only then move to explicit confirm/cancel.
7. Human-readable names must be used in auth review and path editing flows:
   - project names,
   - branch names,
   - phase names,
   - never raw ids as the primary human review label.
8. Path editing should not be free-text in the auth review surface.
9. Scope editing should happen in a dedicated picker flow that displays names while preserving the raw approved path underneath.
10. If the approved scope/path is narrowed or changed, that change must be visible in the requester-facing MCP response and audit/note trail.
11. MCP requesters need a real waiting state while human review is pending; the agent should not be left in a blind poll/hope loop.
12. Claiming or attaching to an existing approved auth/session context must not be a bypass:
   - requester-bound proof is required,
   - a different client/principal cannot silently adopt existing approval,
   - reuse/attachment must itself be explicitly user-approved if supported later.
13. Users need clear auth inventory and revocation surfaces:
   - list auth requests by project and globally,
   - list active auth sessions by project and globally,
   - revoke clearly and easily from CLI and TUI.
14. CLI and TUI must both support approval, denial, note capture, request/session listing, and revoke without forcing the user into another surface.
15. Orchestrator/subagent auth remains constrained:
   - subagents are single-project rooted only,
   - orchestrators may be single-project, multi-project, or general/global,
   - only orchestrators may receive multi-project or general/global scope.

## Wave Scope

In scope:
1. full-screen or otherwise clearly separated auth review flow
2. simple approve/default confirm flow
3. deny note flow with explicit confirm/cancel
4. removal of auth-review `h/l` confirm switching and related typing conflicts
5. dedicated scope picker with human-readable names
6. requester-visible approval-delta reporting
7. MCP waiting/progress behavior while human review is pending
8. anti-adoption gatekeeping for existing approvals/sessions
9. auth inventory and revoke UX in CLI and TUI
10. orchestrator/subagent scope-rule enforcement where needed for this dogfood wave
11. docs/test coverage for the above

Out of scope:
1. remote/team auth tenancy
2. unrelated product roadmap work
3. `till mcp-inspect`

## Acceptance Checklist

1. Auth review no longer feels like a generic confirm modal.
2. Approve is the default path and can be completed without unnecessary field traversal.
3. Deny requires or at least strongly centers the optional requester-facing note flow before final confirmation.
4. Typing in auth review inputs is not interfered with by `h/l`-style confirm bindings.
5. Scope editing uses a picker flow, not raw text entry as the primary UX.
6. Scope picker shows names, not ids, as the primary selection labels.
7. Auth review shows both:
   - a human-readable scope label,
   - the underlying raw approved path where relevant.
8. MCP requesters get a clear pending/waiting state while approval is unresolved.
9. Approved scope deltas are returned to the requester and visible in audit/state surfaces.
10. Existing approved auth contexts cannot be adopted by a different requester without explicit gatekeeping.
11. CLI can list auth requests by project and globally.
12. CLI can list sessions by project and globally.
13. CLI can revoke sessions clearly and deterministically.
14. TUI exposes pending auth work clearly enough to review and revoke where appropriate.
15. CLI and TUI both preserve the difference between requested scope and approved scope whenever approval narrows access.
16. Auth review defaults to approve without making the operator traverse a hidden decision menu.
17. Auth review and scope-picker labels use human-readable names as the primary surface, with raw paths available underneath for auditability.
18. Existing approved auth contexts cannot be adopted by a different requester without explicit gatekeeping.
19. Orchestrator/subagent scope rules are enforced and test-covered for the implemented surfaces.
20. `just check` passes.
21. `just ci` passes.
22. Final push is watched with `gh run watch --exit-status`.

## Lane Plan

### BLD-UX

Objective:
Implement the human-facing auth review and inventory UX.

Lock scope:
1. `internal/tui/**`

Out of scope:
1. `cmd/till/**`
2. `internal/app/**`
3. `internal/adapters/auth/**`
4. `internal/adapters/server/**`
5. `internal/domain/**`
6. `PLAN.md`

Primary targets:
1. dedicated auth review surface
2. approve/deny simplification
3. scope picker with human-readable labels
4. TUI auth inventory/revoke surfaces if needed in this lane

Worker checks:
1. `just test-pkg ./internal/tui`

QA lanes:
1. `QA-UX-1`
2. `QA-UX-2`

### BLD-POLICY

Objective:
Implement core auth policy, MCP waiting/continuation behavior, anti-adoption rules, and CLI auth inventory/revoke improvements.

Lock scope:
1. `cmd/till/**`
2. `internal/app/**`
3. `internal/domain/**`
4. `internal/adapters/auth/**`
5. `internal/adapters/server/common/**`
6. `internal/adapters/server/mcpapi/**`

Out of scope:
1. `internal/tui/**`
2. `PLAN.md`

Primary targets:
1. requester-bound waiting/claim behavior
2. anti-adoption guardrails
3. CLI request/session inventory filters
4. project/global revoke surfaces and semantics
5. orchestrator/subagent scope enforcement on the core auth path

Worker checks:
1. `just test-pkg ./cmd/till`
2. `just test-pkg ./internal/app`
3. `just test-pkg ./internal/domain`
4. `just test-pkg ./internal/adapters/auth/autentauth`
5. `just test-pkg ./internal/adapters/server/common`
6. `just test-pkg ./internal/adapters/server/mcpapi`

QA lanes:
1. `QA-POLICY-1`
2. `QA-POLICY-2`

## QA Contract

Every QA lane must:
1. review code and touched tests in its assigned build lane
2. compare landed behavior against both this file and `PLAN.md`
3. verify no acceptance item in this file is silently skipped
4. verify Context7 compliance is stated in the worker handoff
5. verify docs/comments/tests are adequate for the changed behavior
6. report blockers first, then residual risks, then pass/fail

## Integration Contract

The integrator must:
1. review each worker handoff before merging
2. keep `PLAN.md` as the canonical ledger
3. run `just check`
4. run `just ci`
5. commit logical checkpoints
6. push only after explicit user-approved progress in this conversation
7. run `gh run watch --exit-status` after push

## Post-Implementation Next Step

After this wave is green locally and in GitHub Actions, create and execute one full collaborative E2E dogfood worksheet covering:
1. native MCP auth request create
2. TUI human review
3. MCP waiting/progress
4. MCP claim/resume
5. authenticated mutation
6. revoke/fail-closed retest
7. orchestrator/subagent scoped auth
8. anti-adoption/bypass checks

## Implementation Checkpoint (2026-03-21)

Implemented in this wave:
1. TUI auth review now uses a dedicated full-screen review surface with visible decision controls instead of a generic confirm-modal fallback.
2. Approve is the default TUI review path, while deny stays note-first with explicit confirm/cancel.
3. Auth review no longer relies on auth-specific `h/l` confirm switching that interferes with typing.
4. Human-facing auth review and summary labels prefer human-readable scope names while preserving the raw path as the actual contract.
5. TUI auth inventory now distinguishes pending requests, resolved requests, and active approved sessions, and exposes session revoke directly from the inventory surface.
6. MCP claim/resume remains requester-bound and returns pending vs approved vs denied results cleanly.
7. The native MCP claim path rejects requester mismatch/adoption attempts instead of letting a different requester silently claim an approved context.
8. Auth scope paths now support `project/...`, `projects/...`, and `global`, with broader scopes reserved for orchestrators.
9. SQLite now provisions one hidden global auth backing project so global auth requests and global notifications can persist without leaking a fake project into normal project inventory.
10. CLI auth help/examples now document project, multi-project, and global scope paths explicitly.

Commands run and outcomes:
1. `just test-pkg ./internal/adapters/auth/autentauth`
Outcome: pass after seeding broader-scope project fixtures and adding hidden global auth backing support.
2. `just test-pkg ./internal/adapters/storage/sqlite`
Outcome: pass with hidden global auth project coverage.
3. `just test-pkg ./internal/app`
Outcome: pass.
4. `just test-pkg ./internal/tui`
Outcome: pass.
5. `just test-pkg ./cmd/till`
Outcome: pass after auth help/examples were expanded for `projects/...` and `global`.
6. `just test-pkg ./internal/domain`
Outcome: pass.
7. `just test-pkg ./internal/adapters/server/common`
Outcome: pass.
8. `just test-pkg ./internal/adapters/server/mcpapi`
Outcome: pass.
9. `just check`
Outcome: pass.
10. `just ci`
Outcome: pass; `internal/tui` coverage remained above the floor at 70.2%.

QA outcomes:
1. `QA-UX-1`
Outcome: pass after focused re-check cleared the stale multi-project/global label blocker.
2. `QA-UX-2`
Outcome: pass with residual risk on resolved-row detail ergonomics.
3. `QA-POLICY-1`
Outcome: pass after focused re-check cleared the stale broader-scope policy blocker.
4. `QA-POLICY-2`
Outcome: pass; no remaining blocker found for non-requester claim/adoption, broader-scope policy enforcement, CLI project/global inventory+revoke, or hidden global backing-project handling.

Remaining gaps before full dogfood closeout:
1. end-to-end collaborative dogfood worksheet and live retest evidence
2. final remote CI watch after push
