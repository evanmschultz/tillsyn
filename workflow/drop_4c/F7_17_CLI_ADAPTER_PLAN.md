# Drop 4c — F.7.17 CLI Adapter Seam (PLAN)

**Author date:** 2026-05-04.
**Author role:** go-planning-agent (parallel-pass: F.7.17 only).
**Companion plans (NOT this file's scope):** F.7.1-F.7.16 core spawn pipeline; F.7.18 context aggregator.

---

## Scope

F.7.17 architects the CLI adapter seam so Drop 4d (codex) is **purely additive**. Drop 4c ships ONLY the `claude` adapter; codex stub paths return "not yet supported." The F.7 multi-CLI roadmap is paper-spec only inside this drop — `ConsumeStream` does NOT land here.

In-scope deliverables:

- `Command []string`, `ArgsPrefix []string`, `Env []string`, `CLIKind string` on `templates.AgentBinding` (Schema-1 of the F.7 wave).
- `validateAgentBindingCommandTokens` (regex literal pinned + closed shell-interpreter denylist).
- `validateAgentBindingEnvNames` (`^[A-Za-z][A-Za-z0-9_]*$`).
- `CLIKind` closed enum + `CLIAdapter` interface + canonical value objects (`BindingResolved`, `BundlePaths`, `StreamEvent`, `ToolDenial`, `TerminalReport`).
- `claudeAdapter` struct implementing `CLIAdapter` (moves F.7.3 + F.7.4 logic into adapter methods, no behavior change).
- `MockAdapter` test fixture proving the seam is multi-adapter-ready.
- `BuildSpawnCommand` rewired to dispatch via `CLIKind` → adapter lookup.
- `manifest.json` `cli_kind` field + orphan-scan adapter routing.
- `permission_grants` SQLite schema gains `cli_kind` column.
- `BindingResolved` priority-cascade resolver (`CLI > MCP > TUI > template TOML > absent`).
- CLI-agnostic monitor refactor (F.7.4 retro-edit).
- Marketplace install-time interactive confirmation **paper-spec** (F.4 owns the CLI).
- Adapter-authoring documentation droplet.

Out-of-scope:

- F.4 marketplace CLI implementation (separate theme).
- F.7.18 context aggregator (separate planner).
- Schema-2 (`Context` sub-struct on `AgentBinding`) + Schema-3 (`Tillsyn` top-level globals) — owned by F.7.18 planner.
- The `ConsumeStream` interface rewrite (future-drop coordinated breaking change; documented in droplet 4c.F.7.17.11 only).
- Codex adapter implementation (Drop 4d).
- Windows / non-POSIX support.
- Adversarial OS-level sandbox; real-time interactive prompts.

---

## Hard Prerequisites

- Drop 4a closed (dispatcher core, manual-trigger CLI, locks).
- Drop 4b closed (gate execution).
- Schema-1 (this plan's first droplet) lands BEFORE any other F.7 droplet that consumes a wider `AgentBinding`. F.7.18 Schema-2 lands AFTER Schema-1; F.7.18 Schema-3 lands AFTER Schema-2.
- 4a.19 stub `BuildSpawnCommand` (`internal/app/dispatcher/spawn.go:106-166`) is the wholesale-replacement target.

---

## Locked Architectural Decisions

Sourced from `SKETCH.md` lines 147-180, `4c_F7_EXT_PLANNER_REVIEW.md` P1-P13, `4c_F7_EXT_QA_FALSIFICATION_R2.md` A1-A8 verdicts, dev decisions ratified 2026-05-05.

| ID | Decision | Source |
|----|----------|--------|
| L1 | `command` is argv-list (`[]string`), never a string. No shell parsing, no `sh -c`. | SKETCH §F.7.17 line 152 |
| L2 | Per-token regex `^[A-Za-z0-9_./-]+$` enforced via `regexp.MustCompile(...).MatchString(token)` literally. | A1.c |
| L3 | Closed `command[0]` shell-interpreter denylist: `{sh, bash, zsh, ksh, dash, fish, tcsh, csh, ash, busybox, env, exec, eval, /bin/sh, /bin/bash, /usr/bin/env, python, python3, perl, ruby, node}`. | A1.a |
| L4 | `env` is list of NAMES; values resolved via `os.Getenv(name)` at spawn time; missing fails loud per-action-item with `metadata.failure_reason`. | SKETCH line 154, P5 |
| L5 | `env` regex `^[A-Za-z][A-Za-z0-9_]*$` (BOTH uppercase and lowercase allowed; rejects `=`, whitespace, dashes, dots, empty, duplicates). | A2.d, P4 |
| L6 | Closed POSIX env baseline: `PATH, HOME, USER, LANG, LC_ALL, TZ, TMPDIR, XDG_CONFIG_HOME, XDG_CACHE_HOME` PLUS resolved per-binding `env` names. | A2.a |
| L7 | `PATH` value = `os.Getenv("PATH")` (inherit-PATH; the closed-baseline purpose is to block direnv-style secret-bearing env vars, not to relocate binaries). | A2.b |
| L8 | `os.Environ()` is NOT inherited. `cmd.Env` is set explicitly to L6 contents. | SKETCH line 155 |
| L9 | POSIX-only (macOS / Linux); Windows deferred to post-MVP refinement. | A2.c |
| L10 | `CLIAdapter` interface has THREE methods: `BuildCommand(ctx, BindingResolved, BundlePaths) (*exec.Cmd, error)`, `ParseStreamEvent(line []byte) (StreamEvent, error)`, `ExtractTerminalReport(StreamEvent) (TerminalReport, bool)`. | A4.b, P1 |
| L11 | `TerminalReport struct { Cost *float64; Denials []ToolDenial; Reason string; Errors []string }` — pointer-cost / pointer-denials so non-cost-emitting CLIs degrade cleanly. | SKETCH line 169 |
| L12 | Future non-JSONL CLI = HARD-CUT interface rewrite (no add-then-deprecate). Documented; does NOT land in Drop 4c. | A4.a |
| L13 | `BundlePaths` is a thin handle: `Root, StreamLog, Manifest`. Each adapter materializes CLI-specific subdirs inside `Root` via `BuildCommand`. | P2 |
| L14 | `StreamEvent` is the minimal cross-CLI canonical shape (terminal? cost? denials? final-text?); CLI-specific fields stay inside the adapter. | P1 |
| L15 | Default missing `cli_kind` resolves to `claude` for backward-compat. | SKETCH line 149 |
| L16 | `BindingResolved` priority cascade: `CLI flag > MCP arg > TUI override > template TOML > absent`. Pure function, consumed by every adapter. | SKETCH line 129, memory §3 |
| L17 | `MockAdapter` test fixture in `internal/app/dispatcher/cli_adapter_test.go` exercises the contract WITHOUT real CLI binaries. | A8.a |
| L18 | `manifest.json` carries `cli_kind` so orphan-scan picks the right adapter for liveness checks. | P13 §6.1 |
| L19 | `permission_grants(project_id, kind, rule, granted_by, granted_at, cli_kind)` — `cli_kind` discriminator added; pre-MVP rule = dev fresh-DB, NO migration code. | P13 §6.4 |
| L20 | Marketplace install-time confirmation is paper-spec only inside Drop 4c (F.4 owns the CLI). Project-local templates skip confirmation. | SKETCH line 160 |
| L21 | Schema-bundle SPLIT into three sequential droplets (Schema-1 = F.7.17, Schema-2 + Schema-3 = F.7.18). | A3.b |
| L22 | `exec.ErrNotFound` UX: dispatcher surfaces verbatim `os/exec` error + the binding's TOML position. Tillsyn does NOT recommend any specific install URL. | SKETCH line 161 |

---

## Cross-Droplet Sequencing (DAG)

```
                    +-----------------------------+
                    | 4c.F.7.17.1 Schema-1        |  <-- FIRST in F.7 wave
                    | (per-binding fields +       |
                    |  validators)                |
                    +--------+--------------------+
                             |
              +--------------+--------------+
              |                             |
              v                             v
+-----------------------------+   +-----------------------------+
| 4c.F.7.17.2 Pure types      |   | 4c.F.7.17.8 BindingResolved |
| (CLIKind, CLIAdapter,       |   | (priority cascade resolver) |
|  value objects)             |   +--------+--------------------+
+--------+--------------------+            |
         |                                 |
         v                                 |
+-----------------------------+            |
| 4c.F.7.17.3 claudeAdapter   |            |
| (implements CLIAdapter)     |            |
+--------+--------------------+            |
         |                                 |
         v                                 |
+-----------------------------+            |
| 4c.F.7.17.4 MockAdapter     |            |
| test fixture                |            |
+--------+--------------------+            |
         |                                 |
         v                                 |
+-----------------------------+            |
| 4c.F.7.17.5 Dispatcher      | <----------+
| wiring (BuildSpawnCommand)  |
+--------+--------------------+
         |
   +-----+------+
   |            |
   v            v
+--------+  +--------+
| 4c.F.7.|  | 4c.F.7.|
| 17.6   |  | 17.7   |
| manifest  | grants |
+--------+  +--------+
   |            |
   +-----+------+
         |
         v
+-----------------------------+
| 4c.F.7.17.9 monitor refactor|
| (CLI-agnostic via adapter)  |
+--------+--------------------+
         |
         v
+-----------------------------+
| 4c.F.7.17.10 Marketplace    |
| install confirmation        |
| (paper-spec MD only)        |
+--------+--------------------+
         |
         v
+-----------------------------+
| 4c.F.7.17.11 Adapter-       |
| authoring docs              |
+-----------------------------+
```

Hard constraints on ordering:

- **4c.F.7.17.1 (Schema-1) MUST land before any sibling F.7 droplet that reads a wider `AgentBinding`.** Strict-decode (`load.go:88-95`) rejects unknown nested fields.
- **4c.F.7.17.4 (MockAdapter) MUST land before 4c.F.7.17.5 (dispatcher wiring).** Wiring tests reference the mock to assert dispatcher-side adapter selection without invoking real CLIs.
- **4c.F.7.17.5 (dispatcher wiring) MUST land before 4c.F.7.17.6 + 4c.F.7.17.7.** Manifest + grants schemas read `cli_kind` from the dispatcher's resolved binding path.
- **4c.F.7.17.9 (monitor refactor) MUST land after 4c.F.7.17.5.** Monitor consumes adapter-returned `StreamEvent` values.

---

## Per-Droplet Decomposition

### 4c.F.7.17.1 — Schema-1: per-binding F.7.17 fields + validators

**Goal:** widen `templates.AgentBinding` with `Command`, `ArgsPrefix`, `Env`, `CLIKind` and bake load-time validators that close the marketplace-RCE vector.

**Builder model:** opus.

**Hard prereqs:** Drop 4b merged. NO sibling F.7 droplet may read a wider `AgentBinding` until this lands.

**Files to edit/create:**
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/templates/schema.go` (add four fields + denylist constant + new `CLIKind` type alias).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/templates/load.go` (wire two new validators into the `Load` chain).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/templates/load_test.go` (NEW unit-test cases — happy path + every reject case).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/templates/schema_test.go` (round-trip TOML decode for new fields).

**Packages locked:** `internal/templates`.

**Acceptance criteria:**
- [ ] `AgentBinding` gains four exported fields with TOML tags `command`, `args_prefix`, `env`, `cli_kind`.
- [ ] `CLIKind` type defined as `string` with closed-enum constants `CLIKindClaude = "claude"` and `CLIKindCodex = "codex"`. `IsValidCLIKind(CLIKind) bool` returns true for those two only. `cli_kind = ""` is permitted at the schema level (resolves to `claude` per L15) — empty string is a sentinel handled at adapter-lookup time, NOT a load-time reject.
- [ ] `validateAgentBindingCommandTokens(binding AgentBinding) error` invoked from `templates.Load` post-strict-decode for every `agent_bindings.<kind>` entry. Implementation contains EXACTLY:
  ```go
  var commandTokenRegex = regexp.MustCompile(`^[A-Za-z0-9_./-]+$`)
  // ...
  if !commandTokenRegex.MatchString(token) { /* reject */ }
  ```
  Anchors are part of the pattern AND the call uses `MatchString` against a compiled regex — A1.c mitigation pinned literally.
- [ ] `validateAgentBindingCommandTokens` rejects empty `command` array (when `command` field is non-nil but empty) AND rejects relative paths in `command[0]` matching `./` or `../` prefixes.
- [ ] Closed denylist constant `shellInterpreterDenylist = []string{"sh","bash","zsh","ksh","dash","fish","tcsh","csh","ash","busybox","env","exec","eval","/bin/sh","/bin/bash","/usr/bin/env","python","python3","perl","ruby","node"}` lives in `internal/templates/`. `validateAgentBindingCommandTokens` rejects when `command[0]` matches any denylist entry (exact-match, no fold).
- [ ] `validateAgentBindingEnvNames(binding AgentBinding) error` enforces:
  - Each entry matches `^[A-Za-z][A-Za-z0-9_]*$` (lowercase allowed).
  - Empty strings rejected.
  - Duplicates rejected (case-sensitive).
- [ ] New sentinel errors: `ErrInvalidAgentBindingCommand` + `ErrInvalidAgentBindingEnv`. Wrap `ErrInvalidAgentBinding` so `errors.Is(err, ErrInvalidAgentBinding)` continues to work.
- [ ] `Load` chain calls both validators after `validateGateKinds`. Both errors include the offending kind name + offending token/name in the wrapped message.
- [ ] When `command` is nil/absent in TOML, the binding is valid (default = `[]string{"claude"}` resolved at adapter-dispatch time, NOT at load time — keeps load-time semantics pure).

**Test scenarios (happy + edge):**
- Happy: `command = ["claude"]` loads.
- Happy: `command = ["wrapper-cli", "claude"]` loads.
- Happy: `command = ["bin/run.sh"]` loads.
- Happy: `command = ["/usr/local/bin/claude"]` loads.
- Reject: `command = []` (empty array).
- Reject: `command = ["rm; ls"]` (semicolon — A1.c).
- Reject: `command = ["valid; injected"]` (A1.c proof point).
- Reject: `command = ["sh", "-c", "claude"]` (denylist hit on `sh` — A1.a).
- Reject: `command = ["bash", "-lc", "claude"]` (denylist hit on `bash`).
- Reject: `command = ["/bin/sh", "-c", "claude"]` (denylist absolute-path hit).
- Reject: `command = ["python", "-m", "claude_runner"]` (denylist hit on `python`).
- Reject: `command = ["./relative/path"]` (relative path).
- Happy: `env = ["TILLSYN_API_KEY", "ANTHROPIC_API_KEY"]`.
- Happy: `env = ["https_proxy"]` (lowercase — A2.d).
- Reject: `env = ["FOO=bar"]` (contains `=`).
- Reject: `env = [""]`.
- Reject: `env = ["FOO", "FOO"]` (duplicate).
- Reject: `env = ["MY-VAR"]` (hyphen).
- Reject: `env = ["MY.VAR"]` (dot).
- Reject: `env = ["1FOO"]` (digit-leading).
- Strict-decode reject: TOML carrying unknown nested key under `[agent_bindings.build]` (e.g. `mystery_field = "x"`) fails with `ErrUnknownTemplateKey`.

**Falsification mitigations to bake in:**
- A1.a: closed shell-interpreter denylist as an explicit constant + tests asserting EVERY entry rejects.
- A1.c: regex literal pinned in code; test fixture `["rm; ls"]` MUST fail; `["valid_token"]` MUST pass; `["valid; injected"]` MUST fail.
- A2.d: env regex allows lowercase; `https_proxy` test passes.

**Verification gates:** `mage check` + `mage ci` + `4c.F.7.17.1.qa-proof` + `4c.F.7.17.1.qa-falsification`.

**Out of scope:**
- `cli_kind` validation at load time beyond "string field with TOML tag" — adapter-lookup-time resolution stays in droplet 5.
- Schema-2 (Context sub-struct) and Schema-3 (`Tillsyn` top-level) — owned by F.7.18 planner.
- Spawn-time `os.Getenv` resolution + missing-env error path — owned by droplet 3 (`claudeAdapter.BuildCommand`).

---

### 4c.F.7.17.2 — Pure types: `CLIAdapter` interface + value objects

**Goal:** land the cross-CLI canonical type vocabulary in `internal/app/dispatcher/` with NO behavior. Tests assert struct shape + enum membership only.

**Builder model:** opus.

**Hard prereqs:** 4c.F.7.17.1 merged (so the `templates.CLIKind` enum exists for cross-package reference if needed; this droplet does NOT depend on `templates` directly — it defines the dispatcher-side mirror).

**Files to edit/create:**
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/cli_adapter.go` (NEW — interface + value objects).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/cli_adapter_types_test.go` (NEW — assertion tests).

**Packages locked:** `internal/app/dispatcher`.

**Acceptance criteria:**
- [ ] `CLIAdapter` interface declared with three methods (signatures pinned per L10):
  ```go
  type CLIAdapter interface {
      BuildCommand(ctx context.Context, br BindingResolved, paths BundlePaths) (*exec.Cmd, error)
      ParseStreamEvent(line []byte) (StreamEvent, error)
      ExtractTerminalReport(ev StreamEvent) (TerminalReport, bool)
  }
  ```
- [ ] `BindingResolved` struct exposes resolved binding fields: `AgentName, Model, MaxBudgetUSD *float64, MaxTurns *int, Effort *string, Command []string, ArgsPrefix []string, Env []string, CLIKind CLIKind, SystemPromptPath string, AppendSystemPromptPath string, ToolsAllowed []string, ToolsDisallowed []string`. Pointer fields per L16 honor `absent` semantics.
- [ ] `BundlePaths` struct (per L13): `{ Root string; StreamLog string; Manifest string }`. Per-CLI subdirs are NOT in this struct — adapters compute them under `Root`.
- [ ] `StreamEvent` struct (per L14): minimal cross-CLI shape — `{ Type string; IsTerminal bool; RawJSON []byte }`. Adapters MAY decode `RawJSON` into adapter-private types when needed; the dispatcher only consumes `Type` + `IsTerminal`.
- [ ] `ToolDenial` struct: `{ ToolName string; ToolInput json.RawMessage }`.
- [ ] `TerminalReport` struct (per L11): `{ Cost *float64; Denials []ToolDenial; Reason string; Errors []string }`. Pointer-cost so non-cost-emitting CLIs degrade cleanly.
- [ ] `CLIKind` type re-declared in dispatcher package as `type CLIKind string` (mirror of `templates.CLIKind`) with constants `CLIKindClaude = "claude"`, `CLIKindCodex = "codex"`. `ResolveCLIKind(s string) CLIKind` returns `CLIKindClaude` for empty string (default per L15) and the explicit kind otherwise.
- [ ] Tests: enum membership; struct round-trip via reflection (assert field count + names so a future field rename is caught).
- [ ] Doc-comments on every exported type + method, citing L10 / L11 / L13 / L14 + this plan path.

**Test scenarios (happy + edge):**
- `ResolveCLIKind("")` → `CLIKindClaude`.
- `ResolveCLIKind("claude")` → `CLIKindClaude`.
- `ResolveCLIKind("codex")` → `CLIKindCodex`.
- `ResolveCLIKind("bogus")` → `""` (caller decides reject vs default; this droplet doesn't reject).
- Reflection test: `BindingResolved` has exactly the named fields above (catches accidental rename).
- `TerminalReport{}.Cost == nil` is the absent-cost case; explicit fixture asserts.

**Falsification mitigations to bake in:**
- A4.b: method named `ExtractTerminalReport` (not `ExtractTerminalCost`).
- A4 (broader): `TerminalReport.Cost` is `*float64` so non-cost CLIs return `nil` cleanly.
- P1 (planner-review): `StreamEvent` carries opaque `RawJSON []byte` so adapter-specific decoding stays inside the adapter.

**Verification gates:** `mage check` + `mage ci` + `4c.F.7.17.2.qa-proof` + `4c.F.7.17.2.qa-falsification`.

**Out of scope:**
- Any concrete adapter implementation.
- Stream-parsing logic.
- `BindingResolved` priority cascade — droplet 8.

---

### 4c.F.7.17.3 — `claudeAdapter` implementing `CLIAdapter`

**Goal:** move the existing 4a.19 stub `BuildSpawnCommand` argv-emission logic + the F.7.4 stream parser logic into `claudeAdapter` methods. NO behavior change visible to callers; tests assert byte-for-byte argv parity for fixed inputs.

**Builder model:** opus.

**Hard prereqs:** 4c.F.7.17.2 merged (interface + value objects exist).

**Files to edit/create:**
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/cli_adapter_claude.go` (NEW — adapter implementation).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/cli_adapter_claude_test.go` (NEW).

**Packages locked:** `internal/app/dispatcher`.

**Acceptance criteria:**
- [ ] `claudeAdapter` struct (zero-config; constructor `NewClaudeAdapter() CLIAdapter`).
- [ ] `BuildCommand(ctx, br, paths)` returns `*exec.Cmd` with:
  - `Path` derived from `br.Command[0]` (default `"claude"` when `br.Command` is nil/empty).
  - `Args` shape: `[Path] ++ ArgsPrefix ++ ["--bare", "--plugin-dir", <paths.Root>/plugin, "--agent", br.AgentName, "--system-prompt-file", <paths.Root>/system-prompt.md, "--settings", <paths.Root>/plugin/settings.json, "--setting-sources", "", "--strict-mcp-config", "--permission-mode", "acceptEdits", "--output-format", "stream-json", "--verbose", "--no-session-persistence", "--exclude-dynamic-system-prompt-sections"]` plus conditional `--max-budget-usd <N>` / `--max-turns <N>` / `--effort <e>` / `--model <m>` / `--append-system-prompt-file <path>` / `--tools <flag>` (each emitted ONLY when the corresponding pointer-typed `BindingResolved` field is non-nil).
  - `Env` set explicitly (NOT inherited via `os.Environ`) to the closed POSIX baseline (L6) PLUS the resolved values for every name in `br.Env`. `PATH = os.Getenv("PATH")` (L7).
  - `Dir` defaulted to the repo primary worktree (callers populate via `BindingResolved` or a future `BundlePaths.WorkingDir` — droplet 5 wires this).
- [ ] Missing required `env` name (where `os.Getenv(name) == ""`) returns wrapped error `ErrMissingRequiredEnv` so `BuildSpawnCommand` (droplet 5) routes it to early-pre-lock failure per P5.
- [ ] `ParseStreamEvent(line)` parses one JSONL line per claude's stream-json taxonomy (`system/init`, `assistant`, `user`, `result`). Returns `StreamEvent{Type, IsTerminal, RawJSON}`. `IsTerminal = true` only when `Type == "result"`.
- [ ] `ExtractTerminalReport(ev)` decodes `ev.RawJSON` into the claude-private result struct and returns `TerminalReport{Cost: &totalCostUSD, Denials: parsedDenials, Reason: terminalReason, Errors: errors}`. Returns `(TerminalReport{}, false)` when `ev.IsTerminal == false`.
- [ ] Argv-parity test: a fixed `BindingResolved + BundlePaths` fixture produces the SAME argv as the 4a.19 stub when no F.7-only flags are set. Assertion is byte-level equality on the joined argv slice. (4a.19 stub is the snapshot baseline; this confirms no behavior drift during the refactor.)
- [ ] `--mcp-config` flag is dropped from the claude adapter's argv. The 4a.19 stub emitted it but the post-F.7 architecture (memory §3) uses `--plugin-dir <bundle>/plugin` with the bundle's `.mcp.json` so `--mcp-config` is redundant. Document in droplet's commit message that this is the intentional drop replacing `mcpConfigPlaceholderPath`.
- [ ] Stream-event round-trip test: a recorded claude `system/init` line + `result` line in `testdata/` round-trip through `ParseStreamEvent` → `ExtractTerminalReport` and produce the expected `TerminalReport`.

**Test scenarios (happy + edge):**
- Argv parity vs 4a.19 stub for default binding.
- Argv with `args_prefix = ["--profile", "dev"]` correctly inserts the prefix between `Path` and the canonical flags.
- Argv with `command = ["wrapper-cli", "claude"]` runs `wrapper-cli` with `claude` as the first arg of its own argv (i.e. `Path = "wrapper-cli"`, `Args[1] = "claude"`, then the canonical flags).
- `BuildCommand` with `env = ["UNSET_VAR"]` returns `ErrMissingRequiredEnv`.
- `BuildCommand` with `env = ["TILLSYN_API_KEY"]` (set in test setup) carries that var into `cmd.Env` AND the closed-baseline names AND nothing else.
- `cmd.Env` does NOT contain `AWS_ACCESS_KEY_ID` even when present in the orchestrator's environment (proves L8 isolation).
- `ParseStreamEvent` on malformed JSON returns wrapped error.
- `ExtractTerminalReport` on a `result` event with no `total_cost_usd` returns `Cost: nil` (not zero).
- Recorded fixture: `testdata/claude_stream_minimal.jsonl` (3-line trace) round-trips.

**Falsification mitigations to bake in:**
- L8 isolation test: spawn-time `cmd.Env` excludes parent secret-bearing names.
- L11: nil-cost path tested.
- A4.b: method name verified at compile time.
- L20 / L22: `exec.ErrNotFound` UX is droplet 5's concern (BuildSpawnCommand wraps); this droplet only verifies the adapter returns whatever `os/exec` produces, no shadowing.

**Verification gates:** `mage check` + `mage ci` + `4c.F.7.17.3.qa-proof` + `4c.F.7.17.3.qa-falsification`.

**Out of scope:**
- Bundle materialization (`<paths.Root>/plugin/...` file writes) — F.7.1 (other planner).
- `BindingResolved` resolution from raw `templates.AgentBinding` — droplet 8.
- Dispatcher-level adapter selection — droplet 5.

---

### 4c.F.7.17.4 — `MockAdapter` test fixture

**Goal:** ship a `MockAdapter` exercising the `CLIAdapter` interface contract WITHOUT touching real CLI binaries. Confirms multi-adapter readiness pre-Drop-4d.

**Builder model:** opus.

**Hard prereqs:** 4c.F.7.17.3 merged (claude adapter exists for parity comparison).

**Files to edit/create:**
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/cli_adapter_test.go` (NEW — Mock + contract tests).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/testdata/claude_stream_minimal.jsonl` (NEW recorded fixture if not added by droplet 3).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/testdata/mock_stream_minimal.jsonl` (NEW recorded fixture for MockAdapter).

**Packages locked:** `internal/app/dispatcher`.

**Acceptance criteria:**
- [ ] `mockAdapter` struct (test-package, NOT exported in production code) implements `CLIAdapter`. Constructor `newMockAdapter(opts mockAdapterOpts)` accepts deterministic-fixture configuration.
- [ ] `BuildCommand` test: assert `*exec.Cmd.Path = "mock-cli"`, `*exec.Cmd.Args[0] = "mock-cli"`, `*exec.Cmd.Args` includes `[--mock-flag, fixture-value]`, `*exec.Cmd.Env` matches the closed-baseline + forwarded names.
- [ ] `ParseStreamEvent` round-trip test: recorded fixture line `{"type":"mock_terminal","cost":0.5,"denials":[]}` decodes into `StreamEvent{Type: "mock_terminal", IsTerminal: true, RawJSON: <bytes>}`.
- [ ] `ExtractTerminalReport` populates `TerminalReport{Cost: ptr(0.5), Denials: nil, Reason: "ok", Errors: nil}`.
- [ ] **Contract conformance test** — table-driven over `[]CLIAdapter{newMockAdapter(...), claudeAdapter{}}`. Each adapter's interface methods are called with a canonical fixture; assertions verify each method returns non-nil correctly + handles the nil/empty edge case. This is the load-bearing multi-adapter readiness proof.

**Test scenarios (happy + edge):**
- MockAdapter `BuildCommand` produces deterministic argv.
- MockAdapter `ParseStreamEvent` rejects malformed JSON with wrapped error.
- MockAdapter `ExtractTerminalReport` returns `(_, false)` for non-terminal events.
- Contract test: BOTH adapters in `[]CLIAdapter{mock, claude}` pass the same assertion suite.

**Falsification mitigations to bake in:**
- A8.a: this droplet IS the A8.a mitigation. The test exists in code, not just in spec.
- The contract test using `[]CLIAdapter` proves the seam is truly polymorphic — if `claudeAdapter` snuck a claude-specific assumption into the interface contract, the mock-side assertion would fail.

**Verification gates:** `mage check` + `mage ci` + `4c.F.7.17.4.qa-proof` + `4c.F.7.17.4.qa-falsification`.

**Out of scope:**
- Codex adapter (Drop 4d).
- Production-side mock for use by other packages (this is internal test fixture only).

---

### 4c.F.7.17.5 — Dispatcher wiring (`BuildSpawnCommand` rewrite)

**Goal:** rewrite `BuildSpawnCommand` to dispatch via `CLIKind` → adapter lookup. Default missing `cli_kind` resolves to `claude` for backward-compat. The 4a.19 stub argv-emission code is wholesale replaced.

**Builder model:** opus.

**Hard prereqs:** 4c.F.7.17.3 + 4c.F.7.17.4 merged.

**Files to edit/create:**
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/spawn.go` (REWRITE).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/spawn_test.go` (UPDATE — new assertions; existing 4a.19 tests retained where relevant for backward-compat).

**Packages locked:** `internal/app/dispatcher`.

**Acceptance criteria:**
- [ ] `BuildSpawnCommand` accepts an additional dependency: `adapters map[CLIKind]CLIAdapter` (constructor-injected at dispatcher-init time so tests substitute mocks).
- [ ] Dispatch sequence:
  1. `binding, ok := catalog.LookupAgentBinding(item.Kind)` (existing).
  2. `binding.Validate()` (existing defensive re-validate).
  3. `kind := ResolveCLIKind(string(binding.CLIKind))` — empty → `CLIKindClaude` (L15).
  4. `adapter, ok := adapters[kind]` — missing → `ErrUnknownCLIKind`.
  5. `resolved := ResolveBinding(binding, overrides)` (droplet 8 dependency; placeholder no-op resolver in this droplet that just copies fields).
  6. `paths := bundlePathsFor(item, project)` (placeholder; F.7.1 fills in).
  7. `cmd, err := adapter.BuildCommand(ctx, resolved, paths)`.
  8. Wrap `os/exec` "executable file not found in $PATH" errors with structured context naming the binding's TOML position (L22). The dispatcher-level wrap calls `errors.Is(err, exec.ErrNotFound)` and prepends a structured prefix `"dispatcher: command %q for kind %q not on $PATH"`.
- [ ] Backward-compat test: a binding with `cli_kind` omitted (zero-value) AND a binding with `cli_kind = "claude"` produce IDENTICAL `*exec.Cmd` argv + env.
- [ ] Forward-compat stub: `cli_kind = "codex"` resolves but `adapters["codex"]` is nil — error is `ErrCodexAdapterNotImplemented` (or wrapped `ErrUnknownCLIKind` with explanatory message). Test asserts the error message instructs the dev to wait for Drop 4d.
- [ ] Missing-env (`ErrMissingRequiredEnv` from droplet 3) bubbles up from `adapter.BuildCommand` BEFORE any lock acquisition — verified via test that asserts no `cmd` is returned (P5).
- [ ] `mcpConfigPlaceholderPath` removed from `spawn.go` (post-F.7 architecture uses `--plugin-dir`; the legacy placeholder path no longer applies).
- [ ] `assemblePrompt` continues to emit `task_id`, `project_id`, `project_dir`, `kind`, `title`, move-state directive. **`hylla_artifact_ref` is removed from the prompt body** (F.7.10 owns the broader removal — droplet 5 stops EMITTING it; F.7.10's broader sweep ensures no caller still pumps the field). Document the cross-droplet handoff in the commit message.
- [ ] `SpawnDescriptor.MCPConfigPath` field is removed (no longer applies post-rewrite). `SpawnDescriptor` gains `CLIKind` + `Command []string` + `Env []string` for monitor/log handoff.

**Test scenarios (happy + edge):**
- `cli_kind = ""` → claude adapter selected.
- `cli_kind = "claude"` → claude adapter selected; argv identical to omitted-cli_kind case.
- `cli_kind = "codex"` → `ErrCodexAdapterNotImplemented` returned.
- `cli_kind = "bogus"` → `ErrUnknownCLIKind`.
- `binding.Env = ["UNSET_VAR"]` → returns `ErrMissingRequiredEnv` early; no `*exec.Cmd` produced.
- `binding.Command = ["nonexistent-bin-xyz"]` AND that binary is NOT on `$PATH` → `BuildCommand` returns whatever `os/exec` returns (the dispatcher does NOT short-circuit). `BuildSpawnCommand` then wraps with structured prefix per L22 when `errors.Is(err, exec.ErrNotFound)`.
- Mock-injection test: `adapters[CLIKindClaude] = newMockAdapter(...)` — `BuildSpawnCommand` returns the mock's `*exec.Cmd` verbatim.

**Falsification mitigations to bake in:**
- L15 backward-compat assertion test.
- P5 early-pre-lock failure test.
- L22 structured `exec.ErrNotFound` wrap.
- Drop 4d additivity proof: the codex error message is the only place the dispatcher needs to change when Drop 4d lands (it'll just register the codex adapter into the `adapters` map).

**Verification gates:** `mage check` + `mage ci` + `4c.F.7.17.5.qa-proof` + `4c.F.7.17.5.qa-falsification`.

**Out of scope:**
- Lock acquisition (sits between `BuildSpawnCommand` and `cmd.Start()` — owned by 4a.20 / F.7.1).
- Bundle materialization — F.7.1.
- The codex adapter itself — Drop 4d.

---

### 4c.F.7.17.6 — `manifest.json` `cli_kind` field + orphan-scan adapter routing

**Goal:** the per-spawn `manifest.json` (F.7.1) gains `cli_kind` so the orphan-scan path (F.7.8) routes liveness checks via the recorded adapter.

**Builder model:** opus.

**Hard prereqs:** 4c.F.7.17.5 merged. Cross-planner: F.7.1 (other planner) defines the `manifest.json` schema struct. This droplet WIDENS that struct and consumes it in orphan-scan.

**Files to edit/create:**
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/manifest.go` (file owned by F.7.1; this droplet adds the `CLIKind` field — coordinate via cross-planner handoff in PLAN-synthesis).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/manifest_test.go` (UPDATE — round-trip test for `cli_kind`).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/orphan_scan.go` (file owned by F.7.8; this droplet adds adapter-routing logic — same cross-planner note).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/orphan_scan_test.go` (UPDATE).

**Packages locked:** `internal/app/dispatcher`.

**Acceptance criteria:**
- [ ] `manifest.json` Go struct `Manifest` (F.7.1's name) gains `CLIKind CLIKind` field with JSON tag `cli_kind`.
- [ ] At spawn-write time (F.7.1's responsibility): `Manifest.CLIKind = ResolveCLIKind(binding.CLIKind)`. This droplet wires the assignment.
- [ ] Orphan-scan reads `Manifest.CLIKind` and looks up the adapter. For `CLIKindClaude`, runs the existing PID liveness check (F.7.8's claude path). For `CLIKindCodex`, returns `ErrCodexAdapterNotImplemented` (the orphan stays as-is for Drop 4d to handle).
- [ ] Manifest round-trip test: write a manifest with `CLIKind = CLIKindClaude` → read back → field round-trips.
- [ ] Old manifests (pre-F.7.17.6 schema, no `cli_kind` key) decode with `CLIKind = ""` → `ResolveCLIKind` defaults to `CLIKindClaude`. Backward-compat assertion test.
- [ ] Orphan-scan returns explicit `ErrCodexAdapterNotImplemented` (not silent skip) so dev sees the gap if a stale codex orphan ever lands during Drop 4c.

**Test scenarios (happy + edge):**
- Write+read claude manifest.
- Read manifest missing `cli_kind` (legacy) → defaults to claude.
- Orphan-scan with claude PID alive → leaves item alone.
- Orphan-scan with claude PID dead → moves item to `failed` with `metadata.failure_reason = "dispatcher_restart_orphan"` (F.7.8's existing semantic, unchanged).
- Orphan-scan with codex manifest → returns `ErrCodexAdapterNotImplemented`.

**Falsification mitigations to bake in:**
- Backward-compat with legacy manifests via L15.
- P13 §6.1: explicit cli_kind surfacing prevents adapter-mismatch silent failures.

**Verification gates:** `mage check` + `mage ci` + `4c.F.7.17.6.qa-proof` + `4c.F.7.17.6.qa-falsification`.

**Cross-planner coordination note:**
F.7.1 owns `manifest.go` creation; F.7.8 owns `orphan_scan.go` creation. This droplet ASSUMES both files exist post-F.7.1/F.7.8 and ADDS the `cli_kind` field + routing. If F.7.1/F.7.8 land in the same drop alongside this droplet, the `blocked_by` chain is enforced by the droplet's hard-prereqs line. PLAN.md synthesis (orchestrator-level) MUST verify the cross-planner ordering.

**Out of scope:**
- The actual codex liveness check (Drop 4d).
- Bundle materialization (F.7.1).

---

### 4c.F.7.17.7 — `permission_grants` schema gets `cli_kind` column

**Goal:** SQLite `permission_grants` schema gains `cli_kind TEXT NOT NULL`. Pre-MVP rule: NO migration logic; dev fresh-DBs.

**Builder model:** opus.

**Hard prereqs:** 4c.F.7.17.5 merged.

**Files to edit/create:**
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/adapters/storage/sqlite/schema.sql` (file may already exist; ADD `cli_kind` column to `permission_grants` table). Path TBD by builder — discover via `mage check` artifact + `find internal/adapters/storage/sqlite -name "*.sql"` at build time.
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/adapters/storage/sqlite/permission_grants.go` (or wherever F.7.5 lands the Go-side write/read logic; ADD `cli_kind` to the SQL INSERT + SELECT).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/adapters/storage/sqlite/permission_grants_test.go` (NEW).

**Packages locked:** `internal/adapters/storage/sqlite`.

**Acceptance criteria:**
- [ ] `permission_grants` table schema includes `cli_kind TEXT NOT NULL` (no DEFAULT — every row must declare; absence is a load-time TOML / dispatcher bug).
- [ ] INSERT path: `INSERT INTO permission_grants(project_id, kind, rule, granted_by, granted_at, cli_kind) VALUES (?, ?, ?, ?, ?, ?)`.
- [ ] SELECT path: query honors `cli_kind` filter so a claude-authored grant doesn't leak into a codex spawn (or vice versa). Default behavior in Drop 4c: every grant is `cli_kind = "claude"`; the discriminator is plumbed end-to-end but only one value populates it.
- [ ] Test: insert claude grant → read with `cli_kind = "claude"` filter → returns row. Read with `cli_kind = "codex"` filter → returns no rows.
- [ ] Test: empty `cli_kind` insert → fails (NOT NULL).
- [ ] Doc-comment on the `cli_kind` column citing P13 §6.4 + this plan path.
- [ ] **Pre-MVP rule callout in commit message + droplet acceptance:** "Schema change is dev-fresh-DB; no migration code per `feedback_no_migration_logic_pre_mvp.md`. Dev deletes `~/.tillsyn/tillsyn.db` before next launch."

**Test scenarios (happy + edge):**
- Insert claude grant → read back with claude filter.
- Insert claude grant → read with codex filter returns empty.
- Insert with empty `cli_kind` fails NOT NULL.
- Read on a fresh DB returns zero rows (no migration; dev deleted DB).

**Falsification mitigations to bake in:**
- P13 §6.4: schema discriminator prevents cross-CLI grant pollution.
- Pre-MVP rule explicit: no migration code; the droplet does not introduce schema-version tracking, no `ALTER TABLE`, no migration runner.

**Verification gates:** `mage check` + `mage ci` + `4c.F.7.17.7.qa-proof` + `4c.F.7.17.7.qa-falsification`.

**Out of scope:**
- The TUI handshake itself (F.7.5).
- Codex grant authoring path (Drop 4d).
- Schema migration tooling (post-MVP).

---

### 4c.F.7.17.8 — `BindingResolved` priority-cascade resolver

**Goal:** pure function `ResolveBinding(rawBinding templates.AgentBinding, overrides Overrides) BindingResolved` implements `CLI flag > MCP arg > TUI override > template TOML > absent` (L16).

**Builder model:** opus.

**Hard prereqs:** 4c.F.7.17.1 merged (per-binding fields exist) + 4c.F.7.17.2 merged (`BindingResolved` type exists).

**Files to edit/create:**
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/binding_resolver.go` (NEW).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/binding_resolver_test.go` (NEW).

**Packages locked:** `internal/app/dispatcher`.

**Acceptance criteria:**
- [ ] `Overrides` struct with optional override-source fields (`CLIBudget *float64, MCPBudget *float64, TUIBudget *float64`, etc. — one trio per binding field that's overridable). Pointer-typed; nil = "not supplied at this level."
- [ ] `ResolveBinding(rawBinding, overrides)` returns `BindingResolved` with each field resolved via the cascade. The first non-nil pointer in the priority order wins; if all pointers are nil, the field comes from `rawBinding` (template TOML); if `rawBinding`'s field is also zero-value, the resolved field is the type's zero value (which adapters then interpret as "absent").
- [ ] Pure function, no I/O, no logging side effects (logs go through the dispatcher, not the resolver).
- [ ] Resolver populates `BindingResolved.CLIKind` via `ResolveCLIKind(rawBinding.CLIKind)` (default `CLIKindClaude` per L15).
- [ ] Resolver populates `BindingResolved.Command` from `rawBinding.Command`. When nil/empty, sets a sentinel for the dispatcher to default to `[]string{"claude"}` at adapter-call time. (The default is centralized in droplet 5, not here.)
- [ ] Table-driven test covering every priority level for at least three fields (`MaxBudgetUSD`, `MaxTurns`, `Model`).
- [ ] Doc-comment cites L16 + the SKETCH F.7.3 priority cascade.

**Test scenarios (happy + edge):**
- All overrides nil + template has `max_budget_usd = 5.0` → resolved = 5.0.
- TUI override = 3.0 + template = 5.0 → resolved = 3.0.
- MCP override = 2.0 + TUI = 3.0 + template = 5.0 → resolved = 2.0.
- CLI override = 1.0 + MCP = 2.0 + TUI = 3.0 + template = 5.0 → resolved = 1.0.
- All nil + template field also zero → resolved field is zero (adapter interprets as "absent" — flag NOT emitted).
- `cli_kind` resolution: empty → claude; "codex" → codex.

**Falsification mitigations to bake in:**
- L16: cascade is pure-function + tested at every level.
- Pointer-types preserve "absent" vs "explicitly zero" distinction so adapters can decide whether to emit a flag.

**Verification gates:** `mage check` + `mage ci` + `4c.F.7.17.8.qa-proof` + `4c.F.7.17.8.qa-falsification`.

**Out of scope:**
- Source-of-overrides plumbing (CLI flags / MCP args / TUI surfaces) — post-MVP.
- Validation of resolved values — adapters defensive-validate at `BuildCommand`.

---

### 4c.F.7.17.9 — CLI-agnostic monitor refactor — **MERGED INTO F.7-CORE F.7.4 per REVISIONS REV-7**

**STATUS: MERGED.** F.7-CORE F.7.4 already builds the CLI-agnostic monitor (per F.7-CORE plan acceptance line 356 — monitor consumes via `adapter.ParseStreamEvent` from inception, not as a refactor). This droplet is redundant. Builders MUST NOT implement this droplet as a separate unit. F.7-CORE F.7.4 absorbs every acceptance criterion below into its own acceptance.

(Original body kept for reference only; do not implement.)

**Goal (HISTORICAL):** F.7.4's stream-jsonl monitor (originally planned to assume claude taxonomy) refactored to consume via `adapter.ParseStreamEvent` + `adapter.ExtractTerminalReport`. Dispatcher monitor stays CLI-agnostic.

**Builder model:** opus.

**Hard prereqs:** 4c.F.7.17.5 merged. Cross-planner: F.7.4 (other planner) authors the monitor; this droplet WIDENS its consumption pattern.

**Files to edit/create:**
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/monitor.go` (file owned by F.7.4; this droplet refactors event-parsing path to call `adapter.ParseStreamEvent` rather than inline claude logic).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/monitor_test.go` (UPDATE — assertions against MockAdapter prove CLI-agnosticism).

**Packages locked:** `internal/app/dispatcher`.

**Acceptance criteria:**
- [ ] Monitor accepts an `adapter CLIAdapter` parameter at start time (`Monitor.Start(ctx, descriptor SpawnDescriptor, adapter CLIAdapter)` or equivalent).
- [ ] Monitor's per-line loop: `ev, err := adapter.ParseStreamEvent(line)`; if `ev.IsTerminal`, call `adapter.ExtractTerminalReport(ev)` and route per existing F.7.4 logic (write `metadata.actual_cost_usd`, surface `permission_denials[]` to F.7.5 handshake, etc.).
- [ ] Monitor has ZERO references to claude-specific event types (no `"system/init"`, `"assistant"`, `"result"` literals in monitor code — all routing is via `StreamEvent.Type` and `StreamEvent.IsTerminal`).
- [ ] Test using MockAdapter: monitor processes mock-emitted lines correctly (proves polymorphism).
- [ ] Test using claudeAdapter: monitor processes recorded claude jsonl trace correctly (proves no regression vs F.7.4 baseline).

**Test scenarios (happy + edge):**
- Monitor + MockAdapter: 3-line trace processes; terminal event triggers cost capture.
- Monitor + claudeAdapter: recorded `testdata/claude_stream_minimal.jsonl` processes identically to pre-refactor F.7.4 baseline.
- Monitor handles malformed JSON line: `ParseStreamEvent` error logged but loop continues (existing F.7.4 semantic).
- Monitor handles spurious empty lines: skipped without error.

**Falsification mitigations to bake in:**
- P13 §6.3: monitor never holds claude-specific assumptions in code.
- Polymorphism proof via MockAdapter test.

**Verification gates:** `mage check` + `mage ci` + `4c.F.7.17.9.qa-proof` + `4c.F.7.17.9.qa-falsification`.

**Cross-planner coordination note:**
F.7.4 (other planner) ships the monitor. This droplet's sequencing assumes F.7.4 lands first in the F.7 wave OR within the same drop; PLAN.md synthesis at the orchestrator level MUST verify ordering. If F.7.4 hasn't landed when this droplet is dispatched, the droplet is BLOCKED.

**Out of scope:**
- F.7.5 TUI handshake (consumes monitor output but lives separately).
- Real-time mid-stream `tool_result is_error: true` parsing (optional Drop 4c+ per SKETCH).

---

### 4c.F.7.17.10 — Marketplace install-time confirmation (paper-spec) — **REMOVED per REVISIONS REV-4**

**STATUS: REMOVED.** This droplet is superseded by REVISIONS POST-AUTHORING REV-4 below. The `command` and `args_prefix` fields were dropped from the design; without `command[0]` to confirm at install time, this droplet has no scope. Builders MUST NOT implement this droplet. Acceptance criteria, test scenarios, and other body text below are NULL — kept only as a marker so historical references resolve.

(Original body deleted; refer to REVISIONS REV-4 for context.)

---

### 4c.F.7.17.11 — Adapter-authoring documentation (renumbered to 4c.F.7.17.10 per REVISIONS REV-1)

**Goal:** companion to F.7.11. One MD droplet documenting "how to add a new CLI adapter to Tillsyn" so future-adapter contributors (Drop 4d codex authors, post-MVP cursor/goose/aider authors) have a clear contract.

**Builder model:** opus.

**Hard prereqs:** 4c.F.7.17.4 merged (MockAdapter exists as the canonical reference).

**Files to edit/create:**
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/docs/architecture/cli_adapter_authoring.md` (NEW).
- (NO Go code.)

**Packages locked:** none (MD-only).

**Acceptance criteria:**
- [ ] Doc covers:
  1. **CLI-shape invariants** (P3): process-per-spawn, exit-code authoritative, stderr is NOT the event channel, JSONL stream taxonomy. CLIs violating any property need a different adapter family, not a wider `CLIAdapter` interface.
  2. **Three-method contract** (L10): exact signatures + per-method semantic.
  3. **`BindingResolved` consumption** (L16): adapters do NOT re-resolve; they consume the resolved struct.
  4. **`BundlePaths` materialization** (L13): adapters compute their CLI-specific subdir under `paths.Root` themselves.
  5. **Closed env baseline** (L6, L7, L8): every adapter's `BuildCommand` MUST construct `cmd.Env` explicitly; `os.Environ()` MUST NOT be inherited.
  6. **`TerminalReport` shape** (L11): adapters return pointer-cost `nil` when their CLI doesn't emit a cost channel.
  7. **The hard-cut migration story for non-JSONL CLIs** (L12, A4.a): future SSE / framed-binary CLI lands via a coordinated breaking interface rewrite — NOT add-then-deprecate. Document the exact migration sequence: rewrite interface, refactor all adapters + monitor in one drop, no compat shim.
  8. **MockAdapter as reference** (L17): point new adapter authors at `cli_adapter_test.go` for the canonical contract conformance test.
  9. **Test fixtures pattern**: every new adapter ships at least one recorded `testdata/<adapter>_stream_minimal.jsonl` + a contract conformance test entry.
- [ ] Doc cross-references F.7.11 (Tillsyn architecture overview), this plan, and `project_drop_4c_spawn_architecture.md` memory.
- [ ] Doc is concise (≤ 400 lines); the source of truth is the code, not the doc.

**Test scenarios:**
- (No code tests — MD-only droplet.)
- Doc-review acceptance: QA reads the doc against the actual `cli_adapter.go` + `cli_adapter_claude.go` and verifies no claim drifts from code.

**Falsification mitigations to bake in:**
- A4.a: hard-cut migration framing documented; future contributors don't believe the migration is "just additive."
- L11: nil-cost path documented as first-class.
- P3: CLI-shape invariants documented as a checklist.

**Verification gates:** `mage check` + `4c.F.7.17.11.qa-proof` + `4c.F.7.17.11.qa-falsification`.

**Out of scope:**
- Codex-specific footguns (Drop 4d's responsibility to extend this doc).
- Post-MVP CLIs.

---

## Open Questions for Plan-QA Twins

- **Q1 — Cross-planner coordination on `manifest.json`.** Droplet 6 widens a struct that F.7.1 (the OTHER planner) creates. PLAN-synthesis at the orchestrator level MUST resolve which planner authors `manifest.go` first AND ensure the field-add lands in the right droplet's commit. If F.7.1's manifest droplet ships pre-Drop-4c (already merged), this droplet's hard-prereqs line is satisfied trivially; if F.7.1's manifest droplet is in-flight in the same drop, sequencing is critical. Recommendation: orchestrator-level dependency manifest in unified PLAN.md.

- **Q2 — `--mcp-config` flag retirement.** Droplet 3 drops `--mcp-config` from claude argv (post-F.7 architecture uses `--plugin-dir <bundle>/plugin`). The 4a.19 stub emits it AND populates `mcpConfigPlaceholderPath`. Confirm with QA-falsification that nothing else in the dispatcher reads `SpawnDescriptor.MCPConfigPath` — if it does, that consumer is broken by this droplet and needs an update in the same droplet.

- **Q3 — `BindingResolved.Command` defaulting.** Droplet 8 leaves `Command` as-is when `rawBinding.Command` is empty; droplet 5 supplies the default `[]string{"claude"}` at adapter-call time. Alternative: droplet 8 supplies the default; droplet 5 just consumes. Centralized default logic is cleaner; rejected here because mixing resolution with defaults makes the resolver impure (it'd need to know `claude` is the canonical default, which is a CLIKind concern). **Open for plan-QA debate**: if QA flags this as overly-spread, collapse the default into droplet 8 instead.

- **Q4 — `permission_grants.cli_kind` enforcement granularity.** Droplet 7 makes `cli_kind` NOT NULL. Alternative: nullable column with a CHECK constraint that empty-string is forbidden. NOT NULL is stricter; chose stricter per L19's spirit. Confirm with QA-falsification that no path inserts an empty value.

- **Q5 — Recorded fixture format for MockAdapter (droplet 4).** The test fixture file `testdata/mock_stream_minimal.jsonl` defines the mock's wire format. We propose `{"type":"mock_terminal","cost":0.5,"denials":[]}` as the simplest possible terminal event. Confirm with QA-falsification that this is sufficient to exercise the contract (cost, denials, terminal flag) AND that adding more event types in droplet 4 would be over-engineering for "test fixture only."

- **Q6 — Cross-droplet `assemblePrompt` ownership of `hylla_artifact_ref` removal.** F.7.10 (other planner) owns the broader removal sweep. Droplet 5 stops EMITTING `hylla_artifact_ref` from `assemblePrompt`. If F.7.10 is in-flight or post-Drop-4c, dropletQ 5's commit is the FIRST place `hylla_artifact_ref` stops appearing in spawn prompts. Confirm sequencing in unified PLAN.md.

---

## REVISIONS POST-AUTHORING (2026-05-05) — supersedes affected portions above

The dev approved two architectural changes after this sub-plan was authored. **Where this section conflicts with text above, this section wins.** Builders read this section first.

### REV-1 — `command` and `args_prefix` fields DROPPED entirely

The `Command []string` and `ArgsPrefix []string` fields are NOT added to `AgentBinding`. The wrapper-interop knob is GONE from Tillsyn. Adapters invoke their CLI binary directly (`claude` for claude adapter, `codex` for codex adapter); adopters who want process isolation use OS-level mechanisms.

Concrete supersessions:

- **L1, L2, L3 (lines 50-52)**: superseded — `Command` / `ArgsPrefix` no longer in scope.
- **L5 (per-token regex)**: superseded — no regex needed without `command` field.
- **L6 (shell-interpreter denylist)**: superseded — no denylist needed.
- **F.7.17.1 (Schema-1) acceptance criteria**: now adds ONLY `Env []string` + `CLIKind string` to `AgentBinding`. Does NOT add `Command` or `ArgsPrefix`. The `validateAgentBindingCommandTokens` validator + `shellInterpreterDenylist` constant + 12 command-validation test scenarios are GONE from this droplet. Only `validateAgentBindingEnvNames` validator remains.
- **F.7.17.10 (marketplace install paper-spec droplet)**: REMOVED entirely. Without `command[0]`, no install-time argv-list to confirm. F.7.17 droplet count: 11 → 10.
- **F.7.17.11 (adapter-authoring docs)**: renumbered to F.7.17.10. Still ships.
- **`BindingResolved.Command` and `BindingResolved.ArgsPrefix`** in F.7.17.2 + F.7.17.8: REMOVED. `BindingResolved` now carries `Env []string` + `CLIKind string` (plus existing fields like `Model`, `MaxBudgetUSD`, `MaxTurns`, etc.). Q3 (BindingResolved.Command defaulting) is moot.
- **F.7.17.5 dispatcher wiring**: adapter is selected by `CLIKind`; the adapter's `BuildCommand` hardcodes its CLI binary name internally. No "command override" path.

### REV-2 — L4 closed env baseline expanded

L4 baseline now includes proxy + TLS-cert vars: `HTTP_PROXY, HTTPS_PROXY, NO_PROXY, http_proxy, https_proxy, no_proxy, SSL_CERT_FILE, SSL_CERT_DIR, CURL_CA_BUNDLE` PLUS the prior process-basics list (`PATH, HOME, USER, LANG, LC_ALL, TZ, TMPDIR, XDG_CONFIG_HOME, XDG_CACHE_HOME`). Per-binding `env` allow-list still ADDS to baseline.

Affected droplets: F.7.17.3 (`claudeAdapter`) sets `cmd.Env` to the expanded baseline + binding's resolved `env` values. Acceptance test: corporate-network adopter with `https_proxy` in shell env (NOT declared in any binding's `env` field) sees the spawn inherit `https_proxy` correctly.

### REV-3 — `Manifest.CLIKind` ownership: F.7.17.6 SOLE OWNER

F.7-CORE F.7.1 ships `Manifest` WITHOUT a `CLIKind` field. F.7.17.6 is the sole droplet that adds it. Sequencing: F.7.17.6 must land BEFORE F.7-CORE F.7.8 (orphan scan) since orphan scan reads `manifest.CLIKind` to route adapter liveness checks. This is added to the master PLAN §5 DAG as `F.7-CORE F.7.8 blocked_by F.7.17.6`.

### REV-4 — Marketplace install confirmation: SUPERSEDED

The "marketplace install-time interactive confirmation" surface (F.7.17.10 in original sub-plan, L7 in original locked-decisions) is GONE. Without `command[0]` to confirm, the trust boundary is just regular template-load validation (`validateAgentBindingEnvNames` + closed-struct strict-decode). Marketplace concerns reduce to "the marketplace template's `[agent_bindings]` declarations are dev-validated at install time via the F.4 marketplace CLI's normal display flow" — but Drop 4c does NOT ship F.4 (it's deferred to Drop 4d-prime), so this is post-Drop-4c work entirely.

### REV-5 — `ExtractTerminalReport` rename

Method renamed from `ExtractTerminalCost` to `ExtractTerminalReport` per round-1 falsification A4.b. F.7.17.2 + F.7.17.3 ship `ExtractTerminalReport(StreamEvent) (TerminalReport, bool)`.

### REV-9 — `BindingOverrides` layered cascade shape (vs original per-source struct)

F.7.17.8 builder shipped a layered cascade via `ResolveBinding(rawBinding, overrides ...*BindingOverrides)` where each override layer is a `*BindingOverrides` with 8 pointer fields, walked highest-first. Original plan body (lines 540-541) specified per-source-tagged pointers (e.g. `CLIBudget`, `MCPBudget`, `TUIBudget`). Layered cascade is functionally equivalent + simpler + extensible — caller constructs N layers in priority order rather than coupling field names to source enums.

Future override-source wiring (CLI / MCP / TUI surfaces) constructs a `*BindingOverrides` per source and passes them in priority order to `ResolveBinding`. Drop 4d codex adapter slot, future Cursor/Goose adapters: same shape applies; sources remain orthogonal to the resolver.

### REV-7 — F.7.17.9 MERGED into F.7-CORE F.7.4

F.7.17.9 (CLI-agnostic monitor refactor) is **REMOVED** as a separate droplet. F.7-CORE F.7.4 already builds the CLI-agnostic monitor from inception (its acceptance criteria specify `adapter.ParseStreamEvent` consumption, not inline claude logic). Refactoring something that's already CLI-agnostic adds no value.

F.7.17 droplet count after REV-1 + REV-7: 11 → 9 (removed F.7.17.10 marketplace + F.7.17.9 monitor refactor; renumbered F.7.17.11 → F.7.17.9 adapter docs).

### REV-8 — Procedural rule: builder spawn prompts MUST instruct REVISIONS-FIRST reading

Per Falsification R3 N1: every F.7-touching builder spawn prompt MUST include a directive: *"Before reading the body of this sub-plan, read the REVISIONS POST-AUTHORING section at the bottom — it supersedes any conflicting body text."* Without this directive, a builder reading top-down may implement to the contradicted body before discovering the supersession.

### REV-6 — Adapter-authoring docs (renumbered F.7.17.10 per REV-1, then F.7.17.9 per REV-7) MUST cover

- "How to add a new CLI adapter to Tillsyn" — interface contract, fixture pattern, MockAdapter example, registration into the adapter map.
- **Security model documentation** (NEW): "Tillsyn trusts the user's `$PATH` to resolve `claude` / `codex` binaries. Adopters who want hardened binary resolution set up their own PATH-shadowed shim hierarchy outside Tillsyn (PATH-shadowed binary, container wrapping the entire Tillsyn binary, sandbox-exec). Tillsyn does NOT surface a `command` override field — process isolation is an OS-level concern."
- **Vendored-binary pattern** (NEW): "A project that ships `./vendored/claude` for reproducibility prepends `<project>/vendored` to `PATH` before launching `till dispatcher run`. Tillsyn's spawn pipeline inherits PATH (per L4) and resolves `claude` to the vendored copy."
- "Hard-cut migration" — when the first non-JSONL CLI lands, ALL adapters + dispatcher monitor refactored in one drop. No backward-compat shim. Documented in this droplet so future contributors know the rule.

---

## References

- `workflow/drop_4c/SKETCH.md` — F.7.17 spec (lines 147-180), three-schema split (lines 232-242), sequencing (line 175), pre-MVP rules (lines 257-263).
- `workflow/drop_4c/4c_F7_EXT_PLANNER_REVIEW.md` — P1-P13 plan recommendations + §6 cross-section retro-edits.
- `workflow/drop_4c/4c_F7_EXT_QA_FALSIFICATION_R2.md` — A1.a (sh -c bypass + denylist), A1.c (regex literal pinning), A2.a-d (env baseline + lowercase), A3.a-c (schema split), A4.a (hard-cut migration), A4.b (rename `ExtractTerminalReport`), A8.a (MockAdapter fixture).
- `~/.claude/projects/-Users-evanschultz-Documents-Code-hylla-tillsyn/memory/project_drop_4c_spawn_architecture.md` — canonical architecture; §3 (priority cascade), §6 (event taxonomy), §11 (non-goals).
- `internal/templates/schema.go` lines 285-332 — current `AgentBinding` struct (extension target for droplet 1).
- `internal/templates/load.go` lines 80-95 — strict-decode chain (`DisallowUnknownFields`); validators wired here in droplet 1.
- `internal/app/dispatcher/spawn.go` lines 106-166 — 4a.19 `BuildSpawnCommand` stub (replacement target for droplet 5); lines 177-179 (`mcpConfigPlaceholderPath` retired in droplet 5).
- `~/.claude/projects/-Users-evanschultz-Documents-Code-hylla-tillsyn/memory/feedback_no_migration_logic_pre_mvp.md` — pre-MVP rule referenced by droplet 7.
- `~/.claude/projects/-Users-evanschultz-Documents-Code-hylla-tillsyn/memory/feedback_orchestrator_no_build.md` — orchestrator-never-edits-Go rule; this plan respects by emitting only droplet specs.
