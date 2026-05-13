# W7.D3 — BUILD-QA-PROOF Verdict

**Droplet:** `4c.6.1.W7.D3` — DELETE HTTP RESIDUE: remove what's left in `internal/adapters/server/` + `till serve` CLI
**Date:** 2026-05-13
**Reviewer:** go-qa-proof-agent (opus)
**Overall verdict:** PASS WITH NITS

---

## Acceptance Bullet Coverage

### A1. `internal/adapters/server/` directory does NOT exist post-deletion

- Evidence:
  - `ls internal/adapters/server` → `No such file or directory`.
  - `git status --short` lists four `D ` entries for the package's files (`server.go`, `httpapi/handler.go`, `httpapi/handler_integration_test.go`, `httpapi/handler_test.go`) — i.e. tracked deletions staged via `git rm -r`.
- Verdict: PASS.

### A2. `git grep "internal/adapters/server"` returns ZERO hits in Go source files

- Evidence:
  - `git grep "internal/adapters/server" -- '*.go'` → no output (exit 1, "ZERO GO HITS" sentinel).
  - Broader `git grep "adapters/server" -- '*.go'` → no output.
- Verdict: PASS.

### A3. `till serve` does NOT appear in `till --help` output

- Evidence:
  - `cmd/till/main.go` diff: `serveCmd` cobra command construction (~40 lines), `serveCommandRunner`, `serveCommandOptions`, `dispatcherLifecycle`, `dispatcherFactoryFunc`, `runServe` function, and `case "serve":` in `executeCommandFlow` all removed.
  - `rootCmd.AddCommand(...)` call at main.go:1823 no longer includes `serveCmd`; only `mcpCmd, authCmd, projectCmd, actionItemCmd, dispatcherCmd, ...` etc.
  - Root help example list no longer contains `"  till serve --http 127.0.0.1:4848"`.
  - `cmd/till/main.go` short description now reads "Local-first planning TUI with stdio MCP adapter" (HTTP-mention removed).
  - `cmd/till/help.go` diff: `"till serve"` entry in `commandHelpSpecs` map removed (lines 33–46 in pre-diff).
  - `cmd/till/main_test.go` diff: the `TestRunRootHelp` expected-commands list dropped `"serve"`; subcommand-help table dropped the `serve --help` case.
  - `git grep -i "serve" cmd/till/help.go` → exit 1 (no hits).
- Verdict: PASS.

### A4. `till mcp` STILL WORKS

- Evidence:
  - `mcpCmd` cobra command construction preserved at `cmd/till/main.go:480-498`.
  - `mcpCommandRunner` callsite preserved at `cmd/till/main.go:2321` via `runMCP(ctx, svc, authSvc, rootOpts.appName, mcpOpts)`.
  - `mcpCommandOptions` struct retained at `cmd/till/main.go:96-98` with `mcpEndpoint` field.
  - Retained tests exercise it end-to-end:
    - `TestRunMCPCommandWiresStdioAndSharedRuntime` (line 2301)
    - `TestRunMCPCommandConfigOverrideUsesConfiguredDB` (line 2350)
    - `TestRunMCPCommandTreatsCanceledRunnerAsCleanShutdown` (line 2379)
    - `TestRunMCPCommandUsesInterruptEchoSuppression` (line 2407)
  - `mage ci` reports 3219/3219 tests passing including the four above.
- Verdict: PASS.

### A5. `till capture-state` STILL WORKS

- Evidence:
  - `captureStateCmd` cobra command construction preserved at `cmd/till/main.go:975`.
  - Retained test: `TestRunCaptureStateCommand` at `cmd/till/main_test.go:1455`.
  - Listed in `executeCommandFlow` and in `ensureRuntimePathParents` whitelist.
  - `mage ci` GREEN.
- Verdict: PASS.

### A6. Auth-mutation tests in `cmd/till/main_test.go` STILL PASS (migrated in W7.D2 — not removed here)

- Evidence:
  - `TestRunAuthRequestCreateStampsCLIClientType` retained at `cmd/till/main_test.go:1174`.
  - `TestRunAuthRequestCreateRejectsClientTypeFlag` retained at `cmd/till/main_test.go:1250`.
  - `TestRunIssueSession*` (multiple, around lines 1200–1400) retained.
  - `mage ci` 3219/3219 GREEN.
- Verdict: PASS.

