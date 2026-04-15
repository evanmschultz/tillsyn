package common

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// fakeCaptureReadModel provides deterministic capture-state read data for tests.
type fakeCaptureReadModel struct {
	projects        []domain.Project
	columns         []domain.Column
	tasks           []domain.Task
	comments        []domain.Comment
	listProjectsErr error
	listColumnsErr  error
	listTasksErr    error
	listCommentsErr error
}

// ListProjects returns configured projects or one injected error.
func (f fakeCaptureReadModel) ListProjects(_ context.Context, _ bool) ([]domain.Project, error) {
	if f.listProjectsErr != nil {
		return nil, f.listProjectsErr
	}
	return append([]domain.Project(nil), f.projects...), nil
}

// ListColumns returns configured columns or one injected error.
func (f fakeCaptureReadModel) ListColumns(_ context.Context, _ string, _ bool) ([]domain.Column, error) {
	if f.listColumnsErr != nil {
		return nil, f.listColumnsErr
	}
	return append([]domain.Column(nil), f.columns...), nil
}

// ListTasks returns configured tasks or one injected error.
func (f fakeCaptureReadModel) ListTasks(_ context.Context, _ string, _ bool) ([]domain.Task, error) {
	if f.listTasksErr != nil {
		return nil, f.listTasksErr
	}
	return append([]domain.Task(nil), f.tasks...), nil
}

// ListCommentsByTarget returns configured comments or one injected error.
func (f fakeCaptureReadModel) ListCommentsByTarget(_ context.Context, _ domain.CommentTarget) ([]domain.Comment, error) {
	if f.listCommentsErr != nil {
		return nil, f.listCommentsErr
	}
	return append([]domain.Comment(nil), f.comments...), nil
}

// fakeCaptureAttentionService provides deterministic attention rows for capture tests.
type fakeCaptureAttentionService struct {
	items []AttentionItem
	err   error
}

// ListAttentionItems returns configured attention rows or one injected error.
func (f fakeCaptureAttentionService) ListAttentionItems(_ context.Context, _ ListAttentionItemsRequest) ([]AttentionItem, error) {
	if f.err != nil {
		return nil, f.err
	}
	return append([]AttentionItem(nil), f.items...), nil
}

// RaiseAttentionItem is unused in these tests.
func (fakeCaptureAttentionService) RaiseAttentionItem(context.Context, RaiseAttentionItemRequest) (AttentionItem, error) {
	return AttentionItem{}, errors.New("not implemented")
}

// ResolveAttentionItem is unused in these tests.
func (fakeCaptureAttentionService) ResolveAttentionItem(context.Context, ResolveAttentionItemRequest) (AttentionItem, error) {
	return AttentionItem{}, errors.New("not implemented")
}

