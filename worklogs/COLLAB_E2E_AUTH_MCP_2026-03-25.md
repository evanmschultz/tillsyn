# Collaborative E2E Auth And MCP Worksheet

Created: 2026-03-25
Updated: 2026-03-29
Status: Active. `C2` and `C3` are proven live, the delegated child-self-claim/requester-cleanup remediation slice is green locally, the refreshed `C4` rerun now proves child self-claim plus the current builder-vs-QA `create-child` policy split on the live MCP path, and the `C5` approved-path handoff/auth-context blocker is fixed locally while the fresh live `C5` runtime rerun now passes through `update_handoff`, missing-lease fail-closed, readiness/discovery, revoke visibility, and post-revoke fail-closed checks; the subsequent TUI follow-up first exposed a coordination-screen overflow/scroll bug, then a live/history readability gap, then a board/help cleanup pass that removes the legacy `Selection` panel and keeps coordination guidance in the bottom/help surfaces, and the latest follow-up now fixes actionable coordination routing from the board/global notifications panels plus `enter`-to-detail behavior inside `Coordination`, with the local slice green again on `just test-pkg ./internal/tui`, `just fmt`, `just check`, and `just ci`, and one fresh user reopen still pending to confirm the rebuilt TUI.

## Purpose

Run one dated collaborative end-to-end auth and MCP dogfood pass that:

1. keeps `PLAN.md` as the canonical status and evidence ledger,
2. keeps human time focused on live behavior that still needs real proof,
3. treats historically fixed auth-review and notification-routing regressions as short spot-checks only unless they fail again,
4. fully reruns the fresh orchestrator, claim/resume, delegated subagent, revoke, anti-adoption, lease, handoff, and recovery flows.

## Authority And Execution Rules

1. `PLAN.md` remains the canonical source of truth for status, findings, and closeout.
2. This worksheet exists because the user explicitly requested a new split collaborative worksheet for the live run.
3. Stop on fail:
   - if any section fails, log the finding in `PLAN.md`,
   - fix only that scope,
   - rerun the same section,
   - then continue.
4. Do not reopen already-fixed UX areas into a long rediscovery loop unless the live spot-check shows a regression.
5. Prefer real product paths:
   - MCP request creation and claim/resume over shell glue,
   - TUI approval/review over shell-only operator approval,
   - product discovery/readiness surfaces over insider memory.

## Current Baseline

1. Slice 7 follow-up is green locally and remotely.
2. GitHub Actions run `23569389061` finished green before this worksheet was prepared.
3. The active pre-collab checklist already lives in `PLAN.md`.
4. Historical auth UX worksheets are reference-only inspiration for scope split, not active authority.
5. Cross-process auth wait remains the intended live path for this worksheet.
6. The latest pre-rerun blocker was Windows SQLite-open portability under GitHub Actions:
   - the URI-normalization-only fix was not sufficient on Windows,
   - the current local follow-up stack now opens SQLite with the raw filesystem path and applies the required PRAGMAs after `sql.Open(...)`,
   - QA then required tighter proof for that pivot, so the local regression coverage was strengthened to assert the full PRAGMA contract and the real file-backed `Open(temp-path)` path,
   - local evidence on the tightened follow-up is green:
     - `just fmt`
     - `just test-pkg ./internal/adapters/storage/sqlite`
     - `just test-pkg ./cmd/till`
     - `just check`
     - `just ci`
   - replacement run `23586624405` proved the original Windows SQLite-open failure is resolved,
   - that run then exposed two separate Windows-only test regressions in `internal/adapters/livewait/localipc` and `internal/tui`,
   - the current local follow-up now fixes both:
     - `newID()` in the local IPC broker no longer relies on wall-clock resolution alone and is now proven with a frozen same-tick regression,
     - stale-subscription cleanup now targets a closed loopback address instead of a hard-coded dead port,
     - the archived-task notice Enter test now targets the attention row directly and runs the immediate reload command without relying on the generic timeout helper,
     - the mouse-wheel board-selection regression test now sets its starting board state explicitly before wheel input,
   - local evidence on that follow-up is green:
     - `just fmt`
     - `just test-pkg ./internal/adapters/livewait/localipc`
     - `just test-pkg ./internal/tui`
     - `just check`
     - `just ci`
   - do not resume the live worksheet until the next replacement GitHub Actions run is green.

## Scope Split

### Spot-Check Only

1. Runtime/path parity and clean `Ctrl-C`.
2. Auth review UX shape:
   - approve-default,
   - deny note-first,
   - no hidden-hotkey dependence,
   - human-readable scope labels with raw path still visible.
3. Historical notification-routing fixes.
4. Role-only or targetless handoff rendering.

### Full Live Rerun Required

