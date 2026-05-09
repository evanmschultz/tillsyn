# Spawn Pipeline Truth — what a spawned `claude` actually sees today

Read-only research deliverable. Traces the spawn pipeline as actually shipped in Drop 4c (`internal/app/dispatcher/cli_claude/...`) byte-for-byte. Every claim cites `file:line`. No code changes. No recommendations.

The dev's mental model: *"agents run in their own thing and get their own files and settings and so on so they don't inherit even the project stuff."* The investigation below tests that claim against the code Drop 4c actually shipped.

---

## A. Bundle assembly — every file written

`internal/app/dispatcher/cli_claude/render/render.go:125-179` (`Render`) writes exactly five files into the per-spawn bundle. Every byte is generated in pure-Go code. No file-system read against any project-local or user-local source contributes content. Below: filename, body construction, and source of every field.

### A.1 `<bundle.Root>/system-prompt.md`

Written by `renderSystemPrompt` (`render.go:220-226`) calling `assembleSystemPromptBody` (`render.go:246-279`). Body is built by `strings.Builder` from `(item domain.ActionItem, project domain.Project)` only:

- `task_id: <item.ID>` (line 248-250)
- `project_id: <project.ID>` (line 251-253)
- `project_dir: <project.RepoPrimaryWorktree>` (line 254-256)
- `kind: <item.Kind>` (line 257-259)
- `title: <item.Title>` (line 260-264, conditional on non-empty)
- `paths: <comma-joined item.Paths>` (line 265-269, conditional)
- `packages: <comma-joined item.Packages>` (line 270-274, conditional)
- A literal three-line move-state directive (line 275-277).

**No role definition, no tool discipline prose, no Section 0 scaffold.** The doc-comment at line 219 explicitly omits Hylla awareness ("Hylla awareness is deliberately omitted per F.7.10").

This is what `claude --system-prompt-file` reads. It is the **entire** substantive system prompt the spawned process gets unless the dev's `~/.claude/` adds something on top — see C.1.

### A.2 `<bundle.Root>/plugin/.claude-plugin/plugin.json`

Written by `renderPluginManifest` (`render.go:294-305`). Single JSON field `name: "spawn-<bundle.SpawnID>"` (line 299). Cosmetic plugin-manifest scaffolding so `claude` recognizes the bundle as a plugin tree per its plugin schema. Identical body shape every spawn except the spawn-id suffix.

### A.3 `<bundle.Root>/plugin/agents/<binding.AgentName>.md`

Written by `renderAgentFile` (`render.go:326-333`) calling `assembleAgentFileBody` (`render.go:340-364`). Body is hard-coded text:

```
---
name: <binding.AgentName>
description: Tillsyn-spawned <binding.AgentName> subagent.
allowedTools: <comma-joined binding.ToolsAllowed>     (conditional, line 349-353)
disallowedTools: <comma-joined binding.ToolsDisallowed>  (conditional, line 354-358)
---

Tillsyn-spawned subagent stub. Behavior loaded from the canonical
<binding.AgentName> template at the system-installed plugin path.
```

**This is a one-liner pointer stub.** The doc-comment at `render.go:307-319` explicitly states: *"The full canonical templates at ~/.claude/agents/<name>.md remain the source of truth for behavior — they are loaded by claude from the system-installed plugin path (Path B per memory §1), not from the per-spawn plugin (Path A)."*

The frontmatter does **NOT** include a `model:` field (see D below for what this means). The `tools:` allowlist/denylist mirrors the binding for human-readability only — Layer A in the SPAWN_PIPELINE.md "Two-layer tool-gating" model. Layer B (settings.json — A.5) is authoritative.

### A.4 `<bundle.Root>/plugin/.mcp.json`

Written by `renderMCPConfig` (`render.go:391-407`). Hard-coded JSON (line 396-401):

```json
{ "tillsyn": { "command": "till", "args": ["serve-mcp"] } }
```

Identical for every spawn. Registers `till serve-mcp` as a stdio MCP server named `tillsyn`. The literal `till` is resolved via the spawned process's `PATH` (which the closed-baseline env carries — see B.2).

### A.5 `<bundle.Root>/plugin/settings.json`

Written by `renderSettings` (`render.go:462-492`). JSON shape (lines 419-435):

