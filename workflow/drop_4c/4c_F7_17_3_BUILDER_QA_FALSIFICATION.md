# Drop 4c · F.7.17.3 — `claudeAdapter` QA Falsification

**Reviewer:** go-qa-falsification-agent (read-only adversarial pass).
**Date:** 2026-05-04.
**Builder output under attack:** `internal/app/dispatcher/cli_claude/{adapter.go,argv.go,env.go,stream.go,adapter_test.go,testdata/claude_stream_minimal.jsonl}`.
**Anchors:** `workflow/drop_4c/F7_17_CLI_ADAPTER_PLAN.md` REVISIONS REV-1/REV-2/REV-5/REV-7; `~/.claude/projects/-Users-evanschultz-Documents-Code-hylla-tillsyn/memory/project_drop_4c_spawn_architecture.md` §3 + §6.

---

## A1. Hardcoded `claude` binary discovery — REFUTED (with documented L11 carve-out)

**Hypothesis:** PATH-resolution failure when dispatcher's `cmd.Env` PATH is set but lacks `~/.local/bin` (or wherever `claude` lives in dev's shell).

**Trace.** `adapter.go:85` `exec.CommandContext(ctx, claudeBinaryName, argv[1:]...)` → Go's `exec.LookPath` runs at `cmd.Run`/`cmd.Start` time using `os.Environ()` PATH at the time `exec.CommandContext` was *constructed*, not `cmd.Env` PATH. **This is a Go stdlib gotcha**: even though `cmd.Env` is overwritten with the closed-baseline PATH (`env.go` line 130), `exec.LookPath` already resolved against the *orchestrator's* PATH at line 85 and stored the absolute path on `cmd.Path`.

Verify: `os/exec` package — when you call `exec.Command("claude", ...)`, `LookPath` runs immediately, and on success `cmd.Path` is set to the resolved absolute path. Subsequent `cmd.Env` mutation does NOT re-resolve.

**So:** as long as `claude` is on the orchestrator's PATH at `BuildCommand` call time, the spawn finds the binary. Setting `cmd.Env` PATH afterwards governs what the spawned `claude` process sees for its OWN child processes (e.g., `claude` invoking `gh`, `git`, `mage`), not the resolution of `claude` itself.

This is actually **stronger** isolation than the docstring suggests, and matches how `internal/app/dispatcher/spawn.go:152` (the existing 4a stub) behaves. The L11 OS-trust-the-PATH concession is the right place for this — adopters who hide `claude` outside the orchestrator's PATH must fix it externally.

**Verdict:** REFUTED. No counterexample. Optionally improve adapter.go's `claudeBinaryName` doc-comment to note that PATH at `BuildCommand` call time is the resolution gate, not `cmd.Env` PATH — but that's a NIT, not a defect.

---

## A2. Argv recipe completeness vs memory §3 — REFUTED

**Walk every flag in memory §3 against `argv.go:67-122`.**

