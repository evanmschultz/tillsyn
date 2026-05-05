# Drop 4b — Wave C Plan (Auth Auto-Revoke + Git-Status Pre-Check + Auto-Promotion + Hylla Reingest Gate)

**Mode:** filesystem-MD only. No Tillsyn plan items, no per-droplet Tillsyn auth.
**Source brief:** `workflow/drop_4b/REVISION_BRIEF.md` (locked 2026-05-04, decisions L1–L7, scope Option β).
**Sketch:** `workflow/drop_4b/SKETCH.md` Wave C (3 droplets).
**Hard prereqs (all on `main`):** Drop 4a closed at commit `618c7d2`; specifically 4a.11 (always-on parent-blocks-on-failed-child), 4a.12 (project first-class fields), 4a.15 (`LiveWaitEventActionItemChanged`), 4a.18 (walker), 4a.19 (spawn stub), 4a.22 (cleanup hook with auth-revoke stub), 4a.23 (`RunOnce` 8-stage pipeline + dispatcher CLI), 4a.24–4a.28 (orch-self-approval auth surface).

Scope: 3 droplets — `4b.5` auth auto-revoke, `4b.6` git-status pre-check, `4b.7` auto-promotion subscriber + `hylla_reingest` gate stub.

---

## 1. Wave C Overview

### 1.1 Droplet Set

| Droplet | Title | Scope-summary | LOC est. |
| ------- | ----- | -------------- | -------- |
| 4b.5    | AUTH AUTO-REVOKE WIRING | Replace `revokeAuthBundleStub` with real session lookup + revoke. | ~120 (60 prod / 60 test) |
| 4b.6    | GIT-STATUS PRE-CHECK ON `Service.CreateActionItem` | Domain-level guard: reject creation when declared `paths` are dirty. | ~150 (75 prod / 75 test) |
| 4b.7    | AUTO-PROMOTION SUBSCRIBER + `hylla_reingest` GATE STUB | Wire dispatcher's `Start`/`Stop` against `LiveWaitEventActionItemChanged`; ship `hylla_reingest` gate stub. | ~220 (130 prod / 90 test) |

Total: ~490 LOC across the wave.

### 1.2 Cross-Wave Sequencing

- **Wave A → C:** independent. Wave A ships the gate runner framework + `mage_ci` / `mage_test_pkg` gates in `internal/app/dispatcher/gates.go` (NEW). Wave C's `4b.7` adds a `hylla_reingest` gate that registers atop the same framework — `4b.7` IS `blocked_by` Wave A's gate-runner droplet (likely `4b.2`) because the `gateFunc` registry signature is owned by Wave A.
- **Wave A → C cross-droplet edges:**
  - `4b.7` blocked_by `4b.2` (gate-runner registry) — `hylla_reingest` registers as a `gateFunc`.
  - `4b.5` and `4b.6` are independent of Wave A entirely.
- **Within Wave C:** all three droplets are file/package-disjoint and can run in parallel from a same-wave perspective. Concrete disjointness analysis in §5 below.

### 1.3 LOC + Surface Budget

- **New files:** `internal/app/dispatcher/auth_revoke.go` + test (4b.5); `internal/domain/git_status.go` + test (4b.6) — domain-light helper for the porcelain check; `internal/app/dispatcher/continuous.go` + test, `internal/app/dispatcher/hylla_reingest.go` + test (4b.7).
- **Edited files:** `internal/app/dispatcher/cleanup.go` (4b.5: replace `revokeAuthBundleStub` wiring + `newCleanupHook` signature widening); `internal/app/service.go` (4b.6: extend `CreateActionItem` with pre-check call); `internal/app/dispatcher/dispatcher.go` (4b.7: wire `Start`/`Stop` real implementations + accept gate registry); `cmd/till/main.go` (4b.7: spin up the dispatcher subscriber alongside `runServe`).

---

## 2. Droplet 4b.5 — AUTH AUTO-REVOKE WIRING

### 2.1 Goal

Replace `revokeAuthBundleStub` (cleanup.go:253-256) with a real revoke that:

1. Resolves the action item's session ID from the `auth_sessions` index keyed by approved-path scope.
2. Calls `Service.RevokeAuthSession(ctx, sessionID, reason)` (auth_requests.go:860).
3. Aggregates errors via `errors.Join` so a revoke failure never blocks lock release.
4. No-ops cleanly when the action item has no live session (never-claimed / already-revoked path).

### 2.2 Paths

- `internal/app/dispatcher/cleanup.go` (EDIT — replace `revokeAuthBundleStub` references; widen `newCleanupHook` signature to accept the auth-revoke seam).
- `internal/app/dispatcher/cleanup_test.go` (EDIT — extend tests with revoke success, revoke-error-aggregation, missing-session no-op).
- `internal/app/dispatcher/auth_revoke.go` (NEW — `revokeAuthBundleForActionItem` helper + the `actionItemAuthRevoker` interface that abstracts `*app.Service` for test injection).
- `internal/app/dispatcher/auth_revoke_test.go` (NEW — table-driven tests exercising the helper directly with a stub revoker).
- `internal/app/dispatcher/dispatcher.go` (EDIT — `NewDispatcher` constructs the new `actionItemAuthRevoker` adapter from `*app.Service` and threads it into `newCleanupHook`).

### 2.3 Packages

- `internal/app/dispatcher`

### 2.4 Acceptance Criteria

