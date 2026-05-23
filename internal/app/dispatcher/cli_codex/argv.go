package cli_codex

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/log"

	"github.com/evanmschultz/tillsyn/internal/app/dispatcher"
)

// assembleArgv returns the full argv slice (including argv[0] = "codex")
// the dispatcher passes to exec.Command for one codex spawn. Per F.7.17
// REV-1 the binary name is hardcoded — no Command override path.
//
// Argv shape (per OQ1 codex exec --help):
//
//	codex exec --json --ephemeral --sandbox workspace-write \
//	  --skip-git-repo-check -C <paths.Root> \
//	  [-m <model>] \
//	  [-c model_reasoning_effort=<effort>] \
//	  [-c mcp_servers.<server>=<inline-toml>]
//
// The positional [PROMPT] argument is intentionally omitted — the caller
// (BuildCommand) feeds the system prompt via cmd.Stdin instead. codex reads
// its prompt from stdin when no positional argument is provided (OQ1:
// "If not provided as an argument (or if `-` is used), instructions are read
// from stdin"). This mirrors cli_claude's system-prompt-file pattern.
//
// Conditional flags (-m, -c model_reasoning_effort, -c mcp_servers.*) emit
// ONLY when the corresponding BindingResolved field is non-nil per F.7.17 L9.
// nil means "the lower-priority layer is authoritative"; codex's CLI defaults
// apply when the flag is omitted entirely.
//
// MCP server injection (Drop 4d D2): per-spawn -c flags inject inline-TOML
// configuration for each MCP server the agent needs. Per the canonical
// reference ~/.claude/codex-mcp-dispatch-tool-conversion.md:
//
//   - Each server gets one -c flag with the key `mcp_servers.<server-name>`.
//   - The value is an inline TOML table with `command`, `args`, and `tools`.
//   - Tool names with dots MUST be quoted (e.g., "till.action_item").
//   - Each tool entry uses `approval_mode="approve"` per-tool (not shared default).
//   - Auth env is passed via binding.Env or process inheritance.
//
// Note: paths.Root is passed via -C so codex's working root is the bundle
// directory. Adapters compute their own CLI-specific subdirs under Root per
// F.7.17 L13; codex's bundle layout (if needed) is computed here or in future
// D3/D4 droplets.
func assembleArgv(binding dispatcher.BindingResolved, paths dispatcher.BundlePaths) []string {
	// Cap initial slice at a reasonable upper bound so we avoid mid-build
	// reallocs without over-reserving. Slack covers conditional flags.
	argv := make([]string, 0, 32)

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
		"--sandbox", "workspace-write",
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

	// MCP server injection: for each MCP server the agent's definition declares,
	// build an inline-TOML configuration via -c flag. Drop 4d D1 populates
	// binding.MCPServers; D2.5 provides the tool-name conversion table.
	// Per the conversion doc, each server gets:
	//
	//   -c mcp_servers.{server-name}={command=..., args=[...], tools={...}}
	//
	// The tools block uses per-tool approval_mode="approve" entries with
	// quoted names for tools that have dots (e.g., "till.action_item").
	//
	// Server names are iterated in sorted (alphabetical) order for deterministic argv.
	// Server names containing dots are rejected (skipped with a warning) because
	// dots parse ambiguously in TOML key-path syntax (mcp_servers.my.server is
	// parsed as nested table my > server, not flat key my.server).
	if binding.MCPServers != nil {
		// Collect and sort server names for deterministic iteration.
		serverNames := make([]string, 0, len(binding.MCPServers))
		for serverName := range binding.MCPServers {
			serverNames = append(serverNames, serverName)
		}
		sort.Strings(serverNames)

		// Iterate in sorted order, rejecting dotted names.
		for _, serverName := range serverNames {
			// Reject server names containing dots (TOML parse ambiguity).
			if strings.Contains(serverName, ".") {
				log.Warn("cli_codex: MCP server name contains dot; skipping (TOML key-path ambiguity)",
					"server_name", serverName,
				)
				continue
			}
			config := binding.MCPServers[serverName]
			inline := buildMCPServerConfig(serverName, config)
			argv = append(argv, "-c", inline)
		}
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

// buildMCPServerConfig constructs a single -c mcp_servers.<server-name>=... inline-TOML value.
// Per Drop 4d D2 and the canonical reference ~/.claude/codex-mcp-dispatch-tool-conversion.md,
// the format is:
//
//	mcp_servers.<server-name>={command=<cmd>, args=[<args>], tools={<tool-entries>}}
//
// where each tool entry is `"<tool-name>"={approval_mode="approve"}`.
// Tool names with dots MUST be quoted.
func buildMCPServerConfig(serverName string, config dispatcher.MCPServerConfig) string {
	// Build the args array: args=[<comma-sep quoted strings>]
	var argsStr string
	if len(config.Args) > 0 {
		quoted := make([]string, len(config.Args))
		for i, arg := range config.Args {
			quoted[i] = fmt.Sprintf("%q", arg)
		}
		argsStr = "[" + strings.Join(quoted, ",") + "]"
	} else {
		argsStr = "[]"
	}

	// Build the tools block: tools={<tool-entries>}
	// Each tool gets an entry: "<tool-name>"={approval_mode="approve"}
	// Tool names with dots MUST be quoted unconditionally.
	var toolsStr string
	if len(config.Tools) > 0 {
		toolEntries := make([]string, len(config.Tools))
		for i, toolName := range config.Tools {
			// Always quote tool names to handle dots safely.
			toolEntries[i] = fmt.Sprintf("%q={approval_mode=\"approve\"}", toolName)
		}
		toolsStr = "{" + strings.Join(toolEntries, ",") + "}"
	} else {
		toolsStr = "{}"
	}

	// Assemble the full inline-TOML value:
	// mcp_servers.<server-name>={command="...", args=[...], tools={...}}
	return fmt.Sprintf("mcp_servers.%s={command=%q,args=%s,tools=%s}", serverName, config.Command, argsStr, toolsStr)
}