| §3 Flag | argv.go line | Status |
|---|---|---|
| `--bare` | 68 | Always-on. PASS |
| `--plugin-dir <bundle>/plugin` | 69 | Always-on, derived from `paths.Root`. PASS |
| `--agent <name>` | 70 | Always-on, from `binding.AgentName`. PASS |
| `--system-prompt-file <bundle>/system-prompt.md` | 71 | Always-on, from `paths.SystemPromptPath`. PASS |
| `--append-system-prompt-file` (conditional) | 77-79 | Emits ONLY when `paths.SystemAppendPath` non-empty. PASS |
| `--settings <bundle>/plugin/settings.json` | 82 | Always-on, derived from `paths.Root`. PASS |
| `--setting-sources ""` | 83 | Always-on, empty arg literal. PASS |
| `--strict-mcp-config` | 84 | Always-on bare switch. PASS |
| `--permission-mode acceptEdits` | 85 | Always-on. PASS |
| `--output-format stream-json` | 86 | Always-on. PASS |
| `--verbose` | 87 | Always-on bare switch. PASS |
| `--no-session-persistence` | 88 | Always-on bare switch. PASS |
| `--exclude-dynamic-system-prompt-sections` | 89 | Always-on bare switch. PASS |
| `--mcp-config <bundle>/plugin/.mcp.json` | 90 | Always-on. PASS |
| `--max-budget-usd <N>` (conditional) | 97-99 | Pointer-non-nil gate. PASS |
| `--max-turns <N>` (conditional) | 100-102 | Pointer-non-nil gate. PASS |
| `--effort <e>` (conditional) | 103-105 | Pointer-non-nil gate. PASS |
| `--model <m>` (conditional) | 106-108 | Pointer-non-nil gate. PASS |
| `--tools <list>` (conditional) | 114-116 | Slice-non-nil gate (per BindingResolved doc-comment). PASS |
| `-p "<prompt>"` | 121 | Always-on, empty arg placeholder per A3. PASS for shape; A3 covers the empty arg. |

**Extra emissions?** None. Adapter does NOT emit `--allowed-tools` / `--disallowed-tools` (per memory §5 these are skipped — Layer C is brittle alone, gating happens at Layer B settings.json). Note: `BindingResolved` carries `ToolsAllowed` + `ToolsDisallowed` fields and the adapter does NOT plumb them. **This is intentional per memory §5** but is undocumented in argv.go — a NIT (see A11).

**Verdict:** REFUTED on completeness. Memory §3 is honored verbatim.

---

## A3. `-p ""` empty placeholder — NIT (deferred to F.7.17.5; documented)

**Hypothesis:** claude headless with `-p ""` either errors, hangs, or runs with empty prompt.

**Probe-grounded ground truth (memory §6.6):** the actual probes that produced the cost data were `claude --bare -p "say hi" --output-format stream-json --verbose`. None of the probes used `-p ""`. So we don't have direct evidence.

**Behavioral inference from `claude --help` (memory §13 cites it):** `-p` (alias `--print`) takes a prompt argument. Empty string is syntactically valid argv (length-0 string), but semantically: claude with empty prompt likely either (a) errors with "no prompt provided" before any spawn turn, or (b) starts a turn with empty user message and the agent immediately terminates with `result/error` (or `result/end_turn` with no work).

**However:** F.7.17.3 IS NOT THE END STATE. The spawn-prompt explicitly says F.7.17.5 wires the real prompt. If the dispatcher actually executed this `*exec.Cmd` today, claude would error or terminate immediately. **But this droplet's contract is "BuildCommand returns a well-shaped Cmd"; Start/Run is droplet 5's job.** The argv shape contract is satisfied.

**Risk surface:** if the dispatcher's monitor (F.7.17.9) tries to consume the stream from a `-p ""` invocation BEFORE F.7.17.5 lands, it'll see a zero-or-one-event stream (likely just a `result` event). That's recoverable (parser would handle a single `result` event correctly per A6/A8). **NOT a blocker for this droplet.**

**Verdict:** NIT — flag the `-p ""` placeholder is a known gap blocking real spawns. argv.go:118-121 already documents it inline. Worklog §4 calls it out. F.7.17.5 is the explicit follow-up. No counterexample to this droplet's contract.

---

## A4. 18-name baseline coverage — REFUTED

**Walk `closedBaselineEnvNames` (env.go:37-58):**

Process basics (9, F.7.17 L6):
1. `PATH` — line 39
2. `HOME` — line 40
3. `USER` — line 41
4. `LANG` — line 42
5. `LC_ALL` — line 43
6. `TZ` — line 44
7. `TMPDIR` — line 45
8. `XDG_CONFIG_HOME` — line 46
9. `XDG_CACHE_HOME` — line 47

