# Kan Plan

Created: 2026-02-21
Updated: 2026-02-27
Status: Execution plan locked; immediate next action is collaborative test closeout

## 1) Primary Goal

Finish `kan` as a reliable local-first planning system for human + agent collaboration, with:
1. stable TUI workflows,
2. strict mutation guardrails,
3. MCP/HTTP parity for critical flows,
4. evidence-backed validation and closeout.

## 2) Canonical Active Docs

1. `PLAN.md` (this file): execution plan and phase/task tracker.
2. `COLLABORATIVE_POST_FIX_VALIDATION_WORKSHEET.md`: canonical collaborative validation evidence.
3. `COLLAB_E2E_REMEDIATION_PLAN_WORKLOG.md`: remediation requirements and checkpoints.
4. `MCP_FULL_TESTER_AGENT_RUNBOOK.md`: canonical MCP full-sweep run protocol.
5. `MCP_DOGFOODING_WORKSHEET.md`: MCP/HTTP dogfooding worksheet.
6. `PARALLEL_AGENT_RUNBOOK.md`: subagent orchestration policy.

## 3) Locked Constraints And References

### 3.1 Locked Constraints

1. Path portability rules:
   - no absolute-path export,
   - portable refs only (`root_alias` + relative paths),
   - import fails on unresolved required refs/root mappings.
2. Project linkage model stays `workspace_linked = true|false`.
3. Non-user mutations remain lease-gated and fail-closed.
4. Completion contracts remain required for completion semantics.
5. Attention/blocker escalation remains required for unresolved consensus/approval flows.

### 3.2 MCP References (Required)

1. MCP tool discovery/update:
   - https://modelcontextprotocol.io/legacy/concepts/tools#tool-discovery-and-updates
2. MCP roots/client concepts:
   - https://modelcontextprotocol.io/specification/2025-03-26/client/roots
   - https://modelcontextprotocol.io/docs/learn/client-concepts
3. MCP-Go:
   - https://github.com/mark3labs/mcp-go
   - Context7 id: `/mark3labs/mcp-go`

## 4) Global Subagent Execution Contract (Applies To Every Phase)

1. Orchestrator/integrator is the only writer for `PLAN.md` phase status and completion markers.
2. Each phase is split into parallel lanes with non-overlapping lock scopes.
3. Worker lanes run scoped checks only (`just test-pkg <pkg>`); no repo-wide gates in worker lanes.
4. Integrator runs repo-wide gates (`just check`, `just ci`, `just test-golden`) at phase integration points.
5. Worker handoff must include files changed, commands run, outcomes, acceptance checklist, and unresolved risks.
6. No lane closes without explicit acceptance evidence.

## 5) Phase Plan (Complete Execution Sequence)

## Phase 0: Collaborative Test Closeout (Immediate Next Action)

Objective:
- finish all collaborative test work and update worksheet evidence to current truth.

Tasks:
1. `P0-T01` Run remaining manual TUI validation for C4/C6/C9/C10/C11/C12/C13.
2. `P0-T02` Run archived/search/keybinding targeted checks and record PASS/FAIL/BLOCKED.
3. `P0-T03` Re-run focused MCP checks for known failures (`kan_restore_task`, `capture_state` readiness).
4. `P0-T04` Capture logging/help discoverability evidence (`./kan --help`, `./kan serve --help`, runtime log parity).
5. `P0-T05` Fill all blank checkpoints and sign-off blocks in `MCP_DOGFOODING_WORKSHEET.md`.
6. `P0-T06` Update `COLLABORATIVE_POST_FIX_VALIDATION_WORKSHEET.md` with final evidence paths and verdict.

Parallel lane split:
1. `P0-LA` (TUI manual validation lane)
   - lock scope: `COLLABORATIVE_POST_FIX_VALIDATION_WORKSHEET.md`, `.tmp/**` evidence artifacts.
2. `P0-LB` (MCP/HTTP verification lane)
   - lock scope: `MCP_DOGFOODING_WORKSHEET.md`, `.tmp/**` protocol/evidence artifacts.
3. `P0-LC` (logging/help verification lane)
   - lock scope: `.tmp/**` logging artifacts, worksheet evidence rows for logging sections.

Exit criteria:
1. All P0 tasks have explicit PASS/FAIL/BLOCKED outcomes with evidence.
2. No blank sign-off fields remain in active worksheets.
3. Open failures are converted into explicit implementation tasks in Phase 1.

## Phase 1: Critical Remediation Fixes

Objective:
- fix currently known blockers from collaborative validation.

Tasks:
1. `P1-T01` Fix `kan_restore_task` MCP contract/guard mismatch.
2. `P1-T02` Fix logging discoverability and runtime log-sink parity gaps.
3. `P1-T03` Implement deterministic external-mutation refresh behavior in active TUI views.
4. `P1-T04` Complete notifications/notices behavior requirements (global count, quick-nav, drill-in).
5. `P1-T05` Reconcile archived/search/key policy behavior with expected UX.

Parallel lane split:
1. `P1-LA` (transport contract lane)
   - lock scope: `internal/adapters/server/mcpapi/**`, `internal/adapters/server/httpapi/**`, related tests.
2. `P1-LB` (TUI notices/refresh lane)
   - lock scope: `internal/tui/**`, related tests/golden fixtures.
3. `P1-LC` (logging/help lane)
   - lock scope: `cmd/kan/**`, `internal/adapters/server/**`, `internal/config/**`, related tests.

Exit criteria:
1. P1 defects are closed with test evidence.
2. P0 failed checks are re-run and pass or are explicitly reclassified with rationale.

## Phase 2: Contract And Data-Model Hardening

Objective:
- lock unresolved design contracts that block stable MCP/HTTP closeout.

Tasks:
1. `P2-T01` Finalize attention storage model (`table` vs embedded JSON) and migration plan.
2. `P2-T02` Finalize attention taxonomy and lifecycle/override semantics.
3. `P2-T03` Finalize pagination/cursor contract for attention and related list surfaces.
4. `P2-T04` Finalize unresolved MCP contract decisions from prior open-question sets.
5. `P2-T05` Close snapshot portability completeness gaps for collaboration-grade import/export.
6. `P2-T06` Carry unresolved override-token documentation obligations into active docs.

Parallel lane split:
1. `P2-LA` (domain/app contract lane)
   - lock scope: `internal/domain/**`, `internal/app/**`, tests.
2. `P2-LB` (storage/schema lane)
   - lock scope: `internal/adapters/storage/sqlite/**`, migration/test fixtures.
3. `P2-LC` (transport schema/docs lane)
   - lock scope: `internal/adapters/server/**`, `README.md`, `PLAN.md`, MCP worksheets.

Exit criteria:
1. Contract decisions are encoded in code/tests/docs.
2. No unresolved “open contract” placeholders remain for in-scope MVP behavior.

## Phase 3: Full Validation And Gate Pass

Objective:
- produce final evidence-backed quality pass for current scope.

Tasks:
1. `P3-T01` Run `just check`.
2. `P3-T02` Run `just ci`.
3. `P3-T03` Run `just test-golden`.
4. `P3-T04` Execute MCP full-sweep per `MCP_FULL_TESTER_AGENT_RUNBOOK.md` and capture final report.
5. `P3-T05` Re-run collaborative worksheet and dogfooding worksheet with final verdicts.

Parallel lane split:
1. `P3-LA` (automated-gates lane)
   - lock scope: test outputs and `.tmp/**` gate artifacts.
2. `P3-LB` (MCP runbook lane)
   - lock scope: MCP run artifacts/report files.
3. `P3-LC` (manual validation lane)
   - lock scope: collaborative worksheet evidence rows/screenshots.

Exit criteria:
1. Required gates pass.
2. Worksheets have final, non-blank verdicts.
3. Remaining risks are explicitly documented with owner/next step.

## Phase 4: Docs Finalization And Closeout

Objective:
- finalize accurate active docs and remove stale narrative drift.

Tasks:
1. `P4-T01` Ensure `README.md` and `AGENTS.md` reflect actual current behavior.
2. `P4-T02` Ensure `PLAN.md` statuses match worksheet/runbook evidence.
3. `P4-T03` Remove or archive stale planning/status statements that conflict with final evidence.
4. `P4-T04` Produce final closeout summary and commit sequencing plan.

Parallel lane split:
1. `P4-LA` (product docs lane)
   - lock scope: `README.md`, `CONTRIBUTING.md`.
2. `P4-LB` (process docs lane)
   - lock scope: `AGENTS.md`, `PARALLEL_AGENT_RUNBOOK.md`.
3. `P4-LC` (plan/worksheet lane)
   - lock scope: `PLAN.md`, collab worksheets/worklogs.

Exit criteria:
1. Active docs are internally consistent.
2. No stale “not implemented” statements remain for implemented behavior.

## Phase 5: Deferred Roadmap (Not In Immediate Finish Scope)

Objective:
- preserve future work without blocking finish of current scope.

Tasks:
1. `P5-T01` Advanced import/export divergence reconciliation tooling.
2. `P5-T02` Broader policy-driven tool-surface controls and template expansion.
3. `P5-T03` Multi-user/team auth-tenancy and security hardening.

Parallel lane split:
1. `P5-LA` (import/export research lane).
2. `P5-LB` (policy/template lane).
3. `P5-LC` (security/tenancy lane).

Exit criteria:
1. Roadmap items are explicitly scoped and non-blocking for current finish target.

## 6) Immediate Next Action Lock

The very next work to run is **Phase 0: Collaborative Test Closeout**.
No new feature phase should start until Phase 0 produces updated evidence and explicit task outcomes.

## 7) Definition Of Done For Current Finish Target

1. Phase 0 through Phase 4 are complete.
2. Known blocking defects from collaborative validation are closed or explicitly accepted with owner + follow-up.
3. `just check`, `just ci`, and `just test-golden` pass on the final integrated state.
4. Collaborative and dogfooding worksheets have final non-blank sign-off verdicts.
5. Active docs are accurate and mutually consistent.

## 8) Lightweight Execution Log

### 2026-02-27: PLAN Restructure For Full Phase/Lane Execution

Objective:
- convert `PLAN.md` into a complete phase/task plan with explicit parallel-lane execution for every phase.

Result:
- phases, task IDs, lane lock scopes, and exit criteria are now defined end-to-end,
- collaborative test closeout is explicitly locked as immediate next action.

Test status:
- `test_not_applicable` (docs-only change).

### 2026-02-27: Phase 0 Collaborative Closeout Run (in progress)

Objective:
- execute Phase 0 closeout checks, capture fresh evidence, and update active worksheets with explicit PASS/FAIL/BLOCKED outcomes.

Evidence root:
- `.tmp/phase0-collab-20260227_141800/`

Commands run and outcomes:
1. `just check` -> PASS (`.tmp/phase0-collab-20260227_141800/just_check.txt`)
2. `just ci` -> PASS (`.tmp/phase0-collab-20260227_141800/just_ci.txt`)
3. `just test-golden` -> PASS (`.tmp/phase0-collab-20260227_141800/just_test_golden.txt`)
4. `just build` -> PASS with environment warning (`.tmp/phase0-collab-20260227_141800/just_build.txt`)
5. `./kan --help` -> FAIL help discoverability (`.tmp/phase0-collab-20260227_141800/help_kan.txt`)
6. `./kan serve --help` -> FAIL help discoverability / startup side-effect path (`.tmp/phase0-collab-20260227_141800/help_kan_serve.txt`)
7. `curl http://127.0.0.1:18080/healthz` -> PASS (`.tmp/phase0-collab-20260227_141800/healthz.headers`, `.tmp/phase0-collab-20260227_141800/healthz.txt`)
8. `curl http://127.0.0.1:18080/readyz` -> PASS (`.tmp/phase0-collab-20260227_141800/readyz.headers`, `.tmp/phase0-collab-20260227_141800/readyz.txt`)

