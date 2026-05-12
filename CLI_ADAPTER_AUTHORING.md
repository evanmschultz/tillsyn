# Authoring a New CLI Adapter

How to add support for a new headless CLI (codex, cursor-agent, goose, aider, …) to Tillsyn's spawn pipeline. Companion to `SPAWN_PIPELINE.md`.

## Adapter Contract

A CLI adapter implements `internal/app/dispatcher.CLIAdapter`:

```go
type CLIAdapter interface {
    BuildCommand(ctx context.Context, binding BindingResolved, paths BundlePaths) (*exec.Cmd, error)
    ParseStreamEvent(line []byte) (StreamEvent, error)
    ExtractTerminalReport(ev StreamEvent) (TerminalReport, bool)
}
```

Drop 4c ships `claude`. Drop 4d will ship `codex`. Both are line-delimited JSON (JSONL) on stdout.

## Required CLI Properties

Today's `CLIAdapter` interface assumes the wrapped CLI satisfies:

1. **Process-per-spawn.** Each invocation is a fresh `*exec.Cmd` with its own bundle. No daemon mode.
2. **Exit-code authoritative.** Exit 0 = success; non-zero = failure surfaced to caller.
3. **Stdout is the event channel.** Stderr is logs / diagnostics, not events.
4. **Newline-delimited JSON events.** One event per line on stdout. Adapters that emit SSE / WebSocket / framed-binary events do NOT fit today's interface — see "Non-JSONL Extensibility" below.

CLIs that violate any of these need a different adapter family, not a wider `CLIAdapter` interface.

## Step-by-Step

### 1. Create the package

Mirror `internal/app/dispatcher/cli_claude/` layout:

```
internal/app/dispatcher/cli_<name>/
├── adapter.go     # adapter struct, New() constructor, CLIAdapter methods
├── argv.go        # pure assembleArgv(BindingResolved, BundlePaths) []string
├── env.go         # pure assembleEnv(BindingResolved) ([]string, error)
├── stream.go      # parseStreamEvent + extractTerminalReport
├── adapter_test.go
├── init.go        # dispatcher.RegisterAdapter(...) at import time
└── testdata/
    └── <name>_stream_minimal.jsonl
```

The package imports `dispatcher` for the interface + value-object types but NOT vice-versa — `dispatcher.RegisterAdapter` + `dispatcher.lookupAdapter` form the registry seam that breaks the import cycle.

### 2. Hardcode the binary name

The adapter calls its CLI binary directly. **No `command` override field exists in the schema.** Adopters who want process isolation use OS-level wrappers (PATH-shadowed binary, container, sandbox-exec) — Tillsyn names no specific wrapper.

```go
const claudeBinaryName = "claude"  // or "codex", "cursor-agent", ...
```

### 3. Implement BuildCommand

Build the headless argv per the CLI's documented flags. Use conditional emission via `*int` / `*float64` / `*string` pointer fields on `BindingResolved` so flags are emitted only when explicitly set (see `internal/app/dispatcher/cli_claude/argv.go` for the canonical pattern).

Return an `*exec.Cmd` with:

