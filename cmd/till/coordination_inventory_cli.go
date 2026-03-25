package main

import (
	"bytes"
	"fmt"
	"io"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/hylla/tillsyn/internal/domain"
)

// writeCoordinationLeaseList renders one deterministic human-readable lease table.
func writeCoordinationLeaseList(stdout io.Writer, now time.Time, leases []domain.CapabilityLease) error {
	if _, err := io.WriteString(stdout, renderCoordinationLeaseListAt(now, leases)); err != nil {
		return fmt.Errorf("write coordination lease list: %w", err)
	}
	return nil
}

// renderCoordinationLeaseListAt renders one deterministic human-readable lease table.
func renderCoordinationLeaseListAt(now time.Time, leases []domain.CapabilityLease) string {
	ordered := append([]domain.CapabilityLease(nil), leases...)
	slices.SortFunc(ordered, compareCoordinationLeasesForCLI)
	var buf bytes.Buffer
	tw := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "CAPABILITY LEASES")
	_, _ = fmt.Fprintln(tw, "AGENT\tROLE\tPROJECT\tSCOPE\tSTATUS\tID\tEXPIRES")
	if len(ordered) == 0 {
		_, _ = fmt.Fprintln(tw, "(none)\t-\t-\t-\t-\t-\t-")
	} else {
		for _, lease := range ordered {
			_, _ = fmt.Fprintf(
				tw,
				"%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				coordinationLeaseAgentLabel(lease),
				firstNonEmptyTrimmed(string(lease.Role), "-"),
				firstNonEmptyTrimmed(lease.ProjectID, "-"),
				coordinationLeaseScopeLabel(lease),
				coordinationLeaseStatusAt(lease, now),
				firstNonEmptyTrimmed(lease.InstanceID, lease.LeaseToken, "-"),
				lease.ExpiresAt.UTC().Format(time.RFC3339),
			)
		}
	}
	_ = tw.Flush()
	return buf.String()
}

// writeCoordinationLeaseDetail renders one deterministic human-readable lease detail block.
func writeCoordinationLeaseDetail(stdout io.Writer, now time.Time, lease domain.CapabilityLease) error {
	if _, err := io.WriteString(stdout, renderCoordinationLeaseDetailAt(now, lease)); err != nil {
		return fmt.Errorf("write coordination lease detail: %w", err)
	}
	return nil
}

// renderCoordinationLeaseDetailAt renders one deterministic human-readable lease detail block.
func renderCoordinationLeaseDetailAt(now time.Time, lease domain.CapabilityLease) string {
	var b strings.Builder
	fmt.Fprintln(&b, "CAPABILITY LEASE")
	fmt.Fprintf(&b, "agent\t%s\n", coordinationLeaseAgentLabel(lease))
	fmt.Fprintf(&b, "id\t%s\n", firstNonEmptyTrimmed(lease.InstanceID, lease.LeaseToken, "-"))
	fmt.Fprintf(&b, "role\t%s\n", firstNonEmptyTrimmed(string(lease.Role), "-"))
	fmt.Fprintf(&b, "project\t%s\n", firstNonEmptyTrimmed(lease.ProjectID, "-"))
	fmt.Fprintf(&b, "scope\t%s\n", coordinationLeaseScopeLabel(lease))
	fmt.Fprintf(&b, "status\t%s\n", coordinationLeaseStatusAt(lease, now))
	fmt.Fprintf(&b, "parent\t%s\n", firstNonEmptyTrimmed(lease.ParentInstanceID, "-"))
	fmt.Fprintf(&b, "allow equal scope delegation\t%s\n", yesNo(lease.AllowEqualScopeDelegation))
	fmt.Fprintf(&b, "issued\t%s\n", lease.IssuedAt.UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "expires\t%s\n", lease.ExpiresAt.UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "heartbeat\t%s\n", lease.HeartbeatAt.UTC().Format(time.RFC3339))
	if lease.RevokedAt != nil {
		fmt.Fprintf(&b, "revoked\t%s\n", lease.RevokedAt.UTC().Format(time.RFC3339))
	} else {
		fmt.Fprintln(&b, "revoked\t-")
	}
	fmt.Fprintf(&b, "revoked reason\t%s\n", firstNonEmptyTrimmed(lease.RevokedReason, "-"))
	return b.String()
}

// writeCoordinationHandoffList renders one deterministic human-readable handoff table.
func writeCoordinationHandoffList(stdout io.Writer, handoffs []domain.Handoff) error {
	if _, err := io.WriteString(stdout, renderCoordinationHandoffList(handoffs)); err != nil {
		return fmt.Errorf("write coordination handoff list: %w", err)
	}
	return nil
}

