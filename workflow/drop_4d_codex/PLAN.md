# DROP_4D_CODEX — CODEX CLIADAPTER + PLANNER ATOMICITY PROMPTS + EXAMPLE AGENTS.TOML

**State:** planning
**Blocked by:** —
**Paths (expected):** `internal/app/dispatcher/cli_codex/**`, `internal/app/dispatcher/cli_adapter.go`, `internal/templates/builtin/agents/<group>/<role>.md`, `internal/templates/builtin/projects/*/agents.toml`, project CLAUDE.md
**Packages (expected):** `github.com/evanmschultz/tillsyn/internal/app/dispatcher`, `github.com/evanmschultz/tillsyn/internal/app/dispatcher/cli_codex`, `github.com/evanmschultz/tillsyn/internal/app/dispatcher/cli_codex/register`, `github.com/evanmschultz/tillsyn/internal/templates`
**PLAN.md ref:** `main/PLAN.md` → (no row yet; this is the first multi-backend drop)
**Workflow:** `main/workflow/example/drops/WORKFLOW.md`
**Cascade concept:** `main/AGENT_CASCADE_DESIGN.md`
**Started:** 2026-05-20
**Closed:** —

## Scope

Phase 1 of multi-backend dogfood (per `project_multi_backend_dogfood_direction.md` memory ratified by dev 2026-05-19). Lands the codex CLIAdapter inside tillsyn's existing dispatcher framework so that template-driven agent bindings can route `plan` + `plan-qa-falsification` + `build-qa-falsification` + `research` kinds to codex (gpt-5.x with reasoning-effort knobs) while keeping `*-qa-proof` on Claude opus and `build` + `commit` on Claude haiku. Also lands substantive embedded prompt content (planner atomicity rule: ≤4 small blocks per build droplet) and an example multi-backend `agents.toml` baked into the project template. Bootstraps under Anthropic-only spend; after this drop merges, every subsequent drop benefits from codex routing (~60-70% Anthropic spend drop expected).

**Out of scope** (deferred to drop_4e_ollama after a small verification spike): `cli_claude_bare` adapter for ollama-localhost routing via `ANTHROPIC_BASE_URL` redirect. Context7 surfaced that Ollama exposes OpenAI-compatible APIs (not Anthropic-compatible), so ta's documented `--bare` pattern needs hands-on verification before drop scope commit.

## Planner

Planned 2026-05-20 against Hylla snapshot 2 (`github.com/evanmschultz/tillsyn@main` commit `752cb94`).

Evidence base:
- `CLIAdapter` interface source verified via `hylla_node_full`: 3 methods (`BuildCommand`, `ParseStreamEvent`, `ExtractTerminalReport`), depends on `BindingResolved`, `BundlePaths`, `StreamEvent`, `TerminalReport`
- `RegisterAdapter(kind CLIKind, adapter CLIAdapter)` verified via Hylla in `internal/app/dispatcher/spawn.go`
- `cli_claude/init.go` read directly: init pattern confirmed (`RegisterAdapter(CLIKindClaude, New())` + blank import of `cli_claude/render`)
- `cli_claude/stream.go` read directly: parseStreamEvent + extractTerminalReport pattern confirmed
- No existing `cli_codex` package: zero Hylla hits for "codex" in Go code
- `IsValidCLIKind` in `cli_adapter.go` only enumerates `CLIKindClaude` today; must be extended in D2
- Embedded agent prompt stubs in `internal/templates/builtin/agents/gen/` confirmed: all are Drop 4c.6 W1.D1 placeholders
- `agents.example.toml` schema confirmed: runtime-config layer uses `client` field (NOT `cli_kind`)
- `AgentBinding.CLIKind` via Hylla `hylla_node_full`: template layer uses `toml:"cli_kind"`

### Droplet 4d.1 — Add `CLIKindCodex` const + extend `IsValidCLIKind`

**State:** todo
**Role:** builder
**Irreducible:** true
**Paths:** `internal/app/dispatcher/cli_adapter.go`
**Packages:** `github.com/evanmschultz/tillsyn/internal/app/dispatcher`
**Blocked by:** —