Network conventions (9, REV-2):
10. `HTTP_PROXY` — line 49
11. `HTTPS_PROXY` — line 50
12. `NO_PROXY` — line 51
13. `http_proxy` — line 52
14. `https_proxy` — line 53
15. `no_proxy` — line 54
16. `SSL_CERT_FILE` — line 55
17. `SSL_CERT_DIR` — line 56
18. `CURL_CA_BUNDLE` — line 57

**Total: 18 names. Matches builder claim and REV-2 spec exactly.**

**Verdict:** REFUTED. Coverage complete.

---

## A5. `ErrMissingRequiredEnv` with empty `Env` slice — REFUTED

**Trace.** `assembleEnv(binding)` where `binding.Env = nil` or `[]string{}`:

- Line 80-83: `bindingNames := make(map[string]struct{}, 0)` — empty map.
- Line 88: `emitted := make(map[string]string, 0+len(closedBaselineEnvNames))` — sized for baseline.
- Line 95-101: `for _, name := range binding.Env` — **zero iterations**, loop body never runs. No error possible.
- Line 108-117: baseline resolution proceeds normally.
- Returns `(out, nil)`.

**Test coverage:** `TestEnvBaselineNamesAllInherited` and `TestEnvBaselineUnsetNamesOmitted` both run with `binding.Env` unset (zero slice); both pass per worklog §6. So this case is exercised.

**Verdict:** REFUTED. Empty-Env handles cleanly.

---

## A6. `ParseStreamEvent` event family coverage — REFUTED

**Memory §6 lists 4 event types. Map them against `parseStreamEvent` (stream.go:90-151):**

| Memory §6 event | stream.go switch case | Canonical Type | IsTerminal |
|---|---|---|---|
| `{"type":"system","subtype":"init",...}` (§6.1) | line 111-122 | `"system_init"` (when subtype="init"), else `"system"` | `false` |
| `{"type":"assistant","message":{...}}` (§6.2) | line 123-134 | `"assistant"` | `false` |
| `{"type":"user","message":{...}}` (§6.3) | line 135-141 | `"user"` | `false` |
| `{"type":"result","subtype":"...",...}` (§6.4) | line 142-144 | `"result"` | `true` |
| Unknown type (forward-compat) | line 145-147 | raw discriminator string | `false` |

**Test coverage:** `TestParseStreamEventSystemInit`, `TestParseStreamEventAssistantWithTextAndToolUse`, `TestParseStreamEventUserToolResult`, `TestParseStreamEventResultTerminal`, `TestParseStreamEventUnknownType` — all 4 documented + forward-compat all tested.

**Verdict:** REFUTED. 4-family coverage complete.

---

## A7. Tool input extraction with multiple tool_use blocks — REFUTED (with documented behavior)

**Trace.** `populateAssistantBlocks` (stream.go:158-179):

```go
for _, b := range blocks {
    switch b.Type {
    case "text":
        if !textSet {
            ev.Text = b.Text
            textSet = true
        }
    case "tool_use":
        if !toolSet {
            ev.ToolName = b.Name
            ev.ToolInput = b.Input
            toolSet = true
        }
    }
    if textSet && toolSet { return }
}
```

**Behavior with multiple tool_use blocks:** picks the FIRST one (by position in `content` array). Subsequent matches skipped via `!toolSet` guard. Documented in stream.go:154-157 ("Subsequent matches are ignored — the canonical StreamEvent surfaces only the first of each kind. Forensic tooling can re-decode Raw for the full block list.") AND in stream.go:32-35 (claudeAssistantMessage doc-comment).

**Forward-compat fallback:** Raw bytes preserved on `ev.Raw`, so dispatcher consumers needing all tool_use blocks can re-decode. This matches §6.5 which specifies the dispatcher cares about terminal-event `permission_denials[]` (full list), not per-assistant-event tool_use enumeration.

**Counterexample attempted:** an assistant message with 3 tool_use blocks — does the parser drop the latter two? Yes. Is that a defect? **No** — the canonical StreamEvent shape (`cli_adapter.go:228-265`) is single-tool-per-event by design. Multi-tool surfacing is forensic tooling's job (Raw re-decode).

