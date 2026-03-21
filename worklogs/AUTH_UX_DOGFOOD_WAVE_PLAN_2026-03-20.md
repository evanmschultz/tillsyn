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

## Active Implementation Checkpoint
1. Lane `BLD-AUTH-UX-01` is now active.
2. Current implementation direction:
  - add a persisted local auth-request lifecycle with one explicit project-rooted `--path`,
  - keep `autent` as the real session boundary at approval time,
  - mirror pending auth requests into the existing attention/notifications surfaces using the same stable request id,
  - add shell lifecycle commands and MCP request/poll surfaces first,
  - then wire TUI review and local create behavior.
3. Expected code hotspots:
  - `cmd/till/main.go`
  - `cmd/till/main_test.go`
  - `internal/adapters/auth/autentauth/service.go`
  - `internal/adapters/auth/autentauth/service_test.go`
  - `internal/adapters/server/common/**`
  - `internal/adapters/server/mcpapi/**`
  - `internal/tui/model.go`
  - `internal/tui/model_test.go`
  - `internal/adapters/storage/sqlite/repo.go`
  - new domain/app types as needed

## Orchestrator Continuation
1. `git status --short`
Outcome: confirmed only the split worklog was locally dirty before implementation resumed.
2. `sed -n '1,240p' Justfile`
Outcome: revalidated `just check` and `just ci` as final gates and `just test-pkg` as the worker-lane loop.
3. `sed -n '1,260p' PLAN.md`
Outcome: reloaded the active run contract and acceptance matrix before implementation.
4. `sed -n '1,260p' worklogs/AUTH_UX_DOGFOOD_WAVE_PLAN_2026-03-20.md`
Outcome: confirmed current split-worklog state before appending new evidence.
5. `rg -n "type Service interface|ListAttentionItems|ResolveAttentionItem|RaiseAttentionItem|notifications|attention|auth request|issue-session|revoke-session|authCmd|issueSession|revokeSession|NewModel" cmd/till/main.go internal/app internal/adapters/auth internal/adapters/server/common internal/adapters/server/mcpapi internal/tui -g '!**/*_golden*'`
Outcome: refreshed the primary auth CLI, notifications, and TUI integration hotspots.
6. `sed -n '1,280p' internal/adapters/auth/autentauth/service.go`
Outcome: confirmed current shared-DB `autent` adapter capabilities and missing request lifecycle.
7. `sed -n '1,260p' internal/tui/model.go`
Outcome: confirmed the current TUI service surface and lack of auth-request methods.
8. `sed -n '140,360p' cmd/till/main.go`
Outcome: confirmed `till auth` is still the thin two-command seam and identified the current help/output gap.
9. Context7 resolve for `charmbracelet/fang`
Outcome: confirmed Fang remains the right source for nested Cobra command/help/example patterns.
10. Context7 resolve for `evanmschultz/autent`
Outcome: no relevant Context7 entry exists for the local `autent` library; fallback source for `autent` is checked-in local docs and cloned source under `.tmp/autent`.
11. Context7 query for `/charmbracelet/fang`
Outcome: refreshed guidance for nested subcommands plus `Long` and `Example` help text before code changes.
12. `rg -n "type Service struct|func \\(s \\*Service\\) (IssueSession|ValidateSession|RevokeSession|Authorize|List|Request|Grant|Approve|Deny|Cancel|Audit)" .tmp/autent -g '!**/.git/**'`
Outcome: confirmed `autent` already provides session list/validate/grant/audit primitives and that `tillsyn` still needs its own auth-request layer in front of session issuance.
13. `sed -n '420,920p' .tmp/autent/app/service.go`
Outcome: confirmed `RequestGrant` requires a valid existing session, reinforcing the need for a local pre-session request/approval model.
14. `sed -n '11260,11980p' internal/tui/model.go`
Outcome: confirmed the current project/global notifications interaction points that likely need auth-request approve/deny actions.
15. `sed -n '1,240p' internal/domain/attention.go`
Outcome: confirmed `attention_items` currently provide the right user-facing notification shell but do not yet encode auth-request state.
16. `sed -n '1760,2105p' internal/adapters/storage/sqlite/repo.go`
Outcome: confirmed the current attention persistence seam and where a new persisted auth-request table will likely need to land.
17. initial worker-lane attempt via existing idle agent reuse
Outcome: lane did not return; replaced with a fresh worker lane once stale agent budget was cleared.
18. stale-agent cleanup
Outcome: freed thread budget for the current build lane plus later QA lanes.
19. launched fresh worker lane `BLD-AUTH-UX-01`
Outcome: active implementation owner is subagent `Singer` (`019d0ecc-5727-7c12-9d35-82c2e0fd6217`).
20. launched one read-only explorer for TUI insertion points
Outcome: subagent `Turing` (`019d0ecd-140d-7483-9283-eaa75ab191a9`) is gathering minimal-impact TUI approval-hook guidance for later review/integration.