Add `CLIKindCodex CLIKind = "codex"` const with a doc-comment that mirrors the existing `CLIKindClaude` pattern ("the CLI kind for the `codex` headless CLI. Drop 4c ships `CLIKindClaude` only; Drop 4d adds this."). Extend `IsValidCLIKind` to return true for `CLIKindCodex`. Add a test case to `cli_adapter_test.go` asserting `IsValidCLIKind(CLIKindCodex)` returns true. The const is ~2 LOC; the switch extension is 1 line; the test is 1 table row.

Acceptance:
- `CLIKindCodex` const declared with value `"codex"` and a doc-comment referencing Drop 4d
- `IsValidCLIKind(CLIKindCodex)` returns `true`
- `IsValidCLIKind("")` still returns `false` (existing invariant preserved)
- `mage test-pkg github.com/evanmschultz/tillsyn/internal/app/dispatcher` passes with no new failures

---

### Droplet 4d.2 — `cli_codex` adapter package: adapter + argv + env + stream

**State:** todo
**Role:** builder
**Irreducible:** true
**Paths:** `internal/app/dispatcher/cli_codex/adapter.go`, `internal/app/dispatcher/cli_codex/argv.go`, `internal/app/dispatcher/cli_codex/env.go`, `internal/app/dispatcher/cli_codex/stream.go`
**Packages:** `github.com/evanmschultz/tillsyn/internal/app/dispatcher/cli_codex`
**Blocked by:** 4d.1

Implement `dispatcher.CLIAdapter` for `codex exec --json`. Mirror the `cli_claude` package shape:

- `adapter.go`: unexported `codexAdapter{}` struct, compile-time `var _ dispatcher.CLIAdapter = (*codexAdapter)(nil)` assertion, `New() dispatcher.CLIAdapter` constructor. `BuildCommand` calls `assembleArgv` + `assembleEnv`, constructs `exec.CommandContext(ctx, "codex", argv[1:]...)` with explicit `cmd.Env`.
- `argv.go`: `assembleArgv(binding dispatcher.BindingResolved, paths dispatcher.BundlePaths) []string`. Codex argv shape: `codex exec --ephemeral --skip-git-repo-check --output-format json -m <model>` plus `-c model_reasoning_effort=<effort>` when `binding.Effort != nil`. The prompt/task content is passed via the positional arg or `-i` stdin — planner notes this is an **open question** (see Notes below): the builder MUST verify the exact codex exec invocation against the codex CLI docs or `codex --help` output before finalizing argv. The `BundlePaths.SystemPromptPath` may not be used by codex the same way as claude's `--system-prompt` flag.
- `env.go`: `assembleEnv(binding dispatcher.BindingResolved) ([]string, error)`. Mirror `cli_claude/env.go` pattern: construct the closed POSIX baseline + resolve each name in `binding.Env` via `os.Getenv`, fail loud on missing required names. Codex uses `OPENAI_API_KEY` rather than `ANTHROPIC_API_KEY` as its auth env var — builder adds `OPENAI_API_KEY` to the closed POSIX baseline.
- `stream.go`: `ParseStreamEvent(line []byte) (dispatcher.StreamEvent, error)` and `ExtractTerminalReport(ev dispatcher.StreamEvent) (dispatcher.TerminalReport, bool)`. Codex JSONL event shape is **unknown until D4 fixture is captured** — builder implements a best-effort normalization based on available codex documentation, with clear TODO comments where fixture verification is needed. The builder MUST NOT block D4's fixture on their implementation; best-effort now, fixture-verified in D5. A `codexEventDiscriminator` + `codexResultEvent` pattern mirrors cli_claude/stream.go.

Note: `cli_codex` does NOT need a `render/` subpackage for Drop 4d. The render hook (`BundleRenderFunc`) is registered only by `cli_claude/render`. If codex agents need a custom system-prompt render path, that is deferred to a follow-on drop after the codex invocation model is fully understood. The `cli_codex/init.go` (D4d.3) does NOT blank-import any render package.

