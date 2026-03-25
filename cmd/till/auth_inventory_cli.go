package main

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"text/tabwriter"
	"time"
)

// writeAuthRequestListHuman renders auth requests as a stable human-readable table.
func writeAuthRequestListHuman(stdout io.Writer, requests []authRequestPayloadJSON) error {
	rows := append([]authRequestPayloadJSON(nil), requests...)
	slices.SortFunc(rows, compareAuthRequestsForCLI)
	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "AUTH REQUESTS"); err != nil {
		return fmt.Errorf("write auth request list header: %w", err)
	}
	if _, err := fmt.Fprintln(tw, "NAME\tREQUEST ID\tSTATE\tCLIENT\tREQUESTED PATH\tAPPROVED PATH\tREQUESTED BY\tREQUESTED TTL\tAPPROVED TTL\tRESULT SESSION"); err != nil {
		return fmt.Errorf("write auth request list columns: %w", err)
	}
	if len(rows) == 0 {
		if _, err := fmt.Fprintln(tw, "(none)\t-\t-\t-\t-\t-\t-\t-\t-\t-"); err != nil {
			return fmt.Errorf("write empty auth request row: %w", err)
		}
		return flushAuthInventoryTable(tw, "auth request list")
	}
	for _, request := range rows {
		if _, err := fmt.Fprintf(
			tw,
			"%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
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
		); err != nil {
			return fmt.Errorf("write auth request list row: %w", err)
		}
	}
	return flushAuthInventoryTable(tw, "auth request list")
}

// writeAuthRequestDetailHuman renders one auth request as a stable human-readable detail block.
func writeAuthRequestDetailHuman(stdout io.Writer, request authRequestPayloadJSON) error {
	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "AUTH REQUEST"); err != nil {
		return fmt.Errorf("write auth request detail header: %w", err)
	}
	rows := [][2]string{
		{"name", humanAuthPrincipalLabel(request.PrincipalName, request.PrincipalID, request.PrincipalRole)},
		{"request id", firstNonEmptyTrimmed(request.ID, "-")},
		{"state", firstNonEmptyTrimmed(request.State, "-")},
		{"principal type", firstNonEmptyTrimmed(request.PrincipalType, "-")},
		{"client", humanAuthClientLabel(request.ClientName, request.ClientID)},
		{"requested path", firstNonEmptyTrimmed(request.Path, "-")},
		{"approved path", firstNonEmptyTrimmed(request.ApprovedPath, "-")},
		{"requested ttl", humanAuthDurationLabel(request.RequestedSessionTTL)},
		{"approved ttl", humanAuthDurationLabel(request.ApprovedSessionTTL)},
		{"requested by", firstNonEmptyTrimmed(humanAuthActorLabel(request.RequestedByActor, request.RequestedByType), "-")},
		{"reason", firstNonEmptyTrimmed(request.Reason, "-")},
		{"issued session", firstNonEmptyTrimmed(request.IssuedSessionID, "-")},
		{"issued session expires", formatAuthOptionalTime(request.IssuedSessionExpiresAt)},
		{"resolved by", firstNonEmptyTrimmed(humanAuthActorLabel(request.ResolvedByActor, request.ResolvedByType), "-")},
		{"resolved at", formatAuthOptionalTime(request.ResolvedAt)},
		{"resolution note", firstNonEmptyTrimmed(request.ResolutionNote, "-")},
	}
	for _, row := range rows {
		if row[1] == "" {
			continue
		}
		if _, err := fmt.Fprintf(tw, "%s\t%s\n", row[0], row[1]); err != nil {
			return fmt.Errorf("write auth request detail row: %w", err)
		}
	}
	if err := tw.Flush(); err != nil {
		return fmt.Errorf("flush auth request detail: %w", err)
	}
	return nil
}

// writeAuthSessionListHuman renders auth sessions as a stable human-readable table.
func writeAuthSessionListHuman(stdout io.Writer, sessions []authSessionPayloadJSON) error {
	rows := append([]authSessionPayloadJSON(nil), sessions...)
	slices.SortFunc(rows, compareAuthSessionsForCLI)
	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "AUTH SESSIONS"); err != nil {
		return fmt.Errorf("write auth session list header: %w", err)
	}
	if _, err := fmt.Fprintln(tw, "NAME\tSESSION ID\tSTATE\tCLIENT\tPROJECT\tAPPROVED PATH\tEXPIRES\tREVOCATION"); err != nil {
		return fmt.Errorf("write auth session list columns: %w", err)
	}
	if len(rows) == 0 {
		if _, err := fmt.Fprintln(tw, "(none)\t-\t-\t-\t-\t-\t-\t-"); err != nil {
			return fmt.Errorf("write empty auth session row: %w", err)
		}
		return flushAuthInventoryTable(tw, "auth session list")
	}
	for _, session := range rows {
		if _, err := fmt.Fprintf(
			tw,
			"%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			humanAuthPrincipalLabel(session.PrincipalName, session.PrincipalID, session.PrincipalRole),
			firstNonEmptyTrimmed(session.SessionID, "-"),
			firstNonEmptyTrimmed(session.State, "-"),
			humanAuthClientLabel(session.ClientName, session.ClientID),
			firstNonEmptyTrimmed(session.ProjectID, "-"),
			firstNonEmptyTrimmed(session.ApprovedPath, "-"),
			formatAuthTime(session.ExpiresAt),
			firstNonEmptyTrimmed(session.RevocationReason, "-"),
		); err != nil {
			return fmt.Errorf("write auth session list row: %w", err)
		}
	}
	return flushAuthInventoryTable(tw, "auth session list")
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

// flushAuthInventoryTable flushes one tabwriter-backed inventory table.
func flushAuthInventoryTable(tw *tabwriter.Writer, context string) error {
	if err := tw.Flush(); err != nil {
		return fmt.Errorf("flush %s: %w", context, err)
	}
	return nil
}
