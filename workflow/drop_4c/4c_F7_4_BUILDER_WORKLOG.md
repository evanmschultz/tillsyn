# Drop 4c — F.7-CORE F.7.4 Builder Worklog

Stream-JSON monitor (CLI-agnostic) — REV-7 supersession of inline-claude-then-refactor F.7.17.9.

## Outcome

`mage ci` GREEN. dispatcher coverage 74.1% (above 70% floor). All 10 new tests pass. Forbidden-literal grep clean: `monitor.go` contains no `system/init`, `"assistant"`, or `"result"` strings.

Single-line commit (orchestrator runs):

```
feat(dispatcher): CLI-agnostic stream-JSON monitor with adapter routing
```

## What shipped

- `internal/app/dispatcher/monitor.go` — added `MonitorLogger` interface, `Monitor` struct, `NewMonitor` constructor, `Run(ctx)` method, `ErrInvalidMonitorConfig` sentinel, `monitorScannerMaxBytes` const (1 MiB). Existing `processMonitor` (Wave 2.8 cmd-lifecycle tracker) preserved untouched — the new `Monitor` is the stream-JSON consumer; the names do not collide.
- `internal/app/dispatcher/monitor_test.go` — added 10 new test functions:
  - `TestMonitor_Run_MockAdapterIntegration` — fixture round-trip via MockAdapter, asserts sink ordering + terminal report.
  - `TestMonitor_Run_ClaudeAdapterIntegration` — fixture round-trip via real claudeAdapter (resolved via `lookupAdapter(CLIKindClaude)` so we inherit `cli_claude` side-effect import from spawn_test.go).
  - `TestMonitor_Source_NoCLISpecificEventLiterals` — load-bearing CLI-agnosticism guard; reads `monitor.go` source bytes via `runtime.Caller`-derived path and asserts `system/init` / `"assistant"` / `"result"` are absent.
  - `TestMonitor_Run_MalformedLineLoggedAndSkipped` — invalid JSON line skipped with warning; valid surrounding events still flow.
  - `TestMonitor_Run_EmptyLinesSkippedSilently` — blank/whitespace-only lines flow through without warning.
  - `TestMonitor_Run_ContextCancellation` — `ctx.Cancel()` returns `errors.Is(err, context.Canceled)` mid-stream via `blockingReader` test double.
  - `TestMonitor_Run_MultipleTerminalEventsReturnsLast` — defensive against multi-terminal streams; LAST report wins.
  - `TestMonitor_Run_SlowSinkDoesNotBlock` — buffered sink (cap=1) cannot block; dropped events logged.
  - `TestMonitor_Run_NilConfigRejected` — nil adapter / nil reader / nil monitor all wrap `ErrInvalidMonitorConfig`.
  - `TestMonitor_Run_NilSinkDoesNotPanic` — pin contract that nil sink is allowed.

## Algorithm summary (monitor.go)

