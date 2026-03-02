package app

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/hylla/tillsyn/internal/domain"
)

// fakeRepo represents fake repo data used by this package.
type fakeRepo struct {
	projects            map[string]domain.Project
	columns             map[string]domain.Column
	tasks               map[string]domain.Task
	comments            map[string][]domain.Comment
	attentionItems      map[string]domain.AttentionItem
	changeEvents        map[string][]domain.ChangeEvent
	kindDefs            map[domain.KindID]domain.KindDefinition
	projectAllowedKinds map[string][]domain.KindID
	capabilityLeases    map[string]domain.CapabilityLease
}

// newFakeRepo constructs fake repo.
func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		projects:            map[string]domain.Project{},
		columns:             map[string]domain.Column{},
		tasks:               map[string]domain.Task{},
		comments:            map[string][]domain.Comment{},
		attentionItems:      map[string]domain.AttentionItem{},
		changeEvents:        map[string][]domain.ChangeEvent{},
		kindDefs:            map[domain.KindID]domain.KindDefinition{},
		projectAllowedKinds: map[string][]domain.KindID{},
		capabilityLeases:    map[string]domain.CapabilityLease{},
	}
}

// CreateProject creates project.
func (f *fakeRepo) CreateProject(_ context.Context, p domain.Project) error {
	f.projects[p.ID] = p
	return nil
}

// UpdateProject updates state for the requested operation.
func (f *fakeRepo) UpdateProject(_ context.Context, p domain.Project) error {
	f.projects[p.ID] = p
	return nil
}

// DeleteProject deletes one project.
func (f *fakeRepo) DeleteProject(_ context.Context, id string) error {
	if _, ok := f.projects[id]; !ok {
		return ErrNotFound
	}
	delete(f.projects, id)
	for columnID, column := range f.columns {
		if column.ProjectID != id {
			continue
		}
		delete(f.columns, columnID)
	}
	for taskID, task := range f.tasks {
		if task.ProjectID != id {
			continue
		}
		delete(f.tasks, taskID)
	}
	return nil
}

// GetProject returns project.
func (f *fakeRepo) GetProject(_ context.Context, id string) (domain.Project, error) {
	p, ok := f.projects[id]
	if !ok {
		return domain.Project{}, ErrNotFound
	}
	return p, nil
}

