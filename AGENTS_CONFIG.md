# `agents.toml` Configuration Reference

This document is the adopter-facing reference for `agents.toml` and its per-machine companion `agents.local.toml`. It is the single source for the question "how do I configure my agents per-machine for this Tillsyn project?" — model choice, tool allowlists, environment-variable handling, frontmatter strip behavior, and content-injection extension points all live here.

The schema described below is implemented in `internal/config/agents.go`. Symbol names, sentinel-error names, and field semantics in this document mirror the shipped Go API verbatim. Cross-reference targets:

- `CASCADE_METHODOLOGY.md` — methodology spine; explains the cascade kinds (`plan` / `build` / QA pairs / etc.) that `agents.toml` configures per-kind.
- `SPAWN_PIPELINE.md` — the spawn pipeline that consumes the resolved per-kind config to construct each per-spawn bundle.
- `CLI_ADAPTER_AUTHORING.md` — for adapter authors who need to know how `tools_allow` / `env_set` / `env_from_shell` flow into `BuildCommand`.
- `WIKI.md` § "Cascade Vocabulary" — canonical for the `kind` enum that keys `[agents.<kind>]` blocks.

The split of responsibility between Tillsyn and the templates that drive it is load-bearing throughout this doc. Tillsyn **enforces** what the schema and templates declare; templates and `agents.toml` **define** the per-project semantics. `agents.toml` itself is a project-author artifact, not Tillsyn-internal — Tillsyn validates it at load time and rejects invalid shapes loudly, but never silently substitutes defaults for missing required fields. See `feedback_tillsyn_enforces_templates.md` for the structural-vs-semantic split this rule descends from.

## Table of Contents

