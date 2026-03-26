# Collaborative E2E Auth And MCP Worksheet

Created: 2026-03-25
Updated: 2026-03-26
Status: Temporarily paused pending the replacement GitHub Actions run after the Windows SQLite raw-path remediation plus QA-driven regression hardening; all local gates are green and the live worksheet should resume once that replacement remote CI run is green again.

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
   - do not resume the live worksheet until the replacement GitHub Actions run for the current raw-path plus QA-hardening follow-up is green.

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
- orchestrator display name:
- request id:
- approved path:
- denied request id:
- canceled request id:
- pass/fail notes:

Status note before continuing:
- `C2` should still prove current fail-closed auth, TUI visibility, and native claim/resume behavior.
- `C2` can now prove the auth-specific local cross-process wake path for the default human-in-the-loop dogfood flow.
- `C2` should still not be used to claim that the broader session-aware stdio communication layer or comment/handoff consumers already exist.
- `C2` should still treat broader MCP notification reuse, disconnect-aware session cleanup, and HTTP/continuous-listening support as follow-on work.
- Automated evidence before the next live rerun:
  - `just test-pkg ./internal/app` PASS
  - `just check` PASS
  - `just ci` PASS

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
- mutation used:
- out-of-scope mutation used:
- revoke surface used:
- pass/fail notes:

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
- builder display name:
- QA display name:
- anti-adoption attempt:
- pass/fail notes:

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
- lease id:
- handoff id:
- pass/fail notes:

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
