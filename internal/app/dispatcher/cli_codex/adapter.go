// Package cli_codex ships the codexAdapter — Tillsyn's CLI adapter for the
// `codex` headless CLI (OpenAI Codex exec mode, JSONL stream output),
// implementing the dispatcher.CLIAdapter interface.
//
// Per F.7.17 REV-1 the adapter HARDCODES its binary name (`codex`); the
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
// ParseStreamEvent, ExtractTerminalReport. BuildCommand lives in this file;
// ParseStreamEvent + ExtractTerminalReport are in stream.go. Argv assembly
// lives in argv.go; env assembly in env.go.
//
// The codex adapter ships in Drop 4d as the multi-backend dogfood mechanism:
// plan + qa-falsification kinds route to codex while qa-proof stays on claude
// opus and build + commit stay on claude haiku/sonnet.
//
// NOTE: init.go (adapter registration with the dispatcher's adapter registry)
// ships in Drop 4d D3, not here. This package is pure implementation.
package cli_codex

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/evanmschultz/tillsyn/internal/app/dispatcher"
)

// codexBinaryName is the unconditionally-hardcoded binary name the adapter
// invokes. Per F.7.17 REV-1 there is no Command / ArgsPrefix override;
// adopters who want vendored / sandboxed codex binaries set up PATH
// themselves.
const codexBinaryName = "codex"

// codexAdapter implements dispatcher.CLIAdapter for the headless `codex` CLI
// (OpenAI Codex exec mode). It is zero-config — every per-spawn input arrives
// via the BindingResolved + BundlePaths arguments to BuildCommand, and the
// adapter holds no internal state.
//
// Tests construct the adapter via New(); production code consumes it through
// the dispatcher.CLIAdapter interface so the adapter map (Drop 4c F.7.17.5
// wiring, extended in Drop 4d D3) can swap implementations.
type codexAdapter struct{}

// Compile-time assertion that codexAdapter satisfies the
// dispatcher.CLIAdapter contract. If any of the three method signatures
// drifts from the interface, the build fails here.
var _ dispatcher.CLIAdapter = (*codexAdapter)(nil)

// New returns a fresh codexAdapter as a dispatcher.CLIAdapter. The
// concrete struct is unexported so callers cannot reach for adapter-private
// fields by accident; the interface is the only supported surface.
func New() dispatcher.CLIAdapter {
	return &codexAdapter{}
}

// BuildCommand assembles the *exec.Cmd that invokes codex for one spawn.
// It hardcodes the binary name `codex`, relying on the spawn's cmd.Env
// PATH (set by assembleEnv) for binary resolution at exec time.
//
// The system prompt is fed to the codex process via stdin — codex reads
// the prompt from stdin when no positional [PROMPT] argument is provided
// (per OQ1: "If not provided as an argument (or if `-` is used), instructions
// are read from stdin"). The file at paths.SystemPromptPath is read entirely
// into memory with os.ReadFile and injected as a bytes.Reader. This avoids
// an open *os.File fd that would never be closed by cmd.Wait — bytes.Reader
// carries no fd lifecycle concern.
//
// Returns an error if any name in binding.Env is unset in the orchestrator
// process (fail-loud per F.7.17 P5) or if paths.SystemPromptPath cannot be
// read.
//
// ctx is plumbed through exec.CommandContext so the dispatcher's lifecycle
// (timeout / cancellation) propagates to the spawned codex process.
func (a *codexAdapter) BuildCommand(
	ctx context.Context,
	binding dispatcher.BindingResolved,
	paths dispatcher.BundlePaths,
) (*exec.Cmd, error) {
	// Create hermetic CODEX_HOME to isolate the codex process from the
	// orchestrator's global ~/.codex (skills, rules, plugins, memories, etc.).
	// Only the 4 auth files are symlinked; everything else is absent.
	// The hermetic directory is created under paths.Root so Bundle.Cleanup()
	// automatically reaps it post-spawn.
	hermeticHome, err := newHermeticCodexHome(paths.Root)
	if err != nil {
		return nil, fmt.Errorf("cli_codex: build command: hermetic codex home: %w", err)
	}

	// Inject CODEX_HOME into the spawn's environment as a literal (not
	// os.LookupEnv). This overrides the per-binding Env list and takes
	// precedence via the assembleEnv precedence chain: binding.Env >
	// envSetLiterals > defense-in-depth > closed-baseline.
	envWithHermetic := binding.EnvSet
	if envWithHermetic == nil {
		envWithHermetic = make(map[string]string)
	}
	envWithHermetic["CODEX_HOME"] = hermeticHome

	env, err := assembleEnv(binding, envWithHermetic)
	if err != nil {
		return nil, fmt.Errorf("cli_codex: build command: %w", err)
	}

	// Read the system prompt file into memory for stdin injection. codex reads
	// its prompt from stdin when no positional argument is given (OQ1). We
	// read the entire file here so BuildCommand fails loud if the path is
	// absent, and we avoid leaving an open *os.File that cmd.Wait would not
	// close — bytes.Reader carries no fd lifecycle concern.
	promptBytes, err := os.ReadFile(paths.SystemPromptPath)
	if err != nil {
		return nil, fmt.Errorf("cli_codex: build command: read system prompt: %w", err)
	}

	argv := assembleArgv(binding, paths)
	cmd := exec.CommandContext(ctx, codexBinaryName, argv[1:]...)
	cmd.Env = env
	cmd.Stdin = bytes.NewReader(promptBytes)
	return cmd, nil
}

// ParseStreamEvent decodes one JSONL line from codex's --json output channel
// into the cross-CLI canonical dispatcher.StreamEvent shape. Implementation
// lives in stream.go.
func (a *codexAdapter) ParseStreamEvent(line []byte) (dispatcher.StreamEvent, error) {
	return parseStreamEvent(line)
}

// ExtractTerminalReport pulls the terminal report (cost, denials, reason,
// errors) out of a parsed StreamEvent. Returns (zero, false) for events
// that are not the terminal event. Implementation lives in stream.go.
func (a *codexAdapter) ExtractTerminalReport(ev dispatcher.StreamEvent) (dispatcher.TerminalReport, bool) {
	return extractTerminalReport(ev)
}