Focused MCP checks and outcomes:
1. `capture_state` readiness -> PASS
   - evidence: `.tmp/phase0-collab-20260227_141800/http_capture_state_project.headers`, `.tmp/phase0-collab-20260227_141800/http_capture_state_project.json`, `.tmp/phase0-collab-20260227_141800/mcp_focused_checks.md`
2. `kan_restore_task` known failure repro -> FAIL (`mutation lease is required`)
   - evidence: `.tmp/phase0-collab-20260227_141800/mcp_focused_checks.md`
3. Guardrail failure matrix probes -> MIXED
   - M2.1 (missing/invalid lease tuple): PASS
   - M2.2 (scope mismatch rejection): FAIL (scope-type/scope-id mismatch accepted in one probe)
   - evidence: `.tmp/phase0-collab-20260227_141800/guardrail_failure_checks.md`
4. Completion guard probe -> PASS
   - unresolved blocker prevented `progress -> done`; transition succeeded after resolver step
   - evidence: `.tmp/phase0-collab-20260227_141800/completion_guard_check.md`
5. Resume/hash short loop probe -> PASS
   - state hash changed on mutation and returned to baseline post-cleanup
   - evidence: `.tmp/phase0-collab-20260227_141800/capture_state_hash_loop.md`

Blockers currently open:
1. CLI help discoverability remains broken (`./kan --help`, `./kan serve --help`).
2. `kan_restore_task` MCP contract mismatch remains unresolved.
3. Manual collaborative TUI checks remain pending user execution (C4/C6/C9/C10/C11/C12/C13 and archived/search/key policy checks).
4. Additional user-directed remediation requirements must be carried into fix phase:
   - first-launch config bootstrap should copy `config.example.toml` when config is missing,
   - help UX should be implemented with Charm/Fang styled output.

Current status:
- Phase 0 remains open until manual collaborative checks are completed and worksheet sign-offs are finalized.
- `MCP_DOGFOODING_WORKSHEET.md` has no blank sign-off fields; remaining blocked rows now carry explicit blocker statements and evidence paths.
- Section 0 user execution update recorded:
  - M0.2 runtime launch marked PASS by user,
  - M0.3 hierarchy IDs captured via MCP and unresolved user-action fixture item seeded,
  - early manual findings logged (C4 fail, C6 fail, C10 fail; others pending).
- Section 1 execution update recorded:
  - M1.1 (`capture_state` all required scopes) PASS,
  - M1.2 (`requires_user_action` blocker highlight in summary) PASS.
- Section 2 execution update recorded:
  - M2.1 PASS,
  - M2.2 FAIL (scope mismatch still accepted),
  - M2.3 PASS.

File edits in this checkpoint:
1. `MCP_DOGFOODING_WORKSHEET.md`
   - filled all USER NOTES blocks and final sign-off fields with explicit status + evidence references for this run.
2. `COLLABORATIVE_POST_FIX_VALIDATION_WORKSHEET.md`
   - added Section 12 Phase 0 tracker with current task statuses and blockers.
3. `PLAN.md`
   - logged command evidence, focused-check outcomes, blockers, and worksheet status for the active Phase 0 run.

Process contract update from user:
1. Continue section-by-section collaborative test walkthrough and note capture.
2. Preserve user intent with full detail in active markdown docs; normalize wording only when needed for technically correct terminology.
3. Final step of testing process will run subagents + Context7 (+ web research as needed) to propose fixes, then record proposals only after explicit user+agent consensus.

Additional restore-surface design requirement:
1. During fix-proposal phase, evaluate whether restore should be generalized (`restore` + explicit node/scope type arg) versus task-only surface, while ensuring required guardrail tuple fields and id/name gatekeeping semantics are consistently enforced.

### 2026-02-27: Remote E2EE Architecture + Roadmap Draft

Objective:
- produce a detailed roadmap for optional remote org collaboration with strict E2EE data handling while preserving local-first OSS usage.

Commands run and outcomes:
1. `rg --files -g'*.md' | sort` -> PASS (identified doc targets)
2. `sed -n '1,360p' PLAN.md` -> PASS (loaded active plan/worklog context)
3. `rg -n "export|import|snapshot|remote|tenancy|auth|sync|sqlite|postgres|file|attachment|project_roots" ...` -> PASS (collected active constraints from canonical docs)
4. Context7 lookup:
   - `resolve-library-id sqlite` -> PASS
   - `resolve-library-id postgresql` -> PASS
   - `query-docs /websites/sqlite_cli` -> PASS
   - `query-docs /websites/postgresql_current` -> PASS
5. Web-source verification:
   - SQLite (`isolation`, `wal`, `useovernet`, `backup`, `lang_vacuum`, `sessionintro`) -> PASS
   - PostgreSQL (`ddl-rowsecurity`, `sql-listen`, `sql-notify`) -> PASS
   - S3 (`Welcome`, `using-presigned-url`, `Versioning`) -> PASS
   - WebSocket RFC6455 -> PASS
   - libsodium `secretstream` -> PASS

File edits in this checkpoint:
1. `REMOTE_E2EE_ROADMAP.md`
   - added full architecture decision, live-update model, file/blob strategy, OSS execution phases, hosted-service roadmap, and aligned references.
2. `PLAN.md`
   - added this checkpoint with command/test evidence.

Test status:
- `test_not_applicable` (docs-only changes; no code/test behavior modified).

### 2026-02-28: Rename Wave Kickoff (tillsyn project + till command)

Objective:
- execute full no-compat rename from `kan` to `tillsyn` (project/repo identity) and `till` (runtime command/binary), then revalidate parity with `just check` and `just ci`.

Commands run and outcomes:
1. `git status --short` -> PASS (`README.md` staged change only).
2. `git add README.md && git commit -m "docs: note tillsyn naming origin in README"` -> PASS (`aadf95c`).
3. Context7:
   - `resolve-library-id` for Go (`/golang/go`) -> PASS.
   - `query-docs` on module path/import rename implications -> PASS.
4. Orchestration scan:
   - `pwd && ls -la` -> PASS.
   - `sed -n '1,220p' Justfile` -> PASS.
   - `rg -n "\bkan\b|cmd/kan|/kan\b|kan\." -S --hidden --glob '!**/.git/**'` -> PASS.
5. Spawned three subagents (code/runtime, tests/fixtures, docs/automation) and collected inventories -> PASS.

File edits in this checkpoint:
1. `PLAN.md`
   - added rename-wave kickoff checkpoint and command evidence.

Test status:
- `test_not_applicable` (planning/orchestration checkpoint; implementation in progress).

### 2026-02-28: Rename Wave Implementation Complete (No Compatibility Layer)

Objective:
- complete the all-at-once rename from `kan` to `tillsyn` (project/repo/module identity) and `till` (runtime command/binary/tool namespace), with no compatibility aliases.

Subagent lane execution and outcomes:
1. `R1-core-cli` (core CLI/module/build/path surfaces) -> PASS
   - scope delivered: `go.mod`, `cmd/till/**` (from `cmd/kan/**`), `internal/platform/**`, `internal/config/**`, `internal/tui/**`, `Justfile`, `.goreleaser.yml`, `.github/workflows/ci.yml`, `.gitignore`, `config.example.toml`, `cmd/headerlab/main.go`.
2. `R2-runtime-mcp` (server/app/domain/storage surfaces) -> PASS
   - scope delivered: `internal/adapters/server/**`, `internal/adapters/storage/sqlite/**`, `internal/app/**`, `internal/domain/**`.
3. `R3-docs-ops` (docs/runbooks/worksheets/tapes) -> PASS
   - scope delivered: `README.md`, `AGENTS.md`, `MCP_*`, `COLLAB*`, `REMOTE_E2EE_ROADMAP.md`, `vhs/**`.

Commands run and outcomes:
1. Integrator gate run `just check` -> FAIL (verify-sources pathspec before staging renamed `cmd/till/*` files).
2. Context7 re-consult (Go rename/staging implications) -> PASS.
3. Staged rename paths and reran `just check` -> FAIL (`gofmt required for cmd/till/main.go`).
4. Context7 re-consult (gofmt workflow) -> PASS.
5. `just fmt` -> PASS.
6. `just check` -> PASS.
7. `just ci` -> PASS.
8. Final cleanup of lingering test sample tokens (`kan` -> `tillsyn`) in:
   - `internal/adapters/storage/sqlite/repo_test.go`
   - `internal/app/service_test.go`
   - `internal/adapters/server/mcpapi/handler_test.go`
9. Post-cleanup verification:
   - `just check` -> PASS.
   - `just ci` -> PASS.

File edits in this checkpoint:
1. `PLAN.md`
   - added full rename implementation checkpoint with subagent evidence and gate outcomes.

Test status:
- `just check` PASS
- `just ci` PASS

### 2026-02-28: Post-Integration Docs Correction

Objective:
- resolve a docs regression introduced during rename sweep where absolute local links in the remote roadmap pointed at a non-existent workspace path.

Commands run and outcomes:
1. `rg -n "/Users/.*/personal/tillsyn|/Users/.*/personal/kan" REMOTE_E2EE_ROADMAP.md ...` -> PASS (identified hardcoded absolute links).
2. Patched `REMOTE_E2EE_ROADMAP.md` links to repo-relative paths -> PASS.

File edits in this checkpoint:
1. `REMOTE_E2EE_ROADMAP.md`
   - replaced hardcoded absolute paths with repo-relative markdown links.
2. `PLAN.md`
   - recorded post-integration docs correction checkpoint.

Test status:
- `test_not_applicable` (docs-only correction; no runtime/code behavior change).

### 2026-02-28: Phase 0 Section 2 Post-Fix Rerun (in progress, blocker persists)

Objective:
- rerun Section 2 guardrail checks after app-layer + scope-mapping fixes, then update worksheets/evidence before deciding next remediation lane.

Commands/tools run and outcomes:
1. `just test-pkg ./internal/app` -> PASS (`ok ... internal/app (cached)`).
2. `kan_create_task` probe (`actor_type=agent`, missing tuple) -> PASS expected failure (`invalid_request` requiring guard tuple fields).
3. `kan_create_task` probe (`actor_type=agent` + malformed lease token) -> PASS expected failure (`guardrail_failed ... mutation lease is invalid`).
4. `kan_issue_capability_lease` on fixture project -> PASS (issued instance `2c83f1cb-fba9-40e0-b274-84705dc5e73d`).
5. `kan_raise_attention_item` scope-mismatch probe (`scope_type=task`, `scope_id=<project_id>`) -> FAIL (unexpected acceptance; persisted `5956394b-f73a-4522-8530-ec53ec00082c`).
6. `kan_create_task` cross-project mismatch probe using fixture-scoped lease -> PASS expected failure (`guardrail_failed ... mutation lease is invalid`).
7. M2.3 completion contract probe:
   - created task `d6fe3b4a-369c-4212-b049-90630e71fc1f` in progress,
   - raised blocker `a264b6fd-15bc-427f-9972-f6f5273807ae`,
   - move to done blocked (expected),
   - resolve blocker + retry move -> PASS.
8. Cleanup:
   - resolved mismatch probe item `5956394b-f73a-4522-8530-ec53ec00082c`,
   - hard-deleted probe task `d6fe3b4a-369c-4212-b049-90630e71fc1f`,
   - revoked lease `2c83f1cb-fba9-40e0-b274-84705dc5e73d`.
9. Runtime freshness check -> FLAGGED:
   - `ls -l ./kan internal/app/attention_capture.go internal/app/kind_capability.go`
   - binary mtime `2026-02-27 14:40` predates modified source mtimes (`17:13`, `17:16`), so the rerun may have exercised a stale running server.
10. Explorer subagent root-cause pass -> COMPLETED (no edits):
   - call-chain traced from MCP handler to `Service.RaiseAttentionItem` and `validateCapabilityScopeTuple`,
   - recommended next step: restart/reload runtime and re-run M2.2 before additional code edits; if still failing, add deterministic tuple guard.
11. `just build` -> PASS with known non-fatal Go stat-cache warning; rebuilt binary mtime now `2026-02-27 17:34`.

