package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// TestServiceHandoffLifecycle verifies create, list, and update behavior for durable handoffs.
func TestServiceHandoffLifecycle(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)
	project, err := domain.NewProjectFromInput(domain.ProjectInput{ID: "project-1", Name: "Project"}, now)
	if err != nil {
		t.Fatalf("NewProjectFromInput() error = %v", err)
	}
	repo.projects[project.ID] = project

	svc := NewService(repo, func() string { return "handoff-1" }, func() time.Time { return now }, ServiceConfig{})
	handoff, err := svc.CreateHandoff(context.Background(), CreateHandoffInput{
		Level:           domain.LevelTupleInput{ProjectID: project.ID, ScopeType: domain.ScopeLevelProject},
		SourceRole:      "orchestrator",
		TargetScopeType: domain.ScopeLevelActionItem,
		TargetScopeID:   "actionItem-1",
		TargetRole:      "builder",
		Status:          domain.HandoffStatusReady,
		Summary:         "Queue the builder lane",
		NextAction:      "Builder picks up implementation",
		MissingEvidence: []string{"code changes"},
		RelatedRefs:     []string{"actionItem-1"},
		CreatedBy:       "user-1",
		CreatedType:     domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("CreateHandoff() error = %v", err)
	}
	if handoff.ID != "handoff-1" {
		t.Fatalf("ID = %q, want handoff-1", handoff.ID)
	}
	if handoff.ScopeID != project.ID {
		t.Fatalf("ScopeID = %q, want %q", handoff.ScopeID, project.ID)
	}

	listed, err := svc.ListHandoffs(context.Background(), ListHandoffsInput{
		Level: domain.LevelTupleInput{ProjectID: project.ID, ScopeType: domain.ScopeLevelProject},
	})
	if err != nil {
		t.Fatalf("ListHandoffs() error = %v", err)
	}
	if len(listed) != 1 || listed[0].ID != handoff.ID {
		t.Fatalf("unexpected listed handoffs %#v", listed)
	}

	updatedAt := now.Add(10 * time.Minute)
	svc.clock = func() time.Time { return updatedAt }
	updated, err := svc.UpdateHandoff(context.Background(), UpdateHandoffInput{
		HandoffID:       handoff.ID,
		Status:          domain.HandoffStatusResolved,
		Summary:         "Builder completed the lane",
		NextAction:      "Return to orchestrator",
		MissingEvidence: []string{"code changes", "package tests"},
		UpdatedBy:       "qa-1",
		UpdatedType:     domain.ActorTypeUser,
		ResolvedBy:      "qa-1",
		ResolvedType:    domain.ActorTypeUser,
		ResolutionNote:  "validated",
	})
	if err != nil {
		t.Fatalf("UpdateHandoff() error = %v", err)
	}
	if updated.ResolvedAt == nil || !updated.ResolvedAt.Equal(updatedAt) {
		t.Fatalf("ResolvedAt = %v, want %v", updated.ResolvedAt, updatedAt)
	}
	if updated.ResolutionNote != "validated" {
		t.Fatalf("ResolutionNote = %q, want validated", updated.ResolutionNote)
	}
}

// TestServiceHandoffLifecycleSyncsInboxAttention verifies routed handoffs mirror into one updatable inbox attention row.
func TestServiceHandoffLifecycleSyncsInboxAttention(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 3, 21, 13, 0, 0, 0, time.UTC)
	project, err := domain.NewProjectFromInput(domain.ProjectInput{ID: "project-1", Name: "Project"}, now)
	if err != nil {
		t.Fatalf("NewProjectFromInput() error = %v", err)
	}
	repo.projects[project.ID] = project

	svc := NewService(repo, func() string { return "handoff-inbox" }, func() time.Time { return now }, ServiceConfig{})
	handoff, err := svc.CreateHandoff(context.Background(), CreateHandoffInput{
		Level:           domain.LevelTupleInput{ProjectID: project.ID, ScopeType: domain.ScopeLevelProject},
		SourceRole:      "orchestrator",
		TargetScopeType: domain.ScopeLevelActionItem,
		TargetScopeID:   "actionItem-1",
		TargetRole:      "dev",
		Status:          domain.HandoffStatusWaiting,
		Summary:         "Builder should take the next pass",
		NextAction:      "Implement the follow-up",
		CreatedBy:       "user-1",
		CreatedType:     domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("CreateHandoff() error = %v", err)
	}

	mirrored, ok := repo.attentionItems[handoff.ID+"::handoff"]
	if !ok {
		t.Fatalf("expected mirrored handoff attention item, got %#v", repo.attentionItems)
	}
	if mirrored.Kind != domain.AttentionKindHandoff || mirrored.TargetRole != "builder" {
		t.Fatalf("unexpected mirrored handoff attention %#v", mirrored)
	}
	if mirrored.State != domain.AttentionStateOpen || !mirrored.RequiresUserAction {
		t.Fatalf("expected open routed handoff attention, got %#v", mirrored)
	}
	if mirrored.ScopeType != domain.ScopeLevelActionItem || mirrored.ScopeID != "actionItem-1" {
		t.Fatalf("expected target actionItem scope for mirrored handoff, got %#v", mirrored)
	}

	svc.clock = func() time.Time { return now.Add(5 * time.Minute) }
	updated, err := svc.UpdateHandoff(context.Background(), UpdateHandoffInput{
		HandoffID:      handoff.ID,
		Status:         domain.HandoffStatusResolved,
		Summary:        "Builder completed the pass",
		UpdatedBy:      "qa-1",
		UpdatedType:    domain.ActorTypeUser,
		ResolvedBy:     "qa-1",
		ResolvedType:   domain.ActorTypeUser,
		ResolutionNote: "verified",
	})
	if err != nil {
		t.Fatalf("UpdateHandoff() error = %v", err)
	}
	resolved := repo.attentionItems[updated.ID+"::handoff"]
	if resolved.State != domain.AttentionStateResolved || resolved.ResolvedAt == nil {
		t.Fatalf("expected resolved mirrored handoff attention, got %#v", resolved)
	}
	if resolved.TargetRole != "builder" {
		t.Fatalf("expected canonical target role preserved, got %#v", resolved)
	}
}