// ListProjects lists projects.
func (f *fakeRepo) ListProjects(_ context.Context, includeArchived bool) ([]domain.Project, error) {
	out := make([]domain.Project, 0, len(f.projects))
	for _, p := range f.projects {
		if !includeArchived && p.ArchivedAt != nil {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

// SetProjectAllowedKinds updates one project's kind allowlist.
func (f *fakeRepo) SetProjectAllowedKinds(_ context.Context, projectID string, kindIDs []domain.KindID) error {
	f.projectAllowedKinds[projectID] = append([]domain.KindID(nil), kindIDs...)
	return nil
}

// ListProjectAllowedKinds lists one project's kind allowlist.
func (f *fakeRepo) ListProjectAllowedKinds(_ context.Context, projectID string) ([]domain.KindID, error) {
	return append([]domain.KindID(nil), f.projectAllowedKinds[projectID]...), nil
}

// CreateKindDefinition creates one kind definition.
func (f *fakeRepo) CreateKindDefinition(_ context.Context, kind domain.KindDefinition) error {
	f.kindDefs[kind.ID] = kind
	return nil
}

// UpdateKindDefinition updates one kind definition.
func (f *fakeRepo) UpdateKindDefinition(_ context.Context, kind domain.KindDefinition) error {
	if _, ok := f.kindDefs[kind.ID]; !ok {
		return ErrNotFound
	}
	f.kindDefs[kind.ID] = kind
	return nil
}

// GetKindDefinition returns one kind definition by ID.
func (f *fakeRepo) GetKindDefinition(_ context.Context, kindID domain.KindID) (domain.KindDefinition, error) {
	kind, ok := f.kindDefs[kindID]
	if !ok {
		return domain.KindDefinition{}, ErrNotFound
	}
	return kind, nil
}

// ListKindDefinitions lists kind definitions.
func (f *fakeRepo) ListKindDefinitions(_ context.Context, includeArchived bool) ([]domain.KindDefinition, error) {
	out := make([]domain.KindDefinition, 0, len(f.kindDefs))
	for _, kind := range f.kindDefs {
		if !includeArchived && kind.ArchivedAt != nil {
			continue
		}
		out = append(out, kind)
	}
	return out, nil
}

// CreateColumn creates column.
func (f *fakeRepo) CreateColumn(_ context.Context, c domain.Column) error {
	f.columns[c.ID] = c
	return nil
}

// UpdateColumn updates state for the requested operation.
func (f *fakeRepo) UpdateColumn(_ context.Context, c domain.Column) error {
	f.columns[c.ID] = c
	return nil
}

// ListColumns lists columns.
func (f *fakeRepo) ListColumns(_ context.Context, projectID string, includeArchived bool) ([]domain.Column, error) {
	out := make([]domain.Column, 0, len(f.columns))
	for _, c := range f.columns {
		if c.ProjectID != projectID {
			continue
		}
		if !includeArchived && c.ArchivedAt != nil {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

// CreateTask creates task.
func (f *fakeRepo) CreateTask(_ context.Context, t domain.Task) error {
	f.tasks[t.ID] = t
	return nil
}

// UpdateTask updates state for the requested operation.
func (f *fakeRepo) UpdateTask(_ context.Context, t domain.Task) error {
	if _, ok := f.tasks[t.ID]; !ok {
		return ErrNotFound
	}
	f.tasks[t.ID] = t
	return nil
}

// GetTask returns task.
func (f *fakeRepo) GetTask(_ context.Context, id string) (domain.Task, error) {
	t, ok := f.tasks[id]
	if !ok {
		return domain.Task{}, ErrNotFound
	}
	return t, nil
}

// ListTasks lists tasks.
func (f *fakeRepo) ListTasks(_ context.Context, projectID string, includeArchived bool) ([]domain.Task, error) {
	out := make([]domain.Task, 0, len(f.tasks))
	for _, t := range f.tasks {
		if t.ProjectID != projectID {
			continue
		}
		if !includeArchived && t.ArchivedAt != nil {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}

// DeleteTask deletes task.
func (f *fakeRepo) DeleteTask(_ context.Context, id string) error {
	if _, ok := f.tasks[id]; !ok {
		return ErrNotFound
	}
	delete(f.tasks, id)
	return nil
}

// CreateComment creates comment.
func (f *fakeRepo) CreateComment(_ context.Context, comment domain.Comment) error {
	key := comment.ProjectID + "|" + string(comment.TargetType) + "|" + comment.TargetID
	f.comments[key] = append(f.comments[key], comment)
	return nil
}

// ListCommentsByTarget lists comments for a target.
func (f *fakeRepo) ListCommentsByTarget(_ context.Context, target domain.CommentTarget) ([]domain.Comment, error) {
	key := target.ProjectID + "|" + string(target.TargetType) + "|" + target.TargetID
	return append([]domain.Comment(nil), f.comments[key]...), nil
}

// CreateAttentionItem creates one attention item row.
func (f *fakeRepo) CreateAttentionItem(_ context.Context, item domain.AttentionItem) error {
	f.attentionItems[item.ID] = item
	return nil
}

// GetAttentionItem returns one attention item row by id.
func (f *fakeRepo) GetAttentionItem(_ context.Context, attentionID string) (domain.AttentionItem, error) {
	item, ok := f.attentionItems[attentionID]
	if !ok {
		return domain.AttentionItem{}, ErrNotFound
	}
	return item, nil
}

// ListAttentionItems lists scoped attention items in deterministic order.
func (f *fakeRepo) ListAttentionItems(_ context.Context, filter domain.AttentionListFilter) ([]domain.AttentionItem, error) {
	filter, err := domain.NormalizeAttentionListFilter(filter)
	if err != nil {
		return nil, err
	}

	matchesState := func(item domain.AttentionItem) bool {
		if len(filter.States) == 0 {
			return true
		}
		for _, state := range filter.States {
			if item.State == state {
				return true
			}
		}
		return false
	}
	matchesKind := func(item domain.AttentionItem) bool {
		if len(filter.Kinds) == 0 {
			return true
		}
		for _, kind := range filter.Kinds {
			if item.Kind == kind {
				return true
			}
		}
		return false
	}

	out := make([]domain.AttentionItem, 0)
	for _, item := range f.attentionItems {
		if item.ProjectID != filter.ProjectID {
			continue
		}
		if filter.ScopeType != "" && item.ScopeType != filter.ScopeType {
			continue
		}
		if filter.ScopeType != "" && item.ScopeID != filter.ScopeID {
			continue
		}
		if filter.UnresolvedOnly && !item.IsUnresolved() {
			continue
		}
		if filter.RequiresUserAction != nil && item.RequiresUserAction != *filter.RequiresUserAction {
			continue
		}
		if !matchesState(item) || !matchesKind(item) {
			continue
		}
		out = append(out, item)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID > out[j].ID
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	if filter.Limit > 0 && len(out) > filter.Limit {
		out = out[:filter.Limit]
	}
	return out, nil
}

// ResolveAttentionItem resolves one attention item row and returns the updated value.
func (f *fakeRepo) ResolveAttentionItem(_ context.Context, attentionID string, resolvedBy string, resolvedByType domain.ActorType, resolvedAt time.Time) (domain.AttentionItem, error) {
	item, ok := f.attentionItems[attentionID]
	if !ok {
		return domain.AttentionItem{}, ErrNotFound
	}
	if err := item.Resolve(resolvedBy, resolvedByType, resolvedAt); err != nil {
		return domain.AttentionItem{}, err
	}
	f.attentionItems[attentionID] = item
	return item, nil
}

// ListProjectChangeEvents lists change events.
func (f *fakeRepo) ListProjectChangeEvents(_ context.Context, projectID string, limit int) ([]domain.ChangeEvent, error) {
	events := append([]domain.ChangeEvent(nil), f.changeEvents[projectID]...)
	if limit <= 0 || limit >= len(events) {
		return events, nil
	}
	return events[:limit], nil
}

// CreateCapabilityLease creates one capability lease row.
func (f *fakeRepo) CreateCapabilityLease(_ context.Context, lease domain.CapabilityLease) error {
	f.capabilityLeases[lease.InstanceID] = lease
	return nil
}

// UpdateCapabilityLease updates one capability lease row.
func (f *fakeRepo) UpdateCapabilityLease(_ context.Context, lease domain.CapabilityLease) error {
	if _, ok := f.capabilityLeases[lease.InstanceID]; !ok {
		return ErrNotFound
	}
	f.capabilityLeases[lease.InstanceID] = lease
	return nil
}

// GetCapabilityLease returns one capability lease row.
func (f *fakeRepo) GetCapabilityLease(_ context.Context, instanceID string) (domain.CapabilityLease, error) {
	lease, ok := f.capabilityLeases[instanceID]
	if !ok {
		return domain.CapabilityLease{}, ErrNotFound
	}
	return lease, nil
}

// ListCapabilityLeasesByScope lists scope-matching capability leases.
func (f *fakeRepo) ListCapabilityLeasesByScope(_ context.Context, projectID string, scopeType domain.CapabilityScopeType, scopeID string) ([]domain.CapabilityLease, error) {
	out := make([]domain.CapabilityLease, 0)
	for _, lease := range f.capabilityLeases {
		if lease.ProjectID != projectID {
			continue
		}
		if lease.ScopeType != scopeType {
			continue
		}
		if strings.TrimSpace(scopeID) != "" && lease.ScopeID != strings.TrimSpace(scopeID) {
			continue
		}
		out = append(out, lease)
	}
	return out, nil
}

// RevokeCapabilityLeasesByScope revokes all scope-matching leases.
func (f *fakeRepo) RevokeCapabilityLeasesByScope(_ context.Context, projectID string, scopeType domain.CapabilityScopeType, scopeID string, revokedAt time.Time, reason string) error {
	for instanceID, lease := range f.capabilityLeases {
		if lease.ProjectID != projectID {
			continue
		}
		if lease.ScopeType != scopeType {
			continue
		}
		if strings.TrimSpace(scopeID) != "" && lease.ScopeID != strings.TrimSpace(scopeID) {
			continue
		}
		lease.Revoke(reason, revokedAt)
		f.capabilityLeases[instanceID] = lease
	}
	return nil
}

// TestEnsureDefaultProject verifies behavior for the covered scenario.
func TestEnsureDefaultProject(t *testing.T) {
	repo := newFakeRepo()
	idCounter := 0
	svc := NewService(repo, func() string {
		idCounter++
		return "id-" + string(rune('0'+idCounter))
	}, func() time.Time {
		return time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	}, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	project, err := svc.EnsureDefaultProject(context.Background())
	if err != nil {
		t.Fatalf("EnsureDefaultProject() error = %v", err)
	}
	if project.Name != "Inbox" {
		t.Fatalf("unexpected project name %q", project.Name)
	}
	columns, err := svc.ListColumns(context.Background(), project.ID, false)
	if err != nil {
		t.Fatalf("ListColumns() error = %v", err)
	}
	if len(columns) != 3 {
		t.Fatalf("expected 3 default columns, got %d", len(columns))
	}
}

// TestCreateTaskMoveSearchAndDeleteModes verifies behavior for the covered scenario.
func TestCreateTaskMoveSearchAndDeleteModes(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	ids := []string{"p1", "c1", "c2", "t1"}
	idx := 0
	svc := NewService(repo, func() string {
		id := ids[idx]
		idx++
		return id
	}, func() time.Time {
		return now
	}, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	project, err := svc.CreateProject(context.Background(), "Project", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	col1, err := svc.CreateColumn(context.Background(), project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	col2, err := svc.CreateColumn(context.Background(), project.ID, "Done", 1, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}

	task, err := svc.CreateTask(context.Background(), CreateTaskInput{
		ProjectID:   project.ID,
		ColumnID:    col1.ID,
		Title:       "Fix parser",
		Description: "Add tests for parser",
		Priority:    domain.PriorityHigh,
		Labels:      []string{"parser"},
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if task.Position != 0 {
		t.Fatalf("unexpected task position %d", task.Position)
	}

	task, err = svc.MoveTask(context.Background(), task.ID, col2.ID, 1)
	if err != nil {
		t.Fatalf("MoveTask() error = %v", err)
	}
	if task.ColumnID != col2.ID || task.Position != 1 {
		t.Fatalf("unexpected moved task %#v", task)
	}

	search, err := svc.SearchTaskMatches(context.Background(), SearchTasksFilter{
		ProjectID: project.ID,
		Query:     "parser",
	})
	if err != nil {
		t.Fatalf("SearchTaskMatches() error = %v", err)
	}
	if len(search) != 1 {
		t.Fatalf("expected 1 search result, got %d", len(search))
	}

	if err := svc.DeleteTask(context.Background(), task.ID, ""); err != nil {
		t.Fatalf("DeleteTask(archive default) error = %v", err)
	}
	tAfterArchive, err := repo.GetTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if tAfterArchive.ArchivedAt == nil {
		t.Fatal("expected task to be archived")
	}

	restored, err := svc.RestoreTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("RestoreTask() error = %v", err)
	}
	if restored.ArchivedAt != nil {
		t.Fatal("expected task to be restored")
	}

	if err := svc.DeleteTask(context.Background(), task.ID, DeleteModeHard); err != nil {
		t.Fatalf("DeleteTask(hard) error = %v", err)
	}
	if _, err := repo.GetTask(context.Background(), task.ID); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// TestRestoreTaskUsesRequestActorContext verifies restore guard actor type comes from request actor context.
func TestRestoreTaskUsesRequestActorContext(t *testing.T) {
	repo := newFakeRepo()
	ids := []string{"p1", "c1", "t1"}
	idx := 0
	now := time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	svc := NewService(repo, func() string {
		id := ids[idx]
		idx++
		return id
	}, func() time.Time {
		return now
	}, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	project, err := svc.CreateProject(context.Background(), "Restore Guard", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := svc.CreateColumn(context.Background(), project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	task, err := svc.CreateTask(context.Background(), CreateTaskInput{
		ProjectID:     project.ID,
		ColumnID:      column.ID,
		Title:         "archived task",
		Priority:      domain.PriorityMedium,
		UpdatedByType: domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if err := svc.DeleteTask(context.Background(), task.ID, DeleteModeArchive); err != nil {
		t.Fatalf("DeleteTask(archive) error = %v", err)
	}

	archivedTask, err := repo.GetTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("GetTask(archived) error = %v", err)
	}
	// Simulate prior archival attribution from an agent mutation.
	archivedTask.UpdatedByActor = "agent-1"
	archivedTask.UpdatedByType = domain.ActorTypeAgent
	repo.tasks[task.ID] = archivedTask

	ctx := WithMutationActor(context.Background(), MutationActor{
		ActorID:   "user-1",
		ActorType: domain.ActorTypeUser,
	})
	restored, err := svc.RestoreTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("RestoreTask() error = %v", err)
	}
	if restored.ArchivedAt != nil {
		t.Fatal("expected restore to clear archived_at")
	}
	if restored.UpdatedByActor != "user-1" {
		t.Fatalf("restored updated_by_actor = %q, want user-1", restored.UpdatedByActor)
	}
	if restored.UpdatedByType != domain.ActorTypeUser {
		t.Fatalf("restored updated_by_type = %q, want %q", restored.UpdatedByType, domain.ActorTypeUser)
	}
}

// TestRestoreTaskRequiresLeaseForNonUserCaller verifies non-user restore calls fail closed without a lease tuple.
func TestRestoreTaskRequiresLeaseForNonUserCaller(t *testing.T) {
	repo := newFakeRepo()
	ids := []string{"p1", "c1", "t1"}
	idx := 0
	now := time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	svc := NewService(repo, func() string {
		id := ids[idx]
		idx++
		return id
	}, func() time.Time {
		return now
	}, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	project, err := svc.CreateProject(context.Background(), "Restore Guard", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := svc.CreateColumn(context.Background(), project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	task, err := svc.CreateTask(context.Background(), CreateTaskInput{
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Title:     "archived task",
		Priority:  domain.PriorityMedium,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if err := svc.DeleteTask(context.Background(), task.ID, DeleteModeArchive); err != nil {
		t.Fatalf("DeleteTask(archive) error = %v", err)
	}

	ctx := WithMutationActor(context.Background(), MutationActor{
		ActorID:   "agent-1",
		ActorType: domain.ActorTypeAgent,
	})
	_, err = svc.RestoreTask(ctx, task.ID)
	if !errors.Is(err, domain.ErrMutationLeaseRequired) {
		t.Fatalf("RestoreTask() error = %v, want ErrMutationLeaseRequired", err)
	}
}

// TestDeleteTaskModeValidation verifies behavior for the covered scenario.
func TestDeleteTaskModeValidation(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, func() string { return "x" }, time.Now, ServiceConfig{})
	err := svc.DeleteTask(context.Background(), "task-1", DeleteMode("invalid"))
	if err != ErrInvalidDeleteMode {
		t.Fatalf("expected ErrInvalidDeleteMode, got %v", err)
	}
}

// TestRenameTask verifies behavior for the covered scenario.
func TestRenameTask(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: "p1",
		ColumnID:  "c1",
		Position:  0,
		Title:     "old",
		Priority:  domain.PriorityLow,
	}, now)
	repo.tasks[task.ID] = task

	svc := NewService(repo, nil, func() time.Time { return now.Add(time.Minute) }, ServiceConfig{})
	updated, err := svc.RenameTask(context.Background(), task.ID, "new title")
	if err != nil {
		t.Fatalf("RenameTask() error = %v", err)
	}
	if updated.Title != "new title" {
		t.Fatalf("unexpected title %q", updated.Title)
	}
}

// TestUpdateTask verifies behavior for the covered scenario.
func TestUpdateTask(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: "p1",
		ColumnID:  "c1",
		Position:  0,
		Title:     "old",
		Priority:  domain.PriorityLow,
	}, now)
	repo.tasks[task.ID] = task

	svc := NewService(repo, nil, func() time.Time { return now.Add(time.Minute) }, ServiceConfig{})
	due := now.Add(24 * time.Hour)
	updated, err := svc.UpdateTask(context.Background(), UpdateTaskInput{
		TaskID:      task.ID,
		Title:       "new title",
		Description: "details",
		Priority:    domain.PriorityHigh,
		DueAt:       &due,
		Labels:      []string{"frontend", "backend"},
	})
	if err != nil {
		t.Fatalf("UpdateTask() error = %v", err)
	}
	if updated.Title != "new title" || updated.Description != "details" || updated.Priority != domain.PriorityHigh {
		t.Fatalf("unexpected updated task %#v", updated)
	}
	if updated.DueAt == nil || len(updated.Labels) != 2 {
		t.Fatalf("expected due date and labels, got %#v", updated)
	}
}

// TestUpdateTaskAppliesMutationActorContext verifies context-supplied actor attribution is persisted on updates.
func TestUpdateTaskAppliesMutationActorContext(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:             "t1",
		ProjectID:      "p1",
		ColumnID:       "c1",
		Position:       0,
		Title:          "old",
		Priority:       domain.PriorityLow,
		CreatedByActor: "EVAN",
		UpdatedByActor: "EVAN",
		UpdatedByType:  domain.ActorTypeUser,
	}, now)
	repo.tasks[task.ID] = task

	svc := NewService(repo, nil, func() time.Time { return now.Add(time.Minute) }, ServiceConfig{})
	ctx := WithMutationActor(context.Background(), MutationActor{
		ActorID:   "orchestrator-1",
		ActorType: domain.ActorTypeAgent,
	})
	updated, err := svc.UpdateTask(ctx, UpdateTaskInput{
		TaskID: task.ID,
		Title:  "new title",
	})
	if err != nil {
		t.Fatalf("UpdateTask() error = %v", err)
	}
	if updated.UpdatedByActor != "orchestrator-1" {
		t.Fatalf("updated actor id = %q, want orchestrator-1", updated.UpdatedByActor)
	}
	if updated.UpdatedByType != domain.ActorTypeAgent {
		t.Fatalf("updated actor type = %q, want %q", updated.UpdatedByType, domain.ActorTypeAgent)
	}
}

// TestUpdateTaskPreservesPriorityWhenOmitted verifies update behavior when priority is omitted.
func TestUpdateTaskPreservesPriorityWhenOmitted(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: "p1",
		ColumnID:  "c1",
		Position:  0,
		Title:     "old",
		Priority:  domain.PriorityMedium,
	}, now)
	repo.tasks[task.ID] = task

	svc := NewService(repo, nil, func() time.Time { return now.Add(time.Minute) }, ServiceConfig{})
	updated, err := svc.UpdateTask(context.Background(), UpdateTaskInput{
		TaskID: task.ID,
		Title:  "new title",
	})
	if err != nil {
		t.Fatalf("UpdateTask(title-only) error = %v", err)
	}
	if updated.Priority != domain.PriorityMedium {
		t.Fatalf("priority = %q, want %q", updated.Priority, domain.PriorityMedium)
	}
}

// TestListAndSortHelpers verifies behavior for the covered scenario.
func TestListAndSortHelpers(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Project", "", now)
	repo.projects[p.ID] = p
	c1, _ := domain.NewColumn("c1", p.ID, "First", 5, 0, now)
	c2, _ := domain.NewColumn("c2", p.ID, "Second", 1, 0, now)
	repo.columns[c1.ID] = c1
	repo.columns[c2.ID] = c2

	t1, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  2,
		Title:     "later",
		Priority:  domain.PriorityLow,
	}, now)
	t2, _ := domain.NewTask(domain.TaskInput{
		ID:        "t2",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  1,
		Title:     "earlier",
		Priority:  domain.PriorityLow,
	}, now)
	t3, _ := domain.NewTask(domain.TaskInput{
		ID:        "t3",
		ProjectID: p.ID,
		ColumnID:  c2.ID,
		Position:  0,
		Title:     "other column",
		Priority:  domain.PriorityLow,
	}, now)
	repo.tasks[t1.ID] = t1
	repo.tasks[t2.ID] = t2
	repo.tasks[t3.ID] = t3

	svc := NewService(repo, nil, func() time.Time { return now }, ServiceConfig{})
	projects, err := svc.ListProjects(context.Background(), false)
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}

	columns, err := svc.ListColumns(context.Background(), p.ID, false)
	if err != nil {
		t.Fatalf("ListColumns() error = %v", err)
	}
	if columns[0].ID != c2.ID {
		t.Fatalf("expected column c2 first after sort, got %q", columns[0].ID)
	}

	tasks, err := svc.ListTasks(context.Background(), p.ID, false)
	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}
	if tasks[0].ID != t2.ID || tasks[1].ID != t1.ID || tasks[2].ID != t3.ID {
		t.Fatalf("unexpected task order: %#v", tasks)
	}

	allWithEmptyQuery, err := svc.SearchTaskMatches(context.Background(), SearchTasksFilter{
		ProjectID: p.ID,
		Query:     " ",
	})
	if err != nil {
		t.Fatalf("SearchTaskMatches(empty) error = %v", err)
	}
	if len(allWithEmptyQuery) != 3 {
		t.Fatalf("expected 3 results for empty query, got %d", len(allWithEmptyQuery))
	}
}

// TestSearchTaskMatchesAcrossProjectsAndStates verifies behavior for the covered scenario.
func TestSearchTaskMatchesAcrossProjectsAndStates(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p1, _ := domain.NewProject("p1", "Inbox", "", now)
	p2, _ := domain.NewProject("p2", "Client", "", now)
	repo.projects[p1.ID] = p1
	repo.projects[p2.ID] = p2

	c1, _ := domain.NewColumn("c1", p1.ID, "To Do", 0, 0, now)
	c2, _ := domain.NewColumn("c2", p1.ID, "In Progress", 1, 0, now)
	c3, _ := domain.NewColumn("c3", p2.ID, "In Progress", 0, 0, now)
	repo.columns[c1.ID] = c1
	repo.columns[c2.ID] = c2
	repo.columns[c3.ID] = c3

	t1, _ := domain.NewTask(domain.TaskInput{
		ID:          "t1",
		ProjectID:   p1.ID,
		ColumnID:    c1.ID,
		Position:    0,
		Title:       "Roadmap draft",
		Description: "planning",
		Priority:    domain.PriorityMedium,
	}, now)
	t2, _ := domain.NewTask(domain.TaskInput{
		ID:          "t2",
		ProjectID:   p1.ID,
		ColumnID:    c2.ID,
		Position:    0,
		Title:       "Implement parser",
		Description: "roadmap parser",
		Priority:    domain.PriorityHigh,
	}, now)
	t3, _ := domain.NewTask(domain.TaskInput{
		ID:          "t3",
		ProjectID:   p2.ID,
		ColumnID:    c3.ID,
		Position:    0,
		Title:       "Client sync",
		Description: "roadmap review",
		Priority:    domain.PriorityLow,
	}, now)
	t3.Archive(now.Add(time.Minute))
	repo.tasks[t1.ID] = t1
	repo.tasks[t2.ID] = t2
	repo.tasks[t3.ID] = t3

	svc := NewService(repo, nil, func() time.Time { return now }, ServiceConfig{})

	matches, err := svc.SearchTaskMatches(context.Background(), SearchTasksFilter{
		CrossProject:    true,
		IncludeArchived: false,
		States:          []string{"progress"},
		Query:           "parser",
	})
	if err != nil {
		t.Fatalf("SearchTaskMatches() error = %v", err)
	}
	if len(matches) != 1 || matches[0].Task.ID != "t2" || matches[0].StateID != "progress" {
		t.Fatalf("unexpected active matches %#v", matches)
	}

	matches, err = svc.SearchTaskMatches(context.Background(), SearchTasksFilter{
		CrossProject:    true,
		IncludeArchived: true,
		States:          []string{"archived"},
		Query:           "roadmap",
	})
	if err != nil {
		t.Fatalf("SearchTaskMatches(archived) error = %v", err)
	}
	if len(matches) != 1 || matches[0].Task.ID != "t3" || matches[0].StateID != "archived" {
		t.Fatalf("unexpected archived matches %#v", matches)
	}
}

// TestSearchTaskMatchesFuzzyQuery verifies behavior for the covered scenario.
func TestSearchTaskMatchesFuzzyQuery(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p1, _ := domain.NewProject("p1", "Inbox", "", now)
	repo.projects[p1.ID] = p1

	c1, _ := domain.NewColumn("c1", p1.ID, "To Do", 0, 0, now)
	repo.columns[c1.ID] = c1

	t1, _ := domain.NewTask(domain.TaskInput{
		ID:          "t1",
		ProjectID:   p1.ID,
		ColumnID:    c1.ID,
		Position:    0,
		Title:       "Implement parser",
		Description: "tokenization pipeline",
		Priority:    domain.PriorityMedium,
		Labels:      []string{"frontend", "parsing"},
	}, now)
	t2, _ := domain.NewTask(domain.TaskInput{
		ID:          "t2",
		ProjectID:   p1.ID,
		ColumnID:    c1.ID,
		Position:    1,
		Title:       "Write docs",
		Description: "onboarding guide",
		Priority:    domain.PriorityLow,
		Labels:      []string{"docs"},
	}, now)
	repo.tasks[t1.ID] = t1
	repo.tasks[t2.ID] = t2

	svc := NewService(repo, nil, func() time.Time { return now }, ServiceConfig{})

	tests := []struct {
		name    string
		query   string
		wantIDs []string
	}{
		{
			name:    "title subsequence",
			query:   "imppsr",
			wantIDs: []string{"t1"},
		},
		{
			name:    "description subsequence",
			query:   "tkpln",
			wantIDs: []string{"t1"},
		},
		{
			name:    "label subsequence",
			query:   "frnd",
			wantIDs: []string{"t1"},
		},
		{
			name:    "preserves rune order",
			query:   "psrmpi",
			wantIDs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches, err := svc.SearchTaskMatches(context.Background(), SearchTasksFilter{
				ProjectID: p1.ID,
				Query:     tt.query,
			})
			if err != nil {
				t.Fatalf("SearchTaskMatches() error = %v", err)
			}
			if len(matches) != len(tt.wantIDs) {
				t.Fatalf("expected %d results, got %d for query %q", len(tt.wantIDs), len(matches), tt.query)
			}
			for i := range tt.wantIDs {
				if matches[i].Task.ID != tt.wantIDs[i] {
					t.Fatalf("unexpected result order for query %q: got %q want %q", tt.query, matches[i].Task.ID, tt.wantIDs[i])
				}
			}
		})
	}
}

// TestEnsureDefaultProjectAlreadyExists verifies behavior for the covered scenario.
func TestEnsureDefaultProjectAlreadyExists(t *testing.T) {
	repo := newFakeRepo()
	now := time.Now()
	p, _ := domain.NewProject("p1", "Existing", "", now)
	repo.projects[p.ID] = p

	svc := NewService(repo, func() string { return "new-id" }, func() time.Time { return now }, ServiceConfig{})
	got, err := svc.EnsureDefaultProject(context.Background())
	if err != nil {
		t.Fatalf("EnsureDefaultProject() error = %v", err)
	}
	if got.ID != p.ID {
		t.Fatalf("expected existing project id %q, got %q", p.ID, got.ID)
	}
	if len(repo.columns) != 0 {
		t.Fatalf("expected no default columns to be inserted, got %d", len(repo.columns))
	}
}

// TestCreateProjectWithMetadataAndAutoColumns verifies behavior for the covered scenario.
func TestCreateProjectWithMetadataAndAutoColumns(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	ids := []string{"p1", "c1", "c2"}
	idx := 0
	svc := NewService(repo, func() string {
		id := ids[idx]
		idx++
		return id
	}, func() time.Time { return now }, ServiceConfig{
		AutoCreateProjectColumns: true,
		StateTemplates: []StateTemplate{
			{ID: "todo", Name: "To Do", Position: 0},
			{ID: "doing", Name: "Doing", Position: 1},
		},
	})

	project, err := svc.CreateProjectWithMetadata(context.Background(), CreateProjectInput{
		Name:        "Roadmap",
		Description: "Q2 plan",
		Metadata: domain.ProjectMetadata{
			Owner: "Evan",
			Tags:  []string{"Roadmap", "roadmap"},
		},
	})
	if err != nil {
		t.Fatalf("CreateProjectWithMetadata() error = %v", err)
	}
	if project.Metadata.Owner != "Evan" || len(project.Metadata.Tags) != 1 {
		t.Fatalf("unexpected project metadata %#v", project.Metadata)
	}
	columns, err := svc.ListColumns(context.Background(), project.ID, false)
	if err != nil {
		t.Fatalf("ListColumns() error = %v", err)
	}
	if len(columns) != 2 {
		t.Fatalf("expected 2 auto-created columns, got %d", len(columns))
	}
	if columns[0].Name != "To Do" || columns[1].Name != "Doing" {
		t.Fatalf("unexpected column names %#v", columns)
	}
}

// TestUpdateProject verifies behavior for the covered scenario.
func TestUpdateProject(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "old desc", now)
	repo.projects[project.ID] = project

	svc := NewService(repo, nil, func() time.Time { return now.Add(time.Minute) }, ServiceConfig{})
	updated, err := svc.UpdateProject(context.Background(), UpdateProjectInput{
		ProjectID:   project.ID,
		Name:        "Platform",
		Description: "new desc",
		Metadata: domain.ProjectMetadata{
			Owner: "team-tillsyn",
			Tags:  []string{"go", "Go"},
		},
	})
	if err != nil {
		t.Fatalf("UpdateProject() error = %v", err)
	}
	if updated.Name != "Platform" || updated.Description != "new desc" {
		t.Fatalf("unexpected updated project %#v", updated)
	}
	if updated.Metadata.Owner != "team-tillsyn" || len(updated.Metadata.Tags) != 1 || updated.Metadata.Tags[0] != "go" {
		t.Fatalf("unexpected metadata %#v", updated.Metadata)
	}
}

// TestArchiveRestoreAndDeleteProject verifies project archive, restore, and hard-delete behavior.
func TestArchiveRestoreAndDeleteProject(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 24, 8, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "desc", now)
	repo.projects[project.ID] = project

	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	repo.columns[column.ID] = column
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "task",
		Priority:  domain.PriorityMedium,
	}, now)
	repo.tasks[task.ID] = task

	svc := NewService(repo, nil, func() time.Time { return now.Add(time.Minute) }, ServiceConfig{})

	archived, err := svc.ArchiveProject(context.Background(), project.ID)
	if err != nil {
		t.Fatalf("ArchiveProject() error = %v", err)
	}
	if archived.ArchivedAt == nil {
		t.Fatal("expected project archived_at to be set")
	}

	active, err := svc.ListProjects(context.Background(), false)
	if err != nil {
		t.Fatalf("ListProjects(active) error = %v", err)
	}
	if len(active) != 0 {
		t.Fatalf("expected no active projects after archive, got %d", len(active))
	}

	restored, err := svc.RestoreProject(context.Background(), project.ID)
	if err != nil {
		t.Fatalf("RestoreProject() error = %v", err)
	}
	if restored.ArchivedAt != nil {
		t.Fatal("expected project archived_at cleared after restore")
	}

	if err := svc.DeleteProject(context.Background(), project.ID); err != nil {
		t.Fatalf("DeleteProject() error = %v", err)
	}
	if _, err := repo.GetProject(context.Background(), project.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected deleted project not found, got %v", err)
	}
	if _, ok := repo.columns[column.ID]; ok {
		t.Fatal("expected project columns deleted with project")
	}
	if _, ok := repo.tasks[task.ID]; ok {
		t.Fatal("expected project tasks deleted with project")
	}
}

// TestStateTemplateSanitization verifies behavior for the covered scenario.
func TestStateTemplateSanitization(t *testing.T) {
	got := sanitizeStateTemplates([]StateTemplate{
		{ID: "", Name: " To Do ", Position: 3},
		{ID: "todo", Name: "Duplicate", Position: 1},
		{ID: "", Name: "In Progress", Position: 2, WIPLimit: -1},
		{ID: "", Name: " ", Position: 4},
	})
	if len(got) != 2 {
		t.Fatalf("expected 2 sanitized states, got %#v", got)
	}
	if got[0].ID != "progress" || got[1].ID != "todo" {
		t.Fatalf("unexpected sanitized IDs %#v", got)
	}
	if got[0].WIPLimit != 0 {
		t.Fatalf("expected clamped wip limit, got %d", got[0].WIPLimit)
	}
}

// failingRepo represents failing repo data used by this package.
type failingRepo struct {
	*fakeRepo
	err error
}

// ListProjects lists projects.
func (f failingRepo) ListProjects(context.Context, bool) ([]domain.Project, error) {
	return nil, f.err
}

// TestEnsureDefaultProjectErrorPropagation verifies behavior for the covered scenario.
func TestEnsureDefaultProjectErrorPropagation(t *testing.T) {
	expected := errors.New("boom")
	svc := NewService(failingRepo{fakeRepo: newFakeRepo(), err: expected}, nil, time.Now, ServiceConfig{})
	_, err := svc.EnsureDefaultProject(context.Background())
	if !errors.Is(err, expected) {
		t.Fatalf("expected wrapped error %v, got %v", expected, err)
	}
}

// TestMoveTaskBlocksWhenStartCriteriaUnmet verifies behavior for the covered scenario.
func TestMoveTaskBlocksWhenStartCriteriaUnmet(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	repo.projects[project.ID] = project
	todo, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	progress, _ := domain.NewColumn("c2", project.ID, "In Progress", 1, 0, now)
	repo.columns[todo.ID] = todo
	repo.columns[progress.ID] = progress

	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  todo.ID,
		Position:  0,
		Title:     "blocked",
		Priority:  domain.PriorityMedium,
		Metadata: domain.TaskMetadata{
			CompletionContract: domain.CompletionContract{
				StartCriteria: []domain.ChecklistItem{{ID: "s1", Text: "design reviewed", Done: false}},
			},
		},
	}, now)
	repo.tasks[task.ID] = task

	svc := NewService(repo, nil, func() time.Time { return now }, ServiceConfig{})
	_, err := svc.MoveTask(context.Background(), task.ID, progress.ID, 0)
	if err == nil || !errors.Is(err, domain.ErrTransitionBlocked) {
		t.Fatalf("expected ErrTransitionBlocked, got %v", err)
	}
}

// TestMoveTaskAllowsDoneWhenContractsSatisfied verifies behavior for the covered scenario.
func TestMoveTaskAllowsDoneWhenContractsSatisfied(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	repo.projects[project.ID] = project
	progress, _ := domain.NewColumn("c2", project.ID, "In Progress", 1, 0, now)
	done, _ := domain.NewColumn("c3", project.ID, "Done", 2, 0, now)
	repo.columns[progress.ID] = progress
	repo.columns[done.ID] = done

	parent, _ := domain.NewTask(domain.TaskInput{
		ID:             "t-parent",
		ProjectID:      project.ID,
		ColumnID:       progress.ID,
		Position:       0,
		Title:          "parent",
		Priority:       domain.PriorityHigh,
		LifecycleState: domain.StateProgress,
		Metadata: domain.TaskMetadata{
			CompletionContract: domain.CompletionContract{
				CompletionCriteria: []domain.ChecklistItem{{ID: "c1", Text: "tests green", Done: true}},
				CompletionChecklist: []domain.ChecklistItem{
					{ID: "k1", Text: "docs updated", Done: true},
				},
				Policy: domain.CompletionPolicy{RequireChildrenDone: true},
			},
		},
	}, now)
	child, _ := domain.NewTask(domain.TaskInput{
		ID:             "t-child",
		ProjectID:      project.ID,
		ParentID:       parent.ID,
		ColumnID:       done.ID,
		Position:       0,
		Title:          "child",
		Priority:       domain.PriorityLow,
		LifecycleState: domain.StateDone,
	}, now)
	repo.tasks[parent.ID] = parent
	repo.tasks[child.ID] = child

	svc := NewService(repo, nil, func() time.Time { return now.Add(time.Minute) }, ServiceConfig{})
	moved, err := svc.MoveTask(context.Background(), parent.ID, done.ID, 0)
	if err != nil {
		t.Fatalf("MoveTask() error = %v", err)
	}
	if moved.LifecycleState != domain.StateDone {
		t.Fatalf("expected done lifecycle state, got %q", moved.LifecycleState)
	}
	if moved.CompletedAt == nil {
		t.Fatal("expected completed_at to be set")
	}
}

// TestMoveTaskBlocksDoneWhenAnySubtaskIncomplete verifies behavior for the covered scenario.
func TestMoveTaskBlocksDoneWhenAnySubtaskIncomplete(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	repo.projects[project.ID] = project
	progress, _ := domain.NewColumn("c2", project.ID, "In Progress", 1, 0, now)
	done, _ := domain.NewColumn("c3", project.ID, "Done", 2, 0, now)
	repo.columns[progress.ID] = progress
	repo.columns[done.ID] = done

	parent, _ := domain.NewTask(domain.TaskInput{
		ID:             "t-parent",
		ProjectID:      project.ID,
		ColumnID:       progress.ID,
		Position:       0,
		Title:          "parent",
		Priority:       domain.PriorityHigh,
		LifecycleState: domain.StateProgress,
	}, now)
	child, _ := domain.NewTask(domain.TaskInput{
		ID:             "t-child",
		ProjectID:      project.ID,
		ParentID:       parent.ID,
		ColumnID:       progress.ID,
		Position:       1,
		Title:          "child",
		Priority:       domain.PriorityLow,
		LifecycleState: domain.StateProgress,
	}, now)
	repo.tasks[parent.ID] = parent
	repo.tasks[child.ID] = child

	svc := NewService(repo, nil, func() time.Time { return now.Add(time.Minute) }, ServiceConfig{})
	_, err := svc.MoveTask(context.Background(), parent.ID, done.ID, 0)
	if err == nil || !errors.Is(err, domain.ErrTransitionBlocked) {
		t.Fatalf("expected ErrTransitionBlocked, got %v", err)
	}
	if !strings.Contains(err.Error(), "subtasks must be done") {
		t.Fatalf("expected incomplete subtask reason, got %v", err)
	}
}

// TestReparentTaskAndListChildTasks verifies behavior for the covered scenario.
func TestReparentTaskAndListChildTasks(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	repo.projects[project.ID] = project
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	repo.columns[column.ID] = column
	parent, _ := domain.NewTask(domain.TaskInput{
		ID:        "parent",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "parent",
		Priority:  domain.PriorityMedium,
	}, now)
	child, _ := domain.NewTask(domain.TaskInput{
		ID:        "child",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  1,
		Title:     "child",
		Priority:  domain.PriorityLow,
	}, now)
	repo.tasks[parent.ID] = parent
	repo.tasks[child.ID] = child

	svc := NewService(repo, nil, func() time.Time { return now.Add(2 * time.Minute) }, ServiceConfig{})
	updated, err := svc.ReparentTask(context.Background(), child.ID, parent.ID)
	if err != nil {
		t.Fatalf("ReparentTask() error = %v", err)
	}
	if updated.ParentID != parent.ID {
		t.Fatalf("expected parent id %q, got %q", parent.ID, updated.ParentID)
	}
	children, err := svc.ListChildTasks(context.Background(), project.ID, parent.ID, false)
	if err != nil {
		t.Fatalf("ListChildTasks() error = %v", err)
	}
	if len(children) != 1 || children[0].ID != child.ID {
		t.Fatalf("unexpected child list %#v", children)
	}
}

// TestGetProjectDependencyRollup verifies behavior for the covered scenario.
func TestGetProjectDependencyRollup(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	repo.projects[project.ID] = project
	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	repo.columns[column.ID] = column

	readyDep, _ := domain.NewTask(domain.TaskInput{
		ID:             "dep-ready",
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Position:       0,
		Title:          "ready dep",
		Priority:       domain.PriorityLow,
		LifecycleState: domain.StateDone,
	}, now)
	openDep, _ := domain.NewTask(domain.TaskInput{
		ID:             "dep-open",
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Position:       1,
		Title:          "open dep",
		Priority:       domain.PriorityLow,
		LifecycleState: domain.StateProgress,
	}, now)
	blocked, _ := domain.NewTask(domain.TaskInput{
		ID:        "blocked",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  2,
		Title:     "blocked",
		Priority:  domain.PriorityMedium,
		Metadata: domain.TaskMetadata{
			DependsOn:     []string{"dep-ready", "dep-open", "dep-missing"},
			BlockedBy:     []string{"dep-open"},
			BlockedReason: "waiting on review",
		},
	}, now)

	repo.tasks[readyDep.ID] = readyDep
	repo.tasks[openDep.ID] = openDep
	repo.tasks[blocked.ID] = blocked

	svc := NewService(repo, nil, func() time.Time { return now }, ServiceConfig{})
	rollup, err := svc.GetProjectDependencyRollup(context.Background(), project.ID)
	if err != nil {
		t.Fatalf("GetProjectDependencyRollup() error = %v", err)
	}
	if rollup.TotalItems != 3 {
		t.Fatalf("expected 3 total items, got %d", rollup.TotalItems)
	}
	if rollup.ItemsWithDependencies != 1 || rollup.DependencyEdges != 3 {
		t.Fatalf("unexpected dependency counts %#v", rollup)
	}
	if rollup.BlockedItems != 1 || rollup.BlockedByEdges != 1 {
		t.Fatalf("unexpected blocked counts %#v", rollup)
	}
	if rollup.UnresolvedDependencyEdges != 2 {
		t.Fatalf("expected 2 unresolved dependencies, got %d", rollup.UnresolvedDependencyEdges)
	}
}

// TestListProjectChangeEvents verifies behavior for the covered scenario.
func TestListProjectChangeEvents(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	repo.projects[project.ID] = project
	repo.changeEvents[project.ID] = []domain.ChangeEvent{
		{ID: 3, ProjectID: project.ID, WorkItemID: "t1", Operation: domain.ChangeOperationUpdate},
		{ID: 2, ProjectID: project.ID, WorkItemID: "t1", Operation: domain.ChangeOperationCreate},
	}

	svc := NewService(repo, nil, func() time.Time { return now }, ServiceConfig{})
	events, err := svc.ListProjectChangeEvents(context.Background(), project.ID, 1)
	if err != nil {
		t.Fatalf("ListProjectChangeEvents() error = %v", err)
	}
	if len(events) != 1 || events[0].Operation != domain.ChangeOperationUpdate {
		t.Fatalf("unexpected events %#v", events)
	}
}

// TestCreateAndListCommentsByTarget verifies behavior for the covered scenario.
func TestCreateAndListCommentsByTarget(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 23, 9, 0, 0, 0, time.UTC)
	ids := []string{"comment-2", "comment-1"}
	nextID := 0
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	repo.projects[project.ID] = project
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  "c1",
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityLow,
	}, now)
	repo.tasks[task.ID] = task

	svc := NewService(repo, func() string {
		id := ids[nextID]
		nextID++
		return id
	}, func() time.Time {
		// Fixed clock intentionally forces tie timestamps so ID ordering is tested.
		return now
	}, ServiceConfig{})

	if _, err := svc.CreateComment(context.Background(), CreateCommentInput{
		ProjectID:    project.ID,
		TargetType:   domain.CommentTargetTypeTask,
		TargetID:     task.ID,
		BodyMarkdown: "first",
		ActorType:    domain.ActorType("USER"),
		ActorID:      "user-1",
		ActorName:    "user-1",
	}); err != nil {
		t.Fatalf("CreateComment(first) error = %v", err)
	}
	second, err := svc.CreateComment(context.Background(), CreateCommentInput{
		ProjectID:    project.ID,
		TargetType:   domain.CommentTargetTypeTask,
		TargetID:     task.ID,
		BodyMarkdown: "second",
	})
	if err != nil {
		t.Fatalf("CreateComment(second) error = %v", err)
	}
	if second.ActorType != domain.ActorTypeUser {
		t.Fatalf("expected default actor type user, got %q", second.ActorType)
	}
	if second.ActorID != "tillsyn-user" {
		t.Fatalf("expected default actor id tillsyn-user, got %q", second.ActorID)
	}
	if second.ActorName != "tillsyn-user" {
		t.Fatalf("expected default actor name tillsyn-user, got %q", second.ActorName)
	}

	comments, err := svc.ListCommentsByTarget(context.Background(), ListCommentsByTargetInput{
		ProjectID:  project.ID,
		TargetType: domain.CommentTargetTypeTask,
		TargetID:   task.ID,
	})
	if err != nil {
		t.Fatalf("ListCommentsByTarget() error = %v", err)
	}
	if len(comments) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(comments))
	}
	if comments[0].ID != "comment-1" || comments[1].ID != "comment-2" {
		t.Fatalf("expected deterministic id ordering on equal timestamps, got %#v", comments)
	}
}