```go
type settingsFile struct { Permissions permissionsBlock `json:"permissions"` }
type permissionsBlock struct {
    Allow []string `json:"allow"`  // binding.ToolsAllowed + persisted grants
    Ask   []string `json:"ask"`    // explicit empty
    Deny  []string `json:"deny"`   // mirrors binding.ToolsDisallowed
}
```

`allow` is `binding.ToolsAllowed` plus any persisted permission grants merged from the lister (`render.go:508-541` `mergeAllowList`); `deny` mirrors `binding.ToolsDisallowed` (`render.go:484`); `ask` stays explicit empty (`render.go:483`).

The grants lister is the only branch that COULD read external state — but it reads Tillsyn's own SQLite store (`permission_grants` table), not any project-local file. And in production today `BuildSpawnCommand` passes `nil` (`spawn.go:464`); the grants merge is a deferred-plumbing path until a future droplet wires the production handle through (`render.go:120-124` doc-comment).

### A.6 What the bundle does NOT contain

- **No copy of any project-local file.** No read of `<project>/.claude/`, `<project>/CLAUDE.md`, `<project>/.mcp.json`, `<project>/.tillsyn/agents/`, `<project>/agents.toml`, `<project>/SPAWN_PIPELINE.md`, etc.
- **No copy of any user-local file.** No read of `~/.claude/agents/`, `~/.claude/settings.json`, `~/.tillsyn/`. (Per AGENT_ARCHITECTURE_TRUTH.md §3 there's no `.tillsyn/agents/` source for the bundle to copy from anyway — the field exists in the schema but has zero seeding logic.)
- **No project CLAUDE.md inheritance.** Render is a pure function of `(item, project, binding, persisted-grants)`. Project `CLAUDE.md` is never read — and even if it were, the spawn flag `--bare` would skip it (see C.3).

---

## B. Cmd construction — argv + env + cwd + stdin

### B.1 Exact argv

Assembled by `assembleArgv` at `internal/app/dispatcher/cli_claude/argv.go:51-124`. The shape is fixed, in this order:

```
claude
  --bare
  --plugin-dir <bundle.Root>/plugin
  --agent <binding.AgentName>
  --system-prompt-file <bundle.Root>/system-prompt.md
  [--append-system-prompt-file <bundle.Root>/system-append.md]    (only if SystemAppendPath non-empty; argv.go:77-79)
  --settings <bundle.Root>/plugin/settings.json
  --setting-sources ""
  --strict-mcp-config
  --permission-mode acceptEdits
  --output-format stream-json
  --verbose
  --no-session-persistence
  --exclude-dynamic-system-prompt-sections
  --mcp-config <bundle.Root>/plugin/.mcp.json
  [--max-budget-usd <N>]    (only if binding.MaxBudgetUSD non-nil; argv.go:97-99)
  [--max-turns <N>]         (only if binding.MaxTurns non-nil; argv.go:100-102)
  [--effort <s>]            (only if binding.Effort non-nil; argv.go:103-105)
  [--model <s>]             (only if binding.Model non-nil; argv.go:106-108)
  [--tools <csv>]           (only if binding.Tools non-nil; argv.go:114-116)
  -p ""                     (always, but empty placeholder; argv.go:121)
```

Critical flags from an isolation perspective:

- **`--bare`** — claude's flag-name for the per-spawn-bundle mode. The SPAWN_PIPELINE.md "Explicit Non-Goals" section (line 105) names what `--bare` skips: *"Inheritance of orchestrator's CLAUDE.md / output styles / hooks. `--bare` skips them by design. Cascade subagents start in a fresh world; per-kind system prompt template subsumes role definitions."*
- **`--plugin-dir <bundle>/plugin`** — points claude at Tillsyn's per-spawn plugin tree (Path A per SPAWN_PIPELINE.md §"Two Plugin Paths"). `--bare` does NOT prevent claude from also loading system-installed plugins at `~/.claude/plugins/cache/...` (Path B). Path B is the dev's `~/.claude/agents/<name>.md` files.
- **`--settings <bundle>/plugin/settings.json`** + **`--setting-sources ""`** — `--setting-sources` empty quotes is the key. SPAWN_PIPELINE.md line 72: *"Tillsyn invokes claude with `--settings <path> --setting-sources ""` so user/project/local settings are ignored entirely."* The bundle's settings.json becomes the SOLE source of permission rules.
- **`--strict-mcp-config`** — forces claude to use only the `--mcp-config` argument's MCP servers, not project-discovered or user-discovered ones.
- **`--permission-mode acceptEdits`** — pre-approves edits; combined with deny patterns in settings.json this defines the actual gate.
- **`--mcp-config <bundle>/plugin/.mcp.json`** — explicit MCP config path. Not discovered from cwd.
- **`-p ""`** is a documented placeholder — the real prompt body is delivered via `--system-prompt-file` per A.1.

`claudeBinaryName` is hardcoded to `"claude"` at `adapter.go:40`. There is no `command` override field — REV-1 of the Drop 4c plan removed it. Process isolation beyond this is OS-level (sandbox-exec, container, PATH-shadowed shim) — see SPAWN_PIPELINE.md "Explicit Non-Goals" line 106.

### B.2 Exact env

Assembled by `assembleEnv` at `internal/app/dispatcher/cli_claude/env.go:76-147`. The returned slice is the COMPLETE `cmd.Env` — `os.Environ()` is NOT inherited (`env.go:60-65` doc-comment, plus the loop body which never calls `os.Environ()`).

Closed POSIX baseline (`env.go:37-58`, two declared groups):

```
PATH HOME USER LANG LC_ALL TZ TMPDIR XDG_CONFIG_HOME XDG_CACHE_HOME
HTTP_PROXY HTTPS_PROXY NO_PROXY http_proxy https_proxy no_proxy
SSL_CERT_FILE SSL_CERT_DIR CURL_CA_BUNDLE
```

Each baseline name is forwarded only when `os.LookupEnv` returns a value (`env.go:108-117`). Unset names are silently OMITTED — no `NAME=` empty entry.

Plus: each name in `binding.Env`. Missing required env (`os.LookupEnv` returns `false` for any name in `binding.Env`) is fail-loud via `ErrMissingRequiredEnv` (`env.go:17, 96-99`). Per F.7.17 P5 this routes to pre-lock so no spawn lock is acquired against a doomed binding.

That's it. Sentinel values like `AWS_ACCESS_KEY_ID` from the orchestrator's process are NOT visible to the spawn unless the binding's `Env` declares them by name.

### B.3 Working directory (`cmd.Dir`)

`spawn.go:476`:

```go
cmd.Dir = project.RepoPrimaryWorktree
```

The spawned `claude` process's cwd is **the project worktree**, the same path the orchestrator launches from. This is critical for the isolation question — see E.

### B.4 Stdin / stdout / stderr

`adapter.go:74-88` (`BuildCommand`) sets `cmd.Env` and returns. It does NOT touch `cmd.Stdin` / `cmd.Stdout` / `cmd.Stderr` — those default to `nil` for stdin (no input piped) and the Go stdlib's behavior for stdout/stderr (default: `nil` → discarded; or whatever the dispatcher's monitor wires in its launch step). The dispatcher's monitor (`monitor.go`) is what consumes stream-JSON from stdout per SPAWN_PIPELINE.md §"Stream-JSON Event Taxonomy" — that wiring is downstream of `BuildCommand` and outside this adapter's scope.

