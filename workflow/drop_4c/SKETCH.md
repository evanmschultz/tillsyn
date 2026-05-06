# Drop 4c — Pre-Drop-5 Refinement Drop (Sketch)

**Status:** placeholder — NOT a full plan. Full PLAN.md authoring + parallel-planner dispatch + plan-QA twins land post-Drop-4b-merge before any builder fires.
**Author date:** 2026-05-03 (during Drop 4a planning phase).
**Purpose:** capture deferred-from-4a-and-4b items so nothing gets lost when planning starts post-4b.

## Naming

**Drop 4c** = the third drop in the 4-series. Drop 4a (dispatcher core) → Drop 4b (gate execution) → Drop 4c (pre-Drop-5 polish + audit-debt sweep) → Drop 4.5 (TUI overhaul, concurrent with Drop 5) + Drop 5 (dogfood validation).

Descriptive name: "pre-Drop-5 refinement drop." Functional name: "Drop 4c."

## Goal

Bundle the deferred polish + agent-facing hardening items that surfaced during the audit/build of Drops 4a + 4b but were explicitly out-of-scope for those drops. Lands BEFORE Drop 5 dogfooding so the cascade-on-itself loop isn't fighting silent-data-loss bugs or other agent-surface warts. Does NOT block Drop 4.5 (concurrent FE/TUI track).

End-state: cascade-on-itself loop has no known agent-surface footguns; pre-MVP audit gaps closed; dev-hygiene tooling complete.

## In-Scope Items (Captured From Audits)

The list grows as Drops 4a + 4b run. Initial seed items:

### From the Drop 1 audit (PLAN.md §19.1 items still missing post-Drop-4a-and-4b)

- **PATCH semantics on update handlers.** `Service.UpdateActionItem` calls `actionItem.UpdateDetails(...)` which writes every field unconditionally — `update(title="foo")` with empty `description` wipes the stored description. Silent-data-loss bug confirmed in pre-Drop-3 18.10B closeout. Fix: pointer-sentinel input fields (or explicit-replace-all flag) so omitted fields preserve.
- **Reject unknown keys at MCP boundary.** No `DisallowUnknownFields` across MCP server code. Schema-permissive create/update silently drops unknown fields. Surfaced in pre-Drop-3 audits. Fix: per-MCP-tool unknown-key rejection with structured error naming the offending key.
- **Server-infer `client_type` on auth request create.** Currently accepts empty `client_type`; only the approve-path rejects it (asymmetric validation bug). Fix: MCP-stdio adapter stamps `"mcp-stdio"`, TUI stamps `"tui"`, CLI stamps `"cli"`. Tighten `app.Service.CreateAuthRequest` to reject empty `ClientType`.
- **Supersede CLI**: `till action_item supersede <id> --reason "..."` — marks `failed` action item as `metadata.outcome: "superseded"` and transitions `failed → complete`. Bypasses the always-on parent-blocks invariant landed in Drop 4a Wave 1.7. Hard requirement before dogfood — without it, every `failed` child stuck-state requires dev fresh-DB.
- **CLI failure listing**: `till action_item list --state failed` (or `till failures list`). Dev visibility into `failed` action items pre-TUI-rendering (Drop 4.5).
- **Require non-empty outcome on `failed`.** `validateMetadataOutcome` accepts empty outcome. Domain-level validation: any transition to `failed` requires non-empty `metadata.outcome`.
- **`go.mod` `replace` directive cleanup.** Strip every `replace` directive except the fantasy-fork. Pre-cascade hygiene from PLAN.md §19.1 first bullet.

### From Drop 3 refinements memory (`project_drop_3_refinements_raised.md`)

- **R1 — STEWARD field-level guard symmetry on `Persistent` / `DevGated`.** `assertOwnerStateGateUpdateFields` rejects agent-principal Update calls that mutate Owner / DropNumber on STEWARD-owned items, but does NOT reject Persistent / DevGated mutations. Fix: extend to all four fields per L13 (domain primitives, not STEWARD-specific).
- **R2 — `raiseRefinementsGateForgottenAttention` doc-comment idempotency drift.** Helper godoc claims idempotent duplicate-insert handling that the implementation doesn't perform. Fix: either prepend `FindAttentionItemByExternalID`-style lookup OR trim the godoc claim.
- **R3 — `isRefinementsGate` future-cascade precision.** Today the predicate keys on `Owner=STEWARD + StructuralType=Confluence + DropNumber>0`; if a future cascade variant introduces a second STEWARD-owned numbered confluence kind, false-positive risk. Fix: tighten predicate with `Title` shape match OR add `Kind` discriminator.
- **R5 — WIKI Cross-Subtree Exception kind-choice hedge.** WIKI says `kind=closeout or kind=plan ... as appropriate` for ledger / wiki-changelog rollups — under-specified pending STEWARD's actual rollup-kind choices stabilizing. Fix: re-survey post-Drop-4 STEWARD usage; pick canonical rollup kinds.

### From Drop 4a/4b refinements memory (TBD, populated during builds)

- Any plan-QA-falsification PASS-WITH-NIT findings that don't get fixed in their drop.
- Any LSP / gopls-vet hints that surface during builds (mirrors Drop 3 R4 pattern).
- Any audit-gap accept items from outside-repo edits in 4a.32 + 4b's equivalent.
- Open questions Q1–Q12 from Drop 4a's PLAN.md §10 that resolve as "accepted as-is, document for future revisit" — those documented decisions belong here as candidate fixes if the dogfood window flags them.

## Out of Scope

- Anything that lands in Drops 4a or 4b before Drop 4c starts. Drop 4c is the deferred-residue, not duplicate work.
- Drop 4.5 scope (TUI overhaul, columns-table retirement, file-viewer pane).
- Drop 5 scope (dogfood validation).
- Migration logic — pre-MVP rule still in force.

## Tentative Wave / Item Structure