1. CLI/operator bootstrap and readiness path.
2. Unauthenticated mutation fails closed.
3. Fresh orchestrator auth request through MCP.
4. TUI approval, waiting state, and native requester claim/resume.
5. Authenticated mutation after approval.
6. Denied and canceled terminal request states.
7. Revoke and fail-closed retry behavior.
8. Orchestrator-created builder and QA auth choreography.
9. Anti-adoption and requester-bound claim protection.
10. In-scope versus out-of-scope mutation behavior.
11. Lease and handoff lifecycle visibility across CLI, TUI, and MCP.
12. Recovery/readiness visibility for active or hanging collaboration state.
13. Name-first human clarity across the live collaboration surfaces.

## Section C0: Runtime And Operator Preflight

Goal:
- confirm the product starts cleanly from the current runtime contract and that the operator can discover the project state without insider memory.

Driver:
- human with agent observing and recording.

Steps:
1. Run `./till`.
2. Exit cleanly.
3. Run `./till mcp`.
4. Exit with `Ctrl-C` and confirm shutdown looks normal rather than error-like.
5. Run `./till serve`.
6. Exit cleanly.
7. Run `./till project list`.
8. If the intended test project does not exist, run `./till project create`.
9. Run `./till project show --project-id <project-id>`.
10. Run `./till project discover --project-id <project-id>`.
11. Run `./till capture-state --project-id <project-id>`.

Expected:
1. Runtime startup/shutdown is clean.
2. Project discovery works from product surfaces, not insider memory.
3. Names lead and ids remain visible but secondary.
4. Readiness/discovery output points clearly to the next auth and coordination actions.

Evidence:
- project name:
- project id:
- pass/fail notes:

## Section C1: TUI Human Spot-Check

Goal:
- quickly verify the historically fixed auth-review and coordination readability issues still feel right in live use.

Driver:
- human with agent observing and recording.

Steps:
1. Start `./till`.
2. Open the project picker and confirm project names lead.
3. Open the notifications path and the `coordination` or `auth-access` surface.
4. Verify rows are scan-friendly:
   - requests,
   - sessions,
   - leases,
   - handoffs.
5. When an auth request is available later in the run, verify:
   - approve is the obvious default,
   - deny is note-first,
   - scope picker is understandable,
   - `enter`, `esc`, and return paths feel sane.

Expected:
1. No obvious id-first confusion.
2. No cramped modal feel for auth review.
3. Coordination screen is readable enough for a human operator to monitor the run.

Evidence:
- pass/fail notes:
  - BLOCKED on 2026-03-25 after the first live auth-review retest, then remediated in code and automated gates.
  - Confirmed scope picker currently matches the locked `project[/branch[/phase...]]` contract.
  - Landed remediation before rerun:
    - auth review `enter` now opens an explicit confirm modal instead of applying immediately,
    - auth approve/deny confirm modal now defaults to `confirm`,
    - denial note flow remains note-first and now also requires the same final confirm step,
    - auth review notes now stay optional and blank by default,
    - normal dogfood runtime now persists file logs under the resolved runtime `logs` path.
  - Automated evidence after remediation:
    - `just test-pkg ./internal/tui` PASS
    - `just test-pkg ./cmd/till` PASS
    - `just check` PASS
    - `just ci` PASS
  - Next live step:
    - rerun this same `C1` auth-review interaction on the fresh binary and confirm the explicit-confirm flow feels correct.
  - Active rerun request:
    - request id: `8a080168-719c-46b7-bf36-41342558010d`
    - principal: `Codex Collab Wait Orchestrator`
    - requester continuation token: `resume-c1-wait-20260325`
    - one background waiter lane is holding `till.claim_auth_request(wait_timeout=10m)` for this request so we can observe the current continuation behavior after live TUI approval or denial.

## Section C2: Fresh Orchestrator Auth Through MCP

Goal:
- prove the intended dogfood path works with a new orchestrator client instead of shell-first operator glue, including fail-closed unauthenticated mutation and waiting/claim semantics.

Driver:
- joint human + new Codex/orchestrator instance + current agent recording.

Steps:
1. Open a fresh Codex instance that connects to `./till mcp`.
2. From that new instance, use MCP discovery to confirm the project is visible.
3. Before requesting auth, call one mutating MCP tool that requires a session:
   - preferred seam: `till.create_handoff` or `till.issue_capability_lease`.
4. Verify the call fails closed and points back toward the auth request path instead of mutating anything.
5. From that new instance, create one auth request for the target project scope through MCP and include a requester-owned `resume_token` in `continuation_json`.
6. Immediately call the supported MCP claim or continuation path with a wait timeout.
7. In the TUI, verify the request appears in the correct notification/review surface.
8. In the TUI, approve the request with any needed note, path narrowing, or TTL adjustment.
9. From the new Codex instance, claim or resume the approved request through the supported MCP continuation path.
10. Create two short terminal-state requests:
   - one that the human or operator denies,
   - one that the human or operator cancels.
11. Verify neither denied nor canceled request can later return a session secret.