1. `revokeAuthBundleStub` (cleanup.go:253-256) is **deleted** — the function no longer exists in the package.
2. `newCleanupHook` (cleanup.go:135) signature widens to accept an `actionItemAuthRevoker` (or a `func(ctx, actionItemID) error` seam — choose the function-typed shape per existing cleanup.go convention at line 92–105).
3. **Production wiring:** `dispatcher.go:258` `newCleanupHook` call passes a non-stub revoker that delegates to `Service.RevokeAuthSession` after resolving the session.
4. **Session resolution mechanism (the load-bearing decision):** introduce a new `Service.RevokeSessionForActionItem(ctx context.Context, actionItemID, reason string) error` method in `internal/app/auth_requests.go`. The method:
   - Calls `s.authBackend.ListAuthSessions(ctx, AuthSessionFilter{...})` filtered by parsing `session.ApprovedPath` and matching `path.ScopeID == actionItemID`.
   - Picks the first non-revoked session (filter on `session.RevokedAt == nil`).
   - Calls `s.RevokeAuthSession(ctx, session.SessionID, reason)`.
   - Returns `nil` (no error) if zero matching sessions exist (the "never-claimed" path is a clean no-op).
5. **Reason string:** the revoke reason MUST be machine-parseable: `"dispatcher cleanup: terminal state " + string(item.LifecycleState)` (e.g. `"dispatcher cleanup: terminal state complete"`).
6. **Error handling:** `revokeAuthBundleForActionItem` errors propagate up `cleanupHook.OnTerminalState` via `errors.Join` — they do NOT short-circuit lock release or monitor unsubscribe. Existing aggregation logic at cleanup.go:218-237 is unchanged.
7. **Idempotency preservation:** the cleanup hook's existing `cleaned` map (cleanup.go:115) prevents double-revoke. Second call with same `item.ID` short-circuits before the revoke seam fires (cleanup.go:211-216 path unchanged).
8. **Lease revocation:** `Service.RevokeAuthSession` already revokes the underlying session bundle. Per autent's `RevokeAuthSession` contract, revoking the session implicitly invalidates the associated lease (no separate revoke call needed). **Verify in implementation:** if the autent backend keeps lease + session as separate rows, an explicit `s.authBackend.RevokeLease(...)` call lands in this droplet too. (Builder must inspect `internal/adapters/auth/autentauth/service.go` `RevokeAuthSession` body and confirm. Default assumption: session revoke is sufficient because lease validation goes through session lookup.)
9. **Doc-comment update:** `cleanup.go` package doc-comment lines 21–28 are revised — drop "Drop 4c Theme F.7 fills this in" and replace with "Drop 4b.5 wired this in."

### 2.5 Test Scenarios

Named tests in `cleanup_test.go` and `auth_revoke_test.go`:

1. `TestCleanupHookRevokesAuthSessionOnComplete` — happy path: action item terminal-states `complete`, mock revoker observes the call with the right action-item ID + reason string.
2. `TestCleanupHookRevokesAuthSessionOnFailed` — same as above but `failed` state. Reason string contains `"failed"`.
3. `TestCleanupHookRevokesAuthSessionOnArchived` — same as above but `archived` state. Reason string contains `"archived"`.
4. `TestCleanupHookSkipsRevokeWhenNoSession` — revoker returns `nil` (no session found); cleanup completes without error.
5. `TestCleanupHookAggregatesRevokeError` — revoker returns an error; `OnTerminalState` returns a non-nil error joined with any other step errors. Lock releases STILL fired (assert via mock release counters).
6. `TestCleanupHookIdempotentOnRepeatedTerminal` — second `OnTerminalState` call with same `item.ID` does NOT re-fire the revoke seam.
7. `TestRevokeSessionForActionItemMatchesScopeIDPath` (auth_revoke_test.go) — service helper finds the session whose `ApprovedPath` parses to a `ScopeID == actionItemID`.
8. `TestRevokeSessionForActionItemSkipsRevokedSessions` — a session with `RevokedAt != nil` is skipped; an active sibling session is the one revoked.
9. `TestRevokeSessionForActionItemReturnsNilWhenNoMatch` — zero matching sessions; returns nil (clean no-op).

### 2.6 Falsification Attacks + Mitigations

- **A1: Race on cleanup → re-acquisition.** If revoke runs after lock release, a fresh dispatcher tick could re-spawn against the now-failed item before revoke completes. **Mitigation:** the cleanup hook's existing pipeline order (cleanup.go:218-237) is file-lock → package-lock → auth-revoke → monitor-unsub. Lock release happens BEFORE revoke. Acceptable because (a) the action item is in a terminal state, so the walker's `isEligible` predicate (walker.go:167-179) rejects it on the first check (`item.LifecycleState != domain.StateTodo` → false); (b) auth-bundle isn't load-bearing at re-spawn time — only the action item's lifecycle state is. Document the ordering in `cleanup.go` package doc-comment.
- **A2: Silent revoke when session already revoked externally.** Manual `till auth_request revoke` or autent's TTL expiry already revoked the session; the helper finds it but the revoke errors with "session already revoked." **Mitigation:** `Service.RevokeSessionForActionItem` filters `session.RevokedAt != nil` BEFORE calling the backend, so the already-revoked path is a clean no-op (test scenario 8).
- **A3: Multiple sessions per action item.** Test scenario or future feature could leave multiple sessions tied to the same `ScopeID`. **Mitigation:** the helper revokes the FIRST active session it finds. Document this in the helper doc-comment as "best-effort revoke; scope-id collisions are a planner-side bug." Drop 4c F.7 may revisit when temp-bundle architecture lands.
- **A4: ActionItem-without-session (orchestrator-driven creation).** Persistent / human-verify items never claim auth; cleanup must not error. **Mitigation:** test scenario 4. The helper's "no match → nil error" path is the load-bearing answer.
- **A5: Auth backend nil in the test fixture.** Older test suites construct `*app.Service` without `authBackend`. **Mitigation:** the helper's `if s.authBackend == nil { return nil }` short-circuit (mirroring auth_requests.go:861-863's pattern) keeps backward compatibility.
- **A6: Session resolution path parsing failure.** `session.ApprovedPath` is a malformed string. **Mitigation:** the helper uses `domain.ParseAuthRequestPath` (auth_requests.go:1004 precedent); on parse error, the session is skipped (not the revoke aborted).

