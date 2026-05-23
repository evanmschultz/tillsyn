package dispatcher

// MCPServerConfig is the per-server config carried on BindingResolved.MCPServers.
// Promoted from cli_codex per HV3 Option 2 sign-off (2026-05-22) so the type
// is shared across CLI adapters that emit MCP-server argv (codex today; future
// claude `--mcp-server` plumbing tomorrow).
type MCPServerConfig struct {
	// Command is the absolute or PATH-relative command name (e.g., "till").
	Command string
	// Args is the command-line arguments passed to the command
	// (e.g., []string{"mcp"}).
	Args []string
	// Tools is the list of MCP tool names the server exposes
	// (e.g., []string{"till.action_item", "till.comment"}).
	// Each tool gets a per-tool approval_mode="approve" entry.
	Tools []string
}