Acceptance:
- Package `github.com/evanmschultz/tillsyn/internal/app/dispatcher/cli_codex` compiles cleanly
- `New()` returns a value satisfying `dispatcher.CLIAdapter`
- `BuildCommand` returns a non-nil `*exec.Cmd` with `cmd.Env` set explicitly (does not inherit `os.Environ()`)
- `BuildCommand` argv contains `"codex"` as the program, `--output-format`, `json`, `-m`, `<model>` flags
- When `binding.Effort != nil`, argv contains `-c` `model_reasoning_effort=<val>`
- `ParseStreamEvent` returns a non-zero `StreamEvent` with the raw bytes preserved in `Raw`
- `ExtractTerminalReport` returns `(report, true)` for terminal events and `(zero, false)` for non-terminal events
- `mage test-pkg github.com/evanmschultz/tillsyn/internal/app/dispatcher/cli_codex` passes (runs what tests exist; D5 adds the fixture-backed table-driven tests)

---

### Droplet 4d.3 — `cli_codex/register` package: init-time wiring

**State:** todo
**Role:** builder
**Irreducible:** true
**Paths:** `internal/app/dispatcher/cli_codex/register/register.go`
**Packages:** `github.com/evanmschultz/tillsyn/internal/app/dispatcher/cli_codex/register`
**Blocked by:** 4d.1, 4d.2

Create `internal/app/dispatcher/cli_codex/register/register.go`. Mirror `cli_claude/init.go` exactly:

```go
package register

import (
    "github.com/evanmschultz/tillsyn/internal/app/dispatcher"
    "github.com/evanmschultz/tillsyn/internal/app/dispatcher/cli_codex"
)

func init() {
    dispatcher.RegisterAdapter(dispatcher.CLIKindCodex, cli_codex.New())
}
```

Note: the package is named `register` (not `cli_codex`) to mirror the existing `cli_claude` init location. The `cli_claude` adapter currently has its `init()` in its own package (`package cli_claude` in `init.go`) — NOT in a separate `register` package. The builder MUST verify the actual pattern: read `cli_claude/init.go` (the file is already at `internal/app/dispatcher/cli_claude/init.go`, NOT in a `register/` subdir). Decision: for `cli_codex`, place the `init()` in `internal/app/dispatcher/cli_codex/register/register.go` to avoid cluttering the adapter package itself, OR place it in `internal/app/dispatcher/cli_codex/init.go` (same package as adapter). REVISION_BRIEF says "D3 — `internal/app/dispatcher/cli_codex/register/register.go`" — use that path. The builder also adds the blank import to `cmd/till/main.go` (or its init file) so the register package is side-effect-imported in the production binary. Builder must locate where `_ "github.com/evanmschultz/tillsyn/internal/app/dispatcher/cli_claude"` is imported in `cmd/till` and add the parallel `cli_codex/register` import.

Acceptance:
- `internal/app/dispatcher/cli_codex/register/register.go` exists with package `register`
- `init()` calls `dispatcher.RegisterAdapter(dispatcher.CLIKindCodex, cli_codex.New())`
- The production binary imports the register package via blank import (new import added to cmd/till or its designated init wiring file)
- `mage test-pkg github.com/evanmschultz/tillsyn/internal/app/dispatcher/cli_codex/register` passes

---

### Droplet 4d.4 — Fixture capture (HARD PREREQ — dev action, not a builder droplet)

**State:** blocked
**Role:** dev-action
**Paths:** `internal/app/dispatcher/cli_codex/testdata/codex_stream_minimal.jsonl`
**Packages:** — (no Go code)
**Blocked by:** — (external: dev must run `codex exec --json` on dev machine)

**NOT a builder droplet.** The dev runs `codex exec --json` on a trivial prompt and captures the JSONL output into `internal/app/dispatcher/cli_codex/testdata/codex_stream_minimal.jsonl`. Without this fixture, Droplet 4d.5 cannot proceed.

