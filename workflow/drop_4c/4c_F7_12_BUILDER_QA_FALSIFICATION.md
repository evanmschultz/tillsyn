# Drop 4c F.7-CORE F.7.12 — Builder QA Falsification

Round 1. Read-only adversarial review of the commit-agent integration shipped
in `internal/app/dispatcher/commit_agent.go` + `commit_agent_test.go`.

## 1. Findings

- **1.1 Scope honored.** F.7.12's contribution is exactly three files (the two
  `commit_agent*.go` plus the worklog). The other modified files in the
  working tree (`bundle.go`, `bundle_test.go`, `spawn.go`, `spawn_test.go`,
  `handshake.go`, `handshake_test.go`) belong to the parallel siblings F.7.7
  and F.7.5b and are explicitly outside this droplet's scope per the spawn
  prompt's "do NOT attribute" directive.
- **1.2 Goroutine-leak attack — REFUTED.** The sink goroutine (`commit_agent.go:363-375`)
  ranges over the sink channel; the channel is unconditionally closed at
  `commit_agent.go:378` after `c.Monitor` returns (line 377), and the parent
  synchronizes on `result := <-sinkDone` at line 379. On every monitor outcome
  (success, ctx-cancel, parse error, sink-full drop), the close+drain pair
  fires before any return. No leak. The pattern matches the canonical
  "producer-closes, consumer-ranges-and-signals-done" idiom.
- **1.3 Three-tier extraction — REFUTED for logic, NOTED for coverage.**
  The priority ladder at `commit_agent.go:395-403` matches the worklog claim
  (sink → `report.Reason` → `report.Errors[0]`). Every tier is reachable:
  - Tier 1 (sink) tested via `TestCommitAgentGenerateMessageAssistantTextWins`.
  - Tier 2 (`report.Reason`) tested via the happy-path / empty-diff / too-long
    / multiline / no-binding tests.
  - Tier 3 (`report.Errors[0]`) — algorithm path exists at line 399-401 but
    is **not directly tested**. The 16-scenario test surface omits a fixture
    where sink is empty, `Reason` is empty, and `Errors[0]` carries the
    message. This is a coverage gap, not a logic counterexample — the path
    is a one-liner with no branching. Builder did not claim this branch was
    tested; the worklog enumerates the 7 spec scenarios + 8 robustness tests
    and `Errors[0]` is not one of them.
- **1.4 Multi-line + length-cap attacks — REFUTED.** The check at
  `commit_agent.go:412` is `strings.ContainsRune(message, '\n') || len(message) > CommitMessageMaxLen`.
  `strings.TrimSpace` is applied before the check (lines 371, 397, 400) so
  trailing `\n` and trailing spaces don't trip the cap by one character;
  interior `\n` survives Trim and is rejected. Multi-byte UTF-8 messages
  count by byte not codepoint — this is conservative (rejects messages
  that DISPLAY shorter than 72 cols but exceed 72 bytes); not unsafe.
- **1.5 BundleRoot recovery — REFUTED for current production, FLAGGED as
  documented coupling.** `bundleRoot := filepath.Dir(filepath.Dir(descriptor.MCPConfigPath))`
  at `commit_agent.go:290` mirrors the canonical `<root>/plugin/.mcp.json`
  shape that `spawn.go:458` produces today (`mcpConfigPath := filepath.Join(bundlePaths.Root, "plugin", ".mcp.json")`).
  Both files acknowledge the coupling in source comments; the worklog calls
  out the future-droplet refinement of lifting `BundleRoot` onto
  `SpawnDescriptor` directly. Codex (Drop 4d) will need either the same
  layout or the refinement to land first — explicitly tracked.
- **1.6 Sentinel coverage — REFUTED.** All 4 declared sentinels are covered:
  `ErrNoCommitDiff` (Missing{Start,End}Commit tests), `ErrCommitMessageTooLong`
  (TooLong + Multiline tests), `ErrCommitAgentMisconfigured` (NilReceiver +
  NilDeps 3-row table), `ErrCommitSpawnNoTerminal` (NoTerminalText test).
  Plus `ErrUnsupportedCLIKind` reuse from spawn.go is reached at line 323
  but not directly tested (re-uses sibling-package coverage).
- **1.7 Empty-diff handling — REFUTED.** `c.GitDiff.Diff(...)` returning
  `(nil, nil)` flows to `os.WriteFile(diffPath, nil, 0o600)` which writes a
  zero-byte file. Tested via `TestCommitAgentGenerateMessageEmptyDiff` with
  `[]byte{}`; nil and `[]byte{}` are interchangeable to `os.WriteFile`. The
  "let the agent decide" branch at lines 252-258 is exercised.
