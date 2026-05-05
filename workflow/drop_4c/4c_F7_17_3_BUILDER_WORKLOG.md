# Drop 4c · F.7.17.3 — `claudeAdapter` Builder Worklog

**Builder:** go-builder-agent (opus, sonnet-spawn-fallback).
**Date:** 2026-05-04.
**Spawn-prompt anchor:** `workflow/drop_4c/F7_17_CLI_ADAPTER_PLAN.md` §F.7.17.3 + REVISIONS POST-AUTHORING (REV-1, REV-2, REV-5, REV-7).

---

## 1. Outcome

`claudeAdapter` shipped as a NEW sibling package under `internal/app/dispatcher/`:

```
internal/app/dispatcher/cli_claude/
├── adapter.go                            (struct + New() + 3 interface methods)
├── argv.go                               (assembleArgv pure fn)
├── env.go                                (assembleEnv + closed-baseline + ErrMissingRequiredEnv)
├── stream.go                             (parseStreamEvent + extractTerminalReport)
├── adapter_test.go                       (22 test functions, ~95.6% coverage)
└── testdata/
    └── claude_stream_minimal.jsonl       (4-line recorded fixture: system_init / assistant / user / result)
```

Compile-time conformance asserted via `var _ dispatcher.CLIAdapter = (*claudeAdapter)(nil)` in `adapter.go`.

---

## 2. Acceptance criteria status

| Criterion | Status |
|---|---|
| `cli_claude` package compiles; claudeAdapter implements `CLIAdapter` (compile-time assert). | Met (`adapter.go:60`). |
| `BuildCommand` returns `*exec.Cmd` with `Path == "claude"`, argv-recipe shape, env baseline + binding. | Met (`TestBuildCommandHardcodedBinary`, `TestBuildCommandArgvShapeMinimal`, `TestBuildCommandArgvShapeFullyPopulated`). |
| `ParseStreamEvent` maps the 4 documented event families + unknown-pass-through + malformed-fail. | Met (`TestParseStreamEvent*` × 7 cases). |
| `ExtractTerminalReport` returns populated report on terminal, `(zero, false)` on non-terminal. | Met (`TestExtractTerminalReport*` × 3 cases). |
| Tests cover env baseline 18 names + binding append + missing-fail-loud + argv conditional + 4 stream shapes + terminal extract. | Met (22 tests, 95.6% coverage). |
| `os.Environ()` NOT inherited (sentinel test). | Met (`TestEnvOSEnvironNotInherited` + stricter `TestEnvNotInheritedFromOSEnviron` size-bounded check). |
| `mage check` + `mage ci` green. | `mage ci` passed (2350 / 2351 tests; 1 pre-existing unrelated skip). New package coverage 95.6%. |
| Worklog written. | This file. |

---

## 3. Hard-constraint compliance

- **Edit scope:** ONLY `internal/app/dispatcher/cli_claude/` + this worklog. Verified via `git status` — no other tracked files changed.
- **Mage discipline:** every build/test gate ran via `mage testPkg`, `mage formatCheck`, `mage formatPath`, `mage ci`. Zero raw `go test` / `go build` / `go vet` invocations.
- **No `mage install`:** never invoked.
- **Hardcoded binary:** `claudeBinaryName = "claude"` constant (`adapter.go:39`); no Command override path.
- **Closed env baseline:** `closedBaselineEnvNames` slice in `env.go:29-49` carries all 18 names (9 process basics + 9 network/TLS conventions per REV-2). `os.Environ()` never referenced anywhere in package.
- **No Hylla calls:** package implementation grounded in `Read` / `LSP` of in-tree code (`cli_adapter.go`, `spawn.go`) + memory §3 / §6 verbatim probe data.

---

## 4. Design notes

### REV-1: Hardcoded `claude` binary

Per REV-1 the adapter HARDCODES its binary name. `adapter.go` declares `const claudeBinaryName = "claude"` and `BuildCommand` calls `exec.CommandContext(ctx, claudeBinaryName, argv[1:]...)` directly. There is NO Command field on `BindingResolved` to read; F.7.17.2 already removed it. Adopters who want vendored / sandboxed claude binaries set up PATH externally — `cmd.Env`'s PATH (set by `assembleEnv`) governs binary resolution at exec time.

### REV-2: Expanded closed baseline

The closed baseline has 18 names — 9 process basics (PATH, HOME, USER, LANG, LC_ALL, TZ, TMPDIR, XDG_CONFIG_HOME, XDG_CACHE_HOME) plus 9 network/TLS conventions (HTTP_PROXY, HTTPS_PROXY, NO_PROXY + lowercase variants, SSL_CERT_FILE, SSL_CERT_DIR, CURL_CA_BUNDLE). Both upper- and lower-case proxy variants ship because POSIX SDKs split: curl honors lowercase, language SDKs vary.

`assembleEnv` semantics:
- **Baseline** name with `os.Getenv == ""`: silently OMITTED (no `NAME=` emission).
- **Binding** name with `os.Getenv == ""`: FAIL LOUD via `ErrMissingRequiredEnv` wrapping the offending name. Pre-lock failure per F.7.17 P5.
- **Duplicate** (name in both lists): binding's resolved value wins — emitted exactly once in baseline declaration order.
- **`os.Environ()`**: never called. `cmd.Env` size is bounded by `len(baseline) + len(binding.Env)`.

### REV-5: Method named `ExtractTerminalReport`

Method is `ExtractTerminalReport(StreamEvent) (TerminalReport, bool)`. The interface in `cli_adapter.go` already pinned this name; the adapter just satisfies it. `claudeResultEvent` decodes `total_cost_usd → *float64`, `permission_denials[] → []ToolDenial`, `terminal_reason → Reason`, `errors → Errors`.