// TestCaptureStateServiceCaptureStateBuildsSummary verifies deterministic capture summaries, warnings, and scoped comment reads.
func TestCaptureStateServiceCaptureStateBuildsSummary(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 20, 19, 4, 5, 987000000, time.UTC)
	project, err := domain.NewProject("p1", "Inbox", "Dogfood project", now)
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}

	columns := []domain.Column{
		{ID: "c2", ProjectID: project.ID, Name: "Done", Position: 1},
		{ID: "c1", ProjectID: project.ID, Name: "To Do", Position: 0},
	}
	tasks := []domain.Task{
		{
			ID:             "t-archived",
			ProjectID:      project.ID,
			ColumnID:       "c2",
			Position:       4,
			Title:          "Archived",
			LifecycleState: domain.StateArchived,
			ArchivedAt:     ptrTime(now),
		},
		{
			ID:             "t-progress",
			ProjectID:      project.ID,
			ColumnID:       "c1",
			Position:       2,
			Title:          "In progress",
			LifecycleState: domain.StateProgress,
		},
		{
			ID:             "t-parent",
			ProjectID:      project.ID,
			ColumnID:       "c1",
			Position:       0,
			Title:          "Parent",
			LifecycleState: domain.StateTodo,
			Metadata: domain.TaskMetadata{
				BlockedReason: "waiting on child",
				CompletionContract: domain.CompletionContract{
					CompletionCriteria: []domain.ChecklistItem{
						{ID: "cc-1", Text: "capture review", Done: false},
					},
					Policy: domain.CompletionPolicy{RequireChildrenDone: true},
				},
			},
		},
		{
			ID:             "t-done",
			ProjectID:      project.ID,
			ColumnID:       "c2",
			Position:       3,
			Title:          "Done",
			LifecycleState: domain.StateDone,
		},
		{
			ID:             "t-child",
			ProjectID:      project.ID,
			ParentID:       "t-parent",
			ColumnID:       "c1",
			Position:       1,
			Title:          "Child",
			LifecycleState: domain.StateTodo,
		},
		{
			ID:             "t-failed",
			ProjectID:      project.ID,
			ColumnID:       "c1",
			Position:       5,
			Title:          "Failed",
			LifecycleState: domain.StateFailed,
		},
	}
	comments := []domain.Comment{
		{ID: "cm-1", ProjectID: project.ID, TargetType: domain.CommentTargetTypeBranch, TargetID: "b1", BodyMarkdown: "urgent: review this"},
		{ID: "cm-2", ProjectID: project.ID, TargetType: domain.CommentTargetTypeBranch, TargetID: "b1", BodyMarkdown: "plain note"},
	}
	attention := []AttentionItem{
		{ID: "att-2", ProjectID: project.ID, ScopeType: ScopeTypeBranch, ScopeID: "b1", State: AttentionStateOpen, Summary: "later", CreatedAt: now.Add(time.Minute)},
		{ID: "att-1", ProjectID: project.ID, ScopeType: ScopeTypeBranch, ScopeID: "b1", State: AttentionStateOpen, Summary: "first", RequiresUserAction: true, CreatedAt: now},
	}

	service := NewCaptureStateService(fakeCaptureReadModel{
		projects: projectSlice(project),
		columns:  columns,
		tasks:    tasks,
		comments: comments,
	}, fakeCaptureAttentionService{items: attention}, func() time.Time {
		return now
	})

	capture, err := service.CaptureState(context.Background(), CaptureStateRequest{
		ProjectID: project.ID,
		ScopeType: ScopeTypeBranch,
		ScopeID:   "b1",
		View:      "full",
	})
	if err != nil {
		t.Fatalf("CaptureState() error = %v", err)
	}
	if capture.CapturedAt != now.UTC().Truncate(time.Second) {
		t.Fatalf("CapturedAt = %s, want %s", capture.CapturedAt, now.UTC().Truncate(time.Second))
	}
	if len(capture.ScopePath) != 2 || capture.ScopePath[1].ScopeType != ScopeTypeBranch || capture.ScopePath[1].ScopeID != "b1" {
		t.Fatalf("ScopePath = %#v, want project + branch b1", capture.ScopePath)
	}
	if capture.AttentionOverview.OpenCount != 2 || capture.AttentionOverview.RequiresUserAction != 1 {
		t.Fatalf("AttentionOverview = %#v, want open=2 requires=1", capture.AttentionOverview)
	}
	if len(capture.AttentionOverview.Items) != 2 || capture.AttentionOverview.Items[0].ID != "att-1" {
		t.Fatalf("AttentionOverview.Items = %#v, want sorted attention rows", capture.AttentionOverview.Items)
	}
	if capture.WorkOverview.TotalTasks != 6 || capture.WorkOverview.TodoTasks != 2 || capture.WorkOverview.InProgressTasks != 1 || capture.WorkOverview.DoneTasks != 1 || capture.WorkOverview.FailedTasks != 1 || capture.WorkOverview.ArchivedTasks != 1 {
		t.Fatalf("WorkOverview counts = %#v, want todo=2 progress=1 done=1 failed=1 archived=1", capture.WorkOverview)
	}
	if capture.WorkOverview.TasksWithOpenBlockers != 1 || capture.WorkOverview.IncompleteCompletionCriteria != 1 {
		t.Fatalf("WorkOverview blockers = %#v, want one blocker and one incomplete completion criterion", capture.WorkOverview)
	}
	if capture.CommentOverview.RecentCount != 2 || capture.CommentOverview.ImportantCount != 1 {
		t.Fatalf("CommentOverview = %#v, want recent=2 important=1", capture.CommentOverview)
	}
	if len(capture.WarningsOverview.Warnings) != 2 {
		t.Fatalf("WarningsOverview = %#v, want blocker + user-action warnings", capture.WarningsOverview)
	}
	if len(capture.ResumeHints) != 2 || capture.ResumeHints[0].Rel != "till.capture_state" {
		t.Fatalf("ResumeHints = %#v, want default capture/list-attention hints", capture.ResumeHints)
	}
	if strings.TrimSpace(capture.StateHash) == "" {
		t.Fatal("StateHash = empty, want deterministic hash")
	}
}

