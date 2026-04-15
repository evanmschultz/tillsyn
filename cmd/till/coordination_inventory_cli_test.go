package main

import (
	"strings"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// TestRenderCoordinationLeaseListAt renders deterministic, name-first lease inventory output.
func TestRenderCoordinationLeaseListAt(t *testing.T) {
	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	revokedAt := now.Add(-time.Minute)
	leases := []domain.CapabilityLease{
		{
			InstanceID:    "lease-b",
			LeaseToken:    "token-b",
			AgentName:     "QA Bot",
			ProjectID:     "p1",
			ScopeType:     domain.CapabilityScopeProject,
			Role:          domain.CapabilityRoleQA,
			IssuedAt:      now.Add(-2 * time.Hour),
			ExpiresAt:     now.Add(2 * time.Hour),
			HeartbeatAt:   now.Add(-time.Minute),
			RevokedAt:     &revokedAt,
			RevokedReason: "manual revoke",
		},
		{
			InstanceID:  "lease-a",
			LeaseToken:  "token-a",
			AgentName:   "Builder Bot",
			ProjectID:   "p1",
			ScopeType:   domain.CapabilityScopeProject,
			Role:        domain.CapabilityRoleBuilder,
			IssuedAt:    now.Add(-time.Hour),
			ExpiresAt:   now.Add(time.Hour),
			HeartbeatAt: now.Add(-time.Minute),
		},
	}
	got := renderCoordinationLeaseListAt(now, leases)
	for _, want := range []string{
		"Capability Leases",
		"AGENT",
		"ROLE",
		"PROJECT",
		"SCOPE",
		"STATUS",
		"ID",
		"EXPIRES",
		"Builder Bot",
		"builder",
		"project/p1",
		"active",
		"lease-a",
		"QA Bot",
		"qa",
		"revoked",
		"lease-b",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in lease list output:\n%s", want, got)
		}
	}
	if strings.Index(got, "Builder Bot") > strings.Index(got, "QA Bot") {
		t.Fatalf("expected builder lease to sort before QA lease:\n%s", got)
	}
}

// TestRenderCoordinationLeaseDetailAt renders a stable detail block with identifiers visible.
func TestRenderCoordinationLeaseDetailAt(t *testing.T) {
	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	revokedAt := now.Add(-time.Minute)
	lease := domain.CapabilityLease{
		InstanceID:                "lease-a",
		LeaseToken:                "token-a",
		AgentName:                 "Builder Bot",
		ProjectID:                 "p1",
		ScopeType:                 domain.CapabilityScopeProject,
		Role:                      domain.CapabilityRoleBuilder,
		ParentInstanceID:          "parent-1",
		AllowEqualScopeDelegation: true,
		IssuedAt:                  now.Add(-time.Hour),
		ExpiresAt:                 now.Add(time.Hour),
		HeartbeatAt:               now.Add(-time.Minute),
		RevokedAt:                 &revokedAt,
		RevokedReason:             "manual revoke",
	}
	got := renderCoordinationLeaseDetailAt(now, lease)
	for _, want := range []string{
		"Capability Lease",
		"agent",
		"Builder Bot",
		"id",
		"lease-a",
		"role",
		"builder",
		"project",
		"p1",
		"scope",
		"project/p1",
		"status",
		"revoked",
		"parent",
		"parent-1",
		"allow equal scope delegation",
		"yes",
		"issued",
		"expires",
		"heartbeat",
		"revoked reason",
		"manual revoke",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in lease detail output:\n%s", want, got)
		}
	}
}

