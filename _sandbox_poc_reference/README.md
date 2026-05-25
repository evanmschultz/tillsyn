# _sandbox_poc_reference — TEMPORARY bin/sh + hook testbed

> **Temporary.** The proven bin/sh + Python-hook agent-sandbox reference from
> `hylla/polyglot-foundation` (see `../HYLLA_BIN.md`), pulled in to test the
> mechanisms in tillsyn's env while the **type-safe Go translation** (`till gate`
> + hermetic codex argv + `--gate` contract + role-conditional `till_*`
> injection) is built in the D-AGENT-GATE cascade. **Removed once the Go lands.**
> tillsyn's Hard Rule forbids bin/*.sh dispatch in production — this dir is the
> reference testbed only, NOT shipped dispatch, NOT at `bin/`.

## What's wired (so dispatched agents actually have their hooks)

- **Gate hook** placed at `../.claude/hooks/ta_action_gate.py` and **registered**
  in `../.claude/settings.json` PreToolUse (`Edit|Write|MultiEdit|NotebookEdit|Bash`).
  Takes effect next session (Claude Code reads settings at startup).
- **Chains** placed at `../.claude/agent-chains.sh` (closeout=haiku, tillsyn override).
- **Dispatcher** stays here (`agent-dispatch.sh`); its `REPO_ROOT` resolves to
  `main/`, so it reads `../.claude/agent-chains.sh` + `../.claude/agents/<role>.md`.
  Substrate updated: injects `mcp_servers.tillsyn` (`till mcp`, `till_*` tools) +
  `mcp_servers.ta` (schema MDs) + hylla(read, non-build-qa) + context7 + gopls/playwright.
- **gate_test.sh** — 14/14 PASS battery proving the hook logic.

## The two enforcement channels

1. **Built-in Agent tool (OAuth roles: builder/qa-proof/closeout)** — gated by the
   registered PreToolUse hook. The orchestrator MUST embed this block at the top of
   every scoped spawn prompt so the hook can resolve the per-dispatch allowlist:

   ```
   <TA_ALLOWLIST>
   {"edit": ["/abs/file_a.go", "/abs/file_a_test.go"],
    "bash_deny": ["git commit","git push","git add","git reset","git checkout",
                  "git branch","git tag","git stash","git rebase","git merge",
                  "git restore","mage install","go get","go mod"]}
   </TA_ALLOWLIST>
   ```

   - QA / closeout roles → `"edit": []` (deny ALL edits; they only update tillsyn/ta MCP).
   - No block in the prompt ⇒ the hook DEFERS (un-scoped dispatch runs ungated) — so
     forgetting the block degrades to "ungated", it does not break dispatch.
   - The orchestrator (no `agent_id`) is never gated.

2. **codex exec (planning + *-falsification)** — gated by the dispatcher's hermetic
   `CODEX_HOME/rules/default.rules` execpolicy (git/command block) + `--sandbox
   read-only` + `approval_policy="never"`. The PreToolUse hook does NOT apply to
   codex (codex hooks are dead on 0.133.0); execpolicy is the floor.
   `echo "<task>" | ./agent-dispatch.sh --role <role> --cwd <dir> --gate '<json>'`

## Invariant on both channels

GIT IS ORCHESTRATOR-ONLY. No dispatched agent commits/pushes/adds/resets/etc. — the
hook's `bash_deny` (built-in) and the execpolicy `prefix_rule(forbidden)` (codex)
both block it, past global-flag evasions (`git -C dir commit`) and shell-write
bypasses (`echo>`, `sed -i`, `python3 -c`). Verified: `gate_test.sh` 14/14.