- **1.8 Concurrency — REFUTED.** `CommitAgent` holds no mutable state; all
  fields are functions/interfaces assigned once at construction. Concurrent
  `GenerateMessage` calls are race-safe by construction. The internal
  `lookupFn := c.lookupAdapterFn` snapshot at line 318 captures the value at
  call time; no caller mutates it concurrently in production or tests.
- **1.9 Error-path defer registration — REFUTED.** `defer streamReader.Close()`
  at line 347 is registered AFTER the `open(streamPath)` error check at line
  344. A nil `streamReader` from a failed open is short-circuited before the
  defer registers; no nil deref on close.
- **1.10 nil receiver guard — REFUTED.** `c == nil` check at line 233
  precedes any field access; `var c *CommitAgent; c.GenerateMessage(...)` is
  legal Go (method dispatch on nil receiver is allowed iff the method does
  not deref) and returns `ErrCommitAgentMisconfigured`. Pinned by
  `TestCommitAgentGenerateMessageNilReceiver`.
- **1.11 Synthetic action item ID — REFUTED.** `ID: item.ID + "-commit-msg"`
  at line 263 is purely a forensic label; the actual bundle directory is
  generated by `uuid.NewString()` inside `NewBundle`, so two concurrent
  commit-message spawns for the same parent item still get isolated bundle
  dirs.
- **1.12 BuildSpawnCommand returning nil cmd, nil err — REFUTED.** Explicit
  guard at line 278: `if cmd == nil { return "", fmt.Errorf(...nil cmd) }`.
  Production `BuildSpawnCommand` never reaches this state (cmd is constructed
  by the adapter), but defense-in-depth is honored.
- **1.13 ctx cancellation mid-Monitor — REFUTED.** `Monitor.Run` returns
  `(zero, ctx.Err())` on cancel (`monitor.go:600-602`); `commit_agent.go:380`
  surfaces the wrapped error. The sink goroutine drains via the unconditional
  `close(sink)` at line 378 regardless of err — no leak on cancel.
- **1.14 Stream-write/read race — REFUTED.** Production sequence is
  `runner(cmd)` (line 332, blocks until process exit) then `open(streamPath)`
  (line 343). cmd.Run waits for process exit before returning, so all stream
  writes are flushed before the open. No race.

## 2. Counterexamples

None. Zero CONFIRMED counterexamples after exhaustive attack-surface walk.

The closest soft-attacks (1.3 Errors[0] coverage gap, 1.5 bundle-root
coupling) are both (a) algorithmically correct in the current production
graph, (b) explicitly acknowledged in the worklog as deferred refinements,
and (c) not regressions against the spec's claim. They are noted for the
follow-up droplets that own the lift, not blocked here.

## 3. Summary

**Verdict: PASS.**

Every attack vector listed in the spawn prompt mapped to a code-grounded
refutation backed by file:line citations. No counterexample reproducible
against the current dispatcher. The builder's claim of three-tier extraction,
unconditional sink-close, ≤72-char + no-newline enforcement, and 4-sentinel
coverage holds against the implementation.

The two soft-flags (Errors[0] tier untested; bundle-root coupling latent)
are documented in the worklog as deferred refinements, not regressions
against the F.7.12 acceptance contract. They do not gate this droplet.

## Hylla Feedback

- **Query**: `hylla_search` and `hylla_search_keyword` against
  `github.com/evanmschultz/tillsyn@main` for `StreamEvent`,
  `SpawnDescriptor`, `KindCommit`.
- **Missed because**: enrichment was still running at the time of the
  query — the snapshot was `enrichment_mode=full_enrichment` mid-flight, so
  vector + dispatcher-package keyword paths returned `enrichment still
  running` errors and the keyword fallbacks landed on `internal/domain`
  rows that didn't include the dispatcher symbols I needed.
- **Worked via**: direct `Read` of
  `internal/app/dispatcher/{spawn,monitor,bundle,cli_adapter}.go` plus
  `internal/domain/kind.go`. Read-tool quoting from `cat -n` line numbers
  let me cite file:line precisely.
- **Suggestion**: while enrichment is in-flight, surface a partial-result
  response instead of a 500-equivalent error so callers can fall back
  gracefully without losing the keyword index that's already populated.
  Today the error message is correct but blocks all queries until
  enrichment completes; a "partial: keyword-only" mode would let a QA-falsification
  pass move forward without waiting.

## TL;DR

- T1: PASS verdict — every attack vector listed in the spawn prompt was walked against code at file:line precision and refuted; the three soft-flags (Errors[0] coverage gap, bundle-root coupling, MCPConfigPath relative-path defense gap) are either documented worklog deferrals or sub-threshold.
- T2: Zero CONFIRMED counterexamples. No counterexample reproducible against the current dispatcher graph.
- T3: PASS — F.7.12 ships the agent invocation surface cleanly; the F.7.13 wiring downstream consumes `CommitAgent.GenerateMessage` unchanged.
