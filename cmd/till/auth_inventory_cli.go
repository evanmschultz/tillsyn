package main

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"time"
)

// writeAuthRequestListHuman renders auth requests as a stable human-readable table.
func writeAuthRequestListHuman(stdout io.Writer, requests []authRequestPayloadJSON) error {
	rows := append([]authRequestPayloadJSON(nil), requests...)
	slices.SortFunc(rows, compareAuthRequestsForCLI)
	renderRows := make([][]string, 0, len(rows))
	for _, request := range rows {
		renderRows = append(renderRows, []string{
			humanAuthPrincipalLabel(request.PrincipalName, request.PrincipalID, request.PrincipalRole),
			firstNonEmptyTrimmed(request.ID, "-"),
			firstNonEmptyTrimmed(request.State, "-"),
			humanAuthClientLabel(request.ClientName, request.ClientID),
			firstNonEmptyTrimmed(request.Path, "-"),
			firstNonEmptyTrimmed(request.ApprovedPath, "-"),
			firstNonEmptyTrimmed(humanAuthActorLabel(request.RequestedByActor, request.RequestedByType), "-"),
			humanAuthDurationLabel(request.RequestedSessionTTL),
			humanAuthDurationLabel(request.ApprovedSessionTTL),
			firstNonEmptyTrimmed(request.IssuedSessionID, "-"),
		})
	}
	return writeCLITable(
		stdout,
		"Auth Requests",
		[]string{"NAME", "REQUEST ID", "STATE", "CLIENT", "REQUESTED PATH", "APPROVED PATH", "REQUESTED BY", "REQUESTED TTL", "APPROVED TTL", "RESULT SESSION"},
		renderRows,
		"No auth requests found.",
	)
}

// writeAuthRequestDetailHuman renders one auth request as a stable human-readable detail block.
func writeAuthRequestDetailHuman(stdout io.Writer, request authRequestPayloadJSON) error {
	return writeCLIKV(stdout, "Auth Request", authRequestDetailRows(request, false))
}

// writeAuthRequestResultHuman renders one auth request result block and includes issued credentials when present.
func writeAuthRequestResultHuman(stdout io.Writer, request authRequestPayloadJSON) error {
	return writeCLIKV(stdout, "Auth Request", authRequestDetailRows(request, true))
}

func authRequestDetailRows(request authRequestPayloadJSON, includeSecret bool) [][2]string {
	rows := [][2]string{
		{"name", humanAuthPrincipalLabel(request.PrincipalName, request.PrincipalID, request.PrincipalRole)},
		{"request id", firstNonEmptyTrimmed(request.ID, "-")},
		{"state", firstNonEmptyTrimmed(request.State, "-")},
		{"project", firstNonEmptyTrimmed(request.ProjectID, "-")},
		{"branch", firstNonEmptyTrimmed(request.BranchID, "-")},
		{"phases", renderAuthStringList(request.PhaseIDs)},
		{"scope", humanAuthScopeLabel(request.ProjectID, request.ScopeType, request.ScopeID)},
		{"principal type", firstNonEmptyTrimmed(request.PrincipalType, "-")},
		{"client", humanAuthClientLabel(request.ClientName, request.ClientID)},
		{"requested path", firstNonEmptyTrimmed(request.Path, "-")},
		{"approved path", firstNonEmptyTrimmed(request.ApprovedPath, "-")},
		{"requested ttl", humanAuthDurationLabel(request.RequestedSessionTTL)},
		{"approved ttl", humanAuthDurationLabel(request.ApprovedSessionTTL)},
		{"requested by", firstNonEmptyTrimmed(humanAuthActorLabel(request.RequestedByActor, request.RequestedByType), "-")},
		{"reason", firstNonEmptyTrimmed(request.Reason, "-")},
		{"has continuation", yesNo(request.HasContinuation)},
		{"created at", formatAuthTime(request.CreatedAt)},
		{"expires at", formatAuthTime(request.ExpiresAt)},
		{"issued session", firstNonEmptyTrimmed(request.IssuedSessionID, "-")},
		{"issued session expires", formatAuthOptionalTime(request.IssuedSessionExpiresAt)},
		{"resolved by", firstNonEmptyTrimmed(humanAuthActorLabel(request.ResolvedByActor, request.ResolvedByType), "-")},
		{"resolved at", formatAuthOptionalTime(request.ResolvedAt)},
		{"resolution note", firstNonEmptyTrimmed(request.ResolutionNote, "-")},
	}
	if includeSecret {
		if secret := strings.TrimSpace(request.IssuedSessionSecret); secret != "" {
			rows = append(rows, [2]string{"issued session secret", secret})
		}
	}
	return rows
}

