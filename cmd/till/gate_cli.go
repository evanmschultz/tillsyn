package main

import (
	"encoding/json"
	"strings"

	"github.com/spf13/cobra"
)

// gateRunCommandOptions carries flags consumed by `till gate`.
// The skeleton accepts no flags; all behavior is driven by stdin.
type gateRunCommandOptions struct{}

// runGate is the gate CLI's RunE body, implementing the PreToolUse event
// handler skeleton. It reads a JSON-encoded PreToolUse event from stdin,
// evaluates the gate decision via the pretoolgate package, and exits 0
// regardless of outcome (fail-open — a gate bug must never block a tool call).
//
// The function is responsible for:
//   - Reading stdin (standard io.Reader via cobra).
//   - Parsing the event JSON or deferring on parse error.
//   - Evaluating the gate decision (deferred to FND.1 package, filled by A).
//   - Writing the decision to stdout as JSON (when logic is ready).
//   - Returning nil (always — fail-open semantics).
func runGate(ctx *cobra.Command, _ []string) error {
	// Skeleton: read stdin, parse JSON, defer on error, exit 0.
	// The actual gate logic (14 cases from ta_action_gate.py) is deferred
	// to a later droplet that fills the decision logic on top of this
	// skeleton.

	stdin := ctx.InOrStdin()
	if stdin == nil {
		stdin = strings.NewReader("")
	}

	event := struct{}{}
	if err := json.NewDecoder(stdin).Decode(&event); err != nil {
		// Parse error: defer to parent (dev keeps normal control).
		// exit 0, no output.
		return nil
	}

	// The gate decision logic will be populated by a later droplet (A).
	// For now, this skeleton just defers: exit 0, no output.
	// When A fills the decision logic, it will use the parsed event
	// and the pretoolgate package to evaluate and write the decision.

	// Always exit 0 (fail-open).
	return nil
}

// newGateCommand creates the `till gate` cobra subcommand.
func newGateCommand() *cobra.Command {
	gateCmd := &cobra.Command{
		Use:   "gate",
		Short: "PreToolUse event handler (agent sandboxing gate)",
		Long: strings.TrimSpace(`
Evaluate a PreToolUse hook event and decide whether to allow, deny, or defer
the tool invocation. The event is read from stdin as JSON; the decision is
written to stdout (or deferred via exit 0 with no output).

This is the security boundary that confines dispatched subagents to their
dispatch-time allowlist. The gate is fail-open: a gate bug must never block
a tool call.

For orchestrator/dev sessions (no agent_id), the gate defers. For scoped
subagents, the dispatch allowlist is the sole authority — every action is
either explicitly allowed (no dev prompt) or explicitly denied (with a reason).
`),
		Example: strings.Join([]string{
			"  echo '{...event...}' | till gate",
			"  till gate < /tmp/event.json",
		}, "\n"),
		Args: cobra.NoArgs,
		RunE: runGate,
	}
	return gateCmd
}
