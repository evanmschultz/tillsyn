package main

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/hylla/tillsyn/internal/domain"
)

// writeCoordinationLeaseList renders one deterministic human-readable lease table.
func writeCoordinationLeaseList(stdout io.Writer, now time.Time, leases []domain.CapabilityLease) error {
	ordered := append([]domain.CapabilityLease(nil), leases...)
	slices.SortFunc(ordered, compareCoordinationLeasesForCLI)
	rows := make([][]string, 0, len(ordered))
	for _, lease := range ordered {
		rows = append(rows, []string{
			coordinationLeaseAgentLabel(lease),
			firstNonEmptyTrimmed(string(lease.Role), "-"),
			firstNonEmptyTrimmed(lease.ProjectID, "-"),
			coordinationLeaseScopeLabel(lease),
			coordinationLeaseStatusAt(lease, now),
			firstNonEmptyTrimmed(lease.InstanceID, lease.LeaseToken, "-"),
			lease.ExpiresAt.UTC().Format(time.RFC3339),
		})
	}
	printer := newCLIPrinter(stdout)
	if len(rows) == 0 {
		if err := writeCLIPanelWithPrinter(printer, "Capability Leases", "No capability leases found.", ""); err != nil {
			return fmt.Errorf("write coordination lease empty state: %w", err)
		}
		return nil
	}
	if err := writeCLITableWithPrinter(
		printer,
		"Capability Leases",
		[]string{"AGENT", "ROLE", "PROJECT", "SCOPE", "STATUS", "ID", "EXPIRES"},
		rows,
		"No capability leases found.",
	); err != nil {
		return fmt.Errorf("write coordination lease list: %w", err)
	}
	return nil
}

// renderCoordinationLeaseListAt renders one deterministic human-readable lease table.
func renderCoordinationLeaseListAt(now time.Time, leases []domain.CapabilityLease) string {
	ordered := append([]domain.CapabilityLease(nil), leases...)
	slices.SortFunc(ordered, compareCoordinationLeasesForCLI)
	rows := make([][]string, 0, len(ordered))
	for _, lease := range ordered {
		rows = append(rows, []string{
			coordinationLeaseAgentLabel(lease),
			firstNonEmptyTrimmed(string(lease.Role), "-"),
			firstNonEmptyTrimmed(lease.ProjectID, "-"),
			coordinationLeaseScopeLabel(lease),
			coordinationLeaseStatusAt(lease, now),
			firstNonEmptyTrimmed(lease.InstanceID, lease.LeaseToken, "-"),
			lease.ExpiresAt.UTC().Format(time.RFC3339),
		})
	}
	var out strings.Builder
	printer := newCLIPrinter(&out)
	if len(rows) == 0 {
		_ = writeCLIPanelWithPrinter(printer, "Capability Leases", "No capability leases found.", "")
		return out.String()
	}
	_ = writeCLITableWithPrinter(
		printer,
		"Capability Leases",
		[]string{"AGENT", "ROLE", "PROJECT", "SCOPE", "STATUS", "ID", "EXPIRES"},
		rows,
		"No capability leases found.",
	)
	return out.String()
}

// writeCoordinationLeaseDetail renders one deterministic human-readable lease detail block.
func writeCoordinationLeaseDetail(stdout io.Writer, now time.Time, lease domain.CapabilityLease) error {
	rows := [][2]string{
		{"agent", coordinationLeaseAgentLabel(lease)},
		{"id", firstNonEmptyTrimmed(lease.InstanceID, lease.LeaseToken, "-")},
		{"role", firstNonEmptyTrimmed(string(lease.Role), "-")},
		{"project", firstNonEmptyTrimmed(lease.ProjectID, "-")},
		{"scope", coordinationLeaseScopeLabel(lease)},
		{"status", coordinationLeaseStatusAt(lease, now)},
		{"parent", firstNonEmptyTrimmed(lease.ParentInstanceID, "-")},
		{"allow equal scope delegation", yesNo(lease.AllowEqualScopeDelegation)},
		{"issued", lease.IssuedAt.UTC().Format(time.RFC3339)},
		{"expires", lease.ExpiresAt.UTC().Format(time.RFC3339)},
		{"heartbeat", lease.HeartbeatAt.UTC().Format(time.RFC3339)},
	}
	if lease.RevokedAt != nil {
		rows = append(rows, [2]string{"revoked", lease.RevokedAt.UTC().Format(time.RFC3339)})
	} else {
		rows = append(rows, [2]string{"revoked", "-"})
	}
	rows = append(rows, [2]string{"revoked reason", firstNonEmptyTrimmed(lease.RevokedReason, "-")})
	if err := writeCLIKV(stdout, "Capability Lease", rows); err != nil {
		return fmt.Errorf("write coordination lease detail: %w", err)
	}
	return nil
}

