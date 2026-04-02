package domain

import "strings"

// normalizeCoordinationRoleLabel canonicalizes coordination-facing role labels and common aliases.
func normalizeCoordinationRoleLabel(raw string) string {
	role := strings.TrimSpace(strings.ToLower(raw))
	switch role {
	case "dev":
		return "builder"
	case "researcher":
		return "research"
	default:
		return role
	}
}
