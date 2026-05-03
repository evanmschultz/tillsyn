package domain

import (
	"testing"
	"time"
)

// TestNewKindDefinitionValidation verifies catalog normalization and validation behavior.
//
// Per Drop 3 droplet 3.15 the legacy KindTemplate / AllowedParentScopes /
// AllowsParentScope surface was deleted; KindDefinition now carries only
// catalog-shape fields (id, display name, description, applies_to, payload
// schema JSON, timestamps). Parent/child nesting flows through
// templates.Template.AllowsNesting + the project's baked KindCatalog.
func TestNewKindDefinitionValidation(t *testing.T) {
	now := time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	kind, err := NewKindDefinition(KindDefinitionInput{
		ID:                  " Refactor ",
		DisplayName:         " Refactor Work ",
		DescriptionMarkdown: " refactor tasks ",
		AppliesTo:           []KindAppliesTo{KindAppliesToBuild, KindAppliesToBuild, KindAppliesToResearch},
		PayloadSchemaJSON:   `{"type":"object","required":["package"],"properties":{"package":{"type":"string"}}}`,
	}, now)
	if err != nil {
		t.Fatalf("NewKindDefinition() error = %v", err)
	}
	if kind.ID != KindID("refactor") {
		t.Fatalf("expected normalized id refactor, got %q", kind.ID)
	}
	if kind.DisplayName != "Refactor Work" {
		t.Fatalf("expected trimmed display name, got %q", kind.DisplayName)
	}
	if kind.DescriptionMarkdown != "refactor tasks" {
		t.Fatalf("expected trimmed description, got %q", kind.DescriptionMarkdown)
	}
	if !kind.AppliesToScope(KindAppliesToBuild) {
		t.Fatal("expected applies_to build")
	}
	if !kind.AppliesToScope(KindAppliesToResearch) {
		t.Fatal("expected applies_to research after de-duplication")
	}
	if len(kind.AppliesTo) != 2 {
		t.Fatalf("expected de-duplicated applies_to, got %#v", kind.AppliesTo)
	}
	if !kind.CreatedAt.Equal(now.UTC()) || !kind.UpdatedAt.Equal(now.UTC()) {
		t.Fatalf("expected UTC timestamps, got created=%s updated=%s", kind.CreatedAt, kind.UpdatedAt)
	}
}

// TestNewKindDefinitionRejectsInvalidValues verifies validation errors for malformed entries.
func TestNewKindDefinitionRejectsInvalidValues(t *testing.T) {
	now := time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	if _, err := NewKindDefinition(KindDefinitionInput{ID: "", AppliesTo: []KindAppliesTo{KindAppliesToBuild}}, now); err != ErrInvalidKindID {
		t.Fatalf("expected ErrInvalidKindID, got %v", err)
	}
	if _, err := NewKindDefinition(KindDefinitionInput{ID: "x", AppliesTo: []KindAppliesTo{KindAppliesTo("bad")}}, now); err == nil {
		t.Fatal("expected invalid applies_to error")
	}
	if _, err := NewKindDefinition(KindDefinitionInput{ID: "x", AppliesTo: []KindAppliesTo{KindAppliesToBuild}, PayloadSchemaJSON: "{"}, now); err != ErrInvalidKindPayloadSchema {
		t.Fatalf("expected ErrInvalidKindPayloadSchema, got %v", err)
	}
}

// TestCapabilityLeaseLifecycle verifies active/expired/revoked lease behavior.
func TestCapabilityLeaseLifecycle(t *testing.T) {
	now := time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	lease, err := NewCapabilityLease(CapabilityLeaseInput{
		InstanceID: "inst-1",
		LeaseToken: "token-1",
		AgentName:  "orch-1",
		ProjectID:  "p1",
		ScopeType:  CapabilityScopeProject,
		Role:       CapabilityRoleOrchestrator,
		ExpiresAt:  now.Add(time.Hour),
	}, now)
	if err != nil {
		t.Fatalf("NewCapabilityLease() error = %v", err)
	}
	if !lease.IsActive(now.Add(10 * time.Minute)) {
		t.Fatal("expected lease to be active")
	}
	if lease.ScopeID != "p1" {
		t.Fatalf("expected project scope id to normalize to p1, got %q", lease.ScopeID)
	}
	if !lease.MatchesScope(CapabilityScopeActionItem, "any") {
		t.Fatal("expected project-scope lease to match descendant scope")
	}
	if !lease.MatchesIdentity("orch-1", "token-1") {
		t.Fatal("expected identity match")
	}

	lease.Revoke("manual", now.Add(5*time.Minute))
	if !lease.IsRevoked() {
		t.Fatal("expected revoked lease")
	}
	if lease.IsActive(now.Add(6 * time.Minute)) {
		t.Fatal("expected revoked lease to be inactive")
	}
}