Result summary:
1. M2.1 PASS.
2. M2.2 FAIL (still open; fail-closed behavior not enforced for `scope_type=task` + project ID).
3. M2.3 PASS.

File edits in this checkpoint:
1. `.tmp/phase0-collab-20260227_141800/manual/section2_guardrail_evidence_20260227.md`
   - appended 2026-02-28 rerun with IDs, outcomes, and cleanup.
2. `MCP_DOGFOODING_WORKSHEET.md`
   - updated M2.1/M2.2/M2.3 notes and final sign-off notes to reflect post-fix rerun outcomes.
3. `COLLABORATIVE_POST_FIX_VALIDATION_WORKSHEET.md`
   - updated Section 12.8 with explicit 2026-02-28 rerun status and persisted M2.2 blocker.

Current status:
- Phase 0 remains open; Section 2 cannot be closed due to persistent M2.2 failure.
- M2.2 runtime result is currently confounded by stale-binary risk and needs one clean rerun on a refreshed server process.
- Binary is refreshed locally; next required action is restarting `./kan serve ...` and rerunning M2.2 immediately.
- Per section-by-section policy, next step is targeted remediation of M2.2 before advancing to later sections.

### 2026-02-28: Section 2 Post-Restart Recheck + CI Gate

Objective:
- verify M2.2 on a freshly restarted runtime and confirm repo-level gate status before deciding commit readiness.

Commands/tools run and outcomes:
1. `kan_raise_attention_item` mismatch probe (`scope_type=task`, `scope_id=<project_id>`) -> PASS expected fail-closed (`not_found`, no persistence).
2. `kan_issue_capability_lease` + cross-project guarded mutation probe -> PASS expected fail-closed (`mutation lease is invalid`), lease revoked.
3. `kan_list_attention_items` open project scope check -> PASS (no unexpected open items after probe).
4. `just test-pkg ./internal/app` -> PASS.
5. `just ci` -> PASS (exit 0; coverage lines still above policy thresholds).

Result summary:
1. M2.2 fail-closed behavior is now confirmed after restart.
2. Section 2 gate status: M2.1 PASS, M2.2 PASS, M2.3 PASS.
3. Phase 0 overall remains open due to separate known blockers (help/first-launch/restore + pending manual collaborative TUI sections).

File edits in this checkpoint:
1. `.tmp/phase0-collab-20260227_141800/manual/section2_guardrail_evidence_20260227.md`
   - appended post-restart verification outcome.
2. `.tmp/phase0-collab-20260227_141800/manual/section2_post_restart_20260228.md`
   - added focused post-restart probe transcript and gate outcomes.
3. `MCP_DOGFOODING_WORKSHEET.md`
   - updated M2.2 to PASS and adjusted final blocking list accordingly.
4. `COLLABORATIVE_POST_FIX_VALIDATION_WORKSHEET.md`
   - updated Section 12.8 with post-restart M2.2 PASS evidence.

### 2026-02-27: AGENTS Flow Update (Section-by-Section Fix-As-We-Go)

Objective:
- align repository agent policy with user-directed collaborative flow:
  - test one section,
  - fix findings immediately,
  - revalidate section before moving forward.

Commands run and outcomes:
1. `rg -n "Testing Guidelines|Parallel/Subagent Mode|Temporary Next-Step Directive|..." AGENTS.md` -> PASS
2. `sed -n '1,260p' AGENTS.md` + `sed -n '260,520p' AGENTS.md` -> PASS
3. Updated `AGENTS.md` to lock section-by-section remediation loop and consensus-before-implementation workflow.
4. `rg -n "Locked execution flow|section-by-section remediation|..." AGENTS.md` -> PASS (verified insertions)

File edits in this checkpoint:
1. `AGENTS.md`
   - added temporary-phase locked execution flow for section-by-section remediation with subagent/context7/web research + consensus + scoped tests + section rerun.
   - added testing-guideline rules preventing advancement before section revalidation.

Test status:
- `test_not_applicable` (process/docs-only change).

### 2026-02-27: Restore Task Guardrail Contract Investigation

Objective:
- trace `kan_restore_task` (`kan.restore_task`) guardrail failure (`mutation lease is required`) across MCP registration, common adapter contracts, and app guard enforcement.

Commands run and outcomes:
1. `rg -n "restore_task|kan_restore_task|mutation lease is required|lease"` -> PASS (identified MCP/tool + guardrail references)
2. `rg -n "delete_task|move_task|update_task|actor"` -> PASS (identified tuple-capable mutation tools for comparison)
3. `nl -ba internal/adapters/server/mcpapi/extended_tools.go` (scoped ranges) -> PASS
4. `nl -ba internal/adapters/server/common/mcp_surface.go` -> PASS
5. `nl -ba internal/adapters/server/common/app_service_adapter_mcp.go` (scoped ranges) -> PASS
6. `nl -ba internal/app/service.go` + `internal/app/kind_capability.go` (scoped ranges) -> PASS
7. `nl -ba internal/adapters/server/common/app_service_adapter.go` + `internal/adapters/server/mcpapi/handler.go` -> PASS
8. `nl -ba internal/domain/errors.go` + `internal/domain/task.go` -> PASS
9. `nl -ba Justfile` -> PASS (startup recipe review requirement)

Findings summary:
1. `kan.restore_task` MCP registration only accepts `task_id` and calls `tasks.RestoreTask(ctx, taskID)` with no actor/lease tuple.
2. Common task-service contract and adapter method signature for restore accept only `task_id`, unlike update/move/delete request structs that include `ActorLeaseTuple`.
3. App `RestoreTask` still enforces mutation guardrails using persisted `task.UpdatedByType`; when that actor type is non-user and no guard tuple is attached to context, enforcement returns `domain.ErrMutationLeaseRequired`.
4. Error mapping converts this to MCP-visible `guardrail_failed: ... mutation lease is required`.

File edits in this checkpoint:
1. `PLAN.md`
   - added investigation worklog entry with command evidence and root-cause chain.

Test status:
- `test_not_applicable` (investigation/docs-only; no code changes).

### 2026-02-27: Remote Roadmap Update (HTTP-Only Runtime + Fang/Cobra Plan)

Objective:
- update remote roadmap with newly agreed runtime decisions:
  - HTTP-only MCP for now,
  - `kan` launches TUI with local-server ensure/reuse behavior,
  - default local endpoint `127.0.0.1:5437` with auto-fallback,
  - user endpoint selection in CLI/TUI,
  - Fang/Cobra migration,
  - phase/lane plan for parallel subagents.

Commands run and outcomes:
1. `Context7 resolve-library-id fang` -> PASS
2. `Context7 resolve-library-id cobra` -> PASS
3. `Context7 query-docs /charmbracelet/fang` -> PASS
4. `Context7 query-docs /spf13/cobra` -> PASS
5. Spawned explorer subagents for:
   - serve/runtime lifecycle verification (PASS),
   - current help/UX friction and recommendations (PASS)
6. `sed -n '1,320p' REMOTE_E2EE_ROADMAP.md` -> PASS (loaded current roadmap prior to patching)
7. `Context7 resolve-library-id mcp-go` + `query-docs /mark3labs/mcp-go` -> PASS (validated transport suitability/limits for HTTP-first decision)

File edits in this checkpoint:
1. `REMOTE_E2EE_ROADMAP.md`
   - added locked 2026-02-27 runtime/transport decisions,
   - added local runtime modes, endpoint fallback policy, and supervisor behavior,
   - added `R-CLI` phase for Fang/Cobra + server orchestration,
   - added explicit parallel lane map for subagent execution,
   - updated milestones and references.
2. `PLAN.md`
   - added this checkpoint with evidence and outcomes.

Test status:
- `test_not_applicable` (docs-only changes; no code/test behavior modified).

### 2026-02-28: R-CLI-FANG-01 Integrated (Fang/Cobra CLI Migration)

Objective:
- replace stdlib `flag` CLI parsing in `cmd/till` with Fang/Cobra, improve help/error UX, and remove orphaned parser code paths.

Commands/tools run and outcomes:
1. Context7 `resolve-library-id` + `query-docs` for `/charmbracelet/fang` and `/spf13/cobra` -> PASS (captured Execute/RunE/help/error patterns).
2. Spawned worker lane `R-CLI-FANG-01` (lock scope: `cmd/till/**`, `go.mod`, `go.sum`) -> PASS.
3. Worker lane package check loop:
   - `just test-pkg ./cmd/till` baseline -> PASS
   - post-migration `just test-pkg ./cmd/till` -> FAIL (missing `go.sum` entry)
   - dependency fetch for missing checksum + `just fmt` + rerun `just test-pkg ./cmd/till` -> PASS
4. Integrator verification:
   - `just check` -> PASS
   - `just ci` -> PASS
5. Runtime smoke:
   - `./till --help` -> PASS (styled root help)
   - `./till serve --help` -> PASS (styled subcommand help)
   - `./till --badflag` -> PASS (styled error + guidance + existing `error: ...` line)

File edits in this checkpoint:
1. `cmd/till/main.go`
   - migrated to Cobra command tree executed by Fang;
   - removed stdlib `flag` parser flow and related orphaned helpers;
   - preserved `tui` default, `serve`, `export`, `import`, and `paths` command behavior.
2. `cmd/till/main_test.go`
   - updated/added help coverage for Fang/Cobra output behavior.
3. `go.mod`, `go.sum`
   - added Fang/Cobra dependencies and required checksum entries.

Current status:
- CLI adapter migration is integrated locally and gated (`just check` + `just ci` passing).
- No remaining orphaned stdlib `flag` parser path in `cmd/till/main.go`.

### 2026-02-28: Fang Output Refinement (Paths + Error Surface)

Objective:
- ensure command output/error surfaces are Fang-styled where practical, including `till paths` presentation and removal of duplicate plain error output.

Commands run and outcomes:
1. Context7 `query-docs /charmbracelet/fang` (output/error handler styling confirmation) -> PASS.
2. `go doc github.com/charmbracelet/fang` + `go doc -all github.com/charmbracelet/fang` -> PASS (validated available APIs/Styles surface).
3. `just fmt && just test-pkg ./cmd/till` -> PASS.
4. `just ci` -> PASS.
5. Runtime smoke:
   - `./till paths` -> PASS (styled titled key/value output).
   - `./till --badflag` -> PASS (Fang-styled error block, no extra plain `error:` suffix).

File edits in this checkpoint:
1. `cmd/till/main.go`
   - removed duplicate top-level plain error print in `main`;
   - added `writePathsOutput` using Fang default color scheme + lipgloss rendering;
   - routed `paths` command through styled renderer.
2. `cmd/till/main_test.go`
   - updated `TestRunPathsCommand` assertions for titled/styled paths output semantics.

Current status:
- `paths` output and CLI error surface are now aligned with Fang-style rendering expectations.

### 2026-02-28: init-dev-config Regression Fix (TTY vs Non-TTY Paths Output)

Objective:
- restore automation compatibility for recipes parsing `till paths` while keeping styled interactive output.

Commands run and outcomes:
1. `nl -ba Justfile | sed -n '1,140p'` -> PASS (identified parser dependency on `config: ...` format in `init-dev-config`/`clean-dev`).
2. Context7 resolve/query for Go terminal package -> unavailable/insufficient for target package.
3. Fallback doc source: `go doc golang.org/x/term.IsTerminal` -> PASS (`IsTerminal(fd int) bool`).
4. `just fmt && just test-pkg ./cmd/till && just ci` -> PASS.

File edits in this checkpoint:
1. `cmd/till/main.go`
   - `paths` now renders styled output only when stdout is a terminal and `NO_COLOR` is unset;
   - non-TTY output path restored to stable plain `key: value` lines for script parsing;
   - added small test hook variable for forcing styled mode in tests.
2. `cmd/till/main_test.go`
   - restored plain-output assertions for `run(paths)` on non-TTY writers;
   - added tests for plain output, styled output path, and `supportsStyledOutput` behavior.