// writeCoordinationLeaseRevocationSummary renders one deterministic human-readable revoke-all summary.
func writeCoordinationLeaseRevocationSummary(stdout io.Writer, projectID string, scopeType domain.CapabilityScopeType, scopeID, reason string) error {
	rows := [][2]string{
		{"project", firstNonEmptyTrimmed(projectID, "-")},
		{"scope", coordinationLeaseScopeLabelFromParts(projectID, scopeType, scopeID)},
		{"reason", firstNonEmptyTrimmed(reason, "-")},
		{"status", "revoked"},
	}
	if err := writeCLIKV(stdout, "Capability Lease Revocation", rows); err != nil {
		return fmt.Errorf("write coordination lease revoke-all summary: %w", err)
	}
	return nil
}

// renderCoordinationLeaseDetailAt renders one deterministic human-readable lease detail block.
func renderCoordinationLeaseDetailAt(now time.Time, lease domain.CapabilityLease) string {
	rows := [][2]string{
		{"agent", coordinationLeaseAgentLabel(lease)},
		{"id", firstNonEmptyTrimmed(lease.InstanceID, lease.LeaseToken, "-")},
		{"role", firstNonEmptyTrimmed(string(lease.Role), "-")},
		{"project", firstNonEmptyTrimmed(lease.ProjectID, "-")},
		{"scope", coordinationLeaseScopeLabel(lease)},
		{"status", coordinationLeaseStatusAt(lease, now)},
		{"parent", firstNonEmptyTrimmed(lease.ParentInstanceID, "-")},
		{"allow equal scope delegation", yesNo(lease.AllowEqualScopeDelegation)},
		{"issued", lease.IssuedAt.UTC().Format(time.RFC3339)},
		{"expires", lease.ExpiresAt.UTC().Format(time.RFC3339)},
		{"heartbeat", lease.HeartbeatAt.UTC().Format(time.RFC3339)},
	}
	if lease.RevokedAt != nil {
		rows = append(rows, [2]string{"revoked", lease.RevokedAt.UTC().Format(time.RFC3339)})
	} else {
		rows = append(rows, [2]string{"revoked", "-"})
	}
	rows = append(rows, [2]string{"revoked reason", firstNonEmptyTrimmed(lease.RevokedReason, "-")})
	var out strings.Builder
	_ = writeCLIKV(&out, "Capability Lease", rows)
	return out.String()
}

// writeCoordinationHandoffList renders one deterministic human-readable handoff table.
func writeCoordinationHandoffList(stdout io.Writer, handoffs []domain.Handoff) error {
	ordered := append([]domain.Handoff(nil), handoffs...)
	slices.SortFunc(ordered, compareCoordinationHandoffsForCLI)
	rows := make([][]string, 0, len(ordered))
	for _, handoff := range ordered {
		rows = append(rows, []string{
			coordinationHandoffFlowLabel(handoff),
			firstNonEmptyTrimmed(string(handoff.Status), "-"),
			coordinationHandoffScopeLabel(handoff),
			coordinationHandoffTargetLabel(handoff),
			firstNonEmptyTrimmed(handoff.ID, "-"),
			compactText(handoff.Summary),
		})
	}
	printer := newCLIPrinter(stdout)
	if len(rows) == 0 {
		if err := writeCLIPanelWithPrinter(printer, "Handoffs", "No handoffs found.", ""); err != nil {
			return fmt.Errorf("write coordination handoff empty state: %w", err)
		}
		return nil
	}
	if err := writeCLITableWithPrinter(
		printer,
		"Handoffs",
		[]string{"FLOW", "STATUS", "SCOPE", "TARGET", "ID", "SUMMARY"},
		rows,
		"No handoffs found.",
	); err != nil {
		return fmt.Errorf("write coordination handoff list: %w", err)
	}
	return nil
}

// renderCoordinationHandoffList renders one deterministic human-readable handoff table.
func renderCoordinationHandoffList(handoffs []domain.Handoff) string {
	ordered := append([]domain.Handoff(nil), handoffs...)
	slices.SortFunc(ordered, compareCoordinationHandoffsForCLI)
	rows := make([][]string, 0, len(ordered))
	for _, handoff := range ordered {
		rows = append(rows, []string{
			coordinationHandoffFlowLabel(handoff),
			firstNonEmptyTrimmed(string(handoff.Status), "-"),
			coordinationHandoffScopeLabel(handoff),
			coordinationHandoffTargetLabel(handoff),
			firstNonEmptyTrimmed(handoff.ID, "-"),
			compactText(handoff.Summary),
		})
	}
	var out strings.Builder
	printer := newCLIPrinter(&out)
	if len(rows) == 0 {
		_ = writeCLIPanelWithPrinter(printer, "Handoffs", "No handoffs found.", "")
		return out.String()
	}
	_ = writeCLITableWithPrinter(
		printer,
		"Handoffs",
		[]string{"FLOW", "STATUS", "SCOPE", "TARGET", "ID", "SUMMARY"},
		rows,
		"No handoffs found.",
	)
	return out.String()
}

