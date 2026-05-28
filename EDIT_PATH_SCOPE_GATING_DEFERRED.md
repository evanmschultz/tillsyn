# Edit-Path Scope Gating — DEFERRED Design

Status: **DEFERRED 2026-05-27**. Captured here so when we come back to it we don't re-do the design discovery.

## The Problem

Each dispatched agent (built-in `Agent` tool subagent OR `claude -p` headless subprocess) should be confined to editing ONLY a declared set of file paths. Different dispatches of the SAME persona may need different path sets — `ta-fe-builder` running on `ui/foo.tsx` should not be able to edit `ui/bar.tsx`, even though another concurrent `ta-fe-builder` dispatch DOES have `ui/bar.tsx` in its scope.

In short: per-dispatch path scope, NOT per-persona path scope.

This makes the scope DYNAMIC (per-dispatch) and CONCURRENT-SAFE (parallel same-persona dispatches must not interfere).

## Why It Matters

Today's gating shape is per-persona — the persona's `tools:` frontmatter and (in the proposed architecture) the persona's `settings.json` carry static allow/deny rules for tool surface (which MCPs, which Bash patterns). That's enough for "can this persona use git commit" but NOT for "can this dispatched ta-fe-builder edit `ui/foo.tsx`."

Without per-dispatch path scope:

- A misbehaving agent can edit ANY file in its `Edit` allowlist (typically `**/*` for builders).
- Two parallel same-persona dispatches step on each other's files.
- We lose the cascade methodology's "1-2 small code blocks per droplet" structural enforcement that file-scope provides.

## Constraints

Any solution must satisfy:

1. **Per-dispatch granularity**: each Agent / bin/sh `-p` dispatch can declare its own path slice.
2. **Concurrent same-persona safety**: 3 `ta-fe-builder` dispatched in parallel must each see only their own paths.
3. **Both dispatch paths**: works for built-in `Agent` tool AND bin/sh `claude -p`. Eventually for sand (Go MCP) too.
4. **No requirement of Tillsyn**: must work today for projects that don't yet use Tillsyn (ta, valv, sand, hylla-poly).
5. **Tillsyn integration is additive**: when Tillsyn is wired, replace the `/tmp` mechanism with `action_item.paths` reads. Same hook enforcement logic.

## Hook Input Reality (Empirically Verified)

The PreToolUse hook fired by a subagent's tool call receives (per `ta_action_gate.py` lines 7-15, empirically verified 2026-05-24):

