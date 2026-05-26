package pretoolgate

// Channel is the dispatch channel a role's gate is enforced on. Local to
// pretoolgate to keep the package leaf-pure (no parent-dispatcher import).
type Channel string

const (
	// ChannelBuiltin is the claude built-in Agent tool (OAuth/subscription roles).
	ChannelBuiltin Channel = "builtin"
	// ChannelSubprocess is the headless channel (claude -p --bare, API-key tier, or codex exec).
	ChannelSubprocess Channel = "subprocess"
)

// RoleGate is the input type the §4 per-role matrix validator (D2B,
// ValidateRoleGate) inspects and the TOML decode (D3, DecodeGateConfig)
// populates. It carries all the information needed to validate a role's
// dispatch configuration against the multi-backend sandboxing matrix.
//
// Scope boundary (D2 droplet constraint): RoleGate carries role × CLIKind ×
// Channel × OAuth × MCPGrants × embedded GateSpec. The validator (D2B) checks
// the forbidden-grant rules (hylla on build-qa), channel/oauth mismatches, and
// edit-vs-writable_dirs mismatches. Language-conditional positive MCP grants
// (gopls only on go, playwright only on fe) are resolved at MCP-INJECT.1's
// resolveRoleMCPSet; this type does NOT discriminate language — that is the
// resolver's job.
//
// CLIKind mirrors dispatcher.CLIKind ("claude" or "codex") as a plain string
// field to avoid importing the parent dispatcher package (which consumes
// pretoolgate via A's Decide). The string values are the same as the upstream
// constants at internal/app/dispatcher/cli_adapter.go.
//
// Spec embeds the FND.1 GateSpec contract (file/dir/bash/network); it is
// not redefined here.
type RoleGate struct {
	// Role is the agent role identifier (e.g. "planning", "plan-qa-proof",
	// "builder", "build-qa-falsification", "closeout").
	Role string

	// CLIKind is the dispatch backend ("claude" or "codex"). Plain string,
	// mirroring dispatcher.CLIKind values without importing dispatcher
	// (avoids parent-package import cycle).
	CLIKind string

	// Channel is the dispatch channel (builtin for OAuth/subscription roles
	// on the built-in Agent tool; subprocess for API-key headless and codex exec).
	Channel Channel

	// OAuth is true for OAuth/subscription roles that require the built-in
	// Agent tool; false for API-key (headless) and codex roles.
	OAuth bool

	// MCPGrants is the per-role MCP server grant list (e.g. ["hylla",
	// "context7", "gopls"]). The validator (D2B) checks the forbidden-grant
	// rules (hylla forbidden on build-qa); language-conditional positive
	// inclusion (gopls only on go, playwright only on fe) is resolved at
	// MCP-INJECT.1's resolveRoleMCPSet.
	MCPGrants []string

	// Spec is the embedded FND.1 contract defining the file/dir/bash/network
	// grants and denials.
	Spec GateSpec
}
