# Drop 4c — Theme F.7 Spawn-Pipeline-Core PLAN (F.7.1–F.7.16)

**Drop:** 4c (pre-Drop-5 refinement drop).
**Theme:** F.7 — spawn pipeline redesign, replaces 4a.19 stub wholesale.
**Plan scope:** F.7.1 through F.7.16 (16 droplets — spawn-pipeline-core surface).
**Out of scope (sibling plans):** F.7.17 (CLI adapter seam, separate planner) + F.7.18 (context aggregator, separate planner). Cross-plan dependencies declared inline below.
**Author:** go-planning-agent.
**Date:** 2026-05-04.

---

## Architectural Decisions Locked From SKETCH

These are non-negotiable for any droplet author working under this plan. Source citations point at SKETCH lines or the spawn architecture memory.

1. **Per-spawn ephemeral bundle, NOT persistent global config.** SKETCH §F.7.1; memory §1-§2. `os.MkdirTemp` per spawn, `defer os.RemoveAll` on terminal-state.
2. **Bundle root knob has two values only:** `os_tmp` (default) | `project`. SKETCH §F.7.1, §F.7.7. No third option.
3. **Bundle layout phasing — cross-CLI shell vs CLI-specific subtree** per planner-review §6.1: `<bundle>/manifest.json`, `<bundle>/system-prompt.md`, `<bundle>/system-append.md`, `<bundle>/stream.jsonl`, `<bundle>/context/` are cross-CLI; `<bundle>/plugin/...` (claude) and future `<bundle>/codex_home/...` (codex) are adapter-owned.
4. **Tool-gating two-layer strategy.** Memory §5; SKETCH §F.7.2. `settings.json` deny rules AUTHORITATIVE (Layer B); agent-frontmatter `disallowedTools` mirrors B for human readability (Layer A); `--allowed-tools`/`--disallowed-tools` CLI flags SKIPPED for typical kinds (probe-grounded). `--tools` flag emitted only when kind wants engine-level minimization.
5. **Conditional argv flag emission via `*int` / `*float64` / `*string`.** Memory §3; SKETCH §F.7.3. Priority cascade: `CLI flag > MCP arg > TUI override > template TOML default > absent`. Spawn.go emits ONLY when value resolves through the cascade.
6. **Settings precedence locked.** Memory §4. `--settings <file> --setting-sources ""` makes Tillsyn's per-spawn `settings.json` the SOLE settings source — user/project/local settings ignored.
7. **Permission-denied → TUI handshake AT TERMINAL EVENT, not real-time.** SKETCH §F.7.5; memory §6.5. Real-time mid-stream variant explicitly OUT OF SCOPE for Drop 4c.
8. **Dispatcher monitor stays CLI-agnostic** per planner-review §6.3. F.7.4's stream parser is the **claude adapter's** implementation detail; the dispatcher monitor consumes adapter-returned `StreamEvent` values from the F.7.17 seam.
9. **F.7.10 ONLY removes the hardcoded `hylla_artifact_ref` line from the prompt body.** It does NOT remove `domain.Project.HyllaArtifactRef` or `project.metadata.hylla_artifact_ref` — those legitimately exist for adopter-local templates that opt into Hylla MCP.
10. **Commit/push gates default OFF** via `dispatcher_commit_enabled` + `dispatcher_push_enabled` project metadata pointer-bools (nil-means-disabled, Drop 4a Wave 4a.25 precedent). Default template ships gate sequence with both gates listed but toggle-disabled until dogfood proves them safe.
11. **`mage install` is dev-only.** Per project CLAUDE.md "Build Verification" §3 — every droplet acceptance criterion uses `mage ci` for verification, NEVER `mage install`. No exceptions.
12. **No migration logic in Go.** Pre-MVP rule. SQLite schema additions (F.7.5 `permission_grants`) are dev-fresh-DB; F.7.5 droplet ships the table-creation DDL inside the storage layer init path, not a one-shot migration script.

---

## Hard Prereqs (Cross-Plan + Cross-Drop)

This plan blocks on artifacts authored in sibling plans. Builders cannot start any F.7.X droplet until its declared prereqs are landed.

| Prereq | Provided by | Required by F.7.X droplets |
|---|---|---|
| Drop 4a closed (dispatcher exists; spawn.go stub) | merged | All F.7 |
| Drop 4b closed (gate runner, lock manager, post-build pipeline) | merged | F.7.13, F.7.14, F.7.16 |
| **Schema-1**: per-binding `command`, `args_prefix`, `env`, `cli_kind` fields on `AgentBinding` + validators | **F.7.17 plan (sibling)** | F.7.3 (argv emission consumes `command` + `args_prefix` + `env`); F.7.6 (uses `cli_kind` to pick the plugin-list parser); F.7.12 (commit-agent binding has `cli_kind`) |
| **Schema-2**: `Context` sub-struct on `AgentBinding` | **F.7.18 plan (sibling)** | None in F.7.1–F.7.16; F.7.3's `system-append.md` emission consumes the resolved context bundle from F.7.18's aggregator engine |
| **Schema-3**: `Tillsyn` top-level struct (`max_context_bundle_chars`, `max_aggregator_duration`) | **F.7.18 plan (sibling)** | None in F.7.1–F.7.16 |
| `CLIKind` enum + `CLIAdapter` interface scaffold + `BindingResolved` + `BundlePaths` + `StreamEvent` + `ToolDenial` types | **F.7.17 plan (sibling)** | F.7.1 (bundle paths reference `BundlePaths` shape); F.7.3 (claude adapter implements `BuildCommand`); F.7.4 (claude adapter implements `ParseStreamEvent` + `ExtractTerminalReport`); F.7.5 (TUI handshake consumes `ToolDenial`); F.7.8 (orphan scan reads `manifest.json cli_kind`) |
| `claudeAdapter` struct skeleton (registered with adapter registry) | **F.7.17 plan (sibling)** | F.7.3, F.7.4 land their logic INSIDE the claude adapter |

**Sequencing rule:** the F.7.17 schema-1 droplet + adapter scaffold droplet MUST land before ANY of F.7.1–F.7.16 except F.7.10 (independent — pure prompt-body deletion) and F.7.9 (independent — domain metadata fields). The plan-QA twins on this plan and the F.7.17/F.7.18 plans must align cross-plan ordering.

---

## Cross-Droplet Sequencing Diagram (Text DAG)

```
                              [Drop 4b merged]
                                    │
                                    ▼
            ┌───────────── F.7.10 (drop hylla_artifact_ref) ─────────────┐
            │   independent — pure prompt-body deletion                  │
            │                                                            │
            │   F.7.9 (action-item metadata fields) ─────────────────────┤
            │   independent — domain.ActionItem.Metadata extension       │
            └────────────────────────────────────────────────────────────┘
                                    │
                  ┌─── F.7.17 schema-1 + adapter scaffold (sibling plan) ───┐
                  │     CLIAdapter interface, claudeAdapter struct           │
                  │     Schema-1 fields (command, args_prefix, env,         │
                  │     cli_kind) on AgentBinding                            │
                  └──────────────────────────────────────────────────────────┘
                                    │
       ┌────────────────────────────┼────────────────────────────┐
       ▼                            ▼                            ▼
  F.7.2 (TOML schema:         F.7.1 (per-spawn temp           F.7.6 (system-plugin
  tools_allowed/disallowed,   bundle lifecycle —              pre-flight check)
  sandbox.*, system_prompt    MkdirTemp, manifest.json,
  _template_path)             defer RemoveAll)
       │                            │                            │
       │                            ▼                            │
       │                    F.7.7 (gitignore auto-add            │
       │                    when spawn_temp_root="project")      │
       │                            │                            │
       └────────────┬───────────────┴────────────────────────────┘
                    ▼
           F.7.3 (headless argv emission — claude adapter's
           BuildCommand implementation; uses Schema-1 command/
           args_prefix/env; uses F.7.2 tool-gating fields;
           uses F.7.1 bundle paths)
                    │
                    ▼
           F.7.4 (stream-JSON monitor parser — claude adapter's
           ParseStreamEvent + ExtractTerminalReport;
           dispatcher monitor stays CLI-agnostic)
                    │
                    ▼
           F.7.5 (permission-denial → TUI handshake +
           SQLite permission_grants table with cli_kind
           column; consumes ToolDenial from F.7.4)
                    │
                    ▼
           F.7.8 (crash-recovery / orphan scan on Tillsyn
           startup — reads manifest.json cli_kind; PID
           liveness check)
                    │
                    ▼
           F.7.12 (commit-agent integration via new spawn
           pipeline — uses F.7.1 bundle, F.7.3 argv,
           F.7.4 monitor)
                    │
                    ▼
           F.7.15 (project-metadata toggles
           dispatcher_commit_enabled +
           dispatcher_push_enabled)
                    │
            ┌───────┴───────┐
            ▼               ▼
       F.7.13 (commit   F.7.14 (push gate
       gate impl)       impl)
            │               │
            └───────┬───────┘
                    ▼
           F.7.16 (default template [gates.build]
           expansion: ["mage_ci", "commit", "push"]
           with toggles default-off)
                    │
                    ▼
           F.7.11 (Tillsyn architecture docs —
           references spawn architecture memory as
           canonical source; can land any time after
           F.7.1–F.7.10 stabilize)
```

**Critical-path summary:**
- F.7.10 + F.7.9 land first/parallel (independent).
- Schema-1 + adapter scaffold (sibling plan) gates everything else.
- F.7.1, F.7.2, F.7.6 land in parallel after the scaffold.
- F.7.7 follows F.7.1 (`spawn_temp_root` knob lives on F.7.1).
- F.7.3 → F.7.4 → F.7.5 → F.7.8 form the spawn-pipeline backbone (sequential).
- F.7.12 → F.7.15 → (F.7.13 ‖ F.7.14) → F.7.16 form the commit/push integration.
- F.7.11 (docs) lands last but can stage incrementally.

---

## Per-Droplet Specifications

### F.7.1 — Per-spawn temp bundle lifecycle

**Goal:** establish per-spawn ephemeral bundle directory with `os.MkdirTemp` + `manifest.json` + `defer os.RemoveAll`, configurable via `spawn_temp_root = "os_tmp" | "project"` knob; bundle layout follows the cross-CLI shell + CLI-specific subtree split per planner-review §6.1.

**Builder model:** opus.

**Hard prereqs:**
- F.7.17 schema-1 droplet (`BundlePaths` type defined in `internal/app/dispatcher/`).
- F.7.17 adapter scaffold (`CLIAdapter` interface defined; `claudeAdapter` registered).
- `[tillsyn]` block partial schema for `spawn_temp_root` field — F.7.1 OWNS this field on the `[tillsyn]` table; the F.7.18 plan owns `max_context_bundle_chars` + `max_aggregator_duration` on the same table. Cross-plan ordering: F.7.18 Schema-3 droplet adds the `Tillsyn` struct + `Template.Tillsyn` field; F.7.1 extends the same struct with `SpawnTempRoot string` (TOML tag `spawn_temp_root`).

