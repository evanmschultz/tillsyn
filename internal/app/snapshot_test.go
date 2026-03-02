package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hylla/tillsyn/internal/domain"
)

// TestExportSnapshotIncludesExpectedData verifies behavior for the covered scenario.
func TestExportSnapshotIncludesExpectedData(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 22, 10, 0, 0, 0, time.UTC)

	p1, _ := domain.NewProject("p1", "Alpha", "", now)
	p1.Metadata = domain.ProjectMetadata{Owner: "team-a", Tags: []string{"alpha"}}
	p2, _ := domain.NewProject("p2", "Beta", "", now)
	p2.Archive(now.Add(time.Minute))
	repo.projects[p1.ID] = p1
	repo.projects[p2.ID] = p2

	c1, _ := domain.NewColumn("c1", p1.ID, "To Do", 0, 0, now)
	c2, _ := domain.NewColumn("c2", p2.ID, "Done", 0, 0, now)
	repo.columns[c1.ID] = c1
	repo.columns[c2.ID] = c2

	t1, _ := domain.NewTask(domain.TaskInput{ID: "t1", ProjectID: p1.ID, ColumnID: c1.ID, Position: 0, Title: "Task A", Priority: domain.PriorityLow}, now)
	t2, _ := domain.NewTask(domain.TaskInput{ID: "t2", ProjectID: p2.ID, ColumnID: c2.ID, Position: 0, Title: "Task B", Priority: domain.PriorityHigh}, now)
	t2.Archive(now.Add(2 * time.Minute))
	repo.tasks[t1.ID] = t1
	repo.tasks[t2.ID] = t2

	kind, err := domain.NewKindDefinition(domain.KindDefinitionInput{
		ID:          "refactor",
		DisplayName: "Refactor",
		AppliesTo:   []domain.KindAppliesTo{domain.KindAppliesToTask},
	}, now)
	if err != nil {
		t.Fatalf("NewKindDefinition() error = %v", err)
	}
	repo.kindDefs[kind.ID] = kind
	repo.projectAllowedKinds[p1.ID] = []domain.KindID{kind.ID}

	projectComment, err := domain.NewComment(domain.CommentInput{
		ID:           "comment-1",
		ProjectID:    p1.ID,
		TargetType:   domain.CommentTargetTypeProject,
		TargetID:     p1.ID,
		BodyMarkdown: "Project comment",
		ActorID:      "tester",
		ActorName:    "tester",
		ActorType:    domain.ActorTypeUser,
	}, now)
	if err != nil {
		t.Fatalf("NewComment() error = %v", err)
	}
	commentKey := p1.ID + "|" + string(projectComment.TargetType) + "|" + projectComment.TargetID
	repo.comments[commentKey] = []domain.Comment{projectComment}

	lease, err := domain.NewCapabilityLease(domain.CapabilityLeaseInput{
		InstanceID: "lease-1",
		LeaseToken: "token-1",
		AgentName:  "orchestrator",
		ProjectID:  p1.ID,
		ScopeType:  domain.CapabilityScopeTask,
		ScopeID:    t1.ID,
		Role:       domain.CapabilityRoleOrchestrator,
		ExpiresAt:  now.Add(2 * time.Hour),
	}, now)
	if err != nil {
		t.Fatalf("NewCapabilityLease() error = %v", err)
	}
	repo.capabilityLeases[lease.InstanceID] = lease

	svc := NewService(repo, nil, func() time.Time { return now.Add(3 * time.Minute) }, ServiceConfig{})

	snapActive, err := svc.ExportSnapshot(context.Background(), false)
	if err != nil {
		t.Fatalf("ExportSnapshot(active) error = %v", err)
	}
	if snapActive.Version != SnapshotVersion {
		t.Fatalf("unexpected version %q", snapActive.Version)
	}
	if len(snapActive.Projects) != 1 || snapActive.Projects[0].ID != p1.ID {
		t.Fatalf("unexpected active projects %#v", snapActive.Projects)
	}
	if len(snapActive.Columns) != 1 || snapActive.Columns[0].ID != c1.ID {
		t.Fatalf("unexpected active columns %#v", snapActive.Columns)
	}
	if len(snapActive.Tasks) != 1 || snapActive.Tasks[0].ID != t1.ID {
		t.Fatalf("unexpected active tasks %#v", snapActive.Tasks)
	}

	snapAll, err := svc.ExportSnapshot(context.Background(), true)
	if err != nil {
		t.Fatalf("ExportSnapshot(all) error = %v", err)
	}
	if len(snapAll.Projects) != 2 || len(snapAll.Columns) != 2 || len(snapAll.Tasks) != 2 {
		t.Fatalf("unexpected all snapshot sizes p=%d c=%d t=%d", len(snapAll.Projects), len(snapAll.Columns), len(snapAll.Tasks))
	}
	if len(snapAll.KindDefinitions) != 1 {
		t.Fatalf("expected kind definition closure in snapshot, got %#v", snapAll.KindDefinitions)
	}
	if len(snapAll.ProjectAllowedKinds) != 1 || snapAll.ProjectAllowedKinds[0].ProjectID != p1.ID {
		t.Fatalf("expected project allowlist closure in snapshot, got %#v", snapAll.ProjectAllowedKinds)
	}
	if len(snapAll.Comments) != 1 || snapAll.Comments[0].ID != "comment-1" {
		t.Fatalf("expected comment closure in snapshot, got %#v", snapAll.Comments)
	}
	if len(snapAll.CapabilityLeases) != 1 || snapAll.CapabilityLeases[0].InstanceID != "lease-1" {
		t.Fatalf("expected capability lease closure in snapshot, got %#v", snapAll.CapabilityLeases)
	}
	foundMeta := false
	for _, sp := range snapAll.Projects {
		if sp.ID == p1.ID && sp.Metadata.Owner == "team-a" {
			foundMeta = true
			break
		}
	}
	if !foundMeta {
		t.Fatalf("expected metadata to round-trip in export, got %#v", snapAll.Projects)
	}
}