Dev action required:
1. Ensure `codex` CLI is installed and authed with a ChatGPT-Plus / API key on dev machine
2. Run: `codex exec --ephemeral --output-format json --model <model> <<< "Say: ok"` (or equivalent) and capture the JSONL output
3. Save to `internal/app/dispatcher/cli_codex/testdata/codex_stream_minimal.jsonl`
4. Post the file content as a Tillsyn comment on action item `ee5f16f8-931e-4730-bc7f-a03b1d506804` OR commit directly to the drop branch

Acceptance (for Droplet 4d.5 unblock):
- The fixture file exists at the declared path
- The file contains at least one complete codex JSON stream sequence (init event + assistant event + result/terminal event)

---

### Droplet 4d.5 — `cli_codex` table-driven tests against fixture

**State:** blocked (pending 4d.4)
**Role:** builder
**Irreducible:** true
**Paths:** `internal/app/dispatcher/cli_codex/adapter_test.go`
**Packages:** `github.com/evanmschultz/tillsyn/internal/app/dispatcher/cli_codex`
**Blocked by:** 4d.2, 4d.4

Table-driven tests for the `cli_codex` adapter, mirroring `cli_claude/adapter_test.go`. Required test functions (new, not yet in tree):

- `TestBuildCommandArgvShapeMinimal` — minimal `BindingResolved` (no Model, no Effort); verify binary name is `"codex"`, required flags present
- `TestBuildCommandArgvShapeWithEffort` — `BindingResolved.Effort = ptr("medium")`; verify `-c model_reasoning_effort=medium` in argv
- `TestBuildCommandHardcodedBinary` — verify `cmd.Path` or argv[0] is `"codex"` (mirrors `TestBuildCommandHardcodedBinary` in cli_claude)
- `TestEnvNotInheritedFromOSEnviron` — verify `cmd.Env` is set explicitly, does not contain the full `os.Environ()` noise (mirrors `TestEnvNotInheritedFromOSEnviron` in cli_claude)
- `TestParseStreamEventFromFixture` — read `testdata/codex_stream_minimal.jsonl`, parse each line via `ParseStreamEvent`, verify expected canonical types
- `TestExtractTerminalReportFromFixture` — drive `ExtractTerminalReport` on the fixture's terminal event, verify `(report, true)` with non-nil fields

Acceptance:
- All above test functions exist in `cli_codex/adapter_test.go`
- `mage test-pkg github.com/evanmschultz/tillsyn/internal/app/dispatcher/cli_codex` passes green
- Tests use `testdata/codex_stream_minimal.jsonl` (D4 fixture) for fixture-backed cases

---

### Droplet 4d.6 — Substantive embedded agent prompts (gen group)

**State:** todo
**Role:** builder
**Irreducible:** true
**Paths:** `internal/templates/builtin/agents/gen/planning-agent.md`, `internal/templates/builtin/agents/gen/plan-qa-falsification-agent.md`, `internal/templates/builtin/agents/gen/plan-qa-proof-agent.md`, `internal/templates/builtin/agents/gen/build-qa-falsification-agent.md`, `internal/templates/builtin/agents/gen/build-qa-proof-agent.md`, `internal/templates/builtin/agents/gen/builder-agent.md`, `internal/templates/builtin/agents/gen/commit-message-agent.md`
**Packages:** `github.com/evanmschultz/tillsyn/internal/templates` (embed — no Go code changes, but the embed.go directive covers these files)
**Blocked by:** —

Replace Drop 4c.6 placeholder stubs with substantive embedded prompt content. Each file keeps its TOML frontmatter (`name:`, `description:`, optional `hooks:`) and replaces the `# PLACEHOLDER` body.

Per `EMBED-PROMPTS-R1` refinement + `project_multi_backend_dogfood_direction.md` atomicity thesis:

**planning-agent.md**: The planner atomicity rule MUST be the core invariant. Required content: (1) role statement — planner decomposes work into atomic build droplets; (2) atomicity rule — "Every `build` droplet touches ≤4 small code blocks (including tests). If a planned build droplet is larger, decompose further into sibling builds with explicit `blocked_by` between them"; (3) paths/packages declaration rule — "Every `build` droplet MUST declare `paths []string` and `packages []string`. No empty declarations"; (4) brief on using Tillsyn MCP tools (`till.action_item`, `till.comment`); (5) Section 0 reasoning directive.