### A7. `mage ci` GREEN — failure here surfaces any missed extraction in W7.D2 (mandatory belt-and-suspenders check)

- Evidence:
  - `mage ci` invocation reports:
    - Test summary: `tests: 3219 / passed: 3219 / failed: 0 / skipped: 0 / packages: 28 / pkg passed: 28 / pkg failed: 0`.
    - "[SUCCESS] All tests passed"
    - "[SUCCESS] Coverage threshold met — All packages are at or above 70.0% coverage."
    - `cmd/till` coverage: 76.7%.
    - `mcp_common`: 71.0%, `mcp_rpc`: 74.2%, `mcp_stdio`: implied by package list (28 packages including all three new mcp_* packages).
    - "[SUCCESS] Built till from ./cmd/till"
- Verdict: PASS.

---

## NITs

### N1 — `"serve"` literal still referenced in two tests as a function input string

- Severity: low.
- Locations:
  - `cmd/till/main_test.go:2280` — `commands := []string{"", "mcp", "serve"}` inside `TestResolveRuntimePathsCommandsShareDefaultNonDevRuntime`.
  - `cmd/till/embeddings_cli_test.go:29` and :51 — `buildEmbeddingRuntimeConfig(cfg, "tillsyn-test", "serve")` plus an assertion that `runtimeCfg.WorkerID` contains `"tillsyn-test:serve:"`.
- Why it is a NIT, not a FAIL:
  - Both call sites pass `"serve"` as an opaque string parameter to functions that do NOT switch on it against any command-registry membership (`resolveRuntimePaths` is command-agnostic; `buildEmbeddingRuntimeConfig` only uses the string to compose a worker-id label). The tests still pass under `mage ci`.
  - However, the literal references a removed cobra command. Future readers will be misled into thinking `serve` is still a valid command path. The cleaner alternative is to replace `"serve"` with `"mcp"` (or any other still-registered command) in both call sites and the WorkerID-substring assertion, or to drop the `serve` case from the `TestResolveRuntimePathsCommandsShareDefaultNonDevRuntime` table entirely.
- Fix hint: `cmd/till/main_test.go:2280` → swap `"serve"` for an additional retained command name (or remove it); `cmd/till/embeddings_cli_test.go:29,51` → swap to `"mcp"` and update the substring assertion to match.

### N2 — Two doc-string references to "HTTP server surface" / "HTTP" remain in retained help text

- Severity: low.
- Locations:
  - `cmd/till/main.go:487` — `till mcp` Long doc reads "instead of the HTTP server surface".
  - `cmd/till/main.go:980` — `till capture-state` Long doc reads "the same capture_state bundle exposed through MCP and HTTP".
- Why it is a NIT, not a FAIL:
  - The spec does not literally require deleting all prose mentions of HTTP. The verbatim acceptance bullets focus on directory removal, `git grep` zero-hit on the import path, and `till serve` absence from `--help`. The two prose references survive `git grep "internal/adapters/server"` (they say "HTTP", not the import path) and `git grep -i "serve"` (the second one mentions "MCP and HTTP" but not the `serve` token outside `server-side` etc.).
  - Future readers will be confused by docs that pitch `till mcp` as an alternative to a nonexistent HTTP surface.
- Fix hint: drop or rephrase the HTTP-surface clause in the `till mcp` Long doc; for `till capture-state` either drop "and HTTP" or rephrase to "exposed through MCP".

---

## Verdict rationale

All seven acceptance bullets PASS with explicit, citation-backed evidence. `mage ci` is GREEN at 3219/3219, the new `internal/adapters/mcp_*` packages plus `cmd/till` all clear the 70 percent coverage floor, the `internal/adapters/server/` directory is gone, `git grep "internal/adapters/server"` on Go source returns zero hits, and the `till serve` cobra command (plus its help spec, examples, and test scaffolding) has been removed without disturbing `till mcp`, `till capture-state`, or the auth-mutation test surface.

Two low-severity NITs remain. N1 (stale `"serve"` string literals in `main_test.go` and `embeddings_cli_test.go`) leaves a token referencing a deleted command in the test surface — confusing to future readers but not a behavior bug because the functions consume the string opaquely. N2 (two HTTP-surface prose mentions in retained help text) similarly leaves stale prose referencing the deleted HTTP/serve surface in the `till mcp` and `till capture-state` long docs.

Neither NIT blocks PASS. Verdict: PASS WITH NITS.