## Implementation Checkpoint

Timestamp (UTC): 2026-03-21T01:30:00Z  
Scope: auth UX implementation closeout checkpoint with local test gates green.

### Landed Surfaces
1. `till auth` now includes request lifecycle commands plus session inventory and validation surfaces.
2. MCP now exposes `till.create_auth_request`, `till.list_auth_requests`, and `till.get_auth_request`.
3. Shared-DB `autent` now persists pre-session auth requests, scoped approvals, denial/cancel flows, and app-facing session wrappers.
4. TUI notifications now surface auth requests in focused-project vs global panels, support approve and deny actions, and preserve constrained path and TTL approval edits.

### Follow-up Coverage Work
1. Added `internal/adapters/server/common/capture_test.go` to cover summary-first capture behavior, work counters, attention ordering, comment rollups, and failure mapping.
2. Expanded `internal/adapters/server/common/app_service_adapter_lifecycle_test.go` to cover bootstrap guidance, attention lifecycle, capture-state mapping, and capability lease wrappers.
3. Expanded `internal/adapters/auth/autentauth/service_test.go` and `service_app_sessions_test.go` to cover app-facing session wrappers and raw session filter validation.
4. Expanded `internal/tui/model_test.go` to cover auth confirm helper behavior and keep the TUI package at the coverage floor.

### Commands And Outcomes
1. `just fmt`
Outcome: pass.
2. `just test-pkg ./internal/adapters/server/common`
Outcome: pass after capture and lifecycle coverage additions.
3. `just test-pkg ./internal/adapters/auth/autentauth`
Outcome: pass; coverage rose to 74.1%.
4. `just test-pkg ./internal/tui`
Outcome: pass; coverage reached 70.3%.
5. `just test-pkg ./internal/app`
Outcome: pass.
6. `just check`
Outcome: pass.
7. `just ci`
Outcome: pass.
8. Final QA follow-up remediation:
  - fixed app-facing auth session list state validation to fail closed on unsupported values,
  - widened auth-request continuation metadata from string-only maps to real JSON objects across CLI, domain, sqlite, app, and MCP layers.
9. `just check`
Outcome: pass after the QA remediation patch.
10. `just ci`
Outcome: pass after the QA remediation patch.
11. post-commit `just fmt`
Outcome: PASS; normalized two lingering gofmt-only test files.
12. post-format `just check`
Outcome: PASS.
13. post-format `just ci`
Outcome: PASS.

### Current Status
1. Local implementation and local gates are green.
2. `PLAN.md` has been updated as the single-source checkpoint ledger for this run.
3. Remaining closeout work is collaborative dogfood retest plus final QA sign-off against the active checklist in `PLAN.md`.

## Coverage Recovery Checkpoint
Timestamp (UTC): 2026-03-21T01:20:00Z

Objective:
- finish the auth UX implementation checkpoint by restoring all package coverage floors and re-closing `just check` plus `just ci`.

