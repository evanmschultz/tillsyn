---
name: builder-agent
description: PLACEHOLDER — Go+mage-tuned builder agent (till-go group). Substantive content lands in Drop 4c.8 W4.
hooks:
  PreToolUse:
    - matcher: "Edit|Write|Bash"
      hooks:
        - type: command
          command: "./.claude/hooks/validate-action-item-paths.sh"
---

# PLACEHOLDER — substantive content lands in Drop 4c.8 W4

This file is a Drop 4c.6 W1.D1 scaffolding placeholder. Its only purpose is to
let the embedded-FS resolver path land before Drop 4c.8 W4 authors the
substantive prompt content.

## Contract

The builder agent DOES NOT call `mcp__tillsyn__till_action_item` with
`operation=move_state state=complete`. Reporting success via Tillsyn metadata
and a closing comment is sufficient; the monitor (wired in Drop 4b) owns the
final `in_progress -> complete` transition after post-build gates pass.

On unrecoverable error the builder MAY set `metadata.outcome=failure` (or
`metadata.outcome=blocked`) and exit while leaving the action item in
`in_progress`. The monitor reads `metadata.outcome` to decide the terminal
state transition — it is NOT the builder's responsibility.

## Hook Environment Variables

The `validate-action-item-paths.sh` hook declared in this file's frontmatter
expects two environment variables set by the dispatcher before the agent runs:

- `TILLSYN_ACTION_ITEM_ID` — the UUID of the action item this agent instance
  owns. The hook uses this to fetch the declared `paths` and `packages` fields
  and enforce that every `Edit`, `Write`, or `Bash` file-mutation stays within
  the declared scope.
- `TILLSYN_BIN` — the absolute path to the `till` binary. The hook invokes
  `$TILLSYN_BIN action_item get $TILLSYN_ACTION_ITEM_ID` to resolve the scope
  at hook-fire time.
