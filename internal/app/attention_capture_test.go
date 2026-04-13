package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/hylla/tillsyn/internal/domain"
)

// TestServiceRaiseListResolveAttentionItem verifies attention lifecycle APIs on the app service.
func TestServiceRaiseListResolveAttentionItem(t *testing.T) {
	repo := newFakeRepo()
	ids := []string{"p1", "c1", "t1", "attn-1"}
	nextID := 0
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	svc := NewService(repo, func() string {
		id := ids[nextID]
		nextID++
		return id
	}, func() time.Time {
		return now
	}, ServiceConfig{})

	project, err := svc.CreateProject(context.Background(), "Attention", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := svc.CreateColumn(context.Background(), project.ID, "In Progress", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	task, err := svc.CreateTask(context.Background(), CreateTaskInput{
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Title:          "Investigate failure",
		Priority:       domain.PriorityHigh,
		UpdatedByType:  domain.ActorTypeUser,
		UpdatedByActor: "user-1",
		CreatedByActor: "user-1",
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	created, err := svc.RaiseAttentionItem(context.Background(), RaiseAttentionItemInput{
		Level: domain.LevelTupleInput{
			ProjectID: project.ID,
			ScopeType: domain.ScopeLevelTask,
			ScopeID:   task.ID,
		},
		Kind:               domain.AttentionKindBlocker,
		Summary:            "Need user decision",
		RequiresUserAction: true,
		CreatedBy:          "user-1",
		CreatedType:        domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("RaiseAttentionItem() error = %v", err)
	}
	if created.State != domain.AttentionStateOpen {
		t.Fatalf("expected open attention state, got %q", created.State)
	}

	unresolved, err := svc.ListAttentionItems(context.Background(), ListAttentionItemsInput{
		Level: domain.LevelTupleInput{
			ProjectID: project.ID,
			ScopeType: domain.ScopeLevelTask,
			ScopeID:   task.ID,
		},
		UnresolvedOnly: true,
	})
	if err != nil {
		t.Fatalf("ListAttentionItems(unresolved) error = %v", err)
	}
	if len(unresolved) != 1 || unresolved[0].ID != created.ID {
		t.Fatalf("expected one unresolved attention item, got %#v", unresolved)
	}

	resolved, err := svc.ResolveAttentionItem(context.Background(), ResolveAttentionItemInput{
		AttentionID:  created.ID,
		ResolvedBy:   "user-2",
		ResolvedType: domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("ResolveAttentionItem() error = %v", err)
	}
	if resolved.State != domain.AttentionStateResolved || resolved.ResolvedAt == nil {
		t.Fatalf("expected resolved attention item, got %#v", resolved)
	}
}

// TestServiceListAttentionItemsWaitsForLiveChange verifies attention list wait_timeout resumes on the next project-scoped inbox change.
func TestServiceListAttentionItemsWaitsForLiveChange(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 4, 2, 10, 15, 0, 0, time.UTC)
	svc := NewService(repo, func() string { return "attn-live" }, func() time.Time { return now }, ServiceConfig{})

	project, err := svc.CreateProject(context.Background(), "Attention Wait", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	resultCh := make(chan []domain.AttentionItem, 1)
	errCh := make(chan error, 1)
	go func() {
		items, listErr := svc.ListAttentionItems(context.Background(), ListAttentionItemsInput{
			Level: domain.LevelTupleInput{
				ProjectID: project.ID,
				ScopeType: domain.ScopeLevelProject,
				ScopeID:   project.ID,
			},
			UnresolvedOnly: true,
			WaitTimeout:    time.Second,
		})
		if listErr != nil {
			errCh <- listErr
			return
		}
		resultCh <- items
	}()

	select {
	case got := <-resultCh:
		t.Fatalf("ListAttentionItems() returned early with %#v before a live change", got)
	case err := <-errCh:
		t.Fatalf("ListAttentionItems() early error = %v", err)
	case <-time.After(25 * time.Millisecond):
	}

	if _, err := svc.RaiseAttentionItem(context.Background(), RaiseAttentionItemInput{
		Level: domain.LevelTupleInput{
			ProjectID: project.ID,
			ScopeType: domain.ScopeLevelProject,
			ScopeID:   project.ID,
		},
		Kind:               domain.AttentionKindRiskNote,
		Summary:            "Need orchestration follow-up",
		RequiresUserAction: false,
		CreatedBy:          "user-1",
		CreatedType:        domain.ActorTypeUser,
	}); err != nil {
		t.Fatalf("RaiseAttentionItem() error = %v", err)
	}

	select {
	case err := <-errCh:
		t.Fatalf("ListAttentionItems() error = %v", err)
	case items := <-resultCh:
		if len(items) != 1 || items[0].ID != "attn-live" {
			t.Fatalf("ListAttentionItems() = %#v, want attn-live after wake", items)
		}
	case <-time.After(time.Second):
		t.Fatal("ListAttentionItems() did not wake after a live attention change")
	}
}

// TestRaiseAttentionItemValidatesScopeEntityConsistency verifies scope_type/scope_id tuple validation.
func TestRaiseAttentionItemValidatesScopeEntityConsistency(t *testing.T) {
	repo := newFakeRepo()
	ids := []string{"p1", "c1", "t1", "attn-project", "attn-task"}
	nextID := 0
	now := time.Date(2026, 2, 24, 12, 30, 0, 0, time.UTC)
	svc := NewService(repo, func() string {
		id := ids[nextID]
		nextID++
		return id
	}, func() time.Time {
		return now
	}, ServiceConfig{})

	project, err := svc.CreateProject(context.Background(), "Attention tuple validation", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := svc.CreateColumn(context.Background(), project.ID, "In Progress", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	task, err := svc.CreateTask(context.Background(), CreateTaskInput{
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Title:          "Validate attention scope tuple",
		Priority:       domain.PriorityHigh,
		UpdatedByType:  domain.ActorTypeUser,
		UpdatedByActor: "user-1",
		CreatedByActor: "user-1",
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	_, err = svc.RaiseAttentionItem(context.Background(), RaiseAttentionItemInput{
		Level: domain.LevelTupleInput{
			ProjectID: project.ID,
			ScopeType: domain.ScopeLevelTask,
			ScopeID:   project.ID,
		},
		Kind:               domain.AttentionKindBlocker,
		Summary:            "Invalid tuple",
		RequiresUserAction: true,
		CreatedBy:          "user-1",
		CreatedType:        domain.ActorTypeUser,
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for mismatched task tuple, got %v", err)
	}

	projectScoped, err := svc.RaiseAttentionItem(context.Background(), RaiseAttentionItemInput{
		Level: domain.LevelTupleInput{
			ProjectID: project.ID,
			ScopeType: domain.ScopeLevelProject,
			ScopeID:   project.ID,
		},
		Kind:               domain.AttentionKindRiskNote,
		Summary:            "Project-scoped attention",
		RequiresUserAction: false,
		CreatedBy:          "user-1",
		CreatedType:        domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("RaiseAttentionItem(project) error = %v", err)
	}
	if projectScoped.ScopeType != domain.ScopeLevelProject || projectScoped.ScopeID != project.ID {
		t.Fatalf("unexpected project-scoped attention tuple %#v", projectScoped)
	}

	taskScoped, err := svc.RaiseAttentionItem(context.Background(), RaiseAttentionItemInput{
		Level: domain.LevelTupleInput{
			ProjectID: project.ID,
			ScopeType: domain.ScopeLevelTask,
			ScopeID:   task.ID,
		},
		Kind:               domain.AttentionKindBlocker,
		Summary:            "Task-scoped attention",
		RequiresUserAction: true,
		CreatedBy:          "user-1",
		CreatedType:        domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("RaiseAttentionItem(task) error = %v", err)
	}
	if taskScoped.ScopeType != domain.ScopeLevelTask || taskScoped.ScopeID != task.ID {
		t.Fatalf("unexpected task-scoped attention tuple %#v", taskScoped)
	}
}

// TestMoveTaskBlocksDoneWhenBlockingAttentionUnresolved verifies completion transition guard behavior.
func TestMoveTaskBlocksDoneWhenBlockingAttentionUnresolved(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 24, 13, 0, 0, 0, time.UTC)
	svc := NewService(repo, nil, func() time.Time { return now }, ServiceConfig{})

	project, _ := domain.NewProject("p1", "Guardrails", "", now)
	repo.projects[project.ID] = project
	progress, _ := domain.NewColumn("c-progress", project.ID, "In Progress", 0, 0, now)
	done, _ := domain.NewColumn("c-done", project.ID, "Done", 1, 0, now)
	repo.columns[progress.ID] = progress
	repo.columns[done.ID] = done

	task, _ := domain.NewTask(domain.TaskInput{
		ID:             "t1",
		ProjectID:      project.ID,
		Scope:          domain.KindAppliesToTask,
		ColumnID:       progress.ID,
		Position:       0,
		Title:          "Ship release",
		Priority:       domain.PriorityHigh,
		LifecycleState: domain.StateProgress,
	}, now)
	repo.tasks[task.ID] = task

	attention, err := domain.NewAttentionItem(domain.AttentionItemInput{
		ID:                 "attn-1",
		ProjectID:          project.ID,
		ScopeType:          domain.ScopeLevelTask,
		ScopeID:            task.ID,
		Kind:               domain.AttentionKindRiskNote,
		Summary:            "User must approve rollback plan",
		RequiresUserAction: true,
	}, now)
	if err != nil {
		t.Fatalf("NewAttentionItem() error = %v", err)
	}
	repo.attentionItems[attention.ID] = attention

	_, err = svc.MoveTask(context.Background(), task.ID, done.ID, 0)
	if !errors.Is(err, domain.ErrTransitionBlocked) {
		t.Fatalf("expected ErrTransitionBlocked, got %v", err)
	}
	if !strings.Contains(err.Error(), "unresolved attention") {
		t.Fatalf("expected unresolved attention reason, got %v", err)
	}
}

// TestCaptureStateSummary verifies summary-first capture-state output fields.
func TestCaptureStateSummary(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 24, 14, 0, 0, 0, time.UTC)
	ids := []string{"p1", "c1", "t1", "attn-1"}
	nextID := 0
	svc := NewService(repo, func() string {
		id := ids[nextID]
		nextID++
		return id
	}, func() time.Time {
		return now
	}, ServiceConfig{})

	project, err := svc.CreateProject(context.Background(), "Capture", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := svc.CreateColumn(context.Background(), project.ID, "In Progress", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	task, err := svc.CreateTask(context.Background(), CreateTaskInput{
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Title:     "Focus task",
		Priority:  domain.PriorityMedium,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if _, err := svc.RaiseAttentionItem(context.Background(), RaiseAttentionItemInput{
		Level: domain.LevelTupleInput{
			ProjectID: project.ID,
			ScopeType: domain.ScopeLevelTask,
			ScopeID:   task.ID,
		},
		Kind:               domain.AttentionKindBlocker,
		Summary:            "Need sign-off",
		RequiresUserAction: true,
	}); err != nil {
		t.Fatalf("RaiseAttentionItem() error = %v", err)
	}

	captured, err := svc.CaptureState(context.Background(), CaptureStateInput{
		Level: domain.LevelTupleInput{
			ProjectID: project.ID,
			ScopeType: domain.ScopeLevelTask,
			ScopeID:   task.ID,
		},
		View: CaptureStateViewSummary,
	})
	if err != nil {
		t.Fatalf("CaptureState() error = %v", err)
	}
	if captured.Level.ScopeType != domain.ScopeLevelTask || captured.Level.ScopeID != task.ID {
		t.Fatalf("unexpected level tuple %#v", captured.Level)
	}
	if captured.AttentionOverview.UnresolvedCount != 1 || captured.AttentionOverview.BlockingCount != 1 {
		t.Fatalf("unexpected attention overview %#v", captured.AttentionOverview)
	}
	if captured.WorkOverview.TotalItems != 1 || captured.WorkOverview.ActiveItems != 1 {
		t.Fatalf("unexpected work overview %#v", captured.WorkOverview)
	}
	if captured.FollowUpPointers.ListAttentionItems == "" || captured.FollowUpPointers.ListProjectChangeEvents == "" {
		t.Fatalf("expected follow-up pointers, got %#v", captured.FollowUpPointers)
	}
}

// TestBuildCaptureStateWorkOverviewCountsFailedItems verifies that failed items are counted in the work overview.
func TestBuildCaptureStateWorkOverviewCountsFailedItems(t *testing.T) {
	level := domain.LevelTuple{
		ProjectID: "p1",
		ScopeType: domain.ScopeLevelProject,
		ScopeID:   "p1",
	}
	tasks := []domain.Task{
		{ID: "t1", ProjectID: "p1", LifecycleState: domain.StateTodo},
		{ID: "t2", ProjectID: "p1", LifecycleState: domain.StateProgress},
		{ID: "t3", ProjectID: "p1", LifecycleState: domain.StateDone},
		{ID: "t4", ProjectID: "p1", LifecycleState: domain.StateFailed},
		{ID: "t5", ProjectID: "p1", LifecycleState: domain.StateFailed},
	}
	overview := buildCaptureStateWorkOverview(level, tasks)
	if overview.FailedItems != 2 {
		t.Fatalf("FailedItems = %d, want 2", overview.FailedItems)
	}
	if overview.DoneItems != 1 {
		t.Fatalf("DoneItems = %d, want 1", overview.DoneItems)
	}
	if overview.InProgressItems != 1 {
		t.Fatalf("InProgressItems = %d, want 1", overview.InProgressItems)
	}
	if overview.TotalItems != 5 {
		t.Fatalf("TotalItems = %d, want 5", overview.TotalItems)
	}
}