Expected:
1. Unauthenticated mutation fails closed before any auth exists.
2. For the current local cross-process dogfood path, `till.claim_auth_request(wait_timeout=...)` should now stay open and wake on approve/deny/cancel without app-layer polling, even when the waiter and reviewer are in different local processes.
3. The request is visible in TUI without shell spelunking.
4. Approval happens in the dedicated TUI flow.
5. Claim/resume works natively for the same requester.
6. Denied and canceled requests never yield a session secret.
7. No manual shell copying is needed as the primary path.

Evidence:
- orchestrator display name: `Codex C2 Orchestrator`
- request id: `a9d80803-0c60-48f4-a660-0fa64866a6ff`
- approved path: `project/cead38cc-3430-4ca1-8425-fbb340e5ccd9`
- denied request id: `1b96f171-7552-4664-a679-8979f67918e6`
- canceled request id: `ccf66945-76ac-4f04-8c02-6f65ac34cce8`
- pass/fail notes:
  - PASS: requester created the auth request through MCP only.
  - PASS: requester immediately called `till.claim_auth_request(wait_timeout=10m)` and stayed blocked while the human resolved the request in TUI.
  - PASS: human approval in TUI woke the same MCP claim call directly; no extra lookup call was needed to discover approval.
  - PASS: claim result returned the approved request plus `session_secret`.
  - issued session id: `1f6b5def-1cba-47b9-94a4-05993d00055a`
  - PASS: a second requester created a denied-path auth request through MCP only.
  - PASS: requester immediately called `till.claim_auth_request(wait_timeout=10m)` and stayed blocked while the human denied the request in TUI.
  - PASS: the same waiting MCP claim call returned the denied terminal request directly with no `session_secret`.
  - MCP follow-up slice landed after the denied-path rerun:
    - `till.cancel_auth_request` is now exposed through MCP with requester-bound continuation proof (`request_id`, `resume_token`, `principal_id`, `client_id`, optional `resolution_note`),
    - local evidence is green:
      - `just test-pkg ./internal/adapters/server/common` PASS
      - `just test-pkg ./internal/adapters/server/mcpapi` PASS
      - `just check` PASS
      - `just ci` PASS
  - PASS: the canceled request path now also works over MCP only.
    - waiting claimant stayed blocked on `till.claim_auth_request(wait_timeout=10m)`,
    - requester called `till.cancel_auth_request(...)` with its continuation proof,
    - the waiting MCP claim resumed directly with `state = canceled`,
    - no `session_secret` was returned.
  - `C2` outcome:
    - PASS: approve, deny, and cancel are all now proven live over the current local MCP wait path.

Status note before continuing:
- `C2` should still prove current fail-closed auth, TUI visibility, and native claim/resume behavior.
- `C2` can now prove the auth-specific local cross-process wake path for the default human-in-the-loop dogfood flow.
- `C2` should still not be used to claim that the broader session-aware stdio communication layer or comment/handoff consumers already exist.
- `C2` should still treat broader MCP notification reuse, disconnect-aware session cleanup, and HTTP/continuous-listening support as follow-on work.
- Automated evidence before the next live rerun:
  - `just test-pkg ./internal/tui` PASS
  - `just test-golden` PASS
  - `just test-pkg ./internal/app` PASS
  - `just check` PASS
  - `just ci` PASS
  - GitHub Actions run `23588942774` PASS (ubuntu, windows, macos, full gate, release snapshot)

## Section C3: Authenticated Mutation And Fail-Closed Revoke

Goal:
- prove the approved requester can mutate state, then loses that power immediately after revocation.

Driver:
- joint human + orchestrator instance + current agent recording.

Steps:
1. From the approved orchestrator instance, perform one authenticated mutation through MCP.
   - preferred mutation for this run: create a handoff tied to the active project.
2. Using the same approved session, perform one in-scope authenticated mutation and one out-of-scope mutation attempt.
3. Verify the in-scope mutation succeeds and the out-of-scope mutation fails closed without mutating anything.
4. Verify the successful mutation is visible from at least one human-facing surface.
5. Revoke the active session from TUI or CLI.
6. Retry the same or another authenticated mutation from the orchestrator instance.

Expected:
1. Approved session succeeds while active.
2. Out-of-scope mutation fails closed rather than mutating anything.
3. Revoked session fails closed.
4. Failure is understandable rather than ambiguous.

Evidence:
- approved request id: `bb5bedfd-abda-4e88-907a-8e3769981d3f`
- approved session id: `93631161-8778-4fde-8f43-adfeafa3515f`
- mutation used:
  - `till.create_handoff` on project `cead38cc-3430-4ca1-8425-fbb340e5ccd9`
  - created handoff id: `fec163b2-c3dc-4b5e-ba9b-11d54b4c85e9`
- out-of-scope mutation used:
  - `till.create_handoff` against project `9b40f103-72eb-49c4-b981-320fd6ab27c0`
- revoke surface used:
  - CLI: `./till auth revoke-session --session-id 93631161-8778-4fde-8f43-adfeafa3515f`