### 2.7 DB Action

**None.** Schema unchanged; only behavior changes.

### 2.8 Blocked By

**Cross-wave:** none. Wave C 4b.5 has no Wave A dependencies (cleanup hook is independent of the gate runner).

**Within Wave C:** none.

**Hard prereqs (Drop 4a):** 4a.22 (cleanup hook stub), 4a.24 (auth-role enum + approve flow), 4a.26 (audit-trail surface). All on `main`.

### 2.9 Verification Gate

- `mage test-pkg internal/app/dispatcher` — all package tests pass; coverage on cleanup.go + auth_revoke.go ≥80%.
- `mage test-pkg internal/app` — `Service.RevokeSessionForActionItem` test passes; no regression on existing auth_requests tests.
- `mage ci` — full clean.

---

## 3. Droplet 4b.6 — GIT-STATUS PRE-CHECK ON `Service.CreateActionItem`

### 3.1 Goal

When `CreateActionItemInput.Paths` is non-empty, run `git status --porcelain <path>` per declared path against `project.RepoPrimaryWorktree`. Reject creation if any path is dirty; return a domain-typed error listing the dirty paths.

### 3.2 Paths

- `internal/domain/git_status.go` (NEW — `GitStatusChecker` interface + concrete implementation; lives in `internal/domain` so it's reusable by future builders without a new adapter package).
- `internal/domain/git_status_test.go` (NEW — table-driven tests using a stub `gitStatusRunner` injected via constructor).
- `internal/domain/errors.go` (EDIT — add `ErrPathsDirty` sentinel).
- `internal/app/service.go` (EDIT — `CreateActionItem` calls the checker before `s.repo.CreateActionItem` at line ~907; constructor wires a default checker).
- `internal/app/service_test.go` (EDIT — extend existing `CreateActionItem` test cases with one dirty-path rejection scenario + one clean-path success scenario).
- `internal/app/ports.go` (EDIT, conditional — if `gitStatusChecker` becomes a port). **Author judgment:** keep it as a struct-field on `*app.Service` set via a new field on the existing constructor's option struct (`ServiceOptions` if it exists; otherwise add a new optional WithX-pattern setter). NOT a port; the checker is in-process and trivial.

### 3.3 Packages

- `internal/domain`
- `internal/app`

### 3.4 Acceptance Criteria

1. **New domain helper** `internal/domain/git_status.go`:
   - `GitStatusChecker` interface: `CheckPathsClean(ctx context.Context, repoRoot string, paths []string) error`.
   - Concrete `osGitStatusChecker` (or equivalent name) backed by `exec.Command("git", "status", "--porcelain", path)`.
   - **Per-path invocation, not batched.** Per REVISION_BRIEF Q6 + sketch L6: path count per droplet is typically <10; per-path is simple + correct; batching adds parsing cost.
   - Returns `ErrPathsDirty` (defined in `internal/domain/errors.go`) wrapped with the dirty path list when any path is dirty. Error format: `domain.ErrPathsDirty: ["path1", "path2"]` (or equivalent — must include all dirty paths).
   - Returns `nil` for empty `paths` slice (degenerate input).
   - Returns `nil` for empty `repoRoot` (degenerate path — see A5 below).
2. **`Service.CreateActionItem` integration** (service.go:813-907):
   - Insert pre-check call AFTER parent lineage validation (line 841) and BEFORE `domain.NewActionItem` (line 907).
   - Pre-check fires **only when `len(in.Paths) > 0`**. Empty `Paths` skips the check (degenerate-input rule).
   - Pre-check loads the project via `s.repo.GetProject(ctx, in.ProjectID)` (or reuses `s.GetProject` if it exists) to read `RepoPrimaryWorktree`.
   - **Project not found / no `RepoPrimaryWorktree`:** the pre-check is **skipped** (NOT a hard reject). Rationale: a project without a worktree configured is a Drop-4a-pre-Drop-4b legacy or test fixture; rejecting would brick existing test suites. Document the skip explicitly in the helper doc-comment.
   - Pre-check failure returns `fmt.Errorf("git status pre-check failed: %w", err)` (caller detects via `errors.Is(err, domain.ErrPathsDirty)`).
3. **Error sentinel:** `domain.ErrPathsDirty = errors.New("declared paths have uncommitted changes")` in `internal/domain/errors.go`.
4. **Service constructor:** the checker is wired via the existing `*app.Service` construction path. **Choose the lowest-disruption surface** — likely a new field `gitStatus GitStatusChecker` on `*app.Service` with a default `osGitStatusChecker` populated by the production constructor. Test code injects a stub via direct struct-field assignment in the same package.
5. **Always-on:** no project-metadata toggle for this check today (per REVISION_BRIEF L4: "Always-on; bypass requires the post-MVP supersede CLI"). Builders MUST clean the working tree before declaring `paths`.
6. **Pre-MVP rule:** dev fresh-DBs after this lands. **No migration code in Go.**

### 3.5 Test Scenarios

Named tests:

1. `TestCreateActionItemRejectsDirtyDeclaredPaths` (service_test.go) — input with `Paths=["internal/foo/bar.go"]`, stub checker returns `ErrPathsDirty`; `CreateActionItem` returns wrapped `ErrPathsDirty`; no row written.
2. `TestCreateActionItemAllowsCleanDeclaredPaths` (service_test.go) — input with `Paths=["internal/foo/bar.go"]`, stub checker returns `nil`; `CreateActionItem` succeeds.
3. `TestCreateActionItemSkipsCheckOnEmptyPaths` (service_test.go) — input with `Paths=[]`; checker is NOT invoked; creation succeeds.
4. `TestCreateActionItemSkipsCheckOnMissingWorktree` (service_test.go) — project has empty `RepoPrimaryWorktree`; checker is NOT invoked; creation succeeds.
5. `TestOsGitStatusCheckerDetectsDirtyPath` (git_status_test.go) — integration-flavored: temp git repo, dirty file in tree, checker returns `ErrPathsDirty` listing the path.
6. `TestOsGitStatusCheckerCleanPathReturnsNil` (git_status_test.go) — temp git repo, clean tree; checker returns `nil`.
7. `TestOsGitStatusCheckerHandlesNonexistentPath` (git_status_test.go) — path that doesn't exist in the worktree; `git status --porcelain` returns empty (untracked-but-also-nonexistent) — checker returns `nil` (path is not dirty because it's not a tracked file). **Document this as the existing-but-degenerate behavior; future refinement may add path-existence validation.**
8. `TestOsGitStatusCheckerHandlesGitBinaryMissing` (git_status_test.go, may be t.Skip on CI) — `exec.Command("git", ...)` returns `exec.ErrNotFound`; checker returns a wrapped error so the dev sees a useful message rather than a panic.
9. `TestOsGitStatusCheckerWalksMultiplePaths` (git_status_test.go) — input with two paths: one dirty, one clean. Returned error lists ONLY the dirty one.