**Verdict:** REFUTED. First-match-wins is documented; fallback (Raw) preserves full data.

---

## A8. `ExtractTerminalReport` field coverage — REFUTED (with documented partial)

**Memory §6.4 lists carries. Compare to `claudeResultEvent` (stream.go:56-61):**

| Memory §6.4 field | Decoded? | Surfaced? | Notes |
|---|---|---|---|
| `is_error` (bool) | NO | NO | Not in TerminalReport schema; consumer reads `Errors` instead. |
| `duration_ms` / `duration_api_ms` / `num_turns` | NO | NO | Adapter-private; recoverable via Raw re-decode. |
| `result` (final agent text) | NO | NO | Adapter-private; recoverable via Raw. |
| `stop_reason` | NO | NO | Adapter-private. |
| `total_cost_usd` | YES (line 57) | YES (`Cost`) | PASS |
| `usage` / `modelUsage` | NO | NO | Adapter-private (memory §6.5: "Tillsyn records actual spend per spawn" — Cost is what's needed). |
| `permission_denials[]` | YES (line 58) | YES (`Denials`) | PASS |
| `terminal_reason` | YES (line 59) | YES (`Reason`) | PASS |
| `errors[]` | YES (line 60) | YES (`Errors`) | PASS |
| `fast_mode_state` | NO | NO | Adapter-private. |

**TerminalReport (cli_adapter.go:287-306) carries: Cost, Denials, Reason, Errors.** All four FIELDS that TerminalReport exposes are decoded. Adapter-private fields (`is_error`, `duration_ms`, `num_turns`, `result`, `stop_reason`, `usage`, `modelUsage`, `fast_mode_state`) are documented as living in `Raw` for forensic re-decode (stream.go:182-190).

**Absent-field handling:** `claudeResultEvent` uses pointer (`*float64`) for `TotalCostUSD` so `Cost == nil` distinguishes absent from zero. Other fields use string/slice zero (empty string / nil slice) which collapses absent and explicit-empty. That's acceptable: an absent `terminal_reason` and an explicit `terminal_reason: ""` are semantically equivalent (CLI didn't report a reason). Same for `errors[]` (absent and `[]` both mean "no errors").

**Test coverage:** `TestExtractTerminalReportPopulated` covers all four fields populated; `TestExtractTerminalReportNoCost` covers absent-cost-as-nil. Test gap: no explicit test for absent `permission_denials[]` (relies on `convertDenials` returning `nil` on empty input — line 217-219 `if len(in) == 0 { return nil }`).

**Verdict:** REFUTED. Field coverage matches TerminalReport schema; absent-cost handled cleanly. Test gap on absent-denials is a NIT but `convertDenials`'s nil-on-empty is correct by inspection.

---

## A9. `ErrMalformedStreamLine` consumer behavior — DEFERRED (consumer is F.7.17.9, not in this droplet)

**Hypothesis:** monitor (consumer) crashes / halts on `ErrMalformedStreamLine`.

**Out of scope.** This droplet ships ONLY the parser. The consumer is F.7.17.9 (per spawn-prompt notes) — a separate droplet. The parser's contract is documented in stream.go:11-15:

> Monitors log the error but do NOT halt the spawn — claude streams may (rarely) emit interleaved progress lines the canonical taxonomy doesn't cover, and we want forward-compat for new event types.

The contract is correctly documented at the producer; consumer compliance is the consumer's QA scope, not this droplet's.

**One observation:** `parseStreamEvent` returns `(StreamEvent{Raw: raw}, error)` on malformed JSON (line 99). The StreamEvent has Type="" (empty discriminator, before the switch ran) and IsTerminal=false. A consumer that ignores the error and calls `extractTerminalReport(ev)` would get `(zero, false)` because `ev.IsTerminal == false`. So the failure mode is silent-drop, which is the documented contract.