~28–40 droplets (Theme F.7 spawn redesign adds 13–18 with F.7.17 CLI adapter seam + F.7.18 context aggregator added 2026-05-04, refined post-plan-QA-falsification 2026-05-04 with security tightening: list-form `command` + closed env baseline + per-rule + total-cap + wall-clock-cap aggregator safeguards, round-history deferred YAGNI, planner-descendants schema rule dropped per flexibility framing). Loose theme grouping (subject to revision):

### Theme A — Silent-data-loss + agent-surface hardening (~4 droplets)

- PATCH semantics on update handlers (per item ).
- Reject unknown keys at MCP boundary.
- Server-infer `client_type` on auth-request create.
- Require non-empty outcome on `failed` transitions.

### Theme B — Dev-facing escape hatches (~2 droplets)

- Supersede CLI (`till action_item supersede`).
- CLI failure listing (`till action_item list --state failed`).

### Theme C — STEWARD + cascade-precision refinements (~3 droplets)

- R1: extend `assertOwnerStateGateUpdateFields` to Persistent + DevGated.
- R2: tighten `raiseRefinementsGateForgottenAttention` doc-comment vs idempotency.
- R3: tighten `isRefinementsGate` predicate.
- R5: re-survey WIKI Cross-Subtree Exception kind-choice (may be MD-only sweep).

### Theme D — Pre-cascade hygiene (~1–2 droplets)

- `go.mod` `replace` directive cleanup.
- Any LSP / gopls-vet hints accumulated through 4a + 4b builds.

### Theme E — Drop-4a/4b-residue (TBD)

- Populated during 4a + 4b builds. Whatever surfaces in `project_drop_4a_refinements_raised.md` and the 4b equivalent that doesn't fit in those drops' scopes lands here.

### Theme F — Template ergonomics (~15–18 droplets)

The big theme. Drop 3 landed the template foundation; Drop 4a's dispatcher consumes it; today the loading + management surface is unfinished. Theme F closes the gaps so adopters can actually use the template system.

**F.1 — Project-template auto-discovery (~3 droplets).** Wire `internal/app/service.go` `loadProjectTemplate()` (currently returns `(zero, false, nil)` per Drop 3.14 deferral) to walk `<project.RepoBareRoot>/.tillsyn/template.toml` first, fall back to `<project.RepoPrimaryWorktree>/.tillsyn/template.toml`, fall back to embedded `default.toml`. Each candidate runs through `templates.Load(r io.Reader)` for full validation. Position-aware errors surface to project-create.

**F.2 — Generic + Go + FE builtin separation (~4 droplets).** Refactor `internal/templates/builtin/`:
- `default-generic.toml` — language-agnostic cascade-vocabulary showcase. `agent_bindings` either empty (project must override) OR placeholder `agent_name = "{language}-builder-agent"` resolved at bake time via `project.Language`.
- `default-go.toml` — generic + Go agent bindings (current `default.toml` content rebadged).
- `default-fe.toml` — generic + FE agent bindings.
- `internal/templates/embed.go` resolver picks the right template based on `project.Language` at bake time.
- This repo gets its own `<project_root>/.tillsyn/template.toml` (NEW file) for tillsyn-self-host dogfood; references `default-go.toml` semantics + tillsyn-flavored agent bindings.

**F.3 — `till.template` MCP tool (~3 droplets).** Operations:
- `till.template(operation=get, project_id=...)` — return project's current template + bake state.
- `till.template(operation=validate, content=<toml-string>)` — run full validation chain on candidate TOML; return findings.
- `till.template(operation=set, project_id=..., content=<toml-string>)` — validate + install + re-bake catalog.
- `till.template(operation=list_builtin)` — enumerate shipped builtins.

**Wire-format decision (locked):** TOML in, TOML out. The MCP argument `content` is a string carrying TOML text verbatim. Server parses TOML, validates, persists TOML. Templates are TOML end-to-end; the MCP transport (JSON-RPC) just carries TOML-as-a-string. Same shape as `cat template.toml | till template validate -` for CLI symmetry.

**F.4 — Marketplace CLI (~5 droplets).** Separate git repo (default: `github.com/evanmschultz/tillsyn-templates`) holds curated cascade templates. Tillsyn binary integration:
- `internal/templates/marketplace.go` — git-shell-out wrapper. `git clone --depth 1` on first fetch, `git pull` on update. Cache to `~/.tillsyn/marketplace/`. Repo URL configurable via `~/.tillsyn/config.toml`.
- `till template list` — show locally cached templates + last-fetched git commit + commit date + git log.
- `till template fetch [--remote <url>]` — clone or pull. Show commit-log delta on update.
- `till template show <name>` — print template content + meta.
- `till template install <name>` — copy to `<project>/.tillsyn/template.toml`.
- `till template validate <path>` — run full validation chain; emit findings. (Useful for marketplace contributors.)

**F.5 — Extended validation (~2 droplets).** New checks layered into `templates.Load`:
- `validateAgentBindingFiles` (warn-only) — walk `agent_bindings.<kind>.agent_name`; check `~/.claude/agents/<agent_name>.md` exists. Soft warning since the file might not be installed yet on this dev's machine; emit a finding without rejecting.
- `validateRequiredChildRules` (error) — enumerate canonical "QA-required" parent kinds (`build`, `plan`); assert each has the mandatory child rules (`build-qa-proof` + `build-qa-falsification`; `plan-qa-proof` + `plan-qa-falsification`). Reject templates that silently remove QA rules.
- `validateChildRuleReachability` — currently a no-op extension point. Grow into kind-orphan detection (every kind reachable from `plan` via child_rules; orphans flagged as findings).
- (Optional) `validateKindStructuralCoherence` — light cross-axis check that each kind's structural-type expectation matches its work pattern (`build` is a leaf, `plan` recurses, `closeout` is drop-end). Soft warning.

