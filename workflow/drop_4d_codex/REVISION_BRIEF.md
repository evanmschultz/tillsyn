# DROP_4D_CODEX — Revision Brief (Planner Input)

This brief is the planner's input. It captures the agreed scope, the architectural target, and the hard prerequisites. The planner decomposes from here into droplets in `PLAN.md`.

**Memory pointer (load-bearing, permanent):** `~/.claude/projects/-Users-evanschultz-Documents-Code-hylla-tillsyn/memory/project_multi_backend_dogfood_direction.md`. The planner MUST read that memory before decomposition — it captures the full Hylla-verified picture of what tillsyn already ships, the routing thesis, and the security stance.

**Project CLAUDE.md `## Hard Rules`** also apply: no time estimates → cascade-shape work; mage-only Go gates; no bash dispatcher bridges; no arbitrary-argv knobs; atomicity is a planner-prompt concern; multi-backend dogfood is the cost-relief mechanism.

## Goal

Land the codex CLIAdapter inside tillsyn's existing dispatcher framework + substantive embedded prompt content + an example multi-backend `agents.toml`. After merge, the dispatcher can route per-kind to either claude or codex based on `BindingResolved.CLIKind` set in the template's `agent_bindings`.

## What's Already Shipped (Do Not Reinvent)

Verified 2026-05-19 via Hylla (`mcp__hylla__hylla_search_keyword` against `github.com/evanmschultz/tillsyn@main`):

- `internal/app/dispatcher/cli_adapter.go` — `CLIAdapter` interface, 3 methods total (locked decision L10). Docstring explicitly anticipates "(claude, codex, …)" as implementations.
- `internal/app/dispatcher/spawn.go` — `RegisterAdapter(kind CLIKind, adapter CLIAdapter)` registry pattern, init-time wiring.
- `internal/app/dispatcher/cli_claude/adapter.go` — reference implementation. Build exec.Cmd, decode JSONL StreamEvent, extract TerminalReport.
- `internal/app/dispatcher/cli_claude/render/` — rendering package that handles system-prompt.md materialization + permission-grants merge. `cli_codex` package may or may not need a render subpackage — planner decides based on whether codex needs CLI-specific bundle subdirs.
- `BindingResolved` struct: `AgentName`, `CLIKind`, `Tools[]`, `ToolsAllowed[]`, `ToolsDisallowed[]`, `Env[]`, `Model *string`, **`Effort *string` (for codex `model_reasoning_effort`)**, `MaxBudgetUSD`, `MaxTurns`, `SystemPromptTemplatePath`, `Sandbox.Filesystem`, `Sandbox.Network`. REV-1 supersession explicitly removed `Command []string` + `ArgsPrefix []string`.
- `ResolveBinding` (pure function) — CLI > MCP > TUI > template-TOML > absent priority cascade. Slice-field override plumbing not yet wired (known limitation; not a blocker for codex).
- `LoadMultiGroupRegistry` at `internal/config/agents.go` — strict TOML decode of multi-group `agents.toml`. `agents.toml` is real, shipped, validated.
- `templates.Load` at `internal/templates/load.go` — 14 validators including `validateAgentBindingToolGating`, `validateAgentBindingEnvNames`, `validateAgentBindingNames` (HARD-FAIL on unresolved agent file), `validateRequiredChildRules` (QA twins required on every plan + build), `validateChildRuleCycles`, `validateChildRuleRecursionDepth` (max depth 5).
- `PermissionHandshake` — closed-loop permission flow. spawn `permission_denials` → TUI attention items → durable grants → next spawn settings.json. Codex adapter inherits this.
- `CommitAgent.GenerateMessage` — end-to-end production flow via dispatcher (haiku). Proof that the full spawn pipeline works on at least one real path today.

## Architectural Target (Droplet Skeleton)

The planner expands these into per-droplet rows in `PLAN.md` with `paths`, `packages`, `acceptance`, and `blocked_by`:

- **D1 — `cli_codex/adapter.go`**: implement `CLIAdapter` for `codex exec --json`. Build `*exec.Cmd` with `codex exec --ephemeral --skip-git-repo-check -m <model> -c model_reasoning_effort=<effort> -C <cwd>`. Map `BindingResolved.Effort` to the `-c` flag. Decode codex JSONL stream events into canonical `dispatcher.StreamEvent` (preserve raw JSON per L10). Extract `TerminalReport` from the codex end-of-stream marker.
- **D2 — `cli_adapter.go`**: add `CLIKindCodex CLIKind = "codex"` const. One-line change.
- **D3 — `cli_codex/register/register.go`**: `init()` → `dispatcher.RegisterAdapter(dispatcher.CLIKindCodex, cli_codex.New())`. Mirrors `cli_claude/register/` pattern (or whatever lives there today — planner verifies).
- **D4 — fixture capture (HARD PREREQ, not a code droplet)**: dev captures `internal/app/dispatcher/cli_codex/testdata/codex_stream_minimal.jsonl` from a real `codex exec --json` run on dev's machine. Without this fixture, D5 cannot proceed. Dev pastes the file content into a Tillsyn comment on the root action item OR commits it directly.
- **D5 — `cli_codex_test.go`**: table-driven tests against the fixture. Verify `BuildCommand`, `DecodeStreamEvent`, `ExtractTerminalReport` all produce expected canonical shapes. Use the same test patterns as `cli_claude_test.go`.
- **D6 — substantive embedded prompts**: `internal/templates/builtin/agents/<group>/<role>.md` for `planning-agent`, `qa-falsification-agent` (build + plan variants), `qa-proof-agent` (build + plan variants), `builder-agent`, `commit-message-agent`. **Planner prompt MUST enforce "≤4 small code blocks per build droplet, declare paths + packages on every build."** This droplet may itself decompose into sub-droplets per agent file. Cross-reference: `EMBED-PROMPTS-R1` refinement.
- **D7 — example `agents.toml` for the `gen` group**: routes per the codex-only subset of the routing thesis:
  - `plan` → `cli_kind = "codex"`, `effort = "medium"`, `model = "gpt-5.x"` (planner verifies the exact codex model identifier today)
  - `plan-qa-falsification` → `cli_kind = "codex"`, `effort = "xhigh"`
  - `plan-qa-proof` → `cli_kind = "claude"`, `model = "opus"`
  - `research` → `cli_kind = "codex"`, `effort = "high"`
  - `build` → `cli_kind = "claude"`, `model = "haiku"`
  - `build-qa-falsification` → `cli_kind = "codex"`, `effort = "xhigh"`
  - `build-qa-proof` → `cli_kind = "claude"`, `model = "opus"`
  - `commit` → `cli_kind = "claude"`, `model = "haiku"`
- **D8 — CLAUDE.md + memory updates**: update project CLAUDE.md "Agent Bindings" table with `cli_kind` column; update memory `feedback_cascade_model_policy.md` to reflect codex routing; update `project_multi_backend_dogfood_direction.md` Phase-1-shipped-state section.

## Hard Prerequisites

- **D4 codex JSONL fixture**: dev captures from real `codex exec --json` run before D5 builds.
- **Codex CLI authed**: dev confirms `codex exec --json -m <model> <<< "Say ok"` returns clean output. Without this, every codex-routed spawn fails.

## Open Questions For Planner Round 1

- Should `cli_codex/render/` be created (mirroring `cli_claude/render/`) or does codex need only the adapter without render? Planner decides based on whether codex needs CLI-specific bundle subdirs.
- Exact codex model identifier (`gpt-5.x` is a placeholder — what's the actual value for the dev's ChatGPT-tier auth)? Planner asks dev before D7.
- Codex `--json` stream event shape — what event families does it emit (assistant_text, tool_use, tool_result, etc.)? Planner reviews the fixture from D4 to design the canonical event normalization in D1.
- Does `permission_denials` translate cleanly from codex (does codex emit denial events the same way claude does)? If not, `PermissionHandshake` may need a per-adapter denial-extraction hook — but that's a follow-on, not a blocker for D1-D7.

## What's Explicitly Out Of Scope

- Ollama routing (deferred to `drop_4e_ollama` after spike).
- Slice-field override plumbing on `ResolveBinding` (known limitation; template-level allowlists work).
- Renaming `~/.claude/agents/go-*.md` to bare names (legacy bridge stays through the dogfood arc).
- `cli_claude_bare` adapter (Phase 2 only).
- New cascade kinds or template features (drop_4d_codex is dispatcher + prompts only).

## Success Criteria

- `mage ci` green on the drop branch including all new codex tests.
- `agents.toml` validates at template Load (passes all 14 validators).
- A smoke-test spawn of a `plan` action item with `cli_kind = "codex"` end-to-end produces a non-empty terminal report.
- Memory + CLAUDE.md updates reflect Phase-1-shipped state.

## Phase 1 Output Expected

Filled `PLAN.md` Planner section with droplet rows (or sub-drop rows if the planner decomposes recursively), each with explicit `paths`, `packages`, `acceptance`, and `blocked_by`. Then plan-QA pair runs (proof + falsification, parallel), then plan revises if needed, then Phase 4 build begins.