// renderCoordinationHandoffList renders one deterministic human-readable handoff table.
func renderCoordinationHandoffList(handoffs []domain.Handoff) string {
	ordered := append([]domain.Handoff(nil), handoffs...)
	slices.SortFunc(ordered, compareCoordinationHandoffsForCLI)
	var buf bytes.Buffer
	tw := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "HANDOFFS")
	_, _ = fmt.Fprintln(tw, "FLOW\tSTATUS\tSCOPE\tTARGET\tID\tSUMMARY")
	if len(ordered) == 0 {
		_, _ = fmt.Fprintln(tw, "(none)\t-\t-\t-\t-\t-")
	} else {
		for _, handoff := range ordered {
			_, _ = fmt.Fprintf(
				tw,
				"%s\t%s\t%s\t%s\t%s\t%s\n",
				coordinationHandoffFlowLabel(handoff),
				firstNonEmptyTrimmed(string(handoff.Status), "-"),
				coordinationHandoffScopeLabel(handoff),
				coordinationHandoffTargetLabel(handoff),
				firstNonEmptyTrimmed(handoff.ID, "-"),
				compactText(handoff.Summary),
			)
		}
	}
	_ = tw.Flush()
	return buf.String()
}

// writeCoordinationHandoffDetail renders one deterministic human-readable handoff detail block.
func writeCoordinationHandoffDetail(stdout io.Writer, handoff domain.Handoff) error {
	if _, err := io.WriteString(stdout, renderCoordinationHandoffDetail(handoff)); err != nil {
		return fmt.Errorf("write coordination handoff detail: %w", err)
	}
	return nil
}

// renderCoordinationHandoffDetail renders one deterministic human-readable handoff detail block.
func renderCoordinationHandoffDetail(handoff domain.Handoff) string {
	var b strings.Builder
	fmt.Fprintln(&b, "HANDOFF")
	fmt.Fprintf(&b, "flow\t%s\n", coordinationHandoffFlowLabel(handoff))
	fmt.Fprintf(&b, "id\t%s\n", firstNonEmptyTrimmed(handoff.ID, "-"))
	fmt.Fprintf(&b, "project\t%s\n", firstNonEmptyTrimmed(handoff.ProjectID, "-"))
	fmt.Fprintf(&b, "scope\t%s\n", coordinationHandoffScopeLabel(handoff))
	fmt.Fprintf(&b, "target\t%s\n", coordinationHandoffTargetLabel(handoff))
	fmt.Fprintf(&b, "status\t%s\n", firstNonEmptyTrimmed(string(handoff.Status), "-"))
	fmt.Fprintf(&b, "summary\t%s\n", compactText(handoff.Summary))
	fmt.Fprintf(&b, "next action\t%s\n", compactText(handoff.NextAction))
	fmt.Fprintf(&b, "missing evidence\t%s\n", renderCoordinationStringList(handoff.MissingEvidence))
	fmt.Fprintf(&b, "related refs\t%s\n", renderCoordinationStringList(handoff.RelatedRefs))
	fmt.Fprintf(&b, "created by\t%s\n", renderCoordinationActorLabel(handoff.CreatedByActor, handoff.CreatedByType))
	fmt.Fprintf(&b, "updated by\t%s\n", renderCoordinationActorLabel(handoff.UpdatedByActor, handoff.UpdatedByType))
	if handoff.ResolvedAt != nil {
		fmt.Fprintf(&b, "resolved at\t%s\n", handoff.ResolvedAt.UTC().Format(time.RFC3339))
	} else {
		fmt.Fprintln(&b, "resolved at\t-")
	}
	fmt.Fprintf(&b, "resolution note\t%s\n", compactText(handoff.ResolutionNote))
	return b.String()
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
	scopeType := strings.TrimSpace(string(lease.ScopeType))
	switch {
	case scopeType == "":
		return firstNonEmptyTrimmed(lease.ProjectID, "-")
	case scopeType == string(domain.CapabilityScopeProject):
		return "project/" + firstNonEmptyTrimmed(lease.ProjectID, "-")
	case strings.TrimSpace(lease.ScopeID) == "":
		return scopeType
	default:
		return scopeType + "/" + strings.TrimSpace(lease.ScopeID)
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
	slices.SortFunc(ordered, func(a, b string) int { return strings.Compare(strings.ToLower(strings.TrimSpace(a)), strings.ToLower(strings.TrimSpace(b))) })
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
