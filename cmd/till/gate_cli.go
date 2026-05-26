package main

import (
	"encoding/json"
	"strings"

	"github.com/spf13/cobra"

	"github.com/evanmschultz/tillsyn/internal/app/dispatcher/pretoolgate"
)

// gateRunCommandOptions carries flags consumed by `till gate`.
// The skeleton accepts no flags; all behavior is driven by stdin.
type gateRunCommandOptions struct{}

// runGate is the gate CLI's RunE body, implementing the PreToolUse event
// handler. It reads a JSON-encoded PreToolUse event from stdin,
// evaluates the gate decision via the pretoolgate package, and exits 0
// regardless of outcome (fail-open — a gate bug must never block a tool call).
//
// The function is responsible for:
//   - Reading stdin (standard io.Reader via cobra).
//   - Parsing the event JSON into a pretoolgate.Event or deferring on parse error.
//   - Evaluating the gate decision via pretoolgate.Decide(event).
//   - Writing the decision to stdout as JSON (when deferred is false).
//   - Returning nil (always — fail-open semantics).
func runGate(ctx *cobra.Command, _ []string) error {
	stdin := ctx.InOrStdin()
	if stdin == nil {
		stdin = strings.NewReader("")
	}

	var event pretoolgate.Event
	if err := json.NewDecoder(stdin).Decode(&event); err != nil {
		// Parse error: defer to parent (dev keeps normal control).
		// exit 0, no output.
		return nil
	}

	// Evaluate the gate decision against the resolved allowlist.
	decision := pretoolgate.Decide(event)

	// For a deferred decision (ungated orchestrator), write nothing and exit 0.
	// Otherwise, marshal the allow/deny decision JSON and write to stdout.
	if !decision.Defer {
		output, err := pretoolgate.MarshalDecision(decision)
		if err != nil {
			// Marshal error: fail-open, defer to parent.
			return nil
		}
		if len(output) > 0 {
			ctx.OutOrStdout().Write(output)
		}
	}

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