- `Path` set via `exec.CommandContext(ctx, binaryName, ...)` (uses `LookPath` against `cmd.Env`'s PATH).
- `Args` matching the CLI's argv recipe.
- `Env` set EXPLICITLY (do NOT inherit `os.Environ()`):

```go
// Closed POSIX baseline: process basics + network conventions.
// `os.Environ()` is NOT inherited.
baseline := []string{
    "PATH="+os.Getenv("PATH"),  // inherit-PATH so the binary resolves
    "HOME="+os.Getenv("HOME"),
    "USER="+os.Getenv("USER"),
    "LANG="+os.Getenv("LANG"),
    "LC_ALL="+os.Getenv("LC_ALL"),
    "TZ="+os.Getenv("TZ"),
    "TMPDIR="+os.Getenv("TMPDIR"),
    "XDG_CONFIG_HOME="+os.Getenv("XDG_CONFIG_HOME"),
    "XDG_CACHE_HOME="+os.Getenv("XDG_CACHE_HOME"),
    // Network conventions (corporate adopters):
    "HTTP_PROXY="+os.Getenv("HTTP_PROXY"),
    "HTTPS_PROXY="+os.Getenv("HTTPS_PROXY"),
    "NO_PROXY="+os.Getenv("NO_PROXY"),
    "http_proxy="+os.Getenv("http_proxy"),
    "https_proxy="+os.Getenv("https_proxy"),
    "no_proxy="+os.Getenv("no_proxy"),
    "SSL_CERT_FILE="+os.Getenv("SSL_CERT_FILE"),
    "SSL_CERT_DIR="+os.Getenv("SSL_CERT_DIR"),
    "CURL_CA_BUNDLE="+os.Getenv("CURL_CA_BUNDLE"),
}
// Plus: each name in binding.Env, resolved via os.Getenv. Fail loud on missing.
```

Filter out unset names so the spawn doesn't see empty-string env vars where the orchestrator had nothing.

### 4. Implement ParseStreamEvent

Map per-line bytes to the canonical `StreamEvent`:

```go
type StreamEvent struct {
    Type       string          // canonical key: "system_init", "assistant", "user", "result"
    Subtype    string          // optional refinement
    IsTerminal bool             // true for the final event
    Text       string          // final agent text or content
    ToolName   string           // when content is tool_use
    ToolInput  json.RawMessage  // raw tool input args
    Raw        json.RawMessage  // full raw event for forensic capture
}
```

Each adapter owns the per-CLI-specific decoding. For unknown event types, emit `Type: <verbatim>` with `IsTerminal: false` so unrecognized events pass through without halting the monitor.

### 5. Implement ExtractTerminalReport

Only fires when caller checks `ev.IsTerminal == true`. Parse the raw terminal event for:

```go
type TerminalReport struct {
    Cost     *float64       // pointer — adapters lacking cost telemetry pass nil
    Denials  []ToolDenial   // permission denials surfaced to TUI handshake
    Reason   string         // terminal reason ("completed", "max_turns", ...)
    Errors   []string       // error messages
}
```

Return `(zero, false)` for non-terminal events.

### 6. Register the adapter

In `init.go`:

```go
package cli_<name>

import "github.com/evanmschultz/tillsyn/internal/app/dispatcher"

func init() {
    dispatcher.RegisterAdapter(dispatcher.CLIKindNew, New())
}
```

Add the new `CLIKind<Name>` constant to `internal/app/dispatcher/cli_adapter.go` and update `IsValidCLIKind`.

### 7. Wire the blank import

In `cmd/till/main.go`:

```go
import (
    _ "github.com/evanmschultz/tillsyn/internal/app/dispatcher/cli_claude"
    _ "github.com/evanmschultz/tillsyn/internal/app/dispatcher/cli_<name>"  // NEW
)
```

The blank import triggers `init()` self-registration.

### 8. Add a MockAdapter contract test

The dispatcher's `MockAdapter` test fixture (`internal/app/dispatcher/mock_adapter_test.go`) exercises the `CLIAdapter` interface contract WITHOUT touching real CLI binaries. Extend the table-driven contract test to include your new adapter so multi-adapter readiness is verified at compile + test time.

### 9. Test fixtures

Record real CLI output to `testdata/<name>_stream_minimal.jsonl` so `ParseStreamEvent` regression tests have ground truth. At minimum: one `system_init`-style event, one assistant turn, one terminal event with cost + (optionally) a denial.

## Security Model

Tillsyn trusts the user's `$PATH` to resolve the binary. Adopters who want hardened binary resolution:

- Set up a PATH-shadowed shim hierarchy outside Tillsyn (e.g. `~/.local/bin/claude` symlinked to a vendored or wrapper script).
- Wrap the entire Tillsyn binary in a container (Docker / Firejail / sandbox-exec).
- Use `direnv`-managed PATH per worktree.

Tillsyn does NOT surface a `command` override field — process isolation is an OS-level concern, not a Tillsyn concern.

### Vendored-Binary Pattern

A project that ships `./vendored/claude` for reproducibility prepends `<project>/vendored` to `PATH` before launching `till dispatcher run`. Tillsyn's spawn pipeline inherits PATH (via the closed baseline's `PATH=os.Getenv("PATH")`) and resolves `claude` to the vendored copy.

## Non-JSONL Extensibility (Future)

Today's interface assumes line-delimited JSON. Future CLIs that emit SSE / WebSocket / framed-binary events require a coordinated breaking change to the `CLIAdapter` interface (per `feedback_orphan_via_collapse_defer_refinement.md`):

1. Replace `ParseStreamEvent(line []byte) (StreamEvent, error)` with `ConsumeStream(ctx context.Context, reader io.Reader, sink chan<- StreamEvent) error`.
2. Refactor every existing adapter (`claude`, `codex`, future) to implement `ConsumeStream` by looping `bufio.Scanner` over the reader internally.
3. Refactor the dispatcher monitor to consume via channel sink.