- pass/fail notes:
  - PASS: approved session created an in-scope handoff on the Evan project.
  - PASS: the same approved session failed closed on an out-of-scope mutation with `auth_denied: auth denied: authorization denied`.
  - PASS: after CLI revoke, the same session failed closed on retry with `invalid_auth: invalid session or secret: invalid authentication`.
  - FINDING: TUI session revoke is not yet discoverable enough for this flow.
    - the command-palette auth/history surface is confusing enough that it should not be the expected operator revoke path yet,
    - for this run the reliable operator revoke path was CLI,
    - this is follow-up UX work, not a blocker to auth/session correctness.

## Section C4: Builder And QA Delegation With Anti-Adoption

Goal:
- prove orchestrator-driven subagent auth stays scoped, requester-bound, and transparent to the human operator.

Driver:
- joint human + orchestrator instance + one builder instance + one QA instance + current agent recording.

Steps:
1. From the orchestrator instance, create one builder auth request through MCP for the project scope.
2. From the orchestrator instance, create one QA auth request through MCP for the project scope.
3. In the TUI, review and approve those requests.
4. From the builder instance, claim only the builder request.
5. From the QA instance, claim only the QA request.
6. Attempt continuation-binding and anti-adoption checks:
   - use a wrong `resume_token`,
   - have builder try to claim orchestrator or QA approval,
   - or have QA try to claim builder approval.
7. Confirm that the wrong requester cannot adopt the unrelated auth context.

Expected:
1. Builder and QA are clearly distinguishable in visible surfaces.
2. Wrong-token claim protection holds.
3. Requester-bound claim protection holds.
4. Human operator can see who requested what and who is now active.

Evidence:
- builder display name: `Codex Builder Agent`
- QA display name: `Codex QA Agent`
- builder request id: `1f03c7e7-026f-4bbc-b754-ef946abd867f`
- QA request id: `45475763-77e7-40ee-b4d5-1cd5c19e84db`
- anti-adoption attempt:
  - builder request with wrong `resume_token`
  - builder principal/client trying to claim QA request
  - QA principal/client trying to claim builder request
- pass/fail notes:
  - FINDING: the TUI auth-review/request inventory is still confusing enough that the user had to hunt for the right surface even though the builder and QA requests were eventually visible and operable.
    - this is follow-up UX work,
    - it does not block the underlying MCP auth/gatekeeping proof for `C4`.
  - PASS: all three anti-adoption probes failed closed with `auth request claim mismatch`.
  - FINDING: child principals could not directly claim their own on-behalf-of requests once `requested_by_actor` and `requester_client_id` were set.
    - this is not a random bug; it matches the current code/test contract,
    - continuation claim is currently requester-bound to the orchestrator for on-behalf-of requests.
  - PASS: the orchestrator requester identity successfully claimed both approved child requests and received child-scoped sessions.
    - builder issued session id: `994f07fd-9d2a-42f9-ac8b-bfb5ef1afccf`
    - QA issued session id: `9469d9ab-cc95-41d7-b04d-d196d217fde2`
  - FINDING: approved child sessions still needed capability leases before mutation.
    - without a lease tuple, mutation failed with `agent_name, agent_instance_id, and lease_token are required for authenticated agent mutations`.
  - PASS: after issuing matching project-scoped capability leases, both child sessions behaved consistently:
    - builder in-scope handoff create -> PASS
    - builder out-of-scope handoff create -> FAIL CLOSED with `auth_denied`
    - QA in-scope handoff create -> PASS
    - QA out-of-scope handoff create -> FAIL CLOSED with `auth_denied`
  - INTERPRETATION: that equal builder/QA success for handoff creation is currently product-policy behavior, not an auth failure.
    - handoff create is guarded as `CapabilityActionComment`,
    - both builder and QA currently include `comment` in their default capability actions.
  - SCOPE OF THIS SECTION:
    - `C4` proved requester-bound continuation, anti-adoption, capability-lease enforcement, and project/path scope enforcement,
    - `C4` did **not** prove the future node-type or template-driven builder-vs-QA work-lane policy model because that layer is not built yet.
  - LATER REMEDIATION LANDING:
    - approved delegated child requests now self-claim through the child principal/client,
    - delegated requester claim attempts now fail closed instead of adopting the child continuation,
    - requester-side cancel cleanup stays separate and requester-bound,
    - the future node-type/template policy model remains follow-on and was not tested by `C4`.
  - CURRENT RETEST TARGET:
    - builder child self-claims its own approved request,
    - QA child self-claims its own approved request,
    - wrong token still fails closed,
    - builder cannot adopt QA claim,
    - QA cannot adopt builder claim,
    - orchestrator/requester cannot adopt either child continuation any longer.
  - MODEL NOTE FOR THIS RETEST:
    - this proves the current interim delegated-auth hardening only,
    - orchestrator/requester still creates the delegated child auth envelope,
    - approved child principal/client owns the continuation claim,
    - requester/orchestrator keeps separate cancel/revoke cleanup authority,
    - the longer-term target is stronger still: direct child wakeup on its own live channel plus child-only session-secret delivery on the normal path.
  - CURRENT LIVE BLOCKER:
    - the corrected child-client retest pair was created and approved,
    - but the live MCP claim path still behaved like the older requester-bound contract:
      - child builder claimant failed with `auth request claim mismatch`,
      - orchestrator/requester claim against that same corrected request still succeeded,
    - that does not match the current repository code or local green tests,
    - so this `C4` rerun is blocked on restarting the live MCP side on the newest build before continuing.
  - Local remediation validation:
    - `just test-pkg ./internal/app` PASS
    - `just test-pkg ./internal/adapters/auth/autentauth` PASS
    - `just test-pkg ./internal/adapters/server/common` PASS
    - `just test-pkg ./internal/adapters/server/mcpapi` PASS
    - `just check` PASS
    - `just ci` PASS
  - FRESH LIVE RERUN RESULT ON THE RESTARTED MCP PATH:
    - fresh builder request id: `fad675d9-e2e4-4e14-86f3-9f03c4bd0a33`
    - fresh QA request id: `30f19c52-79bb-4a2f-9fbd-63c2e34f2127`
    - pre-approval child `till.claim_auth_request(wait_timeout=1s)` returned `waiting = true` for both requests while pending, confirming the live MCP path was now on the refreshed child-owned continuation code.
    - PASS: builder child self-claimed the builder request and received builder session id `e77b8584-367d-4cfc-8db2-259a51dba135`.
    - PASS: QA child self-claimed the QA request and received QA session id `707fa65e-207e-4ad5-b2ed-7155b1d20de7`.
    - PASS: builder wrong `resume_token` failed closed with `invalid auth request continuation`.
    - PASS: builder principal/client trying to claim the QA request failed closed with `auth request claim mismatch`.
    - PASS: QA principal/client trying to claim the builder request failed closed with `auth request claim mismatch`.
    - PASS: orchestrator/requester trying to claim the builder request failed closed with `auth request claim mismatch`.
    - TOOLING BLOCKER ONLY: the mirrored orchestrator/requester -> QA claim probe was canceled by the external tool safety layer before it reached `tillsyn`, so this session has no product-side result to record for that one redundant symmetry check.
    - FINDING: the first `create-child` role probe failed closed as `mutation lease is invalid` because the issued child sessions resolved to the existing stored principal display names `Codex Builder Agent` / `Codex QA Agent`, not the fresh request labels `Codex C4 Builder Agent` / `Codex C4 QA Agent`.
    - PASS: after reissuing project-scoped capability leases with the authenticated session names, the role-distinguishing `create-child` probe now shows the live capability policy does distinguish builder vs QA:
      - builder `till.create_task` with `kind=subtask`, `scope=subtask`, parent `380d8f50-5974-4be8-96fc-90eed6c498e9` -> PASS, created task id `46e16863-b219-4e48-818d-84e92b0e97aa`
      - QA same in-scope `till.create_task` path -> FAIL CLOSED with `invalid capability action`
    - INTERPRETATION:
      - refreshed live `C4` now proves the interim child-self-claim hardening on fresh requests,
      - builder-vs-QA policy distinction already exists on the real `create-child` seam,
      - the session-name versus request-label lease identity nuance is operational follow-up, not a blocker to auth or role-policy correctness.