Commands And Outcomes:
1. `sed -n '1,220p' Justfile`
Outcome: revalidated `just check`, `just test-pkg`, and `just ci` as the only allowed test loop.
2. `git status --short`
Outcome: confirmed the auth UX implementation wave was still in progress with local code/test changes across CLI, auth adapter, common adapter, app, sqlite, and TUI packages.
3. Context7 query for `/golang/go/go1.26.0`
Outcome: refreshed deterministic Go testing guidance before adding more package-focused coverage tests.
4. launched explicit read-only subagent coverage audit lanes
Outcome: audit lanes did not return usable findings before the local fix loop completed, so the coverage recovery proceeded with package-local inspection and `just test-pkg` loops.
5. `just ci`
Outcome: FAIL only on coverage floor for `internal/adapters/auth/autentauth`, `internal/adapters/server/common`, and `internal/tui`.
6. Added coverage-focused tests in:
  - `internal/adapters/server/common/capture_test.go`
  - `internal/adapters/auth/autentauth/service_app_sessions_test.go`
  - `internal/tui/model_test.go`
Outcome: covered previously untested capture-state service logic, app-facing auth session wrappers, and auth-request helper branches.
7. `just fmt`
Outcome: PASS.
8. `just test-pkg ./internal/adapters/auth/autentauth`
Outcome: PASS.
9. `just test-pkg ./internal/tui`
Outcome: PASS.
10. `just test-pkg ./internal/adapters/server/common`
Outcome: initial FAIL on incorrect assumptions in new capture and lifecycle assertions.
11. Context7 query for `/golang/go/go1.26.0`
Outcome: re-run before the next edit after the failing package test, per repo policy.
12. `rg` / `sed` inspection of:
  - `internal/domain/task.go`
  - `internal/domain/capability.go`
  - `internal/app/kind_capability.go`
  - `internal/adapters/server/common/app_service_adapter_lifecycle_test.go`
Outcome: confirmed the actual completion-criteria, comment-importance, and lease-renewal contracts.
13. Corrected stale test assumptions in:
  - `internal/adapters/server/common/capture_test.go`
  - `internal/adapters/server/common/app_service_adapter_lifecycle_test.go`
  - `internal/adapters/auth/autentauth/service_app_sessions_test.go`
Outcome: aligned new coverage tests with real package behavior and removed duplicate auth session test naming.
14. `just fmt`
Outcome: PASS.
15. `just test-pkg ./internal/adapters/server/common`
Outcome: PASS.
16. `just check`
Outcome: PASS.
17. `just ci`
Outcome: PASS.
18. read-only QA lane `QA-FINAL-CODE-01`
Outcome: found two medium issues:
  - invalid auth session list state was not fail-closed in the app-facing auth adapter,
  - continuation metadata was narrower than the documented JSON object contract.
19. read-only QA lane `QA-FINAL-DOCS-02`
Outcome: no blocking doc issues; one stale historical `70.0%` TUI note in `PLAN.md` was corrected to `70.3%`.
20. implemented QA remediation in:
  - `internal/adapters/auth/autentauth/service.go`
  - `internal/domain/auth_request.go`
  - `internal/app/auth_requests.go`
  - `cmd/till/main.go`
  - `internal/adapters/server/common/app_service_adapter_mcp.go`
  - related tests in `cmd/till/main_test.go`, `internal/adapters/auth/autentauth/service_app_sessions_test.go`, `internal/adapters/auth/autentauth/service_test.go`, `internal/adapters/server/common/app_service_adapter_auth_requests_test.go`, `internal/adapters/server/common/app_service_adapter_mcp_helpers_test.go`, `internal/adapters/storage/sqlite/repo_test.go`, and `internal/domain/auth_request_test.go`
Outcome: aligned the implementation with the documented auth/session and continuation contracts.
21. `just check`
Outcome: PASS.
22. `just ci`
Outcome: PASS.

Key Results:
1. `internal/adapters/auth/autentauth` coverage rose to `74.1%`.
2. `internal/adapters/server/common` coverage rose to `78.2%`.
3. `internal/tui` coverage rose to `70.3%`.
4. The full local gate is green again after the auth UX wave, including the final QA remediation pass.

Current Status:
1. The auth UX implementation checkpoint is locally complete from a code + gate perspective.
2. Remaining work for run closeout is collaborative dogfood retest, final `PLAN.md` status alignment, and the next logical commit.