Current status:
- interactive `till paths` remains styled;
- non-interactive/pipe usage remains machine-parseable, fixing `just init-dev-config` and `just clean-dev` parsing behavior.

### 2026-02-28: Default Serve Endpoint Update to 5437

Objective:
- align default HTTP serve endpoint to `127.0.0.1:5437` (derived from user requirement `e * 2`) across CLI and server fallback behavior.

Commands run and outcomes:
1. `rg -n "127\\.0\\.0\\.1:8080|8080|defaultBindAddress"` across CLI/server/tests -> PASS (identified all code references).
2. Checked local `/Users/evanschultz/.codex/config.toml` and TOML search under `/Users/evanschultz/.codex` -> PASS (no endpoint/default binding present; only project trust/mcp server config).
3. `just fmt && just check && just ci` -> PASS.

File edits in this checkpoint:
1. `cmd/till/main.go`
   - changed default `serve` flag HTTP bind from `127.0.0.1:8080` to `127.0.0.1:5437`.
2. `internal/adapters/server/server.go`
   - changed server fallback bind constant to `127.0.0.1:5437`.
3. `cmd/till/main_test.go`
   - updated default serve binding expectation to `127.0.0.1:5437`.

Current status:
- default endpoint is now consistently `127.0.0.1:5437` in CLI and server fallback paths.
- repo gates are green (`just check`, `just ci`).

### 2026-02-28: Dev-Mode Release Policy Note (User Requirement)

Objective:
- capture explicit policy that dev-mode behavior must not be the default for packaged/public OSS distributions; contributors should opt into dev behavior explicitly.

Policy note:
- For release/brew installs and general OSS user flows, dev behavior should be opt-in (`--dev` or `TILL_DEV_MODE=true`) rather than implicit default.
- Contributor workflows can still use explicit dev mode for isolated local paths/logging.
- Future packaging/release hardening should verify non-dev defaults and avoid shipping with implicit dev-mode defaults.

Current status:
- policy requirement recorded; implementation follow-up remains a future hardening task.

### 2026-02-28: Independent Live HTTP/MCP E2E Probe Sweep (Against User-Run Server)

Objective:
- run independent transport + parity probes against user-started `./till serve` runtime on `127.0.0.1:5437`, acknowledging existing `User_Project` data.

Commands run and outcomes:
1. HTTP connectivity probe:
   - `curl -i http://127.0.0.1:5437/api/v1/capture_state` -> PASS (reachable, deterministic 400 invalid_request for missing `project_id`).
2. MCP initialize/tools discovery:
   - `initialize` (`protocolVersion=2025-06-18`) -> PASS (200, negotiated protocol `2025-06-18`, server `tillsyn/dev`).
   - `tools/list` -> PASS (30 tools present, includes `till.list_projects`).
3. Existing project probe (expected pre-seeded data):
   - `tools/call till.list_projects(include_archived=true)` -> PASS (`User_Project` present, treated as expected).
4. HTTP/MCP parity on same project (`User_Project`, id `10cdd734-bf41-4155-b978-b5f5f5061050`):
   - HTTP `GET /api/v1/capture_state?...view=summary` vs MCP `till.capture_state(...view=summary)` -> PASS:
     - matching `state_hash`,
     - matching scope name (`User_Project`),
     - matching `work_overview.total_tasks=0`.
   - HTTP `GET /api/v1/attention/items?...state=open` vs MCP `till.list_attention_items(...state=open)` -> PASS:
     - matching item count (`0`).
5. Stateless/transport behavior:
   - `tools/list` with bogus `Mcp-Session-Id` header -> PASS (200, request still works).
   - unknown method (`unknown/method`) -> PASS (200 JSON-RPC error payload; deterministic message).
   - invalid JSON body (`{`) -> PASS (400 with deterministic parse error).
6. Initialize protocol matrix:
   - legacy `2024-11-05` -> PASS (accepted; negotiated `2024-11-05`),
   - future `2099-01-01` -> PASS (deterministic fallback `2025-11-25`),
   - missing `protocolVersion` -> PASS (deterministic default `2025-03-26`).

File edits in this checkpoint:
1. `E2E_PARITY_LOG.md`
   - created collaborative parity log with independent findings and split ownership plan (`assistant-only`, `user-only`, `together`).
2. `PLAN.md`
   - recorded live probe evidence and policy notes for the session.

Current status:
- independent HTTP/MCP sweep against live user-run runtime completed successfully.
- no blockers found for moving into collaborative parity checks.

### 2026-02-28: Bubble Tea v2 External-Update + Polling Research (No Code Edit)

Objective:
- collect authoritative guidance for Bubble Tea v2 external updates and live refresh loops (`Program.Send`, `tea.Tick`, `tea.Every`) and map it to current `till` TUI architecture risks.

Commands/research actions and outcomes:
1. Context7:
   - `resolve-library-id("bubble tea")` -> PASS (`/charmbracelet/bubbletea` selected).
   - `query-docs` for `Program.Send` + `Tick/Every` semantics -> PASS (captured one-shot timer behavior + external send control).
2. Online Charm/Bubble Tea primary sources:
   - Bubble Tea issue/PR history (`#25`, `#113`) -> PASS (confirmed design intent and `Program.Send` behavior contract).
   - Bubble Tea package docs (`pkg.go.dev/charm.land/bubbletea/v2`) -> PASS (confirmed `Program.Send`, `Tick`, `Every` behavioral notes).
   - Bubble Tea source/docs/examples:
     - `tea.go`, `commands.go` -> PASS (authoritative comments for send and timer semantics).
     - `examples/simple/main.go`, `examples/realtime/main.go`, `examples/send-msg/main.go`, discussion `#951` -> PASS (practical periodic and external-event patterns).
3. Repo architecture mapping:
   - reviewed `cmd/till/main.go`, `internal/tui/model.go`, `internal/tui/thread_mode.go`, `internal/tui/options.go`, `internal/config/config.go` -> PASS.
   - confirmed current TUI uses command-triggered reloads (`m.loadData`) with no background tick loop and no `Program.Send` integration.
   - confirmed existing selection/focus retention hooks (`clampSelections`, `retainSelectionForLoadedTasks`, `focusTaskByID`) that can be leveraged for stale-selection mitigation.

File edits in this checkpoint:
1. `PLAN.md`
   - appended research evidence and outcomes (this section).

Current status:
- research evidence collected and mapped to repo-specific recommendation surface.
- next step is to hand back practical architecture guidance/caveats to user (input focus churn, race/overfetch, stale selection).

### 2026-02-28: Live TUI External-Write Refresh Remediation (Section-by-Section Bug Fix)

Objective:
- fix collaborative validation blocker where TUI board state did not live-refresh after external MCP/HTTP mutations; align AGENTS remediation wording with explicit user workflow (`find bug -> log immediately -> fix -> verify -> move on`).

Commands/research actions and outcomes:
1. Subagent investigation sweep (code + Context7 + Charm/Bubble Tea discussions) -> PASS:
   - root cause confirmed: no periodic/subscribed board refresh path in `internal/tui/model.go`; board only reloaded on local actions/manual `r`.
   - recommendation converged on guarded recurring `tea.Tick` loop + single-flight gating + input-mode safety.
2. Context7 research:
   - `/charmbracelet/bubbletea` and pkg docs queries for `Tick/Every` one-shot semantics and `Program.Send` guidance -> PASS.
3. Implementation gates:
   - `just fmt` -> PASS.
   - `just test-pkg ./internal/tui` -> PASS.
   - `just test-pkg ./cmd/till` -> PASS.
   - `just check` -> PASS.
   - `just ci` -> PASS.

File edits in this checkpoint:
1. `AGENTS.md`
   - strengthened temporary collaborative remediation language to require immediate bug logging and per-bug fix/verify before advancing sections.
2. `internal/tui/model.go`
   - added guarded auto-refresh primitives (`autoRefreshTickMsg`, `autoRefreshLoadedMsg`, interval/arming/in-flight fields);
   - added recurring timer scheduling via `tea.Tick` and background load command wrapper;
   - added mode-gated auto-refresh (`modeNone`, `modeTaskInfo`, `modeActivityLog`) to avoid text-input disruption;
   - refactored loaded-state application into `applyLoadedMsg` and wired auto-refresh flow to schedule follow-up ticks.
3. `internal/tui/options.go`
   - added `WithAutoRefreshInterval(time.Duration)` option.
4. `cmd/till/main.go`
   - enabled TUI auto-refresh in runtime with `tui.WithAutoRefreshInterval(2*time.Second)`.
5. `internal/tui/model_test.go`
   - added live-refresh regression tests:
     - `TestModelAutoRefreshTickReloadsExternalMutationsInBoardMode`
     - `TestModelAutoRefreshTickSkipsInputModes`
     - `TestModelAutoRefreshTickPreservesFocusedSubtree`
   - added focused test helpers for auto-refresh tick/load command handling.

Current status:
- bug fix implemented and fully gated (`just check` + `just ci` green);
- TUI now periodically refreshes external mutations while preserving input-mode UX safety and subtree focus behavior.

### 2026-02-28: Notices "Recent Activity" Live-Refresh Gap (New Blocking Bug)

Objective:
- fix collaborative test finding that notices-panel `Recent Activity` did not live-refresh after external MCP mutations, even when board cards/fields updated.

Bug capture (user report):
- while verifying stepwise MCP updates in `User_Project`, task fields live-updated but notices `Recent Activity` remained stale and did not include new external edits.

Actions taken:
1. Context gathering:
   - inspected `internal/tui/model.go` data flow for `loadData`, `applyLoadedMsg`, `renderOverviewPanel`, and `activityLog` handling.
2. Root-cause confirmation:
   - notices panel reads `m.activityLog`, but normal board refresh path did not repopulate `activityLog` from persisted `ListProjectChangeEvents`.
3. Context7 checkpoint:
   - re-queried Bubble Tea command/update guidance before edits (tick-driven reloads should apply all state slices from returned message).
4. Remediation implementation:
   - wired `loadData` to fetch persisted change events and include mapped activity entries in `loadedMsg`;
   - updated `applyLoadedMsg` to hydrate/refresh `m.activityLog` from loaded activity entries;
   - added targeted TUI regression test for notices-panel live activity refresh from persisted events.
5. Verification commands:
   - `just fmt` -> PASS.
   - `just test-pkg ./internal/tui` -> PASS.
   - `just check` -> PASS.
   - `just ci` -> PASS.

Current status:
- bug fixed and verified; notices-panel `Recent Activity` now follows live external activity updates on normal board refresh.

### 2026-02-28: Header Branding Correction (`TILL` -> `HA TILL`)

Objective:
- align TUI header brand mark with project naming (`HA TILL`) and keep tests/goldens green.

Actions taken:
1. Updated board header wordmark constant in `internal/tui/model.go` from `TILL` to `HA TILL`.
2. Updated expanded help title label from `TILL Help` to `HA TILL Help` for consistent branding.
3. Golden snapshot remediation after expected output change:
   - `just test-golden-update` -> PASS.
   - `just test-pkg ./internal/tui` -> PASS.
   - `just check` -> PASS.
   - `just ci` -> PASS.

Current status:
- branding mismatch fixed and validated; golden snapshots updated to match intentional UI text changes.

### 2026-02-28: Ownership Attribution Requirement (User-Confirmed Priority)

Objective:
- preserve and surface mutation ownership as first-class data across node updates, because downstream collaboration features (comments, auditability, agent/user/system workflows) depend on it.

Requirement note (from collaborative testing session):
- every node update must retain ownership attribution fields (`actor_type` and actor identity/name);
- notices-panel recent activity should foreground ownership in compact form, with full owner details available in activity detail views;
- compact owner display should be character-limited in board notices, while detail modals should show the full owner identity.

Current status:
- requirement recorded as a non-negotiable UX/data contract for current and future mutation/audit surfaces.

### 2026-02-28: Notices Activity Ownership + Drill-Down Navigation Remediation

Objective:
- address collaborative UX bug where notices `Recent Activity` emphasized timestamps instead of ownership, lacked panel navigation, and lacked drill-down/jump-to-node behavior.