// TestCapabilityRolePolicyHelpers verifies role normalization, default actions, and delegation helpers.
func TestCapabilityRolePolicyHelpers(t *testing.T) {
	t.Parallel()

	if got := NormalizeCapabilityRole(" builder "); got != CapabilityRoleBuilder {
		t.Fatalf("NormalizeCapabilityRole(builder) = %q, want builder", got)
	}
	if got := NormalizeCapabilityRole("worker"); got != CapabilityRoleBuilder {
		t.Fatalf("NormalizeCapabilityRole(worker) = %q, want builder alias", got)
	}
	if got := NormalizeCapabilityRole("subagent"); got != CapabilityRoleBuilder {
		t.Fatalf("NormalizeCapabilityRole(subagent) = %q, want builder alias", got)
	}
	if got := NormalizeCapabilityRole(" qa "); got != CapabilityRoleQA {
		t.Fatalf("NormalizeCapabilityRole(qa) = %q, want qa", got)
	}
	if got := NormalizeCapabilityRole(" research "); got != CapabilityRoleResearch {
		t.Fatalf("NormalizeCapabilityRole(research) = %q, want research", got)
	}
	if !IsValidCapabilityAction(CapabilityActionApproveAuthWithinBounds) {
		t.Fatal("IsValidCapabilityAction(approve-auth-within-bounds) = false, want true")
	}
	if IsValidCapabilityAction(CapabilityAction("invalid")) {
		t.Fatal("IsValidCapabilityAction(invalid) = true, want false")
	}
	if !CapabilityRoleBuilder.CanPerform(CapabilityActionEditNode) {
		t.Fatal("CapabilityRoleBuilder.CanPerform(edit-node) = false, want true")
	}
	if !CapabilityRoleQA.CanPerform(CapabilityActionEditNode) {
		t.Fatal("CapabilityRoleQA.CanPerform(edit-node) = false, want true")
	}
	if CapabilityRoleBuilder.CanPerform(CapabilityActionSignoff) {
		t.Fatal("CapabilityRoleBuilder.CanPerform(signoff) = true, want false")
	}
	if !CapabilityRoleQA.CanPerform(CapabilityActionSignoff) {
		t.Fatal("CapabilityRoleQA.CanPerform(signoff) = false, want true")
	}
	if !CapabilityRoleResearch.CanPerform(CapabilityActionEditNode) {
		t.Fatal("CapabilityRoleResearch.CanPerform(edit-node) = false, want true")
	}
	if CapabilityRoleResearch.CanPerform(CapabilityActionSignoff) {
		t.Fatal("CapabilityRoleResearch.CanPerform(signoff) = true, want false")
	}
	if !CapabilityRoleOrchestrator.CanPerform(CapabilityActionApproveAuthWithinBounds) {
		t.Fatal("CapabilityRoleOrchestrator.CanPerform(approve-auth-within-bounds) = false, want true")
	}
	if !CapabilityRoleOrchestrator.CanPerform(CapabilityActionEditNode) {
		t.Fatal("CapabilityRoleOrchestrator.CanPerform(edit-node) = false, want true")
	}
	if !CapabilityRoleOrchestrator.CanDelegateTo(CapabilityRoleBuilder) {
		t.Fatal("CapabilityRoleOrchestrator.CanDelegateTo(builder) = false, want true")
	}
	if !CapabilityRoleOrchestrator.CanDelegateTo(CapabilityRoleQA) {
		t.Fatal("CapabilityRoleOrchestrator.CanDelegateTo(qa) = false, want true")
	}
	if !CapabilityRoleOrchestrator.CanDelegateTo(CapabilityRoleResearch) {
		t.Fatal("CapabilityRoleOrchestrator.CanDelegateTo(research) = false, want true")
	}
	if CapabilityRoleBuilder.CanDelegateTo(CapabilityRoleQA) {
		t.Fatal("CapabilityRoleBuilder.CanDelegateTo(qa) = true, want false")
	}
	if CapabilityRoleQA.CanTargetBroaderThanProject() {
		t.Fatal("CapabilityRoleQA.CanTargetBroaderThanProject() = true, want false")
	}
	if !CapabilityRoleOrchestrator.CanTargetBroaderThanProject() {
		t.Fatal("CapabilityRoleOrchestrator.CanTargetBroaderThanProject() = false, want true")
	}
	if !CapabilityRoleSystem.IsInternalOnly() {
		t.Fatal("CapabilityRoleSystem.IsInternalOnly() = false, want true")
	}
}