// TestServiceListHandoffsWaitsForLiveChange verifies handoff list wait_timeout resumes on the next project-scoped handoff change.
func TestServiceListHandoffsWaitsForLiveChange(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC)
	project, err := domain.NewProjectFromInput(domain.ProjectInput{ID: "project-1", Name: "Project"}, now)
	if err != nil {
		t.Fatalf("NewProjectFromInput() error = %v", err)
	}
	repo.projects[project.ID] = project

	svc := NewService(repo, func() string { return "handoff-live" }, func() time.Time { return now }, ServiceConfig{})
	resultCh := make(chan []domain.Handoff, 1)
	errCh := make(chan error, 1)
	go func() {
		items, listErr := svc.ListHandoffs(context.Background(), ListHandoffsInput{
			Level:       domain.LevelTupleInput{ProjectID: project.ID, ScopeType: domain.ScopeLevelProject},
			WaitTimeout: time.Second,
		})
		if listErr != nil {
			errCh <- listErr
			return
		}
		resultCh <- items
	}()

	select {
	case got := <-resultCh:
		t.Fatalf("ListHandoffs() returned early with %#v before a live change", got)
	case err := <-errCh:
		t.Fatalf("ListHandoffs() early error = %v", err)
	case <-time.After(25 * time.Millisecond):
	}

	if _, err := svc.CreateHandoff(context.Background(), CreateHandoffInput{
		Level:       domain.LevelTupleInput{ProjectID: project.ID, ScopeType: domain.ScopeLevelProject},
		SourceRole:  "builder",
		TargetRole:  "qa",
		Status:      domain.HandoffStatusWaiting,
		Summary:     "builder ready for qa",
		CreatedBy:   "user-1",
		CreatedType: domain.ActorTypeUser,
	}); err != nil {
		t.Fatalf("CreateHandoff() error = %v", err)
	}

	select {
	case err := <-errCh:
		t.Fatalf("ListHandoffs() error = %v", err)
	case items := <-resultCh:
		if len(items) != 1 || items[0].ID != "handoff-live" {
			t.Fatalf("ListHandoffs() = %#v, want handoff-live after wake", items)
		}
	case <-time.After(time.Second):
		t.Fatal("ListHandoffs() did not wake after a live handoff change")
	}
}

// TestServiceCreateHandoffUsesResolvedMutationActor verifies context identity wins for persisted attribution.
func TestServiceCreateHandoffUsesResolvedMutationActor(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)
	project, err := domain.NewProjectFromInput(domain.ProjectInput{ID: "project-1", Name: "Project"}, now)
	if err != nil {
		t.Fatalf("NewProjectFromInput() error = %v", err)
	}
	repo.projects[project.ID] = project

	svc := NewService(repo, func() string { return "handoff-actor" }, func() time.Time { return now }, ServiceConfig{
		RequireAgentLease: boolPtr(true),
	})
	lease, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		ScopeID:         project.ID,
		Role:            domain.CapabilityRoleBuilder,
		AgentName:       "agent-1",
		AgentInstanceID: "agent-1-instance",
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease() error = %v", err)
	}
	ctx := WithMutationGuard(context.Background(), MutationGuard{
		AgentName:       lease.AgentName,
		AgentInstanceID: lease.InstanceID,
		LeaseToken:      lease.LeaseToken,
	})
	ctx = WithMutationActor(ctx, MutationActor{
		ActorID:   "agent-1",
		ActorName: "Agent One",
		ActorType: domain.ActorTypeAgent,
	})

	handoff, err := svc.CreateHandoff(ctx, CreateHandoffInput{
		Level:       domain.LevelTupleInput{ProjectID: project.ID, ScopeType: domain.ScopeLevelProject},
		SourceRole:  "orchestrator",
		TargetRole:  "builder",
		Summary:     "Use resolved actor",
		CreatedBy:   "user-ignored",
		CreatedType: domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("CreateHandoff() error = %v", err)
	}
	if handoff.CreatedByActor != "agent-1" || handoff.UpdatedByActor != "agent-1" {
		t.Fatalf("expected resolved actor attribution, got %#v", handoff)
	}
	if handoff.CreatedByType != domain.ActorTypeAgent || handoff.UpdatedByType != domain.ActorTypeAgent {
		t.Fatalf("expected resolved actor types, got %#v", handoff)
	}
}