### REV-7: Stream parsing inside the adapter

Stream parsing lives at `stream.go:parseStreamEvent` — exported as a method on `claudeAdapter` via the interface. The four documented event families map as:

| Claude event | Canonical `StreamEvent.Type` | `IsTerminal` |
|---|---|---|
| `{"type":"system","subtype":"init"}` | `"system_init"` | `false` |
| `{"type":"assistant","message":{...}}` | `"assistant"` (+ extracted Text / ToolName / ToolInput) | `false` |
| `{"type":"user","message":{...}}` | `"user"` | `false` |
| `{"type":"result","subtype":"..."}` | `"result"` | `true` |

Unknown types pass through with raw type string + `IsTerminal=false` for forward-compat. Malformed JSON wraps `ErrMalformedStreamLine`.

### `-p` prompt placeholder

`BindingResolved` and `BundlePaths` (the F.7.17.2-landed types) carry no prompt body. The `-p` flag emits with an empty argument in this droplet — F.7.17.5 dispatcher wiring will route the assembled prompt body through a follow-up extension to either `BundlePaths` or the BuildCommand contract. The flag itself ships always-on per memory §3 so the argv shape stays stable across that wiring landing. Code comment in `argv.go:53-60` flags this for the F.7.17.5 builder.

### Path divergence vs sub-plan body

The F.7.17.3 sub-plan body (lines 295-296) names files at `internal/app/dispatcher/cli_adapter_claude.go` (flat). The spawn prompt explicitly directs to a NEW package `internal/app/dispatcher/cli_claude/`. Spawn prompt won — this aligns with the broader Drop 4c trajectory of one-package-per-CLI (so `cli_codex/` is purely additive in Drop 4d).

---

## 5. Test inventory (22 tests)

- `TestNewReturnsCLIAdapter` — constructor returns interface type.
- `TestBuildCommandHardcodedBinary` — REV-1 hardcoded `claude`.
- `TestBuildCommandArgvShapeMinimal` — always-on flags + conditional flags absent on minimal binding.
- `TestBuildCommandArgvShapeFullyPopulated` — all conditional flags emit when pointers non-nil.
- `TestBuildCommandMaxBudgetWholeNumberFormatting` — budget formatter parity with 4a.19 stub.
- `TestEnvBaselineNamesAllInherited` — all 18 baseline names propagate when set.
- `TestEnvBaselineUnsetNamesOmitted` — absent baseline names silently dropped.
- `TestEnvBindingNamesAppended` — binding.Env names resolve + appear.
- `TestEnvMissingBindingNameFailsLoud` — `ErrMissingRequiredEnv` wrap + name in message + nil cmd.
- `TestEnvOSEnvironNotInherited` — sentinel orchestrator var doesn't leak.
- `TestEnvNotInheritedFromOSEnviron` — strict size bound: `len(cmd.Env) == len(baseline) + len(binding.Env)`.
- `TestParseStreamEventSystemInit` — system/init mapping.
- `TestParseStreamEventAssistantWithTextAndToolUse` — first-text + first-tool-use extraction.
- `TestParseStreamEventUserToolResult` — user event mapping.
- `TestParseStreamEventResultTerminal` — result event sets IsTerminal=true.
- `TestParseStreamEventMalformedJSON` — wraps `ErrMalformedStreamLine`.
- `TestParseStreamEventMissingType` — discriminator-validation path.
- `TestParseStreamEventUnknownType` — forward-compat passthrough.
- `TestExtractTerminalReportPopulated` — cost + denials + reason + errors decode correctly.
- `TestExtractTerminalReportNoCost` — `Cost == nil` (NOT zero) on absent `total_cost_usd`.
- `TestExtractTerminalReportNonTerminalReturnsZeroFalse` — `(zero, false)` contract.
- `TestRecordedFixtureRoundTrip` — `testdata/claude_stream_minimal.jsonl` 4-line round-trip.

---

## 6. mage ci output (truncated)

```
Test summary
  tests: 2351
  passed: 2350
  failed: 0
  skipped: 1
  packages: 22

github.com/evanmschultz/tillsyn/internal/app/dispatcher/cli_claude   | 95.6%
Minimum package coverage: 70.0%.
[SUCCESS] Coverage threshold met
```

Skipped test (`TestStewardIntegrationDropOrchSupersedeRejected` in mcpapi) is pre-existing and unrelated.

---

## 7. Out-of-scope confirmations

- **No bundle materialization.** `<paths.Root>/plugin/...` files (`.claude-plugin/plugin.json`, `agents/<name>.md`, `.mcp.json`, `settings.json`) are NOT written by this droplet — F.7-CORE F.7.1 owns. `assembleArgv` just plumbs the conventional paths.
- **No `BindingResolved` resolution.** Droplet 8 priority cascade. We consume the resolved struct verbatim.
- **No dispatcher wiring.** Droplet 5 wires `BuildSpawnCommand → adapter.BuildCommand`. We don't touch `spawn.go`.
- **No prompt body source.** F.7.17.5 will route the assembled prompt body. `-p ""` placeholder for now.
- **No MockAdapter.** Droplet 4 ships in `cli_adapter_test.go` (parent package). Compile-time conformance assertion in `adapter.go` is sufficient for this droplet.

---

## 8. Hand-off to QA twins

QA Proof + QA Falsification spawn against this droplet next. They verify:

- Proof: every acceptance criterion has a corresponding test that actually exercises the asserted behavior (not vacuous).
- Falsification: counterexamples — env merge ordering with duplicates, malformed assistant message, terminal event with `is_error=true` but no `terminal_reason`, conditional flag emission with explicit-zero-pointer (`*int` pointing to `0`), os.Environ leak via `cmd.Env = nil` regression.
