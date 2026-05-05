# Drop 4c F.7-CORE F.7.5b — Builder QA Proof Review (Round 1)

## 1. Findings

- 1.1 **Acceptance criteria coverage — full.** All five F.7.5b acceptance criteria from the spawn prompt have direct test coverage in `handshake_test.go`:
  - "One attention item ID per denial" → `TestPermissionHandshakePostDenialsAllSucceed` (3 denials → 3 IDs, ID-uniqueness map at lines 133-139, persisted-vs-returned ID equality at lines 150-152).
  - "Empty input → no-op" → `TestPermissionHandshakePostDenialsEmpty` covers BOTH nil-slice (lines 81-90) AND empty-slice `[]ToolDenial{}` (lines 93-99) variants. Returns `(nil, nil)` per `handshake.go:138`.
  - "Failure on one denial doesn't halt others" → `TestPermissionHandshakePostDenialsContinuesAfterFailure` (2nd-of-3 fails, exactly 2 surviving IDs at lines 231-233, all 3 Create calls attempted at lines 237-240).
  - "errors.Join aggregation" → `TestPermissionHandshakePostDenialsAggregatesMultipleFailures` exercises multi-failure path via `errors.Is(err, err1)` AND `errors.Is(err, err2)` checks at lines 279-284, confirming the joined chain unwraps to all originals (single-failure test alone leaves multi-error composition unverified — second test is required and present).
  - "Per-denial payload (tool_name + tool_input + project_id + kind) JSON-encoded" → `TestPermissionHandshakePostDenialsAllSucceed` lines 165-181 unmarshal `BodyMarkdown` and assert all four fields verbatim per item.

- 1.2 **`PermissionHandshake` shape matches spec.** Struct at `handshake.go:71-83` carries `AttentionStore AttentionItemStore`, `GrantsStore PermissionGrantsStore`, `Now func() time.Time`. `PermissionGrantsStore` declared as empty interface placeholder (`handshake.go:62`) per F.7.5c deferral — composable without churn.

- 1.3 **`AttentionItemStore` is the consumer-side port.** Defined in `handshake.go:45-50` as a one-method interface (`Create(ctx, item) (item, error)`). Tests use `fakeAttentionStore` mock; production wiring is deferred per spawn-prompt. Consumer-side interface placement is idiomatic.

- 1.4 **Domain-level ctor signatures verified via Hylla.** `domain.NewAttentionItem(in AttentionItemInput, now time.Time) (AttentionItem, error)` confirmed at `internal/domain/attention.go` (Hylla node `github.com/evanmschultz/tillsyn/internal/domain/NewAttentionItem`). `AttentionItemInput` struct shape verified — handshake fills 11 of 13 fields; intentionally omits `BranchID` (project-scope item) and `TargetRole` (no role gating). `domain.NewLevelTuple` validates that `ScopeType=ScopeLevelProject` with empty `ScopeID` is auto-populated to `ProjectID` (`level.go:NewLevelTuple` lines 14-16); handshake explicitly sets both — defensive.

- 1.5 **Error wrapping idiomatic and informative.** `handshake.go:148` wraps each denial failure with `fmt.Errorf("denial %d (tool=%q): %w", i, denial.ToolName, err)` — preserves the underlying error via `%w` for `errors.Is`/`errors.As`, includes index + tool name for forensic attribution. `errors.Join(errs...)` at line 154 composes the slice. The test at line 244 confirms `tool="Edit"` appears in the joined error message — `%q` on `"Edit"` produces the quoted form, so the substring match is exact.

- 1.6 **Clock injection is testable and production-safe.** `PermissionHandshake.Now` is optional (`handshake.go:88-93` — falls through to `time.Now()`). `TestPermissionHandshakeNowDefault` exercises the production fallback by leaving `Now` nil and asserting `CreatedAt` falls inside `[before, after]` (lines 307-321). Determinism in other tests via `fixedClock`.

