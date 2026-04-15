package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// templateContractFixture stores common project/task state for node-contract enforcement tests.
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

// storeTask persists one task directly into the fake repository for a focused service test.
func (f templateContractFixture) storeTask(t *testing.T, in domain.TaskInput) domain.Task {
	t.Helper()

	task, err := domain.NewTask(in, f.now)
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}
	f.repo.tasks[task.ID] = task
	return task
}

// storeNodeContract persists one generated-node contract snapshot for the provided task.
func (f templateContractFixture) storeNodeContract(t *testing.T, task domain.Task, in domain.NodeContractSnapshotInput) {
	t.Helper()

	in.NodeID = task.ID
	in.ProjectID = task.ProjectID
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
	f.repo.nodeContracts[task.ID] = snapshot
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

// TestUpdateTaskBlocksGeneratedNodeEditForWrongActorKind verifies generated-node edit permissions fail closed for the wrong actor kind.
func TestUpdateTaskBlocksGeneratedNodeEditForWrongActorKind(t *testing.T) {
	fixture := newTemplateContractFixture(t)
	task := fixture.storeTask(t, domain.TaskInput{
		ID:             "task-qa",
		ProjectID:      fixture.project.ID,
		ColumnID:       fixture.progress.ID,
		Position:       0,
		Title:          "QA pass",
		Priority:       domain.PriorityMedium,
		LifecycleState: domain.StateProgress,
	})
	fixture.storeNodeContract(t, task, domain.NodeContractSnapshotInput{
		ResponsibleActorKind:    domain.TemplateActorKindQA,
		EditableByActorKinds:    []domain.TemplateActorKind{domain.TemplateActorKindQA},
		CompletableByActorKinds: []domain.TemplateActorKind{domain.TemplateActorKindQA},
	})

	ctx := fixture.leaseContext(t, domain.CapabilityScopeTask, task.ID, domain.CapabilityRoleBuilder, "builder-1")
	_, err := fixture.svc.UpdateTask(ctx, UpdateTaskInput{
		TaskID:      task.ID,
		Title:       task.Title,
		Description: "builder edit",
		UpdatedBy:   "builder-1",
		UpdatedType: domain.ActorTypeAgent,
	})
	if !errors.Is(err, domain.ErrNodeContractForbidden) {
		t.Fatalf("UpdateTask() error = %v, want ErrNodeContractForbidden", err)
	}
}

// TestUpdateTaskAllowsHumanEditForGeneratedNode verifies human edits remain allowed even when the generated-node contract is role-restricted.
func TestUpdateTaskAllowsHumanEditForGeneratedNode(t *testing.T) {
	fixture := newTemplateContractFixture(t)
	task := fixture.storeTask(t, domain.TaskInput{
		ID:             "task-qa",
		ProjectID:      fixture.project.ID,
		ColumnID:       fixture.progress.ID,
		Position:       0,
		Title:          "QA pass",
		Priority:       domain.PriorityMedium,
		LifecycleState: domain.StateProgress,
	})
	fixture.storeNodeContract(t, task, domain.NodeContractSnapshotInput{
		ResponsibleActorKind:    domain.TemplateActorKindQA,
		EditableByActorKinds:    []domain.TemplateActorKind{domain.TemplateActorKindQA},
		CompletableByActorKinds: []domain.TemplateActorKind{domain.TemplateActorKindQA},
	})

	updated, err := fixture.svc.UpdateTask(context.Background(), UpdateTaskInput{
		TaskID:      task.ID,
		Title:       task.Title,
		Description: "human edit",
		UpdatedBy:   "user-1",
		UpdatedType: domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("UpdateTask() error = %v", err)
	}
	if updated.Description != "human edit" {
		t.Fatalf("updated.Description = %q, want human edit", updated.Description)
	}
}

// TestMoveTaskBlocksGeneratedNodeCompletionForWrongActorKind verifies generated-node completion is limited to the contract completer kinds.
func TestMoveTaskBlocksGeneratedNodeCompletionForWrongActorKind(t *testing.T) {
	fixture := newTemplateContractFixture(t)
	task := fixture.storeTask(t, domain.TaskInput{
		ID:             "task-qa",
		ProjectID:      fixture.project.ID,
		ColumnID:       fixture.progress.ID,
		Position:       0,
		Title:          "QA pass",
		Priority:       domain.PriorityMedium,
		LifecycleState: domain.StateProgress,
	})
	fixture.storeNodeContract(t, task, domain.NodeContractSnapshotInput{
		ResponsibleActorKind:    domain.TemplateActorKindQA,
		EditableByActorKinds:    []domain.TemplateActorKind{domain.TemplateActorKindQA},
		CompletableByActorKinds: []domain.TemplateActorKind{domain.TemplateActorKindQA},
	})

	ctx := fixture.leaseContext(t, domain.CapabilityScopeTask, task.ID, domain.CapabilityRoleBuilder, "builder-1")
	_, err := fixture.svc.MoveTask(ctx, task.ID, fixture.done.ID, 0)
	if !errors.Is(err, domain.ErrNodeContractForbidden) {
		t.Fatalf("MoveTask() error = %v, want ErrNodeContractForbidden", err)
	}
}

// TestCreateTaskBlocksChildCreationUnderGeneratedNodeForWrongActorKind verifies create-child cannot bypass generated-node edit ownership.
func TestCreateTaskBlocksChildCreationUnderGeneratedNodeForWrongActorKind(t *testing.T) {
	fixture := newTemplateContractFixture(t)
	parent := fixture.storeTask(t, domain.TaskInput{
		ID:             "task-qa",
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

	ctx := fixture.leaseContext(t, domain.CapabilityScopeTask, parent.ID, domain.CapabilityRoleBuilder, "builder-1")
	_, err := fixture.svc.CreateTask(ctx, CreateTaskInput{
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
		t.Fatalf("CreateTask() error = %v, want ErrNodeContractForbidden", err)
	}
}

// TestMoveTaskAllowsOrchestratorOverrideWhenContractPermits verifies the stored orchestrator override flag is honored.
func TestMoveTaskAllowsOrchestratorOverrideWhenContractPermits(t *testing.T) {
	fixture := newTemplateContractFixture(t)
	task := fixture.storeTask(t, domain.TaskInput{
		ID:             "task-qa",
		ProjectID:      fixture.project.ID,
		ColumnID:       fixture.progress.ID,
		Position:       0,
		Title:          "QA pass",
		Priority:       domain.PriorityMedium,
		LifecycleState: domain.StateProgress,
	})
	fixture.storeNodeContract(t, task, domain.NodeContractSnapshotInput{
		ResponsibleActorKind:    domain.TemplateActorKindQA,
		EditableByActorKinds:    []domain.TemplateActorKind{domain.TemplateActorKindQA},
		CompletableByActorKinds: []domain.TemplateActorKind{domain.TemplateActorKindQA},
		OrchestratorMayComplete: true,
	})

	ctx := fixture.leaseContext(t, domain.CapabilityScopeTask, task.ID, domain.CapabilityRoleOrchestrator, "orch-1")
	moved, err := fixture.svc.MoveTask(ctx, task.ID, fixture.done.ID, 0)
	if err != nil {
		t.Fatalf("MoveTask() error = %v", err)
	}
	if moved.LifecycleState != domain.StateDone {
		t.Fatalf("moved.LifecycleState = %q, want done", moved.LifecycleState)
	}
}

// TestMoveTaskAllowsDoneWithOptionalIncompleteChild verifies optional incomplete children no longer block done by default.
func TestMoveTaskAllowsDoneWithOptionalIncompleteChild(t *testing.T) {
	fixture := newTemplateContractFixture(t)
	parent := fixture.storeTask(t, domain.TaskInput{
		ID:             "task-parent",
		ProjectID:      fixture.project.ID,
		ColumnID:       fixture.progress.ID,
		Position:       0,
		Title:          "Parent",
		Priority:       domain.PriorityHigh,
		LifecycleState: domain.StateProgress,
	})
	child := fixture.storeTask(t, domain.TaskInput{
		ID:             "task-child",
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

	moved, err := fixture.svc.MoveTask(context.Background(), parent.ID, fixture.done.ID, 0)
	if err != nil {
		t.Fatalf("MoveTask() error = %v", err)
	}
	if moved.LifecycleState != domain.StateDone {
		t.Fatalf("moved.LifecycleState = %q, want done", moved.LifecycleState)
	}
}

// TestMoveTaskBlocksDoneWhenRequiredParentContractChildOpen verifies required direct-child blockers stop parent completion.
func TestMoveTaskBlocksDoneWhenRequiredParentContractChildOpen(t *testing.T) {
	fixture := newTemplateContractFixture(t)
	parent := fixture.storeTask(t, domain.TaskInput{
		ID:             "task-parent",
		ProjectID:      fixture.project.ID,
		ColumnID:       fixture.progress.ID,
		Position:       0,
		Title:          "Parent",
		Priority:       domain.PriorityHigh,
		LifecycleState: domain.StateProgress,
	})
	child := fixture.storeTask(t, domain.TaskInput{
		ID:             "task-child",
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

	_, err := fixture.svc.MoveTask(context.Background(), parent.ID, fixture.done.ID, 0)
	if !errors.Is(err, domain.ErrTransitionBlocked) {
		t.Fatalf("MoveTask() error = %v, want ErrTransitionBlocked", err)
	}
	if !strings.Contains(err.Error(), "parent blocker") {
		t.Fatalf("MoveTask() error = %v, want parent blocker detail", err)
	}
}

// TestMoveTaskBlocksDoneWhenRequiredContainingContractDescendantOpen verifies containing-scope blockers stop ancestor completion.
func TestMoveTaskBlocksDoneWhenRequiredContainingContractDescendantOpen(t *testing.T) {
	fixture := newTemplateContractFixture(t)
	phase := fixture.storeTask(t, domain.TaskInput{
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
	task := fixture.storeTask(t, domain.TaskInput{
		ID:             "task-1",
		ProjectID:      fixture.project.ID,
		ParentID:       phase.ID,
		ColumnID:       fixture.done.ID,
		Position:       1,
		Title:          "Build task",
		Priority:       domain.PriorityMedium,
		Kind:           domain.WorkKindTask,
		Scope:          domain.KindAppliesToTask,
		LifecycleState: domain.StateDone,
	})
	descendant := fixture.storeTask(t, domain.TaskInput{
		ID:             "subtask-qa",
		ProjectID:      fixture.project.ID,
		ParentID:       task.ID,
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

	_, err := fixture.svc.MoveTask(context.Background(), phase.ID, fixture.done.ID, 0)
	if !errors.Is(err, domain.ErrTransitionBlocked) {
		t.Fatalf("MoveTask() error = %v, want ErrTransitionBlocked", err)
	}
	if !strings.Contains(err.Error(), "containing scope blocker") {
		t.Fatalf("MoveTask() error = %v, want containing scope blocker detail", err)
	}
}