// writeCoordinationHandoffDetail renders one deterministic human-readable handoff detail block.
func writeCoordinationHandoffDetail(stdout io.Writer, handoff domain.Handoff) error {
	rows := [][2]string{
		{"flow", coordinationHandoffFlowLabel(handoff)},
		{"id", firstNonEmptyTrimmed(handoff.ID, "-")},
		{"project", firstNonEmptyTrimmed(handoff.ProjectID, "-")},
		{"scope", coordinationHandoffScopeLabel(handoff)},
		{"target", coordinationHandoffTargetLabel(handoff)},
		{"status", firstNonEmptyTrimmed(string(handoff.Status), "-")},
		{"summary", compactText(handoff.Summary)},
		{"next action", compactText(handoff.NextAction)},
		{"missing evidence", renderCoordinationStringList(handoff.MissingEvidence)},
		{"related refs", renderCoordinationStringList(handoff.RelatedRefs)},
		{"created by", renderCoordinationActorLabel(handoff.CreatedByActor, handoff.CreatedByType)},
		{"updated by", renderCoordinationActorLabel(handoff.UpdatedByActor, handoff.UpdatedByType)},
	}
	if handoff.ResolvedAt != nil {
		rows = append(rows, [2]string{"resolved at", handoff.ResolvedAt.UTC().Format(time.RFC3339)})
	} else {
		rows = append(rows, [2]string{"resolved at", "-"})
	}
	rows = append(rows, [2]string{"resolution note", compactText(handoff.ResolutionNote)})
	if err := writeCLIKV(stdout, "Handoff", rows); err != nil {
		return fmt.Errorf("write coordination handoff detail: %w", err)
	}
	return nil
}

// renderCoordinationHandoffDetail renders one deterministic human-readable handoff detail block.
func renderCoordinationHandoffDetail(handoff domain.Handoff) string {
	rows := [][2]string{
		{"flow", coordinationHandoffFlowLabel(handoff)},
		{"id", firstNonEmptyTrimmed(handoff.ID, "-")},
		{"project", firstNonEmptyTrimmed(handoff.ProjectID, "-")},
		{"scope", coordinationHandoffScopeLabel(handoff)},
		{"target", coordinationHandoffTargetLabel(handoff)},
		{"status", firstNonEmptyTrimmed(string(handoff.Status), "-")},
		{"summary", compactText(handoff.Summary)},
		{"next action", compactText(handoff.NextAction)},
		{"missing evidence", renderCoordinationStringList(handoff.MissingEvidence)},
		{"related refs", renderCoordinationStringList(handoff.RelatedRefs)},
		{"created by", renderCoordinationActorLabel(handoff.CreatedByActor, handoff.CreatedByType)},
		{"updated by", renderCoordinationActorLabel(handoff.UpdatedByActor, handoff.UpdatedByType)},
	}
	if handoff.ResolvedAt != nil {
		rows = append(rows, [2]string{"resolved at", handoff.ResolvedAt.UTC().Format(time.RFC3339)})
	} else {
		rows = append(rows, [2]string{"resolved at", "-"})
	}
	rows = append(rows, [2]string{"resolution note", compactText(handoff.ResolutionNote)})
	var out strings.Builder
	_ = writeCLIKV(&out, "Handoff", rows)
	return out.String()
}

// compareCoordinationLeasesForCLI sorts leases by agent, then role, then id for stable operator output.
func compareCoordinationLeasesForCLI(a, b domain.CapabilityLease) int {
	if c := strings.Compare(strings.ToLower(coordinationLeaseAgentLabel(a)), strings.ToLower(coordinationLeaseAgentLabel(b))); c != 0 {
		return c
	}
	if c := strings.Compare(strings.ToLower(string(a.Role)), strings.ToLower(string(b.Role))); c != 0 {
		return c
	}
	return strings.Compare(strings.ToLower(firstNonEmptyTrimmed(a.InstanceID, a.LeaseToken)), strings.ToLower(firstNonEmptyTrimmed(b.InstanceID, b.LeaseToken)))
}

// compareCoordinationHandoffsForCLI sorts handoffs by flow, then scope, then id for stable operator output.
func compareCoordinationHandoffsForCLI(a, b domain.Handoff) int {
	if c := strings.Compare(strings.ToLower(coordinationHandoffFlowLabel(a)), strings.ToLower(coordinationHandoffFlowLabel(b))); c != 0 {
		return c
	}
	if c := strings.Compare(strings.ToLower(coordinationHandoffScopeLabel(a)), strings.ToLower(coordinationHandoffScopeLabel(b))); c != 0 {
		return c
	}
	return strings.Compare(strings.ToLower(firstNonEmptyTrimmed(a.ID, "")), strings.ToLower(firstNonEmptyTrimmed(b.ID, "")))
}