**Verdict:** DEFERRED. Out of this droplet's scope. Producer-side contract documented + tested.

---

## A10. `os.Environ()` leak guard — REFUTED

**Verified by direct read of `env.go`:** the file imports `os` (line 6) but uses ONLY `os.LookupEnv`. No `os.Environ()` call. `cmd.Env = env` at adapter.go:86 — the `env` slice is the assembled return from `assembleEnv`, which is bounded by `len(closedBaselineEnvNames) + len(binding.Env)`.

**Test coverage:** `TestEnvOSEnvironNotInherited` and `TestEnvNotInheritedFromOSEnviron` (the size-bounded variant) explicitly assert no leak. The size-bounded variant catches the regression where someone wires `cmd.Env = append(os.Environ(), ...)`.

**Verdict:** REFUTED. No leak path. Two tests guard the regression.

---

## A11. `AutoPush` / `CommitAgent` / `BlockedRetries` / `BlockedRetryCooldown` / `MaxTries` not surfaced as argv — REFUTED (correctly out of scope)

**Cross-check.** `BindingResolved` (cli_adapter.go:102-179) carries 13 fields. `assembleArgv` consumes:

- Surfaced as argv: `AgentName` (line 70), `MaxBudgetUSD` (97-99), `MaxTurns` (100-102), `Effort` (103-105), `Model` (106-108), `Tools` (114-116).
- Surfaced as env: `Env` (via assembleEnv).
- NOT surfaced (correctly): `CLIKind` (used by dispatcher to pick THIS adapter), `MaxTries` (dispatcher retry counter — pre-spawn), `AutoPush` (post-build gate — F.7-CORE F.7.13/F.7.14), `CommitAgent` (commit-message agent name — post-build), `BlockedRetries`/`BlockedRetryCooldown` (dispatcher retry policy on `outcome=blocked` — pre/post-spawn, not adapter), `ToolsAllowed`/`ToolsDisallowed` (memory §5: "Skip Layer C entirely... Use `--tools` ... ONLY when a kind wants engine-level minimization").

**Verify post-build gate consumers don't expect adapter:** `gates.go`, `gate_mage_ci.go`, `gate_mage_test_pkg.go` exist as sibling files in `internal/app/dispatcher/`. These are gate executors that run AFTER spawn termination. They consume `AutoPush` / `CommitAgent` directly from binding (not from adapter argv). No code path expects the claude adapter to expose them.

**Memory §5 confirms `ToolsAllowed`/`ToolsDisallowed` policy:**
> Skip Layer C entirely for typical kinds (CLI flags duplicate B with weaker pattern syntax).

The adapter not plumbing `ToolsAllowed`/`ToolsDisallowed` into argv aligns with this. **However**, this is undocumented in argv.go — a NIT. The adapter could add a doc-comment in argv.go explicitly noting "ToolsAllowed/ToolsDisallowed intentionally not plumbed; gating happens via settings.json deny patterns per memory §5".

**Verdict:** REFUTED on correctness. NIT on documentation.

---

## A12. Compile-time interface assertion location — REFUTED

**Location.** `adapter.go:55`:
```go
var _ dispatcher.CLIAdapter = (*claudeAdapter)(nil)
```

This is a **top-level package declaration**. Any signature drift on the three interface methods (`BuildCommand`, `ParseStreamEvent`, `ExtractTerminalReport`) breaks the build at this line. The `_` assignment forces the conversion to be type-checked at package load time without producing a runtime variable.

**Test coverage:** `TestNewReturnsCLIAdapter` asserts `New() != nil` and returns `dispatcher.CLIAdapter`. Combined with the compile-time assert, both signature conformance AND constructor return type are pinned.

**Verdict:** REFUTED. Compile-time guard correctly placed.

---

## A13. Test fixture format authenticity — REFUTED (with documented gap)

