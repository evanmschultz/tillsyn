package domain

import (
	"testing"
	"time"
)

// TestNewKindDefinitionValidation verifies catalog normalization and validation behavior.
func TestNewKindDefinitionValidation(t *testing.T) {
	now := time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	kind, err := NewKindDefinition(KindDefinitionInput{
		ID:                  " Refactor ",
		DisplayName:         " Refactor Work ",
		DescriptionMarkdown: " refactor tasks ",
		AppliesTo:           []KindAppliesTo{KindAppliesToTask, KindAppliesToTask, KindAppliesToSubtask},
		AllowedParentScopes: []KindAppliesTo{KindAppliesToPhase},
		PayloadSchemaJSON:   `{"type":"object","required":["package"],"properties":{"package":{"type":"string"}}}`,
		Template: KindTemplate{
			CompletionChecklist: []ChecklistItem{{ID: "c1", Text: "run tests", Done: false}},
			AutoCreateChildren: []KindTemplateChildSpec{{
				Title:     "scan packages",
				Kind:      "task",
				AppliesTo: KindAppliesToSubtask,
			}},
			ProjectMetadataDefaults: &ProjectMetadata{
				Owner:    "  Team A ",
				Tags:     []string{"Alpha", "alpha"},
				Homepage: " https://example.com ",
			},
			TaskMetadataDefaults: &TaskMetadata{
				Objective:       "  default objective  ",
				CommandSnippets: []string{"make test", "make test"},
				CompletionContract: CompletionContract{
					CompletionChecklist: []ChecklistItem{{Text: "default check"}},
					Policy:              CompletionPolicy{RequireChildrenDone: true},
				},
			},
		},
	}, now)
	if err != nil {
		t.Fatalf("NewKindDefinition() error = %v", err)
	}
	if kind.ID != KindID("refactor") {
		t.Fatalf("expected normalized id refactor, got %q", kind.ID)
	}
	if !kind.AppliesToScope(KindAppliesToTask) {
		t.Fatal("expected applies_to task")
	}
	if !kind.AllowsParentScope(KindAppliesToPhase) {
		t.Fatal("expected allowed parent scope phase")
	}
	if len(kind.Template.AutoCreateChildren) != 1 {
		t.Fatalf("expected one child template, got %d", len(kind.Template.AutoCreateChildren))
	}
	if kind.Template.ProjectMetadataDefaults == nil {
		t.Fatal("expected normalized project metadata defaults")
	}
	if kind.Template.ProjectMetadataDefaults.Owner != "Team A" {
		t.Fatalf("unexpected project default owner %q", kind.Template.ProjectMetadataDefaults.Owner)
	}
	if len(kind.Template.ProjectMetadataDefaults.Tags) != 1 || kind.Template.ProjectMetadataDefaults.Tags[0] != "alpha" {
		t.Fatalf("unexpected project default tags %#v", kind.Template.ProjectMetadataDefaults.Tags)
	}
	if kind.Template.TaskMetadataDefaults == nil {
		t.Fatal("expected normalized task metadata defaults")
	}
	if kind.Template.TaskMetadataDefaults.Objective != "default objective" {
		t.Fatalf("unexpected task default objective %q", kind.Template.TaskMetadataDefaults.Objective)
	}
	if len(kind.Template.TaskMetadataDefaults.CommandSnippets) != 1 || kind.Template.TaskMetadataDefaults.CommandSnippets[0] != "make test" {
		t.Fatalf("unexpected task default command snippets %#v", kind.Template.TaskMetadataDefaults.CommandSnippets)
	}
	if !kind.Template.TaskMetadataDefaults.CompletionContract.Policy.RequireChildrenDone {
		t.Fatal("expected normalized task default completion policy")
	}
	if !kind.CreatedAt.Equal(now.UTC()) || !kind.UpdatedAt.Equal(now.UTC()) {
		t.Fatalf("expected UTC timestamps, got created=%s updated=%s", kind.CreatedAt, kind.UpdatedAt)
	}
}

// TestNewKindDefinitionRejectsInvalidValues verifies validation errors for malformed entries.
func TestNewKindDefinitionRejectsInvalidValues(t *testing.T) {
	now := time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	if _, err := NewKindDefinition(KindDefinitionInput{ID: "", AppliesTo: []KindAppliesTo{KindAppliesToTask}}, now); err != ErrInvalidKindID {
		t.Fatalf("expected ErrInvalidKindID, got %v", err)
	}
	if _, err := NewKindDefinition(KindDefinitionInput{ID: "x", AppliesTo: []KindAppliesTo{KindAppliesTo("bad")}}, now); err == nil {
		t.Fatal("expected invalid applies_to error")
	}
	if _, err := NewKindDefinition(KindDefinitionInput{ID: "x", AppliesTo: []KindAppliesTo{KindAppliesToTask}, PayloadSchemaJSON: "{"}, now); err != ErrInvalidKindPayloadSchema {
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
	if !lease.MatchesScope(CapabilityScopeTask, "any") {
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
	if !IsValidCapabilityAction(CapabilityActionApproveAuthWithinBounds) {
		t.Fatal("IsValidCapabilityAction(approve-auth-within-bounds) = false, want true")
	}
	if IsValidCapabilityAction(CapabilityAction("invalid")) {
		t.Fatal("IsValidCapabilityAction(invalid) = true, want false")
	}
	if !CapabilityRoleBuilder.CanPerform(CapabilityActionEditNode) {
		t.Fatal("CapabilityRoleBuilder.CanPerform(edit-node) = false, want true")
	}
	if CapabilityRoleBuilder.CanPerform(CapabilityActionSignoff) {
		t.Fatal("CapabilityRoleBuilder.CanPerform(signoff) = true, want false")
	}
	if !CapabilityRoleQA.CanPerform(CapabilityActionSignoff) {
		t.Fatal("CapabilityRoleQA.CanPerform(signoff) = false, want true")
	}
	if !CapabilityRoleOrchestrator.CanPerform(CapabilityActionApproveAuthWithinBounds) {
		t.Fatal("CapabilityRoleOrchestrator.CanPerform(approve-auth-within-bounds) = false, want true")
	}
	if !CapabilityRoleOrchestrator.CanDelegateTo(CapabilityRoleBuilder) {
		t.Fatal("CapabilityRoleOrchestrator.CanDelegateTo(builder) = false, want true")
	}
	if !CapabilityRoleOrchestrator.CanDelegateTo(CapabilityRoleQA) {
		t.Fatal("CapabilityRoleOrchestrator.CanDelegateTo(qa) = false, want true")
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