1. [File Locations and Resolution](#1-file-locations-and-resolution)
2. [Schema — `[agents]` Defaults Block](#2-schema--agents-defaults-block)
3. [Schema — `[agents.<kind>]` Per-Kind Override Blocks](#3-schema--agentskind-per-kind-override-blocks)
4. [Override Semantics — Project + Local Two-Layer Merge](#4-override-semantics--project--local-two-layer-merge)
5. [`env_set` vs `env_from_shell`](#5-env_set-vs-env_from_shell)
6. [`tools_allow` vs `tools_deny` Override Scope](#6-tools_allow-vs-tools_deny-override-scope)
7. [Frontmatter Strip Behavior](#7-frontmatter-strip-behavior)
8. [`claude_md_addons` — Content-Injection Extension Point](#8-claude_md_addons--content-injection-extension-point)
9. [Worked Examples — Anthropic / Bedrock / Vertex / OpenRouter / Ollama Cloud](#9-worked-examples--anthropic--bedrock--vertex--openrouter--ollama-cloud)
10. [Error Handling — `*ConfigError` Envelope](#10-error-handling--configerror-envelope)
11. [Validation Rules and Failure Modes](#11-validation-rules-and-failure-modes)
12. [Implementation Notes](#12-implementation-notes)

---

## 1. File Locations and Resolution

`agents.toml` lives at the **project root** (the same directory as `.tillsyn/`, the project's `agents/` directory, and the `.gitignore` that excludes per-machine secrets). It is **required** for any project that uses cascade dispatch — Tillsyn fails loud at startup if it is missing.

`agents.local.toml` lives **next to `agents.toml`**, also at the project root, and is **optional**. The `.gitignore` shipped by `till init` adds `agents.local.toml` automatically — its purpose is per-machine override of fields that vary across contributors (model endpoints, API keys via shell-env rename, locally available tools).

When a Tillsyn process starts, configuration is resolved in two layers:

1. **Load `agents.toml`** via `LoadRegistry(path)` (defined in `internal/config/agents.go`). The decoder is strict: unknown top-level fields are rejected via `DisallowUnknownFields()`, so typos in field names fail loud rather than silently drop. The result is an `*AgentsRegistry` with a populated `Preset` (from `[agents]`) and an `Overrides map[domain.Kind]Override` keyed by the closed 12-value `kind` enum.
2. **If `agents.local.toml` exists, deep-merge** via `MergeLocal(project, local)`. Local non-zero fields win at the Preset layer; local non-nil pointers win field-by-field at the per-kind Override layer. The merge is rejected up-front if local sets `tools_deny` (see § 6).

After the merge, per-kind config is resolved on demand via `Resolve(registry, kind)` which returns an `AgentRuntime` — the effective concrete config for one cascade kind. The dispatcher calls `Resolve` once per spawn and feeds the result into the spawn pipeline (see `SPAWN_PIPELINE.md`).

The order is load-bearing: deep-merge runs at the registry level **before** resolution flattens to `AgentRuntime`, because per-kind blocks in `agents.local.toml` must field-merge into the project's per-kind blocks. Running `Resolve` first would collapse each side to a flat `AgentRuntime` and lose the pointer-vs-zero discrimination that `Override` carries.

---

## 2. Schema — `[agents]` Defaults Block

Every project's `agents.toml` carries a `[agents]` table that defines the floor every cascade kind inherits from. Fields not set in any `[agents.<kind>]` block fall through to this block. The Go-side struct (`Preset` in `internal/config/agents.go`) carries each field as a concrete value (no pointer wrapper) — at this layer "absent" and "zero value" are not distinguished; that distinction lives in the per-kind `Override` (§ 3).

```toml
[agents]
client = "claude"                  # which CLI adapter to dispatch ("claude" today; "codex" Drop 4d)
model = "claude-sonnet-4-6"        # default model for every kind
effort = "medium"                  # CLI effort knob ("low" / "medium" / "high")
max_tries = 3                      # how many times the dispatcher retries a failed spawn
max_budget_usd = 5.0               # spawn-time budget cap; agent self-aborts if exceeded
max_turns = 50                     # CLI --max-turns
blocked_retries = 0                # how many times to retry a "blocked" outcome before escalating
blocked_retry_cooldown = "30s"     # parsed by time.ParseDuration
auto_push = false                  # post-build commit-and-push gate (off by default per L20)
env_set = {}                       # literal env-var injection (non-secret)
env_from_shell = {}                # rename map: spawn-name = orch-shell-name
cli_args = []                      # extra argv tokens appended after Tillsyn-managed flags
tools_allow = ["Read", "Grep", "Glob", "Bash", "LSP"]
tools_deny = []                    # safety floor (NOT user-overridable)
claude_md_addons = []              # absolute-path body-injection list (see § 8)
```

The shipped Go field-by-field correspondence:

| TOML key                  | `Preset` field         | Go type             | Notes                                                |
| ------------------------- | ---------------------- | ------------------- | ---------------------------------------------------- |
| `client`                  | `Client`               | `string`            | adapter dispatch identity                            |
| `model`                   | `Model`                | `string`            | passed to CLI as `--model`                           |
| `effort`                  | `Effort`               | `string`            | passed to CLI as `--effort`                          |
| `max_tries`               | `MaxTries`             | `int`               | dispatcher retry policy                              |
| `max_budget_usd`          | `MaxBudgetUSD`         | `float64`           | per-spawn budget                                     |
| `max_turns`               | `MaxTurns`             | `int`               | passed to CLI as `--max-turns`                       |
| `blocked_retries`         | `BlockedRetries`       | `int`               | dispatcher retry-on-blocked-outcome count            |
| `blocked_retry_cooldown`  | `BlockedRetryCooldown` | `string`            | duration string; validated downstream                |
| `auto_push`               | `AutoPush`             | `bool`              | post-build push gate                                 |
| `env_set`                 | `EnvSet`               | `map[string]string` | literal k=v env injection                            |
| `env_from_shell`          | `EnvFromShell`         | `map[string]string` | rename map; reads orch shell at spawn                |
| `cli_args`                | `CliArgs`              | `[]string`          | extra argv tokens                                    |
| `tools_allow`             | `ToolsAllow`           | `[]string`          | tools available to the agent                         |
| `tools_deny`              | `ToolsDeny`            | `[]string`          | tools explicitly forbidden (safety floor)            |
| `claude_md_addons`        | `ClaudeMDAddons`       | `[]string`          | absolute paths whose contents append to system prompt |

Field naming follows PascalCase Go convention with snake_case TOML keys. The schema source-of-truth for ordering and rationale lives in `workflow/drop_4c_6/SKETCH.md` § 4.1.

---

## 3. Schema — `[agents.<kind>]` Per-Kind Override Blocks

Each cascade `kind` (`plan`, `build`, `research`, `plan-qa-proof`, `plan-qa-falsification`, `build-qa-proof`, `build-qa-falsification`, `closeout`, `commit`, `refinement`, `discussion`, `human-verify`) may have its own `[agents.<kind>]` block. Per-kind blocks are **partial-shape** — every field is optional. The Go-side struct (`Override` in `internal/config/agents.go`) carries each field as a **pointer** so the loader can distinguish two semantically distinct cases:

- **Pointer is nil** — the field is absent from the per-kind block. The kind inherits the `[agents]` default.
- **Pointer is non-nil** — the field is present in the per-kind block. The dereferenced value wins, **even if it is the zero value** of the Go type.

This pointer-vs-zero discrimination is load-bearing for adopters who need to override a default with the type's zero value (`0`, `""`, `false`, `[]`, `{}`). Without the pointer wrapper, "absent" and "zero" would collapse and the override would be unrepresentable.

```toml
# Per-kind blocks override only what differs (cascade-dogfooding model assignments)

[agents.plan]
# inherits client = "claude", model = "claude-sonnet-4-6", effort = "medium"
max_budget_usd = 8.0
max_turns = 80
tools_allow = ["Read", "Grep", "Glob", "Bash", "LSP", "mcp__plugin_context7_context7__resolve-library-id"]

[agents.build]
# inherits everything from [agents]
tools_allow = ["Read", "Edit", "Write", "Grep", "Glob", "Bash", "LSP"]

[agents.plan-qa-proof]
model = "claude-opus-4-7"          # QA pair runs on opus for deeper reasoning
max_budget_usd = 4.0

[agents.plan-qa-falsification]
model = "claude-opus-4-7"
max_budget_usd = 4.0

[agents.build-qa-proof]
model = "claude-opus-4-7"
max_budget_usd = 3.0

[agents.build-qa-falsification]
model = "claude-opus-4-7"
max_budget_usd = 3.0

[agents.commit]
model = "claude-haiku-4-5-20251001"  # commits run cheap+fast
max_budget_usd = 0.10
max_turns = 5
tools_allow = ["Read", "Bash"]
```

This is the **cascade-dogfooding model policy** the till-go template ships by default — planners + builders run sonnet for speed/cost, QA pair (proof + falsification, both plan-side and build-side) runs opus for deeper reasoning, commit-message-agent runs haiku for cheap-and-fast generation. Adopters change the policy by editing per-kind blocks; the schema does not pin specific model identifiers.

The closed 12-value `kind` enum is enforced at load time. Unknown TOML keys under `[agents]` (e.g. `[agents.misspelled-kind]`) are rejected by `DisallowUnknownFields()` with a position-tagged error. Use `WIKI.md` § "Cascade Vocabulary" for the canonical enum.

---

## 4. Override Semantics — Project + Local Two-Layer Merge

`MergeLocal(project, local)` (in `internal/config/agents.go`) deep-merges `agents.local.toml` over `agents.toml`. Three semantic layers compose:

**(a) Top-level Preset (concrete fields)** — the `[agents]` defaults block.

The Preset uses **concrete (non-pointer) fields**, so "zero value" is the only signal for "absent" available at this layer. Local non-zero values win; local zero values are treated as absent (project survives). This necessarily collapses "explicit zero override" and "absent" at the top-level — adopters who need explicit-zero override semantics must use a per-kind `[agents.<kind>]` Override block, which carries the pointer-based discrimination.

**(b) Per-kind Override blocks (pointer-shaped)** — the `[agents.<kind>]` partial blocks.

Local's non-nil pointers win field-by-field over project's pointers; nil pointers preserve project's pointers. Pointer-to-slice and pointer-to-map preserve the explicit-empty-vs-absent distinction.

**(c) Map-field per-key merge** — the two map fields `env_set` and `env_from_shell`.

Map fields merge per-key with **local keys winning on collision**. The merge is non-destructive: keys present in project but absent in local survive; keys present in both take local's value. Resolution at `Resolve(registry, kind)` time additionally layers the per-kind override map onto the merged Preset map.

**(d) List-field semantics** — the four list fields `cli_args`, `tools_allow`, `tools_deny`, `claude_md_addons`.

List fields are **full-replace** when present (non-empty in local Preset; non-nil pointer in local Override), **inherit** otherwise. The pointer-vs-nil distinction at the per-kind Override layer means a non-nil empty slice (`tools_allow = []`) explicitly drops a non-empty Preset list — load-bearing for adopters who need to clear a default list under a specific kind.

The merge produces a fresh `*AgentsRegistry` whose contents alias neither input. Map and slice values are deep-cloned so callers cannot mutate either input through the merged registry.

---

## 5. `env_set` vs `env_from_shell`

These are two **orthogonal** fields with different security implications. They are not interchangeable.

**`env_set`** carries **literal key=value pairs**. The values are written directly into the spawned process's environment.

```toml
env_set = { ANTHROPIC_BASE_URL = "https://openrouter.ai/api/v1" }
```

Use `env_set` for non-secret runtime configuration: API base URLs, region names, deployment identifiers, feature flags. **Never put secrets in `env_set`** — `agents.toml` is git-tracked, and committing an API key to git is a credential leak.

**`env_from_shell`** carries a **rename map**. The TOML key is the **spawn env-var name**; the TOML value is the **orch's shell env-var name**. At spawn time Tillsyn calls `os.Getenv(<shell-name>)` and injects the result under `<spawn-name>` in the spawned process's environment.

```toml
env_from_shell = { ANTHROPIC_API_KEY = "OPENROUTER_API_KEY" }
```

The example above tells Tillsyn: "read `$OPENROUTER_API_KEY` from my shell at spawn time and inject it as `$ANTHROPIC_API_KEY` into the spawned process's environment." This is the canonical secrets pattern — the secret stays in the orch's shell environment (loaded from `.envrc` / direnv / `~/.zshrc` / KeePass-derived export, etc.) and never appears in any tracked or per-project file.

Both keys and values must match the regex `^[A-Za-z][A-Za-z0-9_]*$`. Validation runs at TOML decode time; malformed names produce a `*ConfigError` envelope (see § 10) with a TOML-line pointer. A missing shell variable at spawn time is a hard failure — Tillsyn does not silently inject an empty string.

Tillsyn never validates the model name, endpoint URL, or API-key value — only the schema shape. If `ANTHROPIC_BASE_URL` points at a misconfigured proxy and the spawned CLI fails to authenticate, the failure surfaces from the CLI, not from Tillsyn.

---

## 6. `tools_allow` vs `tools_deny` Override Scope

The two list fields encode different policies and have different override scopes:

**`tools_allow`** — the **available-tools list** the agent sees. Per-machine system-tool availability varies (a contributor without `rg` installed may swap `Grep` for `Glob` + `Bash`); `tools_allow` is therefore **user-overridable** via `agents.local.toml`. A non-empty `tools_allow` in `.local.toml` replaces the project's resolved list for that kind.

**`tools_deny`** — the **safety floor**. The project author declares "this kind must never have access to these tools" (typically dangerous tools or specific MCP names). Per-machine override of the safety floor is an anti-pattern: it would let a single contributor disable a security boundary the project owner declared.

`tools_deny` is therefore **NOT user-overridable**. Setting any non-empty `tools_deny` in `agents.local.toml` — whether in the `[agents]` defaults block (`Preset.ToolsDeny`) or in any per-kind `[agents.<kind>]` Override block (`Override.ToolsDeny`) — is rejected with the closed sentinel `ErrToolsDenyNotOverridable` (defined in `internal/config/agents.go`):

```
agents.local.toml [agents]:0: tools_deny is not user-overridable; remove the field
```

The rejection happens up-front at `MergeLocal` time, before any field-merge work — so users see the violation immediately and unambiguously. The error is wrapped in a `*ConfigError` envelope with the file label `agents.local.toml` and the offending TOML block name (`[agents]` or `[agents.<kind>]`); Block-context is sufficient because successful TOML decode does not yield per-field source-line metadata. Inspect the rejection contract via `errors.Is(err, ErrToolsDenyNotOverridable)`.

The iteration order over the per-kind `Overrides` map is **deterministic** — Tillsyn iterates the closed 12-value `kind` enum in the order documented at `internal/config/agents.go` `deterministicKindOrder` rather than using Go's randomized map iteration. This produces stable, reproducible error messages when multiple per-kind blocks set `tools_deny`.

---

## 7. Frontmatter Strip Behavior

Agent files (`.tillsyn/agents/<name>.md` per-project, `~/.tillsyn/agents/<group>/<name>.md` per-user, or the embedded `internal/templates/builtin/agents/<group>/<name>.md`) carry YAML frontmatter at the head:

```yaml
---
name: builder-agent
description: Implements one atomic build droplet.
---
```

The shipped frontmatter shape is **`name` + `description` only**. Two keys that appear in some legacy or hand-authored agent files are deliberately **not** part of the shipped frontmatter shape:

- `model:` — moved to `agents.toml` (`Preset.Model` and `Override.Model`).
- `tools:` (or `allowedTools:` / `disallowedTools:`) — moved to `agents.toml` (`Preset.ToolsAllow` / `Preset.ToolsDeny` and the Override-pointer equivalents).

When the spawn pipeline renders an agent file into a per-spawn bundle (see `SPAWN_PIPELINE.md`), the **frontmatter strip helper** `StripFrontmatterKeys` (in `internal/config/frontmatter.go`) removes these keys from the frontmatter that lands in the bundle's `<bundle>/plugin/agents/<name>.md`. The rule:

- If `agents.toml` (effective resolution after MergeLocal + Resolve) sets `model =` for the kind, **strip `model:` from the frontmatter**. `agents.toml` is authoritative; the spawned CLI sees only the `--model <m>` argv flag.
- If `agents.toml` does NOT set `model =` (neither in `[agents]` defaults nor `[agents.<kind>]`), **the frontmatter `model:` survives**. Adopters who explicitly want "let the CLI's default win" omit `model =` from `agents.toml`.
- The same rule applies to `tools` / `allowedTools` / `disallowedTools` against `tools_allow` / `tools_deny`.

The strip is a **pure function** with no I/O. It accepts the frontmatter string and a `has-model` / `has-tools` flag, and returns the frontmatter string with the appropriate keys removed. The render layer in the spawn pipeline calls it once per spawn during bundle assembly.

This rule is the rendering-layer reflection of the schema-layer rule: `agents.toml` is the single source of truth for runtime configuration; frontmatter is structural agent metadata only (`name` for routing/search, `description` for the orchestrator's display).

---

## 8. `claude_md_addons` — Content-Injection Extension Point

`claude_md_addons` is a list of **absolute filesystem paths** whose contents Tillsyn loads at spawn time and **concatenates onto the agent's system prompt** (after the agent's own body). It is the extension point for adopters who want behavioral overlays without forking the shipped agent files.

```toml
[agents.build]
claude_md_addons = [
  "/Users/me/dev-rules/karpathy-four-principles.md",
  "/Users/me/dev-rules/my-team-go-style.md",
]
```

Each path is read at spawn time; missing paths are a hard error. The contents append to the rendered system prompt verbatim — Tillsyn does not render templates, expand variables, or otherwise transform the addon body.

The addon mechanism is **opt-in** and **additive**. The four behavioral principles informally referenced as "Karpathy four" (Think Before Coding / Simplicity First / Surgical Changes / Goal-Driven Execution) are baked **into the agent body itself** in the till-go and till-gen agent files — the addon mechanism is for adopters who want **additional** overlays beyond what the shipped agent body covers, not for replacing the shipped behavior.

Use cases include: per-team coding-style rules, project-specific architectural invariants, dogfood-overrides that demand specific MCP-tool usage patterns, and methodology overlays (e.g. an SDD-style spec-conformance reminder for a specific drop).

`claude_md_addons` resolves like every other field through the `Preset` → `[agents.<kind>]` Override → `MergeLocal` chain. The list is **full-replace** semantics (per § 4): a non-empty list at any layer replaces the layer below for that kind.

---

## 9. Worked Examples — Anthropic / Bedrock / Vertex / OpenRouter / Ollama Cloud

The schema is endpoint-agnostic — what differs across providers is the `model` identifier, the `ANTHROPIC_BASE_URL` (set via `env_set`), and the API-key shell name (mapped via `env_from_shell`). The five canonical worked examples below are sketches; full provider-specific configs depend on the adopter's auth flow.

### 9.1 Anthropic Direct (default)

```toml
[agents]
client = "claude"
model = "claude-sonnet-4-6"
env_from_shell = { ANTHROPIC_API_KEY = "ANTHROPIC_API_KEY" }
```

The default. The orch's `$ANTHROPIC_API_KEY` is read at spawn time and injected as `$ANTHROPIC_API_KEY` into the spawned `claude` CLI process's environment. No `env_set` needed because the CLI's default base URL hits Anthropic's API directly.

### 9.2 OpenRouter

```toml
[agents.build]
model = "anthropic/claude-opus-4"
env_set = { ANTHROPIC_BASE_URL = "https://openrouter.ai/api/v1" }
env_from_shell = { ANTHROPIC_API_KEY = "OPENROUTER_API_KEY" }
```

The `ANTHROPIC_BASE_URL` env-var redirects the `claude` CLI's API calls through OpenRouter. The orch's `$OPENROUTER_API_KEY` shell variable holds the OpenRouter key and is injected as `$ANTHROPIC_API_KEY` into the spawn — the CLI doesn't know it's not talking to Anthropic directly. Model identifiers follow OpenRouter's `<vendor>/<model>` format.

### 9.3 Amazon Bedrock

```toml
[agents.build]
model = "us.anthropic.claude-sonnet-4-6:0"
env_set = { ANTHROPIC_BEDROCK_BASE_URL = "https://bedrock-runtime.us-east-1.amazonaws.com", AWS_REGION = "us-east-1" }
env_from_shell = { AWS_ACCESS_KEY_ID = "AWS_ACCESS_KEY_ID", AWS_SECRET_ACCESS_KEY = "AWS_SECRET_ACCESS_KEY" }
```

Bedrock-flavored model IDs and AWS-style auth. The model identifier follows Bedrock's regional cross-account format (`<region-prefix>.anthropic.<model-name>`). The orch's AWS credentials pass through the `env_from_shell` rename map.

### 9.4 Google Vertex AI

```toml
[agents.build]
model = "claude-sonnet-4-6@20260901"
env_set = { CLOUD_ML_REGION = "us-east5", ANTHROPIC_VERTEX_PROJECT_ID = "my-gcp-project" }
env_from_shell = { GOOGLE_APPLICATION_CREDENTIALS = "GOOGLE_APPLICATION_CREDENTIALS" }
```

Vertex's model identifiers carry an `@<version-date>` suffix. The `GOOGLE_APPLICATION_CREDENTIALS` shell variable points at a service-account JSON key file path; that file path passes through to the spawned CLI.

### 9.5 Ollama Cloud (or local Ollama)

```toml
[agents.build]
model = "claude-sonnet-4-6"
env_set = { ANTHROPIC_BASE_URL = "https://ollama.ai/v1" }
env_from_shell = { ANTHROPIC_API_KEY = "OLLAMA_API_KEY" }
```

Same redirection pattern as OpenRouter, different base URL and key name. For local Ollama, point `ANTHROPIC_BASE_URL` at `http://localhost:11434/v1` and use a placeholder key (Ollama's local server typically ignores the `Authorization` header but a non-empty value is required for the SDK).

In every case Tillsyn validates only the schema shape — the model name, the endpoint URL, and the key value are passed through verbatim. Misconfigured providers surface their own error messages from the spawned CLI; Tillsyn does not pre-flight provider connectivity.

---

## 10. Error Handling — `*ConfigError` Envelope

Every error returned from `LoadRegistry` or `MergeLocal` is wrapped in a `*ConfigError` envelope (defined in `internal/config/agents.go`). The envelope carries file/block/line position context alongside the underlying cause:

```go
type ConfigError struct {
    File  string // user-facing file label (e.g. "agents.toml" or "agents.local.toml")
    Block string // TOML table path in bracket form (e.g. "[agents.build]")
    Line  int    // 1-based source line; 0 if unavailable
    Cause error  // wrapped underlying error
}
```

The canonical error format reads:

```
agents.local.toml [agents.build]:42: tools_deny is not user-overridable; remove the field
```

Empty `Block` or zero `Line` gracefully degrade the format — the envelope never produces misleading `:0:` artifacts in user output.

The envelope's `Unwrap()` returns `Cause`, so `errors.Is` and `errors.As` walk transitively against:

- **Sentinel errors** like `ErrToolsDenyNotOverridable` (the `tools_deny`-rejection sentinel from § 6).
- **Inner `*toml.DecodeError`** emitted by `pelletier/go-toml/v2` on malformed input — recoverable via `errors.As(err, &decodeErr)` for raw position metadata.
- **The envelope itself** — recoverable via `errors.As(err, &cfgErr)` for the formatted file/block/line context.

The envelope is **single-level by design** — composing envelope-of-envelope is out of scope. Downstream consumers (the spawn pipeline, the MCP boundary, future template validators) can extend with their own wrappers around the cause but should not re-wrap a `*ConfigError`.

---

## 11. Validation Rules and Failure Modes

Tillsyn validates `agents.toml` at load time and fails loud on invalid shapes. The current rules:

1. **`agents.toml` is required.** `LoadRegistry` returns a wrapping `fmt.Errorf("read agents.toml at %q: %w", ...)` if the file is missing or unreadable. `MergeLocal(nil, _)` returns an error with the "project registry is nil" message.
2. **Strict TOML decode.** Unknown top-level fields under `[agents]` and unknown subtables under `[agents]` fail loud via `DisallowUnknownFields()`. Typos like `[agents.bulid]` or `[agents] mxa_tries = 3` surface as `*ConfigError` envelopes wrapping `*toml.DecodeError` with TOML-line pointers.
3. **`tools_deny` rejection in `.local.toml`.** Any non-empty `tools_deny` in `agents.local.toml` — whether at `[agents]` or `[agents.<kind>]` — returns a `*ConfigError` wrapping `ErrToolsDenyNotOverridable`. Inspect via `errors.Is(err, ErrToolsDenyNotOverridable)`.
4. **Closed `kind` enum.** Per-kind blocks must use one of the closed 12 values. Unknown kind names are rejected at decode time (the per-kind fields in `agentsTOMLBlock` are explicitly typed rather than using a `map[string]Override`, so unknowns fail).
5. **Env-var name validation.** Both keys and values in `env_set` and `env_from_shell` must match `^[A-Za-z][A-Za-z0-9_]*$`. Validation runs at decode time; malformed names produce a `*ConfigError`.
6. **Missing shell variable at spawn.** `env_from_shell = { X = "MISSING" }` with no `$MISSING` in the orch's shell at spawn time is a hard failure, not a silent empty injection.

Future validation rules — the W0.5 wave's six template validators (cycle detection in `[[child_rules]]`, `blocked_by` acyclicity, `agent_name` existence across the 3-tier resolution priority, etc.) — operate on the **template** layer rather than `agents.toml`, but they share the same `*ConfigError` shape and fail-loud-at-load-time discipline.

---

## 12. Implementation Notes

The shipped Go API surface lives in `internal/config/agents.go`:

- **Types**: `Preset`, `Override`, `AgentRuntime`, `AgentsRegistry`, `ConfigError`.
- **Sentinel errors**: `ErrToolsDenyNotOverridable`.
- **Loaders**: `LoadRegistry(path string) (*AgentsRegistry, error)`.
- **Merge**: `MergeLocal(project, local *AgentsRegistry) (*AgentsRegistry, error)`.
- **Resolve**: `Resolve(registry *AgentsRegistry, kind domain.Kind) (AgentRuntime, error)`.
- **Frontmatter helper**: `StripFrontmatterKeys` (in the sibling `internal/config/frontmatter.go`).

Adopters typically interact with `agents.toml` and `agents.local.toml` only — the Go API is consumed by the spawn pipeline and the `till init` command, not by adopter code. If you are authoring a new CLI adapter (codex / cursor-agent / goose / aider / …) and need to consume the resolved `AgentRuntime`, see `CLI_ADAPTER_AUTHORING.md` for the adapter contract and the canonical pattern for building argv/env from `BindingResolved` (which is itself constructed downstream of `Resolve`).

The two-layer split — `Preset` (concrete) at the top and `Override` (pointer-shaped) per-kind — is intentional. The concrete-field top-level layer trades expressivity for ergonomics: adopters do not have to wrap every default in a pointer literal. The per-kind layer trades ergonomics for expressivity: explicit-zero override is rare but representable.

The merged registry `*AgentsRegistry` returned from `MergeLocal` carries `Path = project.Path` — the project's filesystem path, not the local file's path. User-facing error messages from `MergeLocal` use the hardcoded `localPathLabel = "agents.local.toml"` constant rather than the local file's actual on-disk path so messages remain stable regardless of the adopter's filesystem layout.

For the structural-vs-semantic split that informs why Tillsyn validates `agents.toml` shape but never validates field values (model names, endpoint URLs, API keys), see `feedback_tillsyn_enforces_templates.md`. Tillsyn enforces template and schema rules; templates and project authors define semantics.

---

*Cross-references: `CASCADE_METHODOLOGY.md` (cascade methodology spine), `SPAWN_PIPELINE.md` (per-spawn bundle pipeline that consumes `AgentRuntime`), `CLI_ADAPTER_AUTHORING.md` (CLI-adapter contract for new headless CLIs), `WIKI.md` § "Cascade Vocabulary" (canonical `kind` enum). Schema source-of-truth: `workflow/drop_4c_6/SKETCH.md` § 4–6 + § 12. Implementation source-of-truth: `internal/config/agents.go`.*