**Files to edit/create:**
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/spawn_bundle.go` (new — `Bundle` type, `MkdirBundle`, `RemoveBundle`).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/spawn_manifest.go` (new — `Manifest` struct, JSON-marshalable).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/spawn_bundle_test.go` (new).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/spawn_manifest_test.go` (new).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/templates/schema.go` (extend the F.7.18-Schema-3-introduced `Tillsyn` struct with `SpawnTempRoot string` — coordinate with F.7.18 plan).

**Packages locked:** `internal/app/dispatcher`, `internal/templates`.

**Acceptance criteria:**
- [ ] `MkdirBundle(spawnID, tempRoot, projectWorktree string) (Bundle, error)` creates a directory under `os.TempDir()/tillsyn/<spawnID>/` when `tempRoot == "os_tmp"`; under `<projectWorktree>/.tillsyn/spawns/<spawnID>/` when `tempRoot == "project"`. Both modes return a populated `Bundle` value with `Root`, `ManifestPath`, `StreamLogPath`, `SystemPromptPath`, `SystemAppendPath`, `ContextDir` fields.
- [ ] `Bundle` struct exposes `BundlePaths` (the F.7.17-defined cross-CLI handle) via accessor; CLI-specific subdirs (e.g. `<Root>/plugin/`) are NOT pre-created by `MkdirBundle` — adapters create them in their `BuildCommand`.
- [ ] `Manifest` struct has fields: `SpawnID string`, `ActionItemID string`, `Kind domain.Kind`, `CLIKind string` (cross-plan dependency on F.7.17 schema-1; populated by the adapter), `ClaudePID int` (zero until cmd.Start succeeds), `StartedAt time.Time`, `BundlePath string`, `Paths []string` (mirrors `action_item.paths`).
- [ ] `WriteManifest(b Bundle, m Manifest) error` writes JSON to `<Root>/manifest.json`; fsync on close.
- [ ] `ReadManifest(bundlePath string) (Manifest, error)` round-trips JSON; returns wrapped `os.ErrNotExist` when file missing.
- [ ] `RemoveBundle(b Bundle) error` is idempotent; `defer`-friendly; returns nil on second call after success.
- [ ] Empty `tempRoot` defaults to `"os_tmp"`; empty `projectWorktree` rejected when `tempRoot == "project"`.
- [ ] Closed-enum reject: `tempRoot` values other than `"os_tmp"` / `"project"` return wrapped `ErrInvalidSpawnInput`.
- [ ] Schema validator on `tillsyn.spawn_temp_root` rejects values outside the closed set at template load.

**Test scenarios (happy + edge):**
- `MkdirBundle("abc-123", "os_tmp", "")` creates `os.TempDir()/tillsyn/abc-123/`.
- `MkdirBundle("abc-123", "project", "/Users/dev/proj")` creates `/Users/dev/proj/.tillsyn/spawns/abc-123/`.
- `MkdirBundle("abc-123", "project", "")` returns wrapped `ErrInvalidSpawnInput`.
- `MkdirBundle("abc-123", "tmpfs", "")` returns wrapped `ErrInvalidSpawnInput`.
- Round-trip: `WriteManifest(b, m1); m2, _ := ReadManifest(b.ManifestPath); reflect.DeepEqual(m1, m2)`.
- `RemoveBundle` called twice returns nil both times.
- Falsification cross-check (Attack 5): `ClaudePID` field is `0` in `WriteManifest`'s initial call; spawn caller MUST update via `UpdateManifestPID(b, pid)` AFTER `cmd.Start()` returns successfully.

**Falsification mitigations to bake in:**
- (Attack 5 from QA Falsification §) `ClaudePID` is zero until `cmd.Start()` succeeds — orphan scan (F.7.8) treats PID-zero manifests as "spawn not yet started" and leaves them alone.
- Bundle root in `project` mode requires non-empty worktree; rejecting empty prevents accidentally creating `.tillsyn/spawns/` at filesystem root.

**Verification gates:** `mage test-pkg ./internal/app/dispatcher` + `mage ci` + per-droplet QA twins (proof + falsification, parallel).

**Out of scope:**
- Writing `system-prompt.md` / `system-append.md` content (F.7.3 owns).
- `plugin/` subdirectory contents (claude adapter owns inside `BuildCommand`).
- Writing `stream.jsonl` (F.7.4 captures from running cmd).
- Cleanup-on-terminal-state hook (already exists in Drop 4a Wave 2.7 cleanup hook; F.7.1 only ensures `RemoveBundle` is idempotent + safe).
- Bundle root materialization for codex (F.7.17 plan / Drop 4d).

---

### F.7.2 — TOML template schema widening for tool-gating + sandbox fields

**Goal:** add per-binding tool-gating + sandbox fields to `AgentBinding` so the spawn pipeline can render `settings.json` permission rules + sandbox config from template TOML.

**Builder model:** opus.

**Hard prereqs:**
- F.7.17 schema-1 droplet landed (so this droplet extends the same `AgentBinding` struct, not a stale copy).

**Files to edit/create:**
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/templates/schema.go` (extend `AgentBinding`; add `SandboxFilesystem`, `SandboxNetwork` sub-structs).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/templates/load.go` (extend `validateAgentBinding` chain with `validateAgentBindingSandbox`).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/templates/schema_test.go` (extend).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/templates/load_test.go` (extend).

**Packages locked:** `internal/templates`.

**Acceptance criteria:**
- [ ] `AgentBinding` gains the following fields with explicit TOML tags:
  - `ToolsAllowed []string` (TOML tag `tools_allowed`) — Layer A frontmatter `allowedTools`.
  - `ToolsDisallowed []string` (TOML tag `tools_disallowed`) — Layer A frontmatter `disallowedTools`; ALSO mirrored into Layer B `settings.json` `permissions.deny` patterns.
  - `SystemPromptTemplatePath string` (TOML tag `system_prompt_template_path`) — relative to project worktree; resolved at spawn time.
  - `ToolsEngineMinimal []string` (TOML tag `tools_engine_minimal`) — when non-empty, emitted as `--tools "Read,Grep,Glob"` flag for engine-level minimization. Distinct from `ToolsAllowed` (which renders to settings.json allow patterns).
- [ ] New `Sandbox` struct on `AgentBinding` (TOML tag `sandbox`), with two sub-tables:
  - `Filesystem SandboxFilesystem` (TOML tag `filesystem`):
    - `AllowWrite []string` (TOML tag `allow_write`).
    - `DenyRead []string` (TOML tag `deny_read`).
  - `Network SandboxNetwork` (TOML tag `network`):
    - `AllowedDomains []string` (TOML tag `allowed_domains`).
    - `DeniedDomains []string` (TOML tag `denied_domains`).
- [ ] All new fields use explicit TOML tags so strict-decode (`DisallowUnknownFields`) automatically rejects unknown keys (per planner-review §A3.c).
- [ ] `validateAgentBindingSandbox` rejects `AllowWrite` paths that escape the project worktree (resolved via `filepath.Clean` + ancestry check). Falsification mitigation #9.
- [ ] `validateAgentBindingSandbox` rejects empty strings inside any of the four slice fields.
- [ ] Validator rejects `SystemPromptTemplatePath` containing `..` segments (path traversal guard).
- [ ] Backward-compat: `AgentBinding` zero-value (no `tools_*`, no `sandbox`, no `system_prompt_template_path`) loads + validates clean — existing templates from Drop 3 don't break.
- [ ] Unit tests cover happy path, every reject case, AND a strict-decode unknown-key rejection on the `[sandbox]` sub-table.

**Test scenarios (happy + edge):**
- Happy: `[agent_bindings.build] tools_allowed = ["Read", "Edit", "Bash(mage *)"]`, `tools_disallowed = ["WebFetch"]`, `[agent_bindings.build.sandbox.filesystem] allow_write = ["./src"]` — loads + validates.
- Reject: `allow_write = ["/etc"]` (escapes worktree) → `ErrInvalidAgentBinding`.
- Reject: `allow_write = ["../sibling"]` (traversal) → `ErrInvalidAgentBinding`.
- Reject: `allow_write = ["", "./src"]` (empty entry) → `ErrInvalidAgentBinding`.
- Reject: `system_prompt_template_path = "../escape.md"` → `ErrInvalidAgentBinding`.
- Reject: `[agent_bindings.build.sandbox.filesystem] unknown_key = "x"` → strict-decode rejects.
- Backward-compat: default-go template loads clean with no `tools_*` / `sandbox` blocks.

**Falsification mitigations to bake in:**
- Attack #9: filesystem allowWrite escape — resolved-path ancestry check at template-load.
- Strict-decode unknown-key rejection on sandbox sub-tables — verified with explicit unit test.

**Verification gates:** `mage test-pkg ./internal/templates` + `mage ci` + per-droplet QA twins.

**Out of scope:**
- Rendering settings.json from these fields (F.7.3 owns the rendering).
- Rendering agent-frontmatter from these fields (F.7.3 owns).
- Per-binding `command`, `args_prefix`, `env`, `cli_kind` (F.7.17 plan owns — schema-1).
- `[agent_bindings.<kind>.context]` sub-struct (F.7.18 plan owns — schema-2).
- `[tillsyn]` top-level globals (F.7.18 plan owns — schema-3, except `spawn_temp_root` which F.7.1 adds).

---

### F.7.3 — Headless argv emission (claude adapter's BuildCommand)

**Goal:** implement `claudeAdapter.BuildCommand` per the spawn architecture memory §3 recipe; emit conditional flags via `*int`/`*float64`/`*string` priority cascade; render settings.json + agent frontmatter from F.7.2 fields.

**Builder model:** opus.

**Hard prereqs:**
- F.7.1 (bundle layout + `BundlePaths`).
- F.7.2 (tool-gating + sandbox fields on `AgentBinding`).
- F.7.17 schema-1 (per-binding `command`, `args_prefix`, `env`, `cli_kind`).
- F.7.17 adapter scaffold (`claudeAdapter` struct, `BindingResolved` type, registry).

**Files to edit/create:**
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/cli_claude/build_command.go` (new — `claudeAdapter.BuildCommand`).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/cli_claude/render_settings.go` (new — renders `settings.json` from `BindingResolved`).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/cli_claude/render_agent_md.go` (new — renders `agents/<name>.md` frontmatter + body).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/cli_claude/render_plugin_json.go` (new — minimal `.claude-plugin/plugin.json`).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/cli_claude/render_mcp_json.go` (new — `.mcp.json` self-registration entry).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/cli_claude/render_system_prompt.go` (new — assembles `system-prompt.md` from binding's `SystemPromptTemplatePath` + per-spawn dynamic context).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/cli_claude/build_command_test.go` (new).
- Test files for each render_* helper.
- DELETE: argv-emission logic in `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/spawn.go` (replaced wholesale; the file becomes a thin facade calling into the adapter).

**Packages locked:** `internal/app/dispatcher/cli_claude` (new), `internal/app/dispatcher`.

