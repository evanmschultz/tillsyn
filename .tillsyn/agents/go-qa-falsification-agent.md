---
name: go-qa-falsification-agent
description: PLACEHOLDER — legacy go-prefixed QA-falsification name retained until Drop 4c.6 W5.D3 strips the go- prefix from default-go.toml's agent_bindings. Substantive content lands in Drop 4c.8 W4.
---

# PLACEHOLDER — substantive content lands in Drop 4c.8 W4

This placeholder satisfies the W0.5 `validateAgentBindingNames` validator's
embedded-tier lookup for the legacy `go-qa-falsification-agent` name still
referenced by `internal/templates/builtin/default-go.toml` (used by both
`plan-qa-falsification` and `build-qa-falsification` agent_bindings). The
default-go.toml rename + name-strip lands in Drop 4c.6 W5.D1 / W5.D3; this
file goes away alongside that cleanup.