// TestCreateCommentValidation verifies behavior for the covered scenario.
func TestCreateCommentValidation(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 23, 9, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Inbox", "", now)
	repo.projects[project.ID] = project
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: project.ID,
		ColumnID:  "c1",
		Position:  0,
		Title:     "Task",
		Priority:  domain.PriorityLow,
	}, now)
	repo.tasks[task.ID] = task
	svc := NewService(repo, func() string { return "comment-1" }, time.Now, ServiceConfig{})

	_, err := svc.CreateComment(context.Background(), CreateCommentInput{
		ProjectID:    "",
		TargetType:   domain.CommentTargetTypeProject,
		TargetID:     "p1",
		BodyMarkdown: "body",
	})
	if err != domain.ErrInvalidID {
		t.Fatalf("expected ErrInvalidID for missing project id, got %v", err)
	}

	_, err = svc.CreateComment(context.Background(), CreateCommentInput{
		ProjectID:    project.ID,
		TargetType:   domain.CommentTargetTypeTask,
		TargetID:     task.ID,
		BodyMarkdown: " ",
	})
	if err != domain.ErrInvalidBodyMarkdown {
		t.Fatalf("expected ErrInvalidBodyMarkdown, got %v", err)
	}
	_, err = svc.CreateComment(context.Background(), CreateCommentInput{
		ProjectID:    project.ID,
		TargetType:   domain.CommentTargetTypeTask,
		TargetID:     "missing-task",
		BodyMarkdown: "body",
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for unknown target, got %v", err)
	}

	_, err = svc.ListCommentsByTarget(context.Background(), ListCommentsByTargetInput{
		ProjectID:  project.ID,
		TargetType: domain.CommentTargetType("invalid"),
		TargetID:   task.ID,
	})
	if err != domain.ErrInvalidTargetType {
		t.Fatalf("expected ErrInvalidTargetType, got %v", err)
	}
}