// writeAuthSessionDetailHuman renders one auth session as a stable human-readable detail block.
func writeAuthSessionDetailHuman(stdout io.Writer, session authSessionPayloadJSON, sessionSecret string) error {
	rows := [][2]string{
		{"name", humanAuthPrincipalLabel(session.PrincipalName, session.PrincipalID, session.PrincipalRole)},
		{"session id", firstNonEmptyTrimmed(session.SessionID, "-")},
		{"state", firstNonEmptyTrimmed(session.State, "-")},
		{"project", firstNonEmptyTrimmed(session.ProjectID, "-")},
		{"auth request", firstNonEmptyTrimmed(session.AuthRequestID, "-")},
		{"principal type", firstNonEmptyTrimmed(session.PrincipalType, "-")},
		{"client", humanAuthClientLabel(session.ClientName, session.ClientID)},
		{"client type", firstNonEmptyTrimmed(session.ClientType, "-")},
		{"approved path", firstNonEmptyTrimmed(session.ApprovedPath, "-")},
		{"expires", formatAuthTime(session.ExpiresAt)},
		{"revoked at", formatAuthOptionalTime(session.RevokedAt)},
		{"revocation reason", firstNonEmptyTrimmed(session.RevocationReason, "-")},
	}
	if secret := strings.TrimSpace(sessionSecret); secret != "" {
		rows = append(rows, [2]string{"session secret", secret})
	}
	return writeCLIKV(stdout, "Auth Session", rows)
}

// writeAuthSessionListHuman renders auth sessions as a stable human-readable table.
func writeAuthSessionListHuman(stdout io.Writer, sessions []authSessionPayloadJSON) error {
	rows := append([]authSessionPayloadJSON(nil), sessions...)
	slices.SortFunc(rows, compareAuthSessionsForCLI)
	renderRows := make([][]string, 0, len(rows))
	for _, session := range rows {
		renderRows = append(renderRows, []string{
			humanAuthPrincipalLabel(session.PrincipalName, session.PrincipalID, session.PrincipalRole),
			firstNonEmptyTrimmed(session.SessionID, "-"),
			firstNonEmptyTrimmed(session.State, "-"),
			humanAuthClientLabel(session.ClientName, session.ClientID),
			firstNonEmptyTrimmed(session.ProjectID, "-"),
			firstNonEmptyTrimmed(session.ApprovedPath, "-"),
			formatAuthTime(session.ExpiresAt),
			firstNonEmptyTrimmed(session.RevocationReason, "-"),
		})
	}
	return writeCLITable(
		stdout,
		"Auth Sessions",
		[]string{"NAME", "SESSION ID", "STATE", "CLIENT", "PROJECT", "APPROVED PATH", "EXPIRES", "REVOCATION"},
		renderRows,
		"No auth sessions found.",
	)
}

// compareAuthRequestsForCLI sorts auth requests by operator-visible name, then id.
func compareAuthRequestsForCLI(a, b authRequestPayloadJSON) int {
	if cmp := strings.Compare(strings.ToLower(humanAuthPrincipalLabel(a.PrincipalName, a.PrincipalID, a.PrincipalRole)), strings.ToLower(humanAuthPrincipalLabel(b.PrincipalName, b.PrincipalID, b.PrincipalRole))); cmp != 0 {
		return cmp
	}
	if cmp := strings.Compare(strings.ToLower(a.ID), strings.ToLower(b.ID)); cmp != 0 {
		return cmp
	}
	return strings.Compare(strings.ToLower(a.ClientID), strings.ToLower(b.ClientID))
}