// TestImportSnapshotCreatesAndUpdates verifies behavior for the covered scenario.
func TestImportSnapshotCreatesAndUpdates(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 22, 10, 0, 0, 0, time.UTC)

	existingProject, _ := domain.NewProject("p1", "Old Name", "", now)
	existingCol, _ := domain.NewColumn("c1", existingProject.ID, "Old Col", 0, 0, now)
	existingTask, _ := domain.NewTask(domain.TaskInput{ID: "t1", ProjectID: existingProject.ID, ColumnID: existingCol.ID, Position: 0, Title: "Old Task", Priority: domain.PriorityLow}, now)

	repo.projects[existingProject.ID] = existingProject
	repo.columns[existingCol.ID] = existingCol
	repo.tasks[existingTask.ID] = existingTask

	svc := NewService(repo, nil, func() time.Time { return now }, ServiceConfig{})
	due := now.Add(48 * time.Hour)
	snap := Snapshot{
		Version: SnapshotVersion,
		Projects: []SnapshotProject{
			{ID: "p1", Name: "New Name", Description: "updated", Slug: "new-name", Metadata: domain.ProjectMetadata{Owner: "owner-1", Tags: []string{"a"}}, CreatedAt: now, UpdatedAt: now.Add(time.Minute)},
			{ID: "p2", Name: "Project Two", Description: "new", Slug: "project-two", CreatedAt: now, UpdatedAt: now.Add(time.Minute)},
		},
		Columns: []SnapshotColumn{
			{ID: "c1", ProjectID: "p1", Name: "Doing", Position: 1, WIPLimit: 2, CreatedAt: now, UpdatedAt: now.Add(time.Minute)},
			{ID: "c2", ProjectID: "p2", Name: "To Do", Position: 0, WIPLimit: 0, CreatedAt: now, UpdatedAt: now.Add(time.Minute)},
		},
		Tasks: []SnapshotTask{
			{ID: "t1", ProjectID: "p1", ColumnID: "c1", Position: 2, Title: "Updated Task", Description: "details", Priority: domain.PriorityHigh, DueAt: &due, Labels: []string{"a", "b"}, CreatedAt: now, UpdatedAt: now.Add(time.Minute)},
			{ID: "t2", ProjectID: "p2", ColumnID: "c2", Position: 0, Title: "New Task", Priority: domain.PriorityMedium, CreatedAt: now, UpdatedAt: now.Add(time.Minute)},
		},
		KindDefinitions: []SnapshotKindDefinition{
			{
				ID:          "refactor",
				DisplayName: "Refactor",
				AppliesTo:   []domain.KindAppliesTo{domain.KindAppliesToTask},
				CreatedAt:   now,
				UpdatedAt:   now.Add(time.Minute),
			},
		},
		ProjectAllowedKinds: []SnapshotProjectAllowedKinds{
			{ProjectID: "p1", KindIDs: []domain.KindID{"refactor"}},
		},
		Comments: []SnapshotComment{
			{
				ID:           "comment-1",
				ProjectID:    "p1",
				TargetType:   domain.CommentTargetTypeProject,
				TargetID:     "p1",
				BodyMarkdown: "Imported project comment",
				ActorID:      "importer",
				ActorName:    "importer",
				ActorType:    domain.ActorTypeUser,
				CreatedAt:    now,
				UpdatedAt:    now.Add(time.Minute),
			},
		},
		CapabilityLeases: []SnapshotCapabilityLease{
			{
				InstanceID:  "lease-1",
				LeaseToken:  "token-1",
				AgentName:   "orchestrator",
				ProjectID:   "p1",
				ScopeType:   domain.CapabilityScopeTask,
				ScopeID:     "t1",
				Role:        domain.CapabilityRoleOrchestrator,
				IssuedAt:    now,
				ExpiresAt:   now.Add(24 * time.Hour),
				HeartbeatAt: now.Add(2 * time.Minute),
			},
		},
	}

	if err := svc.ImportSnapshot(context.Background(), snap); err != nil {
		t.Fatalf("ImportSnapshot() error = %v", err)
	}

	if got := repo.projects["p1"]; got.Name != "New Name" || got.Description != "updated" {
		t.Fatalf("unexpected updated project %#v", got)
	}
	if got := repo.projects["p1"]; got.Metadata.Owner != "owner-1" {
		t.Fatalf("expected metadata owner updated, got %#v", got.Metadata)
	}
	if _, ok := repo.projects["p2"]; !ok {
		t.Fatal("expected new project p2")
	}
	if got := repo.columns["c1"]; got.Name != "Doing" || got.Position != 1 {
		t.Fatalf("unexpected updated column %#v", got)
	}
	if _, ok := repo.columns["c2"]; !ok {
		t.Fatal("expected new column c2")
	}
	if got := repo.tasks["t1"]; got.Title != "Updated Task" || got.Priority != domain.PriorityHigh {
		t.Fatalf("unexpected updated task %#v", got)
	}
	if _, ok := repo.tasks["t2"]; !ok {
		t.Fatal("expected new task t2")
	}
	if _, ok := repo.kindDefs[domain.KindID("refactor")]; !ok {
		t.Fatal("expected imported kind definition refactor")
	}
	allowed := repo.projectAllowedKinds["p1"]
	if len(allowed) != 1 || allowed[0] != domain.KindID("refactor") {
		t.Fatalf("expected imported project allowlist for p1, got %#v", allowed)
	}
	commentKey := "p1|project|p1"
	if len(repo.comments[commentKey]) != 1 || repo.comments[commentKey][0].ID != "comment-1" {
		t.Fatalf("expected imported project comment closure, got %#v", repo.comments[commentKey])
	}
	if _, ok := repo.capabilityLeases["lease-1"]; !ok {
		t.Fatal("expected imported capability lease lease-1")
	}
}