// TestRenderCoordinationHandoffList renders deterministic handoff inventory output.
func TestRenderCoordinationHandoffList(t *testing.T) {
	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	handoffs := []domain.Handoff{
		{
			ID:             "handoff-b",
			ProjectID:      "p1",
			ScopeType:      domain.ScopeLevelProject,
			SourceRole:     "qa",
			TargetRole:     "builder",
			Status:         domain.HandoffStatusBlocked,
			Summary:        "qa blocks builder",
			NextAction:     "builder fixes follow-up",
			CreatedByActor: "lane-user",
			CreatedByType:  domain.ActorTypeUser,
		},
		{
			ID:             "handoff-a",
			ProjectID:      "p1",
			ScopeType:      domain.ScopeLevelProject,
			SourceRole:     "builder",
			TargetRole:     "qa",
			Status:         domain.HandoffStatusWaiting,
			Summary:        "builder to qa handoff",
			NextAction:     "qa verifies run",
			CreatedByActor: "lane-user",
			CreatedByType:  domain.ActorTypeUser,
			CreatedAt:      now,
		},
	}
	got := renderCoordinationHandoffList(handoffs)
	for _, want := range []string{
		"Handoffs",
		"FLOW",
		"STATUS",
		"SCOPE",
		"TARGET",
		"ID",
		"SUMMARY",
		"builder -> qa",
		"waiting",
		"project/p1",
		"role:qa",
		"handoff-a",
		"qa -> builder",
		"blocked",
		"role:builder",
		"handoff-b",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in handoff list output:\n%s", want, got)
		}
	}
	if strings.Index(got, "builder -> qa") > strings.Index(got, "qa -> builder") {
		t.Fatalf("expected builder->qa handoff to sort before qa->builder:\n%s", got)
	}
}

// TestRenderCoordinationHandoffDetail renders a stable handoff detail block with scope and target visibility.
func TestRenderCoordinationHandoffDetail(t *testing.T) {
	resolvedAt := time.Date(2026, 3, 23, 12, 15, 0, 0, time.UTC)
	handoff := domain.Handoff{
		ID:              "handoff-a",
		ProjectID:       "p1",
		BranchID:        "branch-1",
		ScopeType:       domain.ScopeLevelProject,
		ScopeID:         "p1",
		SourceRole:      "builder",
		TargetBranchID:  "branch-1",
		TargetScopeType: domain.ScopeLevelTask,
		TargetScopeID:   "task-1",
		TargetRole:      "qa",
		Status:          domain.HandoffStatusWaiting,
		Summary:         "builder to qa handoff",
		NextAction:      "qa verifies run",
		MissingEvidence: []string{"coverage", "logs"},
		RelatedRefs:     []string{"task-1", "branch-1"},
		CreatedByActor:  "lane-user",
		CreatedByType:   domain.ActorTypeUser,
		UpdatedByActor:  "lane-user",
		UpdatedByType:   domain.ActorTypeUser,
		ResolvedAt:      &resolvedAt,
		ResolutionNote:  "follow-up done",
	}
	got := renderCoordinationHandoffDetail(handoff)
	for _, want := range []string{
		"Handoff",
		"flow",
		"builder -> qa",
		"id",
		"handoff-a",
		"project",
		"p1",
		"scope",
		"project/p1",
		"target",
		"branch-1 -> task:task-1",
		"status",
		"waiting",
		"summary",
		"builder to qa handoff",
		"next action",
		"qa verifies run",
		"missing evidence",
		"coverage, logs",
		"related refs",
		"branch-1, task-1",
		"created by",
		"lane-user (user)",
		"updated by",
		"lane-user (user)",
		"resolved at",
		"resolution note",
		"follow-up done",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in handoff detail output:\n%s", want, got)
		}
	}
}

// TestRenderCoordinationHandoffRoleOnlyTarget verifies role-only handoffs remain visible in target details.
func TestRenderCoordinationHandoffRoleOnlyTarget(t *testing.T) {
	handoff := domain.Handoff{
		ID:         "handoff-role",
		ProjectID:  "p1",
		ScopeType:  domain.ScopeLevelProject,
		SourceRole: "builder",
		TargetRole: "qa",
		Status:     domain.HandoffStatusWaiting,
		Summary:    "qa review handoff",
	}
	got := renderCoordinationHandoffDetail(handoff)
	for _, want := range []string{
		"flow",
		"builder -> qa",
		"target",
		"role:qa",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in role-only handoff detail output:\n%s", want, got)
		}
	}
}

// TestRenderCoordinationHandoffTargetlessFallback verifies empty role-less targets render as a clean dash.
func TestRenderCoordinationHandoffTargetlessFallback(t *testing.T) {
	handoff := domain.Handoff{
		ID:        "handoff-empty",
		ProjectID: "p1",
		ScopeType: domain.ScopeLevelProject,
		Status:    domain.HandoffStatusWaiting,
		Summary:   "unscoped note",
	}
	if got := coordinationHandoffTargetLabel(handoff); got != "-" {
		t.Fatalf("coordinationHandoffTargetLabel() = %q, want -", got)
	}
}