1. Validate adapter + reader non-nil; fail loud with wrapped `ErrInvalidMonitorConfig`.
2. Wrap reader in `bufio.Scanner` with 1 MiB max-token buffer (claude assistant events can blow past 64 KiB default).
3. Per scanned line:
   - Check `ctx.Done()` between iterations (Scan itself doesn't observe ctx).
   - Skip empty / whitespace-only lines silently.
   - Copy line bytes (defend against scanner buffer reuse).
   - `adapter.ParseStreamEvent(buf)` → on error, log warning + continue.
   - Forward event to optional sink via non-blocking `select-default` (drop + log on full).
   - On `event.IsTerminal == true` → `adapter.ExtractTerminalReport` → remember the last one.
4. On scanner error → wrap and return.
5. On clean EOF → return last terminal report (zero value if none seen) + nil.
6. If ctx cancelled exactly at EOF → return `ctx.Err()` over the report (cancellation signal is more important).

## CLI-agnostic invariant

The monitor source contains no claude-specific wire-format event-type literals (no `system/init`, no `"assistant"`, no `"result"`). Verified via:

- `rg -l 'system/init|"assistant"|"result"' monitor.go` → no match.
- `TestMonitor_Source_NoCLISpecificEventLiterals` runs the same check from inside the test suite, regression-pinning the rule.

The Monitor routes events purely through `StreamEvent.Type` (string) + `IsTerminal` (bool); adapter-private decoding stays inside `adapter.ParseStreamEvent` + `adapter.ExtractTerminalReport`.

## Decisions / tradeoffs

1. **Single-file vs new file.** Spec listed `monitor.go (NEW)` but `monitor.go` already exists as the Wave 2.8 process-monitor (cmd-lifecycle). Adding to the existing file rather than creating a sibling keeps me inside the listed paths. The two concepts cohabit cleanly because the type names don't collide (`processMonitor` vs `Monitor`) and a long doc-comment header demarcates the section.
2. **Scanner buffer cap = 1 MiB.** Default `bufio.Scanner` 64 KiB ceiling is too small for claude's largest assistant events (long thinking blocks, multi-paragraph text, large tool inputs). 1 MiB is generous for every recorded fixture and small enough not to OOM on a runaway stream.
3. **Non-blocking sink.** Per droplet acceptance, slow consumers must NOT deadlock the reader. Implemented via `select { case sink <- event: default: drop+log }`. The slow-sink test pins the property by sending 17 events through a cap-1 channel and asserting Run completes in under 2 s.
4. **Sink is `chan<- StreamEvent`** (send-only direction). Caller owns receive + close. Tests close the sink AFTER Run returns so the drain loop terminates cleanly.
5. **Logger is an injected `MonitorLogger` interface** (single `Printf` method). Nil = discard. Production callers wire a `charmbracelet/log` adapter at the dispatcher boundary; the Monitor doesn't depend on a specific logger package.
6. **Multi-terminal is defensive, not contract.** Per memory §6 claude emits exactly one terminal `result` event per spawn. The Monitor still returns the LAST terminal report seen so future / alternate CLIs that emit trailing heartbeats don't silently lose the report.

## Hard guard re-discovered: blank-import collisions across test files

First attempt added `_ "github.com/.../cli_claude"` to `monitor_test.go` while `spawn_test.go` already had the same blank import. The dispatcher package then failed to compile under `go test -json` — the laslig/gotestout renderer suppressed the actual error message, only surfacing `[PKG FAIL] (0.00s)` with `0 build errors`. Removed my redundant blank import (the existing one in spawn_test.go covers the side-effect for the whole test binary), used `lookupAdapter(CLIKindClaude)` to fetch the registered claudeAdapter, and the package compiled cleanly.

Reporting this to the orchestrator: the `mage ci` / `mage testPkg` runner does NOT surface go-build errors when they originate inside `_test.go` files — `[PKG FAIL] (0.00s)` with `0 build errors` is the only signal, and the underlying compiler message is silently dropped. Recommendation: open a refinement against the laslig/gotestout integration so build errors propagate visibly. Without it, build agents (and humans) burn time guessing.

## Scope expansion reported

- `internal/app/dispatcher/spawn.go` — restored the `os` import that had been deleted (likely by a stray gofumpt or pre-existing unfinished edit) before I started. Without that one-liner the dispatcher package would not compile at all and acceptance verification was impossible. Single-line revert; non-controversial.

## Acceptance checklist

- [x] `Monitor` struct + `NewMonitor` constructor + `Run(ctx)` method shipped.
- [x] CLI-agnostic: monitor.go contains zero claude-specific event-type literals (verified by both shell `rg -l` and in-suite `TestMonitor_Source_NoCLISpecificEventLiterals`).
- [x] MockAdapter integration test passes (`TestMonitor_Run_MockAdapterIntegration`).
- [x] claudeAdapter integration test passes (`TestMonitor_Run_ClaudeAdapterIntegration`) — proves the seam is multi-adapter ready.
- [x] All 8 spec scenarios + 2 defensive nil-config tests pass (10 new tests).
- [x] `mage ci` GREEN.
- [x] Worklog written.
- [x] **No commit by builder.** Orchestrator commits after QA pair returns green.

## Hylla Feedback

N/A — task touched non-Go files only as far as test fixtures go (the `claude_stream_minimal.jsonl` and `mock_stream_minimal.jsonl` fixtures), and the Go work itself was confined to the dispatcher package whose adapter contracts I read directly via `Read` rather than Hylla because a) the spec embedded the relevant struct shapes verbatim, and b) the changes spanned uncommitted-since-last-ingest territory (REV-7 just landed). No Hylla queries fired, so no misses to report.