**F.6 — Cleanup of legacy KindTemplate stub (~1 droplet).** `internal/app/kind_capability.go:1002` `mergeActionItemMetadataWithKindTemplate` is a no-op pass-through stub kept "during the transition." Drop 3.15 retired the legacy KindTemplate surface; this stub can fold into its caller (`internal/app/service.go:716`). Doc comment confirms: *"a future drop will fold it into the caller."* Drop 4c is that future drop.

**F.7 — Spawn pipeline redesign (~10–14 droplets — primary Theme F focus, replaces 4a.19 stub wholesale).** Drop 4a's `internal/app/dispatcher/spawn.go` is a stub: hardcodes `hylla_artifact_ref` (Hylla is local-only, not part of Tillsyn's shipped cascade — must remove), emits `--mcp-config <stub-path>`, hand-rolls a prompt with no `--system-prompt-file` / `--settings` / `--plugin-dir` plumbing, no stream-json output capture. Drop 4c replaces it wholesale with the architecture canonized in `~/.claude/projects/-Users-evanschultz-Documents-Code-hylla-tillsyn/memory/project_drop_4c_spawn_architecture.md`. Itemized:

- **F.7.1 — Per-spawn temp bundle lifecycle.** `os.MkdirTemp` per spawn (root chosen by `tillsyn.spawn_temp_root = "os_tmp" | "project"` TOML knob, default `os_tmp`). Bundle layout: `manifest.json` + `plugin/` (with `.claude-plugin/plugin.json` + `agents/<name>.md` + `.mcp.json` + `settings.json`) + `system-prompt.md` + optional `system-append.md` + `stream.jsonl` capture. `defer os.RemoveAll` on spawn exit; cleanup on terminal-state transition (complete/failed/archived).

- **F.7.2 — TOML template schema widening.** Add `[agent_bindings.<kind>]` fields: `tools_allowed`, `tools_disallowed`, `system_prompt_template_path`, `[agent.sandbox.filesystem]` allowWrite/denyRead, `[agent.sandbox.network]` allowed_domains/denied_domains, optional `tools_engine_minimal` (renders to `--tools "..."` flag). Tool-gating render strategy: settings.json `permissions` deny rules are AUTHORITATIVE (Layer B); agent-file frontmatter `disallowedTools` mirrors B for human readability (Layer A); CLI `--allowed-tools`/`--disallowed-tools` flags SKIPPED for typical kinds (probe-grounded — agents route around removed tools via Bash, only deny patterns catch workarounds).