### B.5 `--agent <name>` source

`argv.go:71`:

```go
"--agent", binding.AgentName,
```

`binding.AgentName` is `BindingResolved.AgentName` (`cli_adapter.go:106`), populated verbatim from `templates.AgentBinding.AgentName` by `ResolveBinding` (`binding_resolved.go:118`). For Go projects this comes from `default-go.toml`'s `[agent_bindings.<kind>].agent_name = "go-builder-agent"` etc. (`default-go.toml:389, 418, 431, 466, 493, 520, 554, 588, 601, 615`). No lookup, no resolution — straight string passthrough.

---

## C. What the spawned `claude` actually reads at run time

This is the load-bearing question. The bundle and argv are observable; what claude does with them is governed by claude's own behavior. The evidence below combines (a) the argv shape, (b) SPAWN_PIPELINE.md's documented contract, and (c) the explicit non-goals.

### C.1 System prompt source

**The bundle's `<bundle>/system-prompt.md` IS what claude uses as its system prompt** — it's passed via `--system-prompt-file`. The agent stub at `<bundle>/plugin/agents/<name>.md` (A.3) is the agent-DEFINITION file claude reads when `--agent <name>` is set.

CRITICAL: the bundle's stub agent file (A.3) only carries the frontmatter (`name`, `description`, `allowedTools`/`disallowedTools`) plus a single body line that says behavior is loaded "from the canonical template at the system-installed plugin path." The render.go doc-comment at `render.go:311-319` explicitly states this two-source model:

> *"The full canonical templates at ~/.claude/agents/<name>.md remain the source of truth for behavior — they are loaded by claude from the system-installed plugin path (Path B per memory §1), not from the per-spawn plugin (Path A)."*

So claude's behavior in a spawned subagent process today depends on **both**:

1. The bundle's per-spawn plugin tree (Path A — pure stub, frontmatter only).
2. The dev's `~/.claude/agents/<name>.md` (Path B — the substantive system prompt the agent actually follows).

Whether claude actually merges these two on `--agent <name>` is governed by claude's plugin-loader code, not by Tillsyn. SPAWN_PIPELINE.md treats Path B as authoritative for behavior content (line 28-30) and Path A as the integration surface Tillsyn owns.

**Concrete consequence: removing `~/.claude/agents/go-builder-agent.md` from the dev's machine leaves the spawned builder with the bundle's pointer stub only — frontmatter and a redirect message. AGENT_ARCHITECTURE_TRUTH.md §1 confirms there is no in-repo source for the canonical content, so a fresh contributor or CI runner has nothing.**

### C.2 Tool allowlist source

Layered per SPAWN_PIPELINE.md §"settings.json Authority" (line 70-77):

- **Layer B (authoritative)** — `<bundle>/plugin/settings.json` `permissions.{allow,ask,deny}`. Per `--settings <path> --setting-sources ""`, this is the SOLE source claude consults — no merge with `~/.claude/settings.json` or `<project>/.claude/settings.json`.
- **Layer A (mirror)** — `<bundle>/plugin/agents/<name>.md` frontmatter `allowedTools` / `disallowedTools`. Mirrors B for human readability; not authoritative.

The `--tools <csv>` argv flag is conditional (`argv.go:114-116`); typical kinds emit nothing there and rely on settings.json.

### C.3 Model name / effort / argv flags

- `--model <s>` and `--effort <s>` are emitted only when `BindingResolved.Model` / `Effort` are non-nil (`argv.go:103-108`). `ResolveBinding` always promotes the rawBinding scalar to a non-nil pointer (`binding_resolved.go:143-144` via `resolveStringPtr` which always returns a non-nil pointer per `binding_resolved.go:160-172`), so for templated kinds the flag is always emitted.
- The bundle's agent stub frontmatter does NOT include a `model:` field today (`render.go:340-364` writes only `name`, `description`, `allowedTools`, `disallowedTools`). This matters for D.

### C.4 Env vars

Per B.2 — the closed POSIX baseline plus binding-declared names. Nothing else.

### C.5 Working directory

Per B.3 — `project.RepoPrimaryWorktree`. The spawned process runs IN the project tree, not in the bundle dir.

### C.6 Settings sources

`--setting-sources ""` (argv.go:83) shuts off claude's normal settings discovery from `~/.claude/`, `<project>/.claude/`, etc. Only the bundle's `settings.json` applies. SPAWN_PIPELINE.md line 72 documents this verbatim.

### C.7 Project agent files

`<project>/.claude/agents/` is NOT explicitly added or removed by Tillsyn argv. Whether claude discovers it from cwd is governed by claude's behavior. SPAWN_PIPELINE.md is silent on whether `--bare` + `--plugin-dir` skips project-cwd-discovered agents; it does say `--bare` skips "orchestrator's CLAUDE.md / output styles / hooks." Tillsyn's project tree at `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/.claude/` contains only `settings.local.json` (project ls confirms — no `agents/` subdir under `.claude/` in this project). For a project that DID have `<project>/.claude/agents/`, whether the spawn sees it is governed by claude itself. Tillsyn does not surface a flag to suppress that; it relies on `--bare`'s declared scope.

### C.8 MCP server registrations

`--mcp-config <bundle>/plugin/.mcp.json` + `--strict-mcp-config` together mean: claude only registers MCP servers from the bundle's `.mcp.json`. No discovery from `~/.claude/`, `<project>/.mcp.json`, or other paths.

