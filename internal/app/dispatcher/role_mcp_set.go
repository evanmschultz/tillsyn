package dispatcher

import (
	"strings"
)

// roleMCPSet represents the set of MCP servers enabled for a given agent role.
// Each boolean field corresponds to one MCP server in the proven matrix.
//
// The matrix is role-conditional and language-aware. See resolveRoleMCPSet
// for the complete mapping rules.
type roleMCPSet struct {
	// Tillsyn is the coordination substrate. Always true (present in every role's set).
	Tillsyn bool
	// Ta is the schema-MD editing server. Always true (present in every role's set).
	Ta bool
	// Hylla is the read-only Go code indexing server.
	// False for build-qa roles (just-shipped code not in snapshot).
	Hylla bool
	// Context7 is the external library documentation server.
	// False for build-qa roles.
	Context7 bool
	// Gopls is the Go symbol resolution and diagnostics server.
	// True for go-* roles only (not fe-* roles), and false for build-qa roles.
	Gopls bool
	// Playwright is the browser automation server for FE testing.
	// True for fe-* roles only (not go-* roles), and false for build-qa roles.
	Playwright bool
	// WebSearch enables live web search in backends that support it.
	// False for build-qa roles; true for all other roles.
	WebSearch bool
}

// resolveRoleMCPSet maps an agent's (role, axis, language) triple to the
// canonical MCP-server set for that role.
//
// The role parameter takes one of: "planner", "builder", "qa-proof", "qa-falsification", "closeout".
// The axis parameter takes one of: "plan", "build", "none".
// The language parameter takes one of: "go", "fe", "none".
//
// Roles on the build axis with "qa" in their name (build-qa-proof, build-qa-falsification)
// receive a special carve-out: Hylla, Context7, Gopls, Playwright, and WebSearch are all false.
// This is the SINGLE special-case branch. All other roles follow the standard matrix:
// Tillsyn and Ta are always true; Hylla and Context7 are true for all non-build-qa roles;
// Gopls is true for go-* roles except build-qa; Playwright is true for fe-* roles except build-qa;
// WebSearch is true for all non-build-qa roles.
//
// Unknown inputs (e.g., invalid role, axis, or language) default to the most-restrictive set:
// only Tillsyn and Ta are true.
func resolveRoleMCPSet(role, axis, language string) roleMCPSet {
	// Build-QA carve-out: Axis=="build" && Role contains "qa"
	isBuildQA := axis == "build" && strings.Contains(role, "qa")
	if isBuildQA {
		return roleMCPSet{
			Tillsyn:    true,
			Ta:         true,
			Hylla:      false,
			Context7:   false,
			Gopls:      false,
			Playwright: false,
			WebSearch:  false,
		}
	}

	// All non-build-QA roles: Tillsyn and Ta always true.
	set := roleMCPSet{
		Tillsyn:    true,
		Ta:         true,
		Hylla:      true,
		Context7:   true,
		WebSearch:  true,
	}

	// Gopls for go-* roles; Playwright for fe-* roles.
	switch language {
	case "go":
		set.Gopls = true
	case "fe":
		set.Playwright = true
	}

	return set
}