### 3.6 Falsification Attacks + Mitigations

- **A1: Path declared in `Paths` but actual builder edit is OUTSIDE `Paths`.** Pre-check passes; builder dirties an undeclared file. **Mitigation:** out of scope for 4b.6 — this is a runtime-enforcement problem (the post-build gate framework Wave A is where it would land if/when added). Document as Unknown for Drop 4c / Drop 5 dogfood.
- **A2: Symbolic link / canonical-path mismatch.** Declared path is `./foo/bar.go`; git operates on `internal/foo/bar.go`. **Mitigation:** the checker passes the path verbatim to `git status --porcelain <path>`; git itself normalizes within the worktree. **Builder must declare paths with `RepoPrimaryWorktree`-relative path semantics.** Document in helper doc-comment.
- **A3: Git-binary missing on PATH.** Test scenario 8. **Mitigation:** wrap `exec.ErrNotFound` with a clear message: `fmt.Errorf("git status pre-check unavailable: git binary not found on PATH: %w", err)`. Caller can choose to treat as fatal or log+skip; default behavior is fatal.
- **A4: Worktree on wrong commit.** Builder is on `main`, but the action item targets a feature branch. Git status returns clean against `main` head; builder's "clean" state is misleading. **Mitigation:** out of scope. The action item's `ProjectID → RepoPrimaryWorktree` is the single source of truth; cross-branch coordination is a planner concern, not a CreateActionItem concern. Document as Unknown.
- **A5: Empty `RepoPrimaryWorktree` on a real project.** Acceptance criterion 2 says skip. **Mitigation:** pre-MVP escape valve. Drop 5 dogfood will validate that real projects always have `RepoPrimaryWorktree` populated. Add a worklog note on every skipped check (`logger.Warn("git status pre-check skipped: project has empty repo_primary_worktree", "project_id", in.ProjectID)`).
- **A6: Concurrent modifications between pre-check and `repo.CreateActionItem`.** Builder's git tree is clean at check-time, dirty 50ms later when another process writes. **Mitigation:** unavoidable race; acceptable. The check is a soft guard, not a hard contract — the test suite is the load-bearing assertion.
- **A7: Path traversal / escape (`../../../etc/passwd`).** `git status --porcelain ../../../etc/passwd` on `RepoPrimaryWorktree` — git rejects paths outside the worktree with an error. **Mitigation:** wrap git's error verbatim; the dev sees the clear error message.
- **A8: Massive path list.** Builder declares 50 paths; per-path invocation = 50 git executions. **Mitigation:** acceptable per REVISION_BRIEF Q6 — typical droplet has <10 paths. Document the per-path-cost trade-off in the helper doc-comment. Future refinement: batch via `git status --porcelain --pathspec-from-file -` if profiling shows hot path.

### 3.7 DB Action

**Dev fresh-DBs `~/.tillsyn/tillsyn.db` BEFORE `mage ci`.** No schema change, but existing in-progress action items may have been created with dirty paths and would fail re-validation if a re-creation flow ran them. Pre-MVP rule per REVISION_BRIEF §5.

