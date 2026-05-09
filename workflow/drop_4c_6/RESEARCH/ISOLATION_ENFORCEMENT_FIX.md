# Isolation Enforcement Fix — Definitive Answer

Read-only research deliverable. Combines (a) authoritative Claude Code documentation citations and (b) Tillsyn render-pipeline code citations to give the dev a single, actionable answer to two coupled questions:

1. What does Claude Code's plugin / agent / settings / MCP / skills loader actually do today under the current Tillsyn argv?
2. What is the concrete fix to GUARANTEE the dev's isolation requirement (no inheritance from `~/.claude/`, system, or project locations — only the per-spawn bundle defines the agent)?

The answer below is unhedged. Every load-bearing claim cites either Anthropic docs (URL) or a Tillsyn source line (`file:line`).

---

## A. Claude Code documentation review

### A.1 `--bare` flag — what it skips

Authoritative source: <https://code.claude.com/docs/en/headless#start-faster-with-bare-mode> + <https://code.claude.com/docs/en/cli-reference> (CLI reference table row for `--bare`).

CLI reference exact phrasing:

> `--bare` — *"Minimal mode: skip auto-discovery of hooks, skills, plugins, MCP servers, auto memory, and CLAUDE.md so scripted calls start faster. Claude has access to Bash, file read, and file edit tools. Sets `CLAUDE_CODE_SIMPLE`."*

Headless docs exact phrasing:

> *"Add `--bare` to reduce startup time by skipping auto-discovery of hooks, skills, plugins, MCP servers, auto memory, and CLAUDE.md. Without it, `claude -p` loads the same context an interactive session would, including anything configured in the working directory or `~/.claude`."*
>
> *"Bare mode is useful for CI and scripts where you need the same result on every machine. **A hook in a teammate's `~/.claude` or an MCP server in the project's `.mcp.json` won't run, because bare mode never reads them. Only flags you pass explicitly take effect.**"*

The headless table that immediately follows lists what flags are required to load anything in `--bare`:

| To load                 | Use                                                     |
| ----------------------- | ------------------------------------------------------- |
| System prompt additions | `--append-system-prompt`, `--append-system-prompt-file` |
| Settings                | `--settings <file-or-json>`                             |
| MCP servers             | `--mcp-config <file-or-json>`                           |
| Custom agents           | `--agents <json>`                                       |
| A plugin                | `--plugin-dir <path>`, `--plugin-url <url>`             |

This table is the definitive answer to half the dev's question. **Under `--bare`, plugins are skipped UNLESS opted in via `--plugin-dir` (or `--plugin-url`); custom agents are loaded ONLY via `--agents <json>` or via a plugin's `agents/` directory.** No project-local `<cwd>/.claude/agents/` discovery, no user-local `~/.claude/agents/` fallback, no system-installed plugin cache. CLAUDE.md (project + user) is in the skip list. Skills are in the skip list. Hooks are in the skip list. Auto-memory is in the skip list.

### A.2 `--plugin-dir` flag

Authoritative source: <https://code.claude.com/docs/en/cli-reference> table row for `--plugin-dir`:

> `--plugin-dir` — *"Load a plugin from a directory or `.zip` archive for this session only. Each flag takes one path. Repeat the flag for multiple plugins."*

In **non-bare** mode `--plugin-dir` is **additive** to the system-installed plugin cache (search results confirmed this; community gist <https://gist.github.com/gwpl/776057bec49c47c1327afda07fcc75d2> documents the additive-merge behavior). But that additivity only applies in non-bare mode. Under `--bare` the plugin auto-discovery is OFF entirely (per A.1 quote), so `--plugin-dir` becomes the only loaded plugin source.

### A.3 `--agent <name>`, agent file resolution priority

Authoritative source: <https://code.claude.com/docs/en/sub-agents#choose-the-subagent-scope>:

> *"Subagents are Markdown files with YAML frontmatter. Store them in different locations depending on scope. When multiple subagents share the same name, the higher-priority location wins."*

Priority table (verbatim):

| Location                     | Scope                   | Priority    |
| ---------------------------- | ----------------------- | :---------- |
| Managed settings             | Organization-wide       | 1 (highest) |
| `--agents` CLI flag          | Current session         | 2           |
| `.claude/agents/`            | Current project         | 3           |
| `~/.claude/agents/`          | All your projects       | 4           |
| Plugin's `agents/` directory | Where plugin is enabled | 5 (lowest)  |

Two consequences for Tillsyn:

1. The plugin's `agents/<name>.md` is the LOWEST priority. When Tillsyn ships its own plugin via `--plugin-dir <bundle>/plugin` AND a same-named agent file exists at `~/.claude/agents/<name>.md` AND `~/.claude/agents/` is being loaded (e.g. non-bare mode), the user-level file wins — Tillsyn's bundle stub is shadowed. **This is the leakage path the dev is worried about.**
2. Under `--bare` mode, levels 3 and 4 of the table (`.claude/agents/` and `~/.claude/agents/`) are not loaded at all (per A.1, plugins are also disabled but `--plugin-dir` opts the bundle's plugin back in). With `--bare` + `--plugin-dir <bundle>/plugin` + no `--agents` flag, the bundle's plugin agent is the ONLY agent definition Claude Code sees for that name.

### A.4 `--system-prompt-file`

CLI reference:

> `--system-prompt-file` — *"Load system prompt from a file, replacing the default prompt."*

Sub-agents docs (<https://code.claude.com/docs/en/sub-agents#invoke-subagents-explicitly>):

> *"The subagent's system prompt replaces the default Claude Code system prompt entirely, the same way `--system-prompt` does. **CLAUDE.md files and project memory still load through the normal message flow.** The agent name appears as `@<name>` in the startup header so you can confirm it's active."*

This second sentence is the gotcha for non-bare mode: even with `--agent <name>`, CLAUDE.md still loads. Under `--bare`, A.1 disables CLAUDE.md auto-discovery. So `--bare` + `--system-prompt-file` + `--agent` is the combination that fully suppresses CLAUDE.md.

### A.5 `--setting-sources` (empty)

CLI reference:

> `--setting-sources` — *"Comma-separated list of setting sources to load (`user`, `project`, `local`)."*

Settings docs (<https://code.claude.com/docs/en/settings>) confirm an empty list disables all three layers. Combined with `--settings <bundle>/plugin/settings.json`, the bundle's settings.json is the SOLE source.

### A.6 `--strict-mcp-config`

CLI reference:

> `--strict-mcp-config` — *"Only use MCP servers from `--mcp-config`, ignoring all other MCP configurations."*

This is unambiguous — the bundle's `.mcp.json` is the SOLE MCP source.

### A.7 `--exclude-dynamic-system-prompt-sections`

CLI reference:

> `--exclude-dynamic-system-prompt-sections` — *"Move per-machine sections from the system prompt (working directory, environment info, memory paths, git status) into the first user message. Improves prompt-cache reuse across different users and machines running the same task. **Only applies with the default system prompt; ignored when `--system-prompt` or `--system-prompt-file` is set.** Use with `-p` for scripted, multi-user workloads."*

Tillsyn passes `--system-prompt-file`, so this flag is a no-op today. Keep it (cheap, future-proof) or drop it (cleaner). Either is fine; this is not load-bearing for isolation.

### A.8 `--no-session-persistence`

CLI reference:

> `--no-session-persistence` — *"Disable session persistence so sessions are not saved to disk and cannot be resumed. Print mode only."*

Tangentially related: prevents disk-side session leakage between spawns. Keep it.

### A.9 Definitive answer to question 1

Under the current Tillsyn argv (`claude --bare --plugin-dir <bundle>/plugin --agent <name> --system-prompt-file ... --settings <bundle>/plugin/settings.json --setting-sources "" --strict-mcp-config --permission-mode acceptEdits ...`), Claude Code today:

- **Does NOT load `~/.claude/CLAUDE.md`** — `--bare` skips CLAUDE.md auto-discovery (A.1).
- **Does NOT load `<cwd>/CLAUDE.md`** — same reason. The dev's project worktree IS the cwd (`spawn.go:476`), but `--bare` skips CLAUDE.md regardless of where it sits.
- **Does NOT load `~/.claude/agents/<name>.md`** — `--bare` skips plugin/agent auto-discovery; the priority-table entries 3 and 4 are not consulted at session init in bare mode.
- **Does NOT load `<cwd>/.claude/agents/<name>.md`** — same reason.
- **Does NOT load `~/.claude/skills/`** — `--bare` skips skill directory walks.
- **Does NOT load `~/.claude/settings.json` / `<cwd>/.claude/settings.json` / `<cwd>/.claude/settings.local.json`** — `--setting-sources ""` excludes all three layers.
- **Does NOT load `~/.claude/plugins/cache/...`** (system-installed plugins) — `--bare` skips plugin auto-discovery.
- **Does NOT load `~/.claude/.mcp.json` / `<cwd>/.mcp.json`** — `--strict-mcp-config` restricts MCPs to the `--mcp-config` argument.
- **Does NOT load `~/.claude/hooks/`** — `--bare` skips hooks.
- **DOES load `<bundle>/plugin/agents/<name>.md`** — this is the bundle's stub, opted in via `--plugin-dir`. Per the priority table, this is the only agent definition that survives.
- **DOES load `<bundle>/plugin/settings.json`** — sole settings source.
- **DOES load `<bundle>/plugin/.mcp.json`** — sole MCP source.
- **DOES load `<bundle>/system-prompt.md`** — sole system prompt.

The earlier research's "two paths" model (SPAWN_PIPELINE.md:24-31) is a CORRECT description of Claude Code's plugin loader in the GENERAL case — but `--bare` is exactly the flag that collapses Path B. **The Tillsyn render code's doc-comment at `render.go:307-319` saying the canonical templates live at `~/.claude/agents/<name>.md` and "remain the source of truth" is FACTUALLY WRONG under the actual argv Tillsyn ships.** Under `--bare`, those files are not read. The bundle stub IS the only agent definition; today it is a one-line redirect, so the agent has frontmatter but no system-prompt body content.

The bundle's `agents/<name>.md` body content is the load-bearing field that's currently empty — not because Path B is silently winning, but because nothing populates the bundle stub's body.

---

## B. Tillsyn render code review

### B.1 Bundle stub today — confirmed one-line redirect

`render.go:340-364` (`assembleAgentFileBody`):

```go
b.WriteString("---\n")
b.WriteString("name: ")
b.WriteString(binding.AgentName)
b.WriteString("\n")
b.WriteString("description: Tillsyn-spawned ")
b.WriteString(binding.AgentName)
b.WriteString(" subagent.\n")
if len(binding.ToolsAllowed) > 0 {
    b.WriteString("allowedTools: ")
    b.WriteString(strings.Join(binding.ToolsAllowed, ", "))
    b.WriteString("\n")
}
if len(binding.ToolsDisallowed) > 0 {
    b.WriteString("disallowedTools: ")
    b.WriteString(strings.Join(binding.ToolsDisallowed, ", "))
    b.WriteString("\n")
}
b.WriteString("---\n\n")
b.WriteString("Tillsyn-spawned subagent stub. Behavior loaded from the canonical ")
b.WriteString(binding.AgentName)
b.WriteString(" template at the system-installed plugin path.\n")
```

Confirmed. Frontmatter (`name`, `description`, `allowedTools`, `disallowedTools`) plus a single body sentence claiming Path B is authoritative. **Per A.9, that claim is false under `--bare`.** No `model:` field, no `tools:` field (separate from `allowedTools`), no system-prompt body content.

### B.2 Argv today — confirmed `--bare` + `--setting-sources ""` + `--strict-mcp-config` + `--plugin-dir`

`argv.go:51-124`. The exact always-on shape:

```
claude
  --bare
  --plugin-dir <bundle>/plugin
  --agent <binding.AgentName>
  --system-prompt-file <bundle>/system-prompt.md
  [--append-system-prompt-file ...]
  --settings <bundle>/plugin/settings.json
  --setting-sources ""
  --strict-mcp-config
  --permission-mode acceptEdits
  --output-format stream-json
  --verbose
  --no-session-persistence
  --exclude-dynamic-system-prompt-sections
  --mcp-config <bundle>/plugin/.mcp.json
  [--max-budget-usd N] [--max-turns N] [--effort e] [--model m] [--tools csv]
  -p ""
```

Confirmed via `argv.go:67-91, 97-116, 121`. `claudeBinaryName` is hardcoded `"claude"` at `adapter.go:40`.

### B.3 No argv flag today disables Path B "fallthrough" explicitly

This is mooted by A.9 — `--bare` already disables Path B (and project-local agent dirs, project CLAUDE.md, system MCPs, hooks, skills, auto-memory). There is no separate "disable Path B" flag because Claude Code's own design already collapses it under `--bare`. The previous research's framing ("`--bare` does NOT prevent claude from also loading system-installed plugins at `~/.claude/plugins/cache/...` (Path B)") was speculative; the actual docs at A.1 list plugins among the things `--bare` skips.

### B.4 `--agent <name>` source

Confirmed `argv.go:71`: `"--agent", binding.AgentName`. `binding.AgentName` flows from `templates.AgentBinding.AgentName` via `ResolveBinding` (`binding_resolved.go:118`). For Go projects, sourced from `default-go.toml` `[agent_bindings.<kind>].agent_name = "go-builder-agent"` etc.

### B.5 Render pipeline structure

`render.go:125-179` (`Render`) writes exactly five files. None are read from any user/project local source. Render is a pure function of `(item, project, binding, persisted-grants)`. Confirmed in §A of `SPAWN_PIPELINE_TRUTH.md`.

The only mutable input that depends on dev-local state is `os.LookupEnv` lookups in `env.go:108-117` (closed POSIX baseline) and `env.go:96-99` (binding's `Env` allowlist). Neither feeds the render — they only populate `cmd.Env` for the spawned process.

---

## C. The two-paths model — what actually happens

### C.1 Quote from SPAWN_PIPELINE.md

`SPAWN_PIPELINE.md:24-31`:

> ## Two Plugin Paths
>
> Tillsyn distinguishes two ways CLI-side plugins are loaded. They are not interchangeable:
>
> - **Path A — Per-spawn bundle plugin (`--plugin-dir <bundle>/plugin`).** Tillsyn writes a fresh plugin directory per spawn (`plugin.json`, `agents/<name>.md`, `.mcp.json`, `settings.json`). Pure local file I/O. No network. Bundle deletes on terminal-state. ~2ms file overhead per spawn. This is the integration surface Tillsyn owns.
> - **Path B — System-installed plugins (`claude plugin install <name>`).** Persistent install at `~/.claude/plugins/cache/...`. Dev runs `claude plugin install` once per machine. Tillsyn never installs/uninstalls — only pre-flight-checks via `claude plugin list --json` against project-declared `tillsyn.requires_plugins`.
>
> There is no Path C (per-spawn install/uninstall).

### C.2 Why this model exists

It's a Tillsyn-side framing of Claude Code's general plugin model. In **non-bare mode**, both paths are loaded and merged (Path B is system-installed plugins; `--plugin-dir` is additive). The Tillsyn doc is correct that the two coexist in the general case. **Under `--bare`, Path B is disabled by Claude Code itself** (A.1 quote: `--bare` skips plugin auto-discovery). Tillsyn's own argv already opts the bundle (Path A) back in via `--plugin-dir`, but Path B is suppressed.

### C.3 Is Path B ALWAYS loaded?

No — see above. In non-bare mode yes, under `--bare` no. Tillsyn ships `--bare`, so Path B is dead today.

### C.4 Could Path A and Path B both load if Tillsyn shipped full content in Path A?

Under `--bare`, no — Path B is disabled by `--bare`. Under non-bare, yes — Claude Code merges plugin agents from `--plugin-dir` with installed plugins (additive per the gist + community confirmation). **Tillsyn's existing `--bare` already prevents that merge.**

### C.5 Net finding for Path B

Tillsyn's `SPAWN_PIPELINE.md`'s Path-B framing and `render.go:307-319`'s doc-comment claim that `~/.claude/agents/<name>.md` is "the source of truth for behavior" are **both factually incorrect** for the argv Tillsyn actually emits. The bundle's stub IS the only agent file Claude Code consults. Today the stub is empty of substantive prompt content, so the agent runs with frontmatter (model/tools) and an empty body — no role definition, no tool discipline, no Section 0 scaffold. This is the user-visible bug behind the dev's complaint.

---

## D. The fix — concrete recommendation

The fix is in three coupled parts. Each closes a specific path; none alone is sufficient.

### D.1 Fix change 1 — render the FULL agent body into the bundle stub

**WHAT.** Replace `assembleAgentFileBody` at `render.go:340-364` so the body carries the full canonical agent content per kind (role definition, tool discipline, contract, evidence-source order, output format, the works). The frontmatter still comes from `binding`; the body is sourced from a new field on `BindingResolved` (or directly from `binding.SystemPromptTemplatePath` resolved against a new `<project>/.tillsyn/agents/` directory or an embedded asset).

**WHY.** Closes the "bundle stub is empty" gap (C.5). Without this, the agent has no behavior definition regardless of any other change.

**Two options for the source:**

- **Option D.1.a — Embedded defaults shipped in the binary.** Add a `//go:embed` directory `internal/templates/builtin/agents/<name>.md` carrying default content for the 9 canonical agents. `Render` looks up the embedded file by `binding.AgentName` and writes its body into the bundle. **Pros:** zero per-project setup, agents work on a fresh clone, single source of truth in-tree, dogfood-able with no additional infrastructure. **Cons:** dev must edit Go-tracked files to update agent prompts (less ergonomic than editing `.tillsyn/agents/<name>.md`). Mitigation: ship the embedded defaults but layer per-project override on top (D.1.c).
- **Option D.1.b — Per-project `.tillsyn/agents/<name>.md`.** Wire `templates.AgentBinding.SystemPromptTemplatePath` (`schema.go:556`, validator at `load.go:1031-1055`) through to `BindingResolved`, then have `Render` read `<project>/<system_prompt_template_path>` and inject into the body. **Pros:** matches the dev's stated mental model (per-project `.tillsyn/` ownership). **Cons:** requires per-project file seeding (no `till init` or `till bootstrap` exists today per AGENT_ARCHITECTURE_TRUTH.md §3) — fresh clones break until files are checked in.
- **Option D.1.c — Both, with override semantics.** Embedded default in-tree (Option a); if `<project>/.tillsyn/agents/<name>.md` exists, it overrides. The dev gets a working out-of-the-box default plus the per-project ergonomics knob. This is the recommendation. AGENT_ARCHITECTURE_TRUTH.md §3 confirms there is no `<project>/.tillsyn/agents/` content or seeding logic anywhere today, so the override path is a clean greenfield addition.

**COST.** Option D.1.c: roughly 3 droplets — one to add `//go:embed` defaults + the 9 agent-content MD files (~40 LOC + content), one to wire `BindingResolved.SystemPrompt` (or rename `SystemPromptTemplatePath` to a resolved-content field) and modify `assembleAgentFileBody` (~30 LOC), one to add the per-project override read with worktree-escape validation (~50 LOC + tests). Plus the agent-content MDs themselves (the substantive prompts), which are dev-authored content not Go LOC.

**RISK.** Migrating the existing `~/.claude/agents/<name>.md` content into the in-tree embedded MDs: the dev's local files become non-load-bearing the moment `--bare` is on (already true), so the migration is a one-time copy-paste with no on-disk-state risk. CI / fresh-clone contributors get working agents on first try after this lands.

### D.2 Fix change 2 — strip / normalize frontmatter `model:`

**WHAT.** Add YAML-frontmatter parsing to `assembleAgentFileBody` (or to a wrapper that runs before it). When the binding has a non-nil `Model` AND the embedded/override MD's frontmatter contains `model:`, strip the file's `model:` and emit only the binding's `model:` in the rendered frontmatter. `agents.toml` (post-Drop-4c.6) becomes the runtime authority; the in-MD frontmatter is decoration.

**WHY.** Per the dev's question quoted in `SPAWN_PIPELINE_TRUTH.md` §D: clear precedence rule needed. Argv `--model <m>` already wins over agent-frontmatter `model:` per Claude Code's documented "CLI flag overrides setting" pattern (sub-agents docs §"Run the whole session as a subagent" — *"The CLI flag overrides the setting if both are present."*), so functionally argv `--model` is already authoritative when emitted. But: (a) the agent-frontmatter `model:` adds noise the dev wants gone, (b) when `agents.toml` does NOT define `model =` for a kind, `--model` will not be emitted (per `argv.go:106-108` only emits when non-nil), so the frontmatter would silently take effect. Stripping it makes `agents.toml` AUTHORITATIVE rather than "default-fallback."

**WHERE.** A new pure helper `transformAgentFrontmatter(body string, binding BindingResolved) string` in `render.go` (or a new `frontmatter.go` in the same package). Called from `assembleAgentFileBody` after the embedded/override body is loaded but before disk write.

**YAML helper.** `gopkg.in/yaml.v3` is the conventional Go choice but adds a dependency. For a minimal frontmatter-only parser (the bundle MDs are dev-authored — we control the shape), a regex-based strip is sufficient and zero-dep. Recommendation: hand-rolled regex strip plus structural `name:` / `model:` line replacement. ~25 LOC + tests.

**Implementation shape:**

```go
// stripFrontmatterField removes `<field>: ...` lines (single-line scalar values
// only) from a YAML frontmatter block. Returns the body unchanged if no
// frontmatter delimiter is present.
func stripFrontmatterField(body, field string) string {
    // 1. Locate the leading "---\n" delimiter.
    // 2. Locate the closing "---\n" delimiter.
    // 3. Within that block, drop any line matching ^<field>:.*$.
    // 4. Return the modified block + remaining body.
}
```

**WHY this matters for isolation specifically.** This is not strictly an isolation closure — it's a separate hardening item. But: leaving frontmatter `model:` in the bundle creates a path where dev-machine-local edits to the embedded MD content (after override at `<project>/.tillsyn/agents/`) could accidentally pin a model that conflicts with `agents.toml`. Stripping at render time prevents the desync.

**COST.** ~25 LOC parser + 3-4 unit tests. Negligible.

**RISK.** Misparse on multi-line YAML blocks. Mitigation: assert in tests that frontmatter MDs only use single-line scalar values for `model:` / `name:` / `description:` / `tools:` / etc. The conventional Claude Code agent format uses single-line scalars throughout (per <https://code.claude.com/docs/en/sub-agents#supported-frontmatter-fields>).

### D.3 Fix change 3 — defense-in-depth env vars

**WHAT.** Set the following env vars on `cmd.Env` for every spawn, in addition to the existing closed POSIX baseline:

- `CLAUDE_CODE_DISABLE_BACKGROUND_TASKS=1` — prevents the spawned subagent from forking background subagents that might escape the per-spawn permission gate. Per <https://code.claude.com/docs/en/sub-agents#run-subagents-in-foreground-or-background>.
- `CLAUDE_CODE_FORK_SUBAGENT=0` (explicit zero) — prevents enable-fork-mode if the binding's env declares it.
- `DISABLE_AUTOUPDATER=1` — prevents the spawned `claude` from running its own update logic mid-spawn.
- `DISABLE_TELEMETRY=1` — privacy + reproducibility.
- Optional: `CLAUDE_CODE_MAX_OUTPUT_TOKENS` — cap response size if the dev wants budget guard.

These are **defense-in-depth**, not the primary isolation gate (`--bare` + `--strict-mcp-config` + `--setting-sources ""` are the primary gates per A.9). They close failure modes where future Claude Code versions might add a new auto-discovery path that `--bare` doesn't cover yet.

**WHERE.** Extend `closedBaselineEnvNames` in `env.go:37-58` with a new struct that carries (name, value) pairs for the always-on env injections (vs the current name-only list that does `os.LookupEnv` passthrough). New code in `assembleEnv` writes them unconditionally — they're literal-valued, not orchestrator-inherited.

**COST.** ~15 LOC + 4 unit tests.

**RISK.** A future `claude` flag may want one of these as a non-default value. Mitigation: make them overridable by `binding.Env` (binding wins over the always-on injection — symmetric to today's binding-vs-baseline rule at `env.go:108-117`).

### D.4 Fix change 4 — fail-loud post-render assertion

**WHAT.** After `Render` completes, run a self-check that:

1. The bundle's `agents/<name>.md` body is non-empty AND non-trivial (more than N lines, or contains a sentinel marker like a required role string from `binding.AgentName`).
2. The bundle's `settings.json` `permissions.allow` is exactly `binding.ToolsAllowed` plus persisted grants — no extras, no missing entries.
3. The bundle's `.mcp.json` contains exactly one server (`tillsyn`).
4. The bundle's `system-prompt.md` carries `task_id: <item.ID>` and `project_dir: <project.RepoPrimaryWorktree>`.

**WHY.** Catches accidental render-pipeline regressions where a future change leaves the bundle stub thin. Today's tests cover much of this (`render_test.go:113-138` checks files exist + non-zero size; `render_test.go:335-364` checks frontmatter), but there's no assertion that the BODY is substantive. Without this guard, a future `assembleAgentFileBody` change could re-introduce the empty-body bug silently.

**WHERE.** New `validateBundle(bundle, item, binding) error` after the five `render*` calls in `Render` (after line 176). Returns a wrapped error so the bundle rolls back on failure.

**COST.** ~50 LOC validator + 4-6 tests.

**RISK.** Brittle if the sentinel-checking is too tight. Mitigation: assert on minimum body length (e.g. > 200 chars after frontmatter) and frontmatter completeness rather than exact-string matches.

### D.5 Fix change 5 — kill the misleading doc-comments

**WHAT.** Edit:

- `render.go:307-319` — remove the "canonical templates at `~/.claude/agents/<name>.md` remain the source of truth" claim. Replace with the truth: "Bundle agent file is the SOLE source under `--bare`. Body is sourced from embedded default + per-project override at `<project>/.tillsyn/agents/<name>.md`."
- `SPAWN_PIPELINE.md:24-31` — keep the two-paths section but add a note: "Tillsyn ships `--bare`; under `--bare` Path B is disabled by Claude Code itself. The bundle's plugin (Path A) is the SOLE source for agents, settings, and MCP. The two-paths model is informational for non-bare callers; Tillsyn's adapters never ship without `--bare`."
- The line in `render.go:340-364` body that says "Behavior loaded from the canonical ... template at the system-installed plugin path" — DELETE entirely; replace with the actual canonical body content per D.1.

**WHY.** Misleading doc-comments are the original cause of the dev's confusion ("agents ARE NOT supposed to be running from the 'systems' agents list"). The code's comments say one thing, the actual `--bare` argv does another.

**COST.** Doc-only. <30 LOC of edits.

**RISK.** None.

### D.6 Net summary of the fix

The dev's hard requirement is "ONLY tillsyn defined and allowed and managed files and ONLY the specific one that is picked for that particular cascade agent from the agents.toml in the project's .tillsyn/ dir." Closing it requires **all five** of D.1, D.2, D.3, D.4, D.5 — each closes a distinct surface:

| Fix     | Closes                                                                |
| ------- | --------------------------------------------------------------------- |
| D.1     | Bundle stub is empty → bundle carries full agent content              |
| D.2     | Frontmatter `model:` desync → `agents.toml` is sole runtime authority |
| D.3     | Future-proof against new auto-discovery paths Claude Code may add     |
| D.4     | Render-pipeline regression that leaves bundle thin                    |
| D.5     | Misleading doc-comments that confuse future dev / contributor reads   |

The **primary** gate is already in place via the existing argv (`--bare` + `--setting-sources ""` + `--strict-mcp-config` + `--plugin-dir`); the bug is purely that the bundle is empty (D.1) plus residual confusion (D.5). D.2/D.3/D.4 are hardening, not isolation closure.

**No new argv flag is needed.** The argv shape Tillsyn ships today is correct for isolation. The fix is in the bundle body + the doc-comments + a few env-var injections.

---

## E. Verification strategy

### E.1 Test entry points

`internal/app/dispatcher/cli_claude/render/render_test.go` — render-layer unit tests. Add new test cases here.
`internal/app/dispatcher/cli_claude/adapter_test.go` — argv-layer tests. Already covers the `--bare` + `--setting-sources ""` + `--strict-mcp-config` + `--plugin-dir` triplet (lines 78-91); extend to assert env-var injections from D.3.

### E.2 New test patterns

**E.2.1 — Bundle body non-emptiness pin.** New `TestRenderAgentFileBodyContainsCanonicalContent` — assert the rendered agent file body contains expected role-definition tokens (e.g. for a `go-builder-agent` binding, the rendered body must contain `Builder`, `mage`, `TDD`, etc.). Use embedded fixture content as the source of truth.

**E.2.2 — Sentinel-injection isolation test.** New `TestSpawnDoesNotInheritUserAgentDefinition` — integration test that:

1. Creates a tempdir.
2. Writes `<tempdir>/.claude/agents/go-builder-agent.md` with a sentinel string `SENTINEL_USER_AGENT_INHERITED_LEAK`.
3. Writes `<tempdir>/.claude/CLAUDE.md` with `SENTINEL_USER_CLAUDE_MD_LEAK`.
4. Writes `<tempdir>/.claude/agents/go-builder-agent.md` with a sentinel `SENTINEL_PROJECT_AGENT_LEAK`.
5. Sets `cmd.Env` to point `HOME=<tempdir>` and `XDG_CONFIG_HOME=<tempdir>/.config`.
6. Invokes the bundle render.
7. Asserts the rendered bundle contents (system-prompt.md + agent stub + settings.json + .mcp.json) contain none of the sentinels.

This pins the render layer against any future regression that secretly reads from project-or-user-local files.

**E.2.3 — Argv hardening test.** Extend `TestBuildCommandArgvShapeMinimal` to assert the four critical isolation flags are present in EVERY spawn:

```go
expectFlagPresent(t, cmd.Args, "--bare", "")
expectFlagPresent(t, cmd.Args, "--setting-sources", "")  // empty value!
expectFlagPresent(t, cmd.Args, "--strict-mcp-config", "")
expectFlagPresent(t, cmd.Args, "--plugin-dir", expectedBundlePluginPath)
```

Already covered by `adapter_test.go:78-91`. Add a negative assertion: `expectFlagAbsent(t, cmd.Args, "--setting-sources", "user")` — guard against a future "helpful" change that adds back user-level settings.

**E.2.4 — Env hardening test (post-D.3).** New `TestEnvCarriesIsolationEnvVars` — assert `cmd.Env` contains `CLAUDE_CODE_DISABLE_BACKGROUND_TASKS=1`, `CLAUDE_CODE_FORK_SUBAGENT=0`, `DISABLE_AUTOUPDATER=1`, `DISABLE_TELEMETRY=1`.

**E.2.5 — Post-render validator test (post-D.4).** New `TestRenderValidatorFailsLoudOnEmptyAgentBody` — feed the validator a bundle with a deliberately-empty `agents/<name>.md` body, assert it returns the documented error.

**E.2.6 — Frontmatter strip test (post-D.2).** New `TestStripFrontmatterFieldRemovesModelLine` and `TestRenderAgentFileEmitsBindingModelOverFrontmatter` — given an embedded MD with `model: opus` in frontmatter, assert the rendered bundle's frontmatter has only one `model:` line and it matches `binding.Model`.

### E.3 CI gating

The post-D.4 validator runs on every render — failures bubble through `BuildSpawnCommand`'s error path and prevent the spawn from launching. The integration test (E.2.2) gates merge in `mage ci`. The argv negative-assertion (E.2.3 absent-assertion) prevents a future drop from accidentally re-introducing inheritance via a "helpful" flag change.

### E.4 Manual smoke test (post-merge)

After D.1+D.2+D.3+D.4+D.5 land:

1. `git stash` your `~/.claude/agents/go-builder-agent.md` (or rename it).
2. `git stash` your `~/.claude/CLAUDE.md`.
3. Run a real Tillsyn spawn against a build action item.
4. Inspect the spawn's stream-json output for the `system_init` event — confirm it lists exactly the bundle plugin (no system plugins, no user agents).
5. Inspect the bundle dir before terminal-state cleanup — confirm `agents/<name>.md` body is the embedded content, not a stub redirect.

---

## F. Frontmatter `model:` strip — implementation sketch

Per D.2, the natural place is `render.go`. Two suborderings work:

**Order A.** Embedded MD content is loaded; pass through `transformAgentFrontmatter(body, binding)`; concatenate with binding-derived `tools:` / `disallowedTools:` / etc.

**Order B.** Embedded MD content is split into (frontmatter, body). Frontmatter is parsed into a struct; struct fields conflict-resolved against binding (binding wins for `model`, `tools`, etc.); struct re-serialized to YAML; concatenated with body.

**Recommendation:** Order A is simpler and zero-dep (regex strip + line append). Order B is more correct but pulls in `gopkg.in/yaml.v3`.

Implementation outline (Order A):

```go
// At render-time, after loading the MD body for this kind:
body := loadEmbeddedAgentBody(binding.AgentName)  // new helper, D.1
body = stripFrontmatterField(body, "model")
body = stripFrontmatterField(body, "tools")           // optional — Tillsyn manages this
body = stripFrontmatterField(body, "disallowedTools") // optional — Tillsyn manages this

// Then inject Tillsyn-managed frontmatter values:
body = injectFrontmatterField(body, "model", derefString(binding.Model))
if len(binding.ToolsAllowed) > 0 {
    body = injectFrontmatterField(body, "allowedTools", strings.Join(binding.ToolsAllowed, ", "))
}
if len(binding.ToolsDisallowed) > 0 {
    body = injectFrontmatterField(body, "disallowedTools", strings.Join(binding.ToolsDisallowed, ", "))
}
```

`stripFrontmatterField` and `injectFrontmatterField` operate only on the leading `---\n...---\n` block (idempotent if the field is absent). ~50 LOC total, zero deps. Tests pin behavior on:

- Frontmatter-absent input (returns unchanged).
- Field-absent in frontmatter (strip is no-op; inject appends).
- Field-present in frontmatter (strip removes; inject appends).
- Multi-line scalar values (assert these are not used in the embedded MDs; tests forbid).

---

## G. Net actionable summary

For Drop 4c.6 (or a follow-on drop the orchestrator scopes), land:

1. **D.1.c — Embedded default agent MDs + per-project override** (3 droplets + content authoring).
2. **D.2 — Strip + inject frontmatter `model:` / `tools:`** (1 droplet, ~50 LOC).
3. **D.3 — Inject defense-in-depth env vars** (1 droplet, ~15 LOC).
4. **D.4 — Post-render validator** (1 droplet, ~50 LOC).
5. **D.5 — Doc-comment corrections** (folded into D.1 droplets).

Verification per E.1-E.4. Do NOT add new argv flags — the existing `--bare` + `--setting-sources ""` + `--strict-mcp-config` + `--plugin-dir` is already the right shape.

---

## Hylla Feedback

- **Query:** `hylla_search_keyword(query="SystemPromptTemplatePath", fields=["content"])` and `(query="SystemPromptTemplate", fields=["docstring", "content"])` — both returned zero results.
- **Missed because:** Hylla appears not to index `templates/schema.go` field declarations under the keyword search path used here. The previous research at AGENT_ARCHITECTURE_TRUTH.md §2.3 cited `schema.go:556` and `load.go:1031-1055` for this field, so the symbol exists in the tree — Hylla's keyword index just isn't catching it from these tokens.
- **Worked via:** Reused `AGENT_ARCHITECTURE_TRUTH.md` § 2.3 as the source of the file-line cites (research-deliverable cross-reference, not direct Hylla).
- **Suggestion:** Symbol-level field name search would help here — `templates.AgentBinding.SystemPromptTemplatePath` is the kind of dotted-path identifier the dev queries by, but the keyword search seems to tokenize on whole-word boundaries that miss field-declaration sites.

- **Query:** Bash invocations of `find` / `grep` / `ls` were uniformly denied by the agent permission gate, forcing a reliance on `Read`/`Glob`-equivalent paths through the assistant tool surface and on the prior research deliverable for cross-references. This is consistent with prior AGENT_ARCHITECTURE_TRUTH.md and SPAWN_PIPELINE_TRUTH.md sessions; the recommendation to relax research-agent Bash policy to allow `git grep`, `find -name`, `ls -la` against the project root would close this gap permanently.

- **Ergonomic gripe:** The `hylla_search_keyword` "deduped" return shape is helpful but offered zero diagnostic on WHY a query returned empty. A "reason" field (e.g. "no symbols matched", "matched but filtered by visibility_mode", "matched in tests but hide_tests is on") would let me decide whether to broaden the search or fall back to `Read` immediately rather than re-trying with three different field combinations.