**plan-qa-proof-agent.md** and **plan-qa-falsification-agent.md**: Evidence-verification vs adversarial-counterexample roles respectively. QA-falsification MUST attack: missing `blocked_by` between droplets sharing a file or package, droplets exceeding 4-code-block atomicity, missing `paths`/`packages` declarations, cycles in `blocked_by`.

**build-qa-proof-agent.md** and **build-qa-falsification-agent.md**: Build-level QA passes. Read-only. Falsification attacks: does the implementation match acceptance criteria, are there untested paths, do all `mage` targets pass.

**builder-agent.md**: Existing frontmatter with `hooks:` is load-bearing — keep it. Body: implements exactly the declared `paths`; runs `mage test-pkg <pkg>` after changes; sets `metadata.outcome=success` on completion; does NOT call `move_state=complete` (monitor owns that transition).

**commit-message-agent.md**: Minimal — reads the git diff from `context/git_diff.patch`, emits a single conventional-commit subject line (≤72 chars, no body), returns immediately.

These are MARKDOWN files only. No Go code changes. The embed.go directive already covers `builtin/agents/gen/*.md`.

Acceptance:
- All 7 files no longer contain `# PLACEHOLDER` body text
- `planning-agent.md` body contains the atomicity rule and paths/packages rule verbatim
- `plan-qa-falsification-agent.md` body attacks missing `blocked_by` and over-sized droplets
- `builder-agent.md` body mentions `mage test-pkg` and `metadata.outcome=success`
- `mage test-pkg github.com/evanmschultz/tillsyn/internal/templates` passes (embed load + validateAgentBindingNames still resolves all gen/ agent names)
- Files load cleanly under `templates.Load` with no new errors

---

### Droplet 4d.7 — Multi-backend routing example in `agents.example.toml`

**State:** todo
**Role:** builder
**Irreducible:** true
**Paths:** `internal/templates/builtin/agents.example.toml`
**Packages:** `github.com/evanmschultz/tillsyn/internal/config` (LoadMultiGroupRegistry validates this file in tests)
**Blocked by:** —

Add or update per-kind overrides in the `[gen]` group within `agents.example.toml` to demonstrate the multi-backend routing thesis. The routing thesis from `project_multi_backend_dogfood_direction.md`:

- `plan` → codex, effort=medium
- `plan-qa-falsification` → codex, effort=xhigh
- `plan-qa-proof` → claude, model=opus
- `research` → codex, effort=high
- `build` → claude, model=haiku
- `build-qa-falsification` → codex, effort=xhigh
- `build-qa-proof` → claude, model=opus
- `commit` → claude, model=haiku (already in `agents.example.toml`)

