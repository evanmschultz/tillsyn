package domain

import (
	"regexp"
	"slices"
	"strings"
)

// Role represents the closed 9-value enum of action-item roles.
type Role string

// Built-in role values. String values are lowercase, hyphenated.
const (
	RoleBuilder         Role = "builder"
	RoleQAProof         Role = "qa-proof"
	RoleQAFalsification Role = "qa-falsification"
	RoleQAA11y          Role = "qa-a11y"
	RoleQAVisual        Role = "qa-visual"
	RoleDesign          Role = "design"
	RoleCommit          Role = "commit"
	RolePlanner         Role = "planner"
	RoleResearch        Role = "research"
)

// validRoles stores every member of the closed 9-value Role enum.
var validRoles = []Role{
	RoleBuilder,
	RoleQAProof,
	RoleQAFalsification,
	RoleQAA11y,
	RoleQAVisual,
	RoleDesign,
	RoleCommit,
	RolePlanner,
	RoleResearch,
}

// roleDescriptionRegex matches a line of the form `Role: <value>` where
// `<value>` is composed of lowercase ASCII letters, digits, and hyphens. The
// `(?m)` flag enables multiline mode so `^` and `$` anchor to line boundaries
// inside a multi-line description rather than just the string boundaries. The
// pattern is intentionally case-sensitive — a capitalized variant such as
// `Role: Builder` produces no match, matching the acceptance contract that
// only the canonical lowercase form is recognized.
//
// The character class `[a-z0-9-]+` includes digits because the closed Role
// enum contains values with digits (`qa-a11y`). Droplet 2.2's PLAN.md
// acceptance text wrote the class as `[a-z-]+` but also requires every one
// of the 9 enum values to round-trip; the class is widened minimally to
// satisfy both. Uppercase letters remain excluded so the case-sensitivity
// contract holds.
var roleDescriptionRegex = regexp.MustCompile(`(?m)^Role:\s*([a-z0-9-]+)\s*$`)

// IsValidRole reports whether role is a member of the closed Role enum.
// The empty string is considered invalid; callers that want to permit an
// optional / unset role should short-circuit on emptiness before calling
// IsValidRole.
func IsValidRole(role Role) bool {
	return slices.Contains(validRoles, Role(strings.TrimSpace(strings.ToLower(string(role)))))
}

// NormalizeRole canonicalizes a Role value by trimming surrounding whitespace
// and lowercasing the input. Empty input returns the empty string unchanged.
func NormalizeRole(role Role) Role {
	trimmed := strings.TrimSpace(string(role))
	if trimmed == "" {
		return ""
	}
	return Role(strings.ToLower(trimmed))
}

// ParseRoleFromDescription extracts a Role from a free-form action-item
// description by scanning for the first line of the form `Role: <value>`.
// The regex anchors to line boundaries via `(?m)`, so mid-paragraph
// occurrences of `Role:` are ignored.
//
// Return contract:
//   - No `Role:` line found → ("", nil).
//   - First matching line carries a value in the closed Role enum →
//     (Role, nil) where Role is the typed constant.
//   - First matching line carries a value that does not appear in the closed
//     enum → ("", ErrInvalidRole).
//
// Only the first match is consulted; subsequent `Role:` lines are not
// inspected, so the description's earliest declaration wins.
func ParseRoleFromDescription(desc string) (Role, error) {
	match := roleDescriptionRegex.FindStringSubmatch(desc)
	if match == nil {
		return "", nil
	}
	candidate := Role(match[1])
	if !IsValidRole(candidate) {
		return "", ErrInvalidRole
	}
	return candidate, nil
}
