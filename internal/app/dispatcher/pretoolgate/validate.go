package pretoolgate

import (
	"errors"
	"fmt"
)

// ValidateRoleGate validates that a RoleGate complies with the AGENT_SANDBOX_SPEC
// §3 (channel-mismatch rules) and §4 (per-role grant matrix). It returns nil if
// the gate is valid, or a wrapped sentinel error if validation fails.
//
// The validator enforces four fail-loud cases (per AGENT_SANDBOX_SPEC):
//
//  1. codex role given Edit but no read-only spec — codex is always read-only
//     (never edits; writable_dirs is inapplicable) → ErrCodexReadOnly.
//  2. OAuth role dispatched as subprocess — OAuth requires the built-in
//     Agent tool (cannot use headless API-key tier) → ErrOAuthMustUseBuiltin.
//  3. build-qa-proof or build-qa-falsification role granted hylla MCP —
//     build-qa roles have no hylla read (just-shipped code not in snapshot)
//     → ErrForbiddenMCP.
//  4. non-editing role (planning, plan-qa-*, build-qa-*, closeout) with
//     nil-typed Edit (inapplicable) — must declare Edit:[] (present-empty,
//     read-only) → ErrNonEditingRoleNilEdit. Also: non-editing role with
//     non-empty Edit slice (attempted file edits on read-only role)
//     → ErrNonEditingRoleHasEdit.
//
// Calls errors.Is to detect sentinel errors; UX messages name the offending
// role and field (mirroring internal/templates/load.go pattern).
func ValidateRoleGate(rg RoleGate) error {
	// Case 1: codex + Edit non-empty → codex is read-only, no edits allowed.
	if rg.CLIKind == "codex" && len(rg.Spec.Edit) > 0 {
		return fmt.Errorf("%w: role %q (codex) declares edit files but codex is read-only; edit must be absent or empty", ErrCodexReadOnly, rg.Role)
	}

	// Case 2: OAuth + subprocess channel → OAuth requires built-in Agent tool.
	if rg.OAuth && rg.Channel == ChannelSubprocess {
		return fmt.Errorf("%w: role %q is OAuth but channel is %q; OAuth roles must use %q", ErrOAuthMustUseBuiltin, rg.Role, rg.Channel, ChannelBuiltin)
	}

	// Case 3: build-qa-proof / build-qa-falsification with hylla grant →
	// these roles forbid hylla MCP (just-shipped code not in snapshot).
	// Exact role name match, NOT substring (avoid catching plan-qa-*).
	if rg.Role == "build-qa-proof" || rg.Role == "build-qa-falsification" {
		if containsGrant(rg.MCPGrants, "hylla") {
			return fmt.Errorf("%w: role %q (build-qa) cannot access hylla MCP; just-shipped code not in snapshot", ErrForbiddenMCP, rg.Role)
		}
	}

	// Case 4: non-editing roles must have Edit declared as present-empty ([]),
	// not nil (inapplicable). Also reject non-empty Edit on read-only roles.
	if isNonEditingRole(rg.Role) {
		if rg.Spec.Edit == nil {
			return fmt.Errorf("%w: role %q (read-only) must declare Edit:[] (present-empty); nil indicates inapplicable spec", ErrNonEditingRoleNilEdit, rg.Role)
		}
		if len(rg.Spec.Edit) > 0 {
			return fmt.Errorf("%w: role %q (read-only) declares edit files but may only edit:[] (present-empty)", ErrNonEditingRoleHasEdit, rg.Role)
		}
	}

	return nil
}

// nonEditingRoles is the closed set of roles that do not edit code.
// These roles must declare Edit:[] (present-empty), never a populated slice.
var nonEditingRoles = map[string]struct{}{
	"planning":              {},
	"plan-qa-proof":         {},
	"plan-qa-falsification": {},
	"build-qa-proof":        {},
	"build-qa-falsification": {},
	"closeout":              {},
}

// isNonEditingRole reports whether role is a member of the closed
// nonEditingRoles set.
func isNonEditingRole(role string) bool {
	_, ok := nonEditingRoles[role]
	return ok
}

// containsGrant reports whether grant is a member of the grants slice
// (case-sensitive exact match).
func containsGrant(grants []string, grant string) bool {
	for _, g := range grants {
		if g == grant {
			return true
		}
	}
	return false
}

// ErrCodexReadOnly is returned when a codex role attempts to declare file
// edits. Codex is always read-only; writable_dirs for codex is the correct
// primitive (if needed at all).
var ErrCodexReadOnly = errors.New("codex role declared as read-only; edit files are forbidden")

// ErrOAuthMustUseBuiltin is returned when an OAuth/subscription role is
// dispatched on a non-builtin channel (e.g., subprocess / API-key headless).
// OAuth roles require the built-in Agent tool.
var ErrOAuthMustUseBuiltin = errors.New("OAuth role requires built-in Agent tool channel")

// ErrForbiddenMCP is returned when a build-qa role requests an MCP server
// that is forbidden (e.g., hylla, which requires code to be in the Hylla
// snapshot).
var ErrForbiddenMCP = errors.New("role cannot access this MCP server")

// ErrNonEditingRoleNilEdit is returned when a read-only role (planning,
// plan-qa-*, build-qa-*, closeout) has a nil-typed Edit field (inapplicable
// spec). Read-only roles must declare Edit:[] (present-empty).
var ErrNonEditingRoleNilEdit = errors.New("read-only role must declare Edit:[] (present-empty), not nil")

// ErrNonEditingRoleHasEdit is returned when a read-only role declares a
// non-empty Edit slice (attempted file edits on a read-only role).
var ErrNonEditingRoleHasEdit = errors.New("read-only role cannot declare file edits; use Edit:[] (present-empty)")