// TestServiceCreateHandoffRequiresValidScope verifies scope validation on create.
func TestServiceCreateHandoffRequiresValidScope(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, func() string { return "handoff-1" }, time.Now, ServiceConfig{})
	if _, err := svc.CreateHandoff(context.Background(), CreateHandoffInput{
		Level:       domain.LevelTupleInput{ProjectID: "project-1", ScopeType: domain.ScopeLevelActionItem},
		SourceRole:  "builder",
		TargetRole:  "qa",
		Summary:     "bad scope",
		CreatedType: domain.ActorTypeUser,
	}); err == nil {
		t.Fatal("expected create handoff scope validation error")
	}
}

// TestServiceListHandoffsRequiresExistingScope verifies list rejects unknown scopes.
func TestServiceListHandoffsRequiresExistingScope(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)
	project, err := domain.NewProjectFromInput(domain.ProjectInput{ID: "project-1", Name: "Project"}, now)
	if err != nil {
		t.Fatalf("NewProjectFromInput() error = %v", err)
	}
	repo.projects[project.ID] = project

	svc := NewService(repo, func() string { return "handoff-1" }, func() time.Time { return now }, ServiceConfig{})
	_, err = svc.ListHandoffs(context.Background(), ListHandoffsInput{
		Level: domain.LevelTupleInput{ProjectID: project.ID, ScopeType: domain.ScopeLevelActionItem, ScopeID: "missing-actionItem"},
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("ListHandoffs() error = %v, want %v", err, ErrNotFound)
	}
}

// TestServiceUpdateHandoffClearsOptionalFields verifies update can clear target and next-action state.
func TestServiceUpdateHandoffClearsOptionalFields(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)
	project, err := domain.NewProjectFromInput(domain.ProjectInput{ID: "project-1", Name: "Project"}, now)
	if err != nil {
		t.Fatalf("NewProjectFromInput() error = %v", err)
	}
	repo.projects[project.ID] = project

	svc := NewService(repo, func() string { return "handoff-clear" }, func() time.Time { return now }, ServiceConfig{})
	handoff, err := svc.CreateHandoff(context.Background(), CreateHandoffInput{
		Level:           domain.LevelTupleInput{ProjectID: project.ID, ScopeType: domain.ScopeLevelProject},
		SourceRole:      "orchestrator",
		TargetScopeType: domain.ScopeLevelActionItem,
		TargetScopeID:   "actionItem-1",
		TargetRole:      "builder",
		Summary:         "Initial handoff",
		NextAction:      "Do the work",
		MissingEvidence: []string{"tests"},
		RelatedRefs:     []string{"actionItem-1"},
		CreatedBy:       "user-1",
		CreatedType:     domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("CreateHandoff() error = %v", err)
	}

	updated, err := svc.UpdateHandoff(context.Background(), UpdateHandoffInput{
		HandoffID:       handoff.ID,
		Status:          domain.HandoffStatusBlocked,
		SourceRole:      "orchestrator",
		Summary:         "Blocked after review",
		NextAction:      "",
		MissingEvidence: []string{},
		RelatedRefs:     []string{},
		UpdatedBy:       "user-1",
		UpdatedType:     domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("UpdateHandoff() error = %v", err)
	}
	if updated.TargetScopeID != "actionItem-1" || updated.TargetScopeType != domain.ScopeLevelActionItem || updated.TargetRole != "builder" {
		t.Fatalf("expected status-only update to preserve target fields, got %#v", updated)
	}
	if updated.NextAction != "" {
		t.Fatalf("expected next action cleared, got %q", updated.NextAction)
	}
	if len(updated.MissingEvidence) != 0 || len(updated.RelatedRefs) != 0 {
		t.Fatalf("expected optional list fields cleared, got evidence=%#v refs=%#v", updated.MissingEvidence, updated.RelatedRefs)
	}
}