// TestCaptureStateServiceErrorAndHelperPaths verifies invalid requests, optional surfaces, and helper behavior.
func TestCaptureStateServiceErrorAndHelperPaths(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 20, 19, 5, 0, 0, time.UTC)
	project, err := domain.NewProject("p1", "Inbox", "", now)
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}

	t.Run("unconfigured service", func(t *testing.T) {
		if _, err := (*CaptureStateService)(nil).CaptureState(context.Background(), CaptureStateRequest{ProjectID: "p1"}); !errors.Is(err, ErrInvalidCaptureStateRequest) {
			t.Fatalf("CaptureState(nil) error = %v, want ErrInvalidCaptureStateRequest", err)
		}
	})

	t.Run("normalization", func(t *testing.T) {
		if _, err := normalizeCaptureStateRequest(CaptureStateRequest{}); !errors.Is(err, ErrInvalidCaptureStateRequest) {
			t.Fatalf("normalizeCaptureStateRequest(missing project) error = %v, want invalid request", err)
		}
		if _, err := normalizeCaptureStateRequest(CaptureStateRequest{ProjectID: "p1", ScopeType: "phase"}); !errors.Is(err, ErrUnsupportedScope) {
			t.Fatalf("normalizeCaptureStateRequest(missing scope id) error = %v, want unsupported scope", err)
		}
		if _, err := normalizeCaptureStateRequest(CaptureStateRequest{ProjectID: "p1", View: "sideways"}); !errors.Is(err, ErrInvalidCaptureStateRequest) {
			t.Fatalf("normalizeCaptureStateRequest(bad view) error = %v, want invalid request", err)
		}
		req, err := normalizeCaptureStateRequest(CaptureStateRequest{ProjectID: "p1"})
		if err != nil {
			t.Fatalf("normalizeCaptureStateRequest(project defaults) error = %v", err)
		}
		if req.ScopeType != ScopeTypeProject || req.ScopeID != "p1" || req.View != "summary" {
			t.Fatalf("normalizeCaptureStateRequest(project defaults) = %#v, want project scope defaults", req)
		}
	})

	t.Run("helper functions", func(t *testing.T) {
		supported := SupportedScopeTypes()
		supported[0] = "mutated"
		if SupportedScopeTypes()[0] != ScopeTypeProject {
			t.Fatalf("SupportedScopeTypes() exposed backing slice: %#v", SupportedScopeTypes())
		}
		if targetType, ok := commentTargetTypeFromScope(ScopeTypeTask); !ok || targetType != domain.CommentTargetTypeTask {
			t.Fatalf("commentTargetTypeFromScope(task) = %q, %t, want task true", targetType, ok)
		}
		if _, ok := commentTargetTypeFromScope("weird"); ok {
			t.Fatal("commentTargetTypeFromScope(weird) = true, want false")
		}
		if _, ok := findProjectByID([]domain.Project{project}, "missing"); ok {
			t.Fatal("findProjectByID(missing) = true, want false")
		}
		if canonicalLifecycleState("doing") != domain.StateProgress {
			t.Fatalf("canonicalLifecycleState(doing) = %q, want progress", canonicalLifecycleState("doing"))
		}
		if got := buildWarningsOverview(WorkOverview{TasksWithOpenBlockers: 1}, AttentionOverview{RequiresUserAction: 1}); len(got.Warnings) != 2 {
			t.Fatalf("buildWarningsOverview() = %#v, want two warnings", got)
		}
	})

	t.Run("attention unavailable", func(t *testing.T) {
		service := NewCaptureStateService(fakeCaptureReadModel{projects: projectSlice(project)}, nil, func() time.Time { return now })
		capture, err := service.CaptureState(context.Background(), CaptureStateRequest{ProjectID: project.ID})
		if err != nil {
			t.Fatalf("CaptureState(no attention) error = %v", err)
		}
		if capture.AttentionOverview.Available {
			t.Fatalf("AttentionOverview.Available = %t, want false", capture.AttentionOverview.Available)
		}
	})

	t.Run("attention error", func(t *testing.T) {
		service := NewCaptureStateService(
			fakeCaptureReadModel{projects: projectSlice(project)},
			fakeCaptureAttentionService{err: errors.New("boom")},
			func() time.Time { return now },
		)
		if _, err := service.CaptureState(context.Background(), CaptureStateRequest{ProjectID: project.ID}); err == nil || !strings.Contains(err.Error(), "list attention items") {
			t.Fatalf("CaptureState(attention error) = %v, want wrapped attention error", err)
		}
	})

	t.Run("comment error", func(t *testing.T) {
		service := NewCaptureStateService(
			fakeCaptureReadModel{
				projects:        projectSlice(project),
				listCommentsErr: errors.New("comment boom"),
			},
			nil,
			func() time.Time { return now },
		)
		if _, err := service.CaptureState(context.Background(), CaptureStateRequest{
			ProjectID: project.ID,
			ScopeType: ScopeTypeProject,
			ScopeID:   project.ID,
		}); err == nil || !strings.Contains(err.Error(), "list comments by target") {
			t.Fatalf("CaptureState(comment error) = %v, want wrapped comment error", err)
		}
	})
}

// projectSlice wraps one project into a fresh slice for helper construction.
func projectSlice(project domain.Project) []domain.Project {
	return []domain.Project{project}
}

// ptrTime returns a stable time pointer for test fixtures.
func ptrTime(ts time.Time) *time.Time {
	value := ts.UTC()
	return &value
}
