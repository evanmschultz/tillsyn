# Spawn Pipeline Architecture

How Tillsyn dispatches headless Claude (and future) CLI subagents during the cascade. This document is the canonical reference for the spawn pipeline shipped in Drop 4c. For day-to-day Tillsyn usage see `WIKI.md`; for cascade vocabulary see `WIKI.md` § "Cascade Vocabulary".

## Overview

When the dispatcher promotes an action item to `in_progress`, it spawns a headless `claude` (or future `codex`) process to perform the work. The spawn pipeline produces a per-spawn temp bundle, invokes the CLI binary with a precise argv, captures the stream-JSON event log, and translates terminal events into Tillsyn outcomes.

```
Action item promoted to in_progress
        │
        ▼
1. Resolve binding (CLI > MCP > TUI > template > absent priority cascade)
2. Pick adapter via cli_kind (claude today; codex Drop 4d)
3. Create per-spawn bundle (manifest.json + system-prompt.md + plugin/...)
4. Render bundle artifacts (system prompt, settings.json, agent file, .mcp.json)
5. Inject permission grants from prior dev approvals
6. Build *exec.Cmd via adapter.BuildCommand
7. Spawn process; tail stream.jsonl line-by-line
8. On terminal event: extract cost + permission denials
9. On dead spawn: orphan-scan reaps PID + posts attention item
```

## Two Plugin Paths

Tillsyn distinguishes two ways CLI-side plugins are loaded. They are not interchangeable:

- **Path A — Per-spawn bundle plugin (`--plugin-dir <bundle>/plugin`).** Tillsyn writes a fresh plugin directory per spawn (`plugin.json`, `agents/<name>.md`, `.mcp.json`, `settings.json`). Pure local file I/O. No network. Bundle deletes on terminal-state. ~2ms file overhead per spawn. This is the integration surface Tillsyn owns.
- **Path B — System-installed plugins (`claude plugin install <name>`).** Persistent install at `~/.claude/plugins/cache/...`. Dev runs `claude plugin install` once per machine. Tillsyn never installs/uninstalls — only pre-flight-checks via `claude plugin list --json` against project-declared `tillsyn.requires_plugins`.

There is no Path C (per-spawn install/uninstall).

## Per-Spawn Bundle Layout

Each spawn produces a temp bundle:

```
<bundle-root>/
├── manifest.json              # Tillsyn-internal: spawn_id, action_item_id, kind,
│                              # claude_pid, started_at, paths, cli_kind, bundle_path
├── system-prompt.md           # Per-spawn rendered prompt (action item shape +
│                              # auth session_id + working dir + bundle paths)
├── stream.jsonl               # Captured --output-format stream-json events
├── context/                   # Optional pre-staged context (F.7.18 aggregator)
│   └── <rule>.md              # Per-rule rendered content (parent, ancestors, ...)
└── plugin/                    # CLI-specific subtree (claude shape)
    ├── .claude-plugin/
    │   └── plugin.json        # {"name": "spawn-<id>"}
    ├── agents/
    │   └── <name>.md          # Rendered from canonical agent template
    ├── .mcp.json              # Tillsyn-MCP self-registration (stdio child)
    └── settings.json          # permissions.allow|ask|deny + sandbox.* per binding
```

Bundle root is `os.TempDir()/tillsyn-spawn-<uuid>/` by default (`tillsyn.spawn_temp_root = "os_tmp"`) or `<worktree>/.tillsyn/spawns/<spawn-id>/` in project mode (`tillsyn.spawn_temp_root = "project"`). Project mode auto-adds `.tillsyn/spawns/` to `.gitignore` once per process.

## Stream-JSON Event Taxonomy

Tillsyn parses `--output-format stream-json` line by line per the canonical claude event types. The dispatcher monitor (`internal/app/dispatcher/monitor.go`) is CLI-agnostic; it routes via `adapter.ParseStreamEvent` returning a cross-CLI canonical `StreamEvent`. Each adapter maps its CLI's events into the canonical shape.

Key canonical event types:

- **`system_init`** — session initialization. Carries final tool list, mcp servers, model, permission mode.
- **`assistant`** — agent turn. Each content block is `thinking`, `text`, or `tool_use`.
- **`user`** — tool result fed back to the agent.
- **`result`** (terminal) — single per-spawn event carrying `total_cost_usd`, `permission_denials[]`, `terminal_reason`, `errors[]`.

Per-spawn cost is recorded to `metadata.actual_cost_usd` when the terminal event arrives. Permission denials are surfaced via the TUI handshake (`internal/app/dispatcher/handshake.go`) as Tillsyn attention items the dev approves/denies.

## settings.json Authority

The per-spawn `settings.json` is the SOLE source of permission rules. Tillsyn invokes claude with `--settings <path> --setting-sources ""` so user/project/local settings are ignored entirely. Two-layer tool-gating strategy:

- **Layer B (authoritative) — `settings.json` `permissions.allow|ask|deny`.** Pattern syntax expressive enough to cover Bash workarounds (e.g. `Bash(curl *)` to deny network access via Bash even when WebFetch is denied).
- **Layer A (mirror, human-readable) — agent frontmatter `disallowedTools`.** Mirrors B for documentation; not authoritative.

CLI flag `--disallowed-tools` is skipped for typical kinds — agents route around tool removal via Bash, only deny patterns catch workarounds. `--tools` is emitted only when a kind wants engine-level minimization (e.g. research agents get `Read,Grep,Glob` only).

## Sandbox Semantics

`settings.json` `sandbox` block declares filesystem and network policy:

- **Bash sandbox is OS-level (capability-dropping).** Real enforcement; subprocess Bash invocations honor the sandbox.
- **Read/Edit/Write tools rely on Claude's permission engine, NOT kernel sandbox.** For an adversarial subagent (e.g. model jailbreak), cooperative deny rules are insufficient. Tillsyn does not ship adversarial OS-level sandboxing for these tools — adopters who need it wrap the entire `claude` invocation in Docker / Firejail externally (Tillsyn names no specific wrapper; see `command` spec in `CLAUDE.md`).

## Crash Recovery

Tillsyn restart enumerates every `in_progress` action item. For each:

1. Read `metadata.spawn_bundle_path` → manifest.json.
2. Check `claude_pid` liveness via `os.FindProcess` + signal 0 + cmdline-match against the manifest's `cli_kind`.
3. **Alive** → leave; dispatcher re-monitors via SQLite state changes.
4. **Dead** → action item moves to `failed` with `metadata.failure_reason = "dispatcher_restart_orphan"`; bundle deleted; dev sees in TUI and decides re-dispatch.

PID-zero distinguishes "spawn not yet started" from "spawn started and may be alive."

## Permission Handshake

When a spawn's terminal event carries `permission_denials[]`, Tillsyn posts one attention item per denial to the dev's TUI. Dev approves "allow once" / "allow always" / "deny." Approved-always entries persist to SQLite `permission_grants(project_id, kind, rule, cli_kind, granted_by, granted_at)` table. Next spawn of the same `(project, kind, cli_kind)` reads grants and injects them into `settings.json` `permissions.allow`.

## Explicit Non-Goals

- **Adversarial OS-level sandbox for Read/Edit/Write tools.** Bash gets cap-dropping via `sandbox.filesystem`; Read/Edit/Write rely on cooperative permission rules. Wrap the binary in Docker if adversarial isolation is required.
- **Real-time interactive permission prompts.** Tillsyn's TUI cannot intercept Claude's stdin prompt. The handshake is failure-loop: terminal `permission_denials[]` → TUI → dev approval → next spawn picks up the grant.
- **Inheritance of orchestrator's CLAUDE.md / output styles / hooks.** `--bare` skips them by design. Cascade subagents start in a fresh world; per-kind system prompt template subsumes role definitions.
- **Wrapper-interop knob in Tillsyn core.** No `command` field, no Docker awareness, no OAuth registry. Adopters who want process isolation install OS-level wrappers (PATH-shadowed binary, container wrapping the entire Tillsyn binary, sandbox-exec). The adapter calls its CLI binary directly.

## CLI Adapter Seam

The pipeline is multi-CLI extensible via the `CLIAdapter` interface (see `CLI_ADAPTER_AUTHORING.md`). Drop 4c ships the `claude` adapter; Drop 4d adds `codex`. Both are JSONL-stream CLIs. Non-JSONL extensibility (SSE / framed-binary / no-stream) is a future-roadmap concern requiring a hard-cut interface rewrite.

## References

- `internal/app/dispatcher/spawn.go` — `BuildSpawnCommand` entrypoint.
- `internal/app/dispatcher/cli_adapter.go` — `CLIAdapter` interface + `BindingResolved` + `BundlePaths` + `StreamEvent` + `TerminalReport`.
- `internal/app/dispatcher/cli_claude/` — claude adapter implementation.
- `internal/app/dispatcher/bundle.go` — per-spawn bundle lifecycle.
- `internal/app/dispatcher/monitor.go` — stream-JSON event monitor.
- `internal/app/dispatcher/handshake.go` — permission-denial → TUI handshake.
- `internal/app/dispatcher/orphan_scan.go` — crash-recovery PID liveness check.
- `internal/templates/builtin/default.toml` — default template with `[gates.build] = ["mage_ci", "commit", "push"]`.
- `WIKI.md` § "Cascade Vocabulary" — kind / role / structural_type axes.
- `CLAUDE.md` — project orchestration discipline.