## Section C5: Lease, Handoff, And Recovery Visibility

Goal:
- prove the new coordination surfaces are understandable and durable enough for real collaboration monitoring, including guarded authenticated-agent mutation.

Driver:
- joint human + orchestrator instance + current agent recording.

Steps:
1. Issue one capability lease.
2. List capability leases from CLI.
3. Heartbeat and renew that lease once.
4. Create one handoff with the live session plus the live guard tuple:
   - `agent_instance_id`
   - `lease_token`
5. List and inspect handoffs from CLI.
6. Update the handoff status.
7. Retry a guarded authenticated-agent mutation without the required guard tuple and verify it fails closed.
8. Open the TUI coordination surface and verify the same lease and handoff state is visible and understandable.
9. Run `./till project discover --project-id <project-id>` again and verify readiness/recovery output reflects the live collaboration state.
10. Revoke the lease and verify the change is reflected in CLI and TUI.
11. Verify later heartbeat or renew calls on the revoked lease fail closed.

Expected:
1. CLI and TUI tell a coherent story about leases and handoffs.
2. Names lead and ids remain secondary.
3. Guarded authenticated-agent mutation requires the live lease tuple and fails closed without it.
4. Recovery/readiness surfaces make active or hanging work visible enough for an orchestrator or human to recover safely.