- `agent_id` — unique per subagent instance. Present ONLY for subagent tool calls. Absent for top-level claude code sessions (orchestrator + bin/sh `claude -p` dispatches).
- `agent_type` — persona name (e.g. `ta-go-builder`).
- `transcript_path` — the PARENT (orchestrator) transcript path. Subagents do not have their own transcript file.
- `tool_name` / `tool_input` / `cwd` / `session_id` (parent's).

What is NOT available:

- A "subagent spawned" hook event with `agent_id` BEFORE the subagent's first tool call. No `SubagentStart` event exists in claude code 2.1.152. The closest events are `PreToolUse` on `Agent` matcher (fires BEFORE dispatch — no `agent_id` yet) and `PostToolUse` on `Agent` matcher (fires AFTER dispatch ends — too late, every Edit already ran).

## Solutions Considered

### Solution A — Dispatch state file in /tmp + token-in-prompt (built-in) OR env-var (bin/sh)

The unified shape:

1. Orch generates UUID4 `TOKEN`, writes `/tmp/till-dispatch-<TOKEN>.json` with full call context (paths, prompt, persona, model, etc.).
2. For built-in: orch dispatches `Agent(subagent_type=X, prompt="<TILL_DISPATCH_TOKEN>UUID</TILL_DISPATCH_TOKEN>\n\nactual prompt")`. Token embedded in spawn prompt.
3. For bin/sh: dispatcher sets `TILL_DISPATCH_TOKEN=<TOKEN>` env on spawned `claude -p`.
4. PreToolUse hook on subagent's first tool call (built-in) OR top-level subprocess's first tool call (bin/sh):
   - If `agent_id` present (built-in): scan parent transcript for Agent tool_use with matching `agent_id` in `tool_response`, extract `TILL_DISPATCH_TOKEN` from that Agent's `input.prompt`.
   - If `agent_id` absent (bin/sh): read `TILL_DISPATCH_TOKEN` from hook's own env.
   - Either way: read `/tmp/till-dispatch-<TOKEN>.json`. Apply `paths_allowed` against tool's target file.

Pros: works today, no claude code source changes. Decouples dispatch mechanism from persistence (Tillsyn can replace `/tmp` later with no hook logic change).

Cons: parent-transcript scan on every built-in subagent tool call (O(transcript) per call) unless refined.

### Solution B — Rename-on-first-tool-call (refinement of A)

Same as A, but on the FIRST built-in subagent tool call, the hook RENAMES `/tmp/till-dispatch-<TOKEN>.json` → `/tmp/till-dispatch-<agent_id>.json`. Subsequent tool calls look up by `agent_id` directly — no transcript scan needed.

Cost: first call pays the scan (transcript scan + atomic file rename). Subsequent calls are O(1) file read.

Pros: amortizes transcript scan to once per dispatch.

Cons: code complexity slightly higher (rename logic + idempotency).

### Solution C — Tillsyn-mediated (future)

When Tillsyn is the orchestrator's first-class system:

1. Orch sets `action_item.metadata.dispatch_paths` (or just relies on the already-existing `action_item.paths` field) before dispatching.
2. Orch dispatches with the action_item_id embedded in spawn prompt.
3. Hook reads action_item_id from spawn prompt → queries `mcp__tillsyn__till_action_item(get, action_item_id)` → reads `.paths` → enforces.

Pros: single source of truth in Tillsyn. No `/tmp` files. Survives crashes. Auditable.

Cons: hook makes MCP call per tool invocation (slower than file read). Requires Tillsyn-orchestrated dispatch flow (not all projects today).

### Solution D — Hook on agent creation (NOT POSSIBLE)

Tried and discarded. Claude code 2.1.152 has no hook event that fires at the moment `agent_id` is generated and BEFORE the subagent's first tool call. `PreToolUse` on `Agent` matcher fires too early (no `agent_id` yet); `PostToolUse` on `Agent` and `SubagentStop` fire too late (subagent done).

### Solution E — Persona-cooperation token surfacing

Persona body instructed to call `Bash echo TILL_DISPATCH_TOKEN=<token>` as its FIRST tool call. Hook captures the token from that command. Subsequent tool calls look up by `agent_id`.

Pros: token-passing is explicit + auditable.

Cons: requires persona compliance; bricks if persona reorders; one wasted tool call per dispatch.

## Why Deferred

We confirmed the design space and shape; implementation is non-trivial (extend `ta_action_gate.py`, change dispatcher to write state files + pass env vars, design /tmp lifecycle / cleanup, handle edge cases). Per dev directive 2026-05-27, the immediate priority is:

1. Drop `--bare` from dispatcher (parity with built-in).
2. Per-persona `settings.json` for static tool-surface gating (allow/deny patterns).
3. Hook enforcement of per-persona settings for built-in Agent path (since `--settings` doesn't work mid-dispatch for built-in).

Per-dispatch dynamic path scope is the next layer. We defer until the above lands and we have validated the simpler shape.

## Pickup Notes (when we resume)

Recommended path on resume: **Solution B (rename-on-first-tool-call)** for the interim, with the understanding that Solution C (Tillsyn-mediated) replaces it once Tillsyn-orchestrated dispatch is the dominant path. Solution B's mechanism naturally migrates: orch writes the same state-file shape; the only change is the source of truth becomes Tillsyn instead of /tmp.

State file schema to design when we resume:

```json
{
  "dispatch_token": "uuid4-string",
  "dispatched_at": "RFC3339 timestamp",
  "dispatched_by": "principal-id",
  "persona": "ta-fe-builder",
  "model": "claude-haiku-4-5",
  "channel": "built-in-agent | claude-p-ollama | claude-p-oauth | codex-exec",
  "paths_allowed": ["abs/path/to/file1", "abs/path/to/file2"],
  "tillsyn_action_item_id": "optional"
}
```

Hook logic to add to `ta_action_gate.py`:

1. On any Edit/Write/MultiEdit tool call:
   - Resolve dispatch context (agent_id → transcript scan → token, OR env var → token).
   - Read `/tmp/till-dispatch-<TOKEN>.json` (or `/tmp/till-dispatch-<agent_id>.json` after rename).
   - Compare tool's target file against `paths_allowed`. Allow / deny.
2. On first subagent tool call (any type), if `/tmp/till-dispatch-<TOKEN>.json` exists but `/tmp/till-dispatch-<agent_id>.json` doesn't: rename.
3. Lifecycle: orch deletes `/tmp/till-dispatch-<TOKEN>.json` (or `/tmp/till-dispatch-<agent_id>.json`) after dispatch completes. Stale-file sweep at orch start (TTL 1h) as safety net.

## Cross-References

- `AGENT_SANDBOX_SPEC.md` — the broader dispatch architecture.
- `tillsyn/main/.claude/hooks/ta_action_gate.py` — existing hook that already does `<TA_ALLOWLIST>` parsing on Agent's `input.prompt`. The extension landing here will REPLACE that mechanism with the state-file form.
- Project CLAUDE.md `## Paths and Packages` — action_item's `paths`/`packages` fields. These are the ALREADY-EXISTING Tillsyn primitives that Solution C reads from.
