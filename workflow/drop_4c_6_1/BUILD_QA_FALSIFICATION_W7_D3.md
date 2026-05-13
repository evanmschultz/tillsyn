# W7.D3 — BUILD-QA-FALSIFICATION Verdict

**Date:** 2026-05-13
**Reviewer:** go-qa-falsification-agent (opus)
**Overall verdict:** PASS WITH NITS

---

## Attack Hypotheses Tested

### Hypothesis 1 — `till mcp` still works
- **Method:** Inspect `runMCP` body at `cmd/till/main.go:2564-2575`; verify mcpCmd registration at line 480 and dispatch case at line 2321; run `mage test-pkg ./cmd/till/...`.
- **Result:** REFUTED. `runMCP` cleanly wired through new `mcpCommandOptions{mcpEndpoint: "/mcp"}` initializer at line 386; `mcpCommandRunner` invokes `mcpstdio.RunStdio`. Tests `TestRunMCPCommandWiresStdioAndSharedRuntime` + `TestRunMCPCommandConfigOverrideUsesConfiguredDB` + `TestRunMCPCommandTreatsCanceledRunnerAsCleanShutdown` + `TestRunMCPCommandUsesInterruptEchoSuppression` all PASS in mage test-pkg run.

### Hypothesis 2 — `till capture-state` still works
- **Method:** Grep `TestRunCaptureState` in `cmd/till/main_test.go`; verify dispatch case retained.
- **Result:** REFUTED. `TestRunCaptureStateCommand` at line 1454 preserved; `mage test-pkg ./cmd/till/...` PASSES at 334 tests / 76.7% coverage.

### Hypothesis 3 — Auth-mutation tests preserved
- **Method:** Grep `TestRunAuthRequest*`, `TestRunIssueSession*`, `TestRunSession*` in main_test.go.
- **Result:** REFUTED. All auth-mutation tests still present: `TestRunAuthRequestApproveLifecycle` (line 836), `TestRunAuthRequestTerminalStatesAndFilters` (line 1040), `TestRunAuthRequestTimeoutMaterializesExpiredState` (line 1125), `TestRunAuthRequestCreateStampsCLIClientType` (line 1174), `TestRunAuthRequestCreateRejectsClientTypeFlag` (line 1250), plus session tests. No auth-mutation tests removed.

### Hypothesis 4 — `internal/adapters/server/` truly absent
- **Method:** `ls internal/adapters/server/` — should fail.
- **Result:** REFUTED. `ls` returns `No such file or directory`. `git grep "internal/adapters/server" -- '*.go'` returns ZERO hits. All four files (handler.go, handler_integration_test.go, handler_test.go, server.go) are staged for deletion (status `D` in column 1 after `git add -A`).

### Hypothesis 5 — `mcpCommandOptions` repurposed correctly
- **Method:** Read struct definition at `cmd/till/main.go:96-99`; trace through executeCommandFlow parameter, `runMCP` signature, and field access.
- **Result:** REFUTED. `mcpCommandOptions{mcpEndpoint string}` at line 97 is wired through executeCommandFlow signature (line 2076), passed to `runMCP(... opts mcpCommandOptions)` at line 2565, and read as `opts.mcpEndpoint` into `mcpcommon.Config.MCPEndpoint` at line 2568. Field flows into `mcpstdio.RunStdio` via `EndpointPath` (verified at `internal/adapters/mcp_stdio/stdio.go:34`).

### Hypothesis 6 — `dispatcher` import removed
- **Method:** `git grep "internal/app/dispatcher" cmd/till/main.go`.
- **Result:** REFUTED. Only the side-effect import `_ "github.com/evanmschultz/tillsyn/internal/app/dispatcher/cli_claude"` (line 30) remains, which is unrelated to the deleted `runServe` flow. The direct `internal/app/dispatcher` import used by `dispatcherFactoryFunc` is GONE.

### Hypothesis 7 — Help text consistency (`till --help` no longer lists `serve`)
- **Method:** Read `cmd/till/help.go` (no "till serve" key). Read `TestRunRootHelp` at main_test.go:471 — assertion list no longer contains "serve". Read `TestRunSubcommandHelp` at line 487 — "serve" sub-case removed.
- **Result:** REFUTED. `help.go` has zero serve-related entries (diff: -14 lines). Test assertion updated cleanly.

### Hypothesis 8 — `runMCP` signature consistency
- **Method:** Grep all `runMCP` references in cmd/till.
- **Result:** REFUTED. Single definition at line 2565 uses `mcpCommandOptions`; single call site at line 2321 passes `mcpOpts` (typed as `mcpCommandOptions`). No drift.

### Hypothesis 9 — No dangling `serveCommandRunner` references
- **Method:** `git grep "serveCommandRunner|serveCommandOptions|runServe|dispatcherFactoryFunc|dispatcherLifecycle|stubDispatcherLifecycle"` against all Go files.
- **Result:** REFUTED. Zero hits in any Go file. All symbols cleanly removed.

### Hypothesis 10 — `stubDispatcherLifecycle` removal complete
- **Method:** Grep `stubDispatcherLifecycle` against main_test.go + entire repo.
- **Result:** REFUTED. Zero hits. Removed `sync/atomic` import simultaneously (no longer needed once stub removed).

### Hypothesis 11 — `shouldMuteRuntimeConsole` serve case removed
- **Method:** Inspect diff at `TestShouldMuteRuntimeConsole` (line 1782) — serve case removed; verify implementation at `executeCommandFlow` line 2554 now lists only `"mcp"`.
- **Result:** REFUTED. Both production switch and test cases drop `"serve"` symmetrically.

