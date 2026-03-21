# AUTH_UX_DOGFOOD_WAVE_PLAN_2026-03-20

Timestamp (UTC): 2026-03-20T23:58:00Z  
Scope: docs-only planning checkpoint for the next dogfood auth UX wave. No production code edits.

## Objective
- Convert the active next-step discussion into one implementation-ready plan that:
  - copies the strongest `blick` shell auth/grant lifecycle patterns,
  - preserves `tillsyn`'s TUI-first approval UX for normal users,
  - locks one explicit build lane plus two explicit QA lanes,
  - keeps `PLAN.md` as the active run authority while this file records the split checkpoint details.

## Commands And Outcomes
1. `git status --short`
Outcome: clean worktree before docs edits.
2. `find worklogs -maxdepth 2 -type f | sort`
Outcome: confirmed existing split-worklog naming patterns and that `worklogs/` is active in this repo.
3. `sed -n '1,260p' PLAN.md`
Outcome: confirmed active run constraints and current auth/runtime contract.
4. `sed -n '1,220p' README.md`
Outcome: confirmed current public runtime/auth summary and in-progress items.
5. `sed -n '1,220p' Justfile`
Outcome: confirmed `just check` and `just ci` remain final gates; docs-only checkpoint does not require test execution.
6. `gh repo clone evanmschultz/blick .tmp/blick`
Outcome: cloned `blick` for local reference.
7. `rg -n "fang|auth|grant|approve|approval|session|request" .tmp/blick/go.mod .tmp/blick/cmd .tmp/blick/internal .tmp/blick -g '!**/.git/**'`
Outcome: identified the relevant `blick` CLI and auth/grant files.
8. `sed -n '1,260p' .tmp/blick/cmd/blick/main.go`
Outcome: confirmed Fang-wrapped root CLI structure and shared-DB auth service setup.
9. `sed -n '1,260p' .tmp/blick/cmd/blick/main_test.go`
Outcome: confirmed help/examples are treated as test-enforced product behavior.
10. `sed -n '1,360p' .tmp/blick/cmd/blick/auth_cmd.go`
Outcome: confirmed structured `auth` command tree with examples and required flags.
11. `sed -n '360,760p' .tmp/blick/cmd/blick/auth_cmd.go`
Outcome: confirmed `session` and `audit` lifecycle command details and deterministic output format.
12. `sed -n '260,760p' .tmp/blick/cmd/blick/main.go`
Outcome: confirmed persisted `grant` lifecycle command tree and explicit approval labels.
13. `sed -n '1,320p' .tmp/blick/internal/app/auth/grant_service.go`
Outcome: confirmed separation between auth lifecycle and grant lifecycle.
14. `sed -n '1,260p' .tmp/blick/internal/adapters/auth/autent/grant_backend.go`
Outcome: confirmed persisted request/approval/audit state is encoded through `autent`, not shell-local behavior.
15. Context7 `/charmbracelet/fang` resolve + query
Outcome: confirmed Fang/Cobra integration and example-driven help patterns that match the `blick` implementation direction.
16. `rg -n "issue-session|revoke-session|Manage local dogfood auth sessions|principal-id|session-id" cmd/till/main.go`
Outcome: confirmed current `tillsyn` auth CLI is still the minimal two-command seam.
17. `sed -n '140,320p' cmd/till/main.go`
Outcome: confirmed current `till auth` command tree shape and help limitations.
18. `sed -n '900,1060p' cmd/till/main.go`
Outcome: confirmed `issue-session` and `revoke-session` remain low-level session helpers.
19. independent read-only subagent audit over `PLAN.md`, `README.md`, current `till` auth CLI, and `.tmp/blick` auth/grant files
Outcome: confirmed the plan direction and highlighted one extra refinement: treat approval labels plus timeout/cancel lifecycle behavior as explicit product requirements.

## Key Findings
1. `blick` has the right shell/operator quality bar.
- Explicit auth and grant lifecycle trees exist in:
  - `.tmp/blick/cmd/blick/auth_cmd.go`
  - `.tmp/blick/cmd/blick/main.go`
- Help/examples are enforced in tests in:
  - `.tmp/blick/cmd/blick/main_test.go`

2. `blick` cleanly separates auth lifecycle from approval/grant lifecycle.
- Auth lifecycle:
  - principal/client/session/audit in `.tmp/blick/cmd/blick/auth_cmd.go`
- Approval/grant lifecycle:
  - request/list/approve/deny/cancel in `.tmp/blick/cmd/blick/main.go`
  - service normalization in `.tmp/blick/internal/app/auth/grant_service.go`

3. `tillsyn` should copy the shell lifecycle discipline, not the access-profile abstraction.
- `blick` is access-profile-centered.
- `tillsyn` must stay project-path-centered, per the active run contract in `PLAN.md`.

4. `tillsyn` should keep TUI-first approval for normal users while still supporting shell approvals.
- Current `tillsyn` CLI only exposes:
  - `issue-session`
  - `revoke-session`
- That is not enough for operator inventory/review/approval.

5. The next implementation wave should use one build lane and two QA lanes.
- Build lane owns CLI, auth adapter, request state, TUI routing, and tests.
- QA lane 1 inspects CLI/auth lifecycle and test evidence.
- QA lane 2 inspects TUI notification/approval routing and test evidence.

6. Approval labels and timeout/cancel lifecycle behavior need to be explicit in the product contract.
- These should be visible in:
  - CLI help and list/show output
  - TUI review surfaces
  - auth audit/history

## Planned Command/UX Shape
1. Shell/operator path:
  - `till auth request create`
  - `till auth request list`
  - `till auth request show`
  - `till auth request approve`
  - `till auth request deny`
  - `till auth request cancel`
  - `till auth session list`
  - `till auth session validate`
  - `till auth session revoke`
2. Temporary low-level seam:
  - `till auth issue-session`
  - retained only as a secondary operator/dev path, not the primary workflow
3. Scope contract:
  - one explicit `--path` rooted at `project/<project-id>`
  - optional `/branch/<branch-id>`
  - optional repeated `/phase/<phase-id>`
4. TUI contract:
  - focused-project requests go to that project's notifications
  - off-project requests go to global notifications
  - approvals/denials must be actionable without shell fallback

## Files Expected In The Next Build Lane
- `cmd/till/main.go`
- `cmd/till/main_test.go`
- `internal/adapters/auth/autentauth/**`
- `internal/app/**`
- `internal/adapters/server/common/**`
- `internal/adapters/server/mcpapi/**`
- `internal/tui/**`

## QA Lane Plan
1. `QA-AUTH-CLI-01`
- Read-only review of:
  - CLI auth/request/session tree
  - help/examples
  - deterministic outputs
  - package-test evidence

2. `QA-AUTH-TUI-01`
- Read-only review of:
  - notifications routing
  - approve/deny surfaces
  - focused-vs-global behavior
  - external refresh behavior
  - package-test evidence

## Files Edited In This Checkpoint
1. `PLAN.md`
2. `worklogs/AUTH_UX_DOGFOOD_WAVE_PLAN_2026-03-20.md`

## Tests
1. Not run.
- Rationale: docs-only planning checkpoint.

## Current Status
1. `PLAN.md` remains the active run authority for the next auth UX wave.
2. This worklog records the split planning checkpoint because the user explicitly requested a separate worklog.
3. Next implementation should start from the lane model and the `blick`-inspired shell/TUI parity contract now recorded in `PLAN.md`.