// TestSnapshotCommentTargetTypeForTaskSupportsHierarchyNodes verifies branch/subphase comment target mapping.
func TestSnapshotCommentTargetTypeForTaskSupportsHierarchyNodes(t *testing.T) {
	tests := []struct {
		name string
		task domain.Task
		want domain.CommentTargetType
	}{
		{
			name: "branch kind",
			task: domain.Task{Kind: domain.WorkKind(domain.KindAppliesToBranch)},
			want: domain.CommentTargetTypeBranch,
		},
		{
			name: "branch scope fallback",
			task: domain.Task{Kind: domain.WorkKindTask, Scope: domain.KindAppliesToBranch},
			want: domain.CommentTargetTypeBranch,
		},
		{
			name: "subphase scope on phase kind",
			task: domain.Task{Kind: domain.WorkKindPhase, Scope: domain.KindAppliesToSubphase},
			want: domain.CommentTargetTypeSubphase,
		},
		{
			name: "subphase kind",
			task: domain.Task{Kind: domain.WorkKind(domain.KindAppliesToSubphase)},
			want: domain.CommentTargetTypeSubphase,
		},
		{
			name: "phase backward compatible",
			task: domain.Task{Kind: domain.WorkKindPhase, Scope: domain.KindAppliesToPhase},
			want: domain.CommentTargetTypePhase,
		},
		{
			name: "task backward compatible",
			task: domain.Task{Kind: domain.WorkKindTask, Scope: domain.KindAppliesToTask},
			want: domain.CommentTargetTypeTask,
		},
	}

	for _, tc := range tests {
		got := snapshotCommentTargetTypeForTask(tc.task)
		if got != tc.want {
			t.Fatalf("%s: snapshotCommentTargetTypeForTask() = %q, want %q", tc.name, got, tc.want)
		}
	}
}

