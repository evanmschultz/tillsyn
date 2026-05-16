---
name: builder-agent
description: PLACEHOLDER — language-agnostic builder agent (till-gen group). Substantive content lands in Drop 4c.8 W4.
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
