package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// templateContractFixture stores common project/actionItem state for node-contract enforcement tests.
type templateContractFixture struct {
	repo     *fakeRepo
	svc      *Service
	now      time.Time
	project  domain.Project
	todo     domain.Column
	progress domain.Column
	done     domain.Column
}

// newTemplateContractFixture builds a service fixture with one project and canonical board columns.
func newTemplateContractFixture(t *testing.T) templateContractFixture {
	t.Helper()

	repo := newFakeRepo()
	now := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	project, err := domain.NewProject("p1", "Inbox", "", now)
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	todo, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	progress, _ := domain.NewColumn("c2", project.ID, "In Progress", 1, 0, now)
	done, _ := domain.NewColumn("c3", project.ID, "Done", 2, 0, now)
	repo.projects[project.ID] = project
	repo.columns[todo.ID] = todo
	repo.columns[progress.ID] = progress
	repo.columns[done.ID] = done

	svc := NewService(repo, nil, func() time.Time { return now.Add(time.Minute) }, ServiceConfig{})
	return templateContractFixture{
		repo:     repo,
		svc:      svc,
		now:      now,
		project:  project,
		todo:     todo,
		progress: progress,
		done:     done,
	}
}

// storeActionItem persists one actionItem directly into the fake repository for a focused service test.
func (f templateContractFixture) storeActionItem(t *testing.T, in domain.ActionItemInput) domain.ActionItem {
	t.Helper()

	actionItem, err := domain.NewActionItem(in, f.now)
	if err != nil {
		t.Fatalf("NewActionItem() error = %v", err)
	}
	f.repo.tasks[actionItem.ID] = actionItem
	return actionItem
}

// storeNodeContract persists one generated-node contract snapshot for the provided actionItem.
func (f templateContractFixture) storeNodeContract(t *testing.T, actionItem domain.ActionItem, in domain.NodeContractSnapshotInput) {
	t.Helper()

	in.NodeID = actionItem.ID
	in.ProjectID = actionItem.ProjectID
	if strings.TrimSpace(in.SourceLibraryID) == "" {
		in.SourceLibraryID = "lib-1"
	}
	if strings.TrimSpace(in.SourceNodeTemplateID) == "" {
		in.SourceNodeTemplateID = "tmpl-1"
	}
	if strings.TrimSpace(in.SourceChildRuleID) == "" {
		in.SourceChildRuleID = "rule-1"
	}
	snapshot, err := domain.NewNodeContractSnapshot(in, f.now)
	if err != nil {
		t.Fatalf("NewNodeContractSnapshot() error = %v", err)
	}
	f.repo.nodeContracts[actionItem.ID] = snapshot
}

// leaseContext issues one capability lease and returns the matching mutation context for tests.
func (f templateContractFixture) leaseContext(t *testing.T, scopeType domain.CapabilityScopeType, scopeID string, role domain.CapabilityRole, actorID string) context.Context {
	t.Helper()

	lease, err := f.svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       f.project.ID,
		ScopeType:       scopeType,
		ScopeID:         scopeID,
		Role:            role,
		AgentName:       actorID,
		AgentInstanceID: actorID,
		RequestedTTL:    time.Hour,
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease() error = %v", err)
	}
	ctx := WithMutationGuard(context.Background(), MutationGuard{
		AgentName:       lease.AgentName,
		AgentInstanceID: lease.InstanceID,
		LeaseToken:      lease.LeaseToken,
	})
	return WithMutationActor(ctx, MutationActor{
		ActorID:   actorID,
		ActorName: actorID,
		ActorType: domain.ActorTypeAgent,
	})
}

// TestUpdateActionItemBlocksGeneratedNodeEditForWrongActorKind verifies generated-node edit permissions fail closed for the wrong actor kind.
func TestUpdateActionItemBlocksGeneratedNodeEditForWrongActorKind(t *testing.T) {
	fixture := newTemplateContractFixture(t)
	actionItem := fixture.storeActionItem(t, domain.ActionItemInput{
		ID:             "actionItem-qa",
		ProjectID:      fixture.project.ID,
		ColumnID:       fixture.progress.ID,
		Position:       0,
		Title:          "QA pass",
		Priority:       domain.PriorityMedium,
		LifecycleState: domain.StateProgress,
	})
	fixture.storeNodeContract(t, actionItem, domain.NodeContractSnapshotInput{
		ResponsibleActorKind:    domain.TemplateActorKindQA,
		EditableByActorKinds:    []domain.TemplateActorKind{domain.TemplateActorKindQA},
		CompletableByActorKinds: []domain.TemplateActorKind{domain.TemplateActorKindQA},
	})

	ctx := fixture.leaseContext(t, domain.CapabilityScopeActionItem, actionItem.ID, domain.CapabilityRoleBuilder, "builder-1")
	_, err := fixture.svc.UpdateActionItem(ctx, UpdateActionItemInput{
		ActionItemID: actionItem.ID,
		Title:        actionItem.Title,
		Description:  "builder edit",
		UpdatedBy:    "builder-1",
		UpdatedType:  domain.ActorTypeAgent,
	})
	if !errors.Is(err, domain.ErrNodeContractForbidden) {
		t.Fatalf("UpdateActionItem() error = %v, want ErrNodeContractForbidden", err)
	}
}

