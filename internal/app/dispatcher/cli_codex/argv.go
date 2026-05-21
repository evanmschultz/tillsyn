package cli_codex

import (
	"github.com/charmbracelet/log"

	"github.com/evanmschultz/tillsyn/internal/app/dispatcher"
)

// assembleArgv returns the full argv slice (including argv[0] = "codex")
// the dispatcher passes to exec.Command for one codex spawn. Per F.7.17
// REV-1 the binary name is hardcoded — no Command override path.
//
// Argv shape (per OQ1 codex exec --help):
//
//	codex exec --json --ephemeral --skip-git-repo-check -C <paths.Root> \
//	  [-m <model>] \
//	  [-c model_reasoning_effort=<effort>]
//
// The positional [PROMPT] argument is intentionally omitted — the caller
// (BuildCommand) feeds the system prompt via cmd.Stdin instead. codex reads
// its prompt from stdin when no positional argument is provided (OQ1:
// "If not provided as an argument (or if `-` is used), instructions are read
// from stdin"). This mirrors cli_claude's system-prompt-file pattern.
//
// Conditional flags (-m, -c model_reasoning_effort) emit ONLY when the
// corresponding pointer-typed BindingResolved field is non-nil per F.7.17 L9.
// nil means "the lower-priority layer is authoritative"; codex's CLI defaults
// apply when the flag is omitted entirely.
//
// Note: paths.Root is passed via -C so codex's working root is the bundle
// directory. Adapters compute their own CLI-specific subdirs under Root per
// F.7.17 L13; codex's bundle layout (if needed) is computed here or in future
// D3/D4 droplets.
func assembleArgv(binding dispatcher.BindingResolved, paths dispatcher.BundlePaths) []string {
	// Cap initial slice at a reasonable upper bound so we avoid mid-build
	// reallocs without over-reserving. Slack covers conditional flags.
	argv := make([]string, 0, 16)

	// argv[0] is the binary name. exec.CommandContext resolves it via PATH from
	// the spawn's cmd.Env (set by assembleEnv) at exec time.
	argv = append(argv, codexBinaryName)

	// Sub-command first.
	argv = append(argv, "exec")

	// Always-on flags in a stable order (important for test snapshotting and
	// log readability).
	argv = append(argv,
		"--json",
		"--ephemeral",
		"--skip-git-repo-check",
		"-C", paths.Root,
	)

	// Conditional pointer-typed flags. Emit-only-on-non-nil is the F.7.17 L9
	// contract: nil means "the lower-priority layer is authoritative" and
	// the adapter MUST NOT synthesize a value.
	if binding.Model != nil {
		argv = append(argv, "-m", *binding.Model)
	}
	if binding.Effort != nil {
		// codex exposes reasoning effort via -c config override:
		//   -c model_reasoning_effort=<value>
		// (per OQ1: -c, --config <key=value>  Override a config value)
		argv = append(argv, "-c", "model_reasoning_effort="+*binding.Effort)
	}

	// MaxTurns and MaxBudgetUSD are not supported by the codex CLI adapter.
	// The codex exec sub-command has no equivalent flags for these fields.
	// Silently dropping them would violate parity-and-clarity doctrine
	// (feedback_parity_clarity_no_silent_failures.md). Log a WARN at
	// BuildCommand time so the caller receives an observable signal without
	// breaking the spawn — these are informational caps, not hard blockers.
	if binding.MaxTurns != nil {
		log.Warn("cli_codex: MaxTurns is not supported by the codex adapter; field ignored",
			"agent", binding.AgentName,
			"max_turns", *binding.MaxTurns,
		)
	}
	if binding.MaxBudgetUSD != nil {
		log.Warn("cli_codex: MaxBudgetUSD is not supported by the codex adapter; field ignored",
			"agent", binding.AgentName,
			"max_budget_usd", *binding.MaxBudgetUSD,
		)
	}

	return argv
}