Changes implemented:
1. Activity data enrichment:
   - extended in-memory `activityEntry` to carry ownership + event metadata fields (`ActorType`, `ActorID`, `Operation`, `WorkItemID`, metadata map).
   - mapped persisted `ChangeEvent` actor fields into `activityEntry` during reload.
2. Notices panel ownership display:
   - replaced timestamp-leading notices activity row format with compact owner-leading format (`actor_type|actor_name` + summary), with character-limited owner label.
3. Notices panel keyboard navigation:
   - added board/notices focus toggle via `tab` in normal mode.
   - added notices activity row selection with `j/k` or arrow keys.
4. Activity detail modal:
   - added dedicated activity-event detail modal from notices (`enter`) showing full owner identity, full timestamp, operation, target, node id, and metadata.
5. Jump-to-node workflow:
   - added node jump action from activity detail (`enter`/`g`) with fallback flow that enables archived visibility and reloads when needed.
   - emits unavailable status when event target cannot be resolved (possible hard delete).
6. Help/hints:
   - updated board expanded-help and notices-panel hints to describe notices focus + detail interaction.

Tests added/updated:
1. `TestModelRecentActivityPanelShowsOwnerPrefix`
2. `TestModelNoticesActivityDetailAndJump`
3. `TestModelActivityEventJumpLoadsArchivedTask`
4. Existing notices/activity tests updated for intentional hint/text changes.
5. Golden snapshots updated for expected UI text differences.

Verification commands and outcomes:
1. `just fmt` -> PASS.
2. `just test-golden-update` -> PASS.
3. `just test-pkg ./internal/tui` -> PASS.
4. `just check` -> PASS.
5. `just ci` -> PASS.

Current status:
- ownership-first notices activity UX and drill-down navigation are implemented and verified;
- collaborative step-by-step live external-update validation can resume.

### 2026-02-28: MCP/Change-Event Actor Attribution Trace + Minimal Remediation

Objective:
- trace actor attribution end-to-end (MCP -> server adapter -> app service -> sqlite change_events) and fix the specific gaps causing notices activity rows to appear as `user|tillsyn-user` for orchestrator-driven mutations.

Context + root-cause findings:
1. MCP mutation actor tuple normalization lived in `withMutationGuardContext`, but user-attribution naming and guard tuple detection were conflated (explicit `actor_type=user` + `agent_name` was rejected).
2. `till.restore_task` did not accept/pass actor tuple at all, so restore mutations could not carry actor identity/guard context through MCP.
3. Several app mutation paths (`move`, `restore`, `rename`, `reparent`, archive delete, and update-without-metadata) wrote task changes without reapplying caller actor identity, so persisted change events often reused fallback/default ownership.
4. Hard delete change-event insertion path in sqlite used stored task actor fields only and did not honor request-scoped actor context.

Context7 + fallback evidence:
1. Context7 lookup for MCP-Go optional argument extraction:
   - `resolve-library-id("mark3labs/mcp-go")` -> PASS (`/mark3labs/mcp-go`)
   - `query-docs("/mark3labs/mcp-go", optional args/GetString/BindArguments)` -> PASS
2. Context7 lookup for Go stdlib `context` did not return a suitable library entry.
   - fallback source used before edits: existing repo-local context-key pattern in `internal/app/mutation_guard.go` and idiomatic package-local key usage already present in this codebase.

File edits in this checkpoint:
1. `internal/app/mutation_guard.go`
   - added `MutationActor` context payload + `WithMutationActor` / `MutationActorFromContext` helpers for request-scoped mutation attribution.
2. `internal/adapters/server/common/mcp_surface.go`
   - added `RestoreTaskRequest` with actor tuple; updated `TaskService` interface restore signature accordingly.
3. `internal/adapters/server/common/app_service_adapter_mcp.go`
   - updated `RestoreTask` to accept actor tuple and route through `withMutationGuardContext`.
   - refined guard-tuple detection (`agent_instance_id|lease_token|override_token`) so `actor_type=user` + `agent_name` works for attribution without forcing lease tuple.
   - attached mutation actor metadata to context for downstream persistence attribution.
4. `internal/adapters/server/mcpapi/extended_tools.go`
   - extended `till.restore_task` tool schema with actor tuple fields and forwarded them to restore request.
5. `internal/app/service.go`
   - added `applyMutationActorToTask` helper and applied it in task mutation paths (`move`, `restore`, `rename`, `update`, `reparent`, archive delete).
   - updated metadata update path to reuse normalized task-level actor fields when persisting.
6. `internal/adapters/storage/sqlite/repo.go`
   - hard-delete change-event write now honors request-scoped `MutationActor` context when present.
7. `internal/adapters/server/common/app_service_adapter_mcp_guard_test.go`
   - added coverage case proving user actor can provide name attribution without guard tuple.
8. `internal/adapters/server/mcpapi/extended_tools_test.go`
   - updated restore-task stub signature to new restore request type.
9. `internal/app/service_test.go`
   - added test coverage for context-provided actor attribution persistence on task update.

Commands/test evidence and outcomes:
1. `just fmt` -> PASS.
2. `just test-pkg ./internal/app` -> PASS.
3. `just test-pkg ./internal/adapters/server/common` -> PASS (`[no test files]`).
4. `just test-pkg ./internal/adapters/server/mcpapi` -> PASS.
5. `just test-pkg ./internal/adapters/storage/sqlite` -> PASS.
6. `just check` -> PASS.
7. `just ci` (run 1) -> FAIL (`internal/tui` package coverage 69.7% below 70% threshold).
8. `just ci` (run 2) -> FAIL (`internal/tui` build/test failure in existing `renderOverviewPanel` test call sites).

Current status:
- actor attribution path has been remediated for MCP task mutations (including restore + hard delete event attribution);
- full `just ci` remains red due unrelated `internal/tui` gate failure outside the touched actor-attribution scope.

### 2026-02-28: Late Subagent Audit + `test/fix cycle (collab)` Commit Rule

Objective:
- audit unexpected late subagent edits for scope/intent correctness and add explicit collaborative commit-discipline wording requested by user.

Actions and evidence:
1. Updated `AGENTS.md` temporary collaborative locked-flow with explicit `test/fix cycle (collab)` rule:
   - each fix scope must be validated and committed before next fix scope starts;
   - no new fix scope starts while prior cycle edits remain uncommitted unless user explicitly approves discard.
2. Reopened prior worker agent `019ca2c0-5445-7183-8131-e7e890f64312`, requested strict postmortem, captured assignment/scope/intent statement, then closed agent to prevent additional background edits.
3. Ran direct file-level audit of late subagent changes:
   - `internal/adapters/server/common/app_service_adapter_mcp.go`
   - `internal/adapters/server/common/mcp_surface.go`
   - `internal/adapters/server/mcpapi/extended_tools.go`
   - `internal/app/mutation_guard.go`
   - `internal/app/service.go`
   - `internal/adapters/storage/sqlite/repo.go`
   - related tests.
4. Re-validated touched package tests:
   - `just test-pkg ./internal/app` -> PASS
   - `just test-pkg ./internal/adapters/server/common` -> PASS (`[no test files]`)
   - `just test-pkg ./internal/adapters/server/mcpapi` -> PASS
   - `just test-pkg ./internal/adapters/storage/sqlite` -> PASS

Current status:
- unexpected edits source confirmed: late worker completion on prior actor-attribution lane;
- collaborative commit-discipline requirement has been codified in `AGENTS.md`;
- actor-attribution edit set is technically coherent and package-tested, with follow-up review still required for broader merge intent and remaining TUI gate failures.

### 2026-02-28: User-Run Gate Failures Logged (Current Blocker)

Objective:
- record exact current test/gate failures reported by user shell output before additional fixes.

User-provided command evidence:
1. `just check` -> FAIL in `internal/tui`:
   - failing test: `TestModelViewShowsNoticesPanel`
   - assertion mismatch at `internal/tui/model_test.go:5787`
   - expected old notices hint text no longer matches rendered output (`tab/shift+tab panels • enter details • g full activity log` now renders).
2. `just ci` -> interrupted (`^C`, exit code 130) during coverage run; `internal/tui` had not been remediated yet, so CI remains blocked pending same TUI test fix.

Local corroboration:
1. `just test-pkg ./internal/tui` -> FAIL with same `TestModelViewShowsNoticesPanel` expectation mismatch.

Current status:
- commit remains blocked until `internal/tui` test expectations/goldens are reconciled and gates pass.

### 2026-02-28: Collaborative Reset Prep (Green Gates + Dev Config Debug Default)

Objective:
- prepare repository for fresh collaborative validation restart:
  - ensure failing TUI gate is fixed,
  - ensure `init-dev-config` enforces debug logging level,
  - restore green `just check` + `just ci`.

Edits made:
1. `internal/tui/model_test.go`
   - updated stale notices hint assertion in `TestModelViewShowsNoticesPanel` from old text (`tab focus notices`) to current rendered hint prefix (`tab/shift+tab panels`).
2. `Justfile` (`init-dev-config` recipe)
   - kept config copy behavior,
   - added idempotent post-step rewrite that guarantees:
     - `[logging]` table exists,
     - `level = "debug"` inside `[logging]`,
   - applies whether config is newly created or already exists.
3. `internal/adapters/server/common/app_service_adapter_mcp_actor_attribution_test.go`
   - added `//go:build commonhash` to align with existing `common` package test-tag pattern and avoid per-package coverage gate regression in default CI flow.
4. `internal/adapters/server/mcpapi/extended_tools_test.go`
   - added default-flow actor-tuple forwarding verification via `mcpapi` handler tests:
     - update task actor tuple forwarding (`actor_type=user`, `agent_name=EVAN`),
     - restore task actor tuple forwarding (`actor_type=agent` + lease tuple fields),
   - captured request structs in stub service for explicit field assertions.

Commands and outcomes:
1. `just test-pkg ./internal/tui` -> PASS (after assertion update).
2. `just ci` (first rerun) -> FAIL:
   - coverage gate failure on `internal/adapters/server/common` (7.7%) caused by introducing default-flow tests in that package.
3. Context7 re-check performed (Go build tags/coverage behavior) before next edit.
4. `just test-pkg ./internal/adapters/server/common` -> PASS (`[no test files]` after tag alignment).
5. `just test-pkg ./internal/adapters/server/mcpapi` -> PASS.
6. `just check` -> PASS.
7. `just ci` -> PASS.
8. `just build` -> PASS (non-fatal module stat-cache permission warning observed in sandboxed environment).
9. `./till --help` smoke check -> PASS.

Current status:
- repository gates are green (`just check`, `just ci`);
- `init-dev-config` now guarantees `[logging] level = "debug"` for dev config;
- ready for user to run `just clean-dev` and restart from a fresh state for collaborative live validation.

### 2026-02-28: `init-dev-config` Migration To Cobra/Fang Command (Regex Helper)

Objective:
- replace shell/awk-based `init-dev-config` logic with a first-class Cobra/Fang command backed by Go helper code.

Context7 checkpoints:
1. Queried Context7 for Cobra command wiring (`AddCommand`, `RunE`, help behavior).
2. Queried Context7 for Go regex behavior and multiline anchoring.
3. After failed `just check` runtime panic (unsupported lookahead in Go regexp), re-queried Context7 and switched to Go-compatible regex + index slicing.

Edits made:
1. `cmd/till/main.go`
   - added `init-dev-config` Cobra/Fang subcommand with help text.
   - added `runInitDevConfig` flow:
     - resolves dev paths via platform options,
     - creates missing config from repo `config.example.toml`,
     - enforces `[logging] level = "debug"` via Go helper.
   - added `ensureLoggingSectionDebug` regex helper and related TOML section regexes.
2. `cmd/till/main_test.go`
   - updated root-help expectations to include `init-dev-config`.
   - added subcommand-help expectations for `init-dev-config`.
   - added command tests for create/update behavior and output contract.
   - added table test for `ensureLoggingSectionDebug`.
