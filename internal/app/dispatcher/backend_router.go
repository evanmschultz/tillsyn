// Package dispatcher — backend_router.go routes agent personas to CLI kinds.
//
// BackendRouter selects the CLI kind (client identifier) for a given agent
// persona, given the resolved template + agents.toml registry. At runtime,
// exactly one of (template.Client, preset.Client) is non-empty per the
// boot-time ValidateAgentsTemplateClientConflict validator — D5's job is
// simply to pick the single non-empty value.
package dispatcher

import (
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/evanmschultz/tillsyn/internal/app/dispatcher/pretoolgate"
	"github.com/evanmschultz/tillsyn/internal/config"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// ErrUnroutablePersona is returned when both the template and preset have
// empty Client fields — no CLI kind is configured for this persona, making
// it impossible to route to a backend.
var ErrUnroutablePersona = errors.New("dispatcher: unroutable persona — no cli_kind configured")

// ErrEnvFromShellMissingShellVar signals that an EnvFromShell entry names a
// shell-side variable (the map value) that is unset in the orchestrator's
// environment. Fail-loud per project hard-rule "parity and clarity, never
// silent failures."
var ErrEnvFromShellMissingShellVar = errors.New("dispatcher: EnvFromShell shell-side variable is unset")

// ResolvedTemplate carries the resolved template bindings for a given kind.
// Client holds the CLI kind (e.g. "claude", "codex") from the template layer.
type ResolvedTemplate struct {
	Client string
}

// BackendRouter selects the CLI kind (backend) for an agent persona given
// the resolved template + agents.toml preset registry. The router is keyed
// by persona name (e.g. "ta-go-builder").
//
// Boot-time validator ValidateAgentsTemplateClientConflict guarantees at
// most one non-empty Client between template and preset, so ResolveBackend
// performs a straightforward selection: pick the single non-empty value or
// error on dual-empty. No "preset wins" precedence — that would contradict
// the validator's PEER semantics.
type BackendRouter struct {
	registry *config.AgentsRegistry
	template ResolvedTemplate
}

// NewBackendRouter constructs a router for the given template and registry.
func NewBackendRouter(registry *config.AgentsRegistry, template ResolvedTemplate) *BackendRouter {
	return &BackendRouter{
		registry: registry,
		template: template,
	}
}

// ResolveBackend returns the CLI kind to use for the given persona name.
// Exactly one of (template.Client, preset.Client) is non-empty at runtime
// thanks to the boot-time validator. ResolveBackend picks the single non-empty
// value or returns ErrUnroutablePersona if both are empty.
//
// Arguments:
//   - personaName: the agent persona name (e.g. "ta-go-builder")
//   - group: the agents.toml group name (e.g. "go")
//   - kind: the kind identifier (e.g. "build")
//
// Returns the CLI kind string or ErrUnroutablePersona if no client is configured.
func (r *BackendRouter) ResolveBackend(personaName, group, kind string) (string, error) {
	// Resolve the preset from the registry.
	preset, err := config.Resolve(*r.registry, group, kind)
	if err != nil {
		// Resolve currently never returns non-nil error, but forward-compat
		// the check per the function's reserved error signature.
		return "", fmt.Errorf("%w: resolve preset: %w", ErrUnroutablePersona, err)
	}

	templateClient := r.template.Client
	presetClient := preset.Client

	// Count non-empty clients.
	templateEmpty := templateClient == ""
	presetEmpty := presetClient == ""

	switch {
	case templateEmpty && presetEmpty:
		// Both empty — unroutable.
		return "", fmt.Errorf("%w: persona %q (group %q, kind %q)", ErrUnroutablePersona, personaName, group, kind)
	case !templateEmpty && presetEmpty:
		// Template only.
		return templateClient, nil
	case templateEmpty && !presetEmpty:
		// Preset only.
		return presetClient, nil
	default:
		// Both non-empty. The boot-time validator guarantees they are equal.
		// Defense-in-depth: reject if they differ (validator should have caught this).
		if templateClient != presetClient {
			return "", fmt.Errorf("%w: persona %q has conflicting cli_kinds: template=%q preset=%q", ErrUnroutablePersona, personaName, templateClient, presetClient)
		}
		return templateClient, nil
	}
}

// ResolveMCPServers returns the per-spawn MCP server map for the given
// agent definition. A1 REPLACE: the role matrix (resolveRoleMCPSet) is
// AUTHORITATIVE; def.MCPServers frontmatter is NOT consulted.
//
// The function resolves the (Role, Axis, Language) triple from def,
// queries the canonical MCP-server set for that persona, and maps each
// enabled boolean to a concrete MCPServerConfig (command + args + tools).
// WebSearch is NOT an MCP server entry (backend flag only).
//
// Returns nil when def is nil.
//
// Future overrides (CLI/MCP/TUI per-spawn knobs) merge via BindingOverrides
// at the resolver step UPSTREAM of this router, NOT here. The router is
// the config-broker seam: it answers "what config does this item want?"
// without knowing about override layers above it.
func (r *BackendRouter) ResolveMCPServers(def *AgentDefinition) map[string]MCPServerConfig {
	if def == nil {
		return nil
	}

	// Resolve the role-canonical MCP set from def's (role, axis, language).
	mcp := resolveRoleMCPSet(def.Role, def.Axis, def.Language)

	// Map enabled booleans to MCPServerConfig entries.
	out := make(map[string]MCPServerConfig)

	if mcp.Tillsyn {
		out["tillsyn"] = MCPServerConfig{
			Command: "till",
			Args:    []string{"mcp"},
			Tools:   []string{"till.action_item", "till.comment", "till.attention_item", "till.handoff", "till.auth_request", "till.capability_lease", "till.capture_state", "till.get_instructions"},
		}
	}

	if mcp.Ta {
		out["ta"] = MCPServerConfig{
			Command: "ta",
			Args:    []string{"--project", "/Users/evanschultz/Documents/Code/hylla/tillsyn/main"},
			Tools:   []string{"ta.schema", "ta.list_sections", "ta.get", "ta.create", "ta.update", "ta.delete", "ta.search"},
		}
	}

	if mcp.Hylla {
		out["hylla"] = MCPServerConfig{
			Command: "/Users/evanschultz/go/bin/hylla",
			Args:    []string{"mcp"},
			Tools:   []string{"hylla.search", "hylla.node_full", "hylla.search_keyword", "hylla.refs_find", "hylla.graph_nav"},
		}
	}

	if mcp.Context7 {
		out["context7"] = MCPServerConfig{
			Command: "context7",
			Args:    []string{},
			Tools:   []string{"context7.resolve_library_id", "context7.query_docs"},
		}
	}

	if mcp.Gopls {
		out["gopls"] = MCPServerConfig{
			Command: "gopls",
			Args:    []string{"mcp"},
			Tools:   []string{"gopls.hover", "gopls.definition", "gopls.references", "gopls.diagnostics"},
		}
	}

	if mcp.Playwright {
		out["playwright"] = MCPServerConfig{
			Command: "playwright-mcp",
			Args:    []string{"--headless", "--isolated"},
			Tools:   []string{"playwright.browser_navigate", "playwright.browser_screenshot", "playwright.browser_snapshot"},
		}
	}

	// WebSearch is NOT an MCP server — it's a backend flag (e.g., `-c web_search="live"`).
	// mcp.WebSearch is not consulted here.

	return out
}

// ResolveWebSearch returns the WebSearch flag value for the given agent
// definition. The flag indicates whether the backend may emit live web-search
// flags (e.g., codex's `-c web_search="live"`). Build-QA roles always return
// false; all other roles return true. Returns false when def is nil.
//
// The function resolves the (Role, Axis, Language) triple from def, queries
// the canonical role-MCP-set for that persona, and returns the WebSearch
// boolean. The result is consumed by adapters at argv assembly time.
func (r *BackendRouter) ResolveWebSearch(def *AgentDefinition) bool {
	if def == nil {
		return false
	}

	// Resolve the role-canonical MCP set from def's (role, axis, language).
	mcp := resolveRoleMCPSet(def.Role, def.Axis, def.Language)
	return mcp.WebSearch
}

// ResolveEnvSet returns the per-spawn EnvSet + EnvFromShell-resolved Env
// entries for the given action item + (group, kind) pair. EnvSet is a
// cloned copy of preset.EnvSet (defensive). EnvFromShell carries
// "SPAWN_NAME=<value>" pairs after looking up the shell-side variable
// (preset.EnvFromShell map value) via os.LookupEnv. Missing shell-side
// variables return ErrEnvFromShellMissingShellVar wrapped with the offending
// spawn name + shell name.
//
// Semantics: preset.EnvFromShell is a MAP[string]string where KEY = spawn-side
// env var name, VALUE = orchestrator-side shell var name to read. Example:
//
//	OLLAMA_AUTH = "ANTHROPIC_AUTH_TOKEN"
//
// Means "set the spawn's OLLAMA_AUTH to the orchestrator's $ANTHROPIC_AUTH_TOKEN
// value."
//
// Future overrides (CLI/MCP/TUI per-spawn EnvSet/EnvFromShell knobs) merge
// via BindingOverrides at the resolver step UPSTREAM, not here. The router
// is the config-broker seam.
func (r *BackendRouter) ResolveEnvSet(_ domain.ActionItem, group, kind string) (map[string]string, []string, error) {
	preset, err := config.Resolve(*r.registry, group, kind)
	if err != nil {
		return nil, nil, fmt.Errorf("dispatcher: resolve preset for env: %w", err)
	}

	// Clone EnvSet defensively so caller mutations don't bleed back to the
	// preset.
	var envSet map[string]string
	if len(preset.EnvSet) > 0 {
		envSet = make(map[string]string, len(preset.EnvSet))
		for k, v := range preset.EnvSet {
			envSet[k] = v
		}
	}

	// Resolve EnvFromShell per asymmetric mapping. Sort keys for deterministic
	// ordering (test-stable; matches NIT-3-style discipline).
	var envFromShell []string
	if len(preset.EnvFromShell) > 0 {
		spawnNames := make([]string, 0, len(preset.EnvFromShell))
		for sn := range preset.EnvFromShell {
			spawnNames = append(spawnNames, sn)
		}
		sort.Strings(spawnNames)
		for _, spawnName := range spawnNames {
			shellName := preset.EnvFromShell[spawnName]
			val, ok := os.LookupEnv(shellName)
			if !ok {
				return nil, nil, fmt.Errorf("%w: spawn_name=%q shell_name=%q (agent persona expects $%s in orchestrator env)", ErrEnvFromShellMissingShellVar, spawnName, shellName, shellName)
			}
			envFromShell = append(envFromShell, spawnName+"="+val)
		}
	}

	return envSet, envFromShell, nil
}

// roleGateToGateSpec projects a RoleGate onto a GateSpec, enforcing
// read-only for codex roles. This is the pure projector function tested
// directly; the resolve-seam wiring (follow-up droplet, blocked on
// ResolveAgentPath + constructRoleGate) calls it with the agent-definition-
// derived RoleGate and assigns the result to BindingResolved.GateSpec.
//
// For codex roles, the returned GateSpec has WritableDirs=nil and Edit=nil
// (read-only). Other roles return nil (ungated).
func roleGateToGateSpec(rg *pretoolgate.RoleGate) *pretoolgate.GateSpec {
	if rg == nil {
		return nil
	}

	// Codex roles are read-only: zero out WritableDirs and Edit.
	// Note: RoleGate.CLIKind is a string, so compare to string value "codex".
	if rg.CLIKind == string(CLIKindCodex) {
		return &pretoolgate.GateSpec{
			WritableDirs: nil,
			Edit:         nil,
			BashDeny:     cloneStringSlice(rg.Spec.BashDeny),
			Network:      rg.Spec.Network,
		}
	}

	// Non-codex roles are ungated (nil).
	return nil
}
