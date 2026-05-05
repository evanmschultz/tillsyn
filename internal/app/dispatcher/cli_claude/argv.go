package cli_claude

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/evanmschultz/tillsyn/internal/app/dispatcher"
)

// pluginSubdir is the conventional path under BundlePaths.Root that the
// dispatcher's bundle-materializer (F.7-CORE F.7.1) populates with the
// claude-shaped plugin tree:
//
//	<bundle.Root>/plugin/.claude-plugin/plugin.json
//	<bundle.Root>/plugin/agents/<agent-name>.md
//	<bundle.Root>/plugin/.mcp.json
//	<bundle.Root>/plugin/settings.json
//
// claudeAdapter materializes its own subdir layout under Root per F.7.17
// L13; this constant is the claude-specific convention. Other adapters
// (codex in Drop 4d) compute their own subdirs.
const pluginSubdir = "plugin"

// assembleArgv returns the full argv slice (including argv[0] = "claude")
// the dispatcher passes to exec.Command for one claude spawn. Per F.7.17
// REV-1 the binary name is hardcoded — no Command override path.
//
// Argv shape per project_drop_4c_spawn_architecture.md §3:
//
//	claude --bare --plugin-dir <bundle>/plugin --agent <name> \
//	  --system-prompt-file <bundle>/system-prompt.md \
//	  [--append-system-prompt-file <bundle>/system-append.md] \
//	  --settings <bundle>/plugin/settings.json --setting-sources "" \
//	  --strict-mcp-config --permission-mode acceptEdits \
//	  --output-format stream-json --verbose \
//	  --no-session-persistence --exclude-dynamic-system-prompt-sections \
//	  --mcp-config <bundle>/plugin/.mcp.json \
//	  [--max-budget-usd <N>] [--max-turns <N>] [--effort <e>] \
//	  [--model <m>] [--tools <list>] \
//	  -p ""
//
// Conditional flags (--max-budget-usd, --max-turns, --effort, --model,
// --append-system-prompt-file, --tools) emit ONLY when the corresponding
// pointer-typed BindingResolved field is non-nil per F.7.17 L9. The -p
// argument is intentionally empty in this droplet — F.7.17.5 dispatcher
// wiring will route the assembled prompt body through a follow-up
// extension to BundlePaths or the BuildCommand contract. Until then
// claudeAdapter emits the flag with an empty argument so the always-on
// shape in §3 is preserved verbatim.
func assembleArgv(binding dispatcher.BindingResolved, paths dispatcher.BundlePaths) []string {
	pluginDir := filepath.Join(paths.Root, pluginSubdir)
	mcpConfigPath := filepath.Join(pluginDir, ".mcp.json")
	settingsPath := filepath.Join(pluginDir, "settings.json")

	// Cap initial slice at a generous upper bound (~32) so we avoid mid-build
	// reallocs without over-reserving. Slack covers all conditional flags.
	argv := make([]string, 0, 32)

	// argv[0] is the binary name. exec.Command resolves it via PATH from
	// the spawn's cmd.Env (set by assembleEnv) at exec time.
	argv = append(argv, claudeBinaryName)

	// Always-on flags first, in the §3 order. Order is not strictly
	// claude-required (claude accepts flags in any order), but stability
	// matters for test snapshotting and log readability.
	argv = append(argv,
		"--bare",
		"--plugin-dir", pluginDir,
		"--agent", binding.AgentName,
		"--system-prompt-file", paths.SystemPromptPath,
	)

	// Optional system-append file: emit ONLY when the bundle has one
	// configured (BundlePaths.SystemAppendPath documented as empty when
	// no append file is configured per cli_adapter.go).
	if strings.TrimSpace(paths.SystemAppendPath) != "" {
		argv = append(argv, "--append-system-prompt-file", paths.SystemAppendPath)
	}

	argv = append(argv,
		"--settings", settingsPath,
		"--setting-sources", "",
		"--strict-mcp-config",
		"--permission-mode", "acceptEdits",
		"--output-format", "stream-json",
		"--verbose",
		"--no-session-persistence",
		"--exclude-dynamic-system-prompt-sections",
		"--mcp-config", mcpConfigPath,
	)

	// Conditional pointer-typed flags. Emit-only-on-non-nil is the F.7.17 L9
	// contract: nil means "the lower-priority layer is authoritative" and
	// the adapter MUST NOT synthesize a value. claude's CLI defaults apply
	// when the flag is omitted entirely.
	if binding.MaxBudgetUSD != nil {
		argv = append(argv, "--max-budget-usd", formatBudget(*binding.MaxBudgetUSD))
	}
	if binding.MaxTurns != nil {
		argv = append(argv, "--max-turns", strconv.Itoa(*binding.MaxTurns))
	}
	if binding.Effort != nil {
		argv = append(argv, "--effort", *binding.Effort)
	}
	if binding.Model != nil {
		argv = append(argv, "--model", *binding.Model)
	}

	// Tools flag: claude expects a single comma-separated argument. Nil
	// slice = "use CLI default"; non-nil empty slice = "deny all" if claude
	// supports that semantic (we honor the BindingResolved.Tools doc-comment
	// faithfully — non-nil empty emits an empty `--tools ""`).
	if binding.Tools != nil {
		argv = append(argv, "--tools", strings.Join(binding.Tools, ","))
	}

	// -p prompt: empty placeholder in this droplet; F.7.17.5 wires the real
	// prompt source. The flag itself ships always-on per §3 so the argv
	// shape stays stable across the wiring landing.
	argv = append(argv, "-p", "")

	return argv
}

// formatBudget renders a *float64 budget value for the --max-budget-usd
// CLI flag. Whole values render without a trailing decimal ("5", not
// "5.00"); fractional values render with the minimum digits required to
// round-trip via strconv.FormatFloat. Lifted verbatim from spawn.go's
// existing formatBudget so argv parity with the 4a.19 stub holds.
func formatBudget(v float64) string {
	if v == float64(int64(v)) {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}