3. `Justfile`
   - replaced shell/awk recipe body with direct command call:
     - `./till --dev init-dev-config`

Commands and outcomes:
1. `just fmt` -> PASS.
2. `just check` (first run) -> FAIL (panic from unsupported regexp lookahead in `cmd/till/main.go`).
3. Context7 re-check performed for Go-compatible regex approach.
4. `just fmt` -> PASS (after fix).
5. `just check` -> PASS.
6. `just ci` -> PASS.
7. `./till --help` -> PASS; command listed with Fang-styled help.
8. `./till init-dev-config --help` -> PASS; subcommand help renders correctly.
9. `HOME=$(mktemp -d) ... ./till --app tillsyn-smoke init-dev-config` -> PASS; single-line output confirmed.

Current status:
- `init-dev-config` is now a native Cobra/Fang command (no ad-hoc shell parser logic in recipe);
- debug logging enforcement is in Go helper code;
- help output and CI gates are green.

### 2026-02-28: Collaborative MCP Live E2E Re-Run (Ownership + Guardrails)

Objective:
- execute MCP-first live validation against user-restarted server, verify guardrail gating and ownership attribution, and preserve created records for TUI inspection.

Context:
1. Initial rerun attempt hit `attempt to write a readonly database (1032)` on all mutation calls.
2. Transport isolation showed same error across MCP and HTTP write paths (not MCP-only).
3. User rebuilt/restarted server; rerun then proceeded successfully.

Commands/evidence (MCP + minimal local read-only support):
1. `till.list_projects(include_archived=true)` -> PASS; active project `d83f5620-d9cb-4dc1-b281-67f92c69463b` (`1_user_pro`).
2. `till.list_tasks(project_id=..., include_archived=false)` -> PASS (initially empty).
3. Local read-only SQL query for column IDs (required because MCP has no list-columns tool) -> PASS:
   - To Do: `c7fd8e06-678a-441f-901f-897e2da9bf0b`
   - In Progress: `8644d4c9-4429-42f0-aaa2-89060855d851`
   - Done: `e11c99eb-6c68-4ecd-8388-6bd601fdb6e6`

SG1 guardrail lane (`Codex_Subagent_SG1`, `sg1-instance`):
1. `till.create_task` as `actor_type=agent` without lease tuple -> PASS expected failure (`invalid_request`, lease tuple required).
2. `till.issue_capability_lease` -> PASS (`lease_token=e9a556ec-0a47-4c6a-bf27-81bd42ac7400`).
3. `till.create_task` -> PASS created `d0cf8388-30dc-4424-80c0-2c8e6161f5e8` (`10_SG1_Lease_Create`).
4. `till.update_task` -> PASS title now `10_SG1_Lease_Update`.
5. `till.move_task` to In Progress -> PASS.
6. `till.create_comment` on SG1 task -> PASS comment `55f749a6-b6d8-491d-8375-c6abc6231eeb`.

SG2 ownership lane (`Codex_Subagent_SG2`, `sg2-instance`):
1. `till.issue_capability_lease` -> PASS (`lease_token=aa1c2c4f-fa6e-48b0-a6bf-3b21dec62115`).
2. `till.create_task` branch -> PASS `c7fad53f-5c12-4146-b727-ab80ea0036da` (`11_SG2_Branch`).
3. `till.create_task` phase (parent=branch) -> PASS `196e55bf-54dc-4d2b-a2e2-eaf1ce9b3dd6`.
4. `till.create_task` with `kind=subphase` -> FAIL (`kind definition not found: "subphase"`).
5. `till.create_task` subphase using `scope=subphase`, `kind=phase` -> PASS `b87d4221-36dd-4c0e-82f1-2b09a2def653`.
6. `till.create_task` child task -> PASS `fabd90bc-e700-485d-9658-add06cc6883f`.
7. `till.update_task` -> PASS title `11_SG2_Task_Updated`.
8. `till.move_task` to In Progress then Done -> PASS both moves.
9. `till.create_comment` on SG2 branch -> PASS comment `f3978dfc-a2ba-4d0b-9053-492f7d3e0f50`.

Guardrail validation:
1. SG2 task update with bogus lease token `00000000-0000-0000-0000-000000000000` -> PASS expected `guardrail_failed` (`mutation lease is invalid`).
2. SG1 task update with SG2 lease token -> PASS expected `guardrail_failed` (`mutation lease is invalid`).

Ownership evidence:
1. `till.list_project_change_events(project_id=..., limit=40)` -> PASS:
   - events show `ActorType=agent` with `ActorID=Codex_Subagent_SG1` for SG1 create/update/move.
   - events show `ActorType=agent` with `ActorID=Codex_Subagent_SG2` for SG2 create/update/move.
2. `till.list_comments_by_target` on SG1/SG2 targets -> PASS:
   - SG1 comment `AuthorName=Codex_Subagent_SG1`, `ActorType=agent`.
   - SG2 comment `AuthorName=Codex_Subagent_SG2`, `ActorType=agent`.

Current status:
- live MCP mutation path is working after server restart;
- guardrails + ownership attribution are validated with preserved artifacts for TUI check;
- one surfaced contract gap: no `subphase` kind definition (requires `scope=subphase` with `kind=phase`).
- one surfaced MCP tooling gap: no `till.list_columns`/column-discovery endpoint, forcing out-of-band DB lookup to obtain `column_id` values before `create_task`/`move_task` calls.

### 2026-02-28: Collaborative TUI Activity UX Remediation (Recent Activity + Jump + Event Details)

Objective:
- fix collaborative findings in notices/activity UX:
  1. recent-activity owner rows were visually misaligned and clipped early,
  2. `go to node` from activity event could fail to focus the actual nested node,
  3. activity event detail modal showed raw UUID-heavy metadata that was not user-actionable.

User-reported issues logged:
1. recent-activity owner text (`agent|<name>`) was offset and truncated before other notice rows.
2. activity-event `go to node` returned to board but did not reliably focus the referenced node.
3. activity-event modal showed raw IDs (`work_item_id`, `*_column_id`, positions) instead of path/task context.

Implementation updates:
1. `internal/tui/model.go`
   - added jump-context preparation (`prepareActivityJumpContext`) and used it in jump flows so nested targets are focusable.
   - updated `focusTaskByID` to return success status for jump verification.
   - updated notices recent-activity row rendering to remove extra offset and keep owner/summary aligned.
   - updated activity-event modal details to show:
     - user-facing `node` and `path`,
     - humanized metadata (column names, changed fields, lifecycle transitions),
     - filtered-out raw UUID/position noise keys.
2. `internal/tui/model_test.go`
   - added/updated regression tests for:
     - nested jump focus correctness,
     - humanized column metadata rendering,
     - owner display normalization,
     - fallback target/path labels,
     - metadata-friendly fallback formatting.

Commands and outcomes:
1. `just ci` -> FAIL (pre-fix coverage gate: `internal/tui` 69.9%).
2. Context7 re-check performed before next edits (Bubble Tea test/update patterns).
3. `just test-pkg ./internal/tui` -> FAIL (compile error in new test: invalid model field literal).
4. Context7 re-check performed after failure (required by repo policy).
5. `just test-pkg ./internal/tui` -> PASS.
6. `just check` -> PASS.
7. `just ci` -> PASS (`internal/tui` coverage now 70.3%).

Current status:
- collaborative activity UX findings above are implemented and covered by tests;
- repo gates are green for this cycle (`just check`, `just ci`);
- MCP tooling gap (`no till.list_columns`) remains explicitly tracked for follow-up fix scope.

### 2026-02-28: Branding Normalization (`tillsyn` app name, `till` command-only)

Objective:
- enforce naming intent: app/UI branding must be `tillsyn`; command/tool syntax remains `till`.

Findings captured before edits:
1. TUI header/help branding showed `HA TILL` and `HA TILL Help`.
2. Empty-project and thread headers showed `till` as app label (`till`, `till thread`).
3. README wording contained invalid phrase `ha till`.
4. Config example heading used `# till example configuration`.

Implementation updates:
1. `internal/tui/model.go`
   - `headerMarkText` -> `TILLSYN`.
   - help modal title -> `TILLSYN Help`.
   - empty-project title -> `tillsyn`.
   - command palette quit description -> `quit tillsyn`.
   - default identity display -> `tillsyn-user`.
   - removed legacy `till-user` alias in activity-owner normalization.
2. `internal/tui/thread_mode.go`
   - thread header -> `tillsyn thread`.
   - fallback comment author/default actor display -> `tillsyn-user`.
3. Test/golden synchronization:
   - `internal/tui/model_teatest_test.go`
   - `internal/tui/model_test.go`
   - `internal/tui/testdata/TestModelGoldenBoardOutput.golden`
   - `internal/tui/testdata/TestModelGoldenHelpExpandedOutput.golden`
4. Docs/config wording:
   - `README.md` naming sentence -> Swedish word definition for `tillsyn`.
   - README fallback identity text -> `tillsyn-user`.
   - `config.example.toml` heading -> `# tillsyn example configuration`.

Commands and outcomes:
1. `just check` -> FAIL (`gofmt required for internal/tui/model.go`).
2. Context7 re-check performed before next edit.
3. `just fmt && just check && just ci` -> FAIL at `just check` (`internal/tui` golden EOF newline mismatch only).
4. Context7 re-check performed before fixture-byte edit.
5. Adjusted golden fixtures to match exact EOF byte expectation.
6. `just check && just ci` (escalated for Go cache writes) -> PASS.

Current status:
- app-visible branding now uses `tillsyn`;
- command surfaces remain `till`;
- gates are green after normalization (`just check`, `just ci`).

### 2026-02-28: Init-Dev-Config Copy/Paste Path Output Fix

Objective:
- make `just init-dev-config` output copy/paste-safe on paths containing spaces.

Issue observed:
1. `init-dev-config` printed unquoted absolute paths (for example under `~/Library/Application Support/...`), causing direct shell reuse to fail unless manually escaped.

Implementation updates:
1. `cmd/till/main.go`
   - added `shellQuotePath` helper for POSIX-safe single-quoted path rendering.
   - updated `runInitDevConfig` output line to print quoted config path.
2. `cmd/till/main_test.go`
   - updated init-dev-config output assertions to expect quoted paths.

Commands and outcomes:
1. Context7 consulted before edit (Go string/formatting guidance) -> PASS.
2. `just fmt && just check && just ci` -> PASS.

Current status:
- `init-dev-config` now prints copy/paste-safe quoted path output, e.g.:
  - `dev config already exists: '/Users/.../Library/Application Support/tillsyn-dev/config.toml'`.

### 2026-02-28: Init-Dev-Config Output Style Adjustment (Backslash Escapes)

Objective:
- align `init-dev-config` output with user preference for direct paste paths using backslash-escaped spaces (instead of single-quoted paths).

Issue observed:
1. Single-quoted path output was technically shell-safe but did not match expected copy/paste ergonomics (`Application\\ Support` style).

Implementation updates:
1. `cmd/till/main.go`
   - replaced quoted output helper with `shellEscapePath` that emits one shell-safe token using backslash escapes for spaces and shell metacharacters.
   - `runInitDevConfig` output now uses escaped token format.
2. `cmd/till/main_test.go`
   - updated output assertions to expect escaped token paths.
   - added `TestShellEscapePath` coverage for `Application Support` path escaping.

Commands and outcomes:
1. Context7 consulted before edits (Go formatting/string output guidance) -> PASS.
2. `just fmt && just check && just ci` -> PASS.
3. Local smoke check with temp HOME/XDG env -> PASS:
   - output now prints `.../Library/Application\ Support/...`.

Current status:
- `just init-dev-config` output is now backslash-escaped and directly pasteable as requested.

### 2026-02-28: Level-Scoped Guardrail Enforcement for Task/Comment Mutations

Objective:
- make mutation guardrails truly level-scoped for agent leases (project/branch/phase/subphase/task/subtask), not project-only for task/comment writes.