// TestUpdateActionItemAllowsHumanEditForGeneratedNode verifies human edits remain allowed even when the generated-node contract is role-restricted.
func TestUpdateActionItemAllowsHumanEditForGeneratedNode(t *testing.T) {
	fixture := newTemplateContractFixture(t)
	actionItem := fixture.storeActionItem(t, domain.ActionItemInput{
		ID:             "actionItem-qa",
		ProjectID:      fixture.project.ID,
		ColumnID:       fixture.progress.ID,
		Position:       0,
		Title:          "QA pass",
		Priority:       domain.PriorityMedium,
		LifecycleState: domain.StateProgress,
	})
	fixture.storeNodeContract(t, actionItem, domain.NodeContractSnapshotInput{
		ResponsibleActorKind:    domain.TemplateActorKindQA,
		EditableByActorKinds:    []domain.TemplateActorKind{domain.TemplateActorKindQA},
		CompletableByActorKinds: []domain.TemplateActorKind{domain.TemplateActorKindQA},
	})

	updated, err := fixture.svc.UpdateActionItem(context.Background(), UpdateActionItemInput{
		ActionItemID: actionItem.ID,
		Title:        actionItem.Title,
		Description:  "human edit",
		UpdatedBy:    "user-1",
		UpdatedType:  domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("UpdateActionItem() error = %v", err)
	}
	if updated.Description != "human edit" {
		t.Fatalf("updated.Description = %q, want human edit", updated.Description)
	}
}

// TestMoveActionItemBlocksGeneratedNodeCompletionForWrongActorKind verifies generated-node completion is limited to the contract completer kinds.
func TestMoveActionItemBlocksGeneratedNodeCompletionForWrongActorKind(t *testing.T) {
	fixture := newTemplateContractFixture(t)
	actionItem := fixture.storeActionItem(t, domain.ActionItemInput{
		ID:             "actionItem-qa",
		ProjectID:      fixture.project.ID,
		ColumnID:       fixture.progress.ID,
		Position:       0,
		Title:          "QA pass",
		Priority:       domain.PriorityMedium,
		LifecycleState: domain.StateProgress,
	})
	fixture.storeNodeContract(t, actionItem, domain.NodeContractSnapshotInput{
		ResponsibleActorKind:    domain.TemplateActorKindQA,
		EditableByActorKinds:    []domain.TemplateActorKind{domain.TemplateActorKindQA},
		CompletableByActorKinds: []domain.TemplateActorKind{domain.TemplateActorKindQA},
	})

	ctx := fixture.leaseContext(t, domain.CapabilityScopeActionItem, actionItem.ID, domain.CapabilityRoleBuilder, "builder-1")
	_, err := fixture.svc.MoveActionItem(ctx, actionItem.ID, fixture.done.ID, 0)
	if !errors.Is(err, domain.ErrNodeContractForbidden) {
		t.Fatalf("MoveActionItem() error = %v, want ErrNodeContractForbidden", err)
	}
}

// TestCreateActionItemBlocksChildCreationUnderGeneratedNodeForWrongActorKind verifies create-child cannot bypass generated-node edit ownership.
func TestCreateActionItemBlocksChildCreationUnderGeneratedNodeForWrongActorKind(t *testing.T) {
	fixture := newTemplateContractFixture(t)
	parent := fixture.storeActionItem(t, domain.ActionItemInput{
		ID:             "actionItem-qa",
		ProjectID:      fixture.project.ID,
		ColumnID:       fixture.progress.ID,
		Position:       0,
		Title:          "QA pass",
		Priority:       domain.PriorityMedium,
		LifecycleState: domain.StateProgress,
	})
	fixture.storeNodeContract(t, parent, domain.NodeContractSnapshotInput{
		ResponsibleActorKind:    domain.TemplateActorKindQA,
		EditableByActorKinds:    []domain.TemplateActorKind{domain.TemplateActorKindQA},
		CompletableByActorKinds: []domain.TemplateActorKind{domain.TemplateActorKindQA},
	})

	ctx := fixture.leaseContext(t, domain.CapabilityScopeActionItem, parent.ID, domain.CapabilityRoleBuilder, "builder-1")
	_, err := fixture.svc.CreateActionItem(ctx, CreateActionItemInput{
		ProjectID:      fixture.project.ID,
		ParentID:       parent.ID,
		ColumnID:       fixture.progress.ID,
		Title:          "Builder child",
		Kind:           domain.WorkKindSubtask,
		UpdatedByActor: "builder-1",
		UpdatedByName:  "builder-1",
		UpdatedByType:  domain.ActorTypeAgent,
	})
	if !errors.Is(err, domain.ErrNodeContractForbidden) {
		t.Fatalf("CreateActionItem() error = %v, want ErrNodeContractForbidden", err)
	}
}

// TestMoveActionItemAllowsOrchestratorOverrideWhenContractPermits verifies the stored orchestrator override flag is honored.
func TestMoveActionItemAllowsOrchestratorOverrideWhenContractPermits(t *testing.T) {
	fixture := newTemplateContractFixture(t)
	actionItem := fixture.storeActionItem(t, domain.ActionItemInput{
		ID:             "actionItem-qa",
		ProjectID:      fixture.project.ID,
		ColumnID:       fixture.progress.ID,
		Position:       0,
		Title:          "QA pass",
		Priority:       domain.PriorityMedium,
		LifecycleState: domain.StateProgress,
	})
	fixture.storeNodeContract(t, actionItem, domain.NodeContractSnapshotInput{
		ResponsibleActorKind:    domain.TemplateActorKindQA,
		EditableByActorKinds:    []domain.TemplateActorKind{domain.TemplateActorKindQA},
		CompletableByActorKinds: []domain.TemplateActorKind{domain.TemplateActorKindQA},
		OrchestratorMayComplete: true,
	})

	ctx := fixture.leaseContext(t, domain.CapabilityScopeActionItem, actionItem.ID, domain.CapabilityRoleOrchestrator, "orch-1")
	moved, err := fixture.svc.MoveActionItem(ctx, actionItem.ID, fixture.done.ID, 0)
	if err != nil {
		t.Fatalf("MoveActionItem() error = %v", err)
	}
	if moved.LifecycleState != domain.StateDone {
		t.Fatalf("moved.LifecycleState = %q, want done", moved.LifecycleState)
	}
}

// TestMoveActionItemAllowsDoneWithOptionalIncompleteChild verifies optional incomplete children no longer block done by default.
func TestMoveActionItemAllowsDoneWithOptionalIncompleteChild(t *testing.T) {
	fixture := newTemplateContractFixture(t)
	parent := fixture.storeActionItem(t, domain.ActionItemInput{
		ID:             "actionItem-parent",
		ProjectID:      fixture.project.ID,
		ColumnID:       fixture.progress.ID,
		Position:       0,
		Title:          "Parent",
		Priority:       domain.PriorityHigh,
		LifecycleState: domain.StateProgress,
	})
	child := fixture.storeActionItem(t, domain.ActionItemInput{
		ID:             "actionItem-child",
		ProjectID:      fixture.project.ID,
		ParentID:       parent.ID,
		ColumnID:       fixture.progress.ID,
		Position:       1,
		Title:          "Optional child",
		Priority:       domain.PriorityLow,
		LifecycleState: domain.StateProgress,
	})
	fixture.storeNodeContract(t, child, domain.NodeContractSnapshotInput{
		ResponsibleActorKind:    domain.TemplateActorKindQA,
		EditableByActorKinds:    []domain.TemplateActorKind{domain.TemplateActorKindQA},
		CompletableByActorKinds: []domain.TemplateActorKind{domain.TemplateActorKindQA},
	})

	moved, err := fixture.svc.MoveActionItem(context.Background(), parent.ID, fixture.done.ID, 0)
	if err != nil {
		t.Fatalf("MoveActionItem() error = %v", err)
	}
	if moved.LifecycleState != domain.StateDone {
		t.Fatalf("moved.LifecycleState = %q, want done", moved.LifecycleState)
	}
}

// TestMoveActionItemBlocksDoneWhenRequiredParentContractChildOpen verifies required direct-child blockers stop parent completion.
func TestMoveActionItemBlocksDoneWhenRequiredParentContractChildOpen(t *testing.T) {
	fixture := newTemplateContractFixture(t)
	parent := fixture.storeActionItem(t, domain.ActionItemInput{
		ID:             "actionItem-parent",
		ProjectID:      fixture.project.ID,
		ColumnID:       fixture.progress.ID,
		Position:       0,
		Title:          "Parent",
		Priority:       domain.PriorityHigh,
		LifecycleState: domain.StateProgress,
	})
	child := fixture.storeActionItem(t, domain.ActionItemInput{
		ID:             "actionItem-child",
		ProjectID:      fixture.project.ID,
		ParentID:       parent.ID,
		ColumnID:       fixture.progress.ID,
		Position:       1,
		Title:          "QA blocker",
		Priority:       domain.PriorityLow,
		LifecycleState: domain.StateProgress,
	})
	fixture.storeNodeContract(t, child, domain.NodeContractSnapshotInput{
		ResponsibleActorKind:    domain.TemplateActorKindQA,
		EditableByActorKinds:    []domain.TemplateActorKind{domain.TemplateActorKindQA},
		CompletableByActorKinds: []domain.TemplateActorKind{domain.TemplateActorKindQA},
		RequiredForParentDone:   true,
	})

	_, err := fixture.svc.MoveActionItem(context.Background(), parent.ID, fixture.done.ID, 0)
	if !errors.Is(err, domain.ErrTransitionBlocked) {
		t.Fatalf("MoveActionItem() error = %v, want ErrTransitionBlocked", err)
	}
	if !strings.Contains(err.Error(), "parent blocker") {
		t.Fatalf("MoveActionItem() error = %v, want parent blocker detail", err)
	}
}

// TestMoveActionItemBlocksDoneWhenRequiredContainingContractDescendantOpen verifies containing-scope blockers stop ancestor completion.
func TestMoveActionItemBlocksDoneWhenRequiredContainingContractDescendantOpen(t *testing.T) {
	fixture := newTemplateContractFixture(t)
	phase := fixture.storeActionItem(t, domain.ActionItemInput{
		ID:             "phase-1",
		ProjectID:      fixture.project.ID,
		ColumnID:       fixture.progress.ID,
		Position:       0,
		Title:          "Phase",
		Priority:       domain.PriorityHigh,
		Scope:          domain.KindAppliesToPhase,
		Kind:           domain.WorkKindPhase,
		LifecycleState: domain.StateProgress,
	})
	actionItem := fixture.storeActionItem(t, domain.ActionItemInput{
		ID:             "actionItem-1",
		ProjectID:      fixture.project.ID,
		ParentID:       phase.ID,
		ColumnID:       fixture.done.ID,
		Position:       1,
		Title:          "Build actionItem",
		Priority:       domain.PriorityMedium,
		Kind:           domain.WorkKindActionItem,
		Scope:          domain.KindAppliesToActionItem,
		LifecycleState: domain.StateDone,
	})
	descendant := fixture.storeActionItem(t, domain.ActionItemInput{
		ID:             "subtask-qa",
		ProjectID:      fixture.project.ID,
		ParentID:       actionItem.ID,
		ColumnID:       fixture.progress.ID,
		Position:       2,
		Title:          "Phase QA",
		Priority:       domain.PriorityLow,
		Kind:           domain.WorkKindSubtask,
		Scope:          domain.KindAppliesToSubtask,
		LifecycleState: domain.StateProgress,
	})
	fixture.storeNodeContract(t, descendant, domain.NodeContractSnapshotInput{
		ResponsibleActorKind:      domain.TemplateActorKindQA,
		EditableByActorKinds:      []domain.TemplateActorKind{domain.TemplateActorKindQA},
		CompletableByActorKinds:   []domain.TemplateActorKind{domain.TemplateActorKindQA},
		RequiredForContainingDone: true,
	})

	_, err := fixture.svc.MoveActionItem(context.Background(), phase.ID, fixture.done.ID, 0)
	if !errors.Is(err, domain.ErrTransitionBlocked) {
		t.Fatalf("MoveActionItem() error = %v, want ErrTransitionBlocked", err)
	}
	if !strings.Contains(err.Error(), "containing scope blocker") {
		t.Fatalf("MoveActionItem() error = %v, want containing scope blocker detail", err)
	}
}