### Hypothesis 12 — Test count regression accounting
- **Method:** Stash W7.D3, run `mage test-pkg ./cmd/till/...` — 343 tests pre. Unstash, re-stage, re-run — 334 tests post. Diff = 9 (7 top-level serve `func Test*`s + 2 sub-test cases removed from `TestRunSubcommandHelp`/`TestRunHelpPathsDoNotSeedMissingConfig`/`TestShouldMuteRuntimeConsole`).
- **Result:** REFUTED (with NIT). Plan spec said "X minus 7 serve tests"; actual delta is 9 because of removed sub-cases. Not a counterexample — the 2 extra deletions are sub-cases within shared table-driven tests, which is correct cleanup hygiene.

### Hypothesis 13 — Coverage didn't drop catastrophically
- **Method:** `mage ci` after staging deletions; observe cmd/till coverage = 76.7%.
- **Result:** REFUTED. Plan acceptance was 76.7% post vs 76.9% pre — within tolerance. mage ci passes the 70% minimum gate cleanly.

### Hypothesis 14 — No new exports
- **Method:** `git grep "func [A-Z]" cmd/till/main.go`.
- **Result:** REFUTED. Zero exported functions in `package main` — clean (and consistent with idiomatic Go for executables).

### Additional Attack — `mage ci` actually GREEN
- **Method:** Run `mage ci` from working tree.
- **Result:** Initial unstaged state caused `mage format` to fail with `lstat internal/adapters/server/...: no such file or directory` because gofumpt was invoked against a stale tracked-files list. After `git add -A` (which is the spec's `git rm -r` equivalent for staging deletions), `mage ci` PASSES: 3219 tests / 28 packages / cmd/till at 76.7%. **NOT a counterexample to the builder's claim** — the deletions WERE in the working tree; the issue is purely about staging discipline. Builder's claim of green ci assumes staged deletions, which is the correct contract per spec ContextBlocks line 719.

### Additional Attack — Residual literal `"serve"` strings in test
- **Method:** `git grep "\"serve\""` cmd/till/main.go cmd/till/main_test.go cmd/till/help.go.
- **Result:** ONE residual hit at `cmd/till/main_test.go:2280`. See NITs.

### Additional Attack — File/package gating bypass
- **Method:** Confirm edits limited to declared `paths`: `internal/adapters/server/` (DELETE), `cmd/till/main.go`, `cmd/till/main_test.go`, `cmd/till/help.go`.
- **Result:** REFUTED. `git status --porcelain` shows exactly these four file groups changed. No collateral edits.

### Additional Attack — Hidden dispatcher/dispatcher-import leftovers
- **Method:** Grep `dispatcher\\.` in main.go for residual direct uses of the deleted `dispatcher` package import.
- **Result:** REFUTED. Remaining hits at lines 864, 895, 1941, 2481, 2482 are unrelated: docstring text, `runFlow(cmd.Context(), "dispatcher.run")` literal string, command-name comparison literal. None require the removed `internal/app/dispatcher` direct import.

---

## Unmitigated Counterexamples

**None.** All 14 plan-spec hypotheses + 4 additional attack families produced no counterexample. mage ci is GREEN once deletions are staged (per spec ContextBlock invariant).

---

## NITs

### NIT-1 — Stale `"serve"` reference in `TestResolveRuntimePathsCommandsShareDefaultNonDevRuntime`
**Location:** `cmd/till/main_test.go:2268-2298` (docstring at 2268; iteration list at 2280).

The test docstring still reads `verifies root, mcp, and serve resolve the same non-dev default runtime` and the iteration is `commands := []string{"", "mcp", "serve"}`. The test happens to PASS post-W7.D3 because `resolveRuntimePaths` (line 1908) ignores its `command` parameter entirely — the function only reads `opts.configPath` / `opts.dbPath` / env-overrides / platform defaults. So passing `"serve"` produces identical output to `""` or `"mcp"`.

**Why it's a NIT not a counterexample:** the test does not fail. But it is semantically stale — exercising a command that no longer exists is dead test surface. The fix is one-character: drop `"serve"` from the slice and update the docstring to read `verifies root and mcp resolve the same non-dev default runtime`.

**Recommendation:** spawn a one-line cleanup builder, or absorb inline before commit.

---

## Verdict rationale

The builder's claim ("Deleted `internal/adapters/server/` + dependent code in cmd/till. mage ci GREEN at 76.7%. 0 grep hits for `adapters/server`.") survives all 14 plan-spec hypotheses and 4 additional attack families:

- `git grep "internal/adapters/server" -- '*.go'` returns ZERO hits — confirms extraction in W7.D2 was complete (the spec's mandatory belt-and-suspenders check).
- `mage ci` is GREEN once deletions are staged: 3219 tests pass across 28 packages; cmd/till coverage 76.7% (above 70% minimum).
- `till mcp`, `till capture-state`, and all auth-mutation tests are preserved (per the ContextBlocks `constraint` critical at spec line 718).
- `mcpCommandOptions` repurposing wires cleanly through executeCommandFlow → runMCP → mcpcommon.Config → mcpstdio.RunStdio.
- All 7 serve-specific top-level `func Test*` blocks deleted; 2 sub-test cases deleted from shared tables; `stubDispatcherLifecycle` + `dispatcherFactoryFunc` + `serveCommandRunner` cleanly removed alongside their only `sync/atomic` import.
- Diff-stat 7 files changed / +14 / -2510 matches the expected deletion shape.

The one NIT (stale `"serve"` reference at line 2280) does not block: the test PASSES and the residual reference is semantic-only documentation drift.

**Overall verdict: PASS WITH NITS.**