// coordinationLeaseAgentLabel returns one name-first agent label for a lease row.
func coordinationLeaseAgentLabel(lease domain.CapabilityLease) string {
	return firstNonEmptyTrimmed(lease.AgentName, lease.InstanceID, lease.LeaseToken, "-")
}

// coordinationLeaseScopeLabel returns one stable lease scope label.
func coordinationLeaseScopeLabel(lease domain.CapabilityLease) string {
	return coordinationLeaseScopeLabelFromParts(lease.ProjectID, lease.ScopeType, lease.ScopeID)
}

func coordinationLeaseScopeLabelFromParts(projectID string, scopeType domain.CapabilityScopeType, scopeID string) string {
	rawScopeType := strings.TrimSpace(string(scopeType))
	switch {
	case rawScopeType == "":
		return firstNonEmptyTrimmed(projectID, "-")
	case rawScopeType == string(domain.CapabilityScopeProject):
		return "project/" + firstNonEmptyTrimmed(projectID, "-")
	case strings.TrimSpace(scopeID) == "":
		return rawScopeType
	default:
		return rawScopeType + "/" + strings.TrimSpace(scopeID)
	}
}

// coordinationLeaseStatusAt returns the current lifecycle label for one lease.
func coordinationLeaseStatusAt(lease domain.CapabilityLease, now time.Time) string {
	switch {
	case lease.RevokedAt != nil:
		return "revoked"
	case !lease.ExpiresAt.IsZero() && !now.Before(lease.ExpiresAt.UTC()):
		return "expired"
	default:
		return "active"
	}
}

// coordinationHandoffFlowLabel returns one name-first handoff flow label.
func coordinationHandoffFlowLabel(handoff domain.Handoff) string {
	source := strings.TrimSpace(handoff.SourceRole)
	target := strings.TrimSpace(handoff.TargetRole)
	switch {
	case source != "" && target != "":
		return source + " -> " + target
	case source != "":
		return source
	case target != "":
		return "to " + target
	case strings.TrimSpace(handoff.Summary) != "":
		return compactText(handoff.Summary)
	default:
		return "-"
	}
}

// coordinationHandoffScopeLabel returns one stable handoff scope label.
func coordinationHandoffScopeLabel(handoff domain.Handoff) string {
	scopeType := strings.TrimSpace(string(handoff.ScopeType))
	switch {
	case scopeType == "":
		return firstNonEmptyTrimmed(handoff.ProjectID, "-")
	case scopeType == string(domain.ScopeLevelProject):
		return "project/" + firstNonEmptyTrimmed(handoff.ProjectID, "-")
	case strings.TrimSpace(handoff.ScopeID) == "":
		return scopeType
	default:
		return scopeType + "/" + strings.TrimSpace(handoff.ScopeID)
	}
}

// coordinationHandoffTargetLabel returns one stable handoff target label.
func coordinationHandoffTargetLabel(handoff domain.Handoff) string {
	branchID := strings.TrimSpace(handoff.TargetBranchID)
	targetType := strings.TrimSpace(string(handoff.TargetScopeType))
	targetID := strings.TrimSpace(handoff.TargetScopeID)
	targetRole := strings.TrimSpace(handoff.TargetRole)
	switch {
	case branchID == "" && targetType == "" && targetID == "":
		if targetRole == "" {
			return "-"
		}
		return "role:" + targetRole
	case targetType == string(domain.ScopeLevelProject):
		return firstNonEmptyTrimmed("project/"+targetID, targetID, "-")
	case branchID == "":
		if targetType == "" {
			return firstNonEmptyTrimmed(targetID, "-")
		}
		return targetType + ":" + firstNonEmptyTrimmed(targetID, "-")
	case targetType == "" && targetID == "":
		return branchID
	case targetType == "":
		return branchID + " -> " + targetID
	default:
		return branchID + " -> " + targetType + ":" + targetID
	}
}

// renderCoordinationActorLabel renders one actor identifier with its type when available.
func renderCoordinationActorLabel(actorID string, actorType domain.ActorType) string {
	actorID = strings.TrimSpace(actorID)
	actorType = domain.ActorType(strings.TrimSpace(strings.ToLower(string(actorType))))
	switch {
	case actorID != "" && actorType != "":
		return actorID + " (" + string(actorType) + ")"
	case actorID != "":
		return actorID
	case actorType != "":
		return string(actorType)
	default:
		return "-"
	}
}

// renderCoordinationStringList renders one stable comma-separated list or a fallback dash.
func renderCoordinationStringList(values []string) string {
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

// yesNo renders a boolean as yes or no.
func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}
