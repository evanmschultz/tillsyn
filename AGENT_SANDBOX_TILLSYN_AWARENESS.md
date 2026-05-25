# Agent-sandbox awareness — tillsyn

> Authored from `hylla/polyglot-foundation`, 2026-05-25. tillsyn already carries the canonical
> `AGENT_SANDBOX_SPEC.md` + `AGENT_DISPATCH.md`. This note tells tillsyn **what changed in the
> 2026-05-25 consensus + the project owner's chain spec**, and how it applies to tillsyn specifically.
> tillsyn does NOT need to act now — this is the awareness record so tillsyn's build stays aligned.

## 1. tillsyn is Go-only, like sand

tillsyn's dispatch + gate logic is **Go code, never `bin/*.sh`**. The bin/sh dispatcher + the Python
`ta_action_gate.py` that now exist in hylla/ta/valv/sand are the **interim proof-of-concept reference**
— tillsyn translates the same contract into Go. The ONLY `.sh` permitted is a user's own hook.

## 2. Substrate difference (the one thing unique to tillsyn)

Everywhere the spec / the bin/sh reference says `mcp__ta__*` (the ta cascade substrate used by
hylla/ta/valv/sand), tillsyn uses **`mcp__tillsyn__till_*`** (the tillsyn coordination runtime:
`till_comment` / `till_handoff` / `till_attention_item` / `till_project` / etc.). The dispatcher's
role-conditional MCP injection must inject the **tillsyn** server, not `ta`. Everything else
(hylla read-only, context7, gopls for `*-go-*`, Playwright for `*-fe-*`) is identical.

## 3. The project owner's canonical chains (same as sand — reproduce exactly)

FE + Go identical. The dispatcher adds `-c approval_policy="never"` for codex (sandbox is inert in
exec without it); every codex role is read-only (planning + all QA-falsification are NON-EDITING).

| Role | backend | model | effort / sandbox |
|---|---|---|---|
| planning | codex-exec | gpt-5.5 | effort=low, read-only |
| plan-qa-proof | claude-native | opus | — |
| plan-qa-falsification | codex-exec | gpt-5.5 | effort=high, read-only |
| builder | claude-native | haiku (sonnet fallback) | — |
| build-qa-proof | claude-native | **sonnet** | — |
| build-qa-falsification | codex-exec | gpt-5.5 | effort=low, read-only |
| closeout | claude-native | opus | — |

Two deltas vs the older spec text tillsyn may still carry: **build-qa-proof is sonnet (not opus)**, and
**codex roles are gpt-5.5 with explicit effort (planning + build-qa-falsif = low; plan-qa-falsif =
high)**. Builders are **claude built-in haiku, not ollama** (ollama is an optional flexibility tier).

## 4. Invariants (never config-overridable)

- **GIT IS ORCHESTRATOR-ONLY** — no dispatched agent may mutate git (commit/push/add/reset/checkout/
  branch/tag/stash/rebase/merge/restore); enforced at the harness/OS layer, never by prompt.
- **EDIT-SCOPE** — an agent writes ONLY its granted file(s) (claude, per-FILE) or dir (codex, per-DIR
  `-C`), regardless of prompt. Off-scope writes FAIL; the agent reports the contradiction. QA roles get
  `edit:[]` ⇒ edit nothing.
- **ALL FE roles get Playwright MCP**; planning + plan-qa get hylla read-only; build-qa gets no hylla.
- **Veracity** — every channel RETURNS the full tool-call trace; self-report ≠ truth.

## 5. What tillsyn will EVENTUALLY need (awareness only — do NOT build this now)

This is the shape tillsyn's own implementation will eventually take, recorded so tillsyn knows it
exists and stays aligned. **It is NOT a directive to build now** — tillsyn does this on its own
schedule, and likely AFTER sand's MCP proves the pattern. Listed for awareness:

- A Go **`till gate`** PreToolUse subcommand (exec form), implementing the proven `--gate` contract:
  per-FILE edit-scope, `edit:[]` ⇒ deny-all, `bash_deny` git/command block, explicit-`allow`,
  fail-closed for scoped agents / defer for the orchestrator.
- Hermetic codex (`--ignore-user-config` + own `CODEX_HOME/rules/default.rules` execpolicy; **NOT**
  `--ignore-rules`), `-c approval_policy="never"`, `--sandbox read-only|workspace-write -C`.
- Role-conditional MCP injection on the **tillsyn** substrate (§2) + Playwright for `*-fe-*`.
- Per-channel claude routing: built-in Agent tool for OAuth roles, `claude -p --bare
  --allowedTools "Edit(//abs)"` for the API-key/ollama tier only.
- Per-run audit trace: return AND persist every dispatch's full tool-call stream, stdout, stderr, and
  run metadata (the bin/sh writes `.claude/agent-runs/<run>.{out,err,meta.json}`) so an orchestrator
  can verify agent truthfulness and nothing happened that shouldn't have.

## 6. Reference — the proven, known-good source to model from

`AGENT_SANDBOX_SPEC.md` (canonical, this repo) + `AGENT_DISPATCH.md`. The **E2E-proven (2026-05-25)**
bin/sh reference — the exact files to study/translate — lives in `hylla/polyglot-foundation`:

- `/Users/evanschultz/Documents/Code/hylla/hylla/polyglot-foundation/bin/agent-dispatch.sh` — codex
  hermetic + execpolicy git-block, `--gate` translation, role-conditional MCP injection, and **per-run
  audit capture** (`.claude/agent-runs/<run>.{out,err,meta.json}` — full tool-call trace + stderr +
  metadata, the veracity record).
- `/Users/evanschultz/Documents/Code/hylla/hylla/polyglot-foundation/.claude/agent-chains.sh` — the
  per-axis chain spec (§3).
- `/Users/evanschultz/Documents/Code/hylla/hylla/polyglot-foundation/.claude/hooks/ta_action_gate.py`
  — the PreToolUse gate: per-file edit-scope + **git-block hardened past global flags** (`git -C dir
  commit`, `git -c k=v commit`, abs-path, env-prefix, `--git-dir=`, `&&`-chained all DENY) + a
  **shell-write-bypass block** (edit-scoped agent's `cat>`/`python -c`/`sed -i`/`tee`/`cp`/`mv`/`dd of=`
  all DENIED so the per-file scope can't be circumvented via Bash).
- `/Users/evanschultz/Documents/Code/hylla/hylla/polyglot-foundation/.claude/settings.json` — the
  PreToolUse hook registration.

**Proven:** gate logic 14/14; codex read-only + execpolicy block git + writes under live `gpt-5.5`;
`-p`/ollama per-file `Edit(//abs)` + `--disallowedTools` block off-scope write + git; full trace
persisted per run. Sand's `SAND_SELF_BUILD_GUIDE.md` + `SAND_FE_AGENT_GUIDE.md` are the parallel build
briefs; tillsyn follows the same shape on its own substrate (`mcp__tillsyn__till_*`).
