// Package cli_claude ships the claudeAdapter — Tillsyn's only CLI adapter
// in Drop 4c, implementing the dispatcher.CLIAdapter interface for the
// `claude` headless CLI.
//
// Per F.7.17 REV-1 the adapter HARDCODES its binary name (`claude`); the
// wrapper-interop knob (Command / ArgsPrefix on AgentBinding) is gone from
// Tillsyn's design. Process isolation is an OS-level concern (PATH-shadowed
// shim, container, sandbox) — not a Tillsyn surface.
//
// Per F.7.17 REV-2 + L4 / L6 / L7 / L8 the adapter constructs cmd.Env
// explicitly: `os.Environ()` is NOT inherited. Cmd.Env is the closed POSIX
// baseline (process-basics + proxy/TLS-cert names) PLUS the resolved values
// for every name in BindingResolved.Env. Missing required env-var names
// fail loud at BuildCommand time so the dispatcher routes the failure to
// pre-lock per F.7.17 P5.
//
// Per F.7.17 L10 the interface has exactly three methods: BuildCommand,
// ParseStreamEvent, ExtractTerminalReport. Their implementations live in
// this file (BuildCommand) and stream.go (ParseStreamEvent +
// ExtractTerminalReport). Argv assembly lives in argv.go; env assembly in
// env.go.
//
// Future Drop 4d codex adapter ships in a sibling package
// (`internal/app/dispatcher/cli_codex/`) following the same shape.
package cli_claude

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/evanmschultz/tillsyn/internal/app/dispatcher"
)

// claudeBinaryName is the unconditionally-hardcoded binary name the adapter
// invokes. Per F.7.17 REV-1 there is no Command / ArgsPrefix override;
// adopters who want vendored / sandboxed claude binaries set up PATH
// themselves (see docs/architecture/cli_adapter_authoring.md when it lands
// in F.7.17.9).
const claudeBinaryName = "claude"

// claudeAdapter implements dispatcher.CLIAdapter for the headless `claude`
// CLI. It is zero-config — every per-spawn input arrives via the
// BindingResolved + BundlePaths arguments to BuildCommand, and the adapter
// holds no internal state.
//
// Tests construct the adapter via New(); production code consumes it
// through the dispatcher.CLIAdapter interface so the adapter map (Drop
// 4c F.7.17.5 wiring) can swap implementations.
type claudeAdapter struct{}

// Compile-time assertion that claudeAdapter satisfies the
// dispatcher.CLIAdapter contract. If any of the three method signatures
// drifts from the interface, the build fails here.
var _ dispatcher.CLIAdapter = (*claudeAdapter)(nil)

// New returns a fresh claudeAdapter as a dispatcher.CLIAdapter. The
// concrete struct is unexported so callers cannot reach for adapter-private
// fields by accident; the interface is the only supported surface.
func New() dispatcher.CLIAdapter {
	return &claudeAdapter{}
}

// BuildCommand assembles the *exec.Cmd that invokes claude for one spawn.
// It hardcodes the binary name `claude`, relying on the spawn's cmd.Env
// PATH (set by assembleEnv) for binary resolution at exec time.
//
// Returns an error if any name in binding.Env is unset in the orchestrator
// process (fail-loud per F.7.17 P5; the dispatcher routes the failure to
// pre-lock so no lock is held against a doomed spawn).
//
// ctx is plumbed through exec.CommandContext so the dispatcher's lifecycle
// (timeout / cancellation) propagates to the spawned claude process.
func (a *claudeAdapter) BuildCommand(
	ctx context.Context,
	binding dispatcher.BindingResolved,
	paths dispatcher.BundlePaths,
) (*exec.Cmd, error) {
	env, err := assembleEnv(binding)
	if err != nil {
		return nil, fmt.Errorf("cli_claude: build command: %w", err)
	}

	argv := assembleArgv(binding, paths)
	cmd := exec.CommandContext(ctx, claudeBinaryName, argv[1:]...)
	cmd.Env = env
	return cmd, nil
}

// ParseStreamEvent decodes one JSONL line from claude's --output-format
// stream-json channel into the cross-CLI canonical dispatcher.StreamEvent
// shape. Implementation lives in stream.go.
func (a *claudeAdapter) ParseStreamEvent(line []byte) (dispatcher.StreamEvent, error) {
	return parseStreamEvent(line)
}

// ExtractTerminalReport pulls the terminal report (cost, denials, reason,
// errors) out of a parsed StreamEvent. Returns (zero, false) for events
// that are not the terminal `result` event. Implementation lives in
// stream.go.
func (a *claudeAdapter) ExtractTerminalReport(ev dispatcher.StreamEvent) (dispatcher.TerminalReport, bool) {
	return extractTerminalReport(ev)
}