The `agents.example.toml` uses `client` as the field name for the runtime-config layer (not `cli_kind`). However, this is an **open question for dev** (see Notes): the strict TOML decoder in `LoadMultiGroupRegistry` rejects unknown fields. The builder MUST verify what field name the `config.Preset` / `config.Override` struct uses for the CLI-family selection before writing the file. If the struct does not yet have a `client` field (it's in `agents.example.toml` today as a comment/string but may not be decoded into a struct field), this droplet may need to add the field to `internal/config/agents.go` OR the routing-per-kind demonstration lives in a comment. The builder checks `internal/config/agents.go` struct definitions before proceeding.

RiskNote: The `agents.example.toml` `client` field may NOT map to a `config.Override.Client` struct field yet — the config schema landing in Drop 4c.6.1 W0 may not have wired up `cli_kind`/`client` routing in the config layer. If it doesn't, the builder adds a `# TODO(drop_4d): per-kind cli_kind/client routing pending config schema update` comment block and documents the per-kind routing intent as comments until the wiring lands.

Acceptance:
- `agents.example.toml` contains a `[gen]` section with per-kind overrides that demonstrate codex routing for plan + QA-falsification + research kinds
- The file parses cleanly under `mage test-pkg github.com/evanmschultz/tillsyn/internal/config` (no strict-decode rejection)
- A comment block explains the multi-backend routing intent for each kind override
- If `client`/`cli_kind` field is not yet in the config schema, the builder documents what's needed as a comment and routes the gap as a finding in `BUILDER_WORKLOG.md`

---

### Drop-end Orchestrator Action — D8: CLAUDE.md + memory updates

**State:** todo
**Role:** orchestrator-direct (not a builder droplet)
**Paths:** `CLAUDE.md` (project root), `~/.claude/projects/…/memory/feedback_cascade_model_policy.md`, `~/.claude/projects/…/memory/project_multi_backend_dogfood_direction.md`

After D1-D7 complete and `mage ci` is green, the orchestrator updates:
1. Project `CLAUDE.md` "Agent Bindings" table: add `cli_kind` column showing codex vs claude routing per kind
2. Memory file `feedback_cascade_model_policy.md`: update to reflect that codex routes plan/QA-falsification/research post-Drop-4d
3. Memory file `project_multi_backend_dogfood_direction.md`: update Phase-1-shipped-state section to note Drop 4d merged

This is orch-direct markdown work per feedback_orchestrator_no_build.md and feedback_md_update_qa.md — no builder subagent needed.

## Notes

### Open Questions For Dev (Required Before Builders Proceed On Affected Droplets)

**OQ1 — Codex CLI invocation shape (blocks D2 argv design):**
What is the exact `codex exec --json` argv for a production dispatch? Specifically:
- Does codex accept the agent prompt via `--system` flag, `-i <file>`, positional arg, or stdin?
- Is `--ephemeral` the right flag for stateless sessions in the codex CLI?
- Is `--skip-git-repo-check` a valid codex flag?
- What is the exact format for passing `model_reasoning_effort` — is it `-c model_reasoning_effort=<val>` or `--effort <val>` or something else?

**OQ2 — Codex model identifier (blocks D7 per-kind model values):**
`gpt-5.x` is a placeholder in REVISION_BRIEF. What is the exact model string for the dev's ChatGPT-tier auth (e.g. `gpt-4o`, `o3`, `o4-mini`)?

**OQ3 — `cli_codex/render/` subpackage (blocks D2/D3 design decision):**
Does codex need a `render/` subpackage (like `cli_claude/render/`) for system-prompt materialization? The render package registers `BundleRenderFunc` — codex may handle the system prompt differently (e.g. via a different flag or via stdin). Answer determines whether D3's init.go blank-imports a render subpackage.

**OQ4 — D7 target layer (blocks D7):**
Should D7 update `agents.example.toml` (runtime-config layer, `client` field) or `till-gen.toml` (template layer, `[agent_bindings]` table with `cli_kind` field)? REVISION_BRIEF says "baked-in agents.toml template" for the `gen` group — this implies the template layer. But `till-gen.toml` deliberately has NO `[agent_bindings]` section by design. Does Drop 4d add `[agent_bindings]` to `till-gen.toml`, or does it create a new template file, or does it update `agents.example.toml`?

**OQ5 — `config.Preset` / `config.Override` struct has `client` field?:**
Does `internal/config/agents.go` currently have a `client` (or `cli_kind`) field on the `Preset`/`Override` structs? `agents.example.toml` shows `client = "claude"` at group level, which means `LoadMultiGroupRegistry` (strict decoder) must have a struct field for it. The builder for D7 must check before writing per-kind overrides with `client = "codex"`.

### Parallelization Summary

- D1, D6, D7 run in parallel (disjoint packages: `dispatcher`, `templates`/embed, `config`)
- D2 runs immediately (same as D1 but wait — D2 is in `dispatcher` package; D1 is in `cli_codex` package which is a DIFFERENT package. D2 touches `dispatcher` package, D1 touches `cli_codex` package — they are disjoint. D1 and D2 can run in parallel.)
- D3 runs after D1 + D2 (imports both)
- D4 is a dev action that runs any time
- D5 runs after D2 + D4 (needs CLIKindCodex const + fixture)
