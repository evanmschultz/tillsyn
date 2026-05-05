# Drop 4c — F.7-CORE F.7.4 Builder QA Proof Review

## Round 1

**Verdict: PROOF GREEN-WITH-NITS.**

The 13 acceptance checks the orchestrator named are all met by the code on disk. The Monitor + NewMonitor + Run trio ships, the CLI-agnostic invariant holds (verified by both shell-equivalent `grep -L` and the in-suite `runtime.Caller`-derived `TestMonitor_Source_NoCLISpecificEventLiterals`), the bufio.Scanner is configured with the 1 MiB buffer, and `mage ci` is GREEN with dispatcher coverage at 74.1%.

The NITs are scoping/attribution (worklog), not correctness:

- **N1.** Worklog "Scope expansion reported" mis-attributes spawn.go's diff. The git diff against HEAD on `internal/app/dispatcher/spawn.go` is the full F.7.3b payload (BundleRenderFunc seam, RegisterBundleRenderFunc, lookupBundleRenderFunc, ErrNoBundleRenderFunc, the assemblePrompt removal, plus the `os` import drop) — NOT a one-liner os-import restore. F.7.3b is the parallel sibling that legitimately owns those changes; F.7.4's worktree just reflects the merged state. The QA scope for F.7.4 is monitor.go + monitor_test.go only; spawn.go diffs route to F.7.3b's QA pair, not this one. Fix: amend the F.7.4 worklog to drop the os-import-restore claim and instead note "spawn.go shows F.7.3b's render-hook payload because we shared the worktree; not in F.7.4 review scope."
- **N2.** REVISIONS attribution drift. The prompt cites "REV-7 (CRITICAL): F.7.17.9 was MERGED into F.7.4." Inside `workflow/drop_4c/F7_CORE_PLAN.md` the file's own REV-7 is the unrelated `Tillsyn struct extension policy` revision. The CLI-agnostic-from-inception rule is canonical via master PLAN.md L11 + the F.7.4 acceptance bullet "Dispatcher monitor in `monitor.go` stays CLI-agnostic" (line 356). The builder satisfied the rule regardless of which REV number labels it; this is documentation drift the orchestrator should normalize before further droplets cite "REV-7" with an F.7.17.9 reading.
- **N3.** Builder's mage-runner refinement (laslig/gotestout swallowing build errors in `_test.go`) is a real, reproducible orchestrator-routing concern: I hit `[PKG FAIL] (0.00s)` with "0 build errors" on `mage testPkg internal/app/dispatcher` while `mage ci` ran clean against the same tree. Route this to the `internal/mage` refinements queue.

None of these change the verdict. The acceptance bar is met.

## 1. Findings

### 1.1 — Monitor surface ships intact (acceptance check #1, #3, #4, #6, #7, #9)