### 3.8 Blocked By

**Cross-wave:** none.

**Within Wave C:** none — file/package-disjoint from 4b.5 and 4b.7.

**Hard prereqs (Drop 4a):** 4a.5 (`paths []string` on `ActionItem`), 4a.12 (project `RepoPrimaryWorktree`). Both on `main`.

### 3.9 Verification Gate

- `mage test-pkg internal/domain` — git_status_test.go passes; `mage ci`'s 70% line-coverage threshold met.
- `mage test-pkg internal/app` — service_test.go passes; new test scenarios cover happy + reject + skip paths.
- `mage ci` — full clean.

---

## 4. Droplet 4b.7 — AUTO-PROMOTION SUBSCRIBER + `hylla_reingest` GATE STUB

### 4.1 Goal

Wire the dispatcher's continuous-mode loop:

- **Subscriber:** dispatcher's `Start(ctx)` (dispatcher.go:814 — currently returns `ErrNotImplemented`) becomes a real subscriber that calls `subscribeBroker` (broker_sub.go:47) to listen for `LiveWaitEventActionItemChanged` events. On every event, walks the project tree via `walker.EligibleForPromotion` (walker.go:130) and calls `RunOnce` for each eligible item.
- **Lifecycle:** `Stop(ctx)` cancels the subscriber goroutine and waits for it to exit. The subscriber goroutine has a defer-recover for panic safety.
- **`till serve` integration:** `runServe` (main.go:2583) constructs the `Dispatcher`, calls `Start(ctx)` alongside the HTTP/MCP servers, and `Stop(ctx)` on shutdown.
- **`hylla_reingest` gate stub:** registers as a `gateFunc` in the gate runner's registry (Wave A `4b.2`). The stub's body: log a structured warning ("Hylla MCP not connected; skipping reingest"), return nil. **No real Hylla MCP call in 4b.** Drop 5 dogfood evaluates whether the gate ever fires real ingests.

### 4.2 Paths