**Walk fixture (testdata/claude_stream_minimal.jsonl):**

Line 1: `{"type":"system","subtype":"init","cwd":"/tmp/work","session_id":"sess-abc","tools":["Read","Bash"],"model":"opus","permissionMode":"acceptEdits"}`

Compare against memory §6.1 documented carries: `cwd, session_id, tools, mcp_servers, model, permissionMode, slash_commands, agents, skills, plugins, output_style, claude_code_version, apiKeySource`.

Fixture has 6 of 13 documented fields. **Missing fields don't matter** because `parseStreamEvent` only reads `type` + `subtype`; the rest sit in Raw. The minimal fixture exercises the discriminator path correctly.

Line 2: `{"type":"assistant","message":{"content":[{"type":"thinking","thinking":"plan","signature":"sig"},{"type":"text","text":"hello world"},{"type":"tool_use","id":"tu-1","name":"Read","input":{"file_path":"/tmp/x"}}]},"session_id":"sess-abc","uuid":"u-1"}`

Matches memory §6.2 verbatim shape: `content` is array of blocks each with `type` discriminator. Three block types tested: `thinking`, `text`, `tool_use`. Round-trip test (`TestRecordedFixtureRoundTrip`) verifies first text="hello world".

Line 3: `{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"tu-1","content":"ok","is_error":false}]},"session_id":"sess-abc"}`

Matches memory §6.3 verbatim.

Line 4: `{"type":"result","subtype":"success","is_error":false,"duration_ms":1234,"num_turns":2,"result":"done","stop_reason":"end_turn","total_cost_usd":0.0123,"terminal_reason":"completed","permission_denials":[{"tool_name":"Bash","tool_use_id":"tu-2","tool_input":{"command":"curl evil.com"}}],"errors":[]}`

Matches memory §6.4: carries `is_error`, `duration_ms`, `num_turns`, `result`, `stop_reason`, `total_cost_usd`, `terminal_reason`, `permission_denials[]`, `errors[]`. Missing from §6.4 docs: `usage`, `modelUsage`, `fast_mode_state`. Same argument as line 1 — adapter doesn't decode them, so omission doesn't affect the round-trip test.

**Authenticity question:** is this output from a real `claude --output-format stream-json --verbose` probe, or a builder approximation? The shape matches memory §13's verbatim probe data closely (memory §6.6 records `total_cost_usd: 0.0123` for one probe — fixture uses 0.0123 exactly). This suggests the builder lifted the value from memory rather than capturing fresh. **Acceptable**: the fixture is a synthesis of documented per-field shapes, exercising the parser end-to-end. Real-world drift (claude adds new fields, renames, etc.) is a forward-compat concern handled by `default` case in switch + Raw retention.

**Counterexample attempted:** does the fixture contain any field claude doesn't actually emit? Cross-reference memory §6 verbatim claims — every fixture field is documented as a real claude field. No counterexample.

**Verdict:** REFUTED on shape authenticity. NIT: a follow-up could re-capture a fresh probe and diff against fixture; not blocking.

---

## A14. Memory rule conflicts — REFUTED

- **`feedback_no_migration_logic_pre_mvp.md`:** new package, no migration code. PASS.
- **`feedback_subagents_short_contexts.md`:** single surface (one CLI adapter). PASS.
- **`feedback_opus_builders_pre_mvp.md`:** worklog notes `(opus, sonnet-spawn-fallback)` — model selection is orchestrator's job, not adapter's. N/A.
- **`feedback_orchestrator_no_build.md`:** subagent edited code; orchestrator did not. PASS.

**Verdict:** REFUTED. No memory-rule conflict.

---

## A15. `os.Getenv` empty-string vs unset — REFUTED

**Code uses `os.LookupEnv` exclusively (env.go:96, 112).** `os.LookupEnv(name)` returns `(value, ok)` and distinguishes "set to empty" from "unset" via the bool. The code uses the bool correctly:

- Line 96-99: binding.Env path — `if !ok { return ErrMissingRequiredEnv }`. So a binding name set to empty string would be treated as PRESENT (the empty value would emit `NAME=`). That's the right semantic: dev set the var explicitly to "" (e.g., `export PROXY=`), the adapter forwards.
- Line 112-115: baseline path — `if !ok { continue }`. Unset name silently dropped.

**`TestEnvBaselineUnsetNamesOmitted` test caveat:** the test docstring at lines 207-208 acknowledges `t.Setenv` with empty string still EMITS NAME=, which is why the test uses `os.Unsetenv` directly. This shows the builder understands the distinction.

**Counterexample attempted:** binding.Env name set to "" — fails the strict check? No: `os.LookupEnv` returns `("", true)`, the `!ok` branch doesn't fire, value `""` stored in `emitted[name]`. Adapter emits `NAME=` to claude. Adopter who explicitly set the env var to empty wanted that semantic.

**Verdict:** REFUTED. `LookupEnv` correctly distinguishes; builder used the right primitive.

---

## NIT roll-up

Items raised but NOT counterexamples:

- **N1.** argv.go `claudeBinaryName` doc-comment could note `cmd.Env` PATH ≠ resolution PATH (resolution happens at construct time via orchestrator's PATH). [A1]
- **N2.** Worklog mentions test gap on baseline+binding overlap: `TestEnvNotInheritedFromOSEnviron` size assertion would fail if `binding.Env=["PATH"]` (overlap with baseline). The CODE handles overlap correctly (dedup on emit), but the TEST as written would crash on overlap input. Add a test case that exercises overlap explicitly. [Inferred from env.go trace; not in builder's enumerated list]
- **N3.** Test gap on absent `permission_denials[]` in `extractTerminalReport`. `convertDenials(nil)` returns `nil` by inspection but no test exercises the `len(in) == 0` branch through the public surface. [A8]
- **N4.** argv.go could explicitly document why `ToolsAllowed`/`ToolsDisallowed` are intentionally NOT plumbed (memory §5 Layer C skip). [A11]
- **N5.** `-p ""` placeholder is a known limitation blocking real spawns; F.7.17.5 is the named follow-up. Inline doc-comment at argv.go:118-121 + worklog §4 already track it. [A3]
- **N6.** Test fixture is a synthesis of documented shapes rather than a re-captured fresh probe. Forward-compat is handled correctly via default-switch + Raw retention; a follow-up "re-capture and diff" task could de-risk drift. [A13]

None of N1-N6 block this droplet's contract. All are deferred-improvement candidates.

---

## Hylla Feedback

N/A — action item touched only Go files but per the constraint "No Hylla calls", I did not query Hylla. Falsification grounded in `Read` of the package's six source files + `cli_adapter.go` + `spawn.go` + the spawn architecture memory + the F.7.17 plan REVISIONS.

---

## Final verdict — PASS

All 15 attacks REFUTED or correctly DEFERRED. NITs roll up for follow-up consideration but do not block droplet completion.

Builder ships a clean `dispatcher.CLIAdapter` implementation matching:

- F.7.17 REV-1 (hardcoded `claude` binary, no Command override).
- F.7.17 REV-2 (18-name closed POSIX baseline, 9 process basics + 9 network/TLS).
- F.7.17 REV-5 (method named `ExtractTerminalReport`).
- F.7.17 REV-7 (stream parsing inside the adapter).
- Memory §3 argv recipe verbatim.
- Memory §6 4-family event taxonomy.
- F.7.17 L4 / L6 / L7 / L8 / L9 / L10 / L11 / L13 / L14 / L15 locked decisions.
- F.7.17 P5 fail-loud-on-missing-required-env routes to pre-lock.

PASS-WITH-NITS-FOR-FOLLOW-UP. The NITs are all enumerated above and tracked at this file's location for the orchestrator's drop-end roll-up.