The bundle's `.mcp.json` ships exactly one server — `tillsyn` (= `till serve-mcp`). So a spawned subagent can call `till.*` MCP methods (action_item, comment, etc.) but does NOT have access to the dev's other MCP servers (e.g. `hylla`, `context7`, project-named MCPs).

---

## D. The frontmatter `model:` question

**Quote of the dev's question**: *"for 'model: <opus | sonnet | haiku> # default; agents.toml binding can override', how will this work?"*

### D.1 Where `model` is set today

The TOML template carries `model` per binding — `default-go.toml:390, 419, 432, 467, 494, 521, 555, 589, 602, 616`. `ResolveBinding` populates `BindingResolved.Model` (`binding_resolved.go:143`). `assembleArgv` emits `--model <m>` to the spawned `claude` process when non-nil (`argv.go:106-108`).

**The bundle's agent .md frontmatter does NOT include a `model:` field today** — `assembleAgentFileBody` (`render.go:340-364`) emits only `name`, `description`, `allowedTools`, `disallowedTools`. So the question "who reads the frontmatter `model:` if the agent .md has one" has no in-Tillsyn answer for the bundle's stub — Tillsyn never writes that field there.

For the dev's `~/.claude/agents/<name>.md` (Path B), if the dev writes `model: opus` in the frontmatter, claude reads it natively as part of its agent-loading. That is claude's behavior, not Tillsyn's.

### D.2 Conflict shape

So the conflict shape today between argv `--model <m>` (Tillsyn-emitted) and frontmatter `model:` (dev-authored at `~/.claude/agents/<name>.md`) is determined by claude's CLI precedence rules — argv flags typically win over frontmatter in claude's documented behavior, but this is governed by claude's code, not Tillsyn's. SPAWN_PIPELINE.md does not document the precedence; CLI_ADAPTER_AUTHORING.md does not either.

### D.3 The `agents.toml` future state

The Drop 4c.6 SKETCH at `workflow/drop_4c_6/SKETCH.md:34-48` proposes moving runtime config (`model`, `effort`, etc.) from `default-go.toml` into a new `<project>/agents.toml` (project default, git-tracked) plus `<project>/agents.local.toml` (user override, gitignored). Resolution order per SKETCH §6:

1. At project load: read `agents.toml` (required); deep-merge `agents.local.toml` if present.
2. At spawn: dispatcher reads `binding.agent_name` + `binding.tools` + `binding.context` from template; reads runtime block from cached `agents.toml` resolution by kind; constructs `cmd.Env` and `cmd.Args`.

If both the agent .md frontmatter (`~/.claude/agents/<name>.md`) and the `agents.toml` carry a `model`, **today's plan is that `agents.toml` is the runtime authority because Tillsyn passes `--model <m>` argv**, which claude treats as the runtime override. The frontmatter `model:` would only matter if claude's loader reads it independently of argv — and even then claude's argv-vs-frontmatter precedence (controlled by claude's code) decides.

The SKETCH does NOT propose extending `assembleAgentFileBody` to write `model:` into the bundle's stub frontmatter. SKETCH §10 explicitly defers "Agent system-prompt overhauls — pending research findings." So the prompt-shaping question (and any frontmatter model field) is a separate drop.

### D.4 Concrete answer to the dev's question

- If "frontmatter model" means the bundle's stub: Tillsyn does not write that field today and has no plan to. Argv `--model` is authoritative.
- If "frontmatter model" means `~/.claude/agents/<name>.md`: it's claude's loader's call whether to honor it; argv `--model` from `agents.toml` (post-4c.6) or `default-go.toml` (today) is what Tillsyn passes, and claude's documented argv-precedence determines the winner.

---

## E. The "agents inherit project state" question

The dev's mental model: *"agents run in their own thing and get their own files and settings and so on so they don't inherit even the project stuff."*

### E.1 Verify or refute against the code

**E.1.1 — cwd is the project worktree.** `spawn.go:476` sets `cmd.Dir = project.RepoPrimaryWorktree`. The spawned `claude` process literally runs in `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/`. It can `os.ReadFile("CLAUDE.md")`, `os.ReadFile(".claude/settings.json")`, `os.ReadFile(".git/config")`, `os.ReadFile("WIKI.md")` — anything tracked in the project — at the OS level.

**E.1.2 — no OS-level sandboxing.** `cmd.Path` (resolved from argv[0] via `cmd.Env`'s `PATH`) is the dev's `~/.local/bin/claude` (per `which claude`). `BuildCommand` does not wrap it in `sandbox-exec`, Docker, Firejail, chroot, or anything else (`adapter.go:74-88`). SPAWN_PIPELINE.md "Explicit Non-Goals" line 103-106 confirms: *"Adversarial OS-level sandbox for Read/Edit/Write tools. Bash gets cap-dropping via sandbox.filesystem; Read/Edit/Write rely on cooperative permission rules. Wrap the binary in Docker if adversarial isolation is required."* Today's `BindingResolved` carries no Sandbox sub-struct (`render.go:419-425` — `settingsFile.Sandbox` is omitted as zero-value because the binding doesn't carry it yet). So `sandbox.filesystem` cap-dropping is not actually emitted into settings.json today either.

**E.1.3 — argv-level isolation via `--bare` + `--setting-sources ""` + `--strict-mcp-config` + `--plugin-dir`.** This is the actual isolation guarantee Drop 4c shipped. SPAWN_PIPELINE.md line 105 names what `--bare` skips:

> *"Inheritance of orchestrator's CLAUDE.md / output styles / hooks. `--bare` skips them by design. Cascade subagents start in a fresh world; per-kind system prompt template subsumes role definitions."*

And line 72 on `--setting-sources ""`:

> *"Tillsyn invokes claude with `--settings <path> --setting-sources ""` so user/project/local settings are ignored entirely."*

So claude — when it cooperates with `--bare` + `--setting-sources ""` — does not load:

- `~/.claude/CLAUDE.md` global instructions.
- `<project>/CLAUDE.md` project instructions.
- `~/.claude/output-styles/*.md`.
- `~/.claude/settings.json`.
- `<project>/.claude/settings.json`.
- `<project>/.claude/settings.local.json`.
- `~/.claude/hooks/*.sh`.

It DOES load:

- `<bundle>/system-prompt.md` (Tillsyn's prompt — A.1).
- `<bundle>/plugin/agents/<name>.md` (Tillsyn's stub — A.3).
- `~/.claude/agents/<name>.md` (Path B per SPAWN_PIPELINE.md §"Two Plugin Paths") — the dev's substantive prompt content. SPAWN_PIPELINE.md does NOT document `--bare` skipping system-installed plugins; the two-paths model on line 28-30 treats Path B as the canonical content source.
- `<bundle>/plugin/settings.json` (Tillsyn's permissions — A.5).
- `<bundle>/plugin/.mcp.json` (Tillsyn's MCP config — A.4).

**E.1.4 — file-system reachability vs prompt-loading.** This is the subtlety. The spawned `claude` process can OS-level read `<project>/CLAUDE.md` (cwd is `<project>`, no sandbox), but its prompt-loading code is told via argv to skip those files. Whether claude's tool calls (Read, Edit, Bash) can be steered by an adversarial agent toward those files is a different question — settings.json `permissions.allow|ask|deny` patterns govern what tool calls succeed, but the files themselves are reachable to a tool that the agent has permission to use.

Tillsyn's approach is "cooperative permission rules" (SPAWN_PIPELINE.md line 103-104). For an adversarial subagent, OS-level sandboxing is the dev's responsibility (Docker, Firejail, sandbox-exec) — Tillsyn does not ship it.

### E.2 What isolation Drop 4c shipped

Quoting SPAWN_PIPELINE.md "Explicit Non-Goals" line 103-106 verbatim:

> *"Adversarial OS-level sandbox for Read/Edit/Write tools. Bash gets cap-dropping via `sandbox.filesystem`; Read/Edit/Write rely on cooperative permission rules. Wrap the binary in Docker if adversarial isolation is required.* … *Wrapper-interop knob in Tillsyn core. No `command` field, no Docker awareness, no OAuth registry. Adopters who want process isolation install OS-level wrappers (PATH-shadowed binary, container wrapping the entire Tillsyn binary, sandbox-exec). The adapter calls its CLI binary directly."*

So the actual isolation guarantee is:

1. **System prompt isolation** — `--bare` + `--system-prompt-file` ensure the spawned process's system prompt is exactly Tillsyn's bundle file. CLAUDE.md (project + user) is skipped.
2. **Settings/permissions isolation** — `--settings` + `--setting-sources ""` ensure the bundle's `settings.json` is the SOLE source of permission rules.
3. **MCP isolation** — `--mcp-config` + `--strict-mcp-config` ensure only the bundle's `.mcp.json` MCP servers are registered.
4. **Env isolation** — `cmd.Env` is the closed POSIX baseline + binding's `Env` allowlist. `os.Environ()` is NOT inherited.
5. **NO file-system isolation.** The spawned process runs in the project worktree with full read access to anything the OS allows.
6. **NO Bash sandbox today** — the binding carries no sandbox sub-struct yet (`render.go:419-425`); the cap-dropping mechanism named in SPAWN_PIPELINE.md line 84 is documented but not yet wired into render.

---

## F. Cross-check against the dev's mental model

The dev said: *"agents run in their own thing and get their own files and settings and so on so they don't inherit even the project stuff."*

### F.1 Claim — *"agents run in their own thing"*

**PARTIAL.** Each spawn gets its own per-spawn bundle directory (`<bundle.Root>/`) with five generated files (A.1-A.5). Each spawn is a distinct `*exec.Cmd` with its own argv, env, and pid. In that sense yes, each agent runs in its own thing.

But **the spawned process's cwd is the project worktree** (B.3, `cmd.Dir = project.RepoPrimaryWorktree`). It's not chrooted, not containerized, not `sandbox-exec`'d. So while the bundle is per-spawn, the process is sitting in `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/` with full OS read/write access (subject to filesystem perms). The "own thing" is the bundle, not a sandbox.

### F.2 Claim — *"and get their own files and settings"*

**PARTIAL — TRUE for argv-controlled prompt + settings + MCP; FALSE if "files" means complete file-system isolation.**

True parts:

- Own system prompt: `--system-prompt-file <bundle>/system-prompt.md` (A.1, B.1).
- Own settings: `--settings <bundle>/plugin/settings.json --setting-sources ""` shuts off project + user settings inheritance (A.5, B.1).
- Own MCP config: `--mcp-config <bundle>/plugin/.mcp.json --strict-mcp-config` (A.4, B.1).
- Own permission grants: persisted in Tillsyn's SQLite (`permission_grants` table) and merged into bundle settings.json (A.5, render.go:529).

Not-true parts:

- The substantive agent behavior content is NOT in the bundle — it's at `~/.claude/agents/<name>.md` on the dev's machine (A.3, AGENT_ARCHITECTURE_TRUTH.md §1). The bundle's `agents/<name>.md` is a one-line redirect stub. So in practice the spawned agent's behavior is governed by user-machine-local files, not bundle files.
- The agent has full OS read access to project files via Read/Edit/Bash tools (subject to settings.json permission patterns). It can `cat <project>/CLAUDE.md` if Bash is allowed.

### F.3 Claim — *"so they don't inherit even the project stuff"*

**PARTIAL — TRUE for what claude's prompt-loader / settings-loader / MCP-loader inherits at startup; FALSE for what tools can reach at run time.**

True parts:

- Project CLAUDE.md: NOT loaded as system prompt (`--bare` skips it per SPAWN_PIPELINE.md line 105).
- Project `.claude/settings.json`: NOT loaded (`--setting-sources ""`).
- Project `.mcp.json`: NOT loaded (`--strict-mcp-config`).
- Project `<project>/.claude/output-styles/`: NOT loaded.
- Orchestrator's hooks (`~/.claude/hooks/`): NOT loaded (`--bare`).

Not-true parts:

- Project file content is reachable via Read/Bash if the agent's tool-permission patterns allow. Cooperative gate, not OS-level lockout.
- The `~/.claude/agents/<name>.md` files (user-local, NOT project-local) ARE the substantive system prompt content the cascade depends on per A.3 + SPAWN_PIPELINE.md §"Two Plugin Paths" line 28-30. They're "user stuff" not "project stuff" — but they're inherited from outside the bundle.

### F.4 Net summary

The dev's mental model is **directionally correct** about argv-level isolation (`--bare` + `--setting-sources ""` + `--strict-mcp-config` + per-spawn bundle), but **materially overstates** the isolation in two ways:

1. **The substantive agent prompt content is NOT in the bundle.** It lives at `~/.claude/agents/<name>.md`. This means: a fresh contributor or CI runner without those files in `~/.claude/agents/` sees only the bundle's pointer-stub redirect message. AGENT_ARCHITECTURE_TRUTH.md §3 confirms there's no `.tillsyn/agents/` source in-repo for the bundle to copy from. The cascade is undocumented from any other dev's perspective today. (That research's blunt phrasing: *"If the dev's machine is the only place the cascade-agent contracts exist, the cascade is undocumented from any other dev's perspective."*)

2. **There is no OS-level sandbox.** Spawned agents run in `cmd.Dir = <project worktree>` with full read access to project files. SPAWN_PIPELINE.md "Explicit Non-Goals" line 103-104 is explicit that adversarial OS-level isolation is not Tillsyn's surface — Read/Edit/Write rely on cooperative permission rules. Bash sandbox cap-dropping is documented (line 83-84) but the binding's Sandbox sub-struct is not yet wired into render (render.go:419-425).

The Drop 4c plan-vs-reality gap the dev sensed (*"because it seems like a lot is off from what we planned for pre dogfood"*) is real on dimension #1. The plan presumed `~/.claude/agents/` would be authoritative (Path B per the spawn architecture memory), but the cost of that decision — that the cascade's behavior contracts live on one dev's machine — was underweighted relative to multi-dev / CI dogfood viability.

The Drop 4c.6 SKETCH (workflow/drop_4c_6/SKETCH.md) does not address dimension #1; it covers runtime config (model/endpoint/retries/budgets) only. SKETCH §10 explicitly defers "Agent system-prompt overhauls — pending research findings." Per AGENT_ARCHITECTURE_TRUTH.md §7 the per-project agent-file shipping question is the next decision point — Option A (absorb into 4c.6) vs Option B (defer to dedicated drop) vs Option C (keep `~/.claude/agents/` as authoritative and accept the multi-dev hole).

---

## Hylla Feedback

- **Query**: `hylla_search_keyword(query="cmd.Dir RepoPrimaryWorktree", fields=["content"])` — zero results.
- **Missed because**: the search failed to match identifier-shaped tokens in code bodies. `cmd.Dir` is a struct-field access pattern; the keyword index appears not to tokenize `.`-separated identifiers as searchable units, and the symbol index returns symbol-level entries (not occurrences inside function bodies).
- **Worked via**: direct file `Read` of `internal/app/dispatcher/spawn.go` (already loaded for §A) — `cmd.Dir = project.RepoPrimaryWorktree` is at line 476.
- **Suggestion**: support keyword search of `.`-separated method/field-access patterns in code bodies, OR expose a dedicated "occurrences in body text" mode distinct from symbol-level search. Today's fallback (`Read` once you already know the file) only works because the file is small enough to scan visually.

- **Query**: `hylla_search_keyword(query="--setting-sources --bare --plugin-dir", fields=["content"])` — returned ten results, none relevant (returned `internal/config/Default`, `UpsertIdentity`, `LoggingDevFileConfig`, etc.).
- **Missed because**: token-level OR ranking on three independently-rare tokens (`--setting-sources`, `--bare`, `--plugin-dir`) produced no matches because the actual token in source is the literal Go string `"--setting-sources"` (with quotes) rather than a bareword. Hylla's tokenizer appears to strip surrounding quotes but match-fail on the embedded `--` prefix when combined with no fielded scope, so it falls back to noisy whole-content keyword scoring.
- **Worked via**: direct read of `internal/app/dispatcher/cli_claude/argv.go:67-91` which lists all the always-on flags in declaration order. Plus `SPAWN_PIPELINE.md:72` for the documented contract phrasing.
- **Suggestion**: improve handling of CLI-flag-shaped tokens (`--<word>`, `-x`) — they're high-signal in adapter / argv / shell-wiring code but low-signal in narrative prose, so a per-field weighting (content > docstring > summary) would help. Or document a phrase-search mode that preserves token order so multi-token CLI-flag queries like `"--setting-sources" "--bare"` actually work.

Bash sandbox note: the Bash policy for this agent role denied `grep -n` and `find` invocations against the project root (permission denied) but allowed `git status`, `pwd`, and `which`. `git grep` would likely have helped close the keyword gaps above; the AGENT_ARCHITECTURE_TRUTH.md research session called out the same asymmetry. Confirms it's a research-agent-policy issue not a per-session glitch. Prior research's recommendation to surface a "structural-only" or "best-effort partial" Hylla mode would also help when keyword search's ranking misfires this badly on CLI-flag tokens.