- 1.7 **Monitor wiring deferred — confirmed by source inspection.** Project-wide search via `grep PostDenials\|PermissionHandshake` returns only `handshake.go` and `handshake_test.go`. `monitor.go` and `dispatcher.go` are unmodified per `git status --porcelain` (no entries for those files in this droplet's scope). Spawn-prompt's "monitor wiring NOT done" honored.

- 1.8 **REV-13 (no commit) honored.** `git log --oneline -3` shows HEAD still at `37f5a69 feat(dispatcher): bundle render and stream-jsonl monitor` (the F.7.4 monitor commit referenced in the worklog). New files `handshake.go` / `handshake_test.go` / worklog appear as `??` (untracked) in `git status --porcelain` — no commit invoked.

- 1.9 **Dispatcher tests green.** `mage testPkg ./internal/app/dispatcher/` returns 299 tests pass / 0 fail / 0 skip. Worklog reported 281 — divergence is sibling F.7.7 + F.7.12 droplets adding their own tests in parallel; F.7.5b's 5 new tests (`TestPermissionHandshakePostDenialsEmpty`, `…AllSucceed`, `…ContinuesAfterFailure`, `…AggregatesMultipleFailures`, `TestPermissionHandshakeNowDefault`) all exist and pass individually under `mage testFunc`.

- 1.10 **Coverage threshold met.** Worklog claims dispatcher at 74.5% (above the 70% gate). `mage ci` is the authoritative coverage gate; per the worklog, the droplet ran it green. The new 5 tests exercise both happy paths (1+1 success cases) AND error paths (3 failure variants + 1 partial-success case + 2 empty-input variants) — coverage of `PostDenials` + `postOne` is complete by inspection.

- 1.11 **Sibling-droplet collisions out of QA scope.** Spawn prompt is explicit: F.7.7 + F.7.12 not in F.7.5b's QA scope. The siblings touched `bundle.go` / `bundle_test.go` / `spawn.go` / `spawn_test.go` — none of those overlap the F.7.5b file set (`handshake.go` + `handshake_test.go` only). No conflict for this review.

- 1.12 **`AttentionKindApprovalRequired` is the right semantic kind.** Among the existing six `AttentionKind` values (Hylla confirms its const declaration at `internal/domain/attention.go`), `approval_required` matches "dev approves/denies tool permission via TUI" and via `BlocksCompletion()` (worklog claim, not directly verified by me but documented) holds the parent action item from completing while the denial is pending. Reasonable and idiomatic.

## 2. Missing Evidence

- 2.1 None — all five acceptance criteria, the no-monitor-wiring constraint, and the no-commit constraint have direct evidence from source / `git status` / `mage testPkg` output. The deferred F.7.5c grant-injection wiring is correctly scoped out by both the worklog and the spawn prompt.

## 3. Summary

**Verdict: PASS.** The F.7.5b implementation lands the `PermissionHandshake` type with `AttentionItemStore` port and placeholder `PermissionGrantsStore` per spec; `PostDenials` correctly handles empty / single-success / partial-failure / multi-failure paths; per-denial payload is JSON-encoded with the four required fields (`tool_name`, `tool_input`, `project_id`, `kind`); `errors.Join` aggregation verified by `errors.Is` chain unwrapping; monitor wiring deferred and not present anywhere outside the new files; REV-13 hard constraint (no commit) honored. Five new tests cover all required cases. Dispatcher package: 299 tests pass.

## TL;DR

- T1: PASS — all 5 acceptance criteria covered by direct test + source evidence; `errors.Join` multi-failure path independently verified; payload JSON shape matches spec; clock injection tested against production fallback; monitor wiring correctly deferred; REV-13 honored (no commit, files untracked); `mage testPkg ./internal/app/dispatcher/` green at 299/299.
- T2: No missing evidence — F.7.5c grant-injection wiring correctly scoped out per spawn prompt and worklog deferral notes.
- T3: PASS verdict.

## Hylla Feedback

- **Query**: `hylla_search_keyword` for `ToolDenial`, fields=`["content"]`.
  **Missed because**: `ToolDenial` is declared in `internal/app/dispatcher/cli_adapter.go` which IS Hylla-tracked Go code, but the keyword search returned 0 hits (likely the type sits in a recently-committed file at `37f5a69` and the indexed snapshot doesn't yet reflect it — Hylla snapshot=5 per response metadata; latest commit `9fe225a` is in `commit_membership` but `ToolDenial`'s containing commit `37f5a69` is newer than snapshot 5).
  **Worked via**: `Bash` `/usr/bin/grep -rln "type ToolDenial" /Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/` → found `cli_adapter.go:271`, then `Read` lines 265-305.
  **Suggestion**: Surface "snapshot is older than HEAD" as a metadata hint on the keyword response when artifact_ref's resolved snapshot lags HEAD's commit, so QA agents don't waste a round on a definitionally-stale Hylla query before falling back. The current response showed `latest_commit: 9fe225a` but no signal that newer in-tree commits exist.