Evidence:
- lease id: `codex-c5-builder-lease-20260328-a`
- handoff id: `9b96d055-2b33-407d-9de1-412bdeab2741`
- pass/fail notes:
  - PASS: issued a live builder lease over MCP using builder session id `e77b8584-367d-4cfc-8db2-259a51dba135`.
    - lease token: `7f4531e5-ce9c-40f5-9926-adb048486dd2`
    - lease `agent_name`: `Codex Builder Agent`
  - PASS: CLI `lease list`, `lease heartbeat`, and `lease renew --ttl 36h` all worked against that lease.
  - PASS: MCP `till.create_handoff` succeeded with the same session + lease tuple and created handoff id `9b96d055-2b33-407d-9de1-412bdeab2741`.
  - PASS: CLI `handoff list` and `handoff get` both showed the created handoff coherently.
  - FAIL: MCP `till.update_handoff` using the same approved builder session and same lease tuple failed closed with `auth_denied: auth denied: authorization denied`.
  - STOP ON FAIL: did not continue `C5` steps 7-11 after this live blocker.
  - Diagnosis from code inspection:
    - `till.create_handoff` authorizes with project-scoped context (`namespace = project:<project-id>`, plus `project_id` in auth context),
    - `till.update_handoff` currently authorizes with `namespace = tillsyn` and only `handoff_id` in auth context,
    - approved-path auth derives the allowed path from `project_id` or `project:<project-id>`,
    - so the update is denied before lease validation because the MCP handler does not pass enough project-scoped context.
  - Local validation around the failure remains green:
    - `just test-pkg ./internal/adapters/server/common` PASS
    - `just test-pkg ./internal/adapters/server/mcpapi` PASS
    - `just test-pkg ./internal/adapters/auth/autentauth` PASS
  - Focused local remediation result on 2026-03-28:
    - widened the fix from a handoff-only patch to one shared mutation-auth normalization seam in `AppServiceAdapter.AuthorizeMutation`,
    - kept hierarchy/path derivation in shared helpers instead of adding a handoff-only transport shim,
    - added app/common auth helpers so by-id mutations resolve project/branch/phase context before approved-path auth runs,
    - aligned explicit-scope MCP auth args for `create_task`, `create_comment`, `create_handoff`, and `issue_capability_lease`.
  - New local regression coverage:
    - shared adapter approved-path auth-context coverage in `internal/adapters/server/common/app_service_adapter_auth_context_test.go`, widened to cover additional sibling task and lease lifecycle actions through the same resolver
    - real MCP `update_handoff` approved-path regression in `internal/adapters/server/mcpapi/handler_integration_test.go`, including out-of-scope fail-closed transport coverage
    - real HTTP by-id approved-path regression in `internal/adapters/server/httpapi/handler_integration_test.go`, including out-of-scope fail-closed transport coverage
  - Focused local validation after remediation:
    - `just test-pkg ./internal/app` PASS
    - `just test-pkg ./internal/adapters/auth/autentauth` PASS
    - `just test-pkg ./internal/adapters/server/common` PASS
    - `just test-pkg ./internal/adapters/server/mcpapi` PASS
    - `just test-pkg ./internal/adapters/server/httpapi` PASS
    - `just test-pkg ./cmd/till` PASS
    - `just check` PASS
    - `just ci` PASS
  - QA/process note:
    - QA was executed as separate local review passes over the shared auth seam, transport coverage, remaining server-surface mutation-auth seams, and cross-surface noun/semantics consistency.
    - local review found no remaining project-rooted mutation seam on the current MCP/HTTP server surface after the shared resolver plus explicit-scope arg alignments.
    - intentionally non-project-rooted server auth calls that remain are projectless/global admin/operator flows such as `create_project` and kind-definition policy mutation.
    - exploratory parallel QA sweeps were also launched where available, but gate closeout for this local fix wave relies on the recorded repository evidence and live rerun remains pending.
  - FRESH LIVE CONTINUATION RESULT ON 2026-03-29:
    - PASS: created one fresh builder auth request over MCP for the Evan project:
      - request id: `ec63bfa1-7d03-4451-9fcd-694d33c65da5`
      - pre-approval child `till.claim_auth_request(wait_timeout=1s)` returned `waiting = true`
    - PASS: after TUI approval, the builder principal self-claimed the fresh request and received session id `78072889-d526-43a9-b4ab-8e1133042d42`.
    - PASS: `./till auth session validate --session-id 78072889-d526-43a9-b4ab-8e1133042d42 --session-secret <redacted>` confirmed the live builder identity:
      - principal name: `Codex Builder Agent`
      - principal id: `codex-c5-builder-20260328-b`
      - client id: `codex-c5-builder-client-20260328-b`
    - PASS: issued fresh builder lease `codex-c5-builder-lease-20260328-b` over MCP with lease token `9ed62815-a385-4ccb-9302-95ab68599790`.
    - PASS: CLI lifecycle surfaces worked against that lease:
      - `./till lease list --project-id cead38cc-3430-4ca1-8425-fbb340e5ccd9 --include-revoked`
      - `./till lease heartbeat --agent-instance-id codex-c5-builder-lease-20260328-b --lease-token 9ed62815-a385-4ccb-9302-95ab68599790`
      - `./till lease renew --agent-instance-id codex-c5-builder-lease-20260328-b --lease-token 9ed62815-a385-4ccb-9302-95ab68599790 --ttl 36h`
      - renewed `expires_at`: `2026-03-30T13:48:27.248952Z`
    - PASS: MCP `till.create_handoff` succeeded with that same approved builder session plus lease tuple and created handoff id `841492e1-5ecc-485d-86dd-13c85cc804d3`.
    - PASS: CLI `handoff list` and `handoff get` both showed that fresh handoff coherently.
    - PASS: MCP `till.update_handoff` on the same approved builder session, the same lease tuple, and handoff `841492e1-5ecc-485d-86dd-13c85cc804d3` now succeeds live.
      - result: handoff status `resolved`
      - resolution note: `Resolved during the refreshed C5 live rerun after the shared approved-path auth-context fix.`
    - PASS: the guardrail negative probe still fails closed:
      - `till.update_handoff` retried without `agent_name`, `agent_instance_id`, or `lease_token`
      - response: `invalid_request: agent_name, agent_instance_id, and lease_token are required for authenticated agent mutations`
    - PASS: `./till project discover --project-id cead38cc-3430-4ca1-8425-fbb340e5ccd9` reflected live collaboration state:
      - `active_auth_sessions = 1`
      - `active_agent_sessions = 1`
      - `active_orchestrator_sessions = 0`
      - `project_leases = 10`
      - `open_project_handoffs = 4`
      - next-step guidance pointed at requesting an orchestrator session because none was active for the project.
    - PASS: lease cleanup and post-revoke fail-closed behavior:
      - `./till lease revoke --agent-instance-id codex-c5-builder-lease-20260328-b --reason "C5 live rerun cleanup after successful handoff update"` -> PASS
      - `./till lease list --project-id cead38cc-3430-4ca1-8425-fbb340e5ccd9 --include-revoked` -> PASS; lease now shows `revoked`
      - `./till lease heartbeat --agent-instance-id codex-c5-builder-lease-20260328-b --lease-token 9ed62815-a385-4ccb-9302-95ab68599790` -> FAIL CLOSED with `mutation lease is revoked`
      - `./till lease renew --agent-instance-id codex-c5-builder-lease-20260328-b --lease-token 9ed62815-a385-4ccb-9302-95ab68599790 --ttl 1h` -> FAIL CLOSED with `mutation lease is revoked`
    - USER-CONFIRMATION PENDING:
      - the user was asked to confirm that the TUI coordination surface showed active then revoked lease state for `codex-c5-builder-lease-20260328-b` and resolved handoff state for `841492e1-5ecc-485d-86dd-13c85cc804d3`,
      - but explicit fresh-pass confirmation was not captured before the run continued.
  - LIVE TUI FOLLOW-UP ON 2026-03-28:
    - FAIL: during the requested TUI confirmation, the user reported that the lower coordination content went below the page and would not move into view while scrolling.
    - local root cause: the coordination/auth-inventory screen was still rendered as one static full-page string body instead of using a viewport-backed full-page surface.
    - landed local fix:
      - added a dedicated `authInventoryBody` viewport,
      - synced it on window resize, inventory load, scope reopen, keyboard movement, and mouse-wheel movement,
      - switched the coordination renderer to `renderFullPageSurfaceViewport(...)`,
      - added one short-terminal regression that proves mouse-wheel scrolling can reach the lower lease and handoff sections.
    - focused evidence:
      - Context7 `/charmbracelet/bubbles` viewport docs rechecked before the fix and again after each failed TUI test iteration while tightening the regression.
      - `just test-pkg ./internal/tui` -> FAIL
      - `just test-pkg ./internal/tui` -> FAIL
      - `just test-pkg ./internal/tui` -> FAIL
      - `just test-pkg ./internal/tui` -> FAIL
      - `just test-pkg ./internal/tui` -> FAIL
      - `just test-pkg ./internal/tui` -> PASS
      - `just check` -> PASS
      - `just ci` -> PASS
    - next live step:
      - reopen the TUI `Coordination` screen and confirm the lower `capability leases` and `handoffs` sections are now reachable by scrolling.
  - SECOND LIVE USABILITY FOLLOW-UP ON 2026-03-28:
    - PARTIAL LIVE CONFIRMATION:
      - after restarting on the fresh binary, the user reached the lower `active sessions` and `capability leases` sections in the `Coordination` screen, which confirmed the original clipping bug was fixed live.
    - NEW LIVE UX GAP:
      - the user reported that the coordination screen was still confusing because it mixed live and historical rows in one long list,
      - and the current live coordination state was still hidden behind the command-palette screen instead of also surfacing in the board notifications panel.
    - landed local follow-up:
      - added one compact inline `Live Coordination` summary row to the project notifications panel so pending requests, active sessions, active leases, and open handoffs are visible from the board,
      - removed the legacy `Selection` notices section so the project panel stays focused on warnings, action-required rows, recent activity, and live coordination instead of duplicating the selected task card,
      - split the full-screen `Coordination` surface into `live` and `history` slices with `h` toggle behavior,
      - tightened the coordination viewport alignment logic to use wrapped line offsets so mouse-wheel and keyboard movement keep the selected lease/handoff section visible even when long detail rows soft-wrap,
      - moved the detailed coordination key guidance fully into the bottom help bar plus the expanded `?` overlay and removed the duplicated inline hint block from the coordination body,
      - refreshed the TUI goldens for the intentional notices-panel summary change.
    - focused evidence:
      - Context7 `/charmbracelet/bubbles` viewport docs rechecked before the first edit and again after each failed TUI/check loop in this follow-up.
      - `just test-pkg ./internal/tui` -> FAIL
      - `just test-golden-update` -> PASS
      - `just test-pkg ./internal/tui` -> PASS
      - `just fmt` -> PASS
      - `just check` -> PASS
      - `just ci` -> PASS
    - files changed:
      - `internal/tui/model.go`
      - `internal/tui/model_test.go`
      - `internal/tui/testdata/TestModelGoldenBoardOutput.golden`
      - `internal/tui/testdata/TestModelGoldenHelpExpandedOutput.golden`
    - next live step:
      - reopen the fresh TUI and confirm:
        - the board shows the compact `Live Coordination` summary row,
        - `Coordination` defaults to live/actionable rows,
        - `h` cleanly toggles to history,
        - and the lower handoff rows remain reachable after the wrapped-line viewport fix.
  - FRESH LIVE REOPEN ON 2026-03-29 AFTER COMMIT `d1dbb44`:
    - PASS:
      - `?` detailed help opened correctly on `Coordination`,
      - `h` toggled live/history correctly,
      - `esc` returned to the board,
      - the legacy `Selection` notices section is gone.
    - FAIL:
      - the board/global notifications panels still do not surface actionable auth/coordination rows; they only show the compact summary plus existing warnings/activity content,
      - and `enter` on coordination leases/handoffs still does not open a dedicated detail/info surface.
    - active next fix scope:
      - add selectable coordination rows to the project notifications panel with count-by-type labels,
      - add project-grouped coordination rows to the global notifications panel,
      - route `enter` on those notification rows into the related project coordination screen,
      - and make `enter` on coordination rows open a real detail/info surface.
  - LOCAL FOLLOW-UP FIX ON 2026-03-29 AFTER THE FRESH REOPEN:
    - implementation outcome:
      - the project notifications panel now keeps one compact actionable `Live Coordination` summary row instead of a tall four-row slice, so warnings/action-required/activity remain visible,
      - the global notifications panel now includes per-project coordination summary rows that open the related project coordination screen on `enter`,
      - `enter` on coordination sessions/leases/handoffs now opens a centered detail/info modal and `esc` returns to coordination,
      - and the unused per-kind deep-link experiment was removed so the notifications/coordination path stays compact and DRY.
    - validation evidence:
      - `just test-pkg ./internal/tui` -> FAIL initially only on the expected pre-update goldens after the layout change
      - `just test-golden-update` -> PASS
      - `just test-pkg ./internal/tui` -> PASS
      - `just fmt` -> PASS
      - `just check` -> PASS
      - `just ci` -> PASS
    - files changed:
      - `internal/tui/model.go`
      - `internal/tui/model_test.go`
      - `internal/tui/testdata/TestModelGoldenBoardOutput.golden`
      - `internal/tui/testdata/TestModelGoldenHelpExpandedOutput.golden`
    - next live confirmation:
      - restart `./till` and confirm:
        - the project notifications panel shows one selectable `Live Coordination` summary row,
        - the global notifications panel shows coordination rows grouped by project,
        - `enter` on either notification row opens the appropriate coordination screen,
        - and `enter` on a coordination session/lease/handoff row opens item detail instead of only updating inline status text.