// compareAuthSessionsForCLI sorts auth sessions by operator-visible name, then session id.
func compareAuthSessionsForCLI(a, b authSessionPayloadJSON) int {
	if cmp := strings.Compare(strings.ToLower(humanAuthPrincipalLabel(a.PrincipalName, a.PrincipalID, a.PrincipalRole)), strings.ToLower(humanAuthPrincipalLabel(b.PrincipalName, b.PrincipalID, b.PrincipalRole))); cmp != 0 {
		return cmp
	}
	if cmp := strings.Compare(strings.ToLower(a.SessionID), strings.ToLower(b.SessionID)); cmp != 0 {
		return cmp
	}
	return strings.Compare(strings.ToLower(a.ClientID), strings.ToLower(b.ClientID))
}

// humanAuthPrincipalLabel renders one principal name with role suffix when available.
func humanAuthPrincipalLabel(name, id, role string) string {
	label := humanAuthNameWithSecondaryID(name, id)
	role = strings.TrimSpace(role)
	if role == "" {
		return label
	}
	if label == "" {
		return role
	}
	return label + " • " + role
}

// humanAuthNameWithSecondaryID keeps a friendly name primary while retaining the identifier for disambiguation.
func humanAuthNameWithSecondaryID(name, id string) string {
	name = strings.TrimSpace(name)
	id = strings.TrimSpace(id)
	switch {
	case name == "":
		return id
	case id == "" || name == id || strings.Contains(name, id):
		return name
	default:
		return name + " [" + id + "]"
	}
}

// humanAuthClientLabel renders one client name with an id fallback.
func humanAuthClientLabel(name, id string) string {
	return firstNonEmptyTrimmed(name, id)
}

// humanAuthActorLabel renders one actor with its type when available.
func humanAuthActorLabel(actorID string, actorType string) string {
	actorID = strings.TrimSpace(actorID)
	actorType = strings.TrimSpace(actorType)
	if actorID == "" && actorType == "" {
		return ""
	}
	if actorID == "" {
		return actorType
	}
	if actorType == "" {
		return actorID
	}
	return actorID + " (" + actorType + ")"
}

// humanAuthDurationLabel renders one duration string in compact human-friendly form.
func humanAuthDurationLabel(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "-" {
		return "-"
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return raw
	}
	switch {
	case d%time.Hour == 0:
		return fmt.Sprintf("%dh", int(d/time.Hour))
	case d%time.Minute == 0:
		return fmt.Sprintf("%dm", int(d/time.Minute))
	default:
		return d.Round(time.Second).String()
	}
}

// humanAuthScopeLabel renders one auth scope with project fallback for operator output.
func humanAuthScopeLabel(projectID, scopeType, scopeID string) string {
	scopeType = strings.TrimSpace(scopeType)
	scopeID = strings.TrimSpace(scopeID)
	projectID = strings.TrimSpace(projectID)
	switch {
	case scopeType == "":
		return firstNonEmptyTrimmed(projectID, scopeID, "-")
	case scopeType == "project":
		return "project/" + firstNonEmptyTrimmed(projectID, "-")
	case scopeID == "":
		return scopeType
	default:
		return scopeType + "/" + scopeID
	}
}

// renderAuthStringList renders one stable comma-separated list or a fallback dash.
func renderAuthStringList(values []string) string {
	if len(values) == 0 {
		return "-"
	}
	ordered := append([]string(nil), values...)
	slices.SortFunc(ordered, func(a, b string) int {
		return strings.Compare(strings.ToLower(strings.TrimSpace(a)), strings.ToLower(strings.TrimSpace(b)))
	})
	parts := make([]string, 0, len(ordered))
	for _, value := range ordered {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		parts = append(parts, value)
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, ", ")
}

// formatAuthTime renders one timestamp in UTC RFC3339 form.
func formatAuthTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.UTC().Format(time.RFC3339)
}

// formatAuthOptionalTime renders one optional timestamp in UTC RFC3339 form.
func formatAuthOptionalTime(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return formatAuthTime(*t)
}

// firstNonEmptyTrimmed returns the first non-empty trimmed string in order.
func firstNonEmptyTrimmed(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