- `internal/app/dispatcher/continuous.go` (NEW — `dispatcher.Start` + `Stop` real implementations; subscriber goroutine; debounce + per-event walk).
- `internal/app/dispatcher/continuous_test.go` (NEW — table-driven tests with stub broker and stub `RunOnce`).
- `internal/app/dispatcher/dispatcher.go` (EDIT — replace `Start`/`Stop` stubs at lines 814 + 820; thread the gate registry through `Options`).
- `internal/app/dispatcher/hylla_reingest.go` (NEW — `hyllaReingestGate` `gateFunc`; logs warning + returns nil).
- `internal/app/dispatcher/hylla_reingest_test.go` (NEW — verifies the stub's contract: returns nil, logs the expected warning).
- `cmd/till/main.go` (EDIT — `runServe` at line 2583 spawns the dispatcher via `Start(ctx)`; shutdown path calls `Stop`).
- `cmd/till/main_test.go` (EDIT — verify the dispatcher subscriber starts + stops cleanly during `runServe` tests; existing tests should not regress).

### 4.3 Packages

- `internal/app/dispatcher`
- `cmd/till`

### 4.4 Acceptance Criteria

1. **`dispatcher.Start(ctx)` (dispatcher.go:814):**
   - Replaces `return ErrNotImplemented` with a real subscriber.
   - Spawns a goroutine via `go d.runContinuousLoop(ctx)`.
   - Stores a cancellable child context on the dispatcher struct so `Stop` can terminate cleanly.
   - Returns `nil` immediately (non-blocking start).
   - Idempotent: a second `Start` call without an intervening `Stop` returns `ErrAlreadyStarted` (new sentinel) rather than spawning two goroutines.
2. **`dispatcher.Stop(ctx)` (dispatcher.go:820):**
   - Cancels the subscriber's context.
   - Waits up to a bounded deadline (use `ctx`'s deadline, default 5s if `ctx` has none) for the goroutine to exit.
   - Returns `nil` on clean exit, `ctx.Err()` on deadline exceeded.
   - Idempotent: a second `Stop` call after the first returns `nil` immediately.
3. **`runContinuousLoop` (continuous.go, NEW):**
   - Calls `d.subscribeBroker(ctx, projectID)` for each project the dispatcher tracks. **Open design Q4.4.1 below — single-project vs multi-project subscription.**
   - Deferred `recover()` + `logger.Error` on panic; the loop restarts itself once after a panic (1 retry total) — defensive against transient bugs without infinite-restart-loop hazards.
   - Walks the project tree on every received event via `d.walker.EligibleForPromotion(ctx, projectID)`.
   - For each eligible item: calls `d.RunOnce(ctx, item.ID, projectID)`. Errors are logged + ignored (the loop keeps draining the broker). Skipped/blocked outcomes are logged at Info; spawned outcomes at Info.
4. **Debounce:** consecutive events arriving within a 100ms window are coalesced into a single tree-walk. Implementation: a 100ms timer reset on each event; tree walk fires on timer expiry. **Rationale:** REVISION_BRIEF / SKETCH attack A7 — burst-events would otherwise trigger O(N) tree walks per second.
5. **`Options` struct extension:** `internal/app/dispatcher/dispatcher.go:79-84` (the `Options` struct):
   - Add field `GateRegistry map[string]gateFunc` (or whatever Wave A's registry type is named — coordinate at builder time).
   - Add field `ProjectIDs []string` for the subscriber's scope. Empty means "all projects" (loaded via `s.repo.ListProjects`).
6. **`hyllaReingestGate` (hylla_reingest.go, NEW):**
   - Signature matches Wave A's `gateFunc` (TBD; pseudocode: `func(ctx context.Context, item domain.ActionItem, project domain.Project) error`).
   - Body: `logger.Warn("hylla_reingest gate: Hylla MCP not connected in dispatcher process; skipping reingest", "action_item_id", item.ID, "project_id", project.ID); return nil`.
   - Doc-comment: "Drop 4b stub — Drop 5 dogfood evaluates real Hylla MCP wiring. Memory rule `feedback_orchestrator_runs_ingest.md` keeps the manual reingest contract authoritative."
   - Registers in the gate framework's registry under name `"hylla_reingest"` (per L2 closed enum).
7. **`runServe` integration (cmd/till/main.go:2583):**
   - After `runServe` constructs `*app.Service`, construct `*app.LiveWaitBroker` (or use the existing one if `runFlow` already has one).
   - Construct the dispatcher: `disp, err := dispatcher.NewDispatcher(svc, broker, dispatcher.Options{ProjectIDs: ..., GateRegistry: ...})`.
   - Call `disp.Start(ctx)`.
   - Defer `disp.Stop(...)` with a fresh 5s timeout context for graceful shutdown.
   - **Don't break existing `runServe` flow** — HTTP server + MCP server still launch; the dispatcher runs alongside them.
8. **Sentinel errors:**
   - `ErrAlreadyStarted = errors.New("dispatcher: already started")` in dispatcher.go.
   - **Delete or repurpose** `ErrNotImplemented` (dispatcher.go:117) once `Start`/`Stop` are wired. **Author choice:** keep the sentinel as a deprecated alias for backward compat with existing test fixtures that check for it; mark with `// Deprecated: Drop 4b wired Start/Stop.` Builder verifies no tests break by deleting it; if tests rely on the sentinel, keep + deprecate.

### 4.5 Open Design Question (Q4.4.1)

**Single-project vs multi-project subscription.** Three options:

- **Option A:** Subscriber subscribes to ONE project (passed via `Options.ProjectIDs`, with len-1 enforced). Multi-project requires the dev to launch one `till serve` per project. Simplest.
- **Option B:** Subscriber subscribes to ALL projects (`s.repo.ListProjects` at startup, one `subscribeBroker` goroutine per project). New projects added during runtime are NOT picked up (require `till serve` restart).
- **Option C:** Subscriber subscribes to ALL projects + listens for project-creation events (a future `LiveWaitEventProjectCreated`). Most ergonomic; most complex.

**Author recommendation: Option B.** The single MVP dogfood (Drop 5) needs multi-project support without per-project process management. Project-creation-during-runtime is rare enough to defer. Builder MUST `s.repo.ListProjects` at `Start` time and spin one subscriber goroutine per project. Document new-project-during-runtime as known limitation — Drop 4c can add the project-creation event if dogfood needs it.

### 4.6 Test Scenarios

Named tests in `continuous_test.go`, `hylla_reingest_test.go`, and `main_test.go`:

1. `TestDispatcherStartSubscribesAndDispatches` — stub broker emits an `ActionItemChanged` event; stub walker returns one eligible item; assert `RunOnce` is called with that item's ID.
2. `TestDispatcherStartIsIdempotent` — second `Start` returns `ErrAlreadyStarted`.
3. `TestDispatcherStopCancelsSubscriber` — `Stop` causes the subscriber goroutine to exit within deadline.
4. `TestDispatcherStopIsIdempotent` — second `Stop` returns nil immediately.
5. `TestDispatcherDebouncesBurstEvents` — 5 events emitted within 50ms; the walker is invoked exactly ONCE (or at most twice — bounded by debounce window). Assert call count.
6. `TestDispatcherRecoversFromPanic` — stub `RunOnce` panics; subscriber logs + restarts once. Second panic exits the loop cleanly.
7. `TestDispatcherSubscribesPerProject` — `Options.ProjectIDs = ["proj-a", "proj-b"]`; both projects' brokers receive subscriptions.
8. `TestRunOnceErrorsDoNotKillSubscriber` — `RunOnce` returns a non-nil error; the loop logs at Error and continues to the next event.
9. `TestHyllaReingestGateLogsWarningAndSkips` (hylla_reingest_test.go) — gate is invoked; assert logger captures the expected "Hylla MCP not connected" warning; gate returns nil.
10. `TestRunServeStartsAndStopsDispatcher` (main_test.go) — happy path: `runServe` boots, dispatcher starts, ctx cancellation triggers shutdown sequence; no goroutine leak (assert via `goleak` or equivalent if available; otherwise time-bounded shutdown).

### 4.7 Falsification Attacks + Mitigations

- **A1: Subscriber goroutine panics → silent stop.** Per Acceptance #3 above, defer-recover + 1-retry pattern. Verified by test scenario 6.
- **A2: Burst-event CPU starvation.** Acceptance #4 debounce (100ms window). Test scenario 5 verifies coalescing.
- **A3: `RunOnce` blocks indefinitely (e.g., spawn pipeline hangs).** **Mitigation:** the subscriber's per-event tree walk runs sequentially. A blocked `RunOnce` blocks the next event drain. **Acceptable** because Wave 2.8 monitor.go:Track is non-blocking (returns immediately after `cmd.Start`). If a future spawn becomes blocking, ship a per-`RunOnce` timeout.
- **A4: `Stop` deadlock — subscriber holds a lock or channel.** **Mitigation:** the subscriber goroutine reads from `subscribeBroker`'s output channel which is closed on ctx cancel (broker_sub.go:62-96 contract). `Stop`'s deadline-bounded wait ensures forward progress.
- **A5: Hylla MCP unavailable mid-drop.** Stub gate logs warning + returns nil → drop closeout proceeds. **Acceptable** per L7. Drop 5 dogfood validates whether real ingest gating is needed.
- **A6: Project disappearing during runtime (project archived/deleted).** Subscriber's per-project goroutine continues subscribing to a project that no longer exists. **Mitigation:** acceptable for 4b — `subscribeBroker` will receive zero events; the goroutine idles. Archived-project handling is a Drop-5-dogfood refinement.
- **A7: Race between `Start` and `Stop`.** **Mitigation:** the cancellable child context + state-flag pattern. Use a `sync.Mutex` to serialize the start/stop state transitions. Document explicitly in continuous.go.
- **A8: Walker returns thousands of eligible items.** **Mitigation:** `RunOnce` per item is sequential. For a healthy tree this is bounded by the active-cascade size (typically <50 in-progress items). If pathological, add per-walk concurrency bound — defer to Drop 5 evaluation.
- **A9: Hylla gate registered twice.** If Wave A's registry pattern allows duplicate registration, the second call panics or silently overwrites. **Mitigation:** Wave A's registry contract (TBD) must reject duplicate registration. Builder coordinates with Wave A's builder. Test scenario 9 asserts single registration.
- **A10: `runServe` shutdown sequence leaves dispatcher running.** **Mitigation:** the deferred `disp.Stop` with a fresh timeout context (5s) is the load-bearing safety. Test scenario 10 asserts shutdown.
- **A11: Multi-project subscription leaks one goroutine per project on Stop.** **Mitigation:** all per-project goroutines share the same parent ctx. Cancel propagates to all. Test scenario 7 asserts shutdown across both projects.

### 4.8 DB Action

**None.** Schema unchanged.

### 4.9 Blocked By

**Cross-wave (Wave A → C):** **`4b.7` BLOCKED_BY `4b.2`** (gate-runner registry). The `hylla_reingest` `gateFunc` registers atop Wave A's framework; the registry type signature is owned by Wave A.

**Within Wave C:** none — file/package-disjoint from 4b.5 and 4b.6.

**Hard prereqs (Drop 4a):** 4a.15 (`LiveWaitEventActionItemChanged` + `subscribeBroker`), 4a.18 (walker), 4a.19 (spawn — used indirectly via `RunOnce`'s 8-stage pipeline), 4a.23 (`RunOnce` real implementation). All on `main`.

### 4.10 Verification Gate

- `mage test-pkg internal/app/dispatcher` — continuous_test.go + hylla_reingest_test.go + existing tests all pass; coverage on continuous.go ≥80%.
- `mage test-pkg cmd/till` — main_test.go passes; runServe shutdown test verifies dispatcher lifecycle.
- `mage test-func internal/app/dispatcher TestDispatcherDebouncesBurstEvents` — explicit hot-path assertion.
- `mage ci` — full clean. No goroutine leaks (manual check via test output + `runtime.NumGoroutine` spot-check during shutdown test).

---

## 5. Cross-Droplet Disjointness Audit

### 5.1 File-Level

| Droplet | Files (NEW) | Files (EDIT) |
| ------- | ----------- | ------------ |
| 4b.5    | `internal/app/dispatcher/auth_revoke.go`, `..._test.go` | `internal/app/dispatcher/cleanup.go`, `cleanup_test.go`, `dispatcher.go`; `internal/app/auth_requests.go`; `internal/app/auth_requests_test.go` |
| 4b.6    | `internal/domain/git_status.go`, `..._test.go` | `internal/domain/errors.go`; `internal/app/service.go`; `internal/app/service_test.go` |
| 4b.7    | `internal/app/dispatcher/continuous.go`, `..._test.go`, `hylla_reingest.go`, `..._test.go` | `internal/app/dispatcher/dispatcher.go`; `cmd/till/main.go`; `cmd/till/main_test.go` |

**Same-file overlaps:**

- `internal/app/dispatcher/dispatcher.go`: edited by **4b.5** (`NewDispatcher` wires real revoker into `newCleanupHook`) **AND** **4b.7** (`Start`/`Stop` real impls + `Options` extension).

### 5.2 Package-Level

| Droplet | Packages |
| ------- | -------- |
| 4b.5    | `internal/app/dispatcher`, `internal/app` |
| 4b.6    | `internal/domain`, `internal/app` |
| 4b.7    | `internal/app/dispatcher`, `cmd/till` |

**Package overlaps:**

- `internal/app/dispatcher`: 4b.5 + 4b.7.
- `internal/app`: 4b.5 + 4b.6.

### 5.3 Resolution

Per `CLAUDE.md` § "File- and package-level blocking": package-level lock is enforced because same-package edits break each other's compile. Therefore:

- **4b.5 and 4b.7 share `internal/app/dispatcher` AND `dispatcher.go`** → **`4b.7` MUST be `blocked_by: 4b.5`**. (Or vice versa — author choice. Recommendation: **4b.5 first**, because its surface is smaller and 4b.7's `Options` extension is forward-looking.)
- **4b.5 and 4b.6 share `internal/app`** but DIFFERENT files (`auth_requests.go` vs `service.go`). Same-package compile-lock applies. Per Drop 3 droplet 3.21 / Drop 4a 4a.12 precedent, **textually disjoint same-package edits at different file/line ranges are allowed to parallelize**, with the explicit caveat that builder verifies disjointness pre-edit and `.githooks/pre-commit` (`mage format-check`) catches any rebase-induced collision before push. **Recommendation: parallelize 4b.5 and 4b.6** with a worklog note.
- **4b.6 and 4b.7 share NO package** → fully parallel.

### 5.4 Final blocked_by Wiring

```
4b.5 ──────────────── (no in-wave predecessors)
4b.6 ──────────────── (no in-wave predecessors)
4b.7 ──── blocked_by: 4b.5 (same-package + same-file dispatcher.go lock)
4b.7 ──── blocked_by: 4b.2  (Wave A — gate registry)
```

Sequencing: 4b.5 + 4b.6 fan out parallel from Wave A's prereq close. 4b.7 fires after BOTH 4b.5 and 4b.2 land.

---

## 6. Verification Gates Summary

Every droplet ends with the same `mage ci` gate. Per-droplet additions:

| Droplet | Targeted mage targets |
| ------- | --------------------- |
| 4b.5    | `mage test-pkg internal/app/dispatcher`, `mage test-pkg internal/app`, `mage ci` |
| 4b.6    | `mage test-pkg internal/domain`, `mage test-pkg internal/app`, `mage ci` |
| 4b.7    | `mage test-pkg internal/app/dispatcher`, `mage test-pkg cmd/till`, `mage test-func internal/app/dispatcher TestDispatcherDebouncesBurstEvents`, `mage ci` |

**Universal:** every builder runs `mage ci` cold-cache (`GOMODCACHE=tmp` precommit-equivalent) before push per memory rule `feedback_mage_precommit_ci_parity.md`.

---

## 7. Pre-MVP Rules (Carried Forward)

Per REVISION_BRIEF §5:

- No migration logic in Go. **4b.6 schema impact: none** (no schema change). **4b.5 / 4b.7: no schema change.** Dev fresh-DBs `~/.tillsyn/tillsyn.db` BEFORE `mage ci` is a defense-in-depth recommendation only.
- No closeout MD rollups (LEDGER / WIKI_CHANGELOG / REFINEMENTS / HYLLA_FEEDBACK). Each droplet writes a per-droplet worklog only.
- Opus builders. Every builder spawn carries `model: opus`.
- Filesystem-MD mode. No Tillsyn-runtime per-droplet plan items.
- Tillsyn-flow output style + Section 0 SEMI-FORMAL REASONING in every subagent response.
- Single-line conventional commits ≤72 chars.
- NEVER raw `go test` / `go build` / `go vet` / `mage install`. Always `mage <target>`.
- Hylla is Go-only today. Markdown sweeps fall back to `Read` / `rg` without logging Hylla misses.

---

## 8. Open Questions Routed To Plan-QA Falsification

- **Q1 — 4b.5 lease vs session revocation.** Acceptance #8 assumes session-revoke is sufficient. If `internal/adapters/auth/autentauth/service.go::RevokeAuthSession` does NOT cascade to lease invalidation, the planner's claim is wrong. **Builder verifies pre-edit; if cascade is absent, lands an explicit lease revoke alongside the session revoke.**
- **Q2 — 4b.6 path-traversal handling.** Acceptance §3.6 A7 says "wrap git's error verbatim." Is that the right user-experience for a malformed path, or should the helper pre-validate paths before invoking git? **Author's stance:** wrap verbatim — git's error is informative. Plan-QA falsification reviews.
- **Q3 — 4b.7 Option B (multi-project subscription).** Author recommends Option B in §4.5. Plan-QA falsification: is Option A (one process per project) actually simpler given dogfood operations? **Author's stance:** Option B. Single dogfood process. Drop 5 validates.
- **Q4 — 4b.7 debounce window.** 100ms is a guess. Lower would batch less, higher would delay promotion noticeably. **Author's stance:** 100ms is conservative; Drop 5 dogfood profiles + tunes. Builder makes the constant a package-level `const` for easy adjustment.
- **Q5 — 4b.7 gate registry shape.** Acceptance #5 + #6 reference Wave A's registry type. The exact signature is owned by `4b.2`. **Builder coordinates with Wave A's builder** at implementation time; the plan assumes a `map[string]gateFunc` registry where `gateFunc` is `func(ctx, item, project) error`. If Wave A's actual shape differs, 4b.7 builder adapts the `hyllaReingestGate` signature.
- **Q6 — 4b.7 panic-restart count.** Acceptance #3 says 1 retry. Is that too few (gives up too fast) or too many (masks bugs)? **Author's stance:** 1 retry is correct — it survives transient panics but surfaces persistent bugs to the dev. Plan-QA falsification reviews.
- **Q7 — 4b.5 `Service.RevokeSessionForActionItem` placement.** Should it live on `*app.Service` (auth_requests.go) OR be a free function in `internal/app/dispatcher/auth_revoke.go` that takes a narrow `actionItemAuthRevoker` interface? **Author's stance:** put the lookup-and-revoke convenience method on `*app.Service` to mirror the existing `RevokeAuthSession` pattern. The dispatcher-side `revokeAuthBundleForActionItem` adapter wraps it through the `actionItemAuthRevoker` interface for test injection.