// TestIssueCapabilityLeaseOverlapPolicy verifies orchestrator overlap behavior and override token handling.
func TestIssueCapabilityLeaseOverlapPolicy(t *testing.T) {
	repo := newFakeRepo()
	ids := []string{"p1", "lease-a", "lease-token-a", "lease-b", "lease-token-b", "lease-c", "lease-token-c", "lease-d", "lease-token-d"}
	idx := 0
	svc := NewService(repo, func() string {
		id := ids[idx]
		idx++
		return id
	}, func() time.Time {
		return time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	}, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	project, err := svc.CreateProjectWithMetadata(context.Background(), CreateProjectInput{
		Name:        "Lease Policy",
		Description: "",
		Metadata: domain.ProjectMetadata{
			CapabilityPolicy: domain.ProjectCapabilityPolicy{
				AllowOrchestratorOverride: true,
				OrchestratorOverrideToken: "override-123",
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateProjectWithMetadata() error = %v", err)
	}

	if _, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		Role:            domain.CapabilityRoleOrchestrator,
		AgentName:       "orch-a",
		AgentInstanceID: "orch-a",
	}); err != nil {
		t.Fatalf("IssueCapabilityLease(first) error = %v", err)
	}

	if _, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		Role:            domain.CapabilityRoleOrchestrator,
		AgentName:       "orch-b",
		AgentInstanceID: "orch-b",
	}); err != domain.ErrOverrideTokenRequired {
		t.Fatalf("expected ErrOverrideTokenRequired, got %v", err)
	}

	if _, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		Role:            domain.CapabilityRoleOrchestrator,
		AgentName:       "orch-c",
		AgentInstanceID: "orch-c",
		OverrideToken:   "wrong",
	}); err != domain.ErrOverrideTokenInvalid {
		t.Fatalf("expected ErrOverrideTokenInvalid, got %v", err)
	}

	if _, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		Role:            domain.CapabilityRoleOrchestrator,
		AgentName:       "orch-d",
		AgentInstanceID: "orch-d",
		OverrideToken:   "override-123",
	}); err != nil {
		t.Fatalf("IssueCapabilityLease(override) error = %v", err)
	}
}