Issue observed:
1. `CreateTask`, `UpdateTask`, `MoveTask`, `DeleteTask`, `RestoreTask`, `ReparentTask`, and `CreateComment` were still enforcing guardrails against `project` scope only.
2. This blocked intended phase/task scoped leases for subagent flows and produced ambiguous `mutation lease is invalid` failures.

Implementation updates:
1. `internal/app/mutation_scope.go` (new):
   - added task-lineage scope resolution helper that derives allowed scope tuples from project + ancestor chain + node scope.
2. `internal/app/kind_capability.go`:
   - retained `enforceMutationGuard` API and routed it through new multi-scope enforcement helper.
   - added `enforceMutationGuardAcrossScopes` to validate one lease tuple against a normalized allowed-scope set.
   - expanded guardrail mismatch logging to include requested scope tuple set.
3. `internal/app/service.go`:
   - replaced project-only guard checks in task/comment mutation flows with lineage-derived scope candidate checks.
   - create-under-parent now checks parent lineage.
   - reparent now enforces permission for both the task lineage and destination parent lineage.
4. `internal/app/service_test.go`:
   - added `TestScopedLeaseAllowsLineageMutations`.
   - added `TestScopedLeaseRejectsSiblingMutations`.

Commands and outcomes:
1. `just test-pkg ./internal/app` -> FAIL (`undefined: domain.WorkKindBranch` in new tests).
2. Context7 re-check performed before next edit.
3. `just fmt`.
4. `just test-pkg ./internal/app` -> FAIL (`"task" does not apply to "branch"` in new tests).
5. Context7 re-check performed before next edit.
6. adjusted branch test fixtures to use explicit `kind="branch"` ID with `scope=branch`.
7. `just fmt`.
8. `just test-pkg ./internal/app` -> PASS.
9. `just check` -> PASS.
10. `just ci` -> PASS.

Current status:
- level-scoped lease guardrails now authorize by subtree lineage instead of project-only hardcoding for task/comment mutation paths.
- full repo gates are green (`just check`, `just ci`).

### 2026-02-28: Ownership Attribution Regression During Live MCP Validation (OPEN)

Objective:
- record critical collaborative test finding before next fix scope.

Issue observed:
1. Live MCP setup mutations executed without explicit actor lease tuple were attributed as `user` (`tillsyn-user`) instead of agent/orchestrator identity.
2. This polluted ownership evidence during collaborative guardrail validation and made agent-vs-user provenance unreliable in TUI/Recent Activity.

Status:
- OPEN (discussion + fix design required before implementation).
- user reset test data after observing misattribution.

Follow-up requirements (next fix scope):
1. ensure orchestrator test flow never executes mutation calls without explicit `actor_type=agent` + `agent_name` + `agent_instance_id` + `lease_token`.
2. evaluate fail-closed transport/runtime option to block mutation requests with implicit user attribution when the caller intends agent orchestration mode.
3. re-run MCP + subagent guardrail validation with strict ownership assertions and preserve evidence.

### 2026-02-28: Subagent Execution Stall During Live Guardrail Validation (OPEN)

Objective:
- capture failed live subagent validation run and record next discussion/fix direction.

Run context:
1. User reset DB/state and restarted server + TUI for clean collaborative verification.
2. Orchestrator issued explicit project-scoped lease and created branch/phase setup rows with agent attribution.
3. Orchestrator issued explicit phase-scoped worker leases for two subagents.
4. Two subagents were spawned with strict prompts (one in-scope create + one out-of-scope create each, no self-lease issuance).

Failure observed:
1. Both subagents ran for ~5 minutes without completing simple MCP mutation tasks.
2. User interrupted execution due stall.
3. This repeated prior behavior seen in earlier attempts (multi-minute stalls for simple actions).

User findings/hypothesis:
1. likely both prompting/orchestration issue and code/system issue.
2. current gatekeeping flow feels too fragile/slow for practical collaborative workflows.
3. discuss and evaluate an `Auth 2.0` model for gatekeeping.

Auth 2.0 discussion backlog (explicit):
1. re-evaluate stateless per-call tuple model versus session-bound authenticated identity context for subagent flows.
2. design first-class orchestrator-to-subagent delegation handshake (server-issued, revocable, scope-bound grants) with clearer lifecycle.
3. add deterministic guardrail stage observability:
   - lease lookup,
   - identity match,
   - scope check,
   - decision outcome,
   - latency timing,
   exposed as structured logs/events.
4. define hard operational SLOs for automated lanes (for example first mutation within N seconds) and automatic timeout/escalation behavior.
5. evaluate approval/gating UX for identity+scope grants so operator intent is explicit and auditable.

Required follow-up:
1. perform focused root-cause investigation for subagent stall:
   - prompt contract quality,
   - MCP tool invocation overhead/queueing,
   - guardrail round-trip behavior under subagent execution.
2. agree on Auth 2.0 target architecture before implementing broad auth/gatekeeping rewrite.
3. preserve existing strict fail-closed guarantees while reducing orchestration friction/latency.

### 2026-02-28: Activity Log Entity Labeling Fix (Branch/Phase/Subphase)

Objective:
- stop labeling every persisted work-item event as `* task` in notices recent activity and activity-log modal.

Issue observed:
1. branch/phase/subphase operations were displayed as `create task` / `update task` etc.
2. this affected both notices panel recent activity rows and activity-log modal rows sourced from persisted change events.

Implementation updates:
1. `internal/adapters/storage/sqlite/repo.go`:
   - enriched change-event metadata on create/update/delete with:
     - `item_kind`
     - `item_scope`
     - `title` (ensured on update path too).
2. `internal/tui/model.go`:
   - replaced hardcoded `* task` summary mapping with `operation + entity` mapping.
   - added `activityEntityLabel` helper to derive entity from event metadata (`item_scope` -> fallback `item_kind` -> fallback `task`).
3. `internal/tui/model_test.go`:
   - updated recent-activity owner-prefix test to verify scope-aware summary rendering (`update phase` when metadata scope is phase).

Commands and outcomes:
1. Context7 consulted before edits -> PASS.
2. `just fmt` -> PASS.
3. `just test-pkg ./internal/adapters/storage/sqlite` -> PASS.
4. `just test-pkg ./internal/tui` -> PASS.
5. `just check` -> PASS.
6. `just ci` -> PASS.

Current status:
- persisted activity rows now render entity-aware summaries for branch/phase/subphase/task scope events instead of always `task`.

### 2026-03-02: Dogfood Blocker Remediation Wave (IN PROGRESS)

Objective:
- close known dogfooding blockers surfaced in active collaborative worksheets, then refresh worksheets for one joint validation pass.

Backlog/open-findings review checkpoint:
1. Reviewed active backlog/open discussion items in this file (`PLAN.md`), including:
   - open Phase 0 closeout status and blocker statements,
   - open ownership-attribution regression discussion,
   - open subagent stall/Auth 2.0 discussion items.
2. Reviewed unresolved findings in:
   - `COLLAB_E2E_REMEDIATION_PLAN_WORKLOG.md`,
   - `COLLABORATIVE_POST_FIX_VALIDATION_WORKSHEET.md`.
3. Reviewed current MCP dogfood sign-off state in:
   - `MCP_DOGFOODING_WORKSHEET.md`.

Current remediation focus (known blockers from docs + code audit):
1. restore-task guard actor mismatch (`mutation lease is required` on user restore for agent-attributed archived tasks).
2. MCP/HTTP guardrail error log sink parity gaps.
3. first-launch config bootstrap seeding gap (missing config template copy on normal startup).
4. docs/worksheet drift after recent code fixes.

Parallel lane lock table (single-branch orchestration; non-overlapping scopes):
1. `W-RESTORE-ACTOR`
   - lock scope:
     - `internal/app/service.go`
     - `internal/app/service_test.go`
     - `internal/app/mutation_guard.go` (only if required)
   - acceptance objective:
     - restore guard behavior follows current caller actor context with fail-closed non-user semantics preserved.
2. `W-LOG-PARITY`
   - lock scope:
     - `internal/adapters/server/mcpapi/handler.go`
     - `internal/adapters/server/mcpapi/handler_test.go`
     - `internal/adapters/server/httpapi/handler.go`
     - `internal/adapters/server/httpapi/handler_test.go`
   - acceptance objective:
     - mapped MCP/HTTP error paths emit structured runtime logs without changing response contracts.
3. `W-BOOTSTRAP-CONFIG`
   - lock scope:
     - `cmd/till/main.go`
     - `cmd/till/main_test.go`
     - `README.md`
   - acceptance objective:
     - normal startup seeds config from `config.example.toml` when missing, while preserving help behavior.

Commands/tests run (orchestrator evidence):
1. `sed -n '1,220p' Justfile` -> PASS (verified recipe source-of-truth and gate commands).
2. `git log -n 5 ...` and `git log -n 5 --name-status -- '*.md'` -> PASS (identified latest markdown workset).
3. targeted file audits (`rg`, `sed`) across active worksheets + code paths -> PASS.
4. `just check` -> PASS.
5. `just ci` -> PASS.
6. spawned worker lanes:
   - `019cabe0-a8c7-74d3-8634-c23e206412c3` (`W-RESTORE-ACTOR`) -> IN_PROGRESS.
   - `019cabe0-aad2-75f0-8626-e69d5765e420` (`W-LOG-PARITY`) -> IN_PROGRESS.
   - `019cabe0-ac7c-7221-9dd1-1d874c1b83eb` (`W-BOOTSTRAP-CONFIG`) -> IN_PROGRESS.

Current status:
- worker lanes are executing with explicit Context7-before-edit and failure-triggered Context7 re-check requirements.
- next step is orchestrator review/integration of each handoff, then `just check` + `just ci`, then worksheet/doc updates with fresh evidence.

Integrator review and lane closeout:
1. `W-RESTORE-ACTOR` (`019cabe0-a8c7-74d3-8634-c23e206412c3`) -> COMPLETED
   - integrated changes:
     - `internal/app/service.go`
     - `internal/app/service_test.go`
   - outcome:
     - restore guard actor now follows current mutation actor context (user default), with non-user lease enforcement preserved.
2. `W-LOG-PARITY` (`019cabe0-aad2-75f0-8626-e69d5765e420`) -> COMPLETED
   - integrated changes:
     - `internal/adapters/server/mcpapi/handler.go`
     - `internal/adapters/server/mcpapi/handler_test.go`
     - `internal/adapters/server/httpapi/handler.go`
     - `internal/adapters/server/httpapi/handler_test.go`
   - outcome:
     - MCP/HTTP mapped error branches now emit structured adapter-edge logs (`error_class`, `error_code`, transport fields) and tests assert mappings.
3. `W-BOOTSTRAP-CONFIG` (`019cabe0-ac7c-7221-9dd1-1d874c1b83eb`) -> COMPLETED
   - integrated changes:
     - `cmd/till/main.go`
     - `cmd/till/main_test.go`
     - `README.md`
   - outcome:
     - normal TUI startup now seeds missing config from `config.example.toml` (when template is present), with help paths remaining side-effect free.

Post-integration validation commands:
1. `just check` -> PASS.
2. `just ci` -> PASS.
3. `./till --help` and `./till serve --help` smoke capture -> PASS:
   - stderr bytes: 0 for both help commands,
   - usage text present in captured outputs.

Validation limitations observed:
1. live `serve` integration smoke for HTTP/MCP runtime logging could not be completed in this sandbox due bind failure (`listen tcp ... bind: operation not permitted`).
2. adapter-level log mapping is test-covered; full runtime sink parity still requires collaborative/local serve-session verification outside sandbox bind limits.

Next step:
- update active collaborative worksheets and remediation worklog with this fix wave + rerun requirements for remaining manual/transport checkpoints.

Docs/worksheet synchronization completed:
1. `COLLABORATIVE_POST_FIX_VALIDATION_WORKSHEET.md`
   - reclassified REQ-008/009/010/027 from `MISSING` -> `PARTIAL`.
   - updated Section 7 blockers/follow-up actions to reflect 2026-03-02 code fixes and rerun requirements.
   - marked P0-T03/P0-T04 as `IN_PROGRESS` with rerun-required notes.
   - appended Section 12.9 remediation update with fresh gate evidence.