// TestImportSnapshotValidateErrors verifies behavior for the covered scenario.
func TestImportSnapshotValidateErrors(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, nil, time.Now, ServiceConfig{})

	badVersion := Snapshot{Version: "tillsyn.snapshot.v999"}
	if err := svc.ImportSnapshot(context.Background(), badVersion); err == nil {
		t.Fatal("expected version validation error")
	}
	missingVersion := Snapshot{}
	if err := svc.ImportSnapshot(context.Background(), missingVersion); err == nil {
		t.Fatal("expected missing version validation error")
	}

	now := time.Date(2026, 2, 22, 10, 0, 0, 0, time.UTC)
	badRefs := Snapshot{
		Version:  SnapshotVersion,
		Projects: []SnapshotProject{{ID: "p1", Name: "A", Slug: "a", CreatedAt: now, UpdatedAt: now}},
		Columns:  []SnapshotColumn{{ID: "c1", ProjectID: "missing", Name: "To Do", Position: 0, CreatedAt: now, UpdatedAt: now}},
	}
	if err := svc.ImportSnapshot(context.Background(), badRefs); err == nil {
		t.Fatal("expected reference validation error")
	}
}

// failingSnapshotRepo represents failing snapshot repo data used by this package.
type failingSnapshotRepo struct {
	*fakeRepo
	err error
}

// ListProjects lists projects.
func (f failingSnapshotRepo) ListProjects(context.Context, bool) ([]domain.Project, error) {
	return nil, f.err
}

// TestExportSnapshotPropagatesError verifies behavior for the covered scenario.
func TestExportSnapshotPropagatesError(t *testing.T) {
	expected := errors.New("boom")
	svc := NewService(failingSnapshotRepo{fakeRepo: newFakeRepo(), err: expected}, nil, time.Now, ServiceConfig{})
	_, err := svc.ExportSnapshot(context.Background(), false)
	if !errors.Is(err, expected) {
		t.Fatalf("expected error %v, got %v", expected, err)
	}
}