// TestCreateTaskMutationGuardRequiredForAgent verifies strict guard enforcement for non-user actor writes.
func TestCreateTaskMutationGuardRequiredForAgent(t *testing.T) {
	repo := newFakeRepo()
	ids := []string{"p1", "c1", "t1", "lease-1", "lease-token-1", "t2"}
	idx := 0
	svc := NewService(repo, func() string {
		id := ids[idx]
		idx++
		return id
	}, func() time.Time {
		return time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	}, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	project, err := svc.CreateProject(context.Background(), "Guard Project", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := svc.CreateColumn(context.Background(), project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}

	_, err = svc.CreateTask(context.Background(), CreateTaskInput{
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Title:          "agent task",
		Priority:       domain.PriorityMedium,
		CreatedByActor: "agent-1",
		UpdatedByActor: "agent-1",
		UpdatedByType:  domain.ActorTypeAgent,
	})
	if err != domain.ErrMutationLeaseRequired {
		t.Fatalf("expected ErrMutationLeaseRequired, got %v", err)
	}

	lease, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		Role:            domain.CapabilityRoleWorker,
		AgentName:       "agent-1",
		AgentInstanceID: "agent-1",
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease() error = %v", err)
	}

	guardedCtx := WithMutationGuard(context.Background(), MutationGuard{
		AgentName:       "agent-1",
		AgentInstanceID: lease.InstanceID,
		LeaseToken:      lease.LeaseToken,
	})
	created, err := svc.CreateTask(guardedCtx, CreateTaskInput{
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Title:          "guarded agent task",
		Priority:       domain.PriorityMedium,
		CreatedByActor: "agent-1",
		UpdatedByActor: "agent-1",
		UpdatedByType:  domain.ActorTypeAgent,
	})
	if err != nil {
		t.Fatalf("CreateTask(guarded) error = %v", err)
	}
	if strings.TrimSpace(created.ID) == "" {
		t.Fatal("expected created task id to be populated")
	}
	if created.UpdatedByType != domain.ActorTypeAgent {
		t.Fatalf("expected agent attribution on guarded task, got %q", created.UpdatedByType)
	}
}

// TestScopedLeaseAllowsLineageMutations verifies branch/phase/task scoped lease behavior in-subtree.
func TestScopedLeaseAllowsLineageMutations(t *testing.T) {
	repo := newFakeRepo()
	ids := []string{
		"p1", "c1",
		"branch-1", "phase-1", "task-1",
		"lease-branch", "lease-token-branch",
		"lease-phase", "lease-token-phase",
		"task-2",
		"lease-task", "lease-token-task",
		"comment-1",
	}
	idx := 0
	svc := NewService(repo, func() string {
		id := ids[idx]
		idx++
		return id
	}, func() time.Time {
		return time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	}, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	project, err := svc.CreateProject(context.Background(), "Scoped", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := svc.CreateColumn(context.Background(), project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	branch, err := svc.CreateTask(context.Background(), CreateTaskInput{
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Kind:      domain.WorkKind("branch"),
		Scope:     domain.KindAppliesToBranch,
		Title:     "Branch A",
		Priority:  domain.PriorityMedium,
	})
	if err != nil {
		t.Fatalf("CreateTask(branch) error = %v", err)
	}
	phase, err := svc.CreateTask(context.Background(), CreateTaskInput{
		ProjectID: project.ID,
		ParentID:  branch.ID,
		ColumnID:  column.ID,
		Kind:      domain.WorkKindPhase,
		Scope:     domain.KindAppliesToPhase,
		Title:     "Phase A",
		Priority:  domain.PriorityMedium,
	})
	if err != nil {
		t.Fatalf("CreateTask(phase) error = %v", err)
	}
	task, err := svc.CreateTask(context.Background(), CreateTaskInput{
		ProjectID: project.ID,
		ParentID:  phase.ID,
		ColumnID:  column.ID,
		Kind:      domain.WorkKindTask,
		Scope:     domain.KindAppliesToTask,
		Title:     "Task A1",
		Priority:  domain.PriorityMedium,
	})
	if err != nil {
		t.Fatalf("CreateTask(task) error = %v", err)
	}

	branchLease, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeBranch,
		ScopeID:         branch.ID,
		Role:            domain.CapabilityRoleWorker,
		AgentName:       "branch-agent",
		AgentInstanceID: "branch-agent",
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease(branch) error = %v", err)
	}
	branchCtx := WithMutationGuard(context.Background(), MutationGuard{
		AgentName:       branchLease.AgentName,
		AgentInstanceID: branchLease.InstanceID,
		LeaseToken:      branchLease.LeaseToken,
	})
	if _, err := svc.UpdateTask(branchCtx, UpdateTaskInput{
		TaskID:      task.ID,
		Title:       "Task A1",
		Description: "branch-updated",
		Priority:    domain.PriorityMedium,
		UpdatedBy:   "branch-agent",
		UpdatedType: domain.ActorTypeAgent,
	}); err != nil {
		t.Fatalf("UpdateTask(branch scoped) error = %v", err)
	}

	phaseLease, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopePhase,
		ScopeID:         phase.ID,
		Role:            domain.CapabilityRoleWorker,
		AgentName:       "phase-agent",
		AgentInstanceID: "phase-agent",
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease(phase) error = %v", err)
	}
	phaseCtx := WithMutationGuard(context.Background(), MutationGuard{
		AgentName:       phaseLease.AgentName,
		AgentInstanceID: phaseLease.InstanceID,
		LeaseToken:      phaseLease.LeaseToken,
	})
	if _, err := svc.CreateTask(phaseCtx, CreateTaskInput{
		ProjectID:      project.ID,
		ParentID:       phase.ID,
		ColumnID:       column.ID,
		Kind:           domain.WorkKindTask,
		Scope:          domain.KindAppliesToTask,
		Title:          "Task A2",
		Priority:       domain.PriorityMedium,
		CreatedByActor: "phase-agent",
		UpdatedByActor: "phase-agent",
		UpdatedByType:  domain.ActorTypeAgent,
	}); err != nil {
		t.Fatalf("CreateTask(phase scoped) error = %v", err)
	}

	taskLease, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeTask,
		ScopeID:         task.ID,
		Role:            domain.CapabilityRoleWorker,
		AgentName:       "task-agent",
		AgentInstanceID: "task-agent",
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease(task) error = %v", err)
	}
	taskCtx := WithMutationGuard(context.Background(), MutationGuard{
		AgentName:       taskLease.AgentName,
		AgentInstanceID: taskLease.InstanceID,
		LeaseToken:      taskLease.LeaseToken,
	})
	if _, err := svc.CreateComment(taskCtx, CreateCommentInput{
		ProjectID:    project.ID,
		TargetType:   domain.CommentTargetTypeTask,
		TargetID:     task.ID,
		BodyMarkdown: "task scoped comment",
		ActorType:    domain.ActorTypeAgent,
		ActorID:      "task-agent",
		ActorName:    "task-agent",
	}); err != nil {
		t.Fatalf("CreateComment(task scoped) error = %v", err)
	}
}

// TestScopedLeaseRejectsSiblingMutations verifies out-of-scope sibling writes fail closed.
func TestScopedLeaseRejectsSiblingMutations(t *testing.T) {
	repo := newFakeRepo()
	ids := []string{
		"p1", "c1",
		"branch-1", "phase-a", "phase-b",
		"task-a", "task-b",
		"lease-phase-a", "lease-token-phase-a",
	}
	idx := 0
	svc := NewService(repo, func() string {
		id := ids[idx]
		idx++
		return id
	}, func() time.Time {
		return time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	}, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	project, err := svc.CreateProject(context.Background(), "Scoped Deny", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := svc.CreateColumn(context.Background(), project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	branch, err := svc.CreateTask(context.Background(), CreateTaskInput{
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Kind:      domain.WorkKind("branch"),
		Scope:     domain.KindAppliesToBranch,
		Title:     "Branch",
		Priority:  domain.PriorityMedium,
	})
	if err != nil {
		t.Fatalf("CreateTask(branch) error = %v", err)
	}
	phaseA, err := svc.CreateTask(context.Background(), CreateTaskInput{
		ProjectID: project.ID,
		ParentID:  branch.ID,
		ColumnID:  column.ID,
		Kind:      domain.WorkKindPhase,
		Scope:     domain.KindAppliesToPhase,
		Title:     "Phase A",
		Priority:  domain.PriorityMedium,
	})
	if err != nil {
		t.Fatalf("CreateTask(phaseA) error = %v", err)
	}
	phaseB, err := svc.CreateTask(context.Background(), CreateTaskInput{
		ProjectID: project.ID,
		ParentID:  branch.ID,
		ColumnID:  column.ID,
		Kind:      domain.WorkKindPhase,
		Scope:     domain.KindAppliesToPhase,
		Title:     "Phase B",
		Priority:  domain.PriorityMedium,
	})
	if err != nil {
		t.Fatalf("CreateTask(phaseB) error = %v", err)
	}
	taskB, err := svc.CreateTask(context.Background(), CreateTaskInput{
		ProjectID: project.ID,
		ParentID:  phaseB.ID,
		ColumnID:  column.ID,
		Kind:      domain.WorkKindTask,
		Scope:     domain.KindAppliesToTask,
		Title:     "Task B1",
		Priority:  domain.PriorityMedium,
	})
	if err != nil {
		t.Fatalf("CreateTask(taskB) error = %v", err)
	}

	phaseALease, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopePhase,
		ScopeID:         phaseA.ID,
		Role:            domain.CapabilityRoleWorker,
		AgentName:       "phase-a-agent",
		AgentInstanceID: "phase-a-agent",
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease(phaseA) error = %v", err)
	}
	phaseACtx := WithMutationGuard(context.Background(), MutationGuard{
		AgentName:       phaseALease.AgentName,
		AgentInstanceID: phaseALease.InstanceID,
		LeaseToken:      phaseALease.LeaseToken,
	})

	if _, err := svc.UpdateTask(phaseACtx, UpdateTaskInput{
		TaskID:      taskB.ID,
		Title:       "Task B1",
		Description: "out of scope",
		Priority:    domain.PriorityMedium,
		UpdatedBy:   "phase-a-agent",
		UpdatedType: domain.ActorTypeAgent,
	}); !errors.Is(err, domain.ErrMutationLeaseInvalid) {
		t.Fatalf("UpdateTask(out of scope) error = %v, want ErrMutationLeaseInvalid", err)
	}

	if _, err := svc.CreateTask(phaseACtx, CreateTaskInput{
		ProjectID:      project.ID,
		ParentID:       phaseB.ID,
		ColumnID:       column.ID,
		Kind:           domain.WorkKindTask,
		Scope:          domain.KindAppliesToTask,
		Title:          "Task B2",
		Priority:       domain.PriorityMedium,
		CreatedByActor: "phase-a-agent",
		UpdatedByActor: "phase-a-agent",
		UpdatedByType:  domain.ActorTypeAgent,
	}); !errors.Is(err, domain.ErrMutationLeaseInvalid) {
		t.Fatalf("CreateTask(out of scope) error = %v, want ErrMutationLeaseInvalid", err)
	}
}

// TestCreateTaskKindPayloadValidation verifies schema-based runtime validation for dynamic kinds.
func TestCreateTaskKindPayloadValidation(t *testing.T) {
	repo := newFakeRepo()
	ids := []string{"p1", "c1", "t1", "t2"}
	idx := 0
	svc := NewService(repo, func() string {
		id := ids[idx]
		idx++
		return id
	}, func() time.Time {
		return time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	}, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	project, err := svc.CreateProject(context.Background(), "Kinds", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := svc.CreateColumn(context.Background(), project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}

	_, err = svc.UpsertKindDefinition(context.Background(), CreateKindDefinitionInput{
		ID:                "refactor",
		DisplayName:       "Refactor",
		AppliesTo:         []domain.KindAppliesTo{domain.KindAppliesToTask},
		PayloadSchemaJSON: `{"type":"object","required":["package"],"properties":{"package":{"type":"string"}},"additionalProperties":false}`,
	})
	if err != nil {
		t.Fatalf("UpsertKindDefinition() error = %v", err)
	}
	if err := svc.SetProjectAllowedKinds(context.Background(), SetProjectAllowedKindsInput{
		ProjectID: project.ID,
		KindIDs:   []domain.KindID{"refactor", domain.DefaultProjectKind, domain.KindID(domain.WorkKindTask), domain.KindID(domain.WorkKindSubtask)},
	}); err != nil {
		t.Fatalf("SetProjectAllowedKinds() error = %v", err)
	}

	_, err = svc.CreateTask(context.Background(), CreateTaskInput{
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Kind:      domain.WorkKind("refactor"),
		Title:     "invalid payload",
		Priority:  domain.PriorityMedium,
		Metadata:  domain.TaskMetadata{KindPayload: json.RawMessage(`{"missing":"value"}`)},
	})
	if !errors.Is(err, domain.ErrInvalidKindPayload) {
		t.Fatalf("expected ErrInvalidKindPayload, got %v", err)
	}

	created, err := svc.CreateTask(context.Background(), CreateTaskInput{
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Kind:      domain.WorkKind("refactor"),
		Title:     "valid payload",
		Priority:  domain.PriorityMedium,
		Metadata:  domain.TaskMetadata{KindPayload: json.RawMessage(`{"package":"internal/app"}`)},
	})
	if err != nil {
		t.Fatalf("CreateTask(valid payload) error = %v", err)
	}
	if created.Kind != domain.WorkKind("refactor") {
		t.Fatalf("expected refactor kind, got %q", created.Kind)
	}
}

// TestReparentTaskRejectsCycle verifies cycle prevention during reparenting.
func TestReparentTaskRejectsCycle(t *testing.T) {
	repo := newFakeRepo()
	ids := []string{"p1", "c1", "t-parent", "t-child"}
	idx := 0
	svc := NewService(repo, func() string {
		id := ids[idx]
		idx++
		return id
	}, func() time.Time {
		return time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	}, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	project, err := svc.CreateProject(context.Background(), "Hierarchy", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := svc.CreateColumn(context.Background(), project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	parent, err := svc.CreateTask(context.Background(), CreateTaskInput{
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Title:     "parent",
		Priority:  domain.PriorityMedium,
	})
	if err != nil {
		t.Fatalf("CreateTask(parent) error = %v", err)
	}
	child, err := svc.CreateTask(context.Background(), CreateTaskInput{
		ProjectID: project.ID,
		ParentID:  parent.ID,
		Kind:      domain.WorkKindSubtask,
		ColumnID:  column.ID,
		Title:     "child",
		Priority:  domain.PriorityMedium,
	})
	if err != nil {
		t.Fatalf("CreateTask(child) error = %v", err)
	}

	if _, err := svc.ReparentTask(context.Background(), parent.ID, child.ID); err != domain.ErrInvalidParentID {
		t.Fatalf("expected ErrInvalidParentID, got %v", err)
	}
}