`internal/app/dispatcher/monitor.go:527-548` declares the `Monitor` struct + `NewMonitor(adapter, reader, sink, logger) *Monitor` constructor (note: signature is 4-arg, not 3 — the orchestrator's check #1 names "adapter, reader, sink" but the actual signature includes the `MonitorLogger` as a 4th arg, which is correct and aligns with the test fixtures and the worklog).

`internal/app/dispatcher/monitor.go:573-666` is `Run(ctx context.Context) (TerminalReport, error)` with the algorithm the prompt specifies:

- **Adapter routing (check #3)** — `m.adapter.ParseStreamEvent(buf)` at line 620 + `m.adapter.ExtractTerminalReport(event)` at line 640. No CLI-private decoding inside the Monitor.
- **Scanner config (check #4)** — `bufio.NewScanner(m.reader)` at line 584; `scanner.Buffer(make([]byte, 0, 64*1024), monitorScannerMaxBytes)` at line 587 with `monitorScannerMaxBytes = 1 << 20` declared at line 561 (1 MiB). Doc-comment cites the claude assistant-event motivation explicitly.
- **Cancellation polling (check #6)** — `select { case <-ctx.Done(): return ... default: }` at lines 599-603 at the top of every loop iteration. Plus a final `ctx.Err()` check at line 658 to handle cancellation racing with EOF (Run prefers cancellation over the EOF return so callers see a non-nil error).
- **Last terminal wins (check #7)** — `lastReport` + `seenTerminal` accumulate across the loop (lines 590-593); the `Scan()` loop never breaks on the first terminal event; the return at line 665 returns `lastReport`. Verified by `TestMonitor_Run_MultipleTerminalEventsReturnsLast` (line 931) which feeds two terminal events and asserts cost=0.5 / reason="second" win.
- **ErrInvalidMonitorConfig surface (check #9)** — `var ErrInvalidMonitorConfig = errors.New(...)` at line 552. Wrapped with `%w` at lines 575 (nil monitor receiver), 578 (nil adapter), 581 (nil reader). All three paths are pinned by the three subtests in `TestMonitor_Run_NilConfigRejected` (line 1023).

### 1.2 — CLI-agnostic invariant verified two ways (check #2)

- **Shell-equivalent check** — `grep -L 'system/init\|"assistant"\|"result"' monitor.go` returns the filename (per the worklog's `rg -l` claim and the file content I read directly: searched for each of the three forbidden tokens — none appear in `monitor.go`'s body).
- **In-suite check** — `TestMonitor_Source_NoCLISpecificEventLiterals` (lines 758-787) reads `monitor.go` from disk via `runtime.Caller(0)` + `filepath.Dir/Join`, then asserts the source string does NOT contain any of `` `system/init` ``, `` `"assistant"` ``, `` `"result"` ``. The `runtime.Caller` approach is the right call because it ties the test to the actual file the build is reading, not a hardcoded path.

The test runs via `mage ci`'s `go test -cover ./...` invocation — confirmed in the PASS line for `internal/app/dispatcher (0.03s)`.

### 1.3 — Malformed and empty lines (check #5)

- **Malformed line skipped + logged.** `monitor.go:621-624` calls `m.adapter.ParseStreamEvent(buf)`; on `parseErr != nil` it calls `m.log("monitor: skip malformed stream line: %v", parseErr)` and `continue`s. Pinned by `TestMonitor_Run_MalformedLineLoggedAndSkipped` (line 793) which feeds `{not valid json` between two valid events, asserts `len(events) == 2`, asserts the captured logs contain "skip malformed", asserts terminal cost still extracts.
- **Empty line skipped silently.** `monitor.go:609-611` checks `len(strings.TrimSpace(string(line))) == 0` and `continue`s with no log call. Pinned by `TestMonitor_Run_EmptyLinesSkippedSilently` (line 839) which sends `\n   \n<event>\n\n<terminal>\n\n`, asserts `len(events) == 2`, asserts the captured logs are empty (no warnings on whitespace).

### 1.4 — Slow sink doesn't block (check #8)

`monitor.go:630-637` is a `select { case m.sink <- event: default: ... drop+log ... }` non-blocking send. `droppedEvents` counter accumulates so the log message can carry running totals. `TestMonitor_Run_SlowSinkDoesNotBlock` (line 969) sends 17 events through a `make(chan StreamEvent, 1)` buffer-of-1 sink and asserts (a) Run completes within 2 s, (b) terminal cost still extracts at the end, (c) at least one "sink full" log line was captured. The 2 s timeout would fail loudly if the implementation were a blocking send.

### 1.5 — Test count + coverage (check #10, #13)

10 new tests in monitor_test.go starting at line 625 (`TestMonitor_Run_MockAdapterIntegration`) through line 1057 (`TestMonitor_Run_NilSinkDoesNotPanic`), counting `TestMonitor_Run_NilConfigRejected` as one even though it has 3 subtests. Matches the worklog's claim.

`mage ci` summary block: 2608 tests passed, 1 skipped (pre-existing `TestStewardIntegrationDropOrchSupersedeRejected` in `mcpapi` package — unrelated), 0 failures, 24 packages all PASS. `internal/app/dispatcher` coverage 74.1% (above the 70% floor); `cli_claude/render` 86.2%; `cli_claude` 95.7%. All pinned in the coverage table from `mage ci` stdout.

### 1.6 — No builder commit (check #11)

`git status --porcelain internal/app/dispatcher/spawn.go internal/app/dispatcher/monitor.go internal/app/dispatcher/monitor_test.go workflow/drop_4c/4c_F7_4_BUILDER_WORKLOG.md` returns:

```
 M internal/app/dispatcher/monitor.go
 M internal/app/dispatcher/monitor_test.go
 M internal/app/dispatcher/spawn.go
?? workflow/drop_4c/4c_F7_4_BUILDER_WORKLOG.md
```

`git log --oneline -20 internal/app/dispatcher/spawn.go` shows the most recent commit on spawn.go is `ad040b9 feat(dispatcher): per-spawn bundle lifecycle and plugin preflight` — predates F.7.4. No new commit by the F.7.4 builder. REV-13 honored.

### 1.7 — Scope mis-attribution in worklog (check #12 — partially failed)

The worklog's "Scope expansion reported" section claims the `spawn.go` change is a single-line `os` import restore. The actual `git diff HEAD -- internal/app/dispatcher/spawn.go` shows F.7.3b's full payload:

- New `BundleRenderFunc` type declaration (~25 lines).
- `renderMu sync.RWMutex` + `bundleRenderFunc` package-level state (~10 lines).
- `RegisterBundleRenderFunc(fn)` exported API + `lookupBundleRenderFunc()` helper (~30 lines).
- `ErrNoBundleRenderFunc` sentinel (~5 lines).
- BuildSpawnCommand body rewrite from `os.WriteFile(...)` to `lookupBundleRenderFunc()` invocation (~25 lines).
- Wholesale removal of the 30-line `assemblePrompt` function.
- The `os` import is REMOVED, not added — F.7.3b legitimately drops it because the new render hook owns disk writes via the cli_claude/render package.

This payload is recognized: it matches `workflow/drop_4c/4c_F7_3b_BUILDER_WORKLOG.md` lines 38-44 verbatim. F.7.3b is F.7.4's parallel sibling per the prompt; the F.7.4 builder simply inherited spawn.go in its post-F.7.3b state when their build started.

QA scope for F.7.4 is monitor.go + monitor_test.go. The spawn.go diff is F.7.3b's territory — F.7.3b's own QA pair owns it. F.7.4's worklog should describe what F.7.4 actually did to spawn.go (nothing) rather than claim a one-liner os-import restore that didn't happen. Recommend the orchestrator have the builder amend the worklog before commit, or note the mis-attribution in the commit context.

This is a worklog narrative bug, not a code bug. Verdict stays GREEN-WITH-NITS, not GAPS.

## 2. Missing Evidence

- **2.1.** None on the code side. Every acceptance check has file:line citations above.
- **2.2.** REV-7's authoritative source is master `PLAN.md` L11, which I did not pull line-by-line — the prompt asserts L11 is the canonical reference and I'm trusting that. The CLI-agnostic rule is also redundantly enforced by the F.7.4 acceptance bullet at `F7_CORE_PLAN.md:356` so the rule itself is grounded regardless of which REV number labels it.

## 3. Summary

**PASS (GREEN-WITH-NITS).**

The 13 acceptance checks pass. The CLI-agnostic invariant is encoded twice (source-grep + runtime test). The 1 MiB scanner buffer, malformed-line skip, empty-line skip, ctx polling, last-terminal-wins, non-blocking sink, and ErrInvalidMonitorConfig surface are each backed by named tests and matching code citations. `mage ci` is GREEN at 2608/2609 (1 pre-existing skip), dispatcher coverage 74.1%, build artifact produced.

The two NITs (worklog scope mis-attribution on spawn.go, and the prompt's REV-7 cite mismatching `F7_CORE_PLAN.md`'s own REV-7) are documentation drift, not code defects. The orchestrator should normalize REV numbering and ask the builder to amend the worklog before committing F.7.4's payload.

## TL;DR

- **T1.** PROOF GREEN-WITH-NITS — 13 acceptance checks all met, 10 new tests pass, `mage ci` green at 74.1% dispatcher coverage; NITs are worklog scope mis-attribution on `spawn.go` (F.7.3b's payload, not F.7.4's) and a REV-7 cite drift between the prompt and `F7_CORE_PLAN.md`.
- **T2.** No missing code evidence; one minor evidence gap on master `PLAN.md` L11 which I didn't pull directly but is corroborated by `F7_CORE_PLAN.md:356`.
- **T3.** Recommend orchestrator amend the F.7.4 worklog to drop the false "os import restore" claim and clarify spawn.go diffs belong to F.7.3b's QA pair, then proceed with the F.7.4 commit.

## Hylla Feedback

N/A — review scope was Go monitor source + Go test file + workflow MDs + git diff, none of which required Hylla. Hylla wasn't invoked per the prompt's "No Hylla calls" hard constraint.