This is a hard-cut interface rewrite — no backward-compat shim, no add-then-deprecate. Pre-MVP rule "no tech debt; if legacy isn't right, kill it" applies. Per-non-JSONL-CLI cost: ~5-8 droplets per adapter + ~2-3 droplets for the upfront refactor.

## Permission Handshake Compatibility

Future CLIs may use different permission-denial event shapes. The `TerminalReport.Denials []ToolDenial` is the cross-CLI canonical shape. Each adapter's `ExtractTerminalReport` is responsible for mapping the CLI's native denial structure into `[]ToolDenial{ToolName, ToolInput}`.

The Tillsyn `permission_grants` SQLite table includes a `cli_kind` column so a grant authored against one adapter does NOT apply to a different adapter's spawn (rule-syntax may differ between CLIs).

## Isolation Discipline for New Adapters

**`--bare`-collapsed isolation is a load-bearing contract, not a nicety.** Every adapter that wraps a Claude Code-compatible CLI MUST emit flags that enforce isolation equivalent to Tillsyn's `claude` adapter. The correct minimal shape:

```
<cli-binary>
  --bare
  --plugin-dir <bundle>/plugin
  --setting-sources ""
  --strict-mcp-config
  --settings  <bundle>/plugin/settings.json
  --mcp-config <bundle>/plugin/.mcp.json
```

**Why these four flags together:**

- `--bare` — skips all auto-discovery: plugins, agents, CLAUDE.md (project + user), skills, hooks, auto-memory. Under bare mode only explicitly passed flags take effect (Anthropic headless docs). Path B (system-installed plugins, `~/.claude/agents/`, `~/.claude/plugins/cache/`) is disabled entirely by this one flag.
- `--plugin-dir <bundle>/plugin` — opts the per-spawn bundle's plugin back in as the SOLE plugin source. Under `--bare`, this is the only plugin Claude Code loads.
- `--setting-sources ""` — excludes all three standard settings layers (user / project / local). Combined with `--settings <bundle>/plugin/settings.json`, the bundle is the sole permissions source.
- `--strict-mcp-config` — restricts MCP to `--mcp-config` argument only, ignoring project `.mcp.json` and user `~/.claude.json`.

**Common mistake for Drop 4d (codex) and beyond:** do NOT rely on agent-file priority tables to "win" over user-installed definitions. The priority table (managed settings → `--agents` flag → `.claude/agents/` → `~/.claude/agents/` → plugin) applies in non-bare mode. Under `--bare`, entries 3 and 4 are never consulted. Ship `--bare` and the bundle plugin; do not rely on priority-table ordering to enforce isolation.

**Bundle body must be substantive.** The bundle's `plugin/agents/<name>.md` is the ONLY agent definition Claude Code sees under `--bare`. If the body is empty or a one-liner stub, the agent runs with frontmatter only — no role definition, no tool discipline, no output format. Drop 4c.6 W3 wires embedded-default full agent content into every rendered bundle body and adds a post-render validator that rejects thin bodies at build time. Future adapters must apply equivalent logic: the bundle body is not a stub redirect, it IS the agent.

## Adapter Authoring Checklist

- [ ] Package created at `internal/app/dispatcher/cli_<name>/`.
- [ ] `adapter.go` with `New()` constructor + `var _ dispatcher.CLIAdapter = (*<name>Adapter)(nil)` compile-time assertion.
- [ ] `argv.go`, `env.go`, `stream.go` separated for clarity.
- [ ] Binary name hardcoded; no `command` override.
- [ ] Closed POSIX env baseline + `os.Environ()` NOT inherited.
- [ ] `ParseStreamEvent` maps every CLI event type cleanly to canonical `StreamEvent`.
- [ ] `ExtractTerminalReport` populates `TerminalReport.Cost` (pointer; nil if absent), `Denials`, `Reason`, `Errors`.
- [ ] `init.go` self-registration via `dispatcher.RegisterAdapter`.
- [ ] Blank import in `cmd/till/main.go`.
- [ ] `CLIKind<Name>` constant added; `IsValidCLIKind` updated.
- [ ] MockAdapter contract test extended.
- [ ] `testdata/<name>_stream_minimal.jsonl` fixture committed.
- [ ] `mage check` + `mage ci` green.

## References

- `SPAWN_PIPELINE.md` — pipeline architecture overview.
- `internal/app/dispatcher/cli_claude/` — reference implementation.
- `internal/app/dispatcher/cli_adapter.go` — interface + value-object types.
- `internal/app/dispatcher/mock_adapter_test.go` — contract test fixture.
- `WIKI.md` § "Cascade Vocabulary" — kind / role / structural_type axes.