- **F.7.3 — Headless argv emission.** Spawn.go emits per memory §3 recipe: `--bare`, `--plugin-dir <bundle>/plugin`, `--agent <name>`, `--system-prompt-file <bundle>/system-prompt.md`, `--settings <bundle>/plugin/settings.json`, `--setting-sources ""` (Tillsyn's settings is sole source — user/project/local ignored), `--strict-mcp-config`, `--permission-mode acceptEdits`, `--output-format stream-json`, `--verbose`, `--no-session-persistence`, `--exclude-dynamic-system-prompt-sections`, plus conditional `--max-budget-usd` / `--max-turns` / `--effort` / `--model` / `--append-system-prompt-file` / `--tools` (each emitted only when value resolves through CLI > MCP > TUI > TOML > absent priority cascade; spawn.go uses `*int` / `*float64` / `*string` types).

- **F.7.4 — Stream-JSON monitor parser.** Parse `stream.jsonl` line-by-line per memory §6 taxonomy: `system/init` (verify tool list rendered correctly), `assistant` (text/thinking/tool_use blocks), `user` (tool_result with is_error mid-stream), `result` (terminal — `total_cost_usd`, `permission_denials[]`, `terminal_reason`, `errors[]`). Tillsyn writes per-spawn cost to action item `metadata.actual_cost_usd`. **Drop 4a 4a.21 process monitor stays minimal** (PID + exit watch); F.7.4 LAYERS the stream parser on top.

- **F.7.5 — Permission-denial → TUI handshake.** On terminal `result` event, parse `permission_denials[]`. For each `{tool_name, tool_input}` pair, post a Tillsyn attention-item to dev's TUI: "Agent X for kind Y wants to call <tool> with <args>. Allow once / Allow always / Deny." Dev approves → SQLite `permission_grants(project_id, kind, rule, granted_by, granted_at)` row written. Next spawn of same kind reads grants, injects into per-spawn `settings.json`. Real-time mid-stream variant (watch `tool_result is_error: true`) is optional Drop 4c+.

- **F.7.6 — Required system-plugin pre-flight check.** `till bootstrap` (or per-dispatch pre-flight) shells out to `claude plugin list --json`, parses installed-plugin set, fails hard if any project TOML `tillsyn.requires_plugins = [...]` entry is missing with clear instruction: `Run: claude plugin install <name>`. OSS-friendly team-standards enforcement.

- **F.7.7 — Auto-add `.tillsyn/spawns/` to `.gitignore`** when `spawn_temp_root = "project"` mode AND project doesn't already have it ignored. Skipped in `os_tmp` default mode.

- **F.7.8 — Crash-recovery / orphan scan.** On Tillsyn startup, enumerate every `in_progress` action item, read `<bundle>/manifest.json` → `claude_pid`, check PID liveness via `os.FindProcess` + signal 0 + cmdline match. Live → leave (re-monitor via SQLite state). Dead → move action item to `failed` with `metadata.failure_reason = "dispatcher_restart_orphan"` + cleanup bundle + dev decides re-dispatch.

- **F.7.9 — Action-item metadata fields.** New: `metadata.spawn_bundle_path`, `metadata.spawn_history[]` (append-only audit trail of `{spawn_id, bundle_path, started_at, terminated_at, outcome, total_cost_usd}`), `metadata.actual_cost_usd`. Wire into `domain.ActionItem` if needed; otherwise metadata blob.

- **F.7.10 — Drop `hylla_artifact_ref` from spawn.go's prompt body.** Hylla is dev-local, NOT part of Tillsyn's shipped cascade. Remove the hardcoded reference from `assemblePrompt` (it currently leaks Tillsyn-internal Hylla awareness into every spawn). Local Tillsyn template can include Hylla MCP server in its plugin bundle if dev opts in; shipped Tillsyn binary has zero Hylla awareness.

- **F.7.11 — Documentation: write Tillsyn architecture docs** referencing memory `project_drop_4c_spawn_architecture.md` as canonical source. Cover: two plugin paths (system-installed vs --plugin-dir bundle), per-spawn temp file inventory, stream-json event taxonomy, settings.json authority, sandbox semantics, crash recovery, explicit non-goals (adversarial OS sandbox for Read/Edit/Write, real-time interactive prompts).

**F.7.17 — CLI adapter seam (`CLIKind` enum + `CLIAdapter` interface).** Architects the spawn pipeline for multi-CLI extensibility WITHOUT shipping the second adapter inside Drop 4c. Drop 4d (post-Drop-4c-merge; lands BEFORE Drop 5 multi-CLI dogfood) lands the `codex` adapter. Later drops MAY extend the seam to non-JSONL CLIs (SSE / framed-binary / no-stream); Drop 4c scope is JSONL-stream only.

- **`CLIKind` closed enum** in `internal/app/dispatcher/`: today `claude`; Drop 4d adds `codex`. Lives on `[agent_bindings.<kind>] cli_kind = "claude"` (default omitted = `claude` for backward-compat).
- **`CLIAdapter` interface** with three methods: `BuildCommand(ctx, BindingResolved, BundlePaths) (*exec.Cmd, error)`, `ParseStreamEvent(line []byte) (StreamEvent, error)`, `ExtractTerminalReport(StreamEvent) (TerminalReport, bool)`. Each adapter owns its CLI's argv shape + event taxonomy. Both adapters in scope (claude, codex) emit newline-delimited JSON, so the byte-line signature is correct for the JSONL family. **Non-JSONL extensibility (SSE / framed / no-stream) is a roadmap concern (see "Multi-CLI roadmap" below; hard-cut interface rewrite, no backward compat).**
- **No `command` override field. No wrapper-interop knob.** Tillsyn invokes the CLI binary directly via the adapter (claude adapter calls `claude`; codex adapter calls `codex`). Adopters who want process isolation use OS-level mechanisms (PATH-shadowed `claude` shim, symlink replacement, container wrapping the entire Tillsyn binary, sandbox-exec) — Tillsyn does not surface a wrapper integration point. **Why this is right:** removing `command` entirely closes the marketplace-RCE vector (no shell-interpreter denylist needed, no per-token argv regex needed, no install-time argv-list confirmation needed) while keeping the architecture honest. Tillsyn's job is "spawn claude / codex headless"; the OS's job is "isolate processes."
- **Per-binding TOML overrides** (Tillsyn never holds secrets — env var NAMES only; closed POSIX env baseline; spawn pipeline targets POSIX (macOS / Linux) only):
  - `[agent_bindings.<kind>] cli_kind = "claude"` (default omitted = `claude`; Drop 4d adds `"codex"`). Routes the dispatcher to the adapter for that CLI. Internal dispatch concern, NOT a binary-path override.
  - `[agent_bindings.<kind>] env = ["ANTHROPIC_API_KEY", "https_proxy"]` — list of env-var NAMES to forward. Resolution: `os.Getenv(name)` at spawn time; missing required-env fails loud at SPAWN time with a structured error naming the missing var.
- **Closed POSIX env baseline** (CRITICAL: `os.Environ()` is NOT inherited). Spawn `cmd.Env` is set explicitly to:
  - **Process basics**: `PATH` (value `os.Getenv("PATH")` — inherit-PATH so the spawn finds `claude` / `codex` on the user's normal PATH), `HOME`, `USER`, `LANG`, `LC_ALL`, `TZ`, `TMPDIR`, `XDG_CONFIG_HOME`, `XDG_CACHE_HOME`.
  - **Network conventions** (corporate-network adopters get a working spawn out-of-box; none carry secrets): `HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY`, `http_proxy`, `https_proxy`, `no_proxy` (POSIX-canonical lowercase variants), `SSL_CERT_FILE`, `SSL_CERT_DIR`, `CURL_CA_BUNDLE`.
  - **Plus** the resolved values for each name in the binding's `env` list.
  - The closed-baseline purpose is to block `AWS_*` / `STRIPE_*` direnv-style secret-bearing names from leaking into the spawn, NOT to break network connectivity for corporate adopters.
- **Schema validation on `env` (load-time):**
  - Each entry MUST match the env-var-name regex `^[A-Za-z][A-Za-z0-9_]*$` — alphanumerics + underscore, leading letter. Allows BOTH uppercase (`HTTP_PROXY`) and lowercase (`https_proxy`, the conventional cURL form). Rejects values containing `=`, whitespace, dashes, dots. Bake into `validateAgentBindingEnvNames`.
  - Empty strings rejected. Duplicates rejected.
- **No OAuth registry, no Docker awareness, no container model, no wrapper-interop in Tillsyn core.** All process-isolation concerns live outside Tillsyn (OS-level wrappers, container runtimes, PATH shadowing).

**Multi-CLI roadmap (PLAN.md-bound).** Drop 4c ships claude only. Drop 4d ships codex (JSONL family — same `(line []byte) → StreamEvent` interface shape). To add a non-JSONL CLI later (SSE / WebSocket / framed-binary / no-stream), the path is a **hard-cut interface rewrite** (not backward-compat — pre-MVP rule "no tech debt; if legacy isn't right, kill it"):

1. **Rewrite the `CLIAdapter` interface** in one drop: `ParseStreamEvent(line []byte)` is REMOVED, `ConsumeStream(ctx, io.Reader, sink chan<- StreamEvent) error` is added. ALL existing adapters (claude, codex) AND the dispatcher monitor are refactored in the same drop. JSONL adapters implement `ConsumeStream` by looping `bufio.Scanner` over the reader internally. Coordinated breaking change across the dispatcher subtree — no compat shim, no add-then-deprecate transition.
2. **Generalize `TerminalReport` value object** to carry optional pointer-cost / pointer-denials so CLIs lacking those telemetry channels can signal absence cleanly. Drop 4c ships `TerminalReport struct { Cost *float64; Denials []ToolDenial; Reason string; Errors []string }` from the start — third method renamed `ExtractTerminalReport(StreamEvent) (TerminalReport, bool)` so the method name matches the return type.
3. **Document the seam's assumed adapter properties**: process-per-spawn, exit-code authoritative, stderr is not the event channel. CLIs violating any of these properties (daemon-mode, ambiguous exit code, stderr-as-events) need a different adapter family, not a wider `CLIAdapter` interface.
4. **What it would take to add a specific non-JSONL CLI** (Goose SSE, Aider post-run-JSON, Cursor framed-binary): the upfront `ConsumeStream` rewrite (2–3 droplets, touches every existing adapter + monitor); then the new adapter (5–8 droplets per non-JSONL CLI: write the adapter; populate `TerminalReport` from whatever the CLI's terminal channel happens to be; add the `CLIKind` enum value; add tests against a recorded fixture stream).

PLAN.md MUST surface this roadmap so future-adapter contributors know the path. Drop 4c does NOT pre-bake the `ConsumeStream` push model — YAGNI for two known JSONL adapters.

**MockAdapter test fixture** (Drop 4c F.7.17 acceptance criterion). Drop 4c ships `internal/app/dispatcher/cli_adapter_test.go` with a `MockAdapter` exercising the `CLIAdapter` interface contract WITHOUT touching `claude` or `codex` binaries. Asserts: (a) `BuildCommand` returns an `*exec.Cmd` with the expected `Path`, `Args`, `Env` shape; (b) `ParseStreamEvent` round-trips a recorded fixture line; (c) `ExtractTerminalReport` correctly populates `TerminalReport` from a recorded terminal-event fixture. Confirms the seam is multi-adapter-ready before Drop 4d adds the second real adapter.

**Drop 4d preview (NOT in 4c scope; the SECOND-CLI dogfood drop).** ~7–10 droplets adding `codex` adapter on top of the F.7.17 seam. Codex CLI verified via `codex --help` + `codex exec --help` 2026-05-04: supports `codex exec --json`, `--profile <name>` references `$CODEX_HOME/config.toml` profiles, `--ignore-user-config`, `--ignore-rules`, `--sandbox`, `--ephemeral`. Maps cleanly onto `CLIAdapter`. **Sequencing: Drop 4c → Drop 5 (claude-only dogfood, validates the cascade-on-itself loop without conflating second-CLI integration risk) → Drop 4d (codex adapter) → Drop 5.5/6 (multi-CLI dogfood validation).**

**F.7.18 — Context aggregator (FLEXIBLE, not prescriptive).** Templates MAY declare what context to pre-stage into a spawn so an agent can call MCP only to update its own node. **This is OPTIONAL — Tillsyn supports the pattern but does not require it.** Templates that want full live MCP querying inside the spawn just leave the `[agent_bindings.<kind>.context]` table absent. Both paths are equally first-class — choose based on cost / latency / determinism preference, not based on Tillsyn-recommended path.

- **Schema (NEW closed-struct sub-table, all fields optional):** decoded into a named Go struct on `AgentBinding` (NOT `map[string]any`), so `templates.Load`'s existing strict-decode chain (`internal/templates/load.go:88-95`) automatically rejects unknown keys at load time. No new validator needed for unknown-key rejection.
  ```toml
  [agent_bindings.build.context]
  parent = true                        # render parent action-item details
  parent_git_diff = true               # capture git diff if parent has start_commit/end_commit
  siblings_by_kind = ["build-qa-proof", "build-qa-falsification"]  # latest round only
  ancestors_by_kind = ["plan"]         # walk up to first matching ancestor (semantics: first-match)
  descendants_by_kind = []             # walk down — usually empty; trust template authors to use sensibly
  delivery = "file"                    # "inline" (system-append.md) | "file" (<bundle>/context/*.md, agent uses Read)
  max_chars = 50000                    # per-rule budget; auto-truncate oversized rule with marker
  ```
- **Aggregator engine** lives in new `internal/app/dispatcher/context/` package. Pure-function `Resolve(binding, item, repo) (Bundle, error)` returns `{rendered_inline string, files map[string][]byte}`. **Per-binding scope only** — `Resolve` consults `binding.Context` for the spawning kind; other bindings in the catalog are not iterated. Spawn pipeline writes `files` under `<bundle>/context/`, appends `rendered_inline` to system-append.md, then exits.
- **Default-template seeds** (just defaults — projects override). Note: these are seeds in `internal/templates/builtin/default-go.toml`, builder-edited per `feedback_orchestrator_no_build.md`. Adopters writing project-local templates pick whatever shape fits their cascade.
  - **Spawn-prompt always-delivered baseline** (NOT aggregator-controlled — orthogonal): every spawn's `system-prompt.md` carries the action-item shape (id, kind, parent_id, paths, packages, completion_contract, metadata.outcome / blocked_reason if present), the agent's auth `session_id` for `till.*` MCP calls, and the working-directory + bundle paths. F.7.18 aggregator stages ADDITIONAL relational context on top of this baseline; it does NOT replace it.
  - `build`: parent + `parent_git_diff` + `ancestors_by_kind = ["plan"]` (delivery=file). Builder gets the diff pre-staged because it's the building lens (see what's already changed in the parent).
  - `build-qa-proof`: parent + `ancestors_by_kind = ["plan"]` + sibling builder's worklog if shipped via metadata. **NO `parent_git_diff`** — QA verifies independently by running `git diff` itself via Bash + `Read` tools. Pre-staging the diff would bias QA toward the builder's framing; QA must pull its own evidence to genuinely verify claims.
  - `build-qa-falsification`: same as proof but with falsification framing. Same NO `parent_git_diff` rationale — falsification needs to attack independently, not against a pre-rendered narrative.
  - `plan-qa-proof` / `plan-qa-falsification`: parent + `ancestors_by_kind = ["plan"]` up to root. No git diff needed (plan QA reviews planning artifacts, not code).
  - `plan` (planner agent runs the `kind=plan` binding): parent + `ancestors_by_kind = ["plan"]`. **No schema rule against `descendants_by_kind` on `kind=plan`** — template authors trusted to use the field appropriately. Use cases like round-history fix-planners or tree-pruner planners legitimately need descendants.
  - **Why builder gets diff but QA doesn't:** different lenses. Builder is implementing — pre-staging "here's what's already in the parent's diff" reduces redundant tool calls. QA is verifying — pre-staging "here's the diff" pre-frames the analysis. Independent QA verification is load-bearing for the cascade-on-itself loop's trustworthiness.
- **Token-budget safeguards.**
  - **Per-rule char cap**: `max_chars` default 50KB. A rule whose rendered output exceeds its `max_chars` is truncated mid-content with a `[truncated to <N> chars; full content at <bundle>/context/<rule>.full]` marker. The full content is written to disk for the agent to Read tool if needed.
  - **Total-bundle char cap (greedy-fit)**: `[tillsyn] max_context_bundle_chars = 200000` (default). Algorithm: iterate rules in **TOML declaration order**; for each rule, if `cumulative + rule_size <= cap`, include it (cumulative += rule_size); else **SKIP this rule with a `[skipped: <rule_name> (would have added <N> chars; bundle remaining = <M>)]` marker AND CONTINUE to subsequent rules**. Subsequent rules that fit still land. Adopters who want strict-priority order can reduce later rules' `max_chars` so they don't ever evict an earlier-busting rule. Greedy chosen over serial-drop because cascade adopters typically order context as "primary then supporting" not "monotonically decreasing priority" — landing what fits is more useful than a deterministic stop-point at the first bust.
  - **Per-rule wall-clock cap**: `[agent_bindings.<kind>.context.limits] max_rule_duration = "500ms"` (default). Each rule independently enforces its own timeout via `context.WithTimeout`. Slow rule (e.g. `parent_git_diff` on a 10000-line change) times out at 500ms with a per-rule marker `[rule <name> timed out after <duration>; partial output discarded]`; subsequent rules continue.
  - **Per-bundle wall-clock cap (hard ceiling)**: `[tillsyn] max_aggregator_duration = "2s"` (default). Aggregator-wide timeout via outer `context.WithTimeout`. If hit before per-rule timeouts catch slow rules, partial bundle + marker `[aggregator timed out after <duration>; rules pending: <list>]`. Two-axis design: per-rule cap localizes failures, per-bundle cap remains hard ceiling against pathological trees.
- **Round-history aggregation: DEFERRED.** YAGNI today — the high-signal fix-builder context (worklog MD, gate output, QA findings) comes from sources other than raw stream-json events. `metadata.spawn_history[]` (F.7.9) remains an audit trail (cost, denials, terminal_reason) for ledger / dashboard, not for re-prompting. If a concrete use case for raw stream-json round-history surfaces post-Drop-5, add it as a refinement-drop item with dedicated `prior_round_*` rules (`prior_round_worklog`, `prior_round_gate_output`, `prior_round_qa_findings`) that target the actual high-signal artifacts.
- **Why this is FLEXIBLE not REQUIRED.** Two equally first-class configurations:
  - **Bounded mode** (declare `[context]`): agent receives pre-staged context per the binding's `[context]` declaration (see per-binding seeds above for which kinds get which fields — NOTE: builder gets parent_git_diff, QA does NOT to preserve independent verification), calls MCP only on completion. Predictable cost, lower latency, less round-tripping.
  - **Agentic mode** (omit `[context]`): agent receives only its own action-item ID + system-prompt, calls MCP for whatever context it needs. Higher cost, more flexibility, more round-tripping.
  - Pick based on the agent kind's actual needs — neither is the recommended default. Default-go template picks bounded for cost predictability; default-generic ships empty `[context]` tables (omit).
- **Schema validation summary** (load-time, all in `templates.Load`):
  - `[agent_bindings.<kind>.context]` decodes into a closed struct with explicit TOML tags on every field → strict-decode rejects unknown keys automatically.
  - `delivery` MUST be `"inline"` or `"file"` (closed two-value enum).
  - `max_chars` MUST be positive int.
  - `max_rule_duration` MUST be positive duration.
  - `ancestors_by_kind` / `siblings_by_kind` / `descendants_by_kind` entries MUST reference valid kinds from the closed kind enum.
  - Cross-cap: `max_chars` per rule SHOULD NOT exceed `max_context_bundle_chars` (warn-only — single rule consuming the entire bundle is unusual but allowed).
- **Round-history future-contributor pointer (PLAN.md authoring concern).** F.7.9 droplet acceptance criteria MUST require a doc-comment on `metadata.spawn_history[]` citing its audit-only role and linking to the F.7.18 round-history-deferred decision. Survives the SKETCH-to-PLAN handoff so future contributors know to add `prior_round_*` rules (not raw stream-json round-history) if the use case surfaces post-Drop-5.

**F.7 absorbs Drop 4b Wave B (per Option β decision 2026-05-04):** Drop 4b deferred its commit-agent integration + commit gate + push gate + project-metadata toggles for commit/push because all four items depend on the spawn pipeline that F.7 replaces. F.7 absorbs them as additional sub-items:

- **F.7.12 — Commit-agent (haiku) integration via the new spawn pipeline.** `claude --agent commit-message-agent` invoked through F.7.1's per-spawn temp-bundle materialization (not the legacy 4a.19 stub path). Reads `git diff <action_item.start_commit>..<action_item.end_commit>` (Wave 1 first-class fields). Returns single-line conventional commit message. Tool gating per F.7.2's `[agent_bindings.commit].tools_allowed` (Read + Bash for git diff inspection only; nothing else).
- **F.7.13 — `commit` gate implementation.** Runs `git add <action_item.paths>` (path-scoped, never `git add -A`); runs `git commit -m "<haiku-output>"`; populates `action_item.end_commit = git rev-parse HEAD`. Honors project metadata `dispatcher_commit_enabled` toggle (default false; dogfood flips to true).
- **F.7.14 — `push` gate implementation.** Runs `git push origin <branch>` when project `dispatcher_push_enabled = true`. On failure: action item moves to `failed` with `metadata.BlockedReason = "git push: <error>"`. No auto-rollback of the local commit; surfaces to dev via attention-item.
- **F.7.15 — Project-metadata toggles** `dispatcher_commit_enabled bool` + `dispatcher_push_enabled bool` on `domain.ProjectMetadata`. Pointer-bool nil-means-disabled per Drop 4a 4a.25 precedent (default off until dogfood proves them safe).
- **F.7.16 — Default template `[gates.build]` expansion.** When F.7.13 + F.7.14 land, update `internal/templates/builtin/default.toml` `[gates.build]` from `["mage_ci"]` (Drop 4b state) to `["mage_ci", "commit", "push"]`. Each gate is independently toggleable via the project metadata flags.

**F.7 explicit non-goals** (carried forward from memory §11):
- Adversarial OS-level sandbox for Read/Edit/Write tools (cooperative deny rules sufficient for non-adversarial subagents; if ever needed, wrap entire `claude` invocation in Docker/Firejail).
- Real-time interactive permission prompts (Tillsyn's TUI cannot intercept Claude's stdin prompt; failure-loop handshake via terminal `permission_denials[]` is the design).
- Inheritance of orchestrator's CLAUDE.md / output styles / hooks (--bare skips them by design; per-kind system prompt template subsumes role definitions).

**F.7 dependencies + supersession:**
- Supersedes 4a.19 spawn.go entirely. NITs from 4a.19 (R6 in `project_drop_4a_refinements_raised.md`) die naturally with the rewrite.
- Builds on 4a.21 process monitor (PID watch) + 4a.22 cleanup hook (lock release on terminal state).
- Pre-flight check (F.7.6) needs `claude plugin list --json` parsing — depends on the Tillsyn-side plugin-config schema landing in F.7.2.
- F.7.17 (CLI adapter seam) is internal-refactor-only inside Drop 4c — Drop 4c ships only the `claude` adapter. Drop 4d lands `codex`. The seam exists pre-Drop-4d so Drop 4d is purely additive.
- F.7.18 (context aggregator) builds on F.7.1 (bundle layout) + F.7.2 (template schema widening) + F.7.3 (system-append.md plumbing). All three are F.7-internal so ordering is a planning concern only.
- **F.7 schema-additions sequencing (THREE schema droplets at start of F.7 wave).** Single-bundle was a SPOF — ~400 LOC review surface across multiple distinct additions, where one bug blocks the whole F.7 wave. Split into three sequential schema droplets, each ~1/3 the review surface with independent failure domains:
  1. **Schema-1: F.7.17 per-binding fields** — adds `Env []string` and `CLIKind string` to `AgentBinding` in `internal/templates/schema.go`. (Note: `Command` and `ArgsPrefix` were dropped from the design 2026-05-05 — Tillsyn does NOT surface a binary-path or wrapper override; OS-level isolation is the adopter's responsibility.) Includes `validateAgentBindingEnvNames` validator. Unit tests cover happy-path + every reject case (malformed env names, empty/duplicate entries, values containing `=`).
  2. **Schema-2: F.7.18 `Context` sub-struct on `AgentBinding`** — adds `Context ContextRules` field with TOML tag `context`. Closed sub-struct with `Parent`, `ParentGitDiff`, `SiblingsByKind`, `AncestorsByKind`, `DescendantsByKind`, `Delivery`, `MaxChars`, `MaxRuleDuration` fields, all explicit TOML tags. Validators: `delivery` two-value enum, `max_chars` positive int, `max_rule_duration` positive duration, kind references in `*_by_kind` slices match the closed kind enum.
  3. **Schema-3: F.7.18 `[tillsyn]` top-level globals** — adds NEW top-level `Tillsyn` struct in `internal/templates/schema.go` with two fields `MaxContextBundleChars int` (TOML tag `max_context_bundle_chars`) and `MaxAggregatorDuration Duration` (TOML tag `max_aggregator_duration`); adds NEW `Tillsyn Tillsyn` field on `Template` struct with TOML tag `tillsyn`. Validators reject zero / negative values on either field. **WITHOUT this droplet, `[tillsyn]` keys are rejected by strict-decode** (`internal/templates/load.go:88-95` `DisallowUnknownFields()`). Unit test asserts an unknown-key TOML payload fails strict-decode for the new struct, proving the closed-struct unknown-key rejection actually fires.
- **Sequencing constraint**: Schema-1 → Schema-2 → Schema-3 must land in order. Per-binding seeds (default-template `[context]` blocks) MUST land AFTER Schema-2; `[tillsyn]` block in default templates MUST land AFTER Schema-3. Other F.7 droplets that consume the wider struct land after their respective schema droplet. Strict-decode coherent throughout because no droplet ships seed TOML referencing fields whose schema droplet hasn't landed yet.

### Theme G — Post-MVP marketplace evolution (NOT in Drop 4c scope; captured for persistence)

Documented here so the design is preserved across compactions. **NONE of these land in Drop 4c.** They're post-MVP candidates.

- **G.1 — TUI marketplace browser** (Drop 4.5+ scope; FE/TUI track). Visual template list, diff against current, install one-click, history view by commit + date.
- **G.2 — Vector search.** Marketplace repo CI precomputes `<name>.embedding.json` per template (and per template tag). Tillsyn binary downloads embeddings during `fetch`. `till template search "<query>"` runs cosine-sim locally against cached embeddings. Embedding storage in marketplace repo (NOT in Tillsyn binary) keeps embeddings updateable without binary release.
- **G.3 — User contribution flow.** GitHub PR against the marketplace repo. CI runs `tillsyn template validate --strict <path>` on each PR file. Manual review for design quality; merge auto-updates `INDEX.toml`. Eventually allow signed templates / curator review.
- **G.4 — Live-runtime validation / dry-cascade simulation.** Take a synthetic action-item tree, walk dispatcher logic against the template, assert no orphans / no infinite loops / every promotion has a binding. Heavier than Theme F.5's static checks; requires dispatcher reusability for simulation mode.
- **G.5 — Template inheritance / extends.** A project template may declare `extends = "go-cascade"` and override only specific bindings. Reduces duplication for adopter projects that follow the canonical Go cascade with one or two tweaks.
- **G.6 — Template-bound agent prompts.** Today agent prompt files are global (`~/.claude/agents/*.md`). Marketplace templates may want to ship custom agent prompts inline (e.g. a `[agent_prompts.go-builder-agent]` table). Requires sandboxing semantics + adopter trust.
- **G.7 — Versioned template references on Project.** `project.template_ref = "tillsyn-templates@v1.4.0/go-cascade"` so a project pins a specific marketplace version. Update flow: `till template update` re-fetches + re-bakes if the pinned ref hasn't moved.

## Pre-MVP Rules (carried forward)

- No migration logic in Go; dev fresh-DBs.
- No closeout MD rollups.
- Opus builders.
- Filesystem-MD mode (or Tillsyn-runtime if Drops 4b's auto-promotion has matured the runtime by then).
- Single-line commits.
- NEVER raw `go test` / `mage install`.

## Open Questions To Resolve At Full-Planning Time

- **Q1 — Pre-MVP rule transition.** Drops 4a + 4b plus dogfood-prep may justify flipping some pre-MVP rules. Specifically: `feedback_no_closeout_md_pre_dogfood.md` (skip rollups) — does Drop 4c want to start writing real LEDGER / REFINEMENTS entries to dogfood the rollup loop?
- **Q2 — Theme E sizing.** How many residue items will surface from 4a + 4b? Could be 0 (clean sweep) or 5+ (significant findings). Final droplet count drifts with this.
- **Q3 — Supersede CLI scope.** Just the basic `till action_item supersede` or also `till action_item list --state failed`? Possibly bundled in same droplet.
- **Q4 — Drop 4.5 coupling.** TUI overhaul (4.5) would benefit from CLI failure listing landing first (so 4.5 has the data layer to render). Soft sequencing — does Drop 4c block Drop 4.5, or run in parallel?
- **Q5 — Drop 5 readiness gate.** Drop 5 is "dogfood validation" — at what point during Drop 4c does Drop 5 become startable? After Theme A + B (silent-data-loss + escape hatches) but before C/D? Or wait for the full 4c close?

## Approximate Size

~38–50 droplets total (Themes A ~4 + B ~2 + C ~3 + D ~1–2 + F.1–F.6 ~15–18 + F.7 spawn redesign ~13–18; Theme E populated post-4a/4b). Theme F.7 grew once F.7.17 (CLI adapter seam) + F.7.18 (context aggregator) added 2026-05-04 to architect for multi-CLI extensibility + declarative flexible context delivery. Plan-QA falsification 2026-05-04 added the schema-bundle droplet (must land at the start of F.7 wave) and the security tightening (list-form `command`, closed env baseline, total-cap + wall-clock aggregator caps). **Drop 4d (NEW, post-Drop-4c-merge, after Drop 5 claude-only dogfood):** ~7–10 droplets adding `codex` adapter on top of the F.7.17 seam. Sequencing locked: Drop 4c → Drop 5 (claude-only dogfood, validates cascade-on-itself loop without conflating second-CLI integration risk) → Drop 4d (codex) → Drop 5.5/6 (multi-CLI dogfood validation).

## Hard Prerequisites

- Drop 4a closes (dispatcher exists; can the Drop 4c items be dispatched-built or do they require manual orch builds? Decision deferred to planning time).
- Drop 4b closes (gate execution + commit pipeline exists; Drop 4c items benefit from the pipeline once they're dispatcher-eligible).
- `project_drop_4a_refinements_raised.md` exists and is populated with mid-build findings.
- `project_drop_4b_refinements_raised.md` (TBD) exists and is populated.

## Workflow Cross-References

- `workflow/drop_4a/PLAN.md` — Drop 4a's plan (deferrals from 4a end up here).
- `workflow/drop_4b/SKETCH.md` — Drop 4b's sketch (deferrals from 4b end up here).
- `project_drop_3_refinements_raised.md` — R1, R2, R3, R5 source.
- PLAN.md §19.1 — original Drop 1 audit-list source for items 1–7.
- Memory `feedback_no_closeout_md_pre_dogfood.md` — pre-MVP rules that may transition during this drop.

## Open Tasks Before Full Planning

1. Drop 4a closes; Drop 4b closes (in sequence).
2. Drop 4a + 4b refinements memories reviewed; Theme E populated with everything that didn't make it into the 4a/4b drops.
3. Drop 5 readiness gate decision: does Drop 5 start mid-Drop-4c (after Theme A + B), or wait for full Drop 4c close?
4. Full REVISION_BRIEF authored, parallel-planner dispatch (likely 4 theme planners, one per Theme A–D), unified PLAN.md synthesis, plan-QA twins → green → builder dispatch.

## Anti-Goals

- **Not a "fix everything" drop.** Drop 4c is bounded by the MVP-feature-complete gate. Items that aren't blocking Drop 5 dogfood readiness stay deferred to a later refinement.
- **Not a refactor drop.** Each item is a narrow fix on top of existing primitives. No package reorganizations, no API rewrites.
- **Not a TUI work.** TUI overhaul is Drop 4.5, runs concurrent with Drop 5.