## Section C6: Display-Name Clarity And Readiness Bridge

Goal:
- verify that the collaboration-critical surfaces are understandable without forcing humans to decode ids or remember hidden next steps.

Driver:
- human with agent observing and recording.

Steps:
1. Revisit:
   - `./till project discover --project-id <project-id>`
   - TUI coordination
   - TUI notifications
   - any CLI list surfaces used during the run.
2. Look specifically for:
   - id-primary labels,
   - ambiguous same-name rows,
   - unclear role labels,
   - unclear next-step guidance.

Expected:
1. The product itself points the operator toward the next action.
2. Remaining clarity issues, if any, are small enough to log as follow-up rather than blocking dogfood use.

Evidence:
- pass/fail notes:

## Section C7: Final Verdict

Goal:
- decide whether the current auth and collaboration path is dogfood-ready enough to keep using for the next loop.

Driver:
- joint human + agent.

Checklist:
1. Runtime and discovery preflight passed.
2. TUI auth review and coordination spot-check passed.
3. Fresh orchestrator MCP request and claim/resume passed.
4. Authenticated mutation passed.
5. Revoke failed closed.
6. Builder and QA delegation passed.
7. Anti-adoption protection passed.
8. In-scope versus out-of-scope mutation behavior passed.
9. Lease and handoff lifecycle visibility passed.
10. Readiness/recovery visibility passed.
11. Name-first clarity is acceptable for current dogfood use.

Verdict:
- overall pass/fail:
- blockers:
- follow-up items:
