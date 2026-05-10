---
name: orchestrator-managed
description: PLACEHOLDER — orchestrator-managed coordination kinds (closeout, refinement, discussion, human-verify) bind this name in default-go.toml. Substantive content lands in Drop 4c.8 W4 if these kinds get dedicated agents; otherwise the binding remains a dispatcher-side placeholder per CLAUDE.md "Agent Bindings" table.
---

# PLACEHOLDER — substantive content lands in Drop 4c.8 W4

Coordination kinds (closeout / refinement / discussion / human-verify) are
orchestrator-managed today per `CLAUDE.md` "Agent Bindings" table — they do
not spawn an automated agent. The `orchestrator-managed` binding exists in
`default-go.toml` so the dispatcher (Drop 4) has a row to look up; this
placeholder satisfies the W0.5 `validateAgentBindingNames` embedded-tier
lookup. If a future drop wires real automated agents for these kinds, the
substantive prompt content lands here in Drop 4c.8 W4 (or later).