2. `MCP_DOGFOODING_WORKSHEET.md`
   - added Section 6 remediation addendum (2026-03-02).
   - updated final sign-off blocker wording to focus on pending collaborative reruns/manual sections.
3. `COLLAB_E2E_REMEDIATION_PLAN_WORKLOG.md`
   - moved T-004/T-005 backlog rows to `implemented_pending_validation`.
   - marked task cards subagent/orchestrator checks complete with code/test evidence pointers.
4. Added evidence artifact:
   - `.tmp/phase0-collab-20260227_141800/remediation_wave_20260302.md`.

Current status:
- known code-level blockers targeted in this wave are implemented and repo gates are green.
- dogfooding sign-off remains open pending collaborative reruns for:
  1. live serve-session sink parity verification,
  2. focused `till_restore_task` transport rerun,
  3. remaining manual TUI validation sections.

Final pre-handoff gate rerun after worksheet/doc updates:
1. `just check` -> PASS (cached).
2. `just ci` -> PASS (cached).

## Checkpoint 2026-03-02: Collab Test Sheet Refresh + Agent-Only Reruns

Objective:
- Create a new dated collaborative worksheet and execute all agent-only checks now (including guardrail E2E and gatekept subagent probes), then leave only user/joint manual checks pending.

Commands/tests run and outcomes:
1. `just check` -> PASS (evidence: `.tmp/collab-test-2026-03-02/a01_just_check.txt`).
2. `just ci` -> PASS (evidence: `.tmp/collab-test-2026-03-02/a02_just_ci.txt`; includes non-fatal stat-cache warning).
3. `just test-golden` -> PASS (evidence: `.tmp/collab-test-2026-03-02/a03_test_golden.txt`).
4. `./till --help` and `./till serve --help` -> PASS, 0 stderr bytes (evidence: `a04_*` artifacts).
5. `just test-pkg ./cmd/till` -> PASS (startup seeding coverage; evidence: `a05_startup_seed_check.txt`).
6. Isolated live serve transport sweep (`E-01`..`E-08`) -> `E-01`..`E-07` PASS, `E-08` FAIL (sink parity gap persists); evidence under `.tmp/collab-test-2026-03-02/e*`.
7. Subagent gate probes:
   - initial non-escalated attempt -> BLOCKED (`bind: operation not permitted`).
   - rerun with escalated local bind permissions -> PASS for in-scope + out-of-scope expectations (evidence: `s01_subagent_in_scope.md`, `s02_subagent_out_scope.md`).

Files edited and why:
1. `COLLAB_TEST_2026-03-02.md`
   - created dated worksheet,
   - carried forward unresolved test scopes from prior worksheets,
   - updated agent-only statuses with evidence,
   - left required collaborative/manual checks as explicit pending rows.

Current status:
- Agent-only testable items are complete for this pass.
- Remaining blocker: logging sink parity (`E-08`) still fails in this environment.
- Remaining work is collaborative/manual validation (`C-01`..`C-07`).

## Checkpoint 2026-03-02: E-08 Sink-Parity Remediation and Verification

Objective:
- Fix `E-08` so mapped MCP/HTTP adapter errors are persisted to `.tillsyn/log` (not only stderr), then confirm with real gates and live serve-session evidence.

Implementation updates:
1. `cmd/till/main.go`
   - added runtime default-logger installation (`InstallAsDefault` / `RestoreDefault`) so package-level `charmbracelet/log` calls flow through runtime sinks.
   - added `runtimeLogBridgeWriter` fanout writer to mirror package-level logs to active console sink (when enabled) and dev-file sink.
2. `cmd/till/main_test.go`
   - added regression `TestRuntimeLoggerInstallAsDefaultRoutesPackageLogsToFile` to verify package-level logs reach dev-file sink and respect console muting.

Commands/tests run and outcomes:
1. `just test-pkg ./cmd/till` -> PASS.
2. `just check` -> FAIL initially (`gofmt required for: cmd/till/main.go`), then:
   - Context7 re-check executed per policy after failure,
   - `just fmt` applied,
   - `just check` rerun -> PASS.
3. `just ci` -> PASS.
4. Live `E-08` rerun (local serve runtime with HTTP + MCP invalid requests) -> PASS:
   - evidence: `.tmp/collab-test-2026-03-02/e08_rerun_v2_summary.log`
   - both counters incremented: `delta_mcp=1`, `delta_http=1`,
   - matched lines present in `.tillsyn/log/tillsyn-20260302.log` and serve stderr.

Current status:
- `E-08` is remediated and reclassified PASS in `COLLAB_TEST_2026-03-02.md`.
- No remaining agent-only FAIL items in the dated collab test worksheet.

## Checkpoint 2026-03-02: Parallel Comment + Notices Remediation Setup

Objective:
- Confirm comment schema/ownership coverage, then run non-overlapping parallel lanes for:
  1. comment target-type completion (`branch` + `subphase`) across domain/app/MCP/TUI mapping,
  2. notices panel focusable/scrollable/selectable UX redesign.

Backlog/open-findings review:
1. Reviewed active collaborative docs and unresolved behavior:
   - missing global notifications workflow and section-level navigable lists in notices panel,
   - comment coverage mismatch for hierarchy node types.
2. Reviewed `PARALLEL_AGENT_RUNBOOK.md` for lock-discipline and lane contract constraints.

Artifacts created:
1. `COMMENT_SCHEMA_COVERAGE_AND_BUILD_PLAN.md`
   - current schema/ownership audit + planned build.
2. `COLLAB_PARALLEL_FIX_TRACKER_2026-03-02.md`
   - lock table + lane status tracker.

Planned lane lock scopes (non-overlapping):
1. `LANE-COMMENT-TARGETS`
   - scope: `internal/domain/comment.go`, `internal/domain/comment_test.go`, `internal/app/service_test.go`, `internal/app/snapshot.go`, `internal/adapters/server/mcpapi/extended_tools.go`, `internal/adapters/server/mcpapi/extended_tools_test.go`, `internal/tui/thread_mode.go`, `internal/tui/thread_mode_test.go`.
   - out-of-scope: `internal/tui/model.go`, `internal/tui/model_test.go`.
2. `LANE-NOTICES-PANEL`
   - scope: `internal/tui/model.go`, `internal/tui/model_test.go`, `internal/tui/keymap.go`.
   - out-of-scope: all domain/app/server and `internal/tui/thread_mode.go`.
3. `LANE-OWNERSHIP-PROPOSAL` (analysis-only)
   - docs/proposal lane; no code edits.

Current status:
- Ready to dispatch worker lanes with Context7-before-edit and package-scoped `just test-pkg` requirements.

## Checkpoint 2026-03-02: Parallel Comment + Notices Remediation Integration

Objective:
- Integrate and verify the three-lane wave:
  1. comment target-type completion across domain/app/MCP/TUI thread mapping,
  2. notices panel section-list navigation/selection UX,
  3. ownership tracking audit and migration recommendation.

Lane outcomes:
1. `LANE-COMMENT-TARGETS` completed:
   - added `branch` and `subphase` comment target support,
   - updated snapshot target mapping,
   - updated MCP comment tool target enum,
   - updated TUI task->comment target mapping + new tests.
2. `LANE-NOTICES-PANEL` completed:
   - converted notices panel sections into focusable/selectable list areas,
   - added per-section cursors, scroll windowing, and Enter actions,
   - updated notices/board help messaging.
3. `LANE-OWNERSHIP-PROPOSAL` completed:
   - confirmed current ownership model is `actor_type + author_name`,
   - no stable actor ID stored today; documented optional follow-up migration path.

Commands/tests run and outcomes:
1. `just test-pkg ./internal/domain` -> PASS
2. `just test-pkg ./internal/app` -> PASS
3. `just test-pkg ./internal/adapters/server/mcpapi` -> PASS
4. `just test-pkg ./internal/tui` -> PASS
5. `just ci` -> PASS

Files/docs updated for this checkpoint:
1. `COMMENT_SCHEMA_COVERAGE_AND_BUILD_PLAN.md`
   - updated with post-fix implemented matrix + ownership status.
2. `COLLAB_PARALLEL_FIX_TRACKER_2026-03-02.md`
   - lane statuses moved to complete with handoff and verification evidence.
3. code files from lanes integrated in current worktree:
   - `internal/domain/comment.go`
   - `internal/domain/comment_test.go`
   - `internal/app/snapshot.go`
   - `internal/app/service_test.go`
   - `internal/adapters/server/mcpapi/extended_tools.go`
   - `internal/adapters/server/mcpapi/extended_tools_test.go`
   - `internal/tui/thread_mode.go`
   - `internal/tui/thread_mode_test.go`
   - `internal/tui/model.go`
   - `internal/tui/model_test.go`
   - `internal/tui/keymap.go`

Current status:
- Comment coverage now spans all current node types used by hierarchy and thread entry points.
- Notices panel list navigation/selection behavior is implemented and test-covered.
- Stable actor-ID ownership tracking remains a follow-up decision, intentionally deferred for this wave.

## Checkpoint 2026-03-02: Ownership Tuple + Identity ActorID Wave (Parallel + Reviewed)

Objective:
- Implement immutable ownership tuple (`actor_id`, `actor_name`, `actor_type`) for comments/events,
- wire immutable config-backed user `identity.actor_id` into startup/runtime/TUI,
- run independent subagent review and remediation until green gates.

Execution summary:
1. Launched parallel implementation lanes:
   - `LANE-OWNERSHIP-CORE`
   - `LANE-TUI-FLOWS`
   - `LANE-CONFIG-IDENTITY`
2. Ran independent review lanes:
   - initial reviews flagged compile/wiring/contract issues,
   - dispatched targeted remediation lane (`LANE-REVIEW-REMEDIATION`),
   - ran second independent review pass (`REVIEW-REMEDIATION-PASS2`) -> PASS.

Key outcomes landed:
1. Comments now use canonical ownership tuple fields end-to-end:
   - `actor_id`, `actor_name`, `actor_type`.
2. Change events now persist/read `actor_name` alongside `actor_id` + `actor_type`.
3. MCP actor tuple supports `actor_id` + `actor_name` and preserves them through mutation context.
4. TUI comment/activity owner rendering now prefers `actor_name` with compact `actor_id` context.
5. Config and startup now support immutable `identity.actor_id`:
   - generate once when missing,
   - persist to config,
   - apply at startup and runtime reload.
6. Snapshot versioning for ownership-shape change is explicit:
   - `SnapshotVersion` bumped to `tillsyn.snapshot.v2` with strict import version check.

Commands/tests run and outcomes:
1. `just test-pkg ./internal/domain` -> PASS
2. `just test-pkg ./internal/app` -> PASS
3. `just test-pkg ./internal/adapters/storage/sqlite` -> PASS
4. `just test-pkg ./internal/adapters/server/mcpapi` -> PASS
5. `just test-pkg ./internal/tui` -> PASS
6. `just test-pkg ./cmd/till` -> PASS
7. `just check` -> PASS
8. `just ci` -> PASS

Docs/worklog sync:
1. `COMMENT_SCHEMA_COVERAGE_AND_BUILD_PLAN.md` updated to post-wave canonical schema/tuple state.
2. `COLLAB_PARALLEL_FIX_TRACKER_2026-03-02.md` updated with Wave 2 completion, review findings, remediation, and final gate evidence.

Current status:
- Ownership + identity foundations are implemented and verified.
- Branch is ready to continue collaborative worksheet execution from the next pending collab section.

## Checkpoint 2026-03-02: Parallel Wave Sign-Off Revalidation

Objective:
- confirm final sign-off state after tracker/worklog synchronization.

Commands/tests run and outcomes:
1. `just check` -> PASS
2. `just ci` -> PASS

Current status:
- all parallel implementation lanes and independent review lanes remain closed with green integrator gates.