**Acceptance criteria:**
- [ ] `claudeAdapter.BuildCommand(ctx, BindingResolved, BundlePaths) (*exec.Cmd, error)` returns an `*exec.Cmd` whose `Args` are exactly the memory §3 recipe with the always-emitted baseline plus conditionally-emitted flags.
- [ ] Always-emitted baseline: `--bare`, `--plugin-dir <bundle>/plugin`, `--agent <name>`, `--system-prompt-file <bundle>/system-prompt.md`, `--settings <bundle>/plugin/settings.json`, `--setting-sources ""`, `--strict-mcp-config`, `--permission-mode acceptEdits`, `--output-format stream-json`, `--verbose`, `--no-session-persistence`, `--exclude-dynamic-system-prompt-sections`, `-p "<minimal-prompt>"`.
- [ ] Conditional flags emitted only when value resolves through the priority cascade `CLI flag > MCP arg > TUI override > template TOML default > absent`: `--max-budget-usd`, `--max-turns`, `--effort`, `--model`, `--append-system-prompt-file`, `--tools`. Implemented via `BindingResolved` carrying `*int`/`*float64`/`*string` pointers; nil pointer = flag omitted.
- [ ] `--tools` flag emitted ONLY when `BindingResolved.ToolsEngineMinimal` is non-empty; rendered as comma-joined string.
- [ ] `BuildCommand` materializes the claude-specific bundle subtree before returning the cmd:
  - `<Root>/plugin/.claude-plugin/plugin.json` with `{"name": "spawn-<id>"}` minimum.
  - `<Root>/plugin/agents/<agent-name>.md` rendered from canonical Tillsyn agent template + binding's `ToolsAllowed`/`ToolsDisallowed` frontmatter.
  - `<Root>/plugin/.mcp.json` with `{"tillsyn": {"command": "till", "args": ["serve-mcp"]}}` self-registration.
  - `<Root>/plugin/settings.json` rendered with permissions block (allow/ask/deny from `ToolsAllowed`/`ToolsDisallowed` + dev-approved grants from F.7.5's `permission_grants`) + sandbox block (filesystem + network from `Sandbox.Filesystem` + `Sandbox.Network`).
  - `<Root>/system-prompt.md` rendered from binding's `SystemPromptTemplatePath` (or canonical default) + per-spawn dynamic context (action item ID, kind, retry attempt#).
  - `<Root>/system-append.md` (only when F.7.18 context aggregator yields inline content; cross-plan dependency).
- [ ] `cmd.Dir = projectWorktree`. `cmd.Env` is the closed POSIX baseline + per-binding `env` resolved via `os.Getenv` (cross-plan dependency on F.7.17 schema-1 `env` field; F.7.17 plan owns the closed-baseline list + resolution semantics).
- [ ] Tests assert byte-for-byte argv parity for a fixed input — table-driven tests with golden argv slices.
- [ ] Tests assert each render_* helper produces JSON / TOML / Markdown matching golden fixtures.
- [ ] Settings.json deny rules MIRROR `ToolsDisallowed` AND auto-include workaround patterns: `Bash(curl *)`, `Bash(wget *)`, `Bash(http *)`, `Bash(nc *)` whenever `WebFetch` is in `ToolsDisallowed` (per memory §5 probe-grounded conclusion).

**Test scenarios (happy + edge):**
- Happy: Drop-4a default-go binding produces argv = recipe baseline + `--max-budget-usd` (binding has explicit value) + `--max-turns`.
- Edge: binding with `MaxBudgetUSD == nil` → `--max-budget-usd` flag NOT emitted.
- Edge: binding with `ToolsEngineMinimal = ["Read", "Grep", "Glob"]` → `--tools "Read,Grep,Glob"` emitted.
- Edge: binding with `ToolsDisallowed = ["WebFetch"]` → settings.json deny includes `WebFetch`, `Bash(curl *)`, `Bash(wget *)`, `Bash(http *)`, `Bash(nc *)`.
- Edge: missing system-prompt template file → `BuildCommand` returns wrapped error.
- Falsification cross-check: `assemblePrompt`'s `hylla_artifact_ref` line REMOVED (F.7.10 mitigation; verified by absence test).

**Falsification mitigations to bake in:**
- F.7.10 verification: prompt body must NOT contain `hylla_artifact_ref` substring.
- Tool-gating workaround patterns are auto-injected when WebFetch is denied (memory §5).

**Verification gates:** `mage test-pkg ./internal/app/dispatcher/cli_claude` + `mage test-pkg ./internal/app/dispatcher` + `mage ci` + per-droplet QA twins.

**Out of scope:**
- Stream-JSON parsing (F.7.4 owns).
- Permission-grants storage (F.7.5 owns).
- Codex adapter `BuildCommand` (Drop 4d).
- F.7.18 context aggregator integration (cross-plan).
- Closed-baseline env list semantics (F.7.17 plan owns).

---

### F.7.4 — Stream-JSON monitor parser (claude adapter's ParseStreamEvent + ExtractTerminalReport)

**Goal:** implement `claudeAdapter.ParseStreamEvent` + `ExtractTerminalReport` to parse `--output-format stream-json` events per memory §6 taxonomy; populate cross-CLI `StreamEvent` + `TerminalReport` types; integrate with dispatcher monitor (which stays CLI-agnostic).

**Builder model:** opus.

**Hard prereqs:**
- F.7.3 (claude adapter `BuildCommand` produces `<bundle>/stream.jsonl` via `--output-format stream-json` flag + cmd Stdout/Stderr capture).
- F.7.17 cross-CLI types (`StreamEvent`, `TerminalReport`, `ToolDenial`).
- F.7.17 adapter scaffold (`claudeAdapter` struct registered).
- Drop 4a 4a.21 process monitor (PID + exit watch) — F.7.4 LAYERS the stream parser on top.

**Files to edit/create:**
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/cli_claude/parse_stream_event.go` (new).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/cli_claude/extract_terminal_report.go` (new).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/cli_claude/parse_stream_event_test.go` (new — uses recorded fixture stream from memory §6 probe data).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/cli_claude/extract_terminal_report_test.go` (new).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/cli_claude/testdata/stream_*.jsonl` (new — 3+ fixture streams).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/monitor.go` (extend Drop 4a 4a.21 — wire stream-event consumption via `adapter.ParseStreamEvent`).

**Packages locked:** `internal/app/dispatcher/cli_claude`, `internal/app/dispatcher`.

**Acceptance criteria:**
- [ ] `claudeAdapter.ParseStreamEvent(line []byte) (StreamEvent, error)` parses each stream-json line into the cross-CLI `StreamEvent` shape. Recognized event types per memory §6: `system/init`, `assistant`, `user`, `result`. Unknown event types preserved as `StreamEvent{Type: "unknown", Raw: line}` rather than rejected.
- [ ] `claudeAdapter.ExtractTerminalReport(StreamEvent) (TerminalReport, bool)` returns `(report, true)` when the event is the terminal `result` event; `(_, false)` otherwise. The `bool` is the "is-terminal" signal the dispatcher monitor uses to decide when to stop tailing the stream.
- [ ] `TerminalReport` populated from result event:
  - `Cost *float64` ← `total_cost_usd`.
  - `Denials []ToolDenial` ← `permission_denials[]` mapped via `{tool_name, tool_input}` (note: claude's `tool_use_id` field is dropped — not part of cross-CLI shape per planner-review §6.4).
  - `Reason string` ← `terminal_reason` (or `subtype` when `terminal_reason` absent).
  - `Errors []string` ← `errors[]` (string-coerced).
- [ ] Malformed JSON line returns wrapped error; dispatcher monitor logs + skips, does not crash.
- [ ] Truncated stream (no terminal event) — dispatcher monitor reports `terminal_reason = "stream_unavailable"`, `Cost = nil`, `Denials = nil`. Falsification mitigation #6.
- [ ] Empty `<bundle>/stream.jsonl` (claude crashed pre-init) — same handling: `stream_unavailable` terminal reason synthesized by monitor.
- [ ] Dispatcher monitor in `monitor.go` stays CLI-agnostic — it consumes `StreamEvent` from `adapter.ParseStreamEvent`, does NOT branch on `cli_kind`. Adapter selection happens via `adapter := adapterRegistry.Get(cliKind)`.
- [ ] Action item `metadata.actual_cost_usd` written from `TerminalReport.Cost` (when non-nil) on terminal-state transition.

**Test scenarios (happy + edge):**
- Happy: fixture `stream_simple.jsonl` (1 system/init + 1 assistant + 1 result) → 3 events parsed; terminal extracts `Cost = 0.006039`, no denials.
- Happy: fixture `stream_with_denial.jsonl` (system/init + assistant + user/tool_result is_error=true + assistant + result with permission_denials) → 5 events; terminal extracts `Denials = [{tool_name: "Bash", tool_input: {...}}]`.
- Edge: malformed JSON line in middle → that line returns error; subsequent lines parse normally.
- Edge: no terminal event (truncated) → monitor synthesizes `stream_unavailable`.
- Edge: `total_cost_usd` field missing from result event → `Cost = nil` (not zero — distinction matters for cross-CLI report semantics where some CLIs don't emit cost).
- Falsification cross-check: unknown event type doesn't crash parser (forward-compat for future claude versions).

**Falsification mitigations to bake in:**
- Attack #6: missing/empty/truncated stream file → structured `stream_unavailable` terminal reason.
- Forward-compat: unknown event types preserved, not rejected.

**Verification gates:** `mage test-pkg ./internal/app/dispatcher/cli_claude` + `mage ci` + per-droplet QA twins.

**Out of scope:**
- Permission-grants storage (F.7.5 owns).
- Real-time mid-stream denial detection (explicit non-goal per SKETCH §F.7.5; Drop 4c+ optional).
- Codex adapter parser (Drop 4d).
- Stream events for non-claude CLIs (F.7.17 / Drop 4d).

---

### F.7.5 — Permission-denial → TUI handshake + SQLite permission_grants table

**Goal:** on terminal `result` event with `permission_denials[]`, post Tillsyn attention-item to dev's TUI for approve/deny; persist approve-always grants to new SQLite `permission_grants` table; next spawn of same kind reads grants and injects into per-spawn `settings.json`.

**Builder model:** opus.

**Hard prereqs:**
- F.7.4 (`TerminalReport.Denials` populated).
- F.7.17 cross-CLI types (`ToolDenial`).
- F.7.17 cli_kind column on `permission_grants` (per planner-review §6.4 retro-edit). **Cross-plan dependency: the F.7.17 plan owns the cli_kind retro-edit; F.7.5 here defines the rest of the table schema.**
- Drop 4a Tillsyn TUI attention-item plumbing.

**Files to edit/create:**
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/adapters/storage/sqlite/permission_grants.go` (new — table DDL, insert, list, query).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/adapters/storage/sqlite/permission_grants_test.go` (new).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/permission_handshake.go` (new — terminal-event handler that posts attention-item per denial).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/permission_handshake_test.go` (new).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/cli_claude/render_settings.go` (extend F.7.3's render — read grants for `(project_id, kind)` and merge into permissions.allow patterns).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/adapters/storage/sqlite/init.go` or equivalent init path — add `permission_grants` table-creation DDL to fresh-DB init (NO migration script per pre-MVP rule).

**Packages locked:** `internal/adapters/storage/sqlite`, `internal/app/dispatcher`, `internal/app/dispatcher/cli_claude`.

**Acceptance criteria:**
- [ ] `permission_grants` SQLite table schema:
  - `id TEXT PRIMARY KEY`
  - `project_id TEXT NOT NULL`
  - `kind TEXT NOT NULL` (action-item kind, closed-enum at app layer)
  - `cli_kind TEXT NOT NULL` (cross-plan with F.7.17; default `claude`)
  - `rule TEXT NOT NULL` (claude-pattern-syntax string, e.g. `Bash(curl https://api.example.com)`)
  - `granted_by TEXT NOT NULL` (principal ID; `STEWARD` / dev TUI principal)
  - `granted_at TEXT NOT NULL` (RFC3339)
  - `INDEX (project_id, kind, cli_kind)` for the per-spawn lookup hot path.
- [ ] `PermissionGrantsRepo` Go interface in `internal/app/dispatcher/permission_handshake.go`: `Insert(ctx, grant) error`, `ListByKind(ctx, projectID, kind, cliKind) ([]Grant, error)`. Adapter implementation in `internal/adapters/storage/sqlite/`.
- [ ] On terminal-event handler (F.7.4 dispatch path), iterate `TerminalReport.Denials`:
  - For each `ToolDenial`, post a Tillsyn attention-item with structured payload `{tool_name, tool_input, kind, cli_kind, action_item_id}` and three options: "Allow once" / "Allow always" / "Deny".
  - "Allow once" → no DB write; current spawn already failed; dev re-dispatches manually.
  - "Allow always" → DB row inserted: `(project_id, kind, cli_kind, rule, granted_by=dev_principal, granted_at=now)`.
  - "Deny" → action item moves to `failed` with `metadata.failure_reason = "permission_denied"`.
- [ ] Next spawn of the same `(project_id, kind, cli_kind)` reads grants via `ListByKind` and renders allow patterns into `settings.json` per F.7.3.
- [ ] `rule` column stores the claude-syntax pattern verbatim — no cross-CLI translation in Drop 4c (codex grant translation is Drop 4d concern; the cli_kind discriminator prevents cross-CLI grant misuse).
- [ ] Tests cover: insert grant; list grants by kind; verify grants mid-stream of next spawn's settings.json render; verify cli_kind discriminator (claude grant doesn't surface in codex spawn lookup).

**Test scenarios (happy + edge):**
- Happy: spawn denies `Bash(curl https://x.com)` → terminal `Denials = [...]` → attention-item posted → dev approves "always" → DB row inserted → next spawn of same kind has `permissions.allow` includes `Bash(curl https://x.com)`.
- Edge: empty `Denials` → no attention-items posted; no DB writes.
- Edge: same `(project_id, kind, cli_kind, rule)` granted twice → second insert is a no-op (idempotent — UNIQUE constraint in DDL OR upsert).
- Falsification cross-check: cross-CLI isolation — claude grant for kind=build does NOT appear in codex spawn's render (codex spawn filters by `cli_kind = "codex"`).

**Falsification mitigations to bake in:**
- cli_kind discriminator prevents cross-CLI grant misuse (planner-review §6.4 retro-edit).
- Allow once vs allow always distinction prevents accidental persistent over-grants.

**Verification gates:** `mage test-pkg ./internal/adapters/storage/sqlite` + `mage test-pkg ./internal/app/dispatcher` + `mage ci` + per-droplet QA twins.

**Out of scope:**
- Real-time mid-stream handshake (explicit non-goal).
- TUI rendering of attention-item (TUI Drop 4.5 owns presentation).
- Cross-CLI rule translation (Drop 4d).
- Per-grant TTL (out of scope for Drop 4c; persistent grants are forever until dev revokes).

---

### F.7.6 — Required system-plugin pre-flight check

**Goal:** at `till bootstrap` AND per-dispatch, shell out to `claude plugin list --json`, parse installed-plugin set, fail hard if any project TOML `tillsyn.requires_plugins = [...]` entry is missing with clear "Run: claude plugin install <name>" instruction.

**Builder model:** opus.

**Hard prereqs:**
- F.7.17 schema-1 `cli_kind` field on `AgentBinding` (so the pre-flight check can pick the right CLI's plugin-list parser).
- F.7.17 `CLIAdapter` may expose a `PluginListCheck` method; if not, claudeAdapter implements its own plugin-list shell-out. **Cross-plan: confirm in F.7.17 plan whether plugin-list is part of the adapter interface or claude-specific.**

**Files to edit/create:**
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/preflight/plugin_check.go` (new).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/preflight/plugin_check_test.go` (new).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/templates/schema.go` (extend F.7.18-Schema-3-introduced `Tillsyn` struct with `RequiresPlugins []string` field, TOML tag `requires_plugins`).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/cmd/till/bootstrap.go` (new or extend — invoke plugin-check at bootstrap).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/spawn.go` (or wherever pre-dispatch runs — invoke plugin-check before spawn).

**Packages locked:** `internal/app/preflight`, `internal/templates`, `cmd/till`, `internal/app/dispatcher`.

**Acceptance criteria:**
- [ ] `Tillsyn.RequiresPlugins []string` field on the `Tillsyn` schema struct (cross-plan: F.7.18 Schema-3 droplet introduces the struct; this droplet adds the field).
- [ ] `CheckRequiredPlugins(ctx, requires []string) error`:
  - Shells out to `claude plugin list --json` via `os/exec`.
  - Parses stdout as JSON (per memory §1 Path B).
  - For each `requires` entry, checks the parsed installed-plugin set.
  - Returns wrapped `ErrMissingPlugin` with explicit message: `tillsyn: missing required plugin <name>; run: claude plugin install <name>`.
- [ ] `claude plugin list --json` shell-out timeout: 5s. Failure (claude binary not installed, command unavailable) returns wrapped error pointing dev at install instructions for claude itself.
- [ ] `till bootstrap` invokes `CheckRequiredPlugins` once on each `tillsyn bootstrap`. Pre-dispatch hook also invokes it (Attack #7 mitigation: catches plugin uninstall between bootstrap and dispatch).
- [ ] Pre-dispatch invocation is fast: `claude plugin list --json` typically <50ms; plus per-call result is NOT cached across dispatches (per-dispatch confirmation is the point).
- [ ] Tests use a fake exec that returns canned JSON output — covers happy path (all plugins installed), missing-one-plugin, missing-claude-binary, malformed JSON.

**Test scenarios (happy + edge):**
- Happy: `requires_plugins = ["context7@claude-plugins-official"]`, fake exec returns JSON with `context7` installed → no error.
- Edge: `requires_plugins = ["context7@claude-plugins-official"]`, fake exec returns JSON without `context7` → `ErrMissingPlugin` wrapped with install instruction.
- Edge: `requires_plugins = []` (empty) → no error, no exec call.
- Edge: claude binary missing → wrapped error pointing at claude install URL.
- Edge: malformed JSON output → wrapped parsing error.
- Falsification cross-check (Attack #7): plugin uninstalled between bootstrap and dispatch → next dispatch fails fast with the same `ErrMissingPlugin`.

**Falsification mitigations to bake in:**
- Attack #7: pre-flight runs both at bootstrap AND per-dispatch (cheap recheck).

**Verification gates:** `mage test-pkg ./internal/app/preflight` + `mage ci` + per-droplet QA twins.

**Out of scope:**
- Auto-install missing plugins (explicitly out — dev runs `claude plugin install` manually).
- Codex plugin-list semantics (Drop 4d).
- Plugin version constraint validation (out of scope; just name-membership check).

---

### F.7.7 — Auto-add `.tillsyn/spawns/` to `.gitignore` when `spawn_temp_root = "project"`

**Goal:** on first project-mode dispatch, append `.tillsyn/spawns/` line to project worktree's `.gitignore` (idempotent — check existing lines first); skip when `spawn_temp_root = "os_tmp"`; emit one-time TUI notice on first append.

**Builder model:** opus.

**Hard prereqs:**
- F.7.1 (`spawn_temp_root` knob landed on `Tillsyn` struct).

**Files to edit/create:**
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/gitignore_helper.go` (new).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/gitignore_helper_test.go` (new).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/spawn.go` or `spawn_bundle.go` — invoke helper when `tempRoot == "project"` AND only on the first dispatch per (project, session).

**Packages locked:** `internal/app/dispatcher`.

**Acceptance criteria:**
- [ ] `EnsureSpawnsGitignored(projectWorktree string) (added bool, err error)`:
  - Reads `<worktree>/.gitignore` if exists.
  - Checks line-by-line for exact match `.tillsyn/spawns/` (with or without trailing slash variants `.tillsyn/spawns`, `/.tillsyn/spawns`, etc. — checks 4 forms).
  - If absent, appends `.tillsyn/spawns/\n` to file. Returns `added = true`.
  - If present, no-op. Returns `added = false`.
  - If `.gitignore` doesn't exist, creates it with single line. Returns `added = true`.
- [ ] Helper invoked from spawn pipeline ONLY when `tempRoot == "project"`. Skipped entirely when `tempRoot == "os_tmp"`.
- [ ] Helper invoked at MOST once per (project, session) — uses an in-memory dispatcher-scoped flag to skip subsequent calls. Falsification mitigation #1.
- [ ] First successful append emits structured log: `gitignore: added .tillsyn/spawns/ to <path>`. Future TUI integration optional.
- [ ] Idempotent: running twice in same session is no-op on second call.
- [ ] Tests: empty gitignore, gitignore with entry, gitignore without entry, missing gitignore file, malformed gitignore (unreadable bytes — should fail gracefully, not corrupt).

**Test scenarios (happy + edge):**
- Happy: empty `.gitignore` → after call, file contains `.tillsyn/spawns/\n`. `added = true`.
- Happy: `.gitignore` with `.tillsyn/spawns/` already → `added = false`.
- Edge: `.gitignore` with `/.tillsyn/spawns` (leading slash variant) → `added = false` (matched).
- Edge: missing `.gitignore` → file created with single line.
- Edge: spawn_temp_root = "os_tmp" → helper not called at all.
- Falsification cross-check (Attack #1): helper called twice in same session → second call is no-op (per-session flag).

**Falsification mitigations to bake in:**
- Attack #1: silent mutation guarded by (a) project-mode-only invocation, (b) idempotency check, (c) per-session flag.

**Verification gates:** `mage test-pkg ./internal/app/dispatcher` + `mage ci` + per-droplet QA twins.

**Out of scope:**
- Removing the line on `spawn_temp_root` switch (one-way write; dev cleans up manually).
- Other gitignore entries (only `.tillsyn/spawns/`).

---

### F.7.8 — Crash-recovery / orphan scan on Tillsyn startup

**Goal:** on Tillsyn startup, enumerate every `in_progress` action item; read `<bundle>/manifest.json` → `claude_pid`; check PID liveness via `os.FindProcess` + signal 0 + cmdline match; live → leave; dead → move to `failed` with `metadata.failure_reason = "dispatcher_restart_orphan"` + cleanup bundle.

**Builder model:** opus.

**Hard prereqs:**
- F.7.1 (`Manifest` struct, `ReadManifest`, `RemoveBundle`).
- F.7.17 schema-1 `cli_kind` on manifest (per planner-review §6.4 retro-edit).
- Drop 4a action-item state machine (`in_progress` → `failed` transition).

**Files to edit/create:**
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/orphan_scan.go` (new).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/orphan_scan_test.go` (new).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/pid_liveness.go` (new — POSIX-only PID check helper).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/cmd/till/bootstrap.go` or daemon entry — invoke orphan scan after auth bootstrap.

**Packages locked:** `internal/app/dispatcher`, `cmd/till`.

**Acceptance criteria:**
- [ ] `OrphanScan(ctx, repo ActionItemRepo, adapterRegistry CLIAdapterRegistry) error`:
  - Lists every action item with state `in_progress`.
  - For each, reads `metadata.spawn_bundle_path` and `ReadManifest`.
  - If manifest's `claude_pid == 0` (spawn started but cmd.Start hadn't returned — race window from F.7.1 Attack #5) → leave alone, log a warning.
  - If `claude_pid > 0`, route to `adapterRegistry.Get(manifest.CLIKind).IsPIDAlive(pid)` — claude adapter's check uses `os.FindProcess(pid)` + `process.Signal(syscall.Signal(0))` + cmdline match against `claude` binary path. Codex adapter's check is similar (Drop 4d).
  - Live → leave. Bundle stays. Tillsyn re-monitors via SQLite state changes.
  - Dead → move action item to `failed`; set `metadata.failure_reason = "dispatcher_restart_orphan"`; `RemoveBundle(bundle)`; emit attention-item to dev.
- [ ] PID liveness check is POSIX-only (cross-plan: F.7.17 plan declares POSIX-only scope per QA-R2 A2.c). Windows path returns wrapped error; Tillsyn doesn't run on Windows in Drop 4c.
- [ ] Tests use a fake `pidIsAlive` to deterministically exercise live + dead branches; uses a fake `ActionItemRepo` for state transitions.
- [ ] Bundle cleanup is idempotent (F.7.1 acceptance).

**Test scenarios (happy + edge):**
- Happy: 3 in_progress action items; 2 PIDs alive, 1 dead → 2 left alone, 1 moved to `failed` + bundle removed + attention-item posted.
- Edge: action item in_progress with `metadata.spawn_bundle_path` empty → log + leave alone (don't crash).
- Edge: bundle path exists but manifest.json missing → log + leave alone.
- Edge: manifest's `claude_pid == 0` → log warning + leave alone (Attack #5 race window).
- Edge: cmdline mismatch (PID alive but reused by unrelated process) → treated as dead.
- Edge: action item already in `failed` (race with another orphan-scan caller) → skip.
- Falsification cross-check (Attack #5): freshly-spawned claude with cmd.Start succeeded but manifest update lagged → manifest pid==0; orphan scan leaves alone.

**Falsification mitigations to bake in:**
- Attack #5: PID-zero handling.
- cmdline mismatch detection prevents PID-reuse false-positives.

**Verification gates:** `mage test-pkg ./internal/app/dispatcher` + `mage ci` + per-droplet QA twins.

**Out of scope:**
- Auto-redispatch on orphan (dev decides; Drop 5 dogfood may motivate auto-redispatch refinement).
- Codex PID liveness (Drop 4d).
- Windows PID semantics (out of scope; POSIX-only Drop 4c).

---

### F.7.9 — Action-item metadata fields (`spawn_bundle_path`, `spawn_history[]`, `actual_cost_usd`)

**Goal:** add three metadata fields to `domain.ActionItem` (or its metadata blob): `spawn_bundle_path` (current bundle), `spawn_history[]` (audit-only append-only history), `actual_cost_usd` (current spawn cost). Document `spawn_history[]` doc-comment requirement per planner-review P-§5.b — audit-only role + link to F.7.18 round-history-deferred decision.

**Builder model:** opus.

**Hard prereqs:** none — independent domain extension; can land in parallel with F.7.10.

**Files to edit/create:**
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/domain/action_item.go` (extend metadata fields).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/domain/action_item_test.go` (extend).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/adapters/storage/sqlite/action_items.go` (extend persistence path — JSON-encoded metadata blob OR new columns; pick lower-friction option).

**Packages locked:** `internal/domain`, `internal/adapters/storage/sqlite`.

**Acceptance criteria:**
- [ ] `ActionItem` metadata gains:
  - `spawn_bundle_path string` — absolute path to current spawn's bundle (set on dispatch; cleared on terminal-state). Drives F.7.8 orphan scan + bundle cleanup.
  - `spawn_history []SpawnHistoryEntry` — append-only; each entry: `{spawn_id, bundle_path, started_at, terminated_at, outcome, total_cost_usd}`.
  - `actual_cost_usd *float64` — current/most-recent spawn's terminal cost; populated by F.7.4 monitor on terminal event.
- [ ] **Doc-comment on `spawn_history` MUST cite (a) audit-only role — "this field is for ledger/dashboard rendering, NOT for re-prompting fix-builders"; (b) link to F.7.18 round-history-deferred decision: "if a use case for raw stream-json round-history surfaces post-Drop-5, add `prior_round_*` rules per F.7.18 commentary, NOT raw spawn_history reads"**. Falsification mitigation P-§5.b.
- [ ] `AppendSpawnHistory(item, entry)` helper appends in-memory; persists via existing `UpdateActionItem` path. Atomicity inherited from action-item-scoped lock (Drop 4a).
- [ ] SpawnHistoryEntry has explicit JSON tags so it persists cleanly through the metadata blob.
- [ ] Round-trip test: `actionItem` with populated fields persists to SQLite, reads back, deep-equal.
- [ ] Backward-compat: action items with empty/nil values for the three new fields work — Drop 4a-era items don't break.

**Test scenarios (happy + edge):**
- Happy: dispatch action item → `spawn_bundle_path` populated → terminal event → history entry appended → `actual_cost_usd` set → next dispatch (retry) → history grows to 2 entries; `spawn_bundle_path` updates to new bundle.
- Edge: terminal event with `Cost = nil` → `actual_cost_usd = nil`, history entry `total_cost_usd = nil`.
- Edge: action item never dispatched → all three fields empty/nil.
- Falsification cross-check (Attack #8): two concurrent terminal events on same action item (shouldn't happen but fuzz the path) → both appends serialized via lock; history has 2 entries.

**Falsification mitigations to bake in:**
- Attack #8: history-append serialized via action-item-scoped lock.
- P-§5.b: doc-comment carries audit-only role + F.7.18 round-history-deferred pointer.

**Verification gates:** `mage test-pkg ./internal/domain` + `mage test-pkg ./internal/adapters/storage/sqlite` + `mage ci` + per-droplet QA twins.

**Out of scope:**
- Reading `spawn_history` as round-history input for fix-builders (deferred per F.7.18).
- Aggregating history into ledger MD (post-dogfood; pre-MVP rule prohibits closeout MD rollups).
- TUI rendering of history (Drop 4.5).

---

### F.7.10 — Drop hardcoded `hylla_artifact_ref` from spawn.go's prompt body

**Goal:** remove the hardcoded `hylla_artifact_ref:` line from `assemblePrompt` in `internal/app/dispatcher/spawn.go`. Hylla is dev-local, NOT part of Tillsyn's shipped cascade.

**Builder model:** opus.

**Hard prereqs:** none — fully independent. Lands first/parallel with F.7.9.

**Files to edit/create:**
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/spawn.go` (delete the `hylla_artifact_ref:` lines from `assemblePrompt`).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/spawn_test.go` (extend — assert the substring `hylla_artifact_ref` is NOT in the rendered prompt).

**Packages locked:** `internal/app/dispatcher`.

**Acceptance criteria:**
- [ ] `assemblePrompt` no longer writes `hylla_artifact_ref:` and the corresponding `project.HyllaArtifactRef` line. The lines (currently spawn.go:211–213) are removed.
- [ ] `domain.Project.HyllaArtifactRef` field PRESERVED — F.7.10 only removes the prompt-body line. Adopters who opt into Hylla MCP via their own template can still surface this through their own system-prompt template (F.7.2's `system_prompt_template_path`).
- [ ] `metadata.hylla_artifact_ref` on the project blob PRESERVED.
- [ ] Test asserts the rendered prompt does NOT contain the substring `hylla_artifact_ref`.
- [ ] Test asserts `domain.Project.HyllaArtifactRef` field still exists + still persists round-trip.

**Test scenarios (happy + edge):**
- Happy: project with `HyllaArtifactRef = "github.com/x/y@main"` → rendered prompt does NOT contain `hylla_artifact_ref` substring.
- Edge: project with empty `HyllaArtifactRef` → still no substring (unchanged).
- Falsification cross-check (Attack #2): `domain.Project.HyllaArtifactRef` field is NOT removed — adopters still have access for their own templates.

**Falsification mitigations to bake in:**
- Attack #2: scope-creep guard — only the prompt-body line is removed; data field + storage column untouched.

**Verification gates:** `mage test-pkg ./internal/app/dispatcher` + `mage ci` + per-droplet QA twins.

**Out of scope:**
- Removing `domain.Project.HyllaArtifactRef`.
- Removing project metadata `hylla_artifact_ref`.
- Removing Hylla awareness from CLAUDE.md / docs (out — those are project-level dev tooling, not shipped Tillsyn binary surface).

---

### F.7.11 — Tillsyn architecture documentation

**Goal:** write Tillsyn architecture docs in `docs/` referencing `~/.claude/projects/-Users-evanschultz-Documents-Code-hylla-tillsyn/memory/project_drop_4c_spawn_architecture.md` as canonical source. Cover: two plugin paths, bundle layout, event taxonomy, settings authority, sandbox semantics, crash recovery, explicit non-goals.

**Builder model:** opus.

**Hard prereqs:** F.7.1–F.7.10 stabilized (so docs reflect landed reality, not stub state). Docs can stage incrementally — first draft after F.7.5; final after F.7.16.

**Files to edit/create:**
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/docs/spawn-pipeline-architecture.md` (new — high-level cascade dispatch flow).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/docs/cli-adapter-extensibility.md` (new — companion to F.7.17, explains seam + roadmap).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/docs/permission-handshake.md` (new — F.7.5 dev-facing flow).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/docs/sandbox-semantics.md` (new — Layer A/B/C strategy + non-adversarial scope).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/CLAUDE.md` (extend with cross-references to the above; do not duplicate content).

**Packages locked:** none (docs only); locks `docs/` directory paths.

**Acceptance criteria:**
- [ ] Each doc is 200–500 lines, written for Tillsyn adopters and contributors. Source-of-truth pointer to the spawn architecture memory at the top of each file.
- [ ] `spawn-pipeline-architecture.md` covers: cascade dispatch flow, per-spawn bundle materialization, two plugin paths (Path A `--plugin-dir` + Path B system-installed), event taxonomy summary (with link to memory §6 for verbatim probe data), settings authority, crash recovery model.
- [ ] `cli-adapter-extensibility.md` covers: `CLIAdapter` interface contract, three CLI-shape invariants (planner-review P3), multi-CLI roadmap (hard-cut interface rewrite path per QA-R2 A4.a), per-binding `command`/`args_prefix`/`env` schema (cross-plan reference to F.7.17).
- [ ] `permission-handshake.md` covers: terminal-event handshake flow, attention-item shape, `permission_grants` SQLite schema, dev approve/deny semantics, cross-CLI grant isolation via cli_kind discriminator.
- [ ] `sandbox-semantics.md` covers: three layers, two-layer strategy, sandbox.filesystem cap-dropping for Bash, cooperative deny rules for Read/Edit/Write, explicit non-goal: adversarial OS sandbox not in scope.
- [ ] CLAUDE.md gains a `## Architecture Docs` section with one-liners linking to each new doc.
- [ ] No closeout MD rollups produced (pre-MVP rule). The architecture docs are dev-facing reference, not drop ledger.
- [ ] Spelling + link-validity check via `mage check` (or equivalent) — broken internal links fail.

**Test scenarios (happy + edge):**
- N/A (docs only; verified via `mage check` markdown lint + manual review by plan-QA twins).

**Falsification mitigations to bake in:**
- Source-of-truth pointer at top of each file prevents doc drift across compactions.
- Cross-references between docs prevent duplication.

**Verification gates:** `mage check` (markdown lint) + per-droplet QA twins (proof + falsification, parallel).

**Out of scope:**
- Per-droplet design docs (each droplet's PR description carries the design rationale).
- Operator runbook (Drop 5 dogfood may motivate one).
- Marketplace template authoring guide (Theme F.4 territory).

---

### F.7.12 — Commit-agent (haiku) integration via the new spawn pipeline

**Goal:** dispatch `claude --agent commit-message-agent` through F.7.1's per-spawn temp-bundle materialization (NOT the legacy 4a.19 stub path). Reads `git diff <action_item.start_commit>..<action_item.end_commit>`. Returns single-line conventional commit message. Tool gating: Read + Bash for git diff inspection only.

**Builder model:** opus.

**Hard prereqs:**
- F.7.1, F.7.2, F.7.3, F.7.4 (spawn pipeline operational).
- F.7.17 schema-1 (`commit-message-agent` binding has `cli_kind = "claude"`).
- Drop 4a Wave 1 first-class fields: `action_item.start_commit`, `action_item.end_commit`.

**Files to edit/create:**
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/commit_agent.go` (new).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/commit_agent_test.go` (new).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/templates/builtin/default.toml` (extend — add `[agent_bindings.commit]` binding).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/.claude/agents/commit-message-agent.md` (new — agent prompt template; haiku-tier; tool-gated).

**Packages locked:** `internal/app/dispatcher`, `internal/templates`.

**Acceptance criteria:**
- [ ] `RunCommitAgent(ctx, item ActionItem, project Project) (commitMsg string, err error)`:
  - Resolves the `commit` agent binding from project's baked catalog.
  - Computes git diff between `item.StartCommit..item.EndCommit`. If `StartCommit == ""` → diff against `HEAD~1..HEAD` falls back; if `EndCommit == ""` → diff against `HEAD`. Falsification mitigation #4.
  - Materializes per-spawn bundle via F.7.1.
  - Builds claude cmd via F.7.3.
  - Runs cmd, captures terminal event via F.7.4.
  - Extracts the agent's final assistant text → trims to first non-empty line → single-line conventional commit message.
  - Validates: line ≤72 chars, conventional-commit format `<type>(<scope>): <subject>` per project CLAUDE.md "Single-Line Commits" rule.
  - On validation failure, returns wrapped error; commit gate (F.7.13) handles routing.
- [ ] Default-template `[agent_bindings.commit]` binding:
  - `agent_name = "commit-message-agent"`.
  - `model = "haiku"`.
  - `tools_allowed = ["Read", "Bash(git diff*)", "Bash(git log*)", "Bash(git status*)"]`.
  - `tools_disallowed = ["WebFetch", "Edit", "Write", "Bash(git commit*)", "Bash(git push*)"]` (commit-agent advises ONLY; F.7.13 gate executes the actual commit).
  - `max_turns = 3`, `max_budget_usd = 0.05` (haiku is cheap; budget cap protects against runaway).
  - `cli_kind = "claude"` (cross-plan F.7.17 default).
- [ ] Empty diff → commit-agent returns empty string; gate (F.7.13) treats as "no changes to commit" no-op.
- [ ] Tests use a fake spawn pipeline + canned terminal events to exercise: happy path, validation failure, empty diff, malformed agent output.

**Test scenarios (happy + edge):**
- Happy: `start_commit = "abc123"`, `end_commit = "def456"`, diff has 5 file changes → agent returns `feat(dispatcher): add per-spawn bundle lifecycle` (≤72 chars, valid format).
- Edge: empty `start_commit` → falls back to `HEAD~1..HEAD` diff.
- Edge: empty `end_commit` → diff against `HEAD`.
- Edge: agent returns multi-line text → first non-empty line taken; line >72 chars → validation failure.
- Edge: agent returns non-conventional-format → validation failure.
- Falsification cross-check (Attack #4): empty start_commit fallback works on first-build-on-fresh-worktree.

**Falsification mitigations to bake in:**
- Attack #4: empty start_commit fallback to `HEAD~1..HEAD` (or `HEAD` if no parent).
- Tool-gating: commit-agent CANNOT run `git commit` itself — the gate (F.7.13) executes the commit.

**Verification gates:** `mage test-pkg ./internal/app/dispatcher` + `mage ci` + per-droplet QA twins.

**Out of scope:**
- The `git commit` invocation itself (F.7.13).
- Push (F.7.14).
- Commit-message validation rules beyond the project CLAUDE.md single-line rule.
- Codex commit-agent (Drop 4d may add a codex-flavored binding).

---

### F.7.13 — `commit` gate implementation

**Goal:** implement `commit` gate. On post-build pipeline, when `dispatcher_commit_enabled` is true: invoke F.7.12 commit-agent → run `git add <action_item.paths>` (path-scoped) → run `git commit -m "<haiku-output>"` → populate `action_item.end_commit = git rev-parse HEAD`.

**Builder model:** opus.

**Hard prereqs:**
- F.7.12 (commit-agent operational).
- F.7.15 (project metadata toggle `dispatcher_commit_enabled`).
- Drop 4b gate runner (`internal/app/dispatcher/gate_*.go` pattern).

**Files to edit/create:**
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/gate_commit.go` (new — implements `Gate` interface).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/gate_commit_test.go` (new).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/templates/schema.go` (extend GateKind constant set with `GateKindCommit GateKind = "commit"` and update `validGateKinds`).

**Packages locked:** `internal/app/dispatcher`, `internal/templates`.

**Acceptance criteria:**
- [ ] `GateKindCommit GateKind = "commit"` constant added to schema.go's closed-enum set; `validGateKinds` updated; `IsValidGateKind` accepts the value.
- [ ] `commitGate` struct implements the Drop 4b Gate interface (`Run(ctx, item, project) (GateResult, error)`).
- [ ] Gate body when `project.Metadata.DispatcherCommitEnabled` is nil OR false → returns `GateResult{Success: true, Skipped: true, Reason: "dispatcher_commit_enabled toggle off"}`. Falsification mitigation #10.
- [ ] Gate body when toggle true:
  - Acquires worktree-wide commit lock (Drop 4a Wave 2 lock manager — coordinate with sibling builds touching the same worktree). Falsification mitigation #3.
  - Calls F.7.12 `RunCommitAgent`. On failure → gate fails with structured reason.
  - If commit-message empty (no diff) → returns `GateResult{Success: true, Skipped: true, Reason: "no changes to commit"}`.
  - Runs `git add <path>` for each path in `item.Paths` (path-scoped, NEVER `git add -A`).
  - Runs `git commit -m "<haiku-output>"`. On failure → gate fails; existing `git add` staged changes remain (dev cleans up via TUI).
  - Reads `git rev-parse HEAD` → updates `item.EndCommit` via repo persistence.
  - Returns `GateResult{Success: true, Reason: "<commit-msg>"}`.
- [ ] Tests use a fake git executor + fake commit-agent to exercise: toggle off, toggle on + happy, toggle on + empty diff, toggle on + commit-agent failure, toggle on + git add failure, toggle on + git commit failure.

**Test scenarios (happy + edge):**
- Happy: toggle on, paths=`["a.go","b.go"]`, commit-agent returns valid message → `git add a.go b.go && git commit -m "..."` runs; `EndCommit` populated.
- Edge: toggle off → gate returns success+skipped immediately; no git commands executed.
- Edge: toggle on but empty diff → success+skipped; no commit created.
- Edge: paths empty → gate fails with `paths empty; cannot commit` reason (defensive — path-scoped requires non-empty paths).
- Falsification cross-check (Attack #3): two concurrent commit gates on same worktree → second blocks on commit lock until first releases.
- Falsification cross-check (Attack #10): fresh project with `dispatcher_commit_enabled = nil` → gate skipped; no surprise commits.

**Falsification mitigations to bake in:**
- Attack #3: worktree-wide commit lock prevents concurrent commit races.
- Attack #10: toggle default-off prevents surprise commits on fresh projects.
- Path-scoped `git add` prevents accidental staging of unrelated changes.

**Verification gates:** `mage test-pkg ./internal/app/dispatcher` + `mage test-pkg ./internal/templates` + `mage ci` + per-droplet QA twins.

**Out of scope:**
- Push (F.7.14).
- Auto-rollback of failed commit (out — dev decides via TUI).
- Multi-commit batching (out — one commit per gate run).

---

### F.7.14 — `push` gate implementation

**Goal:** implement `push` gate. On post-commit pipeline, when `dispatcher_push_enabled` true: run `git push origin <branch>`. On failure: action item moves to `failed` with `metadata.BlockedReason = "git push: <error>"`. No auto-rollback of local commit.

**Builder model:** opus.

**Hard prereqs:**
- F.7.13 (commit gate created the local commit).
- F.7.15 (project metadata toggle `dispatcher_push_enabled`).

**Files to edit/create:**
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/gate_push.go` (new).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/gate_push_test.go` (new).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/templates/schema.go` (extend GateKind closed-enum set with `GateKindPush GateKind = "push"`; update `validGateKinds`).

**Packages locked:** `internal/app/dispatcher`, `internal/templates`.

**Acceptance criteria:**
- [ ] `GateKindPush GateKind = "push"` added to schema closed-enum set; `validGateKinds` updated.
- [ ] `pushGate` struct implements Gate interface.
- [ ] Gate body when `project.Metadata.DispatcherPushEnabled` is nil OR false → returns `GateResult{Success: true, Skipped: true, Reason: "dispatcher_push_enabled toggle off"}`.
- [ ] Gate body when toggle true:
  - Reads current branch via `git symbolic-ref --short HEAD`.
  - Runs `git push origin <branch>`. Captures stdout + stderr.
  - On success → `GateResult{Success: true, Reason: "pushed to origin/<branch>"}`.
  - On failure → `GateResult{Success: false, Reason: "git push: <stderr>"}`. Action item moves to `failed` with `metadata.BlockedReason` populated. NO auto-rollback of local commit (dev decides via TUI attention-item).
- [ ] Push timeout: 60s (network operations need real-world breathing room).
- [ ] Tests use a fake git executor for: toggle off, toggle on + happy, toggle on + push failure, toggle on + timeout.

**Test scenarios (happy + edge):**
- Happy: toggle on → `git push origin main` succeeds → success.
- Edge: toggle off → skipped.
- Edge: toggle on + auth failure → gate fails with stderr in reason; action item moved to `failed`; local commit preserved.
- Edge: toggle on + network timeout → gate fails after 60s.
- Falsification cross-check: post-failure local commit is preserved (verifiable via `git log` showing the commit still on local branch).

**Falsification mitigations to bake in:**
- No auto-rollback of local commit — dev preserves work even if push fails.
- Timeout prevents indefinite hang.

**Verification gates:** `mage test-pkg ./internal/app/dispatcher` + `mage test-pkg ./internal/templates` + `mage ci` + per-droplet QA twins.

**Out of scope:**
- `gh run watch --exit-status` (drop-end orchestrator concern, not per-build gate).
- Hylla reingest (drop-end only; `mage ci` separate pipeline).
- Force-push handling (out — never auto-force).
- Multi-remote push (out — `origin` only).

---

### F.7.15 — Project-metadata toggles `dispatcher_commit_enabled` + `dispatcher_push_enabled`

**Goal:** add two pointer-bool fields to `domain.ProjectMetadata`: `DispatcherCommitEnabled *bool` + `DispatcherPushEnabled *bool`. Nil-means-disabled per Drop 4a 4a.25 precedent. F.7.13/14 gates read these toggles.

**Builder model:** opus.

**Hard prereqs:** Drop 4a 4a.25 precedent for pointer-bool toggles.

**Files to edit/create:**
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/domain/project.go` (extend `ProjectMetadata` struct).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/domain/project_test.go` (extend).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/adapters/storage/sqlite/projects.go` (extend persistence path — JSON-encoded metadata blob OR new columns).

**Packages locked:** `internal/domain`, `internal/adapters/storage/sqlite`.

**Acceptance criteria:**
- [ ] `ProjectMetadata.DispatcherCommitEnabled *bool` field added with TOML/JSON tags `dispatcher_commit_enabled`.
- [ ] `ProjectMetadata.DispatcherPushEnabled *bool` field added with TOML/JSON tags `dispatcher_push_enabled`.
- [ ] Both default nil (means disabled). Round-trip-clean through SQLite + JSON.
- [ ] Helper accessors: `(p Project) IsDispatcherCommitEnabled() bool`, `(p Project) IsDispatcherPushEnabled() bool` — return `false` for nil OR `false`; return `true` only when pointer is non-nil AND points to `true`.
- [ ] Tests cover: nil → disabled; pointer to false → disabled; pointer to true → enabled; round-trip persistence.

**Test scenarios (happy + edge):**
- Happy: project with `DispatcherCommitEnabled = &true` → `IsDispatcherCommitEnabled() == true`.
- Edge: nil → false.
- Edge: pointer to false → false.
- Round-trip: project persists, reads back, deep-equal.

**Falsification mitigations to bake in:**
- Pointer-bool semantics inherited from Drop 4a 4a.25 — accessor handles all three states.

**Verification gates:** `mage test-pkg ./internal/domain` + `mage test-pkg ./internal/adapters/storage/sqlite` + `mage ci` + per-droplet QA twins.

**Out of scope:**
- TUI for toggling (Drop 4.5).
- CLI subcommand for toggling (out for Drop 4c; dev edits SQLite directly OR via existing `till project` MCP if it supports metadata writes).

---

### F.7.16 — Default template `[gates.build]` expansion

**Goal:** when F.7.13 + F.7.14 land, update `internal/templates/builtin/default.toml` `[gates.build]` from `["mage_ci"]` (Drop 4b state) to `["mage_ci", "commit", "push"]`. Each gate is independently toggleable via project metadata flags from F.7.15.

**Builder model:** opus.

**Hard prereqs:**
- F.7.13 (commit gate registered).
- F.7.14 (push gate registered).
- F.7.15 (toggles in place).

**Files to edit/create:**
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/templates/builtin/default.toml` (extend `[gates.build]`).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/templates/builtin/default_test.go` or equivalent (extend — assert default loads + gate runner registers commit + push gates).
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/templates/builtin/default-go.toml` (if separate from default.toml, mirror the change).

**Packages locked:** `internal/templates/builtin`.

**Acceptance criteria:**
- [ ] `[gates.build]` in `default.toml` is `gates.build = ["mage_ci", "commit", "push"]`.
- [ ] Default loads + validates clean (closed-enum gate kinds all valid post-F.7.13/14).
- [ ] Fresh project with default template + nil metadata flags → gate runner runs `mage_ci`, then `commit` (skipped per F.7.13), then `push` (skipped per F.7.14). Action item moves to `complete` after all three gates report success (skipped counts as success per Drop 4b semantics).
- [ ] Project with `dispatcher_commit_enabled = &true` + `dispatcher_push_enabled = &true` → all three gates run real (mage_ci, commit, push); commit msg authored by haiku via F.7.12; commit + push executed.
- [ ] Tests cover: default-go template loads; gate runner registration; fresh-project skip behavior; toggled-on full-pipeline behavior.

**Test scenarios (happy + edge):**
- Happy: fresh project, defaults nil → 3 gates run, 2 skipped, action item completes clean.
- Happy: project with both toggles on → 3 gates run, all real, action item completes after push.
- Edge: project with `dispatcher_commit_enabled = &true` + `dispatcher_push_enabled = nil` → mage_ci real, commit real (creates local commit), push skipped.
- Falsification cross-check (Attack #10): fresh project sees no surprise commit/push.

**Falsification mitigations to bake in:**
- Attack #10: default-off via toggles ensures fresh-project safety.

**Verification gates:** `mage test-pkg ./internal/templates/builtin` + `mage ci` + per-droplet QA twins.

**Out of scope:**
- Other gate sequences (`[gates.plan]`, `[gates.research]` — leave as-is from Drop 4b).
- Per-kind gate-rules table (out — Drop 4b's gate_rules forward-compat seam).

---

## Open Questions For Plan-QA Twins To Resolve

| ID | Question | Suggested Routing |
|---|---|---|
| Q1 | Does F.7.2 absorb sandbox struct validation, or split into separate validator droplet? | Plan-QA proof — verify F.7.2 acceptance criteria are tractable for one builder spawn (~20-30 minutes). If too large, split sandbox validation into F.7.2a + F.7.2b. |
| Q2 | Bundle cleanup timing — does it happen on commit/push gate failure (state stays in_progress until gate completes) or after terminal state? | Plan-QA falsification — attack the path: gate fails → action item moves to `failed` → bundle should be removed (forensics retained per `spawn_temp_root = "project"` mode only). Confirm Drop 4a Wave 2.7 cleanup hook covers `failed` state, not just `complete`. |
| Q3 | F.7.5 `cli_kind` column on `permission_grants` — owned by F.7.17 plan or F.7.5 here? | Plan-QA proof — cross-plan boundary check. Recommendation: F.7.5 here ships the FULL table including cli_kind column; F.7.17 plan ships the cli_kind field on `AgentBinding` (which the gate reads) + the manifest cli_kind field. Two separate places that both reference cli_kind. |
| Q4 | F.7.13 commit gate — `git add <action_item.paths>` semantics on first build. If `start_commit == ""`, do paths still drive the add scope, or is `git add -A` acceptable as a one-time bootstrap? | Plan-QA falsification — `git add -A` is explicitly forbidden in mitigation #3. Always use path-scoped add. Empty paths → gate fails defensively. |
| Q5 | F.7.16 default template — does it ship for adopters with both gates listed (skipped by default toggle) or with only mage_ci (toggle-flip adds commit/push)? | Plan-QA proof — declared in F.7.16 acceptance: gates listed, toggles default-off. This makes toggle-flip a one-line `dispatcher_commit_enabled = true` change for adopters, no template re-bake needed. |
| Q6 | F.7.6 plugin pre-flight — does pre-dispatch invocation cache `claude plugin list --json` output for the dispatcher's lifetime, or call afresh every spawn? | Plan-QA falsification — Attack #7 says always-call-fresh (uninstalled-between-bootstrap-and-dispatch case). Cost: ~50ms per spawn. Acceptable for safety. Confirm. |
| Q7 | F.7.11 docs — written by orchestrator (per `feedback_orchestrator_no_build.md` MD edits OK) or by a builder subagent? | Plan-QA proof — MD-only files are orchestrator-editable. Recommend orchestrator authors directly; F.7.11 droplet's "builder" is a no-op formality. Plan-QA confirms. |

---

## References

- `workflow/drop_4c/SKETCH.md` — F.7 + three-schema-droplet sequencing block (lines 123–243).
- `workflow/drop_4c/4c_F7_EXT_PLANNER_REVIEW.md` — P1–P13 + four cross-section retro-edits (§6.1–§6.4).
- `workflow/drop_4c/4c_F7_EXT_QA_FALSIFICATION_R2.md` — round-2 verdicts; surviving must-fixes baked into droplet acceptance criteria.
- `~/.claude/projects/-Users-evanschultz-Documents-Code-hylla-tillsyn/memory/project_drop_4c_spawn_architecture.md` — canonical spawn architecture; 13 sections of probe-grounded findings.
- `internal/app/dispatcher/spawn.go` — current 4a.19 stub (target of F.7.10 + F.7.3 wholesale rewrite).
- `internal/templates/schema.go` — current `AgentBinding` struct (extended by F.7.2 + sibling F.7.17 schema-1).
- `internal/templates/load.go` — strict-decode pattern (`DisallowUnknownFields`) referenced by F.7.2.
- Project `CLAUDE.md` — "Build Verification" §3 (`mage install` is dev-only; gates use `mage ci`); "Single-Line Commits" rule (≤72 chars conventional commit).
- Project `PLAN.md` — drop sequencing.
- `WIKI.md` — Cascade Vocabulary (kind enum + structural_type axis).

---

## REVISIONS POST-AUTHORING (2026-05-05) — supersedes affected portions above

The dev approved architectural changes after this sub-plan was authored. **Where this section conflicts with text above, this section wins.** Builders read this section first.

### REV-1 — `command` and `args_prefix` GONE from F.7.17 schema

The F.7.17 sub-plan no longer adds `Command []string` / `ArgsPrefix []string` to `AgentBinding`. F.7.17 Schema-1 ships ONLY `Env []string` + `CLIKind string`.

Concrete impact on this sub-plan:

- **F.7.3 hard prereqs**: now reference only `(env, cli_kind)`, NOT `(command, args_prefix, env, cli_kind)`.
- **F.7.3 argv emission**: adapter hardcodes its CLI binary name internally (`claude` for claude adapter). No `command` override path; no `args_prefix` prepending.
- **F.7.3 sandbox/tool-gating fields** (`tools_allowed`, `tools_disallowed`, `system_prompt_template_path`, `[agent.sandbox.*]`) remain in F.7.3 scope per original plan — these are F.7.2 schema additions, NOT F.7.17 fields.

### REV-2 — F.7.3 split into F.7.3a + F.7.3b

F.7.3 (originally ~600-800 LOC, 8+ files) splits per Falsification round-2 #11:

- **F.7.3a — claude argv emission.** Builds the headless argv slice from `BindingResolved` + bundle paths. Single principal file (`internal/app/dispatcher/cli_claude/argv.go`). Conditional flag emission via `*int` / `*float64` / `*string` resolved through priority cascade. ~200 LOC. Acceptance: byte-for-byte argv parity tests against fixtures.
- **F.7.3b — bundle render.** Writes `system-prompt.md`, `system-append.md`, `settings.json`, `agents/<name>.md`, `.claude-plugin/plugin.json`, `.mcp.json`. Six render helpers under `internal/app/dispatcher/cli_claude/render/`. ~400 LOC across 6 helpers + 6 test files. Each helper is a single-file pure function. Per Falsification R2 NIT recommendation: if F.7.3b's review surface still feels heavy at QA time, split further into F.7.3b-1 (settings + permissions) / F.7.3b-2 (system-prompt + system-append) / F.7.3b-3 (plugin + mcp). Plan-QA-twins decide at builder dispatch time.

### REV-3 — F.7.5 split into F.7.5a + F.7.5b + F.7.5c

F.7.5 (originally ~500 LOC, 5+ files) splits:

- **F.7.5a — `permission_grants` table + storage.** SQLite DDL inline in storage init path (per L22 — no migration logic; dev-fresh-DB). Storage adapter ports (`PermissionGrantsStore` interface + SQLite impl). Acceptance: explicit dev-fresh-DB callout; CRUD round-trip test.
- **F.7.5b — TUI handshake.** Parses `permission_denials[]` from terminal `result` event (via adapter's `ExtractTerminalReport`); posts attention-item to dev with `{tool_name, tool_input}` payload; dev approves/denies via TUI. Acceptance: terminal-event fixture with denials → attention-item created; dev-approval path writes grant via F.7.5a's storage.
- **F.7.5c — settings.json grant injection.** Reads stored grants per-spawn via F.7.5a's storage; renders into settings.json `permissions.allow` entries. Acceptance: spawn after grant approval includes grant in rendered settings.json. Per Falsification round-2 B5: settings.json grant injection happens at SPAWN-TIME (F.7.5c reads grants at the start of each spawn), so a grant approved during Spawn-N is available for Spawn-N+1 without explicit cross-spawn sync.

### REV-4 — F.7.1 Manifest does NOT include `CLIKind`

F.7-CORE F.7.1 ships `Manifest` struct WITHOUT a `CLIKind` field. F.7.17.6 is the sole owner of `Manifest.CLIKind`. F.7.1 acceptance criteria: list manifest fields as `{spawn_id, action_item_id, kind, claude_pid, started_at, paths}` — NO `CLIKind`.

### REV-5 — F.7.8 orphan scan blocked_by F.7.17.6

F.7-CORE F.7.8 reads `manifest.CLIKind` to route adapter liveness checks. F.7.17.6 adds the field. New explicit blocked_by edge: **F.7.8 blocked_by F.7.17.6**. F.7.8 cannot start until F.7.17.6 lands.

### REV-6 — F.7.9 metadata locked to JSON-blob (not new columns)

F.7-CORE F.7.9 ships `spawn_bundle_path string`, `spawn_history []SpawnHistoryEntry`, `actual_cost_usd float64` as JSON-encoded fields inside `domain.ActionItem.Metadata` (which is `map[string]any` today, JSON-marshalled). NO new SQLite columns. NO migration. Acceptance criteria removes "OR new columns; pick lower-friction option" — JSON blob is the locked choice.

Per Falsification R2 NIT A5: F.7.15 (project metadata toggles `dispatcher_commit_enabled` / `dispatcher_push_enabled`) extends `domain.ProjectMetadata` — NOT a JSON blob, but a struct field add per Drop 4a Wave 3 W3.2 precedent (`OrchSelfApprovalEnabled *bool` at `internal/domain/project.go:119-145`). Schema addition lives inside the storage init path with explicit dev-fresh-DB callout, same pattern as F.7.5a.

### REV-7 — `Tillsyn` struct extension policy

F.7.18.2 owns the initial `Tillsyn` top-level struct declaration with two fields: `MaxContextBundleChars int` + `MaxAggregatorDuration Duration`. F.7-CORE droplets that add fields to `Tillsyn`:

- **F.7.1** extends with `SpawnTempRoot string` (TOML `tillsyn.spawn_temp_root = "os_tmp" | "project"`).
- **F.7.6** extends with `RequiresPlugins []string` (TOML `tillsyn.requires_plugins = [...]`).

Each extending droplet's acceptance criteria says: "extends `Tillsyn` struct (initially declared in F.7.18.2)" + adds ONLY its named field + ships a unit test asserting strict-decode rejects an unknown key on the extended struct.

Sequencing: F.7.18.2 lands FIRST; F.7.1 + F.7.6 land AFTER (consume the wider struct).

### REV-8 — L4 closed env baseline expanded

F.7-CORE droplets that touch env (F.7.3a argv emission's `cmd.Env`) use the expanded baseline:

**Process basics:** `PATH, HOME, USER, LANG, LC_ALL, TZ, TMPDIR, XDG_CONFIG_HOME, XDG_CACHE_HOME`.

**Network conventions:** `HTTP_PROXY, HTTPS_PROXY, NO_PROXY, http_proxy, https_proxy, no_proxy, SSL_CERT_FILE, SSL_CERT_DIR, CURL_CA_BUNDLE`.

**Plus** per-binding `env` resolved values. `os.Environ()` NOT inherited.

### REV-9 — Round-history aggregation: DEFERRED

F.7.18 dropped round-history aggregation entirely. F.7-CORE F.7.9's `spawn_history[]` is audit-only. Doc-comment on `spawn_history[]` MUST cite audit-only role + link to round-history-deferred decision in F.7.18 commentary. Owner of the doc-comment requirement: F.7.18.6 (per master PLAN §5).

### REV-16 — F.7.8 adapter-routing surface deferred to Drop 4d (codex)

F.7.8 acceptance criteria L562 / L566 originally specified `OrphanScan(ctx, repo, adapterRegistry CLIAdapterRegistry)` routing PID-liveness through `adapterRegistry.Get(manifest.CLIKind).IsPIDAlive(pid)`. F.7.17.10's `CLIAdapter` interface is locked to three methods (`BuildCommand`, `ParseStreamEvent`, `ExtractTerminalReport`) — no `IsPIDAlive` method. The two specs are mutually inconsistent.

**Resolution**: F.7.8 ships generic `ProcessChecker` interface + `DefaultProcessChecker`. `OnOrphanFound` callback receives the action item which carries `manifest.CLIKind` via the bundle's `metadata.spawn_bundle_path` → `ReadManifest` chain. Future codex adapter (Drop 4d) introduces a per-CLIKind ProcessChecker registry (or a typed-lookup helper) without changing the OrphanScanner API. F.7.8 does NOT extend the CLIAdapter interface today.

**Why not extend CLIAdapter**: F.7.17.10 locks the interface as a public contract for adapter authors. Adding `IsPIDAlive` (which is OS-level POSIX semantics, not CLI-specific) would conflate concerns.

### REV-15 — F.7.3a absorbed into F.7.17.3 + F.7.18.4 absorbed into F.7.18.3 + F.7.18.6 absorbed into F.7.9

Three droplets absorbed during build:

- **F.7.3a (claude argv emission)** — F.7.17.3 builder shipped `internal/app/dispatcher/cli_claude/argv.go` containing `assembleArgv(BindingResolved, BundlePaths) []string` with the full headless claude argv recipe per spawn architecture memory §3. F.7.3a is therefore redundant; do NOT dispatch separately.
- **F.7.18.4 (greedy-fit + caps)** — F.7.18.3 builder shipped greedy-fit + per-rule + per-bundle wall-clock caps inline in the aggregator engine. Verified by F.7.18.3 QA-Proof which confirmed the algorithm semantics. F.7.18.4 is therefore redundant.
- **F.7.18.6 (`spawn_history[]` doc-comment)** — F.7.9 builder shipped the doc-comment citing audit-only role + linking to F.7.18 round-history-deferred decision per the spawn-prompt directive. F.7.18.6 is therefore redundant.

Net droplet count: -3. Drop 4c total: 31 - 3 = 28 droplets active.

The remaining F.7.3b (bundle render — system-prompt.md, settings.json, agents/<name>.md, plugin.json, .mcp.json) is still distinct from F.7.17.3 (claudeAdapter struct + argv) and remains in scope.

### REV-14 — F.7.5a/b/c rescoped: `permission_grants` table FULLY OWNED by F.7.17.7

F.7.17.7 (committed pending) ships the entire `permission_grants` table including `cli_kind` column from inception. F.7-CORE F.7.5 was originally split into:
- F.7.5a: permission_grants table + storage
- F.7.5b: TUI handshake
- F.7.5c: settings.json grant injection

**Rescoping post-F.7.17.7:** F.7.5a is REMOVED — F.7.17.7 absorbs it (the table + storage adapter + 3 interface methods). F.7.5b (TUI handshake parsing `permission_denials[]` → attention-item posting) and F.7.5c (settings.json grant injection per-spawn) remain in scope; both consume the F.7.17.7-shipped `permission_grants` storage.

Net droplet count adjustment: F.7-CORE F.7.5a removed → F.7-CORE 19 → 18. Total drop droplet count: 35 → 34.

The F.7.5b spawn prompt MUST reference `app.PermissionGrantsStore` interface from `internal/app/permission_grants_store.go` (already merged via F.7.17.7) rather than re-creating the storage layer.

### REV-11 — F.7.2 worktree-escape check DEFERRED to spawn-time (F.7.3b / claudeAdapter)

The F.7-CORE F.7.2 acceptance body (lines ~228, ~236) mandated `validateAgentBindingSandbox rejects AllowWrite paths that escape the project worktree (resolved via filepath.Clean + ancestry check)` plus an explicit `allow_write = ["/etc"]` reject example. This requirement is **deferred to spawn-time** because `templates.Load(reader)` has no project context — the loader parses TOML without knowing which project's worktree is the ancestry root.

**Where the check actually lives**: F.7.3b bundle render OR claudeAdapter `assembleEnv` / `BuildCommand` at spawn time, where the project worktree path is available from `BindingResolved` / `BundlePaths.Root` context. The schema-layer validators in F.7.2 enforce only path-shape rules (absolute, no `..`, no `//`); semantic worktree-ancestry enforcement is a render-layer concern.

F.7.2 ships as committed at `f6aec8b`. F.7.3b acceptance criteria gain the worktree-ancestry check requirement (orchestrator surfaces this when dispatching F.7.3b).

### REV-12 — `AgentBinding.Validate()` asymmetry on F.7.2 fields DEFERRED to refinements

QA Falsification on F.7.2 (`workflow/drop_4c/4c_F7_2_BUILDER_QA_FALSIFICATION.md` A11) flagged that `AgentBinding.Validate()` (in-memory programmatic validation) was not extended with the new tool-gating + sandbox + system-prompt-template field checks. The TOML-load path (`templates.Load`) catches malformed values; the programmatic construction path does NOT.

**Why deferred**: there are no in-tree consumers of programmatic `AgentBinding` construction today — every path goes through TOML decode. The asymmetry is theoretical until F.7.5+ dispatcher flows arrive. Per `feedback_no_migration_logic_pre_mvp.md` spirit (no code for hypothetical scenarios), defer the symmetry fix to a refinements droplet that lands once a programmatic-construction consumer actually appears.

### REV-13 — Builder spawn prompts MUST explicitly forbid self-commit

QA Falsification on F.7.2 confirmed builder self-committed `f6aec8b` before QA pair fired (process violation per `feedback_qa_before_commit.md` + `feedback_orchestrator_commits_directly.md`). Going forward, every builder spawn prompt MUST include the directive: *"You are NOT permitted to run `git commit` or `git add ... && git commit`. The orchestrator drives commits AFTER QA pair (proof + falsification) returns green. End your work by writing the worklog and reporting completion to the orchestrator-facing response."*

Procedural fix; no code change.

### REV-10 — Dispatch sequencing (post-revisions)

Net droplet count: 16 (F.7-CORE original) +1 (F.7.3 split) +2 (F.7.5 split) = 19 in F.7-CORE.

Combined with F.7.17 (10 after marketplace droplet removal) + F.7.18 (6) = **35 droplets total in Drop 4c**.

---

## Hylla Feedback

`N/A — planning touched non-Go files only` (the SKETCH.md, planner-review MD, falsification-R2 MD, and spawn architecture memory). The Go symbols cited (e.g. `domain.ActionItem`, `templates.AgentBinding`, `internal/app/dispatcher/spawn.go:assemblePrompt`) were verified via direct `Read` on the source files — same pattern as the planner-review document established. No Hylla queries issued; no miss to record.
